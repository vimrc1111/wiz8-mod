package calc

// The first 69 inputs describe the selected records exactly as in 0x51d610.
// The remaining inputs include both addresses because 0x51d8d4 compares the
// instance address with the character's equipment records, rather than IDs.
// Both addresses must name valid objects; definition indices are -1 or below
// the loaded definition count (entry assertions at 0x51d805..0x51d87c).
func itemActivationAllowedNode(in []int64, r Rand) []int64 {
	if len(in) != 77 {
		panic("item activation: expected 77 inputs")
	}
	if in[75] <= 0 || in[75] > 4294967295 || in[74] <= 0 || in[74] > 4294967295 ||
		in[76] <= 0 || in[76] > 2147483647 || in[0] < -1 || in[0] >= in[76] {
		panic("item activation: invalid object or definition index")
	}
	for _, v := range in[69:74] {
		if v < 0 || v > 255 {
			panic("item activation: expected byte fields")
		}
	}
	no := []int64{0}
	if in[0] == -1 {
		return no // Empty instance: 0x51d881..0x51d88b.
	}
	use := append([]int64(nil), in[:69]...)
	// The child adapter restricts its synthetic record selector to 0..3.
	// The selector only locates a record (0x51d61b..0x51d63a); all selected
	// record contents are already inputs, so normalize that addressing token.
	use[0] = 0
	if itemUseRequirementsNode(use, r)[0] == 0 {
		return no // Combined use qualifications: 0x51d88c..0x51d89d.
	}
	// Flag bit 0x40 forces equipment checking (0x51d8b3). Without that
	// flag, a type with no default slot bypasses it (0x51d8ba..0x51d8c6).
	if in[71]&64 != 0 || itemDefaultEquipSlotNode([]int64{in[0], in[69], in[70]}, r)[0] != -1 {
		found := false
		// Equipment base 0xf5d, stride 12, count 12: 0x51d8ce..0x51d8df.
		for slot := uint32(0); slot < 12; slot++ {
			if uint32(in[75])+0xf5d+12*slot == uint32(in[74]) {
				// Slots 8/9 cannot activate: 0x51d8e7..0x51d8f6.
				if slot == 8 || slot == 9 {
					return no
				}
				found = true
				break
			}
		}
		if !found {
			return no
		}
	}
	// Kinds 3 and 0 cannot activate (0x51d912..0x51d926). Definition
	// 671 bypasses the nonzero spell requirement (0x51d927..0x51d93b).
	if in[15] == 3 || in[15] == 0 || (in[0] != 671 && in[16] == 0) {
		return no
	}
	// Charge mode 2 requires a nonzero instance charge byte, not quantity
	// or identification (0x51d93c..0x51d94e).
	if in[72] == 2 && in[73] == 0 {
		return no
	}
	return []int64{1} // Only AL is defined, 0x51d951.
}

func init() {
	names := append([]string(nil), Nodes["0051d610"].In...)
	names = append(names, "itemType", "gameStarted", "activationFlags", "chargeMode", "charges", "instanceAddress", "characterAddress", "itemDefinitionCount")
	Nodes["0051d800"] = Node{Addr: "0051d800", In: names, Out: []string{"activationAllowed"}, F: itemActivationAllowedNode}
}
