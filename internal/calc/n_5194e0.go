package calc

import "math"

// The fixture supplies eight indexed party/action records, four independent
// items and five synthetic spell records. Queued/action spell IDs share one
// record; the four item spells have distinct IDs 2..5. Geometry and formation
// fields are independent inputs. Feedback uses the original duplicate-message
// mask; its return value does not affect AL (0x519770..0x51977b).
func monsterTargetAttackRangeNode(in []int64, r Rand) []int64 {
	if len(in) != 78 {
		panic("monster attack range: expected 78 inputs")
	}
	no := []int64{0}
	if in[4] != 1 { // M+0x28a, 0x5194e8.
		return no
	}
	if queuedSpellRuleErr != nil {
		panic(queuedSpellRuleErr)
	}
	reach, spellAttack := int64(0), true
	if in[7] != 0 || in[8] != 0 || in[9] != 0 || in[10] != 0 {
		category := int64(-1)
		if in[11] != 0 && in[12] != 0 && uint32(in[13]) < 18 { // 0x53c270.
			switch in[2] {
			case 0, 6: // context 6 resolves to main action with selected character -1.
				category = in[31]
			case 1:
				category = in[32]
			case 2:
				if in[9] != 0 {
					category = 7
				} else {
					category = 8
				}
			case 3:
				category = 7
			case 4:
				category = 8
			case 5:
				category = 2
			case 7:
				category = in[33]
			case 8:
				category = 0
			default:
				panic("monster attack range: invalid context")
			}
		}
		reach = -1
		switch category {
		case 0, 1:
			reach = charAttackRangeNode([]int64{in[1], in[18], in[14], in[15], in[16], in[17], in[19], in[20], in[21], in[22], in[23], in[24]}, r)[0]
		case 2:
			reach = 2 // 0x5195fc: fixed long range.
		case 5:
			reach = 0 // 0x5195f8: fixed short range.
		case 7:
			if in[25] != 0 {
				reach = in[26]
			} // F+0x12f, 0x5195db.
		case 8:
			// The case target list refers to item 0, whose embedded spell is
			// read by 0x5207e0; special mode uses that same item record.
			if in[27] != 0 {
				reach = in[34]
			}
		}
		if reach == -1 {
			return no
		} // 0x519606 rejects unavailable attacks.
		spellAttack = category == 3 || category > 5 && category <= 8
		if in[8] != 0 && reach >= 0 && reach <= 1 {
			formation := []int64{in[0]}
			formation = append(formation, in[46:54]...)
			formation = append(formation, in[41], in[42], in[43], in[38], in[39], in[40], in[54])
			formation = append(formation, in[55:]...)
			penalty := formationRangePenaltyNode(formation, r)[0]
			// 0x519665 checks zero before each decrement: exhausting
			// short reach fails immediately, never produces range -1.
			if penalty > reach {
				return no
			}
			reach -= penalty
		}
	}
	allowed := in[5]
	if spellAttack {
		allowed = in[6]
	} // M+0x28b/+0x28c, 0x519531.
	if allowed == 0 {
		return no
	}
	var units float64
	switch reach {
	case 0:
		units = 2
	case 1:
		units = 4
	case 2:
		units = 25
	case 3:
		units = 50
	default:
		panic("monster attack range: invalid range class")
	} // Literal lengths at 0x5196cf..0x5196e4; scale from exe 0x5ebc40.
	f := func(i int) float64 { return float64(math.Float32frombits(uint32(in[i]))) }
	d := monsterAttackTargetDistance([3]float64{f(41), f(42), f(43)}, [3]float64{f(38), f(39), f(40)}, true, f(44), f(45))
	if monsterAttackTargetWithin(int64(units), d) {
		return []int64{1}
	}
	return no
}

func init() {
	Nodes["005194e0"] = Node{Addr: "005194e0", In: []string{"characterSlot", "hand", "attackContext", "showFailure", "monsterTargetEnabled", "monsterPhysicalAllowed", "monsterSpellAllowed", "combatPhase", "combatMode", "specialSpellMode", "specialItemMode", "characterPresent", "characterHP", "characterCondition", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "characterInParty", "mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range", "spellId", "spellRange", "item0Spell", "item1Spell", "item2Spell", "item3Spell", "mainActionKind", "offActionKind", "battleActionKind", "item0SpellRange", "item1SpellRange", "item2SpellRange", "item3SpellRange", "partyX", "partyY", "partyZ", "monsterX", "monsterY", "monsterZ", "monsterRadius", "partyRadius", "character0Quadrant", "character1Quadrant", "character2Quadrant", "character3Quadrant", "character4Quadrant", "character5Quadrant", "character6Quadrant", "character7Quadrant", "partyHeadingDegrees", "quadrant0Slot0", "quadrant0Slot1", "quadrant0Slot2", "quadrant1Slot0", "quadrant1Slot1", "quadrant1Slot2", "quadrant2Slot0", "quadrant2Slot1", "quadrant2Slot2", "quadrant3Slot0", "quadrant3Slot1", "quadrant3Slot2", "quadrant4Slot0", "quadrant4Slot1", "quadrant4Slot2", "character0FormationUnavailable", "character1FormationUnavailable", "character2FormationUnavailable", "character3FormationUnavailable", "character4FormationUnavailable", "character5FormationUnavailable", "character6FormationUnavailable", "character7FormationUnavailable"}, Out: []string{"canAttack"}, F: monsterTargetAttackRangeNode}
}
