package calc

import "fmt"

// Fixture wiring (case 0538e00): eight character records (stride 0x1862 at
// 0x538fde) and eight combat slots (stride 0x106 at 0x538fd8), monster IDs
// 100..103 in the active list (count at most rtrActive), 104/105 in the reserve
// list (index 0x2710+), group IDs 200/201 active and 202 reserve. Lookups return
// the first match, active before reserve (0x4e5550, 0x5100b0).
// Callees run as address stubs in the replay (decisions/2026-10-09-compositional-stubs.md):
// 0x4c5eb0 (excluded), 0x536570, 0x547310, 0x53c490 and 0x4e8000 (D4 formulas).
// Stub returns come from inputs; the last call's arguments and the call count are outputs.
// The 0x53c490 request struct (kind, slot or monster ID, -1 fields) is built from
// the same data and is verified only by this function against the asm, not by diff.
const (
	rtrChars      = 8 // 0x538fe5: slot loop ends at 0x830 / 0x106.
	rtrActive     = 4 // fixture cap on the active monster list.
	rtrReserveN   = 2 // fixture reserve monster list.
	rtrCharBase   = 0x20001000
	rtrCharStride = 0x1862 // 0x538fde.
	rtrSlotBase   = 0x2000E000
	rtrSlotStride = 0x106 // 0x538fd8.
	rtrPendMax    = 3     // fixture cap on a pending list (stride 0x118 at 0x53ac44).
	rtrGroupMax   = 3     // fixture cap on a group member list.
	rtrMons       = 6
)

// Input layout.
const (
	rtrCheck    = 0
	rtrT        = 1
	rtrF94      = 9
	rtrF95      = 10
	rtrF96      = 11
	rtrMode     = 12
	rtrCur      = 13
	rtrUI       = 14
	rtrUI2f0    = 15
	rtrUI2f8    = 16
	rtrUIF4     = 17
	rtrG69F8    = 18
	rtrRet53    = 19
	rtrRet547   = 20
	rtrSpell    = 21 // ten spell-table kinds
	rtrMonCount = 31
	rtrMon      = 32 // 4 x (8 target dwords, kind 0x2e1, spell 0x2e5)
	rtrMask     = 72
	rtrChar     = 78 // 8 x (+0xb11, +0xb01)
	rtrSlot     = 94 // 8 x 23
	rtrPend     = 278
	rtrGrp      = 310
	rtrIn       = 322
)

var rtrSpellIDs = [10]int64{0, 1, 2, 3, 4, 5, 6, 7, 0x29, 0x95}

type rtrRec struct {
	n int64
	a [5]int64
}

