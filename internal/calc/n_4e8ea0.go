package calc

import "math"

// Node 0x4e8ea0: the per-frame combat phase driver (monster_repeat_action_phase).
// Outside combat input it settles the pending spell effect object, then either
// advances the movement percentage (movement phase) or runs the frame gates:
// party auto-select of the next able member, the monster-still-busy scan, the
// pending sound handle, the effect lists and the party item use of the first
// two slots. It clears each living monster's redraw flag, then dispatches the
// action state of the combat block [0x6836a8]: start of an action, start of
// the actor's turn, the attack resolution of a party member, or the repeat
// stage (a party member's action record is released and rebuilt; a monster
// pays its fatigue cost, may pick its action again and moves its phase on).
// With no running action it settles defeat and starts the next round.
//
// It runs over the case's byte world (cgW cells). The D4 callees are address
// stubs in the case; Emu replays their returns and writes from case inputs.
// F runs the real Go nodes of the D4 callees whose results the node reads
// (0x4e79a0, 0x524a10, 0x547310, 0x4e9f90, 0x4ebfe0, 0x53e250, 0x52c240,
// 0x5323f0, 0x523c00, 0x4f0480) on parked rows of their own cases with the
// live values the original hands them; the other D4 callees (0x4e6ce0,
// 0x4e8000, 0x4e8370, 0x4e9490, 0x4ea5c0, 0x4eac90, 0x4eae20, 0x4eb5a0,
// 0x5049c0) and the registered-out ones only record their calls.
// Fixture caps: two monsters, monster templates 0..3, two effect list entries,
// two 0x500e50 entries, three item ids, two groups per group list, two members
// per group, four pending ids per party slot, 0x80 bytes of heap.

type mrapStub struct {
	cnt, ret, retN int64
	rec            [][4]int64 // {argument (1-based), first record address, records, size}
	wr             [][3]int64 // {destination, size, source address}
}

type mrapW struct {
	lsW
	heap int64
	real bool
}

func (w *mrapW) call(k int, args ...int64) int64 {
	s := &mrapStubs[k]
	n := w.u32(s.cnt)
	for _, r := range s.rec {
		if n >= r[2] {
			panic("monster repeat action phase: more stubbed calls than the case records")
		}
		w.wr(r[1]+r[3]*n, int(r[3]), args[r[0]-1])
	}
	w.p32(s.cnt, n+1)
	return n
}

func (w *mrapW) ret(k int, n int64) int64 {
	s := &mrapStubs[k]
	if s.retN == 0 {
		return w.u32(s.ret)
	}
	if n >= s.retN {
		panic("monster repeat action phase: more stubbed calls than the case returns")
	}
	return w.u32(s.ret + 4*n)
}

func (w *mrapW) writes(k int) {
	for _, x := range mrapStubs[k].wr {
		w.wr(x[0], int(x[1]), w.rd(x[2], int(x[1])))
	}
}

// stub records one call and replays its return and writes (Emu, and the
// callees F does not run).
func (w *mrapW) stub(k int, args ...int64) int64 {
	n := w.call(k, args...)
	v := int64(0)
	if mrapStubs[k].ret != 0 {
		v = w.ret(k, n)
	}
	w.writes(k)
	return v
}

func (w *mrapW) tick() int64 { return w.u32(0x6eb70c) }

// left is 0x406c00(x): time left until tick x.
func (w *mrapW) left(x int64) int64 {
	if t := w.tick(); x > t {
		return x - t
	}
	return 0
}

// tmpl is 0x4e5720(m): the template cache entry of monster m.
func (w *mrapW) tmpl(m int64) int64 {
	if m == 0 {
		panic("monster repeat action phase: 0x4e5720 asserts on a NULL monster")
	}
	id := w.u32(m + 8)
	if id >= 1000 {
		panic("monster repeat action phase: 0x4e5720 asserts on a template id past 999")
	}
	t := w.u32(0x6840c7 + 4*id)
	if t == 0 {
		panic("monster repeat action phase: template cache miss is outside the fixture (file load)")
	}
	return t
}

// fxBusy is 0x4a5790(e): the effect object has not finished (flag +0x1e0 clear, animation state not 1).
func (w *mrapW) fxBusy(e int64) bool {
	if w.u8(e+0x1e0) != 0 {
		return false
	}
	return int8(w.u8(w.u32(e+0x1dc)+0xa4)) != 1
}

// fxNext is 0x4a2760(reset): walk the effect list [[0x659ab4]+0xb8].
func (w *mrapW) fxNext(reset bool) int64 {
	p := w.u32(0x659ab4)
	if p == 0 {
		return 0
	}
	L := w.u32(p + 0xb8)
	if L == 0 {
		return 0
	}
	if reset {
		w.p32(0x65bde4, 0)
	}
	c, n := w.i32(0x65bde4), w.i32(L+4)
	if n > 2 {
		panic("monster repeat action phase: effect list past the two-entry fixture")
	}
	if c < n {
		w.p32(0x65bde4, c+1)
		return w.u32(w.u32(L+0xc) + 4*c)
	}
	return 0
}

// sound is 0x408ef0(h): whether the sound handle h is still playing.
func (w *mrapW) sound(h int64) bool {
	if w.u8(0x650e50) == 0 {
		return false
	}
	for i := int64(0); i < 32; i++ {
		b := 0x6e4128 + 0x4c*i
		if w.u32(b+0x10) != h {
			continue
		}
		v := int64(2)
		for k, imp := range []int{mrapSAILS, mrapSAILT, mrapSAIL3} {
			if x := w.u32(b + 4*int64(k)); x != 0 {
				v = w.stub(imp, x)
			}
		}
		return v != 2 && v != 8
	}
	return false
}

// ready is 0x500e50: every entry of [0x689b64] has its +4 set.
func (w *mrapW) ready() bool {
	n := w.i32(0x689b5c)
	if n > 2 {
		panic("monster repeat action phase: 0x500e50 list past the two-entry fixture")
	}
	a := w.u32(0x689b64)
	for i := int64(0); i < n; i++ {
		if w.u32(w.u32(a+4*i)+4) == 0 {
			return false
		}
	}
	return true
}

// actorKind is 0x4c5b40(o, sel) for the selectors the node uses.
func (w *mrapW) actorKind(o int64) int64 {
	if o == 0 {
		return -1
	}
	return int64(int8(w.u8(w.u32(o+0x1d8) + 0xa4)))
}

// busy is 0x4e8c30: some monster is still in the middle of its action.
func (w *mrapW) busy() bool {
	G := w.A()
	eps := math.Float32frombits(uint32(vauExeU32(0x5ebc84))) // 0.001f
	for i := int64(0); i < w.monCount(); i++ {
		e := w.entry(i)
		o := w.u32(e + 0xc)
		t := w.actorKind(o)
		if t == 0x15 && w.rd(w.tmpl(e)+0x187, 2) == 0x234 {
			return true
		}
		if w.u8(e+0x15) == 0 {
			continue
		}
		if o == 0 {
			panic("monster repeat action phase: monster without a visual is outside the fixture")
		}
		A := w.u32(o + 0x1d8)
		if w.u8(A+0xa7) != 0xff { // 0x4c5710
			return true
		}
		if w.u32(o+0x1dc)>>3&1 != 0 { // 0x4c5ee0
			return true
		}
		if w.u8(A+0x6d) != 0 && t != 1 && t != 2 && t != 4 && t != 0x15 { // 0x4c59e0
			if w.u32(G+0x7b0) != 2 || w.u32(G+0x7b8) != e {
				return true
			}
			switch w.kindClass(t) {
			case 6:
				if int64(vauExeU32(uint32(w.u32(A)+0xc))) != 0x4bf970 {
					panic("monster repeat action phase: actor vtable slot +0xc is not 0x4bf970")
				}
				k0 := int64(int32(w.stub(mrapS4BF970, w.u8(A+0xa4)))) // 0x4c4660 selector 0
				if w.u8(A+0x64) < (k0-(k0>>31))>>1 {                  // selector 4
					return true
				}
			case 7:
				return w.u8(e+0x2df) == 0
			default:
				return true
			}
		}
		f := float32(float64(w.f32(o+0xf0)) - float64(w.f32(o+0xec)))
		if d := float32(math.Abs(float64(f))); !(d < eps) && d == d && w.u32(e+0x107) < 0xe {
			return true
		}
	}
	return false
}

// kindClass is 0x5430c0(t) over the exe jump table 0x5430e0.
func (w *mrapW) kindClass(t int64) int64 {
	i := uint32(t - 9)
	if i > 8 {
		return t
	}
	switch vauExeU32(0x5430e0 + 4*i) {
	case 0x5430d3:
		return 6
	case 0x5430d9:
		return 7
	case 0x5430de:
		return t
	}
	panic("monster repeat action phase: 0x5430c0 jump table target outside the known cases")
}

func (w *mrapW) elig(x int64) bool {
	n := w.call(mrapS4E79A0, x, 1, 0, 0)
	if !w.real {
		return lsB(w.ret(mrapS4E79A0, n))
	}
	v := append([]int64(nil), lsPark4e79a0[:]...)
	v[0], v[1] = x, 1
	for j := int64(0); j < 8; j++ {
		v[30+3*j], v[31+3*j], v[32+3*j] = w.u8(w.S(j)), w.i32(w.C(j)+0xb11), w.u32(w.C(j)+0xb01)
	}
	v[54], v[55] = w.i32(w.C(x)+0x69), w.i32(w.C(x)+0x71)
	return lsB(pcActionEligibilityNode(v, w.r)[0])
}

func (w *mrapW) able(x int64) bool {
	n := w.call(mrapS524A10, x)
	if !w.real {
		return lsB(w.ret(mrapS524A10, n))
	}
	return lsB(characterActionEligibilityNode([]int64{x, w.u8(w.S(x)), w.u32(w.C(x) + 0xb11), w.u32(w.C(x) + 0xb01)}, w.r)[0])
}

func (w *mrapW) off(x int64) bool {
	S := w.S(x)
	n := w.call(mrapS547310, w.C(x), w.i32(S+0x3d), w.i32(S+0x41), S+0x45)
	if !w.real {
		return lsB(w.ret(mrapS547310, n))
	}
	v := append([]int64(nil), lsPark547310[:]...)
	v[0], v[1] = w.i32(S+0x3d), w.i32(S+0x41)
	v[4], v[5] = w.i32(w.C(x)+0x69), w.i32(w.C(x)+0x71)
	return lsB(offensiveActionAllowedNode(v, w.r)[0])
}

// autoSelect is 0x4e8da0: hand the selection to the next member able to act.
func (w *mrapW) autoSelect() {
	if w.u8(0x6850d3) == 0 || w.u8(w.uip()+0x154) != 0 || w.u8(w.A()+1) == 0 ||
		w.u32(0x683fe7) != 0 || w.u8(0x683f95) != 0 || w.u8(0x683f96) != 0 {
		return
	}
	sel := w.sel()
	if !w.elig(sel) && w.able(sel) {
		return
	}
	if sel == -1 {
		panic("monster repeat action phase: 0x4e8da0 asserts on no selected member")
	}
	x := sel + 1
	if uint32(x) >= 8 {
		x = 0
	}
	for x != sel {
		if !w.elig(x) && w.able(x) {
			if sel != x {
				w.selectMember(x)
			}
			return
		}
		if x++; uint32(x) >= 8 {
			x = 0
		}
	}
}

// selectMember is 0x565740(x).
func (w *mrapW) selectMember(x int64) {
	if w.u8(w.S(x)) == 0 {
		panic("monster repeat action phase: 0x565740 asserts on an absent member")
	}
	if w.u32(w.C(x)+0xb01) >= 0x13 {
		return
	}
	if w.sel() == x {
		w.stub(mrapS5554A0, x)
		return
	}
	if !w.f94() {
		w.stub(mrapS52E690, w.C(x), w.u32(0x68c568), 0, int64(vauExeU32(0x5ed8c8)), int64(vauExeU32(0x5ed914)))
	}
	old := w.sel()
	if old != -1 {
		w.refresh(old)
	}
	w.p32(0x68518d, x)
	w.refresh(x)
	if old != -1 && w.selFlags() {
		panic("monster repeat action phase: 0x53b050 is unreachable here (0x4e8da0 gates on 0x683f95/96)")
	}
	u := w.uip()
	w.p32(u+0x214, 0x1f4+w.tick())
	w.p8(u+0x218, 1)
	if u != 0 {
		if w.f94() {
			w.p8(u+0x2c0, 1)
		}
		w.p8(u+0x2c8, 1)
	}
	w.stub(mrapS5554A0, x)
	if w.selFlags() {
		panic("monster repeat action phase: 0x59f490/0x59cc40 are unreachable here (0x4e8da0 gates on 0x683f95/96)")
	}
	sel := w.sel()
	if w.u8(0x683f97) != 0 {
		w.stub(mrapS56EE20, sel)
	}
	if w.u8(0x683f98) != 0 {
		w.stub(mrapS587A30)
	}
	if w.u8(0x683f99) != 0 {
		w.stub(mrapS58A860)
	}
	if w.u8(0x683f9a) != 0 {
		w.stub(mrapS5B3100, sel)
	}
	if w.rd(u+0x2ca, 2) != 0xffff {
		w.stub(mrapS596CF0)
	}
	if w.u8(u+0x314) != 0 && x != w.i32(0x64c1c8) {
		w.stub(mrapS592E60)
	}
	w.stub(mrapS53A320, 0)
	if w.mode() == 7 && u != 0 {
		w.p32(u+0xf4, w.u32(u+0xf4)|1<<uint(x)|0x201000)
	}
	w.p8(u+0x154, 1)
}

