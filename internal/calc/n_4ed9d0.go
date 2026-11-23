package calc

import (
	"fmt"
	"math"
	"math/big"
)

// rebuild_character_rule_abilities 0x4ed9d0 (re/emu/cases/004ed9d0.json): recompute the
// derived rule fields of one party character record ch (cdecl, one argument). A record with
// class >= 15, race >= 16 or +0x75 == -1 is left alone. Otherwise the steps are, in order:
// clamp the level +0x89 and the 15 class levels +0x8d.. to 50; effective attributes
// +0xe9+20i (5539e0) and the 41 skill values +0x1a3+0x26k with the favoured-skill bonus
// (553a60), each step followed by the equipment check 0x51d960; hit-point style total and
// stamina counters +0xb0d/+0xb11 (52a2f0), +0xb19/+0xb1d/+0x169e (52a3e0), the six school
// pools +0xb25/+0xb45/+0xe5d and the school maximum +0xe7d (52a500, 52a540); the armour
// classes (4ee9d0); the float multipliers +0xb69, +0xb71, +0xb79+8r (502b50); the damage
// reduction +0xed9; the resistance rows +0xedd/+0xee1 (551a60); the carry capacity +0xbc5
// (x2/3 with ability 0x18), the carried weight +0xbb9 (4edc60), the shared weight +0xbbd
// (4edd20) and the load category +0xbc9; the base attack bonus +0xe81 (4ee000); the hand
// abilities (4ee220) and, for a character in use outside mode 3, the interface refresh (55ee30).
//
// Out is the record state the function writes plus the call count and arguments of the
// stubbed callees (every written field listed in rcraOutG, ui6+0xf8 and ui7+0xf4).
//
// Stubs (case "stubs"): 0x51d960 (D4, 48 calls, the return is ignored), 0x51bfd0 (D4, 20
// calls, the weights are the case inputs r51bfd0), 0x4edd20, 0x4ee220, 0x4ee9d0 (D4, their
// writes are not read afterwards; 0x4edd20 writes +0xbbd/+0xbc1/+0xbc9 which the node
// reads: the case input +0xbbd stands for them in Emu), 0x522fe0 (D4, return ignored) and
// 0x52e690 (排除). Native in the emulator and transliterated here: 5539e0, 553a60, 52a2f0,
// 52a3e0, 52a500, 52a540 (with 4f96a0 and the qsort comparator 517a10), 502b50, 551a60, 4edc60,
// 4ee000, 547940, 5479b0, 55ee30, 562a50, 517ec0. Every table is read from the player's exe
// at run time (n4Mem), nothing is embedded.
//
// Fixture caps (calc panics past them): the school index of a spell flagged 1 or 2 is
// 0..5 (the original would write outside the pool array); a zero level divisor in
// 52a3e0 (+0xb21 >= the computed rating, a fault in the original); 517ec0 asserts when the
// character is not in use or not one of the eight party records.
// generated tables (scratch gen.py): n4Grp{name, kind (g global, o arena, a stack arg), address, size, signed, count, step, index field, stride}
var rcraInG = []n4Grp{
	{"chPtr", 'a', 0x0, 4, false, 1, 4, "", 0x0},
	{"who", 'o', 0x20003ff0, 4, false, 1, 4, "", 0x0},
	{"inuse", 'o', 0x20004004, 1, false, 1, 1, "who", 0x1862},
	{"cls", 'o', 0x20004069, 4, true, 1, 4, "who", 0x1862},
	{"race", 'o', 0x20004071, 4, true, 1, 4, "who", 0x1862},
	{"f75", 'o', 0x20004075, 4, false, 1, 4, "who", 0x1862},
	{"f6d", 'o', 0x2000406d, 4, false, 1, 4, "who", 0x1862},
	{"lvl", 'o', 0x20004089, 4, false, 1, 4, "who", 0x1862},
	{"lvls", 'o', 0x2000408d, 4, false, 15, 4, "who", 0x1862},
	{"base", 'o', 0x200040e5, 4, false, 7, 20, "who", 0x1862},
	{"sb7", 'o', 0x2000577c, 1, true, 7, 1, "who", 0x1862},
	{"sb1770", 'o', 0x20005770, 1, true, 1, 1, "who", 0x1862},
	{"sb1776", 'o', 0x20005776, 1, true, 1, 1, "who", 0x1862},
	{"sb1777", 'o', 0x20005777, 1, true, 1, 1, "who", 0x1862},
	{"sb17ac", 'o', 0x200057ac, 1, true, 6, 1, "who", 0x1862},
	{"sbskill", 'o', 0x20005783, 1, true, 41, 1, "who", 0x1862},
	{"f17b2", 'o', 0x200057b2, 1, false, 1, 1, "who", 0x1862},
	{"f17b3", 'o', 0x200057b3, 1, false, 1, 1, "who", 0x1862},
	{"f17b4", 'o', 0x200057b4, 1, false, 1, 1, "who", 0x1862},
	{"f71b", 'o', 0x2000471b, 1, false, 1, 1, "who", 0x1862},
	{"f721", 'o', 0x20004721, 4, false, 1, 4, "who", 0x1862},
	{"f6f5", 'o', 0x200046f5, 1, false, 1, 1, "who", 0x1862},
	{"f6fb", 'o', 0x200046fb, 4, false, 1, 4, "who", 0x1862},
	{"f767", 'o', 0x20004767, 1, false, 1, 1, "who", 0x1862},
	{"f76d", 'o', 0x2000476d, 4, false, 1, 4, "who", 0x1862},
	{"fa79", 'o', 0x20004a79, 4, false, 1, 4, "who", 0x1862},
	{"skill", 'o', 0x2000419f, 4, false, 41, 38, "who", 0x1862},
	{"flag", 'o', 0x20004bcd, 4, false, 114, 4, "who", 0x1862},
	{"b15", 'o', 0x20004b15, 4, true, 1, 4, "who", 0x1862},
	{"b0d", 'o', 0x20004b0d, 4, false, 1, 4, "who", 0x1862},
	{"b11", 'o', 0x20004b11, 4, false, 1, 4, "who", 0x1862},
	{"b19", 'o', 0x20004b19, 4, false, 1, 4, "who", 0x1862},
	{"b1d", 'o', 0x20004b1d, 4, false, 1, 4, "who", 0x1862},
	{"b21", 'o', 0x20004b21, 4, false, 1, 4, "who", 0x1862},
	{"b25", 'o', 0x20004b25, 4, false, 6, 4, "who", 0x1862},
	{"b45", 'o', 0x20004b45, 4, false, 6, 4, "who", 0x1862},
	{"bb9", 'o', 0x20004bb9, 4, false, 1, 4, "who", 0x1862},
	{"bbd", 'o', 0x20004bbd, 4, false, 1, 4, "who", 0x1862},
	{"bc5", 'o', 0x20004bc5, 4, false, 1, 4, "who", 0x1862},
	{"g517c", 'g', 0x68517c, 1, false, 1, 1, "", 0x0},
	{"mode", 'g', 0x68ec78, 4, false, 1, 4, "", 0x0},
	{"g558", 'g', 0x68c558, 4, false, 1, 4, "", 0x0},
	{"g8edcc", 'g', 0x68edcc, 4, false, 1, 4, "", 0x0},
	{"ui6f8", 'o', 0x200110f8, 4, false, 1, 4, "", 0x0},
	{"ui7f4", 'o', 0x200112f4, 4, false, 1, 4, "", 0x0},
	{"splv", 'o', 0x20012049, 4, false, 114, 447, "", 0x0},
	{"spls", 'o', 0x20012133, 4, false, 114, 447, "", 0x0},
	{"r51bfd0", 'o', 0x20040000, 4, false, 20, 4, "", 0x0},
	{"fvid", 'o', 0x20050000, 4, true, 20, 4, "", 0x0},
	{"fvqty", 'o', 0x20050050, 4, false, 20, 4, "", 0x0},
	{"fvunit", 'o', 0x200500a0, 4, false, 20, 4, "", 0x0},
	{"fcomb", 'o', 0x200500f0, 4, false, 1, 4, "", 0x0},
	{"fother", 'o', 0x200500f4, 4, false, 1, 4, "", 0x0},
	{"fitems", 'o', 0x200500f8, 4, false, 1, 4, "", 0x0},
	{"fsitemzero", 'o', 0x200500fc, 4, true, 1, 4, "", 0x0},
	{"fsqtyzero", 'o', 0x20050100, 4, false, 1, 4, "", 0x0},
	{"fsitemone", 'o', 0x20050104, 4, true, 1, 4, "", 0x0},
	{"fsqtyone", 'o', 0x20050108, 4, false, 1, 4, "", 0x0},
	{"fuwzero", 'o', 0x2005010c, 4, false, 1, 4, "", 0x0},
	{"fuwone", 'o', 0x20050110, 4, false, 1, 4, "", 0x0},
	{"fpres", 'o', 0x20050114, 4, false, 8, 4, "", 0x0},
	{"fstat", 'o', 0x20050134, 4, false, 8, 4, "", 0x0},
	{"fpers", 'o', 0x20050154, 4, false, 8, 4, "", 0x0},
	{"fcap", 'o', 0x20050174, 4, false, 8, 4, "", 0x0},
}

