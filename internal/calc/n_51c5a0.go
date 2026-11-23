package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Fixture cap: item definitions 0..autoEquipItems-1 (stride 269 at global
// 0x68516c, notes/re/loot.md:9). Any id used as a table index past the cap panics.
const autoEquipItems = 6

var autoEquipDispatch, autoEquipDispatchErr = loadAutoEquipDispatch()

// Five item-type entries (0..4): 0x51c63f; dispatch table: 0x51c83c.
func loadAutoEquipDispatch() ([5]uint32, error) {
	var targets [5]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return targets, fmt.Errorf("auto equip slot: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return targets, err
	}
	off, err := peOffset(b, 0x51c83c)
	if err != nil {
		return targets, err
	}
	if len(b)-off < 4*len(targets) {
		return targets, fmt.Errorf("auto equip slot: truncated dispatch table")
	}
	for i := range targets {
		targets[i] = binary.LittleEndian.Uint32(b[off+4*i:])
	}
	return targets, nil
}

// Inputs: gameStarted (u8 at 0x68517c), character +0x44b (u32), item index
// (arg 2), the four hand records at character +0xfa5/+0xfb1/+0xfbd/+0xfc9
// (i32 id, u32 at +4, u32 at +8; id -1 = empty), then per item definition
// k<autoEquipItems: +0x3e type, +0x41 flags (bit 2 = two-hand), +0x46 slot byte.
// Outputs: slot, then the four hand records after the call. Callees run as
// their Go nodes: 0x51c4e0 itemDefaultEquipSlotNode, 0x51cea0
// itemEquipSlotAllowedNode; 0x51fe30 (record move, character pointer NULL so
// its 0x520d10 refresh returns at once, 0x520f14) is modelled here.
func itemAutoEquipSlotNode(in []int64, r Rand) []int64 {
	if len(in) != 15+3*autoEquipItems {
		panic("auto equip slot: wrong input count")
	}
	started, count, idx := in[0], in[1], in[2]
	if started < 0 || started > 255 || count < 0 || count > 4294967295 {
		panic("auto equip slot: gameStarted is u8, +0x44b is u32")
	}
	var rec [4][3]int64
	for h := range rec {
		copy(rec[h][:], in[3+3*h:6+3*h])
		if rec[h][0] < -2147483648 || rec[h][0] > 2147483647 || rec[h][1] < 0 || rec[h][1] > 4294967295 || rec[h][2] < 0 || rec[h][2] > 4294967295 {
			panic("auto equip slot: hand record out of range")
		}
	}
	tbl := in[15:]
	for _, v := range tbl {
		if v < 0 || v > 255 {
			panic("auto equip slot: item bytes are u8")
		}
	}
	item := func(id int64) (typ, flags, slotByte int64) {
		if id < 0 || id >= autoEquipItems {
			panic("auto equip slot: item index outside fixture item table")
		}
		return tbl[3*id], tbl[3*id+1], tbl[3*id+2]
	}
	finish := func(slot int64) []int64 {
		out := []int64{slot}
		for _, x := range rec {
			out = append(out, x[:]...)
		}
		return out
	}
	typ, _, _ := item(idx)
	def := itemDefaultEquipSlotNode([]int64{idx, typ, started}, r)[0]
	if started != 0 || (def != 6 && def != 7) {
		return finish(def) // 0x51c5c0, 0x51c5c6..0x51c5ce
	}
	if typ > 4 {
		return finish(def) // 0x51c636
	}
	if autoEquipDispatchErr != nil {
		panic(autoEquipDispatchErr)
	}
	// The slot check 0x51cea0 sees the tested item, slot, ignore=0 and the hand ids.
	allowed := func(id, slot int64) bool {
		t, f, s := item(id)
		var fa, fc int64
		if rec[0][0] != -1 {
			_, fa, _ = item(rec[0][0])
		}
		if rec[2][0] != -1 {
			_, fc, _ = item(rec[2][0])
		}
		return itemEquipSlotAllowedNode([]int64{0, 0, 0, 0, 0, 0, id, t, f, s, slot, 0,
			rec[0][0], rec[1][0], rec[2][0], rec[3][0], fa, fc}, r)[0] != 0
	}
	// 0x51fe30(dst, src, NULL, 1): +0xa of src cleared, 12 bytes copied, src emptied.
	move := func(dst, src int) {
		rec[dst] = [3]int64{rec[src][0], rec[src][1], rec[src][2] &^ 0xff0000}
		rec[src] = [3]int64{-1, 0, 0}
	}
	plain := func(id int64) bool { // type 0/1, not two-hand: 0x51c68d..0x51c69c
		t, f, _ := item(id)
		return (t == 0 || t == 1) && f&4 == 0
	}
	switch target := autoEquipDispatch[typ]; target {
	case 0x51c830: // type 2
		return finish(def)
	case 0x51c646: // types 0, 1
		if count == 0 {
			return finish(def) // 0x51c64e
		}
		if allowed(idx, 7) {
			if rec[0][0] != -1 && rec[1][0] == -1 && plain(rec[0][0]) {
				def = 7
			}
			if rec[2][0] != -1 && rec[3][0] == -1 && plain(rec[2][0]) {
				return finish(9) // 0x51c6e3
			}
			return finish(def)
		}
		if rec[0][0] != -1 && rec[1][0] == -1 && allowed(rec[0][0], 7) {
			move(1, 0)
			return finish(6) // 0x51c73a
		}
		if rec[2][0] != -1 && rec[3][0] == -1 && allowed(rec[2][0], 9) {
			move(3, 2)
			return finish(8) // 0x51c78c
		}
		return finish(def)
	case 0x51c79e: // type 3
		if rec[0][0] == -1 && rec[1][0] == -1 {
			return finish(6) // 0x51c7ad
		}
		if rec[2][0] == -1 && rec[3][0] == -1 {
			return finish(8) // 0x51c7cb
		}
		return finish(def)
	case 0x51c7dd: // type 4
		if rec[0][0] != -1 {
			if t, _, _ := item(rec[0][0]); t == 3 {
				return finish(7) // 0x51c7fb
			}
		}
		if rec[2][0] != -1 {
			if t, _, _ := item(rec[2][0]); t == 3 {
				return finish(9) // 0x51c822
			}
		}
		return finish(-1) // 0x51c82c
	default:
		panic(fmt.Sprintf("auto equip slot: unknown dispatch target %#x", target))
	}
}

func init() {
	in := []string{"gameStarted", "charCount0x44b", "itemIdx"}
	hands := []string{"mainHand", "offHand", "setBMainHand", "setBOffHand"}
	for _, h := range hands {
		in = append(in, h+".id", h+".word4", h+".word8")
	}
	out := append([]string{"equipSlot"}, in[3:15]...)
	for k := 0; k < autoEquipItems; k++ {
		in = append(in, fmt.Sprintf("item[%d].type", k), fmt.Sprintf("item[%d].flags", k), fmt.Sprintf("item[%d].slotByte", k))
	}
	Nodes["0051c5a0"] = Node{Addr: "0051c5a0", In: in, Out: out, F: itemAutoEquipSlotNode}
}
