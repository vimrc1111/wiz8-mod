package calc

import "fmt"

// monster_kill_rewards 0x4e46f0(monster, announce): the kill bookkeeping when a monster
// dies (build/wiz8.asm:306887..). Template = cache *0x6840c7[species]; killer = party slot
// at monster+0x308 when the death source (+0x304) is type 1. In order:
//   - kill message 0x58aad0(8|9, 0x614a3c, display name 0x4e5150, string 0x391) when
//     announce, not +0x253 and +0x28a;
//   - 0x5248d0(monster);
//   - unless +0x2da == 1 or +0x253: faction kill 0x535cf0(0, 1, template+0x25f, entry)
//     (killer, or source type 2 whose monster +0x16 == 2), then the death-fact dispatch
//     0x5090c0(template+0x187);
//   - when +0x15 and (+0x16 == 1, +0x8b == 0 or +0x16 == 2, +0x8b != 0): party kills
//     *0x6836a8+0xc, the map's kill word 0x686b77+33*map (map < 0x2f), the killer's
//     +0x9f9 kills and, with template+0x268, 0x52e690 when template+0x24f is at least the
//     average level of the present party plus 2; experience *0x6836a8+0x10 += template
//     +0x26b, else +0x181 (0x5182e0 when both are 0); species seen flag 0x688291[species].
//
// Modelled helpers: 4e5550/4e5620/4e5720 (entry lookup: monster list *0x683fad, then the
// group list *0x683fb9 as 0x2710+j), 53bea0/53bf10 (source type), 4e5150 (display name:
// 0x18d swprintf into 0x6875c3, record 5100b0 shown flag +0x2c, 4ca4c0 animation 0x15),
// 535cf0/535d50/535ea0/535b40 (faction standing table 0x68d6e8, 14 bytes per faction:
// standing, kills, stamp; classes <0x22, <0x43, else), 5090c0 (8 template ids),
// 5061a0 -> 5588f0 (fact byte 0x689b78+id, event log *0x68de40) -> 506670 (only
// 0x19b -> fact 0x273, 0x2a6 = 0 -> 5289b0(0x3b, 0) message queue 0x68c4c0; every other
// reached pair is the default exit, the rest panics), 444170/444030/443dc0 (string
// counter table 0x6598fc names, 0x659994 values, 0x6598e4 owner maps). Allocations
// follow the emulator's bump heap at 0x50000000 (notes/re/emu.md).
//
// Address stubs of the case: 5248d0 and 523330 (D4, returns ignored by the caller),
// 排除 58aad0 (message log), 52e690, 5182e0. Recorded arguments and counts are outputs.
// Fixture caps: species 0..999, killer and 0x6875ef party slots 0..7, map >= -1, factions
// 0..20, group list 0..1 entries, event log 8 entries, name/value/owner/queue tables 4
// entries, heap 64 dwords. Exempt in cov-exempt: asserts, template cache miss (template
// file I/O), the empty-party average (killer must be present). All of them panic here.

const (
	mkrA        = 0x20000000 // arena
	mkrStub     = mkrA + 0x0
	mkrMonster  = mkrA + 0x1000
	mkrOther    = mkrA + 0x1400
	mkrTpl      = mkrA + 0x2000
	mkrDecoyTpl = mkrA + 0x2400
	mkrMlist    = mkrA + 0x2800
	mkrMentries = mkrA + 0x2820
	mkrRlist    = mkrA + 0x2900
	mkrRentries = mkrA + 0x2920
	mkrRec0     = mkrA + 0x2a00
	mkrRec1     = mkrA + 0x2a40
	mkrGlist    = mkrA + 0x2b00
	mkrGentries = mkrA + 0x2b20
	mkrGrp      = mkrA + 0x2b40
	mkrRender   = mkrA + 0x2c00
	mkrRstate   = mkrA + 0x2e00
	mkrBattle   = mkrA + 0x3000
	mkrEvl      = mkrA + 0x3100
	mkrEvbuf    = mkrA + 0x3200
	mkrNames    = mkrA + 0x3300
	mkrVals     = mkrA + 0x3340
	mkrOwns     = mkrA + 0x3380
	mkrQueue    = mkrA + 0x3600
	mkrZs       = mkrA + 0x4000
	mkrStrtab   = mkrA + 0x5000
	mkrQ        = mkrA + 0x7000
	mkrChars    = mkrA + 0x8000
	mkrLvl      = mkrA + 0x20000
	mkrHeap     = 0x50000000 // emulator malloc base (notes/re/emu.md)
	mkrHeapMax  = 64 * 4     // heap window of the case outputs

	mkrCh       = 0x1862   // character record stride (*0x685174)
	mkrQs       = 0x106    // party presence stride (*0x685178)
	mkrCache    = 0x6840c7 // template cache, 1000 dwords
	mkrSeen     = 0x688291 // species seen flags, 1000 dwords
	mkrFacts    = 0x689b78 // story fact bytes
	mkrMap      = 0x686a70 // current map
	mkrRoster   = 0x686b77 // per-map kill words, 33-byte stride, maps below 0x2f
	mkrClock    = 0x686a48
	mkrFaction  = 0x68d6e8 // 21 factions x 14 bytes
	mkrAltName  = 0x6875c3 // 0x4e5150 swprintf buffer
	mkrAltFlag  = 0x6875f9
	mkrAltChar  = 0x6875ef
	mkrNameCnt  = 0x6598fc
	mkrValCnt   = 0x659994
	mkrOwnCnt   = 0x6598e4
	mkrQCnt     = 0x68c4c0
	mkrQStamp   = 0x68c4ac
	mkrKey      = 0x619d98 // 5090c0's counter key string (exe)
	mkrAltFmt   = 0x614b44 // 4e5150's swprintf format (exe, UTF-16)
	mkrJump     = 0x535e80 // 535d50's switch on faction-6 (exe)
	mkrProtect  = 0x535d9e // its case "standing >= 0x22 resets to class 0"
	mkrGeneral  = 0x535df2 // its default case
	mkrNEv      = 8
	mkrNTab     = 4
	mkrFactions = 21
)

