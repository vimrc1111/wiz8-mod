package calc

import (
	"math"
	"math/big"
)

// ---- 0x531540: the monster's action choice ----
//
// Picks the monster's action (m+0x2e1) for one turn: the A/B early outs, the
// hurt-and-flee chance roll, the self-heal / ranged / spell branches, and the
// closing busy check. It runs over the same memory image as the emulator case
// (mstWorld). The address-stubbed D4 callees (0x5323f0, 0x5326f0, 0x533260,
// 0x533320, 0x519c00, 0x500330, 0x4fd030) read their return and written words
// from case inputs and record every call; the registered-out callees (0x4d9080,
// 0x5182e0) are replayed in both runs. Emu replays the stubs; F calls the real Go
// nodes on default worlds of their own (mchPark*), so F and Emu differ wherever
// the case's stub inputs differ from what those worlds compute.

const (
	mchLog   = iota // 0x5182e0 (registered out): message, args 2 and 3 recorded
	mchUI           // 0x4d9080 (registered out)
	mchAvail        // 0x519c00
	mchCand         // 0x533320
	mchTgt          // 0x5326f0
	mchSpell        // 0x533260
	mchPick         // 0x5323f0
	mchPower        // 0x500330
	mchArea         // 0x4fd030
	mchStubs
)

// 0x61cd48 / 0x61cdfc / 0x61d2e0 / 0x61d304 / 0x61d328 are the message strings the
// logger gets; the scratch words stand for the callee-owned stack lists.
const (
	mchLocalList = 0x7f001000 // 0x532550 / 0x534a40: the 0x533320 result list
	mchAreaList  = 0x7f001100 // 0x5353e0: the 0x4fd030 list
)

type mchSpec struct {
	mstSpec
	stub [mchStubs]macStub
	// First word of the inputs a stub writes: 0x533320 result count (candW),
	// 0x5326f0 eight target words (tgtW), 0x5323f0 thirteen words (pickW) and
	// 0x4fd030 result (areaW).
	candW, tgtW, pickW, areaW uint32
}

type mchCallees struct {
	pick  func(w *mchWorld, m, a, b, flag uint32) bool
	tgt   func(w *mchWorld, m, spell uint32) bool
	spell func(w *mchWorld, m, t uint32) uint32
	cand  func(w *mchWorld, m, spell, list uint32)
	avail func(w *mchWorld, m, i, a5 uint32) bool
	power func(w *mchWorld, m, t, spell uint32) uint32
	area  func(w *mchWorld, spell, power, list uint32)
}

type mchWorld struct {
	*mstWorld
	sp *mchSpec
	c  *mchCallees
}

// call records one stubbed call (arguments are 1-based stack positions) and
// returns its sequence number; more calls than the case records panic.
func (w *mchWorld) call(k int, args ...uint32) uint32 {
	s := &w.sp.stub[k]
	n := w.u32(s.cnt)
	for _, r := range s.rec {
		if n >= r[3] {
			panic("monster action choice: more stubbed calls than the case records")
		}
		w.put(r[2]+n*r[1], int(r[1]), uint64(args[r[0]-1]))
	}
	w.put(s.cnt, 4, uint64(n+1))
	return n
}

func (w *mchWorld) ret(k int, n uint32) uint32 {
	s := &w.sp.stub[k]
	if s.retN == 0 {
		return w.u32(s.ret)
	}
	if n >= s.retN {
		panic("monster action choice: more stubbed calls than the case returns")
	}
	return w.u32(s.ret + 4*n)
}

// word is input word i of a stub-write block.
func (w *mchWorld) word(at uint32, i uint32) uint32 { return w.u32(at + 4*i) }

var mchEmu = mchCallees{
	pick: func(w *mchWorld, m, a, b, flag uint32) bool {
		n := w.call(mchPick, m, a, b, flag)
		cmb := w.u32(m + 0x10)
		w.put(m+0x2e1, 4, uint64(w.word(w.sp.pickW, 0)))
		w.put(m+0x2e5, 4, uint64(w.word(w.sp.pickW, 1)))
		for i := uint32(0); i < 8; i++ {
			w.put(m+0x2ba+4*i, 4, uint64(w.word(w.sp.pickW, 2+i)))
		}
		w.put(cmb+5, 4, uint64(w.word(w.sp.pickW, 10)))
		w.put(cmb+9, 4, uint64(w.word(w.sp.pickW, 11)))
		w.put(cmb+0x11, 4, uint64(w.word(w.sp.pickW, 12)))
		return byte(w.ret(mchPick, n)) != 0
	},
	tgt: func(w *mchWorld, m, spell uint32) bool {
		n := w.call(mchTgt, m, spell)
		for i := uint32(0); i < 8; i++ {
			w.put(m+0x2ba+4*i, 4, uint64(w.word(w.sp.tgtW, i)))
		}
		return byte(w.ret(mchTgt, n)) != 0
	},
	spell: func(w *mchWorld, m, t uint32) uint32 { return w.ret(mchSpell, w.call(mchSpell, m, t)) },
	cand: func(w *mchWorld, m, spell, list uint32) {
		w.call(mchCand, m, spell, list)
		w.put(list+4, 4, uint64(w.u32(w.sp.candW)))
	},
	avail: func(w *mchWorld, m, i, a5 uint32) bool {
		return byte(w.ret(mchAvail, w.call(mchAvail, m, i, a5))) != 0
	},
	power: func(w *mchWorld, m, t, spell uint32) uint32 { return w.ret(mchPower, w.call(mchPower, m, t, spell)) },
	area: func(w *mchWorld, spell, power, list uint32) {
		w.call(mchArea, spell, power)
		w.put(list+4, 4, uint64(w.u32(w.sp.areaW)))
	},
}

// tmpl is 0x4e5720: the template cache is a direct array at 0x6840c7. A miss
// would load the template file; the fixture preloads ids 0, 1 and 0x224.
func (w *mchWorld) tmpl(m uint32) uint32 {
	id := w.u32(m + 8)
	if id >= 0x3e8 {
		panic("monster action choice: template id past 0x3e7 (0x4e5749 asserts)")
	}
	if id != 0 && id != 1 && id != 0x224 {
		panic("monster action choice: template cache miss is outside the fixture (file load)")
	}
	return w.u32(0x6840c7 + 4*id)
}

func mchF32(bits uint32) *big.Float {
	return new(big.Float).SetPrec(64).SetFloat64(float64(math.Float32frombits(bits)))
}

// mchRound32 is fstp dword: the x87 value stored as a float32 (nearest even).
func mchRound32(x *big.Float) *big.Float {
	f, _ := x.Float32()
	return new(big.Float).SetPrec(64).SetFloat64(float64(f))
}

// reach is 0x51a9a0: the range length for reach class p, times 500.0.
func (w *mchWorld) reach(p int32) *big.Float {
	n := new(big.Float).SetPrec(64).SetInt64(w.mstLen(0x51aa10, mstLenSelf, uint32(p+1)))
	return n.Mul(n, monsterAttackTargetExt(interceptData.scale))
}

// rangeOf is 0x51a800(template, 1): the largest range byte of the active attack
// entries whose range is at most 1, -1 for none.
func (w *mchWorld) rangeOf(t uint32) int32 {
	best := int32(-1)
	for i := uint32(0); i < 3; i++ {
		if w.u8(t+0xe7+0x22*i) == 0 {
			continue
		}
		if r := int32(w.u8(t + 0xea + 0x22*i)); r <= 1 && r > best {
			best = r
		}
	}
	return best
}

// nearest is 0x51ac30(m, buf, 2): the smallest reachable distance, as a float32.
func (w *mchWorld) nearest(m uint32) *big.Float {
	best := mchF32(0x497423f0)
	found := false
	if w.u8(m+0x16) == 1 && w.u8(m+0x348+5) != 0 { // 0x505dd0 mode 2 on the record at m+0x348
		d := w.mstDistParty(w.u32(m + 0xc))
		if d.Cmp(mchF32(w.u32(0x5ed868))) < 0 {
			best, found = mchRound32(d), true
		}
	}
	if found && w.u8(w.u32(m+0x10)+0x152) != 0 {
		return best
	}
	l := w.u32(0x683fad)
	for i := uint32(0); i < w.listCount(l); i++ {
		o := w.monsterRecord(i)
		if o == m || w.u8(o+0x14) == 0 || w.u32(o+0x2b) == 0 || w.u8(o+0x15) == 0 {
			continue
		}
		if w.mstStance(m, o) != 1 {
			continue
		}
		e := w.mstRelation(m, o)
		if e == 0 || w.u8(e+5) == 0 { // 0x505dd0 mode 2
			continue
		}
		if d := w.mstDistMon(w.u32(m+0xc), w.u32(o+0xc)); d.Cmp(best) < 0 {
			best = mchRound32(d)
		}
	}
	return best
}

// chance is 0x531c00: the percent chance that the monster gives up its turn.
func (w *mchWorld) chance(m, t uint32) uint32 {
	hp, mx := w.u32(m+0x2b), w.u32(m+0x27)
	if mx == 0 {
		panic("monster action choice: maximum hit points 0 (integer divide fault)")
	}
	pct := uint32(uint64(hp)*100) / mx
	if w.u8(m+0x15) == 0 {
		panic("monster action choice: inactive monster (0x531c00 asserts)")
	}
	cmb := w.u32(m + 0x10)
	if w.u8(t+0xd0)&1 != 0 && w.u8(m+0x16) != 1 && pct <= 0x21 {
		return 100
	}
	if w.u8(cmb+0x14b) != 0 {
		if w.reach(1).Cmp(w.nearest(m)) < 0 {
			return 100
		}
		w.put(cmb+0x14b, 1, 0)
		return 0
	}
	if e := w.u8(t + 0xe2); e != 0 {
		return uint32(e)
	}
	if w.u8(t+0x1c0) != 0 {
		return 0
	}
	return 100
}

// spot6 is 0x556050.
func (w *mchWorld) hits(x byte) bool {
	idx := uint32(x) - 5
	if idx > 0x18 {
		return false
	}
	switch w.u32(0x556074 + 4*uint32(w.u8(0x55607c+idx))) {
	case 0x55606b:
		return true
	case 0x55606e:
		return false
	}
	panic("monster action choice: unknown 0x556050 arm")
}

func (w *mchWorld) logMsg(msg, t uint32) { w.call(mchLog, 0, msg, t) }

// blocked is 0x53d4b0: 0 when attack i has a target, 3 when none, 1 when skipped.
func (w *mchWorld) attack(m, t, i, a5 uint32) uint32 {
	if i >= 3 {
		panic("monster action choice: attack entry past 2 (assert)")
	}
	if w.u8(t+0xe7+i*0x22) == 0 {
		return 1
	}
	if w.u8(t+0xfc+i*0x22) == 0 && w.u8(t+0xfd+i*0x22) == 0 {
		w.logMsg(0x61d328, t)
		return 1
	}
	if w.u8(t+0xe5) == 0 {
		w.logMsg(0x61d304, t)
		return 1
	}
	if w.u8(t+0xe6) == 0 {
		w.logMsg(0x61d2e0, t)
		return 1
	}
	state := w.u32(m + 0x2e1)
	w.put(m+0x2e1, 4, 0)
	ok := w.c.avail(w, m, i, a5)
	w.put(m+0x2e1, 4, uint64(state))
	if ok {
		return 0
	}
	return 3
}

// ready is 0x53d450.
func (w *mchWorld) ready(m, t, a3 uint32) byte {
	if w.u8(m+0x24e) != 0 {
		return 1
	}
	var best uint32
	for i := uint32(0); i < 3; i++ {
		r := w.attack(m, t, i, a3)
		if r == 0 {
			return 0
		}
		if r > best {
			best = r
		}
	}
	return byte(best)
}

// canCast is 0x532550 with the flag set.
func (w *mchWorld) canCast(m, spell uint32) bool {
	if spell == 0 {
		return false
	}
	rec := w.spellRecord(spell)
	if w.u8(rec+0x126) == 0 || w.spellBlockedByKind(m, spell) {
		return false
	}
	if byte(w.ret(mchUI, w.call(mchUI, m, 4, 0))) != 0 {
		return false
	}
	if spell == 0x3c && w.u32(m+0x344) != 0xffffffff {
		return false
	}
	if w.u32(rec+0x133) == 0 && w.u8(0x652da4) != 0 {
		return false
	}
	w.c.cand(w, m, spell, mchLocalList)
	return w.u32(mchLocalList+4) != 0
}

// aimed is 0x534a40(m, t, a3).
func (w *mchWorld) aimed(m, t, a3 uint32) bool {
	if w.u8(t+0xe1) == 0 || w.u8(t+0xe3) == 0 {
		return false
	}
	if !w.animFlag(w.u32(m+0xc), 0x12) {
		if w.u8(0x68d525) == 0 {
			w.logMsg(0x61cdfc, t)
			w.put(0x68d525, 1, 1)
		}
		return false
	}
	cmb := w.u32(m + 0x10)
	if w.u8(0x683f94) != 0 && w.u8(cmb+0x146) > 0 {
		return false
	}
	if w.u32(m+0x33) < w.u32(m+0x2f)/10 {
		return false
	}
	if w.u32(m+0x77) != 0 && w.hits(w.u8(t+0xe3)) {
		return false
	}
	if w.u32(0x61eefc+8*uint32(w.u8(t+0xe3))) == 6 {
		if w.reach(2).Cmp(mchRound32(w.mstDistParty(w.u32(m+0xc)))) < 0 || w.u8(m+0x16) == 2 {
			return false
		}
		return byte(a3) == 0
	}
	w.put(mchLocalList+4, 4, 0)
	w.c.cand(w, m, 0x77, mchLocalList)
	return w.u32(mchLocalList+4) != 0
}

