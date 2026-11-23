package calc

import (
	"fmt"
	"math"
)

// ---- 0x540170: monster attack resolution ----
//
// One swing of monster M (arg 1) with attack slot C+0x11 of template T (arg 2),
// C = M+0x10. The attack kind (0x51a730 on M+0x2e1) picks melee or missile.
//
// Melee rolls d100 against the to-hit chance (0x542720 mode 0); 1..5 always
// hits, 96..100 always misses. The fumble chance is n*trunc((100-x)^3/1e5+0.5)
// *10/100 clamped to 0..100, with x = 0x542720 mode 1 and n the size of the
// fumble target list (0x544010). A fumble (d100 past 100-fumble, no sure hit)
// retargets a random list entry (0x544250 logs it) and re-rolls as a sure hit.
// Missiles take the outcome flag G+0x8c4 (1 sure hit, 2 blocked, else miss).
// Sure misses and blocks log, clear the target (0x536170) and go to the tail.
//
// Otherwise the target G+0x8c5 must still be valid (0x536190, else log and
// return 3); 0x545e50 may intercept it. The hit location is rolled from the
// character weights T+0x157 or the target template's seven weights +0x15f. A
// to-hit miss plays the block reaction (0x543110) or the miss voice / monster
// alert (0x4ece00, 0x52e690, 0x4e4db0). A hit computes the penetration
// chance: (M+0x1dc + M+0x1dd)*5 + A+4 + mode modifier (0x542e10) - fatigue
// penalty (attack types at 0x61d284) - 5*defPen (0x542fc0) + (M stats -
// target stats)/10, scaled 60% / 140% by the difficulty (0x55ccb0). Passing it
// deals 0x542960's damage (fumbles capped by 0x545a00) through 0x52a890 /
// 0x52bb60, then 0x543270 and the character poison follow-up 0x553770.
// Melee runs the target's periodic damage (0x553540). 0x52c500 charges the
// target's stamina by the stage reached. The tail spends the swing (C+0xd),
// re-picks a target for multi-target templates (0x5323f0, 0x541630) and
// returns 2 (swing again) or 3 after the report (0x5412b0, 0x546b40, 0x590b40).
//
// The world is the emulator case's memory image (mstWorld, built from
// re/emu/cases/00540170.json's description). 排除 callees are address stubs
// whose arguments and call counts are outputs. The D4 callees are address
// stubs: Emu replays them; F runs the Go node of each one whose return or
// writes this node reads (0x542720, 0x544010, 0x545e50, 0x536190, 0x53c3f0,
// 0x542fc0, 0x542960, 0x545a00, 0x52a890, 0x5323f0) on the live values, with
// the rest of that node's inputs parked from its own case. 0x4ecc80 runs as
// code: with [0x6850e5] = 2 it returns at once (other values panic).

const (
	// 0x540170's stack locals in the emulator (entry esp 0x10008000).
	m17Src   = 0x10007e44 // 0x53be50's source record (13 dwords)
	marHits  = 0x10007e40 // 0x542960's hit count
	m17Pack  = 0x10007e84 // 0x543270's damage pack (12 dwords)
	marSaved = 0x10007eb4 // M+0x2ba at entry (8 dwords)
	m17Buf   = 0x10007ed4 // 0x53be00 / 0x53be50 record for 0x553540 / 0x553770
	marLocal = 0x10007f08 // local result record (0x28 dwords + a word)
	m17Loc   = 0x10007fac // the hit-location text
	marPick  = 0x10007fd4 // M+0x2ba before 0x5323f0 (8 dwords)
	// 0x4ed260's vsprintf buffers, only an argument of the trace 0x58aad0.
	marTraceFumble = 0x10007d14
	marTracePen    = 0x10007d1c
	m17Target      = 0x8c5 // G+0x8c5: the attack target (8 dwords)
	marResult      = 0x9a6 // G+0x9a6: the shared result record
)

func marArg(w *mcaWorld, n uint32) uint32 { return w.u32(0x10008004 + 4*n) }

func m17Copy(w *mcaWorld, dst, src, n uint32) {
	for i := uint32(0); i < n; i++ {
		w.put(dst+4*i, 4, uint64(w.u32(src+4*i)))
	}
}

func marSame(w *mcaWorld, a, b uint32) bool {
	for i := uint32(0); i < 8; i++ {
		if w.u32(a+4*i) != w.u32(b+4*i) {
			return false
		}
	}
	return true
}

func m17Zero(w *mcaWorld, a, n uint32) {
	for i := uint32(0); i < n; i++ {
		w.put(a+4*i, 4, 0)
	}
}

// marRecord is 0x53be00 (character, i >= 0) / 0x53be50 (monster m) into buf.
func marRecord(w *mcaWorld, buf, typ, idx, id uint32) {
	m17Zero(w, buf, 13)
	w.put(buf, 4, uint64(typ))
	w.put(buf+4, 4, uint64(idx))
	w.put(buf+8, 4, uint64(id))
}

// m17Clear is 0x536170: an empty target.
func m17Clear(w *mcaWorld, d uint32) {
	m17Zero(w, d, 8)
	for _, o := range []uint32{4, 8, 0xc} {
		w.put(d+o, 4, 0xffffffff)
	}
}

// marResultClear zeroes a result record (0x28 dwords and a word).
func marResultClear(w *mcaWorld, res uint32) {
	m17Zero(w, res, 0x28)
	w.put(res+0xa0, 2, 0)
}

// marKind is 0x51a730(M, T, idx): < 2 is melee.
func marKind(w *mcaWorld, m, t, idx uint32) uint32 {
	switch w.u32(m + 0x2e1) {
	case 0:
		if idx >= 3 {
			panic("monster attack resolution: attack slot past 2 (0x51a730 asserts)")
		}
		return uint32(w.u8(t + 0xea + idx*0x22))
	case 2:
		return w.u32(w.u32(0x65be1c) + w.u32(m+0x2e5)*0x1bf + 0x12f)
	case 3:
		return 2
	case 8:
		return 0
	}
	return 0xffffffff
}

// marTemplateID is 0x4e57c0(id) on the preloaded template cache.
func marTemplateID(w *mcaWorld, id uint32) uint32 {
	if id >= 0x3e8 {
		panic("monster attack resolution: template id past 0x3e7 (0x4e57c0 asserts)")
	}
	if id >= mstTemplates {
		panic("monster attack resolution: template cache miss is outside the fixture (file load)")
	}
	t := w.u32(0x6840c7 + 4*id)
	if t == 0 {
		panic("monster attack resolution: template cache entry is empty")
	}
	return t
}

// marPut is 0x517a90: swprintf into 0x689fd0.
func marPut(w *mcaWorld, f uint32, args ...uint32) uint32 {
	srdPutW(w, 0x689fd0, sdmFormat(w, f, args...))
	return 0x689fd0
}

// marTargetText is 0x546b40(src, d) with a monster source.
func marTargetText(w *mcaWorld, src, d uint32) uint32 {
	switch w.u32(d) {
	case 0, 6:
		return 0x689b34
	case 2:
		return mpnStr(w, 0x1e3)
	case 5:
		return mpnStr(w, 0x1e4)
	case 8:
		return mpnStr(w, 0x1e5)
	case 1, 9:
		if w.u32(d+4) == 0xffffffff {
			panic("monster attack resolution: character target without index (0x546b40 asserts)")
		}
		return marPut(w, w.charBase(w.u32(d+4))+5) // the source is a monster (0x53bea0 false)
	case 3:
		if w.u32(d+8) == 0xffffffff {
			panic("monster attack resolution: monster target without id (0x546b40 asserts)")
		}
		m := w.findMonster(w.u32(d + 8))
		t := w.template(m)
		if w.u32(src+8) == w.u32(d+8) {
			return marPut(w, mpnStr(w, w.u16(0x61e436+8*uint32(w.u8(t+0xcc)))))
		}
		return marPut(w, sdmName(w, m))
	case 4:
		g := w.group(w.u32(d + 0xc))
		t := marTemplateID(w, w.u32(g+0x18))
		sel := mstB(w.u32(g+4) != 1)
		if w.u16(t+0x187) == 0x18d {
			srdPutW(w, 0x6875c3, sdmFormat(w, 0x614b44, w.charBase(w.u32(0x6875ef))+5))
			return marPut(w, 0x6875c3)
		}
		if w.u8(g+0x2c) == 0 {
			sel += 2
		}
		return marPut(w, t+0x30*sel)
	case 7:
		return marPut(w, mpnStr(w, 0x1e6), sdmStub(w, 0x51b5c0, w.u32(d+0x1c), 0))
	}
	panic("monster attack resolution: target type outside 0x546b40's switch (asserts)")
}

// m17Report is 0x5412b0(M, res).
func m17Report(w *mcaWorld, m, res uint32) {
	tail := func() {
		if v := w.u32(res + 0x74); v != 0 {
			sdmStub(w, 0x590b40, m, mpnStr(w, 0x267), v)
		}
		if v := w.u32(res + 0x78); v != 0 {
			sdmStub(w, 0x590b40, m, mpnStr(w, 0x268), v)
		}
	}
	if w.u32(res+0x50) == 0 {
		hits := w.u32(res + 4)
		if hits == 0 {
			id := uint32(0x209)
			if w.u8(res+0x81) != 0 {
				id = 0x20b
			}
			sdmStub(w, 0x58ac00, 9, mpnStr(w, id), 0xffffffff, 0xffffffff, 0)
			return
		}
		switch dmg := w.u32(res); {
		case dmg == 0:
			sdmStub(w, 0x58ac00, 9, mpnStr(w, 0x20e), 0xffffffff, 0xffffffff, 0)
		case hits == 1:
			sdmStub(w, 0x590b40, m, mpnStr(w, 0x265), dmg)
		default:
			sdmStub(w, 0x590b40, m, mpnStr(w, 0x266), hits, dmg)
		}
		tail()
		typ, idx := w.u32(res+0x82), w.u32(res+0x86)
		var tm, chk uint32
		if typ == 3 {
			tm = w.findMonster(w.u32(res + 0x8a))
			chk = tm + 0x9f
		} else {
			chk = w.charBase(idx) + 0xa49
		}
		if w.u32(chk) == 0 {
			for i, id := range []uint32{0x1a6, 0x264, 0x261} {
				if v := w.u32(res + 0x68 + 4*uint32(i)); v != 0 {
					marLog(w, 9, mpnStr(w, id), v)
				}
			}
			for k := uint32(0); k < 20; k++ {
				if k == 1 || w.u32(res+8+4*k) == 0 {
					continue
				}
				text := func() uint32 { return mpnStr(w, w.u16(0x61e572+8*k)) }
				switch {
				case k == 0x13:
					if typ == 1 && w.u8(w.charBase(idx)+0x1830) != 0 {
						sdmNotice(w, idx, 0x6068e4, mpnStr(w, 0x1d5))
					}
				case typ == 3:
					marLog(w, 9, 0x614a3c, sdmName(w, tm), text())
				case typ == 1:
					sdmNotice(w, idx, 0x6175c0, text())
				}
			}
		}
	} else {
		tail()
	}
	for int32(w.u32(res+0x5c)) > 0 {
		n, arr := w.u32(res+0x5c), w.u32(res+0x64)
		e := w.u32(arr)
		for i := uint32(0); i+1 < n; i++ {
			w.put(arr+4*i, 4, uint64(w.u32(arr+4*i+4)))
		}
		w.put(res+0x5c, 4, uint64(n-1))
		if e == 0 {
			continue
		}
		text := mpnStr(w, w.u16(0x61e602))
		switch w.u32(e) {
		case 1:
			sdmNotice(w, w.u32(e+4), 0x6175c0, text)
		case 3:
			marLog(w, 9, 0x614a3c, e+8, text)
		}
	}
	marResultClear(w, res)
}

// marFumbleText is 0x544250(src, tgt) with a monster source: "<name> accidentally
// strikes <target>" to G+0x6b8 and the combat log.
func marFumbleText(w *mcaWorld, src, tgt uint32) {
	line := w.u32(0x6836a8) + 0x6b8
	if w.u32(src+8) == 0xffffffff {
		panic("monster attack resolution: monster source without id (0x53bf10 asserts)")
	}
	m := w.findMonster(w.u32(src + 8))
	t := w.template(m)
	srdPutW(w, line, sdmFormat(w, 0x61db58, sdmName(w, m), 0x61db68))
	c2 := mpnTargetSide(w, src, tgt)
	len1 := mpnLen(w, line)
	switch typ := w.u32(tgt); {
	case typ == 3 && w.u32(tgt+8) == w.u32(src+8):
		mpnCat(w, line, mpnStr(w, w.u16(0x61e436+8*uint32(w.u8(t+0xcc)))))
	case typ == 1:
		mpnCat(w, line, w.charBase(w.u32(tgt+4))+5)
	default:
		mpnCat(w, line, sdmName(w, w.findMonster(w.u32(tgt+8))))
	}
	len2 := mpnLen(w, line)
	sdmStub(w, 0x58ac00, 9, line, 0xffffffff, 0xffffffff, 0)
	if c2 != 9 {
		sdmStub(w, 0x58b410, uint32(c2), len1&0xff, len2&0xff, 0xffffffff)
	}
}

// marLookup is 0x549eb0(a, b): the sound table at 0x68d850.
func marLookup(w *mcaWorld, a, b int32) uint32 {
	if a < 0 || a >= 0x1c || b < 0 || b >= 0xc {
		return 0x61e060
	}
	if v := w.u32(0x68d850 + 4*uint32(12*a+b)); v != 0 {
		return v
	}
	return w.u32(0x68d850 + 4*uint32(b))
}

// marItemValue is the item record's +0xc1 dword (-1: none, gives def).
func marItemValue(w *mcaWorld, it, def uint32) uint32 {
	if it == 0xffffffff {
		return def
	}
	return w.u32(w.u32(0x68516c) + it*0x10d + 0xc1)
}

// marImpact is 0x54a270(A, tgt, loc, x): the impact sound.
func marImpact(w *mcaWorld, a, tgt, loc, x uint32) {
	if a == 0 || tgt == 0 {
		return
	}
	var v uint32
	switch w.u32(tgt) {
	case 1:
		if loc > 4 {
			panic("monster attack resolution: hit location past 4 (0x54a270 asserts)")
		}
		slot := []uint32{0, 4, 10, 5, 11}[loc]
		v = marItemValue(w, w.u32(w.charBase(w.u32(tgt+4))+0xf5d+12*slot), 0)
	case 3:
		v = w.u32(w.template(w.findMonster(w.u32(tgt+8))) + 0x263)
	default:
		panic("monster attack resolution: impact target neither character nor monster (0x54a270 asserts)")
	}
	sdmStub(w, 0x5499d0, marLookup(w, int32(int8(w.u8(a+0x1c))), int32(v)), 1, 1, x)
}

// m17Blocked is 0x543110(snd, tgt, roll, chance, 9): a blocked miss.
func m17Blocked(w *mcaWorld, snd int32, tgt uint32, roll, chance int32) bool {
	g := w.u32(0x6836a8)
	var u uint32
	switch w.u32(tgt) {
	case 1:
		c := w.charBase(w.u32(tgt + 4))
		s := int32(0)
		if w.u8(g+0x9a4) == 0 && w.u8(g+0x9a5) == 0 {
			s = int32(w.u32(c+0xe99)) * 5
		}
		if s < roll-chance {
			return false
		}
		u = marItemValue(w, w.u32(c+0xfb1), 2)
	case 3:
		t := w.template(w.findMonster(w.u32(tgt + 8)))
		if w.u8(t+0xd0)&2 == 0 || roll-chance > 10 {
			return false
		}
		u = 2
	default:
		panic("monster attack resolution: blocked-miss target neither character nor monster (0x543110 asserts)")
	}
	if w.u8(0x68510c) != 0 {
		// The third slot is 0x543110's saved register: the caller's stage 1.
		marLog(w, 9, mpnStr(w, 0x213), 1)
	}
	sdmStub(w, 0x5499d0, marLookup(w, snd, int32(u)), 1, 1, 0xffffffff)
	return true
}

// marWince is 0x4ece00(tgt): a character under half health may cry out.
func marWince(w *mcaWorld, tgt uint32) bool {
	if w.u32(tgt) != 1 {
		return false
	}
	c := w.charBase(w.u32(tgt + 4))
	most := w.u32(c + 0xb0d)
	if most == 0 {
		panic("monster attack resolution: max HP 0 divides by zero (0x4ece3f)")
	}
	if w.u32(c+0xb11)*100/most >= w.u32(0x5ed8f8) || uint32(w.r.R(100)) >= w.u32(0x5ed908) {
		return false
	}
	sdmStub(w, 0x52e690, c, w.u32(0x5ee610), 0, w.u32(0x5ed8c8), w.u32(0x5ed914))
	return true
}

// marLight is 0x55ccb0(M, &v): the difficulty scaling.
func marLight(w *mcaWorld, m uint32, v int32) int32 {
	k := [2]uint32{uint32(w.u8(m + 0x16)), w.u32(0x6850d5)}
	switch k {
	case [2]uint32{1, 0}, [2]uint32{2, 2}:
		return v * 60 / 100
	case [2]uint32{1, 2}, [2]uint32{2, 0}:
		return v * 140 / 100
	}
	return v
}

// marFumbleChance is 0x540170's fumble percentage before the clamp: n list
// entries times trunc((100-x)^3 * 1e-5 + 0.5) * 10 / 100 (constants read from
// the exe: 3.0 at 0x5ee7b0, 1e-5 at 0x5ee7a8, 0.5 at 0x5ebe80).
func marFumbleChance(w *mcaWorld, x int32, n uint32) int32 {
	if x >= 100 {
		return 0
	}
	f64 := func(a uint32) float64 { return math.Float64frombits(uint64(w.u32(a)) | uint64(w.u32(a+4))<<32) }
	p := math.Pow(float64(100-x), f64(0x5ee7b0))*f64(0x5ee7a8) + f64(0x5ebe80)
	if p >= 1<<63 {
		panic("monster attack resolution: fumble power past int64 (ftol overflow)")
	}
	if n == 0 {
		panic("monster attack resolution: empty fumble target list (0x540409 asserts)")
	}
	return int32(n * uint32(int64(p)) * 10 / 100)
}

func m17Body(w *mcaWorld) uint32 {
	m, t := marArg(w, 0), marArg(w, 1)
	g := w.u32(0x6836a8)
	tgt := g + m17Target
	res := g + marResult
	b0 := w.u8(0x68510c) != 0
	w.alloc(0x14)
	c := w.u32(m + 0x10)
	idx := w.u32(c + 0x11)
	mode := w.u32(m + 0x2e5)
	a := t + 0xe7 + idx*0x22
	src := uint32(m17Src)
	marRecord(w, src, 2, 0xffffffff, w.u32(m))
	m17Copy(w, marSaved, m+0x2ba, 8)
	if w.u32(marSaved) == 1 {
		i := w.u32(marSaved + 4)
		ch := w.charBase(i)
		if w.u8(ch+0x33f) != 0 {
			w.put(g+0xcb+i*0xd4, 1, 1)
		}
		if w.u8(ch+0x281) != 0 && w.u8(g+0x9a4) == 0 && w.u8(g+0x9a5) == 0 && int32(w.u32(ch+0xe99)) > 0 {
			w.put(g+0xc6+i*0xd4, 1, 1)
		}
		if w.u8(ch+0x741) != 0 {
			w.put(g+0xe6+i*0xd4, 1, 1)
		}
	}
	kind := marKind(w, m, t, idx)
	melee := int32(kind) < 2
	var chance, roll int32
	var sure, sureHit, autoMiss, blocked, fumble bool
	if melee {
		chance = marArhit(w, m, a, mode, 0)
		roll = int32(w.r.R(100)) + 1
		sure, autoMiss = roll <= 5, roll > 95
		list := w.alloc(0xc)
		w.put(list, 4, uint64(w.alloc(0x28)))
		w.put(list+4, 4, 0xa)
		w.put(list+8, 4, 0)
		marFumbleList(w, m, t, idx, list)
		x := marArhit(w, m, a, mode, 1)
		n := w.listCount(list)
		f := marFumbleChance(w, x, n)
		f = min(max(f, 0), 100)
		fumble = uint32(100-f) < uint32(roll)
		if w.u8(0x68d810) != 0 {
			marLog(w, 7, 0x617664, marTraceFumble)
		}
		if !sure && fumble {
			if !b0 {
				res = marLocal
				marResultClear(w, res)
			}
			if n == 0 {
				panic("monster attack resolution: empty fumble target list (0x5404c5 asserts)")
			}
			e := w.listAt(list, uint32(w.r.R(int(n))))
			if e == 0 {
				panic("monster attack resolution: null fumble target (0x5404f5 asserts)")
			}
			m17Copy(w, tgt, e, 8)
			marFumbleText(w, src, tgt)
			sure, sureHit, autoMiss = true, false, false
			roll = int32(w.r.R(100)) + 1
			w.put(g+0x9a4, 1, 0)
			w.put(g+0x9a5, 1, 0)
		} else {
			m17Copy(w, tgt, m+0x2ba, 8)
			sureHit = sure
		}
	} else {
		if w.u8(m+0x2df) == 0 {
			panic("monster attack resolution: missile not released (0x5405a3 asserts)")
		}
		switch w.u8(g + 0x8c4) {
		case 1:
			sure = true
			roll = int32(w.r.R(100)) + 1
		case 2:
			blocked = true
		default:
			autoMiss = true
		}
	}
	if !b0 {
		w.put(res+0x80, 1, 1)
	}
	stage := uint32(0)
	switch {
	case autoMiss:
		if w.u8(0x68510c) != 0 {
			sdmStub(w, 0x58ac00, 9, mpnStr(w, 0x209), 0xffffffff, 0xffffffff, 0)
		}
		m17Clear(w, tgt)
	case blocked:
		if w.u8(0x68510c) == 0 {
			w.put(res+0x81, 1, 1)
		} else {
			sdmStub(w, 0x58ac00, 9, mpnStr(w, 0x20a), 0xffffffff, 0xffffffff, 0)
		}
		m17Clear(w, tgt)
	default:
		// The changed flag lives in the source record's byte +0x1e.
		w.put(src+0x1e, 1, uint64(mstB(!marSame(w, tgt, m+0x2ba))))
		if !mpnValid(w, tgt) {
			marInvalid(w, tgt)
			return 3
		}
		if !b0 && w.u8(src+0x1e) != 0 {
			res = marLocal
			marResultClear(w, res)
		}
		marIntercept(w, src, tgt)
		if w.u32(0x6850e5) != 2 {
			panic("monster attack resolution: 0x4ecc80 with [0x6850e5] != 2 is outside the fixture")
		}
		r, ok := marResolve(w, m, t, a, src, tgt, res, mode, melee, sure, sureHit, fumble, chance, roll, b0)
		if !ok {
			return 3
		}
		stage = r
		marStamina(w, tgt, stage)
	}
	return marTail(w, m, t, c, src, res, mode, melee, fumble, b0)
}

