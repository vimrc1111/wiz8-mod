package calc

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// character_replacement_gold_cost 0x4ef7e0 (re/emu/cases/004ef7e0.json): the party member
// in record ch is replaced by the character record src. The old backpack is filed into the
// shared pack (0x521ef0), src is copied over ch, the presence and runtime records of the
// slot are reset (0x54b470, 0x54b300 with its R(5000) timer), the slot enters the first
// free place of the order table 0x686a50, the counters 0x685185 and 0x68517d grow, and
// with pay != 0 the price of the starting kit (0x558640) is taken from the gold 0x685189
// (the kit itself is then handed over by 0x557430).
//
// Out is the structural state: slot (al), gold, the counters, ch[4], ch class, the 20
// inventory slots (id and the two following dwords), the pack count and 24 entries, the
// order table, eight presence-record fields of the slot, fourteen runtime-record fields of
// every aux record, the formation byte 0x687526+12*slot, and the call count and arguments
// of the 排除 callee 0x4ef610.
//
// Transliterated here (D4 sub-nodes are not reachable as nodes, their effects feed Out through
// the stub result sequences): 517ec0, 521ef0, 54b470, 54b300, 406bf0, 51fe30, 51c300, 520310,
// 557430, 558640, 51c020 (temp branch), 5227d0, 522640, 51d960, the skill part of 4ed9d0 /
// 5539e0 / 553a60, 50e980 / 50f030 / 50f090 (skill bytes), 50e5c0 / 50e540.
// NOT modelled (native in the emulator, outside Out): the other stages of 4ed9d0 (stats, hp,
// stamina, mana, resistances, load), 554ae0 (formation, writes outside Out), 561ec0 (UI
// refresh, mode 7), the story flags 0x506280 / 0x5061a0 reached through 0x522640 and 0x5227d0,
// the message 0x58aad0 (vswprintf is a case stub, text only), 4edc60 / 4edd20 / 520d10 records,
// the non-skill bytes of 50e980.
//
// Stubs (case "stubs", each result a case input): 0x51d610, 0x51c5a0, 0x51f900 (+ its out
// flag, arg3 is a stack address so only the null/non-null effect is modelled), 0x51b840.
// F runs their D4 Go (real51d610 and the others); Emu replays the case's results.
// Fixture caps: ids and shapes are limited by the D4 Go of the stubbed callees; the
// emulator-only assert paths (cost > gold, formation count 1, unknown stack kind, item
// definition past the fixture) panic like srAssertFail stops the emulator.
// generated tables: n4Grp{name, kind (g global, o arena, a stack arg, r eax), address, size, signed, count, step, index field, stride}
var n4InG = []n4Grp{
	{"chPtr", 'a', 0x0, 4, false, 1, 4, "", 0x0},
	{"who", 'o', 0x20003ff0, 4, false, 1, 4, "", 0x0},
	{"pay", 'a', 0x2, 1, false, 1, 1, "", 0x0},
	{"gold", 'g', 0x685189, 4, false, 1, 4, "", 0x0},
	{"mode", 'g', 0x68ec78, 4, false, 1, 4, "", 0x0},
	{"p6850ce", 'g', 0x6850ce, 4, false, 1, 4, "", 0x0},
	{"clock", 'g', 0x6eb70c, 4, false, 1, 4, "", 0x0},
	{"cntA", 'g', 0x685185, 4, false, 1, 4, "", 0x0},
	{"cntB", 'g', 0x68517d, 4, false, 1, 4, "", 0x0},
	{"order", 'g', 0x686a50, 4, false, 8, 4, "", 0x0},
	{"srcCls", 'o', 0x20011069, 4, false, 1, 4, "", 0x0},
	{"srcRes", 'o', 0x20011071, 4, false, 1, 4, "", 0x0},
	{"srcB01", 'o', 0x20011b01, 4, false, 1, 4, "", 0x0},
	{"srcB11", 'o', 0x20011b11, 4, false, 1, 4, "", 0x0},
	{"src89", 'o', 0x20011089, 4, false, 1, 4, "", 0x0},
	{"gstFlags", 'o', 0x200141fc, 4, false, 1, 4, "", 0x0},
	{"src215", 'o', 0x20011215, 4, false, 1, 4, "", 0x0},
	{"src261", 'o', 0x20011261, 4, false, 1, 4, "", 0x0},
	{"src1c9", 'o', 0x200111c9, 4, false, 1, 4, "", 0x0},
	{"src1a3", 'o', 0x200111a3, 4, false, 1, 4, "", 0x0},
	{"oldAid", 'o', 0x20004f5d, 4, false, 12, 12, "who", 0x1862},
	{"srcAid", 'o', 0x20011f5d, 4, false, 12, 12, "", 0x0},
	{"oldAq1", 'o', 0x20004f61, 4, false, 12, 12, "who", 0x1862},
	{"srcAq1", 'o', 0x20011f61, 4, false, 12, 12, "", 0x0},
	{"oldAq2", 'o', 0x20004f65, 4, false, 12, 12, "who", 0x1862},
	{"srcAq2", 'o', 0x20011f65, 4, false, 12, 12, "", 0x0},
	{"oldBid", 'o', 0x20005029, 4, false, 8, 12, "who", 0x1862},
	{"srcBid", 'o', 0x20012029, 4, false, 8, 12, "", 0x0},
	{"oldBq1", 'o', 0x2000502d, 4, false, 8, 12, "who", 0x1862},
	{"srcBq1", 'o', 0x2001202d, 4, false, 8, 12, "", 0x0},
	{"oldBq2", 'o', 0x20005031, 4, false, 8, 12, "who", 0x1862},
	{"srcBq2", 'o', 0x20012031, 4, false, 8, 12, "", 0x0},
	{"packCount", 'g', 0x686901, 4, false, 1, 4, "", 0x0},
	{"packid", 'g', 0x685191, 4, false, 4, 12, "", 0x0},
	{"packq1", 'g', 0x685195, 4, false, 4, 12, "", 0x0},
	{"packq2", 'g', 0x685199, 4, false, 4, 12, "", 0x0},
	{"s51d610.ret", 'o', 0x20060000, 4, false, 300, 4, "", 0x0},
	{"s51c5a0.ret", 'o', 0x200604b0, 4, false, 300, 4, "", 0x0},
	{"s51f900.ret", 'o', 0x20060960, 4, false, 400, 4, "", 0x0},
	{"s51f900.wr", 'o', 0x20060fa0, 1, false, 1, 1, "", 0x0},
	{"s51b840.ret", 'o', 0x20060fa1, 4, false, 300, 4, "", 0x0},
}