// reaches is 0x5353e0(m, spell, tgt).
func (w *mchWorld) reaches(m, spell, tgt uint32) bool {
	t := w.tmpl(m)
	power := w.c.power(w, m, t, spell)
	w.c.area(w, spell, power, mchAreaList)
	return int32(w.u32(mchAreaList+4)) > 0
}

// busy is 0x535150.
func (w *mchWorld) busy(m uint32) bool {
	switch w.u32(m + 0x2e1) {
	case 0:
		return w.u32(m+0x2ba) == 1
	case 2:
		spell := w.u32(m + 0x2e5)
		if !w.mstSpellFits(spell) {
			return false
		}
		k := w.u32(m+0x2ba) - 1
		if k > 5 {
			return false
		}
		switch w.u32(0x5351e4 + 4*k) {
		case 0x53519a:
			return true
		case 0x5351de:
			return false
		case 0x5351cb:
			return w.reaches(m, spell, m+0x2ba)
		}
		panic("monster action choice: unknown 0x535150 arm")
	case 3:
		if w.u32(0x61eefc+8*uint32(w.u8(w.tmpl(m)+0xe3))) == 6 {
			return false
		}
		return w.reaches(m, 0x77, m+0x2ba)
	}
	return false
}

// group is 0x511cb0.
func (w *mchWorld) group(id uint32) bool { return w.u8(w.mstGroupAt(w.mstGroupHandle(id))+0xc8) != 0 }

// idle writes the "no action" words (0x5318fb and the unarmed branch).
func (w *mchWorld) idle(m uint32) {
	w.put(m+0x2e1, 4, 0xffffffff)
	cmb := w.u32(m + 0x10)
	w.put(cmb, 4, 0)
	w.put(cmb+4, 1, 1)
}

// ratio is the hurt test: hit points over the (signed) maximum against 0.95f.
func (w *mchWorld) ratio(m, t uint32) uint32 {
	mx := int32(w.u32(m + 0x27))
	if mx == 0 {
		panic("monster action choice: maximum hit points 0 (integer divide fault)")
	}
	r := new(big.Float).SetPrec(64).Quo(new(big.Float).SetPrec(64).SetInt64(int64(w.u32(m+0x2b))), new(big.Float).SetPrec(64).SetInt64(int64(mx)))
	if r.Cmp(mchF32(w.u32(0x5ee770))) <= 0 && w.u8(t+0x1b9) == 0 {
		return 7
	}
	return 5
}

func (w *mchWorld) end(m uint32) {
	if !w.busy(m) && w.group(w.u32(m+4)) {
		w.put(m+0x2e1, 4, 6)
	}
}

func (w *mchWorld) run(m uint32) {
	if w.u32(m+0x107) >= 0xf {
		w.idle(m)
		return
	}
	t := w.tmpl(m)
	if w.u32(m+8) == 0x224 {
		w.put(m+0x2e1, 4, 1)
		return
	}
	pct := w.chance(m, t)
	if uint32(w.r.R(100)) < pct {
		w.put(m+0x2e1, 4, 4)
		w.end(m)
		return
	}
	if w.u8(m+0x16) == 0 {
		if w.u8(t+0x249) == 0 {
			w.put(m+0x2e1, 4, 6)
		} else {
			w.idle(m)
		}
		w.end(m)
		return
	}
	if w.u8(t+0x1b9) == 0 {
		if p := w.rangeOf(t); p != -1 && (w.u32(m+0x87) == 0 || w.u8(t+0xcb) == 0xc) {
			if w.reach(p).Cmp(mchRound32(w.nearest(m))) < 0 {
				w.put(m+0x2e1, 4, uint64(w.ratio(m, t)))
				w.end(m)
				return
			}
		}
	}
	s := w.ready(m, t, 0)
	if e1 := uint32(w.u8(t + 0xe1)); e1 != 0 {
		if s != 0 {
			e1 = 100
		}
		if w.aimed(m, t, 0) && uint32(w.r.R(100)) < e1 {
			w.put(m+0x2e1, 4, 3)
			if w.u32(0x61eefc+8*uint32(w.u8(t+0xe3))) == 6 {
				tg := m + 0x2ba
				for i := uint32(0); i < 8; i++ { // 0x536170
					w.put(tg+4*i, 4, 0)
				}
				for i := uint32(1); i < 4; i++ {
					w.put(tg+4*i, 4, 0xffffffff)
				}
				w.put(tg, 4, 6)
				anim := w.u32(m+0xc) + 0x118
				w.put(m+0x2ca, 4, uint64(w.u32(anim)))
				w.put(m+0x2ce, 4, uint64(w.u32(anim+4)))
				w.put(m+0x2d2, 4, uint64(w.u32(anim+8)))
				w.end(m)
				return
			}
			if !w.c.tgt(w, m, 0x77) {
				panic("monster action choice: 0x5326f0 found no target (0x531784 asserts)")
			}
			w.end(m)
			return
		}
	}
	if e0 := uint32(w.u8(t + 0xe0)); e0 != 0 {
		if s != 0 {
			e0 = 100
		}
		if !w.animFlag(w.u32(m+0xc), 0x19) {
			if w.u8(0x68d524) == 0 {
				w.logMsg(0x61cd48, t)
				w.put(0x68d524, 1, 1)
			}
		} else {
			for i := uint32(0); i < 10; i++ {
				if !w.canCast(m, uint32(w.u8(t+0x14d+i))) {
					continue
				}
				if uint32(w.r.R(100)) < e0 {
					w.put(m+0x2e1, 4, 2)
					sp := w.c.spell(w, m, t)
					w.put(m+0x2e5, 4, uint64(sp))
					if !w.c.tgt(w, m, sp) {
						panic("monster action choice: 0x5326f0 found no target (0x531854 asserts)")
					}
					w.end(m)
					return
				}
				break
			}
		}
	}
	switch s {
	case 0:
		if !w.c.pick(w, m, 0, 0, 1) {
			w.put(m+0x2e1, 4, 1)
		}
	case 3:
		if w.u32(m+0x87) != 0 && w.u8(t+0xcb) != 0xc {
			w.put(m+0x2e1, 4, 6)
		} else {
			w.put(m+0x2e1, 4, uint64(w.ratio(m, w.tmpl(m))))
		}
	default:
		w.put(m+0x2e1, 4, 6)
	}
	w.end(m)
}

func mchNode(spec *mchSpec, c *mchCallees) func([]int64, Rand) []int64 {
	return func(in []int64, r Rand) []int64 {
		if len(in) != len(spec.in) {
			panic("monster action choice: wrong input count")
		}
		w := &mchWorld{mstWorld: mstBuild(&spec.mstSpec, in), sp: spec, c: c}
		w.r = r
		w.run(w.u32(mstStackBase + 4))
		out := make([]int64, len(spec.out))
		for i, f := range spec.out {
			out[i] = w.read(f, in)
		}
		return out
	}
}

func monsterActionChoiceNode(in []int64, r Rand) []int64 {
	return mchNode(&mchSpec_main, &mchReal)(in, r)
}

func init() {
	Nodes["00531540"] = Node{Addr: "00531540", In: mchNames_in(), Out: mchNames_out(),
		F: monsterActionChoiceNode, Emu: mchNode(&mchSpec_main, &mchEmu)}
}