// marResolve is the target's half: location, to-hit, penetration, damage and
// the periodic damage; it returns the stage for 0x52c500 (false: invalid type).
func marResolve(w *mcaWorld, m, t, a, src, tgt, res, mode uint32, melee, sure, sureHit, fumble bool, chance, roll int32, b0 bool) (uint32, bool) {
	g := w.u32(0x6836a8)
	var loc, tm uint32
	var text uint32
	switch w.u32(tgt) {
	case 1:
		r, cum := uint32(w.r.R(100)), uint32(0)
		loc = 5
		for i := uint32(0); i < 5; i++ {
			cum += uint32(w.u8(t + 0x157 + i))
			if r < cum {
				loc = i
				break
			}
		}
		if loc == 5 {
			sdmStub(w, 0x5182e0, 0, 0x61d7a8, cum, sdmName(w, m))
			loc = 1
		}
		text = mpnStr(w, w.u16(0x61e7b0+4*loc))
	case 3:
		tm = w.findMonster(w.u32(g + 0x8cd))
		tt := w.template(tm)
		r, cum := uint32(w.r.R(100)), uint32(0)
		loc = 7
		for i := uint32(0); i < 7; i++ {
			cum += uint32(w.u8(tt + 0x15f + i))
			if r < cum {
				loc = i
				break
			}
		}
		if loc == 7 {
			sdmStub(w, 0x5182e0, 0, 0x61d548, cum, tt)
			loc = 3
		}
		text = 0x689b34
		if w.u8(tt+0x15f+loc) < 100 {
			text = mpnStr(w, w.u16(0x61ea24+2*(uint32(w.u8(tt+0x15e))+6*loc)))
		}
	default:
		marInvalid(w, tgt)
		return 0, false
	}
	srdPutW(w, m17Loc, srdWstr(w, text))
	if !sure && chance < roll {
		if !m17Blocked(w, int32(int8(w.u8(a+0x1c))), tgt, roll, chance) {
			if b0 {
				marLog(w, 9, mpnStr(w, 0x209))
			}
			cried := marWince(w, tgt)
			if !cried && w.u32(tgt) == 1 && w.r.R(100) < 50 {
				voice := [3]uint32{w.u32(0x68c570), w.u32(0x68c560), w.u32(0x68c574)}
				sdmStub(w, 0x52e690, w.charBase(w.u32(tgt+4)), voice[w.r.R(3)])
			}
			if w.u32(tgt) == 3 && byte(sdmStub(w, 0x4c4ca0, w.u32(m+0xc))) != 0 && w.u8(tm+0x24e) == 0 {
				sdmStub(w, 0x4e4db0, tm, 0x13, 1)
			}
		}
		return 1, true
	}
	if w.u8(0x68510c) == 0 {
		m17Copy(w, res+0x82, tgt, 8)
	} else {
		marLog(w, 9, mpnStr(w, 0x20c), m17Loc)
	}
	if mode > 8 {
		panic(fmt.Sprintf("monster attack resolution: attack mode %d past 8 (0x542e10 asserts)", mode))
	}
	pen := (int32(int8(w.u8(m+0x1dd)))+int32(int8(w.u8(m+0x1dc))))*5 + int32(w.u8(a+4)) + attackModePenModifier[mode]
	for i := uint32(0); i < 18; i++ {
		if w.u8(a+1) == w.u8(0x61d284+i) {
			pen -= int32(aiFatiguePenalty(w.u32(m + 0x242)))
			break
		}
	}
	pen += -5 * mpnDefPen(w, tgt, mode, loc)
	var ts uint32
	switch w.u32(tgt) {
	case 3:
		for i := uint32(0); i < 4; i++ {
			ts += uint32(w.u8(tm + 0x248 + i))
		}
	case 1:
		ch := w.charBase(w.u32(tgt + 4))
		ts = w.u32(ch+0x161) + w.u32(ch+0x14d) + w.u32(ch+0x139) + w.u32(ch+0xfd)
	default:
		// 0x540c3d: the location switch above already left on other kinds.
		panic("monster attack resolution: stats of a target neither character nor monster (0x540c3d)")
	}
	var ms uint32
	for i := uint32(0); i < 4; i++ {
		ms += uint32(w.u8(m + 0x248 + i))
	}
	pen += int32(ms-ts) / 10
	pen = marLight(w, m, pen)
	if w.u8(0x68d810) != 0 {
		marLog(w, 7, 0x617664, marTracePen)
	}
	msg := uint32(0x20f)
	missed := !sureHit && pen < roll
	stage := uint32(2)
	if !missed {
		stage = 3
		d := marDamage(w, m, a)
		if d == 0 {
			msg = 0x20e
		} else {
			if fumble {
				d = marCap(w, d, tgt)
			} else if w.u32(marHits) > 1 && b0 {
				marLog(w, 9, mpnStr(w, 0x20d), w.u32(marHits))
			}
			marImpact(w, a, tgt, loc, 0xffffffff)
			kill := res
			if b0 {
				kill = 0
			}
			var r uint32
			if w.u32(tgt) == 1 {
				r = sdmCharDamage(w, w.u32(tgt+4), d, b0, b0, kill)
			} else {
				r = sdmHit(w, tm, d, src, b0, b0, kill)
			}
			if r != 0 {
				m17Zero(w, m17Pack, 12)
				for i, o := range []uint32{5, 9, 0xd, 0x11} {
					w.put(m17Pack+8+4*uint32(i), 4, uint64(w.u32(a+o)))
				}
				b := uint32(w.u8(t + 0x24f))
				w.put(m17Pack+0x18, 4, uint64(b+min(b, 0xf)))
				w.put(m17Pack+0x1c, 4, uint64(w.u8(a+0x1b)))
				sdmStub(w, 0x543270, src, tgt, m17Pack, uint32(w.u8(0x68510c)), 0, res)
				if melee && w.u32(tgt) == 1 && loc == 1 && w.u32(w.charBase(w.u32(tgt+4))+0xf8d) == 500 {
					marRecord(w, m17Buf, 1, w.u32(tgt+4), 0xffffffff)
					sdmStub(w, 0x553770, m, m17Buf, d)
				}
			}
			// A zero after 0x545a00 reaches the miss block with the
			// penetration chance as the message id (0x545a00 never caps a
			// non-zero damage to zero).
			msg = uint32(pen)
		}
		missed = d == 0
	}
	if missed {
		marImpact(w, a, tgt, loc, 0x2a)
		if w.u32(tgt) == 3 {
			sdmStub(w, 0x52beb0, tm, src, 0)
		}
		if !b0 {
			w.put(res+4, 4, uint64(w.u32(res+4)+1))
		} else {
			marLog(w, 9, mpnStr(w, msg))
		}
	}
	if melee {
		if w.u32(tgt) == 3 {
			if w.u32(tm+0xd3) != 0 {
				marRecord(w, m17Buf, 2, 0xffffffff, w.u32(tm))
				sdmStub(w, 0x553540, m, m17Buf, tm+0xcb)
				n := w.u32(tm+0xcb) - 1
				w.put(tm+0xcb, 4, uint64(n))
				if n == 0 {
					// 0x524390(id, 3): clear buff slot 3, 0x4acd80, rebuild.
					id := w.u32(tm)
					x := w.findMonster(id)
					for _, o := range []uint32{0xcb, 0xcf, 0xd3} {
						w.put(x+o, 4, 0)
					}
					sdmStub(w, 0x4acd80, w.u32(x+0xc), 0x13, 0)
					w.rebuild(id)
				}
			}
		} else {
			i := w.u32(tgt + 4)
			p := w.charBase(i) + 0xa89
			if w.u32(p+8) != 0 {
				marRecord(w, m17Buf, 1, i, 0xffffffff)
				sdmStub(w, 0x553540, m, m17Buf, p)
				n := w.u32(p) - 1
				w.put(p, 4, uint64(n))
				if n == 0 {
					sdmStub(w, 0x523a80, i, 3)
				}
			}
		}
	}
	return stage, true
}

// marInvalid logs an attack on a target that is gone or of another kind.
func marInvalid(w *mcaWorld, tgt uint32) {
	g := w.u32(0x6836a8)
	sdmStub(w, 0x5182e0, w.u32(g+0x8cd)&^0xff|mstB(w.u8(0x689b32) == 0), 0x61d584, w.u32(tgt))
}

// marLog is 0x58aad0(channel, f, ...): the stub records three argument
// slots; a shorter call leaves the zeroed slot.
func marLog(w *mcaWorld, args ...uint32) {
	for len(args) < 3 {
		args = append(args, 0)
	}
	sdmStub(w, 0x58aad0, args...)
}

// marStamina is 0x52c500(tgt, stage, 1).
func marStamina(w *mcaWorld, tgt, stage uint32) {
	switch w.u32(tgt) {
	case 1:
		sdmStub(w, 0x52af50, w.u32(tgt+4), stage, 0, 0)
	case 3:
		sdmFatigue(w, w.findMonster(w.u32(tgt+8)), stage, 0)
	}
}

// marAlive is 0x537160(d, 2).
func marAlive(w *mcaWorld, d uint32) bool {
	switch w.u32(d) {
	case 1:
		if w.u32(d+4) == 0xffffffff {
			return false
		}
	case 3:
		if w.u32(d+8) == 0xffffffff {
			return false
		}
	default:
		return false
	}
	return mpnValid(w, d)
}

func marTail(w *mcaWorld, m, t, c, src, res, mode uint32, melee, fumble, b0 bool) uint32 {
	g := w.u32(0x6836a8)
	if w.u32(c+0xd) == 0 {
		panic("monster attack resolution: no swings remaining (0x54102b asserts)")
	}
	w.put(c+0xd, 4, uint64(w.u32(c+0xd)-1))
	if w.u32(m+0x2b) == 0 && w.u32(m+0x107) > 0xe {
		w.put(c+0xd, 4, 0)
	}
	changed := w.u8(src+0x1e) != 0
	report := func() {
		sdmStub(w, 0x590b40, m, mpnStr(w, 0x269), marTargetText(w, src, m+0x2ba))
	}
	if w.u32(c+0xd) != 0 {
		picked := false
		if w.u8(t+0x247) != 0 && melee {
			m17Copy(w, marPick, m+0x2ba, 8)
			if !marRepick(w, m) || w.u32(m+0x2e1) != 0 {
				w.put(c+0xd, 4, 0)
			} else {
				picked = true
				if !marSame(w, marPick, m+0x2ba) {
					m17Report(w, m, res)
					if fumble && w.u8(g+0xa26) != 0 {
						m17Report(w, m, g+marResult)
					}
					srdPutW(w, g+0x6b8, srdWstr(w, mpnStr(w, 0x210)))
					sdmStub(w, 0x541630, m, t, 1)
				}
			}
		}
		if w.u32(c+0xd) != 0 && (marSame(w, marSaved, m+0x2ba) || picked) && marAlive(w, m+0x2ba) {
			if !picked {
				sdmStub(w, 0x53ffe0, m, mode)
			}
			if (!b0 && fumble) || changed {
				m17Report(w, m, res)
				report()
			}
			return 2
		}
	}
	if !b0 {
		m17Report(w, m, res)
		if fumble || changed {
			report()
			if w.u8(g+0xa26) != 0 {
				m17Report(w, m, g+marResult)
			}
		}
	}
	return 3
}

// marArhit is 0x542720(M, A, mode, flag) on its parked world with the live
// attacker, attack and target fields.
func marArhit(w *mcaWorld, m, a, mode, flag uint32) int32 {
	k := sdmCall(w, 0x542720, m, a, mode, flag)
	if !w.real {
		return int32(sdmReplay(w, 0x542720, k))
	}
	g := w.u32(0x6836a8)
	d := func(a uint32) int64 { return int64(w.u32(a)) }
	sd := func(a uint32) int64 { return int64(int32(w.u32(a))) }
	b := func(a uint32) int64 { return int64(w.u8(a)) }
	sb := func(a uint32) int64 { return int64(int8(w.u8(a))) }
	x := append([]int64(nil), marPark542720[:]...)
	x[0], x[1], x[3] = d(m+0x2ba), int64(mode), b(0x683f94)
	x[4], x[5] = b(g+0x9a4), b(g+0x9a5)
	x[15], x[16], x[17], x[18] = d(g+0x918), d(g+0x90c), d(g+0x910), d(w.u32(0x652dac))
	x[24], x[28], x[29], x[30], x[31] = d(m+0x2c2), int64(g), sb(m+0x1dc), b(a+1), b(a+4)
	x[32] = d(m + 0x242)
	for i := uint32(0); i < 4; i++ {
		x[33+i] = b(m + 0x248 + i)
	}
	x[37], x[38], x[39], x[40], x[41] = d(w.u32(m+0x10)+0x141), d(m+0x87), b(m+0x16), d(0x6850d5), int64(byte(flag))
	switch w.u32(m + 0x2ba) {
	case 1:
		i := w.u32(m + 0x2be)
		c := w.charBase(i)
		x[2], x[6], x[7], x[8], x[19], x[20] = int64(i), sd(c+0xe91), sd(c+0xe99), b(c+0x17b5), d(c+0xb11), d(c+0xb01)
		x[21], x[22], x[23], x[26] = b(g+0x4c+i*0xd4), d(w.u32(0x685178)+1+i*0x106), d(w.u32(0x685178)+0x3d+i*0x106), sd(c+0xe85)
		for j, o := range []uint32{0xfd, 0x139, 0x14d, 0x161} {
			x[42+j] = sd(c + o)
		}
	case 3:
		t := w.findMonster(w.u32(m + 0x2c2))
		x[9], x[10], x[11], x[12], x[13] = sb(t+0x226), sb(t+0x1df), d(t+0x242), d(t+0x2e1), b(t+0x220)
		x[25], x[14], x[27] = d(t+8), b(w.template(t)+0xd0), sb(w.template(t)+0x15d)
		for j := uint32(0); j < 4; j++ {
			x[46+j] = b(t + 0x248 + j)
		}
	}
	return int32(monsterArhitNode(x, w.r)[0])
}

// marFumbleList is 0x544010(M, T, idx, list). F runs the node on its parked
// world with the live attacker and keeps only the candidate count: the list
// entries stay the fixture's candidate records (fe0..fe7).
func marFumbleList(w *mcaWorld, m, t, idx, list uint32) {
	k := sdmCall(w, 0x544010, m, t, idx, list)
	if !w.real {
		sdmReplay(w, 0x544010, k, m, t, idx, list)
		return
	}
	x := append([]int64(nil), marPark544010[:]...)
	for i, o := range []uint32{0x2ba, 0x2be, 0x2c2, 0} {
		x[i] = int64(w.u32(m + o))
	}
	// The attacker's active byte M+0x14 is not part of 0x540170's world; it
	// stays the parked 1 (an attacking monster is active).
	x[6], x[7] = int64(w.u32(m+0x2b)), int64(w.u8(m+0x15))
	x[82], x[83], x[84] = int64(w.u8(t+0x24a)), int64(idx), int64(w.u32(m+0x2e5))
	x[12], x[13] = int64(w.u32(list+4)), int64(w.u32(list+8))
	n := uint32(monsterFumbleTargetListNode(x, w.r)[1])
	if n > 8 {
		panic("monster attack resolution: fumble candidates past the fixture's 8 records")
	}
	w.put(list, 4, uint64(w.u32(0x20004658)))
	w.put(list+8, 4, uint64(n))
}

// marIntercept is 0x545e50(src, tgt): F runs the node on its parked party
// with the live source and target and writes the target back.
func marIntercept(w *mcaWorld, src, tgt uint32) {
	k := sdmCall(w, 0x545e50, src, tgt)
	if !w.real {
		sdmReplay(w, 0x545e50, k, src, tgt)
		return
	}
	x := append([]int64(nil), marPark545e50[:]...)
	x[0], x[1], x[2] = int64(w.u32(src)), int64(int32(w.u32(src+4))), int64(w.u32(src+8))
	for i := uint32(0); i < 8; i++ {
		x[3+i] = int64(w.u32(tgt + 4*i))
	}
	o := attackInterceptNode(x, w.r)
	for i := uint32(0); i < 8; i++ { // Out 152..159: target_0..target_7
		w.put(tgt+4*i, 4, uint64(uint32(o[152+i])))
	}
}

// marDamage is 0x542960(M, A, &hits): F runs the node on its parked world with
// the live target, attacker and attack dice.
func marDamage(w *mcaWorld, m, a uint32) uint32 {
	k := sdmCall(w, 0x542960, m, a, marHits)
	if !w.real {
		return sdmReplay(w, 0x542960, k, m, a, marHits)
	}
	g := w.u32(0x6836a8)
	x := append([]int64(nil), marPark542960[:]...)
	x[0], x[1], x[2] = int64(w.u32(g+0x8c5)), int64(int32(w.u32(g+0x8c9))), int64(w.u32(g+0x8cd))
	x[3], x[4], x[5] = int64(w.u32(g+0x90c)), int64(w.u32(g+0x910)), int64(w.u32(g+0x918))
	x[6], x[7], x[8], x[9] = int64(w.u8(g+0x9a4)), int64(w.u8(g+0x9a5)), int64(w.u32(m+0x2e5)), int64(w.u32(m+8))
	x[17] = int64(w.u8(w.template(m) + 0x24f))
	switch w.u32(g + 0x8c5) {
	case 1:
		c := w.charBase(w.u32(g + 0x8c9))
		x[18], x[19], x[20] = int64(w.u8(c+0x17b5)), int64(int32(w.u32(c+0xed9))), int64(w.u8(c+0x71b))
	case 3:
		t := w.findMonster(w.u32(g + 0x8cd))
		x[10], x[11], x[12], x[13] = int64(w.u32(t+8)), int64(w.u32(t+0x2e1)), int64(w.u8(t+0x220)), int64(int8(w.u8(t+0x1e1)))
	}
	x[34], x[35], x[36] = int64(int16(w.u16(a+0x17))), int64(w.u8(a+0x19)), int64(w.u8(a+0x1a))
	o := monsterDamageNode(x, w.r)
	w.put(marHits, 4, uint64(uint32(o[1])))
	return uint32(o[0])
}

// marCap is 0x545a00(d): F runs the node on the live target, the monster put
// in the node's first monster slot (id 0).
func marCap(w *mcaWorld, d, tgt uint32) uint32 {
	k := sdmCall(w, 0x545a00, d)
	if !w.real {
		return sdmReplay(w, 0x545a00, k, d)
	}
	x := make([]int64, 14)
	x[0], x[1], x[2] = int64(d), int64(w.u32(tgt)), int64(w.u32(tgt+4))
	for i := uint32(0); i < 8; i++ {
		x[4+i] = int64(w.u32(w.charBase(i) + 0xb0d))
	}
	if w.u32(tgt) == 3 {
		x[12] = int64(w.u32(w.findMonster(w.u32(tgt+8)) + 0x27))
	}
	return uint32(fumbleDamageCapNode(x, w.r)[0])
}

// marRepick is 0x5323f0(M, 0, 1, 0) on its parked world (mchParkPick), as
// pickWith but with the arguments 0x540170 passes.
func marRepick(w *mcaWorld, m uint32) bool {
	k := sdmCall(w, 0x5323f0, m, 0, 1, 0)
	if !w.real {
		return byte(sdmReplay(w, 0x5323f0, k, m, 0, 1, 0)) != 0
	}
	c := w.u32(m + 0x10)
	x := append([]int64(nil), mchParkPick[:]...)
	x[0], x[1], x[2], x[3] = 0, 1, 0, 0
	x[4] = int64(w.u8(w.template(m) + 0xe5))
	x[8], x[9] = rpkListPtr, 0
	x[108], x[109], x[110] = int64(w.u32(c+5)), int64(w.u32(c+9)), int64(w.u32(c+0x11))
	o := monsterActionRandomPickNode(x, w.r)
	w.put(m+0x2e1, 4, uint64(uint32(o[0])))
	w.put(m+0x2e5, 4, uint64(uint32(o[1])))
	for i := uint32(0); i < 8; i++ {
		w.put(m+0x2ba+4*i, 4, uint64(uint32(o[2+i])))
	}
	w.put(c+5, 4, uint64(uint32(o[10])))
	w.put(c+9, 4, uint64(uint32(o[11])))
	w.put(c+0x11, 4, uint64(uint32(o[12])))
	return byte(o[13]) != 0
}

func m17Run(real bool) func([]int64, Rand) []int64 {
	return func(in []int64, r Rand) []int64 {
		if len(in) != len(m17Spec.in) {
			panic("monster attack resolution: wrong input count")
		}
		w := &mcaWorld{mstWorld: mstBuild(&m17Spec, in), real: real, tab: m17Stubs}
		w.r = r
		w.put(mstRet, 4, uint64(m17Body(w)))
		out := make([]int64, len(m17Spec.out))
		for i, f := range m17Spec.out {
			out[i] = w.read(f, in)
		}
		return out
	}
}

func monsterAttackResolutionNode(in []int64, r Rand) []int64 { return m17Run(true)(in, r) }

func m17Emu(in []int64, r Rand) []int64 { return m17Run(false)(in, r) }

