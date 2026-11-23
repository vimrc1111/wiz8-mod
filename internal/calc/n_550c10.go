package calc

import (
	"fmt"
	"strings"
)

// spellResourceDrainNode is 0x550c10 spell_resource_drain(effect).
//
// Character targets (effect +0xf4 count, +0xfc party indices): roll the effect
// dice (0x517970 on +0xb4), reduce by the spell realm's resistance
// (0x552250(&v, &record{1, index}, realm, strength +0xc8); realm is the u32 at
// [0x65be1c] + spell*0x1bf + 0x133, spell = effect +0). A reduced 0 prints the
// resisted notice (0x590950 with text 0x6cc) when byte 0x68510c is set; a
// non-zero amount goes to 0x52b7e0(index, amount, notice) and is counted.
// 0x53bea0(effect +0x5c, 1) then checks the source record (its asserts panic).
// Monster targets (+0xe4 count, +0xec ids): roll, add the +0xd4 percentage
// unless 9999 (0x517f90), reduce through 0x552250 with the character loop's
// last record (0x550c10's stack record; with no character target the
// emulator's uninitialised stack reads 0, so type 0 index 0), notice as above,
// and a non-zero amount runs 0x52bb60(monster, amount, effect +0x5c, 0,
// notice, 0, notice ? 0 : effect +0x126, 0) natively: damage shield +0xc5
// (duration +0xc7; overflow clears buff slot 2 through 0x524390 and rebuilds
// through 0x50e8c0), result total/hits, hit message, last-source copy +0x304,
// party credit, death (event record on the result list, 0x4e4690) or HP loss
// with stamina cost 0x52c070 and the 0x52beb0 response, then the presentation
// tail 0x56a2a0 / 0x4e5a50 / 0x4c6c30. Without the notice byte and with any
// hit: 0x5905e0/0x5905f0/0x5905c0 message-box calls, the summary
// 0x517a90(text 0x698, total) for exactly one target in both counts, else
// 0x517a90(text 0x69c, hits, total/hits), flags +0x124/+0x125; then the result
// list +0x126+0x5c drains, each event announced (type 1: 0x590950, type 3:
// 0x58aad0(9, ...)) and freed.
//
// Address stubs (re/emu/cases/00550c10.json): D4 0x517970, 0x552250,
// 0x52b7e0, 0x52beb0, 0x4e4280, 0x4e46f0, 0x523c00; 排除 0x590b40, 0x58aad0,
// 0x58b410, 0x48f650, 0x4acd80, 0x4e4db0, 0x4e3af0, 0x5182e0, 0x4c6c30. Outputs are their
// call counts and every call's recorded arguments. Emu replays the case's dice
// returns and 0x552250 writes; F rolls diceNode and runs
// resistanceQuantityReductionNode (27-input form) on the character records.
// F panics on a repeated monster id: the stubbed D4 callees' monster writes
// would be read by the second hit.
//
// Fixture (setup, fixed): UI mode 0x68ec78 = 0 and UI object 0x68edcc = 0 (the
// message-box and portrait-refresh bodies are idle), an empty visual collection
// (0x4c6a50 allocates 0x14 bytes and returns 0, so 0x4c6990 is not reached),
// status +0xa3 = 0 (rebuild),
// message texts without '%' other than the fixture formats.
// Actor +0x1dc is a full u32 input; presentation calls record post-shield damage and count.
// Caps (panic past them): three targets per list, party indices 0..7, monster
// ids 100..101, eight event slots, the 0x1000-byte heap window.
func spellResourceDrainNode(in []int64, r Rand) []int64 {
	return mcaRun(&srdSpec, true, srdBody)(in, r)
}

func srdEmu(in []int64, r Rand) []int64 { return mcaRun(&srdSpec, false, srdBody)(in, r) }

func init() {
	Nodes["00550c10"] = Node{Addr: "00550c10", In: srdIn, Out: srdOut, F: spellResourceDrainNode, Emu: srdEmu}
}

const (
	srdEffect = 0x20000000 // the case's argument (object eff)
	srdLocal  = 0x7d000000 // 0x550c10's stack value v (outside the arena)
	// 0x590950 passes its stack prefix buffer to 0x58aad0; the address depends
	// on the call depth (emulator stack, initial esp 0x10008000).
	srdPrefixTarget = 0x10005fa8 // from 0x550c10's target loops
	srdPrefixDrain  = 0x10005fa4 // from 0x550c10's drain loop
)

func srdCall(w *mcaWorld, a uint32, args ...uint32) uint32 {
	s := srdStubs[a]
	if w.tab != nil { // a caller with its own case table (0x53e250)
		s = w.tab[a]
	}
	k := w.u32(s.cnt)
	for _, r := range s.rec {
		if k >= r[3] {
			panic(fmt.Sprintf("spell resource drain: stub %#x called more than %d times (fixture record cap)", a, r[3]))
		}
		w.put(r[2]+k*r[1], int(r[1]), uint64(args[r[0]-1]))
	}
	w.put(s.cnt, 4, uint64(k+1))
	return k
}

