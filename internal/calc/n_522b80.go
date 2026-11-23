package calc

// itemUseRejectionNode is 0x522b80: the mirror of 0x522a30 (item_use_eligibility)
// that also reports why an item use is refused. Output 0 is the raw return byte
// (1 = refused); outputs 1..5 are what the refusal message box 0x55f260 was
// called with: call count, then its four stack arguments (message id, the
// caller's third argument, 1, 0). 0x55f260 is registered as an engine-flow
// exclusion, so the case replaces it by an address stub and the calc reproduces
// the recorded arguments and the count (all zero when nothing is shown).
//
// Flow (0x522b80..0x522d38): item types 0x17 and 0x19 go to the flag test
// 0x522d0f; otherwise 0x500010 (itemMagicUsableNode) true also goes there. The
// flag test refuses with message id [0x68c09c]+0x1e98 when any of 0x683f94,
// 0x683f9b, 0x683f9c, 0x683f9d is set, and passes (returns 0) when all are zero.
// Character field +0xa21 nonzero with item type 0xd refuses with id +0x1e9c.
// When 0x683f96 is set, 0x59e1e0 publishes the instance and 0x53d010
// (spellTargetAvailableNode, item spell, self-use flag 1 for types 0x10, 0x15,
// 0x16) is asked; zero refuses with id +0x1ea0. Then 0x5001e0 (spell usable in
// the current context): nonzero returns 0 (usable), zero refuses with id +0x1e98.
// The value it returns is the raw AL (spell 0x12 in context 4 gives the
// 0x683f9d byte), only its zero-ness matters here. The fixture wiring of the
// first 248 inputs is identical to itemUseEligibilityNode, which this node
// does not call because it must stop at each intermediate verdict.
// Panics (fixture caps, never silent): everything itemUseEligibilityNode
// panics on, plus a message id or argument outside u32. 0x5001e0 asserts for a
// spell at or above 0x96 or a context above 4 (the original stops, srAssertFail).
func itemUseRejectionNode(in []int64, r Rand) []int64 {
	const (
		iWho = iota
		iItemIndex
		iIdentified
		iA21
		iFlag9c
		iFlag9d
		iItemFields
		iCharFields = iItemFields + 12
		iContext    = iCharFields + 51
		iKind       = iContext + 4
		iLearned    = iKind + 4
		iHead       = iLearned + 32
		nBase       = iHead + 139
		iMsgArg     = nBase
		iMsgLock    = nBase + 1
		iMsgA21     = nBase + 2
		iMsgTarget  = nBase + 3
		nIn         = nBase + 4
	)
	if len(in) != nIn {
		panic("item use rejection: expected 252 inputs")
	}
	for _, v := range in[iMsgArg:] {
		if v < 0 || v > 4294967295 {
			panic("item use rejection: message id or argument out of range")
		}
	}
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
	refuse := func(id int64) []int64 { return []int64{1, 1, id, in[iMsgArg], 1, 0} }
	pass := []int64{0, 0, 0, 0, 0, 0}
	who, idx := in[iWho], in[iItemIndex]
	if who < 0 || who > 7 || idx < 0 || idx > 3 {
		panic("item use rejection: slot or item record outside fixture")
	}
	typ, spell := in[at(71)+int(idx)], in[at(67)+int(idx)]
	for _, v := range []int64{typ, spell, in[iIdentified], in[iFlag9c], in[iFlag9d]} {
		if v < 0 || v > 255 {
			panic("item use rejection: byte field out of range")
		}
	}
	if in[iA21] < 0 || in[iA21] > 4294967295 {
		panic("item use rejection: field +0xa21 out of range")
	}
	// 0x522d0f: refused while any of the four flags is set.
	flagTest := func() []int64 {
		if in[at(50)] != 0 || in[at(140)] != 0 || in[iFlag9c] != 0 || in[iFlag9d] != 0 {
			return refuse(in[iMsgLock])
		}
		return pass
	}
	if typ == 0x17 || typ == 0x19 {
		return flagTest()
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
		return flagTest()
	}
	if in[iA21] != 0 && typ == 0xd {
		return refuse(in[iMsgA21])
	}
	if in[at(52)] != 0 { // 0x683f96
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
			return refuse(in[iMsgTarget])
		}
	}
	if spell >= 0x96 {
		panic("item use rejection: spell outside the spell table (0x5001e0 asserts)")
	}
	ctx := in[iContext+last(spell)]
	if ctx > 4 {
		panic("item use rejection: spell context above 4 (0x5001e0 asserts)")
	}
	combat, lock, cm, cx := in[at(50)], in[at(140)], in[iFlag9c], in[iFlag9d]
	usable := false
	if in[at(44)] == 6 && combat == 0 && lock != 0 && (spell == 0x17 || spell == 0x3a) {
		usable = true
	} else {
		anyFlag := cm != 0 || cx != 0
		switch ctx {
		case 0:
			usable = lock == 0 && !anyFlag
		case 1:
			usable = combat == 1 && lock == 0 && !anyFlag
		case 2:
			usable = combat == 0 && lock == 0 && !anyFlag
		case 3:
			usable = lock != 0 && !anyFlag
		case 4:
			if spell == 0x27 {
				usable = anyFlag
			} else if spell == 0x12 {
				usable = cx != 0
			}
		}
	}
	if usable {
		return pass
	}
	return refuse(in[iMsgLock])
}