var rcraOutG = []n4Grp{
	{"A.lvl", 'o', 0x20004089, 4, false, 1, 4, "who", 0x1862},
	{"A.lvls", 'o', 0x2000408d, 4, false, 15, 4, "who", 0x1862},
	{"A.b0d", 'o', 0x20004b0d, 4, false, 1, 4, "who", 0x1862},
	{"A.b11", 'o', 0x20004b11, 4, false, 1, 4, "who", 0x1862},
	{"A.b19", 'o', 0x20004b19, 4, false, 1, 4, "who", 0x1862},
	{"A.b1d", 'o', 0x20004b1d, 4, false, 1, 4, "who", 0x1862},
	{"A.b25", 'o', 0x20004b25, 4, false, 6, 4, "who", 0x1862},
	{"A.b45", 'o', 0x20004b45, 4, false, 6, 4, "who", 0x1862},
	{"A.bb9", 'o', 0x20004bb9, 4, false, 1, 4, "who", 0x1862},
	{"A.bbd", 'o', 0x20004bbd, 4, false, 1, 4, "who", 0x1862},
	{"A.bc5", 'o', 0x20004bc5, 4, false, 1, 4, "who", 0x1862},
	{"A.ui6f8", 'o', 0x200110f8, 4, false, 1, 4, "", 0x0},
	{"A.ui7f4", 'o', 0x200112f4, 4, false, 1, 4, "", 0x0},
	{"e9", 'o', 0x200040e9, 4, false, 7, 20, "who", 0x1862},
	{"s1a7", 'o', 0x200041a7, 4, false, 41, 38, "who", 0x1862},
	{"s1a3", 'o', 0x200041a3, 4, false, 41, 38, "who", 0x1862},
	{"e5d", 'o', 0x20004e5d, 4, false, 6, 4, "who", 0x1862},
	{"b69", 'o', 0x20004b69, 4, false, 1, 4, "who", 0x1862},
	{"b71", 'o', 0x20004b71, 4, false, 1, 4, "who", 0x1862},
	{"b79", 'o', 0x20004b79, 4, false, 6, 8, "who", 0x1862},
	{"f169e", 'o', 0x2000569e, 4, false, 1, 4, "who", 0x1862},
	{"e7d", 'o', 0x20004e7d, 4, false, 1, 4, "who", 0x1862},
	{"ed9", 'o', 0x20004ed9, 4, false, 1, 4, "who", 0x1862},
	{"ea", 'o', 0x20004edd, 4, false, 6, 16, "who", 0x1862},
	{"eb", 'o', 0x20004ee1, 4, false, 6, 16, "who", 0x1862},
	{"bc1", 'o', 0x20004bc1, 4, false, 1, 4, "who", 0x1862},
	{"bc9", 'o', 0x20004bc9, 4, false, 1, 4, "who", 0x1862},
	{"e81", 'o', 0x20004e81, 4, false, 1, 4, "who", 0x1862},
	{"s51d960.calls", 'o', 0x20000000, 4, false, 1, 4, "", 0x0},
	{"s51d960.a1", 'o', 0x20000010, 4, false, 48, 4, "", 0x0},
	{"s51bfd0.calls", 'o', 0x200000e0, 4, false, 1, 4, "", 0x0},
	{"s51bfd0.a1", 'o', 0x20000100, 4, false, 20, 4, "", 0x0},
	{"s4edd20.calls", 'o', 0x20000300, 4, false, 1, 4, "", 0x0},
	{"s4ee220.calls", 'o', 0x20000310, 4, false, 1, 4, "", 0x0},
	{"s4ee220.a1", 'o', 0x20000314, 4, false, 1, 4, "", 0x0},
	{"s4ee9d0.calls", 'o', 0x20000320, 4, false, 1, 4, "", 0x0},
	{"s4ee9d0.a1", 'o', 0x20000324, 4, false, 2, 4, "", 0x0},
	{"s522fe0.calls", 'o', 0x20000340, 4, false, 1, 4, "", 0x0},
	{"s522fe0.a1", 'o', 0x20000344, 4, false, 1, 4, "", 0x0},
	{"s522fe0.a2", 'o', 0x20000348, 4, false, 1, 4, "", 0x0},
	{"s522fe0.a3", 'o', 0x2000034c, 4, false, 1, 4, "", 0x0},
	{"s522fe0.a4", 'o', 0x20000350, 4, false, 1, 4, "", 0x0},
	{"s522fe0.a5", 'o', 0x20000354, 4, false, 1, 4, "", 0x0},
	{"s522fe0.a6", 'o', 0x20000358, 4, false, 1, 4, "", 0x0},
	{"s52e690.calls", 'o', 0x20000360, 4, false, 1, 4, "", 0x0},
	{"s52e690.a1", 'o', 0x20000364, 4, false, 1, 4, "", 0x0},
	{"s52e690.a2", 'o', 0x20000368, 4, false, 1, 4, "", 0x0},
	{"s52e690.a3", 'o', 0x2000036c, 4, false, 1, 4, "", 0x0},
	{"s52e690.a4", 'o', 0x20000370, 4, false, 1, 4, "", 0x0},
	{"s52e690.a5", 'o', 0x20000374, 4, false, 1, 4, "", 0x0},
}

