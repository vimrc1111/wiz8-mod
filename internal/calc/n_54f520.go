package calc

// spellStaminaRestoreNode 0x54f520 composes the effect dice, the percentage
// bonus and stamina restoration for the effect's character and monster target
// lists (effect +0xf4/+0xfc and +0xe4/+0xec).
//
// Character targets go through characterStaminaHealNode (0x52b1c0, stamina
// +0xb1d/+0xb19). Monster targets go through the in-file monsterStamina
// restore (0x52c140: stamina +0x33/+0x2f, hit points +0x2b, primary condition
// +0x107, fatigue +0x242; at full-exhaustion duration 9999 and stamina >= 10
// it clears condition 0x11 through 0x523f40 and re-picks the primary
// condition). Message text is not modelled; the message flags are outputs.
//
// Input layout: 0..4 Z, N, S, bonusPercent, individualNotice (byte at
// 0x68510c); 5..6 character and monster target counts (i32); 7..8 character
// targets; 9..10 monster ids; 11 character array base (0x685174); 12 + 6k
// per character k: state, hit points, maximum stamina, current stamina,
// fatigue, gate; 60 + 10j per monster j: hit points, maximum stamina,
// current stamina, primary condition, condition-17 duration, durations of
// conditions 1, 5, 8, 18, fatigue; 80..84 fullNoticeLead, healNoticeLead,
// messagePending, effectMessageFlag, effectResultFlag.
//
// Fixture caps (panic past them): at most two targets per list, character
// indices 0..7, monster ids 100/101 (the case's synthetic records).
// Not modelled: condition durations other than 1, 5, 8, 17, 18 (all zero in
// the case; condition 19 makes 0x50eac0 dereference a null character for a
// monster, which faults in the original), the derived-stat cache that 0x523f40
// rebuilds through 0x50e8c0, and the monster-flag path (0x4e60b0) behind
// monster +0x14, which the case holds at zero.
func spellStaminaRestoreNode(in []int64, r Rand) []int64 {
	if len(in) != 85 {
		panic("spell stamina restore: expected 85 inputs")
	}
	u := func(i int) uint32 { return uint32(in[i]) }
	individual := uint8(in[4]) != 0 // 0x54f63a / 0x54f745 / 0x54f786 test the low byte
	charCount, monCount := int32(in[5]), int32(in[6])
	if charCount > 2 || monCount > 2 {
		panic("spell stamina restore: more than two targets per list")
	}
	base := u(11)
	var chr [8][6]uint32
	for k := range chr {
		for f := range chr[k] {
			chr[k][f] = u(12 + 6*k + f)
		}
	}
	var mon [2][10]uint32
	for j := range mon {
		for f := range mon[j] {
			mon[j][f] = u(60 + 10*j + f)
		}
	}
	out := make([]int64, 41)
	var armor, cond, notice int
	allRestored := int64(1)
	count := 0
	// 0x517970 plus 0x517f90: the 9999 sentinel skips the bonus; the product
	// truncates to u32 before the unsigned division by 100.
	heal := func() uint32 {
		v := uint32(diceNode(in[:3], r)[0])
		if v != 9999 {
			v += (v * u(3)) / 100
		}
		return v
	}
	for i := 0; i < int(charCount); i++ {
		idx := in[7+i]
		if idx < 0 || idx > 7 {
			panic("spell stamina restore: character index outside party slots 0..7")
		}
		c := &chr[idx]
		if c[3] == c[2] { // 0x54f5af: equal stamina skips the target
			continue
		}
		v := heal()
		if v == 0 {
			continue
		}
		if gap := c[2] - c[3]; v >= gap {
			v = gap
		}
		res := characterStaminaHealNode([]int64{int64(base), idx, int64(v), in[4], int64(c[0]), int64(c[1]), int64(c[2]), int64(c[3]), int64(c[4]), int64(c[5]), in[80], in[81]}, r)
		c[3], c[4] = uint32(res[0]), uint32(res[1])
		if res[2] != 0 {
			out[18+armor] = res[3]
			armor++
		}
		if res[4] != 0 {
			out[21+cond], out[23+cond], out[25+cond] = res[5], res[6], res[7]
			cond++
		}
		if res[8] != 0 {
			out[28+notice] = res[9]
			notice++
		}
		if int32(c[3]) < int32(c[2]) { // 0x54f631 signed
			allRestored = 0
		}
		count++
	}
	for i := 0; i < int(monCount); i++ {
		j := int(in[9+i]) - 100
		if j < 0 || j > 1 {
			panic("spell stamina restore: monster id outside the case records 100/101")
		}
		m := &mon[j]
		if m[2] == m[1] { // 0x54f6dd
			continue
		}
		v := heal()
		if v == 0 {
			continue
		}
		if gap := m[1] - m[2]; v >= gap {
			v = gap
		}
		// 0x52c140: only living monsters in a primary condition below 0x12.
		if m[3] < 0x12 && m[0] != 0 {
			m[2] += v
			if m[2] > m[1] {
				m[2] = m[1]
			}
			if m[1] == 0 {
				panic("spell stamina restore: division by zero maximum stamina")
			}
			m[9] = staminaFatigueLevel(m[2], m[1])
			if m[4] == 9999 && m[2] >= 10 {
				// 0x523f40 with condition 0x11: clear it, then take the
				// highest condition that still has a duration.
				m[4] = 0
				dur := map[int]uint32{1: m[5], 5: m[6], 8: m[7], 17: m[4], 18: m[8]}
				// Slot 19 is omitted: 0x50eac0 passes a null character to
				// 0x517ec0 for it, so a monster never holds it.
				p := 18
				for p > 0 && dur[p] == 0 {
					p--
				}
				m[3] = uint32(p)
			}
		}
		if m[2] < m[1] { // 0x54f73c unsigned
			allRestored = 0
		}
		count++
	}
	for k := range chr {
		out[1+k], out[9+k] = int64(chr[k][3]), int64(chr[k][4])
	}
	out[0] = allRestored
	out[17], out[20], out[27] = int64(armor), int64(cond), int64(notice)
	for j := range mon {
		out[30+4*j], out[31+4*j], out[32+4*j], out[33+4*j] = int64(mon[j][2]), int64(mon[j][9]), int64(mon[j][3]), int64(mon[j][4])
	}
	out[38], out[39], out[40] = in[82], in[83], in[84]
	if !individual && count > 0 { // 0x54f786 / 0x54f78e
		out[38], out[39], out[40] = 1, 1, 1
	}
	return out
}

