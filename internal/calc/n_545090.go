package calc

import "fmt"

// ---- 0x545090: missile penetration resolution ----
//
// A missile (record p1, source descriptor p1+0x22c, target descriptor p1+0x260)
// that 0x536190 still finds valid writes "<text> <target name>[ <text>]" to the
// combat log (0x58ac00, portrait highlight 0x58b410 when the target side 0x53c3f0
// differs from the source side 0x53c320). Unless aborted, it rolls the hit
// location (monster: the template's seven location weights, else the fixed
// weights at 0x616308), the penetration chance 50 + mode modifier (0x542e10)
// + 5*(10 - defPen (0x542fc0)) against d100, and on a pass applies the effect
// quantity (0x551a20) as damage: 0x52bb60 on a monster, 0x52a890 on a character,
// then 0x543270.
//
// The world is the emulator case's memory image (mstWorld, built from
// re/emu/cases/00545090.json's description). 排除 callees are address stubs
// whose arguments and call counts are outputs. The D4 callees 0x536190,
// 0x53c3f0, 0x542fc0 and 0x551a20 are address stubs whose returns this node
// reads: Emu replays them, F runs their Go nodes on the live values (parked
// worlds for the rest), so F and Emu differ wherever the case's stub inputs
// differ from what those nodes compute. 0x52bb60 runs as code (unregistered)
// with its D4 callees (0x523c00, 0x52beb0, 0x4e4280, 0x4e46f0) stubbed.

const (
	mpnStrTable = 0x20008000 // fixture [0x68c09c]: string pointer table
	mpnStrText  = 0x2000a000 // fixture string i: u16 {i+1, 0}
	mpnStrCount = 2048       // fixture cap: string ids 0..2047
	// 0x545090's stack buffers in the emulator (initial esp 0x10008000).
	mpnLine     = 0x10007f10 // the log line (esp+0x50 after the four pushes)
	mpnLocation = 0x10007ee8 // the location text (esp+0x28)
	// 0x4ed260's vsprintf buffer is outside the case's outputs; its address is
	// only an unrecorded 0x58aad0 argument.
	mpnTraceBuf = 0
)

func mpnStrings(s mstSpec) mstSpec {
	for i := uint32(0); i < mpnStrCount; i++ {
		s.set = append(s.set, mstSet{mpnStrTable + 4*i, 4, mpnStrText + 4*i}, mstSet{mpnStrText + 4*i, 2, i + 1})
	}
	return s
}

// str is the string pointer [[0x68c09c] + 4i].
func mpnStr(w *mcaWorld, i uint32) uint32 {
	if i >= mpnStrCount {
		panic("missile penetration: string id past the fixture's 2048")
	}
	return srdMsg(w, i)
}

func mpnCat(w *mcaWorld, dst, src uint32) { srdPutW(w, dst, srdWstr(w, dst)+srdWstr(w, src)) }

func mpnLen(w *mcaWorld, a uint32) uint32 { return uint32(len([]rune(srdWstr(w, a)))) }

// side is 0x53c320: 8 party, 9 monsters, 12 neither.
func mpnSide(w *mcaWorld, x uint32) uint32 {
	t := w.u32(x)
	switch {
	case t == 1:
		if w.u32(x+4) == 0xffffffff {
			panic("missile penetration: character source without index (0x53c331 asserts)")
		}
		return 8
	case t == 3 && w.u32(x+4) != 0xffffffff:
		if w.u8(x+0x1c) == 0 && w.u8(x+0x1b) == 0 {
			panic("missile penetration: type-3 party source without flags (0x53c369 asserts)")
		}
		return 8
	case t == 2:
		if w.u32(x+8) == 0xffffffff {
			panic("missile penetration: monster source without id (0x53c390 asserts)")
		}
		return 9
	case t == 3 && w.u32(x+8) != 0xffffffff:
		if w.u8(x+0x1c) == 0 && w.u8(x+0x1b) == 0 {
			panic("missile penetration: type-3 monster source without flags (0x53c3c5 asserts)")
		}
		return 9
	}
	return 12
}

// name is 0x4e5150(m, 0, 0).
func mpnName(w *mcaWorld, m uint32) uint32 {
	t := w.template(m)
	if w.u16(t+0x187) == 0x18d {
		srdPutW(w, 0x6875c3, srdFormat(w, 0x614b44, w.charBase(w.u32(0x6875ef))+5))
		return 0x6875c3
	}
	if w.u32(m+4) == 0 {
		if !w.act15(w.u32(m + 0xc)) {
			w.stub(0x5182e0, 1, 0x614b20, w.u32(m))
		}
		return t + 0x60
	}
	if w.u8(w.group(w.u32(m+4))+0x2c) == 0 {
		return t + 0x60
	}
	return t
}

