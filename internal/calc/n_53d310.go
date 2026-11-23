package calc

// anyHandAttackAvailableNode receives the selected party slot and the runtime
// character/item fields used by the original range query. No exe table is read:
// 0x685174 and 0x68516c point to runtime character and item records.
func anyHandAttackAvailableNode(in []int64, r Rand) []int64 {
	if len(in) != 11 || in[0] < 0 || in[0] > 7 {
		panic("any hand attack available: expected party slot 0..7 and 11 inputs")
	}
	// Eight party records: 0x517ef4..0x517f02. The outer node selects
	// C = party base + slot*0x1862 (0x53d31f..0x53d32f).
	for _, i := range []int{1, 3} {
		if in[i] < 0 || in[i] > 4294967295 {
			panic("any hand attack available: expected u32 attack kind")
		}
	}
	for _, i := range []int{2, 4, 7, 8, 9, 10} {
		if in[i] < 0 || in[i] > 255 {
			panic("any hand attack available: expected u8 availability and range")
		}
	}
	for _, i := range []int{5, 6} {
		if in[i] < 0 || in[i] > 3 {
			panic("any hand attack available: item index outside four supplied records")
		}
	}
	// The child adapter's in-party flag is needed only for disabled-hand
	// diagnostics. This caller skips disabled hands (0x53d350..0x53d35f),
	// so that flag is never read by the original combined call.
	child := append([]int64{0, 1}, in[1:]...)
	// Two hands, availability at C+0x114d + hand*0x5b:
	// 0x53d350, 0x53d36d and 0x53d370. Preserve the short circuit.
	for hand := 0; hand < 2; hand++ {
		if in[2+2*hand] == 0 {
			continue
		}
		child[0] = int64(hand)
		if charAttackRangeNode(child, r)[0] != -1 { // 0x53d367
			return []int64{1}
		}
	}
	return []int64{0} // Boolean AL, 0x53d381; upper EAX is unspecified.
}

func init() {
	Nodes["0053d310"] = Node{
		Addr: "0053d310",
		In: []string{"partySlot", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable",
			"mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range"},
		Out: []string{"anyHandAttackAvailable"}, F: anyHandAttackAvailableNode,
	}
}
