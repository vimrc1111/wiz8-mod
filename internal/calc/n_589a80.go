package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strings"
)

// 0x589a80 trap_attempt_training(panel): the per-frame step of the trap panel (ecx = the
// panel, the same object as 0x68f2d4). It advances the panel's three countdown timers and
// runs the action selected by the panel state +0x18.
//
// Timer object (0x43a190 progress, 0x439b80 set length, 0x439d80 restart): +4 kind,
// +8 flags (low byte), +0x10 start, +0x14 end, +0x18 period, +0x1c seconds (float32),
// +0x20 rate (float32). "now" is ((days*86400000)+ticks)*10 from 0x687595/0x6874f7 when
// kind == 1; otherwise the clock import getUTime(0) when flags&1, else the frozen tick
// 0x6598c8 when byte 0x6598d0 is set, else getUTime(0) - 0x6598c4 (all u32 wrapping).
// progress = (now-start)/(end-start) as u32 values divided in x87 extended precision and
// stored as float32; its truncation n, when n >= 1 (signed), moves the window to
// start = end + (n-1)*period, end = start + period. The float is returned unless flags&8,
// byte 0x6598d1, or (byte 0x6598d0 or byte 0x6598d2) with flags&1 clear zero it (exe
// 0x5ebb34 = 0.0f).
//
//  1. State 9: when progress(T0 = panel+0x154) >= 1.0f (exe 0x5ebb38; NaN fails) and
//     0x52e590 finds no party slot k with both slot byte [0x685178]+0x106k and voice byte
//     0x6836b8+0x118k set, 0x43d940(-1) on the trap (panel+8) and state = 10.
//  2. State 10: busy byte 0x683f99 = 0; the panel at 0x68f2d4 is destroyed (its child
//     widget and button strip clear their control-enable dwords 0x61f238+12*code, codes
//     from the widgets' +0x48: 1 and 3 in the case); 0x68f2d4 = 0; 0x58f6b0(0) calls
//     0x562a50(0x800); 0x562580([0x68f2c4], 1); 0x562a50 with 0x200, 0x100, 0x1000. Return.
//  3. Child widget (panel+0xc) timer T2 at +0x118 while byte +0x140: n = ftol(progress);
//     n != 0 adds n to +0x13c and sets byte +0x141.
//  4. Child timer T1 at +0x94 while byte +0x84: progress below float +0x88 (or unordered)
//     is stored in +0x8c and the function returns; otherwise byte +0x84 = 0, the child's
//     show(0) sets bytes +5 and +0x34 and calls 0x562a50(0x80000000), and every one of the
//     eight buttons [child+0x54+4k] whose byte +0x34 has no bit of exe byte 0x5ed570 (2)
//     gets byte +0xbc = 0.
//  5. Dispatch on state (others do nothing):
//     1: state 0, 0x58a060 (trap inspection).
//     2: state 0, +0x150 = 0, 0x553f10(character panel+0x1c, 10, 1, 0); 0x589d90 refreshes
//     the widgets (below); then, unless some part i in 0..7 has 0x5e3730(panel+0x34, i)
//     nonzero while byte panel+0x3c+i is 0, state = 3.
//     3: the three controls (codes 1, 2, 3) are cleared; 0x5e3780 springs the trap
//     (+0x370 = 1; a linked object [+0x234] of byte kind 0x0a loses flag bit 2 of byte +8;
//     R(100) < 40 calls 0x52e5c0([0x68c510], -1, 0, [0x5ed8c8]); the message goes out through
//     0x5299b0; 0x43d940(-1)); then the state-10 teardown.
//     4: a playing sound handle +0x150 is released (0x408f70; nothing to release in the
//     case) and zeroed; 0x408860(0x64bcd0 "Trap Sprung.wav", 0); the three controls are
//     cleared; 0x5e3ab0(trap); state 9; T0 gets seconds 2.0f and
//     period = ftol(rate*seconds*10000.0f) (exe 0x5ec0a8), then restarts at now.
//     5: state 0, 0x58a200 (spell attempt, below). 6: state 0, 0x58a3e0.
//     7, 8: progress(T0) >= 1.0f sets state 3 (from 7) or 4 (from 8).
//
// 0x589d90 (character c = 0x68518d): v = +0x31f when 0x524a10(c) and (byte +0x319 or
// dword +0x31f), else -1; buttons without bit 2 get +0xbc = (v < 0). The remaining calls
// that matter here are counts: 0x524a10 once, then the spell check (0x524a10 and
// dword +0xc69 == 1 calls 0x4ff7f0; else a second 0x524a10 and +0xc15 == 1 calls
// 0x4ff7f0), and 0x588e60 calls 0x524a10 three times and 0x4ff7f0 for each of
// +0xc69 == 1 and +0xc15 == 1 while 0x524a10 holds. 0x4ff7f0's return only selects
// display text. (+0xc69 and +0xc15 are the known flags +0xbcd+4sp of spells 0x27 and 0x12.)
// 0x58a200: the same spell check; if passed and 0x4f9750(ch, 0x27) or 0x4f9750(ch, 0x12)
// (spell known dword +0xbcd+4sp == 1 and realm skill +0xb45+4*realm >= requirement, from
// the record [0x65be1c]+447*sp: realm +0x133, requirement +0x49), busy = 0, the three
// controls are cleared, byte 0x683f9d = 1, the 0x58f6b0/0x562580/0x562a50 sequence above,
// and 0x5a0110(0x12 if 0x12 is usable else 0x27, -1, -1).
//
// Emu address-stubs the D4 callees 0x58a060, 0x553f10, 0x5e3ab0, 0x524a10, 0x4ff7f0 and the
// 排除 callees 0x5a0110, 0x58a3e0, 0x43d940, 0x562580, 0x408860, 0x5299b0, 0x52e5c0,
// 0x562a50 (count and arguments are outputs). Of the stubbed callees this function reads
// the return of 0x524a10 (input ret524, low byte) and the 0x553f10 writes +0x319/+0x31f
// of character c (inputs post319/post31f). F runs the real characterActionEligibilityNode
// on (c, slot byte, elHP, elStatus) and the real naturalSkillGrowthNode on the F-only sg_*
// inputs (as in n_58a060.go), whose skill-10 flag and effective value replace post319 and
// post31f when panel+0x1c == c; its R draws come before 0x5e3780's.
// 0x4ff7f0 returns a realm skill index 24..27; its value only addresses display text.
//
// Fixture caps (calc panics past them): c (0x68518d) and the stubbed 0x4ff7f0 return are
// ID ranges (0..7, 24..27); the realms of spells 0x12/0x27 are 0..5; the trap type id
// (+0x37c) indexes the 15-entry message table 0x61e9ec; the link pointer is 0 or the case's
// link object; the 0x5e3730 row panel+0x34 must stay inside the exe image; getUTime is
// stubbed for four calls.
const (
	tatArena = 0x20000000
	tatPanel = tatArena + 0x16000 // the case's panel object (0x68f2d4)
	tatTrig  = tatArena + 0x1e800 // the case's trap object (panel+8)
	tatLink  = tatArena + 0x1f000 // the case's linked object
	tatSnd   = 0x64bcd0           // 0x589ce0: first argument of 0x408860
)

