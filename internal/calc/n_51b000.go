package calc

func init() {
	Nodes["0051b000"] = Node{
		Addr: "0051b000",
		In: []string{
			"attacker", "target",
			"character0Quadrant", "character1Quadrant", "character2Quadrant", "character3Quadrant",
			"character4Quadrant", "character5Quadrant", "character6Quadrant", "character7Quadrant",
			"centerSlot0", "centerSlot1", "centerSlot2",
			"character0FormationUnavailable", "character1FormationUnavailable",
			"character2FormationUnavailable", "character3FormationUnavailable",
			"character4FormationUnavailable", "character5FormationUnavailable",
			"character6FormationUnavailable", "character7FormationUnavailable",
		},
		Out: []string{"rangePenalty"},
		F:   partyFormationRangePenaltyNode,
	}
}

// The formation and character records are mutable runtime inputs, not exe
// constant tables. Quadrants: 0x687525 + character*12 (0x51b008/0x51b016).
// Center slots: 0x68751d..0x68751f (0x51b05b); unavailable flags:
// [0x685174] + character*0x1862 + 0x17b5 (0x51b065..0x51b080).
func partyFormationRangePenaltyNode(in []int64, r Rand) []int64 {
	// Eight party records: 0x55494e..0x554964 (also n_51ae60.go).
	// Center region 4, opposite separation 2: 0x51b02e..0x51b04d.
	// Three slots, empty byte 255: 0x51b061/0x51b085.
	const partySize, centerRegion, oppositeSeparation, centerSlots, emptySlot = 8, 4, 2, 3, 255
	const quadrantsStart = 2
	const slotsStart = quadrantsStart + partySize
	const flagsStart = slotsStart + centerSlots
	if len(in) != flagsStart+partySize {
		panic("party formation range: expected two indices, eight quadrants, three center slots and eight flags")
	}
	for _, index := range in[:quadrantsStart] {
		if index < 0 || index >= partySize {
			panic("party formation range: expected character index 0..7")
		}
	}
	for _, value := range in[quadrantsStart:] {
		if value < 0 || value > 255 {
			panic("party formation range: expected u8 formation fields")
		}
	}
	for _, slot := range in[slotsStart:flagsStart] {
		if slot != emptySlot && slot >= partySize {
			panic("party formation range: expected center character index 0..7 or empty 255")
		}
	}
	a, b := in[quadrantsStart+int(in[0])], in[quadrantsStart+int(in[1])]
	if a == b || a == centerRegion || b == centerRegion {
		return []int64{0}
	}
	// The target byte is interpreted as signed at 0x51b037; preserve this
	// asymmetry even for quadrant bytes outside the game's regions 0..4.
	separation := a - int64(int8(b))
	if separation < 0 {
		separation = -separation
	}
	if separation != oppositeSeparation {
		return []int64{0}
	}
	for _, slot := range in[slotsStart:flagsStart] {
		if slot != emptySlot && in[flagsStart+int(slot)] == 0 {
			return []int64{1}
		}
	}
	// Callers consume AL, so case output reads only the low byte (0x51b095).
	return []int64{0}
}
