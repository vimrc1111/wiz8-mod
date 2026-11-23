package calc

import (
	"fmt"
	"strings"
)

// spellDamageNode is 0x54e950 spell_damage(effect).
//
// realm = u32 at [0x65be1c] + spell*0x1bf + 0x133 (spell = effect +0), strength
// = effect +0xc8; byte 0x68510c (notice) is saved, then 0x53bea0(effect +0x5c,
// 1) checks the source record (its asserts panic).
// Character targets (+0xf4 count, +0xfc party indices): roll the effect dice
// (0x517970 on +0xb4), add the +0xd4 percentage unless 9999 (0x517f90), reduce
// through 0x552250(&v, &record{1, index}, realm, strength). A reduced 0 prints
// the resisted notice (0x590950, text 0x1b3) when the notice byte is set; a
// non-zero amount goes to 0x52a890(index, v, 0, notice, 0, notice ? 0 :
// effect +0x126, 0).
// Monster targets (+0xe4 count, +0xec ids; 0x4e5550 + 0x4e5620 lookup): roll
// as above, reduce through 0x552250 with record{3, -, id}. A reduced 0 with
// the notice byte prints 0x590b40(monster, text 0x1b3, handle); every reduced
// 0 then calls 0x4c6c30 on the actor (排除 stub; +0x1dc is a free input). A
// non-zero amount runs 0x52bb60(monster, v, effect +0x5c, 0, notice, 0,
// notice ? 0 : effect +0x126, 0) natively (shield, result total/hits, hit
// message, last source, party credit, death event record or HP loss with
// stamina cost 0x52c070 and 0x52beb0).
// Without the saved notice byte 0x54e710(effect) prints the result: with hits
// (+0x126+4), the 0x5905e0 separator (", " 0x619794 after an earlier result,
// else " -- " 0x619788), the summary 0x517a90(text 0x19a, total) for exactly one
// target in both lists, else (text 0x199, hits, total/hits); then each non-zero
// tally k = 0..17 (u32 at +0x126+8+4k) prints "%ld %s" (0x619778) with the
// word text id u16 at 0x61e572+8k (tally 1) or 0x61e574+8k (read from the
// exe); each line goes to 0x5905f0 and 0x5905c0(1, -1). Then the result list
// (+0x126+0x5c count, +0x64 data) drains: type 1 events 0x590950(index, "%s!"
// 0x6175c0, text u16 0x61e602), type 3 events 0x58aad0(9, "%s %s!" 0x614a3c,
// name, same text).
//
// Address stubs (re/emu/cases/0054e950.json): D4 0x517970, 0x552250, 0x52a890,
// 0x52beb0, 0x4e4280, 0x4e46f0, 0x523c00; 排除 0x590b40, 0x58aad0, 0x58b410,
// 0x5905f0, 0x48f650, 0x4acd80, 0x4e4db0, 0x4e3af0, 0x5182e0, 0x4c6c30. Outputs are their
// call counts and every call's recorded arguments. Emu replays the case's dice
// returns and 0x552250 writes; the 0x52a890 stub changes nothing. F rolls
// diceNode, runs resistanceQuantityReductionNode (27-input form; type 1 for
// characters, type 3 with template and monster fields for monsters) and
// characterDamageApplyNode on the parked 0x52a890 world (sdmCdmParked) with
// this call's arguments, the member's +0xb11 and the result list, feeding back
// +0xb11, the result totals and the death record.
// F panics on a repeated monster id once an earlier hit reached a stubbed D4
// monster callee (its writes would be read by the second hit).
//
// Fixture (setup, fixed): UI mode 0x68ec78 = 0 and UI object 0x68edcc = 0 (the
// message-box and portrait bodies are idle), an empty visual collection, status +0xa3 = 0 (rebuild), message texts
// without '%' other than the fixture formats.
// Caps (panic past them): three targets per list, party indices 0..7, monster
// ids 100..101, eight event slots, the 0x1000-byte heap window, message ids
// below 0x3a0.
func spellDamageNode(in []int64, r Rand) []int64 {
	return mcaRun(&sdmSpec, true, sdmBody)(in, r)
}

func sdmEmu(in []int64, r Rand) []int64 { return mcaRun(&sdmSpec, false, sdmBody)(in, r) }

func init() {
	Nodes["0054e950"] = Node{Addr: "0054e950", In: sdmIn, Out: sdmOut, F: spellDamageNode, Emu: sdmEmu}
}

const (
	sdmEffect   = 0x20000000 // the case's argument (object eff)
	sdmMsgTable = 0x20002000 // fixture [0x68c09c]
	sdmMsgText  = 0x2000a000 // fixture: message id i is the one-unit text 0x4e00+i
	sdmMsgFirst = 0x340      // fixture: generic texts for ids 0x340..0x39f
	sdmMsgCount = 0x3a0
	sdmArr      = 0x20003300 // fixture result-list data (eight slots)
	sdmLocal    = 0x7d000000 // 0x54e950's stack value v (outside the arena)
	// 0x590950 passes its stack prefix buffer to 0x58aad0; the address depends
	// on the call depth (emulator stack, initial esp 0x10008000).
	sdmPrefixTarget = 0x10005fb0 // from 0x54e950's character loop
	sdmPrefixDrain  = 0x10005f98 // from 0x54e710's drain loop
)

func sdmMessages(s mstSpec) mstSpec {
	gen := []mstSet{}
	for i := uint32(sdmMsgFirst); i < sdmMsgCount; i++ {
		gen = append(gen, mstSet{sdmMsgTable + 4*i, 4, sdmMsgText + 4*i}, mstSet{sdmMsgText + 4*i, 4, 0x4e00 + i})
	}
	s.set = append(gen, s.set...)
	return s
}

// sdmStubAt is the world's own stub table when it has one (0x540170 reuses
// these helpers with its stubs), else sdmStubs.
func sdmStubAt(w *mcaWorld, a uint32) *mcaStub {
	if w.tab != nil {
		return w.tab[a]
	}
	return sdmStubs[a]
}

func sdmCall(w *mcaWorld, a uint32, args ...uint32) uint32 {
	s := sdmStubAt(w, a)
	k := w.u32(s.cnt)
	for _, r := range s.rec {
		if k >= r[3] {
			panic(fmt.Sprintf("spell damage: stub %#x called more than %d times (fixture record cap)", a, r[3]))
		}
		w.put(r[2]+k*r[1], int(r[1]), uint64(args[r[0]-1]))
	}
	w.put(s.cnt, 4, uint64(k+1))
	return k
}

func sdmReplay(w *mcaWorld, a, k uint32, args ...uint32) uint32 {
	s := sdmStubAt(w, a)
	for _, x := range s.wr {
		at := x.at
		if x.n > 0 {
			if k >= x.n {
				panic(fmt.Sprintf("spell damage: stub %#x write past its %d values", a, x.n))
			}
			at += k * x.size
		}
		dst := x.addr
		if x.arg > 0 {
			dst += args[x.arg-1]
		}
		var v uint64
		for i := uint32(0); i < x.size; i++ {
			v |= uint64(w.u8(at+i)) << (8 * i)
		}
		w.put(dst, int(x.size), v)
	}
	if s.retN > 0 {
		if k >= s.retN {
			panic(fmt.Sprintf("spell damage: stub %#x return past its %d values", a, s.retN))
		}
		return w.u32(s.ret + 4*k)
	}
	return w.u32(s.ret)
}

func sdmStub(w *mcaWorld, a uint32, args ...uint32) uint32 {
	return sdmReplay(w, a, sdmCall(w, a, args...), args...)
}

