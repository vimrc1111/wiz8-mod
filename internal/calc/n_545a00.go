package calc

func init() {
	Nodes["00545a00"] = Node{
		Addr: "00545a00",
		In: []string{"damage", "targetType", "characterIndex", "monsterID",
			"characterMaxHP0", "characterMaxHP1", "characterMaxHP2", "characterMaxHP3",
			"characterMaxHP4", "characterMaxHP5", "characterMaxHP6", "characterMaxHP7",
			"monsterMaxHP0", "monsterMaxHP1"},
		Out: []string{"damage"}, F: fumbleDamageCapNode,
	}
}

// The replay fixture has eight party records and two monster records: ID 0
// in the active list, ID 1 in the reserve list. Original helpers 0x4e5550
// and 0x4e5620 resolve those IDs before reading M+0x27 (0x545a57).
func fumbleDamageCapNode(in []int64, r Rand) []int64 {
	if len(in) != 14 {
		panic("fumble damage cap: expected damage, target selectors, eight character HPs and two monster HPs")
	}
	for _, v := range in {
		// Original damage and HP reads/comparisons are u32 (0x545a7b..0x545a99).
		if v < 0 || v > 4294967295 {
			panic("fumble damage cap: expected u32 inputs")
		}
	}
	if in[2] > 7 || in[3] > 1 {
		panic("fumble damage cap: selector outside replay fixture")
	}
	var hp int64
	switch in[1] {
	case 1: // Character target at G+0x8c5, read C+0xb0d (0x545a0b..0x545a2b).
		hp = in[4+in[2]]
	case 3: // Monster target at G+0x8c5 (0x545a34).
		hp = in[12+in[3]]
	default:
		panic("fumble damage cap: target must be a character (1) or monster (3)")
	}
	// 0x545a7b..0x545a8a: unsigned HP / 5, with a minimum cap of 1.
	cap := max(int64(1), hp/5)
	return []int64{min(in[0], cap)}
}
