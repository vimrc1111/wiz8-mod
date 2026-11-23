package calc

import "fmt"

// 0x523c00 monster_condition_apply(id, cond, duration, penalty, source, message)
// and its cycle partner 0x4eb7f0 (n_4eb7f0.go). Both members share one world
// layout (re/emu/cases/00523c00.json, 004eb7f0.json). Callees that are 排除 or
// whose writes are not read again inside the group are address stubs here and in
// the case; 0x52c240, 0x5323f0, 0x5049c0 and 0x4f0480 are D4 callees whose
// results are read later, so F runs their Go nodes and Emu replays the case stub.

// mcaWrite is one stub write: arg (1-based, 0 = absolute addr), addr (offset or
// absolute), size, the input it copies (at), and n (> 0: call k reads at+k*size).
type mcaWrite struct{ arg, addr, size, at, n uint32 }

// mcaStub mirrors emu.py's address stub: record (arg, size, at, n), count, ret
// ({at} or {at, n}: call k reads at+4k), write.
type mcaStub struct {
	ret, retN uint32
	rec       [][4]uint32
	cnt       uint32
	wr        []mcaWrite
}

type mcaWorld struct {
	*mstWorld
	real bool
	tab  map[uint32]*mcaStub // the case's address stubs; nil means mcaStubs
}

func (w *mcaWorld) stubAt(a uint32) *mcaStub {
	if w.tab != nil {
		return w.tab[a]
	}
	return mcaStubs[a]
}

const (
	mcaStrTable = 0x20003000 // fixture [0x68c09c]: string pointer table
	mcaStrCount = 920        // fixture cap: string ids 0..919
	mcaStrValue = 0x30000000 // entry i holds mcaStrValue + i
	mcaLocal    = 0x7e000000 // 0x52c070's stack local (0x536150 source record)
)

func mcaStrings(s mstSpec) mstSpec {
	for i := uint32(0); i < mcaStrCount; i++ {
		s.set = append(s.set, mstSet{mcaStrTable + 4*i, 4, mcaStrValue + i})
	}
	return s
}

// call records the arguments of call k and bumps the counter (emu.py order).
func (w *mcaWorld) call(a uint32, args ...uint32) uint32 {
	s := w.stubAt(a)
	k := w.u32(s.cnt)
	for _, r := range s.rec {
		if k >= r[3] {
			panic(fmt.Sprintf("monster condition apply: stub %#x called more than %d times (fixture record cap)", a, r[3]))
		}
		w.put(r[2]+k*r[1], int(r[1]), uint64(args[r[0]-1]))
	}
	w.put(s.cnt, 4, uint64(k+1))
	return k
}

// replay is the case stub's effect for call k: its writes and return value.
func (w *mcaWorld) replay(a, k uint32, args ...uint32) uint32 {
	s := w.stubAt(a)
	for _, x := range s.wr {
		at := x.at
		if x.n > 0 {
			if k >= x.n {
				panic(fmt.Sprintf("monster condition apply: stub %#x write past its %d values", a, x.n))
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
			panic(fmt.Sprintf("monster condition apply: stub %#x return past its %d values", a, s.retN))
		}
		return w.u32(s.ret + 4*k)
	case s.ret != 0:
		return w.u32(s.ret)
	}
	return 0
}

func (w *mcaWorld) stub(a uint32, args ...uint32) uint32 {
	return w.replay(a, w.call(a, args...), args...)
}

func (w *mcaWorld) u16(a uint32) uint32 { return uint32(w.u8(a)) | uint32(w.u8(a+1))<<8 }

// apply is 0x523c00.
func (w *mcaWorld) apply(id, cond, duration, penalty, src, message uint32) {
	if penalty != 0 && cond != 7 {
		panic("monster condition apply: penalty without condition 7 (0x523c15 asserts)")
	}
	if cond == 7 && penalty == 0 {
		return
	}
	dur := duration
	if cond-2 <= 0x11 { // 0x523c3c: jump table 0x523f1c indexed by byte table 0x523f24
		switch t := w.u32(0x523f1c + 4*uint32(w.u8(0x523f24+cond-2))); t {
		case 0x523c53:
			dur = 0x270f
		case 0x523c5a:
		default:
			panic(fmt.Sprintf("monster condition apply: unknown duration destination %#x", t))
		}
	}
	if dur == 0 {
		return
	}
	m := w.findMonster(id)
	kind := w.u8(w.template(m) + 0xcb)
	if w.u32(m+0x2b) == 0 {
		return
	}
	// 0x523ca2: immunity rows 0x6171aa..0x6172a0 (stride 0x52), kind byte at
	// row-2, twenty condition dwords.
	for row := uint32(0x6171aa); row < 0x6172a0; row += 0x52 {
		if w.u8(row-2) != kind {
			continue
		}
		for i := uint32(0); i < 20; i++ {
			if w.u32(row+4*i) == cond {
				return
			}
		}
	}
	if cond > 19 {
		panic("monster condition apply: condition past the 20 duration slots at M+0x57 (0x523cd0)")
	}
	slot := m + 0x57 + 4*cond
	old := w.u32(slot)
	changed := false
	if dur > old {
		w.put(slot, 4, uint64(dur))
		if old == 0 {
			switch cond {
			case 9, 12:
				w.awareness(m)
			case 13:
				b := w.u8(m + 0x16)
				if b == 0 { // 0x523cfc clears the slot just written
					w.put(m+0x8b, 4, 0)
					return
				}
				d := byte(1)
				if b == 1 {
					d = 2
				}
				w.morale(m, d)
			}
		}
		top := uint32(0) // 0x523d24: highest set slot 19..1
		for i := uint32(19); i > 0; i-- {
			if w.u32(m+0x57+4*i) != 0 {
				top = i
				break
			}
		}
		w.put(m+0x107, 4, uint64(top))
		w.groupCount(w.group(w.u32(m + 4)))
		if w.u8(m+0x14) != 0 {
			w.visual(m, w.u32(m+0x107) >= 0xe)
		}
		if old == 0 && cond >= 1 && cond <= 0x12 {
			w.stub(0x4acd80, w.u32(m+0xc), cond-1, 1)
		}
		if w.u8(m+0x15) != 0 && w.isChar(src) {
			ch := w.u32(src + 4)
			if ch >= mstMaxChar {
				panic("monster condition apply: source character index past 7 (fixture cap)")
			}
			p := w.u32(m+0x10) + 0x1a + 4*ch
			w.put(p, 4, uint64(w.u32(p)+uint32(w.u8(w.template(m)+0x24f))*cond/3))
		}
		changed = true
	}
	if (w.isChar(src) || w.isMon(src)) && w.u8(src+0x1c) == 0 && w.u8(src+0x1b) == 0 && w.u8(src+0x1e) == 0 {
		for i := uint32(0); i < 13; i++ { // 0x523e41: copy the source record
			w.put(m+0x304+4*i, 4, uint64(w.u32(src+4*i)))
		}
	}
	if penalty > w.u32(m+0x10b) {
		w.put(m+0x10b, 4, uint64(penalty))
		changed = true
	}
	w.rebuild(id)
	if !changed {
		return
	}
	if cond >= 13 { // 0x536170
		for i := uint32(0); i < 8; i++ {
			w.put(m+0x2ba+4*i, 4, 0)
		}
		for i := uint32(1); i < 4; i++ {
			w.put(m+0x2ba+4*i, 4, 0xffffffff)
		}
	}
	if cond >= 0x12 {
		w.downed(m, message)
		return
	}
	if byte(message) != 0 && (w.u8(0x683f94) != 0 || w.u8(m+0x2ab) != 0) {
		str := w.u32(w.u32(0x68c09c) + 4*w.u16(0x61e572+8*cond))
		w.stub(0x58aad0, 9, 0x614a3c, w.name(m), str)
	}
	a7 := w.u8(w.u32(w.u32(m+0xc)+0x1d8) + 0xa7)
	if byte(w.stub(0x4c2cf0, src&^0xff|uint32(a7))) != 0 {
		w.stub(0x4e4db0, m, 0x14, 1)
	}
}

// isChar is 0x53bea0(x, 0): type 1 is a character; index -1 asserts (0x53beb1).
func (w *mcaWorld) isChar(x uint32) bool {
	if w.u32(x) != 1 {
		return false
	}
	if w.u32(x+4) == 0xffffffff {
		panic("monster condition apply: character source without index (0x53beb1 asserts)")
	}
	return true
}

// isMon is 0x53bf10(x, 0): type 2 is a monster; id -1 asserts (0x53bf21).
func (w *mcaWorld) isMon(x uint32) bool {
	if w.u32(x) != 2 {
		return false
	}
	if w.u32(x+8) == 0xffffffff {
		panic("monster condition apply: monster source without id (0x53bf21 asserts)")
	}
	return true
}

// group is 0x5100b0 (assert flag) then 0x5101b0: the group record whose +0 is
// gid, list 0x683fb1 first, then list 0x683fbd.
func (w *mcaWorld) group(gid uint32) uint32 {
	for _, l := range [2]uint32{w.u32(0x683fb1), w.u32(0x683fbd)} {
		for i := uint32(0); i < w.listCount(l); i++ {
			g := w.listAt(l, i)
			if g == 0 {
				panic("monster condition apply: null group record")
			}
			if w.u32(g) == gid {
				return g
			}
		}
	}
	panic("monster condition apply: group id not found (0x510168 asserts)")
}

// groupCount is 0x510350: members with main condition < 13 and +0x2fd != 1;
// a change sets UI flag 0x20000 (0x565420).
func (w *mcaWorld) groupCount(g uint32) {
	l := w.u32(g + 8)
	n := uint32(0)
	for i := uint32(0); i < w.listCount(l); i++ {
		m := w.findMonster(w.listAt(l, i))
		if w.u32(m+0x107) < 13 && w.u32(m+0x2fd) != 1 {
			n++
		}
	}
	if n == w.u32(g+0x14) {
		return
	}
	w.put(g+0x14, 4, uint64(n))
	if ui := w.u32(0x68edcc); w.u32(0x68ec78) == 7 && ui != 0 {
		w.put(ui+0xf4, 4, uint64(w.u32(ui+0xf4)|0x20000))
	}
}

// act15 is 0x4ca4c0: the actor's state byte (+0xa4 signed, or +0xa7) is 0x15.
func (w *mcaWorld) act15(vis uint32) bool {
	a := w.u32(vis + 0x1d8)
	return int8(w.u8(a+0xa4)) == 0x15 || w.u8(a+0xa7) == 0x15
}

// name is 0x4e5150(m, 0, 0): the display name pointer.
func (w *mcaWorld) name(m uint32) uint32 {
	t := w.template(m)
	if w.u16(t+0x187) == 0x18d {
		return 0x6875c3
	}
	if w.u32(m+4) == 0 {
		// 0x4ca4c0 false logs through 0x5182e0 (排除, no state); the pointer is the same.
		return t + 0x60
	}
	if w.u8(w.group(w.u32(m+4))+0x2c) == 0 {
		return t + 0x60
	}
	return t
}

// visual is 0x4e60b0: M+0x24e follows the flag; the animation calls are 排除.
func (w *mcaWorld) visual(m uint32, flag bool) {
	was := w.u8(m + 0x24e)
	w.put(m+0x24e, 1, uint64(mstB(flag)))
	vis := w.u32(m + 0xc)
	switch {
	case !flag && was != 0:
		w.stub(0x4c5a00, vis, 1)
		if w.u8(w.u32(w.u32(m+0xc)+0x1d8)+0xa7) == 0xff {
			w.stub(0x4e4db0, m, 1, 3)
		}
	case flag && was == 0:
		switch w.u8(w.u32(vis+0x1d8) + 0xa7) {
		case 0xff:
			w.stub(0x4c5a00, vis, 0)
		case 0x14:
		default:
			w.stub(0x4c5aa0, vis, 0xffffffff)
			w.stub(0x4c5a00, vis, 0)
		}
	}
}

// downed is 0x4e4690.
func (w *mcaWorld) downed(m, message uint32) {
	if w.act15(w.u32(m + 0xc)) {
		return
	}
	w.stub(0x4e4db0, m, 0x15, 1)
	w.stub(0x4e4280, m)
	w.stub(0x4e46f0, m, message)
	h, ok := w.monsterHandle(w.u32(m))
	if !ok {
		panic("monster condition apply: monster id not found (0x4e5550 asserts)")
	}
	w.stub(0x4e3af0, h, 0)
}

// morale is 0x5477d0.
func (w *mcaWorld) morale(m uint32, d byte) {
	old := w.u8(m + 0x16)
	if old == d || w.u8(w.u32(m+0xc)+0x332) != 0 {
		return
	}
	w.put(m+0x16, 1, uint64(d))
	if w.u8(m+0x15) != 0 && (d == 1 || old == 1) {
		w.stub(0x546e70)
	}
	if old != 0 { // 0x538d60(id, 1): local relation record, then 0x538e00(local, 1)
		w.stub(0x538e00, 0, 1)
	}
	if w.u8(m+0x15) == 0 || w.u32(m+0x2b) == 0 || w.u8(m+0x16) == 0 {
		return
	}
	g := w.u32(0x6836a8)
	if w.u8(0x683f94) != 0 && w.u32(g+0x7b0) != 1 && w.u32(g+0x7b8) == m {
		w.nextPhase(m)
		w.put(m+0x2e1, 4, 0xffffffff)
		w.put(w.u32(m+0x10), 4, 0)
		return
	}
	t := w.u32(m + 0x10)
	if w.u8(t+4) == 0 {
		w.stub(0x531540, m)
		return
	}
	w.put(m+0x2e1, 4, 0xffffffff)
	w.put(t, 4, 0)
}

// fatigue is 0x52c070(m, cost, 0): 0x4eb7f0 leaves 0 as the third argument.
func (w *mcaWorld) fatigue(m, cost uint32) {
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
		panic("monster condition apply: max SP 0 divides by zero (0x52c0b8)")
	}
	w.put(m+0x242, 4, uint64(staminaFatigueLevel(sp, most)))
	if sp == 0 && w.u32(m+0x9b) < 0x270f {
		for i := uint32(0); i < 13; i++ { // 0x536150
			w.put(mcaLocal+4*i, 4, 0)
		}
		w.put(mcaLocal+4, 4, 0xffffffff)
		w.put(mcaLocal+8, 4, 0xffffffff)
		w.apply(w.u32(m), 0x11, 0x270f, 0, mcaLocal, 1) // message = (third arg == 0)
	}
}

