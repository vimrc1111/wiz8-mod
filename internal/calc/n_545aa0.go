package calc

import "strconv"

func init() {
	Nodes["00545aa0"] = Node{Addr: "00545aa0", In: []string{
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
		"mainActionSpell",
		"mainActionSpellRange",
		"offActionSpell",
		"offActionSpellRange",
		"mainSpellTarget",
		"item0SpellTarget",
		"item0Type",
		"monsterUnavailable",
	}, Out: []string{"guardAttackAvailable"}, F: guardAttackAvailableNode}
}

// The actor's main action has context 0 and reachability hint 0 (0x545aa5).
// Records/geometry match 00545c20; target categories are read at runtime from
// the player's executable by loadCurrentTargetChecks, using exe.go's PE map.
func guardAttackAvailableNode(in []int64, r Rand) []int64 {
	// Avoid init ordering dependencies: obtain child names when F is called.
	names := Nodes["00545c20"].In
	if len(in) != len(names)+4 {
		panic("guard attack available: wrong input count")
	}
	fields := make(map[string]int64, len(in))
	for i, name := range names {
		fields[name] = in[i]
	}
	actor, target := fields["characterSlot"], fields["targetCharacter"]
	if actor < 0 || actor > 7 || target < 0 || target > 7 {
		panic("guard attack available: expected party slots 0..7")
	}
	suffix := strconv.Itoa(int(actor))
	active, hp, status := fields["slotActive"+suffix], fields["currentHP"+suffix], fields["mainStatus"+suffix]
	no := []int64{0}
	// 0x545ab7/0x545adb: action eligibility, then unsigned status <12.
	if characterActionEligibilityNode([]int64{actor, active, int64(uint32(hp)), status}, r)[0] == 0 ||
		status >= 12 || fields["mainAttackAvailable"] == 0 {
		return no
	}
	kind := fields["targetType"]
	// 0x545c26/0x545c59 accepts only character and monster targets. Other
	// fixture target types can never enable the combined rule.
	if kind != 1 && kind != 3 {
		return no
	}
	if currentTargetChecksErr != nil {
		panic(currentTargetChecksErr)
	}
	// Main item action wrapper points to the synthetic item-0 instance.
	// Battle object is present in this fixture (0x68edcc); context stays 0.
	// Instance pointer: cases/00545aa0.json objects.targets, arena base
	// 0x20000000 plus synthetic offset 172896; its definition index is 0.
	category := actionTargetTypeNode([]int64{fields["mainActionKind"], fields["mainActionSpell"],
		0x20000000 + 172896, 0, fields["item0Spell"], in[len(names)+2], in[len(names)], 1}, r)[0]
	if fields["mainActionKind"] == 8 {
		category = actionTargetTypeNode([]int64{8, 0, 0x20000000 + 172896, 0,
			fields["item0Spell"], in[len(names)+2], in[len(names)+1], 1}, r)[0]
	}
	if category >= 1 && category <= 8 {
		switch currentTargetChecks[category-1] {
		case 0x53717c: // 0x536170: living character or active monster.
			if kind == 3 && (fields["monsterHP"] == 0 || fields["monsterActive"] == 0 || in[len(names)+3] != 0) {
				return no
			}
		case 0x5371f4: // Unrestricted target category.
		default: // Dead characters, groups, items and locations cannot be 1/3.
			return no
		}
	}
	if kind == 1 && actor != target {
		reach := int64(-1)
		switch fields["mainActionKind"] {
		case 0, 1:
			args := []int64{0}
			for _, name := range Nodes["00519ac0"].In[1:] {
				args = append(args, fields[name])
			}
			reach = charAttackRangeNode(args, r)[0]
		case 2:
			reach = 2 // 0x519a8c.
		case 5:
			reach = 0 // 0x519a88.
		case 7:
			if fields["mainActionSpell"] != 0 {
				reach = fields["mainActionSpellRange"]
			}
		case 8:
			if fields["item0Spell"] != 0 {
				reach = fields["item0SpellRange"]
			}
		}
		if reach == -1 {
			return no
		}
		// Both the short-range check and final intercept gate require zero
		// party obstruction (0x519233, 0x545c4c); the child checks it below.
	}
	if kind == 3 {
		// 0x519180 uses the main action's range. Its template lookup does
		// not apply the intercept prohibition at template +0x24a.
		main := append([]int64(nil), in[:len(names)]...)
		for i, name := range names {
			if name == "interceptRangeSelector" {
				main[i] = 1 // 0x545cc5: context 0 for nonzero selector.
			}
			if name == "monsterSpecialTemplate" {
				main[i] = 0
			}
		}
		if interceptTargetAvailableNode(main, r)[0] == 0 {
			return no
		}
	}
	return interceptTargetAvailableNode(in[:len(names)], r)
}
