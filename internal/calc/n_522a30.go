package calc

// itemUseEligibilityNode is 0x522a30: can character slot `who` use the item
// instance (its definition index is instance +0, identified byte +6) now.
//
// Flow: item types 0x17 and 0x19 skip straight to the flag test (0x522b4a).
// Otherwise 0x500010 (scroll qualification, itemMagicUsableNode) true also goes
// to the flag test; a character field +0xa21 nonzero with item type 0xd
// refuses; 0x59e1e0 publishes the instance as the 0x69b9a4 override, 0x53d010
// (spellTargetAvailableNode) is asked with the item spell and a self-use flag
// that is 1 for types 0x10, 0x15 and 0x16; its zero refuses, else 0x5001e0
// (spell usable in the current context) decides. The flag test passes (1) only
// when 0x683f94, 0x683f9b, 0x683f9c and 0x683f9d are all zero.
//
// Fixture wiring: four item records, eight character slots. Fields indexed by
// the instance's definition index (the 0x500010 item fields) or by the slot
// are single inputs for the selected record; the learned state cell
// (char +0xbcd + 4*spell) and the spell record kind (+0x137) and context
// (+0x13b) are indexed by an item spell, so there is one input per item (and
// slot for the learned state) and the last input written to a cell wins. The
// learned cells of spells 246 and 249 overlap the 0x53d010 fields of the
// main/off hand item index of the same character, written later, so those win.
// Panics (fixture caps, never silent): slot outside 0..7, item index outside
// 0..3, spell 0x96 or above or context above 4 when 0x5001e0 is reached (the
// original asserts), plus every cap of 0x500010 and 0x53d010. The result is
// the raw AL: 0x5001e0 for spell 0x12 in context 4 returns the 0x683f9d byte.
func itemUseEligibilityNode(in []int64, r Rand) []int64 {
	const (
		iWho = iota
		iItemIndex
		iIdentified
		iA21
		iFlag9c
		iFlag9d
		iItemFields                    // 11 requirement inputs, then the item kind
		iCharFields = iItemFields + 12 // profession, race, gender, 7 attributes, 41 skills
		iContext    = iCharFields + 51
		iKind       = iContext + 4
		iLearned    = iKind + 4 // slot-major, 4 items per slot
		iHead       = iLearned + 32
		nIn         = iHead + 139
	)
	if len(in) != nIn {
		panic("item use eligibility: expected 248 inputs")
	}
	// Position of 0x53d010 input i (not 0, 1, 2, 4, 100, 102) in this node.
	at := func(i int) int {
		switch {
		case i == 3:
			return iHead
		case i <= 99:
			return iHead + i - 4
		case i == 101:
			return iHead + 96
		}
		return iHead + i - 6
	}
	who, idx := in[iWho], in[iItemIndex]
	if who < 0 || who > 7 || idx < 0 || idx > 3 {
		panic("item use eligibility: slot or item record outside fixture")
	}
	typ, spell := in[at(71)+int(idx)], in[at(67)+int(idx)]
	for _, v := range []int64{typ, spell, in[iIdentified], in[iFlag9c], in[iFlag9d]} {
		if v < 0 || v > 255 {
			panic("item use eligibility: byte field out of range")
		}
	}
	if in[iA21] < 0 || in[iA21] > 4294967295 {
		panic("item use eligibility: field +0xa21 out of range")
	}
	// 0x522b4a: usable only while none of the four flags is set.
	flagsClear := func() []int64 {
		if in[at(50)] != 0 || in[at(140)] != 0 || in[iFlag9c] != 0 || in[iFlag9d] != 0 {
			return []int64{0}
		}
		return []int64{1}
	}
	if typ == 0x17 || typ == 0x19 {
		return flagsClear()
	}
	// The input written last to a cell wins: the last item whose spell equals sp.
	last := func(sp int64) int {
		j := 0
		for k := 0; k < 4; k++ {
			if in[at(67)+k] == sp {
				j = k
			}
		}
		return j
	}
	learned := in[iLearned+4*int(who)+last(spell)]
	if spell == 246 {
		learned = in[at(61)]
	} else if spell == 249 {
		learned = in[at(62)]
	}
	m := make([]int64, 67)
	m[0] = idx
	copy(m[1:4], in[iCharFields:iCharFields+3])
	copy(m[4:15], in[iItemFields:iItemFields+11])
	m[15], m[16], m[17] = in[iItemFields+11], spell, learned
	copy(m[18:66], in[iCharFields+3:iCharFields+51])
	m[66] = in[iIdentified]
	if itemMagicUsableNode(m, r)[0] != 0 {
		return flagsClear()
	}
	if in[iA21] != 0 && typ == 0xd {
		return []int64{0}
	}
	flag := int64(0)
	if typ == 0x10 || typ == 0x15 || typ == 0x16 {
		flag = 1
	}
	v := make([]int64, 145)
	v[0], v[1], v[2], v[3] = who, spell, flag, in[at(3)]
	v[4] = in[iKind+last(spell)]
	copy(v[5:100], in[at(5):at(5)+95])
	v[100] = 1 // the published instance pointer; 0x536ad0 only tests it for zero
	v[101], v[102] = in[at(101)], idx
	copy(v[103:], in[at(103):at(103)+42])
	if spellTargetAvailableNode(v, r)[0] == 0 {
		return []int64{0}
	}
	// 0x5001e0 (spell, 0): a spell at or above 0x96 asserts, as does a context above 4.
	if spell >= 0x96 {
		panic("item use eligibility: spell outside the spell table (0x5001e0 asserts)")
	}
	ctx := in[iContext+last(spell)]
	if ctx > 4 {
		panic("item use eligibility: spell context above 4 (0x5001e0 asserts)")
	}
	combat, lock, cm, cx := in[at(50)], in[at(140)], in[iFlag9c], in[iFlag9d]
	if in[at(44)] == 6 && combat == 0 && lock != 0 && (spell == 0x17 || spell == 0x3a) {
		return []int64{1}
	}
	anyFlag := cm != 0 || cx != 0
	ok := false
	switch ctx {
	case 0:
		ok = lock == 0 && !anyFlag
	case 1:
		ok = combat == 1 && lock == 0 && !anyFlag
	case 2:
		ok = combat == 0 && lock == 0 && !anyFlag
	case 3:
		ok = lock != 0 && !anyFlag
	case 4:
		if spell == 0x27 {
			ok = anyFlag
		} else if spell == 0x12 {
			return []int64{cx} // the byte itself, not a boolean
		}
	}
	if ok {
		return []int64{1}
	}
	return []int64{0}
}

