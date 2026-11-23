package calc

import (
	"math"
	"math/big"
)

func init() {
	Nodes["004f00c0"] = Node{
		Addr: "004f00c0",
		In: []string{"currentPhase", "partyActionMode", "movementStartPhase",
			"partyActionStage", "partyMovementEnabled", "distanceMovedBits", "distanceLimitBits"},
		Out: []string{"advanceCombatPhase"}, F: combatMovementPhaseLimitNode,
	}
}

// Inputs include the assertion preconditions and IEEE754 f32 distance bits.
// True permits advancing the combat phase (caller exe 0x4e9734..0x4e9751).
func combatMovementPhaseLimitNode(in []int64, r Rand) []int64 {
	if len(in) != 7 {
		panic("combat movement phase limit: expected seven inputs")
	}
	for i, v := range in {
		if v < 0 || v > 4294967295 || (i == 4 && v > 255) {
			panic("combat movement phase limit: expected u32 fields/bits and u8 enabled")
		}
	}
	distance := float64(math.Float32frombits(uint32(in[5])))
	limit := float64(math.Float32frombits(uint32(in[6])))
	// Assertion preconditions: exe 0x4f00ce, 0x4f00f7, 0x4f0117,
	// and 0x4f0176. The last test rejects zero, not negative denominators.
	if in[3] != 1 || in[4] == 0 || !(limit > 0) || uint32(in[2]) == 101 {
		panic("combat movement phase limit: original assertion precondition failed")
	}
	// exe 0x4f015b..0x4f0172: end phase + 1 = 101, then subtract
	// movement start; numerator adds the mode's step (1 or 10) and 1.
	// combatPhaseStepNode matches this node's zero/nonzero mode rule.
	step := uint32(combatPhaseStepNode([]int64{in[1]}, r)[0])
	numerator := uint32(in[0]) - uint32(in[2]) + step + 1
	denominator := int32(uint32(101) - uint32(in[2]))
	f := func(v float64) *big.Float {
		return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).SetFloat64(v)
	}
	// exe 0x4f0146/0x4f0172 stores the x87 quotient to f32 before
	// comparison. Keep its 64-bit significand before that store.
	var fraction float32
	if math.IsNaN(distance) || math.IsInf(distance, 0) || math.IsInf(limit, 0) {
		fraction = float32(distance / limit)
	} else {
		fraction, _ = f(distance).Quo(f(distance), f(limit)).Float32()
	}
	// exe 0x4f019a loads the numerator as a positive 64-bit integer,
	// but 0x4f01ab divides by the signed 32-bit denominator. This ratio
	// stays in x87 precision, unlike the stored distance fraction.
	phaseFraction := f(float64(numerator)).Quo(f(float64(numerator)), f(float64(denominator)))
	// exe 0x4f01b7 accepts less, equal, or unordered; only AL is boolean.
	if math.IsNaN(float64(fraction)) || phaseFraction.Cmp(f(float64(fraction))) <= 0 {
		return []int64{1}
	}
	return []int64{0}
}
