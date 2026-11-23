package calc

// 0x50baf0 pickpocket_chance(C, npc, itemId, quantity): the steal threshold
// clamped to 0..127 (build/wiz8.asm:361364, notes/re/abilities.md:109).
//
//	base   = C+0x391 + (class 7: C+0x139/10, else when C+0x391 > 0: C+0x139/20)
//	base  += (attitude-50)/4 (signed, toward zero)          attitude = 0x50a280(npc)
//	base  -= (R(5)+10) * npc+0x84 (signed byte)
//	base  += 5 * (C+0x89 - monsterTemplate+0x24f)           monster of npc+0x16
//	pen    = C+0x391>>2 (logical)
//	pen   -= weight*quantity/10 and value/100 when itemId != -1
//	                                                        weight = item+0x8a (u16),
//	                                                        value  = 0x51b840(temp item)
//	base  += min(pen, 0); base = 0x55cc00 difficulty(base); clamp 0..127
//
// All arithmetic is 32-bit (wraps); the divisions are unsigned except the
// attitude one. Callees: 0x4e5550/0x4e5620/0x4e5720 (monster lookup),
// 0x517ec0 (roster slot) run natively in the case and are modelled by the
// fields above. 0x50a280, 0x51c020 and 0x51b840 (D4 formulas) are address
// stubs in the case: the attitude and the item value are node inputs and the
// stubbed call counts and the item id given to 0x51c020 are outputs (MG
// 2026-10-09 composition ruling). The real Go callees are not composed: the
// temp item of 0x51c020 is a stack address that its Go model cannot hold.
// 0x55cc00 mutates the value it is given, which a stub cannot express from a
// case input, so it runs natively and difficultyNode is reused.
//
// Fixture caps (calc panics past them): item ids 0..15, monster id other than
// the fixture's decoy id 0xffffffff, character in roster slot 0 (the slot only
// selects C+0xa35, which is C's own field). When the NPC has no linked monster
// (npc+0x1a or +0x25 zero) the original passes NULL to 0x4e5720, which asserts
// and then reads address 8: no returning behaviour, calc panics.
func pickpocketChanceNode(in []int64, r Rand) []int64 {
	const (
		iClass = iota
		iSteal
		iDex
		iLevel
		iAttempts
		iLinked
		iAvailable
		iMonster
		iMonsterLevel
		iWeight
		iReverse
		iDifficulty
		iAttitude
		iValue
		iItem
		iQuantity
		nIn
	)
	if len(in) != nIn {
		panic("pickpocket_chance: wrong input count")
	}
	ranges := [nIn][2]int64{{-2147483648, 2147483647}, {-2147483648, 2147483647}, {0, 4294967295}, {0, 4294967295},
		{-128, 127}, {1, 255}, {1, 255}, {0, 4294967294}, {0, 255}, {0, 65535}, {-2147483648, 2147483647},
		{-2147483648, 2147483647}, {0, 255}, {0, 4294967295}, {-1, 15}, {0, 4294967295}}
	for i, v := range in {
		if v < ranges[i][0] || v > ranges[i][1] {
			panic("pickpocket_chance: input outside the fixture range")
		}
	}
	if in[iLinked] == 0 || in[iAvailable] == 0 {
		panic("pickpocket_chance: NPC without linked monster asserts in 0x4e5720(NULL)")
	}
	steal, dex := int32(in[iSteal]), uint32(in[iDex])
	base := uint32(steal)
	if in[iClass] == 7 {
		base += dex / 10
	} else if steal > 0 {
		base += dex / 20
	}
	base += uint32((int32(int8(in[iAttitude])) - 50) / 4)
	base -= uint32((int32(r.R(5)) + 10) * int32(in[iAttempts]))
	base += 5 * (uint32(in[iLevel]) - uint32(in[iMonsterLevel]))
	pen := int32(uint32(steal) >> 2)
	var initCalls, initItem, valueCalls int64
	if in[iItem] != -1 {
		initCalls, initItem, valueCalls = 1, in[iItem], 1
		pen -= int32((uint32(in[iWeight]) * uint32(in[iQuantity])) / 10)
		pen -= int32(uint32(in[iValue]) / 100)
	}
	if pen > 0 {
		pen = 0
	}
	base += uint32(pen)
	// 0x55cc00(slot of C, &base): slot 0 of the roster, so only C+0xa35 and the
	// difficulty matter.
	scaled := difficultyNode([]int64{0, 0, 0, 0, 0, 0, 0, 0, int64(int32(base)), in[iDifficulty], in[iReverse], 0}, r)[0]
	if scaled < 0 {
		scaled = 0
	}
	if scaled > 127 {
		scaled = 127
	}
	return []int64{scaled, 1, initCalls, initItem, valueCalls}
}

func init() {
	Nodes["0050baf0"] = Node{Addr: "0050baf0", In: []string{
		"characterClass", "stealBaseRating", "effectiveDexterity", "characterTotalLevel", "npcStealAttempts",
		"monsterLinkEnabled", "linkedMonsterAvailable", "linkedMonsterIdentity", "monsterTemplateLevel",
		"unitWeight", "reverseDifficultyState", "difficulty", "attitudeRet", "itemValueRet", "itemId", "quantity",
	}, Out: []string{"chance", "attitudeCalls", "itemInitCalls", "itemInitId", "itemValueCalls"}, F: pickpocketChanceNode}
}