var mchSpec_main = mchSpec{
	mstSpec: mstSpec{
		set: []mstSet{
			{0x7f000004, 4, 0x2001cb00},
			{0x683fad, 4, 0x2001e300},
			{0x683fb1, 4, 0x2001e3a0},
			{0x683fb9, 4, 0x2001e5c0},
			{0x683fbd, 4, 0x2001e5c0},
			{0x65a0f8, 4, 0x2001e5d0},
			{0x659c0c, 4, 0x2001e6d0},
			{0x65be1c, 4, 0x20000500},
			{0x6840c7, 4, 0x2001c500},
			{0x6840cb, 4, 0x2001c800},
			{0x684957, 4, 0x2001c500},
			{0x2001e300, 4, 0x2001e310},
			{0x2001e310, 4, 0x2001cb00},
			{0x2001e314, 4, 0x2001d100},
			{0x2001e318, 4, 0x2001d500},
			{0x2001e320, 4, 0x2001e330},
			{0x2001e330, 4, 0x2001e340},
			{0x2001e334, 4, 0x2001e360},
			{0x2001e338, 4, 0x2001e380},
			{0x2001e340, 4, 0x65},
			{0x2001e360, 4, 0x66},
			{0x2001e380, 4, 0x67},
			{0x2001e3a0, 4, 0x2001e3b0},
			{0x2001e3a8, 4, 0x2},
			{0x2001e3b0, 4, 0x2001e3c0},
			{0x2001e3b4, 4, 0x2001e4c0},
			{0x2001e3c0, 4, 0xa},
			{0x2001e4c0, 4, 0xb},
			{0x2001cb00, 4, 0x64},
			{0x2001cb0c, 4, 0x2001d900},
			{0x2001d100, 4, 0x65},
			{0x2001d10c, 4, 0x2001db00},
			{0x2001d500, 4, 0x66},
			{0x2001d50c, 4, 0x2001dd00},
			{0x2001cb10, 4, 0x2001cf00},
			{0x2001cb14, 1, 0x1},
			{0x2001cdb6, 4, 0x2001e320},
			{0x2001d114, 1, 0x1},
			{0x2001d514, 1, 0x1},
			{0x2001d900, 4, 0x5ed22c},
			{0x2001dad8, 4, 0x2001df00},
			{0x2001db00, 4, 0x5ed22c},
			{0x2001dcd8, 4, 0x2001df00},
			{0x2001dd00, 4, 0x5ed22c},
			{0x2001ded8, 4, 0x2001df00},
			{0x2001dfb0, 4, 0x77},
			{0x2001dfc0, 4, 0x77},
			{0x2001dfd0, 4, 0x77},
			{0x2001dfe0, 4, 0x77},
			{0x2001dff0, 4, 0x77},
			{0x2001e000, 4, 0x77},
			{0x2001e010, 4, 0x77},
			{0x2001e020, 4, 0x77},
			{0x2001e030, 4, 0x77},
			{0x2001e040, 4, 0x77},
			{0x2001e050, 4, 0x77},
			{0x2001e060, 4, 0x77},
			{0x2001e070, 4, 0x77},
			{0x2001e080, 4, 0x77},
			{0x2001e090, 4, 0x77},
			{0x2001e0a0, 4, 0x77},
			{0x2001e0b0, 4, 0x77},
			{0x2001e0c0, 4, 0x77},
			{0x2001e0e0, 4, 0x77},
			{0x2001e0f0, 4, 0x77},
			{0x2001e100, 4, 0x77},
			{0x2001e110, 4, 0x77},
			{0x2001e120, 4, 0x77},
			{0x2001e130, 4, 0x77},
			{0x2001e150, 4, 0x77},
			{0x2001c5c0, 4, 0x3c3c3c3c},
			{0x2001c5c4, 4, 0x3c3c3c3c},
			{0x2001c5c8, 4, 0x3c3c3c3c},
			{0x2001c5cc, 4, 0x3c3c3c3c},
			{0x2001c5d0, 4, 0x3c3c3c3c},
			{0x2001c5d4, 4, 0x3c3c3c3c},
			{0x2001c5d8, 4, 0x3c3c3c3c},
			{0x2001c5dc, 4, 0x3c3c3c3c},
			{0x2001c5e0, 4, 0x3c3c3c3c},
			{0x2001c5e4, 4, 0x3c3c3c3c},
			{0x2001c5e8, 4, 0x3c3c3c3c},
			{0x2001c5ec, 4, 0x3c3c3c3c},
			{0x2001c5f0, 4, 0x3c3c3c3c},
			{0x2001c5f4, 4, 0x3c3c3c3c},
			{0x2001c5f8, 4, 0x3c3c3c3c},
			{0x2001c5fc, 4, 0x3c3c3c3c},
			{0x2001c600, 4, 0x3c3c3c3c},
			{0x2001c604, 4, 0x3c3c3c3c},
			{0x2001c608, 4, 0x3c3c3c3c},
			{0x2001c60c, 4, 0x3c3c3c3c},
			{0x2001c610, 4, 0x3c3c3c3c},
			{0x2001c614, 4, 0x3c3c3c3c},
			{0x2001c618, 4, 0x3c3c3c3c},
			{0x2001c61c, 4, 0x3c3c3c3c},
			{0x2001c620, 4, 0x3c3c3c3c},
			{0x2001c624, 4, 0x3c3c3c3c},
			{0x2001c628, 4, 0x3c3c3c3c},
			{0x2001c62c, 4, 0x3c3c3c3c},
			{0x2001c630, 4, 0x3c3c3c3c},
			{0x2001c634, 4, 0x3c3c3c3c},
			{0x2001c638, 4, 0x3c3c3c3c},
			{0x2001c63c, 4, 0x3c3c3c3c},
			{0x2001c640, 4, 0x3c3c3c3c},
			{0x2001c644, 4, 0x3c3c3c3c},
			{0x2001c648, 4, 0x3c3c3c3c},
			{0x2001c64c, 4, 0x3c3c3c3c},
			{0x2001c650, 4, 0x3c3c3c3c},
			{0x2001c654, 4, 0x3c3c3c3c},
			{0x2001c658, 4, 0x3c3c3c3c},
			{0x2001c65c, 4, 0x3c3c3c3c},
			{0x2001c660, 4, 0x3c3c3c3c},
			{0x2001c664, 4, 0x3c3c3c3c},
			{0x2001c668, 4, 0x3c3c3c3c},
			{0x2001c66c, 4, 0x3c3c3c3c},
			{0x2001c670, 4, 0x3c3c3c3c},
			{0x2001c674, 4, 0x3c3c3c3c},
			{0x2001c678, 4, 0x3c3c3c3c},
			{0x2001c67c, 4, 0x3c3c3c3c},
			{0x2001c680, 4, 0x3c3c3c3c},
			{0x2001c684, 4, 0x3c3c3c3c},
			{0x2001c688, 4, 0x3c3c3c3c},
			{0x2001c68c, 4, 0x3c3c3c3c},
			{0x2001c690, 4, 0x3c3c3c3c},
			{0x2001c694, 4, 0x3c3c3c3c},
			{0x2001c698, 4, 0x3c3c3c3c},
			{0x2001c69c, 4, 0x3c3c3c3c},
			{0x2001c6a0, 4, 0x3c3c3c3c},
			{0x2001c6a4, 4, 0x3c3c3c3c},
			{0x2001c6a8, 4, 0x3c3c3c3c},
			{0x2001c6ac, 4, 0x3c3c3c3c},
			{0x2001c6b0, 4, 0x3c3c3c3c},
			{0x2001c6b4, 4, 0x3c3c3c3c},
			{0x2001c6b8, 4, 0x3c3c3c3c},
			{0x2001c6bc, 4, 0x3c3c3c3c},
			{0x2001c6c0, 4, 0x3c3c3c3c},
			{0x2001c6c4, 4, 0x3c3c3c3c},
			{0x2001c6c8, 4, 0x3c3c3c3c},
			{0x2001c6cc, 4, 0x3c3c3c3c},
			{0x2001c6d0, 4, 0x3c3c3c3c},
			{0x2001c6d4, 4, 0x3c3c3c3c},
			{0x2001c6d8, 4, 0x3c3c3c3c},
			{0x2001c6dc, 4, 0x3c3c3c3c},
			{0x2001c6e0, 4, 0x3c3c3c3c},
			{0x2001c6e4, 4, 0x3c3c3c3c},
			{0x2001c6e8, 4, 0x3c3c3c3c},
			{0x2001c6ec, 4, 0x3c3c3c3c},
			{0x2001c6f0, 4, 0x3c3c3c3c},
			{0x2001c6f4, 4, 0x3c3c3c3c},
			{0x2001c6f8, 4, 0x3c3c3c3c},
			{0x2001c6fc, 4, 0x3c3c3c3c},
			{0x2001c700, 4, 0x3c3c3c3c},
			{0x2001c704, 4, 0x3c3c3c3c},
			{0x2001c708, 4, 0x3c3c3c3c},
			{0x2001c70c, 4, 0x3c3c3c3c},
			{0x2001c710, 4, 0x3c3c3c3c},
			{0x2001c714, 4, 0x3c3c3c3c},
			{0x2001c718, 4, 0x3c3c3c3c},
			{0x2001c71c, 4, 0x3c3c3c3c},
			{0x2001c720, 4, 0x3c3c3c3c},
			{0x2001c724, 4, 0x3c3c3c3c},
			{0x2001c728, 4, 0x3c3c3c3c},
			{0x2001c72c, 4, 0x3c3c3c3c},
			{0x2001c730, 4, 0x3c3c3c3c},
			{0x2001c734, 4, 0x3c3c3c3c},
			{0x2001c738, 4, 0x3c3c3c3c},
			{0x2001c73c, 4, 0x3c3c3c3c},
			{0x2001c740, 4, 0x3c3c3c3c},
			{0x2001c744, 4, 0x3c3c3c3c},
			{0x2001c748, 4, 0x3c3c3c3c},
			{0x2001c74c, 4, 0x3c3c3c3c},
			{0x2001c750, 4, 0x3c3c3c3c},
			{0x2001c754, 4, 0x3c3c3c3c},
			{0x2001c758, 4, 0x3c3c3c3c},
			{0x2001c75c, 4, 0x3c3c3c3c},
			{0x2001c760, 4, 0x3c3c3c3c},
			{0x2001c8c0, 4, 0x5a5a5a5a},
			{0x2001c8c4, 4, 0x5a5a5a5a},
			{0x2001c8c8, 4, 0x5a5a5a5a},
			{0x2001c8cc, 4, 0x5a5a5a5a},
			{0x2001c8d0, 4, 0x5a5a5a5a},
			{0x2001c8d4, 4, 0x5a5a5a5a},
			{0x2001c8d8, 4, 0x5a5a5a5a},
			{0x2001c8dc, 4, 0x5a5a5a5a},
			{0x2001c8e0, 4, 0x5a5a5a5a},
			{0x2001c8e4, 4, 0x5a5a5a5a},
			{0x2001c8e8, 4, 0x5a5a5a5a},
			{0x2001c8ec, 4, 0x5a5a5a5a},
			{0x2001c8f0, 4, 0x5a5a5a5a},
			{0x2001c8f4, 4, 0x5a5a5a5a},
			{0x2001c8f8, 4, 0x5a5a5a5a},
			{0x2001c8fc, 4, 0x5a5a5a5a},
			{0x2001c900, 4, 0x5a5a5a5a},
			{0x2001c904, 4, 0x5a5a5a5a},
			{0x2001c908, 4, 0x5a5a5a5a},
			{0x2001c90c, 4, 0x5a5a5a5a},
			{0x2001c910, 4, 0x5a5a5a5a},
			{0x2001c914, 4, 0x5a5a5a5a},
			{0x2001c918, 4, 0x5a5a5a5a},
			{0x2001c91c, 4, 0x5a5a5a5a},
			{0x2001c920, 4, 0x5a5a5a5a},
			{0x2001c924, 4, 0x5a5a5a5a},
			{0x2001c928, 4, 0x5a5a5a5a},
			{0x2001c92c, 4, 0x5a5a5a5a},
			{0x2001c930, 4, 0x5a5a5a5a},
			{0x2001c934, 4, 0x5a5a5a5a},
			{0x2001c938, 4, 0x5a5a5a5a},
			{0x2001c93c, 4, 0x5a5a5a5a},
			{0x2001c940, 4, 0x5a5a5a5a},
			{0x2001c944, 4, 0x5a5a5a5a},
			{0x2001c948, 4, 0x5a5a5a5a},
			{0x2001c94c, 4, 0x5a5a5a5a},
			{0x2001c950, 4, 0x5a5a5a5a},
			{0x2001c954, 4, 0x5a5a5a5a},
			{0x2001c958, 4, 0x5a5a5a5a},
			{0x2001c95c, 4, 0x5a5a5a5a},
			{0x2001c960, 4, 0x5a5a5a5a},
			{0x2001c964, 4, 0x5a5a5a5a},
			{0x2001c968, 4, 0x5a5a5a5a},
			{0x2001c96c, 4, 0x5a5a5a5a},
			{0x2001c970, 4, 0x5a5a5a5a},
			{0x2001c974, 4, 0x5a5a5a5a},
			{0x2001c978, 4, 0x5a5a5a5a},
			{0x2001c97c, 4, 0x5a5a5a5a},
			{0x2001c980, 4, 0x5a5a5a5a},
			{0x2001c984, 4, 0x5a5a5a5a},
			{0x2001c988, 4, 0x5a5a5a5a},
			{0x2001c98c, 4, 0x5a5a5a5a},
			{0x2001c990, 4, 0x5a5a5a5a},
			{0x2001c994, 4, 0x5a5a5a5a},
			{0x2001c998, 4, 0x5a5a5a5a},
			{0x2001c99c, 4, 0x5a5a5a5a},
			{0x2001c9a0, 4, 0x5a5a5a5a},
			{0x2001c9a4, 4, 0x5a5a5a5a},
			{0x2001c9a8, 4, 0x5a5a5a5a},
			{0x2001c9ac, 4, 0x5a5a5a5a},
			{0x2001c9b0, 4, 0x5a5a5a5a},
			{0x2001c9b4, 4, 0x5a5a5a5a},
			{0x2001c9b8, 4, 0x5a5a5a5a},
			{0x2001c9bc, 4, 0x5a5a5a5a},
			{0x2001c9c0, 4, 0x5a5a5a5a},
			{0x2001c9c4, 4, 0x5a5a5a5a},
			{0x2001c9c8, 4, 0x5a5a5a5a},
			{0x2001c9cc, 4, 0x5a5a5a5a},
			{0x2001c9d0, 4, 0x5a5a5a5a},
			{0x2001c9d4, 4, 0x5a5a5a5a},
			{0x2001c9d8, 4, 0x5a5a5a5a},
			{0x2001c9dc, 4, 0x5a5a5a5a},
			{0x2001c9e0, 4, 0x5a5a5a5a},
			{0x2001c9e4, 4, 0x5a5a5a5a},
			{0x2001c9e8, 4, 0x5a5a5a5a},
			{0x2001c9ec, 4, 0x5a5a5a5a},
			{0x2001c9f0, 4, 0x5a5a5a5a},
			{0x2001c9f4, 4, 0x5a5a5a5a},
			{0x2001c9f8, 4, 0x5a5a5a5a},
			{0x2001c9fc, 4, 0x5a5a5a5a},
			{0x2001ca00, 4, 0x5a5a5a5a},
			{0x2001ca04, 4, 0x5a5a5a5a},
			{0x2001ca08, 4, 0x5a5a5a5a},
			{0x2001ca0c, 4, 0x5a5a5a5a},
			{0x2001ca10, 4, 0x5a5a5a5a},
			{0x2001ca14, 4, 0x5a5a5a5a},
			{0x2001ca18, 4, 0x5a5a5a5a},
			{0x2001ca1c, 4, 0x5a5a5a5a},
			{0x2001ca20, 4, 0x5a5a5a5a},
			{0x2001ca24, 4, 0x5a5a5a5a},
			{0x2001ca28, 4, 0x5a5a5a5a},
			{0x2001ca2c, 4, 0x5a5a5a5a},
			{0x2001ca30, 4, 0x5a5a5a5a},
			{0x2001ca34, 4, 0x5a5a5a5a},
			{0x2001ca38, 4, 0x5a5a5a5a},
			{0x2001ca3c, 4, 0x5a5a5a5a},
			{0x2001ca40, 4, 0x5a5a5a5a},
			{0x2001ca44, 4, 0x5a5a5a5a},
			{0x2001ca48, 4, 0x5a5a5a5a},
			{0x2001ca4c, 4, 0x5a5a5a5a},
			{0x2001ca50, 4, 0x5a5a5a5a},
			{0x2001ca54, 4, 0x5a5a5a5a},
			{0x2001ca58, 4, 0x5a5a5a5a},
			{0x2001ca5c, 4, 0x5a5a5a5a},
			{0x2001ca60, 4, 0x5a5a5a5a},
			{0x20000626, 1, 0x0},
			{0x20000548, 1, 0x0},
			{0x20000633, 4, 0x1},
			{0x20000637, 4, 0x3},
			{0x20000ee1, 1, 0x1},
			{0x20000e03, 1, 0x1},
			{0x20000eee, 4, 0x1},
			{0x20000ef2, 4, 0x3},
			{0x200010a0, 1, 0x0},
			{0x20000fc2, 1, 0x0},
			{0x200010ad, 4, 0x0},
			{0x200010b1, 4, 0x4},
			{0x2000125f, 1, 0x1},
			{0x20001181, 1, 0x1},
			{0x2000126c, 4, 0x0},
			{0x20001270, 4, 0x5},
			{0x2000141e, 1, 0x1},
			{0x20001340, 1, 0x0},
			{0x2000142b, 4, 0x0},
			{0x2000142f, 4, 0x6},
			{0x200015dd, 1, 0x0},
			{0x200014ff, 1, 0x1},
			{0x200015ea, 4, 0x0},
			{0x200015ee, 4, 0x7},
			{0x2000179c, 1, 0x1},
			{0x200016be, 1, 0x0},
			{0x200017a9, 4, 0x1},
			{0x200017ad, 4, 0x3},
			{0x2000195b, 1, 0x1},
			{0x2000187d, 1, 0x1},
			{0x20001968, 4, 0x0},
			{0x2000196c, 4, 0x4},
			{0x20001b1a, 1, 0x0},
			{0x20001a3c, 1, 0x0},
			{0x20001b27, 4, 0x0},
			{0x20001b2b, 4, 0x5},
			{0x20001cd9, 1, 0x1},
			{0x20001bfb, 1, 0x1},
			{0x20001ce6, 4, 0x0},
			{0x20001cea, 4, 0x6},
			{0x20001e98, 1, 0x1},
			{0x20001dba, 1, 0x0},
			{0x20001ea5, 4, 0x0},
			{0x20001ea9, 4, 0x7},
			{0x20002057, 1, 0x0},
			{0x20001f79, 1, 0x1},
			{0x20002064, 4, 0x1},
			{0x20002068, 4, 0x3},
			{0x20002216, 1, 0x1},
			{0x20002138, 1, 0x0},
			{0x20002223, 4, 0x0},
			{0x20002227, 4, 0x4},
			{0x200023d5, 1, 0x1},
			{0x200022f7, 1, 0x1},
			{0x200023e2, 4, 0x0},
			{0x200023e6, 4, 0x5},
			{0x20002594, 1, 0x0},
			{0x200024b6, 1, 0x0},
			{0x200025a1, 4, 0x0},
			{0x200025a5, 4, 0x6},
			{0x20002753, 1, 0x1},
			{0x20002675, 1, 0x1},
			{0x20002760, 4, 0x0},
			{0x20002764, 4, 0x7},
			{0x20002912, 1, 0x1},
			{0x20002834, 1, 0x0},
			{0x2000291f, 4, 0x1},
			{0x20002923, 4, 0x3},
			{0x20002ad1, 1, 0x0},
			{0x200029f3, 1, 0x1},
			{0x20002ade, 4, 0x0},
			{0x20002ae2, 4, 0x4},
			{0x20002c90, 1, 0x1},
			{0x20002bb2, 1, 0x0},
			{0x20002c9d, 4, 0x0},
			{0x20002ca1, 4, 0x5},
			{0x20002e4f, 1, 0x1},
			{0x20002d71, 1, 0x1},
			{0x20002e5c, 4, 0x0},
			{0x20002e60, 4, 0x6},
			{0x2000300e, 1, 0x0},
			{0x20002f30, 1, 0x0},
			{0x2000301b, 4, 0x0},
			{0x2000301f, 4, 0x7},
			{0x200031cd, 1, 0x1},
			{0x200030ef, 1, 0x1},
			{0x200031da, 4, 0x1},
			{0x200031de, 4, 0x3},
			{0x2000338c, 1, 0x1},
			{0x200032ae, 1, 0x0},
			{0x20003399, 4, 0x0},
			{0x2000339d, 4, 0x4},
			{0x2000354b, 1, 0x0},
			{0x2000346d, 1, 0x1},
			{0x20003558, 4, 0x0},
			{0x2000355c, 4, 0x5},
			{0x2000370a, 1, 0x1},
			{0x2000362c, 1, 0x0},
			{0x20003717, 4, 0x0},
			{0x2000371b, 4, 0x6},
			{0x200038c9, 1, 0x1},
			{0x200037eb, 1, 0x1},
			{0x200038d6, 4, 0x0},
			{0x200038da, 4, 0x7},
			{0x20003a88, 1, 0x0},
			{0x200039aa, 1, 0x0},
			{0x20003a95, 4, 0x1},
			{0x20003a99, 4, 0x3},
			{0x20003c47, 1, 0x1},
			{0x20003b69, 1, 0x1},
			{0x20003c54, 4, 0x0},
			{0x20003c58, 4, 0x4},
			{0x20003e06, 1, 0x1},
			{0x20003d28, 1, 0x0},
			{0x20003e13, 4, 0x0},
			{0x20003e17, 4, 0x5},
			{0x20003fc5, 1, 0x0},
			{0x20003ee7, 1, 0x1},
			{0x20003fd2, 4, 0x0},
			{0x20003fd6, 4, 0x6},
			{0x20004184, 1, 0x1},
			{0x200040a6, 1, 0x0},
			{0x20004191, 4, 0x0},
			{0x20004195, 4, 0x7},
			{0x20004343, 1, 0x1},
			{0x20004265, 1, 0x1},
			{0x20004350, 4, 0x1},
			{0x20004354, 4, 0x3},
			{0x20004502, 1, 0x0},
			{0x20004424, 1, 0x0},
			{0x2000450f, 4, 0x0},
			{0x20004513, 4, 0x4},
			{0x200046c1, 1, 0x1},
			{0x200045e3, 1, 0x1},
			{0x200046ce, 4, 0x0},
			{0x200046d2, 4, 0x5},
			{0x20004880, 1, 0x1},
			{0x200047a2, 1, 0x0},
			{0x2000488d, 4, 0x0},
			{0x20004891, 4, 0x6},
			{0x20004a3f, 1, 0x0},
			{0x20004961, 1, 0x1},
			{0x20004a4c, 4, 0x0},
			{0x20004a50, 4, 0x7},
			{0x20004bfe, 1, 0x1},
			{0x20004b20, 1, 0x0},
			{0x20004c0b, 4, 0x1},
			{0x20004c0f, 4, 0x3},
			{0x20004f7c, 1, 0x0},
			{0x20004e9e, 1, 0x0},
			{0x20004f89, 4, 0x0},
			{0x20004f8d, 4, 0x5},
			{0x2000513b, 1, 0x1},
			{0x2000505d, 1, 0x1},
			{0x20005148, 4, 0x0},
			{0x2000514c, 4, 0x6},
			{0x200052fa, 1, 0x1},
			{0x2000521c, 1, 0x0},
			{0x20005307, 4, 0x0},
			{0x2000530b, 4, 0x7},
			{0x200054b9, 1, 0x0},
			{0x200053db, 1, 0x1},
			{0x200054c6, 4, 0x1},
			{0x200054ca, 4, 0x3},
			{0x20005678, 1, 0x1},
			{0x2000559a, 1, 0x0},
			{0x20005685, 4, 0x0},
			{0x20005689, 4, 0x4},
			{0x20005837, 1, 0x1},
			{0x20005759, 1, 0x1},
			{0x20005844, 4, 0x0},
			{0x20005848, 4, 0x5},
			{0x200059f6, 1, 0x0},
			{0x20005918, 1, 0x0},
			{0x20005a03, 4, 0x0},
			{0x20005a07, 4, 0x6},
			{0x20005bb5, 1, 0x1},
			{0x20005ad7, 1, 0x1},
			{0x20005bc2, 4, 0x0},
			{0x20005bc6, 4, 0x7},
			{0x20005d74, 1, 0x1},
			{0x20005c96, 1, 0x0},
			{0x20005d81, 4, 0x1},
			{0x20005d85, 4, 0x3},
			{0x20005f33, 1, 0x0},
			{0x20005e55, 1, 0x1},
			{0x20005f40, 4, 0x0},
			{0x20005f44, 4, 0x4},
			{0x200060f2, 1, 0x1},
			{0x20006014, 1, 0x0},
			{0x200060ff, 4, 0x0},
			{0x20006103, 4, 0x5},
			{0x200062b1, 1, 0x1},
			{0x200061d3, 1, 0x1},
			{0x200062be, 4, 0x0},
			{0x200062c2, 4, 0x6},
			{0x20006470, 1, 0x0},
			{0x20006392, 1, 0x0},
			{0x2000647d, 4, 0x0},
			{0x20006481, 4, 0x7},
			{0x2000662f, 1, 0x1},
			{0x20006551, 1, 0x1},
			{0x2000663c, 4, 0x1},
			{0x20006640, 4, 0x3},
			{0x200067ee, 1, 0x1},
			{0x20006710, 1, 0x0},
			{0x200067fb, 4, 0x0},
			{0x200067ff, 4, 0x4},
			{0x200069ad, 1, 0x0},
			{0x200068cf, 1, 0x1},
			{0x200069ba, 4, 0x0},
			{0x200069be, 4, 0x5},
			{0x20006b6c, 1, 0x1},
			{0x20006a8e, 1, 0x0},
			{0x20006b79, 4, 0x0},
			{0x20006b7d, 4, 0x6},
			{0x20006d2b, 1, 0x1},
			{0x20006c4d, 1, 0x1},
			{0x20006d38, 4, 0x0},
			{0x20006d3c, 4, 0x7},
			{0x200070a9, 1, 0x1},
			{0x20006fcb, 1, 0x1},
			{0x200070b6, 4, 0x0},
			{0x200070ba, 4, 0x4},
			{0x20007268, 1, 0x1},
			{0x2000718a, 1, 0x0},
			{0x20007275, 4, 0x0},
			{0x20007279, 4, 0x5},
			{0x20007427, 1, 0x0},
			{0x20007349, 1, 0x1},
			{0x20007434, 4, 0x0},
			{0x20007438, 4, 0x6},
			{0x200075e6, 1, 0x1},
			{0x20007508, 1, 0x0},
			{0x200075f3, 4, 0x0},
			{0x200075f7, 4, 0x7},
			{0x200077a5, 1, 0x1},
			{0x200076c7, 1, 0x1},
			{0x200077b2, 4, 0x1},
			{0x200077b6, 4, 0x3},
			{0x20007964, 1, 0x0},
			{0x20007886, 1, 0x0},
			{0x20007971, 4, 0x0},
			{0x20007975, 4, 0x4},
			{0x20007b23, 1, 0x1},
			{0x20007a45, 1, 0x1},
			{0x20007b30, 4, 0x0},
			{0x20007b34, 4, 0x5},
			{0x20007ce2, 1, 0x1},
			{0x20007c04, 1, 0x0},
			{0x20007cef, 4, 0x0},
			{0x20007cf3, 4, 0x6},
			{0x20007ea1, 1, 0x0},
			{0x20007dc3, 1, 0x1},
			{0x20007eae, 4, 0x0},
			{0x20007eb2, 4, 0x7},
			{0x20008060, 1, 0x1},
			{0x20007f82, 1, 0x0},
			{0x2000806d, 4, 0x1},
			{0x20008071, 4, 0x3},
			{0x2000821f, 1, 0x1},
			{0x20008141, 1, 0x1},
			{0x2000822c, 4, 0x0},
			{0x20008230, 4, 0x4},
			{0x200083de, 1, 0x0},
			{0x20008300, 1, 0x0},
			{0x200083eb, 4, 0x0},
			{0x200083ef, 4, 0x5},
			{0x2000859d, 1, 0x1},
			{0x200084bf, 1, 0x1},
			{0x200085aa, 4, 0x0},
			{0x200085ae, 4, 0x6},
			{0x2000875c, 1, 0x1},
			{0x2000867e, 1, 0x0},
			{0x20008769, 4, 0x0},
			{0x2000876d, 4, 0x7},
			{0x2000891b, 1, 0x0},
			{0x2000883d, 1, 0x1},
			{0x20008928, 4, 0x1},
			{0x2000892c, 4, 0x3},
			{0x20008ada, 1, 0x1},
			{0x200089fc, 1, 0x0},
			{0x20008ae7, 4, 0x0},
			{0x20008aeb, 4, 0x4},
			{0x20008c99, 1, 0x1},
			{0x20008bbb, 1, 0x1},
			{0x20008ca6, 4, 0x0},
			{0x20008caa, 4, 0x5},
			{0x20008e58, 1, 0x0},
			{0x20008d7a, 1, 0x0},
			{0x20008e65, 4, 0x0},
			{0x20008e69, 4, 0x6},
			{0x20009017, 1, 0x1},
			{0x20008f39, 1, 0x1},
			{0x20009024, 4, 0x0},
			{0x20009028, 4, 0x7},
			{0x200091d6, 1, 0x1},
			{0x200090f8, 1, 0x0},
			{0x200091e3, 4, 0x1},
			{0x200091e7, 4, 0x3},
			{0x20009395, 1, 0x0},
			{0x200092b7, 1, 0x1},
			{0x200093a2, 4, 0x0},
			{0x200093a6, 4, 0x4},
			{0x20009554, 1, 0x1},
			{0x20009476, 1, 0x0},
			{0x20009561, 4, 0x0},
			{0x20009565, 4, 0x5},
			{0x20009713, 1, 0x1},
			{0x20009635, 1, 0x1},
			{0x20009720, 4, 0x0},
			{0x20009724, 4, 0x6},
			{0x200098d2, 1, 0x0},
			{0x200097f4, 1, 0x0},
			{0x200098df, 4, 0x0},
			{0x200098e3, 4, 0x7},
			{0x20009a91, 1, 0x1},
			{0x200099b3, 1, 0x1},
			{0x20009a9e, 4, 0x1},
			{0x20009aa2, 4, 0x3},
			{0x20009c50, 1, 0x1},
			{0x20009b72, 1, 0x0},
			{0x20009c5d, 4, 0x0},
			{0x20009c61, 4, 0x4},
			{0x20009e0f, 1, 0x0},
			{0x20009d31, 1, 0x1},
			{0x20009e1c, 4, 0x0},
			{0x20009e20, 4, 0x5},
			{0x20009fce, 1, 0x1},
			{0x20009ef0, 1, 0x0},
			{0x20009fdb, 4, 0x0},
			{0x20009fdf, 4, 0x6},
			{0x2000a18d, 1, 0x1},
			{0x2000a0af, 1, 0x1},
			{0x2000a19a, 4, 0x0},
			{0x2000a19e, 4, 0x7},
			{0x2000a34c, 1, 0x0},
			{0x2000a26e, 1, 0x0},
			{0x2000a359, 4, 0x1},
			{0x2000a35d, 4, 0x3},
			{0x2000a50b, 1, 0x1},
			{0x2000a42d, 1, 0x1},
			{0x2000a518, 4, 0x0},
			{0x2000a51c, 4, 0x4},
			{0x2000a6ca, 1, 0x1},
			{0x2000a5ec, 1, 0x0},
			{0x2000a6d7, 4, 0x0},
			{0x2000a6db, 4, 0x5},
			{0x2000a889, 1, 0x0},
			{0x2000a7ab, 1, 0x1},
			{0x2000a896, 4, 0x0},
			{0x2000a89a, 4, 0x6},
			{0x2000aa48, 1, 0x1},
			{0x2000a96a, 1, 0x0},
			{0x2000aa55, 4, 0x0},
			{0x2000aa59, 4, 0x7},
			{0x2000ac07, 1, 0x1},
			{0x2000ab29, 1, 0x1},
			{0x2000ac14, 4, 0x1},
			{0x2000ac18, 4, 0x3},
			{0x2000adc6, 1, 0x0},
			{0x2000ace8, 1, 0x0},
			{0x2000add3, 4, 0x0},
			{0x2000add7, 4, 0x4},
			{0x2000af85, 1, 0x1},
			{0x2000aea7, 1, 0x1},
			{0x2000af92, 4, 0x0},
			{0x2000af96, 4, 0x5},
			{0x2000b144, 1, 0x1},
			{0x2000b066, 1, 0x0},
			{0x2000b151, 4, 0x0},
			{0x2000b155, 4, 0x6},
			{0x2000b303, 1, 0x0},
			{0x2000b225, 1, 0x1},
			{0x2000b310, 4, 0x0},
			{0x2000b314, 4, 0x7},
			{0x2000b4c2, 1, 0x1},
			{0x2000b3e4, 1, 0x0},
			{0x2000b4cf, 4, 0x1},
			{0x2000b4d3, 4, 0x3},
			{0x2000b681, 1, 0x1},
			{0x2000b5a3, 1, 0x1},
			{0x2000b68e, 4, 0x0},
			{0x2000b692, 4, 0x4},
			{0x2000b840, 1, 0x0},
			{0x2000b762, 1, 0x0},
			{0x2000b84d, 4, 0x0},
			{0x2000b851, 4, 0x5},
			{0x2000b9ff, 1, 0x1},
			{0x2000b921, 1, 0x1},
			{0x2000ba0c, 4, 0x0},
			{0x2000ba10, 4, 0x6},
			{0x2000bbbe, 1, 0x1},
			{0x2000bae0, 1, 0x0},
			{0x2000bbcb, 4, 0x0},
			{0x2000bbcf, 4, 0x7},
			{0x2000bd7d, 1, 0x0},
			{0x2000bc9f, 1, 0x1},
			{0x2000bd8a, 4, 0x1},
			{0x2000bd8e, 4, 0x3},
			{0x2000bf3c, 1, 0x1},
			{0x2000be5e, 1, 0x0},
			{0x2000bf49, 4, 0x0},
			{0x2000bf4d, 4, 0x4},
			{0x2000c0fb, 1, 0x1},
			{0x2000c01d, 1, 0x1},
			{0x2000c108, 4, 0x0},
			{0x2000c10c, 4, 0x5},
			{0x2000c2ba, 1, 0x0},
			{0x2000c1dc, 1, 0x0},
			{0x2000c2c7, 4, 0x0},
			{0x2000c2cb, 4, 0x6},
			{0x2000c479, 1, 0x1},
			{0x2000c39b, 1, 0x1},
			{0x2000c486, 4, 0x0},
			{0x2000c48a, 4, 0x7},
			{0x2000c638, 1, 0x1},
			{0x2000c55a, 1, 0x0},
			{0x2000c645, 4, 0x1},
			{0x2000c649, 4, 0x3},
			{0x2000c7f7, 1, 0x0},
			{0x2000c719, 1, 0x1},
			{0x2000c804, 4, 0x0},
			{0x2000c808, 4, 0x4},
			{0x2000c9b6, 1, 0x1},
			{0x2000c8d8, 1, 0x0},
			{0x2000c9c3, 4, 0x0},
			{0x2000c9c7, 4, 0x5},
			{0x2000cb75, 1, 0x1},
			{0x2000ca97, 1, 0x1},
			{0x2000cb82, 4, 0x0},
			{0x2000cb86, 4, 0x6},
			{0x2000cd34, 1, 0x0},
			{0x2000cc56, 1, 0x0},
			{0x2000cd41, 4, 0x0},
			{0x2000cd45, 4, 0x7},
			{0x2000cef3, 1, 0x1},
			{0x2000ce15, 1, 0x1},
			{0x2000cf00, 4, 0x1},
			{0x2000cf04, 4, 0x3},
			{0x2000d0b2, 1, 0x1},
			{0x2000cfd4, 1, 0x0},
			{0x2000d0bf, 4, 0x0},
			{0x2000d0c3, 4, 0x4},
			{0x2000d271, 1, 0x0},
			{0x2000d193, 1, 0x1},
			{0x2000d27e, 4, 0x0},
			{0x2000d282, 4, 0x5},
			{0x2000d430, 1, 0x1},
			{0x2000d352, 1, 0x0},
			{0x2000d43d, 4, 0x0},
			{0x2000d441, 4, 0x6},
			{0x2000d7ae, 1, 0x0},
			{0x2000d6d0, 1, 0x0},
			{0x2000d7bb, 4, 0x1},
			{0x2000d7bf, 4, 0x3},
			{0x2000d96d, 1, 0x1},
			{0x2000d88f, 1, 0x1},
			{0x2000d97a, 4, 0x0},
			{0x2000d97e, 4, 0x4},
			{0x2000db2c, 1, 0x1},
			{0x2000da4e, 1, 0x0},
			{0x2000db39, 4, 0x0},
			{0x2000db3d, 4, 0x5},
			{0x2000dceb, 1, 0x0},
			{0x2000dc0d, 1, 0x1},
			{0x2000dcf8, 4, 0x0},
			{0x2000dcfc, 4, 0x6},
			{0x2000deaa, 1, 0x1},
			{0x2000ddcc, 1, 0x0},
			{0x2000deb7, 4, 0x0},
			{0x2000debb, 4, 0x7},
			{0x2000e069, 1, 0x1},
			{0x2000df8b, 1, 0x1},
			{0x2000e076, 4, 0x1},
			{0x2000e07a, 4, 0x3},
			{0x2000e228, 1, 0x0},
			{0x2000e14a, 1, 0x0},
			{0x2000e235, 4, 0x0},
			{0x2000e239, 4, 0x4},
			{0x2000e3e7, 1, 0x1},
			{0x2000e309, 1, 0x1},
			{0x2000e3f4, 4, 0x0},
			{0x2000e3f8, 4, 0x5},
			{0x2000e5a6, 1, 0x1},
			{0x2000e4c8, 1, 0x0},
			{0x2000e5b3, 4, 0x0},
			{0x2000e5b7, 4, 0x6},
			{0x2000e765, 1, 0x0},
			{0x2000e687, 1, 0x1},
			{0x2000e772, 4, 0x0},
			{0x2000e776, 4, 0x7},
			{0x2000e924, 1, 0x1},
			{0x2000e846, 1, 0x0},
			{0x2000e931, 4, 0x1},
			{0x2000e935, 4, 0x3},
			{0x2000eae3, 1, 0x1},
			{0x2000ea05, 1, 0x1},
			{0x2000eaf0, 4, 0x0},
			{0x2000eaf4, 4, 0x4},
			{0x2000eca2, 1, 0x0},
			{0x2000ebc4, 1, 0x0},
			{0x2000ecaf, 4, 0x0},
			{0x2000ecb3, 4, 0x5},
			{0x2000ee61, 1, 0x1},
			{0x2000ed83, 1, 0x1},
			{0x2000ee6e, 4, 0x0},
			{0x2000ee72, 4, 0x6},
			{0x2000f020, 1, 0x1},
			{0x2000ef42, 1, 0x0},
			{0x2000f02d, 4, 0x0},
			{0x2000f031, 4, 0x7},
			{0x2000f1df, 1, 0x0},
			{0x2000f101, 1, 0x1},
			{0x2000f1ec, 4, 0x1},
			{0x2000f1f0, 4, 0x3},
			{0x2000f39e, 1, 0x1},
			{0x2000f2c0, 1, 0x0},
			{0x2000f3ab, 4, 0x0},
			{0x2000f3af, 4, 0x4},
			{0x2000f55d, 1, 0x1},
			{0x2000f47f, 1, 0x1},
			{0x2000f56a, 4, 0x0},
			{0x2000f56e, 4, 0x5},
			{0x2000f71c, 1, 0x0},
			{0x2000f63e, 1, 0x0},
			{0x2000f729, 4, 0x0},
			{0x2000f72d, 4, 0x6},
			{0x2000f8db, 1, 0x1},
			{0x2000f7fd, 1, 0x1},
			{0x2000f8e8, 4, 0x0},
			{0x2000f8ec, 4, 0x7},
			{0x2000fa9a, 1, 0x1},
			{0x2000f9bc, 1, 0x0},
			{0x2000faa7, 4, 0x1},
			{0x2000faab, 4, 0x3},
			{0x2000fc59, 1, 0x0},
			{0x2000fb7b, 1, 0x1},
			{0x2000fc66, 4, 0x0},
			{0x2000fc6a, 4, 0x4},
			{0x2000fe18, 1, 0x1},
			{0x2000fd3a, 1, 0x0},
			{0x2000fe25, 4, 0x0},
			{0x2000fe29, 4, 0x5},
			{0x2000ffd7, 1, 0x1},
			{0x2000fef9, 1, 0x1},
			{0x2000ffe4, 4, 0x0},
			{0x2000ffe8, 4, 0x6},
			{0x20010196, 1, 0x0},
			{0x200100b8, 1, 0x0},
			{0x200101a3, 4, 0x0},
			{0x200101a7, 4, 0x7},
			{0x20010355, 1, 0x1},
			{0x20010277, 1, 0x1},
			{0x20010362, 4, 0x1},
			{0x20010366, 4, 0x3},
			{0x20010514, 1, 0x1},
			{0x20010436, 1, 0x0},
			{0x20010521, 4, 0x0},
			{0x20010525, 4, 0x4},
			{0x200106d3, 1, 0x0},
			{0x200105f5, 1, 0x1},
			{0x200106e0, 4, 0x0},
			{0x200106e4, 4, 0x5},
			{0x20010892, 1, 0x1},
			{0x200107b4, 1, 0x0},
			{0x2001089f, 4, 0x0},
			{0x200108a3, 4, 0x6},
			{0x20010c10, 1, 0x0},
			{0x20010b32, 1, 0x0},
			{0x20010c1d, 4, 0x1},
			{0x20010c21, 4, 0x3},
			{0x2001cb00, 4, 0x64},
		},
		in: []mstFld{
			{addr: 0x683f94, size: 1, signed: false, idx: -1},
			{addr: 0x68d524, size: 1, signed: false, idx: -1},
			{addr: 0x68d525, size: 1, signed: false, idx: -1},
			{addr: 0x652da4, size: 1, signed: false, idx: -1},
			{addr: 0x2001e65c, size: 4, signed: false, idx: -1},
			{addr: 0x2001e660, size: 4, signed: false, idx: -1},
			{addr: 0x2001e664, size: 4, signed: false, idx: -1},
			{addr: 0x2001e844, size: 4, signed: false, idx: -1},
			{addr: 0x2001da18, size: 4, signed: false, idx: -1},
			{addr: 0x2001da1c, size: 4, signed: false, idx: -1},
			{addr: 0x2001da20, size: 4, signed: false, idx: -1},
			{addr: 0x2001da8c, size: 4, signed: false, idx: -1},
			{addr: 0x2001dc18, size: 4, signed: false, idx: -1},
			{addr: 0x2001dc1c, size: 4, signed: false, idx: -1},
			{addr: 0x2001dc20, size: 4, signed: false, idx: -1},
			{addr: 0x2001dc8c, size: 4, signed: false, idx: -1},
			{addr: 0x2001de18, size: 4, signed: false, idx: -1},
			{addr: 0x2001de1c, size: 4, signed: false, idx: -1},
			{addr: 0x2001de20, size: 4, signed: false, idx: -1},
			{addr: 0x2001de8c, size: 4, signed: false, idx: -1},
			{addr: 0x2001e0d0, size: 4, signed: false, idx: -1},
			{addr: 0x2001e140, size: 4, signed: false, idx: -1},
			{addr: 0x2001cf00, size: 4, signed: false, idx: -1},
			{addr: 0x2001cf04, size: 1, signed: false, idx: -1},
			{addr: 0x2001cf05, size: 4, signed: false, idx: -1},
			{addr: 0x2001cf09, size: 4, signed: false, idx: -1},
			{addr: 0x2001cf11, size: 4, signed: false, idx: -1},
			{addr: 0x2001d046, size: 1, signed: false, idx: -1},
			{addr: 0x2001d04b, size: 1, signed: false, idx: -1},
			{addr: 0x2001d052, size: 1, signed: false, idx: -1},
			{addr: 0x2001e308, size: 4, signed: false, idx: -1},
			{addr: 0x2001e328, size: 4, signed: false, idx: -1},
			{addr: 0x2001e345, size: 1, signed: false, idx: -1},
			{addr: 0x2001e365, size: 1, signed: false, idx: -1},
			{addr: 0x2001e385, size: 1, signed: false, idx: -1},
			{addr: 0x2001e488, size: 1, signed: false, idx: -1},
			{addr: 0x2001e588, size: 1, signed: false, idx: -1},
			{addr: 0x2001cb04, size: 4, signed: false, idx: -1},
			{addr: 0x2001cb08, size: 4, signed: false, idx: -1},
			{addr: 0x2001cb15, size: 1, signed: false, idx: -1},
			{addr: 0x2001cb16, size: 1, signed: false, idx: -1},
			{addr: 0x2001cb27, size: 4, signed: false, idx: -1},
			{addr: 0x2001cb2b, size: 4, signed: false, idx: -1},
			{addr: 0x2001cb2f, size: 4, signed: false, idx: -1},
			{addr: 0x2001cb33, size: 4, signed: false, idx: -1},
			{addr: 0x2001cb77, size: 4, signed: false, idx: -1},
			{addr: 0x2001cb87, size: 4, signed: false, idx: -1},
			{addr: 0x2001cb8b, size: 1, signed: false, idx: -1},
			{addr: 0x2001cc07, size: 4, signed: false, idx: -1},
			{addr: 0x2001cd4e, size: 1, signed: false, idx: -1},
			{addr: 0x2001cdba, size: 4, signed: false, idx: -1},
			{addr: 0x2001cdbe, size: 4, signed: false, idx: -1},
			{addr: 0x2001cdc2, size: 4, signed: false, idx: -1},
			{addr: 0x2001cdc6, size: 4, signed: false, idx: -1},
			{addr: 0x2001cdca, size: 4, signed: false, idx: -1},
			{addr: 0x2001cdce, size: 4, signed: false, idx: -1},
			{addr: 0x2001cdd2, size: 4, signed: false, idx: -1},
			{addr: 0x2001cdd6, size: 4, signed: false, idx: -1},
			{addr: 0x2001cde1, size: 4, signed: false, idx: -1},
			{addr: 0x2001cde5, size: 4, signed: false, idx: -1},
			{addr: 0x2001ce44, size: 4, signed: false, idx: -1},
			{addr: 0x2001ce4d, size: 1, signed: false, idx: -1},
			{addr: 0x2001d108, size: 4, signed: false, idx: -1},
			{addr: 0x2001d114, size: 1, signed: false, idx: -1},
			{addr: 0x2001d115, size: 1, signed: false, idx: -1},
			{addr: 0x2001d116, size: 1, signed: false, idx: -1},
			{addr: 0x2001d12b, size: 4, signed: false, idx: -1},
			{addr: 0x2001d18b, size: 1, signed: false, idx: -1},
			{addr: 0x2001d508, size: 4, signed: false, idx: -1},
			{addr: 0x2001d514, size: 1, signed: false, idx: -1},
			{addr: 0x2001d515, size: 1, signed: false, idx: -1},
			{addr: 0x2001d516, size: 1, signed: false, idx: -1},
			{addr: 0x2001d52b, size: 4, signed: false, idx: -1},
			{addr: 0x2001d58b, size: 1, signed: false, idx: -1},
			{addr: 0x2001c5cb, size: 1, signed: false, idx: -1},
			{addr: 0x2001c5d0, size: 1, signed: false, idx: -1},
			{addr: 0x2001c5e0, size: 1, signed: false, idx: -1},
			{addr: 0x2001c5e1, size: 1, signed: false, idx: -1},
			{addr: 0x2001c5e2, size: 1, signed: false, idx: -1},
			{addr: 0x2001c5e3, size: 1, signed: false, idx: -1},
			{addr: 0x2001c5e5, size: 1, signed: false, idx: -1},
			{addr: 0x2001c5e6, size: 1, signed: false, idx: -1},
			{addr: 0x2001c6b9, size: 1, signed: false, idx: -1},
			{addr: 0x2001c6c0, size: 1, signed: false, idx: -1},
			{addr: 0x2001c749, size: 1, signed: false, idx: -1},
			{addr: 0x2001c75f, size: 4, signed: false, idx: -1},
			{addr: 0x2001c64d, size: 1, signed: false, idx: -1},
			{addr: 0x2001c64e, size: 1, signed: false, idx: -1},
			{addr: 0x2001c64f, size: 1, signed: false, idx: -1},
			{addr: 0x2001c650, size: 1, signed: false, idx: -1},
			{addr: 0x2001c651, size: 1, signed: false, idx: -1},
			{addr: 0x2001c652, size: 1, signed: false, idx: -1},
			{addr: 0x2001c653, size: 1, signed: false, idx: -1},
			{addr: 0x2001c654, size: 1, signed: false, idx: -1},
			{addr: 0x2001c655, size: 1, signed: false, idx: -1},
			{addr: 0x2001c656, size: 1, signed: false, idx: -1},
			{addr: 0x2001c5e7, size: 1, signed: false, idx: -1},
			{addr: 0x2001c5ea, size: 1, signed: false, idx: -1},
			{addr: 0x2001c5fc, size: 2, signed: false, idx: -1},
			{addr: 0x2001c609, size: 1, signed: false, idx: -1},
			{addr: 0x2001c60c, size: 1, signed: false, idx: -1},
			{addr: 0x2001c61e, size: 2, signed: false, idx: -1},
			{addr: 0x2001c62b, size: 1, signed: false, idx: -1},
			{addr: 0x2001c62e, size: 1, signed: false, idx: -1},
			{addr: 0x2001c640, size: 2, signed: false, idx: -1},
			{addr: 0x2001c8cb, size: 1, signed: false, idx: -1},
			{addr: 0x2001c8d0, size: 1, signed: false, idx: -1},
			{addr: 0x2001c8e0, size: 1, signed: false, idx: -1},
			{addr: 0x2001c8e1, size: 1, signed: false, idx: -1},
			{addr: 0x2001c8e2, size: 1, signed: false, idx: -1},
			{addr: 0x2001c8e3, size: 1, signed: false, idx: -1},
			{addr: 0x2001c8e5, size: 1, signed: false, idx: -1},
			{addr: 0x2001c8e6, size: 1, signed: false, idx: -1},
			{addr: 0x2001c9b9, size: 1, signed: false, idx: -1},
			{addr: 0x2001c9c0, size: 1, signed: false, idx: -1},
			{addr: 0x2001ca49, size: 1, signed: false, idx: -1},
			{addr: 0x2001ca5f, size: 4, signed: false, idx: -1},
			{addr: 0x2001c94d, size: 1, signed: false, idx: -1},
			{addr: 0x2001c94e, size: 1, signed: false, idx: -1},
			{addr: 0x2001c94f, size: 1, signed: false, idx: -1},
			{addr: 0x2001c950, size: 1, signed: false, idx: -1},
			{addr: 0x2001c951, size: 1, signed: false, idx: -1},
			{addr: 0x2001c952, size: 1, signed: false, idx: -1},
			{addr: 0x2001c953, size: 1, signed: false, idx: -1},
			{addr: 0x2001c954, size: 1, signed: false, idx: -1},
			{addr: 0x2001c955, size: 1, signed: false, idx: -1},
			{addr: 0x2001c956, size: 1, signed: false, idx: -1},
			{addr: 0x2001c8e7, size: 1, signed: false, idx: -1},
			{addr: 0x2001c8ea, size: 1, signed: false, idx: -1},
			{addr: 0x2001c8fc, size: 2, signed: false, idx: -1},
			{addr: 0x2001c909, size: 1, signed: false, idx: -1},
			{addr: 0x2001c90c, size: 1, signed: false, idx: -1},
			{addr: 0x2001c91e, size: 2, signed: false, idx: -1},
			{addr: 0x2001c92b, size: 1, signed: false, idx: -1},
			{addr: 0x2001c92e, size: 1, signed: false, idx: -1},
			{addr: 0x2001c940, size: 2, signed: false, idx: -1},
			{addr: 0x200007e5, size: 1, signed: false, idx: -1},
			{addr: 0x20000707, size: 1, signed: false, idx: -1},
			{addr: 0x200007f2, size: 4, signed: false, idx: -1},
			{addr: 0x200007f6, size: 4, signed: false, idx: -1},
			{addr: 0x200009a4, size: 1, signed: false, idx: -1},
			{addr: 0x200008c6, size: 1, signed: false, idx: -1},
			{addr: 0x200009b1, size: 4, signed: false, idx: -1},
			{addr: 0x200009b5, size: 4, signed: false, idx: -1},
			{addr: 0x20000b63, size: 1, signed: false, idx: -1},
			{addr: 0x20000a85, size: 1, signed: false, idx: -1},
			{addr: 0x20000b70, size: 4, signed: false, idx: -1},
			{addr: 0x20000b74, size: 4, signed: false, idx: -1},
			{addr: 0x20000d22, size: 1, signed: false, idx: -1},
			{addr: 0x20000c44, size: 1, signed: false, idx: -1},
			{addr: 0x20000d2f, size: 4, signed: false, idx: -1},
			{addr: 0x20000d33, size: 4, signed: false, idx: -1},
			{addr: 0x20004dbd, size: 1, signed: false, idx: -1},
			{addr: 0x20004cdf, size: 1, signed: false, idx: -1},
			{addr: 0x20004dca, size: 4, signed: false, idx: -1},
			{addr: 0x20004dce, size: 4, signed: false, idx: -1},
			{addr: 0x20006eea, size: 1, signed: false, idx: -1},
			{addr: 0x20006e0c, size: 1, signed: false, idx: -1},
			{addr: 0x20006ef7, size: 4, signed: false, idx: -1},
			{addr: 0x20006efb, size: 4, signed: false, idx: -1},
			{addr: 0x2000d5ef, size: 1, signed: false, idx: -1},
			{addr: 0x2000d511, size: 1, signed: false, idx: -1},
			{addr: 0x2000d5fc, size: 4, signed: false, idx: -1},
			{addr: 0x2000d600, size: 4, signed: false, idx: -1},
			{addr: 0x20010a51, size: 1, signed: false, idx: -1},
			{addr: 0x20010973, size: 1, signed: false, idx: -1},
			{addr: 0x20010a5e, size: 4, signed: false, idx: -1},
			{addr: 0x20010a62, size: 4, signed: false, idx: -1},
			{addr: 0x20000300, size: 4, signed: false, idx: -1},
			{addr: 0x20000304, size: 4, signed: false, idx: -1},
			{addr: 0x20000308, size: 4, signed: false, idx: -1},
			{addr: 0x2000030c, size: 4, signed: false, idx: -1},
			{addr: 0x20000310, size: 4, signed: false, idx: -1},
			{addr: 0x20000314, size: 4, signed: false, idx: -1},
			{addr: 0x20000318, size: 4, signed: false, idx: -1},
			{addr: 0x2000031c, size: 4, signed: false, idx: -1},
			{addr: 0x20000320, size: 4, signed: false, idx: -1},
			{addr: 0x20000324, size: 4, signed: false, idx: -1},
			{addr: 0x20000328, size: 4, signed: false, idx: -1},
			{addr: 0x2000032c, size: 4, signed: false, idx: -1},
			{addr: 0x20000330, size: 4, signed: false, idx: -1},
			{addr: 0x20000334, size: 4, signed: false, idx: -1},
			{addr: 0x20000338, size: 4, signed: false, idx: -1},
			{addr: 0x2000033c, size: 4, signed: false, idx: -1},
			{addr: 0x20000340, size: 4, signed: false, idx: -1},
			{addr: 0x20000344, size: 4, signed: false, idx: -1},
			{addr: 0x20000348, size: 4, signed: false, idx: -1},
			{addr: 0x2000034c, size: 4, signed: false, idx: -1},
			{addr: 0x20000350, size: 4, signed: false, idx: -1},
			{addr: 0x20000354, size: 4, signed: false, idx: -1},
			{addr: 0x20000358, size: 4, signed: false, idx: -1},
			{addr: 0x2000035c, size: 4, signed: false, idx: -1},
			{addr: 0x20000360, size: 4, signed: false, idx: -1},
			{addr: 0x20000364, size: 4, signed: false, idx: -1},
			{addr: 0x20000368, size: 4, signed: false, idx: -1},
			{addr: 0x2000036c, size: 4, signed: false, idx: -1},
			{addr: 0x20000370, size: 4, signed: false, idx: -1},
			{addr: 0x20000374, size: 4, signed: false, idx: -1},
			{addr: 0x20000378, size: 4, signed: false, idx: -1},
			{addr: 0x2000037c, size: 4, signed: false, idx: -1},
			{addr: 0x20000380, size: 4, signed: false, idx: -1},
			{addr: 0x20000384, size: 4, signed: false, idx: -1},
			{addr: 0x20000388, size: 4, signed: false, idx: -1},
			{addr: 0x2000038c, size: 4, signed: false, idx: -1},
			{addr: 0x20000390, size: 4, signed: false, idx: -1},
			{addr: 0x20000394, size: 4, signed: false, idx: -1},
			{addr: 0x20000398, size: 4, signed: false, idx: -1},
			{addr: 0x2000039c, size: 4, signed: false, idx: -1},
		},
		out: []mstFld{
			{addr: 0x2001cde1, size: 4, idx: -1},
			{addr: 0x2001cde5, size: 4, idx: -1},
			{addr: 0x2001cdba, size: 4, idx: -1},
			{addr: 0x2001cdbe, size: 4, idx: -1},
			{addr: 0x2001cdc2, size: 4, idx: -1},
			{addr: 0x2001cdc6, size: 4, idx: -1},
			{addr: 0x2001cdca, size: 4, idx: -1},
			{addr: 0x2001cdce, size: 4, idx: -1},
			{addr: 0x2001cdd2, size: 4, idx: -1},
			{addr: 0x2001cdd6, size: 4, idx: -1},
			{addr: 0x2001cf00, size: 4, idx: -1},
			{addr: 0x2001cf04, size: 1, idx: -1},
			{addr: 0x2001cf05, size: 4, idx: -1},
			{addr: 0x2001cf09, size: 4, idx: -1},
			{addr: 0x2001cf11, size: 4, idx: -1},
			{addr: 0x2001d04b, size: 1, idx: -1},
			{addr: 0x68d524, size: 1, idx: -1},
			{addr: 0x68d525, size: 1, idx: -1},
			{addr: 0x20000100, size: 4, idx: -1},
			{addr: 0x20000104, size: 4, idx: -1},
			{addr: 0x20000108, size: 4, idx: -1},
			{addr: 0x2000010c, size: 4, idx: -1},
			{addr: 0x20000110, size: 4, idx: -1},
			{addr: 0x20000114, size: 4, idx: -1},
			{addr: 0x20000118, size: 4, idx: -1},
			{addr: 0x2000011c, size: 4, idx: -1},
			{addr: 0x20000120, size: 4, idx: -1},
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
		},
	},
	stub: [mchStubs]macStub{
		mchLog:   {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20000140, 6}, {3, 4, 0x20000158, 6}}, cnt: 0x20000100},
		mchUI:    {ret: 0x20000300, retN: 10, rec: [][4]uint32{{1, 4, 0x20000170, 10}, {2, 4, 0x20000198, 10}, {3, 4, 0x200001c0, 10}}, cnt: 0x20000104},
		mchAvail: {ret: 0x20000328, retN: 3, rec: [][4]uint32{{1, 4, 0x200001e8, 3}, {2, 4, 0x200001f4, 3}, {3, 4, 0x20000200, 3}}, cnt: 0x20000108},
		mchCand:  {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000020c, 11}, {2, 4, 0x20000238, 11}}, cnt: 0x2000010c},
		mchTgt:   {ret: 0x20000338, retN: 0, rec: [][4]uint32{{1, 4, 0x20000264, 1}, {2, 4, 0x20000268, 1}}, cnt: 0x20000110},
		mchSpell: {ret: 0x2000035c, retN: 0, rec: [][4]uint32{{1, 4, 0x2000026c, 1}, {2, 4, 0x20000270, 1}}, cnt: 0x20000114},
		mchPick:  {ret: 0x20000360, retN: 0, rec: [][4]uint32{{1, 4, 0x20000274, 1}, {2, 4, 0x20000278, 1}, {3, 4, 0x2000027c, 1}, {4, 4, 0x20000280, 1}}, cnt: 0x20000118},
		mchPower: {ret: 0x20000398, retN: 0, rec: [][4]uint32{{1, 4, 0x20000284, 1}, {2, 4, 0x20000288, 1}, {3, 4, 0x2000028c, 1}}, cnt: 0x2000011c},
		mchArea:  {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000290, 1}, {2, 4, 0x20000294, 1}}, cnt: 0x20000120},
	},
	candW: 0x20000334, tgtW: 0x2000033c, pickW: 0x20000364, areaW: 0x2000039c,
}

