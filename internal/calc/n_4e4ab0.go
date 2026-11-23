package calc

import (
	"math"
	"math/big"
)

// monster_identification_growth 0x4e4ab0: walk the group list at *0x683fb1. For
// each group record g (0x5101b0) that is alive (+0x28), still has an unfinished
// stage (+0x2c identification or +0x2d knowledge growth), passes the party-view
// gate (flag byte 0x683f94, group +0x29) and has a member the party can see
// (0x511b40), then:
//   - identification (+0x2c == 0): every present, able party member (presence
//     byte at *0x685178 stride 0x106; character words +0xb11 != 0, +0xb01 < 11,
//     +0xa31 == 0, record at *0x685174 stride 0x1862) rolls R(100) against
//     3*(K - level) + word(+0xfd)/10 + word(+0x4c1), K being the character's known
//     count byte at +0x12b6+species and level the species template byte +0x24f.
//     A roll below the value scores value-roll; if the character's byte +0x4bb is
//     set the skill-0x15 growth 0x553f10(C, 0x15, 10, 0) runs. The best scorer
//     (first on ties) is announced: 0x590950 and the group-name message 0x58aad0.
//   - growth (+0x2d == 0): every such member's K += group +4, capped at 255.
//
// After the list, an announcement refreshes the battle bar 0x565420.
//
// Emu address-stubs 0x553f10 (D4), 0x58aad0 (排除) and the library/object
// callees listed in the case; the stub counts and recorded arguments are
// outputs. F runs the real naturalSkillGrowthNode for every 0x553f10 call and
// feeds its +0x4bb / +0x4c1 results forward (the node rereads them on later
// passes). F's extra inputs are the 0x553f10 vector (in[nmiOwn:]) the call is
// built from; only the skill-0x15 cells are per character, every other cell
// is shared by the eight calls.
const (
	nmiGroups    = 2 // fixture cap: group records
	nmiMembers   = 2 // fixture cap: members per group
	nmiTemplates = 3 // fixture cap: template ids 0..2 are preloaded in the cache
	nmiParty     = 8 // loop bound 0x830 / 0x106
	nmiSlotSize  = 0x106
	nmiBattleOr  = 0x20000 // 0x565420 sets this bit in the battle object's +0xf4
)

func init() {
	Nodes["004e4ab0"] = Node{Addr: "004e4ab0", In: nmiNamesIn(), Out: nmiNamesOut(),
		F:   monsterIdentificationGrowthNode,
		Emu: func(in []int64, r Rand) []int64 { return nmiRun(in, r, false) }}
}

func monsterIdentificationGrowthNode(in []int64, r Rand) []int64 { return nmiRun(in, r, true) }

type nmiState struct {
	w    *mstWorld
	real bool
	cur  n553In       // F: running 0x553f10 vector
	slot [8]*nmiCells // F: per-character skill cells
}

// nmiCells are the per-character arrays of the 0x553f10 vector.
type nmiCells struct{ raw, cnt, f19d, f1af, f1b0, eff [n553Skills]int64 }

func nmiRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(nmiSpec.in) {
		panic("monster identification growth: wrong input count")
	}
	w := mstBuild(&nmiSpec, in)
	w.r = r
	n := &nmiState{w: w, real: real}
	if real {
		n.cur = n553Parse(in[nmiOwn:])
	}
	n.run()
	out := make([]int64, len(nmiSpec.out))
	for i, f := range nmiSpec.out {
		out[i] = w.read(f, in)
	}
	return out
}

// nmiNext bumps a stub call counter and returns the call index; past the
// fixture's recorded calls the case cannot compare, so it panics.
func nmiNext(w *mstWorld, cnt uint32, limit uint32) uint32 {
	k := w.u32(cnt)
	if k >= limit {
		panic("monster identification growth: more stub calls than the fixture records")
	}
	w.put(cnt, 4, uint64(k+1))
	return k
}

// nmiTemplate is 0x4e57c0 / 0x4e5720: the template cache is a direct array at
// 0x6840c7; a miss would load the template file (outside the fixture).
func nmiTemplate(w *mstWorld, id uint32) uint32 {
	if id >= 0x3e8 {
		panic("monster identification growth: template id past 0x3e7 (0x4e5749 asserts)")
	}
	if id >= nmiTemplates {
		panic("monster identification growth: template cache miss is outside the fixture (file load)")
	}
	t := w.u32(0x6840c7 + 4*id)
	if t == 0 {
		panic("monster identification growth: template cache entry is empty")
	}
	return t
}

func (n *nmiState) run() {
	w := n.w
	cnt := w.listCount(w.u32(0x683fb1))
	if cnt == 0 {
		return
	}
	if cnt > nmiGroups {
		panic("monster identification growth: group list longer than the fixture")
	}
	notice := false
	for i := uint32(0); i < w.listCount(w.u32(0x683fb1)); i++ { // 0x5e2c70 re-read each pass
		g := w.listAt(w.u32(0x683fb1), i) // 0x5101b0
		if g == 0 {
			panic("monster identification growth: group record missing (0x5101b0 asserts)")
		}
		if w.u8(g+0x28) == 0 || (w.u8(g+0x2c) != 0 && w.u8(g+0x2d) != 0) {
			continue
		}
		if w.u8(0x683f94) != 0 && w.u8(g+0x29) == 0 {
			continue
		}
		if !n.partySees(g) {
			continue
		}
		if w.u8(g+0x2c) == 0 && n.identify(g) {
			notice = true
		}
		if w.u8(g+0x2d) == 0 {
			sp := nmiSpecies(w, g)
			for c := uint32(0); c < nmiParty; c++ {
				if n.able(c) {
					k := w.charBase(c) + 0x12b6 + sp
					v := w.u32(g+4) + uint32(w.u8(k)) // 32-bit wrap
					if v > 0xff {
						v = 0xff
					}
					w.put(k, 1, uint64(v))
				}
			}
			w.put(g+0x2d, 1, 1)
		}
	}
	if notice && w.u32(0x68ec78) == 7 && nmiBattle(w) != 0 { // 0x565420
		b := w.u32(0x68edcc) + 0xf4
		w.put(b, 4, uint64(w.u32(b)|nmiBattleOr))
	}
}

// nmiBattle returns the battle object pointer (*0x68edcc); only the fixture
// object exists. 0x58b410 needs it in mode 7, so a null one cannot be modelled.
func nmiBattle(w *mstWorld) uint32 {
	b := w.u32(0x68edcc)
	if b != 0 && b != nmiFix {
		panic("monster identification growth: battle object outside the fixture")
	}
	return b
}

// nmiSpecies is the group's species/template id (+0x18); the K cells and the
// template cache only exist for the fixture's ids.
func nmiSpecies(w *mstWorld, g uint32) uint32 {
	sp := w.u32(g + 0x18)
	if sp >= nmiTemplates {
		panic("monster identification growth: species id past the fixture's template cache")
	}
	return sp
}

// able is the loop's character test: present, +0xb11 set, +0xb01 < 11, +0xa31 clear.
func (n *nmiState) able(c uint32) bool {
	w := n.w
	b := w.charBase(c)
	return w.u8(w.u32(0x685178)+c*nmiSlotSize) != 0 && w.u32(b+0xb11) != 0 && w.u32(b+0xb01) < 0xb && w.u32(b+0xa31) == 0
}

// identify is the +0x2c == 0 stage; it reports whether a best scorer was announced.
func (n *nmiState) identify(g uint32) bool {
	w := n.w
	sp := nmiSpecies(w, g)
	lvl := uint32(w.u8(nmiTemplate(w, sp) + 0x24f)) // 0x510180
	best, bestIdx := uint32(0), -1
	for c := uint32(0); c < nmiParty; c++ {
		if !n.able(c) {
			continue
		}
		b := w.charBase(c)
		v := (uint32(w.u8(b+0x12b6+sp))-lvl)*3 + w.u32(b+0xfd)/10 + w.u32(b+0x4c1)
		roll := int32(w.r.R(100)) // 0x40efa0
		if roll >= int32(v) {
			continue
		}
		d := v - uint32(roll)
		if w.u8(b+0x4bb) != 0 {
			n.grow(c)
		}
		if best < d {
			best, bestIdx = d, int(c)
		}
	}
	if bestIdx < 0 {
		return false
	}
	// 0x590950(best, fmt, name): the stub records the character-name pointer.
	k := nmiNext(w, nmiStubCnt58, nmiN58)
	n.rec58(k, 8, 0x64bf5c, w.charBase(uint32(bestIdx))+5)
	if w.u32(0x68ec78) == 7 { // 0x58b410 dereferences the battle object
		nmiBattle(w)
	}
	w.put(g+0x2c, 1, 1)
	vowel := n.vowel(g)
	text := w.u32(w.u32(0x68c09c) + 0x734)
	if w.u32(g+4) == 1 {
		text = w.u32(w.u32(0x68c09c) + 0x72c)
		if vowel {
			text = w.u32(w.u32(0x68c09c) + 0x730)
		}
	}
	k = nmiNext(w, nmiStubCnt58, nmiN58)
	n.rec58(k, 8, 0x614a3c, text)
	return true
}

func (n *nmiState) rec58(k uint32, a1, a2, a3 uint32) {
	w := n.w
	w.put(nmiStubA58_0+4*k, 4, uint64(a1))
	w.put(nmiStubA58_1+4*k, 4, uint64(a2))
	w.put(nmiStubA58_2+4*k, 4, uint64(a3))
}

// vowel: first letter of the group name (0x510280), upper-cased by the library
// (the stub's return is a case input), is one of A E I O U (jump table in the exe).
func (n *nmiState) vowel(g uint32) bool {
	w := n.w
	first := n.nameFirst(g)
	k := nmiNext(w, nmiStubCntUp, nmiNUp)
	w.put(nmiStubAUp+4*k, 4, uint64(first))
	c := w.u32(nmiStubRUp+4*k) & 0xffff
	idx := c - 0x41
	return idx <= 0x14 && w.u32(0x4e4d8c+4*uint32(w.u8(0x4e4d94+idx))) == 0x4e4c8a
}

// nameFirst is the first wide char of 0x510280(g) once +0x2c is set: the species
// template's "Al-%s" format for the special template, else the name at
// template + 0x30 * (g+4 != 1).
func (n *nmiState) nameFirst(g uint32) uint32 {
	w := n.w
	t := nmiTemplate(w, nmiSpecies(w, g))
	if uint32(w.u8(t+0x187))|uint32(w.u8(t+0x188))<<8 == 0x18d {
		if w.u32(0x6875ef) >= nmiParty {
			panic("monster identification growth: Al- character index past the party")
		}
		return uint32(w.u8(0x614b44)) | uint32(w.u8(0x614b45))<<8
	}
	dl := uint32(0)
	if w.u32(g+4) != 1 {
		dl = 1
	}
	p := t + 0x30*dl
	return uint32(w.u8(p)) | uint32(w.u8(p+1))<<8
}

// grow is the 0x553f10(C, 0x15, 10, 0) call at character c.
func (n *nmiState) grow(c uint32) {
	w := n.w
	k := nmiNext(w, nmiStubCnt553, nmiN553)
	b := w.charBase(c)
	for j, a := range [4]uint32{nmiStubA553_0, nmiStubA553_1, nmiStubA553_2, nmiStubA553_3} {
		w.put(a+4*k, 4, uint64([4]uint32{b, 0x15, 10, 0}[j]))
	}
	if !n.real {
		return
	}
	if n.slot[c] == nil {
		n.slot[c] = &nmiCells{raw: n.cur.raw, cnt: n.cur.cnt, f19d: n.cur.f19d, f1af: n.cur.f1af, f1b0: n.cur.f1b0, eff: n.cur.eff}
	}
	cl := n.slot[c]
	v := n.cur
	v.raw, v.cnt, v.f19d, v.f1af, v.f1b0, v.eff = cl.raw, cl.cnt, cl.f19d, cl.f1af, cl.f1b0, cl.eff
	v.slot, v.charAddr, v.s, v.n, v.quiet = int64(c), int64(b), 0x15, 10, 0
	v.f19d[0x15], v.eff[0x15] = int64(w.u8(b+0x4bb)), int64(w.u32(b+0x4c1))
	var vec []int64
	v.bind(func(_ string, p *int64, _ int) { vec = append(vec, *p) })
	res := naturalSkillGrowthNode(vec, w.r)
	var o n553Out
	i := 0
	o.bind(func(_ string, p *int64) { *p = res[i]; i++ })
	cl.raw, cl.cnt, cl.f19d, cl.f1af, cl.f1b0, cl.eff = o.raw, o.cnt, o.f19d, o.f1af, o.f1b0, o.eff
	w.put(b+0x4bb, 1, uint64(o.f19d[0x15]))
	w.put(b+0x4c1, 4, uint64(o.eff[0x15]))
	n.cur.f506d, n.cur.qcount, n.cur.qcap, n.cur.evSrc = o.f506d, o.qcount, o.qcap, o.evSrc
	n.cur.kraw, n.cur.keff, n.cur.uval, n.cur.u1bc, n.cur.k19e = o.kraw, o.keff, o.uval, o.u1bc, o.k19e
	n.cur.u64, n.cur.wf4, n.cur.wf8, n.cur.grid = o.u64, o.wf4, o.wf8, o.grid
}

// partySees is 0x511b40(g, 0): some member of the group's monster list is seen
// by the camera (0x4ca2a0) and passes the 0x4c7c00 test.
func (n *nmiState) partySees(g uint32) bool {
	w := n.w
	vec := w.u32(g + 8)
	cnt := w.listCount(vec)
	if cnt > nmiMembers {
		panic("monster identification growth: group member list longer than the fixture")
	}
	for i := uint32(0); i < cnt; i++ {
		m := w.findMonster(w.listAt(vec, i))
		obj := w.u32(m + 0xc)
		if n.inView(obj) && n.counts(obj) {
			return true
		}
	}
	return false
}

// inView is 0x4ca2a0: a flagged object asks testFlag (a case input); otherwise
// the object is in view when its squared distance to the camera does not exceed
// the squared clip range (0 without a camera). The comparison runs in x87
// extended precision on float32 coordinates; unordered operands are not in view.
func (n *nmiState) inView(obj uint32) bool {
	w := n.w
	if f := w.u32(obj + 0x308); f != 0 {
		k := nmiNext(w, nmiStubCntTf, nmiNTf)
		return w.u32(nmiStubRTf+4*k) == 0
	}
	cam := w.u32(0x659ab4) // 0x451280
	if cam != 0 && cam != nmiCam {
		panic("monster identification growth: camera object outside the fixture")
	}
	radius := float32(0)
	if cam != 0 && w.u32(cam+0x44) != 0 {
		nmiNext(w, nmiStubCntCl, 1<<31)
		far := math.Float64frombits(uint64(w.u32(nmiStubFar)) | uint64(w.u32(nmiStubFar+4))<<32)
		radius = float32(far) // 0x46e440 result stored through a float32
	}
	p := w.u32(0x659c0c) + 0x100
	var c, o [3]float32
	for a := uint32(0); a < 3; a++ {
		c[a] = math.Float32frombits(w.u32(p + 4*a))
		o[a] = math.Float32frombits(w.u32(obj + 0x118 + 4*a))
	}
	return nmiWithin(c, o, radius)
}

// nmiWithin: !(r*r < (dz*dz + dy*dy) + dx*dx) and ordered, with d = cam - obj
// evaluated in 64-bit-mantissa arithmetic.
func nmiWithin(cam, obj [3]float32, r float32) bool {
	vals := []float32{cam[0], cam[1], cam[2], obj[0], obj[1], obj[2], r}
	for _, v := range vals {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			var s [3]float64
			for a := 0; a < 3; a++ {
				s[a] = float64(cam[a]) - float64(obj[a])
			}
			dist := (s[2]*s[2] + s[1]*s[1]) + s[0]*s[0]
			rr := float64(r) * float64(r)
			return !(rr < dist) && !math.IsNaN(rr) && !math.IsNaN(dist)
		}
	}
	nf := func(x float64) *big.Float { return new(big.Float).SetPrec(64).SetFloat64(x) }
	sq := func(a int) *big.Float {
		x := new(big.Float).SetPrec(64).Sub(nf(float64(cam[a])), nf(float64(obj[a])))
		return x.Mul(x, x)
	}
	dist := new(big.Float).SetPrec(64).Add(sq(2), sq(1))
	dist.Add(dist, sq(0))
	rr := nf(float64(r))
	rr.Mul(rr, rr)
	return rr.Cmp(dist) >= 0
}

// counts is 0x4c7c00(obj, 1) != 0.
func (n *nmiState) counts(obj uint32) bool {
	w := n.w
	switch {
	case w.u8(obj+0x217) != 0:
		return false
	case w.u8(obj+0x215) != 0, w.u32(obj+0x1e4) == 0xffffffff, w.u32(obj+0x1dc)>>8&1 != 0, w.u8(obj+0x330) != 0:
		return true
	}
	m := w.findMonster(w.u32(obj + 0x1e4))
	if w.u8(nmiTemplate(w, w.u32(m+8))+0x248) != 0 {
		return w.u8(m+0x28d) != 0
	}
	return w.u8(m+0x2ab) != 0
}

// Tables generated from re/emu/cases/004e4ab0.json (object layout, setup, input and output places).

const (
	nmiO_chars    = 0x20000000
	nmiO_slots    = 0x2000d000
	nmiO_grpvec   = 0x2000e000
	nmiO_grpdata  = 0x2000e100
	nmiO_grp0     = 0x2000e200
	nmiO_grp1     = 0x2000e400
	nmiO_mvec0    = 0x2000e600
	nmiO_mdata0   = 0x2000e680
	nmiO_mvec1    = 0x2000e700
	nmiO_mdata1   = 0x2000e780
	nmiO_actvec   = 0x2000f000
	nmiO_actdata  = 0x2000f080
	nmiO_resvec   = 0x2000f100
	nmiO_resdata  = 0x2000f180
	nmiO_mon0     = 0x20010000
	nmiO_mon1     = 0x20010400
	nmiO_mon2     = 0x20010800
	nmiO_obj0     = 0x20011000
	nmiO_obj1     = 0x20011400
	nmiO_obj2     = 0x20011800
	nmiO_tmpl0    = 0x20012000
	nmiO_tmpl1    = 0x20012400
	nmiO_tmpl2    = 0x20012800
	nmiO_txt      = 0x20013000
	nmiO_fmt      = 0x20013800
	nmiO_s1       = 0x20013810
	nmiO_s2       = 0x20013820
	nmiO_s3       = 0x20013830
	nmiO_cam      = 0x20014000
	nmiO_pos      = 0x20014200
	nmiO_bat      = 0x20014800
	nmiO_stub     = 0x20015000
	nmiO_fonly    = 0x20016000
	nmiStubCnt553 = nmiO_stub + 0x0
	nmiStubA553_0 = nmiO_stub + 0x40
	nmiStubA553_1 = nmiO_stub + 0x80
	nmiStubA553_2 = nmiO_stub + 0xc0
	nmiStubA553_3 = nmiO_stub + 0x100
	nmiStubCnt58  = nmiO_stub + 0x140
	nmiStubA58_0  = nmiO_stub + 0x150
	nmiStubA58_1  = nmiO_stub + 0x160
	nmiStubA58_2  = nmiO_stub + 0x170
	nmiStubCntUp  = nmiO_stub + 0x180
	nmiStubAUp    = nmiO_stub + 0x190
	nmiStubRUp    = nmiO_stub + 0x1a0
	nmiStubCntTf  = nmiO_stub + 0x1c0
	nmiStubRTf    = nmiO_stub + 0x1d0
	nmiStubCntCl  = nmiO_stub + 0x200
	nmiStubFar    = nmiO_stub + 0x208
	nmiN553       = 16
	nmiN58        = 4
	nmiNUp        = 2
	nmiNTf        = 4
	nmiOwn        = 174        // inputs before the 0x553f10 template vector
	nmiFix        = 0x20014800 // battle object fixture
	nmiCam        = 0x20014000 // camera object fixture
)

