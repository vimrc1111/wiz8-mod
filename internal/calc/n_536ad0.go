package calc

// characterMonsterTargetOkNode is 0x536ad0: may character `slot` target the
// monster whose id is monsterId. Fixture wiring: the 0x536d60 party/queue/item/
// spell layout, plus two monsters (ids 123 and 124) that each carry the reject
// flags, a group id and a template index 1..3 (template flag bytes are inputs).
// Differences from characterGroupTargetOkNode (0x536d60): context 0 does not
// end the node, it continues with spell target type 0 and reaches 0x5194e0 with
// the queue's main action (spell, entry pointer and entry item index are
// inputs); an action whose target type is 5 goes to 0x519920 with the monster's
// group id; the failure-feedback flag is passed through. The ranged 0x5194e0
// call gets the resolved context, as in 0x536d60 (the original passes 6 and
// 0x5194e0 resolves it again). Panics: monster id outside {123,124}, template
// index outside 1..3, a group list count above 1 (one group entry), spell id outside the fixture, a live battle item action
// (context 7, action 8, null wrapper in the original), and a non-null item
// entry whose index is -1 or >= 4 reaching the range check of 0x5194e0
// (assert in 0x5207e0).
func characterMonsterTargetOkNode(in []int64, r Rand) []int64 {
	const (
		iCharacterSlot = iota
		iMonsterId
		iTargetFlag
		iShowFailure
		iCombatPhase
		iBattlePointer
		iBattleActionKind
		iBattleSpecialFlag
		iBattleSpell
		iSelectedCharacterSlot
		iCombatMode
		iSpecialSpellMode
		iSpecialItemMode
		iCombatState
		iCharacterPresent
		iCharacterHP
		iCharacterCondition
		iMainAttackKind
		iMainAttackAvailable
		iOffAttackKind
		iOffAttackAvailable
		iMainItemIndex
		iOffItemIndex
		iItem0Range
		iItem1Range
		iItem2Range
		iItem3Range
		iItem0Spell
		iItem1Spell
		iItem2Spell
		iItem3Spell
		iItem0Type
		iItem1Type
		iItem2Type
		iItem3Type
		iSpell1Range
		iSpell2Range
		iSpell3Range
		iSpell4Range
		iSpell5Range
		iSpell6Range
		iSpell88Range
		iSpell1Target
		iSpell2Target
		iSpell3Target
		iSpell4Target
		iSpell5Target
		iSpell6Target
		iSpell88Target
		iMainActionKind
		iOffActionKind
		iOffSpell
		iOffInstance
		iOffItemIndexEntry
		iQueuedSpell
		iQueuedItemInstance
		iQueuedItemIndexEntry
		iSpecialBaseSpell
		iSpecialOverrideSpell
		iSpecialBaseInstance
		iSpecialOverrideInstance
		iSpecialBaseItemIndex
		iSpecialOverrideItemIndex
		iPartyX
		iPartyY
		iPartyZ
		iPartyRadius
		iMemberCount
		iCharacter0Quadrant
		iPartyHeadingDegrees
		iMainSpell
		iMainInstance
		iMainItemIndexEntry
		iMonster1Active
		iMonster1F2b
		iMonster1F9f
		iMonster1GroupId
		iMonster1Template
		iMonster1Enabled
		iMonster1PhysicalAllowed
		iMonster1SpellAllowed
		iMonster1X
		iMonster1Y
		iMonster1Z
		iMonster1Radius
		iMonster2Active
		iMonster2F2b
		iMonster2F9f
		iMonster2GroupId
		iMonster2Template
		iMonster2Enabled
		iMonster2PhysicalAllowed
		iMonster2SpellAllowed
		iMonster2X
		iMonster2Y
		iMonster2Z
		iMonster2Radius
		iTemplate1Flag
		iTemplate2Flag
		iTemplate3Flag
		iTargetLockFlag
		iTargetLockMonster
		iActiveGroupCount
		iReserveGroupCount
		iStoredGroupId
		nIn
	)
	if len(in) != nIn {
		panic("character monster target: expected 105 inputs")
	}
	if in[iCharacterSlot] < 0 || in[iCharacterSlot] > 7 || in[iMemberCount] < 0 || in[iMemberCount] > 2 {
		panic("character monster target: slot or member count outside fixture")
	}
	if in[iActiveGroupCount] < 0 || in[iActiveGroupCount] > 1 || in[iReserveGroupCount] < 0 || in[iReserveGroupCount] > 1 {
		panic("character monster target: group list count outside fixture")
	}
	for _, i := range []int{iOffItemIndexEntry, iQueuedItemIndexEntry, iSpecialBaseItemIndex, iSpecialOverrideItemIndex, iMainItemIndexEntry} {
		if in[i] < -1 || in[i] > 3 {
			panic("character monster target: item index outside fixture")
		}
	}
	for i := 0; i < 7; i++ {
		if in[iSpell1Range+i] < -1 || in[iSpell1Range+i] > 3 || in[iSpell1Target+i] < 0 || in[iSpell1Target+i] > 10 {
			panic("character monster target: spell record outside fixture")
		}
	}
	no, yes := []int64{0}, []int64{1}
	mi := 0 // monster record: 0 = id 123, 1 = id 124.
	switch in[iMonsterId] {
	case 123:
	case 124:
		mi = 1
	default:
		panic("character monster target: monster id outside fixture")
	}
	m := iMonster1Active + 12*mi
	const (
		mActive = iota
		mF2b
		mF9f
		mGroup
		mTemplate
		mEnabled
		mPhysical
		mSpell
		mX
	)
	// 0x536aef..0x536b4a: record flags, then the locked target.
	if in[m+mActive]&0xff == 0 || in[m+mF2b] == 0 || in[m+mF9f] != 0 {
		return no
	}
	if in[iTargetLockFlag] != 0 && in[iTargetLockMonster] != in[iMonsterId] {
		return no
	}
	// Attack context, as the inline copy of 0x53bc10 (never 5, 6 or 8).
	c := int64(0)
	switch {
	case in[iCombatPhase] == 7 && in[iBattlePointer] != 0 && in[iBattleActionKind] != 4294967295:
		c = 7
		if in[iBattleSpecialFlag] != 0 && in[iBattleActionKind] == 7 {
			c = 3
		}
		if in[iBattleSpecialFlag] != 0 && in[iBattleActionKind] == 8 {
			c = 4
		}
	case in[iCharacterSlot] == in[iSelectedCharacterSlot] && (in[iSpecialSpellMode] != 0 || in[iSpecialItemMode] != 0):
		c = 2
	case in[iCombatMode] != 0:
		c = 1
	}
	// Action source of 0x4e77b0; context 0 is only read inside 0x5194e0.
	act, spell, inst, idx, wrapper := int64(0), int64(0), int64(0), int64(0), false
	switch c {
	case 0:
		act, spell, inst, idx, wrapper = in[iMainActionKind], in[iMainSpell], in[iMainInstance], in[iMainItemIndexEntry], true
	case 1:
		act, spell, inst, idx, wrapper = in[iOffActionKind], in[iOffSpell], in[iOffInstance], in[iOffItemIndexEntry], true
	case 2:
		if in[iSpecialSpellMode] != 0 { // 0x5a1350: override spell, else base spell.
			act, spell = 7, in[iSpecialOverrideSpell]
			if spell == 0 {
				spell = in[iSpecialBaseSpell]
			}
		} else { // 0x59e0d0: override entry when non-null, else base entry.
			act, wrapper = 8, true
			inst, idx = in[iSpecialBaseInstance], in[iSpecialBaseItemIndex]
			if in[iSpecialOverrideInstance] != 0 {
				inst, idx = in[iSpecialOverrideInstance], in[iSpecialOverrideItemIndex]
			}
		}
	case 3:
		act, spell = 7, in[iQueuedSpell]
	case 4:
		act, inst, idx, wrapper = 8, in[iQueuedItemInstance], in[iQueuedItemIndexEntry], true
	case 7:
		act, spell = in[iBattleActionKind], in[iBattleSpell]
	}
	// 0x53c270 alive test; only actions 10 and 11 survive a failed test.
	alive := in[iCharacterPresent] != 0 && in[iCharacterHP] != 0 && uint32(in[iCharacterCondition]) < 18
	if !alive && act != 10 && act != 11 {
		act = 4294967295
	}
	spellAt := func(id int64) int {
		switch {
		case id >= 1 && id <= 6:
			return int(id - 1)
		case id == 88:
			return 6
		}
		panic("character monster target: spell id outside fixture")
	}
	// 0x5010f0: spell 0 has no target type; the item flag is 0x5207c0.
	targetOf := func(id, itemType int64) int64 {
		if id == 0 {
			return 0
		}
		return itemSpellTargetTypeNode([]int64{1, 0, id, itemType, in[iSpell1Target+spellAt(id)], in[iBattlePointer]}, r)[0]
	}
	esi, itemSpell := int64(0), int64(0)
	if c != 0 {
		switch act { // jump table at 0x536f38; kinds above 8 (and -1) give 0.
		case 0, 1:
			esi = 2
		case 2:
			esi = 4
		case 5:
			esi = 1
		case 7:
			esi = targetOf(spell, 0)
		case 8:
			if !wrapper {
				panic("character monster target: battle item action has no wrapper")
			}
			if inst != 0 && idx != -1 { // 0x536e02..0x536e0b
				itemSpell = in[iItem0Spell+int(idx)]
				if itemSpell != 0 {
					esi = targetOf(itemSpell, in[iItem0Type+int(idx)])
				}
			}
		}
	}
	switch esi {
	case 2, 5:
	case 1:
		if in[iTargetFlag] == 0 {
			return no
		}
	case 0:
		if in[iCombatMode] != 0 {
			return no
		}
	default:
		return no
	}
	// 0x4e5720: the monster template's flag byte (+0x24a).
	if in[m+mTemplate] < 1 || in[m+mTemplate] > 3 {
		panic("character monster target: template index outside fixture")
	}
	if in[iTemplate1Flag+int(in[m+mTemplate])-1]&0xff != 0 && esi != 0 {
		return no
	}
	// The ranged checks read the item entry of the action only for action 8.
	// Context 0 (the main action) is read inside 0x5194e0 from the queue.
	if act == 8 && inst != 0 && (idx < 0 || idx > 3) {
		if c != 0 {
			// 0x5194e0 repeats the entry lookup for its range check.
			if in[m+mEnabled]&0xff == 1 && alive {
				panic("character monster target: item entry index asserts in 0x5207e0")
			}
		} else if in[m+mEnabled]&0xff == 1 && alive && (in[iCombatPhase] != 0 || in[iCombatMode] != 0 || in[iSpecialSpellMode] != 0 || in[iSpecialItemMode] != 0) {
			panic("character monster target: item entry index asserts in 0x5207e0")
		}
	}
	if act == 8 && c == 0 && inst != 0 && idx >= 0 && idx <= 3 {
		itemSpell = in[iItem0Spell+int(idx)]
	}
	if esi == 5 {
		// 0x519920 with the monster's group id.
		v := make([]int64, 34)
		copy(v[0:4], []int64{in[iCharacterSlot], in[m+mGroup], c, in[iShowFailure]})
		copy(v[4:7], []int64{in[iActiveGroupCount], in[iReserveGroupCount], in[iStoredGroupId]})
		copy(v[7:10], []int64{in[iCharacterPresent], in[iCharacterHP], in[iCharacterCondition]})
		sp := spell
		if act == 8 {
			sp = itemSpell
		}
		v[10] = sp
		if sp != 0 {
			v[11] = in[iSpell1Range+spellAt(sp)]
		}
		v[12] = in[iMemberCount]
		copy(v[13:16], in[iPartyX:iPartyX+3])
		copy(v[16:20], in[iMonster1X:iMonster1X+4])
		v[20] = in[iPartyRadius]
		copy(v[21:25], in[iMonster2X:iMonster2X+4])
		v[25], v[26], v[27], v[28] = in[iMainAttackKind], in[iMainAttackAvailable], in[iOffAttackKind], in[iOffAttackAvailable]
		v[29], v[30] = in[iItem0Range+int(in[iMainItemIndex])&3], in[iItem0Range+int(in[iOffItemIndex])&3]
		v[31] = in[iSpecialSpellMode]
		v[32], v[33] = in[iMainActionKind], in[iOffActionKind]
		return monsterGroupTargetValidNode(v, r)
	}
	// 0x5194e0, hand 2, with the resolved context.
	v := make([]int64, 78)
	copy(v[0:4], []int64{in[iCharacterSlot], 2, c, in[iShowFailure]})
	copy(v[4:7], in[m+mEnabled:m+mEnabled+3])
	copy(v[7:25], []int64{in[iCombatState], in[iCombatMode], in[iSpecialSpellMode], in[iSpecialItemMode], in[iCharacterPresent], in[iCharacterHP], in[iCharacterCondition],
		in[iMainAttackKind], in[iMainAttackAvailable], in[iOffAttackKind], in[iOffAttackAvailable], 1, in[iMainItemIndex], in[iOffItemIndex],
		in[iItem0Range], in[iItem1Range], in[iItem2Range], in[iItem3Range]})
	if act == 7 && spell != 0 {
		v[25], v[26] = spell, in[iSpell1Range+spellAt(spell)]
	}
	if act == 8 && itemSpell != 0 {
		v[27], v[34] = itemSpell, in[iSpell1Range+spellAt(itemSpell)]
	}
	v[31], v[32], v[33] = in[iMainActionKind], in[iOffActionKind], in[iBattleActionKind]
	copy(v[38:41], in[iPartyX:iPartyX+3])
	copy(v[41:45], in[m+mX:m+mX+4])
	v[45], v[46], v[54] = in[iPartyRadius], in[iCharacter0Quadrant], in[iPartyHeadingDegrees]
	for i := 55; i < 70; i++ {
		v[i] = 255
	}
	if monsterTargetAttackRangeNode(v, r)[0] != 0 {
		return yes
	}
	return no
}

