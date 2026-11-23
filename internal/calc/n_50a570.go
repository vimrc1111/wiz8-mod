package calc

import "fmt"

// 0x50a570 npc_attitude_adjustment(npc, mode, charArg, item, amount): five modes move the NPC
// attitude byte (+0x1b) or the gift allowance word (+0xca), then the NPC attitude (0x50a280) is
// recomputed and returned. The jump table at 0x50a9a4 maps mode 0..4 to 0x50a58c, 0x50a654,
// 0x50a82b, 0x50a8d4, 0x50a7d8; any other mode byte (sign-extended, unsigned compare at
// 0x50a575) goes to the plain 0x50a280 tail (0x50a996).
//
// Callees run natively in the case and are called here as their Go functions: 0x554490
// (bestPartySkillNode), 0x51b840 (itemValueNode), 0x50a280 (npcAttitudeNode); 0x4e5550 /
// 0x4e5620 / 0x4e5720 (monster lookup, fixture: record found by id, template slot cached) and
// 0x51bf40 (party gold subtract, saturating at 0, 0x51bf44..0x51bf5d) are modelled here.
// 0x553f10 (D4) is address-stubbed in the case: its four arguments and call count are outputs.
const (
	n50a570Persuasion = 0x16 // skill id passed to 0x554490 and 0x553f10 (0x50a5ac, 0x50a63c)
	n50a570Stride     = 0x1862
	// Case initial values of NPC bytes the function overwrites only in some modes
	// (setup in re/emu/cases/0050a570.json); the outputs show whether they changed.
	n50a570C8 = 0x7e       // +0xc8 set to 1 by mode 0 (0x50a58f)
	n50a570C9 = 0x7d       // +0xc9 set to 1 by mode 1 (0x50a659)
	n50a570CC = 0x11223344 // +0xcc set to the 0x686a48 value by mode 0 (0x50a59b)
	n50a570D0 = 0x55667788 // +0xd0 set to the 0x686a48 value by mode 1 (0x50a666)
	// Fixture caps (record counts / id range): four item definitions, the decoy monster id.
	n50a570Items = 4
	n50a570Decoy = 0xffffffff
)

type n50a570Slot struct{ active, life, status, f89, skill int64 }

func n50a570Names() []string {
	n := []string{"mode", "charArg", "amount", "itemIndex", "itemQuantity", "itemType", "itemStackable",
		"itemBaseValue", "baseAttitude", "monsterLinked", "monsterAvailable", "monsterID", "npcKind",
		"giftAllowance", "kindAllowance", "gameTime", "partyGold", "persuadeMin", "persuadeAdj",
		"monsterTemplateLevel", "characterTable"}
	for i := 0; i < 8; i++ {
		for _, f := range []string{"active", "life", "status", "f89", "skill"} {
			n = append(n, fmt.Sprintf("slot%d.%s", i, f))
		}
	}
	n = append(n, "monsterAttitudeModifier", "monsterAttitudeOverride", "faction", "fixedFactionAttitude", "alliedFaction")
	for i := 0; i <= 20; i++ {
		n = append(n, fmt.Sprintf("factionAttitude[%d]", i))
	}
	return append(n, "npcListPointer", "npcListCount", "slot0.npcIndex", "slot1.npcIndex", "npcEntry0",
		"npc0.unavailable", "npc0.faction", "npcEntry1", "npc1.unavailable", "npc1.faction")
}

// n50a570Q is the compiler's reciprocal multiply: high word of m*x, arithmetic shift, plus the
// sign bit (0x50a5ba..0x50a5d9: /5 with 0x66666667, /-5 with 0x99999999; 0x50a5de: /100).
func n50a570Q(m, x int32, sh uint) int32 {
	e := int32((int64(m)*int64(x))>>32) >> sh
	return e + int32(uint32(e)>>31)
}

// n50a570Scaled is adj + best*(adj/5)/100 for adj>0 and best*(adj/-5)/100 + adj otherwise
// (0x50a5a5..0x50a5f4, 0x50a712..0x50a763), all in 32-bit wrap arithmetic.
func n50a570Scaled(adj int32, best uint32) int32 {
	m := int32(0x66666667)
	if adj <= 0 {
		m = -0x66666667
	}
	return adj + n50a570Q(0x51eb851f, int32(best*uint32(n50a570Q(m, adj, 1))), 5)
}

// n50a570Move adds d to the attitude byte with the 0..99 clamp (0x50a5f6..0x50a61a).
func n50a570Move(base byte, d int8) byte {
	switch s := int(base) + int(d); {
	case s > 99:
		return 99
	case s < 0:
		return 0
	}
	return base + byte(d)
}

// n50a570Credit is v + best*((v&0xffff)/5)/100 (0x50a848..0x50a875, 0x50a8f3..0x50a920).
func n50a570Credit(v, best uint32) uint32 {
	return v + uint32(n50a570Q(0x51eb851f, int32(best*uint32(n50a570Q(0x66666667, int32(v&0xffff), 1))), 5))
}

