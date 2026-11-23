package calc

import (
	"math"
	"math/big"
)

// 0x503990(m, ticks, fullFlag): one monster turn of upkeep. Positional
// bookkeeping (homing / return-home checks, the previous-position record),
// the two action timers, damage over time, HP/SP regeneration and the
// fractional recovery accumulators, condition decay, buff expiry, and the
// monster's own and team spell-slot countdowns.
//
// Address stubs (D4, ret from the case's ignoredReturn): 0x524110 condition
// decay, 0x52beb0 hit response, 0x523c00 condition apply, 0x5526f0 team spell
// tick, 0x4e4280 / 0x4e46f0 death bookkeeping, 0x5049c0 awareness. F runs
// their nodes / models instead (Emu replays the case stubs). Engine-side
// queries and presentation are 排除 stubs (0x462e60, 0x4be940, 0x434b60
// return the case's inputs; 0x510cc0, 0x4acd80, 0x590b40, 0x58aad0, 0x48f650,
// 0x4e4db0, 0x4e3af0, 0x4c5a00 record arguments). x87 arithmetic runs at
// extended precision (unicorn's default control word): big.Float, prec 64.
func init() {
	Nodes["00503990"] = Node{Addr: "00503990", In: mtrIn, Out: mtrOut, F: monsterTurnRecoveryAndStatusNode, Emu: mtrEmu}
}

func monsterTurnRecoveryAndStatusNode(in []int64, r Rand) []int64 { return mtrRun(true)(in, r) }

func mtrEmu(in []int64, r Rand) []int64 { return mtrRun(false)(in, r) }

const (
	mtrZero32   = 0x5ebb34   // f32 0.0: target-unset test and the factor floor
	mtrHomeDist = 0x5ec150   // f64 500.0: homing / return distance
	mtrQuarter  = 0x5ebcc8   // f32 0.25: recovery factor while +0x5f (condition 2) is set
	mtrTrigger  = 0x5ec254   // f32 12.0: [0x60a390] must equal it for the return check
	mtrCondText = 0x61e570   // 0x523f40: u16 text id per condition, stride 8
	mtrSlotVis  = 0x60cffc   // 0x5524e0: visual id per spell, stride 8
	mtrSpells   = 0x60       // fixture cap: spell ids 0..0x5f (0x5524e0's table read)
	mtrInnerCnt = 0x2000d000 // F-only inner stub counters (outside the case's objects)
)

// mtrX is an x87 register value: big.Float at 64-bit precision, or NaN.
type mtrX struct {
	v   *big.Float
	nan bool
	nb  uint32 // float32 bits stored for a NaN (quiet, payload kept)
}

func mtrNew() *big.Float { return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven) }

func mtrF32(bits uint32) mtrX {
	f := math.Float32frombits(bits)
	if f != f {
		return mtrX{nan: true, nb: bits | 0x400000}
	}
	return mtrX{v: mtrNew().SetFloat64(float64(f))}
}

func mtrF64(v float64) mtrX { return mtrX{v: mtrNew().SetFloat64(v)} }

func mtrOp(a, b mtrX, op func(z, x, y *big.Float) *big.Float) (z mtrX) {
	if a.nan {
		return a
	}
	if b.nan {
		return b
	}
	defer func() {
		if e := recover(); e != nil {
			if _, ok := e.(big.ErrNaN); !ok {
				panic(e)
			}
			z = mtrX{nan: true, nb: 0xffc00000} // x87 default NaN
		}
	}()
	return mtrX{v: op(mtrNew(), a.v, b.v)}
}

func mtrAdd(a, b mtrX) mtrX { return mtrOp(a, b, (*big.Float).Add) }
func mtrSub(a, b mtrX) mtrX { return mtrOp(a, b, (*big.Float).Sub) }
func mtrMul(a, b mtrX) mtrX { return mtrOp(a, b, (*big.Float).Mul) }

// mtrRound is fstp to float32 (round to nearest even).
func mtrRound(a mtrX) mtrX {
	if a.nan {
		return a
	}
	f, _ := a.v.Float32()
	return mtrF32(math.Float32bits(f))
}

func mtrBits(a mtrX) uint32 {
	if a.nan {
		return a.nb
	}
	f, _ := a.v.Float32()
	return math.Float32bits(f)
}

// mtrCmp is fcom: -1 / 0 / 1, or 2 when unordered.
func mtrCmp(a, b mtrX) int {
	if a.nan || b.nan {
		return 2
	}
	return a.v.Cmp(b.v)
}

func mtrSqrt(a mtrX) mtrX {
	if a.nan {
		return a
	}
	if a.v.Sign() < 0 {
		return mtrX{nan: true, nb: 0xffc00000}
	}
	return mtrX{v: mtrNew().Sqrt(a.v)}
}

// mtrFtol is 0x5e1c22: truncate toward zero to int64; NaN, infinity and
// overflow give the x87 integer indefinite 0x8000000000000000.
func mtrFtol(a mtrX) uint64 {
	if a.nan || a.v.IsInf() {
		return 1 << 63
	}
	i, acc := a.v.Int64()
	if acc != big.Exact && (i == math.MaxInt64 || i == math.MinInt64) {
		return 1 << 63
	}
	return uint64(i)
}

// mtrSrc is 0x503990's 0x536150 source record: its stack local in the case's
// frame (entry esp 0x10008000, local at esp-0x34), as ostSrc in n_5526f0.go.
// The address leaks through ecx into 0x4e46f0's second argument.
const mtrSrc = 0x10007fcc

const (
	mtrNoPlace = iota // return-home check outcomes
	mtrClear
	mtrReset
)

// mtrInnerStubs serve the stubs F's real 0x523c00 reaches: counters only, so
// they stay out of the case's records (Emu replays 0x523c00 as one stub).
var mtrInnerStubs = func() map[uint32]*mcaStub {
	t := map[uint32]*mcaStub{}
	for i, a := range []uint32{0x4acd80, 0x4c2cf0, 0x4c5a00, 0x4c5aa0, 0x4e3af0, 0x4e4280, 0x4e46f0, 0x4e4db0,
		0x531540, 0x538e00, 0x546e70, 0x58aad0, 0x4f0480, 0x5049c0, 0x52c240, 0x5323f0, 0x5182e0} {
		t[a] = &mcaStub{cnt: mtrInnerCnt + 4*uint32(i)}
	}
	return t
}()

func mtrRun(real bool) func([]int64, Rand) []int64 {
	return mcaRun(&mtrSpec, real, func(w *mcaWorld) {
		w.tab = mtrStubs
		mtrBody(w)
	})
}

func mtrF64At(w *mcaWorld, a uint32) mtrX {
	return mtrF64(math.Float64frombits(uint64(w.u32(a)) | uint64(w.u32(a+4))<<32))
}

// mtrPos is 0x4b8890: the visual's position, loaded and stored as float32.
func mtrPos(w *mcaWorld, actor uint32) [3]mtrX {
	v := w.u32(actor + 0x1d8)
	var p [3]mtrX
	for i := range p {
		p[i] = mtrF32(mtrBits(mtrF32(w.u32(v + 4 + 4*uint32(i)))))
	}
	return p
}

// mtrTargetFix: an unset target (all three coordinates compare equal to 0.0)
// takes the actor's home point.
func mtrTargetFix(w *mcaWorld, actor uint32) {
	zero := mtrF32(w.u32(mtrZero32))
	for i := uint32(0); i < 3; i++ {
		if c := mtrCmp(mtrF32(w.u32(actor+0x280+4*i)), zero); c != 0 && c != 2 {
			return
		}
	}
	for i := uint32(0); i < 3; i++ {
		w.put(actor+0x280+4*i, 4, uint64(w.u32(actor+0x118+4*i)))
	}
}

// mtrBlocked is 0x4be940 then 0x434b60 (engine queries, case returns).
func mtrBlocked(w *mcaWorld) bool {
	return byte(w.stub(0x4be940)) != 0 || byte(w.stub(0x434b60)) != 0
}

// mtrPlace moves the monster's group and its partner groups (0x510cc0, 排除).
func mtrPlace(w *mcaWorld, m uint32) {
	h := w.u32(w.u32(m+0xc) + 0xec)
	g := w.group(w.u32(m + 4))
	w.stub(0x510cc0, g, 0, h)
	for k := uint32(0); k < 4; k++ {
		if p := w.u32(g + 0xa7 + 4*k); p != 0 {
			w.stub(0x510cc0, w.group(p), 0, h)
		}
	}
}

func mtrSq(a mtrX) mtrX { return mtrMul(a, a) }

// mtrReturn is the return-home check of the idle branch.
func mtrReturn(w *mcaWorld, m uint32) {
	actor := w.u32(m + 0xc)
	pos := mtrPos(w, actor)
	var d [3]mtrX
	for i := uint32(0); i < 3; i++ {
		w.put(m+0x17+4*i, 4, uint64(mtrBits(pos[i])))
		d[i] = mtrRound(mtrSub(pos[i], mtrF32(w.u32(m+0x338+4*i))))
	}
	res := mtrNoPlace
	switch st := int8(w.u8(m + 0x254)); {
	case st < 2:
	case st >= 4:
		if mtrBlocked(w) {
			break
		}
		mtrTargetFix(w, actor)
		if mtrBlocked(w) {
			break
		}
		mtrTargetFix(w, actor)
		mtrPlace(w, m)
		res = mtrReset
	default:
		s := mtrSqrt(mtrAdd(mtrAdd(mtrSq(d[2]), mtrSq(d[1])), mtrSq(d[0])))
		if c := mtrCmp(s, mtrF64At(w, mtrHomeDist)); c != -1 && c != 2 {
			break
		}
		res = mtrClear
		if byte(w.stub(0x462e60)) != 0 || w.u8(m+0x2ab) != 0 || mtrBlocked(w) {
			break
		}
		mtrPlace(w, m)
		res = mtrReset
	}
	if res == mtrReset {
		w.put(m+0x254, 1, 0)
	}
	if res != mtrNoPlace {
		w.put(m+0x255, 1, 0)
		w.put(m+0x246, 1, 0)
	}
	if st := int8(w.u8(m + 0x254)); st > 0 {
		w.put(m+0x254, 1, uint64(byte(st+1)))
	}
	for i := uint32(0); i < 3; i++ {
		w.put(m+0x338+4*i, 4, uint64(w.u32(m+0x17+4*i)))
	}
}

