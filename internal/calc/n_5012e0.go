package calc

func init() {
	Nodes["005012e0"] = Node{Addr: "005012e0", In: []string{
		"spellId", "spellTargetType", "spellRange", "targetKind", "characterTarget", "monsterTarget", "groupTarget",
		"characterPresent", "characterHP", "characterCondition", "characterRecoverable", "characterRecoveryBlocked",
		"monsterHP", "monsterRemoved", "monsterPresent", "monsterVisible", "monsterReachable", "groupActive",
		"targetX", "partyX", "casterFormation", "targetFormation", "frontCharacter", "characterFormationDisabled",
		"casterPresent", "casterHP", "casterCondition", "monsterX", "targetY", "targetZ", "partyY", "partyZ",
		"monsterY", "monsterZ", "monsterRadius", "partyRadius", "coordinatePartyRadius", "groupMemberCount",
		"groupSize", "targetListItem",
		"slotMode", "learned", "realmPoints", "costPerLevel", "realm", "heldCondition", "context",
		"modeFlag", "flag9b", "flag9c", "flag9d", "mode6", "restrictByte",
	}, Out: []string{"usable"}, F: queuedSpellUsableNode}
}

// queuedSpellUsableNode (exe 0x5012e0) for character slot 0 (the stack argument; the fixture is caster slot 0).
// Inputs 0..39 are the target-eligibility inputs of queuedSpellTargetEligibilityNode (0x501530 chain: spell id
// Q+0x75, spell target type F+0x137, range F+0x12f, queued target Q+0x81.., world/party/monster/group fields),
// in the same order without its battlePresent input. 0x537220 runs that same chain (0x5010f0, 0x53b7f0,
// 0x537160, 0x519180 with context 3 instead of 6), so it is called directly with battlePresent forced to 0:
// 0x5010f0 category 9 then yields 6 whichever way 0x68edcc is set.
// Inputs 40..52: queue slot mode (Q+0x79), learned flag (character +0xbcd+4*spell), realm points (+0xb45+4*realm),
// spell cost per level (F+0x49), realm (F+0x133), held-item condition (F+0x12b), usage context (F+0x13b),
// bytes 0x683f94 / 0x683f9b / 0x683f9c / 0x683f9d, dword 0x68ec78, character byte +0x181f (read by 0x5248a0).
func queuedSpellUsableNode(in []int64, r Rand) []int64 {
	if len(in) != 53 {
		panic("queued spell usable: expected 53 inputs")
	}
	spell, cat := in[0], in[1]
	slot, learned, points, perLevel := uint32(in[40]), uint32(in[41]), int32(in[42]), int32(in[43])
	held, ctx := uint32(in[45]), in[46]
	mode, f9b, f9c, f9d, mode6, restrict := byte(in[47]), byte(in[48]), byte(in[49]), byte(in[50]), uint32(in[51]), byte(in[52])
	if spell < 0 || spell >= 0x96 || ctx < 0 || ctx > 4 || cat < 0 || cat > 10 || in[44] < 0 || in[44] > 7 {
		panic("queued spell usable: input outside the modeled domain")
	}
	result := func(ok bool) []int64 {
		if ok {
			return []int64{1}
		}
		return []int64{0}
	}

	// 0x5001e0 with its second argument 0: spell usable in the current context.
	spellContextOK := func() bool {
		combo := f9c != 0 || f9d != 0
		if mode6 == 6 && mode == 0 && f9b != 0 && (spell == 0x17 || spell == 0x3a) {
			return true
		}
		switch ctx {
		case 0:
			return f9b == 0 && !combo
		case 1:
			return mode == 1 && f9b == 0 && !combo
		case 2:
			return mode == 0 && f9b == 0 && !combo
		case 3:
			return f9b != 0 && !combo
		}
		if spell == 0x27 {
			return combo
		}
		return spell == 0x12 && f9d != 0
	}

	if spell == 0 {
		return result(false)
	}
	if spell == 0x3c && restrict != 0 { // 0x5248a0(who, 0)
		return result(false)
	}
	if learned != 1 {
		return result(false)
	}
	if slot == 8 && held == 1 && mode != 0 {
		return result(false)
	}
	mult := slot
	if mult == 8 {
		mult = 1
	}
	if points < int32(uint32(perLevel)*mult) {
		return result(false)
	}
	if !spellContextOK() {
		return result(false)
	}
	if mode != 0 {
		return result(true)
	}
	target := make([]int64, 0, 41)
	target = append(target, in[:24]...)
	target = append(target, 0) // battlePresent
	target = append(target, in[24:40]...)
	return queuedSpellTargetEligibilityNode(target, r) // 0x537220(who, spell, 3)
}
