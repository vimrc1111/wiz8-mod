package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var pcActionDispatch, pcActionDispatchErr = loadPCActionDispatch()

// The action dispatch at exe 0x4e7c78 is read from the player's executable.
// Class/race features and item dispatches reuse the existing immutable loaders.
func loadPCActionDispatch() ([9]uint32, error) {
	var t [9]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("pc action eligibility: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	off, err := peOffset(b, 0x4e7c78)
	if err != nil {
		return t, err
	}
	if len(b)-off < 36 {
		return t, fmt.Errorf("pc action eligibility: truncated dispatch")
	}
	for i := range t {
		t[i] = binary.LittleEndian.Uint32(b[off+4*i:])
	}
	return t, nil
}

// Fixture: eight independent party records, four items, 113 learned spells,
// six realm SP pools, and the UI/battle action queues. All positions use the
// same formation row; monster/group lists are empty. Spell cost/realm/level
// are shared synthetic definition fields, not values copied from the exe.
// Weapon repair calls run in the original; these fixtures have no compatible
// ammunition or refill items, so they do not rebuild the hand attack records.
func pcActionEligibilityNode(in []int64, r Rand) []int64 {
	if len(in) != 264 {
		panic("pc action eligibility: expected 264 inputs")
	}
	if in[0] < 0 || in[0] > 7 || in[1] < 0 || in[1] > 8 || in[1] == 2 && in[8] == 0 && in[9] == 0 || in[15] < -1 || in[15] > 3 || in[21] < 0 || in[21] > 7 || in[54] < -1 || in[54] > 14 || in[55] < -1 || in[55] > 15 || in[71] < 0 || in[71] > 5 {
		panic("pc action eligibility: invalid record selector")
	}
	for i, v := range in {
		width := pcActionInputWidths[i]
		bits := uint(absPCActionWidth(width) * 8)
		lo, hi := int64(0), int64((uint64(1)<<bits)-1)
		if width < 0 {
			lo = -(int64(1) << (bits - 1))
			hi = (int64(1) << (bits - 1)) - 1
		}
		if v < lo || v > hi {
			panic("pc action eligibility: input storage width exceeded")
		}
	}
	if pcActionDispatchErr != nil {
		panic(pcActionDispatchErr)
	}
	yes, no := []int64{1}, []int64{0}
	slot := int(in[0])
	record := func(i int) []int64 { return []int64{int64(i), in[30+3*i], in[31+3*i], in[32+3*i]} }
	targetable := func(i int) bool { a := record(i); return a[1] != 0 && a[2] != 0 && a[3] < 18 } // 0x53c283..0x53c2ab.
	// The global free-action gate (0x561440) reads six distinct UI mode bytes.
	free := in[7] == 0 && in[8] == 0 && in[9] == 0 && in[27] == 0 && in[28] == 0 && in[29] == 0
	if in[26] == 0 && free {
		for i := 0; i < 8; i++ {
			if characterActionEligibilityNode(record(i), r)[0] != 0 {
				return yes
			}
		}
		return no
	}
	if !targetable(slot) || in[32+3*slot] >= 14 {
		return no
	} // 0x4e7a0f.
	context := in[1]
	if context == 6 {
		context = pcActionContext(in)
	}
	action, spell, wrapper := pcActionRecord(in, context)
	if action == -1 {
		return no
	}
	if action >= 0 && action <= 8 {
		switch dest := pcActionDispatch[action]; dest {
		case 0x4e7a7a: // Ordinary hand attack: 0x53d310.
			if in[61] == 0 && in[63] == 0 {
				return no
			}
		case 0x4e7af8: // Berserk feature 20, and main-hand range <= 1.
			if !pcActionFeature(in, 20) || in[61] == 0 && in[63] == 0 {
				return no
			}
			if pcActionHandRange(in, 0) > 1 {
				return no
			}
		case 0x4e7b76:
			if guardAvailableNode(append(record(slot), in[61]), r)[0] == 0 {
				return no
			}
		case 0x4e7c04:
			if breathStaminaEligibilityNode([]int64{in[0], in[54], in[55], in[57], in[58], in[59]}, r)[0] == 0 {
				return no
			}
		case 0x4e7b8e:
			// Automatic battle spells allow any learned affordable spell (0x4f96f0).
			if context == 7 {
				for id := int64(1); id < 114; id++ {
					if pcActionSpell(in, id) {
						return yes
					}
				}
			}
			if !pcActionSpell(in, spell) {
				return no
			}
		case 0x4e7bcc:
			if context == 7 && !wrapper {
				return yes
			}
			if !wrapper || in[14] == 0 || !pcActionItem(in, r) {
				return no
			}
		case 0x4e7c6e:
			return yes // Action 4 bypasses target validation.
		case 0x4e7c44: // Actions 3 and 6 have only the common target gate.
		default:
			panic(fmt.Sprintf("pc action eligibility: unknown action destination %#x", dest))
		}
	}
	if in[24] != 0 {
		return yes
	}
	if pcActionTarget(in, context, r) {
		return yes
	}
	return no
}

func pcActionContext(in []int64) int64 {
	// 0x53bc90: battle mode 7 takes priority, then the selected special queue,
	// then the combat byte. No data table supplies these predicates.
	if in[2] == 7 && in[3] != 0 && in[4] != -1 {
		if in[5] != 0 {
			if in[4] == 7 {
				return 3
			}
			if in[4] == 8 {
				return 4
			}
		}
		return 7
	}
	if in[0] == in[6] && (in[8] != 0 || in[9] != 0) {
		return 2
	}
	if in[7] != 0 {
		return 1
	}
	return 0
}

func pcActionRecord(in []int64, context int64) (action, spell int64, wrapper bool) {
	// Source queues: 0x4e77fa..0x4e78ef. Battle and empty context have no
	// item wrapper or target pointer; the other queues are arena records.
	switch context {
	case 0:
		return in[10], in[12], true
	case 1:
		return in[11], in[13], true
	case 2:
		if in[8] != 0 {
			return 7, in[12], true
		}
		return 8, -1, true
	case 3:
		return 7, in[12], true
	case 4:
		return 8, -1, true
	case 5:
		return 2, -1, false
	case 7:
		return in[4], 0, false // Battle queue spell parameter is fixed to zero in the fixture.
	case 8:
		return 0, -1, false
	}
	panic("pc action eligibility: invalid context")
}

func pcActionFeature(in []int64, feature int32) bool {
	if breathTablesErr != nil {
		panic(breathTablesErr)
	}
	if in[54] != -1 {
		for _, f := range breathTables.class[in[54]] {
			if f == feature {
				return true
			}
		}
	}
	if in[55] != -1 {
		for _, f := range breathTables.race[in[55]] {
			if f == feature {
				return true
			}
		}
	}
	return feature == 28 && in[57] != 0 // 0x54798f..0x54799c.
}

func pcActionHandRange(in []int64, hand int) int64 {
	if in[61+2*hand] == 0 {
		return -1
	}
	kind := in[60+2*hand]
	if kind != 1 && kind != 3 {
		return 0
	}
	index := in[64+hand]
	if index < 0 || index > 3 {
		panic("pc action eligibility: enabled weapon requires item definition")
	}
	return in[67+index]
}

func pcActionSpell(in []int64, id int64) bool {
	if id == 0 || id == -1 {
		return false
	}
	if id < 1 || id > 113 {
		panic("pc action eligibility: spell outside loaded definitions")
	}
	// Exactly learned-state 1; signed SP/cost compare, 0x4f975c..0x4f978b.
	return in[79+id] == 1 && in[74+in[71]] >= in[72]
}

func pcActionItem(in []int64, r Rand) bool {
	if in[15] == -1 {
		return false
	}
	child := []int64{in[15], int64(uint32(in[54])), int64(uint32(in[55])), in[56]}
	child = append(child, in[193:205]...)
	child = append(child, in[16], in[205], in[206])
	learned := int64(0)
	if in[16] > 0 && in[16] <= 113 {
		learned = in[79+in[16]]
	}
	child = append(child, learned, in[73])
	child = append(child, in[207:255]...)
	if itemUseRequirementsNode(child, r)[0] == 0 {
		return false
	}
	// Equippable/flagged items must occupy a gear slot other than hand 8/9.
	// This fixture's external instance is not in the character's gear array.
	if in[258]&64 != 0 || itemDefaultEquipSlotNode([]int64{in[15], in[17], in[257]}, r)[0] != -1 {
		return false
	}
	// 0x51d912..0x51d947: kind 0/3 forbidden, spell required; charged
	// items of recharge kind 2 require a nonzero remaining charge byte.
	return in[204] != 0 && in[204] != 3 && in[16] != 0 && (in[261] != 2 || in[255] != 0)
}

func pcActionTarget(in []int64, context int64, r Rand) bool {
	if context == 7 || context == 8 {
		return false
	}
	// 0x53ba20 converts the source to the same context consumed by the
	// current-target rule. The six supplied queues share target contents.
	action, spell, _ := pcActionRecord(in, context)
	spell = int64(uint8(spell))
	cat := actionTargetTypeNode([]int64{action, spell, in[14], in[15], in[16], in[17], in[18], in[3]}, r)[0]
	if currentTargetChecksErr != nil {
		panic(currentTargetChecksErr)
	}
	kind, target := in[20], int(in[21])
	valid := true
	if cat >= 1 && cat <= 8 {
		switch currentTargetChecks[cat-1] {
		case 0x53717c:
			valid = kind == 1 && in[30+3*target] != 0 && in[31+3*target] != 0 && in[32+3*target] < 18
		case 0x5371a4:
			valid = kind == 9 && in[30+3*target] != 0 && in[31+3*target] == 0 && in[262] != 0 && in[263] == 0 // 0x536269..0x53627b.
		case 0x5371b9:
			valid = kind == 2
		case 0x5371cf:
			valid = kind == 7 && in[15] != -1
		case 0x5371e5:
			valid = kind == 6
		case 0x5371f4:
		}
	}
	if !valid {
		return false
	}
	switch kind {
	case 1:
		if int(in[0]) == target {
			return true
		}
		reach := int64(-1)
		switch action {
		case 0, 1:
			reach = max(pcActionHandRange(in, 0), pcActionHandRange(in, 1))
		case 5:
			reach = 0
		case 2:
			reach = 2
		case 7:
			if spell != 0 {
				reach = in[19]
			}
		case 8:
			if in[14] != 0 && in[15] != -1 && in[16] != 0 {
				reach = in[19]
			}
		}
		return reach != -1
	case 3, 4:
		return false
	case 6:
		panic("pc action eligibility: spatial target requires scene records")
	default:
		return true
	}
}

func absPCActionWidth(w int8) int8 {
	if w < 0 {
		return -w
	}
	return w
}

// Case field storage widths, in the public input order; negative denotes signed.
var pcActionInputWidths = [...]int8{
	4, 4, 4, 4, -4, 1, -4, 1, 1, 1, -4, -4, 4, 4, 4, -4, 1, 1, -4, -4, 4, -4, -4, -4,
	1, 1, 4, 1, 1, 1, 1, 4, 4, 1, 4, 4, 1, 4, 4, 1, 4, 4, 1, 4, 4, 1, 4, 4,
	1, 4, 4, 1, 4, 4, -4, -4, 4, 4, 4, -4, 4, 1, 4, 1, -4, -4, 1, 1, 1, 1, 1, 4,
	-4, 1, -4, -4, -4, -4, -4, -4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 2, 4, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 1, 1, 1, 1, 2, 1, 1, 4, 4,
}

func init() {
	Nodes["004e79a0"] = Node{Addr: "004e79a0", In: []string{
		"characterSlot", "attackContext", "interfaceMode", "battle", "battleAction",
		"battleSpecial", "selectedCharacter", "combat", "specialUsesSpell", "specialUsesItem",
		"mainAction", "offAction", "mainSpell", "offSpell", "itemInstance",
		"itemIndex", "itemSpell", "itemType", "spellTarget", "spellRange",
		"targetKind", "targetCharacter", "targetMonster", "targetGroup", "ignoreTarget",
		"prepareWeapon", "actionOverride", "otherMode0", "otherMode1", "otherMode2",
		"present0", "hp0", "status0", "present1", "hp1",
		"status1", "present2", "hp2", "status2", "present3",
		"hp3", "status3", "present4", "hp4", "status4",
		"present5", "hp5", "status5", "present6", "hp6",
		"status6", "present7", "hp7", "status7", "profession",
		"race", "gender", "breathEffect", "maxStamina", "currentStamina",
		"mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex",
		"offItemIndex", "weaponCharges", "itemRange0", "itemRange1", "itemRange2",
		"itemRange3", "spellRealm", "spellCost", "spellLevel", "realmSP0",
		"realmSP1", "realmSP2", "realmSP3", "realmSP4", "realmSP5",
		"learned1", "learned2", "learned3", "learned4", "learned5",
		"learned6", "learned7", "learned8", "learned9", "learned10",
		"learned11", "learned12", "learned13", "learned14", "learned15",
		"learned16", "learned17", "learned18", "learned19", "learned20",
		"learned21", "learned22", "learned23", "learned24", "learned25",
		"learned26", "learned27", "learned28", "learned29", "learned30",
		"learned31", "learned32", "learned33", "learned34", "learned35",
		"learned36", "learned37", "learned38", "learned39", "learned40",
		"learned41", "learned42", "learned43", "learned44", "learned45",
		"learned46", "learned47", "learned48", "learned49", "learned50",
		"learned51", "learned52", "learned53", "learned54", "learned55",
		"learned56", "learned57", "learned58", "learned59", "learned60",
		"learned61", "learned62", "learned63", "learned64", "learned65",
		"learned66", "learned67", "learned68", "learned69", "learned70",
		"learned71", "learned72", "learned73", "learned74", "learned75",
		"learned76", "learned77", "learned78", "learned79", "learned80",
		"learned81", "learned82", "learned83", "learned84", "learned85",
		"learned86", "learned87", "learned88", "learned89", "learned90",
		"learned91", "learned92", "learned93", "learned94", "learned95",
		"learned96", "learned97", "learned98", "learned99", "learned100",
		"learned101", "learned102", "learned103", "learned104", "learned105",
		"learned106", "learned107", "learned108", "learned109", "learned110",
		"learned111", "learned112", "learned113", "professionMask", "raceMask",
		"genderMask", "attributeId0", "attributeMinimum0", "attributeId1", "attributeMinimum1",
		"skillId0", "skillMinimum0", "skillId1", "skillMinimum1", "itemKind",
		"instrumentLevel", "deviceLevel", "strength", "intelligence", "piety",
		"vitality", "dexterity", "speed", "senses", "skill0",
		"skill1", "skill2", "skill3", "skill4", "skill5",
		"skill6", "skill7", "skill8", "skill9", "skill10",
		"skill11", "skill12", "skill13", "skill14", "skill15",
		"skill16", "skill17", "skill18", "skill19", "skill20",
		"skill21", "skill22", "skill23", "skill24", "skill25",
		"skill26", "skill27", "skill28", "skill29", "skill30",
		"skill31", "skill32", "skill33", "skill34", "skill35",
		"skill36", "skill37", "skill38", "skill39", "skill40",
		"instanceCharges", "instanceCount", "gameStarted", "itemFlags", "weaponSubtype",
		"weaponRechargeType", "itemRechargeType", "targetStamina", "targetFatigue",
	}, Out: []string{"eligible"}, F: pcActionEligibilityNode}
}
