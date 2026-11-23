package calc

import "fmt"

// script_attribute_skill_gain 0x506670(ev, flag): the NPC-script "fact changed" dispatcher.
// 0x5061a0 sets fact byte 0x689b78+ev and then calls this with the new value; each handler
// (jump tables 0x507c7c, 0x507c90, 0x507cbc/0x507d24, 0x507dc0/0x507e24, 0x507ef4/0x507f14,
// 0x507f24/0x507f78, 0x507fc0, 0x507fe8/0x50801c) queues messages, resets faction
// standings, moves NPCs, closes script triggers, grants stock, or raises attributes and
// skills. Every other event (and every negative one) leaves at 0x507c74.
//
// The model runs on the case's memory (n4Mem): inputs are written at their case addresses,
// the callees below are transliterated onto it, and outputs are read back. Address-stubbed
// callees (re/emu/cases/00506670.json "stubs") are replayed by stub(): arguments and call
// counts are outputs, returns come from inputs. 0x55a7b0 (D4) is the one stubbed formula
// whose return is read: Emu replays the stub, F runs shopStockAddNode and writes its stock
// list back (F-only inputs: item class/kind at the F-only area).
//
// Fixture caps (calc panics past them): table buffers hold 4 entries before they move to
// the heap, heap use <= 0x400 bytes, current character 0..7, NPC party index 0..7, class
// 0..14, spell school 0..5, stock lists are the NPC's own fixture list with <= 6 entries.
const (
	sagArena  = 0x20000000
	sagHeap   = 0x50000000
	sagHeapW  = 0x400
	sagTxt    = sagArena + 0x24000 // string table *0x68c09c (fixture)
	sagZS     = sagArena + 0x25800 // empty wide strings the table points at
	sagTxtN   = 0x5c0
	sagEvBuf  = sagArena + 0x26420
	sagQBuf   = sagArena + 0x26500
	sagNB     = sagArena + 0x26600
	sagVB     = sagArena + 0x26640
	sagOB     = sagArena + 0x26680
	sagTabN   = 4
	sagNpc0   = sagArena + 0x1E100
	sagNpcS   = 0x140
	sagNpcN   = 7
	sagStk0   = sagArena + 0x1F000
	sagStkS   = 0x100
	sagStkN   = 6
	sagFOnly  = sagArena + 0x29000
	sagFacts  = 0x689b78 // fact bytes, exe 0x5061b6
	sagMap    = 0x686a70 // current map, exe 0x50c44a
	sagClock  = 0x686a48 // game clock, exe 0x55aaa6
	sagMode   = 0x68ec78 // screen mode, exe 0x553b08
	sagQCnt   = 0x68c4c0 // message queue count/cap/buf, exe 0x528872
	sagQStamp = 0x68c4ac // message stamp, exe 0x528864
	sagNames  = 0x6598fc // counter names count/cap/buf, exe 0x444170
	sagVals   = 0x659994 // counter values, exe 0x444030
	sagOwns   = 0x6598e4 // counter owner maps, exe 0x444170
	sagChS    = 0x1862   // character stride, exe 0x517ef9
	sagPresS  = 0x106    // party presence stride, exe 0x50b8f0
	sagSkS    = 0x26     // skill block stride, exe 0x553b40
)

// sagItems are the stock items the handlers grant (exe 0x5066a0..0x5074cc pushes); the
// F-only area holds their class/kind byte pairs in this order.
var sagItems = []uint32{0x28a, 0x28b, 0x28c, 0x28d, 0x28e, 0x242, 0x1b0, 0x1f4, 0x1f5, 0x1f8}

var sagIns, sagInNames = n4Expand(sagInG)
var sagOuts, sagOutNames = n4Expand(sagOutG)
var sagInAt = func() map[string]int {
	m := map[string]int{}
	for i, n := range sagInNames {
		m[n] = i
	}
	return m
}()

type sagC struct {
	m     n4Mem
	r     Rand
	real  bool
	ins   []int64
	calls map[string]int
	heap  uint32
	evHi  uint32 // upper bytes of the event value's stack slot (see event)
}

func sagFail(what string) { panic("script attribute skill gain: " + what) }

func scriptAttributeSkillGainNode(in []int64, r Rand) []int64 { return sagRun(in, r, true) }

func init() {
	Nodes["00506670"] = Node{Addr: "00506670", In: sagInNames, Out: sagOutNames, F: scriptAttributeSkillGainNode,
		Emu: func(in []int64, r Rand) []int64 { return sagRun(in, r, false) }}
}

func sagRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(sagIns) {
		sagFail("wrong input count")
	}
	if n553TablesErr != nil {
		panic(n553TablesErr)
	}
	c := &sagC{m: n4Mem{}, r: r, real: real, ins: in, calls: map[string]int{}}
	for _, s := range sagSetupG {
		for k := 0; k < s.count; k++ {
			c.m.put(s.addr+uint32(k*s.step), s.size, s.val)
		}
	}
	for k := uint32(0); k < sagTxtN; k++ {
		c.m.w32(sagTxt+4*k, sagZS+2*k)
	}
	for i, f := range sagIns {
		if f.g.kind != 'a' {
			c.m.put(f.g.addr+uint32(f.k*f.g.step), f.g.size, uint32(in[i]))
		}
	}
	c.node(int32(in[0]), uint32(in[1])&0xff)
	out := make([]int64, len(sagOuts))
	for i, f := range sagOuts {
		g := f.g
		v := c.m.get(g.addr+uint32(f.k*g.step), g.size)
		out[i] = int64(v)
		if g.signed {
			out[i] = int64(int32(v << (32 - 8*uint(g.size)) >> (32 - 8*uint(g.size))))
		}
	}
	return out
}

func (c *sagC) in(name string) int64 { return c.ins[sagInAt[name]] }

// stub replays the case's stub of a callee (re/emu/emu.py apply_stub): records, count, ret.
func (c *sagC) stub(name string, args ...uint32) uint32 {
	var s *n4Stub
	for i := range sagStubs {
		if sagStubs[i].name == name {
			s = &sagStubs[i]
		}
	}
	if s == nil {
		sagFail("no stub " + name)
	}
	k := c.calls[name]
	c.calls[name] = k + 1
	for _, r := range s.recs {
		if k >= r.n {
			sagFail(fmt.Sprintf("stub %s record sequence full", name))
		}
		c.m.put(r.at+uint32(k*r.size), r.size, args[r.arg-1])
	}
	c.m.w32(s.countAt, c.m.r32(s.countAt)+1)
	switch {
	case s.rets > 0:
		if k >= s.rets {
			sagFail(fmt.Sprintf("stub %s ret sequence exhausted", name))
		}
		return c.m.r32(s.retAt + uint32(4*k))
	case s.rets < 0:
		return c.m.r32(s.retAt)
	}
	return s.retAt
}

// alloc is the emulator heap behind malloc/0x5e1ce0: bump, rounded to 8, min 8, zeroed.
func (c *sagC) alloc(n uint32) uint32 {
	a := sagHeap + c.heap
	step := (n + 7) &^ 7
	if step < 8 {
		step = 8
	}
	c.heap += step
	if c.heap > sagHeapW {
		sagFail("heap above the output window")
	}
	return a
}

// slot checks a table write lands in its 4-entry fixture buffer or on the heap.
func (c *sagC) slot(buf uint32, i int32, size, fixture uint32) uint32 {
	switch {
	case buf == fixture && i >= 0 && i < sagTabN:
	case buf >= sagHeap && buf < sagHeap+sagHeapW && i >= 0:
	default:
		sagFail("table write outside its fixture")
	}
	return buf + uint32(i)*size
}

// grow appends a record of `size` bytes to a (count, cap, buf) table, reallocating to
// count+1 entries when count+1 > cap (signed): 0x5289b0, 0x443dc0, 0x5588f0.
func (c *sagC) grow(cntA uint32, size uint32, fixture uint32) uint32 {
	m := c.m
	cnt := int32(m.r32(cntA))
	need := cnt + 1
	if need > int32(m.r32(cntA+4)) {
		old := m.r32(cntA + 8)
		nb := c.alloc(uint32(need) * size)
		m.w32(cntA+8, nb)
		m.w32(cntA+4, uint32(need))
		if cnt > 0 {
			m.copy(nb, old, int(uint32(cnt)*size))
		}
	}
	e := c.slot(m.r32(cntA+8), cnt, size, fixture)
	m.w32(cntA, uint32(cnt+1))
	return e
}

func (c *sagC) cstr(a uint32) []byte {
	var s []byte
	for ; c.m.r8(a) != 0; a++ {
		s = append(s, byte(c.m.r8(a)))
	}
	return s
}

func (c *sagC) strdup(a uint32) uint32 {
	s := append(c.cstr(a), 0)
	d := c.alloc(uint32(len(s)))
	for i, b := range s {
		c.m.w8(d+uint32(i), uint32(b))
	}
	return d
}

func (c *sagC) str(k uint32) uint32 { return c.m.r32(c.m.r32(0x68c09c) + 4*k) }

func (c *sagC) char(s uint32) uint32 { return c.m.r32(0x685174) + sagChS*s }
func (c *sagC) pres(s uint32) uint32 { return c.m.r32(0x685178) + sagPresS*s }

// ---- lists (0x5e2c70 count, 0x5e2870 get) and the NPC list *0x689f94

func (c *sagC) listCount(l uint32) int32 {
	if l == 0 {
		return 0
	}
	return int32(c.m.r32(l + 8))
}

func (c *sagC) listGet(l uint32, i int32) uint32 {
	if l == 0 || i >= c.listCount(l) {
		return 0
	}
	if i < 0 {
		sagFail("negative list index")
	}
	return c.m.r32(c.m.r32(l) + 4*uint32(i))
}

func (c *sagC) npcEntry(e uint32) uint32 {
	if e == 0 {
		sagFail("null NPC list entry (the original faults)")
	}
	return e
}

// npcFind is 0x50b830(id): the NPC whose definition ([e+6]+0x58) has this id, or 0.
func (c *sagC) npcFind(id uint32) uint32 {
	l := c.m.r32(0x689f94)
	if l == 0 {
		sagFail("null NPC list")
	}
	n := int32(c.m.r32(l + 4))
	for i := int32(0); i < n; i++ {
		e := c.npcEntry(c.m.r32(c.m.r32(l+0xc) + 4*uint32(i)))
		if c.m.r32(c.m.r32(e+6)+0x58) == id {
			return e
		}
	}
	return 0
}

// npcAt is the list entry at k (k == count reads entry 0): 0x50b8f0, 0x50c6ea.
func (c *sagC) npcAt(k int32) uint32 {
	l := c.m.r32(0x689f94)
	if l == 0 {
		sagFail("null NPC list")
	}
	if k < 0 {
		sagFail("negative NPC index")
	}
	d := c.m.r32(l + 0xc)
	if k < int32(c.m.r32(l+4)) {
		return c.npcEntry(c.m.r32(d + 4*uint32(k)))
	}
	return c.npcEntry(c.m.r32(d))
}

// monFind is 0x4e5550(line, file, id, 1): monster list *0x683fad index, else 0x2710 +
// group list *0x683fb9 index; not found asserts.
func (c *sagC) monFind(id uint32) uint32 {
	ml := c.m.r32(0x683fad)
	for i := int32(0); i < c.listCount(ml); i++ {
		if c.m.r32(c.monGet(uint32(i))) == id {
			return uint32(i)
		}
	}
	gl := c.m.r32(0x683fb9)
	for j := int32(0); j < c.listCount(gl); j++ {
		e := c.listGet(gl, j)
		if e == 0 {
			sagFail("null group entry (the original faults)")
		}
		if c.m.r32(e) == id {
			return 0x2710 + uint32(j)
		}
	}
	sagFail("srAssertFail 0x4e5550: monster id not found")
	return 0
}

// monGet is 0x4e5620(i).
func (c *sagC) monGet(i uint32) uint32 {
	var e uint32
	if i >= 0x2710 && i < 0x4e20 {
		e = c.listGet(c.m.r32(0x683fb9), int32(i-0x2710))
	} else {
		e = c.listGet(c.m.r32(0x683fad), int32(i))
	}
	if e == 0 {
		sagFail("srAssertFail 0x4e5620: monster index out of range")
	}
	return e
}

// npcMon is 0x50a3c0(npc): the monster object of a placed NPC, else 0.
func (c *sagC) npcMon(npc uint32) uint32 {
	if c.m.r8(npc+0x1a) == 0 || c.m.r8(npc+0x25) == 0 {
		return 0
	}
	return c.monGet(c.monFind(c.m.r32(npc + 0x16)))
}

// rFind is 0x510b60(id): first record with +0x18 == id in *0x683fb1 (via 0x5101b0), then
// *0x683fbd; 0 when absent.
func (c *sagC) rFind(id uint32) uint32 {
	rl := c.m.r32(0x683fb1)
	for i := int32(0); i < c.listCount(rl); i++ {
		e := c.listGet(rl, i)
		if e == 0 {
			sagFail("srAssertFail 0x5101b0: null record")
		}
		if c.m.r32(e+0x18) == id {
			return e
		}
	}
	gl := c.m.r32(0x683fbd)
	for j := int32(0); j < c.listCount(gl); j++ {
		e := c.listGet(gl, j)
		if e == 0 {
			sagFail("null record (the original faults)")
		}
		if c.m.r32(e+0x18) == id {
			return e
		}
	}
	return 0
}

// ---- trigger objects (0x443780 find, 0x43d940 close)

// find is 0x443780(name): strdup, _strupr (import stub, no-op), getClassNode(0x10008),
// srRegistry::find(cls, dup, 0) (import stub, sequenced ret), free (no-op).
func (c *sagC) find(name uint32) uint32 {
	d := c.strdup(name)
	return c.stub("find", 0, d, 0)
}

func (c *sagC) close(name uint32) {
	if o := c.find(name); o != 0 {
		c.stub("s43d940", 0xffffffff)
	}
}

func (c *sagC) clear8(name uint32) {
	if o := c.find(name); o != 0 {
		c.m.w32(o+0xa0, c.m.r32(o+0xa0)&^0x100)
	}
}

// closeIfBit19Clear: 0x506c11 / 0x506c54 / 0x506bfe.
func (c *sagC) closeIfBit19Clear(name uint32) {
	if o := c.find(name); o != 0 && c.m.r32(o+0xa0)&0x80000 == 0 {
		c.stub("s43d940", 0xffffffff)
	}
}

// ---- named counters (0x444170 find, 0x443dc0 add, 0x4440d0 get, 0x444030 set)

func (c *sagC) counterFind(key uint32) int32 {
	m := c.m
	k := c.cstr(key)
	n := int32(m.r32(sagNames))
	for i := int32(0); i < n; i++ {
		if !mkrStricmp(c.cstr(m.r32(m.r32(sagNames+8)+4*uint32(i))), k) {
			continue
		}
		own := m.r32(sagOwns + 8)
		if i < int32(m.r32(sagOwns)) {
			own += 4 * uint32(i)
		}
		if m.r32(own) == m.r32(sagMap) {
			return i
		}
	}
	return -1
}

// counterAdd is 0x443dc0(key, v): nothing on a hit, else strdup and append to the three tables.
func (c *sagC) counterAdd(key, v uint32) {
	if c.counterFind(key) != -1 {
		return
	}
	m := c.m
	d := c.strdup(key)
	m.w32(c.grow(sagNames, 4, sagNB), d)
	m.w32(c.grow(sagVals, 4, sagVB), v)
	m.w32(c.grow(sagOwns, 4, sagOB), m.r32(sagMap))
}

func (c *sagC) counterGet(key uint32) uint32 {
	i := c.counterFind(key)
	b := c.m.r32(sagVals + 8)
	if i < int32(c.m.r32(sagVals)) {
		return c.m.r32(b + 4*uint32(i))
	}
	return c.m.r32(b)
}

func (c *sagC) counterSet(key, v uint32) {
	i := c.counterFind(key)
	if i == -1 {
		sagFail("srAssertFail 0x444030: counter not found")
	}
	if i < int32(c.m.r32(sagVals)) {
		c.m.w32(c.slot(c.m.r32(sagVals+8), i, 4, sagVB), v)
	}
}

// counterOr is 0x5077d5 / 0x507879: add the counter with v, or OR v into it.
func (c *sagC) counterOr(key, v uint32) {
	if c.counterFind(key) == -1 {
		c.counterAdd(key, v)
		return
	}
	c.counterSet(key, c.counterGet(key)|v)
}

// ---- messages, facts, events, factions

// message is 0x5289b0(kind, x).
func (c *sagC) message(kind, x uint32) {
	m := c.m
	o := c.alloc(0x24)
	m.w32(o, 0xffffffff)
	m.w32(o+0xc, kind)
	m.w32(o+0x10, x)
	m.w32(o+0x20, m.r32(sagQStamp))
	m.w32(c.grow(sagQCnt, 4, sagQBuf), o)
}

// message528830 is 0x528830(0, 0, 0, 0): a zeroed message with the stamp, appended.
func (c *sagC) message528830() {
	m := c.m
	o := c.alloc(0x24)
	m.w32(o+0x20, m.r32(sagQStamp))
	m.w32(c.grow(sagQCnt, 4, sagQBuf), o)
}

// set is 0x5061a0(id, val, 0); the debug text (0x683f8c, 0x688290) is off in the case.
func (c *sagC) set(id, val uint32) {
	old := c.m.r8(sagFacts + id)
	c.m.w8(sagFacts+id, val)
	if old != val {
		c.event(id)
	}
	c.node(int32(id), val)
}

// event is 0x5588f0(id): append {map, id, zero-extended low byte of 0x506280(id)} to the
// event log *0x68de40; the follow-up text (0x68de44, 0x6850d5, 0x68edcc) is off in the case.
func (c *sagC) event(id uint32) {
	m := c.m
	evl := m.r32(0x68de40)
	al := c.stub("s506280", id)
	e := c.grow(evl+4, 12, sagEvBuf)
	m.w32(e, m.r32(sagMap))
	m.w32(e+4, id)
	// 0x5588f0 stores only al into a 4-byte local and copies the whole dword: the upper
	// bytes are whatever that stack slot held (normally 0; see the 0xcd/0xe0 handler).
	m.w32(e+8, c.evHi|al&0xff)
	c.evHi = 0
	if int8(m.r8(m.r32(0x6836ac)+id*0x1d8+0x37)) > 2 {
		return
	}
	c.stub("s506280", id)
}

// reset is 0x535b40(f, cls): standing 0x19 / 0x37 / 0x4b and the faction log on a change.
func (c *sagC) reset(f uint32, cls int) {
	m := c.m
	if f >= 0x15 {
		sagFail("srAssertFail 0x535b40: faction")
	}
	base := 0x68d6e8 + 14*f
	oc := mkrClass(int8(m.r8(base)))
	m.w8(base, []uint32{0x19, 0x37, 0x4b}[cls])
	if cls == oc {
		return
	}
	m.w32(base+6, m.r32(sagClock))
	code := uint32(5)
	if cls < oc {
		code = 0
	}
	c.stub("s58aad0", code, c.str(0x247), c.str(m.r16(0x61eacc+2*f)))
}

// ---- NPC moves

// npcRemove is the body of 0x50c680 after the lookup (also 0x50c440's same-map path).
func (c *sagC) npcRemove(n uint32) {
	m := c.m
	if m.r8(n+0x1a) == 0 {
		return
	}
	if m.r8(n+0x25) != 0 {
		if t := c.monGet(c.monFind(m.r32(n + 0x16))); t != 0 {
			c.stub("s4e3af0", c.monFind(m.r32(t)), 1)
		}
	}
	l := m.r32(0x689f94)
	k := int32(m.r8(n + 0x2c))
	if k > int32(m.r32(l+4)) {
		return
	}
	e := c.npcAt(k)
	m.w8(e+0x1a, 0)
	c.stub("s55a0a0", m.r32(e))
	m.w32(e, 0)
	if m.r8(m.r32(e+6)+0x54) != 0 {
		m.w8(e+0xc7, 1)
	}
}

// remove is 0x50c680(id).
func (c *sagC) remove(id uint32) {
	n := c.npcFind(id)
	if n == 0 {
		sagFail("0x50c680: NPC absent (the original faults)")
	}
	c.npcRemove(n)
}

// leave is 0x50c440(npc, b).
func (c *sagC) leave(npc, b uint32) {
	m := c.m
	if int32(int8(b)) != int32(m.r32(sagMap)) {
		m.w8(npc+0x112, 1)
		m.w8(npc+0x113, b)
		return
	}
	n := c.npcFind(m.r8(npc + 0x2e))
	if n == 0 {
		sagFail("0x50c440: NPC absent (the original faults)")
	}
	c.npcRemove(n)
}

// arrive is 0x50c1c0(id, b, s).
func (c *sagC) arrive(id, b, s uint32) {
	m := c.m
	n := c.npcFind(id)
	if n == 0 {
		n = c.stub("s509aa0", id)
	}
	if int32(int8(b)) == int32(m.r32(sagMap)) {
		c.stub("s50c560", n, s)
		m.w8(n+0xea, 0)
		return
	}
	m.w8(n+0xc5, 1)
	src := append(c.cstr(s), 0)
	for i, x := range src {
		m.w8(n+0x9d+uint32(i), uint32(x))
	}
	m.w8(n+0xc6, b)
}

// flagNpc is 0x50ca30(id).
func (c *sagC) flagNpc(id uint32) {
	if n := c.npcFind(id); n != 0 {
		c.m.w8(n+0xe9, 1)
		c.m.w8(n+0x114, 1)
	}
}

// partyHas is 0x50b8f0(7): a party NPC slot holds NPC id 7 while its character is fit.
func (c *sagC) partyHas() bool {
	for s := uint32(0); s < 2; s++ {
		p := c.pres(s)
		if c.m.r8(p) == 0 {
			continue
		}
		e := c.npcAt(int32(c.m.r32(p + 0xfa)))
		if c.m.r8(e+0xc7) != 0 {
			sagFail("0x50b8f0: dead party NPC (the original faults)")
		}
		if c.m.r8(e+0x2e) == 7 && c.m.r32(c.char(s)+0xb01) < 0xf {
			return true
		}
	}
	return false
}

// ---- stock (0x55aa80)

func (c *sagC) shop(npc, item, dt uint32) {
	m := c.m
	if npc == 0 {
		return
	}
	r := c.stub("s55a7b0", npc, item, 1)
	if c.real {
		r = c.realAdd(npc, item)
	}
	if r == 0xffffffff {
		return
	}
	e := c.listGet(m.r32(npc+0xa), int32(r))
	if e == 0 {
		sagFail("0x55aa80: stock entry absent (the original faults)")
	}
	m.w32(e, m.r32(sagClock)+dt)
}

