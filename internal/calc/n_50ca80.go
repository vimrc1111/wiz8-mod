package calc

import "fmt"

// npc_region_status 0x50ca80(): the per-tick region check of the two NPC party slots
// (build/wiz8.asm:362838..363215). Nothing runs in combat (byte 0x683f94) or while
// resting (byte 0x683fc5).
//
// Part A (byte 0x6875b6 pending): the NPC table *0x689f94 (+4 count, +0xc element
// pointers) is scanned for the first NPC whose template (NPC +6) has class 0x42 at +0x58.
// When that NPC is alive (byte +4) the flag is cleared and nothing else happens. Otherwise
// the region key (0x42b740 on map 0x686a70) decides: keys 9 and 10 leave the flag set,
// any other key gives party member *0x6875ef state 10 for 9999 (0x522fe0), a voice line
// *0x5ee638 first when the member's +0x75 is 0, clears the flag and sets story fact 0x2a6
// (0x5061a0).
//
// Part B (byte 0x68759e pending, more than 60 clock ticks since 0x68759a): the region key
// is looked up in the region table at exe 0x619df8 (voice byte, key, region mask; stride
// 12, key -1 ends it); an unknown key clears the flag. For NPC party slots 0 and 1 (party
// presence byte, character +0xb11 non-zero; NPC id at presence +0xfa) whose main state
// +0xb01 is below 15: the NPC kind byte +0x2e picks a complaint voice (0x6c for kind 0x11
// when no kind-0x10 NPC is up, 0x67 for kind 0x10 when no kind-0x11 NPC is up, 0x50b8f0),
// followed by 0x577520 and message 0x5289b0(7, slot); otherwise (and always on key 13) the
// NPC remembers the region (+0xd4+row), voices the region line when its template regions
// (+0x68) miss the row mask (rows 12 and 13 stay silent), and when they hit, the penalty
// switch (presence +0xfe) is set, the time stored at NPC +0xeb, the member rebuilt
// (0x50e650) and voice 0x56 played; the NPC's penalty byte +0xe8 is cleared. A state of
// 15..17 sets +0xe8 and keeps the pending flag; otherwise the flag is cleared at the end.
//
// Emu address-stubs the D4 callees 0x522fe0, 0x50e650 and 0x506670 (inside 0x5061a0) and
// the 排除 callees 0x52e690, 0x41f0d0 and 0x55ee70 (inside 0x577520): arguments and call
// counts are outputs; no caller reads their return. F also runs the real
// characterConditionApplyNode for 0x522fe0, whose +0xb01/+0xb11 writes Part B reads.
// 0x50e650's writes (character rule modifiers) and 0x506670(0x2a6, 1) (no handler for a
// set flag) are not read again by the node, so F records them like Emu.
//
// Modelled helpers: 0x42b740 (ctrsC.selKey), 0x50b800 / 0x5c3790 (NPC lookup: index at
// or past the count reads element 0, flagged +0xc7 counts as none), 0x50c9e0 (template
// regions & row mask), 0x50b8f0, 0x577520 (byte 0x6875a5 = 1), 0x5289b0 (message queue
// 0x68c4c0), 0x5061a0 -> 0x5588f0 (fact byte 0x689b78+id, event log *0x68de40). The
// debug/UI switches 0x688290, 0x683f8c, 0x68edcc, 0x68de44 and 0x6850d5 are fixed 0 in the
// case (0x506670 precedent); a set one panics. A null or flagged NPC where the node
// dereferences it, and a null NPC table in Part A, crash the original and panic here.
// Fixture caps: party slot 0..7, NPC ids >= 0, at most 4 NPC elements read, event log and
// message queue fixture buffers of 4 entries, heap use <= 0x100 bytes, backpack <= 6.
const (
	nrsArena   = 0x20000000
	nrsChars   = nrsArena           // *0x685174, character records 0x1862 bytes
	nrsPres    = nrsArena + 0x10000 // *0x685178, party presence records 0x106 bytes
	nrsTab     = nrsArena + 0x11000 // *0x689f94, NPC table
	nrsArr     = nrsArena + 0x11100 // NPC table +0xc, element pointers
	nrsObj     = nrsArena + 0x11200 // NPC objects, 0x100 apart
	nrsTpl     = nrsArena + 0x11800 // NPC templates, 0x80 apart
	nrsEvl     = nrsArena + 0x12000 // *0x68de40, event log (+4 count, +8 cap, +0xc buffer)
	nrsEvBuf   = nrsArena + 0x12100
	nrsQBuf    = nrsArena + 0x12200
	nrsStubAt  = nrsArena + 0x13000
	nrsFactTab = nrsArena + 0x80000 // *0x6836ac, fact text records 0x1d8 bytes (0x5588f0)
	nrsArrN    = 4
	nrsFixN    = 4
	nrsHeap    = 0x50000000
	nrsHeapW   = 0x100
	nrsChS     = 0x1862 // exe 0x50cb06..0x50cb1a
	nrsPresS   = 0x106  // exe 0x50cf30
	nrsFacts   = 0x689b78
	nrsRegions = 0x619df8 // region table: voice byte, +4 key, +8 mask (exe 0x50cc14, 0x50cea6)
	nrsQCnt    = 0x68c4c0
	nrsLock    = 0x6875a5 // set by 0x577520
)

