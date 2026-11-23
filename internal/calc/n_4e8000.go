package calc

func init() {
	Nodes["004e8000"] = Node{Addr: "004e8000", In: append(append([]string{}, cgInNames...), "a_w", "a_action", "a_p3", "a_p4", "a_p5"),
		Out: cgOutNames, F: actionPhaseAdjustNode}
}

func actionPhaseAdjustNode(in []int64, r Rand) []int64 {
	return cgRun(in, 5, 0, func(w *cgW, a []int64) int64 { w.setAction(a[0], a[1], a[2], a[3], a[4]); return 0 })
}

// setAction is 0x4e8000(w, action, id, ptr4, flag5).
func (w *cgW) setAction(x, action, id, ptr4, flag5 int64) {
	S, D, C := w.S(x), w.D(x), w.C(x)
	old := w.i32(S + 0x3d)
	if !w.selFlags() {
		w.p32(S+0x3d, -1)
		w.record(x, cgRec{0, -1, -1, -1}, 6)
	}
	if action >= 12 {
		panic("cg: 0x4e8000 action >= 12")
	}
	w.p32(S+0x3d, action)
	w.p32(S+0x41, id)
	if ptr4 != 0 {
		w.p32(S+0x45, w.i32(ptr4))
		w.p32(S+0x49, w.i32(ptr4+4))
	} else {
		w.p32(S+0x45, 0)
		w.p32(S+0x49, 0)
	}
	w.p8(D+0xbf, 1)
	if w.i32(C+0xb11) != 0 && w.u32(C+0xb01) < 0xd && action != -1 {
		if w.st(cgS4e79a0, x, 1, 1, flag5)&0xff != 0 {
			if w.st(cgS536f60, x, 2)&0xff == 0 && w.target(x, 1, flag5)&0xff == 1 {
				w.guard(x, S+0x4d)
			}
			if flag5&0xff != 0 {
				w.cursor(x)
			}
		} else {
			w.record(x, cgRec{0, -1, -1, -1}, 1)
		}
	}
	if !w.f94() {
		return
	}
	cb := w.u32(0x6836a8)
	if action != old && w.u8(cb) != 0 && w.u8(D+0x4c) == 0 {
		v := w.u32(cb + 8)
		w.p32(D+0x18, w.u32(D+0x18)+v-w.u32(D+0xb4))
		if p := w.u32(D + 0x18); p > 100 { // 0x5179f0 clamp to [v,100]
			w.p32(D+0x18, 100)
		} else if p < v {
			w.p32(D+0x18, v)
		}
		w.st(cgS4f0480, D+0x18, v)
		w.p32(D+0xb4, v)
	}
	w.ui(1 << uint(x))
	w.p8(w.uip()+0x154, 0)
	if action == 9 || old == 9 && !(w.i32(w.u32(0x6836a8)+0x7b4) == x && w.i32(S+1) == 9) {
		w.notify(x)
	}
	w.st(cgS4ee9d0, w.u32(0x685174)+0x1862*x)
}

// record is 0x538620(x, kind, m): store {kind,-1,-1,-1,0,0,0,0} with 0x5387f0.
func (w *cgW) record(x int64, r cgRec, m int64) { w.store(x, r, m) }

// store is 0x5387f0(x, rec, m): set source m's action record.
// An old record of kind 4 with field 3 != -1 would call 0x538c00 (-> 0x4c5eb0 at 0x538cd3); under the fixture's empty group tables that call returns -1 before 0x4c5eb0, so it is not modelled.
func (w *cgW) store(x int64, rec cgRec, m int64) {
	cur := w.rptr(x, m)
	k := w.kind(x, m)
	bl := m != 0
	if w.rec(cur) != rec {
		if bl {
			if pc := w.i32(w.pcAddr(x)); pc > 0 {
				w.pendingEach(x, 0)
				w.p32(w.pcAddr(x), 0)
			} else {
				c := w.rec(cur)
				if c[0] == 3 && c[2] != -1 {
					w.setMask(x, c[2], 0)
				}
			}
		}
		w.putRec(cur, rec)
		if bl {
			if w.st(cgS4e79a0, x, k, 0, 0)&0xff != 0 && w.kind(x, w.resolve(x)) == k {
				w.release(x, rec)
			}
			w.refresh(x)
			if x == w.sel() {
				w.ui(0x20000)
				w.ui(0x200000)
			}
			w.p8(w.uip()+0x154, 0)
		}
		if k == 1 && w.f94() && rec[0] != 0 {
			a, _, _, _ := w.act(x, 1)
			S := w.S(x)
			switch a {
			case 7:
				w.putRec(S+0x81, rec)
			case 8:
				w.putRec(S+0xa9, rec)
			case 2:
				w.putRec(S+0xd1, rec)
			}
		}
	}
	if rec[0] != 0 && bl && x == w.sel() {
		if w.st(cgS4e79a0, x, 6, 1, 0)&0xff != 0 {
			v := w.rptr(x, 6)
			s := int64(0)
			if w.kind(x, w.resolve(x)) != 0 {
				a, b, _, d := w.act(x, 6)
				s = w.st(cgS536a20, a, b, d)
			}
			if !w.kindOK(v, s) {
				w.phase(s)
				return
			}
		}
		w.phase(0)
	}
}