func srdReplay(w *mcaWorld, a, k uint32, args ...uint32) uint32 {
	s := srdStubs[a]
	if w.tab != nil { // a caller with its own case table (0x53e250)
		s = w.tab[a]
	}
	for _, x := range s.wr {
		at := x.at
		if x.n > 0 {
			if k >= x.n {
				panic(fmt.Sprintf("spell resource drain: stub %#x write past its %d values", a, x.n))
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
			panic(fmt.Sprintf("spell resource drain: stub %#x return past its %d values", a, s.retN))
		}
		return w.u32(s.ret + 4*k)
	}
	return w.u32(s.ret)
}

func srdStub(w *mcaWorld, a uint32, args ...uint32) uint32 {
	return srdReplay(w, a, srdCall(w, a, args...), args...)
}

func srdMsg(w *mcaWorld, i uint32) uint32 { return w.u32(w.u32(0x68c09c) + 4*i) }

func srdWstr(w *mcaWorld, a uint32) string {
	var b strings.Builder
	for ; w.u16(a) != 0; a += 2 {
		b.WriteRune(rune(w.u16(a)))
	}
	return b.String()
}

func srdPutW(w *mcaWorld, a uint32, s string) {
	for _, c := range s {
		w.put(a, 2, uint64(c))
		a += 2
	}
	w.put(a, 2, 0)
}

// srdFormat is swprintf for the directives the reached texts use: %u, %s.
func srdFormat(w *mcaWorld, f uint32, args ...uint32) string {
	var b strings.Builder
	s := []rune(srdWstr(w, f))
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			b.WriteRune(s[i])
			continue
		}
		i++
		if i >= len(s) || len(args) == 0 {
			panic("spell resource drain: format directive outside the fixture texts")
		}
		switch s[i] {
		case 'u':
			fmt.Fprint(&b, args[0])
		case 's':
			b.WriteString(srdWstr(w, args[0]))
		default:
			panic("spell resource drain: format directive outside the fixture texts")
		}
		args = args[1:]
	}
	return b.String()
}

// srdNotice is 0x590950(index, fmt, args): "<name> <text>" through 0x58aad0 (a
// leading ' or : drops the space) and the portrait highlight 0x58b410 with the
// name length (+1 for ', +1 more for 's).
func srdNotice(w *mcaWorld, idx, prefix, f uint32, args ...uint32) {
	t := []rune(srdFormat(w, f, args...) + "\x00\x00")
	name := w.charBase(idx) + 5
	srdStub(w, 0x58aad0, 8, 0x64bf5c, name, prefix)
	n := uint32(len([]rune(srdWstr(w, name))))
	if t[0] == 0x27 {
		n++
		if t[1] == 0x73 {
			n++
		}
	}
	srdStub(w, 0x58b410, uint32(w.u8(w.u32(0x685178)+idx*0x106+0xf1)), 0, n, 0xffffffff)
}

// srdName is 0x4e5150(m, 0, 0).
func srdName(w *mcaWorld, m uint32) uint32 {
	t := w.template(m)
	if w.u16(t+0x187) == 0x18d {
		srdPutW(w, 0x6875c3, srdFormat(w, 0x614b44, w.charBase(w.u32(0x6875ef))+5))
		return 0x6875c3
	}
	if w.u32(m+4) == 0 {
		if !w.act15(w.u32(m + 0xc)) {
			srdStub(w, 0x5182e0, 1, 0x614b20, w.u32(m))
		}
		return t + 0x60
	}
	if w.u8(w.group(w.u32(m+4))+0x2c) == 0 {
		return t + 0x60
	}
	return t
}

// srdSource is 0x53bea0(x, flag).
func srdSource(w *mcaWorld, x uint32, flag bool) bool {
	switch t := w.u32(x); {
	case t == 1:
		if w.u32(x+4) == 0xffffffff {
			panic("spell resource drain: character source without index (0x53beb1 asserts)")
		}
		return true
	case flag && t == 3 && w.u32(x+4) != 0xffffffff:
		if w.u8(x+0x1c) == 0 && w.u8(x+0x1b) == 0 {
			panic("spell resource drain: type-3 source without flags (0x53beee asserts)")
		}
		return true
	}
	return false
}

func srdDice(w *mcaWorld) uint32 {
	k := srdCall(w, 0x517970, srdEffect+0xb4)
	if !w.real {
		return srdReplay(w, 0x517970, k)
	}
	z, n, s := int64(int16(w.u16(srdEffect+0xb4))), int64(w.u8(srdEffect+0xb6)), int64(w.u8(srdEffect+0xb7))
	return uint32(int32(diceNode([]int64{z, n, s}, w.r)[0]))
}

// srdResist is 0x552250(&v, &record, realm, strength).
func srdResist(w *mcaWorld, v, recType, recIdx uint32) uint32 {
	realm := w.u32(w.u32(0x65be1c) + w.u32(srdEffect)*0x1bf + 0x133)
	strength := w.u32(srdEffect + 0xc8)
	k := srdCall(w, 0x552250, 0, 0, realm, strength)
	if !w.real {
		srdReplay(w, 0x552250, k, srdLocal) // the case writes the reduced value through arg 1
		return w.u32(srdLocal)
	}
	if recType == 3 {
		panic("spell resource drain: monster record in the character resistance path")
	}
	c := w.charBase(recIdx)
	in := make([]int64, 27)
	in[0], in[1], in[2], in[3], in[4] = int64(v), int64(w.u32(c+0xee1)), int64(w.u32(c+0x89)), int64(strength), int64(recType)
	in[5], in[6] = int64(realm), int64(recIdx)
	for j := uint32(1); j < 6; j++ {
		in[6+j] = int64(w.u32(c + 0xee1 + 16*j))
	}
	return uint32(resistanceQuantityReductionNode(in, w.r)[0])
}