var tatInNames = strings.Fields(`state charIdx trapType ident0 ident1 ident2 ident3 ident4 ident5 ident6 ident7 sound
t0Kind t0Flags t0Start t0End t0Period t0Secs t0Rate t1Kind t1Flags t1Start t1End t1Period
t2Kind t2Flags t2Start t2End t2Period c84 c88 c13c c140 c141 c8c child5 child34
btnFlag0 btnFlag1 btnFlag2 btnFlag3 btnFlag4 btnFlag5 btnFlag6 btnFlag7
btnSel0 btnSel1 btnSel2 btnSel3 btnSel4 btnSel5 btnSel6 btnSel7
clockBase frozenTicks frozenOn pausedAll pausedGame gameDays gameTicks ms0 ms1 ms2 ms3
slot0 slot1 slot2 slot3 slot4 slot5 slot6 slot7 voice0 voice1 voice2 voice3 voice4 voice5 voice6 voice7
trigType trigSprung linkPtr linkKind linkFlags ctl1 ctl2 ctl3 voiceMgr screenMode busyIn spellModeIn
activeChar known12 known27 realmSkill0 realmSkill1 realmSkill2 realmSkill3 realmSkill4 realmSkill5
realm12 req12 realm27 req27 ret524 ret4ff ret5a0 ret553 post319 post31f elHP elStatus`)

