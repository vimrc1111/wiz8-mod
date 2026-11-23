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

// monsterActionReconsiderChanceNode is 0x4eb5a0 monster_action_reconsider_chance(M),
// cdecl, M = m123 of the case fixture. Dispatch on the action word M+0x2e1
// (jump table 0x4eb7c8, ten entries; above 9 returns 3):
//
//   - 0: 0x540170(M, template) attack resolution; its return is the result.
//     template is 0x4e5720(M), the species-template cache slot *(0x6840c7+4*M.8).
//   - 1, 8, >9: 3.
//   - 2: cast pending byte M+0x2e0 zero is an assert block (0x4eb76e,
//     cov-exempt; calc panics); otherwise 2.
//   - 3: breath, 0x4eba70(M, template), run natively in the case:
//     cstate = *(M+0x10). Byte cstate+0x145 zero returns 2. Otherwise the
//     source descriptor {2, -1, M.id} (0x53be50); unless special[tmpl+0xe3]
//     (monsterTargetInRangeData) is 6: presentation byte +0x1bf = 0,
//     0x4ecc80(src, M+0x2ba), 0x4fd030(0x77, 0, src, M+0x2ba, mons, chars, 0),
//     0x547120(src, mons). By-value copies of both lists (monsters first,
//     buffers new(count*4)) go to 0x5560a0(tmpl+0xe3, src, M+0x2ba, chars, mons).
//     cstate+0x146 = tmpl+0x15c; 0x52c070(M, u32(M.2f)/10, 0); returns 3.
//   - 4, 5, 6, 7, 9: reconsider. 0x48f650(M, 0, 1); presentation P = *(M+0xc);
//     byte P+0x3e zero returns 2. For 5 and 7 with M.hp != 0, u32 M.107 < 0xe and
//     M.87 == 0: 0x51ac30(M, &pick, 3); pick kind 2 calls 0x4c6240(P, 0), kind 3
//     calls 0x4c62c0(P, 0x4e5950(pick id)). 0x505810 runs natively (two passes of
//     0x5049c0 over the active monster list, 0x593330/0x53bf80 when byte
//     0x683f94 is set); its eax is overwritten by fnstsw at 0x4eb668.
//     v = ftol(P.134/P.130 * 100.0) when float P.130 > 0.0, else ftol(0.0*100.0);
//     t = v if u32 v < 100 else 100. R(100) < 100-t reconsiders: unless the
//     action is 9, 0x531540(M) rechooses; if the new action is < 4 or > 7 and not
//     9 (signed), cstate.phase = min(u32(phase+G.8), 100), byte cstate+4 = 0 and
//     0x4f0480(cstate, G.8). Returns 3.
//
// Every path ends with: if G+0x7b8 == M and G+0x7b0 != result, G+0x7b0 = result.
//
// Stubs: 0x48f650, 0x48f800, 0x51ac30, 0x4c6240, 0x4c62c0, 0x593330, 0x5471d0
// (排除) and 0x5049c0, 0x53bf80, 0x523c00, 0x5560a0, 0x4fd030, 0x531540,
// 0x540170 (D4) are address stubs; arguments other than stack pointers and
// call counts are outputs. Only 0x540170's return is read.
//
// Emu (TestEmu) replays the stubs: 0x540170's return from attackResult,
// 0x531540's write of M+0x2e1 from newAction, 0x4fd030's lists from
// charCount/char0..7 and monCount/mon0..7. F calls the real nodes for those
// three with the remaining world parked (first boundary rows of their cases),
// and feeds forward the D4 outputs that 0x4eb5a0 reads or reports (M+0x2e1,
// M+0x33, cstate+4, the list ids with the 0x4fd030 buffer growth replay).
// 0x536190, 0x547080 and 0x4f0480 run natively in the case and are their D4
// nodes in both.
//
// Fixture caps (calc panics past them): list counts <= 8, list monster ids
// 123..125 (any other id hits the not-found assert in 0x4e5550), group leader
// 123..125, character target index 0..7.
const (
	mrcArena  = 0x20000000
	mrcM      = mrcArena + 0xe200
	mrcTmpl   = mrcArena + 0xf200
	mrcPres   = mrcArena + 0xee00
	mrcTgt    = mrcM + 0x2ba
	mrcHeap   = 0x50000000
	mrcListVT = 0x5ebfe0
	mrcMaxIDs = 8
)