func combatTargetRelationRevalidateNode(in []int64, _ Rand) []int64 {
	if len(in) != rtrIn {
		panic("combat target revalidate: expected 322 inputs")
	}
	chk := in[rtrCheck]
	var T [8]int64
	copy(T[:], in[rtrT:rtrT+8])
	var mask [rtrMons]int64
	for k := range mask {
		mask[k] = in[rtrMask+k] & 0xff
	}
	var spec2 [rtrChars][8]int64
	var pend [rtrChars]int64
	for c := range spec2 {
		copy(spec2[c][:], in[rtrSlot+23*c+13:rtrSlot+23*c+21])
		pend[c] = in[rtrPend+4*c]
	}
	var monSpec [rtrActive][8]int64
	for k := range monSpec {
		copy(monSpec[k][:], in[rtrMon+10*k:rtrMon+10*k+8])
	}
	uiF4, g69F8 := in[rtrUIF4]&0xffffffff, in[rtrG69F8]&0xffffffff
	var r4c5eb0, r536570, r547310, r53c490, r4e8000 rtrRec
	call := func(r *rtrRec, a ...int64) {
		r.n++
		r.a = [5]int64{}
		copy(r.a[:], a)
	}
	b2i := func(v bool) int64 {
		if v {
			return 1
		}
		return 0
	}
	mode, cur := in[rtrMode], in[rtrCur]
	uiSet := in[rtrUI] != 0
	activeN := func() int {
		n := in[rtrMonCount]
		if uint32(n) > rtrActive {
			panic("combat target revalidate: active monster count outside replay fixture")
		}
		return int(n)
	}
	// 0x4e5550 searches active then reserve; the entry index then resolves through 0x4e5620.
	monster := func(id int64, must bool) int {
		for i := 0; i < activeN(); i++ {
			if id == int64(100+i) {
				return i
			}
		}
		for j := 0; j < rtrReserveN; j++ {
			if id == int64(104+j) {
				return rtrActive + j
			}
		}
		if must {
			panic("combat target revalidate: unknown monster ID asserts at 0x4e55f3")
		}
		return -1
	}
	// 0x5100b0 searches active then reserve; 0x5101b0 resolves the group.
	group := func(id int64) int {
		for g := 0; g < 3; g++ {
			if id == int64(200+g) {
				return g
			}
		}
		return -1
	}
	// 0x4c5820 / 0x4c5840: monster mask byte; 0x539630 clears one slot bit, then notifies 0x4c5eb0.
	release := func(slot, id int64) {
		if k := monster(id, false); k >= 0 {
			mask[k] = mask[k] &^ (1 << uint(slot)) & 0xff
			call(&r4c5eb0, slot, id, 0)
		}
	}
	// 0x53bc10 returns the slot's input state (2: current slot with flag 0x683f95/96).
	state := func(slot int64) int64 {
		if mode == 7 && uiSet && in[rtrUI2f0] != -1 {
			switch {
			case in[rtrUI2f0] == 7 && in[rtrUI2f8]&0xff != 0:
				return 3
			case in[rtrUI2f0] == 8 && in[rtrUI2f8]&0xff != 0:
				return 4
			}
			return 7
		}
		if slot == cur && (in[rtrF95]&0xff != 0 || in[rtrF96]&0xff != 0) {
			return 2
		}
		return b2i(in[rtrF94]&0xff != 0)
	}
	// 0x53ac30: drop the slot's claim on the pending monsters, or on the spec's monster or group.
	clearClaims := func(slot int64) {
		n := pend[slot]
		if n > 0 {
			if n > rtrPendMax {
				panic("combat target revalidate: pending list outside replay fixture")
			}
			for j := int64(0); j < n; j++ {
				release(slot, in[rtrPend+int(4*slot)+1+int(j)])
			}
			pend[slot] = 0
			return
		}
		s := spec2[slot]
		if s[0] == 3 && s[2] != -1 {
			release(slot, s[2])
		}
		if s[0] == 4 && s[3] != -1 {
			g := group(s[3])
			if g < 0 {
				return
			}
			cnt := in[rtrGrp+4*g]
			if uint32(cnt) > rtrGroupMax {
				panic("combat target revalidate: group member list outside replay fixture")
			}
			for j := 0; j < int(cnt); j++ {
				release(slot, in[rtrGrp+4*g+1+j])
			}
		}
	}
	// 0x4e82e0: clear the slot's request, flag the screens, call 0x4e8000.
	cancel := func(slot int64) {
		if state(slot) != 2 {
			clearClaims(slot)
		}
		spec2[slot] = [8]int64{0, -1, -1, -1} // 0x536170.
		if mode == 6 {
			g69F8 |= 0x100 // 0x55ee52.
		}
		if mode == 7 && uiSet {
			uiF4 |= 1 << uint(slot) // 0x55ee3d -> 0x562a50.
		}
		if slot == cur && mode == 7 && uiSet {
			uiF4 |= 0x20000 // 0x565420.
		}
		call(&r4e8000, slot, in[rtrSlot+23*int(slot)+21], in[rtrSlot+23*int(slot)+22], 0, 0)
	}
	// 0x547440: kinds 0 and 3 allow, kind 2 allows a spell of table kind 3..7 other than 3 and 0x29.
	allowed := func(kind, spell int64) int64 {
		switch kind {
		case 0, 3:
			return 1
		case 2:
			if spell >= 0x96 {
				panic("combat target revalidate: spell number asserts at 0x547475")
			}
			if spell == 3 || spell == 0x29 {
				return 0
			}
			t := int64(0)
			for i, id := range rtrSpellIDs {
				if id == spell {
					t = in[rtrSpell+i]
				}
			}
			if spell < 0 {
				panic("combat target revalidate: negative spell outside replay fixture")
			}
			return b2i(t >= 3 && t <= 7)
		}
		return 0
	}

	if T[0] == 3 {
		mask[monster(T[2], true)] = 0 // 0x538e31 via 0x4c5840.
	}
	for c := int64(0); c < rtrChars; c++ {
		sb := rtrSlot + 23*int(c)
		if in[sb]&0xff == 0 {
			continue
		}
		if uint32(in[rtrChar+2*c]) == 0 || uint32(in[rtrChar+2*c+1]) >= 0x12 {
			continue
		}
		if T[0] == 3 {
			call(&r4c5eb0, c, T[2], 0)
		}
		charPtr := int64(rtrCharBase + rtrCharStride*c)
		slotPtr := int64(rtrSlotBase + rtrSlotStride*c)
		if in[rtrF94]&0xff != 0 && spec2[c] == T {
			call(&r547310, charPtr, in[sb+11], in[sb+12], slotPtr+0x45)
			call(&r53c490, 0, 0, chk&0xff, in[rtrRet547]&0xff)
			if in[rtrRet53]&0xff != 0 {
				if in[sb+11] == 5 {
					cancel(c)
				} else {
					call(&r536570, c, 1, 0)
				}
			}
		}
		var spec1 [8]int64
		copy(spec1[:], in[sb+3:sb+11])
		if spec1 == T {
			call(&r547310, charPtr, in[sb+1], in[sb+2], slotPtr+0x15)
			call(&r53c490, 0, 0, chk&0xff, in[rtrRet547]&0xff)
			if in[rtrRet53]&0xff != 0 {
				call(&r536570, c, 0, 0)
			}
		}
	}
	for i := 0; i < activeN(); i++ {
		if monSpec[i] == T {
			ok := allowed(in[rtrMon+10*i+8], in[rtrMon+10*i+9])
			call(&r53c490, 0, 0, chk&0xff, ok)
			if in[rtrRet53]&0xff != 0 {
				monSpec[i] = [8]int64{0, -1, -1, -1} // 0x5390c0.
			}
		}
	}

	var out []int64
	out = append(out, mask[:]...)
	for c := range spec2 {
		for _, v := range spec2[c] {
			out = append(out, v)
		}
	}
	for k := range monSpec {
		out = append(out, monSpec[k][:]...)
	}
	out = append(out, pend[:]...)
	out = append(out, uiF4, g69F8)
	u := func(v int64) int64 { return int64(uint32(v)) }
	for _, r := range []struct {
		r *rtrRec
		n int
		b bool
	}{{&r4c5eb0, 3, false}, {&r536570, 3, false}, {&r547310, 4, false}, {&r53c490, 4, true}, {&r4e8000, 5, false}} {
		out = append(out, r.r.n)
		for i := 0; i < r.n; i++ {
			v := r.r.a[i]
			if r.b {
				if i < 2 {
					continue
				}
				v &= 0xff
			} else {
				v = u(v)
			}
			out = append(out, v)
		}
	}
	return out
}

