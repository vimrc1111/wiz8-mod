package calc

import "strconv"

// actionTargetAvailableNode is 0x53cdf0: does character `slot` have any legal
// target for the action it has queued. The action source is chosen as in
// 0x536ad0 (attack context 6 resolved by 0x53bc10: main action, off-hand
// action, special-mode record, queued record or battle record), and an
// attacker that is absent, at 0 HP or in condition >= 18 has its action
// replaced by -1 unless the action is 10 or 11 (0x53c270). The action kind then
// picks the check (unsigned compare against 8, jump table 0x53cfe4):
//
//	0, 1  true when 0x536ad0(slot, id, 0, 0) accepts any listed monster
//	5     0x545c20 for every other party slot (type 1), then for every listed
//	      monster (type 3); true when one passes
//	7     true when the battle record byte +0x2f8 (0x68edcc) is zero, else
//	      0x53d010(slot, spell, 0)
//	8     same byte test; then false for a null entry pointer, else
//	      0x53d010(slot, item spell, flag) with flag nonzero for item types
//	      0x10, 0x15, 0x16 (0x5207c0)
//	else  true (including -1)
//
// Fixture wiring is the 0x53d010 world (spellTargetAvailableNode) plus the
// monster visible byte (+21), the condition word of every slot and the range
// selector byte of the combat record (0x6836a8, +76, stride 212) that the
// 0x545c20 branch reads. The quadrant bytes of slots 1..7, the formation
// unavailable bytes and the quadrant slot table keep their idle values
// (0, 0, 255) in the case and here, so only character 0's quadrant varies the
// formation range check.
// Differences from the existing nodes: interceptTargetAvailableNode (0x545c20)
// pins the hand item to item 0 through its own list pointers; here the hand
// item is the one the hand's entry pointer and index select (a null pointer
// gives no item spell). The 0x53d010 argument spell and flag come from the
// action source; spells outside {1..6, 88} have target kind 0 (an unwritten
// spell record).
// Panics (fixture caps, never silent): action 7 or 8 reached with a null battle
// pointer (the original reads +0x2f8 of it), an item entry index outside 0..3
// on an item action that reaches the item record, a monster template index
// outside 1..3, a spell above 255, plus every cap of the nodes called. A
// non-zero battle pointer means "the battle record".
func actionTargetAvailableNode(in []int64, r Rand) []int64 {
	const (
		iSlot = iota
		iPartyOnly
		iMonsterCount
		iG1Active
		iG1Template
		iG2Id
		iG2Active
		iG2Template
		iG2Count
		iParty                 // 8 slots x (present, hp, 0xa49, 0xa4d)
		iStatus  = iParty + 32 // condition word of each slot
		iSelect  = iStatus + 8 // combat record byte +76 of the attacker
		iVisible = iSelect + 1 // monster +21 of the two monsters
		iBase    = iVisible + 2
		nIn      = iBase + 101
	)
	if len(in) != nIn {
		panic("action target available: expected 153 inputs")
	}
	if in[iSlot] < 0 || in[iSlot] > 7 || in[iMonsterCount] < 0 || in[iMonsterCount] > 2 {
		panic("action target available: slot or monster count outside fixture")
	}
	slot := int(in[iSlot])
	at := map[string]int{} // 0x536ad0 input name -> index here
	for j, n := range Nodes["00536ad0"].In {
		if j >= 4 {
			at[n] = iBase + j - 4
		}
	}
	o := func(name string) int64 {
		k, ok := at[name]
		if !ok {
			panic("action target available: unknown input " + name)
		}
		return in[k]
	}
	yes, no := []int64{1}, []int64{0}

	// Attack context, as the inline copy of 0x53bc10 in 0x536ad0 (never 5, 6, 8).
	c := int64(0)
	switch {
	case o("combatPhase") == 7 && o("battlePointer") != 0 && o("battleActionKind") != 4294967295:
		c = 7
		if o("battleSpecialFlag") != 0 && o("battleActionKind") == 7 {
			c = 3
		}
		if o("battleSpecialFlag") != 0 && o("battleActionKind") == 8 {
			c = 4
		}
	case in[iSlot] == o("selectedCharacterSlot") && (o("specialSpellMode") != 0 || o("specialItemMode") != 0):
		c = 2
	case o("combatMode") != 0:
		c = 1
	}
	// Action source of 0x4e77b0.
	act, spell, inst, idx := int64(0), int64(0), int64(0), int64(0)
	switch c {
	case 0:
		act, spell, inst, idx = o("mainActionKind"), o("mainSpell"), o("mainInstance"), o("mainItemIndexEntry")
	case 1:
		act, spell, inst, idx = o("offActionKind"), o("offSpell"), o("offInstance"), o("offItemIndexEntry")
	case 2:
		if o("specialSpellMode") != 0 { // 0x5a1350: override spell, else base spell.
			act, spell = 7, o("specialOverrideSpell")
			if spell == 0 {
				spell = o("specialBaseSpell")
			}
		} else { // 0x59e0d0: override entry when non-null, else base entry.
			act = 8
			inst, idx = o("specialBaseInstance"), o("specialBaseItemIndex")
			if o("specialOverrideInstance") != 0 {
				inst, idx = o("specialOverrideInstance"), o("specialOverrideItemIndex")
			}
		}
	case 3:
		act, spell = 7, o("queuedSpell")
	case 4:
		act, inst, idx = 8, o("queuedItemInstance"), o("queuedItemIndexEntry")
	case 7:
		act, spell = o("battleActionKind"), o("battleSpell")
	}
	if !(o("characterPresent") != 0 && o("characterHP") != 0 && uint32(o("characterCondition")) < 18) && act != 10 && act != 11 { // 0x53c270
		act = 4294967295
	}
	// The byte tested by actions 7 and 8 (battle record +0x2f8).
	battleByte := func() bool {
		if o("battlePointer") == 0 {
			panic("action target available: null battle pointer read for action 7 or 8")
		}
		return o("battleSpecialFlag")&0xff != 0
	}
	spellRange := func(sp int64) int64 { // spell record +303; unwritten records read 0
		switch {
		case sp >= 1 && sp <= 6:
			return o("spell" + strconv.FormatInt(sp, 10) + "Range")
		case sp == 88:
			return o("spell88Range")
		}
		return 0
	}
	monster := func(i int64) func(string) int64 { // record i of the monster list
		return func(f string) int64 { return o("monster" + strconv.FormatInt(i+1, 10) + f) }
	}

	switch uint32(act) { // jump table 0x53cfe4
	case 0, 1:
		// 0x536ad0 over the monster list, with the group wiring of 0x53d010.
		type group struct{ id, active, template, count int64 }
		groups := [2]group{
			{o("storedGroupId"), in[iG1Active] & 0xff, in[iG1Template], o("memberCount")},
			{in[iG2Id], in[iG2Active] & 0xff, in[iG2Template], in[iG2Count]},
		}
		activeN, reserveN := o("activeGroupCount"), o("reserveGroupCount")
		if activeN < 0 || activeN > 2 || reserveN < 0 || reserveN > 1 || in[iG2Count] < 0 || in[iG2Count] > 2 {
			panic("action target available: group list count outside fixture")
		}
		for i := int64(0); i < in[iMonsterCount]; i++ { // 0x4e5620 ids 123+i
			v := make([]int64, 105)
			copy(v[4:], in[iBase:])
			v[0], v[1] = in[iSlot], 123+i
			target := monster(i)("GroupId")
			found, count := false, int64(0)
			for g := 0; g < 2 && !found; g++ { // 0x5100b0: active list first, then reserve (group 1 only)
				if (int64(g) < activeN || (g == 0 && reserveN >= 1)) && uint32(groups[g].id) == uint32(target) {
					if groups[g].count < 0 || groups[g].count > 2 || groups[g].count > in[iMonsterCount] {
						panic("action target available: group member not in the monster list (0x4e5550 asserts)")
					}
					found, count = true, groups[g].count
				}
			}
			v[at["memberCount"]-iBase+4], v[at["activeGroupCount"]-iBase+4], v[at["reserveGroupCount"]-iBase+4], v[at["storedGroupId"]-iBase+4] = count, 0, 0, 0
			if found {
				v[at["activeGroupCount"]-iBase+4], v[at["storedGroupId"]-iBase+4] = 1, target
			}
			if characterMonsterTargetOkNode(v, r)[0] != 0 {
				return yes
			}
		}
		return no
	case 7, 8:
		if !battleByte() {
			return yes
		}
		flag := int64(0)
		if act == 8 {
			if inst == 0 {
				return no
			}
			if idx < 0 || idx > 3 {
				panic("action target available: item entry index outside fixture")
			}
			switch o("item"+strconv.FormatInt(idx, 10)+"Type") & 0xff { // 0x5207c0
			case 0x10, 0x15, 0x16:
				flag = 1
			}
			spell = o("item"+strconv.FormatInt(idx, 10)+"Spell") & 0xff // item +0x63
		}
		if spell < 0 || spell > 255 {
			panic("action target available: spell above 255")
		}
		v := make([]int64, 145)
		v[0], v[1], v[2], v[3] = in[iSlot], spell, flag, in[iPartyOnly]
		copy(v[5:12], in[iMonsterCount:iG2Count+1])
		copy(v[12:44], in[iParty:iStatus])
		copy(v[44:], in[iBase:])
		return spellTargetAvailableNode(v, r)
	case 5:
		want := Nodes["00545c20"].In
		// The hand 0x545c20 reads: selector 0 gives the off hand, else the main hand.
		handKind, handInst, handIdx := o("mainActionKind"), o("mainInstance"), o("mainItemIndexEntry")
		if in[iSelect]&0xff == 0 {
			handKind, handInst, handIdx = o("offActionKind"), o("offInstance"), o("offItemIndexEntry")
		}
		item0Spell := o("item0Spell")
		if handKind == 8 { // the item the hand's entry selects
			item0Spell = 0
			if handInst != 0 {
				if handIdx < 0 || handIdx > 3 {
					panic("action target available: item entry index outside fixture")
				}
				item0Spell = o("item" + strconv.FormatInt(handIdx, 10) + "Spell")
			}
		}
		call := func(typ, target int64, mi int64) bool {
			f := map[string]int64{
				"targetType": typ, "targetCharacter": target, "monsterID": mi,
				"interceptRangeSelector": in[iSelect], "characterSlot": in[iSlot],
				"combatPhase": o("combatState"), "combatMode": o("combatMode"),
				"specialSpellMode": o("specialSpellMode"), "specialItemMode": o("specialItemMode"),
				"characterPresent": o("characterPresent"), "characterHP": o("characterHP"), "characterCondition": o("characterCondition"),
				"mainAttackKind": o("mainAttackKind"), "mainAttackAvailable": o("mainAttackAvailable"),
				"offAttackKind": o("offAttackKind"), "offAttackAvailable": o("offAttackAvailable"),
				"characterInParty": 1, "mainItemIndex": o("mainItemIndex"), "offItemIndex": o("offItemIndex"),
				"spellId": o("queuedSpell"), "spellRange": 0,
				"mainActionKind": o("mainActionKind"), "offActionKind": o("offActionKind"), "battleActionKind": o("battleActionKind"),
				"partyX": o("partyX"), "partyY": o("partyY"), "partyZ": o("partyZ"), "partyRadius": o("partyRadius"),
				"character0Quadrant": o("character0Quadrant"), "partyHeadingDegrees": o("partyHeadingDegrees"),
				"mainActionSpell": o("mainSpell"), "mainActionSpellRange": spellRange(o("mainSpell")),
				"offActionSpell": o("offSpell"), "offActionSpellRange": spellRange(o("offSpell")),
			}
			for k := 0; k < 8; k++ {
				ks := strconv.Itoa(k)
				present, hp, status := in[iParty+4*k], in[iParty+4*k+1], in[iStatus+k]
				if k == slot {
					present, hp, status = o("characterPresent"), o("characterHP"), o("characterCondition")
				}
				// 0x53c29d reads the HP word as signed.
				f["slotActive"+ks], f["currentHP"+ks], f["mainStatus"+ks] = present, int64(int32(uint32(hp))), status
				f["character"+ks+"FormationUnavailable"] = 0
				if k > 0 {
					f["character"+ks+"Quadrant"] = 0
				}
			}
			for q := 0; q < 5; q++ {
				for s := 0; s < 3; s++ {
					f["quadrant"+strconv.Itoa(q)+"Slot"+strconv.Itoa(s)] = 255
				}
			}
			for n := int64(0); n < 4; n++ {
				ns := strconv.FormatInt(n, 10)
				f["item"+ns+"Range"] = o("item" + ns + "Range")
				sp := o("item" + ns + "Spell")
				if n == 0 {
					sp = item0Spell
				}
				f["item"+ns+"Spell"], f["item"+ns+"SpellRange"] = sp, spellRange(sp)
			}
			if typ == 3 {
				m := monster(mi)
				tpl := m("Template")
				if tpl < 1 || tpl > 3 {
					panic("action target available: monster template index outside fixture")
				}
				f["monsterActive"], f["monsterHP"], f["monsterVisible"] = m("Active"), m("F2b"), in[iVisible+int(mi)]
				f["monsterSpecialTemplate"] = o("template"+strconv.FormatInt(tpl, 10)+"Flag") & 0xff
				f["monsterTargetEnabled"], f["monsterPhysicalAllowed"], f["monsterSpellAllowed"] = m("Enabled"), m("PhysicalAllowed"), m("SpellAllowed")
				f["monsterX"], f["monsterY"], f["monsterZ"], f["monsterRadius"] = m("X"), m("Y"), m("Z"), m("Radius")
			} else {
				for _, n := range []string{"monsterActive", "monsterHP", "monsterVisible", "monsterSpecialTemplate", "monsterTargetEnabled", "monsterPhysicalAllowed", "monsterSpellAllowed", "monsterX", "monsterY", "monsterZ", "monsterRadius"} {
					f[n] = 0
				}
			}
			v := make([]int64, len(want))
			for i, n := range want {
				x, ok := f[n]
				if !ok {
					panic("action target available: no value for 0x545c20 input " + n)
				}
				v[i] = x
			}
			return interceptTargetAvailableNode(v, r)[0] != 0
		}
		for t := 0; t < 8; t++ { // 0x53cf23: every other party slot
			if t != slot && call(1, int64(t), 0) {
				return yes
			}
		}
		for i := int64(0); i < in[iMonsterCount]; i++ { // 0x53cf75: every listed monster
			if call(3, 0, i) {
				return yes
			}
		}
		return no
	}
	return yes
}

