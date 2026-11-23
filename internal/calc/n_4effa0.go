package calc

import (
	"fmt"
	"math"
	"math/big"
)

func init() {
	names := []string{"partyActionMode"}
	// Use the child's eight complete records, without its optional pointer inputs.
	for slot := 0; slot < 8; slot++ {
		for _, field := range []string{"Active", "MovementExcludedCondition", "MovementBuffStrength", "MovementBuffBonusPercent", "MovementBuffDuration"} {
			names = append(names, fmt.Sprintf("slot%d%s", slot, field))
		}
	}
	Nodes["004effa0"] = Node{Addr: "004effa0", In: names,
		Out: []string{"movementAllowanceBits"}, F: partyMovementAllowanceNode}
}

// partyMovementAllowanceNode returns the IEEE754 double bits of the x87 result.
// The allowance's physical unit is unproven; no character field is modified.
func partyMovementAllowanceNode(in []int64, r Rand) []int64 {
	if len(in) != 41 || in[0] < 0 || in[0] > 4294967295 {
		panic("party movement allowance: expected u32 mode and eight records")
	}
	child := make([]int64, 42)
	// The original always passes a nonnull local output (exe 0x4effc1).
	child[0] = 1
	copy(child[2:], in[1:])
	effect := partyMovementEffectAverageNode(child, r)
	// Base weights 1 and 1.5: exe 0x4effaf and 0x4effb9; mode 2 only.
	base := 1.0
	if in[0] == 2 {
		base = 1.5
	}
	// These are scalar f32 constants, not tables: exe 0x5ed490 = f32(0.01),
	// exe 0x5ec0a8 = 10000. Preserve the f32 value and the x87 64-bit
	// significand at each operation before rounding the output to double.
	f := func(v float64) *big.Float {
		return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).SetFloat64(v)
	}
	value := f(base)
	if effect[0] != 0 {
		value.Mul(f(float64(effect[1])), f(float64(float32(0.01))))
		value.Add(value, f(base))
	}
	value.Mul(value, f(10000))
	out, _ := value.Float64()
	return []int64{int64(math.Float64bits(out))}
}
