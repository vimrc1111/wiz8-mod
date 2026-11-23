package calc

import "fmt"

// 0x50bc90 pickpocket_resolve(C, npc, itemOut, moneyOut) -> result code (al)
// (build/wiz8.asm:361622). C is a roster record (stride 0x1862 from *0x685174),
// npc the victim record: byte +0x2e, item words +0x30 (40 slots), money u32 +0x80,
// attempts i8 +0x84. Result 0 = item stolen, 1 = money stolen (amount to
// *moneyOut), 2 = failed quietly, 3 = caught, 4 = nothing to steal.
//
//  1. srand(seed): seed = sum of the dwords at C-record(2+j)+0x9ed whose presence
//     byte *(0x685178)+0x20c+0x106*j is non-zero, j = 0..5 (32-bit wrap).
//  2. R(100) is drawn and discarded n times, n = slot(C) + 11*npc[0x2e] +
//     7*(i8 npc[0x84]) as a 32-bit count (zero skips; a negative n would loop
//     about 2^32 times: calc panics).
//  3. Every item word that is not 0xffff, 0x242, 0x243, 0x244 joins a list.
//  4. Empty list or R(100) < 33: money mode, chance = 0x50baf0(C, npc, -1, 0).
//     Otherwise item mode: i = R(count) picks a list entry, 0x51c020 builds a
//     temp item (id, 1, 1, 0) and chance = 0x50baf0(C, npc, tempItem.id, 1).
//  5. attempts += 1 while the i8 is below 100.
//  6. roll = R(100): roll > 95 or roll > 2*chance -> 3; roll >= chance -> 2.
//  7. Success. Money mode: money 0 -> 2 when the list is not empty, else 4.
//     Otherwise 0x553f10(C, 0xd, 5, 0), amount = min(7*R(100), money), money -=
//     amount (a result with bit 31 set is stored as 0), *moneyOut = amount, 1.
//     Item mode: when itemOut != NULL 0x51c020(itemOut, id, 1, 1, 0); then
//     0x553f10(C, 0xd, 5, 0), the slot becomes 0xffff, 0.
//
// 0x50baf0 runs natively in the case and is reused as pickpocketChanceNode
// (same R stream; its 0x51c020 call is (id,0,0,0)). 0x51c020 and 0x553f10 (D4)
// are address stubs: their returns are ignored by the caller (checked with
// ret 0x5a5a5a5a: diff stayed 0) and the only write read back is temp item +0,
// which the stub copies from the id it just recorded, the value the real
// 0x51c020 stores (n_51c020.go: p+0 = id). F does not compose the real 0x51c020:
// its fixture caps (item records <= 4) would panic on legal ids. Their calls are
// outputs: counts and every argument. The srand import is a stub that records
// the seed. The vector growth 0x4addf0 (operator new) is assumed to succeed.
//
// Fixture caps (calc panics past them): roster slot 0..7 with charPtr =
// roster + 0x1862*slot; item ids of the chosen slot 0..15 (table of 0x50baf0);
// itemOut NULL or the fixture object; NPC link flags non-zero (0x50baf0 asserts
// otherwise).
const (
	pprChars    = 0x20000000 + 4096
	pprStride   = 0x1862
	pprOutPtr   = 0x20000000 + 82432
	pprSlots    = 40
	pprIDs      = 16
	pprInitCap  = 4
	pprSkillCap = 2
)

var pprIn = func() []string {
	n := []string{"slot", "charPtr", "characterClass", "stealBaseRating", "effectiveDexterity", "characterTotalLevel",
		"reverseDifficultyState", "difficulty"}
	for j := 0; j < 6; j++ {
		n = append(n, fmt.Sprintf("seedPresent%d", j))
	}
	for j := 0; j < 6; j++ {
		n = append(n, fmt.Sprintf("seedValue%d", j))
	}
	n = append(n, "npcByte2e", "npcMoney", "npcStealAttempts", "monsterLinkEnabled", "linkedMonsterAvailable", "linkedMonsterIdentity")
	for i := 0; i < pprSlots; i++ {
		n = append(n, fmt.Sprintf("npcItem%d", i))
	}
	n = append(n, "monsterTemplateLevel")
	for i := 0; i < pprIDs; i++ {
		n = append(n, fmt.Sprintf("unitWeight%d", i))
	}
	return append(n, "attitudeRet", "itemValueRet", "itemOutPtr")
}()

var pprOut = func() []string {
	n := []string{"result", "moneyOut", "npcMoney", "npcStealAttempts"}
	for i := 0; i < pprSlots; i++ {
		n = append(n, fmt.Sprintf("npcItem%d", i))
	}
	n = append(n, "seed", "seedCalls", "attitudeCalls", "valueCalls", "initCalls")
	for _, p := range []string{"initId", "initQty", "initA4", "initA5"} {
		for k := 0; k < pprInitCap; k++ {
			n = append(n, fmt.Sprintf("%s%d", p, k))
		}
	}
	n = append(n, "itemOutId", "skillCalls")
	for _, p := range []string{"skillChar", "skillId", "skillA3", "skillA4"} {
		for k := 0; k < pprSkillCap; k++ {
			n = append(n, fmt.Sprintf("%s%d", p, k))
		}
	}
	return n
}()

func init() {
	Nodes["0050bc90"] = Node{Addr: "0050bc90", In: pprIn, Out: pprOut, F: pickpocketResolveNode}
}

