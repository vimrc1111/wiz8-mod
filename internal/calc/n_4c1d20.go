package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
)

// Number of cycle slots: the loop bound is a signed byte (exe 0x4c1e3f).
const monsterInstanceScaleSlots = 127

var monsterInstanceScaleNames = func() []string {
	n := []string{"variantChancePercent", "sizeMin", "sizeMax", "cycleEnabled", "cycleOffset",
		"cycleMin", "cycleMax", "sizePrev", "minScalePrev", "minScaleFlagPrev",
		"cycleCount", "cycleListCount", "driftXPrev", "driftYPrev", "scaleCap",
		"driftLow", "driftHigh", "swayLow", "swayHigh"}
	for i := 0; i < monsterInstanceScaleSlots; i++ {
		n = append(n, fmt.Sprintf("cycleRecord%dPrev", i))
	}
	for i := 0; i < monsterInstanceScaleSlots; i++ {
		n = append(n, fmt.Sprintf("cycleList%dPrev", i))
	}
	return n
}()

func init() {
	out := []string{"instanceVariantFlag", "driftX", "driftY", "phase", "driftXTotal", "driftYTotal", "swayValue",
		"size", "minScale", "minScaleFlag"}
	for i := 0; i < monsterInstanceScaleSlots; i++ {
		out = append(out, fmt.Sprintf("cycleRecord%d", i))
	}
	for i := 0; i < monsterInstanceScaleSlots; i++ {
		out = append(out, fmt.Sprintf("cycleList%d", i))
	}
	Nodes["004c1d20"] = Node{Addr: "004c1d20", In: monsterInstanceScaleNames, Out: out, F: monsterInstanceScaleNode}
}

// Constants read from the player's exe: 0x5ebb38 = 1.0f, 0x5ec128 = f32(0.001),
// 0x5ebc40 = 500.0f, 0x5ec318 = double 2*pi. (0x5ebb34 = 0.0f is compared to by sign.)
var monsterInstanceScaleConst, monsterInstanceScaleConstErr = monsterInstanceScaleLoad()

func monsterInstanceScaleLoad() ([4]float64, error) {
	var t [4]float64
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("monster instance scale: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	for i, va := range []uint32{0x5ebb38, 0x5ec128, 0x5ebc40, 0x5ec318} {
		o, err := peOffset(b, va)
		if err != nil {
			return t, err
		}
		w := 4
		if i == 3 {
			w = 8
		}
		if len(b)-o < w {
			return t, fmt.Errorf("monster instance scale: truncated constant at %#x", va)
		}
		if w == 8 {
			t[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[o:]))
		} else {
			t[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[o:])))
		}
	}
	return t, nil
}

// Rolls the model's size, its animation-cycle scales and the instance's
// drift/sway parameters. x87 arithmetic keeps a 64-bit significand until an
// explicit float32 store. Inputs are raw float32 bits; floats must be finite.
func monsterInstanceScaleNode(in []int64, r Rand) []int64 {
	if monsterInstanceScaleConstErr != nil {
		panic(monsterInstanceScaleConstErr)
	}
	if len(in) != len(monsterInstanceScaleNames) {
		panic("monster instance scale: wrong input count")
	}
	for i, v := range in {
		if i == 0 || i == 11 {
			if v < -2147483648 || v > 2147483647 {
				panic("monster instance scale: expected i32 input")
			}
		} else if v < 0 || v > 4294967295 {
			panic("monster instance scale: expected u32 input")
		}
	}
	if in[3] > 255 || in[9] > 255 {
		panic("monster instance scale: expected byte field")
	}
	f := func(v float64) *big.Float { return new(big.Float).SetPrec(64).SetFloat64(v) }
	ld := func(i int) *big.Float {
		v := float64(math.Float32frombits(uint32(in[i])))
		if math.IsNaN(v) || math.IsInf(v, 0) {
			panic("monster instance scale: float inputs must be finite")
		}
		return f(v)
	}
	mul := func(a, b *big.Float) *big.Float { return f(0).Mul(a, b) }
	sub := func(a, b *big.Float) *big.Float { return f(0).Sub(a, b) }
	add := func(a, b *big.Float) *big.Float { return f(0).Add(a, b) }
	st := func(v *big.Float) int64 { x, _ := v.Float32(); return int64(math.Float32bits(x)) }
	one, milli, scale500, twoPi := f(monsterInstanceScaleConst[0]), f(monsterInstanceScaleConst[1]),
		f(monsterInstanceScaleConst[2]), f(monsterInstanceScaleConst[3])
	// lerp: (hi-lo) * (roll * 0.001) + lo
	lerp := func(lo, hi *big.Float, roll int) *big.Float {
		return add(mul(mul(f(float64(roll)), milli), sub(hi, lo)), lo)
	}

	out := make([]int64, 10+2*monsterInstanceScaleSlots)
	copy(out[10:], in[19:])
	out[7], out[8], out[9] = in[7], in[8], in[9]
	out[4], out[5] = in[12], in[13]

	if int64(r.R(100)) < in[0] {
		out[0] = 1
	}
	if ld(1).Sign() != 0 && ld(2).Sign() != 0 {
		if ld(1).Cmp(ld(2)) > 0 {
			panic("monster instance scale: sizeMin > sizeMax (engine assertion)")
		}
		out[7] = st(lerp(ld(1), ld(2), r.R(1000)))
	}
	if in[3] != 0 {
		base := f(float64(math.Float32frombits(uint32(st(lerp(ld(5), ld(6), r.R(1000)))))))
		n := int(int8(in[10]))
		for i := 0; i < n; i++ {
			if int64(i) >= in[10] {
				panic("monster instance scale: negative cycle count")
			}
			v := add(base, ld(4))
			if one.Cmp(v) >= 0 {
				v = one
			}
			out[10+i] = st(v)
			if int64(i) < in[11] {
				out[10+monsterInstanceScaleSlots+i] = st(v)
			}
		}
	}
	d1 := mul(lerp(ld(15), ld(16), r.R(1000)), scale500)
	out[1] = st(d1)
	d2 := mul(lerp(ld(17), ld(18), r.R(1000)), scale500)
	out[2] = st(d2)
	phase := mul(f(float64(r.R(1000))), milli)
	out[3] = st(phase)
	arg, _ := mul(phase, twoPi).Float64()
	d1s, d2s := f(0), f(0)
	d1s.SetFloat64(float64(math.Float32frombits(uint32(out[1]))))
	d2s.SetFloat64(float64(math.Float32frombits(uint32(out[2]))))
	out[6] = st(add(mul(f(math.Sin(arg)), d2s), d1s))
	out[4] = st(add(d1s, ld(12)))
	out[5] = st(add(d1s, ld(13)))
	if ld(14).Cmp(one) < 0 {
		out[8] = in[14]
		out[9] = 1
	}
	return out
}
