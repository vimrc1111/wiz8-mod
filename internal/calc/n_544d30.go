package calc

import (
	"math"
	"math/big"
)

// missileEffectResolutionNode is 0x544d30 missile_effect_resolution(projectile P).
//
// 0x53bea0(P+0x22c, 0) checks the source record (its assert panics). The spell
// record is [0x65be1c] + 0x501a60(u32 P+0x1d8)*0x1bf: 0x501a60 maps type t to a
// record index through the selector bytes at 0x501b44 (t-1, 0..0x21) and the
// jump table at 0x501af4, whose handlers are "mov eax, imm32; ret"; t-1 above
// 0x21 (and the 0x501aef handler) gives 0. Both tables and the handler
// immediates are read from the exe at run time. A non-empty sound name at the
// record's +0x14b goes to 0x517a70("Data\Missiles\Sounds\%s.wav" 0x61dc0c,
// name, 0) and its return to 0x408860(path, 0) (both 排除 address stubs).
//
// Radius r = f32 P+0x1fc compared with 0.0f ([0x5ebb34]): r > 0 (ordered) is
// the area path, otherwise (r <= 0 or NaN) the single path.
//   - Area, target P+0x260 of type 3: the target monster (0x4e5550 assert 0x1460,
//     0x4e5620) gives centre c = its actor +0x118..+0x120 (0x4534c0, integer
//     copy). 0x52ba80(&c, r bits, P+0x200, P+0x22c, P+0x280) is called, the
//     notice byte 0x68510c replaces the low byte of the frame's target pointer
//     and of its notice local, the query (0x536170: {0,-1,-1,-1,0,0,0,0}) gets
//     +0 = 3. For every monster index i below the list count of [0x683fad]
//     (re-read each pass), an active record (+0x14) gets its visual position
//     (0x4c5730/0x4b8890: actor +0x1d8 -> +4..+0xc via fld/fstp, signalling NaNs
//     quieted), offsets d = c - p evaluated in 64-bit-mantissa x87 arithmetic and
//     stored as f32, and length sqrt((dx*dx + dy*dy) + dz*dz) (0x421700, 64-bit
//     mantissa). Length <= r or unordered is a hit: query +8 = monster id and
//     0x543270(source, &query, P+0x1fc, notice local, target slot, 0).
//   - Area, other target types: 0x52a820(P+0x200, P+0x280, 1), the same frame
//     writes, query +0 = 1 and for each party slot k = 0..7 with a non-zero
//     byte at [0x685178] + 0x106k: query +4 = k, 0x543270 as above.
//   - Single: q = 0x551a20(P+0x1fc); q != 0 runs 0x552340(&q, P+0x260, realm =
//     record +0x133, power = P+0x214): 0x552250 reduces q, and a reduced 0 with
//     the notice byte prints text 0x1b3 (type 3: 0x590b40(monster, text) after
//     0x4e5550 assert 0xeae, else 0x590950(index, text)). A remaining q goes to
//     0x52bb60(monster, q, source, 0, notice, 0, P+0x280, 0) natively for type 3
//     (0x4e5550 assert 0x147f), else to 0x52a890(index, q, 0, notice, 0,
//     P+0x280, 0). Finally 0x543270(source, P+0x260, P+0x1fc, notice, notice,
//     P+0x280).
//
// Address stubs (re/emu/cases/00544d30.json): D4 0x551a20, 0x552250, 0x52a890,
// 0x52a820, 0x52ba80, 0x543270, 0x52beb0, 0x4e4280, 0x4e46f0, 0x523c00; 排除
// 0x517a70, 0x408860, 0x590b40, 0x58aad0, 0x58b410, 0x48f650, 0x4acd80,
// 0x4e4db0, 0x4e3af0, 0x5182e0, 0x4c6c30. Outputs are their call counts and
// every call's recorded arguments, the monsters, party credit, result list and
// the frame's locals left on the stack. The node reads back only 0x551a20's
// return and 0x552250's write: Emu replays them from the case, F runs
// effectQuantityNode (P+0x200 dice, P+0x220 bonus) and
// resistanceQuantityReductionNode (27-input form). 0x543270's fourth argument
// is compared as its low byte (the upper bytes are a leftover eax / stack
// local).
//
// Fixture (setup, fixed): 123 spell records (realm k%6, sound name on odd k),
// two monsters (ids 100, 101) in the list, an empty reserve list, the
// 0x54e950 fixture for messages, templates, party and UI (sdmMessages).
// Caps (panic past them): monster ids 100..101, party indices 0..7, eight
// 0x543270 calls, record indices below 123, the stub record counts.
func missileEffectResolutionNode(in []int64, r Rand) []int64 {
	return mcaRun(&merSpec, true, merBody)(in, r)
}

func merEmu(in []int64, r Rand) []int64 { return mcaRun(&merSpec, false, merBody)(in, r) }

func init() {
	// sdmCall/sdmReplay look stubs up in sdmStubs; this node's own stubs live at
	// addresses 0x54e950 never calls.
	for a, s := range merStubs {
		sdmStubs[a] = s
	}
	Nodes["00544d30"] = Node{Addr: "00544d30", In: merIn, Out: merOut, F: missileEffectResolutionNode, Emu: merEmu}
}

const (
	merProj    = 0x20030000 // the case's argument (object proj)
	merRecords = 123        // fixture spell records (largest 0x501a60 index is 0x7a)
	merFrame   = 0x10007f94 // 0x544d30's esp after its prologue (emulator entry esp 0x10008000)
	merQuery   = merFrame + 0x4c
	// 0x590950 passes its stack prefix buffer to 0x58aad0; measured on 0x544d30 ->
	// 0x552340 -> 0x590950 (emulator stack).
	merPrefix = 0x10005f68
)

// merRecordIndex is 0x501a60(t).
func merRecordIndex(w *mcaWorld, t uint32) uint32 {
	if t-1 > 0x21 {
		return 0
	}
	h := w.u32(0x501af4 + 4*uint32(w.u8(0x501b44+t-1)))
	if h == 0x501aef { // xor eax, eax; ret
		return 0
	}
	if w.u8(h) != 0xb8 || w.u8(h+5) != 0xc3 {
		panic("missile effect resolution: 0x501a60 handler is not mov eax, imm32; ret")
	}
	return w.u32(h + 1)
}

func merNaN(b uint32) bool { return b&0x7f800000 == 0x7f800000 && b&0x7fffff != 0 }
func merQuiet(b uint32) uint32 { // fld + fstp of an f32
	if merNaN(b) {
		return b | 0x400000
	}
	return b
}

// merSub is fld a; fsub b; fstp dword with FPCW 0x37f.
func merSub(a, b uint32) uint32 {
	switch an, bn := merNaN(a), merNaN(b); {
	case an && bn:
		qa, qb := merQuiet(a), merQuiet(b)
		switch ma, mb := qa&0x7fffff, qb&0x7fffff; {
		case ma > mb:
			return qa
		case mb > ma:
			return qb
		}
		return qa &^ 0x80000000 // equal significands: the positive one
	case an:
		return merQuiet(a)
	case bn:
		return merQuiet(b)
	}
	x, y := math.Float32frombits(a), math.Float32frombits(b)
	if math.IsInf(float64(x), 0) && math.IsInf(float64(y), 0) && x == y {
		return 0xffc00000 // invalid: the default NaN
	}
	if x == 0 && y == 0 {
		return (a &^ b) & 0x80000000 // -0 - +0 is -0, other zero pairs +0
	}
	d := new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Sub(big.NewFloat(float64(x)), big.NewFloat(float64(y)))
	if d.Sign() == 0 {
		return 0
	}
	f, _ := d.Float32()
	return math.Float32bits(f)
}

// merWithin is 0x421700's length compared with fcomp r: hit unless length > r.
func merWithin(d [3]uint32, r float32) bool {
	inf := false
	for _, b := range d {
		if merNaN(b) {
			return true
		}
		inf = inf || math.IsInf(float64(math.Float32frombits(b)), 0)
	}
	if inf {
		return math.IsInf(float64(r), 1)
	}
	sq := func(b uint32) *big.Float {
		x := new(big.Float).SetPrec(64).SetFloat64(float64(math.Float32frombits(b)))
		return x.Mul(x, x) // exact: 48 significant bits
	}
	s := new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Add(sq(d[0]), sq(d[1]))
	s.Add(s, sq(d[2]))
	l := new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Sqrt(s)
	return l.Cmp(big.NewFloat(float64(r))) <= 0
}

func merMonster(w *mcaWorld, id uint32) uint32 {
	if id < 100 || id > 101 {
		panic("missile effect resolution: monster id outside 100..101 (fixture cap)")
	}
	return w.findMonster(id)
}

func merAttack(w *mcaWorld, args ...uint32) {
	if w.u32(sdmStubs[0x543270].cnt) >= 8 {
		panic("missile effect resolution: more than eight 0x543270 calls (fixture cap)")
	}
	sdmStub(w, 0x543270, args...)
}

