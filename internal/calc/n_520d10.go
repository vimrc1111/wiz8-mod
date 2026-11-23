package calc

import "fmt"

// Cycle group 0x520d10 / 0x4e8000 / 0x536570 / 0x53c990. The four bodies call
// each other (520d10 -> 4e82e0 -> 4e8000 -> 536570 -> 53c990 -> 520d10), so one
// sparse byte world (cgW) holds the fixture memory and every member is a
// transliteration over it. In = the catalogued memory cells (cgIn, re/emu/cases)
// followed by the member's arguments; Out = every catalogued cell afterwards
// (cgOut) plus the member's return value. Address-stubbed callees (cgStubs) read
// their return from an input cell and record their arguments and a call count in
// the stub object (decisions/2026-10-09-compositional-stubs.md, 2026-10-09
// TargetCone decision for 0x4add30).
//
// Fixture caps (calc panics past them): four party slots, two monsters, empty
// 0x683fb1/0x683fb9/0x683fbd group tables, no sprite creation behind 0x4c5eb0.
// A coordinate target (0x519180 record kind 6) runs natively over the camera
// and spell cells; its map-ray query 0x434f20 is address-stubbed (the cell it
// returns is the only input, so its internal block rolls are not run).
//
// Phase / draw caps (0x53a320 -> 0x56aab0 / 0x56aa30): the registered exclusions
// 0x55ee70 (cursor icon), 0x420b40, 0x4f21b0, 0x4f23d0, 0x5a1950, 0x4263f0 and
// 0x422d50 are address stubs that record arguments and call counts (every case of
// the group carries them in Out). The clock 0x482990 and the mover helpers 0x4531a0
// and 0x41f160 run natively over the cells; the fixture tick is fixed
// (case "tick"), so the clock delta is 0 and the time-of-day update 0x482a20 is
// never reached (its day-counter write path is not modelled: 0x482a20 is ~300
// instructions of sun-light vector math with 75 FPU ops and five unregistered
// callees 0x41e880 / 0x421680 / 0x4218e0 / 0x4848d0 / 0x4b8850). 0x5a19a0, 0x4f2380
// and 0x4f2040 (phase 4) are registered exclusions and address-stubbed, so phase 4
// with 0x683fcd != 0, or 0x683f94 == 0 with 0x6840bd != 0, runs. Mode 7 with a
// pending-source kind (ui+0x2f0) that is not -1 and not 7/8 with its flag resolves
// to the NULL record of 0x53b7f0, which 0x5387f0 dereferences; the generator keeps
// ui+0x2f0 == -1 then.
// 0x538c00 (store, old record kind 4) never reaches its 0x4c5eb0 call at 0x538cd3
// here: the group tables 0x683fb1/0x683fbd are empty, so it returns -1 first.

const cgA = 0x20000000

type cgCell struct {
	a int64
	s int // size; negative = signed
}
type cgInitCell struct {
	a int64
	s int
	v int64
}
type cgW struct {
	m   map[int64]byte
	seq []int64 // when set: the per-call returns of 0x4e79a0 (case "ret":{at,n}), consumed in call order
}

func (w *cgW) rd(a int64, n int) int64 {
	if a < 0x10000 {
		panic(fmt.Sprintf("cg: read of unmapped address %#x", a))
	}
	var v uint64
	for i := n - 1; i >= 0; i-- {
		v = v<<8 | uint64(w.m[a+int64(i)])
	}
	return int64(v)
}
func (w *cgW) rs(a int64, n int) int64 {
	v := w.rd(a, n)
	sh := uint(64 - 8*n)
	return v << sh >> sh
}
func (w *cgW) wr(a int64, n int, v int64) {
	if a < 0x10000 {
		panic(fmt.Sprintf("cg: write of unmapped address %#x", a))
	}
	for i := 0; i < n; i++ {
		w.m[a+int64(i)] = byte(v >> (8 * uint(i)))
	}
}
func (w *cgW) u8(a int64) int64  { return w.rd(a, 1) }
func (w *cgW) u32(a int64) int64 { return w.rd(a, 4) }
func (w *cgW) i32(a int64) int64 { return w.rs(a, 4) }
func (w *cgW) p32(a, v int64)    { w.wr(a, 4, v) }
func (w *cgW) p8(a, v int64)     { w.wr(a, 1, v) }

// cgRec is an 8-dword action record.
type cgRec [8]int64

func (w *cgW) rec(a int64) (r cgRec) {
	for i := range r {
		r[i] = w.i32(a + 4*int64(i))
	}
	return
}
func (w *cgW) putRec(a int64, r cgRec) {
	for i, v := range r {
		w.p32(a+4*int64(i), v)
	}
}

// Stub indices: the order of the case file's stub table (spec STUBS).
const (
	cgS4e79a0 = iota
	cgS536f60
	cgS4f0480
	cgS4ee9d0
	cgS5458a0
	cgS5201b0
	cgS4ee220
	cgS50e5c0
	cgS51d180
	cgS51d3b0
	cgS53c720
	cgS536a20
	cgS536190
	cgS537330
	cgS4fd030
	cgS4add30
	cgS593330
	cgS595600
	cgS5194e0
	cgS58aad0
	cgS434f20
	cgS5199f0
	cgS51b000
	cgS4c5eb0
	cgS55ee70 // 55ee70 and the six below: registered exclusions of the phase and 56aab0 draw paths
	cgS420b40
	cgS4f21b0
	cgS4f23d0
	cgS5a1950
	cgS4263f0
	cgS422d50
	cgS5a19a0
	cgS4f2380
	cgS4f2040
)

// st is one stubbed call: count the call, record the arguments the case records
// (0x4fd030 records arguments 1,2,5,7, 0x58aad0 arguments 1-3 and 0x434f20 arguments 3-7; the rest are stack pointers), and
// return the signed dword read from the stub's return cell.
func (w *cgW) st(i int, args ...int64) int64 {
	const base = cgA + 0x17000
	ci := base + 0x800 + 4*int64(i)
	recAt := base + 0x200 + 0x40*int64(i)
	if i > cgS4c5eb0 { // the 0x55ee70.. block: counts at 0x880, records at 0xb00 (the 0x200 table would run into the counts; 0xa00-0xac0 is the 0x4e79a0 return sequence of 0x53bf80)
		j := int64(i - cgS55ee70)
		ci, recAt = base+0x880+4*j, base+0xb00+0x40*j
	}
	w.p32(ci, w.u32(ci)+1)
	rec := []int{}
	if i == cgS4fd030 {
		rec = []int{1, 2, 5, 7}
	} else if i == cgS58aad0 {
		rec = []int{1, 2, 3}
	} else if i == cgS434f20 { // arguments 1,2 are stack pointers
		rec = []int{3, 4, 5, 6, 7}
	} else {
		for k := range args {
			rec = append(rec, k+1)
		}
	}
	for _, k := range rec {
		w.p32(recAt+4*int64(k-1), args[k-1])
	}
	if i == cgS4e79a0 && w.seq != nil {
		k := int64(w.u32(ci)) - 1
		if k >= int64(len(w.seq)) {
			panic("cg: 0x4e79a0 return sequence exhausted")
		}
		return int64(int32(w.seq[k]))
	}
	return w.i32(base + 0x100 + 4*int64(i))
}

// notify is 0x590950(x, fmt): the message formatter. Its one call into the
// excluded 0x58aad0 is recorded (arguments 1-3; 0x64bf5c is the format pointer
// pushed at 0x5909d7); the text itself has no rule effect.
func (w *cgW) notify(x int64) { w.st(cgS58aad0, 8, 0x64bf5c, w.C(x)+5) }

func (w *cgW) who(x int64) int64 {
	if x < 0 || x >= 4 {
		panic("cg: party slot outside the four-slot fixture")
	}
	return x
}
func (w *cgW) S(x int64) int64      { return w.u32(0x685178) + 0x106*w.who(x) }
func (w *cgW) C(x int64) int64      { return w.u32(0x685174) + 0x1862*w.who(x) }
func (w *cgW) D(x int64) int64      { return w.u32(0x6836a8) + 0xd4*w.who(x) }
func (w *cgW) mode() int64          { return w.i32(0x68ec78) }
func (w *cgW) uip() int64           { return w.u32(0x68edcc) }
func (w *cgW) sel() int64           { return w.i32(0x68518d) }
func (w *cgW) f94() bool            { return w.u8(0x683f94) != 0 }
func (w *cgW) selFlags() bool       { return w.u8(0x683f95) != 0 || w.u8(0x683f96) != 0 }
func (w *cgW) pcAddr(x int64) int64 { return 0x683794 + 0x118*x }

// resolve is 0x53bc10: the attack context 6 resolved to a source 0..4 or 7.
func (w *cgW) resolve(x int64) int64 {
	if u := w.uip(); w.mode() == 7 && u != 0 && w.i32(u+0x2f0) != -1 {
		a, b := w.i32(u+0x2f0), w.u8(u+0x2f8) != 0
		if a == 7 && b {
			return 3
		}
		if a == 8 && b {
			return 4
		}
		return 7
	}
	if x == w.sel() && w.selFlags() {
		return 2
	}
	if w.f94() {
		return 1
	}
	return 0
}

// kind is 0x53b920: the source index, asserting on 6 (after resolution) and >7.
func (w *cgW) kind(x, m int64) int64 {
	if m == 6 {
		m = w.resolve(x)
	}
	if m < 0 || m == 6 || m > 7 {
		panic("cg: 0x53b920 assert")
	}
	return m
}

// rptr is 0x53b7f0: the action record of source m; 7 has none (NULL).
func (w *cgW) rptr(x, m int64) int64 {
	if m == 6 {
		m = w.resolve(x)
	}
	S := w.S(x)
	switch m {
	case 0:
		return S + 0x1d
	case 1:
		return S + 0x4d
	case 2:
		return 0x68408b
	case 3:
		return S + 0x81
	case 4:
		return S + 0xa9
	case 5:
		return S + 0xd1
	case 7:
		return 0
	}
	panic("cg: 0x53b7f0 assert")
}

// targetable is 0x53c270.
func (w *cgW) targetable(x int64) bool {
	c := w.C(x)
	return w.u8(w.S(x)) != 0 && w.i32(c+0xb11) != 0 && w.u32(c+0xb01) < 0x12
}