// valid is 0x536190(tgt).
func mpnValid(w *mcaWorld, tgt uint32) bool {
	k := w.call(0x536190, tgt)
	if !w.real {
		return byte(w.replay(0x536190, k)) != 0
	}
	d := func(a uint32) int64 { return int64(w.u32(a)) }
	x := mprFeed(mpnPark536190[:], []string{"targetPointer", "targetKind", "characterIndex", "characterPresent",
		"characterHP", "characterPrimaryStatus", "resurrectionAvailable", "resurrectionBlocked", "monsterID", "groupID"},
		map[string]int64{"targetKind": d(tgt), "characterIndex": d(tgt + 4), "monsterID": d(tgt + 8), "groupID": d(tgt + 0xc)})
	return byte(combatTargetValidNode(x, w.r)[0]) != 0
}

// targetSide is 0x53c3f0(src, tgt).
func mpnTargetSide(w *mcaWorld, src, tgt uint32) byte {
	k := w.call(0x53c3f0, src, tgt)
	if !w.real {
		return byte(w.replay(0x53c3f0, k))
	}
	x := []int64{int64(w.u32(src)), int64(int32(w.u32(src + 4))), int64(w.u8(src + 0x1c)), int64(w.u8(src + 0x1b)),
		int64(w.u32(tgt)), int64(int32(w.u32(tgt + 4)))}
	side := w.u32(0x685178)
	for i := uint32(0); i < 8; i++ {
		x = append(x, int64(w.u8(side+i*0x106+0xf1)))
	}
	return byte(combatTargetEffectSelectorNode(x, w.r)[0])
}

// defPen is 0x542fc0(tgt, mode, location).
func mpnDefPen(w *mcaWorld, tgt, mode, loc uint32) int32 {
	k := w.call(0x542fc0, tgt, mode, loc)
	if !w.real {
		return int32(w.replay(0x542fc0, k, tgt, mode, loc))
	}
	x := append([]int64(nil), mpnPark542fc0[:]...)
	x[0], x[1], x[28] = int64(w.u32(tgt)), int64(mode), int64(loc) // targetType, mode, hitLocation
	return int32(defpenNode(x, w.r)[0])
}

// quantity is 0x551a20(pack).
func mpnQuantity(w *mcaWorld, pack uint32) uint32 {
	k := w.call(0x551a20)
	if !w.real {
		return w.replay(0x551a20, k)
	}
	x := []int64{int64(int16(w.u16(pack + 4))), int64(w.u8(pack + 6)), int64(w.u8(pack + 7)), int64(int32(w.u32(pack + 0x24)))}
	return uint32(effectQuantityNode(x, w.r)[0])
}

// mpnWeights is the non-monster hit-location weights (exe 0x616308, five bytes).
func mpnWeights(w *mcaWorld) [5]uint32 {
	var c [5]uint32
	for i := range c {
		c[i] = uint32(w.u8(0x616308 + uint32(i)))
	}
	return c
}

