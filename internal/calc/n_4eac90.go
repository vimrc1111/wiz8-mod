package calc

func init() {
	Nodes["004eac90"] = Node{
		Addr: "004eac90",
		In: []string{"characterSlot", "minimumPhase", "partyActionMode", "phase", "mainRemaining", "offRemaining",
			"currentHand", "intervalBase", "mainSpeed", "offSpeed",
			"interfaceMode", "battle", "battleAction", "battleSpecial", "selectedCharacter", "combat",
			"specialUsesSpell", "specialUsesItem", "mainAction", "offAction", "mainSpell", "offSpell",
			"itemInstance", "itemIndex", "itemSpell", "itemType", "spellTarget", "spellRange", "targetKind",
			"targetCharacter", "targetMonster", "targetGroup", "targetPresent", "targetHP", "targetStatus",
			"targetStamina", "targetFatigue",
			"mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable",
			"mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range"},
		Out: []string{"phase", "currentHand"},
		F:   characterRepeatAttackPhaseNode,
	}
}

// characterRepeatAttackPhaseNode picks the hand that still has attacks left and
// computes the character's next combat phase (exe 0x4eac90..0x4eae16). Inputs
// 10..36 are the 0x536f60 inputs after slot and context; 37..46 are the
// 0x53d310 inputs after slot. Legal states only: hand 0/1, minimumPhase 1..100
// (0x4f0480 asserts on a final phase outside 1..100, as the callee node does),
// and a remaining count below 0xffffffff (the +1 divisor must not wrap to 0).
func characterRepeatAttackPhaseNode(in []int64, r Rand) []int64 {
	if len(in) != 47 {
		panic("characterRepeatAttackPhase: expected 47 inputs")
	}
	u32 := func(v int64) uint32 { return uint32(v) }
	slot, minimum, mode := in[0], u32(in[1]), in[2]
	phase, hand := u32(in[3]), in[6]
	counts := [2]uint32{u32(in[4]), u32(in[5])}
	speeds := [2]int32{int32(in[8]), int32(in[9])}
	if hand != 0 && hand != 1 {
		panic("characterRepeatAttackPhase: current hand must be 0 or 1")
	}
	zero := []int64{0, hand} // 0x4eae0c: only the phase is cleared
	// 0x4eacbc..0x4eacc7: main action 0 or 1, then 0x53d310, then 0x536f60 (context 0, hint 2).
	if in[18] != 0 && in[18] != 1 {
		return zero
	}
	if anyHandAttackAvailableNode(append([]int64{slot}, in[37:47]...), r)[0] == 0 {
		return zero
	}
	// 0x536f60 gets source context 0 and reach hint 2 (any hand). A non-self character
	// target takes its reach from 0x5199f0, i.e. the max over enabled hands (0x519ac0).
	// 0x53d310 above already guarantees an enabled hand with a reach other than -1, so
	// the fixed short reach inside characterCurrentTargetOkNode stays exact here.
	if characterCurrentTargetOkNode(append([]int64{slot, 0}, in[10:37]...), r)[0] == 0 {
		return zero
	}
	// 0x4eacf3..0x4ead57: the other hand is preferred only when the current hand has
	// more attacks left (unsigned) and a signed-lower speed value.
	cur, other := hand, 1-hand
	sel := other
	if counts[other] == 0 {
		sel = cur
	} else if counts[cur] > counts[other] && speeds[cur] > speeds[other] {
		sel = cur
	}
	if counts[sel] == 0 {
		return zero
	}
	// 0x4ead63..0x4ead94: 32-bit wrapping arithmetic, unsigned division.
	if sel == cur || counts[cur] >= counts[other] {
		d := counts[sel] + 1
		if d == 0 {
			panic("characterRepeatAttackPhase: remaining count 0xffffffff (divide error)")
		}
		phase += (u32(in[7]) - minimum + 100) / d
	} else {
		phase = minimum
	}
	if phase > 100 {
		return zero
	}
	// 0x4ead9d..0x4eade1: a changed hand carries the speed difference over.
	if sel != cur {
		phase += uint32(speeds[cur]) - uint32(speeds[sel])
	}
	// 0x4eadef: 0x5179f0 with lower bound minimumPhase and upper bound 100,
	// upper bound tested first (unsigned).
	if phase > 100 {
		phase = 100
	} else if phase < minimum {
		phase = minimum
	}
	// 0x4eadff: 0x4f0480 rounds by party action mode and asserts 1..100.
	phase = u32(combatPhaseRoundingNode([]int64{mode, int64(phase), int64(minimum)}, r)[0])
	return []int64{int64(phase), sel}
}
