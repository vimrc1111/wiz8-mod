package calc

func init() {
	Nodes["005519c0"] = Node{
		Addr: "005519c0",
		In:   []string{"effectPower", "durationBonusPercent", "durationBase", "durationPerPower"},
		Out:  []string{"duration"},
		F:    effectDurationNode,
	}
}

// effectDurationNode consumes only the effect's four duration fields.
// All arithmetic wraps at 32 bits; duration comparisons are unsigned.
func effectDurationNode(in []int64, r Rand) []int64 {
	if len(in) != 4 || in[1] < -2147483648 || in[1] > 2147483647 {
		panic("effect duration: expected u32 power/base/per-power and i32 bonus percent")
	}
	for _, i := range []int{0, 2, 3} {
		if in[i] < 0 || in[i] > 4294967295 {
			panic("effect duration: expected u32 power/base/per-power")
		}
	}
	duration := uint32(in[2]) + uint32(in[0])*uint32(in[3])
	// Indefinite duration 9999 bypasses both RNG and bonus (0x5519cf;
	// notes/re/formulas-2.md:47), even when reached through wraparound.
	if duration == 9999 {
		return []int64{int64(duration)}
	}
	// Fixed +1 and R(4): 0x5519d6..0x5519dd. An extra +1 applies
	// to roll 0 above 1, or roll 1 at/below 2 (0x5519ed..0x5519fd).
	duration++
	roll := r.R(4)
	if (roll == 0 && duration > 1) || (roll == 1 && duration <= 2) {
		duration++
	}
	// 0x517f99..0x517faa: unsigned low-32-bit percentage numerator,
	// divisor 100. SpellDamage instead uses wide, signed floor arithmetic,
	// which differs for negative bonuses and overflowing products.
	product := duration * uint32(in[1])
	duration += product / 100
	return []int64{int64(duration)}
}
