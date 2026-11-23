package calc

import "math"

// Fixture wiring: eight party/queue records, one group (template id 2) with
// two monster members, four item records, spells {1..6, 88} with independent
// range and target kind, and four item-instance entries (off hand, queued item,
// special base, special override). Instance inputs only tell null from non-null;
// calc never dereferences them. Phase, battle, selection and mode flags choose the
// attack context the same way as 0x53bc10; 0x4e77b0 then picks the action and
// its spell or item. Differences from the existing nodes: the ranged check
// 0x537780 for the "area" target type only needs the spell range that
// 0x5199f0 would return for a spell or item action, so it is computed here
// instead of through actionAttackRangeNode; 0x5194e0 is called with the
// resolved context (the existing case only exercised context 6 == main hand).
// A live battle item action (context 7, action 8) dereferences a null wrapper
// in the original; this node panics there.
func characterGroupTargetOkNode(in []int64, r Rand) []int64 {
	const (
		iSlot = iota
		iTemplate
		iPhase
		iBattle
		iBattleAct
		iBattleSpecial
		iBattleSpell
		iSelected
		iCombat
		iSpecSpell
		iSpecItem
		iCombatState
		iPresent
		iHP
		iCond
		iMainKind
		iMainAvail
		iOffKind
		iOffAvail
		iMainItem
		iOffItem
		iItemRange
		_
		_
		_
		iItemSpell
		_
		_
		_
		iItemType
		_
		_
		_
		iSpellRange
		_
		_
		_
		_
		_
		_
		iSpellKind
		_
		_
		_
		_
		_
		_
		iMainAct
		iOffAct
		iOffSpell
		iOffInst
		iOffIdx
		iQSpell
		iQInst
		iQIdx
		iSpBaseSpell
		iSpOvSpell
		iSpBaseInst
		iSpOvInst
		iSpBaseIdx
		iSpOvIdx
		iPartyX
		iPartyY
		iPartyZ
		iPartyR
		iCount
		iMember // 7 fields per member: enabled, physical, spell, x, y, z, radius
		_
		_
		_
		_
		_
		_
		_
		_
		_
		_
		_
		_
		_
		iQuad
		iHeading
		nIn
	)
	if len(in) != nIn {
		panic("character group target: expected 82 inputs")
	}
	if in[iSlot] < 0 || in[iSlot] > 7 || in[iCount] < 0 || in[iCount] > 2 {
		panic("character group target: slot or member count outside fixture")
	}
	for _, i := range []int{iOffIdx, iQIdx, iSpBaseIdx, iSpOvIdx} {
		if in[i] < -1 || in[i] > 3 {
			panic("character group target: item index outside fixture")
		}
	}
	for i := 0; i < 7; i++ {
		if in[iSpellRange+i] < -1 || in[iSpellRange+i] > 3 || in[iSpellKind+i] < 0 || in[iSpellKind+i] > 10 {
			panic("character group target: spell record outside fixture")
		}
	}
	no, yes := []int64{0}, []int64{1}
	// The group's template flag byte (+0x24a), read through 0x510180.
	if in[iTemplate] != 0 {
		return no
	}
	// Attack context, 0x53bc10 (0x53b920 passes 0..5 and 7 through unchanged).
	c := int64(0)
	switch {
	case in[iPhase] == 7 && in[iBattle] != 0 && in[iBattleAct] != 4294967295:
		c = 7
		if in[iBattleSpecial] != 0 && in[iBattleAct] == 7 {
			c = 3
		}
		if in[iBattleSpecial] != 0 && in[iBattleAct] == 8 {
			c = 4
		}
	case in[iSlot] == in[iSelected] && (in[iSpecSpell] != 0 || in[iSpecItem] != 0):
		c = 2
	case in[iCombat] != 0:
		c = 1
	}
	if c == 0 {
		return no
	}
	// Action source of 0x4e77b0 for this context.
	act, spell, inst, idx, wrapper := int64(0), int64(0), int64(0), int64(0), false
	switch c {
	case 1:
		act, spell, inst, idx, wrapper = in[iOffAct], in[iOffSpell], in[iOffInst], in[iOffIdx], true
	case 2:
		if in[iSpecSpell] != 0 { // 0x5a1350: override spell, else base spell.
			act, spell = 7, in[iSpOvSpell]
			if spell == 0 {
				spell = in[iSpBaseSpell]
			}
		} else { // 0x59e0d0: override entry when non-null, else base entry.
			act, wrapper = 8, true
			inst, idx = in[iSpBaseInst], in[iSpBaseIdx]
			if in[iSpOvInst] != 0 {
				inst, idx = in[iSpOvInst], in[iSpOvIdx]
			}
		}
	case 3:
		act, spell = 7, in[iQSpell]
	case 4:
		act, inst, idx, wrapper = 8, in[iQInst], in[iQIdx], true
	case 7:
		act, spell = in[iBattleAct], in[iBattleSpell]
	}
	// 0x53c270 alive test; only actions 10 and 11 survive a failed test.
	if !(in[iPresent] != 0 && in[iHP] != 0 && uint32(in[iCond]) < 18) && act != 10 && act != 11 {
		act = 4294967295
	}
	spellAt := func(id int64) int {
		switch {
		case id >= 1 && id <= 6:
			return int(id - 1)
		case id == 88:
			return 6
		}
		panic("character group target: spell id outside fixture")
	}
	// 0x5010f0: spell 0 has no target type; the item flag is 0x5207c0.
	targetOf := func(id, itemType int64) int64 {
		if id == 0 {
			return 0
		}
		return itemSpellTargetTypeNode([]int64{1, 0, id, itemType, in[iSpellKind+spellAt(id)], in[iBattle]}, r)[0]
	}
	esi, itemSpell := int64(0), int64(0)
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
			panic("character group target: battle item action has no wrapper")
		}
		if inst != 0 && idx != -1 { // 0x536e02..0x536e0b
			itemSpell = in[iItemSpell+int(idx)]
			if itemSpell != 0 {
				esi = targetOf(itemSpell, in[iItemType+int(idx)])
			}
		}
	}
	count := int(in[iCount])
	switch esi {
	case 5: // 0x537780: any member within the action's reach.
		if queuedSpellRuleErr != nil {
			panic(queuedSpellRuleErr)
		}
		t := queuedSpellRuleData
		id := spell
		if act == 8 {
			id = itemSpell
		}
		var units float64
		switch in[iSpellRange+spellAt(id)] {
		case -1:
			units = 0
		case 0:
			units = 2
		case 1:
			units = 4
		case 2:
			units = 25
		case 3:
			units = 50
		}
		reach := float32(units * t.scale)
		f := func(i int) float64 { return float64(math.Float32frombits(uint32(in[i]))) }
		partyY := float64(float32(f(iPartyY) - t.height))
		for k := 0; k < count; k++ {
			m := iMember + 7*k
			dx, dy, dz := f(m+3)-f(iPartyX), f(m+4)-partyY, f(m+5)-f(iPartyZ)
			d := math.Sqrt(dz*dz+dy*dy+dx*dx) - f(m+6) - f(iPartyR)
			if !(d >= t.minimum) {
				d = t.minimum
			}
			if d32 := float32(d); d32 <= reach || math.IsNaN(float64(d32)) || math.IsNaN(float64(reach)) {
				return yes
			}
		}
		return no
	case 1, 2: // 0x5194e0 against every group member, hand 2, no failure feedback.
		hits := 0
		for k := 0; k < count; k++ {
			m := iMember + 7*k
			v := make([]int64, 78)
			copy(v[0:4], []int64{in[iSlot], 2, c, 0})
			copy(v[4:7], in[m:m+3])
			copy(v[7:25], []int64{in[iCombatState], in[iCombat], in[iSpecSpell], in[iSpecItem], in[iPresent], in[iHP], in[iCond],
				in[iMainKind], in[iMainAvail], in[iOffKind], in[iOffAvail], 1, in[iMainItem], in[iOffItem],
				in[iItemRange], in[iItemRange+1], in[iItemRange+2], in[iItemRange+3]})
			if act == 7 {
				v[25], v[26] = spell, in[iSpellRange+spellAt(spell)]
			}
			if act == 8 {
				v[27], v[34] = itemSpell, in[iSpellRange+spellAt(itemSpell)]
			}
			v[31], v[32], v[33] = in[iMainAct], in[iOffAct], in[iBattleAct]
			copy(v[38:41], in[iPartyX:iPartyX+3])
			copy(v[41:45], in[m+3:m+7])
			v[45], v[46], v[54] = in[iPartyR], in[iQuad], in[iHeading]
			for i := 55; i < 70; i++ {
				v[i] = 255
			}
			if monsterTargetAttackRangeNode(v, r)[0] != 0 {
				hits++
			}
		}
		if hits > 0 {
			return yes
		}
	}
	return no
}

