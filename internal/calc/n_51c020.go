package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// item_stack_charge_init 0x51c020 (initNewItem): fill a fresh item instance p
// from item definition `id` (notes/re/loot.md:9, :19): quantity or charges
// (a dice roll, or the maximum when maxRoll is set) go to p+4 (kind 1) or p+5
// (kind 2..4), flags to p+6 and p+0xb. Before that p is reset; when p is the
// held-item instance 0x6874cb the cursor-reset 0x55f1e0 runs, otherwise
// 0x520d10(p,0,1); a pack slot that p points into is then removed from the
// shared pack (0x685191, 12-byte slots) and 0x4edd20 reruns.
//
// Callees: 0x55f1e0 runs natively and is transliterated here. 0x517970,
// 0x520d10, 0x4edd20 (D4 formulas) and 0x427ab0, 0x427ff0 (cursor image,
// registered 排除) are address stubs in the case: each return value is a case
// input (diceRet, refreshRet, encRet, blitRet, resetRet), the arguments and call
// counts are outputs. A real query rolls the dice with diceNode(Z,N,S) instead
// of reading diceRet (the F below is the differential composition).
//
// Fixture facts (re/emu/cases/0051c020.json): 0x650e20=1 (video initialised,
// otherwise 0x5493e0 asserts) and the loaded flag 0x62c464 nonzero (lazy
// loader behind 0x549390 is resource I/O). Fixture caps: pack count <= 4,
// item records <= 4; calc panics past them. Assert-only paths (id >= item
// count, Z+N*S >= 256, kind >= 5) panic like srAssertFail stops the emulator.
const (
	iscCap     = 4
	iscTabN    = 3*iscCap + 4 // table dwords modelled (slots plus 16 spare bytes)
	iscItems   = 4
	iscTab     = 0x685191
	iscHeld    = 0x6874cb
	iscInst    = 0x20004000 // arena object "inst" (case objects)
	iscPerItem = 5
)

var iscIn = func() []string {
	n := []string{"p", "id", "maxRoll", "forceFlag6", "setFlagB", "itemsBase", "itemCount", "packCount"}
	for i := 0; i < iscTabN; i++ {
		n = append(n, fmt.Sprintf("t%d", i))
	}
	n = append(n, "inst0", "inst4", "inst8", "heldCA", "heldCB", "heldCF", "heldD3", "g6840c0", "g6840c4", "g6840c5", "cursorDB", "cursorDF", "cursorE3")
	for j := 0; j < iscItems; j++ {
		for _, f := range []string{"kind", "z", "n", "s", "attr"} {
			n = append(n, fmt.Sprintf("i%d.%s", j, f))
		}
	}
	return append(n, "diceRet", "refreshRet", "encRet", "blitRet", "resetRet")
}()

var iscOut = func() []string {
	n := []string{"inst0", "inst4", "inst8"}
	for i := 0; i < iscTabN; i++ {
		n = append(n, fmt.Sprintf("t%d", i))
	}
	return append(n, "packCount", "g6840c0", "g6840c4", "g6840c5", "heldCA", "heldCB", "heldCF", "heldD3", "cursorDB", "cursorDF", "cursorE3",
		"diceArg", "diceCalls", "refreshP", "refreshA2", "refreshA3", "refreshCalls", "encumbranceCalls",
		"blitA1", "blitA2", "blitA3", "blitA4", "blitCalls", "resetCalls")
}()

func init() {
	Nodes["0051c020"] = Node{Addr: "0051c020", In: iscIn, Out: iscOut, F: itemStackChargeInitNode}
}

// iscCursorTable reads what 0x5493e0 and 0x549390 return for (0,0): a word at
// 0x6448cc and the dword at 0x62c468+60*[0x6448c8], from the player's exe.
func iscCursorTable() (word, dword int64, idx int64) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		panic("item_stack_charge_init: " + err.Error())
	}
	get := func(va uint32, n int) int64 {
		o, err := peOffset(b, va)
		if err != nil || len(b)-o < n {
			panic(fmt.Sprintf("item_stack_charge_init: exe table %#x unreadable", va))
		}
		if n == 2 {
			return int64(binary.LittleEndian.Uint16(b[o:]))
		}
		return int64(binary.LittleEndian.Uint32(b[o:]))
	}
	idx = get(0x6448c8, 4)
	if idx != 0 {
		panic("item_stack_charge_init: fixture loaded flag only covers table index 0")
	}
	return get(0x6448cc, 2), get(0x62c468+60*uint32(idx), 4), idx
}

type iscMem map[int64]byte

func (m iscMem) put(a int64, n int, v int64) {
	for i := 0; i < n; i++ {
		m[a+int64(i)] = byte(v >> (8 * uint(i)))
	}
}
func (m iscMem) wr(a int64, n int, v int64) {
	for i := 0; i < n; i++ {
		if _, ok := m[a+int64(i)]; !ok {
			panic(fmt.Sprintf("item_stack_charge_init: write to unmodelled address %#x", a+int64(i)))
		}
	}
	m.put(a, n, v)
}
func (m iscMem) rd(a int64, n int) int64 {
	var v int64
	for i := n - 1; i >= 0; i-- {
		b, ok := m[a+int64(i)]
		if !ok {
			panic(fmt.Sprintf("item_stack_charge_init: read of unmodelled address %#x", a+int64(i)))
		}
		v = v<<8 | int64(b)
	}
	return v
}