func init() {
	Nodes["00540170"] = Node{Addr: "00540170", In: m17In, Out: m17Out, F: monsterAttackResolutionNode, Emu: m17Emu}
}

// Code below is generated from the same description as re/emu/cases/00540170.json.

var m17Spec = mpnStrings(mstSpec{
	set: []mstSet{
		{0x10008004, 4, 0x20000400},
		{0x10008008, 4, 0x20000c00},
		{0x683fad, 4, 0x200036a0},
		{0x683fb9, 4, 0x0},
		{0x683fb1, 4, 0x20003640},
		{0x683fbd, 4, 0x0},
		{0x6840c7, 4, 0x20000c00},
		{0x6840cb, 4, 0x20000f00},
		{0x68c09c, 4, 0x20008000},
		{0x68516c, 4, 0x20003800},
		{0x685174, 4, 0x20010000},
		{0x685178, 4, 0x2000c000},
		{0x6836a8, 4, 0x2001d000},
		{0x65be1c, 4, 0x2001f000},
		{0x6850e5, 4, 0x2},
		{0x68ec78, 4, 0x0},
		{0x68edcc, 4, 0x0},
		{0x689b34, 2, 0x0},
		{0x683f97, 1, 0x0},
		{0x683f9b, 1, 0x0},
		{0x6598b8, 4, 0x20002540},
		{0x68c560, 4, 0xc560000},
		{0x68c570, 4, 0xc570000},
		{0x68c574, 4, 0xc574000},
		{0x20002540, 4, 0x5ec078},
		{0x20002400, 4, 0x20002440},
		{0x20002458, 4, 0x20002500},
		{0x20002500, 1, 0x4},
		{0x200036a0, 4, 0x200036c0},
		{0x200036a8, 4, 0x2},
		{0x200036c0, 4, 0x20000400},
		{0x200036c4, 4, 0x20000800},
		{0x20003640, 4, 0x20003660},
		{0x20003648, 4, 0x1},
		{0x20003660, 4, 0x20003600},
		{0x20003600, 4, 0x1},
		{0x20003608, 4, 0x20003680},
		{0x2001da06, 4, 0x4},
		{0x2001da0a, 4, 0x2001e300},
		{0x2001e300, 4, 0x2001e340},
		{0x2001e304, 4, 0x2001e380},
		{0x20004658, 4, 0x2001e000},
		{0x20000400, 4, 0x5},
		{0x20000408, 4, 0x0},
		{0x2000040c, 4, 0x20001200},
		{0x20000410, 4, 0x20003400},
		{0x200004bf, 4, 0x5150},
		{0x200004c3, 2, 0x6162},
		{0x200006df, 1, 0x1},
		{0x20001200, 4, 0x5ed22c},
		{0x200013d8, 4, 0x20001400},
		{0x200013dc, 4, 0x400},
		{0x200019e8, 4, 0x20002600},
		{0x20001608, 4, 0x20002400},
		{0x20000704, 4, 0xdeca0000},
		{0x20000708, 4, 0xdeca0001},
		{0x2000070c, 4, 0xdeca0002},
		{0x20000710, 4, 0xdeca0003},
		{0x20000714, 4, 0xdeca0004},
		{0x20000718, 4, 0xdeca0005},
		{0x2000071c, 4, 0xdeca0006},
		{0x20000720, 4, 0xdeca0007},
		{0x20000724, 4, 0xdeca0008},
		{0x20000728, 4, 0xdeca0009},
		{0x2000072c, 4, 0xdeca000a},
		{0x20000730, 4, 0xdeca000b},
		{0x20000734, 4, 0xdeca000c},
		{0x200006c6, 4, 0x1},
		{0x200006ca, 4, 0x7a000004},
		{0x200006ce, 4, 0x7a000005},
		{0x200006d2, 4, 0x7a000006},
		{0x200006d6, 4, 0x7a000007},
		{0x20000800, 4, 0x6},
		{0x20000808, 4, 0x1},
		{0x2000080c, 4, 0x20001300},
		{0x20000810, 4, 0x20003500},
		{0x200008bf, 4, 0x5151},
		{0x200008c3, 2, 0x6163},
		{0x20000adf, 1, 0x1},
		{0x20001300, 4, 0x5ed22c},
		{0x200014d8, 4, 0x20001c00},
		{0x200014dc, 4, 0x400},
		{0x200021e8, 4, 0x20002600},
		{0x20001e08, 4, 0x20002400},
		{0x20000b04, 4, 0xdeca0100},
		{0x20000b08, 4, 0xdeca0101},
		{0x20000b0c, 4, 0xdeca0102},
		{0x20000b10, 4, 0xdeca0103},
		{0x20000b14, 4, 0xdeca0104},
		{0x20000b18, 4, 0xdeca0105},
		{0x20000b1c, 4, 0xdeca0106},
		{0x20000b20, 4, 0xdeca0107},
		{0x20000b24, 4, 0xdeca0108},
		{0x20000b28, 4, 0xdeca0109},
		{0x20000b2c, 4, 0xdeca010a},
		{0x20000b30, 4, 0xdeca010b},
		{0x20000b34, 4, 0xdeca010c},
		{0x20000ac6, 4, 0x1},
		{0x20000aca, 4, 0x7a000104},
		{0x20000ace, 4, 0x7a000105},
		{0x20000ad2, 4, 0x7a000106},
		{0x20000ad6, 4, 0x7a000107},
		{0x20000c00, 2, 0x41},
		{0x20000c02, 2, 0x62},
		{0x20000c30, 2, 0x41},
		{0x20000c32, 2, 0x62},
		{0x20000c34, 2, 0x73},
		{0x20000c60, 2, 0x43},
		{0x20000c62, 2, 0x64},
		{0x20000c64, 2, 0x65},
		{0x20000c66, 2, 0x66},
		{0x20000c90, 2, 0x43},
		{0x20000c92, 2, 0x64},
		{0x20000c94, 2, 0x65},
		{0x20000c96, 2, 0x66},
		{0x20000c98, 2, 0x73},
		{0x20000f00, 2, 0x47},
		{0x20000f30, 2, 0x47},
		{0x20000f32, 2, 0x78},
		{0x20000f60, 2, 0x48},
		{0x20000f62, 2, 0x69},
		{0x20000f64, 2, 0x6a},
		{0x20000f66, 2, 0x6b},
		{0x20000f68, 2, 0x6c},
		{0x20000f90, 2, 0x48},
		{0x20000f92, 2, 0x69},
		{0x20000f94, 2, 0x6a},
		{0x20000f96, 2, 0x6b},
		{0x20000f98, 2, 0x6c},
		{0x20000f9a, 2, 0x6d},
		{0x2000341a, 4, 0x1000},
		{0x20010005, 2, 0x41},
		{0x2000341e, 4, 0x1001},
		{0x20011867, 2, 0x42},
		{0x20011869, 2, 0x42},
		{0x20003422, 4, 0x1002},
		{0x200130c9, 2, 0x43},
		{0x200130cb, 2, 0x43},
		{0x200130cd, 2, 0x43},
		{0x20003426, 4, 0x1003},
		{0x2001492b, 2, 0x44},
		{0x2001492d, 2, 0x44},
		{0x2001492f, 2, 0x44},
		{0x20014931, 2, 0x44},
		{0x2000342a, 4, 0x1004},
		{0x2001618d, 2, 0x45},
		{0x2001618f, 2, 0x45},
		{0x20016191, 2, 0x45},
		{0x20016193, 2, 0x45},
		{0x20016195, 2, 0x45},
		{0x2000342e, 4, 0x1005},
		{0x200179ef, 2, 0x46},
		{0x200179f1, 2, 0x46},
		{0x200179f3, 2, 0x46},
		{0x200179f5, 2, 0x46},
		{0x200179f7, 2, 0x46},
		{0x200179f9, 2, 0x46},
		{0x20003432, 4, 0x1006},
		{0x20019251, 2, 0x47},
		{0x20019253, 2, 0x47},
		{0x20019255, 2, 0x47},
		{0x20019257, 2, 0x47},
		{0x20019259, 2, 0x47},
		{0x2001925b, 2, 0x47},
		{0x2001925d, 2, 0x47},
		{0x20003436, 4, 0x1007},
		{0x2001aab3, 2, 0x48},
		{0x2001aab5, 2, 0x48},
		{0x2001aab7, 2, 0x48},
		{0x2001aab9, 2, 0x48},
		{0x2001aabb, 2, 0x48},
		{0x2001aabd, 2, 0x48},
		{0x2001aabf, 2, 0x48},
		{0x2001aac1, 2, 0x48},
		{0x2001e000, 4, 0x2001e100},
		{0x2001e10c, 4, 0x7f000003},
		{0x2001e110, 4, 0x7f000004},
		{0x2001e114, 4, 0x7f000005},
		{0x2001e118, 4, 0x7f000006},
		{0x2001e11c, 4, 0x7f000007},
		{0x2001e004, 4, 0x2001e120},
		{0x2001e12c, 4, 0x7f000103},
		{0x2001e130, 4, 0x7f000104},
		{0x2001e134, 4, 0x7f000105},
		{0x2001e138, 4, 0x7f000106},
		{0x2001e13c, 4, 0x7f000107},
		{0x2001e008, 4, 0x2001e140},
		{0x2001e14c, 4, 0x7f000203},
		{0x2001e150, 4, 0x7f000204},
		{0x2001e154, 4, 0x7f000205},
		{0x2001e158, 4, 0x7f000206},
		{0x2001e15c, 4, 0x7f000207},
		{0x2001e00c, 4, 0x2001e160},
		{0x2001e16c, 4, 0x7f000303},
		{0x2001e170, 4, 0x7f000304},
		{0x2001e174, 4, 0x7f000305},
		{0x2001e178, 4, 0x7f000306},
		{0x2001e17c, 4, 0x7f000307},
		{0x2001e010, 4, 0x2001e180},
		{0x2001e18c, 4, 0x7f000403},
		{0x2001e190, 4, 0x7f000404},
		{0x2001e194, 4, 0x7f000405},
		{0x2001e198, 4, 0x7f000406},
		{0x2001e19c, 4, 0x7f000407},
		{0x2001e014, 4, 0x2001e1a0},
		{0x2001e1ac, 4, 0x7f000503},
		{0x2001e1b0, 4, 0x7f000504},
		{0x2001e1b4, 4, 0x7f000505},
		{0x2001e1b8, 4, 0x7f000506},
		{0x2001e1bc, 4, 0x7f000507},
		{0x2001e018, 4, 0x2001e1c0},
		{0x2001e1cc, 4, 0x7f000603},
		{0x2001e1d0, 4, 0x7f000604},
		{0x2001e1d4, 4, 0x7f000605},
		{0x2001e1d8, 4, 0x7f000606},
		{0x2001e1dc, 4, 0x7f000607},
		{0x2001e01c, 4, 0x2001e1e0},
		{0x2001e1ec, 4, 0x7f000703},
		{0x2001e1f0, 4, 0x7f000704},
		{0x2001e1f4, 4, 0x7f000705},
		{0x2001e1f8, 4, 0x7f000706},
		{0x2001e1fc, 4, 0x7f000707},
		{0x2001e348, 2, 0x45},
		{0x2001e34a, 2, 0x76},
		{0x2001e34c, 2, 0x30},
		{0x2001e388, 2, 0x45},
		{0x2001e38a, 2, 0x76},
		{0x2001e38c, 2, 0x31},
		{0x2001da34, 4, 0x7e000003},
		{0x2001da38, 4, 0x7e000004},
		{0x2001da3c, 4, 0x7e000005},
		{0x2001da40, 4, 0x7e000006},
		{0x2001da44, 4, 0x7e000007},
		{0x6875c3, 2, 0x7700},
		{0x6875c5, 2, 0x7701},
		{0x6875c7, 2, 0x7702},
		{0x6875c9, 2, 0x7703},
		{0x6875cb, 2, 0x7704},
		{0x6875cd, 2, 0x7705},
		{0x6875cf, 2, 0x7706},
		{0x6875d1, 2, 0x7707},
		{0x6875d3, 2, 0x7708},
		{0x6875d5, 2, 0x7709},
		{0x6875d7, 2, 0x770a},
		{0x6875d9, 2, 0x770b},
		{0x689fd0, 2, 0xbe00},
		{0x689fd2, 2, 0xbe01},
		{0x689fd4, 2, 0xbe02},
		{0x689fd6, 2, 0xbe03},
		{0x689fd8, 2, 0xbe04},
		{0x689fda, 2, 0xbe05},
		{0x689fdc, 2, 0xbe06},
		{0x689fde, 2, 0xbe07},
		{0x689fe0, 2, 0xbe08},
		{0x689fe2, 2, 0xbe09},
		{0x689fe4, 2, 0xbe0a},
		{0x689fe6, 2, 0xbe0b},
		{0x689fe8, 2, 0xbe0c},
		{0x689fea, 2, 0xbe0d},
		{0x689fec, 2, 0xbe0e},
		{0x689fee, 2, 0xbe0f},
		{0x689ff0, 2, 0xbe10},
		{0x689ff2, 2, 0xbe11},
		{0x689ff4, 2, 0xbe12},
		{0x689ff6, 2, 0xbe13},
		{0x689ff8, 2, 0xbe14},
		{0x689ffa, 2, 0xbe15},
		{0x689ffc, 2, 0xbe16},
		{0x689ffe, 2, 0xbe17},
	},
	in: []mstFld{
		{addr: 0x68510c, size: 1, signed: false, idx: -1},
		{addr: 0x6850d5, size: 1, signed: false, idx: -1},
		{addr: 0x68d810, size: 1, signed: false, idx: -1},
		{addr: 0x689b32, size: 1, signed: false, idx: -1},
		{addr: 0x6875ef, size: 4, signed: false, idx: -1},
		{addr: 0x2001d8c4, size: 1, signed: false, idx: -1},
		{addr: 0x2001d9a4, size: 1, signed: false, idx: -1},
		{addr: 0x2001d9a5, size: 1, signed: false, idx: -1},
		{addr: 0x2001d9a6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9aa, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ae, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9b2, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9b6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ba, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9be, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9c2, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9c6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ca, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ce, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9d2, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9d6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9da, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9de, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9e2, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9e6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ea, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ee, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9f2, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9f6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9fa, size: 4, signed: false, idx: -1},
		{addr: 0x2001da02, size: 4, signed: false, idx: -1},
		{addr: 0x2001da0e, size: 4, signed: false, idx: -1},
		{addr: 0x2001da12, size: 4, signed: false, idx: -1},
		{addr: 0x2001da16, size: 4, signed: false, idx: -1},
		{addr: 0x2001da1a, size: 4, signed: false, idx: -1},
		{addr: 0x2001da1e, size: 4, signed: false, idx: -1},
		{addr: 0x2001da26, size: 1, signed: false, idx: -1},
		{addr: 0x2001da27, size: 1, signed: false, idx: -1},
		{addr: 0x2001da28, size: 4, signed: false, idx: -1},
		{addr: 0x2001da2c, size: 4, signed: false, idx: -1},
		{addr: 0x2001da30, size: 4, signed: false, idx: -1},
		{addr: 0x2001e340, size: 4, signed: false, idx: -1},
		{addr: 0x2001e344, size: 4, signed: false, idx: -1},
		{addr: 0x2001e380, size: 4, signed: false, idx: -1},
		{addr: 0x2001e384, size: 4, signed: false, idx: -1},
		{addr: 0x2001033f, size: 1, signed: false, idx: -1},
		{addr: 0x20010281, size: 1, signed: false, idx: -1},
		{addr: 0x20010e99, size: 4, signed: true, idx: -1},
		{addr: 0x20010741, size: 1, signed: false, idx: -1},
		{addr: 0x20010b11, size: 4, signed: false, idx: -1},
		{addr: 0x20010b0d, size: 4, signed: false, idx: -1},
		{addr: 0x20010fb1, size: 4, signed: true, idx: -1},
		{addr: 0x200100fd, size: 4, signed: true, idx: -1},
		{addr: 0x20010139, size: 4, signed: true, idx: -1},
		{addr: 0x2001014d, size: 4, signed: true, idx: -1},
		{addr: 0x20010161, size: 4, signed: true, idx: -1},
		{addr: 0x20010a89, size: 4, signed: false, idx: -1},
		{addr: 0x20010a91, size: 4, signed: false, idx: -1},
		{addr: 0x20010a49, size: 4, signed: false, idx: -1},
		{addr: 0x20011830, size: 1, signed: false, idx: -1},
		{addr: 0x20010075, size: 4, signed: false, idx: -1},
		{addr: 0x20010f5d, size: 4, signed: true, idx: -1},
		{addr: 0x20010f8d, size: 4, signed: true, idx: -1},
		{addr: 0x20010f99, size: 4, signed: true, idx: -1},
		{addr: 0x20010fd5, size: 4, signed: true, idx: -1},
		{addr: 0x20010fe1, size: 4, signed: true, idx: -1},
		{addr: 0x20011ba1, size: 1, signed: false, idx: -1},
		{addr: 0x20011ae3, size: 1, signed: false, idx: -1},
		{addr: 0x200126fb, size: 4, signed: true, idx: -1},
		{addr: 0x20011fa3, size: 1, signed: false, idx: -1},
		{addr: 0x20012373, size: 4, signed: false, idx: -1},
		{addr: 0x2001236f, size: 4, signed: false, idx: -1},
		{addr: 0x20012813, size: 4, signed: true, idx: -1},
		{addr: 0x2001195f, size: 4, signed: true, idx: -1},
		{addr: 0x2001199b, size: 4, signed: true, idx: -1},
		{addr: 0x200119af, size: 4, signed: true, idx: -1},
		{addr: 0x200119c3, size: 4, signed: true, idx: -1},
		{addr: 0x200122eb, size: 4, signed: false, idx: -1},
		{addr: 0x200122f3, size: 4, signed: false, idx: -1},
		{addr: 0x200122ab, size: 4, signed: false, idx: -1},
		{addr: 0x20013092, size: 1, signed: false, idx: -1},
		{addr: 0x200118d7, size: 4, signed: false, idx: -1},
		{addr: 0x200127bf, size: 4, signed: true, idx: -1},
		{addr: 0x200127ef, size: 4, signed: true, idx: -1},
		{addr: 0x200127fb, size: 4, signed: true, idx: -1},
		{addr: 0x20012837, size: 4, signed: true, idx: -1},
		{addr: 0x20012843, size: 4, signed: true, idx: -1},
		{addr: 0x20013403, size: 1, signed: false, idx: -1},
		{addr: 0x20013345, size: 1, signed: false, idx: -1},
		{addr: 0x20013f5d, size: 4, signed: true, idx: -1},
		{addr: 0x20013805, size: 1, signed: false, idx: -1},
		{addr: 0x20013bd5, size: 4, signed: false, idx: -1},
		{addr: 0x20013bd1, size: 4, signed: false, idx: -1},
		{addr: 0x20014075, size: 4, signed: true, idx: -1},
		{addr: 0x200131c1, size: 4, signed: true, idx: -1},
		{addr: 0x200131fd, size: 4, signed: true, idx: -1},
		{addr: 0x20013211, size: 4, signed: true, idx: -1},
		{addr: 0x20013225, size: 4, signed: true, idx: -1},
		{addr: 0x20013b4d, size: 4, signed: false, idx: -1},
		{addr: 0x20013b55, size: 4, signed: false, idx: -1},
		{addr: 0x20013b0d, size: 4, signed: false, idx: -1},
		{addr: 0x200148f4, size: 1, signed: false, idx: -1},
		{addr: 0x20013139, size: 4, signed: false, idx: -1},
		{addr: 0x20014021, size: 4, signed: true, idx: -1},
		{addr: 0x20014051, size: 4, signed: true, idx: -1},
		{addr: 0x2001405d, size: 4, signed: true, idx: -1},
		{addr: 0x20014099, size: 4, signed: true, idx: -1},
		{addr: 0x200140a5, size: 4, signed: true, idx: -1},
		{addr: 0x20014c65, size: 1, signed: false, idx: -1},
		{addr: 0x20014ba7, size: 1, signed: false, idx: -1},
		{addr: 0x200157bf, size: 4, signed: true, idx: -1},
		{addr: 0x20015067, size: 1, signed: false, idx: -1},
		{addr: 0x20015437, size: 4, signed: false, idx: -1},
		{addr: 0x20015433, size: 4, signed: false, idx: -1},
		{addr: 0x200158d7, size: 4, signed: true, idx: -1},
		{addr: 0x20014a23, size: 4, signed: true, idx: -1},
		{addr: 0x20014a5f, size: 4, signed: true, idx: -1},
		{addr: 0x20014a73, size: 4, signed: true, idx: -1},
		{addr: 0x20014a87, size: 4, signed: true, idx: -1},
		{addr: 0x200153af, size: 4, signed: false, idx: -1},
		{addr: 0x200153b7, size: 4, signed: false, idx: -1},
		{addr: 0x2001536f, size: 4, signed: false, idx: -1},
		{addr: 0x20016156, size: 1, signed: false, idx: -1},
		{addr: 0x2001499b, size: 4, signed: false, idx: -1},
		{addr: 0x20015883, size: 4, signed: true, idx: -1},
		{addr: 0x200158b3, size: 4, signed: true, idx: -1},
		{addr: 0x200158bf, size: 4, signed: true, idx: -1},
		{addr: 0x200158fb, size: 4, signed: true, idx: -1},
		{addr: 0x20015907, size: 4, signed: true, idx: -1},
		{addr: 0x200164c7, size: 1, signed: false, idx: -1},
		{addr: 0x20016409, size: 1, signed: false, idx: -1},
		{addr: 0x20017021, size: 4, signed: true, idx: -1},
		{addr: 0x200168c9, size: 1, signed: false, idx: -1},
		{addr: 0x20016c99, size: 4, signed: false, idx: -1},
		{addr: 0x20016c95, size: 4, signed: false, idx: -1},
		{addr: 0x20017139, size: 4, signed: true, idx: -1},
		{addr: 0x20016285, size: 4, signed: true, idx: -1},
		{addr: 0x200162c1, size: 4, signed: true, idx: -1},
		{addr: 0x200162d5, size: 4, signed: true, idx: -1},
		{addr: 0x200162e9, size: 4, signed: true, idx: -1},
		{addr: 0x20016c11, size: 4, signed: false, idx: -1},
		{addr: 0x20016c19, size: 4, signed: false, idx: -1},
		{addr: 0x20016bd1, size: 4, signed: false, idx: -1},
		{addr: 0x200179b8, size: 1, signed: false, idx: -1},
		{addr: 0x200161fd, size: 4, signed: false, idx: -1},
		{addr: 0x200170e5, size: 4, signed: true, idx: -1},
		{addr: 0x20017115, size: 4, signed: true, idx: -1},
		{addr: 0x20017121, size: 4, signed: true, idx: -1},
		{addr: 0x2001715d, size: 4, signed: true, idx: -1},
		{addr: 0x20017169, size: 4, signed: true, idx: -1},
		{addr: 0x20017d29, size: 1, signed: false, idx: -1},
		{addr: 0x20017c6b, size: 1, signed: false, idx: -1},
		{addr: 0x20018883, size: 4, signed: true, idx: -1},
		{addr: 0x2001812b, size: 1, signed: false, idx: -1},
		{addr: 0x200184fb, size: 4, signed: false, idx: -1},
		{addr: 0x200184f7, size: 4, signed: false, idx: -1},
		{addr: 0x2001899b, size: 4, signed: true, idx: -1},
		{addr: 0x20017ae7, size: 4, signed: true, idx: -1},
		{addr: 0x20017b23, size: 4, signed: true, idx: -1},
		{addr: 0x20017b37, size: 4, signed: true, idx: -1},
		{addr: 0x20017b4b, size: 4, signed: true, idx: -1},
		{addr: 0x20018473, size: 4, signed: false, idx: -1},
		{addr: 0x2001847b, size: 4, signed: false, idx: -1},
		{addr: 0x20018433, size: 4, signed: false, idx: -1},
		{addr: 0x2001921a, size: 1, signed: false, idx: -1},
		{addr: 0x20017a5f, size: 4, signed: false, idx: -1},
		{addr: 0x20018947, size: 4, signed: true, idx: -1},
		{addr: 0x20018977, size: 4, signed: true, idx: -1},
		{addr: 0x20018983, size: 4, signed: true, idx: -1},
		{addr: 0x200189bf, size: 4, signed: true, idx: -1},
		{addr: 0x200189cb, size: 4, signed: true, idx: -1},
		{addr: 0x2001958b, size: 1, signed: false, idx: -1},
		{addr: 0x200194cd, size: 1, signed: false, idx: -1},
		{addr: 0x2001a0e5, size: 4, signed: true, idx: -1},
		{addr: 0x2001998d, size: 1, signed: false, idx: -1},
		{addr: 0x20019d5d, size: 4, signed: false, idx: -1},
		{addr: 0x20019d59, size: 4, signed: false, idx: -1},
		{addr: 0x2001a1fd, size: 4, signed: true, idx: -1},
		{addr: 0x20019349, size: 4, signed: true, idx: -1},
		{addr: 0x20019385, size: 4, signed: true, idx: -1},
		{addr: 0x20019399, size: 4, signed: true, idx: -1},
		{addr: 0x200193ad, size: 4, signed: true, idx: -1},
		{addr: 0x20019cd5, size: 4, signed: false, idx: -1},
		{addr: 0x20019cdd, size: 4, signed: false, idx: -1},
		{addr: 0x20019c95, size: 4, signed: false, idx: -1},
		{addr: 0x2001aa7c, size: 1, signed: false, idx: -1},
		{addr: 0x200192c1, size: 4, signed: false, idx: -1},
		{addr: 0x2001a1a9, size: 4, signed: true, idx: -1},
		{addr: 0x2001a1d9, size: 4, signed: true, idx: -1},
		{addr: 0x2001a1e5, size: 4, signed: true, idx: -1},
		{addr: 0x2001a221, size: 4, signed: true, idx: -1},
		{addr: 0x2001a22d, size: 4, signed: true, idx: -1},
		{addr: 0x2001aded, size: 1, signed: false, idx: -1},
		{addr: 0x2001ad2f, size: 1, signed: false, idx: -1},
		{addr: 0x2001b947, size: 4, signed: true, idx: -1},
		{addr: 0x2001b1ef, size: 1, signed: false, idx: -1},
		{addr: 0x2001b5bf, size: 4, signed: false, idx: -1},
		{addr: 0x2001b5bb, size: 4, signed: false, idx: -1},
		{addr: 0x2001ba5f, size: 4, signed: true, idx: -1},
		{addr: 0x2001abab, size: 4, signed: true, idx: -1},
		{addr: 0x2001abe7, size: 4, signed: true, idx: -1},
		{addr: 0x2001abfb, size: 4, signed: true, idx: -1},
		{addr: 0x2001ac0f, size: 4, signed: true, idx: -1},
		{addr: 0x2001b537, size: 4, signed: false, idx: -1},
		{addr: 0x2001b53f, size: 4, signed: false, idx: -1},
		{addr: 0x2001b4f7, size: 4, signed: false, idx: -1},
		{addr: 0x2001c2de, size: 1, signed: false, idx: -1},
		{addr: 0x2001ab23, size: 4, signed: false, idx: -1},
		{addr: 0x2001ba0b, size: 4, signed: true, idx: -1},
		{addr: 0x2001ba3b, size: 4, signed: true, idx: -1},
		{addr: 0x2001ba47, size: 4, signed: true, idx: -1},
		{addr: 0x2001ba83, size: 4, signed: true, idx: -1},
		{addr: 0x2001ba8f, size: 4, signed: true, idx: -1},
		{addr: 0x20024625, size: 1, signed: false, idx: -1},
		{addr: 0x200038c1, size: 1, signed: false, idx: -1},
		{addr: 0x200039ce, size: 1, signed: false, idx: -1},
		{addr: 0x20003adb, size: 1, signed: false, idx: -1},
		{addr: 0x20003be8, size: 1, signed: false, idx: -1},
		{addr: 0x2000c0f1, size: 1, signed: false, idx: -1},
		{addr: 0x2000c1f7, size: 1, signed: false, idx: -1},
		{addr: 0x2000c2fd, size: 1, signed: false, idx: -1},
		{addr: 0x2000c403, size: 1, signed: false, idx: -1},
		{addr: 0x2000c509, size: 1, signed: false, idx: -1},
		{addr: 0x2000c60f, size: 1, signed: false, idx: -1},
		{addr: 0x2000c715, size: 1, signed: false, idx: -1},
		{addr: 0x2000c81b, size: 1, signed: false, idx: -1},
		{addr: 0x2001f12f, size: 4, signed: true, idx: -1},
		{addr: 0x2001f2ee, size: 4, signed: true, idx: -1},
		{addr: 0x2001f4ad, size: 4, signed: true, idx: -1},
		{addr: 0x2001f66c, size: 4, signed: true, idx: -1},
		{addr: 0x2001f82b, size: 4, signed: true, idx: -1},
		{addr: 0x2001f9ea, size: 4, signed: true, idx: -1},
		{addr: 0x2001fba9, size: 4, signed: true, idx: -1},
		{addr: 0x2001fd68, size: 4, signed: true, idx: -1},
		{addr: 0x2001ff27, size: 4, signed: true, idx: -1},
		{addr: 0x20003411, size: 4, signed: false, idx: -1},
		{addr: 0x2000340d, size: 4, signed: false, idx: -1},
		{addr: 0x200006e5, size: 4, signed: false, idx: -1},
		{addr: 0x200006e1, size: 4, signed: false, idx: -1},
		{addr: 0x200006ba, size: 4, signed: false, idx: -1},
		{addr: 0x200006be, size: 4, signed: false, idx: -1},
		{addr: 0x200006c2, size: 4, signed: false, idx: -1},
		{addr: 0x200005dc, size: 1, signed: true, idx: -1},
		{addr: 0x200005dd, size: 1, signed: true, idx: -1},
		{addr: 0x20000642, size: 4, signed: false, idx: -1},
		{addr: 0x20000648, size: 1, signed: false, idx: -1},
		{addr: 0x20000649, size: 1, signed: false, idx: -1},
		{addr: 0x2000064a, size: 1, signed: false, idx: -1},
		{addr: 0x2000064b, size: 1, signed: false, idx: -1},
		{addr: 0x20000416, size: 1, signed: false, idx: -1},
		{addr: 0x20000507, size: 4, signed: false, idx: -1},
		{addr: 0x20000d87, size: 2, signed: false, idx: -1},
		{addr: 0x20000d5e, size: 1, signed: false, idx: -1},
		{addr: 0x20000d5f, size: 1, signed: false, idx: -1},
		{addr: 0x20000d60, size: 1, signed: false, idx: -1},
		{addr: 0x20000d61, size: 1, signed: false, idx: -1},
		{addr: 0x20000d62, size: 1, signed: false, idx: -1},
		{addr: 0x20000d63, size: 1, signed: false, idx: -1},
		{addr: 0x20000d64, size: 1, signed: false, idx: -1},
		{addr: 0x20000d65, size: 1, signed: false, idx: -1},
		{addr: 0x20000cd0, size: 1, signed: false, idx: -1},
		{addr: 0x20000e63, size: 1, signed: false, idx: -1},
		{addr: 0x20000ccc, size: 1, signed: false, idx: -1},
		{addr: 0x20001087, size: 2, signed: false, idx: -1},
		{addr: 0x2000105e, size: 1, signed: false, idx: -1},
		{addr: 0x2000105f, size: 1, signed: false, idx: -1},
		{addr: 0x20001060, size: 1, signed: false, idx: -1},
		{addr: 0x20001061, size: 1, signed: false, idx: -1},
		{addr: 0x20001062, size: 1, signed: false, idx: -1},
		{addr: 0x20001063, size: 1, signed: false, idx: -1},
		{addr: 0x20001064, size: 1, signed: false, idx: -1},
		{addr: 0x20001065, size: 1, signed: false, idx: -1},
		{addr: 0x20000fd0, size: 1, signed: false, idx: -1},
		{addr: 0x20001163, size: 1, signed: false, idx: -1},
		{addr: 0x20000fcc, size: 1, signed: false, idx: -1},
		{addr: 0x20000d57, size: 1, signed: false, idx: -1},
		{addr: 0x20000d58, size: 1, signed: false, idx: -1},
		{addr: 0x20000d59, size: 1, signed: false, idx: -1},
		{addr: 0x20000d5a, size: 1, signed: false, idx: -1},
		{addr: 0x20000d5b, size: 1, signed: false, idx: -1},
		{addr: 0x20000e47, size: 1, signed: false, idx: -1},
		{addr: 0x20000e4f, size: 1, signed: false, idx: -1},
		{addr: 0x20000ce7, size: 1, signed: false, idx: -1},
		{addr: 0x20000ce8, size: 1, signed: false, idx: -1},
		{addr: 0x20000cea, size: 1, signed: false, idx: -1},
		{addr: 0x20000ceb, size: 1, signed: false, idx: -1},
		{addr: 0x20000cec, size: 4, signed: false, idx: -1},
		{addr: 0x20000cf0, size: 4, signed: false, idx: -1},
		{addr: 0x20000cf4, size: 4, signed: false, idx: -1},
		{addr: 0x20000cf8, size: 4, signed: false, idx: -1},
		{addr: 0x20000d02, size: 1, signed: false, idx: -1},
		{addr: 0x20000d03, size: 1, signed: true, idx: -1},
		{addr: 0x20000d09, size: 1, signed: false, idx: -1},
		{addr: 0x20000d0a, size: 1, signed: false, idx: -1},
		{addr: 0x20000d0c, size: 1, signed: false, idx: -1},
		{addr: 0x20000d0d, size: 1, signed: false, idx: -1},
		{addr: 0x20000d0e, size: 4, signed: false, idx: -1},
		{addr: 0x20000d12, size: 4, signed: false, idx: -1},
		{addr: 0x20000d16, size: 4, signed: false, idx: -1},
		{addr: 0x20000d1a, size: 4, signed: false, idx: -1},
		{addr: 0x20000d24, size: 1, signed: false, idx: -1},
		{addr: 0x20000d25, size: 1, signed: true, idx: -1},
		{addr: 0x20000d2b, size: 1, signed: false, idx: -1},
		{addr: 0x20000d2c, size: 1, signed: false, idx: -1},
		{addr: 0x20000d2e, size: 1, signed: false, idx: -1},
		{addr: 0x20000d2f, size: 1, signed: false, idx: -1},
		{addr: 0x20000d30, size: 4, signed: false, idx: -1},
		{addr: 0x20000d34, size: 4, signed: false, idx: -1},
		{addr: 0x20000d38, size: 4, signed: false, idx: -1},
		{addr: 0x20000d3c, size: 4, signed: false, idx: -1},
		{addr: 0x20000d46, size: 1, signed: false, idx: -1},
		{addr: 0x20000d47, size: 1, signed: true, idx: -1},
		{addr: 0x20000404, size: 4, signed: false, idx: -1},
		{addr: 0x20000415, size: 1, signed: false, idx: -1},
		{addr: 0x2000042b, size: 4, signed: false, idx: -1},
		{addr: 0x2000042f, size: 4, signed: false, idx: -1},
		{addr: 0x20000433, size: 4, signed: false, idx: -1},
		{addr: 0x2000049b, size: 4, signed: false, idx: -1},
		{addr: 0x200004c5, size: 2, signed: false, idx: -1},
		{addr: 0x200004c7, size: 4, signed: false, idx: -1},
		{addr: 0x200006ab, size: 1, signed: false, idx: -1},
		{addr: 0x2000049f, size: 4, signed: false, idx: -1},
		{addr: 0x200004cb, size: 4, signed: false, idx: -1},
		{addr: 0x200004d3, size: 4, signed: false, idx: -1},
		{addr: 0x2000064e, size: 1, signed: false, idx: -1},
		{addr: 0x200014a7, size: 1, signed: false, idx: -1},
		{addr: 0x200014a4, size: 1, signed: false, idx: -1},
		{addr: 0x20000804, size: 4, signed: false, idx: -1},
		{addr: 0x20000815, size: 1, signed: false, idx: -1},
		{addr: 0x2000082b, size: 4, signed: false, idx: -1},
		{addr: 0x2000082f, size: 4, signed: false, idx: -1},
		{addr: 0x20000833, size: 4, signed: false, idx: -1},
		{addr: 0x2000089b, size: 4, signed: false, idx: -1},
		{addr: 0x200008c5, size: 2, signed: false, idx: -1},
		{addr: 0x200008c7, size: 4, signed: false, idx: -1},
		{addr: 0x20000aab, size: 1, signed: false, idx: -1},
		{addr: 0x2000089f, size: 4, signed: false, idx: -1},
		{addr: 0x200008cb, size: 4, signed: false, idx: -1},
		{addr: 0x200008d3, size: 4, signed: false, idx: -1},
		{addr: 0x20000a4e, size: 1, signed: false, idx: -1},
		{addr: 0x20000a48, size: 1, signed: false, idx: -1},
		{addr: 0x20000a49, size: 1, signed: false, idx: -1},
		{addr: 0x20000a4a, size: 1, signed: false, idx: -1},
		{addr: 0x20000a4b, size: 1, signed: false, idx: -1},
		{addr: 0x20001ca7, size: 1, signed: false, idx: -1},
		{addr: 0x20001ca4, size: 1, signed: false, idx: -1},
		{addr: 0x2000362c, size: 1, signed: false, idx: -1},
		{addr: 0x20003604, size: 4, signed: false, idx: -1},
		{addr: 0x20003618, size: 4, signed: false, idx: -1},
		{addr: 0x2001e100, size: 4, signed: false, idx: -1},
		{addr: 0x2001e104, size: 4, signed: false, idx: -1},
		{addr: 0x2001e108, size: 4, signed: false, idx: -1},
		{addr: 0x2001e120, size: 4, signed: false, idx: -1},
		{addr: 0x2001e124, size: 4, signed: false, idx: -1},
		{addr: 0x2001e128, size: 4, signed: false, idx: -1},
		{addr: 0x2001e140, size: 4, signed: false, idx: -1},
		{addr: 0x2001e144, size: 4, signed: false, idx: -1},
		{addr: 0x2001e148, size: 4, signed: false, idx: -1},
		{addr: 0x2001e160, size: 4, signed: false, idx: -1},
		{addr: 0x2001e164, size: 4, signed: false, idx: -1},
		{addr: 0x2001e168, size: 4, signed: false, idx: -1},
		{addr: 0x2001e180, size: 4, signed: false, idx: -1},
		{addr: 0x2001e184, size: 4, signed: false, idx: -1},
		{addr: 0x2001e188, size: 4, signed: false, idx: -1},
		{addr: 0x2001e1a0, size: 4, signed: false, idx: -1},
		{addr: 0x2001e1a4, size: 4, signed: false, idx: -1},
		{addr: 0x2001e1a8, size: 4, signed: false, idx: -1},
		{addr: 0x2001e1c0, size: 4, signed: false, idx: -1},
		{addr: 0x2001e1c4, size: 4, signed: false, idx: -1},
		{addr: 0x2001e1c8, size: 4, signed: false, idx: -1},
		{addr: 0x2001e1e0, size: 4, signed: false, idx: -1},
		{addr: 0x2001e1e4, size: 4, signed: false, idx: -1},
		{addr: 0x2001e1e8, size: 4, signed: false, idx: -1},
		{addr: 0x20004000, size: 1, signed: false, idx: -1},
		{addr: 0x2000462c, size: 4, signed: false, idx: -1},
		{addr: 0x20004644, size: 4, signed: true, idx: -1},
		{addr: 0x20004648, size: 4, signed: true, idx: -1},
		{addr: 0x2000465c, size: 4, signed: false, idx: -1},
		{addr: 0x20004660, size: 4, signed: false, idx: -1},
		{addr: 0x2000466c, size: 4, signed: false, idx: -1},
		{addr: 0x20004670, size: 4, signed: false, idx: -1},
		{addr: 0x20004674, size: 4, signed: false, idx: -1},
		{addr: 0x20004678, size: 4, signed: false, idx: -1},
		{addr: 0x20004680, size: 4, signed: false, idx: -1},
		{addr: 0x20004684, size: 4, signed: false, idx: -1},
		{addr: 0x20004694, size: 4, signed: true, idx: -1},
		{addr: 0x200046a4, size: 4, signed: false, idx: -1},
		{addr: 0x200046a8, size: 4, signed: false, idx: -1},
		{addr: 0x200046b4, size: 4, signed: false, idx: -1},
		{addr: 0x200046c0, size: 4, signed: false, idx: -1},
		{addr: 0x200046f4, size: 4, signed: false, idx: -1},
		{addr: 0x2000470c, size: 4, signed: false, idx: -1},
		{addr: 0x2000471c, size: 4, signed: false, idx: -1},
		{addr: 0x2000472c, size: 1, signed: false, idx: -1},
		{addr: 0x20004730, size: 4, signed: false, idx: -1},
		{addr: 0x20004734, size: 4, signed: false, idx: -1},
		{addr: 0x20004738, size: 4, signed: false, idx: -1},
		{addr: 0x2000473c, size: 4, signed: false, idx: -1},
		{addr: 0x20004750, size: 4, signed: false, idx: -1},
		{addr: 0x2000475c, size: 4, signed: false, idx: -1},
		{addr: 0x2000476c, size: 4, signed: false, idx: -1},
		{addr: 0x2000477c, size: 4, signed: false, idx: -1},
		{addr: 0x2000479c, size: 4, signed: false, idx: -1},
		{addr: 0x200047ac, size: 4, signed: false, idx: -1},
		{addr: 0x200047d8, size: 4, signed: false, idx: -1},
		{addr: 0x200047e8, size: 4, signed: false, idx: -1},
		{addr: 0x200047fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004800, size: 4, signed: false, idx: -1},
	},
	out: []mstFld{
		{addr: mstRet, size: 4, signed: false, idx: -1},
		{addr: 0x2001d8c5, size: 4, signed: false, idx: -1},
		{addr: 0x2001d8c9, size: 4, signed: false, idx: -1},
		{addr: 0x2001d8cd, size: 4, signed: false, idx: -1},
		{addr: 0x2001d8d1, size: 4, signed: false, idx: -1},
		{addr: 0x2001d8d5, size: 4, signed: false, idx: -1},
		{addr: 0x2001d8d9, size: 4, signed: false, idx: -1},
		{addr: 0x2001d8dd, size: 4, signed: false, idx: -1},
		{addr: 0x2001d8e1, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9a4, size: 1, signed: false, idx: -1},
		{addr: 0x2001d9a5, size: 1, signed: false, idx: -1},
		{addr: 0x2001d9a6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9aa, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ae, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9b2, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9b6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ba, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9be, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9c2, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9c6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ca, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ce, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9d2, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9d6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9da, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9de, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9e2, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9e6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ea, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9ee, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9f2, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9f6, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9fa, size: 4, signed: false, idx: -1},
		{addr: 0x2001d9fe, size: 4, signed: false, idx: -1},
		{addr: 0x2001da02, size: 4, signed: false, idx: -1},
		{addr: 0x2001da06, size: 4, signed: false, idx: -1},
		{addr: 0x2001da0a, size: 4, signed: false, idx: -1},
		{addr: 0x2001da0e, size: 4, signed: false, idx: -1},
		{addr: 0x2001da12, size: 4, signed: false, idx: -1},
		{addr: 0x2001da16, size: 4, signed: false, idx: -1},
		{addr: 0x2001da1a, size: 4, signed: false, idx: -1},
		{addr: 0x2001da1e, size: 4, signed: false, idx: -1},
		{addr: 0x2001da22, size: 4, signed: false, idx: -1},
		{addr: 0x2001da26, size: 4, signed: false, idx: -1},
		{addr: 0x2001da2a, size: 4, signed: false, idx: -1},
		{addr: 0x2001da2e, size: 4, signed: false, idx: -1},
		{addr: 0x2001da32, size: 4, signed: false, idx: -1},
		{addr: 0x2001da36, size: 4, signed: false, idx: -1},
		{addr: 0x2001da3a, size: 4, signed: false, idx: -1},
		{addr: 0x2001da3e, size: 4, signed: false, idx: -1},
		{addr: 0x2001da42, size: 4, signed: false, idx: -1},
		{addr: 0x2001da46, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6b8, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6ba, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6bc, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6be, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6c0, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6c2, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6c4, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6c6, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6c8, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6ca, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6cc, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6ce, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6d0, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6d2, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6d4, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6d6, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6d8, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6da, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6dc, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6de, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6e0, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6e2, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6e4, size: 2, signed: false, idx: -1},
		{addr: 0x2001d6e6, size: 2, signed: false, idx: -1},
		{addr: 0x2001d0c6, size: 1, signed: false, idx: -1},
		{addr: 0x2001d0cb, size: 1, signed: false, idx: -1},
		{addr: 0x2001d0e6, size: 1, signed: false, idx: -1},
		{addr: 0x20010a89, size: 4, signed: false, idx: -1},
		{addr: 0x2001d19a, size: 1, signed: false, idx: -1},
		{addr: 0x2001d19f, size: 1, signed: false, idx: -1},
		{addr: 0x2001d1ba, size: 1, signed: false, idx: -1},
		{addr: 0x200122eb, size: 4, signed: false, idx: -1},
		{addr: 0x2001d26e, size: 1, signed: false, idx: -1},
		{addr: 0x2001d273, size: 1, signed: false, idx: -1},
		{addr: 0x2001d28e, size: 1, signed: false, idx: -1},
		{addr: 0x20013b4d, size: 4, signed: false, idx: -1},
		{addr: 0x2001d342, size: 1, signed: false, idx: -1},
		{addr: 0x2001d347, size: 1, signed: false, idx: -1},
		{addr: 0x2001d362, size: 1, signed: false, idx: -1},
		{addr: 0x200153af, size: 4, signed: false, idx: -1},
		{addr: 0x2001d416, size: 1, signed: false, idx: -1},
		{addr: 0x2001d41b, size: 1, signed: false, idx: -1},
		{addr: 0x2001d436, size: 1, signed: false, idx: -1},
		{addr: 0x20016c11, size: 4, signed: false, idx: -1},
		{addr: 0x2001d4ea, size: 1, signed: false, idx: -1},
		{addr: 0x2001d4ef, size: 1, signed: false, idx: -1},
		{addr: 0x2001d50a, size: 1, signed: false, idx: -1},
		{addr: 0x20018473, size: 4, signed: false, idx: -1},
		{addr: 0x2001d5be, size: 1, signed: false, idx: -1},
		{addr: 0x2001d5c3, size: 1, signed: false, idx: -1},
		{addr: 0x2001d5de, size: 1, signed: false, idx: -1},
		{addr: 0x20019cd5, size: 4, signed: false, idx: -1},
		{addr: 0x2001d692, size: 1, signed: false, idx: -1},
		{addr: 0x2001d697, size: 1, signed: false, idx: -1},
		{addr: 0x2001d6b2, size: 1, signed: false, idx: -1},
		{addr: 0x2001b537, size: 4, signed: false, idx: -1},
		{addr: 0x200006ba, size: 4, signed: false, idx: -1},
		{addr: 0x200006be, size: 4, signed: false, idx: -1},
		{addr: 0x200006c2, size: 4, signed: false, idx: -1},
		{addr: 0x200006c6, size: 4, signed: false, idx: -1},
		{addr: 0x200006ca, size: 4, signed: false, idx: -1},
		{addr: 0x200006ce, size: 4, signed: false, idx: -1},
		{addr: 0x200006d2, size: 4, signed: false, idx: -1},
		{addr: 0x200006d6, size: 4, signed: false, idx: -1},
		{addr: 0x200006e1, size: 4, signed: false, idx: -1},
		{addr: 0x2000042b, size: 4, signed: false, idx: -1},
		{addr: 0x20000433, size: 4, signed: false, idx: -1},
		{addr: 0x20000642, size: 4, signed: false, idx: -1},
		{addr: 0x200004c5, size: 2, signed: false, idx: -1},
		{addr: 0x200004c7, size: 4, signed: false, idx: -1},
		{addr: 0x200004bf, size: 4, signed: false, idx: -1},
		{addr: 0x200004c3, size: 2, signed: false, idx: -1},
		{addr: 0x200004cb, size: 4, signed: false, idx: -1},
		{addr: 0x200004cf, size: 4, signed: false, idx: -1},
		{addr: 0x200004d3, size: 4, signed: false, idx: -1},
		{addr: 0x20000704, size: 4, signed: false, idx: -1},
		{addr: 0x20000708, size: 4, signed: false, idx: -1},
		{addr: 0x2000070c, size: 4, signed: false, idx: -1},
		{addr: 0x20000710, size: 4, signed: false, idx: -1},
		{addr: 0x20000714, size: 4, signed: false, idx: -1},
		{addr: 0x20000718, size: 4, signed: false, idx: -1},
		{addr: 0x2000071c, size: 4, signed: false, idx: -1},
		{addr: 0x20000720, size: 4, signed: false, idx: -1},
		{addr: 0x20000724, size: 4, signed: false, idx: -1},
		{addr: 0x20000728, size: 4, signed: false, idx: -1},
		{addr: 0x2000072c, size: 4, signed: false, idx: -1},
		{addr: 0x20000730, size: 4, signed: false, idx: -1},
		{addr: 0x20000734, size: 4, signed: false, idx: -1},
		{addr: 0x20000aba, size: 4, signed: false, idx: -1},
		{addr: 0x20000abe, size: 4, signed: false, idx: -1},
		{addr: 0x20000ac2, size: 4, signed: false, idx: -1},
		{addr: 0x20000ac6, size: 4, signed: false, idx: -1},
		{addr: 0x20000aca, size: 4, signed: false, idx: -1},
		{addr: 0x20000ace, size: 4, signed: false, idx: -1},
		{addr: 0x20000ad2, size: 4, signed: false, idx: -1},
		{addr: 0x20000ad6, size: 4, signed: false, idx: -1},
		{addr: 0x20000ae1, size: 4, signed: false, idx: -1},
		{addr: 0x2000082b, size: 4, signed: false, idx: -1},
		{addr: 0x20000833, size: 4, signed: false, idx: -1},
		{addr: 0x20000a42, size: 4, signed: false, idx: -1},
		{addr: 0x200008c5, size: 2, signed: false, idx: -1},
		{addr: 0x200008c7, size: 4, signed: false, idx: -1},
		{addr: 0x200008bf, size: 4, signed: false, idx: -1},
		{addr: 0x200008c3, size: 2, signed: false, idx: -1},
		{addr: 0x200008cb, size: 4, signed: false, idx: -1},
		{addr: 0x200008cf, size: 4, signed: false, idx: -1},
		{addr: 0x200008d3, size: 4, signed: false, idx: -1},
		{addr: 0x20000b04, size: 4, signed: false, idx: -1},
		{addr: 0x20000b08, size: 4, signed: false, idx: -1},
		{addr: 0x20000b0c, size: 4, signed: false, idx: -1},
		{addr: 0x20000b10, size: 4, signed: false, idx: -1},
		{addr: 0x20000b14, size: 4, signed: false, idx: -1},
		{addr: 0x20000b18, size: 4, signed: false, idx: -1},
		{addr: 0x20000b1c, size: 4, signed: false, idx: -1},
		{addr: 0x20000b20, size: 4, signed: false, idx: -1},
		{addr: 0x20000b24, size: 4, signed: false, idx: -1},
		{addr: 0x20000b28, size: 4, signed: false, idx: -1},
		{addr: 0x20000b2c, size: 4, signed: false, idx: -1},
		{addr: 0x20000b30, size: 4, signed: false, idx: -1},
		{addr: 0x20000b34, size: 4, signed: false, idx: -1},
		{addr: 0x2000340d, size: 4, signed: false, idx: -1},
		{addr: 0x2000341a, size: 4, signed: false, idx: -1},
		{addr: 0x2000341e, size: 4, signed: false, idx: -1},
		{addr: 0x20003422, size: 4, signed: false, idx: -1},
		{addr: 0x20003426, size: 4, signed: false, idx: -1},
		{addr: 0x2000342a, size: 4, signed: false, idx: -1},
		{addr: 0x2000342e, size: 4, signed: false, idx: -1},
		{addr: 0x20003432, size: 4, signed: false, idx: -1},
		{addr: 0x20003436, size: 4, signed: false, idx: -1},
		{addr: 0x689fd0, size: 2, signed: false, idx: -1},
		{addr: 0x689fd2, size: 2, signed: false, idx: -1},
		{addr: 0x689fd4, size: 2, signed: false, idx: -1},
		{addr: 0x689fd6, size: 2, signed: false, idx: -1},
		{addr: 0x689fd8, size: 2, signed: false, idx: -1},
		{addr: 0x689fda, size: 2, signed: false, idx: -1},
		{addr: 0x689fdc, size: 2, signed: false, idx: -1},
		{addr: 0x689fde, size: 2, signed: false, idx: -1},
		{addr: 0x689fe0, size: 2, signed: false, idx: -1},
		{addr: 0x689fe2, size: 2, signed: false, idx: -1},
		{addr: 0x689fe4, size: 2, signed: false, idx: -1},
		{addr: 0x689fe6, size: 2, signed: false, idx: -1},
		{addr: 0x689fe8, size: 2, signed: false, idx: -1},
		{addr: 0x689fea, size: 2, signed: false, idx: -1},
		{addr: 0x689fec, size: 2, signed: false, idx: -1},
		{addr: 0x689fee, size: 2, signed: false, idx: -1},
		{addr: 0x689ff0, size: 2, signed: false, idx: -1},
		{addr: 0x689ff2, size: 2, signed: false, idx: -1},
		{addr: 0x689ff4, size: 2, signed: false, idx: -1},
		{addr: 0x689ff6, size: 2, signed: false, idx: -1},
		{addr: 0x689ff8, size: 2, signed: false, idx: -1},
		{addr: 0x689ffa, size: 2, signed: false, idx: -1},
		{addr: 0x689ffc, size: 2, signed: false, idx: -1},
		{addr: 0x689ffe, size: 2, signed: false, idx: -1},
		{addr: 0x20004004, size: 4, signed: false, idx: -1},
		{addr: 0x20004008, size: 4, signed: false, idx: -1},
		{addr: 0x2000400c, size: 4, signed: false, idx: -1},
		{addr: 0x20004010, size: 4, signed: false, idx: -1},
		{addr: 0x20004014, size: 4, signed: false, idx: -1},
		{addr: 0x20004018, size: 4, signed: false, idx: -1},
		{addr: 0x2000401c, size: 4, signed: false, idx: -1},
		{addr: 0x20004020, size: 4, signed: false, idx: -1},
		{addr: 0x20004024, size: 4, signed: false, idx: -1},
		{addr: 0x20004028, size: 4, signed: false, idx: -1},
		{addr: 0x2000402c, size: 4, signed: false, idx: -1},
		{addr: 0x20004030, size: 4, signed: false, idx: -1},
		{addr: 0x20004034, size: 4, signed: false, idx: -1},
		{addr: 0x20004038, size: 4, signed: false, idx: -1},
		{addr: 0x2000403c, size: 4, signed: false, idx: -1},
		{addr: 0x20004040, size: 4, signed: false, idx: -1},
		{addr: 0x20004044, size: 4, signed: false, idx: -1},
		{addr: 0x20004048, size: 4, signed: false, idx: -1},
		{addr: 0x2000404c, size: 4, signed: false, idx: -1},
		{addr: 0x20004050, size: 4, signed: false, idx: -1},
		{addr: 0x20004054, size: 4, signed: false, idx: -1},
		{addr: 0x20004058, size: 4, signed: false, idx: -1},
		{addr: 0x2000405c, size: 4, signed: false, idx: -1},
		{addr: 0x20004060, size: 4, signed: false, idx: -1},
		{addr: 0x20004064, size: 4, signed: false, idx: -1},
		{addr: 0x20004068, size: 4, signed: false, idx: -1},
		{addr: 0x2000406c, size: 4, signed: false, idx: -1},
		{addr: 0x20004070, size: 4, signed: false, idx: -1},
		{addr: 0x20004074, size: 4, signed: false, idx: -1},
		{addr: 0x20004078, size: 4, signed: false, idx: -1},
		{addr: 0x2000407c, size: 4, signed: false, idx: -1},
		{addr: 0x20004080, size: 4, signed: false, idx: -1},
		{addr: 0x20004084, size: 4, signed: false, idx: -1},
		{addr: 0x20004088, size: 4, signed: false, idx: -1},
		{addr: 0x2000408c, size: 4, signed: false, idx: -1},
		{addr: 0x20004090, size: 4, signed: false, idx: -1},
		{addr: 0x20004094, size: 4, signed: false, idx: -1},
		{addr: 0x20004098, size: 4, signed: false, idx: -1},
		{addr: 0x2000409c, size: 4, signed: false, idx: -1},
		{addr: 0x200040a0, size: 4, signed: false, idx: -1},
		{addr: 0x200040a4, size: 4, signed: false, idx: -1},
		{addr: 0x200040a8, size: 4, signed: false, idx: -1},
		{addr: 0x200040ac, size: 4, signed: false, idx: -1},
		{addr: 0x200040b0, size: 4, signed: false, idx: -1},
		{addr: 0x200040b4, size: 4, signed: false, idx: -1},
		{addr: 0x200040b8, size: 4, signed: false, idx: -1},
		{addr: 0x200040bc, size: 4, signed: false, idx: -1},
		{addr: 0x200040c0, size: 4, signed: false, idx: -1},
		{addr: 0x200040c4, size: 4, signed: false, idx: -1},
		{addr: 0x200040c8, size: 4, signed: false, idx: -1},
		{addr: 0x200040cc, size: 4, signed: false, idx: -1},
		{addr: 0x200040d0, size: 4, signed: false, idx: -1},
		{addr: 0x200040d4, size: 4, signed: false, idx: -1},
		{addr: 0x200040d8, size: 4, signed: false, idx: -1},
		{addr: 0x200040dc, size: 4, signed: false, idx: -1},
		{addr: 0x200040e0, size: 4, signed: false, idx: -1},
		{addr: 0x200040e4, size: 4, signed: false, idx: -1},
		{addr: 0x200040e8, size: 4, signed: false, idx: -1},
		{addr: 0x200040ec, size: 4, signed: false, idx: -1},
		{addr: 0x200040f0, size: 4, signed: false, idx: -1},
		{addr: 0x200040f4, size: 4, signed: false, idx: -1},
		{addr: 0x200040f8, size: 4, signed: false, idx: -1},
		{addr: 0x200040fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004100, size: 4, signed: false, idx: -1},
		{addr: 0x20004104, size: 4, signed: false, idx: -1},
		{addr: 0x20004108, size: 4, signed: false, idx: -1},
		{addr: 0x2000410c, size: 4, signed: false, idx: -1},
		{addr: 0x20004110, size: 4, signed: false, idx: -1},
		{addr: 0x20004114, size: 4, signed: false, idx: -1},
		{addr: 0x20004118, size: 4, signed: false, idx: -1},
		{addr: 0x2000411c, size: 4, signed: false, idx: -1},
		{addr: 0x20004120, size: 4, signed: false, idx: -1},
		{addr: 0x20004124, size: 4, signed: false, idx: -1},
		{addr: 0x20004128, size: 4, signed: false, idx: -1},
		{addr: 0x2000412c, size: 4, signed: false, idx: -1},
		{addr: 0x20004130, size: 4, signed: false, idx: -1},
		{addr: 0x20004134, size: 4, signed: false, idx: -1},
		{addr: 0x20004138, size: 4, signed: false, idx: -1},
		{addr: 0x2000413c, size: 4, signed: false, idx: -1},
		{addr: 0x20004140, size: 4, signed: false, idx: -1},
		{addr: 0x20004144, size: 4, signed: false, idx: -1},
		{addr: 0x20004148, size: 4, signed: false, idx: -1},
		{addr: 0x2000414c, size: 4, signed: false, idx: -1},
		{addr: 0x20004150, size: 4, signed: false, idx: -1},
		{addr: 0x20004154, size: 4, signed: false, idx: -1},
		{addr: 0x20004158, size: 4, signed: false, idx: -1},
		{addr: 0x2000415c, size: 4, signed: false, idx: -1},
		{addr: 0x20004160, size: 4, signed: false, idx: -1},
		{addr: 0x20004164, size: 4, signed: false, idx: -1},
		{addr: 0x20004168, size: 4, signed: false, idx: -1},
		{addr: 0x2000416c, size: 4, signed: false, idx: -1},
		{addr: 0x20004170, size: 4, signed: false, idx: -1},
		{addr: 0x20004174, size: 4, signed: false, idx: -1},
		{addr: 0x20004178, size: 4, signed: false, idx: -1},
		{addr: 0x2000417c, size: 4, signed: false, idx: -1},
		{addr: 0x20004180, size: 4, signed: false, idx: -1},
		{addr: 0x20004184, size: 4, signed: false, idx: -1},
		{addr: 0x20004188, size: 4, signed: false, idx: -1},
		{addr: 0x2000418c, size: 4, signed: false, idx: -1},
		{addr: 0x20004190, size: 4, signed: false, idx: -1},
		{addr: 0x20004194, size: 4, signed: false, idx: -1},
		{addr: 0x20004198, size: 4, signed: false, idx: -1},
		{addr: 0x2000419c, size: 4, signed: false, idx: -1},
		{addr: 0x200041a0, size: 4, signed: false, idx: -1},
		{addr: 0x200041a4, size: 4, signed: false, idx: -1},
		{addr: 0x200041a8, size: 4, signed: false, idx: -1},
		{addr: 0x200041ac, size: 4, signed: false, idx: -1},
		{addr: 0x200041b0, size: 4, signed: false, idx: -1},
		{addr: 0x200041b4, size: 4, signed: false, idx: -1},
		{addr: 0x200041b8, size: 4, signed: false, idx: -1},
		{addr: 0x200041bc, size: 4, signed: false, idx: -1},
		{addr: 0x200041c0, size: 4, signed: false, idx: -1},
		{addr: 0x200041c4, size: 4, signed: false, idx: -1},
		{addr: 0x200041c8, size: 4, signed: false, idx: -1},
		{addr: 0x200041cc, size: 4, signed: false, idx: -1},
		{addr: 0x200041d0, size: 4, signed: false, idx: -1},
		{addr: 0x200041d4, size: 4, signed: false, idx: -1},
		{addr: 0x200041d8, size: 4, signed: false, idx: -1},
		{addr: 0x200041dc, size: 4, signed: false, idx: -1},
		{addr: 0x200041e0, size: 4, signed: false, idx: -1},
		{addr: 0x200041e4, size: 4, signed: false, idx: -1},
		{addr: 0x200041e8, size: 4, signed: false, idx: -1},
		{addr: 0x200041ec, size: 4, signed: false, idx: -1},
		{addr: 0x200041f0, size: 4, signed: false, idx: -1},
		{addr: 0x200041f4, size: 4, signed: false, idx: -1},
		{addr: 0x200041f8, size: 4, signed: false, idx: -1},
		{addr: 0x200041fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004200, size: 4, signed: false, idx: -1},
		{addr: 0x20004204, size: 4, signed: false, idx: -1},
		{addr: 0x20004208, size: 4, signed: false, idx: -1},
		{addr: 0x2000420c, size: 4, signed: false, idx: -1},
		{addr: 0x20004210, size: 4, signed: false, idx: -1},
		{addr: 0x20004214, size: 4, signed: false, idx: -1},
		{addr: 0x20004218, size: 4, signed: false, idx: -1},
		{addr: 0x2000421c, size: 4, signed: false, idx: -1},
		{addr: 0x20004220, size: 4, signed: false, idx: -1},
		{addr: 0x20004224, size: 4, signed: false, idx: -1},
		{addr: 0x20004228, size: 4, signed: false, idx: -1},
		{addr: 0x2000422c, size: 4, signed: false, idx: -1},
		{addr: 0x20004230, size: 4, signed: false, idx: -1},
		{addr: 0x20004234, size: 4, signed: false, idx: -1},
		{addr: 0x20004238, size: 4, signed: false, idx: -1},
		{addr: 0x2000423c, size: 4, signed: false, idx: -1},
		{addr: 0x20004240, size: 4, signed: false, idx: -1},
		{addr: 0x20004244, size: 4, signed: false, idx: -1},
		{addr: 0x20004248, size: 4, signed: false, idx: -1},
		{addr: 0x2000424c, size: 4, signed: false, idx: -1},
		{addr: 0x20004250, size: 4, signed: false, idx: -1},
		{addr: 0x20004254, size: 4, signed: false, idx: -1},
		{addr: 0x20004258, size: 4, signed: false, idx: -1},
		{addr: 0x2000425c, size: 4, signed: false, idx: -1},
		{addr: 0x20004260, size: 4, signed: false, idx: -1},
		{addr: 0x20004264, size: 4, signed: false, idx: -1},
		{addr: 0x20004268, size: 4, signed: false, idx: -1},
		{addr: 0x2000426c, size: 1, signed: false, idx: -1},
		{addr: 0x2000426d, size: 1, signed: false, idx: -1},
		{addr: 0x2000426e, size: 1, signed: false, idx: -1},
		{addr: 0x2000426f, size: 1, signed: false, idx: -1},
		{addr: 0x20004270, size: 1, signed: false, idx: -1},
		{addr: 0x20004271, size: 1, signed: false, idx: -1},
		{addr: 0x20004272, size: 1, signed: false, idx: -1},
		{addr: 0x20004273, size: 1, signed: false, idx: -1},
		{addr: 0x20004274, size: 1, signed: false, idx: -1},
		{addr: 0x20004275, size: 1, signed: false, idx: -1},
		{addr: 0x20004276, size: 1, signed: false, idx: -1},
		{addr: 0x20004277, size: 1, signed: false, idx: -1},
		{addr: 0x20004278, size: 1, signed: false, idx: -1},
		{addr: 0x20004279, size: 1, signed: false, idx: -1},
		{addr: 0x2000427a, size: 1, signed: false, idx: -1},
		{addr: 0x2000427b, size: 1, signed: false, idx: -1},
		{addr: 0x2000427c, size: 1, signed: false, idx: -1},
		{addr: 0x2000427d, size: 1, signed: false, idx: -1},
		{addr: 0x2000427e, size: 1, signed: false, idx: -1},
		{addr: 0x2000427f, size: 1, signed: false, idx: -1},
		{addr: 0x20004280, size: 1, signed: false, idx: -1},
		{addr: 0x20004281, size: 1, signed: false, idx: -1},
		{addr: 0x20004282, size: 1, signed: false, idx: -1},
		{addr: 0x20004283, size: 1, signed: false, idx: -1},
		{addr: 0x20004284, size: 1, signed: false, idx: -1},
		{addr: 0x20004285, size: 1, signed: false, idx: -1},
		{addr: 0x20004286, size: 1, signed: false, idx: -1},
		{addr: 0x20004287, size: 1, signed: false, idx: -1},
		{addr: 0x20004288, size: 1, signed: false, idx: -1},
		{addr: 0x20004289, size: 1, signed: false, idx: -1},
		{addr: 0x2000428a, size: 1, signed: false, idx: -1},
		{addr: 0x2000428b, size: 1, signed: false, idx: -1},
		{addr: 0x2000428c, size: 1, signed: false, idx: -1},
		{addr: 0x2000428d, size: 1, signed: false, idx: -1},
		{addr: 0x2000428e, size: 1, signed: false, idx: -1},
		{addr: 0x2000428f, size: 1, signed: false, idx: -1},
		{addr: 0x20004290, size: 1, signed: false, idx: -1},
		{addr: 0x20004291, size: 1, signed: false, idx: -1},
		{addr: 0x20004292, size: 1, signed: false, idx: -1},
		{addr: 0x20004293, size: 1, signed: false, idx: -1},
		{addr: 0x20004294, size: 1, signed: false, idx: -1},
		{addr: 0x20004295, size: 1, signed: false, idx: -1},
		{addr: 0x20004296, size: 1, signed: false, idx: -1},
		{addr: 0x20004297, size: 1, signed: false, idx: -1},
		{addr: 0x20004298, size: 1, signed: false, idx: -1},
		{addr: 0x20004299, size: 1, signed: false, idx: -1},
		{addr: 0x2000429a, size: 1, signed: false, idx: -1},
		{addr: 0x2000429b, size: 1, signed: false, idx: -1},
		{addr: 0x2000429c, size: 1, signed: false, idx: -1},
		{addr: 0x2000429d, size: 1, signed: false, idx: -1},
		{addr: 0x2000429e, size: 1, signed: false, idx: -1},
		{addr: 0x2000429f, size: 1, signed: false, idx: -1},
		{addr: 0x200042a0, size: 1, signed: false, idx: -1},
		{addr: 0x200042a1, size: 1, signed: false, idx: -1},
		{addr: 0x200042a2, size: 1, signed: false, idx: -1},
		{addr: 0x200042a3, size: 1, signed: false, idx: -1},
		{addr: 0x200042a4, size: 1, signed: false, idx: -1},
		{addr: 0x200042a5, size: 1, signed: false, idx: -1},
		{addr: 0x200042a6, size: 1, signed: false, idx: -1},
		{addr: 0x200042a7, size: 1, signed: false, idx: -1},
		{addr: 0x200042a8, size: 1, signed: false, idx: -1},
		{addr: 0x200042a9, size: 1, signed: false, idx: -1},
		{addr: 0x200042aa, size: 1, signed: false, idx: -1},
		{addr: 0x200042ab, size: 1, signed: false, idx: -1},
		{addr: 0x200042ac, size: 1, signed: false, idx: -1},
		{addr: 0x200042ad, size: 1, signed: false, idx: -1},
		{addr: 0x200042ae, size: 1, signed: false, idx: -1},
		{addr: 0x200042af, size: 1, signed: false, idx: -1},
		{addr: 0x200042b0, size: 1, signed: false, idx: -1},
		{addr: 0x200042b1, size: 1, signed: false, idx: -1},
		{addr: 0x200042b2, size: 1, signed: false, idx: -1},
		{addr: 0x200042b3, size: 1, signed: false, idx: -1},
		{addr: 0x200042b4, size: 1, signed: false, idx: -1},
		{addr: 0x200042b5, size: 1, signed: false, idx: -1},
		{addr: 0x200042b6, size: 1, signed: false, idx: -1},
		{addr: 0x200042b7, size: 1, signed: false, idx: -1},
		{addr: 0x200042b8, size: 1, signed: false, idx: -1},
		{addr: 0x200042b9, size: 1, signed: false, idx: -1},
		{addr: 0x200042ba, size: 1, signed: false, idx: -1},
		{addr: 0x200042bb, size: 1, signed: false, idx: -1},
		{addr: 0x200042bc, size: 1, signed: false, idx: -1},
		{addr: 0x200042bd, size: 1, signed: false, idx: -1},
		{addr: 0x200042be, size: 1, signed: false, idx: -1},
		{addr: 0x200042bf, size: 1, signed: false, idx: -1},
		{addr: 0x200042c0, size: 1, signed: false, idx: -1},
		{addr: 0x200042c1, size: 1, signed: false, idx: -1},
		{addr: 0x200042c2, size: 1, signed: false, idx: -1},
		{addr: 0x200042c3, size: 1, signed: false, idx: -1},
		{addr: 0x200042c4, size: 1, signed: false, idx: -1},
		{addr: 0x200042c5, size: 1, signed: false, idx: -1},
		{addr: 0x200042c6, size: 1, signed: false, idx: -1},
		{addr: 0x200042c7, size: 1, signed: false, idx: -1},
		{addr: 0x200042c8, size: 1, signed: false, idx: -1},
		{addr: 0x200042c9, size: 1, signed: false, idx: -1},
		{addr: 0x200042ca, size: 1, signed: false, idx: -1},
		{addr: 0x200042cb, size: 1, signed: false, idx: -1},
		{addr: 0x200042cc, size: 1, signed: false, idx: -1},
		{addr: 0x200042cd, size: 1, signed: false, idx: -1},
		{addr: 0x200042ce, size: 1, signed: false, idx: -1},
		{addr: 0x200042cf, size: 1, signed: false, idx: -1},
		{addr: 0x200042d0, size: 1, signed: false, idx: -1},
		{addr: 0x200042d1, size: 1, signed: false, idx: -1},
		{addr: 0x200042d2, size: 1, signed: false, idx: -1},
		{addr: 0x200042d3, size: 1, signed: false, idx: -1},
		{addr: 0x200042d4, size: 1, signed: false, idx: -1},
		{addr: 0x200042d5, size: 1, signed: false, idx: -1},
		{addr: 0x200042d6, size: 1, signed: false, idx: -1},
		{addr: 0x200042d7, size: 1, signed: false, idx: -1},
		{addr: 0x200042d8, size: 1, signed: false, idx: -1},
		{addr: 0x200042d9, size: 1, signed: false, idx: -1},
		{addr: 0x200042da, size: 1, signed: false, idx: -1},
		{addr: 0x200042db, size: 1, signed: false, idx: -1},
		{addr: 0x200042dc, size: 1, signed: false, idx: -1},
		{addr: 0x200042dd, size: 1, signed: false, idx: -1},
		{addr: 0x200042de, size: 1, signed: false, idx: -1},
		{addr: 0x200042df, size: 1, signed: false, idx: -1},
		{addr: 0x200042e0, size: 1, signed: false, idx: -1},
		{addr: 0x200042e1, size: 1, signed: false, idx: -1},
		{addr: 0x200042e2, size: 1, signed: false, idx: -1},
		{addr: 0x200042e3, size: 1, signed: false, idx: -1},
		{addr: 0x200042e4, size: 4, signed: false, idx: -1},
		{addr: 0x200042e8, size: 4, signed: false, idx: -1},
		{addr: 0x200042ec, size: 4, signed: false, idx: -1},
		{addr: 0x200042f0, size: 4, signed: false, idx: -1},
		{addr: 0x200042f4, size: 4, signed: false, idx: -1},
		{addr: 0x200042f8, size: 4, signed: false, idx: -1},
		{addr: 0x200042fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004300, size: 4, signed: false, idx: -1},
		{addr: 0x20004304, size: 4, signed: false, idx: -1},
		{addr: 0x20004308, size: 4, signed: false, idx: -1},
		{addr: 0x2000430c, size: 4, signed: false, idx: -1},
		{addr: 0x20004310, size: 4, signed: false, idx: -1},
		{addr: 0x20004314, size: 4, signed: false, idx: -1},
		{addr: 0x20004318, size: 4, signed: false, idx: -1},
		{addr: 0x2000431c, size: 4, signed: false, idx: -1},
		{addr: 0x20004320, size: 4, signed: false, idx: -1},
		{addr: 0x20004324, size: 4, signed: false, idx: -1},
		{addr: 0x20004328, size: 4, signed: false, idx: -1},
		{addr: 0x2000432c, size: 4, signed: false, idx: -1},
		{addr: 0x20004330, size: 4, signed: false, idx: -1},
		{addr: 0x20004334, size: 4, signed: false, idx: -1},
		{addr: 0x20004338, size: 4, signed: false, idx: -1},
		{addr: 0x2000433c, size: 4, signed: false, idx: -1},
		{addr: 0x20004340, size: 4, signed: false, idx: -1},
		{addr: 0x20004344, size: 4, signed: false, idx: -1},
		{addr: 0x20004348, size: 4, signed: false, idx: -1},
		{addr: 0x2000434c, size: 4, signed: false, idx: -1},
		{addr: 0x20004350, size: 4, signed: false, idx: -1},
		{addr: 0x20004354, size: 4, signed: false, idx: -1},
		{addr: 0x20004358, size: 4, signed: false, idx: -1},
		{addr: 0x2000435c, size: 4, signed: false, idx: -1},
		{addr: 0x20004360, size: 4, signed: false, idx: -1},
		{addr: 0x20004364, size: 4, signed: false, idx: -1},
		{addr: 0x20004368, size: 4, signed: false, idx: -1},
		{addr: 0x2000436c, size: 4, signed: false, idx: -1},
		{addr: 0x20004370, size: 4, signed: false, idx: -1},
		{addr: 0x20004374, size: 4, signed: false, idx: -1},
		{addr: 0x20004378, size: 4, signed: false, idx: -1},
		{addr: 0x2000437c, size: 4, signed: false, idx: -1},
		{addr: 0x20004380, size: 4, signed: false, idx: -1},
		{addr: 0x20004384, size: 4, signed: false, idx: -1},
		{addr: 0x20004388, size: 4, signed: false, idx: -1},
		{addr: 0x2000438c, size: 4, signed: false, idx: -1},
		{addr: 0x20004390, size: 4, signed: false, idx: -1},
		{addr: 0x20004394, size: 4, signed: false, idx: -1},
		{addr: 0x20004398, size: 4, signed: false, idx: -1},
		{addr: 0x2000439c, size: 4, signed: false, idx: -1},
		{addr: 0x200043a0, size: 4, signed: false, idx: -1},
		{addr: 0x200043a4, size: 4, signed: false, idx: -1},
		{addr: 0x200043a8, size: 4, signed: false, idx: -1},
		{addr: 0x200043ac, size: 4, signed: false, idx: -1},
		{addr: 0x200043b0, size: 4, signed: false, idx: -1},
		{addr: 0x200043b4, size: 4, signed: false, idx: -1},
		{addr: 0x200043b8, size: 4, signed: false, idx: -1},
		{addr: 0x200043bc, size: 4, signed: false, idx: -1},
		{addr: 0x200043c0, size: 4, signed: false, idx: -1},
		{addr: 0x200043c4, size: 4, signed: false, idx: -1},
		{addr: 0x200043c8, size: 4, signed: false, idx: -1},
		{addr: 0x200043cc, size: 4, signed: false, idx: -1},
		{addr: 0x200043d0, size: 4, signed: false, idx: -1},
		{addr: 0x200043d4, size: 4, signed: false, idx: -1},
		{addr: 0x200043d8, size: 4, signed: false, idx: -1},
		{addr: 0x200043dc, size: 4, signed: false, idx: -1},
		{addr: 0x200043e0, size: 4, signed: false, idx: -1},
		{addr: 0x200043e4, size: 4, signed: false, idx: -1},
		{addr: 0x200043e8, size: 4, signed: false, idx: -1},
		{addr: 0x200043ec, size: 4, signed: false, idx: -1},
		{addr: 0x200043f0, size: 4, signed: false, idx: -1},
		{addr: 0x200043f4, size: 4, signed: false, idx: -1},
		{addr: 0x200043f8, size: 4, signed: false, idx: -1},
		{addr: 0x200043fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004400, size: 4, signed: false, idx: -1},
		{addr: 0x20004404, size: 4, signed: false, idx: -1},
		{addr: 0x20004408, size: 4, signed: false, idx: -1},
		{addr: 0x2000440c, size: 4, signed: false, idx: -1},
		{addr: 0x20004410, size: 4, signed: false, idx: -1},
		{addr: 0x20004414, size: 4, signed: false, idx: -1},
		{addr: 0x20004418, size: 4, signed: false, idx: -1},
		{addr: 0x2000441c, size: 4, signed: false, idx: -1},
		{addr: 0x20004420, size: 4, signed: false, idx: -1},
		{addr: 0x20004424, size: 4, signed: false, idx: -1},
		{addr: 0x20004428, size: 4, signed: false, idx: -1},
		{addr: 0x2000442c, size: 4, signed: false, idx: -1},
		{addr: 0x20004430, size: 4, signed: false, idx: -1},
		{addr: 0x20004434, size: 4, signed: false, idx: -1},
		{addr: 0x20004438, size: 4, signed: false, idx: -1},
		{addr: 0x2000443c, size: 4, signed: false, idx: -1},
		{addr: 0x20004440, size: 4, signed: false, idx: -1},
		{addr: 0x20004444, size: 4, signed: false, idx: -1},
		{addr: 0x20004448, size: 4, signed: false, idx: -1},
		{addr: 0x2000444c, size: 4, signed: false, idx: -1},
		{addr: 0x20004450, size: 4, signed: false, idx: -1},
		{addr: 0x20004454, size: 4, signed: false, idx: -1},
		{addr: 0x20004458, size: 4, signed: false, idx: -1},
		{addr: 0x2000445c, size: 4, signed: false, idx: -1},
		{addr: 0x20004460, size: 4, signed: false, idx: -1},
		{addr: 0x20004464, size: 4, signed: false, idx: -1},
		{addr: 0x20004468, size: 4, signed: false, idx: -1},
		{addr: 0x2000446c, size: 4, signed: false, idx: -1},
		{addr: 0x20004470, size: 4, signed: false, idx: -1},
		{addr: 0x20004474, size: 4, signed: false, idx: -1},
		{addr: 0x20004478, size: 4, signed: false, idx: -1},
		{addr: 0x2000447c, size: 4, signed: false, idx: -1},
		{addr: 0x20004480, size: 4, signed: false, idx: -1},
		{addr: 0x20004484, size: 4, signed: false, idx: -1},
		{addr: 0x20004488, size: 4, signed: false, idx: -1},
		{addr: 0x2000448c, size: 4, signed: false, idx: -1},
		{addr: 0x20004490, size: 4, signed: false, idx: -1},
		{addr: 0x20004494, size: 4, signed: false, idx: -1},
		{addr: 0x20004498, size: 4, signed: false, idx: -1},
		{addr: 0x2000449c, size: 4, signed: false, idx: -1},
		{addr: 0x200044a0, size: 4, signed: false, idx: -1},
		{addr: 0x200044a4, size: 4, signed: false, idx: -1},
		{addr: 0x200044a8, size: 4, signed: false, idx: -1},
		{addr: 0x200044ac, size: 4, signed: false, idx: -1},
		{addr: 0x200044b0, size: 4, signed: false, idx: -1},
		{addr: 0x200044b4, size: 4, signed: false, idx: -1},
		{addr: 0x200044b8, size: 4, signed: false, idx: -1},
		{addr: 0x200044bc, size: 4, signed: false, idx: -1},
		{addr: 0x200044c0, size: 4, signed: false, idx: -1},
		{addr: 0x200044c4, size: 4, signed: false, idx: -1},
		{addr: 0x200044c8, size: 4, signed: false, idx: -1},
		{addr: 0x200044cc, size: 4, signed: false, idx: -1},
		{addr: 0x200044d0, size: 4, signed: false, idx: -1},
		{addr: 0x200044d4, size: 4, signed: false, idx: -1},
		{addr: 0x200044d8, size: 4, signed: false, idx: -1},
		{addr: 0x200044dc, size: 4, signed: false, idx: -1},
		{addr: 0x200044e0, size: 4, signed: false, idx: -1},
		{addr: 0x200044e4, size: 4, signed: false, idx: -1},
		{addr: 0x200044e8, size: 4, signed: false, idx: -1},
		{addr: 0x200044ec, size: 4, signed: false, idx: -1},
		{addr: 0x200044f0, size: 4, signed: false, idx: -1},
		{addr: 0x200044f4, size: 4, signed: false, idx: -1},
		{addr: 0x200044f8, size: 4, signed: false, idx: -1},
		{addr: 0x200044fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004500, size: 4, signed: false, idx: -1},
		{addr: 0x20004504, size: 4, signed: false, idx: -1},
		{addr: 0x20004508, size: 4, signed: false, idx: -1},
		{addr: 0x2000450c, size: 4, signed: false, idx: -1},
		{addr: 0x20004510, size: 4, signed: false, idx: -1},
		{addr: 0x20004514, size: 4, signed: false, idx: -1},
		{addr: 0x20004518, size: 4, signed: false, idx: -1},
		{addr: 0x2000451c, size: 4, signed: false, idx: -1},
		{addr: 0x20004520, size: 4, signed: false, idx: -1},
		{addr: 0x20004524, size: 4, signed: false, idx: -1},
		{addr: 0x20004528, size: 4, signed: false, idx: -1},
		{addr: 0x2000452c, size: 4, signed: false, idx: -1},
		{addr: 0x20004530, size: 4, signed: false, idx: -1},
		{addr: 0x20004534, size: 4, signed: false, idx: -1},
		{addr: 0x20004538, size: 4, signed: false, idx: -1},
		{addr: 0x2000453c, size: 4, signed: false, idx: -1},
		{addr: 0x20004540, size: 4, signed: false, idx: -1},
		{addr: 0x20004544, size: 4, signed: false, idx: -1},
		{addr: 0x20004548, size: 4, signed: false, idx: -1},
		{addr: 0x2000454c, size: 4, signed: false, idx: -1},
		{addr: 0x20004550, size: 4, signed: false, idx: -1},
		{addr: 0x20004554, size: 4, signed: false, idx: -1},
		{addr: 0x20004558, size: 4, signed: false, idx: -1},
		{addr: 0x2000455c, size: 4, signed: false, idx: -1},
		{addr: 0x20004560, size: 4, signed: false, idx: -1},
		{addr: 0x20004564, size: 4, signed: false, idx: -1},
		{addr: 0x20004568, size: 4, signed: false, idx: -1},
		{addr: 0x2000456c, size: 4, signed: false, idx: -1},
		{addr: 0x20004570, size: 4, signed: false, idx: -1},
		{addr: 0x20004574, size: 4, signed: false, idx: -1},
		{addr: 0x20004578, size: 4, signed: false, idx: -1},
		{addr: 0x2000457c, size: 4, signed: false, idx: -1},
		{addr: 0x20004580, size: 4, signed: false, idx: -1},
		{addr: 0x20004584, size: 4, signed: false, idx: -1},
		{addr: 0x20004588, size: 4, signed: false, idx: -1},
		{addr: 0x2000458c, size: 4, signed: false, idx: -1},
		{addr: 0x20004590, size: 4, signed: false, idx: -1},
		{addr: 0x20004594, size: 4, signed: false, idx: -1},
		{addr: 0x20004598, size: 4, signed: false, idx: -1},
		{addr: 0x2000459c, size: 4, signed: false, idx: -1},
		{addr: 0x200045a0, size: 4, signed: false, idx: -1},
		{addr: 0x200045a4, size: 4, signed: false, idx: -1},
		{addr: 0x200045a8, size: 4, signed: false, idx: -1},
		{addr: 0x200045ac, size: 4, signed: false, idx: -1},
		{addr: 0x200045b0, size: 4, signed: false, idx: -1},
		{addr: 0x200045b4, size: 4, signed: false, idx: -1},
		{addr: 0x200045b8, size: 4, signed: false, idx: -1},
		{addr: 0x200045bc, size: 4, signed: false, idx: -1},
		{addr: 0x200045c0, size: 4, signed: false, idx: -1},
		{addr: 0x200045c4, size: 4, signed: false, idx: -1},
		{addr: 0x200045c8, size: 4, signed: false, idx: -1},
		{addr: 0x200045cc, size: 4, signed: false, idx: -1},
		{addr: 0x200045d0, size: 4, signed: false, idx: -1},
		{addr: 0x200045d4, size: 4, signed: false, idx: -1},
		{addr: 0x200045d8, size: 4, signed: false, idx: -1},
		{addr: 0x200045dc, size: 4, signed: false, idx: -1},
		{addr: 0x200045e0, size: 4, signed: false, idx: -1},
		{addr: 0x200045e4, size: 4, signed: false, idx: -1},
		{addr: 0x200045e8, size: 4, signed: false, idx: -1},
		{addr: 0x200045ec, size: 4, signed: false, idx: -1},
		{addr: 0x200045f0, size: 4, signed: false, idx: -1},
		{addr: 0x200045f4, size: 4, signed: false, idx: -1},
		{addr: 0x200045f8, size: 4, signed: false, idx: -1},
		{addr: 0x200045fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004600, size: 4, signed: false, idx: -1},
		{addr: 0x20004604, size: 4, signed: false, idx: -1},
		{addr: 0x20004608, size: 4, signed: false, idx: -1},
		{addr: 0x2000460c, size: 4, signed: false, idx: -1},
		{addr: 0x20004610, size: 4, signed: false, idx: -1},
		{addr: 0x20004614, size: 4, signed: false, idx: -1},
		{addr: 0x20004618, size: 4, signed: false, idx: -1},
		{addr: 0x2000461c, size: 4, signed: false, idx: -1},
		{addr: 0x20004620, size: 4, signed: false, idx: -1},
		{addr: 0x20004624, size: 4, signed: false, idx: -1},
		{addr: 0x20004628, size: 4, signed: false, idx: -1},
		{addr: 0x20004630, size: 4, signed: false, idx: -1},
		{addr: 0x20004634, size: 4, signed: false, idx: -1},
		{addr: 0x20004638, size: 4, signed: false, idx: -1},
		{addr: 0x2000463c, size: 4, signed: false, idx: -1},
		{addr: 0x20004640, size: 4, signed: false, idx: -1},
		{addr: 0x2000464c, size: 4, signed: false, idx: -1},
		{addr: 0x20004650, size: 4, signed: false, idx: -1},
		{addr: 0x20004654, size: 4, signed: false, idx: -1},
		{addr: 0x20004664, size: 4, signed: false, idx: -1},
		{addr: 0x20004668, size: 4, signed: false, idx: -1},
		{addr: 0x2000467c, size: 4, signed: false, idx: -1},
		{addr: 0x20004688, size: 4, signed: false, idx: -1},
		{addr: 0x2000468c, size: 4, signed: false, idx: -1},
		{addr: 0x20004690, size: 4, signed: false, idx: -1},
		{addr: 0x20004698, size: 4, signed: false, idx: -1},
		{addr: 0x2000469c, size: 4, signed: false, idx: -1},
		{addr: 0x200046a0, size: 4, signed: false, idx: -1},
		{addr: 0x200046ac, size: 4, signed: false, idx: -1},
		{addr: 0x200046b0, size: 4, signed: false, idx: -1},
		{addr: 0x200046b8, size: 4, signed: false, idx: -1},
		{addr: 0x200046bc, size: 4, signed: false, idx: -1},
		{addr: 0x200046c4, size: 4, signed: false, idx: -1},
		{addr: 0x200046c8, size: 4, signed: false, idx: -1},
		{addr: 0x200046cc, size: 4, signed: false, idx: -1},
		{addr: 0x200046d0, size: 4, signed: false, idx: -1},
		{addr: 0x200046d4, size: 4, signed: false, idx: -1},
		{addr: 0x200046d8, size: 4, signed: false, idx: -1},
		{addr: 0x200046dc, size: 1, signed: false, idx: -1},
		{addr: 0x200046dd, size: 1, signed: false, idx: -1},
		{addr: 0x200046de, size: 1, signed: false, idx: -1},
		{addr: 0x200046e0, size: 1, signed: false, idx: -1},
		{addr: 0x200046e1, size: 1, signed: false, idx: -1},
		{addr: 0x200046e2, size: 1, signed: false, idx: -1},
		{addr: 0x200046e4, size: 4, signed: false, idx: -1},
		{addr: 0x200046e8, size: 4, signed: false, idx: -1},
		{addr: 0x200046ec, size: 4, signed: false, idx: -1},
		{addr: 0x200046f0, size: 4, signed: false, idx: -1},
		{addr: 0x200046f8, size: 4, signed: false, idx: -1},
		{addr: 0x200046fc, size: 4, signed: false, idx: -1},
		{addr: 0x20004700, size: 1, signed: false, idx: -1},
		{addr: 0x20004704, size: 4, signed: false, idx: -1},
		{addr: 0x20004708, size: 4, signed: false, idx: -1},
		{addr: 0x20004710, size: 4, signed: false, idx: -1},
		{addr: 0x20004714, size: 4, signed: false, idx: -1},
		{addr: 0x20004718, size: 4, signed: false, idx: -1},
		{addr: 0x20004720, size: 4, signed: false, idx: -1},
		{addr: 0x20004724, size: 4, signed: false, idx: -1},
		{addr: 0x20004728, size: 4, signed: false, idx: -1},
		{addr: 0x20004740, size: 4, signed: false, idx: -1},
		{addr: 0x20004744, size: 4, signed: false, idx: -1},
		{addr: 0x20004748, size: 4, signed: false, idx: -1},
		{addr: 0x2000474c, size: 4, signed: false, idx: -1},
		{addr: 0x20004754, size: 4, signed: false, idx: -1},
		{addr: 0x20004758, size: 4, signed: false, idx: -1},
		{addr: 0x20004760, size: 4, signed: false, idx: -1},
		{addr: 0x20004764, size: 4, signed: false, idx: -1},
		{addr: 0x20004768, size: 4, signed: false, idx: -1},
		{addr: 0x20004770, size: 4, signed: false, idx: -1},
		{addr: 0x20004774, size: 4, signed: false, idx: -1},
		{addr: 0x20004778, size: 4, signed: false, idx: -1},
		{addr: 0x20004780, size: 4, signed: false, idx: -1},
		{addr: 0x20004784, size: 4, signed: false, idx: -1},
		{addr: 0x20004788, size: 4, signed: false, idx: -1},
		{addr: 0x2000478c, size: 4, signed: false, idx: -1},
		{addr: 0x20004790, size: 4, signed: false, idx: -1},
		{addr: 0x20004794, size: 4, signed: false, idx: -1},
		{addr: 0x20004798, size: 4, signed: false, idx: -1},
		{addr: 0x200047a0, size: 4, signed: false, idx: -1},
		{addr: 0x200047a4, size: 4, signed: false, idx: -1},
		{addr: 0x200047a8, size: 4, signed: false, idx: -1},
		{addr: 0x200047b0, size: 4, signed: false, idx: -1},
		{addr: 0x200047b4, size: 4, signed: false, idx: -1},
		{addr: 0x200047b8, size: 4, signed: false, idx: -1},
		{addr: 0x200047bc, size: 4, signed: false, idx: -1},
		{addr: 0x200047c0, size: 4, signed: false, idx: -1},
		{addr: 0x200047c4, size: 4, signed: false, idx: -1},
		{addr: 0x200047c8, size: 4, signed: false, idx: -1},
		{addr: 0x200047cc, size: 4, signed: false, idx: -1},
		{addr: 0x200047d0, size: 1, signed: false, idx: -1},
		{addr: 0x200047d1, size: 1, signed: false, idx: -1},
		{addr: 0x200047d4, size: 4, signed: false, idx: -1},
		{addr: 0x200047dc, size: 4, signed: false, idx: -1},
		{addr: 0x200047e0, size: 4, signed: false, idx: -1},
		{addr: 0x200047e4, size: 4, signed: false, idx: -1},
		{addr: 0x200047ec, size: 4, signed: false, idx: -1},
		{addr: 0x200047f0, size: 4, signed: false, idx: -1},
		{addr: 0x200047f4, size: 1, signed: false, idx: -1},
		{addr: 0x200047f5, size: 1, signed: false, idx: -1},
		{addr: 0x200047f8, size: 4, signed: false, idx: -1},
		{addr: 0x20004804, size: 4, signed: false, idx: -1},
		{addr: 0x20004808, size: 4, signed: false, idx: -1},
		{addr: 0x2000480c, size: 4, signed: false, idx: -1},
		{addr: 0x20004810, size: 4, signed: false, idx: -1},
		{addr: 0x20004814, size: 4, signed: false, idx: -1},
	},
})

