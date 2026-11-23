package calc

func init() {
	Nodes["0053c3f0"] = Node{
		Addr: "0053c3f0",
		In: []string{"sourceType", "sourceCharacterIndex", "sourceReflection", "sourceBackfire", "targetType", "targetCharacterIndex",
			"characterEffect0", "characterEffect1", "characterEffect2", "characterEffect3", "characterEffect4", "characterEffect5", "characterEffect6", "characterEffect7"},
		Out: []string{"effectSelector"}, F: combatTargetEffectSelectorNode,
	}
}

// The eight effect bytes are runtime character state, not an exe constant table.
// The low byte is the complete return value; callers at 0x5443e1 use that byte.
func combatTargetEffectSelectorNode(in []int64, r Rand) []int64 {
	if len(in) != 14 {
		panic("combat target effect selector: expected source, target and eight character effect bytes")
	}
	for _, i := range []int{0, 1, 4, 5} {
		if in[i] < -2147483648 || in[i] > 2147483647 {
			panic("combat target effect selector: expected i32 type and index")
		}
	}
	for _, i := range []int{2, 3, 6, 7, 8, 9, 10, 11, 12, 13} {
		if in[i] < 0 || in[i] > 255 {
			panic("combat target effect selector: expected u8 flags and effects")
		}
	}
	// Source kind 1 and BAD_INDEX -1: exe 0x53c3f6/0x53c3fb.
	if in[0] == 1 {
		if in[1] == -1 {
			panic("combat target effect selector: character source has BAD_INDEX")
		}
		// Fixed source effect 8: exe 0x53c451.
		return []int64{8}
	}
	// Kind 3 with a character index represents a redirected monster source.
	// exe 0x53c423/0x53c429/0x53c430 require backfire or reflection.
	if in[0] == 3 && in[1] != -1 {
		if in[2] == 0 && in[3] == 0 {
			panic("combat target effect selector: redirected monster needs backfire or reflection")
		}
		return []int64{8}
	}
	if in[4] == 1 {
		// exe 0x53c45f..0x53c478 reads Q+0xf1, stride 262 (0x106).
		// Eight party slots: notes/re/formulas.md:9 and exe 0x4ef420.
		if in[5] < 0 || in[5] > 7 {
			panic("combat target effect selector: expected character target index 0..7")
		}
		return []int64{in[6+int(in[5])]}
	}
	// exe 0x53c479..0x53c486: monster kind 3 yields 9; every other
	// non-character kind yields 12. The return type discards upper bits.
	if in[4] == 3 {
		return []int64{9}
	}
	return []int64{12}
}