var n4OutG = []n4Grp{
	{"s4ef610.a1", 'o', 0x20000000, 4, false, 1, 4, "", 0x0},
	{"s4ef610.a2", 'o', 0x20000004, 4, false, 1, 4, "", 0x0},
	{"s4ef610.calls", 'o', 0x20000008, 4, false, 1, 4, "", 0x0},
	{"slot", 'r', 0x0, 1, false, 1, 1, "", 0x0},
	{"goldAfter", 'g', 0x685189, 4, false, 1, 4, "", 0x0},
	{"cntAafter", 'g', 0x685185, 4, false, 1, 4, "", 0x0},
	{"cntBafter", 'g', 0x68517d, 4, false, 1, 4, "", 0x0},
	{"chInUse", 'o', 0x20004004, 1, false, 1, 1, "who", 0x1862},
	{"chCls", 'o', 0x20004069, 4, false, 1, 4, "who", 0x1862},
	{"chAid", 'o', 0x20004f5d, 4, false, 12, 12, "who", 0x1862},
	{"chAq1", 'o', 0x20004f61, 4, false, 12, 12, "who", 0x1862},
	{"chAq2", 'o', 0x20004f65, 4, false, 12, 12, "who", 0x1862},
	{"chBid", 'o', 0x20005029, 4, false, 8, 12, "who", 0x1862},
	{"chBq1", 'o', 0x2000502d, 4, false, 8, 12, "who", 0x1862},
	{"chBq2", 'o', 0x20005031, 4, false, 8, 12, "who", 0x1862},
	{"packCnt", 'g', 0x686901, 4, false, 1, 4, "", 0x0},
	{"packid", 'g', 0x685191, 4, false, 24, 12, "", 0x0},
	{"packq1", 'g', 0x685195, 4, false, 24, 12, "", 0x0},
	{"packq2", 'g', 0x685199, 4, false, 24, 12, "", 0x0},
	{"orderAfter", 'g', 0x686a50, 4, false, 8, 4, "", 0x0},
	{"pres0", 'o', 0x20013000, 1, false, 1, 1, "who", 0x106},
	{"pres6d", 'o', 0x2001306d, 4, false, 1, 4, "who", 0x106},
	{"pres71", 'o', 0x20013071, 4, false, 1, 4, "who", 0x106},
	{"pres75", 'o', 0x20013075, 4, false, 1, 4, "who", 0x106},
	{"presd0", 'o', 0x200130d0, 1, false, 1, 1, "who", 0x106},
	{"pres104", 'o', 0x20013104, 1, false, 1, 1, "who", 0x106},
	{"presf1", 'o', 0x200130f1, 4, false, 1, 4, "who", 0x106},
	{"presfa", 'o', 0x200130fa, 4, false, 1, 4, "who", 0x106},
	{"aux0", 'g', 0x6836b8, 1, false, 8, 280, "", 0x0},
	{"aux1", 'g', 0x6836b9, 4, false, 8, 280, "", 0x0},
	{"aux75", 'g', 0x68372d, 4, false, 8, 280, "", 0x0},
	{"aux79", 'g', 0x683731, 4, false, 8, 280, "", 0x0},
	{"aux7d", 'g', 0x683735, 4, false, 8, 280, "", 0x0},
	{"aux81", 'g', 0x683739, 4, false, 8, 280, "", 0x0},
	{"aux85", 'g', 0x68373d, 4, false, 8, 280, "", 0x0},
	{"aux89", 'g', 0x683741, 4, false, 8, 280, "", 0x0},
	{"aux8d", 'g', 0x683745, 4, false, 8, 280, "", 0x0},
	{"aux91", 'g', 0x683749, 4, false, 8, 280, "", 0x0},
	{"aux95", 'g', 0x68374d, 4, false, 8, 280, "", 0x0},
	{"auxa3", 'g', 0x68375b, 4, false, 8, 280, "", 0x0},
	{"auxab", 'g', 0x683763, 1, false, 8, 280, "", 0x0},
	{"auxca", 'g', 0x683782, 4, false, 8, 280, "", 0x0},
	{"auxd2", 'g', 0x68378a, 4, false, 8, 280, "", 0x0},
	{"auxbe", 'g', 0x683776, 4, false, 8, 280, "", 0x0},
	{"gstFlagsAfter", 'o', 0x200141fc, 4, false, 1, 4, "", 0x0},
	{"fbyte", 'g', 0x687526, 1, false, 8, 12, "", 0x0},
}

// setup constants (fixed environment), applied before the inputs
var n4SetupG = []n4Set{
	{0x685174, 4, 0x20004000, 1, 4},
	{0x685178, 4, 0x20013000, 1, 4},
	{0x683fd7, 4, 0x20016000, 1, 4},
	{0x69c0f4, 4, 0x20014100, 1, 4},
	{0x68c09c, 4, 0x20014200, 1, 4},
	{0x68edcc, 4, 0x20014000, 1, 4},
	{0x683f78, 4, 800, 1, 4},
	{0x68516c, 4, 0x20020000, 1, 4},
	{0x20004004, 1, 1, 8, 6242},
	{0x687511, 1, 255, 1, 1},
	{0x687512, 1, 255, 1, 1},
	{0x687513, 1, 255, 1, 1},
	{0x687514, 1, 255, 1, 1},
	{0x687515, 1, 255, 1, 1},
	{0x687516, 1, 255, 1, 1},
	{0x687517, 1, 255, 1, 1},
	{0x687518, 1, 255, 1, 1},
	{0x687519, 1, 255, 1, 1},
	{0x68751a, 1, 255, 1, 1},
	{0x68751b, 1, 255, 1, 1},
	{0x68751c, 1, 255, 1, 1},
	{0x68751d, 1, 255, 1, 1},
	{0x68751e, 1, 255, 1, 1},
	{0x68751f, 1, 255, 1, 1},
	{0x687520, 1, 2, 1, 1},
	{0x687522, 1, 2, 1, 1},
	{0x687524, 1, 2, 1, 1},
	{0x687525, 1, 255, 8, 12},
	{0x687526, 1, 255, 8, 12},
	{0x687527, 1, 255, 8, 12},
	{0x20004f5d, 4, 4294967295, 12, 12},
	{0x20005029, 4, 4294967295, 8, 12},
	{0x200067bf, 4, 4294967295, 12, 12},
	{0x2000688b, 4, 4294967295, 8, 12},
	{0x20008021, 4, 4294967295, 12, 12},
	{0x200080ed, 4, 4294967295, 8, 12},
	{0x20009883, 4, 4294967295, 12, 12},
	{0x2000994f, 4, 4294967295, 8, 12},
	{0x2000b0e5, 4, 4294967295, 12, 12},
	{0x2000b1b1, 4, 4294967295, 8, 12},
	{0x2000c947, 4, 4294967295, 12, 12},
	{0x2000ca13, 4, 4294967295, 8, 12},
	{0x2000e1a9, 4, 4294967295, 12, 12},
	{0x2000e275, 4, 4294967295, 8, 12},
	{0x2000fa0b, 4, 4294967295, 12, 12},
	{0x2000fad7, 4, 4294967295, 8, 12},
	{0x20011f5d, 4, 4294967295, 12, 12},
	{0x20012029, 4, 4294967295, 8, 12},
	{0x20004f5d, 4, 5, 8, 6242},
	{0x20004f61, 4, 119, 8, 6242},
	{0x20004f65, 4, 120, 8, 6242},
	{0x20005029, 4, 5, 8, 6242},
	{0x2000502d, 4, 119, 8, 6242},
	{0x20005031, 4, 120, 8, 6242},
	{0x20004b11, 4, 1, 4, 12484},
	{0x20004b01, 4, 3, 4, 12484},
	{0x20006363, 4, 20, 4, 12484},
	{0x2001499c, 4, 0x20018000, 1, 4},
	{0x20015368, 4, 0x20018100, 1, 4},
	{0x20016068, 4, 0x2001c000, 1, 4},
	{0x20018000, 2, 37, 1, 2},
	{0x20018002, 2, 115, 1, 2},
	{0x20018100, 2, 73, 1, 2},
	{0x6836b8, 1, 90, 8, 280},
	{0x6836b9, 4, 1515870810, 8, 280},
	{0x68372d, 4, 1515870810, 8, 280},
	{0x683731, 4, 1515870810, 8, 280},
	{0x683735, 4, 1515870810, 8, 280},
	{0x683739, 4, 1515870810, 8, 280},
	{0x68373d, 4, 1515870810, 8, 280},
	{0x683741, 4, 1515870810, 8, 280},
	{0x683745, 4, 1515870810, 8, 280},
	{0x683749, 4, 1515870810, 8, 280},
	{0x68374d, 4, 1515870810, 8, 280},
	{0x68375b, 4, 1515870810, 8, 280},
	{0x683763, 1, 90, 8, 280},
	{0x683782, 4, 1515870810, 8, 280},
	{0x68378a, 4, 1515870810, 8, 280},
	{0x683776, 4, 1515870810, 8, 280},
	{0x20013000, 1, 165, 8, 262},
	{0x2001306d, 4, 2779096485, 8, 262},
	{0x20013071, 4, 2779096485, 8, 262},
	{0x20013075, 4, 2779096485, 8, 262},
	{0x200130d0, 1, 165, 8, 262},
	{0x20013104, 1, 165, 8, 262},
	{0x200130f1, 4, 2779096485, 8, 262},
	{0x200130fa, 4, 2779096485, 8, 262},
}