func sdmMsg(w *mcaWorld, i uint32) uint32 {
	if i >= sdmMsgCount {
		panic("spell damage: message id past the fixture table")
	}
	return w.u32(w.u32(0x68c09c) + 4*i)
}

// sdmFormat is swprintf for the directives the reached texts use: %u, %s, %ld.
func sdmFormat(w *mcaWorld, f uint32, args ...uint32) string {
	var b strings.Builder
	s := []rune(srdWstr(w, f))
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			b.WriteRune(s[i])
			continue
		}
		i++
		if i < len(s) && s[i] == 'l' {
			i++
			if i >= len(s) || s[i] != 'd' {
				panic("spell damage: format directive outside the fixture texts")
			}
			s[i] = 'D'
		}
		if i >= len(s) || len(args) == 0 {
			panic("spell damage: format directive outside the fixture texts")
		}
		switch s[i] {
		case 'u':
			fmt.Fprint(&b, args[0])
		case 'D':
			fmt.Fprint(&b, int32(args[0]))
		case 's':
			b.WriteString(srdWstr(w, args[0]))
		default:
			panic("spell damage: format directive outside the fixture texts")
		}
		args = args[1:]
	}
	return b.String()
}

// sdmNotice is 0x590950(index, fmt, args): "<name> <text>" through 0x58aad0 (a
// leading ' or : drops the space) and the portrait highlight 0x58b410 with the
// name length (+1 for ', +1 more for 's).
func sdmNotice(w *mcaWorld, idx, prefix, f uint32, args ...uint32) {
	t := []rune(sdmFormat(w, f, args...) + "\x00\x00")
	name := w.charBase(idx) + 5
	sdmStub(w, 0x58aad0, 8, 0x64bf5c, name, prefix)
	n := uint32(len([]rune(srdWstr(w, name))))
	if t[0] == 0x27 {
		n++
		if t[1] == 0x73 {
			n++
		}
	}
	sdmStub(w, 0x58b410, uint32(w.u8(w.u32(0x685178)+idx*0x106+0xf1)), 0, n, 0xffffffff)
}

// sdmName is 0x4e5150(m, 0, 0).
func sdmName(w *mcaWorld, m uint32) uint32 {
	t := w.template(m)
	if w.u16(t+0x187) == 0x18d {
		srdPutW(w, 0x6875c3, sdmFormat(w, 0x614b44, w.charBase(w.u32(0x6875ef))+5))
		return 0x6875c3
	}
	if w.u32(m+4) == 0 {
		if !w.act15(w.u32(m + 0xc)) {
			sdmStub(w, 0x5182e0, 1, 0x614b20, w.u32(m))
		}
		return t + 0x60
	}
	if w.u8(w.group(w.u32(m+4))+0x2c) == 0 {
		return t + 0x60
	}
	return t
}

// sdmSource is 0x53bea0(x, flag).
func sdmSource(w *mcaWorld, x uint32, flag bool) bool {
	switch t := w.u32(x); {
	case t == 1:
		if w.u32(x+4) == 0xffffffff {
			panic("spell damage: character source without index (0x53beb1 asserts)")
		}
		return true
	case flag && t == 3 && w.u32(x+4) != 0xffffffff:
		if w.u8(x+0x1c) == 0 && w.u8(x+0x1b) == 0 {
			panic("spell damage: type-3 source without flags (0x53beee asserts)")
		}
		return true
	}
	return false
}

// sdmRoll is 0x517970 on +0xb4 then 0x517f90 (+0xd4 percent) unless 9999.
func sdmRoll(w *mcaWorld) uint32 {
	k := sdmCall(w, 0x517970, sdmEffect+0xb4)
	var v uint32
	if !w.real {
		v = sdmReplay(w, 0x517970, k)
	} else {
		z, n, s := int64(int16(w.u16(sdmEffect+0xb4))), int64(w.u8(sdmEffect+0xb6)), int64(w.u8(sdmEffect+0xb7))
		v = uint32(int32(diceNode([]int64{z, n, s}, w.r)[0]))
	}
	if v != 0x270f {
		v += v * w.u32(sdmEffect+0xd4) / 100
	}
	return v
}

// sdmResist is 0x552250(&v, &record{typ, index, id}, realm, strength); for
// monsters (typ 3) idx is the monster record.
func sdmResist(w *mcaWorld, v, typ, idx uint32) uint32 {
	realm := w.u32(w.u32(0x65be1c) + w.u32(sdmEffect)*0x1bf + 0x133)
	strength := w.u32(sdmEffect + 0xc8)
	k := sdmCall(w, 0x552250, 0, 0, realm, strength)
	if !w.real {
		sdmReplay(w, 0x552250, k, sdmLocal) // the case writes the reduced value through arg 1
		return w.u32(sdmLocal)
	}
	in := make([]int64, 27)
	in[0], in[3], in[4], in[5] = int64(v), int64(strength), int64(typ), int64(realm)
	if typ == 1 {
		c := w.charBase(idx)
		in[1], in[2], in[6] = int64(w.u32(c+0xee1)), int64(w.u32(c+0x89)), int64(idx)
		for j := uint32(1); j < 6; j++ {
			in[6+j] = int64(w.u32(c + 0xee1 + 16*j))
		}
	} else {
		m, t := idx, w.template(idx)
		for j := uint32(0); j < 6; j++ {
			in[12+j] = int64(w.u8(t + 0x176 + j))
			in[19+j] = int64(int8(w.u8(m + 0x217 + j)))
		}
		in[18], in[25], in[26] = int64(w.u8(t+0x24f)), int64(w.u32(m)), int64(w.u32(m+8))
	}
	return uint32(resistanceQuantityReductionNode(in, w.r)[0])
}

// sdmCdmParked is boundary row 639 of re/emu/cases/0052a890.json (mode 8, the
// battle object, absorb pool off, no state slots): F runs the real 0x52a890 on
// that world with this call's arguments, +0xb11 and the result list.
var sdmCdmParked = []int64{5, 5000, 130816, 257, 255, 0, 4294967040, 1, 299, 1073741824, 17, 0, 17, 0, 9999, 100, 0, 0, 0, 1, 8, 536956928, 1, 0, 0, 0, 2477782034, 3608133732, 1732709280, 0, 234, 64, 537035520, 537035264, 1003623434, 2147483648, 1, 14, 100, 537035008, 975908286, 641230208, 281927813, 2746986576, 1064563834, 745747191, 4148921583, 3973579198, 2704914464, 3245710852, 4000465325, 2164794189, 2603488401, 2988491214, 3568233915, 3869371772, 1295787223, 3755115460, 562033917, 2360747443, 1220454672, 2346037578, 1967891237, 728346776, 4282794120, 3838766064, 3308070138, 1109600884, 2013929386, 1555868465, 2029110429, 2752557599, 537036544, 537036288, 0, 0, 4294967295, 54, 255, 0, 55, 0, 128, 0, 1, 1, 1, 1, 255, 0, 116, 4294967295, 1, 0, 1, 255, 0, 255, 1, 1, 255, 0, 1, 255, 0, 124, 1, 0, 128, 0, 0, 97628296, 255, 179, 0, 1, 0, 0, 0, 1, 0, 3, 2, 1, 325, 122, 536870912, 536940544, 0, 0, 0, 0, 0, 537133056, 0, 0, 0, 537137152, 0, 0, 0, 537138176, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, -1, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 536956928, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, 255, 0, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}

var sdmCdmOutX = func() map[string]int {
	_, out := cdmNames()
	m := map[string]int{}
	for i, n := range out {
		m[n] = i
	}
	return m
}()