// rebuild is 0x50e8c0: the condition rules (0x50eac0/0x50ecc0/0x50edc0 via
// monsterActivationRebuildRules), the in-combat record deltas, attributes and
// recovery floats.
func (w *mcaWorld) rebuild(id uint32) {
	m := w.findMonster(id)
	monsterCacheRebuild(w.u8, w.u32, w.put, m, w.template(m))
}

// monsterCacheRebuild is 0x50e8c0's body on monster record m with template t
// (shared with 0x5500c0, n_5500c0.go).
func monsterCacheRebuild(u8 func(uint32) byte, u32 func(uint32) uint32, put func(uint32, int, uint64), m, t uint32) {
	u16 := func(a uint32) uint32 { return uint32(u8(a)) | uint32(u8(a+1))<<8 }
	if u32(m+0xa3) != 0 {
		panic("monster condition apply: status 19 dereferences a null character (0x50eac0 -> 0x517ec0)")
	}
	in, out := make([]int64, 262), make([]int64, 130)
	in[4], in[165], in[186] = int64(u32(m+0x27)), int64(u32(m+0x2f)), int64(u32(m+0x10b))
	for i := uint32(0); i < 5; i++ {
		in[153+i] = int64(u8(t + 0xd1 + i))
	}
	for i := uint32(0); i < 20; i++ {
		in[166+i] = int64(u32(m + 0x57 + 4*i))
	}
	for i := uint32(0); i < 8; i++ {
		b := m + 0xa7 + 12*i
		in[187+3*i], in[188+3*i], in[189+3*i] = int64(u8(b)), int64(u16(b+4)), int64(u32(b+8))
	}
	for i := uint32(0); i < 12; i++ {
		e := m + 0x10f + 0x11*i
		in[211+4*i], in[212+4*i], in[213+4*i], in[214+4*i] = int64(u8(e)), int64(u32(e+1)), int64(u8(e+5)), int64(u32(e+9))
	}
	monsterActivationRebuildRules(in, out)
	var mod [103]uint8
	for i := range mod {
		mod[i] = uint8(out[20+i])
	}
	if u8(m+0x15) != 0 {
		c := u32(m + 0x10)
		for i := uint32(0); i < 9; i++ { // 0x50e92c
			r := c + 0x3e + 0x11*i
			if u8(r) != 0 && u32(r+1) == 0x31 {
				mod[5] -= u8(r + 5)
			}
		}
		// 0x517f60
		scale := func(v uint8, mult uint32) uint8 { return uint8((uint32(v)*(mult+100) + 50) / 100) }
		for i := uint32(0); i < 6; i++ { // 0x50ef50: aura records
			r := c + 0xd7 + 0x11*i
			if u8(r) == 0 {
				continue
			}
			power, mult := u8(r+5), u32(r+9)
			switch u32(r + 1) {
			case 2:
				v := scale(2, mult)
				mod[4] += v
				mod[1] += v
			case 0x35:
				v := scale(power*7, mult)
				for j := 0x3c; j < 0x40; j++ {
					mod[j] += v
				}
			case 0x3b:
				v := scale(power*10, mult)
				mod[0x40] += v
				mod[0x41] += v
			}
		}
	}
	for i, v := range mod {
		put(m+0x1db+uint32(i), 1, uint64(v))
	}
	for i := uint32(0); i < 5; i++ {
		put(m+0x247+i, 1, uint64(out[123+i]))
	}
	put(m+0x47, 4, uint64(out[128]))
	put(m+0x4f, 4, uint64(out[129]))
}