func init() {
	Nodes["00522b80"] = Node{Addr: "00522b80", In: []string{"characterSlot", "itemIndex", "identified", "characterA21", "flag683f9c", "flag683f9d", "professionMask", "raceMask", "genderMask", "attributeId0", "attributeMinimum0", "attributeId1", "attributeMinimum1", "skillId0", "skillMinimum0", "skillId1", "skillMinimum1", "itemKind", "profession", "race", "gender", "strength", "intelligence", "piety", "vitality", "dexterity", "speed", "senses", "skill0", "skill1", "skill2", "skill3", "skill4", "skill5", "skill6", "skill7", "skill8", "skill9", "skill10", "skill11", "skill12", "skill13", "skill14", "skill15", "skill16", "skill17", "skill18", "skill19", "skill20", "skill21", "skill22", "skill23", "skill24", "skill25", "skill26", "skill27", "skill28", "skill29", "skill30", "skill31", "skill32", "skill33", "skill34", "skill35", "skill36", "skill37", "skill38", "skill39", "skill40", "spellContextItem0", "spellContextItem1", "spellContextItem2", "spellContextItem3", "spellKindItem0", "spellKindItem1", "spellKindItem2", "spellKindItem3", "learnedSlot0Item0", "learnedSlot0Item1", "learnedSlot0Item2", "learnedSlot0Item3", "learnedSlot1Item0", "learnedSlot1Item1", "learnedSlot1Item2", "learnedSlot1Item3", "learnedSlot2Item0", "learnedSlot2Item1", "learnedSlot2Item2", "learnedSlot2Item3", "learnedSlot3Item0", "learnedSlot3Item1", "learnedSlot3Item2", "learnedSlot3Item3", "learnedSlot4Item0", "learnedSlot4Item1", "learnedSlot4Item2", "learnedSlot4Item3", "learnedSlot5Item0", "learnedSlot5Item1", "learnedSlot5Item2", "learnedSlot5Item3", "learnedSlot6Item0", "learnedSlot6Item1", "learnedSlot6Item2", "learnedSlot6Item3", "learnedSlot7Item0", "learnedSlot7Item1", "learnedSlot7Item2", "learnedSlot7Item3", "partyOnlyByte", "monsterCount", "group1Active", "group1Template", "group2Id", "group2Active", "group2Template", "group2MemberCount", "slot0Present", "slot0HP", "slot0A49", "slot0A4d", "slot1Present", "slot1HP", "slot1A49", "slot1A4d", "slot2Present", "slot2HP", "slot2A49", "slot2A4d", "slot3Present", "slot3HP", "slot3A49", "slot3A4d", "slot4Present", "slot4HP", "slot4A49", "slot4A4d", "slot5Present", "slot5HP", "slot5A49", "slot5A4d", "slot6Present", "slot6HP", "slot6A49", "slot6A4d", "slot7Present", "slot7HP", "slot7A49", "slot7A4d", "combatPhase", "battlePointer", "battleActionKind", "battleSpecialFlag", "battleSpell", "selectedCharacterSlot", "combatMode", "specialSpellMode", "specialItemMode", "combatState", "characterPresent", "characterHP", "characterCondition", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range", "item0Spell", "item1Spell", "item2Spell", "item3Spell", "item0Type", "item1Type", "item2Type", "item3Type", "spell1Range", "spell2Range", "spell3Range", "spell4Range", "spell5Range", "spell6Range", "spell88Range", "spell1Target", "spell2Target", "spell3Target", "spell4Target", "spell5Target", "spell6Target", "spell88Target", "mainActionKind", "offActionKind", "offSpell", "offInstance", "offItemIndexEntry", "queuedSpell", "queuedItemInstance", "queuedItemIndexEntry", "specialBaseSpell", "specialOverrideSpell", "specialBaseInstance", "specialBaseItemIndex", "partyX", "partyY", "partyZ", "partyRadius", "memberCount", "character0Quadrant", "partyHeadingDegrees", "mainSpell", "mainInstance", "mainItemIndexEntry", "monster1Active", "monster1F2b", "monster1F9f", "monster1GroupId", "monster1Template", "monster1Enabled", "monster1PhysicalAllowed", "monster1SpellAllowed", "monster1X", "monster1Y", "monster1Z", "monster1Radius", "monster2Active", "monster2F2b", "monster2F9f", "monster2GroupId", "monster2Template", "monster2Enabled", "monster2PhysicalAllowed", "monster2SpellAllowed", "monster2X", "monster2Y", "monster2Z", "monster2Radius", "template1Flag", "template2Flag", "template3Flag", "targetLockFlag", "targetLockMonster", "activeGroupCount", "reserveGroupCount", "storedGroupId", "messageArg", "msgIdLock", "msgIdA21", "msgIdTarget"}, Out: []string{"rejected", "msgCalls", "msgId", "msgArg", "msgMode", "msgFlag"}, F: itemUseRejectionNode}
}