// sdmCharDamage is the 0x52a890(index, v, 0, notice, a5, kill, 0) call; it
// returns 0x52a890's result.
func sdmCharDamage(w *mcaWorld, idx, v uint32, notice, a5 bool, kill uint32) uint32 {
	k := sdmCall(w, 0x52a890, idx, v, 0, mstB(notice), mstB(a5), kill, 0)
	if !w.real {
		return sdmReplay(w, 0x52a890, k) // the case's return input; writes nothing
	}
	c := w.charBase(idx)
	x := append([]int64(nil), sdmCdmParked...)
	set := func(n string, v int64) { x[cdmX[n]] = v }
	set("who", int64(idx))
	set("dmg", int64(v))
	set("a3", 0)
	set("a4", int64(mstB(notice)))
	set("a5", int64(mstB(a5)))
	set("a7", 0)
	set("hp", int64(w.u32(c+0xb11)))
	set("kill", 0)
	var n int32
	if kill != 0 {
		n = int32(w.u32(kill + 0x5c))
		set("kill", cdmKl)
		for i, f := range []uint32{0, 4, 0x4c, 0x5c, 0x60} {
			set([]string{"kl0", "kl4", "kl4c", "kl5c", "kl60"}[i], int64(w.u32(kill+f)))
		}
		set("kl64", cdmKlData)
		for i := int32(0); i < cdmListMax; i++ {
			e := int64(0)
			if i < n {
				e = int64(w.u32(w.u32(kill+0x64) + 4*uint32(i)))
			}
			set(fmt.Sprintf("kd%d", i), e)
		}
	}
	o := characterDamageApplyNode(x, w.r)
	get := func(n string) uint32 { return uint32(o[sdmCdmOutX[n]]) }
	w.put(c+0xb11, 4, uint64(get("hp")))
	ret := get("ret")
	if kill == 0 {
		return ret
	}
	for i, f := range []uint32{0, 4, 0x4c} {
		w.put(kill+f, 4, uint64(get([]string{"kl0", "kl4", "kl4c"}[i])))
	}
	if int32(get("kl5c")) == n {
		return ret
	}
	e := w.alloc(0x6c)
	for i := uint32(0); i < cdmNewObjW; i++ {
		w.put(e+4*i, 4, uint64(get(fmt.Sprintf("no%d", i))))
	}
	if get("kl64") == cdmNewBuf {
		old, arr := w.u32(kill+0x64), w.alloc(4*uint32(n+1))
		for i := uint32(0); i < uint32(n); i++ {
			w.put(arr+4*i, 4, uint64(w.u32(old+4*i)))
		}
		w.put(kill+0x64, 4, uint64(arr))
		w.put(kill+0x60, 4, uint64(get("kl60")))
	}
	if n >= 8 && w.u32(kill+0x64) == sdmArr {
		panic("spell damage: event list past the fixture's eight slots")
	}
	w.put(w.u32(kill+0x64)+4*uint32(n), 4, uint64(e))
	w.put(kill+0x5c, 4, uint64(n+1))
	return ret
}

// sdmD4Calls counts the stubbed D4 monster callees.
func sdmD4Calls(w *mcaWorld) uint32 {
	n := uint32(0)
	for _, a := range []uint32{0x52beb0, 0x4e4280, 0x4e46f0, 0x523c00} {
		n += w.u32(sdmStubs[a].cnt)
	}
	return n
}

func sdmBody(w *mcaWorld) {
	const eff, res = sdmEffect, sdmEffect + 0x126
	notice := w.u8(0x68510c) != 0
	sdmSource(w, eff+0x5c, true)
	chars, mons := int32(w.u32(eff+0xf4)), int32(w.u32(eff+0xe4))
	if chars > 3 || mons > 3 {
		panic("spell damage: more than three targets in a list (fixture cap)")
	}
	for i := int32(0); i < chars; i++ {
		idx := w.u32(w.u32(eff+0xfc) + 4*uint32(i))
		if idx == 0xffffffff {
			panic("spell damage: character target -1 (0x54e9c8 asserts)")
		}
		v := sdmResist(w, sdmRoll(w), 1, idx)
		if v == 0 {
			if w.u8(0x68510c) != 0 {
				sdmNotice(w, idx, sdmPrefixTarget, sdmMsg(w, 0x1b3))
			}
			continue
		}
		var kill uint32
		if !notice {
			kill = sdmEffect + 0x126
		}
		sdmCharDamage(w, idx, v, notice, false, kill)
	}
	touched := map[uint32]bool{}
	for i := int32(0); i < mons; i++ {
		id := w.u32(w.u32(eff+0xec) + 4*uint32(i))
		if w.real && touched[id] {
			panic("spell damage: repeated monster target after a stubbed D4 callee (its writes would be read)")
		}
		if id < 100 || id > 101 {
			panic("spell damage: monster id outside 100..101 (fixture cap)")
		}
		m := w.findMonster(id)
		v := sdmResist(w, sdmRoll(w), 3, m)
		if v == 0 {
			if w.u8(0x68510c) != 0 {
				h, _ := w.monsterHandle(id)
				sdmStub(w, 0x590b40, m, sdmMsg(w, 0x1b3), h)
			}
			// 0x4c6c30(0) on the actor +0xc: 排除 address stub, return unread.
			sdmStub(w, 0x4c6c30)
			continue
		}
		var r uint32
		if !notice {
			r = res
		}
		before := sdmD4Calls(w)
		sdmHit(w, m, v, eff+0x5c, notice, false, r)
		if sdmD4Calls(w) != before {
			touched[id] = true
		}
	}
	if !notice {
		sdmResult(w, eff)
	}
}

// sdmSep is 0x5905e0 and the separator line (", " after an earlier result,
// else " -- ") through 0x5905f0 / 0x5905c0(1, -1).
func sdmSep(w *mcaWorld, eff uint32) {
	if w.u32(0x68ec78) == 6 || w.u32(0x68ec78) == 7 {
		panic("spell damage: message box in UI mode 6/7 is outside the fixture")
	}
	if w.u8(0x69b7b8) == 0 {
		return
	}
	sep := uint32(0x619788)
	if w.u8(eff+0x124) != 0 {
		sep = 0x619794
	}
	sdmStub(w, 0x5905f0, sep)
}

// sdmResult is 0x54e710(effect).
func sdmResult(w *mcaWorld, eff uint32) {
	res := eff + 0x126
	if w.u8(0x68510c) != 0 {
		return
	}
	if hits := w.u32(res + 4); hits != 0 {
		sdmSep(w, eff)
		text := sdmFormat(w, sdmMsg(w, 0x199), hits, w.u32(res)/hits)
		if w.u32(eff+0xf4)+w.u32(eff+0xe4) == 1 {
			text = sdmFormat(w, sdmMsg(w, 0x19a), w.u32(res))
		}
		srdPutW(w, 0x689fd0, text)
		sdmStub(w, 0x5905f0, 0x689fd0)
		w.put(0x69b7b8, 1, 1)
		w.put(eff+0x124, 1, 1)
		w.put(eff+0x125, 1, 1)
	}
	for k := uint32(0); k < 18; k++ {
		c := w.u32(res + 8 + 4*k)
		if c == 0 {
			continue
		}
		word := w.u16(0x61e574 + 8*k)
		if c == 1 {
			word = w.u16(0x61e572 + 8*k)
		}
		sdmSep(w, eff)
		srdPutW(w, 0x689fd0, sdmFormat(w, 0x619778, c, sdmMsg(w, word)))
		sdmStub(w, 0x5905f0, 0x689fd0)
		w.put(0x69b7b8, 1, 1)
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
		text := sdmMsg(w, w.u16(0x61e602))
		switch w.u32(e) {
		case 1:
			sdmNotice(w, w.u32(e+4), sdmPrefixDrain, 0x6175c0, text)
		case 3:
			sdmStub(w, 0x58aad0, 9, 0x614a3c, e+8, text)
		}
	}
}