// act is 0x4e77b0: action kind a, spell/id b, record c and extra d of source k.
func (w *cgW) act(x, k int64) (a, b, c, d int64) {
	if k == 6 {
		k = w.resolve(x)
	}
	if k < 0 || k > 8 {
		panic("cg: 0x4e77b0 assert")
	}
	S := w.S(x)
	switch k {
	case 0:
		a, b, c, d = w.i32(S+1), w.i32(S+5), S+0x1d, S+0x15
	case 1:
		a, b, c, d = w.i32(S+0x3d), w.i32(S+0x41), S+0x4d, S+0x45
	case 2:
		if w.u8(0x683f95) != 0 {
			b = w.i32(w.u32(0x69bf3c) + 0x104)
			if b == 0 {
				b = w.i32(w.u32(0x69bf3c) + 0x100)
			}
			a, c, d = 7, 0x68408b, 0x6840ab
		} else if w.u8(0x683f96) != 0 {
			v := w.i32(0x69b9a4)
			if v == 0 {
				v = w.i32(0x69b9a0)
			}
			w.p32(0x6840af, v)
			a, b, c, d = 8, -1, 0x68408b, 0x6840ab
		} else {
			panic("cg: 0x4e77b0 assert")
		}
	case 3:
		a, b, c, d = 7, w.i32(S+0x75), S+0x81, S+0x79
	case 4:
		a, b, c, d = 8, -1, S+0xa9, S+0xa1
	case 5:
		a, b, c, d = 2, -1, S+0xd1, 0
	case 7:
		u := w.uip()
		a, b = w.i32(u+0x2f0), w.i32(u+0x2f4)
	case 8:
		a, b = 0, -1
	}
	if !w.targetable(x) && a != 0xa && a != 0xb {
		a, b, c, d = -1, -1, 0, 0
	}
	return
}

// Monster list (0x683fad): 0x4e5620 entry i, 0x4e5550 id search.
func (w *cgW) entry(i int64) int64 {
	v := w.u32(0x683fad)
	if v == 0 || i >= w.i32(v+8) {
		panic("cg: 0x4e5620 assert")
	}
	p := w.u32(w.u32(v) + 4*i)
	if p == 0 {
		panic("cg: 0x4e5620 NULL entry")
	}
	return p
}
func (w *cgW) find(id int64, assert bool) int64 {
	if v := w.u32(0x683fad); v != 0 {
		for i := int64(0); i < w.i32(v+8); i++ {
			if w.i32(w.entry(i)) == id {
				return i
			}
		}
	}
	if assert {
		panic("cg: 0x4e5550 assert")
	}
	return -1
}
func (w *cgW) monCount() int64 {
	if v := w.u32(0x683fad); v != 0 {
		return w.i32(v + 8)
	}
	return 0
}

// setMask is 0x539630: set or clear party member x's bit in monster id's mask.
func (w *cgW) setMask(x, id, flag int64) {
	e := w.find(id, false)
	if e == -1 {
		return
	}
	p := w.u32(w.entry(e) + 0xc)
	if p == 0 {
		panic("cg: 0x539630 NULL record")
	}
	a := w.u32(p+0x1d8) + 0x5bc
	if flag != 0 {
		w.p8(a, w.u8(a)|1<<uint(x))
	} else {
		w.p8(a, w.u8(a)&^(1<<uint(x)))
	}
	f := int64(0)
	if flag != 0 {
		f = 1
	}
	w.st(cgS4c5eb0, x, id, f) // 0x5396a3, 0x5396bf: scene marker, excluded (return unused)
}

// pending is the list 0x683794+0x118x (count) / 0x68379c+0x118x (pointer).
func (w *cgW) pendingEach(x, flag int64) {
	pc := w.i32(w.pcAddr(x))
	list := w.u32(0x68379c + 0x118*x)
	for i := int64(0); i < int64(uint32(pc)); i++ {
		w.setMask(x, w.i32(list+4*i), flag)
	}
}

// Presentation state (0x562a50, 0x565420, 0x55ee30, 0x5653f0).
func (w *cgW) ui(mask int64) {
	if u := w.uip(); w.mode() == 7 && u != 0 {
		w.p32(u+0xf4, w.u32(u+0xf4)|mask)
	}
}
func (w *cgW) refresh(x int64) {
	switch w.mode() {
	case 6:
		a := w.u32(0x69c0f4) + 0xf8
		w.p32(a, w.u32(a)|0x100)
	case 7:
		w.ui(1 << uint(x))
	}
}
func (w *cgW) cone() {
	if u := w.uip(); u != 0 {
		if w.f94() {
			w.p8(u+0x2c0, 1)
		}
		w.p8(u+0x2c8, 1)
	}
}

// kindOK is 0x537160: whether record r suits target class k.
func (w *cgW) kindOK(r, k int64) bool {
	if r == 0 {
		return false
	}
	if uint32(k-1) > 7 {
		return true
	}
	r0 := w.i32(r)
	var ok bool
	switch k {
	case 1, 2, 8:
		ok = r0 == 1 && w.i32(r+4) != -1 || r0 == 3 && w.i32(r+8) != -1
	case 7:
		ok = r0 == 9 && w.i32(r+4) != -1
	case 5:
		ok = r0 == 2 || r0 == 4 && w.i32(r+12) != -1
	case 6:
		ok = r0 == 7 && w.i32(r+0x1c) != 0
	case 3, 4:
		ok = r0 == 6
	}
	return ok && w.st(cgS536190, r)&0xff != 0
}

// phase is 0x53a320 (also inlined in 0x537540). 0x55ee70 (cursor icon: writes
// 0x683fdb, 0x683fdf, 0x683fe3) is a registered exclusion, so its call with the
// new phase code e is only recorded; the cursor cells keep their input values.
func (w *cgW) phase(x int64) {
	w.p32(0x683fe7, x)
	e := int64(-1)
	switch x {
	case 1, 6, 7:
		e = 3
	case 2:
		e = 0
	case 3:
		e = 2
	case 4:
		e = 12
	case 5:
		e = 11
	}
	if e != w.i32(0x683fdb) {
		w.st(cgS55ee70, e)
	}
	w.p32(0x68407f, 0)
	w.p32(0x684083, 0)
	w.p32(0x684087, 0)
	w.cone()
	if x == 4 {
		w.st(cgS4add30, 1)
		w.drawSet()
		return
	}
	w.st(cgS4add30, 0)
	if w.mode() == 7 {
		w.draw()
	}
}

// cgTick is the case's fixed GetTickCount value ("tick" in the case files).
const cgTick = 0x1000

// clock is 0x482990(on). With on, 0x60a394 = 1 and 0x65b9a8 = tick; the elapsed
// time between its two GetTickCount calls is 0 under the fixed tick, so the
// time-of-day update 0x482a20 never runs (fixture cap: the real clock moves).
func (w *cgW) clock(on bool) {
	if on {
		w.p8(0x60a394, 1)
		w.p32(0x65b9a8, cgTick)
	} else {
		w.p8(0x60a394, 0)
	}
}

// movers is 0x4c61f0 (0x4531a0, clear) or 0x4c61e0 (0x453160, set) over the
// mover array 0x659b3c / count 0x659b34; 0x4a9c20 stamps obj+0x28 with the tick.
func (w *cgW) movers(set bool) {
	for i := int64(0); i < w.i32(0x659b34); i++ {
		m := w.u32(w.u32(0x659b3c) + 4*i)
		if set {
			w.p8(m+0x25, 1)
			w.p32(m+0xf4, 0)
			w.p32(m+0xf8, 0)
			w.p32(m+0xfc, 0)
			continue
		}
		o := w.u32(m + 0x68)
		w.p8(m+0x25, 0)
		if o != 0 {
			w.p32(o+0x28, cgTick)
		}
		if o = w.u32(m + 0x68); o != 0 {
			w.p32(o+0x28, cgTick)
		}
	}
}

// ctl is 0x41f160 (clear bit 0x40) or 0x41f0d0 (set it) on the dword at 0x652dac.
func (w *cgW) ctl(set bool) {
	p := w.u32(0x652dac)
	if p == 0 {
		return
	}
	if !set {
		w.p32(p, w.u32(p)&^0x40)
		return
	}
	w.p32(p, w.u32(p)|0x40)
	if w.u8(p)&1 == 0 {
		for _, o := range []int64{0x40, 0x44, 0x48, 0x4c, 0x50, 0x54, 0x7c, 0x80, 0x84, 0x64, 0x68, 0x6c, 0x70, 0x74, 0x78, 0xa0, 0xa4, 0xa8} {
			w.p32(p+o, 0)
		}
	}
	w.p32(p, w.u32(p)&^0x100)
}

// draw is 0x56aab0 (phase tail in mode 7). 0x420b40, 0x4f21b0, 0x4f23d0, 0x5a1950,
// 0x4263f0 and 0x422d50 are registered exclusions: calls recorded only.
func (w *cgW) draw() {
	if w.u8(0x683f95) != 0 || w.u8(0x683f97) != 0 || w.u8(0x683f96) != 0 || w.u8(0x683f9a) != 0 {
		return
	}
	if !w.f94() {
		if w.u8(0x6840bd) != 0 {
			w.st(cgS420b40, 4)
			w.st(cgS4f21b0, 0x137)
			w.st(cgS4f23d0, 0x137)
		}
		w.clock(true)
		w.movers(false)
		if w.u8(0x683f98) == 0 && w.u8(0x683f99) == 0 && w.u8(0x683f9c) == 0 && w.u8(0x683f9d) == 0 {
			w.ctl(false)
		}
	}
	w.p8(0x6840bc, 0)
	w.p8(0x6840bd, 0)
	if w.mode() != 7 {
		return
	}
	u := w.uip()
	if w.u8(u+0x327) != 0 {
		return
	}
	if w.u8(0x683fcd) != 0 {
		w.st(cgS5a1950)
	}
	w.st(cgS4263f0, 0xb1, 0x13f, 0x1cf, 0x153)
	w.st(cgS422d50, 0xb1, 0x13f, 0x1cf, 0x153, 0)
	if u != 0 {
		w.p32(u+0xf4, w.u32(u+0xf4)|0x8000)
	}
}

// drawSet is 0x56aa30 (phase 4). 0x5a19a0, 0x4f2380 and 0x4f2040 are registered
// exclusions: address stubs that record their calls.
func (w *cgW) drawSet() {
	w.p8(0x6840bc, 1)
	if w.u8(0x683fcd) != 0 {
		w.st(cgS5a19a0)
	}
	if !w.f94() {
		if w.u8(0x6840bd) != 0 {
			w.st(cgS420b40, 1)
			w.st(cgS4f2380, 0x137)
			w.st(cgS4f2040, 0x137)
		}
		w.clock(false)
		w.movers(true)
		w.ctl(true)
	}
	if u := w.uip(); w.mode() == 7 && u != 0 {
		w.p32(u+0xf4, w.u32(u+0xf4)|0x8000)
	}
}

func init() {
	Nodes["00520d10"] = Node{Addr: "00520d10", In: append(append([]string{}, cgInNames...), "a_p1", "a_p2", "a_p3"),
		Out: cgOutNames, F: equipmentActionRefreshNode}
}

func equipmentActionRefreshNode(in []int64, r Rand) []int64 {
	return cgRun(in, 3, 0, func(w *cgW, a []int64) int64 { w.refreshEquip(a[0], a[1], a[2]); return 0 })
}