// mrcMon is the fixture's monster and presentation record per id.
var mrcMon = map[int64][2]int64{
	123: {mrcM, mrcPres}, 124: {mrcArena + 0xe600, mrcArena + 0xf700}, 125: {mrcArena + 0xea00, mrcArena + 0xf740},
}

var mrcIn = []string{"action", "castPending", "hp", "maxStamina", "stamina", "moveLimit", "summonOwner", "exhaustion",
	"removed", "active", "disposition", "group", "topStatus", "tgtKind", "tgtChar", "tgtMonster", "tgtGroup",
	"m124.group", "m124.hp", "m124.removed", "m124.active", "m125.group", "m125.hp", "m125.removed", "m125.active",
	"presShown", "presTotalBits", "presDoneBits", "presBreathFlag", "phase", "phaseFlag", "breathReady",
	"breathCooldown", "special", "breathCooldownTemplate", "phaseStep", "dispatch", "dispatchMonster",
	"partyActionMode", "cameraMode", "cameraCharacter", "partyAutoTarget", "group.id", "group.members",
	"group.active", "group.leader", "pickKind", "pickId", "newAction", "attackResult", "charCount",
	"char0", "char1", "char2", "char3", "char4", "char5", "char6", "char7",
	"monCount", "mon0", "mon1", "mon2", "mon3", "mon4", "mon5", "mon6", "mon7"}

var mrcOut = mrcOutNames()

func mrcOutNames() []string {
	out := []string{"result", "dispatch", "action", "phase", "phaseFlag", "breathCooldown", "presBreathFlag", "stamina",
		"staminaTier", "s4fd030_calls", "s4fd030_a1", "s4fd030_a2", "s4fd030_a4", "s4fd030_a7", "s5471d0_calls", "s5560a0_calls"}
	for _, a := range []int{1, 3, 4, 5, 6, 7, 8, 9, 10, 11} {
		out = append(out, fmt.Sprintf("s5560a0_a%d", a))
	}
	out = append(out, "s523c00_calls", "s523c00_a1", "s523c00_a2", "s523c00_a3", "s523c00_a4", "s523c00_a6",
		"s48f650_calls", "s48f650_a1", "s48f800_calls", "s51ac30_calls", "s4c6240_calls", "s4c6240_a1", "s4c6240_a2",
		"s4c62c0_calls", "s4c62c0_a1", "s4c62c0_a2", "s5049c0_calls")
	for _, a := range []int{1, 2} {
		for k := 0; k < 4; k++ {
			out = append(out, fmt.Sprintf("s5049c0_a%d_%d", a, k))
		}
	}
	out = append(out, "s593330_calls", "s53bf80_calls", "s531540_calls", "s540170_calls", "s540170_a2")
	for i := 0; i < 16; i++ {
		out = append(out, fmt.Sprintf("heap%02x", 0x30+4*i))
	}
	return out
}

// mrcConst holds the float constants 0.0 (0x5ebb34) and 100.0 (0x5ebcd0).
var mrcConst, mrcConstErr = mrcLoad()

func mrcLoad() (c [2]float32, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return c, fmt.Errorf("monster action reconsider: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	for i, va := range []uint32{0x5ebb34, 0x5ebcd0} {
		o, e := peOffset(b, va)
		if e != nil {
			return c, e
		}
		if len(b)-o < 4 {
			return c, fmt.Errorf("monster action reconsider: truncated constant %#x", va)
		}
		c[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))
	}
	return c, nil
}