func merBody(w *mcaWorld) {
	const p, src, tgt, res, rad = merProj, merProj + 0x22c, merProj + 0x260, merProj + 0x280, merProj + 0x1fc
	w.put(merFrame+0x70, 4, src)
	w.put(merFrame+0x14, 4, tgt)
	sdmSource(w, src, false)
	idx := merRecordIndex(w, w.u32(p+0x1d8))
	if idx >= merRecords {
		panic("missile effect resolution: spell record index past the fixture records")
	}
	rec := w.u32(0x65be1c) + idx*0x1bf
	if w.u8(rec+0x14b) != 0 {
		path := sdmStub(w, 0x517a70, 0x61dc0c, rec+0x14b, 0)
		sdmStub(w, 0x408860, path, 0)
	}
	r := math.Float32frombits(w.u32(rad))
	if r > 0 {
		notice := uint32(w.u8(0x68510c))
		mon := w.u32(tgt) == 3
		if mon {
			m := merMonster(w, w.u32(tgt+8))
			for a := uint32(0); a < 3; a++ {
				w.put(merFrame+0x28+4*a, 4, uint64(w.u32(w.u32(m+0xc)+0x118+4*a)))
			}
			sdmStub(w, 0x52ba80, merFrame+0x28, w.u32(rad), p+0x200, src, res)
		} else {
			sdmStub(w, 0x52a820, p+0x200, res, 1)
		}
		w.put(merFrame+0x14, 1, uint64(notice))
		w.put(merFrame+0x18, 1, uint64(notice))
		if mon {
			for a := uint32(0); a < 3; a++ {
				w.put(merFrame+0x34+4*a, 4, uint64(w.u32(merFrame+0x28+4*a)))
			}
		}
		for k, v := range []uint32{0, 0xffffffff, 0xffffffff, 0xffffffff, 0, 0, 0, 0} {
			w.put(merQuery+4*uint32(k), 4, uint64(v))
		}
		slot, local := w.u32(merFrame+0x14), w.u32(merFrame+0x18)
		if !mon {
			w.put(merQuery, 4, 1)
			for k := uint32(0); k < 8; k++ {
				if w.u8(w.u32(0x685178)+0x106*k) != 0 {
					w.put(merQuery+4, 4, uint64(k))
					merAttack(w, src, merQuery, rad, local, slot, 0)
				}
			}
			return
		}
		w.put(merQuery, 4, 3)
		w.put(merFrame+0x70, 4, 0)
		for i := uint32(0); i < w.listCount(w.u32(0x683fad)); i++ {
			w.put(merFrame+0x70, 4, uint64(i))
			m := w.monsterRecord(i)
			if w.u8(m+0x14) != 0 {
				vis := w.u32(w.u32(m+0xc) + 0x1d8)
				var d [3]uint32
				for a := uint32(0); a < 3; a++ {
					pos := merQuiet(w.u32(vis + 4 + 4*a))
					w.put(merFrame+0x1c+4*a, 4, uint64(pos))
					d[a] = merSub(w.u32(merFrame+0x34+4*a), pos)
					w.put(merFrame+0x40+4*a, 4, uint64(d[a]))
					w.put(merFrame+0x28+4*a, 4, uint64(d[a]))
				}
				if merWithin(d, r) {
					w.put(merQuery+8, 4, uint64(w.u32(m)))
					merAttack(w, src, merQuery, rad, local, slot, 0)
				}
			}
			w.put(merFrame+0x70, 4, uint64(i+1))
		}
		return
	}
	notice := w.u8(0x68510c)
	q := merQuantity(w)
	w.put(merFrame+0x10, 4, uint64(q))
	if q != 0 {
		merReduce(w, rec)
		q = w.u32(merFrame + 0x10)
		if q != 0 {
			if w.u32(tgt) == 3 {
				sdmHit(w, merMonster(w, w.u32(tgt+8)), q, src, notice != 0, false, res)
			} else {
				sdmStub(w, 0x52a890, w.u32(tgt+4), q, 0, uint32(notice), 0, res, 0)
			}
		}
	}
	notice = w.u8(0x68510c)
	merAttack(w, src, tgt, rad, uint32(notice), uint32(notice), res)
}

// merQuantity is 0x551a20(P+0x1fc).
func merQuantity(w *mcaWorld) uint32 {
	const p = merProj
	k := sdmCall(w, 0x551a20, p+0x1fc)
	if !w.real {
		return sdmReplay(w, 0x551a20, k)
	}
	in := []int64{int64(int16(w.u16(p + 0x200))), int64(w.u8(p + 0x202)), int64(w.u8(p + 0x203)), int64(int32(w.u32(p + 0x220)))}
	return uint32(effectQuantityNode(in, w.r)[0])
}

// merReduce is 0x552340(&q, P+0x260, realm, power) with q at frame +0x10.
func merReduce(w *mcaWorld, rec uint32) {
	const tgt, q = merProj + 0x260, merFrame + 0x10
	realm, power, typ := w.u32(rec+0x133), w.u32(merProj+0x214), w.u32(tgt)
	k := sdmCall(w, 0x552250, q, tgt, realm, power)
	if !w.real {
		sdmReplay(w, 0x552250, k, q, tgt, realm, power)
	} else {
		in := make([]int64, 27)
		in[0], in[3], in[4], in[5] = int64(w.u32(q)), int64(power), int64(typ), int64(realm)
		if typ == 3 {
			m := merMonster(w, w.u32(tgt+8))
			t := w.template(m)
			for j := uint32(0); j < 6; j++ {
				in[12+j] = int64(w.u8(t + 0x176 + j))
				in[19+j] = int64(int8(w.u8(m + 0x217 + j)))
			}
			in[18], in[25], in[26] = int64(w.u8(t+0x24f)), int64(w.u32(m)), int64(w.u32(m+8))
		} else {
			idx := w.u32(tgt + 4)
			c := w.charBase(idx)
			in[1], in[2], in[6] = int64(w.u32(c+0xee1)), int64(w.u32(c+0x89)), int64(idx)
			for j := uint32(1); j < 6; j++ {
				in[6+j] = int64(w.u32(c + 0xee1 + 16*j))
			}
		}
		w.put(q, 4, uint64(uint32(resistanceQuantityReductionNode(in, w.r)[0])))
	}
	if w.u32(q) != 0 || w.u8(0x68510c) == 0 {
		return
	}
	if typ == 3 {
		h, ok := w.monsterHandle(w.u32(tgt + 8))
		if !ok {
			panic("missile effect resolution: monster id not found (0x4e5550 asserts)")
		}
		sdmStub(w, 0x590b40, merMonster(w, w.u32(tgt+8)), sdmMsg(w, 0x1b3), h)
		return
	}
	sdmNotice(w, w.u32(tgt+4), merPrefix, sdmMsg(w, 0x1b3))
}

// Code below is generated from the same description as re/emu/cases/00544d30.json.

