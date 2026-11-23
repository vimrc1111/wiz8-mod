package calc

func init() {
	Nodes["005a1150"] = Node{
		Addr: "005a1150",
		In: []string{"selectedSlot", "previousSlot", "slotSpell", "slotState", "activeCharacter",
			"sp0", "sp1", "sp2", "sp3", "sp4", "sp5",
			"spellPowerClass", "spellBaseCost", "spellRealm", "spellSchoolFlags", "spellKind"},
		Out: []string{"selectedSlot", "panelSpell", "panelPowerClass", "panelMaxPower"},
		F:   spellAffordablePowerNode,
	}
}

// spellAffordablePowerNode selects a spell slot on the spell panel. It returns
// the panel's selected slot (+0x4f8), spell (+0x100), power class (+0x10c) and
// maximum affordable power (+0x110). Panel fields it does not write read as 0,
// the arena's initial value. spellSchoolFlags and spellKind only steer UI
// children and do not change the four outputs.
func spellAffordablePowerNode(in []int64, r Rand) []int64 {
	if len(in) != 16 {
		panic("spellAffordablePower: expected 16 inputs")
	}
	slot, prev, spell, state, char := in[0], in[1], in[2], in[3], in[4]
	class, cost, realm := in[11], in[12], in[13]
	// exe 0x5a1150: slot -1 returns before any panel write; 0x4f8 keeps its old value.
	if slot < -1 || slot > 31 {
		panic("spellAffordablePower: slot outside 0x580 slot table")
	}
	if slot == -1 {
		return []int64{prev, 0, 0, 0}
	}
	// exe 0x5a1182 stores the new slot into +0x4f8; an empty slot stops there.
	if spell == 0 {
		return []int64{slot, 0, 0, 0}
	}
	// exe 0x5a12c4: state byte 0 or 4 deselects (+0x4f8 = -1) and runs UI children.
	if state == 0 || state == 4 {
		return []int64{-1, 0, 0, 0}
	}
	// exe 0x5a11b4 asserts a non-negative power class (Magic line 2100); none returns.
	if class < 0 {
		panic("spellAffordablePower: power class assertion")
	}
	if char < 0 || char > 5 || realm < 0 || realm > 5 {
		panic("spellAffordablePower: character or realm outside table")
	}
	if cost == 0 {
		panic("spellAffordablePower: zero base cost divides by zero")
	}
	// exe 0x5a11fd reads the active character's spell power for the realm,
	// negative values count as 0 (signed 32-bit inputs).
	sp := in[5+realm]
	if sp < 0 {
		sp = 0
	}
	// signed idiv truncates toward zero, same as Go.
	v := sp / cost
	// exe 0x5a1261 via 0x5179f0: unsigned clamp to 7, so negative quotients also give 7.
	if uint32(v) > 7 {
		v = 7
	}
	if v == 0 {
		return []int64{-1, 0, 0, 0}
	}
	return []int64{slot, spell, class, v}
}