// mrc540170Park and the other parks: first boundary row of each callee case.
// mrc540170Park is re/emu/cases/00540170.json boundary[0].
var mrc540170Park = [408]int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 5, 3, 0, 3, 0, 0, 0, 0, 0, 50, 50, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 50, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 50, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 50, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 50, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 50, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 50, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 50, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 3, 1, 6, 0, 0, 0, 0, 0, 0, 0, 0, 13, 396, 0, 15, 15, 15, 15, 15, 15, 15, 0, 0, 0, 396, 0, 15, 15, 15, 15, 15, 15, 15, 0, 0, 0, 20, 20, 20, 20, 20, 0, 0, 1, 0, 0, 30, 0, 0, 0, 0, 0, 0, 1, 0, 0, 30, 0, 0, 0, 0, 0, 0, 1, 0, 0, 30, 0, 0, 0, 0, 0, 0, 0, 0, 30, 20, 20, 10000, 0, 0, 0, 0, 0, 0, 0, 20, 20, 0, 0, 30, 20, 20, 10000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 20, 20, 0, 0, 0, 3, 0, 6, 3, 0, 6, 3, 0, 6, 3, 0, 6, 3, 0, 6, 3, 0, 6, 3, 0, 6, 3, 0, 6, 0, 0, 50, 100, 0, 1, 0, 3, 1, 6, 1, 1, 0, 5, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

type mrcState struct {
	v    map[string]int64
	o    map[string]int64
	r    Rand
	real bool
}

func (s *mrcState) u(n string) uint32 { return uint32(s.v[n]) }

func monsterActionReconsiderChanceNode(in []int64, r Rand) []int64 { return mrcRun(in, r, true) }

func mrcRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(mrcIn) {
		panic("monster action reconsider: input arity")
	}
	s := &mrcState{v: map[string]int64{}, o: map[string]int64{}, r: r, real: real}
	for i, n := range mrcIn {
		s.v[n] = in[i]
	}
	for _, n := range []string{"dispatch", "action", "phase", "phaseFlag", "breathCooldown", "presBreathFlag", "stamina"} {
		s.o[n] = s.v[n]
	}
	var res int64
	switch a := s.u("action"); {
	case a == 0:
		res = s.attack()
	case a == 2:
		if s.v["castPending"]&0xff == 0 {
			panic("monster action reconsider: cast pending byte zero asserts (0x4eb76e)")
		}
		res = 2
	case a == 3:
		res = s.breath()
	case a >= 4 && a <= 9 && a != 8:
		res = s.reconsider(a)
	default:
		res = 3
	}
	s.o["result"] = res
	if s.u("dispatchMonster") == mrcM && s.o["dispatch"] != res {
		s.o["dispatch"] = res
	}
	o := make([]int64, len(mrcOut))
	for i, n := range mrcOut {
		o[i] = s.o[n]
	}
	return o
}

// attack is the action-0 path: 0x540170(M, template).
func (s *mrcState) attack() int64 {
	s.o["s540170_calls"], s.o["s540170_a2"] = 1, mrcTmpl
	if !s.real {
		return int64(s.u("attackResult"))
	}
	live := map[string]int64{"M.kind": s.v["action"], "M.tgtType": s.v["tgtKind"], "M.tgtIdx": s.v["tgtChar"],
		"M.tgtId": s.v["tgtMonster"], "M.side": s.v["disposition"], "M.107": s.v["topStatus"], "M.group": s.v["group"],
		"M.HP": s.v["hp"], "M.maxSP": s.v["maxStamina"], "M.SP": s.v["stamina"], "M.exhaustion": s.v["exhaustion"],
		"M.9f": s.v["removed"]}
	o := monsterAttackResolutionNode(mprFeed(mrc540170Park[:], m17In, live), s.r)
	s.o["action"], s.o["stamina"] = o[116], o[118] // M.kind, M.SP
	return o[0]
}