func mpnBody(w *mcaWorld) {
	p1 := mcaArg(w, 0)
	abort := byte(mcaArg(w, 1))
	src, tgt := p1+0x22c, p1+0x260
	if !mpnValid(w, tgt) {
		return
	}
	srdSource(w, src, false) // 0x53bea0(src, 0): only its assert matters
	srdPutW(w, mpnLine, srdFormat(w, 0x60cc78, mpnStr(w, 0x1bf)))
	len1 := mpnLen(w, mpnLine)
	if w.u32(tgt) == 3 {
		mpnCat(w, mpnLine, mpnName(w, w.findMonster(w.u32(tgt+8))))
	} else {
		mpnCat(w, mpnLine, w.charBase(w.u32(tgt+4))+5)
	}
	if w.u8(p1+0x322) != 0 {
		mpnCat(w, mpnLine, 0x60cc74)
		mpnCat(w, mpnLine, mpnStr(w, 0x1c0))
	}
	len2 := mpnLen(w, mpnLine)
	c1 := mpnSide(w, src)
	c2 := mpnTargetSide(w, src, tgt)
	w.stub(0x58ac00, c1, mpnLine, 0xffffffff, 0xffffffff, 0)
	if c2 != byte(c1) {
		w.stub(0x58b410, uint32(c2), len1&0xff, len2&0xff, 0xffffffff)
	}
	if abort != 0 {
		w.stub(0x58ac00, c1, mpnStr(w, 0x214), 0xffffffff, 0xffffffff, 0)
		return
	}
	var loc uint32
	var m uint32
	if w.u32(tgt) == 3 {
		m = w.findMonster(w.u32(tgt + 8))
		t := w.template(m)
		roll, cum := uint32(w.r.R(100)), uint32(0)
		loc = 7
		for i := uint32(0); i < 7; i++ {
			cum += uint32(w.u8(t + 0x15f + i))
			if roll < cum {
				loc = i
				break
			}
		}
		if loc == 7 {
			w.stub(0x5182e0, 0, 0x61d548, cum, t)
			loc = 3
		}
		if w.u8(t+0x15f+loc) < 100 {
			id := w.u16(0x61ea24 + 2*(uint32(w.u8(t+0x15e))+6*loc))
			srdPutW(w, mpnLocation, srdWstr(w, mpnStr(w, id)))
		} else {
			srdPutW(w, mpnLocation, srdWstr(w, 0x689b34))
		}
	} else {
		roll, cum := uint32(w.r.R(100)), uint32(0)
		loc = 5
		for i, x := range mpnWeights(w) {
			cum += x
			if roll < cum {
				loc = uint32(i)
				break
			}
		}
		if loc == 5 {
			panic("missile penetration: d100 past the location weights summing to 100 (block 0x545318)")
		}
		srdPutW(w, mpnLocation, srdWstr(w, mpnStr(w, w.u16(0x61e7b0+4*loc))))
	}
	if w.u8(0x68510c) != 0 {
		w.stub(0x58aad0, c1, mpnStr(w, 0x20c), mpnLocation)
	}
	mode := w.u32(w.u32(0x65bde0) + 485*w.u32(p1+0x1d8) + 0x144)
	if mode > 8 {
		panic(fmt.Sprintf("missile penetration: attack mode %d past 8 (0x542e10 asserts)", mode))
	}
	chance := 50 + attackModePenModifier[mode] + 5*(10-mpnDefPen(w, tgt, mode, loc))
	roll := int32(w.r.R(100)) + 1
	// 0x4ed260: the trace line goes to 0x58aad0 only with the trace flag.
	if w.u8(0x68d810) != 0 {
		w.stub(0x58aad0, 7, 0x617664, mpnTraceBuf)
	}
	if roll > chance {
		w.stub(0x58aad0, c1, mpnStr(w, 0x20f))
		w.stub(0x54a0e0, p1, tgt, loc, 0xffffffff)
		return
	}
	pack := p1 + 0x1fc
	dmg := mpnQuantity(w, pack)
	if dmg == 0 {
		if w.u8(0x68510c) != 0 {
			w.stub(0x58aad0, c1, mpnStr(w, 0x20e))
		}
		return
	}
	w.stub(0x54a0e0, p1, tgt, loc, 0xffffffff)
	if w.u32(tgt) == 3 {
		mpnHit(w, m, dmg, src)
	} else {
		w.stub(0x52a890, w.u32(tgt+4), dmg, 0, 1, 1, 0, 0)
	}
	w.stub(0x543270, src, tgt, pack, 1, 0, 0)
}

// mpnHit is 0x52bb60(m, dmg, src, 0, 1, 1, 0, 0): no result record, the notice
// without the name (text 0x254), death message flag 1.
func mpnHit(w *mcaWorld, m, dmg, src uint32) {
	if w.u32(m+0x2b) == 0 {
		return
	}
	if w.u32(m+0xc7) != 0 {
		msg, shield := mpnStr(w, 0x193), w.u16(m+0xc5)
		if shield >= dmg {
			w.stub(0x590b40, m, msg, dmg)
			w.put(m+0xc5, 2, uint64(shield-dmg))
			return
		}
		w.stub(0x590b40, m, msg, shield)
		dmg -= shield
		// 0x524390(id, 2): clear buff slot 2, 0x4acd80, rebuild.
		w.put(m+0xbf, 4, 0)
		w.put(m+0xc3, 4, 0)
		w.put(m+0xc7, 4, 0)
		w.stub(0x4acd80, w.u32(m+0xc), 0x12, 0)
		w.rebuild(w.u32(m))
	}
	if dmg == 0 {
		panic("missile penetration: zero damage reaches 0x52bb60's tail (0x545090 passes only non-zero)")
	}
	if w.u8(0x683f94) != 0 || w.u8(m+0x2ab) != 0 {
		w.stub(0x48f650, m, 0, 1)
		who := uint32(9)
		if srdSource(w, src, false) && w.u32(src+4) != 0xffffffff {
			who = 8
		}
		w.stub(0x58aad0, who, mpnStr(w, 0x254), dmg)
	}
	if w.u8(src+0x1c) == 0 && w.u8(src+0x1b) == 0 && w.u8(src+0x1e) == 0 {
		for i := uint32(0); i < 13; i++ {
			w.put(m+0x304+4*i, 4, uint64(w.u32(src+4*i)))
		}
		if w.u8(m+0x15) != 0 && srdSource(w, src, false) && w.u32(src+4) != 0xffffffff {
			idx := w.u32(src + 4)
			if idx >= 8 {
				panic("missile penetration: crediting party index past 7")
			}
			a := w.u32(m+0x10) + 0x1a + 4*idx
			w.put(a, 4, uint64(w.u32(a)+dmg))
		}
	}
	if dmg >= w.u32(m+0x2b) {
		w.put(m+0x2b, 4, 0)
		w.downed(m, 1)
	} else {
		w.put(m+0x2b, 4, uint64(w.u32(m+0x2b)-dmg))
		mpnFatigue(w, m, 2*dmg/3)
		w.stub(0x52beb0, m, src, 0)
	}
	// 0x56a2a0 is idle with UI object 0x68edcc = 0; 0x4e5a50 -> 0x4c6a50
	// allocates its 0x14-byte collection, finds the fixture's empty one and
	// returns 0; 0x4c6c30 is an excluded presentation stub, with the post-shield damage argument.
	if w.u32(0x68edcc) != 0 {
		panic("missile penetration: UI object outside the fixture")
	}
	if a := w.u32(m + 0xc); a != 0 {
		w.alloc(0x14)
		w.stub(0x4c6c30, dmg)
	}
}