func mchNames_in() []string {
	return []string{"g94", "g524", "g525", "g2da4", "party.x", "party.y", "party.z", "cam.r", "a0.x", "a0.y", "a0.z", "a0.r", "a1.x", "a1.y", "a1.z", "a1.r", "a2.x", "a2.y", "a2.z", "a2.r", "as.s12", "as.s19", "c.f0", "c.f4", "c.f5", "c.f9", "c.f11", "c.146", "c.14b", "c.152", "mlh.n", "rlh.n", "rel0.b5", "rel1.b5", "rel2.b5", "g0.c8", "g1.c8", "m.grp", "m.tid", "m.15", "m.16", "m.maxhp", "m.hp", "m.2f", "m.33", "m.77", "m.87", "m.8b", "m.107", "m.24e", "m.tgt0", "m.tgt1", "m.tgt2", "m.tgt3", "m.tgt4", "m.tgt5", "m.tgt6", "m.tgt7", "m.2e1", "m.2e5", "m.344", "m.34d", "m1.tid", "m1.14", "m1.15", "m1.16", "m1.2b", "m1.8b", "m2.tid", "m2.14", "m2.15", "m2.16", "m2.2b", "m2.8b", "T0.cb", "T0.d0", "T0.e0", "T0.e1", "T0.e2", "T0.e3", "T0.e5", "T0.e6", "T0.b9", "T0.c0", "T0.249", "T0.25f", "T0.sp0", "T0.sp1", "T0.sp2", "T0.sp3", "T0.sp4", "T0.sp5", "T0.sp6", "T0.sp7", "T0.sp8", "T0.sp9", "T0.act0", "T0.rng0", "T0.w0", "T0.act1", "T0.rng1", "T0.w1", "T0.act2", "T0.rng2", "T0.w2", "T1.cb", "T1.d0", "T1.e0", "T1.e1", "T1.e2", "T1.e3", "T1.e5", "T1.e6", "T1.b9", "T1.c0", "T1.249", "T1.25f", "T1.sp0", "T1.sp1", "T1.sp2", "T1.sp3", "T1.sp4", "T1.sp5", "T1.sp6", "T1.sp7", "T1.sp8", "T1.sp9", "T1.act0", "T1.rng0", "T1.w0", "T1.act1", "T1.rng1", "T1.w1", "T1.act2", "T1.rng2", "T1.w2", "s1.126", "s1.48", "s1.133", "s1.137", "s2.126", "s2.48", "s2.133", "s2.137", "s3.126", "s3.48", "s3.133", "s3.137", "s4.126", "s4.48", "s4.133", "s4.137", "s29.126", "s29.48", "s29.133", "s29.137", "s3c.126", "s3c.48", "s3c.133", "s3c.137", "s77.126", "s77.48", "s77.133", "s77.137", "s95.126", "s95.48", "s95.133", "s95.137", "ui0", "ui1", "ui2", "ui3", "ui4", "ui5", "ui6", "ui7", "ui8", "ui9", "av0", "av1", "av2", "cand.n", "tgt.ret", "tgt.w0", "tgt.w1", "tgt.w2", "tgt.w3", "tgt.w4", "tgt.w5", "tgt.w6", "tgt.w7", "spl.ret", "pick.ret", "pick.w0", "pick.w1", "pick.w2", "pick.w3", "pick.w4", "pick.w5", "pick.w6", "pick.w7", "pick.w8", "pick.w9", "pick.w10", "pick.w11", "pick.w12", "pwr.ret", "area.n"}
}

