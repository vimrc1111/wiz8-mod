package calc

func init() {
	Nodes["00547510"] = Node{
		Addr: "00547510",
		In:   []string{"combatActive", "combatOpeningBlocked", "combatRound"},
		Out:  []string{"openingEligible"},
		F:    combatOpeningConditionNode,
	}
}

// combatOpeningConditionNode reads the combat byte at 0x683f94 and the
// supplied combatState fields +0xa54 (u8) and +0x4 (u32). There are no tables
// or random calls. Field meanings are recorded in notes/re/iface/547510.tsv.
func combatOpeningConditionNode(in []int64, _ Rand) []int64 {
	// Byte and u32 limits follow the reads at 0x547510, 0x54751e, 0x547528.
	if len(in) != 3 || in[0] < 0 || in[0] > 255 ||
		in[1] < 0 || in[1] > 255 || in[2] < 0 || in[2] > 4294967295 {
		panic("combat opening condition: expected u8 active, u8 blocked, u32 round")
	}
	// 0x547528 sets the last eligible round to 1; 0x54752e returns 1,
	// while all failed conditions return 0 at 0x547534.
	if in[0] != 0 && in[1] == 0 && in[2] <= 1 {
		return []int64{1}
	}
	return []int64{0}
}
