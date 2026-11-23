package calc

func init() {
	Nodes["004f0480"] = Node{
		Addr: "004f0480",
		In:   []string{"partyActionMode", "phase", "minimumPhase"},
		Out:  []string{"phase"},
		F:    combatPhaseRoundingNode,
	}
}

// combatPhaseRoundingNode models the phase pointer's final value. The combat
// state pointer at exe 0x6836a8 only connects partyActionMode (+0x910).
// All inputs are u32; legal returning calls have a final phase in 1..100.
func combatPhaseRoundingNode(in []int64, r Rand) []int64 {
	if len(in) != 3 {
		panic("combatPhaseRounding: expected partyActionMode, phase, minimumPhase")
	}
	for _, v := range in {
		if v < 0 || v > 4294967295 {
			panic("combatPhaseRounding: expected u32 inputs")
		}
	}
	phase, lower := uint32(in[1]), uint32(in[2])
	if in[0] != 0 {
		// exe 0x4f0496..0x4f04a7: half-step 5, step 10; the addition
		// wraps at 32 bits before the unsigned remainder is removed.
		phase += 5
		phase -= phase % 10
		// exe 0x4f04a9..0x4f04bf: ceil the lower bound to step 10,
		// retaining 32-bit wraparound even near the u32 limit.
		if rem := lower % 10; rem != 0 {
			lower += 10 - rem
		}
		// exe 0x4f04c1 supplies cap 100. Child 0x5179f0 tests the
		// upper bound first, then the lower bound; inverted bounds
		// therefore do not behave like a conventional min/max clamp.
		if phase > 100 {
			phase = 100
		} else if phase < lower {
			phase = lower
		}
	}
	// exe 0x4f04cd..0x4f04e9 asserts this invariant in every mode.
	// The emulator's existing srAssertFail stub also stops such calls.
	if phase == 0 || phase > 100 {
		panic("combatPhaseRounding: final phase outside 1..100 (srAssertFail)")
	}
	return []int64{int64(phase)}
}