func init() {
	Nodes["0053cdf0"] = Node{Addr: "0053cdf0", In: []string{"characterSlot", "partyOnlyByte", "monsterCount", "group1Active", "group1Template", "group2Id", "group2Active", "group2Template", "group2MemberCount", "slot0Present", "slot0HP", "slot0A49", "slot0A4d", "slot1Present", "slot1HP", "slot1A49", "slot1A4d", "slot2Present", "slot2HP", "slot2A49", "slot2A4d", "slot3Present", "slot3HP", "slot3A49", "slot3A4d", "slot4Present", "slot4HP", "slot4A49", "slot4A4d", "slot5Present", "slot5HP", "slot5A49", "slot5A4d", "slot6Present", "slot6HP", "slot6A49", "slot6A4d", "slot7Present", "slot7HP", "slot7A49", "slot7A4d", "mainStatus0", "mainStatus1", "mainStatus2", "mainStatus3", "mainStatus4", "mainStatus5", "mainStatus6", "mainStatus7", "interceptRangeSelector", "monster1Visible", "monster2Visible", "combatPhase", "battlePointer", "battleActionKind", "battleSpecialFlag", "battleSpell", "selectedCharacterSlot", "combatMode", "specialSpellMode", "specialItemMode", "combatState", "characterPresent", "characterHP", "characterCondition", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range", "item0Spell", "item1Spell", "item2Spell", "item3Spell", "item0Type", "item1Type", "item2Type", "item3Type", "spell1Range", "spell2Range", "spell3Range", "spell4Range", "spell5Range", "spell6Range", "spell88Range", "spell1Target", "spell2Target", "spell3Target", "spell4Target", "spell5Target", "spell6Target", "spell88Target", "mainActionKind", "offActionKind", "offSpell", "offInstance", "offItemIndexEntry", "queuedSpell", "queuedItemInstance", "queuedItemIndexEntry", "specialBaseSpell", "specialOverrideSpell", "specialBaseInstance", "specialOverrideInstance", "specialBaseItemIndex", "specialOverrideItemIndex", "partyX", "partyY", "partyZ", "partyRadius", "memberCount", "character0Quadrant", "partyHeadingDegrees", "mainSpell", "mainInstance", "mainItemIndexEntry", "monster1Active", "monster1F2b", "monster1F9f", "monster1GroupId", "monster1Template", "monster1Enabled", "monster1PhysicalAllowed", "monster1SpellAllowed", "monster1X", "monster1Y", "monster1Z", "monster1Radius", "monster2Active", "monster2F2b", "monster2F9f", "monster2GroupId", "monster2Template", "monster2Enabled", "monster2PhysicalAllowed", "monster2SpellAllowed", "monster2X", "monster2Y", "monster2Z", "monster2Radius", "template1Flag", "template2Flag", "template3Flag", "targetLockFlag", "targetLockMonster", "activeGroupCount", "reserveGroupCount", "storedGroupId"}, Out: []string{"ok"}, F: actionTargetAvailableNode}
}
