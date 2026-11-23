package calc

import (
	"fmt"
	"math"
	"math/big"
)

// ---- 0x4eae20: the monster's action preparation ----
//
// Runs once when a monster's combat turn starts: the status override (0x4ec1e0)
// may force or veto an action, then the action loop tries the stored action
// (M+0x2e1) and falls back through 0x531540's choice until one sticks: melee
// swings (0x53fea0), spell (0x500330 power, 0x534290 target, 0x4fb0a0 readiness),
// special attack (0x4eb980), the move/approach branch (reach classes 0x51a800 /
// 0x51a840, the reach lengths 0x51a9a0) and intercept (0x545b20). The closing
// block clears the turn flags and asks 0x535150 whether the monster stays busy.
//
// It runs over the same memory image as the emulator case (mstWorld, built by
// gen from re/emu/cases/004eae20.json's description). Registered-out callees are
// address stubs replayed from case inputs in both runs. The D4 callees
// (0x4ec1e0, 0x5323f0, 0x53fea0, 0x500330, 0x531540, 0x5326f0, 0x545cf0,
// 0x5327e0, 0x4fd030, 0x536190, 0x533320) are address stubs too: Emu replays
// them, F calls their Go nodes on their parked worlds (mprPark*, mchPark*) with
// the values this world hands them, so F and Emu differ wherever the case's stub
// inputs differ from what those worlds compute.

const (
	mprStrTable = 0x20008000 // fixture [0x68c09c]: string pointer table
	mprStrCount = 2048       // fixture cap: string ids 0..2047
	mprPos      = 0x7e001000 // 0x4eae20 stack local: the 0x51b320 position (3 dwords)
	mprL95      = 0x7e001010 // 0x4eae20 stack local byte the 0x4c60d0 / 0x4c6070 stubs write
	mprAreaMons = 0x7e001020 // 0x5353e0 / 0x5330e0 monster list header (count at +4)
	mprAreaChrs = 0x7e001030 // 0x5330e0 character list header
)

func mprStrings(s mstSpec) mstSpec {
	for i := uint32(0); i < mprStrCount; i++ {
		s.set = append(s.set, mstSet{mprStrTable + 4*i, 4, mcaStrValue + i})
	}
	return s
}

// mprMchSpec lets the 0x531540 helpers (canCast, reach, busy) record into this
// case's stub slots.
var mprMchSpec = func() mchSpec {
	var s mchSpec
	for k, a := range map[int]uint32{mchUI: 0x4d9080, mchCand: 0x533320, mchTgt: 0x5326f0, mchPick: 0x5323f0,
		mchPower: 0x500330, mchArea: 0x4fd030} {
		x := mprStubs[a]
		s.stub[k] = macStub{ret: x.ret, retN: x.retN, rec: x.rec, cnt: x.cnt}
	}
	return s
}()

type mprWorld struct {
	*mcaWorld
	mv    *mchWorld
	tried [11]bool
	base  uint32 // 0x5330e0's first allocation: the 0x4fd030 monster ids
}

// str is the string pointer [[0x68c09c] + 4i].
func (p *mprWorld) str(i uint32) uint32 {
	if i >= mprStrCount {
		panic("monster action prepare: string id past the fixture's 2048")
	}
	return p.u32(p.u32(0x68c09c) + 4*i)
}

// feed is a parked callee world with the live values put in by name.
func mprFeed(park []int64, names []string, live map[string]int64) []int64 {
	x := append([]int64(nil), park...)
	n := 0
	for i, nm := range names {
		if v, ok := live[nm]; ok {
			x[i] = v
			n++
		}
	}
	if n != len(live) {
		panic("monster action prepare: parked world lacks a live field")
	}
	return x
}

func (p *mprWorld) b(a uint32) int64  { return int64(p.u8(a)) }
func (p *mprWorld) d(a uint32) int64  { return int64(p.u32(a)) }
func (p *mprWorld) sd(a uint32) int64 { return int64(int32(p.u32(a))) }

// override is 0x4ec1e0 on the descriptor {2, -1, id} (0x53be50).
func (p *mprWorld) override(m uint32) int32 {
	k := p.call(0x4ec1e0)
	if !p.real {
		return int32(p.replay(0x4ec1e0, k))
	}
	// The 0x4ec1e0 node's monster path holds only its fixture monster (id 77),
	// so F runs it on its parked world as is.
	return int32(statusActionOverrideNode(mprPark4ec1e0[:], p.r)[0])
}

// swings is 0x53fea0(m, t): ret and the four target words it writes.
func (p *mprWorld) swings(m, t uint32) bool {
	k := p.call(0x53fea0, m, t)
	if !p.real {
		return byte(p.replay(0x53fea0, k, m, t)) != 0
	}
	o := monsterAttackSwingsNode(mprPark53fea0[:], p.r)
	for i := uint32(0); i < 4; i++ { // Out: ret swingsOut budgetOut mTarget0Out..mTarget3Out
		p.put(m+0x2ba+4*i, 4, uint64(uint32(o[3+i])))
	}
	return byte(o[0]) != 0
}

func (p *mprWorld) interceptOK(m uint32) bool {
	k := p.call(0x545cf0, m, m+0x2ba)
	if !p.real {
		return byte(p.replay(0x545cf0, k, m, m+0x2ba)) != 0
	}
	return byte(monsterInterceptAvailableNode(mprPark545cf0[:], p.r)[0]) != 0
}

func (p *mprWorld) spellTargetOK(m, sp uint32) bool {
	k := p.call(0x5327e0, m, sp, m+0x2ba)
	if !p.real {
		return byte(p.replay(0x5327e0, k, m, sp, m+0x2ba)) != 0
	}
	rec := p.spellRecord(sp)
	x := mprFeed(mprPark5327e0[:], mstNames_ok_in(), map[string]int64{"spell": int64(sp), "q0": p.d(m + 0x2ba),
		"q1": p.d(m + 0x2be), "q2": p.d(m + 0x2c2), "spell.type": p.d(rec + 0x137), "spell.f48": p.b(rec + 0x48),
		"spell.f126": p.b(rec + 0x126), "spell.f133": p.d(rec + 0x133), "spell.f49": p.sd(rec + 0x49),
		"spell.f56": p.sd(rec + 0x56), "M.f15": p.b(m + 0x15), "M.f16": p.b(m + 0x16)})
	return byte(monsterSpellTargetOkNode(x, p.r)[0]) != 0
}

func (p *mprWorld) targetValid(q uint32) bool {
	k := p.call(0x536190, q)
	if !p.real {
		return byte(p.replay(0x536190, k, q)) != 0
	}
	x := mprFeed(mprPark536190[:], []string{"targetPointer", "targetKind", "characterIndex", "characterPresent", "characterHP",
		"characterPrimaryStatus", "resurrectionAvailable", "resurrectionBlocked", "monsterID", "groupID"},
		map[string]int64{"targetKind": p.d(q), "characterIndex": p.d(q + 4), "monsterID": p.d(q + 8), "groupID": p.d(q + 0xc)})
	return byte(combatTargetValidNode(x, p.r)[0]) != 0
}

// choose is 0x531540(m): action, spell, target words and C+4.
func (p *mprWorld) choose(m uint32) {
	k := p.call(0x531540, m)
	if !p.real {
		p.replay(0x531540, k, m)
		return
	}
	c := p.u32(m + 0x10)
	live := map[string]int64{"m.15": p.b(m + 0x15), "m.16": p.b(m + 0x16), "m.maxhp": p.d(m + 0x27), "m.hp": p.d(m + 0x2b),
		"m.2f": p.d(m + 0x2f), "m.77": p.d(m + 0x77), "m.87": p.d(m + 0x87), "m.107": p.d(m + 0x107), "m.24e": p.b(m + 0x24e),
		"m.2e1": p.d(m + 0x2e1), "m.2e5": p.d(m + 0x2e5), "m.344": p.d(m + 0x344), "c.f4": p.b(c + 4), "c.152": p.b(c + 0x152)}
	for i := uint32(0); i < 8; i++ {
		live[fmt.Sprintf("m.tgt%d", i)] = p.d(m + 0x2ba + 4*i)
	}
	o := monsterActionChoiceNode(mprFeed(mprPark531540[:], mchNames_in(), live), p.r)
	// Out: M.2e1, M.2e5, M.tgt0..7, c.f0, c.f4
	p.put(m+0x2e1, 4, uint64(uint32(o[0])))
	p.put(m+0x2e5, 4, uint64(uint32(o[1])))
	for i := uint32(0); i < 8; i++ {
		p.put(m+0x2ba+4*i, 4, uint64(uint32(o[2+i])))
	}
	p.put(c+4, 1, uint64(o[11]))
}

// blocks is 0x5330e0 through mstBlocks: the 0x4fd030 monster ids sit in its
// first allocation.
func (p *mprWorld) blocks(m, sp uint32) bool {
	c := mstCallees{power: func(_ *mstWorld, m, tmpl, spell uint32) uint32 { return p.mv.c.power(p.mv, m, tmpl, spell) }}
	if p.real {
		c.area = func(_ *mstWorld, spell, pl uint32) ([]uint32, []uint32) {
			p.call(0x4fd030, spell, pl, 0, 0, mprAreaMons, mprAreaChrs)
			x := append([]int64(nil), mchParkArea[:]...)
			x[13], x[18], x[29] = int64(spell), int64(pl), 0 // spellID, powerLevel, ignoreLineOfSight as mchReal.area
			o := spellAreaTargetSelectionNode(x, p.r)
			mons := make([]uint32, o[0])
			for i := range mons {
				mons[i] = uint32(o[2+i])
			}
			return mons, nil
		}
	} else {
		c.area = func(_ *mstWorld, spell, pl uint32) ([]uint32, []uint32) {
			p.stub(0x4fd030, spell, pl, 0, 0, mprAreaMons, mprAreaChrs)
			n := p.u32(mprAreaMons + 4)
			if n > 3 {
				panic("monster action prepare: 0x4fd030 monster count past the fixture's 3 ids")
			}
			mons := make([]uint32, n)
			for i := range mons {
				mons[i] = p.u32(p.base + 4*uint32(i))
			}
			return mons, nil
		}
	}
	p.base = mstHeap + p.heap
	return p.mstBlocks(&c, m, sp)
}

// clear14 is 0x523f40(id, 0xe): status 14 ends.
func (p *mprWorld) clear14(id uint32) {
	m := p.findMonster(id)
	if p.u32(m+0x2b) == 0 {
		return
	}
	if p.u32(m+0x57+4*14) == 0 {
		panic("monster action prepare: clearing status 14 that is not set (0x523f40 asserts)")
	}
	p.stub(0x58aad0, 9, p.str(0x910/4), p.name(m), p.str(p.u16(0x61e570+8*14)))
	p.put(m+0x57+4*14, 4, 0)
	top := uint32(0)
	for i := uint32(19); i > 0; i-- {
		if p.u32(m+0x57+4*i) != 0 {
			top = i
			break
		}
	}
	p.put(m+0x107, 4, uint64(top))
	p.groupCount(p.group(p.u32(m + 4)))
	if p.u8(m+0x14) != 0 {
		p.visual(m, p.u32(m+0x107) >= 0xe)
	}
	p.stub(0x4acd80, p.u32(m+0xc), 13, 0)
	p.rebuild(id)
}

// resetTarget is 0x536170.
func (p *mprWorld) resetTarget(q uint32) {
	for i := uint32(0); i < 8; i++ {
		p.put(q+4*i, 4, 0)
	}
	for i := uint32(1); i < 4; i++ {
		p.put(q+4*i, 4, 0xffffffff)
	}
}

// spellUsable is 0x547440(kind, spell, ptr).
func (p *mprWorld) spellUsable(kind int32, sp uint32) bool {
	switch kind {
	case 0, 3:
		return true
	case 2:
		return p.mstSpellFits(sp)
	}
	return false
}

// targetFor is 0x534290(m, spell).
func (p *mprWorld) targetFor(m, sp uint32) bool {
	if ty := p.spellType(sp, false); ty >= 5 && ty <= 6 {
		return p.mv.c.tgt(p.mv, m, sp)
	}
	if p.u8(p.spellRecord(sp)+0x13f) != 0 {
		return byte(p.stub(0x53c630, m+0x2ba, 0)) != 0
	}
	return true
}

// targetKind is 0x5011c0: the target class a spell type needs.
func (p *mprWorld) targetKind(sp uint32) uint32 {
	switch ty := p.spellType(sp, false); ty {
	case 0:
		return 8
	case 1:
		if sp == 0x58 {
			return 7
		}
		return 1
	case 3:
		return 2
	case 4:
		return 5
	case 2, 5, 6, 7, 8:
		return 0
	default:
		panic("monster action prepare: spell type past 8 (0x5011c0 asserts)")
	}
}

// targetFits is 0x537160(q, class).
func (p *mprWorld) targetFits(q, x uint32) bool {
	t0, t1, t2, t3 := p.u32(q), p.u32(q+4), p.u32(q+8), p.u32(q+0xc)
	const none = 0xffffffff
	ok := false
	switch x {
	case 0:
		return true
	case 1, 2, 8:
		ok = (t0 == 1 && t1 != none) || (t0 == 3 && t2 != none)
	case 3, 4:
		ok = t0 == 6
	case 5:
		ok = (t0 == 4 && t3 != none) || t0 == 2
	case 6:
		ok = t0 == 7 && p.u32(q+0x1c) != 0
	case 7:
		ok = t0 == 9 && t1 != none
	default:
		return true
	}
	return ok && p.targetValid(q)
}

// ready is 0x4fb0a0(m, spell, power).
func (p *mprWorld) ready(m, sp uint32) bool {
	tp := p.template(m)
	rec := p.spellRecord(sp)
	if p.u8(rec+0x126) == 0 {
		panic("monster action prepare: spell record without +0x126 (0x4fb0f7 asserts)")
	}
	if int32(p.u32(m+0x77)) > 0 {
		if p.u8(rec+0x48) == 0 {
			return false
		}
		if k := p.u8(tp + 0xcb); k != 4 && k != 5 && k != 0xd {
			return false
		}
	}
	if p.spellType(sp, false) != 0 {
		if !p.targetFits(m+0x2ba, p.targetKind(sp)) {
			return false
		}
	} else { // 0x53a2c0: the caster itself
		p.resetTarget(m + 0x2ba)
		p.put(m+0x2ba, 4, 3)
		p.put(m+0x2c2, 4, uint64(p.u32(m)))
	}
	if byte(p.stub(0x519f80, m, tp, 0, m+0x2ba)) == 0 {
		p.resetTarget(m + 0x2ba) // 0x5369f0
		return false
	}
	if byte(p.stub(0x4d9080, m, 4, 0)) != 0 {
		return false
	}
	if !p.spellTargetOK(m, sp) {
		return false
	}
	return !p.blocks(m, sp)
}

// aim is 0x534cb0(m, t).
func (p *mprWorld) aim(m, t uint32) bool {
	if p.u32(0x61eefc+8*uint32(p.u8(t+0xe3))) != 6 {
		return p.mv.c.tgt(p.mv, m, 0x77)
	}
	vis := p.u32(m + 0xc) // 0x4534c0: the object's position at +0x118
	x, y, z := p.u32(vis+0x118), p.u32(vis+0x11c), p.u32(vis+0x120)
	p.resetTarget(m + 0x2ba)
	p.put(m+0x2ba, 4, 6)
	p.put(m+0x2ca, 4, uint64(x))
	p.put(m+0x2ce, 4, uint64(y))
	p.put(m+0x2d2, 4, uint64(z))
	return true
}

// special is 0x4eb980(m, t).
func (p *mprWorld) special(m, t uint32) bool {
	if byte(p.stub(0x534a40, m, t, 0)) == 0 || !p.aim(m, t) {
		return false
	}
	sp := uint32(p.u8(t + 0xe3))
	if p.u32(0x61eefc+8*sp) != 6 {
		p.stub(0x4ec900, m, 0)
		p.stub(0x4ecb40, m)
	}
	p.put(p.u32(m+0x10)+0x145, 1, 0)
	p.stub(0x4e4db0, m, 0x12, 1)
	f := uint32(0x617584)
	if p.u8(0x68510c) != 0 {
		f = 0x614a3c
	}
	p.stub(0x58aad0, 9, f, p.name(m), p.str(p.u32(0x61ec14+4*sp)))
	return true
}

// intercept is 0x545b20(m).
func (p *mprWorld) intercept(m uint32) bool {
	tp := p.template(m)
	return p.u8(m+0x14) != 0 && p.u8(m+0x15) != 0 && p.u32(m+0x2b) != 0 && p.u32(m+0x107) < 0xc &&
		p.u8(tp+0xd0)&4 != 0 && p.u8(tp+0xe7) != 0 && p.interceptOK(m)
}

