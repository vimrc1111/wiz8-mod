package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var currentTargetChecks, currentTargetChecksErr = loadCurrentTargetChecks()

// Target category dispatch is read from the player's exe at 0x5371f8.
// Destinations describe rules; the original table is never embedded.
func loadCurrentTargetChecks() ([8]uint32, error) {
	var checks [8]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return checks, fmt.Errorf("current target: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return checks, err
	}
	off, err := peOffset(b, 0x5371f8)
	if err != nil {
		return checks, err
	}
	if len(b)-off < 32 {
		return checks, fmt.Errorf("current target: truncated category dispatch")
	}
	for i := range checks {
		destination := binary.LittleEndian.Uint32(b[off+4*i:])
		switch destination {
		case 0x53717c, 0x5371a4, 0x5371b9, 0x5371cf, 0x5371e5, 0x5371f4:
			checks[i] = destination
		default:
			return checks, fmt.Errorf("current target: unknown category destination %#x", destination)
		}
	}
	return checks, nil
}

// Explicit fixture records: eight party slots, four indexed item definitions,
// 256 indexed spells, independent main/off-hand actions, and the special and
// battle contexts. Target records in the five queues share the supplied target
// contents. Monster/group collections are empty, so their IDs resolve missing.
// Party formation is fixed to the same row; equipped attacks are available,
// kind 0 (short range). Positional targets need a separate spatial fixture.
// Tables are immutable after package initialization; F only consumes inputs.
func characterCurrentTargetOkNode(in []int64, r Rand) []int64 {
	if len(in) != 29 {
		panic("current target: expected 29 inputs")
	}
	if in[0] < 0 || in[0] > 7 || in[1] < 0 || in[1] > 7 || in[21] < 0 || in[21] > 7 {
		panic("current target: invalid party slot or attack context")
	}
	for _, i := range []int{2, 3, 10, 11, 20, 26, 27, 28} {
		if in[i] < 0 || in[i] > 4294967295 {
			panic("current target: expected u32 record field")
		}
	}
	for _, i := range []int{5, 7, 8, 9, 12, 13, 16, 17, 24} {
		if in[i] < 0 || in[i] > 255 {
			panic("current target: expected u8 record field")
		}
	}
	for _, i := range []int{4, 6, 22, 23, 25} {
		if in[i] < -2147483648 || in[i] > 2147483647 {
			panic("current target: expected i32 record field")
		}
	}
	if in[14] != 0 && in[14] != 0x20010000 || in[15] < -1 || in[15] > 3 || in[18] < 0 || in[18] > 10 || in[19] < -2147483648 || in[19] > 2147483647 {
		panic("current target: invalid item instance, definition, or spell fields")
	}
	if currentTargetChecksErr != nil {
		panic(currentTargetChecksErr)
	}
	result := func(ok bool) []int64 {
		if ok {
			return []int64{1}
		}
		return []int64{0}
	}
	context := in[1]
	// Automatic context: 0x536f74..0x536ff0. Battle actions 7/8 with
	// the special flag select spell/item queues; other valid actions select 7.
	if context == 6 {
		if in[2] == 7 && in[3] != 0 && in[4] != -1 {
			context = 7
			if in[5] != 0 {
				if in[4] == 7 {
					context = 3
				}
				if in[4] == 8 {
					context = 4
				}
			}
		} else if in[0] == in[6] && (in[8] != 0 || in[9] != 0) {
			context = 2
		} else {
			context = 0
			if in[7] != 0 {
				context = 1
			}
		}
	}
	action, spell := int64(0), in[12]
	switch context {
	case 0:
		action = in[10]
	case 1:
		action, spell = in[11], in[13]
	case 2:
		if in[8] != 0 {
			action = 7
		} else if in[9] != 0 {
			action = 8
		} else {
			panic("current target: special queue requires spell/item mode")
		}
	case 3:
		action = 7
	case 4:
		action = 8
	case 5:
		action = 2
	case 7:
		action = in[4]
	}
	// 0x4e791b..0x4e793e suppresses all outputs for an invalid actor.
	// Other slots have the fixed active/HP/status values from the fixture.
	actorAlive := in[0] != in[21] || (in[24] != 0 && in[25] != 0 && in[26] < 18)
	if !actorAlive || context == 7 {
		return result(false)
	} // Battle queue has no target pointer (0x4e78e0).
	category := actionTargetTypeNode([]int64{action, spell, in[14], in[15], in[16], in[17], in[18], in[3]}, r)[0]
	kind := in[20]
	valid := true
	// 0x537160 requires a nonnull target pointer even for unrestricted categories.
	if category >= 1 && category <= 8 {
		switch currentTargetChecks[category-1] {
		case 0x53717c: // Living character or a monster.
			valid = (kind == 1 && in[24] != 0 && in[25] != 0 && in[26] < 18)
			// 0x536291: every monster ID is absent from these collections.
		case 0x5371a4: // Dead character: 0x536239..0x536286.
			valid = kind == 9 && in[24] != 0 && in[25] == 0 && in[27] != 0 && in[28] == 0
		case 0x5371b9: // Enemy group or whole party; groups are absent.
			valid = kind == 2
		case 0x5371cf: // Item instance; fixed target +0x1c points to instance.
			valid = kind == 7 && in[15] != -1
		case 0x5371e5: // Spatial location (0x5371e5).
			valid = kind == 6
		case 0x5371f4: // Unrestricted category.
		}
	}
	if !valid {
		return result(false)
	}
	// Reachability 0x519180: self succeeds, other party members require
	// a valid action range; the same formation row cannot block short reach.
	// All non-character, non-monster, non-group, non-position types succeed
	// at 0x51943f..0x5194b5. Absent monster/group lookups return false.
	switch kind {
	case 1:
		if in[0] == in[21] {
			return result(true)
		}
		reach := int64(-1)
		switch action {
		case 0, 1, 5:
			reach = 0 // Available fixed kind-0 equipped attacks, or ally action.
		case 2:
			reach = 2 // 0x519a8c.
		case 7:
			if spell != 0 {
				reach = in[19]
			} // Spell +0x12f (0x519a6a).
		case 8:
			if in[14] != 0 && in[15] != -1 && in[16] != 0 {
				reach = in[19]
			}
		}
		return result(reach != -1)
	case 3, 4:
		return result(false)
	case 6:
		panic("current target: positional target requires spatial records")
	default:
		return result(true)
	}
}

func init() {
	Nodes["00536f60"] = Node{Addr: "00536f60", In: []string{"characterSlot", "attackContext", "interfaceMode", "battle", "battleAction", "battleSpecial", "selectedCharacter", "combat", "specialUsesSpell", "specialUsesItem", "mainAction", "offAction", "mainSpell", "offSpell", "itemInstance", "itemIndex", "itemSpell", "itemType", "spellTarget", "spellRange", "targetKind", "targetCharacter", "targetMonster", "targetGroup", "targetPresent", "targetHP", "targetStatus", "targetStamina", "targetFatigue"}, Out: []string{"ok"}, F: characterCurrentTargetOkNode}
}