func itemStackChargeInitNode(in []int64, r Rand) []int64 {
	if len(in) != len(iscIn) {
		panic("item_stack_charge_init: wrong input count")
	}
	at := map[string]int{}
	for i, n := range iscIn {
		at[n] = i
	}
	v := func(n string) int64 { return in[at[n]] }
	m := iscMem{}
	for i := 0; i < iscTabN; i++ {
		m.put(iscTab+4*int64(i), 4, v(fmt.Sprintf("t%d", i)))
	}
	for i, n := range []string{"inst0", "inst4", "inst8"} {
		m.put(iscInst+4*int64(i), 4, v(n))
	}
	m.put(0x6874ca, 1, v("heldCA"))
	m.put(0x6874cb, 4, v("heldCB"))
	m.put(0x6874cf, 4, v("heldCF"))
	m.put(0x6874d3, 4, v("heldD3"))
	m.put(0x6840c0, 4, v("g6840c0"))
	m.put(0x6840c4, 1, v("g6840c4"))
	m.put(0x6840c5, 2, v("g6840c5"))
	m.put(0x683fdb, 4, v("cursorDB"))
	m.put(0x683fdf, 4, v("cursorDF"))
	m.put(0x683fe3, 4, v("cursorE3"))
	m.put(0x686901, 4, v("packCount"))

	p, id := v("p")&0xffffffff, v("id")&0xffffffff
	var diceArg, diceCalls, refreshP, refreshA2, refreshA3, refreshCalls, encCalls int64
	var blit [4]int64
	var blitCalls, resetCalls int64

	if id >= v("itemCount")&0xffffffff {
		panic("item_stack_charge_init: srAssertFail uiItemNo < gXStatus.uiItemsInDatabase")
	}
	if p == iscHeld {
		m.wr(0x6840c0, 4, -1)
		m.wr(0x6840c4, 1, 0xff)
		m.wr(0x6840c5, 2, 0xffff)
		// 0x55f1e0
		m.wr(0x6874cb, 4, 0)
		m.wr(0x6874cf, 4, 0)
		m.wr(0x6874d3, 4, 0)
		m.wr(0x6874ca, 1, 0)
		m.wr(0x6874cb, 4, -1)
		if m.rd(0x683fdb, 4) != 0xffffffff {
			word, dword, _ := iscCursorTable()
			blit = [4]int64{word, dword, 0, 0} // 0x427ab0(0x5493e0(0), 0x549390(0,0), 0, 0)
			blitCalls++
			resetCalls++ // 0x427ff0
			m.wr(0x683fdb, 4, -1)
			m.wr(0x683fdf, 4, 0)
			m.wr(0x683fe3, 4, 0)
		}
	} else {
		m.wr(p, 4, 0)
		m.wr(p+4, 4, 0)
		m.wr(p+8, 4, 0)
		m.wr(p, 4, -1)
		refreshP, refreshA2, refreshA3 = p, 0, 1 // 0x520d10(p, 0, 1)
		refreshCalls++
	}
	if count := m.rd(0x686901, 4); p >= iscTab && p <= 0x6868f5 && count != 0 {
		if count > iscCap {
			panic("item_stack_charge_init: pack count past fixture cap")
		}
		for k := int64(0); k < count; k++ {
			if p != iscTab+12*k {
				continue
			}
			if m.rd(p, 4) == 0xffffffff {
				for i := k; i < count-1; i++ {
					for b := int64(0); b < 12; b++ {
						m.wr(iscTab+12*i+b, 1, m.rd(iscTab+12*(i+1)+b, 1))
					}
				}
				last := iscTab + 12*(count-1)
				m.wr(last, 4, 0)
				m.wr(last+4, 4, 0)
				m.wr(last+8, 4, 0)
				m.wr(last, 4, -1)
				m.wr(0x686901, 4, count-1)
				encCalls++
			}
			break
		}
	}
	m.wr(p, 4, id)
	if id >= iscItems {
		panic("item_stack_charge_init: item record past fixture cap")
	}
	item := func(f string) int64 { return v(fmt.Sprintf("i%d.%s", id, f)) }
	if kind := item("kind") & 0xff; kind != 0 {
		z, n, s := item("z"), item("n")&0xff, item("s")&0xff
		if z+n*s >= 0x100 {
			panic("item_stack_charge_init: srAssertFail dice maximum below 256")
		}
		var al int64
		if v("maxRoll")&0xff != 0 {
			al = (s*n + z) & 0xff
		} else {
			al = v("diceRet") & 0xff
			diceArg, diceCalls = v("itemsBase")+269*id+0x67, 1
		}
		switch {
		case kind == 1:
			m.wr(p+4, 1, al)
		case kind <= 4:
			m.wr(p+5, 1, al)
		default:
			panic("item_stack_charge_init: srAssertFail unknown stack kind")
		}
	}
	if v("forceFlag6")&0xff != 0 || item("attr")&1 != 0 {
		m.wr(p+6, 1, 1)
	}
	if v("setFlagB")&0xff != 0 {
		m.wr(p+0xb, 1, 1)
	}

	out := []int64{m.rd(iscInst, 4), m.rd(iscInst+4, 4), m.rd(iscInst+8, 4)}
	for i := 0; i < iscTabN; i++ {
		out = append(out, m.rd(iscTab+4*int64(i), 4))
	}
	out = append(out, m.rd(0x686901, 4), m.rd(0x6840c0, 4), m.rd(0x6840c4, 1), m.rd(0x6840c5, 2),
		m.rd(0x6874ca, 1), m.rd(0x6874cb, 4), m.rd(0x6874cf, 4), m.rd(0x6874d3, 4),
		m.rd(0x683fdb, 4), m.rd(0x683fdf, 4), m.rd(0x683fe3, 4),
		diceArg, diceCalls, refreshP, refreshA2, refreshA3, refreshCalls, encCalls,
		blit[0], blit[1], blit[2], blit[3], blitCalls, resetCalls)
	return out
}
