package calc

// depletedItemRetentionNode is 0x520750: whether an item record (index idx into the
// 269-byte item table) still has any other-effect property, so it is kept after its charges run out.
// In: idx, word at +0x4e, bytes +0x62, +0x6c..+0x6e, +0x6f..+0x74, +0x8c.
func init() {
	Nodes["00520750"] = Node{Addr: "00520750",
		In:  []string{"idx", "w4e", "b62", "b6c", "b6d", "b6e", "b6f", "b70", "b71", "b72", "b73", "b74", "b8c"},
		Out: []string{"retain"}, F: depletedItemRetentionNode}
}

func depletedItemRetentionNode(in []int64, r Rand) []int64 {
	if len(in) != 13 {
		panic("depletedItemRetention: expected 13 inputs")
	}
	// Only bits 0..8 of the +0x4e word are examined (loop of 9 at 0x520770..0x52077f).
	keep := in[1]&0x1ff != 0
	for _, v := range in[2:] {
		if v != 0 {
			keep = true
		}
	}
	if keep {
		return []int64{1}
	}
	return []int64{0}
}