// realAdd is 0x55a7b0(npc, item, 1) run by shopStockAddNode on a callee world made of the
// NPC's stock list; the result is written back (new entries into the fixture slots).
func (c *sagC) realAdd(npc, item uint32) uint32 {
	m := c.m
	i := (npc - sagNpc0) / sagNpcS
	if npc < sagNpc0 || (npc-sagNpc0)%sagNpcS != 0 || i >= sagNpcN {
		sagFail("0x55a7b0: merchant is not a fixture NPC")
	}
	stk := sagStk0 + sagStkS*i
	list := m.r32(npc + 0xa)
	if list != 0 && list != stk {
		sagFail("0x55a7b0: stock list is not the NPC's fixture list")
	}
	n := int64(c.listCount(list))
	if n < 0 || n > sagStkN || (list != 0 && m.r32(list) != stk+0x10) {
		sagFail("0x55a7b0: stock list past the fixture")
	}
	j := -1
	for x, it := range sagItems {
		if it == item {
			j = x
		}
	}
	if j < 0 {
		sagFail("0x55a7b0: item without F-only class/kind")
	}
	ids := map[int64]int64{}
	rev := []int64{}
	mp := func(x int64) int64 {
		if v, ok := ids[x]; ok {
			return v
		}
		if len(rev) >= ssaItemN {
			sagFail("0x55a7b0: more than 16 distinct item ids")
		}
		ids[x] = int64(len(rev))
		rev = append(rev, x)
		return ids[x]
	}
	v := map[string]int64{"merchant": ssaMer, "id": mp(int64(item)), "cnt": 1,
		"cls": int64(m.r8(sagFOnly + 0x10*uint32(j))), "kind": int64(m.r8(sagFOnly + 0x10*uint32(j) + 1)), "cap": 10, "count": n}
	if list != 0 {
		v["listp"] = ssaList
		v["cap"] = int64(m.r32(list + 4))
	}
	for k := int64(0); k < n; k++ {
		d := m.r32(stk + 0x10 + 4*uint32(k))
		if d != 0 {
			if d != stk+0x40+0x20*uint32(k) {
				sagFail("0x55a7b0: stock slot is not the fixture entry")
			}
			v["d"+fmt.Sprint(k)] = ssaEnts + 0x20*k
			v[fmt.Sprintf("e%d.id", k)] = mp(int64(m.r32(d + 4)))
			v[fmt.Sprintf("e%d.q", k)] = int64(m.r8(d + 8))
			v[fmt.Sprintf("e%d.c", k)] = int64(m.r8(d + 0x10))
		}
	}
	vec := make([]int64, len(ssaIn))
	for x, name := range ssaIn {
		vec[x] = v[name]
	}
	out := shopStockAddNode(vec, nil)
	at := map[string]int64{}
	for x, name := range ssaOut {
		at[name] = out[x]
	}
	heap := func(p int64) int64 {
		if p < ssaHeap || p+4 > ssaHeap+ssaWin {
			sagFail("0x55a7b0: callee pointer outside its heap window")
		}
		return at[fmt.Sprintf("h%d", (p-ssaHeap)/4)]
	}
	hdr := func(o int64) int64 { return at[fmt.Sprintf("list%x", o)] }
	if list == 0 {
		lp := at["merList"]
		hdr = func(o int64) int64 { return heap(lp + o) }
	}
	arr := hdr(0)
	slot := func(k int64) int64 {
		if arr == ssaData {
			return at[fmt.Sprintf("data%d", k)]
		}
		return heap(arr + 4*k)
	}
	cnt2 := hdr(8)
	if cnt2 > sagStkN {
		sagFail("0x55a7b0: stock list grew past the 6-entry fixture")
	}
	if list == 0 {
		m.w32(npc+0xa, stk)
		m.w32(stk, stk+0x10)
		m.w32(stk+4, 10)
	}
	word := func(p, o int64) int64 {
		if p >= ssaHeap {
			return heap(p + o)
		}
		return at[fmt.Sprintf("ent%d.%x", (p-ssaEnts)/0x20, o)]
	}
	for k := int64(0); k < cnt2; k++ {
		p := slot(k)
		b := stk + 0x40 + 0x20*uint32(k)
		if k < n {
			if p != 0 {
				m.w32(b+4, uint32(rev[word(p, 4)]))
				m.w8(b+8, uint32(word(p, 8)))
				m.w8(b+0x10, uint32(word(p, 0x10)))
			}
			continue
		}
		m.w32(stk+0x10+4*uint32(k), b)
		m.w32(b, uint32(word(p, 0)))
		m.w32(b+4, uint32(rev[word(p, 4)]))
		m.w32(b+8, uint32(word(p, 8)))
		m.w32(b+0xc, uint32(word(p, 0xc)))
		m.w32(b+0x10, uint32(word(p, 0x10)))
	}
	m.w32(stk+8, uint32(cnt2))
	ret := uint32(at["ret"])
	if ret != 0xffffffff && int64(ret) >= cnt2 {
		sagFail(fmt.Sprintf("0x55a7b0: returned index %d past the stock list (%d)", ret, cnt2))
	}
	return ret
}

// ---- characters (0x553c10, 0x553ad0 and 0x553cd0 through n553State)

func (c *sagC) skillState(ch uint32) *n553State {
	m := c.m
	var v n553In
	v.class = int64(m.r32(ch + 0x69))
	v.race = int64(int32(m.r32(ch + 0x71)))
	if v.class >= n553Classes {
		sagFail("class outside 0..14")
	}
	for k := 0; k < n553Skills; k++ {
		b := ch + 0x19d + sagSkS*uint32(k)
		v.f19d[k] = int64(m.r8(b))
		v.raw[k] = int64(m.r32(b + 2))
		v.bonus[k] = int64(int8(m.r8(ch + 0x1783 + uint32(k))))
		v.ded[k] = int64(m.r32(ch + 0xded + 4*uint32(k)))
	}
	for k := range v.b8d {
		v.b8d[k] = int64(m.r32(ch + 0x8d + 4*uint32(k)))
	}
	for k := range v.att {
		v.att[k] = int64(m.r32(ch + 0xe5 + 20*uint32(k)))
	}
	spr := m.r32(0x65be1c)
	for k := range v.spv {
		v.spv[k] = int64(int32(m.r32(ch + 0xbd1 + 4*uint32(k))))
		v.sch[k] = int64(m.r32(spr + 0x1bf*uint32(k) + 0x2f2))
	}
	return &n553State{in: v, t: &n553Tables}
}

// refresh is 0x553cd0(ch): the usable flags; on the mode-3 screen each change calls the
// stubbed 0x5b1af0(k) / 0x5b1b30(k).
func (c *sagC) refresh(ch uint32, st *n553State) {
	before := st.in.f19d
	st.refresh()
	mode3 := c.m.r32(sagMode) == 3
	for k := 0; k < n553Skills; k++ {
		c.m.w8(ch+0x19d+sagSkS*uint32(k), uint32(st.in.f19d[k]))
		if !mode3 || before[k] == st.in.f19d[k] {
			continue
		}
		if st.in.f19d[k] != 0 {
			c.stub("s5b1af0", uint32(k))
		} else {
			c.stub("s5b1b30", uint32(k))
		}
	}
}

// skill7 is 0x553c10(ch, 7).
func (c *sagC) skill7(ch uint32) {
	st := c.skillState(ch)
	c.refresh(ch, st)
	c.m.w32(ch+0x1a3+sagSkS*7, uint32(st.effective(7)))
	c.stub("s51d960", ch)
	c.stub("s4ed9d0", ch)
}

// attr1 is 0x553ad0(ch, 1).
func (c *sagC) attr1(ch uint32) {
	m := c.m
	a := m.r32(ch + 0xf9)
	f := ch + 0x19d + sagSkS*0x23
	mode := m.r32(sagMode)
	if a >= 100 {
		if m.r8(f) == 0 {
			m.w8(f, 1)
			if mode == 3 {
				c.stub("s5b1af0", 0x23)
			}
			if mode == 7 {
				// 0x517a90 formats the skill name into 0x689fd0 (text not compared) and
				// returns it; 0x569a50(text, 0, 1, 0).
				c.stub("s569a50", 0x689fd0, 0, 1, 0)
			}
		}
	} else if m.r8(f) != 0 {
		m.w8(f, 0)
		if mode == 3 {
			c.stub("s5b1b30", 0x23)
		}
	}
	e := int32(int8(m.r8(ch+0x177d))) + int32(a)
	if e > 0x7d {
		e = 0x7d
	} else if e < 1 {
		e = 1
	}
	m.w32(ch+0xfd, uint32(e))
	c.stub("s51d960", ch)
	c.stub("s4ed9d0", ch)
	for r := uint32(0); r < n553Skills; r++ {
		x0, x1 := m.r32(0x6155b8+16*r), m.r32(0x6155b8+16*r+4)
		m.w32(ch+0x1a7+sagSkS*r, (m.r32(ch+0xe5+20*x1)+m.r32(ch+0xe5+20*x0))>>1)
	}
}

// ---- 0x529be0 / 0x529ef0 screens

func (c *sagC) f53a320() {
	m := c.m
	m.w32(0x683fe7, 1)
	if m.r32(0x683fdb) != 3 { // jump table 0x53a3b4[0] -> 3
		c.stub("s55ee70", 3)
	}
	m.w32(0x68407f, 0)
	m.w32(0x684083, 0)
	m.w32(0x684087, 0)
	c.stub("s4add30", 0)
	if m.r32(sagMode) == 7 {
		c.stub("s56aab0")
	}
}

// f4f2260 is 0x4f2260(i): clear bits 2-3 and 16-31 of the rows 0x61f23c[i]..0x61f240[i].
func (c *sagC) f4f2260(i uint32) {
	m := c.m
	if i >= m.r32(0x617b18) {
		sagFail("srAssertFail 0x4f2260")
	}
	for j := m.r32(0x61f23c + 12*i); j <= m.r32(0x61f240+12*i); j++ {
		if j >= m.r32(0x617b1c) {
			sagFail("srAssertFail 0x4f2260 row")
		}
		a := 0x620048 + 28*j
		m.w32(a, m.r32(a)&0xfff3)
	}
}

func (c *sagC) f529be0() {
	m := c.m
	m.w8(0x68c4f7, 1)
	c.stub("s41f0d0")
	m.w8(0x68506f, 1)
	c.f53a320()
	for s := uint32(0); s < 8; s++ {
		if m.r8(c.pres(s)) != 0 {
			c.stub("s4f21e0", 7+s)
			c.f4f2260(7 + s)
			c.stub("s4f2380", 0x5a+s)
		}
	}
}

func (c *sagC) f529ef0() {
	m := c.m
	m.w8(0x68c4f7, 1)
	m.w8(0x6875a5, 1) // 0x577520
	c.stub("s41f0d0")
	c.stub("s55ee70", 9)
	m.w8(0x68c4f9, 1)
	m.w8(0x68c4fa, 1)
	c.stub("s483fd0", 0xc47a0000)
	for s := uint32(0); s < 8; s++ {
		ch := c.char(s)
		if m.r8(c.pres(s)) != 0 && (m.r32(ch+0xb11) != 0 || m.r32(ch+0xb01) < 0x12) && s != m.r32(0x6875ef) {
			c.stub("s522fe0", s, 0x11, 0x270f, 0, 0, 0)
		}
	}
}

// ---- dispatcher

func (c *sagC) node(ev int32, flag uint32) {
	m := c.m
	f := flag != 0
	// Debug byte 0x688290 is fixed 0 (4e46f0 precedent); its log paths in 0x37, 0xcd/0xe0
	// and 0x203 are cov-exempt, so a set byte must not return silently.
	if m.r8(0x688290) != 0 {
		sagFail("debug byte 0x688290 set")
	}
	cls := 0
	if f {
		cls = 2
	}
	shopItem := map[int32]uint32{0x17: 0x28a, 0x18: 0x28b, 0x19: 0x28c, 0x1a: 0x28d, 0x1b: 0x28e}
	msgF := map[int32]uint32{0x28: 0x2b, 0xb7: 0x2e, 0x16f: 0x22, 0x1a5: 0x26, 0x173: 0x23, 0x1b3: 0x1c, 0x1b5: 0x1b,
		0x1b4: 0x1a, 0x214: 0x2f, 0x27c: 0x2d, 0x2ef: 0x2c,
		0x2ed: 0x2a}
	msgAny := map[int32]uint32{0x19c: 0x24, 0x197: 0x25, 0x14f: 0x12, 0x157: 0xf, 0x1a0: 0x10, 0x208: 0x15, 0x206: 0x16,
		0x1f2: 0x20, 0x20f: 0x30, 0x215: 0x32, 0x1e7: 0x13, 0x213: 0x3d, 0x21b: 0x34, 0x22a: 0x21, 0x2a5: 0x29, 0x2f3: 0x33, 0x2f0: 0x31,
		0x320: 0x35, 0x321: 0x35}
	resets := map[int32]uint32{0xca: 0xc, 0xcb: 0xa, 0xc9: 0xb, 0xcc: 9, 0x1a8: 0xf, 0x183: 4, 0x11e: 5}
	flagNpcs := map[int32]uint32{0x64: 0x29, 0x179: 0x2b, 0x1c9: 0x44, 0x1ec: 0x56, 0x204: 0x52}
	if it, ok := shopItem[ev]; ok {
		if f {
			c.shop(c.npcFind(3), it, 0x15180)
		}
		return
	}
	if k, ok := msgF[ev]; ok {
		if f {
			c.message(k, 0)
		}
		return
	}
	if k, ok := msgAny[ev]; ok {
		c.message(k, 0)
		return
	}
	if k, ok := resets[ev]; ok {
		c.reset(k, cls)
		return
	}
	if k, ok := flagNpcs[ev]; ok {
		if f {
			c.flagNpc(k)
		}
		return
	}
	switch ev {
	case 0x16c:
		if f {
			c.shop(c.npcFind(0x48), 0x242, 0)
		}
	case 0x19e:
		if f {
			c.shop(c.npcFind(0x49), 0x1b0, 0x15180)
		}
	case 0x1dc, 0x1dd, 0x1de:
		if f {
			c.shop(c.npcFind(0x39), []uint32{0x1f4, 0x1f5, 0x1f8}[ev-0x1dc], 0x15180)
		}
	case 0x1f:
		if f && !c.partyHas() {
			if n := c.npcFind(7); n != 0 {
				c.leave(n, 0)
				c.arrive(m.r8(n+0x2e), 0x11, 0x619d88)
			}
		}
	case 0x22:
		if f {
			n := c.npcFind(0x48)
			if n == 0 {
				sagFail("srAssertFail 0x5067c3: NPC 0x48 absent")
			}
			for k := uint32(0); k < 40; k++ {
				if int16(m.r16(n+0x30+2*k)) == 0x242 {
					m.w16(n+0x30+2*k, 0xffff)
					break
				}
			}
		}
	case 0x314:
		c.reset(0x10, cls)
		c.reset(0xf, cls)
	case 0xb6:
		if f {
			c.set(0x308, 1)
		}
	case 0x19b:
		if f {
			c.set(0x273, 1)
		}
	case 0xcd, 0xe0:
		if f {
			if c.partyHas() {
				c.set(0xe2, 1)
				// 5061a0's call to 506670 left its return address 0x506218 in the slot
				// that the next 5061a0 -> 5588f0 uses for the event value.
				c.evHi = 0x506200
			}
			c.set(0x24d, 0)
			c.evHi = 0
		}
	case 0x1f1:
		if f {
			c.remove(0x55)
			c.message(0x39, 0)
		}
	case 0x1c2:
		if f {
			c.f529ef0()
		} else {
			c.message(0x41, 0)
		}
	case 0x2a6:
		if !f {
			c.message(0x3b, 0)
		}
	case 0xc4:
		c.close(0x619d38)
	case 0xac:
		c.close(0x619d2c)
		c.clear8(0x619d14)
	case 0xc7:
		c.close(0x619d08)
	case 0xba:
		c.close(0x612c7c)
		c.close(0x619cfc)
	case 0xa5:
		if f {
			c.close(0x619cbc)
			c.counterAdd(0x619cb0, 1)
		}
	case 0xa6:
		if f {
			c.close(0x619c9c)
		}
	case 0x3a:
		o := c.find(0x619c8c)
		if o == 0 {
			sagFail("srAssertFail 0x506d6e: trigger absent")
		}
		c.stub("s43d940", 0xffffffff)
	case 0x188:
		if f {
			c.close(0x619c18)
			c.close(0x619c00)
		}
	case 0x220, 0x221:
		if f {
			c.close(0x619bac)
		}
	case 0x225, 0x226:
		if f {
			c.close(0x619b94)
			c.close(0x619b7c)
		}
	case 0x222:
		if f {
			c.close(0x619b1c)
		}
	case 0x91:
		if f {
			c.closeIfBit19Clear(0x619cd4)
		}
	case 0xa9:
		if f {
			c.closeIfBit19Clear(0x619ccc)
		}
	case 0xa8:
		if f {
			c.closeIfBit19Clear(0x619cdc)
		}
	case 0x39:
		if f {
			s := int32(m.r32(0x68760c))
			if s < 0 || s > 7 {
				sagFail("current character outside 0..7")
			}
			if m.r32(c.char(uint32(s))+0xa4d) != 0 {
				c.stub("s523330", uint32(s), 0x13, 1)
			}
		}
	case 0xc6:
		if r := c.rFind(0x13); r != 0 {
			c.stub("s547570", r, 1, 0)
		}
	case 0x33:
		if f && c.stub("s4512c0", 0x619cec, 0, 0, 0)&0xff != 0 {
			for k := 0; k < 2; k++ {
				if x := c.stub("s4f6c50", 0x2eb, 0, 3, 1); x != 0 {
					c.stub("s4f6cf0", x)
				}
			}
		}
	case 0x37:
		if m.r8(sagFacts+0x36) == 0 && c.stub("s4512c0", 0x619ce4, 0, 0, 0)&0xff != 0 {
			c.stub("s4d8f00", 0x131, 1, 0, 1, 1, 1, 0)
		}
	case 0x72:
		c.remove(0x2c)
	case 0x44:
		if f {
			if n := c.npcFind(0x20); n != 0 {
				c.stub("s55a0a0", m.r32(n))
				c.stub("s524ca0", n)
				if m.r8(n+0x26) != 0 {
					var p uint32
					if m.r8(m.r32(n+6)+0x57) != 0 {
						p = c.m.r32(0x685174) + sagChS*uint32(int32(int8(m.r8(n+0x2b))))
					}
					c.stub("s52e690", p, 0, 0, m.r32(0x5ed8c8), m.r32(0x5ed914))
				} else {
					m.w8(0x68c4f4, 1) // 0x529560
					c.message528830()
				}
			}
		}
	case 0x89, 0x21e:
		if f {
			if c.npcFind(0x29) == 0 {
				sagFail("srAssertFail: NPC 0x29 absent")
			}
			c.flagNpc(0x29)
		}
	case 0x86:
		if f {
			for s := uint32(0); s < 8; s++ {
				ch := c.char(s)
				if m.r8(c.pres(s)) != 0 && m.r32(ch+0xb01) < 0x12 {
					cl := m.r32(ch + 0x69)
					if cl >= n553Classes {
						sagFail("class outside 0..14")
					}
					if m.r32(0x6159e4+4*cl) != 0 && m.r32(ch+0x2a9) < 10 {
						m.w32(ch+0x2a9, 10)
						c.skill7(ch)
					}
				}
			}
		}
	case 0x195:
		if f {
			for s := uint32(0); s < 8; s++ {
				ch := c.char(s)
				if m.r8(c.pres(s)) != 0 && m.r32(ch+0xb11) != 0 && m.r32(ch+0xb01) < 0x12 {
					a := m.r32(ch + 0xf9)
					if 100-a > 5 {
						m.w32(ch+0xf9, a+5)
					} else {
						m.w32(ch+0xf9, 100)
					}
					c.attr1(ch)
				}
			}
			c.stub("s5299b0", c.str(0x1dc))
			c.clear8(0x619c7c)
		}
	case 0xe8:
		if f {
			c.close(0x619c70)
			c.clear8(0x619c60)
		} else {
			c.close(0x619c60)
		}
	case 0xdb:
		if n := c.npcFind(7); n != 0 {
			if o := c.npcMon(n); o != 0 {
				c.stub("s4c7f10", 0x619c54, 1)
			}
		}
	case 0x151, 0x155, 0x153:
		k := map[int32][3]uint32{0x151: {0x6126b8, 0x619c48, 0x3e}, 0x155: {0x612620, 0x619c3c, 0x3f}, 0x153: {0x612588, 0x619c30, 0x40}}[ev]
		c.clear8(k[0])
		c.close(k[1])
		c.message(k[2], 0)
	case 0x1b8:
		if f {
			c.flagNpc(0x3c)
			c.close(0x619bf0)
		}
	case 0x1ba:
		if f {
			c.flagNpc(0x3a)
			c.close(0x619be0)
		}
	case 0x1bf:
		if f {
			c.f529be0()
		}
	case 0x219:
		m.w32(0x689aff, 1)
	case 0x21c:
		m.w32(0x689b03, 1)
	case 0x203:
		if f {
			c.stub("s420f90")
			if m.r8(sagFacts+0x177) != 0 {
				c.close(0x619bd0)
			} else {
				c.close(0x619bc0)
			}
		}
	case 0x22b:
		if f {
			c.flagNpc(0x18)
			c.stub("s56e800", 0)
		}
	case 0x224:
		if f {
			c.close(0x6134c4)
			c.close(0x619b6c)
			c.close(0x619b58)
			c.counterOr(0x6134cc, 1)
		}
	case 0x223:
		if f {
			c.close(0x6134dc)
			c.close(0x619b44)
			c.close(0x619b30)
			c.counterOr(0x6134cc, 2)
		}
	case 0x234, 0x235:
		if c.stub("s408860", 0x619af4) != 0xffffffff {
			m.w8(0x68c4f7, 1) // 0x529bc0
		}
		if o := c.stub("s451080", m.r32(0x659ab4), 0x619ae8); o != 0 {
			c.stub("s49acd0", 0)
			m.w32(o+0x264, 0)
			m.w32(o+0x188, 0)
			c.stub("s49acd0", 1)
			c.stub("s498d90", 1)
		}
		if n := c.npcFind(0x49); n != 0 {
			if q := c.npcMon(n); q != 0 {
				b := m.r8(m.r32(m.r32(q+0xc)+0x1d8) + 0xa7)
				if c.stub("s4c2cf0", b)&0xff != 0 {
					c.stub("s4e4db0", q, 0x14, 1)
				}
			}
		}
	case 0x259:
		if n := c.npcFind(0x57); n != 0 && m.r8(n+0x1d) == 0 {
			c.remove(0x57)
		}
	case 0x2f4:
		if f {
			m.w32(0x689afb, 1)
			for _, x := range [][2]uint32{{0x234, 2}, {0x1b4, 1}, {0x1b9, 1}} {
				if r := c.rFind(x[0]); r != 0 {
					c.stub("s547570", r, x[1], 0)
				}
			}
			c.npcFind(0x89)
		}
	case 0x322:
		c.stub("s56e800", 0)
		c.stub("s41f0d0")
		if r := c.rFind(0xc2); r != 0 {
			c.monGet(c.monFind(m.r32(r + 0x9f)))
			c.stub("s4c7f10", 0x619ad4, 1)
		}
		c.stub("s408ad0", 0x619ab0, 0)
		e := c.stub("s4ae080", 0x40c00000, 0, 0x3f800000, 0, 0)
		m.w32(e, m.r32(e)|0x20)
		m.w32(e+0x48, 0x5092d0)
	case 0x15f:
		if f {
			c.flagNpc(0x3b)
			m.w8(0x68d6f2+14*0x10, 1) // 0x536110(0x10, 1)
		}
	}
}

// Code below is generated by scratch gen.py from the same lists as re/emu/cases/00506670.json; do not edit.