// release is 0x53a930(x, rec): hand the old selection's monster masks back.
func (w *cgW) release(x int64, rec cgRec) {
	w.p32(w.pcAddr(x), 0)
	a, b, _, d := w.act(x, 6)
	id := int64(0)
	switch a {
	case 2:
		id = 0x77
	case 7:
		id = b
	case 8:
		id = w.itemSpell(w.u32(d + 4))
	}
	if id == 0 {
		if rec[0] == 3 && rec[2] != -1 && w.find(rec[2], false) != -1 {
			w.setMask(x, rec[2], 1) // 0x53ab6b..0x53ab8c is 0x539630 inlined
		}
		return
	}
	w.st(cgS4fd030, id, 1, 0, x, 0x683790+0x118*x, 0, 0)
	for i := int64(0); i < w.monCount(); i++ {
		e := w.entry(i)
		if p := w.u32(e + 0xc); w.u8(e+0x14) != 0 && p != 0 {
			a := w.u32(p+0x1d8) + 0x5bc
			if m := w.u8(a); m&(1<<uint(x)) != 0 {
				w.p8(a, m&^(1<<uint(x)))
				w.st(cgS4c5eb0, x, w.i32(e), 0) // 0x53aaa9
			}
		}
	}
	if w.i32(w.pcAddr(x)) > 0 {
		w.pendingEach(x, 1)
	}
}

// itemSpell is 0x520880: the spell byte of the item behind pointer p.
func (w *cgW) itemSpell(p int64) int64 {
	if p == 0 {
		return 0
	}
	i := w.i32(p)
	if i == -1 || i >= w.i32(0x683f78) {
		panic("cg: 0x520880 assert")
	}
	return w.u8(w.u32(0x68516c) + i*0x10d + 0x63)
}

// cursor is 0x537540(x): refresh the phase from the selected action.
func (w *cgW) cursor(x int64) {
	if x != w.sel() {
		return
	}
	if w.st(cgS4e79a0, x, 6, 1, 0)&0xff == 0 {
		w.phase(0)
		return
	}
	v := w.rptr(x, 6)
	s := int64(0)
	if w.kind(x, w.resolve(x)) != 0 {
		a, b, _, d := w.act(x, 6)
		switch a {
		case 0, 1:
			s = 2
		case 2:
			s = 4
		case 5:
			s = 1
		case 7:
			s = w.spellClass(b, 0, 6)
		case 8:
			s = w.st(cgS537330, w.i32(d+4))
		}
	}
	if w.kindOK(v, s) {
		s = 0
	}
	w.phase(s)
}

// guard is 0x4ecc80(x, R): the monster-targeting side effects of a new action.
func (w *cgW) guard(x, R int64) {
	p := w.i32(0x6850e5)
	if p == 2 || w.st(cgS536190, R)&0xff == 0 {
		return
	}
	if p == 0 && !(x == w.sel() || w.i32(R) == 1 && w.i32(R+4) == w.sel()) {
		return
	}
	switch w.i32(R) {
	case 4:
		panic("cg: 0x4ecc80 reaches 0x5101b0 assert")
	case 3:
		w.find(w.i32(R+8), true)
	}
}

// release4e82e0 is 0x4e82e0(x): re-apply the stored action of slot x.
func (w *cgW) reapply(x int64) {
	S := w.S(x)
	if w.resolve(x) != 2 {
		w.clearMasks(x, w.rec(S+0x4d))
	}
	w.putRec(S+0x4d, cgRec{0, -1, -1, -1})
	w.refresh(x)
	if x == w.sel() {
		w.ui(0x20000)
	}
	w.setAction(x, w.i32(S+0x6d), w.i32(S+0x71), 0, 0)
}

// clearMasks is 0x53ac30(x, rec).
func (w *cgW) clearMasks(x int64, rec cgRec) {
	if pc := w.i32(w.pcAddr(x)); pc > 0 {
		w.pendingEach(x, 0)
		w.p32(w.pcAddr(x), 0)
		return
	}
	if rec[0] == 3 && rec[2] != -1 {
		w.setMask(x, rec[2], 0)
	}
}
