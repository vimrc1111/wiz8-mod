package calc

import (
	"math"
	"strconv"
)

// Node 0x4e8370: the start-of-round preparation of combat (lightning strikes
// round). It resets the round flags of the combat block [0x6836a8], runs the
// pending phase refresh 0x4f0560 (per party slot 0x53a930: re-mark the monsters
// the stored action targets), sets the move allowance into the scene, counts
// the round, clears the per-slot and per-monster round fields, rolls each
// slot's feature-0x10 bonus and each monster's 75% flag, lets every living
// monster choose its action and posts the "is casting" style messages.
//
// It runs over the case's byte world (cgW cells). Address-stubbed D4 callees
// (0x4e79a0, 0x524a10, 0x53c270, 0x547310, 0x4f0500, 0x4effa0, 0x4ee9d0,
// 0x536400, 0x4e89d0, 0x531540, 0x4fd030, 0x4f0e80) and the registered-out
// ones (0x4c5eb0, 0x58b410, 0x58aad0, 0x58ac00, 0x5182e0, 0x595570, 0x595600,
// 0x55ee70, 0x4add30, 0x56aa30, 0x56aab0, 0x5a1890, 0x56a5a0) record every
// call. Emu replays the stub returns from case inputs; F runs the real Go
// nodes of the D4 callees whose results are read later, on parked default
// worlds of their own (lsPark*) with the live values the original hands them,
// so F and Emu differ wherever the case's stub inputs differ from what those
// worlds compute. Fixture caps: four monsters (list count 0..4), templates 0..3
// in the cache, at most four ids in a 0x4fd030 monster list.

type lsStub struct {
	cnt, ret int64
	retN     int64
	rec      [][3]int64 // {argument (1-based), first record address, records}
}

type lsW struct {
	cgW
	r Rand
	c *lsCallees
}

type lsCallees struct {
	elig  func(w *lsW, x int64) bool     // 0x4e79a0(x, 1, 0, 0)
	act   func(w *lsW, x int64) bool     // 0x524a10(x)
	tgt   func(w *lsW, x int64) bool     // 0x53c270(x)
	off   func(w *lsW, x int64) bool     // 0x547310(C, kind, id, S+0x45)
	step  func(w *lsW) int64             // 0x4f0500()
	move  func(w *lsW) float64           // 0x4effa0() st0
	area  func(w *lsW, id, x, rec int64) // 0x4fd030(id, 1, &l, rec, tbl, &l2, 0)
	init  func(w *lsW)                   // 0x4e89d0()
	pick  func(w *lsW, e int64)          // 0x531540(e)
	reval func(w *lsW)                   // 0x536400()
}

// call records one stubbed call (args are the stack arguments, 1-based) and
// returns its sequence number; more calls than the case records panic.
func (w *lsW) call(k int, args ...int64) int64 {
	s := &lsStubs[k]
	n := w.u32(s.cnt)
	for _, r := range s.rec {
		if n >= r[2] {
			panic("lightning strikes round: more stubbed calls than the case records")
		}
		w.p32(r[1]+4*n, args[r[0]-1])
	}
	w.p32(s.cnt, n+1)
	return n
}

func (w *lsW) ret(k int, n int64) int64 {
	s := &lsStubs[k]
	if s.retN == 0 {
		return w.u32(s.ret)
	}
	if n >= s.retN {
		panic("lightning strikes round: more stubbed calls than the case returns")
	}
	return w.u32(s.ret + 4*n)
}

// Party slot x (0..7): slot 0x685178+0x106x, character 0x685174+0x1862x,
// combat row 0x6836a8 block +0xd4x.
func (w *lsW) S(x int64) int64 { return w.u32(0x685178) + 0x106*lsSlot(x) }
func (w *lsW) C(x int64) int64 { return w.u32(0x685174) + 0x1862*lsSlot(x) }
func (w *lsW) D(x int64) int64 { return w.u32(0x6836a8) + 0xd4*lsSlot(x) }
func (w *lsW) A() int64        { return w.u32(0x6836a8) }
func (w *lsW) uip() int64      { return w.u32(0x68edcc) }
func (w *lsW) f94() bool       { return w.u8(0x683f94) != 0 }

func lsSlot(x int64) int64 {
	if x < 0 || x > 7 {
		panic("lightning strikes round: party slot outside 0..7")
	}
	return x
}

// ui is 0x562a50(mask): interface mode 7 ORs the mask into ui+0xf4.
func (w *lsW) ui(mask int64) {
	if u := w.uip(); w.i32(0x68ec78) == 7 && u != 0 {
		w.p32(u+0xf4, w.u32(u+0xf4)|mask)
	}
}

// resolve is 0x53bc10 (attack context 6) over eight slots.
func (w *lsW) resolve(x int64) int64 {
	if u := w.uip(); w.i32(0x68ec78) == 7 && u != 0 && w.i32(u+0x2f0) != -1 {
		a, b := w.i32(u+0x2f0), w.u8(u+0x2f8) != 0
		switch {
		case a == 7 && b:
			return 3
		case a == 8 && b:
			return 4
		}
		return 7
	}
	if x == w.i32(0x68518d) && (w.u8(0x683f95) != 0 || w.u8(0x683f96) != 0) {
		return 2
	}
	if w.f94() {
		return 1
	}
	return 0
}

// act is 0x4e77b0(x, 6): action kind a, id b, extra d of the resolved source;
// 0x53c270 (stub) decides whether the slot can act at all.
func (w *lsW) act(x int64) (a, b, d int64) {
	S := w.S(x)
	switch w.resolve(x) {
	case 0:
		a, b, d = w.i32(S+1), w.i32(S+5), S+0x15
	case 1:
		a, b, d = w.i32(S+0x3d), w.i32(S+0x41), S+0x45
	case 2:
		if w.u8(0x683f95) != 0 {
			b = w.i32(w.u32(0x69bf3c) + 0x104)
			if b == 0 {
				b = w.i32(w.u32(0x69bf3c) + 0x100)
			}
			a, d = 7, 0x6840ab
		} else {
			v := w.i32(0x69b9a4)
			if v == 0 {
				v = w.i32(0x69b9a0)
			}
			w.p32(0x6840af, v)
			a, b, d = 8, -1, 0x6840ab
		}
	case 3:
		a, b, d = 7, w.i32(S+0x75), S+0x79
	case 4:
		a, b, d = 8, -1, S+0xa1
	case 7:
		u := w.uip()
		a, b = w.i32(u+0x2f0), w.i32(u+0x2f4)
	}
	if !w.c.tgt(w, x) && a != 0xa && a != 0xb {
		a, b, d = -1, -1, 0
	}
	return
}

// setMask is 0x539630(x, id, flag): party member x's bit in monster id's mask.
func (w *lsW) setMask(x, id, flag int64) {
	e := w.find(id, false)
	if e == -1 {
		return
	}
	p := w.u32(w.entry(e) + 0xc)
	if p == 0 {
		panic("lightning strikes round: 0x539630 NULL record")
	}
	a := w.u32(p+0x1d8) + 0x5bc
	if flag != 0 {
		w.p8(a, w.u8(a)|1<<uint(x))
	} else {
		w.p8(a, w.u8(a)&^(1<<uint(x)))
	}
	w.call(lsS4C5EB0, x, id, flag)
}

// release is 0x53a930(x, rec): mark the monsters slot x's resolved action
// targets in their masks (0x683790+0x118x is the 0x4fd030 monster list).
func (w *lsW) release(x, rec int64) {
	tbl := 0x683790 + 0x118*x
	w.p32(tbl+4, 0)
	a, b, d := w.act(x)
	id := int64(0)
	switch a {
	case 2:
		id = 0x77 // 0x53a96f
	case 7:
		id = b
	case 8:
		id = w.itemSpell(w.u32(d + 4))
	}
	if id == 0 {
		r := w.rec(rec)
		if r[0] == 3 && r[2] != -1 && w.find(r[2], false) != -1 {
			w.setMask(x, r[2], 1)
		}
		// kind 4 with a group id: 0x5100b0 finds no group (group world excluded) and nothing is marked.
		return
	}
	w.c.area(w, id, x, rec)
	for i := int64(0); i < w.monCount(); i++ {
		e := w.entry(i)
		if p := w.u32(e + 0xc); w.u8(e+0x14) != 0 && p != 0 {
			m := w.u32(p+0x1d8) + 0x5bc
			if v := w.u8(m); v&(1<<uint(x)) != 0 {
				w.p8(m, v&^(1<<uint(x)))
				w.call(lsS4C5EB0, x, w.i32(e), 0)
			}
		}
	}
	n := w.u32(tbl + 4)
	if n > 4 {
		panic("lightning strikes round: 0x4fd030 monster list past the four-id fixture")
	}
	for i := int64(0); i < n; i++ {
		w.setMask(x, w.i32(w.u32(tbl+0xc)+4*i), 1)
	}
}

// menuTail is the common end of 0x4f0aa0 and 0x4f0560.
func (w *lsW) menuTail() {
	A := w.A()
	if (w.u32(A+0x910) == 0 || w.u32(A+0x918) == 3) && w.u32(A+0x90c) == 0 {
		w.call(lsS5A1890)
		return
	}
	u := w.uip()
	w.p32(u+0x2dc, 100)
	w.p32(u+0x2e0, 100)
	m := w.u32(0x69bf4c) // 0x5a1dd0 -> menu vtable 0x5ed5a4 slot 1 (0x4f2e50)
	w.p8(m+5, 1)
	w.p8(m+0x34, 1)
	w.ui(0x80000000)
}

// name is 0x4e5150(e, 0, 0): the monster's display name.
func (w *lsW) name(e int64) int64 {
	t := w.u32(e + 8)
	if t >= 1000 {
		panic("lightning strikes round: 0x4e5150 template id past 999 (assert)")
	}
	tm := w.u32(0x6840c7 + 4*t)
	if tm == 0 {
		panic("lightning strikes round: template cache miss is outside the fixture (file load)")
	}
	if w.rd(tm+0x187, 2) == 0x18d {
		// swprintf(0x6875c3, L"Al-%s" (0x614b44), name of character [0x6875ef])
		src := w.C(w.i32(0x6875ef)) + 5
		s := []int64{'A', 'l', '-'}
		for i := int64(0); ; i++ {
			c := w.rd(src+2*i, 2)
			s = append(s, c)
			if c == 0 {
				break
			}
		}
		for i, c := range s {
			w.wr(0x6875c3+2*int64(i), 2, c)
		}
		return 0x6875c3
	}
	if w.u32(e+4) != 0 {
		panic("lightning strikes round: grouped monster name (group world) is outside the fixture")
	}
	st := w.u32(w.u32(e+0xc) + 0x1d8)
	if !(int8(w.u8(st+0xa4)) == 0x15 || w.u8(st+0xa7) == 0x15) { // 0x4ca4c0
		w.call(lsS5182E0, 1, 0x614b20, w.i32(e))
	}
	return tm + 0x60
}

// say formats fmt with one string argument the way 0x590950/0x590b40 do and
// returns the first two characters of the result.
func (w *lsW) say(f, arg int64) (c0, c1 int64) {
	c0 = w.rd(f, 2)
	if c0 == 0 {
		return 0, 0
	}
	c1 = w.rd(f+2, 2)
	if c0 == '%' || c1 == '%' {
		panic("lightning strikes round: a conversion in the first two format characters is outside the fixture")
	}
	return
}

// msgPC is 0x590950(x, fmt, name).
func (w *lsW) msgPC(x, f, arg int64) {
	c0, c1 := w.say(f, arg)
	w.call(lsS58AAD0, 8, 0x64bf5c, w.C(x)+5)
	n := int64(0)
	for w.rd(w.C(x)+5+2*n, 2) != 0 {
		n++
	}
	if c0 == 0x27 {
		n++
		if c1 == 0x73 {
			n++
		}
	}
	w.call(lsS58B410, w.u8(w.S(x)+0xf1), 0, n, -1)
}

// msgMon is 0x590b40(e, fmt, name).
func (w *lsW) msgMon(e, f, arg int64) {
	w.say(f, arg)
	w.call(lsS58AAD0, 9, 0x64bf5c, w.name(e))
}

// any is 0x4ed460: is anyone still able to act this round.
func (w *lsW) any() bool {
	A := w.A()
	if w.f94() {
		for j := int64(0); j < 2; j++ {
			S, C := w.S(j), w.C(j)
			if w.u8(S) != 0 && w.u32(C+0xb11) > 0 && w.u32(C+0xb01) < 0xf && w.u8(A+0xa58+j) != 0 {
				return false
			}
		}
	}
	for i := int64(0); i < w.monCount(); i++ { // 0x4eee20
		e := w.entry(i)
		if w.u8(e+0x15) != 0 && w.u8(e+0x16) == 1 && w.u32(e+0x2b) > 0 && w.u32(e+0x107) < 0xe {
			return true
		}
	}
	for j := int64(0); j < 8; j++ {
		if w.u8(w.S(j)) != 0 && w.u32(w.C(j)+0xb11) > 0 && w.c.elig(w, j) && w.c.off(w, j) {
			return true
		}
	}
	return false
}

// scene is 0x41eee0(bits, b2, b3) on the scene object [0x652dac].
func (w *lsW) scene(bits, b2, b3 int64) {
	p := w.u32(0x652dac)
	if p == 0 {
		return
	}
	w.p32(p+0x2c, bits)
	for _, o := range []int64{0x24, 0x28, 0x30, 0x4c, 0x50, 0x54, 0x7c, 0x80, 0x84} {
		w.p32(p+o, 0)
	}
	v := w.u32(p) | 0x20
	v &^= 0x180
	if b2&0xff != 0 {
		v |= 0x80
	}
	if b3&0xff != 0 {
		v |= 0x100
	}
	w.p32(p, v)
}

