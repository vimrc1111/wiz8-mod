package calc

// characterHpHealNode combines HP healing with the injury event queries and
// removals (0x52add0, 0x52dd90, 0x52dd20). Thresholds at 0x5ed904 and
// 0x5ed900 are mutable scalar globals supplied as inputs, not an embedded table.
// Event inputs are complete party bitmasks, including the other characters.
// Notification calls 0x590950 run with an absent renderer in the replay fixture;
// their name, format and color fields do not affect HP or injury event state.
func characterHpHealNode(in []int64, r Rand) []int64 {
	if len(in) != 11 {
		panic("character HP heal: expected 11 inputs")
	}
	// The party has eight bit positions (0x52ddbb); character records and
	// occupied slots have strides 0x1862 and 0x106 (0x52add6..0x52adff).
	limits := [][2]int64{{0, 7}, {0, 4294967295}, {0, 4294967295}, {0, 4294967295}, {0, 255}, {0, 4294967295}, {0, 4294967295}, {0, 255}, {0, 255}, {0, 255}, {1, 255}}
	for i, bounds := range limits {
		if in[i] < bounds[0] || in[i] > bounds[1] {
			panic("character HP heal: input outside field width or unoccupied slot")
		}
	}
	hp, maximum := uint32(in[2]), uint32(in[3])
	minor, major, response := uint8(in[7]), uint8(in[8]), uint8(in[9])
	if hp != 0 && hp != maximum {
		if maximum == 0 {
			panic("character HP heal: live character needs nonzero maximum HP")
		}
		// Both addition and comparison are unsigned 32-bit (0x52ae3f..0x52ae4b).
		hp += uint32(in[1])
		if hp > maximum {
			hp = maximum
		}
		// Percentage factor 100 is formed at 0x52ae9c..0x52aea2.
		// Preserve overflow before the unsigned division at 0x52aea7.
		percent := hp * 100 / maximum
		bit := uint8(1 << uint(in[0]))
		if percent >= uint32(in[5]) {
			minor &^= bit
			if percent >= uint32(in[6]) {
				major &^= bit
				response &^= bit
			}
		}
	}
	return []int64{int64(hp), int64(minor), int64(major), int64(response)}
}

func init() {
	Nodes["0052add0"] = Node{
		Addr: "0052add0",
		In: []string{"characterIndex", "healAmount", "currentHP", "maximumHP", "notify",
			"minorInjuryClearPercent", "majorInjuryClearPercent", "minorInjuryCharacters",
			"majorInjuryCharacters", "injuryResponseCharacters", "occupied"},
		Out: []string{"currentHP", "minorInjuryCharacters", "majorInjuryCharacters", "injuryResponseCharacters"},
		F:   characterHpHealNode,
	}
}