// mkrFld is one case field: global or arena address, optionally indexed by another input.
type mkrFld struct {
	name   string
	addr   uint32
	size   int
	signed bool
	idx    string
	stride uint32
	count  int64
}

type mkrStubDef struct {
	addr, base uint32
	nargs, n   int
}

var mkrStubs = []mkrStubDef{
	{0x5248d0, 0x00, 1, 1}, {0x523330, 0x10, 3, 1}, {0x58aad0, 0x20, 3, 4},
	{0x52e690, 0x60, 5, 2}, {0x5182e0, 0xa0, 3, 2},
}

var mkrFacts9 = []uint32{0x36, 0x318, 0x2a6, 0x1b6, 0x1e8, 0x326, 0x19b, 0x273, 0x1be}

var mkrIns = func() []mkrFld {
	var f []mkrFld
	add := func(name string, addr uint32, size int, signed bool) {
		f = append(f, mkrFld{name: name, addr: addr, size: size, signed: signed})
	}
	ix := func(name string, addr uint32, size int, signed bool, idx string, stride uint32, count int64) {
		f = append(f, mkrFld{name, addr, size, signed, idx, stride, count})
	}
	add("arg2", 0, 4, false) // stack argument 2
	add("species", mkrMonster+8, 4, false)
	ix("tplPtr", mkrCache, 4, false, "species", 4, 1000)
	ix("seen", mkrSeen, 4, true, "species", 4, 1000)
	add("monId", mkrMonster, 4, true)
	add("otherId", mkrOther, 4, true)
	add("otherSpecies", mkrOther+8, 4, false)
	add("grpCount", mkrGlist+8, 4, false)
	add("grpId", mkrGrp, 4, true)
	add("grp16", mkrGrp+0x16, 1, false)
	add("other16", mkrOther+0x16, 1, false)
	add("recSel", mkrStub+0x300, 4, true)
	add("recId", mkrRec1, 4, true) // also monster+4 when recSel == 0
	add("recDecoyId", mkrRec0, 4, true)
	add("rec0Shown", mkrRec0+0x2c, 1, false)
	add("rec1Shown", mkrRec1+0x2c, 1, false)
	add("m15", mkrMonster+0x15, 1, false)
	add("m16", mkrMonster+0x16, 1, false)
	add("m253", mkrMonster+0x253, 1, false)
	add("m28a", mkrMonster+0x28a, 1, false)
	add("m2da", mkrMonster+0x2da, 4, false)
	add("srcType", mkrMonster+0x304, 4, false)
	add("srcMon", mkrMonster+0x30c, 4, true)
	add("m8b", mkrMonster+0x8b, 4, false)
	add("anim", mkrRstate+0xa4, 1, false)
	add("anim2", mkrRstate+0xa7, 1, false)
	add("t187", mkrTpl+0x187, 2, false)
	add("t25f", mkrTpl+0x25f, 1, false)
	add("t268", mkrTpl+0x268, 1, false)
	add("t24f", mkrTpl+0x24f, 1, false)
	add("t26b", mkrTpl+0x26b, 4, false)
	add("t181", mkrTpl+0x181, 4, false)
	add("td0", mkrTpl+0xd0, 1, false)
	for w := uint32(0); w < 8; w++ {
		add(fmt.Sprintf("present%d", w), mkrQ+mkrQs*w, 1, false)
		add(fmt.Sprintf("char4_%d", w), mkrChars+mkrCh*w+4, 1, false)
		add(fmt.Sprintf("level%d", w), mkrChars+mkrCh*w+0x89, 4, true)
		add(fmt.Sprintf("kills%d", w), mkrChars+mkrCh*w+0x9f9, 4, false)
		add(fmt.Sprintf("a29_%d", w), mkrChars+mkrCh*w+0xa29, 4, false)
	}
	add("killer", mkrMonster+0x308, 4, true)
	ix("killerPresent", mkrQ, 1, false, "killer", mkrQs, 8)
	ix("killerChar4", mkrChars+4, 1, false, "killer", mkrCh, 8)
	add("map", mkrMap, 4, true)
	for m := -1; m < 47; m++ {
		add(fmt.Sprintf("roster%d", m), uint32(mkrRoster+33*m), 2, false)
	}
	add("partyKills", mkrBattle+0xc, 4, false)
	add("partyExp", mkrBattle+0x10, 4, false)
	add("clock", mkrClock, 4, false)
	for fc := uint32(0); fc < mkrFactions; fc++ {
		b := mkrFaction + 14*fc
		add(fmt.Sprintf("stand%d", fc), b, 1, true)
		add(fmt.Sprintf("fkills%d", fc), b+1, 4, true)
		add(fmt.Sprintf("fstamp%d", fc), b+6, 4, false)
	}
	for _, id := range mkrFacts9 {
		add(fmt.Sprintf("fact%x", id), mkrFacts+id, 1, false)
	}
	add("evCnt", mkrEvl+4, 4, false)
	add("evCap", mkrEvl+8, 4, false)
	add("nameCnt", mkrNameCnt, 4, false)
	add("nameCap", mkrNameCnt+4, 4, false)
	add("valCnt", mkrValCnt, 4, false)
	add("valCap", mkrValCnt+4, 4, false)
	add("ownCnt", mkrOwnCnt, 4, false)
	add("ownCap", mkrOwnCnt+4, 4, false)
	for i := uint32(0); i < mkrNTab; i++ {
		add(fmt.Sprintf("name%d", i), mkrNames+4*i, 4, false)
		add(fmt.Sprintf("val%d", i), mkrVals+4*i, 4, true)
		add(fmt.Sprintf("own%d", i), mkrOwns+4*i, 4, true)
	}
	add("qCount", mkrQCnt, 4, true)
	add("qCap", mkrQCnt+4, 4, true)
	add("qStamp", mkrQStamp, 4, false)
	add("altFlag", mkrAltFlag, 1, false)
	add("altChar", mkrAltChar, 4, false)
	for i := uint32(0); i < 3; i++ {
		add(fmt.Sprintf("altName%d", i), mkrAltName+4*i, 4, false)
	}
	return f
}()

