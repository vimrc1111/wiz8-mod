package calc

func init() {
	Nodes["004eceb0"] = Node{
		Addr: "004eceb0",
		In:   []string{"partyActionMode", "currentPhase", "actionPhase", "previousPhase"},
		Out:  []string{"actionPhase", "previousPhase"},
		F:    actionPhaseRebaseNode,
	}
}

// actionPhaseRebaseNode rebases the action record onto the party's current
// phase. All four fields are u32; calls that reach the child's phase assertion
// have no normal output, matching combatPhaseRoundingNode's input contract.
func actionPhaseRebaseNode(in []int64, r Rand) []int64 {
	if len(in) != 4 {
		panic("actionPhaseRebase: expected partyActionMode, currentPhase, actionPhase, previousPhase")
	}
	for _, v := range in {
		// exe 0x4eceb5..0x4ecec9 and child 0x4f0485 read u32 fields.
		if v < 0 || v > 4294967295 {
			panic("actionPhaseRebase: expected u32 inputs")
		}
	}
	current := uint32(in[1])
	// exe 0x4ecec5..0x4ecec9: the phase delta and sum wrap at 32 bits.
	phase := uint32(in[2]) + (current - uint32(in[3]))
	// exe 0x4eced4 supplies cap 100 to 0x5179f0. That child checks
	// the unsigned upper bound first, even when current exceeds the cap.
	if phase > 100 {
		phase = 100
	} else if phase < current {
		phase = current
	}
	// exe 0x4ecee8 uses current as the rounding child's lower bound.
	phase = uint32(combatPhaseRoundingNode([]int64{in[0], int64(phase), in[1]}, r)[0])
	// exe 0x4ecef8 saves current to the action record's +0x9c field.
	return []int64{int64(phase), in[1]}
}
