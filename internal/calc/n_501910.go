package calc

func init() {
	Nodes["00501910"] = Node{
		Addr: "00501910",
		In:   []string{"characterIndex", "combatActive", "difficulty", "actionProgress", "risk"},
		Out:  []string{"risk"}, F: combatSpellRiskPhaseScalingNode,
	}
}

// Eight arena slots model the party's combat records, stride 0xd4, with
// action progress at +0xb4 (0x50195b..0x501972). No static table is read.
// The output is the updated risk pointee; EAX is incidental arithmetic state.
func combatSpellRiskPhaseScalingNode(in []int64, _ Rand) []int64 {
	if len(in) != 5 || in[0] < 0 || in[0] > 7 || in[1] < 0 || in[1] > 255 {
		panic("spell risk: expected character slot 0..7 and u8 combat flag")
	}
	for _, v := range in[2:] {
		if v < 0 || v > 4294967295 {
			panic("spell risk: expected u32 difficulty, progress and risk")
		}
	}
	risk := uint32(in[4])
	if in[1] == 0 { // 0x501910..0x501917: any nonzero combat flag enables scaling.
		return []int64{int64(risk)}
	}
	var threshold uint32
	switch in[2] {
	case 0:
		threshold = 80 // 0x501956: easy.
	case 1:
		threshold = 60 // 0x50194f: normal.
	case 2:
		threshold = 40 // 0x501948: expert.
	default:
		panic("spell risk: invalid combat difficulty triggers srAssertFail at 0x50192d")
	}
	progress := uint32(in[3])
	if progress >= threshold { // Inclusive unsigned comparison at 0x501978.
		// 0x50197c..0x501990: offset 50, twice the wrapped product.
		// Thresholds are even, so the extra parity term is always zero.
		// 0x501992..0x501999 implements unsigned division by 100.
		risk = ((progress - threshold + 50) * risk * 2) / 100
	}
	return []int64{int64(risk)}
}
