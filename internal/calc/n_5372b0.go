package calc

// Fixture record selectors: four source slots and item records, two character
// records, monster ID 123 and group ID 456. All selected target records share
// the explicit target fields. Pointers select records; they are not rule data.
// Strides 262, 269, 447 and 6242: exe 0x53b7f0, 0x5372d2,
// 0x4ac9d0 and 0x5361e8. The result is the boolean low byte.
func itemTargetCompatibleNode(in []int64, r Rand) []int64 {
	if len(in) != 31 {
		panic("item target compatible: expected 31 inputs")
	}
	for i, v := range in {
		lo, hi := int64(0), int64(4294967295)
		switch i {
		case 0:
			hi = 3
		case 2:
			lo, hi = -1, 3
		case 3, 4, 6, 10, 12, 13, 14, 21, 28, 30:
			hi = 255
		case 5:
			hi = 10 // Legal spell target kinds, exe 0x50110b.
		case 9, 11, 17, 18, 20:
			lo, hi = -2147483648, 2147483647
		case 16:
			lo, hi = -1, 1
		}
		if v < lo || v > hi {
			panic("item target compatible: field outside storage or fixture bounds")
		}
	}
	// Current target source selection, exe 0x53b811..0x53b88c.
	// UI state 7 with a pending action selects spell/item targets (7/8);
	// other active actions select no record. All non-null records are inputs.
	nullTarget := in[7] == 7 && in[8] != 0 && in[9] != -1 &&
		!((in[9] == 7 || in[9] == 8) && in[10] != 0)
	category := itemSpellTargetTypeNode([]int64{in[1], in[2], in[3], in[4], in[5], in[8]}, r)[0]
	// Exe 0x537309..0x537319: category 2 and a nonzero interface flag
	// bypass both the target shape and record qualification.
	if category == 2 && in[6] != 0 {
		return []int64{1}
	}
	if nullTarget {
		return []int64{0} // Exe 0x537166: null current target fails even category 0.
	}
	if category < 1 || category > 8 {
		return []int64{1} // Exe 0x53716f: category 0 accepts any non-null record.
	}
	if queuedSpellRuleErr != nil {
		panic(queuedSpellRuleErr)
	}
	kind := in[15]
	compatible := false
	// Reuse the runtime-loaded 0x5371f8 dispatch, without copying exe tables.
	switch queuedSpellRuleData.compatibility[category-1] {
	case 0x53717c:
		compatible = (kind == 1 && in[16] != -1) || (kind == 3 && in[17] != -1)
	case 0x5371a4:
		compatible = kind == 9 && in[16] != -1
	case 0x5371b9:
		compatible = kind == 2 || (kind == 4 && in[18] != -1)
	case 0x5371cf:
		compatible = kind == 7 && in[19] != 0
	case 0x5371e5:
		compatible = kind == 6
	case 0x5371f4:
		return []int64{1}
	default:
		panic("item target compatible: unknown compatibility destination")
	}
	if compatible && itemTargetRecordValid5372b0(in) {
		return []int64{1}
	}
	return []int64{0}
}

// Record qualification at exe 0x536190..0x5363d1. Character HP is
// C+0xb11 here (not the C+0xb0d read by characterTargetableNode).
func itemTargetRecordValid5372b0(in []int64) bool {
	switch in[15] {
	case 1:
		return in[21] != 0 && in[22] != 0 && in[23] < 18 // Exe 0x5361e2..0x536211.
	case 9:
		return in[21] != 0 && in[22] == 0 && in[24] != 0 && in[25] == 0 // Exe 0x53624c..0x536286.
	case 3:
		// IDs 123/456 belong to the case's single-entry lists, not exe constants.
		return in[17] == 123 && in[26] != 0 && in[27] == 0 && in[28] != 0 // Exe 0x5362b0..0x536314.
	case 4:
		return in[18] == 456 && in[29] != 0 && in[30] != 0 // Exe 0x536341..0x536398.
	case 7:
		return in[20] != -1 // Exe 0x5363c3: item target's record index.
	default:
		return true // Exe 0x5361a5: other target kinds require no qualification.
	}
}

func init() {
	Nodes["005372b0"] = Node{
		Addr: "005372b0",
		In:   []string{"sourceIndex", "itemInstance", "itemIndex", "spellID", "itemType", "spellTarget", "interfaceExemption", "uiState", "battle", "battleAction", "battleActionActive", "selectedSource", "combatMode", "specialMode1", "specialMode2", "targetKind", "characterTarget", "monsterTarget", "groupTarget", "targetItemPointer", "targetItemIndex", "characterPresent", "characterHP", "characterCondition", "characterRecoverable", "characterRecoveryBlocked", "monsterHP", "monsterRemoved", "monsterPresent", "groupSize", "groupActive"},
		Out:  []string{"compatible"}, F: itemTargetCompatibleNode,
	}
}
