package calc

// ---- 0x4ea310: combat exit and XP distribution ----
//
// Runs once when a battle ends (arg0's low byte: nonzero skips the defeat
// messages, the kill rewards and the 0x482990 tail). It cancels any pending
// special mode, settles every listed monster that died this battle (0x4e5c00:
// 0x5248d0, then 0x4e4280 removal and 0x4e46f0 kill rewards unless arg0), ends
// the active groups (0x50fad0), splits the party's battle XP over the eligible
// members (0x4eee80 count, the per-map kill counter at 0x686b75 + 33*map for
// maps below 47, 0x4eef10 award of quotient + bonus) and promotes every seen
// species from 1 to 2. The closing block resets the formation copy (0x554a60),
// the level-up icons (0x59bb70 over 0x4ef3c0), the widget rows (0x561fb0), the
// animations (0x4531a0), the monsters' transient flag 0x255 (0x4e5ea0), and in
// map mode 7 the party status icons (0x4ea1f0).
//
// It runs over the same memory image as the emulator case (mstWorld, built by
// gen from re/emu/cases/004ea310.json's description). Registered-out callees are
// address stubs replayed from case inputs in both runs. The D4 callees 0x4eee80,
// 0x4eef10, 0x4ef3c0, 0x4e79a0, 0x4e4280 and 0x4e46f0 are address stubs too: Emu
// replays them, F calls their Go nodes on their parked worlds (cxdPark*) with the
// values this world hands them and writes their outputs back. The other D4
// callees (0x5244a0, 0x5246c0, 0x524540, 0x552530, 0x53cd60, 0x5248d0, 0x4e4500,
// 0x4ee9d0) have no output this node reads; both runs replay them.

const (
	cxdMonCap    = 3      // fixture cap: three monster records in 0x683fad
	cxdGroupCap  = 2      // fixture cap: two group records in 0x683fb1
	cxdIdCap     = 2      // fixture cap: two monster ids per group list
	cxdAnimCap   = 2      // fixture cap: two animation records at [0x659b3c]
	cxdCharSize  = 0x1862 // character record stride (0x4ea570)
	cxdSlotSize  = 0x106  // party slot stride (0x4ea56a)
	cxdSeen      = 0x688291
	cxdSeenEnd   = 0x689231 // 0x4ea436: 1000 species dwords
	cxdMapKills  = 0x686b75 // 0x4ea3f4: word per map, stride 33
	cxdStatEntry = 0x686b79 // 0x4eef10's statEntry i: dword at +33*i
)

type cxdWorld struct{ *mcaWorld }

func (p *cxdWorld) b(a uint32) int64  { return int64(p.u8(a)) }
func (p *cxdWorld) d(a uint32) int64  { return int64(p.u32(a)) }
func (p *cxdWorld) sd(a uint32) int64 { return int64(int32(p.u32(a))) }

func (p *cxdWorld) chars() uint32 { return p.u32(0x685174) }
func (p *cxdWorld) slot(k uint32) uint32 {
	return p.u32(0x685178) + cxdSlotSize*k
}
func (p *cxdWorld) char(k uint32) uint32 { return p.chars() + cxdCharSize*k }

// cxdFeed is a parked callee world with the live values put in by position.
func cxdFeed(park []int64, live map[int]int64) []int64 {
	x := append([]int64(nil), park...)
	for i, v := range live {
		x[i] = v
	}
	return x
}

var cxdKrIn, cxdKrOut = cxdIndex(mkrFieldNames(mkrIns)), cxdIndex(mkrFieldNames(mkrOuts))

func cxdIndex(names []string) map[string]int {
	m := map[string]int{}
	for i, n := range names {
		if _, ok := m[n]; !ok {
			m[n] = i
		}
	}
	return m
}

func cxdAt(m map[string]int, n string) int {
	i, ok := m[n]
	if !ok {
		panic("combat xp distribution: 0x4e46f0 lacks field " + n)
	}
	return i
}

// eligibleCount is 0x4eee80().
func (p *cxdWorld) eligibleCount() uint32 {
	k := p.call(0x4eee80)
	if !p.real {
		return p.replay(0x4eee80, k)
	}
	live := map[int]int64{}
	for i := uint32(0); i < 8; i++ {
		c := p.char(i)
		live[int(3*i)], live[int(3*i+1)], live[int(3*i+2)] = p.b(p.slot(i)), p.sd(c+0xb11), p.d(c+0xb01)
	}
	return uint32(xpEligibleCountNode(cxdFeed(cxdPark4eee80[:], live), p.r)[0])
}

// award is 0x4eef10(xp, 1): the pools and the per-map stat entries.
func (p *cxdWorld) award(xp uint32) {
	k := p.call(0x4eef10, xp, 1)
	if !p.real {
		p.replay(0x4eef10, k, xp, 1)
		return
	}
	live := map[int]int64{0: int64(xp), 1: 1, 2: p.sd(0x686a70)}
	for i := uint32(0); i < 8; i++ {
		c := p.char(i)
		live[int(3+4*i)], live[int(4+4*i)], live[int(5+4*i)], live[int(6+4*i)] =
			p.b(p.slot(i)), p.sd(c+0xb11), p.d(c+0xb01), p.d(c+0x9ed)
	}
	for i := uint32(0); i < 47; i++ {
		live[int(35+i)] = p.d(cxdStatEntry + 33*i)
	}
	o := xpAwardSaturatingNode(cxdFeed(cxdPark4eef10[:], live), p.r)
	for i := uint32(0); i < 8; i++ { // Out: xpPool0..7, statEntry0..46, ...
		p.put(p.char(i)+0x9ed, 4, uint64(o[i]))
	}
	for i := uint32(0); i < 47; i++ {
		p.put(cxdStatEntry+33*i, 4, uint64(o[8+i]))
	}
}

// levelUp is 0x4ef3c0(k).
func (p *cxdWorld) levelUp(k uint32) bool {
	n := p.call(0x4ef3c0, k)
	if !p.real {
		return byte(p.replay(0x4ef3c0, n, k)) != 0
	}
	live := map[int]int64{0: int64(k)}
	for i := uint32(0); i < 8; i++ {
		c := p.char(i)
		live[int(1+5*i)], live[int(2+5*i)], live[int(3+5*i)], live[int(4+5*i)], live[int(5+5*i)] =
			p.b(p.slot(i)), p.sd(c+0xb11), p.d(c+0xb01), p.d(c+0x9ed), p.d(c+0x9f1)
	}
	return byte(levelUpEligibilityNode(cxdFeed(cxdPark4ef3c0[:], live), p.r)[0]) != 0
}

// actionOK is 0x4e79a0(k, 1, 0, 0).
func (p *cxdWorld) actionOK(k uint32) bool {
	n := p.call(0x4e79a0, k, 1, 0, 0)
	if !p.real {
		return byte(p.replay(0x4e79a0, n, k, 1, 0, 0)) != 0
	}
	// In: characterSlot attackContext interfaceMode battle ... combat(7) specialUsesSpell(8)
	// specialUsesItem(9) ... ignoreTarget(24) prepareWeapon(25) actionOverride(26)
	// otherMode0..2(27..29) present/hp/status(30..53).
	live := map[int]int64{0: int64(k), 1: 1, 2: p.d(0x68ec78), 3: p.d(0x6836a8), 7: p.b(0x683f94),
		8: p.b(0x683f95), 9: p.b(0x683f96), 24: 0, 25: 0, 27: p.b(0x683f97), 28: p.b(0x683f98), 29: p.b(0x683f99)}
	for i := uint32(0); i < 8; i++ {
		c := p.char(i)
		live[int(30+3*i)], live[int(31+3*i)], live[int(32+3*i)] = p.b(p.slot(i)), p.d(c+0xb11), p.d(c+0xb01)
	}
	return byte(pcActionEligibilityNode(cxdFeed(cxdPark4e79a0[:], live), p.r)[0]) != 0
}

// remove is 0x4e4280(m): the monster's enabled byte and HP.
func (p *cxdWorld) remove(m uint32) {
	k := p.call(0x4e4280, m)
	if !p.real {
		p.replay(0x4e4280, k, m)
		return
	}
	// In: enabled(0) combatActive(1) monsterId(2) HP(3) ... mode(27); Out: enabled(0) HP(1).
	o := monsterRemoveStateNode(cxdFeed(cxdPark4e4280[:], map[int]int64{0: p.b(m + 0x14), 1: p.b(0x683f94),
		2: p.sd(m), 3: p.d(m + 0x2b), 27: p.d(0x68ec78)}), p.r)
	p.put(m+0x14, 1, uint64(o[0]))
	p.put(m+0x2b, 4, uint64(o[1]))
}

// rewards is 0x4e46f0(m, 1): the party's kill and XP totals and the species' seen state.
func (p *cxdWorld) rewards(m uint32) {
	k := p.call(0x4e46f0, m, 1)
	if !p.real {
		p.replay(0x4e46f0, k, m, 1)
		return
	}
	g, sp := p.u32(0x6836a8), p.u32(m+8)
	seen := cxdSeen + 4*sp
	o := monsterKillRewardsNode(cxdFeed(cxdPark4e46f0[:], map[int]int64{cxdAt(cxdKrIn, "arg2"): 1,
		cxdAt(cxdKrIn, "species"): int64(sp), cxdAt(cxdKrIn, "seen"): p.sd(seen), cxdAt(cxdKrIn, "monId"): p.sd(m),
		cxdAt(cxdKrIn, "m253"): p.b(m + 0x253), cxdAt(cxdKrIn, "m2da"): p.d(m + 0x2da),
		cxdAt(cxdKrIn, "map"): p.sd(0x686a70), cxdAt(cxdKrIn, "partyKills"): p.d(g + 0xc),
		cxdAt(cxdKrIn, "partyExp"): p.d(g + 0x10)}), p.r)
	p.put(g+0xc, 4, uint64(o[cxdAt(cxdKrOut, "partyKills")]))
	p.put(g+0x10, 4, uint64(o[cxdAt(cxdKrOut, "partyExp")]))
	p.put(seen, 4, uint64(o[cxdAt(cxdKrOut, "seen")]))
}

// monCount is [0x683fad]'s count (0x5e2c70) under the fixture cap.
func (p *cxdWorld) monCount() uint32 {
	n := p.listCount(p.u32(0x683fad))
	if n > cxdMonCap {
		panic("combat xp distribution: monster count past the fixture's 3 records")
	}
	return n
}

// showDefeats is 0x4e5c00(quiet).
func (p *cxdWorld) showDefeats(quiet bool) {
	for i := uint32(0); i < p.monCount(); i++ {
		m := p.monsterRecord(i)
		if p.u8(m+0x14) == 0 || p.u32(m+0x2b) == 0 || p.u32(m+0x9f) != 0 || p.u32(m+0x2da) == 0 {
			continue
		}
		if !quiet {
			name := p.stub(0x4e5150, m, 0, 0)
			p.stub(0x58ab60, 9, 0, p.u32(p.u32(0x68c09c)+0x74c), name)
		}
		p.stub(0x5248d0, m)
		if quiet {
			continue
		}
		p.put(m+0x253, 1, 1)
		// 0x4ca4c0 on [m+0xc]: 0x4c4660(6) is the sign-extended render state byte +0xa4.
		rs := p.u32(p.u32(m+0xc) + 0x1d8)
		if int8(p.u8(rs+0xa4)) == 0x15 || p.u8(rs+0xa7) == 0x15 {
			continue
		}
		p.stub(0x4e4db0, m, 0x15, 1)
		p.remove(m)
		p.rewards(m)
		h, ok := p.monsterHandle(p.u32(m))
		if !ok {
			panic("combat xp distribution: monster id not found (0x4e5550 asserts)")
		}
		p.stub(0x4e3af0, h, 0)
	}
}

// uiFlag is 0x5699b0 (bit 0x100) and 0x565420 (bit 0x20000).
func (p *cxdWorld) uiFlag(bit uint32) {
	if ui := p.u32(0x68edcc); p.u32(0x68ec78) == 7 && ui != 0 {
		p.put(ui+0xf4, 4, uint64(p.u32(ui+0xf4)|bit))
	}
}

// endGroup is 0x50fad0(g).
func (p *cxdWorld) endGroup(g uint32) {
	l := p.u32(g + 8)
	if l == 0 {
		panic("combat xp distribution: null group id list (0x5e2c80 asserts)")
	}
	n := p.listCount(l)
	if n > cxdIdCap {
		panic("combat xp distribution: group id count past the fixture's 2 ids")
	}
	for j := uint32(0); j < n; j++ {
		p.stub(0x4e4500, p.findMonster(p.listAt(l, j)))
	}
	p.put(g+0x29, 1, 0)
	p.uiFlag(0x20000)
	m := p.findMonster(p.u32(g + 0x9f))
	p.put(m+0x255, 1, uint64(p.u8(m+0x255)|0x80))
}

// cancelSpecial is 0x4f0560(-1).
func (p *cxdWorld) cancelSpecial() {
	g := p.u32(0x6836a8)
	p.put(g+0x90c, 4, 0)
	for i := uint32(0); i < 8; i++ {
		if p.actionOK(i) {
			p.stub(0x53a930, i, p.slot(i)+0x4d)
		}
	}
	if p.u32(g+0x910) != 0 && p.u32(g+0x918) != 3 || p.u32(g+0x90c) != 0 {
		ui := p.u32(0x68edcc)
		p.put(ui+0x2dc, 4, 100)
		p.put(ui+0x2e0, 4, 100)
		// 0x5a1dd0 calls [[0x69bf4c]]+4: the fixture's vtable slot is 0x5aa400 (ret 4).
		return
	}
	p.stub(0x5a1890)
}

// formation is 0x554a60: copy the battle's 0x21-dword formation when it changed.
func (p *cxdWorld) formation() {
	src := p.u32(0x6836a8) + 0x920
	same := true
	for i := uint32(0); i < 0x21; i++ {
		if p.u32(src+4*i) != p.u32(0x687511+4*i) {
			same = false
		}
	}
	if same {
		return
	}
	for i := uint32(0); i < 0x21; i++ {
		p.put(0x687511+4*i, 4, uint64(p.u32(src+4*i)))
	}
	p.stub(0x5b1c80)
	p.stub(0x5a24a0)
	p.stub(0x58ac00, 8, p.u32(p.u32(0x68c09c)+0x92c), 0, 0xffffffff, 0)
}

// alarm is 0x517780 with [0x683f94] already cleared.
func (p *cxdWorld) alarm() {
	o9c := p.u32(0x68c09c)
	if p.u8(0x6874ff) != 0 {
		p.put(0x6874ff, 1, 0)
		p.stub(0x58ac00, 0xc, p.u32(o9c+0x1e3c), 0xffffffff, 0xffffffff, 0)
		return
	}
	p.put(0x6874ff, 1, 1)
	p.stub(0x58ac00, 0xc, p.u32(o9c+0x1e38), 0xffffffff, 0xffffffff, 0)
	p.put(0x689fcc, 4, uint64(500+p.u32(0x6eb70c))) // 0x406bf0(0x1f4)
	p.put(0x6834d4, 4, 0)                           // 0x4d96f0
}

// levelIcons is 0x59bb70.
func (p *cxdWorld) levelIcons() {
	p.stub(0x4f21e0, 5)
	for i := uint32(0); i < 8; i++ {
		if p.levelUp(i) && p.u8(p.slot(i)+0x103) != 0 {
			p.stub(0x4f2380, i+0x12)
		} else {
			p.stub(0x4f23d0, i+0x12)
		}
	}
}

// widgets is 0x561fb0: 0x4f2220(4) clears row 4's head dword, 0x4f22f0(4) sets
// bits 2-3 to 01 on the widget dwords of rows [0x61f23c+48]..[0x61f240+48].
func (p *cxdWorld) widgets() {
	const row = 4
	if row >= p.u32(0x617b18) {
		panic("combat xp distribution: widget row past 0x617b18 (0x4f2220 asserts)")
	}
	p.put(0x61f238+12*row, 4, 0)
	for j := p.u32(0x61f23c + 12*row); j <= p.u32(0x61f240+12*row); j++ {
		if j >= p.u32(0x617b1c) {
			panic("combat xp distribution: widget index past 0x617b1c (0x4f22f0 asserts)")
		}
		a := 0x620048 + 28*j
		p.put(a, 4, uint64(p.u32(a)&0xfff3|4))
	}
}