// awareness is 0x505760: 0x5049c0(m, 1, 0) then 0x5049c0(m, 0, 0). The real
// 0x5049c0 runs with arg2 1 on its parked world (re/emu/cases/005049c0.json first
// arg2 == 1 boundary row) with this monster's fields and template bytes; only
// M+0x2ab returns.
func (w *mcaWorld) awareness(m uint32) {
	for _, a2 := range [2]uint32{1, 0} {
		k := w.call(0x5049c0, m, a2, 0)
		if !w.real {
			w.replay(0x5049c0, k, m, a2, 0)
			continue
		}
		if a2 != 1 || w.u32(0x68ec78) != 7 || w.u8(m+0x14) == 0 || w.u32(m+0x2b) == 0 {
			continue
		}
		live := map[string]int64{"gate": 7, "arg2": 1, "arg3": 0, "s14": int64(w.u8(m + 0x14)), "s15": int64(w.u8(m + 0x15)),
			"s16": int64(w.u8(m + 0x16)), "s2b": int64(w.u32(m + 0x2b)), "s87": int64(w.u32(m + 0x87)), "s107": int64(w.u32(m + 0x107))}
		x := append([]int64(nil), mcaVauParked...)
		tid := int64(-1) // the parked group's template slot (+8) takes this monster's template bytes
		for i, f := range vauIn {
			if f.name == "s8" {
				tid = x[i]
			}
		}
		t := w.template(m)
		for _, off := range []uint32{0xcb, 0xcd, 0xd0, 0x248, 0x24f, 0x25f} {
			live[fmt.Sprintf("t%d_%x", tid, off)] = int64(w.u8(t + off))
		}
		for i, f := range vauIn {
			if v, ok := live[f.name]; ok {
				x[i] = v
			}
		}
		o := visibilityAwarenessUpdate(x, w.r, true)
		for i, f := range vauOut {
			if f.name == "s2ab" {
				w.put(m+0x2ab, 1, uint64(o[i]))
			}
		}
	}
}

// cost is 0x52c240(m, 0).
func (w *mcaWorld) cost(m uint32) uint32 {
	k := w.call(0x52c240, m, 0)
	if !w.real {
		return w.replay(0x52c240, k, m, 0)
	}
	in := []int64{int64(int32(w.u32(m + 0x2e1))), int64(int32(w.u32(m + 0x2e5))), int64(w.u32(m + 0x5f))}
	return uint32(monsterActionFatigueCostNode(in, w.r)[0])
}

// pick is 0x5323f0(m, 1, 0, 0) on its parked world (mchParkPick), as mchReal.pick.
func (w *mcaWorld) pick(m uint32) bool { return w.pickWith(m, 0) }

