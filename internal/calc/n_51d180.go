package calc

func init() {
	Nodes["0051d180"] = Node{
		Addr: "0051d180",
		In:   []string{"itemID", "itemCursed", "curseKnown"},
		Out:  []string{"curseIsKnown"},
		F:    itemCurseKnownNode,
	}
}

// itemCurseKnownNode receives the selected template's curse flag, rather than
// the template table itself. The instance ID is u32; both flags are u8.
func itemCurseKnownNode(in []int64, r Rand) []int64 {
	// 0x51d186: 4294967295 is the empty item ID. Flag widths come from
	// instance +0xb at 0x51d1aa and template +0x8c at 0x51d19e.
	const emptyItemID int64 = 4294967295
	const maxFlag int64 = 255
	if len(in) != 3 || in[0] < 0 || in[0] > emptyItemID || in[1] < 0 || in[1] > maxFlag || in[2] < 0 || in[2] > maxFlag {
		panic("itemCurseKnown: expected u32 itemID, u8 itemCursed, u8 curseKnown")
	}
	// 0x51d1b1 / 0x51d1b4 define only the low-byte boolean result.
	if in[0] != emptyItemID && in[1] != 0 && in[2] != 0 {
		return []int64{1}
	}
	return []int64{0}
}
