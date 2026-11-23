package calc

import "fmt"

// 0x50b590 npc_departure_stamina(idx, unused, skipSpawn, joinGroup): an NPC
// leaves the party. Its character record is copied back into the NPC object,
// the party slot is cleared, and unless skipped the NPC is respawned as a monster
// group whose leader inherits the character's conditions (0x5241e0) and stamina.
// World layout: re/emu/cases/0050b590.json. 0x50b2f0, 0x54a760, 0x4ef610,
// 0x55a0a0, 0x562a50, 0x58aad0 and 0x522090 are 排除 address stubs; 0x50f1a0,
// 0x523c00 and 0x51d1c0 are D4 callees whose results are read later, so F runs
// their Go nodes and Emu replays the case stub. 0x546e70 and 0x538e00 are D4
// address stubs in both (nothing after them reads their writes here).

const (
	ndsParty       = 8          // fixture cap: party records 0..7 (0x685178 slots, 0x685174 characters)
	ndsSlotStride  = 0x106      // 0x50b5a0..0x50b5a8: idx*0x106
	ndsCharStride  = 0x1862     // 0x50b634..0x50b644 / 0x50b649 (0x618 dwords + 1 word)
	ndsAuxBase     = 0x6836b8   // 0x50b6af: per-party record, 0x118 bytes (0x50b6aa: 0x46 dwords)
	ndsAuxStride   = 0x118      // 0x50b69c..0x50b6a7: idx*0x118
	ndsTmplStride  = 0x297      // 0x50b71a: template record stride
	ndsItemStride  = 0x10d      // 0x50dc7e..0x50dc89: item record stride
	ndsCacheBase   = 0x6840c7   // 0x4e57e8: monster template cache
	ndsCacheCount  = 0x3e8      // 0x4e57c6: template id bound (srAssertFail past it)
	ndsNameBuf     = 0x6875c3   // 0x510303: "Al-%s" destination
	ndsNameFmt     = 0x614b44   // 0x5102fe: wide format string, read from the exe
	ndsDisposition = 0x61ec0c   // 0x5475f0: u16 string ids by disposition, read from the exe
	ndsLocalPos    = 0x7e000100 // 0x50b590's coordinate local (0x50b5cf), filled by 0x50b2f0
	ndsLocalSrc    = 0x7e000200 // 0x5241e0's 0x536150 source record (stack local)
	ndsLocalRel    = 0x7e000300 // 0x538db0 / 0x538d60 relation record (stack local)
)

type ndsWorld struct{ *mcaWorld }

// call records the arguments of call k and bumps the counter (emu.py order).
func (w ndsWorld) call(a uint32, args ...uint32) uint32 {
	s := ndsStubs[a]
	k := w.u32(s.cnt)
	for _, r := range s.rec {
		if k >= r[3] {
			panic(fmt.Sprintf("npc departure: stub %#x called more than %d times (fixture record cap)", a, r[3]))
		}
		w.put(r[2]+k*r[1], int(r[1]), uint64(args[r[0]-1]))
	}
	w.put(s.cnt, 4, uint64(k+1))
	return k
}