// reachClass is 0x51a840(m, 0, &out): the best reach class of the attacks and,
// for casters, of the special and the castable template spells.
func (p *mprWorld) reachClass(m uint32) (int32, uint32) {
	tp := p.template(m)
	best := int32(-1)
	for i := uint32(0); i < 3; i++ {
		if p.u8(tp+0xe7+0x22*i) != 0 {
			if v := int32(p.u8(tp + 0xea + 0x22*i)); v > best {
				best = v
			}
		}
	}
	cls, out := best, uint32(0)
	if best >= 2 && best <= 3 && p.u8(m+0x37a) != 0 { // 0x505e60
		out = 1
	}
	if p.u8(tp+0x1b9) != 1 {
		return cls, out
	}
	if sp := uint32(p.u8(tp + 0xe3)); p.u8(tp+0xe1) >= 0x50 && sp != 0 && p.u32(0x61eefc+8*sp) != 6 {
		if byte(p.stub(0x534a40, m, tp, 1)) != 0 && best < 2 {
			cls, out = 2, 2
		}
	}
	if p.u8(tp+0xe0) < 0x50 {
		return cls, out
	}
	for i := uint32(0); i < 10; i++ {
		sp := uint32(p.u8(tp + 0x14d + i))
		if !p.mstSpellFits(sp) || !p.mv.canCast(m, sp) {
			continue
		}
		if v := int32(p.u32(p.spellRecord(sp) + 0x12f)); v > cls {
			cls, out = v, 2
			if p.u8(m+0x37c) != 0 { // 0x505e80
				out = 3
			}
		}
	}
	return cls, out
}

const (
	mprLoop = iota
	mprChk
	mprTail
	mprMoved
	mprDone
	mprFin
)

// move is the approach arm (actions 5 and 7, 0x4eb01f).
func (p *mprWorld) move(m, t uint32, tr *uint32) int {
	if p.u32(m+0x87) != 0 && p.u8(t+0xcb) != 0xc {
		return mprChk
	}
	cls, l90 := int32(0), uint32(0)
	if p.u8(t+0x1b9) == 0 {
		cls = p.mv.rangeOf(t)
	} else {
		cls, l90 = p.reachClass(m)
	}
	if cls == -1 {
		p.put(m+0x2e1, 4, 6)
		return mprLoop
	}
	near, k := int32(0), uint32(0x5ebccc)
	if cls > 1 {
		near, k = 1, 0x5ed300
	}
	f94 := mchRound32(new(big.Float).SetPrec(64).Mul(p.mv.reach(near), mchF32(p.u32(k))))
	vis := p.u32(m + 0xc)
	p.stub(0x51b320, vis, l90, mprPos)
	f88 := mchRound32(p.mv.reach(cls))
	p.stub(0x51ac30, m, m+0x2ba, 0)
	p.stub(0x4eb8c0, m)
	a, _ := f94.Float32()
	b, _ := f88.Float32()
	b94, b88 := math.Float32bits(a), math.Float32bits(b)
	x, y, z := p.u32(mprPos), p.u32(mprPos+4), p.u32(mprPos+8)
	var s uint32
	switch {
	case p.u32(m+0x2ba) == 3:
		*tr = p.findMonster(p.u32(m + 0x2c2))
		s = p.stub(0x4c60d0, vis, p.u32(*tr+0xc), b94, b88, x, y, z, l90, mprL95)
	case p.u8(m+0x16) != 1:
		return mprChk
	default:
		s = p.stub(0x4c6070, vis, b94, b88, x, y, z, l90, mprL95)
	}
	s &= 0xffff
	l95 := p.u8(mprL95) != 0
	if s == 1 || (s == 3 && l95) {
		return mprMoved
	}
	if c := p.u32(m + 0x10); p.u8(c+0x152) != 0 {
		p.put(c+0x152, 1, 0)
		return mprLoop
	}
	if l95 {
		return mprChk
	}
	p.put(m+0x2e1, 4, 6)
	return mprLoop
}

// act is one pass of the action loop (0x4eafad).
func (p *mprWorld) act(m, t uint32, i3 int32, res *bool, tr *uint32) int {
	c := p.u32(m + 0x10)
	a := int32(p.u32(m + 0x2e1))
	if a != -1 {
		if a < 0 || a > 10 {
			panic("monster action prepare: action past the 11 tried flags")
		}
		p.tried[a] = true
	}
	if p.spellUsable(a, p.u32(m+0x2e5)) && i3 == -1 && p.u8(c+0x15) == 0 {
		p.stub(0x5471d0)
	}
	if uint32(a) > 9 {
		return mprChk
	}
	name := func() uint32 { return p.name(m) }
	switch dst := p.u32(0x4eb570 + 4*uint32(a)); dst {
	case 0x4eb304:
		*res = p.swings(m, t)
	case 0x4eb494:
		return mprDone
	case 0x4eb310:
		sp := p.u32(m + 0x2e5)
		pw := p.mv.c.power(p.mv, m, t, sp)
		if !p.targetFor(m, sp) {
			*res = false
			return mprTail
		}
		if !p.ready(m, sp) {
			return mprTail
		}
		p.put(m+0x2e9, 4, uint64(pw))
		p.stub(0x4ec900, m, 0)
		p.put(m+0x2e0, 1, 0)
		p.stub(0x4e4db0, m, 0x19, 1)
		return mprDone
	case 0x4eb355:
		*res = p.special(m, t)
	case 0x4eb26f:
		p.stub(0x4eb8c0, m)
		*res = p.stub(0x4c6030, p.u32(m+0xc))&0xffff != 0
		if !*res {
			return mprTail
		}
		p.put(c+0x14b, 1, 1)
		if i3 != 0xc {
			p.stub(0x58aad0, 9, p.str(0x8ec/4), name(), p.str(p.u16(0x61e434+8*uint32(p.u8(t+0xcc)))))
		}
	case 0x4eb01f:
		return p.move(m, t, tr)
	case 0x4eb1d4:
		if p.u32(m+0x2da) != 0 || (p.u8(m+0x16) == 2 && p.u32(0x683fa5) == 0) {
			p.stub(0x58aad0, 9, p.str(0x8f0/4), name())
			p.put(m+0x2e1, 4, 1)
			return mprLoop
		}
		p.stub(0x4eb8c0, m)
		*res = p.stub(0x4c6030, p.u32(m+0xc))&0xffff != 0
		if *res {
			p.stub(0x58aad0, 9, p.str(0x8e8/4), name())
		}
	case 0x4eb361:
		*res = p.intercept(m)
	case 0x4eb2eb:
		p.stub(0x4eb8c0, m)
		*res = p.stub(0x534d50, m)&0xffff != 0
	default:
		panic(fmt.Sprintf("monster action prepare: unknown action destination %#x", dst))
	}
	return mprChk
}

// prepare is 0x4eae20(m, t).
func (p *mprWorld) prepare(m, t uint32) {
	if p.u8(0x683f94) == 0 {
		panic("monster action prepare: called outside combat (0x4eae51 asserts)")
	}
	g := p.u32(0x6836a8)
	if p.u32(g+0x918) == 1 {
		p.stub(0x4efd30)
	}
	p.stub(0x48f650, m, 0, 1)
	c := p.u32(m + 0x10)
	i3 := p.override(m)
	if i3 != -1 {
		p.stub(0x58aad0, 9, 0x617584, p.name(m), p.str(p.u16(0x61e658+2*uint32(i3))))
		flag := uint32(0)
		if uint32(i3) <= 12 { // byte table 0x4eb560 into jump table 0x4eb548
			switch dst := p.u32(0x4eb548 + 4*uint32(p.u8(0x4eb560+uint32(i3)))); dst {
			case 0x4eaefe:
				p.put(m+0x2e1, 4, 0xffffffff)
			case 0x4eaf06:
				p.put(m+0x2e1, 4, 0xffffffff)
				p.clear14(p.u32(m))
			case 0x4eaf1b:
				if p.u8(c+0x15) == 0 {
					p.put(c+0x15, 1, 1)
					flag = 1
				}
			case 0x4eaf30:
				p.put(m+0x2e1, 4, 9)
			case 0x4eaf3c:
				p.put(m+0x2e1, 4, 4)
			case 0x4eaf46:
			default:
				panic(fmt.Sprintf("monster action prepare: unknown override destination %#x", dst))
			}
		}
		if p.u8(c+0x15) != 0 {
			p.put(m+0x2e1, 4, 0)
			if !p.pickWith(m, flag) {
				if i3 == 8 {
					p.stub(0x5182e0, 1, 0x617544, t, p.u32(m))
				}
				p.put(m+0x2e1, 4, 0xffffffff)
			}
		}
	}
	res := false
	tr := uint32(0)
	for st := mprLoop; st != mprFin; {
		switch st {
		case mprLoop:
			st = p.act(m, t, i3, &res, &tr)
		case mprChk:
			st = mprTail
			if res {
				st = mprDone
			}
		case mprTail:
			if i3 != -1 || p.u8(c+4) != 0 || p.u32(m+0x2e1) == 9 {
				p.put(m+0x2e1, 4, 0xffffffff)
				st = mprFin
				break
			}
			p.choose(m)
			if k := int32(p.u32(m + 0x2e1)); k == -1 || (k >= 0 && k <= 10 && p.tried[k]) {
				p.put(m+0x2e1, 4, 1)
			} else if k < -1 || k > 10 {
				panic("monster action prepare: chosen action past the 11 tried flags")
			}
			st = mprLoop
			if p.u32(m+0x2e1) == 1 {
				p.stub(0x58aad0, 9, p.str(0x8f0/4), p.name(m))
				st = mprFin
			}
		case mprMoved:
			ally := p.u32(m+0x2ba) == 3
			switch {
			case p.u32(m+0x2e1) != 5:
				if ally {
					p.stub(0x590b40, m, p.str(0x8e4/4), p.name(tr))
				} else {
					p.stub(0x590b40, m, p.str(0x8dc/4))
				}
			case p.u8(0x68510c) == 0:
			case ally:
				p.stub(0x590b40, m, p.str(0x8e0/4), p.name(tr))
			default:
				p.stub(0x590b40, m, p.str(0x8d8/4))
			}
			st = mprDone
		case mprDone:
			if a := p.u32(m + 0x2e1); a != 1 && a != 8 {
				p.stub(0x4c6200, p.u32(m+0xc), 0)
			}
			st = mprFin
		}
	}
	if a := p.u32(m + 0x2e1); a != 1 && a != 8 {
		p.put(g+0xa55, 1, 0)
	}
	if p.mv.busy(m) {
		p.put(c+0x14c, 4, 0)
	}
	p.put(c+4, 1, 1)
	p.stub(0x562a50, 0x100000)
	p.put(g+0x7b0, 4, 2)
}

func mprRun(real bool) func([]int64, Rand) []int64 {
	return func(in []int64, r Rand) []int64 {
		if len(in) != len(mprSpec.in) {
			panic("monster action prepare: wrong input count")
		}
		w := &mcaWorld{mstWorld: mstBuild(&mprSpec, in), real: real, tab: mprStubs}
		w.r = r
		p := &mprWorld{mcaWorld: w}
		p.mv = &mchWorld{mstWorld: w.mstWorld, sp: &mprMchSpec, c: &mchReal}
		if !real {
			p.mv.c = &mchCallees{
				cand:  func(_ *mchWorld, m, spell, list uint32) { p.stub(0x533320, m, spell, list) },
				tgt:   func(_ *mchWorld, m, spell uint32) bool { return byte(p.stub(0x5326f0, m, spell)) != 0 },
				power: func(_ *mchWorld, m, t, spell uint32) uint32 { return p.stub(0x500330, m, t, spell) },
				area: func(_ *mchWorld, spell, power, list uint32) {
					p.stub(0x4fd030, spell, power, 0, 0, mprAreaMons, list)
				},
			}
		}
		p.prepare(mcaArg(w, 0), mcaArg(w, 1))
		out := make([]int64, len(mprSpec.out))
		for i, f := range mprSpec.out {
			out[i] = w.read(f, in)
		}
		return out
	}
}

func monsterActionPrepareNode(in []int64, r Rand) []int64 { return mprRun(true)(in, r) }

func mprEmu(in []int64, r Rand) []int64 { return mprRun(false)(in, r) }

func init() {
	Nodes["004eae20"] = Node{Addr: "004eae20", In: mprIn, Out: mprOut, F: monsterActionPrepareNode, Emu: mprEmu}
}

// Code below is generated from the same description as re/emu/cases/004eae20.json.

