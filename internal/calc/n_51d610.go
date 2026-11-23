package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var itemUseThresholds, itemUseThresholdErr = loadItemUseThresholds()

// Read the six spell-level dispatch destinations from the player's executable.
// The pure node below uses this immutable data; no executable bytes are stored.
func loadItemUseThresholds() ([6]uint32, error) {
	var t [6]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("item requirements: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	off, err := peOffset(b, 0x4acbf8)
	if err != nil {
		return t, err
	}
	if len(b)-off < 24 {
		return t, fmt.Errorf("item requirements: truncated dispatch")
	}
	for i := range t {
		// Return thresholds at 0x4acbcb..0x4acbee, selected by level minus 2.
		switch dest := binary.LittleEndian.Uint32(b[off+4*i:]); dest {
		case 0x4acbcb:
			t[i] = 3
		case 0x4acbd1:
			t[i] = 5
		case 0x4acbd7:
			t[i] = 8
		case 0x4acbdd:
			t[i] = 11
		case 0x4acbe3:
			t[i] = 14
		case 0x4acbe9:
			t[i] = 18
		default:
			return t, fmt.Errorf("item requirements: unknown dispatch %#x", dest)
		}
	}
	return t, nil
}

// Layout: item selector, character identity, item masks/requirements, kind/spell,
// instrument/device levels, learned state, spell level, seven attributes and
// 41 skills. Record strides 269/447 and character strides 20/38 are from
// 0x51d610..0x51d6d1 and 0x4acba0..0x4acbb8. Only AL is a defined result.
func itemUseRequirementsNode(in []int64, _ Rand) []int64 {
	if len(in) != 69 {
		panic("item requirements: expected 69 inputs")
	}
	for i, v := range in {
		max := int64(4294967295)
		switch i {
		case 0:
			max = 3
		case 4:
			max = 65535
		case 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16:
			max = 255
		}
		if v < 0 || v > max {
			panic("item requirements: storage width exceeded")
		}
	}
	for _, i := range []int{7, 9, 11, 13} {
		limit := int64(6)
		if i >= 11 {
			limit = 40
		}
		if in[i] != 255 && in[i] > limit {
			panic("item requirements: invalid requirement ID")
		}
	}
	no := []int64{0}
	// Shift counts use their low five bits, including the byte-sized gender mask.
	if uint32(in[4])&(uint32(1)<<(uint32(in[1])&31)) == 0 {
		return no
	}
	if uint32(in[5])&(uint32(1)<<(uint32(in[2])&31)) == 0 {
		return no
	}
	if in[6] != 3 && uint32(in[6])&(uint32(1)<<(uint32(in[3])&31)) == 0 {
		return no
	}
	for _, i := range []int{7, 9, 11, 13} {
		if in[i] == 255 {
			continue
		} // No requirement: 0x51d687/0x51d6b8.
		start := 21
		if i >= 11 {
			start = 28
		}
		if uint32(in[start+int(in[i])]) < uint32(in[i+1]) {
			return no
		}
	}
	// Item kinds 3/6/8: scroll/device/instrument (0x51d6e0..0x51d6f3).
	switch in[15] {
	case 3:
		if in[16] == 0 {
			panic("item requirements: scroll without spell")
		}
		if in[19] == 1 {
			return no
		} // Exactly 1, not any nonzero learned state.
	case 6, 8:
		if in[16] == 0 {
			panic("item requirements: spell item without spell")
		}
		if itemUseThresholdErr != nil {
			panic(itemUseThresholdErr)
		}
		threshold := uint32(1) // Default level branch 0x4acbef.
		level := uint32(in[20]) - 2
		if level <= 5 {
			threshold = itemUseThresholds[level]
		}
		index := 17
		if in[15] == 6 {
			index = 18
		}
		if uint32(in[index]) < threshold {
			return no
		}
	}
	return []int64{1}
}

func init() {
	Nodes["0051d610"] = Node{Addr: "0051d610", In: []string{"itemIndex", "profession", "race", "gender", "professionMask", "raceMask", "genderMask", "attributeId0", "attributeMinimum0", "attributeId1", "attributeMinimum1", "skillId0", "skillMinimum0", "skillId1", "skillMinimum1", "itemKind", "spellId", "instrumentLevel", "deviceLevel", "learnedState", "spellLevel", "strength", "intelligence", "piety", "vitality", "dexterity", "speed", "senses", "skill0", "skill1", "skill2", "skill3", "skill4", "skill5", "skill6", "skill7", "skill8", "skill9", "skill10", "skill11", "skill12", "skill13", "skill14", "skill15", "skill16", "skill17", "skill18", "skill19", "skill20", "skill21", "skill22", "skill23", "skill24", "skill25", "skill26", "skill27", "skill28", "skill29", "skill30", "skill31", "skill32", "skill33", "skill34", "skill35", "skill36", "skill37", "skill38", "skill39", "skill40"}, Out: []string{"eligible"}, F: itemUseRequirementsNode}
}
