package calc

func init() {
	Nodes["00520b40"] = Node{
		Addr: "00520b40",
		In: []string{"identifyPower", "identifyBonusPercent", "itemIndex",
			"itemDifficulty0", "itemDifficulty1", "itemDifficulty2",
			"itemIdentification6", "itemIdentification7", "itemIdentification8", "itemIdentification11"},
		Out: []string{"itemIdentification6", "itemIdentification7", "itemIdentification8", "itemIdentification11"},
		F:   identifyPowerApplyNode,
	}
}

// identifyPowerApplyNode takes a valid item instance and three synthetic item
// definitions (269-byte stride at 0x520b61..0x520b75). Only their difficulty
// bytes (+133) affect the rule; these are inputs, not an embedded exe table.
// The index range describes this arena fixture, not a game database limit.
func identifyPowerApplyNode(in []int64, r Rand) []int64 {
	if len(in) != 10 || in[0] < 0 || in[0] > 4294967295 ||
		in[1] < -2147483648 || in[1] > 2147483647 || in[2] < 0 || in[2] > 2 {
		panic("identify: expected u32 power, i32 bonus percent and fixture item index 0..2")
	}
	for _, v := range in[3:] {
		if v < 0 || v > 255 {
			panic("identify: expected u8 item difficulties and identification flags")
		}
	}
	// Multiplier 3: 0x520b4e. Division by 100: 0x517f9e..0x517fa5.
	// Both products and the final sum wrap at 32 bits. The percentage
	// numerator is unsigned, including negative bonuses; SpellDamage's
	// signed, wide arithmetic does not reproduce this helper's behavior.
	power := uint32(in[0]) * 3
	product := power * uint32(in[1])
	power += product / 100
	out := append([]int64(nil), in[6:]...)
	// Unsigned inclusive threshold: 0x520b83..0x520b85. Successful writes
	// use 1 at instance offsets 6, 7, 8, 11 (0x520ba4..0x520baf).
	if power >= uint32(in[3+in[2]]) {
		for i := range out {
			out[i] = 1
		}
	}
	return out
}