// anims is 0x4c61f0 -> 0x4531a0: clear +0x25 and stamp each timer twice via 0x4a9c20.
func (p *cxdWorld) anims() {
	n := int32(p.u32(0x659b34))
	if n > cxdAnimCap {
		panic("combat xp distribution: animation count past the fixture's 2 records")
	}
	for i := int32(0); i < n; i++ {
		a := p.u32(p.u32(0x659b3c) + 4*uint32(i))
		p.put(a+0x25, 1, 0)
		if t := p.u32(a + 0x68); t != 0 {
			p.put(t+0x28, 4, uint64(p.u32(cxdTick))) // GetTickCount, twice
		}
	}
}

// clearTransient is 0x4e5ea0.
func (p *cxdWorld) clearTransient() {
	for i := uint32(0); i < p.monCount(); i++ {
		m := p.monsterRecord(i)
		if p.u32(m+0x2b) == 0 {
			continue
		}
		p.stub(0x452c90)
		if v := p.u8(m + 0x255); v >= 1 && v <= 3 {
			p.put(m+0x255, 1, 0)
		}
	}
}

// item is 0x50b800(id): the item record unless missing or flagged at +0xc7.
func (p *cxdWorld) item(id uint32) uint32 {
	t := p.u32(0x689f94)
	if t == 0 {
		return 0
	}
	arr := p.u32(t + 0xc)
	if int32(id) < int32(p.u32(t+4)) {
		if int32(id) < -1 || int32(id) > 2 {
			panic("combat xp distribution: item index outside the fixture's -1..2")
		}
		arr += 4 * id
	}
	e := p.u32(arr)
	if e != 0 && p.u8(e+0xc7) != 0 {
		return 0
	}
	return e
}

// statusIcons is 0x4ea1f0.
func (p *cxdWorld) statusIcons() {
	for i := uint32(0); i < 2; i++ {
		s := p.slot(i)
		if p.u8(s) == 0 {
			continue
		}
		e := p.item(p.u32(s + 0xfa))
		if e == 0 {
			continue
		}
		st := p.u32(p.char(i) + 0xb01)
		switch {
		case p.u8(e+0xf0) != 0 && st == 0x12:
			p.stub(0x52e5c0, p.u32(0x5ee5bc), 0xffffffff, 0, p.u32(0x5ed8c8))
		case p.u8(e+0xf1) != 0 && st == 0x12:
			p.stub(0x52e5c0, p.u32(0x5ee5d8), 0xffffffff, 0, p.u32(0x5ed8c8))
		}
	}
	for i := uint32(0); i < 8; i++ {
		c := p.char(i)
		if p.u8(p.slot(i)) == 0 || p.u32(c+0xa49) != 0 || p.u32(c+0xa4d) == 0 || p.u8(0x6837a1+0x118*i) != 0 {
			continue
		}
		if p.u8(0x6875f7) != 0 && p.u32(0x6875ff) == i {
			continue
		}
		p.stub(0x52e5c0, p.u32(0x5ee628), i, 0, p.u32(0x5ed8c8))
	}
}

func (p *cxdWorld) exit(arg uint32) {
	quiet := byte(arg) != 0
	if p.u8(0x683fce) != 0 {
		p.stub(0x4f0630)
	}
	if p.u8(0x683fcd) != 0 {
		p.stub(0x5a1890)
	}
	p.uiFlag(0x100)
	p.stub(0x53ae00)
	p.stub(0x53a320, 0)
	p.stub(0x5244a0, 5)
	p.stub(0x5246c0, 0xd)
	p.stub(0x524540)
	p.stub(0x552530)
	p.showDefeats(quiet)
	gl := p.u32(0x683fb1)
	if n := p.listCount(gl); n > cxdGroupCap {
		panic("combat xp distribution: group count past the fixture's 2 records")
	}
	for i := uint32(0); i < p.listCount(gl); i++ {
		g := p.listAt(gl, i) // 0x5101b0
		if g == 0 {
			panic("combat xp distribution: null group record (0x5101b0 asserts)")
		}
		if p.u8(g+0x29) != 0 {
			p.endGroup(g)
		}
	}
	o9c := p.u32(0x68c09c)
	if p.u8(p.u32(0x6836a8)+0xa54) != 0 {
		p.stub(0x58ac00, 0xc, p.u32(o9c+0x8cc), 1, 0xffffffff, 0)
	}
	if n := p.eligibleCount(); n != 0 {
		g := p.u32(0x6836a8)
		if xp := p.u32(g + 0x10); xp != 0 {
			mp := int32(p.u32(0x686a70))
			if mp < -1 {
				panic("combat xp distribution: map index below the fixture's -1")
			}
			if mp < 0x2f {
				a := uint32(int32(cxdMapKills) + 33*mp)
				p.put(a, 2, uint64(p.u16(a)+1))
			}
			p.put(g+0x10, 4, uint64(xp/n))
			p.award(p.u32(g+0x14) + p.u32(g+0x10))
			for a := uint32(cxdSeen); a < cxdSeenEnd; a += 4 {
				if p.u32(a) == 1 {
					p.put(a, 4, 2)
				}
			}
		}
	}
	p.put(0x6850b0, 4, uint64(0x1d4c0+p.u32(0x6eb70c))) // 0x406bf0(0x1d4c0)
	if p.u32(p.u32(0x6836a8)+0x90c) != 0 {
		p.cancelSpecial()
	}
	p.stub(0x56af20, 0)
	p.formation()
	p.stub(0x53cd60)
	p.put(0x683f94, 1, 0)
	if p.u8(p.u32(0x6836a8)+0xa61) != 0 {
		p.alarm()
	}
	p.stub(0x5a3470)
	p.levelIcons()
	p.widgets()
	if p.u32(0x68ec78) == 7 && p.u8(p.u32(0x68edcc)+0x327) == 0 {
		p.stub(0x4263f0, 0x17, 0x34, 0x2d, 0x159)
		p.stub(0x4263f0, 0x253, 0x34, 0x269, 0x159)
		p.stub(0x562a50, 0x810ff)
	}
	if p.u8(0x683f97) == 0 && p.u8(0x683f95) == 0 && p.u8(0x683f96) == 0 {
		p.stub(0x58f6b0, 0)
	}
	p.put(0x6836a8, 4, 0) // after free (0x5eb224) of the battle record
	p.put(0x6081e4, 1, 1) // 0x4c6220(1)
	p.put(0x659c14, 4, 0) // 0x4c6220
	p.anims()
	if !quiet {
		p.stub(0x482990, 1)
	}
	p.clearTransient()
	for i := uint32(0); i < 8; i++ {
		if p.u8(p.slot(i)) != 0 {
			p.stub(0x4ee9d0, p.char(i))
		}
	}
	if p.u8(0x6840bc) != 0 {
		p.stub(0x56aab0)
	}
	if w := p.u32(0x652dac); w != 0 { // 0x41f0c0
		p.put(w, 4, uint64(p.u32(w)&0xffffff1f))
	}
	p.uiFlag(0x20000)
	p.put(0x60ab48, 4, 0x457a0000) // 0x492530
	if p.u32(0x68ec78) == 7 {
		p.statusIcons()
		// 0x515ac0(1)
		if p.u8(0x687599) != 0 {
			p.put(0x68506b, 1, 1)
			p.stub(0x58ac00, 0xc, p.u32(o9c+0x1e0c), 0xffffffff, 0xffffffff, 0)
		}
	}
}

func cxdRun(real bool) func([]int64, Rand) []int64 {
	return func(in []int64, r Rand) []int64 {
		if len(in) != len(cxdSpec.in) {
			panic("combat xp distribution: wrong input count")
		}
		w := &mcaWorld{mstWorld: mstBuild(&cxdSpec, in), real: real, tab: cxdStubs}
		w.r = r
		(&cxdWorld{w}).exit(mcaArg(w, 0))
		out := make([]int64, len(cxdSpec.out))
		for i, f := range cxdSpec.out {
			out[i] = w.read(f, in)
		}
		return out
	}
}

func combatXpDistributionNode(in []int64, r Rand) []int64 { return cxdRun(true)(in, r) }

func cxdEmu(in []int64, r Rand) []int64 { return cxdRun(false)(in, r) }

func init() {
	Nodes["004ea310"] = Node{Addr: "004ea310", In: cxdIn, Out: cxdOut, F: combatXpDistributionNode, Emu: cxdEmu}
}

// Code below is generated from the same description as re/emu/cases/004ea310.json.

const cxdTick = 0x20000354 // the GetTickCount import's case input

