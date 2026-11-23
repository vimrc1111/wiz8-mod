package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
)

// character_hp_deficit_apply 0x52b7e0(who, amount, flag): party member `who`
// (record at *0x685174, stride 0x1862; presence byte at *0x685178 + 0x106*who,
// zero is the srAssertFail path 0x52b814, calc panics) takes `amount` off the
// carried maximum-HP bonus +0xb15. Nothing happens when +0xb0d (max HP) or +0xb11
// (HP) is 0. Otherwise: 0x52af50(who, floor(2*amount mod 2^32 / 3), 0, 0), then
// when the low byte of flag is set 0x58aad0(8, *(*0x68c09c+0x710), amount),
// +0xb15 -= amount (32-bit wrap), then 0x52a2f0(char) (recompute max HP, below),
// then state 1 for 0x270f turns through 0x522fe0(who, 1, 0x270f, 0, 0, 0) when
// +0xa05 is 0, then 0x52f2c0(char) (injury response, below) when +0xb11 != 0.
//
// 0x52a2f0 (unregistered, native in the emulator): unless +0x69 is -1, S is the
// x87 sum over k = 0..14 of ((double)lvl*0.4*0.02+0.6) * float32 0x615570[k] *
// (double)c, where c = u32 +0x8d+4k, plus 1 when k == +0x6d, skipped when c is 0,
// lvl = u32 +0x125 (0.4 0x5ed8b0, 0.02 0x5ed8a8, 0.6 0x5ed8a0, start 0.0 0x5ebb40,
// +0.5 0x5ebe80; every step rounds to the 64-bit significand, _ftol truncates).
// newMax = low32 + +0xb15 (signed), at least 1. A new max replaces +0xb0d and moves
// +0xb11 by the difference, floored at 0; HP 0 then runs 0x522fe0(who, 0x12,
// 0x270f, 0, 0, 1) (0x517ec0 returns the record index, which is who).
//
// 0x52f2c0 (non-formula, native): p = u32(+0xb11*100 mod 2^32) / +0xb0d. With
// f(id) = bit `who` of byte id of the voice flag table (*(*0x683fd7+0x68), ids
// below 500 index directly), bl = f(3), f2 = f(2) (f(28) is read and unused):
// unless +0xb01 is 0xf or 0x11, p < 20 and bl == 0 calls 0x52e690(char, 3, 1, 2,
// 127); else p < 50 draws R(2) and calls it with id 28 (draw != 0, or f2 != 0)
// else 2. Always last R(3) picks one of g68c544, g68c550, g68c51c for
// 0x52e690(char, picked, 1, 1, 127). The event pointers 0x52e690 returns are
// only written to (+0x30, +0x34), never read back. The name byte +4 (0x517ec0
// assert) is fixed non-zero in the case; the 0x4e/0xff record names are not
// inputs.
//
// Emu address-stubs the D4 callees 0x52af50 and 0x522fe0 (composition ruling
// 2026-10-09; returns from case inputs, no eax of either is read) and the
// excluded 0x58aad0 and 0x52e690, recording every call's arguments and the call
// counts. F runs the real characterFatigueApplyNode and
// characterConditionApplyNode on a world made from the node's inputs and the
// F-only cells (ft*, fatigue cells) and feeds their results into the rest of the
// flow (a05, +0xb01, +0xb11, slots).
// Fixture caps (calc panics past them): who 0..7 (eight records), results of the
// x87 sum beyond int64 (the emulator stops there as well).
const (
	hdaMembers  = 8
	hdaFT       = 305 // F-only world cells, ccaWLen
	hdaFat      = 6   // F-only fatigue cells: stamina, staMax, encumbrance, +0xaa9, level, flag
	hdaTab      = 0x615570
	hdaChar     = 0x20000000 // chars object of the case
	hdaCharSize = ccaCharSize
)

// input layout
const (
	hdaWho = iota
	hdaAmount
	hdaFlag
	hdaPres
	hdaText
	hdaB0d
	hdaB11
	hdaB15
	hdaA05
	hdaB01
	hdaF69
	hdaF6d
	hdaE     // 15
	hdaLvl   = hdaE + 15
	hdaVt2   = hdaLvl + 1
	hdaVt3   = hdaVt2 + 1
	hdaG544  = hdaVt3 + 1
	hdaG550  = hdaG544 + 1
	hdaG51c  = hdaG550 + 1
	hdaRet52 = hdaG51c + 1 // ret52af, ret522a, ret522b (stub returns, ignored by the caller)
	hdaFt    = hdaRet52 + 3
	hdaFat0  = hdaFt + hdaFT
	hdaInLen = hdaFat0 + hdaFat
)