// mpnFatigue is 0x52c070(m, cost, 0).
func mpnFatigue(w *mcaWorld, m, cost uint32) {
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
		panic("missile penetration: max SP 0 divides by zero (0x52c0b8)")
	}
	w.put(m+0x242, 4, uint64(staminaFatigueLevel(sp, most)))
	if sp == 0 && w.u32(m+0x9b) < 0x270f {
		w.stub(0x523c00, w.u32(m), 0x11, 0x270f, 0, 0, 1)
	}
}

func mpnRun(real bool) func([]int64, Rand) []int64 {
	return func(in []int64, r Rand) []int64 {
		if len(in) != len(mpnSpec.in) {
			panic("missile penetration: wrong input count")
		}
		w := &mcaWorld{mstWorld: mstBuild(&mpnSpec, in), real: real, tab: mpnStubs}
		w.r = r
		mpnBody(w)
		out := make([]int64, len(mpnSpec.out))
		for i, f := range mpnSpec.out {
			out[i] = w.read(f, in)
		}
		return out
	}
}

func missilePenetrationResolutionNode(in []int64, r Rand) []int64 { return mpnRun(true)(in, r) }

func mpnEmu(in []int64, r Rand) []int64 { return mpnRun(false)(in, r) }

func init() {
	Nodes["00545090"] = Node{Addr: "00545090", In: mpnIn, Out: mpnOut, F: missilePenetrationResolutionNode, Emu: mpnEmu}
}

// Code below is generated from the same description as re/emu/cases/00545090.json.

