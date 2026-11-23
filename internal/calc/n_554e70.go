package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var restoreFacing, restoreFacingError = loadRestoreFacing()

// The region-to-facing dispatch is loaded from the user's executable, not embedded.
func loadRestoreFacing() ([5]byte, error) {
	var facing [5]byte
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return facing, fmt.Errorf("formation restore: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return facing, err
	}
	off, err := peOffset(b, 0x554db8)
	if err != nil {
		return facing, err
	}
	if off > len(b)-20 {
		return facing, fmt.Errorf("formation restore: truncated facing dispatch")
	}
	for q := range facing {
		// Assignment destinations at 0x554cd4..0x554ce6 name the four facings.
		switch binary.LittleEndian.Uint32(b[off+4*q:]) {
		case 0x554cd4:
			facing[q] = 0
		case 0x554cda:
			facing[q] = 1
		case 0x554ce0:
			facing[q] = 2
		case 0x554ce6:
			facing[q] = 3
		case 0x554cea:
			facing[q] = 255 // dispatch destination that skips assignment
		default:
			return facing, fmt.Errorf("formation restore: unknown facing dispatch")
		}
	}
	return facing, nil
}

func init() {
	names := []string{"character", "mainStatus"}
	for q := 0; q < 5; q++ {
		for s := 0; s < 3; s++ {
			names = append(names, fmt.Sprintf("region%dSlot%d", q, s))
		}
	}
	for q := 0; q < 5; q++ {
		names = append(names, fmt.Sprintf("region%dCount", q))
	}
	for c := 0; c < 8; c++ {
		for _, field := range []string{"Region", "PreviousRegion", "Slot", "Facing"} {
			names = append(names, fmt.Sprintf("character%d%s", c, field))
		}
	}
	Nodes["00554e70"] = Node{Addr: "00554e70", In: names, Out: append([]string(nil), names[2:]...), F: formationRestoreAfterStatusNode}
}

// Layout and constants: 0x554e82..0x554e98 (12-byte formation records,
// character stride 0x1862, main status +0xb01); 0x554f99..0x554fa3
// (fallback regions 0,4,2); 0x554c8f (three slots), 0x554ea4 (empty 255).
// The scene notification gate 0x68517c is disabled in the replay: notifications
// do not change the formation rule's records. All mutable formation fields read
// by 0x554bd0 and its compaction helper 0x554dd0 are explicit inputs/outputs.
func formationRestoreAfterStatusNode(in []int64, r Rand) []int64 {
	if len(in) != 54 || in[0] < 0 || in[0] > 7 || in[1] < 0 || in[1] > 4294967295 {
		panic("formation restore: expected character, u32 status and 52 formation bytes")
	}
	if restoreFacingError != nil {
		panic(restoreFacingError)
	}
	var f [116]byte
	for i, v := range in[2:] {
		if v < 0 || v > 255 {
			panic("formation restore: expected u8 fields")
		}
		off := i
		if i >= 20 {
			off = 20 + (i-20)/4*12 + (i-20)%4
		}
		f[off] = byte(v)
	}
	record := func(c int) int {
		if c < 0 || c >= 8 {
			panic("formation restore: invalid member")
		}
		return 20 + 12*c
	}
	var move func(int, int, int, bool, bool)
	move = func(c, q, slot int, compact, turn bool) {
		p := record(c)
		old := int(f[p])
		if old != 255 {
			if old >= 5 || f[p+2] >= 3 || f[3*old+int(f[p+2])] != byte(c) || f[15+old] == 0 {
				panic("formation restore: inconsistent source")
			}
			f[3*old+int(f[p+2])] = 255
			f[15+old]--
		}
		f[p], f[p+2] = byte(q), byte(slot)
		if q != 255 {
			if q < 0 || q >= 5 || slot < 0 || slot >= 3 || f[3*q+slot] != 255 || f[15+q] >= 3 {
				panic("formation restore: inconsistent destination")
			}
			f[3*q+slot] = byte(c)
			f[15+q]++
			if turn && restoreFacing[q] != 255 {
				f[p+3] = restoreFacing[q]
			}
		}
		if compact && old != 255 {
			switch f[15+old] {
			case 1:
				for s := 0; s < 3; s++ {
					if f[3*old+s] != 255 {
						move(int(f[3*old+s]), old, 0, false, true)
						break
					}
				}
			case 2:
				hole := -1
				for s := 2; s >= 0; s-- {
					if f[3*old+s] == 255 {
						hole = s
					} else if hole != -1 {
						move(int(f[3*old+s]), old, hole, false, true)
						hole = s
					}
				}
			}
		}
	}
	who := int(in[0])
	p := record(who)
	if in[1] >= 18 {
		if f[p] != 255 {
			f[p+1] = f[p]
			move(who, 255, 255, true, true)
		}
	} else if f[p] == 255 {
		place := func(q int) bool {
			if q < 0 || q >= 5 {
				panic("formation restore: invalid previous region")
			}
			switch f[15+q] {
			case 0, 2:
				if f[15+q] == 2 && f[3*q] != 255 {
					panic("formation restore: two-member center slot must be empty")
				}
				move(who, q, 0, true, true)
			case 1:
				if f[3*q] == 255 {
					panic("formation restore: single member must occupy center slot")
				}
				move(int(f[3*q]), q, 1, false, true)
				move(who, q, 2, true, true)
			default:
				return false
			}
			return true
		}
		previous := int(f[p+1])
		if previous == 255 || !place(previous) {
			f[p], f[p+2] = 255, 255
			for _, q := range []int{0, 4, 2} {
				if place(q) {
					break
				}
			}
		}
		f[p+1] = 255
	}
	out := make([]int64, 52)
	for i := range out {
		off := i
		if i >= 20 {
			off = 20 + (i-20)/4*12 + (i-20)%4
		}
		out[i] = int64(f[off])
	}
	return out
}