type hdaExeTab struct {
	tab              [15]float64
	d06, d002, d04   float64
	d0, d05          float64
	c20, c50, c127   uint32 // 0x5ed8fc, 0x5ed8f8, 0x5ed914
	c3, c2, c28      uint32 // 0x5ee594, 0x5ee590, 0x5ee5f8
	c1, c1b          uint32 // 0x5ed8d4, 0x5ed8cc
	cB               uint32 // 0x5ed8d0 (2)
	g5ed8c8, g5ee598 uint32 // fatigue constants
}

var hdaX, hdaXErr = hdaLoad()

func hdaLoad() (t hdaExeTab, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	at := func(va uint32, n int) []byte {
		o, e := peOffset(b, va)
		if e != nil || len(b)-o < n {
			err = fmt.Errorf("hp deficit: cannot read %#x", va)
			return make([]byte, n)
		}
		return b[o : o+n]
	}
	u32 := func(va uint32) uint32 { return binary.LittleEndian.Uint32(at(va, 4)) }
	f64 := func(va uint32) float64 { return math.Float64frombits(binary.LittleEndian.Uint64(at(va, 8))) }
	for k := range t.tab {
		t.tab[k] = float64(math.Float32frombits(u32(hdaTab + 4*uint32(k))))
	}
	t.d06, t.d002, t.d04, t.d0, t.d05 = f64(0x5ed8a0), f64(0x5ed8a8), f64(0x5ed8b0), f64(0x5ebb40), f64(0x5ebe80)
	t.c20, t.c50, t.c127 = u32(0x5ed8fc), u32(0x5ed8f8), u32(0x5ed914)
	t.c3, t.c2, t.c28 = u32(0x5ee594), u32(0x5ee590), u32(0x5ee5f8)
	t.c1, t.c1b, t.cB = u32(0x5ed8d4), u32(0x5ed8cc), u32(0x5ed8d0)
	t.g5ed8c8, t.g5ee598 = u32(0x5ed8c8), u32(0x5ee598)
	return t, err
}

func hdaNames() []string {
	n := []string{"who", "amount", "flag", "pres", "textPtr", "b0d", "b11", "b15", "a05", "b01", "f69", "f6d"}
	for k := 0; k < 15; k++ {
		n = append(n, fmt.Sprintf("e%d", k))
	}
	n = append(n, "lvl", "vt2", "vt3", "g544", "g550", "g51c", "ret52af", "ret522a", "ret522b")
	for k := 0; k < hdaFT; k++ {
		n = append(n, fmt.Sprintf("ft%d", k))
	}
	return append(n, "fstamina", "fstaMax", "fenc", "faa9", "flevel", "fflag")
}

func hdaOutNames() []string {
	n := []string{"b0d", "b11", "b15", "a05", "b01", "n52af"}
	for j := 1; j <= 4; j++ {
		n = append(n, fmt.Sprintf("a52af_%d", j))
	}
	n = append(n, "n58")
	for j := 1; j <= 3; j++ {
		n = append(n, fmt.Sprintf("a58_%d", j))
	}
	n = append(n, "n522")
	for j := 1; j <= 6; j++ {
		for k := 0; k < 2; k++ {
			n = append(n, fmt.Sprintf("a522_%d_%d", j, k))
		}
	}
	n = append(n, "n52e")
	for j := 1; j <= 5; j++ {
		for k := 0; k < 2; k++ {
			n = append(n, fmt.Sprintf("a52e_%d_%d", j, k))
		}
	}
	return n
}

func init() {
	Nodes["0052b7e0"] = Node{Addr: "0052b7e0", In: hdaNames(), Out: hdaOutNames(),
		F:   characterHpDeficitApplyNode,
		Emu: func(in []int64, r Rand) []int64 { return hdaRun(in, r, false) }}
}

func characterHpDeficitApplyNode(in []int64, r Rand) []int64 { return hdaRun(in, r, true) }