var sagInG = []n4Grp{
	{"ev", 'a', 0x0, 4, true, 1, 0, "", 0},
	{"flag", 'a', 0x1, 1, false, 1, 0, "", 0},
	{"mode", 'g', 0x68ec78, 4, false, 1, 0, "", 0},
	{"map", 'g', 0x686a70, 4, true, 1, 0, "", 0},
	{"clock", 'g', 0x686a48, 4, false, 1, 0, "", 0},
	{"qstamp", 'g', 0x68c4ac, 4, false, 1, 0, "", 0},
	{"qcnt", 'g', 0x68c4c0, 4, true, 1, 0, "", 0},
	{"qcap", 'g', 0x68c4c4, 4, true, 1, 0, "", 0},
	{"sel", 'g', 0x68760c, 4, true, 1, 0, "", 0},
	{"skip", 'g', 0x6875ef, 4, true, 1, 0, "", 0},
	{"st3", 'g', 0x683fdb, 4, false, 1, 0, "", 0},
	{"fact36", 'g', 0x689bae, 1, false, 1, 0, "", 0},
	{"fact177", 'g', 0x689cef, 1, false, 1, 0, "", 0},
	{"facte2", 'g', 0x689c5a, 1, false, 1, 0, "", 0},
	{"fact24d", 'g', 0x689dc5, 1, false, 1, 0, "", 0},
	{"fact308", 'g', 0x689e80, 1, false, 1, 0, "", 0},
	{"fact273", 'g', 0x689deb, 1, false, 1, 0, "", 0},
	{"lvle2", 'o', 0x2009a0e7, 1, true, 1, 0, "", 0},
	{"lvl24d", 'o', 0x200c3e2f, 1, true, 1, 0, "", 0},
	{"lvl308", 'o', 0x200d96f7, 1, true, 1, 0, "", 0},
	{"lvl273", 'o', 0x200c843f, 1, true, 1, 0, "", 0},
	{"evcnt", 'o', 0x20026404, 4, true, 1, 0, "", 0},
	{"evcap", 'o', 0x20026408, 4, true, 1, 0, "", 0},
	{"ncnt", 'g', 0x6598fc, 4, true, 1, 0, "", 0},
	{"ncap", 'g', 0x659900, 4, true, 1, 0, "", 0},
	{"vcnt", 'g', 0x659994, 4, true, 1, 0, "", 0},
	{"vcap", 'g', 0x659998, 4, true, 1, 0, "", 0},
	{"ocnt", 'g', 0x6598e4, 4, true, 1, 0, "", 0},
	{"ocap", 'g', 0x6598e8, 4, true, 1, 0, "", 0},
	{"name", 'o', 0x20026600, 4, false, 4, 4, "", 0},
	{"val", 'o', 0x20026640, 4, false, 4, 4, "", 0},
	{"own", 'o', 0x20026680, 4, false, 4, 4, "", 0},
	{"stand4", 'g', 0x68d720, 1, true, 1, 0, "", 0},
	{"stand5", 'g', 0x68d72e, 1, true, 1, 0, "", 0},
	{"stand9", 'g', 0x68d766, 1, true, 1, 0, "", 0},
	{"standa", 'g', 0x68d774, 1, true, 1, 0, "", 0},
	{"standb", 'g', 0x68d782, 1, true, 1, 0, "", 0},
	{"standc", 'g', 0x68d790, 1, true, 1, 0, "", 0},
	{"standf", 'g', 0x68d7ba, 1, true, 1, 0, "", 0},
	{"stand10", 'g', 0x68d7c8, 1, true, 1, 0, "", 0},
	{"t620_", 'g', 0x620400, 4, false, 48, 28, "", 0},
	{"pres0", 'o', 0x2001d000, 1, false, 1, 0, "", 0},
	{"c0class", 'o', 0x20010069, 4, false, 1, 0, "", 0},
	{"c0race", 'o', 0x20010071, 4, true, 1, 0, "", 0},
	{"c0b01", 'o', 0x20010b01, 4, false, 1, 0, "", 0},
	{"c0b11", 'o', 0x20010b11, 4, false, 1, 0, "", 0},
	{"c0a2a9", 'o', 0x200102a9, 4, false, 1, 0, "", 0},
	{"c0a4d", 'o', 0x20010a4d, 4, false, 1, 0, "", 0},
	{"c0att", 'o', 0x200100e5, 4, false, 7, 20, "", 0},
	{"c0ab1", 'o', 0x2001177d, 1, true, 1, 0, "", 0},
	{"c0b8d", 'o', 0x2001008d, 4, false, 15, 4, "", 0},
	{"c0f", 'o', 0x2001019d, 1, false, 41, 38, "", 0},
	{"c0bon7", 'o', 0x2001178a, 1, true, 1, 0, "", 0},
	{"c0ded", 'o', 0x20010e5d, 4, false, 6, 4, "", 0},
	{"c0spv", 'o', 0x20010bd1, 4, true, 113, 4, "", 0},
	{"pres1", 'o', 0x2001d106, 1, false, 1, 0, "", 0},
	{"c1class", 'o', 0x200118cb, 4, false, 1, 0, "", 0},
	{"c1race", 'o', 0x200118d3, 4, true, 1, 0, "", 0},
	{"c1b01", 'o', 0x20012363, 4, false, 1, 0, "", 0},
	{"c1b11", 'o', 0x20012373, 4, false, 1, 0, "", 0},
	{"c1a2a9", 'o', 0x20011b0b, 4, false, 1, 0, "", 0},
	{"c1a4d", 'o', 0x200122af, 4, false, 1, 0, "", 0},
	{"c1att", 'o', 0x20011947, 4, false, 7, 20, "", 0},
	{"c1ab1", 'o', 0x20012fdf, 1, true, 1, 0, "", 0},
	{"c1b8d", 'o', 0x200118ef, 4, false, 15, 4, "", 0},
	{"c1f", 'o', 0x200119ff, 1, false, 41, 38, "", 0},
	{"c1bon7", 'o', 0x20012fec, 1, true, 1, 0, "", 0},
	{"c1ded", 'o', 0x200126bf, 4, false, 6, 4, "", 0},
	{"c1spv", 'o', 0x20012433, 4, true, 113, 4, "", 0},
	{"pres2", 'o', 0x2001d20c, 1, false, 1, 0, "", 0},
	{"c2class", 'o', 0x2001312d, 4, false, 1, 0, "", 0},
	{"c2race", 'o', 0x20013135, 4, true, 1, 0, "", 0},
	{"c2b01", 'o', 0x20013bc5, 4, false, 1, 0, "", 0},
	{"c2b11", 'o', 0x20013bd5, 4, false, 1, 0, "", 0},
	{"c2a2a9", 'o', 0x2001336d, 4, false, 1, 0, "", 0},
	{"c2a4d", 'o', 0x20013b11, 4, false, 1, 0, "", 0},
	{"c2att", 'o', 0x200131a9, 4, false, 7, 20, "", 0},
	{"c2ab1", 'o', 0x20014841, 1, true, 1, 0, "", 0},
	{"c2b8d", 'o', 0x20013151, 4, false, 15, 4, "", 0},
	{"c2f", 'o', 0x20013261, 1, false, 41, 38, "", 0},
	{"c2bon7", 'o', 0x2001484e, 1, true, 1, 0, "", 0},
	{"c2ded", 'o', 0x20013f21, 4, false, 6, 4, "", 0},
	{"c2spv", 'o', 0x20013c95, 4, true, 113, 4, "", 0},
	{"pres3", 'o', 0x2001d312, 1, false, 1, 0, "", 0},
	{"c3class", 'o', 0x2001498f, 4, false, 1, 0, "", 0},
	{"c3race", 'o', 0x20014997, 4, true, 1, 0, "", 0},
	{"c3b01", 'o', 0x20015427, 4, false, 1, 0, "", 0},
	{"c3b11", 'o', 0x20015437, 4, false, 1, 0, "", 0},
	{"c3a2a9", 'o', 0x20014bcf, 4, false, 1, 0, "", 0},
	{"c3a4d", 'o', 0x20015373, 4, false, 1, 0, "", 0},
	{"c3att", 'o', 0x20014a0b, 4, false, 7, 20, "", 0},
	{"c3ab1", 'o', 0x200160a3, 1, true, 1, 0, "", 0},
	{"c3b8d", 'o', 0x200149b3, 4, false, 15, 4, "", 0},
	{"c3f", 'o', 0x20014ac3, 1, false, 41, 38, "", 0},
	{"c3bon7", 'o', 0x200160b0, 1, true, 1, 0, "", 0},
	{"c3ded", 'o', 0x20015783, 4, false, 6, 4, "", 0},
	{"c3spv", 'o', 0x200154f7, 4, true, 113, 4, "", 0},
	{"pres4", 'o', 0x2001d418, 1, false, 1, 0, "", 0},
	{"c4class", 'o', 0x200161f1, 4, false, 1, 0, "", 0},
	{"c4race", 'o', 0x200161f9, 4, true, 1, 0, "", 0},
	{"c4b01", 'o', 0x20016c89, 4, false, 1, 0, "", 0},
	{"c4b11", 'o', 0x20016c99, 4, false, 1, 0, "", 0},
	{"c4a2a9", 'o', 0x20016431, 4, false, 1, 0, "", 0},
	{"c4a4d", 'o', 0x20016bd5, 4, false, 1, 0, "", 0},
	{"c4att", 'o', 0x2001626d, 4, false, 7, 20, "", 0},
	{"c4ab1", 'o', 0x20017905, 1, true, 1, 0, "", 0},
	{"c4b8d", 'o', 0x20016215, 4, false, 15, 4, "", 0},
	{"c4f", 'o', 0x20016325, 1, false, 41, 38, "", 0},
	{"c4bon7", 'o', 0x20017912, 1, true, 1, 0, "", 0},
	{"c4ded", 'o', 0x20016fe5, 4, false, 6, 4, "", 0},
	{"c4spv", 'o', 0x20016d59, 4, true, 113, 4, "", 0},
	{"pres5", 'o', 0x2001d51e, 1, false, 1, 0, "", 0},
	{"c5class", 'o', 0x20017a53, 4, false, 1, 0, "", 0},
	{"c5race", 'o', 0x20017a5b, 4, true, 1, 0, "", 0},
	{"c5b01", 'o', 0x200184eb, 4, false, 1, 0, "", 0},
	{"c5b11", 'o', 0x200184fb, 4, false, 1, 0, "", 0},
	{"c5a2a9", 'o', 0x20017c93, 4, false, 1, 0, "", 0},
	{"c5a4d", 'o', 0x20018437, 4, false, 1, 0, "", 0},
	{"c5att", 'o', 0x20017acf, 4, false, 7, 20, "", 0},
	{"c5ab1", 'o', 0x20019167, 1, true, 1, 0, "", 0},
	{"c5b8d", 'o', 0x20017a77, 4, false, 15, 4, "", 0},
	{"c5f", 'o', 0x20017b87, 1, false, 41, 38, "", 0},
	{"c5bon7", 'o', 0x20019174, 1, true, 1, 0, "", 0},
	{"c5ded", 'o', 0x20018847, 4, false, 6, 4, "", 0},
	{"c5spv", 'o', 0x200185bb, 4, true, 113, 4, "", 0},
	{"pres6", 'o', 0x2001d624, 1, false, 1, 0, "", 0},
	{"c6class", 'o', 0x200192b5, 4, false, 1, 0, "", 0},
	{"c6race", 'o', 0x200192bd, 4, true, 1, 0, "", 0},
	{"c6b01", 'o', 0x20019d4d, 4, false, 1, 0, "", 0},
	{"c6b11", 'o', 0x20019d5d, 4, false, 1, 0, "", 0},
	{"c6a2a9", 'o', 0x200194f5, 4, false, 1, 0, "", 0},
	{"c6a4d", 'o', 0x20019c99, 4, false, 1, 0, "", 0},
	{"c6att", 'o', 0x20019331, 4, false, 7, 20, "", 0},
	{"c6ab1", 'o', 0x2001a9c9, 1, true, 1, 0, "", 0},
	{"c6b8d", 'o', 0x200192d9, 4, false, 15, 4, "", 0},
	{"c6f", 'o', 0x200193e9, 1, false, 41, 38, "", 0},
	{"c6bon7", 'o', 0x2001a9d6, 1, true, 1, 0, "", 0},
	{"c6ded", 'o', 0x2001a0a9, 4, false, 6, 4, "", 0},
	{"c6spv", 'o', 0x20019e1d, 4, true, 113, 4, "", 0},
	{"pres7", 'o', 0x2001d72a, 1, false, 1, 0, "", 0},
	{"c7class", 'o', 0x2001ab17, 4, false, 1, 0, "", 0},
	{"c7race", 'o', 0x2001ab1f, 4, true, 1, 0, "", 0},
	{"c7b01", 'o', 0x2001b5af, 4, false, 1, 0, "", 0},
	{"c7b11", 'o', 0x2001b5bf, 4, false, 1, 0, "", 0},
	{"c7a2a9", 'o', 0x2001ad57, 4, false, 1, 0, "", 0},
	{"c7a4d", 'o', 0x2001b4fb, 4, false, 1, 0, "", 0},
	{"c7att", 'o', 0x2001ab93, 4, false, 7, 20, "", 0},
	{"c7ab1", 'o', 0x2001c22b, 1, true, 1, 0, "", 0},
	{"c7b8d", 'o', 0x2001ab3b, 4, false, 15, 4, "", 0},
	{"c7f", 'o', 0x2001ac4b, 1, false, 41, 38, "", 0},
	{"c7bon7", 'o', 0x2001c238, 1, true, 1, 0, "", 0},
	{"c7ded", 'o', 0x2001b90b, 4, false, 6, 4, "", 0},
	{"c7spv", 'o', 0x2001b67f, 4, true, 113, 4, "", 0},
	{"pidx0", 'o', 0x2001d0fa, 4, true, 1, 0, "", 0},
	{"pidx1", 'o', 0x2001d200, 4, true, 1, 0, "", 0},
	{"sch", 'o', 0x200302f2, 4, false, 113, 447, "", 0},
	{"n0id", 'o', 0x2001ec58, 4, false, 1, 0, "", 0},
	{"n0d54", 'o', 0x2001ec54, 1, false, 1, 0, "", 0},
	{"n0d57", 'o', 0x2001ec57, 1, false, 1, 0, "", 0},
	{"n0w0", 'o', 0x2001e100, 4, false, 1, 0, "", 0},
	{"n0lst", 'o', 0x2001e10a, 4, false, 1, 0, "", 0},
	{"n0mid", 'o', 0x2001e116, 4, false, 1, 0, "", 0},
	{"n0b1a", 'o', 0x2001e11a, 1, false, 1, 0, "", 0},
	{"n0b1d", 'o', 0x2001e11d, 1, false, 1, 0, "", 0},
	{"n0b25", 'o', 0x2001e125, 1, false, 1, 0, "", 0},
	{"n0b26", 'o', 0x2001e126, 1, false, 1, 0, "", 0},
	{"n0b2b", 'o', 0x2001e12b, 1, true, 1, 0, "", 0},
	{"n0b2c", 'o', 0x2001e12c, 1, false, 1, 0, "", 0},
	{"n0b2e", 'o', 0x2001e12e, 1, false, 1, 0, "", 0},
	{"n0bc7", 'o', 0x2001e1c7, 1, false, 1, 0, "", 0},
	{"n0ws", 'o', 0x2001e130, 2, true, 40, 2, "", 0},
	{"k0cnt", 'o', 0x2001f008, 4, true, 1, 0, "", 0},
	{"k0d", 'o', 0x2001f010, 4, false, 6, 4, "", 0},
	{"k0e0id", 'o', 0x2001f044, 4, false, 1, 0, "", 0},
	{"k0e0q", 'o', 0x2001f048, 1, false, 1, 0, "", 0},
	{"k0e0c", 'o', 0x2001f050, 1, false, 1, 0, "", 0},
	{"k0e1id", 'o', 0x2001f064, 4, false, 1, 0, "", 0},
	{"k0e1q", 'o', 0x2001f068, 1, false, 1, 0, "", 0},
	{"k0e1c", 'o', 0x2001f070, 1, false, 1, 0, "", 0},
	{"k0e2id", 'o', 0x2001f084, 4, false, 1, 0, "", 0},
	{"k0e2q", 'o', 0x2001f088, 1, false, 1, 0, "", 0},
	{"k0e2c", 'o', 0x2001f090, 1, false, 1, 0, "", 0},
	{"k0e3id", 'o', 0x2001f0a4, 4, false, 1, 0, "", 0},
	{"k0e3q", 'o', 0x2001f0a8, 1, false, 1, 0, "", 0},
	{"k0e3c", 'o', 0x2001f0b0, 1, false, 1, 0, "", 0},
	{"k0e4id", 'o', 0x2001f0c4, 4, false, 1, 0, "", 0},
	{"k0e4q", 'o', 0x2001f0c8, 1, false, 1, 0, "", 0},
	{"k0e4c", 'o', 0x2001f0d0, 1, false, 1, 0, "", 0},
	{"k0e5id", 'o', 0x2001f0e4, 4, false, 1, 0, "", 0},
	{"k0e5q", 'o', 0x2001f0e8, 1, false, 1, 0, "", 0},
	{"k0e5c", 'o', 0x2001f0f0, 1, false, 1, 0, "", 0},
	{"n1id", 'o', 0x2001ecd8, 4, false, 1, 0, "", 0},
	{"n1d54", 'o', 0x2001ecd4, 1, false, 1, 0, "", 0},
	{"n1d57", 'o', 0x2001ecd7, 1, false, 1, 0, "", 0},
	{"n1w0", 'o', 0x2001e240, 4, false, 1, 0, "", 0},
	{"n1lst", 'o', 0x2001e24a, 4, false, 1, 0, "", 0},
	{"n1mid", 'o', 0x2001e256, 4, false, 1, 0, "", 0},
	{"n1b1a", 'o', 0x2001e25a, 1, false, 1, 0, "", 0},
	{"n1b1d", 'o', 0x2001e25d, 1, false, 1, 0, "", 0},
	{"n1b25", 'o', 0x2001e265, 1, false, 1, 0, "", 0},
	{"n1b26", 'o', 0x2001e266, 1, false, 1, 0, "", 0},
	{"n1b2b", 'o', 0x2001e26b, 1, true, 1, 0, "", 0},
	{"n1b2c", 'o', 0x2001e26c, 1, false, 1, 0, "", 0},
	{"n1b2e", 'o', 0x2001e26e, 1, false, 1, 0, "", 0},
	{"n1bc7", 'o', 0x2001e307, 1, false, 1, 0, "", 0},
	{"n1ws", 'o', 0x2001e270, 2, true, 40, 2, "", 0},
	{"k1cnt", 'o', 0x2001f108, 4, true, 1, 0, "", 0},
	{"k1d", 'o', 0x2001f110, 4, false, 6, 4, "", 0},
	{"k1e0id", 'o', 0x2001f144, 4, false, 1, 0, "", 0},
	{"k1e0q", 'o', 0x2001f148, 1, false, 1, 0, "", 0},
	{"k1e0c", 'o', 0x2001f150, 1, false, 1, 0, "", 0},
	{"k1e1id", 'o', 0x2001f164, 4, false, 1, 0, "", 0},
	{"k1e1q", 'o', 0x2001f168, 1, false, 1, 0, "", 0},
	{"k1e1c", 'o', 0x2001f170, 1, false, 1, 0, "", 0},
	{"k1e2id", 'o', 0x2001f184, 4, false, 1, 0, "", 0},
	{"k1e2q", 'o', 0x2001f188, 1, false, 1, 0, "", 0},
	{"k1e2c", 'o', 0x2001f190, 1, false, 1, 0, "", 0},
	{"k1e3id", 'o', 0x2001f1a4, 4, false, 1, 0, "", 0},
	{"k1e3q", 'o', 0x2001f1a8, 1, false, 1, 0, "", 0},
	{"k1e3c", 'o', 0x2001f1b0, 1, false, 1, 0, "", 0},
	{"k1e4id", 'o', 0x2001f1c4, 4, false, 1, 0, "", 0},
	{"k1e4q", 'o', 0x2001f1c8, 1, false, 1, 0, "", 0},
	{"k1e4c", 'o', 0x2001f1d0, 1, false, 1, 0, "", 0},
	{"k1e5id", 'o', 0x2001f1e4, 4, false, 1, 0, "", 0},
	{"k1e5q", 'o', 0x2001f1e8, 1, false, 1, 0, "", 0},
	{"k1e5c", 'o', 0x2001f1f0, 1, false, 1, 0, "", 0},
	{"n2id", 'o', 0x2001ed58, 4, false, 1, 0, "", 0},
	{"n2d54", 'o', 0x2001ed54, 1, false, 1, 0, "", 0},
	{"n2d57", 'o', 0x2001ed57, 1, false, 1, 0, "", 0},
	{"n2w0", 'o', 0x2001e380, 4, false, 1, 0, "", 0},
	{"n2lst", 'o', 0x2001e38a, 4, false, 1, 0, "", 0},
	{"n2mid", 'o', 0x2001e396, 4, false, 1, 0, "", 0},
	{"n2b1a", 'o', 0x2001e39a, 1, false, 1, 0, "", 0},
	{"n2b1d", 'o', 0x2001e39d, 1, false, 1, 0, "", 0},
	{"n2b25", 'o', 0x2001e3a5, 1, false, 1, 0, "", 0},
	{"n2b26", 'o', 0x2001e3a6, 1, false, 1, 0, "", 0},
	{"n2b2b", 'o', 0x2001e3ab, 1, true, 1, 0, "", 0},
	{"n2b2c", 'o', 0x2001e3ac, 1, false, 1, 0, "", 0},
	{"n2b2e", 'o', 0x2001e3ae, 1, false, 1, 0, "", 0},
	{"n2bc7", 'o', 0x2001e447, 1, false, 1, 0, "", 0},
	{"n2ws", 'o', 0x2001e3b0, 2, true, 40, 2, "", 0},
	{"k2cnt", 'o', 0x2001f208, 4, true, 1, 0, "", 0},
	{"k2d", 'o', 0x2001f210, 4, false, 6, 4, "", 0},
	{"k2e0id", 'o', 0x2001f244, 4, false, 1, 0, "", 0},
	{"k2e0q", 'o', 0x2001f248, 1, false, 1, 0, "", 0},
	{"k2e0c", 'o', 0x2001f250, 1, false, 1, 0, "", 0},
	{"k2e1id", 'o', 0x2001f264, 4, false, 1, 0, "", 0},
	{"k2e1q", 'o', 0x2001f268, 1, false, 1, 0, "", 0},
	{"k2e1c", 'o', 0x2001f270, 1, false, 1, 0, "", 0},
	{"k2e2id", 'o', 0x2001f284, 4, false, 1, 0, "", 0},
	{"k2e2q", 'o', 0x2001f288, 1, false, 1, 0, "", 0},
	{"k2e2c", 'o', 0x2001f290, 1, false, 1, 0, "", 0},
	{"k2e3id", 'o', 0x2001f2a4, 4, false, 1, 0, "", 0},
	{"k2e3q", 'o', 0x2001f2a8, 1, false, 1, 0, "", 0},
	{"k2e3c", 'o', 0x2001f2b0, 1, false, 1, 0, "", 0},
	{"k2e4id", 'o', 0x2001f2c4, 4, false, 1, 0, "", 0},
	{"k2e4q", 'o', 0x2001f2c8, 1, false, 1, 0, "", 0},
	{"k2e4c", 'o', 0x2001f2d0, 1, false, 1, 0, "", 0},
	{"k2e5id", 'o', 0x2001f2e4, 4, false, 1, 0, "", 0},
	{"k2e5q", 'o', 0x2001f2e8, 1, false, 1, 0, "", 0},
	{"k2e5c", 'o', 0x2001f2f0, 1, false, 1, 0, "", 0},
	{"n3id", 'o', 0x2001edd8, 4, false, 1, 0, "", 0},
	{"n3d54", 'o', 0x2001edd4, 1, false, 1, 0, "", 0},
	{"n3d57", 'o', 0x2001edd7, 1, false, 1, 0, "", 0},
	{"n3w0", 'o', 0x2001e4c0, 4, false, 1, 0, "", 0},
	{"n3lst", 'o', 0x2001e4ca, 4, false, 1, 0, "", 0},
	{"n3mid", 'o', 0x2001e4d6, 4, false, 1, 0, "", 0},
	{"n3b1a", 'o', 0x2001e4da, 1, false, 1, 0, "", 0},
	{"n3b1d", 'o', 0x2001e4dd, 1, false, 1, 0, "", 0},
	{"n3b25", 'o', 0x2001e4e5, 1, false, 1, 0, "", 0},
	{"n3b26", 'o', 0x2001e4e6, 1, false, 1, 0, "", 0},
	{"n3b2b", 'o', 0x2001e4eb, 1, true, 1, 0, "", 0},
	{"n3b2c", 'o', 0x2001e4ec, 1, false, 1, 0, "", 0},
	{"n3b2e", 'o', 0x2001e4ee, 1, false, 1, 0, "", 0},
	{"n3bc7", 'o', 0x2001e587, 1, false, 1, 0, "", 0},
	{"n3ws", 'o', 0x2001e4f0, 2, true, 40, 2, "", 0},
	{"k3cnt", 'o', 0x2001f308, 4, true, 1, 0, "", 0},
	{"k3d", 'o', 0x2001f310, 4, false, 6, 4, "", 0},
	{"k3e0id", 'o', 0x2001f344, 4, false, 1, 0, "", 0},
	{"k3e0q", 'o', 0x2001f348, 1, false, 1, 0, "", 0},
	{"k3e0c", 'o', 0x2001f350, 1, false, 1, 0, "", 0},
	{"k3e1id", 'o', 0x2001f364, 4, false, 1, 0, "", 0},
	{"k3e1q", 'o', 0x2001f368, 1, false, 1, 0, "", 0},
	{"k3e1c", 'o', 0x2001f370, 1, false, 1, 0, "", 0},
	{"k3e2id", 'o', 0x2001f384, 4, false, 1, 0, "", 0},
	{"k3e2q", 'o', 0x2001f388, 1, false, 1, 0, "", 0},
	{"k3e2c", 'o', 0x2001f390, 1, false, 1, 0, "", 0},
	{"k3e3id", 'o', 0x2001f3a4, 4, false, 1, 0, "", 0},
	{"k3e3q", 'o', 0x2001f3a8, 1, false, 1, 0, "", 0},
	{"k3e3c", 'o', 0x2001f3b0, 1, false, 1, 0, "", 0},
	{"k3e4id", 'o', 0x2001f3c4, 4, false, 1, 0, "", 0},
	{"k3e4q", 'o', 0x2001f3c8, 1, false, 1, 0, "", 0},
	{"k3e4c", 'o', 0x2001f3d0, 1, false, 1, 0, "", 0},
	{"k3e5id", 'o', 0x2001f3e4, 4, false, 1, 0, "", 0},
	{"k3e5q", 'o', 0x2001f3e8, 1, false, 1, 0, "", 0},
	{"k3e5c", 'o', 0x2001f3f0, 1, false, 1, 0, "", 0},
	{"n4id", 'o', 0x2001ee58, 4, false, 1, 0, "", 0},
	{"n4d54", 'o', 0x2001ee54, 1, false, 1, 0, "", 0},
	{"n4d57", 'o', 0x2001ee57, 1, false, 1, 0, "", 0},
	{"n4w0", 'o', 0x2001e600, 4, false, 1, 0, "", 0},
	{"n4lst", 'o', 0x2001e60a, 4, false, 1, 0, "", 0},
	{"n4mid", 'o', 0x2001e616, 4, false, 1, 0, "", 0},
	{"n4b1a", 'o', 0x2001e61a, 1, false, 1, 0, "", 0},
	{"n4b1d", 'o', 0x2001e61d, 1, false, 1, 0, "", 0},
	{"n4b25", 'o', 0x2001e625, 1, false, 1, 0, "", 0},
	{"n4b26", 'o', 0x2001e626, 1, false, 1, 0, "", 0},
	{"n4b2b", 'o', 0x2001e62b, 1, true, 1, 0, "", 0},
	{"n4b2c", 'o', 0x2001e62c, 1, false, 1, 0, "", 0},
	{"n4b2e", 'o', 0x2001e62e, 1, false, 1, 0, "", 0},
	{"n4bc7", 'o', 0x2001e6c7, 1, false, 1, 0, "", 0},
	{"n4ws", 'o', 0x2001e630, 2, true, 40, 2, "", 0},
	{"k4cnt", 'o', 0x2001f408, 4, true, 1, 0, "", 0},
	{"k4d", 'o', 0x2001f410, 4, false, 6, 4, "", 0},
	{"k4e0id", 'o', 0x2001f444, 4, false, 1, 0, "", 0},
	{"k4e0q", 'o', 0x2001f448, 1, false, 1, 0, "", 0},
	{"k4e0c", 'o', 0x2001f450, 1, false, 1, 0, "", 0},
	{"k4e1id", 'o', 0x2001f464, 4, false, 1, 0, "", 0},
	{"k4e1q", 'o', 0x2001f468, 1, false, 1, 0, "", 0},
	{"k4e1c", 'o', 0x2001f470, 1, false, 1, 0, "", 0},
	{"k4e2id", 'o', 0x2001f484, 4, false, 1, 0, "", 0},
	{"k4e2q", 'o', 0x2001f488, 1, false, 1, 0, "", 0},
	{"k4e2c", 'o', 0x2001f490, 1, false, 1, 0, "", 0},
	{"k4e3id", 'o', 0x2001f4a4, 4, false, 1, 0, "", 0},
	{"k4e3q", 'o', 0x2001f4a8, 1, false, 1, 0, "", 0},
	{"k4e3c", 'o', 0x2001f4b0, 1, false, 1, 0, "", 0},
	{"k4e4id", 'o', 0x2001f4c4, 4, false, 1, 0, "", 0},
	{"k4e4q", 'o', 0x2001f4c8, 1, false, 1, 0, "", 0},
	{"k4e4c", 'o', 0x2001f4d0, 1, false, 1, 0, "", 0},
	{"k4e5id", 'o', 0x2001f4e4, 4, false, 1, 0, "", 0},
	{"k4e5q", 'o', 0x2001f4e8, 1, false, 1, 0, "", 0},
	{"k4e5c", 'o', 0x2001f4f0, 1, false, 1, 0, "", 0},
	{"n5id", 'o', 0x2001eed8, 4, false, 1, 0, "", 0},
	{"n5d54", 'o', 0x2001eed4, 1, false, 1, 0, "", 0},
	{"n5d57", 'o', 0x2001eed7, 1, false, 1, 0, "", 0},
	{"n5w0", 'o', 0x2001e740, 4, false, 1, 0, "", 0},
	{"n5lst", 'o', 0x2001e74a, 4, false, 1, 0, "", 0},
	{"n5mid", 'o', 0x2001e756, 4, false, 1, 0, "", 0},
	{"n5b1a", 'o', 0x2001e75a, 1, false, 1, 0, "", 0},
	{"n5b1d", 'o', 0x2001e75d, 1, false, 1, 0, "", 0},
	{"n5b25", 'o', 0x2001e765, 1, false, 1, 0, "", 0},
	{"n5b26", 'o', 0x2001e766, 1, false, 1, 0, "", 0},
	{"n5b2b", 'o', 0x2001e76b, 1, true, 1, 0, "", 0},
	{"n5b2c", 'o', 0x2001e76c, 1, false, 1, 0, "", 0},
	{"n5b2e", 'o', 0x2001e76e, 1, false, 1, 0, "", 0},
	{"n5bc7", 'o', 0x2001e807, 1, false, 1, 0, "", 0},
	{"n5ws", 'o', 0x2001e770, 2, true, 40, 2, "", 0},
	{"k5cnt", 'o', 0x2001f508, 4, true, 1, 0, "", 0},
	{"k5d", 'o', 0x2001f510, 4, false, 6, 4, "", 0},
	{"k5e0id", 'o', 0x2001f544, 4, false, 1, 0, "", 0},
	{"k5e0q", 'o', 0x2001f548, 1, false, 1, 0, "", 0},
	{"k5e0c", 'o', 0x2001f550, 1, false, 1, 0, "", 0},
	{"k5e1id", 'o', 0x2001f564, 4, false, 1, 0, "", 0},
	{"k5e1q", 'o', 0x2001f568, 1, false, 1, 0, "", 0},
	{"k5e1c", 'o', 0x2001f570, 1, false, 1, 0, "", 0},
	{"k5e2id", 'o', 0x2001f584, 4, false, 1, 0, "", 0},
	{"k5e2q", 'o', 0x2001f588, 1, false, 1, 0, "", 0},
	{"k5e2c", 'o', 0x2001f590, 1, false, 1, 0, "", 0},
	{"k5e3id", 'o', 0x2001f5a4, 4, false, 1, 0, "", 0},
	{"k5e3q", 'o', 0x2001f5a8, 1, false, 1, 0, "", 0},
	{"k5e3c", 'o', 0x2001f5b0, 1, false, 1, 0, "", 0},
	{"k5e4id", 'o', 0x2001f5c4, 4, false, 1, 0, "", 0},
	{"k5e4q", 'o', 0x2001f5c8, 1, false, 1, 0, "", 0},
	{"k5e4c", 'o', 0x2001f5d0, 1, false, 1, 0, "", 0},
	{"k5e5id", 'o', 0x2001f5e4, 4, false, 1, 0, "", 0},
	{"k5e5q", 'o', 0x2001f5e8, 1, false, 1, 0, "", 0},
	{"k5e5c", 'o', 0x2001f5f0, 1, false, 1, 0, "", 0},
	{"n6id", 'o', 0x2001ef58, 4, false, 1, 0, "", 0},
	{"n6d54", 'o', 0x2001ef54, 1, false, 1, 0, "", 0},
	{"n6d57", 'o', 0x2001ef57, 1, false, 1, 0, "", 0},
	{"n6w0", 'o', 0x2001e880, 4, false, 1, 0, "", 0},
	{"n6lst", 'o', 0x2001e88a, 4, false, 1, 0, "", 0},
	{"n6mid", 'o', 0x2001e896, 4, false, 1, 0, "", 0},
	{"n6b1a", 'o', 0x2001e89a, 1, false, 1, 0, "", 0},
	{"n6b1d", 'o', 0x2001e89d, 1, false, 1, 0, "", 0},
	{"n6b25", 'o', 0x2001e8a5, 1, false, 1, 0, "", 0},
	{"n6b26", 'o', 0x2001e8a6, 1, false, 1, 0, "", 0},
	{"n6b2b", 'o', 0x2001e8ab, 1, true, 1, 0, "", 0},
	{"n6b2c", 'o', 0x2001e8ac, 1, false, 1, 0, "", 0},
	{"n6b2e", 'o', 0x2001e8ae, 1, false, 1, 0, "", 0},
	{"n6bc7", 'o', 0x2001e947, 1, false, 1, 0, "", 0},
	{"n6ws", 'o', 0x2001e8b0, 2, true, 40, 2, "", 0},
	{"k6cnt", 'o', 0x2001f608, 4, true, 1, 0, "", 0},
	{"k6d", 'o', 0x2001f610, 4, false, 6, 4, "", 0},
	{"k6e0id", 'o', 0x2001f644, 4, false, 1, 0, "", 0},
	{"k6e0q", 'o', 0x2001f648, 1, false, 1, 0, "", 0},
	{"k6e0c", 'o', 0x2001f650, 1, false, 1, 0, "", 0},
	{"k6e1id", 'o', 0x2001f664, 4, false, 1, 0, "", 0},
	{"k6e1q", 'o', 0x2001f668, 1, false, 1, 0, "", 0},
	{"k6e1c", 'o', 0x2001f670, 1, false, 1, 0, "", 0},
	{"k6e2id", 'o', 0x2001f684, 4, false, 1, 0, "", 0},
	{"k6e2q", 'o', 0x2001f688, 1, false, 1, 0, "", 0},
	{"k6e2c", 'o', 0x2001f690, 1, false, 1, 0, "", 0},
	{"k6e3id", 'o', 0x2001f6a4, 4, false, 1, 0, "", 0},
	{"k6e3q", 'o', 0x2001f6a8, 1, false, 1, 0, "", 0},
	{"k6e3c", 'o', 0x2001f6b0, 1, false, 1, 0, "", 0},
	{"k6e4id", 'o', 0x2001f6c4, 4, false, 1, 0, "", 0},
	{"k6e4q", 'o', 0x2001f6c8, 1, false, 1, 0, "", 0},
	{"k6e4c", 'o', 0x2001f6d0, 1, false, 1, 0, "", 0},
	{"k6e5id", 'o', 0x2001f6e4, 4, false, 1, 0, "", 0},
	{"k6e5q", 'o', 0x2001f6e8, 1, false, 1, 0, "", 0},
	{"k6e5c", 'o', 0x2001f6f0, 1, false, 1, 0, "", 0},
	{"m0id", 'o', 0x20020100, 4, false, 1, 0, "", 0},
	{"m1id", 'o', 0x20020140, 4, false, 1, 0, "", 0},
	{"m2id", 'o', 0x20020180, 4, false, 1, 0, "", 0},
	{"g0id", 'o', 0x20020900, 4, false, 1, 0, "", 0},
	{"g1id", 'o', 0x20020940, 4, false, 1, 0, "", 0},
	{"rs0", 'o', 0x200218a7, 1, false, 1, 0, "", 0},
	{"rs1", 'o', 0x200219a7, 1, false, 1, 0, "", 0},
	{"rs2", 'o', 0x20021aa7, 1, false, 1, 0, "", 0},
	{"rs3", 'o', 0x20021ba7, 1, false, 1, 0, "", 0},
	{"rs4", 'o', 0x20021ca7, 1, false, 1, 0, "", 0},
	{"r0id", 'o', 0x20022118, 4, false, 1, 0, "", 0},
	{"r0m", 'o', 0x2002219f, 4, false, 1, 0, "", 0},
	{"r1id", 'o', 0x20022218, 4, false, 1, 0, "", 0},
	{"r1m", 'o', 0x2002229f, 4, false, 1, 0, "", 0},
	{"r2id", 'o', 0x20022318, 4, false, 1, 0, "", 0},
	{"r2m", 'o', 0x2002239f, 4, false, 1, 0, "", 0},
	{"q0id", 'o', 0x20022918, 4, false, 1, 0, "", 0},
	{"q0m", 'o', 0x2002299f, 4, false, 1, 0, "", 0},
	{"q1id", 'o', 0x20022a18, 4, false, 1, 0, "", 0},
	{"q1m", 'o', 0x20022a9f, 4, false, 1, 0, "", 0},
	{"o0a0", 'o', 0x20026ca0, 4, false, 1, 0, "", 0},
	{"o1a0", 'o', 0x20026da0, 4, false, 1, 0, "", 0},
	{"o2a0", 'o', 0x20026ea0, 4, false, 1, 0, "", 0},
	{"e0", 'o', 0x20027000, 4, false, 1, 0, "", 0},
	{"icls28a", 'o', 0x20029000, 1, false, 1, 0, "", 0},
	{"ikind28a", 'o', 0x20029001, 1, false, 1, 0, "", 0},
	{"icls28b", 'o', 0x20029010, 1, false, 1, 0, "", 0},
	{"ikind28b", 'o', 0x20029011, 1, false, 1, 0, "", 0},
	{"icls28c", 'o', 0x20029020, 1, false, 1, 0, "", 0},
	{"ikind28c", 'o', 0x20029021, 1, false, 1, 0, "", 0},
	{"icls28d", 'o', 0x20029030, 1, false, 1, 0, "", 0},
	{"ikind28d", 'o', 0x20029031, 1, false, 1, 0, "", 0},
	{"icls28e", 'o', 0x20029040, 1, false, 1, 0, "", 0},
	{"ikind28e", 'o', 0x20029041, 1, false, 1, 0, "", 0},
	{"icls242", 'o', 0x20029050, 1, false, 1, 0, "", 0},
	{"ikind242", 'o', 0x20029051, 1, false, 1, 0, "", 0},
	{"icls1b0", 'o', 0x20029060, 1, false, 1, 0, "", 0},
	{"ikind1b0", 'o', 0x20029061, 1, false, 1, 0, "", 0},
	{"icls1f4", 'o', 0x20029070, 1, false, 1, 0, "", 0},
	{"ikind1f4", 'o', 0x20029071, 1, false, 1, 0, "", 0},
	{"icls1f5", 'o', 0x20029080, 1, false, 1, 0, "", 0},
	{"ikind1f5", 'o', 0x20029081, 1, false, 1, 0, "", 0},
	{"icls1f8", 'o', 0x20029090, 1, false, 1, 0, "", 0},
	{"ikind1f8", 'o', 0x20029091, 1, false, 1, 0, "", 0},
	{"s408860.ret", 'o', 0x20006000, 4, false, 1, 0, "", 0},
	{"s4512c0.ret", 'o', 0x20006004, 4, false, 1, 0, "", 0},
	{"s4f6c50.ret", 'o', 0x20006008, 4, false, 2, 4, "", 0},
	{"s509aa0.ret", 'o', 0x20006010, 4, false, 1, 0, "", 0},
	{"s451080.ret", 'o', 0x20006014, 4, false, 1, 0, "", 0},
	{"s4c2cf0.ret", 'o', 0x20006018, 4, false, 1, 0, "", 0},
	{"s506280.ret", 'o', 0x2000601c, 4, false, 4, 4, "", 0},
	{"s55a7b0.ret", 'o', 0x2000602c, 4, false, 1, 0, "", 0},
	{"find.ret", 'o', 0x20006030, 4, false, 3, 4, "", 0},
}

