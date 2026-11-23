package calc

import (
	"fmt"
	"math"
	"math/big"
)

// character_turn_recovery_and_status 0x503100(who, ticks): per-turn damage, random
// attribute/HP/stamina loss under condition 2, condition 10 and the 0x243/0x239 ward,
// per-tick HP/stamina/SP rates, fractional regeneration (+0xb69..+0xb7d), condition and
// timed-effect countdowns, and the feature-0x12 rest hook. Record C = *0x685174 +
// who*0x1862, party presence bytes at *0x685178 + 0x106*j (build/wiz8.asm:349463..).
//
// Emu address-stubs every D4 formula callee (52a890, 522fe0, 523330, 52add0, 52b1c0,
// 52af50, 52b4f0, 52b6d0, 5236a0, 523a80, 548e60, 51d960, 4ed9d0; ruling 2026-10-09) and
// the 排除 presentation calls 5b1af0, 5b1b30, 569a50, 58aad0: arguments and call counts
// are outputs, every return is ignored by the caller. 590950, 58b410, 517a90 and 562a50
// run natively in the case and write nothing in Out. F runs the real Go of the D4
// callees on the node's memory and feeds every mapped output back (realCall).
//
// Modelled helpers: 553ad0 (attribute change: flag byte, derived attribute clamp,
// 41 skill averages from the table at 0x6155b8), 521060/520f90 (item lookup), 42b740 ->
// 4d9700 -> 506280 -> 5080f0 (selector key, story facts 0x5b and 0x1ce), 547940/5479b0
// (feature rows, level scale) and 55ee30 (mode 6 screen dirty bit). R(5) above 4 (blocks
// 5031ec, 5031f3) cannot happen and panics. Fixture caps: who 0..7, class -1..14, race
// -1..15, backpack records 0..6.

var ctrsInG = []n4Grp{
	{"who", 'a', 0x0, 4, false, 1, 4, "", 0x0},
	{"ticks", 'a', 0x1, 4, false, 1, 4, "", 0x0},
	{"b1778", 'o', 0x20001778, 1, false, 1, 4, "who", 0x1862},
	{"r1779", 'o', 0x20001779, 1, true, 1, 4, "who", 0x1862},
	{"r177a", 'o', 0x2000177a, 1, true, 1, 4, "who", 0x1862},
	{"r177b", 'o', 0x2000177b, 1, true, 1, 4, "who", 0x1862},
	{"m177c", 'o', 0x2000177c, 1, true, 7, 1, "who", 0x1862},
	{"s", 'o', 0x20000a01, 4, false, 20, 4, "who", 0x1862},
	{"e5", 'o', 0x200000e5, 4, false, 7, 20, "who", 0x1862},
	{"e9", 'o', 0x200000e9, 4, false, 7, 20, "who", 0x1862},
	{"ed", 'o', 0x200000ed, 4, false, 7, 20, "who", 0x1862},
	{"fl", 'o', 0x200006a9, 1, false, 7, 38, "who", 0x1862},
	{"b0d", 'o', 0x20000b0d, 4, false, 1, 4, "who", 0x1862},
	{"b11", 'o', 0x20000b11, 4, false, 1, 4, "who", 0x1862},
	{"b15", 'o', 0x20000b15, 4, false, 1, 4, "who", 0x1862},
	{"b19", 'o', 0x20000b19, 4, false, 1, 4, "who", 0x1862},
	{"b1d", 'o', 0x20000b1d, 4, true, 1, 4, "who", 0x1862},
	{"b21", 'o', 0x20000b21, 4, false, 1, 4, "who", 0x1862},
	{"b25", 'o', 0x20000b25, 4, true, 6, 4, "who", 0x1862},
	{"b45", 'o', 0x20000b45, 4, true, 6, 4, "who", 0x1862},
	{"b65", 'o', 0x20000b65, 4, false, 1, 4, "who", 0x1862},
	{"b69", 'o', 0x20000b69, 4, false, 1, 4, "who", 0x1862},
	{"b6d", 'o', 0x20000b6d, 4, false, 1, 4, "who", 0x1862},
	{"b71", 'o', 0x20000b71, 4, false, 1, 4, "who", 0x1862},
	{"b75", 'o', 0x20000b75, 4, false, 1, 4, "who", 0x1862},
	{"b79", 'o', 0x20000b79, 4, false, 6, 8, "who", 0x1862},
	{"b7d", 'o', 0x20000b7d, 4, false, 6, 8, "who", 0x1862},
	{"a6d", 'o', 0x20000a6d, 4, false, 8, 12, "who", 0x1862},
	{"cls", 'o', 0x20000069, 4, true, 1, 4, "who", 0x1862},
	{"race", 'o', 0x20000071, 4, true, 1, 4, "who", 0x1862},
	{"lvl", 'o', 0x20000089, 4, false, 1, 4, "who", 0x1862},
	{"lvls", 'o', 0x2000008d, 4, false, 15, 4, "who", 0x1862},
	{"it0_", 'o', 0x20000f5d, 4, false, 12, 12, "", 0x0},
	{"eq0_", 'o', 0x20001029, 4, false, 8, 12, "", 0x0},
	{"it1_", 'o', 0x200027bf, 4, false, 12, 12, "", 0x0},
	{"eq1_", 'o', 0x2000288b, 4, false, 8, 12, "", 0x0},
	{"it2_", 'o', 0x20004021, 4, false, 12, 12, "", 0x0},
	{"eq2_", 'o', 0x200040ed, 4, false, 8, 12, "", 0x0},
	{"it3_", 'o', 0x20005883, 4, false, 12, 12, "", 0x0},
	{"eq3_", 'o', 0x2000594f, 4, false, 8, 12, "", 0x0},
	{"it4_", 'o', 0x200070e5, 4, false, 12, 12, "", 0x0},
	{"eq4_", 'o', 0x200071b1, 4, false, 8, 12, "", 0x0},
	{"it5_", 'o', 0x20008947, 4, false, 12, 12, "", 0x0},
	{"eq5_", 'o', 0x20008a13, 4, false, 8, 12, "", 0x0},
	{"it6_", 'o', 0x2000a1a9, 4, false, 12, 12, "", 0x0},
	{"eq6_", 'o', 0x2000a275, 4, false, 8, 12, "", 0x0},
	{"it7_", 'o', 0x2000ba0b, 4, false, 12, 12, "", 0x0},
	{"eq7_", 'o', 0x2000bad7, 4, false, 8, 12, "", 0x0},
	{"pres", 'o', 0x20010000, 1, false, 8, 262, "", 0x0},
	{"mode", 'g', 0x687509, 4, false, 1, 4, "", 0x0},
	{"combat", 'g', 0x683f94, 1, false, 1, 4, "", 0x0},
	{"rest", 'g', 0x683fc5, 1, false, 1, 4, "", 0x0},
	{"ui", 'g', 0x68ec78, 4, false, 1, 4, "", 0x0},
	{"sel", 'g', 0x686a70, 4, false, 1, 4, "", 0x0},
	{"f1ce", 'g', 0x689d46, 1, false, 1, 4, "", 0x0},
	{"curF", 'g', 0x6874ca, 1, false, 1, 4, "", 0x0},
	{"curId", 'g', 0x6874cb, 4, false, 1, 4, "", 0x0},
	{"bpn", 'g', 0x686901, 4, false, 1, 4, "", 0x0},
	{"bp", 'g', 0x685191, 4, false, 6, 12, "", 0x0},
	{"scr", 'o', 0x200120f8, 4, false, 1, 4, "", 0x0},
}

// ctrsStub is one address stub of the case: call count at `at`, argument a of call k at
// at+4+4*n*(a-1)+4*k.
type ctrsStub struct {
	addr, at uint32
	nargs, n int
}

var ctrsStubs = []ctrsStub{
	{0x52a890, 0x20011000, 7, 3}, {0x522fe0, 0x20011058, 6, 3}, {0x523330, 0x200110a4, 3, 2},
	{0x52add0, 0x200110c0, 3, 2}, {0x52b1c0, 0x200110dc, 3, 2}, {0x52af50, 0x200110f8, 4, 1},
	{0x52b4f0, 0x2001110c, 3, 12}, {0x52b6d0, 0x200111a0, 4, 6}, {0x5236a0, 0x20011204, 3, 20},
	{0x523a80, 0x200112f8, 2, 8}, {0x548e60, 0x2001133c, 1, 1}, {0x51d960, 0x20011344, 1, 1},
	{0x4ed9d0, 0x2001134c, 1, 1}, {0x5b1af0, 0x20011354, 1, 1}, {0x5b1b30, 0x2001135c, 1, 1},
	{0x569a50, 0x20011364, 4, 1}, {0x58aad0, 0x20011378, 3, 3},
}

var ctrsOutG = func() []n4Grp {
	w := func(name string, off uint32, size int, signed bool, count, step int) n4Grp {
		return n4Grp{name, 'o', n4Arena + off, size, signed, count, step, "who", ctrsStride}
	}
	g := []n4Grp{w("o.b0d", 0xb0d, 4, false, 1, 4), w("o.b11", 0xb11, 4, false, 1, 4), w("o.b15", 0xb15, 4, false, 1, 4),
		w("o.b19", 0xb19, 4, false, 1, 4), w("o.b1d", 0xb1d, 4, true, 1, 4), w("o.b21", 0xb21, 4, false, 1, 4),
		w("o.e5", 0xe5, 4, false, 7, 20), w("o.e9", 0xe9, 4, false, 7, 20), w("o.ed", 0xed, 4, false, 7, 20),
		w("o.fl", 0x6a9, 1, false, 7, 38), w("o.sk", 0x1a7, 4, false, 41, 38), w("o.b6d", 0xb6d, 4, false, 1, 4),
		w("o.b75", 0xb75, 4, false, 1, 4), w("o.b7d", 0xb7d, 4, false, 6, 8), w("o.a6d", 0xa6d, 4, false, 8, 12),
		w("o.b65", 0xb65, 4, false, 1, 4), {"o.scr", 'o', ctrsScreen + 0xf8, 4, false, 1, 4, "", 0}}
	for _, s := range ctrsStubs {
		g = append(g, n4Grp{fmt.Sprintf("n%x", s.addr), 'o', s.at, 4, false, 1, 4, "", 0})
		for a := 1; a <= s.nargs; a++ {
			g = append(g, n4Grp{fmt.Sprintf("a%x_%d_", s.addr, a), 'o', s.at + 4 + uint32(4*s.n*(a-1)), 4, false, s.n, 4, "", 0})
		}
	}
	return g
}()

const (
	ctrsStride  = 0x1862     // character record size (*0x685174)
	ctrsSlots   = 0x20010000 // case object "slots": presence records, 0x106 bytes each
	ctrsScreen  = 0x20012000 // case object "screen" (*0x69c0f4)
	ctrsCursor  = 0x6874cb   // item on the mouse cursor (byte 0x6874ca says held)
	ctrsBpMax   = 6          // backpack records in the case (fixture cap)
	ctrsNever   = 9999       // 0x270f: condition duration "permanent"
	ctrsLogBuf  = 0x689fd0   // 0x517a90 returns its static text buffer
	ctrsDebugOn = 0x688290   // 0x506280 logs story-fact reads when this byte is set
)

var ctrsIns, ctrsInNames = n4Expand(ctrsInG)
var ctrsOuts, ctrsOutNames = n4Expand(ctrsOutG)

type ctrsC struct {
	m          n4Mem
	r          Rand
	real       bool
	ins        []int64
	calls      map[uint32]int
	who, ticks uint32
	ch         uint32
}

func (c *ctrsC) fieldAddr(f n4Fld) uint32 {
	a := f.g.addr + uint32(f.k*f.g.step)
	if f.g.idx != "" {
		a += c.who * f.g.stride
	}
	return a
}

func (c *ctrsC) f32(a uint32) float32 { return math.Float32frombits(c.m.r32(a)) }

func ctrsRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(ctrsIns) {
		panic("turn recovery: wrong input count")
	}
	who := uint32(in[0])
	if who >= 8 {
		panic("turn recovery: party slot outside 0..7")
	}
	c := &ctrsC{m: n4Mem{}, r: r, real: real, ins: in, calls: map[uint32]int{}, who: who, ticks: uint32(in[1])}
	m := c.m
	m.w32(0x685174, n4Arena)
	m.w32(0x685178, ctrsSlots)
	m.w32(0x69c0f4, ctrsScreen)
	ctrsDecoys(m)
	for i, f := range ctrsIns {
		if f.g.kind != 'a' {
			m.put(c.fieldAddr(f), f.g.size, uint32(in[i]))
		}
	}
	c.ch = n4Arena + who*ctrsStride
	c.node()
	out := make([]int64, len(ctrsOuts))
	for i, f := range ctrsOuts {
		v := m.get(c.fieldAddr(f), f.g.size)
		out[i] = int64(v)
		if f.g.signed {
			out[i] = int64(int32(v))
		}
	}
	return out
}

// ctrsDecoys mirrors the case setup: every record's who-indexed fields (and the 41 skill
// words at +0x1a7) hold values that differ per record, so a wrong record index shows.
func ctrsDecoys(m n4Mem) {
	decoy := uint32(0)
	for j := uint32(0); j < 8; j++ {
		base := n4Arena + j*ctrsStride
		m.w8(base+4, 1)
		m.w16(base+5, 65+j)
		for _, g := range append(ctrsInG, n4Grp{"sk", 'o', n4Arena + 0x1a7, 4, false, 41, 38, "who", ctrsStride}) {
			if g.idx == "" {
				continue
			}
			for k := 0; k < g.count; k++ {
				decoy++
				v := 0x11*(j+1) + 7*uint32(k) + decoy
				switch g.name {
				case "b69", "b6d", "b71", "b75", "b79", "b7d":
					v = 0x3f000000 + decoy*0x1000
				case "cls", "race":
					v = j
				case "sk":
					v = 0x5000 + 0x100*j + uint32(k)
				}
				m.put(g.addr+uint32(k*g.step)+j*ctrsStride, g.size, v)
			}
		}
	}
}

// call is a D4 or 排除 callee: both modes record the stub (arguments, count); F also
// runs the real Go of a D4 callee.
func (c *ctrsC) call(addr uint32, args ...uint32) {
	if c.real {
		defer c.realCall(addr, args)
	}
	for _, s := range ctrsStubs {
		if s.addr != addr {
			continue
		}
		k := c.calls[addr]
		c.calls[addr] = k + 1
		if k >= s.n {
			panic(fmt.Sprintf("turn recovery: stub %x record sequence full", addr))
		}
		for a := 1; a <= s.nargs; a++ {
			c.m.w32(s.at+4+uint32(4*s.n*(a-1))+uint32(4*k), args[a-1])
		}
		c.m.w32(s.at, c.m.r32(s.at)+1)
		return
	}
	panic(fmt.Sprintf("turn recovery: no stub %x", addr))
}

func (c *ctrsC) rnd(n uint32) uint32 { return uint32(c.r.R(int(n))) }

// 0x547940(C, code): class row 0x61507c+12*class (3 codes), race row 0x615130+20*race
// (5 codes), -1 skips a row; code 0x1c (never asked here) also reads +0xa79.
func (c *ctrsC) feat(code uint32) bool {
	m, ch := c.m, c.ch
	cls, race := int32(m.r32(ch+0x69)), int32(m.r32(ch+0x71))
	if cls < -1 || cls > 14 || race < -1 || race > 15 {
		panic("turn recovery: class or race outside the fixture range")
	}
	if cls != -1 {
		for k := uint32(0); k < 3; k++ {
			if m.r32(0x61507c+12*uint32(cls)+4*k) == code {
				return true
			}
		}
	}
	if race != -1 {
		for k := uint32(0); k < 5; k++ {
			if m.r32(0x615130+20*uint32(race)+4*k) == code {
				return true
			}
		}
	}
	return false
}

// 0x5479b0(C, which, m): level L = +0x8d+4*class (+0x89 for class -1); L > 20 gives m,
// else (2L + f32 at 0x5ee800) * m * f32 at 0x5ed490, all in x87 extended precision.
func (c *ctrsC) levelScale(m float32) *big.Float {
	cls := int32(c.m.r32(c.ch + 0x69))
	l := c.m.r32(c.ch + uint32(0x8d+4*cls))
	if l > 20 {
		return strcBig(m)
	}
	two := chdAdd(chdF(float64(l)), chdF(float64(l)))
	return chdMul(chdMul(chdAdd(two, strcBig(c.f32(0x5ee800))), strcBig(m)), strcBig(c.f32(0x5ed490)))
}

// 0x520f90(Cj, id, locp, mode, 0): list A (+0xf5d, 12 entries of 12 bytes), then for a
// non-zero mode list B (+0x1029, 8 entries). Returns the matching entry address or 0.
func (c *ctrsC) findOn(cj, id uint32, both bool) uint32 {
	for k := uint32(0); k < 12; k++ {
		if c.m.r32(cj+0xf5d+12*k) == id {
			return cj + 0xf5d + 12*k
		}
	}
	if both {
		for k := uint32(0); k < 8; k++ {
			if c.m.r32(cj+0x1029+12*k) == id {
				return cj + 0x1029 + 12*k
			}
		}
	}
	return 0
}

// 0x521060(id, locp, ownp, 2, 0): cursor item, then present members 0..7, then the
// backpack (0x686901 records at 0x685191, 12 bytes). Returns entry and owner record.
func (c *ctrsC) findItem(id uint32) (loc, owner uint32, ok bool) {
	m := c.m
	if m.r8(0x6874ca) != 0 && m.r32(ctrsCursor) == id {
		return ctrsCursor, 0, true
	}
	for j := uint32(0); j < 8; j++ {
		if m.r8(m.r32(0x685178)+0x106*j) == 0 {
			continue
		}
		cj := m.r32(0x685174) + ctrsStride*j
		if e := c.findOn(cj, id, true); e != 0 {
			return e, cj, true
		}
	}
	n := m.r32(0x686901)
	for k := uint32(0); k < n; k++ {
		if k >= ctrsBpMax {
			panic("turn recovery: backpack record count past the fixture cap")
		}
		if m.r32(0x685191+12*k) == id {
			return 0x685191 + 12*k, 0, true
		}
	}
	return 0, 0, false
}

// 0x42b740(sel) -> 0x4d9700: selector 0x10/0x26 becomes 0x10 when story fact 0x5b (at
// least two of items 0x242..0x244 held, 0x5080f0) or 0x1ce (byte 0x689b78+0x1ce) holds,
// else 0x26; 0..0x2f index byte 0x6044e2+0x6b*s, anything else gives 0.
func (c *ctrsC) selKey() uint32 {
	m := c.m
	if m.r8(ctrsDebugOn) != 0 {
		panic("turn recovery: 0x506280 debug log not modelled")
	}
	s := int32(m.r32(0x686a70))
	if s == 0x10 || s == 0x26 {
		held := 0
		for id := uint32(0x242); id <= 0x244; id++ {
			if _, _, ok := c.findItem(id); ok {
				held++
			}
		}
		s = 0x26
		if held >= 2 || m.r8(0x689b78+0x1ce) != 0 {
			s = 0x10
		}
	}
	if s < 0 || s > 0x2f {
		return 0
	}
	return m.r8(0x6044e2 + 0x6b*uint32(s))
}

// 0x55ee30(who): interface mode 6 marks the screen (+0xf8 bit 0x100); mode 7 marks the
// member on the battle panel (0x562a50, outside Out).
func (c *ctrsC) refresh() {
	if c.m.r32(0x68ec78) == 6 {
		s := c.m.r32(0x69c0f4) + 0xf8
		c.m.w32(s, c.m.r32(s)|0x100)
	}
}

// 0x553ad0(C, i): attribute i changed. Flag byte +0x19d+38*(i+0x22) tracks "at least
// 100"; derived +0xe9+20i = clamp(+0xe5+20i + i8 +0x177c+i, 1, 0x7d); then 51d960 and
// 4ed9d0 rebuild, and each of the 41 skills (table 0x6155b8, 16 bytes: attribute a at +0,
// b at +4) is (attr a + attr b) >> 1.
func (c *ctrsC) attrChanged(i uint32) {
	m, ch := c.m, c.ch
	ui := m.r32(0x68ec78)
	fl := ch + 0x19d + 38*(i+0x22)
	v := m.r32(ch + 0xe5 + 20*i)
	if v >= 100 {
		if m.r8(fl) == 0 {
			m.w8(fl, 1)
			if ui == 3 {
				c.call(0x5b1af0, i+0x22)
			}
			if ui == 7 {
				c.call(0x569a50, ctrsLogBuf, 0, 1, 0)
			}
		}
	} else if m.r8(fl) != 0 {
		m.w8(fl, 0)
		if ui == 3 {
			c.call(0x5b1b30, i+0x22)
		}
	}
	d := int32(int8(m.r8(ch+0x177c+i))) + int32(v)
	if d > 0x7d {
		d = 0x7d
	} else if d < 1 {
		d = 1
	}
	m.w32(ch+0xe9+20*i, uint32(d))
	c.call(0x51d960, ch)
	c.call(0x4ed9d0, ch)
	for k := uint32(0); k < 41; k++ {
		t := 0x6155b8 + 16*k
		a, b := m.r32(ch+0xe5+20*m.r32(t)), m.r32(ch+0xe5+20*m.r32(t+4))
		m.w32(ch+0x1a7+38*k, (a+b)>>1)
	}
}

// frac is one fractional regeneration: f = ticks*rate*scale + f (x87), the whole part
// (truncated, low 32 bits) goes to the callee, the stored binary32 keeps its fraction.
func (c *ctrsC) frac(rate, fr uint32, scale float32, heal func(n uint32)) {
	sum := chdAdd(chdMul(chdMul(chdF(float64(c.ticks)), strcBig(c.f32(rate))), strcBig(scale)), strcBig(c.f32(fr)))
	st, _ := sum.Float32()
	c.m.w32(fr, math.Float32bits(st))
	heal(chdFtol(sum))
	whole := chdFtol(strcBig(c.f32(fr))) // fild of the 32-bit result with a zero high word
	left, _ := new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Sub(strcBig(c.f32(fr)), chdF(float64(whole))).Float32()
	c.m.w32(fr, math.Float32bits(left))
}

func (c *ctrsC) node() {
	m, ch, who, t := c.m, c.ch, c.who, c.ticks
	flagB := false
	u := func(off uint32) uint32 { return m.r32(ch + off) }
	i32 := func(off uint32) int32 { return int32(m.r32(ch + off)) }

	// per-tick damage +0x1778; half again unless mode 3 outside combat
	if b := m.r8(ch + 0x1778); b != 0 {
		d := b * t
		if !(m.r32(0x687509) == 3 && m.r8(0x683f94) == 0) {
			d += d >> 1
		}
		c.call(0x52a890, who, d, 1, 1, 0, 0, 0)
	}

	// condition 2 (+0xa09): with chance, one of four losses
	if u(0xa09) != 0 {
		flagB = true
		if c.rnd(u(0x125)*10) < t {
			switch c.rnd(4) {
			case 0:
				v := c.rnd(5)
				if v > 4 {
					panic("turn recovery: R(5) above 4 (blocks 5031ec/5031f3)")
				}
				c.call(0x522fe0, who, []uint32{3, 4, 5, 0xc, 0xb}[v], ctrsNever, 0, 0, 1) // table 0x503974
			case 1:
				i := c.rnd(7)
				if e := ch + 0xe5 + 20*i; m.r32(e) > 1 {
					m.w32(e, m.r32(e)-1)
					m.w32(e+8, m.r32(e+8)+1)
					c.attrChanged(i)
					c.call(0x58aad0, 8, 0x64bf5c, ch+5) // via 0x590950
				}
			case 2:
				if u(0xb0d) >= 5 {
					d := c.rnd(4) + 1
					m.w32(ch+0xb0d, u(0xb0d)-d)
					m.w32(ch+0xb15, u(0xb15)-d)
					c.call(0x58aad0, 8, 0x64bf5c, ch+5)
					if h := u(0xb11); h > 1 {
						if d >= h-1 {
							d = h - 1
						}
						m.w32(ch+0xb11, h-d)
					}
					c.refresh()
				}
			case 3:
				if u(0xb19) >= 5 {
					d := c.rnd(4) + 1
					m.w32(ch+0xb19, u(0xb19)-d)
					m.w32(ch+0xb21, u(0xb21)+d)
					c.call(0x58aad0, 8, 0x64bf5c, ch+5)
					if s := i32(0xb1d); s > 1 {
						if d >= uint32(s-1) {
							d = uint32(s - 1)
						}
						m.w32(ch+0xb1d, uint32(s)-d)
					}
					c.refresh()
				}
			}
		}
	}

	// condition 10 (+0xa29)
	if u(0xa29) != 0 {
		if c.selKey() == 9 || c.selKey() == 0xa {
			if u(0xa25) == ctrsNever {
				c.call(0x523330, who, 9, 1)
			}
		} else {
			n := uint32(0)
			for k := t; k > 0; k-- {
				if c.rnd(4) == 0 {
					n++
				}
			}
			if n > 0 {
				c.call(0x52a890, who, n, 0, 1, 0, 0, 0)
			}
			if u(0xa25) == 0 {
				c.call(0x522fe0, who, 9, ctrsNever, 0, 0, 1)
			}
		}
	}

	// item 0x243 held (not on the cursor): condition 11 unless its holder carries 0x239
	if loc, owner, ok := c.findItem(0x243); ok && loc != ctrsCursor {
		if owner != 0 && c.findOn(owner, 0x239, false) != 0 {
			if u(0xa2d) == ctrsNever {
				c.call(0x523330, who, 0xb, 1)
			}
		} else if u(0xa2d) < ctrsNever {
			c.call(0x522fe0, who, 0xb, ctrsNever, 0, 0, 1)
		}
	}

	// per-tick rates: HP +0x1779, stamina +0x177a, SP +0x177b (i8)
	if r := int32(int8(m.r8(ch + 0x1779))); r > 0 {
		if u(0xb11) < u(0xb0d) {
			c.call(0x52add0, who, uint32(r)*t, 0)
		}
	} else if r < 0 {
		c.call(0x52a890, who, -(uint32(r) * t), 0, 1, 0, 0, 0)
	}
	if r := int32(int8(m.r8(ch + 0x177a))); r > 0 {
		if i32(0xb1d) < i32(0xb19) {
			c.call(0x52b1c0, who, uint32(r)*t, 0)
		}
	} else if r < 0 {
		c.call(0x52af50, who, -(uint32(r) * t), 0, 0)
	}
	for i := uint32(0); i < 6; i++ {
		if r := int32(int8(m.r8(ch + 0x177b))); r > 0 {
			if i32(0xb45+4*i) < i32(0xb25+4*i) {
				c.call(0x52b4f0, who, i, uint32(r)*t)
			}
		} else if r < 0 {
			c.call(0x52b6d0, who, i, uint32(-r), 1)
		}
	}

	// regeneration factors (f32 constants 0x5ebb38 1.0, 0x5ebc7c 0.5, 0x5ebb34 0.0,
	// 0x5ebcc8 0.25, 0x5ec340 1.25, 0x5ebccc 0.75)
	f := c.f32(0x5ebb34)
	if m.r8(0x683fc5) != 0 {
		f = c.f32(0x5ebb38)
	} else if m.r32(0x687509) == 3 && m.r8(0x683f94) == 0 {
		f = c.f32(0x5ebc7c)
	}
	if flagB {
		f *= c.f32(0x5ebcc8)
	}
	hpF, spF, stF := f, f, float32(0)
	if spF < c.f32(0x5ebc7c) {
		spF = c.f32(0x5ebc7c)
	}
	if c.feat(1) {
		g := f
		if f == 0 {
			g = c.f32(0x5ebc7c)
		}
		hpF, _ = chdMul(c.levelScale(math.Float32frombits(0x41855c29)), strcBig(g)).Float32() // immediate at 0x503619
	}
	if c.feat(0) && m.r8(0x683f94) != 0 {
		stF, _ = c.levelScale(math.Float32frombits(0x40533333)).Float32() // immediate at 0x503667
	}
	if c.feat(0x1a) && spF > 0 {
		spF, _ = chdMul(strcBig(spF), strcBig(c.f32(0x5ec340))).Float32()
	}
	if c.feat(0x1b) && spF > 0 {
		spF, _ = chdMul(strcBig(spF), strcBig(c.f32(0x5ebccc))).Float32()
	}
	if hpF > 0 && u(0xb11) < u(0xb0d) {
		c.frac(ch+0xb69, ch+0xb6d, hpF, func(n uint32) { c.call(0x52add0, who, n, 0) })
	}
	if stF > 0 && i32(0xb1d) < i32(0xb19) {
		c.frac(ch+0xb71, ch+0xb75, stF, func(n uint32) { c.call(0x52b1c0, who, n, 0) })
	}
	if spF > 0 {
		for i := uint32(0); i < 6; i++ {
			if i32(0xb45+4*i) < i32(0xb25+4*i) {
				c.frac(ch+0xb79+8*i, ch+0xb7d+8*i, spF, func(n uint32) { c.call(0x52b4f0, who, i, n) })
			}
		}
	}
	if u(0xb11) >= u(0xb0d) {
		m.w32(ch+0xb6d, 0)
	}
	if i32(0xb1d) >= i32(0xb19) {
		m.w32(ch+0xb75, 0)
	}
	for i := uint32(0); i < 6; i++ {
		if i32(0xb45+4*i) >= i32(0xb25+4*i) {
			m.w32(ch+0xb7d+8*i, 0)
		}
	}

	// condition countdowns (+0xa01, 20 words) and timed effects (+0xa6d, 8 x 12 bytes)
	for k := uint32(0); k < 20; k++ {
		if s := u(0xa01 + 4*k); s > 0 && s < ctrsNever {
			c.call(0x5236a0, who, k, t)
		}
	}
	for j := uint32(0); j < 8; j++ {
		if e := u(0xa6d + 12*j); e > 0 && e < ctrsNever {
			if e > t {
				m.w32(ch+0xa6d+12*j, e-t)
			} else {
				c.call(0x523a80, who, j)
			}
		}
	}

	// feature 0x12: resting with +0xb65 at 0 calls 0x548e60; otherwise +0xb65 counts down
	if c.feat(0x12) {
		b := u(0xb65)
		if m.r8(0x683fc5) != 0 {
			if b == 0 {
				c.call(0x548e60, ch)
			}
		} else if b > 0 {
			if t >= b {
				m.w32(ch+0xb65, 0)
			} else {
				m.w32(ch+0xb65, b-t)
			}
		}
	}
}

// ctrsLink ties one callee input or output (position pos in the callee's case vector)
// to the node's memory: 'a' call argument x, 'c' constant x (a pointer to an object of
// the callee case, whose other fields are then 0), 'w' the record index, 'g' global x, 'o' *x + off (+ stride * input idx).
type ctrsLink struct {
	pos    int
	k      byte
	x, off uint32
	idx    int
	stride uint32
	size   int
	signed int
}

// ctrsCallee is a D4 callee's Go entry; chars is the record base its case expects in a
// record-pointer argument (0: the argument is passed unchanged).
type ctrsCallee struct {
	fn    func([]int64, Rand) []int64
	nIn   int
	chars uint32
	in    []ctrsLink
	out   []ctrsLink
}

func (c *ctrsC) linkAddr(l ctrsLink, v []int64) uint32 {
	a := c.m.r32(l.x) + l.off
	if l.idx >= 0 {
		a += uint32(v[l.idx]) * l.stride
	}
	return a
}

