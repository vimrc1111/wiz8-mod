package calc

func init() {
	Nodes["0055b290"] = Node{Addr: "0055b290",
		In:  []string{"itemId", "category", "acceptMask"},
		Out: []string{"accepts"}, F: shopAcceptsItemCategoryNode}
}

// Item definitions: base global 0x68516c, 819 records of 269 bytes
// (notes/re/loot.md:9); category is the byte at +0x3e (0x55b2ab). The merchant
// record's u32 at +0x2d6 is the accept mask. The 26-entry jump table at 0x55b530
// maps category c in 0..25 to mask bit c (0x55b2bf..0x55b529); categories above
// 25 (0x55b2af) reject. AL is the boolean result.
func shopAcceptsItemCategoryNode(in []int64, r Rand) []int64 {
	if len(in) != 3 {
		panic("shop accepts item category: expected three inputs")
	}
	id, cat, mask := in[0], in[1], in[2]
	if id < 0 || id > 818 || cat < 0 || cat > 255 || mask < 0 || mask > 0xffffffff {
		panic("shop accepts item category: item id outside 819-record table or field width")
	}
	if cat <= 25 && mask>>uint(cat)&1 != 0 {
		return []int64{1}
	}
	return []int64{0}
}