var tatOutNames = strings.Fields(`state sound t0Start t0End t0Period t0Secs t1Start t1End t2Start t2End
c84 c8c c13c c141 child5 child34 btnSel0 btnSel1 btnSel2 btnSel3 btnSel4 btnSel5 btnSel6 btnSel7
busy99 spellMode curPanel trigSprung linkFlags ctl1 ctl2 ctl3
n58a060 n553 a553_1 a553_2 a553_3 a553_4 n5e3ab0 a5e3ab0 n524 a524 n4ff n5a0 a5a0_1 a5a0_2 a5a0_3 n58a3e0
n43d940 a43d940 n562580 a562580_1 a562580_2 n408860 a408860_1 a408860_2 n5299b0
n52e5c0 a52e5c0_1 a52e5c0_2 a52e5c0_3 a52e5c0_4 n562a50`)

// input width: 1, 2 or 4 bytes; negative = signed 4 bytes
var tatWidth = map[string]int{"ident": 1, "t0Flags": 2, "t1Flags": 2, "t2Flags": 2, "c84": 1, "c140": 1,
	"c141": 1, "child5": 1, "child34": 1, "btnFlag": 1, "btnSel": 1, "frozenOn": 1, "pausedAll": 1,
	"pausedGame": 1, "slot": 1, "voice": 1, "trigSprung": 1, "linkKind": 1, "linkFlags": 1, "busyIn": 1,
	"spellModeIn": 1, "post319": 1, "charIdx": -4, "realmSkill": -4, "req12": -4, "req27": -4, "post31f": -4}

var (
	tatIn    = map[string]int{}
	tatSgIdx []int // for every 0x553f10 input its index in the node input (-1: call constant)
	tatF19d  = -1  // index of f19d10 / eff10 in the 0x553f10 outputs
	tatEff   = -1
)

func init() {
	names := append([]string{}, tatInNames...)
	(&n553In{}).bind(func(name string, _ *int64, _ int) {
		switch name {
		case "slot", "charAddr", "s", "n", "quiet":
			tatSgIdx = append(tatSgIdx, -1)
		default:
			tatSgIdx = append(tatSgIdx, len(names))
			names = append(names, "sg_"+name)
		}
	})
	k := 0
	(&n553Out{}).bind(func(name string, _ *int64) {
		switch name {
		case "f19d10":
			tatF19d = k
		case "eff10":
			tatEff = k
		}
		k++
	})
	for i, n := range tatInNames {
		tatIn[n] = i
	}
	out := append([]string{}, tatOutNames...)
	for k := 0; k < 16; k++ {
		out = append(out, fmt.Sprintf("a562a50_%d", k))
	}
	Nodes["00589a80"] = Node{Addr: "00589a80", In: names, Out: out, F: trapAttemptTrainingNode,
		Emu: func(in []int64, r Rand) []int64 { return tatRun(in, r, false) }}
}

func trapAttemptTrainingNode(in []int64, r Rand) []int64 { return tatRun(in, r, true) }

type tatTimer struct{ kind, flags, start, end, period uint32 }

type tatSim struct {
	in    []int64
	r     Rand
	real  bool
	out   map[string]int64
	utime int // getUTime calls so far
	a562  []int64
}

// v returns input name, checked against its declared width.
func (s *tatSim) v(name string) int64 {
	x := s.in[tatIn[name]]
	w, ok := tatWidth[name]
	if !ok {
		w, ok = tatWidth[strings.TrimRight(name, "0123456789")]
	}
	if !ok {
		w = 4
	}
	lo, hi := int64(-1)<<31, int64(1)<<31-1
	if w > 0 {
		lo, hi = 0, int64(1)<<(8*w)-1
	}
	if x < lo || x > hi {
		panic(fmt.Sprintf("trap attempt training: input %s=%d outside %d..%d", name, x, lo, hi))
	}
	return x
}

func (s *tatSim) u(name string) uint32 { return uint32(s.v(name)) }

func (s *tatSim) count(name string)     { s.out[name]++ }
func (s *tatSim) call562a50(arg uint32) { s.count("n562a50"); s.a562 = append(s.a562, int64(arg)) }
func tatDword(va uint32) uint32         { return binary.LittleEndian.Uint32(ticRead(va, 4)) }
func tatF32(bits uint32) float32        { return math.Float32frombits(bits) }

func (s *tatSim) now(t tatTimer) uint32 {
	if t.kind == 1 {
		return (s.u("gameDays")*86400000 + s.u("gameTicks")) * 10
	}
	if t.flags&1 != 0 {
		return s.getUTime()
	}
	if s.v("frozenOn") != 0 {
		return s.u("frozenTicks")
	}
	return s.getUTime() - s.u("clockBase")
}

