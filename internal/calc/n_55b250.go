package calc

func init() {
	Nodes["0055b250"] = Node{Addr: "0055b250",
		In:  []string{"itemId", "flags", "category", "acceptMask"},
		Out: []string{"canSell"}, F: shopCanSellItemNode}
}

// Item definitions: base global 0x68516c, 269 bytes per record (0x55b25f..0x55b268).
// Flag byte +0x41 bit 0x2 forbids selling (0x55b268); otherwise the result is
// the category-accept test of 0x55b290, reused via its calc function.
func shopCanSellItemNode(in []int64, r Rand) []int64 {
	if len(in) != 4 {
		panic("shop can sell item: expected four inputs")
	}
	if in[1] < 0 || in[1] > 255 {
		panic("shop can sell item: flags outside byte width")
	}
	if in[1]&2 != 0 {
		if in[0] < 0 || in[0] > 818 {
			panic("shop can sell item: item id outside 819-record table")
		}
		return []int64{0}
	}
	return shopAcceptsItemCategoryNode([]int64{in[0], in[2], in[3]}, r)
}