// stub record layout in arena object "stub" (case "stubs"): count cell, then the arguments
// of every call, one array per argument.
type rcraStub struct {
	count, args uint32
	n           int
}

var (
	rcraS51d960 = rcraStub{n4Arena + 0x0, n4Arena + 0x10, 48}
	rcraS51bfd0 = rcraStub{n4Arena + 0xe0, n4Arena + 0x100, 20}
	rcraS4edd20 = rcraStub{n4Arena + 0x300, 0, 0}
	rcraS4ee220 = rcraStub{n4Arena + 0x310, n4Arena + 0x314, 1}
	rcraS4ee9d0 = rcraStub{n4Arena + 0x320, n4Arena + 0x324, 2}
	rcraS522fe0 = rcraStub{n4Arena + 0x340, n4Arena + 0x344, 1}
	rcraS52e690 = rcraStub{n4Arena + 0x360, n4Arena + 0x364, 1}
)

const (
	rcraRet   = n4Arena + 0x40000 // 51bfd0 weights (case inputs, one per call)
	rcraWho   = n4Arena + 0x3ff0
	rcraUI6   = n4Arena + 0x11000
	rcraChars = n4Arena + 0x4000
	rcraSpl   = n4Arena + 0x12000
)

var rcraInAt, rcraOutIdx = func() (map[string]int, int) {
	m := map[string]int{}
	_, names := n4Expand(rcraInG)
	for i, n := range names {
		m[n] = i
	}
	return m, 0
}()