func (s *tatSim) getUTime() uint32 {
	if s.utime >= 4 {
		panic("trap attempt training: more than four getUTime calls")
	}
	s.utime++
	return s.u(fmt.Sprintf("ms%d", s.utime-1))
}

// progress is 0x43a190: the returned float32 bits; t's window may move.
func (s *tatSim) progress(t *tatTimer) uint32 {
	num, den := s.now(*t)-t.start, t.end-t.start
	var bits, n uint32
	switch {
	case den == 0 && num == 0:
		bits = 0xffc00000 // x87 0/0: real indefinite; its ftol is 0
	case den == 0:
		bits = 0x7f800000 // +inf; ftol 0
	default:
		q := new(big.Float).SetPrec(64).SetMode(big.ToNearestEven)
		q.Quo(new(big.Float).SetUint64(uint64(num)), new(big.Float).SetUint64(uint64(den)))
		f, _ := q.Float32()
		bits = math.Float32bits(f)
		n = num / den // the extended quotient never rounds up to the next integer below 2^32
	}
	if n != 0 && int32(n) >= 1 {
		t.start = t.end + (n-1)*t.period
		t.end = t.start + t.period
	}
	fl := t.flags
	if fl&8 != 0 || (s.v("frozenOn") != 0 && fl&1 == 0) || s.v("pausedAll") != 0 ||
		(s.v("pausedGame") != 0 && fl&1 == 0) {
		return tatDword(0x5ebb34)
	}
	return bits
}

// tatFtol is 0x5e1c22 on an x87 value: truncation to int64, low dword; NaN, inf and
// out-of-range values give the integer indefinite 0x8000000000000000 (low dword 0).
func tatFtol(x float64) uint32 {
	if math.IsNaN(x) || x >= 9223372036854775808.0 || x < -9223372036854775808.0 {
		return 0
	}
	return uint32(int64(x))
}

func tatGe1(bits uint32) bool { return tatF32(bits) >= tatF32(tatDword(0x5ebb38)) }

func (s *tatSim) elig() bool {
	s.count("n524")
	ac := s.v("activeChar")
	s.out["a524"] = ac
	if s.real {
		return characterActionEligibilityNode([]int64{ac, s.v(fmt.Sprintf("slot%d", ac)), s.v("elHP"), s.v("elStatus")}, s.r)[0] != 0
	}
	return s.u("ret524")&0xff != 0
}

func (s *tatSim) call4ff() {
	if x := s.v("ret4ff"); x < 24 || x > 27 {
		panic(fmt.Sprintf("trap attempt training: 0x4ff7f0 return %d outside 24..27", x))
	}
	s.count("n4ff")
}

// spellCheck is the 0x524a10 gate on +0xc69 / +0xc15, the known flags (+0xbcd+4sp) of spells
// 0x27 / 0x12, shared by 0x589d90 and 0x58a200.
func (s *tatSim) spellCheck() bool {
	if s.elig() && s.u("known27") == 1 {
		s.call4ff()
		return true
	}
	if s.elig() && s.u("known12") == 1 {
		s.call4ff()
		return true
	}
	return false
}

// usable is 0x4f9750(ch, sp) for sp 0x12 / 0x27.
func (s *tatSim) usable(sp string) bool {
	realm := s.v("realm" + sp)
	if realm > 5 {
		panic(fmt.Sprintf("trap attempt training: realm%s=%d outside 0..5", sp, realm))
	}
	return s.u("known"+sp) == 1 && int32(s.v(fmt.Sprintf("realmSkill%d", realm))) >= int32(s.v("req"+sp))
}

func (s *tatSim) clearControls() {
	for _, c := range []string{"ctl1", "ctl2", "ctl3"} {
		s.out[c] = 0
	}
}

// screenReturn is 0x58f6b0(0) .. 0x562a50(0x1000), shared by states 3, 10 and 0x58a200.
func (s *tatSim) screenReturn() {
	s.call562a50(0x800)
	s.count("n562580")
	s.out["a562580_1"] = s.v("screenMode")
	s.out["a562580_2"] = 1
	for _, a := range []uint32{0x200, 0x100, 0x1000} {
		s.call562a50(a)
	}
}

// teardown is the state-10 block (also the end of state 3): the panel 0x68f2d4 is destroyed.
func (s *tatSim) teardown() {
	s.out["busy99"] = 0
	s.out["ctl1"], s.out["ctl3"] = 0, 0 // destructors of the child widget and the button strip
	s.out["curPanel"] = 0
	s.screenReturn()
}

