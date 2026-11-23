package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var handTypeDispatch, handTypeDispatchErr = loadHandTypeDispatch()

// Load the six item-type destinations at exe 0x5202f4 before calling the
// pure rule. PE translation is shared with exe.go; no table is embedded.
func loadHandTypeDispatch() ([6]uint32, error) {
	var targets [6]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return targets, fmt.Errorf("hand type: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return targets, err
	}
	off, err := peOffset(b, 0x5202f4)
	if err != nil {
		return targets, err
	}
	if len(b)-off < 4*len(targets) {
		return targets, fmt.Errorf("hand type: truncated dispatch")
	}
	for i := range targets {
		targets[i] = binary.LittleEndian.Uint32(b[off+4*i:])
	}
	return targets, nil
}

// Inputs contain four independent runtime item definitions, avoiding aliasing
// when both hands use the same ID. IDs index this fixture's four records;
// record stride 269 and type/subtype offsets 0x3e/0x3f come from 0x520221
// and 0x520268. The global item pointer merely locates these records.
func handTypeFromItemNode(in []int64, _ Rand) []int64 {
	if len(in) != 13 {
		panic("hand type: expected 13 inputs")
	}
	if in[0] != 6 && in[0] != 7 {
		panic("hand type: expected hand slot 6 or 7")
	} // 0x5201f7..0x520209.
	for _, id := range in[1:3] {
		if id < -1 || id > 3 {
			panic("hand type: item ID outside fixture")
		}
	}
	for i := 0; i < 4; i++ {
		if in[3+2*i] < 0 || in[3+2*i] > 255 || in[4+2*i] < -32768 || in[4+2*i] > 32767 {
			panic("hand type: expected u8 type and i16 subtype")
		}
	}
	for _, v := range in[11:] {
		if v < 0 || v > 4294967295 {
			panic("hand type: expected u32 prior hand type")
		}
	}
	out := append([]int64(nil), in[11:]...)
	hand := int(in[0] - 6)
	id := in[1+hand]
	value := int64(0) // Empty ID -1: exe 0x520218..0x52021f.
	if id != -1 {
		kind := in[3+2*id]
		if kind > 5 {
			panic("hand type: original asserts for invalid item type")
		} // 0x52023c.
		if handTypeDispatchErr != nil {
			panic(handTypeDispatchErr)
		}
		switch target := handTypeDispatch[kind]; target {
		case 0x520248:
			value = 1 // exe 0x520248.
		case 0x52024f:
			value = 1 // exe 0x520252.
			// Offhand checks the MAIN hand's subtype, not its own; 0x520259..0x52027e.
			if hand == 1 && in[1] != -1 && in[4+2*in[1]] == 131 {
				value = 3
			}
		case 0x52027e:
			value = 3 // exe 0x52027e.
		case 0x520285:
			value = 2 // exe 0x520285.
		default:
			panic(fmt.Sprintf("hand type: unknown dispatch destination %#x", target))
		}
	}
	out[hand] = value // Only selected hand is written; stride 91, exe 0x52028a..0x52029e.
	return out
}

func init() {
	Nodes["005201b0"] = Node{Addr: "005201b0",
		In:  []string{"slot", "mainItemID", "offItemID", "item0Type", "item0Subtype", "item1Type", "item1Subtype", "item2Type", "item2Subtype", "item3Type", "item3Subtype", "mainHandTypeBefore", "offHandTypeBefore"},
		Out: []string{"mainHandType", "offHandType"}, F: handTypeFromItemNode}
}
