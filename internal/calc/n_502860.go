package calc

import "fmt"

// camping_timed_status_release 0x502860 (build/wiz8.asm:348316..348399), no arguments.
//
// Byte 0x683fc5 := 0, then 0x5029e0 (post_rest_status_duration) and the UI helpers
// 0x4f21b0(0x137), 0x4f23d0(0x137). One rest notice goes to 0x58ac00(0xc, text, -1,
// -1, 0); text is a pointer from the text table *0x68c09c:
//   - byte 0x683fc6 set: it is cleared; +0x1e54 when 0x4e7ca0 holds (some member
//     0..7 passes 0x524a10, characterActionEligibilityNode), else +0x1e48;
//   - else dword 0x683fc7 == 0: +0x1e4c;
//   - else clock 0x686a48 > 0x683fc7 (unsigned): +0x1e44, else +0x1e48.
//
// Release: byte 0x6875f7 non-zero and u32(clock - that byte) > 0x15180 (0x502923)
// clears 0x6875f7, sets byte 0x6875f8, calls 0x523330(who, 0x13, 0) with who the
// dword 0x6875ff, 0x52e690(*0x685174 + 0x1862*who (u32), *0x5ee658, 0, *0x5ed8c8,
// *0x5ed914) and the story flag set 0x5061a0(0xb6, 1, 0).
//
// 0x5061a0 (run natively): fact byte 0x689b78+0xb6 := 1; when the old byte differs,
// 0x5588f0 appends (map 0x686a70, 0xb6, fact) to the event log *0x68de40 (count +4,
// capacity +8, 12-byte entries at +0xc; count+1 > capacity reallocates to count+1
// entries and copies; a null log is first built by 0x558820: 0x10-byte object
// {0x5ee8a0, 0, 5, new(0x3c)}). The entry's third dword is the fact byte over a stack
// slot the case leaves zero. Then 0x506670(0xb6, 1). The debug text (0x683f8c,
// 0x688290) is off and the notice part of 0x5588f0 ends at the null battle object
// 0x68edcc, as in n_4e46f0.go. Allocations follow the emulator's bump heap at
// 0x50000000 (8-byte steps); the allocator never fails there.
//
// Address stubs: D4 0x5029e0, 0x523330, 0x506670 (returns from case inputs r*; no
// one reads them), 排除 0x4f21b0, 0x4f23d0, 0x58ac00, 0x52e690. Counts and arguments
// are outputs. Emu replays the stubs. F runs the real 0x5029e0 (prsRun) first on the
// node's presence, +0xb01, +0xb11 and the F-only inputs fa52, fa161_*, fs*, fb09_*,
// fb0d_*, ft* (as in n_5029e0.go; mode, battle object and widgets zero) and feeds
// the new +0xb01 / +0xb11 into 0x4e7ca0. Nothing after 0x523330 or 0x506670 reads
// their writes, so F does not run them.
//
// Fixture caps (calc panics past them): event log pointer 0 or the fixture, its
// buffer the fixture of 8 entries or the heap, heap window 28 dwords.
const (
	ctsrM      = 8
	ctsrArena  = 0x20000000
	ctsrChars  = ctsrArena
	ctsrEvl    = ctsrArena + 0x13000
	ctsrEvbuf  = ctsrArena + 0x13100
	ctsrNEv    = 8
	ctsrHeap   = 0x50000000
	ctsrHeapW  = 28
	ctsrFact   = 0xb6    // 0x502985
	ctsrWindow = 0x15180 // 0x502923, one day in seconds
)

func ctsrNames() (in, out []string) {
	add := func(l *[]string, f string, a ...any) { *l = append(*l, fmt.Sprintf(f, a...)) }
	in = []string{"fc5", "fc6", "fc7", "clock", "f7", "f8", "who", "t1e44", "t1e48", "t1e4c", "t1e54", "g658", "g8c8", "g914"}
	for _, p := range []string{"pres", "hp", "b01_"} {
		for m := 0; m < ctsrM; m++ {
			add(&in, "%s%d", p, m)
		}
	}
	in = append(in, "fact", "map", "evlPtr", "evCnt", "evCap")
	for k := 0; k < ctsrNEv; k++ {
		for j := 0; j < 3; j++ {
			add(&in, "ev%d_%d", k, j)
		}
	}
	in = append(in, "r5029e0", "r523330", "r506670", "fa52")
	for m := 0; m < ctsrM; m++ {
		add(&in, "fa161_%d", m)
	}
	for m := 0; m < ctsrM; m++ {
		for k := 0; k < ccaSlots; k++ {
			add(&in, "fs%d_%d", m, k)
		}
	}
	for m := 0; m < ctsrM; m++ {
		add(&in, "fb09_%d", m)
		add(&in, "fb0d_%d", m)
	}
	for k := 0; k < ccaWLen; k++ {
		add(&in, "ft%d", k)
	}
	out = []string{"ofc5", "ofc6", "of7", "of8", "ofact", "oevlPtr"}
	for _, s := range ctsrStubs {
		add(&out, "n%x", s.addr)
		for a := 1; a <= s.args; a++ {
			add(&out, "a%x_%d", s.addr, a)
		}
	}
	out = append(out, "oevCnt", "oevCap", "oevBuf")
	for k := 0; k < ctsrNEv; k++ {
		for j := 0; j < 3; j++ {
			add(&out, "oev%d_%d", k, j)
		}
	}
	for i := 0; i < ctsrHeapW; i++ {
		add(&out, "heap%d", i)
	}
	return
}