var mkrOuts = func() []mkrFld {
	var f []mkrFld
	add := func(name string, addr uint32, size int, signed bool) {
		f = append(f, mkrFld{name: name, addr: addr, size: size, signed: signed})
	}
	for _, s := range mkrStubs {
		add(fmt.Sprintf("n%x", s.addr), mkrStub+s.base, 4, false)
		for j := 0; j < s.nargs; j++ {
			for k := 0; k < s.n; k++ {
				add(fmt.Sprintf("a%x_%d_%d", s.addr, j+1, k), mkrStub+s.base+uint32(4+4*s.n*j+4*k), 4, false)
			}
		}
	}
	add("partyKills", mkrBattle+0xc, 4, false)
	add("partyExp", mkrBattle+0x10, 4, false)
	f = append(f, mkrFld{"seen", mkrSeen, 4, true, "species", 4, 1000})
	for m := -1; m < 47; m++ {
		add(fmt.Sprintf("roster%d", m), uint32(mkrRoster+33*m), 2, false)
	}
	for w := uint32(0); w < 8; w++ {
		add(fmt.Sprintf("kills%d", w), mkrChars+mkrCh*w+0x9f9, 4, false)
	}
	for fc := uint32(0); fc < mkrFactions; fc++ {
		b := mkrFaction + 14*fc
		add(fmt.Sprintf("stand%d", fc), b, 1, true)
		add(fmt.Sprintf("fkills%d", fc), b+1, 4, true)
		add(fmt.Sprintf("fstamp%d", fc), b+6, 4, false)
	}
	for _, id := range mkrFacts9 {
		add(fmt.Sprintf("fact%x", id), mkrFacts+id, 1, false)
	}
	add("evCnt", mkrEvl+4, 4, false)
	add("evCap", mkrEvl+8, 4, false)
	for k := uint32(0); k < mkrNEv; k++ {
		for j, nm := range []string{"Stamp", "Id", "Val"} {
			add(fmt.Sprintf("ev%d%s", k, nm), mkrEvbuf+12*k+4*uint32(j), 4, false)
		}
	}
	for _, t := range []struct {
		n string
		a uint32
	}{{"nameCnt", mkrNameCnt}, {"nameCap", mkrNameCnt + 4}, {"valCnt", mkrValCnt}, {"valCap", mkrValCnt + 4}, {"ownCnt", mkrOwnCnt}, {"ownCap", mkrOwnCnt + 4}} {
		add(t.n, t.a, 4, false)
	}
	for i := uint32(0); i < mkrNTab; i++ {
		add(fmt.Sprintf("val%d", i), mkrVals+4*i, 4, true)
		add(fmt.Sprintf("own%d", i), mkrOwns+4*i, 4, true)
		add(fmt.Sprintf("name%d", i), mkrNames+4*i, 4, false)
	}
	add("qCount", mkrQCnt, 4, true)
	add("qCap", mkrQCnt+4, 4, true)
	for i := uint32(0); i < 4; i++ {
		add(fmt.Sprintf("queue%d", i), mkrQueue+4*i, 4, false)
	}
	for i := uint32(0); i < mkrHeapMax/4; i++ {
		add(fmt.Sprintf("heap%d", i), mkrHeap+4*i, 4, false)
	}
	add("evBuf", mkrEvl+12, 4, false)
	add("nameBuf", mkrNameCnt+8, 4, false)
	add("valBuf", mkrValCnt+8, 4, false)
	add("ownBuf", mkrOwnCnt+8, 4, false)
	add("qBuf", mkrQCnt+8, 4, false)
	for i := uint32(0); i < 3; i++ {
		add(fmt.Sprintf("altName%d", i), mkrAltName+4*i, 4, false)
	}
	return f
}()

