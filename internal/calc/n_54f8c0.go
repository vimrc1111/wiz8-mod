package calc

// spellStaminaDrainNode 0x54f8c0 spell_stamina_drain(effect), build/wiz8.asm:455703:
// for each character target (effect +0xf4 count, +0xfc list of party indices)
// whose stamina +0xb1d is non-zero, and each monster target (+0xe4 count, +0xec
// list of monster ids) whose stamina +0x33 is non-zero, roll the effect dice
// (0x517970 on effect +0xb4) and, unless the roll is the 9999 sentinel, add the
// percentage bonus effect +0xd4 (0x517f90: unsigned 32-bit product, /100 by the
// 0x51eb851f reciprocal). Characters spend it through 0x52af50(who, amount, 0,
// 0); monsters through 0x52c070(monster, amount, 0), modelled here: a dead
// monster (+0x2b == 0) is untouched; otherwise a non-zero amount lowers +0x33
// (clamped at 0), +0x242 becomes the 0..4 fatigue step of +0x33 against +0x2f
// (staminaFatigueLevel; a zero maximum is the original's divide fault, calc
// panics), and stamina 0 with condition-17 duration +0x9b below 9999 (unsigned)
// applies condition 0x11 for 9999 turns through 0x523c00(id, 0x11, 0x270f, 0,
// source, 1) with a source record cleared by 0x536150 (13 dwords zero, +4/+8 -1).
//
// The case address-stubs the D4 callees 0x517970 (returns from inputs in call
// order), 0x52af50 and 0x523c00 (both return values are ignored by their
// callers; the case still feeds them from inputs), so outputs are the call
// counts, every call's arguments (the 0x523c00 source pointer is a stack
// address and is not recorded) and the monster stamina/fatigue written by the
// native 0x52c070. Emu replays the stubs; F rolls diceNode and runs
// characterFatigueApplyNode and monsterConditionApplyNode, feeding their
// stamina, fatigue and condition-17 results back to later targets.
//
// Input layout: 0..2 Z, N, S; 3 bonusPercent; 4..5 character and monster
// target counts (i32); 6..8 character targets; 9..11 monster ids; 12..17
// 0x517970 returns in call order; 18..19 0x52af50/0x523c00 returns; 20 + 6k per
// party member k: stamina +0xb1d, hit points +0xb11, maximum stamina +0xb19,
// fatigue level +0x169e, fatigue flag +0x185b, state +0xa45 (the last five are
// F-only); 68 + 4j per monster j (id 100+j): hit points +0x2b, maximum stamina
// +0x2f, stamina +0x33, condition-17 duration +0x9b; 80 template byte +0xcb
// (F-only, monsterConditionApplyNode's tpl.kind).
//
// Fixture caps (panic past them): three targets per list, party indices 0..7,
// monster ids 100..102. Index -1 is the srAssertFail path 0x54f8f2 and a null
// monster pointer the srAssertFail path 0x54f9b7 (notes/re/cov-exempt.d/54f8c0.tsv);
// both panic.
func spellStaminaDrainNode(in []int64, r Rand) []int64 { return spellStaminaDrain(in, r, true) }

func spellStaminaDrainEmu(in []int64, r Rand) []int64 { return spellStaminaDrain(in, r, false) }