func (w *lsW) run() {
	A := w.A()
	if w.u32(A+4) == 0 && w.u8(A+0xa54) != 0 || w.u32(A+0x918) == 3 {
		w.call(lsS4F0E80, 0, 0xc350, 1) // 0x4f1100: party position [0x659c0c]+0x100, radius 50000
	}
	if w.rd(w.uip()+0x2ca, 2) != 0xffff {
		if w.u8(0x6850d2) == 0 {
			w.call(lsS595570)
		} else {
			w.call(lsS595600)
		}
	}
	if w.u8(0x6850d2) == 0 && w.u32(0x683fe7) != 0 { // 0x53a320(0)
		w.p32(0x683fe7, 0)
		if w.i32(0x683fdb) != -1 {
			w.call(lsS55EE70, -1)
		}
		w.p32(0x68407f, 0)
		w.p32(0x684083, 0)
		w.p32(0x684087, 0)
		if u := w.uip(); u != 0 {
			if w.f94() {
				w.p8(u+0x2c0, 1)
			}
			w.p8(u+0x2c8, 1)
		}
		w.call(lsS4ADD30, 0)
		if w.i32(0x68ec78) == 7 {
			w.call(lsS56AAB0)
		}
	}
	w.p8(A, 1)
	w.p32(A+0x908, 0)
	w.p8(A+0xa55, 1)
	w.p8(w.uip()+0x154, 0)
	w.p32(A+0x910, w.u32(A+0x90c))
	w.p32(A+0x918, 0)
	w.menuTail()             // 0x4f0aa0
	if w.u32(A+0x90c) != 0 { // 0x4f0560(-1)
		if !w.f94() {
			panic("lightning strikes round: 0x4f0560 asserts outside combat")
		}
		w.p32(A+0x90c, 0)
		for x := int64(0); x < 8; x++ {
			if w.c.elig(w, x) {
				w.release(x, w.S(x)+0x4d)
			}
		}
		w.menuTail()
	}
	if w.u8(0x6850d2) == 0 {
		w.p8(w.uip()+0x2c8, 1)
		w.p8(A+1, 0)
		w.call(lsS56A5A0)
	}
	w.p32(A+4, w.u32(A+4)+1)
	w.p32(0x659c14, w.u32(0x659c14)+1)
	if m := w.u32(A + 0x910); m == 1 || m == 2 {
		v := int64(math.Float32bits(float32(w.c.move(w))))
		w.p32(0x683fcf, v)
		b3 := int64(0)
		if m == 2 {
			b3 = 1
		}
		w.scene(v, 0, b3)
	} else {
		w.p32(0x683fcf, 0)
		w.scene(0, 1, 0)
	}
	w.p32(A+8, w.c.step(w))
	if w.any() {
		w.call(lsS58AC00, 0xc, 0x689b34, -1, -1, 0)
		w.call(lsS58AAD0, 0xc, w.u32(w.u32(0x68c09c)+0x89c), w.u32(A+4))
	}
	for j := int64(0); j < 8; j++ {
		if w.u8(w.S(j)) == 0 {
			continue
		}
		D, C := w.D(j), w.C(j)
		w.p8(D+0x99, 0)
		w.p8(D+0x98, 0)
		w.p32(D+0xa8, 0)
		w.p32(D+0xac, 0)
		w.p32(D+0xb8, 0)
		down := int64(0)
		if w.u32(C+0xb11) == 0 || w.u32(C+0xb01) >= 0xf {
			down = 1
		}
		w.p8(D+0x4c, down)
		w.p32(D+0xb4, 0)
		w.p8(D+0xbc, 0)
		w.call(lsS4EE9D0, C)
	}
	w.c.reval(w)
	for j := int64(0); j < 8; j++ {
		S := w.S(j)
		if w.u8(S) == 0 {
			continue
		}
		D, C := w.D(j), w.C(j)
		w.p8(D+0x9a, 0)
		if w.u32(S+1) != 0 || !w.feature(C, 0x10) || w.u32(C+0x114e) != 0 {
			continue
		}
		t := lsLevelScale(w.levelOf(C)) // 0x5479b0(C, 0x10, 12.0f): push 0x41400000 at 0x4e8626
		if int64(w.r.R(100)) < t {
			w.p8(D+0x9a, 1)
		}
	}
	for i := int64(0); i < w.monCount(); i++ {
		e := w.entry(i)
		if w.u8(e+0x15) == 0 {
			continue
		}
		p := w.u32(e + 0x10)
		w.p8(p+0x15, 0)
		w.p32(p+0x13d, 0)
		w.p32(p+0x141, 0)
		w.p32(p+0x147, 0)
		w.p8(p+4, 0)
		w.p8(p+0x150, 0)
		w.p32(p+0x14c, w.u32(p+0x14c)+1)
		f := int64(0)
		if w.r.R(100) < 75 { // 0x4e86fa
			f = 1
		}
		w.p8(p+0x152, f)
	}
	w.p32(A+0xa5c, w.u32(A+0xa5c)+1)
	for i := int64(0); i < w.monCount(); i++ { // 0x5314f0
		if e := w.entry(i); w.u8(e+0x15) != 0 && w.u32(e+0x2b) > 0 {
			w.c.pick(w, e)
		}
	}
	w.c.init(w)
	w.ui(0x1000ff)
	w.ui(0x80000)
	msg := w.u32(0x68c09c)
	if w.u8(0x68510c) != 0 && w.u32(A+0x910) == 0 {
		for j := int64(0); j < 8; j++ {
			if !w.c.act(w, j) || !w.c.elig(w, j) {
				continue
			}
			S := w.S(j)
			if w.i32(S+0x3d) == 4 {
				w.msgPC(j, w.u32(msg+0x8a0), 0)
			}
			if w.i32(S+0x3d) != 5 {
				continue
			}
			switch w.i32(S + 0x4d) {
			case 1:
				w.msgPC(j, w.u32(msg+0x8a4), w.C(w.i32(S+0x51))+5)
			case 3:
				n := w.name(w.entry(w.find(w.i32(S+0x55), true)))
				w.msgPC(j, w.u32(msg+0x8a4), n)
			default:
				panic("lightning strikes round: 0x4e8846 asserts (target kind not 1 or 3)")
			}
		}
	}
	for i := int64(0); i < w.monCount(); i++ {
		e := w.entry(i)
		if w.u8(e+0x15) == 0 || w.u32(e+0x2b) == 0 || w.u32(e+0x107) >= 0xe {
			continue
		}
		if w.u32(e+0x2e1) == 1 {
			w.msgMon(e, w.u32(msg+0x8a0), 0)
		}
		if w.u32(e+0x2e1) != 8 {
			continue
		}
		switch w.u32(e + 0x2ba) {
		case 3:
			n := w.name(w.entry(w.find(w.i32(e+0x2c2), true)))
			w.msgMon(e, w.u32(msg+0x8a4), n)
		case 1:
			w.msgMon(e, w.u32(msg+0x8a4), w.C(w.i32(e+0x2be))+5)
		default:
			panic("lightning strikes round: 0x4e896d asserts (target kind not 1 or 3)")
		}
	}
	w.p8(A+0xa60, 1)
}

// feature is 0x547940(C, code): class row C+0x69, race row C+0x71 (code 0x10 never reads +0xa79).
func (w *lsW) feature(C, code int64) bool {
	v := make([]int64, 58)
	v[54], v[55] = w.i32(C+0x69), w.i32(C+0x71)
	if v[54] < -1 || v[54] > 14 || v[55] < -1 || v[55] > 15 {
		panic("lightning strikes round: class or race outside the feature tables")
	}
	return pcActionFeature(v, int32(code))
}

// levelOf is the class level 0x5479b0 reads: C+0x8d+4*class (+0x89 for class -1).
func (w *lsW) levelOf(C int64) int64 { return w.u32(C + 0x8d + 4*w.i32(C+0x69)) }

// lsLevelScale is byte(ftol(0x5479b0(C, 0x10, 12.0f))).
func lsLevelScale(level int64) int64 {
	return int64(byte(chdFtol(chdScale(chdF(12), uint32(level)))))
}

func lsNode(c *lsCallees) func([]int64, Rand) []int64 {
	return func(in []int64, r Rand) []int64 {
		if len(in) != len(lsIn) {
			panic("lightning strikes round: wrong input count")
		}
		w := &lsW{cgW: cgW{m: map[int64]byte{}}, r: r, c: c}
		for _, s := range lsInit {
			w.wr(s.a, s.s, s.v)
		}
		for i, s := range lsIn {
			w.wr(s.a, abs(s.s), in[i])
		}
		w.run()
		out := make([]int64, len(lsOut))
		for i, s := range lsOut {
			if s.s < 0 {
				out[i] = w.rs(s.a, -s.s)
			} else {
				out[i] = w.rd(s.a, s.s)
			}
		}
		return out
	}
}

func lightningStrikesRoundPrepareNode(in []int64, r Rand) []int64 { return lsNode(&lsReal)(in, r) }

func init() {
	Nodes["004e8370"] = Node{Addr: "004e8370", In: lsInNames, Out: lsOutNames, F: lightningStrikesRoundPrepareNode, Emu: lsNode(&lsEmu)}
}

func lsB(v int64) bool { return byte(v) != 0 }

var lsEmu = lsCallees{
	elig: func(w *lsW, x int64) bool { return lsB(w.ret(lsS4E79A0, w.call(lsS4E79A0, x, 1, 0, 0))) },
	act:  func(w *lsW, x int64) bool { return lsB(w.ret(lsS524A10, w.call(lsS524A10, x))) },
	tgt:  func(w *lsW, x int64) bool { return lsB(w.ret(lsS53C270, w.call(lsS53C270, x))) },
	off: func(w *lsW, x int64) bool {
		S := w.S(x)
		return lsB(w.ret(lsS547310, w.call(lsS547310, w.C(x), w.i32(S+0x3d), w.i32(S+0x41), S+0x45)))
	},
	step: func(w *lsW) int64 { w.call(lsS4F0500); return w.ret(lsS4F0500, 0) },
	move: func(w *lsW) float64 {
		w.call(lsS4EFFA0)
		return math.Float64frombits(uint64(w.rd(lsStubs[lsS4EFFA0].ret, 8)))
	},
	area: func(w *lsW, id, x, rec int64) {
		tbl := 0x683790 + 0x118*x
		w.call(lsS4FD030, id, 1, 0, rec, tbl, 0, 0)
		w.p32(tbl+4, w.u32(lsTcount))
		w.p32(tbl+0xc, lsTlist)
	},
	init:  func(w *lsW) { w.call(lsS4E89D0) },
	pick:  func(w *lsW, e int64) { w.call(lsS531540, e) },
	reval: func(w *lsW) { w.call(lsS536400) },
}

// lsReal runs the registered Go nodes. Exact from the live world: 0x53c270,
// 0x524a10, 0x4f0500. The others take a parked row of their own case and
// override what this world decides (slot, context, party records, class and
// race, spell id, party mode); their written results go back into the world.
var lsReal = lsCallees{
	elig: func(w *lsW, x int64) bool {
		w.call(lsS4E79A0, x, 1, 0, 0)
		v := append([]int64(nil), lsPark4e79a0[:]...)
		v[0], v[1] = x, 1               // characterSlot, attackContext
		for j := int64(0); j < 8; j++ { // present/hp/status 0..7
			v[30+3*j], v[31+3*j], v[32+3*j] = w.u8(w.S(j)), w.i32(w.C(j)+0xb11), w.u32(w.C(j)+0xb01)
		}
		v[54], v[55] = w.i32(w.C(x)+0x69), w.i32(w.C(x)+0x71) // profession, race
		return lsB(pcActionEligibilityNode(v, w.r)[0])
	},
	act: func(w *lsW, x int64) bool {
		w.call(lsS524A10, x)
		return lsB(characterActionEligibilityNode([]int64{x, w.u8(w.S(x)), w.u32(w.C(x) + 0xb11), w.u32(w.C(x) + 0xb01)}, w.r)[0])
	},
	tgt: func(w *lsW, x int64) bool {
		w.call(lsS53C270, x)
		v := []int64{x}
		for j := int64(0); j < 8; j++ {
			v = append(v, w.u8(w.S(j)), w.i32(w.C(j)+0xb11), w.u32(w.C(j)+0xb01))
		}
		return lsB(characterTargetableNode(v, w.r)[0])
	},
	off: func(w *lsW, x int64) bool {
		S := w.S(x)
		w.call(lsS547310, w.C(x), w.i32(S+0x3d), w.i32(S+0x41), S+0x45)
		v := append([]int64(nil), lsPark547310[:]...)
		v[0], v[1] = w.i32(S+0x3d), w.i32(S+0x41) // kind, spellArg
		v[4], v[5] = w.i32(w.C(x)+0x69), w.i32(w.C(x)+0x71)
		return lsB(offensiveActionAllowedNode(v, w.r)[0])
	},
	step: func(w *lsW) int64 {
		w.call(lsS4F0500)
		return combatPhaseStepNode([]int64{w.u32(w.A() + 0x910)}, w.r)[0]
	},
	move: func(w *lsW) float64 {
		w.call(lsS4EFFA0)
		v := append([]int64(nil), lsPark4effa0[:]...)
		v[0] = w.u32(w.A() + 0x910)
		for j := int64(0); j < 8; j++ {
			v[1+5*j] = w.u8(w.S(j))
		}
		return math.Float64frombits(uint64(partyMovementAllowanceNode(v, w.r)[0]))
	},
	area: func(w *lsW, id, x, rec int64) {
		tbl := 0x683790 + 0x118*x
		w.call(lsS4FD030, id, 1, 0, rec, tbl, 0, 0)
		v := append([]int64(nil), lsPark4fd030[:]...)
		v[13] = id // spellID
		o := spellAreaTargetSelectionNode(v, w.r)
		if o[0] > 4 {
			panic("lightning strikes round: 0x4fd030 monster list past the four-id fixture")
		}
		for i := int64(0); i < o[0]; i++ {
			w.p32(lsTlist+4*i, o[2+i])
		}
		w.p32(tbl+4, o[0])
		w.p32(tbl+0xc, lsTlist)
	},
	init: func(w *lsW) {
		w.call(lsS4E89D0)
		o := combatInitiativePhaseNode(append([]int64(nil), lsPark4e89d0[:]...), w.r)
		for i := int64(0); i < w.monCount(); i++ {
			w.p32(w.entry(i)+0x2e1, o[26+3*i]) // monsterN_field2e1
		}
	},
	pick: func(w *lsW, e int64) {
		w.call(lsS531540, e)
		w.p32(e+0x2e1, monsterActionChoiceNode(append([]int64(nil), lsPark531540[:]...), w.r)[0]) // M.2e1
	},
	reval: func(w *lsW) {
		w.call(lsS536400)
		o := partyTargetRevalidateNode(append([]int64(nil), lsPark536400[:]...), w.r)
		for x := int64(0); x < 4; x++ { // the callee's world has four slots: s<x>.r4d.0..7
			at := lsIdx536400(x)
			for k := int64(0); k < 8; k++ {
				w.p32(w.S(x)+0x4d+4*k, o[at+k])
			}
		}
	},
}