// realCall runs a D4 callee's Go on a world read from the node's memory (inputs the node
// has no memory for, such as the callee's own stub returns and F-only inputs, are 0, as
// in n_524720) and writes its
// character, presence, screen and global outputs back. 排除 callees only record.
func (c *ctrsC) realCall(addr uint32, args []uint32) {
	cl, ok := ctrsCallees[addr]
	if !ok {
		return
	}
	v := make([]int64, cl.nIn)
	set := func(l ctrsLink, x uint32) {
		v[l.pos] = int64(x)
		if l.signed != 0 {
			sh := 32 - 8*uint(l.size)
			v[l.pos] = int64(int32(x<<sh) >> sh)
		}
	}
	for _, l := range cl.in {
		switch l.k {
		case 'a':
			x := args[l.x]
			if cl.chars != 0 && x == c.ch {
				x = cl.chars + c.who*ctrsStride
			}
			set(l, x)
		case 'c':
			set(l, l.x)
		case 'w':
			set(l, c.who)
		case 'g':
			set(l, c.m.get(l.x, l.size))
		}
	}
	for _, l := range cl.in {
		if l.k == 'o' {
			set(l, c.m.get(c.linkAddr(l, v), l.size))
		}
	}
	out := cl.fn(v, c.r)
	for _, l := range cl.out {
		a := l.x
		if l.k == 'o' {
			a = c.linkAddr(l, v)
		}
		c.m.put(a, l.size, uint32(out[l.pos]))
	}
}