// mtrTimer counts one action timer down; at zero it rearms the other from the
// actor's span: rand(span) + span/2 (signed), at least 1.
func mtrTimer(w *mcaWorld, m, at, other, span, ticks uint32) {
	v := w.u8(m + at)
	if uint32(v) > ticks {
		w.put(m+at, 1, uint64(v-byte(ticks)))
		return
	}
	w.put(m+at, 1, 0)
	s := int32(w.u32(w.u32(m+0xc) + span))
	n := byte(w.r.R(int(uint32(s)))) + byte(s/2)
	if n == 0 {
		n = 1
	}
	w.put(m+other, 1, uint64(n))
}

func mtrFreshSrc(w *mcaWorld) uint32 {
	for i := uint32(0); i < 13; i++ { // 0x536150
		w.put(mtrSrc+4*i, 4, 0)
	}
	w.put(mtrSrc+4, 4, 0xffffffff)
	w.put(mtrSrc+8, 4, 0xffffffff)
	return mtrSrc
}

func mtrBody(w *mcaWorld) {
	m, ticks, full := mcaArg(w, 0), mcaArg(w, 1), mcaArg(w, 2)
	tmpl := w.template(m)
	actor := w.u32(m + 0xc)
	team := w.u32(m + 0x10)
	member := w.u8(m+0x15) != 0
	switch {
	case member:
		mtrOst(w, m, team+0x3e)
	case w.u8(actor+0x291) != 0:
		pos := mtrPos(w, actor)
		mtrTargetFix(w, actor)
		var d [3]mtrX
		for i := uint32(0); i < 3; i++ {
			d[i] = mtrSub(mtrF32(w.u32(actor+0x280+4*i)), pos[i])
		}
		dist := mtrSqrt(mtrAdd(mtrAdd(mtrSq(d[0]), mtrSq(d[2])), mtrSq(d[1])))
		if mtrCmp(dist, mtrF64At(w, mtrHomeDist)) == 1 && !mtrBlocked(w) {
			mtrPlace(w, m)
		}
	default:
		if c := mtrCmp(mtrF32(w.u32(0x60a390)), mtrF32(w.u32(mtrTrigger))); c != 0 && c != 2 {
			break
		}
		if !(w.u32(actor+0x74) != 0 || w.u8(actor+0x3d) != 0 || (int8(w.u8(m+0x254)) <= 1 && w.u8(actor+0x3c) != 0)) {
			mtrReturn(w, m)
		}
		if w.u8(m+0x302) > 0 {
			mtrTimer(w, m, 0x302, 0x303, 0x2cc, ticks)
		} else if w.u8(m+0x303) > 0 {
			mtrTimer(w, m, 0x303, 0x302, 0x2c8, ticks)
		}
	}
	if dot := uint32(w.u8(m + 0x1e3)); dot > 0 {
		dmg := dot * ticks
		if member {
			dmg += dmg >> 1
		}
		mtrDamage(w, m, dmg, mtrFreshSrc(w), 1, uint32(w.u8(0x683f94)), mtrSrc)
	}
	quarter := w.u32(m+0x5f) > 0
	if r := (int32(int8(w.u8(tmpl+0x17c))) + int32(int8(w.u8(m+0x1e4)))) * int32(ticks); r > 0 {
		if w.u32(m+0x2b) < w.u32(m+0x27) {
			mtrHeal(w, m, uint32(r))
		}
	} else if r < 0 {
		mtrDamage(w, m, uint32(-r), mtrFreshSrc(w), 0, 0, mtrSrc)
	}
	if s := (int32(int8(w.u8(m+0x1e5))) + int32(int8(w.u8(tmpl+0xce)))) * int32(ticks); s > 0 {
		if w.u32(m+0x33) < w.u32(m+0x2f) {
			mtrSp(w, m, uint32(s))
		}
	} else if s < 0 {
		mtrFatigue(w, m, uint32(-s))
	}
	factor := mtrF64(0)
	switch {
	case w.u8(0x683fc5) != 0 || byte(full) != 0:
		factor = mtrF64(1)
	case !member:
		factor = mtrF64(0.5)
	}
	if quarter {
		factor = mtrRound(mtrMul(factor, mtrF32(w.u32(mtrQuarter))))
	}
	if mtrCmp(factor, mtrF32(w.u32(mtrZero32))) == 1 {
		if w.u32(m+0x2b) < w.u32(m+0x27) {
			mtrCarry(w, m, 0x47, 0x4b, ticks, factor, mtrHeal)
		}
		if w.u32(m+0x33) < w.u32(m+0x2f) {
			mtrCarry(w, m, 0x4f, 0x53, ticks, factor, mtrSp)
		}
	}
	id := w.u32(m)
	for i := uint32(0); i < 20; i++ {
		if v := w.u32(m + 0x57 + 4*i); v > 0 && v < 0x270f {
			mtrDecay(w, m, id, i, ticks)
		}
	}
	for i := uint32(0); i < 8; i++ {
		v := w.u32(m + 0xaf + 12*i)
		if v == 0 || v >= 0x270f {
			continue
		}
		if v > ticks {
			w.put(m+0xaf+12*i, 4, uint64(v-ticks))
		} else {
			mtrBuffEnd(w, id, i)
		}
	}
	if w.u32(m+0x2b) >= w.u32(m+0x27) {
		w.put(m+0x4b, 4, 0)
	}
	if w.u32(m+0x33) >= w.u32(m+0x2f) {
		w.put(m+0x53, 4, 0)
	}
	mtrSlots(w, m, m+0x10f, 12, ticks)
	if member {
		mtrSlots(w, m, team+0x3e, 9, ticks)
		mtrSlots(w, m, team+0xd7, 6, ticks)
	}
}

func mtrSlots(w *mcaWorld, m, p, n, ticks uint32) {
	for ; n > 0; n, p = n-1, p+0x11 {
		if w.u8(p) == 0 {
			continue
		}
		if left := w.u32(p + 0xd); left > ticks {
			w.put(p+0xd, 4, uint64(left-ticks))
			continue
		}
		if m == 0 {
			panic("monster turn recovery: slot expiry without a monster (0x5524b0 sites, unreachable)")
		}
		mtrSlotEnd(w, m, p)
	}
}

// mtrCarry accumulates a fractional recovery and applies its whole part.
func mtrCarry(w *mcaWorld, m, rate, carry, ticks uint32, factor mtrX, add func(*mcaWorld, uint32, uint32)) {
	v := mtrAdd(mtrMul(mtrMul(mtrF64(float64(ticks)), mtrF32(w.u32(m+rate))), factor), mtrF32(w.u32(m+carry)))
	w.put(m+carry, 4, uint64(mtrBits(v)))
	add(w, m, uint32(mtrFtol(v)))
	c := mtrF32(w.u32(m + carry))
	n := uint32(mtrFtol(c))
	w.put(m+carry, 4, uint64(mtrBits(mtrSub(c, mtrF64(float64(n))))))
}

// mtrOst is 0x5526f0(team spells, monster target): Emu replays the case stub;
// F runs ongoingSpellTickNode's body on a copy with the decoy spell rules.
func mtrOst(w *mcaWorld, m, slots uint32) {
	tgt := uint32(0x7e000200)
	k := w.call(0x5526f0, slots, tgt)
	if !w.real {
		w.replay(0x5526f0, k, slots, tgt)
		return
	}
	for i := uint32(0); i < 9*0x11; i++ {
		w.put(ostSlots+i, 1, uint64(w.u8(slots+i)))
	}
	w.put(ostTarget, 4, 3)
	w.put(ostTarget+4, 4, 0)
	w.put(ostTarget+8, 4, uint64(w.u32(m)))
	for _, s := range ostDecoys(mstSpec{set: []mstSet{{0x65be1c, 4, ostRules}}}).set {
		w.put(s.addr, int(s.size), uint64(s.val))
	}
	ostBody(w)
	for i := uint32(0); i < 9*0x11; i++ {
		w.put(slots+i, 1, uint64(w.u8(ostSlots+i)))
	}
}

// mtrDamage is 0x52bb60(m, dmg, src, a4, a5, 0, 0, 0). ecx is the caller's
// register on entry: 0x4e4690's byte argument is pushed as all of ecx, so its
// upper bytes are whatever the damage path last left there.
func mtrDamage(w *mcaWorld, m, dmg, src, a4, a5, ecx uint32) {
	if w.u32(m+0x2b) == 0 {
		return
	}
	tbl := w.u32(0x68c09c)
	if w.u32(m+0xc7) > 0 { // condition-6 shield pool at +0xc5
		str := w.u32(tbl + 4*0x193)
		if byte(a4) != 0 {
			str = w.u32(tbl + 4*0x192)
		}
		pool := w.u16(m + 0xc5)
		if pool >= dmg {
			w.stub(0x590b40, m, str, dmg)
			w.put(m+0xc5, 2, uint64(pool-dmg))
			return
		}
		w.stub(0x590b40, m, str, pool)
		dmg -= pool
		mtrBuffEnd(w, w.u32(m), 2)
		ecx = 0 // 0x524390 ends in 0x50e8c0, whose last call 0x502c50 zeroes ecx
	}
	if dmg == 0 {
		return
	}
	if w.u8(0x683f94) != 0 || w.u8(m+0x2ab) != 0 {
		w.stub(0x48f650, m, 0, 1)
		e := uint32(9)
		if w.isChar(src) {
			e = 8
		}
		ecx = w.u32(src) // 0x53bea0(src, 0) leaves the source kind in ecx
		if byte(a5) != 0 {
			text := uint32(0x689b34)
			if byte(a4) != 0 {
				text = 0x61c964
			}
			w.stub(0x58aad0, e, w.u32(tbl+0x958), w.name(m), dmg, text)
			ecx = tbl
		}
	}
	if w.u8(src+0x1c) == 0 && w.u8(src+0x1b) == 0 && w.u8(src+0x1e) == 0 && byte(a4) == 0 {
		for i := uint32(0); i < 13; i++ {
			w.put(m+0x304+4*i, 4, uint64(w.u32(src+4*i)))
		}
		ecx = 0 // rep movs
		if w.u8(m+0x15) != 0 {
			ch := w.isChar(src)
			ecx = w.u32(src) // 0x53bea0(src, 0) leaves the source kind in ecx
			if ch {
				a := w.u32(m+0x10) + 0x1a + 4*w.u32(src+4)
				w.put(a, 4, uint64(w.u32(a)+dmg))
				ecx = dmg
			}
		}
	}
	hp := w.u32(m + 0x2b)
	if dmg >= hp {
		w.put(m+0x2b, 4, 0)
		w.downed(m, ecx&^0xff|1)
		return
	}
	w.put(m+0x2b, 4, uint64(hp-dmg))
	mtrFatigue(w, m, uint32(uint64(2*dmg)*0xaaaaaaab>>33))
	mtrHit(w, m, src, a4)
}