var mpnSpec = mpnStrings(mstSpec{
	set: []mstSet{
		{0x7f000004, 4, 0x20000000},
		{0x7f000008, 4, 0x0},
		{0x683fad, 4, 0x200036a0},
		{0x683fb9, 4, 0x0},
		{0x683fb1, 4, 0x20003640},
		{0x683fbd, 4, 0x0},
		{0x6840c7, 4, 0x20000c00},
		{0x6840cb, 4, 0x20000f00},
		{0x68c09c, 4, 0x20008000},
		{0x65bde0, 4, 0x20003800},
		{0x685174, 4, 0x20010000},
		{0x685178, 4, 0x2000c000},
		{0x68ec78, 4, 0x0},
		{0x68edcc, 4, 0x0},
		{0x689b34, 2, 0x0},
		{0x683f97, 1, 0x0},
		{0x683f9b, 1, 0x0},
		{0x6598b8, 4, 0x20002540},
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
		{0x20000400, 4, 0x5},
		{0x20000408, 4, 0x0},
		{0x2000040c, 4, 0x20001200},
		{0x20000410, 4, 0x20003400},
		{0x200004bf, 4, 0x5150},
		{0x200004c3, 2, 0x6162},
		{0x20001200, 4, 0x5ed22c},
		{0x200013d8, 4, 0x20001400},
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
		{0x20000800, 4, 0x6},
		{0x20000808, 4, 0x1},
		{0x2000080c, 4, 0x20001300},
		{0x20000810, 4, 0x20003500},
		{0x200008bf, 4, 0x5151},
		{0x200008c3, 2, 0x6163},
		{0x20001300, 4, 0x5ed22c},
		{0x200014d8, 4, 0x20001c00},
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
		{0x2000082b, 4, 0x1e},
		{0x2000082f, 4, 0xa},
		{0x20000833, 4, 0xa},
		{0x2000089b, 4, 0x2710},
		{0x20000815, 1, 0x1},
		{0x20001ca7, 1, 0x14},
		{0x20001087, 2, 0x18c},
		{0x2000105e, 1, 0x1},
		{0x2000105f, 1, 0xf},
		{0x20001060, 1, 0xf},
		{0x20001061, 1, 0xf},
		{0x20001062, 1, 0xf},
		{0x20001063, 1, 0xf},
		{0x20001064, 1, 0xf},
		{0x20001065, 1, 0xf},
		{0x20000c00, 2, 0x41},
		{0x20000c02, 2, 0x62},
		{0x20000c60, 2, 0x43},
		{0x20000c62, 2, 0x64},
		{0x20000c64, 2, 0x65},
		{0x20000c66, 2, 0x66},
		{0x20000f00, 2, 0x47},
		{0x20000f60, 2, 0x48},
		{0x20000f62, 2, 0x69},
		{0x20000f64, 2, 0x6a},
		{0x20000f66, 2, 0x6b},
		{0x20000f68, 2, 0x6c},
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
		{0x2000023c, 4, 0x5c000010},
		{0x20000240, 4, 0x5c000014},
		{0x2000024c, 4, 0x5c000020},
		{0x20000250, 4, 0x5c000024},
		{0x20000254, 4, 0x5c000028},
		{0x20000258, 4, 0x5c00002c},
		{0x2000025c, 4, 0x5c000030},
		{0x20000244, 1, 0x98},
		{0x20000245, 1, 0x99},
		{0x20000246, 1, 0x9a},
		{0x20000249, 1, 0x9d},
		{0x2000024b, 1, 0x9f},
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
		{0x10007ee8, 2, 0xbeef},
		{0x10007eea, 2, 0xbeee},
		{0x10007f10, 2, 0xbe00},
		{0x10007f12, 2, 0xbe01},
		{0x10007f14, 2, 0xbe02},
		{0x10007f16, 2, 0xbe03},
		{0x10007f18, 2, 0xbe04},
		{0x10007f1a, 2, 0xbe05},
		{0x10007f1c, 2, 0xbe06},
		{0x10007f1e, 2, 0xbe07},
		{0x10007f20, 2, 0xbe08},
		{0x10007f22, 2, 0xbe09},
		{0x10007f24, 2, 0xbe0a},
		{0x10007f26, 2, 0xbe0b},
		{0x10007f28, 2, 0xbe0c},
		{0x10007f2a, 2, 0xbe0d},
		{0x10007f2c, 2, 0xbe0e},
		{0x10007f2e, 2, 0xbe0f},
	},
	in: []mstFld{
		{addr: 0x7f000008, size: 1, signed: false, idx: -1},
		{addr: 0x200001d8, size: 4, signed: false, idx: -1},
		{addr: 0x20003944, size: 4, signed: false, idx: -1},
		{addr: 0x20003b29, size: 4, signed: false, idx: -1},
		{addr: 0x20003d0e, size: 4, signed: false, idx: -1},
		{addr: 0x20003ef3, size: 4, signed: false, idx: -1},
		{addr: 0x20000322, size: 1, signed: false, idx: -1},
		{addr: 0x20000200, size: 2, signed: true, idx: -1},
		{addr: 0x20000202, size: 1, signed: false, idx: -1},
		{addr: 0x20000203, size: 1, signed: false, idx: -1},
		{addr: 0x20000220, size: 4, signed: true, idx: -1},
		{addr: 0x2000022c, size: 4, signed: false, idx: -1},
		{addr: 0x20000230, size: 4, signed: true, idx: -1},
		{addr: 0x20000234, size: 4, signed: true, idx: -1},
		{addr: 0x20000238, size: 4, signed: false, idx: -1},
		{addr: 0x20000247, size: 1, signed: false, idx: -1},
		{addr: 0x20000248, size: 1, signed: false, idx: -1},
		{addr: 0x2000024a, size: 1, signed: false, idx: -1},
		{addr: 0x20000260, size: 4, signed: false, idx: -1},
		{addr: 0x20000264, size: 4, signed: false, idx: -1},
		{addr: 0x20000268, size: 4, signed: false, idx: -1},
		{addr: 0x2000026c, size: 4, signed: false, idx: -1},
		{addr: 0x68510c, size: 1, signed: false, idx: -1},
		{addr: 0x683f94, size: 1, signed: false, idx: -1},
		{addr: 0x68d810, size: 1, signed: false, idx: -1},
		{addr: 0x6875ef, size: 4, signed: false, idx: -1},
		{addr: 0x20000404, size: 4, signed: false, idx: -1},
		{addr: 0x20000415, size: 1, signed: false, idx: -1},
		{addr: 0x2000042b, size: 4, signed: false, idx: -1},
		{addr: 0x2000042f, size: 4, signed: false, idx: -1},
		{addr: 0x20000433, size: 4, signed: false, idx: -1},
		{addr: 0x2000049b, size: 4, signed: false, idx: -1},
		{addr: 0x200004c5, size: 2, signed: false, idx: -1},
		{addr: 0x200004c7, size: 4, signed: false, idx: -1},
		{addr: 0x200006ab, size: 1, signed: false, idx: -1},
		{addr: 0x20000d87, size: 2, signed: false, idx: -1},
		{addr: 0x20000d5e, size: 1, signed: false, idx: -1},
		{addr: 0x20000d5f, size: 1, signed: false, idx: -1},
		{addr: 0x20000d60, size: 1, signed: false, idx: -1},
		{addr: 0x20000d61, size: 1, signed: false, idx: -1},
		{addr: 0x20000d62, size: 1, signed: false, idx: -1},
		{addr: 0x20000d63, size: 1, signed: false, idx: -1},
		{addr: 0x20000d64, size: 1, signed: false, idx: -1},
		{addr: 0x20000d65, size: 1, signed: false, idx: -1},
		{addr: 0x2000362c, size: 1, signed: false, idx: -1},
		{addr: 0x200014a4, size: 1, signed: false, idx: -1},
		{addr: 0x200014a7, size: 1, signed: false, idx: -1},
		{addr: 0x2000c0f1, size: 1, signed: false, idx: -1},
		{addr: 0x2000c1f7, size: 1, signed: false, idx: -1},
		{addr: 0x2000c2fd, size: 1, signed: false, idx: -1},
		{addr: 0x2000c403, size: 1, signed: false, idx: -1},
		{addr: 0x2000c509, size: 1, signed: false, idx: -1},
		{addr: 0x2000c60f, size: 1, signed: false, idx: -1},
		{addr: 0x2000c715, size: 1, signed: false, idx: -1},
		{addr: 0x2000c81b, size: 1, signed: false, idx: -1},
		{addr: 0x20004098, size: 4, signed: false, idx: -1},
		{addr: 0x200040a0, size: 4, signed: false, idx: -1},
		{addr: 0x200040a8, size: 4, signed: true, idx: -1},
		{addr: 0x200040b8, size: 4, signed: false, idx: -1},
		{addr: 0x200013dc, size: 4, signed: false, idx: -1},
		{addr: 0x200014dc, size: 4, signed: false, idx: -1},
	},
	out: []mstFld{
		{addr: 0x2000042b, size: 4, signed: false, idx: -1},
		{addr: 0x20000433, size: 4, signed: false, idx: -1},
		{addr: 0x20000642, size: 4, signed: false, idx: -1},
		{addr: 0x200004c5, size: 2, signed: false, idx: -1},
		{addr: 0x200004bf, size: 4, signed: false, idx: -1},
		{addr: 0x200004c3, size: 2, signed: false, idx: -1},
		{addr: 0x200004c7, size: 4, signed: false, idx: -1},
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
		{addr: 0x2000341a, size: 4, signed: false, idx: -1},
		{addr: 0x2000341e, size: 4, signed: false, idx: -1},
		{addr: 0x20003422, size: 4, signed: false, idx: -1},
		{addr: 0x20003426, size: 4, signed: false, idx: -1},
		{addr: 0x2000342a, size: 4, signed: false, idx: -1},
		{addr: 0x2000342e, size: 4, signed: false, idx: -1},
		{addr: 0x20003432, size: 4, signed: false, idx: -1},
		{addr: 0x20003436, size: 4, signed: false, idx: -1},
		{addr: 0x2000082b, size: 4, signed: false, idx: -1},
		{addr: 0x6875c3, size: 2, signed: false, idx: -1},
		{addr: 0x6875c5, size: 2, signed: false, idx: -1},
		{addr: 0x6875c7, size: 2, signed: false, idx: -1},
		{addr: 0x6875c9, size: 2, signed: false, idx: -1},
		{addr: 0x6875cb, size: 2, signed: false, idx: -1},
		{addr: 0x6875cd, size: 2, signed: false, idx: -1},
		{addr: 0x6875cf, size: 2, signed: false, idx: -1},
		{addr: 0x6875d1, size: 2, signed: false, idx: -1},
		{addr: 0x6875d3, size: 2, signed: false, idx: -1},
		{addr: 0x6875d5, size: 2, signed: false, idx: -1},
		{addr: 0x6875d7, size: 2, signed: false, idx: -1},
		{addr: 0x6875d9, size: 2, signed: false, idx: -1},
		{addr: 0x10007f10, size: 2, signed: false, idx: -1},
		{addr: 0x10007f12, size: 2, signed: false, idx: -1},
		{addr: 0x10007f14, size: 2, signed: false, idx: -1},
		{addr: 0x10007f16, size: 2, signed: false, idx: -1},
		{addr: 0x10007f18, size: 2, signed: false, idx: -1},
		{addr: 0x10007f1a, size: 2, signed: false, idx: -1},
		{addr: 0x10007f1c, size: 2, signed: false, idx: -1},
		{addr: 0x10007f1e, size: 2, signed: false, idx: -1},
		{addr: 0x10007f20, size: 2, signed: false, idx: -1},
		{addr: 0x10007f22, size: 2, signed: false, idx: -1},
		{addr: 0x10007f24, size: 2, signed: false, idx: -1},
		{addr: 0x10007f26, size: 2, signed: false, idx: -1},
		{addr: 0x10007f28, size: 2, signed: false, idx: -1},
		{addr: 0x10007f2a, size: 2, signed: false, idx: -1},
		{addr: 0x10007f2c, size: 2, signed: false, idx: -1},
		{addr: 0x10007f2e, size: 2, signed: false, idx: -1},
		{addr: 0x10007ee8, size: 2, signed: false, idx: -1},
		{addr: 0x10007eea, size: 2, signed: false, idx: -1},
		{addr: 0x20004000, size: 4, signed: false, idx: -1},
		{addr: 0x20004004, size: 4, signed: false, idx: -1},
		{addr: 0x20004008, size: 4, signed: false, idx: -1},
		{addr: 0x2000400c, size: 4, signed: false, idx: -1},
		{addr: 0x20004010, size: 4, signed: false, idx: -1},
		{addr: 0x20004014, size: 1, signed: false, idx: -1},
		{addr: 0x20004018, size: 1, signed: false, idx: -1},
		{addr: 0x2000401c, size: 1, signed: false, idx: -1},
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
		{addr: 0x2000409c, size: 4, signed: false, idx: -1},
		{addr: 0x200040a4, size: 4, signed: false, idx: -1},
		{addr: 0x200040ac, size: 4, signed: false, idx: -1},
		{addr: 0x200040b0, size: 4, signed: false, idx: -1},
		{addr: 0x200040b4, size: 4, signed: false, idx: -1},
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
		{addr: 0x20004ff0, size: 4, signed: false, idx: -1},
		{addr: 0x20004ff4, size: 4, signed: false, idx: -1},
		{addr: 0x20004ff8, size: 4, signed: false, idx: -1},
		{addr: 0x20004ffc, size: 4, signed: false, idx: -1},
	},
})