func mkrFieldNames(fs []mkrFld) []string {
	n := make([]string, len(fs))
	for i, f := range fs {
		n[i] = f.name
	}
	return n
}

var mkrInAt = func() map[string]int {
	m := map[string]int{}
	for i, f := range mkrIns {
		m[f.name] = i
	}
	return m
}()

type mkrC struct {
	m    n4Mem
	in   []int64
	heap uint32
	// res is the dword left in 5588f0's value slot by an earlier set at the same depth:
	// 5588f0 stores the fact byte into its low byte only. 506670(0x2a6, 0) leaves the
	// return address 0x50629e of 506280's inner call there. Nested sets use a deeper slot.
	res   uint32
	depth int
}

// addr resolves a field; ok=false means the emulator skips it (negative index).
func (c *mkrC) addr(f mkrFld) (uint32, bool) {
	if f.idx == "" {
		return f.addr, true
	}
	i := c.in[mkrInAt[f.idx]]
	if i < 0 {
		return 0, false
	}
	if i >= f.count {
		panic("monster kill rewards: " + f.name + " index past the fixture")
	}
	return f.addr + uint32(i)*f.stride, true
}

func monsterKillRewardsNode(in []int64, _ Rand) []int64 {
	if len(in) != len(mkrIns) {
		panic("monster kill rewards: wrong input count")
	}
	c := &mkrC{m: n4Mem{}, in: in}
	m := c.m
	c.setup()
	for i, f := range mkrIns {
		if f.name == "arg2" {
			continue
		}
		if a, ok := c.addr(f); ok {
			m.put(a, f.size, uint32(in[i]))
		}
	}
	if sel := in[mkrInAt["recSel"]]; sel == 0 { // recId's "also" target: monster+4
		m.w32(mkrMonster+4, uint32(in[mkrInAt["recId"]]))
	} else if sel > 0 {
		panic("monster kill rewards: recSel past the fixture")
	}
	c.node(uint8(in[0]))
	out := make([]int64, len(mkrOuts))
	for i, f := range mkrOuts {
		a, ok := c.addr(f)
		if !ok {
			panic("monster kill rewards: output index negative")
		}
		v := m.get(a, f.size)
		out[i] = int64(v)
		if f.signed {
			switch f.size {
			case 1:
				out[i] = int64(int8(v))
			case 2:
				out[i] = int64(int16(v))
			default:
				out[i] = int64(int32(v))
			}
		}
	}
	return out
}

// setup mirrors the case's setup: object links, decoy templates in every cache slot,
// seen flags set, UTF-16 party names, fixed UI/debug flags 0.
func (c *mkrC) setup() {
	m := c.m
	for s := uint32(0); s < 1000; s++ {
		m.w32(mkrCache+4*s, mkrDecoyTpl)
		m.w32(mkrSeen+4*s, 1)
	}
	m.w16(mkrDecoyTpl+0x187, 0x131)
	m.w8(mkrDecoyTpl+0x25f, 5)
	m.w8(mkrDecoyTpl+0x268, 1)
	m.w8(mkrDecoyTpl+0x24f, 200)
	m.w32(mkrDecoyTpl+0x26b, 12345)
	m.w32(mkrDecoyTpl+0x181, 777)
	m.w8(mkrDecoyTpl+0xd0, 1)
	m.w32(0x683fad, mkrMlist)
	m.w32(mkrMlist, mkrMentries)
	m.w32(mkrMlist+8, 2)
	m.w32(mkrMentries, mkrOther)
	m.w32(mkrMentries+4, mkrMonster)
	m.w32(0x683fb9, mkrGlist)
	m.w32(mkrGlist, mkrGentries)
	m.w32(mkrGentries, mkrGrp)
	m.w32(0x683fb1, mkrRlist)
	m.w32(mkrRlist, mkrRentries)
	m.w32(mkrRlist+8, 2)
	m.w32(mkrRentries, mkrRec0)
	m.w32(mkrRentries+4, mkrRec1)
	m.w32(0x683fbd, 0)
	m.w32(mkrMonster+0xc, mkrRender)
	m.w32(mkrRender+0x1d8, mkrRstate)
	m.w32(0x685174, mkrChars)
	m.w32(0x685178, mkrQ)
	m.w32(0x6836a8, mkrBattle)
	m.w32(0x68c09c, mkrStrtab)
	for _, k := range append([]uint32{0x245, 0x246, 0x247, 0x391}, mkrSeq(0x5a1, 0x5b4)...) {
		m.w32(mkrStrtab+4*k, mkrZs+2*k)
	}
	for w := uint32(0); w < 8; w++ {
		for i, ch := range fmt.Sprintf("C%d\x00", w) {
			m.w16(mkrChars+mkrCh*w+5+2*uint32(i), uint32(ch))
		}
	}
	m.w32(0x68de40, mkrEvl)
	m.w32(mkrEvl+12, mkrEvbuf)
	m.w32(0x6836ac, mkrLvl)
	m.w8(0x688290, 0)
	m.w32(0x683f8c, 0)
	m.w32(0x68edcc, 0)
	m.w8(0x68de44, 0)
	m.w32(0x6850d5, 0)
	m.w32(mkrNameCnt+8, mkrNames)
	m.w32(mkrValCnt+8, mkrVals)
	m.w32(mkrOwnCnt+8, mkrOwns)
	m.w32(mkrQCnt+8, mkrQueue)
}