func lsIdx536400(x int64) int64 {
	n := "s" + strconv.Itoa(int(x)) + ".r4d.0"
	for i, s := range cgOutNames[:cg6OutSplit] {
		if s == n {
			return int64(i)
		}
	}
	panic("lightning strikes round: 0x536400 output " + n + " missing")
}

// Parked rows: boundary row 0 of each callee's case (the callee's own default world).
// ---- generated by the ticket's scratch generator from the case layout ----

var lsInit = []cgInitCell{{0x6836a8, 4, 0x20000000}, {0x68edcc, 4, 0x20001000}, {0x685178, 4, 0x20001400}, {0x685174, 4, 0x20002000}, {0x683fad, 4, 0x2000e400}, {0x68c09c, 4, 0x2000e500}, {0x659c0c, 4, 0x20014800}, {0x69bf4c, 4, 0x20014a00}, {0x652dac, 4, 0x20014b00}, {0x69bf3c, 4, 0x20014c00}, {0x69b9a0, 4, 0x20014d00}, {0x68516c, 4, 0x20014e00}, {0x683fb9, 4, 0x0}, {0x69b9a4, 4, 0x0}, {0x683f78, 4, 0x4}, {0x6840c7, 4, 0x20014000}, {0x6840cb, 4, 0x20014200}, {0x6840cf, 4, 0x20014400}, {0x6840d3, 4, 0x20014600}, {0x2000e400, 4, 0x2000e410}, {0x2000e410, 4, 0x20010000}, {0x20010004, 4, 0x0}, {0x2001000c, 4, 0x20011800}, {0x20010010, 4, 0x20011000}, {0x200119d8, 4, 0x20012000}, {0x2000e414, 4, 0x20010400}, {0x20010404, 4, 0x0}, {0x2001040c, 4, 0x20011a00}, {0x20010410, 4, 0x20011200}, {0x20011bd8, 4, 0x20012800}, {0x2000e418, 4, 0x20010800}, {0x20010804, 4, 0x0}, {0x2001080c, 4, 0x20011c00}, {0x20010810, 4, 0x20011400}, {0x20011dd8, 4, 0x20013000}, {0x2000e41c, 4, 0x20010c00}, {0x20010c04, 4, 0x0}, {0x20010c0c, 4, 0x20011e00}, {0x20010c10, 4, 0x20011600}, {0x20011fd8, 4, 0x20013800}, {0x2000ed9c, 4, 0x2000ee00}, {0x2000eda0, 4, 0x2000ee00}, {0x2000eda4, 4, 0x2000ee20}, {0x2000ee04, 2, 0x0}, {0x2000ee24, 2, 0x25}, {0x2000ee26, 2, 0x73}, {0x2000ee28, 2, 0x0}, {0x20014a00, 4, 0x5ed5a4}, {0x20001419, 4, 0x20014d00}, {0x20001449, 4, 0x20014d00}, {0x200014a5, 4, 0x20014d00}, {0x2000200d, 2, 0x0}, {0x683794, 4, 0x0}, {0x68379c, 4, 0x0}, {0x2000151f, 4, 0x20014d00}, {0x2000154f, 4, 0x20014d00}, {0x200015ab, 4, 0x20014d00}, {0x2000386f, 2, 0x0}, {0x6838ac, 4, 0x0}, {0x6838b4, 4, 0x0}, {0x20001625, 4, 0x20014d00}, {0x20001655, 4, 0x20014d00}, {0x200016b1, 4, 0x20014d00}, {0x200050d1, 2, 0x0}, {0x6839c4, 4, 0x0}, {0x6839cc, 4, 0x0}, {0x2000172b, 4, 0x20014d00}, {0x2000175b, 4, 0x20014d00}, {0x200017b7, 4, 0x20014d00}, {0x20006933, 2, 0x0}, {0x683adc, 4, 0x0}, {0x683ae4, 4, 0x0}, {0x20001831, 4, 0x20014d00}, {0x20001861, 4, 0x20014d00}, {0x200018bd, 4, 0x20014d00}, {0x20008195, 2, 0x0}, {0x683bf4, 4, 0x0}, {0x683bfc, 4, 0x0}, {0x20001937, 4, 0x20014d00}, {0x20001967, 4, 0x20014d00}, {0x200019c3, 4, 0x20014d00}, {0x200099f7, 2, 0x0}, {0x683d0c, 4, 0x0}, {0x683d14, 4, 0x0}, {0x20001a3d, 4, 0x20014d00}, {0x20001a6d, 4, 0x20014d00}, {0x20001ac9, 4, 0x20014d00}, {0x2000b259, 2, 0x0}, {0x683e24, 4, 0x0}, {0x683e2c, 4, 0x0}, {0x20001b43, 4, 0x20014d00}, {0x20001b73, 4, 0x20014d00}, {0x20001bcf, 4, 0x20014d00}, {0x2000cabb, 2, 0x0}, {0x683f3c, 4, 0x0}, {0x683f44, 4, 0x0}}
var lsIn = []cgCell{{0x6850d2, 1}, {0x68510c, 1}, {0x683f94, 1}, {0x683f95, 1}, {0x683f96, 1}, {0x683fe7, 4}, {0x68ec78, -4}, {0x68518d, -4}, {0x683fdb, -4}, {0x6875ef, -4}, {0x659c14, 4}, {0x683fcf, 4}, {0x68407f, 4}, {0x684083, 4}, {0x684087, 4}, {0x6840af, 4}, {0x6875c3, 2}, {0x6875c5, 2}, {0x6875c7, 2}, {0x6875c9, 2}, {0x6875cb, 2}, {0x6875cd, 2}, {0x6875cf, 2}, {0x6875d1, 2}, {0x20000000, 1}, {0x20000001, 1}, {0x20000004, 4}, {0x20000008, 4}, {0x20000908, 4}, {0x2000090c, 4}, {0x20000910, 4}, {0x20000918, 4}, {0x20000a54, 1}, {0x20000a55, 1}, {0x20000a58, 1}, {0x20000a59, 1}, {0x20000a5c, 4}, {0x20000a60, 1}, {0x2000004c, 1}, {0x20000098, 1}, {0x20000099, 1}, {0x2000009a, 1}, {0x200000a8, 4}, {0x200000ac, 4}, {0x200000b4, 4}, {0x200000b8, 4}, {0x200000bc, 1}, {0x20000120, 1}, {0x2000016c, 1}, {0x2000016d, 1}, {0x2000016e, 1}, {0x2000017c, 4}, {0x20000180, 4}, {0x20000188, 4}, {0x2000018c, 4}, {0x20000190, 1}, {0x200001f4, 1}, {0x20000240, 1}, {0x20000241, 1}, {0x20000242, 1}, {0x20000250, 4}, {0x20000254, 4}, {0x2000025c, 4}, {0x20000260, 4}, {0x20000264, 1}, {0x200002c8, 1}, {0x20000314, 1}, {0x20000315, 1}, {0x20000316, 1}, {0x20000324, 4}, {0x20000328, 4}, {0x20000330, 4}, {0x20000334, 4}, {0x20000338, 1}, {0x2000039c, 1}, {0x200003e8, 1}, {0x200003e9, 1}, {0x200003ea, 1}, {0x200003f8, 4}, {0x200003fc, 4}, {0x20000404, 4}, {0x20000408, 4}, {0x2000040c, 1}, {0x20000470, 1}, {0x200004bc, 1}, {0x200004bd, 1}, {0x200004be, 1}, {0x200004cc, 4}, {0x200004d0, 4}, {0x200004d8, 4}, {0x200004dc, 4}, {0x200004e0, 1}, {0x20000544, 1}, {0x20000590, 1}, {0x20000591, 1}, {0x20000592, 1}, {0x200005a0, 4}, {0x200005a4, 4}, {0x200005ac, 4}, {0x200005b0, 4}, {0x200005b4, 1}, {0x20000618, 1}, {0x20000664, 1}, {0x20000665, 1}, {0x20000666, 1}, {0x20000674, 4}, {0x20000678, 4}, {0x20000680, 4}, {0x20000684, 4}, {0x20000688, 1}, {0x200012ca, 2}, {0x200012f0, -4}, {0x200012f4, -4}, {0x200012f8, 1}, {0x200010f4, 4}, {0x20001154, 1}, {0x200012c0, 1}, {0x200012c8, 1}, {0x200012dc, 4}, {0x200012e0, 4}, {0x20014a05, 1}, {0x20014a34, 1}, {0x20014b00, 4}, {0x20014b24, 4}, {0x20014b28, 4}, {0x20014b2c, 4}, {0x20014b30, 4}, {0x20014b4c, 4}, {0x20014b50, 4}, {0x20014b54, 4}, {0x20014b7c, 4}, {0x20014b80, 4}, {0x20014b84, 4}, {0x20014d00, -4}, {0x20014d04, -4}, {0x20014d00, -4}, {0x20014e63, 1}, {0x20014f70, 1}, {0x2001507d, 1}, {0x2001518a, 1}, {0x20015310, 4}, {0x20015300, -4}, {0x20015304, -4}, {0x20015308, -4}, {0x2001530c, -4}, {0x683794, 4}, {0x6838ac, 4}, {0x6839c4, 4}, {0x683adc, 4}, {0x683bf4, 4}, {0x683d0c, 4}, {0x683e24, 4}, {0x683f3c, 4}, {0x20001400, 1}, {0x20001401, -4}, {0x2000143d, -4}, {0x20001441, -4}, {0x2000144d, -4}, {0x20001451, -4}, {0x20001455, -4}, {0x20001459, -4}, {0x20001475, -4}, {0x200014f1, 1}, {0x20001506, 1}, {0x20001507, -4}, {0x20001543, -4}, {0x20001547, -4}, {0x20001553, -4}, {0x20001557, -4}, {0x2000155b, -4}, {0x2000155f, -4}, {0x2000157b, -4}, {0x200015f7, 1}, {0x2000160c, 1}, {0x2000160d, -4}, {0x20001649, -4}, {0x2000164d, -4}, {0x20001659, -4}, {0x2000165d, -4}, {0x20001661, -4}, {0x20001665, -4}, {0x20001681, -4}, {0x200016fd, 1}, {0x20001712, 1}, {0x20001713, -4}, {0x2000174f, -4}, {0x20001753, -4}, {0x2000175f, -4}, {0x20001763, -4}, {0x20001767, -4}, {0x2000176b, -4}, {0x20001787, -4}, {0x20001803, 1}, {0x20001818, 1}, {0x20001819, -4}, {0x20001855, -4}, {0x20001859, -4}, {0x20001865, -4}, {0x20001869, -4}, {0x2000186d, -4}, {0x20001871, -4}, {0x2000188d, -4}, {0x20001909, 1}, {0x2000191e, 1}, {0x2000191f, -4}, {0x2000195b, -4}, {0x2000195f, -4}, {0x2000196b, -4}, {0x2000196f, -4}, {0x20001973, -4}, {0x20001977, -4}, {0x20001993, -4}, {0x20001a0f, 1}, {0x20001a24, 1}, {0x20001a25, -4}, {0x20001a61, -4}, {0x20001a65, -4}, {0x20001a71, -4}, {0x20001a75, -4}, {0x20001a79, -4}, {0x20001a7d, -4}, {0x20001a99, -4}, {0x20001b15, 1}, {0x20001b2a, 1}, {0x20001b2b, -4}, {0x20001b67, -4}, {0x20001b6b, -4}, {0x20001b77, -4}, {0x20001b7b, -4}, {0x20001b7f, -4}, {0x20001b83, -4}, {0x20001b9f, -4}, {0x20001c1b, 1}, {0x20002005, 2}, {0x20002007, 2}, {0x20002009, 2}, {0x2000200b, 2}, {0x20002069, -4}, {0x20002071, -4}, {0x20002089, 4}, {0x2000208d, 4}, {0x20002091, 4}, {0x20002095, 4}, {0x20002099, 4}, {0x2000209d, 4}, {0x200020a1, 4}, {0x200020a5, 4}, {0x200020a9, 4}, {0x200020ad, 4}, {0x200020b1, 4}, {0x200020b5, 4}, {0x200020b9, 4}, {0x200020bd, 4}, {0x200020c1, 4}, {0x200020c5, 4}, {0x20002b01, 4}, {0x20002b11, -4}, {0x2000314e, 4}, {0x20003867, 2}, {0x20003869, 2}, {0x2000386b, 2}, {0x2000386d, 2}, {0x200038cb, -4}, {0x200038d3, -4}, {0x200038eb, 4}, {0x200038ef, 4}, {0x200038f3, 4}, {0x200038f7, 4}, {0x200038fb, 4}, {0x200038ff, 4}, {0x20003903, 4}, {0x20003907, 4}, {0x2000390b, 4}, {0x2000390f, 4}, {0x20003913, 4}, {0x20003917, 4}, {0x2000391b, 4}, {0x2000391f, 4}, {0x20003923, 4}, {0x20003927, 4}, {0x20004363, 4}, {0x20004373, -4}, {0x200049b0, 4}, {0x200050c9, 2}, {0x200050cb, 2}, {0x200050cd, 2}, {0x200050cf, 2}, {0x2000512d, -4}, {0x20005135, -4}, {0x2000514d, 4}, {0x20005151, 4}, {0x20005155, 4}, {0x20005159, 4}, {0x2000515d, 4}, {0x20005161, 4}, {0x20005165, 4}, {0x20005169, 4}, {0x2000516d, 4}, {0x20005171, 4}, {0x20005175, 4}, {0x20005179, 4}, {0x2000517d, 4}, {0x20005181, 4}, {0x20005185, 4}, {0x20005189, 4}, {0x20005bc5, 4}, {0x20005bd5, -4}, {0x20006212, 4}, {0x2000692b, 2}, {0x2000692d, 2}, {0x2000692f, 2}, {0x20006931, 2}, {0x2000698f, -4}, {0x20006997, -4}, {0x200069af, 4}, {0x200069b3, 4}, {0x200069b7, 4}, {0x200069bb, 4}, {0x200069bf, 4}, {0x200069c3, 4}, {0x200069c7, 4}, {0x200069cb, 4}, {0x200069cf, 4}, {0x200069d3, 4}, {0x200069d7, 4}, {0x200069db, 4}, {0x200069df, 4}, {0x200069e3, 4}, {0x200069e7, 4}, {0x200069eb, 4}, {0x20007427, 4}, {0x20007437, -4}, {0x20007a74, 4}, {0x2000818d, 2}, {0x2000818f, 2}, {0x20008191, 2}, {0x20008193, 2}, {0x200081f1, -4}, {0x200081f9, -4}, {0x20008211, 4}, {0x20008215, 4}, {0x20008219, 4}, {0x2000821d, 4}, {0x20008221, 4}, {0x20008225, 4}, {0x20008229, 4}, {0x2000822d, 4}, {0x20008231, 4}, {0x20008235, 4}, {0x20008239, 4}, {0x2000823d, 4}, {0x20008241, 4}, {0x20008245, 4}, {0x20008249, 4}, {0x2000824d, 4}, {0x20008c89, 4}, {0x20008c99, -4}, {0x200092d6, 4}, {0x200099ef, 2}, {0x200099f1, 2}, {0x200099f3, 2}, {0x200099f5, 2}, {0x20009a53, -4}, {0x20009a5b, -4}, {0x20009a73, 4}, {0x20009a77, 4}, {0x20009a7b, 4}, {0x20009a7f, 4}, {0x20009a83, 4}, {0x20009a87, 4}, {0x20009a8b, 4}, {0x20009a8f, 4}, {0x20009a93, 4}, {0x20009a97, 4}, {0x20009a9b, 4}, {0x20009a9f, 4}, {0x20009aa3, 4}, {0x20009aa7, 4}, {0x20009aab, 4}, {0x20009aaf, 4}, {0x2000a4eb, 4}, {0x2000a4fb, -4}, {0x2000ab38, 4}, {0x2000b251, 2}, {0x2000b253, 2}, {0x2000b255, 2}, {0x2000b257, 2}, {0x2000b2b5, -4}, {0x2000b2bd, -4}, {0x2000b2d5, 4}, {0x2000b2d9, 4}, {0x2000b2dd, 4}, {0x2000b2e1, 4}, {0x2000b2e5, 4}, {0x2000b2e9, 4}, {0x2000b2ed, 4}, {0x2000b2f1, 4}, {0x2000b2f5, 4}, {0x2000b2f9, 4}, {0x2000b2fd, 4}, {0x2000b301, 4}, {0x2000b305, 4}, {0x2000b309, 4}, {0x2000b30d, 4}, {0x2000b311, 4}, {0x2000bd4d, 4}, {0x2000bd5d, -4}, {0x2000c39a, 4}, {0x2000cab3, 2}, {0x2000cab5, 2}, {0x2000cab7, 2}, {0x2000cab9, 2}, {0x2000cb17, -4}, {0x2000cb1f, -4}, {0x2000cb37, 4}, {0x2000cb3b, 4}, {0x2000cb3f, 4}, {0x2000cb43, 4}, {0x2000cb47, 4}, {0x2000cb4b, 4}, {0x2000cb4f, 4}, {0x2000cb53, 4}, {0x2000cb57, 4}, {0x2000cb5b, 4}, {0x2000cb5f, 4}, {0x2000cb63, 4}, {0x2000cb67, 4}, {0x2000cb6b, 4}, {0x2000cb6f, 4}, {0x2000cb73, 4}, {0x2000d5af, 4}, {0x2000d5bf, -4}, {0x2000dbfc, 4}, {0x2000e408, -4}, {0x20010000, -4}, {0x20010008, -4}, {0x20010014, 1}, {0x20010015, 1}, {0x20010016, 1}, {0x2001002b, 4}, {0x20010107, 4}, {0x200102e1, 4}, {0x200102ba, 4}, {0x200102be, -4}, {0x200102c2, -4}, {0x200125bc, 1}, {0x200120a4, 1}, {0x200120a7, 1}, {0x20011004, 1}, {0x20011015, 1}, {0x2001113d, 4}, {0x20011141, 4}, {0x20011147, 4}, {0x2001114c, 4}, {0x20011150, 1}, {0x20011152, 1}, {0x20014187, 2}, {0x20010400, -4}, {0x20010408, -4}, {0x20010414, 1}, {0x20010415, 1}, {0x20010416, 1}, {0x2001042b, 4}, {0x20010507, 4}, {0x200106e1, 4}, {0x200106ba, 4}, {0x200106be, -4}, {0x200106c2, -4}, {0x20012dbc, 1}, {0x200128a4, 1}, {0x200128a7, 1}, {0x20011204, 1}, {0x20011215, 1}, {0x2001133d, 4}, {0x20011341, 4}, {0x20011347, 4}, {0x2001134c, 4}, {0x20011350, 1}, {0x20011352, 1}, {0x20014387, 2}, {0x20010800, -4}, {0x20010808, -4}, {0x20010814, 1}, {0x20010815, 1}, {0x20010816, 1}, {0x2001082b, 4}, {0x20010907, 4}, {0x20010ae1, 4}, {0x20010aba, 4}, {0x20010abe, -4}, {0x20010ac2, -4}, {0x200135bc, 1}, {0x200130a4, 1}, {0x200130a7, 1}, {0x20011404, 1}, {0x20011415, 1}, {0x2001153d, 4}, {0x20011541, 4}, {0x20011547, 4}, {0x2001154c, 4}, {0x20011550, 1}, {0x20011552, 1}, {0x20014587, 2}, {0x20010c00, -4}, {0x20010c08, -4}, {0x20010c14, 1}, {0x20010c15, 1}, {0x20010c16, 1}, {0x20010c2b, 4}, {0x20010d07, 4}, {0x20010ee1, 4}, {0x20010eba, 4}, {0x20010ebe, -4}, {0x20010ec2, -4}, {0x20013dbc, 1}, {0x200138a4, 1}, {0x200138a7, 1}, {0x20011604, 1}, {0x20011615, 1}, {0x2001173d, 4}, {0x20011741, 4}, {0x20011747, 4}, {0x2001174c, 4}, {0x20011750, 1}, {0x20011752, 1}, {0x20014787, 2}, {0x2000ee00, 2}, {0x2000ee02, 2}, {0x2000ee20, 2}, {0x2000ee22, 2}, {0x20018000, 4}, {0x20018004, 4}, {0x20018008, 4}, {0x2001800c, 4}, {0x20018010, 4}, {0x20018014, 4}, {0x20018018, 4}, {0x2001801c, 4}, {0x20018020, 4}, {0x20018024, 4}, {0x20018028, 4}, {0x2001802c, 4}, {0x20018030, 4}, {0x20018034, 4}, {0x20018038, 4}, {0x2001803c, 4}, {0x20018040, 4}, {0x20018044, 4}, {0x20018048, 4}, {0x2001804c, 4}, {0x20018050, 4}, {0x20018054, 4}, {0x20018058, 4}, {0x2001805c, 4}, {0x20018060, 4}, {0x20018064, 4}, {0x20018068, 4}, {0x2001806c, 4}, {0x20018070, 4}, {0x20018074, 4}, {0x20018078, 4}, {0x2001807c, 4}, {0x20018080, 4}, {0x20018084, 4}, {0x20018088, 4}, {0x2001808c, 4}, {0x20018090, 4}, {0x20018094, 4}, {0x20018098, 4}, {0x2001809c, 4}, {0x200180a0, 4}, {0x200180a4, 4}, {0x200180a8, 4}, {0x200180ac, 4}, {0x200180b0, 4}, {0x200180b4, 4}, {0x200180b8, 4}, {0x200180bc, 4}, {0x200180c0, 4}, {0x200180c4, 8}}
var lsInNames = []string{"g6850d2", "g68510c", "g683f94", "g683f95", "g683f96", "g683fe7", "g68ec78", "g68518d", "g683fdb", "g6875ef", "g659c14", "g683fcf", "g68407f", "g684083", "g684087", "g6840af", "al0", "al1", "al2", "al3", "al4", "al5", "al6", "al7", "A.0", "A.1", "A.4", "A.8", "A.908", "A.90c", "A.910", "A.918", "A.a54", "A.a55", "A.a58", "A.a59", "A.a5c", "A.a60", "D0.4c", "D0.98", "D0.99", "D0.9a", "D0.a8", "D0.ac", "D0.b4", "D0.b8", "D0.bc", "D1.4c", "D1.98", "D1.99", "D1.9a", "D1.a8", "D1.ac", "D1.b4", "D1.b8", "D1.bc", "D2.4c", "D2.98", "D2.99", "D2.9a", "D2.a8", "D2.ac", "D2.b4", "D2.b8", "D2.bc", "D3.4c", "D3.98", "D3.99", "D3.9a", "D3.a8", "D3.ac", "D3.b4", "D3.b8", "D3.bc", "D4.4c", "D4.98", "D4.99", "D4.9a", "D4.a8", "D4.ac", "D4.b4", "D4.b8", "D4.bc", "D5.4c", "D5.98", "D5.99", "D5.9a", "D5.a8", "D5.ac", "D5.b4", "D5.b8", "D5.bc", "D6.4c", "D6.98", "D6.99", "D6.9a", "D6.a8", "D6.ac", "D6.b4", "D6.b8", "D6.bc", "D7.4c", "D7.98", "D7.99", "D7.9a", "D7.a8", "D7.ac", "D7.b4", "D7.b8", "D7.bc", "ui.2ca", "ui.2f0", "ui.2f4", "ui.2f8", "ui.f4", "ui.154", "ui.2c0", "ui.2c8", "ui.2dc", "ui.2e0", "menu.5", "menu.34", "scene.0", "scene.24", "scene.28", "scene.2c", "scene.30", "scene.4c", "scene.50", "scene.54", "scene.7c", "scene.80", "scene.84", "sel.100", "sel.104", "item.idx", "item0.63", "item1.63", "item2.63", "item3.63", "tcount", "tlist0", "tlist1", "tlist2", "tlist3", "tbl0.4", "tbl1.4", "tbl2.4", "tbl3.4", "tbl4.4", "tbl5.4", "tbl6.4", "tbl7.4", "S0", "S0.1", "S0.3d", "S0.41", "S0.4d", "S0.51", "S0.55", "S0.59", "S0.75", "S0.f1", "S1", "S1.1", "S1.3d", "S1.41", "S1.4d", "S1.51", "S1.55", "S1.59", "S1.75", "S1.f1", "S2", "S2.1", "S2.3d", "S2.41", "S2.4d", "S2.51", "S2.55", "S2.59", "S2.75", "S2.f1", "S3", "S3.1", "S3.3d", "S3.41", "S3.4d", "S3.51", "S3.55", "S3.59", "S3.75", "S3.f1", "S4", "S4.1", "S4.3d", "S4.41", "S4.4d", "S4.51", "S4.55", "S4.59", "S4.75", "S4.f1", "S5", "S5.1", "S5.3d", "S5.41", "S5.4d", "S5.51", "S5.55", "S5.59", "S5.75", "S5.f1", "S6", "S6.1", "S6.3d", "S6.41", "S6.4d", "S6.51", "S6.55", "S6.59", "S6.75", "S6.f1", "S7", "S7.1", "S7.3d", "S7.41", "S7.4d", "S7.51", "S7.55", "S7.59", "S7.75", "S7.f1", "C0.name0", "C0.name1", "C0.name2", "C0.name3", "C0.69", "C0.71", "C0.lvl0", "C0.lvl1", "C0.lvl2", "C0.lvl3", "C0.lvl4", "C0.lvl5", "C0.lvl6", "C0.lvl7", "C0.lvl8", "C0.lvl9", "C0.lvl10", "C0.lvl11", "C0.lvl12", "C0.lvl13", "C0.lvl14", "C0.lvl15", "C0.b01", "C0.b11", "C0.114e", "C1.name0", "C1.name1", "C1.name2", "C1.name3", "C1.69", "C1.71", "C1.lvl0", "C1.lvl1", "C1.lvl2", "C1.lvl3", "C1.lvl4", "C1.lvl5", "C1.lvl6", "C1.lvl7", "C1.lvl8", "C1.lvl9", "C1.lvl10", "C1.lvl11", "C1.lvl12", "C1.lvl13", "C1.lvl14", "C1.lvl15", "C1.b01", "C1.b11", "C1.114e", "C2.name0", "C2.name1", "C2.name2", "C2.name3", "C2.69", "C2.71", "C2.lvl0", "C2.lvl1", "C2.lvl2", "C2.lvl3", "C2.lvl4", "C2.lvl5", "C2.lvl6", "C2.lvl7", "C2.lvl8", "C2.lvl9", "C2.lvl10", "C2.lvl11", "C2.lvl12", "C2.lvl13", "C2.lvl14", "C2.lvl15", "C2.b01", "C2.b11", "C2.114e", "C3.name0", "C3.name1", "C3.name2", "C3.name3", "C3.69", "C3.71", "C3.lvl0", "C3.lvl1", "C3.lvl2", "C3.lvl3", "C3.lvl4", "C3.lvl5", "C3.lvl6", "C3.lvl7", "C3.lvl8", "C3.lvl9", "C3.lvl10", "C3.lvl11", "C3.lvl12", "C3.lvl13", "C3.lvl14", "C3.lvl15", "C3.b01", "C3.b11", "C3.114e", "C4.name0", "C4.name1", "C4.name2", "C4.name3", "C4.69", "C4.71", "C4.lvl0", "C4.lvl1", "C4.lvl2", "C4.lvl3", "C4.lvl4", "C4.lvl5", "C4.lvl6", "C4.lvl7", "C4.lvl8", "C4.lvl9", "C4.lvl10", "C4.lvl11", "C4.lvl12", "C4.lvl13", "C4.lvl14", "C4.lvl15", "C4.b01", "C4.b11", "C4.114e", "C5.name0", "C5.name1", "C5.name2", "C5.name3", "C5.69", "C5.71", "C5.lvl0", "C5.lvl1", "C5.lvl2", "C5.lvl3", "C5.lvl4", "C5.lvl5", "C5.lvl6", "C5.lvl7", "C5.lvl8", "C5.lvl9", "C5.lvl10", "C5.lvl11", "C5.lvl12", "C5.lvl13", "C5.lvl14", "C5.lvl15", "C5.b01", "C5.b11", "C5.114e", "C6.name0", "C6.name1", "C6.name2", "C6.name3", "C6.69", "C6.71", "C6.lvl0", "C6.lvl1", "C6.lvl2", "C6.lvl3", "C6.lvl4", "C6.lvl5", "C6.lvl6", "C6.lvl7", "C6.lvl8", "C6.lvl9", "C6.lvl10", "C6.lvl11", "C6.lvl12", "C6.lvl13", "C6.lvl14", "C6.lvl15", "C6.b01", "C6.b11", "C6.114e", "C7.name0", "C7.name1", "C7.name2", "C7.name3", "C7.69", "C7.71", "C7.lvl0", "C7.lvl1", "C7.lvl2", "C7.lvl3", "C7.lvl4", "C7.lvl5", "C7.lvl6", "C7.lvl7", "C7.lvl8", "C7.lvl9", "C7.lvl10", "C7.lvl11", "C7.lvl12", "C7.lvl13", "C7.lvl14", "C7.lvl15", "C7.b01", "C7.b11", "C7.114e", "count", "M0.id", "M0.8", "M0.14", "M0.15", "M0.16", "M0.2b", "M0.107", "M0.2e1", "M0.2ba", "M0.2be", "M0.2c2", "M0.mask", "M0.a4", "M0.a7", "P0.4", "P0.15", "P0.13d", "P0.141", "P0.147", "P0.14c", "P0.150", "P0.152", "T0.187", "M1.id", "M1.8", "M1.14", "M1.15", "M1.16", "M1.2b", "M1.107", "M1.2e1", "M1.2ba", "M1.2be", "M1.2c2", "M1.mask", "M1.a4", "M1.a7", "P1.4", "P1.15", "P1.13d", "P1.141", "P1.147", "P1.14c", "P1.150", "P1.152", "T1.187", "M2.id", "M2.8", "M2.14", "M2.15", "M2.16", "M2.2b", "M2.107", "M2.2e1", "M2.2ba", "M2.2be", "M2.2c2", "M2.mask", "M2.a4", "M2.a7", "P2.4", "P2.15", "P2.13d", "P2.141", "P2.147", "P2.14c", "P2.150", "P2.152", "T2.187", "M3.id", "M3.8", "M3.14", "M3.15", "M3.16", "M3.2b", "M3.107", "M3.2e1", "M3.2ba", "M3.2be", "M3.2c2", "M3.mask", "M3.a4", "M3.a7", "P3.4", "P3.15", "P3.13d", "P3.141", "P3.147", "P3.14c", "P3.150", "P3.152", "T3.187", "f0.c0", "f0.c1", "f1.c0", "f1.c1", "r_4e79a0_0", "r_4e79a0_1", "r_4e79a0_2", "r_4e79a0_3", "r_4e79a0_4", "r_4e79a0_5", "r_4e79a0_6", "r_4e79a0_7", "r_4e79a0_8", "r_4e79a0_9", "r_4e79a0_10", "r_4e79a0_11", "r_4e79a0_12", "r_4e79a0_13", "r_4e79a0_14", "r_4e79a0_15", "r_4e79a0_16", "r_4e79a0_17", "r_4e79a0_18", "r_4e79a0_19", "r_4e79a0_20", "r_4e79a0_21", "r_4e79a0_22", "r_4e79a0_23", "r_524a10_0", "r_524a10_1", "r_524a10_2", "r_524a10_3", "r_524a10_4", "r_524a10_5", "r_524a10_6", "r_524a10_7", "r_53c270_0", "r_53c270_1", "r_53c270_2", "r_53c270_3", "r_53c270_4", "r_53c270_5", "r_53c270_6", "r_53c270_7", "r_547310_0", "r_547310_1", "r_547310_2", "r_547310_3", "r_547310_4", "r_547310_5", "r_547310_6", "r_547310_7", "r_4f0500", "st0_4effa0"}
var lsOut = []cgCell{{0x683fe7, 4}, {0x659c14, 4}, {0x683fcf, 4}, {0x68407f, 4}, {0x684083, 4}, {0x684087, 4}, {0x6840af, 4}, {0x6875c3, 2}, {0x6875c5, 2}, {0x6875c7, 2}, {0x6875c9, 2}, {0x6875cb, 2}, {0x6875cd, 2}, {0x6875cf, 2}, {0x6875d1, 2}, {0x20000000, 1}, {0x20000001, 1}, {0x20000004, 4}, {0x20000008, 4}, {0x20000908, 4}, {0x2000090c, 4}, {0x20000910, 4}, {0x20000918, 4}, {0x20000a55, 1}, {0x20000a5c, 4}, {0x20000a60, 1}, {0x2000004c, 1}, {0x20000098, 1}, {0x20000099, 1}, {0x2000009a, 1}, {0x200000a8, 4}, {0x200000ac, 4}, {0x200000b4, 4}, {0x200000b8, 4}, {0x200000bc, 1}, {0x20000120, 1}, {0x2000016c, 1}, {0x2000016d, 1}, {0x2000016e, 1}, {0x2000017c, 4}, {0x20000180, 4}, {0x20000188, 4}, {0x2000018c, 4}, {0x20000190, 1}, {0x200001f4, 1}, {0x20000240, 1}, {0x20000241, 1}, {0x20000242, 1}, {0x20000250, 4}, {0x20000254, 4}, {0x2000025c, 4}, {0x20000260, 4}, {0x20000264, 1}, {0x200002c8, 1}, {0x20000314, 1}, {0x20000315, 1}, {0x20000316, 1}, {0x20000324, 4}, {0x20000328, 4}, {0x20000330, 4}, {0x20000334, 4}, {0x20000338, 1}, {0x2000039c, 1}, {0x200003e8, 1}, {0x200003e9, 1}, {0x200003ea, 1}, {0x200003f8, 4}, {0x200003fc, 4}, {0x20000404, 4}, {0x20000408, 4}, {0x2000040c, 1}, {0x20000470, 1}, {0x200004bc, 1}, {0x200004bd, 1}, {0x200004be, 1}, {0x200004cc, 4}, {0x200004d0, 4}, {0x200004d8, 4}, {0x200004dc, 4}, {0x200004e0, 1}, {0x20000544, 1}, {0x20000590, 1}, {0x20000591, 1}, {0x20000592, 1}, {0x200005a0, 4}, {0x200005a4, 4}, {0x200005ac, 4}, {0x200005b0, 4}, {0x200005b4, 1}, {0x20000618, 1}, {0x20000664, 1}, {0x20000665, 1}, {0x20000666, 1}, {0x20000674, 4}, {0x20000678, 4}, {0x20000680, 4}, {0x20000684, 4}, {0x20000688, 1}, {0x200010f4, 4}, {0x20001154, 1}, {0x200012c0, 1}, {0x200012c8, 1}, {0x200012dc, 4}, {0x200012e0, 4}, {0x20014a05, 1}, {0x20014a34, 1}, {0x20014b00, 4}, {0x20014b24, 4}, {0x20014b28, 4}, {0x20014b2c, 4}, {0x20014b30, 4}, {0x20014b4c, 4}, {0x20014b50, 4}, {0x20014b54, 4}, {0x20014b7c, 4}, {0x20014b80, 4}, {0x20014b84, 4}, {0x683794, 4}, {0x6838ac, 4}, {0x6839c4, 4}, {0x683adc, 4}, {0x683bf4, 4}, {0x683d0c, 4}, {0x683e24, 4}, {0x683f3c, 4}, {0x200125bc, 1}, {0x20011004, 1}, {0x20011015, 1}, {0x2001113d, 4}, {0x20011141, 4}, {0x20011147, 4}, {0x2001114c, 4}, {0x20011150, 1}, {0x20011152, 1}, {0x20012dbc, 1}, {0x20011204, 1}, {0x20011215, 1}, {0x2001133d, 4}, {0x20011341, 4}, {0x20011347, 4}, {0x2001134c, 4}, {0x20011350, 1}, {0x20011352, 1}, {0x200135bc, 1}, {0x20011404, 1}, {0x20011415, 1}, {0x2001153d, 4}, {0x20011541, 4}, {0x20011547, 4}, {0x2001154c, 4}, {0x20011550, 1}, {0x20011552, 1}, {0x20013dbc, 1}, {0x20011604, 1}, {0x20011615, 1}, {0x2001173d, 4}, {0x20011741, 4}, {0x20011747, 4}, {0x2001174c, 4}, {0x20011750, 1}, {0x20011752, 1}, {0x68379c, 4}, {0x6838b4, 4}, {0x6839cc, 4}, {0x683ae4, 4}, {0x683bfc, 4}, {0x683d14, 4}, {0x683e2c, 4}, {0x683f44, 4}, {0x20016000, 4}, {0x20016004, 4}, {0x20016008, 4}, {0x2001600c, 4}, {0x20016010, 4}, {0x20016014, 4}, {0x20016018, 4}, {0x2001601c, 4}, {0x20016020, 4}, {0x20016024, 4}, {0x20016028, 4}, {0x2001602c, 4}, {0x20016030, 4}, {0x20016034, 4}, {0x20016038, 4}, {0x2001603c, 4}, {0x20016040, 4}, {0x20016044, 4}, {0x20016048, 4}, {0x2001604c, 4}, {0x20016050, 4}, {0x20016054, 4}, {0x20016058, 4}, {0x2001605c, 4}, {0x20016060, 4}, {0x20016064, 4}, {0x20016068, 4}, {0x2001606c, 4}, {0x20016070, 4}, {0x20016074, 4}, {0x20016078, 4}, {0x2001607c, 4}, {0x20016080, 4}, {0x20016084, 4}, {0x20016088, 4}, {0x2001608c, 4}, {0x20016090, 4}, {0x20016094, 4}, {0x20016098, 4}, {0x2001609c, 4}, {0x200160a0, 4}, {0x200160a4, 4}, {0x200160a8, 4}, {0x200160ac, 4}, {0x200160b0, 4}, {0x200160b4, 4}, {0x200160b8, 4}, {0x200160bc, 4}, {0x200160c0, 4}, {0x200160c4, 4}, {0x200160c8, 4}, {0x200160cc, 4}, {0x200160d0, 4}, {0x200160d4, 4}, {0x200160d8, 4}, {0x200160dc, 4}, {0x200160e0, 4}, {0x200160e4, 4}, {0x200160e8, 4}, {0x200160ec, 4}, {0x200160f0, 4}, {0x200160f4, 4}, {0x200160f8, 4}, {0x200160fc, 4}, {0x20016100, 4}, {0x20016104, 4}, {0x20016108, 4}, {0x2001610c, 4}, {0x20016110, 4}, {0x20016114, 4}, {0x20016118, 4}, {0x2001611c, 4}, {0x20016120, 4}, {0x20016124, 4}, {0x20016128, 4}, {0x2001612c, 4}, {0x20016130, 4}, {0x20016134, 4}, {0x20016138, 4}, {0x2001613c, 4}, {0x20016140, 4}, {0x20016144, 4}, {0x20016148, 4}, {0x2001614c, 4}, {0x20016150, 4}, {0x20016154, 4}, {0x20016158, 4}, {0x2001615c, 4}, {0x20016160, 4}, {0x20016164, 4}, {0x20016168, 4}, {0x2001616c, 4}, {0x20016170, 4}, {0x20016174, 4}, {0x20016178, 4}, {0x2001617c, 4}, {0x20016180, 4}, {0x20016184, 4}, {0x20016188, 4}, {0x2001618c, 4}, {0x20016190, 4}, {0x20016194, 4}, {0x20016198, 4}, {0x2001619c, 4}, {0x200161a0, 4}, {0x200161a4, 4}, {0x200161a8, 4}, {0x200161ac, 4}, {0x200161b0, 4}, {0x200161b4, 4}, {0x200161b8, 4}, {0x200161bc, 4}, {0x200161c0, 4}, {0x200161c4, 4}, {0x200161c8, 4}, {0x200161cc, 4}, {0x200161d0, 4}, {0x200161d4, 4}, {0x200161d8, 4}, {0x200161dc, 4}, {0x200161e0, 4}, {0x200161e4, 4}, {0x200161e8, 4}, {0x200161ec, 4}, {0x200161f0, 4}, {0x200161f4, 4}, {0x200161f8, 4}, {0x200161fc, 4}, {0x20016200, 4}, {0x20016204, 4}, {0x20016208, 4}, {0x2001620c, 4}, {0x20016210, 4}, {0x20016214, 4}, {0x20016218, 4}, {0x2001621c, 4}, {0x20016220, 4}, {0x20016224, 4}, {0x20016228, 4}, {0x2001622c, 4}, {0x20016230, 4}, {0x20016234, 4}, {0x20016238, 4}, {0x2001623c, 4}, {0x20016240, 4}, {0x20016244, 4}, {0x20016248, 4}, {0x2001624c, 4}, {0x20016250, 4}, {0x20016254, 4}, {0x20016258, 4}, {0x2001625c, 4}, {0x20016260, 4}, {0x20016264, 4}, {0x20016268, 4}, {0x2001626c, 4}, {0x20016270, 4}, {0x20016274, 4}, {0x20016278, 4}, {0x2001627c, 4}, {0x20016280, 4}, {0x20016284, 4}, {0x20016288, 4}, {0x2001628c, 4}, {0x20016290, 4}, {0x20016294, 4}, {0x20016298, 4}, {0x2001629c, 4}, {0x200162a0, 4}, {0x200162a4, 4}, {0x200162a8, 4}, {0x200162ac, 4}, {0x200162b0, 4}, {0x200162b4, 4}, {0x200162b8, 4}, {0x200162bc, 4}, {0x200162c0, 4}, {0x200162c4, 4}, {0x200162c8, 4}, {0x200162cc, 4}, {0x200162d0, 4}, {0x200162d4, 4}, {0x200162d8, 4}, {0x200162dc, 4}, {0x200162e0, 4}, {0x200162e4, 4}, {0x200162e8, 4}, {0x200162ec, 4}, {0x200162f0, 4}, {0x200162f4, 4}, {0x200162f8, 4}, {0x200162fc, 4}, {0x20016300, 4}, {0x20016304, 4}, {0x20016308, 4}, {0x2001630c, 4}, {0x20016310, 4}, {0x20016314, 4}, {0x20016318, 4}, {0x2001631c, 4}, {0x20016320, 4}, {0x20016324, 4}, {0x20016328, 4}, {0x2001632c, 4}, {0x20016330, 4}, {0x20016334, 4}, {0x20016338, 4}, {0x2001633c, 4}, {0x20016340, 4}, {0x20016344, 4}, {0x20016348, 4}, {0x2001634c, 4}, {0x20016350, 4}, {0x20016354, 4}, {0x20016358, 4}, {0x2001635c, 4}, {0x20016360, 4}, {0x20016364, 4}, {0x20016368, 4}, {0x2001636c, 4}, {0x20016370, 4}, {0x20016374, 4}, {0x20016378, 4}, {0x2001637c, 4}, {0x20016380, 4}, {0x20016384, 4}, {0x20016388, 4}, {0x2001638c, 4}, {0x20016390, 4}, {0x20016394, 4}, {0x20016398, 4}, {0x2001639c, 4}, {0x200163a0, 4}, {0x200163a4, 4}, {0x200163a8, 4}, {0x200163ac, 4}, {0x200163b0, 4}, {0x200163b4, 4}, {0x200163b8, 4}, {0x200163bc, 4}, {0x200163c0, 4}, {0x200163c4, 4}, {0x200163c8, 4}, {0x200163cc, 4}, {0x200163d0, 4}, {0x200163d4, 4}, {0x200163d8, 4}, {0x200163dc, 4}, {0x200163e0, 4}, {0x200163e4, 4}, {0x200163e8, 4}, {0x200163ec, 4}, {0x200163f0, 4}, {0x200163f4, 4}, {0x200163f8, 4}, {0x200163fc, 4}, {0x20016400, 4}, {0x20016404, 4}, {0x20016408, 4}, {0x2001640c, 4}, {0x20016410, 4}, {0x20016414, 4}, {0x20016418, 4}, {0x2001641c, 4}, {0x20016420, 4}, {0x20016424, 4}, {0x20016428, 4}, {0x2001642c, 4}, {0x20016430, 4}, {0x20016434, 4}, {0x20016438, 4}, {0x2001643c, 4}, {0x20016440, 4}, {0x20016444, 4}, {0x20016448, 4}, {0x2001644c, 4}, {0x20016450, 4}, {0x20016454, 4}, {0x20016458, 4}, {0x2001645c, 4}, {0x20016460, 4}, {0x20016464, 4}, {0x20016468, 4}, {0x2001646c, 4}, {0x20016470, 4}, {0x20016474, 4}, {0x20016478, 4}, {0x2001647c, 4}, {0x20016480, 4}, {0x20016484, 4}, {0x20016488, 4}, {0x2001648c, 4}, {0x20016490, 4}, {0x20016494, 4}, {0x20016498, 4}, {0x2001649c, 4}, {0x200164a0, 4}, {0x200164a4, 4}, {0x200164a8, 4}, {0x200164ac, 4}, {0x200164b0, 4}, {0x200164b4, 4}, {0x200164b8, 4}, {0x200164bc, 4}, {0x200164c0, 4}, {0x200164c4, 4}, {0x200164c8, 4}, {0x200164cc, 4}, {0x200164d0, 4}, {0x200164d4, 4}, {0x200164d8, 4}, {0x200164dc, 4}, {0x200164e0, 4}, {0x200164e4, 4}, {0x200164e8, 4}, {0x200164ec, 4}, {0x200164f0, 4}, {0x200164f4, 4}, {0x200164f8, 4}, {0x200164fc, 4}, {0x20016500, 4}, {0x20016504, 4}, {0x20016508, 4}, {0x2001650c, 4}, {0x20016510, 4}, {0x20016514, 4}, {0x20016518, 4}, {0x2001651c, 4}, {0x20016520, 4}, {0x20016524, 4}, {0x20016528, 4}, {0x2001652c, 4}, {0x20016530, 4}, {0x20016534, 4}, {0x20016538, 4}, {0x2001653c, 4}, {0x20016540, 4}, {0x20016544, 4}, {0x20016548, 4}, {0x2001654c, 4}, {0x20016550, 4}, {0x20016554, 4}, {0x20016558, 4}, {0x2001655c, 4}, {0x20016560, 4}, {0x20016564, 4}, {0x20016568, 4}, {0x2001656c, 4}, {0x20016570, 4}, {0x20016574, 4}, {0x20016578, 4}, {0x2001657c, 4}, {0x20016580, 4}, {0x20016584, 4}, {0x20016588, 4}, {0x2001658c, 4}, {0x20016590, 4}, {0x20016594, 4}, {0x20016598, 4}, {0x2001659c, 4}, {0x200165a0, 4}, {0x200165a4, 4}, {0x200165a8, 4}, {0x200165ac, 4}, {0x200165b0, 4}, {0x200165b4, 4}, {0x200165b8, 4}, {0x200165bc, 4}, {0x200165c0, 4}, {0x200165c4, 4}, {0x200165c8, 4}, {0x200165cc, 4}, {0x200165d0, 4}, {0x200165d4, 4}, {0x200165d8, 4}, {0x200165dc, 4}, {0x200165e0, 4}, {0x200165e4, 4}, {0x200165e8, 4}, {0x200165ec, 4}, {0x200165f0, 4}, {0x200165f4, 4}, {0x200165f8, 4}, {0x200165fc, 4}, {0x20016600, 4}, {0x20016604, 4}, {0x20016608, 4}, {0x2001660c, 4}, {0x20016610, 4}, {0x20016614, 4}, {0x20016618, 4}, {0x2001661c, 4}, {0x20016620, 4}, {0x20016624, 4}, {0x20016628, 4}, {0x2001662c, 4}, {0x20016630, 4}, {0x20016634, 4}, {0x20016638, 4}, {0x2001663c, 4}, {0x20016640, 4}, {0x20016644, 4}, {0x20016648, 4}, {0x2001664c, 4}, {0x20016650, 4}, {0x20016654, 4}, {0x20016658, 4}, {0x2001665c, 4}, {0x20016660, 4}, {0x20016664, 4}, {0x20016668, 4}, {0x2001666c, 4}, {0x20016670, 4}, {0x20016674, 4}, {0x20016678, 4}, {0x2001667c, 4}, {0x20016680, 4}, {0x20016684, 4}, {0x20016688, 4}, {0x2001668c, 4}, {0x20016690, 4}, {0x20016694, 4}, {0x20016698, 4}, {0x2001669c, 4}, {0x200166a0, 4}, {0x200166a4, 4}, {0x200166a8, 4}, {0x200166ac, 4}, {0x200166b0, 4}, {0x200166b4, 4}, {0x200166b8, 4}, {0x200166bc, 4}, {0x200166c0, 4}, {0x200166c4, 4}, {0x200166c8, 4}, {0x200166cc, 4}, {0x200166d0, 4}, {0x200166d4, 4}, {0x200166d8, 4}, {0x200166dc, 4}, {0x200166e0, 4}, {0x200166e4, 4}, {0x200166e8, 4}, {0x200166ec, 4}, {0x200166f0, 4}, {0x200166f4, 4}, {0x200166f8, 4}, {0x200166fc, 4}, {0x20016700, 4}, {0x20016704, 4}, {0x20016708, 4}, {0x2001670c, 4}, {0x20016710, 4}, {0x20016714, 4}, {0x20016718, 4}, {0x2001671c, 4}, {0x20016720, 4}, {0x20016724, 4}, {0x20016728, 4}, {0x2001672c, 4}, {0x20016730, 4}, {0x20016734, 4}, {0x20016738, 4}, {0x2001673c, 4}, {0x20016740, 4}, {0x20016744, 4}, {0x20016748, 4}, {0x2001674c, 4}, {0x20016750, 4}, {0x20016754, 4}, {0x20016758, 4}, {0x2001675c, 4}, {0x20016760, 4}, {0x20016764, 4}, {0x20016768, 4}, {0x2001676c, 4}, {0x20016770, 4}, {0x20016774, 4}, {0x20016778, 4}, {0x2001677c, 4}, {0x20016780, 4}, {0x20016784, 4}, {0x20016788, 4}, {0x2001678c, 4}, {0x20016790, 4}, {0x20016794, 4}, {0x20016798, 4}, {0x2001679c, 4}, {0x200167a0, 4}, {0x200167a4, 4}, {0x200167a8, 4}, {0x200167ac, 4}, {0x200167b0, 4}, {0x200167b4, 4}, {0x200167b8, 4}, {0x200167bc, 4}, {0x200167c0, 4}, {0x200167c4, 4}, {0x200167c8, 4}, {0x200167cc, 4}, {0x200167d0, 4}, {0x200167d4, 4}, {0x200167d8, 4}, {0x200167dc, 4}, {0x200167e0, 4}, {0x200167e4, 4}, {0x200167e8, 4}, {0x200167ec, 4}, {0x200167f0, 4}, {0x200167f4, 4}, {0x200167f8, 4}, {0x200167fc, 4}, {0x20016800, 4}, {0x20016804, 4}, {0x20016808, 4}, {0x2001680c, 4}, {0x20016810, 4}, {0x20016814, 4}, {0x20016818, 4}, {0x2001681c, 4}, {0x20016820, 4}, {0x20016824, 4}, {0x20016828, 4}, {0x2001682c, 4}, {0x20016830, 4}, {0x20016834, 4}, {0x20016838, 4}, {0x2001683c, 4}, {0x20016840, 4}, {0x20016844, 4}, {0x20016848, 4}, {0x2001684c, 4}, {0x20016850, 4}, {0x20016854, 4}, {0x20016858, 4}, {0x2001685c, 4}, {0x20016860, 4}, {0x20016864, 4}, {0x20016868, 4}, {0x2001686c, 4}, {0x20016870, 4}}
var lsOutNames = []string{"g683fe7", "g659c14", "g683fcf", "g68407f", "g684083", "g684087", "g6840af", "al0", "al1", "al2", "al3", "al4", "al5", "al6", "al7", "A.0", "A.1", "A.4", "A.8", "A.908", "A.90c", "A.910", "A.918", "A.a55", "A.a5c", "A.a60", "D0.4c", "D0.98", "D0.99", "D0.9a", "D0.a8", "D0.ac", "D0.b4", "D0.b8", "D0.bc", "D1.4c", "D1.98", "D1.99", "D1.9a", "D1.a8", "D1.ac", "D1.b4", "D1.b8", "D1.bc", "D2.4c", "D2.98", "D2.99", "D2.9a", "D2.a8", "D2.ac", "D2.b4", "D2.b8", "D2.bc", "D3.4c", "D3.98", "D3.99", "D3.9a", "D3.a8", "D3.ac", "D3.b4", "D3.b8", "D3.bc", "D4.4c", "D4.98", "D4.99", "D4.9a", "D4.a8", "D4.ac", "D4.b4", "D4.b8", "D4.bc", "D5.4c", "D5.98", "D5.99", "D5.9a", "D5.a8", "D5.ac", "D5.b4", "D5.b8", "D5.bc", "D6.4c", "D6.98", "D6.99", "D6.9a", "D6.a8", "D6.ac", "D6.b4", "D6.b8", "D6.bc", "D7.4c", "D7.98", "D7.99", "D7.9a", "D7.a8", "D7.ac", "D7.b4", "D7.b8", "D7.bc", "ui.f4", "ui.154", "ui.2c0", "ui.2c8", "ui.2dc", "ui.2e0", "menu.5", "menu.34", "scene.0", "scene.24", "scene.28", "scene.2c", "scene.30", "scene.4c", "scene.50", "scene.54", "scene.7c", "scene.80", "scene.84", "tbl0.4", "tbl1.4", "tbl2.4", "tbl3.4", "tbl4.4", "tbl5.4", "tbl6.4", "tbl7.4", "M0.mask", "P0.4", "P0.15", "P0.13d", "P0.141", "P0.147", "P0.14c", "P0.150", "P0.152", "M1.mask", "P1.4", "P1.15", "P1.13d", "P1.141", "P1.147", "P1.14c", "P1.150", "P1.152", "M2.mask", "P2.4", "P2.15", "P2.13d", "P2.141", "P2.147", "P2.14c", "P2.150", "P2.152", "M3.mask", "P3.4", "P3.15", "P3.13d", "P3.141", "P3.147", "P3.14c", "P3.150", "P3.152", "tbl0.c", "tbl1.c", "tbl2.c", "tbl3.c", "tbl4.c", "tbl5.c", "tbl6.c", "tbl7.c", "n_4e79a0", "rec_4e79a0_1_0", "rec_4e79a0_1_1", "rec_4e79a0_1_2", "rec_4e79a0_1_3", "rec_4e79a0_1_4", "rec_4e79a0_1_5", "rec_4e79a0_1_6", "rec_4e79a0_1_7", "rec_4e79a0_1_8", "rec_4e79a0_1_9", "rec_4e79a0_1_10", "rec_4e79a0_1_11", "rec_4e79a0_1_12", "rec_4e79a0_1_13", "rec_4e79a0_1_14", "rec_4e79a0_1_15", "rec_4e79a0_1_16", "rec_4e79a0_1_17", "rec_4e79a0_1_18", "rec_4e79a0_1_19", "rec_4e79a0_1_20", "rec_4e79a0_1_21", "rec_4e79a0_1_22", "rec_4e79a0_1_23", "rec_4e79a0_2_0", "rec_4e79a0_2_1", "rec_4e79a0_2_2", "rec_4e79a0_2_3", "rec_4e79a0_2_4", "rec_4e79a0_2_5", "rec_4e79a0_2_6", "rec_4e79a0_2_7", "rec_4e79a0_2_8", "rec_4e79a0_2_9", "rec_4e79a0_2_10", "rec_4e79a0_2_11", "rec_4e79a0_2_12", "rec_4e79a0_2_13", "rec_4e79a0_2_14", "rec_4e79a0_2_15", "rec_4e79a0_2_16", "rec_4e79a0_2_17", "rec_4e79a0_2_18", "rec_4e79a0_2_19", "rec_4e79a0_2_20", "rec_4e79a0_2_21", "rec_4e79a0_2_22", "rec_4e79a0_2_23", "rec_4e79a0_3_0", "rec_4e79a0_3_1", "rec_4e79a0_3_2", "rec_4e79a0_3_3", "rec_4e79a0_3_4", "rec_4e79a0_3_5", "rec_4e79a0_3_6", "rec_4e79a0_3_7", "rec_4e79a0_3_8", "rec_4e79a0_3_9", "rec_4e79a0_3_10", "rec_4e79a0_3_11", "rec_4e79a0_3_12", "rec_4e79a0_3_13", "rec_4e79a0_3_14", "rec_4e79a0_3_15", "rec_4e79a0_3_16", "rec_4e79a0_3_17", "rec_4e79a0_3_18", "rec_4e79a0_3_19", "rec_4e79a0_3_20", "rec_4e79a0_3_21", "rec_4e79a0_3_22", "rec_4e79a0_3_23", "rec_4e79a0_4_0", "rec_4e79a0_4_1", "rec_4e79a0_4_2", "rec_4e79a0_4_3", "rec_4e79a0_4_4", "rec_4e79a0_4_5", "rec_4e79a0_4_6", "rec_4e79a0_4_7", "rec_4e79a0_4_8", "rec_4e79a0_4_9", "rec_4e79a0_4_10", "rec_4e79a0_4_11", "rec_4e79a0_4_12", "rec_4e79a0_4_13", "rec_4e79a0_4_14", "rec_4e79a0_4_15", "rec_4e79a0_4_16", "rec_4e79a0_4_17", "rec_4e79a0_4_18", "rec_4e79a0_4_19", "rec_4e79a0_4_20", "rec_4e79a0_4_21", "rec_4e79a0_4_22", "rec_4e79a0_4_23", "n_524a10", "rec_524a10_1_0", "rec_524a10_1_1", "rec_524a10_1_2", "rec_524a10_1_3", "rec_524a10_1_4", "rec_524a10_1_5", "rec_524a10_1_6", "rec_524a10_1_7", "n_53c270", "rec_53c270_1_0", "rec_53c270_1_1", "rec_53c270_1_2", "rec_53c270_1_3", "rec_53c270_1_4", "rec_53c270_1_5", "rec_53c270_1_6", "rec_53c270_1_7", "n_547310", "rec_547310_1_0", "rec_547310_1_1", "rec_547310_1_2", "rec_547310_1_3", "rec_547310_1_4", "rec_547310_1_5", "rec_547310_1_6", "rec_547310_1_7", "rec_547310_2_0", "rec_547310_2_1", "rec_547310_2_2", "rec_547310_2_3", "rec_547310_2_4", "rec_547310_2_5", "rec_547310_2_6", "rec_547310_2_7", "rec_547310_3_0", "rec_547310_3_1", "rec_547310_3_2", "rec_547310_3_3", "rec_547310_3_4", "rec_547310_3_5", "rec_547310_3_6", "rec_547310_3_7", "rec_547310_4_0", "rec_547310_4_1", "rec_547310_4_2", "rec_547310_4_3", "rec_547310_4_4", "rec_547310_4_5", "rec_547310_4_6", "rec_547310_4_7", "n_4f0500", "n_4effa0", "n_4ee9d0", "rec_4ee9d0_1_0", "rec_4ee9d0_1_1", "rec_4ee9d0_1_2", "rec_4ee9d0_1_3", "rec_4ee9d0_1_4", "rec_4ee9d0_1_5", "rec_4ee9d0_1_6", "rec_4ee9d0_1_7", "n_536400", "n_4e89d0", "n_531540", "rec_531540_1_0", "rec_531540_1_1", "rec_531540_1_2", "rec_531540_1_3", "n_4fd030", "rec_4fd030_1_0", "rec_4fd030_1_1", "rec_4fd030_1_2", "rec_4fd030_1_3", "rec_4fd030_1_4", "rec_4fd030_1_5", "rec_4fd030_1_6", "rec_4fd030_1_7", "rec_4fd030_2_0", "rec_4fd030_2_1", "rec_4fd030_2_2", "rec_4fd030_2_3", "rec_4fd030_2_4", "rec_4fd030_2_5", "rec_4fd030_2_6", "rec_4fd030_2_7", "rec_4fd030_4_0", "rec_4fd030_4_1", "rec_4fd030_4_2", "rec_4fd030_4_3", "rec_4fd030_4_4", "rec_4fd030_4_5", "rec_4fd030_4_6", "rec_4fd030_4_7", "rec_4fd030_5_0", "rec_4fd030_5_1", "rec_4fd030_5_2", "rec_4fd030_5_3", "rec_4fd030_5_4", "rec_4fd030_5_5", "rec_4fd030_5_6", "rec_4fd030_5_7", "rec_4fd030_7_0", "rec_4fd030_7_1", "rec_4fd030_7_2", "rec_4fd030_7_3", "rec_4fd030_7_4", "rec_4fd030_7_5", "rec_4fd030_7_6", "rec_4fd030_7_7", "n_4f0e80", "rec_4f0e80_2_0", "rec_4f0e80_3_0", "n_4c5eb0", "rec_4c5eb0_1_0", "rec_4c5eb0_1_1", "rec_4c5eb0_1_2", "rec_4c5eb0_1_3", "rec_4c5eb0_1_4", "rec_4c5eb0_1_5", "rec_4c5eb0_1_6", "rec_4c5eb0_1_7", "rec_4c5eb0_1_8", "rec_4c5eb0_1_9", "rec_4c5eb0_1_10", "rec_4c5eb0_1_11", "rec_4c5eb0_1_12", "rec_4c5eb0_1_13", "rec_4c5eb0_1_14", "rec_4c5eb0_1_15", "rec_4c5eb0_1_16", "rec_4c5eb0_1_17", "rec_4c5eb0_1_18", "rec_4c5eb0_1_19", "rec_4c5eb0_1_20", "rec_4c5eb0_1_21", "rec_4c5eb0_1_22", "rec_4c5eb0_1_23", "rec_4c5eb0_1_24", "rec_4c5eb0_1_25", "rec_4c5eb0_1_26", "rec_4c5eb0_1_27", "rec_4c5eb0_1_28", "rec_4c5eb0_1_29", "rec_4c5eb0_1_30", "rec_4c5eb0_1_31", "rec_4c5eb0_1_32", "rec_4c5eb0_1_33", "rec_4c5eb0_1_34", "rec_4c5eb0_1_35", "rec_4c5eb0_1_36", "rec_4c5eb0_1_37", "rec_4c5eb0_1_38", "rec_4c5eb0_1_39", "rec_4c5eb0_1_40", "rec_4c5eb0_1_41", "rec_4c5eb0_1_42", "rec_4c5eb0_1_43", "rec_4c5eb0_1_44", "rec_4c5eb0_1_45", "rec_4c5eb0_1_46", "rec_4c5eb0_1_47", "rec_4c5eb0_1_48", "rec_4c5eb0_1_49", "rec_4c5eb0_1_50", "rec_4c5eb0_1_51", "rec_4c5eb0_1_52", "rec_4c5eb0_1_53", "rec_4c5eb0_1_54", "rec_4c5eb0_1_55", "rec_4c5eb0_1_56", "rec_4c5eb0_1_57", "rec_4c5eb0_1_58", "rec_4c5eb0_1_59", "rec_4c5eb0_1_60", "rec_4c5eb0_1_61", "rec_4c5eb0_1_62", "rec_4c5eb0_1_63", "rec_4c5eb0_2_0", "rec_4c5eb0_2_1", "rec_4c5eb0_2_2", "rec_4c5eb0_2_3", "rec_4c5eb0_2_4", "rec_4c5eb0_2_5", "rec_4c5eb0_2_6", "rec_4c5eb0_2_7", "rec_4c5eb0_2_8", "rec_4c5eb0_2_9", "rec_4c5eb0_2_10", "rec_4c5eb0_2_11", "rec_4c5eb0_2_12", "rec_4c5eb0_2_13", "rec_4c5eb0_2_14", "rec_4c5eb0_2_15", "rec_4c5eb0_2_16", "rec_4c5eb0_2_17", "rec_4c5eb0_2_18", "rec_4c5eb0_2_19", "rec_4c5eb0_2_20", "rec_4c5eb0_2_21", "rec_4c5eb0_2_22", "rec_4c5eb0_2_23", "rec_4c5eb0_2_24", "rec_4c5eb0_2_25", "rec_4c5eb0_2_26", "rec_4c5eb0_2_27", "rec_4c5eb0_2_28", "rec_4c5eb0_2_29", "rec_4c5eb0_2_30", "rec_4c5eb0_2_31", "rec_4c5eb0_2_32", "rec_4c5eb0_2_33", "rec_4c5eb0_2_34", "rec_4c5eb0_2_35", "rec_4c5eb0_2_36", "rec_4c5eb0_2_37", "rec_4c5eb0_2_38", "rec_4c5eb0_2_39", "rec_4c5eb0_2_40", "rec_4c5eb0_2_41", "rec_4c5eb0_2_42", "rec_4c5eb0_2_43", "rec_4c5eb0_2_44", "rec_4c5eb0_2_45", "rec_4c5eb0_2_46", "rec_4c5eb0_2_47", "rec_4c5eb0_2_48", "rec_4c5eb0_2_49", "rec_4c5eb0_2_50", "rec_4c5eb0_2_51", "rec_4c5eb0_2_52", "rec_4c5eb0_2_53", "rec_4c5eb0_2_54", "rec_4c5eb0_2_55", "rec_4c5eb0_2_56", "rec_4c5eb0_2_57", "rec_4c5eb0_2_58", "rec_4c5eb0_2_59", "rec_4c5eb0_2_60", "rec_4c5eb0_2_61", "rec_4c5eb0_2_62", "rec_4c5eb0_2_63", "rec_4c5eb0_3_0", "rec_4c5eb0_3_1", "rec_4c5eb0_3_2", "rec_4c5eb0_3_3", "rec_4c5eb0_3_4", "rec_4c5eb0_3_5", "rec_4c5eb0_3_6", "rec_4c5eb0_3_7", "rec_4c5eb0_3_8", "rec_4c5eb0_3_9", "rec_4c5eb0_3_10", "rec_4c5eb0_3_11", "rec_4c5eb0_3_12", "rec_4c5eb0_3_13", "rec_4c5eb0_3_14", "rec_4c5eb0_3_15", "rec_4c5eb0_3_16", "rec_4c5eb0_3_17", "rec_4c5eb0_3_18", "rec_4c5eb0_3_19", "rec_4c5eb0_3_20", "rec_4c5eb0_3_21", "rec_4c5eb0_3_22", "rec_4c5eb0_3_23", "rec_4c5eb0_3_24", "rec_4c5eb0_3_25", "rec_4c5eb0_3_26", "rec_4c5eb0_3_27", "rec_4c5eb0_3_28", "rec_4c5eb0_3_29", "rec_4c5eb0_3_30", "rec_4c5eb0_3_31", "rec_4c5eb0_3_32", "rec_4c5eb0_3_33", "rec_4c5eb0_3_34", "rec_4c5eb0_3_35", "rec_4c5eb0_3_36", "rec_4c5eb0_3_37", "rec_4c5eb0_3_38", "rec_4c5eb0_3_39", "rec_4c5eb0_3_40", "rec_4c5eb0_3_41", "rec_4c5eb0_3_42", "rec_4c5eb0_3_43", "rec_4c5eb0_3_44", "rec_4c5eb0_3_45", "rec_4c5eb0_3_46", "rec_4c5eb0_3_47", "rec_4c5eb0_3_48", "rec_4c5eb0_3_49", "rec_4c5eb0_3_50", "rec_4c5eb0_3_51", "rec_4c5eb0_3_52", "rec_4c5eb0_3_53", "rec_4c5eb0_3_54", "rec_4c5eb0_3_55", "rec_4c5eb0_3_56", "rec_4c5eb0_3_57", "rec_4c5eb0_3_58", "rec_4c5eb0_3_59", "rec_4c5eb0_3_60", "rec_4c5eb0_3_61", "rec_4c5eb0_3_62", "rec_4c5eb0_3_63", "n_58b410", "rec_58b410_1_0", "rec_58b410_1_1", "rec_58b410_1_2", "rec_58b410_1_3", "rec_58b410_1_4", "rec_58b410_1_5", "rec_58b410_1_6", "rec_58b410_1_7", "rec_58b410_2_0", "rec_58b410_2_1", "rec_58b410_2_2", "rec_58b410_2_3", "rec_58b410_2_4", "rec_58b410_2_5", "rec_58b410_2_6", "rec_58b410_2_7", "rec_58b410_3_0", "rec_58b410_3_1", "rec_58b410_3_2", "rec_58b410_3_3", "rec_58b410_3_4", "rec_58b410_3_5", "rec_58b410_3_6", "rec_58b410_3_7", "rec_58b410_4_0", "rec_58b410_4_1", "rec_58b410_4_2", "rec_58b410_4_3", "rec_58b410_4_4", "rec_58b410_4_5", "rec_58b410_4_6", "rec_58b410_4_7", "n_58aad0", "rec_58aad0_1_0", "rec_58aad0_1_1", "rec_58aad0_1_2", "rec_58aad0_1_3", "rec_58aad0_1_4", "rec_58aad0_1_5", "rec_58aad0_1_6", "rec_58aad0_1_7", "rec_58aad0_1_8", "rec_58aad0_1_9", "rec_58aad0_1_10", "rec_58aad0_1_11", "rec_58aad0_1_12", "rec_58aad0_2_0", "rec_58aad0_2_1", "rec_58aad0_2_2", "rec_58aad0_2_3", "rec_58aad0_2_4", "rec_58aad0_2_5", "rec_58aad0_2_6", "rec_58aad0_2_7", "rec_58aad0_2_8", "rec_58aad0_2_9", "rec_58aad0_2_10", "rec_58aad0_2_11", "rec_58aad0_2_12", "rec_58aad0_3_0", "rec_58aad0_3_1", "rec_58aad0_3_2", "rec_58aad0_3_3", "rec_58aad0_3_4", "rec_58aad0_3_5", "rec_58aad0_3_6", "rec_58aad0_3_7", "rec_58aad0_3_8", "rec_58aad0_3_9", "rec_58aad0_3_10", "rec_58aad0_3_11", "rec_58aad0_3_12", "n_58ac00", "rec_58ac00_1_0", "rec_58ac00_2_0", "rec_58ac00_3_0", "rec_58ac00_4_0", "rec_58ac00_5_0", "n_5182e0", "rec_5182e0_1_0", "rec_5182e0_1_1", "rec_5182e0_1_2", "rec_5182e0_1_3", "rec_5182e0_1_4", "rec_5182e0_1_5", "rec_5182e0_1_6", "rec_5182e0_1_7", "rec_5182e0_1_8", "rec_5182e0_1_9", "rec_5182e0_1_10", "rec_5182e0_1_11", "rec_5182e0_1_12", "rec_5182e0_1_13", "rec_5182e0_1_14", "rec_5182e0_1_15", "rec_5182e0_2_0", "rec_5182e0_2_1", "rec_5182e0_2_2", "rec_5182e0_2_3", "rec_5182e0_2_4", "rec_5182e0_2_5", "rec_5182e0_2_6", "rec_5182e0_2_7", "rec_5182e0_2_8", "rec_5182e0_2_9", "rec_5182e0_2_10", "rec_5182e0_2_11", "rec_5182e0_2_12", "rec_5182e0_2_13", "rec_5182e0_2_14", "rec_5182e0_2_15", "rec_5182e0_3_0", "rec_5182e0_3_1", "rec_5182e0_3_2", "rec_5182e0_3_3", "rec_5182e0_3_4", "rec_5182e0_3_5", "rec_5182e0_3_6", "rec_5182e0_3_7", "rec_5182e0_3_8", "rec_5182e0_3_9", "rec_5182e0_3_10", "rec_5182e0_3_11", "rec_5182e0_3_12", "rec_5182e0_3_13", "rec_5182e0_3_14", "rec_5182e0_3_15", "n_595570", "n_595600", "n_55ee70", "rec_55ee70_1_0", "n_4add30", "rec_4add30_1_0", "n_56aa30", "n_56aab0", "n_5a1890", "n_56a5a0"}