// staminaFatigueLevel is the 0..4 step computed from stamina percent at
// 0x52c1d2..0x52c216 (unsigned 32-bit product, signed threshold compares).
func staminaFatigueLevel(cur, max uint32) uint32 {
	pct := int32(100 - uint32(cur*100)/max)
	switch {
	case pct < 0x32:
		return 0
	case pct < 0x46:
		return 1
	case pct < 0x55:
		return 2
	case pct >= 0x5f:
		return 4
	}
	return 3
}

func init() {
	in := []string{"Z", "N", "S", "bonusPercent", "individualNotice", "characterCount", "monsterCount", "characterTarget0", "characterTarget1", "monsterTarget0", "monsterTarget1", "characterBase"}
	for k := 0; k < 8; k++ {
		for _, n := range []string{"State", "HitPoints", "MaximumStamina", "CurrentStamina", "FatigueLevel", "Gate"} {
			in = append(in, "character"+n+string(rune('0'+k)))
		}
	}
	for j := 0; j < 2; j++ {
		for _, n := range []string{"HitPoints", "MaximumStamina", "CurrentStamina", "PrimaryCondition", "ExhaustionDuration", "Condition1Duration", "Condition5Duration", "Condition8Duration", "Condition18Duration", "FatigueLevel"} {
			in = append(in, "monster"+n+string(rune('0'+j)))
		}
	}
	in = append(in, "fullNoticeLead", "healNoticeLead", "messagePending", "effectMessageFlag", "effectResultFlag")
	out := []string{"allRestored"}
	for k := 0; k < 8; k++ {
		out = append(out, "characterCurrentStamina"+string(rune('0'+k)))
	}
	for k := 0; k < 8; k++ {
		out = append(out, "characterFatigueLevel"+string(rune('0'+k)))
	}
	out = append(out, "armorRecalcCount", "armorRecalcCharacter0", "armorRecalcCharacter1",
		"conditionRemoveCount", "conditionRemoveWho0", "conditionRemoveWho1", "conditionRemoveCondition0", "conditionRemoveCondition1", "conditionRemoveFlag0", "conditionRemoveFlag1",
		"noticeCount", "noticeTextSource0", "noticeTextSource1")
	for j := 0; j < 2; j++ {
		for _, n := range []string{"CurrentStamina", "FatigueLevel", "PrimaryCondition", "ExhaustionDuration"} {
			out = append(out, "monster"+n+string(rune('0'+j)))
		}
	}
	out = append(out, "messagePending", "effectMessageFlag", "effectResultFlag")
	Nodes["0054f520"] = Node{Addr: "0054f520", In: in, Out: out, F: spellStaminaRestoreNode}
}