var sagOutG = []n4Grp{
	{"s408860.a1", 'o', 0x20000000, 4, false, 1, 0, "", 0},
	{"s408860.calls", 'o', 0x20000004, 4, false, 1, 0, "", 0},
	{"s4ae080.a1", 'o', 0x20000008, 4, false, 1, 0, "", 0},
	{"s4ae080.a2", 'o', 0x2000000c, 4, false, 1, 0, "", 0},
	{"s4ae080.a3", 'o', 0x20000010, 4, false, 1, 0, "", 0},
	{"s4ae080.a4", 'o', 0x20000014, 4, false, 1, 0, "", 0},
	{"s4ae080.a5", 'o', 0x20000018, 4, false, 1, 0, "", 0},
	{"s4ae080.calls", 'o', 0x2000001c, 4, false, 1, 0, "", 0},
	{"s4512c0.a1", 'o', 0x20000020, 4, false, 1, 0, "", 0},
	{"s4512c0.a3", 'o', 0x20000024, 4, false, 1, 0, "", 0},
	{"s4512c0.a4", 'o', 0x20000028, 4, false, 1, 0, "", 0},
	{"s4512c0.calls", 'o', 0x2000002c, 4, false, 1, 0, "", 0},
	{"s4f6c50.a1_0", 'o', 0x20000030, 4, false, 1, 0, "", 0},
	{"s4f6c50.a1_1", 'o', 0x20000034, 4, false, 1, 0, "", 0},
	{"s4f6c50.a3_0", 'o', 0x20000038, 4, false, 1, 0, "", 0},
	{"s4f6c50.a3_1", 'o', 0x2000003c, 4, false, 1, 0, "", 0},
	{"s4f6c50.a4_0", 'o', 0x20000040, 4, false, 1, 0, "", 0},
	{"s4f6c50.a4_1", 'o', 0x20000044, 4, false, 1, 0, "", 0},
	{"s4f6c50.calls", 'o', 0x20000048, 4, false, 1, 0, "", 0},
	{"s4f6cf0.a1_0", 'o', 0x2000004c, 4, false, 1, 0, "", 0},
	{"s4f6cf0.a1_1", 'o', 0x20000050, 4, false, 1, 0, "", 0},
	{"s4f6cf0.calls", 'o', 0x20000054, 4, false, 1, 0, "", 0},
	{"s4d8f00.a1", 'o', 0x20000058, 4, false, 1, 0, "", 0},
	{"s4d8f00.a2", 'o', 0x2000005c, 4, false, 1, 0, "", 0},
	{"s4d8f00.a4", 'o', 0x20000060, 4, false, 1, 0, "", 0},
	{"s4d8f00.a5", 'o', 0x20000064, 4, false, 1, 0, "", 0},
	{"s4d8f00.a6", 'o', 0x20000068, 4, false, 1, 0, "", 0},
	{"s4d8f00.a7", 'o', 0x2000006c, 4, false, 1, 0, "", 0},
	{"s4d8f00.calls", 'o', 0x20000070, 4, false, 1, 0, "", 0},
	{"s420f90.calls", 'o', 0x20000074, 4, false, 1, 0, "", 0},
	{"s547570.a1_0", 'o', 0x20000078, 4, false, 1, 0, "", 0},
	{"s547570.a1_1", 'o', 0x2000007c, 4, false, 1, 0, "", 0},
	{"s547570.a1_2", 'o', 0x20000080, 4, false, 1, 0, "", 0},
	{"s547570.a2_0", 'o', 0x20000084, 4, false, 1, 0, "", 0},
	{"s547570.a2_1", 'o', 0x20000088, 4, false, 1, 0, "", 0},
	{"s547570.a2_2", 'o', 0x2000008c, 4, false, 1, 0, "", 0},
	{"s547570.a3_0", 'o', 0x20000090, 4, false, 1, 0, "", 0},
	{"s547570.a3_1", 'o', 0x20000094, 4, false, 1, 0, "", 0},
	{"s547570.a3_2", 'o', 0x20000098, 4, false, 1, 0, "", 0},
	{"s547570.calls", 'o', 0x2000009c, 4, false, 1, 0, "", 0},
	{"s56e800.a1", 'o', 0x200000a0, 4, false, 1, 0, "", 0},
	{"s56e800.calls", 'o', 0x200000a4, 4, false, 1, 0, "", 0},
	{"s50c560.a1", 'o', 0x200000a8, 4, false, 1, 0, "", 0},
	{"s50c560.a2", 'o', 0x200000ac, 4, false, 1, 0, "", 0},
	{"s50c560.calls", 'o', 0x200000b0, 4, false, 1, 0, "", 0},
	{"s509aa0.a1", 'o', 0x200000b4, 4, false, 1, 0, "", 0},
	{"s509aa0.calls", 'o', 0x200000b8, 4, false, 1, 0, "", 0},
	{"s49acd0.a1_0", 'o', 0x200000bc, 4, false, 1, 0, "", 0},
	{"s49acd0.a1_1", 'o', 0x200000c0, 4, false, 1, 0, "", 0},
	{"s49acd0.calls", 'o', 0x200000c4, 4, false, 1, 0, "", 0},
	{"s498d90.a1", 'o', 0x200000c8, 4, false, 1, 0, "", 0},
	{"s498d90.calls", 'o', 0x200000cc, 4, false, 1, 0, "", 0},
	{"s5299b0.a1", 'o', 0x200000d0, 4, false, 1, 0, "", 0},
	{"s5299b0.calls", 'o', 0x200000d4, 4, false, 1, 0, "", 0},
	{"s58aad0.a1_0", 'o', 0x200000d8, 4, false, 1, 0, "", 0},
	{"s58aad0.a1_1", 'o', 0x200000dc, 4, false, 1, 0, "", 0},
	{"s58aad0.a2_0", 'o', 0x200000e0, 4, false, 1, 0, "", 0},
	{"s58aad0.a2_1", 'o', 0x200000e4, 4, false, 1, 0, "", 0},
	{"s58aad0.a3_0", 'o', 0x200000e8, 4, false, 1, 0, "", 0},
	{"s58aad0.a3_1", 'o', 0x200000ec, 4, false, 1, 0, "", 0},
	{"s58aad0.calls", 'o', 0x200000f0, 4, false, 1, 0, "", 0},
	{"s483fd0.a1", 'o', 0x200000f4, 4, false, 1, 0, "", 0},
	{"s483fd0.calls", 'o', 0x200000f8, 4, false, 1, 0, "", 0},
	{"s41f0d0.calls", 'o', 0x200000fc, 4, false, 1, 0, "", 0},
	{"s408ad0.a1", 'o', 0x20000100, 4, false, 1, 0, "", 0},
	{"s408ad0.a2", 'o', 0x20000104, 4, false, 1, 0, "", 0},
	{"s408ad0.calls", 'o', 0x20000108, 4, false, 1, 0, "", 0},
	{"s451080.a1", 'o', 0x2000010c, 4, false, 1, 0, "", 0},
	{"s451080.a2", 'o', 0x20000110, 4, false, 1, 0, "", 0},
	{"s451080.calls", 'o', 0x20000114, 4, false, 1, 0, "", 0},
	{"s5b1af0.a1_0", 'o', 0x20000118, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_1", 'o', 0x20000119, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_2", 'o', 0x2000011a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_3", 'o', 0x2000011b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_4", 'o', 0x2000011c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_5", 'o', 0x2000011d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_6", 'o', 0x2000011e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_7", 'o', 0x2000011f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_8", 'o', 0x20000120, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_9", 'o', 0x20000121, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_10", 'o', 0x20000122, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_11", 'o', 0x20000123, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_12", 'o', 0x20000124, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_13", 'o', 0x20000125, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_14", 'o', 0x20000126, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_15", 'o', 0x20000127, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_16", 'o', 0x20000128, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_17", 'o', 0x20000129, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_18", 'o', 0x2000012a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_19", 'o', 0x2000012b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_20", 'o', 0x2000012c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_21", 'o', 0x2000012d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_22", 'o', 0x2000012e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_23", 'o', 0x2000012f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_24", 'o', 0x20000130, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_25", 'o', 0x20000131, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_26", 'o', 0x20000132, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_27", 'o', 0x20000133, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_28", 'o', 0x20000134, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_29", 'o', 0x20000135, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_30", 'o', 0x20000136, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_31", 'o', 0x20000137, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_32", 'o', 0x20000138, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_33", 'o', 0x20000139, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_34", 'o', 0x2000013a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_35", 'o', 0x2000013b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_36", 'o', 0x2000013c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_37", 'o', 0x2000013d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_38", 'o', 0x2000013e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_39", 'o', 0x2000013f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_40", 'o', 0x20000140, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_41", 'o', 0x20000141, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_42", 'o', 0x20000142, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_43", 'o', 0x20000143, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_44", 'o', 0x20000144, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_45", 'o', 0x20000145, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_46", 'o', 0x20000146, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_47", 'o', 0x20000147, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_48", 'o', 0x20000148, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_49", 'o', 0x20000149, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_50", 'o', 0x2000014a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_51", 'o', 0x2000014b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_52", 'o', 0x2000014c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_53", 'o', 0x2000014d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_54", 'o', 0x2000014e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_55", 'o', 0x2000014f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_56", 'o', 0x20000150, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_57", 'o', 0x20000151, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_58", 'o', 0x20000152, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_59", 'o', 0x20000153, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_60", 'o', 0x20000154, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_61", 'o', 0x20000155, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_62", 'o', 0x20000156, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_63", 'o', 0x20000157, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_64", 'o', 0x20000158, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_65", 'o', 0x20000159, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_66", 'o', 0x2000015a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_67", 'o', 0x2000015b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_68", 'o', 0x2000015c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_69", 'o', 0x2000015d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_70", 'o', 0x2000015e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_71", 'o', 0x2000015f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_72", 'o', 0x20000160, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_73", 'o', 0x20000161, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_74", 'o', 0x20000162, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_75", 'o', 0x20000163, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_76", 'o', 0x20000164, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_77", 'o', 0x20000165, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_78", 'o', 0x20000166, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_79", 'o', 0x20000167, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_80", 'o', 0x20000168, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_81", 'o', 0x20000169, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_82", 'o', 0x2000016a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_83", 'o', 0x2000016b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_84", 'o', 0x2000016c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_85", 'o', 0x2000016d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_86", 'o', 0x2000016e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_87", 'o', 0x2000016f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_88", 'o', 0x20000170, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_89", 'o', 0x20000171, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_90", 'o', 0x20000172, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_91", 'o', 0x20000173, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_92", 'o', 0x20000174, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_93", 'o', 0x20000175, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_94", 'o', 0x20000176, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_95", 'o', 0x20000177, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_96", 'o', 0x20000178, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_97", 'o', 0x20000179, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_98", 'o', 0x2000017a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_99", 'o', 0x2000017b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_100", 'o', 0x2000017c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_101", 'o', 0x2000017d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_102", 'o', 0x2000017e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_103", 'o', 0x2000017f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_104", 'o', 0x20000180, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_105", 'o', 0x20000181, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_106", 'o', 0x20000182, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_107", 'o', 0x20000183, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_108", 'o', 0x20000184, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_109", 'o', 0x20000185, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_110", 'o', 0x20000186, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_111", 'o', 0x20000187, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_112", 'o', 0x20000188, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_113", 'o', 0x20000189, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_114", 'o', 0x2000018a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_115", 'o', 0x2000018b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_116", 'o', 0x2000018c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_117", 'o', 0x2000018d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_118", 'o', 0x2000018e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_119", 'o', 0x2000018f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_120", 'o', 0x20000190, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_121", 'o', 0x20000191, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_122", 'o', 0x20000192, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_123", 'o', 0x20000193, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_124", 'o', 0x20000194, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_125", 'o', 0x20000195, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_126", 'o', 0x20000196, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_127", 'o', 0x20000197, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_128", 'o', 0x20000198, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_129", 'o', 0x20000199, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_130", 'o', 0x2000019a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_131", 'o', 0x2000019b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_132", 'o', 0x2000019c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_133", 'o', 0x2000019d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_134", 'o', 0x2000019e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_135", 'o', 0x2000019f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_136", 'o', 0x200001a0, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_137", 'o', 0x200001a1, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_138", 'o', 0x200001a2, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_139", 'o', 0x200001a3, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_140", 'o', 0x200001a4, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_141", 'o', 0x200001a5, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_142", 'o', 0x200001a6, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_143", 'o', 0x200001a7, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_144", 'o', 0x200001a8, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_145", 'o', 0x200001a9, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_146", 'o', 0x200001aa, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_147", 'o', 0x200001ab, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_148", 'o', 0x200001ac, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_149", 'o', 0x200001ad, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_150", 'o', 0x200001ae, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_151", 'o', 0x200001af, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_152", 'o', 0x200001b0, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_153", 'o', 0x200001b1, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_154", 'o', 0x200001b2, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_155", 'o', 0x200001b3, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_156", 'o', 0x200001b4, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_157", 'o', 0x200001b5, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_158", 'o', 0x200001b6, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_159", 'o', 0x200001b7, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_160", 'o', 0x200001b8, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_161", 'o', 0x200001b9, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_162", 'o', 0x200001ba, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_163", 'o', 0x200001bb, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_164", 'o', 0x200001bc, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_165", 'o', 0x200001bd, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_166", 'o', 0x200001be, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_167", 'o', 0x200001bf, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_168", 'o', 0x200001c0, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_169", 'o', 0x200001c1, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_170", 'o', 0x200001c2, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_171", 'o', 0x200001c3, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_172", 'o', 0x200001c4, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_173", 'o', 0x200001c5, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_174", 'o', 0x200001c6, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_175", 'o', 0x200001c7, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_176", 'o', 0x200001c8, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_177", 'o', 0x200001c9, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_178", 'o', 0x200001ca, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_179", 'o', 0x200001cb, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_180", 'o', 0x200001cc, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_181", 'o', 0x200001cd, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_182", 'o', 0x200001ce, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_183", 'o', 0x200001cf, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_184", 'o', 0x200001d0, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_185", 'o', 0x200001d1, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_186", 'o', 0x200001d2, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_187", 'o', 0x200001d3, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_188", 'o', 0x200001d4, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_189", 'o', 0x200001d5, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_190", 'o', 0x200001d6, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_191", 'o', 0x200001d7, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_192", 'o', 0x200001d8, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_193", 'o', 0x200001d9, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_194", 'o', 0x200001da, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_195", 'o', 0x200001db, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_196", 'o', 0x200001dc, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_197", 'o', 0x200001dd, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_198", 'o', 0x200001de, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_199", 'o', 0x200001df, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_200", 'o', 0x200001e0, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_201", 'o', 0x200001e1, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_202", 'o', 0x200001e2, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_203", 'o', 0x200001e3, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_204", 'o', 0x200001e4, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_205", 'o', 0x200001e5, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_206", 'o', 0x200001e6, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_207", 'o', 0x200001e7, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_208", 'o', 0x200001e8, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_209", 'o', 0x200001e9, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_210", 'o', 0x200001ea, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_211", 'o', 0x200001eb, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_212", 'o', 0x200001ec, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_213", 'o', 0x200001ed, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_214", 'o', 0x200001ee, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_215", 'o', 0x200001ef, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_216", 'o', 0x200001f0, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_217", 'o', 0x200001f1, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_218", 'o', 0x200001f2, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_219", 'o', 0x200001f3, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_220", 'o', 0x200001f4, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_221", 'o', 0x200001f5, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_222", 'o', 0x200001f6, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_223", 'o', 0x200001f7, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_224", 'o', 0x200001f8, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_225", 'o', 0x200001f9, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_226", 'o', 0x200001fa, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_227", 'o', 0x200001fb, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_228", 'o', 0x200001fc, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_229", 'o', 0x200001fd, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_230", 'o', 0x200001fe, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_231", 'o', 0x200001ff, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_232", 'o', 0x20000200, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_233", 'o', 0x20000201, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_234", 'o', 0x20000202, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_235", 'o', 0x20000203, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_236", 'o', 0x20000204, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_237", 'o', 0x20000205, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_238", 'o', 0x20000206, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_239", 'o', 0x20000207, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_240", 'o', 0x20000208, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_241", 'o', 0x20000209, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_242", 'o', 0x2000020a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_243", 'o', 0x2000020b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_244", 'o', 0x2000020c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_245", 'o', 0x2000020d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_246", 'o', 0x2000020e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_247", 'o', 0x2000020f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_248", 'o', 0x20000210, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_249", 'o', 0x20000211, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_250", 'o', 0x20000212, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_251", 'o', 0x20000213, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_252", 'o', 0x20000214, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_253", 'o', 0x20000215, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_254", 'o', 0x20000216, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_255", 'o', 0x20000217, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_256", 'o', 0x20000218, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_257", 'o', 0x20000219, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_258", 'o', 0x2000021a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_259", 'o', 0x2000021b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_260", 'o', 0x2000021c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_261", 'o', 0x2000021d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_262", 'o', 0x2000021e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_263", 'o', 0x2000021f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_264", 'o', 0x20000220, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_265", 'o', 0x20000221, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_266", 'o', 0x20000222, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_267", 'o', 0x20000223, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_268", 'o', 0x20000224, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_269", 'o', 0x20000225, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_270", 'o', 0x20000226, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_271", 'o', 0x20000227, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_272", 'o', 0x20000228, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_273", 'o', 0x20000229, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_274", 'o', 0x2000022a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_275", 'o', 0x2000022b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_276", 'o', 0x2000022c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_277", 'o', 0x2000022d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_278", 'o', 0x2000022e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_279", 'o', 0x2000022f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_280", 'o', 0x20000230, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_281", 'o', 0x20000231, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_282", 'o', 0x20000232, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_283", 'o', 0x20000233, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_284", 'o', 0x20000234, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_285", 'o', 0x20000235, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_286", 'o', 0x20000236, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_287", 'o', 0x20000237, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_288", 'o', 0x20000238, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_289", 'o', 0x20000239, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_290", 'o', 0x2000023a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_291", 'o', 0x2000023b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_292", 'o', 0x2000023c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_293", 'o', 0x2000023d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_294", 'o', 0x2000023e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_295", 'o', 0x2000023f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_296", 'o', 0x20000240, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_297", 'o', 0x20000241, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_298", 'o', 0x20000242, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_299", 'o', 0x20000243, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_300", 'o', 0x20000244, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_301", 'o', 0x20000245, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_302", 'o', 0x20000246, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_303", 'o', 0x20000247, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_304", 'o', 0x20000248, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_305", 'o', 0x20000249, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_306", 'o', 0x2000024a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_307", 'o', 0x2000024b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_308", 'o', 0x2000024c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_309", 'o', 0x2000024d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_310", 'o', 0x2000024e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_311", 'o', 0x2000024f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_312", 'o', 0x20000250, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_313", 'o', 0x20000251, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_314", 'o', 0x20000252, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_315", 'o', 0x20000253, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_316", 'o', 0x20000254, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_317", 'o', 0x20000255, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_318", 'o', 0x20000256, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_319", 'o', 0x20000257, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_320", 'o', 0x20000258, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_321", 'o', 0x20000259, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_322", 'o', 0x2000025a, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_323", 'o', 0x2000025b, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_324", 'o', 0x2000025c, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_325", 'o', 0x2000025d, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_326", 'o', 0x2000025e, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_327", 'o', 0x2000025f, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_328", 'o', 0x20000260, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_329", 'o', 0x20000261, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_330", 'o', 0x20000262, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_331", 'o', 0x20000263, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_332", 'o', 0x20000264, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_333", 'o', 0x20000265, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_334", 'o', 0x20000266, 1, false, 1, 0, "", 0},
	{"s5b1af0.a1_335", 'o', 0x20000267, 1, false, 1, 0, "", 0},
	{"s5b1af0.calls", 'o', 0x20000268, 4, false, 1, 0, "", 0},
	{"s5b1b30.a1_0", 'o', 0x2000026c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_1", 'o', 0x2000026d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_2", 'o', 0x2000026e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_3", 'o', 0x2000026f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_4", 'o', 0x20000270, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_5", 'o', 0x20000271, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_6", 'o', 0x20000272, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_7", 'o', 0x20000273, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_8", 'o', 0x20000274, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_9", 'o', 0x20000275, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_10", 'o', 0x20000276, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_11", 'o', 0x20000277, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_12", 'o', 0x20000278, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_13", 'o', 0x20000279, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_14", 'o', 0x2000027a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_15", 'o', 0x2000027b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_16", 'o', 0x2000027c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_17", 'o', 0x2000027d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_18", 'o', 0x2000027e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_19", 'o', 0x2000027f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_20", 'o', 0x20000280, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_21", 'o', 0x20000281, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_22", 'o', 0x20000282, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_23", 'o', 0x20000283, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_24", 'o', 0x20000284, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_25", 'o', 0x20000285, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_26", 'o', 0x20000286, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_27", 'o', 0x20000287, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_28", 'o', 0x20000288, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_29", 'o', 0x20000289, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_30", 'o', 0x2000028a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_31", 'o', 0x2000028b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_32", 'o', 0x2000028c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_33", 'o', 0x2000028d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_34", 'o', 0x2000028e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_35", 'o', 0x2000028f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_36", 'o', 0x20000290, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_37", 'o', 0x20000291, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_38", 'o', 0x20000292, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_39", 'o', 0x20000293, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_40", 'o', 0x20000294, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_41", 'o', 0x20000295, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_42", 'o', 0x20000296, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_43", 'o', 0x20000297, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_44", 'o', 0x20000298, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_45", 'o', 0x20000299, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_46", 'o', 0x2000029a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_47", 'o', 0x2000029b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_48", 'o', 0x2000029c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_49", 'o', 0x2000029d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_50", 'o', 0x2000029e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_51", 'o', 0x2000029f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_52", 'o', 0x200002a0, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_53", 'o', 0x200002a1, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_54", 'o', 0x200002a2, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_55", 'o', 0x200002a3, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_56", 'o', 0x200002a4, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_57", 'o', 0x200002a5, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_58", 'o', 0x200002a6, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_59", 'o', 0x200002a7, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_60", 'o', 0x200002a8, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_61", 'o', 0x200002a9, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_62", 'o', 0x200002aa, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_63", 'o', 0x200002ab, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_64", 'o', 0x200002ac, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_65", 'o', 0x200002ad, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_66", 'o', 0x200002ae, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_67", 'o', 0x200002af, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_68", 'o', 0x200002b0, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_69", 'o', 0x200002b1, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_70", 'o', 0x200002b2, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_71", 'o', 0x200002b3, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_72", 'o', 0x200002b4, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_73", 'o', 0x200002b5, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_74", 'o', 0x200002b6, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_75", 'o', 0x200002b7, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_76", 'o', 0x200002b8, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_77", 'o', 0x200002b9, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_78", 'o', 0x200002ba, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_79", 'o', 0x200002bb, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_80", 'o', 0x200002bc, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_81", 'o', 0x200002bd, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_82", 'o', 0x200002be, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_83", 'o', 0x200002bf, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_84", 'o', 0x200002c0, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_85", 'o', 0x200002c1, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_86", 'o', 0x200002c2, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_87", 'o', 0x200002c3, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_88", 'o', 0x200002c4, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_89", 'o', 0x200002c5, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_90", 'o', 0x200002c6, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_91", 'o', 0x200002c7, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_92", 'o', 0x200002c8, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_93", 'o', 0x200002c9, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_94", 'o', 0x200002ca, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_95", 'o', 0x200002cb, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_96", 'o', 0x200002cc, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_97", 'o', 0x200002cd, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_98", 'o', 0x200002ce, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_99", 'o', 0x200002cf, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_100", 'o', 0x200002d0, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_101", 'o', 0x200002d1, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_102", 'o', 0x200002d2, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_103", 'o', 0x200002d3, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_104", 'o', 0x200002d4, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_105", 'o', 0x200002d5, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_106", 'o', 0x200002d6, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_107", 'o', 0x200002d7, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_108", 'o', 0x200002d8, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_109", 'o', 0x200002d9, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_110", 'o', 0x200002da, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_111", 'o', 0x200002db, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_112", 'o', 0x200002dc, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_113", 'o', 0x200002dd, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_114", 'o', 0x200002de, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_115", 'o', 0x200002df, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_116", 'o', 0x200002e0, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_117", 'o', 0x200002e1, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_118", 'o', 0x200002e2, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_119", 'o', 0x200002e3, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_120", 'o', 0x200002e4, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_121", 'o', 0x200002e5, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_122", 'o', 0x200002e6, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_123", 'o', 0x200002e7, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_124", 'o', 0x200002e8, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_125", 'o', 0x200002e9, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_126", 'o', 0x200002ea, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_127", 'o', 0x200002eb, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_128", 'o', 0x200002ec, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_129", 'o', 0x200002ed, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_130", 'o', 0x200002ee, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_131", 'o', 0x200002ef, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_132", 'o', 0x200002f0, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_133", 'o', 0x200002f1, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_134", 'o', 0x200002f2, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_135", 'o', 0x200002f3, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_136", 'o', 0x200002f4, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_137", 'o', 0x200002f5, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_138", 'o', 0x200002f6, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_139", 'o', 0x200002f7, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_140", 'o', 0x200002f8, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_141", 'o', 0x200002f9, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_142", 'o', 0x200002fa, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_143", 'o', 0x200002fb, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_144", 'o', 0x200002fc, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_145", 'o', 0x200002fd, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_146", 'o', 0x200002fe, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_147", 'o', 0x200002ff, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_148", 'o', 0x20000300, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_149", 'o', 0x20000301, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_150", 'o', 0x20000302, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_151", 'o', 0x20000303, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_152", 'o', 0x20000304, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_153", 'o', 0x20000305, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_154", 'o', 0x20000306, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_155", 'o', 0x20000307, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_156", 'o', 0x20000308, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_157", 'o', 0x20000309, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_158", 'o', 0x2000030a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_159", 'o', 0x2000030b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_160", 'o', 0x2000030c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_161", 'o', 0x2000030d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_162", 'o', 0x2000030e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_163", 'o', 0x2000030f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_164", 'o', 0x20000310, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_165", 'o', 0x20000311, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_166", 'o', 0x20000312, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_167", 'o', 0x20000313, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_168", 'o', 0x20000314, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_169", 'o', 0x20000315, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_170", 'o', 0x20000316, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_171", 'o', 0x20000317, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_172", 'o', 0x20000318, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_173", 'o', 0x20000319, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_174", 'o', 0x2000031a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_175", 'o', 0x2000031b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_176", 'o', 0x2000031c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_177", 'o', 0x2000031d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_178", 'o', 0x2000031e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_179", 'o', 0x2000031f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_180", 'o', 0x20000320, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_181", 'o', 0x20000321, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_182", 'o', 0x20000322, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_183", 'o', 0x20000323, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_184", 'o', 0x20000324, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_185", 'o', 0x20000325, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_186", 'o', 0x20000326, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_187", 'o', 0x20000327, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_188", 'o', 0x20000328, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_189", 'o', 0x20000329, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_190", 'o', 0x2000032a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_191", 'o', 0x2000032b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_192", 'o', 0x2000032c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_193", 'o', 0x2000032d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_194", 'o', 0x2000032e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_195", 'o', 0x2000032f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_196", 'o', 0x20000330, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_197", 'o', 0x20000331, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_198", 'o', 0x20000332, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_199", 'o', 0x20000333, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_200", 'o', 0x20000334, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_201", 'o', 0x20000335, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_202", 'o', 0x20000336, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_203", 'o', 0x20000337, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_204", 'o', 0x20000338, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_205", 'o', 0x20000339, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_206", 'o', 0x2000033a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_207", 'o', 0x2000033b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_208", 'o', 0x2000033c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_209", 'o', 0x2000033d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_210", 'o', 0x2000033e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_211", 'o', 0x2000033f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_212", 'o', 0x20000340, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_213", 'o', 0x20000341, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_214", 'o', 0x20000342, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_215", 'o', 0x20000343, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_216", 'o', 0x20000344, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_217", 'o', 0x20000345, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_218", 'o', 0x20000346, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_219", 'o', 0x20000347, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_220", 'o', 0x20000348, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_221", 'o', 0x20000349, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_222", 'o', 0x2000034a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_223", 'o', 0x2000034b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_224", 'o', 0x2000034c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_225", 'o', 0x2000034d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_226", 'o', 0x2000034e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_227", 'o', 0x2000034f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_228", 'o', 0x20000350, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_229", 'o', 0x20000351, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_230", 'o', 0x20000352, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_231", 'o', 0x20000353, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_232", 'o', 0x20000354, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_233", 'o', 0x20000355, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_234", 'o', 0x20000356, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_235", 'o', 0x20000357, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_236", 'o', 0x20000358, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_237", 'o', 0x20000359, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_238", 'o', 0x2000035a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_239", 'o', 0x2000035b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_240", 'o', 0x2000035c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_241", 'o', 0x2000035d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_242", 'o', 0x2000035e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_243", 'o', 0x2000035f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_244", 'o', 0x20000360, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_245", 'o', 0x20000361, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_246", 'o', 0x20000362, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_247", 'o', 0x20000363, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_248", 'o', 0x20000364, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_249", 'o', 0x20000365, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_250", 'o', 0x20000366, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_251", 'o', 0x20000367, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_252", 'o', 0x20000368, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_253", 'o', 0x20000369, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_254", 'o', 0x2000036a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_255", 'o', 0x2000036b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_256", 'o', 0x2000036c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_257", 'o', 0x2000036d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_258", 'o', 0x2000036e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_259", 'o', 0x2000036f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_260", 'o', 0x20000370, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_261", 'o', 0x20000371, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_262", 'o', 0x20000372, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_263", 'o', 0x20000373, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_264", 'o', 0x20000374, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_265", 'o', 0x20000375, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_266", 'o', 0x20000376, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_267", 'o', 0x20000377, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_268", 'o', 0x20000378, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_269", 'o', 0x20000379, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_270", 'o', 0x2000037a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_271", 'o', 0x2000037b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_272", 'o', 0x2000037c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_273", 'o', 0x2000037d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_274", 'o', 0x2000037e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_275", 'o', 0x2000037f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_276", 'o', 0x20000380, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_277", 'o', 0x20000381, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_278", 'o', 0x20000382, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_279", 'o', 0x20000383, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_280", 'o', 0x20000384, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_281", 'o', 0x20000385, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_282", 'o', 0x20000386, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_283", 'o', 0x20000387, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_284", 'o', 0x20000388, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_285", 'o', 0x20000389, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_286", 'o', 0x2000038a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_287", 'o', 0x2000038b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_288", 'o', 0x2000038c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_289", 'o', 0x2000038d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_290", 'o', 0x2000038e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_291", 'o', 0x2000038f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_292", 'o', 0x20000390, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_293", 'o', 0x20000391, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_294", 'o', 0x20000392, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_295", 'o', 0x20000393, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_296", 'o', 0x20000394, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_297", 'o', 0x20000395, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_298", 'o', 0x20000396, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_299", 'o', 0x20000397, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_300", 'o', 0x20000398, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_301", 'o', 0x20000399, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_302", 'o', 0x2000039a, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_303", 'o', 0x2000039b, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_304", 'o', 0x2000039c, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_305", 'o', 0x2000039d, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_306", 'o', 0x2000039e, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_307", 'o', 0x2000039f, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_308", 'o', 0x200003a0, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_309", 'o', 0x200003a1, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_310", 'o', 0x200003a2, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_311", 'o', 0x200003a3, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_312", 'o', 0x200003a4, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_313", 'o', 0x200003a5, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_314", 'o', 0x200003a6, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_315", 'o', 0x200003a7, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_316", 'o', 0x200003a8, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_317", 'o', 0x200003a9, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_318", 'o', 0x200003aa, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_319", 'o', 0x200003ab, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_320", 'o', 0x200003ac, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_321", 'o', 0x200003ad, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_322", 'o', 0x200003ae, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_323", 'o', 0x200003af, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_324", 'o', 0x200003b0, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_325", 'o', 0x200003b1, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_326", 'o', 0x200003b2, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_327", 'o', 0x200003b3, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_328", 'o', 0x200003b4, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_329", 'o', 0x200003b5, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_330", 'o', 0x200003b6, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_331", 'o', 0x200003b7, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_332", 'o', 0x200003b8, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_333", 'o', 0x200003b9, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_334", 'o', 0x200003ba, 1, false, 1, 0, "", 0},
	{"s5b1b30.a1_335", 'o', 0x200003bb, 1, false, 1, 0, "", 0},
	{"s5b1b30.calls", 'o', 0x200003bc, 4, false, 1, 0, "", 0},
	{"s569a50.a1_0", 'o', 0x200003c0, 4, false, 1, 0, "", 0},
	{"s569a50.a1_1", 'o', 0x200003c4, 4, false, 1, 0, "", 0},
	{"s569a50.a1_2", 'o', 0x200003c8, 4, false, 1, 0, "", 0},
	{"s569a50.a1_3", 'o', 0x200003cc, 4, false, 1, 0, "", 0},
	{"s569a50.a1_4", 'o', 0x200003d0, 4, false, 1, 0, "", 0},
	{"s569a50.a1_5", 'o', 0x200003d4, 4, false, 1, 0, "", 0},
	{"s569a50.a1_6", 'o', 0x200003d8, 4, false, 1, 0, "", 0},
	{"s569a50.a1_7", 'o', 0x200003dc, 4, false, 1, 0, "", 0},
	{"s569a50.a2_0", 'o', 0x200003e0, 4, false, 1, 0, "", 0},
	{"s569a50.a2_1", 'o', 0x200003e4, 4, false, 1, 0, "", 0},
	{"s569a50.a2_2", 'o', 0x200003e8, 4, false, 1, 0, "", 0},
	{"s569a50.a2_3", 'o', 0x200003ec, 4, false, 1, 0, "", 0},
	{"s569a50.a2_4", 'o', 0x200003f0, 4, false, 1, 0, "", 0},
	{"s569a50.a2_5", 'o', 0x200003f4, 4, false, 1, 0, "", 0},
	{"s569a50.a2_6", 'o', 0x200003f8, 4, false, 1, 0, "", 0},
	{"s569a50.a2_7", 'o', 0x200003fc, 4, false, 1, 0, "", 0},
	{"s569a50.a3_0", 'o', 0x20000400, 4, false, 1, 0, "", 0},
	{"s569a50.a3_1", 'o', 0x20000404, 4, false, 1, 0, "", 0},
	{"s569a50.a3_2", 'o', 0x20000408, 4, false, 1, 0, "", 0},
	{"s569a50.a3_3", 'o', 0x2000040c, 4, false, 1, 0, "", 0},
	{"s569a50.a3_4", 'o', 0x20000410, 4, false, 1, 0, "", 0},
	{"s569a50.a3_5", 'o', 0x20000414, 4, false, 1, 0, "", 0},
	{"s569a50.a3_6", 'o', 0x20000418, 4, false, 1, 0, "", 0},
	{"s569a50.a3_7", 'o', 0x2000041c, 4, false, 1, 0, "", 0},
	{"s569a50.a4_0", 'o', 0x20000420, 4, false, 1, 0, "", 0},
	{"s569a50.a4_1", 'o', 0x20000424, 4, false, 1, 0, "", 0},
	{"s569a50.a4_2", 'o', 0x20000428, 4, false, 1, 0, "", 0},
	{"s569a50.a4_3", 'o', 0x2000042c, 4, false, 1, 0, "", 0},
	{"s569a50.a4_4", 'o', 0x20000430, 4, false, 1, 0, "", 0},
	{"s569a50.a4_5", 'o', 0x20000434, 4, false, 1, 0, "", 0},
	{"s569a50.a4_6", 'o', 0x20000438, 4, false, 1, 0, "", 0},
	{"s569a50.a4_7", 'o', 0x2000043c, 4, false, 1, 0, "", 0},
	{"s569a50.calls", 'o', 0x20000440, 4, false, 1, 0, "", 0},
	{"s56aab0.calls", 'o', 0x20000444, 4, false, 1, 0, "", 0},
	{"s55ee70.a1_0", 'o', 0x20000448, 4, false, 1, 0, "", 0},
	{"s55ee70.a1_1", 'o', 0x2000044c, 4, false, 1, 0, "", 0},
	{"s55ee70.calls", 'o', 0x20000450, 4, false, 1, 0, "", 0},
	{"s4add30.a1", 'o', 0x20000454, 4, false, 1, 0, "", 0},
	{"s4add30.calls", 'o', 0x20000458, 4, false, 1, 0, "", 0},
	{"s4f21e0.a1_0", 'o', 0x2000045c, 4, false, 1, 0, "", 0},
	{"s4f21e0.a1_1", 'o', 0x20000460, 4, false, 1, 0, "", 0},
	{"s4f21e0.a1_2", 'o', 0x20000464, 4, false, 1, 0, "", 0},
	{"s4f21e0.a1_3", 'o', 0x20000468, 4, false, 1, 0, "", 0},
	{"s4f21e0.a1_4", 'o', 0x2000046c, 4, false, 1, 0, "", 0},
	{"s4f21e0.a1_5", 'o', 0x20000470, 4, false, 1, 0, "", 0},
	{"s4f21e0.a1_6", 'o', 0x20000474, 4, false, 1, 0, "", 0},
	{"s4f21e0.a1_7", 'o', 0x20000478, 4, false, 1, 0, "", 0},
	{"s4f21e0.calls", 'o', 0x2000047c, 4, false, 1, 0, "", 0},
	{"s4f2380.a1_0", 'o', 0x20000480, 4, false, 1, 0, "", 0},
	{"s4f2380.a1_1", 'o', 0x20000484, 4, false, 1, 0, "", 0},
	{"s4f2380.a1_2", 'o', 0x20000488, 4, false, 1, 0, "", 0},
	{"s4f2380.a1_3", 'o', 0x2000048c, 4, false, 1, 0, "", 0},
	{"s4f2380.a1_4", 'o', 0x20000490, 4, false, 1, 0, "", 0},
	{"s4f2380.a1_5", 'o', 0x20000494, 4, false, 1, 0, "", 0},
	{"s4f2380.a1_6", 'o', 0x20000498, 4, false, 1, 0, "", 0},
	{"s4f2380.a1_7", 'o', 0x2000049c, 4, false, 1, 0, "", 0},
	{"s4f2380.calls", 'o', 0x200004a0, 4, false, 1, 0, "", 0},
	{"s43d940.a1_0", 'o', 0x200004a4, 4, false, 1, 0, "", 0},
	{"s43d940.a1_1", 'o', 0x200004a8, 4, false, 1, 0, "", 0},
	{"s43d940.a1_2", 'o', 0x200004ac, 4, false, 1, 0, "", 0},
	{"s43d940.calls", 'o', 0x200004b0, 4, false, 1, 0, "", 0},
	{"s4c7f10.a1", 'o', 0x200004b4, 4, false, 1, 0, "", 0},
	{"s4c7f10.a2", 'o', 0x200004b8, 4, false, 1, 0, "", 0},
	{"s4c7f10.calls", 'o', 0x200004bc, 4, false, 1, 0, "", 0},
	{"s4c2cf0.a1", 'o', 0x200004c0, 1, false, 1, 0, "", 0},
	{"s4c2cf0.calls", 'o', 0x200004c4, 4, false, 1, 0, "", 0},
	{"s4e4db0.a1", 'o', 0x200004c8, 4, false, 1, 0, "", 0},
	{"s4e4db0.a2", 'o', 0x200004cc, 4, false, 1, 0, "", 0},
	{"s4e4db0.a3", 'o', 0x200004d0, 4, false, 1, 0, "", 0},
	{"s4e4db0.calls", 'o', 0x200004d4, 4, false, 1, 0, "", 0},
	{"s4e3af0.a1", 'o', 0x200004d8, 4, false, 1, 0, "", 0},
	{"s4e3af0.a2", 'o', 0x200004dc, 4, false, 1, 0, "", 0},
	{"s4e3af0.calls", 'o', 0x200004e0, 4, false, 1, 0, "", 0},
	{"s55a0a0.a1", 'o', 0x200004e4, 4, false, 1, 0, "", 0},
	{"s55a0a0.calls", 'o', 0x200004e8, 4, false, 1, 0, "", 0},
	{"s524ca0.a1", 'o', 0x200004ec, 4, false, 1, 0, "", 0},
	{"s524ca0.calls", 'o', 0x200004f0, 4, false, 1, 0, "", 0},
	{"s52e690.a1", 'o', 0x200004f4, 4, false, 1, 0, "", 0},
	{"s52e690.a2", 'o', 0x200004f8, 4, false, 1, 0, "", 0},
	{"s52e690.a3", 'o', 0x200004fc, 4, false, 1, 0, "", 0},
	{"s52e690.a4", 'o', 0x20000500, 4, false, 1, 0, "", 0},
	{"s52e690.a5", 'o', 0x20000504, 4, false, 1, 0, "", 0},
	{"s52e690.calls", 'o', 0x20000508, 4, false, 1, 0, "", 0},
	{"s506280.a1_0", 'o', 0x2000050c, 4, false, 1, 0, "", 0},
	{"s506280.a1_1", 'o', 0x20000510, 4, false, 1, 0, "", 0},
	{"s506280.a1_2", 'o', 0x20000514, 4, false, 1, 0, "", 0},
	{"s506280.a1_3", 'o', 0x20000518, 4, false, 1, 0, "", 0},
	{"s506280.calls", 'o', 0x2000051c, 4, false, 1, 0, "", 0},
	{"s523330.a1", 'o', 0x20000520, 4, false, 1, 0, "", 0},
	{"s523330.a2", 'o', 0x20000524, 4, false, 1, 0, "", 0},
	{"s523330.a3", 'o', 0x20000528, 4, false, 1, 0, "", 0},
	{"s523330.calls", 'o', 0x2000052c, 4, false, 1, 0, "", 0},
	{"s51d960.a1_0", 'o', 0x20000530, 4, false, 1, 0, "", 0},
	{"s51d960.a1_1", 'o', 0x20000534, 4, false, 1, 0, "", 0},
	{"s51d960.a1_2", 'o', 0x20000538, 4, false, 1, 0, "", 0},
	{"s51d960.a1_3", 'o', 0x2000053c, 4, false, 1, 0, "", 0},
	{"s51d960.a1_4", 'o', 0x20000540, 4, false, 1, 0, "", 0},
	{"s51d960.a1_5", 'o', 0x20000544, 4, false, 1, 0, "", 0},
	{"s51d960.a1_6", 'o', 0x20000548, 4, false, 1, 0, "", 0},
	{"s51d960.a1_7", 'o', 0x2000054c, 4, false, 1, 0, "", 0},
	{"s51d960.calls", 'o', 0x20000550, 4, false, 1, 0, "", 0},
	{"s4ed9d0.a1_0", 'o', 0x20000554, 4, false, 1, 0, "", 0},
	{"s4ed9d0.a1_1", 'o', 0x20000558, 4, false, 1, 0, "", 0},
	{"s4ed9d0.a1_2", 'o', 0x2000055c, 4, false, 1, 0, "", 0},
	{"s4ed9d0.a1_3", 'o', 0x20000560, 4, false, 1, 0, "", 0},
	{"s4ed9d0.a1_4", 'o', 0x20000564, 4, false, 1, 0, "", 0},
	{"s4ed9d0.a1_5", 'o', 0x20000568, 4, false, 1, 0, "", 0},
	{"s4ed9d0.a1_6", 'o', 0x2000056c, 4, false, 1, 0, "", 0},
	{"s4ed9d0.a1_7", 'o', 0x20000570, 4, false, 1, 0, "", 0},
	{"s4ed9d0.calls", 'o', 0x20000574, 4, false, 1, 0, "", 0},
	{"s522fe0.a1_0", 'o', 0x20000578, 4, false, 1, 0, "", 0},
	{"s522fe0.a1_1", 'o', 0x2000057c, 4, false, 1, 0, "", 0},
	{"s522fe0.a1_2", 'o', 0x20000580, 4, false, 1, 0, "", 0},
	{"s522fe0.a1_3", 'o', 0x20000584, 4, false, 1, 0, "", 0},
	{"s522fe0.a1_4", 'o', 0x20000588, 4, false, 1, 0, "", 0},
	{"s522fe0.a1_5", 'o', 0x2000058c, 4, false, 1, 0, "", 0},
	{"s522fe0.a1_6", 'o', 0x20000590, 4, false, 1, 0, "", 0},
	{"s522fe0.a1_7", 'o', 0x20000594, 4, false, 1, 0, "", 0},
	{"s522fe0.a2_0", 'o', 0x20000598, 4, false, 1, 0, "", 0},
	{"s522fe0.a2_1", 'o', 0x2000059c, 4, false, 1, 0, "", 0},
	{"s522fe0.a2_2", 'o', 0x200005a0, 4, false, 1, 0, "", 0},
	{"s522fe0.a2_3", 'o', 0x200005a4, 4, false, 1, 0, "", 0},
	{"s522fe0.a2_4", 'o', 0x200005a8, 4, false, 1, 0, "", 0},
	{"s522fe0.a2_5", 'o', 0x200005ac, 4, false, 1, 0, "", 0},
	{"s522fe0.a2_6", 'o', 0x200005b0, 4, false, 1, 0, "", 0},
	{"s522fe0.a2_7", 'o', 0x200005b4, 4, false, 1, 0, "", 0},
	{"s522fe0.a3_0", 'o', 0x200005b8, 4, false, 1, 0, "", 0},
	{"s522fe0.a3_1", 'o', 0x200005bc, 4, false, 1, 0, "", 0},
	{"s522fe0.a3_2", 'o', 0x200005c0, 4, false, 1, 0, "", 0},
	{"s522fe0.a3_3", 'o', 0x200005c4, 4, false, 1, 0, "", 0},
	{"s522fe0.a3_4", 'o', 0x200005c8, 4, false, 1, 0, "", 0},
	{"s522fe0.a3_5", 'o', 0x200005cc, 4, false, 1, 0, "", 0},
	{"s522fe0.a3_6", 'o', 0x200005d0, 4, false, 1, 0, "", 0},
	{"s522fe0.a3_7", 'o', 0x200005d4, 4, false, 1, 0, "", 0},
	{"s522fe0.a4_0", 'o', 0x200005d8, 4, false, 1, 0, "", 0},
	{"s522fe0.a4_1", 'o', 0x200005dc, 4, false, 1, 0, "", 0},
	{"s522fe0.a4_2", 'o', 0x200005e0, 4, false, 1, 0, "", 0},
	{"s522fe0.a4_3", 'o', 0x200005e4, 4, false, 1, 0, "", 0},
	{"s522fe0.a4_4", 'o', 0x200005e8, 4, false, 1, 0, "", 0},
	{"s522fe0.a4_5", 'o', 0x200005ec, 4, false, 1, 0, "", 0},
	{"s522fe0.a4_6", 'o', 0x200005f0, 4, false, 1, 0, "", 0},
	{"s522fe0.a4_7", 'o', 0x200005f4, 4, false, 1, 0, "", 0},
	{"s522fe0.a5_0", 'o', 0x200005f8, 4, false, 1, 0, "", 0},
	{"s522fe0.a5_1", 'o', 0x200005fc, 4, false, 1, 0, "", 0},
	{"s522fe0.a5_2", 'o', 0x20000600, 4, false, 1, 0, "", 0},
	{"s522fe0.a5_3", 'o', 0x20000604, 4, false, 1, 0, "", 0},
	{"s522fe0.a5_4", 'o', 0x20000608, 4, false, 1, 0, "", 0},
	{"s522fe0.a5_5", 'o', 0x2000060c, 4, false, 1, 0, "", 0},
	{"s522fe0.a5_6", 'o', 0x20000610, 4, false, 1, 0, "", 0},
	{"s522fe0.a5_7", 'o', 0x20000614, 4, false, 1, 0, "", 0},
	{"s522fe0.a6_0", 'o', 0x20000618, 4, false, 1, 0, "", 0},
	{"s522fe0.a6_1", 'o', 0x2000061c, 4, false, 1, 0, "", 0},
	{"s522fe0.a6_2", 'o', 0x20000620, 4, false, 1, 0, "", 0},
	{"s522fe0.a6_3", 'o', 0x20000624, 4, false, 1, 0, "", 0},
	{"s522fe0.a6_4", 'o', 0x20000628, 4, false, 1, 0, "", 0},
	{"s522fe0.a6_5", 'o', 0x2000062c, 4, false, 1, 0, "", 0},
	{"s522fe0.a6_6", 'o', 0x20000630, 4, false, 1, 0, "", 0},
	{"s522fe0.a6_7", 'o', 0x20000634, 4, false, 1, 0, "", 0},
	{"s522fe0.calls", 'o', 0x20000638, 4, false, 1, 0, "", 0},
	{"s55a7b0.a1", 'o', 0x2000063c, 4, false, 1, 0, "", 0},
	{"s55a7b0.a2", 'o', 0x20000640, 4, false, 1, 0, "", 0},
	{"s55a7b0.a3", 'o', 0x20000644, 4, false, 1, 0, "", 0},
	{"s55a7b0.calls", 'o', 0x20000648, 4, false, 1, 0, "", 0},
	{"find.a2_0", 'o', 0x2000064c, 4, false, 1, 0, "", 0},
	{"find.a2_1", 'o', 0x20000650, 4, false, 1, 0, "", 0},
	{"find.a2_2", 'o', 0x20000654, 4, false, 1, 0, "", 0},
	{"find.calls", 'o', 0x20000658, 4, false, 1, 0, "", 0},
	{"heap", 'o', 0x50000000, 4, false, 256, 4, "", 0},
	{"evcnt", 'o', 0x20026404, 4, false, 1, 0, "", 0},
	{"evcap", 'o', 0x20026408, 4, false, 1, 0, "", 0},
	{"evbuf", 'o', 0x2002640c, 4, false, 1, 0, "", 0},
	{"evb", 'o', 0x20026420, 4, false, 12, 4, "", 0},
	{"qcnt", 'g', 0x68c4c0, 4, false, 1, 0, "", 0},
	{"qcap", 'g', 0x68c4c4, 4, false, 1, 0, "", 0},
	{"qbuf", 'g', 0x68c4c8, 4, false, 1, 0, "", 0},
	{"qb", 'o', 0x20026500, 4, false, 4, 4, "", 0},
	{"ncnt", 'g', 0x6598fc, 4, false, 1, 0, "", 0},
	{"ncap", 'g', 0x659900, 4, false, 1, 0, "", 0},
	{"nbuf", 'g', 0x659904, 4, false, 1, 0, "", 0},
	{"vcnt", 'g', 0x659994, 4, false, 1, 0, "", 0},
	{"vcap", 'g', 0x659998, 4, false, 1, 0, "", 0},
	{"vbuf", 'g', 0x65999c, 4, false, 1, 0, "", 0},
	{"ocnt", 'g', 0x6598e4, 4, false, 1, 0, "", 0},
	{"ocap", 'g', 0x6598e8, 4, false, 1, 0, "", 0},
	{"obuf", 'g', 0x6598ec, 4, false, 1, 0, "", 0},
	{"nb", 'o', 0x20026600, 4, false, 4, 4, "", 0},
	{"vb", 'o', 0x20026640, 4, false, 4, 4, "", 0},
	{"ob", 'o', 0x20026680, 4, false, 4, 4, "", 0},
	{"b68c4f4", 'g', 0x68c4f4, 1, false, 1, 0, "", 0},
	{"b68c4f7", 'g', 0x68c4f7, 1, false, 1, 0, "", 0},
	{"b68506f", 'g', 0x68506f, 1, false, 1, 0, "", 0},
	{"b6875a5", 'g', 0x6875a5, 1, false, 1, 0, "", 0},
	{"b68c4f9", 'g', 0x68c4f9, 1, false, 1, 0, "", 0},
	{"b68c4fa", 'g', 0x68c4fa, 1, false, 1, 0, "", 0},
	{"d683fdb", 'g', 0x683fdb, 4, false, 1, 0, "", 0},
	{"d683fe7", 'g', 0x683fe7, 4, false, 1, 0, "", 0},
	{"d68407f", 'g', 0x68407f, 4, false, 1, 0, "", 0},
	{"d684083", 'g', 0x684083, 4, false, 1, 0, "", 0},
	{"d684087", 'g', 0x684087, 4, false, 1, 0, "", 0},
	{"d689aff", 'g', 0x689aff, 4, false, 1, 0, "", 0},
	{"d689b03", 'g', 0x689b03, 4, false, 1, 0, "", 0},
	{"d689afb", 'g', 0x689afb, 4, false, 1, 0, "", 0},
	{"fact36", 'g', 0x689bae, 1, false, 1, 0, "", 0},
	{"fact177", 'g', 0x689cef, 1, false, 1, 0, "", 0},
	{"facte2", 'g', 0x689c5a, 1, false, 1, 0, "", 0},
	{"fact24d", 'g', 0x689dc5, 1, false, 1, 0, "", 0},
	{"fact308", 'g', 0x689e80, 1, false, 1, 0, "", 0},
	{"fact273", 'g', 0x689deb, 1, false, 1, 0, "", 0},
	{"stand4", 'g', 0x68d720, 1, false, 1, 0, "", 0},
	{"stamp4", 'g', 0x68d726, 4, false, 1, 0, "", 0},
	{"stand5", 'g', 0x68d72e, 1, false, 1, 0, "", 0},
	{"stamp5", 'g', 0x68d734, 4, false, 1, 0, "", 0},
	{"stand9", 'g', 0x68d766, 1, false, 1, 0, "", 0},
	{"stamp9", 'g', 0x68d76c, 4, false, 1, 0, "", 0},
	{"standa", 'g', 0x68d774, 1, false, 1, 0, "", 0},
	{"stampa", 'g', 0x68d77a, 4, false, 1, 0, "", 0},
	{"standb", 'g', 0x68d782, 1, false, 1, 0, "", 0},
	{"stampb", 'g', 0x68d788, 4, false, 1, 0, "", 0},
	{"standc", 'g', 0x68d790, 1, false, 1, 0, "", 0},
	{"stampc", 'g', 0x68d796, 4, false, 1, 0, "", 0},
	{"standf", 'g', 0x68d7ba, 1, false, 1, 0, "", 0},
	{"stampf", 'g', 0x68d7c0, 4, false, 1, 0, "", 0},
	{"stand10", 'g', 0x68d7c8, 1, false, 1, 0, "", 0},
	{"stamp10", 'g', 0x68d7ce, 4, false, 1, 0, "", 0},
	{"fa10", 'g', 0x68d7d2, 1, false, 1, 0, "", 0},
	{"t620_", 'g', 0x620400, 4, false, 48, 28, "", 0},
	{"c0a2a9", 'o', 0x200102a9, 4, false, 1, 0, "", 0},
	{"c0f", 'o', 0x2001019d, 1, false, 41, 38, "", 0},
	{"c0eff7", 'o', 0x200102ad, 4, false, 1, 0, "", 0},
	{"c0att1", 'o', 0x200100f9, 4, false, 1, 0, "", 0},
	{"c0fd", 'o', 0x200100fd, 4, false, 1, 0, "", 0},
	{"c0apt", 'o', 0x200101a7, 4, false, 41, 38, "", 0},
	{"c1a2a9", 'o', 0x20011b0b, 4, false, 1, 0, "", 0},
	{"c1f", 'o', 0x200119ff, 1, false, 41, 38, "", 0},
	{"c1eff7", 'o', 0x20011b0f, 4, false, 1, 0, "", 0},
	{"c1att1", 'o', 0x2001195b, 4, false, 1, 0, "", 0},
	{"c1fd", 'o', 0x2001195f, 4, false, 1, 0, "", 0},
	{"c1apt", 'o', 0x20011a09, 4, false, 41, 38, "", 0},
	{"c2a2a9", 'o', 0x2001336d, 4, false, 1, 0, "", 0},
	{"c2f", 'o', 0x20013261, 1, false, 41, 38, "", 0},
	{"c2eff7", 'o', 0x20013371, 4, false, 1, 0, "", 0},
	{"c2att1", 'o', 0x200131bd, 4, false, 1, 0, "", 0},
	{"c2fd", 'o', 0x200131c1, 4, false, 1, 0, "", 0},
	{"c2apt", 'o', 0x2001326b, 4, false, 41, 38, "", 0},
	{"c3a2a9", 'o', 0x20014bcf, 4, false, 1, 0, "", 0},
	{"c3f", 'o', 0x20014ac3, 1, false, 41, 38, "", 0},
	{"c3eff7", 'o', 0x20014bd3, 4, false, 1, 0, "", 0},
	{"c3att1", 'o', 0x20014a1f, 4, false, 1, 0, "", 0},
	{"c3fd", 'o', 0x20014a23, 4, false, 1, 0, "", 0},
	{"c3apt", 'o', 0x20014acd, 4, false, 41, 38, "", 0},
	{"c4a2a9", 'o', 0x20016431, 4, false, 1, 0, "", 0},
	{"c4f", 'o', 0x20016325, 1, false, 41, 38, "", 0},
	{"c4eff7", 'o', 0x20016435, 4, false, 1, 0, "", 0},
	{"c4att1", 'o', 0x20016281, 4, false, 1, 0, "", 0},
	{"c4fd", 'o', 0x20016285, 4, false, 1, 0, "", 0},
	{"c4apt", 'o', 0x2001632f, 4, false, 41, 38, "", 0},
	{"c5a2a9", 'o', 0x20017c93, 4, false, 1, 0, "", 0},
	{"c5f", 'o', 0x20017b87, 1, false, 41, 38, "", 0},
	{"c5eff7", 'o', 0x20017c97, 4, false, 1, 0, "", 0},
	{"c5att1", 'o', 0x20017ae3, 4, false, 1, 0, "", 0},
	{"c5fd", 'o', 0x20017ae7, 4, false, 1, 0, "", 0},
	{"c5apt", 'o', 0x20017b91, 4, false, 41, 38, "", 0},
	{"c6a2a9", 'o', 0x200194f5, 4, false, 1, 0, "", 0},
	{"c6f", 'o', 0x200193e9, 1, false, 41, 38, "", 0},
	{"c6eff7", 'o', 0x200194f9, 4, false, 1, 0, "", 0},
	{"c6att1", 'o', 0x20019345, 4, false, 1, 0, "", 0},
	{"c6fd", 'o', 0x20019349, 4, false, 1, 0, "", 0},
	{"c6apt", 'o', 0x200193f3, 4, false, 41, 38, "", 0},
	{"c7a2a9", 'o', 0x2001ad57, 4, false, 1, 0, "", 0},
	{"c7f", 'o', 0x2001ac4b, 1, false, 41, 38, "", 0},
	{"c7eff7", 'o', 0x2001ad5b, 4, false, 1, 0, "", 0},
	{"c7att1", 'o', 0x2001aba7, 4, false, 1, 0, "", 0},
	{"c7fd", 'o', 0x2001abab, 4, false, 1, 0, "", 0},
	{"c7apt", 'o', 0x2001ac55, 4, false, 41, 38, "", 0},
	{"n0w0", 'o', 0x2001e100, 4, false, 1, 0, "", 0},
	{"n0b1a", 'o', 0x2001e11a, 1, false, 1, 0, "", 0},
	{"n0bc7", 'o', 0x2001e1c7, 1, false, 1, 0, "", 0},
	{"n0be9", 'o', 0x2001e1e9, 1, false, 1, 0, "", 0},
	{"n0b114", 'o', 0x2001e214, 1, false, 1, 0, "", 0},
	{"n0b112", 'o', 0x2001e212, 1, false, 1, 0, "", 0},
	{"n0b113", 'o', 0x2001e213, 1, false, 1, 0, "", 0},
	{"n0bea", 'o', 0x2001e1ea, 1, false, 1, 0, "", 0},
	{"n0bc5", 'o', 0x2001e1c5, 1, false, 1, 0, "", 0},
	{"n0bc6", 'o', 0x2001e1c6, 1, false, 1, 0, "", 0},
	{"n0s", 'o', 0x2001e19d, 1, false, 13, 1, "", 0},
	{"n0ws", 'o', 0x2001e130, 2, false, 40, 2, "", 0},
	{"k0cnt", 'o', 0x2001f008, 4, false, 1, 0, "", 0},
	{"k0d", 'o', 0x2001f010, 4, false, 6, 4, "", 0},
	{"k0e0_0", 'o', 0x2001f040, 4, false, 1, 0, "", 0},
	{"k0e0_4", 'o', 0x2001f044, 4, false, 1, 0, "", 0},
	{"k0e0_8", 'o', 0x2001f048, 4, false, 1, 0, "", 0},
	{"k0e0_10", 'o', 0x2001f050, 4, false, 1, 0, "", 0},
	{"k0e1_0", 'o', 0x2001f060, 4, false, 1, 0, "", 0},
	{"k0e1_4", 'o', 0x2001f064, 4, false, 1, 0, "", 0},
	{"k0e1_8", 'o', 0x2001f068, 4, false, 1, 0, "", 0},
	{"k0e1_10", 'o', 0x2001f070, 4, false, 1, 0, "", 0},
	{"k0e2_0", 'o', 0x2001f080, 4, false, 1, 0, "", 0},
	{"k0e2_4", 'o', 0x2001f084, 4, false, 1, 0, "", 0},
	{"k0e2_8", 'o', 0x2001f088, 4, false, 1, 0, "", 0},
	{"k0e2_10", 'o', 0x2001f090, 4, false, 1, 0, "", 0},
	{"k0e3_0", 'o', 0x2001f0a0, 4, false, 1, 0, "", 0},
	{"k0e3_4", 'o', 0x2001f0a4, 4, false, 1, 0, "", 0},
	{"k0e3_8", 'o', 0x2001f0a8, 4, false, 1, 0, "", 0},
	{"k0e3_10", 'o', 0x2001f0b0, 4, false, 1, 0, "", 0},
	{"k0e4_0", 'o', 0x2001f0c0, 4, false, 1, 0, "", 0},
	{"k0e4_4", 'o', 0x2001f0c4, 4, false, 1, 0, "", 0},
	{"k0e4_8", 'o', 0x2001f0c8, 4, false, 1, 0, "", 0},
	{"k0e4_10", 'o', 0x2001f0d0, 4, false, 1, 0, "", 0},
	{"k0e5_0", 'o', 0x2001f0e0, 4, false, 1, 0, "", 0},
	{"k0e5_4", 'o', 0x2001f0e4, 4, false, 1, 0, "", 0},
	{"k0e5_8", 'o', 0x2001f0e8, 4, false, 1, 0, "", 0},
	{"k0e5_10", 'o', 0x2001f0f0, 4, false, 1, 0, "", 0},
	{"n1w0", 'o', 0x2001e240, 4, false, 1, 0, "", 0},
	{"n1b1a", 'o', 0x2001e25a, 1, false, 1, 0, "", 0},
	{"n1bc7", 'o', 0x2001e307, 1, false, 1, 0, "", 0},
	{"n1be9", 'o', 0x2001e329, 1, false, 1, 0, "", 0},
	{"n1b114", 'o', 0x2001e354, 1, false, 1, 0, "", 0},
	{"n1b112", 'o', 0x2001e352, 1, false, 1, 0, "", 0},
	{"n1b113", 'o', 0x2001e353, 1, false, 1, 0, "", 0},
	{"n1bea", 'o', 0x2001e32a, 1, false, 1, 0, "", 0},
	{"n1bc5", 'o', 0x2001e305, 1, false, 1, 0, "", 0},
	{"n1bc6", 'o', 0x2001e306, 1, false, 1, 0, "", 0},
	{"n1s", 'o', 0x2001e2dd, 1, false, 13, 1, "", 0},
	{"n1ws", 'o', 0x2001e270, 2, false, 40, 2, "", 0},
	{"k1cnt", 'o', 0x2001f108, 4, false, 1, 0, "", 0},
	{"k1d", 'o', 0x2001f110, 4, false, 6, 4, "", 0},
	{"k1e0_0", 'o', 0x2001f140, 4, false, 1, 0, "", 0},
	{"k1e0_4", 'o', 0x2001f144, 4, false, 1, 0, "", 0},
	{"k1e0_8", 'o', 0x2001f148, 4, false, 1, 0, "", 0},
	{"k1e0_10", 'o', 0x2001f150, 4, false, 1, 0, "", 0},
	{"k1e1_0", 'o', 0x2001f160, 4, false, 1, 0, "", 0},
	{"k1e1_4", 'o', 0x2001f164, 4, false, 1, 0, "", 0},
	{"k1e1_8", 'o', 0x2001f168, 4, false, 1, 0, "", 0},
	{"k1e1_10", 'o', 0x2001f170, 4, false, 1, 0, "", 0},
	{"k1e2_0", 'o', 0x2001f180, 4, false, 1, 0, "", 0},
	{"k1e2_4", 'o', 0x2001f184, 4, false, 1, 0, "", 0},
	{"k1e2_8", 'o', 0x2001f188, 4, false, 1, 0, "", 0},
	{"k1e2_10", 'o', 0x2001f190, 4, false, 1, 0, "", 0},
	{"k1e3_0", 'o', 0x2001f1a0, 4, false, 1, 0, "", 0},
	{"k1e3_4", 'o', 0x2001f1a4, 4, false, 1, 0, "", 0},
	{"k1e3_8", 'o', 0x2001f1a8, 4, false, 1, 0, "", 0},
	{"k1e3_10", 'o', 0x2001f1b0, 4, false, 1, 0, "", 0},
	{"k1e4_0", 'o', 0x2001f1c0, 4, false, 1, 0, "", 0},
	{"k1e4_4", 'o', 0x2001f1c4, 4, false, 1, 0, "", 0},
	{"k1e4_8", 'o', 0x2001f1c8, 4, false, 1, 0, "", 0},
	{"k1e4_10", 'o', 0x2001f1d0, 4, false, 1, 0, "", 0},
	{"k1e5_0", 'o', 0x2001f1e0, 4, false, 1, 0, "", 0},
	{"k1e5_4", 'o', 0x2001f1e4, 4, false, 1, 0, "", 0},
	{"k1e5_8", 'o', 0x2001f1e8, 4, false, 1, 0, "", 0},
	{"k1e5_10", 'o', 0x2001f1f0, 4, false, 1, 0, "", 0},
	{"n2w0", 'o', 0x2001e380, 4, false, 1, 0, "", 0},
	{"n2b1a", 'o', 0x2001e39a, 1, false, 1, 0, "", 0},
	{"n2bc7", 'o', 0x2001e447, 1, false, 1, 0, "", 0},
	{"n2be9", 'o', 0x2001e469, 1, false, 1, 0, "", 0},
	{"n2b114", 'o', 0x2001e494, 1, false, 1, 0, "", 0},
	{"n2b112", 'o', 0x2001e492, 1, false, 1, 0, "", 0},
	{"n2b113", 'o', 0x2001e493, 1, false, 1, 0, "", 0},
	{"n2bea", 'o', 0x2001e46a, 1, false, 1, 0, "", 0},
	{"n2bc5", 'o', 0x2001e445, 1, false, 1, 0, "", 0},
	{"n2bc6", 'o', 0x2001e446, 1, false, 1, 0, "", 0},
	{"n2s", 'o', 0x2001e41d, 1, false, 13, 1, "", 0},
	{"n2ws", 'o', 0x2001e3b0, 2, false, 40, 2, "", 0},
	{"k2cnt", 'o', 0x2001f208, 4, false, 1, 0, "", 0},
	{"k2d", 'o', 0x2001f210, 4, false, 6, 4, "", 0},
	{"k2e0_0", 'o', 0x2001f240, 4, false, 1, 0, "", 0},
	{"k2e0_4", 'o', 0x2001f244, 4, false, 1, 0, "", 0},
	{"k2e0_8", 'o', 0x2001f248, 4, false, 1, 0, "", 0},
	{"k2e0_10", 'o', 0x2001f250, 4, false, 1, 0, "", 0},
	{"k2e1_0", 'o', 0x2001f260, 4, false, 1, 0, "", 0},
	{"k2e1_4", 'o', 0x2001f264, 4, false, 1, 0, "", 0},
	{"k2e1_8", 'o', 0x2001f268, 4, false, 1, 0, "", 0},
	{"k2e1_10", 'o', 0x2001f270, 4, false, 1, 0, "", 0},
	{"k2e2_0", 'o', 0x2001f280, 4, false, 1, 0, "", 0},
	{"k2e2_4", 'o', 0x2001f284, 4, false, 1, 0, "", 0},
	{"k2e2_8", 'o', 0x2001f288, 4, false, 1, 0, "", 0},
	{"k2e2_10", 'o', 0x2001f290, 4, false, 1, 0, "", 0},
	{"k2e3_0", 'o', 0x2001f2a0, 4, false, 1, 0, "", 0},
	{"k2e3_4", 'o', 0x2001f2a4, 4, false, 1, 0, "", 0},
	{"k2e3_8", 'o', 0x2001f2a8, 4, false, 1, 0, "", 0},
	{"k2e3_10", 'o', 0x2001f2b0, 4, false, 1, 0, "", 0},
	{"k2e4_0", 'o', 0x2001f2c0, 4, false, 1, 0, "", 0},
	{"k2e4_4", 'o', 0x2001f2c4, 4, false, 1, 0, "", 0},
	{"k2e4_8", 'o', 0x2001f2c8, 4, false, 1, 0, "", 0},
	{"k2e4_10", 'o', 0x2001f2d0, 4, false, 1, 0, "", 0},
	{"k2e5_0", 'o', 0x2001f2e0, 4, false, 1, 0, "", 0},
	{"k2e5_4", 'o', 0x2001f2e4, 4, false, 1, 0, "", 0},
	{"k2e5_8", 'o', 0x2001f2e8, 4, false, 1, 0, "", 0},
	{"k2e5_10", 'o', 0x2001f2f0, 4, false, 1, 0, "", 0},
	{"n3w0", 'o', 0x2001e4c0, 4, false, 1, 0, "", 0},
	{"n3b1a", 'o', 0x2001e4da, 1, false, 1, 0, "", 0},
	{"n3bc7", 'o', 0x2001e587, 1, false, 1, 0, "", 0},
	{"n3be9", 'o', 0x2001e5a9, 1, false, 1, 0, "", 0},
	{"n3b114", 'o', 0x2001e5d4, 1, false, 1, 0, "", 0},
	{"n3b112", 'o', 0x2001e5d2, 1, false, 1, 0, "", 0},
	{"n3b113", 'o', 0x2001e5d3, 1, false, 1, 0, "", 0},
	{"n3bea", 'o', 0x2001e5aa, 1, false, 1, 0, "", 0},
	{"n3bc5", 'o', 0x2001e585, 1, false, 1, 0, "", 0},
	{"n3bc6", 'o', 0x2001e586, 1, false, 1, 0, "", 0},
	{"n3s", 'o', 0x2001e55d, 1, false, 13, 1, "", 0},
	{"n3ws", 'o', 0x2001e4f0, 2, false, 40, 2, "", 0},
	{"k3cnt", 'o', 0x2001f308, 4, false, 1, 0, "", 0},
	{"k3d", 'o', 0x2001f310, 4, false, 6, 4, "", 0},
	{"k3e0_0", 'o', 0x2001f340, 4, false, 1, 0, "", 0},
	{"k3e0_4", 'o', 0x2001f344, 4, false, 1, 0, "", 0},
	{"k3e0_8", 'o', 0x2001f348, 4, false, 1, 0, "", 0},
	{"k3e0_10", 'o', 0x2001f350, 4, false, 1, 0, "", 0},
	{"k3e1_0", 'o', 0x2001f360, 4, false, 1, 0, "", 0},
	{"k3e1_4", 'o', 0x2001f364, 4, false, 1, 0, "", 0},
	{"k3e1_8", 'o', 0x2001f368, 4, false, 1, 0, "", 0},
	{"k3e1_10", 'o', 0x2001f370, 4, false, 1, 0, "", 0},
	{"k3e2_0", 'o', 0x2001f380, 4, false, 1, 0, "", 0},
	{"k3e2_4", 'o', 0x2001f384, 4, false, 1, 0, "", 0},
	{"k3e2_8", 'o', 0x2001f388, 4, false, 1, 0, "", 0},
	{"k3e2_10", 'o', 0x2001f390, 4, false, 1, 0, "", 0},
	{"k3e3_0", 'o', 0x2001f3a0, 4, false, 1, 0, "", 0},
	{"k3e3_4", 'o', 0x2001f3a4, 4, false, 1, 0, "", 0},
	{"k3e3_8", 'o', 0x2001f3a8, 4, false, 1, 0, "", 0},
	{"k3e3_10", 'o', 0x2001f3b0, 4, false, 1, 0, "", 0},
	{"k3e4_0", 'o', 0x2001f3c0, 4, false, 1, 0, "", 0},
	{"k3e4_4", 'o', 0x2001f3c4, 4, false, 1, 0, "", 0},
	{"k3e4_8", 'o', 0x2001f3c8, 4, false, 1, 0, "", 0},
	{"k3e4_10", 'o', 0x2001f3d0, 4, false, 1, 0, "", 0},
	{"k3e5_0", 'o', 0x2001f3e0, 4, false, 1, 0, "", 0},
	{"k3e5_4", 'o', 0x2001f3e4, 4, false, 1, 0, "", 0},
	{"k3e5_8", 'o', 0x2001f3e8, 4, false, 1, 0, "", 0},
	{"k3e5_10", 'o', 0x2001f3f0, 4, false, 1, 0, "", 0},
	{"n4w0", 'o', 0x2001e600, 4, false, 1, 0, "", 0},
	{"n4b1a", 'o', 0x2001e61a, 1, false, 1, 0, "", 0},
	{"n4bc7", 'o', 0x2001e6c7, 1, false, 1, 0, "", 0},
	{"n4be9", 'o', 0x2001e6e9, 1, false, 1, 0, "", 0},
	{"n4b114", 'o', 0x2001e714, 1, false, 1, 0, "", 0},
	{"n4b112", 'o', 0x2001e712, 1, false, 1, 0, "", 0},
	{"n4b113", 'o', 0x2001e713, 1, false, 1, 0, "", 0},
	{"n4bea", 'o', 0x2001e6ea, 1, false, 1, 0, "", 0},
	{"n4bc5", 'o', 0x2001e6c5, 1, false, 1, 0, "", 0},
	{"n4bc6", 'o', 0x2001e6c6, 1, false, 1, 0, "", 0},
	{"n4s", 'o', 0x2001e69d, 1, false, 13, 1, "", 0},
	{"n4ws", 'o', 0x2001e630, 2, false, 40, 2, "", 0},
	{"k4cnt", 'o', 0x2001f408, 4, false, 1, 0, "", 0},
	{"k4d", 'o', 0x2001f410, 4, false, 6, 4, "", 0},
	{"k4e0_0", 'o', 0x2001f440, 4, false, 1, 0, "", 0},
	{"k4e0_4", 'o', 0x2001f444, 4, false, 1, 0, "", 0},
	{"k4e0_8", 'o', 0x2001f448, 4, false, 1, 0, "", 0},
	{"k4e0_10", 'o', 0x2001f450, 4, false, 1, 0, "", 0},
	{"k4e1_0", 'o', 0x2001f460, 4, false, 1, 0, "", 0},
	{"k4e1_4", 'o', 0x2001f464, 4, false, 1, 0, "", 0},
	{"k4e1_8", 'o', 0x2001f468, 4, false, 1, 0, "", 0},
	{"k4e1_10", 'o', 0x2001f470, 4, false, 1, 0, "", 0},
	{"k4e2_0", 'o', 0x2001f480, 4, false, 1, 0, "", 0},
	{"k4e2_4", 'o', 0x2001f484, 4, false, 1, 0, "", 0},
	{"k4e2_8", 'o', 0x2001f488, 4, false, 1, 0, "", 0},
	{"k4e2_10", 'o', 0x2001f490, 4, false, 1, 0, "", 0},
	{"k4e3_0", 'o', 0x2001f4a0, 4, false, 1, 0, "", 0},
	{"k4e3_4", 'o', 0x2001f4a4, 4, false, 1, 0, "", 0},
	{"k4e3_8", 'o', 0x2001f4a8, 4, false, 1, 0, "", 0},
	{"k4e3_10", 'o', 0x2001f4b0, 4, false, 1, 0, "", 0},
	{"k4e4_0", 'o', 0x2001f4c0, 4, false, 1, 0, "", 0},
	{"k4e4_4", 'o', 0x2001f4c4, 4, false, 1, 0, "", 0},
	{"k4e4_8", 'o', 0x2001f4c8, 4, false, 1, 0, "", 0},
	{"k4e4_10", 'o', 0x2001f4d0, 4, false, 1, 0, "", 0},
	{"k4e5_0", 'o', 0x2001f4e0, 4, false, 1, 0, "", 0},
	{"k4e5_4", 'o', 0x2001f4e4, 4, false, 1, 0, "", 0},
	{"k4e5_8", 'o', 0x2001f4e8, 4, false, 1, 0, "", 0},
	{"k4e5_10", 'o', 0x2001f4f0, 4, false, 1, 0, "", 0},
	{"n5w0", 'o', 0x2001e740, 4, false, 1, 0, "", 0},
	{"n5b1a", 'o', 0x2001e75a, 1, false, 1, 0, "", 0},
	{"n5bc7", 'o', 0x2001e807, 1, false, 1, 0, "", 0},
	{"n5be9", 'o', 0x2001e829, 1, false, 1, 0, "", 0},
	{"n5b114", 'o', 0x2001e854, 1, false, 1, 0, "", 0},
	{"n5b112", 'o', 0x2001e852, 1, false, 1, 0, "", 0},
	{"n5b113", 'o', 0x2001e853, 1, false, 1, 0, "", 0},
	{"n5bea", 'o', 0x2001e82a, 1, false, 1, 0, "", 0},
	{"n5bc5", 'o', 0x2001e805, 1, false, 1, 0, "", 0},
	{"n5bc6", 'o', 0x2001e806, 1, false, 1, 0, "", 0},
	{"n5s", 'o', 0x2001e7dd, 1, false, 13, 1, "", 0},
	{"n5ws", 'o', 0x2001e770, 2, false, 40, 2, "", 0},
	{"k5cnt", 'o', 0x2001f508, 4, false, 1, 0, "", 0},
	{"k5d", 'o', 0x2001f510, 4, false, 6, 4, "", 0},
	{"k5e0_0", 'o', 0x2001f540, 4, false, 1, 0, "", 0},
	{"k5e0_4", 'o', 0x2001f544, 4, false, 1, 0, "", 0},
	{"k5e0_8", 'o', 0x2001f548, 4, false, 1, 0, "", 0},
	{"k5e0_10", 'o', 0x2001f550, 4, false, 1, 0, "", 0},
	{"k5e1_0", 'o', 0x2001f560, 4, false, 1, 0, "", 0},
	{"k5e1_4", 'o', 0x2001f564, 4, false, 1, 0, "", 0},
	{"k5e1_8", 'o', 0x2001f568, 4, false, 1, 0, "", 0},
	{"k5e1_10", 'o', 0x2001f570, 4, false, 1, 0, "", 0},
	{"k5e2_0", 'o', 0x2001f580, 4, false, 1, 0, "", 0},
	{"k5e2_4", 'o', 0x2001f584, 4, false, 1, 0, "", 0},
	{"k5e2_8", 'o', 0x2001f588, 4, false, 1, 0, "", 0},
	{"k5e2_10", 'o', 0x2001f590, 4, false, 1, 0, "", 0},
	{"k5e3_0", 'o', 0x2001f5a0, 4, false, 1, 0, "", 0},
	{"k5e3_4", 'o', 0x2001f5a4, 4, false, 1, 0, "", 0},
	{"k5e3_8", 'o', 0x2001f5a8, 4, false, 1, 0, "", 0},
	{"k5e3_10", 'o', 0x2001f5b0, 4, false, 1, 0, "", 0},
	{"k5e4_0", 'o', 0x2001f5c0, 4, false, 1, 0, "", 0},
	{"k5e4_4", 'o', 0x2001f5c4, 4, false, 1, 0, "", 0},
	{"k5e4_8", 'o', 0x2001f5c8, 4, false, 1, 0, "", 0},
	{"k5e4_10", 'o', 0x2001f5d0, 4, false, 1, 0, "", 0},
	{"k5e5_0", 'o', 0x2001f5e0, 4, false, 1, 0, "", 0},
	{"k5e5_4", 'o', 0x2001f5e4, 4, false, 1, 0, "", 0},
	{"k5e5_8", 'o', 0x2001f5e8, 4, false, 1, 0, "", 0},
	{"k5e5_10", 'o', 0x2001f5f0, 4, false, 1, 0, "", 0},
	{"n6w0", 'o', 0x2001e880, 4, false, 1, 0, "", 0},
	{"n6b1a", 'o', 0x2001e89a, 1, false, 1, 0, "", 0},
	{"n6bc7", 'o', 0x2001e947, 1, false, 1, 0, "", 0},
	{"n6be9", 'o', 0x2001e969, 1, false, 1, 0, "", 0},
	{"n6b114", 'o', 0x2001e994, 1, false, 1, 0, "", 0},
	{"n6b112", 'o', 0x2001e992, 1, false, 1, 0, "", 0},
	{"n6b113", 'o', 0x2001e993, 1, false, 1, 0, "", 0},
	{"n6bea", 'o', 0x2001e96a, 1, false, 1, 0, "", 0},
	{"n6bc5", 'o', 0x2001e945, 1, false, 1, 0, "", 0},
	{"n6bc6", 'o', 0x2001e946, 1, false, 1, 0, "", 0},
	{"n6s", 'o', 0x2001e91d, 1, false, 13, 1, "", 0},
	{"n6ws", 'o', 0x2001e8b0, 2, false, 40, 2, "", 0},
	{"k6cnt", 'o', 0x2001f608, 4, false, 1, 0, "", 0},
	{"k6d", 'o', 0x2001f610, 4, false, 6, 4, "", 0},
	{"k6e0_0", 'o', 0x2001f640, 4, false, 1, 0, "", 0},
	{"k6e0_4", 'o', 0x2001f644, 4, false, 1, 0, "", 0},
	{"k6e0_8", 'o', 0x2001f648, 4, false, 1, 0, "", 0},
	{"k6e0_10", 'o', 0x2001f650, 4, false, 1, 0, "", 0},
	{"k6e1_0", 'o', 0x2001f660, 4, false, 1, 0, "", 0},
	{"k6e1_4", 'o', 0x2001f664, 4, false, 1, 0, "", 0},
	{"k6e1_8", 'o', 0x2001f668, 4, false, 1, 0, "", 0},
	{"k6e1_10", 'o', 0x2001f670, 4, false, 1, 0, "", 0},
	{"k6e2_0", 'o', 0x2001f680, 4, false, 1, 0, "", 0},
	{"k6e2_4", 'o', 0x2001f684, 4, false, 1, 0, "", 0},
	{"k6e2_8", 'o', 0x2001f688, 4, false, 1, 0, "", 0},
	{"k6e2_10", 'o', 0x2001f690, 4, false, 1, 0, "", 0},
	{"k6e3_0", 'o', 0x2001f6a0, 4, false, 1, 0, "", 0},
	{"k6e3_4", 'o', 0x2001f6a4, 4, false, 1, 0, "", 0},
	{"k6e3_8", 'o', 0x2001f6a8, 4, false, 1, 0, "", 0},
	{"k6e3_10", 'o', 0x2001f6b0, 4, false, 1, 0, "", 0},
	{"k6e4_0", 'o', 0x2001f6c0, 4, false, 1, 0, "", 0},
	{"k6e4_4", 'o', 0x2001f6c4, 4, false, 1, 0, "", 0},
	{"k6e4_8", 'o', 0x2001f6c8, 4, false, 1, 0, "", 0},
	{"k6e4_10", 'o', 0x2001f6d0, 4, false, 1, 0, "", 0},
	{"k6e5_0", 'o', 0x2001f6e0, 4, false, 1, 0, "", 0},
	{"k6e5_4", 'o', 0x2001f6e4, 4, false, 1, 0, "", 0},
	{"k6e5_8", 'o', 0x2001f6e8, 4, false, 1, 0, "", 0},
	{"k6e5_10", 'o', 0x2001f6f0, 4, false, 1, 0, "", 0},
	{"o0a0", 'o', 0x20026ca0, 4, false, 1, 0, "", 0},
	{"o1a0", 'o', 0x20026da0, 4, false, 1, 0, "", 0},
	{"o2a0", 'o', 0x20026ea0, 4, false, 1, 0, "", 0},
	{"o451_264", 'o', 0x20028264, 4, false, 1, 0, "", 0},
	{"o451_188", 'o', 0x20028188, 4, false, 1, 0, "", 0},
	{"e0", 'o', 0x20027000, 4, false, 1, 0, "", 0},
	{"e48", 'o', 0x20027048, 4, false, 1, 0, "", 0},
}