// sdmHit is 0x52bb60(m, dmg, src, 0, notice, plain, res, 0); it returns the
// damage past the shield (0 when the monster is already down or the shield
// takes it all).
func sdmHit(w *mcaWorld, m, dmg, src uint32, notice, plain bool, res uint32) uint32 {
	if w.u32(m+0x2b) == 0 {
		return 0
	}
	if w.u32(m+0xc7) != 0 {
		msg, shield := sdmMsg(w, 0x193), w.u16(m+0xc5) // a4 = 0 selects text 0x193
		if shield >= dmg {
			sdmStub(w, 0x590b40, m, msg, dmg)
			w.put(m+0xc5, 2, uint64(shield-dmg))
			if res != 0 {
				w.put(res+4, 4, uint64(w.u32(res+4)+1))
			}
			return 0
		}
		sdmStub(w, 0x590b40, m, msg, shield)
		dmg -= shield
		// 0x524390(id, 2): clear buff slot 2, 0x4acd80, rebuild.
		w.put(m+0xbf, 4, 0)
		w.put(m+0xc3, 4, 0)
		w.put(m+0xc7, 4, 0)
		sdmStub(w, 0x4acd80, w.u32(m+0xc), 0x12, 0)
		w.rebuild(w.u32(m))
	}
	if res != 0 {
		w.put(res, 4, uint64(w.u32(res)+dmg))
		w.put(res+4, 4, uint64(w.u32(res+4)+1))
	}
	if dmg == 0 { // 0x52bc11: straight to the epilogue
		return 0
	}
	if w.u8(0x683f94) != 0 || w.u8(m+0x2ab) != 0 {
		sdmStub(w, 0x48f650, m, 0, 1)
		who := uint32(9)
		if sdmSource(w, src, false) && w.u32(src+4) != 0xffffffff {
			who = 8
		}
		if notice && plain { // 0x52bcac: text 0x254 (0x68c09c+0x950)
			sdmStub(w, 0x58aad0, who, sdmMsg(w, 0x254), dmg)
		} else if notice {
			sdmStub(w, 0x58aad0, who, sdmMsg(w, 0x256), sdmName(w, m), dmg, 0x689b34)
		}
	}
	if w.u8(src+0x1c) == 0 && w.u8(src+0x1b) == 0 && w.u8(src+0x1e) == 0 {
		for i := uint32(0); i < 13; i++ {
			w.put(m+0x304+4*i, 4, uint64(w.u32(src+4*i)))
		}
		if w.u8(m+0x15) != 0 && sdmSource(w, src, false) && w.u32(src+4) != 0xffffffff {
			idx := w.u32(src + 4)
			if idx >= 8 {
				panic("spell damage: crediting party index past 7")
			}
			a := w.u32(m+0x10) + 0x1a + 4*idx
			w.put(a, 4, uint64(w.u32(a)+dmg))
		}
	}
	if dmg >= w.u32(m+0x2b) {
		w.put(m+0x2b, 4, 0)
		if res != 0 {
			e := w.alloc(0x6c)
			w.put(e, 4, 3)
			srdPutW(w, e+8, srdWstr(w, sdmName(w, m)))
			n := w.u32(res+0x5c) + 1
			if int32(n) > int32(w.u32(res+0x60)) {
				old, arr := w.u32(res+0x64), w.alloc(4*n)
				w.put(res+0x64, 4, uint64(arr))
				w.put(res+0x60, 4, uint64(n))
				for i := int32(0); i < int32(w.u32(res+0x5c)); i++ {
					w.put(arr+4*uint32(i), 4, uint64(w.u32(old+4*uint32(i))))
				}
			}
			c := w.u32(res + 0x5c)
			if c >= 8 && w.u32(res+0x64) == sdmArr {
				panic("spell damage: event list past the fixture's eight slots")
			}
			w.put(w.u32(res+0x64)+4*c, 4, uint64(e))
			w.put(res+0x5c, 4, uint64(c+1))
		}
		sdmDowned(w, m, mstB(res == 0))
	} else {
		w.put(m+0x2b, 4, uint64(w.u32(m+0x2b)-dmg))
		sdmFatigue(w, m, 2*dmg/3, res)
		sdmStub(w, 0x52beb0, m, src, 0)
	}
	// 0x56a2a0 is idle with UI object 0x68edcc = 0; 0x4e5a50 -> 0x4c6a50
	// allocates its 0x14-byte collection, finds the fixture's empty one and
	// returns 0; 0x4c6c30 is an 排除 stub (its return is unread).
	if w.u32(0x68edcc) != 0 {
		panic("spell damage: UI object outside the fixture")
	}
	if a := w.u32(m + 0xc); a != 0 {
		w.alloc(0x14)
		sdmStub(w, 0x4c6c30)
	}
	return dmg
}

// sdmDowned is 0x4e4690(m, flag).
func sdmDowned(w *mcaWorld, m, flag uint32) {
	if w.act15(w.u32(m + 0xc)) {
		return
	}
	sdmStub(w, 0x4e4db0, m, 0x15, 1)
	sdmStub(w, 0x4e4280, m)
	sdmStub(w, 0x4e46f0, m, flag)
	h, ok := w.monsterHandle(w.u32(m))
	if !ok {
		panic("spell damage: monster id not found (0x4e5550 asserts)")
	}
	sdmStub(w, 0x4e3af0, h, 0)
}

// sdmFatigue is 0x52c070(m, cost, res).
func sdmFatigue(w *mcaWorld, m, cost, res uint32) {
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
		panic("spell damage: max SP 0 divides by zero (0x52c0b8)")
	}
	w.put(m+0x242, 4, uint64(staminaFatigueLevel(sp, most)))
	if sp == 0 && w.u32(m+0x9b) < 0x270f {
		sdmStub(w, 0x523c00, w.u32(m), 0x11, 0x270f, 0, 0, mstB(res == 0))
		if res != 0 {
			w.put(res+0x4c, 4, uint64(w.u32(res+0x4c)+1))
		}
	}
}

// Code below is generated from the same description as re/emu/cases/0054e950.json.