var cxdSpec = mstSpec{
	set: []mstSet{
		{0x7f000004, 4, 0},
		{0x6836a8, 4, 0x20001000},
		{0x685178, 4, 0x20002000},
		{0x685174, 4, 0x20010000},
		{0x683fb1, 4, 0x20003000},
		{0x683fad, 4, 0x20004500},
		{0x683fb9, 4, 0x20005b00},
		{0x683fbd, 4, 0x0},
		{0x68edcc, 4, 0x20003400},
		{0x68c09c, 4, 0x20008000},
		{0x652dac, 4, 0x20004200},
		{0x69bf4c, 4, 0x20006e00},
		{0x20006e00, 4, 0x20006e10},
		{0x20006e14, 4, 0x5aa400},
		{0x683f94, 1, 0x1},
		{0x659b3c, 4, 0x20007000},
		{0x20007000, 4, 0x20007100},
		{0x20007004, 4, 0x20007200},
		{0x20003000, 4, 0x20003020},
		{0x20003020, 4, 0x20003100},
		{0x20003024, 4, 0x20003200},
		{0x20004500, 4, 0x20004520},
		{0x20005b00, 4, 0x20005b20},
		{0x20005b08, 4, 0x3},
		{0x2000690c, 4, 0x20006a04},
		{0x20006a04, 4, 0x20006b00},
		{0x20006a08, 4, 0x20006c00},
		{0x20006a0c, 4, 0x0},
		{0x20006a00, 4, 0x20006d00},
		{0x20003108, 4, 0x20003300},
		{0x20003300, 4, 0x20003320},
		{0x20003208, 4, 0x20003340},
		{0x20003340, 4, 0x20003360},
		{0x20004520, 4, 0x20004600},
		{0x2000460c, 4, 0x20005200},
		{0x200053d8, 4, 0x20005800},
		{0x20005b20, 4, 0x20006000},
		{0x20006000, 4, 0x1},
		{0x20004524, 4, 0x20004a00},
		{0x20004a0c, 4, 0x20005400},
		{0x200055d8, 4, 0x20005900},
		{0x20005b24, 4, 0x20006300},
		{0x20006300, 4, 0x2},
		{0x20004528, 4, 0x20004e00},
		{0x20004e0c, 4, 0x20005600},
		{0x200057d8, 4, 0x20005a00},
		{0x20005b28, 4, 0x20006600},
		{0x20006600, 4, 0x3},
		{0x2000874c, 4, 0x3000074c},
		{0x200088cc, 4, 0x300008cc},
		{0x2000892c, 4, 0x3000092c},
		{0x20009e0c, 4, 0x30001e0c},
		{0x20009e38, 4, 0x30001e38},
		{0x20009e3c, 4, 0x30001e3c},
		{0x688291, 4, 0x0},
		{0x688295, 4, 0x1},
		{0x688299, 4, 0x2},
		{0x68829d, 4, 0x0},
		{0x6882a1, 4, 0x1},
		{0x6882a5, 4, 0x2},
		{0x6882a9, 4, 0x0},
		{0x6882ad, 4, 0x1},
		{0x6882b1, 4, 0x2},
		{0x6882b5, 4, 0x0},
		{0x6882b9, 4, 0x1},
		{0x6882bd, 4, 0x2},
		{0x6882c1, 4, 0x0},
		{0x6882c5, 4, 0x1},
		{0x6882c9, 4, 0x2},
		{0x6882cd, 4, 0x0},
		{0x6882d1, 4, 0x1},
		{0x6882d5, 4, 0x2},
		{0x6882d9, 4, 0x0},
		{0x6882dd, 4, 0x1},
		{0x6882e1, 4, 0x2},
		{0x6882e5, 4, 0x0},
		{0x6882e9, 4, 0x1},
		{0x6882ed, 4, 0x2},
		{0x6882f1, 4, 0x0},
		{0x6882f5, 4, 0x1},
		{0x6882f9, 4, 0x2},
		{0x6882fd, 4, 0x0},
		{0x688301, 4, 0x1},
		{0x688305, 4, 0x2},
		{0x688309, 4, 0x0},
		{0x68830d, 4, 0x1},
		{0x688311, 4, 0x2},
		{0x688315, 4, 0x0},
		{0x688319, 4, 0x1},
		{0x68831d, 4, 0x2},
		{0x688321, 4, 0x0},
		{0x688325, 4, 0x1},
		{0x688329, 4, 0x2},
		{0x68832d, 4, 0x0},
		{0x688331, 4, 0x1},
		{0x688335, 4, 0x2},
		{0x688339, 4, 0x0},
		{0x68833d, 4, 0x1},
		{0x688341, 4, 0x2},
		{0x688345, 4, 0x0},
		{0x688349, 4, 0x1},
		{0x68834d, 4, 0x2},
		{0x688351, 4, 0x0},
		{0x688355, 4, 0x1},
		{0x688359, 4, 0x2},
		{0x68835d, 4, 0x0},
		{0x688361, 4, 0x1},
		{0x688365, 4, 0x2},
		{0x688369, 4, 0x0},
		{0x68836d, 4, 0x1},
		{0x688371, 4, 0x2},
		{0x688375, 4, 0x0},
		{0x688379, 4, 0x1},
		{0x68837d, 4, 0x2},
		{0x688381, 4, 0x0},
		{0x688385, 4, 0x1},
		{0x688389, 4, 0x2},
		{0x68838d, 4, 0x0},
		{0x688391, 4, 0x1},
		{0x688395, 4, 0x2},
		{0x688399, 4, 0x0},
		{0x68839d, 4, 0x1},
		{0x6883a1, 4, 0x2},
		{0x6883a5, 4, 0x0},
		{0x6883a9, 4, 0x1},
		{0x6883ad, 4, 0x2},
		{0x6883b1, 4, 0x0},
		{0x6883b5, 4, 0x1},
		{0x6883b9, 4, 0x2},
		{0x6883bd, 4, 0x0},
		{0x6883c1, 4, 0x1},
		{0x6883c5, 4, 0x2},
		{0x6883c9, 4, 0x0},
		{0x6883cd, 4, 0x1},
		{0x6883d1, 4, 0x2},
		{0x6883d5, 4, 0x0},
		{0x6883d9, 4, 0x1},
		{0x6883dd, 4, 0x2},
		{0x6883e1, 4, 0x0},
		{0x6883e5, 4, 0x1},
		{0x6883e9, 4, 0x2},
		{0x6883ed, 4, 0x0},
		{0x6883f1, 4, 0x1},
		{0x6883f5, 4, 0x2},
		{0x6883f9, 4, 0x0},
		{0x6883fd, 4, 0x1},
		{0x688401, 4, 0x2},
		{0x688405, 4, 0x0},
		{0x688409, 4, 0x1},
		{0x68840d, 4, 0x2},
		{0x688411, 4, 0x0},
		{0x688415, 4, 0x1},
		{0x688419, 4, 0x2},
		{0x68841d, 4, 0x0},
		{0x688421, 4, 0x1},
		{0x688425, 4, 0x2},
		{0x688429, 4, 0x0},
		{0x68842d, 4, 0x1},
		{0x688431, 4, 0x2},
		{0x688435, 4, 0x0},
		{0x688439, 4, 0x1},
		{0x68843d, 4, 0x2},
		{0x688441, 4, 0x0},
		{0x688445, 4, 0x1},
		{0x688449, 4, 0x2},
		{0x68844d, 4, 0x0},
		{0x688451, 4, 0x1},
		{0x688455, 4, 0x2},
		{0x688459, 4, 0x0},
		{0x68845d, 4, 0x1},
		{0x688461, 4, 0x2},
		{0x688465, 4, 0x0},
		{0x688469, 4, 0x1},
		{0x68846d, 4, 0x2},
		{0x688471, 4, 0x0},
		{0x688475, 4, 0x1},
		{0x688479, 4, 0x2},
		{0x68847d, 4, 0x0},
		{0x688481, 4, 0x1},
		{0x688485, 4, 0x2},
		{0x688489, 4, 0x0},
		{0x68848d, 4, 0x1},
		{0x688491, 4, 0x2},
		{0x688495, 4, 0x0},
		{0x688499, 4, 0x1},
		{0x68849d, 4, 0x2},
		{0x6884a1, 4, 0x0},
		{0x6884a5, 4, 0x1},
		{0x6884a9, 4, 0x2},
		{0x6884ad, 4, 0x0},
		{0x6884b1, 4, 0x1},
		{0x6884b5, 4, 0x2},
		{0x6884b9, 4, 0x0},
		{0x6884bd, 4, 0x1},
		{0x6884c1, 4, 0x2},
		{0x6884c5, 4, 0x0},
		{0x6884c9, 4, 0x1},
		{0x6884cd, 4, 0x2},
		{0x6884d1, 4, 0x0},
		{0x6884d5, 4, 0x1},
		{0x6884d9, 4, 0x2},
		{0x6884dd, 4, 0x0},
		{0x6884e1, 4, 0x1},
		{0x6884e5, 4, 0x2},
		{0x6884e9, 4, 0x0},
		{0x6884ed, 4, 0x1},
		{0x6884f1, 4, 0x2},
		{0x6884f5, 4, 0x0},
		{0x6884f9, 4, 0x1},
		{0x6884fd, 4, 0x2},
		{0x688501, 4, 0x0},
		{0x688505, 4, 0x1},
		{0x688509, 4, 0x2},
		{0x68850d, 4, 0x0},
		{0x688511, 4, 0x1},
		{0x688515, 4, 0x2},
		{0x688519, 4, 0x0},
		{0x68851d, 4, 0x1},
		{0x688521, 4, 0x2},
		{0x688525, 4, 0x0},
		{0x688529, 4, 0x1},
		{0x68852d, 4, 0x2},
		{0x688531, 4, 0x0},
		{0x688535, 4, 0x1},
		{0x688539, 4, 0x2},
		{0x68853d, 4, 0x0},
		{0x688541, 4, 0x1},
		{0x688545, 4, 0x2},
		{0x688549, 4, 0x0},
		{0x68854d, 4, 0x1},
		{0x688551, 4, 0x2},
		{0x688555, 4, 0x0},
		{0x688559, 4, 0x1},
		{0x68855d, 4, 0x2},
		{0x688561, 4, 0x0},
		{0x688565, 4, 0x1},
		{0x688569, 4, 0x2},
		{0x68856d, 4, 0x0},
		{0x688571, 4, 0x1},
		{0x688575, 4, 0x2},
		{0x688579, 4, 0x0},
		{0x68857d, 4, 0x1},
		{0x688581, 4, 0x2},
		{0x688585, 4, 0x0},
		{0x688589, 4, 0x1},
		{0x68858d, 4, 0x2},
		{0x688591, 4, 0x0},
		{0x688595, 4, 0x1},
		{0x688599, 4, 0x2},
		{0x68859d, 4, 0x0},
		{0x6885a1, 4, 0x1},
		{0x6885a5, 4, 0x2},
		{0x6885a9, 4, 0x0},
		{0x6885ad, 4, 0x1},
		{0x6885b1, 4, 0x2},
		{0x6885b5, 4, 0x0},
		{0x6885b9, 4, 0x1},
		{0x6885bd, 4, 0x2},
		{0x6885c1, 4, 0x0},
		{0x6885c5, 4, 0x1},
		{0x6885c9, 4, 0x2},
		{0x6885cd, 4, 0x0},
		{0x6885d1, 4, 0x1},
		{0x6885d5, 4, 0x2},
		{0x6885d9, 4, 0x0},
		{0x6885dd, 4, 0x1},
		{0x6885e1, 4, 0x2},
		{0x6885e5, 4, 0x0},
		{0x6885e9, 4, 0x1},
		{0x6885ed, 4, 0x2},
		{0x6885f1, 4, 0x0},
		{0x6885f5, 4, 0x1},
		{0x6885f9, 4, 0x2},
		{0x6885fd, 4, 0x0},
		{0x688601, 4, 0x1},
		{0x688605, 4, 0x2},
		{0x688609, 4, 0x0},
		{0x68860d, 4, 0x1},
		{0x688611, 4, 0x2},
		{0x688615, 4, 0x0},
		{0x688619, 4, 0x1},
		{0x68861d, 4, 0x2},
		{0x688621, 4, 0x0},
		{0x688625, 4, 0x1},
		{0x688629, 4, 0x2},
		{0x68862d, 4, 0x0},
		{0x688631, 4, 0x1},
		{0x688635, 4, 0x2},
		{0x688639, 4, 0x0},
		{0x68863d, 4, 0x1},
		{0x688641, 4, 0x2},
		{0x688645, 4, 0x0},
		{0x688649, 4, 0x1},
		{0x68864d, 4, 0x2},
		{0x688651, 4, 0x0},
		{0x688655, 4, 0x1},
		{0x688659, 4, 0x2},
		{0x68865d, 4, 0x0},
		{0x688661, 4, 0x1},
		{0x688665, 4, 0x2},
		{0x688669, 4, 0x0},
		{0x68866d, 4, 0x1},
		{0x688671, 4, 0x2},
		{0x688675, 4, 0x0},
		{0x688679, 4, 0x1},
		{0x68867d, 4, 0x2},
		{0x688681, 4, 0x0},
		{0x688685, 4, 0x1},
		{0x688689, 4, 0x2},
		{0x68868d, 4, 0x0},
		{0x688691, 4, 0x1},
		{0x688695, 4, 0x2},
		{0x688699, 4, 0x0},
		{0x68869d, 4, 0x1},
		{0x6886a1, 4, 0x2},
		{0x6886a5, 4, 0x0},
		{0x6886a9, 4, 0x1},
		{0x6886ad, 4, 0x2},
		{0x6886b1, 4, 0x0},
		{0x6886b5, 4, 0x1},
		{0x6886b9, 4, 0x2},
		{0x6886bd, 4, 0x0},
		{0x6886c1, 4, 0x1},
		{0x6886c5, 4, 0x2},
		{0x6886c9, 4, 0x0},
		{0x6886cd, 4, 0x1},
		{0x6886d1, 4, 0x2},
		{0x6886d5, 4, 0x0},
		{0x6886d9, 4, 0x1},
		{0x6886dd, 4, 0x2},
		{0x6886e1, 4, 0x0},
		{0x6886e5, 4, 0x1},
		{0x6886e9, 4, 0x2},
		{0x6886ed, 4, 0x0},
		{0x6886f1, 4, 0x1},
		{0x6886f5, 4, 0x2},
		{0x6886f9, 4, 0x0},
		{0x6886fd, 4, 0x1},
		{0x688701, 4, 0x2},
		{0x688705, 4, 0x0},
		{0x688709, 4, 0x1},
		{0x68870d, 4, 0x2},
		{0x688711, 4, 0x0},
		{0x688715, 4, 0x1},
		{0x688719, 4, 0x2},
		{0x68871d, 4, 0x0},
		{0x688721, 4, 0x1},
		{0x688725, 4, 0x2},
		{0x688729, 4, 0x0},
		{0x68872d, 4, 0x1},
		{0x688731, 4, 0x2},
		{0x688735, 4, 0x0},
		{0x688739, 4, 0x1},
		{0x68873d, 4, 0x2},
		{0x688741, 4, 0x0},
		{0x688745, 4, 0x1},
		{0x688749, 4, 0x2},
		{0x68874d, 4, 0x0},
		{0x688751, 4, 0x1},
		{0x688755, 4, 0x2},
		{0x688759, 4, 0x0},
		{0x68875d, 4, 0x1},
		{0x688761, 4, 0x2},
		{0x688765, 4, 0x0},
		{0x688769, 4, 0x1},
		{0x68876d, 4, 0x2},
		{0x688771, 4, 0x0},
		{0x688775, 4, 0x1},
		{0x688779, 4, 0x2},
		{0x68877d, 4, 0x0},
		{0x688781, 4, 0x1},
		{0x688785, 4, 0x2},
		{0x688789, 4, 0x0},
		{0x68878d, 4, 0x1},
		{0x688791, 4, 0x2},
		{0x688795, 4, 0x0},
		{0x688799, 4, 0x1},
		{0x68879d, 4, 0x2},
		{0x6887a1, 4, 0x0},
		{0x6887a5, 4, 0x1},
		{0x6887a9, 4, 0x2},
		{0x6887ad, 4, 0x0},
		{0x6887b1, 4, 0x1},
		{0x6887b5, 4, 0x2},
		{0x6887b9, 4, 0x0},
		{0x6887bd, 4, 0x1},
		{0x6887c1, 4, 0x2},
		{0x6887c5, 4, 0x0},
		{0x6887c9, 4, 0x1},
		{0x6887cd, 4, 0x2},
		{0x6887d1, 4, 0x0},
		{0x6887d5, 4, 0x1},
		{0x6887d9, 4, 0x2},
		{0x6887dd, 4, 0x0},
		{0x6887e1, 4, 0x1},
		{0x6887e5, 4, 0x2},
		{0x6887e9, 4, 0x0},
		{0x6887ed, 4, 0x1},
		{0x6887f1, 4, 0x2},
		{0x6887f5, 4, 0x0},
		{0x6887f9, 4, 0x1},
		{0x6887fd, 4, 0x2},
		{0x688801, 4, 0x0},
		{0x688805, 4, 0x1},
		{0x688809, 4, 0x2},
		{0x68880d, 4, 0x0},
		{0x688811, 4, 0x1},
		{0x688815, 4, 0x2},
		{0x688819, 4, 0x0},
		{0x68881d, 4, 0x1},
		{0x688821, 4, 0x2},
		{0x688825, 4, 0x0},
		{0x688829, 4, 0x1},
		{0x68882d, 4, 0x2},
		{0x688831, 4, 0x0},
		{0x688835, 4, 0x1},
		{0x688839, 4, 0x2},
		{0x68883d, 4, 0x0},
		{0x688841, 4, 0x1},
		{0x688845, 4, 0x2},
		{0x688849, 4, 0x0},
		{0x68884d, 4, 0x1},
		{0x688851, 4, 0x2},
		{0x688855, 4, 0x0},
		{0x688859, 4, 0x1},
		{0x68885d, 4, 0x2},
		{0x688861, 4, 0x0},
		{0x688865, 4, 0x1},
		{0x688869, 4, 0x2},
		{0x68886d, 4, 0x0},
		{0x688871, 4, 0x1},
		{0x688875, 4, 0x2},
		{0x688879, 4, 0x0},
		{0x68887d, 4, 0x1},
		{0x688881, 4, 0x2},
		{0x688885, 4, 0x0},
		{0x688889, 4, 0x1},
		{0x68888d, 4, 0x2},
		{0x688891, 4, 0x0},
		{0x688895, 4, 0x1},
		{0x688899, 4, 0x2},
		{0x68889d, 4, 0x0},
		{0x6888a1, 4, 0x1},
		{0x6888a5, 4, 0x2},
		{0x6888a9, 4, 0x0},
		{0x6888ad, 4, 0x1},
		{0x6888b1, 4, 0x2},
		{0x6888b5, 4, 0x0},
		{0x6888b9, 4, 0x1},
		{0x6888bd, 4, 0x2},
		{0x6888c1, 4, 0x0},
		{0x6888c5, 4, 0x1},
		{0x6888c9, 4, 0x2},
		{0x6888cd, 4, 0x0},
		{0x6888d1, 4, 0x1},
		{0x6888d5, 4, 0x2},
		{0x6888d9, 4, 0x0},
		{0x6888dd, 4, 0x1},
		{0x6888e1, 4, 0x2},
		{0x6888e5, 4, 0x0},
		{0x6888e9, 4, 0x1},
		{0x6888ed, 4, 0x2},
		{0x6888f1, 4, 0x0},
		{0x6888f5, 4, 0x1},
		{0x6888f9, 4, 0x2},
		{0x6888fd, 4, 0x0},
		{0x688901, 4, 0x1},
		{0x688905, 4, 0x2},
		{0x688909, 4, 0x0},
		{0x68890d, 4, 0x1},
		{0x688911, 4, 0x2},
		{0x688915, 4, 0x0},
		{0x688919, 4, 0x1},
		{0x68891d, 4, 0x2},
		{0x688921, 4, 0x0},
		{0x688925, 4, 0x1},
		{0x688929, 4, 0x2},
		{0x68892d, 4, 0x0},
		{0x688931, 4, 0x1},
		{0x688935, 4, 0x2},
		{0x688939, 4, 0x0},
		{0x68893d, 4, 0x1},
		{0x688941, 4, 0x2},
		{0x688945, 4, 0x0},
		{0x688949, 4, 0x1},
		{0x68894d, 4, 0x2},
		{0x688951, 4, 0x0},
		{0x688955, 4, 0x1},
		{0x688959, 4, 0x2},
		{0x68895d, 4, 0x0},
		{0x688961, 4, 0x1},
		{0x688965, 4, 0x2},
		{0x688969, 4, 0x0},
		{0x68896d, 4, 0x1},
		{0x688971, 4, 0x2},
		{0x688975, 4, 0x0},
		{0x688979, 4, 0x1},
		{0x68897d, 4, 0x2},
		{0x688981, 4, 0x0},
		{0x688985, 4, 0x1},
		{0x688989, 4, 0x2},
		{0x68898d, 4, 0x0},
		{0x688991, 4, 0x1},
		{0x688995, 4, 0x2},
		{0x688999, 4, 0x0},
		{0x68899d, 4, 0x1},
		{0x6889a1, 4, 0x2},
		{0x6889a5, 4, 0x0},
		{0x6889a9, 4, 0x1},
		{0x6889ad, 4, 0x2},
		{0x6889b1, 4, 0x0},
		{0x6889b5, 4, 0x1},
		{0x6889b9, 4, 0x2},
		{0x6889bd, 4, 0x0},
		{0x6889c1, 4, 0x1},
		{0x6889c5, 4, 0x2},
		{0x6889c9, 4, 0x0},
		{0x6889cd, 4, 0x1},
		{0x6889d1, 4, 0x2},
		{0x6889d5, 4, 0x0},
		{0x6889d9, 4, 0x1},
		{0x6889dd, 4, 0x2},
		{0x6889e1, 4, 0x0},
		{0x6889e5, 4, 0x1},
		{0x6889e9, 4, 0x2},
		{0x6889ed, 4, 0x0},
		{0x6889f1, 4, 0x1},
		{0x6889f5, 4, 0x2},
		{0x6889f9, 4, 0x0},
		{0x6889fd, 4, 0x1},
		{0x688a01, 4, 0x2},
		{0x688a05, 4, 0x0},
		{0x688a09, 4, 0x1},
		{0x688a0d, 4, 0x2},
		{0x688a11, 4, 0x0},
		{0x688a15, 4, 0x1},
		{0x688a19, 4, 0x2},
		{0x688a1d, 4, 0x0},
		{0x688a21, 4, 0x1},
		{0x688a25, 4, 0x2},
		{0x688a29, 4, 0x0},
		{0x688a2d, 4, 0x1},
		{0x688a31, 4, 0x2},
		{0x688a35, 4, 0x0},
		{0x688a39, 4, 0x1},
		{0x688a3d, 4, 0x2},
		{0x688a41, 4, 0x0},
		{0x688a45, 4, 0x1},
		{0x688a49, 4, 0x2},
		{0x688a4d, 4, 0x0},
		{0x688a51, 4, 0x1},
		{0x688a55, 4, 0x2},
		{0x688a59, 4, 0x0},
		{0x688a5d, 4, 0x1},
		{0x688a61, 4, 0x2},
		{0x688a65, 4, 0x0},
		{0x688a69, 4, 0x1},
		{0x688a6d, 4, 0x2},
		{0x688a71, 4, 0x0},
		{0x688a75, 4, 0x1},
		{0x688a79, 4, 0x2},
		{0x688a7d, 4, 0x0},
		{0x688a81, 4, 0x1},
		{0x688a85, 4, 0x2},
		{0x688a89, 4, 0x0},
		{0x688a8d, 4, 0x1},
		{0x688a91, 4, 0x2},
		{0x688a95, 4, 0x0},
		{0x688a99, 4, 0x1},
		{0x688a9d, 4, 0x2},
		{0x688aa1, 4, 0x0},
		{0x688aa5, 4, 0x1},
		{0x688aa9, 4, 0x2},
		{0x688aad, 4, 0x0},
		{0x688ab1, 4, 0x1},
		{0x688ab5, 4, 0x2},
		{0x688ab9, 4, 0x0},
		{0x688abd, 4, 0x1},
		{0x688ac1, 4, 0x2},
		{0x688ac5, 4, 0x0},
		{0x688ac9, 4, 0x1},
		{0x688acd, 4, 0x2},
		{0x688ad1, 4, 0x0},
		{0x688ad5, 4, 0x1},
		{0x688ad9, 4, 0x2},
		{0x688add, 4, 0x0},
		{0x688ae1, 4, 0x1},
		{0x688ae5, 4, 0x2},
		{0x688ae9, 4, 0x0},
		{0x688aed, 4, 0x1},
		{0x688af1, 4, 0x2},
		{0x688af5, 4, 0x0},
		{0x688af9, 4, 0x1},
		{0x688afd, 4, 0x2},
		{0x688b01, 4, 0x0},
		{0x688b05, 4, 0x1},
		{0x688b09, 4, 0x2},
		{0x688b0d, 4, 0x0},
		{0x688b11, 4, 0x1},
		{0x688b15, 4, 0x2},
		{0x688b19, 4, 0x0},
		{0x688b1d, 4, 0x1},
		{0x688b21, 4, 0x2},
		{0x688b25, 4, 0x0},
		{0x688b29, 4, 0x1},
		{0x688b2d, 4, 0x2},
		{0x688b31, 4, 0x0},
		{0x688b35, 4, 0x1},
		{0x688b39, 4, 0x2},
		{0x688b3d, 4, 0x0},
		{0x688b41, 4, 0x1},
		{0x688b45, 4, 0x2},
		{0x688b49, 4, 0x0},
		{0x688b4d, 4, 0x1},
		{0x688b51, 4, 0x2},
		{0x688b55, 4, 0x0},
		{0x688b59, 4, 0x1},
		{0x688b5d, 4, 0x2},
		{0x688b61, 4, 0x0},
		{0x688b65, 4, 0x1},
		{0x688b69, 4, 0x2},
		{0x688b6d, 4, 0x0},
		{0x688b71, 4, 0x1},
		{0x688b75, 4, 0x2},
		{0x688b79, 4, 0x0},
		{0x688b7d, 4, 0x1},
		{0x688b81, 4, 0x2},
		{0x688b85, 4, 0x0},
		{0x688b89, 4, 0x1},
		{0x688b8d, 4, 0x2},
		{0x688b91, 4, 0x0},
		{0x688b95, 4, 0x1},
		{0x688b99, 4, 0x2},
		{0x688b9d, 4, 0x0},
		{0x688ba1, 4, 0x1},
		{0x688ba5, 4, 0x2},
		{0x688ba9, 4, 0x0},
		{0x688bad, 4, 0x1},
		{0x688bb1, 4, 0x2},
		{0x688bb5, 4, 0x0},
		{0x688bb9, 4, 0x1},
		{0x688bbd, 4, 0x2},
		{0x688bc1, 4, 0x0},
		{0x688bc5, 4, 0x1},
		{0x688bc9, 4, 0x2},
		{0x688bcd, 4, 0x0},
		{0x688bd1, 4, 0x1},
		{0x688bd5, 4, 0x2},
		{0x688bd9, 4, 0x0},
		{0x688bdd, 4, 0x1},
		{0x688be1, 4, 0x2},
		{0x688be5, 4, 0x0},
		{0x688be9, 4, 0x1},
		{0x688bed, 4, 0x2},
		{0x688bf1, 4, 0x0},
		{0x688bf5, 4, 0x1},
		{0x688bf9, 4, 0x2},
		{0x688bfd, 4, 0x0},
		{0x688c01, 4, 0x1},
		{0x688c05, 4, 0x2},
		{0x688c09, 4, 0x0},
		{0x688c0d, 4, 0x1},
		{0x688c11, 4, 0x2},
		{0x688c15, 4, 0x0},
		{0x688c19, 4, 0x1},
		{0x688c1d, 4, 0x2},
		{0x688c21, 4, 0x0},
		{0x688c25, 4, 0x1},
		{0x688c29, 4, 0x2},
		{0x688c2d, 4, 0x0},
		{0x688c31, 4, 0x1},
		{0x688c35, 4, 0x2},
		{0x688c39, 4, 0x0},
		{0x688c3d, 4, 0x1},
		{0x688c41, 4, 0x2},
		{0x688c45, 4, 0x0},
		{0x688c49, 4, 0x1},
		{0x688c4d, 4, 0x2},
		{0x688c51, 4, 0x0},
		{0x688c55, 4, 0x1},
		{0x688c59, 4, 0x2},
		{0x688c5d, 4, 0x0},
		{0x688c61, 4, 0x1},
		{0x688c65, 4, 0x2},
		{0x688c69, 4, 0x0},
		{0x688c6d, 4, 0x1},
		{0x688c71, 4, 0x2},
		{0x688c75, 4, 0x0},
		{0x688c79, 4, 0x1},
		{0x688c7d, 4, 0x2},
		{0x688c81, 4, 0x0},
		{0x688c85, 4, 0x1},
		{0x688c89, 4, 0x2},
		{0x688c8d, 4, 0x0},
		{0x688c91, 4, 0x1},
		{0x688c95, 4, 0x2},
		{0x688c99, 4, 0x0},
		{0x688c9d, 4, 0x1},
		{0x688ca1, 4, 0x2},
		{0x688ca5, 4, 0x0},
		{0x688ca9, 4, 0x1},
		{0x688cad, 4, 0x2},
		{0x688cb1, 4, 0x0},
		{0x688cb5, 4, 0x1},
		{0x688cb9, 4, 0x2},
		{0x688cbd, 4, 0x0},
		{0x688cc1, 4, 0x1},
		{0x688cc5, 4, 0x2},
		{0x688cc9, 4, 0x0},
		{0x688ccd, 4, 0x1},
		{0x688cd1, 4, 0x2},
		{0x688cd5, 4, 0x0},
		{0x688cd9, 4, 0x1},
		{0x688cdd, 4, 0x2},
		{0x688ce1, 4, 0x0},
		{0x688ce5, 4, 0x1},
		{0x688ce9, 4, 0x2},
		{0x688ced, 4, 0x0},
		{0x688cf1, 4, 0x1},
		{0x688cf5, 4, 0x2},
		{0x688cf9, 4, 0x0},
		{0x688cfd, 4, 0x1},
		{0x688d01, 4, 0x2},
		{0x688d05, 4, 0x0},
		{0x688d09, 4, 0x1},
		{0x688d0d, 4, 0x2},
		{0x688d11, 4, 0x0},
		{0x688d15, 4, 0x1},
		{0x688d19, 4, 0x2},
		{0x688d1d, 4, 0x0},
		{0x688d21, 4, 0x1},
		{0x688d25, 4, 0x2},
		{0x688d29, 4, 0x0},
		{0x688d2d, 4, 0x1},
		{0x688d31, 4, 0x2},
		{0x688d35, 4, 0x0},
		{0x688d39, 4, 0x1},
		{0x688d3d, 4, 0x2},
		{0x688d41, 4, 0x0},
		{0x688d45, 4, 0x1},
		{0x688d49, 4, 0x2},
		{0x688d4d, 4, 0x0},
		{0x688d51, 4, 0x1},
		{0x688d55, 4, 0x2},
		{0x688d59, 4, 0x0},
		{0x688d5d, 4, 0x1},
		{0x688d61, 4, 0x2},
		{0x688d65, 4, 0x0},
		{0x688d69, 4, 0x1},
		{0x688d6d, 4, 0x2},
		{0x688d71, 4, 0x0},
		{0x688d75, 4, 0x1},
		{0x688d79, 4, 0x2},
		{0x688d7d, 4, 0x0},
		{0x688d81, 4, 0x1},
		{0x688d85, 4, 0x2},
		{0x688d89, 4, 0x0},
		{0x688d8d, 4, 0x1},
		{0x688d91, 4, 0x2},
		{0x688d95, 4, 0x0},
		{0x688d99, 4, 0x1},
		{0x688d9d, 4, 0x2},
		{0x688da1, 4, 0x0},
		{0x688da5, 4, 0x1},
		{0x688da9, 4, 0x2},
		{0x688dad, 4, 0x0},
		{0x688db1, 4, 0x1},
		{0x688db5, 4, 0x2},
		{0x688db9, 4, 0x0},
		{0x688dbd, 4, 0x1},
		{0x688dc1, 4, 0x2},
		{0x688dc5, 4, 0x0},
		{0x688dc9, 4, 0x1},
		{0x688dcd, 4, 0x2},
		{0x688dd1, 4, 0x0},
		{0x688dd5, 4, 0x1},
		{0x688dd9, 4, 0x2},
		{0x688ddd, 4, 0x0},
		{0x688de1, 4, 0x1},
		{0x688de5, 4, 0x2},
		{0x688de9, 4, 0x0},
		{0x688ded, 4, 0x1},
		{0x688df1, 4, 0x2},
		{0x688df5, 4, 0x0},
		{0x688df9, 4, 0x1},
		{0x688dfd, 4, 0x2},
		{0x688e01, 4, 0x0},
		{0x688e05, 4, 0x1},
		{0x688e09, 4, 0x2},
		{0x688e0d, 4, 0x0},
		{0x688e11, 4, 0x1},
		{0x688e15, 4, 0x2},
		{0x688e19, 4, 0x0},
		{0x688e1d, 4, 0x1},
		{0x688e21, 4, 0x2},
		{0x688e25, 4, 0x0},
		{0x688e29, 4, 0x1},
		{0x688e2d, 4, 0x2},
		{0x688e31, 4, 0x0},
		{0x688e35, 4, 0x1},
		{0x688e39, 4, 0x2},
		{0x688e3d, 4, 0x0},
		{0x688e41, 4, 0x1},
		{0x688e45, 4, 0x2},
		{0x688e49, 4, 0x0},
		{0x688e4d, 4, 0x1},
		{0x688e51, 4, 0x2},
		{0x688e55, 4, 0x0},
		{0x688e59, 4, 0x1},
		{0x688e5d, 4, 0x2},
		{0x688e61, 4, 0x0},
		{0x688e65, 4, 0x1},
		{0x688e69, 4, 0x2},
		{0x688e6d, 4, 0x0},
		{0x688e71, 4, 0x1},
		{0x688e75, 4, 0x2},
		{0x688e79, 4, 0x0},
		{0x688e7d, 4, 0x1},
		{0x688e81, 4, 0x2},
		{0x688e85, 4, 0x0},
		{0x688e89, 4, 0x1},
		{0x688e8d, 4, 0x2},
		{0x688e91, 4, 0x0},
		{0x688e95, 4, 0x1},
		{0x688e99, 4, 0x2},
		{0x688e9d, 4, 0x0},
		{0x688ea1, 4, 0x1},
		{0x688ea5, 4, 0x2},
		{0x688ea9, 4, 0x0},
		{0x688ead, 4, 0x1},
		{0x688eb1, 4, 0x2},
		{0x688eb5, 4, 0x0},
		{0x688eb9, 4, 0x1},
		{0x688ebd, 4, 0x2},
		{0x688ec1, 4, 0x0},
		{0x688ec5, 4, 0x1},
		{0x688ec9, 4, 0x2},
		{0x688ecd, 4, 0x0},
		{0x688ed1, 4, 0x1},
		{0x688ed5, 4, 0x2},
		{0x688ed9, 4, 0x0},
		{0x688edd, 4, 0x1},
		{0x688ee1, 4, 0x2},
		{0x688ee5, 4, 0x0},
		{0x688ee9, 4, 0x1},
		{0x688eed, 4, 0x2},
		{0x688ef1, 4, 0x0},
		{0x688ef5, 4, 0x1},
		{0x688ef9, 4, 0x2},
		{0x688efd, 4, 0x0},
		{0x688f01, 4, 0x1},
		{0x688f05, 4, 0x2},
		{0x688f09, 4, 0x0},
		{0x688f0d, 4, 0x1},
		{0x688f11, 4, 0x2},
		{0x688f15, 4, 0x0},
		{0x688f19, 4, 0x1},
		{0x688f1d, 4, 0x2},
		{0x688f21, 4, 0x0},
		{0x688f25, 4, 0x1},
		{0x688f29, 4, 0x2},
		{0x688f2d, 4, 0x0},
		{0x688f31, 4, 0x1},
		{0x688f35, 4, 0x2},
		{0x688f39, 4, 0x0},
		{0x688f3d, 4, 0x1},
		{0x688f41, 4, 0x2},
		{0x688f45, 4, 0x0},
		{0x688f49, 4, 0x1},
		{0x688f4d, 4, 0x2},
		{0x688f51, 4, 0x0},
		{0x688f55, 4, 0x1},
		{0x688f59, 4, 0x2},
		{0x688f5d, 4, 0x0},
		{0x688f61, 4, 0x1},
		{0x688f65, 4, 0x2},
		{0x688f69, 4, 0x0},
		{0x688f6d, 4, 0x1},
		{0x688f71, 4, 0x2},
		{0x688f75, 4, 0x0},
		{0x688f79, 4, 0x1},
		{0x688f7d, 4, 0x2},
		{0x688f81, 4, 0x0},
		{0x688f85, 4, 0x1},
		{0x688f89, 4, 0x2},
		{0x688f8d, 4, 0x0},
		{0x688f91, 4, 0x1},
		{0x688f95, 4, 0x2},
		{0x688f99, 4, 0x0},
		{0x688f9d, 4, 0x1},
		{0x688fa1, 4, 0x2},
		{0x688fa5, 4, 0x0},
		{0x688fa9, 4, 0x1},
		{0x688fad, 4, 0x2},
		{0x688fb1, 4, 0x0},
		{0x688fb5, 4, 0x1},
		{0x688fb9, 4, 0x2},
		{0x688fbd, 4, 0x0},
		{0x688fc1, 4, 0x1},
		{0x688fc5, 4, 0x2},
		{0x688fc9, 4, 0x0},
		{0x688fcd, 4, 0x1},
		{0x688fd1, 4, 0x2},
		{0x688fd5, 4, 0x0},
		{0x688fd9, 4, 0x1},
		{0x688fdd, 4, 0x2},
		{0x688fe1, 4, 0x0},
		{0x688fe5, 4, 0x1},
		{0x688fe9, 4, 0x2},
		{0x688fed, 4, 0x0},
		{0x688ff1, 4, 0x1},
		{0x688ff5, 4, 0x2},
		{0x688ff9, 4, 0x0},
		{0x688ffd, 4, 0x1},
		{0x689001, 4, 0x2},
		{0x689005, 4, 0x0},
		{0x689009, 4, 0x1},
		{0x68900d, 4, 0x2},
		{0x689011, 4, 0x0},
		{0x689015, 4, 0x1},
		{0x689019, 4, 0x2},
		{0x68901d, 4, 0x0},
		{0x689021, 4, 0x1},
		{0x689025, 4, 0x2},
		{0x689029, 4, 0x0},
		{0x68902d, 4, 0x1},
		{0x689031, 4, 0x2},
		{0x689035, 4, 0x0},
		{0x689039, 4, 0x1},
		{0x68903d, 4, 0x2},
		{0x689041, 4, 0x0},
		{0x689045, 4, 0x1},
		{0x689049, 4, 0x2},
		{0x68904d, 4, 0x0},
		{0x689051, 4, 0x1},
		{0x689055, 4, 0x2},
		{0x689059, 4, 0x0},
		{0x68905d, 4, 0x1},
		{0x689061, 4, 0x2},
		{0x689065, 4, 0x0},
		{0x689069, 4, 0x1},
		{0x68906d, 4, 0x2},
		{0x689071, 4, 0x0},
		{0x689075, 4, 0x1},
		{0x689079, 4, 0x2},
		{0x68907d, 4, 0x0},
		{0x689081, 4, 0x1},
		{0x689085, 4, 0x2},
		{0x689089, 4, 0x0},
		{0x68908d, 4, 0x1},
		{0x689091, 4, 0x2},
		{0x689095, 4, 0x0},
		{0x689099, 4, 0x1},
		{0x68909d, 4, 0x2},
		{0x6890a1, 4, 0x0},
		{0x6890a5, 4, 0x1},
		{0x6890a9, 4, 0x2},
		{0x6890ad, 4, 0x0},
		{0x6890b1, 4, 0x1},
		{0x6890b5, 4, 0x2},
		{0x6890b9, 4, 0x0},
		{0x6890bd, 4, 0x1},
		{0x6890c1, 4, 0x2},
		{0x6890c5, 4, 0x0},
		{0x6890c9, 4, 0x1},
		{0x6890cd, 4, 0x2},
		{0x6890d1, 4, 0x0},
		{0x6890d5, 4, 0x1},
		{0x6890d9, 4, 0x2},
		{0x6890dd, 4, 0x0},
		{0x6890e1, 4, 0x1},
		{0x6890e5, 4, 0x2},
		{0x6890e9, 4, 0x0},
		{0x6890ed, 4, 0x1},
		{0x6890f1, 4, 0x2},
		{0x6890f5, 4, 0x0},
		{0x6890f9, 4, 0x1},
		{0x6890fd, 4, 0x2},
		{0x689101, 4, 0x0},
		{0x689105, 4, 0x1},
		{0x689109, 4, 0x2},
		{0x68910d, 4, 0x0},
		{0x689111, 4, 0x1},
		{0x689115, 4, 0x2},
		{0x689119, 4, 0x0},
		{0x68911d, 4, 0x1},
		{0x689121, 4, 0x2},
		{0x689125, 4, 0x0},
		{0x689129, 4, 0x1},
		{0x68912d, 4, 0x2},
		{0x689131, 4, 0x0},
		{0x689135, 4, 0x1},
		{0x689139, 4, 0x2},
		{0x68913d, 4, 0x0},
		{0x689141, 4, 0x1},
		{0x689145, 4, 0x2},
		{0x689149, 4, 0x0},
		{0x68914d, 4, 0x1},
		{0x689151, 4, 0x2},
		{0x689155, 4, 0x0},
		{0x689159, 4, 0x1},
		{0x68915d, 4, 0x2},
		{0x689161, 4, 0x0},
		{0x689165, 4, 0x1},
		{0x689169, 4, 0x2},
		{0x68916d, 4, 0x0},
		{0x689171, 4, 0x1},
		{0x689175, 4, 0x2},
		{0x689179, 4, 0x0},
		{0x68917d, 4, 0x1},
		{0x689181, 4, 0x2},
		{0x689185, 4, 0x0},
		{0x689189, 4, 0x1},
		{0x68918d, 4, 0x2},
		{0x689191, 4, 0x0},
		{0x689195, 4, 0x1},
		{0x689199, 4, 0x2},
		{0x68919d, 4, 0x0},
		{0x6891a1, 4, 0x1},
		{0x6891a5, 4, 0x2},
		{0x6891a9, 4, 0x0},
		{0x6891ad, 4, 0x1},
		{0x6891b1, 4, 0x2},
		{0x6891b5, 4, 0x0},
		{0x6891b9, 4, 0x1},
		{0x6891bd, 4, 0x2},
		{0x6891c1, 4, 0x0},
		{0x6891c5, 4, 0x1},
		{0x6891c9, 4, 0x2},
		{0x6891cd, 4, 0x0},
		{0x6891d1, 4, 0x1},
		{0x6891d5, 4, 0x2},
		{0x6891d9, 4, 0x0},
		{0x6891dd, 4, 0x1},
		{0x6891e1, 4, 0x2},
		{0x6891e5, 4, 0x0},
		{0x6891e9, 4, 0x1},
		{0x6891ed, 4, 0x2},
		{0x6891f1, 4, 0x0},
		{0x6891f5, 4, 0x1},
		{0x6891f9, 4, 0x2},
		{0x6891fd, 4, 0x0},
		{0x689201, 4, 0x1},
		{0x689205, 4, 0x2},
		{0x689209, 4, 0x0},
		{0x68920d, 4, 0x1},
		{0x689211, 4, 0x2},
		{0x689215, 4, 0x0},
		{0x689219, 4, 0x1},
		{0x68921d, 4, 0x2},
		{0x689221, 4, 0x0},
		{0x689225, 4, 0x1},
		{0x689229, 4, 0x2},
		{0x68922d, 4, 0x0},
		{0x686b54, 2, 0x100},
		{0x686b75, 2, 0x101},
		{0x686b96, 2, 0x102},
		{0x686bb7, 2, 0x103},
		{0x686bd8, 2, 0x104},
		{0x686bf9, 2, 0x105},
		{0x686c1a, 2, 0x106},
		{0x686c3b, 2, 0x107},
		{0x686c5c, 2, 0x108},
		{0x686c7d, 2, 0x109},
		{0x686c9e, 2, 0x10a},
		{0x686cbf, 2, 0x10b},
		{0x686ce0, 2, 0x10c},
		{0x686d01, 2, 0x10d},
		{0x686d22, 2, 0x10e},
		{0x686d43, 2, 0x10f},
		{0x686d64, 2, 0x110},
		{0x686d85, 2, 0x111},
		{0x686da6, 2, 0x112},
		{0x686dc7, 2, 0x113},
		{0x686de8, 2, 0x114},
		{0x686e09, 2, 0x115},
		{0x686e2a, 2, 0x116},
		{0x686e4b, 2, 0x117},
		{0x686e6c, 2, 0x118},
		{0x686e8d, 2, 0x119},
		{0x686eae, 2, 0x11a},
		{0x686ecf, 2, 0x11b},
		{0x686ef0, 2, 0x11c},
		{0x686f11, 2, 0x11d},
		{0x686f32, 2, 0x11e},
		{0x686f53, 2, 0x11f},
		{0x686f74, 2, 0x120},
		{0x686f95, 2, 0x121},
		{0x686fb6, 2, 0x122},
		{0x686fd7, 2, 0x123},
		{0x686ff8, 2, 0x124},
		{0x687019, 2, 0x125},
		{0x68703a, 2, 0x126},
		{0x68705b, 2, 0x127},
		{0x68707c, 2, 0x128},
		{0x68709d, 2, 0x129},
		{0x6870be, 2, 0x12a},
		{0x6870df, 2, 0x12b},
		{0x687100, 2, 0x12c},
		{0x687121, 2, 0x12d},
		{0x687142, 2, 0x12e},
		{0x687163, 2, 0x12f},
		{0x687184, 2, 0x130},
		{0x6871a5, 2, 0x131},
		{0x6871c6, 2, 0x132},
		{0x6871e7, 2, 0x133},
	},
	in: []mstFld{
		{addr: 0x7f000004, size: 4, signed: false, idx: -1},
		{addr: 0x683fce, size: 1, signed: false, idx: -1},
		{addr: 0x683fcd, size: 1, signed: false, idx: -1},
		{addr: 0x68ec78, size: 4, signed: false, idx: -1},
		{addr: 0x200034f4, size: 4, signed: false, idx: -1},
		{addr: 0x20003727, size: 1, signed: false, idx: -1},
		{addr: 0x200036dc, size: 4, signed: false, idx: -1},
		{addr: 0x200036e0, size: 4, signed: false, idx: -1},
		{addr: 0x20001010, size: 4, signed: false, idx: -1},
		{addr: 0x20001014, size: 4, signed: false, idx: -1},
		{addr: 0x20001a54, size: 1, signed: false, idx: -1},
		{addr: 0x2000190c, size: 4, signed: false, idx: -1},
		{addr: 0x20001a61, size: 1, signed: false, idx: -1},
		{addr: 0x20001910, size: 4, signed: false, idx: -1},
		{addr: 0x20001918, size: 4, signed: false, idx: -1},
		{addr: 0x20001920, size: 4, signed: false, idx: -1},
		{addr: 0x200019a0, size: 4, signed: false, idx: -1},
		{addr: 0x687511, size: 4, signed: false, idx: -1},
		{addr: 0x687591, size: 4, signed: false, idx: -1},
		{addr: 0x686a70, size: 4, signed: true, idx: -1},
		{addr: 0x686b75, size: 2, signed: false, idx: 19, count: 51, stride: 33},
		{addr: 0x686b54, size: 2, signed: false, idx: -1},
		{addr: 0x20000ff0, size: 4, signed: false, idx: -1},
		{addr: 0x688291, size: 4, signed: false, idx: 22, count: 1000, stride: 4},
		{addr: 0x6eb70c, size: 4, signed: false, idx: -1},
		{addr: 0x6874ff, size: 1, signed: false, idx: -1},
		{addr: 0x687599, size: 1, signed: false, idx: -1},
		{addr: 0x6840bc, size: 4, signed: false, idx: -1},
		{addr: 0x6875f7, size: 1, signed: false, idx: -1},
		{addr: 0x6875ff, size: 4, signed: false, idx: -1},
		{addr: 0x20004200, size: 4, signed: false, idx: -1},
		{addr: 0x689fcc, size: 4, signed: false, idx: -1},
		{addr: 0x6834d4, size: 4, signed: false, idx: -1},
		{addr: 0x68506b, size: 1, signed: false, idx: -1},
		{addr: 0x60ab48, size: 4, signed: false, idx: -1},
		{addr: 0x6081e4, size: 1, signed: false, idx: -1},
		{addr: 0x659c14, size: 4, signed: false, idx: -1},
		{addr: 0x61f268, size: 4, signed: false, idx: -1},
		{addr: 0x620160, size: 4, signed: false, idx: -1},
		{addr: 0x62017c, size: 4, signed: false, idx: -1},
		{addr: 0x620198, size: 4, signed: false, idx: -1},
		{addr: 0x6201b4, size: 4, signed: false, idx: -1},
		{addr: 0x6201d0, size: 4, signed: false, idx: -1},
		{addr: 0x6201ec, size: 4, signed: false, idx: -1},
		{addr: 0x620208, size: 4, signed: false, idx: -1},
		{addr: 0x620224, size: 4, signed: false, idx: -1},
		{addr: 0x20010b11, size: 4, signed: false, idx: -1},
		{addr: 0x20010b01, size: 4, signed: false, idx: -1},
		{addr: 0x20010a49, size: 4, signed: false, idx: -1},
		{addr: 0x20010a4d, size: 4, signed: false, idx: -1},
		{addr: 0x200109ed, size: 4, signed: false, idx: -1},
		{addr: 0x200109f1, size: 4, signed: false, idx: -1},
		{addr: 0x20002000, size: 1, signed: false, idx: -1},
		{addr: 0x200020fa, size: 4, signed: true, idx: -1},
		{addr: 0x20002103, size: 1, signed: false, idx: -1},
		{addr: 0x6837a1, size: 1, signed: false, idx: -1},
		{addr: 0x20012373, size: 4, signed: false, idx: -1},
		{addr: 0x20012363, size: 4, signed: false, idx: -1},
		{addr: 0x200122ab, size: 4, signed: false, idx: -1},
		{addr: 0x200122af, size: 4, signed: false, idx: -1},
		{addr: 0x2001224f, size: 4, signed: false, idx: -1},
		{addr: 0x20012253, size: 4, signed: false, idx: -1},
		{addr: 0x20002106, size: 1, signed: false, idx: -1},
		{addr: 0x20002200, size: 4, signed: true, idx: -1},
		{addr: 0x20002209, size: 1, signed: false, idx: -1},
		{addr: 0x6838b9, size: 1, signed: false, idx: -1},
		{addr: 0x20013bd5, size: 4, signed: false, idx: -1},
		{addr: 0x20013bc5, size: 4, signed: false, idx: -1},
		{addr: 0x20013b0d, size: 4, signed: false, idx: -1},
		{addr: 0x20013b11, size: 4, signed: false, idx: -1},
		{addr: 0x20013ab1, size: 4, signed: false, idx: -1},
		{addr: 0x20013ab5, size: 4, signed: false, idx: -1},
		{addr: 0x2000220c, size: 1, signed: false, idx: -1},
		{addr: 0x20002306, size: 4, signed: true, idx: -1},
		{addr: 0x2000230f, size: 1, signed: false, idx: -1},
		{addr: 0x6839d1, size: 1, signed: false, idx: -1},
		{addr: 0x20015437, size: 4, signed: false, idx: -1},
		{addr: 0x20015427, size: 4, signed: false, idx: -1},
		{addr: 0x2001536f, size: 4, signed: false, idx: -1},
		{addr: 0x20015373, size: 4, signed: false, idx: -1},
		{addr: 0x20015313, size: 4, signed: false, idx: -1},
		{addr: 0x20015317, size: 4, signed: false, idx: -1},
		{addr: 0x20002312, size: 1, signed: false, idx: -1},
		{addr: 0x2000240c, size: 4, signed: true, idx: -1},
		{addr: 0x20002415, size: 1, signed: false, idx: -1},
		{addr: 0x683ae9, size: 1, signed: false, idx: -1},
		{addr: 0x20016c99, size: 4, signed: false, idx: -1},
		{addr: 0x20016c89, size: 4, signed: false, idx: -1},
		{addr: 0x20016bd1, size: 4, signed: false, idx: -1},
		{addr: 0x20016bd5, size: 4, signed: false, idx: -1},
		{addr: 0x20016b75, size: 4, signed: false, idx: -1},
		{addr: 0x20016b79, size: 4, signed: false, idx: -1},
		{addr: 0x20002418, size: 1, signed: false, idx: -1},
		{addr: 0x20002512, size: 4, signed: true, idx: -1},
		{addr: 0x2000251b, size: 1, signed: false, idx: -1},
		{addr: 0x683c01, size: 1, signed: false, idx: -1},
		{addr: 0x200184fb, size: 4, signed: false, idx: -1},
		{addr: 0x200184eb, size: 4, signed: false, idx: -1},
		{addr: 0x20018433, size: 4, signed: false, idx: -1},
		{addr: 0x20018437, size: 4, signed: false, idx: -1},
		{addr: 0x200183d7, size: 4, signed: false, idx: -1},
		{addr: 0x200183db, size: 4, signed: false, idx: -1},
		{addr: 0x2000251e, size: 1, signed: false, idx: -1},
		{addr: 0x20002618, size: 4, signed: true, idx: -1},
		{addr: 0x20002621, size: 1, signed: false, idx: -1},
		{addr: 0x683d19, size: 1, signed: false, idx: -1},
		{addr: 0x20019d5d, size: 4, signed: false, idx: -1},
		{addr: 0x20019d4d, size: 4, signed: false, idx: -1},
		{addr: 0x20019c95, size: 4, signed: false, idx: -1},
		{addr: 0x20019c99, size: 4, signed: false, idx: -1},
		{addr: 0x20019c39, size: 4, signed: false, idx: -1},
		{addr: 0x20019c3d, size: 4, signed: false, idx: -1},
		{addr: 0x20002624, size: 1, signed: false, idx: -1},
		{addr: 0x2000271e, size: 4, signed: true, idx: -1},
		{addr: 0x20002727, size: 1, signed: false, idx: -1},
		{addr: 0x683e31, size: 1, signed: false, idx: -1},
		{addr: 0x2001b5bf, size: 4, signed: false, idx: -1},
		{addr: 0x2001b5af, size: 4, signed: false, idx: -1},
		{addr: 0x2001b4f7, size: 4, signed: false, idx: -1},
		{addr: 0x2001b4fb, size: 4, signed: false, idx: -1},
		{addr: 0x2001b49b, size: 4, signed: false, idx: -1},
		{addr: 0x2001b49f, size: 4, signed: false, idx: -1},
		{addr: 0x2000272a, size: 1, signed: false, idx: -1},
		{addr: 0x20002824, size: 4, signed: true, idx: -1},
		{addr: 0x2000282d, size: 1, signed: false, idx: -1},
		{addr: 0x683f49, size: 1, signed: false, idx: -1},
		{addr: 0x689f94, size: 4, signed: false, idx: -1},
		{addr: 0x20006904, size: 4, signed: true, idx: -1},
		{addr: 0x20006bc7, size: 1, signed: false, idx: -1},
		{addr: 0x20006bf0, size: 1, signed: false, idx: -1},
		{addr: 0x20006bf1, size: 1, signed: false, idx: -1},
		{addr: 0x20006cc7, size: 1, signed: false, idx: -1},
		{addr: 0x20006cf0, size: 1, signed: false, idx: -1},
		{addr: 0x20006cf1, size: 1, signed: false, idx: -1},
		{addr: 0x20006dc7, size: 1, signed: false, idx: -1},
		{addr: 0x20006df0, size: 1, signed: false, idx: -1},
		{addr: 0x20006df1, size: 1, signed: false, idx: -1},
		{addr: 0x20004508, size: 4, signed: false, idx: -1},
		{addr: 0x20004600, size: 4, signed: false, idx: -1},
		{addr: 0x20004608, size: 4, signed: false, idx: -1},
		{addr: 0x20004614, size: 1, signed: false, idx: -1},
		{addr: 0x2000462b, size: 4, signed: false, idx: -1},
		{addr: 0x2000469f, size: 4, signed: false, idx: -1},
		{addr: 0x200048da, size: 4, signed: false, idx: -1},
		{addr: 0x20004853, size: 1, signed: false, idx: -1},
		{addr: 0x20004855, size: 1, signed: false, idx: -1},
		{addr: 0x200058a4, size: 1, signed: false, idx: -1},
		{addr: 0x200058a7, size: 1, signed: false, idx: -1},
		{addr: 0x20006255, size: 1, signed: false, idx: -1},
		{addr: 0x20004a00, size: 4, signed: false, idx: -1},
		{addr: 0x20004a08, size: 4, signed: false, idx: -1},
		{addr: 0x20004a14, size: 1, signed: false, idx: -1},
		{addr: 0x20004a2b, size: 4, signed: false, idx: -1},
		{addr: 0x20004a9f, size: 4, signed: false, idx: -1},
		{addr: 0x20004cda, size: 4, signed: false, idx: -1},
		{addr: 0x20004c53, size: 1, signed: false, idx: -1},
		{addr: 0x20004c55, size: 1, signed: false, idx: -1},
		{addr: 0x200059a4, size: 1, signed: false, idx: -1},
		{addr: 0x200059a7, size: 1, signed: false, idx: -1},
		{addr: 0x20006555, size: 1, signed: false, idx: -1},
		{addr: 0x20004e00, size: 4, signed: false, idx: -1},
		{addr: 0x20004e08, size: 4, signed: false, idx: -1},
		{addr: 0x20004e14, size: 1, signed: false, idx: -1},
		{addr: 0x20004e2b, size: 4, signed: false, idx: -1},
		{addr: 0x20004e9f, size: 4, signed: false, idx: -1},
		{addr: 0x200050da, size: 4, signed: false, idx: -1},
		{addr: 0x20005053, size: 1, signed: false, idx: -1},
		{addr: 0x20005055, size: 1, signed: false, idx: -1},
		{addr: 0x20005aa4, size: 1, signed: false, idx: -1},
		{addr: 0x20005aa7, size: 1, signed: false, idx: -1},
		{addr: 0x20006855, size: 1, signed: false, idx: -1},
		{addr: 0x20003008, size: 4, signed: false, idx: -1},
		{addr: 0x20003129, size: 1, signed: false, idx: -1},
		{addr: 0x2000319f, size: 4, signed: false, idx: -1},
		{addr: 0x20003308, size: 4, signed: false, idx: -1},
		{addr: 0x20003320, size: 4, signed: false, idx: -1},
		{addr: 0x20003324, size: 4, signed: false, idx: -1},
		{addr: 0x20003229, size: 1, signed: false, idx: -1},
		{addr: 0x2000329f, size: 4, signed: false, idx: -1},
		{addr: 0x20003348, size: 4, signed: false, idx: -1},
		{addr: 0x20003360, size: 4, signed: false, idx: -1},
		{addr: 0x20003364, size: 4, signed: false, idx: -1},
		{addr: 0x659b34, size: 4, signed: true, idx: -1},
		{addr: 0x20007125, size: 1, signed: false, idx: -1},
		{addr: 0x20007168, size: 4, signed: false, idx: -1},
		{addr: 0x20007328, size: 4, signed: false, idx: -1},
		{addr: 0x20007225, size: 1, signed: false, idx: -1},
		{addr: 0x20007268, size: 4, signed: false, idx: -1},
		{addr: 0x20007428, size: 4, signed: false, idx: -1},
		{addr: 0x2000008c, size: 1, signed: false, idx: -1},
		{addr: 0x20000090, size: 1, signed: false, idx: -1},
		{addr: 0x20000094, size: 1, signed: false, idx: -1},
		{addr: 0x200000c4, size: 4, signed: false, idx: -1},
		{addr: 0x200000c8, size: 4, signed: false, idx: -1},
		{addr: 0x200000cc, size: 4, signed: false, idx: -1},
		{addr: 0x20000218, size: 4, signed: false, idx: -1},
		{addr: 0x20000220, size: 4, signed: false, idx: -1},
		{addr: 0x20000224, size: 4, signed: false, idx: -1},
		{addr: 0x20000228, size: 4, signed: false, idx: -1},
		{addr: 0x2000022c, size: 4, signed: false, idx: -1},
		{addr: 0x20000230, size: 4, signed: false, idx: -1},
		{addr: 0x20000234, size: 4, signed: false, idx: -1},
		{addr: 0x20000238, size: 4, signed: false, idx: -1},
		{addr: 0x2000023c, size: 4, signed: false, idx: -1},
		{addr: 0x20000280, size: 4, signed: false, idx: -1},
		{addr: 0x20000284, size: 4, signed: false, idx: -1},
		{addr: 0x20000288, size: 4, signed: false, idx: -1},
		{addr: 0x2000029c, size: 4, signed: false, idx: -1},
		{addr: 0x200002a0, size: 4, signed: false, idx: -1},
		{addr: 0x200002a4, size: 4, signed: false, idx: -1},
		{addr: 0x200002cc, size: 4, signed: false, idx: -1},
		{addr: 0x200002d0, size: 4, signed: false, idx: -1},
		{addr: 0x200002d4, size: 4, signed: false, idx: -1},
		{addr: 0x200002d8, size: 4, signed: false, idx: -1},
		{addr: 0x200002dc, size: 4, signed: false, idx: -1},
		{addr: 0x200002e0, size: 4, signed: false, idx: -1},
		{addr: 0x200002e4, size: 4, signed: false, idx: -1},
		{addr: 0x200002e8, size: 4, signed: false, idx: -1},
		{addr: 0x20000310, size: 4, signed: false, idx: -1},
		{addr: 0x20000314, size: 4, signed: false, idx: -1},
		{addr: 0x20000318, size: 4, signed: false, idx: -1},
		{addr: 0x2000031c, size: 4, signed: false, idx: -1},
		{addr: 0x20000320, size: 4, signed: false, idx: -1},
		{addr: 0x20000324, size: 4, signed: false, idx: -1},
		{addr: 0x20000328, size: 4, signed: false, idx: -1},
		{addr: 0x2000032c, size: 4, signed: false, idx: -1},
		{addr: 0x20000354, size: 4, signed: false, idx: -1},
	},
	out: []mstFld{
		{addr: 0x20001010, size: 4, signed: false, idx: -1},
		{addr: 0x2000190c, size: 4, signed: false, idx: -1},
		{addr: 0x686b54, size: 2, signed: false, idx: -1},
		{addr: 0x686b75, size: 2, signed: false, idx: -1},
		{addr: 0x686b96, size: 2, signed: false, idx: -1},
		{addr: 0x686bb7, size: 2, signed: false, idx: -1},
		{addr: 0x686bd8, size: 2, signed: false, idx: -1},
		{addr: 0x686bf9, size: 2, signed: false, idx: -1},
		{addr: 0x686c1a, size: 2, signed: false, idx: -1},
		{addr: 0x686c3b, size: 2, signed: false, idx: -1},
		{addr: 0x686c5c, size: 2, signed: false, idx: -1},
		{addr: 0x686c7d, size: 2, signed: false, idx: -1},
		{addr: 0x686c9e, size: 2, signed: false, idx: -1},
		{addr: 0x686cbf, size: 2, signed: false, idx: -1},
		{addr: 0x686ce0, size: 2, signed: false, idx: -1},
		{addr: 0x686d01, size: 2, signed: false, idx: -1},
		{addr: 0x686d22, size: 2, signed: false, idx: -1},
		{addr: 0x686d43, size: 2, signed: false, idx: -1},
		{addr: 0x686d64, size: 2, signed: false, idx: -1},
		{addr: 0x686d85, size: 2, signed: false, idx: -1},
		{addr: 0x686da6, size: 2, signed: false, idx: -1},
		{addr: 0x686dc7, size: 2, signed: false, idx: -1},
		{addr: 0x686de8, size: 2, signed: false, idx: -1},
		{addr: 0x686e09, size: 2, signed: false, idx: -1},
		{addr: 0x686e2a, size: 2, signed: false, idx: -1},
		{addr: 0x686e4b, size: 2, signed: false, idx: -1},
		{addr: 0x686e6c, size: 2, signed: false, idx: -1},
		{addr: 0x686e8d, size: 2, signed: false, idx: -1},
		{addr: 0x686eae, size: 2, signed: false, idx: -1},
		{addr: 0x686ecf, size: 2, signed: false, idx: -1},
		{addr: 0x686ef0, size: 2, signed: false, idx: -1},
		{addr: 0x686f11, size: 2, signed: false, idx: -1},
		{addr: 0x686f32, size: 2, signed: false, idx: -1},
		{addr: 0x686f53, size: 2, signed: false, idx: -1},
		{addr: 0x686f74, size: 2, signed: false, idx: -1},
		{addr: 0x686f95, size: 2, signed: false, idx: -1},
		{addr: 0x686fb6, size: 2, signed: false, idx: -1},
		{addr: 0x686fd7, size: 2, signed: false, idx: -1},
		{addr: 0x686ff8, size: 2, signed: false, idx: -1},
		{addr: 0x687019, size: 2, signed: false, idx: -1},
		{addr: 0x68703a, size: 2, signed: false, idx: -1},
		{addr: 0x68705b, size: 2, signed: false, idx: -1},
		{addr: 0x68707c, size: 2, signed: false, idx: -1},
		{addr: 0x68709d, size: 2, signed: false, idx: -1},
		{addr: 0x6870be, size: 2, signed: false, idx: -1},
		{addr: 0x6870df, size: 2, signed: false, idx: -1},
		{addr: 0x687100, size: 2, signed: false, idx: -1},
		{addr: 0x687121, size: 2, signed: false, idx: -1},
		{addr: 0x687142, size: 2, signed: false, idx: -1},
		{addr: 0x687163, size: 2, signed: false, idx: -1},
		{addr: 0x687184, size: 2, signed: false, idx: -1},
		{addr: 0x6871a5, size: 2, signed: false, idx: -1},
		{addr: 0x6871c6, size: 2, signed: false, idx: -1},
		{addr: 0x6871e7, size: 2, signed: false, idx: -1},
		{addr: 0x688291, size: 4, signed: false, idx: 22, count: 1000, stride: 4},
		{addr: 0x688291, size: 4, signed: false, idx: -1},
		{addr: 0x688295, size: 4, signed: false, idx: -1},
		{addr: 0x688299, size: 4, signed: false, idx: -1},
		{addr: 0x688a61, size: 4, signed: false, idx: -1},
		{addr: 0x689229, size: 4, signed: false, idx: -1},
		{addr: 0x68922d, size: 4, signed: false, idx: -1},
		{addr: 0x6850b0, size: 4, signed: false, idx: -1},
		{addr: 0x683f94, size: 1, signed: false, idx: -1},
		{addr: 0x6836a8, size: 4, signed: false, idx: -1},
		{addr: 0x200034f4, size: 4, signed: false, idx: -1},
		{addr: 0x200036dc, size: 4, signed: false, idx: -1},
		{addr: 0x200036e0, size: 4, signed: false, idx: -1},
		{addr: 0x687511, size: 4, signed: false, idx: -1},
		{addr: 0x687591, size: 4, signed: false, idx: -1},
		{addr: 0x6874ff, size: 1, signed: false, idx: -1},
		{addr: 0x689fcc, size: 4, signed: false, idx: -1},
		{addr: 0x6834d4, size: 4, signed: false, idx: -1},
		{addr: 0x68506b, size: 1, signed: false, idx: -1},
		{addr: 0x61f268, size: 4, signed: false, idx: -1},
		{addr: 0x620160, size: 4, signed: false, idx: -1},
		{addr: 0x62017c, size: 4, signed: false, idx: -1},
		{addr: 0x620198, size: 4, signed: false, idx: -1},
		{addr: 0x6201b4, size: 4, signed: false, idx: -1},
		{addr: 0x6201d0, size: 4, signed: false, idx: -1},
		{addr: 0x6201ec, size: 4, signed: false, idx: -1},
		{addr: 0x620208, size: 4, signed: false, idx: -1},
		{addr: 0x620224, size: 4, signed: false, idx: -1},
		{addr: 0x60ab48, size: 4, signed: false, idx: -1},
		{addr: 0x6081e4, size: 1, signed: false, idx: -1},
		{addr: 0x659c14, size: 4, signed: false, idx: -1},
		{addr: 0x20004200, size: 4, signed: false, idx: -1},
		{addr: 0x683f95, size: 1, signed: false, idx: -1},
		{addr: 0x683f96, size: 1, signed: false, idx: -1},
		{addr: 0x683f97, size: 1, signed: false, idx: -1},
		{addr: 0x2000462b, size: 4, signed: false, idx: -1},
		{addr: 0x20004853, size: 1, signed: false, idx: -1},
		{addr: 0x20004855, size: 1, signed: false, idx: -1},
		{addr: 0x20006255, size: 1, signed: false, idx: -1},
		{addr: 0x20004a2b, size: 4, signed: false, idx: -1},
		{addr: 0x20004c53, size: 1, signed: false, idx: -1},
		{addr: 0x20004c55, size: 1, signed: false, idx: -1},
		{addr: 0x20006555, size: 1, signed: false, idx: -1},
		{addr: 0x20004e2b, size: 4, signed: false, idx: -1},
		{addr: 0x20005053, size: 1, signed: false, idx: -1},
		{addr: 0x20005055, size: 1, signed: false, idx: -1},
		{addr: 0x20006855, size: 1, signed: false, idx: -1},
		{addr: 0x20003129, size: 1, signed: false, idx: -1},
		{addr: 0x20003229, size: 1, signed: false, idx: -1},
		{addr: 0x20007125, size: 1, signed: false, idx: -1},
		{addr: 0x20007328, size: 4, signed: false, idx: -1},
		{addr: 0x20007225, size: 1, signed: false, idx: -1},
		{addr: 0x20007428, size: 4, signed: false, idx: -1},
		{addr: 0x200109ed, size: 4, signed: false, idx: -1},
		{addr: 0x2001224f, size: 4, signed: false, idx: -1},
		{addr: 0x20013ab1, size: 4, signed: false, idx: -1},
		{addr: 0x20015313, size: 4, signed: false, idx: -1},
		{addr: 0x20016b75, size: 4, signed: false, idx: -1},
		{addr: 0x200183d7, size: 4, signed: false, idx: -1},
		{addr: 0x20019c39, size: 4, signed: false, idx: -1},
		{addr: 0x2001b49b, size: 4, signed: false, idx: -1},
		{addr: 0x20000010, size: 4, signed: false, idx: -1},
		{addr: 0x20000014, size: 4, signed: false, idx: -1},
		{addr: 0x20000018, size: 4, signed: false, idx: -1},
		{addr: 0x2000001c, size: 4, signed: false, idx: -1},
		{addr: 0x20000020, size: 4, signed: false, idx: -1},
		{addr: 0x20000024, size: 4, signed: false, idx: -1},
		{addr: 0x20000028, size: 4, signed: false, idx: -1},
		{addr: 0x2000002c, size: 4, signed: false, idx: -1},
		{addr: 0x20000030, size: 4, signed: false, idx: -1},
		{addr: 0x20000034, size: 4, signed: false, idx: -1},
		{addr: 0x20000038, size: 4, signed: false, idx: -1},
		{addr: 0x2000003c, size: 4, signed: false, idx: -1},
		{addr: 0x20000040, size: 4, signed: false, idx: -1},
		{addr: 0x20000044, size: 4, signed: false, idx: -1},
		{addr: 0x20000048, size: 4, signed: false, idx: -1},
		{addr: 0x2000004c, size: 4, signed: false, idx: -1},
		{addr: 0x20000050, size: 4, signed: false, idx: -1},
		{addr: 0x20000054, size: 4, signed: false, idx: -1},
		{addr: 0x20000058, size: 4, signed: false, idx: -1},
		{addr: 0x2000005c, size: 4, signed: false, idx: -1},
		{addr: 0x20000060, size: 4, signed: false, idx: -1},
		{addr: 0x20000064, size: 4, signed: false, idx: -1},
		{addr: 0x20000068, size: 4, signed: false, idx: -1},
		{addr: 0x2000006c, size: 4, signed: false, idx: -1},
		{addr: 0x20000070, size: 4, signed: false, idx: -1},
		{addr: 0x20000074, size: 4, signed: false, idx: -1},
		{addr: 0x20000078, size: 4, signed: false, idx: -1},
		{addr: 0x2000007c, size: 4, signed: false, idx: -1},
		{addr: 0x20000080, size: 4, signed: false, idx: -1},
		{addr: 0x20000084, size: 4, signed: false, idx: -1},
		{addr: 0x20000088, size: 4, signed: false, idx: -1},
		{addr: 0x20000098, size: 4, signed: false, idx: -1},
		{addr: 0x2000009c, size: 4, signed: false, idx: -1},
		{addr: 0x200000a0, size: 4, signed: false, idx: -1},
		{addr: 0x200000a4, size: 4, signed: false, idx: -1},
		{addr: 0x200000a8, size: 4, signed: false, idx: -1},
		{addr: 0x200000ac, size: 4, signed: false, idx: -1},
		{addr: 0x200000b0, size: 4, signed: false, idx: -1},
		{addr: 0x200000b4, size: 4, signed: false, idx: -1},
		{addr: 0x200000b8, size: 4, signed: false, idx: -1},
		{addr: 0x200000bc, size: 4, signed: false, idx: -1},
		{addr: 0x200000c0, size: 4, signed: false, idx: -1},
		{addr: 0x200000d0, size: 4, signed: false, idx: -1},
		{addr: 0x200000d4, size: 4, signed: false, idx: -1},
		{addr: 0x200000d8, size: 4, signed: false, idx: -1},
		{addr: 0x200000dc, size: 4, signed: false, idx: -1},
		{addr: 0x200000e0, size: 4, signed: false, idx: -1},
		{addr: 0x200000e4, size: 4, signed: false, idx: -1},
		{addr: 0x200000e8, size: 4, signed: false, idx: -1},
		{addr: 0x200000ec, size: 4, signed: false, idx: -1},
		{addr: 0x200000f0, size: 4, signed: false, idx: -1},
		{addr: 0x200000f4, size: 4, signed: false, idx: -1},
		{addr: 0x200000f8, size: 4, signed: false, idx: -1},
		{addr: 0x200000fc, size: 4, signed: false, idx: -1},
		{addr: 0x20000100, size: 4, signed: false, idx: -1},
		{addr: 0x20000104, size: 4, signed: false, idx: -1},
		{addr: 0x20000108, size: 4, signed: false, idx: -1},
		{addr: 0x2000010c, size: 4, signed: false, idx: -1},
		{addr: 0x20000110, size: 4, signed: false, idx: -1},
		{addr: 0x20000114, size: 4, signed: false, idx: -1},
		{addr: 0x20000118, size: 4, signed: false, idx: -1},
		{addr: 0x2000011c, size: 4, signed: false, idx: -1},
		{addr: 0x20000120, size: 4, signed: false, idx: -1},
		{addr: 0x20000124, size: 4, signed: false, idx: -1},
		{addr: 0x20000128, size: 4, signed: false, idx: -1},
		{addr: 0x2000012c, size: 4, signed: false, idx: -1},
		{addr: 0x20000130, size: 4, signed: false, idx: -1},
		{addr: 0x20000134, size: 4, signed: false, idx: -1},
		{addr: 0x20000138, size: 4, signed: false, idx: -1},
		{addr: 0x2000013c, size: 4, signed: false, idx: -1},
		{addr: 0x20000140, size: 4, signed: false, idx: -1},
		{addr: 0x20000144, size: 4, signed: false, idx: -1},
		{addr: 0x20000148, size: 4, signed: false, idx: -1},
		{addr: 0x2000014c, size: 4, signed: false, idx: -1},
		{addr: 0x20000150, size: 4, signed: false, idx: -1},
		{addr: 0x20000154, size: 4, signed: false, idx: -1},
		{addr: 0x20000158, size: 4, signed: false, idx: -1},
		{addr: 0x2000015c, size: 4, signed: false, idx: -1},
		{addr: 0x20000160, size: 4, signed: false, idx: -1},
		{addr: 0x20000164, size: 4, signed: false, idx: -1},
		{addr: 0x20000168, size: 4, signed: false, idx: -1},
		{addr: 0x2000016c, size: 4, signed: false, idx: -1},
		{addr: 0x20000170, size: 4, signed: false, idx: -1},
		{addr: 0x20000174, size: 4, signed: false, idx: -1},
		{addr: 0x20000178, size: 4, signed: false, idx: -1},
		{addr: 0x2000017c, size: 4, signed: false, idx: -1},
		{addr: 0x20000180, size: 4, signed: false, idx: -1},
		{addr: 0x20000184, size: 4, signed: false, idx: -1},
		{addr: 0x20000188, size: 4, signed: false, idx: -1},
		{addr: 0x2000018c, size: 4, signed: false, idx: -1},
		{addr: 0x20000190, size: 4, signed: false, idx: -1},
		{addr: 0x20000194, size: 4, signed: false, idx: -1},
		{addr: 0x20000198, size: 4, signed: false, idx: -1},
		{addr: 0x2000019c, size: 4, signed: false, idx: -1},
		{addr: 0x200001a0, size: 4, signed: false, idx: -1},
		{addr: 0x200001a4, size: 4, signed: false, idx: -1},
		{addr: 0x200001a8, size: 4, signed: false, idx: -1},
		{addr: 0x200001ac, size: 4, signed: false, idx: -1},
		{addr: 0x200001b0, size: 4, signed: false, idx: -1},
		{addr: 0x200001b4, size: 4, signed: false, idx: -1},
		{addr: 0x200001b8, size: 4, signed: false, idx: -1},
		{addr: 0x200001bc, size: 4, signed: false, idx: -1},
		{addr: 0x200001c0, size: 4, signed: false, idx: -1},
		{addr: 0x200001c4, size: 4, signed: false, idx: -1},
		{addr: 0x200001c8, size: 4, signed: false, idx: -1},
		{addr: 0x200001cc, size: 4, signed: false, idx: -1},
		{addr: 0x200001d0, size: 4, signed: false, idx: -1},
		{addr: 0x200001d4, size: 4, signed: false, idx: -1},
		{addr: 0x200001d8, size: 4, signed: false, idx: -1},
		{addr: 0x200001dc, size: 4, signed: false, idx: -1},
		{addr: 0x200001e0, size: 4, signed: false, idx: -1},
		{addr: 0x200001e4, size: 4, signed: false, idx: -1},
		{addr: 0x200001e8, size: 4, signed: false, idx: -1},
		{addr: 0x200001ec, size: 4, signed: false, idx: -1},
		{addr: 0x200001f0, size: 4, signed: false, idx: -1},
		{addr: 0x200001f4, size: 4, signed: false, idx: -1},
		{addr: 0x200001f8, size: 4, signed: false, idx: -1},
		{addr: 0x200001fc, size: 4, signed: false, idx: -1},
		{addr: 0x20000200, size: 4, signed: false, idx: -1},
		{addr: 0x20000204, size: 4, signed: false, idx: -1},
		{addr: 0x20000208, size: 4, signed: false, idx: -1},
		{addr: 0x2000020c, size: 4, signed: false, idx: -1},
		{addr: 0x20000210, size: 4, signed: false, idx: -1},
		{addr: 0x20000214, size: 4, signed: false, idx: -1},
		{addr: 0x2000021c, size: 4, signed: false, idx: -1},
		{addr: 0x20000240, size: 4, signed: false, idx: -1},
		{addr: 0x20000244, size: 4, signed: false, idx: -1},
		{addr: 0x20000248, size: 4, signed: false, idx: -1},
		{addr: 0x2000024c, size: 4, signed: false, idx: -1},
		{addr: 0x20000250, size: 4, signed: false, idx: -1},
		{addr: 0x20000254, size: 4, signed: false, idx: -1},
		{addr: 0x20000258, size: 4, signed: false, idx: -1},
		{addr: 0x2000025c, size: 4, signed: false, idx: -1},
		{addr: 0x20000260, size: 4, signed: false, idx: -1},
		{addr: 0x20000264, size: 4, signed: false, idx: -1},
		{addr: 0x20000268, size: 4, signed: false, idx: -1},
		{addr: 0x2000026c, size: 4, signed: false, idx: -1},
		{addr: 0x20000270, size: 4, signed: false, idx: -1},
		{addr: 0x20000274, size: 4, signed: false, idx: -1},
		{addr: 0x20000278, size: 4, signed: false, idx: -1},
		{addr: 0x2000027c, size: 4, signed: false, idx: -1},
		{addr: 0x2000028c, size: 4, signed: false, idx: -1},
		{addr: 0x20000290, size: 4, signed: false, idx: -1},
		{addr: 0x20000294, size: 4, signed: false, idx: -1},
		{addr: 0x20000298, size: 4, signed: false, idx: -1},
		{addr: 0x200002a8, size: 4, signed: false, idx: -1},
		{addr: 0x200002ac, size: 4, signed: false, idx: -1},
		{addr: 0x200002b0, size: 4, signed: false, idx: -1},
		{addr: 0x200002b4, size: 4, signed: false, idx: -1},
		{addr: 0x200002b8, size: 4, signed: false, idx: -1},
		{addr: 0x200002bc, size: 4, signed: false, idx: -1},
		{addr: 0x200002c0, size: 4, signed: false, idx: -1},
		{addr: 0x200002c4, size: 4, signed: false, idx: -1},
		{addr: 0x200002c8, size: 4, signed: false, idx: -1},
		{addr: 0x200002ec, size: 4, signed: false, idx: -1},
		{addr: 0x200002f0, size: 4, signed: false, idx: -1},
		{addr: 0x200002f4, size: 4, signed: false, idx: -1},
		{addr: 0x200002f8, size: 4, signed: false, idx: -1},
		{addr: 0x200002fc, size: 4, signed: false, idx: -1},
		{addr: 0x20000300, size: 4, signed: false, idx: -1},
		{addr: 0x20000304, size: 4, signed: false, idx: -1},
		{addr: 0x20000308, size: 4, signed: false, idx: -1},
		{addr: 0x2000030c, size: 4, signed: false, idx: -1},
		{addr: 0x20000330, size: 4, signed: false, idx: -1},
		{addr: 0x20000334, size: 4, signed: false, idx: -1},
		{addr: 0x20000338, size: 4, signed: false, idx: -1},
		{addr: 0x2000033c, size: 4, signed: false, idx: -1},
		{addr: 0x20000340, size: 4, signed: false, idx: -1},
		{addr: 0x20000344, size: 4, signed: false, idx: -1},
		{addr: 0x20000348, size: 4, signed: false, idx: -1},
		{addr: 0x2000034c, size: 4, signed: false, idx: -1},
		{addr: 0x20000350, size: 4, signed: false, idx: -1},
	},
}