func srdBody(w *mcaWorld) {
	const eff, res = srdEffect, srdEffect + 0x126
	notice := w.u8(0x68510c)
	var sum, hits, recType, recIdx uint32
	chars, mons := int32(w.u32(eff+0xf4)), int32(w.u32(eff+0xe4))
	if chars > 3 || mons > 3 {
		panic("spell resource drain: more than three targets in a list (fixture cap)")
	}
	for i := int32(0); i < chars; i++ {
		idx := w.u32(w.u32(eff+0xfc) + 4*uint32(i))
		if idx == 0xffffffff {
			panic("spell resource drain: character target -1 (0x550c6a asserts)")
		}
		recType, recIdx = 1, idx
		v := srdResist(w, srdDice(w), recType, recIdx)
		if v == 0 {
			if notice != 0 {
				srdNotice(w, recIdx, srdPrefixTarget, srdMsg(w, 0x1b3))
			}
			continue
		}
		srdStub(w, 0x52b7e0, idx, v, uint32(notice))
		sum += v
		hits++
	}
	srdSource(w, eff+0x5c, true)
	seen := map[uint32]bool{}
	for i := int32(0); i < mons; i++ {
		id := w.u32(w.u32(eff+0xec) + 4*uint32(i))
		if w.real && seen[id] {
			panic("spell resource drain: repeated monster target (stubbed D4 writes would be read)")
		}
		seen[id] = true
		if id < 100 || id > 101 {
			panic("spell resource drain: monster id outside 100..101 (fixture cap)")
		}
		m := w.findMonster(id)
		v := srdDice(w)
		if v != 0x270f {
			v += v * w.u32(eff+0xd4) / 100 // 0x517f90
		}
		v = srdResist(w, v, recType, recIdx)
		if v == 0 {
			if notice != 0 {
				if recType == 3 {
					panic("spell resource drain: monster record in the notice path (0x550e8e)")
				}
				srdNotice(w, recIdx, srdPrefixTarget, srdMsg(w, 0x1b3))
			}
			continue
		}
		var r uint32
		if notice == 0 {
			r = res
		}
		srdHit(w, m, v, eff+0x5c, notice != 0, r)
		sum += v
		hits++
	}
	if notice != 0 {
		return
	}
	if hits != 0 {
		// 0x5905e0 / 0x5905f0 / 0x5905c0: the message box (idle outside UI modes 6/7).
		if w.u32(0x68ec78) == 6 || w.u32(0x68ec78) == 7 {
			panic("spell resource drain: message box in UI mode 6/7 is outside the fixture")
		}
		text := srdFormat(w, srdMsg(w, 0x1a7), hits, sum/hits)
		if w.u32(eff+0xe4)+w.u32(eff+0xf4) == 1 {
			text = srdFormat(w, srdMsg(w, 0x1a6), sum)
		}
		srdPutW(w, 0x689fd0, text)
		w.put(0x69b7b8, 1, 1)
		w.put(eff+0x124, 1, 1)
		w.put(eff+0x125, 1, 1)
	}
	srdDrain(w, srdPrefixDrain)
}

// srdDrain empties the result list (count +0x126+0x5c, data +0x64), announcing
// each event (type 1: 0x590950 through the prefix buffer, type 3: 0x58aad0(9,
// ...)) before it is freed. 0x550c10 and 0x54e710 share this tail.
func srdDrain(w *mcaWorld, prefix uint32) {
	const res = srdEffect + 0x126
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
		text := srdMsg(w, w.u16(0x61e602))
		switch w.u32(e) {
		case 1:
			srdNotice(w, w.u32(e+4), prefix, 0x6175c0, text)
		case 3:
			srdStub(w, 0x58aad0, 9, 0x614a3c, e+8, text)
		}
	}
}