var sagSetupG = []n4Set{
	{0x685174, 4, 0x20010000, 1, 0},
	{0x685178, 4, 0x2001d000, 1, 0},
	{0x689f94, 4, 0x2001e000, 1, 0},
	{0x2001e004, 4, 0x6, 1, 0},
	{0x2001e00c, 4, 0x2001e010, 1, 0},
	{0x2001e010, 4, 0x2001e100, 1, 0},
	{0x2001e106, 4, 0x2001ec00, 1, 0},
	{0x2001f000, 4, 0x2001f010, 1, 0},
	{0x2001f004, 4, 0xa, 1, 0},
	{0x2001e014, 4, 0x2001e240, 1, 0},
	{0x2001e246, 4, 0x2001ec80, 1, 0},
	{0x2001f100, 4, 0x2001f110, 1, 0},
	{0x2001f104, 4, 0xa, 1, 0},
	{0x2001e018, 4, 0x2001e380, 1, 0},
	{0x2001e386, 4, 0x2001ed00, 1, 0},
	{0x2001f200, 4, 0x2001f210, 1, 0},
	{0x2001f204, 4, 0xa, 1, 0},
	{0x2001e01c, 4, 0x2001e4c0, 1, 0},
	{0x2001e4c6, 4, 0x2001ed80, 1, 0},
	{0x2001f300, 4, 0x2001f310, 1, 0},
	{0x2001f304, 4, 0xa, 1, 0},
	{0x2001e020, 4, 0x2001e600, 1, 0},
	{0x2001e606, 4, 0x2001ee00, 1, 0},
	{0x2001f400, 4, 0x2001f410, 1, 0},
	{0x2001f404, 4, 0xa, 1, 0},
	{0x2001e024, 4, 0x2001e740, 1, 0},
	{0x2001e746, 4, 0x2001ee80, 1, 0},
	{0x2001f500, 4, 0x2001f510, 1, 0},
	{0x2001f504, 4, 0xa, 1, 0},
	{0x2001e886, 4, 0x2001ef00, 1, 0},
	{0x2001f600, 4, 0x2001f610, 1, 0},
	{0x2001f604, 4, 0xa, 1, 0},
	{0x683fad, 4, 0x20020000, 1, 0},
	{0x20020000, 4, 0x20020010, 1, 0},
	{0x20020008, 4, 0x3, 1, 0},
	{0x20020010, 4, 0x20020100, 1, 0},
	{0x20020014, 4, 0x20020140, 1, 0},
	{0x20020018, 4, 0x20020180, 1, 0},
	{0x683fb9, 4, 0x20020800, 1, 0},
	{0x20020800, 4, 0x20020810, 1, 0},
	{0x20020808, 4, 0x2, 1, 0},
	{0x20020810, 4, 0x20020900, 1, 0},
	{0x20020814, 4, 0x20020940, 1, 0},
	{0x683fb1, 4, 0x20022000, 1, 0},
	{0x20022000, 4, 0x20022010, 1, 0},
	{0x20022008, 4, 0x3, 1, 0},
	{0x20022010, 4, 0x20022100, 1, 0},
	{0x20022014, 4, 0x20022200, 1, 0},
	{0x20022018, 4, 0x20022300, 1, 0},
	{0x683fbd, 4, 0x20022800, 1, 0},
	{0x20022800, 4, 0x20022810, 1, 0},
	{0x20022808, 4, 0x2, 1, 0},
	{0x20022810, 4, 0x20022900, 1, 0},
	{0x20022814, 4, 0x20022a00, 1, 0},
	{0x2002010c, 4, 0x20020c00, 1, 0},
	{0x2002014c, 4, 0x20020e00, 1, 0},
	{0x2002018c, 4, 0x20021000, 1, 0},
	{0x2002090c, 4, 0x20021200, 1, 0},
	{0x2002094c, 4, 0x20021400, 1, 0},
	{0x20020dd8, 4, 0x20021800, 1, 0},
	{0x20020fd8, 4, 0x20021900, 1, 0},
	{0x200211d8, 4, 0x20021a00, 1, 0},
	{0x200213d8, 4, 0x20021b00, 1, 0},
	{0x200215d8, 4, 0x20021c00, 1, 0},
	{0x68c09c, 4, 0x20024000, 1, 0},
	{0x68de40, 4, 0x20026400, 1, 0},
	{0x2002640c, 4, 0x20026420, 1, 0},
	{0x20026420, 4, 0xe0e00000, 1, 0},
	{0x20026424, 4, 0xe0e00001, 1, 0},
	{0x20026428, 4, 0xe0e00002, 1, 0},
	{0x2002642c, 4, 0xe0e00003, 1, 0},
	{0x20026430, 4, 0xe0e00004, 1, 0},
	{0x20026434, 4, 0xe0e00005, 1, 0},
	{0x20026438, 4, 0xe0e00006, 1, 0},
	{0x2002643c, 4, 0xe0e00007, 1, 0},
	{0x20026440, 4, 0xe0e00008, 1, 0},
	{0x20026444, 4, 0xe0e00009, 1, 0},
	{0x20026448, 4, 0xe0e0000a, 1, 0},
	{0x2002644c, 4, 0xe0e0000b, 1, 0},
	{0x68c4c8, 4, 0x20026500, 1, 0},
	{0x20026500, 4, 0xc0c00000, 1, 0},
	{0x20026504, 4, 0xc0c00001, 1, 0},
	{0x20026508, 4, 0xc0c00002, 1, 0},
	{0x2002650c, 4, 0xc0c00003, 1, 0},
	{0x659904, 4, 0x20026600, 1, 0},
	{0x65999c, 4, 0x20026640, 1, 0},
	{0x6598ec, 4, 0x20026680, 1, 0},
	{0x6836ac, 4, 0x20080000, 1, 0},
	{0x65be1c, 4, 0x20030000, 1, 0},
	{0x5eb70c, 4, 0x20027200, 1, 0},
	{0x2002722c, 4, 0x20027300, 1, 0},
	{0x20028264, 4, 0x26426426, 1, 0},
	{0x20028188, 4, 0x18818818, 1, 0},
	{0x20027048, 4, 0x48484848, 1, 0},
	{0x68407f, 4, 0x77777777, 1, 0},
	{0x684083, 4, 0x77777777, 1, 0},
	{0x684087, 4, 0x77777777, 1, 0},
	{0x689aff, 4, 0x66666666, 1, 0},
	{0x689b03, 4, 0x66666666, 1, 0},
	{0x689afb, 4, 0x66666666, 1, 0},
	{0x68d726, 4, 0x5a0004, 1, 0},
	{0x68d734, 4, 0x5a0005, 1, 0},
	{0x68d76c, 4, 0x5a0009, 1, 0},
	{0x68d77a, 4, 0x5a000a, 1, 0},
	{0x68d788, 4, 0x5a000b, 1, 0},
	{0x68d796, 4, 0x5a000c, 1, 0},
	{0x68d7c0, 4, 0x5a000f, 1, 0},
	{0x68d7ce, 4, 0x5a0010, 1, 0},
	{0x68d7d2, 1, 0x33, 1, 0},
}