func spellStaminaDrain(in []int64, r Rand, real bool) []int64 {
	if len(in) != 81 {
		panic("spell stamina drain: expected 81 inputs")
	}
	u := func(i int) uint32 { return uint32(in[i]) }
	charCount, monCount := int32(in[4]), int32(in[5])
	if charCount > 3 || monCount > 3 {
		panic("spell stamina drain: more than three targets per list")
	}
	var chr [8][6]int64
	for k := range chr {
		copy(chr[k][:], in[20+6*k:26+6*k])
	}
	var mon [3][5]uint32 // hit points, maximum stamina, stamina, condition-17 duration, fatigue
	for j := range mon {
		for f := 0; f < 4; f++ {
			mon[j][f] = u(68 + 4*j + f)
		}
	}
	out := make([]int64, 36)
	rolls := 0
	roll := func() uint32 {
		var v uint32
		if real {
			v = uint32(int32(diceNode(in[:3], r)[0]))
		} else {
			if rolls >= 6 {
				panic("spell stamina drain: 0x517970 called past its six case returns")
			}
			v = u(12 + rolls)
		}
		rolls++
		if v != 9999 { // 0x54f937 / 0x54f9e2
			v += (v * u(3)) / 100
		}
		return v
	}
	var nFat, nCond int
	for i := 0; i < int(charCount); i++ {
		idx := in[6+i]
		if idx == -1 {
			panic("spell stamina drain: character index -1 is the srAssertFail path 0x54f8f2")
		}
		if idx < 0 || idx > 7 {
			panic("spell stamina drain: character index outside party slots 0..7")
		}
		c := &chr[idx]
		if int32(c[0]) == 0 { // 0x54f924
			continue
		}
		v := roll()
		if nFat >= 3 {
			panic("spell stamina drain: 0x52af50 called more than three times (fixture record cap)")
		}
		out[2+nFat], out[5+nFat] = idx, int64(v) // scale and target stay 0
		nFat++
		if real {
			res := characterFatigueApplyNode([]int64{idx, int64(v), 0, 0, c[0], c[1], c[2], 0, 0, 0, c[3], c[4], c[5], 0, 0, 0, 0, 0, 0, 0}, r)
			c[0], c[3], c[4] = res[0], res[1], res[2]
		}
	}
	for i := 0; i < int(monCount); i++ {
		id := in[9+i]
		j := int(id) - 100
		if j < 0 || j > 2 {
			panic("spell stamina drain: monster id outside the case records 100..102 (other ids hit the null-record assert 0x54f9b7)")
		}
		m := &mon[j]
		if m[2] == 0 { // 0x54f9cf
			continue
		}
		v := roll()
		if m[0] == 0 { // 0x52c07b
			continue
		}
		if v != 0 { // 0x52c08a
			if v >= m[2] {
				m[2] = 0
			} else {
				m[2] -= v
			}
		}
		if m[1] == 0 {
			panic("spell stamina drain: maximum stamina 0 divides by zero (0x52c0b8)")
		}
		m[4] = staminaFatigueLevel(m[2], m[1])
		if m[2] != 0 || m[3] >= 0x270f { // 0x52c0f6 / 0x52c102 unsigned
			continue
		}
		if nCond >= 3 {
			panic("spell stamina drain: 0x523c00 called more than three times (fixture record cap)")
		}
		// penalty 0; message = (third argument of 0x52c070 == 0) = 1
		out[21+nCond], out[24+nCond], out[27+nCond], out[33+nCond] = id, 0x11, 0x270f, 1
		nCond++
		if real {
			x := append([]int64(nil), sdrApplyParked...)
			set := func(name string, v int64) {
				for i, n := range mcaSpecApplyIn {
					if n == name {
						x[i] = v
						return
					}
				}
				panic("spell stamina drain: parked input " + name + " missing")
			}
			set("cond", 0x11)
			set("duration", 0x270f)
			set("penalty", 0)
			set("message", 1)
			set("M.HP", int64(m[0]))
			set("M.maxSP", int64(m[1]))
			set("M.SP", int64(m[2]))
			set("M.cond17", int64(m[3]))
			set("tpl.kind", in[80])
			set("a5.type", 0)
			set("a5.char", -1)
			set("a5.monster", -1)
			set("a5.f1b", 0)
			set("a5.f1c", 0)
			set("a5.f1e", 0)
			o := monsterConditionApplyNode(x, r)
			var sp, c17, fat, phased uint32
			for i, n := range mcaOutNames {
				switch n {
				case "M.SP":
					sp = uint32(o[i])
				case "M.cond17":
					c17 = uint32(o[i])
				case "M.fatigue":
					fat = uint32(o[i])
				case "s52c240_calls":
					phased = uint32(o[i])
				}
			}
			m[2], m[3] = sp, c17
			// +0x242 is not an input of the parked world; it is only rewritten when
			// 0x523c00 reaches 0x4eb7f0 (morale change), which calls 0x52c240 once.
			if phased != 0 {
				m[4] = fat
			}
		}
	}
	out[0], out[1], out[20] = int64(rolls), int64(nFat), int64(nCond)
	for j := range mon {
		out[14+2*j], out[15+2*j] = int64(mon[j][2]), int64(mon[j][4])
	}
	return out
}

// sdrApplyParked is the first boundary row of re/emu/cases/00523c00.json: F runs
// the real 0x523c00 on that world with this monster's stamina fields and the
// 0x52c070 call's arguments and cleared source record.
var sdrApplyParked = []int64{3, 2, 0, 1, 1, 1, 1, 50, 2, 20, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16, 0, 0, 0, 0, 0, 396, 30, 0, 0, 20, 19, 0, 1, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 7, 0, 10, 0, 536870912, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 50, 1, 2, 0, 0, 0, 0}

func init() {
	in := []string{"Z", "N", "S", "bonusPercent", "characterCount", "monsterCount",
		"characterTarget0", "characterTarget1", "characterTarget2", "monsterTarget0", "monsterTarget1", "monsterTarget2"}
	for i := 0; i < 6; i++ {
		in = append(in, "diceRoll"+string(rune('0'+i)))
	}
	in = append(in, "ret52af50", "ret523c00")
	for k := 0; k < 8; k++ {
		for _, n := range []string{"Stamina", "HitPoints", "MaximumStamina", "FatigueLevel", "FatigueFlag", "StateA45"} {
			in = append(in, "character"+n+string(rune('0'+k)))
		}
	}
	for j := 0; j < 3; j++ {
		for _, n := range []string{"HitPoints", "MaximumStamina", "CurrentStamina", "ExhaustionDuration"} {
			in = append(in, "monster"+n+string(rune('0'+j)))
		}
	}
	in = append(in, "templateKind")
	out := []string{"diceCount", "fatigueApplyCount"}
	for _, n := range []string{"Who", "Amount", "Scale", "Target"} {
		for i := 0; i < 3; i++ {
			out = append(out, "fatigueApply"+n+string(rune('0'+i)))
		}
	}
	for j := 0; j < 3; j++ {
		out = append(out, "monsterCurrentStamina"+string(rune('0'+j)), "monsterFatigueLevel"+string(rune('0'+j)))
	}
	out = append(out, "conditionApplyCount")
	for _, n := range []string{"Monster", "Condition", "Duration", "Penalty", "Message"} {
		for i := 0; i < 3; i++ {
			out = append(out, "conditionApply"+n+string(rune('0'+i)))
		}
	}
	Nodes["0054f8c0"] = Node{Addr: "0054f8c0", In: in, Out: out, F: spellStaminaDrainNode, Emu: spellStaminaDrainEmu}
}