func mchNames_out() []string {
	return []string{"M.2e1", "M.2e5", "M.tgt0", "M.tgt1", "M.tgt2", "M.tgt3", "M.tgt4", "M.tgt5", "M.tgt6", "M.tgt7", "c.f0", "c.f4", "c.f5", "c.f9", "c.f11", "c.14b", "g524", "g525", "n_log", "n_ui", "n_avail", "n_cand", "n_tgt", "n_spell", "n_pick", "n_power", "n_area", "r_log_a2_0", "r_log_a2_1", "r_log_a2_2", "r_log_a2_3", "r_log_a2_4", "r_log_a2_5", "r_log_a3_0", "r_log_a3_1", "r_log_a3_2", "r_log_a3_3", "r_log_a3_4", "r_log_a3_5", "r_ui_a1_0", "r_ui_a1_1", "r_ui_a1_2", "r_ui_a1_3", "r_ui_a1_4", "r_ui_a1_5", "r_ui_a1_6", "r_ui_a1_7", "r_ui_a1_8", "r_ui_a1_9", "r_ui_a2_0", "r_ui_a2_1", "r_ui_a2_2", "r_ui_a2_3", "r_ui_a2_4", "r_ui_a2_5", "r_ui_a2_6", "r_ui_a2_7", "r_ui_a2_8", "r_ui_a2_9", "r_ui_a3_0", "r_ui_a3_1", "r_ui_a3_2", "r_ui_a3_3", "r_ui_a3_4", "r_ui_a3_5", "r_ui_a3_6", "r_ui_a3_7", "r_ui_a3_8", "r_ui_a3_9", "r_avail_a1_0", "r_avail_a1_1", "r_avail_a1_2", "r_avail_a2_0", "r_avail_a2_1", "r_avail_a2_2", "r_avail_a3_0", "r_avail_a3_1", "r_avail_a3_2", "r_cand_a1_0", "r_cand_a1_1", "r_cand_a1_2", "r_cand_a1_3", "r_cand_a1_4", "r_cand_a1_5", "r_cand_a1_6", "r_cand_a1_7", "r_cand_a1_8", "r_cand_a1_9", "r_cand_a1_10", "r_cand_a2_0", "r_cand_a2_1", "r_cand_a2_2", "r_cand_a2_3", "r_cand_a2_4", "r_cand_a2_5", "r_cand_a2_6", "r_cand_a2_7", "r_cand_a2_8", "r_cand_a2_9", "r_cand_a2_10", "r_tgt_a1_0", "r_tgt_a2_0", "r_spell_a1_0", "r_spell_a2_0", "r_pick_a1_0", "r_pick_a2_0", "r_pick_a3_0", "r_pick_a4_0", "r_power_a1_0", "r_power_a2_0", "r_power_a3_0", "r_area_a1_0", "r_area_a2_0"}
}