func init() {
	var ins, outs []string
	ins = append(ins, "check")
	for i := 0; i < 8; i++ {
		ins = append(ins, fmt.Sprintf("request%d", i))
	}
	ins = append(ins, "combatFlag94", "combatFlag95", "combatFlag96", "screenMode", "currentCharacter", "screenObject",
		"screenKind", "screenKindFlag", "screenFlags", "screenModeFlags", "relationAccepted", "offensiveAllowed")
	for _, s := range rtrSpellIDs {
		ins = append(ins, fmt.Sprintf("spell%dKind", s))
	}
	ins = append(ins, "activeMonsterCount")
	for k := 0; k < 4; k++ {
		for i := 0; i < 8; i++ {
			ins = append(ins, fmt.Sprintf("monster%dRequest%d", k, i))
		}
		ins = append(ins, fmt.Sprintf("monster%dActionKind", k), fmt.Sprintf("monster%dSpell", k))
	}
	for k := 0; k < rtrMons; k++ {
		ins = append(ins, fmt.Sprintf("monster%dMask", k))
	}
	for c := 0; c < 8; c++ {
		ins = append(ins, fmt.Sprintf("character%dHealthB11", c), fmt.Sprintf("character%dClassB01", c))
	}
	for c := 0; c < 8; c++ {
		ins = append(ins, fmt.Sprintf("slot%dActive", c), fmt.Sprintf("slot%dKind1", c), fmt.Sprintf("slot%dArg1", c))
		for i := 0; i < 8; i++ {
			ins = append(ins, fmt.Sprintf("slot%dRequest1_%d", c, i))
		}
		ins = append(ins, fmt.Sprintf("slot%dKind2", c), fmt.Sprintf("slot%dArg2", c))
		for i := 0; i < 8; i++ {
			ins = append(ins, fmt.Sprintf("slot%dRequest2_%d", c, i))
		}
		ins = append(ins, fmt.Sprintf("slot%dFinishA", c), fmt.Sprintf("slot%dFinishB", c))
	}
	for c := 0; c < 8; c++ {
		ins = append(ins, fmt.Sprintf("pending%dCount", c))
		for j := 0; j < 3; j++ {
			ins = append(ins, fmt.Sprintf("pending%dMonster%d", c, j))
		}
	}
	for g := 0; g < 3; g++ {
		ins = append(ins, fmt.Sprintf("group%dCount", g))
		for j := 0; j < 3; j++ {
			ins = append(ins, fmt.Sprintf("group%dMonster%d", g, j))
		}
	}
	for k := 0; k < rtrMons; k++ {
		outs = append(outs, fmt.Sprintf("monster%dMask", k))
	}
	for c := 0; c < 8; c++ {
		for i := 0; i < 8; i++ {
			outs = append(outs, fmt.Sprintf("slot%dRequest2_%d", c, i))
		}
	}
	for k := 0; k < 4; k++ {
		for i := 0; i < 8; i++ {
			outs = append(outs, fmt.Sprintf("monster%dRequest%d", k, i))
		}
	}
	for c := 0; c < 8; c++ {
		outs = append(outs, fmt.Sprintf("pending%dCount", c))
	}
	outs = append(outs, "screenFlags", "screenModeFlags")
	for _, s := range []struct {
		n string
		c int
	}{{"clearTarget", 3}, {"fallbackTarget", 3}, {"offensiveAllowed", 4}, {"relationAccepted", 2}, {"cancelRequest", 5}} {
		outs = append(outs, s.n+"Calls")
		for i := 1; i <= s.c; i++ {
			outs = append(outs, fmt.Sprintf("%sArg%d", s.n, i))
		}
	}
	Nodes["00538e00"] = Node{Addr: "00538e00", In: ins, Out: outs, F: combatTargetRelationRevalidateNode}
}
