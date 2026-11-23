package calc

import (
	"math"
	"math/big"
)

func init() {
	Nodes["004e65d0"] = Node{Addr: "004e65d0",
		In:  []string{"baseSizeBits", "sizeMinBits", "sizeMaxBits", "monsterMaximumHp", "templateHpBase", "templateHpDiceCount", "templateHpDiceSides"},
		Out: []string{"sizeBits"}, F: monsterHpScaleNode}
}

// monsterHpScaleNode returns the IEEE754 double bits of the x87 result (exe 0x4e65d0).
// Inputs are the f32 bit patterns of the model fields (+0x5f0 base, +0x5f4 min, +0x5f8 max),
// the monster's u32 maximum HP (+0x27) and the cached template's HP dice (s16 +0xd6,
// u8 +0xd8, u8 +0xd9). The template cache is assumed filled (exe 0x4e665b..0x4e669d).
// Scalar f32 constants, not tables: 0x5ebb34 = 0, 0x5ed2a8 = f32(0.0004), 0x5ec040 = f32(0.2).
// Each x87 operation keeps the 64-bit significand.
func monsterHpScaleNode(in []int64, r Rand) []int64 {
	if len(in) != 7 {
		panic("monster hp scale: expected 7 inputs")
	}
	for i, v := range in {
		lo, hi := int64(0), int64(0xffffffff)
		switch i {
		case 4:
			lo, hi = -32768, 32767
		case 5, 6:
			hi = 255
		}
		if v < lo || v > hi {
			panic("monster hp scale: field width overflow")
		}
	}
	f32 := func(v int64) float32 { return math.Float32frombits(uint32(v)) }
	base, lo, hi := f32(in[0]), f32(in[1]), f32(in[2])
	if math.IsNaN(float64(base)) || math.IsInf(float64(base), 0) || math.IsInf(float64(lo), 0) || math.IsInf(float64(hi), 0) {
		panic("monster hp scale: infinite size field or non-finite base")
	}
	ret := func(v float64) []int64 { return []int64{int64(math.Float64bits(v))} }
	// 0x4e65fd, 0x4e6612: a zero (or unordered NaN) bound sends the call to 0x4c5780, which returns the base size.
	if lo == 0 || hi == 0 || lo != lo || hi != hi {
		return ret(float64(base))
	}
	x := func(v float64) *big.Float { return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).SetFloat64(v) }
	dMin := in[4] + in[5]       // 0x4e66bb: Z + N
	dMax := in[4] + in[5]*in[6] // 0x4e66e3: Z + N*S
	span := x(float64(hi)).Sub(x(float64(hi)), x(float64(lo)))
	// 0x4e66d9..0x4e66fa: t = (max-min)*(hp-dMin)/(dMax-dMin) + min, stored as f32.
	prod := new(big.Float).SetPrec(64).Mul(span, x(float64(in[3]-dMin)))
	var t float32
	if dMax-dMin == 0 {
		switch prod.Sign() { // x87 masked divide: 0/0 is NaN, x/0 is a signed infinity.
		case 0:
			t = float32(math.NaN())
		case 1:
			t = float32(math.Inf(1))
		default:
			t = float32(math.Inf(-1))
		}
	} else {
		q := new(big.Float).SetPrec(64).Quo(prod, x(float64(dMax-dMin)))
		q.Add(q, x(float64(lo)))
		t, _ = q.Float32()
	}
	// 0x4e6700: R(1000); 0x4e671a/0x4e6720: r*f32(0.0004) - f32(0.2); then (max-min)*that + t.
	k := x(float64(r.R(1000)))
	k.Mul(k, x(float64(float32(0.0004))))
	k.Sub(k, x(float64(float32(0.2))))
	k.Mul(span, k)
	if math.IsNaN(float64(t)) {
		return ret(float64(lo)) // unordered compare at 0x4e6734 takes the lower clamp.
	}
	if math.IsInf(float64(t), 0) {
		if t > 0 {
			return ret(float64(hi))
		}
		return ret(float64(lo))
	}
	k.Add(k, x(float64(t)))
	// 0x4e6734 clamps below to min, 0x4e674a clamps above to max.
	if k.Cmp(x(float64(lo))) < 0 {
		return ret(float64(lo))
	}
	if k.Cmp(x(float64(hi))) <= 0 {
		v, _ := k.Float64()
		return ret(v)
	}
	return ret(float64(hi))
}
