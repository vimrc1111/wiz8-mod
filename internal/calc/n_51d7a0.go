package calc

import "strconv"

func init() {
	// Shared loaded item/spell fields, followed by eight independent characters.
	// The child layout and semantic names are defined in n_51d610.go.
	child := Nodes["0051d610"].In
	in := []string{child[0]}
	in = append(in, child[4:17]...)
	in = append(in, child[20])
	// Eight slots, including companions: exe 0x51d7e0..0x51d7f2.
	for slot := 0; slot < 8; slot++ {
		suffix := strconv.Itoa(slot)
		for _, field := range []string{"slotActive", "currentHP", "mainStatus"} {
			in = append(in, field+suffix)
		}
		for _, fields := range [][]string{child[1:4], child[17:20], child[21:69]} {
			for _, field := range fields {
				in = append(in, field+suffix)
			}
		}
	}
	Nodes["0051d7a0"] = Node{Addr: "0051d7a0", In: in,
		Out: []string{"eligible"}, F: partyItemUseEligibleNode}
}

// Input layout: 15 shared item/spell values and 57 values per party slot.
// Each slot supplies presence, HP, status, identity, class levels, selected
// spell's learned state, seven effective attributes and 41 effective skills.
// The child reads its immutable dispatch from the user's exe at initialization;
// this composition is pure and neither reads files nor changes input records.
func partyItemUseEligibleNode(in []int64, r Rand) []int64 {
	if len(in) != 471 {
		panic("party item use: expected shared requirements and eight character records")
	}
	// Storage widths follow 0x51d610..0x51d6e2 and 0x51d7b0..0x51d7c9.
	for i, v := range in {
		low, high := int64(0), int64(4294967295)
		if i < 15 {
			switch i {
			case 0:
				high = 3 // Four indexed loaded item records in the arena fixture.
			case 1:
				high = 65535 // Profession mask is u16 at item +0x76.
			case 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13:
				high = 255
			}
		} else {
			switch (i - 15) % 57 {
			case 0:
				high = 255
			case 1:
				low, high = -2147483648, 2147483647
			}
		}
		if v < low || v > high {
			panic("party item use: storage width exceeded")
		}
	}
	// Legal requirement IDs and absent sentinel: child 0x51d685..0x51d6d1.
	for _, i := range []int{4, 6, 8, 10} {
		limit := int64(6)
		if i >= 8 {
			limit = 40
		}
		if in[i] != 255 && in[i] > limit {
			panic("party item use: invalid requirement ID")
		}
	}
	child := make([]int64, 69)
	child[0] = in[0]
	copy(child[4:17], in[1:14])
	child[20] = in[14]
	for slot := 0; slot < 8; slot++ {
		start := 15 + slot*57
		// 0x51d7b0..0x51d7d0: presence and HP nonzero, unsigned status <18.
		if in[start] == 0 || in[start+1] == 0 || in[start+2] >= 18 {
			continue
		}
		copy(child[1:4], in[start+3:start+6])
		copy(child[17:20], in[start+6:start+9])
		copy(child[21:69], in[start+9:start+57])
		// 0x51d7d4..0x51d7de: first successful child's AL ends the search.
		if itemUseRequirementsNode(child, r)[0] != 0 {
			return []int64{1}
		}
	}
	return []int64{0} // 0x51d7f4: all eight slots exhausted.
}