// mchReal runs the registered Go nodes of the D4 callees. The live world decides
// the values the original hands them (spell, attack index, flag, template byte,
// power level inputs); everything else of a callee's own world is the parked
// default below. So F and Emu differ wherever the case's world differs from that.
func (w *mchWorld) spellWords(x []int64, spell uint32) {
	rec := w.spellRecord(spell)
	x[0], x[1], x[2], x[3], x[4] = int64(spell), int64(w.u32(rec+0x137)), int64(w.u8(rec+0x48)), int64(w.u8(rec+0x126)), int64(w.u32(rec+0x133))
	x[5], x[6], x[7] = int64(int32(w.u32(rec+0x49))), int64(int32(w.u32(rec+0x56))), int64(int32(w.u32(rec+0x12f)))
	x[8], x[9], x[10] = int64(w.u8(0x683f94)), int64(w.u8(0x652da4)), int64(w.u8(0x68d524))
}

var mchReal = mchCallees{
	pick: func(w *mchWorld, m, a, b, flag uint32) bool {
		w.call(mchPick, m, a, b, flag)
		cmb := w.u32(m + 0x10)
		x := append([]int64(nil), mchParkPick[:]...)
		x[0], x[1], x[2], x[3] = int64(a), int64(b), int64(flag), 0
		x[4] = int64(w.u8(w.tmpl(m) + 0xe5)) // template slot 0 carries the live action byte
		x[8], x[9] = rpkListPtr, 0
		x[108], x[109], x[110] = int64(w.u32(cmb+5)), int64(w.u32(cmb+9)), int64(w.u32(cmb+0x11))
		o := monsterActionRandomPickNode(x, w.r)
		w.put(m+0x2e1, 4, uint64(uint32(o[0])))
		w.put(m+0x2e5, 4, uint64(uint32(o[1])))
		for i := uint32(0); i < 8; i++ {
			w.put(m+0x2ba+4*i, 4, uint64(uint32(o[2+i])))
		}
		w.put(cmb+5, 4, uint64(uint32(o[10])))
		w.put(cmb+9, 4, uint64(uint32(o[11])))
		w.put(cmb+0x11, 4, uint64(uint32(o[12])))
		return byte(o[13]) != 0
	},
	tgt: func(w *mchWorld, m, spell uint32) bool {
		w.call(mchTgt, m, spell)
		// 0x5326f0 rebuilds the 0x533320 list itself: its world is the candidate
		// builder's parked world without the two list inputs (mstRndDropIn), so
		// the list 0x534a40 saw non-empty is the list 0x5326f0 draws from.
		x := append(append([]int64(nil), mchParkCand[:mstRndDropIn[0]]...), mchParkCand[mstRndDropIn[1]+1:]...)
		w.spellWords(x, spell)
		o := monsterSpellTargetRandomNode(x, w.r)
		n := len(o)
		if byte(o[n-1]) == 0 {
			return false
		}
		for i := 0; i < 8; i++ {
			w.put(m+0x2ba+4*uint32(i), 4, uint64(uint32(o[n-9+i])))
		}
		return true
	},
	spell: func(w *mchWorld, m, t uint32) uint32 {
		w.call(mchSpell, m, t)
		return uint32(monsterSpellWeightedChoiceNode(append([]int64(nil), mchParkSpell[:]...), w.r)[0])
	},
	cand: func(w *mchWorld, m, spell, list uint32) {
		w.call(mchCand, m, spell, list)
		x := append([]int64(nil), mchParkCand[:]...)
		w.spellWords(x, spell)
		x[mstRndDropIn[0]] = 0 // 0x532550 and 0x534a40 hand it a fresh (empty) list
		w.put(list+4, 4, uint64(uint32(monsterSpellTargetCandidatesNode(x, w.r)[0])))
	},
	avail: func(w *mchWorld, m, i, a5 uint32) bool {
		w.call(mchAvail, m, i, a5)
		x := append([]int64(nil), mchParkAvail[:]...)
		x[0], x[1] = int64(i), int64(a5&0xff)
		return byte(monsterAttackTargetAvailableNode(x, w.r)[0]) != 0
	},
	power: func(w *mchWorld, m, t, spell uint32) uint32 {
		w.call(mchPower, m, t, spell)
		rec := w.spellRecord(spell)
		return uint32(monsterSpellPowerLevelNode([]int64{int64(int32(w.u32(m + 0x2f9))), int64(int32(w.u32(t + 0x26f))), int64(w.u32(m + 8)),
			int64(spell), int64(int32(w.u32(rec + 0x49))), int64(int32(w.u32(rec + 0x56)))}, w.r)[0])
	},
	area: func(w *mchWorld, spell, power, list uint32) {
		w.call(mchArea, spell, power)
		x := append([]int64(nil), mchParkArea[:]...)
		x[13], x[18], x[29] = int64(spell), int64(power), 0
		w.put(list+4, 4, uint64(uint32(spellAreaTargetSelectionNode(x, w.r)[1]))) // the character list is arg 6
	},
}

