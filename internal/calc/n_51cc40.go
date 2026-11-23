package calc

func init() {
	Nodes["0051cc40"] = Node{Addr: "0051cc40",
		In: []string{"firstItemIndex", "secondItemIndex", "firstType", "firstSubtype", "firstFlags", "firstRange",
			"secondType", "secondSubtype", "secondFlags", "secondRange"},
		Out: []string{"compatible"}, F: compatibleHandItemsNode}
}

// Inputs describe the selected definitions at global 0x68516c, stride 269
// (notes/re/loot.md:9). The second definition's writes win when IDs alias.
// Only AL is the boolean result, as consumed by the caller at 0x51d507.
func compatibleHandItemsNode(in []int64, r Rand) []int64 {
	if len(in) != 10 {
		panic("compatible hand items: expected ten inputs")
	}
	for i, v := range in {
		lo, hi := int64(0), int64(255) // u8 type, flags, range at +0x3e/+0x41/+0x47.
		if i < 2 {
			lo, hi = -1, 818 // Empty ID at 0x51cc44/0x51cc54; count 819 in loot.md:9.
		} else if i == 3 || i == 7 {
			hi = 65535 // u16 subtype at +0x3f, read by 0x51c8f0.
		}
		if v < lo || v > hi {
			panic("compatible hand items: invalid item index or field width")
		}
	}
	if in[0] == -1 || in[1] == -1 {
		return []int64{1}
	}
	first, second := in[2:6], in[6:10]
	if in[0] == in[1] {
		first = second
	}
	// Two-handed occupancy bit 4: 0x51cc6f and 0x51cc84.
	if first[2]&4 != 0 || second[2]&4 != 0 {
		return []int64{0}
	}
	// Ranged type 3 or ammunition type 4 chooses the partner direction;
	// first-direction precedence is from 0x51cc8d..0x51cca0.
	if first[0] == 3 || second[0] == 4 {
		return compatiblePartnerItemsNode([]int64{in[0], in[1], first[1], first[3], second[0], second[1], second[3]}, r)
	}
	if second[0] == 3 || first[0] == 4 {
		return compatiblePartnerItemsNode([]int64{in[1], in[0], second[1], second[3], first[0], first[1], first[3]}, r)
	}
	// Types above 3 bypass range matching; others need equal range bytes
	// (0x51cca2..0x51ccb1).
	if first[0] > 3 || second[0] > 3 || first[3] == second[3] {
		return []int64{1}
	}
	return []int64{0}
}
