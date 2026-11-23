package calc

import "fmt"

func init() {
	in := []string{"partyActionMode", "partyThreshold", "partyActionPhase"}
	out := []string{"partyActionPhase"}
	for i := 0; i < 8; i++ {
		in = append(in, fmt.Sprintf("slot%dAction", i), fmt.Sprintf("slot%dDone", i))
		out = append(out, fmt.Sprintf("slot%dAction", i))
	}
	in = append(in, "monsterListPointer", "monsterCount")
	for i := 0; i < 8; i++ {
		in = append(in, fmt.Sprintf("monster%dActive", i), fmt.Sprintf("monster%dAction", i), fmt.Sprintf("monster%dDone", i))
		out = append(out, fmt.Sprintf("monster%dAction", i))
	}
	Nodes["004f0c80"] = Node{Addr: "004f0c80", In: in, Out: out, F: combatActorPhaseRoundingNode}
}

// combatActorPhaseRoundingNode: in = mode, threshold, party phase, 8 x (action, done byte),
// monster list pointer, monster count, 8 x (active byte, action, done byte).
// Only the first monsterCount monsters of the 8 fixture records exist.
func combatActorPhaseRoundingNode(in []int64, r Rand) []int64 {
	if len(in) != 3+16+2+24 {
		panic("combatActorPhaseRounding: wrong input count")
	}
	for _, v := range in {
		if v < 0 || v > 4294967295 {
			panic("combatActorPhaseRounding: expected unsigned inputs")
		}
	}
	mode, threshold := in[0] != 0, uint32(in[1])
	// exe 0x4f0c9c..0x4f0cc6 / 0x4f0d20..0x4f0d4a / 0x4f0ddc..0x4f0e06: with a non-zero
	// mode, add 5 (32-bit wrap), drop the remainder mod 10, ceil the threshold to a
	// multiple of 10 (32-bit wrap) and call 0x5179f0 with cap 100: cap tested first.
	round := func(v uint32) uint32 {
		if mode {
			v += 5
			v -= v % 10
			lower := threshold
			if rem := lower % 10; rem != 0 {
				lower += 10 - rem
			}
			if v > 100 {
				v = 100
			} else if v < lower {
				v = lower
			}
		}
		// exe 0x4f0cd3, 0x4f0d57, 0x4f0e0e: srAssertFail unless 1..100, in every mode.
		if v == 0 || v > 100 {
			panic("combatActorPhaseRounding: result outside 1..100 (srAssertFail)")
		}
		return v
	}
	out := []int64{int64(round(uint32(in[2])))}
	// Party slots (exe 0x4f0cff..0x4f0d8d): stride 0xd4, skipped when action < threshold or done byte != 0.
	for i := 0; i < 8; i++ {
		a, done := uint32(in[3+2*i]), in[4+2*i]
		if a >= threshold && done == 0 {
			a = round(a)
		}
		out = append(out, int64(a))
	}
	// Monsters (exe 0x4f0d93..0x4f0e45): null list pointer counts as 0 (0x5e2c70); active byte at +0x15.
	n := in[20]
	if in[19] == 0 {
		n = 0
	}
	if n > 8 {
		panic("combatActorPhaseRounding: only 8 monster fixture records")
	}
	for i := 0; i < 8; i++ {
		active, a, done := in[21+3*i], uint32(in[22+3*i]), in[23+3*i]
		if int64(i) < n && active != 0 && a >= threshold && done == 0 {
			a = round(a)
		}
		out = append(out, int64(a))
	}
	return out
}
