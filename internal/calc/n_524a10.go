package calc

func init() {
	Nodes["00524a10"] = Node{
		Addr: "00524a10",
		In:   []string{"characterIndex", "characterSlotActive", "currentHP", "mainStatus"},
		Out:  []string{"eligible"},
		F:    characterActionEligibilityNode,
	}
}

// characterActionEligibilityNode models the candidate action gate identified in
// notes/re/abilities.md:74. Only the low return byte is the boolean result.
// The record fields are scalar inputs; this node reads no static exe tables.
func characterActionEligibilityNode(in []int64, r Rand) []int64 {
	if len(in) != 4 || in[0] < -2147483648 || in[0] > 2147483647 ||
		in[1] < 0 || in[1] > 255 || in[2] < 0 || in[2] > 4294967295 ||
		in[3] < 0 || in[3] > 4294967295 {
		panic("characterActionEligibility: expected i32 index, u8 active, u32 HP bits and status")
	}
	// 0x524a14..0x524a1e: the signed character index must be in [0, 8).
	const characterSlots = 8
	// 0x524a60: unsigned main status must be below 0xd.
	const statusLimit = 13
	// 0x524a32 and 0x524a56: both active and HP are tested for nonzero;
	// HP's high bit does not make it ineligible.
	if in[0] >= 0 && in[0] < characterSlots && in[1] != 0 && in[2] != 0 && in[3] < statusLimit {
		return []int64{1}
	}
	return []int64{0}
}