func mkrSeq(lo, hi uint32) []uint32 {
	var s []uint32
	for k := lo; k < hi; k++ {
		s = append(s, k)
	}
	return s
}

// call records one address-stub call like the case's "record"/"count".
func (c *mkrC) call(addr uint32, args ...uint32) {
	for _, s := range mkrStubs {
		if s.addr != addr {
			continue
		}
		if len(args) != s.nargs {
			panic("monster kill rewards: stub argument count")
		}
		at := uint32(mkrStub + s.base)
		k := c.m.r32(at)
		if int(k) >= s.n {
			panic(fmt.Sprintf("monster kill rewards: stub %#x called past its record cap", addr))
		}
		for j, a := range args {
			c.m.w32(at+uint32(4+4*s.n*j)+4*k, a)
		}
		c.m.w32(at, k+1)
		return
	}
	panic("monster kill rewards: unknown stub")
}

func (c *mkrC) alloc(n uint32) uint32 {
	a := mkrHeap + c.heap
	step := (n + 7) &^ 7
	if step < 8 {
		step = 8
	}
	c.heap += step
	if c.heap > mkrHeapMax {
		panic("monster kill rewards: heap above the output window")
	}
	return a
}

// slot checks a table write lands in its fixture buffer (cap) or on the heap.
func (c *mkrC) slot(buf uint32, i int32, size uint32, fixture uint32, cap int32) uint32 {
	switch {
	case buf == fixture && i >= 0 && i < cap:
	case buf >= mkrHeap && buf < mkrHeap+mkrHeapMax && i >= 0:
	default:
		panic("monster kill rewards: table write outside its fixture")
	}
	return buf + uint32(i)*size
}

// grow appends v to a (count, capacity, buffer) dword table, reallocating to count+1 when
// full (5289b0 / 443dc0 / 5588f0 shape).
func (c *mkrC) grow(cntA uint32, v uint32, fixture uint32) {
	m := c.m
	cnt := int32(m.r32(cntA))
	need := cnt + 1
	if need > int32(m.r32(cntA+4)) {
		old := m.r32(cntA + 8)
		nb := c.alloc(uint32(need) * 4)
		m.w32(cntA+8, nb)
		m.w32(cntA+4, uint32(need))
		for i := int32(0); i < cnt; i++ {
			m.w32(nb+4*uint32(i), m.r32(old+4*uint32(i)))
		}
	}
	m.w32(c.slot(m.r32(cntA+8), cnt, 4, fixture, mkrNTab), v)
	m.w32(cntA, uint32(cnt+1))
}

func (c *mkrC) node(announce uint8) {
	m := c.m
	mon := uint32(mkrMonster)
	species := m.r32(mon + 8)
	if species >= 1000 {
		panic("monster kill rewards: species assert")
	}
	tpl := m.r32(mkrCache + 4*species)
	if tpl == 0 {
		panic("monster kill rewards: template cache miss (template file I/O, exempt)")
	}
	idx := c.find(m.r32(mon), true)
	chars, q := m.r32(0x685174), m.r32(0x685178)
	killer := int32(-1)
	msg := uint32(9)
	if c.src(mon+0x304, 1) {
		killer = int32(m.r32(mon + 0x308))
		if killer < 0 || killer >= 8 || m.r8(q+mkrQs*uint32(killer)) == 0 || m.r8(chars+mkrCh*uint32(killer)+4) == 0 {
			panic("monster kill rewards: killer slot assert")
		}
		msg = 8
	}
	if announce != 0 && m.r8(mon+0x253) == 0 && m.r8(mon+0x28a) != 0 {
		name := c.name(mon, tpl, chars)
		c.call(0x58aad0, msg, 0x614a3c, name)
	}
	c.call(0x5248d0, mon)
	if m.r32(mon+0x2da) == 1 || m.r8(mon+0x253) != 0 {
		return
	}
	if killer != -1 {
		c.faction(0, 1, uint8(m.r8(tpl+0x25f)), idx)
	} else if c.src(mon+0x304, 2) {
		if k := c.find(m.r32(mon+0x30c), false); k != -1 {
			if m.r8(c.entry(k)+0x16) == 2 {
				c.faction(0, 1, uint8(m.r8(tpl+0x25f)), idx)
			}
		}
	}
	c.deathFacts(m.r16(tpl+0x187), chars)
	if m.r8(mon+0x15) == 0 {
		return
	}
	m16, m8b := m.r8(mon+0x16), m.r32(mon+0x8b)
	if !(m16 == 1 && m8b == 0) && !(m16 == 2 && m8b != 0) {
		return
	}
	battle := m.r32(0x6836a8)
	m.w32(battle+0xc, m.r32(battle+0xc)+1)
	if mp := int32(m.r32(mkrMap)); mp < 0x2f {
		if mp < -1 {
			panic("monster kill rewards: map below the roster fixture")
		}
		a := uint32(mkrRoster + 33*mp)
		m.w16(a, m.r16(a)+1)
	}
	if killer != -1 {
		ch := chars + mkrCh*uint32(killer)
		m.w32(ch+0x9f9, m.r32(ch+0x9f9)+1)
		if m.r8(tpl+0x268) != 0 {
			var sum, cnt int32
			for w := uint32(0); w < 8; w++ {
				if m.r8(q+mkrQs*w) != 0 {
					sum += int32(m.r32(chars + mkrCh*w + 0x89))
					cnt++
				}
			}
			if cnt == 0 {
				panic("monster kill rewards: no present member (killer must be present, exempt)")
			}
			if int32(m.r8(tpl+0x24f)) >= sum/cnt+2 { // fild/fidiv then _ftol: truncation
				c.call(0x52e690, ch, m.r32(0x5ee61c), 0, m.r32(0x5ed8c8), m.r32(0x5ed914))
			}
		}
	}
	exp := m.r32(tpl + 0x26b)
	if exp == 0 {
		exp = m.r32(tpl + 0x181)
		if exp == 0 {
			c.call(0x5182e0, 0, 0x614a00, tpl)
		}
	}
	m.w32(battle+0x10, m.r32(battle+0x10)+exp)
	if s := mkrSeen + 4*species; m.r32(s) == 0 {
		m.w32(s, 1)
	}
}

