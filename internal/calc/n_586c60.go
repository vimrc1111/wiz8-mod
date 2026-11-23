package calc

import "fmt"

// 0x586c60 lock_pick_pin_chance (thiscall, this = the lock-picking dialog D): one pick attempt
// by party member who = D+0x2c.
//
//   - pass over the D+0xc pin slots: a pin i whose owner dword D+0x38+4i equals who and whose
//     "settled" byte D+0x58+i is zero gets the roll R(100) (0x40efa0, only for such pins). With
//     e = effective lock-picking skill (C+0x31f of character who, record stride 0x1862 at
//     *0x685174) and d = difficulty (*0x6850d5) the threshold is t = max(e - 5d + 5, 0) & 0xff
//     (exe 0x586cbd..0x586cd4; the mask wraps 256 to 0). roll >= t drops the pin: owner := -1,
//     pin object (pointer at P+0x54+4i, P = D+0x10) byte +0x34 := 0 and +0x36 := 1, P+0x74 := 1,
//     and the pin-falling sound plays (not an output);
//   - pick counter D+0x60+4*who += 1; D+0x38+4*D[+0x30] := who (after the pass, so it overrides
//     a pin the pass just freed);
//   - R(5); on 0, 0x4457a0 (counter at *(D+8)+0x368+0x1c: >0 decrements and succeeds) and then
//     0x553f10(C, 10, 1, 0) natural skill growth for skill 10; when C+0x31f changed 0x585ed0
//     refreshes the dialog text (presentation, not modelled);
//   - if no pin among the first D+0xc owners is -1: success sound, 0x58ac00 / 0x4f30f0 x2 /
//     0x439b80 / 0x439d80 dialog teardown (presentation, not modelled) and D+0x34 := 8.
//
// Emu: the case address-stubs the D4 callee 0x553f10 (composition ruling 2026-10-09): C+0x31f
// is written from the input effAfter, its arguments and call count are outputs. F runs the real
// naturalSkillGrowthNode on a callee world made of the F-only inputs cls, raw10, apt10, cnt10,
// f19d10, bonus10 and b8dClass (the stubbed callee reads them in the emulator from a scratch
// object, nothing reads them there), feeds its C+0x31f forward, and ignores effAfter. The
// callee's other inputs are fixed: state 0, creation-screen copy and party flags zero, event
// queue empty, idx75 = -1 (these do not change skill 10's effective value or the roll sequence).
//
// Fixture caps (panic past them): pin count <= 8 (the dword table D+0x38.. holds 8 owners),
// who and D+0x30 in 0..7 (the counter table D+0x60.. and the owner table hold 8 entries; party
// records are the eight at *0x685174), the pin pointer table P+0x54+4i holds the fixture's
// eight distinct pin objects, F: cls <= 14. P+0x48 and (D+0x18)+0x48 are asserted non-zero by
// 0x4f30f0 (srAssertFail otherwise), so a zero there is not a legal state and is not an input.
const (
	lppArena   = 0x20000000
	lppCharLen = 6242 // 0x1862
	lppPins    = 8
)

var lppIn, lppOut = lppNames()

func lppNames() (in, out []string) {
	in = []string{"count", "who", "wr", "diff", "eff", "ctrX"}
	add := func(p string, n int) {
		for i := 0; i < n; i++ {
			in = append(in, fmt.Sprintf("%s%d", p, i))
		}
	}
	add("slot", lppPins)
	add("flag", lppPins)
	add("ctr", lppPins)
	add("q34_", lppPins)
	add("q36_", lppPins)
	in = append(in, "p74", "st34", "effAfter", "cls", "raw10", "apt10", "cnt10", "f19d10", "bonus10", "b8dClass", "retv")
	for _, p := range []string{"slotOut", "ctrOut", "q34Out", "q36Out"} {
		for i := 0; i < lppPins; i++ {
			out = append(out, fmt.Sprintf("%s%d", p, i))
		}
	}
	out = append(out, "p74Out", "st34Out", "ctrXOut", "effOut", "stubCount", "stubArg1", "stubArg2", "stubArg3", "stubArg4")
	return
}