// mtrHit is 0x52beb0(m, src, a4): F runs monsterHitConditionResponseNode.
func mtrHit(w *mcaWorld, m, src, a4 uint32) {
	k := w.call(0x52beb0, m, src, a4)
	if !w.real {
		w.replay(0x52beb0, k, m, src, a4)
		return
	}
	kind := w.u32(src)
	if kind == 1 || kind == 2 {
		panic("monster turn recovery: hit response from a character or monster source (fixture has blank sources)")
	}
	in := make([]int64, 41)
	in[0] = int64(w.u32(m + 0x2b))
	for i := uint32(0); i < 20; i++ {
		in[1+i] = int64(w.u32(m + 0x57 + 4*i))
	}
	in[21], in[22] = int64(w.u32(m+0x107)), int64(w.u32(m+0x10b))
	in[23], in[24], in[25] = int64(w.u8(m+0x24b)), int64(w.u32(m+0x2fd)), int64(w.u8(m+0x255))
	in[26], in[27], in[29] = int64(byte(a4)), int64(kind), int64(w.u32(src+8))
	// characterIndex stays 0: the node only reads it for kind 1 and rejects -1.
	in[30], in[31], in[32] = int64(w.u8(src+0x1b)), int64(w.u8(src+0x1c)), int64(w.u8(src+0x1e))
	in[33], in[37], in[39] = int64(w.u8(m+0x16)), int64(w.u32(m+8)), int64(w.u8(w.template(m)+0x25f))
	o := monsterHitConditionResponseNode(in, w.r)
	for i := uint32(0); i < 20; i++ {
		w.put(m+0x57+4*i, 4, uint64(o[i]))
	}
	w.put(m+0x107, 4, uint64(o[20]))
	w.put(m+0x10b, 4, uint64(o[21]))
	w.put(m+0x2fd, 4, uint64(o[22]))
	w.put(m+0x255, 1, uint64(o[23]))
}

// mtrDecay is 0x524110(id, cond, ticks): F runs monsterConditionDecayNode.
func mtrDecay(w *mcaWorld, m, id, cond, ticks uint32) {
	k := w.call(0x524110, id, cond, ticks)
	if !w.real {
		w.replay(0x524110, k, id, cond, ticks)
		return
	}
	in := []int64{int64(id), int64(cond), int64(ticks), int64(w.u32(m + 0x2b))}
	for i := uint32(0); i < 20; i++ {
		in = append(in, int64(w.u32(m+0x57+4*i)))
	}
	in = append(in, int64(w.u32(m+0x10b)), int64(w.u32(m+0x107)))
	o := monsterConditionDecayNode(in, w.r)
	for i := uint32(0); i < 20; i++ {
		w.put(m+0x57+4*i, 4, uint64(o[i]))
	}
	w.put(m+0x10b, 4, uint64(o[20]))
	w.put(m+0x107, 4, uint64(o[21]))
}

// mtrHeal is 0x52bfd0(m, v, 0).
func mtrHeal(w *mcaWorld, m, v uint32) {
	hp, most := w.u32(m+0x2b), w.u32(m+0x27)
	if hp == 0 || hp == most || v == 0 {
		return
	}
	hp += v
	if hp > most {
		hp = most
	}
	w.put(m+0x2b, 4, uint64(hp))
}

// mtrSp is 0x52c140(m, v, 0).
func mtrSp(w *mcaWorld, m, v uint32) {
	sp, most := w.u32(m+0x33), w.u32(m+0x2f)
	if w.u32(m+0x107) >= 0x12 || w.u32(m+0x2b) == 0 || sp == most {
		return
	}
	sp += v
	if sp > most {
		sp = most
	}
	w.put(m+0x33, 4, uint64(sp))
	w.put(m+0x242, 4, uint64(staminaFatigueLevel(sp, most)))
	if w.u32(m+0x9b) == 0x270f && sp >= 10 {
		mtrCure(w, w.u32(m), sp)
	}
}

// mtrFatigue is 0x52c070(m, cost, 0); its condition 0x11 goes through the
// 0x523c00 stub (Emu replays, F runs the real apply).
func mtrFatigue(w *mcaWorld, m, cost uint32) {
	if w.u32(m+0x2b) == 0 {
		return
	}
	if cost != 0 {
		w.template(m)
		sp := w.u32(m + 0x33)
		if cost >= sp {
			sp = 0
		} else {
			sp -= cost
		}
		w.put(m+0x33, 4, uint64(sp))
	}
	sp, most := w.u32(m+0x33), w.u32(m+0x2f)
	if most == 0 {
		panic("monster turn recovery: max SP 0 divides by zero (0x52c0b8)")
	}
	w.put(m+0x242, 4, uint64(staminaFatigueLevel(sp, most)))
	if sp != 0 || w.u32(m+0x9b) >= 0x270f {
		return
	}
	for i := uint32(0); i < 13; i++ { // 0x536150
		w.put(mcaLocal+4*i, 4, 0)
	}
	w.put(mcaLocal+4, 4, 0xffffffff)
	w.put(mcaLocal+8, 4, 0xffffffff)
	id := w.u32(m)
	k := w.call(0x523c00, id, 0x11, 0x270f, 0, mcaLocal, 1)
	if w.real {
		w.tab = mtrInnerStubs
		w.apply(id, 0x11, 0x270f, 0, mcaLocal, 1)
		w.tab = mtrStubs
	} else {
		w.replay(0x523c00, k, id, 0x11, 0x270f, 0, mcaLocal, 1)
	}
}

// mtrCure is 0x523f40(id, 0x11): its only caller here is 0x52c140. The
// message call's fifth stack slot is 0x523f40's saved edi, 0x52c140's new SP.
func mtrCure(w *mcaWorld, id, edi uint32) {
	const cond = 0x11
	m := w.findMonster(id)
	if w.u32(m+0x2b) == 0 {
		return
	}
	if w.u32(m+0x57+4*cond) == 0 {
		panic("monster turn recovery: curing an inactive condition (0x523f8c asserts)")
	}
	if w.u8(0x683f94) != 0 || w.u8(m+0x2ab) != 0 {
		tbl := w.u32(0x68c09c)
		w.stub(0x58aad0, 9, w.u32(tbl+0x910), w.name(m), w.u32(tbl+4*w.u16(mtrCondText+8*cond)), edi)
	}
	w.put(m+0x57+4*cond, 4, 0)
	p := uint32(19)
	for ; p > 0 && w.u32(m+0x57+4*p) == 0; p-- {
	}
	w.put(m+0x107, 4, uint64(p))
	w.groupCount(w.group(w.u32(m + 4)))
	if w.u8(m+0x14) != 0 {
		w.visual(m, p >= 0xe)
	}
	w.stub(0x4acd80, w.u32(m+0xc), cond-1, 0)
	w.rebuild(id)
}

// mtrBuffEnd is 0x524390(id, i).
func mtrBuffEnd(w *mcaWorld, id, i uint32) {
	m := w.findMonster(id)
	for j := uint32(0); j < 3; j++ {
		w.put(m+0xa7+12*i+4*j, 4, 0)
	}
	w.stub(0x4acd80, w.u32(m+0xc), i+0x10, 0)
	w.rebuild(id)
	if i == 6 {
		w.awareness(m)
	}
}

// mtrSlotEnd is 0x5524e0(m, slot).
func mtrSlotEnd(w *mcaWorld, m, p uint32) {
	if w.u8(p) != 0 {
		spell := w.u32(p + 1)
		if spell >= mtrSpells {
			panic("monster turn recovery: spell id past the fixture's 0x60 visual ids")
		}
		w.stub(0x4acd80, w.u32(m+0xc), w.u32(mtrSlotVis+8*spell), 0)
	}
	w.put(p, 1, 0)
	w.put(p+1, 4, 0)
	w.put(p+5, 4, 0)
	w.put(p+0xd, 4, 0)
	w.rebuild(w.u32(m))
}

// Code below is generated from the same description as re/emu/cases/00503990.json.

