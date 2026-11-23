package calc

func init() {
	Nodes["00520a70"] = Node{Addr: "00520a70",
		In:  []string{"itemIndex", "identifyRequirement", "bonusPercent"},
		Out: []string{"identifyPowerRequired"}, F: identifyPowerRequiredNode}
}

// identifyPowerRequiredNode consumes the selected item's requirement directly;
// the item records are mutable game state, not an immutable exe data table.
func identifyPowerRequiredNode(in []int64, _ Rand) []int64 {
	if len(in) != 3 || in[0] < -1 || in[0] > 2147483647 || in[1] < 0 || in[1] > 255 || in[2] < -2147483648 || in[2] > 2147483647 {
		panic("identify power: expected item index -1 or nonnegative i32, u8 requirement and i32 percentage")
	}
	// Empty item sentinel and minimum power 1: exe 0x520a98 and 0x520b2d.
	if in[0] == -1 {
		return []int64{1}
	}
	requirement := uint32(in[1])
	// Three requirement points per power level: exe 0x520ad1 and 0x520ae6.
	power := (requirement + 2) / 3
	for power > 1 {
		threshold := (power - 1) * 3
		// Child 0x517f90 uses the low 32 bits of the product, unsigned
		// division by 100, then wrapping addition (exe 0x517f99..0x517faa).
		// SpellDamage instead uses signed wide arithmetic; retain the child
		// semantics here, including negative percentages and overflow.
		adjusted := threshold + threshold*uint32(in[2])/100
		if adjusted < requirement {
			return []int64{int64(power)}
		}
		power--
	}
	return []int64{1}
}