// pickWith is 0x5323f0(m, 1, 0, flag).
func (w *mcaWorld) pickWith(m, flag uint32) bool {
	k := w.call(0x5323f0, m, 1, 0, flag)
	if !w.real {
		return byte(w.replay(0x5323f0, k, m, 1, 0, flag)) != 0
	}
	c := w.u32(m + 0x10)
	x := append([]int64(nil), mchParkPick[:]...)
	x[0], x[1], x[2], x[3] = 1, 0, int64(flag), 0
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

// round is 0x4f0480(phase, [0x6836a8]+8).
func (w *mcaWorld) round(t uint32) {
	g := w.u32(0x6836a8)
	k := w.call(0x4f0480, t, w.u32(g+8))
	if !w.real {
		w.replay(0x4f0480, k, t, w.u32(g+8))
		return
	}
	o := combatPhaseRoundingNode([]int64{int64(w.u32(g + 0x910)), int64(w.u32(t)), int64(w.u32(g + 8))}, w.r)
	w.put(t, 4, uint64(o[0]))
}

func mcaRun(spec *mstSpec, real bool, body func(w *mcaWorld)) func([]int64, Rand) []int64 {
	return func(in []int64, r Rand) []int64 {
		if len(in) != len(spec.in) {
			panic("monster condition apply: wrong input count")
		}
		w := &mcaWorld{mstWorld: mstBuild(spec, in), real: real}
		w.r = r
		body(w)
		out := make([]int64, len(spec.out))
		for i, f := range spec.out {
			out[i] = w.read(f, in)
		}
		return out
	}
}

func mcaArg(w *mcaWorld, n uint32) uint32 { return w.u32(mstStackBase + 4 + 4*n) }

func mcaApplyBody(w *mcaWorld) {
	w.apply(mcaArg(w, 0), mcaArg(w, 1), mcaArg(w, 2), mcaArg(w, 3), mcaArg(w, 4), mcaArg(w, 5))
}

func monsterConditionApplyNode(in []int64, r Rand) []int64 {
	return mcaRun(&mcaSpecApply, true, mcaApplyBody)(in, r)
}

func mcaEmuApply(in []int64, r Rand) []int64 {
	return mcaRun(&mcaSpecApply, false, mcaApplyBody)(in, r)
}

func init() {
	Nodes["00523c00"] = Node{Addr: "00523c00", In: mcaSpecApplyIn, Out: mcaOutNames, F: monsterConditionApplyNode, Emu: mcaEmuApply}
}

// Code below is generated from the same description as re/emu/cases/00523c00.json.

var mcaSpecApply = mcaStrings(mstSpec{
	set: []mstSet{
		{0x7f000004, 4, 0x5},
		{0x7f000014, 4, 0x20001c00},
		{0x683fad, 4, 0x20001b00},
		{0x683fb9, 4, 0x20001b20},
		{0x683fb1, 4, 0x20001b40},
		{0x683fbd, 4, 0x20001b60},
		{0x6840c7, 4, 0x20000800},
		{0x6836a8, 4, 0x20004000},
		{0x68edcc, 4, 0x20001a00},
		{0x68c09c, 4, 0x20003000},
		{0x685174, 4, 0x20001d00},
		{0x20000000, 4, 0x5},
		{0x20000004, 4, 0x3},
		{0x20000008, 4, 0x0},
		{0x2000000c, 4, 0x20000b00},
		{0x20000010, 4, 0x20000f40},
		{0x20000400, 4, 0x7},
		{0x20000404, 4, 0x3},
		{0x20000408, 4, 0x0},
		{0x2000040c, 4, 0x20000b00},
		{0x20000410, 4, 0x20000f40},
		{0x20000cd8, 4, 0x20000e40},
		{0x200010c0, 4, 0x3},
		{0x200010c8, 4, 0x20001100},
		{0x20001100, 4, 0x20001110},
		{0x20001108, 4, 0x2},
		{0x20001110, 4, 0x5},
		{0x20001114, 4, 0x7},
		{0x20001b00, 4, 0x20001b10},
		{0x20001b08, 4, 0x2},
		{0x20001b10, 4, 0x20000000},
		{0x20001b14, 4, 0x20000400},
		{0x20001b20, 4, 0x20001b30},
		{0x20001b28, 4, 0x0},
		{0x20001b40, 4, 0x20001b50},
		{0x20001b48, 4, 0x1},
		{0x20001b50, 4, 0x200010c0},
		{0x20001b60, 4, 0x20001b70},
		{0x20001b68, 4, 0x0},
		{0x20001d05, 4, 0x41},
		{0x20000304, 4, 0xd0d00000},
		{0x20000308, 4, 0xd0d00001},
		{0x2000030c, 4, 0xd0d00002},
		{0x20000310, 4, 0xd0d00003},
		{0x20000314, 4, 0xd0d00004},
		{0x20000318, 4, 0xd0d00005},
		{0x2000031c, 4, 0xd0d00006},
		{0x20000320, 4, 0xd0d00007},
		{0x20000324, 4, 0xd0d00008},
		{0x20000328, 4, 0xd0d00009},
		{0x2000032c, 4, 0xd0d0000a},
		{0x20000330, 4, 0xd0d0000b},
		{0x20000334, 4, 0xd0d0000c},
		{0x200001db, 1, 0x80},
		{0x200001dc, 1, 0x81},
		{0x200001dd, 1, 0x82},
		{0x200001de, 1, 0x83},
		{0x200001df, 1, 0x84},
		{0x200001e0, 1, 0x85},
		{0x200001e1, 1, 0x86},
		{0x200001e2, 1, 0x87},
		{0x200001e3, 1, 0x88},
		{0x200001e4, 1, 0x89},
		{0x200001e5, 1, 0x8a},
		{0x200001e6, 1, 0x8b},
		{0x200001e7, 1, 0x8c},
		{0x200001e8, 1, 0x8d},
		{0x200001e9, 1, 0x8e},
		{0x200001ea, 1, 0x8f},
		{0x200001eb, 1, 0x90},
		{0x200001ec, 1, 0x91},
		{0x200001ed, 1, 0x92},
		{0x200001ee, 1, 0x93},
		{0x200001ef, 1, 0x94},
		{0x200001f0, 1, 0x95},
		{0x200001f1, 1, 0x96},
		{0x200001f2, 1, 0x97},
		{0x200001f3, 1, 0x98},
		{0x200001f4, 1, 0x99},
		{0x200001f5, 1, 0x9a},
		{0x200001f6, 1, 0x9b},
		{0x200001f7, 1, 0x9c},
		{0x200001f8, 1, 0x9d},
		{0x200001f9, 1, 0x9e},
		{0x200001fa, 1, 0x9f},
		{0x200001fb, 1, 0xa0},
		{0x200001fc, 1, 0xa1},
		{0x200001fd, 1, 0xa2},
		{0x200001fe, 1, 0xa3},
		{0x200001ff, 1, 0xa4},
		{0x20000200, 1, 0xa5},
		{0x20000201, 1, 0xa6},
		{0x20000202, 1, 0xa7},
		{0x20000203, 1, 0xa8},
		{0x20000204, 1, 0xa9},
		{0x20000205, 1, 0xaa},
		{0x20000206, 1, 0xab},
		{0x20000207, 1, 0xac},
		{0x20000208, 1, 0xad},
		{0x20000209, 1, 0xae},
		{0x2000020a, 1, 0xaf},
		{0x2000020b, 1, 0xb0},
		{0x2000020c, 1, 0xb1},
		{0x2000020d, 1, 0xb2},
		{0x2000020e, 1, 0xb3},
		{0x2000020f, 1, 0xb4},
		{0x20000210, 1, 0xb5},
		{0x20000211, 1, 0xb6},
		{0x20000212, 1, 0xb7},
		{0x20000213, 1, 0xb8},
		{0x20000214, 1, 0xb9},
		{0x20000215, 1, 0xba},
		{0x20000216, 1, 0xbb},
		{0x20000217, 1, 0xbc},
		{0x20000218, 1, 0xbd},
		{0x20000219, 1, 0xbe},
		{0x2000021a, 1, 0xbf},
		{0x2000021b, 1, 0xc0},
		{0x2000021c, 1, 0xc1},
		{0x2000021d, 1, 0xc2},
		{0x2000021e, 1, 0xc3},
		{0x2000021f, 1, 0xc4},
		{0x20000220, 1, 0xc5},
		{0x20000221, 1, 0xc6},
		{0x20000222, 1, 0xc7},
		{0x20000223, 1, 0xc8},
		{0x20000224, 1, 0xc9},
		{0x20000225, 1, 0xca},
		{0x20000226, 1, 0xcb},
		{0x20000227, 1, 0xcc},
		{0x20000228, 1, 0xcd},
		{0x20000229, 1, 0xce},
		{0x2000022a, 1, 0xcf},
		{0x2000022b, 1, 0xd0},
		{0x2000022c, 1, 0xd1},
		{0x2000022d, 1, 0xd2},
		{0x2000022e, 1, 0xd3},
		{0x2000022f, 1, 0xd4},
		{0x20000230, 1, 0xd5},
		{0x20000231, 1, 0xd6},
		{0x20000232, 1, 0xd7},
		{0x20000233, 1, 0xd8},
		{0x20000234, 1, 0xd9},
		{0x20000235, 1, 0xda},
		{0x20000236, 1, 0xdb},
		{0x20000237, 1, 0xdc},
		{0x20000238, 1, 0xdd},
		{0x20000239, 1, 0xde},
		{0x2000023a, 1, 0xdf},
		{0x2000023b, 1, 0xe0},
		{0x2000023c, 1, 0xe1},
		{0x2000023d, 1, 0xe2},
		{0x2000023e, 1, 0xe3},
		{0x2000023f, 1, 0xe4},
		{0x20000240, 1, 0xe5},
		{0x20000241, 1, 0xe6},
		{0x20000247, 1, 0xee},
		{0x20000248, 1, 0xed},
		{0x20000249, 1, 0xec},
		{0x2000024a, 1, 0xeb},
		{0x2000024b, 1, 0xea},
		{0x20000047, 4, 0xdecade01},
		{0x2000004f, 4, 0xdecade02},
		{0x20000242, 4, 0xdecade03},
		{0x200002ba, 4, 0xdecad100},
		{0x200002be, 4, 0xdecad101},
		{0x200002c2, 4, 0xdecad102},
		{0x200002c6, 4, 0xdecad103},
		{0x200002ca, 4, 0xdecad104},
		{0x200002ce, 4, 0xdecad105},
		{0x200002d2, 4, 0xdecad106},
		{0x200002d6, 4, 0xdecad107},
		{0x20001c00, 4, 0xa5a5a500},
		{0x20001c04, 4, 0xa5a5a501},
		{0x20001c08, 4, 0xa5a5a502},
		{0x20001c0c, 4, 0xa5a5a503},
		{0x20001c10, 4, 0xa5a5a504},
		{0x20001c14, 4, 0xa5a5a505},
		{0x20001c18, 4, 0xa5a5a506},
		{0x20001c1c, 4, 0xa5a5a507},
		{0x20001c20, 4, 0xa5a5a508},
		{0x20001c24, 4, 0xa5a5a509},
		{0x20001c28, 4, 0xa5a5a50a},
		{0x20001c2c, 4, 0xa5a5a50b},
		{0x20001c30, 4, 0xa5a5a50c},
	},
	in: []mstFld{
		{addr: 0x7f000008, size: 4, signed: false, idx: -1},
		{addr: 0x7f00000c, size: 4, signed: false, idx: -1},
		{addr: 0x7f000010, size: 4, signed: false, idx: -1},
		{addr: 0x7f000018, size: 4, signed: false, idx: -1},
		{addr: 0x20000014, size: 1, signed: false, idx: -1},
		{addr: 0x20000015, size: 1, signed: false, idx: -1},
		{addr: 0x20000016, size: 1, signed: false, idx: -1},
		{addr: 0x20000027, size: 4, signed: false, idx: -1},
		{addr: 0x2000002b, size: 4, signed: false, idx: -1},
		{addr: 0x2000002f, size: 4, signed: false, idx: -1},
		{addr: 0x20000033, size: 4, signed: false, idx: -1},
		{addr: 0x20000057, size: 4, signed: false, idx: -1},
		{addr: 0x2000005b, size: 4, signed: false, idx: -1},
		{addr: 0x2000005f, size: 4, signed: false, idx: -1},
		{addr: 0x20000063, size: 4, signed: false, idx: -1},
		{addr: 0x20000067, size: 4, signed: false, idx: -1},
		{addr: 0x2000006b, size: 4, signed: false, idx: -1},
		{addr: 0x2000006f, size: 4, signed: false, idx: -1},
		{addr: 0x20000073, size: 4, signed: false, idx: -1},
		{addr: 0x20000077, size: 4, signed: false, idx: -1},
		{addr: 0x2000007b, size: 4, signed: false, idx: -1},
		{addr: 0x2000007f, size: 4, signed: false, idx: -1},
		{addr: 0x20000083, size: 4, signed: false, idx: -1},
		{addr: 0x20000087, size: 4, signed: false, idx: -1},
		{addr: 0x2000008b, size: 4, signed: false, idx: -1},
		{addr: 0x2000008f, size: 4, signed: false, idx: -1},
		{addr: 0x20000093, size: 4, signed: false, idx: -1},
		{addr: 0x20000097, size: 4, signed: false, idx: -1},
		{addr: 0x2000009b, size: 4, signed: false, idx: -1},
		{addr: 0x2000009f, size: 4, signed: false, idx: -1},
		{addr: 0x200000a3, size: 4, signed: false, idx: -1},
		{addr: 0x200000e3, size: 1, signed: false, idx: -1},
		{addr: 0x200000e7, size: 2, signed: false, idx: -1},
		{addr: 0x200000eb, size: 4, signed: false, idx: -1},
		{addr: 0x200000ef, size: 1, signed: false, idx: -1},
		{addr: 0x200000f3, size: 2, signed: false, idx: -1},
		{addr: 0x200000f7, size: 4, signed: false, idx: -1},
		{addr: 0x200000fb, size: 1, signed: false, idx: -1},
		{addr: 0x200000ff, size: 2, signed: false, idx: -1},
		{addr: 0x20000103, size: 4, signed: false, idx: -1},
		{addr: 0x20000107, size: 4, signed: false, idx: -1},
		{addr: 0x2000010b, size: 4, signed: false, idx: -1},
		{addr: 0x2000010f, size: 1, signed: false, idx: -1},
		{addr: 0x20000110, size: 4, signed: false, idx: -1},
		{addr: 0x20000114, size: 1, signed: false, idx: -1},
		{addr: 0x20000118, size: 4, signed: false, idx: -1},
		{addr: 0x20000164, size: 1, signed: false, idx: -1},
		{addr: 0x20000165, size: 4, signed: false, idx: -1},
		{addr: 0x20000169, size: 1, signed: false, idx: -1},
		{addr: 0x2000016d, size: 4, signed: false, idx: -1},
		{addr: 0x200001ca, size: 1, signed: false, idx: -1},
		{addr: 0x200001cb, size: 4, signed: false, idx: -1},
		{addr: 0x200001cf, size: 1, signed: false, idx: -1},
		{addr: 0x200001d3, size: 4, signed: false, idx: -1},
		{addr: 0x2000024e, size: 1, signed: false, idx: -1},
		{addr: 0x200002ab, size: 1, signed: false, idx: -1},
		{addr: 0x200002e1, size: 4, signed: true, idx: -1},
		{addr: 0x200002e5, size: 4, signed: false, idx: -1},
		{addr: 0x200002fd, size: 4, signed: false, idx: -1},
		{addr: 0x20000507, size: 4, signed: false, idx: -1},
		{addr: 0x200006fd, size: 4, signed: false, idx: -1},
		{addr: 0x200008cb, size: 1, signed: false, idx: -1},
		{addr: 0x200008d1, size: 1, signed: false, idx: -1},
		{addr: 0x200008d2, size: 1, signed: false, idx: -1},
		{addr: 0x200008d3, size: 1, signed: false, idx: -1},
		{addr: 0x200008d4, size: 1, signed: false, idx: -1},
		{addr: 0x200008d5, size: 1, signed: false, idx: -1},
		{addr: 0x20000987, size: 2, signed: false, idx: -1},
		{addr: 0x20000a4f, size: 1, signed: false, idx: -1},
		{addr: 0x200008e5, size: 1, signed: false, idx: -1},
		{addr: 0x20000e32, size: 1, signed: false, idx: -1},
		{addr: 0x20000ee4, size: 1, signed: false, idx: -1},
		{addr: 0x20000ee7, size: 1, signed: false, idx: -1},
		{addr: 0x20000f40, size: 4, signed: false, idx: -1},
		{addr: 0x20000f44, size: 1, signed: false, idx: -1},
		{addr: 0x20000f45, size: 4, signed: false, idx: -1},
		{addr: 0x20000f49, size: 4, signed: false, idx: -1},
		{addr: 0x20000f51, size: 4, signed: false, idx: -1},
		{addr: 0x20000f5a, size: 4, signed: false, idx: -1},
		{addr: 0x20000f5e, size: 4, signed: false, idx: -1},
		{addr: 0x20000f62, size: 4, signed: false, idx: -1},
		{addr: 0x20000f66, size: 4, signed: false, idx: -1},
		{addr: 0x20000f6a, size: 4, signed: false, idx: -1},
		{addr: 0x20000f6e, size: 4, signed: false, idx: -1},
		{addr: 0x20000f72, size: 4, signed: false, idx: -1},
		{addr: 0x20000f76, size: 4, signed: false, idx: -1},
		{addr: 0x20000f7e, size: 1, signed: false, idx: -1},
		{addr: 0x20000f7f, size: 4, signed: false, idx: -1},
		{addr: 0x20000f83, size: 1, signed: false, idx: -1},
		{addr: 0x20000f8f, size: 1, signed: false, idx: -1},
		{addr: 0x20000f90, size: 4, signed: false, idx: -1},
		{addr: 0x20000f94, size: 1, signed: false, idx: -1},
		{addr: 0x20000fa0, size: 1, signed: false, idx: -1},
		{addr: 0x20000fa1, size: 4, signed: false, idx: -1},
		{addr: 0x20000fa5, size: 1, signed: false, idx: -1},
		{addr: 0x20000fb1, size: 1, signed: false, idx: -1},
		{addr: 0x20000fb2, size: 4, signed: false, idx: -1},
		{addr: 0x20000fb6, size: 1, signed: false, idx: -1},
		{addr: 0x20000fc2, size: 1, signed: false, idx: -1},
		{addr: 0x20000fc3, size: 4, signed: false, idx: -1},
		{addr: 0x20000fc7, size: 1, signed: false, idx: -1},
		{addr: 0x20000fd3, size: 1, signed: false, idx: -1},
		{addr: 0x20000fd4, size: 4, signed: false, idx: -1},
		{addr: 0x20000fd8, size: 1, signed: false, idx: -1},
		{addr: 0x20000fe4, size: 1, signed: false, idx: -1},
		{addr: 0x20000fe5, size: 4, signed: false, idx: -1},
		{addr: 0x20000fe9, size: 1, signed: false, idx: -1},
		{addr: 0x20000ff5, size: 1, signed: false, idx: -1},
		{addr: 0x20000ff6, size: 4, signed: false, idx: -1},
		{addr: 0x20000ffa, size: 1, signed: false, idx: -1},
		{addr: 0x20001006, size: 1, signed: false, idx: -1},
		{addr: 0x20001007, size: 4, signed: false, idx: -1},
		{addr: 0x2000100b, size: 1, signed: false, idx: -1},
		{addr: 0x20001017, size: 1, signed: false, idx: -1},
		{addr: 0x20001018, size: 4, signed: false, idx: -1},
		{addr: 0x2000101c, size: 1, signed: false, idx: -1},
		{addr: 0x20001020, size: 4, signed: false, idx: -1},
		{addr: 0x20001028, size: 1, signed: false, idx: -1},
		{addr: 0x20001029, size: 4, signed: false, idx: -1},
		{addr: 0x2000102d, size: 1, signed: false, idx: -1},
		{addr: 0x20001031, size: 4, signed: false, idx: -1},
		{addr: 0x20001039, size: 1, signed: false, idx: -1},
		{addr: 0x2000103a, size: 4, signed: false, idx: -1},
		{addr: 0x2000103e, size: 1, signed: false, idx: -1},
		{addr: 0x20001042, size: 4, signed: false, idx: -1},
		{addr: 0x2000104a, size: 1, signed: false, idx: -1},
		{addr: 0x2000104b, size: 4, signed: false, idx: -1},
		{addr: 0x2000104f, size: 1, signed: false, idx: -1},
		{addr: 0x20001053, size: 4, signed: false, idx: -1},
		{addr: 0x2000105b, size: 1, signed: false, idx: -1},
		{addr: 0x2000105c, size: 4, signed: false, idx: -1},
		{addr: 0x20001060, size: 1, signed: false, idx: -1},
		{addr: 0x20001064, size: 4, signed: false, idx: -1},
		{addr: 0x2000106c, size: 1, signed: false, idx: -1},
		{addr: 0x2000106d, size: 4, signed: false, idx: -1},
		{addr: 0x20001071, size: 1, signed: false, idx: -1},
		{addr: 0x20001075, size: 4, signed: false, idx: -1},
		{addr: 0x200010d4, size: 4, signed: false, idx: -1},
		{addr: 0x200010ec, size: 1, signed: false, idx: -1},
		{addr: 0x683f94, size: 1, signed: false, idx: -1},
		{addr: 0x68ec78, size: 4, signed: false, idx: -1},
		{addr: 0x20001af4, size: 4, signed: false, idx: -1},
		{addr: 0x20004008, size: 4, signed: false, idx: -1},
		{addr: 0x200047b0, size: 4, signed: false, idx: -1},
		{addr: 0x200047b8, size: 4, signed: false, idx: -1},
		{addr: 0x20004910, size: 4, signed: false, idx: -1},
		{addr: 0x20002000, size: 4, signed: false, idx: -1},
		{addr: 0x20002004, size: 4, signed: false, idx: -1},
		{addr: 0x20002008, size: 1, signed: false, idx: -1},
		{addr: 0x20002009, size: 1, signed: false, idx: -1},
		{addr: 0x2000200c, size: 4, signed: false, idx: -1},
		{addr: 0x20002010, size: 4, signed: false, idx: -1},
		{addr: 0x20002014, size: 4, signed: false, idx: -1},
		{addr: 0x20002018, size: 4, signed: false, idx: -1},
		{addr: 0x2000201c, size: 4, signed: false, idx: -1},
		{addr: 0x20002020, size: 4, signed: false, idx: -1},
		{addr: 0x20002024, size: 4, signed: false, idx: -1},
		{addr: 0x20002028, size: 4, signed: false, idx: -1},
		{addr: 0x2000202c, size: 4, signed: false, idx: -1},
		{addr: 0x20002030, size: 4, signed: false, idx: -1},
		{addr: 0x20002034, size: 4, signed: false, idx: -1},
		{addr: 0x20002038, size: 4, signed: false, idx: -1},
		{addr: 0x2000203c, size: 4, signed: false, idx: -1},
		{addr: 0x20002040, size: 4, signed: false, idx: -1},
		{addr: 0x20002044, size: 4, signed: false, idx: -1},
		{addr: 0x20002048, size: 4, signed: false, idx: -1},
		{addr: 0x20001c00, size: 4, signed: false, idx: -1},
		{addr: 0x20001c04, size: 4, signed: true, idx: -1},
		{addr: 0x20001c08, size: 4, signed: true, idx: -1},
		{addr: 0x20001c1b, size: 1, signed: false, idx: -1},
		{addr: 0x20001c1c, size: 1, signed: false, idx: -1},
		{addr: 0x20001c1e, size: 1, signed: false, idx: -1},
	},
	out: []mstFld{
		{addr: 0x20000057, size: 4, signed: false, idx: -1},
		{addr: 0x2000005b, size: 4, signed: false, idx: -1},
		{addr: 0x2000005f, size: 4, signed: false, idx: -1},
		{addr: 0x20000063, size: 4, signed: false, idx: -1},
		{addr: 0x20000067, size: 4, signed: false, idx: -1},
		{addr: 0x2000006b, size: 4, signed: false, idx: -1},
		{addr: 0x2000006f, size: 4, signed: false, idx: -1},
		{addr: 0x20000073, size: 4, signed: false, idx: -1},
		{addr: 0x20000077, size: 4, signed: false, idx: -1},
		{addr: 0x2000007b, size: 4, signed: false, idx: -1},
		{addr: 0x2000007f, size: 4, signed: false, idx: -1},
		{addr: 0x20000083, size: 4, signed: false, idx: -1},
		{addr: 0x20000087, size: 4, signed: false, idx: -1},
		{addr: 0x2000008b, size: 4, signed: false, idx: -1},
		{addr: 0x2000008f, size: 4, signed: false, idx: -1},
		{addr: 0x20000093, size: 4, signed: false, idx: -1},
		{addr: 0x20000097, size: 4, signed: false, idx: -1},
		{addr: 0x2000009b, size: 4, signed: false, idx: -1},
		{addr: 0x2000009f, size: 4, signed: false, idx: -1},
		{addr: 0x200000a3, size: 4, signed: false, idx: -1},
		{addr: 0x20000107, size: 4, signed: false, idx: -1},
		{addr: 0x20000016, size: 1, signed: false, idx: -1},
		{addr: 0x2000024e, size: 1, signed: false, idx: -1},
		{addr: 0x2000010b, size: 4, signed: false, idx: -1},
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
		{addr: 0x20000033, size: 4, signed: false, idx: -1},
		{addr: 0x20000242, size: 4, signed: false, idx: -1},
		{addr: 0x200002e1, size: 4, signed: true, idx: -1},
		{addr: 0x200002e5, size: 4, signed: false, idx: -1},
		{addr: 0x200002ba, size: 4, signed: false, idx: -1},
		{addr: 0x200002be, size: 4, signed: false, idx: -1},
		{addr: 0x200002c2, size: 4, signed: false, idx: -1},
		{addr: 0x200002c6, size: 4, signed: false, idx: -1},
		{addr: 0x200002ca, size: 4, signed: false, idx: -1},
		{addr: 0x200002ce, size: 4, signed: false, idx: -1},
		{addr: 0x200002d2, size: 4, signed: false, idx: -1},
		{addr: 0x200002d6, size: 4, signed: false, idx: -1},
		{addr: 0x200001db, size: 1, signed: false, idx: -1},
		{addr: 0x200001dc, size: 1, signed: false, idx: -1},
		{addr: 0x200001dd, size: 1, signed: false, idx: -1},
		{addr: 0x200001de, size: 1, signed: false, idx: -1},
		{addr: 0x200001df, size: 1, signed: false, idx: -1},
		{addr: 0x200001e0, size: 1, signed: false, idx: -1},
		{addr: 0x200001e1, size: 1, signed: false, idx: -1},
		{addr: 0x200001e2, size: 1, signed: false, idx: -1},
		{addr: 0x200001e3, size: 1, signed: false, idx: -1},
		{addr: 0x200001e4, size: 1, signed: false, idx: -1},
		{addr: 0x200001e5, size: 1, signed: false, idx: -1},
		{addr: 0x200001e6, size: 1, signed: false, idx: -1},
		{addr: 0x200001e7, size: 1, signed: false, idx: -1},
		{addr: 0x200001e8, size: 1, signed: false, idx: -1},
		{addr: 0x200001e9, size: 1, signed: false, idx: -1},
		{addr: 0x200001ea, size: 1, signed: false, idx: -1},
		{addr: 0x200001eb, size: 1, signed: false, idx: -1},
		{addr: 0x200001ec, size: 1, signed: false, idx: -1},
		{addr: 0x200001ed, size: 1, signed: false, idx: -1},
		{addr: 0x200001ee, size: 1, signed: false, idx: -1},
		{addr: 0x200001ef, size: 1, signed: false, idx: -1},
		{addr: 0x200001f0, size: 1, signed: false, idx: -1},
		{addr: 0x200001f1, size: 1, signed: false, idx: -1},
		{addr: 0x200001f2, size: 1, signed: false, idx: -1},
		{addr: 0x200001f3, size: 1, signed: false, idx: -1},
		{addr: 0x200001f4, size: 1, signed: false, idx: -1},
		{addr: 0x200001f5, size: 1, signed: false, idx: -1},
		{addr: 0x200001f6, size: 1, signed: false, idx: -1},
		{addr: 0x200001f7, size: 1, signed: false, idx: -1},
		{addr: 0x200001f8, size: 1, signed: false, idx: -1},
		{addr: 0x200001f9, size: 1, signed: false, idx: -1},
		{addr: 0x200001fa, size: 1, signed: false, idx: -1},
		{addr: 0x200001fb, size: 1, signed: false, idx: -1},
		{addr: 0x200001fc, size: 1, signed: false, idx: -1},
		{addr: 0x200001fd, size: 1, signed: false, idx: -1},
		{addr: 0x200001fe, size: 1, signed: false, idx: -1},
		{addr: 0x200001ff, size: 1, signed: false, idx: -1},
		{addr: 0x20000200, size: 1, signed: false, idx: -1},
		{addr: 0x20000201, size: 1, signed: false, idx: -1},
		{addr: 0x20000202, size: 1, signed: false, idx: -1},
		{addr: 0x20000203, size: 1, signed: false, idx: -1},
		{addr: 0x20000204, size: 1, signed: false, idx: -1},
		{addr: 0x20000205, size: 1, signed: false, idx: -1},
		{addr: 0x20000206, size: 1, signed: false, idx: -1},
		{addr: 0x20000207, size: 1, signed: false, idx: -1},
		{addr: 0x20000208, size: 1, signed: false, idx: -1},
		{addr: 0x20000209, size: 1, signed: false, idx: -1},
		{addr: 0x2000020a, size: 1, signed: false, idx: -1},
		{addr: 0x2000020b, size: 1, signed: false, idx: -1},
		{addr: 0x2000020c, size: 1, signed: false, idx: -1},
		{addr: 0x2000020d, size: 1, signed: false, idx: -1},
		{addr: 0x2000020e, size: 1, signed: false, idx: -1},
		{addr: 0x2000020f, size: 1, signed: false, idx: -1},
		{addr: 0x20000210, size: 1, signed: false, idx: -1},
		{addr: 0x20000211, size: 1, signed: false, idx: -1},
		{addr: 0x20000212, size: 1, signed: false, idx: -1},
		{addr: 0x20000213, size: 1, signed: false, idx: -1},
		{addr: 0x20000214, size: 1, signed: false, idx: -1},
		{addr: 0x20000215, size: 1, signed: false, idx: -1},
		{addr: 0x20000216, size: 1, signed: false, idx: -1},
		{addr: 0x20000217, size: 1, signed: false, idx: -1},
		{addr: 0x20000218, size: 1, signed: false, idx: -1},
		{addr: 0x20000219, size: 1, signed: false, idx: -1},
		{addr: 0x2000021a, size: 1, signed: false, idx: -1},
		{addr: 0x2000021b, size: 1, signed: false, idx: -1},
		{addr: 0x2000021c, size: 1, signed: false, idx: -1},
		{addr: 0x2000021d, size: 1, signed: false, idx: -1},
		{addr: 0x2000021e, size: 1, signed: false, idx: -1},
		{addr: 0x2000021f, size: 1, signed: false, idx: -1},
		{addr: 0x20000220, size: 1, signed: false, idx: -1},
		{addr: 0x20000221, size: 1, signed: false, idx: -1},
		{addr: 0x20000222, size: 1, signed: false, idx: -1},
		{addr: 0x20000223, size: 1, signed: false, idx: -1},
		{addr: 0x20000224, size: 1, signed: false, idx: -1},
		{addr: 0x20000225, size: 1, signed: false, idx: -1},
		{addr: 0x20000226, size: 1, signed: false, idx: -1},
		{addr: 0x20000227, size: 1, signed: false, idx: -1},
		{addr: 0x20000228, size: 1, signed: false, idx: -1},
		{addr: 0x20000229, size: 1, signed: false, idx: -1},
		{addr: 0x2000022a, size: 1, signed: false, idx: -1},
		{addr: 0x2000022b, size: 1, signed: false, idx: -1},
		{addr: 0x2000022c, size: 1, signed: false, idx: -1},
		{addr: 0x2000022d, size: 1, signed: false, idx: -1},
		{addr: 0x2000022e, size: 1, signed: false, idx: -1},
		{addr: 0x2000022f, size: 1, signed: false, idx: -1},
		{addr: 0x20000230, size: 1, signed: false, idx: -1},
		{addr: 0x20000231, size: 1, signed: false, idx: -1},
		{addr: 0x20000232, size: 1, signed: false, idx: -1},
		{addr: 0x20000233, size: 1, signed: false, idx: -1},
		{addr: 0x20000234, size: 1, signed: false, idx: -1},
		{addr: 0x20000235, size: 1, signed: false, idx: -1},
		{addr: 0x20000236, size: 1, signed: false, idx: -1},
		{addr: 0x20000237, size: 1, signed: false, idx: -1},
		{addr: 0x20000238, size: 1, signed: false, idx: -1},
		{addr: 0x20000239, size: 1, signed: false, idx: -1},
		{addr: 0x2000023a, size: 1, signed: false, idx: -1},
		{addr: 0x2000023b, size: 1, signed: false, idx: -1},
		{addr: 0x2000023c, size: 1, signed: false, idx: -1},
		{addr: 0x2000023d, size: 1, signed: false, idx: -1},
		{addr: 0x2000023e, size: 1, signed: false, idx: -1},
		{addr: 0x2000023f, size: 1, signed: false, idx: -1},
		{addr: 0x20000240, size: 1, signed: false, idx: -1},
		{addr: 0x20000241, size: 1, signed: false, idx: -1},
		{addr: 0x20000247, size: 1, signed: false, idx: -1},
		{addr: 0x20000248, size: 1, signed: false, idx: -1},
		{addr: 0x20000249, size: 1, signed: false, idx: -1},
		{addr: 0x2000024a, size: 1, signed: false, idx: -1},
		{addr: 0x2000024b, size: 1, signed: false, idx: -1},
		{addr: 0x20000047, size: 4, signed: false, idx: -1},
		{addr: 0x2000004f, size: 4, signed: false, idx: -1},
		{addr: 0x200002ab, size: 1, signed: false, idx: -1},
		{addr: 0x20000f40, size: 4, signed: false, idx: -1},
		{addr: 0x20000f45, size: 4, signed: false, idx: -1},
		{addr: 0x20000f49, size: 4, signed: false, idx: -1},
		{addr: 0x20000f51, size: 4, signed: false, idx: -1},
		{addr: 0x20000f5a, size: 4, signed: false, idx: -1},
		{addr: 0x20000f5e, size: 4, signed: false, idx: -1},
		{addr: 0x20000f62, size: 4, signed: false, idx: -1},
		{addr: 0x20000f66, size: 4, signed: false, idx: -1},
		{addr: 0x20000f6a, size: 4, signed: false, idx: -1},
		{addr: 0x20000f6e, size: 4, signed: false, idx: -1},
		{addr: 0x20000f72, size: 4, signed: false, idx: -1},
		{addr: 0x20000f76, size: 4, signed: false, idx: -1},
		{addr: 0x200010d4, size: 4, signed: false, idx: -1},
		{addr: 0x20001af4, size: 4, signed: false, idx: -1},
		{addr: 0x200047b0, size: 4, signed: false, idx: -1},
		{addr: 0x200047b8, size: 4, signed: false, idx: -1},
		{addr: 0x2000204c, size: 1, signed: false, idx: -1},
		{addr: 0x2000204d, size: 1, signed: false, idx: -1},
		{addr: 0x20002050, size: 4, signed: false, idx: -1},
		{addr: 0x20002054, size: 4, signed: false, idx: -1},
		{addr: 0x20002058, size: 4, signed: false, idx: -1},
		{addr: 0x2000205c, size: 4, signed: false, idx: -1},
		{addr: 0x20002060, size: 4, signed: false, idx: -1},
		{addr: 0x20002064, size: 4, signed: false, idx: -1},
		{addr: 0x20002068, size: 4, signed: false, idx: -1},
		{addr: 0x2000206c, size: 4, signed: false, idx: -1},
		{addr: 0x20002070, size: 4, signed: false, idx: -1},
		{addr: 0x20002074, size: 4, signed: false, idx: -1},
		{addr: 0x20002078, size: 4, signed: false, idx: -1},
		{addr: 0x2000207c, size: 4, signed: false, idx: -1},
		{addr: 0x20002080, size: 4, signed: false, idx: -1},
		{addr: 0x20002084, size: 4, signed: false, idx: -1},
		{addr: 0x20002088, size: 4, signed: false, idx: -1},
		{addr: 0x2000208c, size: 4, signed: false, idx: -1},
		{addr: 0x20002090, size: 4, signed: false, idx: -1},
		{addr: 0x20002094, size: 4, signed: false, idx: -1},
		{addr: 0x20002098, size: 4, signed: false, idx: -1},
		{addr: 0x2000209c, size: 4, signed: false, idx: -1},
		{addr: 0x200020a0, size: 4, signed: false, idx: -1},
		{addr: 0x200020a4, size: 4, signed: false, idx: -1},
		{addr: 0x200020a8, size: 4, signed: false, idx: -1},
		{addr: 0x200020ac, size: 4, signed: false, idx: -1},
		{addr: 0x200020b0, size: 4, signed: false, idx: -1},
		{addr: 0x200020b4, size: 4, signed: false, idx: -1},
		{addr: 0x200020b8, size: 4, signed: false, idx: -1},
		{addr: 0x200020bc, size: 4, signed: false, idx: -1},
		{addr: 0x200020c0, size: 4, signed: false, idx: -1},
		{addr: 0x200020c4, size: 4, signed: false, idx: -1},
		{addr: 0x200020c8, size: 4, signed: false, idx: -1},
		{addr: 0x200020cc, size: 4, signed: false, idx: -1},
		{addr: 0x200020d0, size: 4, signed: false, idx: -1},
		{addr: 0x200020d4, size: 4, signed: false, idx: -1},
		{addr: 0x200020d8, size: 4, signed: false, idx: -1},
		{addr: 0x200020dc, size: 4, signed: false, idx: -1},
		{addr: 0x200020e0, size: 4, signed: false, idx: -1},
		{addr: 0x200020e4, size: 4, signed: false, idx: -1},
		{addr: 0x200020e8, size: 4, signed: false, idx: -1},
		{addr: 0x200020ec, size: 4, signed: false, idx: -1},
		{addr: 0x200020f0, size: 4, signed: false, idx: -1},
		{addr: 0x200020f4, size: 4, signed: false, idx: -1},
		{addr: 0x200020f8, size: 4, signed: false, idx: -1},
		{addr: 0x200020fc, size: 4, signed: false, idx: -1},
		{addr: 0x20002100, size: 4, signed: false, idx: -1},
		{addr: 0x20002104, size: 4, signed: false, idx: -1},
		{addr: 0x20002108, size: 4, signed: false, idx: -1},
		{addr: 0x2000210c, size: 4, signed: false, idx: -1},
	},
})

var mcaSpecApplyIn = []string{"cond", "duration", "penalty", "message", "M.active", "M.inCombat", "M.morale", "M.maxHP", "M.HP", "M.maxSP", "M.SP", "M.cond0", "M.cond1", "M.cond2", "M.cond3", "M.cond4", "M.cond5", "M.cond6", "M.cond7", "M.cond8", "M.cond9", "M.cond10", "M.cond11", "M.cond12", "M.cond13", "M.cond14", "M.cond15", "M.cond16", "M.cond17", "M.cond18", "M.cond19", "M.buff5.power", "M.buff5.mult", "M.buff5.dur", "M.buff6.power", "M.buff6.mult", "M.buff6.dur", "M.buff7.power", "M.buff7.mult", "M.buff7.dur", "M.mainCond", "M.attackPenalty", "M.eff0.active", "M.eff0.type", "M.eff0.power", "M.eff0.mult", "M.eff5.active", "M.eff5.type", "M.eff5.power", "M.eff5.mult", "M.eff11.active", "M.eff11.type", "M.eff11.power", "M.eff11.mult", "M.disabledVisual", "M.aware", "M.action", "M.subaction", "M.removed", "M2.mainCond", "M2.removed", "tpl.kind", "tpl.attr0", "tpl.attr1", "tpl.attr2", "tpl.attr3", "tpl.attr4", "tpl.species", "tpl.f24f", "tpl.fe5", "vis.f332", "actor.fa4", "actor.fa7", "T.phase", "T.f4", "T.f5", "T.f9", "T.f11", "T.char0", "T.char1", "T.char2", "T.char3", "T.char4", "T.char5", "T.char6", "T.char7", "T.sp0.active", "T.sp0.code", "T.sp0.value", "T.sp1.active", "T.sp1.code", "T.sp1.value", "T.sp2.active", "T.sp2.code", "T.sp2.value", "T.sp3.active", "T.sp3.code", "T.sp3.value", "T.sp4.active", "T.sp4.code", "T.sp4.value", "T.sp5.active", "T.sp5.code", "T.sp5.value", "T.sp6.active", "T.sp6.code", "T.sp6.value", "T.sp7.active", "T.sp7.code", "T.sp7.value", "T.sp8.active", "T.sp8.code", "T.sp8.value", "T.aura0.active", "T.aura0.code", "T.aura0.power", "T.aura0.mult", "T.aura1.active", "T.aura1.code", "T.aura1.power", "T.aura1.mult", "T.aura2.active", "T.aura2.code", "T.aura2.power", "T.aura2.mult", "T.aura3.active", "T.aura3.code", "T.aura3.power", "T.aura3.mult", "T.aura4.active", "T.aura4.code", "T.aura4.power", "T.aura4.mult", "T.aura5.active", "T.aura5.code", "T.aura5.power", "T.aura5.mult", "grp.living", "grp.f2c", "g683f94", "gate", "ui.flags", "G.f8", "G.f7b0", "G.f7b8", "G.f910", "r4c2cf0_0", "r4c2cf0_1", "w5049c0_0", "w5049c0_1", "r52c240", "r5323f0", "w5323f0_M2e1", "w5323f0_M2e5", "w5323f0_M2ba0", "w5323f0_M2ba1", "w5323f0_M2ba2", "w5323f0_M2ba3", "w5323f0_M2ba4", "w5323f0_M2ba5", "w5323f0_M2ba6", "w5323f0_M2ba7", "w5323f0_T5", "w5323f0_T9", "w5323f0_T11", "w4f0480", "a5.type", "a5.char", "a5.monster", "a5.f1b", "a5.f1c", "a5.f1e"}

var mcaStubs = map[uint32]*mcaStub{
	0x4c2cf0: {ret: 0x20002000, retN: 2, rec: [][4]uint32{{1, 1, 0x2000204c, 2}}, cnt: 0x20002050, wr: []mcaWrite{}},
	0x4e4db0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20002054, 4}, {3, 4, 0x20002064, 4}}, cnt: 0x20002074, wr: []mcaWrite{}},
	0x58aad0: {ret: 0x0, retN: 0, rec: [][4]uint32{{3, 4, 0x20002078, 2}, {4, 4, 0x20002080, 2}}, cnt: 0x20002088, wr: []mcaWrite{}},
	0x4acd80: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x2000208c, 2}}, cnt: 0x20002094, wr: []mcaWrite{}},
	0x4c5a00: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20002098, 2}}, cnt: 0x200020a0, wr: []mcaWrite{}},
	0x4c5aa0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200020a4, 2}}, cnt: 0x200020ac, wr: []mcaWrite{}},
	0x4e4280: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x200020b0, wr: []mcaWrite{}},
	0x4e46f0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200020b4, 1}}, cnt: 0x200020b8, wr: []mcaWrite{}},
	0x4e3af0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200020bc, 1}}, cnt: 0x200020c0, wr: []mcaWrite{}},
	0x546e70: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x200020c4, wr: []mcaWrite{}},
	0x538e00: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200020c8, 1}}, cnt: 0x200020cc, wr: []mcaWrite{}},
	0x531540: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x200020d0, wr: []mcaWrite{}},
	0x4c6200: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200020d4, 1}}, cnt: 0x200020d8, wr: []mcaWrite{}},
	0x4f0480: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200020dc, 1}}, cnt: 0x200020e0, wr: []mcaWrite{{1, 0x0, 4, 0x20002048, 0}}},
	0x52c240: {ret: 0x2000200c, retN: 0, rec: [][4]uint32{{2, 4, 0x200020e4, 1}}, cnt: 0x200020e8, wr: []mcaWrite{}},
	0x5323f0: {ret: 0x20002010, retN: 0, rec: [][4]uint32{{2, 4, 0x200020ec, 1}, {3, 4, 0x200020f0, 1}, {4, 4, 0x200020f4, 1}}, cnt: 0x200020f8, wr: []mcaWrite{{1, 0x2e1, 4, 0x20002014, 0}, {1, 0x2e5, 4, 0x20002018, 0}, {1, 0x2ba, 4, 0x2000201c, 0}, {1, 0x2be, 4, 0x20002020, 0}, {1, 0x2c2, 4, 0x20002024, 0}, {1, 0x2c6, 4, 0x20002028, 0}, {1, 0x2ca, 4, 0x2000202c, 0}, {1, 0x2ce, 4, 0x20002030, 0}, {1, 0x2d2, 4, 0x20002034, 0}, {1, 0x2d6, 4, 0x20002038, 0}, {0, 0x20000f45, 4, 0x2000203c, 0}, {0, 0x20000f49, 4, 0x20002040, 0}, {0, 0x20000f51, 4, 0x20002044, 0}}},
	0x5049c0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x200020fc, 2}, {3, 4, 0x20002104, 2}}, cnt: 0x2000210c, wr: []mcaWrite{{1, 0x2ab, 1, 0x20002008, 2}}},
}