// srdHit is 0x52bb60(m, dmg, src, 0, notice, 0, res, 0).
func srdHit(w *mcaWorld, m, dmg, src uint32, notice bool, res uint32) {
	if w.u32(m+0x2b) == 0 {
		return
	}
	if w.u32(m+0xc7) != 0 {
		msg, shield := srdMsg(w, 0x193), w.u16(m+0xc5) // a4 = 0 selects text 0x193
		if shield >= dmg {
			srdStub(w, 0x590b40, m, msg, dmg)
			w.put(m+0xc5, 2, uint64(shield-dmg))
			if res != 0 {
				w.put(res+4, 4, uint64(w.u32(res+4)+1))
			}
			return
		}
		srdStub(w, 0x590b40, m, msg, shield)
		dmg -= shield
		// 0x524390(id, 2): clear buff slot 2, 0x4acd80, rebuild.
		w.put(m+0xbf, 4, 0)
		w.put(m+0xc3, 4, 0)
		w.put(m+0xc7, 4, 0)
		srdStub(w, 0x4acd80, w.u32(m+0xc), 0x12, 0)
		w.rebuild(w.u32(m))
	}
	if res != 0 {
		w.put(res, 4, uint64(w.u32(res)+dmg))
		w.put(res+4, 4, uint64(w.u32(res+4)+1))
	}
	if dmg == 0 {
		panic("spell resource drain: zero damage reaches 0x52bb60's tail (0x550c10 passes only non-zero)")
	}
	if w.u8(0x683f94) != 0 || w.u8(m+0x2ab) != 0 {
		srdStub(w, 0x48f650, m, 0, 1)
		who := uint32(9)
		if srdSource(w, src, false) && w.u32(src+4) != 0xffffffff {
			who = 8
		}
		if notice {
			srdStub(w, 0x58aad0, who, srdMsg(w, 0x256), srdName(w, m), dmg, 0x689b34)
		}
	}
	if w.u8(src+0x1c) == 0 && w.u8(src+0x1b) == 0 && w.u8(src+0x1e) == 0 {
		for i := uint32(0); i < 13; i++ {
			w.put(m+0x304+4*i, 4, uint64(w.u32(src+4*i)))
		}
		if w.u8(m+0x15) != 0 && srdSource(w, src, false) && w.u32(src+4) != 0xffffffff {
			idx := w.u32(src + 4)
			if idx >= 8 {
				panic("spell resource drain: crediting party index past 7")
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
			srdPutW(w, e+8, srdWstr(w, srdName(w, m)))
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
			if c >= 8 && w.u32(res+0x64) == srdEffect+0x3300 {
				panic("spell resource drain: event list past the fixture's eight slots")
			}
			w.put(w.u32(res+0x64)+4*c, 4, uint64(e))
			w.put(res+0x5c, 4, uint64(c+1))
		}
		srdDowned(w, m, mstB(res == 0))
	} else {
		w.put(m+0x2b, 4, uint64(w.u32(m+0x2b)-dmg))
		srdFatigue(w, m, 2*dmg/3, res)
		srdStub(w, 0x52beb0, m, src, 0)
	}
	// 0x56a2a0 is idle with UI object 0x68edcc = 0; 0x4e5a50 -> 0x4c6a50
	// allocates its 0x14-byte collection, finds the fixture's empty one and
	// returns 0; 0x4c6c30 is an excluded presentation stub, with the post-shield damage argument.
	if w.u32(0x68edcc) != 0 {
		panic("spell resource drain: UI object outside the fixture")
	}
	if a := w.u32(m + 0xc); a != 0 {
		w.alloc(0x14)
		if w.tab == nil || w.tab[0x4c6c30] != nil {
			srdStub(w, 0x4c6c30, dmg)
		} else {
			// The separate 0x53e250 fixture still runs this presentation call natively.
			if w.u32(a+0x1dc)&0x400 == 0 {
				panic("spell resource drain: actor flag outside the native presentation fixture")
			}
		}
	}
}

// srdDowned is 0x4e4690(m, flag).
func srdDowned(w *mcaWorld, m, flag uint32) {
	if w.act15(w.u32(m + 0xc)) {
		return
	}
	srdStub(w, 0x4e4db0, m, 0x15, 1)
	srdStub(w, 0x4e4280, m)
	srdStub(w, 0x4e46f0, m, flag)
	h, ok := w.monsterHandle(w.u32(m))
	if !ok {
		panic("spell resource drain: monster id not found (0x4e5550 asserts)")
	}
	srdStub(w, 0x4e3af0, h, 0)
}

// srdFatigue is 0x52c070(m, cost, res).
func srdFatigue(w *mcaWorld, m, cost, res uint32) {
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
		panic("spell resource drain: max SP 0 divides by zero (0x52c0b8)")
	}
	w.put(m+0x242, 4, uint64(staminaFatigueLevel(sp, most)))
	if sp == 0 && w.u32(m+0x9b) < 0x270f {
		srdStub(w, 0x523c00, w.u32(m), 0x11, 0x270f, 0, 0, mstB(res == 0))
		if res != 0 {
			w.put(res+0x4c, 4, uint64(w.u32(res+0x4c)+1))
		}
	}
}

