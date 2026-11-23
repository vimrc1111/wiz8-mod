package calc

import "fmt"

func init() {
	in := []string{"characterIndex", "removeCursePower", "bonusPercent"}
	out := []string{"result"}
	// Twelve equipment instances: 0x520c06; instance layout: 0x520c0b,
	// 0x520c3f, 0x520c46. Template fields: 0x520c26 and 0x520c32.
	for i := 0; i < 12; i++ {
		in = append(in, fmt.Sprintf("equipmentItem%d", i), fmt.Sprintf("curseRemoved%d", i))
		out = append(out, fmt.Sprintf("curseRemoved%d", i))
	}
	for i := 0; i < 12; i++ {
		in = append(in, fmt.Sprintf("itemCursed%d", i), fmt.Sprintf("itemDifficulty%d", i))
	}
	Nodes["00520bc0"] = Node{Addr: "00520bc0", In: in, Out: out, F: removeCursePowerApplyNode}
}

// The fixture supplies eight characters and twelve independent item definitions.
// These bounds describe arena records, not limits on the game's item database.
// Item definitions are input data; this rule reads no static exe table.
func removeCursePowerApplyNode(in []int64, r Rand) []int64 {
	if len(in) != 51 || in[0] < 0 || in[0] > 7 ||
		in[1] < 0 || in[1] > 4294967295 || in[2] < -2147483648 || in[2] > 2147483647 {
		panic("remove curse: expected fixture character 0..7, u32 power and i32 bonus percent")
	}
	for i := 0; i < 12; i++ {
		if in[3+2*i] < -1 || in[3+2*i] > 11 || in[4+2*i] < 0 || in[4+2*i] > 255 {
			panic("remove curse: expected fixture item -1..11 and u8 removed flag")
		}
	}
	for _, v := range in[27:] {
		if v < 0 || v > 255 {
			panic("remove curse: expected u8 curse flags and difficulties")
		}
	}
	// Threefold power: 0x520be2. Percentage divisor 100: 0x517f9e..
	// 0x517fa5. The helper uses an unsigned, wrapped product and sum,
	// unlike SpellDamage's signed, wide percentage arithmetic.
	power := uint32(in[1]) * 3
	power += (power * uint32(in[2])) / 100
	out := make([]int64, 13)
	succeeded, failed := 0, 0
	for i := 0; i < 12; i++ {
		item := in[3+2*i]
		out[1+i] = in[4+2*i]
		// Empty sentinel -1: 0x520c0d; any nonzero curse byte counts.
		if item == -1 || in[27+2*item] == 0 {
			continue
		}
		// Inclusive unsigned threshold: 0x520c38. Prior removed flags
		// do not affect either count; successful writes always set 1.
		if power >= uint32(in[28+2*item]) {
			succeeded++
			out[1+i] = 1
		} else {
			failed++
		}
	}
	// Result codes 0/1/2: 0x520c4c..0x520c61, respectively no success,
	// success with remaining failures, success without failures.
	if succeeded > 0 {
		out[0] = 1
		if failed == 0 {
			out[0] = 2
		}
	}
	return out
}
