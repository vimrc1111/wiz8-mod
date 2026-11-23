package calc

import "fmt"

func init() {
	in := []string{"characterIndex"}
	// Twelve equipment slots: 0x520ca2; item and instance flag:
	// 0x520ca7, 0x520cc8. Definition fields: 0x520cbe, 0x520ccf.
	for i := 0; i < 12; i++ {
		in = append(in, fmt.Sprintf("equipmentItem%d", i), fmt.Sprintf("curseRemoved%d", i))
	}
	for i := 0; i < 12; i++ {
		in = append(in, fmt.Sprintf("itemCursed%d", i), fmt.Sprintf("itemDifficulty%d", i))
	}
	Nodes["00520c70"] = Node{Addr: "00520c70", In: in,
		Out: []string{"removeCursePowerRequired"}, F: removeCursePowerRequiredNode}
}

// Inputs describe eight arena characters and twelve independent definitions;
// the item ID bounds are fixture capacity, not limits on the game's database.
// Globals 0x685174 and 0x68516c locate those records, with strides 0x1862
// and 0x10d respectively (0x520c75..0x520c94, 0x520cae..0x520cbb).
// Equipment starts at character +0xf5d with stride 12 (0x520c9c, 0x520cdb).
// Loaded item fields are inputs; this node reads no static exe data table.
func removeCursePowerRequiredNode(in []int64, r Rand) []int64 {
	if len(in) != 49 || in[0] < 0 || in[0] > 7 {
		panic("remove curse required: expected fixture character 0..7 and 48 equipment/definition fields")
	}
	for i := 0; i < 12; i++ {
		if in[1+2*i] < -1 || in[1+2*i] > 11 || in[2+2*i] < 0 || in[2+2*i] > 255 {
			panic("remove curse required: expected fixture item -1..11 and u8 removed flag")
		}
	}
	for _, v := range in[25:] {
		if v < 0 || v > 255 {
			panic("remove curse required: expected u8 curse flags and difficulties")
		}
	}
	required := int64(0)
	for i := 0; i < 12; i++ {
		item := in[1+2*i]
		// Empty sentinel -1 (0x520ca9), definition curse byte +0x8c
		// (0x520cbe), and instance removed byte +0xa (0x520cc8).
		if item == -1 || in[25+2*item] == 0 || in[2+2*i] != 0 {
			continue
		}
		// Unsigned difficulty +0x85; equal/lower values leave the maximum
		// unchanged (0x520ccf..0x520cd9).
		required = max(required, in[26+2*item])
	}
	// Divisor 3 and upward rounding: 0x520ce5..0x520d09.
	return []int64{(required + 2) / 3}
}
