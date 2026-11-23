package calc

import "math"

func init() {
	Nodes["0048bc30"] = Node{
		Addr: "0048bc30",
		In: []string{
			"monsterLevel", "Z", "N", "S", "difficulty",
			"slot2Active", "slot2TotalLevel", "slot3Active", "slot3TotalLevel",
			"slot4Active", "slot4TotalLevel", "slot5Active", "slot5TotalLevel",
			"slot6Active", "slot6TotalLevel", "slot7Active", "slot7TotalLevel",
		},
		Out: []string{"groupSize"},
		F:   encounterGroupSizeNode,
	}
}

// encounterGroupSizeNode: group-size dice at monster template +0xc1 (Z i16,
// N u8, S u8), level at +0x251, difficulty global 0x6850d5. Inputs 5..16 are
// the party-average inputs of partyAverageLevelFloatNode (0x4efb60).
// With S=0 and N>0 the dice minimum exceeds the maximum, so for ratio >= 1.2
// and difficulty not 0 or 2 the original reroll loop never ends; so does this.
func encounterGroupSizeNode(in []int64, r Rand) []int64 {
	if len(in) != 17 || in[0] < 0 || in[0] > 255 || in[4] < -1<<31 || in[4] > 1<<31-1 {
		panic("encounter group size: expected u8 level, dice, i32 difficulty, party slots")
	}
	diff := in[4]
	z, n, s := in[1], in[2], in[3]
	avg := math.Float64frombits(uint64(partyAverageLevelFloatNode(in[5:], r)[0]))
	// Level and ratio are stored as float32 (exe 0x48bc48, 0x48bc70); the
	// ratio is level / party average (exe 0x48bc51). Average 0 gives Inf/NaN.
	ratio := float64(float32(float64(in[0]) / avg))
	// Midpoint of the dice minimum Z+N and maximum Z+N*S, halved (0.5 at exe
	// 0x5ebc7c) and stored as float32 (exe 0x48bc7a..0x48bc8e).
	mid := float64(float32(float64(z+n*s+z+n) * 0.5))
	roll := diceNode(in[1:4], r)[0]
	// x87 compares: "below" and "below or equal" tests also hold for NaN.
	below := func(a, b float64) bool { return !(a >= b) }
	belowEq := func(a, b float64) bool { return !(a > b) }
	// Ratio thresholds are float32 constants 1.2 (exe 0x5ec394), 0.8 (0x5ebb30).
	if !below(ratio, float64(float32(1.2))) {
		switch diff {
		case 0:
			return []int64{z + n} // exe 0x48bcba: dice minimum
		case 2:
			return []int64{roll}
		}
		for !belowEq(float64(roll), mid) { // reroll above the midpoint, exe 0x48bce9
			roll = diceNode(in[1:4], r)[0]
		}
		return []int64{roll}
	}
	if !belowEq(ratio, float64(float32(0.8))) {
		return []int64{roll}
	}
	switch diff {
	case 2:
		return []int64{z + n*s} // exe 0x48bd2b: dice maximum
	case 0:
		return []int64{roll}
	}
	for below(float64(roll), mid) { // reroll below the midpoint, exe 0x48bd59
		roll = diceNode(in[1:4], r)[0]
	}
	return []int64{roll}
}
