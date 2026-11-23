package calc

func init() {
	Nodes["0051bfd0"] = Node{
		Addr: "0051bfd0",
		In:   []string{"itemIndex", "quantity", "unitWeight"},
		Out:  []string{"stackWeight"},
		F:    itemWeightNode,
	}
}

// itemWeightNode takes the selected definition's weight as an input, not an
// embedded item table. The definition base is 0x68516c and the record stride is
// 269 (notes/re/formulas.md:9); +0x8a is u16 (0x51bff3).
func itemWeightNode(in []int64, r Rand) []int64 {
	if len(in) != 3 || in[0] < -1 || in[0] > 2147483647 ||
		in[1] < 0 || in[1] > 255 || in[2] < 0 || in[2] > 65535 {
		panic("item weight: expected itemIndex -1..2147483647, u8 quantity and u16 unitWeight")
	}
	// 0x51bfd8: -1 denotes an empty item, regardless of quantity or weight.
	if in[0] == -1 {
		return []int64{0}
	}
	quantity := in[1]
	// 0x51c011: a zero quantity uses one unit; other quantities are u8
	// (0x51bffd). The largest product is 65535*255, so u32 cannot overflow.
	if quantity == 0 {
		quantity = 1
	}
	return []int64{quantity * in[2]}
}