var sagStubs = []n4Stub{
	{0x408860, "s408860", -1, 0x20006000, []n4Rec{{1, 4, 1, 0x20000000}}, 0x20000004, n4Wr{}},
	{0x4ae080, "s4ae080", 0, 0x20027000, []n4Rec{{1, 4, 1, 0x20000008}, {2, 4, 1, 0x2000000c}, {3, 4, 1, 0x20000010}, {4, 4, 1, 0x20000014}, {5, 4, 1, 0x20000018}}, 0x2000001c, n4Wr{}},
	{0x4512c0, "s4512c0", -1, 0x20006004, []n4Rec{{1, 4, 1, 0x20000020}, {3, 4, 1, 0x20000024}, {4, 4, 1, 0x20000028}}, 0x2000002c, n4Wr{}},
	{0x4f6c50, "s4f6c50", 2, 0x20006008, []n4Rec{{1, 4, 2, 0x20000030}, {3, 4, 2, 0x20000038}, {4, 4, 2, 0x20000040}}, 0x20000048, n4Wr{}},
	{0x4f6cf0, "s4f6cf0", 0, 0x0, []n4Rec{{1, 4, 2, 0x2000004c}}, 0x20000054, n4Wr{}},
	{0x4d8f00, "s4d8f00", 0, 0x0, []n4Rec{{1, 4, 1, 0x20000058}, {2, 4, 1, 0x2000005c}, {4, 4, 1, 0x20000060}, {5, 4, 1, 0x20000064}, {6, 4, 1, 0x20000068}, {7, 4, 1, 0x2000006c}}, 0x20000070, n4Wr{}},
	{0x420f90, "s420f90", 0, 0x0, []n4Rec{}, 0x20000074, n4Wr{}},
	{0x547570, "s547570", 0, 0x0, []n4Rec{{1, 4, 3, 0x20000078}, {2, 4, 3, 0x20000084}, {3, 4, 3, 0x20000090}}, 0x2000009c, n4Wr{}},
	{0x56e800, "s56e800", 0, 0x0, []n4Rec{{1, 4, 1, 0x200000a0}}, 0x200000a4, n4Wr{}},
	{0x50c560, "s50c560", 0, 0x0, []n4Rec{{1, 4, 1, 0x200000a8}, {2, 4, 1, 0x200000ac}}, 0x200000b0, n4Wr{}},
	{0x509aa0, "s509aa0", -1, 0x20006010, []n4Rec{{1, 4, 1, 0x200000b4}}, 0x200000b8, n4Wr{}},
	{0x49acd0, "s49acd0", 0, 0x0, []n4Rec{{1, 4, 2, 0x200000bc}}, 0x200000c4, n4Wr{}},
	{0x498d90, "s498d90", 0, 0x0, []n4Rec{{1, 4, 1, 0x200000c8}}, 0x200000cc, n4Wr{}},
	{0x5299b0, "s5299b0", 0, 0x0, []n4Rec{{1, 4, 1, 0x200000d0}}, 0x200000d4, n4Wr{}},
	{0x58aad0, "s58aad0", 0, 0x0, []n4Rec{{1, 4, 2, 0x200000d8}, {2, 4, 2, 0x200000e0}, {3, 4, 2, 0x200000e8}}, 0x200000f0, n4Wr{}},
	{0x483fd0, "s483fd0", 0, 0x0, []n4Rec{{1, 4, 1, 0x200000f4}}, 0x200000f8, n4Wr{}},
	{0x41f0d0, "s41f0d0", 0, 0x0, []n4Rec{}, 0x200000fc, n4Wr{}},
	{0x408ad0, "s408ad0", 0, 0x0, []n4Rec{{1, 4, 1, 0x20000100}, {2, 4, 1, 0x20000104}}, 0x20000108, n4Wr{}},
	{0x451080, "s451080", -1, 0x20006014, []n4Rec{{1, 4, 1, 0x2000010c}, {2, 4, 1, 0x20000110}}, 0x20000114, n4Wr{}},
	{0x5b1af0, "s5b1af0", 0, 0x0, []n4Rec{{1, 1, 336, 0x20000118}}, 0x20000268, n4Wr{}},
	{0x5b1b30, "s5b1b30", 0, 0x0, []n4Rec{{1, 1, 336, 0x2000026c}}, 0x200003bc, n4Wr{}},
	{0x569a50, "s569a50", 0, 0x0, []n4Rec{{1, 4, 8, 0x200003c0}, {2, 4, 8, 0x200003e0}, {3, 4, 8, 0x20000400}, {4, 4, 8, 0x20000420}}, 0x20000440, n4Wr{}},
	{0x56aab0, "s56aab0", 0, 0x0, []n4Rec{}, 0x20000444, n4Wr{}},
	{0x55ee70, "s55ee70", 0, 0x0, []n4Rec{{1, 4, 2, 0x20000448}}, 0x20000450, n4Wr{}},
	{0x4add30, "s4add30", 0, 0x0, []n4Rec{{1, 4, 1, 0x20000454}}, 0x20000458, n4Wr{}},
	{0x4f21e0, "s4f21e0", 0, 0x0, []n4Rec{{1, 4, 8, 0x2000045c}}, 0x2000047c, n4Wr{}},
	{0x4f2380, "s4f2380", 0, 0x0, []n4Rec{{1, 4, 8, 0x20000480}}, 0x200004a0, n4Wr{}},
	{0x43d940, "s43d940", 0, 0x0, []n4Rec{{1, 4, 3, 0x200004a4}}, 0x200004b0, n4Wr{}},
	{0x4c7f10, "s4c7f10", 0, 0x0, []n4Rec{{1, 4, 1, 0x200004b4}, {2, 4, 1, 0x200004b8}}, 0x200004bc, n4Wr{}},
	{0x4c2cf0, "s4c2cf0", -1, 0x20006018, []n4Rec{{1, 1, 1, 0x200004c0}}, 0x200004c4, n4Wr{}},
	{0x4e4db0, "s4e4db0", 0, 0x0, []n4Rec{{1, 4, 1, 0x200004c8}, {2, 4, 1, 0x200004cc}, {3, 4, 1, 0x200004d0}}, 0x200004d4, n4Wr{}},
	{0x4e3af0, "s4e3af0", 0, 0x0, []n4Rec{{1, 4, 1, 0x200004d8}, {2, 4, 1, 0x200004dc}}, 0x200004e0, n4Wr{}},
	{0x55a0a0, "s55a0a0", 0, 0x0, []n4Rec{{1, 4, 1, 0x200004e4}}, 0x200004e8, n4Wr{}},
	{0x524ca0, "s524ca0", 0, 0x0, []n4Rec{{1, 4, 1, 0x200004ec}}, 0x200004f0, n4Wr{}},
	{0x52e690, "s52e690", 0, 0x0, []n4Rec{{1, 4, 1, 0x200004f4}, {2, 4, 1, 0x200004f8}, {3, 4, 1, 0x200004fc}, {4, 4, 1, 0x20000500}, {5, 4, 1, 0x20000504}}, 0x20000508, n4Wr{}},
	{0x506280, "s506280", 4, 0x2000601c, []n4Rec{{1, 4, 4, 0x2000050c}}, 0x2000051c, n4Wr{}},
	{0x523330, "s523330", 0, 0x0, []n4Rec{{1, 4, 1, 0x20000520}, {2, 4, 1, 0x20000524}, {3, 4, 1, 0x20000528}}, 0x2000052c, n4Wr{}},
	{0x51d960, "s51d960", 0, 0x0, []n4Rec{{1, 4, 8, 0x20000530}}, 0x20000550, n4Wr{}},
	{0x4ed9d0, "s4ed9d0", 0, 0x0, []n4Rec{{1, 4, 8, 0x20000554}}, 0x20000574, n4Wr{}},
	{0x522fe0, "s522fe0", 0, 0x0, []n4Rec{{1, 4, 8, 0x20000578}, {2, 4, 8, 0x20000598}, {3, 4, 8, 0x200005b8}, {4, 4, 8, 0x200005d8}, {5, 4, 8, 0x200005f8}, {6, 4, 8, 0x20000618}}, 0x20000638, n4Wr{}},
	{0x55a7b0, "s55a7b0", -1, 0x2000602c, []n4Rec{{1, 4, 1, 0x2000063c}, {2, 4, 1, 0x20000640}, {3, 4, 1, 0x20000644}}, 0x20000648, n4Wr{}},
	{0x0, "find", 3, 0x20006030, []n4Rec{{2, 4, 3, 0x2000064c}}, 0x20000658, n4Wr{}},
}