var cxdIn = []string{"arg0", "f683fce", "f683fcd", "mode", "ui.flags", "ui.f327", "ui.f2dc", "ui.f2e0", "G.xp", "G.bonus", "G.fa54", "G.f90c", "G.fa61", "G.f910", "G.f918", "G.form0", "G.form32", "form0", "form32", "map", "mapKills", "mapKillsM1", "seenIdx", "seenVal", "clock", "f6874ff", "f687599", "f6840bc", "f6875f7", "d6875ff", "w652", "d689fcc", "d6834d4", "f68506b", "d60ab48", "f6081e4", "d659c14", "d61f268", "widget10", "widget11", "widget12", "widget13", "widget14", "widget15", "widget16", "widget17", "c0.hp", "c0.status", "c0.a49", "c0.a4d", "c0.xp", "c0.next", "q0.present", "q0.item", "q0.f103", "busy0", "c1.hp", "c1.status", "c1.a49", "c1.a4d", "c1.xp", "c1.next", "q1.present", "q1.item", "q1.f103", "busy1", "c2.hp", "c2.status", "c2.a49", "c2.a4d", "c2.xp", "c2.next", "q2.present", "q2.item", "q2.f103", "busy2", "c3.hp", "c3.status", "c3.a49", "c3.a4d", "c3.xp", "c3.next", "q3.present", "q3.item", "q3.f103", "busy3", "c4.hp", "c4.status", "c4.a49", "c4.a4d", "c4.xp", "c4.next", "q4.present", "q4.item", "q4.f103", "busy4", "c5.hp", "c5.status", "c5.a49", "c5.a4d", "c5.xp", "c5.next", "q5.present", "q5.item", "q5.f103", "busy5", "c6.hp", "c6.status", "c6.a49", "c6.a4d", "c6.xp", "c6.next", "q6.present", "q6.item", "q6.f103", "busy6", "c7.hp", "c7.status", "c7.a49", "c7.a4d", "c7.xp", "c7.next", "q7.present", "q7.item", "q7.f103", "busy7", "itemTable", "eqt.count", "eq0.fc7", "eq0.ff0", "eq0.ff1", "eq1.fc7", "eq1.ff0", "eq1.ff1", "eq2.fc7", "eq2.ff0", "eq2.ff1", "mcount", "mon0.id", "mon0.species", "mon0.f14", "mon0.hp", "mon0.f9f", "mon0.f2da", "mon0.f253", "mon0.f255", "rstate0.a4", "rstate0.a7", "og0.f255", "mon1.id", "mon1.species", "mon1.f14", "mon1.hp", "mon1.f9f", "mon1.f2da", "mon1.f253", "mon1.f255", "rstate1.a4", "rstate1.a7", "og1.f255", "mon2.id", "mon2.species", "mon2.f14", "mon2.hp", "mon2.f9f", "mon2.f2da", "mon2.f253", "mon2.f255", "rstate2.a4", "rstate2.a7", "og2.f255", "ngroups", "grp0.f29", "grp0.leader", "grp0.n", "grp0.id0", "grp0.id1", "grp1.f29", "grp1.leader", "grp1.n", "grp1.id0", "grp1.id1", "anims", "anim0.f25", "anim0.timer", "tobj0.stamp", "anim1.f25", "anim1.timer", "tobj1.stamp", "w56af20_683f95", "w56af20_683f96", "w56af20_683f97", "r4e5150_0", "r4e5150_1", "r4e5150_2", "r4eee80", "w4eef10_xp0", "w4eef10_xp1", "w4eef10_xp2", "w4eef10_xp3", "w4eef10_xp4", "w4eef10_xp5", "w4eef10_xp6", "w4eef10_xp7", "w4e4280_hp_0", "w4e4280_hp_1", "w4e4280_hp_2", "w4e46f0_xp_0", "w4e46f0_xp_1", "w4e46f0_xp_2", "r4e79a0_0", "r4e79a0_1", "r4e79a0_2", "r4e79a0_3", "r4e79a0_4", "r4e79a0_5", "r4e79a0_6", "r4e79a0_7", "r4ef3c0_0", "r4ef3c0_1", "r4ef3c0_2", "r4ef3c0_3", "r4ef3c0_4", "r4ef3c0_5", "r4ef3c0_6", "r4ef3c0_7", "tick"}

