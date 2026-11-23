package calc

func init() {
	Nodes["005a6310"] = Node{
		Addr: "005a6310",
		In:   []string{"itemDefinitionIndex", "itemType"},
		Out:  []string{"combatItemTypeException"},
		F:    combatItemTypeExceptionNode,
	}
}

// combatItemTypeExceptionNode takes the instance's definition index and the
// selected definition's type. The index locates a record, not a rule operand:
// 0x68516c supplies the loaded definition base, stride is 0x10d, and type is
// at +0x3e (0x5a6314..0x5a6329; notes/re/loot.md:9). No exe table is used.
func combatItemTypeExceptionNode(in []int64, _ Rand) []int64 {
	if len(in) != 2 || in[0] < 0 || in[0] > 4294967295 || in[1] < 0 || in[1] > 255 {
		panic("combat item type exception: expected u32 definition index and u8 item type")
	}
	// Type codes 2 and 4: 0x5a632d and 0x5a6331; boolean result: 0x5a6335/0x5a6338.
	if in[1] == 2 || in[1] == 4 {
		return []int64{1}
	}
	return []int64{0}
}