func (s *mrcState) reconsider(a uint32) int64 {
	s.o["s48f650_calls"], s.o["s48f650_a1"] = 1, mrcM
	if s.v["presShown"]&0xff == 0 {
		return 2
	}
	if (a == 5 || a == 7) && s.u("hp") != 0 && s.u("topStatus") < 0xe && s.u("moveLimit") == 0 {
		s.o["s51ac30_calls"] = 1
		switch s.u("pickKind") {
		case 2:
			s.o["s4c6240_calls"], s.o["s4c6240_a1"], s.o["s4c6240_a2"] = 1, mrcPres, 0
		case 3:
			s.o["s4c62c0_calls"], s.o["s4c62c0_a1"] = 1, mrcPres
			s.o["s4c62c0_a2"] = mrcMon[s.v["pickId"]][1] // 0x4e5950: -1 or not found gives 0
		}
	}
	// 0x505810: two passes over the two-entry active list.
	s.o["s5049c0_calls"] = 4
	for k, m := range []int64{mrcM, mrcMon[124][0], mrcM, mrcMon[124][0]} {
		s.o[fmt.Sprintf("s5049c0_a1_%d", k)] = m
		s.o[fmt.Sprintf("s5049c0_a2_%d", k)] = int64(1 - k/2)
	}
	if s.v["partyAutoTarget"]&0xff != 0 {
		s.o["s593330_calls"], s.o["s53bf80_calls"] = 1, 1
	}
	if mrcConstErr != nil {
		panic(mrcConstErr)
	}
	t := mrcFtol(mrcPercent(math.Float32frombits(s.u("presDoneBits")), math.Float32frombits(s.u("presTotalBits"))))
	if t >= 100 {
		t = 100
	}
	if uint32(s.r.R(100)) >= 100-t {
		return 3
	}
	if a != 9 {
		s.choose()
	}
	if a2 := int32(uint32(s.o["action"])); a2 < 4 || (a2 > 7 && a2 != 9) {
		p := s.u("phase") + s.u("phaseStep")
		s.o["phaseFlag"] = 0
		if p > 100 {
			p = 100
		}
		s.o["phase"] = combatPhaseRoundingNode([]int64{s.v["partyActionMode"], int64(p), s.v["phaseStep"]}, s.r)[0]
	}
	return 3
}

// mrcPercent is P.134 / P.130 * 100.0 on the x87 at 64-bit precision, or
// 0.0 * 100.0 when P.130 > 0.0 fails (also unordered).
func mrcPercent(done, total float32) *big.Float {
	zero, hundred := mrcConst[0], mrcConst[1]
	if !(total > zero) {
		return chdMul(chdF(float64(zero)), chdF(float64(hundred)))
	}
	if math.IsNaN(float64(done)) || (math.IsInf(float64(done), 0) && math.IsInf(float64(total), 0)) {
		panic("monster action reconsider: NaN percentage reaches ftol (emulator limitation)")
	}
	q := new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Quo(chdF(float64(done)), chdF(float64(total)))
	return chdMul(q, chdF(float64(hundred)))
}

// mrcFtol is 0x5e1c22: truncate toward zero, low 32 bits; out of int64 range
// stops the emulator, so calc panics.
func mrcFtol(x *big.Float) uint32 {
	i, acc := x.Int64()
	if acc != big.Exact && (x.Cmp(chdF(math.MaxInt64)) >= 0 || x.Cmp(chdF(math.MinInt64)) <= 0) {
		panic("monster action reconsider: ftol overflow (emulator limitation)")
	}
	return uint32(i)
}

// choose is 0x531540(M): the new action word (and in F, byte cstate+4).
func (s *mrcState) choose() {
	s.o["s531540_calls"] = 1
	if !s.real {
		s.o["action"] = int64(s.u("newAction"))
		return
	}
	// Fields that are not 0x4eb5a0 inputs (max HP among them) keep the parked world.
	live := map[string]int64{"m.16": s.v["disposition"] & 0xff, "m.hp": s.v["hp"], "m.2f": s.v["maxStamina"],
		"m.87": s.v["moveLimit"], "m.107": s.v["topStatus"], "m.2e1": s.o["action"], "c.f4": s.o["phaseFlag"] & 0xff}
	t := s.target()
	for i, w := range t[:4] { // words 4..7 (the item pointer) are the parked world's
		live[fmt.Sprintf("m.tgt%d", i)] = w
	}
	o := monsterActionChoiceNode(mprFeed(mprPark531540[:], mchNames_in(), live), s.r)
	s.o["action"], s.o["phaseFlag"] = o[0], o[11]
}