var mtrSpec = mcaStrings(mstSpec{
	set: []mstSet{
		{mstStackBase + 4, 4, 0x20009000}, {mstStackBase + 8, 4, 0}, {mstStackBase + 12, 4, 0},
		{0x68c09c, 4, 0x20003000},
		{0x683fad, 4, 0x2000b300},
		{0x683fb1, 4, 0x2000b340},
		{0x683fb9, 4, 0x0},
		{0x683fbd, 4, 0x0},
		{0x6840c7, 4, 0x20008000},
		{0x6840cb, 4, 0x20008400},
		{0x68ec78, 4, 0x0},
		{0x68edcc, 4, 0x0},
		{0x689b34, 2, 0x0},
		{0x65a0f8, 4, 0x2000c400},
		{0x6598a4, 4, 0x2000c500},
		{0x2000c680, 4, 0x2000c500},
		{0x2000c600, 4, 0x80000},
		{0x2000c48c, 4, 0x3f800000},
		{0x2000c490, 4, 0x3f800001},
		{0x2000c494, 4, 0x3f800002},
		{0x2000b300, 4, 0x2000b320},
		{0x2000b308, 4, 0x2},
		{0x2000b340, 4, 0x2000b360},
		{0x2000b348, 4, 0x2},
		{0x2000b000, 4, 0x2000b040},
		{0x2000b058, 4, 0x2000b100},
		{0x2000b100, 1, 0x4},
		{0x2000b360, 4, 0x2000b400},
		{0x2000b400, 4, 0x1c8},
		{0x2000b408, 4, 0x2000b380},
		{0x2000b42a, 1, 0x2},
		{0x2000b42c, 1, 0x1},
		{0x2000b320, 4, 0x20009000},
		{0x20009000, 4, 0x64},
		{0x20009008, 4, 0x0},
		{0x2000900c, 4, 0x20009800},
		{0x20009010, 4, 0x2000c000},
		{0x20009800, 4, 0x5ed22c},
		{0x200099d8, 4, 0x2000a000},
		{0x200099dc, 4, 0x400},
		{0x20009984, 4, 0x2000c600},
		{0x2000a0a7, 1, 0x14},
		{0x2000a5e8, 4, 0x2000b200},
		{0x2000a208, 4, 0x2000b000},
		{0x20008187, 2, 0x18c},
		{0x2000817c, 1, 0x7f},
		{0x200080ce, 1, 0x7e},
		{0x2000902b, 4, 0x46},
		{0x20009027, 4, 0x5a},
		{0x20009033, 4, 0x1e},
		{0x2000902f, 4, 0x32},
		{0x20009302, 1, 0x9},
		{0x20009303, 1, 0x0},
		{0x2000b364, 4, 0x2000b500},
		{0x2000b500, 4, 0x1c9},
		{0x2000b508, 4, 0x2000b380},
		{0x2000b52a, 1, 0x3},
		{0x2000b52c, 1, 0x1},
		{0x2000b324, 4, 0x20009400},
		{0x20009400, 4, 0x65},
		{0x20009408, 4, 0x1},
		{0x2000940c, 4, 0x20009c00},
		{0x20009410, 4, 0x2000c000},
		{0x20009c00, 4, 0x5ed22c},
		{0x20009dd8, 4, 0x2000a800},
		{0x20009ddc, 4, 0x400},
		{0x20009d84, 4, 0x2000c600},
		{0x2000a8a7, 1, 0x14},
		{0x2000ade8, 4, 0x2000b200},
		{0x2000aa08, 4, 0x2000b000},
		{0x20008587, 2, 0x18c},
		{0x2000857c, 1, 0x7e},
		{0x200084ce, 1, 0x7d},
		{0x2000942b, 4, 0x47},
		{0x20009427, 4, 0x5b},
		{0x20009433, 4, 0x1f},
		{0x2000942f, 4, 0x33},
		{0x20009702, 1, 0xa},
		{0x20009703, 1, 0x0},
		{0x200090a7, 4, 0x11110000},
		{0x200090ab, 4, 0x22220000},
		{0x200090b3, 4, 0x11110001},
		{0x200090b7, 4, 0x22220001},
		{0x200090bf, 4, 0x11110002},
		{0x200090c3, 4, 0x22220002},
		{0x200090cb, 4, 0x11110003},
		{0x200090cf, 4, 0x22220003},
		{0x200090d7, 4, 0x11110004},
		{0x200090db, 4, 0x22220004},
		{0x200090e3, 4, 0x11110005},
		{0x200090e7, 4, 0x22220005},
		{0x200090ef, 4, 0x11110006},
		{0x200090f3, 4, 0x22220006},
		{0x200090fb, 4, 0x11110007},
		{0x200090ff, 4, 0x22220007},
		{0x20009114, 4, 0x33330000},
		{0x20009125, 4, 0x33330001},
		{0x20009136, 4, 0x33330002},
		{0x20009147, 4, 0x33330003},
		{0x20009158, 4, 0x33330004},
		{0x20009169, 4, 0x33330005},
		{0x2000917a, 4, 0x33330006},
		{0x2000918b, 4, 0x33330007},
		{0x2000919c, 4, 0x33330008},
		{0x200091ad, 4, 0x33330009},
		{0x200091be, 4, 0x3333000a},
		{0x200091cf, 4, 0x3333000b},
		{0x2000c043, 4, 0x44440000},
		{0x2000c054, 4, 0x44440001},
		{0x2000c065, 4, 0x44440002},
		{0x2000c076, 4, 0x44440003},
		{0x2000c087, 4, 0x44440004},
		{0x2000c098, 4, 0x44440005},
		{0x2000c0a9, 4, 0x44440006},
		{0x2000c0ba, 4, 0x44440007},
		{0x2000c0cb, 4, 0x44440008},
		{0x2000c0dc, 4, 0x44440009},
		{0x2000c0ed, 4, 0x4444000a},
		{0x2000c0fe, 4, 0x4444000b},
		{0x2000c10f, 4, 0x4444000c},
		{0x2000c120, 4, 0x4444000d},
		{0x2000c131, 4, 0x4444000e},
		{0x2000c14b, 1, 0x1},
	},
	in: []mstFld{
		{0x7f000008, 4, false, -1, 0, 0},
		{0x7f00000c, 1, false, -1, 0, 0},
		{0x683fc5, 1, false, -1, 0, 0},
		{0x683f94, 1, false, -1, 0, 0},
		{0x60a390, 4, false, -1, 0, 0},
		{0x20009015, 1, false, -1, 0, 0},
		{0x20009014, 1, false, -1, 0, 0},
		{0x2000924e, 1, false, -1, 0, 0},
		{0x20009a91, 1, false, -1, 0, 0},
		{0x20009874, 4, false, -1, 0, 0},
		{0x2000983d, 1, false, -1, 0, 0},
		{0x2000983c, 1, false, -1, 0, 0},
		{0x20009254, 1, true, -1, 0, 0},
		{0x20009255, 1, false, -1, 0, 0},
		{0x20009246, 1, false, -1, 0, 0},
		{0x200092ab, 1, false, -1, 0, 0},
		{0x2000a004, 4, false, -1, 0, 0},
		{0x2000a008, 4, false, -1, 0, 0},
		{0x2000a00c, 4, false, -1, 0, 0},
		{0x20009017, 4, false, -1, 0, 0},
		{0x2000901b, 4, false, -1, 0, 0},
		{0x2000901f, 4, false, -1, 0, 0},
		{0x20009338, 4, false, -1, 0, 0},
		{0x2000933c, 4, false, -1, 0, 0},
		{0x20009340, 4, false, -1, 0, 0},
		{0x20009a80, 4, false, -1, 0, 0},
		{0x20009a84, 4, false, -1, 0, 0},
		{0x20009a88, 4, false, -1, 0, 0},
		{0x20009918, 4, false, -1, 0, 0},
		{0x2000991c, 4, false, -1, 0, 0},
		{0x20009920, 4, false, -1, 0, 0},
		{0x2000c628, 4, false, -1, 0, 0},
		{0x2000c62c, 4, false, -1, 0, 0},
		{0x2000c630, 4, false, -1, 0, 0},
		{0x200098ec, 4, false, -1, 0, 0},
		{0x20009004, 4, false, -1, 0, 0},
		{0x2000b4a7, 4, false, -1, 0, 0},
		{0x2000b4ab, 4, false, -1, 0, 0},
		{0x2000b4af, 4, false, -1, 0, 0},
		{0x2000b4b3, 4, false, -1, 0, 0},
		{0x2000b5a7, 4, false, -1, 0, 0},
		{0x2000b5ab, 4, false, -1, 0, 0},
		{0x2000b5af, 4, false, -1, 0, 0},
		{0x2000b5b3, 4, false, -1, 0, 0},
		{0x20004000, 4, false, -1, 0, 0},
		{0x20004004, 4, false, -1, 0, 0},
		{0x20004008, 4, false, -1, 0, 0},
		{0x2000400c, 4, false, -1, 0, 0},
		{0x20004010, 4, false, -1, 0, 0},
		{0x20004014, 4, false, -1, 0, 0},
		{0x20004018, 1, false, -1, 0, 0},
		{0x20004019, 1, false, -1, 0, 0},
		{0x20009302, 1, false, -1, 0, 0},
		{0x20009303, 1, false, -1, 0, 0},
		{0x20009acc, 4, true, -1, 0, 0},
		{0x20009ac8, 4, true, -1, 0, 0},
		{0x200091e3, 1, false, -1, 0, 0},
		{0x200091e4, 1, true, -1, 0, 0},
		{0x200091e5, 1, true, -1, 0, 0},
		{0x2000817c, 1, true, -1, 0, 0},
		{0x200080ce, 1, true, -1, 0, 0},
		{0x2000902b, 4, false, -1, 0, 0},
		{0x20009027, 4, false, -1, 0, 0},
		{0x20009033, 4, false, -1, 0, 0},
		{0x2000902f, 4, false, -1, 0, 0},
		{0x20009047, 4, false, -1, 0, 0},
		{0x2000904b, 4, false, -1, 0, 0},
		{0x2000904f, 4, false, -1, 0, 0},
		{0x20009053, 4, false, -1, 0, 0},
		{0x200090c5, 2, false, -1, 0, 0},
		{0x20009057, 4, false, -1, 0, 0},
		{0x2000905b, 4, false, -1, 0, 0},
		{0x2000905f, 4, false, -1, 0, 0},
		{0x20009063, 4, false, -1, 0, 0},
		{0x20009067, 4, false, -1, 0, 0},
		{0x2000906b, 4, false, -1, 0, 0},
		{0x2000906f, 4, false, -1, 0, 0},
		{0x20009073, 4, false, -1, 0, 0},
		{0x20009077, 4, false, -1, 0, 0},
		{0x2000907b, 4, false, -1, 0, 0},
		{0x2000907f, 4, false, -1, 0, 0},
		{0x20009083, 4, false, -1, 0, 0},
		{0x20009087, 4, false, -1, 0, 0},
		{0x2000908b, 4, false, -1, 0, 0},
		{0x2000908f, 4, false, -1, 0, 0},
		{0x20009093, 4, false, -1, 0, 0},
		{0x20009097, 4, false, -1, 0, 0},
		{0x2000909b, 4, false, -1, 0, 0},
		{0x2000909f, 4, false, -1, 0, 0},
		{0x200090a3, 4, false, -1, 0, 0},
		{0x20009107, 4, false, -1, 0, 0},
		{0x2000910b, 4, false, -1, 0, 0},
		{0x2000924b, 1, false, -1, 0, 0},
		{0x200092fd, 4, false, -1, 0, 0},
		{0x20009016, 1, false, -1, 0, 0},
		{0x200090af, 4, false, -1, 0, 0},
		{0x200090bb, 4, false, -1, 0, 0},
		{0x200090c7, 4, false, -1, 0, 0},
		{0x200090d3, 4, false, -1, 0, 0},
		{0x200090df, 4, false, -1, 0, 0},
		{0x200090eb, 4, false, -1, 0, 0},
		{0x200090f7, 4, false, -1, 0, 0},
		{0x20009103, 4, false, -1, 0, 0},
		{0x2000910f, 1, false, -1, 0, 0},
		{0x20009110, 4, false, -1, 0, 0},
		{0x2000911c, 4, false, -1, 0, 0},
		{0x20009120, 1, false, -1, 0, 0},
		{0x20009121, 4, false, -1, 0, 0},
		{0x2000912d, 4, false, -1, 0, 0},
		{0x20009131, 1, false, -1, 0, 0},
		{0x20009132, 4, false, -1, 0, 0},
		{0x2000913e, 4, false, -1, 0, 0},
		{0x20009142, 1, false, -1, 0, 0},
		{0x20009143, 4, false, -1, 0, 0},
		{0x2000914f, 4, false, -1, 0, 0},
		{0x20009153, 1, false, -1, 0, 0},
		{0x20009154, 4, false, -1, 0, 0},
		{0x20009160, 4, false, -1, 0, 0},
		{0x20009164, 1, false, -1, 0, 0},
		{0x20009165, 4, false, -1, 0, 0},
		{0x20009171, 4, false, -1, 0, 0},
		{0x20009175, 1, false, -1, 0, 0},
		{0x20009176, 4, false, -1, 0, 0},
		{0x20009182, 4, false, -1, 0, 0},
		{0x20009186, 1, false, -1, 0, 0},
		{0x20009187, 4, false, -1, 0, 0},
		{0x20009193, 4, false, -1, 0, 0},
		{0x20009197, 1, false, -1, 0, 0},
		{0x20009198, 4, false, -1, 0, 0},
		{0x200091a4, 4, false, -1, 0, 0},
		{0x200091a8, 1, false, -1, 0, 0},
		{0x200091a9, 4, false, -1, 0, 0},
		{0x200091b5, 4, false, -1, 0, 0},
		{0x200091b9, 1, false, -1, 0, 0},
		{0x200091ba, 4, false, -1, 0, 0},
		{0x200091c6, 4, false, -1, 0, 0},
		{0x200091ca, 1, false, -1, 0, 0},
		{0x200091cb, 4, false, -1, 0, 0},
		{0x200091d7, 4, false, -1, 0, 0},
		{0x2000c03e, 1, false, -1, 0, 0},
		{0x2000c03f, 4, false, -1, 0, 0},
		{0x2000c04b, 4, false, -1, 0, 0},
		{0x2000c04f, 1, false, -1, 0, 0},
		{0x2000c050, 4, false, -1, 0, 0},
		{0x2000c05c, 4, false, -1, 0, 0},
		{0x2000c060, 1, false, -1, 0, 0},
		{0x2000c061, 4, false, -1, 0, 0},
		{0x2000c06d, 4, false, -1, 0, 0},
		{0x2000c071, 1, false, -1, 0, 0},
		{0x2000c072, 4, false, -1, 0, 0},
		{0x2000c07e, 4, false, -1, 0, 0},
		{0x2000c082, 1, false, -1, 0, 0},
		{0x2000c083, 4, false, -1, 0, 0},
		{0x2000c08f, 4, false, -1, 0, 0},
		{0x2000c093, 1, false, -1, 0, 0},
		{0x2000c094, 4, false, -1, 0, 0},
		{0x2000c0a0, 4, false, -1, 0, 0},
		{0x2000c0a4, 1, false, -1, 0, 0},
		{0x2000c0a5, 4, false, -1, 0, 0},
		{0x2000c0b1, 4, false, -1, 0, 0},
		{0x2000c0b5, 1, false, -1, 0, 0},
		{0x2000c0b6, 4, false, -1, 0, 0},
		{0x2000c0c2, 4, false, -1, 0, 0},
		{0x2000c0c6, 1, false, -1, 0, 0},
		{0x2000c0c7, 4, false, -1, 0, 0},
		{0x2000c0d3, 4, false, -1, 0, 0},
		{0x2000c0d7, 1, false, -1, 0, 0},
		{0x2000c0d8, 4, false, -1, 0, 0},
		{0x2000c0e4, 4, false, -1, 0, 0},
		{0x2000c0e8, 1, false, -1, 0, 0},
		{0x2000c0e9, 4, false, -1, 0, 0},
		{0x2000c0f5, 4, false, -1, 0, 0},
		{0x2000c0f9, 1, false, -1, 0, 0},
		{0x2000c0fa, 4, false, -1, 0, 0},
		{0x2000c106, 4, false, -1, 0, 0},
		{0x2000c10a, 1, false, -1, 0, 0},
		{0x2000c10b, 4, false, -1, 0, 0},
		{0x2000c117, 4, false, -1, 0, 0},
		{0x2000c11b, 1, false, -1, 0, 0},
		{0x2000c11c, 4, false, -1, 0, 0},
		{0x2000c128, 4, false, -1, 0, 0},
		{0x2000c12c, 1, false, -1, 0, 0},
		{0x2000c12d, 4, false, -1, 0, 0},
		{0x2000c139, 4, false, -1, 0, 0},
		{0x2000a0a4, 1, false, -1, 0, 0},
	},
	out: []mstFld{
		{0x20004040, 4, false, -1, 0, 0},
		{0x20004044, 4, false, -1, 0, 0},
		{0x20004048, 4, false, -1, 0, 0},
		{0x2000404c, 4, false, -1, 0, 0},
		{0x20004050, 4, false, -1, 0, 0},
		{0x20004054, 4, false, -1, 0, 0},
		{0x20004058, 4, false, -1, 0, 0},
		{0x2000405c, 4, false, -1, 0, 0},
		{0x20004060, 4, false, -1, 0, 0},
		{0x20004064, 4, false, -1, 0, 0},
		{0x20004068, 4, false, -1, 0, 0},
		{0x2000406c, 4, false, -1, 0, 0},
		{0x20004070, 4, false, -1, 0, 0},
		{0x20004074, 4, false, -1, 0, 0},
		{0x20004078, 4, false, -1, 0, 0},
		{0x2000407c, 4, false, -1, 0, 0},
		{0x20004080, 4, false, -1, 0, 0},
		{0x20004084, 4, false, -1, 0, 0},
		{0x20004088, 4, false, -1, 0, 0},
		{0x2000408c, 4, false, -1, 0, 0},
		{0x20004090, 4, false, -1, 0, 0},
		{0x20004094, 4, false, -1, 0, 0},
		{0x20004098, 4, false, -1, 0, 0},
		{0x2000409c, 4, false, -1, 0, 0},
		{0x200040a0, 4, false, -1, 0, 0},
		{0x200040a4, 4, false, -1, 0, 0},
		{0x200040a8, 4, false, -1, 0, 0},
		{0x200040ac, 4, false, -1, 0, 0},
		{0x200040b0, 4, false, -1, 0, 0},
		{0x200040b4, 4, false, -1, 0, 0},
		{0x200040b8, 4, false, -1, 0, 0},
		{0x200040bc, 4, false, -1, 0, 0},
		{0x200040c0, 4, false, -1, 0, 0},
		{0x200040c4, 4, false, -1, 0, 0},
		{0x200040c8, 4, false, -1, 0, 0},
		{0x200040cc, 4, false, -1, 0, 0},
		{0x200040d0, 4, false, -1, 0, 0},
		{0x200040d4, 4, false, -1, 0, 0},
		{0x200040d8, 4, false, -1, 0, 0},
		{0x200040dc, 4, false, -1, 0, 0},
		{0x200040e0, 4, false, -1, 0, 0},
		{0x200040e4, 4, false, -1, 0, 0},
		{0x200040e8, 4, false, -1, 0, 0},
		{0x200040ec, 4, false, -1, 0, 0},
		{0x200040f0, 4, false, -1, 0, 0},
		{0x200040f4, 4, false, -1, 0, 0},
		{0x200040f8, 4, false, -1, 0, 0},
		{0x200040fc, 4, false, -1, 0, 0},
		{0x20004100, 4, false, -1, 0, 0},
		{0x20004104, 4, false, -1, 0, 0},
		{0x20004108, 4, false, -1, 0, 0},
		{0x2000410c, 4, false, -1, 0, 0},
		{0x20004110, 4, false, -1, 0, 0},
		{0x20004114, 4, false, -1, 0, 0},
		{0x20004118, 4, false, -1, 0, 0},
		{0x2000411c, 4, false, -1, 0, 0},
		{0x20004120, 4, false, -1, 0, 0},
		{0x20004124, 4, false, -1, 0, 0},
		{0x20004128, 4, false, -1, 0, 0},
		{0x2000412c, 4, false, -1, 0, 0},
		{0x20004130, 4, false, -1, 0, 0},
		{0x20004134, 4, false, -1, 0, 0},
		{0x20004138, 4, false, -1, 0, 0},
		{0x2000413c, 4, false, -1, 0, 0},
		{0x20004140, 4, false, -1, 0, 0},
		{0x20004144, 4, false, -1, 0, 0},
		{0x20004148, 4, false, -1, 0, 0},
		{0x2000414c, 4, false, -1, 0, 0},
		{0x20004150, 4, false, -1, 0, 0},
		{0x20004154, 4, false, -1, 0, 0},
		{0x20004158, 4, false, -1, 0, 0},
		{0x2000415c, 4, false, -1, 0, 0},
		{0x20004160, 4, false, -1, 0, 0},
		{0x20004164, 4, false, -1, 0, 0},
		{0x20004168, 4, false, -1, 0, 0},
		{0x2000416c, 4, false, -1, 0, 0},
		{0x20004170, 4, false, -1, 0, 0},
		{0x20004174, 4, false, -1, 0, 0},
		{0x20004178, 4, false, -1, 0, 0},
		{0x2000417c, 1, false, -1, 0, 0},
		{0x2000417d, 1, false, -1, 0, 0},
		{0x2000417e, 1, false, -1, 0, 0},
		{0x2000417f, 4, false, -1, 0, 0},
		{0x20004183, 4, false, -1, 0, 0},
		{0x20004187, 4, false, -1, 0, 0},
		{0x2000418b, 4, false, -1, 0, 0},
		{0x2000418f, 4, false, -1, 0, 0},
		{0x20004193, 4, false, -1, 0, 0},
		{0x20004197, 4, false, -1, 0, 0},
		{0x2000419b, 4, false, -1, 0, 0},
		{0x2000419f, 4, false, -1, 0, 0},
		{0x200041a3, 4, false, -1, 0, 0},
		{0x200041a7, 4, false, -1, 0, 0},
		{0x200041ab, 4, false, -1, 0, 0},
		{0x200041af, 4, false, -1, 0, 0},
		{0x200041b3, 4, false, -1, 0, 0},
		{0x200041b7, 4, false, -1, 0, 0},
		{0x200041bb, 4, false, -1, 0, 0},
		{0x200041bf, 4, false, -1, 0, 0},
		{0x200041c3, 4, false, -1, 0, 0},
		{0x200041c7, 4, false, -1, 0, 0},
		{0x200041cb, 4, false, -1, 0, 0},
		{0x200041cf, 4, false, -1, 0, 0},
		{0x200041d3, 4, false, -1, 0, 0},
		{0x200041d7, 4, false, -1, 0, 0},
		{0x200041db, 4, false, -1, 0, 0},
		{0x200041df, 4, false, -1, 0, 0},
		{0x200041e3, 4, false, -1, 0, 0},
		{0x200041e7, 4, false, -1, 0, 0},
		{0x200041eb, 4, false, -1, 0, 0},
		{0x200041ef, 4, false, -1, 0, 0},
		{0x200041f3, 4, false, -1, 0, 0},
		{0x200041f7, 4, false, -1, 0, 0},
		{0x200041fb, 4, false, -1, 0, 0},
		{0x200041ff, 4, false, -1, 0, 0},
		{0x20004203, 4, false, -1, 0, 0},
		{0x20004207, 4, false, -1, 0, 0},
		{0x2000420b, 4, false, -1, 0, 0},
		{0x2000420f, 4, false, -1, 0, 0},
		{0x20004213, 4, false, -1, 0, 0},
		{0x20004217, 4, false, -1, 0, 0},
		{0x2000421b, 4, false, -1, 0, 0},
		{0x2000421f, 4, false, -1, 0, 0},
		{0x20004223, 4, false, -1, 0, 0},
		{0x20004227, 4, false, -1, 0, 0},
		{0x2000422b, 4, false, -1, 0, 0},
		{0x2000422f, 4, false, -1, 0, 0},
		{0x20004233, 4, false, -1, 0, 0},
		{0x20004237, 4, false, -1, 0, 0},
		{0x2000423b, 4, false, -1, 0, 0},
		{0x2000423f, 4, false, -1, 0, 0},
		{0x20004243, 4, false, -1, 0, 0},
		{0x20004247, 4, false, -1, 0, 0},
		{0x2000424b, 4, false, -1, 0, 0},
		{0x2000424f, 4, false, -1, 0, 0},
		{0x20004253, 4, false, -1, 0, 0},
		{0x20004257, 4, false, -1, 0, 0},
		{0x2000425b, 4, false, -1, 0, 0},
		{0x2000425f, 4, false, -1, 0, 0},
		{0x20004263, 4, false, -1, 0, 0},
		{0x20004267, 4, false, -1, 0, 0},
		{0x2000426b, 4, false, -1, 0, 0},
		{0x2000426f, 4, false, -1, 0, 0},
		{0x20004273, 4, false, -1, 0, 0},
		{0x20004277, 4, false, -1, 0, 0},
		{0x2000427b, 4, false, -1, 0, 0},
		{0x2000427f, 4, false, -1, 0, 0},
		{0x20004283, 4, false, -1, 0, 0},
		{0x20004287, 4, false, -1, 0, 0},
		{0x2000428b, 4, false, -1, 0, 0},
		{0x2000428f, 4, false, -1, 0, 0},
		{0x20004293, 4, false, -1, 0, 0},
		{0x20004297, 4, false, -1, 0, 0},
		{0x2000429b, 4, false, -1, 0, 0},
		{0x2000429f, 4, false, -1, 0, 0},
		{0x200042a3, 4, false, -1, 0, 0},
		{0x200042a7, 4, false, -1, 0, 0},
		{0x200042ab, 4, false, -1, 0, 0},
		{0x200042af, 4, false, -1, 0, 0},
		{0x200042b3, 4, false, -1, 0, 0},
		{0x200042b7, 4, false, -1, 0, 0},
		{0x200042bb, 4, false, -1, 0, 0},
		{0x200042bf, 4, false, -1, 0, 0},
		{0x200042c3, 4, false, -1, 0, 0},
		{0x200042c7, 4, false, -1, 0, 0},
		{0x200042cb, 4, false, -1, 0, 0},
		{0x200042cf, 4, false, -1, 0, 0},
		{0x200042d3, 4, false, -1, 0, 0},
		{0x200042d7, 4, false, -1, 0, 0},
		{0x200042db, 4, false, -1, 0, 0},
		{0x200042df, 4, false, -1, 0, 0},
		{0x200042e3, 4, false, -1, 0, 0},
		{0x200042e7, 4, false, -1, 0, 0},
		{0x200042eb, 4, false, -1, 0, 0},
		{0x200042ef, 4, false, -1, 0, 0},
		{0x200042f3, 4, false, -1, 0, 0},
		{0x200042f7, 4, false, -1, 0, 0},
		{0x200042fb, 4, false, -1, 0, 0},
		{0x200042ff, 4, false, -1, 0, 0},
		{0x20004303, 4, false, -1, 0, 0},
		{0x20004307, 4, false, -1, 0, 0},
		{0x2000430b, 4, false, -1, 0, 0},
		{0x2000430f, 4, false, -1, 0, 0},
		{0x20004313, 4, false, -1, 0, 0},
		{0x20004317, 4, false, -1, 0, 0},
		{0x2000431b, 4, false, -1, 0, 0},
		{0x2000431f, 4, false, -1, 0, 0},
		{0x20004323, 4, false, -1, 0, 0},
		{0x20004327, 4, false, -1, 0, 0},
		{0x2000432b, 4, false, -1, 0, 0},
		{0x2000432f, 4, false, -1, 0, 0},
		{0x20004333, 4, false, -1, 0, 0},
		{0x20004337, 4, false, -1, 0, 0},
		{0x2000433b, 4, false, -1, 0, 0},
		{0x2000433f, 4, false, -1, 0, 0},
		{0x20004343, 4, false, -1, 0, 0},
		{0x20004347, 4, false, -1, 0, 0},
		{0x2000434b, 4, false, -1, 0, 0},
		{0x2000902b, 4, false, -1, 0, 0},
		{0x20009033, 4, false, -1, 0, 0},
		{0x2000904b, 4, false, -1, 0, 0},
		{0x20009053, 4, false, -1, 0, 0},
		{0x20009242, 4, false, -1, 0, 0},
		{0x20009107, 4, false, -1, 0, 0},
		{0x2000910b, 4, false, -1, 0, 0},
		{0x20009254, 1, false, -1, 0, 0},
		{0x20009255, 1, false, -1, 0, 0},
		{0x20009246, 1, false, -1, 0, 0},
		{0x200092ab, 1, false, -1, 0, 0},
		{0x20009302, 1, false, -1, 0, 0},
		{0x20009303, 1, false, -1, 0, 0},
		{0x200090c5, 2, false, -1, 0, 0},
		{0x2000924e, 1, false, -1, 0, 0},
		{0x20009016, 1, false, -1, 0, 0},
		{0x20009047, 4, false, -1, 0, 0},
		{0x2000904f, 4, false, -1, 0, 0},
		{0x200092fd, 4, false, -1, 0, 0},
		{0x20009017, 4, false, -1, 0, 0},
		{0x20009338, 4, false, -1, 0, 0},
		{0x20009a80, 4, false, -1, 0, 0},
		{0x2000901b, 4, false, -1, 0, 0},
		{0x2000933c, 4, false, -1, 0, 0},
		{0x20009a84, 4, false, -1, 0, 0},
		{0x2000901f, 4, false, -1, 0, 0},
		{0x20009340, 4, false, -1, 0, 0},
		{0x20009a88, 4, false, -1, 0, 0},
		{0x20009057, 4, false, -1, 0, 0},
		{0x2000905b, 4, false, -1, 0, 0},
		{0x2000905f, 4, false, -1, 0, 0},
		{0x20009063, 4, false, -1, 0, 0},
		{0x20009067, 4, false, -1, 0, 0},
		{0x2000906b, 4, false, -1, 0, 0},
		{0x2000906f, 4, false, -1, 0, 0},
		{0x20009073, 4, false, -1, 0, 0},
		{0x20009077, 4, false, -1, 0, 0},
		{0x2000907b, 4, false, -1, 0, 0},
		{0x2000907f, 4, false, -1, 0, 0},
		{0x20009083, 4, false, -1, 0, 0},
		{0x20009087, 4, false, -1, 0, 0},
		{0x2000908b, 4, false, -1, 0, 0},
		{0x2000908f, 4, false, -1, 0, 0},
		{0x20009093, 4, false, -1, 0, 0},
		{0x20009097, 4, false, -1, 0, 0},
		{0x2000909b, 4, false, -1, 0, 0},
		{0x2000909f, 4, false, -1, 0, 0},
		{0x200090a3, 4, false, -1, 0, 0},
		{0x200090a7, 4, false, -1, 0, 0},
		{0x200090ab, 4, false, -1, 0, 0},
		{0x200090af, 4, false, -1, 0, 0},
		{0x200090b3, 4, false, -1, 0, 0},
		{0x200090b7, 4, false, -1, 0, 0},
		{0x200090bb, 4, false, -1, 0, 0},
		{0x200090bf, 4, false, -1, 0, 0},
		{0x200090c3, 4, false, -1, 0, 0},
		{0x200090c7, 4, false, -1, 0, 0},
		{0x200090cb, 4, false, -1, 0, 0},
		{0x200090cf, 4, false, -1, 0, 0},
		{0x200090d3, 4, false, -1, 0, 0},
		{0x200090d7, 4, false, -1, 0, 0},
		{0x200090db, 4, false, -1, 0, 0},
		{0x200090df, 4, false, -1, 0, 0},
		{0x200090e3, 4, false, -1, 0, 0},
		{0x200090e7, 4, false, -1, 0, 0},
		{0x200090eb, 4, false, -1, 0, 0},
		{0x200090ef, 4, false, -1, 0, 0},
		{0x200090f3, 4, false, -1, 0, 0},
		{0x200090f7, 4, false, -1, 0, 0},
		{0x200090fb, 4, false, -1, 0, 0},
		{0x200090ff, 4, false, -1, 0, 0},
		{0x20009103, 4, false, -1, 0, 0},
		{0x20009304, 4, false, -1, 0, 0},
		{0x20009308, 4, false, -1, 0, 0},
		{0x2000930c, 4, false, -1, 0, 0},
		{0x20009310, 4, false, -1, 0, 0},
		{0x20009314, 4, false, -1, 0, 0},
		{0x20009318, 4, false, -1, 0, 0},
		{0x2000931c, 4, false, -1, 0, 0},
		{0x20009320, 4, false, -1, 0, 0},
		{0x20009324, 4, false, -1, 0, 0},
		{0x20009328, 4, false, -1, 0, 0},
		{0x2000932c, 4, false, -1, 0, 0},
		{0x20009330, 4, false, -1, 0, 0},
		{0x20009334, 4, false, -1, 0, 0},
		{0x2000910f, 1, false, -1, 0, 0},
		{0x20009110, 4, false, -1, 0, 0},
		{0x20009114, 4, false, -1, 0, 0},
		{0x2000911c, 4, false, -1, 0, 0},
		{0x20009120, 1, false, -1, 0, 0},
		{0x20009121, 4, false, -1, 0, 0},
		{0x20009125, 4, false, -1, 0, 0},
		{0x2000912d, 4, false, -1, 0, 0},
		{0x20009131, 1, false, -1, 0, 0},
		{0x20009132, 4, false, -1, 0, 0},
		{0x20009136, 4, false, -1, 0, 0},
		{0x2000913e, 4, false, -1, 0, 0},
		{0x20009142, 1, false, -1, 0, 0},
		{0x20009143, 4, false, -1, 0, 0},
		{0x20009147, 4, false, -1, 0, 0},
		{0x2000914f, 4, false, -1, 0, 0},
		{0x20009153, 1, false, -1, 0, 0},
		{0x20009154, 4, false, -1, 0, 0},
		{0x20009158, 4, false, -1, 0, 0},
		{0x20009160, 4, false, -1, 0, 0},
		{0x20009164, 1, false, -1, 0, 0},
		{0x20009165, 4, false, -1, 0, 0},
		{0x20009169, 4, false, -1, 0, 0},
		{0x20009171, 4, false, -1, 0, 0},
		{0x20009175, 1, false, -1, 0, 0},
		{0x20009176, 4, false, -1, 0, 0},
		{0x2000917a, 4, false, -1, 0, 0},
		{0x20009182, 4, false, -1, 0, 0},
		{0x20009186, 1, false, -1, 0, 0},
		{0x20009187, 4, false, -1, 0, 0},
		{0x2000918b, 4, false, -1, 0, 0},
		{0x20009193, 4, false, -1, 0, 0},
		{0x20009197, 1, false, -1, 0, 0},
		{0x20009198, 4, false, -1, 0, 0},
		{0x2000919c, 4, false, -1, 0, 0},
		{0x200091a4, 4, false, -1, 0, 0},
		{0x200091a8, 1, false, -1, 0, 0},
		{0x200091a9, 4, false, -1, 0, 0},
		{0x200091ad, 4, false, -1, 0, 0},
		{0x200091b5, 4, false, -1, 0, 0},
		{0x200091b9, 1, false, -1, 0, 0},
		{0x200091ba, 4, false, -1, 0, 0},
		{0x200091be, 4, false, -1, 0, 0},
		{0x200091c6, 4, false, -1, 0, 0},
		{0x200091ca, 1, false, -1, 0, 0},
		{0x200091cb, 4, false, -1, 0, 0},
		{0x200091cf, 4, false, -1, 0, 0},
		{0x200091d7, 4, false, -1, 0, 0},
		{0x2000c03e, 1, false, -1, 0, 0},
		{0x2000c03f, 4, false, -1, 0, 0},
		{0x2000c043, 4, false, -1, 0, 0},
		{0x2000c04b, 4, false, -1, 0, 0},
		{0x2000c04f, 1, false, -1, 0, 0},
		{0x2000c050, 4, false, -1, 0, 0},
		{0x2000c054, 4, false, -1, 0, 0},
		{0x2000c05c, 4, false, -1, 0, 0},
		{0x2000c060, 1, false, -1, 0, 0},
		{0x2000c061, 4, false, -1, 0, 0},
		{0x2000c065, 4, false, -1, 0, 0},
		{0x2000c06d, 4, false, -1, 0, 0},
		{0x2000c071, 1, false, -1, 0, 0},
		{0x2000c072, 4, false, -1, 0, 0},
		{0x2000c076, 4, false, -1, 0, 0},
		{0x2000c07e, 4, false, -1, 0, 0},
		{0x2000c082, 1, false, -1, 0, 0},
		{0x2000c083, 4, false, -1, 0, 0},
		{0x2000c087, 4, false, -1, 0, 0},
		{0x2000c08f, 4, false, -1, 0, 0},
		{0x2000c093, 1, false, -1, 0, 0},
		{0x2000c094, 4, false, -1, 0, 0},
		{0x2000c098, 4, false, -1, 0, 0},
		{0x2000c0a0, 4, false, -1, 0, 0},
		{0x2000c0a4, 1, false, -1, 0, 0},
		{0x2000c0a5, 4, false, -1, 0, 0},
		{0x2000c0a9, 4, false, -1, 0, 0},
		{0x2000c0b1, 4, false, -1, 0, 0},
		{0x2000c0b5, 1, false, -1, 0, 0},
		{0x2000c0b6, 4, false, -1, 0, 0},
		{0x2000c0ba, 4, false, -1, 0, 0},
		{0x2000c0c2, 4, false, -1, 0, 0},
		{0x2000c0c6, 1, false, -1, 0, 0},
		{0x2000c0c7, 4, false, -1, 0, 0},
		{0x2000c0cb, 4, false, -1, 0, 0},
		{0x2000c0d3, 4, false, -1, 0, 0},
		{0x2000c0d7, 1, false, -1, 0, 0},
		{0x2000c0d8, 4, false, -1, 0, 0},
		{0x2000c0dc, 4, false, -1, 0, 0},
		{0x2000c0e4, 4, false, -1, 0, 0},
		{0x2000c0e8, 1, false, -1, 0, 0},
		{0x2000c0e9, 4, false, -1, 0, 0},
		{0x2000c0ed, 4, false, -1, 0, 0},
		{0x2000c0f5, 4, false, -1, 0, 0},
		{0x2000c0f9, 1, false, -1, 0, 0},
		{0x2000c0fa, 4, false, -1, 0, 0},
		{0x2000c0fe, 4, false, -1, 0, 0},
		{0x2000c106, 4, false, -1, 0, 0},
		{0x2000c10a, 1, false, -1, 0, 0},
		{0x2000c10b, 4, false, -1, 0, 0},
		{0x2000c10f, 4, false, -1, 0, 0},
		{0x2000c117, 4, false, -1, 0, 0},
		{0x2000c11b, 1, false, -1, 0, 0},
		{0x2000c11c, 4, false, -1, 0, 0},
		{0x2000c120, 4, false, -1, 0, 0},
		{0x2000c128, 4, false, -1, 0, 0},
		{0x2000c12c, 1, false, -1, 0, 0},
		{0x2000c12d, 4, false, -1, 0, 0},
		{0x2000c131, 4, false, -1, 0, 0},
		{0x2000c139, 4, false, -1, 0, 0},
		{0x2000c14b, 1, false, -1, 0, 0},
		{0x2000942b, 4, false, -1, 0, 0},
		{0x20009702, 1, false, -1, 0, 0},
	},
})

