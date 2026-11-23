package calc

// spellHpHealNode composes dice, percentage bonus and HP restoration.
// Field layout and unsigned comparisons: 0x54f190..0x54f505. Message
// formatting uses synthetic empty names and an absent renderer in the replay;
// message flags and injury masks remain observable outputs.
// The case supplies eight character records (party size at 0x52ddbb), two
// monster records with synthetic IDs 100/101, and two target slots per list.
// Its wcslen import returns zero: presentation length is unused with the
// absent renderer (0x59060c..0x590614). It does not model displayed text.
func spellHpHealNode(in []int64, r Rand) []int64 {
	if len(in) != 39 {
		panic("spell HP heal: expected 39 inputs")
	}
	hp := make([]uint32, 10)
	maximum := make([]uint32, 10)
	for i := range hp {
		maximum[i] = uint32(in[11+2*i])
		hp[i] = uint32(in[12+2*i])
	}
	minor, major, response := in[33], in[34], in[35]
	full := int64(1)
	messages := 0
	// Dice's 9999 sentinel bypasses the bonus at 0x54f249 / 0x54f35f.
	heal := func() uint32 {
		v := uint32(diceNode(in[:3], r)[0])
		if v != 9999 {
			// 0x517f90 truncates the product to u32 before unsigned division
			// by 100; addition also wraps. This is not signed percentage math.
			v += (v * uint32(in[3])) / 100
		}
		return v
	}
	for group := 0; group < 2; group++ {
		for j := 0; j < int(in[5+group]); j++ {
			idx := int(in[7+j])
			if group == 1 {
				// IDs 100/101 are fixture keys in cases/0054f190.json,
				// not constants from an executable table.
				idx = int(in[9+j]) - 100 + 8
			}
			gap := maximum[idx] - hp[idx]
			if gap == 0 {
				continue
			}
			amount := min(heal(), gap)
			if group == 0 {
				result := characterHpHealNode([]int64{int64(idx), int64(amount), int64(hp[idx]), int64(maximum[idx]), in[4], in[31], in[32], minor, major, response, 1}, r)
				hp[idx] = uint32(result[0])
				minor, major, response = result[1], result[2], result[3]
			} else if hp[idx] != 0 && hp[idx] != maximum[idx] && amount > 0 {
				// 0x52bfd0 leaves dead monsters unchanged and caps unsigned HP.
				hp[idx] = min(hp[idx]+amount, maximum[idx])
			}
			if hp[idx] < maximum[idx] {
				full = 0
			}
			if in[4] == 0 {
				messages++
			}
		}
	}
	pending, flag, resultFlag := in[36], in[37], in[38]
	if in[4] == 0 && messages > 0 {
		pending, flag, resultFlag = 1, 1, 1
	}
	out := []int64{full}
	for _, v := range hp {
		out = append(out, int64(v))
	}
	return append(out, minor, major, response, pending, flag, resultFlag)
}

func init() {
	Nodes["0054f190"] = Node{Addr: "0054f190",
		In:  []string{"Z", "N", "S", "bonusPercent", "notify", "characterCount", "monsterCount", "characterTarget0", "characterTarget1", "monsterTarget0", "monsterTarget1", "characterMaximumHP0", "characterCurrentHP0", "characterMaximumHP1", "characterCurrentHP1", "characterMaximumHP2", "characterCurrentHP2", "characterMaximumHP3", "characterCurrentHP3", "characterMaximumHP4", "characterCurrentHP4", "characterMaximumHP5", "characterCurrentHP5", "characterMaximumHP6", "characterCurrentHP6", "characterMaximumHP7", "characterCurrentHP7", "monsterMaximumHP0", "monsterCurrentHP0", "monsterMaximumHP1", "monsterCurrentHP1", "minorInjuryClearPercent", "majorInjuryClearPercent", "minorInjuryCharacters", "majorInjuryCharacters", "injuryResponseCharacters", "messagePending", "effectMessageFlag", "effectResultFlag"},
		Out: []string{"allFull", "characterCurrentHP0", "characterCurrentHP1", "characterCurrentHP2", "characterCurrentHP3", "characterCurrentHP4", "characterCurrentHP5", "characterCurrentHP6", "characterCurrentHP7", "monsterCurrentHP0", "monsterCurrentHP1", "minorInjuryCharacters", "majorInjuryCharacters", "injuryResponseCharacters", "messagePending", "effectMessageFlag", "effectResultFlag"}, F: spellHpHealNode}
}
