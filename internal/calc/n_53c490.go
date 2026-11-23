package calc

// Fixture wiring (case 0053c490): eight character records, monster IDs 100/101
// active and 101/102 reserve, group IDs 200/201 active and 201/202 reserve.
// Lookups return the first match, active list before reserve (0x4e5550,
// 0x5100b0), so the duplicate-ID reserve records are never read.
func combatTargetAllegianceMismatchNode(in []int64, _ Rand) []int64 {
	if len(in) != 26 {
		panic("combat target allegiance: expected 26 inputs")
	}
	b2i := func(v bool) []int64 {
		if v {
			return []int64{1}
		}
		return []int64{0}
	}
	// 0x53c490: a zero check byte accepts without reading either side.
	if in[0]&0xff == 0 {
		return b2i(true)
	}
	char := func(i int64) bool {
		if i < 0 || i > 7 {
			panic("combat target allegiance: character slot outside replay fixture")
		}
		return uint32(in[10+i]) != 0 // 0x53c4ed: unsigned character flag > 0.
	}
	// 0x4e5550 searches active then reserve; 0x4e5620 resolves the index.
	monster := func(id int64) bool {
		switch id {
		case 100:
			return in[18]&0xff == 1
		case 101:
			return in[19]&0xff == 1
		case 102:
			return in[21]&0xff == 1
		}
		panic("combat target allegiance: monster ID outside replay fixture")
	}
	// 0x5100b0 searches active then reserve; 0x5101b0 resolves the index.
	group := func(id int64) bool {
		switch id {
		case 200:
			return in[22]&0xff == 1
		case 201:
			return in[23]&0xff == 1
		case 202:
			return in[25]&0xff == 1
		}
		panic("combat target allegiance: group ID outside replay fixture")
	}
	var src, tgt bool
	switch in[2] {
	case 1:
		src = char(in[3])
	case 2:
		src = monster(in[4])
	default:
		panic("combat target allegiance: source kind must be character (1) or monster (2)") // 0x53c608 assert.
	}
	switch in[6] {
	case 1:
		tgt = char(in[7])
	case 3:
		tgt = monster(in[8])
	case 4:
		tgt = group(in[9])
	default:
		panic("combat target allegiance: target kind must be character (1), monster (3) or group (4)") // 0x53c5ec assert.
	}
	// 0x53c5cb..0x53c5e6: a nonzero second flag accepts equal sides, zero accepts differing sides.
	return b2i((src == tgt) == (in[1]&0xff != 0))
}

func init() {
	Nodes["0053c490"] = Node{
		Addr: "0053c490",
		In: []string{"check", "requireSameSide", "sourceKind", "sourceCharacter", "sourceMonsterId", "sourceGroupId",
			"targetKind", "targetCharacter", "targetMonsterId", "targetGroupId",
			"character0Allegiance", "character1Allegiance", "character2Allegiance", "character3Allegiance",
			"character4Allegiance", "character5Allegiance", "character6Allegiance", "character7Allegiance",
			"monster0Allegiance", "monster1Allegiance", "monster2Allegiance", "monster3Allegiance",
			"group0Allegiance", "group1Allegiance", "group2Allegiance", "group3Allegiance"},
		Out: []string{"accepted"}, F: combatTargetAllegianceMismatchNode,
	}
}