// ctsrStubs lists the address stubs in output order.
var ctsrStubs = []struct{ addr, args int }{
	{0x5029e0, 0}, {0x4f21b0, 1}, {0x4f23d0, 1}, {0x58ac00, 5}, {0x523330, 3}, {0x52e690, 5}, {0x506670, 2}}

func init() {
	in, out := ctsrNames()
	Nodes["00502860"] = Node{Addr: "00502860", In: in, Out: out,
		F:   campingTimedStatusReleaseNode,
		Emu: func(in []int64, r Rand) []int64 { return campingTimedStatusRelease(in, r, false) }}
}

func campingTimedStatusReleaseNode(in []int64, r Rand) []int64 {
	return campingTimedStatusRelease(in, r, true)
}

// ctsrMem is the event log's reachable memory: the fixture log, its 8-entry buffer
// and the heap window.
type ctsrMem struct {
	evl  [3]uint32 // +4 count, +8 capacity, +0xc buffer
	ev   [ctsrNEv * 3]uint32
	heap [ctsrHeapW]uint32
	top  uint32
}

func (c *ctsrMem) cell(a uint32) *uint32 {
	switch {
	case a >= ctsrEvl+4 && a < ctsrEvl+16 && a%4 == 0:
		return &c.evl[(a-ctsrEvl-4)/4]
	case a >= ctsrEvbuf && a < ctsrEvbuf+12*ctsrNEv && a%4 == 0:
		return &c.ev[(a-ctsrEvbuf)/4]
	case a >= ctsrHeap && a < ctsrHeap+4*ctsrHeapW && a%4 == 0:
		return &c.heap[(a-ctsrHeap)/4]
	}
	panic(fmt.Sprintf("camping_timed_status_release: event log access %#x outside the fixture", a))
}

func (c *ctsrMem) alloc(n uint32) uint32 {
	a := ctsrHeap + c.top
	step := (n + 7) &^ 7
	if step < 8 {
		step = 8
	}
	if step > 4*ctsrHeapW-c.top {
		panic("camping_timed_status_release: heap above the output window")
	}
	c.top += step
	return a
}

