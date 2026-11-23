package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// starting_party_resources 0x54b100 (cdecl, no arguments): reset the party
// resource block 0x685170..0x689b32 and give the party its starting items
// (build/wiz8.asm:449970..).
//   - free and re-allocate the two heap blocks 0x685174 / 0x685178 (0xc310 and
//     0x830 bytes, zeroed); clear the 350-entry x 4 table at 0x68f2d8 through
//     0x58fd30 (every entry ends zero; frees and list teardown are not observable).
//   - clear the held item (0x520070 on 0x6874cb: markers 0x6840c0/4/5 = -1, held
//     record id -1, cursor graphic released through the stubbed 0x427ab0/0x427ff0
//     when 0x683fdb is not -1) and the 500 shared-pack slots (id -1, one stubbed
//     0x520d10 each).
//   - for each of the six ids in the exe table at 0x6164dc: 0x51c020 builds the
//     item with the maximum roll and flags 6 and 0xb set, byte +4 is set to 1 and
//     0x521ef0 adds it to the pack: kind 1 items are first merged into the
//     existing slots through 0x51f900 (0x4edd20 on a merge or a partial merge), an
//     unmerged item is inserted at slot 0 behind the shifted pack through 0x51fe30
//     (0x4edd20 once more); the tail sets the mode-6 bits in [0x69c0f4]+0xfc and
//     calls 0x522640 / 0x5227d0 (no-ops for these ids; 0x5227d0 calls the stubbed
//     0x518150(0,0,-1,-1) once unless mode 0x68ec78 is 5).
//   - 0x685189 = 500, 0x68518d = 0x518230(1,1,-1) = -1 (the arrays are zeroed),
//     0x686a70 = -1, 0x687511 formation block (0x554580), 0x686a50.. = -1 x 8.
//
// Callees: 0x51c020 (itemStackChargeInitNode) and 0x51f900 (itemStackMergeNode,
// found by name at call time) are called directly. 0x520d10, 0x4edd20, 0x518150
// (formula, address stubs with return from a case input), 0x427ab0, 0x427ff0
// (排除, address stubs): counts and recorded args are outputs.
//
// Fixture facts and caps: the table ids are mapped to dense indices 0..3 (in
// order of first use) for the two callee fixtures; real-id checks (item count,
// ids with special branches in 0x522640 / 0x5227d0) are made here. Pack
// count <= 16 in the merge fixture (the pack never exceeds 6 from a zeroed
// state). srAssertFail path: id >= item count (0x51c020) panics. Heap block
// addresses are not outputs (the bump allocator hands out different ones in
// the original and the heap pointers are not rule data); a failed malloc is
// not modelled. Table pointers handed to 0x58fd30 must be 0 or valid arena
// lists (invalid ones crash the original); they are not rule inputs here.
const sprTable = 0x6164dc

var sprEntries = []int{0, 349, 350, 1399} // table entries probed by the case (all 1400 end zero)

var sprIn = func() []string {
	n := []string{"mode", "itemCount", "cursorDB", "cursorDF", "cursorE3", "old174", "old178", "m6fc"}
	for j := 0; j < 4; j++ {
		for _, f := range []string{"kind", "z", "n", "s", "attr", "limit"} {
			n = append(n, fmt.Sprintf("d%d.%s", j, f))
		}
	}
	n = append(n, "ret520d10", "ret4edd20", "ret427ab0", "ret427ff0", "ret518150")
	for _, e := range sprEntries {
		for d := 0; d < 9; d++ {
			if d == 6 {
				n = append(n, fmt.Sprintf("t%d.list", e))
			} else {
				n = append(n, fmt.Sprintf("t%d.d%d", e, d))
			}
		}
	}
	return append(n, "lstCount")
}()

var sprOut = func() []string {
	var n []string
	for _, k := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 499} {
		n = append(n, fmt.Sprintf("pack%d.id", k), fmt.Sprintf("pack%d.tail", k))
	}
	n = append(n, "packCount", "held.id", "held.tail", "heldMark", "g6840c0", "g6840c4", "g6840c5", "cursorDB", "cursorDF", "cursorE3",
		"g685189", "g68518d", "g686a70")
	for k := 0; k < 8; k++ {
		n = append(n, fmt.Sprintf("g686a50.%d", k))
	}
	for k := 0; k < 17; k++ {
		n = append(n, fmt.Sprintf("form%d", k))
	}
	n = append(n, "m6fc")
	for _, e := range sprEntries {
		for d := 0; d < 9; d++ {
			n = append(n, fmt.Sprintf("t%d.d%d", e, d))
		}
	}
	return append(n, "n520d10", "a520d10.2", "a520d10.3", "n4edd20", "n427ab0",
		"a427ab0.1", "a427ab0.2", "a427ab0.3", "a427ab0.4", "n427ff0", "n518150",
		"a518150.1", "a518150.2", "a518150.3", "a518150.4")
}()

func init() {
	Nodes["0054b100"] = Node{Addr: "0054b100", In: sprIn, Out: sprOut, F: startingPartyResourcesNode}
}