var mprSpec = mprStrings(mstSpec{
	set: []mstSet{
		{0x7f000004, 4, 0x20000000},
		{0x7f000008, 4, 0x20000c00},
		{0x683fad, 4, 0x20003c00},
		{0x683fb9, 4, 0x20003c20},
		{0x683fb1, 4, 0x20003c40},
		{0x683fbd, 4, 0x20003c60},
		{0x6840c7, 4, 0x20000c00},
		{0x6840cb, 4, 0x20000f00},
		{0x6836a8, 4, 0x20003000},
		{0x68edcc, 4, 0x20003d00},
		{0x68c09c, 4, 0x20008000},
		{0x65be1c, 4, 0x20010000},
		{0x685174, 4, 0x20002600},
		{0x20002605, 4, 0x41},
		{0x20000000, 4, 0x5},
		{0x20000004, 4, 0x3},
		{0x20000008, 4, 0x0},
		{0x2000000c, 4, 0x20001200},
		{0x20000010, 4, 0x20002000},
		{0x200013d8, 4, 0x20001c80},
		{0x20000400, 4, 0x6},
		{0x20000404, 4, 0x3},
		{0x20000408, 4, 0x1},
		{0x2000040c, 4, 0x20001580},
		{0x20000410, 4, 0x20002180},
		{0x20001758, 4, 0x20001d80},
		{0x20000800, 4, 0x7},
		{0x20000804, 4, 0x3},
		{0x20000808, 4, 0x1},
		{0x2000080c, 4, 0x20001900},
		{0x20000810, 4, 0x20002300},
		{0x20001ad8, 4, 0x20001e80},
		{0x20003c80, 4, 0x3},
		{0x20003c88, 4, 0x20003cc0},
		{0x20003cc0, 4, 0x20003cd0},
		{0x20003cc8, 4, 0x3},
		{0x20003cd0, 4, 0x5},
		{0x20003cd4, 4, 0x6},
		{0x20003cd8, 4, 0x7},
		{0x20003c00, 4, 0x20003c10},
		{0x20003c08, 4, 0x3},
		{0x20003c10, 4, 0x20000000},
		{0x20003c14, 4, 0x20000400},
		{0x20003c18, 4, 0x20000800},
		{0x20003c20, 4, 0x20003c30},
		{0x20003c28, 4, 0x0},
		{0x20003c40, 4, 0x20003c50},
		{0x20003c48, 4, 0x1},
		{0x20003c50, 4, 0x20003c80},
		{0x20003c60, 4, 0x20003c70},
		{0x20003c68, 4, 0x0},
		{0x200001db, 1, 0x80},
		{0x200001dc, 1, 0x81},
		{0x200001dd, 1, 0x82},
		{0x200001de, 1, 0x83},
		{0x200001df, 1, 0x84},
		{0x200001e0, 1, 0x85},
		{0x200001e1, 1, 0x86},
		{0x200001e2, 1, 0x87},
		{0x200001e3, 1, 0x88},
		{0x200001e4, 1, 0x89},
		{0x200001e5, 1, 0x8a},
		{0x200001e6, 1, 0x8b},
		{0x200001e7, 1, 0x8c},
		{0x200001e8, 1, 0x8d},
		{0x200001e9, 1, 0x8e},
		{0x200001ea, 1, 0x8f},
		{0x200001eb, 1, 0x90},
		{0x200001ec, 1, 0x91},
		{0x200001ed, 1, 0x92},
		{0x200001ee, 1, 0x93},
		{0x200001ef, 1, 0x94},
		{0x200001f0, 1, 0x95},
		{0x200001f1, 1, 0x96},
		{0x200001f2, 1, 0x97},
		{0x200001f3, 1, 0x98},
		{0x200001f4, 1, 0x99},
		{0x200001f5, 1, 0x9a},
		{0x200001f6, 1, 0x9b},
		{0x200001f7, 1, 0x9c},
		{0x200001f8, 1, 0x9d},
		{0x200001f9, 1, 0x9e},
		{0x200001fa, 1, 0x9f},
		{0x200001fb, 1, 0xa0},
		{0x200001fc, 1, 0xa1},
		{0x200001fd, 1, 0xa2},
		{0x200001fe, 1, 0xa3},
		{0x200001ff, 1, 0xa4},
		{0x20000200, 1, 0xa5},
		{0x20000201, 1, 0xa6},
		{0x20000202, 1, 0xa7},
		{0x20000203, 1, 0xa8},
		{0x20000204, 1, 0xa9},
		{0x20000205, 1, 0xaa},
		{0x20000206, 1, 0xab},
		{0x20000207, 1, 0xac},
		{0x20000208, 1, 0xad},
		{0x20000209, 1, 0xae},
		{0x2000020a, 1, 0xaf},
		{0x2000020b, 1, 0xb0},
		{0x2000020c, 1, 0xb1},
		{0x2000020d, 1, 0xb2},
		{0x2000020e, 1, 0xb3},
		{0x2000020f, 1, 0xb4},
		{0x20000210, 1, 0xb5},
		{0x20000211, 1, 0xb6},
		{0x20000212, 1, 0xb7},
		{0x20000213, 1, 0xb8},
		{0x20000214, 1, 0xb9},
		{0x20000215, 1, 0xba},
		{0x20000216, 1, 0xbb},
		{0x20000217, 1, 0xbc},
		{0x20000218, 1, 0xbd},
		{0x20000219, 1, 0xbe},
		{0x2000021a, 1, 0xbf},
		{0x2000021b, 1, 0xc0},
		{0x2000021c, 1, 0xc1},
		{0x2000021d, 1, 0xc2},
		{0x2000021e, 1, 0xc3},
		{0x2000021f, 1, 0xc4},
		{0x20000220, 1, 0xc5},
		{0x20000221, 1, 0xc6},
		{0x20000222, 1, 0xc7},
		{0x20000223, 1, 0xc8},
		{0x20000224, 1, 0xc9},
		{0x20000225, 1, 0xca},
		{0x20000226, 1, 0xcb},
		{0x20000227, 1, 0xcc},
		{0x20000228, 1, 0xcd},
		{0x20000229, 1, 0xce},
		{0x2000022a, 1, 0xcf},
		{0x2000022b, 1, 0xd0},
		{0x2000022c, 1, 0xd1},
		{0x2000022d, 1, 0xd2},
		{0x2000022e, 1, 0xd3},
		{0x2000022f, 1, 0xd4},
		{0x20000230, 1, 0xd5},
		{0x20000231, 1, 0xd6},
		{0x20000232, 1, 0xd7},
		{0x20000233, 1, 0xd8},
		{0x20000234, 1, 0xd9},
		{0x20000235, 1, 0xda},
		{0x20000236, 1, 0xdb},
		{0x20000237, 1, 0xdc},
		{0x20000238, 1, 0xdd},
		{0x20000239, 1, 0xde},
		{0x2000023a, 1, 0xdf},
		{0x2000023b, 1, 0xe0},
		{0x2000023c, 1, 0xe1},
		{0x2000023d, 1, 0xe2},
		{0x2000023e, 1, 0xe3},
		{0x2000023f, 1, 0xe4},
		{0x20000240, 1, 0xe5},
		{0x20000241, 1, 0xe6},
		{0x20000247, 1, 0xee},
		{0x20000248, 1, 0xed},
		{0x20000249, 1, 0xec},
		{0x2000024a, 1, 0xeb},
		{0x2000024b, 1, 0xea},
		{0x20000047, 4, 0xdecade01},
		{0x2000004f, 4, 0xdecade02},
		{0x200002e9, 4, 0xdecad2e9},
		{0x200002e0, 1, 0xa5},
		{0x20002005, 4, 0xdecad005},
		{0x20002009, 4, 0xdecad009},
		{0x20002011, 4, 0xdecad011},
		{0x20002145, 1, 0x45},
		{0x2000214b, 1, 0x4b},
		{0x2000214c, 4, 0xdecad14c},
		{0x20003a55, 1, 0x55},
		{0x200037b0, 4, 0xdecad7b0},
		{0x20000414, 1, 0x1},
		{0x20000415, 1, 0x1},
		{0x2000042b, 4, 0x7},
		{0x200006e1, 4, 0xdecad001},
		{0x20000814, 1, 0x1},
		{0x20000815, 1, 0x1},
		{0x2000082b, 4, 0x7},
		{0x20000ae1, 4, 0xdecad001},
		{0x20000f60, 2, 0x7e7e},
		{0x20000c60, 2, 0x6f6f},
	},
	in: []mstFld{
		{addr: 0x20000014, size: 1, signed: false, idx: -1},
		{addr: 0x20000015, size: 1, signed: false, idx: -1},
		{addr: 0x20000016, size: 1, signed: false, idx: -1},
		{addr: 0x20000027, size: 4, signed: false, idx: -1},
		{addr: 0x2000002b, size: 4, signed: false, idx: -1},
		{addr: 0x2000002f, size: 4, signed: false, idx: -1},
		{addr: 0x20000077, size: 4, signed: false, idx: -1},
		{addr: 0x20000087, size: 4, signed: false, idx: -1},
		{addr: 0x20000057, size: 4, signed: false, idx: -1},
		{addr: 0x2000005b, size: 4, signed: false, idx: -1},
		{addr: 0x2000005f, size: 4, signed: false, idx: -1},
		{addr: 0x20000063, size: 4, signed: false, idx: -1},
		{addr: 0x20000067, size: 4, signed: false, idx: -1},
		{addr: 0x2000006b, size: 4, signed: false, idx: -1},
		{addr: 0x2000006f, size: 4, signed: false, idx: -1},
		{addr: 0x20000073, size: 4, signed: false, idx: -1},
		{addr: 0x20000077, size: 4, signed: false, idx: -1},
		{addr: 0x2000007b, size: 4, signed: false, idx: -1},
		{addr: 0x2000007f, size: 4, signed: false, idx: -1},
		{addr: 0x20000083, size: 4, signed: false, idx: -1},
		{addr: 0x20000087, size: 4, signed: false, idx: -1},
		{addr: 0x2000008b, size: 4, signed: false, idx: -1},
		{addr: 0x2000008f, size: 4, signed: false, idx: -1},
		{addr: 0x20000093, size: 4, signed: false, idx: -1},
		{addr: 0x20000097, size: 4, signed: false, idx: -1},
		{addr: 0x2000009b, size: 4, signed: false, idx: -1},
		{addr: 0x2000009f, size: 4, signed: false, idx: -1},
		{addr: 0x200000a3, size: 4, signed: false, idx: -1},
		{addr: 0x200000e3, size: 1, signed: false, idx: -1},
		{addr: 0x200000e7, size: 2, signed: false, idx: -1},
		{addr: 0x200000eb, size: 4, signed: false, idx: -1},
		{addr: 0x200000ef, size: 1, signed: false, idx: -1},
		{addr: 0x200000f3, size: 2, signed: false, idx: -1},
		{addr: 0x200000f7, size: 4, signed: false, idx: -1},
		{addr: 0x200000fb, size: 1, signed: false, idx: -1},
		{addr: 0x200000ff, size: 2, signed: false, idx: -1},
		{addr: 0x20000103, size: 4, signed: false, idx: -1},
		{addr: 0x20000107, size: 4, signed: false, idx: -1},
		{addr: 0x2000010b, size: 4, signed: false, idx: -1},
		{addr: 0x2000010f, size: 1, signed: false, idx: -1},
		{addr: 0x20000110, size: 4, signed: false, idx: -1},
		{addr: 0x20000114, size: 1, signed: false, idx: -1},
		{addr: 0x20000118, size: 4, signed: false, idx: -1},
		{addr: 0x20000164, size: 1, signed: false, idx: -1},
		{addr: 0x20000165, size: 4, signed: false, idx: -1},
		{addr: 0x20000169, size: 1, signed: false, idx: -1},
		{addr: 0x2000016d, size: 4, signed: false, idx: -1},
		{addr: 0x200001ca, size: 1, signed: false, idx: -1},
		{addr: 0x200001cb, size: 4, signed: false, idx: -1},
		{addr: 0x200001cf, size: 1, signed: false, idx: -1},
		{addr: 0x200001d3, size: 4, signed: false, idx: -1},
		{addr: 0x2000024e, size: 1, signed: false, idx: -1},
		{addr: 0x200002ba, size: 4, signed: false, idx: -1},
		{addr: 0x200002be, size: 4, signed: false, idx: -1},
		{addr: 0x200002c2, size: 4, signed: false, idx: -1},
		{addr: 0x200002c6, size: 4, signed: false, idx: -1},
		{addr: 0x200002ca, size: 4, signed: false, idx: -1},
		{addr: 0x200002ce, size: 4, signed: false, idx: -1},
		{addr: 0x200002d2, size: 4, signed: false, idx: -1},
		{addr: 0x200002d6, size: 4, signed: false, idx: -1},
		{addr: 0x200002da, size: 4, signed: false, idx: -1},
		{addr: 0x200002e1, size: 4, signed: true, idx: -1},
		{addr: 0x200002e5, size: 4, signed: false, idx: -1},
		{addr: 0x200002fd, size: 4, signed: false, idx: -1},
		{addr: 0x20000344, size: 4, signed: true, idx: -1},
		{addr: 0x2000037a, size: 1, signed: false, idx: -1},
		{addr: 0x2000037c, size: 1, signed: false, idx: -1},
		{addr: 0x20000416, size: 1, signed: false, idx: -1},
		{addr: 0x20000507, size: 4, signed: false, idx: -1},
		{addr: 0x200006fd, size: 4, signed: false, idx: -1},
		{addr: 0x20000816, size: 1, signed: false, idx: -1},
		{addr: 0x20000907, size: 4, signed: false, idx: -1},
		{addr: 0x20000afd, size: 4, signed: false, idx: -1},
		{addr: 0x20000ccb, size: 1, signed: false, idx: -1},
		{addr: 0x20000ccc, size: 1, signed: false, idx: -1},
		{addr: 0x20000cd0, size: 1, signed: false, idx: -1},
		{addr: 0x20000cd1, size: 1, signed: false, idx: -1},
		{addr: 0x20000cd2, size: 1, signed: false, idx: -1},
		{addr: 0x20000cd3, size: 1, signed: false, idx: -1},
		{addr: 0x20000cd4, size: 1, signed: false, idx: -1},
		{addr: 0x20000cd5, size: 1, signed: false, idx: -1},
		{addr: 0x20000ce0, size: 1, signed: false, idx: -1},
		{addr: 0x20000ce1, size: 1, signed: false, idx: -1},
		{addr: 0x20000ce3, size: 1, signed: false, idx: -1},
		{addr: 0x20000ce7, size: 1, signed: false, idx: -1},
		{addr: 0x20000cea, size: 1, signed: false, idx: -1},
		{addr: 0x20000d09, size: 1, signed: false, idx: -1},
		{addr: 0x20000d0c, size: 1, signed: false, idx: -1},
		{addr: 0x20000d2b, size: 1, signed: false, idx: -1},
		{addr: 0x20000d2e, size: 1, signed: false, idx: -1},
		{addr: 0x20000d4d, size: 1, signed: false, idx: -1},
		{addr: 0x20000d4e, size: 1, signed: false, idx: -1},
		{addr: 0x20000d4f, size: 1, signed: false, idx: -1},
		{addr: 0x20000d50, size: 1, signed: false, idx: -1},
		{addr: 0x20000d51, size: 1, signed: false, idx: -1},
		{addr: 0x20000d52, size: 1, signed: false, idx: -1},
		{addr: 0x20000d53, size: 1, signed: false, idx: -1},
		{addr: 0x20000d54, size: 1, signed: false, idx: -1},
		{addr: 0x20000d55, size: 1, signed: false, idx: -1},
		{addr: 0x20000d56, size: 1, signed: false, idx: -1},
		{addr: 0x20000d87, size: 2, signed: false, idx: -1},
		{addr: 0x20000db9, size: 1, signed: false, idx: -1},
		{addr: 0x20001087, size: 2, signed: false, idx: -1},
		{addr: 0x20001318, size: 4, signed: false, idx: -1},
		{addr: 0x2000131c, size: 4, signed: false, idx: -1},
		{addr: 0x20001320, size: 4, signed: false, idx: -1},
		{addr: 0x20001d27, size: 1, signed: false, idx: -1},
		{addr: 0x20002004, size: 1, signed: false, idx: -1},
		{addr: 0x20002015, size: 1, signed: false, idx: -1},
		{addr: 0x20002152, size: 1, signed: false, idx: -1},
		{addr: 0x2000203e, size: 1, signed: false, idx: -1},
		{addr: 0x2000203f, size: 4, signed: false, idx: -1},
		{addr: 0x20002043, size: 1, signed: false, idx: -1},
		{addr: 0x2000204f, size: 1, signed: false, idx: -1},
		{addr: 0x20002050, size: 4, signed: false, idx: -1},
		{addr: 0x20002054, size: 1, signed: false, idx: -1},
		{addr: 0x20002060, size: 1, signed: false, idx: -1},
		{addr: 0x20002061, size: 4, signed: false, idx: -1},
		{addr: 0x20002065, size: 1, signed: false, idx: -1},
		{addr: 0x20002071, size: 1, signed: false, idx: -1},
		{addr: 0x20002072, size: 4, signed: false, idx: -1},
		{addr: 0x20002076, size: 1, signed: false, idx: -1},
		{addr: 0x20002082, size: 1, signed: false, idx: -1},
		{addr: 0x20002083, size: 4, signed: false, idx: -1},
		{addr: 0x20002087, size: 1, signed: false, idx: -1},
		{addr: 0x20002093, size: 1, signed: false, idx: -1},
		{addr: 0x20002094, size: 4, signed: false, idx: -1},
		{addr: 0x20002098, size: 1, signed: false, idx: -1},
		{addr: 0x200020a4, size: 1, signed: false, idx: -1},
		{addr: 0x200020a5, size: 4, signed: false, idx: -1},
		{addr: 0x200020a9, size: 1, signed: false, idx: -1},
		{addr: 0x200020b5, size: 1, signed: false, idx: -1},
		{addr: 0x200020b6, size: 4, signed: false, idx: -1},
		{addr: 0x200020ba, size: 1, signed: false, idx: -1},
		{addr: 0x200020c6, size: 1, signed: false, idx: -1},
		{addr: 0x200020c7, size: 4, signed: false, idx: -1},
		{addr: 0x200020cb, size: 1, signed: false, idx: -1},
		{addr: 0x200020d7, size: 1, signed: false, idx: -1},
		{addr: 0x200020d8, size: 4, signed: false, idx: -1},
		{addr: 0x200020dc, size: 1, signed: false, idx: -1},
		{addr: 0x200020e0, size: 4, signed: false, idx: -1},
		{addr: 0x200020e8, size: 1, signed: false, idx: -1},
		{addr: 0x200020e9, size: 4, signed: false, idx: -1},
		{addr: 0x200020ed, size: 1, signed: false, idx: -1},
		{addr: 0x200020f1, size: 4, signed: false, idx: -1},
		{addr: 0x200020f9, size: 1, signed: false, idx: -1},
		{addr: 0x200020fa, size: 4, signed: false, idx: -1},
		{addr: 0x200020fe, size: 1, signed: false, idx: -1},
		{addr: 0x20002102, size: 4, signed: false, idx: -1},
		{addr: 0x2000210a, size: 1, signed: false, idx: -1},
		{addr: 0x2000210b, size: 4, signed: false, idx: -1},
		{addr: 0x2000210f, size: 1, signed: false, idx: -1},
		{addr: 0x20002113, size: 4, signed: false, idx: -1},
		{addr: 0x2000211b, size: 1, signed: false, idx: -1},
		{addr: 0x2000211c, size: 4, signed: false, idx: -1},
		{addr: 0x20002120, size: 1, signed: false, idx: -1},
		{addr: 0x20002124, size: 4, signed: false, idx: -1},
		{addr: 0x2000212c, size: 1, signed: false, idx: -1},
		{addr: 0x2000212d, size: 4, signed: false, idx: -1},
		{addr: 0x20002131, size: 1, signed: false, idx: -1},
		{addr: 0x20002135, size: 4, signed: false, idx: -1},
		{addr: 0x20003c94, size: 4, signed: false, idx: -1},
		{addr: 0x20003cac, size: 1, signed: false, idx: -1},
		{addr: 0x20003918, size: 4, signed: false, idx: -1},
		{addr: 0x683f94, size: 1, signed: false, idx: -1},
		{addr: 0x683fa5, size: 4, signed: false, idx: -1},
		{addr: 0x68510c, size: 1, signed: false, idx: -1},
		{addr: 0x652da4, size: 1, signed: false, idx: -1},
		{addr: 0x68ec78, size: 4, signed: false, idx: -1},
		{addr: 0x20003df4, size: 4, signed: false, idx: -1},
		{addr: 0x20010048, size: 1, signed: false, idx: -1},
		{addr: 0x20010126, size: 1, signed: false, idx: -1},
		{addr: 0x2001012f, size: 4, signed: true, idx: -1},
		{addr: 0x20010133, size: 4, signed: false, idx: -1},
		{addr: 0x20010137, size: 4, signed: false, idx: -1},
		{addr: 0x2001013f, size: 1, signed: false, idx: -1},
		{addr: 0x20010207, size: 1, signed: false, idx: -1},
		{addr: 0x200102e5, size: 1, signed: false, idx: -1},
		{addr: 0x200102ee, size: 4, signed: true, idx: -1},
		{addr: 0x200102f2, size: 4, signed: false, idx: -1},
		{addr: 0x200102f6, size: 4, signed: false, idx: -1},
		{addr: 0x200102fe, size: 1, signed: false, idx: -1},
		{addr: 0x200103c6, size: 1, signed: false, idx: -1},
		{addr: 0x200104a4, size: 1, signed: false, idx: -1},
		{addr: 0x200104ad, size: 4, signed: true, idx: -1},
		{addr: 0x200104b1, size: 4, signed: false, idx: -1},
		{addr: 0x200104b5, size: 4, signed: false, idx: -1},
		{addr: 0x200104bd, size: 1, signed: false, idx: -1},
		{addr: 0x20010585, size: 1, signed: false, idx: -1},
		{addr: 0x20010663, size: 1, signed: false, idx: -1},
		{addr: 0x2001066c, size: 4, signed: true, idx: -1},
		{addr: 0x20010670, size: 4, signed: false, idx: -1},
		{addr: 0x20010674, size: 4, signed: false, idx: -1},
		{addr: 0x2001067c, size: 1, signed: false, idx: -1},
		{addr: 0x20010744, size: 1, signed: false, idx: -1},
		{addr: 0x20010822, size: 1, signed: false, idx: -1},
		{addr: 0x2001082b, size: 4, signed: true, idx: -1},
		{addr: 0x2001082f, size: 4, signed: false, idx: -1},
		{addr: 0x20010833, size: 4, signed: false, idx: -1},
		{addr: 0x2001083b, size: 1, signed: false, idx: -1},
		{addr: 0x20010903, size: 1, signed: false, idx: -1},
		{addr: 0x200109e1, size: 1, signed: false, idx: -1},
		{addr: 0x200109ea, size: 4, signed: true, idx: -1},
		{addr: 0x200109ee, size: 4, signed: false, idx: -1},
		{addr: 0x200109f2, size: 4, signed: false, idx: -1},
		{addr: 0x200109fa, size: 1, signed: false, idx: -1},
		{addr: 0x20010ac2, size: 1, signed: false, idx: -1},
		{addr: 0x20010ba0, size: 1, signed: false, idx: -1},
		{addr: 0x20010ba9, size: 4, signed: true, idx: -1},
		{addr: 0x20010bad, size: 4, signed: false, idx: -1},
		{addr: 0x20010bb1, size: 4, signed: false, idx: -1},
		{addr: 0x20010bb9, size: 1, signed: false, idx: -1},
		{addr: 0x20010c81, size: 1, signed: false, idx: -1},
		{addr: 0x20010d5f, size: 1, signed: false, idx: -1},
		{addr: 0x20010d68, size: 4, signed: true, idx: -1},
		{addr: 0x20010d6c, size: 4, signed: false, idx: -1},
		{addr: 0x20010d70, size: 4, signed: false, idx: -1},
		{addr: 0x20010d78, size: 1, signed: false, idx: -1},
		{addr: 0x200040c4, size: 4, signed: false, idx: -1},
		{addr: 0x200040c8, size: 4, signed: false, idx: -1},
		{addr: 0x200040d0, size: 4, signed: false, idx: -1},
		{addr: 0x200040d8, size: 4, signed: false, idx: -1},
		{addr: 0x200040dc, size: 4, signed: false, idx: -1},
		{addr: 0x200040e0, size: 4, signed: false, idx: -1},
		{addr: 0x200040e4, size: 4, signed: false, idx: -1},
		{addr: 0x200040e8, size: 4, signed: false, idx: -1},
		{addr: 0x20004104, size: 4, signed: false, idx: -1},
		{addr: 0x20004108, size: 4, signed: false, idx: -1},
		{addr: 0x2000410c, size: 4, signed: false, idx: -1},
		{addr: 0x20004110, size: 4, signed: false, idx: -1},
		{addr: 0x20004114, size: 4, signed: false, idx: -1},
		{addr: 0x20004118, size: 4, signed: false, idx: -1},
		{addr: 0x2000411c, size: 4, signed: false, idx: -1},
		{addr: 0x20004120, size: 4, signed: false, idx: -1},
		{addr: 0x20004124, size: 4, signed: false, idx: -1},
		{addr: 0x20004128, size: 4, signed: false, idx: -1},
		{addr: 0x2000412c, size: 4, signed: false, idx: -1},
		{addr: 0x20004130, size: 4, signed: false, idx: -1},
		{addr: 0x20004134, size: 4, signed: false, idx: -1},
		{addr: 0x20004138, size: 4, signed: false, idx: -1},
		{addr: 0x2000413c, size: 4, signed: false, idx: -1},
		{addr: 0x20004140, size: 4, signed: false, idx: -1},
		{addr: 0x20004144, size: 4, signed: false, idx: -1},
		{addr: 0x20004148, size: 4, signed: false, idx: -1},
		{addr: 0x2000414c, size: 4, signed: false, idx: -1},
		{addr: 0x20004150, size: 4, signed: false, idx: -1},
		{addr: 0x20004154, size: 4, signed: false, idx: -1},
		{addr: 0x20004158, size: 4, signed: false, idx: -1},
		{addr: 0x2000415c, size: 4, signed: false, idx: -1},
		{addr: 0x20004160, size: 4, signed: false, idx: -1},
		{addr: 0x20004164, size: 4, signed: false, idx: -1},
		{addr: 0x20004168, size: 4, signed: false, idx: -1},
		{addr: 0x2000416c, size: 4, signed: false, idx: -1},
		{addr: 0x20004170, size: 4, signed: false, idx: -1},
		{addr: 0x20004174, size: 4, signed: false, idx: -1},
		{addr: 0x20004178, size: 4, signed: false, idx: -1},
		{addr: 0x2000417c, size: 4, signed: false, idx: -1},
		{addr: 0x20004180, size: 4, signed: false, idx: -1},
		{addr: 0x20004184, size: 4, signed: false, idx: -1},
		{addr: 0x20004188, size: 4, signed: false, idx: -1},
		{addr: 0x2000418c, size: 4, signed: false, idx: -1},
		{addr: 0x20004190, size: 4, signed: false, idx: -1},
		{addr: 0x20004194, size: 4, signed: false, idx: -1},
		{addr: 0x20004198, size: 4, signed: false, idx: -1},
		{addr: 0x2000419c, size: 4, signed: false, idx: -1},
		{addr: 0x200041a0, size: 4, signed: false, idx: -1},
		{addr: 0x200041a4, size: 4, signed: false, idx: -1},
		{addr: 0x200041ac, size: 1, signed: false, idx: -1},
		{addr: 0x200041ad, size: 1, signed: false, idx: -1},
		{addr: 0x200041ae, size: 1, signed: false, idx: -1},
		{addr: 0x200041af, size: 1, signed: false, idx: -1},
		{addr: 0x200041b0, size: 4, signed: false, idx: -1},
		{addr: 0x200041b4, size: 4, signed: false, idx: -1},
		{addr: 0x200041b8, size: 4, signed: false, idx: -1},
		{addr: 0x200041bc, size: 4, signed: false, idx: -1},
		{addr: 0x20004234, size: 1, signed: false, idx: -1},
		{addr: 0x20004235, size: 1, signed: false, idx: -1},
		{addr: 0x20004236, size: 1, signed: false, idx: -1},
		{addr: 0x20004237, size: 1, signed: false, idx: -1},
		{addr: 0x20004238, size: 4, signed: false, idx: -1},
		{addr: 0x2000423c, size: 4, signed: false, idx: -1},
		{addr: 0x20004240, size: 4, signed: false, idx: -1},
		{addr: 0x20004244, size: 4, signed: false, idx: -1},
		{addr: 0x200042ac, size: 4, signed: false, idx: -1},
		{addr: 0x200042b0, size: 4, signed: false, idx: -1},
		{addr: 0x200042b4, size: 4, signed: false, idx: -1},
		{addr: 0x200042b8, size: 4, signed: false, idx: -1},
		{addr: 0x200042bc, size: 4, signed: false, idx: -1},
		{addr: 0x200042c0, size: 4, signed: false, idx: -1},
		{addr: 0x200042c4, size: 4, signed: false, idx: -1},
		{addr: 0x200042c8, size: 4, signed: false, idx: -1},
		{addr: 0x200042cc, size: 4, signed: false, idx: -1},
		{addr: 0x200042d0, size: 4, signed: false, idx: -1},
		{addr: 0x200042d4, size: 4, signed: false, idx: -1},
		{addr: 0x200042d8, size: 4, signed: false, idx: -1},
		{addr: 0x200042f0, size: 4, signed: false, idx: -1},
		{addr: 0x200042f4, size: 4, signed: false, idx: -1},
		{addr: 0x200042f8, size: 4, signed: false, idx: -1},
		{addr: 0x200042fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004300, size: 4, signed: false, idx: -1},
		{addr: 0x20004304, size: 4, signed: false, idx: -1},
		{addr: 0x20004308, size: 4, signed: false, idx: -1},
		{addr: 0x2000430c, size: 4, signed: false, idx: -1},
		{addr: 0x20004310, size: 4, signed: false, idx: -1},
		{addr: 0x20004314, size: 4, signed: false, idx: -1},
		{addr: 0x20004318, size: 4, signed: false, idx: -1},
		{addr: 0x2000431c, size: 4, signed: false, idx: -1},
		{addr: 0x20004320, size: 4, signed: false, idx: -1},
		{addr: 0x20004324, size: 4, signed: false, idx: -1},
		{addr: 0x20004328, size: 4, signed: false, idx: -1},
		{addr: 0x2000432c, size: 4, signed: false, idx: -1},
		{addr: 0x20004330, size: 4, signed: false, idx: -1},
		{addr: 0x20004334, size: 4, signed: false, idx: -1},
		{addr: 0x20004338, size: 4, signed: false, idx: -1},
		{addr: 0x2000433c, size: 4, signed: false, idx: -1},
		{addr: 0x20004340, size: 4, signed: false, idx: -1},
		{addr: 0x20004344, size: 4, signed: false, idx: -1},
		{addr: 0x20004348, size: 4, signed: false, idx: -1},
		{addr: 0x2000434c, size: 4, signed: false, idx: -1},
		{addr: 0x20004350, size: 4, signed: false, idx: -1},
		{addr: 0x20004354, size: 4, signed: false, idx: -1},
		{addr: 0x20004358, size: 4, signed: false, idx: -1},
		{addr: 0x2000435c, size: 4, signed: false, idx: -1},
		{addr: 0x20004360, size: 4, signed: false, idx: -1},
		{addr: 0x20004364, size: 4, signed: false, idx: -1},
		{addr: 0x20004368, size: 4, signed: false, idx: -1},
		{addr: 0x2000436c, size: 4, signed: false, idx: -1},
		{addr: 0x20004384, size: 4, signed: false, idx: -1},
		{addr: 0x20004388, size: 4, signed: false, idx: -1},
		{addr: 0x2000438c, size: 4, signed: false, idx: -1},
		{addr: 0x20004390, size: 4, signed: false, idx: -1},
		{addr: 0x2000439c, size: 4, signed: false, idx: -1},
		{addr: 0x200043a0, size: 4, signed: false, idx: -1},
		{addr: 0x200043a4, size: 4, signed: false, idx: -1},
		{addr: 0x200043a8, size: 4, signed: false, idx: -1},
		{addr: 0x200043ac, size: 4, signed: false, idx: -1},
		{addr: 0x200043b0, size: 4, signed: false, idx: -1},
		{addr: 0x200043b4, size: 4, signed: false, idx: -1},
		{addr: 0x200043b8, size: 4, signed: false, idx: -1},
		{addr: 0x200043bc, size: 4, signed: false, idx: -1},
		{addr: 0x200043c8, size: 4, signed: true, idx: -1},
		{addr: 0x200043d0, size: 4, signed: true, idx: -1},
		{addr: 0x200043d4, size: 4, signed: false, idx: -1},
		{addr: 0x200043d8, size: 4, signed: false, idx: -1},
		{addr: 0x200043dc, size: 4, signed: false, idx: -1},
		{addr: 0x200043e0, size: 4, signed: false, idx: -1},
		{addr: 0x200043e4, size: 4, signed: false, idx: -1},
		{addr: 0x200043e8, size: 4, signed: false, idx: -1},
		{addr: 0x200043ec, size: 4, signed: false, idx: -1},
		{addr: 0x200043f0, size: 4, signed: false, idx: -1},
		{addr: 0x200043f4, size: 4, signed: false, idx: -1},
		{addr: 0x200043f8, size: 4, signed: false, idx: -1},
		{addr: 0x200043fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004400, size: 4, signed: false, idx: -1},
		{addr: 0x20004404, size: 4, signed: false, idx: -1},
		{addr: 0x20004418, size: 4, signed: false, idx: -1},
		{addr: 0x2000441c, size: 4, signed: false, idx: -1},
		{addr: 0x20004420, size: 4, signed: false, idx: -1},
		{addr: 0x20004424, size: 4, signed: false, idx: -1},
		{addr: 0x20004428, size: 4, signed: false, idx: -1},
		{addr: 0x20004430, size: 4, signed: false, idx: -1},
		{addr: 0x20004438, size: 4, signed: false, idx: -1},
		{addr: 0x20004444, size: 4, signed: false, idx: -1},
		{addr: 0x2000444c, size: 4, signed: false, idx: -1},
		{addr: 0x20004450, size: 4, signed: false, idx: -1},
		{addr: 0x20004454, size: 4, signed: false, idx: -1},
		{addr: 0x20004468, size: 4, signed: false, idx: -1},
		{addr: 0x2000446c, size: 4, signed: false, idx: -1},
		{addr: 0x20004470, size: 4, signed: false, idx: -1},
		{addr: 0x20004474, size: 4, signed: false, idx: -1},
		{addr: 0x20004478, size: 4, signed: false, idx: -1},
		{addr: 0x2000447c, size: 4, signed: false, idx: -1},
		{addr: 0x20004480, size: 4, signed: false, idx: -1},
		{addr: 0x20004484, size: 4, signed: false, idx: -1},
		{addr: 0x20004488, size: 4, signed: false, idx: -1},
		{addr: 0x2000448c, size: 4, signed: false, idx: -1},
		{addr: 0x20004490, size: 4, signed: false, idx: -1},
		{addr: 0x20004494, size: 4, signed: false, idx: -1},
		{addr: 0x20004498, size: 4, signed: false, idx: -1},
		{addr: 0x2000449c, size: 4, signed: false, idx: -1},
		{addr: 0x200044a0, size: 4, signed: false, idx: -1},
		{addr: 0x200044a4, size: 4, signed: false, idx: -1},
		{addr: 0x200044a8, size: 4, signed: false, idx: -1},
		{addr: 0x200044ac, size: 4, signed: false, idx: -1},
		{addr: 0x200044bc, size: 4, signed: false, idx: -1},
		{addr: 0x200044c0, size: 4, signed: false, idx: -1},
		{addr: 0x200044c4, size: 4, signed: false, idx: -1},
		{addr: 0x200044c8, size: 4, signed: false, idx: -1},
		{addr: 0x200044cc, size: 4, signed: false, idx: -1},
		{addr: 0x200044d0, size: 4, signed: false, idx: -1},
		{addr: 0x200044d4, size: 4, signed: false, idx: -1},
		{addr: 0x200044d8, size: 4, signed: false, idx: -1},
		{addr: 0x200044dc, size: 4, signed: false, idx: -1},
		{addr: 0x200044e0, size: 4, signed: false, idx: -1},
		{addr: 0x200044e4, size: 4, signed: false, idx: -1},
		{addr: 0x200044e8, size: 4, signed: false, idx: -1},
		{addr: 0x200044ec, size: 4, signed: false, idx: -1},
		{addr: 0x200044f0, size: 4, signed: false, idx: -1},
		{addr: 0x200044f4, size: 4, signed: false, idx: -1},
		{addr: 0x200044f8, size: 4, signed: false, idx: -1},
		{addr: 0x200044fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004500, size: 4, signed: false, idx: -1},
		{addr: 0x20004504, size: 4, signed: false, idx: -1},
		{addr: 0x20004508, size: 4, signed: false, idx: -1},
		{addr: 0x2000450c, size: 4, signed: false, idx: -1},
		{addr: 0x20004510, size: 4, signed: false, idx: -1},
		{addr: 0x20004514, size: 4, signed: false, idx: -1},
		{addr: 0x20004518, size: 4, signed: false, idx: -1},
		{addr: 0x2000451c, size: 4, signed: false, idx: -1},
		{addr: 0x20004520, size: 4, signed: false, idx: -1},
		{addr: 0x20004524, size: 4, signed: false, idx: -1},
		{addr: 0x20004528, size: 4, signed: false, idx: -1},
		{addr: 0x2000452c, size: 4, signed: false, idx: -1},
		{addr: 0x20004530, size: 4, signed: false, idx: -1},
		{addr: 0x20004534, size: 4, signed: false, idx: -1},
		{addr: 0x20004538, size: 4, signed: false, idx: -1},
		{addr: 0x2000453c, size: 4, signed: false, idx: -1},
		{addr: 0x20004540, size: 4, signed: false, idx: -1},
		{addr: 0x20004544, size: 4, signed: false, idx: -1},
		{addr: 0x20004548, size: 4, signed: false, idx: -1},
		{addr: 0x2000454c, size: 4, signed: false, idx: -1},
		{addr: 0x20004550, size: 4, signed: false, idx: -1},
		{addr: 0x20004554, size: 4, signed: false, idx: -1},
		{addr: 0x20004558, size: 4, signed: false, idx: -1},
		{addr: 0x20004600, size: 4, signed: false, idx: -1},
		{addr: 0x20004604, size: 4, signed: false, idx: -1},
		{addr: 0x20004608, size: 4, signed: false, idx: -1},
		{addr: 0x2000460c, size: 4, signed: false, idx: -1},
		{addr: 0x20004610, size: 4, signed: false, idx: -1},
		{addr: 0x20004614, size: 4, signed: false, idx: -1},
		{addr: 0x20004618, size: 4, signed: false, idx: -1},
		{addr: 0x2000461c, size: 4, signed: false, idx: -1},
		{addr: 0x20004620, size: 4, signed: false, idx: -1},
		{addr: 0x20004624, size: 4, signed: false, idx: -1},
		{addr: 0x2000463c, size: 4, signed: true, idx: -1},
		{addr: 0x20004640, size: 4, signed: true, idx: -1},
		{addr: 0x20004644, size: 4, signed: true, idx: -1},
		{addr: 0x20004648, size: 4, signed: true, idx: -1},
		{addr: 0x2000464c, size: 4, signed: true, idx: -1},
		{addr: 0x20004650, size: 4, signed: true, idx: -1},
		{addr: 0x20004654, size: 4, signed: true, idx: -1},
		{addr: 0x20004658, size: 4, signed: true, idx: -1},
		{addr: 0x2000465c, size: 4, signed: true, idx: -1},
		{addr: 0x20004660, size: 4, signed: true, idx: -1},
		{addr: 0x20004664, size: 4, signed: true, idx: -1},
		{addr: 0x20004668, size: 4, signed: false, idx: -1},
		{addr: 0x2000466c, size: 4, signed: false, idx: -1},
		{addr: 0x20004670, size: 4, signed: false, idx: -1},
		{addr: 0x20004674, size: 4, signed: false, idx: -1},
		{addr: 0x20004678, size: 4, signed: false, idx: -1},
		{addr: 0x2000467c, size: 4, signed: false, idx: -1},
		{addr: 0x20004680, size: 4, signed: false, idx: -1},
		{addr: 0x20004684, size: 4, signed: false, idx: -1},
		{addr: 0x20004688, size: 4, signed: false, idx: -1},
		{addr: 0x2000468c, size: 4, signed: false, idx: -1},
		{addr: 0x20004690, size: 4, signed: false, idx: -1},
		{addr: 0x20004694, size: 4, signed: false, idx: -1},
		{addr: 0x20004698, size: 4, signed: false, idx: -1},
		{addr: 0x2000469c, size: 4, signed: false, idx: -1},
		{addr: 0x200046a0, size: 4, signed: false, idx: -1},
		{addr: 0x200046a4, size: 4, signed: false, idx: -1},
		{addr: 0x200046a8, size: 4, signed: false, idx: -1},
		{addr: 0x200046ac, size: 4, signed: false, idx: -1},
		{addr: 0x200046b0, size: 4, signed: false, idx: -1},
		{addr: 0x200046b4, size: 4, signed: false, idx: -1},
		{addr: 0x200046b8, size: 4, signed: false, idx: -1},
		{addr: 0x200046bc, size: 4, signed: false, idx: -1},
		{addr: 0x200046c0, size: 4, signed: false, idx: -1},
		{addr: 0x200046c4, size: 4, signed: false, idx: -1},
		{addr: 0x200046c8, size: 4, signed: false, idx: -1},
		{addr: 0x200046cc, size: 4, signed: false, idx: -1},
		{addr: 0x200046d0, size: 4, signed: false, idx: -1},
		{addr: 0x200046d4, size: 4, signed: false, idx: -1},
		{addr: 0x200046d8, size: 4, signed: false, idx: -1},
		{addr: 0x200046dc, size: 4, signed: false, idx: -1},
		{addr: 0x200046e0, size: 4, signed: false, idx: -1},
		{addr: 0x200046e4, size: 4, signed: false, idx: -1},
		{addr: 0x200046e8, size: 4, signed: false, idx: -1},
		{addr: 0x200046ec, size: 4, signed: false, idx: -1},
		{addr: 0x200046f0, size: 4, signed: false, idx: -1},
		{addr: 0x200046f4, size: 4, signed: false, idx: -1},
		{addr: 0x200046f8, size: 4, signed: false, idx: -1},
		{addr: 0x200046fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004700, size: 4, signed: false, idx: -1},
		{addr: 0x20004704, size: 4, signed: false, idx: -1},
		{addr: 0x20004708, size: 4, signed: false, idx: -1},
		{addr: 0x2000470c, size: 4, signed: false, idx: -1},
		{addr: 0x20004710, size: 4, signed: false, idx: -1},
		{addr: 0x20004714, size: 4, signed: false, idx: -1},
		{addr: 0x20004718, size: 4, signed: false, idx: -1},
		{addr: 0x2000471c, size: 4, signed: false, idx: -1},
		{addr: 0x20004720, size: 4, signed: false, idx: -1},
		{addr: 0x20004724, size: 4, signed: false, idx: -1},
		{addr: 0x20004728, size: 4, signed: false, idx: -1},
		{addr: 0x2000472c, size: 4, signed: false, idx: -1},
		{addr: 0x20004730, size: 4, signed: false, idx: -1},
		{addr: 0x20004734, size: 4, signed: false, idx: -1},
		{addr: 0x20004738, size: 4, signed: false, idx: -1},
		{addr: 0x2000473c, size: 4, signed: false, idx: -1},
		{addr: 0x20004740, size: 4, signed: false, idx: -1},
		{addr: 0x20004744, size: 4, signed: false, idx: -1},
		{addr: 0x20004748, size: 4, signed: false, idx: -1},
		{addr: 0x2000474c, size: 4, signed: false, idx: -1},
		{addr: 0x20004750, size: 4, signed: false, idx: -1},
		{addr: 0x20004754, size: 4, signed: false, idx: -1},
		{addr: 0x20004758, size: 4, signed: false, idx: -1},
		{addr: 0x2000475c, size: 4, signed: false, idx: -1},
		{addr: 0x20004760, size: 4, signed: false, idx: -1},
		{addr: 0x20004764, size: 4, signed: false, idx: -1},
		{addr: 0x20004768, size: 4, signed: false, idx: -1},
		{addr: 0x2000476c, size: 4, signed: false, idx: -1},
		{addr: 0x20004770, size: 4, signed: false, idx: -1},
		{addr: 0x20004774, size: 4, signed: false, idx: -1},
		{addr: 0x20004778, size: 4, signed: false, idx: -1},
		{addr: 0x2000477c, size: 4, signed: false, idx: -1},
		{addr: 0x20004780, size: 4, signed: false, idx: -1},
		{addr: 0x20004784, size: 4, signed: false, idx: -1},
		{addr: 0x20004788, size: 4, signed: false, idx: -1},
		{addr: 0x2000478c, size: 4, signed: false, idx: -1},
		{addr: 0x20004790, size: 4, signed: false, idx: -1},
		{addr: 0x20004794, size: 4, signed: false, idx: -1},
		{addr: 0x20004798, size: 4, signed: false, idx: -1},
		{addr: 0x2000479c, size: 4, signed: false, idx: -1},
		{addr: 0x200047a0, size: 4, signed: false, idx: -1},
		{addr: 0x200047a4, size: 4, signed: false, idx: -1},
		{addr: 0x200047a8, size: 4, signed: false, idx: -1},
		{addr: 0x200047ac, size: 4, signed: false, idx: -1},
		{addr: 0x200047b0, size: 4, signed: false, idx: -1},
		{addr: 0x200047b4, size: 4, signed: false, idx: -1},
		{addr: 0x200047b8, size: 4, signed: false, idx: -1},
		{addr: 0x200047bc, size: 4, signed: false, idx: -1},
		{addr: 0x200047c0, size: 4, signed: false, idx: -1},
		{addr: 0x200047c4, size: 4, signed: false, idx: -1},
		{addr: 0x200047c8, size: 4, signed: false, idx: -1},
		{addr: 0x200047cc, size: 4, signed: false, idx: -1},
		{addr: 0x200047d0, size: 4, signed: false, idx: -1},
		{addr: 0x200047d4, size: 4, signed: false, idx: -1},
		{addr: 0x200047d8, size: 4, signed: false, idx: -1},
		{addr: 0x200047dc, size: 4, signed: false, idx: -1},
		{addr: 0x200047e0, size: 4, signed: false, idx: -1},
		{addr: 0x200047e4, size: 4, signed: false, idx: -1},
		{addr: 0x200047e8, size: 4, signed: false, idx: -1},
		{addr: 0x200047ec, size: 4, signed: false, idx: -1},
		{addr: 0x200047f0, size: 4, signed: false, idx: -1},
		{addr: 0x200047f4, size: 1, signed: false, idx: -1},
		{addr: 0x200047f5, size: 1, signed: false, idx: -1},
		{addr: 0x200047f6, size: 1, signed: false, idx: -1},
		{addr: 0x200047f7, size: 1, signed: false, idx: -1},
		{addr: 0x200047f8, size: 1, signed: false, idx: -1},
		{addr: 0x200047f9, size: 1, signed: false, idx: -1},
		{addr: 0x200047fa, size: 1, signed: false, idx: -1},
		{addr: 0x200047fb, size: 1, signed: false, idx: -1},
		{addr: 0x200047fc, size: 1, signed: false, idx: -1},
		{addr: 0x200047fd, size: 1, signed: false, idx: -1},
		{addr: 0x200047fe, size: 1, signed: false, idx: -1},
	},
	out: []mstFld{
		{addr: 0x200002e1, size: 4, signed: true, idx: -1},
		{addr: 0x200002e5, size: 4, signed: false, idx: -1},
		{addr: 0x200002e9, size: 4, signed: false, idx: -1},
		{addr: 0x200002e0, size: 1, signed: false, idx: -1},
		{addr: 0x200002ba, size: 4, signed: false, idx: -1},
		{addr: 0x200002be, size: 4, signed: false, idx: -1},
		{addr: 0x200002c2, size: 4, signed: false, idx: -1},
		{addr: 0x200002c6, size: 4, signed: false, idx: -1},
		{addr: 0x200002ca, size: 4, signed: false, idx: -1},
		{addr: 0x200002ce, size: 4, signed: false, idx: -1},
		{addr: 0x200002d2, size: 4, signed: false, idx: -1},
		{addr: 0x200002d6, size: 4, signed: false, idx: -1},
		{addr: 0x2000008f, size: 4, signed: false, idx: -1},
		{addr: 0x20000107, size: 4, signed: false, idx: -1},
		{addr: 0x2000024e, size: 1, signed: false, idx: -1},
		{addr: 0x200001db, size: 1, signed: false, idx: -1},
		{addr: 0x200001dc, size: 1, signed: false, idx: -1},
		{addr: 0x200001dd, size: 1, signed: false, idx: -1},
		{addr: 0x200001de, size: 1, signed: false, idx: -1},
		{addr: 0x200001df, size: 1, signed: false, idx: -1},
		{addr: 0x200001e0, size: 1, signed: false, idx: -1},
		{addr: 0x200001e1, size: 1, signed: false, idx: -1},
		{addr: 0x200001e2, size: 1, signed: false, idx: -1},
		{addr: 0x200001e3, size: 1, signed: false, idx: -1},
		{addr: 0x200001e4, size: 1, signed: false, idx: -1},
		{addr: 0x200001e5, size: 1, signed: false, idx: -1},
		{addr: 0x200001e6, size: 1, signed: false, idx: -1},
		{addr: 0x200001e7, size: 1, signed: false, idx: -1},
		{addr: 0x200001e8, size: 1, signed: false, idx: -1},
		{addr: 0x200001e9, size: 1, signed: false, idx: -1},
		{addr: 0x200001ea, size: 1, signed: false, idx: -1},
		{addr: 0x200001eb, size: 1, signed: false, idx: -1},
		{addr: 0x200001ec, size: 1, signed: false, idx: -1},
		{addr: 0x200001ed, size: 1, signed: false, idx: -1},
		{addr: 0x200001ee, size: 1, signed: false, idx: -1},
		{addr: 0x200001ef, size: 1, signed: false, idx: -1},
		{addr: 0x200001f0, size: 1, signed: false, idx: -1},
		{addr: 0x200001f1, size: 1, signed: false, idx: -1},
		{addr: 0x200001f2, size: 1, signed: false, idx: -1},
		{addr: 0x200001f3, size: 1, signed: false, idx: -1},
		{addr: 0x200001f4, size: 1, signed: false, idx: -1},
		{addr: 0x200001f5, size: 1, signed: false, idx: -1},
		{addr: 0x200001f6, size: 1, signed: false, idx: -1},
		{addr: 0x200001f7, size: 1, signed: false, idx: -1},
		{addr: 0x200001f8, size: 1, signed: false, idx: -1},
		{addr: 0x200001f9, size: 1, signed: false, idx: -1},
		{addr: 0x200001fa, size: 1, signed: false, idx: -1},
		{addr: 0x200001fb, size: 1, signed: false, idx: -1},
		{addr: 0x200001fc, size: 1, signed: false, idx: -1},
		{addr: 0x200001fd, size: 1, signed: false, idx: -1},
		{addr: 0x200001fe, size: 1, signed: false, idx: -1},
		{addr: 0x200001ff, size: 1, signed: false, idx: -1},
		{addr: 0x20000200, size: 1, signed: false, idx: -1},
		{addr: 0x20000201, size: 1, signed: false, idx: -1},
		{addr: 0x20000202, size: 1, signed: false, idx: -1},
		{addr: 0x20000203, size: 1, signed: false, idx: -1},
		{addr: 0x20000204, size: 1, signed: false, idx: -1},
		{addr: 0x20000205, size: 1, signed: false, idx: -1},
		{addr: 0x20000206, size: 1, signed: false, idx: -1},
		{addr: 0x20000207, size: 1, signed: false, idx: -1},
		{addr: 0x20000208, size: 1, signed: false, idx: -1},
		{addr: 0x20000209, size: 1, signed: false, idx: -1},
		{addr: 0x2000020a, size: 1, signed: false, idx: -1},
		{addr: 0x2000020b, size: 1, signed: false, idx: -1},
		{addr: 0x2000020c, size: 1, signed: false, idx: -1},
		{addr: 0x2000020d, size: 1, signed: false, idx: -1},
		{addr: 0x2000020e, size: 1, signed: false, idx: -1},
		{addr: 0x2000020f, size: 1, signed: false, idx: -1},
		{addr: 0x20000210, size: 1, signed: false, idx: -1},
		{addr: 0x20000211, size: 1, signed: false, idx: -1},
		{addr: 0x20000212, size: 1, signed: false, idx: -1},
		{addr: 0x20000213, size: 1, signed: false, idx: -1},
		{addr: 0x20000214, size: 1, signed: false, idx: -1},
		{addr: 0x20000215, size: 1, signed: false, idx: -1},
		{addr: 0x20000216, size: 1, signed: false, idx: -1},
		{addr: 0x20000217, size: 1, signed: false, idx: -1},
		{addr: 0x20000218, size: 1, signed: false, idx: -1},
		{addr: 0x20000219, size: 1, signed: false, idx: -1},
		{addr: 0x2000021a, size: 1, signed: false, idx: -1},
		{addr: 0x2000021b, size: 1, signed: false, idx: -1},
		{addr: 0x2000021c, size: 1, signed: false, idx: -1},
		{addr: 0x2000021d, size: 1, signed: false, idx: -1},
		{addr: 0x2000021e, size: 1, signed: false, idx: -1},
		{addr: 0x2000021f, size: 1, signed: false, idx: -1},
		{addr: 0x20000220, size: 1, signed: false, idx: -1},
		{addr: 0x20000221, size: 1, signed: false, idx: -1},
		{addr: 0x20000222, size: 1, signed: false, idx: -1},
		{addr: 0x20000223, size: 1, signed: false, idx: -1},
		{addr: 0x20000224, size: 1, signed: false, idx: -1},
		{addr: 0x20000225, size: 1, signed: false, idx: -1},
		{addr: 0x20000226, size: 1, signed: false, idx: -1},
		{addr: 0x20000227, size: 1, signed: false, idx: -1},
		{addr: 0x20000228, size: 1, signed: false, idx: -1},
		{addr: 0x20000229, size: 1, signed: false, idx: -1},
		{addr: 0x2000022a, size: 1, signed: false, idx: -1},
		{addr: 0x2000022b, size: 1, signed: false, idx: -1},
		{addr: 0x2000022c, size: 1, signed: false, idx: -1},
		{addr: 0x2000022d, size: 1, signed: false, idx: -1},
		{addr: 0x2000022e, size: 1, signed: false, idx: -1},
		{addr: 0x2000022f, size: 1, signed: false, idx: -1},
		{addr: 0x20000230, size: 1, signed: false, idx: -1},
		{addr: 0x20000231, size: 1, signed: false, idx: -1},
		{addr: 0x20000232, size: 1, signed: false, idx: -1},
		{addr: 0x20000233, size: 1, signed: false, idx: -1},
		{addr: 0x20000234, size: 1, signed: false, idx: -1},
		{addr: 0x20000235, size: 1, signed: false, idx: -1},
		{addr: 0x20000236, size: 1, signed: false, idx: -1},
		{addr: 0x20000237, size: 1, signed: false, idx: -1},
		{addr: 0x20000238, size: 1, signed: false, idx: -1},
		{addr: 0x20000239, size: 1, signed: false, idx: -1},
		{addr: 0x2000023a, size: 1, signed: false, idx: -1},
		{addr: 0x2000023b, size: 1, signed: false, idx: -1},
		{addr: 0x2000023c, size: 1, signed: false, idx: -1},
		{addr: 0x2000023d, size: 1, signed: false, idx: -1},
		{addr: 0x2000023e, size: 1, signed: false, idx: -1},
		{addr: 0x2000023f, size: 1, signed: false, idx: -1},
		{addr: 0x20000240, size: 1, signed: false, idx: -1},
		{addr: 0x20000241, size: 1, signed: false, idx: -1},
		{addr: 0x20000247, size: 1, signed: false, idx: -1},
		{addr: 0x20000248, size: 1, signed: false, idx: -1},
		{addr: 0x20000249, size: 1, signed: false, idx: -1},
		{addr: 0x2000024a, size: 1, signed: false, idx: -1},
		{addr: 0x2000024b, size: 1, signed: false, idx: -1},
		{addr: 0x20000047, size: 4, signed: false, idx: -1},
		{addr: 0x2000004f, size: 4, signed: false, idx: -1},
		{addr: 0x20002004, size: 1, signed: false, idx: -1},
		{addr: 0x20002005, size: 4, signed: false, idx: -1},
		{addr: 0x20002009, size: 4, signed: false, idx: -1},
		{addr: 0x20002011, size: 4, signed: false, idx: -1},
		{addr: 0x20002015, size: 1, signed: false, idx: -1},
		{addr: 0x20002145, size: 1, signed: false, idx: -1},
		{addr: 0x2000214b, size: 1, signed: false, idx: -1},
		{addr: 0x2000214c, size: 4, signed: false, idx: -1},
		{addr: 0x20002152, size: 1, signed: false, idx: -1},
		{addr: 0x20003c94, size: 4, signed: false, idx: -1},
		{addr: 0x20003df4, size: 4, signed: false, idx: -1},
		{addr: 0x20003a55, size: 1, signed: false, idx: -1},
		{addr: 0x200037b0, size: 4, signed: false, idx: -1},
		{addr: 0x20004000, size: 4, signed: false, idx: -1},
		{addr: 0x20004004, size: 4, signed: false, idx: -1},
		{addr: 0x20004008, size: 4, signed: false, idx: -1},
		{addr: 0x2000400c, size: 4, signed: false, idx: -1},
		{addr: 0x20004010, size: 4, signed: false, idx: -1},
		{addr: 0x20004014, size: 4, signed: false, idx: -1},
		{addr: 0x20004018, size: 4, signed: false, idx: -1},
		{addr: 0x2000401c, size: 4, signed: false, idx: -1},
		{addr: 0x20004020, size: 4, signed: false, idx: -1},
		{addr: 0x20004024, size: 4, signed: false, idx: -1},
		{addr: 0x20004028, size: 4, signed: false, idx: -1},
		{addr: 0x2000402c, size: 4, signed: false, idx: -1},
		{addr: 0x20004030, size: 4, signed: false, idx: -1},
		{addr: 0x20004034, size: 4, signed: false, idx: -1},
		{addr: 0x20004038, size: 4, signed: false, idx: -1},
		{addr: 0x2000403c, size: 4, signed: false, idx: -1},
		{addr: 0x20004040, size: 4, signed: false, idx: -1},
		{addr: 0x20004044, size: 4, signed: false, idx: -1},
		{addr: 0x20004048, size: 4, signed: false, idx: -1},
		{addr: 0x2000404c, size: 4, signed: false, idx: -1},
		{addr: 0x20004050, size: 4, signed: false, idx: -1},
		{addr: 0x20004054, size: 4, signed: false, idx: -1},
		{addr: 0x20004058, size: 4, signed: false, idx: -1},
		{addr: 0x2000405c, size: 4, signed: false, idx: -1},
		{addr: 0x20004060, size: 4, signed: false, idx: -1},
		{addr: 0x20004064, size: 4, signed: false, idx: -1},
		{addr: 0x20004068, size: 4, signed: false, idx: -1},
		{addr: 0x2000406c, size: 4, signed: false, idx: -1},
		{addr: 0x20004070, size: 4, signed: false, idx: -1},
		{addr: 0x20004074, size: 4, signed: false, idx: -1},
		{addr: 0x20004078, size: 4, signed: false, idx: -1},
		{addr: 0x2000407c, size: 4, signed: false, idx: -1},
		{addr: 0x20004080, size: 4, signed: false, idx: -1},
		{addr: 0x20004084, size: 4, signed: false, idx: -1},
		{addr: 0x20004088, size: 4, signed: false, idx: -1},
		{addr: 0x2000408c, size: 4, signed: false, idx: -1},
		{addr: 0x20004090, size: 4, signed: false, idx: -1},
		{addr: 0x20004094, size: 4, signed: false, idx: -1},
		{addr: 0x20004098, size: 4, signed: false, idx: -1},
		{addr: 0x2000409c, size: 4, signed: false, idx: -1},
		{addr: 0x200040a0, size: 4, signed: false, idx: -1},
		{addr: 0x200040a4, size: 4, signed: false, idx: -1},
		{addr: 0x200040a8, size: 4, signed: false, idx: -1},
		{addr: 0x200040ac, size: 4, signed: false, idx: -1},
		{addr: 0x200040b0, size: 4, signed: false, idx: -1},
		{addr: 0x200040b4, size: 4, signed: false, idx: -1},
		{addr: 0x200040b8, size: 4, signed: false, idx: -1},
		{addr: 0x200040bc, size: 4, signed: false, idx: -1},
		{addr: 0x200040c0, size: 4, signed: false, idx: -1},
		{addr: 0x200040cc, size: 4, signed: false, idx: -1},
		{addr: 0x200040d4, size: 4, signed: false, idx: -1},
		{addr: 0x200040ec, size: 4, signed: false, idx: -1},
		{addr: 0x200040f0, size: 4, signed: false, idx: -1},
		{addr: 0x200040f4, size: 4, signed: false, idx: -1},
		{addr: 0x200040f8, size: 4, signed: false, idx: -1},
		{addr: 0x200040fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004100, size: 4, signed: false, idx: -1},
		{addr: 0x200041a8, size: 4, signed: false, idx: -1},
		{addr: 0x200041c0, size: 4, signed: false, idx: -1},
		{addr: 0x200041c4, size: 4, signed: false, idx: -1},
		{addr: 0x200041c8, size: 4, signed: false, idx: -1},
		{addr: 0x200041cc, size: 4, signed: false, idx: -1},
		{addr: 0x200041d0, size: 4, signed: false, idx: -1},
		{addr: 0x200041d4, size: 4, signed: false, idx: -1},
		{addr: 0x200041d8, size: 4, signed: false, idx: -1},
		{addr: 0x200041dc, size: 4, signed: false, idx: -1},
		{addr: 0x200041e0, size: 4, signed: false, idx: -1},
		{addr: 0x200041e4, size: 4, signed: false, idx: -1},
		{addr: 0x200041e8, size: 4, signed: false, idx: -1},
		{addr: 0x200041ec, size: 4, signed: false, idx: -1},
		{addr: 0x200041f0, size: 4, signed: false, idx: -1},
		{addr: 0x200041f4, size: 4, signed: false, idx: -1},
		{addr: 0x200041f8, size: 4, signed: false, idx: -1},
		{addr: 0x200041fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004200, size: 4, signed: false, idx: -1},
		{addr: 0x20004204, size: 4, signed: false, idx: -1},
		{addr: 0x20004208, size: 4, signed: false, idx: -1},
		{addr: 0x2000420c, size: 4, signed: false, idx: -1},
		{addr: 0x20004210, size: 4, signed: false, idx: -1},
		{addr: 0x20004214, size: 4, signed: false, idx: -1},
		{addr: 0x20004218, size: 4, signed: false, idx: -1},
		{addr: 0x2000421c, size: 4, signed: false, idx: -1},
		{addr: 0x20004220, size: 4, signed: false, idx: -1},
		{addr: 0x20004224, size: 4, signed: false, idx: -1},
		{addr: 0x20004228, size: 4, signed: false, idx: -1},
		{addr: 0x2000422c, size: 4, signed: false, idx: -1},
		{addr: 0x20004230, size: 4, signed: false, idx: -1},
		{addr: 0x20004248, size: 4, signed: false, idx: -1},
		{addr: 0x2000424c, size: 4, signed: false, idx: -1},
		{addr: 0x20004250, size: 4, signed: false, idx: -1},
		{addr: 0x20004254, size: 4, signed: false, idx: -1},
		{addr: 0x20004258, size: 4, signed: false, idx: -1},
		{addr: 0x2000425c, size: 4, signed: false, idx: -1},
		{addr: 0x20004260, size: 4, signed: false, idx: -1},
		{addr: 0x20004264, size: 4, signed: false, idx: -1},
		{addr: 0x20004268, size: 4, signed: false, idx: -1},
		{addr: 0x2000426c, size: 4, signed: false, idx: -1},
		{addr: 0x20004270, size: 4, signed: false, idx: -1},
		{addr: 0x20004274, size: 4, signed: false, idx: -1},
		{addr: 0x20004278, size: 4, signed: false, idx: -1},
		{addr: 0x2000427c, size: 4, signed: false, idx: -1},
		{addr: 0x20004280, size: 4, signed: false, idx: -1},
		{addr: 0x20004284, size: 4, signed: false, idx: -1},
		{addr: 0x20004288, size: 4, signed: false, idx: -1},
		{addr: 0x2000428c, size: 4, signed: false, idx: -1},
		{addr: 0x20004290, size: 4, signed: false, idx: -1},
		{addr: 0x20004294, size: 4, signed: false, idx: -1},
		{addr: 0x20004298, size: 4, signed: false, idx: -1},
		{addr: 0x2000429c, size: 4, signed: false, idx: -1},
		{addr: 0x200042a0, size: 4, signed: false, idx: -1},
		{addr: 0x200042a4, size: 4, signed: false, idx: -1},
		{addr: 0x200042a8, size: 4, signed: false, idx: -1},
		{addr: 0x200042dc, size: 4, signed: false, idx: -1},
		{addr: 0x200042e0, size: 4, signed: false, idx: -1},
		{addr: 0x200042e4, size: 4, signed: false, idx: -1},
		{addr: 0x200042e8, size: 4, signed: false, idx: -1},
		{addr: 0x200042ec, size: 4, signed: false, idx: -1},
		{addr: 0x20004370, size: 4, signed: false, idx: -1},
		{addr: 0x20004374, size: 4, signed: false, idx: -1},
		{addr: 0x20004378, size: 4, signed: false, idx: -1},
		{addr: 0x2000437c, size: 4, signed: false, idx: -1},
		{addr: 0x20004380, size: 4, signed: false, idx: -1},
		{addr: 0x20004394, size: 4, signed: false, idx: -1},
		{addr: 0x20004398, size: 4, signed: false, idx: -1},
		{addr: 0x200043c0, size: 4, signed: false, idx: -1},
		{addr: 0x200043c4, size: 4, signed: false, idx: -1},
		{addr: 0x200043cc, size: 4, signed: false, idx: -1},
		{addr: 0x20004408, size: 4, signed: false, idx: -1},
		{addr: 0x2000440c, size: 4, signed: false, idx: -1},
		{addr: 0x20004410, size: 4, signed: false, idx: -1},
		{addr: 0x20004414, size: 4, signed: false, idx: -1},
		{addr: 0x2000442c, size: 4, signed: false, idx: -1},
		{addr: 0x20004434, size: 4, signed: false, idx: -1},
		{addr: 0x2000443c, size: 4, signed: false, idx: -1},
		{addr: 0x20004440, size: 4, signed: false, idx: -1},
		{addr: 0x20004448, size: 4, signed: false, idx: -1},
		{addr: 0x20004458, size: 4, signed: false, idx: -1},
		{addr: 0x2000445c, size: 4, signed: false, idx: -1},
		{addr: 0x20004460, size: 4, signed: false, idx: -1},
		{addr: 0x20004464, size: 4, signed: false, idx: -1},
		{addr: 0x200044b0, size: 4, signed: false, idx: -1},
		{addr: 0x200044b4, size: 4, signed: false, idx: -1},
		{addr: 0x200044b8, size: 4, signed: false, idx: -1},
		{addr: 0x2000455c, size: 4, signed: false, idx: -1},
		{addr: 0x20004560, size: 4, signed: false, idx: -1},
		{addr: 0x20004564, size: 4, signed: false, idx: -1},
		{addr: 0x20004568, size: 4, signed: false, idx: -1},
		{addr: 0x2000456c, size: 4, signed: false, idx: -1},
		{addr: 0x20004570, size: 4, signed: false, idx: -1},
		{addr: 0x20004574, size: 4, signed: false, idx: -1},
		{addr: 0x20004578, size: 4, signed: false, idx: -1},
		{addr: 0x2000457c, size: 4, signed: false, idx: -1},
		{addr: 0x20004580, size: 4, signed: false, idx: -1},
		{addr: 0x20004584, size: 4, signed: false, idx: -1},
		{addr: 0x20004588, size: 4, signed: false, idx: -1},
		{addr: 0x2000458c, size: 4, signed: false, idx: -1},
		{addr: 0x20004590, size: 4, signed: false, idx: -1},
		{addr: 0x20004594, size: 4, signed: false, idx: -1},
		{addr: 0x20004598, size: 4, signed: false, idx: -1},
		{addr: 0x2000459c, size: 4, signed: false, idx: -1},
		{addr: 0x200045a0, size: 4, signed: false, idx: -1},
		{addr: 0x200045a4, size: 4, signed: false, idx: -1},
		{addr: 0x200045a8, size: 4, signed: false, idx: -1},
		{addr: 0x200045ac, size: 4, signed: false, idx: -1},
		{addr: 0x200045b0, size: 4, signed: false, idx: -1},
		{addr: 0x200045b4, size: 4, signed: false, idx: -1},
		{addr: 0x200045b8, size: 4, signed: false, idx: -1},
		{addr: 0x200045bc, size: 4, signed: false, idx: -1},
		{addr: 0x200045c0, size: 4, signed: false, idx: -1},
		{addr: 0x200045c4, size: 4, signed: false, idx: -1},
		{addr: 0x200045c8, size: 4, signed: false, idx: -1},
		{addr: 0x200045cc, size: 4, signed: false, idx: -1},
		{addr: 0x200045d0, size: 4, signed: false, idx: -1},
		{addr: 0x200045d4, size: 4, signed: false, idx: -1},
		{addr: 0x200045d8, size: 4, signed: false, idx: -1},
		{addr: 0x200045dc, size: 4, signed: false, idx: -1},
		{addr: 0x200045e0, size: 4, signed: false, idx: -1},
		{addr: 0x200045e4, size: 4, signed: false, idx: -1},
		{addr: 0x200045e8, size: 4, signed: false, idx: -1},
		{addr: 0x200045ec, size: 4, signed: false, idx: -1},
		{addr: 0x200045f0, size: 4, signed: false, idx: -1},
		{addr: 0x200045f4, size: 4, signed: false, idx: -1},
		{addr: 0x200045f8, size: 4, signed: false, idx: -1},
		{addr: 0x200045fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004628, size: 4, signed: false, idx: -1},
		{addr: 0x2000462c, size: 4, signed: false, idx: -1},
		{addr: 0x20004630, size: 4, signed: false, idx: -1},
		{addr: 0x20004634, size: 4, signed: false, idx: -1},
		{addr: 0x20004638, size: 4, signed: false, idx: -1},
		{addr: 0x20004800, size: 4, signed: false, idx: -1},
	},
})