var n4Stubs = []n4Stub{
	{0x4ef610, "s4ef610", 0, 0x0, []n4Rec{{1, 4, 1, 0x20000000}, {2, 4, 1, 0x20000004}}, 0x20000008, n4Wr{0, 0, 0x20060000}},
	{0x51d610, "s51d610", 300, 0x20060000, []n4Rec{}, 0x0, n4Wr{0, 0, 0x20060000}},
	{0x51c5a0, "s51c5a0", 300, 0x200604b0, []n4Rec{}, 0x0, n4Wr{0, 0, 0x20060000}},
	{0x51f900, "s51f900", 400, 0x20060960, []n4Rec{}, 0x0, n4Wr{3, 1, 0x20060fa0}},
	{0x51b840, "s51b840", 300, 0x20060fa1, []n4Rec{}, 0x0, n4Wr{0, 0, 0x20060000}},
}

const (
	n4Arena   = 0x20000000
	n4Ch      = 0x1862 // character record stride at *0x685174
	n4Pres    = 0x106  // presence record stride at *0x685178
	n4Aux     = 0x6836b8
	n4AuxStr  = 280
	n4Tmp     = 0x10006000 // a stack-temporary item instance (557430's local), not an arena object
	n4DefStr  = 269
	n4ChSlots = 8
	n4Src     = 0x20011000 // arena object "src": the replacement character record
)

type n4Grp struct {
	name   string
	kind   byte // 'g' global, 'o' arena object, 'a' stack argument, 'r' eax
	addr   uint32
	size   int
	signed bool
	count  int
	step   int
	idx    string // in-field that selects the record (stride), "" for none
	stride uint32
}
type n4Set struct {
	addr  uint32
	size  int
	val   uint32
	count int
	step  int
}
type n4Rec struct {
	arg, size, n int
	at           uint32
}
type n4Wr struct {
	arg, size int
	at        uint32
}
type n4Stub struct {
	addr    uint32
	name    string
	rets    int // >0 sequence of rets, -1 one scalar input, 0 constant retAt
	retAt   uint32
	recs    []n4Rec
	countAt uint32
	wr      n4Wr
}
type n4Fld struct {
	g *n4Grp
	k int
}

func n4Expand(gs []n4Grp) (fs []n4Fld, names []string) {
	for i := range gs {
		g := &gs[i]
		for k := 0; k < g.count; k++ {
			fs = append(fs, n4Fld{g, k})
			if g.count == 1 {
				names = append(names, g.name)
			} else {
				names = append(names, fmt.Sprintf("%s%d", g.name, k))
			}
		}
	}
	return
}

var n4Ins, n4InNames = n4Expand(n4InG)
var n4Outs, n4OutNames = n4Expand(n4OutG)

// the exe image, read at first use (tables are read from the player's Wiz8.exe, not embedded)
var n4ExeOnce sync.Once
var n4ExeB []byte
var n4ExeErr error

func n4Exe() []byte {
	n4ExeOnce.Do(func() {
		path := os.Getenv("WIZ8_EXE")
		if path == "" {
			_, source, _, _ := runtime.Caller(0)
			path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
		}
		n4ExeB, n4ExeErr = os.ReadFile(path)
	})
	if n4ExeErr != nil {
		panic("character replacement: " + n4ExeErr.Error())
	}
	return n4ExeB
}

// n4Mem is a sparse byte memory. An address never written reads as the exe image byte
// (0 in .bss and in the arena), exactly the emulator's initial memory.
type n4Mem map[uint32]byte

