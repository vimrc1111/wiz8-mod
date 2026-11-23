package calc

import "strconv"

func init() {
	// Eight party slots: the party traversal at exe 0x4ef420 uses 8 records.
	in := []string{"characterIndex"}
	for i := 0; i < 8; i++ {
		for _, field := range []string{"slotActive", "currentHP", "mainStatus", "totalXP", "nextLevelXP"} {
			in = append(in, field+strconv.Itoa(i))
		}
	}
	Nodes["004ef3c0"] = Node{Addr: "004ef3c0", In: in, Out: []string{"eligible"}, F: levelUpEligibilityNode}
}

// levelUpEligibilityNode takes a slot index and five fields for each party slot.
// The arena bases are wiring, not rule inputs; no executable data table is read.
func levelUpEligibilityNode(in []int64, r Rand) []int64 {
	if len(in) != 41 || in[0] < 0 || in[0] > 7 {
		panic("levelUpEligibility: expected index 0..7 and eight slot records")
	}
	for i := 1; i < len(in); i += 5 {
		if in[i] < 0 || in[i] > 255 || in[i+1] < -2147483648 || in[i+1] > 2147483647 {
			panic("levelUpEligibility: expected u8 slotActive and i32 currentHP")
		}
		for _, v := range in[i+2 : i+5] {
			if v < 0 || v > 4294967295 {
				panic("levelUpEligibility: expected u32 status and XP fields")
			}
		}
	}
	i := 1 + 5*int(in[0])
	// exe 0x4ef3d3: slot presence; 0x4ef3f0: current HP must be nonzero,
	// including negative i32 values. Record strides are 0x106 and 0x1862.
	// exe 0x4ef401: unsigned main status must be below the literal cutoff 18.
	// exe 0x4ef40d: unsigned cumulative XP must reach the next-level threshold.
	// exe 0x4ef41d: only the low byte is the eligibility return value.
	if in[i] != 0 && in[i+1] != 0 && in[i+2] < 18 && in[i+3] >= in[i+4] {
		return []int64{1}
	}
	return []int64{0}
}
