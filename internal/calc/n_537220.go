package calc

import "math"

// 0x537220: spell target class (0x5010f0), target descriptor (0x53b7f0),
// descriptor/class compatibility (0x537160 with 0x536190) and target
// reachability (0x519180). All sub-results are rebuilt from record fields.
//
// Inputs: the combat/special mode flags 0x683f94..96, the combat phase
// 0x683fe7, the phase type 0x68ec78, the battle record action 0x68edcc+0x2f0
// and the selected character 0x68518d are real inputs. Attack context 6 is
// resolved the way 0x53b7f0/0x53b920 do it (battle action, special selection,
// combat mode) into an action source 0..5 or 7; each of 0..5 has its own
// descriptor slot (Q+0x1d, +0x4d, 0x68408b, +0x81, +0xa9, +0xd1). Source 7
// has a NULL descriptor, which 0x537160 rejects. A 136-input vector (every
// mode global zero) is accepted as the legacy shape used by callers.
// Records exist for party slots 0..7, spell ids 0..9, 88 and 119, one monster
// (ID 123) and one group (ID 456) in an empty world; list pointers in action
// sources are either NULL or the one fixture list. When the original would take
// an assertion or fault (spell kind above 10, coordinate target without a
// coordinate spell, character target -1 reaching the formation test, item
// spell from a missing list, an unmapped battle record while the phase type
// is 7) calc panics instead of inventing a result.
func actionTargetEligibleNode(in []int64, r Rand) []int64 {
	const (
		iCharacterSlot = iota
		iSpell
		iAttackContext
		iCharacterPresent
		iCharacterHP
		iCharacterCondition
		iCharacterInParty
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
		iMainActionKind
		iOffActionKind
		iMainSpell
		iOffSpell
		iQueuedSpell
		iSpell0Range
		iSpell1Range
		iSpell2Range
		iSpell3Range
		iSpell4Range
		iSpell5Range
		iSpell6Range
		iSpell7Range
		iSpell8Range
		iSpell9Range
		iItemEntriesPointer
		iSelectedItemIndex
		iBattlePresent
		iTargetKindCtx3
		iTargetKindCtx4
		iTargetKindCtx5
		iTargetKindCtx6
		iCharacterTargetCtx3
		iCharacterTargetCtx4
		iCharacterTargetCtx5
		iCharacterTargetCtx6
		iMonsterTargetCtx3
		iMonsterTargetCtx4
		iMonsterTargetCtx5
		iMonsterTargetCtx6
		iGroupTargetCtx3
		iGroupTargetCtx4
		iGroupTargetCtx5
		iGroupTargetCtx6
		iTargetXCtx3
		iTargetXCtx4
		iTargetXCtx5
		iTargetXCtx6
		iTargetYCtx3
		iTargetYCtx4
		iTargetYCtx5
		iTargetYCtx6
		iTargetZCtx3
		iTargetZCtx4
		iTargetZCtx5
		iTargetZCtx6
		iTargetListItem
		iTargetPresentCtx3
		iTargetPresentCtx4
		iTargetPresentCtx5
		iTargetPresentCtx6
		iTargetHPCtx3
		iTargetHPCtx4
		iTargetHPCtx5
		iTargetHPCtx6
		iTargetConditionCtx3
		iTargetConditionCtx4
		iTargetConditionCtx5
		iTargetConditionCtx6
		iTargetRecoverableCtx3
		iTargetRecoverableCtx4
		iTargetRecoverableCtx5
		iTargetRecoverableCtx6
		iTargetRecoveryBlockedCtx3
		iTargetRecoveryBlockedCtx4
		iTargetRecoveryBlockedCtx5
		iTargetRecoveryBlockedCtx6
		iSpell1Kind
		iSpell2Kind
		iSpell3Kind
		iSpell4Kind
		iSpell5Kind
		iSpell6Kind
		iSpell7Kind
		iSpell8Kind
		iSpell9Kind
		iSpell88Kind
		iSpell119Kind
		iSpell119Range
		iMonsterHP
		iMonsterRemoved
		iMonsterPresent
		iMonsterVisible
		iMonsterReachable
		iGroupActive
		iGroupSize
		iGroupMemberCount
		iPartyX
		iPartyY
		iPartyZ
		iMonsterX
		iMonsterY
		iMonsterZ
		iMonsterRadius
		iPartyRadius
		iCoordinatePartyRadius
		iCharacter0Quadrant
		iCharacter1Quadrant
		iCharacter2Quadrant
		iCharacter3Quadrant
		iCharacter4Quadrant
		iCharacter5Quadrant
		iCharacter6Quadrant
		iCharacter7Quadrant
		iCenterSlot0
		iCenterSlot1
		iCenterSlot2
		iCharacter0FormationUnavailable
		iCharacter1FormationUnavailable
		iCharacter2FormationUnavailable
		iCharacter3FormationUnavailable
		iCharacter4FormationUnavailable
		iCharacter5FormationUnavailable
		iCharacter6FormationUnavailable
		iCharacter7FormationUnavailable
		iCombatMode
		iSpecialSpellMode
		iSpecialItemMode
		iCombatPhase
		iPhaseType
		iSelectedCharacterSlot
		iBattleActionKind
		iBattleSpecialFlag
		iSpecialBaseSpell
		iSpecialOverrideSpell
		iSpecialItemOverridePointer
		iSpecialBaseItemIndex
		iSpecialOverrideItemIndex
		iMonsterPhysicalAllowed
		iPartyHeadingDegrees
		iQuadrant0Slot0
		iQuadrant0Slot1
		iQuadrant0Slot2
		iQuadrant1Slot0
		iQuadrant1Slot1
		iQuadrant1Slot2
		iQuadrant2Slot0
		iQuadrant2Slot1
		iQuadrant2Slot2
		iQuadrant3Slot0
		iQuadrant3Slot1
		iQuadrant3Slot2
		iTargetKindOff
		iCharacterTargetOff
		iMonsterTargetOff
		iGroupTargetOff
		iTargetXOff
		iTargetYOff
		iTargetZOff
		iTargetPresentOff
		iTargetHPOff
		iTargetConditionOff
		iTargetRecoverableOff
		iTargetRecoveryBlockedOff
		iTargetKindGlob
		iCharacterTargetGlob
		iMonsterTargetGlob
		iGroupTargetGlob
		iTargetXGlob
		iTargetYGlob
		iTargetZGlob
		iTargetPresentGlob
		iTargetHPGlob
		iTargetConditionGlob
		iTargetRecoverableGlob
		iTargetRecoveryBlockedGlob
		inputCount
	)
	const legacyCount, battleRecord = iCombatMode, 536870912 + 75000
	lim := [inputCount][2]int64{
		{0, 7},                    // characterSlot
		{0, 4294967295},           // spell
		{3, 6},                    // attackContext
		{0, 255},                  // characterPresent
		{0, 4294967295},           // characterHP
		{0, 4294967295},           // characterCondition
		{1, 255},                  // characterInParty
		{0, 4294967295},           // mainAttackKind
		{0, 255},                  // mainAttackAvailable
		{0, 4294967295},           // offAttackKind
		{0, 255},                  // offAttackAvailable
		{0, 4294967295},           // mainItemIndex
		{0, 4294967295},           // offItemIndex
		{0, 255},                  // item0Range
		{0, 255},                  // item1Range
		{0, 255},                  // item2Range
		{0, 255},                  // item3Range
		{0, 9},                    // item0Spell
		{0, 9},                    // item1Spell
		{0, 9},                    // item2Spell
		{0, 9},                    // item3Spell
		{0, 4294967295},           // mainActionKind
		{0, 4294967295},           // offActionKind
		{0, 9},                    // mainSpell
		{0, 9},                    // offSpell
		{0, 9},                    // queuedSpell
		{-1, 3},                   // spell0Range
		{-1, 3},                   // spell1Range
		{-1, 3},                   // spell2Range
		{-1, 3},                   // spell3Range
		{-1, 3},                   // spell4Range
		{-1, 3},                   // spell5Range
		{-1, 3},                   // spell6Range
		{-1, 3},                   // spell7Range
		{-1, 3},                   // spell8Range
		{-1, 3},                   // spell9Range
		{0, 536949912},            // itemEntriesPointer
		{0, 3},                    // selectedItemIndex
		{0, 4294967295},           // battlePresent
		{0, 4294967295},           // targetKindCtx3
		{0, 4294967295},           // targetKindCtx4
		{0, 4294967295},           // targetKindCtx5
		{0, 4294967295},           // targetKindCtx6
		{-1, 7},                   // characterTargetCtx3
		{-1, 7},                   // characterTargetCtx4
		{-1, 7},                   // characterTargetCtx5
		{-1, 7},                   // characterTargetCtx6
		{0, 4294967295},           // monsterTargetCtx3
		{0, 4294967295},           // monsterTargetCtx4
		{0, 4294967295},           // monsterTargetCtx5
		{0, 4294967295},           // monsterTargetCtx6
		{0, 4294967295},           // groupTargetCtx3
		{0, 4294967295},           // groupTargetCtx4
		{0, 4294967295},           // groupTargetCtx5
		{0, 4294967295},           // groupTargetCtx6
		{0, 4294967295},           // targetXCtx3
		{0, 4294967295},           // targetXCtx4
		{0, 4294967295},           // targetXCtx5
		{0, 4294967295},           // targetXCtx6
		{0, 4294967295},           // targetYCtx3
		{0, 4294967295},           // targetYCtx4
		{0, 4294967295},           // targetYCtx5
		{0, 4294967295},           // targetYCtx6
		{0, 4294967295},           // targetZCtx3
		{0, 4294967295},           // targetZCtx4
		{0, 4294967295},           // targetZCtx5
		{0, 4294967295},           // targetZCtx6
		{-2147483648, 2147483647}, // targetListItem
		{0, 255},                  // targetPresentCtx3
		{0, 255},                  // targetPresentCtx4
		{0, 255},                  // targetPresentCtx5
		{0, 255},                  // targetPresentCtx6
		{0, 4294967295},           // targetHPCtx3
		{0, 4294967295},           // targetHPCtx4
		{0, 4294967295},           // targetHPCtx5
		{0, 4294967295},           // targetHPCtx6
		{0, 4294967295},           // targetConditionCtx3
		{0, 4294967295},           // targetConditionCtx4
		{0, 4294967295},           // targetConditionCtx5
		{0, 4294967295},           // targetConditionCtx6
		{0, 4294967295},           // targetRecoverableCtx3
		{0, 4294967295},           // targetRecoverableCtx4
		{0, 4294967295},           // targetRecoverableCtx5
		{0, 4294967295},           // targetRecoverableCtx6
		{0, 4294967295},           // targetRecoveryBlockedCtx3
		{0, 4294967295},           // targetRecoveryBlockedCtx4
		{0, 4294967295},           // targetRecoveryBlockedCtx5
		{0, 4294967295},           // targetRecoveryBlockedCtx6
		{0, 4294967295},           // spell1Kind
		{0, 4294967295},           // spell2Kind
		{0, 4294967295},           // spell3Kind
		{0, 4294967295},           // spell4Kind
		{0, 4294967295},           // spell5Kind
		{0, 4294967295},           // spell6Kind
		{0, 4294967295},           // spell7Kind
		{0, 4294967295},           // spell8Kind
		{0, 4294967295},           // spell9Kind
		{0, 4294967295},           // spell88Kind
		{0, 4294967295},           // spell119Kind
		{-1, 3},                   // spell119Range
		{0, 4294967295},           // monsterHP
		{0, 4294967295},           // monsterRemoved
		{0, 255},                  // monsterPresent
		{0, 255},                  // monsterVisible
		{0, 255},                  // monsterReachable
		{0, 255},                  // groupActive
		{0, 4294967295},           // groupSize
		{0, 4294967295},           // groupMemberCount
		{0, 4294967295},           // partyX
		{0, 4294967295},           // partyY
		{0, 4294967295},           // partyZ
		{0, 4294967295},           // monsterX
		{0, 4294967295},           // monsterY
		{0, 4294967295},           // monsterZ
		{0, 4294967295},           // monsterRadius
		{0, 4294967295},           // partyRadius
		{0, 4294967295},           // coordinatePartyRadius
		{0, 255},                  // character0Quadrant
		{0, 255},                  // character1Quadrant
		{0, 255},                  // character2Quadrant
		{0, 255},                  // character3Quadrant
		{0, 255},                  // character4Quadrant
		{0, 255},                  // character5Quadrant
		{0, 255},                  // character6Quadrant
		{0, 255},                  // character7Quadrant
		{0, 255},                  // centerSlot0
		{0, 255},                  // centerSlot1
		{0, 255},                  // centerSlot2
		{0, 255},                  // character0FormationUnavailable
		{0, 255},                  // character1FormationUnavailable
		{0, 255},                  // character2FormationUnavailable
		{0, 255},                  // character3FormationUnavailable
		{0, 255},                  // character4FormationUnavailable
		{0, 255},                  // character5FormationUnavailable
		{0, 255},                  // character6FormationUnavailable
		{0, 255},                  // character7FormationUnavailable
		{0, 255},                  // combatMode
		{0, 255},                  // specialSpellMode
		{0, 255},                  // specialItemMode
		{0, 4294967295},           // combatPhase
		{0, 4294967295},           // phaseType
		{-1, 7},                   // selectedCharacterSlot
		{0, 4294967295},           // battleActionKind
		{0, 255},                  // battleSpecialFlag
		{0, 9},                    // specialBaseSpell
		{0, 9},                    // specialOverrideSpell
		{0, 536950112},            // specialItemOverridePointer
		{0, 3},                    // specialBaseItemIndex
		{0, 3},                    // specialOverrideItemIndex
		{0, 255},                  // monsterPhysicalAllowed
		{0, 359},                  // partyHeadingDegrees
		{0, 255},                  // quadrant0Slot0
		{0, 255},                  // quadrant0Slot1
		{0, 255},                  // quadrant0Slot2
		{0, 255},                  // quadrant1Slot0
		{0, 255},                  // quadrant1Slot1
		{0, 255},                  // quadrant1Slot2
		{0, 255},                  // quadrant2Slot0
		{0, 255},                  // quadrant2Slot1
		{0, 255},                  // quadrant2Slot2
		{0, 255},                  // quadrant3Slot0
		{0, 255},                  // quadrant3Slot1
		{0, 255},                  // quadrant3Slot2
		{0, 4294967295},           // targetKindOff
		{-1, 7},                   // characterTargetOff
		{0, 4294967295},           // monsterTargetOff
		{0, 4294967295},           // groupTargetOff
		{0, 4294967295},           // targetXOff
		{0, 4294967295},           // targetYOff
		{0, 4294967295},           // targetZOff
		{0, 255},                  // targetPresentOff
		{0, 4294967295},           // targetHPOff
		{0, 4294967295},           // targetConditionOff
		{0, 4294967295},           // targetRecoverableOff
		{0, 4294967295},           // targetRecoveryBlockedOff
		{0, 4294967295},           // targetKindGlob
		{-1, 7},                   // characterTargetGlob
		{0, 4294967295},           // monsterTargetGlob
		{0, 4294967295},           // groupTargetGlob
		{0, 4294967295},           // targetXGlob
		{0, 4294967295},           // targetYGlob
		{0, 4294967295},           // targetZGlob
		{0, 255},                  // targetPresentGlob
		{0, 4294967295},           // targetHPGlob
		{0, 4294967295},           // targetConditionGlob
		{0, 4294967295},           // targetRecoverableGlob
		{0, 4294967295},           // targetRecoveryBlockedGlob
	}
	if len(in) == legacyCount {
		// The legacy vector keeps every mode global at zero. The two new
		// descriptor slots only matter to sources 1 and 2, which then cannot
		// be selected; their character targets must not alias a party slot.
		full := make([]int64, inputCount)
		copy(full, in)
		full[iCharacterTargetOff], full[iCharacterTargetGlob] = -1, -1
		in = full
	}
	if len(in) != inputCount {
		panic("target eligible: unexpected input count")
	}
	for i, v := range in {
		if v < lim[i][0] || v > lim[i][1] {
			panic("target eligible: field exceeds fixture or storage domain")
		}
	}
	if itemSpellTargetsErr != nil {
		panic(itemSpellTargetsErr)
	}
	if queuedSpellRuleErr != nil {
		panic(queuedSpellRuleErr)
	}
	t := queuedSpellRuleData
	who, spell, ctx := in[iCharacterSlot], in[iSpell], in[iAttackContext]
	// Only the spell records of the fixture exist.
	kindOf := func(id int64) int64 {
		switch {
		case id == 0:
			return 0 // 0x4ac9d0 returns zero for spell 0 without reading a record.
		case id >= 1 && id <= 9:
			return in[iSpell1Kind+int(id)-1]
		case id == 88:
			return in[iSpell88Kind]
		case id == 119:
			return in[iSpell119Kind]
		}
		panic("target eligible: spell id outside fixture records")
	}
	rangeOf := func(id int64) int64 {
		switch {
		case id >= 1 && id <= 9:
			return in[iSpell0Range+int(id)]
		case id == 119:
			return in[iSpell119Range]
		}
		panic("target eligible: spell id has no fixture range")
	}
	// 0x5010f0: target class of the spell argument (0x501190 dispatch).
	cls := int64(0)
	if spell != 0 {
		k := kindOf(spell)
		if k > 10 {
			panic("target eligible: spell target kind outside the dispatch table")
		}
		cls = itemSpellTargets[k]
		switch cls {
		case -1:
			cls = 1
			if spell == 88 {
				cls = 7
			}
		case -2:
			cls = 0
			if in[iBattlePresent] == 0 || ctx == 3 || ctx == 4 {
				cls = 6
			}
		}
	}
	// 0x53b7f0 / 0x53b920: context 6 becomes the action source e. A battle
	// phase type 7 with a battle record chooses by the record's action kind
	// (0x2f0, flag 0x2f8); otherwise the selected character in a special mode
	// takes source 2 and the combat mode source 1, else source 0.
	e := ctx
	if ctx == 6 {
		battle := in[iPhaseType] == 7 && in[iBattlePresent] != 0
		if battle && in[iBattlePresent] != battleRecord {
			panic("target eligible: battle record pointer is not mapped")
		}
		switch {
		case battle && in[iBattleActionKind] != 4294967295:
			switch {
			case in[iBattleActionKind] == 7 && in[iBattleSpecialFlag] != 0:
				e = 3
			case in[iBattleActionKind] == 8 && in[iBattleSpecialFlag] != 0:
				e = 4
			default:
				e = 7
			}
		case who == in[iSelectedCharacterSlot] && (in[iSpecialSpellMode] != 0 || in[iSpecialItemMode] != 0):
			e = 2
		case in[iCombatMode] != 0:
			e = 1
		default:
			e = 0
		}
	}
	if e == 7 {
		return []int64{0} // NULL descriptor: 0x537160 returns false at 0x537168.
	}
	// 0x53b7f0 picks the descriptor slot of the source: Q+0x81, +0xa9, +0xd1
	// for 3..5, Q+0x1d for 0, Q+0x4d for 1 and the globals at 0x68408b for 2.
	// Each slot has its own inputs; a record written by several slots keeps the
	// value of the last slot (the case writes them in the order below).
	const (
		slotCtx3 = iota
		slotCtx4
		slotCtx5
		slotMain
		slotOff
		slotGlob
		slotCount
	)
	sl := [8]int{slotMain, slotOff, slotGlob, slotCtx3, slotCtx4, slotCtx5}[e]
	family := func(ctx3, off, glob int) [slotCount]int {
		return [slotCount]int{ctx3, ctx3 + 1, ctx3 + 2, ctx3 + 3, off, glob}
	}
	fKind := family(iTargetKindCtx3, iTargetKindOff, iTargetKindGlob)
	fChar := family(iCharacterTargetCtx3, iCharacterTargetOff, iCharacterTargetGlob)
	fMon := family(iMonsterTargetCtx3, iMonsterTargetOff, iMonsterTargetGlob)
	fGrp := family(iGroupTargetCtx3, iGroupTargetOff, iGroupTargetGlob)
	fX := family(iTargetXCtx3, iTargetXOff, iTargetXGlob)
	fY := family(iTargetYCtx3, iTargetYOff, iTargetYGlob)
	fZ := family(iTargetZCtx3, iTargetZOff, iTargetZGlob)
	fPresent := family(iTargetPresentCtx3, iTargetPresentOff, iTargetPresentGlob)
	fHP := family(iTargetHPCtx3, iTargetHPOff, iTargetHPGlob)
	fCond := family(iTargetConditionCtx3, iTargetConditionOff, iTargetConditionGlob)
	fRecoverable := family(iTargetRecoverableCtx3, iTargetRecoverableOff, iTargetRecoverableGlob)
	fBlocked := family(iTargetRecoveryBlockedCtx3, iTargetRecoveryBlockedOff, iTargetRecoveryBlockedGlob)
	dKind, dChar, dMon, dGrp := in[fKind[sl]], in[fChar[sl]], in[fMon[sl]], in[fGrp[sl]]
	rec := func(f [slotCount]int) int64 {
		for s := slotCount - 1; s >= 0; s-- {
			if in[fChar[s]] == dChar {
				return in[f[s]]
			}
		}
		panic("unreachable")
	}
	valid := func() bool {
		switch dKind {
		case 1:
			return dChar != -1 && rec(fPresent) != 0 && rec(fHP) != 0 && rec(fCond) < 18 // 0x5361b2..0x536217.
		case 9:
			return dChar != -1 && rec(fPresent) != 0 && rec(fHP) == 0 && rec(fRecoverable) != 0 && rec(fBlocked) == 0 // 0x53621c..0x53628c.
		case 3:
			return dMon == 123 && in[iMonsterHP] != 0 && in[iMonsterRemoved] == 0 && in[iMonsterPresent] != 0 // 0x536291..0x53631e.
		case 4:
			return dGrp == 456 && in[iGroupSize] != 0 && in[iGroupActive] != 0 // 0x53631f..0x53639e.
		case 7:
			return in[iTargetListItem] != -1 // 0x53639f..0x5363c6.
		default:
			return true
		}
	}
	compatible := true // 0x53716f..0x537173: class 0 and classes above 8 pass.
	if cls >= 1 && cls <= 8 {
		switch t.compatibility[cls-1] {
		case 0x53717c:
			compatible = ((dKind == 1 && dChar != -1) || (dKind == 3 && dMon != -1)) && valid()
		case 0x5371a4:
			compatible = dKind == 9 && dChar != -1 && valid()
		case 0x5371b9:
			compatible = (dKind == 2 || (dKind == 4 && dGrp != -1)) && valid()
		case 0x5371cf:
			compatible = dKind == 7 && valid()
		case 0x5371e5:
			compatible = dKind == 6 && valid()
		case 0x5371f4:
			compatible = true
		}
	}
	if !compatible {
		return []int64{0}
	}
	// 0x519180. The caster and the target character share one record when
	// the descriptor names the caster; the later fixture writes are the target's.
	cp, ch, cc := in[iCharacterPresent], in[iCharacterHP], in[iCharacterCondition]
	for s := slotCount - 1; s >= 0; s-- {
		if in[fChar[s]] == who {
			cp, ch, cc = in[fPresent[s]], in[fHP[s]], in[fCond[s]]
			break
		}
	}
	// The selected item of an item action source: sources 0, 1 and 4 use the
	// item list of the action when it exists; source 2 reads the special
	// selection (0x59e0d0: override item, then base).
	itemIndex, itemList := in[iSelectedItemIndex], in[iItemEntriesPointer] != 0
	if e == 2 {
		itemIndex, itemList = in[iSpecialBaseItemIndex], true
		if in[iSpecialItemOverridePointer] != 0 {
			itemIndex = in[iSpecialOverrideItemIndex]
		}
	}
	// Action source records (0x4e77fa..0x4e78ef), the same table 0x5199f0 uses.
	category, sourceSpell := int64(0), int64(0)
	switch e {
	case 0:
		category, sourceSpell = in[iMainActionKind], in[iMainSpell]
	case 1:
		category, sourceSpell = in[iOffActionKind], in[iOffSpell]
	case 2:
		if in[iSpecialSpellMode] != 0 {
			category, sourceSpell = 7, in[iSpecialOverrideSpell] // 0x5a1356: override spell, then base.
			if sourceSpell == 0 {
				sourceSpell = in[iSpecialBaseSpell]
			}
		} else {
			category = 8
		}
	case 3:
		category, sourceSpell = 7, in[iQueuedSpell]
	case 4:
		category = 8
	case 5:
		category = 2
	}
	if (cp == 0 || ch == 0 || cc >= 18) && category != 10 && category != 11 {
		category = 4294967295 // 0x53c283..0x53c2b7, 0x4e792c.
	}
	// 0x5199f0 for hand 0 and the resolved source.
	attackRange := func() int64 {
		v := make([]int64, 51)
		v[0], v[2] = who, e
		v[3], v[4], v[5] = cp, ch, cc
		copy(v[6:23], in[iCharacterInParty:iCharacterInParty+17])
		v[23] = in[iBattleActionKind]
		copy(v[24:27], in[iMainSpell:iMainSpell+3])
		v[28], v[29] = in[iSpecialBaseSpell], in[iSpecialOverrideSpell]
		copy(v[30:40], in[iSpell0Range:iSpell0Range+10])
		v[40], v[41], v[42] = in[iCombatMode], in[iSpecialSpellMode], in[iSpecialItemMode]
		v[46], v[47] = in[iItemEntriesPointer], in[iSelectedItemIndex]
		v[48], v[49], v[50] = in[iSpecialItemOverridePointer], in[iSpecialBaseItemIndex], in[iSpecialOverrideItemIndex]
		return actionAttackRangeNode(v, r)[0]
	}
	f := func(i int) float64 { return float64(math.Float32frombits(uint32(in[i]))) }
	round := func(x float64) float64 { return float64(float32(x)) }
	// Range lengths are literal returns at 0x51933d..0x519356 and
	// 0x51a9b7..0x51a9d0; scale is loaded from 0x5ebc40.
	length := func(k int64) float64 {
		switch k {
		case -1:
			return 0
		case 0:
			return 2 * t.scale
		case 1:
			return 4 * t.scale
		case 2:
			return 25 * t.scale
		case 3:
			return 50 * t.scale
		}
		panic("target eligible: invalid range category")
	}
	monsterDistance := func() float64 {
		dx, dy, dz := f(iMonsterX)-f(iPartyX), f(iMonsterY)-round(f(iPartyY)-t.height), f(iMonsterZ)-f(iPartyZ)
		d := math.Sqrt(dx*dx+dy*dy+dz*dz) - f(iMonsterRadius) - f(iPartyRadius)
		return math.Max(t.minimum, d) // 0x4c7cb0..0x4c7d4a.
	}
	yes := []int64{1}
	switch dKind {
	case 1:
		if dChar == who {
			return yes // Self target, 0x51920b.
		}
		reach := attackRange()
		if reach == -1 {
			return []int64{0}
		}
		if reach != 0 {
			return yes
		}
		if dChar < -1 || dChar > 7 {
			panic("target eligible: character target outside -1..7")
		}
		// 0x51b000 on the eight quadrants, three center slots and flags.
		// Target -1 reads 0x687519 (0x687525 - 12), which is the byte of
		// quadrant 2 slot 2 of the slot table; only the quadrant is read through
		// the target index (flags are read through the center slots).
		formation := make([]int64, 0, 21)
		quadrants := append([]int64(nil), in[iCharacter0Quadrant:iCharacter0Quadrant+8]...)
		target := dChar
		if dChar == -1 {
			target = (who + 1) % 8
			quadrants[target] = in[iQuadrant2Slot2]
		}
		formation = append(formation, who, target)
		formation = append(formation, quadrants...)
		formation = append(formation, in[iCenterSlot0:iCenterSlot0+3]...)
		formation = append(formation, in[iCharacter0FormationUnavailable:iCharacter0FormationUnavailable+8]...)
		if partyFormationRangePenaltyNode(formation, r)[0] != 0 {
			return []int64{0}
		}
		return yes
	case 3:
		// 0x5194e0 on the monster record. The node takes one spell and one item
		// for its source; give it the ones of source e.
		if dMon != 123 {
			return []int64{0}
		}
		spellID, itemSpell := sourceSpell, int64(0)
		if category == 8 && itemList {
			itemSpell = in[iItem0Spell+int(itemIndex)]
		}
		spellRange, itemRange := int64(0), int64(0)
		if spellID != 0 {
			spellRange = rangeOf(spellID)
		}
		if itemSpell != 0 {
			itemRange = rangeOf(itemSpell)
		}
		v := make([]int64, 78)
		v[0], v[2] = who, e
		v[4], v[5], v[6] = in[iMonsterVisible], in[iMonsterPhysicalAllowed], in[iMonsterReachable]
		v[7], v[8], v[9], v[10] = in[iCombatPhase], in[iCombatMode], in[iSpecialSpellMode], in[iSpecialItemMode]
		v[11], v[12], v[13] = cp, ch, cc
		copy(v[14:18], in[iMainAttackKind:iMainAttackKind+4])
		v[18] = in[iCharacterInParty]
		v[19], v[20] = in[iMainItemIndex], in[iOffItemIndex]
		copy(v[21:25], in[iItem0Range:iItem0Range+4])
		v[25], v[26] = spellID, spellRange
		v[27], v[34] = itemSpell, itemRange
		v[31], v[32], v[33] = in[iMainActionKind], in[iOffActionKind], in[iBattleActionKind]
		v[38], v[39], v[40] = in[iPartyX], in[iPartyY], in[iPartyZ]
		v[41], v[42], v[43] = in[iMonsterX], in[iMonsterY], in[iMonsterZ]
		v[44], v[45] = in[iMonsterRadius], in[iPartyRadius]
		copy(v[46:54], in[iCharacter0Quadrant:iCharacter0Quadrant+8])
		v[54] = in[iPartyHeadingDegrees]
		copy(v[55:67], in[iQuadrant0Slot0:iQuadrant0Slot0+12])
		copy(v[67:70], in[iCenterSlot0:iCenterSlot0+3])
		copy(v[70:78], in[iCharacter0FormationUnavailable:iCharacter0FormationUnavailable+8])
		return monsterTargetAttackRangeNode(v, r)
	case 4:
		// 0x537780 traverses group members; a missing caster range is zero distance.
		if dGrp == 456 && in[iGroupMemberCount] != 0 && round(monsterDistance()) <= round(length(attackRange())) {
			return yes
		}
		return []int64{0}
	case 6:
		// 0x53a8d0 derives the spell from the action source (0x4e77b0).
		sp := int64(0)
		switch category {
		case 7:
			sp = sourceSpell
		case 8:
			if !itemList {
				panic("target eligible: item spell needs the item list")
			}
			sp = in[iItem0Spell+int(itemIndex)]
		case 2:
			sp = 119
		}
		sk := kindOf(sp)
		if sk != 5 && sk != 6 && sk != 8 {
			panic("target eligible: coordinate target requires spell target kind 5, 6 or 8")
		}
		limit := t.unlimited
		dx, dy, dz := f(fX[sl])-f(iPartyX), f(fY[sl])-f(iPartyY), f(fZ[sl])-f(iPartyZ)
		if sk != 5 {
			limit = length(rangeOf(sp)) + t.tolerance
			dy = round(f(fY[sl])-t.eyeHeight) - round(f(iPartyY)-t.height) // 0x51938e..0x5193a6.
		}
		// Empty scene indices have no obstructing triangles; original
		// 0x434f20 still executes. No import or game function is replaced.
		if math.Sqrt(dx*dx+dy*dy+dz*dz)-f(iCoordinatePartyRadius) <= limit {
			return yes
		}
		return []int64{0}
	default:
		return yes // 0x51943c..0x5194b5.
	}
}

