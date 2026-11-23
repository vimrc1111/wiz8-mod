package calc

// Node 0x501400: highest affordable power level of the queued spell.
// Character 0 is fixed; all selected records are explicit inputs. Callees:
// 0x5248a0 reads one byte of the character record, 0x5001e0 is the spell
// usability-by-context test below, 0x537220 is the target test (the same
// composition as 0x501530 with context 3 for the target class).
func queuedSpellAffordablePowerNode(in []int64, r Rand) []int64 {
	if len(in) != 54 {
		panic("queued spell affordable power: expected 54 inputs")
	}
	spell, level := in[0], in[1]
	if spell < 0 || spell > 149 || level < 0 || level > 8 || in[5] < 0 || in[5] > 7 ||
		in[8] < 0 || in[8] > 4 {
		panic("queued spell affordable power: record selector outside fixture")
	}
	if spell == 0 {
		return []int64{0} // 0x501428.
	}
	if spell == 0x3c && in[3] != 0 { // 0x5248a0 byte, 0x501433.
		return []int64{0}
	}
	if in[2] != 1 { // Learned flag, 0x501466.
		return []int64{0}
	}
	combat := in[9] != 0
	if level == 8 && in[4] == 1 && combat { // 0x501474..0x5014a0.
		return []int64{0}
	}
	// 0x5001e0 with its second argument zero: ctx is spell +0x13b.
	usable := func() bool {
		ctx, ui := in[8], in[13]
		bUi, cm, cx := in[10] != 0, in[11] != 0, in[12] != 0
		if ui == 6 && !combat && bUi && (spell == 0x17 || spell == 0x3a) {
			return true // 0x50024f..0x500268.
		}
		anyF := cm || cx
		switch ctx {
		case 0:
			return !bUi && !anyF
		case 1:
			return in[9] == 1 && !bUi && !anyF // Byte flag must be exactly 1.
		case 2:
			return !combat && !bUi && !anyF
		case 3:
			return bUi && !anyF
		case 4:
			if spell == 0x27 {
				return cm || cx
			}
			return spell == 0x12 && cx
		}
		return false
	}
	if !usable() {
		return []int64{0}
	}
	if !combat {
		// 0x537220: class from 0x5010f0 with context 3 never depends on
		// the battle object, so force it absent in the 0x501530 composition.
		t := append([]int64{spell}, in[14:]...)
		t[24] = 0
		if queuedSpellTargetEligibilityNode(t, r)[0] == 0 {
			return []int64{0}
		}
	}
	if level == 8 {
		level = 1 // 0x5014d7.
	}
	if level == 0 {
		return []int64{0}
	}
	cost, sp := int32(in[6]), int32(in[7])
	need := int32(int32(level) * cost) // 32-bit imul, 0x501518.
	for ; level != 0; level-- {
		if sp >= need {
			return []int64{level}
		}
		need -= cost
	}
	return []int64{0}
}

func init() {
	Nodes["00501400"] = Node{
		Addr: "00501400",
		In: []string{"spellId", "powerLevel", "learned", "specialFlag", "spellFlag12b", "school", "costPerPL", "domainSP", "useContext", "combatFlag", "flag683f9b", "flag683f9c", "flag683f9d", "uiState",
			"spellTargetType", "spellRange", "targetKind", "characterTarget", "monsterTarget", "groupTarget", "characterPresent", "characterHP", "characterCondition", "characterRecoverable", "characterRecoveryBlocked", "monsterHP", "monsterRemoved", "monsterPresent", "monsterVisible", "monsterReachable", "groupActive", "targetX", "partyX", "casterFormation", "targetFormation", "frontCharacter", "characterFormationDisabled", "battlePresent", "casterPresent", "casterHP", "casterCondition", "monsterX", "targetY", "targetZ", "partyY", "partyZ", "monsterY", "monsterZ", "monsterRadius", "partyRadius", "coordinatePartyRadius", "groupMemberCount", "groupSize", "targetListItem"},
		Out: []string{"power"}, F: queuedSpellAffordablePowerNode,
	}
}
