package calc

import "strconv"

func init() {
	// exe 0x4eee96: eight party slots, including the two companion slots.
	in := make([]string, 0, 24)
	for i := 0; i < 8; i++ {
		for _, field := range []string{"slotActive", "currentHP", "mainStatus"} {
			in = append(in, field+strconv.Itoa(i))
		}
	}
	Nodes["004eee80"] = Node{Addr: "004eee80", In: in, Out: []string{"xpEligibleCount"}, F: xpEligibleCountNode}
}

// xpEligibleCountNode takes presence, current HP and main status for each slot.
// Arena pointers only locate records; this node reads no executable data table.
func xpEligibleCountNode(in []int64, r Rand) []int64 {
	if len(in) != 24 {
		panic("xpEligibleCount: expected eight slot records")
	}
	var count int64
	for i := 0; i < len(in); i += 3 {
		if in[i] < 0 || in[i] > 255 || in[i+1] < -2147483648 || in[i+1] > 2147483647 || in[i+2] < 0 || in[i+2] > 4294967295 {
			panic("xpEligibleCount: expected u8 slotActive, i32 currentHP and u32 mainStatus")
		}
		// exe 0x4eee9b/0x4eeea0: Q+0 presence and C+0xb11 HP are nonzero.
		// exe 0x4eeea7: unsigned C+0xb01 status is below the literal cutoff 18.
		// HP is tested for zero, so negative i32 values also qualify.
		if in[i] != 0 && in[i+1] != 0 && in[i+2] < 18 {
			count++
		}
	}
	return []int64{count}
}
