package calc

// spellEffectCompletionNode replays the spell-effect completion gate: three
// ordered checks (controller completion enable, last animation frame, effect
// activated) set the completion byte; spell 35 additionally dispatches a
// notification unless combat suppresses it; a per-spell definition flag then
// merges 22 u32 statistics and transfers the auxiliary result list into the
// containing parent effect. The notification and the not-completed advance
// branch leave every listed output unchanged in the differential fixture, so
// only the completion byte, the parent totals and the two result lists are
// outputs. The frame gate compares the current frame byte against the total
// frame count minus one as 32-bit integers (no wraparound: frame 255 with
// total 0 does not complete). No RNG calls.
// Sources: exe 0x4a4050 (gate sequence), 0x4a4640 query 2 with the frame
// total accessor 0x4a2710 (frame compare), 0x4a49e0 (notification), spell
// definition table with 485-byte records and merge flag +0x154, 0x500460
// (parent search, 22-word add, list transfer), 0x4a9260 and 0x4a6e20
// (not-completed branch). Fixture scope: one containing parent whose result
// capacity is 4, zero-initialized notification effect 131, linked-effect
// kind 1, no loaded terrain. No executable table is embedded.
func spellEffectCompletionNode(in []int64, r Rand) []int64 {
	if len(in) != 70 {
		panic("spell completion: expected 70 inputs")
	}
	// Arena base from notes/re/emu.md; the emulator places the spell object
	// there, and the parent search compares raw child pointers against it.
	const arenaBase = 0x20000000
	out := make([]int64, 33)
	out[0] = in[4]
	for k := 0; k < 22; k++ {
		out[1+k] = in[27+2*k]
	}
	out[23], out[24] = in[16], in[17]
	copy(out[25:29], in[18:22])
	copy(out[29:33], in[22:26])
	enabled := uint8(in[0]) != 0   // controller +0x6c, exe 0x4a4059
	finished := in[1] == in[2]-1   // frame == total-1, exe 0x4a4064
	activated := uint8(in[3]) != 0 // effect +0x1e1, exe 0x4a4071
	if !(enabled && finished && activated) {
		return out
	}
	out[0] = 1 // effect +0x1e0, exe 0x4a4082
	// Spell 0x23 notification (exe 0x4a407b..0x4a409f) runs unless combat
	// state is present in mode 2 (exe 0x4a408b..0x4a409b); it changes no
	// listed output in this fixture.
	if uint8(in[8]) == 0 { // merge flag +0x154, exe 0x4a40bb
		return out
	}
	if in[14] <= 0 { // parent count, exe 0x50046c
		return out
	}
	if in[15] != arenaBase { // containing-parent search, exe 0x500486
		return out
	}
	for k := 0; k < 22; k++ { // effect +0x280 into parent +0x126, exe 0x5004c8
		out[1+k] = int64(uint32(in[27+2*k]) + uint32(in[26+2*k]))
	}
	eff := append([]int64(nil), in[18:22]...)
	par := append([]int64(nil), in[22:26]...)
	ec, pc := in[16], in[17]
	for ec > 0 { // list transfer, exe 0x500508
		first := eff[0]
		for j := int64(0); j < ec-1; j++ {
			eff[j] = eff[j+1]
		}
		ec--
		if pc < 4 { // capacity 4; overflow goes to the heap, arena copy stays
			par[pc] = first
		}
		pc++
	}
	out[23], out[24] = ec, pc
	copy(out[25:29], eff)
	copy(out[29:33], par)
	return out
}

func init() {
	Nodes["004a4050"] = Node{Addr: "004a4050", In: []string{"completionEnabled", "frame", "frameCount", "activated", "completed", "spellID", "combatState", "combatMode", "aggregateOnCompletion", "linkedEffect", "casterSlot", "worldXBits", "worldYBits", "worldZBits", "parentCount", "childEffect", "effectResultCount", "parentResultCount", "effectResult0", "effectResult1", "effectResult2", "effectResult3", "parentResult0", "parentResult1", "parentResult2", "parentResult3", "effectTotal0", "parentTotal0", "effectTotal1", "parentTotal1", "effectTotal2", "parentTotal2", "effectTotal3", "parentTotal3", "effectTotal4", "parentTotal4", "effectTotal5", "parentTotal5", "effectTotal6", "parentTotal6", "effectTotal7", "parentTotal7", "effectTotal8", "parentTotal8", "effectTotal9", "parentTotal9", "effectTotal10", "parentTotal10", "effectTotal11", "parentTotal11", "effectTotal12", "parentTotal12", "effectTotal13", "parentTotal13", "effectTotal14", "parentTotal14", "effectTotal15", "parentTotal15", "effectTotal16", "parentTotal16", "effectTotal17", "parentTotal17", "effectTotal18", "parentTotal18", "effectTotal19", "parentTotal19", "effectTotal20", "parentTotal20", "effectTotal21", "parentTotal21"}, Out: []string{"completed", "parentTotal0", "parentTotal1", "parentTotal2", "parentTotal3", "parentTotal4", "parentTotal5", "parentTotal6", "parentTotal7", "parentTotal8", "parentTotal9", "parentTotal10", "parentTotal11", "parentTotal12", "parentTotal13", "parentTotal14", "parentTotal15", "parentTotal16", "parentTotal17", "parentTotal18", "parentTotal19", "parentTotal20", "parentTotal21", "effectResultCount", "parentResultCount", "effectResult0", "effectResult1", "effectResult2", "effectResult3", "parentResult0", "parentResult1", "parentResult2", "parentResult3"}, F: spellEffectCompletionNode}
}