var rcraIns, rcraInNames = n4Expand(rcraInG)
var rcraOuts, rcraOutNames = n4Expand(rcraOutG)

type rcraC struct {
	m    n4Mem
	ins  []int64
	real bool
	ch   uint32
}

func (c *rcraC) in(name string) int64 { return c.ins[rcraInAt[name]] }

func (c *rcraC) fieldAddr(f n4Fld) uint32 {
	g := f.g
	a := g.addr + uint32(f.k*g.step)
	if g.idx != "" {
		i := c.in(g.idx)
		if i < 0 || i >= 8 {
			panic("rebuild rule abilities: record index out of range")
		}
		a += uint32(i) * g.stride
	}
	return a
}

// decoy value of the other character records (case "setup"): distinct per record and field,
// so a read of the wrong record shows up as a difference.
func rcraDecoy(j, e, size int) uint32 {
	v := (uint32(e+1)*0x9e3779b1 + uint32(j+1)*0x85ebca6b) & (uint32(1)<<(8*uint(size)) - 1)
	if size == 4 {
		v = uint32(e+1)*0x9e3779b1 + uint32(j+1)*0x85ebca6b
	}
	if v == 0 {
		v = 1
	}
	return v
}

func rcraRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(rcraIns) {
		panic("rebuild rule abilities: wrong input count")
	}
	c := &rcraC{m: n4Mem{}, ins: in, real: real}
	m := c.m
	m.w32(0x685174, rcraChars)
	m.w32(0x69c0f4, rcraUI6)
	m.w32(0x65be1c, rcraSpl)
	seen := map[[2]uint32]bool{}
	e := 0
	for _, fs := range [][]n4Fld{rcraIns, rcraOuts} {
		for _, f := range fs {
			g := f.g
			if g.idx == "" {
				continue
			}
			off := g.addr - rcraChars + uint32(f.k*g.step)
			key := [2]uint32{off, uint32(g.size)}
			if seen[key] {
				continue
			}
			seen[key] = true
			for j := 0; j < 8; j++ {
				m.put(rcraChars+off+uint32(j)*n4Ch, g.size, rcraDecoy(j, e, g.size))
			}
			e++
		}
	}
	for i, f := range rcraIns {
		if f.g.kind != 'a' {
			m.put(c.fieldAddr(f), f.g.size, uint32(in[i]))
		}
	}
	c.ch = uint32(in[rcraInAt["chPtr"]])
	c.node()
	out := make([]int64, len(rcraOuts))
	for i, f := range rcraOuts {
		v := m.get(c.fieldAddr(f), f.g.size)
		out[i] = int64(v)
		if f.g.signed {
			out[i] = int64(int32(v << (32 - 8*uint(f.g.size)) >> (32 - 8*uint(f.g.size))))
		}
	}
	return out
}

func rcraEmu(in []int64, r Rand) []int64 { return rcraRun(in, r, false) }

// callStub replays a case address stub: counts the call and files the arguments.
func (c *rcraC) callStub(s rcraStub, a ...uint32) {
	k := int(c.m.r32(s.count))
	if k >= s.n && len(a) > 0 {
		panic("rebuild rule abilities: stub record sequence full")
	}
	for i, v := range a {
		c.m.w32(s.args+uint32(4*(s.n*i+k)), v)
	}
	c.m.w32(s.count, uint32(k+1))
}

func (c *rcraC) f32(a uint32) *big.Float { return chdF(float64(math.Float32frombits(c.m.r32(a)))) }
func (c *rcraC) f64(a uint32) *big.Float {
	return chdF(math.Float64frombits(uint64(c.m.r32(a)) | uint64(c.m.r32(a+4))<<32))
}
func rcraU(v uint32) *big.Float { return chdF(float64(v)) }
func rcraI(v int64) *big.Float  { return chdF(float64(v)) }

