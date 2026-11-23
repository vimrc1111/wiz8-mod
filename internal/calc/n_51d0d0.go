package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// Twelve equipment records, stride 12: caller 0x50de15..0x50de1c.
const curseActivationSlots = 12

type curseActivationTables struct {
	applicable     [curseActivationSlots]bool
	name, quantity []uint16
}

var curseActivationTable, curseActivationErr = loadCurseActivationTables()

func loadCurseActivationTables() (curseActivationTables, error) {
	var t curseActivationTables
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("curse activation: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	off, err := peOffset(b, 0x648c5c)
	if err != nil {
		return t, err
	}
	if len(b)-off < 4*curseActivationSlots {
		return t, fmt.Errorf("curse activation: truncated slot table")
	}
	for i := range t.applicable {
		t.applicable[i] = binary.LittleEndian.Uint32(b[off+4*i:]) != 0xffffffff
	}
	// Name presentation formats used by child 0x51b5c0; read, never embed.
	readText := func(va uint32) ([]uint16, error) {
		off, err := peOffset(b, va)
		if err != nil {
			return nil, err
		}
		var text []uint16
		for off+2 <= len(b) {
			ch := binary.LittleEndian.Uint16(b[off:])
			off += 2
			if ch == 0 {
				return text, nil
			}
			text = append(text, ch)
		}
		return nil, fmt.Errorf("curse activation: unterminated format at %#x", va)
	}
	t.name, err = readText(0x6068e4)
	if err != nil {
		return t, err
	}
	t.quantity, err = readText(0x61a700)
	return t, err
}

// Child 0x51b5c0 formats a UTF-16 name, optionally with u8 quantity.
// Preserve code units, including unpaired surrogates, as the CRT replay does.
func curseActivationName(format []uint16, name uint16, quantity int64) []int64 {
	var out []int64
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			out = append(out, int64(format[i]))
			continue
		}
		i++
		if i == len(format) {
			panic("curse activation: incomplete name format")
		}
		switch format[i] {
		case 's':
			if name != 0 {
				out = append(out, int64(name))
			}
		case 'd':
			for _, ch := range strconv.FormatInt(quantity, 10) {
				out = append(out, int64(ch))
			}
		case '%':
			out = append(out, '%')
		default:
			panic("curse activation: unsupported name format")
		}
	}
	return append(out, 0)
}

// Pure after exe initialization. The selected template has stride 269
// (0x51d0f5..0x51d100). Slot +0xf68 is an in/out byte, not an EAX result.
// The replay has a populated category-name cache and no message-window object;
// the real children still format the name and prompt before the UI no-op return.
func itemCurseActivationNode(in []int64, r Rand) []int64 {
	// Inputs: slot, item ID, curse/exemption/activation, quantity, identification,
	// template name, category name, category ID, and eight prior UTF-16 buffer units.
	if len(in) != 18 || in[0] < 0 || in[0] >= curseActivationSlots || in[1] < -1 || in[1] > 7 {
		panic("curse activation: expected slot 0..11, itemID -1..7 and 18 inputs")
	}
	for i := 2; i <= 6; i++ {
		if in[i] < 0 || in[i] > 255 {
			panic("curse activation: expected u8 flags/quantity")
		}
	}
	for _, i := range []int{7, 8, 10, 11, 12, 13, 14, 15, 16, 17} {
		if in[i] < 0 || in[i] > 65535 {
			panic("curse activation: expected UTF-16 code units")
		}
	}
	if in[9] < 0 || in[9] > 7 {
		panic("curse activation: expected cached category 0..7")
	}
	out := append([]int64{in[4]}, in[10:]...)
	if in[1] == -1 {
		return out
	}
	if curseActivationErr != nil {
		panic(curseActivationErr)
	}
	if !curseActivationTable.applicable[in[0]] || in[4] != 0 {
		return out
	}
	// 0x51d16d/0x51d133 both write literal 1 to the activation byte.
	out[0] = 1
	if in[2] == 0 || in[3] != 0 {
		return out
	}
	name := in[7]
	if in[6] == 0 {
		name = in[8]
	}
	format := curseActivationTable.name
	if in[5] >= 2 {
		format = curseActivationTable.quantity
	}
	text := curseActivationName(format, uint16(name), in[5])
	if len(text) > len(out)-1 {
		panic("curse activation: name exceeds replay buffer projection")
	}
	copy(out[1:], text)
	return out
}

func init() {
	Nodes["0051d0d0"] = Node{
		Addr: "0051d0d0",
		In:   []string{"slot", "itemID", "itemCursed", "slotCurseExempt", "activation", "quantity", "identified", "templateName", "categoryName", "categoryID", "nameBuffer0", "nameBuffer1", "nameBuffer2", "nameBuffer3", "nameBuffer4", "nameBuffer5", "nameBuffer6", "nameBuffer7"},
		Out:  []string{"activation", "nameBuffer0", "nameBuffer1", "nameBuffer2", "nameBuffer3", "nameBuffer4", "nameBuffer5", "nameBuffer6", "nameBuffer7"},
		F:    itemCurseActivationNode,
	}
}
