package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// character_condition_apply 0x522fe0(who, st, time, strength, flagB, flagC): put
// state `st` (slot character +0xa01+4*st) on party member `who` for `time`
// (build/wiz8.asm:394671..). The member's presence byte (*0x685178 + 0x106*who)
// must be set; a non-zero strength is only legal for state 7.
//
// Flow: state 7 without strength is a no-op; state 0x12 on a member with class or
// race feature 2 and slot 0xa45 < 7 runs 0x547a50 (cheatDeathHpNode, same group)
// instead. States 2/10/18/19 force time 0x270f, a zero time ends the call
// (states outside 2..19 take the plain route). With 0x687500 set only a notice is
// shown. States 2..15 may be refused by member features (jump tables at exe
// 0x5232ec/0x5232f4 and 0x523308/0x52331c: features 3, 0x1e, 0xe, the latter two
// skipped for the permanent time 0x270f). A longer time replaces the slot; then
// 0x5237e0 recomputes the main state (characterPrimaryConditionNode), the strength
// slot +0xb09 is raised, 0x50e650 rebuilds the member, deaths (0x12) run 0x52abf0
// and states above 0x11 0x538d10(who,0); a changed main state (+0xb01) runs
// 0x52f430, flagC shows the result notice, and a member who cannot act in combat
// runs 0x53aeb0.
//
// Callees: 0x547940 (feature test: class row 0x61507c, race row 0x615130, feature
// 0x1c also via +0xa79), 0x5237e0 (reused), 0x547a50 (this group). Emu address-stubs
// 0x50e650 and 0x538e00 (inside 0x538d10), 0x52abf0, 0x58aad0 (inside the 0x590950
// notice), 0x4c5eb0 (inside 0x53aeb0), 0x52e690 and 0x518150 (inside 0x52f430);
// the arguments of the last call and the call counts are outputs (the 0x58aad0 and
// 0x4c5eb0 stubs keep both calls of a group run). The two 0x518150 stub returns are
// 51 case values. The 0x52e690 arguments and count are not outputs: only the
// random draws and exits of 0x52f430 are modelled (b01 2/7/9/10: R(2); 0x11, 0x12:
// R(3), after the voice flags 0x68c4fa/0x6875f8).
//
// Emu replays the 0x52abf0 stub (no change to the member). F runs the real
// characterDeathApplyNode on a fixture of this node's cells (combat flag, eight
// members present, +0xb11 and +0xb01 of `who`) and feeds the new +0xb11/+0xb01
// forward; the state slots are not fed back because that node does not output them.
//
// Fixture caps (calc panics past them): who 0..7, st 0..19 (no caller passes
// others, slot array is 20 words); two monsters (0x683fad list), their record
// pointers are the fixture objects or 0; the notice stub records two calls.
const (
	ccaChars    = 8
	ccaSlots    = 20
	ccaPCLen    = 5 + 2*pcMembers + 3*pcFormLen
	ccaCharSize = 0x1862
	ccaNotice   = 0x64bf5c // sprintf buffer handed to 0x58aad0
	ccaArena    = 0x20000000
	ccaMonObj   = ccaArena + 262144
	ccaMonRec0  = ccaMonObj + 4096
	ccaMonRec1  = ccaMonObj + 5120
	ccaRecObj   = ccaArena + 69632
	ccaMaxMon   = 2
	ccaMaxCalls = 2

	// W order (inputs after the call parameters).
	ccaWBase    = 0
	ccaWRecPtr  = 1
	ccaWCombat  = 2
	ccaWG7500   = 3
	ccaWG6840   = 4
	ccaWVoice   = 5
	ccaWVClr    = 6
	ccaWMonPtr  = 7
	ccaWMonN    = 8
	ccaWMon     = 9 // 2 x (id, flag, recp, mask)
	ccaWPres    = 17
	ccaWWhoPres = 25
	ccaWCls     = 26
	ccaWRace    = 27
	ccaWNm1     = 28
	ccaWN       = 29 // 15
	ccaWA79     = 44
	ccaWS       = 45 // 20
	ccaWB01     = 65
	ccaWB09     = 66
	ccaWB0d     = 67
	ccaWB11     = 68
	ccaWRecBe   = 69 // 8
	ccaWPC      = 77 // ccaPCLen
	ccaWR518    = ccaWPC + ccaPCLen
	ccaWLen     = ccaWR518 + 51
)

