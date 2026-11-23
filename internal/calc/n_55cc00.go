package calc

func init() {
	Nodes["0055cc00"] = Node{
		Addr: "0055cc00",
		In: []string{
			"rosterReverseDifficultyState0", "rosterReverseDifficultyState1",
			"rosterReverseDifficultyState2", "rosterReverseDifficultyState3",
			"rosterReverseDifficultyState4", "rosterReverseDifficultyState5",
			"rosterReverseDifficultyState6", "rosterReverseDifficultyState7",
			"value", "difficulty", "reverseDifficultyState", "characterIndex",
		},
		Out: []string{"value"},
		F:   difficultyNode,
	}
}

// difficultyNode adjusts the pointed-to rating; the return register is not a
// rule output. The case initializes eight distinct-from-selected background
// states, then writes reverseDifficultyState only at characterIndex (record
// stride 0x1862, notes/re/growth.md:67). Background states are fixture inputs;
// the rule reads only the selected character's state.
func difficultyNode(in []int64, r Rand) []int64 {
	if len(in) != 12 {
		panic("difficulty: expected eight roster states, value, difficulty, reverseDifficultyState, characterIndex")
	}
	for _, v := range in[:11] {
		if v < -2147483648 || v > 2147483647 {
			panic("difficulty: expected i32 roster states, value, difficulty and reverseDifficultyState")
		}
	}
	if in[11] < 0 || in[11] > 7 {
		panic("difficulty: expected characterIndex 0..7")
	}
	value, difficulty, reverseState := in[8], in[9], in[10]
	if difficulty != 0 && difficulty != 2 {
		return []int64{value}
	}
	// Difficulty 0/2 uses 140/60 percent, swapped for any nonzero C+0xa35.
	// Sources: notes/re/combat-2.md:34; original 0x55cc19..0x55cc90.
	multiplier := int64(140)
	if (difficulty == 2) != (reverseState != 0) {
		multiplier = 60
	}
	// The original forms a 32-bit product before signed division by 100
	// (0x55cc63..0x55cc78 and 0x55cc8d..0x55cca2). Preserve wraparound
	// even when a mathematically equivalent wide percentage would differ.
	return []int64{int64(int32(value*multiplier)) / 100}
}