// Code below is generated from the same description as re/emu/cases/00550c10.json.
var srdSpec = mstSpec{
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
		{0x20002698, 4, 0x20003010},
		{0x2000269c, 4, 0x20003020},
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
		{0x20004000, 4, false, -1, 0, 0},
		{0x20004004, 4, false, -1, 0, 0},
		{0x20004008, 4, false, -1, 0, 0},
		{0x2000400c, 4, false, -1, 0, 0},
		{0x20004010, 4, false, -1, 0, 0},
		{0x20004014, 4, false, -1, 0, 0},
		{0x20004018, 4, false, -1, 0, 0},
		{0x2000401c, 4, false, -1, 0, 0},
		{0x20004020, 4, false, -1, 0, 0},
		{0x20004024, 4, false, -1, 0, 0},
		{0x20004028, 4, false, -1, 0, 0},
		{0x2000402c, 4, false, -1, 0, 0},
		{0x20004030, 4, false, -1, 0, 0},
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
		{0x200070a4, 1, false, -1, 0, 0},
		{0x20005187, 2, false, -1, 0, 0},
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
		{0x200078a4, 1, false, -1, 0, 0},
		{0x20005587, 2, false, -1, 0, 0},
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
		{0x200069dc, 4, false, -1, 0, 0},
		{0x20006bdc, 4, false, -1, 0, 0},
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
		{0x20004094, 1, false, -1, 0, 0},
		{0x20004095, 1, false, -1, 0, 0},
		{0x20004096, 1, false, -1, 0, 0},
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
		{0x200040dc, 1, false, -1, 0, 0},
		{0x200040dd, 1, false, -1, 0, 0},
		{0x200040de, 1, false, -1, 0, 0},
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
		{0x2000411c, 1, false, -1, 0, 0},
		{0x2000411d, 1, false, -1, 0, 0},
		{0x2000411e, 1, false, -1, 0, 0},
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
		{0x2000417c, 4, false, -1, 0, 0},
		{0x20004180, 4, false, -1, 0, 0},
		{0x20004184, 4, false, -1, 0, 0},
		{0x20004188, 4, false, -1, 0, 0},
		{0x2000418c, 4, false, -1, 0, 0},
		{0x20004190, 4, false, -1, 0, 0},
		{0x20004194, 4, false, -1, 0, 0},
		{0x20004198, 4, false, -1, 0, 0},
		{0x2000419c, 4, false, -1, 0, 0},
		{0x200041a0, 4, false, -1, 0, 0},
		{0x200041a4, 4, false, -1, 0, 0},
		{0x200041a8, 4, false, -1, 0, 0},
		{0x200041ac, 4, false, -1, 0, 0},
		{0x200041b0, 4, false, -1, 0, 0},
		{0x200041b4, 4, false, -1, 0, 0},
		{0x200041b8, 4, false, -1, 0, 0},
		{0x200041bc, 4, false, -1, 0, 0},
		{0x200041c0, 4, false, -1, 0, 0},
		{0x200041c4, 4, false, -1, 0, 0},
		{0x200041c8, 4, false, -1, 0, 0},
		{0x200041cc, 4, false, -1, 0, 0},
		{0x200041d0, 4, false, -1, 0, 0},
		{0x200041d4, 4, false, -1, 0, 0},
		{0x200041d8, 4, false, -1, 0, 0},
		{0x200041dc, 4, false, -1, 0, 0},
		{0x200041e0, 4, false, -1, 0, 0},
		{0x200041e4, 4, false, -1, 0, 0},
		{0x200041e8, 4, false, -1, 0, 0},
		{0x200041ec, 4, false, -1, 0, 0},
		{0x200041f0, 4, false, -1, 0, 0},
		{0x200041f4, 4, false, -1, 0, 0},
		{0x200041f8, 4, false, -1, 0, 0},
		{0x200041fc, 4, false, -1, 0, 0},
		{0x20004200, 4, false, -1, 0, 0},
		{0x20004204, 4, false, -1, 0, 0},
		{0x20004208, 4, false, -1, 0, 0},
		{0x2000420c, 4, false, -1, 0, 0},
		{0x20004210, 4, false, -1, 0, 0},
		{0x20004214, 4, false, -1, 0, 0},
		{0x20004218, 4, false, -1, 0, 0},
		{0x2000421c, 4, false, -1, 0, 0},
		{0x20004220, 4, false, -1, 0, 0},
		{0x20004224, 4, false, -1, 0, 0},
		{0x20004228, 4, false, -1, 0, 0},
		{0x2000422c, 4, false, -1, 0, 0},
		{0x20004230, 4, false, -1, 0, 0},
		{0x20004234, 4, false, -1, 0, 0},
		{0x20004238, 4, false, -1, 0, 0},
		{0x2000423c, 4, false, -1, 0, 0},
		{0x20004240, 4, false, -1, 0, 0},
		{0x20004244, 4, false, -1, 0, 0},
		{0x20004248, 4, false, -1, 0, 0},
		{0x2000424c, 4, false, -1, 0, 0},
		{0x20004250, 4, false, -1, 0, 0},
		{0x20004254, 4, false, -1, 0, 0},
		{0x20004258, 4, false, -1, 0, 0},
		{0x2000425c, 4, false, -1, 0, 0},
		{0x20004260, 4, false, -1, 0, 0},
		{0x20004264, 4, false, -1, 0, 0},
		{0x20004268, 4, false, -1, 0, 0},
		{0x2000426c, 4, false, -1, 0, 0},
		{0x20004270, 4, false, -1, 0, 0},
		{0x20004274, 4, false, -1, 0, 0},
		{0x20004278, 4, false, -1, 0, 0},
		{0x2000427c, 4, false, -1, 0, 0},
		{0x20004280, 4, false, -1, 0, 0},
		{0x20004284, 4, false, -1, 0, 0},
		{0x20004288, 4, false, -1, 0, 0},
		{0x2000428c, 4, false, -1, 0, 0},
		{0x20004290, 4, false, -1, 0, 0},
		{0x20004294, 4, false, -1, 0, 0},
		{0x20004298, 4, false, -1, 0, 0},
		{0x2000429c, 4, false, -1, 0, 0},
		{0x200042a0, 4, false, -1, 0, 0},
		{0x200042a4, 4, false, -1, 0, 0},
		{0x200042a8, 4, false, -1, 0, 0},
		{0x200042ac, 4, false, -1, 0, 0},
		{0x200042b0, 4, false, -1, 0, 0},
		{0x200042b4, 4, false, -1, 0, 0},
		{0x200042b8, 4, false, -1, 0, 0},
		{0x200042bc, 4, false, -1, 0, 0},
		{0x200042c0, 4, false, -1, 0, 0},
		{0x200042c4, 4, false, -1, 0, 0},
		{0x200042c8, 4, false, -1, 0, 0},
		{0x200042cc, 4, false, -1, 0, 0},
		{0x200042d0, 4, false, -1, 0, 0},
		{0x200042d4, 4, false, -1, 0, 0},
		{0x200042d8, 4, false, -1, 0, 0},
		{0x200042dc, 4, false, -1, 0, 0},
		{0x200042e0, 4, false, -1, 0, 0},
		{0x200042e4, 4, false, -1, 0, 0},
		{0x200042e8, 4, false, -1, 0, 0},
		{0x200042ec, 4, false, -1, 0, 0},
		{0x200042f0, 4, false, -1, 0, 0},
		{0x200042f4, 4, false, -1, 0, 0},
		{0x200042f8, 4, false, -1, 0, 0},
		{0x200042fc, 4, false, -1, 0, 0},
		{0x20004300, 4, false, -1, 0, 0},
		{0x20004304, 4, false, -1, 0, 0},
		{0x20004308, 4, false, -1, 0, 0},
		{0x2000430c, 4, false, -1, 0, 0},
		{0x20004310, 4, false, -1, 0, 0},
		{0x20004314, 4, false, -1, 0, 0},
		{0x20004318, 4, false, -1, 0, 0},
		{0x2000431c, 4, false, -1, 0, 0},
		{0x20004320, 4, false, -1, 0, 0},
		{0x20004324, 4, false, -1, 0, 0},
		{0x20004328, 4, false, -1, 0, 0},
		{0x2000432c, 4, false, -1, 0, 0},
		{0x20004330, 4, false, -1, 0, 0},
		{0x20004334, 4, false, -1, 0, 0},
		{0x20004338, 4, false, -1, 0, 0},
		{0x2000433c, 4, false, -1, 0, 0},
		{0x20004340, 4, false, -1, 0, 0},
		{0x20004344, 4, false, -1, 0, 0},
		{0x20004348, 4, false, -1, 0, 0},
		{0x2000434c, 4, false, -1, 0, 0},
		{0x20004350, 4, false, -1, 0, 0},
		{0x20004354, 4, false, -1, 0, 0},
		{0x20004358, 4, false, -1, 0, 0},
		{0x2000435c, 4, false, -1, 0, 0},
		{0x20004360, 4, false, -1, 0, 0},
		{0x20004364, 4, false, -1, 0, 0},
		{0x20004368, 4, false, -1, 0, 0},
		{0x2000436c, 4, false, -1, 0, 0},
		{0x20004370, 4, false, -1, 0, 0},
		{0x20004374, 4, false, -1, 0, 0},
		{0x20004378, 4, false, -1, 0, 0},
		{0x2000437c, 4, false, -1, 0, 0},
		{0x20004380, 4, false, -1, 0, 0},
		{0x20004384, 4, false, -1, 0, 0},
		{0x20004388, 4, false, -1, 0, 0},
		{0x2000438c, 4, false, -1, 0, 0},
		{0x20004390, 4, false, -1, 0, 0},
		{0x20004394, 4, false, -1, 0, 0},
		{0x20004398, 4, false, -1, 0, 0},
		{0x2000439c, 4, false, -1, 0, 0},
		{0x200043a0, 4, false, -1, 0, 0},
		{0x200043a4, 4, false, -1, 0, 0},
		{0x200043a8, 4, false, -1, 0, 0},
		{0x200043ac, 4, false, -1, 0, 0},
		{0x200043b0, 4, false, -1, 0, 0},
		{0x200043b4, 4, false, -1, 0, 0},
		{0x200043b8, 4, false, -1, 0, 0},
		{0x200043bc, 4, false, -1, 0, 0},
		{0x200043c0, 4, false, -1, 0, 0},
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
		{0x20004ff0, 4, false, -1, 0, 0},
		{0x20004ff4, 4, false, -1, 0, 0},
		{0x20004ff8, 4, false, -1, 0, 0},
		{0x20004ffc, 4, false, -1, 0, 0},
	},
}