var sdmSpec = sdmMessages(mstSpec{
	set: []mstSet{
		{0x65be1c, 4, 0x20001000},
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
		{0x689fd0, 4, 0xdddddddd},
		{0x689fd4, 4, 0xdddddddd},
		{0x689fd8, 4, 0xdddddddd},
		{0x689fdc, 4, 0xdddddddd},
		{0x689fe0, 4, 0xdddddddd},
		{0x689fe4, 4, 0xdddddddd},
		{0x689fe8, 4, 0xdddddddd},
		{0x689fec, 4, 0xdddddddd},
		{0x689ff0, 4, 0xdddddddd},
		{0x689ff4, 4, 0xdddddddd},
		{0x689ff8, 4, 0xdddddddd},
		{0x689ffc, 4, 0xdddddddd},
		{0x200000ec, 4, 0x20000220},
		{0x200000fc, 4, 0x20000200},
		{0x2000005c, 4, 0x10101010},
		{0x20000060, 4, 0x11111111},
		{0x20000064, 4, 0x12121212},
		{0x20000068, 4, 0x13131313},
		{0x2000006c, 4, 0x14141414},
		{0x20000070, 4, 0x15151515},
		{0x20000074, 4, 0x16161616},
		{0x20000078, 4, 0x17171717},
		{0x2000007c, 4, 0x18181818},
		{0x20000080, 4, 0x19191919},
		{0x20000084, 4, 0x1a1a1a1a},
		{0x20000088, 4, 0x1b1b1b1b},
		{0x2000008c, 4, 0x1c1c1c1c},
		{0x2000018a, 4, 0x20003300},
		{0x20003300, 4, 0x20003100},
		{0x20003304, 4, 0x20003180},
		{0x20003308, 4, 0x20003200},
		{0x20003108, 2, 0x45},
		{0x2000310a, 2, 0x76},
		{0x2000310c, 2, 0x30},
		{0x2000310e, 2, 0x0},
		{0x20003188, 2, 0x45},
		{0x2000318a, 2, 0x76},
		{0x2000318c, 2, 0x31},
		{0x2000318e, 2, 0x78},
		{0x20003190, 2, 0x0},
		{0x20003208, 2, 0x45},
		{0x2000320a, 2, 0x76},
		{0x2000320c, 2, 0x32},
		{0x2000320e, 2, 0x78},
		{0x20003210, 2, 0x78},
		{0x20003212, 2, 0x0},
		{0x20001133, 4, 0x40},
		{0x200012f2, 4, 0x41},
		{0x200014b1, 4, 0x42},
		{0x20001670, 4, 0x43},
		{0x20002648, 4, 0x20003060},
		{0x2000264c, 4, 0x20003064},
		{0x20002668, 4, 0x20003010},
		{0x20002664, 4, 0x20003020},
		{0x200026cc, 4, 0x20003000},
		{0x20002958, 4, 0x20003068},
		{0x20002e44, 4, 0x20003040},
		{0x20003010, 2, 0x25},
		{0x20003012, 2, 0x75},
		{0x20003014, 2, 0x0},
		{0x20003020, 2, 0x25},
		{0x20003022, 2, 0x75},
		{0x20003024, 2, 0x2f},
		{0x20003026, 2, 0x25},
		{0x20003028, 2, 0x75},
		{0x2000302a, 2, 0x0},
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
		{0x20003044, 2, 0x0},
	},
	in: []mstFld{
		{0x68510c, 1, false, -1, 0, 0},
		{0x683f94, 1, false, -1, 0, 0},
		{0x69b7b8, 1, false, -1, 0, 0},
		{0x6875ef, 4, false, -1, 0, 0},
		{0x20000000, 4, false, -1, 0, 0},
		{0x20001133, 4, false, 4, 4, 0x1bf},
		{0x200000b4, 2, true, -1, 0, 0},
		{0x200000b6, 1, false, -1, 0, 0},
		{0x200000b7, 1, false, -1, 0, 0},
		{0x200000c8, 4, false, -1, 0, 0},
		{0x200000d4, 4, false, -1, 0, 0},
		{0x200000f4, 4, true, -1, 0, 0},
		{0x20000200, 4, true, -1, 0, 0},
		{0x20000204, 4, true, -1, 0, 0},
		{0x20000208, 4, true, -1, 0, 0},
		{0x200000e4, 4, true, -1, 0, 0},
		{0x20000220, 4, false, -1, 0, 0},
		{0x20000224, 4, false, -1, 0, 0},
		{0x20000228, 4, false, -1, 0, 0},
		{0x20000124, 1, false, -1, 0, 0},
		{0x20000125, 1, false, -1, 0, 0},
		{0x2000005c, 4, false, -1, 0, 0},
		{0x20000060, 4, true, -1, 0, 0},
		{0x20000077, 1, false, -1, 0, 0},
		{0x20000078, 1, false, -1, 0, 0},
		{0x2000007a, 1, false, -1, 0, 0},
		{0x20000126, 4, false, -1, 0, 0},
		{0x2000012a, 4, false, -1, 0, 0},
		{0x2000012e, 4, false, -1, 0, 0},
		{0x20000132, 4, false, -1, 0, 0},
		{0x20000136, 4, false, -1, 0, 0},
		{0x2000013a, 4, false, -1, 0, 0},
		{0x2000013e, 4, false, -1, 0, 0},
		{0x20000142, 4, false, -1, 0, 0},
		{0x20000146, 4, false, -1, 0, 0},
		{0x2000014a, 4, false, -1, 0, 0},
		{0x2000014e, 4, false, -1, 0, 0},
		{0x20000152, 4, false, -1, 0, 0},
		{0x20000156, 4, false, -1, 0, 0},
		{0x2000015a, 4, false, -1, 0, 0},
		{0x2000015e, 4, false, -1, 0, 0},
		{0x20000162, 4, false, -1, 0, 0},
		{0x20000166, 4, false, -1, 0, 0},
		{0x2000016a, 4, false, -1, 0, 0},
		{0x2000016e, 4, false, -1, 0, 0},
		{0x20000172, 4, false, -1, 0, 0},
		{0x20000182, 4, true, -1, 0, 0},
		{0x20000186, 4, true, -1, 0, 0},
		{0x20003100, 4, false, -1, 0, 0},
		{0x20003104, 4, false, -1, 0, 0},
		{0x20003180, 4, false, -1, 0, 0},
		{0x20003184, 4, false, -1, 0, 0},
		{0x20003200, 4, false, -1, 0, 0},
		{0x20003204, 4, false, -1, 0, 0},
		{0x20003000, 2, false, -1, 0, 0},
		{0x20003002, 2, false, -1, 0, 0},
		{0x20003040, 2, false, -1, 0, 0},
		{0x20003042, 2, false, -1, 0, 0},
		{0x2000b000, 4, false, -1, 0, 0},
		{0x2000b004, 4, false, -1, 0, 0},
		{0x2000b008, 4, false, -1, 0, 0},
		{0x2000b00c, 4, false, -1, 0, 0},
		{0x2000b010, 4, false, -1, 0, 0},
		{0x2000b014, 4, false, -1, 0, 0},
		{0x2000b018, 4, false, -1, 0, 0},
		{0x2000b01c, 4, false, -1, 0, 0},
		{0x2000b020, 4, false, -1, 0, 0},
		{0x2000b024, 4, false, -1, 0, 0},
		{0x2000b028, 4, false, -1, 0, 0},
		{0x2000b02c, 4, false, -1, 0, 0},
		{0x2000b030, 4, false, -1, 0, 0},
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
		{0x20010b11, 4, false, -1, 0, 0},
		{0x20012373, 4, false, -1, 0, 0},
		{0x20013bd5, 4, false, -1, 0, 0},
		{0x20015437, 4, false, -1, 0, 0},
		{0x20016c99, 4, false, -1, 0, 0},
		{0x200184fb, 4, false, -1, 0, 0},
		{0x20019d5d, 4, false, -1, 0, 0},
		{0x2001b5bf, 4, false, -1, 0, 0},
	},
	out: []mstFld{
		{0x2000b040, 4, false, -1, 0, 0},
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
		{0x2000b368, 4, false, -1, 0, 0},
		{0x2000b36c, 4, false, -1, 0, 0},
		{0x2000b370, 4, false, -1, 0, 0},
		{0x2000b374, 4, false, -1, 0, 0},
		{0x2000b378, 4, false, -1, 0, 0},
		{0x2000b37c, 4, false, -1, 0, 0},
		{0x2000b380, 4, false, -1, 0, 0},
		{0x2000b384, 4, false, -1, 0, 0},
		{0x2000b388, 4, false, -1, 0, 0},
		{0x2000b38c, 4, false, -1, 0, 0},
		{0x2000b390, 4, false, -1, 0, 0},
		{0x2000b394, 4, false, -1, 0, 0},
		{0x2000b398, 4, false, -1, 0, 0},
		{0x2000b39c, 4, false, -1, 0, 0},
		{0x2000b3a0, 4, false, -1, 0, 0},
		{0x2000b3a4, 4, false, -1, 0, 0},
		{0x2000b3a8, 4, false, -1, 0, 0},
		{0x2000b3ac, 4, false, -1, 0, 0},
		{0x2000b3b0, 4, false, -1, 0, 0},
		{0x2000b3b4, 4, false, -1, 0, 0},
		{0x2000b3b8, 4, false, -1, 0, 0},
		{0x2000b3bc, 4, false, -1, 0, 0},
		{0x2000b3c0, 4, false, -1, 0, 0},
		{0x2000b3c4, 4, false, -1, 0, 0},
		{0x2000b3c8, 4, false, -1, 0, 0},
		{0x2000b3cc, 4, false, -1, 0, 0},
		{0x2000b3d0, 4, false, -1, 0, 0},
		{0x2000b3d4, 4, false, -1, 0, 0},
		{0x2000b3d8, 4, false, -1, 0, 0},
		{0x2000b3dc, 4, false, -1, 0, 0},
		{0x2000b3e0, 4, false, -1, 0, 0},
		{0x2000b3e4, 4, false, -1, 0, 0},
		{0x2000b3e8, 4, false, -1, 0, 0},
		{0x2000b3ec, 4, false, -1, 0, 0},
		{0x2000b3f0, 4, false, -1, 0, 0},
		{0x2000b3f4, 4, false, -1, 0, 0},
		{0x2000b3f8, 4, false, -1, 0, 0},
		{0x2000b3fc, 4, false, -1, 0, 0},
		{0x2000b400, 4, false, -1, 0, 0},
		{0x2000b404, 4, false, -1, 0, 0},
		{0x2000b408, 4, false, -1, 0, 0},
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
		{0x20000126, 4, false, -1, 0, 0},
		{0x2000012a, 4, false, -1, 0, 0},
		{0x2000012e, 4, false, -1, 0, 0},
		{0x20000132, 4, false, -1, 0, 0},
		{0x20000136, 4, false, -1, 0, 0},
		{0x2000013a, 4, false, -1, 0, 0},
		{0x2000013e, 4, false, -1, 0, 0},
		{0x20000142, 4, false, -1, 0, 0},
		{0x20000146, 4, false, -1, 0, 0},
		{0x2000014a, 4, false, -1, 0, 0},
		{0x2000014e, 4, false, -1, 0, 0},
		{0x20000152, 4, false, -1, 0, 0},
		{0x20000156, 4, false, -1, 0, 0},
		{0x2000015a, 4, false, -1, 0, 0},
		{0x2000015e, 4, false, -1, 0, 0},
		{0x20000162, 4, false, -1, 0, 0},
		{0x20000166, 4, false, -1, 0, 0},
		{0x2000016a, 4, false, -1, 0, 0},
		{0x2000016e, 4, false, -1, 0, 0},
		{0x20000172, 4, false, -1, 0, 0},
		{0x20000182, 4, false, -1, 0, 0},
		{0x20000186, 4, false, -1, 0, 0},
		{0x2000018a, 4, false, -1, 0, 0},
		{0x20003300, 4, false, -1, 0, 0},
		{0x20003304, 4, false, -1, 0, 0},
		{0x20003308, 4, false, -1, 0, 0},
		{0x2000330c, 4, false, -1, 0, 0},
		{0x20003310, 4, false, -1, 0, 0},
		{0x20003314, 4, false, -1, 0, 0},
		{0x20003318, 4, false, -1, 0, 0},
		{0x2000331c, 4, false, -1, 0, 0},
		{0x20000124, 1, false, -1, 0, 0},
		{0x20000125, 1, false, -1, 0, 0},
		{0x69b7b8, 1, false, -1, 0, 0},
		{0x689fd0, 4, false, -1, 0, 0},
		{0x689fd4, 4, false, -1, 0, 0},
		{0x689fd8, 4, false, -1, 0, 0},
		{0x689fdc, 4, false, -1, 0, 0},
		{0x689fe0, 4, false, -1, 0, 0},
		{0x689fe4, 4, false, -1, 0, 0},
		{0x689fe8, 4, false, -1, 0, 0},
		{0x689fec, 4, false, -1, 0, 0},
		{0x689ff0, 4, false, -1, 0, 0},
		{0x689ff4, 4, false, -1, 0, 0},
		{0x689ff8, 4, false, -1, 0, 0},
		{0x689ffc, 4, false, -1, 0, 0},
		{0x6875c3, 4, false, -1, 0, 0},
		{0x6875c7, 4, false, -1, 0, 0},
		{0x6875cb, 4, false, -1, 0, 0},
		{0x6875cf, 4, false, -1, 0, 0},
		{0x6875d3, 4, false, -1, 0, 0},
		{0x6875d7, 4, false, -1, 0, 0},
		{0x6875db, 4, false, -1, 0, 0},
		{0x6875df, 4, false, -1, 0, 0},
		{0x20010b11, 4, false, -1, 0, 0},
		{0x20012373, 4, false, -1, 0, 0},
		{0x20013bd5, 4, false, -1, 0, 0},
		{0x20015437, 4, false, -1, 0, 0},
		{0x20016c99, 4, false, -1, 0, 0},
		{0x200184fb, 4, false, -1, 0, 0},
		{0x20019d5d, 4, false, -1, 0, 0},
		{0x2001b5bf, 4, false, -1, 0, 0},
	},
})