// item is 0x50b800(id).
func (w *mrapW) item(id int64) int64 {
	L := w.u32(0x689f94)
	if L == 0 {
		return 0
	}
	n := w.i32(L + 4)
	if id < 0 || n > 3 {
		panic("monster repeat action phase: item id or item count outside the fixture")
	}
	a := w.u32(L + 0xc)
	if id >= n {
		id = 0
	}
	if e := w.u32(a + 4*id); e != 0 && w.u8(e+0xc7) == 0 {
		return e
	}
	return 0
}

// group is 0x5101b0(j) over the list [l].
func (w *mrapW) group(l, j int64) int64 { return w.u32(w.u32(w.u32(l)) + 4*j) }

func (w *mrapW) groups(l int64) int64 {
	n := w.u32(w.u32(l) + 8)
	if n > 2 {
		panic("monster repeat action phase: group list past the two-group fixture")
	}
	return n
}

func (w *mrapW) hasLeader(kind int64) bool {
	for j := int64(0); j < w.groups(0x683fb1); j++ {
		g := w.group(0x683fb1, j)
		if w.u8(g+0x28) != 0 && w.u8(g+0x29) != 0 && w.u8(g+0x2a) == 1 {
			t := w.u32(g + 0x18)
			if t > 3 {
				panic("monster repeat action phase: group template outside the cache fixture")
			}
			if w.u32(w.u32(0x6840c7+4*t)+0x25f) == kind { // 0x510180
				return true
			}
		}
	}
	return false
}

func (w *mrapW) alloc(n int64) int64 {
	a := w.heap
	w.heap += max(8, (n+7)&^7)
	if w.heap > 0x50000080 {
		panic("monster repeat action phase: heap past the 0x80-byte fixture")
	}
	return a
}

// post is 0x5289b0(kind, x): queue a message event.
func (w *mrapW) post(kind, x int64) {
	m := w.alloc(0x24)
	for i := int64(0); i < 9; i++ {
		w.p32(m+4*i, 0)
	}
	w.p32(m, -1)
	w.p32(m+0xc, kind)
	w.p32(m+0x10, x)
	w.p32(m+0x1c, 0)
	w.p32(m+0x20, w.u32(0x68c4ac))
	cnt, buf := w.i32(0x68c4c0), w.u32(0x68c4c8)
	if cnt+1 > w.i32(0x68c4c4) {
		nb := w.alloc((cnt + 1) * 4)
		w.p32(0x68c4c8, nb)
		w.p32(0x68c4c4, cnt+1)
		for i := int64(0); i < cnt; i++ {
			w.p32(nb+4*i, w.u32(buf+4*i))
		}
		buf = nb
	}
	w.p32(buf+4*cnt, m)
	w.p32(0x68c4c0, cnt+1)
}

func (w *mrapW) wstr(a int64) []int64 {
	var s []int64
	for i := int64(0); ; i++ {
		c := w.rd(a+2*i, 2)
		if c == 0 {
			return s
		}
		s = append(s, c)
	}
}

// partyItem is 0x4ed710: the first two members use a pending item.
func (w *mrapW) partyItem() bool {
	if !w.f94() {
		return false
	}
	G := w.A()
	for i := int64(0); i < 2; i++ {
		S, C := w.S(i), w.C(i)
		if w.u8(S) == 0 || w.u32(C+0xb11) == 0 || w.u32(C+0xb01) >= 0xb || w.u8(G+0xa58+i) != 0 {
			continue
		}
		it := w.item(w.i32(S + 0xfa))
		if it == 0 {
			continue
		}
		hit := false
		if k := w.u8(w.u32(it+6) + 0x5f); k != 0 && k != 1 {
			hit = w.hasLeader(k)
		}
		if !hit && w.u8(it+0x2e) == 0x38 {
			hit = w.hasLeader(4)
		}
		if !hit || byte(w.stub(mrapS50B2F0, i, 0)) == 0 { // 0x50b2d0
			continue
		}
		w.p8(0x6875a5, 1) // 0x577520
		w.stub(mrapS41F0D0)
		w.stub(mrapS55EE70, 9)
		f := ticRead(0x617584, 12)
		if string([]byte{f[0], f[2], f[4], f[6], f[8], f[10]}) != "%s %s\x00" {
			panic("monster repeat action phase: format 0x617584 is not %s %s")
		}
		s := append(append(w.wstr(w.u32(it+6)+4), ' '), w.wstr(w.u32(w.u32(0x68c09c)+0x9c8))...)
		for k, c := range append(s, 0) {
			w.wr(0x689fd0+2*int64(k), 2, c)
		}
		w.stub(mrapS5299B0, 0x689fd0)
		w.post(0x3a, i)
		w.post(0x36, i)
		w.p8(G+0xa58+i, 1)
		return true
	}
	return false
}

// clearMark is 0x539630(x, id, 0) when the monster is listed.
func (w *mrapW) clearMark(x, id int64) {
	e := w.find(id, false)
	if e == -1 {
		return
	}
	a := w.u32(w.u32(w.entry(e)+0xc)+0x1d8) + 0x5bc
	w.p8(a, w.u8(a)&^(1<<uint(x)))
	w.stub(mrapS4C5EB0, x, id, 0)
}

// release is 0x53ac30(x, rec): clear member x's marks on the monsters its action targets.
func (w *mrapW) release(x, rec int64) {
	pc := 0x683794 + 0x118*x
	if n := w.i32(pc); n > 0 {
		if n > 4 {
			panic("monster repeat action phase: pending list past the four-id fixture")
		}
		l := w.u32(0x68379c + 0x118*x)
		for i := int64(0); i < n; i++ {
			w.clearMark(x, w.i32(l+4*i))
		}
		w.p32(pc, 0)
		return
	}
	r := w.rec(rec)
	if r[0] == 3 && r[2] != -1 {
		w.clearMark(x, r[2])
	}
	if r[0] != 4 || r[3] == -1 {
		return
	}
	g := int64(0)
	for _, l := range []int64{0x683fb1, 0x683fbd} {
		for j := int64(0); j < w.groups(l) && g == 0; j++ {
			if h := w.group(l, j); w.i32(h) == r[3] {
				g = h
			}
		}
		if g != 0 {
			break
		}
	}
	if g == 0 {
		return
	}
	L := w.u32(g + 8)
	n := w.u32(L + 8)
	if n > 2 {
		panic("monster repeat action phase: group member list past the two-member fixture")
	}
	for k := int64(0); k < n; k++ {
		w.clearMark(x, w.i32(w.u32(L)+4*k))
	}
}

func (w *mrapW) cost(m int64) int64 {
	n := w.call(mrapS52C240, m, 0)
	if !w.real {
		return w.ret(mrapS52C240, n)
	}
	return monsterActionFatigueCostNode([]int64{w.i32(m + 0x2e1), w.i32(m + 0x2e5), w.u32(m + 0x5f)}, w.r)[0]
}

// fatigue is 0x52c070(m, cost, 0).
func (w *mrapW) fatigue(m, cost int64) {
	if w.u32(m+0x2b) == 0 {
		return
	}
	if int32(cost) > 0 {
		w.tmpl(m)
		sp := w.u32(m + 0x33)
		if uint32(cost) >= uint32(sp) {
			w.p32(m+0x33, 0)
		} else {
			w.p32(m+0x33, sp-cost)
		}
	}
	sp := w.u32(m + 0x33)
	c := int32(100 - uint32(sp*100)/uint32(w.u32(m+0x2f)))
	lv := int64(0)
	switch {
	case c < 50:
	case c < 70:
		lv = 1
	case c < 85:
		lv = 2
	case c < 95:
		lv = 3
	default:
		lv = 4
	}
	w.p32(m+0x242, lv)
	if sp == 0 && w.u32(m+0x9b) < 0x270f {
		w.call(mrapS523C00, w.i32(m), 0x11, 0x270f, 0, 0, 1)
		if w.real {
			w.apply(m)
		} else {
			w.writes(mrapS523C00)
		}
	}
}

// apply runs the real 0x523c00(id, 0x11, 0x270f, 0, a5, 1) on the parked row
// of its case with this world mapped in by address (objects M, tpl, vis,
// actor, T, ui, G and the globals); its object outputs go back into the world.
func (w *mrapW) apply(m int64) {
	vis := w.u32(m + 0xc)
	where := func(a uint32) (int64, bool) {
		o := int64(a) - mstArena
		switch {
		case a < mstArena:
			return int64(a), a < mstStackBase
		case o < 0x400:
			return m + o, true
		case o >= 0x800 && o < 0xb00:
			return w.tmpl(m) + o - 0x800, true
		case o >= 0xb00 && o < 0xe40:
			return vis + o - 0xb00, true
		case o >= 0xe40 && o < 0xf40:
			return w.u32(vis+0x1d8) + o - 0xe40, true
		case o >= 0xf40 && o < 0x10c0:
			return w.u32(m+0x10) + o - 0xf40, true
		case o >= 0x1a00 && o < 0x1b00:
			return w.uip() + o - 0x1a00, true
		case o >= 0x4000 && o < mstArenaSize:
			return w.A() + o - 0x4000, true
		}
		return 0, false
	}
	set := func(a int64, n uint8) bool {
		for i := int64(0); i < int64(n); i++ {
			if _, ok := w.m[a+i]; ok {
				return true
			}
		}
		return false
	}
	x := append([]int64(nil), sdrApplyParked...)
	for i, f := range mcaSpecApply.in {
		switch mcaSpecApplyIn[i] {
		case "cond":
			x[i] = 0x11
		case "duration":
			x[i] = 0x270f
		case "penalty", "a5.type", "a5.f1b", "a5.f1c", "a5.f1e":
			x[i] = 0
		case "message":
			x[i] = 1
		case "a5.char", "a5.monster":
			x[i] = -1
		default:
			if a, ok := where(f.addr); ok && f.idx < 0 && set(a, f.size) {
				if f.signed {
					x[i] = w.rs(a, int(f.size))
				} else {
					x[i] = w.rd(a, int(f.size))
				}
			}
		}
	}
	o := monsterConditionApplyNode(x, w.r)
	for i, f := range mcaSpecApply.out {
		if a, ok := where(f.addr); ok && f.idx < 0 && f.addr >= mstArena {
			w.wr(a, int(f.size), o[i])
		}
	}
}

// pick is 0x5323f0(m, 1, 0, 0).
func (w *mrapW) pick(m int64) bool {
	n := w.call(mrapS5323F0, m, 1, 0, 0)
	if !w.real {
		v := w.ret(mrapS5323F0, n)
		w.writes(mrapS5323F0)
		return byte(v) != 0
	}
	T := w.u32(m + 0x10)
	x := append([]int64(nil), mchParkPick[:]...)
	x[0], x[1], x[2], x[3] = 1, 0, 0, 0
	x[4] = w.u8(w.tmpl(m) + 0xe5)
	x[8], x[9] = rpkListPtr, 0
	x[108], x[109], x[110] = w.u32(T+5), w.u32(T+9), w.u32(T+0x11)
	o := monsterActionRandomPickNode(x, w.r)
	w.p32(m+0x2e1, o[0])
	w.p32(m+0x2e5, o[1])
	for i := int64(0); i < 8; i++ {
		w.p32(m+0x2ba+4*i, o[2+i])
	}
	w.p32(T+5, o[10])
	w.p32(T+9, o[11])
	w.p32(T+0x11, o[12])
	return byte(o[13]) != 0
}

// round is 0x4f0480(T, [G+8]).
func (w *mrapW) round(T int64) {
	G := w.A()
	w.call(mrapS4F0480, T, w.u32(G+8))
	if !w.real {
		w.writes(mrapS4F0480)
		return
	}
	w.p32(T, combatPhaseRoundingNode([]int64{w.u32(G + 0x910), w.u32(T), w.u32(G + 8)}, w.r)[0])
}

// attack is 0x53e250(x) or 0x4ebfe0(x): the member's attack resolution.
func (w *mrapW) attack(k int, x int64) int64 {
	n := w.call(k, x)
	v := w.ret(k, n)
	w.writes(k) // the resolution's write of [G+0x7b4] comes from the case input in both
	if !w.real {
		return v
	}
	if k == mrapS4EBFE0 {
		p := append([]int64(nil), mrapPark4ebfe0[:]...)
		p[0] = x
		return breathStaminaCostNode(p, w.r)[0]
	}
	p := append([]int64(nil), mrapPark53e250[:]...)
	p[0] = x
	return meleeAttackResolutionNode(p, w.r)[0]
}

// defeat is 0x4e9f90(flag).
func (w *mrapW) defeat(flag int64) bool {
	n := w.call(mrapS4E9F90, flag)
	if !w.real {
		return byte(w.ret(mrapS4E9F90, n)) != 0
	}
	v := append([]int64(nil), mrapPark4e9f90[:]...)
	v[0] = flag
	for j := int64(0); j < 8; j++ {
		v[14+j], v[22+j], v[30+j] = w.u8(w.S(j)), w.u32(w.C(j)+0xb11), w.u32(w.C(j)+0xb01)
	}
	return byte(combatDefeatResolutionNode(v, w.r)[0]) != 0
}

// gates is the frame gate run outside the movement phase; true = the node returns.
func (w *mrapW) gates() bool {
	G := w.A()
	w.autoSelect()
	if w.u8(w.u32(0x65a0f8)+0x88) != 0 { // 0x420e10
		return true
	}
	if w.busy() {
		return true
	}
	if w.u8(G+0x7c0) != 0 {
		if w.sound(w.u32(G + 0x7bc)) {
			return true
		}
		w.p8(G+0x7c0, 0)
	}
	if !w.ready() {
		return true
	}
	for e := w.fxNext(true); e != 0; e = w.fxNext(false) {
		k := w.i32(e + 0x1d8)
		if e == w.u32(G+0x8c0) || w.u8(w.u32(0x65bde0)+0x1e5*k+0x154) != 0 {
			if w.fxBusy(e) {
				return true
			}
		}
	}
	if w.f94() {
		for i := int64(0); i < 2; i++ {
			if w.u8(w.S(i)) != 0 && w.u32(w.C(i)+0xb11) > 0 && w.u32(w.C(i)+0xb01) < 0xf && w.u8(G+0xa58+i) != 0 {
				return true
			}
		}
	}
	return false
}

