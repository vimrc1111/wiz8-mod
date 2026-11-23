package calc

// ---- 0x531ce0: the monster's action candidate list builder ----
//
// Builds the candidate list at combat+0x16: the intercept scan (a=b=0), the
// per-character swing rolls and the natural-skill growth pass. It runs over the
// same memory image as the emulator case (mstWorld), so every pointer the
// original follows is followed here. The address-stubbed D4 callees (0x532360,
// 0x545cf0, 0x536190, 0x546f10, 0x519c00, 0x51aec0, 0x553f10) and the
// registered-out log (0x5182e0) read their return from case inputs and record
// every call; the Emu run replays them, the F run calls the real Go callees.

const (
	macRoll  = iota // 0x532360
	macIcpt         // 0x545cf0
	macValid        // 0x536190
	macDisp         // 0x546f10
	macAvail        // 0x519c00
	macPen          // 0x51aec0
	macGrow         // 0x553f10
	macLog          // 0x5182e0
	macStubs
)

// macStub places one stub: the return word (retN > 0: call k reads ret+4k), the
// recorded stack arguments {arg, size, addr, cap} (call k lands at addr+k*size)
// and the call counter.
type macStub struct {
	ret, retN uint32
	rec       [][4]uint32
	cnt       uint32
}

type macSpec struct {
	mstSpec
	stub [macStubs]macStub
	// The 0x532360 stub overwrites the new list's count word (0x50000008) with
	// the case input at rollW: off is that address minus the monster pointer.
	rollOff, rollW uint32
	// Start of the F-only parked callee vectors in the input list.
	parkIcpt, parkValid, parkAvail, parkPen int
}

type macCallees struct {
	roll  func(w *macWorld, m, a, b, c, d, e uint32)
	icpt  func(w *macWorld, m, tgt uint32) bool
	valid func(w *macWorld, tgt uint32) bool
	disp  func(w *macWorld, ch, m uint32) byte
	avail func(w *macWorld, m, k, f uint32) bool
	pen   func(w *macWorld, ch, m uint32) byte
	grow  func(w *macWorld, ch uint32)
}

type macWorld struct {
	*mstWorld
	sp *macSpec
	c  *macCallees
	in []int64
}

// call records one stubbed call (arguments are 1-based stack positions) and
// returns its sequence number; more calls than the case records panic.
func (w *macWorld) call(k int, args ...uint32) uint32 {
	s := &w.sp.stub[k]
	n := w.u32(s.cnt)
	for _, r := range s.rec {
		if n >= r[3] {
			panic("monster action candidates: more stubbed calls than the case records")
		}
		w.put(r[2]+n*r[1], int(r[1]), uint64(args[r[0]-1]))
	}
	w.put(s.cnt, 4, uint64(n+1))
	return n
}

// ret is the stub's return word for call n.
func (w *macWorld) ret(k int, n uint32) uint32 {
	s := &w.sp.stub[k]
	if s.retN == 0 {
		return w.u32(s.ret)
	}
	if n >= s.retN {
		panic("monster action candidates: more stubbed calls than the case returns")
	}
	return w.u32(s.ret + 4*n)
}

func (w *macWorld) parked(at, n int) []int64 { return append([]int64(nil), w.in[at:at+n]...) }

func (w *macWorld) u16(a uint32) uint32 { return uint32(w.u8(a)) | uint32(w.u8(a+1))<<8 }

var macEmu = macCallees{
	roll: func(w *macWorld, m, a, b, c, d, e uint32) {
		w.call(macRoll, m, a, b, c, d, e)
		w.put(m+w.sp.rollOff, 4, uint64(w.u32(w.sp.rollW)))
	},
	icpt: func(w *macWorld, m, tgt uint32) bool {
		return byte(w.ret(macIcpt, w.call(macIcpt, m, tgt))) != 0
	},
	valid: func(w *macWorld, tgt uint32) bool {
		return byte(w.ret(macValid, w.call(macValid, tgt))) != 0
	},
	disp: func(w *macWorld, ch, m uint32) byte {
		return byte(w.ret(macDisp, w.call(macDisp, ch, m)))
	},
	avail: func(w *macWorld, m, k, f uint32) bool {
		return byte(w.ret(macAvail, w.call(macAvail, m, k, f))) != 0
	},
	pen: func(w *macWorld, ch, m uint32) byte {
		return byte(w.ret(macPen, w.call(macPen, ch, m)))
	},
	grow: func(w *macWorld, ch uint32) { w.call(macGrow, ch) },
}