// replay is the case stub's effect for call k: its writes and return value.
func (w ndsWorld) replay(a, k uint32, args ...uint32) uint32 {
	s := ndsStubs[a]
	for _, x := range s.wr {
		at := x.at
		if x.n > 0 {
			if k >= x.n {
				panic(fmt.Sprintf("npc departure: stub %#x write past its %d values", a, x.n))
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
	switch {
	case s.retN > 0:
		if k >= s.retN {
			panic(fmt.Sprintf("npc departure: stub %#x return past its %d values", a, s.retN))
		}
		return w.u32(s.ret + 4*k)
	case s.ret != 0:
		return w.u32(s.ret)
	}
	return 0
}

func (w ndsWorld) stub(a uint32, args ...uint32) uint32 {
	return w.replay(a, w.call(a, args...), args...)
}

func (w ndsWorld) zero(a, n uint32) {
	for i := uint32(0); i < n; i++ {
		w.put(a+i, 1, 0)
	}
}

// depart is 0x50b590.
func (w ndsWorld) depart(idx uint32, skip, join bool) bool {
	if idx >= ndsParty {
		panic("npc departure: party index past the fixture's 8 records")
	}
	slot := w.u32(0x685178) + idx*ndsSlotStride
	if int32(w.u32(slot+0xfa)) == -1 { // 0x50b5b7
		return false
	}
	if !skip {
		w.stub(0x50b2f0, idx, 1, ndsLocalPos)
	}
	vec, id := w.u32(0x689f94), int32(w.u32(slot+0xfa))
	if vec == 0 { // 0x50b5fc
		return false
	}
	ep := w.u32(vec + 0xc)
	if id < int32(w.u32(vec+4)) { // 0x50b602 signed; otherwise element 0 (0x50b60d)
		ep += uint32(id) * 4
	}
	obj := w.u32(ep)
	if obj == 0 || w.u8(obj+0xc7) != 0 {
		return false
	}
	char := w.u32(0x685174) + idx*ndsCharStride
	dst := w.u32(obj + 0x27)
	for i := uint32(0); i < ndsCharStride; i++ { // 0x50b64e rep movs
		w.put(dst+i, 1, uint64(w.u8(char+i)))
	}
	w.put(obj+0x26, 1, 0)
	w.stub(0x55a0a0, w.u32(obj))
	w.put(obj, 4, 0)
	w.stub(0x4ef610, idx, 0)
	w.zero(char, ndsCharStride)
	w.zero(slot, ndsSlotStride)
	w.zero(ndsAuxBase+idx*ndsAuxStride, ndsAuxStride)
	w.put(slot+0xfa, 4, 0xffffffff)
	if w.u32(w.u32(obj+0x27)+0xb01) != 0x12 && !skip { // 0x50b6d4 / 0x50b6e1
		// 0x54a760 writes the template buffer into the skipSpawn argument slot.
		buf := uint32(mstStackBase + 4 + 4*2)
		w.stub(0x54a760, buf)
		recs, n, k := w.u32(buf), w.u32(0x683f84), uint32(0)
		for ; k < n; k++ {
			r := recs + k*ndsTmplStride
			if w.u8(r+0xd0)&1 != 0 && w.u8(r+0xcd) == w.u8(obj+0x2e) {
				break
			}
		}
		// 0x54a880 frees the buffer (free is a no-op in the replay).
		if k == w.u32(0x683f84) {
			return false
		}
		g := w.spawn(k)
		h, ok := w.monsterHandle(w.u32(g + 0x9f))
		if !ok {
			panic("npc departure: spawned leader id not found (0x4e5550 asserts)")
		}
		mon := w.monsterRecord(h)
		copied := w.u32(obj + 0x27)
		w.conditions(copied, mon)
		if w.u32(mon+0x9b) == 0x270f { // 0x50b78e
			sp, max := w.u32(copied+0xb1d), w.u32(copied+0xb19)
			if sp < max {
				max = sp
			}
			w.put(mon+0x33, 4, uint64(max))
		}
		if join {
			w.joinGroup(g)
			w.put(g+0xca, 1, 1)
		}
	}
	w.stub(0x562a50, 0xffffffff)
	w.items(obj, w.u32(obj+0x27))
	w.put(obj+0x1c, 1, 1)
	w.put(obj+0x1e, 4, 0)
	w.put(obj+0xe9, 1, 1)
	return true
}

// cache is 0x4e57c0: a miss loads the template file (0x54a8a0), which the
// replay cannot do.
func (w ndsWorld) cache(id uint32) uint32 {
	if id >= ndsCacheCount {
		panic("npc departure: template id past 0x3e8 (0x4e57ce asserts)")
	}
	t := w.u32(ndsCacheBase + 4*id)
	if t == 0 {
		panic("npc departure: template cache miss loads the template file (0x54a8a0)")
	}
	return t
}

// spawn is 0x50f1a0(k, 1, &pos, 1, 0, 1). The group record is the one the
// case's stub returns; F fills it the way egRun.spawn does.
func (w ndsWorld) spawn(k uint32) uint32 {
	g := w.replay(0x50f1a0, w.call(0x50f1a0, k, 1, ndsLocalPos, 1, 0, 1), k, 1, ndsLocalPos, 1, 0, 1)
	if !w.real {
		return g
	}
	t := w.cache(k)
	s16 := func(a uint32) int64 { return int64(int16(w.u16(a))) }
	b := func(a uint32) int64 { return int64(w.u8(a)) }
	d := func(a uint32) int64 { return int64(w.u32(a)) }
	out := monsterGroupSpawnNode([]int64{
		s16(t + 0xd6), b(t + 0xd8), b(t + 0xd9), s16(t + 0xda), b(t + 0xdc), b(t + 0xdd),
		b(t + 0x26a), d(0x6874be), d(ndsLocalPos), d(ndsLocalPos + 4), d(ndsLocalPos + 8), 0, 0, 0,
		int64(int32(w.u32(0x68ec78))), d(ndsFOnly), 1, 1, 0, 1, b(t + 0xd0), d(0x6874ba),
		d(0x686a48), b(0x689b32), 0, int64(w.u16(t + 0x187)), d(ndsFOnly + 4), d(ndsFOnly + 8),
	}, w.r)
	w.put(g+4, 4, uint64(out[2]))
	w.put(g+0x18, 4, uint64(k))
	w.put(g+0x9f, 4, uint64(out[19]))
	w.put(0x6874ba, 4, uint64(out[21]))
	w.put(0x6874be, 4, uint64(out[22]))
	return g
}

// conditions is 0x5241e0(copy, mon): each non-zero character condition is
// applied to the monster; condition 7 also carries its penalty (+0xb09).
func (w ndsWorld) conditions(copied, mon uint32) {
	for i := uint32(0); i < 20; i++ {
		v := w.u32(copied + 0xa01 + 4*i)
		if v == 0 {
			continue
		}
		pen := uint32(0)
		if i == 7 {
			pen = w.u32(copied + 0xb09)
		}
		id := w.u32(mon)
		w.replay(0x523c00, w.call(0x523c00, id, i, v, pen, ndsLocalSrc, 0))
		if w.real {
			w.applyReal(mon, i, v, pen)
		}
	}
}

// applyReal runs the real 0x523c00 on the parked world of the 0x523c00 case
// (sdrApplyParked) with this monster's fields, its visual/actor and the UI word
// mapped in by address, and copies the monster/UI results back.
func (w ndsWorld) applyReal(mon, cond, duration, penalty uint32) {
	vis := w.u32(mon + 0xc)
	ui := w.u32(0x68edcc)
	where := func(a uint32) (uint32, bool) {
		switch {
		case a < mstArena || a >= mstArena+mstArenaSize:
			return a, a < mstStackBase
		case a < mstArena+0x400: // object M
			return mon + a - mstArena, true
		case a >= mstArena+0xb00 && a < mstArena+0xe40: // object vis
			return vis + a - mstArena - 0xb00, true
		case a >= mstArena+0xe40 && a < mstArena+0xf40: // object actor
			return w.u32(vis+0x1d8) + a - mstArena - 0xe40, true
		case a >= mstArena+0x1a00 && a < mstArena+0x1b00 && ui != 0: // object ui
			return ui + a - mstArena - 0x1a00, true
		}
		return 0, false
	}
	x := append([]int64(nil), sdrApplyParked...)
	for i, f := range mcaSpecApply.in {
		switch mcaSpecApplyIn[i] {
		case "cond":
			x[i] = int64(cond)
		case "duration":
			x[i] = int64(duration)
		case "penalty":
			x[i] = int64(penalty)
		case "message":
			x[i] = 0
		case "a5.type", "a5.f1b", "a5.f1c", "a5.f1e": // 0x536150
			x[i] = 0
		case "a5.char", "a5.monster":
			x[i] = -1
		default:
			if a, ok := where(f.addr); ok && f.idx < 0 {
				x[i] = w.read(mstFld{addr: a, size: f.size, signed: f.signed, idx: -1}, nil)
			}
		}
	}
	o := monsterConditionApplyNode(x, w.r)
	for i, f := range mcaSpecApply.out {
		if a, ok := where(f.addr); ok && f.idx < 0 && f.addr >= mstArena {
			w.put(a, int(f.size), uint64(o[i]))
		}
	}
}

// joinGroup is 0x547570(g, 0, 0).
func (w ndsWorld) joinGroup(g uint32) {
	if w.allDown(g) { // 0x511850
		return
	}
	if w.u8(w.u32(w.findMonster(w.u32(g+0x9f))+0xc)+0x332) != 0 { // 0x5475a9
		return
	}
	if w.u8(g+0x2a) == 0 { // 0x5475be: already the requested state 0
		return
	}
	w.put(g+0x2a, 1, 0)
	w.put(g+0xca, 1, 0)
	if w.anyAwake(g) { // 0x505ea0
		st, sel := w.u32(0x68c09c), uint32(0)
		if w.u32(g+4) != 1 {
			sel = 1
		}
		w.stub(0x58aad0, 9, 0x61dd24, w.groupName(g), w.u32(st+0x75c+4*sel), w.u32(st+4*w.u16(ndsDisposition)))
	}
	w.put(g+0xcb, 4, uint64(w.u32(0x686a48)))
	w.stub(0x538e00, ndsLocalRel, 1) // 0x538db0([g], 1): type 4 record
	l := w.u32(g + 8)
	for i := uint32(0); i < w.listCount(l); i++ {
		w.calm(w.member(l, i))
	}
	for _, off := range [3]uint32{0xa3, 0xa7, 0xab} {
		if id := w.u32(g + off); id != 0 {
			w.joinGroup(w.group(id))
		}
	}
	// 0x547728: the third argument is 0, so the 0x510180 block is skipped.
	if ui := w.u32(0x68edcc); w.u32(0x68ec78) == 7 && ui != 0 { // 0x565420
		w.put(ui+0xf4, 4, uint64(w.u32(ui+0xf4)|0x20000))
	}
}

// member is 0x5e2c80 then 0x4e5550 (assert flag) and 0x4e5620.
func (w ndsWorld) member(l, i uint32) uint32 {
	h, ok := w.monsterHandle(w.listAt(l, i))
	if !ok {
		panic("npc departure: group member id not found (0x4e5550 asserts)")
	}
	return w.monsterRecord(h)
}

// allDown is 0x511850: every member's actor is in state 0x15.
func (w ndsWorld) allDown(g uint32) bool {
	l := w.u32(g + 8)
	for i := uint32(0); i < w.listCount(l); i++ {
		if !w.act15(w.u32(w.member(l, i) + 0xc)) {
			return false
		}
	}
	return true
}

// anyAwake is 0x505ea0.
func (w ndsWorld) anyAwake(g uint32) bool {
	l := w.u32(g + 8)
	for i := uint32(0); i < w.listCount(l); i++ {
		m := w.findMonster(w.listAt(l, i))
		if w.u8(m+0x14) != 0 && !w.act15(w.u32(m+0xc)) && w.u32(m+0x2b) != 0 && w.u32(m+0x107) < 0xc && w.u8(m+0x28a) == 1 {
			return true
		}
	}
	return false
}

// calm is 0x5477d0(m, 0).
func (w ndsWorld) calm(m uint32) {
	old := w.u8(m + 0x16)
	if old == 0 || w.u8(w.u32(m+0xc)+0x332) != 0 {
		return
	}
	w.put(m+0x16, 1, 0)
	if w.u8(m+0x15) != 0 && old == 1 { // 0x5477fe..0x547807
		w.stub(0x546e70)
	}
	w.stub(0x538e00, ndsLocalRel, 1) // 0x538d60([m], 1): type 3 record
	// 0x54782d: +0x16 is now 0, so nothing else runs.
}

// groupName is 0x510280.
func (w ndsWorld) groupName(g uint32) uint32 {
	t := w.cache(w.u32(g + 0x18))
	sel := uint32(0)
	if w.u32(g+4) != 1 {
		sel = 1
	}
	if w.u16(t+0x187) == 0x18d {
		w.alName(w.u32(0x685174) + w.u32(0x6875ef)*ndsCharStride + 5)
		return ndsNameBuf
	}
	if w.u8(g+0x2c) != 0 {
		return t + 0x30*sel
	}
	return t + 0x30*(sel+2)
}

// alName is swprintf(0x6875c3, L"Al-%s", name) (0x510308); the format is read
// from the exe and must be a literal prefix followed by %s.
func (w ndsWorld) alName(src uint32) {
	dst, f := uint32(ndsNameBuf), uint32(ndsNameFmt)
	for ; w.u16(f) != '%'; f += 2 {
		if w.u16(f) == 0 {
			panic("npc departure: format at 0x614b44 has no %s")
		}
		w.put(dst, 2, uint64(w.u16(f)))
		dst += 2
	}
	if w.u16(f+2) != 's' || w.u16(f+4) != 0 {
		panic("npc departure: format at 0x614b44 is not <prefix>%s")
	}
	for ; w.u16(src) != 0; src += 2 {
		w.put(dst, 2, uint64(w.u16(src)))
		dst += 2
	}
	w.put(dst, 2, 0)
}

// items is 0x50ddc0(obj, copy): equipped (12, only if 0x51d1c0 allows removal)
// and backpack (8) items the NPC does not keep go through 0x522090.
func (w ndsWorld) items(obj, copied uint32) {
	lost, kept := false, false
	drop := func(slot uint32) {
		if w.keeps(obj, slot) {
			return
		}
		if byte(w.stub(0x522090, slot, 0)) != 0 {
			kept = true
		} else {
			lost = true
		}
	}
	for i := uint32(0); i < 12; i++ {
		slot := copied + 0xf5d + 12*i
		if w.u32(slot) == 0xffffffff || !w.removable(copied, i, slot) {
			continue
		}
		drop(slot)
	}
	for i := uint32(0); i < 8; i++ {
		if slot := copied + 0x1029 + 12*i; w.u32(slot) != 0xffffffff {
			drop(slot)
		}
	}
	st := w.u32(0x68c09c)
	if kept {
		w.stub(0x58aad0, 0, w.u32(st+0x7f4), w.u32(obj+6)+4)
	}
	if lost {
		w.stub(0x58aad0, 0, w.u32(st+0x7f8), w.u32(obj+6)+4)
	}
}

// removable is 0x51d1c0(copy, i).
func (w ndsWorld) removable(copied, i, slot uint32) bool {
	r := w.replay(0x51d1c0, w.call(0x51d1c0, copied, i))
	if !w.real {
		return byte(r) != 0
	}
	id := w.u32(slot)
	o := itemRemovalAllowedNode([]int64{int64(i), int64(id), int64(w.u8(w.u32(0x68516c) + id*ndsItemStride + 0x8c)),
		int64(w.u8(copied + 0xf67 + 12*i)), int64(w.u32(copied + 0xa49))}, w.r)
	return o[0] != 0
}

// keeps is 0x50dc50(obj, slot): the NPC's three keep-item ids (+0x2bb, 1-based)
// match the slot's item, or both are kind 0x83.
func (w ndsWorld) keeps(obj, slot uint32) bool {
	keep, items, id := w.u32(obj+6)+0x2bb, w.u32(0x68516c), w.u32(slot)
	for j := uint32(0); j < 3; j++ {
		s := int32(int16(w.u16(keep+2*j))) - 1
		if s < 0 {
			continue
		}
		if uint32(s) == id || w.u16(items+uint32(s)*ndsItemStride+0x3f) == 0x83 && w.u16(items+id*ndsItemStride+0x3f) == 0x83 {
			if w.u8(obj+0x26) != 0 {
				panic("npc departure: 0x50dcc3 is unreachable, 0x50b652 cleared obj+0x26")
			}
			return true
		}
	}
	return false
}

func ndsRun(in []int64, r Rand, real bool) []int64 {
	return mcaRun(&ndsSpec, real, func(m *mcaWorld) {
		w := ndsWorld{m}
		ok := w.depart(mcaArg(m, 0), byte(mcaArg(m, 2)) != 0, byte(mcaArg(m, 3)) != 0)
		v := uint64(0)
		if ok {
			v = 1
		}
		w.put(mstRet, 1, v)
	})(in, r)
}

func npcDepartureStaminaNode(in []int64, r Rand) []int64 { return ndsRun(in, r, true) }

func ndsEmu(in []int64, r Rand) []int64 { return ndsRun(in, r, false) }

func init() {
	Nodes["0050b590"] = Node{Addr: "0050b590", In: ndsInNames, Out: ndsOutNames, F: npcDepartureStaminaNode, Emu: ndsEmu}
}

// Code below is generated from the same description as re/emu/cases/0050b590.json.
const (
	ndsPos   = 0x20000000 // 0x50b2f0 stub coordinate inputs
	ndsFOnly = 0x20000014 // F-only 0x50f1a0 inputs: battle flags, animation frames, unborn count
)

var ndsSpec = mstSpec{
	set: []mstSet{
		{0x685178, 4, 0x20001000},
		{0x685174, 4, 0x2000a000},
		{0x68516c, 4, 0x20003c00},
		{0x68c09c, 4, 0x20002000},
		{0x683fad, 4, 0x20004940},
		{0x683fb9, 4, 0x20004980},
		{0x683fb1, 4, 0x20004800},
		{0x683fbd, 4, 0x20004840},
		{0x68edcc, 4, 0x20001f00},
		{0x6840c7, 4, 0x20002800},
		{0x6840cb, 4, 0x20002b00},
		{0x6840cf, 4, 0x2000d000},
		{0x6840d3, 4, 0x2000d300},
		{0x6874be, 4, 0x65},
		{0x6874ba, 4, 0x9},
		{0x2000275c, 4, 0x300001d7},
		{0x20002760, 4, 0x300001d8},
		{0x200027f4, 4, 0x300001fd},
		{0x200027f8, 4, 0x300001fe},
		{0x200027fc, 4, 0x300001ff},
		{0x2000190c, 4, 0x20001910},
		{0x20001910, 4, 0x20001a00},
		{0x20001914, 4, 0x20001b00},
		{0x20001918, 4, 0x0},
		{0x2000191c, 4, 0x20001a00},
		{0x20001a27, 4, 0x20008000},
		{0x20001a06, 4, 0x20001c00},
		{0x20001bc7, 1, 0x1},
		{0x20001b27, 4, 0x20008000},
		{0x20001b06, 4, 0x20001c00},
		{0x200010fa, 4, 0x14},
		{0x20001000, 4, 0x100},
		{0x2000ab1d, 4, 0x28},
		{0x2000ab19, 4, 0x3c},
		{0x2000ab01, 4, 0x11},
		{0x2000aa01, 4, 0x5},
		{0x2000a005, 2, 0x41},
		{0x2000a007, 2, 0x61},
		{0x6836b8, 4, 0x200},
		{0x6837cc, 4, 0x300},
		{0x2000af5d, 4, 0xffffffff},
		{0x2000af69, 4, 0xffffffff},
		{0x2000af75, 4, 0xffffffff},
		{0x2000af81, 4, 0xffffffff},
		{0x2000af8d, 4, 0xffffffff},
		{0x2000af99, 4, 0xffffffff},
		{0x2000afa5, 4, 0xffffffff},
		{0x2000afb1, 4, 0xffffffff},
		{0x2000afbd, 4, 0xffffffff},
		{0x2000afc9, 4, 0xffffffff},
		{0x2000afd5, 4, 0xffffffff},
		{0x2000afe1, 4, 0xffffffff},
		{0x2000b029, 4, 0xffffffff},
		{0x2000b035, 4, 0xffffffff},
		{0x2000b041, 4, 0xffffffff},
		{0x2000b04d, 4, 0xffffffff},
		{0x2000b059, 4, 0xffffffff},
		{0x2000b065, 4, 0xffffffff},
		{0x2000b071, 4, 0xffffffff},
		{0x2000b07d, 4, 0xffffffff},
		{0x20001200, 4, 0x15},
		{0x20001106, 4, 0x101},
		{0x2000c37f, 4, 0x29},
		{0x2000c37b, 4, 0x3d},
		{0x2000c363, 4, 0x11},
		{0x2000c267, 4, 0x6},
		{0x2000b867, 2, 0x42},
		{0x2000b869, 2, 0x62},
		{0x6837d0, 4, 0x201},
		{0x6838e4, 4, 0x301},
		{0x2000c7bf, 4, 0xffffffff},
		{0x2000c7cb, 4, 0xffffffff},
		{0x2000c7d7, 4, 0xffffffff},
		{0x2000c7e3, 4, 0xffffffff},
		{0x2000c7ef, 4, 0xffffffff},
		{0x2000c7fb, 4, 0xffffffff},
		{0x2000c807, 4, 0xffffffff},
		{0x2000c813, 4, 0xffffffff},
		{0x2000c81f, 4, 0xffffffff},
		{0x2000c82b, 4, 0xffffffff},
		{0x2000c837, 4, 0xffffffff},
		{0x2000c843, 4, 0xffffffff},
		{0x2000c88b, 4, 0xffffffff},
		{0x2000c897, 4, 0xffffffff},
		{0x2000c8a3, 4, 0xffffffff},
		{0x2000c8af, 4, 0xffffffff},
		{0x2000c8bb, 4, 0xffffffff},
		{0x2000c8c7, 4, 0xffffffff},
		{0x2000c8d3, 4, 0xffffffff},
		{0x2000c8df, 4, 0xffffffff},
		{0x20001306, 4, 0x16},
		{0x2000120c, 4, 0x102},
		{0x2000dbe1, 4, 0x2a},
		{0x2000dbdd, 4, 0x3e},
		{0x2000dbc5, 4, 0x11},
		{0x2000dacd, 4, 0x7},
		{0x2000d0c9, 2, 0x43},
		{0x2000d0cb, 2, 0x63},
		{0x6838e8, 4, 0x202},
		{0x6839fc, 4, 0x302},
		{0x2000e021, 4, 0xffffffff},
		{0x2000e02d, 4, 0xffffffff},
		{0x2000e039, 4, 0xffffffff},
		{0x2000e045, 4, 0xffffffff},
		{0x2000e051, 4, 0xffffffff},
		{0x2000e05d, 4, 0xffffffff},
		{0x2000e069, 4, 0xffffffff},
		{0x2000e075, 4, 0xffffffff},
		{0x2000e081, 4, 0xffffffff},
		{0x2000e08d, 4, 0xffffffff},
		{0x2000e099, 4, 0xffffffff},
		{0x2000e0a5, 4, 0xffffffff},
		{0x2000e0ed, 4, 0xffffffff},
		{0x2000e0f9, 4, 0xffffffff},
		{0x2000e105, 4, 0xffffffff},
		{0x2000e111, 4, 0xffffffff},
		{0x2000e11d, 4, 0xffffffff},
		{0x2000e129, 4, 0xffffffff},
		{0x2000e135, 4, 0xffffffff},
		{0x2000e141, 4, 0xffffffff},
		{0x2000140c, 4, 0x17},
		{0x20001312, 4, 0x103},
		{0x2000f443, 4, 0x2b},
		{0x2000f43f, 4, 0x3f},
		{0x2000f427, 4, 0x11},
		{0x2000f333, 4, 0x8},
		{0x2000e92b, 2, 0x44},
		{0x2000e92d, 2, 0x64},
		{0x683a00, 4, 0x203},
		{0x683b14, 4, 0x303},
		{0x2000f883, 4, 0xffffffff},
		{0x2000f88f, 4, 0xffffffff},
		{0x2000f89b, 4, 0xffffffff},
		{0x2000f8a7, 4, 0xffffffff},
		{0x2000f8b3, 4, 0xffffffff},
		{0x2000f8bf, 4, 0xffffffff},
		{0x2000f8cb, 4, 0xffffffff},
		{0x2000f8d7, 4, 0xffffffff},
		{0x2000f8e3, 4, 0xffffffff},
		{0x2000f8ef, 4, 0xffffffff},
		{0x2000f8fb, 4, 0xffffffff},
		{0x2000f907, 4, 0xffffffff},
		{0x2000f94f, 4, 0xffffffff},
		{0x2000f95b, 4, 0xffffffff},
		{0x2000f967, 4, 0xffffffff},
		{0x2000f973, 4, 0xffffffff},
		{0x2000f97f, 4, 0xffffffff},
		{0x2000f98b, 4, 0xffffffff},
		{0x2000f997, 4, 0xffffffff},
		{0x2000f9a3, 4, 0xffffffff},
		{0x20001512, 4, 0x18},
		{0x20001418, 4, 0x104},
		{0x20010ca5, 4, 0x2c},
		{0x20010ca1, 4, 0x40},
		{0x20010c89, 4, 0x11},
		{0x20010b99, 4, 0x9},
		{0x2001018d, 2, 0x45},
		{0x2001018f, 2, 0x65},
		{0x683b18, 4, 0x204},
		{0x683c2c, 4, 0x304},
		{0x200110e5, 4, 0xffffffff},
		{0x200110f1, 4, 0xffffffff},
		{0x200110fd, 4, 0xffffffff},
		{0x20011109, 4, 0xffffffff},
		{0x20011115, 4, 0xffffffff},
		{0x20011121, 4, 0xffffffff},
		{0x2001112d, 4, 0xffffffff},
		{0x20011139, 4, 0xffffffff},
		{0x20011145, 4, 0xffffffff},
		{0x20011151, 4, 0xffffffff},
		{0x2001115d, 4, 0xffffffff},
		{0x20011169, 4, 0xffffffff},
		{0x200111b1, 4, 0xffffffff},
		{0x200111bd, 4, 0xffffffff},
		{0x200111c9, 4, 0xffffffff},
		{0x200111d5, 4, 0xffffffff},
		{0x200111e1, 4, 0xffffffff},
		{0x200111ed, 4, 0xffffffff},
		{0x200111f9, 4, 0xffffffff},
		{0x20011205, 4, 0xffffffff},
		{0x20001618, 4, 0x19},
		{0x2000151e, 4, 0x105},
		{0x20012507, 4, 0x2d},
		{0x20012503, 4, 0x41},
		{0x200124eb, 4, 0x11},
		{0x200123ff, 4, 0xa},
		{0x200119ef, 2, 0x46},
		{0x200119f1, 2, 0x66},
		{0x683c30, 4, 0x205},
		{0x683d44, 4, 0x305},
		{0x20012947, 4, 0xffffffff},
		{0x20012953, 4, 0xffffffff},
		{0x2001295f, 4, 0xffffffff},
		{0x2001296b, 4, 0xffffffff},
		{0x20012977, 4, 0xffffffff},
		{0x20012983, 4, 0xffffffff},
		{0x2001298f, 4, 0xffffffff},
		{0x2001299b, 4, 0xffffffff},
		{0x200129a7, 4, 0xffffffff},
		{0x200129b3, 4, 0xffffffff},
		{0x200129bf, 4, 0xffffffff},
		{0x200129cb, 4, 0xffffffff},
		{0x20012a13, 4, 0xffffffff},
		{0x20012a1f, 4, 0xffffffff},
		{0x20012a2b, 4, 0xffffffff},
		{0x20012a37, 4, 0xffffffff},
		{0x20012a43, 4, 0xffffffff},
		{0x20012a4f, 4, 0xffffffff},
		{0x20012a5b, 4, 0xffffffff},
		{0x20012a67, 4, 0xffffffff},
		{0x2000171e, 4, 0x1a},
		{0x20001624, 4, 0x106},
		{0x20013d69, 4, 0x2e},
		{0x20013d65, 4, 0x42},
		{0x20013d4d, 4, 0x11},
		{0x20013c65, 4, 0xb},
		{0x20013251, 2, 0x47},
		{0x20013253, 2, 0x67},
		{0x683d48, 4, 0x206},
		{0x683e5c, 4, 0x306},
		{0x200141a9, 4, 0xffffffff},
		{0x200141b5, 4, 0xffffffff},
		{0x200141c1, 4, 0xffffffff},
		{0x200141cd, 4, 0xffffffff},
		{0x200141d9, 4, 0xffffffff},
		{0x200141e5, 4, 0xffffffff},
		{0x200141f1, 4, 0xffffffff},
		{0x200141fd, 4, 0xffffffff},
		{0x20014209, 4, 0xffffffff},
		{0x20014215, 4, 0xffffffff},
		{0x20014221, 4, 0xffffffff},
		{0x2001422d, 4, 0xffffffff},
		{0x20014275, 4, 0xffffffff},
		{0x20014281, 4, 0xffffffff},
		{0x2001428d, 4, 0xffffffff},
		{0x20014299, 4, 0xffffffff},
		{0x200142a5, 4, 0xffffffff},
		{0x200142b1, 4, 0xffffffff},
		{0x200142bd, 4, 0xffffffff},
		{0x200142c9, 4, 0xffffffff},
		{0x20001824, 4, 0x1b},
		{0x2000172a, 4, 0x107},
		{0x200155cb, 4, 0x2f},
		{0x200155c7, 4, 0x43},
		{0x200155af, 4, 0x11},
		{0x200154cb, 4, 0xc},
		{0x20014ab3, 2, 0x48},
		{0x20014ab5, 2, 0x68},
		{0x683e60, 4, 0x207},
		{0x683f74, 4, 0x307},
		{0x20015a0b, 4, 0xffffffff},
		{0x20015a17, 4, 0xffffffff},
		{0x20015a23, 4, 0xffffffff},
		{0x20015a2f, 4, 0xffffffff},
		{0x20015a3b, 4, 0xffffffff},
		{0x20015a47, 4, 0xffffffff},
		{0x20015a53, 4, 0xffffffff},
		{0x20015a5f, 4, 0xffffffff},
		{0x20015a6b, 4, 0xffffffff},
		{0x20015a77, 4, 0xffffffff},
		{0x20015a83, 4, 0xffffffff},
		{0x20015a8f, 4, 0xffffffff},
		{0x20015ad7, 4, 0xffffffff},
		{0x20015ae3, 4, 0xffffffff},
		{0x20015aef, 4, 0xffffffff},
		{0x20015afb, 4, 0xffffffff},
		{0x20015b07, 4, 0xffffffff},
		{0x20015b13, 4, 0xffffffff},
		{0x20015b1f, 4, 0xffffffff},
		{0x20015b2b, 4, 0xffffffff},
		{0x20004800, 4, 0x20004810},
		{0x20004808, 4, 0x3},
		{0x20004810, 4, 0x20004500},
		{0x20004814, 4, 0x20004600},
		{0x20004818, 4, 0x20004700},
		{0x20004840, 4, 0x20004850},
		{0x20004848, 4, 0x0},
		{0x20004940, 4, 0x20004950},
		{0x20004948, 4, 0x4},
		{0x20004950, 4, 0x20005000},
		{0x20004954, 4, 0x20005400},
		{0x20004958, 4, 0x20005800},
		{0x2000495c, 4, 0x20005c00},
		{0x20004980, 4, 0x20004990},
		{0x20004988, 4, 0x0},
		{0x20004880, 4, 0x20004890},
		{0x20004888, 4, 0x2},
		{0x20004890, 4, 0x65},
		{0x20004894, 4, 0x66},
		{0x200048c0, 4, 0x200048d0},
		{0x200048c8, 4, 0x1},
		{0x200048d0, 4, 0x67},
		{0x20004900, 4, 0x20004910},
		{0x20004908, 4, 0x1},
		{0x20004910, 4, 0x68},
		{0x20004500, 4, 0x1},
		{0x20004508, 4, 0x20004880},
		{0x20004600, 4, 0x2},
		{0x20004608, 4, 0x200048c0},
		{0x20004700, 4, 0x3},
		{0x20004708, 4, 0x20004900},
		{0x200046a3, 4, 0x1},
		{0x20004604, 4, 0x1},
		{0x2000462c, 1, 0x0},
		{0x20004618, 4, 0x1},
		{0x20004704, 4, 0x2},
		{0x2000472c, 1, 0x1},
		{0x20004718, 4, 0x0},
		{0x20005000, 4, 0x65},
		{0x2000500c, 4, 0x20006000},
		{0x20005008, 4, 0x0},
		{0x200061d8, 4, 0x20007000},
		{0x20005400, 4, 0x66},
		{0x2000540c, 4, 0x20006400},
		{0x20005408, 4, 0x0},
		{0x200065d8, 4, 0x20007100},
		{0x20005800, 4, 0x67},
		{0x2000580c, 4, 0x20006800},
		{0x20005808, 4, 0x0},
		{0x200069d8, 4, 0x20007200},
		{0x20005c00, 4, 0x68},
		{0x20005c0c, 4, 0x20006c00},
		{0x20005c08, 4, 0x0},
		{0x20006dd8, 4, 0x20007300},
		{0x200028d6, 2, 0xa},
		{0x200028d8, 1, 0x2},
		{0x200028d9, 1, 0x6},
		{0x200028da, 2, 0x5},
		{0x200028dc, 1, 0x1},
		{0x200028dd, 1, 0x4},
		{0x20002bd6, 2, 0xa},
		{0x20002bd8, 1, 0x2},
		{0x20002bd9, 1, 0x6},
		{0x20002bda, 2, 0x5},
		{0x20002bdc, 1, 0x1},
		{0x20002bdd, 1, 0x4},
		{0x2000d0d6, 2, 0xa},
		{0x2000d0d8, 1, 0x2},
		{0x2000d0d9, 1, 0x6},
		{0x2000d0da, 2, 0x5},
		{0x2000d0dc, 1, 0x1},
		{0x2000d0dd, 1, 0x4},
		{0x2000d3d6, 2, 0xa},
		{0x2000d3d8, 1, 0x2},
		{0x2000d3d9, 1, 0x6},
		{0x2000d3da, 2, 0x5},
		{0x2000d3dc, 1, 0x1},
		{0x2000d3dd, 1, 0x4},
	},
	in: []mstFld{
		{0x7f000004, 4, false, -1, 0, 0x0},
		{0x7f00000c, 1, false, -1, 0, 0x0},
		{0x7f000010, 1, false, -1, 0, 0x0},
		{0x200010fa, 4, true, 0, 8, 0x106},
		{0x689f94, 4, false, -1, 0, 0x0},
		{0x20001904, 4, true, -1, 0, 0x0},
		{0x20001a00, 4, false, -1, 0, 0x0},
		{0x20001ac7, 1, false, -1, 0, 0x0},
		{0x20001a2e, 1, false, -1, 0, 0x0},
		{0x20001a1c, 1, false, -1, 0, 0x0},
		{0x20001a26, 1, false, -1, 0, 0x0},
		{0x2000ab01, 4, false, 0, 8, 0x1862},
		{0x2000aa01, 4, false, 0, 8, 0x1862},
		{0x2000aa05, 4, false, 0, 8, 0x1862},
		{0x2000aa09, 4, false, 0, 8, 0x1862},
		{0x2000aa0d, 4, false, 0, 8, 0x1862},
		{0x2000aa11, 4, false, 0, 8, 0x1862},
		{0x2000aa15, 4, false, 0, 8, 0x1862},
		{0x2000aa19, 4, false, 0, 8, 0x1862},
		{0x2000aa1d, 4, false, 0, 8, 0x1862},
		{0x2000aa21, 4, false, 0, 8, 0x1862},
		{0x2000aa25, 4, false, 0, 8, 0x1862},
		{0x2000aa29, 4, false, 0, 8, 0x1862},
		{0x2000aa2d, 4, false, 0, 8, 0x1862},
		{0x2000aa31, 4, false, 0, 8, 0x1862},
		{0x2000aa35, 4, false, 0, 8, 0x1862},
		{0x2000aa39, 4, false, 0, 8, 0x1862},
		{0x2000aa3d, 4, false, 0, 8, 0x1862},
		{0x2000aa41, 4, false, 0, 8, 0x1862},
		{0x2000aa45, 4, false, 0, 8, 0x1862},
		{0x2000aa49, 4, false, 0, 8, 0x1862},
		{0x2000aa4d, 4, false, 0, 8, 0x1862},
		{0x2000ab09, 4, false, 0, 8, 0x1862},
		{0x2000ab19, 4, false, 0, 8, 0x1862},
		{0x2000ab1d, 4, false, 0, 8, 0x1862},
		{0x2000aa49, 4, false, 0, 8, 0x1862},
		{0x2000af5d, 4, true, 0, 8, 0x1862},
		{0x2000af67, 1, false, 0, 8, 0x1862},
		{0x2000af69, 4, true, 0, 8, 0x1862},
		{0x2000af73, 1, false, 0, 8, 0x1862},
		{0x2000af75, 4, true, 0, 8, 0x1862},
		{0x2000af7f, 1, false, 0, 8, 0x1862},
		{0x2000af81, 4, true, 0, 8, 0x1862},
		{0x2000af8b, 1, false, 0, 8, 0x1862},
		{0x2000af8d, 4, true, 0, 8, 0x1862},
		{0x2000af97, 1, false, 0, 8, 0x1862},
		{0x2000af99, 4, true, 0, 8, 0x1862},
		{0x2000afa3, 1, false, 0, 8, 0x1862},
		{0x2000afa5, 4, true, 0, 8, 0x1862},
		{0x2000afaf, 1, false, 0, 8, 0x1862},
		{0x2000afb1, 4, true, 0, 8, 0x1862},
		{0x2000afbb, 1, false, 0, 8, 0x1862},
		{0x2000afbd, 4, true, 0, 8, 0x1862},
		{0x2000afc7, 1, false, 0, 8, 0x1862},
		{0x2000afc9, 4, true, 0, 8, 0x1862},
		{0x2000afd3, 1, false, 0, 8, 0x1862},
		{0x2000afd5, 4, true, 0, 8, 0x1862},
		{0x2000afdf, 1, false, 0, 8, 0x1862},
		{0x2000afe1, 4, true, 0, 8, 0x1862},
		{0x2000afeb, 1, false, 0, 8, 0x1862},
		{0x2000b029, 4, true, 0, 8, 0x1862},
		{0x2000b035, 4, true, 0, 8, 0x1862},
		{0x2000b041, 4, true, 0, 8, 0x1862},
		{0x2000b04d, 4, true, 0, 8, 0x1862},
		{0x2000b059, 4, true, 0, 8, 0x1862},
		{0x2000b065, 4, true, 0, 8, 0x1862},
		{0x2000b071, 4, true, 0, 8, 0x1862},
		{0x2000b07d, 4, true, 0, 8, 0x1862},
		{0x20001ebb, 2, true, -1, 0, 0x0},
		{0x20001ebd, 2, true, -1, 0, 0x0},
		{0x20001ebf, 2, true, -1, 0, 0x0},
		{0x20003c3f, 2, false, -1, 0, 0x0},
		{0x20003c8c, 1, false, -1, 0, 0x0},
		{0x20003d4c, 2, false, -1, 0, 0x0},
		{0x20003d99, 1, false, -1, 0, 0x0},
		{0x20003e59, 2, false, -1, 0, 0x0},
		{0x20003ea6, 1, false, -1, 0, 0x0},
		{0x20003f66, 2, false, -1, 0, 0x0},
		{0x20003fb3, 1, false, -1, 0, 0x0},
		{0x20004073, 2, false, -1, 0, 0x0},
		{0x200040c0, 1, false, -1, 0, 0x0},
		{0x20004180, 2, false, -1, 0, 0x0},
		{0x200041cd, 1, false, -1, 0, 0x0},
		{0x2000428d, 2, false, -1, 0, 0x0},
		{0x200042da, 1, false, -1, 0, 0x0},
		{0x2000439a, 2, false, -1, 0, 0x0},
		{0x200043e7, 1, false, -1, 0, 0x0},
		{0x683f84, 4, false, -1, 0, 0x0},
		{0x200030d0, 1, false, -1, 0, 0x0},
		{0x200030cd, 1, false, -1, 0, 0x0},
		{0x20003367, 1, false, -1, 0, 0x0},
		{0x20003364, 1, false, -1, 0, 0x0},
		{0x200035fe, 1, false, -1, 0, 0x0},
		{0x200035fb, 1, false, -1, 0, 0x0},
		{0x20003895, 1, false, -1, 0, 0x0},
		{0x20003892, 1, false, -1, 0, 0x0},
		{0x20002987, 2, false, -1, 0, 0x0},
		{0x20002c87, 2, false, -1, 0, 0x0},
		{0x2000d187, 2, false, -1, 0, 0x0},
		{0x2000d487, 2, false, -1, 0, 0x0},
		{0x6875ef, 4, false, -1, 0, 0x0},
		{0x68ec78, 4, false, -1, 0, 0x0},
		{0x20001ff4, 4, false, -1, 0, 0x0},
		{0x686a48, 4, false, -1, 0, 0x0},
		{0x2000459f, 4, false, -1, 0, 0x0},
		{0x2000452a, 1, false, -1, 0, 0x0},
		{0x20004504, 4, false, -1, 0, 0x0},
		{0x2000452c, 1, false, -1, 0, 0x0},
		{0x20004518, 4, false, -1, 0, 0x0},
		{0x200045a3, 4, false, -1, 0, 0x0},
		{0x200045a7, 4, false, -1, 0, 0x0},
		{0x200045ab, 4, false, -1, 0, 0x0},
		{0x2000469f, 4, false, -1, 0, 0x0},
		{0x2000462a, 1, false, -1, 0, 0x0},
		{0x2000479f, 4, false, -1, 0, 0x0},
		{0x2000472a, 1, false, -1, 0, 0x0},
		{0x20005014, 1, false, -1, 0, 0x0},
		{0x20005015, 1, false, -1, 0, 0x0},
		{0x20005016, 1, false, -1, 0, 0x0},
		{0x2000502b, 4, false, -1, 0, 0x0},
		{0x20005033, 4, false, -1, 0, 0x0},
		{0x2000509b, 4, false, -1, 0, 0x0},
		{0x20005107, 4, false, -1, 0, 0x0},
		{0x2000528a, 1, false, -1, 0, 0x0},
		{0x20006332, 1, false, -1, 0, 0x0},
		{0x200070a4, 1, false, -1, 0, 0x0},
		{0x200070a7, 1, false, -1, 0, 0x0},
		{0x20005414, 1, false, -1, 0, 0x0},
		{0x20005415, 1, false, -1, 0, 0x0},
		{0x20005416, 1, false, -1, 0, 0x0},
		{0x2000542b, 4, false, -1, 0, 0x0},
		{0x20005433, 4, false, -1, 0, 0x0},
		{0x2000549b, 4, false, -1, 0, 0x0},
		{0x20005507, 4, false, -1, 0, 0x0},
		{0x2000568a, 1, false, -1, 0, 0x0},
		{0x20006732, 1, false, -1, 0, 0x0},
		{0x200071a4, 1, false, -1, 0, 0x0},
		{0x200071a7, 1, false, -1, 0, 0x0},
		{0x20005814, 1, false, -1, 0, 0x0},
		{0x20005815, 1, false, -1, 0, 0x0},
		{0x20005816, 1, false, -1, 0, 0x0},
		{0x2000582b, 4, false, -1, 0, 0x0},
		{0x20005833, 4, false, -1, 0, 0x0},
		{0x2000589b, 4, false, -1, 0, 0x0},
		{0x20005907, 4, false, -1, 0, 0x0},
		{0x20005a8a, 1, false, -1, 0, 0x0},
		{0x20006b32, 1, false, -1, 0, 0x0},
		{0x200072a4, 1, false, -1, 0, 0x0},
		{0x200072a7, 1, false, -1, 0, 0x0},
		{0x20005c14, 1, false, -1, 0, 0x0},
		{0x20005c15, 1, false, -1, 0, 0x0},
		{0x20005c16, 1, false, -1, 0, 0x0},
		{0x20005c2b, 4, false, -1, 0, 0x0},
		{0x20005c33, 4, false, -1, 0, 0x0},
		{0x20005c9b, 4, false, -1, 0, 0x0},
		{0x20005d07, 4, false, -1, 0, 0x0},
		{0x20005e8a, 1, false, -1, 0, 0x0},
		{0x20006f32, 1, false, -1, 0, 0x0},
		{0x200073a4, 1, false, -1, 0, 0x0},
		{0x200073a7, 1, false, -1, 0, 0x0},
		{0x20000000, 4, false, -1, 0, 0x0},
		{0x20000004, 4, false, -1, 0, 0x0},
		{0x20000008, 4, false, -1, 0, 0x0},
		{0x2000000c, 4, false, -1, 0, 0x0},
		{0x20000010, 4, false, -1, 0, 0x0},
		{0x20000014, 4, false, -1, 0, 0x0},
		{0x20000018, 4, false, -1, 0, 0x0},
		{0x2000001c, 4, false, -1, 0, 0x0},
		{0x20000020, 4, false, -1, 0, 0x0},
		{0x20000024, 4, false, -1, 0, 0x0},
		{0x20000028, 4, false, -1, 0, 0x0},
		{0x2000002c, 4, false, -1, 0, 0x0},
		{0x20000030, 4, false, -1, 0, 0x0},
		{0x20000034, 4, false, -1, 0, 0x0},
		{0x20000038, 4, false, -1, 0, 0x0},
		{0x2000003c, 4, false, -1, 0, 0x0},
		{0x20000040, 4, false, -1, 0, 0x0},
		{0x20000044, 4, false, -1, 0, 0x0},
		{0x20000048, 4, false, -1, 0, 0x0},
		{0x2000004c, 4, false, -1, 0, 0x0},
		{0x20000050, 4, false, -1, 0, 0x0},
		{0x20000054, 4, false, -1, 0, 0x0},
		{0x20000058, 4, false, -1, 0, 0x0},
		{0x2000005c, 4, false, -1, 0, 0x0},
		{0x20000060, 4, false, -1, 0, 0x0},
		{0x20000064, 4, false, -1, 0, 0x0},
		{0x20000068, 4, false, -1, 0, 0x0},
		{0x2000006c, 4, false, -1, 0, 0x0},
		{0x20000070, 4, false, -1, 0, 0x0},
		{0x20000074, 4, false, -1, 0, 0x0},
		{0x20000078, 4, false, -1, 0, 0x0},
		{0x2000007c, 4, false, -1, 0, 0x0},
		{0x20000080, 4, false, -1, 0, 0x0},
		{0x20000084, 4, false, -1, 0, 0x0},
		{0x20000088, 4, false, -1, 0, 0x0},
		{0x2000008c, 4, false, -1, 0, 0x0},
		{0x20000090, 4, false, -1, 0, 0x0},
		{0x20000094, 4, false, -1, 0, 0x0},
		{0x20000098, 4, false, -1, 0, 0x0},
		{0x2000009c, 4, false, -1, 0, 0x0},
	},
	out: []mstFld{
		{0xfffffff0, 1, false, -1, 0, 0},
		{0x20001a00, 4, false, -1, 0, 0},
		{0x20001a26, 1, false, -1, 0, 0},
		{0x20001a1c, 1, false, -1, 0, 0},
		{0x20001a1e, 4, false, -1, 0, 0},
		{0x20001ae9, 1, false, -1, 0, 0},
		{0x200010fa, 4, true, -1, 0, 0},
		{0x20001000, 4, false, -1, 0, 0},
		{0x2000ab1d, 4, false, -1, 0, 0},
		{0x6836b8, 4, false, -1, 0, 0},
		{0x6837cc, 4, false, -1, 0, 0},
		{0x20001200, 4, true, -1, 0, 0},
		{0x20001106, 4, false, -1, 0, 0},
		{0x2000c37f, 4, false, -1, 0, 0},
		{0x6837d0, 4, false, -1, 0, 0},
		{0x6838e4, 4, false, -1, 0, 0},
		{0x20001306, 4, true, -1, 0, 0},
		{0x2000120c, 4, false, -1, 0, 0},
		{0x2000dbe1, 4, false, -1, 0, 0},
		{0x6838e8, 4, false, -1, 0, 0},
		{0x6839fc, 4, false, -1, 0, 0},
		{0x2000140c, 4, true, -1, 0, 0},
		{0x20001312, 4, false, -1, 0, 0},
		{0x2000f443, 4, false, -1, 0, 0},
		{0x683a00, 4, false, -1, 0, 0},
		{0x683b14, 4, false, -1, 0, 0},
		{0x20001512, 4, true, -1, 0, 0},
		{0x20001418, 4, false, -1, 0, 0},
		{0x20010ca5, 4, false, -1, 0, 0},
		{0x683b18, 4, false, -1, 0, 0},
		{0x683c2c, 4, false, -1, 0, 0},
		{0x20001618, 4, true, -1, 0, 0},
		{0x2000151e, 4, false, -1, 0, 0},
		{0x20012507, 4, false, -1, 0, 0},
		{0x683c30, 4, false, -1, 0, 0},
		{0x683d44, 4, false, -1, 0, 0},
		{0x2000171e, 4, true, -1, 0, 0},
		{0x20001624, 4, false, -1, 0, 0},
		{0x20013d69, 4, false, -1, 0, 0},
		{0x683d48, 4, false, -1, 0, 0},
		{0x683e5c, 4, false, -1, 0, 0},
		{0x20001824, 4, true, -1, 0, 0},
		{0x2000172a, 4, false, -1, 0, 0},
		{0x200155cb, 4, false, -1, 0, 0},
		{0x683e60, 4, false, -1, 0, 0},
		{0x683f74, 4, false, -1, 0, 0},
		{0x20008b01, 4, false, -1, 0, 0},
		{0x20008a01, 4, false, -1, 0, 0},
		{0x20008a1d, 4, false, -1, 0, 0},
		{0x20008b19, 4, false, -1, 0, 0},
		{0x20008b1d, 4, false, -1, 0, 0},
		{0x20008f5d, 4, false, -1, 0, 0},
		{0x20009029, 4, false, -1, 0, 0},
		{0x20005016, 1, false, -1, 0, 0},
		{0x20005033, 4, false, -1, 0, 0},
		{0x20005416, 1, false, -1, 0, 0},
		{0x20005433, 4, false, -1, 0, 0},
		{0x20005816, 1, false, -1, 0, 0},
		{0x20005833, 4, false, -1, 0, 0},
		{0x20005c16, 1, false, -1, 0, 0},
		{0x20005c33, 4, false, -1, 0, 0},
		{0x2000452a, 1, false, -1, 0, 0},
		{0x200045ca, 1, false, -1, 0, 0},
		{0x200045cb, 4, false, -1, 0, 0},
		{0x2000462a, 1, false, -1, 0, 0},
		{0x200046ca, 1, false, -1, 0, 0},
		{0x200046cb, 4, false, -1, 0, 0},
		{0x2000472a, 1, false, -1, 0, 0},
		{0x200047ca, 1, false, -1, 0, 0},
		{0x200047cb, 4, false, -1, 0, 0},
		{0x20001ff4, 4, false, -1, 0, 0},
		{0x6875c9, 4, false, -1, 0, 0},
		{0x6875cd, 4, false, -1, 0, 0},
		{0x200000b8, 4, false, -1, 0, 0},
		{0x200000bc, 4, false, -1, 0, 0},
		{0x200000c0, 4, false, -1, 0, 0},
		{0x200000a0, 4, false, -1, 0, 0},
		{0x200000a4, 4, false, -1, 0, 0},
		{0x200000a8, 4, false, -1, 0, 0},
		{0x200000c4, 4, false, -1, 0, 0},
		{0x200000c8, 4, false, -1, 0, 0},
		{0x200000cc, 4, false, -1, 0, 0},
		{0x200000d0, 4, false, -1, 0, 0},
		{0x200000d4, 4, false, -1, 0, 0},
		{0x200000d8, 4, false, -1, 0, 0},
		{0x20000278, 4, false, -1, 0, 0},
		{0x2000027c, 4, false, -1, 0, 0},
		{0x20000280, 4, false, -1, 0, 0},
		{0x20000284, 4, false, -1, 0, 0},
		{0x20000288, 4, false, -1, 0, 0},
		{0x2000028c, 4, false, -1, 0, 0},
		{0x20000290, 4, false, -1, 0, 0},
		{0x20000294, 4, false, -1, 0, 0},
		{0x20000298, 4, false, -1, 0, 0},
		{0x2000029c, 4, false, -1, 0, 0},
		{0x200002a0, 4, false, -1, 0, 0},
		{0x200002a4, 4, false, -1, 0, 0},
		{0x200002a8, 4, false, -1, 0, 0},
		{0x200002ac, 4, false, -1, 0, 0},
		{0x200002b0, 4, false, -1, 0, 0},
		{0x200002b4, 4, false, -1, 0, 0},
		{0x200002b8, 4, false, -1, 0, 0},
		{0x200002bc, 4, false, -1, 0, 0},
		{0x200002c0, 4, false, -1, 0, 0},
		{0x200002c4, 4, false, -1, 0, 0},
		{0x200002c8, 4, false, -1, 0, 0},
		{0x200002cc, 4, false, -1, 0, 0},
		{0x200002d0, 4, false, -1, 0, 0},
		{0x200002d4, 4, false, -1, 0, 0},
		{0x200002d8, 4, false, -1, 0, 0},
		{0x200002dc, 4, false, -1, 0, 0},
		{0x200002e0, 4, false, -1, 0, 0},
		{0x200002e4, 4, false, -1, 0, 0},
		{0x200002e8, 4, false, -1, 0, 0},
		{0x200002ec, 4, false, -1, 0, 0},
		{0x200002f0, 4, false, -1, 0, 0},
		{0x200002f4, 4, false, -1, 0, 0},
		{0x200002f8, 4, false, -1, 0, 0},
		{0x200002fc, 4, false, -1, 0, 0},
		{0x20000300, 4, false, -1, 0, 0},
		{0x20000304, 4, false, -1, 0, 0},
		{0x20000308, 4, false, -1, 0, 0},
		{0x2000030c, 4, false, -1, 0, 0},
		{0x20000310, 4, false, -1, 0, 0},
		{0x20000314, 4, false, -1, 0, 0},
		{0x20000318, 4, false, -1, 0, 0},
		{0x2000031c, 4, false, -1, 0, 0},
		{0x20000320, 4, false, -1, 0, 0},
		{0x20000324, 4, false, -1, 0, 0},
		{0x20000328, 4, false, -1, 0, 0},
		{0x2000032c, 4, false, -1, 0, 0},
		{0x20000330, 4, false, -1, 0, 0},
		{0x20000334, 4, false, -1, 0, 0},
		{0x20000338, 4, false, -1, 0, 0},
		{0x2000033c, 4, false, -1, 0, 0},
		{0x20000340, 4, false, -1, 0, 0},
		{0x20000344, 4, false, -1, 0, 0},
		{0x20000348, 4, false, -1, 0, 0},
		{0x2000034c, 4, false, -1, 0, 0},
		{0x200000dc, 4, false, -1, 0, 0},
		{0x200000e0, 4, false, -1, 0, 0},
		{0x200000e4, 4, false, -1, 0, 0},
		{0x200000e8, 4, false, -1, 0, 0},
		{0x200000ec, 4, false, -1, 0, 0},
		{0x200000f0, 4, false, -1, 0, 0},
		{0x200000f4, 4, false, -1, 0, 0},
		{0x200000f8, 4, false, -1, 0, 0},
		{0x200000fc, 4, false, -1, 0, 0},
		{0x20000100, 4, false, -1, 0, 0},
		{0x20000104, 4, false, -1, 0, 0},
		{0x20000108, 4, false, -1, 0, 0},
		{0x2000010c, 4, false, -1, 0, 0},
		{0x20000110, 4, false, -1, 0, 0},
		{0x20000114, 4, false, -1, 0, 0},
		{0x20000118, 4, false, -1, 0, 0},
		{0x2000011c, 4, false, -1, 0, 0},
		{0x20000120, 4, false, -1, 0, 0},
		{0x20000124, 4, false, -1, 0, 0},
		{0x20000128, 4, false, -1, 0, 0},
		{0x2000012c, 4, false, -1, 0, 0},
		{0x20000130, 4, false, -1, 0, 0},
		{0x20000134, 4, false, -1, 0, 0},
		{0x20000138, 4, false, -1, 0, 0},
		{0x2000013c, 4, false, -1, 0, 0},
		{0x20000140, 4, false, -1, 0, 0},
		{0x20000144, 4, false, -1, 0, 0},
		{0x20000148, 4, false, -1, 0, 0},
		{0x2000014c, 4, false, -1, 0, 0},
		{0x20000150, 4, false, -1, 0, 0},
		{0x20000154, 4, false, -1, 0, 0},
		{0x20000158, 4, false, -1, 0, 0},
		{0x2000015c, 4, false, -1, 0, 0},
		{0x20000160, 4, false, -1, 0, 0},
		{0x20000164, 4, false, -1, 0, 0},
		{0x20000168, 4, false, -1, 0, 0},
		{0x2000016c, 4, false, -1, 0, 0},
		{0x20000170, 4, false, -1, 0, 0},
		{0x20000174, 4, false, -1, 0, 0},
		{0x20000178, 4, false, -1, 0, 0},
		{0x2000017c, 4, false, -1, 0, 0},
		{0x20000180, 4, false, -1, 0, 0},
		{0x20000184, 4, false, -1, 0, 0},
		{0x20000188, 4, false, -1, 0, 0},
		{0x2000018c, 4, false, -1, 0, 0},
		{0x20000190, 4, false, -1, 0, 0},
		{0x20000194, 4, false, -1, 0, 0},
		{0x20000198, 4, false, -1, 0, 0},
		{0x2000019c, 4, false, -1, 0, 0},
		{0x200001a0, 4, false, -1, 0, 0},
		{0x200001a4, 4, false, -1, 0, 0},
		{0x200001a8, 4, false, -1, 0, 0},
		{0x200001ac, 4, false, -1, 0, 0},
		{0x200001b0, 4, false, -1, 0, 0},
		{0x200001b4, 4, false, -1, 0, 0},
		{0x200001b8, 4, false, -1, 0, 0},
		{0x200001bc, 4, false, -1, 0, 0},
		{0x200001c0, 4, false, -1, 0, 0},
		{0x200001c4, 4, false, -1, 0, 0},
		{0x200001c8, 4, false, -1, 0, 0},
		{0x200001cc, 4, false, -1, 0, 0},
		{0x200001d0, 4, false, -1, 0, 0},
		{0x200001d4, 4, false, -1, 0, 0},
		{0x200001d8, 4, false, -1, 0, 0},
		{0x200001dc, 4, false, -1, 0, 0},
		{0x200001e0, 4, false, -1, 0, 0},
		{0x200001e4, 4, false, -1, 0, 0},
		{0x200001e8, 4, false, -1, 0, 0},
		{0x200001ec, 4, false, -1, 0, 0},
		{0x200001f0, 4, false, -1, 0, 0},
		{0x200001f4, 4, false, -1, 0, 0},
		{0x200001f8, 4, false, -1, 0, 0},
		{0x200001fc, 4, false, -1, 0, 0},
		{0x20000200, 4, false, -1, 0, 0},
		{0x20000204, 4, false, -1, 0, 0},
		{0x20000208, 4, false, -1, 0, 0},
		{0x2000020c, 4, false, -1, 0, 0},
		{0x20000210, 4, false, -1, 0, 0},
		{0x20000214, 4, false, -1, 0, 0},
		{0x20000218, 4, false, -1, 0, 0},
		{0x2000021c, 4, false, -1, 0, 0},
		{0x20000220, 4, false, -1, 0, 0},
		{0x20000224, 4, false, -1, 0, 0},
		{0x20000228, 4, false, -1, 0, 0},
		{0x2000022c, 4, false, -1, 0, 0},
		{0x20000230, 4, false, -1, 0, 0},
		{0x20000234, 4, false, -1, 0, 0},
		{0x20000238, 4, false, -1, 0, 0},
		{0x2000023c, 4, false, -1, 0, 0},
		{0x20000240, 4, false, -1, 0, 0},
		{0x20000244, 4, false, -1, 0, 0},
		{0x20000248, 4, false, -1, 0, 0},
		{0x2000024c, 4, false, -1, 0, 0},
		{0x20000250, 4, false, -1, 0, 0},
		{0x20000254, 4, false, -1, 0, 0},
		{0x20000258, 4, false, -1, 0, 0},
		{0x2000025c, 4, false, -1, 0, 0},
		{0x20000260, 4, false, -1, 0, 0},
		{0x20000264, 4, false, -1, 0, 0},
		{0x20000268, 4, false, -1, 0, 0},
		{0x2000026c, 4, false, -1, 0, 0},
		{0x200003a0, 4, false, -1, 0, 0},
		{0x200003a4, 4, false, -1, 0, 0},
		{0x200003a8, 4, false, -1, 0, 0},
		{0x200003ac, 4, false, -1, 0, 0},
		{0x200003b0, 4, false, -1, 0, 0},
		{0x200003b4, 4, false, -1, 0, 0},
		{0x200003b8, 4, false, -1, 0, 0},
		{0x200003bc, 4, false, -1, 0, 0},
		{0x200003c0, 4, false, -1, 0, 0},
		{0x2000039c, 4, false, -1, 0, 0},
		{0x200000ac, 4, false, -1, 0, 0},
		{0x200000b0, 4, false, -1, 0, 0},
		{0x200000b4, 4, false, -1, 0, 0},
		{0x20000270, 4, false, -1, 0, 0},
		{0x20000274, 4, false, -1, 0, 0},
		{0x20000350, 4, false, -1, 0, 0},
		{0x20000354, 4, false, -1, 0, 0},
		{0x20000358, 4, false, -1, 0, 0},
		{0x2000035c, 4, false, -1, 0, 0},
		{0x20000360, 4, false, -1, 0, 0},
		{0x20000364, 4, false, -1, 0, 0},
		{0x20000368, 4, false, -1, 0, 0},
		{0x2000036c, 4, false, -1, 0, 0},
		{0x20000370, 4, false, -1, 0, 0},
		{0x20000374, 4, false, -1, 0, 0},
		{0x20000378, 4, false, -1, 0, 0},
		{0x2000037c, 4, false, -1, 0, 0},
		{0x20000380, 4, false, -1, 0, 0},
		{0x20000384, 4, false, -1, 0, 0},
		{0x20000388, 4, false, -1, 0, 0},
		{0x2000038c, 4, false, -1, 0, 0},
		{0x20000390, 4, false, -1, 0, 0},
		{0x20000394, 4, false, -1, 0, 0},
		{0x20000398, 4, false, -1, 0, 0},
	},
}

var ndsStubs = map[uint32]*mcaStub{
	0x4ef610: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200000b8, 1}, {2, 4, 0x200000bc, 1}}, cnt: 0x200000c0, wr: []mcaWrite{}},
	0x50b2f0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200000a0, 1}, {2, 4, 0x200000a4, 1}}, cnt: 0x200000a8, wr: []mcaWrite{{3, 0x0, 4, 0x20000000, 0}, {3, 0x4, 4, 0x20000004, 0}, {3, 0x8, 4, 0x20000008, 0}}},
	0x50f1a0: {ret: 0x20000010, retN: 0, rec: [][4]uint32{{1, 4, 0x200000c4, 1}, {2, 4, 0x200000c8, 1}, {4, 4, 0x200000cc, 1}, {5, 4, 0x200000d0, 1}, {6, 4, 0x200000d4, 1}}, cnt: 0x200000d8, wr: []mcaWrite{}},
	0x51d1c0: {ret: 0x20000020, retN: 12, rec: [][4]uint32{{2, 4, 0x20000278, 12}}, cnt: 0x200002a8, wr: []mcaWrite{}},
	0x522090: {ret: 0x20000050, retN: 20, rec: [][4]uint32{{1, 4, 0x200002ac, 20}, {2, 4, 0x200002fc, 20}}, cnt: 0x2000034c, wr: []mcaWrite{}},
	0x523c00: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200000dc, 20}, {2, 4, 0x2000012c, 20}, {3, 4, 0x2000017c, 20}, {4, 4, 0x200001cc, 20}, {6, 4, 0x2000021c, 20}}, cnt: 0x2000026c, wr: []mcaWrite{}},
	0x538e00: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200003a0, 8}}, cnt: 0x200003c0, wr: []mcaWrite{}},
	0x546e70: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x2000039c, wr: []mcaWrite{}},
	0x54a760: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x200000ac, wr: []mcaWrite{{1, 0x0, 4, 0x2000000c, 0}}},
	0x55a0a0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200000b0, 1}}, cnt: 0x200000b4, wr: []mcaWrite{}},
	0x562a50: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000270, 1}}, cnt: 0x20000274, wr: []mcaWrite{}},
	0x58aad0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20000350, 6}, {2, 4, 0x20000368, 6}, {3, 4, 0x20000380, 6}}, cnt: 0x20000398, wr: []mcaWrite{}},
}

