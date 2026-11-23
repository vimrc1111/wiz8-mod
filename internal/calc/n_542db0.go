package calc

// chooseAttackModeNode selects the R(9)-th enabled mode, counting from zero
// across repeated scans. This is not a uniform draw among enabled modes.
func chooseAttackModeNode(in []int64, r Rand) []int64 {
	if len(in) != 1 || in[0] < 0 || in[0] > 4294967295 {
		panic("choose attack mode: expected u32 attackModes")
	}
	// Only indices 0..8 are scanned (0x542e01); the input is first masked
	// to 16 bits (0x542de6). Bits above index 8 cannot select a mode.
	const modeCount = 9
	const selectableMask = 0x1ff // Bits 0..8 from the scan bounds at 0x542dec..0x542e04.
	mask := uint32(in[0]) & selectableMask
	if mask == 0 {
		// Low 16 bits zero assert at 0x542dbc; other empty selectable masks
		// loop forever at 0x542dec..0x542e08. Neither has a return value.
		panic("choose attack mode: no selectable mode (original asserts or never returns)")
	}
	remaining := r.R(modeCount) // 0x542dd6..0x542dda: one R(9), even for a single mode.
	for {
		for mode := 0; mode < modeCount; mode++ {
			if mask&(1<<mode) != 0 {
				if remaining == 0 {
					return []int64{int64(mode)}
				}
				remaining--
			}
		}
	}
}

func init() {
	Nodes["00542db0"] = Node{
		Addr: "00542db0",
		In:   []string{"attackModes"},
		Out:  []string{"attackMode"}, F: chooseAttackModeNode,
	}
}
