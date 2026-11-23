package calc

func init() {
	Nodes["00547010"] = Node{
		Addr: "00547010",
		In:   []string{"sourceCharm", "targetCharm"},
		Out:  []string{"disposition"},
		F:    characterDispositionNode,
	}
}

// characterDispositionNode takes the two resolved C+0xa35 fields, independent
// of character indices and the runtime array pointer (notes/re/iface/547010.tsv).
// All 32-bit patterns are accepted as unsigned values, including the high bit.
func characterDispositionNode(in []int64, r Rand) []int64 {
	if len(in) != 2 {
		panic("character_disposition: expected sourceCharm, targetCharm")
	}
	for _, v := range in {
		if v < 0 || v > 4294967295 { // Full 32-bit field at 0x54702e/0x547049/0x54706c.
			panic("character_disposition: expected u32 charm fields")
		}
	}
	// 0x547055/0x547075: friendly = 2; 0x547079: hostile = 1.
	// 0x547035..0x547077 classify each field by zero versus nonzero, not equality.
	if (in[0] == 0) == (in[1] == 0) {
		return []int64{2}
	}
	return []int64{1}
}
