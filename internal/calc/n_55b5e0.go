package calc

import (
	"math"
	"math/big"
)

func init() {
	in := []string{"itemDefinitionIndex", "quantity", "identified", "itemType", "stackable", "baseValue",
		"rateLowBits", "rateHighBits", "highRateDirection"}
	for _, slot := range []string{"0", "1", "2", "3", "4", "5", "6", "7"} {
		for _, field := range []string{"Active", "Life", "Status", "Skill"} {
			in = append(in, "member"+slot+field)
		}
	}
	Nodes["0055b5e0"] = Node{Addr: "0055b5e0", In: in, Out: []string{"price"}, F: shopTransactionPriceNode}
}

// shopTransactionPriceNode (exe 0x55b5e0). Inputs: item instance index (+0x0), quantity (+0x4), identified
// flag (+0x6); definition type (+0x3e), stackable (+0x66), base value (+0x86); merchant rates f32 bits
// (+0x2ce, +0x2d2); the direction byte (3rd arg); then active/life/status/skill for the 8 party slots
// (read by 0x554490 for skill 0x16). Scalar f32/f64 constants, not tables: 0x5ee8b8 = f32(0.002),
// 0x5ebe80 = f64(0.5), 0x5ebc7c = f32(0.5), 0x5ebb34 = f32(0), 0x5ebc3c = f32(0.1).
// Every x87 operation keeps a 64-bit significand (control word 0x037f).
func shopTransactionPriceNode(in []int64, r Rand) []int64 {
	if len(in) != 41 {
		panic("shop transaction price: expected 41 inputs")
	}
	if in[2] < 0 || in[2] > 255 || in[8] < 0 || in[8] > 255 || in[6] < 0 || in[6] > 0xffffffff || in[7] < 0 || in[7] > 0xffffffff {
		panic("shop transaction price: field width overflow")
	}
	ext := func(v float64) *big.Float { return new(big.Float).SetPrec(64).SetFloat64(v) }
	f32 := func(v int64) float32 { return math.Float32frombits(uint32(v)) }
	low, high := f32(in[6]), f32(in[7])
	if math.IsNaN(float64(low)) || math.IsInf(float64(low), 0) || math.IsNaN(float64(high)) || math.IsInf(float64(high), 0) {
		panic("shop transaction price: non-finite merchant rate")
	}
	value := itemValueNode([]int64{in[0], in[1], in[3], in[4], in[5]}, r)[0]  // 0x51b840
	skill := bestPartySkillNode(append([]int64{0x16, 0, 0}, in[9:]...), r)[0] // 0x554490, skill 0x16, no index output

	// 0x55b5f7..0x55b61c: best skill (unsigned) * 0.002, stored as f32.
	bonus32, _ := new(big.Float).SetPrec(64).Mul(ext(float64(skill)), ext(float64(float32(0.002)))).Float32()
	bonus := ext(float64(bonus32))
	// 0x55b620..0x55b643: |high - low|; a bonus above half the gap is clamped to half the gap (f32 store).
	gap := new(big.Float).SetPrec(64).Sub(ext(float64(high)), ext(float64(low)))
	gap.Abs(gap)
	if bonus.Cmp(new(big.Float).SetPrec(64).Mul(gap, ext(0.5))) > 0 {
		clamp, _ := new(big.Float).SetPrec(64).Mul(gap, ext(float64(float32(0.5)))).Float32()
		bonus = ext(float64(clamp))
	}
	// 0x55b651..0x55b66b: direction byte nonzero uses high - bonus, otherwise low + bonus.
	mult := new(big.Float).SetPrec(64)
	if in[8] != 0 {
		mult.Sub(ext(float64(high)), bonus)
	} else {
		mult.Add(ext(float64(low)), bonus)
	}
	// 0x55b66f..0x55b67e: a multiplier not above zero is replaced by f32(0.1).
	if mult.Sign() <= 0 {
		mult = ext(float64(float32(0.1)))
	}
	// 0x55b684..0x55b6e5: a stackable (flag 1) multi-quantity non-type-4 item prices the base value and
	// multiplies the truncated unit price by the quantity (32-bit wrap); otherwise the 0x51b840 value
	// (as signed 32-bit) is scaled and truncated (_ftol keeps the low 32 bits).
	trunc := func(x *big.Float) int32 {
		v, _ := x.Int64()
		if v == math.MaxInt64 || v == math.MinInt64 {
			panic("shop transaction price: _ftol overflow")
		}
		return int32(uint32(v))
	}
	var price int32
	if in[4] == 1 && in[1] > 1 && in[3] != 4 {
		price = int32(uint32(trunc(new(big.Float).SetPrec(64).Mul(mult, ext(float64(in[5]))))) * uint32(in[1]))
	} else {
		price = trunc(new(big.Float).SetPrec(64).Mul(mult, ext(float64(int32(value)))))
	}
	// 0x55b6e7..0x55b6f6: unidentified items sell for a quarter (signed, toward zero).
	if in[2] == 0 {
		price = (price + (price>>31)&3) >> 2
	}
	if price == 0 { // 0x55b6f9..0x55b6fd: minimum price 1.
		price = 1
	}
	return []int64{int64(price)}
}
