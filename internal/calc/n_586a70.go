package calc

func init() {
	Nodes["00586a70"] = Node{
		Addr: "00586a70",
		In: []string{"characterIndex", "characterSlotActive", "currentHP", "mainStatus", "lockSpellKnown", "verbalCastingBlocked",
			"wizardryUsable", "divinityUsable", "alchemyUsable", "psionicUsable",
			"wizardryEffective", "divinityEffective", "alchemyEffective", "psionicEffective",
			"wizardryBase", "divinityBase", "alchemyBase", "psionicBase",
			"lockSpellRealm", "realm0Effective", "realm1Effective", "realm2Effective", "realm3Effective", "realm4Effective", "realm5Effective"},
		Out: []string{"lockSpellPower"}, F: lockSpellPowerNode,
	}
}

// The inputs are record values, not embedded spell-table data. The caller supplies
// spell 39's usability and realm fields from its own game state (0x65be1c).
func lockSpellPowerNode(in []int64, r Rand) []int64 {
	if len(in) != 25 {
		panic("lock spell power: expected 25 inputs")
	}
	for i, v := range in {
		lo, hi := int64(0), int64(4294967295)
		if i == 0 || (i >= 10 && i < 14) {
			lo, hi = -2147483648, 2147483647
		}
		if i == 1 || (i >= 6 && i < 10) {
			hi = 255
		}
		if v < lo || v > hi {
			panic("lock spell power: field exceeds storage width")
		}
	}
	if characterActionEligibilityNode(in[:4], r)[0] == 0 {
		return []int64{-1}
	}
	// 0x586a9f checks character +0xc69 for exactly 1, not merely nonzero.
	if in[4] != 1 {
		return []int64{-1}
	}
	// 0x586aad fixes spell=39, casting=1, training=0 and power=7.
	// Specialize 0x4ff7f0 to these arguments: training=0 bypasses risk,
	// class and progress tables. Schools 24..27 are visited in this order
	// (0x4ff8a6..0x4ff8ee); equal effective values retain the earlier school.
	const alchemy = 2 // skill 26 at 0x4ff83f, used by the verbal-block override.
	school, best := -1, int32(-1)
	if in[5] != 0 && in[8] != 0 {
		school = alchemy
	} else {
		for i := 0; i < 4; i++ {
			if in[6+i] != 0 && int32(in[10+i]) > best {
				school, best = i, int32(in[10+i])
			}
		}
	}
	if school < 0 {
		// 0x4ffb9a..0x4ffbc0 reports an assertion when no school is usable.
		panic("lock spell power: no usable school (original assertion)")
	}
	const realmCount = 6 // character realm records at +0x5cb, stride 0x26.
	if in[18] >= realmCount {
		panic("lock spell power: expected realm 0..5")
	}
	// 0x586acd..0x586aea weights the base school skill by 1 and effective
	// realm skill by 4, wraps the sum to uint32, then divides unsigned by 5.
	const realmWeight, divisor = uint32(4), uint32(5)
	weighted := uint32(in[14+school]) + realmWeight*uint32(in[19+in[18]])
	return []int64{int64(weighted / divisor)}
}
