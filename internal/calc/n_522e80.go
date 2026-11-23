package calc

func init() {
	Nodes["00522e80"] = Node{Addr: "00522e80", In: []string{"itemId", "itemType", "count4", "count5"}, Out: []string{"count4", "count5"}, F: itemCountNormalizationNode}
}

// itemCountNormalizationNode normalizes the two u8 quantity fields of an item
// instance (+0x4, +0x5) according to the item definition type byte (+0x66,
// exe 0x522e9e). itemId -1 (exe 0x522e86) is an empty slot: unchanged.
func itemCountNormalizationNode(in []int64, r Rand) []int64 {
	if len(in) != 4 {
		panic("itemCountNormalization: expected itemId, itemType, count4, count5")
	}
	id, t, c4, c5 := in[0], in[1], in[2], in[3]
	if id == -1 {
		return []int64{c4, c5}
	}
	switch {
	case t == 1: // exe 0x522ec7
		switch {
		case c5 == 0:
			if c4 == 0 {
				c4 = 1 // exe 0x522ee5: both zero becomes quantity one
			}
		case c4 == 0:
			c4, c5 = c5, 0 // exe 0x522ed9: move second to first
		default:
			c5 = 0 // exe 0x522ed5: first wins
		}
	case t >= 2 && t <= 4: // exe 0x522ea9
		if c4 != 0 {
			if c5 > 0 {
				c4 = 0 // exe 0x522ebc: second wins
			} else {
				c4, c5 = 0, c4 // exe 0x522ec0: move first to second
			}
		}
	}
	return []int64{c4, c5}
}