const (
	lsS4E79A0 = iota
	lsS524A10
	lsS53C270
	lsS547310
	lsS4F0500
	lsS4EFFA0
	lsS4EE9D0
	lsS536400
	lsS4E89D0
	lsS531540
	lsS4FD030
	lsS4F0E80
	lsS4C5EB0
	lsS58B410
	lsS58AAD0
	lsS58AC00
	lsS5182E0
	lsS595570
	lsS595600
	lsS55EE70
	lsS4ADD30
	lsS56AA30
	lsS56AAB0
	lsS5A1890
	lsS56A5A0
)

var lsStubs = []lsStub{
	{cnt: 0x20016000, ret: 0x20018000, retN: 24, rec: [][3]int64{{1, 0x20016004, 24}, {2, 0x20016064, 24}, {3, 0x200160c4, 24}, {4, 0x20016124, 24}}},
	{cnt: 0x20016184, ret: 0x20018060, retN: 8, rec: [][3]int64{{1, 0x20016188, 8}}},
	{cnt: 0x200161a8, ret: 0x20018080, retN: 8, rec: [][3]int64{{1, 0x200161ac, 8}}},
	{cnt: 0x200161cc, ret: 0x200180a0, retN: 8, rec: [][3]int64{{1, 0x200161d0, 8}, {2, 0x200161f0, 8}, {3, 0x20016210, 8}, {4, 0x20016230, 8}}},
	{cnt: 0x20016250, ret: 0x200180c0, retN: 0, rec: [][3]int64{}},
	{cnt: 0x20016254, ret: 0x200180c4, retN: 0, rec: [][3]int64{}},
	{cnt: 0x20016258, ret: 0x0, retN: 0, rec: [][3]int64{{1, 0x2001625c, 8}}},
	{cnt: 0x2001627c, ret: 0x0, retN: 0, rec: [][3]int64{}},
	{cnt: 0x20016280, ret: 0x0, retN: 0, rec: [][3]int64{}},
	{cnt: 0x20016284, ret: 0x0, retN: 0, rec: [][3]int64{{1, 0x20016288, 4}}},
	{cnt: 0x20016298, ret: 0x0, retN: 0, rec: [][3]int64{{1, 0x2001629c, 8}, {2, 0x200162bc, 8}, {4, 0x200162dc, 8}, {5, 0x200162fc, 8}, {7, 0x2001631c, 8}}},
	{cnt: 0x2001633c, ret: 0x0, retN: 0, rec: [][3]int64{{2, 0x20016340, 1}, {3, 0x20016344, 1}}},
	{cnt: 0x20016348, ret: 0x0, retN: 0, rec: [][3]int64{{1, 0x2001634c, 64}, {2, 0x2001644c, 64}, {3, 0x2001654c, 64}}},
	{cnt: 0x2001664c, ret: 0x0, retN: 0, rec: [][3]int64{{1, 0x20016650, 8}, {2, 0x20016670, 8}, {3, 0x20016690, 8}, {4, 0x200166b0, 8}}},
	{cnt: 0x200166d0, ret: 0x0, retN: 0, rec: [][3]int64{{1, 0x200166d4, 13}, {2, 0x20016708, 13}, {3, 0x2001673c, 13}}},
	{cnt: 0x20016770, ret: 0x0, retN: 0, rec: [][3]int64{{1, 0x20016774, 1}, {2, 0x20016778, 1}, {3, 0x2001677c, 1}, {4, 0x20016780, 1}, {5, 0x20016784, 1}}},
	{cnt: 0x20016788, ret: 0x0, retN: 0, rec: [][3]int64{{1, 0x2001678c, 16}, {2, 0x200167cc, 16}, {3, 0x2001680c, 16}}},
	{cnt: 0x2001684c, ret: 0x0, retN: 0, rec: [][3]int64{}},
	{cnt: 0x20016850, ret: 0x0, retN: 0, rec: [][3]int64{}},
	{cnt: 0x20016854, ret: 0x0, retN: 0, rec: [][3]int64{{1, 0x20016858, 1}}},
	{cnt: 0x2001685c, ret: 0x0, retN: 0, rec: [][3]int64{{1, 0x20016860, 1}}},
	{cnt: 0x20016864, ret: 0x0, retN: 0, rec: [][3]int64{}},
	{cnt: 0x20016868, ret: 0x0, retN: 0, rec: [][3]int64{}},
	{cnt: 0x2001686c, ret: 0x0, retN: 0, rec: [][3]int64{}},
	{cnt: 0x20016870, ret: 0x0, retN: 0, rec: [][3]int64{}},
}