var cxdOut = []string{"G.xp", "G.f90c", "mapKills-1", "mapKills0", "mapKills1", "mapKills2", "mapKills3", "mapKills4", "mapKills5", "mapKills6", "mapKills7", "mapKills8", "mapKills9", "mapKills10", "mapKills11", "mapKills12", "mapKills13", "mapKills14", "mapKills15", "mapKills16", "mapKills17", "mapKills18", "mapKills19", "mapKills20", "mapKills21", "mapKills22", "mapKills23", "mapKills24", "mapKills25", "mapKills26", "mapKills27", "mapKills28", "mapKills29", "mapKills30", "mapKills31", "mapKills32", "mapKills33", "mapKills34", "mapKills35", "mapKills36", "mapKills37", "mapKills38", "mapKills39", "mapKills40", "mapKills41", "mapKills42", "mapKills43", "mapKills44", "mapKills45", "mapKills46", "mapKills47", "mapKills48", "mapKills49", "mapKills50", "seenVal", "seen0", "seen1", "seen2", "seen500", "seen998", "seen999", "d6850b0", "combat", "battle", "ui.flags", "ui.f2dc", "ui.f2e0", "form0", "form32", "f6874ff", "d689fcc", "d6834d4", "f68506b", "d61f268", "widget10", "widget11", "widget12", "widget13", "widget14", "widget15", "widget16", "widget17", "d60ab48", "f6081e4", "d659c14", "w652", "f683f95", "f683f96", "f683f97", "mon0.hp", "mon0.f253", "mon0.f255", "og0.f255", "mon1.hp", "mon1.f253", "mon1.f255", "og1.f255", "mon2.hp", "mon2.f253", "mon2.f255", "og2.f255", "grp0.f29", "grp1.f29", "anim0.f25", "tobj0.stamp", "anim1.f25", "tobj1.stamp", "c0.xp", "c1.xp", "c2.xp", "c3.xp", "c4.xp", "c5.xp", "c6.xp", "c7.xp", "s4f0630_calls", "s53ae00_calls", "s5a1890_calls", "s5a3470_calls", "s5b1c80_calls", "s5a24a0_calls", "s56aab0_calls", "s452c90_calls", "s53a320_a1_0", "s53a320_calls", "s58f6b0_a1_0", "s58f6b0_calls", "s482990_a1_0", "s482990_calls", "s562a50_a1_0", "s562a50_calls", "s4f21e0_a1_0", "s4f21e0_calls", "s58ac00_a1_0", "s58ac00_a1_1", "s58ac00_a1_2", "s58ac00_a1_3", "s58ac00_a2_0", "s58ac00_a2_1", "s58ac00_a2_2", "s58ac00_a2_3", "s58ac00_a3_0", "s58ac00_a3_1", "s58ac00_a3_2", "s58ac00_a3_3", "s58ac00_calls", "s56af20_a1_0", "s56af20_calls", "s4263f0_a1_0", "s4263f0_a1_1", "s4263f0_a2_0", "s4263f0_a2_1", "s4263f0_a3_0", "s4263f0_a3_1", "s4263f0_a4_0", "s4263f0_a4_1", "s4263f0_calls", "s4e5150_a1_0", "s4e5150_a1_1", "s4e5150_a1_2", "s4e5150_calls", "s58ab60_a3_0", "s58ab60_a3_1", "s58ab60_a3_2", "s58ab60_a4_0", "s58ab60_a4_1", "s58ab60_a4_2", "s58ab60_calls", "s4e4db0_a1_0", "s4e4db0_a1_1", "s4e4db0_a1_2", "s4e4db0_calls", "s4e3af0_a1_0", "s4e3af0_a1_1", "s4e3af0_a1_2", "s4e3af0_calls", "s53a930_a1_0", "s53a930_a1_1", "s53a930_a1_2", "s53a930_a1_3", "s53a930_a1_4", "s53a930_a1_5", "s53a930_a1_6", "s53a930_a1_7", "s53a930_a2_0", "s53a930_a2_1", "s53a930_a2_2", "s53a930_a2_3", "s53a930_a2_4", "s53a930_a2_5", "s53a930_a2_6", "s53a930_a2_7", "s53a930_calls", "s4f2380_a1_0", "s4f2380_a1_1", "s4f2380_a1_2", "s4f2380_a1_3", "s4f2380_a1_4", "s4f2380_a1_5", "s4f2380_a1_6", "s4f2380_a1_7", "s4f2380_calls", "s4f23d0_a1_0", "s4f23d0_a1_1", "s4f23d0_a1_2", "s4f23d0_a1_3", "s4f23d0_a1_4", "s4f23d0_a1_5", "s4f23d0_a1_6", "s4f23d0_a1_7", "s4f23d0_calls", "s52e5c0_a1_0", "s52e5c0_a1_1", "s52e5c0_a1_2", "s52e5c0_a1_3", "s52e5c0_a1_4", "s52e5c0_a1_5", "s52e5c0_a1_6", "s52e5c0_a1_7", "s52e5c0_a1_8", "s52e5c0_a1_9", "s52e5c0_a2_0", "s52e5c0_a2_1", "s52e5c0_a2_2", "s52e5c0_a2_3", "s52e5c0_a2_4", "s52e5c0_a2_5", "s52e5c0_a2_6", "s52e5c0_a2_7", "s52e5c0_a2_8", "s52e5c0_a2_9", "s52e5c0_calls", "s524540_calls", "s552530_calls", "s53cd60_calls", "s5244a0_a1_0", "s5244a0_calls", "s5246c0_a1_0", "s5246c0_calls", "s4eee80_calls", "s4eef10_a1_0", "s4eef10_a2_0", "s4eef10_calls", "s4ee9d0_a1_0", "s4ee9d0_a1_1", "s4ee9d0_a1_2", "s4ee9d0_a1_3", "s4ee9d0_a1_4", "s4ee9d0_a1_5", "s4ee9d0_a1_6", "s4ee9d0_a1_7", "s4ee9d0_calls", "s5248d0_a1_0", "s5248d0_a1_1", "s5248d0_a1_2", "s5248d0_calls", "s4e4280_a1_0", "s4e4280_a1_1", "s4e4280_a1_2", "s4e4280_calls", "s4e46f0_a1_0", "s4e46f0_a1_1", "s4e46f0_a1_2", "s4e46f0_calls", "s4e4500_a1_0", "s4e4500_a1_1", "s4e4500_a1_2", "s4e4500_a1_3", "s4e4500_calls", "s4e79a0_a1_0", "s4e79a0_a1_1", "s4e79a0_a1_2", "s4e79a0_a1_3", "s4e79a0_a1_4", "s4e79a0_a1_5", "s4e79a0_a1_6", "s4e79a0_a1_7", "s4e79a0_calls", "s4ef3c0_a1_0", "s4ef3c0_a1_1", "s4ef3c0_a1_2", "s4ef3c0_a1_3", "s4ef3c0_a1_4", "s4ef3c0_a1_5", "s4ef3c0_a1_6", "s4ef3c0_a1_7", "s4ef3c0_calls"}