type ccaExe struct {
	jt1b  [18]uint8
	jt1d  [2]uint32
	jt2b  [14]uint8
	jt2d  [5]uint32
	evd   [18]uint32
	r518n int
}

var ccaTab, ccaTabErr = ccaLoad()

func ccaLoad() (t ccaExe, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("condition apply: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	read := func(va uint32, n int) []byte {
		o, e := peOffset(b, va)
		if e != nil || n > len(b)-o {
			err = fmt.Errorf("condition apply: table %#x not in exe", va)
			return make([]byte, n)
		}
		return b[o : o+n]
	}
	copy(t.jt1b[:], read(0x5232f4, 18))
	copy(t.jt2b[:], read(0x52331c, 14))
	for i := range t.jt1d {
		t.jt1d[i] = binary.LittleEndian.Uint32(read(0x5232ec+uint32(4*i), 4))
	}
	for i := range t.jt2d {
		t.jt2d[i] = binary.LittleEndian.Uint32(read(0x523308+uint32(4*i), 4))
	}
	for i := range t.evd {
		t.evd[i] = binary.LittleEndian.Uint32(read(0x52f73c+uint32(4*i), 4))
	}
	if err != nil {
		return t, err
	}
	for _, i := range t.jt1b {
		if int(i) >= len(t.jt1d) {
			return t, fmt.Errorf("condition apply: dispatch index %d out of range", i)
		}
	}
	for _, d := range t.jt1d {
		if d != 0x5230b7 && d != 0x5230f5 {
			return t, fmt.Errorf("condition apply: unknown dispatch target %#x", d)
		}
	}
	for _, i := range t.jt2b {
		if int(i) >= len(t.jt2d) {
			return t, fmt.Errorf("condition apply: refusal index %d out of range", i)
		}
	}
	for _, d := range t.jt2d {
		switch d {
		case 0x52314a, 0x52311d, 0x523161, 0x523169, 0x523196:
		default:
			return t, fmt.Errorf("condition apply: unknown refusal target %#x", d)
		}
	}
	for _, d := range t.evd {
		switch d {
		case 0x52f598, 0x52f571, 0x52f523, 0x52f6ec, 0x52f54a, 0x52f713, 0x52f5d2,
			0x52f5f9, 0x52f4d6, 0x52f479, 0x52f620, 0x52f732:
		default:
			return t, fmt.Errorf("condition apply: unknown state-change target %#x", d)
		}
	}
	return t, nil
}

func ccaWNames() []string {
	var n []string
	add := func(f string, a ...any) { n = append(n, fmt.Sprintf(f, a...)) }
	n = append(n, "charBase", "recPtr", "combat", "g7500", "g6840bb", "voiceOff", "voiceClr", "monPtr", "monN")
	for k := 0; k < ccaMaxMon; k++ {
		add("mon%d.id", k)
		add("mon%d.flag", k)
		add("mon%d.recp", k)
		add("mon%d.mask", k)
	}
	for j := 0; j < ccaChars; j++ {
		add("pres%d", j)
	}
	n = append(n, "whoPres", "cls", "race", "nm1")
	for k := 0; k < 15; k++ {
		add("n%d", k)
	}
	n = append(n, "a79")
	for k := 0; k < ccaSlots; k++ {
		add("s%d", k)
	}
	n = append(n, "b01", "b09", "b0d", "b11")
	for j := 0; j < ccaChars; j++ {
		add("recBe%d", j)
	}
	n = append(n, "interfaceMode", "battle", "battleMask", "screenFlags", "portraitGate")
	for c := 0; c < pcMembers; c++ {
		add("expr%d", c)
	}
	for c := 0; c < pcMembers; c++ {
		add("flag99_%d", c)
	}
	for _, t := range "ABC" {
		for i := 0; i < pcFormLen; i++ {
			add("%c.f%d", t, i)
		}
	}
	for i := 0; i < 51; i++ {
		add("r518150_%d", i)
	}
	return n
}