var merSpec = sdmMessages(mstSpec{
	set: []mstSet{
		{0x65be1c, 4, 0x20020000},
		{0x685174, 4, 0x20010000},
		{0x685178, 4, 0x20009000},
		{0x68c09c, 4, 0x20002000},
		{0x683fad, 4, 0x20003600},
		{0x683fb1, 4, 0x20003540},
		{0x683fb9, 4, 0x0},
		{0x683fbd, 4, 0x0},
		{0x6840c7, 4, 0x20005000},
		{0x6840cb, 4, 0x20005400},
		{0x68ec78, 4, 0x0},
		{0x68edcc, 4, 0x0},
		{0x689b34, 2, 0x0},
		{0x683f97, 1, 0x0},
		{0x683f9b, 1, 0x0},
		{0x6598b8, 4, 0x20008200},
		{0x20008200, 4, 0x5ec078},
		{0x2003022c, 4, 0x10101010},
		{0x20030230, 4, 0x11111111},
		{0x20030234, 4, 0x12121212},
		{0x20030238, 4, 0x13131313},
		{0x2003023c, 4, 0x14141414},
		{0x20030240, 4, 0x15151515},
		{0x20030244, 4, 0x16161616},
		{0x20030248, 4, 0x17171717},
		{0x2003024c, 4, 0x18181818},
		{0x20030250, 4, 0x19191919},
		{0x20030254, 4, 0x1a1a1a1a},
		{0x20030258, 4, 0x1b1b1b1b},
		{0x2003025c, 4, 0x1c1c1c1c},
		{0x200302e4, 4, 0x20003300},
		{0x20003300, 4, 0x20003100},
		{0x20003304, 4, 0x20003180},
		{0x20003308, 4, 0x20003200},
		{0x20020133, 4, 0x0},
		{0x200202f2, 4, 0x1},
		{0x2002030a, 1, 0x62},
		{0x200204b1, 4, 0x2},
		{0x20020670, 4, 0x3},
		{0x20020688, 1, 0x64},
		{0x2002082f, 4, 0x4},
		{0x200209ee, 4, 0x5},
		{0x20020a06, 1, 0x66},
		{0x20020bad, 4, 0x0},
		{0x20020d6c, 4, 0x1},
		{0x20020d84, 1, 0x68},
		{0x20020f2b, 4, 0x2},
		{0x200210ea, 4, 0x3},
		{0x20021102, 1, 0x6a},
		{0x200212a9, 4, 0x4},
		{0x20021468, 4, 0x5},
		{0x20021480, 1, 0x6c},
		{0x20021627, 4, 0x0},
		{0x200217e6, 4, 0x1},
		{0x200217fe, 1, 0x6e},
		{0x200219a5, 4, 0x2},
		{0x20021b64, 4, 0x3},
		{0x20021b7c, 1, 0x70},
		{0x20021d23, 4, 0x4},
		{0x20021ee2, 4, 0x5},
		{0x20021efa, 1, 0x72},
		{0x200220a1, 4, 0x0},
		{0x20022260, 4, 0x1},
		{0x20022278, 1, 0x74},
		{0x2002241f, 4, 0x2},
		{0x200225de, 4, 0x3},
		{0x200225f6, 1, 0x76},
		{0x2002279d, 4, 0x4},
		{0x2002295c, 4, 0x5},
		{0x20022974, 1, 0x78},
		{0x20022b1b, 4, 0x0},
		{0x20022cda, 4, 0x1},
		{0x20022cf2, 1, 0x7a},
		{0x20022e99, 4, 0x2},
		{0x20023058, 4, 0x3},
		{0x20023070, 1, 0x62},
		{0x20023217, 4, 0x4},
		{0x200233d6, 4, 0x5},
		{0x200233ee, 1, 0x64},
		{0x20023595, 4, 0x0},
		{0x20023754, 4, 0x1},
		{0x2002376c, 1, 0x66},
		{0x20023913, 4, 0x2},
		{0x20023ad2, 4, 0x3},
		{0x20023aea, 1, 0x68},
		{0x20023c91, 4, 0x4},
		{0x20023e50, 4, 0x5},
		{0x20023e68, 1, 0x6a},
		{0x2002400f, 4, 0x0},
		{0x200241ce, 4, 0x1},
		{0x200241e6, 1, 0x6c},
		{0x2002438d, 4, 0x2},
		{0x2002454c, 4, 0x3},
		{0x20024564, 1, 0x6e},
		{0x2002470b, 4, 0x4},
		{0x200248ca, 4, 0x5},
		{0x200248e2, 1, 0x70},
		{0x20024a89, 4, 0x0},
		{0x20024c48, 4, 0x1},
		{0x20024c60, 1, 0x72},
		{0x20024e07, 4, 0x2},
		{0x20024fc6, 4, 0x3},
		{0x20024fde, 1, 0x74},
		{0x20025185, 4, 0x4},
		{0x20025344, 4, 0x5},
		{0x2002535c, 1, 0x76},
		{0x20025503, 4, 0x0},
		{0x200256c2, 4, 0x1},
		{0x200256da, 1, 0x78},
		{0x20025881, 4, 0x2},
		{0x20025a40, 4, 0x3},
		{0x20025a58, 1, 0x7a},
		{0x20025bff, 4, 0x4},
		{0x20025dbe, 4, 0x5},
		{0x20025dd6, 1, 0x62},
		{0x20025f7d, 4, 0x0},
		{0x2002613c, 4, 0x1},
		{0x20026154, 1, 0x64},
		{0x200262fb, 4, 0x2},
		{0x200264ba, 4, 0x3},
		{0x200264d2, 1, 0x66},
		{0x20026679, 4, 0x4},
		{0x20026838, 4, 0x5},
		{0x20026850, 1, 0x68},
		{0x200269f7, 4, 0x0},
		{0x20026bb6, 4, 0x1},
		{0x20026bce, 1, 0x6a},
		{0x20026d75, 4, 0x2},
		{0x20026f34, 4, 0x3},
		{0x20026f4c, 1, 0x6c},
		{0x200270f3, 4, 0x4},
		{0x200272b2, 4, 0x5},
		{0x200272ca, 1, 0x6e},
		{0x20027471, 4, 0x0},
		{0x20027630, 4, 0x1},
		{0x20027648, 1, 0x70},
		{0x200277ef, 4, 0x2},
		{0x200279ae, 4, 0x3},
		{0x200279c6, 1, 0x72},
		{0x20027b6d, 4, 0x4},
		{0x20027d2c, 4, 0x5},
		{0x20027d44, 1, 0x74},
		{0x20027eeb, 4, 0x0},
		{0x200280aa, 4, 0x1},
		{0x200280c2, 1, 0x76},
		{0x20028269, 4, 0x2},
		{0x20028428, 4, 0x3},
		{0x20028440, 1, 0x78},
		{0x200285e7, 4, 0x4},
		{0x200287a6, 4, 0x5},
		{0x200287be, 1, 0x7a},
		{0x20028965, 4, 0x0},
		{0x20028b24, 4, 0x1},
		{0x20028b3c, 1, 0x62},
		{0x20028ce3, 4, 0x2},
		{0x20028ea2, 4, 0x3},
		{0x20028eba, 1, 0x64},
		{0x20029061, 4, 0x4},
		{0x20029220, 4, 0x5},
		{0x20029238, 1, 0x66},
		{0x200293df, 4, 0x0},
		{0x2002959e, 4, 0x1},
		{0x200295b6, 1, 0x68},
		{0x2002975d, 4, 0x2},
		{0x2002991c, 4, 0x3},
		{0x20029934, 1, 0x6a},
		{0x20029adb, 4, 0x4},
		{0x20029c9a, 4, 0x5},
		{0x20029cb2, 1, 0x6c},
		{0x20029e59, 4, 0x0},
		{0x2002a018, 4, 0x1},
		{0x2002a030, 1, 0x6e},
		{0x2002a1d7, 4, 0x2},
		{0x2002a396, 4, 0x3},
		{0x2002a3ae, 1, 0x70},
		{0x2002a555, 4, 0x4},
		{0x2002a714, 4, 0x5},
		{0x2002a72c, 1, 0x72},
		{0x2002a8d3, 4, 0x0},
		{0x2002aa92, 4, 0x1},
		{0x2002aaaa, 1, 0x74},
		{0x2002ac51, 4, 0x2},
		{0x2002ae10, 4, 0x3},
		{0x2002ae28, 1, 0x76},
		{0x2002afcf, 4, 0x4},
		{0x2002b18e, 4, 0x5},
		{0x2002b1a6, 1, 0x78},
		{0x2002b34d, 4, 0x0},
		{0x2002b50c, 4, 0x1},
		{0x2002b524, 1, 0x7a},
		{0x2002b6cb, 4, 0x2},
		{0x2002b88a, 4, 0x3},
		{0x2002b8a2, 1, 0x62},
		{0x2002ba49, 4, 0x4},
		{0x2002bc08, 4, 0x5},
		{0x2002bc20, 1, 0x64},
		{0x2002bdc7, 4, 0x0},
		{0x2002bf86, 4, 0x1},
		{0x2002bf9e, 1, 0x66},
		{0x2002c145, 4, 0x2},
		{0x2002c304, 4, 0x3},
		{0x2002c31c, 1, 0x68},
		{0x2002c4c3, 4, 0x4},
		{0x2002c682, 4, 0x5},
		{0x2002c69a, 1, 0x6a},
		{0x2002c841, 4, 0x0},
		{0x2002ca00, 4, 0x1},
		{0x2002ca18, 1, 0x6c},
		{0x2002cbbf, 4, 0x2},
		{0x2002cd7e, 4, 0x3},
		{0x2002cd96, 1, 0x6e},
		{0x2002cf3d, 4, 0x4},
		{0x2002d0fc, 4, 0x5},
		{0x2002d114, 1, 0x70},
		{0x2002d2bb, 4, 0x0},
		{0x2002d47a, 4, 0x1},
		{0x2002d492, 1, 0x72},
		{0x2002d639, 4, 0x2},
		{0x20002648, 4, 0x20003060},
		{0x2000264c, 4, 0x20003064},
		{0x200026cc, 4, 0x20003000},
		{0x20002958, 4, 0x20003068},
		{0x20010005, 2, 0x43},
		{0x20010007, 2, 0x0},
		{0x200090f1, 1, 0xa0},
		{0x20011867, 2, 0x43},
		{0x20011869, 2, 0x43},
		{0x2001186b, 2, 0x0},
		{0x200091f7, 1, 0xa1},
		{0x200130c9, 2, 0x43},
		{0x200130cb, 2, 0x43},
		{0x200130cd, 2, 0x43},
		{0x200130cf, 2, 0x0},
		{0x200092fd, 1, 0xa2},
		{0x2001492b, 2, 0x43},
		{0x2001492d, 2, 0x43},
		{0x2001492f, 2, 0x43},
		{0x20014931, 2, 0x43},
		{0x20014933, 2, 0x0},
		{0x20009403, 1, 0xa3},
		{0x2001618d, 2, 0x43},
		{0x2001618f, 2, 0x43},
		{0x20016191, 2, 0x43},
		{0x20016193, 2, 0x43},
		{0x20016195, 2, 0x43},
		{0x20016197, 2, 0x0},
		{0x20009509, 1, 0xa4},
		{0x200179ef, 2, 0x43},
		{0x200179f1, 2, 0x43},
		{0x200179f3, 2, 0x43},
		{0x200179f5, 2, 0x43},
		{0x200179f7, 2, 0x43},
		{0x200179f9, 2, 0x43},
		{0x200179fb, 2, 0x0},
		{0x2000960f, 1, 0xa5},
		{0x20019251, 2, 0x43},
		{0x20019253, 2, 0x43},
		{0x20019255, 2, 0x43},
		{0x20019257, 2, 0x43},
		{0x20019259, 2, 0x43},
		{0x2001925b, 2, 0x43},
		{0x2001925d, 2, 0x43},
		{0x2001925f, 2, 0x0},
		{0x20009715, 1, 0xa6},
		{0x2001aab3, 2, 0x43},
		{0x2001aab5, 2, 0x43},
		{0x2001aab7, 2, 0x43},
		{0x2001aab9, 2, 0x43},
		{0x2001aabb, 2, 0x43},
		{0x2001aabd, 2, 0x43},
		{0x2001aabf, 2, 0x43},
		{0x2001aac1, 2, 0x43},
		{0x2001aac3, 2, 0x0},
		{0x2000981b, 1, 0xa7},
		{0x20003600, 4, 0x20003620},
		{0x20003608, 4, 0x2},
		{0x20003540, 4, 0x20003560},
		{0x20003548, 4, 0x1},
		{0x20003560, 4, 0x20003500},
		{0x20003500, 4, 0x1c8},
		{0x20003508, 4, 0x20003580},
		{0x20008000, 4, 0x20008040},
		{0x20008058, 4, 0x20008100},
		{0x20008100, 1, 0x4},
		{0x20003620, 4, 0x20006000},
		{0x20006000, 4, 0x64},
		{0x20006008, 4, 0x0},
		{0x2000600c, 4, 0x20006800},
		{0x20006010, 4, 0x20003400},
		{0x200060bf, 4, 0x5150},
		{0x200060c3, 2, 0x6162},
		{0x20006800, 4, 0x5ed22c},
		{0x200069d8, 4, 0x20007000},
		{0x200070a7, 1, 0x14},
		{0x200075e8, 4, 0x20008300},
		{0x20007208, 4, 0x20008000},
		{0x20005000, 2, 0x54},
		{0x20005002, 2, 0x30},
		{0x20005004, 2, 0x70},
		{0x20005006, 2, 0x6c},
		{0x20005008, 2, 0x75},
		{0x2000500a, 2, 0x72},
		{0x2000500c, 2, 0x61},
		{0x2000500e, 2, 0x6c},
		{0x20005010, 2, 0x0},
		{0x20005060, 2, 0x54},
		{0x20005062, 2, 0x30},
		{0x20005064, 2, 0x0},
		{0x20003624, 4, 0x20006400},
		{0x20006400, 4, 0x65},
		{0x20006408, 4, 0x1},
		{0x2000640c, 4, 0x20006a00},
		{0x20006410, 4, 0x20003400},
		{0x200064bf, 4, 0x5151},
		{0x200064c3, 2, 0x6163},
		{0x20006a00, 4, 0x5ed22c},
		{0x20006bd8, 4, 0x20007800},
		{0x200078a7, 1, 0x14},
		{0x20007de8, 4, 0x20008300},
		{0x20007a08, 4, 0x20008000},
		{0x20005400, 2, 0x54},
		{0x20005402, 2, 0x31},
		{0x20005404, 2, 0x70},
		{0x20005406, 2, 0x6c},
		{0x20005408, 2, 0x75},
		{0x2000540a, 2, 0x72},
		{0x2000540c, 2, 0x61},
		{0x2000540e, 2, 0x6c},
		{0x20005410, 2, 0x0},
		{0x20005460, 2, 0x54},
		{0x20005462, 2, 0x31},
		{0x20005464, 2, 0x0},
		{0x2000341a, 4, 0x3e8},
		{0x2000341e, 4, 0x7d0},
		{0x20003422, 4, 0xbb8},
		{0x20003426, 4, 0xfa0},
		{0x2000342a, 4, 0x1388},
		{0x2000342e, 4, 0x1770},
		{0x20003432, 4, 0x1b58},
		{0x20003436, 4, 0x1f40},
		{0x20003004, 2, 0x0},
		{0x20031004, 4, 0x68bfd0},
	},
	in: []mstFld{
		{0x68510c, 1, false, -1, 0, 0},
		{0x683f94, 1, false, -1, 0, 0},
		{0x6875ef, 4, false, -1, 0, 0},
		{0x200301d8, 4, false, -1, 0, 0},
		{0x200301fc, 4, false, -1, 0, 0},
		{0x20030200, 2, true, -1, 0, 0},
		{0x20030202, 1, false, -1, 0, 0},
		{0x20030203, 1, false, -1, 0, 0},
		{0x20030214, 4, false, -1, 0, 0},
		{0x20030220, 4, true, -1, 0, 0},
		{0x2003022c, 4, false, -1, 0, 0},
		{0x20030230, 4, true, -1, 0, 0},
		{0x20030247, 1, false, -1, 0, 0},
		{0x20030248, 1, false, -1, 0, 0},
		{0x2003024a, 1, false, -1, 0, 0},
		{0x20030260, 4, false, -1, 0, 0},
		{0x20030264, 4, false, -1, 0, 0},
		{0x20030268, 4, false, -1, 0, 0},
		{0x20030280, 4, false, -1, 0, 0},
		{0x20030284, 4, false, -1, 0, 0},
		{0x200302cc, 4, false, -1, 0, 0},
		{0x200302dc, 4, true, -1, 0, 0},
		{0x200302e0, 4, true, -1, 0, 0},
		{0x20003000, 2, false, -1, 0, 0},
		{0x20003002, 2, false, -1, 0, 0},
		{0x20031000, 4, false, -1, 0, 0},
		{0x2000b018, 4, false, -1, 0, 0},
		{0x2000b030, 4, false, -1, 0, 0},
		{0x20009000, 1, false, -1, 0, 0},
		{0x20009106, 1, false, -1, 0, 0},
		{0x2000920c, 1, false, -1, 0, 0},
		{0x20009312, 1, false, -1, 0, 0},
		{0x20009418, 1, false, -1, 0, 0},
		{0x2000951e, 1, false, -1, 0, 0},
		{0x20009624, 1, false, -1, 0, 0},
		{0x2000972a, 1, false, -1, 0, 0},
		{0x20006014, 1, false, -1, 0, 0},
		{0x20007004, 4, false, -1, 0, 0},
		{0x20007008, 4, false, -1, 0, 0},
		{0x2000700c, 4, false, -1, 0, 0},
		{0x20006918, 4, false, -1, 0, 0},
		{0x2000691c, 4, false, -1, 0, 0},
		{0x20006920, 4, false, -1, 0, 0},
		{0x2000602b, 4, false, -1, 0, 0},
		{0x20006027, 4, false, -1, 0, 0},
		{0x200060c5, 2, false, -1, 0, 0},
		{0x200060c7, 4, false, -1, 0, 0},
		{0x2000602f, 4, false, -1, 0, 0},
		{0x20006033, 4, false, -1, 0, 0},
		{0x2000609b, 4, false, -1, 0, 0},
		{0x20006015, 1, false, -1, 0, 0},
		{0x200062ab, 1, false, -1, 0, 0},
		{0x20006004, 4, false, -1, 0, 0},
		{0x200069dc, 4, false, -1, 0, 0},
		{0x200070a4, 1, false, -1, 0, 0},
		{0x20005187, 2, false, -1, 0, 0},
		{0x20006217, 1, true, -1, 0, 0},
		{0x20006218, 1, true, -1, 0, 0},
		{0x20006219, 1, true, -1, 0, 0},
		{0x2000621a, 1, true, -1, 0, 0},
		{0x2000621b, 1, true, -1, 0, 0},
		{0x2000621c, 1, true, -1, 0, 0},
		{0x20005176, 1, false, -1, 0, 0},
		{0x20005177, 1, false, -1, 0, 0},
		{0x20005178, 1, false, -1, 0, 0},
		{0x20005179, 1, false, -1, 0, 0},
		{0x2000517a, 1, false, -1, 0, 0},
		{0x2000517b, 1, false, -1, 0, 0},
		{0x2000524f, 1, false, -1, 0, 0},
		{0x20006414, 1, false, -1, 0, 0},
		{0x20007804, 4, false, -1, 0, 0},
		{0x20007808, 4, false, -1, 0, 0},
		{0x2000780c, 4, false, -1, 0, 0},
		{0x20006b18, 4, false, -1, 0, 0},
		{0x20006b1c, 4, false, -1, 0, 0},
		{0x20006b20, 4, false, -1, 0, 0},
		{0x2000642b, 4, false, -1, 0, 0},
		{0x20006427, 4, false, -1, 0, 0},
		{0x200064c5, 2, false, -1, 0, 0},
		{0x200064c7, 4, false, -1, 0, 0},
		{0x2000642f, 4, false, -1, 0, 0},
		{0x20006433, 4, false, -1, 0, 0},
		{0x2000649b, 4, false, -1, 0, 0},
		{0x20006415, 1, false, -1, 0, 0},
		{0x200066ab, 1, false, -1, 0, 0},
		{0x20006404, 4, false, -1, 0, 0},
		{0x20006bdc, 4, false, -1, 0, 0},
		{0x200078a4, 1, false, -1, 0, 0},
		{0x20005587, 2, false, -1, 0, 0},
		{0x20006617, 1, true, -1, 0, 0},
		{0x20006618, 1, true, -1, 0, 0},
		{0x20006619, 1, true, -1, 0, 0},
		{0x2000661a, 1, true, -1, 0, 0},
		{0x2000661b, 1, true, -1, 0, 0},
		{0x2000661c, 1, true, -1, 0, 0},
		{0x20005576, 1, false, -1, 0, 0},
		{0x20005577, 1, false, -1, 0, 0},
		{0x20005578, 1, false, -1, 0, 0},
		{0x20005579, 1, false, -1, 0, 0},
		{0x2000557a, 1, false, -1, 0, 0},
		{0x2000557b, 1, false, -1, 0, 0},
		{0x2000564f, 1, false, -1, 0, 0},
		{0x2000352c, 1, false, -1, 0, 0},
		{0x20010ee1, 4, false, -1, 0, 0},
		{0x20010ef1, 4, false, -1, 0, 0},
		{0x20010f01, 4, false, -1, 0, 0},
		{0x20010f11, 4, false, -1, 0, 0},
		{0x20010f21, 4, false, -1, 0, 0},
		{0x20010f31, 4, false, -1, 0, 0},
		{0x20010089, 4, false, -1, 0, 0},
		{0x20012743, 4, false, -1, 0, 0},
		{0x20012753, 4, false, -1, 0, 0},
		{0x20012763, 4, false, -1, 0, 0},
		{0x20012773, 4, false, -1, 0, 0},
		{0x20012783, 4, false, -1, 0, 0},
		{0x20012793, 4, false, -1, 0, 0},
		{0x200118eb, 4, false, -1, 0, 0},
		{0x20013fa5, 4, false, -1, 0, 0},
		{0x20013fb5, 4, false, -1, 0, 0},
		{0x20013fc5, 4, false, -1, 0, 0},
		{0x20013fd5, 4, false, -1, 0, 0},
		{0x20013fe5, 4, false, -1, 0, 0},
		{0x20013ff5, 4, false, -1, 0, 0},
		{0x2001314d, 4, false, -1, 0, 0},
		{0x20015807, 4, false, -1, 0, 0},
		{0x20015817, 4, false, -1, 0, 0},
		{0x20015827, 4, false, -1, 0, 0},
		{0x20015837, 4, false, -1, 0, 0},
		{0x20015847, 4, false, -1, 0, 0},
		{0x20015857, 4, false, -1, 0, 0},
		{0x200149af, 4, false, -1, 0, 0},
		{0x20017069, 4, false, -1, 0, 0},
		{0x20017079, 4, false, -1, 0, 0},
		{0x20017089, 4, false, -1, 0, 0},
		{0x20017099, 4, false, -1, 0, 0},
		{0x200170a9, 4, false, -1, 0, 0},
		{0x200170b9, 4, false, -1, 0, 0},
		{0x20016211, 4, false, -1, 0, 0},
		{0x200188cb, 4, false, -1, 0, 0},
		{0x200188db, 4, false, -1, 0, 0},
		{0x200188eb, 4, false, -1, 0, 0},
		{0x200188fb, 4, false, -1, 0, 0},
		{0x2001890b, 4, false, -1, 0, 0},
		{0x2001891b, 4, false, -1, 0, 0},
		{0x20017a73, 4, false, -1, 0, 0},
		{0x2001a12d, 4, false, -1, 0, 0},
		{0x2001a13d, 4, false, -1, 0, 0},
		{0x2001a14d, 4, false, -1, 0, 0},
		{0x2001a15d, 4, false, -1, 0, 0},
		{0x2001a16d, 4, false, -1, 0, 0},
		{0x2001a17d, 4, false, -1, 0, 0},
		{0x200192d5, 4, false, -1, 0, 0},
		{0x2001b98f, 4, false, -1, 0, 0},
		{0x2001b99f, 4, false, -1, 0, 0},
		{0x2001b9af, 4, false, -1, 0, 0},
		{0x2001b9bf, 4, false, -1, 0, 0},
		{0x2001b9cf, 4, false, -1, 0, 0},
		{0x2001b9df, 4, false, -1, 0, 0},
		{0x2001ab37, 4, false, -1, 0, 0},
	},
	out: []mstFld{
		{0x2000b044, 4, false, -1, 0, 0},
		{0x2000b048, 4, false, -1, 0, 0},
		{0x2000b04c, 4, false, -1, 0, 0},
		{0x2000b050, 4, false, -1, 0, 0},
		{0x2000b054, 4, false, -1, 0, 0},
		{0x2000b058, 4, false, -1, 0, 0},
		{0x2000b05c, 4, false, -1, 0, 0},
		{0x2000b060, 4, false, -1, 0, 0},
		{0x2000b064, 4, false, -1, 0, 0},
		{0x2000b068, 4, false, -1, 0, 0},
		{0x2000b06c, 4, false, -1, 0, 0},
		{0x2000b070, 4, false, -1, 0, 0},
		{0x2000b074, 4, false, -1, 0, 0},
		{0x2000b078, 4, false, -1, 0, 0},
		{0x2000b07c, 4, false, -1, 0, 0},
		{0x2000b080, 4, false, -1, 0, 0},
		{0x2000b084, 4, false, -1, 0, 0},
		{0x2000b088, 4, false, -1, 0, 0},
		{0x2000b08c, 4, false, -1, 0, 0},
		{0x2000b090, 4, false, -1, 0, 0},
		{0x2000b094, 1, false, -1, 0, 0},
		{0x2000b095, 1, false, -1, 0, 0},
		{0x2000b096, 1, false, -1, 0, 0},
		{0x2000b0a0, 4, false, -1, 0, 0},
		{0x2000b0a4, 4, false, -1, 0, 0},
		{0x2000b0a8, 4, false, -1, 0, 0},
		{0x2000b0ac, 4, false, -1, 0, 0},
		{0x2000b0b0, 4, false, -1, 0, 0},
		{0x2000b0b4, 4, false, -1, 0, 0},
		{0x2000b0b8, 4, false, -1, 0, 0},
		{0x2000b0bc, 4, false, -1, 0, 0},
		{0x2000b0c0, 4, false, -1, 0, 0},
		{0x2000b0c4, 4, false, -1, 0, 0},
		{0x2000b0c8, 4, false, -1, 0, 0},
		{0x2000b0cc, 4, false, -1, 0, 0},
		{0x2000b0d0, 4, false, -1, 0, 0},
		{0x2000b0d4, 4, false, -1, 0, 0},
		{0x2000b0d8, 4, false, -1, 0, 0},
		{0x2000b0dc, 4, false, -1, 0, 0},
		{0x2000b0e0, 4, false, -1, 0, 0},
		{0x2000b0e4, 4, false, -1, 0, 0},
		{0x2000b0e8, 1, false, -1, 0, 0},
		{0x2000b0e9, 1, false, -1, 0, 0},
		{0x2000b0ea, 1, false, -1, 0, 0},
		{0x2000b0f4, 4, false, -1, 0, 0},
		{0x2000b0f8, 4, false, -1, 0, 0},
		{0x2000b0fc, 4, false, -1, 0, 0},
		{0x2000b100, 4, false, -1, 0, 0},
		{0x2000b104, 4, false, -1, 0, 0},
		{0x2000b108, 4, false, -1, 0, 0},
		{0x2000b10c, 4, false, -1, 0, 0},
		{0x2000b110, 4, false, -1, 0, 0},
		{0x2000b114, 4, false, -1, 0, 0},
		{0x2000b118, 4, false, -1, 0, 0},
		{0x2000b11c, 4, false, -1, 0, 0},
		{0x2000b120, 4, false, -1, 0, 0},
		{0x2000b124, 4, false, -1, 0, 0},
		{0x2000b128, 1, false, -1, 0, 0},
		{0x2000b129, 1, false, -1, 0, 0},
		{0x2000b12a, 1, false, -1, 0, 0},
		{0x2000b134, 4, false, -1, 0, 0},
		{0x2000b138, 4, false, -1, 0, 0},
		{0x2000b13c, 4, false, -1, 0, 0},
		{0x2000b140, 4, false, -1, 0, 0},
		{0x2000b144, 4, false, -1, 0, 0},
		{0x2000b148, 4, false, -1, 0, 0},
		{0x2000b14c, 4, false, -1, 0, 0},
		{0x2000b150, 4, false, -1, 0, 0},
		{0x2000b154, 4, false, -1, 0, 0},
		{0x2000b158, 4, false, -1, 0, 0},
		{0x2000b15c, 4, false, -1, 0, 0},
		{0x2000b160, 4, false, -1, 0, 0},
		{0x2000b164, 4, false, -1, 0, 0},
		{0x2000b168, 4, false, -1, 0, 0},
		{0x2000b16c, 4, false, -1, 0, 0},
		{0x2000b170, 4, false, -1, 0, 0},
		{0x2000b174, 4, false, -1, 0, 0},
		{0x2000b178, 4, false, -1, 0, 0},
		{0x2000b17c, 4, false, -1, 0, 0},
		{0x2000b180, 4, false, -1, 0, 0},
		{0x2000b184, 4, false, -1, 0, 0},
		{0x2000b188, 4, false, -1, 0, 0},
		{0x2000b18c, 4, false, -1, 0, 0},
		{0x2000b190, 4, false, -1, 0, 0},
		{0x2000b194, 4, false, -1, 0, 0},
		{0x2000b198, 4, false, -1, 0, 0},
		{0x2000b19c, 4, false, -1, 0, 0},
		{0x2000b1a0, 4, false, -1, 0, 0},
		{0x2000b1a4, 4, false, -1, 0, 0},
		{0x2000b1a8, 4, false, -1, 0, 0},
		{0x2000b1ac, 4, false, -1, 0, 0},
		{0x2000b1b0, 4, false, -1, 0, 0},
		{0x2000b1b4, 4, false, -1, 0, 0},
		{0x2000b1b8, 4, false, -1, 0, 0},
		{0x2000b1bc, 4, false, -1, 0, 0},
		{0x2000b1c0, 4, false, -1, 0, 0},
		{0x2000b1c4, 4, false, -1, 0, 0},
		{0x2000b1c8, 4, false, -1, 0, 0},
		{0x2000b1cc, 4, false, -1, 0, 0},
		{0x2000b1d0, 4, false, -1, 0, 0},
		{0x2000b1d4, 4, false, -1, 0, 0},
		{0x2000b1d8, 4, false, -1, 0, 0},
		{0x2000b1dc, 4, false, -1, 0, 0},
		{0x2000b1e0, 4, false, -1, 0, 0},
		{0x2000b1e4, 4, false, -1, 0, 0},
		{0x2000b1e8, 4, false, -1, 0, 0},
		{0x2000b1ec, 4, false, -1, 0, 0},
		{0x2000b1f0, 4, false, -1, 0, 0},
		{0x2000b1f4, 4, false, -1, 0, 0},
		{0x2000b1f8, 4, false, -1, 0, 0},
		{0x2000b1fc, 4, false, -1, 0, 0},
		{0x2000b200, 4, false, -1, 0, 0},
		{0x2000b204, 4, false, -1, 0, 0},
		{0x2000b208, 4, false, -1, 0, 0},
		{0x2000b20c, 4, false, -1, 0, 0},
		{0x2000b210, 4, false, -1, 0, 0},
		{0x2000b214, 4, false, -1, 0, 0},
		{0x2000b218, 4, false, -1, 0, 0},
		{0x2000b21c, 4, false, -1, 0, 0},
		{0x2000b220, 4, false, -1, 0, 0},
		{0x2000b224, 4, false, -1, 0, 0},
		{0x2000b228, 4, false, -1, 0, 0},
		{0x2000b22c, 4, false, -1, 0, 0},
		{0x2000b230, 4, false, -1, 0, 0},
		{0x2000b234, 4, false, -1, 0, 0},
		{0x2000b238, 4, false, -1, 0, 0},
		{0x2000b23c, 4, false, -1, 0, 0},
		{0x2000b240, 4, false, -1, 0, 0},
		{0x2000b244, 4, false, -1, 0, 0},
		{0x2000b248, 4, false, -1, 0, 0},
		{0x2000b24c, 4, false, -1, 0, 0},
		{0x2000b250, 4, false, -1, 0, 0},
		{0x2000b254, 4, false, -1, 0, 0},
		{0x2000b258, 4, false, -1, 0, 0},
		{0x2000b25c, 4, false, -1, 0, 0},
		{0x2000b260, 4, false, -1, 0, 0},
		{0x2000b264, 4, false, -1, 0, 0},
		{0x2000b268, 4, false, -1, 0, 0},
		{0x2000b26c, 4, false, -1, 0, 0},
		{0x2000b270, 4, false, -1, 0, 0},
		{0x2000b274, 4, false, -1, 0, 0},
		{0x2000b278, 4, false, -1, 0, 0},
		{0x2000b27c, 4, false, -1, 0, 0},
		{0x2000b280, 4, false, -1, 0, 0},
		{0x2000b284, 4, false, -1, 0, 0},
		{0x2000b288, 4, false, -1, 0, 0},
		{0x2000b28c, 4, false, -1, 0, 0},
		{0x2000b290, 4, false, -1, 0, 0},
		{0x2000b294, 4, false, -1, 0, 0},
		{0x2000b298, 4, false, -1, 0, 0},
		{0x2000b29c, 4, false, -1, 0, 0},
		{0x2000b2a0, 4, false, -1, 0, 0},
		{0x2000b2a4, 4, false, -1, 0, 0},
		{0x2000b2a8, 4, false, -1, 0, 0},
		{0x2000b2ac, 4, false, -1, 0, 0},
		{0x2000b2b0, 4, false, -1, 0, 0},
		{0x2000b2b4, 4, false, -1, 0, 0},
		{0x2000b2b8, 4, false, -1, 0, 0},
		{0x2000b2bc, 4, false, -1, 0, 0},
		{0x2000b2c0, 4, false, -1, 0, 0},
		{0x2000b2c4, 4, false, -1, 0, 0},
		{0x2000b2c8, 4, false, -1, 0, 0},
		{0x2000b2cc, 4, false, -1, 0, 0},
		{0x2000b2d0, 4, false, -1, 0, 0},
		{0x2000b2d4, 4, false, -1, 0, 0},
		{0x2000b2d8, 4, false, -1, 0, 0},
		{0x2000b2dc, 4, false, -1, 0, 0},
		{0x2000b2e0, 4, false, -1, 0, 0},
		{0x2000b2e4, 4, false, -1, 0, 0},
		{0x2000b2e8, 4, false, -1, 0, 0},
		{0x2000b2ec, 4, false, -1, 0, 0},
		{0x2000b2f0, 4, false, -1, 0, 0},
		{0x2000b2f4, 4, false, -1, 0, 0},
		{0x2000b2f8, 4, false, -1, 0, 0},
		{0x2000b2fc, 4, false, -1, 0, 0},
		{0x2000b300, 4, false, -1, 0, 0},
		{0x2000b304, 4, false, -1, 0, 0},
		{0x2000b308, 4, false, -1, 0, 0},
		{0x2000b30c, 4, false, -1, 0, 0},
		{0x2000b310, 4, false, -1, 0, 0},
		{0x2000b314, 4, false, -1, 0, 0},
		{0x2000b318, 4, false, -1, 0, 0},
		{0x2000b31c, 4, false, -1, 0, 0},
		{0x2000b320, 4, false, -1, 0, 0},
		{0x2000b324, 4, false, -1, 0, 0},
		{0x2000b328, 4, false, -1, 0, 0},
		{0x2000b32c, 4, false, -1, 0, 0},
		{0x2000b330, 4, false, -1, 0, 0},
		{0x2000b334, 4, false, -1, 0, 0},
		{0x2000b338, 4, false, -1, 0, 0},
		{0x2000b33c, 4, false, -1, 0, 0},
		{0x2000b340, 4, false, -1, 0, 0},
		{0x2000b344, 4, false, -1, 0, 0},
		{0x2000b348, 4, false, -1, 0, 0},
		{0x2000b34c, 4, false, -1, 0, 0},
		{0x2000b350, 4, false, -1, 0, 0},
		{0x2000b354, 4, false, -1, 0, 0},
		{0x2000b358, 4, false, -1, 0, 0},
		{0x2000b35c, 4, false, -1, 0, 0},
		{0x2000b360, 4, false, -1, 0, 0},
		{0x2000b364, 4, false, -1, 0, 0},
		{0x2000b40c, 4, false, -1, 0, 0},
		{0x2000b410, 4, false, -1, 0, 0},
		{0x2000b414, 4, false, -1, 0, 0},
		{0x2000b418, 4, false, -1, 0, 0},
		{0x2000b41c, 4, false, -1, 0, 0},
		{0x2000b420, 4, false, -1, 0, 0},
		{0x2000b424, 4, false, -1, 0, 0},
		{0x2000b428, 4, false, -1, 0, 0},
		{0x2000b42c, 4, false, -1, 0, 0},
		{0x2000b430, 4, false, -1, 0, 0},
		{0x2000b434, 4, false, -1, 0, 0},
		{0x2000b438, 4, false, -1, 0, 0},
		{0x2000b43c, 4, false, -1, 0, 0},
		{0x2000b440, 4, false, -1, 0, 0},
		{0x2000b444, 4, false, -1, 0, 0},
		{0x2000b448, 4, false, -1, 0, 0},
		{0x2000b44c, 4, false, -1, 0, 0},
		{0x2000b450, 4, false, -1, 0, 0},
		{0x2000b454, 4, false, -1, 0, 0},
		{0x2000b458, 4, false, -1, 0, 0},
		{0x2000b45c, 4, false, -1, 0, 0},
		{0x2000b460, 4, false, -1, 0, 0},
		{0x2000b464, 4, false, -1, 0, 0},
		{0x2000b468, 4, false, -1, 0, 0},
		{0x2000b46c, 4, false, -1, 0, 0},
		{0x2000b470, 4, false, -1, 0, 0},
		{0x2000b474, 4, false, -1, 0, 0},
		{0x2000b478, 4, false, -1, 0, 0},
		{0x2000b47c, 4, false, -1, 0, 0},
		{0x2000b480, 4, false, -1, 0, 0},
		{0x2000b484, 4, false, -1, 0, 0},
		{0x2000b488, 4, false, -1, 0, 0},
		{0x2000b48c, 4, false, -1, 0, 0},
		{0x2000b490, 4, false, -1, 0, 0},
		{0x2000b494, 4, false, -1, 0, 0},
		{0x2000b498, 4, false, -1, 0, 0},
		{0x20031010, 4, false, -1, 0, 0},
		{0x20031014, 4, false, -1, 0, 0},
		{0x20031018, 4, false, -1, 0, 0},
		{0x2003101c, 4, false, -1, 0, 0},
		{0x20031020, 4, false, -1, 0, 0},
		{0x20031024, 4, false, -1, 0, 0},
		{0x20031028, 4, false, -1, 0, 0},
		{0x2003102c, 4, false, -1, 0, 0},
		{0x20031030, 4, false, -1, 0, 0},
		{0x20031034, 4, false, -1, 0, 0},
		{0x20031038, 4, false, -1, 0, 0},
		{0x2003103c, 4, false, -1, 0, 0},
		{0x20031040, 4, false, -1, 0, 0},
		{0x20031044, 4, false, -1, 0, 0},
		{0x20031048, 4, false, -1, 0, 0},
		{0x2003104c, 4, false, -1, 0, 0},
		{0x20031050, 4, false, -1, 0, 0},
		{0x20031054, 4, false, -1, 0, 0},
		{0x20031058, 4, false, -1, 0, 0},
		{0x2003105c, 4, false, -1, 0, 0},
		{0x20031060, 4, false, -1, 0, 0},
		{0x20031064, 4, false, -1, 0, 0},
		{0x20031068, 4, false, -1, 0, 0},
		{0x2003106c, 4, false, -1, 0, 0},
		{0x20031070, 4, false, -1, 0, 0},
		{0x20031074, 4, false, -1, 0, 0},
		{0x20031078, 4, false, -1, 0, 0},
		{0x2003107c, 4, false, -1, 0, 0},
		{0x20031080, 4, false, -1, 0, 0},
		{0x20031084, 4, false, -1, 0, 0},
		{0x20031088, 4, false, -1, 0, 0},
		{0x2003108c, 4, false, -1, 0, 0},
		{0x20031090, 4, false, -1, 0, 0},
		{0x20031094, 4, false, -1, 0, 0},
		{0x20031098, 4, false, -1, 0, 0},
		{0x2003109c, 4, false, -1, 0, 0},
		{0x200310a0, 4, false, -1, 0, 0},
		{0x200310a4, 4, false, -1, 0, 0},
		{0x200310a8, 4, false, -1, 0, 0},
		{0x200310ac, 4, false, -1, 0, 0},
		{0x200310b0, 4, false, -1, 0, 0},
		{0x200310b4, 4, false, -1, 0, 0},
		{0x200310b8, 1, false, -1, 0, 0},
		{0x200310b9, 1, false, -1, 0, 0},
		{0x200310ba, 1, false, -1, 0, 0},
		{0x200310bb, 1, false, -1, 0, 0},
		{0x200310bc, 1, false, -1, 0, 0},
		{0x200310bd, 1, false, -1, 0, 0},
		{0x200310be, 1, false, -1, 0, 0},
		{0x200310bf, 1, false, -1, 0, 0},
		{0x200310c0, 4, false, -1, 0, 0},
		{0x200310c4, 4, false, -1, 0, 0},
		{0x200310c8, 4, false, -1, 0, 0},
		{0x200310cc, 4, false, -1, 0, 0},
		{0x200310d0, 4, false, -1, 0, 0},
		{0x200310d4, 4, false, -1, 0, 0},
		{0x200310d8, 4, false, -1, 0, 0},
		{0x200310dc, 4, false, -1, 0, 0},
		{0x200310e0, 4, false, -1, 0, 0},
		{0x200310e4, 4, false, -1, 0, 0},
		{0x200310e8, 4, false, -1, 0, 0},
		{0x200310ec, 4, false, -1, 0, 0},
		{0x200310f0, 4, false, -1, 0, 0},
		{0x200310f4, 4, false, -1, 0, 0},
		{0x200310f8, 4, false, -1, 0, 0},
		{0x200310fc, 4, false, -1, 0, 0},
		{0x2000602b, 4, false, -1, 0, 0},
		{0x200060c5, 2, false, -1, 0, 0},
		{0x200060c7, 4, false, -1, 0, 0},
		{0x200060bf, 4, false, -1, 0, 0},
		{0x200060c3, 4, false, -1, 0, 0},
		{0x20006033, 4, false, -1, 0, 0},
		{0x20006242, 4, false, -1, 0, 0},
		{0x20006047, 4, false, -1, 0, 0},
		{0x2000604f, 4, false, -1, 0, 0},
		{0x20006247, 4, false, -1, 0, 0},
		{0x2000624b, 1, false, -1, 0, 0},
		{0x20006304, 4, false, -1, 0, 0},
		{0x20006308, 4, false, -1, 0, 0},
		{0x2000630c, 4, false, -1, 0, 0},
		{0x20006310, 4, false, -1, 0, 0},
		{0x20006314, 4, false, -1, 0, 0},
		{0x20006318, 4, false, -1, 0, 0},
		{0x2000631c, 4, false, -1, 0, 0},
		{0x20006320, 4, false, -1, 0, 0},
		{0x20006324, 4, false, -1, 0, 0},
		{0x20006328, 4, false, -1, 0, 0},
		{0x2000632c, 4, false, -1, 0, 0},
		{0x20006330, 4, false, -1, 0, 0},
		{0x20006334, 4, false, -1, 0, 0},
		{0x200061db, 4, false, -1, 0, 0},
		{0x200061df, 4, false, -1, 0, 0},
		{0x200061e3, 4, false, -1, 0, 0},
		{0x200061e7, 4, false, -1, 0, 0},
		{0x200061eb, 4, false, -1, 0, 0},
		{0x200061ef, 4, false, -1, 0, 0},
		{0x200061f3, 4, false, -1, 0, 0},
		{0x200061f7, 4, false, -1, 0, 0},
		{0x200061fb, 4, false, -1, 0, 0},
		{0x200061ff, 4, false, -1, 0, 0},
		{0x20006203, 4, false, -1, 0, 0},
		{0x20006207, 4, false, -1, 0, 0},
		{0x2000620b, 4, false, -1, 0, 0},
		{0x2000620f, 4, false, -1, 0, 0},
		{0x20006213, 4, false, -1, 0, 0},
		{0x20006217, 4, false, -1, 0, 0},
		{0x2000621b, 4, false, -1, 0, 0},
		{0x2000621f, 4, false, -1, 0, 0},
		{0x20006223, 4, false, -1, 0, 0},
		{0x20006227, 4, false, -1, 0, 0},
		{0x2000622b, 4, false, -1, 0, 0},
		{0x2000622f, 4, false, -1, 0, 0},
		{0x20006233, 4, false, -1, 0, 0},
		{0x20006237, 4, false, -1, 0, 0},
		{0x2000623b, 4, false, -1, 0, 0},
		{0x2000623f, 4, false, -1, 0, 0},
		{0x2000642b, 4, false, -1, 0, 0},
		{0x200064c5, 2, false, -1, 0, 0},
		{0x200064c7, 4, false, -1, 0, 0},
		{0x200064bf, 4, false, -1, 0, 0},
		{0x200064c3, 4, false, -1, 0, 0},
		{0x20006433, 4, false, -1, 0, 0},
		{0x20006642, 4, false, -1, 0, 0},
		{0x20006447, 4, false, -1, 0, 0},
		{0x2000644f, 4, false, -1, 0, 0},
		{0x20006647, 4, false, -1, 0, 0},
		{0x2000664b, 1, false, -1, 0, 0},
		{0x20006704, 4, false, -1, 0, 0},
		{0x20006708, 4, false, -1, 0, 0},
		{0x2000670c, 4, false, -1, 0, 0},
		{0x20006710, 4, false, -1, 0, 0},
		{0x20006714, 4, false, -1, 0, 0},
		{0x20006718, 4, false, -1, 0, 0},
		{0x2000671c, 4, false, -1, 0, 0},
		{0x20006720, 4, false, -1, 0, 0},
		{0x20006724, 4, false, -1, 0, 0},
		{0x20006728, 4, false, -1, 0, 0},
		{0x2000672c, 4, false, -1, 0, 0},
		{0x20006730, 4, false, -1, 0, 0},
		{0x20006734, 4, false, -1, 0, 0},
		{0x200065db, 4, false, -1, 0, 0},
		{0x200065df, 4, false, -1, 0, 0},
		{0x200065e3, 4, false, -1, 0, 0},
		{0x200065e7, 4, false, -1, 0, 0},
		{0x200065eb, 4, false, -1, 0, 0},
		{0x200065ef, 4, false, -1, 0, 0},
		{0x200065f3, 4, false, -1, 0, 0},
		{0x200065f7, 4, false, -1, 0, 0},
		{0x200065fb, 4, false, -1, 0, 0},
		{0x200065ff, 4, false, -1, 0, 0},
		{0x20006603, 4, false, -1, 0, 0},
		{0x20006607, 4, false, -1, 0, 0},
		{0x2000660b, 4, false, -1, 0, 0},
		{0x2000660f, 4, false, -1, 0, 0},
		{0x20006613, 4, false, -1, 0, 0},
		{0x20006617, 4, false, -1, 0, 0},
		{0x2000661b, 4, false, -1, 0, 0},
		{0x2000661f, 4, false, -1, 0, 0},
		{0x20006623, 4, false, -1, 0, 0},
		{0x20006627, 4, false, -1, 0, 0},
		{0x2000662b, 4, false, -1, 0, 0},
		{0x2000662f, 4, false, -1, 0, 0},
		{0x20006633, 4, false, -1, 0, 0},
		{0x20006637, 4, false, -1, 0, 0},
		{0x2000663b, 4, false, -1, 0, 0},
		{0x2000663f, 4, false, -1, 0, 0},
		{0x2000341a, 4, false, -1, 0, 0},
		{0x2000341e, 4, false, -1, 0, 0},
		{0x20003422, 4, false, -1, 0, 0},
		{0x20003426, 4, false, -1, 0, 0},
		{0x2000342a, 4, false, -1, 0, 0},
		{0x2000342e, 4, false, -1, 0, 0},
		{0x20003432, 4, false, -1, 0, 0},
		{0x20003436, 4, false, -1, 0, 0},
		{0x20030280, 4, false, -1, 0, 0},
		{0x20030284, 4, false, -1, 0, 0},
		{0x200302cc, 4, false, -1, 0, 0},
		{0x200302dc, 4, false, -1, 0, 0},
		{0x200302e0, 4, false, -1, 0, 0},
		{0x200302e4, 4, false, -1, 0, 0},
		{0x20003300, 4, false, -1, 0, 0},
		{0x20003304, 4, false, -1, 0, 0},
		{0x20003308, 4, false, -1, 0, 0},
		{0x2000330c, 4, false, -1, 0, 0},
		{0x20003310, 4, false, -1, 0, 0},
		{0x20003314, 4, false, -1, 0, 0},
		{0x20003318, 4, false, -1, 0, 0},
		{0x2000331c, 4, false, -1, 0, 0},
		{0x6875c3, 4, false, -1, 0, 0},
		{0x6875c7, 4, false, -1, 0, 0},
		{0x6875cb, 4, false, -1, 0, 0},
		{0x6875cf, 4, false, -1, 0, 0},
		{0x6875d3, 4, false, -1, 0, 0},
		{0x6875d7, 4, false, -1, 0, 0},
		{0x6875db, 4, false, -1, 0, 0},
		{0x6875df, 4, false, -1, 0, 0},
		{0x10007fb0, 4, false, -1, 0, 0},
		{0x10007fb4, 4, false, -1, 0, 0},
		{0x10007fb8, 4, false, -1, 0, 0},
		{0x10007fbc, 4, false, -1, 0, 0},
		{0x10007fc0, 4, false, -1, 0, 0},
		{0x10007fc4, 4, false, -1, 0, 0},
		{0x10007fc8, 4, false, -1, 0, 0},
		{0x10007fcc, 4, false, -1, 0, 0},
		{0x10007fd0, 4, false, -1, 0, 0},
		{0x10007fd4, 4, false, -1, 0, 0},
		{0x10007fd8, 4, false, -1, 0, 0},
		{0x10007fdc, 4, false, -1, 0, 0},
		{0x10007fe0, 4, false, -1, 0, 0},
		{0x10007fe4, 4, false, -1, 0, 0},
		{0x10007fe8, 4, false, -1, 0, 0},
		{0x10007fec, 4, false, -1, 0, 0},
		{0x10007ff0, 4, false, -1, 0, 0},
		{0x10007ff4, 4, false, -1, 0, 0},
		{0x10007ff8, 4, false, -1, 0, 0},
		{0x10007ffc, 4, false, -1, 0, 0},
		{0x10007fa8, 4, false, -1, 0, 0},
		{0x10007fac, 1, false, -1, 0, 0},
		{0x10008004, 4, false, -1, 0, 0},
	},
})

