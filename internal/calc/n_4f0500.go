package calc

func init() {
	Nodes["004f0500"] = Node{
		Addr: "004f0500",
		In:   []string{"partyActionMode"},
		Out:  []string{"phaseStep"},
		F:    combatPhaseStepNode,
	}
}

// combatPhaseStepNode receives the full 32-bit party action mode at
// combatState+0x910 (exe 0x4f0505); the global pointer is arena wiring.
func combatPhaseStepNode(in []int64, r Rand) []int64 {
	if len(in) != 1 || in[0] < 0 || in[0] > 4294967295 {
		panic("combatPhaseStep: expected u32 partyActionMode")
	}
	// exe 0x4f050b..0x4f0512: zero gives step 1, every nonzero
	// 32-bit mode gives step 10. No table or random value is used.
	if in[0] == 0 {
		return []int64{1}
	}
	return []int64{10}
}
