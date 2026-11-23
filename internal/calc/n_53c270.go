package calc

import "strconv"

func init() {
	// Eight party slots, as traversed at exe 0x4ef420.
	in := []string{"characterIndex"}
	for i := 0; i < 8; i++ {
		for _, field := range []string{"slotActive", "currentHP", "mainStatus"} {
			in = append(in, field+strconv.Itoa(i))
		}
	}
	Nodes["0053c270"] = Node{Addr: "0053c270", In: in, Out: []string{"targetable"}, F: characterTargetableNode}
}

// Inputs contain the index and three fields for each of the eight party slots.
// Runtime array pointers are fixture wiring; this rule reads no exe data table.
func characterTargetableNode(in []int64, r Rand) []int64 {
	if len(in) != 25 || in[0] < 0 || in[0] > 7 {
		panic("characterTargetable: expected index 0..7 and eight slot records")
	}
	for i := 1; i < len(in); i += 3 {
		// exe 0x53c283 reads u8 presence; 0x53c29d reads i32 HP;
		// 0x53c2ab compares u32 status. Bounds retain their entire widths.
		if in[i] < 0 || in[i] > 255 || in[i+1] < -2147483648 || in[i+1] > 2147483647 || in[i+2] < 0 || in[i+2] > 4294967295 {
			panic("characterTargetable: expected u8 slotActive, i32 currentHP and u32 mainStatus")
		}
	}
	i := 1 + 3*int(in[0])
	// exe 0x53c283/0x53c2a4 require nonzero presence and HP (negative HP
	// passes); 0x53c2ab uses the literal status cutoff 0x12 (18).
	// Slot and character strides: 0x106 and 0x1862 (notes/re/formulas.md:9).
	// exe 0x53c2b4/0x53c2b7 return the boolean in the low byte only.
	if in[i] != 0 && in[i+1] != 0 && in[i+2] < 18 {
		return []int64{1}
	}
	return []int64{0}
}