func ccaOutNames(ret bool) []string {
	var o []string
	add := func(f string, a ...any) { o = append(o, fmt.Sprintf(f, a...)) }
	if ret {
		o = append(o, "ret")
	}
	for k := 0; k < ccaSlots; k++ {
		add("s%d", k)
	}
	o = append(o, "b01", "b09", "b11", "g6840bb", "voiceClr")
	for j := 0; j < ccaChars; j++ {
		add("recBe%d", j)
	}
	for c := 0; c < pcMembers; c++ {
		add("expr%d", c)
	}
	for c := 0; c < pcMembers; c++ {
		add("flag99_%d", c)
	}
	o = append(o, "battleMask", "screenFlags")
	for _, t := range "ABC" {
		for i := 0; i < pcFormLen; i++ {
			add("%c.f%d", t, i)
		}
	}
	o = append(o, "mon0.mask", "mon1.mask", "r538e00_2", "n538e00", "r50e650_1", "n50e650", "r52abf0_1", "n52abf0")
	for a := 1; a <= 3; a++ {
		for k := 0; k < ccaMaxCalls; k++ {
			add("r58aad0_%d_%d", a, k)
		}
	}
	o = append(o, "n58aad0")
	for a := 1; a <= 3; a++ {
		for k := 0; k < ccaMaxCalls; k++ {
			add("r4c5eb0_%d_%d", a, k)
		}
	}
	return append(o, "n4c5eb0")
}

func init() {
	in := append([]string{"who", "st", "time", "strength", "flagB", "flagC"}, ccaWNames()...)
	Nodes["00522fe0"] = Node{Addr: "00522fe0", In: in, Out: ccaOutNames(true),
		F: characterConditionApplyNode, Emu: characterConditionApplyEmu}
}

// ccaS is the shared state of the group: the node inputs (parameters, then the W
// cells), changed in place, and the stub traces the case outputs.
type ccaS struct {
	v    []int64
	o    int
	r    Rand
	real bool
	// last-call arguments and call counts of the stubbed callees
	a538, n538, a50, n50, a52a, n52a int64
	n58, n4c                         int
	r58, r4c                         [3][ccaMaxCalls]int64
	pc                               []int64 // 0x5237e0 inputs after combat (interface mode ..)
}

func (s *ccaS) g(i int) int64      { return s.v[s.o+i] }
func (s *ccaS) put(i int, x int64) { s.v[s.o+i] = x }
func (s *ccaS) u(i int) uint32     { return uint32(s.v[s.o+i]) }

func ccaNew(in []int64, params int, r Rand, real bool) *ccaS {
	if ccaTabErr != nil {
		panic(ccaTabErr)
	}
	if statusRebuildTablesErr != nil {
		panic(statusRebuildTablesErr)
	}
	if len(in) != params+ccaWLen {
		panic("condition apply: wrong input count")
	}
	return &ccaS{v: append([]int64(nil), in...), o: params, r: r, real: real}
}

// chr returns the base of the member's character record (0x685174 + who*0x1862).
func (s *ccaS) chr(who int64) int64 { return s.g(ccaWBase) + who*ccaCharSize }

// notice is 0x590950(who, ..): sprintf, then 0x58aad0(8, 0x64bf5c, name).
func (s *ccaS) notice(who int64) {
	if s.n58 >= ccaMaxCalls {
		panic("condition apply: more notices than the stub records")
	}
	s.r58[0][s.n58], s.r58[1][s.n58], s.r58[2][s.n58] = 8, ccaNotice, s.chr(who)+5
	s.n58++
}

// feat is 0x547940(C, code).
func (s *ccaS) feat(code int32) bool {
	cls, race := s.g(ccaWCls), s.g(ccaWRace)
	if cls < -1 || cls > 14 || race < -1 || race > 15 {
		panic("condition apply: class or race outside the exe feature tables")
	}
	if cls != -1 {
		for _, f := range statusRebuildTables.character.ClassFeat[cls] {
			if int32(f) == code {
				return true
			}
		}
	}
	if race != -1 {
		for _, f := range statusRebuildTables.raceFeatures[race] {
			if f == code {
				return true
			}
		}
	}
	return code == 0x1c && s.u(ccaWA79) > 0
}

