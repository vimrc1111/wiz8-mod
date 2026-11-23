package calc

// bestPartySkillNode consumes skillIndex, bestIndexPtr, initialBestIndex,
// then (active byte, life u32, status u32, selected skill u32) for each slot.
// The pointer is represented only by whether the caller requests the write;
// the initial cell remains unchanged for a null pointer. No exe table is read.
func bestPartySkillNode(in []int64, r Rand) []int64 {
	// 0x554508/0x554515: 0xc310 / 0x1862 = 8 character slots.
	const slots = 8
	// 41 skill records: internal/calc/exe.go ExeTables.SkillAttr;
	// 0x5544d9–0x5544e2 selects character+0x1a3+skillIndex*0x26.
	const skills = 41
	if len(in) != 3+4*slots {
		panic("best_party_skill: expected skill index, output pointer, initial index and 8 members")
	}
	if in[0] < 0 || in[0] >= skills || in[1] < 0 || in[1] > 0xffffffff || in[2] < -2147483648 || in[2] > 2147483647 {
		panic("best_party_skill: expected skill 0..40, u32 pointer and i32 initial index")
	}
	best, who := int64(0), int64(-1) // 0x5544a0/0x5544ae: empty-party defaults.
	for slot := 0; slot < slots; slot++ {
		member := in[3+4*slot : 7+4*slot]
		if member[0] < 0 || member[0] > 255 {
			panic("best_party_skill: expected u8 active flag")
		}
		for _, value := range member[1:] {
			if value < 0 || value > 0xffffffff {
				panic("best_party_skill: expected u32 life, status and skill")
			}
		}
		// 0x5544c2/0x5544ca/0x5544ce: active, nonzero life, status < 13.
		if member[0] == 0 || member[1] == 0 || member[2] >= 13 {
			continue
		}
		// 0x5544e9–0x5544fc: unsigned maximum; the first eligible zero
		// also wins, and subsequent ties keep the earlier character slot.
		if member[3] > best || who == -1 {
			best, who = member[3], int64(slot)
		}
	}
	indexCell := in[2]
	if in[1] != 0 { // 0x554525: optional best-index output.
		indexCell = who
	}
	return []int64{best, indexCell}
}

func init() {
	in := []string{"skillIndex", "bestIndexPtr", "initialBestIndex"}
	for _, slot := range []string{"0", "1", "2", "3", "4", "5", "6", "7"} {
		for _, field := range []string{"Active", "Life", "Status", "Skill"} {
			in = append(in, "member"+slot+field)
		}
	}
	Nodes["00554490"] = Node{
		Addr: "00554490", In: in, Out: []string{"bestSkill", "bestIndex"}, F: bestPartySkillNode,
	}
}
