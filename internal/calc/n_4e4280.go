package calc

// This composition uses one listed monster, no enabled party members, no
// queued actions, and animation records without attached scene effects.
// Arena pointers describe case wiring; they are not game constants or tables.
// Child functions run normally in the emulator. No executable data table is
// read on these paths. Position inputs and outputs carry float32 bits.
func init() {
	Nodes["004e4280"] = Node{Addr: "004e4280", In: []string{"enabled", "combatActive", "monsterId", "HP", "SP", "actionTime", "condition", "positionX", "positionY", "positionZ", "scenePositionX", "scenePositionY", "scenePositionZ", "sceneEnabled", "sceneFlags", "livingCount", "currentAction", "currentTarget", "effectCount", "animationCount", "sceneTargeted", "previousHostileCount", "previousCharmedCount", "combatOpeningBlocked", "uiPointer", "combatDisplayDirty", "sceneDisplayDirty", "mode", "uiCombatDisplayEnabled", "monsterInCombat", "monsterDisposition", "charmDuration"}, Out: []string{"enabled", "HP", "SP", "actionTime", "condition", "positionX", "positionY", "positionZ", "sceneEnabled", "sceneFlags", "livingCount", "currentAction", "currentTarget", "effectListPointer", "animationCount", "sceneTargeted", "previousHostileCount", "previousCharmedCount", "combatOpeningBlocked", "combatDisplayDirty", "sceneDisplayDirty", "animationCountAfter", "effectStoragePointer"}, F: monsterRemoveStateNode}
}

func monsterRemoveStateNode(in []int64, r Rand) []int64 {
	if len(in) != 32 {
		panic("monster remove state: expected 32 inputs")
	}
	for i, v := range in {
		limit := int64(0xffffffff)
		switch i {
		case 0, 1, 13, 20, 23, 25, 26, 28, 29, 30:
			limit = 255
		}
		if v < 0 || v > limit {
			panic("monster remove state: field width overflow")
		}
	}
	if in[18] > 4 || in[19] > 4 {
		panic("monster remove state: list exceeds four arena records")
	}
	if in[28] != 0 {
		panic("monster remove state: UI combat display must be disabled in this scene")
	}
	if in[24] != 0 && in[24] != 0x20005000 {
		panic("monster remove state: invalid arena UI pointer")
	}
	out := []int64{in[0], in[3], in[4], in[5], in[6], in[7], in[8], in[9], in[13], in[14], in[15], in[16], in[17], 0x20004080, in[19], in[20], in[21], in[22], in[23], in[25], in[26], in[19], 0x200040a0}
	if in[0] == 0 {
		return out
	} // 0x4e42ab: inactive records are unchanged.
	// 0x4e42d1..0x4e42f4: unset action time (9999), removed condition (18),
	// zero HP/SP, disable scene object, replace scene flags with 0x200000.
	out[0], out[1], out[2], out[3], out[4], out[8], out[9] = 0, 0, 0, 9999, 18, 0, 0x200000
	// 0x4acf90 clears animation list count; 0x505c80 destroys the attack/effect
	// list and clears its owner pointer. 0x5e23e0 clears its storage pointer.
	out[13], out[14], out[21], out[22] = 0, 0, 0, 0
	// 0x4c5750 / 0x4b88b0 copy the scene position, preserving finite bits.
	for i := 0; i < 3; i++ {
		bits := uint32(in[10+i])
		if bits&0x7f800000 == 0x7f800000 && bits&0x007fffff != 0 {
			bits |= 0x00400000
		}
		out[5+i] = int64(bits)
	}
	out[10] = int64(uint32(in[15]) - 1) // 0x4e433e: u32 living count wraps.
	if in[1] == 0 {
		return out
	}
	// 0x538d60 / 0x4c5840 clear scene targeting. The removed member is
	// already disabled, so 0x505810 does not update its visibility, and
	// 0x546e70 recounts zero hostile/charmed members. Share its UI behavior.
	out[15] = 0
	counts := make([]int64, 41)
	counts[0], counts[1], counts[2], counts[3], counts[4], counts[5], counts[6] = 1, 1, in[1], in[23], in[24], in[25], in[26]
	counts[10], counts[11], counts[12] = in[29], in[30], in[31]
	counted := hostileMonsterCountNode(counts, r)
	out[16], out[17], out[18], out[19], out[20] = counted[0], counted[1], counted[2], counted[3], counted[4]
	if in[17] == 0x20000000 {
		out[11], out[12] = 0, 0
	} // 0x4e4369..0x4e437c.
	return out
}
