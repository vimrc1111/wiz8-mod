package calc

import "strconv"

// partyHpHealRollNode rolls independently for each occupied party slot, then
// composes the character healing node, including its injury event changes.
// Inputs: N/S/Z, two injury thresholds, three party event masks, then eight
// triples of occupied/currentHP/maximumHP. Outputs: eight HPs and three masks.
func partyHpHealRollNode(in []int64, r Rand) []int64 {
	// Eight slots: 0x52adb4..0x52adc1 traverses 0x830 bytes at stride 0x106.
	const slots = 8
	if len(in) != 8+3*slots {
		panic("party HP heal roll: expected 32 inputs")
	}
	limits := [][2]int64{{0, 255}, {0, 255}, {-32768, 32767},
		{0, 4294967295}, {0, 4294967295}, {0, 255}, {0, 255}, {0, 255}}
	for i := 0; i < slots; i++ {
		limits = append(limits, [2]int64{0, 255}, [2]int64{0, 4294967295}, [2]int64{0, 4294967295})
	}
	for i, bounds := range limits {
		if in[i] < bounds[0] || in[i] > bounds[1] {
			panic("party HP heal roll: input outside field width")
		}
	}
	out := make([]int64, slots+3)
	copy(out[slots:], in[5:8])
	for i := 0; i < slots; i++ {
		at := 8 + 3*i
		occupied, hp, maximum := in[at], in[at+1], in[at+2]
		out[i] = hp
		if occupied == 0 {
			continue
		}
		// 0x52ad81..0x52ad8a packs Z/N/S; 0x52ada2 rolls before the
		// child can skip dead or full HP. Notification is fixed to 1 (0x52ad9f).
		heal := diceNode([]int64{in[2], in[0], in[1]}, r)[0]
		result := characterHpHealNode([]int64{int64(i), int64(uint32(heal)), hp, maximum, 1,
			in[3], in[4], out[slots], out[slots+1], out[slots+2], occupied}, r)
		out[i] = result[0]
		copy(out[slots:], result[1:])
	}
	return out
}

func init() {
	in := []string{"N", "S", "Z", "minorInjuryClearPercent", "majorInjuryClearPercent",
		"minorInjuryCharacters", "majorInjuryCharacters", "injuryResponseCharacters"}
	out := make([]string, 0, 11)
	for i := 0; i < 8; i++ { // Party size established at 0x52adc1.
		suffix := strconv.Itoa(i)
		in = append(in, "occupied"+suffix, "currentHP"+suffix, "maximumHP"+suffix)
		out = append(out, "currentHP"+suffix)
	}
	out = append(out, "minorInjuryCharacters", "majorInjuryCharacters", "injuryResponseCharacters")
	Nodes["0052ad70"] = Node{Addr: "0052ad70", In: in, Out: out, F: partyHpHealRollNode}
}