var m17In = []string{"g68510c", "g6850d5", "g68d810", "g689b32", "g6875ef", "G.ranged", "G.9a4", "G.9a5", "res.total", "res.hits", "res.tally0", "res.tally1", "res.tally2", "res.tally3", "res.tally4", "res.tally5", "res.tally6", "res.tally7", "res.tally8", "res.tally9", "res.tally10", "res.tally11", "res.tally12", "res.tally13", "res.tally14", "res.tally15", "res.tally16", "res.tally17", "res.tally18", "res.tally19", "res.count", "res.v1a", "res.v1b", "res.v1c", "res.v1d", "res.v1e", "res.shown", "res.missed", "res.tgtType", "res.tgtIdx", "res.tgtId", "rent0.type", "rent0.idx", "rent1.type", "rent1.idx", "c0.f33f", "c0.f281", "c0.e99", "c0.f741", "c0.hp", "c0.maxhp", "c0.item", "c0.st0", "c0.st1", "c0.st2", "c0.st3", "c0.per", "c0.perOn", "c0.a49", "c0.f1830", "c0.race", "c0.slot0", "c0.slot4", "c0.slot5", "c0.slot10", "c0.slot11", "c1.f33f", "c1.f281", "c1.e99", "c1.f741", "c1.hp", "c1.maxhp", "c1.item", "c1.st0", "c1.st1", "c1.st2", "c1.st3", "c1.per", "c1.perOn", "c1.a49", "c1.f1830", "c1.race", "c1.slot0", "c1.slot4", "c1.slot5", "c1.slot10", "c1.slot11", "c2.f33f", "c2.f281", "c2.e99", "c2.f741", "c2.hp", "c2.maxhp", "c2.item", "c2.st0", "c2.st1", "c2.st2", "c2.st3", "c2.per", "c2.perOn", "c2.a49", "c2.f1830", "c2.race", "c2.slot0", "c2.slot4", "c2.slot5", "c2.slot10", "c2.slot11", "c3.f33f", "c3.f281", "c3.e99", "c3.f741", "c3.hp", "c3.maxhp", "c3.item", "c3.st0", "c3.st1", "c3.st2", "c3.st3", "c3.per", "c3.perOn", "c3.a49", "c3.f1830", "c3.race", "c3.slot0", "c3.slot4", "c3.slot5", "c3.slot10", "c3.slot11", "c4.f33f", "c4.f281", "c4.e99", "c4.f741", "c4.hp", "c4.maxhp", "c4.item", "c4.st0", "c4.st1", "c4.st2", "c4.st3", "c4.per", "c4.perOn", "c4.a49", "c4.f1830", "c4.race", "c4.slot0", "c4.slot4", "c4.slot5", "c4.slot10", "c4.slot11", "c5.f33f", "c5.f281", "c5.e99", "c5.f741", "c5.hp", "c5.maxhp", "c5.item", "c5.st0", "c5.st1", "c5.st2", "c5.st3", "c5.per", "c5.perOn", "c5.a49", "c5.f1830", "c5.race", "c5.slot0", "c5.slot4", "c5.slot5", "c5.slot10", "c5.slot11", "c6.f33f", "c6.f281", "c6.e99", "c6.f741", "c6.hp", "c6.maxhp", "c6.item", "c6.st0", "c6.st1", "c6.st2", "c6.st3", "c6.per", "c6.perOn", "c6.a49", "c6.f1830", "c6.race", "c6.slot0", "c6.slot4", "c6.slot5", "c6.slot10", "c6.slot11", "c7.f33f", "c7.f281", "c7.e99", "c7.f741", "c7.hp", "c7.maxhp", "c7.item", "c7.st0", "c7.st1", "c7.st2", "c7.st3", "c7.per", "c7.perOn", "c7.a49", "c7.f1830", "c7.race", "c7.slot0", "c7.slot4", "c7.slot5", "c7.slot10", "c7.slot11", "item500.c1", "item0.c1", "item1.c1", "item2.c1", "item3.c1", "side0.f1", "side1.f1", "side2.f1", "side3.f1", "side4.f1", "side5.f1", "side6.f1", "side7.f1", "mtab0", "mtab1", "mtab2", "mtab3", "mtab4", "mtab5", "mtab6", "mtab7", "mtab8", "C.idx", "C.swings", "M.mode", "M.kind", "M.tgtType", "M.tgtIdx", "M.tgtId", "M.1dc", "M.1dd", "M.fatigue", "M.st0", "M.st1", "M.st2", "M.st3", "M.side", "M.107", "T0.species", "T0.body", "T0.loc0", "T0.loc1", "T0.loc2", "T0.loc3", "T0.loc4", "T0.loc5", "T0.loc6", "T0.d0", "T0.snd", "T0.cc", "T1.species", "T1.body", "T1.loc0", "T1.loc1", "T1.loc2", "T1.loc3", "T1.loc4", "T1.loc5", "T1.loc6", "T1.d0", "T1.snd", "T1.cc", "T0.aw0", "T0.aw1", "T0.aw2", "T0.aw3", "T0.aw4", "T0.247", "T0.24f", "A0.on", "A0.type", "A0.r", "A0.pen", "A0.d5", "A0.d9", "A0.dd", "A0.d11", "A0.1b", "A0.snd", "A1.on", "A1.type", "A1.r", "A1.pen", "A1.d5", "A1.d9", "A1.dd", "A1.d11", "A1.1b", "A1.snd", "A2.on", "A2.type", "A2.r", "A2.pen", "A2.d5", "A2.d9", "A2.dd", "A2.d11", "A2.1b", "A2.snd", "M.group", "M.inCombat", "M.HP", "M.maxSP", "M.SP", "M.exhaustion", "M.shield", "M.shieldDuration", "M.messages", "M.9f", "M.per", "M.perOn", "M.24e", "M.a7", "M.a4", "TM.group", "TM.inCombat", "TM.HP", "TM.maxSP", "TM.SP", "TM.exhaustion", "TM.shield", "TM.shieldDuration", "TM.messages", "TM.9f", "TM.per", "TM.perOn", "TM.24e", "TM.st0", "TM.st1", "TM.st2", "TM.st3", "TM.a7", "TM.a4", "group.named", "group.count", "group.tpl", "fe0.type", "fe0.idx", "fe0.id", "fe1.type", "fe1.idx", "fe1.id", "fe2.type", "fe2.idx", "fe2.id", "fe3.type", "fe3.idx", "fe3.id", "fe4.type", "fe4.idx", "fe4.id", "fe5.type", "fe5.idx", "fe5.id", "fe6.type", "fe6.idx", "fe6.id", "fe7.type", "fe7.idx", "fe7.id", "r4c4ca0", "r51b5c0", "r542720_0", "r542720_1", "r544010", "w544010.count", "r545e50", "w545e50.type", "w545e50.idx", "w545e50.id", "r536190_0", "r536190_1", "r542fc0", "r542960", "w542960.hits", "r545a00", "r52a890", "r543270", "r553770", "r553540", "r5323f0", "w5323f0.kind", "w5323f0.type", "w5323f0.idx", "w5323f0.id", "r53ffe0", "r541630", "r523a80", "r52beb0", "r52af50", "r523c00", "r4e4280", "r4e46f0", "r53c3f0_0", "r53c3f0_1"}

