package calc

func init() {
	Nodes["00547080"] = Node{
		Addr: "00547080",
		In: []string{"targetType", "characterIndex", "characterCharm", "monsterID",
			"active0ID", "active0SummonOwner", "active0Disposition",
			"active1ID", "active1SummonOwner", "active1Disposition",
			"reserveID", "reserveSummonOwner", "reserveDisposition"},
		Out: []string{"oppositeDisposition"}, F: oppositeDispositionNode,
	}
}

// oppositeDispositionNode resolves a target against two active monsters and
// one reserve monster. List pointers/counts are fixture structure; IDs and
// rule fields remain inputs. No static tables or random calls are involved.
func oppositeDispositionNode(in []int64, _ Rand) []int64 {
	if len(in) != 13 {
		panic("opposite disposition: expected target and three monster records")
	}
	for i, v := range in {
		// u32 target/ID/charm/owner reads: 0x53bea4, 0x4e5570,
		// 0x5470ac, 0x547104. Disposition is u8 at 0x547111.
		limit := int64(4294967295)
		if i == 1 {
			limit = 7 // Party record indices; stride 0x1862 at 0x547097..0x5470ac.
		} else if i >= 6 && i%3 == 0 {
			limit = 255
		}
		if v < 0 || v > limit {
			panic("opposite disposition: field outside its record range")
		}
	}
	// 0x53bea6 / 0x53bf16 distinguish character=1 and monster=2.
	// The caller passes zero for the optional composite-target recognition.
	if in[0] == 1 {
		if in[2] == 0 { // C+0xa35, 0x5470b3; hostile opposite=1 at 0x547117.
			return []int64{1}
		}
		return []int64{0}
	}
	if in[0] != 2 {
		return []int64{0}
	}
	if in[3] == 4294967295 { // Invalid monster ID asserted by 0x53bf1b.
		panic("opposite disposition: monster ID is the invalid sentinel")
	}
	// 0x4e5550 scans the active list first, then reserve (handle base 10000
	// at 0x4e55d3); 0x4e5620 resolves either handle to the same record.
	for i := 4; i < len(in); i += 3 {
		if in[i] != in[3] {
			continue
		}
		if in[i+1] != 0 { // M+0x8b at 0x547104: summoned/owned has no opposite.
			return []int64{0}
		}
		if in[i+2] == 2 { // Friendly=2 at 0x547111; opposite hostile=1.
			return []int64{1}
		}
		return []int64{2} // Every other byte returns friendly=2 (0x54711b).
	}
	// Missing ID is allowed: 0x5470e7 logs a diagnostic, returns low byte zero.
	return []int64{0}
}