func init() {
	Nodes["00537220"] = Node{
		Addr: "00537220",
		In:   []string{"characterSlot", "spell", "attackContext", "characterPresent", "characterHP", "characterCondition", "characterInParty", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range", "item0Spell", "item1Spell", "item2Spell", "item3Spell", "mainActionKind", "offActionKind", "mainSpell", "offSpell", "queuedSpell", "spell0Range", "spell1Range", "spell2Range", "spell3Range", "spell4Range", "spell5Range", "spell6Range", "spell7Range", "spell8Range", "spell9Range", "itemEntriesPointer", "selectedItemIndex", "battlePresent", "targetKindCtx3", "targetKindCtx4", "targetKindCtx5", "targetKindCtx6", "characterTargetCtx3", "characterTargetCtx4", "characterTargetCtx5", "characterTargetCtx6", "monsterTargetCtx3", "monsterTargetCtx4", "monsterTargetCtx5", "monsterTargetCtx6", "groupTargetCtx3", "groupTargetCtx4", "groupTargetCtx5", "groupTargetCtx6", "targetXCtx3", "targetXCtx4", "targetXCtx5", "targetXCtx6", "targetYCtx3", "targetYCtx4", "targetYCtx5", "targetYCtx6", "targetZCtx3", "targetZCtx4", "targetZCtx5", "targetZCtx6", "targetListItem", "targetPresentCtx3", "targetPresentCtx4", "targetPresentCtx5", "targetPresentCtx6", "targetHPCtx3", "targetHPCtx4", "targetHPCtx5", "targetHPCtx6", "targetConditionCtx3", "targetConditionCtx4", "targetConditionCtx5", "targetConditionCtx6", "targetRecoverableCtx3", "targetRecoverableCtx4", "targetRecoverableCtx5", "targetRecoverableCtx6", "targetRecoveryBlockedCtx3", "targetRecoveryBlockedCtx4", "targetRecoveryBlockedCtx5", "targetRecoveryBlockedCtx6", "spell1Kind", "spell2Kind", "spell3Kind", "spell4Kind", "spell5Kind", "spell6Kind", "spell7Kind", "spell8Kind", "spell9Kind", "spell88Kind", "spell119Kind", "spell119Range", "monsterHP", "monsterRemoved", "monsterPresent", "monsterVisible", "monsterReachable", "groupActive", "groupSize", "groupMemberCount", "partyX", "partyY", "partyZ", "monsterX", "monsterY", "monsterZ", "monsterRadius", "partyRadius", "coordinatePartyRadius", "character0Quadrant", "character1Quadrant", "character2Quadrant", "character3Quadrant", "character4Quadrant", "character5Quadrant", "character6Quadrant", "character7Quadrant", "centerSlot0", "centerSlot1", "centerSlot2", "character0FormationUnavailable", "character1FormationUnavailable", "character2FormationUnavailable", "character3FormationUnavailable", "character4FormationUnavailable", "character5FormationUnavailable", "character6FormationUnavailable", "character7FormationUnavailable", "combatMode", "specialSpellMode", "specialItemMode", "combatPhase", "phaseType", "selectedCharacterSlot", "battleActionKind", "battleSpecialFlag", "specialBaseSpell", "specialOverrideSpell", "specialItemOverridePointer", "specialBaseItemIndex", "specialOverrideItemIndex", "monsterPhysicalAllowed", "partyHeadingDegrees", "quadrant0Slot0", "quadrant0Slot1", "quadrant0Slot2", "quadrant1Slot0", "quadrant1Slot1", "quadrant1Slot2", "quadrant2Slot0", "quadrant2Slot1", "quadrant2Slot2", "quadrant3Slot0", "quadrant3Slot1", "quadrant3Slot2", "targetKindOff", "characterTargetOff", "monsterTargetOff", "groupTargetOff", "targetXOff", "targetYOff", "targetZOff", "targetPresentOff", "targetHPOff", "targetConditionOff", "targetRecoverableOff", "targetRecoveryBlockedOff", "targetKindGlob", "characterTargetGlob", "monsterTargetGlob", "groupTargetGlob", "targetXGlob", "targetYGlob", "targetZGlob", "targetPresentGlob", "targetHPGlob", "targetConditionGlob", "targetRecoverableGlob", "targetRecoveryBlockedGlob"},
		Out:  []string{"eligible"}, F: actionTargetEligibleNode,
	}
}