var ctrsCallees = map[uint32]ctrsCallee{
	0x52a890: {characterDamageApplyNode, 431, 0x20000000, []ctrsLink{
		{0, 'a', 0, 0, -1, 0, 4, 0}, {1, 'a', 1, 0, -1, 0, 4, 0}, {2, 'a', 2, 0, -1, 0, 4, 0}, {3, 'a', 3, 0, -1, 0, 4, 0},
		{4, 'a', 4, 0, -1, 0, 4, 0}, {5, 'a', 5, 0, -1, 0, 4, 0}, {6, 'a', 6, 0, -1, 0, 4, 0}, {7, 'o', 0x685178, 0x0, 0, 0x106, 1, 0},
		{8, 'o', 0x685174, 0xb11, 0, 0x1862, 4, 0}, {9, 'o', 0x685174, 0xb0d, 0, 0x1862, 4, 0}, {10, 'o', 0x685174, 0xb01, 0, 0x1862, 4, 0}, {11, 'o', 0x685174, 0xa85, 0, 0x1862, 4, 0},
		{12, 'o', 0x685174, 0xa83, 0, 0x1862, 2, 0}, {13, 'o', 0x685174, 0xa3d, 0, 0x1862, 4, 0}, {14, 'o', 0x685174, 0xa45, 0, 0x1862, 4, 0}, {15, 'o', 0x685174, 0x161, 0, 0x1862, 4, 0},
		{16, 'o', 0x685174, 0x69, 0, 0x1862, 4, 1}, {17, 'o', 0x685174, 0x71, 0, 0x1862, 4, 1}, {18, 'g', 0x687500, 0, -1, 0, 1, 0}, {19, 'g', 0x683fc5, 0, -1, 0, 1, 0},
		{20, 'g', 0x68ec78, 0, -1, 0, 4, 0}, {21, 'g', 0x68edcc, 0, -1, 0, 4, 0}, {23, 'g', 0x6850ce, 0, -1, 0, 4, 0}, {25, 'g', 0x6eb70c, 0, -1, 0, 4, 0},
		{26, 'g', 0x68c544, 0, -1, 0, 4, 0}, {27, 'g', 0x68c550, 0, -1, 0, 4, 0}, {28, 'g', 0x68c51c, 0, -1, 0, 4, 0}, {32, 'c', 0x20028300, 0, -1, 0, 4, 0},
		{33, 'c', 0x20028200, 0, -1, 0, 4, 0}, {39, 'c', 0x20028100, 0, -1, 0, 4, 0}, {73, 'c', 0x20028700, 0, -1, 0, 4, 0}, {74, 'g', 0x683754, 0, -1, 0, 1, 0},
		{75, 'g', 0x683755, 0, -1, 0, 1, 0}, {76, 'g', 0x683757, 0, -1, 0, 4, 0}, {77, 'g', 0x683775, 0, -1, 0, 1, 0}, {78, 'g', 0x683788, 0, -1, 0, 1, 0},
		{79, 'g', 0x68386c, 0, -1, 0, 1, 0}, {80, 'g', 0x68386d, 0, -1, 0, 1, 0}, {81, 'g', 0x68386f, 0, -1, 0, 4, 0}, {82, 'g', 0x68388d, 0, -1, 0, 1, 0},
		{83, 'g', 0x6838a0, 0, -1, 0, 1, 0}, {84, 'g', 0x683984, 0, -1, 0, 1, 0}, {85, 'g', 0x683985, 0, -1, 0, 1, 0}, {86, 'g', 0x683987, 0, -1, 0, 4, 0},
		{87, 'g', 0x6839a5, 0, -1, 0, 1, 0}, {88, 'g', 0x6839b8, 0, -1, 0, 1, 0}, {89, 'g', 0x683a9c, 0, -1, 0, 1, 0}, {90, 'g', 0x683a9d, 0, -1, 0, 1, 0},
		{91, 'g', 0x683a9f, 0, -1, 0, 4, 0}, {92, 'g', 0x683abd, 0, -1, 0, 1, 0}, {93, 'g', 0x683ad0, 0, -1, 0, 1, 0}, {94, 'g', 0x683bb4, 0, -1, 0, 1, 0},
		{95, 'g', 0x683bb5, 0, -1, 0, 1, 0}, {96, 'g', 0x683bb7, 0, -1, 0, 4, 0}, {97, 'g', 0x683bd5, 0, -1, 0, 1, 0}, {98, 'g', 0x683be8, 0, -1, 0, 1, 0},
		{99, 'g', 0x683ccc, 0, -1, 0, 1, 0}, {100, 'g', 0x683ccd, 0, -1, 0, 1, 0}, {101, 'g', 0x683ccf, 0, -1, 0, 4, 0}, {102, 'g', 0x683ced, 0, -1, 0, 1, 0},
		{103, 'g', 0x683d00, 0, -1, 0, 1, 0}, {104, 'g', 0x683de4, 0, -1, 0, 1, 0}, {105, 'g', 0x683de5, 0, -1, 0, 1, 0}, {106, 'g', 0x683de7, 0, -1, 0, 4, 0},
		{107, 'g', 0x683e05, 0, -1, 0, 1, 0}, {108, 'g', 0x683e18, 0, -1, 0, 1, 0}, {109, 'g', 0x683efc, 0, -1, 0, 1, 0}, {110, 'g', 0x683efd, 0, -1, 0, 1, 0},
		{111, 'g', 0x683eff, 0, -1, 0, 4, 0}, {112, 'g', 0x683f1d, 0, -1, 0, 1, 0}, {113, 'g', 0x683f30, 0, -1, 0, 1, 0}, {126, 'c', 0x20000000, 0, -1, 0, 4, 1},
		{127, 'c', 0x20011000, 0, -1, 0, 4, 1}, {204, 'c', 0x20015000, 0, -1, 0, 4, 1},
	}, []ctrsLink{
		{1, 'o', 0x685174, 0xb11, 0, 0x1862, 4, 0}, {2, 'o', 0x685174, 0xa83, 0, 0x1862, 2, 0}, {73, 'g', 0x683754, 0, -1, 0, 1, 0}, {74, 'g', 0x683755, 0, -1, 0, 1, 0},
		{75, 'g', 0x683757, 0, -1, 0, 4, 0}, {76, 'g', 0x68375b, 0, -1, 0, 4, 0}, {77, 'g', 0x68375f, 0, -1, 0, 4, 0}, {78, 'g', 0x683763, 0, -1, 0, 1, 0},
		{79, 'g', 0x683775, 0, -1, 0, 1, 0}, {80, 'g', 0x683782, 0, -1, 0, 4, 0}, {81, 'g', 0x683787, 0, -1, 0, 1, 0}, {82, 'g', 0x683788, 0, -1, 0, 1, 0},
		{83, 'g', 0x68386c, 0, -1, 0, 1, 0}, {84, 'g', 0x68386d, 0, -1, 0, 1, 0}, {85, 'g', 0x68386f, 0, -1, 0, 4, 0}, {86, 'g', 0x683873, 0, -1, 0, 4, 0},
		{87, 'g', 0x683877, 0, -1, 0, 4, 0}, {88, 'g', 0x68387b, 0, -1, 0, 1, 0}, {89, 'g', 0x68388d, 0, -1, 0, 1, 0}, {90, 'g', 0x68389a, 0, -1, 0, 4, 0},
		{91, 'g', 0x68389f, 0, -1, 0, 1, 0}, {92, 'g', 0x6838a0, 0, -1, 0, 1, 0}, {93, 'g', 0x683984, 0, -1, 0, 1, 0}, {94, 'g', 0x683985, 0, -1, 0, 1, 0},
		{95, 'g', 0x683987, 0, -1, 0, 4, 0}, {96, 'g', 0x68398b, 0, -1, 0, 4, 0}, {97, 'g', 0x68398f, 0, -1, 0, 4, 0}, {98, 'g', 0x683993, 0, -1, 0, 1, 0},
		{99, 'g', 0x6839a5, 0, -1, 0, 1, 0}, {100, 'g', 0x6839b2, 0, -1, 0, 4, 0}, {101, 'g', 0x6839b7, 0, -1, 0, 1, 0}, {102, 'g', 0x6839b8, 0, -1, 0, 1, 0},
		{103, 'g', 0x683a9c, 0, -1, 0, 1, 0}, {104, 'g', 0x683a9d, 0, -1, 0, 1, 0}, {105, 'g', 0x683a9f, 0, -1, 0, 4, 0}, {106, 'g', 0x683aa3, 0, -1, 0, 4, 0},
		{107, 'g', 0x683aa7, 0, -1, 0, 4, 0}, {108, 'g', 0x683aab, 0, -1, 0, 1, 0}, {109, 'g', 0x683abd, 0, -1, 0, 1, 0}, {110, 'g', 0x683aca, 0, -1, 0, 4, 0},
		{111, 'g', 0x683acf, 0, -1, 0, 1, 0}, {112, 'g', 0x683ad0, 0, -1, 0, 1, 0}, {113, 'g', 0x683bb4, 0, -1, 0, 1, 0}, {114, 'g', 0x683bb5, 0, -1, 0, 1, 0},
		{115, 'g', 0x683bb7, 0, -1, 0, 4, 0}, {116, 'g', 0x683bbb, 0, -1, 0, 4, 0}, {117, 'g', 0x683bbf, 0, -1, 0, 4, 0}, {118, 'g', 0x683bc3, 0, -1, 0, 1, 0},
		{119, 'g', 0x683bd5, 0, -1, 0, 1, 0}, {120, 'g', 0x683be2, 0, -1, 0, 4, 0}, {121, 'g', 0x683be7, 0, -1, 0, 1, 0}, {122, 'g', 0x683be8, 0, -1, 0, 1, 0},
		{123, 'g', 0x683ccc, 0, -1, 0, 1, 0}, {124, 'g', 0x683ccd, 0, -1, 0, 1, 0}, {125, 'g', 0x683ccf, 0, -1, 0, 4, 0}, {126, 'g', 0x683cd3, 0, -1, 0, 4, 0},
		{127, 'g', 0x683cd7, 0, -1, 0, 4, 0}, {128, 'g', 0x683cdb, 0, -1, 0, 1, 0}, {129, 'g', 0x683ced, 0, -1, 0, 1, 0}, {130, 'g', 0x683cfa, 0, -1, 0, 4, 0},
		{131, 'g', 0x683cff, 0, -1, 0, 1, 0}, {132, 'g', 0x683d00, 0, -1, 0, 1, 0}, {133, 'g', 0x683de4, 0, -1, 0, 1, 0}, {134, 'g', 0x683de5, 0, -1, 0, 1, 0},
		{135, 'g', 0x683de7, 0, -1, 0, 4, 0}, {136, 'g', 0x683deb, 0, -1, 0, 4, 0}, {137, 'g', 0x683def, 0, -1, 0, 4, 0}, {138, 'g', 0x683df3, 0, -1, 0, 1, 0},
		{139, 'g', 0x683e05, 0, -1, 0, 1, 0}, {140, 'g', 0x683e12, 0, -1, 0, 4, 0}, {141, 'g', 0x683e17, 0, -1, 0, 1, 0}, {142, 'g', 0x683e18, 0, -1, 0, 1, 0},
		{143, 'g', 0x683efc, 0, -1, 0, 1, 0}, {144, 'g', 0x683efd, 0, -1, 0, 1, 0}, {145, 'g', 0x683eff, 0, -1, 0, 4, 0}, {146, 'g', 0x683f03, 0, -1, 0, 4, 0},
		{147, 'g', 0x683f07, 0, -1, 0, 4, 0}, {148, 'g', 0x683f0b, 0, -1, 0, 1, 0}, {149, 'g', 0x683f1d, 0, -1, 0, 1, 0}, {150, 'g', 0x683f2a, 0, -1, 0, 4, 0},
		{151, 'g', 0x683f2f, 0, -1, 0, 1, 0}, {152, 'g', 0x683f30, 0, -1, 0, 1, 0},
	}},
	0x522fe0: {characterConditionApplyNode, 311, 0x0, []ctrsLink{
		{0, 'a', 0, 0, -1, 0, 4, 0}, {1, 'a', 1, 0, -1, 0, 4, 0}, {2, 'a', 2, 0, -1, 0, 4, 0}, {3, 'a', 3, 0, -1, 0, 4, 0},
		{4, 'a', 4, 0, -1, 0, 4, 0}, {5, 'a', 5, 0, -1, 0, 4, 0}, {6, 'g', 0x685174, 0, -1, 0, 4, 0}, {7, 'c', 0x20011000, 0, -1, 0, 4, 0},
		{8, 'g', 0x683f94, 0, -1, 0, 1, 0}, {9, 'g', 0x687500, 0, -1, 0, 1, 0}, {10, 'g', 0x6840bb, 0, -1, 0, 1, 0}, {11, 'g', 0x68c4fa, 0, -1, 0, 1, 0},
		{12, 'g', 0x6875f8, 0, -1, 0, 1, 0}, {13, 'g', 0x683fad, 0, -1, 0, 4, 0}, {23, 'o', 0x685178, 0x0, -1, 0x0, 1, 0}, {24, 'o', 0x685178, 0x106, -1, 0x0, 1, 0},
		{25, 'o', 0x685178, 0x20c, -1, 0x0, 1, 0}, {26, 'o', 0x685178, 0x312, -1, 0x0, 1, 0}, {27, 'o', 0x685178, 0x418, -1, 0x0, 1, 0}, {28, 'o', 0x685178, 0x51e, -1, 0x0, 1, 0},
		{29, 'o', 0x685178, 0x624, -1, 0x0, 1, 0}, {30, 'o', 0x685178, 0x72a, -1, 0x0, 1, 0}, {31, 'o', 0x685178, 0x0, 0, 0x106, 1, 0}, {32, 'o', 0x685174, 0x69, 0, 0x1862, 4, 1},
		{33, 'o', 0x685174, 0x71, 0, 0x1862, 4, 1}, {34, 'o', 0x685174, 0x89, 0, 0x1862, 4, 0}, {35, 'o', 0x685174, 0x8d, 0, 0x1862, 4, 0}, {36, 'o', 0x685174, 0x91, 0, 0x1862, 4, 0},
		{37, 'o', 0x685174, 0x95, 0, 0x1862, 4, 0}, {38, 'o', 0x685174, 0x99, 0, 0x1862, 4, 0}, {39, 'o', 0x685174, 0x9d, 0, 0x1862, 4, 0}, {40, 'o', 0x685174, 0xa1, 0, 0x1862, 4, 0},
		{41, 'o', 0x685174, 0xa5, 0, 0x1862, 4, 0}, {42, 'o', 0x685174, 0xa9, 0, 0x1862, 4, 0}, {43, 'o', 0x685174, 0xad, 0, 0x1862, 4, 0}, {44, 'o', 0x685174, 0xb1, 0, 0x1862, 4, 0},
		{45, 'o', 0x685174, 0xb5, 0, 0x1862, 4, 0}, {46, 'o', 0x685174, 0xb9, 0, 0x1862, 4, 0}, {47, 'o', 0x685174, 0xbd, 0, 0x1862, 4, 0}, {48, 'o', 0x685174, 0xc1, 0, 0x1862, 4, 0},
		{49, 'o', 0x685174, 0xc5, 0, 0x1862, 4, 0}, {50, 'o', 0x685174, 0xa79, 0, 0x1862, 4, 0}, {51, 'o', 0x685174, 0xa01, 0, 0x1862, 4, 0}, {52, 'o', 0x685174, 0xa05, 0, 0x1862, 4, 0},
		{53, 'o', 0x685174, 0xa09, 0, 0x1862, 4, 0}, {54, 'o', 0x685174, 0xa0d, 0, 0x1862, 4, 0}, {55, 'o', 0x685174, 0xa11, 0, 0x1862, 4, 0}, {56, 'o', 0x685174, 0xa15, 0, 0x1862, 4, 0},
		{57, 'o', 0x685174, 0xa19, 0, 0x1862, 4, 0}, {58, 'o', 0x685174, 0xa1d, 0, 0x1862, 4, 0}, {59, 'o', 0x685174, 0xa21, 0, 0x1862, 4, 0}, {60, 'o', 0x685174, 0xa25, 0, 0x1862, 4, 0},
		{61, 'o', 0x685174, 0xa29, 0, 0x1862, 4, 0}, {62, 'o', 0x685174, 0xa2d, 0, 0x1862, 4, 0}, {63, 'o', 0x685174, 0xa31, 0, 0x1862, 4, 0}, {64, 'o', 0x685174, 0xa35, 0, 0x1862, 4, 0},
		{65, 'o', 0x685174, 0xa39, 0, 0x1862, 4, 0}, {66, 'o', 0x685174, 0xa3d, 0, 0x1862, 4, 0}, {67, 'o', 0x685174, 0xa41, 0, 0x1862, 4, 0}, {68, 'o', 0x685174, 0xa45, 0, 0x1862, 4, 0},
		{69, 'o', 0x685174, 0xa49, 0, 0x1862, 4, 0}, {70, 'o', 0x685174, 0xa4d, 0, 0x1862, 4, 0}, {71, 'o', 0x685174, 0xb01, 0, 0x1862, 4, 0}, {72, 'o', 0x685174, 0xb09, 0, 0x1862, 4, 0},
		{73, 'o', 0x685174, 0xb0d, 0, 0x1862, 4, 0}, {74, 'o', 0x685174, 0xb11, 0, 0x1862, 4, 0}, {83, 'g', 0x68ec78, 0, -1, 0, 4, 0}, {84, 'g', 0x68edcc, 0, -1, 0, 4, 0},
		{86, 'o', 0x69c0f4, 0xf8, -1, 0x0, 4, 0}, {87, 'g', 0x683fc5, 0, -1, 0, 1, 0}, {88, 'g', 0x683745, 0, -1, 0, 4, 0}, {89, 'g', 0x68385d, 0, -1, 0, 4, 0},
		{90, 'g', 0x683975, 0, -1, 0, 4, 0}, {91, 'g', 0x683a8d, 0, -1, 0, 4, 0}, {92, 'g', 0x683ba5, 0, -1, 0, 4, 0}, {93, 'g', 0x683cbd, 0, -1, 0, 4, 0},
		{94, 'g', 0x683dd5, 0, -1, 0, 4, 0}, {95, 'g', 0x683eed, 0, -1, 0, 4, 0}, {96, 'g', 0x683751, 0, -1, 0, 1, 0}, {97, 'g', 0x683869, 0, -1, 0, 1, 0},
		{98, 'g', 0x683981, 0, -1, 0, 1, 0}, {99, 'g', 0x683a99, 0, -1, 0, 1, 0}, {100, 'g', 0x683bb1, 0, -1, 0, 1, 0}, {101, 'g', 0x683cc9, 0, -1, 0, 1, 0},
		{102, 'g', 0x683de1, 0, -1, 0, 1, 0}, {103, 'g', 0x683ef9, 0, -1, 0, 1, 0}, {104, 'g', 0x687511, 0, -1, 0, 1, 0}, {105, 'g', 0x687512, 0, -1, 0, 1, 0},
		{106, 'g', 0x687513, 0, -1, 0, 1, 0}, {107, 'g', 0x687514, 0, -1, 0, 1, 0}, {108, 'g', 0x687515, 0, -1, 0, 1, 0}, {109, 'g', 0x687516, 0, -1, 0, 1, 0},
		{110, 'g', 0x687517, 0, -1, 0, 1, 0}, {111, 'g', 0x687518, 0, -1, 0, 1, 0}, {112, 'g', 0x687519, 0, -1, 0, 1, 0}, {113, 'g', 0x68751a, 0, -1, 0, 1, 0},
		{114, 'g', 0x68751b, 0, -1, 0, 1, 0}, {115, 'g', 0x68751c, 0, -1, 0, 1, 0}, {116, 'g', 0x68751d, 0, -1, 0, 1, 0}, {117, 'g', 0x68751e, 0, -1, 0, 1, 0},
		{118, 'g', 0x68751f, 0, -1, 0, 1, 0}, {119, 'g', 0x687520, 0, -1, 0, 1, 0}, {120, 'g', 0x687521, 0, -1, 0, 1, 0}, {121, 'g', 0x687522, 0, -1, 0, 1, 0},
		{122, 'g', 0x687523, 0, -1, 0, 1, 0}, {123, 'g', 0x687524, 0, -1, 0, 1, 0}, {124, 'g', 0x687525, 0, -1, 0, 1, 0}, {125, 'g', 0x687526, 0, -1, 0, 1, 0},
		{126, 'g', 0x687527, 0, -1, 0, 1, 0}, {127, 'g', 0x687528, 0, -1, 0, 1, 0}, {128, 'g', 0x687531, 0, -1, 0, 1, 0}, {129, 'g', 0x687532, 0, -1, 0, 1, 0},
		{130, 'g', 0x687533, 0, -1, 0, 1, 0}, {131, 'g', 0x687534, 0, -1, 0, 1, 0}, {132, 'g', 0x68753d, 0, -1, 0, 1, 0}, {133, 'g', 0x68753e, 0, -1, 0, 1, 0},
		{134, 'g', 0x68753f, 0, -1, 0, 1, 0}, {135, 'g', 0x687540, 0, -1, 0, 1, 0}, {136, 'g', 0x687549, 0, -1, 0, 1, 0}, {137, 'g', 0x68754a, 0, -1, 0, 1, 0},
		{138, 'g', 0x68754b, 0, -1, 0, 1, 0}, {139, 'g', 0x68754c, 0, -1, 0, 1, 0}, {140, 'g', 0x687555, 0, -1, 0, 1, 0}, {141, 'g', 0x687556, 0, -1, 0, 1, 0},
		{142, 'g', 0x687557, 0, -1, 0, 1, 0}, {143, 'g', 0x687558, 0, -1, 0, 1, 0}, {144, 'g', 0x687561, 0, -1, 0, 1, 0}, {145, 'g', 0x687562, 0, -1, 0, 1, 0},
		{146, 'g', 0x687563, 0, -1, 0, 1, 0}, {147, 'g', 0x687564, 0, -1, 0, 1, 0}, {148, 'g', 0x68756d, 0, -1, 0, 1, 0}, {149, 'g', 0x68756e, 0, -1, 0, 1, 0},
		{150, 'g', 0x68756f, 0, -1, 0, 1, 0}, {151, 'g', 0x687570, 0, -1, 0, 1, 0}, {152, 'g', 0x687579, 0, -1, 0, 1, 0}, {153, 'g', 0x68757a, 0, -1, 0, 1, 0},
		{154, 'g', 0x68757b, 0, -1, 0, 1, 0}, {155, 'g', 0x68757c, 0, -1, 0, 1, 0}, {156, 'g', 0x683feb, 0, -1, 0, 1, 0}, {157, 'g', 0x683fec, 0, -1, 0, 1, 0},
		{158, 'g', 0x683fed, 0, -1, 0, 1, 0}, {159, 'g', 0x683fee, 0, -1, 0, 1, 0}, {160, 'g', 0x683fef, 0, -1, 0, 1, 0}, {161, 'g', 0x683ff0, 0, -1, 0, 1, 0},
		{162, 'g', 0x683ff1, 0, -1, 0, 1, 0}, {163, 'g', 0x683ff2, 0, -1, 0, 1, 0}, {164, 'g', 0x683ff3, 0, -1, 0, 1, 0}, {165, 'g', 0x683ff4, 0, -1, 0, 1, 0},
		{166, 'g', 0x683ff5, 0, -1, 0, 1, 0}, {167, 'g', 0x683ff6, 0, -1, 0, 1, 0}, {168, 'g', 0x683ff7, 0, -1, 0, 1, 0}, {169, 'g', 0x683ff8, 0, -1, 0, 1, 0},
		{170, 'g', 0x683ff9, 0, -1, 0, 1, 0}, {171, 'g', 0x683ffa, 0, -1, 0, 1, 0}, {172, 'g', 0x683ffb, 0, -1, 0, 1, 0}, {173, 'g', 0x683ffc, 0, -1, 0, 1, 0},
		{174, 'g', 0x683ffd, 0, -1, 0, 1, 0}, {175, 'g', 0x683ffe, 0, -1, 0, 1, 0}, {176, 'g', 0x683fff, 0, -1, 0, 1, 0}, {177, 'g', 0x684000, 0, -1, 0, 1, 0},
		{178, 'g', 0x684001, 0, -1, 0, 1, 0}, {179, 'g', 0x684002, 0, -1, 0, 1, 0}, {180, 'g', 0x68400b, 0, -1, 0, 1, 0}, {181, 'g', 0x68400c, 0, -1, 0, 1, 0},
		{182, 'g', 0x68400d, 0, -1, 0, 1, 0}, {183, 'g', 0x68400e, 0, -1, 0, 1, 0}, {184, 'g', 0x684017, 0, -1, 0, 1, 0}, {185, 'g', 0x684018, 0, -1, 0, 1, 0},
		{186, 'g', 0x684019, 0, -1, 0, 1, 0}, {187, 'g', 0x68401a, 0, -1, 0, 1, 0}, {188, 'g', 0x684023, 0, -1, 0, 1, 0}, {189, 'g', 0x684024, 0, -1, 0, 1, 0},
		{190, 'g', 0x684025, 0, -1, 0, 1, 0}, {191, 'g', 0x684026, 0, -1, 0, 1, 0}, {192, 'g', 0x68402f, 0, -1, 0, 1, 0}, {193, 'g', 0x684030, 0, -1, 0, 1, 0},
		{194, 'g', 0x684031, 0, -1, 0, 1, 0}, {195, 'g', 0x684032, 0, -1, 0, 1, 0}, {196, 'g', 0x68403b, 0, -1, 0, 1, 0}, {197, 'g', 0x68403c, 0, -1, 0, 1, 0},
		{198, 'g', 0x68403d, 0, -1, 0, 1, 0}, {199, 'g', 0x68403e, 0, -1, 0, 1, 0}, {200, 'g', 0x684047, 0, -1, 0, 1, 0}, {201, 'g', 0x684048, 0, -1, 0, 1, 0},
		{202, 'g', 0x684049, 0, -1, 0, 1, 0}, {203, 'g', 0x68404a, 0, -1, 0, 1, 0}, {204, 'g', 0x684053, 0, -1, 0, 1, 0}, {205, 'g', 0x684054, 0, -1, 0, 1, 0},
		{206, 'g', 0x684055, 0, -1, 0, 1, 0}, {207, 'g', 0x684056, 0, -1, 0, 1, 0},
	}, []ctrsLink{
		{1, 'o', 0x685174, 0xa01, 0, 0x1862, 4, 0}, {2, 'o', 0x685174, 0xa05, 0, 0x1862, 4, 0}, {3, 'o', 0x685174, 0xa09, 0, 0x1862, 4, 0}, {4, 'o', 0x685174, 0xa0d, 0, 0x1862, 4, 0},
		{5, 'o', 0x685174, 0xa11, 0, 0x1862, 4, 0}, {6, 'o', 0x685174, 0xa15, 0, 0x1862, 4, 0}, {7, 'o', 0x685174, 0xa19, 0, 0x1862, 4, 0}, {8, 'o', 0x685174, 0xa1d, 0, 0x1862, 4, 0},
		{9, 'o', 0x685174, 0xa21, 0, 0x1862, 4, 0}, {10, 'o', 0x685174, 0xa25, 0, 0x1862, 4, 0}, {11, 'o', 0x685174, 0xa29, 0, 0x1862, 4, 0}, {12, 'o', 0x685174, 0xa2d, 0, 0x1862, 4, 0},
		{13, 'o', 0x685174, 0xa31, 0, 0x1862, 4, 0}, {14, 'o', 0x685174, 0xa35, 0, 0x1862, 4, 0}, {15, 'o', 0x685174, 0xa39, 0, 0x1862, 4, 0}, {16, 'o', 0x685174, 0xa3d, 0, 0x1862, 4, 0},
		{17, 'o', 0x685174, 0xa41, 0, 0x1862, 4, 0}, {18, 'o', 0x685174, 0xa45, 0, 0x1862, 4, 0}, {19, 'o', 0x685174, 0xa49, 0, 0x1862, 4, 0}, {20, 'o', 0x685174, 0xa4d, 0, 0x1862, 4, 0},
		{21, 'o', 0x685174, 0xb01, 0, 0x1862, 4, 0}, {22, 'o', 0x685174, 0xb09, 0, 0x1862, 4, 0}, {23, 'o', 0x685174, 0xb11, 0, 0x1862, 4, 0}, {24, 'g', 0x6840bb, 0, -1, 0, 1, 0},
		{25, 'g', 0x6875f8, 0, -1, 0, 1, 0}, {34, 'g', 0x683745, 0, -1, 0, 4, 0}, {35, 'g', 0x68385d, 0, -1, 0, 4, 0}, {36, 'g', 0x683975, 0, -1, 0, 4, 0},
		{37, 'g', 0x683a8d, 0, -1, 0, 4, 0}, {38, 'g', 0x683ba5, 0, -1, 0, 4, 0}, {39, 'g', 0x683cbd, 0, -1, 0, 4, 0}, {40, 'g', 0x683dd5, 0, -1, 0, 4, 0},
		{41, 'g', 0x683eed, 0, -1, 0, 4, 0}, {42, 'g', 0x683751, 0, -1, 0, 1, 0}, {43, 'g', 0x683869, 0, -1, 0, 1, 0}, {44, 'g', 0x683981, 0, -1, 0, 1, 0},
		{45, 'g', 0x683a99, 0, -1, 0, 1, 0}, {46, 'g', 0x683bb1, 0, -1, 0, 1, 0}, {47, 'g', 0x683cc9, 0, -1, 0, 1, 0}, {48, 'g', 0x683de1, 0, -1, 0, 1, 0},
		{49, 'g', 0x683ef9, 0, -1, 0, 1, 0}, {51, 'o', 0x69c0f4, 0xf8, -1, 0x0, 4, 0}, {52, 'g', 0x687511, 0, -1, 0, 1, 0}, {53, 'g', 0x687512, 0, -1, 0, 1, 0},
		{54, 'g', 0x687513, 0, -1, 0, 1, 0}, {55, 'g', 0x687514, 0, -1, 0, 1, 0}, {56, 'g', 0x687515, 0, -1, 0, 1, 0}, {57, 'g', 0x687516, 0, -1, 0, 1, 0},
		{58, 'g', 0x687517, 0, -1, 0, 1, 0}, {59, 'g', 0x687518, 0, -1, 0, 1, 0}, {60, 'g', 0x687519, 0, -1, 0, 1, 0}, {61, 'g', 0x68751a, 0, -1, 0, 1, 0},
		{62, 'g', 0x68751b, 0, -1, 0, 1, 0}, {63, 'g', 0x68751c, 0, -1, 0, 1, 0}, {64, 'g', 0x68751d, 0, -1, 0, 1, 0}, {65, 'g', 0x68751e, 0, -1, 0, 1, 0},
		{66, 'g', 0x68751f, 0, -1, 0, 1, 0}, {67, 'g', 0x687520, 0, -1, 0, 1, 0}, {68, 'g', 0x687521, 0, -1, 0, 1, 0}, {69, 'g', 0x687522, 0, -1, 0, 1, 0},
		{70, 'g', 0x687523, 0, -1, 0, 1, 0}, {71, 'g', 0x687524, 0, -1, 0, 1, 0}, {72, 'g', 0x687525, 0, -1, 0, 1, 0}, {73, 'g', 0x687526, 0, -1, 0, 1, 0},
		{74, 'g', 0x687527, 0, -1, 0, 1, 0}, {75, 'g', 0x687528, 0, -1, 0, 1, 0}, {76, 'g', 0x687531, 0, -1, 0, 1, 0}, {77, 'g', 0x687532, 0, -1, 0, 1, 0},
		{78, 'g', 0x687533, 0, -1, 0, 1, 0}, {79, 'g', 0x687534, 0, -1, 0, 1, 0}, {80, 'g', 0x68753d, 0, -1, 0, 1, 0}, {81, 'g', 0x68753e, 0, -1, 0, 1, 0},
		{82, 'g', 0x68753f, 0, -1, 0, 1, 0}, {83, 'g', 0x687540, 0, -1, 0, 1, 0}, {84, 'g', 0x687549, 0, -1, 0, 1, 0}, {85, 'g', 0x68754a, 0, -1, 0, 1, 0},
		{86, 'g', 0x68754b, 0, -1, 0, 1, 0}, {87, 'g', 0x68754c, 0, -1, 0, 1, 0}, {88, 'g', 0x687555, 0, -1, 0, 1, 0}, {89, 'g', 0x687556, 0, -1, 0, 1, 0},
		{90, 'g', 0x687557, 0, -1, 0, 1, 0}, {91, 'g', 0x687558, 0, -1, 0, 1, 0}, {92, 'g', 0x687561, 0, -1, 0, 1, 0}, {93, 'g', 0x687562, 0, -1, 0, 1, 0},
		{94, 'g', 0x687563, 0, -1, 0, 1, 0}, {95, 'g', 0x687564, 0, -1, 0, 1, 0}, {96, 'g', 0x68756d, 0, -1, 0, 1, 0}, {97, 'g', 0x68756e, 0, -1, 0, 1, 0},
		{98, 'g', 0x68756f, 0, -1, 0, 1, 0}, {99, 'g', 0x687570, 0, -1, 0, 1, 0}, {100, 'g', 0x687579, 0, -1, 0, 1, 0}, {101, 'g', 0x68757a, 0, -1, 0, 1, 0},
		{102, 'g', 0x68757b, 0, -1, 0, 1, 0}, {103, 'g', 0x68757c, 0, -1, 0, 1, 0}, {104, 'g', 0x683feb, 0, -1, 0, 1, 0}, {105, 'g', 0x683fec, 0, -1, 0, 1, 0},
		{106, 'g', 0x683fed, 0, -1, 0, 1, 0}, {107, 'g', 0x683fee, 0, -1, 0, 1, 0}, {108, 'g', 0x683fef, 0, -1, 0, 1, 0}, {109, 'g', 0x683ff0, 0, -1, 0, 1, 0},
		{110, 'g', 0x683ff1, 0, -1, 0, 1, 0}, {111, 'g', 0x683ff2, 0, -1, 0, 1, 0}, {112, 'g', 0x683ff3, 0, -1, 0, 1, 0}, {113, 'g', 0x683ff4, 0, -1, 0, 1, 0},
		{114, 'g', 0x683ff5, 0, -1, 0, 1, 0}, {115, 'g', 0x683ff6, 0, -1, 0, 1, 0}, {116, 'g', 0x683ff7, 0, -1, 0, 1, 0}, {117, 'g', 0x683ff8, 0, -1, 0, 1, 0},
		{118, 'g', 0x683ff9, 0, -1, 0, 1, 0}, {119, 'g', 0x683ffa, 0, -1, 0, 1, 0}, {120, 'g', 0x683ffb, 0, -1, 0, 1, 0}, {121, 'g', 0x683ffc, 0, -1, 0, 1, 0},
		{122, 'g', 0x683ffd, 0, -1, 0, 1, 0}, {123, 'g', 0x683ffe, 0, -1, 0, 1, 0}, {124, 'g', 0x683fff, 0, -1, 0, 1, 0}, {125, 'g', 0x684000, 0, -1, 0, 1, 0},
		{126, 'g', 0x684001, 0, -1, 0, 1, 0}, {127, 'g', 0x684002, 0, -1, 0, 1, 0}, {128, 'g', 0x68400b, 0, -1, 0, 1, 0}, {129, 'g', 0x68400c, 0, -1, 0, 1, 0},
		{130, 'g', 0x68400d, 0, -1, 0, 1, 0}, {131, 'g', 0x68400e, 0, -1, 0, 1, 0}, {132, 'g', 0x684017, 0, -1, 0, 1, 0}, {133, 'g', 0x684018, 0, -1, 0, 1, 0},
		{134, 'g', 0x684019, 0, -1, 0, 1, 0}, {135, 'g', 0x68401a, 0, -1, 0, 1, 0}, {136, 'g', 0x684023, 0, -1, 0, 1, 0}, {137, 'g', 0x684024, 0, -1, 0, 1, 0},
		{138, 'g', 0x684025, 0, -1, 0, 1, 0}, {139, 'g', 0x684026, 0, -1, 0, 1, 0}, {140, 'g', 0x68402f, 0, -1, 0, 1, 0}, {141, 'g', 0x684030, 0, -1, 0, 1, 0},
		{142, 'g', 0x684031, 0, -1, 0, 1, 0}, {143, 'g', 0x684032, 0, -1, 0, 1, 0}, {144, 'g', 0x68403b, 0, -1, 0, 1, 0}, {145, 'g', 0x68403c, 0, -1, 0, 1, 0},
		{146, 'g', 0x68403d, 0, -1, 0, 1, 0}, {147, 'g', 0x68403e, 0, -1, 0, 1, 0}, {148, 'g', 0x684047, 0, -1, 0, 1, 0}, {149, 'g', 0x684048, 0, -1, 0, 1, 0},
		{150, 'g', 0x684049, 0, -1, 0, 1, 0}, {151, 'g', 0x68404a, 0, -1, 0, 1, 0}, {152, 'g', 0x684053, 0, -1, 0, 1, 0}, {153, 'g', 0x684054, 0, -1, 0, 1, 0},
		{154, 'g', 0x684055, 0, -1, 0, 1, 0}, {155, 'g', 0x684056, 0, -1, 0, 1, 0},
	}},
	0x523330: {characterConditionRemoveNode, 408, 0x0, []ctrsLink{
		{0, 'g', 0x685174, 0, -1, 0, 4, 0}, {1, 'a', 0, 0, -1, 0, 4, 0}, {2, 'a', 1, 0, -1, 0, 4, 0}, {3, 'a', 2, 0, -1, 0, 1, 0},
		{4, 'g', 0x683f94, 0, -1, 0, 1, 0}, {5, 'g', 0x687500, 0, -1, 0, 1, 0}, {6, 'o', 0x685178, 0x0, -1, 0x0, 1, 0}, {7, 'o', 0x685178, 0x106, -1, 0x0, 1, 0},
		{8, 'o', 0x685178, 0x20c, -1, 0x0, 1, 0}, {9, 'o', 0x685178, 0x312, -1, 0x0, 1, 0}, {10, 'o', 0x685178, 0x418, -1, 0x0, 1, 0}, {11, 'o', 0x685178, 0x51e, -1, 0x0, 1, 0},
		{12, 'o', 0x685178, 0x624, -1, 0x0, 1, 0}, {13, 'o', 0x685178, 0x72a, -1, 0x0, 1, 0}, {14, 'o', 0x685178, 0x0, 1, 0x106, 1, 0}, {15, 'o', 0x685174, 0xa01, 1, 0x1862, 4, 0},
		{16, 'o', 0x685174, 0xa05, 1, 0x1862, 4, 0}, {17, 'o', 0x685174, 0xa09, 1, 0x1862, 4, 0}, {18, 'o', 0x685174, 0xa0d, 1, 0x1862, 4, 0}, {19, 'o', 0x685174, 0xa11, 1, 0x1862, 4, 0},
		{20, 'o', 0x685174, 0xa15, 1, 0x1862, 4, 0}, {21, 'o', 0x685174, 0xa19, 1, 0x1862, 4, 0}, {22, 'o', 0x685174, 0xa1d, 1, 0x1862, 4, 0}, {23, 'o', 0x685174, 0xa21, 1, 0x1862, 4, 0},
		{24, 'o', 0x685174, 0xa25, 1, 0x1862, 4, 0}, {25, 'o', 0x685174, 0xa29, 1, 0x1862, 4, 0}, {26, 'o', 0x685174, 0xa2d, 1, 0x1862, 4, 0}, {27, 'o', 0x685174, 0xa31, 1, 0x1862, 4, 0},
		{28, 'o', 0x685174, 0xa35, 1, 0x1862, 4, 0}, {29, 'o', 0x685174, 0xa39, 1, 0x1862, 4, 0}, {30, 'o', 0x685174, 0xa3d, 1, 0x1862, 4, 0}, {31, 'o', 0x685174, 0xa41, 1, 0x1862, 4, 0},
		{32, 'o', 0x685174, 0xa45, 1, 0x1862, 4, 0}, {33, 'o', 0x685174, 0xa49, 1, 0x1862, 4, 0}, {34, 'o', 0x685174, 0xa4d, 1, 0x1862, 4, 0}, {35, 'o', 0x685174, 0xb01, 1, 0x1862, 4, 0},
		{36, 'o', 0x685174, 0xb11, 1, 0x1862, 4, 0}, {37, 'o', 0x685174, 0xb09, 1, 0x1862, 4, 0}, {38, 'o', 0x685174, 0xb15, 1, 0x1862, 4, 0}, {39, 'o', 0x685174, 0xb21, 1, 0x1862, 4, 0},
		{40, 'g', 0x686a70, 0, -1, 0, 4, 0}, {41, 'g', 0x6840bb, 0, -1, 0, 1, 0}, {42, 'g', 0x6874ca, 0, -1, 0, 1, 0}, {43, 'g', 0x6874cb, 0, -1, 0, 4, 0},
		{44, 'o', 0x685174, 0xf5d, -1, 0x0, 4, 0}, {45, 'o', 0x685174, 0xf69, -1, 0x0, 4, 0}, {46, 'o', 0x685174, 0xf75, -1, 0x0, 4, 0}, {47, 'o', 0x685174, 0xf81, -1, 0x0, 4, 0},
		{48, 'o', 0x685174, 0xf8d, -1, 0x0, 4, 0}, {49, 'o', 0x685174, 0xf99, -1, 0x0, 4, 0}, {50, 'o', 0x685174, 0xfa5, -1, 0x0, 4, 0}, {51, 'o', 0x685174, 0xfb1, -1, 0x0, 4, 0},
		{52, 'o', 0x685174, 0xfbd, -1, 0x0, 4, 0}, {53, 'o', 0x685174, 0xfc9, -1, 0x0, 4, 0}, {54, 'o', 0x685174, 0xfd5, -1, 0x0, 4, 0}, {55, 'o', 0x685174, 0xfe1, -1, 0x0, 4, 0},
		{56, 'o', 0x685174, 0x1029, -1, 0x0, 4, 0}, {57, 'o', 0x685174, 0x1035, -1, 0x0, 4, 0}, {58, 'o', 0x685174, 0x1041, -1, 0x0, 4, 0}, {59, 'o', 0x685174, 0x104d, -1, 0x0, 4, 0},
		{60, 'o', 0x685174, 0x1059, -1, 0x0, 4, 0}, {61, 'o', 0x685174, 0x1065, -1, 0x0, 4, 0}, {62, 'o', 0x685174, 0x1071, -1, 0x0, 4, 0}, {63, 'o', 0x685174, 0x107d, -1, 0x0, 4, 0},
		{64, 'o', 0x685174, 0x27bf, -1, 0x0, 4, 0}, {65, 'o', 0x685174, 0x27cb, -1, 0x0, 4, 0}, {66, 'o', 0x685174, 0x27d7, -1, 0x0, 4, 0}, {67, 'o', 0x685174, 0x27e3, -1, 0x0, 4, 0},
		{68, 'o', 0x685174, 0x27ef, -1, 0x0, 4, 0}, {69, 'o', 0x685174, 0x27fb, -1, 0x0, 4, 0}, {70, 'o', 0x685174, 0x2807, -1, 0x0, 4, 0}, {71, 'o', 0x685174, 0x2813, -1, 0x0, 4, 0},
		{72, 'o', 0x685174, 0x281f, -1, 0x0, 4, 0}, {73, 'o', 0x685174, 0x282b, -1, 0x0, 4, 0}, {74, 'o', 0x685174, 0x2837, -1, 0x0, 4, 0}, {75, 'o', 0x685174, 0x2843, -1, 0x0, 4, 0},
		{76, 'o', 0x685174, 0x288b, -1, 0x0, 4, 0}, {77, 'o', 0x685174, 0x2897, -1, 0x0, 4, 0}, {78, 'o', 0x685174, 0x28a3, -1, 0x0, 4, 0}, {79, 'o', 0x685174, 0x28af, -1, 0x0, 4, 0},
		{80, 'o', 0x685174, 0x28bb, -1, 0x0, 4, 0}, {81, 'o', 0x685174, 0x28c7, -1, 0x0, 4, 0}, {82, 'o', 0x685174, 0x28d3, -1, 0x0, 4, 0}, {83, 'o', 0x685174, 0x28df, -1, 0x0, 4, 0},
		{84, 'o', 0x685174, 0x4021, -1, 0x0, 4, 0}, {85, 'o', 0x685174, 0x402d, -1, 0x0, 4, 0}, {86, 'o', 0x685174, 0x4039, -1, 0x0, 4, 0}, {87, 'o', 0x685174, 0x4045, -1, 0x0, 4, 0},
		{88, 'o', 0x685174, 0x4051, -1, 0x0, 4, 0}, {89, 'o', 0x685174, 0x405d, -1, 0x0, 4, 0}, {90, 'o', 0x685174, 0x4069, -1, 0x0, 4, 0}, {91, 'o', 0x685174, 0x4075, -1, 0x0, 4, 0},
		{92, 'o', 0x685174, 0x4081, -1, 0x0, 4, 0}, {93, 'o', 0x685174, 0x408d, -1, 0x0, 4, 0}, {94, 'o', 0x685174, 0x4099, -1, 0x0, 4, 0}, {95, 'o', 0x685174, 0x40a5, -1, 0x0, 4, 0},
		{96, 'o', 0x685174, 0x40ed, -1, 0x0, 4, 0}, {97, 'o', 0x685174, 0x40f9, -1, 0x0, 4, 0}, {98, 'o', 0x685174, 0x4105, -1, 0x0, 4, 0}, {99, 'o', 0x685174, 0x4111, -1, 0x0, 4, 0},
		{100, 'o', 0x685174, 0x411d, -1, 0x0, 4, 0}, {101, 'o', 0x685174, 0x4129, -1, 0x0, 4, 0}, {102, 'o', 0x685174, 0x4135, -1, 0x0, 4, 0}, {103, 'o', 0x685174, 0x4141, -1, 0x0, 4, 0},
		{104, 'o', 0x685174, 0x5883, -1, 0x0, 4, 0}, {105, 'o', 0x685174, 0x588f, -1, 0x0, 4, 0}, {106, 'o', 0x685174, 0x589b, -1, 0x0, 4, 0}, {107, 'o', 0x685174, 0x58a7, -1, 0x0, 4, 0},
		{108, 'o', 0x685174, 0x58b3, -1, 0x0, 4, 0}, {109, 'o', 0x685174, 0x58bf, -1, 0x0, 4, 0}, {110, 'o', 0x685174, 0x58cb, -1, 0x0, 4, 0}, {111, 'o', 0x685174, 0x58d7, -1, 0x0, 4, 0},
		{112, 'o', 0x685174, 0x58e3, -1, 0x0, 4, 0}, {113, 'o', 0x685174, 0x58ef, -1, 0x0, 4, 0}, {114, 'o', 0x685174, 0x58fb, -1, 0x0, 4, 0}, {115, 'o', 0x685174, 0x5907, -1, 0x0, 4, 0},
		{116, 'o', 0x685174, 0x594f, -1, 0x0, 4, 0}, {117, 'o', 0x685174, 0x595b, -1, 0x0, 4, 0}, {118, 'o', 0x685174, 0x5967, -1, 0x0, 4, 0}, {119, 'o', 0x685174, 0x5973, -1, 0x0, 4, 0},
		{120, 'o', 0x685174, 0x597f, -1, 0x0, 4, 0}, {121, 'o', 0x685174, 0x598b, -1, 0x0, 4, 0}, {122, 'o', 0x685174, 0x5997, -1, 0x0, 4, 0}, {123, 'o', 0x685174, 0x59a3, -1, 0x0, 4, 0},
		{124, 'o', 0x685174, 0x70e5, -1, 0x0, 4, 0}, {125, 'o', 0x685174, 0x70f1, -1, 0x0, 4, 0}, {126, 'o', 0x685174, 0x70fd, -1, 0x0, 4, 0}, {127, 'o', 0x685174, 0x7109, -1, 0x0, 4, 0},
		{128, 'o', 0x685174, 0x7115, -1, 0x0, 4, 0}, {129, 'o', 0x685174, 0x7121, -1, 0x0, 4, 0}, {130, 'o', 0x685174, 0x712d, -1, 0x0, 4, 0}, {131, 'o', 0x685174, 0x7139, -1, 0x0, 4, 0},
		{132, 'o', 0x685174, 0x7145, -1, 0x0, 4, 0}, {133, 'o', 0x685174, 0x7151, -1, 0x0, 4, 0}, {134, 'o', 0x685174, 0x715d, -1, 0x0, 4, 0}, {135, 'o', 0x685174, 0x7169, -1, 0x0, 4, 0},
		{136, 'o', 0x685174, 0x71b1, -1, 0x0, 4, 0}, {137, 'o', 0x685174, 0x71bd, -1, 0x0, 4, 0}, {138, 'o', 0x685174, 0x71c9, -1, 0x0, 4, 0}, {139, 'o', 0x685174, 0x71d5, -1, 0x0, 4, 0},
		{140, 'o', 0x685174, 0x71e1, -1, 0x0, 4, 0}, {141, 'o', 0x685174, 0x71ed, -1, 0x0, 4, 0}, {142, 'o', 0x685174, 0x71f9, -1, 0x0, 4, 0}, {143, 'o', 0x685174, 0x7205, -1, 0x0, 4, 0},
		{144, 'o', 0x685174, 0x8947, -1, 0x0, 4, 0}, {145, 'o', 0x685174, 0x8953, -1, 0x0, 4, 0}, {146, 'o', 0x685174, 0x895f, -1, 0x0, 4, 0}, {147, 'o', 0x685174, 0x896b, -1, 0x0, 4, 0},
		{148, 'o', 0x685174, 0x8977, -1, 0x0, 4, 0}, {149, 'o', 0x685174, 0x8983, -1, 0x0, 4, 0}, {150, 'o', 0x685174, 0x898f, -1, 0x0, 4, 0}, {151, 'o', 0x685174, 0x899b, -1, 0x0, 4, 0},
		{152, 'o', 0x685174, 0x89a7, -1, 0x0, 4, 0}, {153, 'o', 0x685174, 0x89b3, -1, 0x0, 4, 0}, {154, 'o', 0x685174, 0x89bf, -1, 0x0, 4, 0}, {155, 'o', 0x685174, 0x89cb, -1, 0x0, 4, 0},
		{156, 'o', 0x685174, 0x8a13, -1, 0x0, 4, 0}, {157, 'o', 0x685174, 0x8a1f, -1, 0x0, 4, 0}, {158, 'o', 0x685174, 0x8a2b, -1, 0x0, 4, 0}, {159, 'o', 0x685174, 0x8a37, -1, 0x0, 4, 0},
		{160, 'o', 0x685174, 0x8a43, -1, 0x0, 4, 0}, {161, 'o', 0x685174, 0x8a4f, -1, 0x0, 4, 0}, {162, 'o', 0x685174, 0x8a5b, -1, 0x0, 4, 0}, {163, 'o', 0x685174, 0x8a67, -1, 0x0, 4, 0},
		{164, 'o', 0x685174, 0xa1a9, -1, 0x0, 4, 0}, {165, 'o', 0x685174, 0xa1b5, -1, 0x0, 4, 0}, {166, 'o', 0x685174, 0xa1c1, -1, 0x0, 4, 0}, {167, 'o', 0x685174, 0xa1cd, -1, 0x0, 4, 0},
		{168, 'o', 0x685174, 0xa1d9, -1, 0x0, 4, 0}, {169, 'o', 0x685174, 0xa1e5, -1, 0x0, 4, 0}, {170, 'o', 0x685174, 0xa1f1, -1, 0x0, 4, 0}, {171, 'o', 0x685174, 0xa1fd, -1, 0x0, 4, 0},
		{172, 'o', 0x685174, 0xa209, -1, 0x0, 4, 0}, {173, 'o', 0x685174, 0xa215, -1, 0x0, 4, 0}, {174, 'o', 0x685174, 0xa221, -1, 0x0, 4, 0}, {175, 'o', 0x685174, 0xa22d, -1, 0x0, 4, 0},
		{176, 'o', 0x685174, 0xa275, -1, 0x0, 4, 0}, {177, 'o', 0x685174, 0xa281, -1, 0x0, 4, 0}, {178, 'o', 0x685174, 0xa28d, -1, 0x0, 4, 0}, {179, 'o', 0x685174, 0xa299, -1, 0x0, 4, 0},
		{180, 'o', 0x685174, 0xa2a5, -1, 0x0, 4, 0}, {181, 'o', 0x685174, 0xa2b1, -1, 0x0, 4, 0}, {182, 'o', 0x685174, 0xa2bd, -1, 0x0, 4, 0}, {183, 'o', 0x685174, 0xa2c9, -1, 0x0, 4, 0},
		{184, 'o', 0x685174, 0xba0b, -1, 0x0, 4, 0}, {185, 'o', 0x685174, 0xba17, -1, 0x0, 4, 0}, {186, 'o', 0x685174, 0xba23, -1, 0x0, 4, 0}, {187, 'o', 0x685174, 0xba2f, -1, 0x0, 4, 0},
		{188, 'o', 0x685174, 0xba3b, -1, 0x0, 4, 0}, {189, 'o', 0x685174, 0xba47, -1, 0x0, 4, 0}, {190, 'o', 0x685174, 0xba53, -1, 0x0, 4, 0}, {191, 'o', 0x685174, 0xba5f, -1, 0x0, 4, 0},
		{192, 'o', 0x685174, 0xba6b, -1, 0x0, 4, 0}, {193, 'o', 0x685174, 0xba77, -1, 0x0, 4, 0}, {194, 'o', 0x685174, 0xba83, -1, 0x0, 4, 0}, {195, 'o', 0x685174, 0xba8f, -1, 0x0, 4, 0},
		{196, 'o', 0x685174, 0xbad7, -1, 0x0, 4, 0}, {197, 'o', 0x685174, 0xbae3, -1, 0x0, 4, 0}, {198, 'o', 0x685174, 0xbaef, -1, 0x0, 4, 0}, {199, 'o', 0x685174, 0xbafb, -1, 0x0, 4, 0},
		{200, 'o', 0x685174, 0xbb07, -1, 0x0, 4, 0}, {201, 'o', 0x685174, 0xbb13, -1, 0x0, 4, 0}, {202, 'o', 0x685174, 0xbb1f, -1, 0x0, 4, 0}, {203, 'o', 0x685174, 0xbb2b, -1, 0x0, 4, 0},
		{204, 'g', 0x686901, 0, -1, 0, 4, 0}, {205, 'g', 0x685191, 0, -1, 0, 4, 0}, {206, 'g', 0x68519d, 0, -1, 0, 4, 0}, {207, 'g', 0x6851a9, 0, -1, 0, 4, 0},
		{208, 'g', 0x6851b5, 0, -1, 0, 4, 0}, {209, 'g', 0x6851c1, 0, -1, 0, 4, 0}, {210, 'g', 0x6851cd, 0, -1, 0, 4, 0}, {211, 'g', 0x689d46, 0, -1, 0, 1, 0},
		{213, 'o', 0x685178, 0x4d, 1, 0x106, 4, 0}, {214, 'o', 0x685178, 0x51, 1, 0x106, 4, 0}, {215, 'o', 0x685178, 0x55, 1, 0x106, 4, 0}, {216, 'o', 0x685178, 0x59, 1, 0x106, 4, 0},
		{217, 'o', 0x685178, 0x5d, 1, 0x106, 4, 0}, {218, 'o', 0x685178, 0x61, 1, 0x106, 4, 0}, {219, 'o', 0x685178, 0x65, 1, 0x106, 4, 0}, {220, 'o', 0x685178, 0x69, 1, 0x106, 4, 0},
		{221, 'o', 0x685178, 0x1d, 1, 0x106, 4, 0}, {222, 'o', 0x685178, 0x21, 1, 0x106, 4, 0}, {223, 'o', 0x685178, 0x25, 1, 0x106, 4, 0}, {224, 'o', 0x685178, 0x29, 1, 0x106, 4, 0},
		{225, 'o', 0x685178, 0x2d, 1, 0x106, 4, 0}, {226, 'o', 0x685178, 0x31, 1, 0x106, 4, 0}, {227, 'o', 0x685178, 0x35, 1, 0x106, 4, 0}, {228, 'o', 0x685178, 0x39, 1, 0x106, 4, 0},
		{229, 'c', 0x20000000, 0, -1, 0, 4, 0}, {231, 'g', 0x68ec78, 0, -1, 0, 4, 0}, {232, 'g', 0x68edcc, 0, -1, 0, 4, 0}, {234, 'o', 0x69c0f4, 0xf8, -1, 0x0, 4, 0},
		{235, 'g', 0x683fc5, 0, -1, 0, 1, 0}, {236, 'g', 0x683745, 0, -1, 0, 4, 0}, {237, 'g', 0x68385d, 0, -1, 0, 4, 0}, {238, 'g', 0x683975, 0, -1, 0, 4, 0},
		{239, 'g', 0x683a8d, 0, -1, 0, 4, 0}, {240, 'g', 0x683ba5, 0, -1, 0, 4, 0}, {241, 'g', 0x683cbd, 0, -1, 0, 4, 0}, {242, 'g', 0x683dd5, 0, -1, 0, 4, 0},
		{243, 'g', 0x683eed, 0, -1, 0, 4, 0}, {244, 'g', 0x683751, 0, -1, 0, 1, 0}, {245, 'g', 0x683869, 0, -1, 0, 1, 0}, {246, 'g', 0x683981, 0, -1, 0, 1, 0},
		{247, 'g', 0x683a99, 0, -1, 0, 1, 0}, {248, 'g', 0x683bb1, 0, -1, 0, 1, 0}, {249, 'g', 0x683cc9, 0, -1, 0, 1, 0}, {250, 'g', 0x683de1, 0, -1, 0, 1, 0},
		{251, 'g', 0x683ef9, 0, -1, 0, 1, 0}, {252, 'g', 0x687511, 0, -1, 0, 1, 0}, {253, 'g', 0x687512, 0, -1, 0, 1, 0}, {254, 'g', 0x687513, 0, -1, 0, 1, 0},
		{255, 'g', 0x687514, 0, -1, 0, 1, 0}, {256, 'g', 0x687515, 0, -1, 0, 1, 0}, {257, 'g', 0x687516, 0, -1, 0, 1, 0}, {258, 'g', 0x687517, 0, -1, 0, 1, 0},
		{259, 'g', 0x687518, 0, -1, 0, 1, 0}, {260, 'g', 0x687519, 0, -1, 0, 1, 0}, {261, 'g', 0x68751a, 0, -1, 0, 1, 0}, {262, 'g', 0x68751b, 0, -1, 0, 1, 0},
		{263, 'g', 0x68751c, 0, -1, 0, 1, 0}, {264, 'g', 0x68751d, 0, -1, 0, 1, 0}, {265, 'g', 0x68751e, 0, -1, 0, 1, 0}, {266, 'g', 0x68751f, 0, -1, 0, 1, 0},
		{267, 'g', 0x687520, 0, -1, 0, 1, 0}, {268, 'g', 0x687521, 0, -1, 0, 1, 0}, {269, 'g', 0x687522, 0, -1, 0, 1, 0}, {270, 'g', 0x687523, 0, -1, 0, 1, 0},
		{271, 'g', 0x687524, 0, -1, 0, 1, 0}, {272, 'g', 0x687525, 0, -1, 0, 1, 0}, {273, 'g', 0x687526, 0, -1, 0, 1, 0}, {274, 'g', 0x687527, 0, -1, 0, 1, 0},
		{275, 'g', 0x687528, 0, -1, 0, 1, 0}, {276, 'g', 0x687531, 0, -1, 0, 1, 0}, {277, 'g', 0x687532, 0, -1, 0, 1, 0}, {278, 'g', 0x687533, 0, -1, 0, 1, 0},
		{279, 'g', 0x687534, 0, -1, 0, 1, 0}, {280, 'g', 0x68753d, 0, -1, 0, 1, 0}, {281, 'g', 0x68753e, 0, -1, 0, 1, 0}, {282, 'g', 0x68753f, 0, -1, 0, 1, 0},
		{283, 'g', 0x687540, 0, -1, 0, 1, 0}, {284, 'g', 0x687549, 0, -1, 0, 1, 0}, {285, 'g', 0x68754a, 0, -1, 0, 1, 0}, {286, 'g', 0x68754b, 0, -1, 0, 1, 0},
		{287, 'g', 0x68754c, 0, -1, 0, 1, 0}, {288, 'g', 0x687555, 0, -1, 0, 1, 0}, {289, 'g', 0x687556, 0, -1, 0, 1, 0}, {290, 'g', 0x687557, 0, -1, 0, 1, 0},
		{291, 'g', 0x687558, 0, -1, 0, 1, 0}, {292, 'g', 0x687561, 0, -1, 0, 1, 0}, {293, 'g', 0x687562, 0, -1, 0, 1, 0}, {294, 'g', 0x687563, 0, -1, 0, 1, 0},
		{295, 'g', 0x687564, 0, -1, 0, 1, 0}, {296, 'g', 0x68756d, 0, -1, 0, 1, 0}, {297, 'g', 0x68756e, 0, -1, 0, 1, 0}, {298, 'g', 0x68756f, 0, -1, 0, 1, 0},
		{299, 'g', 0x687570, 0, -1, 0, 1, 0}, {300, 'g', 0x687579, 0, -1, 0, 1, 0}, {301, 'g', 0x68757a, 0, -1, 0, 1, 0}, {302, 'g', 0x68757b, 0, -1, 0, 1, 0},
		{303, 'g', 0x68757c, 0, -1, 0, 1, 0}, {304, 'g', 0x683feb, 0, -1, 0, 1, 0}, {305, 'g', 0x683fec, 0, -1, 0, 1, 0}, {306, 'g', 0x683fed, 0, -1, 0, 1, 0},
		{307, 'g', 0x683fee, 0, -1, 0, 1, 0}, {308, 'g', 0x683fef, 0, -1, 0, 1, 0}, {309, 'g', 0x683ff0, 0, -1, 0, 1, 0}, {310, 'g', 0x683ff1, 0, -1, 0, 1, 0},
		{311, 'g', 0x683ff2, 0, -1, 0, 1, 0}, {312, 'g', 0x683ff3, 0, -1, 0, 1, 0}, {313, 'g', 0x683ff4, 0, -1, 0, 1, 0}, {314, 'g', 0x683ff5, 0, -1, 0, 1, 0},
		{315, 'g', 0x683ff6, 0, -1, 0, 1, 0}, {316, 'g', 0x683ff7, 0, -1, 0, 1, 0}, {317, 'g', 0x683ff8, 0, -1, 0, 1, 0}, {318, 'g', 0x683ff9, 0, -1, 0, 1, 0},
		{319, 'g', 0x683ffa, 0, -1, 0, 1, 0}, {320, 'g', 0x683ffb, 0, -1, 0, 1, 0}, {321, 'g', 0x683ffc, 0, -1, 0, 1, 0}, {322, 'g', 0x683ffd, 0, -1, 0, 1, 0},
		{323, 'g', 0x683ffe, 0, -1, 0, 1, 0}, {324, 'g', 0x683fff, 0, -1, 0, 1, 0}, {325, 'g', 0x684000, 0, -1, 0, 1, 0}, {326, 'g', 0x684001, 0, -1, 0, 1, 0},
		{327, 'g', 0x684002, 0, -1, 0, 1, 0}, {328, 'g', 0x68400b, 0, -1, 0, 1, 0}, {329, 'g', 0x68400c, 0, -1, 0, 1, 0}, {330, 'g', 0x68400d, 0, -1, 0, 1, 0},
		{331, 'g', 0x68400e, 0, -1, 0, 1, 0}, {332, 'g', 0x684017, 0, -1, 0, 1, 0}, {333, 'g', 0x684018, 0, -1, 0, 1, 0}, {334, 'g', 0x684019, 0, -1, 0, 1, 0},
		{335, 'g', 0x68401a, 0, -1, 0, 1, 0}, {336, 'g', 0x684023, 0, -1, 0, 1, 0}, {337, 'g', 0x684024, 0, -1, 0, 1, 0}, {338, 'g', 0x684025, 0, -1, 0, 1, 0},
		{339, 'g', 0x684026, 0, -1, 0, 1, 0}, {340, 'g', 0x68402f, 0, -1, 0, 1, 0}, {341, 'g', 0x684030, 0, -1, 0, 1, 0}, {342, 'g', 0x684031, 0, -1, 0, 1, 0},
		{343, 'g', 0x684032, 0, -1, 0, 1, 0}, {344, 'g', 0x68403b, 0, -1, 0, 1, 0}, {345, 'g', 0x68403c, 0, -1, 0, 1, 0}, {346, 'g', 0x68403d, 0, -1, 0, 1, 0},
		{347, 'g', 0x68403e, 0, -1, 0, 1, 0}, {348, 'g', 0x684047, 0, -1, 0, 1, 0}, {349, 'g', 0x684048, 0, -1, 0, 1, 0}, {350, 'g', 0x684049, 0, -1, 0, 1, 0},
		{351, 'g', 0x68404a, 0, -1, 0, 1, 0}, {352, 'g', 0x684053, 0, -1, 0, 1, 0}, {353, 'g', 0x684054, 0, -1, 0, 1, 0}, {354, 'g', 0x684055, 0, -1, 0, 1, 0},
		{355, 'g', 0x684056, 0, -1, 0, 1, 0},
	}, []ctrsLink{
		{0, 'o', 0x685174, 0xa01, 1, 0x1862, 4, 0}, {1, 'o', 0x685174, 0xa05, 1, 0x1862, 4, 0}, {2, 'o', 0x685174, 0xa09, 1, 0x1862, 4, 0}, {3, 'o', 0x685174, 0xa0d, 1, 0x1862, 4, 0},
		{4, 'o', 0x685174, 0xa11, 1, 0x1862, 4, 0}, {5, 'o', 0x685174, 0xa15, 1, 0x1862, 4, 0}, {6, 'o', 0x685174, 0xa19, 1, 0x1862, 4, 0}, {7, 'o', 0x685174, 0xa1d, 1, 0x1862, 4, 0},
		{8, 'o', 0x685174, 0xa21, 1, 0x1862, 4, 0}, {9, 'o', 0x685174, 0xa25, 1, 0x1862, 4, 0}, {10, 'o', 0x685174, 0xa29, 1, 0x1862, 4, 0}, {11, 'o', 0x685174, 0xa2d, 1, 0x1862, 4, 0},
		{12, 'o', 0x685174, 0xa31, 1, 0x1862, 4, 0}, {13, 'o', 0x685174, 0xa35, 1, 0x1862, 4, 0}, {14, 'o', 0x685174, 0xa39, 1, 0x1862, 4, 0}, {15, 'o', 0x685174, 0xa3d, 1, 0x1862, 4, 0},
		{16, 'o', 0x685174, 0xa41, 1, 0x1862, 4, 0}, {17, 'o', 0x685174, 0xa45, 1, 0x1862, 4, 0}, {18, 'o', 0x685174, 0xa49, 1, 0x1862, 4, 0}, {19, 'o', 0x685174, 0xa4d, 1, 0x1862, 4, 0},
		{20, 'o', 0x685174, 0xb09, 1, 0x1862, 4, 0}, {21, 'o', 0x685174, 0xb15, 1, 0x1862, 4, 0}, {22, 'o', 0x685174, 0xb21, 1, 0x1862, 4, 0}, {23, 'g', 0x6840bb, 0, -1, 0, 1, 0},
		{24, 'o', 0x685178, 0x1d, 1, 0x106, 4, 0}, {25, 'o', 0x685178, 0x21, 1, 0x106, 4, 0}, {26, 'o', 0x685178, 0x25, 1, 0x106, 4, 0}, {27, 'o', 0x685178, 0x29, 1, 0x106, 4, 0},
		{28, 'o', 0x685178, 0x2d, 1, 0x106, 4, 0}, {29, 'o', 0x685178, 0x31, 1, 0x106, 4, 0}, {30, 'o', 0x685178, 0x35, 1, 0x106, 4, 0}, {31, 'o', 0x685178, 0x39, 1, 0x106, 4, 0},
		{32, 'g', 0x683745, 0, -1, 0, 4, 0}, {33, 'g', 0x68385d, 0, -1, 0, 4, 0}, {34, 'g', 0x683975, 0, -1, 0, 4, 0}, {35, 'g', 0x683a8d, 0, -1, 0, 4, 0},
		{36, 'g', 0x683ba5, 0, -1, 0, 4, 0}, {37, 'g', 0x683cbd, 0, -1, 0, 4, 0}, {38, 'g', 0x683dd5, 0, -1, 0, 4, 0}, {39, 'g', 0x683eed, 0, -1, 0, 4, 0},
		{40, 'g', 0x683751, 0, -1, 0, 1, 0}, {41, 'g', 0x683869, 0, -1, 0, 1, 0}, {42, 'g', 0x683981, 0, -1, 0, 1, 0}, {43, 'g', 0x683a99, 0, -1, 0, 1, 0},
		{44, 'g', 0x683bb1, 0, -1, 0, 1, 0}, {45, 'g', 0x683cc9, 0, -1, 0, 1, 0}, {46, 'g', 0x683de1, 0, -1, 0, 1, 0}, {47, 'g', 0x683ef9, 0, -1, 0, 1, 0},
		{49, 'o', 0x69c0f4, 0xf8, -1, 0x0, 4, 0}, {50, 'g', 0x687511, 0, -1, 0, 1, 0}, {51, 'g', 0x687512, 0, -1, 0, 1, 0}, {52, 'g', 0x687513, 0, -1, 0, 1, 0},
		{53, 'g', 0x687514, 0, -1, 0, 1, 0}, {54, 'g', 0x687515, 0, -1, 0, 1, 0}, {55, 'g', 0x687516, 0, -1, 0, 1, 0}, {56, 'g', 0x687517, 0, -1, 0, 1, 0},
		{57, 'g', 0x687518, 0, -1, 0, 1, 0}, {58, 'g', 0x687519, 0, -1, 0, 1, 0}, {59, 'g', 0x68751a, 0, -1, 0, 1, 0}, {60, 'g', 0x68751b, 0, -1, 0, 1, 0},
		{61, 'g', 0x68751c, 0, -1, 0, 1, 0}, {62, 'g', 0x68751d, 0, -1, 0, 1, 0}, {63, 'g', 0x68751e, 0, -1, 0, 1, 0}, {64, 'g', 0x68751f, 0, -1, 0, 1, 0},
		{65, 'g', 0x687520, 0, -1, 0, 1, 0}, {66, 'g', 0x687521, 0, -1, 0, 1, 0}, {67, 'g', 0x687522, 0, -1, 0, 1, 0}, {68, 'g', 0x687523, 0, -1, 0, 1, 0},
		{69, 'g', 0x687524, 0, -1, 0, 1, 0}, {70, 'g', 0x687525, 0, -1, 0, 1, 0}, {71, 'g', 0x687526, 0, -1, 0, 1, 0}, {72, 'g', 0x687527, 0, -1, 0, 1, 0},
		{73, 'g', 0x687528, 0, -1, 0, 1, 0}, {74, 'g', 0x687531, 0, -1, 0, 1, 0}, {75, 'g', 0x687532, 0, -1, 0, 1, 0}, {76, 'g', 0x687533, 0, -1, 0, 1, 0},
		{77, 'g', 0x687534, 0, -1, 0, 1, 0}, {78, 'g', 0x68753d, 0, -1, 0, 1, 0}, {79, 'g', 0x68753e, 0, -1, 0, 1, 0}, {80, 'g', 0x68753f, 0, -1, 0, 1, 0},
		{81, 'g', 0x687540, 0, -1, 0, 1, 0}, {82, 'g', 0x687549, 0, -1, 0, 1, 0}, {83, 'g', 0x68754a, 0, -1, 0, 1, 0}, {84, 'g', 0x68754b, 0, -1, 0, 1, 0},
		{85, 'g', 0x68754c, 0, -1, 0, 1, 0}, {86, 'g', 0x687555, 0, -1, 0, 1, 0}, {87, 'g', 0x687556, 0, -1, 0, 1, 0}, {88, 'g', 0x687557, 0, -1, 0, 1, 0},
		{89, 'g', 0x687558, 0, -1, 0, 1, 0}, {90, 'g', 0x687561, 0, -1, 0, 1, 0}, {91, 'g', 0x687562, 0, -1, 0, 1, 0}, {92, 'g', 0x687563, 0, -1, 0, 1, 0},
		{93, 'g', 0x687564, 0, -1, 0, 1, 0}, {94, 'g', 0x68756d, 0, -1, 0, 1, 0}, {95, 'g', 0x68756e, 0, -1, 0, 1, 0}, {96, 'g', 0x68756f, 0, -1, 0, 1, 0},
		{97, 'g', 0x687570, 0, -1, 0, 1, 0}, {98, 'g', 0x687579, 0, -1, 0, 1, 0}, {99, 'g', 0x68757a, 0, -1, 0, 1, 0}, {100, 'g', 0x68757b, 0, -1, 0, 1, 0},
		{101, 'g', 0x68757c, 0, -1, 0, 1, 0}, {102, 'g', 0x683feb, 0, -1, 0, 1, 0}, {103, 'g', 0x683fec, 0, -1, 0, 1, 0}, {104, 'g', 0x683fed, 0, -1, 0, 1, 0},
		{105, 'g', 0x683fee, 0, -1, 0, 1, 0}, {106, 'g', 0x683fef, 0, -1, 0, 1, 0}, {107, 'g', 0x683ff0, 0, -1, 0, 1, 0}, {108, 'g', 0x683ff1, 0, -1, 0, 1, 0},
		{109, 'g', 0x683ff2, 0, -1, 0, 1, 0}, {110, 'g', 0x683ff3, 0, -1, 0, 1, 0}, {111, 'g', 0x683ff4, 0, -1, 0, 1, 0}, {112, 'g', 0x683ff5, 0, -1, 0, 1, 0},
		{113, 'g', 0x683ff6, 0, -1, 0, 1, 0}, {114, 'g', 0x683ff7, 0, -1, 0, 1, 0}, {115, 'g', 0x683ff8, 0, -1, 0, 1, 0}, {116, 'g', 0x683ff9, 0, -1, 0, 1, 0},
		{117, 'g', 0x683ffa, 0, -1, 0, 1, 0}, {118, 'g', 0x683ffb, 0, -1, 0, 1, 0}, {119, 'g', 0x683ffc, 0, -1, 0, 1, 0}, {120, 'g', 0x683ffd, 0, -1, 0, 1, 0},
		{121, 'g', 0x683ffe, 0, -1, 0, 1, 0}, {122, 'g', 0x683fff, 0, -1, 0, 1, 0}, {123, 'g', 0x684000, 0, -1, 0, 1, 0}, {124, 'g', 0x684001, 0, -1, 0, 1, 0},
		{125, 'g', 0x684002, 0, -1, 0, 1, 0}, {126, 'g', 0x68400b, 0, -1, 0, 1, 0}, {127, 'g', 0x68400c, 0, -1, 0, 1, 0}, {128, 'g', 0x68400d, 0, -1, 0, 1, 0},
		{129, 'g', 0x68400e, 0, -1, 0, 1, 0}, {130, 'g', 0x684017, 0, -1, 0, 1, 0}, {131, 'g', 0x684018, 0, -1, 0, 1, 0}, {132, 'g', 0x684019, 0, -1, 0, 1, 0},
		{133, 'g', 0x68401a, 0, -1, 0, 1, 0}, {134, 'g', 0x684023, 0, -1, 0, 1, 0}, {135, 'g', 0x684024, 0, -1, 0, 1, 0}, {136, 'g', 0x684025, 0, -1, 0, 1, 0},
		{137, 'g', 0x684026, 0, -1, 0, 1, 0}, {138, 'g', 0x68402f, 0, -1, 0, 1, 0}, {139, 'g', 0x684030, 0, -1, 0, 1, 0}, {140, 'g', 0x684031, 0, -1, 0, 1, 0},
		{141, 'g', 0x684032, 0, -1, 0, 1, 0}, {142, 'g', 0x68403b, 0, -1, 0, 1, 0}, {143, 'g', 0x68403c, 0, -1, 0, 1, 0}, {144, 'g', 0x68403d, 0, -1, 0, 1, 0},
		{145, 'g', 0x68403e, 0, -1, 0, 1, 0}, {146, 'g', 0x684047, 0, -1, 0, 1, 0}, {147, 'g', 0x684048, 0, -1, 0, 1, 0}, {148, 'g', 0x684049, 0, -1, 0, 1, 0},
		{149, 'g', 0x68404a, 0, -1, 0, 1, 0}, {150, 'g', 0x684053, 0, -1, 0, 1, 0}, {151, 'g', 0x684054, 0, -1, 0, 1, 0}, {152, 'g', 0x684055, 0, -1, 0, 1, 0},
		{153, 'g', 0x684056, 0, -1, 0, 1, 0}, {206, 'o', 0x685174, 0xb01, 1, 0x1862, 4, 0},
	}},
	0x52add0: {characterHpHealNode, 11, 0x20000000, []ctrsLink{
		{0, 'a', 0, 0, -1, 0, 4, 0}, {1, 'a', 1, 0, -1, 0, 4, 0}, {2, 'o', 0x685174, 0xb11, 0, 0x1862, 4, 0}, {3, 'o', 0x685174, 0xb0d, 0, 0x1862, 4, 0},
		{4, 'a', 2, 0, -1, 0, 1, 0}, {5, 'g', 0x5ed904, 0, -1, 0, 4, 0}, {6, 'g', 0x5ed900, 0, -1, 0, 4, 0}, {10, 'o', 0x685178, 0x0, 0, 0x106, 1, 0},
	}, []ctrsLink{
		{0, 'o', 0x685174, 0xb11, 0, 0x1862, 4, 0},
	}},
	0x52b1c0: {characterStaminaHealNode, 12, 0x20000000, []ctrsLink{
		{0, 'c', 0x20000000, 0, -1, 0, 4, 0}, {1, 'a', 0, 0, -1, 0, 4, 0}, {2, 'a', 1, 0, -1, 0, 4, 1}, {3, 'a', 2, 0, -1, 0, 4, 0},
		{4, 'o', 0x685174, 0xb01, 1, 0x1862, 4, 0}, {5, 'o', 0x685174, 0xb11, 1, 0x1862, 4, 0}, {6, 'o', 0x685174, 0xb19, 1, 0x1862, 4, 0}, {7, 'o', 0x685174, 0xb1d, 1, 0x1862, 4, 0},
		{8, 'o', 0x685174, 0x169e, 1, 0x1862, 4, 0}, {9, 'o', 0x685174, 0xa45, 1, 0x1862, 4, 0},
	}, []ctrsLink{
		{0, 'o', 0x685174, 0xb1d, 1, 0x1862, 4, 0}, {1, 'o', 0x685174, 0x169e, 1, 0x1862, 4, 0},
	}},
	0x52af50: {characterFatigueApplyNode, 20, 0x20000000, []ctrsLink{
		{0, 'a', 0, 0, -1, 0, 4, 0}, {1, 'a', 1, 0, -1, 0, 4, 1}, {2, 'a', 2, 0, -1, 0, 4, 0}, {3, 'a', 3, 0, -1, 0, 4, 0},
		{4, 'o', 0x685174, 0xb1d, 0, 0x1862, 4, 1}, {5, 'o', 0x685174, 0xb11, 0, 0x1862, 4, 0}, {6, 'o', 0x685174, 0xb19, 0, 0x1862, 4, 0}, {7, 'o', 0x685174, 0xbc9, 0, 0x1862, 4, 0},
		{8, 'o', 0x685174, 0xa15, 0, 0x1862, 4, 0}, {9, 'o', 0x685174, 0xaa9, 0, 0x1862, 4, 0}, {10, 'o', 0x685174, 0x169e, 0, 0x1862, 4, 0}, {11, 'o', 0x685174, 0x185b, 0, 0x1862, 1, 0},
		{12, 'o', 0x685174, 0xa45, 0, 0x1862, 4, 0}, {14, 'g', 0x5ed8c8, 0, -1, 0, 4, 0}, {15, 'g', 0x5ed914, 0, -1, 0, 4, 0}, {16, 'g', 0x5ee598, 0, -1, 0, 4, 0},
	}, []ctrsLink{
		{0, 'o', 0x685174, 0xb1d, 0, 0x1862, 4, 1}, {1, 'o', 0x685174, 0x169e, 0, 0x1862, 4, 0}, {2, 'o', 0x685174, 0x185b, 0, 0x1862, 1, 0},
	}},
	0x52b4f0: {characterRealmSpHealNode, 61, 0x20000000, []ctrsLink{
		{0, 'a', 0, 0, -1, 0, 4, 0}, {1, 'a', 1, 0, -1, 0, 4, 0}, {2, 'a', 2, 0, -1, 0, 4, 1}, {3, 'o', 0x685174, 0xb45, -1, 0x0, 4, 1},
		{4, 'o', 0x685174, 0xb49, -1, 0x0, 4, 1}, {5, 'o', 0x685174, 0xb4d, -1, 0x0, 4, 1}, {6, 'o', 0x685174, 0xb51, -1, 0x0, 4, 1}, {7, 'o', 0x685174, 0xb55, -1, 0x0, 4, 1},
		{8, 'o', 0x685174, 0xb59, -1, 0x0, 4, 1}, {9, 'o', 0x685174, 0x23a7, -1, 0x0, 4, 1}, {10, 'o', 0x685174, 0x23ab, -1, 0x0, 4, 1}, {11, 'o', 0x685174, 0x23af, -1, 0x0, 4, 1},
		{12, 'o', 0x685174, 0x23b3, -1, 0x0, 4, 1}, {13, 'o', 0x685174, 0x23b7, -1, 0x0, 4, 1}, {14, 'o', 0x685174, 0x23bb, -1, 0x0, 4, 1}, {15, 'o', 0x685174, 0x3c09, -1, 0x0, 4, 1},
		{16, 'o', 0x685174, 0x3c0d, -1, 0x0, 4, 1}, {17, 'o', 0x685174, 0x3c11, -1, 0x0, 4, 1}, {18, 'o', 0x685174, 0x3c15, -1, 0x0, 4, 1}, {19, 'o', 0x685174, 0x3c19, -1, 0x0, 4, 1},
		{20, 'o', 0x685174, 0x3c1d, -1, 0x0, 4, 1}, {21, 'o', 0x685174, 0x546b, -1, 0x0, 4, 1}, {22, 'o', 0x685174, 0x546f, -1, 0x0, 4, 1}, {23, 'o', 0x685174, 0x5473, -1, 0x0, 4, 1},
		{24, 'o', 0x685174, 0x5477, -1, 0x0, 4, 1}, {25, 'o', 0x685174, 0x547b, -1, 0x0, 4, 1}, {26, 'o', 0x685174, 0x547f, -1, 0x0, 4, 1}, {27, 'o', 0x685174, 0x6ccd, -1, 0x0, 4, 1},
		{28, 'o', 0x685174, 0x6cd1, -1, 0x0, 4, 1}, {29, 'o', 0x685174, 0x6cd5, -1, 0x0, 4, 1}, {30, 'o', 0x685174, 0x6cd9, -1, 0x0, 4, 1}, {31, 'o', 0x685174, 0x6cdd, -1, 0x0, 4, 1},
		{32, 'o', 0x685174, 0x6ce1, -1, 0x0, 4, 1}, {33, 'o', 0x685174, 0x852f, -1, 0x0, 4, 1}, {34, 'o', 0x685174, 0x8533, -1, 0x0, 4, 1}, {35, 'o', 0x685174, 0x8537, -1, 0x0, 4, 1},
		{36, 'o', 0x685174, 0x853b, -1, 0x0, 4, 1}, {37, 'o', 0x685174, 0x853f, -1, 0x0, 4, 1}, {38, 'o', 0x685174, 0x8543, -1, 0x0, 4, 1}, {39, 'o', 0x685174, 0x9d91, -1, 0x0, 4, 1},
		{40, 'o', 0x685174, 0x9d95, -1, 0x0, 4, 1}, {41, 'o', 0x685174, 0x9d99, -1, 0x0, 4, 1}, {42, 'o', 0x685174, 0x9d9d, -1, 0x0, 4, 1}, {43, 'o', 0x685174, 0x9da1, -1, 0x0, 4, 1},
		{44, 'o', 0x685174, 0x9da5, -1, 0x0, 4, 1}, {45, 'o', 0x685174, 0xb5f3, -1, 0x0, 4, 1}, {46, 'o', 0x685174, 0xb5f7, -1, 0x0, 4, 1}, {47, 'o', 0x685174, 0xb5fb, -1, 0x0, 4, 1},
		{48, 'o', 0x685174, 0xb5ff, -1, 0x0, 4, 1}, {49, 'o', 0x685174, 0xb603, -1, 0x0, 4, 1}, {50, 'o', 0x685174, 0xb607, -1, 0x0, 4, 1}, {51, 'o', 0x685174, 0xb25, 0, 0x1862, 4, 1},
		{52, 'o', 0x685174, 0xb29, 0, 0x1862, 4, 1}, {53, 'o', 0x685174, 0xb2d, 0, 0x1862, 4, 1}, {54, 'o', 0x685174, 0xb31, 0, 0x1862, 4, 1}, {55, 'o', 0x685174, 0xb35, 0, 0x1862, 4, 1},
		{56, 'o', 0x685174, 0xb39, 0, 0x1862, 4, 1}, {57, 'g', 0x68ec78, 0, -1, 0, 4, 1}, {58, 'o', 0x69c0f4, 0xf8, -1, 0x0, 4, 0}, {60, 'g', 0x68edcc, 0, -1, 0, 4, 0},
	}, []ctrsLink{
		{0, 'o', 0x685174, 0xb45, -1, 0x0, 4, 1}, {1, 'o', 0x685174, 0xb49, -1, 0x0, 4, 1}, {2, 'o', 0x685174, 0xb4d, -1, 0x0, 4, 1}, {3, 'o', 0x685174, 0xb51, -1, 0x0, 4, 1},
		{4, 'o', 0x685174, 0xb55, -1, 0x0, 4, 1}, {5, 'o', 0x685174, 0xb59, -1, 0x0, 4, 1}, {6, 'o', 0x685174, 0x23a7, -1, 0x0, 4, 1}, {7, 'o', 0x685174, 0x23ab, -1, 0x0, 4, 1},
		{8, 'o', 0x685174, 0x23af, -1, 0x0, 4, 1}, {9, 'o', 0x685174, 0x23b3, -1, 0x0, 4, 1}, {10, 'o', 0x685174, 0x23b7, -1, 0x0, 4, 1}, {11, 'o', 0x685174, 0x23bb, -1, 0x0, 4, 1},
		{12, 'o', 0x685174, 0x3c09, -1, 0x0, 4, 1}, {13, 'o', 0x685174, 0x3c0d, -1, 0x0, 4, 1}, {14, 'o', 0x685174, 0x3c11, -1, 0x0, 4, 1}, {15, 'o', 0x685174, 0x3c15, -1, 0x0, 4, 1},
		{16, 'o', 0x685174, 0x3c19, -1, 0x0, 4, 1}, {17, 'o', 0x685174, 0x3c1d, -1, 0x0, 4, 1}, {18, 'o', 0x685174, 0x546b, -1, 0x0, 4, 1}, {19, 'o', 0x685174, 0x546f, -1, 0x0, 4, 1},
		{20, 'o', 0x685174, 0x5473, -1, 0x0, 4, 1}, {21, 'o', 0x685174, 0x5477, -1, 0x0, 4, 1}, {22, 'o', 0x685174, 0x547b, -1, 0x0, 4, 1}, {23, 'o', 0x685174, 0x547f, -1, 0x0, 4, 1},
		{24, 'o', 0x685174, 0x6ccd, -1, 0x0, 4, 1}, {25, 'o', 0x685174, 0x6cd1, -1, 0x0, 4, 1}, {26, 'o', 0x685174, 0x6cd5, -1, 0x0, 4, 1}, {27, 'o', 0x685174, 0x6cd9, -1, 0x0, 4, 1},
		{28, 'o', 0x685174, 0x6cdd, -1, 0x0, 4, 1}, {29, 'o', 0x685174, 0x6ce1, -1, 0x0, 4, 1}, {30, 'o', 0x685174, 0x852f, -1, 0x0, 4, 1}, {31, 'o', 0x685174, 0x8533, -1, 0x0, 4, 1},
		{32, 'o', 0x685174, 0x8537, -1, 0x0, 4, 1}, {33, 'o', 0x685174, 0x853b, -1, 0x0, 4, 1}, {34, 'o', 0x685174, 0x853f, -1, 0x0, 4, 1}, {35, 'o', 0x685174, 0x8543, -1, 0x0, 4, 1},
		{36, 'o', 0x685174, 0x9d91, -1, 0x0, 4, 1}, {37, 'o', 0x685174, 0x9d95, -1, 0x0, 4, 1}, {38, 'o', 0x685174, 0x9d99, -1, 0x0, 4, 1}, {39, 'o', 0x685174, 0x9d9d, -1, 0x0, 4, 1},
		{40, 'o', 0x685174, 0x9da1, -1, 0x0, 4, 1}, {41, 'o', 0x685174, 0x9da5, -1, 0x0, 4, 1}, {42, 'o', 0x685174, 0xb5f3, -1, 0x0, 4, 1}, {43, 'o', 0x685174, 0xb5f7, -1, 0x0, 4, 1},
		{44, 'o', 0x685174, 0xb5fb, -1, 0x0, 4, 1}, {45, 'o', 0x685174, 0xb5ff, -1, 0x0, 4, 1}, {46, 'o', 0x685174, 0xb603, -1, 0x0, 4, 1}, {47, 'o', 0x685174, 0xb607, -1, 0x0, 4, 1},
		{48, 'o', 0x69c0f4, 0xf8, -1, 0x0, 4, 0},
	}},
	0x52b6d0: {characterRealmSpDrainNode, 12, 0x20000000, []ctrsLink{
		{0, 'a', 0, 0, -1, 0, 4, 0}, {1, 'a', 1, 0, -1, 0, 4, 0}, {2, 'a', 2, 0, -1, 0, 4, 0}, {3, 'a', 3, 0, -1, 0, 4, 0},
		{4, 'g', 0x687500, 0, -1, 0, 1, 0}, {5, 'o', 0x685174, 0xb11, 0, 0x1862, 4, 0}, {6, 'o', 0x685174, 0xb45, 0, 0x1862, 4, 0}, {7, 'o', 0x685174, 0xb49, 0, 0x1862, 4, 0},
		{8, 'o', 0x685174, 0xb4d, 0, 0x1862, 4, 0}, {9, 'o', 0x685174, 0xb51, 0, 0x1862, 4, 0}, {10, 'o', 0x685174, 0xb55, 0, 0x1862, 4, 0}, {11, 'o', 0x685174, 0xb59, 0, 0x1862, 4, 0},
	}, []ctrsLink{
		{0, 'o', 0x685174, 0xb45, 0, 0x1862, 4, 0}, {1, 'o', 0x685174, 0xb49, 0, 0x1862, 4, 0}, {2, 'o', 0x685174, 0xb4d, 0, 0x1862, 4, 0}, {3, 'o', 0x685174, 0xb51, 0, 0x1862, 4, 0},
		{4, 'o', 0x685174, 0xb55, 0, 0x1862, 4, 0}, {5, 'o', 0x685174, 0xb59, 0, 0x1862, 4, 0},
	}},
	0x5236a0: {characterConditionDecayNode, 25, 0x20000000, []ctrsLink{
		{0, 'a', 0, 0, -1, 0, 4, 0}, {1, 'a', 1, 0, -1, 0, 4, 0}, {2, 'a', 2, 0, -1, 0, 4, 0}, {3, 'o', 0x685178, 0x0, 0, 0x106, 1, 0},
		{4, 'o', 0x685174, 0xa01, 0, 0x1862, 4, 0}, {5, 'o', 0x685174, 0xa05, 0, 0x1862, 4, 0}, {6, 'o', 0x685174, 0xa09, 0, 0x1862, 4, 0}, {7, 'o', 0x685174, 0xa0d, 0, 0x1862, 4, 0},
		{8, 'o', 0x685174, 0xa11, 0, 0x1862, 4, 0}, {9, 'o', 0x685174, 0xa15, 0, 0x1862, 4, 0}, {10, 'o', 0x685174, 0xa19, 0, 0x1862, 4, 0}, {11, 'o', 0x685174, 0xa1d, 0, 0x1862, 4, 0},
		{12, 'o', 0x685174, 0xa21, 0, 0x1862, 4, 0}, {13, 'o', 0x685174, 0xa25, 0, 0x1862, 4, 0}, {14, 'o', 0x685174, 0xa29, 0, 0x1862, 4, 0}, {15, 'o', 0x685174, 0xa2d, 0, 0x1862, 4, 0},
		{16, 'o', 0x685174, 0xa31, 0, 0x1862, 4, 0}, {17, 'o', 0x685174, 0xa35, 0, 0x1862, 4, 0}, {18, 'o', 0x685174, 0xa39, 0, 0x1862, 4, 0}, {19, 'o', 0x685174, 0xa3d, 0, 0x1862, 4, 0},
		{20, 'o', 0x685174, 0xa41, 0, 0x1862, 4, 0}, {21, 'o', 0x685174, 0xa45, 0, 0x1862, 4, 0}, {22, 'o', 0x685174, 0xa49, 0, 0x1862, 4, 0}, {23, 'o', 0x685174, 0xa4d, 0, 0x1862, 4, 0},
		{24, 'o', 0x685174, 0xb09, 0, 0x1862, 4, 0},
	}, []ctrsLink{
		{0, 'o', 0x685174, 0xa01, 0, 0x1862, 4, 0}, {1, 'o', 0x685174, 0xa05, 0, 0x1862, 4, 0}, {2, 'o', 0x685174, 0xa09, 0, 0x1862, 4, 0}, {3, 'o', 0x685174, 0xa0d, 0, 0x1862, 4, 0},
		{4, 'o', 0x685174, 0xa11, 0, 0x1862, 4, 0}, {5, 'o', 0x685174, 0xa15, 0, 0x1862, 4, 0}, {6, 'o', 0x685174, 0xa19, 0, 0x1862, 4, 0}, {7, 'o', 0x685174, 0xa1d, 0, 0x1862, 4, 0},
		{8, 'o', 0x685174, 0xa21, 0, 0x1862, 4, 0}, {9, 'o', 0x685174, 0xa25, 0, 0x1862, 4, 0}, {10, 'o', 0x685174, 0xa29, 0, 0x1862, 4, 0}, {11, 'o', 0x685174, 0xa2d, 0, 0x1862, 4, 0},
		{12, 'o', 0x685174, 0xa31, 0, 0x1862, 4, 0}, {13, 'o', 0x685174, 0xa35, 0, 0x1862, 4, 0}, {14, 'o', 0x685174, 0xa39, 0, 0x1862, 4, 0}, {15, 'o', 0x685174, 0xa3d, 0, 0x1862, 4, 0},
		{16, 'o', 0x685174, 0xa41, 0, 0x1862, 4, 0}, {17, 'o', 0x685174, 0xa45, 0, 0x1862, 4, 0}, {18, 'o', 0x685174, 0xa49, 0, 0x1862, 4, 0}, {19, 'o', 0x685174, 0xa4d, 0, 0x1862, 4, 0},
		{20, 'o', 0x685174, 0xb09, 0, 0x1862, 4, 0},
	}},
	0x523a80: {characterEffectRemoveNode, 355, 0x20000000, []ctrsLink{
		{0, 'a', 0, 0, -1, 0, 4, 0}, {1, 'o', 0x685174, 0x4, 0, 0x1862, 1, 0}, {2, 'o', 0x685178, 0xfe, -1, 0x0, 1, 0}, {3, 'o', 0x685178, 0x204, -1, 0x0, 1, 0},
		{4, 'o', 0x685178, 0x30a, -1, 0x0, 1, 0}, {5, 'o', 0x685178, 0x410, -1, 0x0, 1, 0}, {6, 'o', 0x685178, 0x516, -1, 0x0, 1, 0}, {7, 'o', 0x685178, 0x61c, -1, 0x0, 1, 0},
		{8, 'o', 0x685178, 0x722, -1, 0x0, 1, 0}, {9, 'o', 0x685178, 0x828, -1, 0x0, 1, 0}, {10, 'o', 0x685174, 0x69, 0, 0x1862, 4, 0}, {11, 'o', 0x685174, 0x71, 0, 0x1862, 4, 0},
		{12, 'o', 0x685174, 0x75, 0, 0x1862, 4, 0}, {13, 'o', 0x685174, 0x89, 0, 0x1862, 4, 0}, {14, 'o', 0x685174, 0x8d, 0, 0x1862, 4, 0}, {15, 'o', 0x685174, 0x91, 0, 0x1862, 4, 0},
		{16, 'o', 0x685174, 0x95, 0, 0x1862, 4, 0}, {17, 'o', 0x685174, 0x99, 0, 0x1862, 4, 0}, {18, 'o', 0x685174, 0x9d, 0, 0x1862, 4, 0}, {19, 'o', 0x685174, 0xa1, 0, 0x1862, 4, 0},
		{20, 'o', 0x685174, 0xa5, 0, 0x1862, 4, 0}, {21, 'o', 0x685174, 0xa9, 0, 0x1862, 4, 0}, {22, 'o', 0x685174, 0xad, 0, 0x1862, 4, 0}, {23, 'o', 0x685174, 0xb1, 0, 0x1862, 4, 0},
		{24, 'o', 0x685174, 0xb5, 0, 0x1862, 4, 0}, {25, 'o', 0x685174, 0xb9, 0, 0x1862, 4, 0}, {26, 'o', 0x685174, 0xbd, 0, 0x1862, 4, 0}, {27, 'o', 0x685174, 0xc1, 0, 0x1862, 4, 0},
		{28, 'o', 0x685174, 0xc5, 0, 0x1862, 4, 0}, {29, 'o', 0x685174, 0xe5, 0, 0x1862, 4, 0}, {30, 'o', 0x685174, 0xf9, 0, 0x1862, 4, 0}, {31, 'o', 0x685174, 0x10d, 0, 0x1862, 4, 0},
		{32, 'o', 0x685174, 0x121, 0, 0x1862, 4, 0}, {33, 'o', 0x685174, 0x135, 0, 0x1862, 4, 0}, {34, 'o', 0x685174, 0x149, 0, 0x1862, 4, 0}, {35, 'o', 0x685174, 0x15d, 0, 0x1862, 4, 0},
		{36, 'o', 0x685174, 0x19f, 0, 0x1862, 4, 0}, {37, 'o', 0x685174, 0x1c5, 0, 0x1862, 4, 0}, {38, 'o', 0x685174, 0x1eb, 0, 0x1862, 4, 0}, {39, 'o', 0x685174, 0x211, 0, 0x1862, 4, 0},
		{40, 'o', 0x685174, 0x237, 0, 0x1862, 4, 0}, {41, 'o', 0x685174, 0x25d, 0, 0x1862, 4, 0}, {42, 'o', 0x685174, 0x283, 0, 0x1862, 4, 0}, {43, 'o', 0x685174, 0x2a9, 0, 0x1862, 4, 0},
		{44, 'o', 0x685174, 0x2cf, 0, 0x1862, 4, 0}, {45, 'o', 0x685174, 0x2f5, 0, 0x1862, 4, 0}, {46, 'o', 0x685174, 0x31b, 0, 0x1862, 4, 0}, {47, 'o', 0x685174, 0x341, 0, 0x1862, 4, 0},
		{48, 'o', 0x685174, 0x367, 0, 0x1862, 4, 0}, {49, 'o', 0x685174, 0x38d, 0, 0x1862, 4, 0}, {50, 'o', 0x685174, 0x3b3, 0, 0x1862, 4, 0}, {51, 'o', 0x685174, 0x3d9, 0, 0x1862, 4, 0},
		{52, 'o', 0x685174, 0x3ff, 0, 0x1862, 4, 0}, {53, 'o', 0x685174, 0x425, 0, 0x1862, 4, 0}, {54, 'o', 0x685174, 0x44b, 0, 0x1862, 4, 0}, {55, 'o', 0x685174, 0x471, 0, 0x1862, 4, 0},
		{56, 'o', 0x685174, 0x497, 0, 0x1862, 4, 0}, {57, 'o', 0x685174, 0x4bd, 0, 0x1862, 4, 0}, {58, 'o', 0x685174, 0x4e3, 0, 0x1862, 4, 0}, {59, 'o', 0x685174, 0x509, 0, 0x1862, 4, 0},
		{60, 'o', 0x685174, 0x52f, 0, 0x1862, 4, 0}, {61, 'o', 0x685174, 0x555, 0, 0x1862, 4, 0}, {62, 'o', 0x685174, 0x57b, 0, 0x1862, 4, 0}, {63, 'o', 0x685174, 0x5a1, 0, 0x1862, 4, 0},
		{64, 'o', 0x685174, 0x5c7, 0, 0x1862, 4, 0}, {65, 'o', 0x685174, 0x5ed, 0, 0x1862, 4, 0}, {66, 'o', 0x685174, 0x613, 0, 0x1862, 4, 0}, {67, 'o', 0x685174, 0x639, 0, 0x1862, 4, 0},
		{68, 'o', 0x685174, 0x65f, 0, 0x1862, 4, 0}, {69, 'o', 0x685174, 0x685, 0, 0x1862, 4, 0}, {70, 'o', 0x685174, 0x6ab, 0, 0x1862, 4, 0}, {71, 'o', 0x685174, 0x6d1, 0, 0x1862, 4, 0},
		{72, 'o', 0x685174, 0x6f7, 0, 0x1862, 4, 0}, {73, 'o', 0x685174, 0x71d, 0, 0x1862, 4, 0}, {74, 'o', 0x685174, 0x743, 0, 0x1862, 4, 0}, {75, 'o', 0x685174, 0x769, 0, 0x1862, 4, 0},
		{76, 'o', 0x685174, 0x78f, 0, 0x1862, 4, 0}, {77, 'o', 0x685174, 0xa01, 0, 0x1862, 4, 0}, {78, 'o', 0x685174, 0xa05, 0, 0x1862, 4, 0}, {79, 'o', 0x685174, 0xa09, 0, 0x1862, 4, 0},
		{80, 'o', 0x685174, 0xa0d, 0, 0x1862, 4, 0}, {81, 'o', 0x685174, 0xa11, 0, 0x1862, 4, 0}, {82, 'o', 0x685174, 0xa15, 0, 0x1862, 4, 0}, {83, 'o', 0x685174, 0xa19, 0, 0x1862, 4, 0},
		{84, 'o', 0x685174, 0xa1d, 0, 0x1862, 4, 0}, {85, 'o', 0x685174, 0xa21, 0, 0x1862, 4, 0}, {86, 'o', 0x685174, 0xa25, 0, 0x1862, 4, 0}, {87, 'o', 0x685174, 0xa29, 0, 0x1862, 4, 0},
		{88, 'o', 0x685174, 0xa2d, 0, 0x1862, 4, 0}, {89, 'o', 0x685174, 0xa31, 0, 0x1862, 4, 0}, {90, 'o', 0x685174, 0xa35, 0, 0x1862, 4, 0}, {91, 'o', 0x685174, 0xa39, 0, 0x1862, 4, 0},
		{92, 'o', 0x685174, 0xa3d, 0, 0x1862, 4, 0}, {93, 'o', 0x685174, 0xa41, 0, 0x1862, 4, 0}, {94, 'o', 0x685174, 0xa45, 0, 0x1862, 4, 0}, {95, 'o', 0x685174, 0xa49, 0, 0x1862, 4, 0},
		{96, 'o', 0x685174, 0xa4d, 0, 0x1862, 4, 0}, {97, 'o', 0x685174, 0xb09, 0, 0x1862, 4, 0}, {98, 'o', 0x685174, 0xa65, 0, 0x1862, 4, 0}, {99, 'o', 0x685174, 0xa69, 0, 0x1862, 4, 0},
		{100, 'o', 0x685174, 0xa6d, 0, 0x1862, 4, 0}, {101, 'o', 0x685174, 0xa71, 0, 0x1862, 4, 0}, {102, 'o', 0x685174, 0xa75, 0, 0x1862, 4, 0}, {103, 'o', 0x685174, 0xa79, 0, 0x1862, 4, 0},
		{104, 'o', 0x685174, 0xa7d, 0, 0x1862, 4, 0}, {105, 'o', 0x685174, 0xa81, 0, 0x1862, 4, 0}, {106, 'o', 0x685174, 0xa85, 0, 0x1862, 4, 0}, {107, 'o', 0x685174, 0xa89, 0, 0x1862, 4, 0},
		{108, 'o', 0x685174, 0xa8d, 0, 0x1862, 4, 0}, {109, 'o', 0x685174, 0xa91, 0, 0x1862, 4, 0}, {110, 'o', 0x685174, 0xa95, 0, 0x1862, 4, 0}, {111, 'o', 0x685174, 0xa99, 0, 0x1862, 4, 0},
		{112, 'o', 0x685174, 0xa9d, 0, 0x1862, 4, 0}, {113, 'o', 0x685174, 0xaa1, 0, 0x1862, 4, 0}, {114, 'o', 0x685174, 0xaa5, 0, 0x1862, 4, 0}, {115, 'o', 0x685174, 0xaa9, 0, 0x1862, 4, 0},
		{116, 'o', 0x685174, 0xaad, 0, 0x1862, 4, 0}, {117, 'o', 0x685174, 0xab1, 0, 0x1862, 4, 0}, {118, 'o', 0x685174, 0xab5, 0, 0x1862, 4, 0}, {119, 'o', 0x685174, 0xab9, 0, 0x1862, 4, 0},
		{120, 'o', 0x685174, 0xabd, 0, 0x1862, 4, 0}, {121, 'o', 0x685174, 0xac1, 0, 0x1862, 4, 0}, {122, 'o', 0x685174, 0x1830, 0, 0x1862, 1, 0}, {123, 'o', 0x685174, 0x1828, 0, 0x1862, 4, 0},
		{124, 'g', 0x686a70, 0, -1, 0, 4, 0}, {126, 'o', 0x685174, 0x1709, 0, 0x1862, 1, 0}, {127, 'o', 0x685174, 0x170a, 0, 0x1862, 1, 0}, {128, 'o', 0x685174, 0x170b, 0, 0x1862, 1, 0},
		{129, 'o', 0x685174, 0x170c, 0, 0x1862, 1, 0}, {130, 'o', 0x685174, 0x170d, 0, 0x1862, 1, 0}, {131, 'o', 0x685174, 0x170e, 0, 0x1862, 1, 0}, {132, 'o', 0x685174, 0x170f, 0, 0x1862, 1, 0},
		{133, 'o', 0x685174, 0x1710, 0, 0x1862, 1, 0}, {134, 'o', 0x685174, 0x1711, 0, 0x1862, 1, 0}, {135, 'o', 0x685174, 0x1712, 0, 0x1862, 1, 0}, {136, 'o', 0x685174, 0x1713, 0, 0x1862, 1, 0},
		{137, 'o', 0x685174, 0x1714, 0, 0x1862, 1, 0}, {138, 'o', 0x685174, 0x1715, 0, 0x1862, 1, 0}, {139, 'o', 0x685174, 0x1716, 0, 0x1862, 1, 0}, {140, 'o', 0x685174, 0x1717, 0, 0x1862, 1, 0},
		{141, 'o', 0x685174, 0x1718, 0, 0x1862, 1, 0}, {142, 'o', 0x685174, 0x1719, 0, 0x1862, 1, 0}, {143, 'o', 0x685174, 0x171a, 0, 0x1862, 1, 0}, {144, 'o', 0x685174, 0x171b, 0, 0x1862, 1, 0},
		{145, 'o', 0x685174, 0x171c, 0, 0x1862, 1, 0}, {146, 'o', 0x685174, 0x171d, 0, 0x1862, 1, 0}, {147, 'o', 0x685174, 0x171e, 0, 0x1862, 1, 0}, {148, 'o', 0x685174, 0x171f, 0, 0x1862, 1, 0},
		{149, 'o', 0x685174, 0x1720, 0, 0x1862, 1, 0}, {150, 'o', 0x685174, 0x1721, 0, 0x1862, 1, 0}, {151, 'o', 0x685174, 0x1722, 0, 0x1862, 1, 0}, {152, 'o', 0x685174, 0x1723, 0, 0x1862, 1, 0},
		{153, 'o', 0x685174, 0x1724, 0, 0x1862, 1, 0}, {154, 'o', 0x685174, 0x1725, 0, 0x1862, 1, 0}, {155, 'o', 0x685174, 0x1726, 0, 0x1862, 1, 0}, {156, 'o', 0x685174, 0x1727, 0, 0x1862, 1, 0},
		{157, 'o', 0x685174, 0x1728, 0, 0x1862, 1, 0}, {158, 'o', 0x685174, 0x1729, 0, 0x1862, 1, 0}, {159, 'o', 0x685174, 0x172a, 0, 0x1862, 1, 0}, {160, 'o', 0x685174, 0x172b, 0, 0x1862, 1, 0},
		{161, 'o', 0x685174, 0x172c, 0, 0x1862, 1, 0}, {162, 'o', 0x685174, 0x172d, 0, 0x1862, 1, 0}, {163, 'o', 0x685174, 0x172e, 0, 0x1862, 1, 0}, {164, 'o', 0x685174, 0x172f, 0, 0x1862, 1, 0},
		{165, 'o', 0x685174, 0x1730, 0, 0x1862, 1, 0}, {166, 'o', 0x685174, 0x1731, 0, 0x1862, 1, 0}, {167, 'o', 0x685174, 0x1732, 0, 0x1862, 1, 0}, {168, 'o', 0x685174, 0x1733, 0, 0x1862, 1, 0},
		{169, 'o', 0x685174, 0x1734, 0, 0x1862, 1, 0}, {170, 'o', 0x685174, 0x1735, 0, 0x1862, 1, 0}, {171, 'o', 0x685174, 0x1736, 0, 0x1862, 1, 0}, {172, 'o', 0x685174, 0x1737, 0, 0x1862, 1, 0},
		{173, 'o', 0x685174, 0x1738, 0, 0x1862, 1, 0}, {174, 'o', 0x685174, 0x1739, 0, 0x1862, 1, 0}, {175, 'o', 0x685174, 0x173a, 0, 0x1862, 1, 0}, {176, 'o', 0x685174, 0x173b, 0, 0x1862, 1, 0},
		{177, 'o', 0x685174, 0x173c, 0, 0x1862, 1, 0}, {178, 'o', 0x685174, 0x173d, 0, 0x1862, 1, 0}, {179, 'o', 0x685174, 0x173e, 0, 0x1862, 1, 0}, {180, 'o', 0x685174, 0x173f, 0, 0x1862, 1, 0},
		{181, 'o', 0x685174, 0x1740, 0, 0x1862, 1, 0}, {182, 'o', 0x685174, 0x1741, 0, 0x1862, 1, 0}, {183, 'o', 0x685174, 0x1742, 0, 0x1862, 1, 0}, {184, 'o', 0x685174, 0x1743, 0, 0x1862, 1, 0},
		{185, 'o', 0x685174, 0x1744, 0, 0x1862, 1, 0}, {186, 'o', 0x685174, 0x1745, 0, 0x1862, 1, 0}, {187, 'o', 0x685174, 0x1746, 0, 0x1862, 1, 0}, {188, 'o', 0x685174, 0x1747, 0, 0x1862, 1, 0},
		{189, 'o', 0x685174, 0x1748, 0, 0x1862, 1, 0}, {190, 'o', 0x685174, 0x1749, 0, 0x1862, 1, 0}, {191, 'o', 0x685174, 0x174a, 0, 0x1862, 1, 0}, {192, 'o', 0x685174, 0x174b, 0, 0x1862, 1, 0},
		{193, 'o', 0x685174, 0x174c, 0, 0x1862, 1, 0}, {194, 'o', 0x685174, 0x174d, 0, 0x1862, 1, 0}, {195, 'o', 0x685174, 0x174e, 0, 0x1862, 1, 0}, {196, 'o', 0x685174, 0x174f, 0, 0x1862, 1, 0},
		{197, 'o', 0x685174, 0x1750, 0, 0x1862, 1, 0}, {198, 'o', 0x685174, 0x1751, 0, 0x1862, 1, 0}, {199, 'o', 0x685174, 0x1752, 0, 0x1862, 1, 0}, {200, 'o', 0x685174, 0x1753, 0, 0x1862, 1, 0},
		{201, 'o', 0x685174, 0x1754, 0, 0x1862, 1, 0}, {202, 'o', 0x685174, 0x1755, 0, 0x1862, 1, 0}, {203, 'o', 0x685174, 0x1756, 0, 0x1862, 1, 0}, {204, 'o', 0x685174, 0x1757, 0, 0x1862, 1, 0},
		{205, 'o', 0x685174, 0x1758, 0, 0x1862, 1, 0}, {206, 'o', 0x685174, 0x1759, 0, 0x1862, 1, 0}, {207, 'o', 0x685174, 0x175a, 0, 0x1862, 1, 0}, {208, 'o', 0x685174, 0x175b, 0, 0x1862, 1, 0},
		{209, 'o', 0x685174, 0x175c, 0, 0x1862, 1, 0}, {210, 'o', 0x685174, 0x175d, 0, 0x1862, 1, 0}, {211, 'o', 0x685174, 0x175e, 0, 0x1862, 1, 0}, {212, 'o', 0x685174, 0x175f, 0, 0x1862, 1, 0},
		{213, 'o', 0x685174, 0x1760, 0, 0x1862, 1, 0}, {214, 'o', 0x685174, 0x1761, 0, 0x1862, 1, 0}, {215, 'o', 0x685174, 0x1762, 0, 0x1862, 1, 0}, {216, 'o', 0x685174, 0x1763, 0, 0x1862, 1, 0},
		{217, 'o', 0x685174, 0x1764, 0, 0x1862, 1, 0}, {218, 'o', 0x685174, 0x1765, 0, 0x1862, 1, 0}, {219, 'o', 0x685174, 0x1766, 0, 0x1862, 1, 0}, {220, 'o', 0x685174, 0x1767, 0, 0x1862, 1, 0},
		{221, 'o', 0x685174, 0x1768, 0, 0x1862, 1, 0}, {222, 'o', 0x685174, 0x1769, 0, 0x1862, 1, 0}, {223, 'o', 0x685174, 0x176a, 0, 0x1862, 1, 0}, {224, 'o', 0x685174, 0x176b, 0, 0x1862, 1, 0},
		{225, 'o', 0x685174, 0x176c, 0, 0x1862, 1, 0}, {226, 'o', 0x685174, 0x176d, 0, 0x1862, 1, 0}, {227, 'o', 0x685174, 0x176e, 0, 0x1862, 1, 0}, {228, 'o', 0x685174, 0x176f, 0, 0x1862, 1, 0},
		{229, 'g', 0x687453, 0, -1, 0, 1, 0}, {230, 'g', 0x687454, 0, -1, 0, 1, 0}, {231, 'g', 0x687455, 0, -1, 0, 1, 0}, {232, 'g', 0x687456, 0, -1, 0, 1, 0},
		{233, 'g', 0x687457, 0, -1, 0, 1, 0}, {234, 'g', 0x687458, 0, -1, 0, 1, 0}, {235, 'g', 0x687459, 0, -1, 0, 1, 0}, {236, 'g', 0x68745a, 0, -1, 0, 1, 0},
		{237, 'g', 0x68745b, 0, -1, 0, 1, 0}, {238, 'g', 0x68745c, 0, -1, 0, 1, 0}, {239, 'g', 0x68745d, 0, -1, 0, 1, 0}, {240, 'g', 0x68745e, 0, -1, 0, 1, 0},
		{241, 'g', 0x68745f, 0, -1, 0, 1, 0}, {242, 'g', 0x687460, 0, -1, 0, 1, 0}, {243, 'g', 0x687461, 0, -1, 0, 1, 0}, {244, 'g', 0x687462, 0, -1, 0, 1, 0},
		{245, 'g', 0x687463, 0, -1, 0, 1, 0}, {246, 'g', 0x687464, 0, -1, 0, 1, 0}, {247, 'g', 0x687465, 0, -1, 0, 1, 0}, {248, 'g', 0x687466, 0, -1, 0, 1, 0},
		{249, 'g', 0x687467, 0, -1, 0, 1, 0}, {250, 'g', 0x687468, 0, -1, 0, 1, 0}, {251, 'g', 0x687469, 0, -1, 0, 1, 0}, {252, 'g', 0x68746a, 0, -1, 0, 1, 0},
		{253, 'g', 0x68746b, 0, -1, 0, 1, 0}, {254, 'g', 0x68746c, 0, -1, 0, 1, 0}, {255, 'g', 0x68746d, 0, -1, 0, 1, 0}, {256, 'g', 0x68746e, 0, -1, 0, 1, 0},
		{257, 'g', 0x68746f, 0, -1, 0, 1, 0}, {258, 'g', 0x687470, 0, -1, 0, 1, 0}, {259, 'g', 0x687471, 0, -1, 0, 1, 0}, {260, 'g', 0x687472, 0, -1, 0, 1, 0},
		{261, 'g', 0x687473, 0, -1, 0, 1, 0}, {262, 'g', 0x687474, 0, -1, 0, 1, 0}, {263, 'g', 0x687475, 0, -1, 0, 1, 0}, {264, 'g', 0x687476, 0, -1, 0, 1, 0},
		{265, 'g', 0x687477, 0, -1, 0, 1, 0}, {266, 'g', 0x687478, 0, -1, 0, 1, 0}, {267, 'g', 0x687479, 0, -1, 0, 1, 0}, {268, 'g', 0x68747a, 0, -1, 0, 1, 0},
		{269, 'g', 0x68747b, 0, -1, 0, 1, 0}, {270, 'g', 0x68747c, 0, -1, 0, 1, 0}, {271, 'g', 0x68747d, 0, -1, 0, 1, 0}, {272, 'g', 0x68747e, 0, -1, 0, 1, 0},
		{273, 'g', 0x68747f, 0, -1, 0, 1, 0}, {274, 'g', 0x687480, 0, -1, 0, 1, 0}, {275, 'g', 0x687481, 0, -1, 0, 1, 0}, {276, 'g', 0x687482, 0, -1, 0, 1, 0},
		{277, 'g', 0x687483, 0, -1, 0, 1, 0}, {278, 'g', 0x687484, 0, -1, 0, 1, 0}, {279, 'g', 0x687485, 0, -1, 0, 1, 0}, {280, 'g', 0x687486, 0, -1, 0, 1, 0},
		{281, 'g', 0x687487, 0, -1, 0, 1, 0}, {282, 'g', 0x687488, 0, -1, 0, 1, 0}, {283, 'g', 0x687489, 0, -1, 0, 1, 0}, {284, 'g', 0x68748a, 0, -1, 0, 1, 0},
		{285, 'g', 0x68748b, 0, -1, 0, 1, 0}, {286, 'g', 0x68748c, 0, -1, 0, 1, 0}, {287, 'g', 0x68748d, 0, -1, 0, 1, 0}, {288, 'g', 0x68748e, 0, -1, 0, 1, 0},
		{289, 'g', 0x68748f, 0, -1, 0, 1, 0}, {290, 'g', 0x687490, 0, -1, 0, 1, 0}, {291, 'g', 0x687491, 0, -1, 0, 1, 0}, {292, 'g', 0x687492, 0, -1, 0, 1, 0},
		{293, 'g', 0x687493, 0, -1, 0, 1, 0}, {294, 'g', 0x687494, 0, -1, 0, 1, 0}, {295, 'g', 0x687495, 0, -1, 0, 1, 0}, {296, 'g', 0x687496, 0, -1, 0, 1, 0},
		{297, 'g', 0x687497, 0, -1, 0, 1, 0}, {298, 'g', 0x687498, 0, -1, 0, 1, 0}, {299, 'g', 0x687499, 0, -1, 0, 1, 0}, {300, 'g', 0x68749a, 0, -1, 0, 1, 0},
		{301, 'g', 0x68749b, 0, -1, 0, 1, 0}, {302, 'g', 0x68749c, 0, -1, 0, 1, 0}, {303, 'g', 0x68749d, 0, -1, 0, 1, 0}, {304, 'g', 0x68749e, 0, -1, 0, 1, 0},
		{305, 'g', 0x68749f, 0, -1, 0, 1, 0}, {306, 'g', 0x6874a0, 0, -1, 0, 1, 0}, {307, 'g', 0x6874a1, 0, -1, 0, 1, 0}, {308, 'g', 0x6874a2, 0, -1, 0, 1, 0},
		{309, 'g', 0x6874a3, 0, -1, 0, 1, 0}, {310, 'g', 0x6874a4, 0, -1, 0, 1, 0}, {311, 'g', 0x6874a5, 0, -1, 0, 1, 0}, {312, 'g', 0x6874a6, 0, -1, 0, 1, 0},
		{313, 'g', 0x6874a7, 0, -1, 0, 1, 0}, {314, 'g', 0x6874a8, 0, -1, 0, 1, 0}, {315, 'g', 0x6874a9, 0, -1, 0, 1, 0}, {316, 'g', 0x6874aa, 0, -1, 0, 1, 0},
		{317, 'g', 0x6874ab, 0, -1, 0, 1, 0}, {318, 'g', 0x6874ac, 0, -1, 0, 1, 0}, {319, 'g', 0x6874ad, 0, -1, 0, 1, 0}, {320, 'g', 0x6874ae, 0, -1, 0, 1, 0},
		{321, 'g', 0x6874af, 0, -1, 0, 1, 0}, {322, 'g', 0x6874b0, 0, -1, 0, 1, 0}, {323, 'g', 0x6874b1, 0, -1, 0, 1, 0}, {324, 'g', 0x6874b2, 0, -1, 0, 1, 0},
		{325, 'g', 0x6874b3, 0, -1, 0, 1, 0}, {326, 'g', 0x6874b4, 0, -1, 0, 1, 0}, {327, 'g', 0x6874b5, 0, -1, 0, 1, 0}, {328, 'g', 0x6874b6, 0, -1, 0, 1, 0},
		{329, 'g', 0x6874b7, 0, -1, 0, 1, 0}, {330, 'g', 0x6874b8, 0, -1, 0, 1, 0}, {331, 'g', 0x6874b9, 0, -1, 0, 1, 0}, {333, 'o', 0x685174, 0x102d, 0, 0x1862, 1, 0},
		{335, 'o', 0x685174, 0x1039, 0, 0x1862, 1, 0}, {337, 'o', 0x685174, 0x1045, 0, 0x1862, 1, 0}, {339, 'o', 0x685174, 0x1051, 0, 0x1862, 1, 0}, {341, 'o', 0x685174, 0x105d, 0, 0x1862, 1, 0},
		{343, 'o', 0x685174, 0x1069, 0, 0x1862, 1, 0}, {345, 'o', 0x685174, 0x1075, 0, 0x1862, 1, 0}, {347, 'o', 0x685174, 0x1081, 0, 0x1862, 1, 0}, {348, 'a', 1, 0, -1, 0, 4, 0},
		{349, 'g', 0x68ec78, 0, -1, 0, 4, 0}, {350, 'g', 0x68edcc, 0, -1, 0, 4, 0}, {351, 'o', 0x69c0f4, 0xf4, -1, 0x0, 4, 0}, {352, 'o', 0x69c0f4, 0xf8, -1, 0x0, 4, 0},
		{353, 'g', 0x6840bb, 0, -1, 0, 1, 0}, {354, 'o', 0x685174, 0xb05, 0, 0x1862, 4, 0},
	}, []ctrsLink{
		{0, 'o', 0x685174, 0x16a2, 0, 0x1862, 1, 0}, {1, 'o', 0x685174, 0x16a3, 0, 0x1862, 1, 0}, {2, 'o', 0x685174, 0x16a4, 0, 0x1862, 1, 0}, {3, 'o', 0x685174, 0x16a5, 0, 0x1862, 1, 0},
		{4, 'o', 0x685174, 0x16a6, 0, 0x1862, 1, 0}, {5, 'o', 0x685174, 0x16a7, 0, 0x1862, 1, 0}, {6, 'o', 0x685174, 0x16a8, 0, 0x1862, 1, 0}, {7, 'o', 0x685174, 0x16a9, 0, 0x1862, 1, 0},
		{8, 'o', 0x685174, 0x16aa, 0, 0x1862, 1, 0}, {9, 'o', 0x685174, 0x16ab, 0, 0x1862, 1, 0}, {10, 'o', 0x685174, 0x16ac, 0, 0x1862, 1, 0}, {11, 'o', 0x685174, 0x16ad, 0, 0x1862, 1, 0},
		{12, 'o', 0x685174, 0x16ae, 0, 0x1862, 1, 0}, {13, 'o', 0x685174, 0x16af, 0, 0x1862, 1, 0}, {14, 'o', 0x685174, 0x16b0, 0, 0x1862, 1, 0}, {15, 'o', 0x685174, 0x16b1, 0, 0x1862, 1, 0},
		{16, 'o', 0x685174, 0x16b2, 0, 0x1862, 1, 0}, {17, 'o', 0x685174, 0x16b3, 0, 0x1862, 1, 0}, {18, 'o', 0x685174, 0x16b4, 0, 0x1862, 1, 0}, {19, 'o', 0x685174, 0x16b5, 0, 0x1862, 1, 0},
		{20, 'o', 0x685174, 0x16b6, 0, 0x1862, 1, 0}, {21, 'o', 0x685174, 0x16b7, 0, 0x1862, 1, 0}, {22, 'o', 0x685174, 0x16b8, 0, 0x1862, 1, 0}, {23, 'o', 0x685174, 0x16b9, 0, 0x1862, 1, 0},
		{24, 'o', 0x685174, 0x16ba, 0, 0x1862, 1, 0}, {25, 'o', 0x685174, 0x16bb, 0, 0x1862, 1, 0}, {26, 'o', 0x685174, 0x16bc, 0, 0x1862, 1, 0}, {27, 'o', 0x685174, 0x16bd, 0, 0x1862, 1, 0},
		{28, 'o', 0x685174, 0x16be, 0, 0x1862, 1, 0}, {29, 'o', 0x685174, 0x16bf, 0, 0x1862, 1, 0}, {30, 'o', 0x685174, 0x16c0, 0, 0x1862, 1, 0}, {31, 'o', 0x685174, 0x16c1, 0, 0x1862, 1, 0},
		{32, 'o', 0x685174, 0x16c2, 0, 0x1862, 1, 0}, {33, 'o', 0x685174, 0x16c3, 0, 0x1862, 1, 0}, {34, 'o', 0x685174, 0x16c4, 0, 0x1862, 1, 0}, {35, 'o', 0x685174, 0x16c5, 0, 0x1862, 1, 0},
		{36, 'o', 0x685174, 0x16c6, 0, 0x1862, 1, 0}, {37, 'o', 0x685174, 0x16c7, 0, 0x1862, 1, 0}, {38, 'o', 0x685174, 0x16c8, 0, 0x1862, 1, 0}, {39, 'o', 0x685174, 0x16c9, 0, 0x1862, 1, 0},
		{40, 'o', 0x685174, 0x16ca, 0, 0x1862, 1, 0}, {41, 'o', 0x685174, 0x16cb, 0, 0x1862, 1, 0}, {42, 'o', 0x685174, 0x16cc, 0, 0x1862, 1, 0}, {43, 'o', 0x685174, 0x16cd, 0, 0x1862, 1, 0},
		{44, 'o', 0x685174, 0x16ce, 0, 0x1862, 1, 0}, {45, 'o', 0x685174, 0x16cf, 0, 0x1862, 1, 0}, {46, 'o', 0x685174, 0x16d0, 0, 0x1862, 1, 0}, {47, 'o', 0x685174, 0x16d1, 0, 0x1862, 1, 0},
		{48, 'o', 0x685174, 0x16d2, 0, 0x1862, 1, 0}, {49, 'o', 0x685174, 0x16d3, 0, 0x1862, 1, 0}, {50, 'o', 0x685174, 0x16d4, 0, 0x1862, 1, 0}, {51, 'o', 0x685174, 0x16d5, 0, 0x1862, 1, 0},
		{52, 'o', 0x685174, 0x16d6, 0, 0x1862, 1, 0}, {53, 'o', 0x685174, 0x16d7, 0, 0x1862, 1, 0}, {54, 'o', 0x685174, 0x16d8, 0, 0x1862, 1, 0}, {55, 'o', 0x685174, 0x16d9, 0, 0x1862, 1, 0},
		{56, 'o', 0x685174, 0x16da, 0, 0x1862, 1, 0}, {57, 'o', 0x685174, 0x16db, 0, 0x1862, 1, 0}, {58, 'o', 0x685174, 0x16dc, 0, 0x1862, 1, 0}, {59, 'o', 0x685174, 0x16dd, 0, 0x1862, 1, 0},
		{60, 'o', 0x685174, 0x16de, 0, 0x1862, 1, 0}, {61, 'o', 0x685174, 0x16df, 0, 0x1862, 1, 0}, {62, 'o', 0x685174, 0x16e0, 0, 0x1862, 1, 0}, {63, 'o', 0x685174, 0x16e1, 0, 0x1862, 1, 0},
		{64, 'o', 0x685174, 0x16e2, 0, 0x1862, 1, 0}, {65, 'o', 0x685174, 0x16e3, 0, 0x1862, 1, 0}, {66, 'o', 0x685174, 0x16e4, 0, 0x1862, 1, 0}, {67, 'o', 0x685174, 0x16e5, 0, 0x1862, 1, 0},
		{68, 'o', 0x685174, 0x16e6, 0, 0x1862, 1, 0}, {69, 'o', 0x685174, 0x16e7, 0, 0x1862, 1, 0}, {70, 'o', 0x685174, 0x16e8, 0, 0x1862, 1, 0}, {71, 'o', 0x685174, 0x16e9, 0, 0x1862, 1, 0},
		{72, 'o', 0x685174, 0x16ea, 0, 0x1862, 1, 0}, {73, 'o', 0x685174, 0x16eb, 0, 0x1862, 1, 0}, {74, 'o', 0x685174, 0x16ec, 0, 0x1862, 1, 0}, {75, 'o', 0x685174, 0x16ed, 0, 0x1862, 1, 0},
		{76, 'o', 0x685174, 0x16ee, 0, 0x1862, 1, 0}, {77, 'o', 0x685174, 0x16ef, 0, 0x1862, 1, 0}, {78, 'o', 0x685174, 0x16f0, 0, 0x1862, 1, 0}, {79, 'o', 0x685174, 0x16f1, 0, 0x1862, 1, 0},
		{80, 'o', 0x685174, 0x16f2, 0, 0x1862, 1, 0}, {81, 'o', 0x685174, 0x16f3, 0, 0x1862, 1, 0}, {82, 'o', 0x685174, 0x16f4, 0, 0x1862, 1, 0}, {83, 'o', 0x685174, 0x16f5, 0, 0x1862, 1, 0},
		{84, 'o', 0x685174, 0x16f6, 0, 0x1862, 1, 0}, {85, 'o', 0x685174, 0x16f7, 0, 0x1862, 1, 0}, {86, 'o', 0x685174, 0x16f8, 0, 0x1862, 1, 0}, {87, 'o', 0x685174, 0x16f9, 0, 0x1862, 1, 0},
		{88, 'o', 0x685174, 0x16fa, 0, 0x1862, 1, 0}, {89, 'o', 0x685174, 0x16fb, 0, 0x1862, 1, 0}, {90, 'o', 0x685174, 0x16fc, 0, 0x1862, 1, 0}, {91, 'o', 0x685174, 0x16fd, 0, 0x1862, 1, 0},
		{92, 'o', 0x685174, 0x16fe, 0, 0x1862, 1, 0}, {93, 'o', 0x685174, 0x16ff, 0, 0x1862, 1, 0}, {94, 'o', 0x685174, 0x1700, 0, 0x1862, 1, 0}, {95, 'o', 0x685174, 0x1701, 0, 0x1862, 1, 0},
		{96, 'o', 0x685174, 0x1702, 0, 0x1862, 1, 0}, {97, 'o', 0x685174, 0x1703, 0, 0x1862, 1, 0}, {98, 'o', 0x685174, 0x1704, 0, 0x1862, 1, 0}, {99, 'o', 0x685174, 0x1705, 0, 0x1862, 1, 0},
		{100, 'o', 0x685174, 0x1706, 0, 0x1862, 1, 0}, {101, 'o', 0x685174, 0x1707, 0, 0x1862, 1, 0}, {102, 'o', 0x685174, 0x1708, 0, 0x1862, 1, 0}, {103, 'o', 0x685174, 0x1770, 0, 0x1862, 1, 0},
		{104, 'o', 0x685174, 0x1771, 0, 0x1862, 1, 0}, {105, 'o', 0x685174, 0x1772, 0, 0x1862, 1, 0}, {106, 'o', 0x685174, 0x1773, 0, 0x1862, 1, 0}, {107, 'o', 0x685174, 0x1774, 0, 0x1862, 1, 0},
		{108, 'o', 0x685174, 0x1775, 0, 0x1862, 1, 0}, {109, 'o', 0x685174, 0x1776, 0, 0x1862, 1, 0}, {110, 'o', 0x685174, 0x1777, 0, 0x1862, 1, 0}, {111, 'o', 0x685174, 0x1778, 0, 0x1862, 1, 0},
		{112, 'o', 0x685174, 0x1779, 0, 0x1862, 1, 0}, {113, 'o', 0x685174, 0x177a, 0, 0x1862, 1, 0}, {114, 'o', 0x685174, 0x177b, 0, 0x1862, 1, 0}, {115, 'o', 0x685174, 0x177c, 0, 0x1862, 1, 0},
		{116, 'o', 0x685174, 0x177d, 0, 0x1862, 1, 0}, {117, 'o', 0x685174, 0x177e, 0, 0x1862, 1, 0}, {118, 'o', 0x685174, 0x177f, 0, 0x1862, 1, 0}, {119, 'o', 0x685174, 0x1780, 0, 0x1862, 1, 0},
		{120, 'o', 0x685174, 0x1781, 0, 0x1862, 1, 0}, {121, 'o', 0x685174, 0x1782, 0, 0x1862, 1, 0}, {122, 'o', 0x685174, 0x1783, 0, 0x1862, 1, 0}, {123, 'o', 0x685174, 0x1784, 0, 0x1862, 1, 0},
		{124, 'o', 0x685174, 0x1785, 0, 0x1862, 1, 0}, {125, 'o', 0x685174, 0x1786, 0, 0x1862, 1, 0}, {126, 'o', 0x685174, 0x1787, 0, 0x1862, 1, 0}, {127, 'o', 0x685174, 0x1788, 0, 0x1862, 1, 0},
		{128, 'o', 0x685174, 0x1789, 0, 0x1862, 1, 0}, {129, 'o', 0x685174, 0x178a, 0, 0x1862, 1, 0}, {130, 'o', 0x685174, 0x178b, 0, 0x1862, 1, 0}, {131, 'o', 0x685174, 0x178c, 0, 0x1862, 1, 0},
		{132, 'o', 0x685174, 0x178d, 0, 0x1862, 1, 0}, {133, 'o', 0x685174, 0x178e, 0, 0x1862, 1, 0}, {134, 'o', 0x685174, 0x178f, 0, 0x1862, 1, 0}, {135, 'o', 0x685174, 0x1790, 0, 0x1862, 1, 0},
		{136, 'o', 0x685174, 0x1791, 0, 0x1862, 1, 0}, {137, 'o', 0x685174, 0x1792, 0, 0x1862, 1, 0}, {138, 'o', 0x685174, 0x1793, 0, 0x1862, 1, 0}, {139, 'o', 0x685174, 0x1794, 0, 0x1862, 1, 0},
		{140, 'o', 0x685174, 0x1795, 0, 0x1862, 1, 0}, {141, 'o', 0x685174, 0x1796, 0, 0x1862, 1, 0}, {142, 'o', 0x685174, 0x1797, 0, 0x1862, 1, 0}, {143, 'o', 0x685174, 0x1798, 0, 0x1862, 1, 0},
		{144, 'o', 0x685174, 0x1799, 0, 0x1862, 1, 0}, {145, 'o', 0x685174, 0x179a, 0, 0x1862, 1, 0}, {146, 'o', 0x685174, 0x179b, 0, 0x1862, 1, 0}, {147, 'o', 0x685174, 0x179c, 0, 0x1862, 1, 0},
		{148, 'o', 0x685174, 0x179d, 0, 0x1862, 1, 0}, {149, 'o', 0x685174, 0x179e, 0, 0x1862, 1, 0}, {150, 'o', 0x685174, 0x179f, 0, 0x1862, 1, 0}, {151, 'o', 0x685174, 0x17a0, 0, 0x1862, 1, 0},
		{152, 'o', 0x685174, 0x17a1, 0, 0x1862, 1, 0}, {153, 'o', 0x685174, 0x17a2, 0, 0x1862, 1, 0}, {154, 'o', 0x685174, 0x17a3, 0, 0x1862, 1, 0}, {155, 'o', 0x685174, 0x17a4, 0, 0x1862, 1, 0},
		{156, 'o', 0x685174, 0x17a5, 0, 0x1862, 1, 0}, {157, 'o', 0x685174, 0x17a6, 0, 0x1862, 1, 0}, {158, 'o', 0x685174, 0x17a7, 0, 0x1862, 1, 0}, {159, 'o', 0x685174, 0x17a8, 0, 0x1862, 1, 0},
		{160, 'o', 0x685174, 0x17a9, 0, 0x1862, 1, 0}, {161, 'o', 0x685174, 0x17aa, 0, 0x1862, 1, 0}, {162, 'o', 0x685174, 0x17ab, 0, 0x1862, 1, 0}, {163, 'o', 0x685174, 0x17ac, 0, 0x1862, 1, 0},
		{164, 'o', 0x685174, 0x17ad, 0, 0x1862, 1, 0}, {165, 'o', 0x685174, 0x17ae, 0, 0x1862, 1, 0}, {166, 'o', 0x685174, 0x17af, 0, 0x1862, 1, 0}, {167, 'o', 0x685174, 0x17b0, 0, 0x1862, 1, 0},
		{168, 'o', 0x685174, 0x17b1, 0, 0x1862, 1, 0}, {169, 'o', 0x685174, 0x17b2, 0, 0x1862, 1, 0}, {170, 'o', 0x685174, 0x17b3, 0, 0x1862, 1, 0}, {171, 'o', 0x685174, 0x17b4, 0, 0x1862, 1, 0},
		{172, 'o', 0x685174, 0x17b5, 0, 0x1862, 1, 0}, {173, 'o', 0x685174, 0x17b6, 0, 0x1862, 1, 0}, {174, 'o', 0x685174, 0x17b7, 0, 0x1862, 1, 0}, {175, 'o', 0x685174, 0x17b8, 0, 0x1862, 1, 0},
		{176, 'o', 0x685174, 0x17b9, 0, 0x1862, 1, 0}, {177, 'o', 0x685174, 0x17ba, 0, 0x1862, 1, 0}, {178, 'o', 0x685174, 0x17bb, 0, 0x1862, 1, 0}, {179, 'o', 0x685174, 0x17bc, 0, 0x1862, 1, 0},
		{180, 'o', 0x685174, 0x17bd, 0, 0x1862, 1, 0}, {181, 'o', 0x685174, 0x17be, 0, 0x1862, 1, 0}, {182, 'o', 0x685174, 0x17bf, 0, 0x1862, 1, 0}, {183, 'o', 0x685174, 0x17c0, 0, 0x1862, 1, 0},
		{184, 'o', 0x685174, 0x17c1, 0, 0x1862, 1, 0}, {185, 'o', 0x685174, 0x17c2, 0, 0x1862, 1, 0}, {186, 'o', 0x685174, 0x17c3, 0, 0x1862, 1, 0}, {187, 'o', 0x685174, 0x17c4, 0, 0x1862, 1, 0},
		{188, 'o', 0x685174, 0x17c5, 0, 0x1862, 1, 0}, {189, 'o', 0x685174, 0x17c6, 0, 0x1862, 1, 0}, {190, 'o', 0x685174, 0x17c7, 0, 0x1862, 1, 0}, {191, 'o', 0x685174, 0x17c8, 0, 0x1862, 1, 0},
		{192, 'o', 0x685174, 0x17c9, 0, 0x1862, 1, 0}, {193, 'o', 0x685174, 0x17ca, 0, 0x1862, 1, 0}, {194, 'o', 0x685174, 0x17cb, 0, 0x1862, 1, 0}, {195, 'o', 0x685174, 0x17cc, 0, 0x1862, 1, 0},
		{196, 'o', 0x685174, 0x17cd, 0, 0x1862, 1, 0}, {197, 'o', 0x685174, 0x17ce, 0, 0x1862, 1, 0}, {198, 'o', 0x685174, 0x17cf, 0, 0x1862, 1, 0}, {199, 'o', 0x685174, 0x17d0, 0, 0x1862, 1, 0},
		{200, 'o', 0x685174, 0x17d1, 0, 0x1862, 1, 0}, {201, 'o', 0x685174, 0x17d2, 0, 0x1862, 1, 0}, {202, 'o', 0x685174, 0x17d3, 0, 0x1862, 1, 0}, {203, 'o', 0x685174, 0x17d4, 0, 0x1862, 1, 0},
		{204, 'o', 0x685174, 0x17d5, 0, 0x1862, 1, 0}, {205, 'o', 0x685174, 0x17d6, 0, 0x1862, 1, 0}, {206, 'o', 0x685174, 0xe9, 0, 0x1862, 4, 0}, {207, 'o', 0x685174, 0xfd, 0, 0x1862, 4, 0},
		{208, 'o', 0x685174, 0x111, 0, 0x1862, 4, 0}, {209, 'o', 0x685174, 0x125, 0, 0x1862, 4, 0}, {210, 'o', 0x685174, 0x139, 0, 0x1862, 4, 0}, {211, 'o', 0x685174, 0x14d, 0, 0x1862, 4, 0},
		{212, 'o', 0x685174, 0x161, 0, 0x1862, 4, 0}, {213, 'o', 0x685174, 0x1a3, 0, 0x1862, 4, 0}, {214, 'o', 0x685174, 0x1c9, 0, 0x1862, 4, 0}, {215, 'o', 0x685174, 0x1ef, 0, 0x1862, 4, 0},
		{216, 'o', 0x685174, 0x215, 0, 0x1862, 4, 0}, {217, 'o', 0x685174, 0x23b, 0, 0x1862, 4, 0}, {218, 'o', 0x685174, 0x261, 0, 0x1862, 4, 0}, {219, 'o', 0x685174, 0x287, 0, 0x1862, 4, 0},
		{220, 'o', 0x685174, 0x2ad, 0, 0x1862, 4, 0}, {221, 'o', 0x685174, 0x2d3, 0, 0x1862, 4, 0}, {222, 'o', 0x685174, 0x2f9, 0, 0x1862, 4, 0}, {223, 'o', 0x685174, 0x31f, 0, 0x1862, 4, 0},
		{224, 'o', 0x685174, 0x345, 0, 0x1862, 4, 0}, {225, 'o', 0x685174, 0x36b, 0, 0x1862, 4, 0}, {226, 'o', 0x685174, 0x391, 0, 0x1862, 4, 0}, {227, 'o', 0x685174, 0x3b7, 0, 0x1862, 4, 0},
		{228, 'o', 0x685174, 0x3dd, 0, 0x1862, 4, 0}, {229, 'o', 0x685174, 0x403, 0, 0x1862, 4, 0}, {230, 'o', 0x685174, 0x429, 0, 0x1862, 4, 0}, {231, 'o', 0x685174, 0x44f, 0, 0x1862, 4, 0},
		{232, 'o', 0x685174, 0x475, 0, 0x1862, 4, 0}, {233, 'o', 0x685174, 0x49b, 0, 0x1862, 4, 0}, {234, 'o', 0x685174, 0x4c1, 0, 0x1862, 4, 0}, {235, 'o', 0x685174, 0x4e7, 0, 0x1862, 4, 0},
		{236, 'o', 0x685174, 0x50d, 0, 0x1862, 4, 0}, {237, 'o', 0x685174, 0x533, 0, 0x1862, 4, 0}, {238, 'o', 0x685174, 0x559, 0, 0x1862, 4, 0}, {239, 'o', 0x685174, 0x57f, 0, 0x1862, 4, 0},
		{240, 'o', 0x685174, 0x5a5, 0, 0x1862, 4, 0}, {241, 'o', 0x685174, 0x5cb, 0, 0x1862, 4, 0}, {242, 'o', 0x685174, 0x5f1, 0, 0x1862, 4, 0}, {243, 'o', 0x685174, 0x617, 0, 0x1862, 4, 0},
		{244, 'o', 0x685174, 0x63d, 0, 0x1862, 4, 0}, {245, 'o', 0x685174, 0x663, 0, 0x1862, 4, 0}, {246, 'o', 0x685174, 0x689, 0, 0x1862, 4, 0}, {247, 'o', 0x685174, 0x6af, 0, 0x1862, 4, 0},
		{248, 'o', 0x685174, 0x6d5, 0, 0x1862, 4, 0}, {249, 'o', 0x685174, 0x6fb, 0, 0x1862, 4, 0}, {250, 'o', 0x685174, 0x721, 0, 0x1862, 4, 0}, {251, 'o', 0x685174, 0x747, 0, 0x1862, 4, 0},
		{252, 'o', 0x685174, 0x76d, 0, 0x1862, 4, 0}, {253, 'o', 0x685174, 0x793, 0, 0x1862, 4, 0}, {254, 'o', 0x685174, 0xbc5, 0, 0x1862, 4, 0}, {255, 'o', 0x685174, 0xbb9, 0, 0x1862, 4, 0},
		{256, 'o', 0x685174, 0xbc9, 0, 0x1862, 4, 0}, {257, 'o', 0x685174, 0xa65, 0, 0x1862, 4, 0}, {258, 'o', 0x685174, 0xa69, 0, 0x1862, 4, 0}, {259, 'o', 0x685174, 0xa6d, 0, 0x1862, 4, 0},
		{260, 'o', 0x685174, 0xa71, 0, 0x1862, 4, 0}, {261, 'o', 0x685174, 0xa75, 0, 0x1862, 4, 0}, {262, 'o', 0x685174, 0xa79, 0, 0x1862, 4, 0}, {263, 'o', 0x685174, 0xa7d, 0, 0x1862, 4, 0},
		{264, 'o', 0x685174, 0xa81, 0, 0x1862, 4, 0}, {265, 'o', 0x685174, 0xa85, 0, 0x1862, 4, 0}, {266, 'o', 0x685174, 0xa89, 0, 0x1862, 4, 0}, {267, 'o', 0x685174, 0xa8d, 0, 0x1862, 4, 0},
		{268, 'o', 0x685174, 0xa91, 0, 0x1862, 4, 0}, {269, 'o', 0x685174, 0xa95, 0, 0x1862, 4, 0}, {270, 'o', 0x685174, 0xa99, 0, 0x1862, 4, 0}, {271, 'o', 0x685174, 0xa9d, 0, 0x1862, 4, 0},
		{272, 'o', 0x685174, 0xaa1, 0, 0x1862, 4, 0}, {273, 'o', 0x685174, 0xaa5, 0, 0x1862, 4, 0}, {274, 'o', 0x685174, 0xaa9, 0, 0x1862, 4, 0}, {275, 'o', 0x685174, 0xaad, 0, 0x1862, 4, 0},
		{276, 'o', 0x685174, 0xab1, 0, 0x1862, 4, 0}, {277, 'o', 0x685174, 0xab5, 0, 0x1862, 4, 0}, {278, 'o', 0x685174, 0xab9, 0, 0x1862, 4, 0}, {279, 'o', 0x685174, 0xabd, 0, 0x1862, 4, 0},
		{280, 'o', 0x685174, 0xac1, 0, 0x1862, 4, 0}, {281, 'o', 0x685174, 0xb05, 0, 0x1862, 4, 0}, {282, 'o', 0x69c0f4, 0xf4, -1, 0x0, 4, 0}, {283, 'o', 0x69c0f4, 0xf8, -1, 0x0, 4, 0},
		{284, 'g', 0x6840bb, 0, -1, 0, 1, 0},
	}},
	0x548e60: {potionCreationNode, 84, 0x20001000, []ctrsLink{
		{0, 'w', 0, 0, -1, 0, 4, 0}, {1, 'a', 0, 0, -1, 0, 4, 0}, {2, 'o', 0x685178, 0x0, -1, 0x0, 1, 0}, {3, 'o', 0x685178, 0x106, -1, 0x0, 1, 0},
		{4, 'o', 0x685178, 0x20c, -1, 0x0, 1, 0}, {5, 'o', 0x685178, 0x312, -1, 0x0, 1, 0}, {6, 'o', 0x685178, 0x418, -1, 0x0, 1, 0}, {7, 'o', 0x685178, 0x51e, -1, 0x0, 1, 0},
		{8, 'o', 0x685178, 0x624, -1, 0x0, 1, 0}, {9, 'o', 0x685178, 0x72a, -1, 0x0, 1, 0}, {10, 'o', 0x685174, 0xb01, -1, 0x0, 4, 0}, {11, 'o', 0x685174, 0x2363, -1, 0x0, 4, 0},
		{12, 'o', 0x685174, 0x3bc5, -1, 0x0, 4, 0}, {13, 'o', 0x685174, 0x5427, -1, 0x0, 4, 0}, {14, 'o', 0x685174, 0x6c89, -1, 0x0, 4, 0}, {15, 'o', 0x685174, 0x84eb, -1, 0x0, 4, 0},
		{16, 'o', 0x685174, 0x9d4d, -1, 0x0, 4, 0}, {17, 'o', 0x685174, 0xb5af, -1, 0x0, 4, 0}, {18, 'o', 0x685174, 0xb9, 0, 0x1862, 4, 1}, {19, 'o', 0x685174, 0x69, 0, 0x1862, 4, 1},
		{20, 'o', 0x685174, 0x71, 0, 0x1862, 4, 1}, {21, 'o', 0x685174, 0x89, 0, 0x1862, 4, 0}, {22, 'o', 0x685174, 0xbb9, 0, 0x1862, 4, 0}, {23, 'o', 0x685174, 0xbc5, 0, 0x1862, 4, 0},
		{24, 'o', 0x685174, 0xb65, 0, 0x1862, 4, 0}, {25, 'o', 0x685174, 0x1029, 0, 0x1862, 4, 1}, {26, 'o', 0x685174, 0x1035, 0, 0x1862, 4, 1}, {27, 'o', 0x685174, 0x1041, 0, 0x1862, 4, 1},
		{28, 'o', 0x685174, 0x104d, 0, 0x1862, 4, 1}, {29, 'o', 0x685174, 0x1059, 0, 0x1862, 4, 1}, {30, 'o', 0x685174, 0x1065, 0, 0x1862, 4, 1}, {31, 'o', 0x685174, 0x1071, 0, 0x1862, 4, 1},
		{32, 'o', 0x685174, 0x107d, 0, 0x1862, 4, 1}, {33, 'g', 0x68ec78, 0, -1, 0, 4, 0}, {34, 'g', 0x68c558, 0, -1, 0, 4, 0}, {36, 'g', 0x686901, 0, -1, 0, 4, 0},
		{37, 'g', 0x685191, 0, -1, 0, 4, 1}, {38, 'g', 0x68519d, 0, -1, 0, 4, 1}, {39, 'g', 0x6851a9, 0, -1, 0, 4, 1}, {40, 'g', 0x6851b5, 0, -1, 0, 4, 1},
		{41, 'g', 0x6851c1, 0, -1, 0, 4, 1}, {42, 'g', 0x6851cd, 0, -1, 0, 4, 1}, {46, 'g', 0x10007ff8, 0, -1, 0, 1, 0}, {47, 'g', 0x10007ff9, 0, -1, 0, 1, 0},
		{48, 'g', 0x10007ffa, 0, -1, 0, 1, 0}, {49, 'g', 0x10007ffb, 0, -1, 0, 1, 0}, {50, 'g', 0x10007ffc, 0, -1, 0, 1, 0}, {51, 'g', 0x10007ffd, 0, -1, 0, 1, 0},
		{52, 'g', 0x10007ffe, 0, -1, 0, 1, 0}, {53, 'g', 0x10007fff, 0, -1, 0, 1, 0},
	}, []ctrsLink{
		{0, 'o', 0x685174, 0xb65, 0, 0x1862, 4, 0}, {26, 'g', 0x10007ff4, 0, -1, 0, 4, 0}, {27, 'g', 0x10007ff8, 0, -1, 0, 4, 0}, {28, 'g', 0x10007ffc, 0, -1, 0, 4, 0},
		{29, 'o', 0x685174, 0xbb9, 0, 0x1862, 4, 0}, {30, 'o', 0x69c0f4, 0xfc, -1, 0x0, 4, 0}, {31, 'g', 0x686901, 0, -1, 0, 4, 0}, {32, 'g', 0x685191, 0, -1, 0, 4, 1},
		{33, 'g', 0x685195, 0, -1, 0, 4, 0}, {34, 'g', 0x685199, 0, -1, 0, 4, 0}, {35, 'g', 0x68519d, 0, -1, 0, 4, 1}, {36, 'g', 0x6851a1, 0, -1, 0, 4, 0},
		{37, 'g', 0x6851a5, 0, -1, 0, 4, 0}, {38, 'g', 0x6851a9, 0, -1, 0, 4, 1}, {39, 'g', 0x6851ad, 0, -1, 0, 4, 0}, {40, 'g', 0x6851b1, 0, -1, 0, 4, 0},
		{41, 'g', 0x6851b5, 0, -1, 0, 4, 1}, {42, 'g', 0x6851b9, 0, -1, 0, 4, 0}, {43, 'g', 0x6851bd, 0, -1, 0, 4, 0}, {44, 'g', 0x6851c1, 0, -1, 0, 4, 1},
		{45, 'g', 0x6851c5, 0, -1, 0, 4, 0}, {46, 'g', 0x6851c9, 0, -1, 0, 4, 0}, {47, 'g', 0x6851cd, 0, -1, 0, 4, 1}, {48, 'g', 0x6851d1, 0, -1, 0, 4, 0},
		{49, 'g', 0x6851d5, 0, -1, 0, 4, 0}, {50, 'g', 0x6851d9, 0, -1, 0, 4, 1}, {51, 'g', 0x6851dd, 0, -1, 0, 4, 0}, {52, 'g', 0x6851e1, 0, -1, 0, 4, 0},
		{53, 'o', 0x685174, 0x1029, 0, 0x1862, 4, 1}, {54, 'o', 0x685174, 0x102d, 0, 0x1862, 4, 0}, {55, 'o', 0x685174, 0x1031, 0, 0x1862, 4, 0}, {56, 'o', 0x685174, 0x1035, 0, 0x1862, 4, 1},
		{57, 'o', 0x685174, 0x1039, 0, 0x1862, 4, 0}, {58, 'o', 0x685174, 0x103d, 0, 0x1862, 4, 0}, {59, 'o', 0x685174, 0x1041, 0, 0x1862, 4, 1}, {60, 'o', 0x685174, 0x1045, 0, 0x1862, 4, 0},
		{61, 'o', 0x685174, 0x1049, 0, 0x1862, 4, 0}, {62, 'o', 0x685174, 0x104d, 0, 0x1862, 4, 1}, {63, 'o', 0x685174, 0x1051, 0, 0x1862, 4, 0}, {64, 'o', 0x685174, 0x1055, 0, 0x1862, 4, 0},
		{65, 'o', 0x685174, 0x1059, 0, 0x1862, 4, 1}, {66, 'o', 0x685174, 0x105d, 0, 0x1862, 4, 0}, {67, 'o', 0x685174, 0x1061, 0, 0x1862, 4, 0}, {68, 'o', 0x685174, 0x1065, 0, 0x1862, 4, 1},
		{69, 'o', 0x685174, 0x1069, 0, 0x1862, 4, 0}, {70, 'o', 0x685174, 0x106d, 0, 0x1862, 4, 0}, {71, 'o', 0x685174, 0x1071, 0, 0x1862, 4, 1}, {72, 'o', 0x685174, 0x1075, 0, 0x1862, 4, 0},
		{73, 'o', 0x685174, 0x1079, 0, 0x1862, 4, 0}, {74, 'o', 0x685174, 0x107d, 0, 0x1862, 4, 1}, {75, 'o', 0x685174, 0x1081, 0, 0x1862, 4, 0}, {76, 'o', 0x685174, 0x1085, 0, 0x1862, 4, 0},
		{77, 'o', 0x685174, 0x1089, 0, 0x1862, 4, 1}, {78, 'o', 0x685174, 0x108d, 0, 0x1862, 4, 0}, {79, 'o', 0x685174, 0x1091, 0, 0x1862, 4, 0}, {80, 'o', 0x685174, 0x1095, 0, 0x1862, 4, 1},
		{81, 'o', 0x685174, 0x1099, 0, 0x1862, 4, 0}, {82, 'o', 0x685174, 0x109d, 0, 0x1862, 4, 0}, {83, 'o', 0x685174, 0x10a1, 0, 0x1862, 4, 1}, {84, 'o', 0x685174, 0x10a5, 0, 0x1862, 4, 0},
		{85, 'o', 0x685174, 0x10a9, 0, 0x1862, 4, 0}, {86, 'o', 0x685174, 0x10ad, 0, 0x1862, 4, 1}, {87, 'o', 0x685174, 0x10b1, 0, 0x1862, 4, 0}, {88, 'o', 0x685174, 0x10b5, 0, 0x1862, 4, 0},
	}},
	0x51d960: {invalidEquipmentRemovalNode, 550, 0x0, []ctrsLink{
		{0, 'a', 0, 0, -1, 0, 4, 0}, {1, 'w', 0, 0, -1, 0, 4, 0}, {2, 'g', 0x685174, 0, -1, 0, 4, 0}, {3, 'g', 0x68516c, 0, -1, 0, 4, 0},
		{4, 'g', 0x68ec78, 0, -1, 0, 4, 0}, {5, 'o', 0x685174, 0x89, 1, 0x1862, 4, 0}, {6, 'o', 0x685174, 0xbb9, 1, 0x1862, 4, 0}, {7, 'o', 0x685174, 0xbc5, 1, 0x1862, 4, 0},
		{8, 'g', 0x5ee664, 0, -1, 0, 4, 0}, {9, 'g', 0x5ee5a8, 0, -1, 0, 4, 0}, {10, 'g', 0x68c558, 0, -1, 0, 4, 0}, {11, 'g', 0x5ed8c8, 0, -1, 0, 4, 0},
		{12, 'g', 0x5ed914, 0, -1, 0, 4, 0}, {13, 'o', 0x69c0f4, 0xfc, -1, 0x0, 4, 0}, {15, 'g', 0x686901, 0, -1, 0, 4, 0}, {16, 'o', 0x685174, 0xe9, 1, 0x1862, 4, 0},
		{17, 'o', 0x685174, 0xfd, 1, 0x1862, 4, 0}, {18, 'o', 0x685174, 0x111, 1, 0x1862, 4, 0}, {19, 'o', 0x685174, 0x125, 1, 0x1862, 4, 0}, {20, 'o', 0x685174, 0x139, 1, 0x1862, 4, 0},
		{21, 'o', 0x685174, 0x14d, 1, 0x1862, 4, 0}, {22, 'o', 0x685174, 0x161, 1, 0x1862, 4, 0}, {23, 'o', 0x685174, 0x175, 1, 0x1862, 4, 0}, {24, 'o', 0x685174, 0x1a3, 1, 0x1862, 4, 0},
		{25, 'o', 0x685174, 0x1c9, 1, 0x1862, 4, 0}, {26, 'o', 0x685174, 0x1ef, 1, 0x1862, 4, 0}, {27, 'o', 0x685174, 0x215, 1, 0x1862, 4, 0}, {28, 'o', 0x685174, 0x23b, 1, 0x1862, 4, 0},
		{29, 'o', 0x685174, 0x261, 1, 0x1862, 4, 0}, {30, 'o', 0x685174, 0x287, 1, 0x1862, 4, 0}, {31, 'o', 0x685174, 0x2ad, 1, 0x1862, 4, 0}, {32, 'o', 0x685174, 0x2d3, 1, 0x1862, 4, 0},
		{33, 'o', 0x685174, 0x2f9, 1, 0x1862, 4, 0}, {34, 'o', 0x685174, 0x31f, 1, 0x1862, 4, 0}, {35, 'o', 0x685174, 0x345, 1, 0x1862, 4, 0}, {36, 'o', 0x685174, 0x36b, 1, 0x1862, 4, 0},
		{37, 'o', 0x685174, 0x391, 1, 0x1862, 4, 0}, {38, 'o', 0x685174, 0x3b7, 1, 0x1862, 4, 0}, {39, 'o', 0x685174, 0x3dd, 1, 0x1862, 4, 0}, {40, 'o', 0x685174, 0x403, 1, 0x1862, 4, 0},
		{41, 'o', 0x685174, 0x429, 1, 0x1862, 4, 0}, {42, 'o', 0x685174, 0x44f, 1, 0x1862, 4, 0}, {43, 'o', 0x685174, 0x475, 1, 0x1862, 4, 0}, {44, 'o', 0x685174, 0x49b, 1, 0x1862, 4, 0},
		{45, 'o', 0x685174, 0x4c1, 1, 0x1862, 4, 0}, {46, 'o', 0x685174, 0x4e7, 1, 0x1862, 4, 0}, {47, 'o', 0x685174, 0x50d, 1, 0x1862, 4, 0}, {48, 'o', 0x685174, 0x533, 1, 0x1862, 4, 0},
		{49, 'o', 0x685174, 0x559, 1, 0x1862, 4, 0}, {50, 'o', 0x685174, 0x57f, 1, 0x1862, 4, 0}, {51, 'o', 0x685174, 0x5a5, 1, 0x1862, 4, 0}, {52, 'o', 0x685174, 0x5cb, 1, 0x1862, 4, 0},
		{53, 'o', 0x685174, 0x5f1, 1, 0x1862, 4, 0}, {54, 'o', 0x685174, 0x617, 1, 0x1862, 4, 0}, {55, 'o', 0x685174, 0x63d, 1, 0x1862, 4, 0}, {56, 'o', 0x685174, 0x663, 1, 0x1862, 4, 0},
		{57, 'o', 0x685174, 0x689, 1, 0x1862, 4, 0}, {58, 'o', 0x685174, 0x6af, 1, 0x1862, 4, 0}, {59, 'o', 0x685174, 0x6d5, 1, 0x1862, 4, 0}, {60, 'o', 0x685174, 0x6fb, 1, 0x1862, 4, 0},
		{61, 'o', 0x685174, 0x721, 1, 0x1862, 4, 0}, {62, 'o', 0x685174, 0x747, 1, 0x1862, 4, 0}, {63, 'o', 0x685174, 0x76d, 1, 0x1862, 4, 0}, {64, 'o', 0x685174, 0x793, 1, 0x1862, 4, 0},
		{65, 'o', 0x685174, 0xf5d, 1, 0x1862, 4, 1}, {66, 'o', 0x685174, 0xf61, 1, 0x1862, 4, 0}, {67, 'o', 0x685174, 0xf65, 1, 0x1862, 4, 0}, {68, 'o', 0x685174, 0xf69, 1, 0x1862, 4, 1},
		{69, 'o', 0x685174, 0xf6d, 1, 0x1862, 4, 0}, {70, 'o', 0x685174, 0xf71, 1, 0x1862, 4, 0}, {71, 'o', 0x685174, 0xf75, 1, 0x1862, 4, 1}, {72, 'o', 0x685174, 0xf79, 1, 0x1862, 4, 0},
		{73, 'o', 0x685174, 0xf7d, 1, 0x1862, 4, 0}, {74, 'o', 0x685174, 0xf81, 1, 0x1862, 4, 1}, {75, 'o', 0x685174, 0xf85, 1, 0x1862, 4, 0}, {76, 'o', 0x685174, 0xf89, 1, 0x1862, 4, 0},
		{77, 'o', 0x685174, 0xf8d, 1, 0x1862, 4, 1}, {78, 'o', 0x685174, 0xf91, 1, 0x1862, 4, 0}, {79, 'o', 0x685174, 0xf95, 1, 0x1862, 4, 0}, {80, 'o', 0x685174, 0xf99, 1, 0x1862, 4, 1},
		{81, 'o', 0x685174, 0xf9d, 1, 0x1862, 4, 0}, {82, 'o', 0x685174, 0xfa1, 1, 0x1862, 4, 0}, {83, 'o', 0x685174, 0xfa5, 1, 0x1862, 4, 1}, {84, 'o', 0x685174, 0xfa9, 1, 0x1862, 4, 0},
		{85, 'o', 0x685174, 0xfad, 1, 0x1862, 4, 0}, {86, 'o', 0x685174, 0xfb1, 1, 0x1862, 4, 1}, {87, 'o', 0x685174, 0xfb5, 1, 0x1862, 4, 0}, {88, 'o', 0x685174, 0xfb9, 1, 0x1862, 4, 0},
		{89, 'o', 0x685174, 0xfbd, 1, 0x1862, 4, 1}, {90, 'o', 0x685174, 0xfc1, 1, 0x1862, 4, 0}, {91, 'o', 0x685174, 0xfc5, 1, 0x1862, 4, 0}, {92, 'o', 0x685174, 0xfc9, 1, 0x1862, 4, 1},
		{93, 'o', 0x685174, 0xfcd, 1, 0x1862, 4, 0}, {94, 'o', 0x685174, 0xfd1, 1, 0x1862, 4, 0}, {95, 'o', 0x685174, 0xfd5, 1, 0x1862, 4, 1}, {96, 'o', 0x685174, 0xfd9, 1, 0x1862, 4, 0},
		{97, 'o', 0x685174, 0xfdd, 1, 0x1862, 4, 0}, {98, 'o', 0x685174, 0xfe1, 1, 0x1862, 4, 1}, {99, 'o', 0x685174, 0xfe5, 1, 0x1862, 4, 0}, {100, 'o', 0x685174, 0xfe9, 1, 0x1862, 4, 0},
		{101, 'o', 0x685174, 0x1029, 1, 0x1862, 4, 1}, {102, 'o', 0x685174, 0x102d, 1, 0x1862, 4, 0}, {103, 'o', 0x685174, 0x1031, 1, 0x1862, 4, 0}, {104, 'o', 0x685174, 0x1035, 1, 0x1862, 4, 1},
		{105, 'o', 0x685174, 0x1039, 1, 0x1862, 4, 0}, {106, 'o', 0x685174, 0x103d, 1, 0x1862, 4, 0}, {107, 'o', 0x685174, 0x1041, 1, 0x1862, 4, 1}, {108, 'o', 0x685174, 0x1045, 1, 0x1862, 4, 0},
		{109, 'o', 0x685174, 0x1049, 1, 0x1862, 4, 0}, {110, 'o', 0x685174, 0x104d, 1, 0x1862, 4, 1}, {111, 'o', 0x685174, 0x1051, 1, 0x1862, 4, 0}, {112, 'o', 0x685174, 0x1055, 1, 0x1862, 4, 0},
		{113, 'o', 0x685174, 0x1059, 1, 0x1862, 4, 1}, {114, 'o', 0x685174, 0x105d, 1, 0x1862, 4, 0}, {115, 'o', 0x685174, 0x1061, 1, 0x1862, 4, 0}, {116, 'o', 0x685174, 0x1065, 1, 0x1862, 4, 1},
		{117, 'o', 0x685174, 0x1069, 1, 0x1862, 4, 0}, {118, 'o', 0x685174, 0x106d, 1, 0x1862, 4, 0}, {119, 'o', 0x685174, 0x1071, 1, 0x1862, 4, 1}, {120, 'o', 0x685174, 0x1075, 1, 0x1862, 4, 0},
		{121, 'o', 0x685174, 0x1079, 1, 0x1862, 4, 0}, {122, 'o', 0x685174, 0x107d, 1, 0x1862, 4, 1}, {123, 'o', 0x685174, 0x1081, 1, 0x1862, 4, 0}, {124, 'o', 0x685174, 0x1085, 1, 0x1862, 4, 0},
		{125, 'o', 0x685174, 0x1089, 1, 0x1862, 4, 1}, {126, 'o', 0x685174, 0x108d, 1, 0x1862, 4, 0}, {127, 'o', 0x685174, 0x1091, 1, 0x1862, 4, 0}, {128, 'o', 0x685174, 0x1095, 1, 0x1862, 4, 1},
		{129, 'o', 0x685174, 0x1099, 1, 0x1862, 4, 0}, {130, 'o', 0x685174, 0x109d, 1, 0x1862, 4, 0}, {131, 'o', 0x685174, 0x10a1, 1, 0x1862, 4, 1}, {132, 'o', 0x685174, 0x10a5, 1, 0x1862, 4, 0},
		{133, 'o', 0x685174, 0x10a9, 1, 0x1862, 4, 0}, {134, 'o', 0x685174, 0x10ad, 1, 0x1862, 4, 1}, {135, 'o', 0x685174, 0x10b1, 1, 0x1862, 4, 0}, {136, 'o', 0x685174, 0x10b5, 1, 0x1862, 4, 0},
		{137, 'g', 0x685191, 0, -1, 0, 4, 1}, {138, 'g', 0x685195, 0, -1, 0, 4, 0}, {139, 'g', 0x685199, 0, -1, 0, 4, 0}, {140, 'g', 0x68519d, 0, -1, 0, 4, 1},
		{141, 'g', 0x6851a1, 0, -1, 0, 4, 0}, {142, 'g', 0x6851a5, 0, -1, 0, 4, 0}, {143, 'g', 0x6837cb, 0, -1, 0, 1, 0}, {144, 'g', 0x6838e3, 0, -1, 0, 1, 0},
		{145, 'g', 0x6839fb, 0, -1, 0, 1, 0}, {146, 'g', 0x683b13, 0, -1, 0, 1, 0}, {147, 'g', 0x683c2b, 0, -1, 0, 1, 0}, {148, 'g', 0x683d43, 0, -1, 0, 1, 0},
		{149, 'g', 0x683e5b, 0, -1, 0, 1, 0}, {150, 'g', 0x683f73, 0, -1, 0, 1, 0},
	}, []ctrsLink{
		{0, 'o', 0x685174, 0xf5d, 1, 0x1862, 4, 0}, {1, 'o', 0x685174, 0xf61, 1, 0x1862, 4, 0}, {2, 'o', 0x685174, 0xf65, 1, 0x1862, 4, 0}, {3, 'o', 0x685174, 0xf69, 1, 0x1862, 4, 0},
		{4, 'o', 0x685174, 0xf6d, 1, 0x1862, 4, 0}, {5, 'o', 0x685174, 0xf71, 1, 0x1862, 4, 0}, {6, 'o', 0x685174, 0xf75, 1, 0x1862, 4, 0}, {7, 'o', 0x685174, 0xf79, 1, 0x1862, 4, 0},
		{8, 'o', 0x685174, 0xf7d, 1, 0x1862, 4, 0}, {9, 'o', 0x685174, 0xf81, 1, 0x1862, 4, 0}, {10, 'o', 0x685174, 0xf85, 1, 0x1862, 4, 0}, {11, 'o', 0x685174, 0xf89, 1, 0x1862, 4, 0},
		{12, 'o', 0x685174, 0xf8d, 1, 0x1862, 4, 0}, {13, 'o', 0x685174, 0xf91, 1, 0x1862, 4, 0}, {14, 'o', 0x685174, 0xf95, 1, 0x1862, 4, 0}, {15, 'o', 0x685174, 0xf99, 1, 0x1862, 4, 0},
		{16, 'o', 0x685174, 0xf9d, 1, 0x1862, 4, 0}, {17, 'o', 0x685174, 0xfa1, 1, 0x1862, 4, 0}, {18, 'o', 0x685174, 0xfa5, 1, 0x1862, 4, 0}, {19, 'o', 0x685174, 0xfa9, 1, 0x1862, 4, 0},
		{20, 'o', 0x685174, 0xfad, 1, 0x1862, 4, 0}, {21, 'o', 0x685174, 0xfb1, 1, 0x1862, 4, 0}, {22, 'o', 0x685174, 0xfb5, 1, 0x1862, 4, 0}, {23, 'o', 0x685174, 0xfb9, 1, 0x1862, 4, 0},
		{24, 'o', 0x685174, 0xfbd, 1, 0x1862, 4, 0}, {25, 'o', 0x685174, 0xfc1, 1, 0x1862, 4, 0}, {26, 'o', 0x685174, 0xfc5, 1, 0x1862, 4, 0}, {27, 'o', 0x685174, 0xfc9, 1, 0x1862, 4, 0},
		{28, 'o', 0x685174, 0xfcd, 1, 0x1862, 4, 0}, {29, 'o', 0x685174, 0xfd1, 1, 0x1862, 4, 0}, {30, 'o', 0x685174, 0xfd5, 1, 0x1862, 4, 0}, {31, 'o', 0x685174, 0xfd9, 1, 0x1862, 4, 0},
		{32, 'o', 0x685174, 0xfdd, 1, 0x1862, 4, 0}, {33, 'o', 0x685174, 0xfe1, 1, 0x1862, 4, 0}, {34, 'o', 0x685174, 0xfe5, 1, 0x1862, 4, 0}, {35, 'o', 0x685174, 0xfe9, 1, 0x1862, 4, 0},
		{36, 'o', 0x685174, 0x1029, 1, 0x1862, 4, 0}, {37, 'o', 0x685174, 0x102d, 1, 0x1862, 4, 0}, {38, 'o', 0x685174, 0x1031, 1, 0x1862, 4, 0}, {39, 'o', 0x685174, 0x1035, 1, 0x1862, 4, 0},
		{40, 'o', 0x685174, 0x1039, 1, 0x1862, 4, 0}, {41, 'o', 0x685174, 0x103d, 1, 0x1862, 4, 0}, {42, 'o', 0x685174, 0x1041, 1, 0x1862, 4, 0}, {43, 'o', 0x685174, 0x1045, 1, 0x1862, 4, 0},
		{44, 'o', 0x685174, 0x1049, 1, 0x1862, 4, 0}, {45, 'o', 0x685174, 0x104d, 1, 0x1862, 4, 0}, {46, 'o', 0x685174, 0x1051, 1, 0x1862, 4, 0}, {47, 'o', 0x685174, 0x1055, 1, 0x1862, 4, 0},
		{48, 'o', 0x685174, 0x1059, 1, 0x1862, 4, 0}, {49, 'o', 0x685174, 0x105d, 1, 0x1862, 4, 0}, {50, 'o', 0x685174, 0x1061, 1, 0x1862, 4, 0}, {51, 'o', 0x685174, 0x1065, 1, 0x1862, 4, 0},
		{52, 'o', 0x685174, 0x1069, 1, 0x1862, 4, 0}, {53, 'o', 0x685174, 0x106d, 1, 0x1862, 4, 0}, {54, 'o', 0x685174, 0x1071, 1, 0x1862, 4, 0}, {55, 'o', 0x685174, 0x1075, 1, 0x1862, 4, 0},
		{56, 'o', 0x685174, 0x1079, 1, 0x1862, 4, 0}, {57, 'o', 0x685174, 0x107d, 1, 0x1862, 4, 0}, {58, 'o', 0x685174, 0x1081, 1, 0x1862, 4, 0}, {59, 'o', 0x685174, 0x1085, 1, 0x1862, 4, 0},
		{60, 'o', 0x685174, 0x1089, 1, 0x1862, 4, 0}, {61, 'o', 0x685174, 0x108d, 1, 0x1862, 4, 0}, {62, 'o', 0x685174, 0x1091, 1, 0x1862, 4, 0}, {63, 'o', 0x685174, 0x1095, 1, 0x1862, 4, 0},
		{64, 'o', 0x685174, 0x1099, 1, 0x1862, 4, 0}, {65, 'o', 0x685174, 0x109d, 1, 0x1862, 4, 0}, {66, 'o', 0x685174, 0x10a1, 1, 0x1862, 4, 0}, {67, 'o', 0x685174, 0x10a5, 1, 0x1862, 4, 0},
		{68, 'o', 0x685174, 0x10a9, 1, 0x1862, 4, 0}, {69, 'o', 0x685174, 0x10ad, 1, 0x1862, 4, 0}, {70, 'o', 0x685174, 0x10b1, 1, 0x1862, 4, 0}, {71, 'o', 0x685174, 0x10b5, 1, 0x1862, 4, 0},
		{72, 'g', 0x685191, 0, -1, 0, 4, 0}, {73, 'g', 0x685195, 0, -1, 0, 4, 0}, {74, 'g', 0x685199, 0, -1, 0, 4, 0}, {75, 'g', 0x68519d, 0, -1, 0, 4, 0},
		{76, 'g', 0x6851a1, 0, -1, 0, 4, 0}, {77, 'g', 0x6851a5, 0, -1, 0, 4, 0}, {78, 'g', 0x6851a9, 0, -1, 0, 4, 0}, {79, 'g', 0x6851ad, 0, -1, 0, 4, 0},
		{80, 'g', 0x6851b1, 0, -1, 0, 4, 0}, {81, 'g', 0x6851b5, 0, -1, 0, 4, 0}, {82, 'g', 0x6851b9, 0, -1, 0, 4, 0}, {83, 'g', 0x6851bd, 0, -1, 0, 4, 0},
		{84, 'g', 0x6851c1, 0, -1, 0, 4, 0}, {85, 'g', 0x6851c5, 0, -1, 0, 4, 0}, {86, 'g', 0x6851c9, 0, -1, 0, 4, 0}, {87, 'g', 0x6851cd, 0, -1, 0, 4, 0},
		{88, 'g', 0x6851d1, 0, -1, 0, 4, 0}, {89, 'g', 0x6851d5, 0, -1, 0, 4, 0}, {90, 'g', 0x6851d9, 0, -1, 0, 4, 0}, {91, 'g', 0x6851dd, 0, -1, 0, 4, 0},
		{92, 'g', 0x6851e1, 0, -1, 0, 4, 0}, {93, 'g', 0x6851e5, 0, -1, 0, 4, 0}, {94, 'g', 0x6851e9, 0, -1, 0, 4, 0}, {95, 'g', 0x6851ed, 0, -1, 0, 4, 0},
		{96, 'g', 0x6851f1, 0, -1, 0, 4, 0}, {97, 'g', 0x6851f5, 0, -1, 0, 4, 0}, {98, 'g', 0x6851f9, 0, -1, 0, 4, 0}, {99, 'g', 0x6851fd, 0, -1, 0, 4, 0},
		{100, 'g', 0x685201, 0, -1, 0, 4, 0}, {101, 'g', 0x685205, 0, -1, 0, 4, 0}, {102, 'g', 0x685209, 0, -1, 0, 4, 0}, {103, 'g', 0x68520d, 0, -1, 0, 4, 0},
		{104, 'g', 0x685211, 0, -1, 0, 4, 0}, {105, 'g', 0x685215, 0, -1, 0, 4, 0}, {106, 'g', 0x685219, 0, -1, 0, 4, 0}, {107, 'g', 0x68521d, 0, -1, 0, 4, 0},
		{108, 'g', 0x685221, 0, -1, 0, 4, 0}, {109, 'g', 0x685225, 0, -1, 0, 4, 0}, {110, 'g', 0x685229, 0, -1, 0, 4, 0}, {111, 'g', 0x68522d, 0, -1, 0, 4, 0},
		{112, 'g', 0x685231, 0, -1, 0, 4, 0}, {113, 'g', 0x685235, 0, -1, 0, 4, 0}, {114, 'g', 0x686901, 0, -1, 0, 4, 0}, {115, 'g', 0x6837cb, 0, -1, 0, 1, 0},
		{116, 'g', 0x6838e3, 0, -1, 0, 1, 0}, {117, 'g', 0x6839fb, 0, -1, 0, 1, 0}, {118, 'g', 0x683b13, 0, -1, 0, 1, 0}, {119, 'g', 0x683c2b, 0, -1, 0, 1, 0},
		{120, 'g', 0x683d43, 0, -1, 0, 1, 0}, {121, 'g', 0x683e5b, 0, -1, 0, 1, 0}, {122, 'g', 0x683f73, 0, -1, 0, 1, 0}, {123, 'o', 0x685174, 0xbb9, 1, 0x1862, 4, 0},
		{124, 'o', 0x69c0f4, 0xfc, -1, 0x0, 4, 0},
	}},
	0x4ed9d0: {rebuildCharacterRuleAbilitiesNode, 628, 0x20004000, []ctrsLink{
		{0, 'a', 0, 0, -1, 0, 4, 0}, {1, 'w', 0, 0, -1, 0, 4, 0}, {2, 'o', 0x685174, 0x4, 1, 0x1862, 1, 0}, {3, 'o', 0x685174, 0x69, 1, 0x1862, 4, 1},
		{4, 'o', 0x685174, 0x71, 1, 0x1862, 4, 1}, {5, 'o', 0x685174, 0x75, 1, 0x1862, 4, 0}, {6, 'o', 0x685174, 0x6d, 1, 0x1862, 4, 0}, {7, 'o', 0x685174, 0x89, 1, 0x1862, 4, 0},
		{8, 'o', 0x685174, 0x8d, 1, 0x1862, 4, 0}, {9, 'o', 0x685174, 0x91, 1, 0x1862, 4, 0}, {10, 'o', 0x685174, 0x95, 1, 0x1862, 4, 0}, {11, 'o', 0x685174, 0x99, 1, 0x1862, 4, 0},
		{12, 'o', 0x685174, 0x9d, 1, 0x1862, 4, 0}, {13, 'o', 0x685174, 0xa1, 1, 0x1862, 4, 0}, {14, 'o', 0x685174, 0xa5, 1, 0x1862, 4, 0}, {15, 'o', 0x685174, 0xa9, 1, 0x1862, 4, 0},
		{16, 'o', 0x685174, 0xad, 1, 0x1862, 4, 0}, {17, 'o', 0x685174, 0xb1, 1, 0x1862, 4, 0}, {18, 'o', 0x685174, 0xb5, 1, 0x1862, 4, 0}, {19, 'o', 0x685174, 0xb9, 1, 0x1862, 4, 0},
		{20, 'o', 0x685174, 0xbd, 1, 0x1862, 4, 0}, {21, 'o', 0x685174, 0xc1, 1, 0x1862, 4, 0}, {22, 'o', 0x685174, 0xc5, 1, 0x1862, 4, 0}, {23, 'o', 0x685174, 0xe5, 1, 0x1862, 4, 0},
		{24, 'o', 0x685174, 0xf9, 1, 0x1862, 4, 0}, {25, 'o', 0x685174, 0x10d, 1, 0x1862, 4, 0}, {26, 'o', 0x685174, 0x121, 1, 0x1862, 4, 0}, {27, 'o', 0x685174, 0x135, 1, 0x1862, 4, 0},
		{28, 'o', 0x685174, 0x149, 1, 0x1862, 4, 0}, {29, 'o', 0x685174, 0x15d, 1, 0x1862, 4, 0}, {30, 'o', 0x685174, 0x177c, 1, 0x1862, 1, 1}, {31, 'o', 0x685174, 0x177d, 1, 0x1862, 1, 1},
		{32, 'o', 0x685174, 0x177e, 1, 0x1862, 1, 1}, {33, 'o', 0x685174, 0x177f, 1, 0x1862, 1, 1}, {34, 'o', 0x685174, 0x1780, 1, 0x1862, 1, 1}, {35, 'o', 0x685174, 0x1781, 1, 0x1862, 1, 1},
		{36, 'o', 0x685174, 0x1782, 1, 0x1862, 1, 1}, {37, 'o', 0x685174, 0x1770, 1, 0x1862, 1, 1}, {38, 'o', 0x685174, 0x1776, 1, 0x1862, 1, 1}, {39, 'o', 0x685174, 0x1777, 1, 0x1862, 1, 1},
		{40, 'o', 0x685174, 0x17ac, 1, 0x1862, 1, 1}, {41, 'o', 0x685174, 0x17ad, 1, 0x1862, 1, 1}, {42, 'o', 0x685174, 0x17ae, 1, 0x1862, 1, 1}, {43, 'o', 0x685174, 0x17af, 1, 0x1862, 1, 1},
		{44, 'o', 0x685174, 0x17b0, 1, 0x1862, 1, 1}, {45, 'o', 0x685174, 0x17b1, 1, 0x1862, 1, 1}, {46, 'o', 0x685174, 0x1783, 1, 0x1862, 1, 1}, {47, 'o', 0x685174, 0x1784, 1, 0x1862, 1, 1},
		{48, 'o', 0x685174, 0x1785, 1, 0x1862, 1, 1}, {49, 'o', 0x685174, 0x1786, 1, 0x1862, 1, 1}, {50, 'o', 0x685174, 0x1787, 1, 0x1862, 1, 1}, {51, 'o', 0x685174, 0x1788, 1, 0x1862, 1, 1},
		{52, 'o', 0x685174, 0x1789, 1, 0x1862, 1, 1}, {53, 'o', 0x685174, 0x178a, 1, 0x1862, 1, 1}, {54, 'o', 0x685174, 0x178b, 1, 0x1862, 1, 1}, {55, 'o', 0x685174, 0x178c, 1, 0x1862, 1, 1},
		{56, 'o', 0x685174, 0x178d, 1, 0x1862, 1, 1}, {57, 'o', 0x685174, 0x178e, 1, 0x1862, 1, 1}, {58, 'o', 0x685174, 0x178f, 1, 0x1862, 1, 1}, {59, 'o', 0x685174, 0x1790, 1, 0x1862, 1, 1},
		{60, 'o', 0x685174, 0x1791, 1, 0x1862, 1, 1}, {61, 'o', 0x685174, 0x1792, 1, 0x1862, 1, 1}, {62, 'o', 0x685174, 0x1793, 1, 0x1862, 1, 1}, {63, 'o', 0x685174, 0x1794, 1, 0x1862, 1, 1},
		{64, 'o', 0x685174, 0x1795, 1, 0x1862, 1, 1}, {65, 'o', 0x685174, 0x1796, 1, 0x1862, 1, 1}, {66, 'o', 0x685174, 0x1797, 1, 0x1862, 1, 1}, {67, 'o', 0x685174, 0x1798, 1, 0x1862, 1, 1},
		{68, 'o', 0x685174, 0x1799, 1, 0x1862, 1, 1}, {69, 'o', 0x685174, 0x179a, 1, 0x1862, 1, 1}, {70, 'o', 0x685174, 0x179b, 1, 0x1862, 1, 1}, {71, 'o', 0x685174, 0x179c, 1, 0x1862, 1, 1},
		{72, 'o', 0x685174, 0x179d, 1, 0x1862, 1, 1}, {73, 'o', 0x685174, 0x179e, 1, 0x1862, 1, 1}, {74, 'o', 0x685174, 0x179f, 1, 0x1862, 1, 1}, {75, 'o', 0x685174, 0x17a0, 1, 0x1862, 1, 1},
		{76, 'o', 0x685174, 0x17a1, 1, 0x1862, 1, 1}, {77, 'o', 0x685174, 0x17a2, 1, 0x1862, 1, 1}, {78, 'o', 0x685174, 0x17a3, 1, 0x1862, 1, 1}, {79, 'o', 0x685174, 0x17a4, 1, 0x1862, 1, 1},
		{80, 'o', 0x685174, 0x17a5, 1, 0x1862, 1, 1}, {81, 'o', 0x685174, 0x17a6, 1, 0x1862, 1, 1}, {82, 'o', 0x685174, 0x17a7, 1, 0x1862, 1, 1}, {83, 'o', 0x685174, 0x17a8, 1, 0x1862, 1, 1},
		{84, 'o', 0x685174, 0x17a9, 1, 0x1862, 1, 1}, {85, 'o', 0x685174, 0x17aa, 1, 0x1862, 1, 1}, {86, 'o', 0x685174, 0x17ab, 1, 0x1862, 1, 1}, {87, 'o', 0x685174, 0x17b2, 1, 0x1862, 1, 0},
		{88, 'o', 0x685174, 0x17b3, 1, 0x1862, 1, 0}, {89, 'o', 0x685174, 0x17b4, 1, 0x1862, 1, 0}, {90, 'o', 0x685174, 0x71b, 1, 0x1862, 1, 0}, {91, 'o', 0x685174, 0x721, 1, 0x1862, 4, 0},
		{92, 'o', 0x685174, 0x6f5, 1, 0x1862, 1, 0}, {93, 'o', 0x685174, 0x6fb, 1, 0x1862, 4, 0}, {94, 'o', 0x685174, 0x767, 1, 0x1862, 1, 0}, {95, 'o', 0x685174, 0x76d, 1, 0x1862, 4, 0},
		{96, 'o', 0x685174, 0xa79, 1, 0x1862, 4, 0}, {97, 'o', 0x685174, 0x19f, 1, 0x1862, 4, 0}, {98, 'o', 0x685174, 0x1c5, 1, 0x1862, 4, 0}, {99, 'o', 0x685174, 0x1eb, 1, 0x1862, 4, 0},
		{100, 'o', 0x685174, 0x211, 1, 0x1862, 4, 0}, {101, 'o', 0x685174, 0x237, 1, 0x1862, 4, 0}, {102, 'o', 0x685174, 0x25d, 1, 0x1862, 4, 0}, {103, 'o', 0x685174, 0x283, 1, 0x1862, 4, 0},
		{104, 'o', 0x685174, 0x2a9, 1, 0x1862, 4, 0}, {105, 'o', 0x685174, 0x2cf, 1, 0x1862, 4, 0}, {106, 'o', 0x685174, 0x2f5, 1, 0x1862, 4, 0}, {107, 'o', 0x685174, 0x31b, 1, 0x1862, 4, 0},
		{108, 'o', 0x685174, 0x341, 1, 0x1862, 4, 0}, {109, 'o', 0x685174, 0x367, 1, 0x1862, 4, 0}, {110, 'o', 0x685174, 0x38d, 1, 0x1862, 4, 0}, {111, 'o', 0x685174, 0x3b3, 1, 0x1862, 4, 0},
		{112, 'o', 0x685174, 0x3d9, 1, 0x1862, 4, 0}, {113, 'o', 0x685174, 0x3ff, 1, 0x1862, 4, 0}, {114, 'o', 0x685174, 0x425, 1, 0x1862, 4, 0}, {115, 'o', 0x685174, 0x44b, 1, 0x1862, 4, 0},
		{116, 'o', 0x685174, 0x471, 1, 0x1862, 4, 0}, {117, 'o', 0x685174, 0x497, 1, 0x1862, 4, 0}, {118, 'o', 0x685174, 0x4bd, 1, 0x1862, 4, 0}, {119, 'o', 0x685174, 0x4e3, 1, 0x1862, 4, 0},
		{120, 'o', 0x685174, 0x509, 1, 0x1862, 4, 0}, {121, 'o', 0x685174, 0x52f, 1, 0x1862, 4, 0}, {122, 'o', 0x685174, 0x555, 1, 0x1862, 4, 0}, {123, 'o', 0x685174, 0x57b, 1, 0x1862, 4, 0},
		{124, 'o', 0x685174, 0x5a1, 1, 0x1862, 4, 0}, {125, 'o', 0x685174, 0x5c7, 1, 0x1862, 4, 0}, {126, 'o', 0x685174, 0x5ed, 1, 0x1862, 4, 0}, {127, 'o', 0x685174, 0x613, 1, 0x1862, 4, 0},
		{128, 'o', 0x685174, 0x639, 1, 0x1862, 4, 0}, {129, 'o', 0x685174, 0x65f, 1, 0x1862, 4, 0}, {130, 'o', 0x685174, 0x685, 1, 0x1862, 4, 0}, {131, 'o', 0x685174, 0x6ab, 1, 0x1862, 4, 0},
		{132, 'o', 0x685174, 0x6d1, 1, 0x1862, 4, 0}, {133, 'o', 0x685174, 0x6f7, 1, 0x1862, 4, 0}, {134, 'o', 0x685174, 0x71d, 1, 0x1862, 4, 0}, {135, 'o', 0x685174, 0x743, 1, 0x1862, 4, 0},
		{136, 'o', 0x685174, 0x769, 1, 0x1862, 4, 0}, {137, 'o', 0x685174, 0x78f, 1, 0x1862, 4, 0}, {138, 'o', 0x685174, 0xbcd, 1, 0x1862, 4, 0}, {139, 'o', 0x685174, 0xbd1, 1, 0x1862, 4, 0},
		{140, 'o', 0x685174, 0xbd5, 1, 0x1862, 4, 0}, {141, 'o', 0x685174, 0xbd9, 1, 0x1862, 4, 0}, {142, 'o', 0x685174, 0xbdd, 1, 0x1862, 4, 0}, {143, 'o', 0x685174, 0xbe1, 1, 0x1862, 4, 0},
		{144, 'o', 0x685174, 0xbe5, 1, 0x1862, 4, 0}, {145, 'o', 0x685174, 0xbe9, 1, 0x1862, 4, 0}, {146, 'o', 0x685174, 0xbed, 1, 0x1862, 4, 0}, {147, 'o', 0x685174, 0xbf1, 1, 0x1862, 4, 0},
		{148, 'o', 0x685174, 0xbf5, 1, 0x1862, 4, 0}, {149, 'o', 0x685174, 0xbf9, 1, 0x1862, 4, 0}, {150, 'o', 0x685174, 0xbfd, 1, 0x1862, 4, 0}, {151, 'o', 0x685174, 0xc01, 1, 0x1862, 4, 0},
		{152, 'o', 0x685174, 0xc05, 1, 0x1862, 4, 0}, {153, 'o', 0x685174, 0xc09, 1, 0x1862, 4, 0}, {154, 'o', 0x685174, 0xc0d, 1, 0x1862, 4, 0}, {155, 'o', 0x685174, 0xc11, 1, 0x1862, 4, 0},
		{156, 'o', 0x685174, 0xc15, 1, 0x1862, 4, 0}, {157, 'o', 0x685174, 0xc19, 1, 0x1862, 4, 0}, {158, 'o', 0x685174, 0xc1d, 1, 0x1862, 4, 0}, {159, 'o', 0x685174, 0xc21, 1, 0x1862, 4, 0},
		{160, 'o', 0x685174, 0xc25, 1, 0x1862, 4, 0}, {161, 'o', 0x685174, 0xc29, 1, 0x1862, 4, 0}, {162, 'o', 0x685174, 0xc2d, 1, 0x1862, 4, 0}, {163, 'o', 0x685174, 0xc31, 1, 0x1862, 4, 0},
		{164, 'o', 0x685174, 0xc35, 1, 0x1862, 4, 0}, {165, 'o', 0x685174, 0xc39, 1, 0x1862, 4, 0}, {166, 'o', 0x685174, 0xc3d, 1, 0x1862, 4, 0}, {167, 'o', 0x685174, 0xc41, 1, 0x1862, 4, 0},
		{168, 'o', 0x685174, 0xc45, 1, 0x1862, 4, 0}, {169, 'o', 0x685174, 0xc49, 1, 0x1862, 4, 0}, {170, 'o', 0x685174, 0xc4d, 1, 0x1862, 4, 0}, {171, 'o', 0x685174, 0xc51, 1, 0x1862, 4, 0},
		{172, 'o', 0x685174, 0xc55, 1, 0x1862, 4, 0}, {173, 'o', 0x685174, 0xc59, 1, 0x1862, 4, 0}, {174, 'o', 0x685174, 0xc5d, 1, 0x1862, 4, 0}, {175, 'o', 0x685174, 0xc61, 1, 0x1862, 4, 0},
		{176, 'o', 0x685174, 0xc65, 1, 0x1862, 4, 0}, {177, 'o', 0x685174, 0xc69, 1, 0x1862, 4, 0}, {178, 'o', 0x685174, 0xc6d, 1, 0x1862, 4, 0}, {179, 'o', 0x685174, 0xc71, 1, 0x1862, 4, 0},
		{180, 'o', 0x685174, 0xc75, 1, 0x1862, 4, 0}, {181, 'o', 0x685174, 0xc79, 1, 0x1862, 4, 0}, {182, 'o', 0x685174, 0xc7d, 1, 0x1862, 4, 0}, {183, 'o', 0x685174, 0xc81, 1, 0x1862, 4, 0},
		{184, 'o', 0x685174, 0xc85, 1, 0x1862, 4, 0}, {185, 'o', 0x685174, 0xc89, 1, 0x1862, 4, 0}, {186, 'o', 0x685174, 0xc8d, 1, 0x1862, 4, 0}, {187, 'o', 0x685174, 0xc91, 1, 0x1862, 4, 0},
		{188, 'o', 0x685174, 0xc95, 1, 0x1862, 4, 0}, {189, 'o', 0x685174, 0xc99, 1, 0x1862, 4, 0}, {190, 'o', 0x685174, 0xc9d, 1, 0x1862, 4, 0}, {191, 'o', 0x685174, 0xca1, 1, 0x1862, 4, 0},
		{192, 'o', 0x685174, 0xca5, 1, 0x1862, 4, 0}, {193, 'o', 0x685174, 0xca9, 1, 0x1862, 4, 0}, {194, 'o', 0x685174, 0xcad, 1, 0x1862, 4, 0}, {195, 'o', 0x685174, 0xcb1, 1, 0x1862, 4, 0},
		{196, 'o', 0x685174, 0xcb5, 1, 0x1862, 4, 0}, {197, 'o', 0x685174, 0xcb9, 1, 0x1862, 4, 0}, {198, 'o', 0x685174, 0xcbd, 1, 0x1862, 4, 0}, {199, 'o', 0x685174, 0xcc1, 1, 0x1862, 4, 0},
		{200, 'o', 0x685174, 0xcc5, 1, 0x1862, 4, 0}, {201, 'o', 0x685174, 0xcc9, 1, 0x1862, 4, 0}, {202, 'o', 0x685174, 0xccd, 1, 0x1862, 4, 0}, {203, 'o', 0x685174, 0xcd1, 1, 0x1862, 4, 0},
		{204, 'o', 0x685174, 0xcd5, 1, 0x1862, 4, 0}, {205, 'o', 0x685174, 0xcd9, 1, 0x1862, 4, 0}, {206, 'o', 0x685174, 0xcdd, 1, 0x1862, 4, 0}, {207, 'o', 0x685174, 0xce1, 1, 0x1862, 4, 0},
		{208, 'o', 0x685174, 0xce5, 1, 0x1862, 4, 0}, {209, 'o', 0x685174, 0xce9, 1, 0x1862, 4, 0}, {210, 'o', 0x685174, 0xced, 1, 0x1862, 4, 0}, {211, 'o', 0x685174, 0xcf1, 1, 0x1862, 4, 0},
		{212, 'o', 0x685174, 0xcf5, 1, 0x1862, 4, 0}, {213, 'o', 0x685174, 0xcf9, 1, 0x1862, 4, 0}, {214, 'o', 0x685174, 0xcfd, 1, 0x1862, 4, 0}, {215, 'o', 0x685174, 0xd01, 1, 0x1862, 4, 0},
		{216, 'o', 0x685174, 0xd05, 1, 0x1862, 4, 0}, {217, 'o', 0x685174, 0xd09, 1, 0x1862, 4, 0}, {218, 'o', 0x685174, 0xd0d, 1, 0x1862, 4, 0}, {219, 'o', 0x685174, 0xd11, 1, 0x1862, 4, 0},
		{220, 'o', 0x685174, 0xd15, 1, 0x1862, 4, 0}, {221, 'o', 0x685174, 0xd19, 1, 0x1862, 4, 0}, {222, 'o', 0x685174, 0xd1d, 1, 0x1862, 4, 0}, {223, 'o', 0x685174, 0xd21, 1, 0x1862, 4, 0},
		{224, 'o', 0x685174, 0xd25, 1, 0x1862, 4, 0}, {225, 'o', 0x685174, 0xd29, 1, 0x1862, 4, 0}, {226, 'o', 0x685174, 0xd2d, 1, 0x1862, 4, 0}, {227, 'o', 0x685174, 0xd31, 1, 0x1862, 4, 0},
		{228, 'o', 0x685174, 0xd35, 1, 0x1862, 4, 0}, {229, 'o', 0x685174, 0xd39, 1, 0x1862, 4, 0}, {230, 'o', 0x685174, 0xd3d, 1, 0x1862, 4, 0}, {231, 'o', 0x685174, 0xd41, 1, 0x1862, 4, 0},
		{232, 'o', 0x685174, 0xd45, 1, 0x1862, 4, 0}, {233, 'o', 0x685174, 0xd49, 1, 0x1862, 4, 0}, {234, 'o', 0x685174, 0xd4d, 1, 0x1862, 4, 0}, {235, 'o', 0x685174, 0xd51, 1, 0x1862, 4, 0},
		{236, 'o', 0x685174, 0xd55, 1, 0x1862, 4, 0}, {237, 'o', 0x685174, 0xd59, 1, 0x1862, 4, 0}, {238, 'o', 0x685174, 0xd5d, 1, 0x1862, 4, 0}, {239, 'o', 0x685174, 0xd61, 1, 0x1862, 4, 0},
		{240, 'o', 0x685174, 0xd65, 1, 0x1862, 4, 0}, {241, 'o', 0x685174, 0xd69, 1, 0x1862, 4, 0}, {242, 'o', 0x685174, 0xd6d, 1, 0x1862, 4, 0}, {243, 'o', 0x685174, 0xd71, 1, 0x1862, 4, 0},
		{244, 'o', 0x685174, 0xd75, 1, 0x1862, 4, 0}, {245, 'o', 0x685174, 0xd79, 1, 0x1862, 4, 0}, {246, 'o', 0x685174, 0xd7d, 1, 0x1862, 4, 0}, {247, 'o', 0x685174, 0xd81, 1, 0x1862, 4, 0},
		{248, 'o', 0x685174, 0xd85, 1, 0x1862, 4, 0}, {249, 'o', 0x685174, 0xd89, 1, 0x1862, 4, 0}, {250, 'o', 0x685174, 0xd8d, 1, 0x1862, 4, 0}, {251, 'o', 0x685174, 0xd91, 1, 0x1862, 4, 0},
		{252, 'o', 0x685174, 0xb15, 1, 0x1862, 4, 1}, {253, 'o', 0x685174, 0xb0d, 1, 0x1862, 4, 0}, {254, 'o', 0x685174, 0xb11, 1, 0x1862, 4, 0}, {255, 'o', 0x685174, 0xb19, 1, 0x1862, 4, 0},
		{256, 'o', 0x685174, 0xb1d, 1, 0x1862, 4, 0}, {257, 'o', 0x685174, 0xb21, 1, 0x1862, 4, 0}, {258, 'o', 0x685174, 0xb25, 1, 0x1862, 4, 0}, {259, 'o', 0x685174, 0xb29, 1, 0x1862, 4, 0},
		{260, 'o', 0x685174, 0xb2d, 1, 0x1862, 4, 0}, {261, 'o', 0x685174, 0xb31, 1, 0x1862, 4, 0}, {262, 'o', 0x685174, 0xb35, 1, 0x1862, 4, 0}, {263, 'o', 0x685174, 0xb39, 1, 0x1862, 4, 0},
		{264, 'o', 0x685174, 0xb45, 1, 0x1862, 4, 0}, {265, 'o', 0x685174, 0xb49, 1, 0x1862, 4, 0}, {266, 'o', 0x685174, 0xb4d, 1, 0x1862, 4, 0}, {267, 'o', 0x685174, 0xb51, 1, 0x1862, 4, 0},
		{268, 'o', 0x685174, 0xb55, 1, 0x1862, 4, 0}, {269, 'o', 0x685174, 0xb59, 1, 0x1862, 4, 0}, {270, 'o', 0x685174, 0xbb9, 1, 0x1862, 4, 0}, {271, 'o', 0x685174, 0xbbd, 1, 0x1862, 4, 0},
		{272, 'o', 0x685174, 0xbc5, 1, 0x1862, 4, 0}, {273, 'g', 0x68517c, 0, -1, 0, 1, 0}, {274, 'g', 0x68ec78, 0, -1, 0, 4, 0}, {275, 'g', 0x68c558, 0, -1, 0, 4, 0},
		{276, 'g', 0x68edcc, 0, -1, 0, 4, 0}, {277, 'o', 0x69c0f4, 0xf8, -1, 0x0, 4, 0}, {278, 'o', 0x69c0f4, 0x2f4, -1, 0x0, 4, 0},
	}, []ctrsLink{
		{0, 'o', 0x685174, 0x89, 1, 0x1862, 4, 0}, {1, 'o', 0x685174, 0x8d, 1, 0x1862, 4, 0}, {2, 'o', 0x685174, 0x91, 1, 0x1862, 4, 0}, {3, 'o', 0x685174, 0x95, 1, 0x1862, 4, 0},
		{4, 'o', 0x685174, 0x99, 1, 0x1862, 4, 0}, {5, 'o', 0x685174, 0x9d, 1, 0x1862, 4, 0}, {6, 'o', 0x685174, 0xa1, 1, 0x1862, 4, 0}, {7, 'o', 0x685174, 0xa5, 1, 0x1862, 4, 0},
		{8, 'o', 0x685174, 0xa9, 1, 0x1862, 4, 0}, {9, 'o', 0x685174, 0xad, 1, 0x1862, 4, 0}, {10, 'o', 0x685174, 0xb1, 1, 0x1862, 4, 0}, {11, 'o', 0x685174, 0xb5, 1, 0x1862, 4, 0},
		{12, 'o', 0x685174, 0xb9, 1, 0x1862, 4, 0}, {13, 'o', 0x685174, 0xbd, 1, 0x1862, 4, 0}, {14, 'o', 0x685174, 0xc1, 1, 0x1862, 4, 0}, {15, 'o', 0x685174, 0xc5, 1, 0x1862, 4, 0},
		{16, 'o', 0x685174, 0xb0d, 1, 0x1862, 4, 0}, {17, 'o', 0x685174, 0xb11, 1, 0x1862, 4, 0}, {18, 'o', 0x685174, 0xb19, 1, 0x1862, 4, 0}, {19, 'o', 0x685174, 0xb1d, 1, 0x1862, 4, 0},
		{20, 'o', 0x685174, 0xb25, 1, 0x1862, 4, 0}, {21, 'o', 0x685174, 0xb29, 1, 0x1862, 4, 0}, {22, 'o', 0x685174, 0xb2d, 1, 0x1862, 4, 0}, {23, 'o', 0x685174, 0xb31, 1, 0x1862, 4, 0},
		{24, 'o', 0x685174, 0xb35, 1, 0x1862, 4, 0}, {25, 'o', 0x685174, 0xb39, 1, 0x1862, 4, 0}, {26, 'o', 0x685174, 0xb45, 1, 0x1862, 4, 0}, {27, 'o', 0x685174, 0xb49, 1, 0x1862, 4, 0},
		{28, 'o', 0x685174, 0xb4d, 1, 0x1862, 4, 0}, {29, 'o', 0x685174, 0xb51, 1, 0x1862, 4, 0}, {30, 'o', 0x685174, 0xb55, 1, 0x1862, 4, 0}, {31, 'o', 0x685174, 0xb59, 1, 0x1862, 4, 0},
		{32, 'o', 0x685174, 0xbb9, 1, 0x1862, 4, 0}, {33, 'o', 0x685174, 0xbbd, 1, 0x1862, 4, 0}, {34, 'o', 0x685174, 0xbc5, 1, 0x1862, 4, 0}, {35, 'o', 0x69c0f4, 0xf8, -1, 0x0, 4, 0},
		{36, 'o', 0x69c0f4, 0x2f4, -1, 0x0, 4, 0}, {37, 'o', 0x685174, 0xe9, 1, 0x1862, 4, 0}, {38, 'o', 0x685174, 0xfd, 1, 0x1862, 4, 0}, {39, 'o', 0x685174, 0x111, 1, 0x1862, 4, 0},
		{40, 'o', 0x685174, 0x125, 1, 0x1862, 4, 0}, {41, 'o', 0x685174, 0x139, 1, 0x1862, 4, 0}, {42, 'o', 0x685174, 0x14d, 1, 0x1862, 4, 0}, {43, 'o', 0x685174, 0x161, 1, 0x1862, 4, 0},
		{44, 'o', 0x685174, 0x1a7, 1, 0x1862, 4, 0}, {45, 'o', 0x685174, 0x1cd, 1, 0x1862, 4, 0}, {46, 'o', 0x685174, 0x1f3, 1, 0x1862, 4, 0}, {47, 'o', 0x685174, 0x219, 1, 0x1862, 4, 0},
		{48, 'o', 0x685174, 0x23f, 1, 0x1862, 4, 0}, {49, 'o', 0x685174, 0x265, 1, 0x1862, 4, 0}, {50, 'o', 0x685174, 0x28b, 1, 0x1862, 4, 0}, {51, 'o', 0x685174, 0x2b1, 1, 0x1862, 4, 0},
		{52, 'o', 0x685174, 0x2d7, 1, 0x1862, 4, 0}, {53, 'o', 0x685174, 0x2fd, 1, 0x1862, 4, 0}, {54, 'o', 0x685174, 0x323, 1, 0x1862, 4, 0}, {55, 'o', 0x685174, 0x349, 1, 0x1862, 4, 0},
		{56, 'o', 0x685174, 0x36f, 1, 0x1862, 4, 0}, {57, 'o', 0x685174, 0x395, 1, 0x1862, 4, 0}, {58, 'o', 0x685174, 0x3bb, 1, 0x1862, 4, 0}, {59, 'o', 0x685174, 0x3e1, 1, 0x1862, 4, 0},
		{60, 'o', 0x685174, 0x407, 1, 0x1862, 4, 0}, {61, 'o', 0x685174, 0x42d, 1, 0x1862, 4, 0}, {62, 'o', 0x685174, 0x453, 1, 0x1862, 4, 0}, {63, 'o', 0x685174, 0x479, 1, 0x1862, 4, 0},
		{64, 'o', 0x685174, 0x49f, 1, 0x1862, 4, 0}, {65, 'o', 0x685174, 0x4c5, 1, 0x1862, 4, 0}, {66, 'o', 0x685174, 0x4eb, 1, 0x1862, 4, 0}, {67, 'o', 0x685174, 0x511, 1, 0x1862, 4, 0},
		{68, 'o', 0x685174, 0x537, 1, 0x1862, 4, 0}, {69, 'o', 0x685174, 0x55d, 1, 0x1862, 4, 0}, {70, 'o', 0x685174, 0x583, 1, 0x1862, 4, 0}, {71, 'o', 0x685174, 0x5a9, 1, 0x1862, 4, 0},
		{72, 'o', 0x685174, 0x5cf, 1, 0x1862, 4, 0}, {73, 'o', 0x685174, 0x5f5, 1, 0x1862, 4, 0}, {74, 'o', 0x685174, 0x61b, 1, 0x1862, 4, 0}, {75, 'o', 0x685174, 0x641, 1, 0x1862, 4, 0},
		{76, 'o', 0x685174, 0x667, 1, 0x1862, 4, 0}, {77, 'o', 0x685174, 0x68d, 1, 0x1862, 4, 0}, {78, 'o', 0x685174, 0x6b3, 1, 0x1862, 4, 0}, {79, 'o', 0x685174, 0x6d9, 1, 0x1862, 4, 0},
		{80, 'o', 0x685174, 0x6ff, 1, 0x1862, 4, 0}, {81, 'o', 0x685174, 0x725, 1, 0x1862, 4, 0}, {82, 'o', 0x685174, 0x74b, 1, 0x1862, 4, 0}, {83, 'o', 0x685174, 0x771, 1, 0x1862, 4, 0},
		{84, 'o', 0x685174, 0x797, 1, 0x1862, 4, 0}, {85, 'o', 0x685174, 0x1a3, 1, 0x1862, 4, 0}, {86, 'o', 0x685174, 0x1c9, 1, 0x1862, 4, 0}, {87, 'o', 0x685174, 0x1ef, 1, 0x1862, 4, 0},
		{88, 'o', 0x685174, 0x215, 1, 0x1862, 4, 0}, {89, 'o', 0x685174, 0x23b, 1, 0x1862, 4, 0}, {90, 'o', 0x685174, 0x261, 1, 0x1862, 4, 0}, {91, 'o', 0x685174, 0x287, 1, 0x1862, 4, 0},
		{92, 'o', 0x685174, 0x2ad, 1, 0x1862, 4, 0}, {93, 'o', 0x685174, 0x2d3, 1, 0x1862, 4, 0}, {94, 'o', 0x685174, 0x2f9, 1, 0x1862, 4, 0}, {95, 'o', 0x685174, 0x31f, 1, 0x1862, 4, 0},
		{96, 'o', 0x685174, 0x345, 1, 0x1862, 4, 0}, {97, 'o', 0x685174, 0x36b, 1, 0x1862, 4, 0}, {98, 'o', 0x685174, 0x391, 1, 0x1862, 4, 0}, {99, 'o', 0x685174, 0x3b7, 1, 0x1862, 4, 0},
		{100, 'o', 0x685174, 0x3dd, 1, 0x1862, 4, 0}, {101, 'o', 0x685174, 0x403, 1, 0x1862, 4, 0}, {102, 'o', 0x685174, 0x429, 1, 0x1862, 4, 0}, {103, 'o', 0x685174, 0x44f, 1, 0x1862, 4, 0},
		{104, 'o', 0x685174, 0x475, 1, 0x1862, 4, 0}, {105, 'o', 0x685174, 0x49b, 1, 0x1862, 4, 0}, {106, 'o', 0x685174, 0x4c1, 1, 0x1862, 4, 0}, {107, 'o', 0x685174, 0x4e7, 1, 0x1862, 4, 0},
		{108, 'o', 0x685174, 0x50d, 1, 0x1862, 4, 0}, {109, 'o', 0x685174, 0x533, 1, 0x1862, 4, 0}, {110, 'o', 0x685174, 0x559, 1, 0x1862, 4, 0}, {111, 'o', 0x685174, 0x57f, 1, 0x1862, 4, 0},
		{112, 'o', 0x685174, 0x5a5, 1, 0x1862, 4, 0}, {113, 'o', 0x685174, 0x5cb, 1, 0x1862, 4, 0}, {114, 'o', 0x685174, 0x5f1, 1, 0x1862, 4, 0}, {115, 'o', 0x685174, 0x617, 1, 0x1862, 4, 0},
		{116, 'o', 0x685174, 0x63d, 1, 0x1862, 4, 0}, {117, 'o', 0x685174, 0x663, 1, 0x1862, 4, 0}, {118, 'o', 0x685174, 0x689, 1, 0x1862, 4, 0}, {119, 'o', 0x685174, 0x6af, 1, 0x1862, 4, 0},
		{120, 'o', 0x685174, 0x6d5, 1, 0x1862, 4, 0}, {121, 'o', 0x685174, 0x6fb, 1, 0x1862, 4, 0}, {122, 'o', 0x685174, 0x721, 1, 0x1862, 4, 0}, {123, 'o', 0x685174, 0x747, 1, 0x1862, 4, 0},
		{124, 'o', 0x685174, 0x76d, 1, 0x1862, 4, 0}, {125, 'o', 0x685174, 0x793, 1, 0x1862, 4, 0}, {126, 'o', 0x685174, 0xe5d, 1, 0x1862, 4, 0}, {127, 'o', 0x685174, 0xe61, 1, 0x1862, 4, 0},
		{128, 'o', 0x685174, 0xe65, 1, 0x1862, 4, 0}, {129, 'o', 0x685174, 0xe69, 1, 0x1862, 4, 0}, {130, 'o', 0x685174, 0xe6d, 1, 0x1862, 4, 0}, {131, 'o', 0x685174, 0xe71, 1, 0x1862, 4, 0},
		{132, 'o', 0x685174, 0xb69, 1, 0x1862, 4, 0}, {133, 'o', 0x685174, 0xb71, 1, 0x1862, 4, 0}, {134, 'o', 0x685174, 0xb79, 1, 0x1862, 4, 0}, {135, 'o', 0x685174, 0xb81, 1, 0x1862, 4, 0},
		{136, 'o', 0x685174, 0xb89, 1, 0x1862, 4, 0}, {137, 'o', 0x685174, 0xb91, 1, 0x1862, 4, 0}, {138, 'o', 0x685174, 0xb99, 1, 0x1862, 4, 0}, {139, 'o', 0x685174, 0xba1, 1, 0x1862, 4, 0},
		{140, 'o', 0x685174, 0x169e, 1, 0x1862, 4, 0}, {141, 'o', 0x685174, 0xe7d, 1, 0x1862, 4, 0}, {142, 'o', 0x685174, 0xed9, 1, 0x1862, 4, 0}, {143, 'o', 0x685174, 0xedd, 1, 0x1862, 4, 0},
		{144, 'o', 0x685174, 0xeed, 1, 0x1862, 4, 0}, {145, 'o', 0x685174, 0xefd, 1, 0x1862, 4, 0}, {146, 'o', 0x685174, 0xf0d, 1, 0x1862, 4, 0}, {147, 'o', 0x685174, 0xf1d, 1, 0x1862, 4, 0},
		{148, 'o', 0x685174, 0xf2d, 1, 0x1862, 4, 0}, {149, 'o', 0x685174, 0xee1, 1, 0x1862, 4, 0}, {150, 'o', 0x685174, 0xef1, 1, 0x1862, 4, 0}, {151, 'o', 0x685174, 0xf01, 1, 0x1862, 4, 0},
		{152, 'o', 0x685174, 0xf11, 1, 0x1862, 4, 0}, {153, 'o', 0x685174, 0xf21, 1, 0x1862, 4, 0}, {154, 'o', 0x685174, 0xf31, 1, 0x1862, 4, 0}, {155, 'o', 0x685174, 0xbc1, 1, 0x1862, 4, 0},
		{156, 'o', 0x685174, 0xbc9, 1, 0x1862, 4, 0}, {157, 'o', 0x685174, 0xe81, 1, 0x1862, 4, 0},
	}},
}

func characterTurnRecoveryAndStatusNode(in []int64, r Rand) []int64 { return ctrsRun(in, r, true) }

func init() {
	Nodes["00503100"] = Node{Addr: "00503100", In: ctrsInNames, Out: ctrsOutNames,
		F:   characterTurnRecoveryAndStatusNode,
		Emu: func(in []int64, r Rand) []int64 { return ctrsRun(in, r, false) }}
}