// storeF32 is fstp dword: nearest-even rounding of the 64-bit register to single precision.
func (c *rcraC) storeF32(a uint32, v *big.Float) {
	f, _ := v.Float32()
	c.m.w32(a, math.Float32bits(f))
}

func (c *rcraC) fail(what string) { panic("rebuild rule abilities: srAssertFail " + what) }

// 0x517ec0: party slot of the record (asserts on a record not in use or outside the table).
func (c *rcraC) f517ec0(ch uint32) uint32 {
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

// 0x547940(ch, code): the class row (3 codes), the race row (5 codes), and for code 0x1c the flag +0xa79.
func (c *rcraC) f547940(code uint32) bool {
	m, ch := c.m, c.ch
	if cls := m.r32(ch + 0x69); cls != 0xffffffff {
		for k := uint32(0); k < 3; k++ {
			if m.r32(0x61507c+12*cls+4*k) == code {
				return true
			}
		}
	}
	if race := m.r32(ch + 0x71); race != 0xffffffff {
		for k := uint32(0); k < 5; k++ {
			if m.r32(0x615130+20*race+4*k) == code {
				return true
			}
		}
	}
	return code == 0x1c && m.r32(ch+0xa79) != 0
}

// 0x5479b0(ch, 6, 15.0f): level scaled by 15, truncated; above level 20 the plain 15.
func (c *rcraC) f5479b0() uint32 {
	m, ch := c.m, c.ch
	lvl := m.r32(ch + 0x8d + 4*m.r32(ch+0x69) - 0) // class -1 reads the level at +0x89
	if lvl > 0x14 {
		return 15
	}
	t := chdMul(chdMul(chdAdd(rcraU(2*lvl), c.f32(0x5ee800)), chdF(15)), c.f32(0x5ed490))
	return chdFtol(t)
}

// 0x5539e0: effective attributes, then the 41 attribute-average values.
func (c *rcraC) f5539e0() {
	m, ch := c.m, c.ch
	for i := uint32(0); i < 7; i++ {
		v := int32(uint32(int32(int8(m.r8(ch+0x177c+i)))) + m.r32(ch+0xe5+20*i))
		if v > 0x7d {
			v = 0x7d
		} else if v < 1 {
			v = 1
		}
		m.w32(ch+0xe9+20*i, uint32(v))
		c.callStub(rcraS51d960, ch)
	}
	for k := uint32(0); k < 41; k++ {
		e := 0x6155b8 + 16*k
		a, b := m.r32(e), m.r32(e+4)
		m.w32(ch+0x1a7+0x26*k, (m.r32(ch+0xe5+20*b)+m.r32(ch+0xe5+20*a))>>1)
	}
}

// 0x553a60: the 41 skill values with the favoured-skill bonus of the class.
func (c *rcraC) f553a60() {
	m, ch := c.m, c.ch
	fav := m.r32(0x6161dc + 4*m.r32(ch+0x69))
	for k := uint32(0); k < 41; k++ {
		v := m.r32(ch + 0x19f + 0x26*k)
		if k == fav {
			d := uint32(uint64(v*25) * 0x51eb851f >> 37)
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
		c.callStub(rcraS51d960, ch)
	}
}

// 0x52a2f0: rating from the class levels (the class of +0x6d counts one more level).
func (c *rcraC) f52a2f0() {
	m, ch := c.m, c.ch
	if m.r32(ch+0x69) == 0xffffffff {
		return
	}
	acc := c.f64(0x5ebb40)
	for i := uint32(0); i < 15; i++ {
		e := m.r32(ch + 0x8d + 4*i)
		if i == m.r32(ch+0x6d) {
			e++
		}
		if e == 0 {
			continue
		}
		t := chdMul(rcraU(m.r32(ch+0x125)), c.f64(0x5ed8b0))
		t = chdAdd(chdMul(t, c.f64(0x5ed8a8)), c.f64(0x5ed8a0))
		t = chdMul(chdMul(t, c.f32(0x615570+4*i)), rcraU(e))
		acc = chdAdd(acc, t)
	}
	eax := chdFtol(chdAdd(acc, c.f64(0x5ebe80))) + m.r32(ch+0xb15)
	if int32(eax) < 1 {
		eax = 1
	}
	old := m.r32(ch + 0xb0d)
	if eax == old {
		return
	}
	ecx := int32(eax - old + m.r32(ch+0xb11))
	if ecx < 0 {
		ecx = 0
	}
	m.w32(ch+0xb0d, eax)
	m.w32(ch+0xb11, uint32(ecx))
	if ecx == 0 {
		c.callStub(rcraS522fe0, c.f517ec0(ch), 0x12, 0x270f, 0, 0, 1)
	}
}

// 0x52a3e0: second rating, its running difference and the 0..4 band at +0x169e.
func (c *rcraC) f52a3e0() {
	m, ch := c.m, c.ch
	s := m.r32(ch+0x111) + m.r32(ch+0x125) + m.r32(ch+0xe9)
	x := chdMul(rcraU(s), c.f32(0x5ec410))
	y := chdAdd(chdMul(rcraU(m.r32(ch+0x89)), c.f32(0x5ed8b8)), c.f32(0x5ec0b0))
	eax := chdFtol(chdAdd(chdMul(x, y), c.f64(0x5ebe80)))
	old := m.r32(ch + 0xb19)
	ecx := m.r32(ch + 0xb21)
	var nw uint32
	if ecx < eax {
		nw = eax - ecx
	}
	m.w32(ch+0xb19, nw)
	if nw != old {
		m.w32(ch+0xb1d, m.r32(ch+0xb1d)+nw-old)
	}
	if nw == 0 {
		panic("rebuild rule abilities: 52a3e0 division by zero (fault in the original)")
	}
	t := 100 - int32(m.r32(ch+0xb1d)*100/nw)
	var band uint32
	switch {
	case t < 0x32:
	case t < 0x46:
		band = 1
	case t < 0x55:
		band = 2
	default:
		band = 3
		if t >= 0x5f {
			band = 4
		}
	}
	m.w32(ch+0x169e, band)
}

// 0x52a500: school maximum for the casting classes.
func (c *rcraC) f52a500() {
	m, ch := c.m, c.ch
	cls := int32(m.r32(ch + 0x69))
	if cls == 0 || cls > 6 && cls <= 9 {
		m.w32(ch+0xe7d, 0)
		return
	}
	m.w32(ch+0xe7d, c.f52a540())
}

// 0x4f96a0 and 0x52a540: spells known per school (+0xe5d), the six school pools (+0xb25, +0xb45).
func (c *rcraC) f52a540() uint32 {
	m, ch := c.m, c.ch
	spl := m.r32(0x65be1c)
	for k := uint32(0); k < 6; k++ {
		m.w32(ch+0xe5d+4*k, 0)
	}
	var lim [6]uint32
	for i := uint32(0); i < 0x72; i++ {
		if f := m.r32(ch + 0xbcd + 4*i); f != 1 && f != 2 {
			continue
		}
		p := spl + 0x1bf*i
		school := m.r32(p + 0x133)
		if school > 5 {
			panic("rebuild rule abilities: spell school outside 0..5")
		}
		m.w32(ch+0xe5d+4*school, m.r32(ch+0xe5d+4*school)+1)
		if v := m.r32(p + 0x49); v > lim[school] {
			lim[school] = v
		}
	}
	var s [4]uint32
	for k := uint32(0); k < 4; k++ {
		s[k] = m.r32(ch + 0x533 + 0x26*k)
	}
	for i := 0; i < 4; i++ { // descending (517a10)
		for j := i + 1; j < 4; j++ {
			if s[j] > s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
	sum := rcraI(int64(s[3]>>3) + int64(s[2]>>2) + int64(s[1]>>1) + int64(int32(s[0])))
	if sum.Cmp(c.f32(0x5ec3f8)) >= 0 {
		sum = c.f32(0x5ec3f8)
	}
	var max uint32
	for r := uint32(0); r < 6; r++ {
		sk := m.r32(ch + 0x5cb + 0x26*r)
		esi := ch + 0xb25 + 4*r
		old := m.r32(esi)
		edi := m.r32(ch + 0xe5d + 4*r)
		t := chdAdd(rcraU(3*sk), rcraI(int64(int32(m.r32(ch+0x111)))))
		t = chdMul(chdAdd(t, sum), c.f32(0x5ecbb4))
		t = chdMul(t, rcraI(int64(int32(edi+m.r32(ch+0x89)+1))))
		eax := chdFtol(chdAdd(t, c.f64(0x5ebe80)))
		if eax > max {
			max = eax
		}
		var ecx uint32
		if edi != 0 {
			ecx = eax
		}
		if lim[r] > ecx {
			ecx = lim[r]
		}
		m.w32(esi, ecx)
		if ecx != old {
			m.w32(esi+0x20, m.r32(esi+0x20)+ecx-old)
		}
	}
	return max
}

// 0x502b50: float multipliers from the two ratings and the six school pools.
func (c *rcraC) f502b50() {
	m, ch := c.m, c.ch
	k20, kInv := c.f32(0x5ed4f0), c.f32(0x5ed7ec)
	for _, p := range []struct{ src, mul, flag, dst uint32 }{{0xb0d, 0x5ebca4, 0x17b2, 0xb69}, {0xb19, 0x5ec390, 0x17b3, 0xb71}} {
		v := chdMul(chdAdd(chdMul(rcraU(m.r32(ch+p.src)), c.f32(p.mul)), k20), kInv)
		c.storeF32(ch+p.dst, v)
		if m.r8(ch+p.flag) != 0 {
			c.storeF32(ch+p.dst, chdMul(v, c.f32(0x5ec3b8)))
		}
	}
	for r := uint32(0); r < 6; r++ {
		dst := ch + 0xb79 + 8*r
		pool := m.r32(ch + 0xb25 + 4*r)
		if pool == 0 {
			m.w32(dst, 0)
			continue
		}
		c.storeF32(dst, chdMul(chdAdd(chdMul(rcraU(pool), c.f32(0x5ed7e8)), k20), kInv))
		if m.r8(ch+0x17b4) != 0 {
			w := chdF(float64(math.Float32frombits(m.r32(dst))))
			c.storeF32(dst, chdMul(w, c.f32(0x5ec3b8)))
		}
	}
}

// 0x551a60: resistance rows +0xedd (base) and +0xee1 (capped sum with the bonus bytes).
func (c *rcraC) f551a60() {
	m, ch := c.m, c.ch
	for r := uint32(0); r < 6; r++ {
		v := m.r32(ch+0x5cb+0x26*r)/10 + 0x19
		if m.r8(ch+0x6f5) != 0 {
			v += m.r32(ch+0x6fb)/5 + 5
		}
		if m.r32(ch+0x69) == 0xe {
			v += 5
		}
		m.w32(ch+0xedd+0x10*r, v)
	}
	if race := m.r32(ch + 0x71); race != 0xffffffff {
		for i := uint32(0); i < 6; i++ {
			e := 0x615270 + 8*(6*race+i)
			typ, val := m.r32(e), m.r32(e+4)
			if typ == 0xffffffff {
				break
			}
			add := val
			if int32(val) > 0x3e8 {
				add = m.r32(ch+0xe5+20*(val-1000)) / 5
			}
			m.w32(ch+0xedd+0x10*typ, m.r32(ch+0xedd+0x10*typ)+add)
		}
	}
	for _, p := range [][2]uint32{{0xfd, 0xf1d}, {0x111, 0xf2d}} {
		if v := m.r32(ch + p[0]); v > 0x50 {
			m.w32(ch+p[1], m.r32(ch+p[1])+(v-0x50)>>1)
		}
	}
	for j := uint32(0); j < 6; j++ {
		v := m.r32(ch+0xedd+0x10*j) + uint32(int32(int8(m.r8(ch+0x1777)))) + uint32(int32(int8(m.r8(ch+0x17ac+j))))
		if v > 0x64 {
			v = 0x64
		}
		m.w32(ch+0xee1+0x10*j, v)
	}
}

// 0x4edc60: carried weight (20 item records) into +0xbb9; true when it changed.
func (c *rcraC) f4edc60() bool {
	m, ch := c.m, c.ch
	old := m.r32(ch + 0xbb9)
	m.w32(ch+0xbb9, 0)
	for k := uint32(0); k < 20; k++ {
		p := ch + 0xf5d + 12*k
		if k >= 12 {
			p = ch + 0x1029 + 12*(k-12)
		}
		m.w32(ch+0xbb9, m.r32(ch+0xbb9)+c.f51bfd0(p))
	}
	total, capa := m.r32(ch+0xbb9), m.r32(ch+0xbc5)
	if old < capa && total > capa {
		e := m.r32(0x5ee5a8)
		if m.r32(0x68ec78) != 6 {
			e = m.r32(0x68c558)
		}
		c.callStub(rcraS52e690, ch, e, 0, m.r32(0x5ed8c8), m.r32(0x5ed914))
	}
	return old != total
}

// 0x51bfd0: Emu replays the case's weight sequence (call k reads weight k); F weighs the
// item the F-only cells hold for call k with the real itemWeightNode.
func (c *rcraC) f51bfd0(p uint32) uint32 {
	k := int(c.m.r32(rcraS51bfd0.count))
	c.callStub(rcraS51bfd0, p)
	if !c.real {
		return c.m.r32(rcraRet + 4*uint32(k))
	}
	w := itemWeightNode([]int64{c.in(fmt.Sprint("fvid", k)), c.in(fmt.Sprint("fvqty", k)), c.in(fmt.Sprint("fvunit", k))}, nil)
	return uint32(w[0])
}

// 0x4ee000: base attack bonus from level, two attributes, the bonus byte and the load band.
func (c *rcraC) f4ee000() {
	m, ch := c.m, c.ch
	e := (m.r32(ch+0x89)+1)>>1 + m.r32(ch+0x161)/5 + m.r32(ch+0x14d)/5 - 10
	if m.r8(ch+0x767) != 0 {
		e += m.r32(ch+0x76d)/10 + 1
	}
	e += uint32(int32(int8(m.r8(ch + 0x1770))))
	band := m.r32(ch + 0xbc9)
	if band > 4 {
		c.fail("4ee000 load band")
	}
	e += [5]uint32{0, ^uint32(0), ^uint32(1), ^uint32(3), ^uint32(7)}[band]
	m.w32(ch+0xe81, e)
}

// 0x55ee30(slot) and 0x562a50(mask): interface refresh requests.
func (c *rcraC) f55ee30(slot uint32) {
	m := c.m
	switch m.r32(0x68ec78) {
	case 6:
		p := m.r32(0x69c0f4) + 0xf8
		m.w32(p, m.r32(p)|0x100)
	case 7:
		if b := m.r32(0x68edcc); b != 0 {
			m.w32(b+0xf4, m.r32(b+0xf4)|1<<(slot&31))
		}
	}
}

func (c *rcraC) node() {
	m, ch := c.m, c.ch
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
	c.f5539e0()
	c.f553a60()
	c.f52a2f0()
	c.f52a3e0()
	c.f52a500()
	c.callStub(rcraS4ee9d0, ch)
	c.f502b50()
	m.w32(ch+0xed9, 0)
	if c.f547940(0x1d) {
		m.w32(ch+0xed9, m.r32(ch+0x125)/10)
	}
	if c.f547940(6) {
		m.w32(ch+0xed9, m.r32(ch+0xed9)+c.f5479b0())
	}
	if m.r8(ch+0x71b) != 0 {
		m.w32(ch+0xed9, m.r32(ch+0xed9)+m.r32(ch+0x721)>>2+5)
	}
	m.w32(ch+0xed9, m.r32(ch+0xed9)+uint32(int32(int8(m.r8(ch+0x1776)))))
	c.f551a60()
	capa := 12 * (m.r32(ch+0x125) + 2*m.r32(ch+0xe9))
	if c.f547940(0x18) {
		capa = uint32(uint64(capa*2) * 0xaaaaaaab >> 33)
	}
	changed := m.r32(ch+0xbc5) != capa
	m.w32(ch+0xbc5, capa)
	heavier := c.f4edc60()
	if m.r8(0x68517c) != 0 {
		if changed || heavier {
			c.f4edd20()
		}
	} else {
		m.w32(ch+0xbbd, 0)
	}
	tot := m.r32(ch+0xbb9) + m.r32(ch+0xbbd)
	m.w32(ch+0xbc1, tot)
	pct := tot * 100 / capa
	var band uint32
	switch {
	case pct < 0x32:
	case pct < 0x46:
		band = 1
	case pct < 0x55:
		band = 2
	case pct <= 0x64:
		band = 3
	default:
		band = 4
	}
	m.w32(ch+0xbc9, band)
	c.f4ee000()
	c.callStub(rcraS4ee220, ch)
	c.callStub(rcraS4ee9d0, ch)
	if m.r8(ch+4) != 0 && m.r32(0x68ec78) != 3 {
		c.f55ee30(c.f517ec0(ch))
	}
}

// 0x4edd20: Emu counts the call (stubbed D4 callee); F runs sharedEncumbranceNode on a world
// made of this record (slot who) and the F-only cells of the other members, and takes back
// the shared weight (+0xbbd), the screen flags and the refresh mask.
func (c *rcraC) f4edd20() {
	c.callStub(rcraS4edd20)
	if !c.real {
		return
	}
	m, ch := c.m, c.ch
	who := int((ch - rcraChars) / n4Ch)
	w := make([]int64, 15+8*7)
	w[0], w[1], w[3] = int64(m.r8(0x68517c)), c.in("fcomb"), c.in("fother")
	w[4], w[8] = int64(m.r32(0x68ec78)), int64(m.r32(rcraUI6+0xf8))
	w[6], w[9], w[10], w[11], w[12] = c.in("fitems"), c.in("fsitemzero"), c.in("fsqtyzero"), c.in("fsitemone"), c.in("fsqtyone")
	w[13], w[14] = c.in("fuwzero"), c.in("fuwone")
	g := m.r32(0x68edcc)
	if g != 0 {
		w[5], w[7] = 1, int64(m.r32(g+0xf4))
	}
	for k := 0; k < 8; k++ {
		e := w[15+7*k : 22+7*k]
		e[0], e[1], e[2], e[3] = c.in(fmt.Sprint("fpres", k)), c.in(fmt.Sprint("fstat", k)), c.in(fmt.Sprint("fpers", k)), c.in(fmt.Sprint("fcap", k))
		if k == who {
			e[2], e[3], e[4] = int64(m.r32(ch+0xbb9)), int64(m.r32(ch+0xbc5)), int64(m.r32(ch+0xbbd))
		}
	}
	out := sharedEncumbranceNode(w, nil)
	m.w32(ch+0xbbd, uint32(out[3*who]))
	m.w32(rcraUI6+0xf8, uint32(out[25]))
	if g != 0 {
		m.w32(g+0xf4, uint32(out[26]))
	}
}

func init() {
	Nodes["004ed9d0"] = Node{Addr: "004ed9d0", In: rcraInNames, Out: rcraOutNames, F: func(in []int64, r Rand) []int64 { return rcraRun(in, r, true) }, Emu: rcraEmu}
}

func rebuildCharacterRuleAbilitiesNode(in []int64, r Rand) []int64 { return rcraRun(in, r, true) }