var cxdStubs = map[uint32]*mcaStub{
	0x4f0630: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20000010, wr: []mcaWrite{}},
	0x53ae00: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20000014, wr: []mcaWrite{}},
	0x5a1890: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20000018, wr: []mcaWrite{}},
	0x5a3470: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x2000001c, wr: []mcaWrite{}},
	0x5b1c80: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20000020, wr: []mcaWrite{}},
	0x5a24a0: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20000024, wr: []mcaWrite{}},
	0x56aab0: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20000028, wr: []mcaWrite{}},
	0x452c90: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x2000002c, wr: []mcaWrite{}},
	0x53a320: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000030, 1}}, cnt: 0x20000034, wr: []mcaWrite{}},
	0x58f6b0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000038, 1}}, cnt: 0x2000003c, wr: []mcaWrite{}},
	0x482990: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000040, 1}}, cnt: 0x20000044, wr: []mcaWrite{}},
	0x562a50: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000048, 1}}, cnt: 0x2000004c, wr: []mcaWrite{}},
	0x4f21e0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000050, 1}}, cnt: 0x20000054, wr: []mcaWrite{}},
	0x58ac00: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000058, 4}, {2, 4, 0x20000068, 4}, {3, 4, 0x20000078, 4}}, cnt: 0x20000088, wr: []mcaWrite{}},
	0x56af20: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000098, 1}}, cnt: 0x2000009c, wr: []mcaWrite{{0, 0x683f95, 1, 0x2000008c, 0}, {0, 0x683f96, 1, 0x20000090, 0}, {0, 0x683f97, 1, 0x20000094, 0}}},
	0x4263f0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200000a0, 2}, {2, 4, 0x200000a8, 2}, {3, 4, 0x200000b0, 2}, {4, 4, 0x200000b8, 2}}, cnt: 0x200000c0, wr: []mcaWrite{}},
	0x4e5150: {ret: 0x200000c4, retN: 3, rec: [][4]uint32{{1, 4, 0x200000d0, 3}}, cnt: 0x200000dc, wr: []mcaWrite{}},
	0x58ab60: {ret: 0x0, retN: 0, rec: [][4]uint32{{3, 4, 0x200000e0, 3}, {4, 4, 0x200000ec, 3}}, cnt: 0x200000f8, wr: []mcaWrite{}},
	0x4e4db0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200000fc, 3}}, cnt: 0x20000108, wr: []mcaWrite{}},
	0x4e3af0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000010c, 3}}, cnt: 0x20000118, wr: []mcaWrite{}},
	0x53a930: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000011c, 8}, {2, 4, 0x2000013c, 8}}, cnt: 0x2000015c, wr: []mcaWrite{}},
	0x4f2380: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000160, 8}}, cnt: 0x20000180, wr: []mcaWrite{}},
	0x4f23d0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000184, 8}}, cnt: 0x200001a4, wr: []mcaWrite{}},
	0x52e5c0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200001a8, 10}, {2, 4, 0x200001d0, 10}}, cnt: 0x200001f8, wr: []mcaWrite{}},
	0x524540: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x200001fc, wr: []mcaWrite{}},
	0x552530: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20000200, wr: []mcaWrite{}},
	0x53cd60: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20000204, wr: []mcaWrite{}},
	0x5244a0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000208, 1}}, cnt: 0x2000020c, wr: []mcaWrite{}},
	0x5246c0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000210, 1}}, cnt: 0x20000214, wr: []mcaWrite{}},
	0x4eee80: {ret: 0x20000218, retN: 0, rec: [][4]uint32{}, cnt: 0x2000021c, wr: []mcaWrite{}},
	0x4eef10: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000240, 1}, {2, 4, 0x20000244, 1}}, cnt: 0x20000248, wr: []mcaWrite{{0, 0x200109ed, 4, 0x20000220, 0}, {0, 0x2001224f, 4, 0x20000224, 0}, {0, 0x20013ab1, 4, 0x20000228, 0}, {0, 0x20015313, 4, 0x2000022c, 0}, {0, 0x20016b75, 4, 0x20000230, 0}, {0, 0x200183d7, 4, 0x20000234, 0}, {0, 0x20019c39, 4, 0x20000238, 0}, {0, 0x2001b49b, 4, 0x2000023c, 0}}},
	0x4ee9d0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000024c, 8}}, cnt: 0x2000026c, wr: []mcaWrite{}},
	0x5248d0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000270, 3}}, cnt: 0x2000027c, wr: []mcaWrite{}},
	0x4e4280: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000028c, 3}}, cnt: 0x20000298, wr: []mcaWrite{{1, 0x2b, 4, 0x20000280, 3}}},
	0x4e46f0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200002a8, 3}}, cnt: 0x200002b4, wr: []mcaWrite{{0, 0x20001010, 4, 0x2000029c, 3}}},
	0x4e4500: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200002b8, 4}}, cnt: 0x200002c8, wr: []mcaWrite{}},
	0x4e79a0: {ret: 0x200002cc, retN: 8, rec: [][4]uint32{{1, 4, 0x200002ec, 8}}, cnt: 0x2000030c, wr: []mcaWrite{}},
	0x4ef3c0: {ret: 0x20000310, retN: 8, rec: [][4]uint32{{1, 4, 0x20000330, 8}}, cnt: 0x20000350, wr: []mcaWrite{}},
}

