package calc

// combatPhaseResumeNode replays exe 0x4f0990: promote each active party
// member's unfinished action phase to the current combat phase, finish members
// whose phase exceeds 100, then hand over to the resume tail.
//
// The fixture has the UI object present (0x68edcc != 0) and the combat flag
// (0x683f94) set; a clear flag stops in srAssertFail at 0x4f09aa. The tail
// selector 0x90c is driven together with UI mode 0x68ec78 (0 selects the
// 0x5a1890 teardown path, which here only clears 0x683fcd and slot 0x1c of the
// table at 0x61f238 via 0x4f2220; any other value selects 0x5a1dd0).
// 0x4f0a58 is unreachable because 0x910 is zeroed at 0x4f09bf.
func combatPhaseResumeNode(in []int64, r Rand) []int64 {
	if len(in) != 27 {
		panic("combat phase resume: expected 27 inputs")
	}
	for i, v := range in {
		if v < 0 || v > 4294967295 || i >= 3 && i < 11 && v > 255 || i >= 19 && v > 255 {
			panic("combat phase resume: input out of range")
		}
	}
	phase, mode, uiFlags := uint32(in[0]), uint32(in[1]), uint32(in[2])
	var val, stamp, mem [8]uint32
	var done [8]uint8
	for i := range val {
		val[i], done[i] = uint32(in[11+i]), uint8(in[19+i])
		stamp[i], mem[i] = 0x5555, 0x12345678 // case setup initial values
	}
	for i := 0; i < 8; i++ {
		if in[3+i] == 0 {
			continue
		}
		// 0x4f09dc..0x4f09ec: unsigned compare, only while not yet finished.
		if val[i] < phase && done[i] == 0 {
			val[i] = phase
		}
		stamp[i] = phase
		if val[i] > 100 { // 0x4f09f7, unsigned
			done[i], val[i], mem[i] = 1, 0, 0xffffffff
			// 0x562a50 sets the bit only in UI mode 7.
			if mode == 7 {
				uiFlags |= 1<<uint(i) | 0x100000
			}
		}
	}
	limitA, limitB, status, slot := uint32(7), uint32(9), uint32(1), uint32(0xdeadbeef)
	if mode == 0 {
		status, slot = 0, 0
	} else {
		limitA, limitB = 100, 100 // 0x4f0a76/0x4f0a82
	}
	out := make([]int64, 0, 38)
	for _, a := range [][]uint32{val[:], u8s(done), stamp[:], mem[:]} {
		for _, v := range a {
			out = append(out, int64(v))
		}
	}
	return append(out, int64(uiFlags), int64(limitA), int64(limitB), 0, int64(status), int64(slot))
}

func u8s(a [8]uint8) []uint32 {
	o := make([]uint32, 8)
	for i, v := range a {
		o[i] = uint32(v)
	}
	return o
}

func init() {
	var inN, outN []string
	inN = []string{"currentPhase", "uiMode", "uiFlags"}
	for _, p := range []string{"active", "phase", "done"} {
		for i := 0; i < 8; i++ {
			inN = append(inN, p+string(rune('0'+i)))
		}
	}
	for _, p := range []string{"phase", "done", "stamp", "memory"} {
		for i := 0; i < 8; i++ {
			outN = append(outN, p+string(rune('0'+i)))
		}
	}
	outN = append(outN, "uiFlags", "uiLimitA", "uiLimitB", "pendingFlag", "combatStatus", "clearedSlot")
	Nodes["004f0990"] = Node{Addr: "004f0990", In: inN, Out: outN, F: combatPhaseResumeNode}
}