// sprIDs reads the exe table 0x6164dc (six dwords, -1 = skipped).
func sprIDs() []int64 {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		panic("starting_party_resources: " + err.Error())
	}
	o, err := peOffset(b, sprTable)
	if err != nil || len(b)-o < 24 {
		panic("starting_party_resources: exe table unreadable")
	}
	var ids []int64
	for k := 0; k < 6; k++ {
		if id := int32(binary.LittleEndian.Uint32(b[o+4*k:])); id != -1 {
			ids = append(ids, int64(id))
		}
	}
	return ids
}

func startingPartyResourcesNode(in []int64, r Rand) []int64 {
	if len(in) != len(sprIn) {
		panic("starting_party_resources: wrong input count")
	}
	at := map[string]int{}
	for i, n := range sprIn {
		at[n] = i
	}
	v := func(n string) int64 { return in[at[n]] }
	mode, itemCount := v("mode")&0xffffffff, v("itemCount")&0xffffffff

	// held item and shared pack cleared; cursor graphic released
	var n520d10, n4edd20, n427ab0, n427ff0, n518150 int64
	var a427ab0 [4]int64
	cur := [3]int64{v("cursorDB") & 0xffffffff, v("cursorDF") & 0xffffffff, v("cursorE3") & 0xffffffff}
	if cur[0] != 0xffffffff {
		word, dword, _ := iscCursorTable()
		a427ab0 = [4]int64{word, dword, 0, 0}
		n427ab0, n427ff0 = 1, 1
		cur = [3]int64{0xffffffff, 0, 0}
	}
	n520d10 += 500

	ids := sprIDs()
	dense := map[int64]int64{}
	var real []int64
	for _, id := range ids {
		if _, ok := dense[id]; !ok {
			dense[id] = int64(len(real))
			real = append(real, id)
		}
		if len(real) > iscItems {
			panic("starting_party_resources: more distinct item ids than the fixture records")
		}
		switch id {
		case 0x239, 0x242, 0x243, 0x244, 0x264, 0x27c, 0x201, 0x1d2:
			panic(fmt.Sprintf("starting_party_resources: item %#x takes a special branch in 0x522640 / 0x5227d0", id))
		}
	}

	mergeIn := Nodes["0051f900"].In
	mergeAt := map[string]int{}
	for i, n := range mergeIn {
		mergeAt[n] = i
	}
	mergeOutAt := map[string]int{}
	for i, n := range Nodes["0051f900"].Out {
		mergeOutAt[n] = i
	}
	word, _, _ := iscCursorTable()

	pack := []ismInst{} // slots 0..count-1; every other slot is (-1, 0)
	for _, id := range ids {
		if id >= int64(itemCount) {
			panic("starting_party_resources: srAssertFail uiItemNo < gXStatus.uiItemsInDatabase")
		}
		dj := dense[id]
		fld := func(f string) int64 { return v(fmt.Sprintf("d%d.%s", dj, f)) }

		// 0x51c020(tmp, id, 1, 1, 1)
		iv := make([]int64, len(iscIn))
		set := func(n string, x int64) {
			for i, name := range iscIn {
				if name == n {
					iv[i] = x
					return
				}
			}
			panic("starting_party_resources: no 0x51c020 input " + n)
		}
		set("p", iscInst)
		set("id", dj)
		set("maxRoll", 1)
		set("forceFlag6", 1)
		set("setFlagB", 1)
		set("itemCount", iscItems)
		for j := 0; j < iscItems; j++ {
			for _, f := range []string{"kind", "z", "n", "s", "attr"} {
				set(fmt.Sprintf("i%d.%s", j, f), v(fmt.Sprintf("d%d.%s", j, f)))
			}
		}
		set("heldCB", 0xffffffff)
		set("g6840c0", 0xffffffff)
		set("g6840c4", 0xff)
		set("g6840c5", 0xffff)
		set("cursorDB", cur[0])
		set("cursorDF", cur[1])
		set("cursorE3", cur[2])
		set("refreshRet", v("ret520d10"))
		set("encRet", v("ret4edd20"))
		set("blitRet", v("ret427ab0"))
		set("resetRet", v("ret427ff0"))
		io := itemStackChargeInitNode(iv, r)
		get := func(n string) int64 {
			for i, name := range iscOut {
				if name == n {
					return io[i]
				}
			}
			panic("starting_party_resources: no 0x51c020 output " + n)
		}
		n520d10 += get("refreshCalls")
		n4edd20 += get("encumbranceCalls")
		if get("blitCalls") != 0 || get("resetCalls") != 0 || get("packCount") != 0 {
			panic("starting_party_resources: 0x51c020 touched the cursor or the pack")
		}
		// tmp: id, then bytes +4..+11 with +4 forced to 1
		tmp := ismInst{int32(dj), uint64(uint32(get("inst4")))&^0xff | 1 | uint64(uint32(get("inst8")))<<32}
		kind := fld("kind") & 0xff

		// 0x521ef0(tmp, 0, 0)
		merged := false
		flag := int64(0)
		if kind == 1 && len(pack) > 0 {
			for k := 0; ; {
				mi := make([]int64, len(mergeIn))
				ms := func(n string, x int64) { mi[mergeAt[n]] = x }
				ms("p1", ismPack+12*int64(k))
				ms("p2", ismPB)
				ms("pFlag", ismPF)
				ms("id1", int64(pack[k].id))
				put := func(prefix string, x ismInst) {
					ms(prefix+".id", int64(x.id))
					ms(prefix+".qty", int64(x.tail&0xff))
					ms(prefix+".b5", int64(x.tail>>8&0xff))
					ms(prefix+".b6", int64(x.tail>>16&0xff))
					ms(prefix+".b7", int64(x.tail>>24&0xff))
					ms(prefix+".w8", int64(x.tail>>32))
				}
				put("A", ismInst{-1, 0})
				put("B", tmp)
				put("hand", ismInst{-1, 0})
				for q := 0; q < ismPool; q++ {
					s := ismInst{-1, 0}
					if q < len(pack) {
						s = pack[q]
					}
					put(fmt.Sprintf("pack%d", q), s)
				}
				if len(pack) > ismPool {
					panic("starting_party_resources: pack past the merge fixture cap")
				}
				ms("packCount", int64(len(pack)))
				ms("mark0", 0xffffffff)
				ms("mark4", 0xff)
				ms("mark5", 0xffff)
				ms("cursorGfx", cur[0])
				ms("cursorB", cur[1])
				ms("cursorC", cur[2])
				ms("cursorTex", 0)
				ms("tblWord", word)
				ms("defType", v(fmt.Sprintf("d%d.kind", pack[k].id))&0xff) // definition of slot k, not of the new item
				ms("defMax", v(fmt.Sprintf("d%d.limit", pack[k].id))&0xff)
				ms("ret520d10", v("ret520d10"))
				ms("ret4edd20", v("ret4edd20"))
				ms("flagByte", flag)
				mo := itemStackMergeNode(mi, r)
				mg := func(n string) int64 { return mo[mergeOutAt[n]] }
				tmp = ismInst{int32(mg("B.id")), uint64(mg("B.tail"))}
				flag = mg("flagByte")
				np := int(mg("packCount"))
				pack = pack[:0]
				for q := 0; q < np; q++ {
					pack = append(pack, ismInst{int32(mg(fmt.Sprintf("pack%d.id", q))), uint64(mg(fmt.Sprintf("pack%d.tail", q)))})
				}
				n520d10 += mg("n520d10")
				n4edd20 += mg("n4edd20")
				if mg("n427ab0") != 0 || mg("n427ff0") != 0 {
					panic("starting_party_resources: merge released the cursor")
				}
				if mg("ret") != 0 {
					merged = true
					break
				}
				k++
				if k >= len(pack) {
					break
				}
			}
		}
		if merged {
			n4edd20++
		} else {
			if flag != 0 {
				n4edd20++
			}
			if len(pack) >= 500 {
				continue
			}
			tmp.tail &^= 0xff << 48 // 0x51fe30 clears byte +0xa of the source
			pack = append([]ismInst{tmp}, pack...)
			n520d10 += 2 // 0x51fe30: refresh of the source and of the destination
			n4edd20++
		}
		if mode != 5 {
			n518150++
		}
	}

	pid := func(x ismInst) int64 {
		if x.id < 0 {
			return -1
		}
		return real[x.id]
	}
	var out []int64
	for _, k := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 499} {
		s := ismInst{-1, 0}
		if k < len(pack) {
			s = pack[k]
		}
		out = append(out, pid(s), int64(s.tail))
	}
	out = append(out, int64(len(pack)), -1, 0, 0, 0xffffffff, 0xff, 0xffff, cur[0], cur[1], cur[2], 500, -1, -1)
	for k := 0; k < 8; k++ {
		out = append(out, -1)
	}
	var nf [132]byte
	for i := 0; i < 15; i++ {
		nf[i] = 255
	}
	for c := 0; c < 8; c++ {
		nf[20+12*c], nf[21+12*c], nf[22+12*c], nf[23+12*c] = 255, 255, 255, 4
	}
	for k := 0; k < 16; k++ {
		out = append(out, int64(binary.LittleEndian.Uint64(nf[8*k:])))
	}
	out = append(out, int64(binary.LittleEndian.Uint32(nf[128:])))
	m6 := v("m6fc") & 0xffffffff
	if mode == 6 && len(ids) > 0 {
		m6 |= 0x7fc00000
	}
	out = append(out, m6)
	for range sprEntries {
		for d := 0; d < 9; d++ {
			out = append(out, 0)
		}
	}
	a518150 := [4]int64{}
	if n518150 > 0 {
		a518150 = [4]int64{0, 0, -1, -1}
	}
	out = append(out, n520d10, 0, 1, n4edd20, n427ab0, a427ab0[0], a427ab0[1], a427ab0[2], a427ab0[3], n427ff0, n518150,
		a518150[0], a518150[1], a518150[2], a518150[3])
	return out
}