type nrsStub struct {
	addr, at uint32
	nargs, n int
}

var nrsStubs = []nrsStub{{0x52e690, nrsStubAt, 5, 5}, {0x522fe0, nrsStubAt + 0x100, 6, 1},
	{0x50e650, nrsStubAt + 0x200, 1, 2}, {0x506670, nrsStubAt + 0x300, 2, 1},
	{0x41f0d0, nrsStubAt + 0x400, 0, 2}, {0x55ee70, nrsStubAt + 0x500, 1, 2}}

func nrsG(name string, addr uint32, size int, signed bool, count, step int) n4Grp {
	kind := byte('g')
	if addr >= nrsArena && addr < nrsHeap {
		kind = 'o'
	}
	return n4Grp{name, kind, addr, size, signed, count, step, "", 0}
}

var nrsInG = func() []n4Grp {
	g := []n4Grp{nrsG("combat", 0x683f94, 1, false, 1, 4), nrsG("rest", 0x683fc5, 1, false, 1, 4),
		nrsG("pendA", 0x6875b6, 1, false, 1, 4), nrsG("pendB", 0x68759e, 1, false, 1, 4),
		nrsG("camLock", nrsLock, 1, false, 1, 4), nrsG("clock", 0x686a48, 4, false, 1, 4),
		nrsG("lastCheck", 0x68759a, 4, false, 1, 4), nrsG("map", 0x686a70, 4, true, 1, 4),
		nrsG("slot", 0x6875ef, 4, false, 1, 4), nrsG("npcTab", 0x689f94, 4, false, 1, 4),
		nrsG("f2a6", nrsFacts+0x2a6, 1, false, 1, 4), nrsG("f1ce", nrsFacts+0x1ce, 1, false, 1, 4),
		nrsG("curF", 0x6874ca, 1, false, 1, 4), nrsG("curId", 0x6874cb, 4, false, 1, 4),
		nrsG("bpn", 0x686901, 4, false, 1, 4), nrsG("bp", 0x685191, 4, false, ctrsBpMax, 12),
		nrsG("qcnt", nrsQCnt, 4, true, 1, 4), nrsG("qcap", nrsQCnt+4, 4, true, 1, 4),
		nrsG("qstamp", 0x68c4ac, 4, false, 1, 4),
		nrsG("evcnt", nrsEvl+4, 4, true, 1, 4), nrsG("evcap", nrsEvl+8, 4, true, 1, 4),
		nrsG("npcN", nrsTab+4, 4, true, 1, 4)}
	for i := uint32(0); i < nrsArrN; i++ {
		g = append(g, nrsG(fmt.Sprintf("arr%d", i), nrsArr+4*i, 4, false, 1, 4))
	}
	for i := uint32(0); i < nrsArrN; i++ {
		o, t, p := nrsObj+0x100*i, nrsTpl+0x80*i, fmt.Sprintf("n%d.", i)
		g = append(g, nrsG(p+"alive", o+4, 1, false, 1, 4), nrsG(p+"tpl", o+6, 4, false, 1, 4),
			nrsG(p+"kind", o+0x2e, 1, false, 1, 4), nrsG(p+"gone", o+0xc7, 1, false, 1, 4),
			nrsG(p+"seen", o+0xd4, 1, false, 14, 1), nrsG(p+"pen", o+0xe8, 1, false, 1, 4),
			nrsG(p+"penAt", o+0xeb, 4, false, 1, 4),
			nrsG(fmt.Sprintf("t%d.class", i), t+0x58, 4, false, 1, 4),
			nrsG(fmt.Sprintf("t%d.regions", i), t+0x68, 4, false, 1, 4))
	}
	g = append(g, nrsG("pres", nrsPres, 1, false, 8, nrsPresS), nrsG("npcId", nrsPres+0xfa, 4, true, 2, nrsPresS),
		nrsG("penOn", nrsPres+0xfe, 1, false, 2, nrsPresS), nrsG("c75", nrsChars+0x75, 4, false, 8, nrsChS),
		nrsG("b01", nrsChars+0xb01, 4, false, 2, nrsChS), nrsG("b11", nrsChars+0xb11, 4, false, 2, nrsChS))
	for j := uint32(0); j < 8; j++ {
		g = append(g, nrsG(fmt.Sprintf("it%d_", j), nrsChars+nrsChS*j+0xf5d, 4, false, 12, 12),
			nrsG(fmt.Sprintf("eq%d_", j), nrsChars+nrsChS*j+0x1029, 4, false, 8, 12))
	}
	return g
}()