var sdmIn = []string{"notice", "messageAll", "messageOpen", "aliasCharacter", "spell", "realm", "Z", "N", "S", "strength", "bonusPercent", "characterCount", "characterTarget0", "characterTarget1", "characterTarget2", "monsterCount", "monsterTarget0", "monsterTarget1", "monsterTarget2", "resultShown", "resultFlag", "sourceType", "sourceCharacter", "sourceFlag1b", "sourceFlag1c", "sourceFlag1e", "resultTotal", "resultHits", "tally0", "tally1", "tally2", "tally3", "tally4", "tally5", "tally6", "tally7", "tally8", "tally9", "tally10", "tally11", "tally12", "tally13", "tally14", "tally15", "tally16", "tally17", "eventCount", "eventCapacity", "event0Type", "event0Character", "event1Type", "event1Character", "event2Type", "event2Character", "resistedText0", "resistedText1", "eventText0", "eventText1", "diceReturn0", "diceReturn1", "diceReturn2", "diceReturn3", "diceReturn4", "diceReturn5", "resistedValue0", "resistedValue1", "resistedValue2", "resistedValue3", "resistedValue4", "resistedValue5", "ignoredReturn", "hp0", "maxHp0", "shield0", "shieldDuration0", "maxStamina0", "stamina0", "exhaustion0", "credits0", "messages0", "group0", "actorFlags0", "actorState0", "templateKind0", "resistanceModifier0Fire", "resistanceModifier0Water", "resistanceModifier0Air", "resistanceModifier0Earth", "resistanceModifier0Mental", "resistanceModifier0Divine", "templateResistance0Fire", "templateResistance0Water", "templateResistance0Air", "templateResistance0Earth", "templateResistance0Mental", "templateResistance0Divine", "templateLevel0", "hp1", "maxHp1", "shield1", "shieldDuration1", "maxStamina1", "stamina1", "exhaustion1", "credits1", "messages1", "group1", "actorFlags1", "actorState1", "templateKind1", "resistanceModifier1Fire", "resistanceModifier1Water", "resistanceModifier1Air", "resistanceModifier1Earth", "resistanceModifier1Mental", "resistanceModifier1Divine", "templateResistance1Fire", "templateResistance1Water", "templateResistance1Air", "templateResistance1Earth", "templateResistance1Mental", "templateResistance1Divine", "templateLevel1", "groupNamed", "character0Resistance0", "character0Resistance1", "character0Resistance2", "character0Resistance3", "character0Resistance4", "character0Resistance5", "character0Level", "character1Resistance0", "character1Resistance1", "character1Resistance2", "character1Resistance3", "character1Resistance4", "character1Resistance5", "character1Level", "character2Resistance0", "character2Resistance1", "character2Resistance2", "character2Resistance3", "character2Resistance4", "character2Resistance5", "character2Level", "character3Resistance0", "character3Resistance1", "character3Resistance2", "character3Resistance3", "character3Resistance4", "character3Resistance5", "character3Level", "character4Resistance0", "character4Resistance1", "character4Resistance2", "character4Resistance3", "character4Resistance4", "character4Resistance5", "character4Level", "character5Resistance0", "character5Resistance1", "character5Resistance2", "character5Resistance3", "character5Resistance4", "character5Resistance5", "character5Level", "character6Resistance0", "character6Resistance1", "character6Resistance2", "character6Resistance3", "character6Resistance4", "character6Resistance5", "character6Level", "character7Resistance0", "character7Resistance1", "character7Resistance2", "character7Resistance3", "character7Resistance4", "character7Resistance5", "character7Level", "character0Hp", "character1Hp", "character2Hp", "character3Hp", "character4Hp", "character5Hp", "character6Hp", "character7Hp"}