// primary is 0x5237e0(who): characterPrimaryConditionNode over the slots, the old
// main state and the pc group; writes the new +0xb01 and the pc group back.
func (s *ccaS) primary(who int64) {
	if s.g(ccaWCombat)&0xff != 0 && s.g(ccaWRecPtr) == 0 {
		panic("condition apply: combat with a null record table 0x6836a8") // 0x5237e0 dereferences it; the original faults
	}
	v := make([]int64, pcForm+3*pcFormLen)
	v[0] = who
	copy(v[1:1+pcSlots], s.v[s.o+ccaWS:s.o+ccaWS+ccaSlots])
	v[pcOld], v[pcCombat] = s.g(ccaWB01), s.g(ccaWCombat)
	copy(v[pcMode:], s.v[s.o+ccaWPC:s.o+ccaWPC+ccaPCLen])
	res := characterPrimaryConditionNode(v, s.r)
	s.put(ccaWB01, res[0])
	pc := s.v[s.o+ccaWPC : s.o+ccaWPC+ccaPCLen]
	copy(pc[5:21], res[1:17])
	pc[2], pc[3] = res[17], res[18]
	copy(pc[21:], res[19:])
}

// death is 0x52abf0(who): stubbed in Emu; F runs characterDeathApplyNode.
func (s *ccaS) death(who int64) {
	s.a52a, s.n52a = who, s.n52a+1
	if !s.real {
		return
	}
	v := make([]int64, len(cdaIn)+1)
	v[0] = who
	for i, c := range cdaIn {
		switch {
		case c.name == "f94":
			v[i+1] = s.g(ccaWCombat) & 0xff
		case c.name[0] == 'p' && len(c.name) == 2:
			v[i+1] = 1
		case c.name == fmt.Sprintf("h%d", who):
			v[i+1] = s.g(ccaWB11)
		case c.name == fmt.Sprintf("t%d", who):
			v[i+1] = s.g(ccaWB01)
		}
	}
	out := characterDeathApplyNode(v, s.r)
	for i, c := range cdaOut {
		switch c.name {
		case fmt.Sprintf("h%d", who):
			s.put(ccaWB11, out[i])
		case fmt.Sprintf("t%d", who):
			s.put(ccaWB01, out[i])
		}
	}
}

// stateChange is 0x52f430(C): the voice flags 0x68c4fa/0x6875f8, then the table
// at exe 0x52f73c by +0xb01-2; only the random draws of the other callees count.
func (s *ccaS) stateChange() {
	if s.g(ccaWVoice)&0xff != 0 {
		return
	}
	if s.g(ccaWVClr)&0xff != 0 {
		s.put(ccaWVClr, 0)
		return
	}
	e := s.u(ccaWB01) - 2
	if e > 0x11 {
		return
	}
	switch ccaTab.evd[e] {
	case 0x52f598:
		s.r.R(2)
	case 0x52f4d6, 0x52f479:
		s.r.R(3)
	}
}

// monsters is 0x53aeb0(who): each listed monster whose record has `who` in its
// target mask loses that bit and gets 0x4c5eb0(who, id, 0).
func (s *ccaS) monsters(who int64) {
	var n uint32
	if s.g(ccaWMonPtr) != 0 {
		if s.g(ccaWMonPtr) != ccaMonObj {
			panic("condition apply: monster list pointer outside the fixture")
		}
		n = s.u(ccaWMonN)
	}
	if n > ccaMaxMon {
		panic("condition apply: more monsters than the two-monster fixture")
	}
	rec := [ccaMaxMon]int64{ccaMonRec0, ccaMonRec1}
	for i := 0; i < int(n); i++ {
		b := ccaWMon + 4*i
		if s.g(b+1)&0xff == 0 || s.g(b+2) == 0 {
			continue
		}
		if s.g(b+2) != rec[i] {
			panic("condition apply: monster record pointer outside the fixture")
		}
		mask := s.g(b+3) & 0xff
		if mask&(1<<uint(who)) == 0 {
			continue
		}
		s.put(b+3, mask&^(1<<uint(who))&0xff)
		if s.n4c >= ccaMaxCalls {
			panic("condition apply: more kill notices than the stub records")
		}
		s.r4c[0][s.n4c], s.r4c[1][s.n4c], s.r4c[2][s.n4c] = who, s.g(b), 0
		s.n4c++
	}
}

