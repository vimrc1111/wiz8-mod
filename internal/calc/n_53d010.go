package calc

// spellTargetAvailableNode is 0x53d010: can character slot `slot` use spell
// `spell` (self-use flag `flag`) on anything right now. Fixture wiring: the
// 0x536ad0 vocabulary (inputs 4..104 of that node follow the head here, in the
// same order), two monster records 123/124 listed in the monster list, two
// active groups (group 2 is extra) whose members are the first n monsters, and
// the reserve list holding group 1. Eight party slots carry independent
// present/HP/0xa49/0xa4d values; the slot-indexed present and HP inputs
// are written after them and win for the argument slot.
//
// Dispatch: 0x5010f0(spell, flag, 6) gives a target type; the jump table at
// 0x53d278 sends types 0,1,3,4,6,8 to "true", 2 (monsters) to a loop over the
// monster list calling 0x536ad0(slot, id, 0, 0) after the byte 0x685116 test,
// 5 (groups) to a loop over the active groups, 7 (party) to the slot scan, and
// types above 8 to "false". Group bodies use the same logic as 0x536d60 (the
// template flag byte decides first), with the attack context 6 resolved by
// 0x53bc10 exactly as in that node, so characterGroupTargetOkNode is called
// directly; monster bodies call characterMonsterTargetOkNode. Differences from
// the existing nodes: none in their formulas; this node maps the multi-group
// world onto their one-group wiring (0x5100b0 search order: active list in
// order, then the reserve list) before calling 0x536ad0.
// Panics (fixture caps, never silent): slot outside 0..7, spell above 255,
// kind outside 0..10 for the argument spell, monster count above 2, active
// list above 2 or reserve list above 1, group member count above 2 or above the
// monster list count for a group in play (4e5550 asserts), group template
// index outside 1..3, plus every cap of the two called nodes.
func spellTargetAvailableNode(in []int64, r Rand) []int64 {
	const (
		iSlot = iota
		iSpell
		iFlag
		iPartyOnly
		iArgKind
		iMonsterCount
		iG1Active
		iG1Template
		iG2Id
		iG2Active
		iG2Template
		iG2Count
		iParty               // 8 slots x (present, hp, 0xa49, 0xa4d)
		iBase  = iParty + 32 // 0x536ad0 input j (>= 4) is in[iBase+j-4]
		nIn    = iBase + 101
	)
	if len(in) != nIn {
		panic("spell target available: expected 145 inputs")
	}
	if in[iSlot] < 0 || in[iSlot] > 7 || in[iSpell] < 0 || in[iSpell] > 255 || in[iFlag] < 0 || in[iFlag] > 4294967295 ||
		in[iMonsterCount] < 0 || in[iMonsterCount] > 2 || in[iG2Count] < 0 || in[iG2Count] > 2 {
		panic("spell target available: slot, spell, flag or list count outside fixture")
	}
	o := func(j int) int64 { return in[iBase+j-4] }
	const (
		oBattle      = 5
		oPresent     = 14
		oHP          = 15
		oSpellKind   = 42 // spell {1..6,88} target kind inputs
		oQuad        = 68
		oHeading     = 69
		oMonsterBase = 73 // two records of 12 inputs: group id at +3, template +4
		oMemberCount = 67
		oTemplate    = 97
		oActive      = 102
		oReserve     = 103
		oStoredId    = 104
	)
	yes, no := []int64{1}, []int64{0}
	// 0x5010f0 -> 0x4ac9d0: kind byte at spell record +0x137; flag low byte
	// (0x4ac9f4) turns kind 1 into 0; a zero spell has no type (0x5010f5).
	typ := int64(0)
	if in[iSpell] != 0 {
		kind := in[iArgKind]
		for k, s := range []int64{1, 2, 3, 4, 5, 6, 88} {
			if in[iSpell] == s {
				kind = o(oSpellKind + k)
			}
		}
		itemType := int64(0)
		if in[iFlag]&0xff != 0 {
			itemType = 16
		}
		typ = itemSpellTargetTypeNode([]int64{1, 0, in[iSpell], itemType, kind, o(oBattle)}, r)[0]
	}
	type group struct{ id, active, template, count int64 }
	groups := [2]group{
		{o(oStoredId), in[iG1Active] & 0xff, in[iG1Template], o(oMemberCount)},
		{in[iG2Id], in[iG2Active] & 0xff, in[iG2Template], in[iG2Count]},
	}
	activeN, reserveN := o(oActive), o(oReserve)
	if activeN < 0 || activeN > 2 || reserveN < 0 || reserveN > 1 {
		panic("spell target available: group list count outside fixture")
	}
	inPlay := func(g int) {
		if groups[g].count < 0 || groups[g].count > 2 || groups[g].count > in[iMonsterCount] {
			panic("spell target available: group member not in the monster list (0x4e5550 asserts)")
		}
	}
	switch typ { // jump table 0x53d278
	case 0, 1, 3, 4, 6, 8:
		return yes
	case 2:
		if in[iPartyOnly]&0xff != 0 { // 0x53d03b
			return yes
		}
		for i := int64(0); i < in[iMonsterCount]; i++ { // 0x53d04e, ids 123+i via 0x4e5620
			v := make([]int64, 105)
			copy(v[4:], in[iBase:])
			v[0], v[1] = in[iSlot], 123+i
			target := o(oMonsterBase + 3 + 12*int(i))
			found, count := false, int64(0)
			for g := 0; g < 2 && !found; g++ { // 0x5100b0: active list first, then reserve (group 1 only)
				if (int64(g) < activeN || (g == 0 && reserveN >= 1)) && uint32(groups[g].id) == uint32(target) {
					inPlay(g)
					found, count = true, groups[g].count
				}
			}
			v[oMemberCount], v[oActive], v[oReserve], v[oStoredId] = count, 0, 0, 0
			if found {
				v[oActive], v[oStoredId] = 1, target
			}
			if characterMonsterTargetOkNode(v, r)[0] != 0 {
				return yes
			}
		}
		return no
	case 5: // 0x53d0a3
		for g := int64(0); g < activeN; g++ {
			gr := groups[g]
			if gr.active == 0 { // group +0x28
				continue
			}
			if gr.template < 1 || gr.template > 3 {
				panic("spell target available: group template index outside fixture")
			}
			inPlay(int(g))
			if o(oTemplate+int(gr.template)-1)&0xff != 0 { // template +0x24a
				continue
			}
			w := make([]int64, 82)
			w[0] = in[iSlot]
			for j := 4; j <= 67; j++ {
				w[j-2] = o(j)
			}
			w[65] = gr.count
			for k := 0; k < 2; k++ {
				for t := 0; t < 7; t++ {
					w[66+7*k+t] = o(oMonsterBase + 5 + 12*k + t)
				}
			}
			w[80], w[81] = o(oQuad), o(oHeading)
			if characterGroupTargetOkNode(w, r)[0] != 0 {
				return yes
			}
		}
		return no
	case 7: // 0x53d22b: a downed character (HP 0) with 0xa49 set and 0xa4d clear
		for k := 0; k < 8; k++ {
			present, hp, a49, a4d := in[iParty+4*k], in[iParty+4*k+1], in[iParty+4*k+2], in[iParty+4*k+3]
			if int64(k) == in[iSlot] {
				present, hp = o(oPresent), o(oHP)
			}
			if present&0xff != 0 && uint32(hp) == 0 && uint32(a49) != 0 && uint32(a4d) == 0 {
				return yes
			}
		}
	}
	return no
}