func init() {
	Nodes["00536d60"] = Node{Addr: "00536d60", In: []string{"characterSlot", "groupTemplateFlag", "combatPhase", "battlePointer", "battleActionKind", "battleSpecialFlag", "battleSpell", "selectedCharacterSlot", "combatMode", "specialSpellMode", "specialItemMode", "combatState", "characterPresent", "characterHP", "characterCondition", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range", "item0Spell", "item1Spell", "item2Spell", "item3Spell", "item0Type", "item1Type", "item2Type", "item3Type", "spell1Range", "spell2Range", "spell3Range", "spell4Range", "spell5Range", "spell6Range", "spell88Range", "spell1Target", "spell2Target", "spell3Target", "spell4Target", "spell5Target", "spell6Target", "spell88Target", "mainActionKind", "offActionKind", "offSpell", "offInstance", "offItemIndexEntry", "queuedSpell", "queuedItemInstance", "queuedItemIndexEntry", "specialBaseSpell", "specialOverrideSpell", "specialBaseInstance", "specialOverrideInstance", "specialBaseItemIndex", "specialOverrideItemIndex", "partyX", "partyY", "partyZ", "partyRadius", "memberCount", "member1Enabled", "member1PhysicalAllowed", "member1SpellAllowed", "member1X", "member1Y", "member1Z", "member1Radius", "member2Enabled", "member2PhysicalAllowed", "member2SpellAllowed", "member2X", "member2Y", "member2Z", "member2Radius", "character0Quadrant", "partyHeadingDegrees"}, Out: []string{"ok"}, F: characterGroupTargetOkNode}
}
