package calc

// spellEffectActivationNode composes activation, unloaded-scene height lookup,
// and completion accounting. The fixture supplies one parent containing this
// effect, empty auxiliary-result lists, and a zero-initialized notification
// effect definition (ID 131). Notification dispatch runs in the emulator;
// with these records it has no RNG calls or accounting outputs.
// Sources: exe 0x4a4c20, 0x4a42b0, 0x4a4640, 0x420bd0,
// 0x4a49e0, 0x4fb4c0 and 0x500460. No executable table is embedded.
func spellEffectActivationNode(in []int64, r Rand) []int64 {
	if len(in) != 62 {
		panic("spell activation: expected 62 inputs")
	}
	out := []int64{in[4], in[5], in[6], in[7], in[9]}
	// Twenty-two u32 statistics at effect +0x280 and parent +0x126:
	// two header totals and twenty further counters, exe 0x5004c8..0x500506.
	for i := 0; i < 22; i++ {
		out = append(out, in[19+2*i])
	}
	// Virtual resource query 0x4a42b0 tests controller +0xdc for presence.
	if uint32(in[0]) != 0 {
		// Query 6 returns signed sequence byte (+0xa4), exe 0x4a46d6.
		if int8(in[1]) != 1 {
			out[1], out[2], out[3] = 1, 1, 1 // exe 0x4a4c63..0x4a4c74.
			if uint8(in[3]) != 0 {
				// No loaded terrain (0x652db0 == 0): 0x420c17 returns input Y.
				// The original replay preserves the raw Y bits, including NaNs.
				out[4] = int64(uint32(in[9]))
			}
		}
	} else {
		out[0] = 1 // completion byte, exe 0x4a4c9c.
		// Notification for spell 35 is suppressed only by a present combat state
		// in mode 2 (0x4a4c95..0x4a4cb8). The supplied empty definition leaves
		// these outputs unchanged, including when notification dispatch executes.
		if uint8(in[13]) != 0 {
			for i := 0; i < 22; i++ {
				out[5+i] = int64(uint32(in[18+2*i]) + uint32(in[19+2*i]))
			}
		}
	}
	return out
}

func init() {
	Nodes["004a4c20"] = Node{Addr: "004a4c20", In: []string{"activationResource", "sequence", "spellID", "snapToGround", "completed", "activated", "controllerActive", "controllerVisible", "xBits", "yBits", "zBits", "combatState", "combatMode", "aggregateOnCompletion", "casterSlot", "worldXBits", "worldYBits", "worldZBits", "effectTotal0", "parentTotal0", "effectTotal1", "parentTotal1", "effectTotal2", "parentTotal2", "effectTotal3", "parentTotal3", "effectTotal4", "parentTotal4", "effectTotal5", "parentTotal5", "effectTotal6", "parentTotal6", "effectTotal7", "parentTotal7", "effectTotal8", "parentTotal8", "effectTotal9", "parentTotal9", "effectTotal10", "parentTotal10", "effectTotal11", "parentTotal11", "effectTotal12", "parentTotal12", "effectTotal13", "parentTotal13", "effectTotal14", "parentTotal14", "effectTotal15", "parentTotal15", "effectTotal16", "parentTotal16", "effectTotal17", "parentTotal17", "effectTotal18", "parentTotal18", "effectTotal19", "parentTotal19", "effectTotal20", "parentTotal20", "effectTotal21", "parentTotal21"}, Out: []string{"completed", "activated", "controllerActive", "controllerVisible", "yBits", "parentTotal0", "parentTotal1", "parentTotal2", "parentTotal3", "parentTotal4", "parentTotal5", "parentTotal6", "parentTotal7", "parentTotal8", "parentTotal9", "parentTotal10", "parentTotal11", "parentTotal12", "parentTotal13", "parentTotal14", "parentTotal15", "parentTotal16", "parentTotal17", "parentTotal18", "parentTotal19", "parentTotal20", "parentTotal21"}, F: spellEffectActivationNode}
}
