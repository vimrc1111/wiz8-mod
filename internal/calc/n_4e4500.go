package calc

import "fmt"

func init() {
	names := []string{"monsterPointer", "combatActive", "monsterActive", "monsterDisposition", "selectedMonsterPointer", "selectedMonsterFlag",
		"statusBufferPointer", "hostileCountBefore", "charmedCountBefore", "combatOpeningBlockedBefore", "uiPointer", "combatDisplayDirtyBefore", "sceneDisplayDirtyBefore"}
	for m := 1; m <= 2; m++ {
		for _, f := range []string{"Present", "Active", "Disposition", "CharmDuration"} {
			names = append(names, fmt.Sprintf("monster%d%s", m, f))
		}
	}
	outs := []string{"monsterStatusPointer", "monsterActiveAfter", "selectedMonsterFlagAfter", "selectedMonsterPointerAfter", "statusBufferPointerAfter",
		"hostileCount", "charmedCount", "combatOpeningBlocked", "combatDisplayDirty", "sceneDisplayDirty"}
	for k := 0; k < 15; k++ {
		for _, f := range []string{"Active", "Id", "Value5", "ValueD"} {
			names = append(names, fmt.Sprintf("slot%d%s", k, f))
			outs = append(outs, fmt.Sprintf("slot%d%sAfter", k, f))
		}
	}
	Nodes["004e4500"] = Node{Addr: "004e4500", In: names, Out: outs, F: monsterCombatStatusClearNode}
}

// monsterCombatStatusClearNode (exe 0x4e4500): in = the 13 scalar fields, 8 fields of
// monsters 1 and 2, then 15 status slots as (active u8, id u32, value5 u32, valueD u32).
//
// Modelled: slot clearing (0x5524e0 zeroes +0, +1, +5, +0xd of every active slot,
// 0x4e4580..0x4e45c7), the party selection reset (0x4e4565), the status-buffer release
// (0x532330 + 0x5e23e0 always zero [status+0x16] when it is set), the monster's
// status pointer and active flag clears, and the enemy recount tail (0x546e70) when the
// disposition is 1.
// Not modelled (not in the outputs): the per-slot side effects of 0x5524e0 beyond the
// slot bytes — 0x50e8c0's recomputation of the monster's derived stats, and 0x4acd80's
// removal of scene effects, which needs a non-empty effect list; the case keeps that list empty.
// A slot id must index a non-empty row of the status table at 0x60cffc, otherwise the
// original stops at an assert (Spells.cpp:2095); the node does not validate it.
func monsterCombatStatusClearNode(in []int64, r Rand) []int64 {
	if len(in) != 21+60 {
		panic("monster combat status clear: expected 81 inputs")
	}
	for i, v := range in {
		limit := int64(0xffffffff)
		if i == 1 || i == 2 || i == 3 || i == 9 || i == 11 || i == 12 || i >= 13 && i < 21 && (i-13)%4 != 3 || i >= 21 && (i-21)%4 == 0 {
			limit = 255
		}
		if v < 0 || v > limit {
			panic("monster combat status clear: field width overflow")
		}
	}
	// 0x4e4500 asserts combat is active (0x683f94 non-zero) and the monster is active (+0x15 non-zero).
	if in[1] == 0 || in[2] == 0 {
		panic("monster combat status clear: assert condition")
	}
	selPtr, selFlag := in[4], in[5]
	if selPtr == in[0] { // 0x4e4565: G+0x7b8 == this monster clears G+0x7b0 and G+0x7b8.
		selPtr, selFlag = 0, 0
	}
	out := []int64{0, 0, selFlag, selPtr, 0, in[7], in[8], in[9], in[11], in[12]}
	for k := 0; k < 15; k++ {
		s := in[21+4*k : 25+4*k]
		if s[0] != 0 { // 0x5524e0 clears an active slot's +0, +1, +5 and +0xd.
			s = []int64{0, 0, 0, 0}
		}
		out = append(out, s...)
	}
	if in[3] != 1 { // 0x4e45df: only disposition 1 tail-calls the recount at 0x546e70.
		return out
	}
	rec := []int64{1, 0, in[3], 0} // this monster: present, active cleared at 0x4e45e5, disposition, charm
	rec = append(rec, in[13:21]...)
	for len(rec) < 32 {
		rec = append(rec, 0)
	}
	cnt := hostileMonsterCountNode(append([]int64{1, 3, in[1], in[9], in[10], in[11], in[12], in[7], in[8]}, rec...), r)
	copy(out[5:10], cnt)
	return out
}