func (s *ccaS) output(ret bool, al int64) []int64 {
	var o []int64
	if ret {
		o = append(o, al)
	}
	for k := 0; k < ccaSlots; k++ {
		o = append(o, int64(s.u(ccaWS+k)))
	}
	o = append(o, int64(s.u(ccaWB01)), int64(s.u(ccaWB09)), int64(s.u(ccaWB11)), s.g(ccaWG6840)&0xff, s.g(ccaWVClr)&0xff)
	for j := 0; j < ccaChars; j++ {
		o = append(o, s.g(ccaWRecBe+j)&0xff)
	}
	pc := s.v[s.o+ccaWPC : s.o+ccaWPC+ccaPCLen]
	o = append(o, pc[5:21]...)
	o = append(o, pc[2], pc[3])
	o = append(o, pc[21:]...)
	o = append(o, s.g(ccaWMon+3)&0xff, s.g(ccaWMon+7)&0xff)
	o = append(o, s.a538, s.n538, s.a50, s.n50, s.a52a, s.n52a)
	for a := 0; a < 3; a++ {
		o = append(o, s.r58[a][:]...)
	}
	o = append(o, int64(s.n58))
	for a := 0; a < 3; a++ {
		o = append(o, s.r4c[a][:]...)
	}
	return append(o, int64(s.n4c))
}

func characterConditionApplyNode(in []int64, r Rand) []int64 { return ccaRun(in, r, true) }

func characterConditionApplyEmu(in []int64, r Rand) []int64 { return ccaRun(in, r, false) }

func ccaRun(in []int64, r Rand, real bool) []int64 {
	s := ccaNew(in, 6, r, real)
	al := ccaApply(s, in[0], in[1], uint32(in[2]), uint32(in[3]), in[5]&0xff != 0)
	return s.output(true, al)
}

// ccaApply is 0x522fe0; it returns the al result.
func ccaApply(s *ccaS, who, st int64, tm, strength uint32, flagC bool) int64 {
	if who < 0 || who >= ccaChars || st < 0 || st >= ccaSlots {
		panic("condition apply: who/state outside the eight members and 20 slots") // no caller passes others
	}
	if s.g(ccaWWhoPres)&0xff == 0 {
		panic("condition apply: assert, member absent") // 0x52301e
	}
	if strength != 0 && st != 7 {
		panic("condition apply: assert, strength only for state 7") // 0x523046
	}
	if st == 7 {
		if strength == 0 {
			return 0
		}
	} else if st == 0x12 && s.feat(2) && s.u(ccaWS+17) < 7 {
		chdRun(s, who)
		return 0
	}
	e := uint32(st) - 2
	if e <= 0x11 && ccaTab.jt1d[ccaTab.jt1b[e]] == 0x5230b7 {
		tm = 0x270f
	}
	if tm == 0 {
		return 0
	}
	if s.g(ccaWG7500)&0xff != 0 {
		s.notice(who)
		return 0
	}
	if e <= 0xd {
		tgt := ccaTab.jt2d[ccaTab.jt2b[e]]
		if tgt == 0x52311d {
			if s.feat(3) {
				s.notice(who)
				return 0
			}
			tgt = 0x52314a
		}
		if tgt == 0x52314a {
			if s.feat(0x1e) {
				return 0
			}
		}
		if tgt == 0x523161 && tm != 0x270f {
			tgt = 0x523169
		}
		if tgt == 0x523169 && s.feat(0xe) {
			s.notice(who)
			return 0
		}
	}
	old := s.u(ccaWB01)
	slot := s.u(ccaWS + int(st))
	bl := false
	if tm > slot {
		if slot == 0 {
			switch st {
			case 9, 12:
				s.put(ccaWG6840, 1)
			case 13:
				s.a538, s.n538 = 1, s.n538+1
			}
		}
		s.put(ccaWS+int(st), int64(tm))
		s.primary(who)
		bl = true
	}
	if strength > s.u(ccaWB09) {
		s.put(ccaWB09, int64(strength))
	}
	s.a50, s.n50 = who, s.n50+1
	if !bl {
		return 0
	}
	if st == 0x12 {
		s.death(who)
	}
	if st >= 0x12 {
		s.a538, s.n538 = 0, s.n538+1
	}
	if old != s.u(ccaWB01) {
		s.stateChange()
	}
	if flagC {
		s.notice(who) // flagB only picks the text (state 0x13), not the call
	}
	if !(s.u(ccaWB11) != 0 && s.u(ccaWB01) < 0xd) && s.g(ccaWCombat)&0xff != 0 {
		s.monsters(who)
	}
	return 1
}