var srdIn = []string{"notice", "messageAll", "messageOpen", "aliasCharacter", "spell", "realm", "Z", "N", "S", "strength", "bonusPercent", "characterCount", "characterTarget0", "characterTarget1", "characterTarget2", "monsterCount", "monsterTarget0", "monsterTarget1", "monsterTarget2", "resultShown", "resultFlag", "sourceType", "sourceCharacter", "sourceFlag1b", "sourceFlag1c", "sourceFlag1e", "resultTotal", "resultHits", "resultExhausted", "eventCount", "eventCapacity", "event0Type", "event0Character", "event1Type", "event1Character", "event2Type", "event2Character", "resistedText0", "resistedText1", "eventText0", "eventText1", "diceReturn0", "diceReturn1", "diceReturn2", "diceReturn3", "diceReturn4", "diceReturn5", "resistedValue0", "resistedValue1", "resistedValue2", "resistedValue3", "resistedValue4", "resistedValue5", "ignoredReturn", "hp0", "maxHp0", "shield0", "shieldDuration0", "maxStamina0", "stamina0", "exhaustion0", "credits0", "messages0", "group0", "actorState0", "templateKind0", "hp1", "maxHp1", "shield1", "shieldDuration1", "maxStamina1", "stamina1", "exhaustion1", "credits1", "messages1", "group1", "actorState1", "templateKind1", "groupNamed", "character0Resistance0", "character0Resistance1", "character0Resistance2", "character0Resistance3", "character0Resistance4", "character0Resistance5", "character0Level", "character1Resistance0", "character1Resistance1", "character1Resistance2", "character1Resistance3", "character1Resistance4", "character1Resistance5", "character1Level", "character2Resistance0", "character2Resistance1", "character2Resistance2", "character2Resistance3", "character2Resistance4", "character2Resistance5", "character2Level", "character3Resistance0", "character3Resistance1", "character3Resistance2", "character3Resistance3", "character3Resistance4", "character3Resistance5", "character3Level", "character4Resistance0", "character4Resistance1", "character4Resistance2", "character4Resistance3", "character4Resistance4", "character4Resistance5", "character4Level", "character5Resistance0", "character5Resistance1", "character5Resistance2", "character5Resistance3", "character5Resistance4", "character5Resistance5", "character5Level", "character6Resistance0", "character6Resistance1", "character6Resistance2", "character6Resistance3", "character6Resistance4", "character6Resistance5", "character6Level", "character7Resistance0", "character7Resistance1", "character7Resistance2", "character7Resistance3", "character7Resistance4", "character7Resistance5", "character7Level", "actorFlags0", "actorFlags1"}