// target is the eight words at M+0x2ba; +0x1c holds the fixture's item pointer.
func (s *mrcState) target() [8]int64 {
	return [8]int64{s.v["tgtKind"], s.v["tgtChar"], s.v["tgtMonster"], s.v["tgtGroup"], 0, 0, 0, mrcArena + 0xf680}
}

// breath is 0x4eba70(M, template).
func (s *mrcState) breath() int64 {
	if s.v["breathReady"]&0xff == 0 {
		return 2
	}
	if monsterTargetInRangeDataErr != nil {
		panic(monsterTargetInRangeDataErr)
	}
	special := s.v["special"] & 0xff
	heap := int64(mrcHeap + 0x30) // after the two 20-byte list buffers
	var mons, chars []int64
	if monsterTargetInRangeData.special[special] != 6 {
		s.o["presBreathFlag"] = 0
		s.aim()
		s.o["s4fd030_calls"], s.o["s4fd030_a1"], s.o["s4fd030_a2"], s.o["s4fd030_a4"], s.o["s4fd030_a7"] = 1, 0x77, 0, mrcTgt, 0
		if s.real {
			t := s.target()
			a := spellAreaTargetSelectionNode(mprFeed(mchParkArea[:], spellAreaInputs, map[string]int64{
				"spellID": 0x77, "powerLevel": 0, "sourceKind": 2, "sourceCharacter": -1, "sourceMonster": 123,
				"ignoreLineOfSight": 0, "monsterInitialCount": 0, "characterInitialCount": 0, "monsterCapacity": 5,
				"characterCapacity": 5, "targetKind": t[0], "targetCharacter": t[1], "targetMonster": t[2], "targetGroup": t[3]}), s.r)
			mons, chars = append([]int64(nil), a[2:2+a[0]]...), append([]int64(nil), a[10:10+a[1]]...)
			// n_4fd030.go buildList: growing past capacity 5 takes a new buffer per entry, monsters first.
			for _, n := range []int64{a[0], a[1]} {
				for i := int64(5); i < n; i++ {
					heap += (4*(i+1) + 7) &^ 7
				}
			}
		} else {
			mons, chars = s.list("monCount", "mon"), s.list("charCount", "char")
		}
		s.mark(mons)
	}
	words := map[int64]int64{}
	alloc := func(ids []int64) int64 {
		p := heap
		heap += max(8, (4*int64(len(ids))+7)&^7) // re/emu/emu.py allocation rounding
		for k, id := range ids {
			words[p+4*int64(k)] = id
		}
		return p
	}
	monData := alloc(mons)
	charData := alloc(chars)
	nc, nm := int64(len(chars)), int64(len(mons))
	for i, x := range []int64{1, special, mrcTgt, mrcListVT, nc, nc, charData, mrcListVT, nm, nm, monData} {
		s.o[mrcOut[15+i]] = x
	}
	for i := int64(0); i < 16; i++ {
		s.o[fmt.Sprintf("heap%02x", 0x30+4*i)] = words[mrcHeap+0x30+4*i]
	}
	s.o["breathCooldown"] = s.v["breathCooldownTemplate"] & 0xff
	s.drain(int64(s.u("maxStamina") / 10))
	return 3
}

func (s *mrcState) list(count, entry string) []int64 {
	n := s.v[count]
	if n < 0 || n > mrcMaxIDs {
		panic("monster action reconsider: list count past the eight-entry fixture")
	}
	ids := make([]int64, n)
	for k := range ids {
		ids[k] = s.v[fmt.Sprintf("%s%d", entry, k)]
	}
	return ids
}

// monster returns the fixture fields of monster id: hp, removed, active, group.
func (s *mrcState) monster(id int64) (hp, removed, active, group int64, ok bool) {
	switch id {
	case 123:
		return s.v["hp"], s.v["removed"], s.v["active"] & 0xff, s.v["group"], true
	case 124, 125:
		p := fmt.Sprintf("m%d.", id)
		return s.v[p+"hp"], s.v[p+"removed"], s.v[p+"active"] & 0xff, s.v[p+"group"], true
	}
	return 0, 0, 0, 0, false
}