var nrsOutG = func() []n4Grp {
	g := []n4Grp{nrsG("pendA", 0x6875b6, 1, false, 1, 4), nrsG("pendB", 0x68759e, 1, false, 1, 4),
		nrsG("camLock", nrsLock, 1, false, 1, 4), nrsG("f2a6", nrsFacts+0x2a6, 1, false, 1, 4),
		nrsG("penOn", nrsPres+0xfe, 1, false, 2, nrsPresS)}
	for i := uint32(0); i < nrsArrN; i++ {
		o, p := nrsObj+0x100*i, fmt.Sprintf("n%d.", i)
		g = append(g, nrsG(p+"seen", o+0xd4, 1, false, 14, 1), nrsG(p+"pen", o+0xe8, 1, false, 1, 4),
			nrsG(p+"penAt", o+0xeb, 4, false, 1, 4))
	}
	g = append(g, nrsG("evcnt", nrsEvl+4, 4, true, 1, 4), nrsG("evcap", nrsEvl+8, 4, true, 1, 4),
		nrsG("evbuf", nrsEvl+0xc, 4, false, 1, 4), nrsG("ev", nrsEvBuf, 4, false, 3*nrsFixN, 4),
		nrsG("qcnt", nrsQCnt, 4, true, 1, 4), nrsG("qcap", nrsQCnt+4, 4, true, 1, 4),
		nrsG("qbuf", nrsQCnt+8, 4, false, 1, 4), nrsG("q", nrsQBuf, 4, false, nrsFixN, 4),
		nrsG("heap", nrsHeap, 4, false, nrsHeapW/4, 4))
	for _, s := range nrsStubs {
		g = append(g, nrsG(fmt.Sprintf("n%x", s.addr), s.at, 4, false, 1, 4))
		for a := 1; a <= s.nargs; a++ {
			g = append(g, nrsG(fmt.Sprintf("a%x_%d_", s.addr, a), s.at+4+uint32(4*s.n*(a-1)), 4, false, s.n, 4))
		}
	}
	return g
}()

var nrsIns, nrsInNames = n4Expand(nrsInG)
var nrsOuts, nrsOutNames = n4Expand(nrsOutG)

type nrsC struct {
	m     n4Mem
	r     Rand
	real  bool
	calls map[uint32]int
	heap  uint32
}

func nrsFail(what string) { panic("npc region status: " + what) }

func npcRegionStatusNode(in []int64, r Rand) []int64 { return nrsRun(in, r, true) }