func (s *tatSim) refresh(post319, post31f int64) {
	v := int32(-1)
	if s.elig() && (post319 != 0 || post31f != 0) {
		v = int32(post31f)
	}
	mask := int64(ticRead(0x5ed570, 1)[0])
	for k := 0; k < 8; k++ {
		if s.v(fmt.Sprintf("btnFlag%d", k))&mask == 0 {
			sel := int64(0)
			if v < 0 {
				sel = 1
			}
			s.out[fmt.Sprintf("btnSel%d", k)] = sel
		}
	}
	if s.spellCheck() {
		s.usable("27")
		s.usable("12")
	}
	s.elig() // 0x588e60 via 0x589090
	if s.elig() && s.u("known27") == 1 {
		s.call4ff()
	}
	if s.elig() && s.u("known12") == 1 {
		s.call4ff()
	}
}

func tatRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(Nodes["00589a80"].In) {
		panic("trap attempt training: wrong input count")
	}
	s := &tatSim{in: in, r: r, real: real, out: map[string]int64{}}
	for _, n := range []string{"state", "sound", "t0Start", "t0End", "t0Period", "t0Secs", "t1Start", "t1End",
		"t2Start", "t2End", "c84", "c8c", "c13c", "c141", "child5", "child34", "trigSprung", "linkFlags",
		"ctl1", "ctl2", "ctl3"} {
		s.out[n] = s.v(n)
	}
	for k := 0; k < 8; k++ {
		n := fmt.Sprintf("btnSel%d", k)
		s.out[n] = s.v(n)
	}
	s.out["busy99"], s.out["spellMode"], s.out["curPanel"] = s.v("busyIn"), s.v("spellModeIn"), tatPanel
	ac := s.v("activeChar")
	if ac > 7 {
		panic(fmt.Sprintf("trap attempt training: activeChar=%d outside 0..7", ac))
	}
	t0 := tatTimer{s.u("t0Kind"), s.u("t0Flags"), s.u("t0Start"), s.u("t0End"), s.u("t0Period")}
	t1 := tatTimer{s.u("t1Kind"), s.u("t1Flags"), s.u("t1Start"), s.u("t1End"), s.u("t1Period")}
	t2 := tatTimer{s.u("t2Kind"), s.u("t2Flags"), s.u("t2Start"), s.u("t2End"), s.u("t2Period")}
	s.body(&t0, &t1, &t2)
	s.out["t0Start"], s.out["t0End"], s.out["t0Period"] = int64(t0.start), int64(t0.end), int64(t0.period)
	s.out["t1Start"], s.out["t1End"] = int64(t1.start), int64(t1.end)
	s.out["t2Start"], s.out["t2End"] = int64(t2.start), int64(t2.end)
	if len(s.a562) > 16 {
		panic("trap attempt training: more than 16 recorded 0x562a50 calls")
	}
	res := make([]int64, 0, len(tatOutNames)+16)
	for _, n := range tatOutNames {
		res = append(res, s.out[n])
	}
	for k := 0; k < 16; k++ {
		x := int64(0)
		if k < len(s.a562) {
			x = s.a562[k]
		}
		res = append(res, x)
	}
	return res
}