var merIn = []string{"notice", "messageAll", "aliasCharacter", "missileType", "radius", "Z", "N", "S", "power", "bonusPercent", "sourceType", "sourceCharacter", "sourceFlag1b", "sourceFlag1c", "sourceFlag1e", "targetType", "targetCharacter", "targetMonster", "resultTotal", "resultHits", "resultExhausted", "eventCount", "eventCapacity", "resistedText0", "resistedText1", "quantityReturn", "resistedValue", "ignoredReturn", "slotActive0", "slotActive1", "slotActive2", "slotActive3", "slotActive4", "slotActive5", "slotActive6", "slotActive7", "active0", "position0X", "position0Y", "position0Z", "center0X", "center0Y", "center0Z", "hp0", "maxHp0", "shield0", "shieldDuration0", "maxStamina0", "stamina0", "exhaustion0", "credits0", "messages0", "group0", "actorFlags0", "actorState0", "templateKind0", "resistanceModifier0Fire", "resistanceModifier0Water", "resistanceModifier0Air", "resistanceModifier0Earth", "resistanceModifier0Mental", "resistanceModifier0Divine", "templateResistance0Fire", "templateResistance0Water", "templateResistance0Air", "templateResistance0Earth", "templateResistance0Mental", "templateResistance0Divine", "templateLevel0", "active1", "position1X", "position1Y", "position1Z", "center1X", "center1Y", "center1Z", "hp1", "maxHp1", "shield1", "shieldDuration1", "maxStamina1", "stamina1", "exhaustion1", "credits1", "messages1", "group1", "actorFlags1", "actorState1", "templateKind1", "resistanceModifier1Fire", "resistanceModifier1Water", "resistanceModifier1Air", "resistanceModifier1Earth", "resistanceModifier1Mental", "resistanceModifier1Divine", "templateResistance1Fire", "templateResistance1Water", "templateResistance1Air", "templateResistance1Earth", "templateResistance1Mental", "templateResistance1Divine", "templateLevel1", "groupNamed", "character0Resistance0", "character0Resistance1", "character0Resistance2", "character0Resistance3", "character0Resistance4", "character0Resistance5", "character0Level", "character1Resistance0", "character1Resistance1", "character1Resistance2", "character1Resistance3", "character1Resistance4", "character1Resistance5", "character1Level", "character2Resistance0", "character2Resistance1", "character2Resistance2", "character2Resistance3", "character2Resistance4", "character2Resistance5", "character2Level", "character3Resistance0", "character3Resistance1", "character3Resistance2", "character3Resistance3", "character3Resistance4", "character3Resistance5", "character3Level", "character4Resistance0", "character4Resistance1", "character4Resistance2", "character4Resistance3", "character4Resistance4", "character4Resistance5", "character4Level", "character5Resistance0", "character5Resistance1", "character5Resistance2", "character5Resistance3", "character5Resistance4", "character5Resistance5", "character5Level", "character6Resistance0", "character6Resistance1", "character6Resistance2", "character6Resistance3", "character6Resistance4", "character6Resistance5", "character6Level", "character7Resistance0", "character7Resistance1", "character7Resistance2", "character7Resistance3", "character7Resistance4", "character7Resistance5", "character7Level"}