var mcaOutNames = []string{"M.cond0", "M.cond1", "M.cond2", "M.cond3", "M.cond4", "M.cond5", "M.cond6", "M.cond7", "M.cond8", "M.cond9", "M.cond10", "M.cond11", "M.cond12", "M.cond13", "M.cond14", "M.cond15", "M.cond16", "M.cond17", "M.cond18", "M.cond19", "M.mainCond", "M.morale", "M.disabledVisual", "M.attackPenalty", "M.source0", "M.source1", "M.source2", "M.source3", "M.source4", "M.source5", "M.source6", "M.source7", "M.source8", "M.source9", "M.source10", "M.source11", "M.source12", "M.SP", "M.fatigue", "M.action", "M.subaction", "M.target0", "M.target1", "M.target2", "M.target3", "M.target4", "M.target5", "M.target6", "M.target7", "M.mod00", "M.mod01", "M.mod02", "M.mod03", "M.mod04", "M.mod05", "M.mod06", "M.mod07", "M.mod08", "M.mod09", "M.mod0a", "M.mod0b", "M.mod0c", "M.mod0d", "M.mod0e", "M.mod0f", "M.mod10", "M.mod11", "M.mod12", "M.mod13", "M.mod14", "M.mod15", "M.mod16", "M.mod17", "M.mod18", "M.mod19", "M.mod1a", "M.mod1b", "M.mod1c", "M.mod1d", "M.mod1e", "M.mod1f", "M.mod20", "M.mod21", "M.mod22", "M.mod23", "M.mod24", "M.mod25", "M.mod26", "M.mod27", "M.mod28", "M.mod29", "M.mod2a", "M.mod2b", "M.mod2c", "M.mod2d", "M.mod2e", "M.mod2f", "M.mod30", "M.mod31", "M.mod32", "M.mod33", "M.mod34", "M.mod35", "M.mod36", "M.mod37", "M.mod38", "M.mod39", "M.mod3a", "M.mod3b", "M.mod3c", "M.mod3d", "M.mod3e", "M.mod3f", "M.mod40", "M.mod41", "M.mod42", "M.mod43", "M.mod44", "M.mod45", "M.mod46", "M.mod47", "M.mod48", "M.mod49", "M.mod4a", "M.mod4b", "M.mod4c", "M.mod4d", "M.mod4e", "M.mod4f", "M.mod50", "M.mod51", "M.mod52", "M.mod53", "M.mod54", "M.mod55", "M.mod56", "M.mod57", "M.mod58", "M.mod59", "M.mod5a", "M.mod5b", "M.mod5c", "M.mod5d", "M.mod5e", "M.mod5f", "M.mod60", "M.mod61", "M.mod62", "M.mod63", "M.mod64", "M.mod65", "M.mod66", "M.attr0", "M.attr1", "M.attr2", "M.attr3", "M.attr4", "M.hpRecovery", "M.spRecovery", "M.aware", "T.phase", "T.f5", "T.f9", "T.f11", "T.char0", "T.char1", "T.char2", "T.char3", "T.char4", "T.char5", "T.char6", "T.char7", "grp.living", "ui.flags", "G.f7b0", "G.f7b8", "s4c2cf0_a1_0", "s4c2cf0_a1_1", "s4c2cf0_calls", "s4e4db0_a2_0", "s4e4db0_a2_1", "s4e4db0_a2_2", "s4e4db0_a2_3", "s4e4db0_a3_0", "s4e4db0_a3_1", "s4e4db0_a3_2", "s4e4db0_a3_3", "s4e4db0_calls", "s58aad0_a3_0", "s58aad0_a3_1", "s58aad0_a4_0", "s58aad0_a4_1", "s58aad0_calls", "s4acd80_a2_0", "s4acd80_a2_1", "s4acd80_calls", "s4c5a00_a2_0", "s4c5a00_a2_1", "s4c5a00_calls", "s4c5aa0_a2_0", "s4c5aa0_a2_1", "s4c5aa0_calls", "s4e4280_calls", "s4e46f0_a2_0", "s4e46f0_calls", "s4e3af0_a1_0", "s4e3af0_calls", "s546e70_calls", "s538e00_a2_0", "s538e00_calls", "s531540_calls", "s4c6200_a2_0", "s4c6200_calls", "s4f0480_a2_0", "s4f0480_calls", "s52c240_a2_0", "s52c240_calls", "s5323f0_a2_0", "s5323f0_a3_0", "s5323f0_a4_0", "s5323f0_calls", "s5049c0_a2_0", "s5049c0_a2_1", "s5049c0_a3_0", "s5049c0_a3_1", "s5049c0_calls"}

