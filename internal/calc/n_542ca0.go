package calc

// chooseHandAttackModeNode consumes the selected hand's state and item record.
// Address resolution is represented by hand and equippedItem; item data is an
// input from the loaded database, rather than an embedded original item table.
func chooseHandAttackModeNode(in []int64, r Rand) []int64 {
	if len(in) != 5 || in[0] < 0 || in[0] > 1 || in[1] < 0 || in[1] > 4294967295 ||
		in[2] < 0 || in[2] > 65535 || in[3] < -2147483648 || in[3] > 2147483647 ||
		in[4] < 0 || in[4] > 65535 {
		panic("choose hand attack mode: expected hand 0/1, u32 handAttackSource, u16 unarmedModes, i32 equippedItem, u16 weaponModes")
	}
	// C+0x1149+0x5b*hand selects the source (0x542cbe..0x542cc8).
	mask := in[2] // C+0x117a+0x5b*hand: unarmed modes.
	if in[1] != 0 {
		// Slot 6 for main hand, 7 for off hand (0x542cd8); the slot's
		// item index selects a 269-byte record's +0x4e (0x542d24).
		if in[3] == -1 {
			panic("choose hand attack mode: missing equipped item (original asserts)")
		}
		mask = in[4]
	}
	// The scan at 0x542d6d..0x542da1 matches 0x542db0, including
	// ignored high bits, one R(9), and masks with no returning path.
	return chooseAttackModeNode([]int64{mask}, r)
}

func init() {
	Nodes["00542ca0"] = Node{
		Addr: "00542ca0",
		In:   []string{"hand", "handAttackSource", "unarmedModes", "equippedItem", "weaponModes"},
		Out:  []string{"attackMode"}, F: chooseHandAttackModeNode,
	}
}