// cgRun builds the world from the input vector (cgInit, then cgIn cells, then
// the nargs member arguments), runs body, and reads cgOut back; ret 1 appends the
// body's result (an unsigned byte), ret 4 a signed dword.
func cgRun(in []int64, nargs, ret int, body func(w *cgW, a []int64) int64) []int64 {
	if len(in) != len(cgIn)+nargs {
		panic("cg: wrong input count")
	}
	w := cgWorld(in)
	r := body(w, in[len(cgIn):])
	out := cgRead(w)
	if ret != 0 {
		out = append(out, r)
	}
	return out
}

// cgWorld builds the sparse world: cgInit, then the cgIn cells from in.
func cgWorld(in []int64) *cgW {
	w := &cgW{m: map[int64]byte{}}
	for _, c := range cgInit {
		w.wr(c.a, c.s, c.v)
	}
	for i, c := range cgIn {
		s := c.s
		if s < 0 {
			s = -s
		}
		w.wr(c.a, s, in[i])
	}
	return w
}

// cgRead reads every cgOut cell.
func cgRead(w *cgW) []int64 {
	out := make([]int64, 0, len(cgOut)+1)
	for _, c := range cgOut {
		if c.s < 0 {
			out = append(out, w.rs(c.a, -c.s))
		} else {
			out = append(out, w.rd(c.a, c.s))
		}
	}
	return out
}

// refreshEquip is 0x520d10(p1, p2, p3): react to a change of character p2's
// slot p1 (equipment or hand item).
func (w *cgW) refreshEquip(p1, p2, p3 int64) {
	if p2 == 0 {
		return
	}
	b5, b6 := p1 == p2+0xfa5, p1 == p2+0xfb1
	if b5 {
		w.st(cgS5201b0, p2, 6)
	}
	if b6 {
		w.st(cgS5201b0, p2, 7)
	}
	bl := b5 || b6
	if w.u8(p2+4) == 0 || w.u8(0x68517c) == 0 {
		return
	}
	slot := w.slotOf(p2)
	if bl {
		w.st(cgS4ee220, p2)
	}
	if p3&0xff == 0 {
		return
	}
	if p1 == p2+0xfe1 {
		w.p8(0x652da6, w.hasItem(0x254))
	}
	w.st(cgS50e5c0, slot)
	S, D := w.S(slot), w.D(slot)
	if w.i32(S+0x3d) == 8 && w.u32(S+0x49) == p1 {
		w.reapply(slot)
	}
	if bl {
		switch t := w.i32(S + 0x6d); {
		case t == 1:
			if w.st(cgS5458a0, slot)&0xff == 0 {
				w.p32(S+0x6d, 0)
				if w.i32(S+0x3d) == 1 {
					w.p32(S+0x3d, 0)
				}
			}
		case t == 0 && w.u8(S+0x104) != 0 && w.st(cgS5458a0, slot)&0xff != 0:
			w.p32(S+0x6d, 1)
			if w.i32(S+0x3d) == 0 {
				w.p32(S+0x3d, 1)
			}
		}
		if w.f94() {
			if act := w.i32(S + 0x3d); act == 0 || act == 1 {
				if w.st(cgS4e79a0, slot, 1, 1, 0)&0xff == 0 {
					w.record(slot, cgRec{0, -1, -1, -1}, 1)
				} else if w.st(cgS536f60, slot, 2)&0xff == 0 {
					w.target(slot, 1, 0)
				}
			}
			w.p8(D+0x99, w.u8(D+0x99)^1)
			if slot == w.sel() {
				w.cone()
			}
		}
		w.ui(1 << uint(slot))
	}
	if u := w.uip(); w.mode() == 7 && u != 0 {
		if int16(w.rd(u+0x2ca, 2)) != -1 {
			w.st(cgS595600)
		}
		w.st(cgS593330)
	}
}

// slotOf is 0x517ec0: the party slot whose character record is p2.
func (w *cgW) slotOf(p2 int64) int64 {
	if w.u8(p2+4) == 0 {
		panic("cg: 0x517ec0 assert")
	}
	for i := int64(0); i < 8; i++ {
		if w.u32(0x685174)+0x1862*i == p2 {
			return w.who(i)
		}
	}
	panic("cg: 0x517ec0 assert")
}

// hasItem is 0x521060(id,0,0,0,0): whether id is on the active list or in any
// present member's list-A.
func (w *cgW) hasItem(id int64) int64 {
	if w.u8(0x6874ca) != 0 && w.i32(0x6874cb) == id {
		return 1
	}
	for m := int64(0); m < 8; m++ {
		if w.u8(w.u32(0x685178)+0x106*m) == 0 {
			continue
		}
		for i := int64(0); i < 12; i++ {
			if w.i32(w.u32(0x685174)+0x1862*m+0xf5d+12*i) == id {
				return 1
			}
		}
	}
	return 0
}