var ndsInNames = []string{"idx", "skipSpawn", "joinGroup", "slotNpc", "vecPtr", "vecCount", "objModel", "objGone", "objKey", "objF1c", "objF26", "C.class", "C.cond0", "C.cond1", "C.cond2", "C.cond3", "C.cond4", "C.cond5", "C.cond6", "C.cond7", "C.cond8", "C.cond9", "C.cond10", "C.cond11", "C.cond12", "C.cond13", "C.cond14", "C.cond15", "C.cond16", "C.cond17", "C.cond18", "C.cond19", "C.penalty", "C.maxSP", "C.SP", "C.curseOverride", "C.equip0", "C.equipExempt0", "C.equip1", "C.equipExempt1", "C.equip2", "C.equipExempt2", "C.equip3", "C.equipExempt3", "C.equip4", "C.equipExempt4", "C.equip5", "C.equipExempt5", "C.equip6", "C.equipExempt6", "C.equip7", "C.equipExempt7", "C.equip8", "C.equipExempt8", "C.equip9", "C.equipExempt9", "C.equip10", "C.equipExempt10", "C.equip11", "C.equipExempt11", "C.pack0", "C.pack1", "C.pack2", "C.pack3", "C.pack4", "C.pack5", "C.pack6", "C.pack7", "N.keep0", "N.keep1", "N.keep2", "item0.kind", "item0.cursed", "item1.kind", "item1.cursed", "item2.kind", "item2.cursed", "item3.kind", "item3.cursed", "item4.kind", "item4.cursed", "item5.kind", "item5.cursed", "item6.kind", "item6.cursed", "item7.kind", "item7.cursed", "tmplCount", "tmpl0.flags", "tmpl0.key", "tmpl1.flags", "tmpl1.key", "tmpl2.flags", "tmpl2.key", "tmpl3.flags", "tmpl3.key", "T0.species", "T1.species", "T2.species", "T3.species", "nameChar", "gate", "ui.flags", "worldTime", "G0.leader", "G0.active", "G0.kind", "G0.named", "G0.template", "G0.link0", "G0.link1", "G0.link2", "G1.leader", "G1.active", "G2.leader", "G2.active", "M0.active", "M0.inCombat", "M0.morale", "M0.HP", "M0.SP", "M0.cond17", "M0.mainCond", "M0.f28a", "V0.f332", "A0.fa4", "A0.fa7", "M1.active", "M1.inCombat", "M1.morale", "M1.HP", "M1.SP", "M1.cond17", "M1.mainCond", "M1.f28a", "V1.f332", "A1.fa4", "A1.fa7", "M2.active", "M2.inCombat", "M2.morale", "M2.HP", "M2.SP", "M2.cond17", "M2.mainCond", "M2.f28a", "V2.f332", "A2.fa4", "A2.fa7", "M3.active", "M3.inCombat", "M3.morale", "M3.HP", "M3.SP", "M3.cond17", "M3.mainCond", "M3.f28a", "V3.f332", "A3.fa4", "A3.fa7", "pos0", "pos1", "pos2", "ret54a760", "ret50f1a0", "F.battleFlags", "F.animFrames", "F.unborn", "ret51d1c0_0", "ret51d1c0_1", "ret51d1c0_2", "ret51d1c0_3", "ret51d1c0_4", "ret51d1c0_5", "ret51d1c0_6", "ret51d1c0_7", "ret51d1c0_8", "ret51d1c0_9", "ret51d1c0_10", "ret51d1c0_11", "ret522090_0", "ret522090_1", "ret522090_2", "ret522090_3", "ret522090_4", "ret522090_5", "ret522090_6", "ret522090_7", "ret522090_8", "ret522090_9", "ret522090_10", "ret522090_11", "ret522090_12", "ret522090_13", "ret522090_14", "ret522090_15", "ret522090_16", "ret522090_17", "ret522090_18", "ret522090_19"}