var sdmOut = []string{"diceCalls", "resistCalls", "resistArg3_0", "resistArg3_1", "resistArg3_2", "resistArg3_3", "resistArg3_4", "resistArg3_5", "resistArg4_0", "resistArg4_1", "resistArg4_2", "resistArg4_3", "resistArg4_4", "resistArg4_5", "charDamageCalls", "charDamageArg1_0", "charDamageArg1_1", "charDamageArg1_2", "charDamageArg2_0", "charDamageArg2_1", "charDamageArg2_2", "charDamageArg4_0", "charDamageArg4_1", "charDamageArg4_2", "charDamageArg6_0", "charDamageArg6_1", "charDamageArg6_2", "hitResponseCalls", "hitResponseArg1_0", "hitResponseArg1_1", "hitResponseArg1_2", "hitResponseArg3_0", "hitResponseArg3_1", "hitResponseArg3_2", "removeStateCalls", "removeStateArg1_0", "removeStateArg1_1", "removeStateArg1_2", "killRewardsCalls", "killRewardsArg1_0", "killRewardsArg1_1", "killRewardsArg1_2", "killRewardsArg2_0", "killRewardsArg2_1", "killRewardsArg2_2", "conditionApplyCalls", "conditionApplyArg1_0", "conditionApplyArg1_1", "conditionApplyArg1_2", "conditionApplyArg2_0", "conditionApplyArg2_1", "conditionApplyArg2_2", "conditionApplyArg3_0", "conditionApplyArg3_1", "conditionApplyArg3_2", "conditionApplyArg4_0", "conditionApplyArg4_1", "conditionApplyArg4_2", "conditionApplyArg6_0", "conditionApplyArg6_1", "conditionApplyArg6_2", "monsterMessageCalls", "monsterMessageArg1_0", "monsterMessageArg1_1", "monsterMessageArg1_2", "monsterMessageArg1_3", "monsterMessageArg1_4", "monsterMessageArg1_5", "monsterMessageArg2_0", "monsterMessageArg2_1", "monsterMessageArg2_2", "monsterMessageArg2_3", "monsterMessageArg2_4", "monsterMessageArg2_5", "monsterMessageArg3_0", "monsterMessageArg3_1", "monsterMessageArg3_2", "monsterMessageArg3_3", "monsterMessageArg3_4", "monsterMessageArg3_5", "messageCalls", "messageArg1_0", "messageArg1_1", "messageArg1_2", "messageArg1_3", "messageArg1_4", "messageArg1_5", "messageArg1_6", "messageArg1_7", "messageArg1_8", "messageArg1_9", "messageArg1_10", "messageArg1_11", "messageArg1_12", "messageArg1_13", "messageArg1_14", "messageArg1_15", "messageArg1_16", "messageArg1_17", "messageArg1_18", "messageArg1_19", "messageArg2_0", "messageArg2_1", "messageArg2_2", "messageArg2_3", "messageArg2_4", "messageArg2_5", "messageArg2_6", "messageArg2_7", "messageArg2_8", "messageArg2_9", "messageArg2_10", "messageArg2_11", "messageArg2_12", "messageArg2_13", "messageArg2_14", "messageArg2_15", "messageArg2_16", "messageArg2_17", "messageArg2_18", "messageArg2_19", "messageArg3_0", "messageArg3_1", "messageArg3_2", "messageArg3_3", "messageArg3_4", "messageArg3_5", "messageArg3_6", "messageArg3_7", "messageArg3_8", "messageArg3_9", "messageArg3_10", "messageArg3_11", "messageArg3_12", "messageArg3_13", "messageArg3_14", "messageArg3_15", "messageArg3_16", "messageArg3_17", "messageArg3_18", "messageArg3_19", "messageArg4_0", "messageArg4_1", "messageArg4_2", "messageArg4_3", "messageArg4_4", "messageArg4_5", "messageArg4_6", "messageArg4_7", "messageArg4_8", "messageArg4_9", "messageArg4_10", "messageArg4_11", "messageArg4_12", "messageArg4_13", "messageArg4_14", "messageArg4_15", "messageArg4_16", "messageArg4_17", "messageArg4_18", "messageArg4_19", "portraitCalls", "portraitArg1_0", "portraitArg1_1", "portraitArg1_2", "portraitArg1_3", "portraitArg1_4", "portraitArg1_5", "portraitArg1_6", "portraitArg1_7", "portraitArg1_8", "portraitArg1_9", "portraitArg1_10", "portraitArg1_11", "portraitArg1_12", "portraitArg1_13", "portraitArg1_14", "portraitArg1_15", "portraitArg1_16", "portraitArg1_17", "portraitArg1_18", "portraitArg1_19", "portraitArg3_0", "portraitArg3_1", "portraitArg3_2", "portraitArg3_3", "portraitArg3_4", "portraitArg3_5", "portraitArg3_6", "portraitArg3_7", "portraitArg3_8", "portraitArg3_9", "portraitArg3_10", "portraitArg3_11", "portraitArg3_12", "portraitArg3_13", "portraitArg3_14", "portraitArg3_15", "portraitArg3_16", "portraitArg3_17", "portraitArg3_18", "portraitArg3_19", "messageBoxCalls", "messageBoxArg1_0", "messageBoxArg1_1", "messageBoxArg1_2", "messageBoxArg1_3", "messageBoxArg1_4", "messageBoxArg1_5", "messageBoxArg1_6", "messageBoxArg1_7", "messageBoxArg1_8", "messageBoxArg1_9", "messageBoxArg1_10", "messageBoxArg1_11", "messageBoxArg1_12", "messageBoxArg1_13", "messageBoxArg1_14", "messageBoxArg1_15", "messageBoxArg1_16", "messageBoxArg1_17", "messageBoxArg1_18", "messageBoxArg1_19", "messageBoxArg1_20", "messageBoxArg1_21", "messageBoxArg1_22", "messageBoxArg1_23", "messageBoxArg1_24", "messageBoxArg1_25", "messageBoxArg1_26", "messageBoxArg1_27", "messageBoxArg1_28", "messageBoxArg1_29", "messageBoxArg1_30", "messageBoxArg1_31", "messageBoxArg1_32", "messageBoxArg1_33", "messageBoxArg1_34", "messageBoxArg1_35", "messageBoxArg1_36", "messageBoxArg1_37", "messageBoxArg1_38", "messageBoxArg1_39", "hitSoundCalls", "hitSoundArg1_0", "hitSoundArg1_1", "hitSoundArg1_2", "shieldVisualCalls", "shieldVisualArg1_0", "shieldVisualArg1_1", "shieldVisualArg1_2", "shieldVisualArg2_0", "shieldVisualArg2_1", "shieldVisualArg2_2", "deathAnimationCalls", "deathAnimationArg1_0", "deathAnimationArg1_1", "deathAnimationArg1_2", "deathAnimationArg2_0", "deathAnimationArg2_1", "deathAnimationArg2_2", "deathNotifyCalls", "deathNotifyArg1_0", "deathNotifyArg1_1", "deathNotifyArg1_2", "nameLogCalls", "nameLogArg1_0", "nameLogArg1_1", "nameLogArg1_2", "nameLogArg1_3", "nameLogArg1_4", "nameLogArg1_5", "nameLogArg3_0", "nameLogArg3_1", "nameLogArg3_2", "nameLogArg3_3", "nameLogArg3_4", "nameLogArg3_5", "zeroDamageVisualCalls", "hp0", "shield0", "shieldDuration0", "shieldPower0", "shieldWord0", "stamina0", "fatigue0", "rebuild470", "rebuild4f0", "rebuild2470", "rebuild24b0", "lastSource0_0", "lastSource0_1", "lastSource0_2", "lastSource0_3", "lastSource0_4", "lastSource0_5", "lastSource0_6", "lastSource0_7", "lastSource0_8", "lastSource0_9", "lastSource0_10", "lastSource0_11", "lastSource0_12", "modifiers0_0", "modifiers0_1", "modifiers0_2", "modifiers0_3", "modifiers0_4", "modifiers0_5", "modifiers0_6", "modifiers0_7", "modifiers0_8", "modifiers0_9", "modifiers0_10", "modifiers0_11", "modifiers0_12", "modifiers0_13", "modifiers0_14", "modifiers0_15", "modifiers0_16", "modifiers0_17", "modifiers0_18", "modifiers0_19", "modifiers0_20", "modifiers0_21", "modifiers0_22", "modifiers0_23", "modifiers0_24", "modifiers0_25", "hp1", "shield1", "shieldDuration1", "shieldPower1", "shieldWord1", "stamina1", "fatigue1", "rebuild471", "rebuild4f1", "rebuild2471", "rebuild24b1", "lastSource1_0", "lastSource1_1", "lastSource1_2", "lastSource1_3", "lastSource1_4", "lastSource1_5", "lastSource1_6", "lastSource1_7", "lastSource1_8", "lastSource1_9", "lastSource1_10", "lastSource1_11", "lastSource1_12", "modifiers1_0", "modifiers1_1", "modifiers1_2", "modifiers1_3", "modifiers1_4", "modifiers1_5", "modifiers1_6", "modifiers1_7", "modifiers1_8", "modifiers1_9", "modifiers1_10", "modifiers1_11", "modifiers1_12", "modifiers1_13", "modifiers1_14", "modifiers1_15", "modifiers1_16", "modifiers1_17", "modifiers1_18", "modifiers1_19", "modifiers1_20", "modifiers1_21", "modifiers1_22", "modifiers1_23", "modifiers1_24", "modifiers1_25", "partyKills0", "partyKills1", "partyKills2", "partyKills3", "partyKills4", "partyKills5", "partyKills6", "partyKills7", "resultTotal", "resultHits", "tally0", "tally1", "tally2", "tally3", "tally4", "tally5", "tally6", "tally7", "tally8", "tally9", "tally10", "tally11", "tally12", "tally13", "tally14", "tally15", "tally16", "tally17", "eventCount", "eventCapacity", "eventArray", "eventSlot0", "eventSlot1", "eventSlot2", "eventSlot3", "eventSlot4", "eventSlot5", "eventSlot6", "eventSlot7", "resultShown", "resultFlag", "messageOpen", "summaryText0", "summaryText1", "summaryText2", "summaryText3", "summaryText4", "summaryText5", "summaryText6", "summaryText7", "summaryText8", "summaryText9", "summaryText10", "summaryText11", "aliasText0", "aliasText1", "aliasText2", "aliasText3", "aliasText4", "aliasText5", "aliasText6", "aliasText7", "character0Hp", "character1Hp", "character2Hp", "character3Hp", "character4Hp", "character5Hp", "character6Hp", "character7Hp"}