// src is 53bea0 (want 1, killer at +4) / 53bf10 (want 2, monster at +8) with flag 0.
func (c *mkrC) src(p uint32, want uint32) bool {
	if c.m.r32(p) != want {
		return false
	}
	if int32(c.m.r32(p+4*want)) == -1 {
		panic("monster kill rewards: death source assert")
	}
	return true
}

func (c *mkrC) listN(list uint32) int32 {
	if list == 0 {
		return 0
	}
	return int32(c.m.r32(list + 8))
}

func (c *mkrC) listAt(list uint32, i int32) uint32 {
	if list == 0 || i >= int32(c.m.r32(list+8)) {
		return 0
	}
	return c.m.r32(c.m.r32(list) + 4*uint32(i))
}

// find is 4e5550: monster list index, else 0x2710 + group index; miss asserts with flag.
func (c *mkrC) find(id uint32, flag bool) int32 {
	ml, gl := c.m.r32(0x683fad), c.m.r32(0x683fb9)
	for i := int32(0); i < c.listN(ml); i++ {
		if c.m.r32(c.entry(i)) == id {
			return i
		}
	}
	if c.listN(gl) > 1 {
		panic("monster kill rewards: group list past the fixture")
	}
	for j := int32(0); j < c.listN(gl); j++ {
		if c.m.r32(c.listAt(gl, j)) == id {
			return 0x2710 + j
		}
	}
	if flag {
		panic("monster kill rewards: entry lookup assert")
	}
	return -1
}

// entry is 4e5620.
func (c *mkrC) entry(k int32) uint32 {
	list := c.m.r32(0x683fad)
	if uint32(k) >= 0x2710 && uint32(k) < 0x4e20 {
		list, k = c.m.r32(0x683fb9), k-0x2710
	}
	if uint32(k) >= uint32(c.listN(list)) {
		panic("monster kill rewards: entry index assert")
	}
	e := c.listAt(list, k)
	if e == 0 {
		panic("monster kill rewards: null entry assert")
	}
	return e
}

// name is 4e5150(monster, 0, 0).
func (c *mkrC) name(mon, tpl, chars uint32) uint32 {
	m := c.m
	if m.r16(tpl+0x187) == 0x18d {
		ac := m.r32(mkrAltChar)
		if ac >= 8 {
			panic("monster kill rewards: 0x6875ef party slot past the fixture")
		}
		c.swprintf(mkrAltName, mkrAltFmt, chars+mkrCh*ac+5)
		return mkrAltName
	}
	if id := m.r32(mon + 4); id != 0 {
		rec := uint32(0)
		rl := m.r32(0x683fb1)
		for i := int32(0); i < c.listN(rl) && rec == 0; i++ {
			if e := c.listAt(rl, i); m.r32(e) == id {
				rec = e
			}
		}
		if rec == 0 {
			panic("monster kill rewards: record lookup assert")
		}
		if m.r8(rec+0x2c) != 0 {
			return tpl
		}
		return tpl + 0x60
	}
	rs := m.r32(m.r32(mon+0xc) + 0x1d8)
	if m.r8(rs+0xa4) != 0x15 && m.r8(rs+0xa7) != 0x15 { // 4ca4c0
		c.call(0x5182e0, 1, 0x614b20, m.r32(mon))
	}
	return tpl + 0x60
}