var ndsOutNames = []string{"ret", "obj.model", "obj.f26", "obj.f1c", "obj.f1e", "obj.fe9", "slot0.npc", "slot0.f0", "char0.SP", "party0.f0", "party0.f114", "slot1.npc", "slot1.f0", "char1.SP", "party1.f0", "party1.f114", "slot2.npc", "slot2.f0", "char2.SP", "party2.f0", "party2.f114", "slot3.npc", "slot3.f0", "char3.SP", "party3.f0", "party3.f114", "slot4.npc", "slot4.f0", "char4.SP", "party4.f0", "party4.f114", "slot5.npc", "slot5.f0", "char5.SP", "party5.f0", "party5.f114", "slot6.npc", "slot6.f0", "char6.SP", "party6.f0", "party6.f114", "slot7.npc", "slot7.f0", "char7.SP", "party7.f0", "party7.f114", "copy.class", "copy.cond0", "copy.cond7", "copy.maxSP", "copy.SP", "copy.equip0", "copy.pack0", "M0.morale", "M0.SP", "M1.morale", "M1.SP", "M2.morale", "M2.SP", "M3.morale", "M3.SP", "G0.active", "G0.fca", "G0.fcb", "G1.active", "G1.fca", "G1.fcb", "G2.active", "G2.fca", "G2.fcb", "ui.flags", "name.0", "name.1", "s4ef610_a1", "s4ef610_a2", "s4ef610_calls", "s50b2f0_a1", "s50b2f0_a2", "s50b2f0_calls", "s50f1a0_a1", "s50f1a0_a2", "s50f1a0_a4", "s50f1a0_a5", "s50f1a0_a6", "s50f1a0_calls", "s51d1c0_a2_0", "s51d1c0_a2_1", "s51d1c0_a2_2", "s51d1c0_a2_3", "s51d1c0_a2_4", "s51d1c0_a2_5", "s51d1c0_a2_6", "s51d1c0_a2_7", "s51d1c0_a2_8", "s51d1c0_a2_9", "s51d1c0_a2_10", "s51d1c0_a2_11", "s51d1c0_calls", "s522090_a1_0", "s522090_a1_1", "s522090_a1_2", "s522090_a1_3", "s522090_a1_4", "s522090_a1_5", "s522090_a1_6", "s522090_a1_7", "s522090_a1_8", "s522090_a1_9", "s522090_a1_10", "s522090_a1_11", "s522090_a1_12", "s522090_a1_13", "s522090_a1_14", "s522090_a1_15", "s522090_a1_16", "s522090_a1_17", "s522090_a1_18", "s522090_a1_19", "s522090_a2_0", "s522090_a2_1", "s522090_a2_2", "s522090_a2_3", "s522090_a2_4", "s522090_a2_5", "s522090_a2_6", "s522090_a2_7", "s522090_a2_8", "s522090_a2_9", "s522090_a2_10", "s522090_a2_11", "s522090_a2_12", "s522090_a2_13", "s522090_a2_14", "s522090_a2_15", "s522090_a2_16", "s522090_a2_17", "s522090_a2_18", "s522090_a2_19", "s522090_calls", "s523c00_a1_0", "s523c00_a1_1", "s523c00_a1_2", "s523c00_a1_3", "s523c00_a1_4", "s523c00_a1_5", "s523c00_a1_6", "s523c00_a1_7", "s523c00_a1_8", "s523c00_a1_9", "s523c00_a1_10", "s523c00_a1_11", "s523c00_a1_12", "s523c00_a1_13", "s523c00_a1_14", "s523c00_a1_15", "s523c00_a1_16", "s523c00_a1_17", "s523c00_a1_18", "s523c00_a1_19", "s523c00_a2_0", "s523c00_a2_1", "s523c00_a2_2", "s523c00_a2_3", "s523c00_a2_4", "s523c00_a2_5", "s523c00_a2_6", "s523c00_a2_7", "s523c00_a2_8", "s523c00_a2_9", "s523c00_a2_10", "s523c00_a2_11", "s523c00_a2_12", "s523c00_a2_13", "s523c00_a2_14", "s523c00_a2_15", "s523c00_a2_16", "s523c00_a2_17", "s523c00_a2_18", "s523c00_a2_19", "s523c00_a3_0", "s523c00_a3_1", "s523c00_a3_2", "s523c00_a3_3", "s523c00_a3_4", "s523c00_a3_5", "s523c00_a3_6", "s523c00_a3_7", "s523c00_a3_8", "s523c00_a3_9", "s523c00_a3_10", "s523c00_a3_11", "s523c00_a3_12", "s523c00_a3_13", "s523c00_a3_14", "s523c00_a3_15", "s523c00_a3_16", "s523c00_a3_17", "s523c00_a3_18", "s523c00_a3_19", "s523c00_a4_0", "s523c00_a4_1", "s523c00_a4_2", "s523c00_a4_3", "s523c00_a4_4", "s523c00_a4_5", "s523c00_a4_6", "s523c00_a4_7", "s523c00_a4_8", "s523c00_a4_9", "s523c00_a4_10", "s523c00_a4_11", "s523c00_a4_12", "s523c00_a4_13", "s523c00_a4_14", "s523c00_a4_15", "s523c00_a4_16", "s523c00_a4_17", "s523c00_a4_18", "s523c00_a4_19", "s523c00_a6_0", "s523c00_a6_1", "s523c00_a6_2", "s523c00_a6_3", "s523c00_a6_4", "s523c00_a6_5", "s523c00_a6_6", "s523c00_a6_7", "s523c00_a6_8", "s523c00_a6_9", "s523c00_a6_10", "s523c00_a6_11", "s523c00_a6_12", "s523c00_a6_13", "s523c00_a6_14", "s523c00_a6_15", "s523c00_a6_16", "s523c00_a6_17", "s523c00_a6_18", "s523c00_a6_19", "s523c00_calls", "s538e00_a2_0", "s538e00_a2_1", "s538e00_a2_2", "s538e00_a2_3", "s538e00_a2_4", "s538e00_a2_5", "s538e00_a2_6", "s538e00_a2_7", "s538e00_calls", "s546e70_calls", "s54a760_calls", "s55a0a0_a1", "s55a0a0_calls", "s562a50_a1", "s562a50_calls", "s58aad0_a1_0", "s58aad0_a1_1", "s58aad0_a1_2", "s58aad0_a1_3", "s58aad0_a1_4", "s58aad0_a1_5", "s58aad0_a2_0", "s58aad0_a2_1", "s58aad0_a2_2", "s58aad0_a2_3", "s58aad0_a2_4", "s58aad0_a2_5", "s58aad0_a3_0", "s58aad0_a3_1", "s58aad0_a3_2", "s58aad0_a3_3", "s58aad0_a3_4", "s58aad0_a3_5", "s58aad0_calls"}
