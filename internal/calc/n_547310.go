package calc

// offensiveActionAllowedNode is 0x547310. Inputs: kind (arg 1), spellArg (arg 2),
// spellTypeAtArg, the 77 inputs of 0x51d800 (offset 3..79), spellTypeAtItem.
// The two spell-type inputs are both entries of the one spell table (record
// stride 447, +0x137, read by 0x4ac9d0): spellTypeAtArg is the entry for spellArg,
// spellTypeAtItem the entry for the item's spell id (item field +0x63). When both
// name the same entry, the later-written input (spellTypeAtItem) is the value.
// The kind dispatch (9-entry jump table at 0x547430/0x547420):
// 0..3 allowed, 4..6 denied, 7 spell number in arg 2, 8 item instance via
// arg 3, above 8 denied (0x547317..0x54731b).
func offensiveActionAllowedNode(in []int64, r Rand) []int64 {
	if len(in) != 81 {
		panic("offensive action: expected 81 inputs")
	}
	kind, spellArg := in[0], in[1]
	itemVec, spellTypeItem := in[3:80], in[80]
	// A spell number is offensive unless it is 3 or 0x29 and its table type is
	// 3..7 (0x547358..0x547381). Numbers >= 0x96 fail an assertion
	// (0x54733c..0x547355, 0x5473ce..0x5473e7) and negatives are no table index.
	offensive := func(spell, tableType int64) int64 {
		if spell < 0 || spell >= 0x96 {
			panic("offensive action: spell number outside the asserted range")
		}
		if spell == 3 || spell == 0x29 || tableType < 3 || tableType > 7 {
			return 0
		}
		return 1
	}
	switch {
	case kind >= 0 && kind <= 3:
		return []int64{1}
	case kind == 7:
		t := in[2]
		if spellArg == itemVec[16] {
			t = spellTypeItem
		}
		return []int64{offensive(spellArg, t)}
	case kind == 8:
		// 0x5473a5 / 0x5473c3: item rejected by 0x51d800 or without a spell id.
		if itemActivationAllowedNode(itemVec, r)[0] == 0 || itemVec[16] == 0 {
			return []int64{0}
		}
		return []int64{offensive(itemVec[16], spellTypeItem)}
	}
	return []int64{0}
}

func init() {
	names := []string{
		"kind", "spellArg", "spellTypeAtArg", "itemIndex", "profession", "race", "gender",
		"professionMask", "raceMask", "genderMask", "attributeId0", "attributeMinimum0", "attributeId1",
		"attributeMinimum1", "skillId0", "skillMinimum0", "skillId1", "skillMinimum1", "itemKind",
		"spellId", "instrumentLevel", "deviceLevel", "learnedState", "spellLevel", "strength",
		"intelligence", "piety", "vitality", "dexterity", "speed", "senses", "skill0", "skill1",
		"skill2", "skill3", "skill4", "skill5", "skill6", "skill7", "skill8", "skill9", "skill10",
		"skill11", "skill12", "skill13", "skill14", "skill15", "skill16", "skill17", "skill18",
		"skill19", "skill20", "skill21", "skill22", "skill23", "skill24", "skill25", "skill26",
		"skill27", "skill28", "skill29", "skill30", "skill31", "skill32", "skill33", "skill34",
		"skill35", "skill36", "skill37", "skill38", "skill39", "skill40", "itemType", "gameStarted",
		"activationFlags", "chargeMode", "charges", "instanceAddress", "characterAddress",
		"itemDefinitionCount", "spellTypeAtItem",
	}
	Nodes["00547310"] = Node{Addr: "00547310", In: names, Out: []string{"offensiveAllowed"}, F: offensiveActionAllowedNode}
}