// swprintf writes fmt with its single %s argument as UTF-16 at dst.
func (c *mkrC) swprintf(dst, fmtA, arg uint32) {
	m := c.m
	var out []uint32
	for a := fmtA; ; a += 2 {
		ch := m.r16(a)
		if ch == 0 {
			break
		}
		if ch == '%' {
			if m.r16(a+2) != 's' {
				panic("monster kill rewards: unexpected swprintf directive")
			}
			for s := arg; m.r16(s) != 0; s += 2 {
				out = append(out, m.r16(s))
			}
			a += 2
			continue
		}
		out = append(out, ch)
	}
	for i, ch := range append(out, 0) {
		m.w16(dst+2*uint32(i), ch)
	}
}

// faction is 535cf0(a, b, f, d).
func (c *mkrC) faction(a, b, f uint8, d int32) {
	if b != 1 || f == 0 || f == 1 {
		return
	}
	switch {
	case a == 0:
		c.factionKill(f, d)
	case int8(a) > 0 && a <= 3:
		c.standing(f, int8(d))
	}
}

func mkrClass(s int8) int {
	switch {
	case s < 0x22:
		return 0
	case s >= 0x43:
		return 2
	}
	return 1
}

func (c *mkrC) factionBase(f uint8) uint32 {
	if int8(f) < 0 || f >= mkrFactions {
		panic("monster kill rewards: faction assert")
	}
	return mkrFaction + 14*uint32(f)
}

// factionKill is 535d50(f, entry).
func (c *mkrC) factionKill(f uint8, d int32) {
	m := c.m
	if int8(f) < 0 || f >= mkrFactions {
		panic("monster kill rewards: faction assert")
	}
	base := mkrFaction + 14*uint32(f)
	if k := int32(m.r32(base + 1)); k < 0xfa {
		m.w32(base+1, uint32(k+1))
	}
	e := c.entry(d)
	sp := m.r32(e + 8)
	if sp >= 1000 {
		panic("monster kill rewards: species assert")
	}
	t := m.r32(mkrCache + 4*sp)
	if t == 0 {
		panic("monster kill rewards: template cache miss")
	}
	if j := uint32(f) - 6; j <= 6 {
		switch m.r32(mkrJump + 4*j) {
		case mkrProtect:
			if int8(m.r8(base)) >= 0x22 {
				c.reset(f, 0)
				return
			}
		case mkrGeneral:
		default:
			panic("monster kill rewards: unexpected faction switch target")
		}
	}
	var delta int8
	switch k := int32(m.r32(base + 1)); {
	case m.r16(t+0x187) == 0x146:
		delta = -40
	case m.r8(t+0xd0)&1 != 0:
		delta = -20
	case k < 0x15:
		delta = -2
	case k >= 0x29:
		delta = -10
	default:
		delta = -6
	}
	c.standing(f, delta)
}

// standing is 535ea0(f, delta): clamp 0..0x63, stamp on class change, log the direction.
func (c *mkrC) standing(f uint8, delta int8) {
	m := c.m
	base := c.factionBase(f)
	old := int8(m.r8(base))
	sum := int32(old) + int32(delta)
	nw := old + delta
	if sum > 0x63 {
		nw = 0x63
	} else if sum < 0 {
		nw = 0
	}
	m.w8(base, uint32(uint8(nw)))
	if mkrClass(nw) != mkrClass(old) {
		m.w32(base+6, m.r32(mkrClock))
	}
	switch {
	case nw < old:
		c.factionLog(0, f)
	case nw > old:
		c.factionLog(5, f)
	}
}

// reset is 535b40(f, cls): standing 0x19 / 0x37 / 0x4b.
func (c *mkrC) reset(f uint8, cls int) {
	m := c.m
	base := c.factionBase(f)
	oc := mkrClass(int8(m.r8(base)))
	m.w8(base, []uint32{0x19, 0x37, 0x4b}[cls])
	if cls == oc {
		return
	}
	m.w32(base+6, m.r32(mkrClock))
	if cls < oc {
		c.factionLog(0, f)
	} else {
		c.factionLog(5, f)
	}
}

// factionLog: 58aad0(code, string 0x247, faction name string from the word table 0x61eacc).
func (c *mkrC) factionLog(code uint32, f uint8) {
	m := c.m
	st := m.r32(0x68c09c)
	c.call(0x58aad0, code, m.r32(st+4*0x247), m.r32(st+4*m.r16(0x61eacc+2*uint32(f))))
}

