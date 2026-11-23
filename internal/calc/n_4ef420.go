package calc

import "strconv"

func init() {
	// exe 0x4ef438: eight party slots, including the two recruit slots.
	in := make([]string, 0, 16)
	for i := 0; i < 8; i++ {
		in = append(in, "slotActive"+strconv.Itoa(i), "totalLevel"+strconv.Itoa(i))
	}
	Nodes["004ef420"] = Node{Addr: "004ef420", In: in, Out: []string{"partyAverageLevel"}, F: partyAverageLevelNode}
}

// partyAverageLevelNode takes eight (u8 slotActive, u32 totalLevel) pairs.
// exe 0x4ef420/0x4ef426 supplies the character and slot array bases;
// level is at character +0x89, with strides 0x1862 and 0x106 respectively.
// These are mutable party records, not executable data tables.
func partyAverageLevelNode(in []int64, r Rand) []int64 {
	if len(in) != 16 {
		panic("partyAverageLevel: expected eight slotActive/totalLevel pairs")
	}
	var total, count uint32
	for i := 0; i < len(in); i += 2 {
		if in[i] < 0 || in[i] > 255 || in[i+1] < 0 || in[i+1] > 4294967295 {
			panic("partyAverageLevel: expected u8 slotActive and u32 totalLevel")
		}
		// exe 0x4ef43d/0x4ef442: every nonzero flag counts once;
		// cumulative levels wrap at 32 bits before division at 0x4ef456.
		if in[i] != 0 {
			total += uint32(in[i+1])
			count++
		}
	}
	// Empty parties fault in the original at exe 0x4ef456. Preserve failure
	// instead of inventing an average; the emulator's rows require a return.
	if count == 0 {
		panic("partyAverageLevel: division by zero for empty party")
	}
	return []int64{int64(total / count)}
}
