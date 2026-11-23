package calc

// 0x50c040 targeted_pickpocket_resolve(C, npc, itemId, quantity): one pickpocket
// attempt on npc by character C, aimed at itemId (build/wiz8.asm:361853).
//
//	slot  = roster slot of C (0x517ec0: C is one of the eight records of *0x685174,
//	        stride 0x1862)
//	lead  = slot + 11*npc+0x2e (u8) + 7*npc+0x84 (signed byte)   32-bit, wraps
//	seed  = sum, over party slots 0..5 whose presence byte (record *0x685178 +0x20c,
//	        stride 0x106) is non-zero, of the u32 at *0x685174 +0x3ab1 + 0x1862*slot
//	srand(seed); then lead rolls R(100) are thrown away
//	chance = 0x50baf0(C, npc, itemId, quantity & 0xff), read as a signed byte
//	npc+0x84 = npc+0x84 + 1 unless it is already >= 100 (signed compare)
//	roll = R(100); roll > 95 or roll > 2*chance: return 2
//	otherwise roll >= chance: return 1; roll < chance: 0x553f10(C, 0xd, 5, 0), return 0
//
// Emu address-stubs the D4 callees 0x50baf0 and 0x553f10 (composition ruling
// 2026-10-09). The chance comes from the case input chanceRet; the recorded
// arguments and call counts are outputs. 0x553f10's results are not read again, so
// F only differs in the chance: F ignores chanceRet and runs the real
// pickpocketChanceNode on the F-only inputs pp* (the character, NPC, monster and
// item fields it reads; npcStealAttempts, itemId and quantity are shared with the
// node's own). That adds one R(5) before the final R(100), which Emu does not
// replay. 0x517ec0 and srand (import) run as described above; srand is stubbed
// in the case, its seed is an output.
//
// Fixture caps (calc panics past them): C must be one of the eight roster
// records (otherwise 0x517ec0 asserts); the number of thrown rolls is at most
// 4000. The original's roll count is a u32, so a negative lead (npc+0x84 negative
// with a small npc+0x2e) would throw about 2^32 rolls; that is past the cap too.
// F's panics: pickpocketChanceNode's own caps (item id -1..15, NPC linked monster).
const (
	tprRoster   = 0x20000000         // case: the roster is the first object of the arena
	tprNPC      = 0x20000000 + 69632 // case: npc object offset
	tprStride   = 0x1862
	tprMaxLead  = 4000
	tprPartyCap = 6
)

const (
	tprInCharAddr = iota
	tprInItem
	tprInQuantity
	tprInAttempts
	tprInTarget
	tprInPresent
	tprInSeed     = tprInPresent + tprPartyCap
	tprInChance   = tprInSeed + tprPartyCap
	tprInFOnly    = tprInChance + 1
	tprInFOnlyLen = 13
	tprInLen      = tprInFOnly + tprInFOnlyLen
)

func init() {
	in := []string{"charAddr", "itemId", "quantity", "npcStealAttempts", "npcTargetByte"}
	for k := 0; k < tprPartyCap; k++ {
		in = append(in, "present"+string(rune('0'+k)))
	}
	for k := 0; k < tprPartyCap; k++ {
		in = append(in, "seedPart"+string(rune('0'+k)))
	}
	in = append(in, "chanceRet", "ppClass", "ppSteal", "ppDex", "ppLevel", "ppLinked", "ppAvailable", "ppMonster",
		"ppMonsterLevel", "ppWeight", "ppReverse", "ppDifficulty", "ppAttitude", "ppValue")
	out := []string{"result", "npcStealAttemptsAfter", "srandCalls", "srandSeed", "pickCalls",
		"pickArg1", "pickArg2", "pickArg3", "pickArg4", "growCalls", "growArg1", "growArg2", "growArg3", "growArg4"}
	Nodes["0050c040"] = Node{Addr: "0050c040", In: in, Out: out,
		F:   func(in []int64, r Rand) []int64 { return targetedPickpocketResolve(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return targetedPickpocketResolve(in, r, false) }}
}

func targetedPickpocketResolveNode(in []int64, r Rand) []int64 {
	return targetedPickpocketResolve(in, r, true)
}

func targetedPickpocketResolve(in []int64, r Rand, real bool) []int64 {
	if len(in) != tprInLen {
		panic("targeted_pickpocket_resolve: wrong input count")
	}
	rel := in[tprInCharAddr] - tprRoster
	if rel < 0 || rel%tprStride != 0 || rel/tprStride > 7 {
		panic("targeted_pickpocket_resolve: C is not a roster record (0x517ec0 asserts)")
	}
	slot := uint32(rel / tprStride)
	attempts := int32(int8(in[tprInAttempts]))
	lead := slot + 11*uint32(in[tprInTarget]&0xff) + 7*uint32(attempts) // exe 0x50c05e..0x50c07c
	if lead > tprMaxLead {
		panic("targeted_pickpocket_resolve: thrown roll count past the fixture cap")
	}
	var seed uint32
	for k := 0; k < tprPartyCap; k++ {
		if in[tprInPresent+k]&0xff != 0 {
			seed += uint32(in[tprInSeed+k])
		}
	}
	for i := uint32(0); i < lead; i++ {
		r.R(100)
	}
	quantity := in[tprInQuantity] & 0xff
	chance := int32(int8(in[tprInChance]))
	if real {
		f := in[tprInFOnly:]
		chance = int32(pickpocketChanceNode([]int64{f[0], f[1], f[2], f[3], in[tprInAttempts], f[4], f[5], f[6], f[7],
			f[8], f[9], f[10], f[11], f[12], in[tprInItem], quantity}, r)[0])
	}
	after := attempts
	if after < 100 {
		after++
	}
	roll := int32(r.R(100))
	var result, grow int64 = 2, 0
	var growArgs [4]int64
	if roll <= 0x5f && roll <= 2*chance {
		if roll >= chance {
			result = 1
		} else {
			result, grow = 0, 1
			growArgs = [4]int64{in[tprInCharAddr], 0xd, 5, 0}
		}
	}
	return []int64{result, int64(after), 1, int64(seed), 1, in[tprInCharAddr], tprNPC, int64(uint32(in[tprInItem])), quantity,
		grow, growArgs[0], growArgs[1], growArgs[2], growArgs[3]}
}