// deathFacts is 5090c0(template id).
func (c *mkrC) deathFacts(tid, chars uint32) {
	m := c.m
	switch tid {
	case 0x131:
		c.set(0x36, 1)
	case 0x1a9:
		c.set(0x318, 1)
		c.faction(2, 1, 0x10, 0x14)
	case 0x14f, 0xdd:
		if i := c.counterFind(mkrKey); i == -1 {
			c.counterAdd(mkrKey, 1)
		} else {
			if i < int32(m.r32(mkrValCnt)) { // 444030(key, 2)
				m.w32(c.slot(m.r32(mkrValCnt+8), i, 4, mkrVals, mkrNTab), 2)
			}
			c.set(0x19b, 1)
			c.reset(7, 0)
		}
	case 0x22b:
		hit := m.r8(mkrFacts + 0x1be)
		if hit != 0 {
			c.set(0x2a6, 0)
		}
		if m.r8(mkrAltFlag) != 0 {
			ac := m.r32(mkrAltChar)
			if ac >= 8 {
				panic("monster kill rewards: 0x6875ef party slot past the fixture")
			}
			ch := chars + mkrCh*ac
			if m.r32(ch+0xa29) != 0 {
				c.call(0x523330, ac, 10, 0)
			}
			c.call(0x52e690, ch, m.r32(0x5ee6f8), 0, m.r32(0x5ed8c8), m.r32(0x5ed914))
		}
		c.set(0x1b6, 1)
	case 0x181:
		c.set(0x1e8, 1)
	case 0x175, 0x222:
		c.set(0x326, 1)
	}
}

func (c *mkrC) cstr(a uint32) []byte {
	var s []byte
	for ; c.m.r8(a) != 0; a++ {
		s = append(s, byte(c.m.r8(a)))
	}
	return s
}

// mkrStricmp is the emulator's _stricmp: equal after folding A-Z.
func mkrStricmp(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if x >= 'A' && x <= 'Z' {
			x += 32
		}
		if y >= 'A' && y <= 'Z' {
			y += 32
		}
		if x != y {
			return false
		}
	}
	return true
}

// counterFind is 444170: name match and owner map == current map.
func (c *mkrC) counterFind(key uint32) int32 {
	m := c.m
	k := c.cstr(key)
	n := int32(m.r32(mkrNameCnt))
	for i := int32(0); i < n; i++ {
		if !mkrStricmp(c.cstr(m.r32(m.r32(mkrNameCnt+8)+4*uint32(i))), k) {
			continue
		}
		own := m.r32(mkrOwnCnt + 8)
		if i < int32(m.r32(mkrOwnCnt)) {
			own += 4 * uint32(i)
		}
		if m.r32(own) == m.r32(mkrMap) {
			return i
		}
	}
	return -1
}

// counterAdd is 443dc0(key, v) on a miss: strdup, then append to the three tables.
func (c *mkrC) counterAdd(key, v uint32) {
	m := c.m
	mp := m.r32(mkrMap)
	s := append(c.cstr(key), 0)
	dup := c.alloc(uint32(len(s)))
	for i, b := range s {
		m.w8(dup+uint32(i), uint32(b))
	}
	c.grow(mkrNameCnt, dup, mkrNames)
	c.grow(mkrValCnt, v, mkrVals)
	c.grow(mkrOwnCnt, mp, mkrOwns)
}

// set is 5061a0(id, val, 0).
func (c *mkrC) set(id, val uint32) {
	m := c.m
	c.depth++
	defer func() { c.depth-- }()
	old := m.r8(mkrFacts + id)
	m.w8(mkrFacts+id, val)
	if old != val {
		c.event(id)
	}
	switch {
	case id == 0x19b && val != 0:
		c.set(0x273, 1)
	case id == 0x2a6 && val == 0:
		c.message(0x3b, 0)
		if c.depth == 1 {
			c.res = 0x50629e
		}
	case val == 1 && (id == 0x36 || id == 0x318 || id == 0x1b6 || id == 0x1e8 || id == 0x326 || id == 0x273):
	default:
		panic(fmt.Sprintf("monster kill rewards: 0x506670(%#x, %d) beyond the modelled exits", id, val))
	}
}

// event is 5588f0(id): append (map, id, fact) to the event log.
func (c *mkrC) event(id uint32) {
	m := c.m
	evl := m.r32(0x68de40)
	cnt := int32(m.r32(evl + 4))
	need := cnt + 1
	if need > int32(m.r32(evl+8)) {
		old := m.r32(evl + 12)
		nb := c.alloc(uint32(need) * 12)
		m.w32(evl+12, nb)
		m.w32(evl+8, uint32(need))
		for i := uint32(0); int32(i) < cnt; i++ {
			for w := uint32(0); w < 3; w++ {
				m.w32(nb+12*i+4*w, m.r32(old+12*i+4*w))
			}
		}
	}
	e := c.slot(m.r32(evl+12), cnt, 12, mkrEvbuf, mkrNEv)
	m.w32(e, m.r32(mkrMap))
	m.w32(e+4, id)
	res := uint32(0)
	if c.depth == 1 {
		res = c.res &^ 0xff
	}
	m.w32(e+8, res|m.r8(mkrFacts+id))
	m.w32(evl+4, uint32(cnt+1))
}

// message is 5289b0(kind, x): a 0x24-byte message appended to the queue at 0x68c4c0.
func (c *mkrC) message(kind, x uint32) {
	m := c.m
	o := c.alloc(0x24)
	m.w32(o, 0xffffffff)
	m.w32(o+0xc, kind)
	m.w32(o+0x10, x)
	m.w32(o+0x20, m.r32(mkrQStamp))
	c.grow(mkrQCnt, o, mkrQueue)
}

func init() {
	Nodes["004e46f0"] = Node{Addr: "004e46f0", In: mkrFieldNames(mkrIns), Out: mkrFieldNames(mkrOuts), F: monsterKillRewardsNode}
}