func campingTimedStatusRelease(in []int64, r Rand, real bool) []int64 {
	names, _ := ctsrNames()
	if len(in) != len(names) {
		panic("camping_timed_status_release: wrong input count")
	}
	ix := make(map[string]int, len(names))
	for i, n := range names {
		ix[n] = i
	}
	u := func(n string) uint32 { return uint32(in[ix[n]]) }
	b := func(n string) uint32 { return uint32(in[ix[n]]) & 0xff }
	stub := map[int][]uint32{}
	call := func(addr int, a ...uint32) {
		if _, ok := stub[addr]; ok {
			panic("camping_timed_status_release: stub called twice")
		}
		stub[addr] = a
	}

	fc6, f7, f8, fact := b("fc6"), b("f7"), b("f8"), b("fact")
	call(0x5029e0)
	pres, hp, b01 := make([]uint32, ctsrM), make([]uint32, ctsrM), make([]uint32, ctsrM)
	for m := 0; m < ctsrM; m++ {
		pres[m], hp[m], b01[m] = b(fmt.Sprint("pres", m)), u(fmt.Sprint("hp", m)), u(fmt.Sprint("b01_", m))
	}
	if real {
		ctsrRest(in, ix, pres, hp, b01, r)
	}
	call(0x4f21b0, 0x137)
	call(0x4f23d0, 0x137)

	var text uint32
	switch fc7, clock := u("fc7"), u("clock"); {
	case fc6 != 0:
		fc6 = 0
		text = u("t1e48")
		for m := 0; m < ctsrM; m++ { // 0x4e7ca0
			if characterActionEligibilityNode([]int64{int64(m), int64(pres[m]), int64(hp[m]), int64(b01[m])}, r)[0] != 0 {
				text = u("t1e54")
				break
			}
		}
	case fc7 == 0:
		text = u("t1e4c")
	case clock > fc7:
		text = u("t1e44")
	default:
		text = u("t1e48")
	}
	call(0x58ac00, 0xc, text, 0xffffffff, 0xffffffff, 0)

	mem := &ctsrMem{evl: [3]uint32{u("evCnt"), u("evCap"), ctsrEvbuf}}
	for k := range mem.ev {
		mem.ev[k] = u(fmt.Sprintf("ev%d_%d", k/3, k%3))
	}
	evl := u("evlPtr")
	if evl != 0 && evl != ctsrEvl {
		panic("camping_timed_status_release: event log pointer outside the fixture")
	}
	if f7 != 0 && u("clock")-f7 > ctsrWindow {
		f7, f8 = 0, 1
		who := u("who")
		call(0x523330, who, 0x13, 0)
		call(0x52e690, ctsrChars+0x1862*who, u("g658"), 0, u("g8c8"), u("g914"))
		// 0x5061a0(0xb6, 1, 0)
		old := fact
		fact = 1
		if old != fact {
			evl = ctsrEvent(mem, evl, u("map"), fact)
		}
		call(0x506670, ctsrFact, 1)
	}

	o := []int64{0, int64(fc6), int64(f7), int64(f8), int64(fact), int64(evl)}
	for _, s := range ctsrStubs {
		a, ok := stub[s.addr]
		if ok {
			o = append(o, 1)
		} else {
			o = append(o, 0)
		}
		for k := 0; k < s.args; k++ {
			if ok {
				o = append(o, int64(a[k]))
			} else {
				o = append(o, 0)
			}
		}
	}
	for _, v := range mem.evl {
		o = append(o, int64(v))
	}
	for _, v := range mem.ev {
		o = append(o, int64(v))
	}
	for _, v := range mem.heap {
		o = append(o, int64(v))
	}
	return o
}

// ctsrEvent is 0x5588f0(0xb6): the log entry (map, id, fact); returns *0x68de40.
func ctsrEvent(c *ctsrMem, evl, mapID, fact uint32) uint32 {
	if evl == 0 { // 0x558820
		evl = c.alloc(0x10)
		d := c.alloc(0x3c)
		*c.cell(evl) = 0x5ee8a0
		*c.cell(evl + 0xc) = d
		*c.cell(evl + 4) = 0
		*c.cell(evl + 8) = 5
	}
	cnt := int32(*c.cell(evl + 4))
	need := cnt + 1
	if need > int32(*c.cell(evl + 8)) {
		old := *c.cell(evl + 0xc)
		nb := c.alloc(uint32(need) * 12)
		*c.cell(evl + 0xc) = nb
		*c.cell(evl + 8) = uint32(need)
		for i := int32(0); i < int32(*c.cell(evl + 4)); i++ {
			for w := uint32(0); w < 3; w++ {
				*c.cell(nb + 12*uint32(i) + 4*w) = *c.cell(old + 12*uint32(i) + 4*w)
			}
		}
	}
	e := *c.cell(evl + 0xc) + 12*uint32(cnt)
	*c.cell(e) = mapID
	*c.cell(e + 4) = ctsrFact
	*c.cell(e + 8) = fact
	*c.cell(evl + 4) = uint32(cnt + 1)
	return evl
}

// ctsrRest runs the real 0x5029e0 (F only) and leaves every member's +0xb01 and
// +0xb11 in b01 and hp.
func ctsrRest(in []int64, ix map[string]int, pres, hp, b01 []uint32, r Rand) {
	pn, _ := prsNames()
	v := make([]int64, len(pn))
	for i, n := range pn {
		var src string
		switch {
		case n == "partyPtr":
			v[i] = prsParty
			continue
		case n == "charBase":
			v[i] = prsChars
			continue
		case n == "a52":
			src = "fa52"
		case len(n) > 5 && n[:5] == "a161_":
			src = "f" + n
		case len(n) > 5 && n[:5] == "fb11_":
			src = "hp" + n[5:]
		case n[:2] == "ft" || n[:2] == "fs" || len(n) > 4 && (n[:4] == "fb09" || n[:4] == "fb0d"),
			len(n) > 4 && n[:4] == "pres", len(n) > 4 && n[:4] == "b01_":
			src = n
		default:
			continue // mode, battle object, widgets, portrait fields: zero
		}
		j, ok := ix[src]
		if !ok {
			panic("camping_timed_status_release: no F input for " + n)
		}
		v[i] = in[j]
	}
	_, nb01, w := prsRun(v, r, true)
	for m := 0; m < ctsrM; m++ {
		b01[m], hp[m] = uint32(nb01[m]), uint32(w.b11[m])
	}
}