// cxdPark4eee80 is the first boundary row of re/emu/cases/004eee80.json (xp eligible count).
var cxdPark4eee80 = [24]int64{0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0}

// cxdPark4eef10 is the first boundary row of re/emu/cases/004eef10.json (xp award).
var cxdPark4eef10 = [85]int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// cxdPark4ef3c0 is the first boundary row of re/emu/cases/004ef3c0.json (level-up eligibility).
var cxdPark4ef3c0 = [41]int64{0, 0, 1, 0, 100, 100, 0, 0, 18, 0, 1, 0, 0, 18, 0, 1, 0, 0, 18, 0, 1, 0, 0, 18, 0, 1, 0, 0, 18, 0, 1, 0, 0, 18, 0, 1, 0, 0, 18, 0, 1}

// cxdPark4e79a0 is the first boundary row of re/emu/cases/004e79a0.json (pc action eligibility).
var cxdPark4e79a0 = [264]int64{0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 1, 1, 536936448, 0, 1, 16, 0, 1, 2, 0, -1, -1, 1, 0, 1, 0, 0, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 0, 0, 0, 0, 100, 20, 0, 1, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 5, 1, 10, 10, 10, 10, 10, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 65535, 4294967295, 3, 255, 0, 255, 0, 255, 0, 255, 0, 1, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 1, 1, 1, 0, 0, 0, 0, 0, 0}

