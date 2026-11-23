package calc

import (
	"math"
	"strings"
)

// Node 0x53fea0: a monster spends one unit of its attack budget, checks that
// the chosen attack may start, and on success rolls its swing count.
// Domain restrictions (the original asserts or the fixture pins them):
//   - attack index 0..2, target slot 0..7; target kind 3 (monster victim) is
//     not covered: that target record is pinned to "none" in the fixture;
//   - range class byte of every attack 0..3, spell ranges -1..3, spell id
//     0..5, monster disposition 0..2, finite float positions;
//   - the other-monsters scan of the range check finds no eligible monster.
const (
	swSlotBase    = 27 // present, hp, status per party slot
	swCharm       = 346
	swPartyY      = 354
	swMonsterY    = 355
	swMonsterR    = 356
	swPartyR      = 357
	swQuadrant    = 358
	swQuadSlots   = 366
	swUnavailable = 381
	swAnimVariant = 389
	swScratch     = 404
)

func monsterAttackSwingsNode(in []int64, r Rand) []int64 {
	if len(in) != 407 {
		panic("monster attack swings: expected 407 inputs")
	}
	if queuedSpellRuleErr != nil {
		panic(queuedSpellRuleErr)
	}
	// 541630 layout: the 330 shared inputs with a zero continuation slot.
	full := append(append(append([]int64{}, in[:279]...), 0), in[279:330]...)
	slot, combat := in[0], in[6]
	targetType, monsterType, idx, mode := in[279], in[280], in[281], in[282]
	budget, swingRange := in[330], in[332]
	e := func(i int) int64 { return full[mfAttack+4*int(idx)+i] }
	if idx < 0 || idx > 2 {
		panic("monster attack swings: expected attack index 0..2")
	}
	if slot < 0 || slot > 7 {
		panic("monster attack swings: expected target slot 0..7")
	}
	spellRange := func() int64 {
		if monsterType < 0 || monsterType > 5 {
			panic("monster attack swings: expected spell id 0..5")
		}
		return int64(int32(full[mfSpellRange+int(monsterType)]))
	}
	// Outputs of a failed or refused attempt: nothing changes except the
	// scratch words, which are always cleared (head and tail).
	out := make([]int64, 40)
	out[1], out[2] = in[303], budget
	out[3], out[4], out[5], out[6] = targetType, slot, 0xffffffff, 0
	out[9], out[10] = in[278], in[310]
	copy(out[11:19], in[320:328])
	out[19], out[20], out[21], out[22], out[23] = e(3), in[390], 255, in[392], in[393]
	out[26] = in[395]
	copy(out[29:37], in[396:404])
	out[39] = in[406]
	if budget == 0 { // 0x53fec0: nothing left to spend, message only.
		return out
	}
	out[2] = int64(uint32(budget - 1))

	// 0x53d4b0: record checks, then the range check in mode 0.
	if e(0) == 0 || in[333+int(idx)] == 0 || in[331] == 0 || swingRange == 0 {
		return out
	}
	t := queuedSpellRuleData
	f := func(i int) float64 { return float64(math.Float32frombits(uint32(in[i]))) }
	dist := func() float64 {
		partyY := float64(float32(f(swPartyY) - t.height))
		dx, dy, dz := f(311)-f(315), f(swMonsterY)-partyY, f(312)-f(316)
		d := math.Sqrt(dz*dz+dy*dy+dx*dx) - f(swMonsterR) - f(swPartyR)
		if !(d >= t.minimum) {
			d = t.minimum
		}
		return d
	}
	penalty := func(c int64) int64 {
		v := []int64{c}
		v = append(v, in[swQuadrant:swQuadrant+8]...)
		v = append(v, in[311], in[swMonsterY], in[312], in[315], in[swPartyY], in[316], in[319])
		v = append(v, in[swQuadSlots:swQuadSlots+15]...)
		v = append(v, in[swUnavailable:swUnavailable+8]...)
		return formationRangePenaltyNode(v, r)[0]
	}
	lengths := [4]float64{2, 4, 25, 50}
	class := e(3)
	if class > 3 {
		panic("monster attack swings: range class above 3")
	}
	// 0x519c00 (mode 0): some eligible party slot is within the attack range.
	need := int64(1)
	if in[343] != 0 && in[345] != 0 {
		need = 2
	}
	kind := int64(0)
	if class == 2 || class == 3 {
		if in[341] != 0 {
			kind = 1
		}
	}
	inRange := false
	for s := int64(0); s < 8 && !inRange; s++ {
		b := swSlotBase + 3*int(s)
		if in[b] == 0 || in[b+1] == 0 || in[b+2] >= 0x12 {
			continue
		}
		if targetDispositionNode([]int64{s, in[swCharm+int(s)], in[344]}, r)[0] != need {
			continue
		}
		if in[336] != 1 || in[337+int(kind)] == 0 {
			continue
		}
		c := class
		if combat != 0 && c <= 1 {
			p := penalty(s)
			if p > c {
				continue
			}
			c -= p
		}
		if lengths[c]*t.scale >= dist() {
			inRange = true
		}
	}
	if !inRange {
		return out
	}

	// 0x537160 / 0x536190: the stored target must still be a living party member.
	valid := false
	if targetType == 1 {
		v := make([]int64, 29)
		v[0], v[1], v[2] = 1, 1, slot
		b := swSlotBase + 3*int(slot)
		v[3], v[4], v[5] = in[b], in[b+1], in[b+2]
		valid = combatTargetValidNode(v, r)[0] != 0
	}
	// 0x519f80 for a party target, using the stored mode.
	if valid {
		valid = false
		kind, class := int64(0), int64(-1)
		switch mode {
		case 0:
			class = e(3)
			if class == 2 || class == 3 {
				if in[341] != 0 {
					kind = 1
				}
			}
		case 2:
			kind = 2
			if in[342] != 0 {
				kind = 3
			}
			class = spellRange()
		case 3:
			kind, class = 2, 2
		case 8:
			class = 0
		}
		if in[336] == 1 && in[337+int(kind)] != 0 && class != -1 {
			if class < -1 || class > 3 {
				panic("monster attack swings: range class outside -1..3")
			}
			if combat != 0 && class <= 1 {
				p := penalty(slot)
				if p > class {
					class = -1
				} else {
					class -= p
				}
			}
			valid = class >= 0 && dist() <= float64(float32(lengths[class]*t.scale))
		}
	}
	if !valid { // 0x5369f0 clears the stored target and returns false.
		out[3], out[4], out[5], out[6] = 0, 0xffffffff, 0xffffffff, 0xffffffff
		out[37], out[38] = 0, 0
		return out
	}

	// Success: swing count first, then the attack flags and range normalisation.
	out[0] = 1
	out[1] = int64(r.R(int(swingRange))) + 1
	copy(out[7:19], monsterAttackFlagsNode(full, r))
	norm := []int64{monsterType, idx, e(3), in[swAnimVariant], mode, in[390], 0, 0, in[391], in[392], in[393], in[394], in[395]}
	norm[6] = int64(int32(full[mfSpellRange+int(monsterType)]))
	norm = append(norm, in[396:404]...)
	norm = append(norm, combat)
	copy(out[19:37], monsterAttackRangeNormalizeNode(norm, r))
	return out
}