var mprIn = []string{"M.active", "M.inCombat", "M.morale", "M.maxHP", "M.HP", "M.maxSP", "M.f77", "M.f87", "M.cond0", "M.cond1", "M.cond2", "M.cond3", "M.cond4", "M.cond5", "M.cond6", "M.cond7", "M.cond8", "M.cond9", "M.cond10", "M.cond11", "M.cond12", "M.cond13", "M.cond14", "M.cond15", "M.cond16", "M.cond17", "M.cond18", "M.cond19", "M.buff5.power", "M.buff5.mult", "M.buff5.dur", "M.buff6.power", "M.buff6.mult", "M.buff6.dur", "M.buff7.power", "M.buff7.mult", "M.buff7.dur", "M.mainCond", "M.attackPenalty", "M.eff0.active", "M.eff0.type", "M.eff0.power", "M.eff0.mult", "M.eff5.active", "M.eff5.type", "M.eff5.power", "M.eff5.mult", "M.eff11.active", "M.eff11.type", "M.eff11.power", "M.eff11.mult", "M.disabledVisual", "M.target0", "M.target1", "M.target2", "M.target3", "M.target4", "M.target5", "M.target6", "M.target7", "M.f2da", "M.action", "M.subaction", "M.removed", "M.f344", "M.f37a", "M.f37c", "M1.morale", "M1.mainCond", "M1.removed", "M2.morale", "M2.mainCond", "M2.removed", "T.kind", "T.fcc", "T.fd0", "T.attr0", "T.attr1", "T.attr2", "T.attr3", "T.attr4", "T.fe0", "T.fe1", "T.special", "T.atk0.on", "T.atk0.range", "T.atk1.on", "T.atk1.range", "T.atk2.on", "T.atk2.range", "T.spell0", "T.spell1", "T.spell2", "T.spell3", "T.spell4", "T.spell5", "T.spell6", "T.spell7", "T.spell8", "T.spell9", "T.species", "T.f1b9", "T1.species", "vis.x", "vis.y", "vis.z", "actor.fa7", "C.f4", "C.f15", "C.f152", "C.sp0.active", "C.sp0.code", "C.sp0.value", "C.sp1.active", "C.sp1.code", "C.sp1.value", "C.sp2.active", "C.sp2.code", "C.sp2.value", "C.sp3.active", "C.sp3.code", "C.sp3.value", "C.sp4.active", "C.sp4.code", "C.sp4.value", "C.sp5.active", "C.sp5.code", "C.sp5.value", "C.sp6.active", "C.sp6.code", "C.sp6.value", "C.sp7.active", "C.sp7.code", "C.sp7.value", "C.sp8.active", "C.sp8.code", "C.sp8.value", "C.aura0.active", "C.aura0.code", "C.aura0.power", "C.aura0.mult", "C.aura1.active", "C.aura1.code", "C.aura1.power", "C.aura1.mult", "C.aura2.active", "C.aura2.code", "C.aura2.power", "C.aura2.mult", "C.aura3.active", "C.aura3.code", "C.aura3.power", "C.aura3.mult", "C.aura4.active", "C.aura4.code", "C.aura4.power", "C.aura4.mult", "C.aura5.active", "C.aura5.code", "C.aura5.power", "C.aura5.mult", "grp.living", "grp.f2c", "G.f918", "g683f94", "g683fa5", "g68510c", "g652da4", "gate", "ui.flags", "sp0.f48", "sp0.f126", "sp0.f12f", "sp0.f133", "sp0.type", "sp0.f13f", "sp1.f48", "sp1.f126", "sp1.f12f", "sp1.f133", "sp1.type", "sp1.f13f", "sp2.f48", "sp2.f126", "sp2.f12f", "sp2.f133", "sp2.type", "sp2.f13f", "sp3.f48", "sp3.f126", "sp3.f12f", "sp3.f133", "sp3.type", "sp3.f13f", "sp4.f48", "sp4.f126", "sp4.f12f", "sp4.f133", "sp4.type", "sp4.f13f", "sp5.f48", "sp5.f126", "sp5.f12f", "sp5.f133", "sp5.type", "sp5.f13f", "sp6.f48", "sp6.f126", "sp6.f12f", "sp6.f133", "sp6.type", "sp6.f13f", "sp7.f48", "sp7.f126", "sp7.f12f", "sp7.f133", "sp7.type", "sp7.f13f", "r4c6030_0", "r4c6030_1", "r534d50", "r534a40_0", "r534a40_1", "r534a40_2", "r534a40_3", "r534a40_4", "r4d9080_0", "r4d9080_1", "r4d9080_2", "r4d9080_3", "r4d9080_4", "r4d9080_5", "r4d9080_6", "r4d9080_7", "r4d9080_8", "r4d9080_9", "r4d9080_10", "r4d9080_11", "r4d9080_12", "r4d9080_13", "r4d9080_14", "r4d9080_15", "r4d9080_16", "r4d9080_17", "r4d9080_18", "r4d9080_19", "r4d9080_20", "r4d9080_21", "r4d9080_22", "r4d9080_23", "r4d9080_24", "r4d9080_25", "r4d9080_26", "r4d9080_27", "r4d9080_28", "r4d9080_29", "r4d9080_30", "r4d9080_31", "r4d9080_32", "r4d9080_33", "r4d9080_34", "r4d9080_35", "r4d9080_36", "r4d9080_37", "r4d9080_38", "r4d9080_39", "r4d9080_40", "w4c60d0_l95_0", "w4c60d0_l95_1", "w4c60d0_l95_2", "w4c60d0_l95_3", "r4c60d0_0", "r4c60d0_1", "r4c60d0_2", "r4c60d0_3", "w4c6070_l95_0", "w4c6070_l95_1", "w4c6070_l95_2", "w4c6070_l95_3", "r4c6070_0", "r4c6070_1", "r4c6070_2", "r4c6070_3", "w51b320_x_0", "w51b320_x_1", "w51b320_x_2", "w51b320_x_3", "w51b320_y_0", "w51b320_y_1", "w51b320_y_2", "w51b320_y_3", "w51b320_z_0", "w51b320_z_1", "w51b320_z_2", "w51b320_z_3", "w51ac30_t0_0", "w51ac30_t0_1", "w51ac30_t0_2", "w51ac30_t0_3", "w51ac30_t1_0", "w51ac30_t1_1", "w51ac30_t1_2", "w51ac30_t1_3", "w51ac30_t2_0", "w51ac30_t2_1", "w51ac30_t2_2", "w51ac30_t2_3", "w51ac30_t3_0", "w51ac30_t3_1", "w51ac30_t3_2", "w51ac30_t3_3", "w51ac30_t4_0", "w51ac30_t4_1", "w51ac30_t4_2", "w51ac30_t4_3", "w51ac30_t5_0", "w51ac30_t5_1", "w51ac30_t5_2", "w51ac30_t5_3", "w51ac30_t6_0", "w51ac30_t6_1", "w51ac30_t6_2", "w51ac30_t6_3", "w51ac30_t7_0", "w51ac30_t7_1", "w51ac30_t7_2", "w51ac30_t7_3", "r53c630", "w53c630_0", "w53c630_1", "w53c630_2", "r519f80", "w519f80_t0", "w519f80_t1", "w519f80_t2", "w519f80_t3", "w519f80_t4", "w519f80_t5", "w519f80_t6", "w519f80_t7", "r4ec1e0", "w5323f0_2e1", "w5323f0_2e5", "w5323f0_t0", "w5323f0_t1", "w5323f0_t2", "w5323f0_t3", "w5323f0_t4", "w5323f0_t5", "w5323f0_t6", "w5323f0_t7", "w5323f0_c5", "w5323f0_c9", "w5323f0_c11", "r5323f0", "r53fea0", "w53fea0_t0", "w53fea0_t1", "w53fea0_t2", "w53fea0_t3", "r545cf0", "r5327e0", "r536190", "r500330_0", "r500330_1", "r500330_2", "r5326f0_0", "r5326f0_1", "w5326f0_t0_0", "w5326f0_t0_1", "w5326f0_t1_0", "w5326f0_t1_1", "w5326f0_t2_0", "w5326f0_t2_1", "w5326f0_t3_0", "w5326f0_t3_1", "w5326f0_t4_0", "w5326f0_t4_1", "w5326f0_t5_0", "w5326f0_t5_1", "w5326f0_t6_0", "w5326f0_t6_1", "w5326f0_t7_0", "w5326f0_t7_1", "w533320_n_0", "w533320_n_1", "w533320_n_2", "w533320_n_3", "w533320_n_4", "w533320_n_5", "w533320_n_6", "w533320_n_7", "w533320_n_8", "w533320_n_9", "w533320_n_10", "w533320_n_11", "w533320_n_12", "w533320_n_13", "w533320_n_14", "w533320_n_15", "w533320_n_16", "w533320_n_17", "w533320_n_18", "w533320_n_19", "w533320_n_20", "w533320_n_21", "w533320_n_22", "w533320_n_23", "w533320_n_24", "w533320_n_25", "w533320_n_26", "w533320_n_27", "w533320_n_28", "w533320_n_29", "w533320_n_30", "w533320_n_31", "w533320_n_32", "w533320_n_33", "w533320_n_34", "w533320_n_35", "w533320_n_36", "w533320_n_37", "w533320_n_38", "w533320_n_39", "w4fd030_mons_0", "w4fd030_mons_1", "w4fd030_chars_0", "w4fd030_chars_1", "w4fd030_id0_0", "w4fd030_id0_1", "w4fd030_id1_0", "w4fd030_id1_1", "w4fd030_id2_0", "w4fd030_id2_1", "w531540_2e1_0", "w531540_2e1_1", "w531540_2e1_2", "w531540_2e1_3", "w531540_2e1_4", "w531540_2e1_5", "w531540_2e1_6", "w531540_2e1_7", "w531540_2e1_8", "w531540_2e1_9", "w531540_2e1_10", "w531540_2e5_0", "w531540_2e5_1", "w531540_2e5_2", "w531540_2e5_3", "w531540_2e5_4", "w531540_2e5_5", "w531540_2e5_6", "w531540_2e5_7", "w531540_2e5_8", "w531540_2e5_9", "w531540_2e5_10", "w531540_t0_0", "w531540_t0_1", "w531540_t0_2", "w531540_t0_3", "w531540_t0_4", "w531540_t0_5", "w531540_t0_6", "w531540_t0_7", "w531540_t0_8", "w531540_t0_9", "w531540_t0_10", "w531540_t1_0", "w531540_t1_1", "w531540_t1_2", "w531540_t1_3", "w531540_t1_4", "w531540_t1_5", "w531540_t1_6", "w531540_t1_7", "w531540_t1_8", "w531540_t1_9", "w531540_t1_10", "w531540_t2_0", "w531540_t2_1", "w531540_t2_2", "w531540_t2_3", "w531540_t2_4", "w531540_t2_5", "w531540_t2_6", "w531540_t2_7", "w531540_t2_8", "w531540_t2_9", "w531540_t2_10", "w531540_t3_0", "w531540_t3_1", "w531540_t3_2", "w531540_t3_3", "w531540_t3_4", "w531540_t3_5", "w531540_t3_6", "w531540_t3_7", "w531540_t3_8", "w531540_t3_9", "w531540_t3_10", "w531540_t4_0", "w531540_t4_1", "w531540_t4_2", "w531540_t4_3", "w531540_t4_4", "w531540_t4_5", "w531540_t4_6", "w531540_t4_7", "w531540_t4_8", "w531540_t4_9", "w531540_t4_10", "w531540_t5_0", "w531540_t5_1", "w531540_t5_2", "w531540_t5_3", "w531540_t5_4", "w531540_t5_5", "w531540_t5_6", "w531540_t5_7", "w531540_t5_8", "w531540_t5_9", "w531540_t5_10", "w531540_t6_0", "w531540_t6_1", "w531540_t6_2", "w531540_t6_3", "w531540_t6_4", "w531540_t6_5", "w531540_t6_6", "w531540_t6_7", "w531540_t6_8", "w531540_t6_9", "w531540_t6_10", "w531540_t7_0", "w531540_t7_1", "w531540_t7_2", "w531540_t7_3", "w531540_t7_4", "w531540_t7_5", "w531540_t7_6", "w531540_t7_7", "w531540_t7_8", "w531540_t7_9", "w531540_t7_10", "w531540_c4_0", "w531540_c4_1", "w531540_c4_2", "w531540_c4_3", "w531540_c4_4", "w531540_c4_5", "w531540_c4_6", "w531540_c4_7", "w531540_c4_8", "w531540_c4_9", "w531540_c4_10"}