func npcAttitudeAdjustmentNode(in []int64, r Rand) []int64 {
	names := n50a570Names()
	if len(in) != len(names) {
		panic("NPC attitude adjustment: wrong input count")
	}
	at := 0
	next := func(lo, hi int64) int64 {
		v := in[at]
		if v < lo || v > hi {
			panic("NPC attitude adjustment: " + names[at] + " outside its field width")
		}
		at++
		return v
	}
	mode, charArg, amount := next(-128, 127), next(0, 1<<32-1), next(0, 1<<32-1)
	itemIndex, itemQty, itemType, itemStack, itemBase := next(0, 1<<32-1), next(0, 255), next(0, 255), next(0, 255), next(0, 1<<32-1)
	base, linked, avail, monsterID := byte(next(0, 255)), next(0, 255), next(0, 255), next(0, 1<<32-1)
	_ = next(0, 255) // npcKind: selects the 0x6836a0 record; the case supplies that record's word as kindAllowance
	allowance, kindAllowance := uint16(next(0, 65535)), uint16(next(0, 65535))
	gameTime, gold := next(0, 1<<32-1), uint32(next(0, 1<<32-1))
	persuadeMin, persuadeAdj, level, table := int32(next(-128, 127)), int32(next(-128, 127)), next(0, 255), next(0, 1<<32-1)
	var slots [8]n50a570Slot
	for i := range slots {
		s := &slots[i]
		s.active, s.life, s.status, s.f89, s.skill = next(0, 255), next(0, 1<<32-1), next(0, 1<<32-1), next(0, 1<<32-1), next(0, 1<<32-1)
	}
	rest := in[at:]
	if monsterID == n50a570Decoy && linked != 0 && avail != 0 {
		panic("NPC attitude adjustment: monster id equals the fixture decoy id")
	}
	// Mode 1 with no linked monster hands 0x4e5720 a null record: assert at 0x4e5741, then a
	// fault reading +8 (0x4e5746); the original has no completed run there.
	if mode == 1 && (linked == 0 || avail == 0) {
		panic("NPC attitude adjustment: mode 1 needs a linked, available monster")
	}
	att := func(b byte) int64 {
		a := make([]int64, 44)
		a[0], a[1], a[2] = int64(b), linked, avail
		a[3], a[4], a[5], a[6], a[7] = rest[0], rest[1], rest[2], rest[3], rest[4]
		copy(a[8:29], rest[5:26])
		a[29], a[30] = rest[26], rest[27]
		a[31], a[32], a[33] = slots[0].active, slots[0].life, rest[28]
		a[34], a[35], a[36] = slots[1].active, slots[1].life, rest[29]
		copy(a[37:43], rest[30:36])
		a[43] = monsterID
		return npcAttitudeNode(a, nil)[0]
	}
	best := func(ptr int64) (uint32, int32) { // 0x554490(0x16, ptr)
		b := []int64{n50a570Persuasion, ptr, 0}
		for _, s := range slots {
			b = append(b, s.active, s.life, s.status, s.skill)
		}
		o := bestPartySkillNode(b, nil)
		return uint32(o[0]), int32(o[1])
	}
	var c8, c9, cc, d0 int64 = n50a570C8, n50a570C9, n50a570CC, n50a570D0
	var calls, rec1, rec2, rec3, rec4 int64
	grow := func(idx int32, repeat int64) { // 0x553f10(chars+idx*0x1862, 0x16, repeat, 0), stubbed
		calls, rec1, rec2, rec3, rec4 = 1, int64(uint32(table)+uint32(idx)*n50a570Stride), n50a570Persuasion, repeat, 0
	}
	switch {
	case mode == 0:
		c8, cc = 1, gameTime
		b, idx := best(1)
		base = n50a570Move(base, int8(n50a570Scaled(persuadeAdj, b)))
		grow(idx, 8)
	case mode == 1:
		c9, d0 = 1, gameTime
		lv := uint32(level)
		if lv < 1 {
			lv = 1
		}
		var count, sum uint32
		for _, s := range slots { // 0x50a6d4..0x50a6f7
			if s.active != 0 && s.life != 0 && s.status < 0xf {
				count++
				sum += uint32(s.f89)
			}
		}
		avg := uint32(1)
		if count != 0 {
			avg = sum / count
		}
		b, _ := best(1)
		v := n50a570Scaled(persuadeMin, b)
		if v > 0 {
			v = int32(avg * uint32(v) / lv)
		}
		base = n50a570Move(base, int8(v))
		grow(int32(charArg), 5) // the index comes from the caller's third argument (0x50a7a1)
	case mode == 2 || mode == 3:
		var v uint32
		if mode == 2 {
			v = uint32(amount)
			if v > gold { // 0x51bf40
				gold = 0
			} else {
				gold -= v
			}
		} else {
			if itemIndex >= n50a570Items {
				panic("NPC attitude adjustment: item definition index past the fixture")
			}
			v = uint32(itemValueNode([]int64{itemIndex, itemQty, itemType, itemStack, itemBase}, nil)[0])
		}
		b, _ := best(0)
		low := uint16(n50a570Credit(v, b))
		if allowance < low {
			allowance = 0
		} else {
			allowance -= low
		}
		if allowance == 0 {
			att(base) // first 0x50a280 result is discarded (0x50a8b1)
			if int8(att(base)) < 0x42 {
				base = 0x4b
			}
			allowance = kindAllowance
		}
	case mode == 4:
		base = n50a570Move(base, int8(amount))
	}
	return []int64{att(base), int64(base), c8, c9, int64(allowance), cc, d0, int64(gold), calls, rec1, rec2, rec3, rec4}
}

func init() {
	Nodes["0050a570"] = Node{Addr: "0050a570", In: n50a570Names(), Out: []string{"attitude", "npcAttitude",
		"flagC8", "flagC9", "giftAllowanceAfter", "stampCC", "stampD0", "partyGoldAfter", "growthCalls",
		"growthCharacter", "growthSkill", "growthRepeat", "growthQuiet"}, F: npcAttitudeAdjustmentNode}
}