var merOut = []string{"resistCalls", "resistArg3_0", "resistArg3_1", "resistArg3_2", "resistArg3_3", "resistArg3_4", "resistArg3_5", "resistArg4_0", "resistArg4_1", "resistArg4_2", "resistArg4_3", "resistArg4_4", "resistArg4_5", "charDamageCalls", "charDamageArg1_0", "charDamageArg1_1", "charDamageArg1_2", "charDamageArg2_0", "charDamageArg2_1", "charDamageArg2_2", "charDamageArg4_0", "charDamageArg4_1", "charDamageArg4_2", "charDamageArg6_0", "charDamageArg6_1", "charDamageArg6_2", "hitResponseCalls", "hitResponseArg1_0", "hitResponseArg1_1", "hitResponseArg1_2", "hitResponseArg3_0", "hitResponseArg3_1", "hitResponseArg3_2", "removeStateCalls", "removeStateArg1_0", "removeStateArg1_1", "removeStateArg1_2", "killRewardsCalls", "killRewardsArg1_0", "killRewardsArg1_1", "killRewardsArg1_2", "killRewardsArg2_0", "killRewardsArg2_1", "killRewardsArg2_2", "conditionApplyCalls", "conditionApplyArg1_0", "conditionApplyArg1_1", "conditionApplyArg1_2", "conditionApplyArg2_0", "conditionApplyArg2_1", "conditionApplyArg2_2", "conditionApplyArg3_0", "conditionApplyArg3_1", "conditionApplyArg3_2", "conditionApplyArg4_0", "conditionApplyArg4_1", "conditionApplyArg4_2", "conditionApplyArg6_0", "conditionApplyArg6_1", "conditionApplyArg6_2", "monsterMessageCalls", "monsterMessageArg1_0", "monsterMessageArg1_1", "monsterMessageArg1_2", "monsterMessageArg1_3", "monsterMessageArg1_4", "monsterMessageArg1_5", "monsterMessageArg2_0", "monsterMessageArg2_1", "monsterMessageArg2_2", "monsterMessageArg2_3", "monsterMessageArg2_4", "monsterMessageArg2_5", "monsterMessageArg3_0", "monsterMessageArg3_1", "monsterMessageArg3_2", "monsterMessageArg3_3", "monsterMessageArg3_4", "monsterMessageArg3_5", "messageCalls", "messageArg1_0", "messageArg1_1", "messageArg1_2", "messageArg1_3", "messageArg1_4", "messageArg1_5", "messageArg1_6", "messageArg1_7", "messageArg1_8", "messageArg1_9", "messageArg1_10", "messageArg1_11", "messageArg1_12", "messageArg1_13", "messageArg1_14", "messageArg1_15", "messageArg1_16", "messageArg1_17", "messageArg1_18", "messageArg1_19", "messageArg2_0", "messageArg2_1", "messageArg2_2", "messageArg2_3", "messageArg2_4", "messageArg2_5", "messageArg2_6", "messageArg2_7", "messageArg2_8", "messageArg2_9", "messageArg2_10", "messageArg2_11", "messageArg2_12", "messageArg2_13", "messageArg2_14", "messageArg2_15", "messageArg2_16", "messageArg2_17", "messageArg2_18", "messageArg2_19", "messageArg3_0", "messageArg3_1", "messageArg3_2", "messageArg3_3", "messageArg3_4", "messageArg3_5", "messageArg3_6", "messageArg3_7", "messageArg3_8", "messageArg3_9", "messageArg3_10", "messageArg3_11", "messageArg3_12", "messageArg3_13", "messageArg3_14", "messageArg3_15", "messageArg3_16", "messageArg3_17", "messageArg3_18", "messageArg3_19", "messageArg4_0", "messageArg4_1", "messageArg4_2", "messageArg4_3", "messageArg4_4", "messageArg4_5", "messageArg4_6", "messageArg4_7", "messageArg4_8", "messageArg4_9", "messageArg4_10", "messageArg4_11", "messageArg4_12", "messageArg4_13", "messageArg4_14", "messageArg4_15", "messageArg4_16", "messageArg4_17", "messageArg4_18", "messageArg4_19", "portraitCalls", "portraitArg1_0", "portraitArg1_1", "portraitArg1_2", "portraitArg1_3", "portraitArg1_4", "portraitArg1_5", "portraitArg1_6", "portraitArg1_7", "portraitArg1_8", "portraitArg1_9", "portraitArg1_10", "portraitArg1_11", "portraitArg1_12", "portraitArg1_13", "portraitArg1_14", "portraitArg1_15", "portraitArg1_16", "portraitArg1_17", "portraitArg1_18", "portraitArg1_19", "portraitArg3_0", "portraitArg3_1", "portraitArg3_2", "portraitArg3_3", "portraitArg3_4", "portraitArg3_5", "portraitArg3_6", "portraitArg3_7", "portraitArg3_8", "portraitArg3_9", "portraitArg3_10", "portraitArg3_11", "portraitArg3_12", "portraitArg3_13", "portraitArg3_14", "portraitArg3_15", "portraitArg3_16", "portraitArg3_17", "portraitArg3_18", "portraitArg3_19", "hitSoundCalls", "hitSoundArg1_0", "hitSoundArg1_1", "hitSoundArg1_2", "shieldVisualCalls", "shieldVisualArg1_0", "shieldVisualArg1_1", "shieldVisualArg1_2", "shieldVisualArg2_0", "shieldVisualArg2_1", "shieldVisualArg2_2", "deathAnimationCalls", "deathAnimationArg1_0", "deathAnimationArg1_1", "deathAnimationArg1_2", "deathAnimationArg2_0", "deathAnimationArg2_1", "deathAnimationArg2_2", "deathNotifyCalls", "deathNotifyArg1_0", "deathNotifyArg1_1", "deathNotifyArg1_2", "nameLogCalls", "nameLogArg1_0", "nameLogArg1_1", "nameLogArg1_2", "nameLogArg1_3", "nameLogArg1_4", "nameLogArg1_5", "nameLogArg3_0", "nameLogArg3_1", "nameLogArg3_2", "nameLogArg3_3", "nameLogArg3_4", "nameLogArg3_5", "zeroDamageVisualCalls", "soundNameCalls", "soundNameArg2_0", "soundCalls", "soundArg1_0", "soundArg2_0", "quantityCalls", "quantityArg1_0", "partyDamageCalls", "partyDamageArg1_0", "partyDamageArg2_0", "partyDamageArg3_0", "radiusDamageCalls", "radiusDamageArg1_0", "radiusDamageArg2_0", "radiusDamageArg3_0", "radiusDamageArg4_0", "radiusDamageArg5_0", "attackEffectsCalls", "attackEffectsArg1_0", "attackEffectsArg1_1", "attackEffectsArg1_2", "attackEffectsArg1_3", "attackEffectsArg1_4", "attackEffectsArg1_5", "attackEffectsArg1_6", "attackEffectsArg1_7", "attackEffectsArg2_0", "attackEffectsArg2_1", "attackEffectsArg2_2", "attackEffectsArg2_3", "attackEffectsArg2_4", "attackEffectsArg2_5", "attackEffectsArg2_6", "attackEffectsArg2_7", "attackEffectsArg3_0", "attackEffectsArg3_1", "attackEffectsArg3_2", "attackEffectsArg3_3", "attackEffectsArg3_4", "attackEffectsArg3_5", "attackEffectsArg3_6", "attackEffectsArg3_7", "attackEffectsArg4_0", "attackEffectsArg4_1", "attackEffectsArg4_2", "attackEffectsArg4_3", "attackEffectsArg4_4", "attackEffectsArg4_5", "attackEffectsArg4_6", "attackEffectsArg4_7", "attackEffectsArg5_0", "attackEffectsArg5_1", "attackEffectsArg5_2", "attackEffectsArg5_3", "attackEffectsArg5_4", "attackEffectsArg5_5", "attackEffectsArg5_6", "attackEffectsArg5_7", "attackEffectsArg6_0", "attackEffectsArg6_1", "attackEffectsArg6_2", "attackEffectsArg6_3", "attackEffectsArg6_4", "attackEffectsArg6_5", "attackEffectsArg6_6", "attackEffectsArg6_7", "hp0", "shield0", "shieldDuration0", "shieldPower0", "shieldWord0", "stamina0", "fatigue0", "rebuild470", "rebuild4f0", "rebuild2470", "rebuild24b0", "lastSource0_0", "lastSource0_1", "lastSource0_2", "lastSource0_3", "lastSource0_4", "lastSource0_5", "lastSource0_6", "lastSource0_7", "lastSource0_8", "lastSource0_9", "lastSource0_10", "lastSource0_11", "lastSource0_12", "modifiers0_0", "modifiers0_1", "modifiers0_2", "modifiers0_3", "modifiers0_4", "modifiers0_5", "modifiers0_6", "modifiers0_7", "modifiers0_8", "modifiers0_9", "modifiers0_10", "modifiers0_11", "modifiers0_12", "modifiers0_13", "modifiers0_14", "modifiers0_15", "modifiers0_16", "modifiers0_17", "modifiers0_18", "modifiers0_19", "modifiers0_20", "modifiers0_21", "modifiers0_22", "modifiers0_23", "modifiers0_24", "modifiers0_25", "hp1", "shield1", "shieldDuration1", "shieldPower1", "shieldWord1", "stamina1", "fatigue1", "rebuild471", "rebuild4f1", "rebuild2471", "rebuild24b1", "lastSource1_0", "lastSource1_1", "lastSource1_2", "lastSource1_3", "lastSource1_4", "lastSource1_5", "lastSource1_6", "lastSource1_7", "lastSource1_8", "lastSource1_9", "lastSource1_10", "lastSource1_11", "lastSource1_12", "modifiers1_0", "modifiers1_1", "modifiers1_2", "modifiers1_3", "modifiers1_4", "modifiers1_5", "modifiers1_6", "modifiers1_7", "modifiers1_8", "modifiers1_9", "modifiers1_10", "modifiers1_11", "modifiers1_12", "modifiers1_13", "modifiers1_14", "modifiers1_15", "modifiers1_16", "modifiers1_17", "modifiers1_18", "modifiers1_19", "modifiers1_20", "modifiers1_21", "modifiers1_22", "modifiers1_23", "modifiers1_24", "modifiers1_25", "partyKills0", "partyKills1", "partyKills2", "partyKills3", "partyKills4", "partyKills5", "partyKills6", "partyKills7", "resultTotal", "resultHits", "resultExhausted", "eventCount", "eventCapacity", "eventArray", "eventSlot0", "eventSlot1", "eventSlot2", "eventSlot3", "eventSlot4", "eventSlot5", "eventSlot6", "eventSlot7", "aliasText0", "aliasText1", "aliasText2", "aliasText3", "aliasText4", "aliasText5", "aliasText6", "aliasText7", "framePosition0", "framePosition1", "framePosition2", "frameOffset0", "frameOffset1", "frameOffset2", "frameCentre0", "frameCentre1", "frameCentre2", "frameVector0", "frameVector1", "frameVector2", "frameQuery0", "frameQuery1", "frameQuery2", "frameQuery3", "frameQuery4", "frameQuery5", "frameQuery6", "frameQuery7", "frameTarget0", "frameNotice0", "frameArgument0"}

