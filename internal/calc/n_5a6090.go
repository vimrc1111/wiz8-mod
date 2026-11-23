package calc

// combatActionChangeAllowedNode: 0x5a6090 returns AL = whether the party's
// action change is allowed. The denial paths also pop up a message (side
// effects only; they do not alter AL).
// in: gate (0x683f94 byte), who (character slot), limit49/45/41/3d (u32 at
// character record +0xa49/+0xa45/+0xa41/+0xa3d), phase (0x6836a8 +0xa50 byte),
// slotFlag (byte at +0x4c+who*0xd4 of the same block, unused by the result),
// slotA/slotB (i32 at 0x685178 record +1/+0x3d, stride 0x106).
// Value 9 is the compared record state (0x5a6133, 0x5a60ff..0x5a6254).
const combatActionChangeLockedState = 9

func combatActionChangeAllowedNode(in []int64, r Rand) []int64 {
	if len(in) != 10 {
		panic("combatActionChangeAllowed: expected 10 inputs")
	}
	gate, limited := in[0]&0xff != 0, in[2] != 0 || in[3] != 0 || in[4] != 0 || in[5] != 0
	phase := in[6]&0xff != 0
	// gate off: always allowed; any character limit: allowed only in phase;
	// no limit: allowed only in phase with slotA == locked state.
	if !gate || phase && (limited || int32(in[8]) == combatActionChangeLockedState) {
		return []int64{1}
	}
	return []int64{0}
}

func init() {
	Nodes["005a6090"] = Node{
		Addr: "005a6090",
		In:   []string{"gate", "who", "limit49", "limit45", "limit41", "limit3d", "phase", "slotFlag", "slotA", "slotB"},
		Out:  []string{"allowed"}, F: combatActionChangeAllowedNode,
	}
}