var nmiSpec = mstSpec{
	set: []mstSet{
		{0x685174, 4, 0x20000000},
		{0x685178, 4, 0x2000d000},
		{0x683fb1, 4, 0x2000e000},
		{0x683fad, 4, 0x2000f000},
		{0x683fb9, 4, 0x2000f100},
		{0x68c09c, 4, 0x20013000},
		{0x6840c7, 4, 0x20012000},
		{0x6840cb, 4, 0x20012400},
		{0x6840cf, 4, 0x20012800},
		{0x659c0c, 4, 0x20014200},
		{0x689b07, 4, 0x2},
		{0x68f2fc, 4, 0x1},
		{0x689b0b, 4, 0x2},
		{0x692434, 4, 0x1},
		{0x689b0f, 4, 0x2},
		{0x69556c, 4, 0x1},
		{0x2000e000, 4, 0x2000e100},
		{0x2000e100, 4, 0x2000e200},
		{0x2000e104, 4, 0x2000e400},
		{0x2000e208, 4, 0x2000e600},
		{0x2000e600, 4, 0x2000e680},
		{0x2000e408, 4, 0x2000e700},
		{0x2000e700, 4, 0x2000e780},
		{0x2000f000, 4, 0x2000f080},
		{0x2000f008, 4, 0x2},
		{0x2000f080, 4, 0x20010000},
		{0x2000f084, 4, 0x20010400},
		{0x2000f100, 4, 0x2000f180},
		{0x2000f108, 4, 0x1},
		{0x2000f180, 4, 0x20010800},
		{0x20010000, 4, 0x0},
		{0x2001000c, 4, 0x20011000},
		{0x20010400, 4, 0x1},
		{0x2001040c, 4, 0x20011400},
		{0x20010800, 4, 0x2},
		{0x2001080c, 4, 0x20011800},
		{0x20013728, 4, 0x20013800},
		{0x2001372c, 4, 0x20013810},
		{0x20013730, 4, 0x20013820},
		{0x20013734, 4, 0x20013830},
		{0x20013800, 2, 0x25},
		{0x20013802, 2, 0x73},
		{0x20013810, 2, 0x25},
		{0x20013812, 2, 0x73},
		{0x20013820, 2, 0x25},
		{0x20013822, 2, 0x73},
		{0x20013830, 2, 0x25},
		{0x20013832, 2, 0x73},
	},
	in: []mstFld{
		{addr: 0x2000e008, size: 4, signed: false, idx: -1},
		{addr: 0x683f94, size: 1, signed: false, idx: -1},
		{addr: 0x68ec78, size: 4, signed: false, idx: -1},
		{addr: 0x68edcc, size: 4, signed: false, idx: -1},
		{addr: 0x200148f4, size: 4, signed: false, idx: -1},
		{addr: 0x659ab4, size: 4, signed: false, idx: -1},
		{addr: 0x20014044, size: 4, signed: false, idx: -1},
		{addr: 0x20015208, size: 4, signed: false, idx: -1},
		{addr: 0x2001520c, size: 4, signed: false, idx: -1},
		{addr: 0x20014300, size: 4, signed: false, idx: -1},
		{addr: 0x20014304, size: 4, signed: false, idx: -1},
		{addr: 0x20014308, size: 4, signed: false, idx: -1},
		{addr: 0x6875ef, size: 4, signed: false, idx: -1},
		{addr: 0x200151d0, size: 4, signed: false, idx: -1},
		{addr: 0x200151d4, size: 4, signed: false, idx: -1},
		{addr: 0x200151d8, size: 4, signed: false, idx: -1},
		{addr: 0x200151dc, size: 4, signed: false, idx: -1},
		{addr: 0x200151a0, size: 4, signed: false, idx: -1},
		{addr: 0x200151a4, size: 4, signed: false, idx: -1},
		{addr: 0x2000e204, size: 4, signed: false, idx: -1},
		{addr: 0x2000e218, size: 4, signed: false, idx: -1},
		{addr: 0x2000e228, size: 1, signed: false, idx: -1},
		{addr: 0x2000e229, size: 1, signed: false, idx: -1},
		{addr: 0x2000e22c, size: 1, signed: false, idx: -1},
		{addr: 0x2000e22d, size: 1, signed: false, idx: -1},
		{addr: 0x2000e608, size: 4, signed: false, idx: -1},
		{addr: 0x2000e680, size: 4, signed: false, idx: -1},
		{addr: 0x2000e684, size: 4, signed: false, idx: -1},
		{addr: 0x2000e404, size: 4, signed: false, idx: -1},
		{addr: 0x2000e418, size: 4, signed: false, idx: -1},
		{addr: 0x2000e428, size: 1, signed: false, idx: -1},
		{addr: 0x2000e429, size: 1, signed: false, idx: -1},
		{addr: 0x2000e42c, size: 1, signed: false, idx: -1},
		{addr: 0x2000e42d, size: 1, signed: false, idx: -1},
		{addr: 0x2000e708, size: 4, signed: false, idx: -1},
		{addr: 0x2000e780, size: 4, signed: false, idx: -1},
		{addr: 0x2000e784, size: 4, signed: false, idx: -1},
		{addr: 0x2000d000, size: 1, signed: false, idx: -1},
		{addr: 0x20000b11, size: 4, signed: false, idx: -1},
		{addr: 0x20000b01, size: 4, signed: false, idx: -1},
		{addr: 0x20000a31, size: 4, signed: false, idx: -1},
		{addr: 0x200000fd, size: 4, signed: false, idx: -1},
		{addr: 0x200004c1, size: 4, signed: false, idx: -1},
		{addr: 0x200004bb, size: 1, signed: false, idx: -1},
		{addr: 0x200012b6, size: 1, signed: false, idx: -1},
		{addr: 0x200012b7, size: 1, signed: false, idx: -1},
		{addr: 0x200012b8, size: 1, signed: false, idx: -1},
		{addr: 0x2000d106, size: 1, signed: false, idx: -1},
		{addr: 0x20002373, size: 4, signed: false, idx: -1},
		{addr: 0x20002363, size: 4, signed: false, idx: -1},
		{addr: 0x20002293, size: 4, signed: false, idx: -1},
		{addr: 0x2000195f, size: 4, signed: false, idx: -1},
		{addr: 0x20001d23, size: 4, signed: false, idx: -1},
		{addr: 0x20001d1d, size: 1, signed: false, idx: -1},
		{addr: 0x20002b18, size: 1, signed: false, idx: -1},
		{addr: 0x20002b19, size: 1, signed: false, idx: -1},
		{addr: 0x20002b1a, size: 1, signed: false, idx: -1},
		{addr: 0x2000d20c, size: 1, signed: false, idx: -1},
		{addr: 0x20003bd5, size: 4, signed: false, idx: -1},
		{addr: 0x20003bc5, size: 4, signed: false, idx: -1},
		{addr: 0x20003af5, size: 4, signed: false, idx: -1},
		{addr: 0x200031c1, size: 4, signed: false, idx: -1},
		{addr: 0x20003585, size: 4, signed: false, idx: -1},
		{addr: 0x2000357f, size: 1, signed: false, idx: -1},
		{addr: 0x2000437a, size: 1, signed: false, idx: -1},
		{addr: 0x2000437b, size: 1, signed: false, idx: -1},
		{addr: 0x2000437c, size: 1, signed: false, idx: -1},
		{addr: 0x2000d312, size: 1, signed: false, idx: -1},
		{addr: 0x20005437, size: 4, signed: false, idx: -1},
		{addr: 0x20005427, size: 4, signed: false, idx: -1},
		{addr: 0x20005357, size: 4, signed: false, idx: -1},
		{addr: 0x20004a23, size: 4, signed: false, idx: -1},
		{addr: 0x20004de7, size: 4, signed: false, idx: -1},
		{addr: 0x20004de1, size: 1, signed: false, idx: -1},
		{addr: 0x20005bdc, size: 1, signed: false, idx: -1},
		{addr: 0x20005bdd, size: 1, signed: false, idx: -1},
		{addr: 0x20005bde, size: 1, signed: false, idx: -1},
		{addr: 0x2000d418, size: 1, signed: false, idx: -1},
		{addr: 0x20006c99, size: 4, signed: false, idx: -1},
		{addr: 0x20006c89, size: 4, signed: false, idx: -1},
		{addr: 0x20006bb9, size: 4, signed: false, idx: -1},
		{addr: 0x20006285, size: 4, signed: false, idx: -1},
		{addr: 0x20006649, size: 4, signed: false, idx: -1},
		{addr: 0x20006643, size: 1, signed: false, idx: -1},
		{addr: 0x2000743e, size: 1, signed: false, idx: -1},
		{addr: 0x2000743f, size: 1, signed: false, idx: -1},
		{addr: 0x20007440, size: 1, signed: false, idx: -1},
		{addr: 0x2000d51e, size: 1, signed: false, idx: -1},
		{addr: 0x200084fb, size: 4, signed: false, idx: -1},
		{addr: 0x200084eb, size: 4, signed: false, idx: -1},
		{addr: 0x2000841b, size: 4, signed: false, idx: -1},
		{addr: 0x20007ae7, size: 4, signed: false, idx: -1},
		{addr: 0x20007eab, size: 4, signed: false, idx: -1},
		{addr: 0x20007ea5, size: 1, signed: false, idx: -1},
		{addr: 0x20008ca0, size: 1, signed: false, idx: -1},
		{addr: 0x20008ca1, size: 1, signed: false, idx: -1},
		{addr: 0x20008ca2, size: 1, signed: false, idx: -1},
		{addr: 0x2000d624, size: 1, signed: false, idx: -1},
		{addr: 0x20009d5d, size: 4, signed: false, idx: -1},
		{addr: 0x20009d4d, size: 4, signed: false, idx: -1},
		{addr: 0x20009c7d, size: 4, signed: false, idx: -1},
		{addr: 0x20009349, size: 4, signed: false, idx: -1},
		{addr: 0x2000970d, size: 4, signed: false, idx: -1},
		{addr: 0x20009707, size: 1, signed: false, idx: -1},
		{addr: 0x2000a502, size: 1, signed: false, idx: -1},
		{addr: 0x2000a503, size: 1, signed: false, idx: -1},
		{addr: 0x2000a504, size: 1, signed: false, idx: -1},
		{addr: 0x2000d72a, size: 1, signed: false, idx: -1},
		{addr: 0x2000b5bf, size: 4, signed: false, idx: -1},
		{addr: 0x2000b5af, size: 4, signed: false, idx: -1},
		{addr: 0x2000b4df, size: 4, signed: false, idx: -1},
		{addr: 0x2000abab, size: 4, signed: false, idx: -1},
		{addr: 0x2000af6f, size: 4, signed: false, idx: -1},
		{addr: 0x2000af69, size: 1, signed: false, idx: -1},
		{addr: 0x2000bd64, size: 1, signed: false, idx: -1},
		{addr: 0x2000bd65, size: 1, signed: false, idx: -1},
		{addr: 0x2000bd66, size: 1, signed: false, idx: -1},
		{addr: 0x2001224f, size: 1, signed: false, idx: -1},
		{addr: 0x20012248, size: 1, signed: false, idx: -1},
		{addr: 0x20012187, size: 2, signed: false, idx: -1},
		{addr: 0x20012000, size: 2, signed: false, idx: -1},
		{addr: 0x20012030, size: 2, signed: false, idx: -1},
		{addr: 0x20012060, size: 2, signed: false, idx: -1},
		{addr: 0x20012090, size: 2, signed: false, idx: -1},
		{addr: 0x2001264f, size: 1, signed: false, idx: -1},
		{addr: 0x20012648, size: 1, signed: false, idx: -1},
		{addr: 0x20012587, size: 2, signed: false, idx: -1},
		{addr: 0x20012400, size: 2, signed: false, idx: -1},
		{addr: 0x20012430, size: 2, signed: false, idx: -1},
		{addr: 0x20012460, size: 2, signed: false, idx: -1},
		{addr: 0x20012490, size: 2, signed: false, idx: -1},
		{addr: 0x20012a4f, size: 1, signed: false, idx: -1},
		{addr: 0x20012a48, size: 1, signed: false, idx: -1},
		{addr: 0x20012987, size: 2, signed: false, idx: -1},
		{addr: 0x20012800, size: 2, signed: false, idx: -1},
		{addr: 0x20012830, size: 2, signed: false, idx: -1},
		{addr: 0x20012860, size: 2, signed: false, idx: -1},
		{addr: 0x20012890, size: 2, signed: false, idx: -1},
		{addr: 0x20010008, size: 4, signed: false, idx: -1},
		{addr: 0x2001028d, size: 1, signed: false, idx: -1},
		{addr: 0x200102ab, size: 1, signed: false, idx: -1},
		{addr: 0x20010408, size: 4, signed: false, idx: -1},
		{addr: 0x2001068d, size: 1, signed: false, idx: -1},
		{addr: 0x200106ab, size: 1, signed: false, idx: -1},
		{addr: 0x20010808, size: 4, signed: false, idx: -1},
		{addr: 0x20010a8d, size: 1, signed: false, idx: -1},
		{addr: 0x20010aab, size: 1, signed: false, idx: -1},
		{addr: 0x20011308, size: 4, signed: false, idx: -1},
		{addr: 0x20011118, size: 4, signed: false, idx: -1},
		{addr: 0x2001111c, size: 4, signed: false, idx: -1},
		{addr: 0x20011120, size: 4, signed: false, idx: -1},
		{addr: 0x20011217, size: 1, signed: false, idx: -1},
		{addr: 0x20011215, size: 1, signed: false, idx: -1},
		{addr: 0x200111e4, size: 4, signed: false, idx: -1},
		{addr: 0x200111dc, size: 4, signed: false, idx: -1},
		{addr: 0x20011330, size: 1, signed: false, idx: -1},
		{addr: 0x20011708, size: 4, signed: false, idx: -1},
		{addr: 0x20011518, size: 4, signed: false, idx: -1},
		{addr: 0x2001151c, size: 4, signed: false, idx: -1},
		{addr: 0x20011520, size: 4, signed: false, idx: -1},
		{addr: 0x20011617, size: 1, signed: false, idx: -1},
		{addr: 0x20011615, size: 1, signed: false, idx: -1},
		{addr: 0x200115e4, size: 4, signed: false, idx: -1},
		{addr: 0x200115dc, size: 4, signed: false, idx: -1},
		{addr: 0x20011730, size: 1, signed: false, idx: -1},
		{addr: 0x20011b08, size: 4, signed: false, idx: -1},
		{addr: 0x20011918, size: 4, signed: false, idx: -1},
		{addr: 0x2001191c, size: 4, signed: false, idx: -1},
		{addr: 0x20011920, size: 4, signed: false, idx: -1},
		{addr: 0x20011a17, size: 1, signed: false, idx: -1},
		{addr: 0x20011a15, size: 1, signed: false, idx: -1},
		{addr: 0x200119e4, size: 4, signed: false, idx: -1},
		{addr: 0x200119dc, size: 4, signed: false, idx: -1},
		{addr: 0x20011b30, size: 1, signed: false, idx: -1},
		{addr: 0x20016000, size: 4, signed: false, idx: -1},
		{addr: 0x20016004, size: 4, signed: false, idx: -1},
		{addr: 0x20016008, size: 4, signed: true, idx: -1},
		{addr: 0x2001600c, size: 4, signed: false, idx: -1},
		{addr: 0x20016010, size: 1, signed: false, idx: -1},
		{addr: 0x20016014, size: 4, signed: false, idx: -1},
		{addr: 0x20016018, size: 4, signed: true, idx: -1},
		{addr: 0x2001601c, size: 4, signed: true, idx: -1},
		{addr: 0x20016020, size: 4, signed: false, idx: -1},
		{addr: 0x20016024, size: 1, signed: false, idx: -1},
		{addr: 0x20016028, size: 1, signed: false, idx: -1},
		{addr: 0x2001602c, size: 1, signed: false, idx: -1},
		{addr: 0x20016030, size: 1, signed: false, idx: -1},
		{addr: 0x20016034, size: 1, signed: false, idx: -1},
		{addr: 0x20016038, size: 1, signed: false, idx: -1},
		{addr: 0x2001603c, size: 4, signed: false, idx: -1},
		{addr: 0x20016040, size: 1, signed: false, idx: -1},
		{addr: 0x20016044, size: 1, signed: false, idx: -1},
		{addr: 0x20016048, size: 4, signed: true, idx: -1},
		{addr: 0x2001604c, size: 4, signed: true, idx: -1},
		{addr: 0x20016050, size: 4, signed: false, idx: -1},
		{addr: 0x20016054, size: 4, signed: false, idx: -1},
		{addr: 0x20016058, size: 4, signed: false, idx: -1},
		{addr: 0x2001605c, size: 4, signed: false, idx: -1},
		{addr: 0x20016060, size: 4, signed: false, idx: -1},
		{addr: 0x20016064, size: 4, signed: false, idx: -1},
		{addr: 0x20016068, size: 4, signed: false, idx: -1},
		{addr: 0x2001606c, size: 4, signed: false, idx: -1},
		{addr: 0x20016070, size: 4, signed: false, idx: -1},
		{addr: 0x20016074, size: 4, signed: false, idx: -1},
		{addr: 0x20016078, size: 4, signed: false, idx: -1},
		{addr: 0x2001607c, size: 4, signed: false, idx: -1},
		{addr: 0x20016080, size: 4, signed: false, idx: -1},
		{addr: 0x20016084, size: 4, signed: false, idx: -1},
		{addr: 0x20016088, size: 4, signed: false, idx: -1},
		{addr: 0x2001608c, size: 4, signed: false, idx: -1},
		{addr: 0x20016090, size: 4, signed: false, idx: -1},
		{addr: 0x20016094, size: 4, signed: false, idx: -1},
		{addr: 0x20016098, size: 4, signed: false, idx: -1},
		{addr: 0x2001609c, size: 4, signed: false, idx: -1},
		{addr: 0x200160a0, size: 4, signed: false, idx: -1},
		{addr: 0x200160a4, size: 4, signed: false, idx: -1},
		{addr: 0x200160a8, size: 4, signed: false, idx: -1},
		{addr: 0x200160ac, size: 4, signed: false, idx: -1},
		{addr: 0x200160b0, size: 4, signed: false, idx: -1},
		{addr: 0x200160b4, size: 4, signed: false, idx: -1},
		{addr: 0x200160b8, size: 4, signed: false, idx: -1},
		{addr: 0x200160bc, size: 4, signed: false, idx: -1},
		{addr: 0x200160c0, size: 4, signed: false, idx: -1},
		{addr: 0x200160c4, size: 4, signed: false, idx: -1},
		{addr: 0x200160c8, size: 4, signed: false, idx: -1},
		{addr: 0x200160cc, size: 4, signed: false, idx: -1},
		{addr: 0x200160d0, size: 4, signed: false, idx: -1},
		{addr: 0x200160d4, size: 4, signed: false, idx: -1},
		{addr: 0x200160d8, size: 4, signed: false, idx: -1},
		{addr: 0x200160dc, size: 4, signed: false, idx: -1},
		{addr: 0x200160e0, size: 4, signed: false, idx: -1},
		{addr: 0x200160e4, size: 4, signed: false, idx: -1},
		{addr: 0x200160e8, size: 4, signed: false, idx: -1},
		{addr: 0x200160ec, size: 4, signed: false, idx: -1},
		{addr: 0x200160f0, size: 4, signed: false, idx: -1},
		{addr: 0x200160f4, size: 4, signed: false, idx: -1},
		{addr: 0x200160f8, size: 4, signed: false, idx: -1},
		{addr: 0x200160fc, size: 4, signed: false, idx: -1},
		{addr: 0x20016100, size: 4, signed: false, idx: -1},
		{addr: 0x20016104, size: 4, signed: false, idx: -1},
		{addr: 0x20016108, size: 4, signed: false, idx: -1},
		{addr: 0x2001610c, size: 4, signed: false, idx: -1},
		{addr: 0x20016110, size: 4, signed: false, idx: -1},
		{addr: 0x20016114, size: 4, signed: false, idx: -1},
		{addr: 0x20016118, size: 4, signed: false, idx: -1},
		{addr: 0x2001611c, size: 4, signed: false, idx: -1},
		{addr: 0x20016120, size: 4, signed: false, idx: -1},
		{addr: 0x20016124, size: 4, signed: false, idx: -1},
		{addr: 0x20016128, size: 4, signed: false, idx: -1},
		{addr: 0x2001612c, size: 4, signed: false, idx: -1},
		{addr: 0x20016130, size: 4, signed: false, idx: -1},
		{addr: 0x20016134, size: 4, signed: false, idx: -1},
		{addr: 0x20016138, size: 4, signed: false, idx: -1},
		{addr: 0x2001613c, size: 4, signed: false, idx: -1},
		{addr: 0x20016140, size: 4, signed: false, idx: -1},
		{addr: 0x20016144, size: 4, signed: false, idx: -1},
		{addr: 0x20016148, size: 4, signed: false, idx: -1},
		{addr: 0x2001614c, size: 4, signed: false, idx: -1},
		{addr: 0x20016150, size: 4, signed: false, idx: -1},
		{addr: 0x20016154, size: 4, signed: false, idx: -1},
		{addr: 0x20016158, size: 4, signed: false, idx: -1},
		{addr: 0x2001615c, size: 4, signed: false, idx: -1},
		{addr: 0x20016160, size: 4, signed: false, idx: -1},
		{addr: 0x20016164, size: 4, signed: false, idx: -1},
		{addr: 0x20016168, size: 4, signed: false, idx: -1},
		{addr: 0x2001616c, size: 4, signed: false, idx: -1},
		{addr: 0x20016170, size: 4, signed: false, idx: -1},
		{addr: 0x20016174, size: 4, signed: false, idx: -1},
		{addr: 0x20016178, size: 4, signed: false, idx: -1},
		{addr: 0x2001617c, size: 4, signed: false, idx: -1},
		{addr: 0x20016180, size: 4, signed: false, idx: -1},
		{addr: 0x20016184, size: 4, signed: false, idx: -1},
		{addr: 0x20016188, size: 4, signed: false, idx: -1},
		{addr: 0x2001618c, size: 4, signed: false, idx: -1},
		{addr: 0x20016190, size: 4, signed: false, idx: -1},
		{addr: 0x20016194, size: 4, signed: false, idx: -1},
		{addr: 0x20016198, size: 4, signed: false, idx: -1},
		{addr: 0x2001619c, size: 4, signed: false, idx: -1},
		{addr: 0x200161a0, size: 4, signed: false, idx: -1},
		{addr: 0x200161a4, size: 4, signed: false, idx: -1},
		{addr: 0x200161a8, size: 4, signed: false, idx: -1},
		{addr: 0x200161ac, size: 4, signed: false, idx: -1},
		{addr: 0x200161b0, size: 4, signed: false, idx: -1},
		{addr: 0x200161b4, size: 4, signed: false, idx: -1},
		{addr: 0x200161b8, size: 4, signed: false, idx: -1},
		{addr: 0x200161bc, size: 4, signed: false, idx: -1},
		{addr: 0x200161c0, size: 4, signed: false, idx: -1},
		{addr: 0x200161c4, size: 4, signed: false, idx: -1},
		{addr: 0x200161c8, size: 4, signed: false, idx: -1},
		{addr: 0x200161cc, size: 4, signed: false, idx: -1},
		{addr: 0x200161d0, size: 4, signed: false, idx: -1},
		{addr: 0x200161d4, size: 4, signed: false, idx: -1},
		{addr: 0x200161d8, size: 4, signed: false, idx: -1},
		{addr: 0x200161dc, size: 4, signed: false, idx: -1},
		{addr: 0x200161e0, size: 4, signed: false, idx: -1},
		{addr: 0x200161e4, size: 4, signed: false, idx: -1},
		{addr: 0x200161e8, size: 4, signed: false, idx: -1},
		{addr: 0x200161ec, size: 4, signed: false, idx: -1},
		{addr: 0x200161f0, size: 4, signed: false, idx: -1},
		{addr: 0x200161f4, size: 4, signed: false, idx: -1},
		{addr: 0x200161f8, size: 4, signed: false, idx: -1},
		{addr: 0x200161fc, size: 4, signed: false, idx: -1},
		{addr: 0x20016200, size: 4, signed: false, idx: -1},
		{addr: 0x20016204, size: 4, signed: false, idx: -1},
		{addr: 0x20016208, size: 4, signed: false, idx: -1},
		{addr: 0x2001620c, size: 4, signed: false, idx: -1},
		{addr: 0x20016210, size: 4, signed: false, idx: -1},
		{addr: 0x20016214, size: 4, signed: false, idx: -1},
		{addr: 0x20016218, size: 4, signed: false, idx: -1},
		{addr: 0x2001621c, size: 4, signed: false, idx: -1},
		{addr: 0x20016220, size: 4, signed: false, idx: -1},
		{addr: 0x20016224, size: 4, signed: false, idx: -1},
		{addr: 0x20016228, size: 4, signed: false, idx: -1},
		{addr: 0x2001622c, size: 4, signed: false, idx: -1},
		{addr: 0x20016230, size: 4, signed: false, idx: -1},
		{addr: 0x20016234, size: 4, signed: false, idx: -1},
		{addr: 0x20016238, size: 4, signed: false, idx: -1},
		{addr: 0x2001623c, size: 4, signed: false, idx: -1},
		{addr: 0x20016240, size: 1, signed: false, idx: -1},
		{addr: 0x20016244, size: 1, signed: false, idx: -1},
		{addr: 0x20016248, size: 1, signed: false, idx: -1},
		{addr: 0x2001624c, size: 1, signed: false, idx: -1},
		{addr: 0x20016250, size: 1, signed: false, idx: -1},
		{addr: 0x20016254, size: 1, signed: false, idx: -1},
		{addr: 0x20016258, size: 1, signed: false, idx: -1},
		{addr: 0x2001625c, size: 1, signed: false, idx: -1},
		{addr: 0x20016260, size: 1, signed: false, idx: -1},
		{addr: 0x20016264, size: 1, signed: false, idx: -1},
		{addr: 0x20016268, size: 1, signed: false, idx: -1},
		{addr: 0x2001626c, size: 1, signed: false, idx: -1},
		{addr: 0x20016270, size: 1, signed: false, idx: -1},
		{addr: 0x20016274, size: 1, signed: false, idx: -1},
		{addr: 0x20016278, size: 1, signed: false, idx: -1},
		{addr: 0x2001627c, size: 1, signed: false, idx: -1},
		{addr: 0x20016280, size: 1, signed: false, idx: -1},
		{addr: 0x20016284, size: 1, signed: false, idx: -1},
		{addr: 0x20016288, size: 1, signed: false, idx: -1},
		{addr: 0x2001628c, size: 1, signed: false, idx: -1},
		{addr: 0x20016290, size: 1, signed: false, idx: -1},
		{addr: 0x20016294, size: 1, signed: false, idx: -1},
		{addr: 0x20016298, size: 1, signed: false, idx: -1},
		{addr: 0x2001629c, size: 1, signed: false, idx: -1},
		{addr: 0x200162a0, size: 1, signed: false, idx: -1},
		{addr: 0x200162a4, size: 1, signed: false, idx: -1},
		{addr: 0x200162a8, size: 1, signed: false, idx: -1},
		{addr: 0x200162ac, size: 1, signed: false, idx: -1},
		{addr: 0x200162b0, size: 1, signed: false, idx: -1},
		{addr: 0x200162b4, size: 1, signed: false, idx: -1},
		{addr: 0x200162b8, size: 1, signed: false, idx: -1},
		{addr: 0x200162bc, size: 1, signed: false, idx: -1},
		{addr: 0x200162c0, size: 1, signed: false, idx: -1},
		{addr: 0x200162c4, size: 1, signed: false, idx: -1},
		{addr: 0x200162c8, size: 1, signed: false, idx: -1},
		{addr: 0x200162cc, size: 1, signed: false, idx: -1},
		{addr: 0x200162d0, size: 1, signed: false, idx: -1},
		{addr: 0x200162d4, size: 1, signed: false, idx: -1},
		{addr: 0x200162d8, size: 1, signed: false, idx: -1},
		{addr: 0x200162dc, size: 1, signed: false, idx: -1},
		{addr: 0x200162e0, size: 1, signed: false, idx: -1},
		{addr: 0x200162e4, size: 1, signed: false, idx: -1},
		{addr: 0x200162e8, size: 1, signed: false, idx: -1},
		{addr: 0x200162ec, size: 1, signed: false, idx: -1},
		{addr: 0x200162f0, size: 1, signed: false, idx: -1},
		{addr: 0x200162f4, size: 1, signed: false, idx: -1},
		{addr: 0x200162f8, size: 1, signed: false, idx: -1},
		{addr: 0x200162fc, size: 1, signed: false, idx: -1},
		{addr: 0x20016300, size: 1, signed: false, idx: -1},
		{addr: 0x20016304, size: 1, signed: false, idx: -1},
		{addr: 0x20016308, size: 1, signed: false, idx: -1},
		{addr: 0x2001630c, size: 1, signed: false, idx: -1},
		{addr: 0x20016310, size: 1, signed: false, idx: -1},
		{addr: 0x20016314, size: 1, signed: false, idx: -1},
		{addr: 0x20016318, size: 1, signed: false, idx: -1},
		{addr: 0x2001631c, size: 1, signed: false, idx: -1},
		{addr: 0x20016320, size: 1, signed: false, idx: -1},
		{addr: 0x20016324, size: 1, signed: false, idx: -1},
		{addr: 0x20016328, size: 1, signed: false, idx: -1},
		{addr: 0x2001632c, size: 1, signed: false, idx: -1},
		{addr: 0x20016330, size: 1, signed: false, idx: -1},
		{addr: 0x20016334, size: 1, signed: false, idx: -1},
		{addr: 0x20016338, size: 1, signed: false, idx: -1},
		{addr: 0x2001633c, size: 1, signed: false, idx: -1},
		{addr: 0x20016340, size: 1, signed: false, idx: -1},
		{addr: 0x20016344, size: 1, signed: false, idx: -1},
		{addr: 0x20016348, size: 1, signed: false, idx: -1},
		{addr: 0x2001634c, size: 1, signed: false, idx: -1},
		{addr: 0x20016350, size: 1, signed: false, idx: -1},
		{addr: 0x20016354, size: 1, signed: false, idx: -1},
		{addr: 0x20016358, size: 1, signed: false, idx: -1},
		{addr: 0x2001635c, size: 1, signed: false, idx: -1},
		{addr: 0x20016360, size: 1, signed: false, idx: -1},
		{addr: 0x20016364, size: 1, signed: false, idx: -1},
		{addr: 0x20016368, size: 1, signed: false, idx: -1},
		{addr: 0x2001636c, size: 1, signed: false, idx: -1},
		{addr: 0x20016370, size: 1, signed: false, idx: -1},
		{addr: 0x20016374, size: 1, signed: false, idx: -1},
		{addr: 0x20016378, size: 1, signed: false, idx: -1},
		{addr: 0x2001637c, size: 1, signed: false, idx: -1},
		{addr: 0x20016380, size: 1, signed: false, idx: -1},
		{addr: 0x20016384, size: 1, signed: false, idx: -1},
		{addr: 0x20016388, size: 1, signed: false, idx: -1},
		{addr: 0x2001638c, size: 1, signed: false, idx: -1},
		{addr: 0x20016390, size: 1, signed: false, idx: -1},
		{addr: 0x20016394, size: 1, signed: false, idx: -1},
		{addr: 0x20016398, size: 1, signed: false, idx: -1},
		{addr: 0x2001639c, size: 1, signed: false, idx: -1},
		{addr: 0x200163a0, size: 1, signed: false, idx: -1},
		{addr: 0x200163a4, size: 1, signed: false, idx: -1},
		{addr: 0x200163a8, size: 1, signed: false, idx: -1},
		{addr: 0x200163ac, size: 1, signed: false, idx: -1},
		{addr: 0x200163b0, size: 1, signed: false, idx: -1},
		{addr: 0x200163b4, size: 1, signed: false, idx: -1},
		{addr: 0x200163b8, size: 1, signed: false, idx: -1},
		{addr: 0x200163bc, size: 1, signed: false, idx: -1},
		{addr: 0x200163c0, size: 1, signed: false, idx: -1},
		{addr: 0x200163c4, size: 1, signed: false, idx: -1},
		{addr: 0x200163c8, size: 1, signed: false, idx: -1},
		{addr: 0x200163cc, size: 1, signed: false, idx: -1},
		{addr: 0x200163d0, size: 1, signed: false, idx: -1},
		{addr: 0x200163d4, size: 1, signed: false, idx: -1},
		{addr: 0x200163d8, size: 1, signed: false, idx: -1},
		{addr: 0x200163dc, size: 1, signed: false, idx: -1},
		{addr: 0x200163e0, size: 1, signed: false, idx: -1},
		{addr: 0x200163e4, size: 1, signed: false, idx: -1},
		{addr: 0x200163e8, size: 1, signed: false, idx: -1},
		{addr: 0x200163ec, size: 1, signed: false, idx: -1},
		{addr: 0x200163f0, size: 1, signed: false, idx: -1},
		{addr: 0x200163f4, size: 1, signed: false, idx: -1},
		{addr: 0x200163f8, size: 1, signed: false, idx: -1},
		{addr: 0x200163fc, size: 1, signed: false, idx: -1},
		{addr: 0x20016400, size: 1, signed: false, idx: -1},
		{addr: 0x20016404, size: 1, signed: false, idx: -1},
		{addr: 0x20016408, size: 1, signed: false, idx: -1},
		{addr: 0x2001640c, size: 1, signed: false, idx: -1},
		{addr: 0x20016410, size: 1, signed: false, idx: -1},
		{addr: 0x20016414, size: 1, signed: false, idx: -1},
		{addr: 0x20016418, size: 1, signed: false, idx: -1},
		{addr: 0x2001641c, size: 1, signed: false, idx: -1},
		{addr: 0x20016420, size: 1, signed: false, idx: -1},
		{addr: 0x20016424, size: 1, signed: false, idx: -1},
		{addr: 0x20016428, size: 1, signed: false, idx: -1},
		{addr: 0x2001642c, size: 4, signed: false, idx: -1},
		{addr: 0x20016430, size: 4, signed: false, idx: -1},
		{addr: 0x20016434, size: 4, signed: false, idx: -1},
		{addr: 0x20016438, size: 4, signed: false, idx: -1},
		{addr: 0x2001643c, size: 4, signed: false, idx: -1},
		{addr: 0x20016440, size: 4, signed: false, idx: -1},
		{addr: 0x20016444, size: 4, signed: false, idx: -1},
		{addr: 0x20016448, size: 4, signed: false, idx: -1},
		{addr: 0x2001644c, size: 4, signed: false, idx: -1},
		{addr: 0x20016450, size: 4, signed: false, idx: -1},
		{addr: 0x20016454, size: 4, signed: false, idx: -1},
		{addr: 0x20016458, size: 4, signed: false, idx: -1},
		{addr: 0x2001645c, size: 4, signed: false, idx: -1},
		{addr: 0x20016460, size: 4, signed: false, idx: -1},
		{addr: 0x20016464, size: 4, signed: false, idx: -1},
		{addr: 0x20016468, size: 4, signed: false, idx: -1},
		{addr: 0x2001646c, size: 4, signed: false, idx: -1},
		{addr: 0x20016470, size: 4, signed: false, idx: -1},
		{addr: 0x20016474, size: 4, signed: false, idx: -1},
		{addr: 0x20016478, size: 4, signed: false, idx: -1},
		{addr: 0x2001647c, size: 4, signed: false, idx: -1},
		{addr: 0x20016480, size: 4, signed: false, idx: -1},
		{addr: 0x20016484, size: 4, signed: false, idx: -1},
		{addr: 0x20016488, size: 4, signed: false, idx: -1},
		{addr: 0x2001648c, size: 4, signed: false, idx: -1},
		{addr: 0x20016490, size: 4, signed: false, idx: -1},
		{addr: 0x20016494, size: 4, signed: false, idx: -1},
		{addr: 0x20016498, size: 4, signed: false, idx: -1},
		{addr: 0x2001649c, size: 4, signed: false, idx: -1},
		{addr: 0x200164a0, size: 4, signed: false, idx: -1},
		{addr: 0x200164a4, size: 4, signed: false, idx: -1},
		{addr: 0x200164a8, size: 4, signed: false, idx: -1},
		{addr: 0x200164ac, size: 4, signed: false, idx: -1},
		{addr: 0x200164b0, size: 4, signed: false, idx: -1},
		{addr: 0x200164b4, size: 4, signed: false, idx: -1},
		{addr: 0x200164b8, size: 4, signed: false, idx: -1},
		{addr: 0x200164bc, size: 4, signed: false, idx: -1},
		{addr: 0x200164c0, size: 4, signed: false, idx: -1},
		{addr: 0x200164c4, size: 4, signed: false, idx: -1},
		{addr: 0x200164c8, size: 4, signed: false, idx: -1},
		{addr: 0x200164cc, size: 4, signed: false, idx: -1},
		{addr: 0x200164d0, size: 1, signed: true, idx: -1},
		{addr: 0x200164d4, size: 1, signed: true, idx: -1},
		{addr: 0x200164d8, size: 1, signed: true, idx: -1},
		{addr: 0x200164dc, size: 1, signed: true, idx: -1},
		{addr: 0x200164e0, size: 1, signed: true, idx: -1},
		{addr: 0x200164e4, size: 1, signed: true, idx: -1},
		{addr: 0x200164e8, size: 1, signed: true, idx: -1},
		{addr: 0x200164ec, size: 1, signed: true, idx: -1},
		{addr: 0x200164f0, size: 1, signed: true, idx: -1},
		{addr: 0x200164f4, size: 1, signed: true, idx: -1},
		{addr: 0x200164f8, size: 1, signed: true, idx: -1},
		{addr: 0x200164fc, size: 1, signed: true, idx: -1},
		{addr: 0x20016500, size: 1, signed: true, idx: -1},
		{addr: 0x20016504, size: 1, signed: true, idx: -1},
		{addr: 0x20016508, size: 1, signed: true, idx: -1},
		{addr: 0x2001650c, size: 1, signed: true, idx: -1},
		{addr: 0x20016510, size: 1, signed: true, idx: -1},
		{addr: 0x20016514, size: 1, signed: true, idx: -1},
		{addr: 0x20016518, size: 1, signed: true, idx: -1},
		{addr: 0x2001651c, size: 1, signed: true, idx: -1},
		{addr: 0x20016520, size: 1, signed: true, idx: -1},
		{addr: 0x20016524, size: 1, signed: true, idx: -1},
		{addr: 0x20016528, size: 1, signed: true, idx: -1},
		{addr: 0x2001652c, size: 1, signed: true, idx: -1},
		{addr: 0x20016530, size: 1, signed: true, idx: -1},
		{addr: 0x20016534, size: 1, signed: true, idx: -1},
		{addr: 0x20016538, size: 1, signed: true, idx: -1},
		{addr: 0x2001653c, size: 1, signed: true, idx: -1},
		{addr: 0x20016540, size: 1, signed: true, idx: -1},
		{addr: 0x20016544, size: 1, signed: true, idx: -1},
		{addr: 0x20016548, size: 1, signed: true, idx: -1},
		{addr: 0x2001654c, size: 1, signed: true, idx: -1},
		{addr: 0x20016550, size: 1, signed: true, idx: -1},
		{addr: 0x20016554, size: 1, signed: true, idx: -1},
		{addr: 0x20016558, size: 1, signed: true, idx: -1},
		{addr: 0x2001655c, size: 1, signed: true, idx: -1},
		{addr: 0x20016560, size: 1, signed: true, idx: -1},
		{addr: 0x20016564, size: 1, signed: true, idx: -1},
		{addr: 0x20016568, size: 1, signed: true, idx: -1},
		{addr: 0x2001656c, size: 1, signed: true, idx: -1},
		{addr: 0x20016570, size: 1, signed: true, idx: -1},
		{addr: 0x20016574, size: 4, signed: false, idx: -1},
		{addr: 0x20016578, size: 4, signed: false, idx: -1},
		{addr: 0x2001657c, size: 4, signed: false, idx: -1},
		{addr: 0x20016580, size: 4, signed: false, idx: -1},
		{addr: 0x20016584, size: 4, signed: false, idx: -1},
		{addr: 0x20016588, size: 4, signed: false, idx: -1},
		{addr: 0x2001658c, size: 4, signed: false, idx: -1},
		{addr: 0x20016590, size: 4, signed: false, idx: -1},
		{addr: 0x20016594, size: 4, signed: false, idx: -1},
		{addr: 0x20016598, size: 4, signed: false, idx: -1},
		{addr: 0x2001659c, size: 4, signed: false, idx: -1},
		{addr: 0x200165a0, size: 4, signed: false, idx: -1},
		{addr: 0x200165a4, size: 4, signed: false, idx: -1},
		{addr: 0x200165a8, size: 4, signed: false, idx: -1},
		{addr: 0x200165ac, size: 4, signed: false, idx: -1},
		{addr: 0x200165b0, size: 4, signed: false, idx: -1},
		{addr: 0x200165b4, size: 4, signed: false, idx: -1},
		{addr: 0x200165b8, size: 4, signed: false, idx: -1},
		{addr: 0x200165bc, size: 4, signed: false, idx: -1},
		{addr: 0x200165c0, size: 4, signed: false, idx: -1},
		{addr: 0x200165c4, size: 4, signed: false, idx: -1},
		{addr: 0x200165c8, size: 4, signed: false, idx: -1},
		{addr: 0x200165cc, size: 4, signed: false, idx: -1},
		{addr: 0x200165d0, size: 4, signed: false, idx: -1},
		{addr: 0x200165d4, size: 4, signed: false, idx: -1},
		{addr: 0x200165d8, size: 4, signed: false, idx: -1},
		{addr: 0x200165dc, size: 4, signed: false, idx: -1},
		{addr: 0x200165e0, size: 4, signed: false, idx: -1},
		{addr: 0x200165e4, size: 4, signed: false, idx: -1},
		{addr: 0x200165e8, size: 4, signed: false, idx: -1},
		{addr: 0x200165ec, size: 4, signed: false, idx: -1},
		{addr: 0x200165f0, size: 4, signed: false, idx: -1},
		{addr: 0x200165f4, size: 4, signed: false, idx: -1},
		{addr: 0x200165f8, size: 4, signed: false, idx: -1},
		{addr: 0x200165fc, size: 4, signed: false, idx: -1},
		{addr: 0x20016600, size: 4, signed: false, idx: -1},
		{addr: 0x20016604, size: 4, signed: false, idx: -1},
		{addr: 0x20016608, size: 4, signed: false, idx: -1},
		{addr: 0x2001660c, size: 4, signed: false, idx: -1},
		{addr: 0x20016610, size: 4, signed: false, idx: -1},
		{addr: 0x20016614, size: 4, signed: false, idx: -1},
		{addr: 0x20016618, size: 4, signed: false, idx: -1},
		{addr: 0x2001661c, size: 4, signed: false, idx: -1},
		{addr: 0x20016620, size: 4, signed: false, idx: -1},
		{addr: 0x20016624, size: 4, signed: false, idx: -1},
		{addr: 0x20016628, size: 4, signed: false, idx: -1},
		{addr: 0x2001662c, size: 4, signed: false, idx: -1},
		{addr: 0x20016630, size: 4, signed: false, idx: -1},
		{addr: 0x20016634, size: 4, signed: false, idx: -1},
		{addr: 0x20016638, size: 4, signed: false, idx: -1},
		{addr: 0x2001663c, size: 4, signed: false, idx: -1},
		{addr: 0x20016640, size: 4, signed: false, idx: -1},
		{addr: 0x20016644, size: 4, signed: false, idx: -1},
		{addr: 0x20016648, size: 4, signed: false, idx: -1},
		{addr: 0x2001664c, size: 4, signed: false, idx: -1},
		{addr: 0x20016650, size: 4, signed: false, idx: -1},
		{addr: 0x20016654, size: 4, signed: false, idx: -1},
		{addr: 0x20016658, size: 4, signed: false, idx: -1},
		{addr: 0x2001665c, size: 4, signed: false, idx: -1},
		{addr: 0x20016660, size: 4, signed: false, idx: -1},
		{addr: 0x20016664, size: 4, signed: false, idx: -1},
		{addr: 0x20016668, size: 4, signed: false, idx: -1},
		{addr: 0x2001666c, size: 4, signed: false, idx: -1},
		{addr: 0x20016670, size: 4, signed: true, idx: -1},
		{addr: 0x20016674, size: 4, signed: true, idx: -1},
		{addr: 0x20016678, size: 4, signed: true, idx: -1},
		{addr: 0x2001667c, size: 4, signed: true, idx: -1},
		{addr: 0x20016680, size: 4, signed: true, idx: -1},
		{addr: 0x20016684, size: 4, signed: true, idx: -1},
		{addr: 0x20016688, size: 4, signed: true, idx: -1},
		{addr: 0x2001668c, size: 4, signed: true, idx: -1},
		{addr: 0x20016690, size: 4, signed: true, idx: -1},
		{addr: 0x20016694, size: 4, signed: true, idx: -1},
		{addr: 0x20016698, size: 4, signed: true, idx: -1},
		{addr: 0x2001669c, size: 4, signed: true, idx: -1},
		{addr: 0x200166a0, size: 4, signed: true, idx: -1},
		{addr: 0x200166a4, size: 4, signed: true, idx: -1},
		{addr: 0x200166a8, size: 4, signed: true, idx: -1},
		{addr: 0x200166ac, size: 4, signed: true, idx: -1},
		{addr: 0x200166b0, size: 4, signed: true, idx: -1},
		{addr: 0x200166b4, size: 4, signed: true, idx: -1},
		{addr: 0x200166b8, size: 4, signed: true, idx: -1},
		{addr: 0x200166bc, size: 4, signed: true, idx: -1},
		{addr: 0x200166c0, size: 4, signed: true, idx: -1},
		{addr: 0x200166c4, size: 4, signed: true, idx: -1},
		{addr: 0x200166c8, size: 4, signed: true, idx: -1},
		{addr: 0x200166cc, size: 4, signed: true, idx: -1},
		{addr: 0x200166d0, size: 4, signed: true, idx: -1},
		{addr: 0x200166d4, size: 4, signed: true, idx: -1},
		{addr: 0x200166d8, size: 4, signed: true, idx: -1},
		{addr: 0x200166dc, size: 4, signed: true, idx: -1},
		{addr: 0x200166e0, size: 4, signed: true, idx: -1},
		{addr: 0x200166e4, size: 4, signed: true, idx: -1},
		{addr: 0x200166e8, size: 4, signed: true, idx: -1},
		{addr: 0x200166ec, size: 4, signed: true, idx: -1},
		{addr: 0x200166f0, size: 4, signed: true, idx: -1},
		{addr: 0x200166f4, size: 4, signed: true, idx: -1},
		{addr: 0x200166f8, size: 4, signed: true, idx: -1},
		{addr: 0x200166fc, size: 4, signed: true, idx: -1},
		{addr: 0x20016700, size: 4, signed: true, idx: -1},
		{addr: 0x20016704, size: 4, signed: true, idx: -1},
		{addr: 0x20016708, size: 4, signed: true, idx: -1},
		{addr: 0x2001670c, size: 4, signed: true, idx: -1},
		{addr: 0x20016710, size: 4, signed: true, idx: -1},
		{addr: 0x20016714, size: 4, signed: true, idx: -1},
		{addr: 0x20016718, size: 4, signed: true, idx: -1},
		{addr: 0x2001671c, size: 4, signed: true, idx: -1},
		{addr: 0x20016720, size: 4, signed: true, idx: -1},
		{addr: 0x20016724, size: 4, signed: true, idx: -1},
		{addr: 0x20016728, size: 4, signed: true, idx: -1},
		{addr: 0x2001672c, size: 4, signed: true, idx: -1},
		{addr: 0x20016730, size: 4, signed: true, idx: -1},
		{addr: 0x20016734, size: 4, signed: true, idx: -1},
		{addr: 0x20016738, size: 4, signed: true, idx: -1},
		{addr: 0x2001673c, size: 4, signed: true, idx: -1},
		{addr: 0x20016740, size: 4, signed: true, idx: -1},
		{addr: 0x20016744, size: 4, signed: true, idx: -1},
		{addr: 0x20016748, size: 4, signed: true, idx: -1},
		{addr: 0x2001674c, size: 4, signed: true, idx: -1},
		{addr: 0x20016750, size: 4, signed: true, idx: -1},
		{addr: 0x20016754, size: 4, signed: true, idx: -1},
		{addr: 0x20016758, size: 4, signed: true, idx: -1},
		{addr: 0x2001675c, size: 4, signed: true, idx: -1},
		{addr: 0x20016760, size: 4, signed: true, idx: -1},
		{addr: 0x20016764, size: 4, signed: true, idx: -1},
		{addr: 0x20016768, size: 4, signed: true, idx: -1},
		{addr: 0x2001676c, size: 4, signed: true, idx: -1},
		{addr: 0x20016770, size: 4, signed: true, idx: -1},
		{addr: 0x20016774, size: 4, signed: true, idx: -1},
		{addr: 0x20016778, size: 4, signed: true, idx: -1},
		{addr: 0x2001677c, size: 4, signed: true, idx: -1},
		{addr: 0x20016780, size: 4, signed: true, idx: -1},
		{addr: 0x20016784, size: 4, signed: true, idx: -1},
		{addr: 0x20016788, size: 4, signed: true, idx: -1},
		{addr: 0x2001678c, size: 4, signed: true, idx: -1},
		{addr: 0x20016790, size: 4, signed: true, idx: -1},
		{addr: 0x20016794, size: 4, signed: true, idx: -1},
		{addr: 0x20016798, size: 4, signed: true, idx: -1},
		{addr: 0x2001679c, size: 4, signed: true, idx: -1},
		{addr: 0x200167a0, size: 4, signed: true, idx: -1},
		{addr: 0x200167a4, size: 4, signed: true, idx: -1},
		{addr: 0x200167a8, size: 4, signed: true, idx: -1},
		{addr: 0x200167ac, size: 4, signed: true, idx: -1},
		{addr: 0x200167b0, size: 4, signed: true, idx: -1},
		{addr: 0x200167b4, size: 4, signed: true, idx: -1},
		{addr: 0x200167b8, size: 4, signed: true, idx: -1},
		{addr: 0x200167bc, size: 4, signed: true, idx: -1},
		{addr: 0x200167c0, size: 4, signed: true, idx: -1},
		{addr: 0x200167c4, size: 4, signed: true, idx: -1},
		{addr: 0x200167c8, size: 4, signed: true, idx: -1},
		{addr: 0x200167cc, size: 4, signed: true, idx: -1},
		{addr: 0x200167d0, size: 4, signed: true, idx: -1},
		{addr: 0x200167d4, size: 4, signed: true, idx: -1},
		{addr: 0x200167d8, size: 4, signed: true, idx: -1},
		{addr: 0x200167dc, size: 4, signed: true, idx: -1},
		{addr: 0x200167e0, size: 4, signed: true, idx: -1},
		{addr: 0x200167e4, size: 4, signed: true, idx: -1},
		{addr: 0x200167e8, size: 4, signed: true, idx: -1},
		{addr: 0x200167ec, size: 4, signed: true, idx: -1},
		{addr: 0x200167f0, size: 4, signed: true, idx: -1},
		{addr: 0x200167f4, size: 4, signed: true, idx: -1},
		{addr: 0x200167f8, size: 4, signed: true, idx: -1},
		{addr: 0x200167fc, size: 4, signed: true, idx: -1},
		{addr: 0x20016800, size: 4, signed: true, idx: -1},
		{addr: 0x20016804, size: 4, signed: true, idx: -1},
		{addr: 0x20016808, size: 4, signed: true, idx: -1},
		{addr: 0x2001680c, size: 4, signed: true, idx: -1},
		{addr: 0x20016810, size: 4, signed: true, idx: -1},
		{addr: 0x20016814, size: 4, signed: true, idx: -1},
		{addr: 0x20016818, size: 4, signed: true, idx: -1},
		{addr: 0x2001681c, size: 4, signed: true, idx: -1},
		{addr: 0x20016820, size: 4, signed: true, idx: -1},
		{addr: 0x20016824, size: 4, signed: true, idx: -1},
		{addr: 0x20016828, size: 4, signed: true, idx: -1},
		{addr: 0x2001682c, size: 4, signed: true, idx: -1},
		{addr: 0x20016830, size: 4, signed: true, idx: -1},
		{addr: 0x20016834, size: 4, signed: false, idx: -1},
		{addr: 0x20016838, size: 4, signed: false, idx: -1},
		{addr: 0x2001683c, size: 4, signed: false, idx: -1},
		{addr: 0x20016840, size: 4, signed: false, idx: -1},
		{addr: 0x20016844, size: 4, signed: false, idx: -1},
		{addr: 0x20016848, size: 4, signed: false, idx: -1},
		{addr: 0x2001684c, size: 4, signed: false, idx: -1},
		{addr: 0x20016850, size: 4, signed: false, idx: -1},
		{addr: 0x20016854, size: 4, signed: false, idx: -1},
		{addr: 0x20016858, size: 4, signed: false, idx: -1},
		{addr: 0x2001685c, size: 4, signed: false, idx: -1},
		{addr: 0x20016860, size: 4, signed: false, idx: -1},
		{addr: 0x20016864, size: 4, signed: false, idx: -1},
		{addr: 0x20016868, size: 4, signed: false, idx: -1},
		{addr: 0x2001686c, size: 4, signed: false, idx: -1},
		{addr: 0x20016870, size: 4, signed: false, idx: -1},
		{addr: 0x20016874, size: 4, signed: false, idx: -1},
		{addr: 0x20016878, size: 4, signed: false, idx: -1},
		{addr: 0x2001687c, size: 4, signed: false, idx: -1},
		{addr: 0x20016880, size: 4, signed: false, idx: -1},
		{addr: 0x20016884, size: 4, signed: false, idx: -1},
		{addr: 0x20016888, size: 4, signed: false, idx: -1},
		{addr: 0x2001688c, size: 4, signed: false, idx: -1},
		{addr: 0x20016890, size: 4, signed: false, idx: -1},
		{addr: 0x20016894, size: 4, signed: false, idx: -1},
		{addr: 0x20016898, size: 4, signed: false, idx: -1},
		{addr: 0x2001689c, size: 4, signed: false, idx: -1},
		{addr: 0x200168a0, size: 4, signed: false, idx: -1},
		{addr: 0x200168a4, size: 4, signed: false, idx: -1},
		{addr: 0x200168a8, size: 4, signed: false, idx: -1},
		{addr: 0x200168ac, size: 4, signed: false, idx: -1},
		{addr: 0x200168b0, size: 4, signed: false, idx: -1},
		{addr: 0x200168b4, size: 4, signed: false, idx: -1},
		{addr: 0x200168b8, size: 4, signed: false, idx: -1},
		{addr: 0x200168bc, size: 4, signed: false, idx: -1},
		{addr: 0x200168c0, size: 4, signed: false, idx: -1},
		{addr: 0x200168c4, size: 4, signed: false, idx: -1},
		{addr: 0x200168c8, size: 4, signed: false, idx: -1},
		{addr: 0x200168cc, size: 4, signed: false, idx: -1},
		{addr: 0x200168d0, size: 4, signed: false, idx: -1},
		{addr: 0x200168d4, size: 4, signed: false, idx: -1},
		{addr: 0x200168d8, size: 4, signed: false, idx: -1},
		{addr: 0x200168dc, size: 4, signed: false, idx: -1},
		{addr: 0x200168e0, size: 4, signed: false, idx: -1},
		{addr: 0x200168e4, size: 4, signed: false, idx: -1},
		{addr: 0x200168e8, size: 4, signed: false, idx: -1},
		{addr: 0x200168ec, size: 4, signed: false, idx: -1},
		{addr: 0x200168f0, size: 4, signed: false, idx: -1},
		{addr: 0x200168f4, size: 4, signed: false, idx: -1},
		{addr: 0x200168f8, size: 4, signed: false, idx: -1},
		{addr: 0x200168fc, size: 4, signed: false, idx: -1},
		{addr: 0x20016900, size: 4, signed: false, idx: -1},
		{addr: 0x20016904, size: 4, signed: false, idx: -1},
		{addr: 0x20016908, size: 4, signed: false, idx: -1},
		{addr: 0x2001690c, size: 4, signed: false, idx: -1},
		{addr: 0x20016910, size: 4, signed: false, idx: -1},
		{addr: 0x20016914, size: 4, signed: false, idx: -1},
		{addr: 0x20016918, size: 4, signed: false, idx: -1},
		{addr: 0x2001691c, size: 4, signed: false, idx: -1},
		{addr: 0x20016920, size: 4, signed: false, idx: -1},
		{addr: 0x20016924, size: 4, signed: false, idx: -1},
		{addr: 0x20016928, size: 4, signed: false, idx: -1},
		{addr: 0x2001692c, size: 4, signed: false, idx: -1},
		{addr: 0x20016930, size: 4, signed: false, idx: -1},
		{addr: 0x20016934, size: 4, signed: false, idx: -1},
		{addr: 0x20016938, size: 4, signed: false, idx: -1},
		{addr: 0x2001693c, size: 4, signed: false, idx: -1},
		{addr: 0x20016940, size: 4, signed: false, idx: -1},
		{addr: 0x20016944, size: 4, signed: false, idx: -1},
		{addr: 0x20016948, size: 4, signed: false, idx: -1},
		{addr: 0x2001694c, size: 4, signed: false, idx: -1},
		{addr: 0x20016950, size: 4, signed: false, idx: -1},
		{addr: 0x20016954, size: 4, signed: false, idx: -1},
		{addr: 0x20016958, size: 4, signed: false, idx: -1},
		{addr: 0x2001695c, size: 4, signed: false, idx: -1},
		{addr: 0x20016960, size: 4, signed: false, idx: -1},
		{addr: 0x20016964, size: 4, signed: false, idx: -1},
		{addr: 0x20016968, size: 4, signed: false, idx: -1},
		{addr: 0x2001696c, size: 4, signed: false, idx: -1},
		{addr: 0x20016970, size: 4, signed: false, idx: -1},
		{addr: 0x20016974, size: 4, signed: false, idx: -1},
		{addr: 0x20016978, size: 4, signed: false, idx: -1},
		{addr: 0x2001697c, size: 4, signed: false, idx: -1},
		{addr: 0x20016980, size: 4, signed: false, idx: -1},
		{addr: 0x20016984, size: 4, signed: false, idx: -1},
		{addr: 0x20016988, size: 4, signed: false, idx: -1},
		{addr: 0x2001698c, size: 4, signed: false, idx: -1},
		{addr: 0x20016990, size: 4, signed: false, idx: -1},
		{addr: 0x20016994, size: 4, signed: false, idx: -1},
		{addr: 0x20016998, size: 4, signed: false, idx: -1},
		{addr: 0x2001699c, size: 4, signed: false, idx: -1},
		{addr: 0x200169a0, size: 4, signed: false, idx: -1},
		{addr: 0x200169a4, size: 4, signed: false, idx: -1},
		{addr: 0x200169a8, size: 4, signed: false, idx: -1},
		{addr: 0x200169ac, size: 4, signed: false, idx: -1},
		{addr: 0x200169b0, size: 4, signed: false, idx: -1},
		{addr: 0x200169b4, size: 4, signed: false, idx: -1},
		{addr: 0x200169b8, size: 4, signed: false, idx: -1},
		{addr: 0x200169bc, size: 4, signed: false, idx: -1},
		{addr: 0x200169c0, size: 4, signed: false, idx: -1},
		{addr: 0x200169c4, size: 4, signed: false, idx: -1},
		{addr: 0x200169c8, size: 4, signed: false, idx: -1},
		{addr: 0x200169cc, size: 4, signed: false, idx: -1},
		{addr: 0x200169d0, size: 4, signed: false, idx: -1},
		{addr: 0x200169d4, size: 4, signed: false, idx: -1},
		{addr: 0x200169d8, size: 4, signed: false, idx: -1},
		{addr: 0x200169dc, size: 4, signed: false, idx: -1},
		{addr: 0x200169e0, size: 4, signed: false, idx: -1},
		{addr: 0x200169e4, size: 4, signed: false, idx: -1},
		{addr: 0x200169e8, size: 4, signed: false, idx: -1},
		{addr: 0x200169ec, size: 4, signed: false, idx: -1},
		{addr: 0x200169f0, size: 4, signed: false, idx: -1},
		{addr: 0x200169f4, size: 4, signed: false, idx: -1},
		{addr: 0x200169f8, size: 4, signed: false, idx: -1},
		{addr: 0x200169fc, size: 4, signed: false, idx: -1},
		{addr: 0x20016a00, size: 4, signed: false, idx: -1},
		{addr: 0x20016a04, size: 4, signed: false, idx: -1},
		{addr: 0x20016a08, size: 4, signed: false, idx: -1},
		{addr: 0x20016a0c, size: 4, signed: false, idx: -1},
		{addr: 0x20016a10, size: 4, signed: false, idx: -1},
		{addr: 0x20016a14, size: 4, signed: false, idx: -1},
		{addr: 0x20016a18, size: 4, signed: false, idx: -1},
		{addr: 0x20016a1c, size: 4, signed: false, idx: -1},
		{addr: 0x20016a20, size: 4, signed: false, idx: -1},
		{addr: 0x20016a24, size: 4, signed: false, idx: -1},
		{addr: 0x20016a28, size: 4, signed: false, idx: -1},
		{addr: 0x20016a2c, size: 4, signed: false, idx: -1},
		{addr: 0x20016a30, size: 4, signed: false, idx: -1},
		{addr: 0x20016a34, size: 4, signed: false, idx: -1},
		{addr: 0x20016a38, size: 4, signed: false, idx: -1},
		{addr: 0x20016a3c, size: 4, signed: false, idx: -1},
		{addr: 0x20016a40, size: 4, signed: false, idx: -1},
		{addr: 0x20016a44, size: 4, signed: false, idx: -1},
		{addr: 0x20016a48, size: 4, signed: false, idx: -1},
		{addr: 0x20016a4c, size: 4, signed: false, idx: -1},
		{addr: 0x20016a50, size: 4, signed: false, idx: -1},
		{addr: 0x20016a54, size: 4, signed: false, idx: -1},
		{addr: 0x20016a58, size: 4, signed: false, idx: -1},
		{addr: 0x20016a5c, size: 4, signed: false, idx: -1},
		{addr: 0x20016a60, size: 4, signed: false, idx: -1},
		{addr: 0x20016a64, size: 4, signed: false, idx: -1},
		{addr: 0x20016a68, size: 4, signed: false, idx: -1},
		{addr: 0x20016a6c, size: 4, signed: false, idx: -1},
		{addr: 0x20016a70, size: 4, signed: false, idx: -1},
		{addr: 0x20016a74, size: 4, signed: false, idx: -1},
		{addr: 0x20016a78, size: 4, signed: false, idx: -1},
		{addr: 0x20016a7c, size: 4, signed: false, idx: -1},
		{addr: 0x20016a80, size: 4, signed: false, idx: -1},
		{addr: 0x20016a84, size: 4, signed: false, idx: -1},
		{addr: 0x20016a88, size: 4, signed: false, idx: -1},
		{addr: 0x20016a8c, size: 4, signed: false, idx: -1},
		{addr: 0x20016a90, size: 4, signed: false, idx: -1},
		{addr: 0x20016a94, size: 4, signed: false, idx: -1},
		{addr: 0x20016a98, size: 4, signed: false, idx: -1},
		{addr: 0x20016a9c, size: 4, signed: false, idx: -1},
		{addr: 0x20016aa0, size: 4, signed: false, idx: -1},
		{addr: 0x20016aa4, size: 4, signed: false, idx: -1},
		{addr: 0x20016aa8, size: 4, signed: false, idx: -1},
		{addr: 0x20016aac, size: 4, signed: false, idx: -1},
		{addr: 0x20016ab0, size: 4, signed: false, idx: -1},
		{addr: 0x20016ab4, size: 4, signed: false, idx: -1},
		{addr: 0x20016ab8, size: 4, signed: false, idx: -1},
		{addr: 0x20016abc, size: 4, signed: false, idx: -1},
		{addr: 0x20016ac0, size: 4, signed: false, idx: -1},
		{addr: 0x20016ac4, size: 4, signed: false, idx: -1},
		{addr: 0x20016ac8, size: 4, signed: false, idx: -1},
		{addr: 0x20016acc, size: 4, signed: false, idx: -1},
		{addr: 0x20016ad0, size: 4, signed: false, idx: -1},
		{addr: 0x20016ad4, size: 4, signed: false, idx: -1},
		{addr: 0x20016ad8, size: 4, signed: false, idx: -1},
		{addr: 0x20016adc, size: 4, signed: false, idx: -1},
		{addr: 0x20016ae0, size: 4, signed: false, idx: -1},
		{addr: 0x20016ae4, size: 4, signed: false, idx: -1},
		{addr: 0x20016ae8, size: 4, signed: false, idx: -1},
		{addr: 0x20016aec, size: 4, signed: false, idx: -1},
		{addr: 0x20016af0, size: 4, signed: false, idx: -1},
		{addr: 0x20016af4, size: 4, signed: false, idx: -1},
		{addr: 0x20016af8, size: 4, signed: false, idx: -1},
		{addr: 0x20016afc, size: 4, signed: false, idx: -1},
		{addr: 0x20016b00, size: 4, signed: false, idx: -1},
		{addr: 0x20016b04, size: 4, signed: false, idx: -1},
		{addr: 0x20016b08, size: 4, signed: false, idx: -1},
		{addr: 0x20016b0c, size: 4, signed: false, idx: -1},
		{addr: 0x20016b10, size: 4, signed: false, idx: -1},
		{addr: 0x20016b14, size: 4, signed: false, idx: -1},
		{addr: 0x20016b18, size: 4, signed: false, idx: -1},
		{addr: 0x20016b1c, size: 4, signed: false, idx: -1},
		{addr: 0x20016b20, size: 4, signed: false, idx: -1},
		{addr: 0x20016b24, size: 4, signed: false, idx: -1},
		{addr: 0x20016b28, size: 4, signed: false, idx: -1},
		{addr: 0x20016b2c, size: 4, signed: false, idx: -1},
		{addr: 0x20016b30, size: 4, signed: false, idx: -1},
		{addr: 0x20016b34, size: 4, signed: false, idx: -1},
		{addr: 0x20016b38, size: 4, signed: false, idx: -1},
		{addr: 0x20016b3c, size: 4, signed: false, idx: -1},
		{addr: 0x20016b40, size: 4, signed: false, idx: -1},
		{addr: 0x20016b44, size: 4, signed: false, idx: -1},
		{addr: 0x20016b48, size: 1, signed: false, idx: -1},
		{addr: 0x20016b4c, size: 1, signed: false, idx: -1},
		{addr: 0x20016b50, size: 1, signed: false, idx: -1},
		{addr: 0x20016b54, size: 1, signed: false, idx: -1},
		{addr: 0x20016b58, size: 1, signed: false, idx: -1},
		{addr: 0x20016b5c, size: 1, signed: false, idx: -1},
		{addr: 0x20016b60, size: 1, signed: false, idx: -1},
		{addr: 0x20016b64, size: 1, signed: false, idx: -1},
		{addr: 0x20016b68, size: 1, signed: false, idx: -1},
		{addr: 0x20016b6c, size: 1, signed: false, idx: -1},
		{addr: 0x20016b70, size: 1, signed: false, idx: -1},
		{addr: 0x20016b74, size: 1, signed: false, idx: -1},
		{addr: 0x20016b78, size: 1, signed: false, idx: -1},
		{addr: 0x20016b7c, size: 1, signed: false, idx: -1},
		{addr: 0x20016b80, size: 1, signed: false, idx: -1},
		{addr: 0x20016b84, size: 1, signed: false, idx: -1},
		{addr: 0x20016b88, size: 1, signed: false, idx: -1},
		{addr: 0x20016b8c, size: 1, signed: false, idx: -1},
		{addr: 0x20016b90, size: 1, signed: false, idx: -1},
		{addr: 0x20016b94, size: 1, signed: false, idx: -1},
		{addr: 0x20016b98, size: 1, signed: false, idx: -1},
		{addr: 0x20016b9c, size: 1, signed: false, idx: -1},
		{addr: 0x20016ba0, size: 1, signed: false, idx: -1},
		{addr: 0x20016ba4, size: 1, signed: false, idx: -1},
		{addr: 0x20016ba8, size: 1, signed: false, idx: -1},
		{addr: 0x20016bac, size: 1, signed: false, idx: -1},
		{addr: 0x20016bb0, size: 1, signed: false, idx: -1},
		{addr: 0x20016bb4, size: 1, signed: false, idx: -1},
		{addr: 0x20016bb8, size: 1, signed: false, idx: -1},
		{addr: 0x20016bbc, size: 1, signed: false, idx: -1},
		{addr: 0x20016bc0, size: 1, signed: false, idx: -1},
		{addr: 0x20016bc4, size: 1, signed: false, idx: -1},
		{addr: 0x20016bc8, size: 1, signed: false, idx: -1},
		{addr: 0x20016bcc, size: 1, signed: false, idx: -1},
		{addr: 0x20016bd0, size: 1, signed: false, idx: -1},
		{addr: 0x20016bd4, size: 1, signed: false, idx: -1},
		{addr: 0x20016bd8, size: 1, signed: false, idx: -1},
		{addr: 0x20016bdc, size: 1, signed: false, idx: -1},
		{addr: 0x20016be0, size: 1, signed: false, idx: -1},
		{addr: 0x20016be4, size: 1, signed: false, idx: -1},
		{addr: 0x20016be8, size: 1, signed: false, idx: -1},
		{addr: 0x20016bec, size: 4, signed: true, idx: -1},
		{addr: 0x20016bf0, size: 4, signed: true, idx: -1},
		{addr: 0x20016bf4, size: 4, signed: true, idx: -1},
		{addr: 0x20016bf8, size: 4, signed: true, idx: -1},
		{addr: 0x20016bfc, size: 4, signed: true, idx: -1},
		{addr: 0x20016c00, size: 4, signed: true, idx: -1},
		{addr: 0x20016c04, size: 4, signed: true, idx: -1},
		{addr: 0x20016c08, size: 4, signed: true, idx: -1},
		{addr: 0x20016c0c, size: 4, signed: true, idx: -1},
		{addr: 0x20016c10, size: 4, signed: true, idx: -1},
		{addr: 0x20016c14, size: 4, signed: true, idx: -1},
		{addr: 0x20016c18, size: 4, signed: true, idx: -1},
		{addr: 0x20016c1c, size: 4, signed: true, idx: -1},
		{addr: 0x20016c20, size: 4, signed: true, idx: -1},
		{addr: 0x20016c24, size: 4, signed: true, idx: -1},
		{addr: 0x20016c28, size: 4, signed: true, idx: -1},
		{addr: 0x20016c2c, size: 4, signed: true, idx: -1},
		{addr: 0x20016c30, size: 4, signed: true, idx: -1},
		{addr: 0x20016c34, size: 4, signed: true, idx: -1},
		{addr: 0x20016c38, size: 4, signed: true, idx: -1},
		{addr: 0x20016c3c, size: 4, signed: true, idx: -1},
		{addr: 0x20016c40, size: 4, signed: true, idx: -1},
		{addr: 0x20016c44, size: 4, signed: true, idx: -1},
		{addr: 0x20016c48, size: 4, signed: true, idx: -1},
		{addr: 0x20016c4c, size: 4, signed: true, idx: -1},
		{addr: 0x20016c50, size: 4, signed: true, idx: -1},
		{addr: 0x20016c54, size: 4, signed: true, idx: -1},
		{addr: 0x20016c58, size: 4, signed: true, idx: -1},
		{addr: 0x20016c5c, size: 4, signed: true, idx: -1},
		{addr: 0x20016c60, size: 4, signed: true, idx: -1},
		{addr: 0x20016c64, size: 4, signed: true, idx: -1},
		{addr: 0x20016c68, size: 4, signed: true, idx: -1},
		{addr: 0x20016c6c, size: 4, signed: true, idx: -1},
		{addr: 0x20016c70, size: 4, signed: true, idx: -1},
		{addr: 0x20016c74, size: 4, signed: true, idx: -1},
		{addr: 0x20016c78, size: 4, signed: true, idx: -1},
		{addr: 0x20016c7c, size: 4, signed: true, idx: -1},
		{addr: 0x20016c80, size: 4, signed: true, idx: -1},
		{addr: 0x20016c84, size: 4, signed: true, idx: -1},
		{addr: 0x20016c88, size: 4, signed: true, idx: -1},
		{addr: 0x20016c8c, size: 4, signed: true, idx: -1},
		{addr: 0x20016c90, size: 4, signed: false, idx: -1},
		{addr: 0x20016c94, size: 4, signed: false, idx: -1},
		{addr: 0x20016c98, size: 4, signed: false, idx: -1},
		{addr: 0x20016c9c, size: 4, signed: false, idx: -1},
		{addr: 0x20016ca0, size: 4, signed: false, idx: -1},
		{addr: 0x20016ca4, size: 4, signed: false, idx: -1},
		{addr: 0x20016ca8, size: 4, signed: false, idx: -1},
		{addr: 0x20016cac, size: 4, signed: false, idx: -1},
		{addr: 0x20016cb0, size: 4, signed: false, idx: -1},
		{addr: 0x20016cb4, size: 4, signed: false, idx: -1},
		{addr: 0x20016cb8, size: 4, signed: false, idx: -1},
		{addr: 0x20016cbc, size: 4, signed: false, idx: -1},
		{addr: 0x20016cc0, size: 4, signed: false, idx: -1},
		{addr: 0x20016cc4, size: 4, signed: false, idx: -1},
		{addr: 0x20016cc8, size: 4, signed: false, idx: -1},
		{addr: 0x20016ccc, size: 4, signed: false, idx: -1},
		{addr: 0x20016cd0, size: 4, signed: false, idx: -1},
		{addr: 0x20016cd4, size: 4, signed: false, idx: -1},
		{addr: 0x20016cd8, size: 4, signed: false, idx: -1},
		{addr: 0x20016cdc, size: 4, signed: false, idx: -1},
		{addr: 0x20016ce0, size: 4, signed: false, idx: -1},
		{addr: 0x20016ce4, size: 4, signed: false, idx: -1},
		{addr: 0x20016ce8, size: 4, signed: false, idx: -1},
		{addr: 0x20016cec, size: 4, signed: false, idx: -1},
		{addr: 0x20016cf0, size: 4, signed: false, idx: -1},
		{addr: 0x20016cf4, size: 4, signed: false, idx: -1},
		{addr: 0x20016cf8, size: 4, signed: false, idx: -1},
		{addr: 0x20016cfc, size: 4, signed: false, idx: -1},
		{addr: 0x20016d00, size: 4, signed: false, idx: -1},
		{addr: 0x20016d04, size: 4, signed: false, idx: -1},
		{addr: 0x20016d08, size: 4, signed: false, idx: -1},
		{addr: 0x20016d0c, size: 4, signed: false, idx: -1},
		{addr: 0x20016d10, size: 4, signed: false, idx: -1},
		{addr: 0x20016d14, size: 4, signed: false, idx: -1},
		{addr: 0x20016d18, size: 4, signed: false, idx: -1},
		{addr: 0x20016d1c, size: 4, signed: false, idx: -1},
		{addr: 0x20016d20, size: 4, signed: false, idx: -1},
		{addr: 0x20016d24, size: 4, signed: false, idx: -1},
		{addr: 0x20016d28, size: 4, signed: false, idx: -1},
		{addr: 0x20016d2c, size: 4, signed: false, idx: -1},
		{addr: 0x20016d30, size: 4, signed: false, idx: -1},
		{addr: 0x20016d34, size: 4, signed: false, idx: -1},
		{addr: 0x20016d38, size: 1, signed: false, idx: -1},
		{addr: 0x20016d3c, size: 1, signed: false, idx: -1},
		{addr: 0x20016d40, size: 1, signed: false, idx: -1},
		{addr: 0x20016d44, size: 1, signed: false, idx: -1},
		{addr: 0x20016d48, size: 1, signed: false, idx: -1},
		{addr: 0x20016d4c, size: 1, signed: false, idx: -1},
		{addr: 0x20016d50, size: 1, signed: false, idx: -1},
		{addr: 0x20016d54, size: 1, signed: false, idx: -1},
		{addr: 0x20016d58, size: 1, signed: false, idx: -1},
		{addr: 0x20016d5c, size: 1, signed: false, idx: -1},
		{addr: 0x20016d60, size: 1, signed: false, idx: -1},
		{addr: 0x20016d64, size: 1, signed: false, idx: -1},
		{addr: 0x20016d68, size: 1, signed: false, idx: -1},
		{addr: 0x20016d6c, size: 1, signed: false, idx: -1},
		{addr: 0x20016d70, size: 1, signed: false, idx: -1},
		{addr: 0x20016d74, size: 1, signed: false, idx: -1},
		{addr: 0x20016d78, size: 1, signed: false, idx: -1},
		{addr: 0x20016d7c, size: 1, signed: false, idx: -1},
		{addr: 0x20016d80, size: 1, signed: false, idx: -1},
		{addr: 0x20016d84, size: 1, signed: false, idx: -1},
		{addr: 0x20016d88, size: 1, signed: false, idx: -1},
		{addr: 0x20016d8c, size: 1, signed: false, idx: -1},
		{addr: 0x20016d90, size: 1, signed: false, idx: -1},
		{addr: 0x20016d94, size: 1, signed: false, idx: -1},
		{addr: 0x20016d98, size: 1, signed: false, idx: -1},
		{addr: 0x20016d9c, size: 1, signed: false, idx: -1},
		{addr: 0x20016da0, size: 1, signed: false, idx: -1},
		{addr: 0x20016da4, size: 1, signed: false, idx: -1},
		{addr: 0x20016da8, size: 1, signed: false, idx: -1},
		{addr: 0x20016dac, size: 1, signed: false, idx: -1},
		{addr: 0x20016db0, size: 1, signed: false, idx: -1},
		{addr: 0x20016db4, size: 1, signed: false, idx: -1},
		{addr: 0x20016db8, size: 1, signed: false, idx: -1},
		{addr: 0x20016dbc, size: 1, signed: false, idx: -1},
		{addr: 0x20016dc0, size: 1, signed: false, idx: -1},
		{addr: 0x20016dc4, size: 1, signed: false, idx: -1},
		{addr: 0x20016dc8, size: 1, signed: false, idx: -1},
		{addr: 0x20016dcc, size: 1, signed: false, idx: -1},
		{addr: 0x20016dd0, size: 1, signed: false, idx: -1},
		{addr: 0x20016dd4, size: 1, signed: false, idx: -1},
		{addr: 0x20016dd8, size: 1, signed: false, idx: -1},
		{addr: 0x20016ddc, size: 1, signed: false, idx: -1},
		{addr: 0x20016de0, size: 1, signed: false, idx: -1},
		{addr: 0x20016de4, size: 1, signed: false, idx: -1},
		{addr: 0x20016de8, size: 1, signed: false, idx: -1},
		{addr: 0x20016dec, size: 1, signed: false, idx: -1},
		{addr: 0x20016df0, size: 1, signed: false, idx: -1},
		{addr: 0x20016df4, size: 1, signed: false, idx: -1},
		{addr: 0x20016df8, size: 1, signed: false, idx: -1},
		{addr: 0x20016dfc, size: 1, signed: false, idx: -1},
		{addr: 0x20016e00, size: 1, signed: false, idx: -1},
		{addr: 0x20016e04, size: 1, signed: false, idx: -1},
		{addr: 0x20016e08, size: 1, signed: false, idx: -1},
		{addr: 0x20016e0c, size: 1, signed: false, idx: -1},
		{addr: 0x20016e10, size: 1, signed: false, idx: -1},
		{addr: 0x20016e14, size: 1, signed: false, idx: -1},
		{addr: 0x20016e18, size: 1, signed: false, idx: -1},
		{addr: 0x20016e1c, size: 1, signed: false, idx: -1},
		{addr: 0x20016e20, size: 1, signed: false, idx: -1},
		{addr: 0x20016e24, size: 1, signed: false, idx: -1},
		{addr: 0x20016e28, size: 1, signed: false, idx: -1},
		{addr: 0x20016e2c, size: 1, signed: false, idx: -1},
		{addr: 0x20016e30, size: 1, signed: false, idx: -1},
		{addr: 0x20016e34, size: 1, signed: false, idx: -1},
		{addr: 0x20016e38, size: 1, signed: false, idx: -1},
		{addr: 0x20016e3c, size: 1, signed: false, idx: -1},
		{addr: 0x20016e40, size: 1, signed: false, idx: -1},
		{addr: 0x20016e44, size: 1, signed: false, idx: -1},
		{addr: 0x20016e48, size: 1, signed: false, idx: -1},
		{addr: 0x20016e4c, size: 1, signed: false, idx: -1},
		{addr: 0x20016e50, size: 1, signed: false, idx: -1},
		{addr: 0x20016e54, size: 1, signed: false, idx: -1},
		{addr: 0x20016e58, size: 1, signed: false, idx: -1},
		{addr: 0x20016e5c, size: 1, signed: false, idx: -1},
		{addr: 0x20016e60, size: 1, signed: false, idx: -1},
		{addr: 0x20016e64, size: 1, signed: false, idx: -1},
		{addr: 0x20016e68, size: 1, signed: false, idx: -1},
		{addr: 0x20016e6c, size: 1, signed: false, idx: -1},
		{addr: 0x20016e70, size: 1, signed: false, idx: -1},
		{addr: 0x20016e74, size: 1, signed: false, idx: -1},
		{addr: 0x20016e78, size: 1, signed: false, idx: -1},
		{addr: 0x20016e7c, size: 1, signed: false, idx: -1},
		{addr: 0x20016e80, size: 1, signed: false, idx: -1},
		{addr: 0x20016e84, size: 1, signed: false, idx: -1},
		{addr: 0x20016e88, size: 1, signed: false, idx: -1},
		{addr: 0x20016e8c, size: 1, signed: false, idx: -1},
		{addr: 0x20016e90, size: 1, signed: false, idx: -1},
		{addr: 0x20016e94, size: 1, signed: false, idx: -1},
		{addr: 0x20016e98, size: 1, signed: false, idx: -1},
		{addr: 0x20016e9c, size: 1, signed: false, idx: -1},
		{addr: 0x20016ea0, size: 1, signed: false, idx: -1},
		{addr: 0x20016ea4, size: 1, signed: false, idx: -1},
		{addr: 0x20016ea8, size: 1, signed: false, idx: -1},
		{addr: 0x20016eac, size: 1, signed: false, idx: -1},
		{addr: 0x20016eb0, size: 1, signed: false, idx: -1},
		{addr: 0x20016eb4, size: 1, signed: false, idx: -1},
		{addr: 0x20016eb8, size: 1, signed: false, idx: -1},
		{addr: 0x20016ebc, size: 1, signed: false, idx: -1},
		{addr: 0x20016ec0, size: 1, signed: false, idx: -1},
		{addr: 0x20016ec4, size: 1, signed: false, idx: -1},
		{addr: 0x20016ec8, size: 1, signed: false, idx: -1},
		{addr: 0x20016ecc, size: 1, signed: false, idx: -1},
		{addr: 0x20016ed0, size: 1, signed: false, idx: -1},
		{addr: 0x20016ed4, size: 1, signed: false, idx: -1},
		{addr: 0x20016ed8, size: 1, signed: false, idx: -1},
		{addr: 0x20016edc, size: 1, signed: false, idx: -1},
		{addr: 0x20016ee0, size: 1, signed: false, idx: -1},
		{addr: 0x20016ee4, size: 1, signed: false, idx: -1},
		{addr: 0x20016ee8, size: 1, signed: false, idx: -1},
		{addr: 0x20016eec, size: 1, signed: false, idx: -1},
		{addr: 0x20016ef0, size: 1, signed: false, idx: -1},
		{addr: 0x20016ef4, size: 1, signed: false, idx: -1},
		{addr: 0x20016ef8, size: 1, signed: false, idx: -1},
		{addr: 0x20016efc, size: 1, signed: false, idx: -1},
		{addr: 0x20016f00, size: 1, signed: false, idx: -1},
		{addr: 0x20016f04, size: 1, signed: false, idx: -1},
		{addr: 0x20016f08, size: 1, signed: false, idx: -1},
		{addr: 0x20016f0c, size: 1, signed: false, idx: -1},
		{addr: 0x20016f10, size: 1, signed: false, idx: -1},
		{addr: 0x20016f14, size: 1, signed: false, idx: -1},
		{addr: 0x20016f18, size: 1, signed: false, idx: -1},
		{addr: 0x20016f1c, size: 1, signed: false, idx: -1},
		{addr: 0x20016f20, size: 1, signed: false, idx: -1},
		{addr: 0x20016f24, size: 1, signed: false, idx: -1},
		{addr: 0x20016f28, size: 1, signed: false, idx: -1},
		{addr: 0x20016f2c, size: 1, signed: false, idx: -1},
		{addr: 0x20016f30, size: 1, signed: false, idx: -1},
		{addr: 0x20016f34, size: 1, signed: false, idx: -1},
		{addr: 0x20016f38, size: 1, signed: false, idx: -1},
		{addr: 0x20016f3c, size: 1, signed: false, idx: -1},
		{addr: 0x20016f40, size: 1, signed: false, idx: -1},
		{addr: 0x20016f44, size: 1, signed: false, idx: -1},
		{addr: 0x20016f48, size: 1, signed: false, idx: -1},
		{addr: 0x20016f4c, size: 1, signed: false, idx: -1},
		{addr: 0x20016f50, size: 1, signed: false, idx: -1},
		{addr: 0x20016f54, size: 1, signed: false, idx: -1},
		{addr: 0x20016f58, size: 1, signed: false, idx: -1},
		{addr: 0x20016f5c, size: 1, signed: false, idx: -1},
		{addr: 0x20016f60, size: 1, signed: false, idx: -1},
		{addr: 0x20016f64, size: 1, signed: false, idx: -1},
		{addr: 0x20016f68, size: 1, signed: false, idx: -1},
		{addr: 0x20016f6c, size: 1, signed: false, idx: -1},
		{addr: 0x20016f70, size: 1, signed: false, idx: -1},
		{addr: 0x20016f74, size: 1, signed: false, idx: -1},
		{addr: 0x20016f78, size: 1, signed: false, idx: -1},
		{addr: 0x20016f7c, size: 1, signed: false, idx: -1},
		{addr: 0x20016f80, size: 1, signed: false, idx: -1},
		{addr: 0x20016f84, size: 1, signed: false, idx: -1},
		{addr: 0x20016f88, size: 1, signed: false, idx: -1},
		{addr: 0x20016f8c, size: 1, signed: false, idx: -1},
		{addr: 0x20016f90, size: 1, signed: false, idx: -1},
		{addr: 0x20016f94, size: 1, signed: false, idx: -1},
		{addr: 0x20016f98, size: 1, signed: false, idx: -1},
		{addr: 0x20016f9c, size: 1, signed: false, idx: -1},
		{addr: 0x20016fa0, size: 1, signed: false, idx: -1},
		{addr: 0x20016fa4, size: 1, signed: false, idx: -1},
		{addr: 0x20016fa8, size: 1, signed: false, idx: -1},
		{addr: 0x20016fac, size: 1, signed: false, idx: -1},
		{addr: 0x20016fb0, size: 1, signed: false, idx: -1},
		{addr: 0x20016fb4, size: 1, signed: false, idx: -1},
		{addr: 0x20016fb8, size: 1, signed: false, idx: -1},
		{addr: 0x20016fbc, size: 1, signed: false, idx: -1},
		{addr: 0x20016fc0, size: 1, signed: false, idx: -1},
		{addr: 0x20016fc4, size: 1, signed: false, idx: -1},
		{addr: 0x20016fc8, size: 1, signed: false, idx: -1},
		{addr: 0x20016fcc, size: 1, signed: false, idx: -1},
		{addr: 0x20016fd0, size: 1, signed: false, idx: -1},
		{addr: 0x20016fd4, size: 1, signed: false, idx: -1},
		{addr: 0x20016fd8, size: 1, signed: false, idx: -1},
		{addr: 0x20016fdc, size: 1, signed: false, idx: -1},
		{addr: 0x20016fe0, size: 1, signed: false, idx: -1},
		{addr: 0x20016fe4, size: 1, signed: false, idx: -1},
		{addr: 0x20016fe8, size: 1, signed: false, idx: -1},
		{addr: 0x20016fec, size: 1, signed: false, idx: -1},
		{addr: 0x20016ff0, size: 1, signed: false, idx: -1},
		{addr: 0x20016ff4, size: 1, signed: false, idx: -1},
		{addr: 0x20016ff8, size: 1, signed: false, idx: -1},
		{addr: 0x20016ffc, size: 1, signed: false, idx: -1},
		{addr: 0x20017000, size: 1, signed: false, idx: -1},
		{addr: 0x20017004, size: 1, signed: false, idx: -1},
		{addr: 0x20017008, size: 1, signed: false, idx: -1},
		{addr: 0x2001700c, size: 1, signed: false, idx: -1},
		{addr: 0x20017010, size: 1, signed: false, idx: -1},
		{addr: 0x20017014, size: 1, signed: false, idx: -1},
		{addr: 0x20017018, size: 1, signed: false, idx: -1},
		{addr: 0x2001701c, size: 1, signed: false, idx: -1},
		{addr: 0x20017020, size: 1, signed: false, idx: -1},
		{addr: 0x20017024, size: 1, signed: false, idx: -1},
		{addr: 0x20017028, size: 1, signed: false, idx: -1},
		{addr: 0x2001702c, size: 1, signed: false, idx: -1},
		{addr: 0x20017030, size: 1, signed: false, idx: -1},
		{addr: 0x20017034, size: 1, signed: false, idx: -1},
		{addr: 0x20017038, size: 1, signed: false, idx: -1},
		{addr: 0x2001703c, size: 1, signed: false, idx: -1},
		{addr: 0x20017040, size: 1, signed: false, idx: -1},
		{addr: 0x20017044, size: 1, signed: false, idx: -1},
		{addr: 0x20017048, size: 1, signed: false, idx: -1},
		{addr: 0x2001704c, size: 1, signed: false, idx: -1},
		{addr: 0x20017050, size: 1, signed: false, idx: -1},
		{addr: 0x20017054, size: 1, signed: false, idx: -1},
		{addr: 0x20017058, size: 1, signed: false, idx: -1},
		{addr: 0x2001705c, size: 1, signed: false, idx: -1},
		{addr: 0x20017060, size: 1, signed: false, idx: -1},
		{addr: 0x20017064, size: 1, signed: false, idx: -1},
		{addr: 0x20017068, size: 1, signed: false, idx: -1},
		{addr: 0x2001706c, size: 1, signed: false, idx: -1},
		{addr: 0x20017070, size: 1, signed: false, idx: -1},
		{addr: 0x20017074, size: 1, signed: false, idx: -1},
		{addr: 0x20017078, size: 1, signed: false, idx: -1},
		{addr: 0x2001707c, size: 1, signed: false, idx: -1},
		{addr: 0x20017080, size: 1, signed: false, idx: -1},
		{addr: 0x20017084, size: 1, signed: false, idx: -1},
		{addr: 0x20017088, size: 1, signed: false, idx: -1},
		{addr: 0x2001708c, size: 1, signed: false, idx: -1},
		{addr: 0x20017090, size: 1, signed: false, idx: -1},
		{addr: 0x20017094, size: 1, signed: false, idx: -1},
		{addr: 0x20017098, size: 1, signed: false, idx: -1},
		{addr: 0x2001709c, size: 1, signed: false, idx: -1},
		{addr: 0x200170a0, size: 1, signed: false, idx: -1},
		{addr: 0x200170a4, size: 1, signed: false, idx: -1},
		{addr: 0x200170a8, size: 1, signed: false, idx: -1},
		{addr: 0x200170ac, size: 1, signed: false, idx: -1},
		{addr: 0x200170b0, size: 1, signed: false, idx: -1},
		{addr: 0x200170b4, size: 1, signed: false, idx: -1},
		{addr: 0x200170b8, size: 1, signed: false, idx: -1},
		{addr: 0x200170bc, size: 1, signed: false, idx: -1},
		{addr: 0x200170c0, size: 1, signed: false, idx: -1},
		{addr: 0x200170c4, size: 1, signed: false, idx: -1},
		{addr: 0x200170c8, size: 1, signed: false, idx: -1},
		{addr: 0x200170cc, size: 1, signed: false, idx: -1},
		{addr: 0x200170d0, size: 1, signed: false, idx: -1},
		{addr: 0x200170d4, size: 1, signed: false, idx: -1},
		{addr: 0x200170d8, size: 1, signed: false, idx: -1},
		{addr: 0x200170dc, size: 1, signed: false, idx: -1},
		{addr: 0x200170e0, size: 1, signed: false, idx: -1},
		{addr: 0x200170e4, size: 1, signed: false, idx: -1},
		{addr: 0x200170e8, size: 1, signed: false, idx: -1},
		{addr: 0x200170ec, size: 1, signed: false, idx: -1},
		{addr: 0x200170f0, size: 1, signed: false, idx: -1},
		{addr: 0x200170f4, size: 1, signed: false, idx: -1},
		{addr: 0x200170f8, size: 1, signed: false, idx: -1},
		{addr: 0x200170fc, size: 1, signed: false, idx: -1},
		{addr: 0x20017100, size: 1, signed: false, idx: -1},
		{addr: 0x20017104, size: 1, signed: false, idx: -1},
		{addr: 0x20017108, size: 1, signed: false, idx: -1},
		{addr: 0x2001710c, size: 1, signed: false, idx: -1},
		{addr: 0x20017110, size: 1, signed: false, idx: -1},
		{addr: 0x20017114, size: 1, signed: false, idx: -1},
		{addr: 0x20017118, size: 1, signed: false, idx: -1},
		{addr: 0x2001711c, size: 1, signed: false, idx: -1},
		{addr: 0x20017120, size: 1, signed: false, idx: -1},
		{addr: 0x20017124, size: 1, signed: false, idx: -1},
		{addr: 0x20017128, size: 1, signed: false, idx: -1},
		{addr: 0x2001712c, size: 1, signed: false, idx: -1},
		{addr: 0x20017130, size: 1, signed: false, idx: -1},
		{addr: 0x20017134, size: 1, signed: false, idx: -1},
		{addr: 0x20017138, size: 1, signed: false, idx: -1},
		{addr: 0x2001713c, size: 1, signed: false, idx: -1},
		{addr: 0x20017140, size: 1, signed: false, idx: -1},
		{addr: 0x20017144, size: 1, signed: false, idx: -1},
		{addr: 0x20017148, size: 1, signed: false, idx: -1},
		{addr: 0x2001714c, size: 1, signed: false, idx: -1},
		{addr: 0x20017150, size: 1, signed: false, idx: -1},
		{addr: 0x20017154, size: 1, signed: false, idx: -1},
		{addr: 0x20017158, size: 1, signed: false, idx: -1},
		{addr: 0x2001715c, size: 1, signed: false, idx: -1},
		{addr: 0x20017160, size: 1, signed: false, idx: -1},
		{addr: 0x20017164, size: 1, signed: false, idx: -1},
		{addr: 0x20017168, size: 1, signed: false, idx: -1},
		{addr: 0x2001716c, size: 1, signed: false, idx: -1},
		{addr: 0x20017170, size: 1, signed: false, idx: -1},
		{addr: 0x20017174, size: 1, signed: false, idx: -1},
		{addr: 0x20017178, size: 1, signed: false, idx: -1},
		{addr: 0x2001717c, size: 1, signed: false, idx: -1},
		{addr: 0x20017180, size: 1, signed: false, idx: -1},
		{addr: 0x20017184, size: 1, signed: false, idx: -1},
		{addr: 0x20017188, size: 1, signed: false, idx: -1},
		{addr: 0x2001718c, size: 1, signed: false, idx: -1},
		{addr: 0x20017190, size: 1, signed: false, idx: -1},
		{addr: 0x20017194, size: 1, signed: false, idx: -1},
		{addr: 0x20017198, size: 1, signed: false, idx: -1},
		{addr: 0x2001719c, size: 1, signed: false, idx: -1},
		{addr: 0x200171a0, size: 1, signed: false, idx: -1},
		{addr: 0x200171a4, size: 1, signed: false, idx: -1},
		{addr: 0x200171a8, size: 1, signed: false, idx: -1},
		{addr: 0x200171ac, size: 1, signed: false, idx: -1},
		{addr: 0x200171b0, size: 1, signed: false, idx: -1},
		{addr: 0x200171b4, size: 1, signed: false, idx: -1},
		{addr: 0x200171b8, size: 1, signed: false, idx: -1},
		{addr: 0x200171bc, size: 1, signed: false, idx: -1},
		{addr: 0x200171c0, size: 1, signed: false, idx: -1},
		{addr: 0x200171c4, size: 1, signed: false, idx: -1},
		{addr: 0x200171c8, size: 1, signed: false, idx: -1},
		{addr: 0x200171cc, size: 1, signed: false, idx: -1},
		{addr: 0x200171d0, size: 1, signed: false, idx: -1},
		{addr: 0x200171d4, size: 1, signed: false, idx: -1},
		{addr: 0x200171d8, size: 1, signed: false, idx: -1},
		{addr: 0x200171dc, size: 1, signed: false, idx: -1},
		{addr: 0x200171e0, size: 1, signed: false, idx: -1},
		{addr: 0x200171e4, size: 1, signed: false, idx: -1},
		{addr: 0x200171e8, size: 1, signed: false, idx: -1},
		{addr: 0x200171ec, size: 1, signed: false, idx: -1},
		{addr: 0x200171f0, size: 1, signed: false, idx: -1},
		{addr: 0x200171f4, size: 1, signed: false, idx: -1},
		{addr: 0x200171f8, size: 1, signed: false, idx: -1},
		{addr: 0x200171fc, size: 1, signed: false, idx: -1},
		{addr: 0x20017200, size: 1, signed: false, idx: -1},
		{addr: 0x20017204, size: 1, signed: false, idx: -1},
		{addr: 0x20017208, size: 1, signed: false, idx: -1},
		{addr: 0x2001720c, size: 1, signed: false, idx: -1},
		{addr: 0x20017210, size: 1, signed: false, idx: -1},
		{addr: 0x20017214, size: 1, signed: false, idx: -1},
		{addr: 0x20017218, size: 1, signed: false, idx: -1},
		{addr: 0x2001721c, size: 1, signed: false, idx: -1},
		{addr: 0x20017220, size: 1, signed: false, idx: -1},
		{addr: 0x20017224, size: 1, signed: false, idx: -1},
		{addr: 0x20017228, size: 1, signed: false, idx: -1},
		{addr: 0x2001722c, size: 1, signed: false, idx: -1},
		{addr: 0x20017230, size: 1, signed: false, idx: -1},
		{addr: 0x20017234, size: 1, signed: false, idx: -1},
		{addr: 0x20017238, size: 1, signed: false, idx: -1},
		{addr: 0x2001723c, size: 1, signed: false, idx: -1},
		{addr: 0x20017240, size: 1, signed: false, idx: -1},
		{addr: 0x20017244, size: 1, signed: false, idx: -1},
		{addr: 0x20017248, size: 1, signed: false, idx: -1},
		{addr: 0x2001724c, size: 1, signed: false, idx: -1},
		{addr: 0x20017250, size: 1, signed: false, idx: -1},
		{addr: 0x20017254, size: 1, signed: false, idx: -1},
		{addr: 0x20017258, size: 4, signed: false, idx: -1},
		{addr: 0x2001725c, size: 4, signed: false, idx: -1},
	},
	out: []mstFld{
		{addr: 0x2000e22c, size: 1, signed: false, idx: -1},
		{addr: 0x2000e22d, size: 1, signed: false, idx: -1},
		{addr: 0x2000e42c, size: 1, signed: false, idx: -1},
		{addr: 0x2000e42d, size: 1, signed: false, idx: -1},
		{addr: 0x200012b6, size: 1, signed: false, idx: -1},
		{addr: 0x200012b7, size: 1, signed: false, idx: -1},
		{addr: 0x200012b8, size: 1, signed: false, idx: -1},
		{addr: 0x20002b18, size: 1, signed: false, idx: -1},
		{addr: 0x20002b19, size: 1, signed: false, idx: -1},
		{addr: 0x20002b1a, size: 1, signed: false, idx: -1},
		{addr: 0x2000437a, size: 1, signed: false, idx: -1},
		{addr: 0x2000437b, size: 1, signed: false, idx: -1},
		{addr: 0x2000437c, size: 1, signed: false, idx: -1},
		{addr: 0x20005bdc, size: 1, signed: false, idx: -1},
		{addr: 0x20005bdd, size: 1, signed: false, idx: -1},
		{addr: 0x20005bde, size: 1, signed: false, idx: -1},
		{addr: 0x2000743e, size: 1, signed: false, idx: -1},
		{addr: 0x2000743f, size: 1, signed: false, idx: -1},
		{addr: 0x20007440, size: 1, signed: false, idx: -1},
		{addr: 0x20008ca0, size: 1, signed: false, idx: -1},
		{addr: 0x20008ca1, size: 1, signed: false, idx: -1},
		{addr: 0x20008ca2, size: 1, signed: false, idx: -1},
		{addr: 0x2000a502, size: 1, signed: false, idx: -1},
		{addr: 0x2000a503, size: 1, signed: false, idx: -1},
		{addr: 0x2000a504, size: 1, signed: false, idx: -1},
		{addr: 0x2000bd64, size: 1, signed: false, idx: -1},
		{addr: 0x2000bd65, size: 1, signed: false, idx: -1},
		{addr: 0x2000bd66, size: 1, signed: false, idx: -1},
		{addr: 0x200148f4, size: 4, signed: false, idx: -1},
		{addr: 0x20015000, size: 4, signed: false, idx: -1},
		{addr: 0x20015040, size: 4, signed: false, idx: -1},
		{addr: 0x20015044, size: 4, signed: false, idx: -1},
		{addr: 0x20015048, size: 4, signed: false, idx: -1},
		{addr: 0x2001504c, size: 4, signed: false, idx: -1},
		{addr: 0x20015050, size: 4, signed: false, idx: -1},
		{addr: 0x20015054, size: 4, signed: false, idx: -1},
		{addr: 0x20015058, size: 4, signed: false, idx: -1},
		{addr: 0x2001505c, size: 4, signed: false, idx: -1},
		{addr: 0x20015060, size: 4, signed: false, idx: -1},
		{addr: 0x20015064, size: 4, signed: false, idx: -1},
		{addr: 0x20015068, size: 4, signed: false, idx: -1},
		{addr: 0x2001506c, size: 4, signed: false, idx: -1},
		{addr: 0x20015070, size: 4, signed: false, idx: -1},
		{addr: 0x20015074, size: 4, signed: false, idx: -1},
		{addr: 0x20015078, size: 4, signed: false, idx: -1},
		{addr: 0x2001507c, size: 4, signed: false, idx: -1},
		{addr: 0x20015080, size: 4, signed: false, idx: -1},
		{addr: 0x20015084, size: 4, signed: false, idx: -1},
		{addr: 0x20015088, size: 4, signed: false, idx: -1},
		{addr: 0x2001508c, size: 4, signed: false, idx: -1},
		{addr: 0x20015090, size: 4, signed: false, idx: -1},
		{addr: 0x20015094, size: 4, signed: false, idx: -1},
		{addr: 0x20015098, size: 4, signed: false, idx: -1},
		{addr: 0x2001509c, size: 4, signed: false, idx: -1},
		{addr: 0x200150a0, size: 4, signed: false, idx: -1},
		{addr: 0x200150a4, size: 4, signed: false, idx: -1},
		{addr: 0x200150a8, size: 4, signed: false, idx: -1},
		{addr: 0x200150ac, size: 4, signed: false, idx: -1},
		{addr: 0x200150b0, size: 4, signed: false, idx: -1},
		{addr: 0x200150b4, size: 4, signed: false, idx: -1},
		{addr: 0x200150b8, size: 4, signed: false, idx: -1},
		{addr: 0x200150bc, size: 4, signed: false, idx: -1},
		{addr: 0x200150c0, size: 4, signed: false, idx: -1},
		{addr: 0x200150c4, size: 4, signed: false, idx: -1},
		{addr: 0x200150c8, size: 4, signed: false, idx: -1},
		{addr: 0x200150cc, size: 4, signed: false, idx: -1},
		{addr: 0x200150d0, size: 4, signed: false, idx: -1},
		{addr: 0x200150d4, size: 4, signed: false, idx: -1},
		{addr: 0x200150d8, size: 4, signed: false, idx: -1},
		{addr: 0x200150dc, size: 4, signed: false, idx: -1},
		{addr: 0x200150e0, size: 4, signed: false, idx: -1},
		{addr: 0x200150e4, size: 4, signed: false, idx: -1},
		{addr: 0x200150e8, size: 4, signed: false, idx: -1},
		{addr: 0x200150ec, size: 4, signed: false, idx: -1},
		{addr: 0x200150f0, size: 4, signed: false, idx: -1},
		{addr: 0x200150f4, size: 4, signed: false, idx: -1},
		{addr: 0x200150f8, size: 4, signed: false, idx: -1},
		{addr: 0x200150fc, size: 4, signed: false, idx: -1},
		{addr: 0x20015100, size: 4, signed: false, idx: -1},
		{addr: 0x20015104, size: 4, signed: false, idx: -1},
		{addr: 0x20015108, size: 4, signed: false, idx: -1},
		{addr: 0x2001510c, size: 4, signed: false, idx: -1},
		{addr: 0x20015110, size: 4, signed: false, idx: -1},
		{addr: 0x20015114, size: 4, signed: false, idx: -1},
		{addr: 0x20015118, size: 4, signed: false, idx: -1},
		{addr: 0x2001511c, size: 4, signed: false, idx: -1},
		{addr: 0x20015120, size: 4, signed: false, idx: -1},
		{addr: 0x20015124, size: 4, signed: false, idx: -1},
		{addr: 0x20015128, size: 4, signed: false, idx: -1},
		{addr: 0x2001512c, size: 4, signed: false, idx: -1},
		{addr: 0x20015130, size: 4, signed: false, idx: -1},
		{addr: 0x20015134, size: 4, signed: false, idx: -1},
		{addr: 0x20015138, size: 4, signed: false, idx: -1},
		{addr: 0x2001513c, size: 4, signed: false, idx: -1},
		{addr: 0x20015140, size: 4, signed: false, idx: -1},
		{addr: 0x20015150, size: 4, signed: false, idx: -1},
		{addr: 0x20015154, size: 4, signed: false, idx: -1},
		{addr: 0x20015158, size: 4, signed: false, idx: -1},
		{addr: 0x2001515c, size: 4, signed: false, idx: -1},
		{addr: 0x20015160, size: 4, signed: false, idx: -1},
		{addr: 0x20015164, size: 4, signed: false, idx: -1},
		{addr: 0x20015168, size: 4, signed: false, idx: -1},
		{addr: 0x2001516c, size: 4, signed: false, idx: -1},
		{addr: 0x20015170, size: 4, signed: false, idx: -1},
		{addr: 0x20015174, size: 4, signed: false, idx: -1},
		{addr: 0x20015178, size: 4, signed: false, idx: -1},
		{addr: 0x2001517c, size: 4, signed: false, idx: -1},
		{addr: 0x20015180, size: 4, signed: false, idx: -1},
		{addr: 0x20015190, size: 4, signed: false, idx: -1},
		{addr: 0x20015194, size: 4, signed: false, idx: -1},
		{addr: 0x200151c0, size: 4, signed: false, idx: -1},
		{addr: 0x20015200, size: 4, signed: false, idx: -1},
	},
}