func init() {
	Nodes["0050ca80"] = Node{Addr: "0050ca80", In: nrsInNames, Out: nrsOutNames, F: npcRegionStatusNode,
		Emu: func(in []int64, r Rand) []int64 { return nrsRun(in, r, false) }}
}

func nrsRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(nrsIns) {
		nrsFail("wrong input count")
	}
	c := &nrsC{m: n4Mem{}, r: r, real: real, calls: map[uint32]int{}}
	m := c.m
	// case setup: record pointers, fixture buffers, debug/UI switches 0, decoy records
	for _, p := range [][2]uint32{{0x685174, nrsChars}, {0x685178, nrsPres}, {nrsTab + 0xc, nrsArr},
		{0x68de40, nrsEvl}, {nrsEvl + 0xc, nrsEvBuf}, {nrsQCnt + 8, nrsQBuf}, {0x6836ac, nrsFactTab}} {
		m.w32(p[0], p[1])
	}
	m.w8(0x688290, 0)
	for _, a := range []uint32{0x683f8c, 0x68edcc, 0x68de44, 0x6850d5} {
		m.w32(a, 0)
	}
	for j := uint32(2); j < 8; j++ {
		m.w32(nrsPres+nrsPresS*j+0xfa, 3)
		m.w8(nrsPres+nrsPresS*j+0xfe, 7)
		m.w32(nrsChars+nrsChS*j+0xb01, 0x10)
		m.w32(nrsChars+nrsChS*j+0xb11, 5)
	}
	for i, f := range nrsIns {
		m.put(f.g.addr+uint32(f.k*f.g.step), f.g.size, uint32(in[i]))
	}
	c.node()
	out := make([]int64, len(nrsOuts))
	for i, f := range nrsOuts {
		v := m.get(f.g.addr+uint32(f.k*f.g.step), f.g.size)
		out[i] = int64(v)
		if f.g.signed {
			out[i] = int64(int32(v))
		}
	}
	return out
}

// call records an address stub (count at `at`, argument a of call k at at+4+4n(a-1)+4k);
// F also runs the real 0x522fe0.
func (c *nrsC) call(addr uint32, args ...uint32) {
	for _, s := range nrsStubs {
		if s.addr != addr {
			continue
		}
		k := c.calls[addr]
		c.calls[addr] = k + 1
		if k >= s.n {
			nrsFail(fmt.Sprintf("stub %x record sequence full", addr))
		}
		for a := 1; a <= s.nargs; a++ {
			c.m.w32(s.at+4+uint32(4*s.n*(a-1))+uint32(4*k), args[a-1])
		}
		c.m.w32(s.at, c.m.r32(s.at)+1)
		if c.real && addr == 0x522fe0 {
			(&ctrsC{m: c.m, r: c.r, who: args[0]}).realCall(addr, args)
		}
		return
	}
	nrsFail(fmt.Sprintf("no stub %x", addr))
}

func (c *nrsC) need(p uint32, what string) uint32 {
	if p == 0 {
		nrsFail(what + " is null (the original faults)")
	}
	return p
}

// key is 0x42b740(*0x686a70), sign-extended by the caller.
func (c *nrsC) key() int32 { return int32(int8((&ctrsC{m: c.m}).selKey())) }

// row finds a key in the region table (0x50cbb4, 0x50ce8e, 0x50c9e0).
func (c *nrsC) row(key int32) (uint32, bool) {
	for i := uint32(0); ; i++ {
		k := int32(c.m.r32(nrsRegions + 4 + 12*i))
		if k == -1 {
			return 0, false
		}
		if k == key {
			return i, true
		}
	}
}

// elem is 0x5c3790(id) on table T: the element slot, element 0 for id >= count.
func (c *nrsC) elem(id int32) uint32 {
	t := c.m.r32(0x689f94)
	arr := c.m.r32(t + 0xc)
	i := int32(0)
	if id < int32(c.m.r32(t+4)) {
		i = id
	}
	if i < 0 || i >= nrsArrN || arr != nrsArr {
		nrsFail("NPC element outside the fixture (negative id or index past 4)")
	}
	return arr + 4*uint32(i)
}

