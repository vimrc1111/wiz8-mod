package calc

// chooseAttackHandNode compares the selected character's two enabled hands.
// Character and item records are runtime inputs; no executable table is used.
func chooseAttackHandNode(in []int64, r Rand) []int64 {
	if len(in) != 15 || in[0] < 0 || in[0] > 7 {
		// Eight party slots, character stride 0x1862: 0x517ef4 and 0x53d399..0x53d3b1.
		panic("choose attack hand: expected party slot 0..7 and 15 inputs")
	}
	for _, i := range []int{1, 3, 11, 13} {
		if in[i] < 0 || in[i] > 4294967295 {
			panic("choose attack hand: expected u32 attack kind and count")
		}
	}
	for _, i := range []int{2, 4, 7, 8, 9, 10} {
		if in[i] < 0 || in[i] > 255 {
			panic("choose attack hand: expected u8 availability and range")
		}
	}
	for _, i := range []int{5, 6} {
		if in[i] < 0 || in[i] > 3 {
			panic("choose attack hand: item index outside four supplied records")
		}
	}
	for _, i := range []int{12, 14} {
		if in[i] < -2147483648 || in[i] > 2147483647 {
			panic("choose attack hand: expected i32 initiative")
		}
	}
	// 0x53d3b3..0x53d3c0: initial count 0, initiative -999, hand -1.
	bestHand, bestCount, bestInitiative := int64(-1), int64(0), int64(-999)
	// The child only reads the in-party flag on disabled-hand diagnostics;
	// this caller skips disabled hands (0x53d3e7..0x53d3f0).
	child := append([]int64{0, 1}, in[1:11]...)
	// Two hands with stride 0x5b: availability C+0x114d,
	// count C+0x115e, initiative C+0x1166 (0x53d3e7..0x53d437).
	for hand := 0; hand < 2; hand++ {
		if in[2+2*hand] == 0 {
			continue
		}
		child[0] = int64(hand)
		if charAttackRangeNode(child, r)[0] == -1 {
			continue
		}
		count, initiative := in[11+2*hand], in[12+2*hand]
		// 0x53d40f..0x53d420: unsigned count, signed initiative;
		// equal values retain the earlier hand, including the initial sentinel.
		if count > bestCount || count == bestCount && initiative > bestInitiative {
			bestHand, bestCount, bestInitiative = int64(hand), count, initiative
		}
	}
	return []int64{bestHand}
}

func init() {
	Nodes["0053d390"] = Node{
		Addr: "0053d390",
		In: []string{"partySlot", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable",
			"mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range",
			"mainAttackCount", "mainInitiative", "offAttackCount", "offInitiative"},
		Out: []string{"bestAttackHand"}, F: chooseAttackHandNode,
	}
}
