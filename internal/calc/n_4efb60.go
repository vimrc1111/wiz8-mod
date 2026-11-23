package calc

import "math"

func init() {
	Nodes["004efb60"] = Node{
		Addr: "004efb60",
		In: []string{
			"slot2Active", "slot2TotalLevel", "slot3Active", "slot3TotalLevel",
			"slot4Active", "slot4TotalLevel", "slot5Active", "slot5TotalLevel",
			"slot6Active", "slot6TotalLevel", "slot7Active", "slot7TotalLevel",
		},
		Out: []string{"averageLevelFloat64Bits"},
		F:   partyAverageLevelFloatNode,
	}
}

// partyAverageLevelFloatNode averages the six party slots at indices 2..7.
// The return value encodes ST0 rounded to double, as required by emu's output.
func partyAverageLevelFloatNode(in []int64, _ Rand) []int64 {
	// Six slots: exe 0x4efb86; u32 total level: character +0x89,
	// exe 0x4efb80/0x4efb90. Slot flags are u8 at party-slot +0.
	const slots = 6
	const maxLevel = 4294967295
	if len(in) != slots*2 {
		panic("party average: expected six u8 active flags and u32 total levels")
	}
	var sum, count int64
	for slot := 0; slot < slots; slot++ {
		active, level := in[slot*2], in[slot*2+1]
		if active < 0 || active > 255 || level < 0 || level > maxLevel {
			panic("party average: expected u8 active flag and u32 total level")
		}
		if active != 0 {
			sum += level
			count++
		}
	}
	// exe 0x5ebb34 is the zero accumulator; 0x5ebb38 is the count
	// increment and empty-party default, both numerically one (0x4efbca).
	average := 1.0
	if count != 0 {
		// At most six u32 levels: the sum is exact in both x87 and double.
		// Divisors 1..6 cannot place this rational close enough to a double
		// midpoint for x87's intermediate rounding to change the final bits.
		average = float64(sum) / float64(count)
	}
	return []int64{int64(math.Float64bits(average))}
}