var mprOut = []string{"M.action", "M.subaction", "M.power", "M.f2e0", "M.target0", "M.target1", "M.target2", "M.target3", "M.target4", "M.target5", "M.target6", "M.target7", "M.cond14", "M.mainCond", "M.disabledVisual", "M.mod00", "M.mod01", "M.mod02", "M.mod03", "M.mod04", "M.mod05", "M.mod06", "M.mod07", "M.mod08", "M.mod09", "M.mod0a", "M.mod0b", "M.mod0c", "M.mod0d", "M.mod0e", "M.mod0f", "M.mod10", "M.mod11", "M.mod12", "M.mod13", "M.mod14", "M.mod15", "M.mod16", "M.mod17", "M.mod18", "M.mod19", "M.mod1a", "M.mod1b", "M.mod1c", "M.mod1d", "M.mod1e", "M.mod1f", "M.mod20", "M.mod21", "M.mod22", "M.mod23", "M.mod24", "M.mod25", "M.mod26", "M.mod27", "M.mod28", "M.mod29", "M.mod2a", "M.mod2b", "M.mod2c", "M.mod2d", "M.mod2e", "M.mod2f", "M.mod30", "M.mod31", "M.mod32", "M.mod33", "M.mod34", "M.mod35", "M.mod36", "M.mod37", "M.mod38", "M.mod39", "M.mod3a", "M.mod3b", "M.mod3c", "M.mod3d", "M.mod3e", "M.mod3f", "M.mod40", "M.mod41", "M.mod42", "M.mod43", "M.mod44", "M.mod45", "M.mod46", "M.mod47", "M.mod48", "M.mod49", "M.mod4a", "M.mod4b", "M.mod4c", "M.mod4d", "M.mod4e", "M.mod4f", "M.mod50", "M.mod51", "M.mod52", "M.mod53", "M.mod54", "M.mod55", "M.mod56", "M.mod57", "M.mod58", "M.mod59", "M.mod5a", "M.mod5b", "M.mod5c", "M.mod5d", "M.mod5e", "M.mod5f", "M.mod60", "M.mod61", "M.mod62", "M.mod63", "M.mod64", "M.mod65", "M.mod66", "M.attr0", "M.attr1", "M.attr2", "M.attr3", "M.attr4", "M.hpRecovery", "M.spRecovery", "C.f4", "C.f5", "C.f9", "C.f11", "C.f15", "C.f145", "C.f14b", "C.f14c", "C.f152", "grp.living", "ui.flags", "G.fa55", "G.f7b0", "s4efd30_calls", "s48f650_calls", "s58aad0_a2_0", "s58aad0_a2_1", "s58aad0_a2_2", "s58aad0_a2_3", "s58aad0_a2_4", "s58aad0_a2_5", "s58aad0_a2_6", "s58aad0_a2_7", "s58aad0_a3_0", "s58aad0_a3_1", "s58aad0_a3_2", "s58aad0_a3_3", "s58aad0_a3_4", "s58aad0_a3_5", "s58aad0_a3_6", "s58aad0_a3_7", "s58aad0_calls", "s5182e0_a2_0", "s5182e0_a3_0", "s5182e0_a4_0", "s5182e0_calls", "s590b40_a2_0", "s590b40_calls", "s562a50_a1_0", "s562a50_calls", "s4c6200_a2_0", "s4c6200_calls", "s4e4db0_a2_0", "s4e4db0_a2_1", "s4e4db0_a2_2", "s4e4db0_a3_0", "s4e4db0_a3_1", "s4e4db0_a3_2", "s4e4db0_calls", "s4ec900_a2_0", "s4ec900_a2_1", "s4ec900_calls", "s4ecb40_calls", "s4eb8c0_calls", "s5471d0_calls", "s4acd80_a2_0", "s4acd80_calls", "s4c5a00_a2_0", "s4c5a00_a2_1", "s4c5a00_calls", "s4c5aa0_a2_0", "s4c5aa0_calls", "s4c6030_calls", "s534d50_calls", "s534a40_a3_0", "s534a40_a3_1", "s534a40_a3_2", "s534a40_a3_3", "s534a40_a3_4", "s534a40_calls", "s4d9080_calls", "s4c60d0_a2_0", "s4c60d0_a2_1", "s4c60d0_a2_2", "s4c60d0_a2_3", "s4c60d0_a3_0", "s4c60d0_a3_1", "s4c60d0_a3_2", "s4c60d0_a3_3", "s4c60d0_a4_0", "s4c60d0_a4_1", "s4c60d0_a4_2", "s4c60d0_a4_3", "s4c60d0_a5_0", "s4c60d0_a5_1", "s4c60d0_a5_2", "s4c60d0_a5_3", "s4c60d0_a6_0", "s4c60d0_a6_1", "s4c60d0_a6_2", "s4c60d0_a6_3", "s4c60d0_a7_0", "s4c60d0_a7_1", "s4c60d0_a7_2", "s4c60d0_a7_3", "s4c60d0_a8_0", "s4c60d0_a8_1", "s4c60d0_a8_2", "s4c60d0_a8_3", "s4c60d0_calls", "s4c6070_a2_0", "s4c6070_a2_1", "s4c6070_a2_2", "s4c6070_a2_3", "s4c6070_a3_0", "s4c6070_a3_1", "s4c6070_a3_2", "s4c6070_a3_3", "s4c6070_a4_0", "s4c6070_a4_1", "s4c6070_a4_2", "s4c6070_a4_3", "s4c6070_a5_0", "s4c6070_a5_1", "s4c6070_a5_2", "s4c6070_a5_3", "s4c6070_a6_0", "s4c6070_a6_1", "s4c6070_a6_2", "s4c6070_a6_3", "s4c6070_a7_0", "s4c6070_a7_1", "s4c6070_a7_2", "s4c6070_a7_3", "s4c6070_calls", "s51b320_a2_0", "s51b320_a2_1", "s51b320_a2_2", "s51b320_a2_3", "s51b320_calls", "s51ac30_a3_0", "s51ac30_a3_1", "s51ac30_a3_2", "s51ac30_a3_3", "s51ac30_calls", "s53c630_a2_0", "s53c630_calls", "s519f80_a3_0", "s519f80_calls", "s4ec1e0_calls", "s5323f0_a2_0", "s5323f0_a3_0", "s5323f0_a4_0", "s5323f0_calls", "s53fea0_calls", "s545cf0_calls", "s5327e0_a2_0", "s5327e0_calls", "s536190_calls", "s500330_a3_0", "s500330_a3_1", "s500330_a3_2", "s500330_calls", "s5326f0_a2_0", "s5326f0_a2_1", "s5326f0_calls", "s533320_a2_0", "s533320_a2_1", "s533320_a2_2", "s533320_a2_3", "s533320_a2_4", "s533320_a2_5", "s533320_a2_6", "s533320_a2_7", "s533320_a2_8", "s533320_a2_9", "s533320_a2_10", "s533320_a2_11", "s533320_a2_12", "s533320_a2_13", "s533320_a2_14", "s533320_a2_15", "s533320_a2_16", "s533320_a2_17", "s533320_a2_18", "s533320_a2_19", "s533320_a2_20", "s533320_a2_21", "s533320_a2_22", "s533320_a2_23", "s533320_a2_24", "s533320_a2_25", "s533320_a2_26", "s533320_a2_27", "s533320_a2_28", "s533320_a2_29", "s533320_a2_30", "s533320_a2_31", "s533320_a2_32", "s533320_a2_33", "s533320_a2_34", "s533320_a2_35", "s533320_a2_36", "s533320_a2_37", "s533320_a2_38", "s533320_a2_39", "s533320_calls", "s4fd030_a1_0", "s4fd030_a1_1", "s4fd030_a2_0", "s4fd030_a2_1", "s4fd030_calls", "s531540_calls"}

