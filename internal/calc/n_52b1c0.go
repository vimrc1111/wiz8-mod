package calc

// characterStaminaHealNode 0x52b1c0: add `heal` to party member `who`'s
// stamina (character +0xb1d, maximum +0xb19), clamp, show the notice, refresh
// the fatigue level (+0x169e) and clear state 0x11 when the gate field allows.
//
// Callees: 0x4ee9d0 (armor classes) and 0x523330 (condition remove) are D4
// formula nodes and are address-stubbed in the emulator case: their calls are
// outputs (count and arguments). 0x590950 (notice) is not a formula; the case
// observes it through its single wcscpy call, whose source pointer depends on
// the first character of the formatted notice (build/wiz8.asm:541091..).
// Maximum stamina 0 with nonzero current stamina divides by zero in the
// original (fault at 0x52b273), so calc panics there.
//
// Inputs: charBase, who, healAmount(i32), notifyFlag(u32, only the low byte
// is tested), state +0xb01, hitPoints +0xb11, maximumStamina, currentStamina,
// fatigueLevel, gate +0xa45, fullNoticeLead, healNoticeLead (first UTF-16
// unit of the two notice formats, 0x68c09c+0x970 / +0x978).
func characterStaminaHealNode(in []int64, r Rand) []int64 {
	if len(in) != 12 {
		panic("stamina heal: expected twelve inputs")
	}
	if in[1] < 0 || in[1] > 7 {
		panic("stamina heal: party index outside 0..7")
	}
	u := func(i int) uint32 { return uint32(in[i]) }
	charBase, who, heal, notify := u(0), u(1), u(2), u(3)
	state, hp, max, cur, fatigue, gate := u(4), u(5), u(6), u(7), u(8), u(9)
	out := make([]int64, 10)
	out[0], out[1] = int64(cur), int64(fatigue)
	// 0x52b1de..0x52b207: unsigned state test, nonzero HP, not already full.
	if state >= 0x12 || hp == 0 || cur == max {
		return out
	}
	cur += heal
	if int32(cur) > int32(max) { // 0x52b213 signed clamp
		cur = max
	}
	if uint8(notify) != 0 { // 0x52b223 low byte only
		lead := in[11]
		if cur == max {
			lead = in[10]
		}
		out[8] = 1
		out[9] = 0x60cc74
		if lead == 0x27 || lead == 0x3a { // 0x590950: ' or : selects 0x689b34
			out[9] = 0x689b34
		}
	}
	if max == 0 {
		panic("stamina heal: division by zero maximum stamina")
	}
	q := uint32(cur*100) / max // 0x52b262..0x52b273 (unsigned 32-bit)
	pct := int32(100 - q)
	var level uint32
	switch {
	case pct < 0x32:
		level = 0
	case pct < 0x46:
		level = 1
	case pct < 0x55:
		level = 2
	case pct >= 0x5f:
		level = 4
	default:
		level = 3
	}
	if level != fatigue {
		out[2], out[3] = 1, int64(charBase+who*0x1862)
	}
	if gate == 0x270f && int32(cur) >= 10 {
		out[4], out[5], out[6], out[7] = 1, int64(who), 0x11, 1
	}
	out[0], out[1] = int64(cur), int64(level)
	return out
}

func init() {
	Nodes["0052b1c0"] = Node{
		Addr: "0052b1c0",
		In: []string{"charBase", "who", "healAmount", "notifyFlag", "state", "hitPoints",
			"maximumStamina", "currentStamina", "fatigueLevel", "gate", "fullNoticeLead", "healNoticeLead"},
		Out: []string{"currentStamina", "fatigueLevel", "armorRecalcCount", "armorRecalcCharacter",
			"conditionRemoveCount", "conditionRemoveWho", "conditionRemoveCondition", "conditionRemoveFlag",
			"noticeCount", "noticeTextSource"},
		F: characterStaminaHealNode,
	}
}