// macReal calls the registered Go nodes. The callees whose inputs are whole
// worlds of their own (0x545cf0, 0x536190, 0x519c00, 0x51aec0) take the parked
// vectors; 0x546f10 and 0x532360 take the values the original hands them.
var macReal = macCallees{
	roll: func(w *macWorld, m, a, b, c, d, e uint32) {
		w.call(macRoll, m, a, b, c, d, e)
		// A fresh empty list per call: the real header's capacity (10) is past the
		// 0x532360 node's fixture; the node draws the one R(100) of the original.
		monsterActionRollNode([]int64{int64(int32(a)), int64(int32(b)), int64(int32(c)), int64(int32(d)), int64(int32(e)), 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, w.r)
		l := w.u32(m+0x10) + 0x16
		h := w.u32(l)
		w.put(h+8, 4, uint64(w.u32(h+8)+1))
	},
	icpt: func(w *macWorld, m, tgt uint32) bool {
		w.call(macIcpt, m, tgt)
		return byte(monsterInterceptAvailableNode(w.parked(w.sp.parkIcpt, 48), w.r)[0]) != 0
	},
	valid: func(w *macWorld, tgt uint32) bool {
		w.call(macValid, tgt)
		return byte(combatTargetValidNode(w.parked(w.sp.parkValid, 29), w.r)[0]) != 0
	},
	disp: func(w *macWorld, ch, m uint32) byte {
		w.call(macDisp, ch, m)
		return byte(targetDispositionNode([]int64{int64(ch), int64(int32(w.u32(w.charBase(ch) + 0xa35))), int64(w.u8(m + 0x16))}, w.r)[0])
	},
	avail: func(w *macWorld, m, k, f uint32) bool {
		w.call(macAvail, m, k, f)
		return byte(monsterAttackTargetAvailableNode(w.parked(w.sp.parkAvail, 167), w.r)[0]) != 0
	},
	pen: func(w *macWorld, ch, m uint32) byte {
		w.call(macPen, ch, m)
		x := w.parked(w.sp.parkPen, 39)
		x[0] = int64(ch)
		return byte(formationRangePenaltyNode(x, w.r)[0])
	},
	// 0x553f10's results are not read by this node: replayed, not parked.
	grow: func(w *macWorld, ch uint32) { w.call(macGrow, ch) },
}

// side is 0x519be0 (via 0x505e60): the second flag slot for reach classes 2 and 3.
func (w *macWorld) side(m, cls uint32) uint32 {
	if cls >= 2 && cls <= 3 && w.u8(m+0x37a) != 0 {
		return 1
	}
	return 0
}

// reachChar is 0x51a2f0 for a monster whose reach state is 0 (class from the
// template's action entry k).
func (w *macWorld) reachChar(m, t, k, ch uint32) bool {
	if w.u8(m+0x34c) != 1 {
		return false
	}
	cls := uint32(w.u8(t + 0xea + k*0x22))
	if w.u8(m+0x34d+w.side(m, cls)) == 0 {
		return false
	}
	if w.u8(0x683f94) != 0 && cls <= 1 {
		for p := w.c.pen(w, ch, m); p != 0; p-- {
			if cls == 0 {
				return false
			}
			cls--
		}
	}
	return monsterAttackTargetWithin(w.mstLen(0x51a4fc, mstLenChar, cls+1), w.mstDistParty(w.u32(m+0xc)))
}

// reachMon is 0x51a510 for reach state 0.
func (w *macWorld) reachMon(m, t, k, o uint32) bool {
	if w.u8(t+0x24a) != 0 {
		return false
	}
	if o == m {
		return true
	}
	cls := uint32(w.u8(t + 0xea + k*0x22))
	e := w.mstRelation(m, o)
	if e == 0 || w.u8(e+4) != 1 || w.u8(e+w.side(m, cls)+5) == 0 {
		return false
	}
	return monsterAttackTargetWithin(w.mstLen(0x51a714, mstLenMon, cls+1), w.mstDistMon(w.u32(m+0xc), w.u32(o+0xc)))
}

// blocked is 0x53d4b0: nonzero when action entry k is skipped.
func (w *macWorld) blocked(m, t, k, flag uint32) bool {
	if k >= 3 {
		panic("monster action candidates: action entry past 2 (assert)")
	}
	if w.u8(t+0xe7+k*0x22) == 0 {
		return true
	}
	if w.u16(t+0xfc+k*0x22) == 0 {
		w.call(macLog, 0, 0x61d328, t, k)
		return true
	}
	if w.u8(t+0xe5) == 0 {
		w.call(macLog, 0, 0x61d304, t, 0)
		return true
	}
	if w.u8(t+0xe6) == 0 {
		w.call(macLog, 0, 0x61d2e0, t, 0)
		return true
	}
	state := w.u32(m + 0x2e1)
	w.put(m+0x2e1, 4, 0)
	ok := w.c.avail(w, m, k, flag)
	w.put(m+0x2e1, 4, uint64(state))
	return !ok
}

// freshTarget is 0x536170: eight zero words, the three ids absent.
func (w *macWorld) freshTarget(p uint32) {
	for i := uint32(0); i < 8; i++ {
		w.put(p+4*i, 4, 0)
	}
	for i := uint32(1); i < 4; i++ {
		w.put(p+4*i, 4, 0xffffffff)
	}
}

// intercepts is 0x545bd0.
func (w *macWorld) intercepts(m, t uint32) bool {
	return w.u8(m+0x14) != 0 && w.u8(m+0x15) != 0 && w.u32(m+0x2b) != 0 && w.u32(m+0x107) < 0xc &&
		w.u8(t+0xd0)&4 != 0 && w.u8(t+0xe7) != 0
}

func (w *macWorld) run(m uint32, a, b byte) {
	t := w.template(m)
	cmb := w.u32(m + 0x10)
	// 0x5e23e0 releases the old list's data block; 0x5e22c0 builds the new one
	// (12-byte header, ten 4-byte slots).
	if l := w.u32(cmb + 0x16); l != 0 && w.u32(l) != 0 {
		w.put(l, 4, 0)
	}
	hdr := w.alloc(12)
	data := w.alloc(0x28)
	w.put(hdr, 4, uint64(data))
	w.put(hdr+4, 4, 10)
	w.put(cmb+0x16, 4, uint64(hdr))
	if hdr == 0 {
		panic("monster action candidates: list allocation returned null (assert)")
	}
	if hdr != mstHeap {
		panic("monster action candidates: the list header is not the first allocation")
	}
	roll := func(x ...uint32) { w.c.roll(w, m, x[0], x[1], x[2], x[3], x[4]) }
	tg := m + 0x2ba
	if a == 0 && b == 0 {
		if w.intercepts(m, t) && w.r.R(100) < int(w.u8(m+0x248)) {
			w.put(m+0x2e1, 4, 8)
			n := w.listCount(w.u32(0x683fad))
			for j := uint32(0); j < n; j++ {
				o := w.monsterRecord(j)
				if o != m {
					w.freshTarget(tg)
					w.put(tg, 4, 3)
					w.put(m+0x2c2, 4, uint64(w.u32(o)))
					if w.c.icpt(w, m, tg) {
						roll(8, 0, 0, 3, w.u32(o))
					}
				}
			}
			for i := uint32(0); i < 8; i++ {
				w.freshTarget(tg)
				w.put(tg, 4, 1)
				w.put(m+0x2be, 4, uint64(i))
				if w.c.icpt(w, m, tg) {
					roll(8, 0, 0, 1, i)
				}
			}
			if int32(w.listCount(w.u32(cmb+0x16))) > 0 {
				return
			}
		}
		if w.u8(t+0x1c0) == 2 && w.u8(t+0x1bf) != 3 && w.r.R(100) < int(w.u8(m+0x248)) {
			roll(1, 0xffffffff, 0, 0, 0)
		}
	}
	w.put(m+0x2e1, 4, 0)
	kFrom, kTo := uint32(0), uint32(3)
	if b != 0 {
		kFrom = w.u32(cmb + 0x11)
		kTo = kFrom + 1
	}
	var flags, pick [mstMaxChar]bool
	var charsOn, monsOn bool
	var cFrom, cTo, mFrom, mTo uint32
	scan := func() { // the per-character swing chance rolls
		for c := cFrom; c < cTo; c++ {
			r := uint32(w.r.R(100))
			if r < w.u32(w.charBase(c)+0x345)*75/100 {
				flags[c] = true
			}
		}
	}
	if a == 0 || b != 0 || w.u8(cmb+0x15) != 0 || w.u8(m+0x249) >= 0x4b {
		charsOn, monsOn = true, true
		cFrom, cTo, mFrom, mTo = 0, mstMaxChar, 0, w.listCount(w.u32(0x683fad))
		scan()
	} else {
		if !w.c.valid(w, tg) {
			return
		}
		switch w.u32(tg) {
		case 1:
			id := w.u32(tg + 4)
			if id == 0xffffffff {
				panic("monster action candidates: character target id -1 (assert)")
			}
			charsOn, cFrom, cTo = true, id, id+1
			scan()
		case 3:
			id := w.u32(tg + 8)
			if id == 0xffffffff {
				panic("monster action candidates: monster target id -1 (assert)")
			}
			h, ok := w.monsterHandle(id)
			if !ok {
				panic("monster action candidates: monster target not found (assert)")
			}
			monsOn, mFrom, mTo = true, h, h+1
		default:
			return
		}
	}
	flag := uint32(0)
	if w.u8(m+0x15) != 0 && w.u8(cmb+0x15) != 0 {
		flag = 1
	}
	need := byte(flag + 1)
	for k := kFrom; k < kTo; k++ {
		if w.blocked(m, t, k, flag) {
			continue
		}
		mask := w.u16(t + 0xfc + k*0x22)
		cls := uint32(w.u8(t + 0xea + k*0x22))
		if charsOn && w.u8(m+0x34d+w.side(m, cls)) != 0 {
			for c := cFrom; c < cTo; c++ {
				ch := w.charBase(c)
				if w.u8(w.u32(0x685178)+0x106*c) == 0 || w.u32(ch+0xb11) == 0 || w.u32(ch+0xb01) >= 0x12 ||
					w.c.disp(w, c, m) != need || !w.reachChar(m, t, k, c) {
					continue
				}
				if flags[c] {
					pick[c] = true
					continue
				}
				for j := uint32(0); j < 9; j++ {
					if mask&(1<<j) != 0 {
						roll(0, j, k, 1, c)
					}
				}
			}
		}
		if monsOn {
			for i := mFrom; i < mTo; i++ {
				o := w.monsterRecord(i)
				if o == m || w.u8(o+0x14) == 0 || w.u32(o+0x2b) == 0 || w.u8(w.template(o)+0x24a) != 0 || w.u8(o+0x15) == 0 ||
					byte(w.mstStance(m, o)) != need || !w.reachMon(m, t, k, o) {
					continue
				}
				for j := uint32(0); j < 9; j++ {
					if mask&(1<<j) != 0 {
						roll(0, j, k, 3, w.u32(o))
					}
				}
			}
		}
	}
	if charsOn {
		for c := cFrom; c < cTo; c++ {
			ch := w.charBase(c)
			if pick[c] && w.r.R(2) == 0 {
				w.c.grow(w, ch)
			}
		}
	}
}

func macNode(spec *macSpec, c *macCallees) func([]int64, Rand) []int64 {
	return func(in []int64, r Rand) []int64 {
		if len(in) != len(spec.in) {
			panic("monster action candidates: wrong input count")
		}
		w := &macWorld{mstWorld: mstBuild(&spec.mstSpec, in), sp: spec, c: c, in: in}
		w.r = r
		w.run(w.u32(mstStackBase+4), w.u8(mstStackBase+8), w.u8(mstStackBase+12))
		out := make([]int64, len(spec.out))
		for i, f := range spec.out {
			out[i] = w.read(f, in)
		}
		return out
	}
}

func monsterActionCandidatesNode(in []int64, r Rand) []int64 {
	return macNode(&macSpec_main, &macReal)(in, r)
}

func init() {
	Nodes["00531ce0"] = Node{Addr: "00531ce0", In: macNames_in(), Out: macNames_out(),
		F: monsterActionCandidatesNode, Emu: macNode(&macSpec_main, &macEmu)}
}

var macSpec_main = macSpec{
	mstSpec: mstSpec{
		set: []mstSet{
			{0x7f000004, 4, 0x200023a0},
			{0x685174, 4, 0x20005830},
			{0x685178, 4, 0x20004ff0},
			{0x683fad, 4, 0x20002360},
			{0x683fb9, 4, 0x20002380},
			{0x6840c7, 4, 0x200047f0},
			{0x6840cb, 4, 0x20004a70},
			{0x65a0f8, 4, 0x20004cf0},
			{0x659c0c, 4, 0x20004df0},
			{0x20002360, 4, 0x20002370},
			{0x20002380, 4, 0x20002390},
			{0x200047a0, 4, 0x200047b0},
			{0x200047b0, 4, 0x200047c0},
			{0x200047b4, 4, 0x200047d0},
			{0x200047b8, 4, 0x200047e0},
			{0x20002656, 4, 0x200047a0},
			{0x200023ac, 4, 0x20003ba0},
			{0x200023b0, 4, 0x20002300},
			{0x200027ac, 4, 0x20003da0},
			{0x200027b0, 4, 0x20002300},
			{0x20002bac, 4, 0x20003fa0},
			{0x20002bb0, 4, 0x20002300},
			{0x20002fac, 4, 0x200041a0},
			{0x20002fb0, 4, 0x20002300},
			{0x200033ac, 4, 0x200043a0},
			{0x200033b0, 4, 0x20002300},
			{0x200037ac, 4, 0x200045a0},
			{0x200037b0, 4, 0x20002300},
		},
		in: []mstFld{
			{addr: 0x7f000008, size: 1, signed: false, idx: -1},
			{addr: 0x7f00000c, size: 1, signed: false, idx: -1},
			{addr: 0x683f94, size: 1, signed: false, idx: -1},
			{addr: 0x20004d7c, size: 4, signed: false, idx: -1},
			{addr: 0x20004d80, size: 4, signed: false, idx: -1},
			{addr: 0x20004d84, size: 4, signed: false, idx: -1},
			{addr: 0x20004f64, size: 4, signed: false, idx: -1},
			{addr: 0x20002311, size: 4, signed: false, idx: -1},
			{addr: 0x20002315, size: 1, signed: false, idx: -1},
			{addr: 0x20002316, size: 4, signed: false, idx: -1},
			{addr: 0x20002340, size: 4, signed: false, idx: -1},
			{addr: 0x20002368, size: 4, signed: false, idx: -1},
			{addr: 0x20002370, size: 4, signed: false, idx: -1},
			{addr: 0x20002374, size: 4, signed: false, idx: -1},
			{addr: 0x20002378, size: 4, signed: false, idx: -1},
			{addr: 0x2000237c, size: 4, signed: false, idx: -1},
			{addr: 0x20002388, size: 4, signed: false, idx: -1},
			{addr: 0x20002390, size: 4, signed: false, idx: -1},
			{addr: 0x20002394, size: 4, signed: false, idx: -1},
			{addr: 0x200023a0, size: 4, signed: false, idx: -1},
			{addr: 0x200023a8, size: 4, signed: false, idx: -1},
			{addr: 0x200023b4, size: 1, signed: false, idx: -1},
			{addr: 0x200023b5, size: 1, signed: false, idx: -1},
			{addr: 0x200023b6, size: 1, signed: false, idx: -1},
			{addr: 0x200023cb, size: 4, signed: false, idx: -1},
			{addr: 0x2000242b, size: 1, signed: false, idx: -1},
			{addr: 0x20003cb8, size: 4, signed: false, idx: -1},
			{addr: 0x20003cbc, size: 4, signed: false, idx: -1},
			{addr: 0x20003cc0, size: 4, signed: false, idx: -1},
			{addr: 0x20003d2c, size: 4, signed: false, idx: -1},
			{addr: 0x200027a0, size: 4, signed: false, idx: -1},
			{addr: 0x200027a8, size: 4, signed: false, idx: -1},
			{addr: 0x200027b4, size: 1, signed: false, idx: -1},
			{addr: 0x200027b5, size: 1, signed: false, idx: -1},
			{addr: 0x200027b6, size: 1, signed: false, idx: -1},
			{addr: 0x200027cb, size: 4, signed: false, idx: -1},
			{addr: 0x2000282b, size: 1, signed: false, idx: -1},
			{addr: 0x20003eb8, size: 4, signed: false, idx: -1},
			{addr: 0x20003ebc, size: 4, signed: false, idx: -1},
			{addr: 0x20003ec0, size: 4, signed: false, idx: -1},
			{addr: 0x20003f2c, size: 4, signed: false, idx: -1},
			{addr: 0x20002ba0, size: 4, signed: false, idx: -1},
			{addr: 0x20002ba8, size: 4, signed: false, idx: -1},
			{addr: 0x20002bb4, size: 1, signed: false, idx: -1},
			{addr: 0x20002bb5, size: 1, signed: false, idx: -1},
			{addr: 0x20002bb6, size: 1, signed: false, idx: -1},
			{addr: 0x20002bcb, size: 4, signed: false, idx: -1},
			{addr: 0x20002c2b, size: 1, signed: false, idx: -1},
			{addr: 0x200040b8, size: 4, signed: false, idx: -1},
			{addr: 0x200040bc, size: 4, signed: false, idx: -1},
			{addr: 0x200040c0, size: 4, signed: false, idx: -1},
			{addr: 0x2000412c, size: 4, signed: false, idx: -1},
			{addr: 0x20002fa0, size: 4, signed: false, idx: -1},
			{addr: 0x20002fa8, size: 4, signed: false, idx: -1},
			{addr: 0x20002fb4, size: 1, signed: false, idx: -1},
			{addr: 0x20002fb5, size: 1, signed: false, idx: -1},
			{addr: 0x20002fb6, size: 1, signed: false, idx: -1},
			{addr: 0x20002fcb, size: 4, signed: false, idx: -1},
			{addr: 0x2000302b, size: 1, signed: false, idx: -1},
			{addr: 0x200042b8, size: 4, signed: false, idx: -1},
			{addr: 0x200042bc, size: 4, signed: false, idx: -1},
			{addr: 0x200042c0, size: 4, signed: false, idx: -1},
			{addr: 0x2000432c, size: 4, signed: false, idx: -1},
			{addr: 0x200033a0, size: 4, signed: false, idx: -1},
			{addr: 0x200033a8, size: 4, signed: false, idx: -1},
			{addr: 0x200033b4, size: 1, signed: false, idx: -1},
			{addr: 0x200033b5, size: 1, signed: false, idx: -1},
			{addr: 0x200033b6, size: 1, signed: false, idx: -1},
			{addr: 0x200033cb, size: 4, signed: false, idx: -1},
			{addr: 0x2000342b, size: 1, signed: false, idx: -1},
			{addr: 0x200044b8, size: 4, signed: false, idx: -1},
			{addr: 0x200044bc, size: 4, signed: false, idx: -1},
			{addr: 0x200044c0, size: 4, signed: false, idx: -1},
			{addr: 0x2000452c, size: 4, signed: false, idx: -1},
			{addr: 0x200037a0, size: 4, signed: false, idx: -1},
			{addr: 0x200037a8, size: 4, signed: false, idx: -1},
			{addr: 0x200037b4, size: 1, signed: false, idx: -1},
			{addr: 0x200037b5, size: 1, signed: false, idx: -1},
			{addr: 0x200037b6, size: 1, signed: false, idx: -1},
			{addr: 0x200037cb, size: 4, signed: false, idx: -1},
			{addr: 0x2000382b, size: 1, signed: false, idx: -1},
			{addr: 0x200046b8, size: 4, signed: false, idx: -1},
			{addr: 0x200046bc, size: 4, signed: false, idx: -1},
			{addr: 0x200046c0, size: 4, signed: false, idx: -1},
			{addr: 0x2000472c, size: 4, signed: false, idx: -1},
			{addr: 0x200024a7, size: 4, signed: false, idx: -1},
			{addr: 0x200025e8, size: 1, signed: false, idx: -1},
			{addr: 0x200025e9, size: 1, signed: false, idx: -1},
			{addr: 0x200026ec, size: 1, signed: false, idx: -1},
			{addr: 0x200026ed, size: 1, signed: false, idx: -1},
			{addr: 0x200026ee, size: 1, signed: false, idx: -1},
			{addr: 0x2000271a, size: 1, signed: false, idx: -1},
			{addr: 0x2000265a, size: 4, signed: false, idx: -1},
			{addr: 0x2000265e, size: 4, signed: false, idx: -1},
			{addr: 0x20002662, size: 4, signed: false, idx: -1},
			{addr: 0x20002666, size: 4, signed: false, idx: -1},
			{addr: 0x2000266a, size: 4, signed: false, idx: -1},
			{addr: 0x2000266e, size: 4, signed: false, idx: -1},
			{addr: 0x20002672, size: 4, signed: false, idx: -1},
			{addr: 0x20002676, size: 4, signed: false, idx: -1},
			{addr: 0x200047a8, size: 4, signed: false, idx: -1},
			{addr: 0x200047c0, size: 4, signed: false, idx: -1},
			{addr: 0x200047c4, size: 1, signed: false, idx: -1},
			{addr: 0x200047c5, size: 1, signed: false, idx: -1},
			{addr: 0x200047c6, size: 1, signed: false, idx: -1},
			{addr: 0x200047d0, size: 4, signed: false, idx: -1},
			{addr: 0x200047d4, size: 1, signed: false, idx: -1},
			{addr: 0x200047d5, size: 1, signed: false, idx: -1},
			{addr: 0x200047d6, size: 1, signed: false, idx: -1},
			{addr: 0x200047e0, size: 4, signed: false, idx: -1},
			{addr: 0x200047e4, size: 1, signed: false, idx: -1},
			{addr: 0x200047e5, size: 1, signed: false, idx: -1},
			{addr: 0x200047e6, size: 1, signed: false, idx: -1},
			{addr: 0x200048c0, size: 1, signed: false, idx: -1},
			{addr: 0x200048d5, size: 1, signed: false, idx: -1},
			{addr: 0x200048d6, size: 1, signed: false, idx: -1},
			{addr: 0x200049af, size: 1, signed: false, idx: -1},
			{addr: 0x200049b0, size: 1, signed: false, idx: -1},
			{addr: 0x20004a3a, size: 1, signed: false, idx: -1},
			{addr: 0x20004a4f, size: 4, signed: false, idx: -1},
			{addr: 0x200048d7, size: 1, signed: false, idx: -1},
			{addr: 0x200048da, size: 1, signed: false, idx: -1},
			{addr: 0x200048ec, size: 2, signed: false, idx: -1},
			{addr: 0x200048f9, size: 1, signed: false, idx: -1},
			{addr: 0x200048fc, size: 1, signed: false, idx: -1},
			{addr: 0x2000490e, size: 2, signed: false, idx: -1},
			{addr: 0x2000491b, size: 1, signed: false, idx: -1},
			{addr: 0x2000491e, size: 1, signed: false, idx: -1},
			{addr: 0x20004930, size: 2, signed: false, idx: -1},
			{addr: 0x20004b40, size: 1, signed: false, idx: -1},
			{addr: 0x20004b55, size: 1, signed: false, idx: -1},
			{addr: 0x20004b56, size: 1, signed: false, idx: -1},
			{addr: 0x20004c2f, size: 1, signed: false, idx: -1},
			{addr: 0x20004c30, size: 1, signed: false, idx: -1},
			{addr: 0x20004cba, size: 1, signed: false, idx: -1},
			{addr: 0x20004ccf, size: 4, signed: false, idx: -1},
			{addr: 0x20004b57, size: 1, signed: false, idx: -1},
			{addr: 0x20004b5a, size: 1, signed: false, idx: -1},
			{addr: 0x20004b6c, size: 2, signed: false, idx: -1},
			{addr: 0x20004b79, size: 1, signed: false, idx: -1},
			{addr: 0x20004b7c, size: 1, signed: false, idx: -1},
			{addr: 0x20004b8e, size: 2, signed: false, idx: -1},
			{addr: 0x20004b9b, size: 1, signed: false, idx: -1},
			{addr: 0x20004b9e, size: 1, signed: false, idx: -1},
			{addr: 0x20004bb0, size: 2, signed: false, idx: -1},
			{addr: 0x20004ff0, size: 1, signed: false, idx: -1},
			{addr: 0x20006341, size: 4, signed: false, idx: -1},
			{addr: 0x20006331, size: 4, signed: false, idx: -1},
			{addr: 0x20005b75, size: 4, signed: false, idx: -1},
			{addr: 0x20006265, size: 4, signed: true, idx: -1},
			{addr: 0x200050f6, size: 1, signed: false, idx: -1},
			{addr: 0x20007ba3, size: 4, signed: false, idx: -1},
			{addr: 0x20007b93, size: 4, signed: false, idx: -1},
			{addr: 0x200073d7, size: 4, signed: false, idx: -1},
			{addr: 0x20007ac7, size: 4, signed: true, idx: -1},
			{addr: 0x200051fc, size: 1, signed: false, idx: -1},
			{addr: 0x20009405, size: 4, signed: false, idx: -1},
			{addr: 0x200093f5, size: 4, signed: false, idx: -1},
			{addr: 0x20008c39, size: 4, signed: false, idx: -1},
			{addr: 0x20009329, size: 4, signed: true, idx: -1},
			{addr: 0x20005302, size: 1, signed: false, idx: -1},
			{addr: 0x2000ac67, size: 4, signed: false, idx: -1},
			{addr: 0x2000ac57, size: 4, signed: false, idx: -1},
			{addr: 0x2000a49b, size: 4, signed: false, idx: -1},
			{addr: 0x2000ab8b, size: 4, signed: true, idx: -1},
			{addr: 0x20005408, size: 1, signed: false, idx: -1},
			{addr: 0x2000c4c9, size: 4, signed: false, idx: -1},
			{addr: 0x2000c4b9, size: 4, signed: false, idx: -1},
			{addr: 0x2000bcfd, size: 4, signed: false, idx: -1},
			{addr: 0x2000c3ed, size: 4, signed: true, idx: -1},
			{addr: 0x2000550e, size: 1, signed: false, idx: -1},
			{addr: 0x2000dd2b, size: 4, signed: false, idx: -1},
			{addr: 0x2000dd1b, size: 4, signed: false, idx: -1},
			{addr: 0x2000d55f, size: 4, signed: false, idx: -1},
			{addr: 0x2000dc4f, size: 4, signed: true, idx: -1},
			{addr: 0x20005614, size: 1, signed: false, idx: -1},
			{addr: 0x2000f58d, size: 4, signed: false, idx: -1},
			{addr: 0x2000f57d, size: 4, signed: false, idx: -1},
			{addr: 0x2000edc1, size: 4, signed: false, idx: -1},
			{addr: 0x2000f4b1, size: 4, signed: true, idx: -1},
			{addr: 0x2000571a, size: 1, signed: false, idx: -1},
			{addr: 0x20010def, size: 4, signed: false, idx: -1},
			{addr: 0x20010ddf, size: 4, signed: false, idx: -1},
			{addr: 0x20010623, size: 4, signed: false, idx: -1},
			{addr: 0x20010d13, size: 4, signed: true, idx: -1},
			{addr: 0x20001100, size: 4, signed: false, idx: -1},
			{addr: 0x20001104, size: 4, signed: false, idx: -1},
			{addr: 0x20001108, size: 4, signed: false, idx: -1},
			{addr: 0x2000110c, size: 4, signed: false, idx: -1},
			{addr: 0x20001110, size: 4, signed: false, idx: -1},
			{addr: 0x20001114, size: 4, signed: false, idx: -1},
			{addr: 0x20001118, size: 4, signed: false, idx: -1},
			{addr: 0x2000111c, size: 4, signed: false, idx: -1},
			{addr: 0x20001120, size: 4, signed: false, idx: -1},
			{addr: 0x20001124, size: 4, signed: false, idx: -1},
			{addr: 0x20001128, size: 4, signed: false, idx: -1},
			{addr: 0x2000112c, size: 4, signed: false, idx: -1},
			{addr: 0x20001130, size: 4, signed: false, idx: -1},
			{addr: 0x20001134, size: 4, signed: false, idx: -1},
			{addr: 0x20001138, size: 4, signed: false, idx: -1},
			{addr: 0x2000113c, size: 4, signed: false, idx: -1},
			{addr: 0x20001140, size: 4, signed: false, idx: -1},
			{addr: 0x20001144, size: 4, signed: false, idx: -1},
			{addr: 0x20001148, size: 4, signed: false, idx: -1},
			{addr: 0x2000114c, size: 4, signed: false, idx: -1},
			{addr: 0x20001150, size: 4, signed: false, idx: -1},
			{addr: 0x20001154, size: 4, signed: false, idx: -1},
			{addr: 0x20001158, size: 4, signed: false, idx: -1},
			{addr: 0x2000115c, size: 4, signed: false, idx: -1},
			{addr: 0x20001160, size: 4, signed: false, idx: -1},
			{addr: 0x20001164, size: 4, signed: false, idx: -1},
			{addr: 0x20001168, size: 4, signed: false, idx: -1},
			{addr: 0x2000116c, size: 4, signed: false, idx: -1},
			{addr: 0x20001170, size: 4, signed: false, idx: -1},
			{addr: 0x20001174, size: 4, signed: false, idx: -1},
			{addr: 0x20001178, size: 4, signed: false, idx: -1},
			{addr: 0x2000117c, size: 4, signed: false, idx: -1},
			{addr: 0x20001180, size: 4, signed: false, idx: -1},
			{addr: 0x20001184, size: 4, signed: false, idx: -1},
			{addr: 0x20001188, size: 4, signed: false, idx: -1},
			{addr: 0x2000118c, size: 4, signed: false, idx: -1},
			{addr: 0x20001190, size: 4, signed: false, idx: -1},
			{addr: 0x20001194, size: 4, signed: false, idx: -1},
			{addr: 0x20001198, size: 4, signed: false, idx: -1},
			{addr: 0x2000119c, size: 4, signed: false, idx: -1},
			{addr: 0x200011a0, size: 4, signed: false, idx: -1},
			{addr: 0x200011a4, size: 4, signed: false, idx: -1},
			{addr: 0x200011a8, size: 4, signed: false, idx: -1},
			{addr: 0x200011ac, size: 4, signed: false, idx: -1},
			{addr: 0x200011b0, size: 4, signed: false, idx: -1},
			{addr: 0x200011b4, size: 4, signed: false, idx: -1},
			{addr: 0x200011b8, size: 4, signed: false, idx: -1},
			{addr: 0x200011bc, size: 4, signed: false, idx: -1},
			{addr: 0x200011c0, size: 4, signed: false, idx: -1},
			{addr: 0x200011c4, size: 4, signed: false, idx: -1},
			{addr: 0x200011c8, size: 4, signed: false, idx: -1},
			{addr: 0x200011cc, size: 4, signed: false, idx: -1},
			{addr: 0x200011d0, size: 4, signed: false, idx: -1},
			{addr: 0x200011d4, size: 4, signed: false, idx: -1},
			{addr: 0x200011d8, size: 4, signed: false, idx: -1},
			{addr: 0x200011dc, size: 4, signed: false, idx: -1},
			{addr: 0x200011e0, size: 4, signed: false, idx: -1},
			{addr: 0x200011e4, size: 4, signed: false, idx: -1},
			{addr: 0x200011e8, size: 4, signed: false, idx: -1},
			{addr: 0x200011ec, size: 4, signed: false, idx: -1},
			{addr: 0x200011f0, size: 4, signed: false, idx: -1},
			{addr: 0x200011f4, size: 4, signed: false, idx: -1},
			{addr: 0x200011f8, size: 4, signed: false, idx: -1},
			{addr: 0x200011fc, size: 4, signed: false, idx: -1},
			{addr: 0x20001200, size: 4, signed: false, idx: -1},
			{addr: 0x20001300, size: 4, signed: true, idx: -1},
			{addr: 0x20001308, size: 4, signed: true, idx: -1},
			{addr: 0x20001310, size: 4, signed: true, idx: -1},
			{addr: 0x20001318, size: 4, signed: true, idx: -1},
			{addr: 0x20001320, size: 4, signed: true, idx: -1},
			{addr: 0x20001328, size: 4, signed: true, idx: -1},
			{addr: 0x20001330, size: 4, signed: true, idx: -1},
			{addr: 0x20001338, size: 4, signed: true, idx: -1},
			{addr: 0x20001340, size: 4, signed: true, idx: -1},
			{addr: 0x20001348, size: 4, signed: true, idx: -1},
			{addr: 0x20001350, size: 4, signed: true, idx: -1},
			{addr: 0x20001358, size: 4, signed: true, idx: -1},
			{addr: 0x20001360, size: 4, signed: true, idx: -1},
			{addr: 0x20001368, size: 4, signed: true, idx: -1},
			{addr: 0x20001370, size: 4, signed: true, idx: -1},
			{addr: 0x20001378, size: 4, signed: true, idx: -1},
			{addr: 0x20001380, size: 4, signed: true, idx: -1},
			{addr: 0x20001388, size: 4, signed: true, idx: -1},
			{addr: 0x20001390, size: 4, signed: true, idx: -1},
			{addr: 0x20001398, size: 4, signed: true, idx: -1},
			{addr: 0x200013a0, size: 4, signed: true, idx: -1},
			{addr: 0x200013a8, size: 4, signed: true, idx: -1},
			{addr: 0x200013b0, size: 4, signed: true, idx: -1},
			{addr: 0x200013b8, size: 4, signed: true, idx: -1},
			{addr: 0x200013c0, size: 4, signed: true, idx: -1},
			{addr: 0x200013c8, size: 4, signed: true, idx: -1},
			{addr: 0x200013d0, size: 4, signed: true, idx: -1},
			{addr: 0x200013d8, size: 4, signed: true, idx: -1},
			{addr: 0x200013e0, size: 4, signed: true, idx: -1},
			{addr: 0x200013e8, size: 4, signed: true, idx: -1},
			{addr: 0x200013f0, size: 4, signed: true, idx: -1},
			{addr: 0x200013f8, size: 4, signed: true, idx: -1},
			{addr: 0x20001400, size: 4, signed: true, idx: -1},
			{addr: 0x20001408, size: 4, signed: true, idx: -1},
			{addr: 0x20001410, size: 4, signed: true, idx: -1},
			{addr: 0x20001418, size: 4, signed: true, idx: -1},
			{addr: 0x20001420, size: 4, signed: true, idx: -1},
			{addr: 0x20001428, size: 4, signed: true, idx: -1},
			{addr: 0x20001430, size: 4, signed: true, idx: -1},
			{addr: 0x20001438, size: 4, signed: true, idx: -1},
			{addr: 0x20001440, size: 4, signed: true, idx: -1},
			{addr: 0x20001448, size: 4, signed: true, idx: -1},
			{addr: 0x20001450, size: 4, signed: true, idx: -1},
			{addr: 0x20001458, size: 4, signed: true, idx: -1},
			{addr: 0x20001460, size: 4, signed: true, idx: -1},
			{addr: 0x20001468, size: 4, signed: true, idx: -1},
			{addr: 0x20001470, size: 4, signed: true, idx: -1},
			{addr: 0x20001478, size: 4, signed: true, idx: -1},
			{addr: 0x20001480, size: 4, signed: true, idx: -1},
			{addr: 0x20001488, size: 4, signed: true, idx: -1},
			{addr: 0x20001490, size: 4, signed: true, idx: -1},
			{addr: 0x20001498, size: 4, signed: true, idx: -1},
			{addr: 0x200014a0, size: 4, signed: true, idx: -1},
			{addr: 0x200014a8, size: 4, signed: true, idx: -1},
			{addr: 0x200014b0, size: 4, signed: true, idx: -1},
			{addr: 0x200014b8, size: 4, signed: true, idx: -1},
			{addr: 0x200014c0, size: 4, signed: true, idx: -1},
			{addr: 0x200014c8, size: 4, signed: true, idx: -1},
			{addr: 0x200014d0, size: 4, signed: true, idx: -1},
			{addr: 0x200014d8, size: 4, signed: true, idx: -1},
			{addr: 0x200014e0, size: 4, signed: true, idx: -1},
			{addr: 0x200014e8, size: 4, signed: true, idx: -1},
			{addr: 0x200014f0, size: 4, signed: true, idx: -1},
			{addr: 0x200014f8, size: 4, signed: true, idx: -1},
			{addr: 0x20001500, size: 4, signed: true, idx: -1},
			{addr: 0x20001508, size: 4, signed: true, idx: -1},
			{addr: 0x20001510, size: 4, signed: true, idx: -1},
			{addr: 0x20001518, size: 4, signed: true, idx: -1},
			{addr: 0x20001520, size: 4, signed: true, idx: -1},
			{addr: 0x20001528, size: 4, signed: true, idx: -1},
			{addr: 0x20001530, size: 4, signed: true, idx: -1},
			{addr: 0x20001538, size: 4, signed: true, idx: -1},
			{addr: 0x20001540, size: 4, signed: true, idx: -1},
			{addr: 0x20001548, size: 4, signed: true, idx: -1},
			{addr: 0x20001550, size: 4, signed: true, idx: -1},
			{addr: 0x20001558, size: 4, signed: true, idx: -1},
			{addr: 0x20001560, size: 4, signed: true, idx: -1},
			{addr: 0x20001568, size: 4, signed: true, idx: -1},
			{addr: 0x20001570, size: 4, signed: true, idx: -1},
			{addr: 0x20001578, size: 4, signed: true, idx: -1},
			{addr: 0x20001580, size: 4, signed: true, idx: -1},
			{addr: 0x20001588, size: 4, signed: true, idx: -1},
			{addr: 0x20001590, size: 4, signed: true, idx: -1},
			{addr: 0x20001598, size: 4, signed: true, idx: -1},
			{addr: 0x200015a0, size: 4, signed: true, idx: -1},
			{addr: 0x200015a8, size: 4, signed: true, idx: -1},
			{addr: 0x200015b0, size: 4, signed: true, idx: -1},
			{addr: 0x200015b8, size: 4, signed: true, idx: -1},
			{addr: 0x200015c0, size: 4, signed: true, idx: -1},
			{addr: 0x200015c8, size: 4, signed: true, idx: -1},
			{addr: 0x200015d0, size: 4, signed: true, idx: -1},
			{addr: 0x200015d8, size: 4, signed: true, idx: -1},
			{addr: 0x200015e0, size: 4, signed: true, idx: -1},
			{addr: 0x200015e8, size: 4, signed: true, idx: -1},
			{addr: 0x200015f0, size: 4, signed: true, idx: -1},
			{addr: 0x200015f8, size: 4, signed: true, idx: -1},
			{addr: 0x20001600, size: 4, signed: true, idx: -1},
			{addr: 0x20001608, size: 4, signed: true, idx: -1},
			{addr: 0x20001610, size: 4, signed: true, idx: -1},
			{addr: 0x20001618, size: 4, signed: true, idx: -1},
			{addr: 0x20001620, size: 4, signed: true, idx: -1},
			{addr: 0x20001628, size: 4, signed: true, idx: -1},
			{addr: 0x20001630, size: 4, signed: true, idx: -1},
			{addr: 0x20001638, size: 4, signed: true, idx: -1},
			{addr: 0x20001640, size: 4, signed: true, idx: -1},
			{addr: 0x20001648, size: 4, signed: true, idx: -1},
			{addr: 0x20001650, size: 4, signed: true, idx: -1},
			{addr: 0x20001658, size: 8, signed: false, idx: -1},
			{addr: 0x20001660, size: 4, signed: true, idx: -1},
			{addr: 0x20001668, size: 4, signed: true, idx: -1},
			{addr: 0x20001670, size: 4, signed: true, idx: -1},
			{addr: 0x20001678, size: 4, signed: true, idx: -1},
			{addr: 0x20001680, size: 8, signed: false, idx: -1},
			{addr: 0x20001688, size: 4, signed: true, idx: -1},
			{addr: 0x20001690, size: 4, signed: true, idx: -1},
			{addr: 0x20001698, size: 4, signed: true, idx: -1},
			{addr: 0x200016a0, size: 4, signed: true, idx: -1},
			{addr: 0x200016a8, size: 4, signed: true, idx: -1},
			{addr: 0x200016b0, size: 4, signed: true, idx: -1},
			{addr: 0x200016b8, size: 4, signed: true, idx: -1},
			{addr: 0x200016c0, size: 4, signed: true, idx: -1},
			{addr: 0x200016c8, size: 4, signed: true, idx: -1},
			{addr: 0x200016d0, size: 4, signed: true, idx: -1},
			{addr: 0x200016d8, size: 4, signed: true, idx: -1},
			{addr: 0x200016e0, size: 4, signed: true, idx: -1},
			{addr: 0x200016e8, size: 4, signed: true, idx: -1},
			{addr: 0x200016f0, size: 4, signed: true, idx: -1},
			{addr: 0x200016f8, size: 8, signed: false, idx: -1},
			{addr: 0x20001700, size: 4, signed: true, idx: -1},
			{addr: 0x20001708, size: 4, signed: true, idx: -1},
			{addr: 0x20001710, size: 4, signed: true, idx: -1},
			{addr: 0x20001718, size: 4, signed: true, idx: -1},
			{addr: 0x20001720, size: 4, signed: true, idx: -1},
			{addr: 0x20001728, size: 4, signed: true, idx: -1},
			{addr: 0x20001730, size: 4, signed: true, idx: -1},
			{addr: 0x20001738, size: 4, signed: true, idx: -1},
			{addr: 0x20001740, size: 4, signed: true, idx: -1},
			{addr: 0x20001748, size: 8, signed: false, idx: -1},
			{addr: 0x20001750, size: 4, signed: true, idx: -1},
			{addr: 0x20001758, size: 4, signed: true, idx: -1},
			{addr: 0x20001760, size: 4, signed: true, idx: -1},
			{addr: 0x20001768, size: 4, signed: true, idx: -1},
			{addr: 0x20001770, size: 4, signed: true, idx: -1},
			{addr: 0x20001778, size: 4, signed: true, idx: -1},
			{addr: 0x20001780, size: 4, signed: true, idx: -1},
			{addr: 0x20001788, size: 4, signed: true, idx: -1},
			{addr: 0x20001790, size: 8, signed: false, idx: -1},
			{addr: 0x20001798, size: 4, signed: true, idx: -1},
			{addr: 0x200017a0, size: 8, signed: false, idx: -1},
			{addr: 0x200017a8, size: 4, signed: true, idx: -1},
			{addr: 0x200017b0, size: 4, signed: true, idx: -1},
			{addr: 0x200017b8, size: 4, signed: true, idx: -1},
			{addr: 0x200017c0, size: 4, signed: true, idx: -1},
			{addr: 0x200017c8, size: 4, signed: true, idx: -1},
			{addr: 0x200017d0, size: 4, signed: true, idx: -1},
			{addr: 0x200017d8, size: 4, signed: true, idx: -1},
			{addr: 0x200017e0, size: 4, signed: true, idx: -1},
			{addr: 0x200017e8, size: 4, signed: true, idx: -1},
			{addr: 0x200017f0, size: 4, signed: true, idx: -1},
			{addr: 0x200017f8, size: 4, signed: true, idx: -1},
			{addr: 0x20001800, size: 4, signed: true, idx: -1},
			{addr: 0x20001808, size: 4, signed: true, idx: -1},
			{addr: 0x20001810, size: 4, signed: true, idx: -1},
			{addr: 0x20001818, size: 4, signed: true, idx: -1},
			{addr: 0x20001820, size: 4, signed: true, idx: -1},
			{addr: 0x20001828, size: 4, signed: true, idx: -1},
			{addr: 0x20001830, size: 4, signed: true, idx: -1},
			{addr: 0x20001838, size: 4, signed: true, idx: -1},
			{addr: 0x20001840, size: 4, signed: true, idx: -1},
			{addr: 0x20001848, size: 4, signed: true, idx: -1},
			{addr: 0x20001850, size: 4, signed: true, idx: -1},
			{addr: 0x20001858, size: 4, signed: true, idx: -1},
			{addr: 0x20001860, size: 4, signed: true, idx: -1},
			{addr: 0x20001868, size: 4, signed: true, idx: -1},
			{addr: 0x20001870, size: 4, signed: true, idx: -1},
			{addr: 0x20001878, size: 4, signed: true, idx: -1},
			{addr: 0x20001880, size: 4, signed: true, idx: -1},
			{addr: 0x20001888, size: 4, signed: true, idx: -1},
			{addr: 0x20001890, size: 4, signed: true, idx: -1},
			{addr: 0x20001898, size: 4, signed: true, idx: -1},
			{addr: 0x200018a0, size: 4, signed: true, idx: -1},
			{addr: 0x200018a8, size: 4, signed: true, idx: -1},
			{addr: 0x200018b0, size: 4, signed: true, idx: -1},
			{addr: 0x200018b8, size: 8, signed: false, idx: -1},
			{addr: 0x200018c0, size: 4, signed: true, idx: -1},
			{addr: 0x200018c8, size: 8, signed: false, idx: -1},
			{addr: 0x200018d0, size: 8, signed: false, idx: -1},
			{addr: 0x200018d8, size: 4, signed: true, idx: -1},
			{addr: 0x200018e0, size: 4, signed: true, idx: -1},
			{addr: 0x200018e8, size: 4, signed: true, idx: -1},
			{addr: 0x200018f0, size: 4, signed: true, idx: -1},
			{addr: 0x200018f8, size: 4, signed: true, idx: -1},
			{addr: 0x20001900, size: 4, signed: true, idx: -1},
			{addr: 0x20001908, size: 8, signed: false, idx: -1},
			{addr: 0x20001910, size: 4, signed: true, idx: -1},
			{addr: 0x20001918, size: 8, signed: false, idx: -1},
			{addr: 0x20001920, size: 4, signed: true, idx: -1},
			{addr: 0x20001928, size: 4, signed: true, idx: -1},
			{addr: 0x20001930, size: 4, signed: true, idx: -1},
			{addr: 0x20001938, size: 4, signed: true, idx: -1},
			{addr: 0x20001940, size: 4, signed: true, idx: -1},
			{addr: 0x20001948, size: 4, signed: true, idx: -1},
			{addr: 0x20001950, size: 4, signed: true, idx: -1},
			{addr: 0x20001958, size: 4, signed: true, idx: -1},
			{addr: 0x20001960, size: 8, signed: false, idx: -1},
			{addr: 0x20001968, size: 8, signed: false, idx: -1},
			{addr: 0x20001970, size: 4, signed: true, idx: -1},
			{addr: 0x20001978, size: 4, signed: true, idx: -1},
			{addr: 0x20001980, size: 4, signed: true, idx: -1},
			{addr: 0x20001988, size: 4, signed: true, idx: -1},
			{addr: 0x20001990, size: 4, signed: true, idx: -1},
			{addr: 0x20001998, size: 4, signed: true, idx: -1},
			{addr: 0x200019a0, size: 4, signed: true, idx: -1},
			{addr: 0x200019a8, size: 4, signed: true, idx: -1},
			{addr: 0x200019b0, size: 4, signed: true, idx: -1},
			{addr: 0x200019b8, size: 4, signed: true, idx: -1},
			{addr: 0x200019c0, size: 4, signed: true, idx: -1},
			{addr: 0x200019c8, size: 4, signed: true, idx: -1},
			{addr: 0x200019d0, size: 4, signed: true, idx: -1},
			{addr: 0x200019d8, size: 4, signed: true, idx: -1},
			{addr: 0x200019e0, size: 4, signed: true, idx: -1},
			{addr: 0x200019e8, size: 4, signed: true, idx: -1},
			{addr: 0x200019f0, size: 8, signed: false, idx: -1},
			{addr: 0x200019f8, size: 4, signed: true, idx: -1},
			{addr: 0x20001a00, size: 4, signed: true, idx: -1},
			{addr: 0x20001a08, size: 4, signed: true, idx: -1},
			{addr: 0x20001a10, size: 4, signed: true, idx: -1},
			{addr: 0x20001a18, size: 4, signed: true, idx: -1},
			{addr: 0x20001a20, size: 4, signed: true, idx: -1},
			{addr: 0x20001a28, size: 4, signed: true, idx: -1},
			{addr: 0x20001a30, size: 4, signed: true, idx: -1},
			{addr: 0x20001a38, size: 4, signed: true, idx: -1},
			{addr: 0x20001a40, size: 4, signed: true, idx: -1},
			{addr: 0x20001a48, size: 4, signed: true, idx: -1},
			{addr: 0x20001a50, size: 4, signed: true, idx: -1},
			{addr: 0x20001a58, size: 4, signed: true, idx: -1},
			{addr: 0x20001a60, size: 4, signed: true, idx: -1},
			{addr: 0x20001a68, size: 4, signed: true, idx: -1},
			{addr: 0x20001a70, size: 4, signed: true, idx: -1},
			{addr: 0x20001a78, size: 4, signed: true, idx: -1},
			{addr: 0x20001a80, size: 4, signed: true, idx: -1},
			{addr: 0x20001a88, size: 4, signed: true, idx: -1},
			{addr: 0x20001a90, size: 4, signed: true, idx: -1},
			{addr: 0x20001a98, size: 4, signed: true, idx: -1},
			{addr: 0x20001aa0, size: 4, signed: true, idx: -1},
			{addr: 0x20001aa8, size: 4, signed: true, idx: -1},
			{addr: 0x20001ab0, size: 4, signed: true, idx: -1},
			{addr: 0x20001ab8, size: 4, signed: true, idx: -1},
			{addr: 0x20001ac0, size: 4, signed: true, idx: -1},
			{addr: 0x20001ac8, size: 4, signed: true, idx: -1},
			{addr: 0x20001ad0, size: 4, signed: true, idx: -1},
			{addr: 0x20001ad8, size: 4, signed: true, idx: -1},
			{addr: 0x20001ae0, size: 4, signed: true, idx: -1},
			{addr: 0x20001ae8, size: 4, signed: true, idx: -1},
			{addr: 0x20001af0, size: 4, signed: true, idx: -1},
			{addr: 0x20001af8, size: 4, signed: true, idx: -1},
			{addr: 0x20001b00, size: 4, signed: true, idx: -1},
			{addr: 0x20001b08, size: 4, signed: true, idx: -1},
			{addr: 0x20001b10, size: 4, signed: true, idx: -1},
			{addr: 0x20001b18, size: 4, signed: true, idx: -1},
			{addr: 0x20001b20, size: 4, signed: true, idx: -1},
			{addr: 0x20001b28, size: 4, signed: true, idx: -1},
			{addr: 0x20001b30, size: 4, signed: true, idx: -1},
			{addr: 0x20001b38, size: 4, signed: true, idx: -1},
			{addr: 0x20001b40, size: 4, signed: true, idx: -1},
			{addr: 0x20001b48, size: 4, signed: true, idx: -1},
			{addr: 0x20001b50, size: 4, signed: true, idx: -1},
			{addr: 0x20001b58, size: 4, signed: true, idx: -1},
			{addr: 0x20001b60, size: 4, signed: true, idx: -1},
			{addr: 0x20001b68, size: 4, signed: true, idx: -1},
			{addr: 0x20001b70, size: 4, signed: true, idx: -1},
			{addr: 0x20001b78, size: 4, signed: true, idx: -1},
			{addr: 0x20001b80, size: 4, signed: true, idx: -1},
			{addr: 0x20001b88, size: 4, signed: true, idx: -1},
			{addr: 0x20001b90, size: 4, signed: true, idx: -1},
			{addr: 0x20001b98, size: 4, signed: true, idx: -1},
			{addr: 0x20001ba0, size: 4, signed: true, idx: -1},
			{addr: 0x20001ba8, size: 4, signed: true, idx: -1},
			{addr: 0x20001bb0, size: 4, signed: true, idx: -1},
			{addr: 0x20001bb8, size: 4, signed: true, idx: -1},
			{addr: 0x20001bc0, size: 4, signed: true, idx: -1},
			{addr: 0x20001bc8, size: 4, signed: true, idx: -1},
			{addr: 0x20001bd0, size: 4, signed: true, idx: -1},
		},
		out: []mstFld{
			{addr: 0x20002681, size: 4, idx: -1},
			{addr: 0x2000265a, size: 4, idx: -1},
			{addr: 0x2000265e, size: 4, idx: -1},
			{addr: 0x20002662, size: 4, idx: -1},
			{addr: 0x20002666, size: 4, idx: -1},
			{addr: 0x2000266a, size: 4, idx: -1},
			{addr: 0x2000266e, size: 4, idx: -1},
			{addr: 0x20002672, size: 4, idx: -1},
			{addr: 0x20002676, size: 4, idx: -1},
			{addr: 0x20002316, size: 4, idx: -1},
			{addr: 0x20002340, size: 4, idx: -1},
			{addr: 0x20000100, size: 4, idx: -1},
			{addr: 0x20000104, size: 4, idx: -1},
			{addr: 0x20000108, size: 4, idx: -1},
			{addr: 0x2000010c, size: 4, idx: -1},
			{addr: 0x20000110, size: 4, idx: -1},
			{addr: 0x20000114, size: 4, idx: -1},
			{addr: 0x20000118, size: 4, idx: -1},
			{addr: 0x2000011c, size: 4, idx: -1},
			{addr: 0x20000120, size: 4, idx: -1},
			{addr: 0x20000124, size: 4, idx: -1},
			{addr: 0x20000128, size: 4, idx: -1},
			{addr: 0x2000012c, size: 4, idx: -1},
			{addr: 0x20000130, size: 4, idx: -1},
			{addr: 0x20000134, size: 4, idx: -1},
			{addr: 0x20000138, size: 4, idx: -1},
			{addr: 0x2000013c, size: 4, idx: -1},
			{addr: 0x20000140, size: 4, idx: -1},
			{addr: 0x20000144, size: 4, idx: -1},
			{addr: 0x20000148, size: 4, idx: -1},
			{addr: 0x2000014c, size: 4, idx: -1},
			{addr: 0x20000150, size: 4, idx: -1},
			{addr: 0x20000154, size: 4, idx: -1},
			{addr: 0x20000158, size: 4, idx: -1},
			{addr: 0x2000015c, size: 4, idx: -1},
			{addr: 0x20000160, size: 4, idx: -1},
			{addr: 0x20000164, size: 4, idx: -1},
			{addr: 0x20000168, size: 4, idx: -1},
			{addr: 0x2000016c, size: 4, idx: -1},
			{addr: 0x20000170, size: 4, idx: -1},
			{addr: 0x20000174, size: 4, idx: -1},
			{addr: 0x20000178, size: 4, idx: -1},
			{addr: 0x2000017c, size: 4, idx: -1},
			{addr: 0x20000180, size: 4, idx: -1},
			{addr: 0x20000184, size: 4, idx: -1},
			{addr: 0x20000188, size: 4, idx: -1},
			{addr: 0x2000018c, size: 4, idx: -1},
			{addr: 0x20000190, size: 4, idx: -1},
			{addr: 0x20000194, size: 4, idx: -1},
			{addr: 0x20000198, size: 4, idx: -1},
			{addr: 0x2000019c, size: 4, idx: -1},
			{addr: 0x200001a0, size: 4, idx: -1},
			{addr: 0x200001a4, size: 4, idx: -1},
			{addr: 0x200001a8, size: 4, idx: -1},
			{addr: 0x200001ac, size: 4, idx: -1},
			{addr: 0x200001b0, size: 4, idx: -1},
			{addr: 0x200001b4, size: 4, idx: -1},
			{addr: 0x200001b8, size: 4, idx: -1},
			{addr: 0x200001bc, size: 4, idx: -1},
			{addr: 0x200001c0, size: 4, idx: -1},
			{addr: 0x200001c4, size: 4, idx: -1},
			{addr: 0x200001c8, size: 4, idx: -1},
			{addr: 0x200001cc, size: 4, idx: -1},
			{addr: 0x200001d0, size: 4, idx: -1},
			{addr: 0x200001d4, size: 4, idx: -1},
			{addr: 0x200001d8, size: 4, idx: -1},
			{addr: 0x200001dc, size: 4, idx: -1},
			{addr: 0x200001e0, size: 4, idx: -1},
			{addr: 0x200001e4, size: 4, idx: -1},
			{addr: 0x200001e8, size: 4, idx: -1},
			{addr: 0x200001ec, size: 4, idx: -1},
			{addr: 0x200001f0, size: 4, idx: -1},
			{addr: 0x200001f4, size: 4, idx: -1},
			{addr: 0x200001f8, size: 4, idx: -1},
			{addr: 0x200001fc, size: 4, idx: -1},
			{addr: 0x20000200, size: 4, idx: -1},
			{addr: 0x20000204, size: 4, idx: -1},
			{addr: 0x20000208, size: 4, idx: -1},
			{addr: 0x2000020c, size: 4, idx: -1},
			{addr: 0x20000210, size: 4, idx: -1},
			{addr: 0x20000214, size: 4, idx: -1},
			{addr: 0x20000218, size: 4, idx: -1},
			{addr: 0x2000021c, size: 4, idx: -1},
			{addr: 0x20000220, size: 4, idx: -1},
			{addr: 0x20000224, size: 4, idx: -1},
			{addr: 0x20000228, size: 4, idx: -1},
			{addr: 0x2000022c, size: 4, idx: -1},
			{addr: 0x20000230, size: 4, idx: -1},
			{addr: 0x20000234, size: 4, idx: -1},
			{addr: 0x20000238, size: 4, idx: -1},
			{addr: 0x2000023c, size: 4, idx: -1},
			{addr: 0x20000240, size: 4, idx: -1},
			{addr: 0x20000244, size: 4, idx: -1},
			{addr: 0x20000248, size: 4, idx: -1},
			{addr: 0x2000024c, size: 4, idx: -1},
			{addr: 0x20000250, size: 4, idx: -1},
			{addr: 0x20000254, size: 4, idx: -1},
			{addr: 0x20000258, size: 4, idx: -1},
			{addr: 0x2000025c, size: 4, idx: -1},
			{addr: 0x20000260, size: 4, idx: -1},
			{addr: 0x20000264, size: 4, idx: -1},
			{addr: 0x20000268, size: 4, idx: -1},
			{addr: 0x2000026c, size: 4, idx: -1},
			{addr: 0x20000270, size: 4, idx: -1},
			{addr: 0x20000274, size: 4, idx: -1},
			{addr: 0x20000278, size: 4, idx: -1},
			{addr: 0x2000027c, size: 4, idx: -1},
			{addr: 0x20000280, size: 4, idx: -1},
			{addr: 0x20000284, size: 4, idx: -1},
			{addr: 0x20000288, size: 4, idx: -1},
			{addr: 0x2000028c, size: 4, idx: -1},
			{addr: 0x20000290, size: 4, idx: -1},
			{addr: 0x20000294, size: 4, idx: -1},
			{addr: 0x20000298, size: 4, idx: -1},
			{addr: 0x2000029c, size: 4, idx: -1},
			{addr: 0x200002a0, size: 4, idx: -1},
			{addr: 0x200002a4, size: 4, idx: -1},
			{addr: 0x200002a8, size: 4, idx: -1},
			{addr: 0x200002ac, size: 4, idx: -1},
			{addr: 0x200002b0, size: 4, idx: -1},
			{addr: 0x200002b4, size: 4, idx: -1},
			{addr: 0x200002b8, size: 4, idx: -1},
			{addr: 0x200002bc, size: 4, idx: -1},
			{addr: 0x200002c0, size: 4, idx: -1},
			{addr: 0x200002c4, size: 4, idx: -1},
			{addr: 0x200002c8, size: 4, idx: -1},
			{addr: 0x200002cc, size: 4, idx: -1},
			{addr: 0x200002d0, size: 4, idx: -1},
			{addr: 0x200002d4, size: 4, idx: -1},
			{addr: 0x200002d8, size: 4, idx: -1},
			{addr: 0x200002dc, size: 4, idx: -1},
			{addr: 0x200002e0, size: 4, idx: -1},
			{addr: 0x200002e4, size: 4, idx: -1},
			{addr: 0x200002e8, size: 4, idx: -1},
			{addr: 0x200002ec, size: 4, idx: -1},
			{addr: 0x200002f0, size: 4, idx: -1},
			{addr: 0x200002f4, size: 4, idx: -1},
			{addr: 0x200002f8, size: 4, idx: -1},
			{addr: 0x200002fc, size: 4, idx: -1},
			{addr: 0x20000300, size: 4, idx: -1},
			{addr: 0x20000304, size: 4, idx: -1},
			{addr: 0x20000308, size: 4, idx: -1},
			{addr: 0x2000030c, size: 4, idx: -1},
			{addr: 0x20000310, size: 4, idx: -1},
			{addr: 0x20000314, size: 4, idx: -1},
			{addr: 0x20000318, size: 4, idx: -1},
			{addr: 0x2000031c, size: 4, idx: -1},
			{addr: 0x20000320, size: 4, idx: -1},
			{addr: 0x20000324, size: 4, idx: -1},
			{addr: 0x20000328, size: 4, idx: -1},
			{addr: 0x2000032c, size: 4, idx: -1},
			{addr: 0x20000330, size: 4, idx: -1},
			{addr: 0x20000334, size: 4, idx: -1},
			{addr: 0x20000338, size: 4, idx: -1},
			{addr: 0x2000033c, size: 4, idx: -1},
			{addr: 0x20000340, size: 4, idx: -1},
			{addr: 0x20000344, size: 4, idx: -1},
			{addr: 0x20000348, size: 4, idx: -1},
			{addr: 0x2000034c, size: 4, idx: -1},
			{addr: 0x20000350, size: 4, idx: -1},
			{addr: 0x20000354, size: 4, idx: -1},
			{addr: 0x20000358, size: 4, idx: -1},
			{addr: 0x2000035c, size: 4, idx: -1},
			{addr: 0x20000360, size: 4, idx: -1},
			{addr: 0x20000364, size: 4, idx: -1},
			{addr: 0x20000368, size: 4, idx: -1},
			{addr: 0x2000036c, size: 4, idx: -1},
			{addr: 0x20000370, size: 4, idx: -1},
			{addr: 0x20000374, size: 4, idx: -1},
			{addr: 0x20000378, size: 4, idx: -1},
			{addr: 0x2000037c, size: 4, idx: -1},
			{addr: 0x20000380, size: 4, idx: -1},
			{addr: 0x20000384, size: 4, idx: -1},
			{addr: 0x20000388, size: 4, idx: -1},
			{addr: 0x2000038c, size: 4, idx: -1},
			{addr: 0x20000390, size: 4, idx: -1},
			{addr: 0x20000394, size: 4, idx: -1},
			{addr: 0x20000398, size: 4, idx: -1},
			{addr: 0x2000039c, size: 4, idx: -1},
			{addr: 0x200003a0, size: 4, idx: -1},
			{addr: 0x200003a4, size: 4, idx: -1},
			{addr: 0x200003a8, size: 4, idx: -1},
			{addr: 0x200003ac, size: 4, idx: -1},
			{addr: 0x200003b0, size: 4, idx: -1},
			{addr: 0x200003b4, size: 4, idx: -1},
			{addr: 0x200003b8, size: 4, idx: -1},
			{addr: 0x200003bc, size: 4, idx: -1},
			{addr: 0x200003c0, size: 4, idx: -1},
			{addr: 0x200003c4, size: 4, idx: -1},
			{addr: 0x200003c8, size: 4, idx: -1},
			{addr: 0x200003cc, size: 4, idx: -1},
			{addr: 0x200003d0, size: 4, idx: -1},
			{addr: 0x200003d4, size: 4, idx: -1},
			{addr: 0x200003d8, size: 4, idx: -1},
			{addr: 0x200003dc, size: 4, idx: -1},
			{addr: 0x200003e0, size: 4, idx: -1},
			{addr: 0x200003e4, size: 4, idx: -1},
			{addr: 0x200003e8, size: 4, idx: -1},
			{addr: 0x200003ec, size: 4, idx: -1},
			{addr: 0x200003f0, size: 4, idx: -1},
			{addr: 0x200003f4, size: 4, idx: -1},
			{addr: 0x200003f8, size: 4, idx: -1},
			{addr: 0x200003fc, size: 4, idx: -1},
			{addr: 0x20000400, size: 4, idx: -1},
			{addr: 0x20000404, size: 4, idx: -1},
			{addr: 0x20000408, size: 4, idx: -1},
			{addr: 0x2000040c, size: 4, idx: -1},
			{addr: 0x20000410, size: 4, idx: -1},
			{addr: 0x20000414, size: 4, idx: -1},
			{addr: 0x20000418, size: 4, idx: -1},
			{addr: 0x2000041c, size: 4, idx: -1},
			{addr: 0x20000420, size: 4, idx: -1},
			{addr: 0x20000424, size: 4, idx: -1},
			{addr: 0x20000428, size: 4, idx: -1},
			{addr: 0x2000042c, size: 4, idx: -1},
			{addr: 0x20000430, size: 4, idx: -1},
			{addr: 0x20000434, size: 4, idx: -1},
			{addr: 0x20000438, size: 4, idx: -1},
			{addr: 0x2000043c, size: 4, idx: -1},
			{addr: 0x20000440, size: 4, idx: -1},
			{addr: 0x20000444, size: 4, idx: -1},
			{addr: 0x20000448, size: 4, idx: -1},
			{addr: 0x2000044c, size: 4, idx: -1},
			{addr: 0x20000450, size: 4, idx: -1},
			{addr: 0x20000454, size: 4, idx: -1},
			{addr: 0x20000458, size: 4, idx: -1},
			{addr: 0x2000045c, size: 4, idx: -1},
			{addr: 0x20000460, size: 4, idx: -1},
			{addr: 0x20000464, size: 4, idx: -1},
			{addr: 0x20000468, size: 4, idx: -1},
			{addr: 0x2000046c, size: 4, idx: -1},
			{addr: 0x20000470, size: 4, idx: -1},
			{addr: 0x20000474, size: 4, idx: -1},
			{addr: 0x20000478, size: 4, idx: -1},
			{addr: 0x2000047c, size: 4, idx: -1},
			{addr: 0x20000480, size: 4, idx: -1},
			{addr: 0x20000484, size: 4, idx: -1},
			{addr: 0x20000488, size: 4, idx: -1},
			{addr: 0x2000048c, size: 4, idx: -1},
			{addr: 0x20000490, size: 4, idx: -1},
			{addr: 0x20000494, size: 4, idx: -1},
			{addr: 0x20000498, size: 4, idx: -1},
			{addr: 0x2000049c, size: 4, idx: -1},
			{addr: 0x200004a0, size: 4, idx: -1},
			{addr: 0x200004a4, size: 4, idx: -1},
			{addr: 0x200004a8, size: 4, idx: -1},
			{addr: 0x200004ac, size: 4, idx: -1},
			{addr: 0x200004b0, size: 4, idx: -1},
			{addr: 0x200004b4, size: 4, idx: -1},
			{addr: 0x200004b8, size: 4, idx: -1},
			{addr: 0x200004bc, size: 4, idx: -1},
			{addr: 0x200004c0, size: 4, idx: -1},
			{addr: 0x200004c4, size: 4, idx: -1},
			{addr: 0x200004c8, size: 4, idx: -1},
			{addr: 0x200004cc, size: 4, idx: -1},
			{addr: 0x200004d0, size: 4, idx: -1},
			{addr: 0x200004d4, size: 4, idx: -1},
			{addr: 0x200004d8, size: 4, idx: -1},
			{addr: 0x200004dc, size: 4, idx: -1},
			{addr: 0x200004e0, size: 4, idx: -1},
			{addr: 0x200004e4, size: 4, idx: -1},
			{addr: 0x200004e8, size: 4, idx: -1},
			{addr: 0x200004ec, size: 4, idx: -1},
			{addr: 0x200004f0, size: 4, idx: -1},
			{addr: 0x200004f4, size: 4, idx: -1},
			{addr: 0x200004f8, size: 4, idx: -1},
			{addr: 0x200004fc, size: 4, idx: -1},
			{addr: 0x20000500, size: 4, idx: -1},
			{addr: 0x20000504, size: 4, idx: -1},
			{addr: 0x20000508, size: 4, idx: -1},
			{addr: 0x2000050c, size: 4, idx: -1},
			{addr: 0x20000510, size: 4, idx: -1},
			{addr: 0x20000514, size: 4, idx: -1},
			{addr: 0x20000518, size: 4, idx: -1},
			{addr: 0x2000051c, size: 4, idx: -1},
			{addr: 0x20000520, size: 4, idx: -1},
			{addr: 0x20000524, size: 4, idx: -1},
			{addr: 0x20000528, size: 4, idx: -1},
			{addr: 0x2000052c, size: 4, idx: -1},
			{addr: 0x20000530, size: 4, idx: -1},
			{addr: 0x20000534, size: 4, idx: -1},
			{addr: 0x20000538, size: 4, idx: -1},
			{addr: 0x2000053c, size: 4, idx: -1},
			{addr: 0x20000540, size: 4, idx: -1},
			{addr: 0x20000544, size: 4, idx: -1},
			{addr: 0x20000548, size: 4, idx: -1},
			{addr: 0x2000054c, size: 4, idx: -1},
			{addr: 0x20000550, size: 4, idx: -1},
			{addr: 0x20000554, size: 4, idx: -1},
			{addr: 0x20000558, size: 4, idx: -1},
			{addr: 0x2000055c, size: 4, idx: -1},
			{addr: 0x20000560, size: 4, idx: -1},
			{addr: 0x20000564, size: 4, idx: -1},
			{addr: 0x20000568, size: 4, idx: -1},
			{addr: 0x2000056c, size: 4, idx: -1},
			{addr: 0x20000570, size: 4, idx: -1},
			{addr: 0x20000574, size: 4, idx: -1},
			{addr: 0x20000578, size: 4, idx: -1},
			{addr: 0x2000057c, size: 4, idx: -1},
			{addr: 0x20000580, size: 4, idx: -1},
			{addr: 0x20000584, size: 4, idx: -1},
			{addr: 0x20000588, size: 4, idx: -1},
			{addr: 0x2000058c, size: 4, idx: -1},
			{addr: 0x20000590, size: 4, idx: -1},
			{addr: 0x20000594, size: 4, idx: -1},
			{addr: 0x20000598, size: 4, idx: -1},
			{addr: 0x2000059c, size: 4, idx: -1},
			{addr: 0x200005a0, size: 4, idx: -1},
			{addr: 0x200005a4, size: 4, idx: -1},
			{addr: 0x200005a8, size: 4, idx: -1},
			{addr: 0x200005ac, size: 4, idx: -1},
			{addr: 0x200005b0, size: 4, idx: -1},
			{addr: 0x200005b4, size: 4, idx: -1},
			{addr: 0x200005b8, size: 4, idx: -1},
			{addr: 0x200005bc, size: 4, idx: -1},
			{addr: 0x200005c0, size: 4, idx: -1},
			{addr: 0x200005c4, size: 4, idx: -1},
			{addr: 0x200005c8, size: 4, idx: -1},
			{addr: 0x200005cc, size: 4, idx: -1},
			{addr: 0x200005d0, size: 4, idx: -1},
			{addr: 0x200005d4, size: 4, idx: -1},
			{addr: 0x200005d8, size: 4, idx: -1},
			{addr: 0x200005dc, size: 4, idx: -1},
			{addr: 0x200005e0, size: 4, idx: -1},
			{addr: 0x200005e4, size: 4, idx: -1},
			{addr: 0x200005e8, size: 4, idx: -1},
			{addr: 0x200005ec, size: 4, idx: -1},
			{addr: 0x200005f0, size: 4, idx: -1},
			{addr: 0x200005f4, size: 4, idx: -1},
			{addr: 0x200005f8, size: 4, idx: -1},
			{addr: 0x200005fc, size: 4, idx: -1},
			{addr: 0x20000600, size: 4, idx: -1},
			{addr: 0x20000604, size: 4, idx: -1},
			{addr: 0x20000608, size: 4, idx: -1},
			{addr: 0x2000060c, size: 4, idx: -1},
			{addr: 0x20000610, size: 4, idx: -1},
			{addr: 0x20000614, size: 4, idx: -1},
			{addr: 0x20000618, size: 4, idx: -1},
			{addr: 0x2000061c, size: 4, idx: -1},
			{addr: 0x20000620, size: 4, idx: -1},
			{addr: 0x20000624, size: 4, idx: -1},
			{addr: 0x20000628, size: 4, idx: -1},
			{addr: 0x2000062c, size: 4, idx: -1},
			{addr: 0x20000630, size: 4, idx: -1},
			{addr: 0x20000634, size: 4, idx: -1},
			{addr: 0x20000638, size: 4, idx: -1},
			{addr: 0x2000063c, size: 4, idx: -1},
			{addr: 0x20000640, size: 4, idx: -1},
			{addr: 0x20000644, size: 4, idx: -1},
			{addr: 0x20000648, size: 4, idx: -1},
			{addr: 0x2000064c, size: 4, idx: -1},
			{addr: 0x20000650, size: 4, idx: -1},
			{addr: 0x20000654, size: 4, idx: -1},
			{addr: 0x20000658, size: 4, idx: -1},
			{addr: 0x2000065c, size: 4, idx: -1},
			{addr: 0x20000660, size: 4, idx: -1},
			{addr: 0x20000664, size: 4, idx: -1},
			{addr: 0x20000668, size: 4, idx: -1},
			{addr: 0x2000066c, size: 4, idx: -1},
			{addr: 0x20000670, size: 4, idx: -1},
			{addr: 0x20000674, size: 4, idx: -1},
			{addr: 0x20000678, size: 4, idx: -1},
			{addr: 0x2000067c, size: 4, idx: -1},
			{addr: 0x20000680, size: 4, idx: -1},
			{addr: 0x20000684, size: 4, idx: -1},
			{addr: 0x20000688, size: 4, idx: -1},
			{addr: 0x2000068c, size: 4, idx: -1},
			{addr: 0x20000690, size: 4, idx: -1},
			{addr: 0x20000694, size: 4, idx: -1},
			{addr: 0x20000698, size: 4, idx: -1},
			{addr: 0x2000069c, size: 4, idx: -1},
			{addr: 0x200006a0, size: 4, idx: -1},
			{addr: 0x200006a4, size: 4, idx: -1},
			{addr: 0x200006a8, size: 4, idx: -1},
			{addr: 0x200006ac, size: 4, idx: -1},
			{addr: 0x200006b0, size: 4, idx: -1},
			{addr: 0x200006b4, size: 4, idx: -1},
			{addr: 0x200006b8, size: 4, idx: -1},
			{addr: 0x200006bc, size: 4, idx: -1},
			{addr: 0x200006c0, size: 4, idx: -1},
			{addr: 0x200006c4, size: 4, idx: -1},
			{addr: 0x200006c8, size: 4, idx: -1},
			{addr: 0x200006cc, size: 4, idx: -1},
			{addr: 0x200006d0, size: 4, idx: -1},
			{addr: 0x200006d4, size: 4, idx: -1},
			{addr: 0x200006d8, size: 4, idx: -1},
			{addr: 0x200006dc, size: 4, idx: -1},
			{addr: 0x200006e0, size: 4, idx: -1},
			{addr: 0x200006e4, size: 4, idx: -1},
			{addr: 0x200006e8, size: 4, idx: -1},
			{addr: 0x200006ec, size: 4, idx: -1},
			{addr: 0x200006f0, size: 4, idx: -1},
			{addr: 0x200006f4, size: 4, idx: -1},
			{addr: 0x200006f8, size: 4, idx: -1},
			{addr: 0x200006fc, size: 4, idx: -1},
			{addr: 0x20000700, size: 4, idx: -1},
			{addr: 0x20000704, size: 4, idx: -1},
			{addr: 0x20000708, size: 4, idx: -1},
			{addr: 0x2000070c, size: 4, idx: -1},
			{addr: 0x20000710, size: 4, idx: -1},
			{addr: 0x20000714, size: 4, idx: -1},
			{addr: 0x20000718, size: 4, idx: -1},
			{addr: 0x2000071c, size: 4, idx: -1},
			{addr: 0x20000720, size: 4, idx: -1},
			{addr: 0x20000724, size: 4, idx: -1},
			{addr: 0x20000728, size: 4, idx: -1},
			{addr: 0x2000072c, size: 4, idx: -1},
			{addr: 0x20000730, size: 4, idx: -1},
			{addr: 0x20000734, size: 4, idx: -1},
			{addr: 0x20000738, size: 4, idx: -1},
			{addr: 0x2000073c, size: 4, idx: -1},
			{addr: 0x20000740, size: 4, idx: -1},
			{addr: 0x20000744, size: 4, idx: -1},
			{addr: 0x20000748, size: 4, idx: -1},
			{addr: 0x2000074c, size: 4, idx: -1},
			{addr: 0x20000750, size: 4, idx: -1},
			{addr: 0x20000754, size: 4, idx: -1},
			{addr: 0x20000758, size: 4, idx: -1},
			{addr: 0x2000075c, size: 4, idx: -1},
			{addr: 0x20000760, size: 4, idx: -1},
			{addr: 0x20000764, size: 4, idx: -1},
			{addr: 0x20000768, size: 4, idx: -1},
			{addr: 0x2000076c, size: 4, idx: -1},
			{addr: 0x20000770, size: 4, idx: -1},
			{addr: 0x20000774, size: 4, idx: -1},
			{addr: 0x20000778, size: 4, idx: -1},
			{addr: 0x2000077c, size: 4, idx: -1},
			{addr: 0x20000780, size: 4, idx: -1},
			{addr: 0x20000784, size: 4, idx: -1},
			{addr: 0x20000788, size: 4, idx: -1},
			{addr: 0x2000078c, size: 4, idx: -1},
			{addr: 0x20000790, size: 4, idx: -1},
			{addr: 0x20000794, size: 4, idx: -1},
			{addr: 0x20000798, size: 4, idx: -1},
			{addr: 0x2000079c, size: 4, idx: -1},
			{addr: 0x200007a0, size: 4, idx: -1},
			{addr: 0x200007a4, size: 4, idx: -1},
			{addr: 0x200007a8, size: 4, idx: -1},
			{addr: 0x200007ac, size: 4, idx: -1},
			{addr: 0x200007b0, size: 4, idx: -1},
			{addr: 0x200007b4, size: 4, idx: -1},
			{addr: 0x200007b8, size: 4, idx: -1},
			{addr: 0x200007bc, size: 4, idx: -1},
			{addr: 0x200007c0, size: 4, idx: -1},
			{addr: 0x200007c4, size: 4, idx: -1},
			{addr: 0x200007c8, size: 4, idx: -1},
			{addr: 0x200007cc, size: 4, idx: -1},
			{addr: 0x200007d0, size: 4, idx: -1},
			{addr: 0x200007d4, size: 4, idx: -1},
			{addr: 0x200007d8, size: 4, idx: -1},
			{addr: 0x200007dc, size: 4, idx: -1},
			{addr: 0x200007e0, size: 4, idx: -1},
			{addr: 0x200007e4, size: 4, idx: -1},
			{addr: 0x200007e8, size: 4, idx: -1},
			{addr: 0x200007ec, size: 4, idx: -1},
			{addr: 0x200007f0, size: 4, idx: -1},
			{addr: 0x200007f4, size: 4, idx: -1},
			{addr: 0x200007f8, size: 4, idx: -1},
			{addr: 0x200007fc, size: 4, idx: -1},
			{addr: 0x20000800, size: 4, idx: -1},
			{addr: 0x20000804, size: 4, idx: -1},
			{addr: 0x20000808, size: 4, idx: -1},
			{addr: 0x2000080c, size: 4, idx: -1},
			{addr: 0x20000810, size: 4, idx: -1},
			{addr: 0x20000814, size: 4, idx: -1},
			{addr: 0x20000818, size: 4, idx: -1},
			{addr: 0x2000081c, size: 4, idx: -1},
			{addr: 0x20000820, size: 4, idx: -1},
			{addr: 0x20000824, size: 4, idx: -1},
			{addr: 0x20000828, size: 4, idx: -1},
			{addr: 0x2000082c, size: 4, idx: -1},
			{addr: 0x20000830, size: 4, idx: -1},
			{addr: 0x20000834, size: 4, idx: -1},
			{addr: 0x20000838, size: 4, idx: -1},
			{addr: 0x2000083c, size: 4, idx: -1},
			{addr: 0x20000840, size: 4, idx: -1},
			{addr: 0x20000844, size: 4, idx: -1},
			{addr: 0x20000848, size: 4, idx: -1},
			{addr: 0x2000084c, size: 4, idx: -1},
			{addr: 0x20000850, size: 4, idx: -1},
			{addr: 0x20000854, size: 4, idx: -1},
			{addr: 0x20000858, size: 4, idx: -1},
			{addr: 0x2000085c, size: 4, idx: -1},
			{addr: 0x20000860, size: 4, idx: -1},
			{addr: 0x20000864, size: 4, idx: -1},
			{addr: 0x20000868, size: 4, idx: -1},
			{addr: 0x2000086c, size: 4, idx: -1},
			{addr: 0x20000870, size: 4, idx: -1},
			{addr: 0x20000874, size: 4, idx: -1},
			{addr: 0x20000878, size: 4, idx: -1},
			{addr: 0x2000087c, size: 4, idx: -1},
			{addr: 0x20000880, size: 4, idx: -1},
			{addr: 0x20000884, size: 4, idx: -1},
			{addr: 0x20000888, size: 4, idx: -1},
			{addr: 0x2000088c, size: 4, idx: -1},
			{addr: 0x20000890, size: 4, idx: -1},
			{addr: 0x20000894, size: 4, idx: -1},
			{addr: 0x20000898, size: 4, idx: -1},
			{addr: 0x2000089c, size: 4, idx: -1},
			{addr: 0x200008a0, size: 4, idx: -1},
			{addr: 0x200008a4, size: 4, idx: -1},
			{addr: 0x200008a8, size: 4, idx: -1},
			{addr: 0x200008ac, size: 4, idx: -1},
			{addr: 0x200008b0, size: 4, idx: -1},
			{addr: 0x200008b4, size: 4, idx: -1},
			{addr: 0x200008b8, size: 4, idx: -1},
			{addr: 0x200008bc, size: 4, idx: -1},
			{addr: 0x200008c0, size: 4, idx: -1},
			{addr: 0x200008c4, size: 4, idx: -1},
			{addr: 0x200008c8, size: 4, idx: -1},
			{addr: 0x200008cc, size: 4, idx: -1},
			{addr: 0x200008d0, size: 4, idx: -1},
			{addr: 0x200008d4, size: 4, idx: -1},
			{addr: 0x200008d8, size: 4, idx: -1},
			{addr: 0x200008dc, size: 4, idx: -1},
			{addr: 0x200008e0, size: 4, idx: -1},
			{addr: 0x200008e4, size: 4, idx: -1},
			{addr: 0x200008e8, size: 4, idx: -1},
			{addr: 0x200008ec, size: 4, idx: -1},
			{addr: 0x200008f0, size: 4, idx: -1},
			{addr: 0x200008f4, size: 4, idx: -1},
			{addr: 0x200008f8, size: 4, idx: -1},
			{addr: 0x200008fc, size: 4, idx: -1},
			{addr: 0x20000900, size: 4, idx: -1},
			{addr: 0x20000904, size: 4, idx: -1},
			{addr: 0x20000908, size: 4, idx: -1},
			{addr: 0x2000090c, size: 4, idx: -1},
			{addr: 0x20000910, size: 4, idx: -1},
			{addr: 0x20000914, size: 4, idx: -1},
			{addr: 0x20000918, size: 4, idx: -1},
			{addr: 0x2000091c, size: 4, idx: -1},
			{addr: 0x20000920, size: 4, idx: -1},
			{addr: 0x20000924, size: 4, idx: -1},
			{addr: 0x20000928, size: 4, idx: -1},
			{addr: 0x2000092c, size: 4, idx: -1},
			{addr: 0x20000930, size: 4, idx: -1},
			{addr: 0x20000934, size: 4, idx: -1},
			{addr: 0x20000938, size: 4, idx: -1},
			{addr: 0x2000093c, size: 4, idx: -1},
			{addr: 0x20000940, size: 4, idx: -1},
			{addr: 0x20000944, size: 4, idx: -1},
			{addr: 0x20000948, size: 4, idx: -1},
			{addr: 0x2000094c, size: 4, idx: -1},
			{addr: 0x20000950, size: 4, idx: -1},
			{addr: 0x20000954, size: 4, idx: -1},
			{addr: 0x20000958, size: 4, idx: -1},
			{addr: 0x2000095c, size: 4, idx: -1},
			{addr: 0x20000960, size: 4, idx: -1},
			{addr: 0x20000964, size: 4, idx: -1},
			{addr: 0x20000968, size: 4, idx: -1},
			{addr: 0x2000096c, size: 4, idx: -1},
			{addr: 0x20000970, size: 4, idx: -1},
			{addr: 0x20000974, size: 4, idx: -1},
			{addr: 0x20000978, size: 4, idx: -1},
			{addr: 0x2000097c, size: 4, idx: -1},
			{addr: 0x20000980, size: 4, idx: -1},
			{addr: 0x20000984, size: 4, idx: -1},
			{addr: 0x20000988, size: 4, idx: -1},
			{addr: 0x2000098c, size: 4, idx: -1},
			{addr: 0x20000990, size: 4, idx: -1},
			{addr: 0x20000994, size: 4, idx: -1},
			{addr: 0x20000998, size: 4, idx: -1},
			{addr: 0x2000099c, size: 4, idx: -1},
			{addr: 0x200009a0, size: 4, idx: -1},
			{addr: 0x200009a4, size: 4, idx: -1},
			{addr: 0x200009a8, size: 4, idx: -1},
			{addr: 0x200009ac, size: 4, idx: -1},
		},
	},
	stub: [macStubs]macStub{
		macRoll:  {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20000120, 96}, {3, 4, 0x200002a0, 96}, {4, 4, 0x20000420, 96}, {5, 4, 0x200005a0, 96}, {6, 4, 0x20000720, 96}}, cnt: 0x20000100},
		macIcpt:  {ret: 0x20001100, retN: 12, rec: [][4]uint32{}, cnt: 0x20000104},
		macValid: {ret: 0x20001130, retN: 0, rec: [][4]uint32{}, cnt: 0x20000108},
		macDisp:  {ret: 0x20001134, retN: 24, rec: [][4]uint32{{1, 4, 0x200008a0, 24}}, cnt: 0x2000010c},
		macAvail: {ret: 0x20001194, retN: 3, rec: [][4]uint32{{2, 4, 0x20000900, 3}, {3, 4, 0x2000090c, 3}}, cnt: 0x20000110},
		macPen:   {ret: 0x200011a0, retN: 24, rec: [][4]uint32{{1, 4, 0x20000918, 24}}, cnt: 0x20000114},
		macGrow:  {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000978, 8}}, cnt: 0x20000118},
		macLog:   {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20000998, 3}, {3, 4, 0x200009a4, 3}}, cnt: 0x2000011c},
	},
	rollOff:  0x2fffdc68,
	rollW:    0x20001200,
	parkIcpt: 250, parkValid: 298, parkAvail: 327, parkPen: 494,
}

