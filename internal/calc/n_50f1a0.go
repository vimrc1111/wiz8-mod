package calc

// monsterGroupSpawnNode reproduces 0x50f1a0 (monster group creation) as replayed
// by re/emu/cases/0050f1a0.json. Inputs follow Node.In below.
// The RNG calls of the original are replayed in order: 0x4e3930 for every member
// (HP dice, SP dice), then per member the model clone (0x4c1d20: one roll of 100
// and three of 1000) and, outside the special list, the idle frame (0x4e3e09).
func monsterGroupSpawnNode(in []int64, r Rand) []int64 {
	if len(in) != 28 {
		panic("monster group spawn: expected 28 inputs")
	}
	const (
		heapBase = 0x50000000 // replay heap base (re/emu/emu.py), not a game address
		// 0x50f1ad allocates 299 bytes (0x12b); the emulator aligns to 8 bytes, so
		// the 0x5e2900 member list header follows at +304.
		memberList = heapBase + 304
	)
	nextGroup, nextMonster := uint32(in[21]), uint32(in[7])
	// 0x50f1f4: 0x4e57c0 returns NULL when the template slot is empty and the
	// gameplay database file cannot be opened; the case only populates slot 0.
	if in[24] != 0 {
		return []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, int64(nextGroup), int64(nextMonster), 0, 0}
	}
	// 0x50f1a0..0x50f1b1: the requested count is clamped to nine members.
	count := uint32(in[16])
	if count > 9 {
		count = 9
	}
	if count == 0 {
		panic("monster group spawn: a group needs at least one member (0x4e5550 asserts)")
	}
	// 0x50f204..0x50f21c: reserve a nonzero group id, wrapping the counter.
	id := nextGroup
	nextGroup++
	if id == 0 {
		id = nextGroup
		nextGroup++
	}
	// 0x50f2f6 loop: 0x4e3930 rolls HP then SP for each member and reserves a
	// nonzero monster id (0x4e395d..0x4e397c).
	for i := uint32(0); i < count; i++ {
		diceNode(in[0:3], r)
		diceNode(in[3:6], r)
		mid := nextMonster
		nextMonster++
		if mid == 0 {
			nextMonster++
		}
	}
	// 0x50f33f..0x50f388: every member is activated through 0x4e3c70.
	for i := uint32(0); i < count; i++ {
		r.R(100)
		r.R(1000)
		r.R(1000)
		r.R(1000)
		// 0x4e3d21: members of the special list skip the idle-frame roll;
		// 0x4e3dc3 rolls it only when no unborn presentation exists.
		if in[6] == 0 && in[27] == 0 {
			r.R(int(in[26]))
		}
	}
	// 0x50f2a1 stores -1 at +0x2f and 0x50f31e at +0x9b; 0x50f2ad copies the
	// world time; 0x50f297 stores the identified flag from param 4 or template flag 0x10.
	identified := int64(0)
	if in[17]&0xff != 0 || in[20]&0x10 != 0 {
		identified = 1
	}
	// 0x50f2bc / 0x50f2c5 select the special or normal group list by template +0x26a.
	normalList, specialList := int64(1), int64(0)
	if in[6] != 0 {
		normalList, specialList = 0, 1
	}
	return []int64{
		heapBase, int64(id), int64(count), memberList, 0, 0, int64(count), in[24],
		int64(uint32(in[8])), int64(uint32(in[9])), int64(uint32(in[10])),
		1, 0, 3, identified, 0, 0, 0xffffffff, 0xffffffff,
		int64(nextMonster - 1), int64(uint32(in[22])), int64(nextGroup), int64(nextMonster),
		normalList, specialList,
	}
}

func init() {
	Nodes["0050f1a0"] = Node{Addr: "0050f1a0",
		In: []string{
			"hpBase", "hpDice", "hpSides", "spBase", "spDice", "spSides", "specialMonster", "nextMonsterId",
			"positionX", "positionY", "positionZ", "partyX", "partyY", "partyZ", "mode", "battleFlags",
			"requestedCount", "forceIdentified", "report", "placementMode", "templateFlags", "nextGroupId", "worldTime", "reportEnabled",
			"templateId", "nameId", "animationFrames", "unbornCount",
		},
		Out: []string{
			"groupPointer", "groupId", "memberCount", "memberListPointer", "field0C", "field10", "memberTotal", "templateIdCopy",
			"positionXCopy", "positionYCopy", "positionZCopy", "presenceFlag", "field29", "field2B", "identified", "field2D", "field2E", "field2F", "monsterIdMark",
			"lastMonsterId", "worldTimeCopy", "nextGroupIdAfter", "nextMonsterIdAfter", "groupListCount", "specialGroupListCount",
		},
		F: monsterGroupSpawnNode,
	}
}
