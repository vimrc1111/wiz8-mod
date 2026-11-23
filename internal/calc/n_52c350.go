package calc

func init() {
	inputs := make([]string, 0, 32)
	// 0x52c39e: the scan covers all eight party slots.
	for _, slot := range []string{"0", "1", "2", "3", "4", "5", "6", "7"} {
		for _, field := range []string{"slotUsed", "mainStatus", "hp", "maxHp"} {
			inputs = append(inputs, field+slot)
		}
	}
	Nodes["0052c350"] = Node{
		Addr: "0052c350", In: inputs, Out: []string{"targetIndex"}, F: lowestHpRatioTargetNode,
	}
}

// lowestHpRatioTargetNode takes eight groups of slotUsed (u8), mainStatus,
// hp and maxHp (u32). The globals 0x685174/0x685178 locate the main/secondary
// records (strides 0x1862/0x106); fields are +0xb01/+0xb11/+0xb0d and +0.
// These are caller-supplied records, not static exe tables (0x52c350..0x52c3a9).
func lowestHpRatioTargetNode(in []int64, _ Rand) []int64 {
	if len(in) != 32 {
		panic("lowest HP ratio target: expected eight groups of slotUsed, mainStatus, hp, maxHp")
	}
	for i, v := range in {
		limit := int64(4294967295) // u32 record fields read at 0x52c373..0x52c386.
		if i%4 == 0 {
			limit = 255 // u8 secondary-record flag at 0x52c36e.
		}
		if v < 0 || v > limit {
			panic("lowest HP ratio target: expected u8 slotUsed and u32 mainStatus, hp, maxHp")
		}
	}
	// 0x52c35f..0x52c366: index defaults to zero and the initial threshold is 100.
	target, lowest := int64(0), uint32(100)
	for slot := 0; slot < 8; slot++ { // 0x52c39e: eight slots, including companions.
		fields := in[slot*4 : slot*4+4]
		if fields[0] == 0 || fields[1] >= 18 { // 0x52c36e..0x52c377.
			continue
		}
		if fields[3] == 0 {
			panic("lowest HP ratio target: eligible character has zero maxHp (original division fault)")
		}
		// 0x52c379..0x52c386: wrap the product to u32 before unsigned division.
		ratio := uint32(fields[2]) * 100 / uint32(fields[3])
		if ratio < lowest { // 0x52c389..0x52c38f: ties retain the earlier slot.
			lowest, target = ratio, int64(slot)
		}
	}
	return []int64{target}
}