func pickpocketResolveNode(in []int64, r Rand) []int64 {
	if len(in) != len(pprIn) {
		panic("pickpocket_resolve: wrong input count")
	}
	at := map[string]int{}
	for i, n := range pprIn {
		at[n] = i
	}
	v := func(n string) int64 { return in[at[n]] }
	slot := v("slot")
	if slot < 0 || slot > 7 || v("charPtr") != pprChars+pprStride*slot {
		panic("pickpocket_resolve: character is not roster slot 0..7 of the fixture")
	}
	if p := v("itemOutPtr"); p != 0 && p != pprOutPtr {
		panic("pickpocket_resolve: itemOut is neither NULL nor the fixture object")
	}
	out := map[string]int64{"moneyOut": 0x5a5a5a5a, "itemOutId": 0xa5a5a5a5}
	var seed int64
	for j := 0; j < 6; j++ {
		if uint8(v(fmt.Sprintf("seedPresent%d", j))) != 0 {
			seed = int64(uint32(seed + v(fmt.Sprintf("seedValue%d", j))))
		}
	}
	out["seed"], out["seedCalls"] = seed, 1
	attempts := int64(int8(v("npcStealAttempts")))
	total := int32(slot) + 11*int32(uint8(v("npcByte2e"))) + 7*int32(attempts)
	if total < 0 {
		panic("pickpocket_resolve: negative discard count loops about 2^32 times")
	}
	for i := int32(0); i < total; i++ {
		r.R(100)
	}
	var list []int
	words := make([]int64, pprSlots)
	for i := range words {
		words[i] = int64(uint16(v(fmt.Sprintf("npcItem%d", i))))
		if w := words[i]; w != 0xffff && w != 0x242 && w != 0x243 && w != 0x244 {
			list = append(list, i)
		}
	}
	// chance runs 0x50baf0 and books the 0x51c020 calls (id, a3, a4, a5)
	var calls [][4]int64
	var attCalls, valCalls int64
	chance := func(id, qty int64) int64 {
		if int(id) >= pprIDs {
			panic("pickpocket_resolve: item id outside the fixture table")
		}
		w := int64(0)
		if id >= 0 {
			w = v(fmt.Sprintf("unitWeight%d", id))
		}
		o := pickpocketChanceNode([]int64{v("characterClass"), v("stealBaseRating"), v("effectiveDexterity"),
			v("characterTotalLevel"), attempts, v("monsterLinkEnabled"), v("linkedMonsterAvailable"),
			v("linkedMonsterIdentity"), v("monsterTemplateLevel"), w, v("reverseDifficultyState"), v("difficulty"),
			v("attitudeRet"), v("itemValueRet"), id, qty}, r)
		attCalls, valCalls = o[1], o[4]
		if o[2] != 0 {
			calls = append(calls, [4]int64{o[3], 0, 0, 0})
		}
		return o[0]
	}
	moneyMode := len(list) == 0 || r.R(100) < 0x21
	var id int64 = -1
	var chosen int
	var bl int64
	if moneyMode {
		bl = chance(-1, 0)
	} else {
		k := r.R(len(list))
		if k >= len(list) {
			panic("pickpocket_resolve: selector at or past the list end is unreachable")
		}
		chosen = list[k]
		id = int64(int16(words[chosen]))
		calls = append(calls, [4]int64{id, 1, 1, 0}) // temp item, +0 becomes id
		bl = chance(id, 1)
	}
	if int8(attempts) < 100 {
		attempts = int64(int8(attempts + 1))
	}
	money := v("npcMoney")
	var skills int64
	result := func() int64 {
		roll := int64(r.R(100))
		if roll > 0x5f || roll > 2*bl {
			return 3
		}
		if roll >= bl {
			return 2
		}
		if moneyMode {
			if money == 0 {
				if len(list) != 0 {
					return 2
				}
				return 4
			}
			skills++
			amt := uint32(r.R(100)) * 7
			if amt > uint32(money) {
				amt = uint32(money)
			}
			left := uint32(money) - amt
			if int32(left) < 0 {
				left = 0
			}
			money = int64(left)
			out["moneyOut"] = int64(amt)
			return 1
		}
		if words[chosen] == 0xffff {
			panic("pickpocket_resolve: chosen slot already empty is unreachable")
		}
		if v("itemOutPtr") != 0 {
			calls = append(calls, [4]int64{id, 1, 1, 0})
			out["itemOutId"] = id
		}
		skills++
		words[chosen] = 0xffff
		return 0
	}()
	if len(calls) > pprInitCap || skills > pprSkillCap {
		panic("pickpocket_resolve: more stub calls than the case records")
	}
	res := []int64{result, out["moneyOut"], money, attempts}
	res = append(res, words...)
	res = append(res, out["seed"], out["seedCalls"], attCalls, valCalls, int64(len(calls)))
	for a := 0; a < 4; a++ {
		for k := 0; k < pprInitCap; k++ {
			var x int64
			if k < len(calls) {
				x = calls[k][a]
			}
			res = append(res, x&0xffffffff)
		}
	}
	res = append(res, out["itemOutId"]&0xffffffff, skills)
	for _, x := range []int64{v("charPtr"), 0xd, 5, 0} {
		for k := 0; k < pprSkillCap; k++ {
			if int64(k) < skills {
				res = append(res, x)
			} else {
				res = append(res, 0)
			}
		}
	}
	return res
}