var srdOut = []string{"diceCalls", "resistCalls", "resistArg3_0", "resistArg3_1", "resistArg3_2", "resistArg3_3", "resistArg3_4", "resistArg3_5", "resistArg4_0", "resistArg4_1", "resistArg4_2", "resistArg4_3", "resistArg4_4", "resistArg4_5", "charDamageCalls", "charDamageArg1_0", "charDamageArg1_1", "charDamageArg1_2", "charDamageArg2_0", "charDamageArg2_1", "charDamageArg2_2", "charDamageArg3_0", "charDamageArg3_1", "charDamageArg3_2", "hitResponseCalls", "hitResponseArg1_0", "hitResponseArg1_1", "hitResponseArg1_2", "hitResponseArg3_0", "hitResponseArg3_1", "hitResponseArg3_2", "removeStateCalls", "removeStateArg1_0", "removeStateArg1_1", "removeStateArg1_2", "killRewardsCalls", "killRewardsArg1_0", "killRewardsArg1_1", "killRewardsArg1_2", "killRewardsArg2_0", "killRewardsArg2_1", "killRewardsArg2_2", "conditionApplyCalls", "conditionApplyArg1_0", "conditionApplyArg1_1", "conditionApplyArg1_2", "conditionApplyArg2_0", "conditionApplyArg2_1", "conditionApplyArg2_2", "conditionApplyArg3_0", "conditionApplyArg3_1", "conditionApplyArg3_2", "conditionApplyArg4_0", "conditionApplyArg4_1", "conditionApplyArg4_2", "conditionApplyArg6_0", "conditionApplyArg6_1", "conditionApplyArg6_2", "shieldMessageCalls", "shieldMessageArg1_0", "shieldMessageArg1_1", "shieldMessageArg1_2", "shieldMessageArg2_0", "shieldMessageArg2_1", "shieldMessageArg2_2", "shieldMessageArg3_0", "shieldMessageArg3_1", "shieldMessageArg3_2", "messageCalls", "messageArg1_0", "messageArg1_1", "messageArg1_2", "messageArg1_3", "messageArg1_4", "messageArg1_5", "messageArg1_6", "messageArg1_7", "messageArg1_8", "messageArg1_9", "messageArg1_10", "messageArg1_11", "messageArg1_12", "messageArg1_13", "messageArg1_14", "messageArg1_15", "messageArg1_16", "messageArg1_17", "messageArg1_18", "messageArg1_19", "messageArg2_0", "messageArg2_1", "messageArg2_2", "messageArg2_3", "messageArg2_4", "messageArg2_5", "messageArg2_6", "messageArg2_7", "messageArg2_8", "messageArg2_9", "messageArg2_10", "messageArg2_11", "messageArg2_12", "messageArg2_13", "messageArg2_14", "messageArg2_15", "messageArg2_16", "messageArg2_17", "messageArg2_18", "messageArg2_19", "messageArg3_0", "messageArg3_1", "messageArg3_2", "messageArg3_3", "messageArg3_4", "messageArg3_5", "messageArg3_6", "messageArg3_7", "messageArg3_8", "messageArg3_9", "messageArg3_10", "messageArg3_11", "messageArg3_12", "messageArg3_13", "messageArg3_14", "messageArg3_15", "messageArg3_16", "messageArg3_17", "messageArg3_18", "messageArg3_19", "messageArg4_0", "messageArg4_1", "messageArg4_2", "messageArg4_3", "messageArg4_4", "messageArg4_5", "messageArg4_6", "messageArg4_7", "messageArg4_8", "messageArg4_9", "messageArg4_10", "messageArg4_11", "messageArg4_12", "messageArg4_13", "messageArg4_14", "messageArg4_15", "messageArg4_16", "messageArg4_17", "messageArg4_18", "messageArg4_19", "portraitCalls", "portraitArg1_0", "portraitArg1_1", "portraitArg1_2", "portraitArg1_3", "portraitArg1_4", "portraitArg1_5", "portraitArg1_6", "portraitArg1_7", "portraitArg1_8", "portraitArg1_9", "portraitArg1_10", "portraitArg1_11", "portraitArg1_12", "portraitArg1_13", "portraitArg1_14", "portraitArg1_15", "portraitArg1_16", "portraitArg1_17", "portraitArg1_18", "portraitArg1_19", "portraitArg3_0", "portraitArg3_1", "portraitArg3_2", "portraitArg3_3", "portraitArg3_4", "portraitArg3_5", "portraitArg3_6", "portraitArg3_7", "portraitArg3_8", "portraitArg3_9", "portraitArg3_10", "portraitArg3_11", "portraitArg3_12", "portraitArg3_13", "portraitArg3_14", "portraitArg3_15", "portraitArg3_16", "portraitArg3_17", "portraitArg3_18", "portraitArg3_19", "hitSoundCalls", "hitSoundArg1_0", "hitSoundArg1_1", "hitSoundArg1_2", "shieldVisualCalls", "shieldVisualArg1_0", "shieldVisualArg1_1", "shieldVisualArg1_2", "shieldVisualArg2_0", "shieldVisualArg2_1", "shieldVisualArg2_2", "deathAnimationCalls", "deathAnimationArg1_0", "deathAnimationArg1_1", "deathAnimationArg1_2", "deathAnimationArg2_0", "deathAnimationArg2_1", "deathAnimationArg2_2", "deathNotifyCalls", "deathNotifyArg1_0", "deathNotifyArg1_1", "deathNotifyArg1_2", "nameLogCalls", "nameLogArg1_0", "nameLogArg1_1", "nameLogArg1_2", "nameLogArg1_3", "nameLogArg1_4", "nameLogArg1_5", "nameLogArg3_0", "nameLogArg3_1", "nameLogArg3_2", "nameLogArg3_3", "nameLogArg3_4", "nameLogArg3_5", "hp0", "shield0", "shieldDuration0", "shieldPower0", "shieldWord0", "stamina0", "fatigue0", "rebuild470", "rebuild4f0", "rebuild2470", "rebuild24b0", "lastSource0_0", "lastSource0_1", "lastSource0_2", "lastSource0_3", "lastSource0_4", "lastSource0_5", "lastSource0_6", "lastSource0_7", "lastSource0_8", "lastSource0_9", "lastSource0_10", "lastSource0_11", "lastSource0_12", "modifiers0_0", "modifiers0_1", "modifiers0_2", "modifiers0_3", "modifiers0_4", "modifiers0_5", "modifiers0_6", "modifiers0_7", "modifiers0_8", "modifiers0_9", "modifiers0_10", "modifiers0_11", "modifiers0_12", "modifiers0_13", "modifiers0_14", "modifiers0_15", "modifiers0_16", "modifiers0_17", "modifiers0_18", "modifiers0_19", "modifiers0_20", "modifiers0_21", "modifiers0_22", "modifiers0_23", "modifiers0_24", "modifiers0_25", "hp1", "shield1", "shieldDuration1", "shieldPower1", "shieldWord1", "stamina1", "fatigue1", "rebuild471", "rebuild4f1", "rebuild2471", "rebuild24b1", "lastSource1_0", "lastSource1_1", "lastSource1_2", "lastSource1_3", "lastSource1_4", "lastSource1_5", "lastSource1_6", "lastSource1_7", "lastSource1_8", "lastSource1_9", "lastSource1_10", "lastSource1_11", "lastSource1_12", "modifiers1_0", "modifiers1_1", "modifiers1_2", "modifiers1_3", "modifiers1_4", "modifiers1_5", "modifiers1_6", "modifiers1_7", "modifiers1_8", "modifiers1_9", "modifiers1_10", "modifiers1_11", "modifiers1_12", "modifiers1_13", "modifiers1_14", "modifiers1_15", "modifiers1_16", "modifiers1_17", "modifiers1_18", "modifiers1_19", "modifiers1_20", "modifiers1_21", "modifiers1_22", "modifiers1_23", "modifiers1_24", "modifiers1_25", "partyKills0", "partyKills1", "partyKills2", "partyKills3", "partyKills4", "partyKills5", "partyKills6", "partyKills7", "resultTotal", "resultHits", "resultExhausted", "eventCount", "eventCapacity", "eventArray", "eventSlot0", "eventSlot1", "eventSlot2", "eventSlot3", "eventSlot4", "eventSlot5", "eventSlot6", "eventSlot7", "resultShown", "resultFlag", "messageOpen", "summaryText0", "summaryText1", "summaryText2", "summaryText3", "summaryText4", "summaryText5", "summaryText6", "summaryText7", "summaryText8", "summaryText9", "summaryText10", "summaryText11", "aliasText0", "aliasText1", "aliasText2", "aliasText3", "aliasText4", "aliasText5", "aliasText6", "aliasText7", "s4c6c30_a1_0", "s4c6c30_a1_1", "s4c6c30_a1_2", "s4c6c30_calls"}