var sdmStubs = map[uint32]*mcaStub{
	0x517970: {ret: 0x2000b000, retN: 6, rec: [][4]uint32{}, cnt: 0x2000b040, wr: []mcaWrite{}},
	0x552250: {ret: 0x2000b030, retN: 0, rec: [][4]uint32{{3, 4, 0x2000b048, 6}, {4, 4, 0x2000b060, 6}}, cnt: 0x2000b044, wr: []mcaWrite{{1, 0x0, 4, 0x2000b018, 6}}},
	0x52a890: {ret: 0x2000b030, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b07c, 3}, {2, 4, 0x2000b088, 3}, {4, 1, 0x2000b094, 3}, {6, 4, 0x2000b0a0, 3}}, cnt: 0x2000b078, wr: []mcaWrite{}},
	0x52beb0: {ret: 0x2000b030, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b0b0, 3}, {3, 4, 0x2000b0bc, 3}}, cnt: 0x2000b0ac, wr: []mcaWrite{}},
	0x4e4280: {ret: 0x2000b030, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b0cc, 3}}, cnt: 0x2000b0c8, wr: []mcaWrite{}},
	0x4e46f0: {ret: 0x2000b030, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b0dc, 3}, {2, 1, 0x2000b0e8, 3}}, cnt: 0x2000b0d8, wr: []mcaWrite{}},
	0x523c00: {ret: 0x2000b030, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b0f8, 3}, {2, 4, 0x2000b104, 3}, {3, 4, 0x2000b110, 3}, {4, 4, 0x2000b11c, 3}, {6, 1, 0x2000b128, 3}}, cnt: 0x2000b0f4, wr: []mcaWrite{}},
	0x590b40: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b138, 6}, {2, 4, 0x2000b150, 6}, {3, 4, 0x2000b168, 6}}, cnt: 0x2000b134, wr: []mcaWrite{}},
	0x58aad0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b184, 20}, {2, 4, 0x2000b1d4, 20}, {3, 4, 0x2000b224, 20}, {4, 4, 0x2000b274, 20}}, cnt: 0x2000b180, wr: []mcaWrite{}},
	0x58b410: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b2c8, 20}, {3, 4, 0x2000b318, 20}}, cnt: 0x2000b2c4, wr: []mcaWrite{}},
	0x5905f0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b36c, 40}}, cnt: 0x2000b368, wr: []mcaWrite{}},
	0x48f650: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b410, 3}}, cnt: 0x2000b40c, wr: []mcaWrite{}},
	0x4acd80: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b420, 3}, {2, 4, 0x2000b42c, 3}}, cnt: 0x2000b41c, wr: []mcaWrite{}},
	0x4e4db0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b43c, 3}, {2, 4, 0x2000b448, 3}}, cnt: 0x2000b438, wr: []mcaWrite{}},
	0x4e3af0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b458, 3}}, cnt: 0x2000b454, wr: []mcaWrite{}},
	0x5182e0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000b468, 6}, {3, 4, 0x2000b480, 6}}, cnt: 0x2000b464, wr: []mcaWrite{}},
	0x4c6c30: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x2000b498, wr: []mcaWrite{}},
}
