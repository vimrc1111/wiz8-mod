package calc

func init() {
	Nodes["0051ae60"] = Node{
		Addr: "0051ae60",
		In: []string{
			"quadrant",
			"quadrant0Slot0", "quadrant0Slot1", "quadrant0Slot2",
			"quadrant1Slot0", "quadrant1Slot1", "quadrant1Slot2",
			"quadrant2Slot0", "quadrant2Slot1", "quadrant2Slot2",
			"quadrant3Slot0", "quadrant3Slot1", "quadrant3Slot2",
			"quadrant4Slot0", "quadrant4Slot1", "quadrant4Slot2",
			"character0FormationUnavailable", "character1FormationUnavailable",
			"character2FormationUnavailable", "character3FormationUnavailable",
			"character4FormationUnavailable", "character5FormationUnavailable",
			"character6FormationUnavailable", "character7FormationUnavailable",
		},
		Out: []string{"quadrantAvailable"},
		F:   formationQuadrantAvailableNode,
	}
}

// formationQuadrantAvailableNode takes the five formation regions and eight
// party records' unavailable flags. The runtime arrays are inputs, not exe
// constant tables: slots start at 0x687511; flags at [0x685174]+0x17b5
// with record stride 0x1862 (notes/re/growth.md:67).
func formationQuadrantAvailableNode(in []int64, r Rand) []int64 {
	// Five regions: 0x55498e..0x554997. Three slots per region and the
	// 255 empty sentinel: 0x51ae76..0x51ae82, 0x51aea4..0x51aea7.
	// Eight party records: 0x55494e..0x554964 advances the companion
	// record offset by 0x106 until 0x830 (eight entries).
	const regions, slotsPerRegion, partySize, emptySlot = 5, 3, 8, 255
	const flagsStart = 1 + regions*slotsPerRegion
	if len(in) != flagsStart+partySize || in[0] < 0 || in[0] > 4294967295 {
		panic("formation quadrant: expected u32 quadrant, 15 slots and 8 u8 unavailable flags")
	}
	// Only the argument's low byte selects a region (0x51ae6a).
	quadrant := int(uint8(in[0]))
	if quadrant >= regions {
		panic("formation quadrant: low byte must select region 0..4")
	}
	for _, slot := range in[1:flagsStart] {
		if slot != emptySlot && (slot < 0 || slot >= partySize) {
			panic("formation quadrant: expected party slot 0..7 or empty 255")
		}
	}
	for _, flag := range in[flagsStart:] {
		if flag < 0 || flag > 255 {
			panic("formation quadrant: expected u8 unavailable flag")
		}
	}
	available := int64(0)
	start := 1 + quadrant*slotsPerRegion
	for _, slot := range in[start : start+slotsPerRegion] {
		// A zero flag qualifies even if other character states would exclude
		// that member elsewhere; this node reads no HP or primary status.
		if slot != emptySlot && in[flagsStart+int(slot)] == 0 {
			available = 1
		}
	}
	// The bool occupies AL (0x51aead); callers consume only that byte
	// (0x53c82b, 0x53c839), so the case output reads one byte of eax.
	return []int64{available}
}
