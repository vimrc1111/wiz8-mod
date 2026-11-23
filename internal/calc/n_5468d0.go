package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var defenseFatigueDispatch, defenseFatigueError = loadDefenseFatigueDispatch()

// Read only the dispatch destinations from the user's executable at startup.
func loadDefenseFatigueDispatch() ([5]uint32, error) {
	var table [5]uint32 // Five fatigue categories, exe 0x52a7d4.
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return table, fmt.Errorf("defense adjustment: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return table, err
	}
	off, err := peOffset(b, 0x52a800)
	if err != nil {
		return table, err
	}
	if len(b)-off < len(table)*4 {
		return table, fmt.Errorf("defense adjustment: truncated fatigue dispatch")
	}
	for i := range table {
		table[i] = binary.LittleEndian.Uint32(b[off+4*i:])
		switch table[i] {
		case 0x52a7e0, 0x52a7e3, 0x52a7e9, 0x52a7ef, 0x52a7f5:
		default:
			return table, fmt.Errorf("defense adjustment: unknown fatigue destination %#x", table[i])
		}
	}
	return table, nil
}

// Pure after executable table initialization. Runtime fields of the selected
// character, monster, template, battle and scene are explicit inputs.
func defenseAdjustmentNode(in []int64, r Rand) []int64 {
	if len(in) != 26 {
		panic("defense adjustment: expected 26 inputs")
	}
	for i, v := range in {
		lo, hi := int64(0), int64(4294967295)
		switch i {
		case 3, 4, 5, 8, 13, 14, 21:
			hi = 255
		case 6, 7:
			lo, hi = -2147483648, 2147483647
		case 9, 10:
			lo, hi = -128, 127
		}
		if v < lo || v > hi {
			panic("defense adjustment: storage width exceeded")
		}
	}
	if (in[0] != 1 && in[0] != 3) || in[2] > 7 || in[25] > 2 {
		panic("defense adjustment: invalid target or fixture record")
	}
	var adjustment int32
	defended, suppressed := false, false
	combat, ambushed := in[3] != 0, in[4] != 0 || in[5] != 0
	if in[0] == 3 {
		fatigue := uint32(in[11])
		if fatigue <= 4 {
			if defenseFatigueError != nil {
				panic(defenseFatigueError)
			}
			// Immediate results from exe 0x52a7e0..0x52a7fa; table remains external.
			switch defenseFatigueDispatch[fatigue] {
			case 0x52a7e0:
				fatigue = 0
			case 0x52a7e3:
				fatigue = 5
			case 0x52a7e9:
				fatigue = 10
			case 0x52a7ef:
				fatigue = 20
			case 0x52a7f5:
				fatigue = 40
			}
		}
		adjustment = int32(in[9]+in[10]) + int32(fatigue/10) // exe 0x54692a.
		if combat {
			if ambushed && in[14]&2 != 0 {
				adjustment -= 2
			} // exe 0x54695b.
			if in[12] == 1 {
				adjustment += 2
			} // exe 0x54696d.
			defended = in[12] == 4
		}
		suppressed = in[13] != 0
	} else {
		if combat {
			if ambushed {
				if in[6] > 0 {
					adjustment = -int32(in[6])
				}
				if in[7] > 0 {
					adjustment -= int32(in[7])
				}
			}
			posture := in[17]
			if in[15] != 0 {
				posture = in[16]
			} // Battle selection, exe 0x4ed2b0.
			// The fixed request is posture 6, apply=false. It never enters the
			// action eligibility child or setter (exe 0x4ed358, 0x4ed370).
			eligible := in[19] != 0 && uint32(in[20]) < 15
			if in[21] != 0 {
				eligible = eligible && in[22] == 6
			} else {
				eligible = eligible && in[23] == 6
			}
			if posture == 0 && eligible {
				adjustment -= 4
			} // exe 0x546a21.
			defended = posture == 2
		} else {
			defended = in[18]&256 != 0 // Scene defensive flag, exe 0x41efb0.
		}
		suppressed = in[8] != 0
	}
	// Unsigned attack mode comparisons; no defense penalty for modes 4,7,8.
	// exe 0x546989..0x54699c; arithmetic retains original 32-bit wrapping.
	if defended && !suppressed && in[1] != 4 && (in[1] <= 6 || in[1] > 8) {
		adjustment -= 4
	}
	return []int64{int64(adjustment)}
}

func init() {
	Nodes["005468d0"] = Node{Addr: "005468d0", In: []string{
		"targetType", "mode", "characterIndex", "combat", "ambush", "surprise",
		"speedDefense", "shieldDefense", "characterDefenseSuppressed", "monsterDefense",
		"monsterHitDefense", "fatigue", "monsterPosture", "monsterDefenseSuppressed",
		"ambushDefenseMask", "completedActionSelector", "completedPosture", "pendingPosture",
		"sceneFlags", "HP", "status", "actionCompleted", "mainAction", "offAction",
		"monsterID", "templateIndex",
	}, Out: []string{"adjustment"}, F: defenseAdjustmentNode}
}
