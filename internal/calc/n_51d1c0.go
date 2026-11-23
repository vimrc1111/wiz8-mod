package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Equipment slots 0..11: the caller at 0x50de15..0x50de1c visits twelve
// records, each twelve bytes apart. 0x51d204 reads their applicability table.
const itemRemovalSlotCount = 12

var itemRemovalRestrictedSlots, itemRemovalTableErr = loadItemRemovalSlots()

func loadItemRemovalSlots() ([itemRemovalSlotCount]bool, error) {
	var restricted [itemRemovalSlotCount]bool
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return restricted, fmt.Errorf("item removal: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return restricted, err
	}
	// 0x648c5c: one i32 per slot; -1 exempts that slot from curse locking.
	offset, err := peOffset(b, 0x648c5c)
	if err != nil {
		return restricted, err
	}
	if len(b)-offset < 4*len(restricted) {
		return restricted, fmt.Errorf("item removal: truncated slot applicability table")
	}
	for slot := range restricted {
		restricted[slot] = binary.LittleEndian.Uint32(b[offset+4*slot:]) != 0xffffffff
	}
	return restricted, nil
}

// Pure after initialization. Inputs are the selected slot/template contents;
// pointers only select records. Template stride 269 is from 0x51d1db..0x51d1e6;
// the template base pointer lives at 0x68516c. AL is the boolean interface
// consumed at 0x50dde7; the other EAX bytes do not belong to the result.
func itemRemovalAllowedNode(in []int64, r Rand) []int64 {
	// u32 widths: item ID at 0x51d1d0 and character override at 0x51d20e.
	// u8 widths: template curse at 0x51d1ee, slot curse exemption at 0x51d1fa.
	const maxU32 int64 = 4294967295
	const maxU8 int64 = 255
	if len(in) != 5 || in[0] < 0 || in[0] >= itemRemovalSlotCount ||
		in[1] < 0 || in[1] > maxU32 || in[2] < 0 || in[2] > maxU8 ||
		in[3] < 0 || in[3] > maxU8 || in[4] < 0 || in[4] > maxU32 {
		panic("item removal: expected slot 0..11, u32 itemID, u8 itemCursed/slotCurseExempt, u32 characterCurseOverride")
	}
	// Empty ID (0x51d1d6), uncursed template, or slot exemption allow removal.
	if in[1] == maxU32 || in[2] == 0 || in[3] != 0 {
		return []int64{1}
	}
	if itemRemovalTableErr != nil {
		panic(itemRemovalTableErr)
	}
	if !itemRemovalRestrictedSlots[in[0]] || in[4] != 0 {
		return []int64{1}
	}
	return []int64{0}
}

func init() {
	Nodes["0051d1c0"] = Node{
		Addr: "0051d1c0",
		In:   []string{"slot", "itemID", "itemCursed", "slotCurseExempt", "characterCurseOverride"},
		Out:  []string{"removalAllowed"}, F: itemRemovalAllowedNode,
	}
}