type hdaS struct {
	w    []int64 // 0x522fe0 world (ccaW cells), carries b01, b11, a05 (slot 1)
	r    Rand
	real bool
	who  int64
	// stub traces
	n52af, n58, n522, n52e int64
	a52af                  [4]int64
	a58                    [3]int64
	a522                   [2][6]int64
	a52e                   [2][5]int64
}

// call522 is a call of 0x522fe0: recorded, and in F run for real with the result fed back.
func (s *hdaS) call522(st int64, tm uint32, flagC int64) {
	if s.n522 >= 2 {
		panic("hp deficit: more 0x522fe0 calls than the node makes")
	}
	s.a522[s.n522] = [6]int64{s.who, st, int64(tm), 0, 0, flagC}
	s.n522++
	if s.real {
		s.run522(st, tm, flagC)
	}
}

// run522 runs the real 0x522fe0 on the world and feeds its result back.
func (s *hdaS) run522(st int64, tm uint32, flagC int64) {
	v := append([]int64{s.who, st, int64(tm), 0, 0, flagC}, s.w...)
	s.feed(characterConditionApplyNode(v, s.r))
}

// feed copies the world cells 0x522fe0 changes back (ccaS.output order).
func (s *hdaS) feed(o []int64) {
	w := s.w
	for k := 0; k < ccaSlots; k++ {
		w[ccaWS+k] = o[1+k]
	}
	w[ccaWB01], w[ccaWB09], w[ccaWB11] = o[1+ccaSlots], o[2+ccaSlots], o[3+ccaSlots]
	w[ccaWG6840], w[ccaWVClr] = o[4+ccaSlots], o[5+ccaSlots]
	i := 6 + ccaSlots
	for j := 0; j < ccaChars; j++ {
		w[ccaWRecBe+j] = o[i]
		i++
	}
	pc := w[ccaWPC : ccaWPC+ccaPCLen]
	i += copy(pc[5:21], o[i:])
	pc[2], pc[3] = o[i], o[i+1]
	i += 2
	i += copy(pc[21:], o[i:])
	w[ccaWMon+3], w[ccaWMon+7] = o[i], o[i+1]
}

func hdaRun(in []int64, r Rand, real bool) []int64 {
	if hdaXErr != nil {
		panic(hdaXErr)
	}
	if len(in) != hdaInLen || ccaWLen != hdaFT {
		panic("hp deficit: wrong input count")
	}
	who := in[hdaWho]
	if who < 0 || who >= hdaMembers {
		panic("hp deficit: member outside the eight fixture records")
	}
	if in[hdaPres]&0xff == 0 {
		panic("hp deficit: assert, member absent") // 0x52b814
	}
	amount := uint32(in[hdaAmount])
	b0d, b15 := uint32(in[hdaB0d]), uint32(in[hdaB15])
	w := append([]int64(nil), in[hdaFt:hdaFt+hdaFT]...)
	w[ccaWPres+int(who)], w[ccaWWhoPres] = in[hdaPres]&0xff, in[hdaPres]&0xff
	w[ccaWB01], w[ccaWB0d], w[ccaWB11], w[ccaWS+1] = in[hdaB01], in[hdaB0d], in[hdaB11], in[hdaA05]
	s := &hdaS{w: w, r: r, real: real, who: who}
	finish := func() []int64 {
		o := []int64{int64(b0d), int64(uint32(s.w[ccaWB11])), int64(b15), int64(uint32(s.w[ccaWS+1])), int64(uint32(s.w[ccaWB01])), s.n52af}
		o = append(o, s.a52af[:]...)
		o = append(o, s.n58)
		o = append(o, s.a58[:]...)
		o = append(o, s.n522)
		for j := 0; j < 6; j++ {
			o = append(o, s.a522[0][j], s.a522[1][j])
		}
		o = append(o, s.n52e)
		for j := 0; j < 5; j++ {
			o = append(o, s.a52e[0][j], s.a52e[1][j])
		}
		return o
	}
	if b0d == 0 || uint32(s.w[ccaWB11]) == 0 {
		return finish()
	}
	t := uint32(uint64(2*amount) / 3) // 2*amount wraps in 32 bits
	s.n52af, s.a52af = 1, [4]int64{who, int64(t), 0, 0}
	if real {
		f := []int64{who, int64(t), 0, 0, in[hdaFat0], int64(uint32(s.w[ccaWB11])), in[hdaFat0+1], in[hdaFat0+2],
			s.w[ccaWS+5], in[hdaFat0+3], in[hdaFat0+4], in[hdaFat0+5], s.w[ccaWS+17], 0,
			int64(hdaX.g5ed8c8), int64(hdaX.c127), int64(hdaX.g5ee598), 0, 0, 0}
		fo := characterFatigueApplyNode(f, r)
		if fo[6] == 1 { // its state 0x11 for 0x270f turns, target 0 so flagC = 1
			s.run522(0x11, 0x270f, 1)
		}
	}
	if in[hdaFlag]&0xff != 0 {
		s.n58, s.a58 = 1, [3]int64{8, int64(uint32(in[hdaText])), int64(amount)}
	}
	b15 -= amount
	// 0x52a2f0
	if int32(in[hdaF69]) != -1 {
		nm := hdaMaxHP(in, b15)
		if nm != b0d {
			hp := nm - b0d + uint32(s.w[ccaWB11])
			if int32(hp) < 0 {
				hp = 0
			}
			b0d, s.w[ccaWB0d], s.w[ccaWB11] = nm, int64(nm), int64(hp)
			if hp == 0 {
				s.call522(0x12, 0x270f, 1)
			}
		}
	}
	if uint32(s.w[ccaWS+1]) == 0 {
		s.call522(1, 0x270f, 0)
	}
	if uint32(s.w[ccaWB11]) != 0 {
		s.injury(in, b0d)
	}
	return finish()
}