// Parked default worlds of the real callees (one boundary row of each callee case),
// the inputs the live world does not decide; the overrides are in mchReal.

var mchParkPick = [645]int64{
	0, 0, 1, 0, 7, 8, 9, 10, 0, 5, 0, 4096,
	8192, 12288, 12289, 12290, 12291, 12292, 12293, 12294, 12295, 1, 4097, 8193,
	12298, 12299, 12300, 12301, 12302, 12303, 12304, 12305, 2, 4098, 8194, 12308,
	12309, 12310, 12311, 12312, 12313, 12314, 12315, 8, 4099, 8195, 12318, 12319,
	12320, 12321, 12322, 12323, 12324, 12325, 0, 4100, 8196, 12328, 12329, 12330,
	12331, 12332, 12333, 12334, 12335, 1, 4101, 8197, 12338, 12339, 12340, 12341,
	12342, 12343, 12344, 12345, 4294967295, 4102, 8198, 12348, 12349, 12350, 12351, 12352,
	12353, 12354, 12355, 3, 4103, 8199, 12358, 12359, 12360, 12361, 12362, 12363,
	12364, 12365, 286331153, 572662306, 858993408, 858993409, 858993410, 858993411, 858993412, 858993413, 858993414, 858993415,
	1145324612, 1431655765, 1717986918, 0, 0, 0, 0, 1193033728, 1140457472, 1140457472, 0, 0,
	1, 536879936, 4294967295, 4, 536880032, 536881056, 536882080, 536884128, 1, 536884128, 536885152, 100,
	0, 1, 1, 1, 1, 3, 1140457472, 1143930880, 0, 0, 101, 1,
	1, 0, 0, 0, 3, 0, 0, 0, 1120403456, 102, 1, 5,
	1, 2, 77, 0, 1153138688, 0, 0, 1157234688, 104, 0, 5, 1,
	0, 1, 0, 1140457472, 0, 1133903872, 1120403456, 105, 0, 1, 1, 2,
	1, 1, 1153138688, 0, 1133903872, 1120403456, 105, 0, 0, 1, 2, 77,
	0, 1153138688, 1143930880, 0, 0, 5, 255, 0, 1, 1, 0, 1,
	1, 3, 4294967295, 4294967295, 9, 9, 0, 9, 0, 101, 1, 0,
	1, 102, 0, 1, 3, 100, 2, 3, 3, 4, 1, 1,
	0, 2, 0, 2, 1, 3, 256, 1, 1, 256, 1, 4,
	853, 4, 1, 1, 3, 2, 255, 2, 1, 9, 682, 0,
	1, 1, 1, 4, 256, 0, 0, 0, 100, 0, 1, 0,
	0, 1073741823, 0, 1, 2147483648, 0, 50, 1, 0, 10, 0, 133,
	0, 1, 0, 0, 133, -1, 1, 0, 18, 50, 1, 1,
	10, 0, 4294967295, 1, 1, 10, 0, 50, 1, 0, 0, 512,
	0, 256, 0, 0, 256, 512, 0, 512, 256, 1, 0, 2,
	1, 1, 0, 258, 258, 257, 257, 0, 0, 3, 257, 1,
	2, 2, 1, 2, 0, 3, 258, 257, 0, 1, 256, 257,
	256, 257, 0, 0, 3, 0, 2, 3, 0, 0, 1, 256,
	2, 0, 1, 257, 257, 0, 3, 257, 3, 1, 257, 0,
	3, 2, 0, 0, 1, 39, 100, 0, 0, 11, 0, 10,
	2, 1, 39, 100, 1, 0, 11, 2, 0, 0, 0, 1,
	0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 1, 0, 1, 1, 0, 1, 0, 10, 20,
	0, 2, 10, 1, 0, 1, 2, 11, 1, 0, 1, 2,
	20, 1, 1, 2, 21, 1, 1, 0, 0, 536977408, 4, 255,
	305, 4, 4, 0, 4, 2, 128, 3, 127, 3, 255, 0,
	4, 255, 7, 5, 7, 2, 4, 1, 255, 2, 4, 4,
	2, 4294967295, 17, 2147483647, 1, 1, 4294967295, 19, 0, 1, 1, 256,
	18, 0, 0, 1, 100, 0, 0, 255, 2, 2147483648, 19, 0,
	0, 0, 100, 19, -2147483648, 255, 1, 4294967295, 1, -2147483648, 0, 1,
	1, 19, 0, 0, 3330239054, 1171044607, 3335716808, 1144976842, 3, 1, 1, 2,
	5, 100, 0, 1, 1, 1, 255, 255, 2, 0, 3, 3,
	2, 2, 2, 1, 1, 0, 1, 255, 1, 255, 0, 3,
	1, 1, 1, 1, 0, 3315563851, 1185216605, 3335048584, 3256122901, 3, 0, 1,
	1, 2, 1, 2147483648, 0, 3322014300, 1168864561, 1174643437, 1152611808, 0, 1, 1,
	1, 0, 4294967295, 4294967295, 5, 1183437705, 1187824152, 1180080956, 1168522540, 0, 2, 1,
	1, 255, 0, 0, 0, 1189303982, 1184962667, 1186763875, 3242314679, 1, 3, 1,
	0, 0, 3, 255, 2, 0, 2, 1, 0, 1, 256, 1,
	0, 0, 0, 1, 1, 3, 0, 0, 2, 3, 4, 0,
	1, 2, 3, 0, 0, 1092616192, 0, 0, 0, 0, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 0, 0, 0, 0, 0, 0, 0, 0,
}