var m17Out = []string{"ret", "G.tgt0", "G.tgt1", "G.tgt2", "G.tgt3", "G.tgt4", "G.tgt5", "G.tgt6", "G.tgt7", "G.9a4", "G.9a5", "res00", "res01", "res02", "res03", "res04", "res05", "res06", "res07", "res08", "res09", "res0a", "res0b", "res0c", "res0d", "res0e", "res0f", "res10", "res11", "res12", "res13", "res14", "res15", "res16", "res17", "res18", "res19", "res1a", "res1b", "res1c", "res1d", "res1e", "res1f", "res20", "res21", "res22", "res23", "res24", "res25", "res26", "res27", "res.tail", "G.text0", "G.text1", "G.text2", "G.text3", "G.text4", "G.text5", "G.text6", "G.text7", "G.text8", "G.text9", "G.text10", "G.text11", "G.text12", "G.text13", "G.text14", "G.text15", "G.text16", "G.text17", "G.text18", "G.text19", "G.text20", "G.text21", "G.text22", "G.text23", "G.c0.c6", "G.c0.cb", "G.c0.e6", "c0.per", "G.c1.c6", "G.c1.cb", "G.c1.e6", "c1.per", "G.c2.c6", "G.c2.cb", "G.c2.e6", "c2.per", "G.c3.c6", "G.c3.cb", "G.c3.e6", "c3.per", "G.c4.c6", "G.c4.cb", "G.c4.e6", "c4.per", "G.c5.c6", "G.c5.cb", "G.c5.e6", "c5.per", "G.c6.c6", "G.c6.cb", "G.c6.e6", "c6.per", "G.c7.c6", "G.c7.cb", "G.c7.e6", "c7.per", "M.tgt0", "M.tgt1", "M.tgt2", "M.tgt3", "M.tgt4", "M.tgt5", "M.tgt6", "M.tgt7", "M.kind", "M.HP", "M.SP", "M.fatigue", "M.shield", "M.shieldDuration", "M.fbf", "M.fc3", "M.per0", "M.per1", "M.per2", "M.lastSource0", "M.lastSource1", "M.lastSource2", "M.lastSource3", "M.lastSource4", "M.lastSource5", "M.lastSource6", "M.lastSource7", "M.lastSource8", "M.lastSource9", "M.lastSource10", "M.lastSource11", "M.lastSource12", "TM.tgt0", "TM.tgt1", "TM.tgt2", "TM.tgt3", "TM.tgt4", "TM.tgt5", "TM.tgt6", "TM.tgt7", "TM.kind", "TM.HP", "TM.SP", "TM.fatigue", "TM.shield", "TM.shieldDuration", "TM.fbf", "TM.fc3", "TM.per0", "TM.per1", "TM.per2", "TM.lastSource0", "TM.lastSource1", "TM.lastSource2", "TM.lastSource3", "TM.lastSource4", "TM.lastSource5", "TM.lastSource6", "TM.lastSource7", "TM.lastSource8", "TM.lastSource9", "TM.lastSource10", "TM.lastSource11", "TM.lastSource12", "C.swings", "C.credit0", "C.credit1", "C.credit2", "C.credit3", "C.credit4", "C.credit5", "C.credit6", "C.credit7", "buf0", "buf1", "buf2", "buf3", "buf4", "buf5", "buf6", "buf7", "buf8", "buf9", "buf10", "buf11", "buf12", "buf13", "buf14", "buf15", "buf16", "buf17", "buf18", "buf19", "buf20", "buf21", "buf22", "buf23", "s4c4ca0_calls", "s4e4db0_a1_0", "s4e4db0_a1_1", "s4e4db0_a1_2", "s4e4db0_a2_0", "s4e4db0_a2_1", "s4e4db0_a2_2", "s4e4db0_a3_0", "s4e4db0_a3_1", "s4e4db0_a3_2", "s4e4db0_calls", "s52e690_a1_0", "s52e690_a1_1", "s52e690_a2_0", "s52e690_a2_1", "s52e690_calls", "s58aad0_a1_0", "s58aad0_a1_1", "s58aad0_a1_2", "s58aad0_a1_3", "s58aad0_a1_4", "s58aad0_a1_5", "s58aad0_a1_6", "s58aad0_a1_7", "s58aad0_a1_8", "s58aad0_a1_9", "s58aad0_a1_10", "s58aad0_a1_11", "s58aad0_a1_12", "s58aad0_a1_13", "s58aad0_a1_14", "s58aad0_a1_15", "s58aad0_a1_16", "s58aad0_a1_17", "s58aad0_a1_18", "s58aad0_a1_19", "s58aad0_a1_20", "s58aad0_a1_21", "s58aad0_a1_22", "s58aad0_a1_23", "s58aad0_a1_24", "s58aad0_a1_25", "s58aad0_a1_26", "s58aad0_a1_27", "s58aad0_a1_28", "s58aad0_a1_29", "s58aad0_a1_30", "s58aad0_a1_31", "s58aad0_a1_32", "s58aad0_a1_33", "s58aad0_a1_34", "s58aad0_a1_35", "s58aad0_a1_36", "s58aad0_a1_37", "s58aad0_a1_38", "s58aad0_a1_39", "s58aad0_a2_0", "s58aad0_a2_1", "s58aad0_a2_2", "s58aad0_a2_3", "s58aad0_a2_4", "s58aad0_a2_5", "s58aad0_a2_6", "s58aad0_a2_7", "s58aad0_a2_8", "s58aad0_a2_9", "s58aad0_a2_10", "s58aad0_a2_11", "s58aad0_a2_12", "s58aad0_a2_13", "s58aad0_a2_14", "s58aad0_a2_15", "s58aad0_a2_16", "s58aad0_a2_17", "s58aad0_a2_18", "s58aad0_a2_19", "s58aad0_a2_20", "s58aad0_a2_21", "s58aad0_a2_22", "s58aad0_a2_23", "s58aad0_a2_24", "s58aad0_a2_25", "s58aad0_a2_26", "s58aad0_a2_27", "s58aad0_a2_28", "s58aad0_a2_29", "s58aad0_a2_30", "s58aad0_a2_31", "s58aad0_a2_32", "s58aad0_a2_33", "s58aad0_a2_34", "s58aad0_a2_35", "s58aad0_a2_36", "s58aad0_a2_37", "s58aad0_a2_38", "s58aad0_a2_39", "s58aad0_a3_0", "s58aad0_a3_1", "s58aad0_a3_2", "s58aad0_a3_3", "s58aad0_a3_4", "s58aad0_a3_5", "s58aad0_a3_6", "s58aad0_a3_7", "s58aad0_a3_8", "s58aad0_a3_9", "s58aad0_a3_10", "s58aad0_a3_11", "s58aad0_a3_12", "s58aad0_a3_13", "s58aad0_a3_14", "s58aad0_a3_15", "s58aad0_a3_16", "s58aad0_a3_17", "s58aad0_a3_18", "s58aad0_a3_19", "s58aad0_a3_20", "s58aad0_a3_21", "s58aad0_a3_22", "s58aad0_a3_23", "s58aad0_a3_24", "s58aad0_a3_25", "s58aad0_a3_26", "s58aad0_a3_27", "s58aad0_a3_28", "s58aad0_a3_29", "s58aad0_a3_30", "s58aad0_a3_31", "s58aad0_a3_32", "s58aad0_a3_33", "s58aad0_a3_34", "s58aad0_a3_35", "s58aad0_a3_36", "s58aad0_a3_37", "s58aad0_a3_38", "s58aad0_a3_39", "s58aad0_calls", "s58ac00_a1_0", "s58ac00_a1_1", "s58ac00_a1_2", "s58ac00_a1_3", "s58ac00_a1_4", "s58ac00_a1_5", "s58ac00_a1_6", "s58ac00_a1_7", "s58ac00_a2_0", "s58ac00_a2_1", "s58ac00_a2_2", "s58ac00_a2_3", "s58ac00_a2_4", "s58ac00_a2_5", "s58ac00_a2_6", "s58ac00_a2_7", "s58ac00_calls", "s58b410_a1_0", "s58b410_a1_1", "s58b410_a1_2", "s58b410_a1_3", "s58b410_a1_4", "s58b410_a1_5", "s58b410_a1_6", "s58b410_a1_7", "s58b410_a1_8", "s58b410_a1_9", "s58b410_a1_10", "s58b410_a1_11", "s58b410_a1_12", "s58b410_a1_13", "s58b410_a1_14", "s58b410_a1_15", "s58b410_a1_16", "s58b410_a1_17", "s58b410_a1_18", "s58b410_a1_19", "s58b410_a1_20", "s58b410_a1_21", "s58b410_a1_22", "s58b410_a1_23", "s58b410_a1_24", "s58b410_a1_25", "s58b410_a1_26", "s58b410_a1_27", "s58b410_a1_28", "s58b410_a1_29", "s58b410_a1_30", "s58b410_a1_31", "s58b410_a1_32", "s58b410_a1_33", "s58b410_a1_34", "s58b410_a1_35", "s58b410_a1_36", "s58b410_a1_37", "s58b410_a1_38", "s58b410_a1_39", "s58b410_a2_0", "s58b410_a2_1", "s58b410_a2_2", "s58b410_a2_3", "s58b410_a2_4", "s58b410_a2_5", "s58b410_a2_6", "s58b410_a2_7", "s58b410_a2_8", "s58b410_a2_9", "s58b410_a2_10", "s58b410_a2_11", "s58b410_a2_12", "s58b410_a2_13", "s58b410_a2_14", "s58b410_a2_15", "s58b410_a2_16", "s58b410_a2_17", "s58b410_a2_18", "s58b410_a2_19", "s58b410_a2_20", "s58b410_a2_21", "s58b410_a2_22", "s58b410_a2_23", "s58b410_a2_24", "s58b410_a2_25", "s58b410_a2_26", "s58b410_a2_27", "s58b410_a2_28", "s58b410_a2_29", "s58b410_a2_30", "s58b410_a2_31", "s58b410_a2_32", "s58b410_a2_33", "s58b410_a2_34", "s58b410_a2_35", "s58b410_a2_36", "s58b410_a2_37", "s58b410_a2_38", "s58b410_a2_39", "s58b410_a3_0", "s58b410_a3_1", "s58b410_a3_2", "s58b410_a3_3", "s58b410_a3_4", "s58b410_a3_5", "s58b410_a3_6", "s58b410_a3_7", "s58b410_a3_8", "s58b410_a3_9", "s58b410_a3_10", "s58b410_a3_11", "s58b410_a3_12", "s58b410_a3_13", "s58b410_a3_14", "s58b410_a3_15", "s58b410_a3_16", "s58b410_a3_17", "s58b410_a3_18", "s58b410_a3_19", "s58b410_a3_20", "s58b410_a3_21", "s58b410_a3_22", "s58b410_a3_23", "s58b410_a3_24", "s58b410_a3_25", "s58b410_a3_26", "s58b410_a3_27", "s58b410_a3_28", "s58b410_a3_29", "s58b410_a3_30", "s58b410_a3_31", "s58b410_a3_32", "s58b410_a3_33", "s58b410_a3_34", "s58b410_a3_35", "s58b410_a3_36", "s58b410_a3_37", "s58b410_a3_38", "s58b410_a3_39", "s58b410_a4_0", "s58b410_a4_1", "s58b410_a4_2", "s58b410_a4_3", "s58b410_a4_4", "s58b410_a4_5", "s58b410_a4_6", "s58b410_a4_7", "s58b410_a4_8", "s58b410_a4_9", "s58b410_a4_10", "s58b410_a4_11", "s58b410_a4_12", "s58b410_a4_13", "s58b410_a4_14", "s58b410_a4_15", "s58b410_a4_16", "s58b410_a4_17", "s58b410_a4_18", "s58b410_a4_19", "s58b410_a4_20", "s58b410_a4_21", "s58b410_a4_22", "s58b410_a4_23", "s58b410_a4_24", "s58b410_a4_25", "s58b410_a4_26", "s58b410_a4_27", "s58b410_a4_28", "s58b410_a4_29", "s58b410_a4_30", "s58b410_a4_31", "s58b410_a4_32", "s58b410_a4_33", "s58b410_a4_34", "s58b410_a4_35", "s58b410_a4_36", "s58b410_a4_37", "s58b410_a4_38", "s58b410_a4_39", "s58b410_calls", "s590b40_a1_0", "s590b40_a1_1", "s590b40_a1_2", "s590b40_a1_3", "s590b40_a1_4", "s590b40_a1_5", "s590b40_a1_6", "s590b40_a1_7", "s590b40_a2_0", "s590b40_a2_1", "s590b40_a2_2", "s590b40_a2_3", "s590b40_a2_4", "s590b40_a2_5", "s590b40_a2_6", "s590b40_a2_7", "s590b40_a3_0", "s590b40_a3_1", "s590b40_a3_2", "s590b40_a3_3", "s590b40_a3_4", "s590b40_a3_5", "s590b40_a3_6", "s590b40_a3_7", "s590b40_calls", "s5182e0_a1_0", "s5182e0_a1_1", "s5182e0_a1_2", "s5182e0_a1_3", "s5182e0_a1_4", "s5182e0_a1_5", "s5182e0_a1_6", "s5182e0_a1_7", "s5182e0_a1_8", "s5182e0_a1_9", "s5182e0_a1_10", "s5182e0_a1_11", "s5182e0_a1_12", "s5182e0_a1_13", "s5182e0_a1_14", "s5182e0_a1_15", "s5182e0_a1_16", "s5182e0_a1_17", "s5182e0_a1_18", "s5182e0_a1_19", "s5182e0_a1_20", "s5182e0_a1_21", "s5182e0_a1_22", "s5182e0_a1_23", "s5182e0_a1_24", "s5182e0_a1_25", "s5182e0_a1_26", "s5182e0_a1_27", "s5182e0_a1_28", "s5182e0_a1_29", "s5182e0_a1_30", "s5182e0_a1_31", "s5182e0_a1_32", "s5182e0_a1_33", "s5182e0_a1_34", "s5182e0_a1_35", "s5182e0_a1_36", "s5182e0_a1_37", "s5182e0_a1_38", "s5182e0_a1_39", "s5182e0_a2_0", "s5182e0_a2_1", "s5182e0_a2_2", "s5182e0_a2_3", "s5182e0_a2_4", "s5182e0_a2_5", "s5182e0_a2_6", "s5182e0_a2_7", "s5182e0_a2_8", "s5182e0_a2_9", "s5182e0_a2_10", "s5182e0_a2_11", "s5182e0_a2_12", "s5182e0_a2_13", "s5182e0_a2_14", "s5182e0_a2_15", "s5182e0_a2_16", "s5182e0_a2_17", "s5182e0_a2_18", "s5182e0_a2_19", "s5182e0_a2_20", "s5182e0_a2_21", "s5182e0_a2_22", "s5182e0_a2_23", "s5182e0_a2_24", "s5182e0_a2_25", "s5182e0_a2_26", "s5182e0_a2_27", "s5182e0_a2_28", "s5182e0_a2_29", "s5182e0_a2_30", "s5182e0_a2_31", "s5182e0_a2_32", "s5182e0_a2_33", "s5182e0_a2_34", "s5182e0_a2_35", "s5182e0_a2_36", "s5182e0_a2_37", "s5182e0_a2_38", "s5182e0_a2_39", "s5182e0_a3_0", "s5182e0_a3_1", "s5182e0_a3_2", "s5182e0_a3_3", "s5182e0_a3_4", "s5182e0_a3_5", "s5182e0_a3_6", "s5182e0_a3_7", "s5182e0_a3_8", "s5182e0_a3_9", "s5182e0_a3_10", "s5182e0_a3_11", "s5182e0_a3_12", "s5182e0_a3_13", "s5182e0_a3_14", "s5182e0_a3_15", "s5182e0_a3_16", "s5182e0_a3_17", "s5182e0_a3_18", "s5182e0_a3_19", "s5182e0_a3_20", "s5182e0_a3_21", "s5182e0_a3_22", "s5182e0_a3_23", "s5182e0_a3_24", "s5182e0_a3_25", "s5182e0_a3_26", "s5182e0_a3_27", "s5182e0_a3_28", "s5182e0_a3_29", "s5182e0_a3_30", "s5182e0_a3_31", "s5182e0_a3_32", "s5182e0_a3_33", "s5182e0_a3_34", "s5182e0_a3_35", "s5182e0_a3_36", "s5182e0_a3_37", "s5182e0_a3_38", "s5182e0_a3_39", "s5182e0_calls", "s5499d0_a1_0", "s5499d0_a1_1", "s5499d0_a1_2", "s5499d0_a4_0", "s5499d0_a4_1", "s5499d0_a4_2", "s5499d0_calls", "s48f650_a1_0", "s48f650_a1_1", "s48f650_a1_2", "s48f650_calls", "s4acd80_a1_0", "s4acd80_a1_1", "s4acd80_a1_2", "s4acd80_a2_0", "s4acd80_a2_1", "s4acd80_a2_2", "s4acd80_calls", "s4c6c30_calls", "s4e3af0_a1_0", "s4e3af0_a1_1", "s4e3af0_a1_2", "s4e3af0_calls", "s51b5c0_a1_0", "s51b5c0_a1_1", "s51b5c0_a2_0", "s51b5c0_a2_1", "s51b5c0_calls", "s542720_a4_0", "s542720_a4_1", "s542720_calls", "s544010_a3_0", "s544010_calls", "s545e50_calls", "s536190_a1_0", "s536190_a1_1", "s536190_calls", "s542fc0_a2_0", "s542fc0_a3_0", "s542fc0_calls", "s542960_a3_0", "s542960_calls", "s545a00_a1_0", "s545a00_calls", "s52a890_a1_0", "s52a890_a1_1", "s52a890_a1_2", "s52a890_a2_0", "s52a890_a2_1", "s52a890_a2_2", "s52a890_a4_0", "s52a890_a4_1", "s52a890_a4_2", "s52a890_a5_0", "s52a890_a5_1", "s52a890_a5_2", "s52a890_a6_0", "s52a890_a6_1", "s52a890_a6_2", "s52a890_calls", "s543270_a1_0", "s543270_a3_0", "s543270_a4_0", "s543270_a6_0", "s543270_calls", "s553770_a2_0", "s553770_a3_0", "s553770_calls", "s553540_a2_0", "s553540_a3_0", "s553540_calls", "s5323f0_a2_0", "s5323f0_a3_0", "s5323f0_a4_0", "s5323f0_calls", "s53ffe0_a2_0", "s53ffe0_calls", "s541630_a2_0", "s541630_a3_0", "s541630_calls", "s523a80_a1_0", "s523a80_a2_0", "s523a80_calls", "s52beb0_a1_0", "s52beb0_a1_1", "s52beb0_a1_2", "s52beb0_a3_0", "s52beb0_a3_1", "s52beb0_a3_2", "s52beb0_calls", "s52af50_a1_0", "s52af50_a2_0", "s52af50_calls", "s523c00_a1_0", "s523c00_a1_1", "s523c00_a2_0", "s523c00_a2_1", "s523c00_a3_0", "s523c00_a3_1", "s523c00_a4_0", "s523c00_a4_1", "s523c00_a6_0", "s523c00_a6_1", "s523c00_calls", "s4e4280_a1_0", "s4e4280_a1_1", "s4e4280_calls", "s4e46f0_a1_0", "s4e46f0_a1_1", "s4e46f0_a2_0", "s4e46f0_a2_1", "s4e46f0_calls", "s53c3f0_a1_0", "s53c3f0_a1_1", "s53c3f0_a2_0", "s53c3f0_a2_1", "s53c3f0_calls"}