// hdaMaxHP is the x87 sum of 0x52a2f0 plus +0xb15, clamped to at least 1.
func hdaMaxHP(in []int64, b15 uint32) uint32 {
	x := &hdaX
	nf := func(v float64) *big.Float { return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).SetFloat64(v) }
	sum := nf(x.d0)
	lvl := nf(float64(uint32(in[hdaLvl])))
	lvl.Mul(lvl, nf(x.d04))
	lvl.Mul(lvl, nf(x.d002))
	lvl.Add(lvl, nf(x.d06))
	for k := 0; k < 15; k++ {
		c := uint32(in[hdaE+k])
		if uint32(in[hdaF6d]) == uint32(k) {
			c++
		}
		if c == 0 {
			continue
		}
		term := nf(0).Mul(lvl, nf(x.tab[k]))
		term.Mul(term, nf(float64(c)))
		sum.Add(sum, term)
	}
	sum.Add(sum, nf(x.d05))
	i, _ := sum.Int(nil) // _ftol truncates
	if !i.IsInt64() {
		panic("hp deficit: max HP sum outside i64 (the emulator stops there too)")
	}
	nm := int32(uint32(i.Int64()) + b15)
	if nm < 1 {
		nm = 1
	}
	return uint32(nm)
}

// injury is 0x52f2c0.
func (s *hdaS) injury(in []int64, b0d uint32) {
	x := &hdaX
	p := uint32(s.w[ccaWB11]) * 100 / b0d
	bit := func(id int64) bool {
		if id != 2 && id != 3 {
			return false // id 28 is read and unused
		}
		return in[hdaVt2+int(id-2)]>>uint(s.who)&1 != 0
	}
	bl, f2 := bit(3), bit(2)
	call := func(a [5]int64) {
		if s.n52e >= 2 {
			panic("hp deficit: more 0x52e690 calls than the node makes")
		}
		s.a52e[s.n52e] = a
		s.n52e++
	}
	b01 := uint32(s.w[ccaWB01])
	if b01 != 0xf && b01 != 0x11 {
		if p < x.c20 && !bl {
			call([5]int64{hdaChar + hdaCharSize*s.who, int64(x.c3), int64(x.c1), int64(x.cB), int64(x.c127)})
		} else if p < x.c50 {
			id := x.c2
			if s.r.R(2) != 0 || f2 {
				id = x.c28
			}
			call([5]int64{hdaChar + hdaCharSize*s.who, int64(id), int64(x.c1), int64(x.cB), int64(x.c127)})
		}
	}
	g := [3]int64{in[hdaG544], in[hdaG550], in[hdaG51c]}
	call([5]int64{hdaChar + hdaCharSize*s.who, g[s.r.R(3)], int64(x.c1), int64(x.c1b), int64(x.c127)})
}