var mchParkCand = [418]int64{
	0, 1, 0, 1, 0, 0, 9, 2, 1, 0, 1, 0,
	1, 0, 96, 0, 1, 0, 5, 1, 5, 0, 1, 0,
	1, 255, 0, 0, 1, 1, 1, 1, 0, 1, 9, 0,
	0, 1, 0, 1, 1, 1, 0, 0, 2, 1, 0, 0,
	0, 0, 0, 0, 1, 1, 1, 1, 0, 0, 0, 0,
	1, 1, 0, 0, 0, 2, 1, 1, 0, 0, 4294967295, 1,
	1, 1, 1, 0, 1, 0, 1, 1, 0, 1, 0, 2,
	0, 2, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0,
	1, 0, 1140457472, 1140457472, 0, 1056964608, 1073741824, 21, 3, 1065353216, 1178816512, 1065353216,
	1056964608, 1056964608, 21, 0, 1157234688, 1187205120, 1148846080, 1056964608, 1065353216, 0, 0, 1148846080,
	1157234688, 1140457472, 1065353216, 1056964608, 0, 3, 1065353216, 0, 1178816512, 1073741824, 1140457472, 0,
	1065353216, 1, 2, 20, 1, 2, 2, 0, 0, 1, 0, 0,
	0, 0, 0, 1, 2, 1, 0, 1, 1, 0, 0, 0,
	0, 1, 1, 0, 0, 1, 2, 0, 1, 0, 2, 2,
	1, 1, 0, 0, 0, 0, 1, 1, 2, 0, 0, 1,
	2, 1, 0, 0, 1, 0, 1, 0, 0, 1, 0, 17,
	0, 2, 2, 2, 1, 0, 0, 1, 1, 1, 1, 0,
	0, 17, 0, 2, 0, 1, 1, 0, 0, 1, 0, 1,
	1, 0, 0, 0, 0, 2, 0, 2, 1, 1, 0, 0,
	0, 0, 1, 0, 0, 17, 0, 2, 0, 0, 0, 1,
	0, 1, 0, 0, 1, 0, 2, 2, 1, 1, 0, 2,
	1, 0, 2, 0, 1, 0, 1, 1, 0, 1, 0, 0,
	1, 0, 0, 1, 0, 0, 1, 3, 3, 3, 0, 1,
	1, 2, 2, 3, 3, 0, 7, 0, 7, 3, 0, 0,
	0, 0, 0, 1065353216, 0, 0, 0, 0, 0, 0, 0, 1,
	5, 1065353216, 0, 2, 1, 3, -1, -1, 0, 1, 0, 7,
	0, 123, 456, 0, 1, 1, 1, 10, 0, 0, 0, 1,
	1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
	1, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0,
	0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0,
	0, 1, 10, 0, 0, 1, 10, 0, 0, 0, 0, 1,
	0, 8, 0, 8, 1065353216, 0, 1073741824, 0, 0, 2, 3, 4,
	0, 1, 2, 3, 0, 0, 1092616192, 0, 0, 0, 0, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 0, 0, 0, 0, 0, 0, 0, 0,
}

var mchParkSpell = [48]int64{
	0, 4, -1, 0, 0, 0, 0, 0, 119, 1, 1, 1,
	1, 1, 1, 1, 2, 1, 1, 1, 3, 1, 1, 1,
	4, 1, 1, 1, 5, 1, 1, 1, 6, 1, 1, 1,
	7, 1, 1, 1, 8, 1, 1, 1, 9, 1, 1, 1,
}

var mchParkAvail = [167]int64{
	0, 2, 536977408, 2, 255, 285, 3, 3, 128, 0, 128, 127,
	255, 128, 255, 255, 3, 2, 6, 0, 4, 4, 0, 1,
	1, 1, 2, 255, 255, 1, 1, 5, 0, 1, 255, 100,
	100, 1, 255, 1, 1, 4294967295, 0, 0, 1, 0, 19, 0,
	1, 1, 4294967295, 17, 0, 0, 1, 256, 0, 0, 0, 255,
	0, 5, 2147483647, 255, 1, 256, 5, 5, 1, 3280812390, 1148455726, 1138877779,
	1127565260, 0, 2, 1, 1, 2147483648, 3, 2, 1, 1, 255, 2,
	2, 0, 1, 0, 0, 0, 1, 1, 1, 2, 1, 255,
	255, 2, 255, 255, 1, 1, 0, 255, 2, 255, 3289483765, 1118330532,
	3289117787, 1127837258, 3, 1, 1, 2, 3, 100, 4294967295, 4294967295, 3281799931, 1131200771,
	3292844299, 1103572357, 3, 548, 1, 1, 2, 1, 5, 1, 1117569105, 1134437449,
	1144137000, 1128403420, 4294967295, 3, 1, 2, 1, 4294967295, 4294967295, 5, 1135326732, 3292777563,
	3291043006, 3255865161, 255, 4294967295, 1, 3, 0, 256, 1, 3, 0, 256,
	1, 1, 0, 1, 2, 1, 0, 0, 2, 3, 3,
}

var mchParkArea = [93]int64{
	0, 0, 0, 0, 0, 1065353216, 0, 0, 0, 0, 0, 0,
	0, 1, 0, 1065353216, 0, 2, 1, 1, 0, -1, 0, 1,
	0, 1, 0, 123, 456, 0, 1, 1, 1, 10, 0, 0,
	0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 1, 1, 1, 10, 0, 0, 1, 10, 0, 0, 1,
	10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1,
	10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 0,
	0, 1, 0, 8, 0, 8, 1065353216, 0, 1073741824,
}