func macNames_in() []string {
	return []string{"a", "b", "g94", "partyX", "partyY", "partyZ", "partyR", "c.k", "c.f15", "c.list", "old.data", "monCount", "monEntry0", "monEntry1", "monEntry2", "monEntry3", "resCount", "resEntry0", "resEntry1", "M.id", "M.tmpl", "M.f14", "M.f15", "M.f16", "M.f2b", "M.f8b", "M.ax", "M.ay", "M.az", "M.ar", "mon1.id", "mon1.tmpl", "mon1.f14", "mon1.f15", "mon1.f16", "mon1.f2b", "mon1.f8b", "mon1.ax", "mon1.ay", "mon1.az", "mon1.ar", "mon2.id", "mon2.tmpl", "mon2.f14", "mon2.f15", "mon2.f16", "mon2.f2b", "mon2.f8b", "mon2.ax", "mon2.ay", "mon2.az", "mon2.ar", "mon3.id", "mon3.tmpl", "mon3.f14", "mon3.f15", "mon3.f16", "mon3.f2b", "mon3.f8b", "mon3.ax", "mon3.ay", "mon3.az", "mon3.ar", "rmon0.id", "rmon0.tmpl", "rmon0.f14", "rmon0.f15", "rmon0.f16", "rmon0.f2b", "rmon0.f8b", "rmon0.ax", "rmon0.ay", "rmon0.az", "rmon0.ar", "rmon1.id", "rmon1.tmpl", "rmon1.f14", "rmon1.f15", "rmon1.f16", "rmon1.f2b", "rmon1.f8b", "rmon1.ax", "rmon1.ay", "rmon1.az", "rmon1.ar", "M.f107", "M.f248", "M.f249", "M.f34c", "M.f34d", "M.f34e", "M.f37a", "M.tgt0", "M.tgt1", "M.tgt2", "M.tgt3", "M.tgt4", "M.tgt5", "M.tgt6", "M.tgt7", "relCount", "rel0.id", "rel0.f4", "rel0.f5", "rel0.f6", "rel1.id", "rel1.f4", "rel1.f5", "rel1.f6", "rel2.id", "rel2.f4", "rel2.f5", "rel2.f6", "tmpl0.d0", "tmpl0.e5", "tmpl0.e6", "tmpl0.f1bf", "tmpl0.f1c0", "tmpl0.f24a", "tmpl0.f25f", "tmpl0.k0.e7", "tmpl0.k0.cls", "tmpl0.k0.mask", "tmpl0.k1.e7", "tmpl0.k1.cls", "tmpl0.k1.mask", "tmpl0.k2.e7", "tmpl0.k2.cls", "tmpl0.k2.mask", "tmpl1.d0", "tmpl1.e5", "tmpl1.e6", "tmpl1.f1bf", "tmpl1.f1c0", "tmpl1.f24a", "tmpl1.f25f", "tmpl1.k0.e7", "tmpl1.k0.cls", "tmpl1.k0.mask", "tmpl1.k1.e7", "tmpl1.k1.cls", "tmpl1.k1.mask", "tmpl1.k2.e7", "tmpl1.k2.cls", "tmpl1.k2.mask", "ch0.slot", "ch0.hp", "ch0.st", "ch0.chance", "ch0.a35", "ch1.slot", "ch1.hp", "ch1.st", "ch1.chance", "ch1.a35", "ch2.slot", "ch2.hp", "ch2.st", "ch2.chance", "ch2.a35", "ch3.slot", "ch3.hp", "ch3.st", "ch3.chance", "ch3.a35", "ch4.slot", "ch4.hp", "ch4.st", "ch4.chance", "ch4.a35", "ch5.slot", "ch5.hp", "ch5.st", "ch5.chance", "ch5.a35", "ch6.slot", "ch6.hp", "ch6.st", "ch6.chance", "ch6.a35", "ch7.slot", "ch7.hp", "ch7.st", "ch7.chance", "ch7.a35", "r545_0", "r545_1", "r545_2", "r545_3", "r545_4", "r545_5", "r545_6", "r545_7", "r545_8", "r545_9", "r545_10", "r545_11", "r536_0", "r546_0", "r546_1", "r546_2", "r546_3", "r546_4", "r546_5", "r546_6", "r546_7", "r546_8", "r546_9", "r546_10", "r546_11", "r546_12", "r546_13", "r546_14", "r546_15", "r546_16", "r546_17", "r546_18", "r546_19", "r546_20", "r546_21", "r546_22", "r546_23", "r519_0", "r519_1", "r519_2", "r51a_0", "r51a_1", "r51a_2", "r51a_3", "r51a_4", "r51a_5", "r51a_6", "r51a_7", "r51a_8", "r51a_9", "r51a_10", "r51a_11", "r51a_12", "r51a_13", "r51a_14", "r51a_15", "r51a_16", "r51a_17", "r51a_18", "r51a_19", "r51a_20", "r51a_21", "r51a_22", "r51a_23", "W_0", "icpt.p0", "icpt.p1", "icpt.p2", "icpt.p3", "icpt.p4", "icpt.p5", "icpt.p6", "icpt.p7", "icpt.p8", "icpt.p9", "icpt.p10", "icpt.p11", "icpt.p12", "icpt.p13", "icpt.p14", "icpt.p15", "icpt.p16", "icpt.p17", "icpt.p18", "icpt.p19", "icpt.p20", "icpt.p21", "icpt.p22", "icpt.p23", "icpt.p24", "icpt.p25", "icpt.p26", "icpt.p27", "icpt.p28", "icpt.p29", "icpt.p30", "icpt.p31", "icpt.p32", "icpt.p33", "icpt.p34", "icpt.p35", "icpt.p36", "icpt.p37", "icpt.p38", "icpt.p39", "icpt.p40", "icpt.p41", "icpt.p42", "icpt.p43", "icpt.p44", "icpt.p45", "icpt.p46", "icpt.p47", "valid.p0", "valid.p1", "valid.p2", "valid.p3", "valid.p4", "valid.p5", "valid.p6", "valid.p7", "valid.p8", "valid.p9", "valid.p10", "valid.p11", "valid.p12", "valid.p13", "valid.p14", "valid.p15", "valid.p16", "valid.p17", "valid.p18", "valid.p19", "valid.p20", "valid.p21", "valid.p22", "valid.p23", "valid.p24", "valid.p25", "valid.p26", "valid.p27", "valid.p28", "avail.p0", "avail.p1", "avail.p2", "avail.p3", "avail.p4", "avail.p5", "avail.p6", "avail.p7", "avail.p8", "avail.p9", "avail.p10", "avail.p11", "avail.p12", "avail.p13", "avail.p14", "avail.p15", "avail.p16", "avail.p17", "avail.p18", "avail.p19", "avail.p20", "avail.p21", "avail.p22", "avail.p23", "avail.p24", "avail.p25", "avail.p26", "avail.p27", "avail.p28", "avail.p29", "avail.p30", "avail.p31", "avail.p32", "avail.p33", "avail.p34", "avail.p35", "avail.p36", "avail.p37", "avail.p38", "avail.p39", "avail.p40", "avail.p41", "avail.p42", "avail.p43", "avail.p44", "avail.p45", "avail.p46", "avail.p47", "avail.p48", "avail.p49", "avail.p50", "avail.p51", "avail.p52", "avail.p53", "avail.p54", "avail.p55", "avail.p56", "avail.p57", "avail.p58", "avail.p59", "avail.p60", "avail.p61", "avail.p62", "avail.p63", "avail.p64", "avail.p65", "avail.p66", "avail.p67", "avail.p68", "avail.p69", "avail.p70", "avail.p71", "avail.p72", "avail.p73", "avail.p74", "avail.p75", "avail.p76", "avail.p77", "avail.p78", "avail.p79", "avail.p80", "avail.p81", "avail.p82", "avail.p83", "avail.p84", "avail.p85", "avail.p86", "avail.p87", "avail.p88", "avail.p89", "avail.p90", "avail.p91", "avail.p92", "avail.p93", "avail.p94", "avail.p95", "avail.p96", "avail.p97", "avail.p98", "avail.p99", "avail.p100", "avail.p101", "avail.p102", "avail.p103", "avail.p104", "avail.p105", "avail.p106", "avail.p107", "avail.p108", "avail.p109", "avail.p110", "avail.p111", "avail.p112", "avail.p113", "avail.p114", "avail.p115", "avail.p116", "avail.p117", "avail.p118", "avail.p119", "avail.p120", "avail.p121", "avail.p122", "avail.p123", "avail.p124", "avail.p125", "avail.p126", "avail.p127", "avail.p128", "avail.p129", "avail.p130", "avail.p131", "avail.p132", "avail.p133", "avail.p134", "avail.p135", "avail.p136", "avail.p137", "avail.p138", "avail.p139", "avail.p140", "avail.p141", "avail.p142", "avail.p143", "avail.p144", "avail.p145", "avail.p146", "avail.p147", "avail.p148", "avail.p149", "avail.p150", "avail.p151", "avail.p152", "avail.p153", "avail.p154", "avail.p155", "avail.p156", "avail.p157", "avail.p158", "avail.p159", "avail.p160", "avail.p161", "avail.p162", "avail.p163", "avail.p164", "avail.p165", "avail.p166", "pen.p0", "pen.p1", "pen.p2", "pen.p3", "pen.p4", "pen.p5", "pen.p6", "pen.p7", "pen.p8", "pen.p9", "pen.p10", "pen.p11", "pen.p12", "pen.p13", "pen.p14", "pen.p15", "pen.p16", "pen.p17", "pen.p18", "pen.p19", "pen.p20", "pen.p21", "pen.p22", "pen.p23", "pen.p24", "pen.p25", "pen.p26", "pen.p27", "pen.p28", "pen.p29", "pen.p30", "pen.p31", "pen.p32", "pen.p33", "pen.p34", "pen.p35", "pen.p36", "pen.p37", "pen.p38"}
}