var mtrIn = []string{"ticks", "fullFlag", "fullGlobal", "messageAll", "trigger", "teamMember", "present", "visualFlag", "homing", "busy", "flag3d", "flag3c", "state", "homeFlag", "homeState", "inCombat", "posx", "posy", "posz", "homex", "homey", "homez", "prevx", "prevy", "prevz", "targetx", "targety", "targetz", "actorHomex", "actorHomey", "actorHomez", "pathPointx", "pathPointy", "pathPointz", "heading", "group", "partner0_0", "partner0_1", "partner0_2", "partner0_3", "partner1_0", "partner1_1", "partner1_2", "partner1_3", "visibleRet0", "visibleRet1", "sceneRet0", "sceneRet1", "placeRet", "ignoredReturn", "awareness0", "awareness1", "timerA", "timerB", "spanA", "spanB", "dot", "hpRegen", "spRegen", "templateHpRegen", "templateSpRegen", "hp", "maxHp", "sp", "maxSp", "hpRate", "hpCarry", "spRate", "spCarry", "shield", "dur0", "dur1", "dur2", "dur3", "dur4", "dur5", "dur6", "dur7", "dur8", "dur9", "dur10", "dur11", "dur12", "dur13", "dur14", "dur15", "dur16", "dur17", "dur18", "dur19", "primary", "poison", "senses", "reaction", "morale", "buff0", "buff1", "buff2", "buff3", "buff4", "buff5", "buff6", "buff7", "slot0Kind", "slot0Spell", "slot0Left", "slot1Kind", "slot1Spell", "slot1Left", "slot2Kind", "slot2Spell", "slot2Left", "slot3Kind", "slot3Spell", "slot3Left", "slot4Kind", "slot4Spell", "slot4Left", "slot5Kind", "slot5Spell", "slot5Left", "slot6Kind", "slot6Spell", "slot6Left", "slot7Kind", "slot7Spell", "slot7Left", "slot8Kind", "slot8Spell", "slot8Left", "slot9Kind", "slot9Spell", "slot9Left", "slot10Kind", "slot10Spell", "slot10Left", "slot11Kind", "slot11Spell", "slot11Left", "team0Kind", "team0Spell", "team0Left", "team1Kind", "team1Spell", "team1Left", "team2Kind", "team2Spell", "team2Left", "team3Kind", "team3Spell", "team3Left", "team4Kind", "team4Spell", "team4Left", "team5Kind", "team5Spell", "team5Left", "team6Kind", "team6Spell", "team6Left", "team7Kind", "team7Spell", "team7Left", "team8Kind", "team8Spell", "team8Left", "team9Kind", "team9Spell", "team9Left", "team10Kind", "team10Spell", "team10Left", "team11Kind", "team11Spell", "team11Left", "team12Kind", "team12Spell", "team12Left", "team13Kind", "team13Spell", "team13Left", "team14Kind", "team14Spell", "team14Left", "actorState"}