func init() {
	Nodes["0053d010"] = Node{Addr: "0053d010", In: []string{"characterSlot", "spellId", "selfUseFlag", "partyOnlyByte", "argSpellKind", "monsterCount", "group1Active", "group1Template", "group2Id", "group2Active", "group2Template", "group2MemberCount", "slot0Present", "slot0HP", "slot0A49", "slot0A4d", "slot1Present", "slot1HP", "slot1A49", "slot1A4d", "slot2Present", "slot2HP", "slot2A49", "slot2A4d", "slot3Present", "slot3HP", "slot3A49", "slot3A4d", "slot4Present", "slot4HP", "slot4A49", "slot4A4d", "slot5Present", "slot5HP", "slot5A49", "slot5A4d", "slot6Present", "slot6HP", "slot6A49", "slot6A4d", "slot7Present", "slot7HP", "slot7A49", "slot7A4d", "combatPhase", "battlePointer", "battleActionKind", "battleSpecialFlag", "battleSpell", "selectedCharacterSlot", "combatMode", "specialSpellMode", "specialItemMode", "combatState", "characterPresent", "characterHP", "characterCondition", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range", "item0Spell", "item1Spell", "item2Spell", "item3Spell", "item0Type", "item1Type", "item2Type", "item3Type", "spell1Range", "spell2Range", "spell3Range", "spell4Range", "spell5Range", "spell6Range", "spell88Range", "spell1Target", "spell2Target", "spell3Target", "spell4Target", "spell5Target", "spell6Target", "spell88Target", "mainActionKind", "offActionKind", "offSpell", "offInstance", "offItemIndexEntry", "queuedSpell", "queuedItemInstance", "queuedItemIndexEntry", "specialBaseSpell", "specialOverrideSpell", "specialBaseInstance", "specialOverrideInstance", "specialBaseItemIndex", "specialOverrideItemIndex", "partyX", "partyY", "partyZ", "partyRadius", "memberCount", "character0Quadrant", "partyHeadingDegrees", "mainSpell", "mainInstance", "mainItemIndexEntry", "monster1Active", "monster1F2b", "monster1F9f", "monster1GroupId", "monster1Template", "monster1Enabled", "monster1PhysicalAllowed", "monster1SpellAllowed", "monster1X", "monster1Y", "monster1Z", "monster1Radius", "monster2Active", "monster2F2b", "monster2F9f", "monster2GroupId", "monster2Template", "monster2Enabled", "monster2PhysicalAllowed", "monster2SpellAllowed", "monster2X", "monster2Y", "monster2Z", "monster2Radius", "template1Flag", "template2Flag", "template3Flag", "targetLockFlag", "targetLockMonster", "activeGroupCount", "reserveGroupCount", "storedGroupId"}, Out: []string{"ok"}, F: spellTargetAvailableNode}
}