func (m n4Mem) b(a uint32) uint32 {
	if v, ok := m[a]; ok {
		return uint32(v)
	}
	if a >= n4Arena && a < n4Arena+0x100000 {
		return 0
	}
	e := n4Exe()
	if o, err := peOffset(e, a); err == nil && o < len(e) {
		return uint32(e[o])
	}
	return 0
}
func (m n4Mem) get(a uint32, n int) uint32 {
	var v uint32
	for i := n - 1; i >= 0; i-- {
		v = v<<8 | m.b(a+uint32(i))
	}
	return v
}
func (m n4Mem) put(a uint32, n int, v uint32) {
	for i := 0; i < n; i++ {
		m[a+uint32(i)] = byte(v >> (8 * uint(i)))
	}
}
func (m n4Mem) r8(a uint32) uint32  { return m.b(a) }
func (m n4Mem) r16(a uint32) uint32 { return m.get(a, 2) }
func (m n4Mem) r32(a uint32) uint32 { return m.get(a, 4) }
func (m n4Mem) w8(a, v uint32)      { m.put(a, 1, v) }
func (m n4Mem) w16(a, v uint32)     { m.put(a, 2, v) }
func (m n4Mem) w32(a, v uint32)     { m.put(a, 4, v) }
func (m n4Mem) copy(dst, src uint32, n int) {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = byte(m.b(src + uint32(i)))
	}
	for i, v := range buf {
		m[dst+uint32(i)] = v
	}
}
func (m n4Mem) fill(a uint32, n int, v byte) {
	for i := 0; i < n; i++ {
		m[a+uint32(i)] = v
	}
}

// n4C is one run of the node. real=false replays the case's stubs (Emu); real=true runs the
// real D4 Go of every stubbed callee whose result is read (F).
type n4C struct {
	m     n4Mem
	r     Rand
	real  bool
	ch    uint32 // character record argument
	eax   uint32
	pay   uint32
	ins   []int64
	calls map[uint32]int
}

func (c *n4C) fieldAddr(f n4Fld) uint32 {
	g := f.g
	a := g.addr + uint32(f.k*g.step)
	if g.idx != "" {
		i := c.in(g.idx)
		if i < 0 || i >= int64(n4ChSlots) {
			panic("character replacement: record index out of range")
		}
		a += uint32(i) * g.stride
	}
	return a
}

var n4InAt = func() map[string]int {
	m := map[string]int{}
	for i, n := range n4InNames {
		m[n] = i
	}
	return m
}()

func (c *n4C) in(name string) int64 { return c.ins[n4InAt[name]] }

func n4Run(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(n4Ins) {
		panic("character replacement: wrong input count")
	}
	c := &n4C{m: n4Mem{}, r: r, real: real, ins: in, calls: map[uint32]int{}}
	n4DefFixture(c.m)
	for _, s := range n4SetupG {
		for k := 0; k < s.count; k++ {
			c.m.put(s.addr+uint32(k*s.step), s.size, s.val)
		}
	}
	for i, f := range n4Ins {
		switch f.g.kind {
		case 'a':
			continue
		default:
			c.m.put(c.fieldAddr(f), f.g.size, uint32(in[i]))
		}
	}
	c.ch, c.pay = uint32(c.in("chPtr")), uint32(c.in("pay"))&0xff
	c.node()
	out := make([]int64, len(n4Outs))
	for i, f := range n4Outs {
		g := f.g
		if g.kind == 'r' {
			out[i] = int64(c.eax & 0xff)
			continue
		}
		v := c.m.get(c.fieldAddr(f), g.size)
		out[i] = int64(v)
		if g.signed {
			out[i] = int64(int32(v << (32 - 8*uint(g.size)) >> (32 - 8*uint(g.size))))
		}
	}
	return out
}

func n4Spec(addr uint32) *n4Stub {
	for i := range n4Stubs {
		if n4Stubs[i].addr == addr {
			return &n4Stubs[i]
		}
	}
	panic(fmt.Sprintf("character replacement: no stub %#x", addr))
}

// stub replays the case's address stub of a callee: records the arguments and the call
// count, returns the sequenced or scalar input (the case's stub semantics, re/emu/emu.py).
func (c *n4C) stub(addr uint32, args ...uint32) uint32 {
	s := n4Spec(addr)
	k := c.calls[addr]
	c.calls[addr] = k + 1
	for _, r := range s.recs {
		if k >= r.n {
			panic(fmt.Sprintf("character replacement: stub %s record sequence full", s.name))
		}
		c.m.put(r.at+uint32(k*r.size), r.size, args[r.arg-1])
	}
	if s.countAt != 0 {
		c.m.w32(s.countAt, c.m.r32(s.countAt)+1)
	}
	switch {
	case s.rets > 0:
		if k >= s.rets {
			panic(fmt.Sprintf("character replacement: stub %s ret sequence exhausted", s.name))
		}
		return c.m.r32(s.retAt + uint32(4*k))
	case s.rets < 0:
		return c.m.r32(s.retAt)
	}
	return s.retAt
}

// stubOut is the value a stub writes through its output pointer argument (emu "write").
func (c *n4C) stubOut(addr uint32) uint32 {
	s := n4Spec(addr)
	return c.m.get(s.wr.at, s.wr.size)
}

// n4DefFixture mirrors the item definition table of the case (gen: re/emu/cases/004ef7e0.json
// "setup", the 770 records at *0x68516c, 269 bytes each): only the fields the node reads.
func n4DefFixture(m n4Mem) {
	for i := uint32(0); i < 770; i++ {
		d := n4Arena + 0x20000 + n4DefStr*i
		m.w8(d+0x3e, i%5)
		m.w8(d+0x66, i%3)
		m.w8(d+0x67, i%5)
		m.w8(d+0x69, i%4)
		m.w8(d+0x6a, i%5+1)
		m.w8(d+0x6b, i%19+1)
		m.w32(d+0x86, i%997+1)
		m.put(d+0x76, 2, 0xffff)
		m.w32(d+0x78, 0xffffffff)
		m.w8(d+0x7c, 3)
		for _, q := range []uint32{0x7d, 0x7f, 0x81, 0x83} {
			m.w8(d+q, 0xff)
		}
	}
}

func (c *n4C) fail(what string) { panic("character replacement: srAssertFail " + what) }

func (c *n4C) pres(slot uint32) uint32   { return c.m.r32(0x685178) + n4Pres*slot }
func (c *n4C) charOf(slot uint32) uint32 { return c.m.r32(0x685174) + n4Ch*slot }

// 0x406bf0: the game clock offset.
func (c *n4C) f406bf0(x uint32) uint32 { return x + c.m.r32(0x6eb70c) }

// 0x517ec0: index of the character record in the party table (asserts on a record that is
// not in use, or not in the table).
func (c *n4C) f517ec0(ch uint32) uint32 {
	if c.m.r8(ch+4) == 0 {
		c.fail("517ec0 character not in use")
	}
	base := c.m.r32(0x685174)
	for i := uint32(0); i < 8; i++ {
		if ch == base+n4Ch*i {
			return i
		}
	}
	c.fail("517ec0 character not in party table")
	return 0
}

// 0x4e8290(slot, a, b): presence record fields +0x6d, +0x71 and the flag +0x104 (a == 1).
func (c *n4C) f4e8290(slot, a, b uint32) {
	p := c.pres(slot)
	c.m.w32(p+0x6d, a)
	c.m.w32(p+0x71, b)
	c.m.w8(p+0x104, n4b2u(a == 1))
}