var mtrOut = []string{"s524110_calls", "s524110_a1_0", "s524110_a1_1", "s524110_a1_2", "s524110_a1_3", "s524110_a1_4", "s524110_a1_5", "s524110_a1_6", "s524110_a1_7", "s524110_a1_8", "s524110_a1_9", "s524110_a1_10", "s524110_a1_11", "s524110_a1_12", "s524110_a1_13", "s524110_a1_14", "s524110_a1_15", "s524110_a1_16", "s524110_a1_17", "s524110_a1_18", "s524110_a1_19", "s524110_a2_0", "s524110_a2_1", "s524110_a2_2", "s524110_a2_3", "s524110_a2_4", "s524110_a2_5", "s524110_a2_6", "s524110_a2_7", "s524110_a2_8", "s524110_a2_9", "s524110_a2_10", "s524110_a2_11", "s524110_a2_12", "s524110_a2_13", "s524110_a2_14", "s524110_a2_15", "s524110_a2_16", "s524110_a2_17", "s524110_a2_18", "s524110_a2_19", "s524110_a3_0", "s524110_a3_1", "s524110_a3_2", "s524110_a3_3", "s524110_a3_4", "s524110_a3_5", "s524110_a3_6", "s524110_a3_7", "s524110_a3_8", "s524110_a3_9", "s524110_a3_10", "s524110_a3_11", "s524110_a3_12", "s524110_a3_13", "s524110_a3_14", "s524110_a3_15", "s524110_a3_16", "s524110_a3_17", "s524110_a3_18", "s524110_a3_19", "s52beb0_calls", "s52beb0_a1_0", "s52beb0_a1_1", "s52beb0_a3_0", "s52beb0_a3_1", "s523c00_calls", "s523c00_a1_0", "s523c00_a1_1", "s523c00_a1_2", "s523c00_a2_0", "s523c00_a2_1", "s523c00_a2_2", "s523c00_a3_0", "s523c00_a3_1", "s523c00_a3_2", "s523c00_a4_0", "s523c00_a4_1", "s523c00_a4_2", "s523c00_a6_0", "s523c00_a6_1", "s523c00_a6_2", "s5526f0_calls", "s5526f0_a1_0", "s4e4280_calls", "s4e4280_a1_0", "s4e4280_a1_1", "s4e46f0_calls", "s4e46f0_a1_0", "s4e46f0_a1_1", "s4e46f0_a2_0", "s4e46f0_a2_1", "s5049c0_calls", "s5049c0_a2_0", "s5049c0_a2_1", "s5049c0_a3_0", "s5049c0_a3_1", "s434b60_calls", "s462e60_calls", "s4be940_calls", "s510cc0_calls", "s510cc0_a1_0", "s510cc0_a1_1", "s510cc0_a1_2", "s510cc0_a1_3", "s510cc0_a1_4", "s510cc0_a3_0", "s510cc0_a3_1", "s510cc0_a3_2", "s510cc0_a3_3", "s510cc0_a3_4", "s4acd80_calls", "s4acd80_a2_0", "s4acd80_a2_1", "s4acd80_a2_2", "s4acd80_a2_3", "s4acd80_a2_4", "s4acd80_a2_5", "s4acd80_a2_6", "s4acd80_a2_7", "s4acd80_a2_8", "s4acd80_a2_9", "s4acd80_a2_10", "s4acd80_a2_11", "s4acd80_a2_12", "s4acd80_a2_13", "s4acd80_a2_14", "s4acd80_a2_15", "s4acd80_a2_16", "s4acd80_a2_17", "s4acd80_a2_18", "s4acd80_a2_19", "s4acd80_a2_20", "s4acd80_a2_21", "s4acd80_a2_22", "s4acd80_a2_23", "s4acd80_a2_24", "s4acd80_a2_25", "s4acd80_a2_26", "s4acd80_a2_27", "s4acd80_a2_28", "s4acd80_a2_29", "s4acd80_a2_30", "s4acd80_a2_31", "s4acd80_a2_32", "s4acd80_a2_33", "s4acd80_a2_34", "s4acd80_a2_35", "s4acd80_a2_36", "s4acd80_a2_37", "s4acd80_a2_38", "s4acd80_a2_39", "s590b40_calls", "s590b40_a1_0", "s590b40_a1_1", "s590b40_a2_0", "s590b40_a2_1", "s590b40_a3_0", "s590b40_a3_1", "s58aad0_calls", "s58aad0_a1_0", "s58aad0_a1_1", "s58aad0_a1_2", "s58aad0_a1_3", "s58aad0_a2_0", "s58aad0_a2_1", "s58aad0_a2_2", "s58aad0_a2_3", "s58aad0_a3_0", "s58aad0_a3_1", "s58aad0_a3_2", "s58aad0_a3_3", "s58aad0_a4_0", "s58aad0_a4_1", "s58aad0_a4_2", "s58aad0_a4_3", "s58aad0_a5_0", "s58aad0_a5_1", "s58aad0_a5_2", "s58aad0_a5_3", "s48f650_calls", "s48f650_a1_0", "s48f650_a1_1", "s4e4db0_calls", "s4e4db0_a2_0", "s4e4db0_a2_1", "s4e4db0_a2_2", "s4e4db0_a2_3", "s4e4db0_a3_0", "s4e4db0_a3_1", "s4e4db0_a3_2", "s4e4db0_a3_3", "s4e3af0_calls", "s4e3af0_a1_0", "s4e3af0_a1_1", "s4c5a00_calls", "s4c5a00_a2_0", "s4c5a00_a2_1", "hp", "sp", "hpCarry", "spCarry", "fatigue", "primary", "poison", "state", "homeFlag", "homeState", "inCombat", "timerA", "timerB", "shield", "visualFlag", "morale", "hpRecovery", "spRecovery", "reaction", "homex", "prevx", "targetx", "homey", "prevy", "targety", "homez", "prevz", "targetz", "dur0", "dur1", "dur2", "dur3", "dur4", "dur5", "dur6", "dur7", "dur8", "dur9", "dur10", "dur11", "dur12", "dur13", "dur14", "dur15", "dur16", "dur17", "dur18", "dur19", "buff0_0", "buff0_1", "buff0_2", "buff1_0", "buff1_1", "buff1_2", "buff2_0", "buff2_1", "buff2_2", "buff3_0", "buff3_1", "buff3_2", "buff4_0", "buff4_1", "buff4_2", "buff5_0", "buff5_1", "buff5_2", "buff6_0", "buff6_1", "buff6_2", "buff7_0", "buff7_1", "buff7_2", "lastSource0", "lastSource1", "lastSource2", "lastSource3", "lastSource4", "lastSource5", "lastSource6", "lastSource7", "lastSource8", "lastSource9", "lastSource10", "lastSource11", "lastSource12", "slot0Kind", "slot0_1", "slot0_5", "slot0_d", "slot1Kind", "slot1_1", "slot1_5", "slot1_d", "slot2Kind", "slot2_1", "slot2_5", "slot2_d", "slot3Kind", "slot3_1", "slot3_5", "slot3_d", "slot4Kind", "slot4_1", "slot4_5", "slot4_d", "slot5Kind", "slot5_1", "slot5_5", "slot5_d", "slot6Kind", "slot6_1", "slot6_5", "slot6_d", "slot7Kind", "slot7_1", "slot7_5", "slot7_d", "slot8Kind", "slot8_1", "slot8_5", "slot8_d", "slot9Kind", "slot9_1", "slot9_5", "slot9_d", "slot10Kind", "slot10_1", "slot10_5", "slot10_d", "slot11Kind", "slot11_1", "slot11_5", "slot11_d", "team0Kind", "team0_1", "team0_5", "team0_d", "team1Kind", "team1_1", "team1_5", "team1_d", "team2Kind", "team2_1", "team2_5", "team2_d", "team3Kind", "team3_1", "team3_5", "team3_d", "team4Kind", "team4_1", "team4_5", "team4_d", "team5Kind", "team5_1", "team5_5", "team5_d", "team6Kind", "team6_1", "team6_5", "team6_d", "team7Kind", "team7_1", "team7_5", "team7_d", "team8Kind", "team8_1", "team8_5", "team8_d", "team9Kind", "team9_1", "team9_5", "team9_d", "team10Kind", "team10_1", "team10_5", "team10_d", "team11Kind", "team11_1", "team11_5", "team11_d", "team12Kind", "team12_1", "team12_5", "team12_d", "team13Kind", "team13_1", "team13_5", "team13_d", "team14Kind", "team14_1", "team14_5", "team14_d", "team14b", "hp1", "timerA1"}

