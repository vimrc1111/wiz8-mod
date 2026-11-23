package calc

// charAttackRangeNode takes a party character and independently selected item records.
// Runtime item data is supplied as input, not copied from the executable.
func charAttackRangeNode(in []int64, _ Rand) []int64 {
	if len(in) != 12 {
		panic("character attack range: expected 12 inputs")
	}
	// 0x517ef4..0x517f02 resolves eight party slots. 0x519aca accepts
	// hand 2 (either); 0x519b1d handles individual hands 0 and 1.
	if in[0] < 0 || in[0] > 2 || in[1] < 1 || in[1] > 255 {
		panic("character attack range: expected hand 0..2 and nonzero u8 in-party flag")
	}
	for _, i := range []int{2, 4} {
		if in[i] < 0 || in[i] > 4294967295 {
			panic("character attack range: expected u32 attack kind")
		}
	}
	for _, i := range []int{3, 5, 8, 9, 10, 11} {
		if in[i] < 0 || in[i] > 255 {
			panic("character attack range: expected u8 availability and item ranges")
		}
	}
	for _, i := range []int{6, 7} {
		if in[i] < 0 || in[i] > 3 {
			panic("character attack range: item index outside four supplied records")
		}
	}
	reach := func(hand int) int64 {
		// C+0x1149/+0x114d, stride 0x5b: 0x519b2d..0x519b53.
		if in[3+2*hand] == 0 {
			return -1 // 0x519b4e, after diagnostic formatting.
		}
		kind := in[2+2*hand]
		// 0x519b59/0x519b5e: kinds 1 and 3 use the equipped item;
		// 0x519b63 returns short range (0) for every other kind.
		if kind != 1 && kind != 3 {
			return 0
		}
		// 0x519b75: slots 6/7 at C+0xf5d with stride 12;
		// 0x519b92: item record stride 0x10d, range byte I+0x47.
		return in[8+int(in[6+hand])]
	}
	if in[0] < 2 {
		return []int64{reach(int(in[0]))}
	}
	// 0x519aed..0x519b16 starts at -1, skips disabled hands and
	// compares signed results. All item ranges are zero-extended u8.
	return []int64{max(reach(0), reach(1))}
}

func init() {
	Nodes["00519ac0"] = Node{
		Addr: "00519ac0",
		In:   []string{"hand", "characterInParty", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range"},
		Out:  []string{"attackRange"}, F: charAttackRangeNode,
	}
}