func nmiNamesIn() []string {
	return []string{"G", "F94", "mode", "batPtr", "batF4", "camPtr", "camSub", "clipLo", "clipHi", "cx", "cy", "cz", "alIdx", "tf0", "tf1", "tf2", "tf3", "up0", "up1", "g0.cnt", "g0.sp", "g0.act", "g0.f29", "g0.f2c", "g0.f2d", "g0.mc", "g0.m0", "g0.m1", "g1.cnt", "g1.sp", "g1.act", "g1.f29", "g1.f2c", "g1.f2d", "g1.mc", "g1.m0", "g1.m1", "c0.slot", "c0.b11", "c0.b01", "c0.a31", "c0.fd", "c0.c4c1", "c0.c4bb", "c0.K0", "c0.K1", "c0.K2", "c1.slot", "c1.b11", "c1.b01", "c1.a31", "c1.fd", "c1.c4c1", "c1.c4bb", "c1.K0", "c1.K1", "c1.K2", "c2.slot", "c2.b11", "c2.b01", "c2.a31", "c2.fd", "c2.c4c1", "c2.c4bb", "c2.K0", "c2.K1", "c2.K2", "c3.slot", "c3.b11", "c3.b01", "c3.a31", "c3.fd", "c3.c4c1", "c3.c4bb", "c3.K0", "c3.K1", "c3.K2", "c4.slot", "c4.b11", "c4.b01", "c4.a31", "c4.fd", "c4.c4c1", "c4.c4bb", "c4.K0", "c4.K1", "c4.K2", "c5.slot", "c5.b11", "c5.b01", "c5.a31", "c5.fd", "c5.c4c1", "c5.c4bb", "c5.K0", "c5.K1", "c5.K2", "c6.slot", "c6.b11", "c6.b01", "c6.a31", "c6.fd", "c6.c4c1", "c6.c4bb", "c6.K0", "c6.K1", "c6.K2", "c7.slot", "c7.b11", "c7.b01", "c7.a31", "c7.fd", "c7.c4c1", "c7.c4bb", "c7.K0", "c7.K1", "c7.K2", "t0.lvl", "t0.f248", "t0.t187", "t0.nm0", "t0.nm1", "t0.nm2", "t0.nm3", "t1.lvl", "t1.f248", "t1.t187", "t1.nm0", "t1.nm1", "t1.nm2", "t1.nm3", "t2.lvl", "t2.f248", "t2.t187", "t2.nm0", "t2.nm1", "t2.nm2", "t2.nm3", "m0.tid", "m0.f28d", "m0.f2ab", "m1.tid", "m1.f28d", "m1.f2ab", "m2.tid", "m2.f28d", "m2.f2ab", "o0.f308", "o0.px", "o0.py", "o0.pz", "o0.f217", "o0.f215", "o0.f1e4", "o0.f1dc", "o0.f330", "o1.f308", "o1.px", "o1.py", "o1.pz", "o1.f217", "o1.f215", "o1.f1e4", "o1.f1dc", "o1.f330", "o2.f308", "o2.px", "o2.py", "o2.pz", "o2.f217", "o2.f215", "o2.f1e4", "o2.f1dc", "o2.f330", "x.slot", "x.charAddr", "x.s", "x.n", "x.quiet", "x.class", "x.race", "x.idx75", "x.state", "x.g0", "x.g1", "x.g2", "x.g3", "x.g4", "x.g5", "x.edd0", "x.f506e", "x.f506d", "x.qcount", "x.qcap", "x.evSrc", "x.raw0", "x.raw1", "x.raw2", "x.raw3", "x.raw4", "x.raw5", "x.raw6", "x.raw7", "x.raw8", "x.raw9", "x.raw10", "x.raw11", "x.raw12", "x.raw13", "x.raw14", "x.raw15", "x.raw16", "x.raw17", "x.raw18", "x.raw19", "x.raw20", "x.raw21", "x.raw22", "x.raw23", "x.raw24", "x.raw25", "x.raw26", "x.raw27", "x.raw28", "x.raw29", "x.raw30", "x.raw31", "x.raw32", "x.raw33", "x.raw34", "x.raw35", "x.raw36", "x.raw37", "x.raw38", "x.raw39", "x.raw40", "x.apt0", "x.apt1", "x.apt2", "x.apt3", "x.apt4", "x.apt5", "x.apt6", "x.apt7", "x.apt8", "x.apt9", "x.apt10", "x.apt11", "x.apt12", "x.apt13", "x.apt14", "x.apt15", "x.apt16", "x.apt17", "x.apt18", "x.apt19", "x.apt20", "x.apt21", "x.apt22", "x.apt23", "x.apt24", "x.apt25", "x.apt26", "x.apt27", "x.apt28", "x.apt29", "x.apt30", "x.apt31", "x.apt32", "x.apt33", "x.apt34", "x.apt35", "x.apt36", "x.apt37", "x.apt38", "x.apt39", "x.apt40", "x.cnt0", "x.cnt1", "x.cnt2", "x.cnt3", "x.cnt4", "x.cnt5", "x.cnt6", "x.cnt7", "x.cnt8", "x.cnt9", "x.cnt10", "x.cnt11", "x.cnt12", "x.cnt13", "x.cnt14", "x.cnt15", "x.cnt16", "x.cnt17", "x.cnt18", "x.cnt19", "x.cnt20", "x.cnt21", "x.cnt22", "x.cnt23", "x.cnt24", "x.cnt25", "x.cnt26", "x.cnt27", "x.cnt28", "x.cnt29", "x.cnt30", "x.cnt31", "x.cnt32", "x.cnt33", "x.cnt34", "x.cnt35", "x.cnt36", "x.cnt37", "x.cnt38", "x.cnt39", "x.cnt40", "x.f19d0", "x.f19d1", "x.f19d2", "x.f19d3", "x.f19d4", "x.f19d5", "x.f19d6", "x.f19d7", "x.f19d8", "x.f19d9", "x.f19d10", "x.f19d11", "x.f19d12", "x.f19d13", "x.f19d14", "x.f19d15", "x.f19d16", "x.f19d17", "x.f19d18", "x.f19d19", "x.f19d20", "x.f19d21", "x.f19d22", "x.f19d23", "x.f19d24", "x.f19d25", "x.f19d26", "x.f19d27", "x.f19d28", "x.f19d29", "x.f19d30", "x.f19d31", "x.f19d32", "x.f19d33", "x.f19d34", "x.f19d35", "x.f19d36", "x.f19d37", "x.f19d38", "x.f19d39", "x.f19d40", "x.f1af0", "x.f1af1", "x.f1af2", "x.f1af3", "x.f1af4", "x.f1af5", "x.f1af6", "x.f1af7", "x.f1af8", "x.f1af9", "x.f1af10", "x.f1af11", "x.f1af12", "x.f1af13", "x.f1af14", "x.f1af15", "x.f1af16", "x.f1af17", "x.f1af18", "x.f1af19", "x.f1af20", "x.f1af21", "x.f1af22", "x.f1af23", "x.f1af24", "x.f1af25", "x.f1af26", "x.f1af27", "x.f1af28", "x.f1af29", "x.f1af30", "x.f1af31", "x.f1af32", "x.f1af33", "x.f1af34", "x.f1af35", "x.f1af36", "x.f1af37", "x.f1af38", "x.f1af39", "x.f1af40", "x.f1b00", "x.f1b01", "x.f1b02", "x.f1b03", "x.f1b04", "x.f1b05", "x.f1b06", "x.f1b07", "x.f1b08", "x.f1b09", "x.f1b010", "x.f1b011", "x.f1b012", "x.f1b013", "x.f1b014", "x.f1b015", "x.f1b016", "x.f1b017", "x.f1b018", "x.f1b019", "x.f1b020", "x.f1b021", "x.f1b022", "x.f1b023", "x.f1b024", "x.f1b025", "x.f1b026", "x.f1b027", "x.f1b028", "x.f1b029", "x.f1b030", "x.f1b031", "x.f1b032", "x.f1b033", "x.f1b034", "x.f1b035", "x.f1b036", "x.f1b037", "x.f1b038", "x.f1b039", "x.f1b040", "x.eff0", "x.eff1", "x.eff2", "x.eff3", "x.eff4", "x.eff5", "x.eff6", "x.eff7", "x.eff8", "x.eff9", "x.eff10", "x.eff11", "x.eff12", "x.eff13", "x.eff14", "x.eff15", "x.eff16", "x.eff17", "x.eff18", "x.eff19", "x.eff20", "x.eff21", "x.eff22", "x.eff23", "x.eff24", "x.eff25", "x.eff26", "x.eff27", "x.eff28", "x.eff29", "x.eff30", "x.eff31", "x.eff32", "x.eff33", "x.eff34", "x.eff35", "x.eff36", "x.eff37", "x.eff38", "x.eff39", "x.eff40", "x.bonus0", "x.bonus1", "x.bonus2", "x.bonus3", "x.bonus4", "x.bonus5", "x.bonus6", "x.bonus7", "x.bonus8", "x.bonus9", "x.bonus10", "x.bonus11", "x.bonus12", "x.bonus13", "x.bonus14", "x.bonus15", "x.bonus16", "x.bonus17", "x.bonus18", "x.bonus19", "x.bonus20", "x.bonus21", "x.bonus22", "x.bonus23", "x.bonus24", "x.bonus25", "x.bonus26", "x.bonus27", "x.bonus28", "x.bonus29", "x.bonus30", "x.bonus31", "x.bonus32", "x.bonus33", "x.bonus34", "x.bonus35", "x.bonus36", "x.bonus37", "x.bonus38", "x.bonus39", "x.bonus40", "x.ded0", "x.ded1", "x.ded2", "x.ded3", "x.ded4", "x.ded5", "x.ded6", "x.ded7", "x.ded8", "x.ded9", "x.ded10", "x.ded11", "x.ded12", "x.ded13", "x.ded14", "x.ded15", "x.ded16", "x.ded17", "x.ded18", "x.ded19", "x.ded20", "x.ded21", "x.ded22", "x.ded23", "x.ded24", "x.ded25", "x.ded26", "x.ded27", "x.ded28", "x.ded29", "x.ded30", "x.ded31", "x.ded32", "x.ded33", "x.ded34", "x.ded35", "x.ded36", "x.ded37", "x.ded38", "x.ded39", "x.ded40", "x.b8d0", "x.b8d1", "x.b8d2", "x.b8d3", "x.b8d4", "x.b8d5", "x.b8d6", "x.b8d7", "x.b8d8", "x.b8d9", "x.b8d10", "x.b8d11", "x.b8d12", "x.b8d13", "x.b8d14", "x.att0", "x.att1", "x.att2", "x.att3", "x.att4", "x.att5", "x.att6", "x.spv0", "x.spv1", "x.spv2", "x.spv3", "x.spv4", "x.spv5", "x.spv6", "x.spv7", "x.spv8", "x.spv9", "x.spv10", "x.spv11", "x.spv12", "x.spv13", "x.spv14", "x.spv15", "x.spv16", "x.spv17", "x.spv18", "x.spv19", "x.spv20", "x.spv21", "x.spv22", "x.spv23", "x.spv24", "x.spv25", "x.spv26", "x.spv27", "x.spv28", "x.spv29", "x.spv30", "x.spv31", "x.spv32", "x.spv33", "x.spv34", "x.spv35", "x.spv36", "x.spv37", "x.spv38", "x.spv39", "x.spv40", "x.spv41", "x.spv42", "x.spv43", "x.spv44", "x.spv45", "x.spv46", "x.spv47", "x.spv48", "x.spv49", "x.spv50", "x.spv51", "x.spv52", "x.spv53", "x.spv54", "x.spv55", "x.spv56", "x.spv57", "x.spv58", "x.spv59", "x.spv60", "x.spv61", "x.spv62", "x.spv63", "x.spv64", "x.spv65", "x.spv66", "x.spv67", "x.spv68", "x.spv69", "x.spv70", "x.spv71", "x.spv72", "x.spv73", "x.spv74", "x.spv75", "x.spv76", "x.spv77", "x.spv78", "x.spv79", "x.spv80", "x.spv81", "x.spv82", "x.spv83", "x.spv84", "x.spv85", "x.spv86", "x.spv87", "x.spv88", "x.spv89", "x.spv90", "x.spv91", "x.spv92", "x.spv93", "x.spv94", "x.spv95", "x.spv96", "x.spv97", "x.spv98", "x.spv99", "x.spv100", "x.spv101", "x.spv102", "x.spv103", "x.spv104", "x.spv105", "x.spv106", "x.spv107", "x.spv108", "x.spv109", "x.spv110", "x.spv111", "x.spv112", "x.spsch0", "x.spsch1", "x.spsch2", "x.spsch3", "x.spsch4", "x.spsch5", "x.spsch6", "x.spsch7", "x.spsch8", "x.spsch9", "x.spsch10", "x.spsch11", "x.spsch12", "x.spsch13", "x.spsch14", "x.spsch15", "x.spsch16", "x.spsch17", "x.spsch18", "x.spsch19", "x.spsch20", "x.spsch21", "x.spsch22", "x.spsch23", "x.spsch24", "x.spsch25", "x.spsch26", "x.spsch27", "x.spsch28", "x.spsch29", "x.spsch30", "x.spsch31", "x.spsch32", "x.spsch33", "x.spsch34", "x.spsch35", "x.spsch36", "x.spsch37", "x.spsch38", "x.spsch39", "x.spsch40", "x.spsch41", "x.spsch42", "x.spsch43", "x.spsch44", "x.spsch45", "x.spsch46", "x.spsch47", "x.spsch48", "x.spsch49", "x.spsch50", "x.spsch51", "x.spsch52", "x.spsch53", "x.spsch54", "x.spsch55", "x.spsch56", "x.spsch57", "x.spsch58", "x.spsch59", "x.spsch60", "x.spsch61", "x.spsch62", "x.spsch63", "x.spsch64", "x.spsch65", "x.spsch66", "x.spsch67", "x.spsch68", "x.spsch69", "x.spsch70", "x.spsch71", "x.spsch72", "x.spsch73", "x.spsch74", "x.spsch75", "x.spsch76", "x.spsch77", "x.spsch78", "x.spsch79", "x.spsch80", "x.spsch81", "x.spsch82", "x.spsch83", "x.spsch84", "x.spsch85", "x.spsch86", "x.spsch87", "x.spsch88", "x.spsch89", "x.spsch90", "x.spsch91", "x.spsch92", "x.spsch93", "x.spsch94", "x.spsch95", "x.spsch96", "x.spsch97", "x.spsch98", "x.spsch99", "x.spsch100", "x.spsch101", "x.spsch102", "x.spsch103", "x.spsch104", "x.spsch105", "x.spsch106", "x.spsch107", "x.spsch108", "x.spsch109", "x.spsch110", "x.spsch111", "x.spsch112", "x.kclass", "x.k8d", "x.kraw0", "x.kraw1", "x.kraw2", "x.kraw3", "x.kraw4", "x.kraw5", "x.kraw6", "x.kraw7", "x.kraw8", "x.kraw9", "x.kraw10", "x.kraw11", "x.kraw12", "x.kraw13", "x.kraw14", "x.kraw15", "x.kraw16", "x.kraw17", "x.kraw18", "x.kraw19", "x.kraw20", "x.kraw21", "x.kraw22", "x.kraw23", "x.kraw24", "x.kraw25", "x.kraw26", "x.kraw27", "x.kraw28", "x.kraw29", "x.kraw30", "x.kraw31", "x.kraw32", "x.kraw33", "x.kraw34", "x.kraw35", "x.kraw36", "x.kraw37", "x.kraw38", "x.kraw39", "x.kraw40", "x.keff0", "x.keff1", "x.keff2", "x.keff3", "x.keff4", "x.keff5", "x.keff6", "x.keff7", "x.keff8", "x.keff9", "x.keff10", "x.keff11", "x.keff12", "x.keff13", "x.keff14", "x.keff15", "x.keff16", "x.keff17", "x.keff18", "x.keff19", "x.keff20", "x.keff21", "x.keff22", "x.keff23", "x.keff24", "x.keff25", "x.keff26", "x.keff27", "x.keff28", "x.keff29", "x.keff30", "x.keff31", "x.keff32", "x.keff33", "x.keff34", "x.keff35", "x.keff36", "x.keff37", "x.keff38", "x.keff39", "x.keff40", "x.k19e0", "x.k19e1", "x.k19e2", "x.k19e3", "x.k19e4", "x.k19e5", "x.k19e6", "x.k19e7", "x.k19e8", "x.k19e9", "x.k19e10", "x.k19e11", "x.k19e12", "x.k19e13", "x.k19e14", "x.k19e15", "x.k19e16", "x.k19e17", "x.k19e18", "x.k19e19", "x.k19e20", "x.k19e21", "x.k19e22", "x.k19e23", "x.k19e24", "x.k19e25", "x.k19e26", "x.k19e27", "x.k19e28", "x.k19e29", "x.k19e30", "x.k19e31", "x.k19e32", "x.k19e33", "x.k19e34", "x.k19e35", "x.k19e36", "x.k19e37", "x.k19e38", "x.k19e39", "x.k19e40", "x.uval0", "x.uval1", "x.uval2", "x.uval3", "x.uval4", "x.uval5", "x.uval6", "x.uval7", "x.uval8", "x.uval9", "x.uval10", "x.uval11", "x.uval12", "x.uval13", "x.uval14", "x.uval15", "x.uval16", "x.uval17", "x.uval18", "x.uval19", "x.uval20", "x.uval21", "x.uval22", "x.uval23", "x.uval24", "x.uval25", "x.uval26", "x.uval27", "x.uval28", "x.uval29", "x.uval30", "x.uval31", "x.uval32", "x.uval33", "x.uval34", "x.uval35", "x.uval36", "x.uval37", "x.uval38", "x.uval39", "x.uval40", "x.u1bc0", "x.u1bc1", "x.u1bc2", "x.u1bc3", "x.u1bc4", "x.u1bc5", "x.u1bc6", "x.u1bc7", "x.u1bc8", "x.u1bc9", "x.u1bc10", "x.u1bc11", "x.u1bc12", "x.u1bc13", "x.u1bc14", "x.u1bc15", "x.u1bc16", "x.u1bc17", "x.u1bc18", "x.u1bc19", "x.u1bc20", "x.u1bc21", "x.u1bc22", "x.u1bc23", "x.u1bc24", "x.u1bc25", "x.u1bc26", "x.u1bc27", "x.u1bc28", "x.u1bc29", "x.u1bc30", "x.u1bc31", "x.u1bc32", "x.u1bc33", "x.u1bc34", "x.u1bc35", "x.u1bc36", "x.u1bc37", "x.u1bc38", "x.u1bc39", "x.u1bc40", "x.u64", "x.grid0_0", "x.grid0_1", "x.grid0_2", "x.grid0_3", "x.grid0_4", "x.grid0_5", "x.grid0_6", "x.grid0_7", "x.grid0_8", "x.grid0_9", "x.grid0_10", "x.grid0_11", "x.grid0_12", "x.grid0_13", "x.grid0_14", "x.grid0_15", "x.grid0_16", "x.grid0_17", "x.grid0_18", "x.grid0_19", "x.grid0_20", "x.grid0_21", "x.grid0_22", "x.grid0_23", "x.grid0_24", "x.grid0_25", "x.grid0_26", "x.grid0_27", "x.grid0_28", "x.grid0_29", "x.grid0_30", "x.grid0_31", "x.grid0_32", "x.grid0_33", "x.grid0_34", "x.grid0_35", "x.grid0_36", "x.grid0_37", "x.grid0_38", "x.grid0_39", "x.grid0_40", "x.grid1_0", "x.grid1_1", "x.grid1_2", "x.grid1_3", "x.grid1_4", "x.grid1_5", "x.grid1_6", "x.grid1_7", "x.grid1_8", "x.grid1_9", "x.grid1_10", "x.grid1_11", "x.grid1_12", "x.grid1_13", "x.grid1_14", "x.grid1_15", "x.grid1_16", "x.grid1_17", "x.grid1_18", "x.grid1_19", "x.grid1_20", "x.grid1_21", "x.grid1_22", "x.grid1_23", "x.grid1_24", "x.grid1_25", "x.grid1_26", "x.grid1_27", "x.grid1_28", "x.grid1_29", "x.grid1_30", "x.grid1_31", "x.grid1_32", "x.grid1_33", "x.grid1_34", "x.grid1_35", "x.grid1_36", "x.grid1_37", "x.grid1_38", "x.grid1_39", "x.grid1_40", "x.grid2_0", "x.grid2_1", "x.grid2_2", "x.grid2_3", "x.grid2_4", "x.grid2_5", "x.grid2_6", "x.grid2_7", "x.grid2_8", "x.grid2_9", "x.grid2_10", "x.grid2_11", "x.grid2_12", "x.grid2_13", "x.grid2_14", "x.grid2_15", "x.grid2_16", "x.grid2_17", "x.grid2_18", "x.grid2_19", "x.grid2_20", "x.grid2_21", "x.grid2_22", "x.grid2_23", "x.grid2_24", "x.grid2_25", "x.grid2_26", "x.grid2_27", "x.grid2_28", "x.grid2_29", "x.grid2_30", "x.grid2_31", "x.grid2_32", "x.grid2_33", "x.grid2_34", "x.grid2_35", "x.grid2_36", "x.grid2_37", "x.grid2_38", "x.grid2_39", "x.grid2_40", "x.grid3_0", "x.grid3_1", "x.grid3_2", "x.grid3_3", "x.grid3_4", "x.grid3_5", "x.grid3_6", "x.grid3_7", "x.grid3_8", "x.grid3_9", "x.grid3_10", "x.grid3_11", "x.grid3_12", "x.grid3_13", "x.grid3_14", "x.grid3_15", "x.grid3_16", "x.grid3_17", "x.grid3_18", "x.grid3_19", "x.grid3_20", "x.grid3_21", "x.grid3_22", "x.grid3_23", "x.grid3_24", "x.grid3_25", "x.grid3_26", "x.grid3_27", "x.grid3_28", "x.grid3_29", "x.grid3_30", "x.grid3_31", "x.grid3_32", "x.grid3_33", "x.grid3_34", "x.grid3_35", "x.grid3_36", "x.grid3_37", "x.grid3_38", "x.grid3_39", "x.grid3_40", "x.grid4_0", "x.grid4_1", "x.grid4_2", "x.grid4_3", "x.grid4_4", "x.grid4_5", "x.grid4_6", "x.grid4_7", "x.grid4_8", "x.grid4_9", "x.grid4_10", "x.grid4_11", "x.grid4_12", "x.grid4_13", "x.grid4_14", "x.grid4_15", "x.grid4_16", "x.grid4_17", "x.grid4_18", "x.grid4_19", "x.grid4_20", "x.grid4_21", "x.grid4_22", "x.grid4_23", "x.grid4_24", "x.grid4_25", "x.grid4_26", "x.grid4_27", "x.grid4_28", "x.grid4_29", "x.grid4_30", "x.grid4_31", "x.grid4_32", "x.grid4_33", "x.grid4_34", "x.grid4_35", "x.grid4_36", "x.grid4_37", "x.grid4_38", "x.grid4_39", "x.grid4_40", "x.grid5_0", "x.grid5_1", "x.grid5_2", "x.grid5_3", "x.grid5_4", "x.grid5_5", "x.grid5_6", "x.grid5_7", "x.grid5_8", "x.grid5_9", "x.grid5_10", "x.grid5_11", "x.grid5_12", "x.grid5_13", "x.grid5_14", "x.grid5_15", "x.grid5_16", "x.grid5_17", "x.grid5_18", "x.grid5_19", "x.grid5_20", "x.grid5_21", "x.grid5_22", "x.grid5_23", "x.grid5_24", "x.grid5_25", "x.grid5_26", "x.grid5_27", "x.grid5_28", "x.grid5_29", "x.grid5_30", "x.grid5_31", "x.grid5_32", "x.grid5_33", "x.grid5_34", "x.grid5_35", "x.grid5_36", "x.grid5_37", "x.grid5_38", "x.grid5_39", "x.grid5_40", "x.grid6_0", "x.grid6_1", "x.grid6_2", "x.grid6_3", "x.grid6_4", "x.grid6_5", "x.grid6_6", "x.grid6_7", "x.grid6_8", "x.grid6_9", "x.grid6_10", "x.grid6_11", "x.grid6_12", "x.grid6_13", "x.grid6_14", "x.grid6_15", "x.grid6_16", "x.grid6_17", "x.grid6_18", "x.grid6_19", "x.grid6_20", "x.grid6_21", "x.grid6_22", "x.grid6_23", "x.grid6_24", "x.grid6_25", "x.grid6_26", "x.grid6_27", "x.grid6_28", "x.grid6_29", "x.grid6_30", "x.grid6_31", "x.grid6_32", "x.grid6_33", "x.grid6_34", "x.grid6_35", "x.grid6_36", "x.grid6_37", "x.grid6_38", "x.grid6_39", "x.grid6_40", "x.grid7_0", "x.grid7_1", "x.grid7_2", "x.grid7_3", "x.grid7_4", "x.grid7_5", "x.grid7_6", "x.grid7_7", "x.grid7_8", "x.grid7_9", "x.grid7_10", "x.grid7_11", "x.grid7_12", "x.grid7_13", "x.grid7_14", "x.grid7_15", "x.grid7_16", "x.grid7_17", "x.grid7_18", "x.grid7_19", "x.grid7_20", "x.grid7_21", "x.grid7_22", "x.grid7_23", "x.grid7_24", "x.grid7_25", "x.grid7_26", "x.grid7_27", "x.grid7_28", "x.grid7_29", "x.grid7_30", "x.grid7_31", "x.grid7_32", "x.grid7_33", "x.grid7_34", "x.grid7_35", "x.grid7_36", "x.grid7_37", "x.grid7_38", "x.grid7_39", "x.grid7_40", "x.wf4", "x.wf8"}
}