var mtrStubs = map[uint32]*mcaStub{
	0x524110: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x20004044, 20}, {2, 4, 0x20004094, 20}, {3, 4, 0x200040e4, 20}}, cnt: 0x20004040, wr: []mcaWrite{}},
	0x52beb0: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x20004138, 2}, {3, 4, 0x20004140, 2}}, cnt: 0x20004134, wr: []mcaWrite{}},
	0x523c00: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x2000414c, 3}, {2, 4, 0x20004158, 3}, {3, 4, 0x20004164, 3}, {4, 4, 0x20004170, 3}, {6, 1, 0x2000417c, 3}}, cnt: 0x20004148, wr: []mcaWrite{}},
	0x5526f0: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x20004183, 1}}, cnt: 0x2000417f, wr: []mcaWrite{}},
	0x4e4280: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x2000418b, 2}}, cnt: 0x20004187, wr: []mcaWrite{}},
	0x4e46f0: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x20004197, 2}, {2, 4, 0x2000419f, 2}}, cnt: 0x20004193, wr: []mcaWrite{}},
	0x5049c0: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{2, 4, 0x200041ab, 2}, {3, 4, 0x200041b3, 2}}, cnt: 0x200041a7, wr: []mcaWrite{{1, 0x2ab, 1, 0x20004018, 2}}},
	0x434b60: {ret: 0x20004008, retN: 2, rec: [][4]uint32{}, cnt: 0x200041bb, wr: []mcaWrite{}},
	0x462e60: {ret: 0x20004010, retN: 0, rec: [][4]uint32{}, cnt: 0x200041bf, wr: []mcaWrite{}},
	0x4be940: {ret: 0x20004000, retN: 2, rec: [][4]uint32{}, cnt: 0x200041c3, wr: []mcaWrite{}},
	0x510cc0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200041cb, 5}, {3, 4, 0x200041df, 5}}, cnt: 0x200041c7, wr: []mcaWrite{}},
	0x4acd80: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200041f7, 40}}, cnt: 0x200041f3, wr: []mcaWrite{}},
	0x590b40: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000429b, 2}, {2, 4, 0x200042a3, 2}, {3, 4, 0x200042ab, 2}}, cnt: 0x20004297, wr: []mcaWrite{}},
	0x58aad0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200042b7, 4}, {2, 4, 0x200042c7, 4}, {3, 4, 0x200042d7, 4}, {4, 4, 0x200042e7, 4}, {5, 4, 0x200042f7, 4}}, cnt: 0x200042b3, wr: []mcaWrite{}},
	0x48f650: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000430b, 2}}, cnt: 0x20004307, wr: []mcaWrite{}},
	0x4e4db0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20004317, 4}, {3, 4, 0x20004327, 4}}, cnt: 0x20004313, wr: []mcaWrite{}},
	0x4e3af0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000433b, 2}}, cnt: 0x20004337, wr: []mcaWrite{}},
	0x4c5a00: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20004347, 2}}, cnt: 0x20004343, wr: []mcaWrite{}},
}