var mprStubs = map[uint32]*mcaStub{
	0x4efd30: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20004000, wr: []mcaWrite{}},
	0x48f650: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20004004, wr: []mcaWrite{}},
	0x58aad0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20004008, 8}, {3, 4, 0x20004028, 8}}, cnt: 0x20004048, wr: []mcaWrite{}},
	0x5182e0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x2000404c, 1}, {3, 4, 0x20004050, 1}, {4, 4, 0x20004054, 1}}, cnt: 0x20004058, wr: []mcaWrite{}},
	0x590b40: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x2000405c, 1}}, cnt: 0x20004060, wr: []mcaWrite{}},
	0x562a50: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004064, 1}}, cnt: 0x20004068, wr: []mcaWrite{}},
	0x4c6200: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x2000406c, 1}}, cnt: 0x20004070, wr: []mcaWrite{}},
	0x4e4db0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20004074, 3}, {3, 4, 0x20004080, 3}}, cnt: 0x2000408c, wr: []mcaWrite{}},
	0x4ec900: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20004090, 2}}, cnt: 0x20004098, wr: []mcaWrite{}},
	0x4ecb40: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x2000409c, wr: []mcaWrite{}},
	0x4eb8c0: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x200040a0, wr: []mcaWrite{}},
	0x5471d0: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x200040a4, wr: []mcaWrite{}},
	0x4acd80: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200040a8, 1}}, cnt: 0x200040ac, wr: []mcaWrite{}},
	0x4c5a00: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200040b0, 2}}, cnt: 0x200040b8, wr: []mcaWrite{}},
	0x4c5aa0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200040bc, 1}}, cnt: 0x200040c0, wr: []mcaWrite{}},
	0x4c6030: {ret: 0x200040c4, retN: 2, rec: [][4]uint32{}, cnt: 0x200040cc, wr: []mcaWrite{}},
	0x534d50: {ret: 0x200040d0, retN: 0, rec: [][4]uint32{}, cnt: 0x200040d4, wr: []mcaWrite{}},
	0x534a40: {ret: 0x200040d8, retN: 5, rec: [][4]uint32{{3, 4, 0x200040ec, 5}}, cnt: 0x20004100, wr: []mcaWrite{}},
	0x4d9080: {ret: 0x20004104, retN: 41, rec: [][4]uint32{}, cnt: 0x200041a8, wr: []mcaWrite{}},
	0x4c60d0: {ret: 0x200041b0, retN: 4, rec: [][4]uint32{{2, 4, 0x200041c0, 4}, {3, 4, 0x200041d0, 4}, {4, 4, 0x200041e0, 4}, {5, 4, 0x200041f0, 4}, {6, 4, 0x20004200, 4}, {7, 4, 0x20004210, 4}, {8, 4, 0x20004220, 4}}, cnt: 0x20004230, wr: []mcaWrite{{9, 0x0, 1, 0x200041ac, 4}}},
	0x4c6070: {ret: 0x20004238, retN: 4, rec: [][4]uint32{{2, 4, 0x20004248, 4}, {3, 4, 0x20004258, 4}, {4, 4, 0x20004268, 4}, {5, 4, 0x20004278, 4}, {6, 4, 0x20004288, 4}, {7, 4, 0x20004298, 4}}, cnt: 0x200042a8, wr: []mcaWrite{{8, 0x0, 1, 0x20004234, 4}}},
	0x51b320: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200042dc, 4}}, cnt: 0x200042ec, wr: []mcaWrite{{3, 0x0, 4, 0x200042ac, 4}, {3, 0x4, 4, 0x200042bc, 4}, {3, 0x8, 4, 0x200042cc, 4}}},
	0x51ac30: {ret: 0x0, retN: 0, rec: [][4]uint32{{3, 4, 0x20004370, 4}}, cnt: 0x20004380, wr: []mcaWrite{{2, 0x0, 4, 0x200042f0, 4}, {2, 0x4, 4, 0x20004300, 4}, {2, 0x8, 4, 0x20004310, 4}, {2, 0xc, 4, 0x20004320, 4}, {2, 0x10, 4, 0x20004330, 4}, {2, 0x14, 4, 0x20004340, 4}, {2, 0x18, 4, 0x20004350, 4}, {2, 0x1c, 4, 0x20004360, 4}}},
	0x53c630: {ret: 0x20004384, retN: 0, rec: [][4]uint32{{2, 4, 0x20004394, 1}}, cnt: 0x20004398, wr: []mcaWrite{{1, 0x10, 4, 0x20004388, 0}, {1, 0x14, 4, 0x2000438c, 0}, {1, 0x18, 4, 0x20004390, 0}}},
	0x519f80: {ret: 0x2000439c, retN: 0, rec: [][4]uint32{{3, 4, 0x200043c0, 1}}, cnt: 0x200043c4, wr: []mcaWrite{{4, 0x0, 4, 0x200043a0, 0}, {4, 0x4, 4, 0x200043a4, 0}, {4, 0x8, 4, 0x200043a8, 0}, {4, 0xc, 4, 0x200043ac, 0}, {4, 0x10, 4, 0x200043b0, 0}, {4, 0x14, 4, 0x200043b4, 0}, {4, 0x18, 4, 0x200043b8, 0}, {4, 0x1c, 4, 0x200043bc, 0}}},
	0x4ec1e0: {ret: 0x200043c8, retN: 0, rec: [][4]uint32{}, cnt: 0x200043cc, wr: []mcaWrite{}},
	0x5323f0: {ret: 0x20004404, retN: 0, rec: [][4]uint32{{2, 4, 0x20004408, 1}, {3, 4, 0x2000440c, 1}, {4, 4, 0x20004410, 1}}, cnt: 0x20004414, wr: []mcaWrite{{1, 0x2e1, 4, 0x200043d0, 0}, {1, 0x2e5, 4, 0x200043d4, 0}, {1, 0x2ba, 4, 0x200043d8, 0}, {1, 0x2be, 4, 0x200043dc, 0}, {1, 0x2c2, 4, 0x200043e0, 0}, {1, 0x2c6, 4, 0x200043e4, 0}, {1, 0x2ca, 4, 0x200043e8, 0}, {1, 0x2ce, 4, 0x200043ec, 0}, {1, 0x2d2, 4, 0x200043f0, 0}, {1, 0x2d6, 4, 0x200043f4, 0}, {0, 0x20002005, 4, 0x200043f8, 0}, {0, 0x20002009, 4, 0x200043fc, 0}, {0, 0x2000200b, 4, 0x20004400, 0}}},
	0x53fea0: {ret: 0x20004418, retN: 0, rec: [][4]uint32{}, cnt: 0x2000442c, wr: []mcaWrite{{1, 0x2ba, 4, 0x2000441c, 0}, {1, 0x2be, 4, 0x20004420, 0}, {1, 0x2c2, 4, 0x20004424, 0}, {1, 0x2c6, 4, 0x20004428, 0}}},
	0x545cf0: {ret: 0x20004430, retN: 0, rec: [][4]uint32{}, cnt: 0x20004434, wr: []mcaWrite{}},
	0x5327e0: {ret: 0x20004438, retN: 0, rec: [][4]uint32{{2, 4, 0x2000443c, 1}}, cnt: 0x20004440, wr: []mcaWrite{}},
	0x536190: {ret: 0x20004444, retN: 0, rec: [][4]uint32{}, cnt: 0x20004448, wr: []mcaWrite{}},
	0x500330: {ret: 0x2000444c, retN: 3, rec: [][4]uint32{{3, 4, 0x20004458, 3}}, cnt: 0x20004464, wr: []mcaWrite{}},
	0x5326f0: {ret: 0x20004468, retN: 2, rec: [][4]uint32{{2, 4, 0x200044b0, 2}}, cnt: 0x200044b8, wr: []mcaWrite{{1, 0x2ba, 4, 0x20004470, 2}, {1, 0x2be, 4, 0x20004478, 2}, {1, 0x2c2, 4, 0x20004480, 2}, {1, 0x2c6, 4, 0x20004488, 2}, {1, 0x2ca, 4, 0x20004490, 2}, {1, 0x2ce, 4, 0x20004498, 2}, {1, 0x2d2, 4, 0x200044a0, 2}, {1, 0x2d6, 4, 0x200044a8, 2}}},
	0x533320: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x2000455c, 40}}, cnt: 0x200045fc, wr: []mcaWrite{{3, 0x4, 4, 0x200044bc, 40}}},
	0x4fd030: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004628, 2}, {2, 4, 0x20004630, 2}}, cnt: 0x20004638, wr: []mcaWrite{{5, 0x4, 4, 0x20004600, 2}, {6, 0x4, 4, 0x20004608, 2}, {0, 0x50000000, 4, 0x20004610, 2}, {0, 0x50000004, 4, 0x20004618, 2}, {0, 0x50000008, 4, 0x20004620, 2}}},
	0x531540: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20004800, wr: []mcaWrite{{1, 0x2e1, 4, 0x2000463c, 11}, {1, 0x2e5, 4, 0x20004668, 11}, {1, 0x2ba, 4, 0x20004694, 11}, {1, 0x2be, 4, 0x200046c0, 11}, {1, 0x2c2, 4, 0x200046ec, 11}, {1, 0x2c6, 4, 0x20004718, 11}, {1, 0x2ca, 4, 0x20004744, 11}, {1, 0x2ce, 4, 0x20004770, 11}, {1, 0x2d2, 4, 0x2000479c, 11}, {1, 0x2d6, 4, 0x200047c8, 11}, {0, 0x20002004, 1, 0x200047f4, 11}}},
}