var m17Stubs = map[uint32]*mcaStub{
	0x4c4ca0: {ret: 0x20004000, retN: 0, rec: [][4]uint32{}, cnt: 0x20004004, wr: []mcaWrite{}},
	0x4e4db0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004008, 3}, {2, 4, 0x20004014, 3}, {3, 4, 0x20004020, 3}}, cnt: 0x2000402c, wr: []mcaWrite{}},
	0x52e690: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004030, 2}, {2, 4, 0x20004038, 2}}, cnt: 0x20004040, wr: []mcaWrite{}},
	0x58aad0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004044, 40}, {2, 4, 0x200040e4, 40}, {3, 4, 0x20004184, 40}}, cnt: 0x20004224, wr: []mcaWrite{}},
	0x58ac00: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004228, 8}, {2, 4, 0x20004248, 8}}, cnt: 0x20004268, wr: []mcaWrite{}},
	0x58b410: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 1, 0x2000426c, 40}, {2, 1, 0x20004294, 40}, {3, 1, 0x200042bc, 40}, {4, 4, 0x200042e4, 40}}, cnt: 0x20004384, wr: []mcaWrite{}},
	0x590b40: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004388, 8}, {2, 4, 0x200043a8, 8}, {3, 4, 0x200043c8, 8}}, cnt: 0x200043e8, wr: []mcaWrite{}},
	0x5182e0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200043ec, 40}, {2, 4, 0x2000448c, 40}, {3, 4, 0x2000452c, 40}}, cnt: 0x200045cc, wr: []mcaWrite{}},
	0x5499d0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200045d0, 3}, {4, 4, 0x200045dc, 3}}, cnt: 0x200045e8, wr: []mcaWrite{}},
	0x48f650: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200045ec, 3}}, cnt: 0x200045f8, wr: []mcaWrite{}},
	0x4acd80: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200045fc, 3}, {2, 4, 0x20004608, 3}}, cnt: 0x20004614, wr: []mcaWrite{}},
	0x4c6c30: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20004618, wr: []mcaWrite{}},
	0x4e3af0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000461c, 3}}, cnt: 0x20004628, wr: []mcaWrite{}},
	0x51b5c0: {ret: 0x2000462c, retN: 0, rec: [][4]uint32{{1, 4, 0x20004630, 2}, {2, 4, 0x20004638, 2}}, cnt: 0x20004640, wr: []mcaWrite{}},
	0x542720: {ret: 0x20004644, retN: 2, rec: [][4]uint32{{4, 4, 0x2000464c, 2}}, cnt: 0x20004654, wr: []mcaWrite{}},
	0x544010: {ret: 0x2000465c, retN: 0, rec: [][4]uint32{{3, 4, 0x20004664, 1}}, cnt: 0x20004668, wr: []mcaWrite{{4, 0x0, 4, 0x20004658, 0}, {4, 0x8, 4, 0x20004660, 0}}},
	0x545e50: {ret: 0x2000466c, retN: 0, rec: [][4]uint32{}, cnt: 0x2000467c, wr: []mcaWrite{{2, 0x0, 4, 0x20004670, 0}, {2, 0x4, 4, 0x20004674, 0}, {2, 0x8, 4, 0x20004678, 0}}},
	0x536190: {ret: 0x20004680, retN: 2, rec: [][4]uint32{{1, 4, 0x20004688, 2}}, cnt: 0x20004690, wr: []mcaWrite{}},
	0x542fc0: {ret: 0x20004694, retN: 0, rec: [][4]uint32{{2, 4, 0x20004698, 1}, {3, 4, 0x2000469c, 1}}, cnt: 0x200046a0, wr: []mcaWrite{}},
	0x542960: {ret: 0x200046a4, retN: 0, rec: [][4]uint32{{3, 4, 0x200046ac, 1}}, cnt: 0x200046b0, wr: []mcaWrite{{3, 0x0, 4, 0x200046a8, 0}}},
	0x545a00: {ret: 0x200046b4, retN: 0, rec: [][4]uint32{{1, 4, 0x200046b8, 1}}, cnt: 0x200046bc, wr: []mcaWrite{}},
	0x52a890: {ret: 0x200046c0, retN: 0, rec: [][4]uint32{{1, 4, 0x200046c4, 3}, {2, 4, 0x200046d0, 3}, {4, 1, 0x200046dc, 3}, {5, 1, 0x200046e0, 3}, {6, 4, 0x200046e4, 3}}, cnt: 0x200046f0, wr: []mcaWrite{}},
	0x543270: {ret: 0x200046f4, retN: 0, rec: [][4]uint32{{1, 4, 0x200046f8, 1}, {3, 4, 0x200046fc, 1}, {4, 1, 0x20004700, 1}, {6, 4, 0x20004704, 1}}, cnt: 0x20004708, wr: []mcaWrite{}},
	0x553770: {ret: 0x2000470c, retN: 0, rec: [][4]uint32{{2, 4, 0x20004710, 1}, {3, 4, 0x20004714, 1}}, cnt: 0x20004718, wr: []mcaWrite{}},
	0x553540: {ret: 0x2000471c, retN: 0, rec: [][4]uint32{{2, 4, 0x20004720, 1}, {3, 4, 0x20004724, 1}}, cnt: 0x20004728, wr: []mcaWrite{}},
	0x5323f0: {ret: 0x2000472c, retN: 0, rec: [][4]uint32{{2, 4, 0x20004740, 1}, {3, 4, 0x20004744, 1}, {4, 4, 0x20004748, 1}}, cnt: 0x2000474c, wr: []mcaWrite{{1, 0x2e1, 4, 0x20004730, 0}, {1, 0x2ba, 4, 0x20004734, 0}, {1, 0x2be, 4, 0x20004738, 0}, {1, 0x2c2, 4, 0x2000473c, 0}}},
	0x53ffe0: {ret: 0x20004750, retN: 0, rec: [][4]uint32{{2, 4, 0x20004754, 1}}, cnt: 0x20004758, wr: []mcaWrite{}},
	0x541630: {ret: 0x2000475c, retN: 0, rec: [][4]uint32{{2, 4, 0x20004760, 1}, {3, 4, 0x20004764, 1}}, cnt: 0x20004768, wr: []mcaWrite{}},
	0x523a80: {ret: 0x2000476c, retN: 0, rec: [][4]uint32{{1, 4, 0x20004770, 1}, {2, 4, 0x20004774, 1}}, cnt: 0x20004778, wr: []mcaWrite{}},
	0x52beb0: {ret: 0x2000477c, retN: 0, rec: [][4]uint32{{1, 4, 0x20004780, 3}, {3, 4, 0x2000478c, 3}}, cnt: 0x20004798, wr: []mcaWrite{}},
	0x52af50: {ret: 0x2000479c, retN: 0, rec: [][4]uint32{{1, 4, 0x200047a0, 1}, {2, 4, 0x200047a4, 1}}, cnt: 0x200047a8, wr: []mcaWrite{}},
	0x523c00: {ret: 0x200047ac, retN: 0, rec: [][4]uint32{{1, 4, 0x200047b0, 2}, {2, 4, 0x200047b8, 2}, {3, 4, 0x200047c0, 2}, {4, 4, 0x200047c8, 2}, {6, 1, 0x200047d0, 2}}, cnt: 0x200047d4, wr: []mcaWrite{}},
	0x4e4280: {ret: 0x200047d8, retN: 0, rec: [][4]uint32{{1, 4, 0x200047dc, 2}}, cnt: 0x200047e4, wr: []mcaWrite{}},
	0x4e46f0: {ret: 0x200047e8, retN: 0, rec: [][4]uint32{{1, 4, 0x200047ec, 2}, {2, 1, 0x200047f4, 2}}, cnt: 0x200047f8, wr: []mcaWrite{}},
	0x53c3f0: {ret: 0x200047fc, retN: 2, rec: [][4]uint32{{1, 4, 0x20004804, 2}, {2, 4, 0x2000480c, 2}}, cnt: 0x20004814, wr: []mcaWrite{}},
}