// aim is 0x4ecc80(src, M+0x2ba): focus the camera on the breath target.
func (s *mrcState) aim() {
	cam := s.u("cameraMode")
	if cam == 2 {
		return
	}
	kind := s.u("tgtKind")
	x := make([]int64, 29)
	x[0], x[1], x[2], x[8], x[9], x[10] = mrcTgt, int64(kind), s.v["tgtChar"], s.v["tgtMonster"], s.v["tgtGroup"], 7
	// Active list {123, 124}, reserve list {125} (0x683fad, 0x683fb9).
	if id := s.v["tgtMonster"]; id >= 123 && id <= 125 {
		hp, removed, active, _, _ := s.monster(id)
		slot := 11
		if id == 125 {
			slot = 16
		}
		copy(x[slot:], []int64{2, id, hp, removed, active})
	}
	if s.v["group.id"] == s.v["tgtGroup"] { // the one active group record
		copy(x[21:], []int64{2, s.v["group.id"], s.v["group.members"], s.v["group.active"] & 0xff})
	}
	if combatTargetValidNode(x, s.r)[0] == 0 || cam == 0 {
		return // camera mode 0 only proceeds for a character target, whose case is empty
	}
	switch kind {
	case 3:
		s.o["s48f650_calls"], s.o["s48f650_a1"] = 1, mrcMon[s.v["tgtMonster"]][0]
	case 4:
		m, ok := mrcMon[s.v["group.leader"]]
		if !ok {
			panic("monster action reconsider: group leader outside the fixture records")
		}
		s.o["s48f650_calls"], s.o["s48f650_a1"] = 1, m[0]
	case 6:
		s.o["s48f800_calls"] = 1
	}
}

// mark is 0x547120(src, mons).
func (s *mrcState) mark(mons []int64) {
	rec := []int64{2, 0, 0, 123, 123, s.v["summonOwner"], s.v["disposition"] & 0xff, 124, 0, 0, 125, 0, 0}
	if oppositeDispositionNode(rec, s.r)[0] == 0 {
		return
	}
	for _, id := range mons {
		if id == 123 { // the source's own id (0x53bf10)
			continue
		}
		_, _, _, group, ok := s.monster(id)
		if !ok {
			panic("monster action reconsider: monster id outside the fixture records")
		}
		if group != 0 {
			s.o["s5471d0_calls"]++
		}
	}
}

// drain is 0x52c070(M, dmg, 0): stamina loss and tier.
func (s *mrcState) drain(dmg int64) {
	if s.u("hp") == 0 {
		return
	}
	sp := uint32(s.o["stamina"])
	if dmg > 0 {
		if uint32(dmg) >= sp {
			sp = 0
		} else {
			sp -= uint32(dmg)
		}
	}
	s.o["stamina"] = int64(sp)
	if s.u("maxStamina") == 0 {
		panic("monster action reconsider: max stamina zero divides by zero (0x52c0b8)")
	}
	c := int32(100 - sp*100/s.u("maxStamina"))
	var tier int64
	switch {
	case c < 50:
		tier = 0
	case c < 70:
		tier = 1
	case c < 85:
		tier = 2
	case c >= 95:
		tier = 4
	default:
		tier = 3
	}
	s.o["staminaTier"] = tier
	if sp == 0 && s.u("exhaustion") < 0x270f {
		s.o["s523c00_calls"], s.o["s523c00_a1"], s.o["s523c00_a2"], s.o["s523c00_a3"] = 1, 123, 0x11, 0x270f
		s.o["s523c00_a4"], s.o["s523c00_a6"] = 0, 1
	}
}

func init() {
	Nodes["004eb5a0"] = Node{Addr: "004eb5a0", In: mrcIn, Out: mrcOut, F: monsterActionReconsiderChanceNode,
		Emu: func(in []int64, r Rand) []int64 { return mrcRun(in, r, false) }}
}