// ---- generated tables
var cgIn = []cgCell{{0x683f94, 1}, {0x683f95, 1}, {0x683f96, 1}, {0x683f97, 1}, {0x683f98, 1}, {0x683f99, 1}, {0x683f9a, 1}, {0x683f9c, 1}, {0x683f9d, 1}, {0x68518d, -4}, {0x68517c, 1}, {0x68ec78, 4}, {0x683fe7, 4}, {0x683fdb, -4}, {0x683fdf, 4}, {0x68407f, 4}, {0x684083, 4}, {0x684087, 4}, {0x6840af, 4}, {0x685115, 1}, {0x68510c, 1}, {0x6850e5, 4}, {0x6874ca, 1}, {0x6874cb, -4}, {0x652da6, 1}, {0x69b9a0, 4}, {0x69b9a4, 4}, {0x6840bc, 1}, {0x6840bd, 1}, {0x683fcd, 1}, {0x68edcc, 4}, {0x68408b, -4}, {0x68408f, -4}, {0x684093, -4}, {0x684097, -4}, {0x68409b, -4}, {0x68409f, -4}, {0x6840a3, -4}, {0x6840a7, -4}, {0x2001c100, 4}, {0x2001c104, 4}, {0x2001b0f8, 4}, {0x200150f4, 4}, {0x20015154, 1}, {0x200152c0, 1}, {0x200152c8, 1}, {0x200152ca, -2}, {0x200152f0, -4}, {0x200152f4, -4}, {0x200152f8, 1}, {0x20015327, 1}, {0x20011000, 1}, {0x20011008, 4}, {0x200117b4, 4}, {0x20012063, 1}, {0x2001203e, 1}, {0x20012170, 1}, {0x2001214b, 1}, {0x2001227d, 1}, {0x20012258, 1}, {0x2001238a, 1}, {0x20012365, 1}, {0x2001e2f6, 4}, {0x2001e4b5, 4}, {0x2001e674, 4}, {0x2001e833, 4}, {0x2001e9f2, 4}, {0x2001ebb1, 4}, {0x2001ed70, 4}, {0x2001ef2f, 4}, {0x2001f0ee, 4}, {0x20027adf, 4}, {0x2002b100, 4}, {0x20030000, -4}, {0x20030004, -4}, {0x20030008, -4}, {0x2003000c, -4}, {0x20031000, 4}, {0x20031004, 4}, {0x20010000, 1}, {0x20010001, -4}, {0x20010005, -4}, {0x20010019, 4}, {0x2001001d, -4}, {0x20010021, -4}, {0x20010025, -4}, {0x20010029, -4}, {0x2001002d, -4}, {0x20010031, -4}, {0x20010035, -4}, {0x20010039, -4}, {0x2001004d, -4}, {0x20010051, -4}, {0x20010055, -4}, {0x20010059, -4}, {0x2001005d, -4}, {0x20010061, -4}, {0x20010065, -4}, {0x20010069, -4}, {0x20010081, -4}, {0x20010085, -4}, {0x20010089, -4}, {0x2001008d, -4}, {0x20010091, -4}, {0x20010095, -4}, {0x20010099, -4}, {0x2001009d, -4}, {0x200100a9, -4}, {0x200100ad, -4}, {0x200100b1, -4}, {0x200100b5, -4}, {0x200100b9, -4}, {0x200100bd, -4}, {0x200100c1, -4}, {0x200100c5, -4}, {0x200100d1, -4}, {0x200100d5, -4}, {0x200100d9, -4}, {0x200100dd, -4}, {0x200100e1, -4}, {0x200100e5, -4}, {0x200100e9, -4}, {0x200100ed, -4}, {0x2001003d, -4}, {0x20010041, -4}, {0x20010045, 4}, {0x20010049, 4}, {0x2001006d, -4}, {0x20010071, -4}, {0x20010075, -4}, {0x200100a5, 4}, {0x200100f5, 1}, {0x20010104, 1}, {0x20010105, 1}, {0x20000004, 1}, {0x20000b11, 4}, {0x20000b01, 4}, {0x20000f5d, 4}, {0x20000f69, 4}, {0x20000f75, 4}, {0x20000f81, 4}, {0x20000f8d, 4}, {0x20000f99, 4}, {0x20000fa5, 4}, {0x20000fb1, 4}, {0x20000fbd, 4}, {0x20000fc9, 4}, {0x20000fd5, 4}, {0x20000fe1, 4}, {0x20011018, 4}, {0x200110b4, 4}, {0x2001104c, 1}, {0x20011099, 1}, {0x200110ac, 4}, {0x200110bf, 1}, {0x6837cb, 1}, {0x683794, 4}, {0x20010106, 1}, {0x20010107, -4}, {0x2001010b, -4}, {0x2001011f, 4}, {0x20010123, -4}, {0x20010127, -4}, {0x2001012b, -4}, {0x2001012f, -4}, {0x20010133, -4}, {0x20010137, -4}, {0x2001013b, -4}, {0x2001013f, -4}, {0x20010153, -4}, {0x20010157, -4}, {0x2001015b, -4}, {0x2001015f, -4}, {0x20010163, -4}, {0x20010167, -4}, {0x2001016b, -4}, {0x2001016f, -4}, {0x20010187, -4}, {0x2001018b, -4}, {0x2001018f, -4}, {0x20010193, -4}, {0x20010197, -4}, {0x2001019b, -4}, {0x2001019f, -4}, {0x200101a3, -4}, {0x200101af, -4}, {0x200101b3, -4}, {0x200101b7, -4}, {0x200101bb, -4}, {0x200101bf, -4}, {0x200101c3, -4}, {0x200101c7, -4}, {0x200101cb, -4}, {0x200101d7, -4}, {0x200101db, -4}, {0x200101df, -4}, {0x200101e3, -4}, {0x200101e7, -4}, {0x200101eb, -4}, {0x200101ef, -4}, {0x200101f3, -4}, {0x20010143, -4}, {0x20010147, -4}, {0x2001014b, 4}, {0x2001014f, 4}, {0x20010173, -4}, {0x20010177, -4}, {0x2001017b, -4}, {0x200101ab, 4}, {0x200101fb, 1}, {0x2001020a, 1}, {0x2001020b, 1}, {0x20001866, 1}, {0x20002373, 4}, {0x20002363, 4}, {0x200027bf, 4}, {0x200027cb, 4}, {0x200027d7, 4}, {0x200027e3, 4}, {0x200027ef, 4}, {0x200027fb, 4}, {0x20002807, 4}, {0x20002813, 4}, {0x2000281f, 4}, {0x2000282b, 4}, {0x20002837, 4}, {0x20002843, 4}, {0x200110ec, 4}, {0x20011188, 4}, {0x20011120, 1}, {0x2001116d, 1}, {0x20011180, 4}, {0x20011193, 1}, {0x6838e3, 1}, {0x6838ac, 4}, {0x2001020c, 1}, {0x2001020d, -4}, {0x20010211, -4}, {0x20010225, 4}, {0x20010229, -4}, {0x2001022d, -4}, {0x20010231, -4}, {0x20010235, -4}, {0x20010239, -4}, {0x2001023d, -4}, {0x20010241, -4}, {0x20010245, -4}, {0x20010259, -4}, {0x2001025d, -4}, {0x20010261, -4}, {0x20010265, -4}, {0x20010269, -4}, {0x2001026d, -4}, {0x20010271, -4}, {0x20010275, -4}, {0x2001028d, -4}, {0x20010291, -4}, {0x20010295, -4}, {0x20010299, -4}, {0x2001029d, -4}, {0x200102a1, -4}, {0x200102a5, -4}, {0x200102a9, -4}, {0x200102b5, -4}, {0x200102b9, -4}, {0x200102bd, -4}, {0x200102c1, -4}, {0x200102c5, -4}, {0x200102c9, -4}, {0x200102cd, -4}, {0x200102d1, -4}, {0x200102dd, -4}, {0x200102e1, -4}, {0x200102e5, -4}, {0x200102e9, -4}, {0x200102ed, -4}, {0x200102f1, -4}, {0x200102f5, -4}, {0x200102f9, -4}, {0x20010249, -4}, {0x2001024d, -4}, {0x20010251, 4}, {0x20010255, 4}, {0x20010279, -4}, {0x2001027d, -4}, {0x20010281, -4}, {0x200102b1, 4}, {0x20010301, 1}, {0x20010310, 1}, {0x20010311, 1}, {0x200030c8, 1}, {0x20003bd5, 4}, {0x20003bc5, 4}, {0x20004021, 4}, {0x2000402d, 4}, {0x20004039, 4}, {0x20004045, 4}, {0x20004051, 4}, {0x2000405d, 4}, {0x20004069, 4}, {0x20004075, 4}, {0x20004081, 4}, {0x2000408d, 4}, {0x20004099, 4}, {0x200040a5, 4}, {0x200111c0, 4}, {0x2001125c, 4}, {0x200111f4, 1}, {0x20011241, 1}, {0x20011254, 4}, {0x20011267, 1}, {0x6839fb, 1}, {0x6839c4, 4}, {0x20010312, 1}, {0x20010313, -4}, {0x20010317, -4}, {0x2001032b, 4}, {0x2001032f, -4}, {0x20010333, -4}, {0x20010337, -4}, {0x2001033b, -4}, {0x2001033f, -4}, {0x20010343, -4}, {0x20010347, -4}, {0x2001034b, -4}, {0x2001035f, -4}, {0x20010363, -4}, {0x20010367, -4}, {0x2001036b, -4}, {0x2001036f, -4}, {0x20010373, -4}, {0x20010377, -4}, {0x2001037b, -4}, {0x20010393, -4}, {0x20010397, -4}, {0x2001039b, -4}, {0x2001039f, -4}, {0x200103a3, -4}, {0x200103a7, -4}, {0x200103ab, -4}, {0x200103af, -4}, {0x200103bb, -4}, {0x200103bf, -4}, {0x200103c3, -4}, {0x200103c7, -4}, {0x200103cb, -4}, {0x200103cf, -4}, {0x200103d3, -4}, {0x200103d7, -4}, {0x200103e3, -4}, {0x200103e7, -4}, {0x200103eb, -4}, {0x200103ef, -4}, {0x200103f3, -4}, {0x200103f7, -4}, {0x200103fb, -4}, {0x200103ff, -4}, {0x2001034f, -4}, {0x20010353, -4}, {0x20010357, 4}, {0x2001035b, 4}, {0x2001037f, -4}, {0x20010383, -4}, {0x20010387, -4}, {0x200103b7, 4}, {0x20010407, 1}, {0x20010416, 1}, {0x20010417, 1}, {0x2000492a, 1}, {0x20005437, 4}, {0x20005427, 4}, {0x20005883, 4}, {0x2000588f, 4}, {0x2000589b, 4}, {0x200058a7, 4}, {0x200058b3, 4}, {0x200058bf, 4}, {0x200058cb, 4}, {0x200058d7, 4}, {0x200058e3, 4}, {0x200058ef, 4}, {0x200058fb, 4}, {0x20005907, 4}, {0x20011294, 4}, {0x20011330, 4}, {0x200112c8, 1}, {0x20011315, 1}, {0x20011328, 4}, {0x2001133b, 1}, {0x683b13, 1}, {0x683adc, 4}, {0x20040008, 4}, {0x20040200, -4}, {0x20040204, -4}, {0x20040214, 1}, {0x200419bc, 1}, {0x20040240, -4}, {0x20040244, -4}, {0x20040254, 1}, {0x200421bc, 1}, {0x2001d000, -4}, {0x2001d004, -4}, {0x2001d010, -4}, {0x2001d014, -4}, {0x2001d020, -4}, {0x2001d024, -4}, {0x2001d030, -4}, {0x2001d034, -4}, {0x20017100, -4}, {0x20017104, -4}, {0x20017110, -4}, {0x20017120, -4}, {0x20017124, -4}, {0x20017128, -4}, {0x2001712c, -4}, {0x20017130, -4}, {0x20017134, -4}, {0x20017148, -4}, {0x60a394, 1}, {0x65b9a8, 4}, {0x659b34, -4}, {0x652dac, 4}, {0x20050600, 4}, {0x20050640, 4}, {0x20050654, 4}, {0x20050664, 4}, {0x20050678, 4}, {0x2005067c, 4}, {0x20050684, 4}, {0x200506a0, 4}, {0x200506a8, 4}, {0x20050125, 1}, {0x20050168, 4}, {0x200501f4, 4}, {0x200501f8, 4}, {0x200501fc, 4}, {0x20050325, 1}, {0x20050368, 4}, {0x200503f4, 4}, {0x200503f8, 4}, {0x200503fc, 4}, {0x20050528, 4}, {0x200505a8, 4}}
var cgOut = []cgCell{{0x683f94, 1}, {0x683f95, 1}, {0x683f96, 1}, {0x683f97, 1}, {0x683f98, 1}, {0x683f99, 1}, {0x683f9a, 1}, {0x683f9c, 1}, {0x683f9d, 1}, {0x68518d, -4}, {0x68517c, 1}, {0x68ec78, 4}, {0x683fe7, 4}, {0x683fdb, -4}, {0x683fdf, 4}, {0x68407f, 4}, {0x684083, 4}, {0x684087, 4}, {0x6840af, 4}, {0x685115, 1}, {0x68510c, 1}, {0x6850e5, 4}, {0x6874ca, 1}, {0x6874cb, -4}, {0x652da6, 1}, {0x69b9a0, 4}, {0x69b9a4, 4}, {0x6840bc, 1}, {0x6840bd, 1}, {0x683fcd, 1}, {0x68edcc, 4}, {0x68408b, -4}, {0x68408f, -4}, {0x684093, -4}, {0x684097, -4}, {0x68409b, -4}, {0x68409f, -4}, {0x6840a3, -4}, {0x6840a7, -4}, {0x2001c100, 4}, {0x2001c104, 4}, {0x2001b0f8, 4}, {0x200150f4, 4}, {0x20015154, 1}, {0x200152c0, 1}, {0x200152c8, 1}, {0x200152ca, -2}, {0x200152f0, -4}, {0x200152f4, -4}, {0x200152f8, 1}, {0x20015327, 1}, {0x20011000, 1}, {0x20011008, 4}, {0x200117b4, 4}, {0x20012063, 1}, {0x2001203e, 1}, {0x20012170, 1}, {0x2001214b, 1}, {0x2001227d, 1}, {0x20012258, 1}, {0x2001238a, 1}, {0x20012365, 1}, {0x2001e2f6, 4}, {0x2001e4b5, 4}, {0x2001e674, 4}, {0x2001e833, 4}, {0x2001e9f2, 4}, {0x2001ebb1, 4}, {0x2001ed70, 4}, {0x2001ef2f, 4}, {0x2001f0ee, 4}, {0x20027adf, 4}, {0x2002b100, 4}, {0x20030000, -4}, {0x20030004, -4}, {0x20030008, -4}, {0x2003000c, -4}, {0x20031000, 4}, {0x20031004, 4}, {0x20010000, 1}, {0x20010001, -4}, {0x20010005, -4}, {0x20010019, 4}, {0x2001001d, -4}, {0x20010021, -4}, {0x20010025, -4}, {0x20010029, -4}, {0x2001002d, -4}, {0x20010031, -4}, {0x20010035, -4}, {0x20010039, -4}, {0x2001004d, -4}, {0x20010051, -4}, {0x20010055, -4}, {0x20010059, -4}, {0x2001005d, -4}, {0x20010061, -4}, {0x20010065, -4}, {0x20010069, -4}, {0x20010081, -4}, {0x20010085, -4}, {0x20010089, -4}, {0x2001008d, -4}, {0x20010091, -4}, {0x20010095, -4}, {0x20010099, -4}, {0x2001009d, -4}, {0x200100a9, -4}, {0x200100ad, -4}, {0x200100b1, -4}, {0x200100b5, -4}, {0x200100b9, -4}, {0x200100bd, -4}, {0x200100c1, -4}, {0x200100c5, -4}, {0x200100d1, -4}, {0x200100d5, -4}, {0x200100d9, -4}, {0x200100dd, -4}, {0x200100e1, -4}, {0x200100e5, -4}, {0x200100e9, -4}, {0x200100ed, -4}, {0x2001003d, -4}, {0x20010041, -4}, {0x20010045, 4}, {0x20010049, 4}, {0x2001006d, -4}, {0x20010071, -4}, {0x20010075, -4}, {0x200100a5, 4}, {0x200100f5, 1}, {0x20010104, 1}, {0x20010105, 1}, {0x20000004, 1}, {0x20000b11, 4}, {0x20000b01, 4}, {0x20000f5d, 4}, {0x20000f69, 4}, {0x20000f75, 4}, {0x20000f81, 4}, {0x20000f8d, 4}, {0x20000f99, 4}, {0x20000fa5, 4}, {0x20000fb1, 4}, {0x20000fbd, 4}, {0x20000fc9, 4}, {0x20000fd5, 4}, {0x20000fe1, 4}, {0x20011018, 4}, {0x200110b4, 4}, {0x2001104c, 1}, {0x20011099, 1}, {0x200110ac, 4}, {0x200110bf, 1}, {0x6837cb, 1}, {0x683794, 4}, {0x20010106, 1}, {0x20010107, -4}, {0x2001010b, -4}, {0x2001011f, 4}, {0x20010123, -4}, {0x20010127, -4}, {0x2001012b, -4}, {0x2001012f, -4}, {0x20010133, -4}, {0x20010137, -4}, {0x2001013b, -4}, {0x2001013f, -4}, {0x20010153, -4}, {0x20010157, -4}, {0x2001015b, -4}, {0x2001015f, -4}, {0x20010163, -4}, {0x20010167, -4}, {0x2001016b, -4}, {0x2001016f, -4}, {0x20010187, -4}, {0x2001018b, -4}, {0x2001018f, -4}, {0x20010193, -4}, {0x20010197, -4}, {0x2001019b, -4}, {0x2001019f, -4}, {0x200101a3, -4}, {0x200101af, -4}, {0x200101b3, -4}, {0x200101b7, -4}, {0x200101bb, -4}, {0x200101bf, -4}, {0x200101c3, -4}, {0x200101c7, -4}, {0x200101cb, -4}, {0x200101d7, -4}, {0x200101db, -4}, {0x200101df, -4}, {0x200101e3, -4}, {0x200101e7, -4}, {0x200101eb, -4}, {0x200101ef, -4}, {0x200101f3, -4}, {0x20010143, -4}, {0x20010147, -4}, {0x2001014b, 4}, {0x2001014f, 4}, {0x20010173, -4}, {0x20010177, -4}, {0x2001017b, -4}, {0x200101ab, 4}, {0x200101fb, 1}, {0x2001020a, 1}, {0x2001020b, 1}, {0x20001866, 1}, {0x20002373, 4}, {0x20002363, 4}, {0x200027bf, 4}, {0x200027cb, 4}, {0x200027d7, 4}, {0x200027e3, 4}, {0x200027ef, 4}, {0x200027fb, 4}, {0x20002807, 4}, {0x20002813, 4}, {0x2000281f, 4}, {0x2000282b, 4}, {0x20002837, 4}, {0x20002843, 4}, {0x200110ec, 4}, {0x20011188, 4}, {0x20011120, 1}, {0x2001116d, 1}, {0x20011180, 4}, {0x20011193, 1}, {0x6838e3, 1}, {0x6838ac, 4}, {0x2001020c, 1}, {0x2001020d, -4}, {0x20010211, -4}, {0x20010225, 4}, {0x20010229, -4}, {0x2001022d, -4}, {0x20010231, -4}, {0x20010235, -4}, {0x20010239, -4}, {0x2001023d, -4}, {0x20010241, -4}, {0x20010245, -4}, {0x20010259, -4}, {0x2001025d, -4}, {0x20010261, -4}, {0x20010265, -4}, {0x20010269, -4}, {0x2001026d, -4}, {0x20010271, -4}, {0x20010275, -4}, {0x2001028d, -4}, {0x20010291, -4}, {0x20010295, -4}, {0x20010299, -4}, {0x2001029d, -4}, {0x200102a1, -4}, {0x200102a5, -4}, {0x200102a9, -4}, {0x200102b5, -4}, {0x200102b9, -4}, {0x200102bd, -4}, {0x200102c1, -4}, {0x200102c5, -4}, {0x200102c9, -4}, {0x200102cd, -4}, {0x200102d1, -4}, {0x200102dd, -4}, {0x200102e1, -4}, {0x200102e5, -4}, {0x200102e9, -4}, {0x200102ed, -4}, {0x200102f1, -4}, {0x200102f5, -4}, {0x200102f9, -4}, {0x20010249, -4}, {0x2001024d, -4}, {0x20010251, 4}, {0x20010255, 4}, {0x20010279, -4}, {0x2001027d, -4}, {0x20010281, -4}, {0x200102b1, 4}, {0x20010301, 1}, {0x20010310, 1}, {0x20010311, 1}, {0x200030c8, 1}, {0x20003bd5, 4}, {0x20003bc5, 4}, {0x20004021, 4}, {0x2000402d, 4}, {0x20004039, 4}, {0x20004045, 4}, {0x20004051, 4}, {0x2000405d, 4}, {0x20004069, 4}, {0x20004075, 4}, {0x20004081, 4}, {0x2000408d, 4}, {0x20004099, 4}, {0x200040a5, 4}, {0x200111c0, 4}, {0x2001125c, 4}, {0x200111f4, 1}, {0x20011241, 1}, {0x20011254, 4}, {0x20011267, 1}, {0x6839fb, 1}, {0x6839c4, 4}, {0x20010312, 1}, {0x20010313, -4}, {0x20010317, -4}, {0x2001032b, 4}, {0x2001032f, -4}, {0x20010333, -4}, {0x20010337, -4}, {0x2001033b, -4}, {0x2001033f, -4}, {0x20010343, -4}, {0x20010347, -4}, {0x2001034b, -4}, {0x2001035f, -4}, {0x20010363, -4}, {0x20010367, -4}, {0x2001036b, -4}, {0x2001036f, -4}, {0x20010373, -4}, {0x20010377, -4}, {0x2001037b, -4}, {0x20010393, -4}, {0x20010397, -4}, {0x2001039b, -4}, {0x2001039f, -4}, {0x200103a3, -4}, {0x200103a7, -4}, {0x200103ab, -4}, {0x200103af, -4}, {0x200103bb, -4}, {0x200103bf, -4}, {0x200103c3, -4}, {0x200103c7, -4}, {0x200103cb, -4}, {0x200103cf, -4}, {0x200103d3, -4}, {0x200103d7, -4}, {0x200103e3, -4}, {0x200103e7, -4}, {0x200103eb, -4}, {0x200103ef, -4}, {0x200103f3, -4}, {0x200103f7, -4}, {0x200103fb, -4}, {0x200103ff, -4}, {0x2001034f, -4}, {0x20010353, -4}, {0x20010357, 4}, {0x2001035b, 4}, {0x2001037f, -4}, {0x20010383, -4}, {0x20010387, -4}, {0x200103b7, 4}, {0x20010407, 1}, {0x20010416, 1}, {0x20010417, 1}, {0x2000492a, 1}, {0x20005437, 4}, {0x20005427, 4}, {0x20005883, 4}, {0x2000588f, 4}, {0x2000589b, 4}, {0x200058a7, 4}, {0x200058b3, 4}, {0x200058bf, 4}, {0x200058cb, 4}, {0x200058d7, 4}, {0x200058e3, 4}, {0x200058ef, 4}, {0x200058fb, 4}, {0x20005907, 4}, {0x20011294, 4}, {0x20011330, 4}, {0x200112c8, 1}, {0x20011315, 1}, {0x20011328, 4}, {0x2001133b, 1}, {0x683b13, 1}, {0x683adc, 4}, {0x20040008, 4}, {0x20040200, -4}, {0x20040204, -4}, {0x20040214, 1}, {0x200419bc, 1}, {0x20040240, -4}, {0x20040244, -4}, {0x20040254, 1}, {0x200421bc, 1}, {0x2001d000, -4}, {0x2001d004, -4}, {0x2001d010, -4}, {0x2001d014, -4}, {0x2001d020, -4}, {0x2001d024, -4}, {0x2001d030, -4}, {0x2001d034, -4}, {0x20017800, 4}, {0x20017200, -4}, {0x20017204, -4}, {0x20017208, -4}, {0x2001720c, -4}, {0x20017804, 4}, {0x20017240, -4}, {0x20017244, -4}, {0x20017808, 4}, {0x20017280, -4}, {0x20017284, -4}, {0x2001780c, 4}, {0x200172c0, -4}, {0x20017810, 4}, {0x20017300, -4}, {0x20017814, 4}, {0x20017340, -4}, {0x20017344, -4}, {0x20017818, 4}, {0x20017380, -4}, {0x2001781c, 4}, {0x200173c0, -4}, {0x20017820, 4}, {0x20017400, -4}, {0x20017824, 4}, {0x20017440, -4}, {0x20017444, -4}, {0x20017448, -4}, {0x20017828, 4}, {0x20017480, -4}, {0x20017484, -4}, {0x20017488, -4}, {0x2001782c, 4}, {0x200174c0, -4}, {0x200174c4, -4}, {0x200174c8, -4}, {0x20017830, 4}, {0x20017500, -4}, {0x20017834, 4}, {0x20017540, -4}, {0x20017838, 4}, {0x20017580, -4}, {0x20017584, -4}, {0x20017590, -4}, {0x20017598, -4}, {0x2001783c, 4}, {0x200175c0, -4}, {0x20017840, 4}, {0x20017844, 4}, {0x20017848, 4}, {0x20017680, -4}, {0x20017684, -4}, {0x20017688, -4}, {0x2001768c, -4}, {0x20017690, -4}, {0x2001784c, 4}, {0x200176c0, -4}, {0x200176c4, -4}, {0x200176c8, -4}, {0x60a394, 1}, {0x65b9a8, 4}, {0x659b34, -4}, {0x652dac, 4}, {0x20050600, 4}, {0x20050640, 4}, {0x20050654, 4}, {0x20050664, 4}, {0x20050678, 4}, {0x2005067c, 4}, {0x20050684, 4}, {0x200506a0, 4}, {0x200506a8, 4}, {0x20050125, 1}, {0x20050168, 4}, {0x200501f4, 4}, {0x200501f8, 4}, {0x200501fc, 4}, {0x20050325, 1}, {0x20050368, 4}, {0x200503f4, 4}, {0x200503f8, 4}, {0x200503fc, 4}, {0x20050528, 4}, {0x200505a8, 4}, {0x20017880, 4}, {0x20017b00, -4}, {0x20017884, 4}, {0x20017b40, -4}, {0x20017888, 4}, {0x20017b80, -4}, {0x2001788c, 4}, {0x20017bc0, -4}, {0x20017890, 4}, {0x20017894, 4}, {0x20017c40, -4}, {0x20017c44, -4}, {0x20017c48, -4}, {0x20017c4c, -4}, {0x20017898, 4}, {0x20017c80, -4}, {0x20017c84, -4}, {0x20017c88, -4}, {0x20017c8c, -4}, {0x20017c90, -4}, {0x2001789c, 4}, {0x200178a0, 4}, {0x20017d00, -4}, {0x200178a4, 4}, {0x20017d40, -4}}
var cgInit = []cgInitCell{{0x685174, 4, 0x20000000}, {0x685178, 4, 0x20010000}, {0x6836a8, 4, 0x20011000}, {0x68516c, 4, 0x20012000}, {0x68c09c, 4, 0x20014000}, {0x65be1c, 4, 0x2001e000}, {0x69bf3c, 4, 0x2001c000}, {0x69c0f4, 4, 0x2001b000}, {0x683f78, 4, 0x4}, {0x650e20, 4, 0x1}, {0x62c464, 4, 0x1}, {0x62c4a0, 4, 0x1}, {0x62c4dc, 4, 0x1}, {0x62c518, 4, 0x1}, {0x62c554, 4, 0x1}, {0x62c590, 4, 0x1}, {0x62c5cc, 4, 0x1}, {0x62c608, 4, 0x1}, {0x62c644, 4, 0x1}, {0x62c680, 4, 0x1}, {0x62c6bc, 4, 0x1}, {0x62c6f8, 4, 0x1}, {0x62c734, 4, 0x1}, {0x62c770, 4, 0x1}, {0x62c7ac, 4, 0x1}, {0x62c7e8, 4, 0x1}, {0x62c824, 4, 0x1}, {0x62c860, 4, 0x1}, {0x62c89c, 4, 0x1}, {0x62c8d8, 4, 0x1}, {0x62c914, 4, 0x1}, {0x62c950, 4, 0x1}, {0x62c98c, 4, 0x1}, {0x62c9c8, 4, 0x1}, {0x62ca04, 4, 0x1}, {0x62ca40, 4, 0x1}, {0x62ca7c, 4, 0x1}, {0x62cab8, 4, 0x1}, {0x62caf4, 4, 0x1}, {0x62cb30, 4, 0x1}, {0x62cb6c, 4, 0x1}, {0x62cba8, 4, 0x1}, {0x62cbe4, 4, 0x1}, {0x62cc20, 4, 0x1}, {0x62cc5c, 4, 0x1}, {0x62cc98, 4, 0x1}, {0x62ccd4, 4, 0x1}, {0x62cd10, 4, 0x1}, {0x62cd4c, 4, 0x1}, {0x62cd88, 4, 0x1}, {0x62cdc4, 4, 0x1}, {0x62ce00, 4, 0x1}, {0x62ce3c, 4, 0x1}, {0x62ce78, 4, 0x1}, {0x62ceb4, 4, 0x1}, {0x62cef0, 4, 0x1}, {0x62cf2c, 4, 0x1}, {0x62cf68, 4, 0x1}, {0x62cfa4, 4, 0x1}, {0x62cfe0, 4, 0x1}, {0x62d01c, 4, 0x1}, {0x62d058, 4, 0x1}, {0x62d094, 4, 0x1}, {0x62d0d0, 4, 0x1}, {0x62d10c, 4, 0x1}, {0x62d148, 4, 0x1}, {0x62d184, 4, 0x1}, {0x62d1c0, 4, 0x1}, {0x62d1fc, 4, 0x1}, {0x62d238, 4, 0x1}, {0x62d274, 4, 0x1}, {0x62d2b0, 4, 0x1}, {0x62d2ec, 4, 0x1}, {0x62d328, 4, 0x1}, {0x62d364, 4, 0x1}, {0x62d3a0, 4, 0x1}, {0x62d3dc, 4, 0x1}, {0x62d418, 4, 0x1}, {0x62d454, 4, 0x1}, {0x62d490, 4, 0x1}, {0x62d4cc, 4, 0x1}, {0x62d508, 4, 0x1}, {0x62d544, 4, 0x1}, {0x62d580, 4, 0x1}, {0x62d5bc, 4, 0x1}, {0x62d5f8, 4, 0x1}, {0x62d634, 4, 0x1}, {0x62d670, 4, 0x1}, {0x62d6ac, 4, 0x1}, {0x62d6e8, 4, 0x1}, {0x62d724, 4, 0x1}, {0x62d760, 4, 0x1}, {0x62d79c, 4, 0x1}, {0x62d7d8, 4, 0x1}, {0x62d814, 4, 0x1}, {0x62d850, 4, 0x1}, {0x62d88c, 4, 0x1}, {0x62d8c8, 4, 0x1}, {0x62d904, 4, 0x1}, {0x62d940, 4, 0x1}, {0x62d97c, 4, 0x1}, {0x62d9b8, 4, 0x1}, {0x62d9f4, 4, 0x1}, {0x62da30, 4, 0x1}, {0x62da6c, 4, 0x1}, {0x62daa8, 4, 0x1}, {0x20014894, 4, 0x20019000}, {0x20014898, 4, 0x20019000}, {0x200149ac, 4, 0x20019000}, {0x200149b0, 4, 0x20019000}, {0x200149b4, 4, 0x20019000}, {0x20019000, 4, 0x25207325}, {0x20019004, 4, 0x74696820}, {0x68379c, 4, 0x2001d000}, {0x6838b4, 4, 0x2001d010}, {0x6839cc, 4, 0x2001d020}, {0x683ae4, 4, 0x2001d030}, {0x2001a000, 4, 0x2001a100}, {0x2001a100, 4, 0x2001a400}, {0x2001a400, 1, 0xc3}, {0x2001a104, 4, 0x2001a408}, {0x2001a408, 1, 0xc3}, {0x2001a108, 4, 0x2001a410}, {0x2001a410, 1, 0xc3}, {0x2001a10c, 4, 0x2001a418}, {0x2001a418, 1, 0xc3}, {0x2001a110, 4, 0x2001a420}, {0x2001a420, 1, 0xc3}, {0x2001a114, 4, 0x2001a428}, {0x2001a428, 1, 0xc3}, {0x2001a118, 4, 0x2001a430}, {0x2001a430, 1, 0xc3}, {0x2001a11c, 4, 0x2001a438}, {0x2001a438, 1, 0xc3}, {0x2001a120, 4, 0x2001a440}, {0x2001a440, 1, 0xc3}, {0x2001a124, 4, 0x2001a448}, {0x2001a448, 1, 0xc3}, {0x2001a128, 4, 0x2001a450}, {0x2001a450, 1, 0xc3}, {0x2001a12c, 4, 0x2001a458}, {0x2001a458, 1, 0xc3}, {0x2001a130, 4, 0x2001a460}, {0x2001a460, 1, 0xc3}, {0x2001a134, 4, 0x2001a468}, {0x2001a468, 1, 0xc3}, {0x2001a138, 4, 0x2001a470}, {0x2001a470, 1, 0xc3}, {0x2001a13c, 4, 0x2001a478}, {0x2001a478, 1, 0xc3}, {0x2001a140, 4, 0x2001a480}, {0x2001a480, 1, 0xc3}, {0x2001a144, 4, 0x2001a488}, {0x2001a488, 1, 0xc3}, {0x2001a148, 4, 0x2001a490}, {0x2001a490, 1, 0xc3}, {0x2001a14c, 4, 0x2001a498}, {0x2001a498, 1, 0xc3}, {0x2001a150, 4, 0x2001a4a0}, {0x2001a4a0, 1, 0xc3}, {0x2001a154, 4, 0x2001a4a8}, {0x2001a4a8, 1, 0xc3}, {0x2001a158, 4, 0x2001a4b0}, {0x2001a4b0, 1, 0xc3}, {0x2001a15c, 4, 0x2001a4b8}, {0x2001a4b8, 1, 0xc3}, {0x2001a160, 4, 0x2001a4c0}, {0x2001a4c0, 1, 0xc3}, {0x2001a164, 4, 0x2001a4c8}, {0x2001a4c8, 1, 0xc3}, {0x2001a168, 4, 0x2001a4d0}, {0x2001a4d0, 1, 0xc3}, {0x2001a16c, 4, 0x2001a4d8}, {0x2001a4d8, 1, 0xc3}, {0x2001a170, 4, 0x2001a4e0}, {0x2001a4e0, 1, 0xc3}, {0x2001a174, 4, 0x2001a4e8}, {0x2001a4e8, 1, 0xc3}, {0x2001a178, 4, 0x2001a4f0}, {0x2001a4f0, 1, 0xc3}, {0x2001a17c, 4, 0x2001a4f8}, {0x2001a4f8, 1, 0xc3}, {0x2001a180, 4, 0x2001a500}, {0x2001a500, 1, 0xc3}, {0x2001a184, 4, 0x2001a508}, {0x2001a508, 1, 0xc3}, {0x2001a188, 4, 0x2001a510}, {0x2001a510, 1, 0xc3}, {0x2001a18c, 4, 0x2001a518}, {0x2001a518, 1, 0xc3}, {0x2001a190, 4, 0x2001a520}, {0x2001a520, 1, 0xc3}, {0x2001a194, 4, 0x2001a528}, {0x2001a528, 1, 0xc3}, {0x2001a198, 4, 0x2001a530}, {0x2001a530, 1, 0xc3}, {0x2001a19c, 4, 0x2001a538}, {0x2001a538, 1, 0xc3}, {0x2001a1a0, 4, 0x2001a540}, {0x2001a540, 1, 0xc3}, {0x2001a1a4, 4, 0x2001a548}, {0x2001a548, 1, 0xc3}, {0x2001a1a8, 4, 0x2001a550}, {0x2001a550, 1, 0xc3}, {0x2001a1ac, 4, 0x2001a558}, {0x2001a558, 1, 0xc3}, {0x2001a1b0, 4, 0x2001a560}, {0x2001a560, 1, 0xc3}, {0x2001a1b4, 4, 0x2001a568}, {0x2001a568, 1, 0xc3}, {0x2001a1b8, 4, 0x2001a570}, {0x2001a570, 1, 0xc3}, {0x2001a1bc, 4, 0x2001a578}, {0x2001a578, 1, 0xc3}, {0x2001a1c0, 4, 0x2001a580}, {0x2001a580, 1, 0xc3}, {0x2001a1c4, 4, 0x2001a588}, {0x2001a588, 1, 0xc3}, {0x2001a1c8, 4, 0x2001a590}, {0x2001a590, 1, 0xc3}, {0x2001a1cc, 4, 0x2001a598}, {0x2001a598, 1, 0xc3}, {0x2001a1d0, 4, 0x2001a5a0}, {0x2001a5a0, 1, 0xc3}, {0x2001a1d4, 4, 0x2001a5a8}, {0x2001a5a8, 1, 0xc3}, {0x2001a1d8, 4, 0x2001a5b0}, {0x2001a5b0, 1, 0xc3}, {0x2001a1dc, 4, 0x2001a5b8}, {0x2001a5b8, 1, 0xc3}, {0x2001a1e0, 4, 0x2001a5c0}, {0x2001a5c0, 1, 0xc3}, {0x2001a1e4, 4, 0x2001a5c8}, {0x2001a5c8, 1, 0xc3}, {0x2001a1e8, 4, 0x2001a5d0}, {0x2001a5d0, 1, 0xc3}, {0x2001a1ec, 4, 0x2001a5d8}, {0x2001a5d8, 1, 0xc3}, {0x2001a1f0, 4, 0x2001a5e0}, {0x2001a5e0, 1, 0xc3}, {0x2001a1f4, 4, 0x2001a5e8}, {0x2001a5e8, 1, 0xc3}, {0x2001a1f8, 4, 0x2001a5f0}, {0x2001a5f0, 1, 0xc3}, {0x2001a1fc, 4, 0x2001a5f8}, {0x2001a5f8, 1, 0xc3}, {0x6eb704, 4, 0x20032000}, {0x20032004, 4, 0x20032100}, {0x6eb6a0, 4, 0x20032200}, {0x6eb6a4, 4, 0x20032200}, {0x6eb6a8, 4, 0x20032200}, {0x6eb6ac, 4, 0x20032200}, {0x6eb6b0, 4, 0x20032200}, {0x6eb6b4, 4, 0x20032200}, {0x6eb6b8, 4, 0x20032200}, {0x6eb6bc, 4, 0x20032200}, {0x6eb6c0, 4, 0x20032200}, {0x6eb6c4, 4, 0x20032200}, {0x6eb6c8, 4, 0x20032200}, {0x6eb6cc, 4, 0x20032200}, {0x6eb6d0, 4, 0x20032200}, {0x6eb6d4, 4, 0x20032200}, {0x6eb6d8, 4, 0x20032200}, {0x6eb6dc, 4, 0x20032200}, {0x20032218, 4, 0x20032300}, {0x683fad, 4, 0x20040000}, {0x20040000, 4, 0x20040100}, {0x20040100, 4, 0x20040200}, {0x2004020c, 4, 0x20041000}, {0x200411d8, 4, 0x20041400}, {0x20040104, 4, 0x20040240}, {0x2004024c, 4, 0x20041800}, {0x200419d8, 4, 0x20041c00}, {0x659688, 4, 0x2001a000}, {0x659690, 4, 0x2001a000}, {0x659b3c, 4, 0x20050000}, {0x20050000, 4, 0x20050100}, {0x20050004, 4, 0x20050300}}
var cgInNames = []string{"f94", "f95", "f96", "f97", "f98", "f99", "f9a", "f9c", "f9d", "sel", "f517c", "mode", "p683fe7", "p683fdb", "p683fdf", "p68407f", "p684083", "p684087", "p6840af", "f685115", "f68510c", "p6850e5", "f6874ca", "p6874cb", "f652da6", "p69b9a0", "p69b9a4", "f6840bc", "f6840bd", "f683fcd", "uip", "rec2.0", "rec2.1", "rec2.2", "rec2.3", "rec2.4", "rec2.5", "rec2.6", "rec2.7", "spc.100", "spc.104", "g69c.f8", "ui.f4", "ui.154", "ui.2c0", "ui.2c8", "ui.2ca", "ui.2f0", "ui.2f4", "ui.2f8", "ui.327", "cb.0", "cb.8", "cb.7b4", "it0.spell", "it0.type", "it1.spell", "it1.type", "it2.spell", "it2.type", "it3.spell", "it3.type", "sp1.kind", "sp2.kind", "sp3.kind", "sp4.kind", "sp5.kind", "sp6.kind", "sp7.kind", "sp8.kind", "sp9.kind", "sp88.kind", "sp119.kind", "ip0", "ip1", "ip2", "ip3", "p4.0", "p4.1", "s0.p", "s0.t0", "s0.id0", "s0.ip0", "s0.r1d.0", "s0.r1d.1", "s0.r1d.2", "s0.r1d.3", "s0.r1d.4", "s0.r1d.5", "s0.r1d.6", "s0.r1d.7", "s0.r4d.0", "s0.r4d.1", "s0.r4d.2", "s0.r4d.3", "s0.r4d.4", "s0.r4d.5", "s0.r4d.6", "s0.r4d.7", "s0.r81.0", "s0.r81.1", "s0.r81.2", "s0.r81.3", "s0.r81.4", "s0.r81.5", "s0.r81.6", "s0.r81.7", "s0.ra9.0", "s0.ra9.1", "s0.ra9.2", "s0.ra9.3", "s0.ra9.4", "s0.ra9.5", "s0.ra9.6", "s0.ra9.7", "s0.rd1.0", "s0.rd1.1", "s0.rd1.2", "s0.rd1.3", "s0.rd1.4", "s0.rd1.5", "s0.rd1.6", "s0.rd1.7", "s0.act", "s0.id1", "s0.a45", "s0.a49", "s0.a6d", "s0.a71", "s0.id3", "s0.ip4", "s0.f5", "s0.f104", "s0.f105", "c0.p", "c0.b11", "c0.b01", "c0.k0", "c0.k1", "c0.k2", "c0.k3", "c0.k4", "c0.k5", "c0.k6", "c0.k7", "c0.k8", "c0.k9", "c0.k10", "c0.k11", "cb0.18", "cb0.b4", "cb0.4c", "cb0.99", "cb0.ac", "cb0.bf", "pb0", "pc0", "s1.p", "s1.t0", "s1.id0", "s1.ip0", "s1.r1d.0", "s1.r1d.1", "s1.r1d.2", "s1.r1d.3", "s1.r1d.4", "s1.r1d.5", "s1.r1d.6", "s1.r1d.7", "s1.r4d.0", "s1.r4d.1", "s1.r4d.2", "s1.r4d.3", "s1.r4d.4", "s1.r4d.5", "s1.r4d.6", "s1.r4d.7", "s1.r81.0", "s1.r81.1", "s1.r81.2", "s1.r81.3", "s1.r81.4", "s1.r81.5", "s1.r81.6", "s1.r81.7", "s1.ra9.0", "s1.ra9.1", "s1.ra9.2", "s1.ra9.3", "s1.ra9.4", "s1.ra9.5", "s1.ra9.6", "s1.ra9.7", "s1.rd1.0", "s1.rd1.1", "s1.rd1.2", "s1.rd1.3", "s1.rd1.4", "s1.rd1.5", "s1.rd1.6", "s1.rd1.7", "s1.act", "s1.id1", "s1.a45", "s1.a49", "s1.a6d", "s1.a71", "s1.id3", "s1.ip4", "s1.f5", "s1.f104", "s1.f105", "c1.p", "c1.b11", "c1.b01", "c1.k0", "c1.k1", "c1.k2", "c1.k3", "c1.k4", "c1.k5", "c1.k6", "c1.k7", "c1.k8", "c1.k9", "c1.k10", "c1.k11", "cb1.18", "cb1.b4", "cb1.4c", "cb1.99", "cb1.ac", "cb1.bf", "pb1", "pc1", "s2.p", "s2.t0", "s2.id0", "s2.ip0", "s2.r1d.0", "s2.r1d.1", "s2.r1d.2", "s2.r1d.3", "s2.r1d.4", "s2.r1d.5", "s2.r1d.6", "s2.r1d.7", "s2.r4d.0", "s2.r4d.1", "s2.r4d.2", "s2.r4d.3", "s2.r4d.4", "s2.r4d.5", "s2.r4d.6", "s2.r4d.7", "s2.r81.0", "s2.r81.1", "s2.r81.2", "s2.r81.3", "s2.r81.4", "s2.r81.5", "s2.r81.6", "s2.r81.7", "s2.ra9.0", "s2.ra9.1", "s2.ra9.2", "s2.ra9.3", "s2.ra9.4", "s2.ra9.5", "s2.ra9.6", "s2.ra9.7", "s2.rd1.0", "s2.rd1.1", "s2.rd1.2", "s2.rd1.3", "s2.rd1.4", "s2.rd1.5", "s2.rd1.6", "s2.rd1.7", "s2.act", "s2.id1", "s2.a45", "s2.a49", "s2.a6d", "s2.a71", "s2.id3", "s2.ip4", "s2.f5", "s2.f104", "s2.f105", "c2.p", "c2.b11", "c2.b01", "c2.k0", "c2.k1", "c2.k2", "c2.k3", "c2.k4", "c2.k5", "c2.k6", "c2.k7", "c2.k8", "c2.k9", "c2.k10", "c2.k11", "cb2.18", "cb2.b4", "cb2.4c", "cb2.99", "cb2.ac", "cb2.bf", "pb2", "pc2", "s3.p", "s3.t0", "s3.id0", "s3.ip0", "s3.r1d.0", "s3.r1d.1", "s3.r1d.2", "s3.r1d.3", "s3.r1d.4", "s3.r1d.5", "s3.r1d.6", "s3.r1d.7", "s3.r4d.0", "s3.r4d.1", "s3.r4d.2", "s3.r4d.3", "s3.r4d.4", "s3.r4d.5", "s3.r4d.6", "s3.r4d.7", "s3.r81.0", "s3.r81.1", "s3.r81.2", "s3.r81.3", "s3.r81.4", "s3.r81.5", "s3.r81.6", "s3.r81.7", "s3.ra9.0", "s3.ra9.1", "s3.ra9.2", "s3.ra9.3", "s3.ra9.4", "s3.ra9.5", "s3.ra9.6", "s3.ra9.7", "s3.rd1.0", "s3.rd1.1", "s3.rd1.2", "s3.rd1.3", "s3.rd1.4", "s3.rd1.5", "s3.rd1.6", "s3.rd1.7", "s3.act", "s3.id1", "s3.a45", "s3.a49", "s3.a6d", "s3.a71", "s3.id3", "s3.ip4", "s3.f5", "s3.f104", "s3.f105", "c3.p", "c3.b11", "c3.b01", "c3.k0", "c3.k1", "c3.k2", "c3.k3", "c3.k4", "c3.k5", "c3.k6", "c3.k7", "c3.k8", "c3.k9", "c3.k10", "c3.k11", "cb3.18", "cb3.b4", "cb3.4c", "cb3.99", "cb3.ac", "cb3.bf", "pb3", "pc3", "mon.n", "mon0.id", "mon0.v", "mon0.f", "mon0.m", "mon1.id", "mon1.v", "mon1.f", "mon1.m", "pl0.0", "pl0.1", "pl1.0", "pl1.1", "pl2.0", "pl2.1", "pl3.0", "pl3.1", "sr_4e79a0", "sr_536f60", "sr_5458a0", "sr_51d180", "sr_51d3b0", "sr_53c720", "sr_536a20", "sr_536190", "sr_537330", "sr_5194e0", "f60a394", "p65b9a8", "n659b34", "p652dac", "pm.0", "pm.40", "pm.54", "pm.64", "pm.78", "pm.7c", "pm.84", "pm.a0", "pm.a8", "mv0.25", "mv0.68", "mv0.f4", "mv0.f8", "mv0.fc", "mv1.25", "mv1.68", "mv1.f4", "mv1.f8", "mv1.fc", "tg0.28", "tg1.28"}
var cgOutNames = []string{"f94", "f95", "f96", "f97", "f98", "f99", "f9a", "f9c", "f9d", "sel", "f517c", "mode", "p683fe7", "p683fdb", "p683fdf", "p68407f", "p684083", "p684087", "p6840af", "f685115", "f68510c", "p6850e5", "f6874ca", "p6874cb", "f652da6", "p69b9a0", "p69b9a4", "f6840bc", "f6840bd", "f683fcd", "uip", "rec2.0", "rec2.1", "rec2.2", "rec2.3", "rec2.4", "rec2.5", "rec2.6", "rec2.7", "spc.100", "spc.104", "g69c.f8", "ui.f4", "ui.154", "ui.2c0", "ui.2c8", "ui.2ca", "ui.2f0", "ui.2f4", "ui.2f8", "ui.327", "cb.0", "cb.8", "cb.7b4", "it0.spell", "it0.type", "it1.spell", "it1.type", "it2.spell", "it2.type", "it3.spell", "it3.type", "sp1.kind", "sp2.kind", "sp3.kind", "sp4.kind", "sp5.kind", "sp6.kind", "sp7.kind", "sp8.kind", "sp9.kind", "sp88.kind", "sp119.kind", "ip0", "ip1", "ip2", "ip3", "p4.0", "p4.1", "s0.p", "s0.t0", "s0.id0", "s0.ip0", "s0.r1d.0", "s0.r1d.1", "s0.r1d.2", "s0.r1d.3", "s0.r1d.4", "s0.r1d.5", "s0.r1d.6", "s0.r1d.7", "s0.r4d.0", "s0.r4d.1", "s0.r4d.2", "s0.r4d.3", "s0.r4d.4", "s0.r4d.5", "s0.r4d.6", "s0.r4d.7", "s0.r81.0", "s0.r81.1", "s0.r81.2", "s0.r81.3", "s0.r81.4", "s0.r81.5", "s0.r81.6", "s0.r81.7", "s0.ra9.0", "s0.ra9.1", "s0.ra9.2", "s0.ra9.3", "s0.ra9.4", "s0.ra9.5", "s0.ra9.6", "s0.ra9.7", "s0.rd1.0", "s0.rd1.1", "s0.rd1.2", "s0.rd1.3", "s0.rd1.4", "s0.rd1.5", "s0.rd1.6", "s0.rd1.7", "s0.act", "s0.id1", "s0.a45", "s0.a49", "s0.a6d", "s0.a71", "s0.id3", "s0.ip4", "s0.f5", "s0.f104", "s0.f105", "c0.p", "c0.b11", "c0.b01", "c0.k0", "c0.k1", "c0.k2", "c0.k3", "c0.k4", "c0.k5", "c0.k6", "c0.k7", "c0.k8", "c0.k9", "c0.k10", "c0.k11", "cb0.18", "cb0.b4", "cb0.4c", "cb0.99", "cb0.ac", "cb0.bf", "pb0", "pc0", "s1.p", "s1.t0", "s1.id0", "s1.ip0", "s1.r1d.0", "s1.r1d.1", "s1.r1d.2", "s1.r1d.3", "s1.r1d.4", "s1.r1d.5", "s1.r1d.6", "s1.r1d.7", "s1.r4d.0", "s1.r4d.1", "s1.r4d.2", "s1.r4d.3", "s1.r4d.4", "s1.r4d.5", "s1.r4d.6", "s1.r4d.7", "s1.r81.0", "s1.r81.1", "s1.r81.2", "s1.r81.3", "s1.r81.4", "s1.r81.5", "s1.r81.6", "s1.r81.7", "s1.ra9.0", "s1.ra9.1", "s1.ra9.2", "s1.ra9.3", "s1.ra9.4", "s1.ra9.5", "s1.ra9.6", "s1.ra9.7", "s1.rd1.0", "s1.rd1.1", "s1.rd1.2", "s1.rd1.3", "s1.rd1.4", "s1.rd1.5", "s1.rd1.6", "s1.rd1.7", "s1.act", "s1.id1", "s1.a45", "s1.a49", "s1.a6d", "s1.a71", "s1.id3", "s1.ip4", "s1.f5", "s1.f104", "s1.f105", "c1.p", "c1.b11", "c1.b01", "c1.k0", "c1.k1", "c1.k2", "c1.k3", "c1.k4", "c1.k5", "c1.k6", "c1.k7", "c1.k8", "c1.k9", "c1.k10", "c1.k11", "cb1.18", "cb1.b4", "cb1.4c", "cb1.99", "cb1.ac", "cb1.bf", "pb1", "pc1", "s2.p", "s2.t0", "s2.id0", "s2.ip0", "s2.r1d.0", "s2.r1d.1", "s2.r1d.2", "s2.r1d.3", "s2.r1d.4", "s2.r1d.5", "s2.r1d.6", "s2.r1d.7", "s2.r4d.0", "s2.r4d.1", "s2.r4d.2", "s2.r4d.3", "s2.r4d.4", "s2.r4d.5", "s2.r4d.6", "s2.r4d.7", "s2.r81.0", "s2.r81.1", "s2.r81.2", "s2.r81.3", "s2.r81.4", "s2.r81.5", "s2.r81.6", "s2.r81.7", "s2.ra9.0", "s2.ra9.1", "s2.ra9.2", "s2.ra9.3", "s2.ra9.4", "s2.ra9.5", "s2.ra9.6", "s2.ra9.7", "s2.rd1.0", "s2.rd1.1", "s2.rd1.2", "s2.rd1.3", "s2.rd1.4", "s2.rd1.5", "s2.rd1.6", "s2.rd1.7", "s2.act", "s2.id1", "s2.a45", "s2.a49", "s2.a6d", "s2.a71", "s2.id3", "s2.ip4", "s2.f5", "s2.f104", "s2.f105", "c2.p", "c2.b11", "c2.b01", "c2.k0", "c2.k1", "c2.k2", "c2.k3", "c2.k4", "c2.k5", "c2.k6", "c2.k7", "c2.k8", "c2.k9", "c2.k10", "c2.k11", "cb2.18", "cb2.b4", "cb2.4c", "cb2.99", "cb2.ac", "cb2.bf", "pb2", "pc2", "s3.p", "s3.t0", "s3.id0", "s3.ip0", "s3.r1d.0", "s3.r1d.1", "s3.r1d.2", "s3.r1d.3", "s3.r1d.4", "s3.r1d.5", "s3.r1d.6", "s3.r1d.7", "s3.r4d.0", "s3.r4d.1", "s3.r4d.2", "s3.r4d.3", "s3.r4d.4", "s3.r4d.5", "s3.r4d.6", "s3.r4d.7", "s3.r81.0", "s3.r81.1", "s3.r81.2", "s3.r81.3", "s3.r81.4", "s3.r81.5", "s3.r81.6", "s3.r81.7", "s3.ra9.0", "s3.ra9.1", "s3.ra9.2", "s3.ra9.3", "s3.ra9.4", "s3.ra9.5", "s3.ra9.6", "s3.ra9.7", "s3.rd1.0", "s3.rd1.1", "s3.rd1.2", "s3.rd1.3", "s3.rd1.4", "s3.rd1.5", "s3.rd1.6", "s3.rd1.7", "s3.act", "s3.id1", "s3.a45", "s3.a49", "s3.a6d", "s3.a71", "s3.id3", "s3.ip4", "s3.f5", "s3.f104", "s3.f105", "c3.p", "c3.b11", "c3.b01", "c3.k0", "c3.k1", "c3.k2", "c3.k3", "c3.k4", "c3.k5", "c3.k6", "c3.k7", "c3.k8", "c3.k9", "c3.k10", "c3.k11", "cb3.18", "cb3.b4", "cb3.4c", "cb3.99", "cb3.ac", "cb3.bf", "pb3", "pc3", "mon.n", "mon0.id", "mon0.v", "mon0.f", "mon0.m", "mon1.id", "mon1.v", "mon1.f", "mon1.m", "pl0.0", "pl0.1", "pl1.0", "pl1.1", "pl2.0", "pl2.1", "pl3.0", "pl3.1", "cnt_4e79a0", "rec_4e79a0_1", "rec_4e79a0_2", "rec_4e79a0_3", "rec_4e79a0_4", "cnt_536f60", "rec_536f60_1", "rec_536f60_2", "cnt_4f0480", "rec_4f0480_1", "rec_4f0480_2", "cnt_4ee9d0", "rec_4ee9d0_1", "cnt_5458a0", "rec_5458a0_1", "cnt_5201b0", "rec_5201b0_1", "rec_5201b0_2", "cnt_4ee220", "rec_4ee220_1", "cnt_50e5c0", "rec_50e5c0_1", "cnt_51d180", "rec_51d180_1", "cnt_51d3b0", "rec_51d3b0_1", "rec_51d3b0_2", "rec_51d3b0_3", "cnt_53c720", "rec_53c720_1", "rec_53c720_2", "rec_53c720_3", "cnt_536a20", "rec_536a20_1", "rec_536a20_2", "rec_536a20_3", "cnt_536190", "rec_536190_1", "cnt_537330", "rec_537330_1", "cnt_4fd030", "rec_4fd030_1", "rec_4fd030_2", "rec_4fd030_5", "rec_4fd030_7", "cnt_4add30", "rec_4add30_1", "cnt_593330", "cnt_595600", "cnt_5194e0", "rec_5194e0_1", "rec_5194e0_2", "rec_5194e0_3", "rec_5194e0_4", "rec_5194e0_5", "cnt_58aad0", "rec_58aad0_1", "rec_58aad0_2", "rec_58aad0_3", "f60a394", "p65b9a8", "n659b34", "p652dac", "pm.0", "pm.40", "pm.54", "pm.64", "pm.78", "pm.7c", "pm.84", "pm.a0", "pm.a8", "mv0.25", "mv0.68", "mv0.f4", "mv0.f8", "mv0.fc", "mv1.25", "mv1.68", "mv1.f4", "mv1.f8", "mv1.fc", "tg0.28", "tg1.28", "cnt_55ee70", "rec_55ee70_1", "cnt_420b40", "rec_420b40_1", "cnt_4f21b0", "rec_4f21b0_1", "cnt_4f23d0", "rec_4f23d0_1", "cnt_5a1950", "cnt_4263f0", "rec_4263f0_1", "rec_4263f0_2", "rec_4263f0_3", "rec_4263f0_4", "cnt_422d50", "rec_422d50_1", "rec_422d50_2", "rec_422d50_3", "rec_422d50_4", "rec_422d50_5", "cnt_5a19a0", "cnt_4f2380", "rec_4f2380_1", "cnt_4f2040", "rec_4f2040_1"}