// mcaVauParked is the first arg2 == 1 boundary row of re/emu/cases/005049c0.json.
var mcaVauParked = []int64{7, 1, 0, 0, 0, 1, 1, 0, 0, 50, 0, 1100, 5, 0, 9, 1, 0, 2, 1115684864, 1101004800, 1091043328, 1069547520, 1073741824, 0, 305419896, 1076887552, 0, 1065353216, 1078984704, 1101004800, 1056964608, 1065353216, 1115684864, 1189765120, 12, 3, 2, 255, 10, 127, 7, 2, 1, 5, 10, 128, 12, 0, 1, 0, 10, 200, 1, 0, 5, 4294967295, 125, 0, 0, 1, 1, 5, 4294967295, 5, 0, 0, 0, 1, 7, 6, 9999, 50, 125, 0, 14, 0, 7, 6, 9999, 4294967295, 125, 1, 18, 1, 1, 6, 9998, 50, 125, 1, 0, 1, 1, 5, 9998, 50, 125, 0, 15, 1, 0, 6, 0, 0, 0, 0, 3, 0, 1, 5, 0, 0, 0, 0, 14, 1, 3, 0, 1, 1, 1093664768, 1, 3, 0, 50, 0, 300, 0, 1, 1203982208, 2, 0, 300, 0, 23, 1, 0, 1, 1, 23, 0, 2, 3, 1, 1, 0, 1, 1115684864, 1115684864, 1090519040, 0, 1056964608, 2, 2, 0, 1, 0, 1090519040, 1065353216, 1101004800, 1069547520, 0, 2, 3, 0, 1, 16909060, 0, 16909060, 0, 0, 16909060, 4294967295, 16909060, 4294967295, 4294967295, 4294967295, 2, 0, 0, 4294967295, 16909060, 0, 0, 16909060, 16909060, 4294967295, 1065353216, 16909060, 1065353216, 3, 0, 0, 1000, 0, 1065353216, 4294967295, 4294967295, 4294967295, 0, 1065353216, 0, 1065353216, 2, 0, 7, 7, 1, 1, 7, 1, 0, 7, 7, 1, 2, 0, 1, 2, 65536, 65536, 5, 536892928}
