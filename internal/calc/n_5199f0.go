package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var actionRangeDispatch, actionRangeDispatchErr = loadActionRangeDispatch()

// Read dispatch data from the player's executable; F performs no IO.
func loadActionRangeDispatch() ([9]uint32, error) {
	var t [9]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("action attack range: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	o, err := peOffset(b, 0x519a9c)
	if err != nil {
		return t, err
	}
	if len(b)-o < 4*len(t) {
		return t, fmt.Errorf("action attack range: truncated dispatch")
	}
	for i := range t {
		t[i] = binary.LittleEndian.Uint32(b[o+4*i:])
		switch t[i] {
		case 0x519a3d, 0x519a4e, 0x519a74, 0x519a87, 0x519a8c, 0x519a94:
		default:
			return t, fmt.Errorf("action attack range: unknown dispatch destination %#x", t[i])
		}
	}
	return t, nil
}

// Inputs select one of eight party records, four independent items and ten
// synthetic spell records. Pointer inputs describe fixture list availability;
// they are never dereferenced by calc. No executable record data is embedded.
func actionAttackRangeNode(in []int64, r Rand) []int64 {
	if len(in) != 51 {
		panic("action attack range: expected 51 inputs")
	}
	for i, v := range in {
		lo, hi := int64(0), int64(4294967295)
		switch {
		case i == 0 || i == 43:
			hi = 7 // Party slots: 0x517ef4..0x517f02.
		case i == 1:
			hi = 2 // HAND_ANY: 0x519aca.
		case i == 2:
			hi = 8 // Context bound: 0x4e77e8.
		case i == 11 || i == 12 || i == 47 || i == 49 || i == 50:
			hi = 3
		case i >= 17 && i <= 20 || i >= 24 && i <= 29:
			hi = 9
		case i >= 30 && i <= 39:
			lo, hi = -2147483648, 2147483647
		case i == 3 || i == 6 || i == 8 || i == 10 || i >= 13 && i <= 16 || i >= 40 && i <= 42 || i == 45:
			hi = 255
		}
		if v < lo || v > hi {
			panic("action attack range: field outside fixture/storage domain")
		}
	}
	if in[6] == 0 {
		panic("action attack range: unavailable-hand diagnostic requires party membership")
	}
	if in[46] != 0 && in[46] != 0x20013498 {
		panic("action attack range: invalid item entries pointer")
	}
	if in[48] != 0 && in[48] != 0x20013560 {
		panic("action attack range: invalid special item override pointer")
	}
	if actionRangeDispatchErr != nil {
		panic(actionRangeDispatchErr)
	}
	context := in[2]
	// 0x53bc90..0x53bdb5 resolves context 6 from phase, battle action,
	// selected character and combat/special selection flags.
	if context == 6 {
		if in[44] == 7 && in[23] != 4294967295 {
			context = 7
			if in[45] != 0 && in[23] == 7 {
				context = 3
			}
			if in[45] != 0 && in[23] == 8 {
				context = 4
			}
		} else if in[0] == in[43] && (in[41] != 0 || in[42] != 0) {
			context = 2
		} else {
			context = 0
			if in[40] != 0 {
				context = 1
			}
		}
	}
	category, spell, item, hasItem := int64(0), int64(0), int64(0), false
	switch context { // Action sources: 0x4e77fa..0x4e78ef.
	case 0:
		category, spell, item, hasItem = in[21], in[24], in[47], in[46] != 0
	case 1:
		category, spell, item, hasItem = in[22], in[25], in[47], in[46] != 0
	case 2:
		if in[41] != 0 {
			category, spell = 7, in[29] // 0x5a1356: override spell, then base.
			if spell == 0 {
				spell = in[28]
			}
		} else {
			if in[42] == 0 {
				panic("action attack range: special context requires a selection mode")
			}
			category, item, hasItem = 8, in[49], true // 0x59e0d0: override item, then base.
			if in[48] != 0 {
				item = in[50]
			}
		}
	case 3:
		category, spell = 7, in[26]
	case 4:
		category, item, hasItem = 8, in[47], in[46] != 0
	case 5:
		category = 2
	case 7:
		category, spell = in[23], in[27]
	case 8:
		category = 0
	}
	// 0x53c283..0x53c2b7 tests presence, nonzero HP and unsigned
	// condition <18; 0x4e792c preserves action kinds 10 and 11 only.
	if (in[3] == 0 || in[4] == 0 || in[5] >= 18) && category != 10 && category != 11 {
		category = 4294967295
	}
	reach := int64(-1) // Invalid actions and spell ID zero: 0x519a94.
	if category <= 8 {
		switch actionRangeDispatch[category] {
		case 0x519a3d:
			rangeIn := append([]int64{in[1], in[6]}, in[7:17]...)
			reach = charAttackRangeNode(rangeIn, r)[0]
		case 0x519a4e:
			if spell != 0 {
				reach = in[30+spell]
			} // F+0x12f, stride 447: 0x519a56..0x519a6a.
		case 0x519a74:
			// Context 7 has no item-list object (0x4e78e0..0x4e78e2);
			// a kind-8 battle action there would dereference NULL at 0x519a78.
			if context == 7 {
				panic("action attack range: battle item action requires item context")
			}
			if hasItem {
				itemSpell := in[17+item] // I+0x63, stride 269: 0x520833..0x52084a.
				if itemSpell != 0 {
					reach = in[30+itemSpell]
				}
			}
		case 0x519a87:
			reach = 0 // Fixed short range, 0x519a87.
		case 0x519a8c:
			reach = 2 // Fixed long range, 0x519a8c.
		}
	}
	return []int64{reach}
}

func init() {
	Nodes["005199f0"] = Node{Addr: "005199f0", In: []string{
		"characterSlot", "hand", "attackContext", "characterPresent", "characterHP", "characterCondition", "characterInParty",
		"mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex",
		"item0Range", "item1Range", "item2Range", "item3Range", "item0Spell", "item1Spell", "item2Spell", "item3Spell",
		"mainActionKind", "offActionKind", "battleActionKind", "mainSpell", "offSpell", "queuedSpell", "battleSpell", "specialBaseSpell", "specialOverrideSpell",
		"spell0Range", "spell1Range", "spell2Range", "spell3Range", "spell4Range", "spell5Range", "spell6Range", "spell7Range", "spell8Range", "spell9Range",
		"combatMode", "specialSpellMode", "specialItemMode", "selectedCharacterSlot", "phaseType", "battleSpecialFlag", "itemEntriesPointer", "selectedItemIndex", "specialItemOverridePointer", "specialBaseItemIndex", "specialOverrideItemIndex",
	}, Out: []string{"attackRange"}, F: actionAttackRangeNode}
}