var mpnIn = []string{"abort", "atk.item", "item0.mode", "item1.mode", "item2.mode", "item3.mode", "atk.f322", "eff.base", "eff.count", "eff.sides", "eff.bonus", "src.type", "src.index", "src.id", "src.fc", "src.backfire", "src.reflection", "src.f1e", "tgt.kind", "tgt.index", "tgt.id", "tgt.group", "g68510c", "g683f94", "g68d810", "g6875ef", "M.group", "M.inCombat", "M.HP", "M.maxSP", "M.SP", "M.exhaustion", "M.shield", "M.shieldDuration", "M.messages", "T.species", "T.body", "T.loc0", "T.loc1", "T.loc2", "T.loc3", "T.loc4", "T.loc5", "T.loc6", "group.named", "vis.a4", "vis.a7", "side0.effect", "side1.effect", "side2.effect", "side3.effect", "side4.effect", "side5.effect", "side6.effect", "side7.effect", "r536190", "r53c3f0", "r542fc0", "r551a20", "actorFlags0", "actorFlags1"}

var mpnOut = []string{"M.HP", "M.SP", "M.fatigue", "M.shield", "M.fbf", "M.fc3", "M.shieldDuration", "M.lastSource0", "M.lastSource1", "M.lastSource2", "M.lastSource3", "M.lastSource4", "M.lastSource5", "M.lastSource6", "M.lastSource7", "M.lastSource8", "M.lastSource9", "M.lastSource10", "M.lastSource11", "M.lastSource12", "C.credit0", "C.credit1", "C.credit2", "C.credit3", "C.credit4", "C.credit5", "C.credit6", "C.credit7", "M1.HP", "alias0", "alias1", "alias2", "alias3", "alias4", "alias5", "alias6", "alias7", "alias8", "alias9", "alias10", "alias11", "line0", "line1", "line2", "line3", "line4", "line5", "line6", "line7", "line8", "line9", "line10", "line11", "line12", "line13", "line14", "line15", "location0", "location1", "s58ac00_a1_0", "s58ac00_a1_1", "s58ac00_a2_0", "s58ac00_a2_1", "s58ac00_calls", "s58b410_a1_0", "s58b410_a2_0", "s58b410_a3_0", "s58b410_a4_0", "s58b410_calls", "s58aad0_a1_0", "s58aad0_a1_1", "s58aad0_a1_2", "s58aad0_a1_3", "s58aad0_a2_0", "s58aad0_a2_1", "s58aad0_a2_2", "s58aad0_a2_3", "s58aad0_calls", "s5182e0_a1_0", "s5182e0_a1_1", "s5182e0_a2_0", "s5182e0_a2_1", "s5182e0_a3_0", "s5182e0_a3_1", "s5182e0_calls", "s590b40_a2_0", "s590b40_a3_0", "s590b40_calls", "s48f650_calls", "s4e4db0_calls", "s4e3af0_a1_0", "s4e3af0_calls", "s4acd80_calls", "s54a0e0_a1_0", "s54a0e0_a2_0", "s54a0e0_a3_0", "s54a0e0_calls", "s536190_calls", "s53c3f0_calls", "s542fc0_a2_0", "s542fc0_a3_0", "s542fc0_calls", "s551a20_calls", "s52a890_a1_0", "s52a890_a2_0", "s52a890_calls", "s543270_calls", "s52beb0_calls", "s523c00_a1_0", "s523c00_a2_0", "s523c00_a3_0", "s523c00_a6_0", "s523c00_calls", "s4e4280_calls", "s4e46f0_a2_0", "s4e46f0_calls", "s4c6c30_a1_0", "s4c6c30_a1_1", "s4c6c30_a1_2", "s4c6c30_calls"}