// mprPark4ec1e0 is the first boundary row of re/emu/cases/004ec1e0.json (status action override).
var mprPark4ec1e0 = [1282]int64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 100, 100, 100, 100, 100, 100, 100, 100, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 50, 50, 50, 50, 50, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// mprPark53fea0 is the first boundary row of re/emu/cases/0053fea0.json (monster attack swings).
var mprPark53fea0 = [407]int64{0, 0, 537108312, -1, 0, -1, 0, 0, 0, 0, 4, 1, 1, 536936448, 0, 1, 16, 0, 1, 1, 1, -1, -1, 1, 0, 0, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 0, 0, 0, 0, 100, 20, 0, 1, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 5, 1, 10, 10, 10, 10, 10, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 65535, 4294967295, 3, 255, 0, 255, 0, 255, 0, 255, 0, 1, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 1, 1, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 101, 102, 103, 536936448, 201, 202, 203, 536936448, 77, 301, 302, 303, 304, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1, 1, 0, 1, 1, 1, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 200, 0, 1084227584, 0, 1092616192, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9, 0, 2, 1, 3, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1148846080, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// mprPark545cf0 is the first boundary row of re/emu/cases/00545cf0.json (monster intercept available).
var mprPark545cf0 = [48]int64{0, 0, 1, 39, 100, 0, 0, 11, 0, 10, 2, 1, 39, 100, 1, 0, 11, 2, 0, 0, 0, 1, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// mprPark5327e0 is the first boundary row of re/emu/cases/005327e0.json (monster spell target ok).
var mprPark5327e0 = [267]int64{0, 1, 5, 0, 1, 1, 0, 1, 3, 3, 0, 1, 0, 5, 0, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1, 0, 1, 0, 0, 1, 0, 1, 0, 28, 0, 5, 5, 59, 0, 0, 0, 0, 1, 46, 0, 0, 1, 255, 0, 0, 1, 1, 1, 1, 1, 60, 0, 0, 0, 1, 0, 1, 1, 0, 0, 0, 0, 255, 1, 0, 0, 0, 1, 0, 1, 46, 0, 1, 1, 2, 2, 2, 2, 0, 0, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 1, 1, 1, 0, 1, 0, 1, 0, 1, 2, 2, 0, 0, 1, 1, 2, 0, 2, 1, 0, 0, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 1, 0, 1, 1, 0, 0, 0, 1, 1, 1, 0, 0, 1, 0, 1, 0, 1, 1, 1, 0, 1, 0, 1, 1, 0, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 1065353216, 0, 0, 0, 0, 0, 0, 0, 1, 5, 1065353216, 0, 2, 1, 3, -1, -1, 0, 1, 0, 7, 0, 123, 456, 0, 1, 1, 1, 10, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 0, 0, 1, 0, 8, 0, 8, 1065353216, 0, 1073741824, 1, 256}

// mprPark536190 is the first boundary row of re/emu/cases/00536190.json (combat target valid).
var mprPark536190 = [29]int64{0, 1, 0, 1, 1, 0, 1, 0, 10, 20, 0, 2, 10, 1, 0, 1, 2, 11, 1, 0, 1, 2, 20, 1, 1, 2, 21, 1, 1}

// mprPark531540 is the first boundary row of re/emu/cases/00531540.json (monster action choice).
var mprPark531540 = [208]int64{0, 0, 0, 0, 0, 1148846080, 0, 0, 0, 0, 0, 0, 1153138688, 0, 0, 0, 1161527296, 0, 0, 0, 1, 1, 17, 34, 51, 68, 85, 0, 0, 0, 3, 3, 1, 1, 1, 0, 1, 10, 0, 1, 1, 100, 100, 100, 10, 0, 0, 0, 0, 0, 0, 17, 34, 51, 68, 85, 102, 119, 0, 0, 4294967295, 0, 0, 1, 1, 0, 50, 0, 1, 1, 1, 0, 50, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 5, 0, 1, 7, 0, 0, 9, 4, 1, 40, 40, 7, 24, 1, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 3, 1, 0, 4, 0, 2, 6, 1, 0, 1, 3, 1, 1, 1, 4, 1, 0, 1, 5, 1, 1, 1, 6, 1, 0, 1, 7, 1, 1, 1, 3, 1, 0, 1, 4, 1, 1, 1, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 5, 6, 7, 8, 9, 10, 11, 12, 60, 1, 0, 119, 1, 2, 3, 4, 5, 6, 7, 8, 165, 182, 199, 7, 1}
