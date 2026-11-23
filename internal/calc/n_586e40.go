package calc

import (
	"fmt"
	"math"
	"strconv"
)

// force_lock_chance 0x586e40: the lock-forcing dialog (strength attempt). Sibling
// of lockSpellPinChanceNode (0x5871a0); `this` is the same dialog.
//
// Chance gate: the active character (global 0x68518d) must pass the action
// eligibility check 0x524a10, have stamina +0xb1d >= 0x50 (signed) and strength
// +0xe9 > 0x32 (unsigned). The chance in percent is (strength-0x32) divided
// (unsigned) by 2^(c-2), c = clamp(difficulty 0x6850d5 + pinCount this+0xc - 1,
// 2, 8) with 32-bit wrap; a quotient with the high bit set (negative as i32)
// ends the function without spending anything. Otherwise 0x52af50(ci, 0x50, 0, 0)
// spends 80 stamina and one R(100) is rolled: roll < chance succeeds.
//
// Success: the sound, then the message "<name> <success text>" is queued by
// 0x5299b0 (3 queue entries, or 1 when the message-suppress bit 6 at 0x652dac
// is set), the control-table cells (0x61f238 + 12*handle) of the pin container
// (handle 1) and the end widget (handle 2) are cleared, this+0x34 becomes 8 and
// the result timer (this+0x80) is set to 1.0 second and restarted (the same
// timer arithmetic as the trap dialog 0x58a930).
//
// Failure: the same with the fail text, no timer and no table change; then the
// dialog is refreshed: the pin children get the tool flag (eligible and the tool
// value at +0x31f, taken when +0x319 or +0x31f is non-zero, is non-negative),
// the spell-point flag widget D20 (learned flag +0xc69 == 1 and realm SP covers
// the cost of spell 0x27, via 0x4f9750) and the strength flag widget D24 (same
// chance test as above, signed non-negative) are set, 0x585ed0 refreshes the
// text panel and the end widget's slot-1 method runs (process flag bit when the
// UI mode is 7).
//
// Stubs (case): 0x524a10 is called once, or seven times on the failure path
// (0x586e57, 0x586fcf, 0x587068, 0x5870f7 and 0x585ed0's three); the returns
// el0..el6 are case inputs and every call's character argument is an output.
// 0x4ff7f0 (at most twice: here and in 0x585ed0) returns a skill index that only
// feeds a dead value (the sign test after the shift can never be taken), so
// sk0/sk1 are ignored. 0x52af50 is stubbed with its four arguments and the call
// count as outputs; the caller ignores its eax and the stub writes nothing, so
// Emu sees the stamina unchanged by the 80 spent.
//
// Emu (what TestEmu replays) takes e_k from el0..el6. F calls the real callees:
// characterActionEligibilityNode on the F-only inputs f_slotActive, f_hp and
// f_mainStatus, and characterFatigueApplyNode on f_staMax .. f_g598 whose stamina
// result feeds the later stamina tests. F assumes the spend does not change the
// eligibility inputs (the 0x522fe0 state it may apply is not modelled), and its
// panics are those of the callees (zero maximum stamina, encumbrance past 4).
//
// Fixture caps (calc panics past them): active character 0..9 whenever the
// eligibility check passes (ten records), pin children <= 8, realm index <= 5.
// The message strings are one UTF-16 unit (nameUnit, failUnit, successUnit; a
// zero unit is the empty string), and the queue holds 64 entries.
const (
	flcRecords = 10
	flcPins    = 8
	flcRealms  = 6
	flcKinds   = 7 // 0x524a10 calls on the failure path
)

// input positions
const (
	flcPinCount = iota
	flcCharIndex
	flcEl0
	_
	_
	_
	_
	_
	flcEl6
	flcSk0
	flcSk1
	flcStamina
	flcStrength
	flcDifficulty
	flcToolByte
	flcToolValue
	flcLearned
	flcRealmSP0
	_
	_
	_
	_
	flcRealmSP5
	flcSpellRealm
	flcSpellCost
	flcUICount
	flcAnchor20
	flcAnchor24
	flcUIMode
	flcNameUnit
	flcFailUnit
	flcSuccessUnit
	flcMsgQuiet
	flcDlgState
	flcTimerKind
	flcGameDays
	flcGameTicks
	flcFrozenClock
	flcTimerRate
	flcTimerStart0
	flcTimerEnd0
	flcTimerDur0
	flcTimerSec0
	flcCtl0
	_
	_
	flcCtl3
	flcFSlotActive
	flcFHP
	flcFStatus
	flcFStaMax
	flcFEnc
	flcFA15
	flcFAA9
	flcFLevel
	flcFFlag
	flcFA45
	flcFG8c8
	flcFG914
	flcFG598
	flcBPinCount
	flcBaseSkill0
)