func (s *tatSim) body(t0, t1, t2 *tatTimer) {
	state := s.u("state")
	if state == 9 && tatGe1(s.progress(t0)) {
		free := true
		for k := 0; k < 8; k++ {
			if s.v(fmt.Sprintf("slot%d", k)) != 0 && s.v(fmt.Sprintf("voice%d", k)) != 0 {
				free = false
				break
			}
		}
		if free {
			s.count("n43d940")
			s.out["a43d940"] = 0xffffffff
			state = 10
			s.out["state"] = 10
		}
	}
	if state == 10 {
		s.teardown()
		return
	}
	if s.v("c140") != 0 {
		if n := tatFtol(float64(tatF32(s.progress(t2)))); n != 0 {
			s.out["c13c"] = int64(uint32(s.out["c13c"]) + n)
			s.out["c141"] = 1
		}
	}
	if s.v("c84") != 0 {
		p := s.progress(t1)
		if !(tatF32(p) >= tatF32(s.u("c88"))) {
			s.out["c8c"] = int64(p)
			return
		}
		s.out["c84"], s.out["child5"], s.out["child34"] = 0, 1, 1
		s.call562a50(0x80000000)
		mask := int64(ticRead(0x5ed570, 1)[0])
		for k := 0; k < 8; k++ {
			if s.v(fmt.Sprintf("btnFlag%d", k))&mask == 0 {
				s.out[fmt.Sprintf("btnSel%d", k)] = 0
			}
		}
	}
	switch state {
	case 1:
		s.out["state"] = 0
		s.count("n58a060")
	case 2:
		s.out["state"], s.out["sound"] = 0, 0
		idx := s.v("charIdx")
		s.count("n553")
		s.out["a553_1"] = int64(uint32(tatArena + idx*0x1862))
		s.out["a553_2"], s.out["a553_3"], s.out["a553_4"] = 10, 1, 0
		post319, post31f := s.v("post319"), s.v("post31f")
		if s.real {
			post319, post31f = s.growth(idx, post319, post31f)
		}
		s.refresh(post319, post31f)
		trap := s.u("trapType")
		for i := uint32(0); i < 8; i++ {
			if tic5e3730(trap, i) != 0 && s.v(fmt.Sprintf("ident%d", i)) == 0 {
				return
			}
		}
		s.out["state"] = 3
	case 3:
		s.clearControls()
		if tt := s.u("trigType"); tt > 14 {
			panic(fmt.Sprintf("trap attempt training: trap type %d outside the message table", tt))
		}
		s.out["trigSprung"] = 1
		switch lp := s.u("linkPtr"); lp {
		case 0:
		case tatLink:
			if s.v("linkKind") == 0x0a {
				s.out["linkFlags"] &= 0xfb
			}
		default:
			panic(fmt.Sprintf("trap attempt training: link pointer %#x is not the case's link object", lp))
		}
		if s.r.R(100) < 40 {
			s.count("n52e5c0")
			s.out["a52e5c0_1"] = s.v("voiceMgr")
			s.out["a52e5c0_2"] = 0xffffffff
			s.out["a52e5c0_3"] = 0
			s.out["a52e5c0_4"] = int64(tatDword(0x5ed8c8))
		}
		s.count("n5299b0")
		s.count("n43d940")
		s.out["a43d940"] = 0xffffffff
		s.teardown()
	case 4:
		s.out["sound"] = 0
		s.count("n408860")
		s.out["a408860_1"], s.out["a408860_2"] = tatSnd, 0
		s.clearControls()
		s.count("n5e3ab0")
		s.out["a5e3ab0"] = tatTrig
		s.out["state"] = 9
		const secs = 0x40000000 // 0x589d0e: 2.0f, above 0.0f so it is stored
		s.out["t0Secs"] = secs
		x := float64(tatF32(s.u("t0Rate"))) * float64(tatF32(secs)) * float64(tatF32(tatDword(0x5ec0a8)))
		t0.period = tatFtol(x) // exact: 24+1+14 significant bits
		t0.start = s.now(*t0)
		t0.end = t0.start + t0.period
	case 5:
		s.out["state"] = 0
		if !s.spellCheck() || !(s.usable("27") || s.usable("12")) {
			return
		}
		school := int64(0x27)
		if s.usable("12") {
			school = 0x12
		}
		s.out["busy99"] = 0
		s.clearControls()
		s.out["spellMode"] = 1
		s.screenReturn()
		s.count("n5a0")
		s.out["a5a0_1"], s.out["a5a0_2"], s.out["a5a0_3"] = school, 0xffffffff, 0xffffffff
	case 6:
		s.out["state"] = 0
		s.count("n58a3e0")
	case 7, 8:
		if tatGe1(s.progress(t0)) {
			s.out["state"] = 4
			if state == 7 {
				s.out["state"] = 3
			}
		}
	}
}

// growth is F's real 0x553f10(character idx, 10, 1, 0): the skill-10 flag and effective
// value it leaves replace the inputs when idx is the character 0x589d90 reads.
func (s *tatSim) growth(idx, post319, post31f int64) (int64, int64) {
	v := make([]int64, len(tatSgIdx))
	for i, j := range tatSgIdx {
		if j >= 0 {
			v[i] = s.in[j]
		}
	}
	i := 0
	(&n553In{}).bind(func(name string, _ *int64, _ int) {
		switch name {
		case "slot":
			v[i] = idx
		case "charAddr":
			v[i] = int64(uint32(tatArena + idx*0x1862))
		case "s":
			v[i] = 10
		case "n":
			v[i] = 1
		case "quiet":
			v[i] = 0
		}
		i++
	})
	g := naturalSkillGrowthNode(v, s.r)
	if idx == s.v("activeChar") {
		return g[tatF19d] & 0xff, int64(int32(g[tatEff]))
	}
	return post319, post31f
}
