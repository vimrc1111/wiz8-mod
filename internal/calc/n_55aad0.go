package calc

func init() {
	Nodes["0055aad0"] = Node{
		Addr: "0055aad0",
		In:   []string{"itemIdx", "itemLevel", "slotActive0", "totalLevel0", "slotActive1", "totalLevel1"},
		Out:  []string{"stockClass"},
		F:    shopStockLevelClassNode,
	}
}

func i8(v int64) int64 { return int64(int8(v)) }

// shopStockLevelClassNode: in = itemIdx (selects the item record, unused by
// the rule), itemLevel (i8 at item record +0x85, exe 0x55aae8), and two
// party slot pairs for the average level (exe 0x4ef420, other slots empty).
// Output is al (callers movsx al at 0x55a6fc, 0x55accf).
func shopStockLevelClassNode(in []int64, r Rand) []int64 {
	if len(in) != 6 {
		panic("shopStockLevelClass: expected 6 inputs")
	}
	// exe 0x55aaef: average party level via the existing 0x4ef420 node.
	avg := partyAverageLevelNode([]int64{in[2], in[3], in[4], in[5], 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, r)[0]
	// exe 0x55aaf4..0x55ab28: every step narrows to i8 (movsx dl/cl/al).
	a := i8(i8(i8(avg)*100/30) / 5)
	cl := i8(a + 4)
	// exe 0x55ab2d..0x55ab41: lo = clamp(a,1,16) signed.
	lo := a
	if a > 16 {
		lo = 16
	} else if a < 1 {
		lo = 1
	}
	// exe 0x55ab43..0x55ab56: hi = clamp(a+4,4,20) signed.
	hi := cl
	if cl > 20 {
		hi = 20
	} else if cl < 4 {
		hi = 4
	}
	b := i8(in[1])
	switch {
	case b > hi:
		return []int64{0} // exe 0x55ab5a
	case b >= lo:
		return []int64{3} // exe 0x55ab62
	case b+4 >= lo: // exe 0x55ab66..0x55ab71 (movsx widened compare)
		return []int64{2}
	}
	return []int64{1}
}