// npc is 0x50b800(id): 0 for a null table or a flagged (+0xc7) NPC.
func (c *nrsC) npc(id int32) uint32 {
	if c.m.r32(0x689f94) == 0 {
		return 0
	}
	e := c.m.r32(c.elem(id))
	if e != 0 && c.m.r8(e+0xc7) != 0 {
		return 0
	}
	return e
}

// regionHit is the template region mask test of 0x50c9e0 / 0x50cea0.
func (c *nrsC) regionHit(e, r uint32) bool {
	return c.m.r32(c.m.r32(e+6)+0x68)&c.m.r32(nrsRegions+8+12*r) != 0
}

// party is 0x50b8f0(kind): an up NPC of that kind in slot 0 or 1 (state below 15).
func (c *nrsC) party(kind uint32) bool {
	m := c.m
	for j := uint32(0); j < 2; j++ {
		p := m.r32(0x685178) + nrsPresS*j
		if m.r8(p) == 0 {
			continue
		}
		e := c.need(c.npc(int32(m.r32(p+0xfa))), "0x50b8f0 NPC")
		if m.r8(e+0x2e) == kind && m.r32(m.r32(0x685174)+nrsChS*j+0xb01) < 0xf {
			return true
		}
	}
	return false
}

func (c *nrsC) voice(ch, v uint32) {
	c.call(0x52e690, ch, v, 0, c.m.r32(0x5ed8c8), c.m.r32(0x5ed914))
}

// alloc is the emulator bump heap (malloc / 0x5e1ce0): rounded to 8, min 8, zeroed.
func (c *nrsC) alloc(n uint32) uint32 {
	a := nrsHeap + c.heap
	step := (n + 7) &^ 7
	if step < 8 {
		step = 8
	}
	c.heap += step
	if c.heap > nrsHeapW {
		nrsFail("heap above the output window")
	}
	return a
}

// grow appends a `size`-byte entry to a (count, cap, buffer) table, reallocating to
// count+1 entries when count+1 > cap (signed): 0x5289b0, 0x5588f0.
func (c *nrsC) grow(cntA, size, fixture uint32) uint32 {
	m := c.m
	cnt := int32(m.r32(cntA))
	if cnt < 0 {
		nrsFail("negative table count")
	}
	if cnt+1 > int32(m.r32(cntA+4)) {
		old := m.r32(cntA + 8)
		nb := c.alloc(uint32(cnt+1) * size)
		m.copy(nb, old, int(uint32(cnt)*size))
		m.w32(cntA+8, nb)
		m.w32(cntA+4, uint32(cnt+1))
	}
	buf := m.r32(cntA + 8)
	if buf == fixture && cnt >= nrsFixN {
		nrsFail("table write past its fixture buffer")
	}
	m.w32(cntA, uint32(cnt+1))
	return buf + uint32(cnt)*size
}

// message is 0x5289b0(kind, x).
func (c *nrsC) message(kind, x uint32) {
	m := c.m
	o := c.alloc(0x24)
	m.w32(o, 0xffffffff)
	m.w32(o+0xc, kind)
	m.w32(o+0x10, x)
	m.w32(o+0x20, m.r32(0x68c4ac))
	m.w32(c.grow(nrsQCnt, 4, nrsQBuf), o)
}

// set is 0x5061a0(id, val, 0): fact byte, event log entry 0x5588f0 on a change, then
// 0x506670(id, val).
func (c *nrsC) set(id, val uint32) {
	m := c.m
	if m.r8(0x688290) != 0 || m.r32(0x683f8c) > id || m.r32(0x68edcc) != 0 {
		nrsFail("debug/UI switch set")
	}
	old := m.r8(nrsFacts + id)
	m.w8(nrsFacts+id, val)
	if old != val {
		e := c.grow(m.r32(0x68de40)+4, 12, nrsEvBuf)
		m.w32(e, m.r32(0x686a70))
		m.w32(e+4, id)
		m.w32(e+8, m.r8(nrsFacts+id))
	}
	c.call(0x506670, id, val)
}

