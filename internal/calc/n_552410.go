package calc

func init() {
	Nodes["00552410"] = Node{
		Addr: "00552410",
		In:   []string{"effectStrength", "monsterCategory", "primaryStatus", "monsterLevel", "mentalResistance", "mentalResistanceBonus"},
		Out:  []string{"resisted"},
		F:    monsterMentalResistRollNode,
	}
}

// The target record selects a monster by ID; the loaded template supplies its
// category, level and base mental resistance (0x552414..0x552477).
func monsterMentalResistRollNode(in []int64, r Rand) []int64 {
	if len(in) != 6 || in[0] < -2147483648 || in[0] > 2147483647 ||
		in[1] < 0 || in[1] > 255 || in[2] < 0 || in[2] > 4294967295 ||
		in[3] < 0 || in[3] > 255 || in[4] < 0 || in[4] > 255 ||
		in[5] < -128 || in[5] > 127 {
		panic("monster mental resist: expected i32 strength, u8 category, u32 status, u8 level/resistance, i8 bonus")
	}
	// Category 20 and unsigned status >=18 return true without RNG
	// (0x552441 and 0x55244a); the category's textual enum is unconfirmed.
	if in[1] == 20 || in[2] >= 18 {
		return []int64{1}
	}
	// Level difference has weight 3 (0x552469); arithmetic wraps to i32
	// before the signed clamp helper 0x5179d0 called at 0x552488.
	resistance := int32((in[3]-in[0])*3 + in[5] + in[4])
	// Bounds 5 and 95: 0x552463..0x552465. RNG bound 100: 0x55248d.
	threshold := min(int32(95), max(int32(5), resistance))
	if r.R(100) < int(threshold) {
		return []int64{1}
	}
	return []int64{0}
}