func (w *mrapW) run() {
	if w.u8(0x6840bc) != 0 || w.u8(0x683fc5) != 0 {
		return
	}
	G := w.A()
	if w.u8(G+0xa48) == 0 && w.u32(0x683fa5) > 0 {
		w.stub(mrapS4E6CE0)
		w.p8(G+0xa48, 1)
	}
	if p := w.u32(G + 0x8c0); p != 0 && !w.fxBusy(p) {
		w.p8(p+0x1e2, 1)
		w.p32(G+0x8c0, 0)
	}
	if w.i32(G+0x918) == 1 {
		if m := w.u32(G + 0x910); m == 1 || m == 2 {
			w.stub(mrapS4F01D0)
		}
	} else if w.gates() {
		return
	}
	for i := int64(0); i < w.monCount(); i++ {
		e := w.entry(i)
		if w.u8(e+0x14) != 0 && w.u8(e+0x15) != 0 && w.u32(e+0x2b) > 0 && w.u8(w.u32(e+0x10)+0x151) != 0 {
			w.stub(mrapS5049C0, e, 1, 0) // 0x505760
			w.stub(mrapS5049C0, e, 0, 0)
			w.p8(w.u32(e+0x10)+0x151, 0)
		}
	}
	if w.u8(G) == 0 {
		w.idle()
		return
	}
	st := w.u32(G + 0x7b0)
	if st > 3 {
		return
	}
	switch vauExeU32(0x4e9460 + 4*uint32(st)) {
	case 0x4e90d1:
		if w.i32(G+0x7b4) != -1 || w.u32(G+0x7b8) != 0 {
			panic("monster repeat action phase: state 0 asserts on a running actor")
		}
		if w.partyItem() {
			return
		}
		w.stub(mrapS4E9490)
	case 0x4e912e:
		w.start()
	case 0x4e91be:
		w.resolve2()
	case 0x4e9256:
		w.repeat()
	default:
		panic("monster repeat action phase: state jump table target outside the known cases")
	}
}

// start is state 1: the actor's turn begins when the delay has run out.
func (w *mrapW) start() {
	G := w.A()
	if w.left(w.u32(G+0x7ac)) != 0 {
		return
	}
	if w.u8(0x685117) != 0 {
		w.p8(G+0xa57, 1)
	}
	if x := w.i32(G + 0x7b4); x != -1 {
		w.stub(mrapS4EA5C0, x)
		return
	}
	m := w.u32(G + 0x7b8)
	if m == 0 {
		panic("monster repeat action phase: state 1 asserts on no actor")
	}
	t := w.tmpl(m)
	w.stub(mrapS4EAE20, w.u32(G+0x7b8), t)
}

// resolve2 is state 2: a party member's attack (or a monster's action).
func (w *mrapW) resolve2() {
	G := w.A()
	x := w.i32(G + 0x7b4)
	if x == -1 {
		m := w.u32(G + 0x7b8)
		if m == 0 {
			panic("monster repeat action phase: state 2 asserts on no actor")
		}
		w.stub(mrapS4EB5A0, m)
		return
	}
	v, r := w.i32(w.S(x)+1), int64(3)
	switch {
	case v < 0:
	case v <= 1:
		r = w.attack(mrapS53E250, x)
	case v == 2:
		r = w.attack(mrapS4EBFE0, x)
	}
	if w.i32(G+0x7b4) == x && w.u32(G+0x7b0) != int64(uint32(r)) {
		w.p32(G+0x7b0, r)
	}
}

// repeat is state 3: the actor's action ends and is set up again.
func (w *mrapW) repeat() {
	G := w.A()
	if x := w.i32(G + 0x7b4); x != -1 {
		S := w.S(x)
		full := w.u8(G+0xd4*x+0xbf) == 0
		if k := w.u32(S + 0x3d); full && k <= 5 {
			switch vauExeU32(0x4e9470 + 4*uint32(k)) {
			case 0x4e92ff:
				full = false
			case 0x4e929d:
			default:
				panic("monster repeat action phase: action kind jump table target outside the known cases")
			}
		}
		if full {
			if w.resolve(x) != 2 {
				w.release(x, S+0x4d)
			}
			w.putRec(S+0x4d, cgRec{0, -1, -1, -1, 0, 0, 0, 0}) // 0x536170
			w.refresh(x)                                       // 0x55ee30
			if x == w.sel() {
				w.ui(0x20000) // 0x565420
			}
			w.stub(mrapS4E8000, x, w.i32(S+0x6d), w.i32(S+0x71), 0, 0)
		}
		w.stub(mrapS4EAC90, x)
		w.p32(G+0x7b0, 0)
		w.p32(G+0x7b4, -1)
		return
	}
	m := w.u32(G + 0x7b8)
	if m == 0 {
		panic("monster repeat action phase: state 3 asserts on no actor")
	}
	w.tmpl(m)
	w.fatigue(m, w.cost(m))
	w.stub(mrapS4C6200, w.u32(m+0xc), 1)
	w.p32(G+0x7b0, 0)
	w.p32(G+0x7b8, 0)
	T := w.u32(m + 0x10)
	if w.u8(T+4) == 0 {
		return
	}
	if w.u32(m+0x2e1) == 0 && w.u32(T+9) > 0 && w.pick(m) {
		v := uint32(w.u32(T)) + uint32(100-w.u32(G+8))/(uint32(w.u32(T+9))+1)
		w.p32(T, int64(v))
		if v <= 100 {
			w.round(T)
			return
		}
	}
	w.p32(T, 0)
}

// idle runs with no action in progress: defeat check, then the next round.
func (w *mrapW) idle() {
	G := w.A()
	if w.partyItem() {
		return
	}
	f := int64(0)
	if w.u8(G) == 0 && w.u8(G+0xa54) != 0 {
		f = 1
	}
	if w.defeat(f) {
		return
	}
	if w.u8(0x6850d2) != 0 && w.left(w.u32(G+0x7a8)) == 0 && w.any() || w.u8(G+0xa52) != 0 {
		w.stub(mrapS4E8370)
	}
}

func mrapNode(real bool) func([]int64, Rand) []int64 {
	return func(in []int64, r Rand) []int64 {
		if len(in) != len(mrapIn) {
			panic("monster repeat action phase: wrong input count")
		}
		w := &mrapW{lsW: lsW{cgW: cgW{m: map[int64]byte{}}, r: r}, heap: 0x50000000, real: real}
		w.c = &lsCallees{
			elig: func(_ *lsW, x int64) bool { return w.elig(x) },
			off:  func(_ *lsW, x int64) bool { return w.off(x) },
		}
		for _, s := range mrapInit {
			w.wr(s.a, s.s, s.v)
		}
		for i, s := range mrapIn {
			w.wr(s.a, abs(s.s), in[i])
		}
		w.run()
		out := make([]int64, len(mrapOut))
		for i, s := range mrapOut {
			if s.s < 0 {
				out[i] = w.rs(s.a, -s.s)
			} else {
				out[i] = w.rd(s.a, s.s)
			}
		}
		return out
	}
}

func monsterRepeatActionPhaseNode(in []int64, r Rand) []int64 { return mrapNode(true)(in, r) }

func init() {
	Nodes["004e8ea0"] = Node{Addr: "004e8ea0", In: mrapInNames, Out: mrapOutNames, F: monsterRepeatActionPhaseNode, Emu: mrapNode(false)}
}

// ---- generated by the ticket's scratch generator from the case layout ----