func n4b2u(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// 0x54b470(slot): reset the presence record (0x106 bytes) of a joining member.
func (c *n4C) f54b470(slot uint32) {
	p := c.pres(slot)
	c.m.fill(p, 0x106, 0)
	c.m.w8(p, 1)
	c.m.w32(p+0x75, 0)
	c.m.w8(p+0xd0, 0xff)
	c.f4e8290(slot, 0, 0xffffffff)
}

// 0x54b300(slot): reset the per-member runtime record at 0x6836b8+280*slot, drawing one
// R(5000) for the +0x95 timer.
func (c *n4C) f54b300(slot uint32) {
	a := uint32(n4Aux) + n4AuxStr*slot
	m := c.m
	m.fill(a, 0x46*4, 0)
	m.w8(a, 0)
	m.w32(a+1, 0xffffffff)
	m.w32(a+0x75, 0xffffffff)
	m.w32(a+0x79, 6)
	m.w8(a+0x99, 0)
	v := uint32(2)
	if m.r32(c.charOf(slot)+0xb01) < 0xf {
		v = 1
	}
	m.w32(a+0x89, v)
	m.w32(a+0x8d, v)
	m.w32(a+0x85, 0xffffffff)
	m.w8(a+0x9a, 0)
	m.w8(a+0x9b, 0)
	m.w32(a+0x7d, c.f406bf0(0))
	m.w32(a+0x81, 0)
	m.w32(a+0x91, c.f406bf0(0))
	m.w32(a+0x95, c.f406bf0(uint32(c.r.R(5000))+0x1388))
	m.w32(a+0xca, c.f406bf0(0))
	m.w8(a+0x9c, 0)
	m.w8(a+0x9d, 0)
	m.w8(a+0x9e, 0)
	m.w32(a+0x9f, 0)
	m.w32(a+0xa3, 0xffffffff)
	m.w32(a+0xa7, 0)
	m.w8(a+0xab, 0x90)
	m.w8(a+0xbd, 0)
	m.w32(a+0xc2, 0xffffffff)
	m.w32(a+0xbe, 0xffffffff)
	m.w32(a+0xc6, 0)
	m.w32(a+0x71, 0)
	m.w8(a+0xbc, 0)
	m.w32(a+0xac, 0)
	m.w32(a+0xb0, 0)
	m.w32(a+0xb8, 0)
	m.w32(a+0xb4, 0)
	m.w8(a+0xce, 0)
	m.w8(a+0xcf, 0)
	m.w8(a+0xd0, 0)
	m.w8(a+0xd1, 0)
	m.w32(a+0xd2, c.f406bf0(0))
	m.w16(a+0xd6, 0)
	m.w8(a+0xe8, 0)
}

// ---- stubbed callees whose result the node reads -------------------------------------------
// Emu replays the case's stub (the result is a case input); F (real=true) runs the callee's D4 Go.

func (c *n4C) f51d610(ch, id uint32) bool {
	if c.real {
		return c.real51d610(ch, id)
	}
	return c.stub(0x51d610, ch, id)&0xff != 0
}

func (c *n4C) f51c5a0(ch, id uint32) uint32 {
	if c.real {
		return c.real51c5a0(ch, id)
	}
	return c.stub(0x51c5a0, ch, id)
}

func (c *n4C) f51b840(slot uint32) uint32 {
	if c.real {
		return c.real51b840(slot)
	}
	return c.stub(0x51b840, slot)
}

// f51f900 also writes its out flag (arg3, a byte) unless arg3 is null.
func (c *n4C) f51f900(slot, inst, flag uint32) bool {
	if c.real {
		return c.real51f900(slot, inst, flag)
	}
	r := c.stub(0x51f900, slot, inst, flag)&0xff != 0
	if flag != 0 {
		c.m.w8(flag, c.stubOut(0x51f900))
	}
	return r
}

// ---- F: the D4 Go nodes of the stubbed callees, fed from the memory model ---------------------
// (the item index argument of 51d610 only selects a case arena record and is not read: 0.)

func (c *n4C) real51d610(ch, id uint32) bool {
	m := c.m
	d := c.defOf(id)
	in := []int64{0, int64(m.r32(ch + 0x69)), int64(m.r32(ch + 0x71)), int64(m.r32(ch + 0x75)),
		int64(m.r16(d + 0x76)), int64(m.r32(d + 0x78)), int64(m.r8(d + 0x7c))}
	for _, q := range []uint32{0x7d, 0x7e, 0x7f, 0x80, 0x81, 0x82, 0x83, 0x84} {
		in = append(in, int64(m.r8(d+q)))
	}
	spell := m.r8(d + 0x63)
	lvl := uint32(0)
	if tbl := m.r32(0x65be1c); tbl != 0 {
		lvl = m.r32(tbl + 447*spell + 0x56)
	}
	in = append(in, int64(m.r8(d+0x42)), int64(spell), int64(m.r32(ch+0xad)), int64(m.r32(ch+0xb1)),
		int64(m.r32(ch+0xbcd+4*spell)), int64(lvl))
	for k := uint32(0); k < 7; k++ {
		in = append(in, int64(m.r32(ch+0xe9+20*k)))
	}
	for k := uint32(0); k < 41; k++ {
		in = append(in, int64(m.r32(ch+0x1a3+38*k)))
	}
	return itemUseRequirementsNode(in, c.r)[0] != 0
}

func (c *n4C) real51b840(slot uint32) uint32 {
	m := c.m
	d := c.defOf(m.r32(slot))
	return uint32(itemValueNode([]int64{int64(m.r32(slot)), int64(m.r8(slot + 4)), int64(m.r8(d + 0x3e)),
		int64(m.r8(d + 0x66)), int64(m.r32(d + 0x86))}, c.r)[0])
}

func (c *n4C) real51c5a0(ch, id uint32) uint32 {
	m := c.m
	in := []int64{int64(m.r8(0x68517c)), int64(m.r32(ch + 0x44b)), int64(id)}
	hands := []uint32{ch + 0xfa5, ch + 0xfb1, ch + 0xfbd, ch + 0xfc9}
	for _, h := range hands {
		in = append(in, int64(int32(m.r32(h))), int64(m.r32(h+4)), int64(m.r32(h+8)))
	}
	for k := uint32(0); k < autoEquipItems; k++ {
		d := c.defOf(k)
		in = append(in, int64(m.r8(d+0x3e)), int64(m.r8(d+0x41)), int64(m.r8(d+0x46)))
	}
	out := itemAutoEquipSlotNode(in, c.r)
	for i, h := range hands {
		m.w32(h, uint32(out[1+3*i]))
		m.w32(h+4, uint32(out[2+3*i]))
		m.w32(h+8, uint32(out[3+3*i]))
	}
	return uint32(int32(out[0]))
}

// 0x51f900 through itemStackMergeNode: a and b are copied into its A / B objects unless they
// are shared-pack slots (the node models the pack at its real address, 16 slots).
func (c *n4C) real51f900(slot, inst, flag uint32) bool {
	m := c.m
	const pack, count = 0x685191, 0x686901
	ptr := func(p, obj uint32) uint32 {
		if p >= pack && p < pack+12*500 {
			if (p-pack)%12 != 0 || (p-pack)/12 >= ismPool {
				panic("character replacement: pack slot past the 51f900 fixture cap")
			}
			return p
		}
		return obj
	}
	pa, pb := ptr(slot, ismPA), ptr(inst, ismPB)
	rec := func(p uint32) []int64 {
		return []int64{int64(int32(m.r32(p))), int64(m.r8(p + 4)), int64(m.r8(p + 5)), int64(m.r8(p + 6)), int64(m.r8(p + 7)), int64(m.r32(p + 8))}
	}
	pf := uint32(0)
	if flag != 0 {
		pf = ismPF
	}
	id := m.r32(slot)
	d := c.defOf(id)
	in := []int64{int64(pa), int64(pb), int64(pf), int64(int32(id))}
	for _, p := range []uint32{slot, inst} {
		in = append(in, rec(p)...)
	}
	in = append(in, -1, 0, 0, 0, 0, 0, 0) // the (empty) hand record and its mark
	for k := uint32(0); k < ismPool; k++ {
		in = append(in, rec(pack+12*k)...)
	}
	in = append(in, int64(m.r32(count)), 0xffffffff, 0xff, 0xffff, 0xffffffff, 0, 0, 0, 0,
		int64(m.r8(d+0x66)), int64(m.r8(d+0x6b)), 0, 0, 0)
	out := itemStackMergeNode(in, c.r)
	put := func(p uint32, id, tail int64) {
		m.w32(p, uint32(id))
		m.w8(p+4, uint32(tail&0xff))
		m.w8(p+5, uint32(tail>>8&0xff))
		m.w8(p+6, uint32(tail>>16&0xff))
		m.w8(p+7, uint32(tail>>24&0xff))
		m.w32(p+8, uint32(tail>>32))
	}
	if pa == ismPA {
		put(slot, out[2], out[3])
	}
	if pb == ismPB {
		put(inst, out[4], out[5])
	}
	for k := uint32(0); k < ismPool; k++ {
		put(pack+12*k, out[9+2*int(k)], out[10+2*int(k)])
	}
	m.w32(count, uint32(out[9+2*ismPool]))
	if flag != 0 {
		m.w8(flag, uint32(out[1]))
	}
	return out[0] != 0
}

// ---- item definition and instance helpers ---------------------------------------------------

func (c *n4C) defOf(id uint32) uint32 { return c.m.r32(0x68516c) + n4DefStr*id }

// 0x51c020 (initNewItem) for a stack-temporary instance with maxRoll=forceFlag6=setFlagB=1:
// the held-item and pack branches are not reachable for a temporary (see n_51c020.go for
// the general function).
func (c *n4C) f51c020(p, id uint32) {
	if id >= c.m.r32(0x683f78) {
		c.fail("51c020 uiItemNo < gXStatus.uiItemsInDatabase")
	}
	m := c.m
	m.w32(p, 0)
	m.w32(p+4, 0)
	m.w32(p+8, 0)
	m.w32(p, 0xffffffff)
	m.w32(p, id)
	d := c.defOf(id)
	if kind := m.r8(d + 0x66); kind != 0 {
		z, n, s := m.r16(d+0x67), m.r8(d+0x69), m.r8(d+0x6a)
		if z+n*s >= 0x100 {
			c.fail("51c020 dice maximum below 256")
		}
		al := (s*n + z) & 0xff
		switch {
		case kind == 1:
			m.w8(p+4, al)
		case kind <= 4:
			m.w8(p+5, al)
		default:
			c.fail("51c020 unknown stack kind")
		}
	}
	m.w8(p+6, 1)
	m.w8(p+0xb, 1)
}

// 0x51fe30(dst, src, ch, flag): move the 12-byte item src into the empty slot dst and
// empty src. (The held-item and pack-source branches are not reachable here: dst and src
// are character slots, the pack head, or a stack temporary.)
func (c *n4C) f51fe30(dst, src uint32) {
	m := c.m
	if dst == src {
		c.fail("51fe30 dst != src")
	}
	if m.r32(dst) != 0xffffffff {
		c.fail("51fe30 dst empty")
	}
	if m.r32(src) == 0xffffffff {
		c.fail("51fe30 src not empty")
	}
	if m.r8(src+0xa) != 0 {
		m.w8(src+0xa, 0)
	}
	m.copy(dst, src, 12)
	m.w32(src, 0)
	m.w32(src+4, 0)
	m.w32(src+8, 0)
	m.w32(src, 0xffffffff)
}

// 0x520310(ch): empty the 12 + 8 inventory slots of a character.
func (c *n4C) f520310(ch uint32) {
	for k := uint32(0); k < 20; k++ {
		p := ch + 0xf5d + 12*k
		if k >= 12 {
			p = ch + 0x1029 + 12*(k-12)
		}
		c.m.w32(p, 0)
		c.m.w32(p+4, 0)
		c.m.w32(p+8, 0)
		c.m.w32(p, 0xffffffff)
	}
}

// 0x51c300(ch, inst, first, 0, 0): put a temporary item into the character. first (arg3): try
// the auto-equip slot first (then recompute the derived stats, 50e5c0). Then the stacking scan
// over the 12 + 8 slots (a stackable kind-1 item; arg5 = 0 so it runs), else the first empty
// backpack slot. The found tail works on backpack entry [ch+0x1029+12*idx] for both scans
// (idx is the scan index, so a hit in the 12-slot scan names a backpack entry: original quirk).
func (c *n4C) f51c300(ch, inst uint32, first bool) bool {
	m := c.m
	id := m.r32(inst)
	if first && c.f51d610(ch, id) {
		if k := c.f51c5a0(ch, id); k != 0xffffffff {
			if k > 11 {
				c.fail("51c300 equipment slot index past the 12 slots")
			}
			p := ch + 0xf5d + 12*k
			var res bool
			if m.r32(p) == 0xffffffff {
				c.f51fe30(p, inst)
				res = true
			} else {
				res = c.f51f900(p, inst, 0)
			}
			if res {
				if m.r8(ch+4) != 0 {
					c.f50e540(ch) // 50e5c0(slot)
				}
				return true
			}
		}
	}
	idx := uint32(0xffffffff)
	if m.r8(c.defOf(id)+0x66) == 1 {
		for k := uint32(0); k < 12 && idx == 0xffffffff; k++ {
			if c.f51f900(ch+0xf5d+12*k, inst, 0) {
				idx = k
			}
		}
		for k := uint32(0); k < 8 && idx == 0xffffffff; k++ {
			if c.f51f900(ch+0x1029+12*k, inst, 0) {
				idx = k
			}
		}
	}
	if idx == 0xffffffff {
		k := uint32(0)
		for k < 8 && m.r32(ch+0x1029+12*k) != 0xffffffff {
			k++
		}
		if k == 8 {
			return false
		}
		idx = k
		c.f51fe30(ch+0x1029+12*k, inst)
	}
	// found: 4edc60 / 4edd20 (no stub, no Out), mode 6 bit, 522640, 5227d0
	if m.r32(0x68ec78) == 6 {
		g := m.r32(0x69c0f4) + 0xfc
		m.w32(g, m.r32(g)|2<<idx)
	}
	e := ch + 0x1029 + 12*idx
	c.f522640(e)
	c.f5227d0(e, false, ch)
	return true
}

// 0x51d960(ch): equipment sanity of the 12 slots: an item the character may not carry
// (51d610 false) is lifted into a temporary, put back via 51c300 (arg3 = 0) or, failing that, filed
// into the pack (521ef0). The refusal message (590950) is a native print outside Out.
func (c *n4C) f51d960(ch uint32) {
	m := c.m
	for k := uint32(0); k < 12; k++ {
		p := ch + 0xf5d + 12*k
		id := m.r32(p)
		if id == 0xffffffff || c.f51d610(ch, id) {
			continue
		}
		t := uint32(n4Tmp + 0x20)
		m.w32(t, 0xffffffff)
		m.w32(t+4, 0)
		m.w32(t+8, 0)
		c.f51fe30(t, p)
		if !c.f51c300(ch, t, false) {
			c.f521ef0(t, 0)
		}
	}
}

// Skill bonus bytes of a character (ch+0x1770+0x13+k, k < 0x29): the byte-wise sum of the
// equipment buffer (ch+0x1709, rebuilt by 50e980), the buffer at ch+0x16a2 and, for a present
// character, the global buffer 0x687453 (50f030 / 50f090). The other bytes of those buffers are
// outside Out and not read by the modelled code.
const n4Skills = 0x29

func (c *n4C) f50f030(ch uint32) {
	m := c.m
	for k := uint32(0); k < n4Skills; k++ {
		v := m.r8(ch+0x1709+0x13+k) + m.r8(ch+0x16a2+0x13+k)
		if m.r8(ch+4) != 0 {
			v += m.r8(0x687453 + 0x13 + k)
		}
		m.w8(ch+0x1770+0x13+k, v&0xff)
	}
}

// 0x50e980(ch, ch+0x1709): the equipment skill bonuses (def +0xb1/+0xb2 into byte 0x13+skill,
// def +0xb3/+0xb4 into byte 0xc+skill) of the worn slots (all but 8 and 9). A skill index
// outside 0..0x28 would write past the skill bytes into the neighbouring record fields:
// panic (the item table has none).
func (c *n4C) f50e980(ch uint32) {
	m := c.m
	var b [0x67]uint32
	for k := uint32(0); k < 12; k++ {
		id := m.r32(ch + 0xf5d + 12*k)
		if k == 8 || k == 9 || id == 0xffffffff {
			continue
		}
		d := c.defOf(id)
		for _, q := range [][3]uint32{{0xb1, 0xb2, 0x13}, {0xb3, 0xb4, 0xc}} {
			if s := m.r8(d + q[0]); s != 0xff {
				if s >= n4Skills {
					panic(fmt.Sprintf("character replacement: item %#x skill bonus index %#x not modelled", id, s))
				}
				b[q[2]+s] += m.r8(d + q[1])
			}
		}
	}
	for k := uint32(0); k < n4Skills; k++ {
		m.w8(ch+0x1709+0x13+k, b[0x13+k]&0xff)
	}
}

// 0x50e5c0(slot) / 0x50e540(ch): rebuild the equipment and total bonus buffers, then 4ed9d0.
func (c *n4C) f50e540(ch uint32) {
	c.f50e980(ch)
	c.f50f030(ch)
	c.f4ed9d0(ch)
}

// 0x4ed9d0(ch): derived stats. Modelled: the clamps it applies to ch fields, 5539e0 (7 x 51d960)
// and 553a60 (the 41 skill values at ch+0x1a3+0x26*k, each followed by 51d960). The other stat
// arithmetic (name buffers, ch+0xe9 tables, 52a2f0..4ee9d0 ...) is outside Out and not read by
// the modelled code.
func (c *n4C) f4ed9d0(ch uint32) {
	m := c.m
	if int32(m.r32(ch+0x69)) >= 0xf || int32(m.r32(ch+0x71)) >= 0x10 || m.r32(ch+0x75) == 0xffffffff {
		return
	}
	if m.r32(ch+0x89) > 0x32 {
		m.w32(ch+0x89, 0x32)
	}
	for k := uint32(0); k < 15; k++ {
		if m.r32(ch+0x8d+4*k) > 0x32 {
			m.w32(ch+0x8d+4*k, 0x32)
		}
	}
	for k := 0; k < 7; k++ {
		c.f51d960(ch)
	}
	fav := m.r32(0x6161dc + 4*m.r32(ch+0x69))
	for k := uint32(0); k < n4Skills; k++ {
		v := m.r32(ch + 0x19f + 0x26*k)
		if k == fav {
			d := uint32(uint64(0x51eb851f) * uint64(v*25) >> 32 >> 5)
			if d == 0 {
				d = 1
			}
			v += d
		}
		v += uint32(int32(int8(m.r8(ch + 0x1783 + k))))
		if int32(v) > 0x7d {
			v = 0x7d
		} else if int32(v) < 0 {
			v = 0
		}
		m.w32(ch+0x1a3+0x26*k, v)
		c.f51d960(ch)
	}
}

// 0x518150(0, 0, -1, -1): the random party member, from the party tables in memory.
func (c *n4C) f518150() int64 {
	m := c.m
	in := []int64{0, 0, -1, -1}
	var pf, fl, lv, ky []int64
	for i := uint32(0); i < 8; i++ {
		ch := m.r32(0x685174) + n4Ch*i
		pf = append(pf, int64(m.r8(m.r32(0x685178)+n4Pres*i)))
		fl = append(fl, int64(m.r32(ch+0xb11)))
		lv = append(lv, int64(m.r32(ch+0xb01)))
		ky = append(ky, int64(int32(m.r32(ch+0x75))))
	}
	for _, v := range [][]int64{pf, fl, lv, ky} {
		in = append(in, v...)
	}
	return randomPartyMemberNode(in, c.r)[0]
}

// 0x522640(item): story-flag bookkeeping for the quest items. The ids it handles are NOT
// modelled (their effects are story flags outside Out): panic instead of a silent difference.
func (c *n4C) f522640(e uint32) {
	switch id := c.m.r32(e); id {
	case 0x1d2, 0x201, 0x239, 0x242, 0x243, 0x244, 0x264, 0x27c:
		panic(fmt.Sprintf("character replacement: 522640/5227d0 story item %#x not modelled", id))
	}
}

// 0x5227d0(item, rnd, ch): rnd=true (pack entries) draws a random party member (R(8) in
// 518150) and does nothing else for a non-story item; rnd=false (a kit item put into ch) marks
// the item (byte +9 bit 0) when its value / [ch+0x89] exceeds 500 and the item byte +6 is set.
func (c *n4C) f5227d0(e uint32, rnd bool, ch uint32) {
	m := c.m
	if m.r32(0x68ec78) == 5 {
		return
	}
	if rnd {
		if c.f518150() == -1 {
			return
		}
		return // the story items (the only others acting for rnd) panic in f522640
	}
	if m.r8(e+9)&1 != 0 {
		return
	}
	id := m.r32(e)
	if !c.f51d610(ch, id) {
		return
	}
	d := m.r32(ch + 0x89)
	if d == 0 {
		c.fail("5227d0 division by zero [ch+0x89]")
	}
	if m.r32(c.defOf(id)+0x86)/d <= 0x1f4 || m.r8(e+6) == 0 {
		return
	}
	m.w8(e+9, m.r8(e+9)|1)
}

// 0x557430(ch): the starting kit of the character's class, then the class extra.
func (c *n4C) f557430(ch uint32) {
	m := c.m
	c.f520310(ch)
	cls := m.r32(ch + 0x69)
	if m.r32(ch+0x71) == 5 {
		cls = 0xf
	}
	for k := uint32(0); k < 6; k++ {
		id := m.r32(0x61635c + 24*cls + 4*k)
		if id != 0xffffffff {
			c.f51c020(n4Tmp, id)
			c.f51c300(ch, n4Tmp, true)
		}
	}
	cls = m.r32(ch + 0x69)
	if cls > 10 {
		c.f50e540(ch)
		return
	}
	var id uint32
	switch m.r32(0x557558 + 4*m.r8(0x55756c+cls)) {
	case 0x5574b1:
		id = 0x52
		if m.r32(ch+0x215) > m.r32(ch+0x261) {
			id = 0x16
		}
	case 0x5574cd:
		id = 7
		if m.r32(ch+0x1c9) > m.r32(ch+0x1a3) {
			id = 0x12
		}
	case 0x5574fe:
		id = 0x257
	case 0x557520:
		id = 0x144
	default:
		c.f50e540(ch)
		return
	}
	c.f51c020(n4Tmp, id)
	c.f51c300(ch, n4Tmp, true)
	c.f50e540(ch)
}

// 0x558640(ch): the price of the kit items (and the character's emptied inventory).
func (c *n4C) f558640(ch uint32) uint32 {
	c.f557430(ch)
	var sum uint32
	for k := uint32(0); k < 20; k++ {
		p := ch + 0xf5d + 12*k
		if k >= 12 {
			p = ch + 0x1029 + 12*(k-12)
		}
		if c.m.r32(p) != 0xffffffff {
			sum += c.f51b840(p)
		}
	}
	c.f520310(ch)
	return sum
}

// 0x521ef0(item, 0, nomerge): file a backpack item into the shared pack, stacking onto a
// matching pack entry when possible, then the tail (mode 6 bit, 522640, 5227d0). Message
// argument 0: 58aad0 is not called.
func (c *n4C) f521ef0(item, nomerge uint32) {
	m := c.m
	const pack, count = 0x685191, 0x686901
	flag := uint32(n4Tmp + 0x80)
	m.w8(flag, 0)
	idx := -1
	if m.r8(c.defOf(m.r32(item))+0x66) == 1 && nomerge == 0 {
		for i := uint32(0); i < m.r32(count); i++ {
			if c.f51f900(pack+12*i, item, flag) {
				idx = int(i)
				break
			}
		}
	}
	if idx < 0 {
		n := m.r32(count)
		if n >= 0x1f4 {
			return
		}
		if n > 0 {
			m.copy(pack+12, pack, int(12*n))
		}
		m.w32(pack, 0)
		m.w32(pack+4, 0)
		m.w32(pack+8, 0)
		m.w32(pack, 0xffffffff)
		c.f51fe30(pack, item)
		m.w32(count, n+1)
		idx = 0
	}
	if m.r32(0x68ec78) == 6 {
		g := m.r32(0x69c0f4) + 0xfc
		m.w32(g, m.r32(g)|0x7fc00000)
	}
	e := pack + 12*uint32(idx)
	c.f522640(e)
	c.f5227d0(e, true, 0)
}

// ---- the node ---------------------------------------------------------------------------

func (c *n4C) node() {
	m := c.m
	ch := c.ch
	slot := c.f517ec0(ch)
	for k := uint32(0); k < 12; k++ {
		if p := ch + 0xf5d + 12*k; m.r32(p) != 0xffffffff {
			c.f521ef0(p, 0)
		}
	}
	for k := uint32(0); k < 8; k++ {
		if p := ch + 0x1029 + 12*k; m.r32(p) != 0xffffffff {
			c.f521ef0(p, 0)
		}
	}
	c.stub(0x4ef610, slot, 0)
	m.copy(ch, n4Src, 0x618*4+2)
	m.w8(ch+4, 1)
	c.f54b470(slot)
	c.f54b300(slot)
	pr := c.pres(slot)
	m.w32(pr+0xfa, 0xffffffff)
	for i := uint32(0); i < 8; i++ {
		if m.r32(0x686a50+4*i) == 0xffffffff {
			m.w32(0x686a50+4*i, slot)
			m.w32(pr+0xf1, i)
			break
		}
	}
	m.w8(0x687526+12*slot, 0xff)
	m.w32(0x685185, m.r32(0x685185)+1)
	m.w32(0x68517d, m.r32(0x68517d)+1)
	c.f50f030(ch)
	c.f4ed9d0(ch)
	if c.pay != 0 {
		cost := c.f558640(ch)
		if cost > m.r32(0x685189) {
			c.fail("4ef7e0 cost <= gold")
		}
		m.w32(0x685189, m.r32(0x685189)-cost)
		c.f557430(ch)
	}
	c.eax = slot
}

// characterReplacementGoldCostNode is F: the D4 Go of the stubbed callees runs instead of the
// case results.
func characterReplacementGoldCostNode(in []int64, r Rand) []int64 { return n4Run(in, r, true) }

func init() {
	Nodes["004ef7e0"] = Node{Addr: "004ef7e0", In: n4InNames, Out: n4OutNames,
		F:   characterReplacementGoldCostNode,
		Emu: func(in []int64, r Rand) []int64 { return n4Run(in, r, false) }}
}
