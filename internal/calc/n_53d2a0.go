package calc

// handAttackAvailableNode selects one party character and one hand. Character
// and item records are runtime inputs; no executable data table is needed.
func handAttackAvailableNode(in []int64, r Rand) []int64 {
	if len(in) != 12 || in[0] < 0 || in[0] > 7 || in[1] < 0 || in[1] > 1 {
		// Eight party slots: 0x517ef4..0x517f02; two hands: 0x53d2a5.
		panic("hand attack available: expected party slot 0..7, hand 0..1 and 12 inputs")
	}
	for _, i := range []int{2, 4} {
		if in[i] < 0 || in[i] > 4294967295 {
			panic("hand attack available: expected u32 attack kind")
		}
	}
	for _, i := range []int{3, 5, 8, 9, 10, 11} {
		if in[i] < 0 || in[i] > 255 {
			panic("hand attack available: expected u8 availability and range")
		}
	}
	for _, i := range []int{6, 7} {
		if in[i] < 0 || in[i] > 3 {
			panic("hand attack available: item index outside four supplied records")
		}
	}
	// C = party base + slot*0x1862; selected availability is
	// C+0x114d + hand*0x5b (0x53d2c5..0x53d2e9).
	if in[3+2*int(in[1])] == 0 {
		return []int64{0} // 0x53d2f5: Boolean AL, upper EAX unspecified.
	}
	// 0x53d2fd queries the same hand, then 0x53d305 tests range != -1.
	// The child's in-party flag is read only for disabled-hand diagnostics;
	// this caller already skipped that path, so its synthetic value is unused.
	child := append([]int64{in[1], 1}, in[2:]...)
	if charAttackRangeNode(child, r)[0] != -1 {
		return []int64{1}
	}
	return []int64{0}
}

func init() {
	Nodes["0053d2a0"] = Node{
		Addr: "0053d2a0",
		In: []string{"partySlot", "hand", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable",
			"mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range"},
		Out: []string{"handAttackAvailable"}, F: handAttackAvailableNode,
	}
}