var srdStubs = map[uint32]*mcaStub{
	0x4c6c30: {ret: 0, rec: [][4]uint32{{1, 4, 0x20004ff0, 3}}, cnt: 0x20004ffc},
	0x517970: {ret: 0x20004000, retN: 6, rec: [][4]uint32{}, cnt: 0x20004040, wr: []mcaWrite{}},
	0x552250: {ret: 0x20004030, retN: 0, rec: [][4]uint32{{3, 4, 0x20004048, 6}, {4, 4, 0x20004060, 6}}, cnt: 0x20004044, wr: []mcaWrite{{1, 0x0, 4, 0x20004018, 6}}},
	0x52b7e0: {ret: 0x20004030, retN: 0, rec: [][4]uint32{{1, 4, 0x2000407c, 3}, {2, 4, 0x20004088, 3}, {3, 1, 0x20004094, 3}}, cnt: 0x20004078, wr: []mcaWrite{}},
	0x52beb0: {ret: 0x20004030, retN: 0, rec: [][4]uint32{{1, 4, 0x200040a4, 3}, {3, 4, 0x200040b0, 3}}, cnt: 0x200040a0, wr: []mcaWrite{}},
	0x4e4280: {ret: 0x20004030, retN: 0, rec: [][4]uint32{{1, 4, 0x200040c0, 3}}, cnt: 0x200040bc, wr: []mcaWrite{}},
	0x4e46f0: {ret: 0x20004030, retN: 0, rec: [][4]uint32{{1, 4, 0x200040d0, 3}, {2, 1, 0x200040dc, 3}}, cnt: 0x200040cc, wr: []mcaWrite{}},
	0x523c00: {ret: 0x20004030, retN: 0, rec: [][4]uint32{{1, 4, 0x200040ec, 3}, {2, 4, 0x200040f8, 3}, {3, 4, 0x20004104, 3}, {4, 4, 0x20004110, 3}, {6, 1, 0x2000411c, 3}}, cnt: 0x200040e8, wr: []mcaWrite{}},
	0x590b40: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000412c, 3}, {2, 4, 0x20004138, 3}, {3, 4, 0x20004144, 3}}, cnt: 0x20004128, wr: []mcaWrite{}},
	0x58aad0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004154, 20}, {2, 4, 0x200041a4, 20}, {3, 4, 0x200041f4, 20}, {4, 4, 0x20004244, 20}}, cnt: 0x20004150, wr: []mcaWrite{}},
	0x58b410: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004298, 20}, {3, 4, 0x200042e8, 20}}, cnt: 0x20004294, wr: []mcaWrite{}},
	0x48f650: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000433c, 3}}, cnt: 0x20004338, wr: []mcaWrite{}},
	0x4acd80: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000434c, 3}, {2, 4, 0x20004358, 3}}, cnt: 0x20004348, wr: []mcaWrite{}},
	0x4e4db0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004368, 3}, {2, 4, 0x20004374, 3}}, cnt: 0x20004364, wr: []mcaWrite{}},
	0x4e3af0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004384, 3}}, cnt: 0x20004380, wr: []mcaWrite{}},
	0x5182e0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004394, 6}, {3, 4, 0x200043ac, 6}}, cnt: 0x20004390, wr: []mcaWrite{}},
}