func nmiNamesOut() []string {
	return []string{"g0.f2c", "g0.f2d", "g1.f2c", "g1.f2d", "c0.K0", "c0.K1", "c0.K2", "c1.K0", "c1.K1", "c1.K2", "c2.K0", "c2.K1", "c2.K2", "c3.K0", "c3.K1", "c3.K2", "c4.K0", "c4.K1", "c4.K2", "c5.K0", "c5.K1", "c5.K2", "c6.K0", "c6.K1", "c6.K2", "c7.K0", "c7.K1", "c7.K2", "batF4", "n553", "s553.a1.0", "s553.a1.1", "s553.a1.2", "s553.a1.3", "s553.a1.4", "s553.a1.5", "s553.a1.6", "s553.a1.7", "s553.a1.8", "s553.a1.9", "s553.a1.10", "s553.a1.11", "s553.a1.12", "s553.a1.13", "s553.a1.14", "s553.a1.15", "s553.a2.0", "s553.a2.1", "s553.a2.2", "s553.a2.3", "s553.a2.4", "s553.a2.5", "s553.a2.6", "s553.a2.7", "s553.a2.8", "s553.a2.9", "s553.a2.10", "s553.a2.11", "s553.a2.12", "s553.a2.13", "s553.a2.14", "s553.a2.15", "s553.a3.0", "s553.a3.1", "s553.a3.2", "s553.a3.3", "s553.a3.4", "s553.a3.5", "s553.a3.6", "s553.a3.7", "s553.a3.8", "s553.a3.9", "s553.a3.10", "s553.a3.11", "s553.a3.12", "s553.a3.13", "s553.a3.14", "s553.a3.15", "s553.a4.0", "s553.a4.1", "s553.a4.2", "s553.a4.3", "s553.a4.4", "s553.a4.5", "s553.a4.6", "s553.a4.7", "s553.a4.8", "s553.a4.9", "s553.a4.10", "s553.a4.11", "s553.a4.12", "s553.a4.13", "s553.a4.14", "s553.a4.15", "n58", "s58.a1.0", "s58.a1.1", "s58.a1.2", "s58.a1.3", "s58.a2.0", "s58.a2.1", "s58.a2.2", "s58.a2.3", "s58.a3.0", "s58.a3.1", "s58.a3.2", "s58.a3.3", "nup", "up.a1.0", "up.a1.1", "ntf", "ncl"}
}