// cxdPark4e4280 is the first boundary row of re/emu/cases/004e4280.json (monster remove state).
var cxdPark4e4280 = [32]int64{0, 0, 31, 80, 20, 42, 3, 1065353216, 1065353216, 1065353216, 1073741824, 1073741824, 1073741824, 1, 4294967295, 1, 536883200, 0, 2, 2, 1, 9, 8, 3, 536891392, 0, 0, 7, 0, 1, 1, 0}

// cxdPark4e46f0 is the first boundary row of re/emu/cases/004e46f0.json (monster kill rewards).
var cxdPark4e46f0 = [228]int64{1, 5, 536879104, -1, 1, 2, 7, 0, 3, 0, 2, -1, 1, 2, 0, 0, 1, 1, 0, 1, 0, 1, 2, 0, 20, 20, 256, 3, 1, 10, 0, 50, 0, 0, 0, 5, 0, 0, 0, 0, 6, 0, 0, 0, 0, 7, 0, 0, 0, 0, 8, 0, 0, 0, 0, 9, 0, 0, 0, 0, 10, 0, 0, 0, 0, 11, 0, 0, 0, 0, 12, 0, 0, 0, 1, 1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 50, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 2, 0, 2, 0, 2, 6397337, 0, -1, 6397337, 0, -1, 6397337, 0, -1, 6397337, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0}