var mpnStubs = map[uint32]*mcaStub{
	0x4c6c30: {ret: 0, rec: [][4]uint32{{1, 4, 0x20004ff0, 3}}, cnt: 0x20004ffc},
	0x58ac00: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004000, 2}, {2, 4, 0x20004008, 2}}, cnt: 0x20004010},
	0x58b410: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 1, 0x20004014, 1}, {2, 1, 0x20004018, 1}, {3, 1, 0x2000401c, 1}, {4, 4, 0x20004020, 1}}, cnt: 0x20004024},
	0x58aad0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004028, 4}, {2, 4, 0x20004038, 4}}, cnt: 0x20004048},
	0x5182e0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000404c, 2}, {2, 4, 0x20004054, 2}, {3, 4, 0x2000405c, 2}}, cnt: 0x20004064},
	0x590b40: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20004068, 1}, {3, 4, 0x2000406c, 1}}, cnt: 0x20004070},
	0x48f650: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20004074},
	0x4e4db0: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20004078},
	0x4e3af0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000407c, 1}}, cnt: 0x20004080},
	0x4acd80: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20004084},
	0x54a0e0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004088, 1}, {2, 4, 0x2000408c, 1}, {3, 4, 0x20004090, 1}}, cnt: 0x20004094},
	0x536190: {ret: 0x20004098, retN: 0, rec: [][4]uint32{}, cnt: 0x2000409c},
	0x53c3f0: {ret: 0x200040a0, retN: 0, rec: [][4]uint32{}, cnt: 0x200040a4},
	0x542fc0: {ret: 0x200040a8, retN: 0, rec: [][4]uint32{{2, 4, 0x200040ac, 1}, {3, 4, 0x200040b0, 1}}, cnt: 0x200040b4},
	0x551a20: {ret: 0x200040b8, retN: 0, rec: [][4]uint32{}, cnt: 0x200040bc},
	0x52a890: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200040c0, 1}, {2, 4, 0x200040c4, 1}}, cnt: 0x200040c8},
	0x543270: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x200040cc},
	0x52beb0: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x200040d0},
	0x523c00: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200040d4, 1}, {2, 4, 0x200040d8, 1}, {3, 4, 0x200040dc, 1}, {6, 4, 0x200040e0, 1}}, cnt: 0x200040e4},
	0x4e4280: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x200040e8},
	0x4e46f0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200040ec, 1}}, cnt: 0x200040f0},
}

// mpnPark536190 is the first boundary row of re/emu/cases/00536190.json (combat target valid).
var mpnPark536190 = [29]int64{0, 1, 0, 1, 1, 0, 1, 0, 10, 20, 0, 2, 10, 1, 0, 1, 2, 11, 1, 0, 1, 2, 20, 1, 1, 2, 21, 1, 1}

// mpnPark542fc0 is the first boundary row of re/emu/cases/00542fc0.json (defpen).
var mpnPark542fc0 = [53]int64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 2, 256, 1, 0, 0, 0, 0, 123, 0, -83490646, -45, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
