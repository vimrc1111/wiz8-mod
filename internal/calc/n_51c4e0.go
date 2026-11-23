package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var defaultEquipDispatch, defaultEquipLoadErr = loadDefaultEquipDispatch()

func loadDefaultEquipDispatch() ([13]uint32, error) {
	// Thirteen type entries (0..12): 0x51c4fd; dispatch table: 0x51c568.
	var targets [13]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return targets, fmt.Errorf("default equipment slot: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return targets, err
	}
	off, err := peOffset(b, 0x51c568)
	if err != nil {
		return targets, err
	}
	if len(b)-off < 4*len(targets) {
		return targets, fmt.Errorf("default equipment slot: truncated dispatch table")
	}
	for i := range targets {
		targets[i] = binary.LittleEndian.Uint32(b[off+4*i:])
	}
	return targets, nil
}

// Pure after exe initialization. itemDefinitionIndex selects a loaded record
// at 0x68516c with stride 269 (notes/re/loot.md:9); itemType is its +0x3e byte
// (0x51c4f9). The index affects addressing only. gameStarted is the entire
// byte at 0x68517c, with every nonzero value selecting the started-game rule.
func itemDefaultEquipSlotNode(in []int64, _ Rand) []int64 {
	if len(in) != 3 || in[0] < 0 || in[0] > 4294967295 || in[1] < 0 || in[1] > 255 || in[2] < 0 || in[2] > 255 {
		panic("default equipment slot: expected u32 definition index and u8 type/gameStarted")
	}
	if in[1] > 12 {
		return []int64{-1} // Unsupported type: 0x51c4fd, 0x51c562.
	}
	if defaultEquipLoadErr != nil {
		panic(defaultEquipLoadErr)
	}
	var slot int64
	// Each target address is the source of its slot constant; the type-to-target
	// mapping is supplied by the user's exe rather than duplicated here.
	switch target := defaultEquipDispatch[in[1]]; target {
	case 0x51c509:
		slot = 6
	case 0x51c50f:
		slot = 6
		if in[2] != 0 {
			slot += 2 // Started-game adjustment: 0x51c518.
		}
	case 0x51c51f:
		slot = 7
		if in[2] != 0 {
			slot += 2 // Started-game adjustment: 0x51c528.
		}
	case 0x51c52f:
		slot = 8
	case 0x51c535:
		slot = 1
	case 0x51c53b:
		slot = 0
	case 0x51c53e:
		slot = 4
	case 0x51c544:
		slot = 10
	case 0x51c54a:
		slot = 5
	case 0x51c550:
		slot = 11
	case 0x51c556:
		slot = 3
	case 0x51c55c:
		slot = 7
	default:
		panic(fmt.Sprintf("default equipment slot: unknown dispatch target %#x", target))
	}
	return []int64{slot}
}

func init() {
	Nodes["0051c4e0"] = Node{
		Addr: "0051c4e0", In: []string{"itemDefinitionIndex", "itemType", "gameStarted"},
		Out: []string{"defaultEquipSlot"}, F: itemDefaultEquipSlotNode,
	}
}