func init() {
	Nodes["00536ad0"] = Node{Addr: "00536ad0", In: []string{"characterSlot", "monsterId", "targetFlag", "showFailure", "combatPhase", "battlePointer", "battleActionKind", "battleSpecialFlag", "battleSpell", "selectedCharacterSlot", "combatMode", "specialSpellMode", "specialItemMode", "combatState", "characterPresent", "characterHP", "characterCondition", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range", "item0Spell", "item1Spell", "item2Spell", "item3Spell", "item0Type", "item1Type", "item2Type", "item3Type", "spell1Range", "spell2Range", "spell3Range", "spell4Range", "spell5Range", "spell6Range", "spell88Range", "spell1Target", "spell2Target", "spell3Target", "spell4Target", "spell5Target", "spell6Target", "spell88Target", "mainActionKind", "offActionKind", "offSpell", "offInstance", "offItemIndexEntry", "queuedSpell", "queuedItemInstance", "queuedItemIndexEntry", "specialBaseSpell", "specialOverrideSpell", "specialBaseInstance", "specialOverrideInstance", "specialBaseItemIndex", "specialOverrideItemIndex", "partyX", "partyY", "partyZ", "partyRadius", "memberCount", "character0Quadrant", "partyHeadingDegrees", "mainSpell", "mainInstance", "mainItemIndexEntry", "monster1Active", "monster1F2b", "monster1F9f", "monster1GroupId", "monster1Template", "monster1Enabled", "monster1PhysicalAllowed", "monster1SpellAllowed", "monster1X", "monster1Y", "monster1Z", "monster1Radius", "monster2Active", "monster2F2b", "monster2F9f", "monster2GroupId", "monster2Template", "monster2Enabled", "monster2PhysicalAllowed", "monster2SpellAllowed", "monster2X", "monster2Y", "monster2Z", "monster2Radius", "template1Flag", "template2Flag", "template3Flag", "targetLockFlag", "targetLockMonster", "activeGroupCount", "reserveGroupCount", "storedGroupId"}, Out: []string{"ok"}, F: characterMonsterTargetOkNode}
}
