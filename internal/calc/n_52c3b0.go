package calc

func init() {
	inputs := make([]string, 0, 112)
	// 0x52c454: eight party slots; 0x52c3f1/0x52c407: six SP realms.
	for _, slot := range []string{"0", "1", "2", "3", "4", "5", "6", "7"} {
		inputs = append(inputs, "slotUsed"+slot, "mainStatus"+slot)
		for _, field := range []string{"maxSp", "currentSp"} {
			for _, realm := range []string{"0", "1", "2", "3", "4", "5"} {
				inputs = append(inputs, field+slot+"Realm"+realm)
			}
		}
	}
	Nodes["0052c3b0"] = Node{
		Addr: "0052c3b0", In: inputs, Out: []string{"targetIndex"}, F: lowestSpRatioTargetNode,
	}
}

// lowestSpRatioTargetNode takes eight groups of slotUsed (u8), mainStatus
// (u32), six maxSp (u32), and six currentSp (i32). Globals 0x685174/0x685178
// locate caller-supplied main/secondary records with strides 0x1862/0x106.
// Main fields are +0xb01, +0xb25 and +0xb45; secondary +0 is slotUsed.
// These records are inputs, not static exe tables (0x52c3b0..0x52c470).
func lowestSpRatioTargetNode(in []int64, _ Rand) []int64 {
	if len(in) != 112 {
		panic("lowest SP ratio target: expected eight groups of fourteen fields")
	}
	for i, v := range in {
		lo, hi := int64(0), int64(4294967295) // u32 fields at 0x52c3e7/0x52c3f6.
		if i%14 == 0 {
			hi = 255 // u8 slot flag at 0x52c3e2.
		} else if i%14 >= 8 {
			lo, hi = -2147483648, 2147483647 // signed current SP at 0x52c40c..0x52c419.
		}
		if v < lo || v > hi {
			panic("lowest SP ratio target: expected u8 slotUsed, u32 status/maxSp, i32 currentSp")
		}
	}
	// 0x52c3bd/0x52c3cc: default target zero, initial percentage threshold 100.
	target, lowest := int64(0), uint32(100)
	for slot := 0; slot < 8; slot++ { // 0x52c454.
		fields := in[slot*14 : slot*14+14]
		if fields[0] == 0 || fields[1] >= 18 { // 0x52c3e2..0x52c3eb.
			continue
		}
		var maximum, current uint32
		for _, v := range fields[2:8] {
			maximum += uint32(v) // 0x52c3f6: sum wraps to u32.
		}
		if maximum == 0 { // 0x52c3fe: zero total skips the character.
			continue
		}
		for _, v := range fields[8:14] {
			if v > 0 { // 0x52c410..0x52c41b: only positive current SP contributes.
				current += uint32(v)
			}
		}
		// 0x52c420..0x52c42b: multiply by 100 with u32 wrap, unsigned division.
		ratio := current * 100 / maximum
		if ratio < lowest { // 0x52c42d..0x52c43b: strict comparison preserves ties.
			lowest, target = ratio, int64(slot)
		}
	}
	return []int64{target}
}