func init() {
	Nodes["00586c60"] = Node{Addr: "00586c60", In: lppIn, Out: lppOut,
		F:   func(in []int64, r Rand) []int64 { return lockPickPin(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return lockPickPin(in, r, false) }}
}

func lockPickPinChanceNode(in []int64, r Rand) []int64 { return lockPickPin(in, r, true) }

func lockPickPin(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(lppIn) {
		panic("lock_pick_pin_chance: wrong input count")
	}
	count, who, wr := int32(in[0]), int32(in[1]), int32(in[2])
	diff, eff, ctrX := uint32(in[3]), uint32(in[4]), int32(in[5])
	if count > lppPins {
		panic("lock_pick_pin_chance: more than 8 pins")
	}
	if who < 0 || who >= lppPins || wr < 0 || wr >= lppPins {
		panic("lock_pick_pin_chance: member or pin index outside 0..7")
	}
	var slot [lppPins]int32
	var flag [lppPins]uint8
	var ctr [lppPins]uint32
	var q34, q36 [lppPins]uint8
	for i := 0; i < lppPins; i++ {
		slot[i], flag[i], ctr[i] = int32(in[6+i]), uint8(in[14+i]), uint32(in[22+i])
		q34[i], q36[i] = uint8(in[30+i]), uint8(in[38+i])
	}
	p74, st34 := uint8(in[46]), int32(in[47])
	edi := int32(eff - 5*diff + 5)
	thr := uint32(0)
	if edi >= 0 {
		thr = uint32(edi) & 0xff
	}
	for i := int32(0); i < count; i++ {
		if slot[i] != who || flag[i] != 0 {
			continue
		}
		if uint32(r.R(100)) < thr {
			continue
		}
		slot[i], q34[i], q36[i], p74 = -1, 0, 1, 1
	}
	ctr[who]++
	slot[wr] = who
	var calls, a1 int64
	effOut := int64(eff)
	if r.R(5) == 0 && ctrX > 0 { // 0x4457a0 only runs after R(5) == 0
		ctrX--
		calls, a1 = 1, lppArena+lppCharLen*int64(who)
		if real {
			effOut = lppGrowth(in, who, eff, r)
		} else {
			effOut = int64(uint32(in[48]))
		}
	}
	done := true
	for i := int32(0); i < count; i++ {
		if slot[i] == -1 {
			done = false
		}
	}
	if done {
		st34 = 8
	}
	var out []int64
	for i := 0; i < lppPins; i++ {
		out = append(out, int64(slot[i]))
	}
	for i := 0; i < lppPins; i++ {
		out = append(out, int64(ctr[i]))
	}
	for i := 0; i < lppPins; i++ {
		out = append(out, int64(q34[i]))
	}
	for i := 0; i < lppPins; i++ {
		out = append(out, int64(q36[i]))
	}
	out = append(out, int64(p74), int64(st34), int64(ctrX), effOut, calls)
	if calls == 0 {
		return append(out, 0, 0, 0, 0)
	}
	return append(out, a1, 10, 1, 0)
}

// lppGrowth runs the real 0x553f10(C, 10, 1, 0) and returns the character's C+0x31f afterwards.
func lppGrowth(in []int64, who int32, eff uint32, r Rand) int64 {
	cls := in[49]
	if cls < 0 || cls > 14 {
		panic("lock_pick_pin_chance: class outside 0..14")
	}
	var v n553In
	v.slot, v.charAddr, v.s, v.n, v.quiet = int64(who), lppArena+lppCharLen*int64(who), 10, 1, 0
	v.class, v.idx75 = cls, -1
	v.raw[10], v.apt[10], v.cnt[10] = in[50], in[51], in[52]
	v.f19d[10], v.bonus[10], v.eff[10] = in[53], in[54], int64(eff)
	v.b8d[cls] = in[55]
	var vec []int64
	v.bind(func(_ string, p *int64, _ int) { vec = append(vec, *p) })
	res := naturalSkillGrowthNode(vec, r)
	var o n553Out
	i := 0
	o.bind(func(_ string, p *int64) { *p = res[i]; i++ })
	return o.eff[10]
}