const lsTcount, lsTlist = 0x20015310, 0x20015300

var lsPark4e79a0 = [264]int64{0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 1, 1, 536936448, 0, 1, 16, 0, 1, 2, 0, -1, -1, 1, 0, 1, 0, 0, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 0, 0, 0, 0, 100, 20, 0, 1, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 5, 1, 10, 10, 10, 10, 10, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 65535, 4294967295, 3, 255, 0, 255, 0, 255, 0, 255, 0, 1, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 1, 1, 1, 0, 0, 0, 0, 0, 0}
var lsPark547310 = [81]int64{0, 5, 5, 0, 0, 0, 0, 65535, 4294967295, 3, 255, 0, 255, 0, 255, 0, 255, 0, 6, 1, 4, 4, 4, 0, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 255, 0, 0, 0, 1, 536877056, 536870912, 700, 5}
var lsPark4effa0 = [41]int64{0, 255, 0, 0, 0, 4294967295, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var lsPark4fd030 = [93]int64{0, 0, 0, 0, 0, 1065353216, 0, 0, 0, 0, 0, 0, 0, 1, 5, 1065353216, 0, 2, 1, 3, -1, -1, 0, 1, 0, 7, 0, 123, 456, 0, 1, 1, 1, 10, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 0, 0, 1, 0, 8, 0, 8, 1065353216, 0, 1073741824}
var lsPark4e89d0 = [222]int64{1, 0, 0, 0, 536950912, 2, 1, 2, 3, 4, 1, 5, 3, 0, 0, 1, 1, 1, 1, 1, 0, 1, 1, 3, 1, 1, 286331153, 64, 1, 5, 3, 1, 0, 1, 1, 1, 1, 1, 1, 2, 1, 4, 1, 1, 286331154, 65, 1, 5, 3, 2, 0, 1, 1, 1, 1, 1, 2, 3, 1, 5, 1, 1, 286331155, 66, 1, 5, 3, 3, 0, 1, 1, 1, 1, 1, 3, 0, 1, 6, 1, 1, 286331156, 67, 1, 5, 3, 4, 0, 1, 1, 1, 1, 1, 0, 1, 1, 7, 1, 1, 286331157, 68, 1, 5, 3, 5, 0, 1, 1, 1, 1, 1, 1, 2, 1, 8, 1, 1, 286331158, 69, 1, 5, 3, 6, 0, 1, 1, 1, 1, 1, 2, 3, 1, 9, 1, 1, 286331159, 70, 1, 5, 3, 7, 0, 1, 1, 1, 1, 1, 3, 0, 1, 10, 1, 1, 286331160, 71, 1, 0, 9, 2, 0, 28672, 48, 1280, 1, 0, 9, 2, 1, 28673, 49, 1281, 1, 0, 9, 2, 2, 28674, 50, 1282, 1, 0, 9, 2, 3, 28675, 51, 1283, 1, 0, 9, 2, 0, 28676, 52, 1284, 1, 0, 9, 2, 1, 28677, 53, 1285, 1, 0, 9, 2, 2, 28678, 54, 1286, 1, 0, 9, 2, 3, 28679, 55, 1287, 10, 20, 30, 40}
var lsPark531540 = [208]int64{0, 0, 0, 0, 0, 1148846080, 0, 0, 0, 0, 0, 0, 1153138688, 0, 0, 0, 1161527296, 0, 0, 0, 1, 1, 17, 34, 51, 68, 85, 0, 0, 0, 3, 3, 1, 1, 1, 0, 1, 10, 0, 1, 1, 100, 100, 100, 10, 0, 0, 0, 0, 0, 0, 17, 34, 51, 68, 85, 102, 119, 0, 0, 4294967295, 0, 0, 1, 1, 0, 50, 0, 1, 1, 1, 0, 50, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 5, 0, 1, 7, 0, 0, 9, 4, 1, 40, 40, 7, 24, 1, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 3, 1, 0, 4, 0, 2, 6, 1, 0, 1, 3, 1, 1, 1, 4, 1, 0, 1, 5, 1, 1, 1, 6, 1, 0, 1, 7, 1, 1, 1, 3, 1, 0, 1, 4, 1, 1, 1, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 5, 6, 7, 8, 9, 10, 11, 12, 60, 1, 0, 119, 1, 2, 3, 4, 5, 6, 7, 8, 165, 182, 199, 7, 1}
var lsPark536400 = [461]int64{0, 1, 0, 1, 0, 0, 0, 1, 1, 2, 1, 3, 3, 0, 5, 0, 0, 0, 0, 1, 0, 1, 1, 595, 1, 4, 9, 0, 1, 0, 536956928, 0, 3, 0, 1, 3, 1, 3, 5, 7, 8, 64, 0, 1, 0, 1, 5, 0, 2, 0, 0, 1, 10, 0, 0, 20, 3, 15, 2, 23, 0, 20, 2, 0, 4, 6, 7, 1, 0, 9, 9, 0, 6, 3, 1, 3, 1, 5, 0, 1, 7, 0, 0, 1, -1, 0, 5, 0, 1, 0, 0, 0, 1, 1, 1, 0, 1148878848, 1148878848, 0, 7, -1, 1, 5, 1, 1, 1, 0, 4, 5, 1, 1, 1, 0, 0, 0, 6, -1, 5, 0, 0, 1, 1, 0, 9, 2, 5, 536893643, 2, 8, 2, 537067528, 0, 0, 0, 1, 1, 18, 595, 596, 596, 0, 596, 0, 596, 596, 596, 596, 595, 0, 0, 100, 0, 1, 2, 0, 1, 1, 1, 8, 1, 537067528, 1, -1, 1, 1, 1, 0, 0, 0, 1, 1, 0, 5, 1148846080, 0, 0, 0, 3, 1, 0, 1, 0, 1, 0, 1, 6, 0, 5, 1, 1, 0, 0, 0, 7, 1, 5, 5, 0, 0, 0, 0, -1, 3, 5, 537067524, 1, 8, 119, 537067520, 0, 1, 1, 0, 1, 3, 0, 596, 0, 0, 0, 0, 0, 595, 0, 0, 595, 0, 0, 100, 1, 0, 2, 0, 1, 2, 1, 1, 2, 537067520, 4, 5, -1, 1, 0, 0, 0, 1, 1, 0, 0, 1, 0, 0, 1178817536, 1, 7, -1, 1, 0, 0, 1, 1, 1, 7, -1, 5, 1, 0, 0, 0, 0, 1, -1, -1, 5, 0, 1, 0, 1, 7, 1, 1, 536874917, 1, 3, 119, 537067528, 1, 1, 1, 1, 0, 12, 595, 0, 0, 0, 595, 0, 0, 0, 596, 595, 596, 0, 100, 50, 0, 1, 2, 1, 1, 1, 0, 9, 3, 0, 3, 0, 1, -1, 0, 0, 0, 0, 0, 1, 1, 1, 0, 1178817536, 1291847194, 0, 7, -1, 1, -1, 1, 1, 0, 0, 2, 0, 5, -1, 1, 0, 1, 0, 7, 0, -1, 1, 0, 0, 0, 1, 9, 119, 1, 537067532, 1, 0, 0, 537067532, 1, 0, 0, 0, 1, 0, 0, 596, 0, 0, 0, 0, 0, 595, 595, 596, 596, 595, 50, 100, 1, 0, 1, 0, 0, 0, 2, 0, 0, 1, 3, 5, 0, 0, 3, 0, -1, 0, -1, 5, 5, 5, 5, 0, 1, 0, 1, 2, 2, 4, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 65536, 0, 1, 0, 0, 3, 1, 0, 1, 2, 0, 2, 2, 3, 0, -998637568, 1315859240, 1184645120}
