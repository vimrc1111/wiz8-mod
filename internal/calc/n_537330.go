package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// 0x501190 has eleven dispatch entries for the legal spell target kinds.
// Store semantic destinations after loading, rather than embedding exe data.
var itemSpellTargets, itemSpellTargetsErr = loadItemSpellTargets()

func loadItemSpellTargets() ([11]int64, error) {
	var targets [11]int64
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return targets, fmt.Errorf("item spell target: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return targets, err
	}
	offset, err := peOffset(b, 0x501190)
	if err != nil {
		return targets, err
	}
	if len(b)-offset < 4*len(targets) {
		return targets, fmt.Errorf("item spell target: truncated dispatch table")
	}
	for i := range targets {
		// Return categories and special cases: 0x501117..0x50118e.
		switch destination := binary.LittleEndian.Uint32(b[offset+4*i:]); destination {
		case 0x501117:
			targets[i] = -1 // Single ally, with spell 88's special category.
		case 0x501126:
			targets[i] = 8
		case 0x50112d:
			targets[i] = 2
		case 0x501134:
			targets[i] = 5
		case 0x50113b:
			targets[i] = 4
		case 0x501142:
			targets[i] = 3
		case 0x501149:
			targets[i] = -2 // Adventure target depends on battle presence.
		case 0x50118b:
			targets[i] = 0
		default:
			return targets, fmt.Errorf("item spell target: unknown destination %#x", destination)
		}
	}
	return targets, nil
}

// Pure after package initialization. Pointers and itemIndex select records;
// their contents are explicit inputs. Item/spell strides are 269/447
// (0x537340..0x537353, 0x4ac9da..0x4ac9e8). battle is only tested for zero;
// source category 6 (0x53735d) never dereferences a battle/character/monster.
func itemSpellTargetTypeNode(in []int64, r Rand) []int64 {
	if len(in) != 6 || in[0] < 0 || in[0] > 4294967295 ||
		in[1] < -2147483648 || in[1] > 2147483647 ||
		in[2] < 0 || in[2] > 255 || in[3] < 0 || in[3] > 255 ||
		in[4] < 0 || in[4] > 10 || in[5] < 0 || in[5] > 4294967295 {
		panic("item spell target: expected u32 instance, i32 index, u8 spell/type, target kind 0..10, u32 battle")
	}
	// Null instance, absent item (-1), no spell: 0x537334..0x53735b.
	if in[0] == 0 || in[1] == -1 || in[2] == 0 {
		return []int64{0}
	}
	if itemSpellTargetsErr != nil {
		panic(itemSpellTargetsErr)
	}
	kind := in[4]
	// Item kinds 16,21,22 supply a true low byte (0x5207c7..0x5207d8).
	// The consumer reads only that byte, ignoring the remaining EAX pointer bits.
	if kind == 1 && (in[3] == 16 || in[3] == 21 || in[3] == 22) {
		kind = 0 // 0x4ac9ef..0x4ac9fc: single ally becomes caster.
	}
	result := itemSpellTargets[kind]
	switch result {
	case -1:
		result = 1
		if in[2] == 88 { // 0x501119: this spell has category 7 instead of 1.
			result = 7
		}
	case -2:
		result = 0
		if in[5] == 0 { // 0x501149..0x501160; source category is fixed at 6.
			result = 6
		}
	}
	return []int64{result}
}

func init() {
	Nodes["00537330"] = Node{
		Addr: "00537330",
		In:   []string{"itemInstance", "itemIndex", "spellID", "itemType", "spellTarget", "battle"},
		Out:  []string{"targetType"}, F: itemSpellTargetTypeNode,
	}
}