func macNames_out() []string {
	return []string{"M.2e1", "M.tgt0", "M.tgt1", "M.tgt2", "M.tgt3", "M.tgt4", "M.tgt5", "M.tgt6", "M.tgt7", "c.list", "old.data", "n_roll", "n_icpt", "n_valid", "n_disp", "n_avail", "n_pen", "n_grow", "n_log", "roll2_0", "roll2_1", "roll2_2", "roll2_3", "roll2_4", "roll2_5", "roll2_6", "roll2_7", "roll2_8", "roll2_9", "roll2_10", "roll2_11", "roll2_12", "roll2_13", "roll2_14", "roll2_15", "roll2_16", "roll2_17", "roll2_18", "roll2_19", "roll2_20", "roll2_21", "roll2_22", "roll2_23", "roll2_24", "roll2_25", "roll2_26", "roll2_27", "roll2_28", "roll2_29", "roll2_30", "roll2_31", "roll2_32", "roll2_33", "roll2_34", "roll2_35", "roll2_36", "roll2_37", "roll2_38", "roll2_39", "roll2_40", "roll2_41", "roll2_42", "roll2_43", "roll2_44", "roll2_45", "roll2_46", "roll2_47", "roll2_48", "roll2_49", "roll2_50", "roll2_51", "roll2_52", "roll2_53", "roll2_54", "roll2_55", "roll2_56", "roll2_57", "roll2_58", "roll2_59", "roll2_60", "roll2_61", "roll2_62", "roll2_63", "roll2_64", "roll2_65", "roll2_66", "roll2_67", "roll2_68", "roll2_69", "roll2_70", "roll2_71", "roll2_72", "roll2_73", "roll2_74", "roll2_75", "roll2_76", "roll2_77", "roll2_78", "roll2_79", "roll2_80", "roll2_81", "roll2_82", "roll2_83", "roll2_84", "roll2_85", "roll2_86", "roll2_87", "roll2_88", "roll2_89", "roll2_90", "roll2_91", "roll2_92", "roll2_93", "roll2_94", "roll2_95", "roll3_0", "roll3_1", "roll3_2", "roll3_3", "roll3_4", "roll3_5", "roll3_6", "roll3_7", "roll3_8", "roll3_9", "roll3_10", "roll3_11", "roll3_12", "roll3_13", "roll3_14", "roll3_15", "roll3_16", "roll3_17", "roll3_18", "roll3_19", "roll3_20", "roll3_21", "roll3_22", "roll3_23", "roll3_24", "roll3_25", "roll3_26", "roll3_27", "roll3_28", "roll3_29", "roll3_30", "roll3_31", "roll3_32", "roll3_33", "roll3_34", "roll3_35", "roll3_36", "roll3_37", "roll3_38", "roll3_39", "roll3_40", "roll3_41", "roll3_42", "roll3_43", "roll3_44", "roll3_45", "roll3_46", "roll3_47", "roll3_48", "roll3_49", "roll3_50", "roll3_51", "roll3_52", "roll3_53", "roll3_54", "roll3_55", "roll3_56", "roll3_57", "roll3_58", "roll3_59", "roll3_60", "roll3_61", "roll3_62", "roll3_63", "roll3_64", "roll3_65", "roll3_66", "roll3_67", "roll3_68", "roll3_69", "roll3_70", "roll3_71", "roll3_72", "roll3_73", "roll3_74", "roll3_75", "roll3_76", "roll3_77", "roll3_78", "roll3_79", "roll3_80", "roll3_81", "roll3_82", "roll3_83", "roll3_84", "roll3_85", "roll3_86", "roll3_87", "roll3_88", "roll3_89", "roll3_90", "roll3_91", "roll3_92", "roll3_93", "roll3_94", "roll3_95", "roll4_0", "roll4_1", "roll4_2", "roll4_3", "roll4_4", "roll4_5", "roll4_6", "roll4_7", "roll4_8", "roll4_9", "roll4_10", "roll4_11", "roll4_12", "roll4_13", "roll4_14", "roll4_15", "roll4_16", "roll4_17", "roll4_18", "roll4_19", "roll4_20", "roll4_21", "roll4_22", "roll4_23", "roll4_24", "roll4_25", "roll4_26", "roll4_27", "roll4_28", "roll4_29", "roll4_30", "roll4_31", "roll4_32", "roll4_33", "roll4_34", "roll4_35", "roll4_36", "roll4_37", "roll4_38", "roll4_39", "roll4_40", "roll4_41", "roll4_42", "roll4_43", "roll4_44", "roll4_45", "roll4_46", "roll4_47", "roll4_48", "roll4_49", "roll4_50", "roll4_51", "roll4_52", "roll4_53", "roll4_54", "roll4_55", "roll4_56", "roll4_57", "roll4_58", "roll4_59", "roll4_60", "roll4_61", "roll4_62", "roll4_63", "roll4_64", "roll4_65", "roll4_66", "roll4_67", "roll4_68", "roll4_69", "roll4_70", "roll4_71", "roll4_72", "roll4_73", "roll4_74", "roll4_75", "roll4_76", "roll4_77", "roll4_78", "roll4_79", "roll4_80", "roll4_81", "roll4_82", "roll4_83", "roll4_84", "roll4_85", "roll4_86", "roll4_87", "roll4_88", "roll4_89", "roll4_90", "roll4_91", "roll4_92", "roll4_93", "roll4_94", "roll4_95", "roll5_0", "roll5_1", "roll5_2", "roll5_3", "roll5_4", "roll5_5", "roll5_6", "roll5_7", "roll5_8", "roll5_9", "roll5_10", "roll5_11", "roll5_12", "roll5_13", "roll5_14", "roll5_15", "roll5_16", "roll5_17", "roll5_18", "roll5_19", "roll5_20", "roll5_21", "roll5_22", "roll5_23", "roll5_24", "roll5_25", "roll5_26", "roll5_27", "roll5_28", "roll5_29", "roll5_30", "roll5_31", "roll5_32", "roll5_33", "roll5_34", "roll5_35", "roll5_36", "roll5_37", "roll5_38", "roll5_39", "roll5_40", "roll5_41", "roll5_42", "roll5_43", "roll5_44", "roll5_45", "roll5_46", "roll5_47", "roll5_48", "roll5_49", "roll5_50", "roll5_51", "roll5_52", "roll5_53", "roll5_54", "roll5_55", "roll5_56", "roll5_57", "roll5_58", "roll5_59", "roll5_60", "roll5_61", "roll5_62", "roll5_63", "roll5_64", "roll5_65", "roll5_66", "roll5_67", "roll5_68", "roll5_69", "roll5_70", "roll5_71", "roll5_72", "roll5_73", "roll5_74", "roll5_75", "roll5_76", "roll5_77", "roll5_78", "roll5_79", "roll5_80", "roll5_81", "roll5_82", "roll5_83", "roll5_84", "roll5_85", "roll5_86", "roll5_87", "roll5_88", "roll5_89", "roll5_90", "roll5_91", "roll5_92", "roll5_93", "roll5_94", "roll5_95", "roll6_0", "roll6_1", "roll6_2", "roll6_3", "roll6_4", "roll6_5", "roll6_6", "roll6_7", "roll6_8", "roll6_9", "roll6_10", "roll6_11", "roll6_12", "roll6_13", "roll6_14", "roll6_15", "roll6_16", "roll6_17", "roll6_18", "roll6_19", "roll6_20", "roll6_21", "roll6_22", "roll6_23", "roll6_24", "roll6_25", "roll6_26", "roll6_27", "roll6_28", "roll6_29", "roll6_30", "roll6_31", "roll6_32", "roll6_33", "roll6_34", "roll6_35", "roll6_36", "roll6_37", "roll6_38", "roll6_39", "roll6_40", "roll6_41", "roll6_42", "roll6_43", "roll6_44", "roll6_45", "roll6_46", "roll6_47", "roll6_48", "roll6_49", "roll6_50", "roll6_51", "roll6_52", "roll6_53", "roll6_54", "roll6_55", "roll6_56", "roll6_57", "roll6_58", "roll6_59", "roll6_60", "roll6_61", "roll6_62", "roll6_63", "roll6_64", "roll6_65", "roll6_66", "roll6_67", "roll6_68", "roll6_69", "roll6_70", "roll6_71", "roll6_72", "roll6_73", "roll6_74", "roll6_75", "roll6_76", "roll6_77", "roll6_78", "roll6_79", "roll6_80", "roll6_81", "roll6_82", "roll6_83", "roll6_84", "roll6_85", "roll6_86", "roll6_87", "roll6_88", "roll6_89", "roll6_90", "roll6_91", "roll6_92", "roll6_93", "roll6_94", "roll6_95", "disp_c0", "disp_c1", "disp_c2", "disp_c3", "disp_c4", "disp_c5", "disp_c6", "disp_c7", "disp_c8", "disp_c9", "disp_c10", "disp_c11", "disp_c12", "disp_c13", "disp_c14", "disp_c15", "disp_c16", "disp_c17", "disp_c18", "disp_c19", "disp_c20", "disp_c21", "disp_c22", "disp_c23", "avail_k0", "avail_k1", "avail_k2", "avail_f0", "avail_f1", "avail_f2", "pen_c0", "pen_c1", "pen_c2", "pen_c3", "pen_c4", "pen_c5", "pen_c6", "pen_c7", "pen_c8", "pen_c9", "pen_c10", "pen_c11", "pen_c12", "pen_c13", "pen_c14", "pen_c15", "pen_c16", "pen_c17", "pen_c18", "pen_c19", "pen_c20", "pen_c21", "pen_c22", "pen_c23", "grow_c0", "grow_c1", "grow_c2", "grow_c3", "grow_c4", "grow_c5", "grow_c6", "grow_c7", "log_fmt0", "log_fmt1", "log_fmt2", "log_t0", "log_t1", "log_t2"}
}
