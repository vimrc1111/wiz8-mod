package calc

import (
	"math"
	"math/big"
	"strconv"
)

func init() {
	Nodes["00545c20"] = Node{Addr: "00545c20", In: []string{
		"targetType",
		"targetCharacter",
		"monsterID",
		"monsterActive",
		"monsterHP",
		"monsterVisible",
		"monsterSpecialTemplate",
		"interceptRangeSelector",
		"characterSlot",
		"monsterTargetEnabled",
		"monsterPhysicalAllowed",
		"monsterSpellAllowed",
		"combatPhase",
		"combatMode",
		"specialSpellMode",
		"specialItemMode",
		"mainAttackKind",
		"mainAttackAvailable",
		"offAttackKind",
		"offAttackAvailable",
		"characterInParty",
		"mainItemIndex",
		"offItemIndex",
		"item0Range",
		"item1Range",
		"item2Range",
		"item3Range",
		"spellId",
		"spellRange",
		"item0Spell",
		"item1Spell",
		"item2Spell",
		"item3Spell",
		"mainActionKind",
		"offActionKind",
		"battleActionKind",
		"item0SpellRange",
		"item1SpellRange",
		"item2SpellRange",
		"item3SpellRange",
		"partyX",
		"partyY",
		"partyZ",
		"monsterX",
		"monsterY",
		"monsterZ",
		"monsterRadius",
		"partyRadius",
		"character0Quadrant",
		"character1Quadrant",
		"character2Quadrant",
		"character3Quadrant",
		"character4Quadrant",
		"character5Quadrant",
		"character6Quadrant",
		"character7Quadrant",
		"partyHeadingDegrees",
		"quadrant0Slot0",
		"quadrant0Slot1",
		"quadrant0Slot2",
		"quadrant1Slot0",
		"quadrant1Slot1",
		"quadrant1Slot2",
		"quadrant2Slot0",
		"quadrant2Slot1",
		"quadrant2Slot2",
		"quadrant3Slot0",
		"quadrant3Slot1",
		"quadrant3Slot2",
		"quadrant4Slot0",
		"quadrant4Slot1",
		"quadrant4Slot2",
		"character0FormationUnavailable",
		"character1FormationUnavailable",
		"character2FormationUnavailable",
		"character3FormationUnavailable",
		"character4FormationUnavailable",
		"character5FormationUnavailable",
		"character6FormationUnavailable",
		"character7FormationUnavailable",
		"slotActive0",
		"currentHP0",
		"mainStatus0",
		"slotActive1",
		"currentHP1",
		"mainStatus1",
		"slotActive2",
		"currentHP2",
		"mainStatus2",
		"slotActive3",
		"currentHP3",
		"mainStatus3",
		"slotActive4",
		"currentHP4",
		"mainStatus4",
		"slotActive5",
		"currentHP5",
		"mainStatus5",
		"slotActive6",
		"currentHP6",
		"mainStatus6",
		"slotActive7",
		"currentHP7",
		"mainStatus7",
		"mainActionSpell", "mainActionSpellRange", "offActionSpell", "offActionSpellRange",
	}, Out: []string{"interceptAvailable"}, F: interceptTargetAvailableNode}
}