// Parked default worlds of the F callees: the first boundary row of each
// callee's own case (re/emu/cases/00<addr>.json).
var marPark542720 = [50]int64{3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 2, 256, 1, 0, 0, 0, 0, 123, 0, -83490646, -45, 536928912, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0}

var marPark544010 = [85]int64{0, 0, 0, 123, 124, 1, 100, 1, 1, 100, 1, 2, 16, 0, 536981912, 536981944, 536981976, 536982008, 536982040, 536982072, 536982104, 536982136, 536982168, 536982200, 536982232, 536982264, 536982296, 536982328, 536982360, 536982392, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

var marPark542960 = [42]int64{3, 0, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 536936448, 536895488, 536899584, 536903680, 5, 6, 7, 8, 9, 10, 0, 0, 0, 0, 1, 6, 0, 0, 0, 0, 0}

var marPark545e50 = [907]int64{1, 0, 20, 1, 0, 99, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 12, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 13, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 396, 0, 0, 0, 0, 0, 396, 0, 0, 0, 0, 0, 0, 20, 20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 4, 1, 1, 536936448, 0, 1, 16, 0, 1, 1, 1, -1, -1, 1, 0, 0, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 0, 0, 0, 0, 100, 20, 0, 1, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 5, 1, 10, 10, 10, 10, 10, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 65535, 4294967295, 3, 255, 0, 255, 0, 255, 0, 255, 0, 1, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 1, 1, 1, 0, 0, 0, 0, 0, 0, 4, 0, 1, 0, 0, 101, 102, 103, 536936448, 201, 202, 203, 536936448, 77, 301, 302, 303, 304, 3, 1, 0, 1, 100, 1, 0, 1, 0, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 1, 0, 2, 3, 4, 5, 0, 1, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 0, 0, 0, 0, 0, 0, 1, 39, 100, 0, 0, 11, 0, 10, 2, 1, 39, 100, 1, 0, 11, 2, 0, 0, 0, 1, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