func (c *nrsC) node() {
	m := c.m
	if m.r8(0x683f94) != 0 || m.r8(0x683fc5) != 0 {
		return
	}
	if m.r8(0x6875b6) != 0 {
		c.partA()
	}
	if m.r8(0x68759e) == 0 || int32(m.r32(0x686a48)-m.r32(0x68759a)) <= 0x3c {
		return
	}
	r, ok := c.row(c.key())
	if !ok {
		m.w8(0x68759e, 0)
		return
	}
	v := m.r8(nrsRegions + 12*r)
	keep := false
	for k := uint32(0); k < 2; k++ {
		p := m.r32(0x685178) + nrsPresS*k
		ch := m.r32(0x685174) + nrsChS*k
		if m.r8(p) == 0 || m.r32(ch+0xb11) == 0 {
			continue
		}
		id := int32(m.r32(p + 0xfa))
		obj := c.npc(id)
		st := m.r32(ch + 0xb01)
		if st >= 0xf {
			if st <= 0x11 {
				keep = true
				m.w8(c.need(obj, "NPC")+0xe8, 1)
			}
			continue
		}
		if c.key()&0xff != 0xd {
			kind := m.r8(c.need(obj, "NPC") + 0x2e)
			line := uint32(0)
			if kind == 0x11 && c.noUp10() {
				line = 0x6c
			} else if kind == 0x10 && !c.party(0x11) {
				line = 0x67
			}
			if line != 0 {
				c.voice(ch, line)
				m.w8(nrsLock, 1) // 0x577520
				c.call(0x41f0d0)
				c.call(0x55ee70, 9)
				c.message(7, k)
				continue
			}
		}
		if m.r32(0x689f94) != 0 {
			e := m.r32(c.elem(id))
			if e != 0 && m.r8(e+0xc7) == 0 && m.r8(e+0xd4+r) == 0 {
				m.w8(e+0xd4+r, 1)
				hit := false
				if r2, ok := c.row(c.key()); ok {
					hit = c.regionHit(e, r2)
				}
				if !hit && r != 0xc && r != 0xd {
					c.voice(ch, v)
				}
			}
			e = m.r32(c.elem(id))
			if e != 0 && m.r8(e+0xc7) == 0 {
				r3, ok := c.row(c.key())
				if !ok {
					nrsFail("region key lost between two lookups (block 50ce9e)")
				}
				if c.regionHit(e, r3) {
					m.w8(p+0xfe, 1)
					m.w32(e+0xeb, m.r32(0x686a48))
					c.call(0x50e650, k)
					c.voice(ch, 0x56)
				}
			}
		}
		m.w8(c.need(obj, "NPC")+0xe8, 0)
	}
	if !keep {
		m.w8(0x68759e, 0)
	}
}

// noUp10: neither NPC slot holds an up (state below 15) NPC of kind 0x10 (0x50ccbb..).
func (c *nrsC) noUp10() bool {
	m := c.m
	p0, ch0 := m.r32(0x685178), m.r32(0x685174)
	for j := uint32(0); j < 2; j++ {
		if m.r8(p0+nrsPresS*j) == 0 {
			continue
		}
		e := c.need(c.npc(int32(m.r32(p0+nrsPresS*j+0xfa))), "0x50b800 NPC")
		if m.r8(e+0x2e) == 0x10 && m.r32(ch0+nrsChS*j+0xb01) < 0xf {
			return false
		}
	}
	return true
}

func (c *nrsC) partA() {
	m := c.m
	t := c.need(m.r32(0x689f94), "NPC table")
	n := int32(m.r32(t + 4))
	for i := int32(0); i < n; i++ {
		e := c.need(m.r32(c.elem(i)), "NPC")
		if m.r32(m.r32(e+6)+0x58) == 0x42 {
			if m.r8(e+4) != 0 {
				m.w8(0x6875b6, 0)
				return
			}
			break
		}
	}
	if k := c.key(); k == 9 || k == 10 {
		return
	}
	slot := m.r32(0x6875ef)
	if slot >= 8 {
		nrsFail("party slot outside 0..7")
	}
	ch := m.r32(0x685174) + nrsChS*slot
	if m.r32(ch+0x75) == 0 {
		c.voice(ch, m.r32(0x5ee638))
	}
	c.call(0x522fe0, slot, 0xa, 0x270f, 0, 0, 1)
	m.w8(0x6875b6, 0)
	c.set(0x2a6, 1)
}