// merStubs are this node's own address stubs (object stub2).
var merStubs = map[uint32]*mcaStub{
	0x517a70: {ret: 0x20031004, retN: 0, rec: [][4]uint32{{2, 4, 0x20031014, 1}}, cnt: 0x20031010, wr: []mcaWrite{}},
	0x408860: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2003101c, 1}, {2, 4, 0x20031020, 1}}, cnt: 0x20031018, wr: []mcaWrite{}},
	0x551a20: {ret: 0x20031000, retN: 0, rec: [][4]uint32{{1, 4, 0x20031028, 1}}, cnt: 0x20031024, wr: []mcaWrite{}},
	0x52a820: {ret: 0x2000b030, retN: 0, rec: [][4]uint32{{1, 4, 0x20031030, 1}, {2, 4, 0x20031034, 1}, {3, 4, 0x20031038, 1}}, cnt: 0x2003102c, wr: []mcaWrite{}},
	0x52ba80: {ret: 0x2000b030, retN: 0, rec: [][4]uint32{{1, 4, 0x20031040, 1}, {2, 4, 0x20031044, 1}, {3, 4, 0x20031048, 1}, {4, 4, 0x2003104c, 1}, {5, 4, 0x20031050, 1}}, cnt: 0x2003103c, wr: []mcaWrite{}},
	0x543270: {ret: 0x2000b030, retN: 0, rec: [][4]uint32{{1, 4, 0x20031058, 8}, {2, 4, 0x20031078, 8}, {3, 4, 0x20031098, 8}, {4, 1, 0x200310b8, 8}, {5, 4, 0x200310c0, 8}, {6, 4, 0x200310e0, 8}}, cnt: 0x20031054, wr: []mcaWrite{}},
}
