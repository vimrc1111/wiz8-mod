package calc

// partySpHealNode takes a signed recovery amount, then eight independent
// character records: active, main status, HP, six maximum and six current SP.
// Notification text uses the same empty-name, absent-renderer setup as 52b910.
func partySpHealNode(in []int64, r Rand) []int64 {
	// Eight slots, fifteen values each (0x52ba62..0x52ba75).
	if len(in) != 121 || in[0] < -2147483648 || in[0] > 2147483647 {
		panic("party SP heal: expected i32 amount and eight character records")
	}
	out := make([]int64, 0, 48)
	for i := 0; i < 8; i++ {
		ch := in[1+i*15 : 1+(i+1)*15]
		for j, v := range ch {
			limit := int64(4294967295)
			if j == 0 {
				limit = 255 // Slot active flag is u8 at 0x52ba19.
			}
			if v < 0 || v > limit {
				panic("party SP heal: character field outside storage width")
			}
		}
		current := ch[9:15]
		// Main status is compared unsigned with 18 (0x52ba1f); HP at
		// character +0xb11 need only be nonzero (0x52ba29..0x52ba32).
		if ch[0] != 0 && ch[1] < 18 && ch[2] != 0 {
			amount := uint32(in[0])
			if in[0] < 0 {
				amount = 0
				// Six maxima at character +0xb25, summed modulo 2^32
				// (0x52ba3c..0x52ba50), even for already full pools.
				for _, maximum := range ch[3:9] {
					amount += uint32(maximum)
				}
			}
			child := make([]int64, 14)
			child[0], child[1] = int64(i), int64(amount)
			copy(child[2:], ch[3:15])
			current = characterSpHealDistributeNode(child, r)[:6]
		}
		out = append(out, current...)
	}
	return out
}

func init() {
	// Realm order follows notes/re/combat-2.md:71; current SP starts
	// at character +0xb45 (0x52b99f). Each character stride is 0x1862,
	// and each active-slot stride is 0x106 (0x52ba62..0x52ba69).
	fields := []string{"active", "mainStatus", "HP", "fireMaximumSP", "waterMaximumSP",
		"airMaximumSP", "earthMaximumSP", "mentalMaximumSP", "divineMaximumSP",
		"fireCurrentSP", "waterCurrentSP", "airCurrentSP", "earthCurrentSP", "mentalCurrentSP", "divineCurrentSP"}
	inputs, outputs := []string{"healAmount"}, []string{}
	for i := 0; i < 8; i++ {
		prefix := "character" + string(rune('0'+i)) + "_"
		for j, field := range fields {
			inputs = append(inputs, prefix+field)
			if j >= 9 {
				outputs = append(outputs, prefix+field)
			}
		}
	}
	Nodes["0052ba00"] = Node{Addr: "0052ba00", In: inputs, Out: outputs, F: partySpHealNode}
}
