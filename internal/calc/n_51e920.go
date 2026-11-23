package calc

func init() {
	Nodes["0051e920"] = Node{
		Addr: "0051e920",
		In:   []string{"itemIndex", "charges", "increment", "chargeBase", "chargeDice", "chargeSides"},
		Out:  []string{"charges", "itemIndex"},
		F:    itemChargeIncrementNode,
	}
}

// itemChargeIncrementNode uses the selected item's supplied charge dice fields.
// The item table pointer is 0x68516c, record stride 0x10d; the charge dice
// fields are +0x67 (i16), +0x69 (u8), +0x6a (u8), at 0x51e940..0x51e959.
// No static table contents or random draws are needed for this calculation.
func itemChargeIncrementNode(in []int64, _ Rand) []int64 {
	if len(in) != 6 || in[0] < 0 || in[0] > 4294967295 ||
		in[1] < 0 || in[1] > 255 || in[2] < -2147483648 || in[2] > 2147483647 ||
		in[3] < -32768 || in[3] > 32767 || in[4] < 0 || in[4] > 255 || in[5] < 0 || in[5] > 255 {
		panic("item charge increment: expected u32 index, u8 charges, i32 increment, i16 base, u8 dice and sides")
	}
	// 0x51e92c: the sum wraps to u8 before the signed comparison at 0x51e967.
	charges := int64(uint8(in[1] + in[2]))
	limit := in[3] + in[4]*in[5]
	if charges >= limit {
		// 0x51e971: even a negative or oversized limit is stored as one byte.
		charges = int64(uint8(limit))
	}
	return []int64{charges, in[0]}
}