func init() {
	Nodes["0053fea0"] = Node{Addr: "0053fea0", In: strings.Fields(`
		characterSlot interfaceMode battle battleAction battleSpecial selectedCharacter combat
		specialUsesSpell specialUsesItem mainAction offAction mainSpell offSpell itemInstance itemIndex
		itemSpell itemType spellTarget spellRange targetKind targetCharacter targetMonster targetGroup
		actionOverride otherMode0 otherMode1 otherMode2 present0 hp0 status0 present1 hp1 status1
		present2 hp2 status2 present3 hp3 status3 present4 hp4 status4 present5 hp5 status5 present6 hp6
		status6 present7 hp7 status7 profession race gender breathEffect maxStamina currentStamina
		mainAttackKind mainAttackAvailable offAttackKind offAttackAvailable mainItemIndex offItemIndex
		weaponCharges itemRange0 itemRange1 itemRange2 itemRange3 spellRealm spellCost spellLevel
		realmSP0 realmSP1 realmSP2 realmSP3 realmSP4 realmSP5 learned1 learned2 learned3 learned4
		learned5 learned6 learned7 learned8 learned9 learned10 learned11 learned12 learned13 learned14
		learned15 learned16 learned17 learned18 learned19 learned20 learned21 learned22 learned23
		learned24 learned25 learned26 learned27 learned28 learned29 learned30 learned31 learned32
		learned33 learned34 learned35 learned36 learned37 learned38 learned39 learned40 learned41
		learned42 learned43 learned44 learned45 learned46 learned47 learned48 learned49 learned50
		learned51 learned52 learned53 learned54 learned55 learned56 learned57 learned58 learned59
		learned60 learned61 learned62 learned63 learned64 learned65 learned66 learned67 learned68
		learned69 learned70 learned71 learned72 learned73 learned74 learned75 learned76 learned77
		learned78 learned79 learned80 learned81 learned82 learned83 learned84 learned85 learned86
		learned87 learned88 learned89 learned90 learned91 learned92 learned93 learned94 learned95
		learned96 learned97 learned98 learned99 learned100 learned101 learned102 learned103 learned104
		learned105 learned106 learned107 learned108 learned109 learned110 learned111 learned112
		learned113 professionMask raceMask genderMask attributeId0 attributeMinimum0 attributeId1
		attributeMinimum1 skillId0 skillMinimum0 skillId1 skillMinimum1 itemKind instrumentLevel
		deviceLevel strength intelligence piety vitality dexterity speed senses skill0 skill1 skill2
		skill3 skill4 skill5 skill6 skill7 skill8 skill9 skill10 skill11 skill12 skill13 skill14 skill15
		skill16 skill17 skill18 skill19 skill20 skill21 skill22 skill23 skill24 skill25 skill26 skill27
		skill28 skill29 skill30 skill31 skill32 skill33 skill34 skill35 skill36 skill37 skill38 skill39
		skill40 instanceCharges instanceCount gameStarted itemFlags weaponSubtype weaponRechargeType
		itemRechargeType targetStamina targetFatigue actionCompleted deferredDefense refreshMask
		mainTargetTail0 mainTargetTail1 mainTargetTail2 mainTargetTail3 offTargetTail0 offTargetTail1
		offTargetTail2 offTargetTail3 mainParameter mainTargetHead0 mainTargetHead1 mainTargetHead2
		mainTargetHead3 interferenceStatus interceptAttempts targetType monsterType attackIndex
		rangeType templateClass templateDisturb attackE0k0 attackE1k0 attackE2k0 attackE3k0 attackE0k1
		attackE1k1 attackE2k1 attackE3k1 attackE0k2 attackE1k2 attackE2k2 attackE3k2 spellRange0
		spellRange1 spellRange2 spellRange3 spellRange4 spellRange5 swingCount victimAlive victimAwake
		victimLevel victimStun victimTurnable victimTurns victimAttempts attackerX attackerZ victimX
		victimZ partyX partyZ attackerFacing victimFacing partyHeading face0 face1 face2 face3 face4
		face5 face6 face7 notifyKind verboseMode budget gateE5 swingRange attackWord0 attackWord1
		attackWord2 reachEnable reach0 reach1 reach2 reach3 altFlag altFlag2 monsterFlag15
		monsterDisposition combatFlag15 charm0 charm1 charm2 charm3 charm4 charm5 charm6 charm7 partyY
		monsterY monsterRadius partyRadius character0Quadrant character1Quadrant character2Quadrant
		character3Quadrant character4Quadrant character5Quadrant character6Quadrant character7Quadrant
		quadrant0Slot0 quadrant0Slot1 quadrant0Slot2 quadrant1Slot0 quadrant1Slot1 quadrant1Slot2
		quadrant2Slot0 quadrant2Slot1 quadrant2Slot2 quadrant3Slot0 quadrant3Slot1 quadrant3Slot2
		quadrant4Slot0 quadrant4Slot1 quadrant4Slot2 character0FormationUnavailable
		character1FormationUnavailable character2FormationUnavailable character3FormationUnavailable
		character4FormationUnavailable character5FormationUnavailable character6FormationUnavailable
		character7FormationUnavailable animationVariants rangedState sceneFlags animationFrame
		animationStart animationTimeBits animationCounter targetWord0 targetWord1 targetWord2
		targetWord3 targetWord4 targetWord5 targetWord6 targetWord7 scratchHead scratchTail
		scratchBeyond`), Out: strings.Fields(`
		ret swingsOut budgetOut mTarget0Out mTarget1Out mTarget2Out mTarget3Out rearFlag specialFlag
		interceptAttemptsOut victimAttemptsOut face0Out face1Out face2Out face3Out face4Out face5Out
		face6Out face7Out rangeAfter rangedStateAfter animationAfter animationFrameAfter
		animationStartAfter animationPending animationPolicy animationCounterAfter
		animationAccessCounter animationTimeAfter targetWordAfter0 targetWordAfter1 targetWordAfter2
		targetWordAfter3 targetWordAfter4 targetWordAfter5 targetWordAfter6 targetWordAfter7
		scratchHeadAfter scratchTailAfter scratchBeyondAfter`), F: monsterAttackSwingsNode}
}