var mrapInit = []cgInitCell{{0x6836a8, 4, 0x20000000}, {0x68edcc, 4, 0x20001000}, {0x685178, 4, 0x20001400}, {0x685174, 4, 0x20002000}, {0x683fad, 4, 0x2000e400}, {0x65bde0, 4, 0x20014800}, {0x65a0f8, 4, 0x20014c00}, {0x689b64, 4, 0x20014d00}, {0x69c0f4, 4, 0x20014e00}, {0x68c4c8, 4, 0x20014f00}, {0x683fb1, 4, 0x20015500}, {0x683fbd, 4, 0x20015700}, {0x68c09c, 4, 0x2001a000}, {0x6840c7, 4, 0x20013000}, {0x6840cb, 4, 0x20013400}, {0x6840cf, 4, 0x20013800}, {0x6840d3, 4, 0x20013c00}, {0x2000e400, 4, 0x2000e410}, {0x2000e410, 4, 0x20010000}, {0x2001000c, 4, 0x20011400}, {0x20010010, 4, 0x20011000}, {0x200115d8, 4, 0x20012000}, {0x20012000, 4, 0x5ed200}, {0x20010242, 4, 0x5a5a0000}, {0x200102ba, 4, 0x5b5b0000}, {0x200102be, 4, 0x5b5b0001}, {0x200102c2, 4, 0x5b5b0002}, {0x200102c6, 4, 0x5b5b0003}, {0x200102ca, 4, 0x5b5b0004}, {0x200102ce, 4, 0x5b5b0005}, {0x200102d2, 4, 0x5b5b0006}, {0x200102d6, 4, 0x5b5b0007}, {0x2000e414, 4, 0x20010400}, {0x2001040c, 4, 0x20011600}, {0x20010410, 4, 0x20011200}, {0x200117d8, 4, 0x20012800}, {0x20012800, 4, 0x5ed200}, {0x20010642, 4, 0x5a5a0001}, {0x200106ba, 4, 0x5b5b0010}, {0x200106be, 4, 0x5b5b0011}, {0x200106c2, 4, 0x5b5b0012}, {0x200106c6, 4, 0x5b5b0013}, {0x200106ca, 4, 0x5b5b0014}, {0x200106ce, 4, 0x5b5b0015}, {0x200106d2, 4, 0x5b5b0016}, {0x200106d6, 4, 0x5b5b0017}, {0x20014710, 4, 0x20014000}, {0x200141dc, 4, 0x20014400}, {0x20014714, 4, 0x20014200}, {0x200143dc, 4, 0x20014500}, {0x2001470c, 4, 0x20014710}, {0x20014d00, 4, 0x20014d40}, {0x20014d04, 4, 0x20014d60}, {0x20014f00, 4, 0x5c5c0000}, {0x20014f04, 4, 0x5c5c0001}, {0x20014f08, 4, 0x5c5c0002}, {0x20014f0c, 4, 0x5c5c0003}, {0x20014f10, 4, 0x5c5c0004}, {0x20014f14, 4, 0x5c5c0005}, {0x20014f18, 4, 0x5c5c0006}, {0x20014f1c, 4, 0x5c5c0007}, {0x2001500c, 4, 0x20015010}, {0x20015010, 4, 0x20015100}, {0x20015014, 4, 0x20015200}, {0x20015018, 4, 0x0}, {0x20015106, 4, 0x20015300}, {0x20015304, 2, 0x41}, {0x20015306, 2, 0x62}, {0x20015308, 2, 0x0}, {0x20015206, 4, 0x20015400}, {0x20015404, 2, 0x43}, {0x20015406, 2, 0x64}, {0x20015408, 2, 0x65}, {0x2001540a, 2, 0x0}, {0x2001b000, 2, 0x46}, {0x2001b002, 2, 0x67}, {0x2001b004, 2, 0x0}, {0x2001a9c8, 4, 0x2001b000}, {0x20015500, 4, 0x20015510}, {0x20015510, 4, 0x20015600}, {0x20015514, 4, 0x20015680}, {0x20015700, 4, 0x20015710}, {0x20015710, 4, 0x20015780}, {0x20015714, 4, 0x20015800}, {0x20015608, 4, 0x20015880}, {0x20015880, 4, 0x20015890}, {0x20015688, 4, 0x200158c0}, {0x200158c0, 4, 0x200158d0}, {0x20015788, 4, 0x20015900}, {0x20015900, 4, 0x20015910}, {0x20015808, 4, 0x20015940}, {0x20015940, 4, 0x20015950}, {0x68379c, 4, 0x2001b100}, {0x20001451, 4, 0x5d5d0001}, {0x2000145d, 4, 0x5d5d0004}, {0x20001461, 4, 0x5d5d0005}, {0x20001465, 4, 0x5d5d0006}, {0x20001469, 4, 0x5d5d0007}, {0x6838b4, 4, 0x2001b110}, {0x20001557, 4, 0x5d5d0011}, {0x20001563, 4, 0x5d5d0014}, {0x20001567, 4, 0x5d5d0015}, {0x2000156b, 4, 0x5d5d0016}, {0x2000156f, 4, 0x5d5d0017}, {0x6839cc, 4, 0x2001b120}, {0x2000165d, 4, 0x5d5d0021}, {0x20001669, 4, 0x5d5d0024}, {0x2000166d, 4, 0x5d5d0025}, {0x20001671, 4, 0x5d5d0026}, {0x20001675, 4, 0x5d5d0027}, {0x683ae4, 4, 0x2001b130}, {0x20001763, 4, 0x5d5d0031}, {0x2000176f, 4, 0x5d5d0034}, {0x20001773, 4, 0x5d5d0035}, {0x20001777, 4, 0x5d5d0036}, {0x2000177b, 4, 0x5d5d0037}, {0x683bfc, 4, 0x2001b140}, {0x20001869, 4, 0x5d5d0041}, {0x20001875, 4, 0x5d5d0044}, {0x20001879, 4, 0x5d5d0045}, {0x2000187d, 4, 0x5d5d0046}, {0x20001881, 4, 0x5d5d0047}, {0x683d14, 4, 0x2001b150}, {0x2000196f, 4, 0x5d5d0051}, {0x2000197b, 4, 0x5d5d0054}, {0x2000197f, 4, 0x5d5d0055}, {0x20001983, 4, 0x5d5d0056}, {0x20001987, 4, 0x5d5d0057}, {0x683e2c, 4, 0x2001b160}, {0x20001a75, 4, 0x5d5d0061}, {0x20001a81, 4, 0x5d5d0064}, {0x20001a85, 4, 0x5d5d0065}, {0x20001a89, 4, 0x5d5d0066}, {0x20001a8d, 4, 0x5d5d0067}, {0x683f44, 4, 0x2001b170}, {0x20001b7b, 4, 0x5d5d0071}, {0x20001b87, 4, 0x5d5d0074}, {0x20001b8b, 4, 0x5d5d0075}, {0x20001b8f, 4, 0x5d5d0076}, {0x20001b93, 4, 0x5d5d0077}, {0x20001214, 4, 0x5e5e0001}, {0x20001218, 1, 0x5e}, {0x6875a5, 1, 0x0}, {0x689fd0, 2, 0x0}, {0x689fd2, 2, 0x0}, {0x689fd4, 2, 0x0}, {0x689fd6, 2, 0x0}, {0x689fd8, 2, 0x0}, {0x689fda, 2, 0x0}, {0x689fdc, 2, 0x0}, {0x689fde, 2, 0x0}, {0x689fe0, 2, 0x0}, {0x689fe2, 2, 0x0}, {0x689fe4, 2, 0x0}, {0x689fe6, 2, 0x0}, {0x689fe8, 2, 0x0}, {0x689fea, 2, 0x0}, {0x689fec, 2, 0x0}, {0x689fee, 2, 0x0}, {0x6e4268, 4, 0x0}, {0x6e4258, 4, 0x0}, {0x6e425c, 4, 0x0}, {0x6e4260, 4, 0x0}, {0x6e42b4, 4, 0x0}, {0x6e42a4, 4, 0x0}, {0x6e42a8, 4, 0x0}, {0x6e42ac, 4, 0x0}, {0x6e4300, 4, 0x0}, {0x6e42f0, 4, 0x0}, {0x6e42f4, 4, 0x0}, {0x6e42f8, 4, 0x0}, {0x6e434c, 4, 0x0}, {0x6e433c, 4, 0x0}, {0x6e4340, 4, 0x0}, {0x6e4344, 4, 0x0}, {0x6e4398, 4, 0x0}, {0x6e4388, 4, 0x0}, {0x6e438c, 4, 0x0}, {0x6e4390, 4, 0x0}, {0x6e43e4, 4, 0x0}, {0x6e43d4, 4, 0x0}, {0x6e43d8, 4, 0x0}, {0x6e43dc, 4, 0x0}, {0x6e4430, 4, 0x0}, {0x6e4420, 4, 0x0}, {0x6e4424, 4, 0x0}, {0x6e4428, 4, 0x0}, {0x6e447c, 4, 0x0}, {0x6e446c, 4, 0x0}, {0x6e4470, 4, 0x0}, {0x6e4474, 4, 0x0}, {0x6e44c8, 4, 0x0}, {0x6e44b8, 4, 0x0}, {0x6e44bc, 4, 0x0}, {0x6e44c0, 4, 0x0}, {0x6e4514, 4, 0x0}, {0x6e4504, 4, 0x0}, {0x6e4508, 4, 0x0}, {0x6e450c, 4, 0x0}, {0x6e4560, 4, 0x0}, {0x6e4550, 4, 0x0}, {0x6e4554, 4, 0x0}, {0x6e4558, 4, 0x0}, {0x6e45ac, 4, 0x0}, {0x6e459c, 4, 0x0}, {0x6e45a0, 4, 0x0}, {0x6e45a4, 4, 0x0}, {0x6e45f8, 4, 0x0}, {0x6e45e8, 4, 0x0}, {0x6e45ec, 4, 0x0}, {0x6e45f0, 4, 0x0}, {0x6e4644, 4, 0x0}, {0x6e4634, 4, 0x0}, {0x6e4638, 4, 0x0}, {0x6e463c, 4, 0x0}, {0x6e4690, 4, 0x0}, {0x6e4680, 4, 0x0}, {0x6e4684, 4, 0x0}, {0x6e4688, 4, 0x0}, {0x6e46dc, 4, 0x0}, {0x6e46cc, 4, 0x0}, {0x6e46d0, 4, 0x0}, {0x6e46d4, 4, 0x0}, {0x6e4728, 4, 0x0}, {0x6e4718, 4, 0x0}, {0x6e471c, 4, 0x0}, {0x6e4720, 4, 0x0}, {0x6e4774, 4, 0x0}, {0x6e4764, 4, 0x0}, {0x6e4768, 4, 0x0}, {0x6e476c, 4, 0x0}, {0x6e47c0, 4, 0x0}, {0x6e47b0, 4, 0x0}, {0x6e47b4, 4, 0x0}, {0x6e47b8, 4, 0x0}, {0x6e480c, 4, 0x0}, {0x6e47fc, 4, 0x0}, {0x6e4800, 4, 0x0}, {0x6e4804, 4, 0x0}, {0x6e4858, 4, 0x0}, {0x6e4848, 4, 0x0}, {0x6e484c, 4, 0x0}, {0x6e4850, 4, 0x0}, {0x6e48a4, 4, 0x0}, {0x6e4894, 4, 0x0}, {0x6e4898, 4, 0x0}, {0x6e489c, 4, 0x0}, {0x6e48f0, 4, 0x0}, {0x6e48e0, 4, 0x0}, {0x6e48e4, 4, 0x0}, {0x6e48e8, 4, 0x0}, {0x6e493c, 4, 0x0}, {0x6e492c, 4, 0x0}, {0x6e4930, 4, 0x0}, {0x6e4934, 4, 0x0}, {0x6e4988, 4, 0x0}, {0x6e4978, 4, 0x0}, {0x6e497c, 4, 0x0}, {0x6e4980, 4, 0x0}, {0x6e49d4, 4, 0x0}, {0x6e49c4, 4, 0x0}, {0x6e49c8, 4, 0x0}, {0x6e49cc, 4, 0x0}, {0x6e4a20, 4, 0x0}, {0x6e4a10, 4, 0x0}, {0x6e4a14, 4, 0x0}, {0x6e4a18, 4, 0x0}, {0x6e4a6c, 4, 0x0}, {0x6e4a5c, 4, 0x0}, {0x6e4a60, 4, 0x0}, {0x6e4a64, 4, 0x0}}
var mrapIn = []cgCell{{0x6840bc, 1}, {0x683fc5, 1}, {0x683f94, 1}, {0x683f95, 1}, {0x683f96, 1}, {0x683f97, 1}, {0x683f98, 1}, {0x683f99, 1}, {0x683f9a, 1}, {0x6850d3, 1}, {0x6850d2, 1}, {0x685117, 1}, {0x650e50, 1}, {0x683fa5, 4}, {0x683fe7, 4}, {0x68518d, -4}, {0x68ec78, -4}, {0x6eb70c, 4}, {0x64c1c8, -4}, {0x68c568, 4}, {0x68c4ac, 4}, {0x68c4c0, -4}, {0x68c4c4, -4}, {0x65bde4, -4}, {0x659ab4, 4}, {0x200146b8, 4}, {0x20014704, -4}, {0x20014c88, 1}, {0x689b5c, -4}, {0x20014d44, 4}, {0x20014d64, 4}, {0x689f94, 4}, {0x20015004, -4}, {0x6e4138, 4}, {0x6e4128, 4}, {0x6e412c, 4}, {0x6e4130, 4}, {0x6e4184, 4}, {0x6e4174, 4}, {0x6e4178, 4}, {0x6e417c, 4}, {0x6e41d0, 4}, {0x6e41c0, 4}, {0x6e41c4, 4}, {0x6e41c8, 4}, {0x6e421c, 4}, {0x6e420c, 4}, {0x6e4210, 4}, {0x6e4214, 4}, {0x20000000, 1}, {0x20000001, 1}, {0x20000008, 4}, {0x200007a8, 4}, {0x200007ac, 4}, {0x200007b0, 4}, {0x200007b4, -4}, {0x200007bc, 4}, {0x200007c0, 1}, {0x20000910, 4}, {0x20000918, 4}, {0x20000a48, 1}, {0x20000a52, 1}, {0x20000a54, 1}, {0x20000a57, 1}, {0x20000a58, 1}, {0x20000a59, 1}, {0x200007b8, 4}, {0x200008c0, 4}, {0x200000bf, 1}, {0x20000193, 1}, {0x20000267, 1}, {0x2000033b, 1}, {0x2000040f, 1}, {0x200004e3, 1}, {0x200005b7, 1}, {0x2000068b, 1}, {0x200010f4, 4}, {0x20001154, 1}, {0x200012c0, 1}, {0x200012c8, 1}, {0x20001314, 1}, {0x200012ca, 2}, {0x200012f0, -4}, {0x200012f8, 1}, {0x20014ef8, 4}, {0x200141d8, 4}, {0x200141e0, 1}, {0x200141e2, 1}, {0x200144a4, 1}, {0x20014954, 1}, {0x200143d8, 4}, {0x200143e0, 1}, {0x200143e2, 1}, {0x200145a4, 1}, {0x20014b39, 1}, {0x20001400, 1}, {0x20001401, -4}, {0x2000143d, -4}, {0x20001441, -4}, {0x2000144d, -4}, {0x20001455, -4}, {0x20001459, -4}, {0x2000146d, 4}, {0x20001471, 4}, {0x200014fa, -4}, {0x20001506, 1}, {0x20001507, -4}, {0x20001543, -4}, {0x20001547, -4}, {0x20001553, -4}, {0x2000155b, -4}, {0x2000155f, -4}, {0x20001573, 4}, {0x20001577, 4}, {0x20001600, -4}, {0x2000160c, 1}, {0x2000160d, -4}, {0x20001649, -4}, {0x2000164d, -4}, {0x20001659, -4}, {0x20001661, -4}, {0x20001665, -4}, {0x20001679, 4}, {0x2000167d, 4}, {0x20001706, -4}, {0x20001712, 1}, {0x20001713, -4}, {0x2000174f, -4}, {0x20001753, -4}, {0x2000175f, -4}, {0x20001767, -4}, {0x2000176b, -4}, {0x2000177f, 4}, {0x20001783, 4}, {0x2000180c, -4}, {0x20001818, 1}, {0x20001819, -4}, {0x20001855, -4}, {0x20001859, -4}, {0x20001865, -4}, {0x2000186d, -4}, {0x20001871, -4}, {0x20001885, 4}, {0x20001889, 4}, {0x20001912, -4}, {0x2000191e, 1}, {0x2000191f, -4}, {0x2000195b, -4}, {0x2000195f, -4}, {0x2000196b, -4}, {0x20001973, -4}, {0x20001977, -4}, {0x2000198b, 4}, {0x2000198f, 4}, {0x20001a18, -4}, {0x20001a24, 1}, {0x20001a25, -4}, {0x20001a61, -4}, {0x20001a65, -4}, {0x20001a71, -4}, {0x20001a79, -4}, {0x20001a7d, -4}, {0x20001a91, 4}, {0x20001a95, 4}, {0x20001b1e, -4}, {0x20001b2a, 1}, {0x20001b2b, -4}, {0x20001b67, -4}, {0x20001b6b, -4}, {0x20001b77, -4}, {0x20001b7f, -4}, {0x20001b83, -4}, {0x20001b97, 4}, {0x20001b9b, 4}, {0x20001c24, -4}, {0x20002069, -4}, {0x20002071, -4}, {0x20002b01, 4}, {0x20002b11, 4}, {0x200038cb, -4}, {0x200038d3, -4}, {0x20004363, 4}, {0x20004373, 4}, {0x2000512d, -4}, {0x20005135, -4}, {0x20005bc5, 4}, {0x20005bd5, 4}, {0x2000698f, -4}, {0x20006997, -4}, {0x20007427, 4}, {0x20007437, 4}, {0x200081f1, -4}, {0x200081f9, -4}, {0x20008c89, 4}, {0x20008c99, 4}, {0x20009a53, -4}, {0x20009a5b, -4}, {0x2000a4eb, 4}, {0x2000a4fb, 4}, {0x2000b2b5, -4}, {0x2000b2bd, -4}, {0x2000bd4d, 4}, {0x2000bd5d, 4}, {0x2000cb17, -4}, {0x2000cb1f, -4}, {0x2000d5af, 4}, {0x2000d5bf, 4}, {0x683794, -4}, {0x2001b100, -4}, {0x2001b104, -4}, {0x2001b108, -4}, {0x2001b10c, -4}, {0x6838ac, -4}, {0x2001b110, -4}, {0x2001b114, -4}, {0x2001b118, -4}, {0x2001b11c, -4}, {0x6839c4, -4}, {0x2001b120, -4}, {0x2001b124, -4}, {0x2001b128, -4}, {0x2001b12c, -4}, {0x683adc, -4}, {0x2001b130, -4}, {0x2001b134, -4}, {0x2001b138, -4}, {0x2001b13c, -4}, {0x683bf4, -4}, {0x2001b140, -4}, {0x2001b144, -4}, {0x2001b148, -4}, {0x2001b14c, -4}, {0x683d0c, -4}, {0x2001b150, -4}, {0x2001b154, -4}, {0x2001b158, -4}, {0x2001b15c, -4}, {0x683e24, -4}, {0x2001b160, -4}, {0x2001b164, -4}, {0x2001b168, -4}, {0x2001b16c, -4}, {0x683f3c, -4}, {0x2001b170, -4}, {0x2001b174, -4}, {0x2001b178, -4}, {0x2001b17c, -4}, {0x2000e408, -4}, {0x20010000, -4}, {0x20010008, 4}, {0x20010014, 1}, {0x20010015, 1}, {0x20010016, 1}, {0x2001002b, 4}, {0x2001002f, 4}, {0x20010033, 4}, {0x2001005f, 4}, {0x2001009b, 4}, {0x20010107, 4}, {0x200102df, 1}, {0x200102e1, 4}, {0x200102e5, 4}, {0x20011000, 4}, {0x20011004, 1}, {0x20011005, 4}, {0x20011009, 4}, {0x20011011, 4}, {0x20011151, 1}, {0x200114ec, 4}, {0x200114f0, 4}, {0x200115dc, 4}, {0x20012064, 1}, {0x2001206d, 1}, {0x200120a4, 1}, {0x200120a7, 1}, {0x200125bc, 1}, {0x20010400, -4}, {0x20010408, 4}, {0x20010414, 1}, {0x20010415, 1}, {0x20010416, 1}, {0x2001042b, 4}, {0x2001042f, 4}, {0x20010433, 4}, {0x2001045f, 4}, {0x2001049b, 4}, {0x20010507, 4}, {0x200106df, 1}, {0x200106e1, 4}, {0x200106e5, 4}, {0x20011200, 4}, {0x20011204, 1}, {0x20011205, 4}, {0x20011209, 4}, {0x20011211, 4}, {0x20011351, 1}, {0x200116ec, 4}, {0x200116f0, 4}, {0x200117dc, 4}, {0x20012864, 1}, {0x2001286d, 1}, {0x200128a4, 1}, {0x200128a7, 1}, {0x20012dbc, 1}, {0x20013187, 2}, {0x200130e5, 1}, {0x2001325f, 4}, {0x20013587, 2}, {0x200134e5, 1}, {0x2001365f, 4}, {0x20013987, 2}, {0x200138e5, 1}, {0x20013a5f, 4}, {0x20013d87, 2}, {0x20013ce5, 1}, {0x20013e5f, 4}, {0x200151c7, 1}, {0x2001512e, 1}, {0x2001535f, 1}, {0x200152c7, 1}, {0x2001522e, 1}, {0x2001545f, 1}, {0x20015508, -4}, {0x20015708, -4}, {0x20015600, -4}, {0x20015618, 4}, {0x20015628, 1}, {0x20015629, 1}, {0x2001562a, 1}, {0x20015888, -4}, {0x20015890, -4}, {0x20015894, -4}, {0x20015680, -4}, {0x20015698, 4}, {0x200156a8, 1}, {0x200156a9, 1}, {0x200156aa, 1}, {0x200158c8, -4}, {0x200158d0, -4}, {0x200158d4, -4}, {0x20015780, -4}, {0x20015798, 4}, {0x200157a8, 1}, {0x200157a9, 1}, {0x200157aa, 1}, {0x20015908, -4}, {0x20015910, -4}, {0x20015914, -4}, {0x20015800, -4}, {0x20015818, 4}, {0x20015828, 1}, {0x20015829, 1}, {0x2001582a, 1}, {0x20015948, -4}, {0x20015950, -4}, {0x20015954, -4}, {0x20018000, 4}, {0x20018004, 4}, {0x20018008, 4}, {0x2001800c, 4}, {0x20018010, 4}, {0x20018014, 4}, {0x20018018, 4}, {0x2001801c, 4}, {0x20018020, 4}, {0x20018024, 4}, {0x20018028, 4}, {0x2001802c, 4}, {0x20018030, 4}, {0x20018034, 4}, {0x20018038, 4}, {0x2001803c, 4}, {0x20018040, 4}, {0x20018044, 4}, {0x20018048, 4}, {0x2001804c, 4}, {0x20018050, 4}, {0x20018054, 4}, {0x20018058, 4}, {0x2001805c, 4}, {0x20018060, 4}, {0x20018064, 4}, {0x20018068, 4}, {0x2001806c, 4}, {0x20018070, 4}, {0x20018074, 4}, {0x20018078, 4}, {0x2001807c, 4}, {0x20018080, 4}, {0x20018084, 4}, {0x20018088, -4}, {0x2001808c, 4}, {0x20018090, -4}, {0x20018094, 4}, {0x20018098, 4}, {0x2001809c, 4}, {0x200180a0, 4}, {0x200180a4, 4}, {0x200180a8, 4}, {0x200180ac, 4}, {0x200180b0, 4}, {0x200180b4, 4}, {0x200180b8, 4}, {0x200180bc, 4}, {0x200180c0, 4}, {0x200180c4, 4}, {0x200180c8, 4}, {0x200180cc, 4}, {0x200180d0, 4}, {0x200180d4, 4}, {0x200180d8, 4}, {0x200180dc, 4}, {0x200180e0, 4}, {0x200180e4, 4}, {0x200180e8, 4}, {0x200180ec, -4}, {0x200180f0, -4}, {0x200180f4, 4}, {0x200180f8, 4}, {0x200180fc, 4}}
var mrapInNames = []string{"g6840bc", "g683fc5", "g683f94", "g683f95", "g683f96", "g683f97", "g683f98", "g683f99", "g683f9a", "g6850d3", "g6850d2", "g685117", "g650e50", "g683fa5", "g683fe7", "sel", "mode", "tick", "g64c1c8", "g68c568", "g68c4ac", "qcnt", "qcap", "cursor", "g659ab4", "fx.b8", "Lfx.4", "w65a.88", "n500", "e500a.4", "e500b.4", "g689f94", "ilist.4", "mss0.h", "mss0.s0", "mss0.s1", "mss0.s2", "mss1.h", "mss1.s0", "mss1.s1", "mss1.s2", "mss2.h", "mss2.s0", "mss2.s1", "mss2.s2", "mss3.h", "mss3.s0", "mss3.s1", "mss3.s2", "G.0", "G.1", "G.8", "G.7a8", "G.7ac", "G.7b0", "G.7b4", "G.7bc", "G.7c0", "G.910", "G.918", "G.a48", "G.a52", "G.a54", "G.a57", "G.a58", "G.a59", "G.7b8", "G.8c0", "G.bf0", "G.bf1", "G.bf2", "G.bf3", "G.bf4", "G.bf5", "G.bf6", "G.bf7", "ui.f4", "ui.154", "ui.2c0", "ui.2c8", "ui.314", "ui.2ca", "ui.2f0", "ui.2f8", "g69c.f8", "E0.1d8", "E0.1e0", "E0.1e2", "Ein0.a4", "bde0", "E1.1d8", "E1.1e0", "E1.1e2", "Ein1.a4", "bde1", "S0", "S0.1", "S0.3d", "S0.41", "S0.r0", "S0.r8", "S0.rc", "S0.6d", "S0.71", "S0.fa", "S1", "S1.1", "S1.3d", "S1.41", "S1.r0", "S1.r8", "S1.rc", "S1.6d", "S1.71", "S1.fa", "S2", "S2.1", "S2.3d", "S2.41", "S2.r0", "S2.r8", "S2.rc", "S2.6d", "S2.71", "S2.fa", "S3", "S3.1", "S3.3d", "S3.41", "S3.r0", "S3.r8", "S3.rc", "S3.6d", "S3.71", "S3.fa", "S4", "S4.1", "S4.3d", "S4.41", "S4.r0", "S4.r8", "S4.rc", "S4.6d", "S4.71", "S4.fa", "S5", "S5.1", "S5.3d", "S5.41", "S5.r0", "S5.r8", "S5.rc", "S5.6d", "S5.71", "S5.fa", "S6", "S6.1", "S6.3d", "S6.41", "S6.r0", "S6.r8", "S6.rc", "S6.6d", "S6.71", "S6.fa", "S7", "S7.1", "S7.3d", "S7.41", "S7.r0", "S7.r8", "S7.rc", "S7.6d", "S7.71", "S7.fa", "C0.69", "C0.71", "C0.b01", "C0.b11", "C1.69", "C1.71", "C1.b01", "C1.b11", "C2.69", "C2.71", "C2.b01", "C2.b11", "C3.69", "C3.71", "C3.b01", "C3.b11", "C4.69", "C4.71", "C4.b01", "C4.b11", "C5.69", "C5.71", "C5.b01", "C5.b11", "C6.69", "C6.71", "C6.b01", "C6.b11", "C7.69", "C7.71", "C7.b01", "C7.b11", "pc0", "pc0.0", "pc0.1", "pc0.2", "pc0.3", "pc1", "pc1.0", "pc1.1", "pc1.2", "pc1.3", "pc2", "pc2.0", "pc2.1", "pc2.2", "pc2.3", "pc3", "pc3.0", "pc3.1", "pc3.2", "pc3.3", "pc4", "pc4.0", "pc4.1", "pc4.2", "pc4.3", "pc5", "pc5.0", "pc5.1", "pc5.2", "pc5.3", "pc6", "pc6.0", "pc6.1", "pc6.2", "pc6.3", "pc7", "pc7.0", "pc7.1", "pc7.2", "pc7.3", "count", "M0.id", "M0.8", "M0.14", "M0.15", "M0.16", "M0.2b", "M0.2f", "M0.33", "M0.5f", "M0.9b", "M0.107", "M0.2df", "M0.2e1", "M0.2e5", "T0.0", "T0.4", "T0.5", "T0.9", "T0.11", "T0.151", "o0.ec", "o0.f0", "o0.1dc", "A0.64", "A0.6d", "A0.a4", "A0.a7", "A0.mask", "M1.id", "M1.8", "M1.14", "M1.15", "M1.16", "M1.2b", "M1.2f", "M1.33", "M1.5f", "M1.9b", "M1.107", "M1.2df", "M1.2e1", "M1.2e5", "T1.0", "T1.4", "T1.5", "T1.9", "T1.11", "T1.151", "o1.ec", "o1.f0", "o1.1dc", "A1.64", "A1.6d", "A1.a4", "A1.a7", "A1.mask", "tmpl0.187", "tmpl0.e5", "tmpl0.25f", "tmpl1.187", "tmpl1.e5", "tmpl1.25f", "tmpl2.187", "tmpl2.e5", "tmpl2.25f", "tmpl3.187", "tmpl3.e5", "tmpl3.25f", "item0.c7", "item0.2e", "def0.5f", "item1.c7", "item1.2e", "def1.5f", "gl1.8", "gl2.8", "grpA.0", "grpA.18", "grpA.28", "grpA.29", "grpA.2a", "mlA.8", "maA.0", "maA.1", "grpB.0", "grpB.18", "grpB.28", "grpB.29", "grpB.2a", "mlB.8", "maB.0", "maB.1", "grpC.0", "grpC.18", "grpC.28", "grpC.29", "grpC.2a", "mlC.8", "maC.0", "maC.1", "grpD.0", "grpD.18", "grpD.28", "grpD.29", "grpD.2a", "mlD.8", "maD.0", "maD.1", "r_4e79a0_0", "r_4e79a0_1", "r_4e79a0_2", "r_4e79a0_3", "r_4e79a0_4", "r_4e79a0_5", "r_4e79a0_6", "r_4e79a0_7", "r_4e79a0_8", "r_4e79a0_9", "r_4e79a0_10", "r_4e79a0_11", "r_4e79a0_12", "r_4e79a0_13", "r_4e79a0_14", "r_4e79a0_15", "r_524a10_0", "r_524a10_1", "r_524a10_2", "r_524a10_3", "r_524a10_4", "r_524a10_5", "r_524a10_6", "r_524a10_7", "r_547310_0", "r_547310_1", "r_547310_2", "r_547310_3", "r_547310_4", "r_547310_5", "r_547310_6", "r_547310_7", "r_4e9f90", "r_4ebfe0", "w_4ebfe0_7b4", "r_53e250", "w_53e250_7b4", "r_52c240", "r_5323f0", "w_5323f0_102e1", "w_5323f0_102e5", "w_5323f0_102ba", "w_5323f0_102be", "w_5323f0_102c2", "w_5323f0_102c6", "w_5323f0_102ca", "w_5323f0_102ce", "w_5323f0_102d2", "w_5323f0_102d6", "w_5323f0_11005", "w_5323f0_11009", "w_5323f0_11011", "w_523c00_102e1", "w_523c00_11000", "w_523c00_11005", "w_523c00_11009", "w_4f0480_11000", "r_50b2f0_0", "r_50b2f0_1", "r_4bf970_0", "r_4bf970_1", "r_AILs", "r_AILt", "r_AIL3"}
var mrapOut = []cgCell{{0x6840bc, 1}, {0x683fc5, 1}, {0x683f94, 1}, {0x683f95, 1}, {0x683f96, 1}, {0x683f97, 1}, {0x683f98, 1}, {0x683f99, 1}, {0x683f9a, 1}, {0x6850d3, 1}, {0x6850d2, 1}, {0x685117, 1}, {0x650e50, 1}, {0x683fa5, 4}, {0x683fe7, 4}, {0x68518d, -4}, {0x68ec78, -4}, {0x6eb70c, 4}, {0x64c1c8, -4}, {0x68c568, 4}, {0x68c4ac, 4}, {0x68c4c0, -4}, {0x68c4c4, -4}, {0x65bde4, -4}, {0x659ab4, 4}, {0x200146b8, 4}, {0x20014704, -4}, {0x20014c88, 1}, {0x689b5c, -4}, {0x20014d44, 4}, {0x20014d64, 4}, {0x689f94, 4}, {0x20015004, -4}, {0x6e4138, 4}, {0x6e4128, 4}, {0x6e412c, 4}, {0x6e4130, 4}, {0x6e4184, 4}, {0x6e4174, 4}, {0x6e4178, 4}, {0x6e417c, 4}, {0x6e41d0, 4}, {0x6e41c0, 4}, {0x6e41c4, 4}, {0x6e41c8, 4}, {0x6e421c, 4}, {0x6e420c, 4}, {0x6e4210, 4}, {0x6e4214, 4}, {0x20000000, 1}, {0x20000001, 1}, {0x20000008, 4}, {0x200007a8, 4}, {0x200007ac, 4}, {0x200007b0, 4}, {0x200007b4, -4}, {0x200007bc, 4}, {0x200007c0, 1}, {0x20000910, 4}, {0x20000918, 4}, {0x20000a48, 1}, {0x20000a52, 1}, {0x20000a54, 1}, {0x20000a57, 1}, {0x20000a58, 1}, {0x20000a59, 1}, {0x200007b8, 4}, {0x200008c0, 4}, {0x200000bf, 1}, {0x20000193, 1}, {0x20000267, 1}, {0x2000033b, 1}, {0x2000040f, 1}, {0x200004e3, 1}, {0x200005b7, 1}, {0x2000068b, 1}, {0x200010f4, 4}, {0x20001154, 1}, {0x200012c0, 1}, {0x200012c8, 1}, {0x20001314, 1}, {0x200012ca, 2}, {0x200012f0, -4}, {0x200012f8, 1}, {0x20014ef8, 4}, {0x200141d8, 4}, {0x200141e0, 1}, {0x200141e2, 1}, {0x200144a4, 1}, {0x20014954, 1}, {0x200143d8, 4}, {0x200143e0, 1}, {0x200143e2, 1}, {0x200145a4, 1}, {0x20014b39, 1}, {0x20001400, 1}, {0x20001401, -4}, {0x2000143d, -4}, {0x20001441, -4}, {0x2000144d, -4}, {0x20001455, -4}, {0x20001459, -4}, {0x2000146d, 4}, {0x20001471, 4}, {0x200014fa, -4}, {0x20001506, 1}, {0x20001507, -4}, {0x20001543, -4}, {0x20001547, -4}, {0x20001553, -4}, {0x2000155b, -4}, {0x2000155f, -4}, {0x20001573, 4}, {0x20001577, 4}, {0x20001600, -4}, {0x2000160c, 1}, {0x2000160d, -4}, {0x20001649, -4}, {0x2000164d, -4}, {0x20001659, -4}, {0x20001661, -4}, {0x20001665, -4}, {0x20001679, 4}, {0x2000167d, 4}, {0x20001706, -4}, {0x20001712, 1}, {0x20001713, -4}, {0x2000174f, -4}, {0x20001753, -4}, {0x2000175f, -4}, {0x20001767, -4}, {0x2000176b, -4}, {0x2000177f, 4}, {0x20001783, 4}, {0x2000180c, -4}, {0x20001818, 1}, {0x20001819, -4}, {0x20001855, -4}, {0x20001859, -4}, {0x20001865, -4}, {0x2000186d, -4}, {0x20001871, -4}, {0x20001885, 4}, {0x20001889, 4}, {0x20001912, -4}, {0x2000191e, 1}, {0x2000191f, -4}, {0x2000195b, -4}, {0x2000195f, -4}, {0x2000196b, -4}, {0x20001973, -4}, {0x20001977, -4}, {0x2000198b, 4}, {0x2000198f, 4}, {0x20001a18, -4}, {0x20001a24, 1}, {0x20001a25, -4}, {0x20001a61, -4}, {0x20001a65, -4}, {0x20001a71, -4}, {0x20001a79, -4}, {0x20001a7d, -4}, {0x20001a91, 4}, {0x20001a95, 4}, {0x20001b1e, -4}, {0x20001b2a, 1}, {0x20001b2b, -4}, {0x20001b67, -4}, {0x20001b6b, -4}, {0x20001b77, -4}, {0x20001b7f, -4}, {0x20001b83, -4}, {0x20001b97, 4}, {0x20001b9b, 4}, {0x20001c24, -4}, {0x20002069, -4}, {0x20002071, -4}, {0x20002b01, 4}, {0x20002b11, 4}, {0x200038cb, -4}, {0x200038d3, -4}, {0x20004363, 4}, {0x20004373, 4}, {0x2000512d, -4}, {0x20005135, -4}, {0x20005bc5, 4}, {0x20005bd5, 4}, {0x2000698f, -4}, {0x20006997, -4}, {0x20007427, 4}, {0x20007437, 4}, {0x200081f1, -4}, {0x200081f9, -4}, {0x20008c89, 4}, {0x20008c99, 4}, {0x20009a53, -4}, {0x20009a5b, -4}, {0x2000a4eb, 4}, {0x2000a4fb, 4}, {0x2000b2b5, -4}, {0x2000b2bd, -4}, {0x2000bd4d, 4}, {0x2000bd5d, 4}, {0x2000cb17, -4}, {0x2000cb1f, -4}, {0x2000d5af, 4}, {0x2000d5bf, 4}, {0x683794, -4}, {0x2001b100, -4}, {0x2001b104, -4}, {0x2001b108, -4}, {0x2001b10c, -4}, {0x6838ac, -4}, {0x2001b110, -4}, {0x2001b114, -4}, {0x2001b118, -4}, {0x2001b11c, -4}, {0x6839c4, -4}, {0x2001b120, -4}, {0x2001b124, -4}, {0x2001b128, -4}, {0x2001b12c, -4}, {0x683adc, -4}, {0x2001b130, -4}, {0x2001b134, -4}, {0x2001b138, -4}, {0x2001b13c, -4}, {0x683bf4, -4}, {0x2001b140, -4}, {0x2001b144, -4}, {0x2001b148, -4}, {0x2001b14c, -4}, {0x683d0c, -4}, {0x2001b150, -4}, {0x2001b154, -4}, {0x2001b158, -4}, {0x2001b15c, -4}, {0x683e24, -4}, {0x2001b160, -4}, {0x2001b164, -4}, {0x2001b168, -4}, {0x2001b16c, -4}, {0x683f3c, -4}, {0x2001b170, -4}, {0x2001b174, -4}, {0x2001b178, -4}, {0x2001b17c, -4}, {0x2000e408, -4}, {0x20010000, -4}, {0x20010008, 4}, {0x20010014, 1}, {0x20010015, 1}, {0x20010016, 1}, {0x2001002b, 4}, {0x2001002f, 4}, {0x20010033, 4}, {0x2001005f, 4}, {0x2001009b, 4}, {0x20010107, 4}, {0x200102df, 1}, {0x200102e1, 4}, {0x200102e5, 4}, {0x20011000, 4}, {0x20011004, 1}, {0x20011005, 4}, {0x20011009, 4}, {0x20011011, 4}, {0x20011151, 1}, {0x200114ec, 4}, {0x200114f0, 4}, {0x200115dc, 4}, {0x20012064, 1}, {0x2001206d, 1}, {0x200120a4, 1}, {0x200120a7, 1}, {0x200125bc, 1}, {0x20010400, -4}, {0x20010408, 4}, {0x20010414, 1}, {0x20010415, 1}, {0x20010416, 1}, {0x2001042b, 4}, {0x2001042f, 4}, {0x20010433, 4}, {0x2001045f, 4}, {0x2001049b, 4}, {0x20010507, 4}, {0x200106df, 1}, {0x200106e1, 4}, {0x200106e5, 4}, {0x20011200, 4}, {0x20011204, 1}, {0x20011205, 4}, {0x20011209, 4}, {0x20011211, 4}, {0x20011351, 1}, {0x200116ec, 4}, {0x200116f0, 4}, {0x200117dc, 4}, {0x20012864, 1}, {0x2001286d, 1}, {0x200128a4, 1}, {0x200128a7, 1}, {0x20012dbc, 1}, {0x20013187, 2}, {0x200130e5, 1}, {0x2001325f, 4}, {0x20013587, 2}, {0x200134e5, 1}, {0x2001365f, 4}, {0x20013987, 2}, {0x200138e5, 1}, {0x20013a5f, 4}, {0x20013d87, 2}, {0x20013ce5, 1}, {0x20013e5f, 4}, {0x200151c7, 1}, {0x2001512e, 1}, {0x2001535f, 1}, {0x200152c7, 1}, {0x2001522e, 1}, {0x2001545f, 1}, {0x20015508, -4}, {0x20015708, -4}, {0x20015600, -4}, {0x20015618, 4}, {0x20015628, 1}, {0x20015629, 1}, {0x2001562a, 1}, {0x20015888, -4}, {0x20015890, -4}, {0x20015894, -4}, {0x20015680, -4}, {0x20015698, 4}, {0x200156a8, 1}, {0x200156a9, 1}, {0x200156aa, 1}, {0x200158c8, -4}, {0x200158d0, -4}, {0x200158d4, -4}, {0x20015780, -4}, {0x20015798, 4}, {0x200157a8, 1}, {0x200157a9, 1}, {0x200157aa, 1}, {0x20015908, -4}, {0x20015910, -4}, {0x20015914, -4}, {0x20015800, -4}, {0x20015818, 4}, {0x20015828, 1}, {0x20015829, 1}, {0x2001582a, 1}, {0x20015948, -4}, {0x20015950, -4}, {0x20015954, -4}, {0x20010242, 4}, {0x20010642, 4}, {0x200102ba, 4}, {0x200102be, 4}, {0x200102c2, 4}, {0x200102c6, 4}, {0x200102ca, 4}, {0x200102ce, 4}, {0x200102d2, 4}, {0x200102d6, 4}, {0x200106ba, 4}, {0x200106be, 4}, {0x200106c2, 4}, {0x200106c6, 4}, {0x200106ca, 4}, {0x200106ce, 4}, {0x200106d2, 4}, {0x200106d6, 4}, {0x20001451, 4}, {0x2000145d, 4}, {0x20001461, 4}, {0x20001465, 4}, {0x20001469, 4}, {0x20001557, 4}, {0x20001563, 4}, {0x20001567, 4}, {0x2000156b, 4}, {0x2000156f, 4}, {0x2000165d, 4}, {0x20001669, 4}, {0x2000166d, 4}, {0x20001671, 4}, {0x20001675, 4}, {0x20001763, 4}, {0x2000176f, 4}, {0x20001773, 4}, {0x20001777, 4}, {0x2000177b, 4}, {0x20001869, 4}, {0x20001875, 4}, {0x20001879, 4}, {0x2000187d, 4}, {0x20001881, 4}, {0x2000196f, 4}, {0x2000197b, 4}, {0x2000197f, 4}, {0x20001983, 4}, {0x20001987, 4}, {0x20001a75, 4}, {0x20001a81, 4}, {0x20001a85, 4}, {0x20001a89, 4}, {0x20001a8d, 4}, {0x20001b7b, 4}, {0x20001b87, 4}, {0x20001b8b, 4}, {0x20001b8f, 4}, {0x20001b93, 4}, {0x20001214, 4}, {0x20001218, 1}, {0x6875a5, 1}, {0x68c4c8, 4}, {0x20014f00, 4}, {0x20014f04, 4}, {0x20014f08, 4}, {0x20014f0c, 4}, {0x20014f10, 4}, {0x20014f14, 4}, {0x20014f18, 4}, {0x20014f1c, 4}, {0x689fd0, 2}, {0x689fd2, 2}, {0x689fd4, 2}, {0x689fd6, 2}, {0x689fd8, 2}, {0x689fda, 2}, {0x689fdc, 2}, {0x689fde, 2}, {0x689fe0, 2}, {0x689fe2, 2}, {0x689fe4, 2}, {0x689fe6, 2}, {0x689fe8, 2}, {0x689fea, 2}, {0x689fec, 2}, {0x689fee, 2}, {0x50000000, 4}, {0x50000004, 4}, {0x50000008, 4}, {0x5000000c, 4}, {0x50000010, 4}, {0x50000014, 4}, {0x50000018, 4}, {0x5000001c, 4}, {0x50000020, 4}, {0x50000024, 4}, {0x50000028, 4}, {0x5000002c, 4}, {0x50000030, 4}, {0x50000034, 4}, {0x50000038, 4}, {0x5000003c, 4}, {0x50000040, 4}, {0x50000044, 4}, {0x50000048, 4}, {0x5000004c, 4}, {0x50000050, 4}, {0x50000054, 4}, {0x50000058, 4}, {0x5000005c, 4}, {0x50000060, 4}, {0x50000064, 4}, {0x50000068, 4}, {0x5000006c, 4}, {0x50000070, 4}, {0x50000074, 4}, {0x50000078, 4}, {0x5000007c, 4}, {0x20016000, 4}, {0x20016004, 4}, {0x20016008, 4}, {0x2001600c, 4}, {0x20016010, 4}, {0x20016014, 4}, {0x20016018, 4}, {0x2001601c, 4}, {0x20016020, 4}, {0x20016024, 4}, {0x20016028, 4}, {0x2001602c, 4}, {0x20016030, 4}, {0x20016034, 4}, {0x20016038, 4}, {0x2001603c, 4}, {0x20016040, 4}, {0x20016044, 4}, {0x20016048, 4}, {0x2001604c, 4}, {0x20016050, 4}, {0x20016054, 4}, {0x20016058, 4}, {0x2001605c, 4}, {0x20016060, 4}, {0x20016064, 4}, {0x20016068, 4}, {0x2001606c, 4}, {0x20016070, 4}, {0x20016074, 4}, {0x20016078, 4}, {0x2001607c, 4}, {0x20016080, 4}, {0x20016084, 4}, {0x20016088, 4}, {0x2001608c, 4}, {0x20016090, 4}, {0x20016094, 4}, {0x20016098, 4}, {0x2001609c, 4}, {0x200160a0, 4}, {0x200160a4, 4}, {0x200160a8, 4}, {0x200160ac, 4}, {0x200160b0, 4}, {0x200160b4, 4}, {0x200160b8, 4}, {0x200160bc, 4}, {0x200160c0, 4}, {0x200160c4, 4}, {0x200160c8, 4}, {0x200160cc, 4}, {0x200160d0, 4}, {0x200160d4, 4}, {0x200160d8, 4}, {0x200160dc, 4}, {0x200160e0, 4}, {0x200160e4, 4}, {0x200160e8, 4}, {0x200160ec, 4}, {0x200160f0, 4}, {0x200160f4, 4}, {0x200160f8, 4}, {0x200160fc, 4}, {0x20016100, 4}, {0x20016104, 4}, {0x20016108, 4}, {0x2001610c, 4}, {0x20016110, 4}, {0x20016114, 4}, {0x20016118, 4}, {0x2001611c, 4}, {0x20016120, 4}, {0x20016124, 4}, {0x20016128, 4}, {0x2001612c, 4}, {0x20016130, 4}, {0x20016134, 4}, {0x20016138, 4}, {0x2001613c, 4}, {0x20016140, 4}, {0x20016144, 4}, {0x20016148, 4}, {0x2001614c, 4}, {0x20016150, 4}, {0x20016154, 4}, {0x20016158, 4}, {0x2001615c, 4}, {0x20016160, 4}, {0x20016164, 4}, {0x20016168, 4}, {0x2001616c, 4}, {0x20016170, 4}, {0x20016174, 4}, {0x20016178, 4}, {0x2001617c, 4}, {0x20016180, 4}, {0x20016184, 4}, {0x20016188, 4}, {0x2001618c, 4}, {0x20016190, 4}, {0x20016194, 4}, {0x20016198, 4}, {0x2001619c, 4}, {0x200161a0, 4}, {0x200161a4, 4}, {0x200161a8, 4}, {0x200161ac, 4}, {0x200161b0, 4}, {0x200161b4, 4}, {0x200161b8, 4}, {0x200161bc, 4}, {0x200161c0, 4}, {0x200161c4, 4}, {0x200161c8, 4}, {0x200161cc, 4}, {0x200161d0, 4}, {0x200161d4, 4}, {0x200161d8, 4}, {0x200161dc, 4}, {0x200161e0, 4}, {0x200161e4, 4}, {0x200161e8, 4}, {0x200161ec, 1}, {0x200161f0, 4}, {0x200161f4, 4}, {0x200161f8, 4}, {0x200161fc, 4}, {0x20016200, 4}, {0x20016204, 4}, {0x20016208, 4}, {0x2001620c, 4}, {0x20016210, 4}, {0x20016214, 4}, {0x20016218, 4}, {0x2001621c, 4}, {0x20016220, 4}, {0x20016224, 4}, {0x20016228, 4}, {0x2001622c, 4}, {0x20016230, 4}, {0x20016234, 4}, {0x20016238, 4}, {0x2001623c, 4}, {0x20016240, 4}, {0x20016244, 4}, {0x20016248, 4}, {0x2001624c, 4}, {0x20016250, 4}, {0x20016254, 4}, {0x20016258, 4}, {0x2001625c, 4}, {0x20016260, 4}, {0x20016264, 4}, {0x20016268, 4}, {0x2001626c, 4}, {0x20016270, 4}, {0x20016274, 4}, {0x20016278, 4}, {0x2001627c, 4}, {0x20016280, 4}, {0x20016284, 4}, {0x20016288, 4}, {0x2001628c, 4}, {0x20016290, 4}, {0x20016294, 4}, {0x20016298, 4}, {0x2001629c, 4}, {0x200162a0, 4}, {0x200162a4, 4}, {0x200162a8, 4}, {0x200162ac, 4}, {0x200162b0, 4}, {0x200162b4, 4}, {0x200162b8, 4}, {0x200162bc, 4}, {0x200162c0, 4}, {0x200162c4, 4}, {0x200162c8, 4}, {0x200162cc, 4}, {0x200162d0, 4}, {0x200162d4, 4}, {0x200162d8, 4}, {0x200162dc, 4}, {0x200162e0, 4}, {0x200162e4, 4}, {0x200162e8, 4}, {0x200162ec, 4}, {0x200162f0, 4}, {0x200162f4, 4}, {0x200162f8, 4}, {0x200162fc, 4}, {0x20016300, 4}, {0x20016304, 4}, {0x20016308, 4}, {0x2001630c, 4}, {0x20016310, 4}, {0x20016314, 4}, {0x20016318, 4}, {0x2001631c, 4}, {0x20016320, 4}, {0x20016324, 4}, {0x20016328, 4}, {0x2001632c, 4}, {0x20016330, 4}, {0x20016334, 4}, {0x20016338, 4}, {0x2001633c, 4}, {0x20016340, 4}, {0x20016344, 4}, {0x20016348, 4}, {0x2001634c, 4}, {0x20016350, 4}, {0x20016354, 4}, {0x20016358, 4}, {0x2001635c, 4}, {0x20016360, 4}, {0x20016364, 4}}
var mrapOutNames = []string{"g6840bc", "g683fc5", "g683f94", "g683f95", "g683f96", "g683f97", "g683f98", "g683f99", "g683f9a", "g6850d3", "g6850d2", "g685117", "g650e50", "g683fa5", "g683fe7", "sel", "mode", "tick", "g64c1c8", "g68c568", "g68c4ac", "qcnt", "qcap", "cursor", "g659ab4", "fx.b8", "Lfx.4", "w65a.88", "n500", "e500a.4", "e500b.4", "g689f94", "ilist.4", "mss0.h", "mss0.s0", "mss0.s1", "mss0.s2", "mss1.h", "mss1.s0", "mss1.s1", "mss1.s2", "mss2.h", "mss2.s0", "mss2.s1", "mss2.s2", "mss3.h", "mss3.s0", "mss3.s1", "mss3.s2", "G.0", "G.1", "G.8", "G.7a8", "G.7ac", "G.7b0", "G.7b4", "G.7bc", "G.7c0", "G.910", "G.918", "G.a48", "G.a52", "G.a54", "G.a57", "G.a58", "G.a59", "G.7b8", "G.8c0", "G.bf0", "G.bf1", "G.bf2", "G.bf3", "G.bf4", "G.bf5", "G.bf6", "G.bf7", "ui.f4", "ui.154", "ui.2c0", "ui.2c8", "ui.314", "ui.2ca", "ui.2f0", "ui.2f8", "g69c.f8", "E0.1d8", "E0.1e0", "E0.1e2", "Ein0.a4", "bde0", "E1.1d8", "E1.1e0", "E1.1e2", "Ein1.a4", "bde1", "S0", "S0.1", "S0.3d", "S0.41", "S0.r0", "S0.r8", "S0.rc", "S0.6d", "S0.71", "S0.fa", "S1", "S1.1", "S1.3d", "S1.41", "S1.r0", "S1.r8", "S1.rc", "S1.6d", "S1.71", "S1.fa", "S2", "S2.1", "S2.3d", "S2.41", "S2.r0", "S2.r8", "S2.rc", "S2.6d", "S2.71", "S2.fa", "S3", "S3.1", "S3.3d", "S3.41", "S3.r0", "S3.r8", "S3.rc", "S3.6d", "S3.71", "S3.fa", "S4", "S4.1", "S4.3d", "S4.41", "S4.r0", "S4.r8", "S4.rc", "S4.6d", "S4.71", "S4.fa", "S5", "S5.1", "S5.3d", "S5.41", "S5.r0", "S5.r8", "S5.rc", "S5.6d", "S5.71", "S5.fa", "S6", "S6.1", "S6.3d", "S6.41", "S6.r0", "S6.r8", "S6.rc", "S6.6d", "S6.71", "S6.fa", "S7", "S7.1", "S7.3d", "S7.41", "S7.r0", "S7.r8", "S7.rc", "S7.6d", "S7.71", "S7.fa", "C0.69", "C0.71", "C0.b01", "C0.b11", "C1.69", "C1.71", "C1.b01", "C1.b11", "C2.69", "C2.71", "C2.b01", "C2.b11", "C3.69", "C3.71", "C3.b01", "C3.b11", "C4.69", "C4.71", "C4.b01", "C4.b11", "C5.69", "C5.71", "C5.b01", "C5.b11", "C6.69", "C6.71", "C6.b01", "C6.b11", "C7.69", "C7.71", "C7.b01", "C7.b11", "pc0", "pc0.0", "pc0.1", "pc0.2", "pc0.3", "pc1", "pc1.0", "pc1.1", "pc1.2", "pc1.3", "pc2", "pc2.0", "pc2.1", "pc2.2", "pc2.3", "pc3", "pc3.0", "pc3.1", "pc3.2", "pc3.3", "pc4", "pc4.0", "pc4.1", "pc4.2", "pc4.3", "pc5", "pc5.0", "pc5.1", "pc5.2", "pc5.3", "pc6", "pc6.0", "pc6.1", "pc6.2", "pc6.3", "pc7", "pc7.0", "pc7.1", "pc7.2", "pc7.3", "count", "M0.id", "M0.8", "M0.14", "M0.15", "M0.16", "M0.2b", "M0.2f", "M0.33", "M0.5f", "M0.9b", "M0.107", "M0.2df", "M0.2e1", "M0.2e5", "T0.0", "T0.4", "T0.5", "T0.9", "T0.11", "T0.151", "o0.ec", "o0.f0", "o0.1dc", "A0.64", "A0.6d", "A0.a4", "A0.a7", "A0.mask", "M1.id", "M1.8", "M1.14", "M1.15", "M1.16", "M1.2b", "M1.2f", "M1.33", "M1.5f", "M1.9b", "M1.107", "M1.2df", "M1.2e1", "M1.2e5", "T1.0", "T1.4", "T1.5", "T1.9", "T1.11", "T1.151", "o1.ec", "o1.f0", "o1.1dc", "A1.64", "A1.6d", "A1.a4", "A1.a7", "A1.mask", "tmpl0.187", "tmpl0.e5", "tmpl0.25f", "tmpl1.187", "tmpl1.e5", "tmpl1.25f", "tmpl2.187", "tmpl2.e5", "tmpl2.25f", "tmpl3.187", "tmpl3.e5", "tmpl3.25f", "item0.c7", "item0.2e", "def0.5f", "item1.c7", "item1.2e", "def1.5f", "gl1.8", "gl2.8", "grpA.0", "grpA.18", "grpA.28", "grpA.29", "grpA.2a", "mlA.8", "maA.0", "maA.1", "grpB.0", "grpB.18", "grpB.28", "grpB.29", "grpB.2a", "mlB.8", "maB.0", "maB.1", "grpC.0", "grpC.18", "grpC.28", "grpC.29", "grpC.2a", "mlC.8", "maC.0", "maC.1", "grpD.0", "grpD.18", "grpD.28", "grpD.29", "grpD.2a", "mlD.8", "maD.0", "maD.1", "M0.242", "M1.242", "M0.2ba0", "M0.2ba1", "M0.2ba2", "M0.2ba3", "M0.2ba4", "M0.2ba5", "M0.2ba6", "M0.2ba7", "M1.2ba0", "M1.2ba1", "M1.2ba2", "M1.2ba3", "M1.2ba4", "M1.2ba5", "M1.2ba6", "M1.2ba7", "S0.r4", "S0.r10", "S0.r14", "S0.r18", "S0.r1c", "S1.r4", "S1.r10", "S1.r14", "S1.r18", "S1.r1c", "S2.r4", "S2.r10", "S2.r14", "S2.r18", "S2.r1c", "S3.r4", "S3.r10", "S3.r14", "S3.r18", "S3.r1c", "S4.r4", "S4.r10", "S4.r14", "S4.r18", "S4.r1c", "S5.r4", "S5.r10", "S5.r14", "S5.r18", "S5.r1c", "S6.r4", "S6.r10", "S6.r14", "S6.r18", "S6.r1c", "S7.r4", "S7.r10", "S7.r14", "S7.r18", "S7.r1c", "ui.214", "ui.218", "g6875a5", "qbuf", "qbuf.0", "qbuf.1", "qbuf.2", "qbuf.3", "qbuf.4", "qbuf.5", "qbuf.6", "qbuf.7", "text0", "text1", "text2", "text3", "text4", "text5", "text6", "text7", "text8", "text9", "text10", "text11", "text12", "text13", "text14", "text15", "heap0", "heap1", "heap2", "heap3", "heap4", "heap5", "heap6", "heap7", "heap8", "heap9", "heap10", "heap11", "heap12", "heap13", "heap14", "heap15", "heap16", "heap17", "heap18", "heap19", "heap20", "heap21", "heap22", "heap23", "heap24", "heap25", "heap26", "heap27", "heap28", "heap29", "heap30", "heap31", "n_4e79a0", "rec_4e79a0_1_0", "rec_4e79a0_1_1", "rec_4e79a0_1_2", "rec_4e79a0_1_3", "rec_4e79a0_1_4", "rec_4e79a0_1_5", "rec_4e79a0_1_6", "rec_4e79a0_1_7", "rec_4e79a0_1_8", "rec_4e79a0_1_9", "rec_4e79a0_1_10", "rec_4e79a0_1_11", "rec_4e79a0_1_12", "rec_4e79a0_1_13", "rec_4e79a0_1_14", "rec_4e79a0_1_15", "rec_4e79a0_2_0", "rec_4e79a0_2_1", "rec_4e79a0_2_2", "rec_4e79a0_2_3", "rec_4e79a0_2_4", "rec_4e79a0_2_5", "rec_4e79a0_2_6", "rec_4e79a0_2_7", "rec_4e79a0_2_8", "rec_4e79a0_2_9", "rec_4e79a0_2_10", "rec_4e79a0_2_11", "rec_4e79a0_2_12", "rec_4e79a0_2_13", "rec_4e79a0_2_14", "rec_4e79a0_2_15", "rec_4e79a0_3_0", "rec_4e79a0_3_1", "rec_4e79a0_3_2", "rec_4e79a0_3_3", "rec_4e79a0_3_4", "rec_4e79a0_3_5", "rec_4e79a0_3_6", "rec_4e79a0_3_7", "rec_4e79a0_3_8", "rec_4e79a0_3_9", "rec_4e79a0_3_10", "rec_4e79a0_3_11", "rec_4e79a0_3_12", "rec_4e79a0_3_13", "rec_4e79a0_3_14", "rec_4e79a0_3_15", "rec_4e79a0_4_0", "rec_4e79a0_4_1", "rec_4e79a0_4_2", "rec_4e79a0_4_3", "rec_4e79a0_4_4", "rec_4e79a0_4_5", "rec_4e79a0_4_6", "rec_4e79a0_4_7", "rec_4e79a0_4_8", "rec_4e79a0_4_9", "rec_4e79a0_4_10", "rec_4e79a0_4_11", "rec_4e79a0_4_12", "rec_4e79a0_4_13", "rec_4e79a0_4_14", "rec_4e79a0_4_15", "n_524a10", "rec_524a10_1_0", "rec_524a10_1_1", "rec_524a10_1_2", "rec_524a10_1_3", "rec_524a10_1_4", "rec_524a10_1_5", "rec_524a10_1_6", "rec_524a10_1_7", "n_547310", "rec_547310_1_0", "rec_547310_1_1", "rec_547310_1_2", "rec_547310_1_3", "rec_547310_1_4", "rec_547310_1_5", "rec_547310_1_6", "rec_547310_1_7", "rec_547310_2_0", "rec_547310_2_1", "rec_547310_2_2", "rec_547310_2_3", "rec_547310_2_4", "rec_547310_2_5", "rec_547310_2_6", "rec_547310_2_7", "rec_547310_3_0", "rec_547310_3_1", "rec_547310_3_2", "rec_547310_3_3", "rec_547310_3_4", "rec_547310_3_5", "rec_547310_3_6", "rec_547310_3_7", "rec_547310_4_0", "rec_547310_4_1", "rec_547310_4_2", "rec_547310_4_3", "rec_547310_4_4", "rec_547310_4_5", "rec_547310_4_6", "rec_547310_4_7", "n_4e9f90", "rec_4e9f90_1", "n_4ebfe0", "rec_4ebfe0_1", "n_53e250", "rec_53e250_1", "n_52c240", "rec_52c240_1", "rec_52c240_2", "n_5323f0", "rec_5323f0_1", "rec_5323f0_2", "rec_5323f0_3", "rec_5323f0_4", "n_523c00", "rec_523c00_1", "rec_523c00_6", "n_4f0480", "rec_4f0480_1", "rec_4f0480_2", "n_4e6ce0", "n_4e8370", "n_4e9490", "n_4e8000", "rec_4e8000_1", "rec_4e8000_2", "rec_4e8000_3", "rec_4e8000_4", "rec_4e8000_5", "n_4ea5c0", "rec_4ea5c0_1", "n_4eac90", "rec_4eac90_1", "n_4eae20", "rec_4eae20_1", "rec_4eae20_2", "n_4eb5a0", "rec_4eb5a0_1", "n_5049c0", "rec_5049c0_1_0", "rec_5049c0_1_1", "rec_5049c0_1_2", "rec_5049c0_1_3", "rec_5049c0_2_0", "rec_5049c0_2_1", "rec_5049c0_2_2", "rec_5049c0_2_3", "rec_5049c0_3_0", "rec_5049c0_3_1", "rec_5049c0_3_2", "rec_5049c0_3_3", "n_4f01d0", "n_4c6200", "rec_4c6200_1", "rec_4c6200_2", "n_4c5eb0", "rec_4c5eb0_1_0", "rec_4c5eb0_1_1", "rec_4c5eb0_1_2", "rec_4c5eb0_1_3", "rec_4c5eb0_1_4", "rec_4c5eb0_1_5", "rec_4c5eb0_2_0", "rec_4c5eb0_2_1", "rec_4c5eb0_2_2", "rec_4c5eb0_2_3", "rec_4c5eb0_2_4", "rec_4c5eb0_2_5", "rec_4c5eb0_3_0", "rec_4c5eb0_3_1", "rec_4c5eb0_3_2", "rec_4c5eb0_3_3", "rec_4c5eb0_3_4", "rec_4c5eb0_3_5", "n_5554a0", "rec_5554a0_1", "n_52e690", "rec_52e690_1", "rec_52e690_2", "rec_52e690_3", "rec_52e690_4", "rec_52e690_5", "n_56ee20", "rec_56ee20_1", "n_587a30", "n_58a860", "n_5b3100", "rec_5b3100_1", "n_596cf0", "n_592e60", "n_53a320", "rec_53a320_1", "n_50b2f0", "rec_50b2f0_1_0", "rec_50b2f0_1_1", "rec_50b2f0_2_0", "rec_50b2f0_2_1", "n_41f0d0", "n_55ee70", "rec_55ee70_1", "n_5299b0", "rec_5299b0_1", "n_4bf970", "rec_4bf970_1_0", "rec_4bf970_1_1", "n_AILs", "rec_AILs_1", "n_AILt", "rec_AILt_1", "n_AIL3", "rec_AIL3_1"}