// The case supplies eight shared party records, four items, and two loaded
// monster templates. IDs 0/1 resolve through the active/reserve lists using
// 0x4e5550/0x4e5620; template pointers are fixture wiring for 0x4e5720.
// Constants and field meanings follow 0x545c20..0x545ce0 and the named
// child nodes. Immutable geometry constants are loaded by the existing
// 0x5194e0 adapter from the user's executable via exe.go's PE mapping.
func interceptTargetAvailableNode(in []int64, r Rand) []int64 {
	names := Nodes["00545c20"].In
	if len(in) != len(names) {
		panic("intercept target: wrong input count")
	}
	fields := make(map[string]int64, len(in))
	for i, name := range names {
		fields[name] = in[i]
	}
	attacker, target := fields["characterSlot"], fields["targetCharacter"]
	if attacker < 0 || attacker > 7 || target < 0 || target > 7 || fields["monsterID"] < 0 || fields["monsterID"] > 1 {
		panic("intercept target: invalid fixture index")
	}
	no := []int64{0}
	switch fields["targetType"] {
	case 1: // 0x545c26: character target.
		alive := []int64{target}
		for i := 0; i < 8; i++ {
			suffix := strconv.Itoa(i)
			alive = append(alive, fields["slotActive"+suffix], fields["currentHP"+suffix], fields["mainStatus"+suffix])
		}
		if characterTargetableNode(alive, r)[0] == 0 {
			return no
		}
		formation := []int64{attacker, target}
		for i := 0; i < 8; i++ {
			formation = append(formation, fields["character"+strconv.Itoa(i)+"Quadrant"])
		}
		for i := 0; i < 3; i++ {
			formation = append(formation, fields["quadrant4Slot"+strconv.Itoa(i)])
		}
		for i := 0; i < 8; i++ {
			formation = append(formation, fields["character"+strconv.Itoa(i)+"FormationUnavailable"])
		}
		// 0x545c4c accepts only a zero formation penalty.
		if partyFormationRangePenaltyNode(formation, r)[0] != 0 {
			return no
		}
		return []int64{1}
	case 3: // 0x545c56: monster target.
		if fields["monsterActive"] == 0 || fields["monsterHP"] == 0 || fields["monsterVisible"] == 0 || fields["monsterSpecialTemplate"] != 0 {
			return no
		}
		args := make([]int64, len(Nodes["005194e0"].In))
		for i, name := range Nodes["005194e0"].In {
			args[i] = fields[name]
		}
		// 0x545cb9..0x545cca chooses context 1 for selector zero, otherwise
		// context 0; hand and feedback arguments are the literal zero.
		args[1], args[2], args[3] = 0, 0, 0
		if fields["interceptRangeSelector"] == 0 {
			args[2] = 1
		}
		suffix := strconv.Itoa(int(attacker))
		args[11], args[12], args[13] = fields["slotActive"+suffix], fields["currentHP"+suffix], fields["mainStatus"+suffix]
		// 0x4e77fa/0x4e780f return the selected action's own spell ID,
		// not the queued spell at Q+0x75. The existing range adapter assumes
		// those IDs share one record; keep the independent records here.
		if args[31+int(args[2])] == 7 {
			name := "mainActionSpell"
			if args[2] == 1 {
				name = "offActionSpell"
			}
			args[25], args[26] = fields[name], fields[name+"Range"]
		}
		return interceptMonsterAttackRange(args, r)
	default:
		return no
	}
}

// Local 0x5194e0 composition keeps the shared adapter unchanged while
// preserving the original extended-precision distance boundary.
func interceptMonsterAttackRange(in []int64, r Rand) []int64 {
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
	t := queuedSpellRuleData
	f := func(i int) float64 { return float64(math.Float32frombits(uint32(in[i]))) }
	// 0x4c7cf4 rounds adjusted height to float32. The remaining distance
	// operations (0x4c7cf8..0x4c7d35) and comparison (0x519720)
	// retain x87's 64-bit significand, including the square-root result.
	wide := func(v float64) *big.Float {
		return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).SetFloat64(v)
	}
	partyY := float64(float32(f(39) - t.height))
	dx := wide(0).Sub(wide(f(41)), wide(f(38)))
	dy := wide(0).Sub(wide(f(42)), wide(partyY))
	dz := wide(0).Sub(wide(f(43)), wide(f(40)))
	square := func(v *big.Float) *big.Float { return wide(0).Mul(v, v) }
	distance := wide(0).Add(square(dz), square(dy))
	distance.Add(distance, square(dx))
	distance.Sqrt(distance)
	distance.Sub(distance, wide(f(44)))
	distance.Sub(distance, wide(f(45)))
	if distance.Cmp(wide(t.minimum)) < 0 {
		distance = wide(t.minimum)
	}
	limit := wide(0).Mul(wide(units), wide(t.scale))
	if limit.Cmp(distance) >= 0 {
		return []int64{1}
	}
	return no
}