const (
	flcToolSkill   = 10 // skill 10's first dword (C+0x31f) is the tool value
	flcSkills      = 27 // skill-index ID range of the 0x4ff7f0 result (calc panics past it)
	flcRealmSkill0 = flcBaseSkill0 + flcSkills - 1
	flcInputs      = flcRealmSkill0 + flcRealms
)

func init() {
	in := []string{"pinCount", "charIndex"}
	for k := 0; k < flcKinds; k++ {
		in = append(in, fmt.Sprintf("el%d", k))
	}
	in = append(in, "sk0", "sk1", "stamina", "strength", "difficulty", "toolByte", "toolValue", "learned")
	for r := 0; r < flcRealms; r++ {
		in = append(in, fmt.Sprintf("realmSP%d", r))
	}
	in = append(in, "spellRealm", "spellCost", "uiCount", "anchor20", "anchor24", "uiMode",
		"nameUnit", "failUnit", "successUnit", "msgQuiet", "dlgState",
		"timerKind", "gameDays", "gameTicks", "frozenClock", "timerRate",
		"tStart0", "tEnd0", "tDur0", "tSec0", "ctl0", "ctl1", "ctl2", "ctl3",
		"f_slotActive", "f_hp", "f_mainStatus", "f_staMax", "f_enc", "f_a15", "f_aa9",
		"f_level", "f_flag", "f_a45", "f_g8c8", "f_g914", "f_g598", "bPinCount")
	for j := 0; j < flcSkills; j++ {
		if j != flcToolSkill {
			in = append(in, fmt.Sprintf("baseSkill%d", j))
		}
	}
	for r := 0; r < flcRealms; r++ {
		in = append(in, fmt.Sprintf("realmSkill%d", r))
	}
	out := []string{"msgCount", "msgW0", "msgW1", "msgW2", "n52af50", "f_arg1", "f_arg2", "f_arg3", "f_arg4", "n524a10"}
	for k := 0; k < flcKinds; k++ {
		out = append(out, fmt.Sprintf("e_arg%d", k))
	}
	out = append(out, "n4ff7f0")
	for a := 2; a <= 5; a++ {
		for k := 0; k < 2; k++ {
			out = append(out, fmt.Sprintf("k_arg%d_%d", a, k))
		}
	}
	for i := 0; i < flcPins; i++ {
		out = append(out, fmt.Sprintf("pinFlag%d", i))
	}
	out = append(out, "A_5", "A_34", "C_5", "C_34", "D20_4", "D20_a4", "D24_4", "D24_a4", "dlgStateOut", "processFlags",
		"tStart", "tEnd", "tDur", "tSec", "ctlOut0", "ctlOut1", "ctlOut2", "ctlOut3")
	Nodes["00586e40"] = Node{Addr: "00586e40", In: in, Out: out,
		F:   func(in []int64, r Rand) []int64 { return forceLockChance(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return forceLockChance(in, r, false) }}
}

func forceLockChanceNode(in []int64, r Rand) []int64 { return forceLockChance(in, r, true) }

// flcTimer is the result timer after 0x439b80(1.0f) and 0x439d80 (see the trap
// dialog node 0x58a930): duration trunc(rate * 1.0 * 0x5ec0a8 factor), start the
// frozen clock or, for kind 1, the game calendar, end = start + duration.
func flcTimer(kind, days, ticks, frozen, rateBits uint32) (start, end, dur uint32) {
	if trapTimerScaleErr != nil {
		panic(trapTimerScaleErr)
	}
	d := math.Trunc(float64(math.Float32frombits(rateBits)) * trapTimerScale)
	if math.IsNaN(d) || math.IsInf(d, 0) || d < -9223372036854775808.0 || d >= 9223372036854775808.0 {
		panic("force lock chance: timer duration outside finite i64 conversion")
	}
	dur = uint32(int64(d))
	start = frozen
	if kind == 1 {
		start = (days*86400000 + ticks) * 10
	}
	return start, start + dur, dur
}

// flcChance is (strength - 0x32) / 2^(clamp(difficulty+pins-1, 2, 8) - 2), the
// unsigned quotient the original tests as signed.
func flcChance(strength uint32, difficulty, pins int32) int32 {
	c := difficulty + pins - 1
	if c < 2 {
		c = 2
	} else if c > 8 {
		c = 8
	}
	return int32((strength - 0x32) / (uint32(1) << uint(c-2)))
}

func forceLockChance(in []int64, r Rand, real bool) []int64 {
	if len(in) != flcInputs {
		panic(fmt.Sprintf("force lock chance: expected %d inputs, got %d", flcInputs, len(in)))
	}
	ci := int32(in[flcCharIndex])
	pins, difficulty := int32(in[flcPinCount]), int32(in[flcDifficulty])
	stamina, strength := int32(in[flcStamina]), uint32(in[flcStrength])

	var (
		buf                 [16]int64 // UTF-16 units of the shared format buffer 0x689fd0
		nFat                int64
		fatArg              [4]int64
		nElig, nSkill       int64
		eligArg             [flcKinds]int64
		skillArg            [4][2]int64
		pinFlag             [flcPins]int64
		a5, a34, c5, c34    = int64(0x5a), int64(0x5a), int64(0x5a), int64(0x5a)
		d20, d24            = [2]int64{0x5a, 0x33}, [2]int64{0x5a, 0x33}
		dlgState            = in[flcDlgState] & 0xffffffff
		process             = int64(0x10)
		tStart, tEnd, tDur  = in[flcTimerStart0] & 0xffffffff, in[flcTimerEnd0] & 0xffffffff, in[flcTimerDur0] & 0xffffffff
		tSec                = in[flcTimerSec0] & 0xffffffff
		ctl                 [4]int64
		msgCount            int64
		learned1            = uint32(in[flcLearned]) == 1
		toolByte, toolValue = uint8(in[flcToolByte]), int32(in[flcToolValue])
	)
	for i := range pinFlag {
		pinFlag[i] = 0x5a
	}
	for h := range ctl {
		ctl[h] = in[flcCtl0+h] & 0xffffffff
	}
	eligible := func() bool { // one 0x524a10(ci) call
		k := nElig
		if k >= flcKinds {
			panic("force lock chance: more than seven eligibility calls")
		}
		nElig++
		eligArg[k] = int64(uint32(ci))
		if real {
			return characterActionEligibilityNode([]int64{int64(ci), in[flcFSlotActive], in[flcFHP], in[flcFStatus]}, r)[0] != 0
		}
		return in[flcEl0+int(k)]&0xff != 0
	}
	skill := func() uint32 { // one 0x4ff7f0(char, 0x27, 1, 0, 7) call: the returned skill index
		if nSkill >= 2 {
			panic("force lock chance: more than two skill selection calls")
		}
		skillArg[0][nSkill], skillArg[1][nSkill], skillArg[2][nSkill], skillArg[3][nSkill] = 0x27, 1, 0, 7
		idx := uint32(in[flcSk0+int(nSkill)])
		nSkill++
		return idx
	}
	format := func(units ...int64) { // 0x517a90: write the text and its terminator, keep the rest
		for i, u := range units {
			buf[i] = u
		}
		buf[len(units)] = 0
	}
	percent := func(v int32) { // "%d%%"
		var u []int64
		for _, c := range strconv.FormatInt(int64(v), 10) {
			u = append(u, int64(c))
		}
		format(append(u, '%')...)
	}
	result := func() []int64 {
		o := []int64{msgCount, buf[0] | buf[1]<<16, buf[2] | buf[3]<<16, buf[4] | buf[5]<<16, nFat, fatArg[0], fatArg[1], fatArg[2], fatArg[3], nElig}
		o = append(o, eligArg[:]...)
		o = append(o, nSkill)
		for a := range skillArg {
			o = append(o, skillArg[a][:]...)
		}
		o = append(o, pinFlag[:]...)
		o = append(o, a5, a34, c5, c34, d20[0], d20[1], d24[0], d24[1], dlgState, process, tStart, tEnd, tDur, tSec)
		return append(o, ctl[:]...)
	}

	if !eligible() { // 0x586e57
		return result()
	}
	if ci < 0 || ci >= flcRecords {
		panic("force lock chance: active character outside the ten fixture records")
	}
	if stamina < 0x50 || strength <= 0x32 { // 0x586e7a, 0x586e8d
		return result()
	}
	chance := flcChance(strength, difficulty, pins)
	if chance < 0 { // 0x586ed3
		return result()
	}
	staminaLeft := stamina
	nFat, fatArg = 1, [4]int64{int64(uint32(ci)), 0x50, 0, 0}
	if real {
		f := characterFatigueApplyNode([]int64{int64(ci), 0x50, 0, 0, int64(stamina), in[flcFHP], in[flcFStaMax],
			in[flcFEnc], in[flcFA15], in[flcFAA9], in[flcFLevel], in[flcFFlag], in[flcFA45], 0,
			in[flcFG8c8], in[flcFG914], in[flcFG598], 0, 0, 0}, r)
		staminaLeft = int32(f[0])
	}
	ok := int32(r.R(100)) < chance // 0x586ef5

	text := in[flcFailUnit]
	if ok {
		text = in[flcSuccessUnit]
	}
	var units []int64 // "%s %s": name, a space, the message text (one unit each at most)
	if u := in[flcNameUnit] & 0xffff; u != 0 {
		units = append(units, u)
	}
	units = append(units, 0x20)
	if u := text & 0xffff; u != 0 {
		units = append(units, u)
	}
	format(units...)
	msgCount = 3 // 0x5299b0 queues 0x1d, the text and a second 0x1d unless bit 6 of 0x652dac is set
	if (in[flcMsgQuiet]>>6)&1 != 0 {
		msgCount = 1
	}
	if ok {
		ctl[1], ctl[2] = 0, 0 // 0x4f30f0(handle, 0) through 0x4f2220
		dlgState = 8
		tStart, tEnd, tDur = 0, 0, 0
		s, e, d := flcTimer(uint32(in[flcTimerKind]), uint32(in[flcGameDays]), uint32(in[flcGameTicks]),
			uint32(in[flcFrozenClock]), uint32(in[flcTimerRate]))
		tStart, tEnd, tDur, tSec = int64(s), int64(e), int64(d), int64(math.Float32bits(1))
		return result()
	}

	// failure: the dialog refresh
	tool := false
	if eligible() && (toolByte != 0 || toolValue != 0) {
		tool = toolValue >= 0
	}
	count := int32(in[flcUICount])
	if count > flcPins {
		panic("force lock chance: more than eight pin children")
	}
	for i := int32(0); i < count; i++ {
		pinFlag[i] = 0
		if tool {
			pinFlag[i] = 1
		}
	}
	canSpend := false
	if eligible() && learned1 {
		skill() // the result only feeds a value that is never used here
		realm := uint32(in[flcSpellRealm])
		if realm >= flcRealms {
			panic("force lock chance: realm index past the six realms")
		}
		canSpend = int32(in[flcRealmSP0+int(realm)]) >= int32(in[flcSpellCost])
	}
	d20 = setPinAnchor(canSpend, int32(in[flcAnchor20]))
	strong := false
	if eligible() && staminaLeft >= 0x50 && strength > 0x32 {
		strong = flcChance(strength, difficulty, pins) >= 0
	}
	d24 = setPinAnchor(strong, int32(in[flcAnchor24]))
	// 0x585ed0 refreshes three text lines through 0x517a90 into the same buffer:
	// the tool value, the spell skill score / 5 and the strength chance.
	if eligible() && (toolByte != 0 || toolValue != 0) && toolValue >= 0 {
		percent(toolValue)
	}
	if eligible() && learned1 {
		idx := skill()
		realm := uint32(in[flcSpellRealm])
		if idx >= flcSkills || realm >= flcRealms {
			panic("force lock chance: skill index past 27 or realm index past 6")
		}
		base := uint32(in[flcToolValue])
		switch {
		case idx < flcToolSkill:
			base = uint32(in[flcBaseSkill0+int(idx)])
		case idx > flcToolSkill:
			base = uint32(in[flcBaseSkill0+int(idx)-1])
		}
		percent(int32((base + uint32(in[flcRealmSkill0+int(realm)])*4) / 5))
	}
	if eligible() && staminaLeft >= 0x50 && strength > 0x32 {
		if c := flcChance(strength, difficulty, int32(in[flcBPinCount])); c >= 0 {
			percent(c)
		}
	}
	c5, c34 = 1, 1
	if in[flcUIMode]&0xffffffff == 7 {
		process |= 0x80000000
	}
	return result()
}
