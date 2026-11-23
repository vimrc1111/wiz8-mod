package calc

// Spell reflection (0x4fe740): swaps the cast source descriptor and the
// target record. Fixture wiring (case 004fe740): eight party slots, monster
// IDs 100/101 active and 101/102 reserve, group IDs 200/201 active and
// 201/202 reserve. Lookups return the first match, active before reserve
// (0x4e5550, 0x5100b0), so the duplicate-ID reserve records are never read.
// Input layout: 0-14 source descriptor (dwords 0-6, byte 0x1c, byte 0x1d,
// half 0x1e, dwords 8-12), 15-22 target record, 23-46 party slots
// (active byte, main status, current HP), 47-50 monster records' group ID,
// 51-54 group records' leader monster ID. Outputs: source 0-14, target 15-22.
const (
	srtNeg1 = 0xffffffff
	srtIn   = 55
)

func spellReflectedSourceTargetNode(in []int64, _ Rand) []int64 {
	if len(in) != srtIn {
		panic("spell reflected source target: expected 55 inputs")
	}
	if in[7]&0xff != 0 {
		panic("spell reflected source target: source byte 0x1c must be zero") // 0x4fe752 assert.
	}
	out := make([]int64, 23)
	copy(out, in[:23])
	s, t := out[:15], out[15:]
	src := [3]int64{in[0], in[1], in[2]} // entry copy: kind, character index, monster ID.
	clear := func(v []int64) {
		for i := range v {
			v[i] = 0
		}
	}
	// 0x53be00: character descriptor over the whole source record.
	charSource := func(idx int64) {
		clear(s)
		s[0], s[1], s[2] = 1, idx, srtNeg1
	}
	// 0x53be50: monster descriptor; its monster ID is the matched record's ID.
	monsterSource := func(id int64) {
		clear(s)
		s[0], s[1], s[2] = 2, srtNeg1, id
	}
	// 0x4e5550 / 0x4e5620 / 0x4e5840: first match of ID, active before reserve; a miss asserts.
	monsterGroupID := func(id int64) int64 {
		switch id {
		case 100:
			return in[47]
		case 101:
			return in[48]
		case 102:
			return in[50]
		}
		panic("spell reflected source target: monster ID outside replay fixture")
	}
	// 0x5100b0 / 0x5101b0: group record, then its leader monster ID at +0x9f.
	groupLeader := func(id int64) int64 {
		switch id {
		case 200:
			return in[51]
		case 201:
			return in[52]
		case 202:
			return in[54]
		}
		panic("spell reflected source target: group ID outside replay fixture")
	}
	// 0x4fe79f..0x4fe802: the old source becomes the target (0x53bea0 / 0x53bf10).
	swapTarget := func() {
		switch src[0] {
		case 1:
			if src[1] == srtNeg1 {
				panic("spell reflected source target: character source without index") // 0x53beb1 assert.
			}
			t[0], t[1] = 1, src[1]
		case 2:
			if src[2] == srtNeg1 {
				panic("spell reflected source target: monster source without ID") // 0x53bf21 assert.
			}
			t[0], t[2] = 3, src[2]
		default:
			panic("spell reflected source target: source kind must be character (1) or monster (2)") // 0x4fe802 assert.
		}
	}
	// 0x4fe8e8..0x4fe9ec: reset the target by 0x536170, then pick its kind from the source kind.
	resetTarget := func() {
		clear(t)
		t[1], t[2], t[3] = srtNeg1, srtNeg1, srtNeg1
		switch src[0] {
		case 1:
			t[0] = 2
		case 2:
			t[0] = 4
			t[3] = monsterGroupID(src[2])
		default:
			panic("spell reflected source target: source kind must be character (1) or monster (2)") // 0x4fe8fe / 0x4fe9af assert.
		}
	}
	switch uint32(t[0]) {
	case 1, 9:
		charSource(t[1])
		swapTarget()
	case 3:
		monsterGroupID(t[2]) // the target monster must exist (0x4fe820).
		monsterSource(t[2])
		swapTarget()
	case 2:
		// 0x4fe8a0..0x4fe8c1: first slot that is active, has nonzero HP and main status below 0x12.
		slot := -1
		for i := 0; i < 8 && slot < 0; i++ {
			if in[23+3*i]&0xff != 0 && uint32(in[25+3*i]) != 0 && uint32(in[24+3*i]) < 0x12 {
				slot = i
			}
		}
		if slot < 0 {
			panic("spell reflected source target: no eligible party slot") // 0x4fe8c3 assert.
		}
		charSource(int64(slot))
		resetTarget()
	case 4:
		m := groupLeader(t[3])
		monsterGroupID(m) // the leader monster must exist (0x4fe988).
		monsterSource(m)
		resetTarget()
	default:
		panic("spell reflected source target: target kind must be 1, 2, 3, 4 or 9") // 0x4fe9fe assert.
	}
	s[8] = 1 // source +0x1d redirect flag.
	return out
}

func init() {
	names := []string{"sourceKind", "sourceCharacterIndex", "sourceMonsterId", "srcDword3", "srcDword4", "srcDword5", "srcDword6",
		"sourceByte1c", "sourceRedirectFlag", "sourceHalf1e", "srcDword8", "srcDword9", "srcDword10", "srcDword11", "srcDword12",
		"targetKind", "targetCharacterIndex", "targetMonsterId", "targetGroupId", "targetDword4", "targetDword5", "targetDword6", "targetDword7"}
	in := append([]string{}, names...)
	for i := 0; i < 8; i++ {
		d := string(rune('0' + i))
		in = append(in, "slot"+d+"Active", "char"+d+"MainStatus", "char"+d+"CurrentHP")
	}
	for i := 0; i < 4; i++ {
		in = append(in, "monster"+string(rune('0'+i))+"GroupId")
	}
	for i := 0; i < 4; i++ {
		in = append(in, "group"+string(rune('0'+i))+"LeaderMonsterId")
	}
	Nodes["004fe740"] = Node{Addr: "004fe740", In: in, Out: names, F: spellReflectedSourceTargetNode}
}
