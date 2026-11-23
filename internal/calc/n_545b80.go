package calc

func init() {
	Nodes["00545b80"] = Node{
		Addr: "00545b80",
		In:   []string{"characterIndex", "characterSlotActive", "currentHP", "mainStatus", "mainHandEnabled"},
		Out:  []string{"guardAvailable"},
		F:    guardAvailableNode,
	}
}

// guardAvailableNode returns the boolean low byte (0x545b8e, 0x545bc1).
// The character's main/secondary records supply scalar fields, with no static
// exe tables. Record pointers and the unused upper return bytes are not inputs.
func guardAvailableNode(in []int64, r Rand) []int64 {
	// Main-hand enable is a u8 at C+0x114d (0x545bb9).
	if len(in) != 5 || in[4] < 0 || in[4] > 255 {
		panic("guard available: expected index, slot active, HP, status and u8 main-hand enable")
	}
	// 0x545b86 calls the original action gate: signed index 0..7, active
	// slot, nonzero u32 HP, and unsigned status below 13 (0x524a10).
	eligible := characterActionEligibilityNode(in[:4], r)[0]
	// Guard applies the stricter unsigned status cutoff 12 at 0x545bac.
	const guardStatusLimit = 12
	if eligible != 0 && in[3] < guardStatusLimit && in[4] != 0 {
		return []int64{1}
	}
	return []int64{0}
}