const (
	mrapS4E79A0 = iota
	mrapS524A10
	mrapS547310
	mrapS4E9F90
	mrapS4EBFE0
	mrapS53E250
	mrapS52C240
	mrapS5323F0
	mrapS523C00
	mrapS4F0480
	mrapS4E6CE0
	mrapS4E8370
	mrapS4E9490
	mrapS4E8000
	mrapS4EA5C0
	mrapS4EAC90
	mrapS4EAE20
	mrapS4EB5A0
	mrapS5049C0
	mrapS4F01D0
	mrapS4C6200
	mrapS4C5EB0
	mrapS5554A0
	mrapS52E690
	mrapS56EE20
	mrapS587A30
	mrapS58A860
	mrapS5B3100
	mrapS596CF0
	mrapS592E60
	mrapS53A320
	mrapS50B2F0
	mrapS41F0D0
	mrapS55EE70
	mrapS5299B0
	mrapS4BF970
	mrapSAILS
	mrapSAILT
	mrapSAIL3
)

var mrapStubs = []mrapStub{
	{cnt: 0x20016000, ret: 0x20018000, retN: 16, rec: [][4]int64{{1, 0x20016004, 16, 4}, {2, 0x20016044, 16, 4}, {3, 0x20016084, 16, 4}, {4, 0x200160c4, 16, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016104, ret: 0x20018040, retN: 8, rec: [][4]int64{{1, 0x20016108, 8, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016128, ret: 0x20018060, retN: 8, rec: [][4]int64{{1, 0x2001612c, 8, 4}, {2, 0x2001614c, 8, 4}, {3, 0x2001616c, 8, 4}, {4, 0x2001618c, 8, 4}}, wr: [][3]int64{}},
	{cnt: 0x200161ac, ret: 0x20018080, retN: 0, rec: [][4]int64{{1, 0x200161b0, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x200161b4, ret: 0x20018084, retN: 0, rec: [][4]int64{{1, 0x200161b8, 1, 4}}, wr: [][3]int64{{0x200007b4, 4, 0x20018088}}},
	{cnt: 0x200161bc, ret: 0x2001808c, retN: 0, rec: [][4]int64{{1, 0x200161c0, 1, 4}}, wr: [][3]int64{{0x200007b4, 4, 0x20018090}}},
	{cnt: 0x200161c4, ret: 0x20018094, retN: 0, rec: [][4]int64{{1, 0x200161c8, 1, 4}, {2, 0x200161cc, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x200161d0, ret: 0x20018098, retN: 0, rec: [][4]int64{{1, 0x200161d4, 1, 4}, {2, 0x200161d8, 1, 4}, {3, 0x200161dc, 1, 4}, {4, 0x200161e0, 1, 4}}, wr: [][3]int64{{0x200102e1, 4, 0x2001809c}, {0x200102e5, 4, 0x200180a0}, {0x200102ba, 4, 0x200180a4}, {0x200102be, 4, 0x200180a8}, {0x200102c2, 4, 0x200180ac}, {0x200102c6, 4, 0x200180b0}, {0x200102ca, 4, 0x200180b4}, {0x200102ce, 4, 0x200180b8}, {0x200102d2, 4, 0x200180bc}, {0x200102d6, 4, 0x200180c0}, {0x20011005, 4, 0x200180c4}, {0x20011009, 4, 0x200180c8}, {0x20011011, 4, 0x200180cc}}},
	{cnt: 0x200161e4, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x200161e8, 1, 4}, {6, 0x200161ec, 1, 1}}, wr: [][3]int64{{0x200102e1, 4, 0x200180d0}, {0x20011000, 4, 0x200180d4}, {0x20011005, 4, 0x200180d8}, {0x20011009, 4, 0x200180dc}}},
	{cnt: 0x200161f0, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x200161f4, 1, 4}, {2, 0x200161f8, 1, 4}}, wr: [][3]int64{{0x20011000, 4, 0x200180e0}}},
	{cnt: 0x200161fc, ret: 0x0, retN: 0, rec: [][4]int64{}, wr: [][3]int64{}},
	{cnt: 0x20016200, ret: 0x0, retN: 0, rec: [][4]int64{}, wr: [][3]int64{}},
	{cnt: 0x20016204, ret: 0x0, retN: 0, rec: [][4]int64{}, wr: [][3]int64{}},
	{cnt: 0x20016208, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x2001620c, 1, 4}, {2, 0x20016210, 1, 4}, {3, 0x20016214, 1, 4}, {4, 0x20016218, 1, 4}, {5, 0x2001621c, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016220, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x20016224, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016228, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x2001622c, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016230, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x20016234, 1, 4}, {2, 0x20016238, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x2001623c, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x20016240, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016244, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x20016248, 4, 4}, {2, 0x20016258, 4, 4}, {3, 0x20016268, 4, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016278, ret: 0x0, retN: 0, rec: [][4]int64{}, wr: [][3]int64{}},
	{cnt: 0x2001627c, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x20016280, 1, 4}, {2, 0x20016284, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016288, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x2001628c, 6, 4}, {2, 0x200162a4, 6, 4}, {3, 0x200162bc, 6, 4}}, wr: [][3]int64{}},
	{cnt: 0x200162d4, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x200162d8, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x200162dc, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x200162e0, 1, 4}, {2, 0x200162e4, 1, 4}, {3, 0x200162e8, 1, 4}, {4, 0x200162ec, 1, 4}, {5, 0x200162f0, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x200162f4, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x200162f8, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x200162fc, ret: 0x0, retN: 0, rec: [][4]int64{}, wr: [][3]int64{}},
	{cnt: 0x20016300, ret: 0x0, retN: 0, rec: [][4]int64{}, wr: [][3]int64{}},
	{cnt: 0x20016304, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x20016308, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x2001630c, ret: 0x0, retN: 0, rec: [][4]int64{}, wr: [][3]int64{}},
	{cnt: 0x20016310, ret: 0x0, retN: 0, rec: [][4]int64{}, wr: [][3]int64{}},
	{cnt: 0x20016314, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x20016318, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x2001631c, ret: 0x200180e4, retN: 2, rec: [][4]int64{{1, 0x20016320, 2, 4}, {2, 0x20016328, 2, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016330, ret: 0x0, retN: 0, rec: [][4]int64{}, wr: [][3]int64{}},
	{cnt: 0x20016334, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x20016338, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x2001633c, ret: 0x0, retN: 0, rec: [][4]int64{{1, 0x20016340, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016344, ret: 0x200180ec, retN: 2, rec: [][4]int64{{1, 0x20016348, 2, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016350, ret: 0x200180f4, retN: 0, rec: [][4]int64{{1, 0x20016354, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016358, ret: 0x200180f8, retN: 0, rec: [][4]int64{{1, 0x2001635c, 1, 4}}, wr: [][3]int64{}},
	{cnt: 0x20016360, ret: 0x200180fc, retN: 0, rec: [][4]int64{{1, 0x20016364, 1, 4}}, wr: [][3]int64{}},
}

// Parked rows: boundary row 0 of each callee's case (the callee's own default world).
var mrapPark4e9f90 = [379]int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 536936448, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var mrapPark4ebfe0 = [106]int64{0, 1, 92, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 123, 124, 125, 123, 124, 125, 123, 124, 0, 0, 1, 0, 7, 0, 123, 456, 0, 0, 1065353216, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 1065353216, 0, 2, 0, 1, 1, 1, 10, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 1065353216, 0, 1073741824}
var mrapPark53e250 = [169]int64{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 5, 0, 0, 1, 6, -1, 2, 1, 1, 0, 0, 0, 0, 0, 3, 4, 5, 0, 1, 1, 0, -1, 0, 0, -1, 1, 2, 10, 0, 40, 30, 0, 0, 0, 0, 0, 0, 0, 1, 0, 6, 16, 0, 0, 0, 0, 0, 0, 6, 16, 0, 0, 0, 0, 0, 0, 0, 2, 1, 11, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 11, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 50, 20, 20, 10000, 0, 0, 0, 0, 0, 0, 0, 0, 15, 15, 15, 15, 15, 15, 15, 0, 0, 0, 0, 1, 1, 0, 0, 60, 60, 2, 3, 0, 5, 1, 3, 0, 3, 0, 6, 1, 0, 0, 1, 1, 1, 1, 3, 4, 5, 0, 50, 0, 10, 1, 0, 12, 8, 1, 4, 1, 0, 4, 1, -1, 0, 0, -1}