func init() {
	Nodes["00522a30"] = Node{Addr: "00522a30", In: []string{"characterSlot", "itemIndex", "identified", "characterA21", "flag683f9c", "flag683f9d", "professionMask", "raceMask", "genderMask", "attributeId0", "attributeMinimum0", "attributeId1", "attributeMinimum1", "skillId0", "skillMinimum0", "skillId1", "skillMinimum1", "itemKind", "profession", "race", "gender", "strength", "intelligence", "piety", "vitality", "dexterity", "speed", "senses", "skill0", "skill1", "skill2", "skill3", "skill4", "skill5", "skill6", "skill7", "skill8", "skill9", "skill10", "skill11", "skill12", "skill13", "skill14", "skill15", "skill16", "skill17", "skill18", "skill19", "skill20", "skill21", "skill22", "skill23", "skill24", "skill25", "skill26", "skill27", "skill28", "skill29", "skill30", "skill31", "skill32", "skill33", "skill34", "skill35", "skill36", "skill37", "skill38", "skill39", "skill40", "spellContextItem0", "spellContextItem1", "spellContextItem2", "spellContextItem3", "spellKindItem0", "spellKindItem1", "spellKindItem2", "spellKindItem3", "learnedSlot0Item0", "learnedSlot0Item1", "learnedSlot0Item2", "learnedSlot0Item3", "learnedSlot1Item0", "learnedSlot1Item1", "learnedSlot1Item2", "learnedSlot1Item3", "learnedSlot2Item0", "learnedSlot2Item1", "learnedSlot2Item2", "learnedSlot2Item3", "learnedSlot3Item0", "learnedSlot3Item1", "learnedSlot3Item2", "learnedSlot3Item3", "learnedSlot4Item0", "learnedSlot4Item1", "learnedSlot4Item2", "learnedSlot4Item3", "learnedSlot5Item0", "learnedSlot5Item1", "learnedSlot5Item2", "learnedSlot5Item3", "learnedSlot6Item0", "learnedSlot6Item1", "learnedSlot6Item2", "learnedSlot6Item3", "learnedSlot7Item0", "learnedSlot7Item1", "learnedSlot7Item2", "learnedSlot7Item3", "partyOnlyByte", "monsterCount", "group1Active", "group1Template", "group2Id", "group2Active", "group2Template", "group2MemberCount", "slot0Present", "slot0HP", "slot0A49", "slot0A4d", "slot1Present", "slot1HP", "slot1A49", "slot1A4d", "slot2Present", "slot2HP", "slot2A49", "slot2A4d", "slot3Present", "slot3HP", "slot3A49", "slot3A4d", "slot4Present", "slot4HP", "slot4A49", "slot4A4d", "slot5Present", "slot5HP", "slot5A49", "slot5A4d", "slot6Present", "slot6HP", "slot6A49", "slot6A4d", "slot7Present", "slot7HP", "slot7A49", "slot7A4d", "combatPhase", "battlePointer", "battleActionKind", "battleSpecialFlag", "battleSpell", "selectedCharacterSlot", "combatMode", "specialSpellMode", "specialItemMode", "combatState", "characterPresent", "characterHP", "characterCondition", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range", "item0Spell", "item1Spell", "item2Spell", "item3Spell", "item0Type", "item1Type", "item2Type", "item3Type", "spell1Range", "spell2Range", "spell3Range", "spell4Range", "spell5Range", "spell6Range", "spell88Range", "spell1Target", "spell2Target", "spell3Target", "spell4Target", "spell5Target", "spell6Target", "spell88Target", "mainActionKind", "offActionKind", "offSpell", "offInstance", "offItemIndexEntry", "queuedSpell", "queuedItemInstance", "queuedItemIndexEntry", "specialBaseSpell", "specialOverrideSpell", "specialBaseInstance", "specialBaseItemIndex", "partyX", "partyY", "partyZ", "partyRadius", "memberCount", "character0Quadrant", "partyHeadingDegrees", "mainSpell", "mainInstance", "mainItemIndexEntry", "monster1Active", "monster1F2b", "monster1F9f", "monster1GroupId", "monster1Template", "monster1Enabled", "monster1PhysicalAllowed", "monster1SpellAllowed", "monster1X", "monster1Y", "monster1Z", "monster1Radius", "monster2Active", "monster2F2b", "monster2F9f", "monster2GroupId", "monster2Template", "monster2Enabled", "monster2PhysicalAllowed", "monster2SpellAllowed", "monster2X", "monster2Y", "monster2Z", "monster2Radius", "template1Flag", "template2Flag", "template3Flag", "targetLockFlag", "targetLockMonster", "activeGroupCount", "reserveGroupCount", "storedGroupId"}, Out: []string{"usable"}, F: itemUseEligibilityNode}
}
