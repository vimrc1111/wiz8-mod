package calc

import "fmt"

// ongoingSpellTickNode is 0x5526f0 ongoing_spell_tick(slots, target): one tick
// of the nine ongoing spell slots (stride 0x11: active byte +0, spell id +1,
// power level +5, bonus percent +9) on one target record (type +0: 2 = party,
// 3 = monster; +4 party index; +8 monster id).
//
// Per active slot, with record = [0x65be1c] + id*0x1bf (notes/re/spells.md):
// power = 0x4ff790(0, id, level) = level + ((+0x49 / 2) + +0x56) / 2 (signed,
// truncating), minus 3 when the spell class +0x137 is 7 (0x4ac9d0 with flag 0),
// floored at 0; then 0x517f90(&power, bonus >> 1) adds the u32 percentage. The
// slot kind comes from the executable's byte table 0x552eb8 indexed by id-0x30
// (ids 0x30..0x5d) and its target table 0x552eac (read at run time): damage
// kinds take the dice word +0x51 with its count byte multiplied by the low byte
// of the level, the status kind 0x5d rolls status 0x12, every other id ends the
// slot.
//
// Monster target (0x4e5840 finds the id): damage = dice (0x517970) plus the
// bonus percentage (0x517f90), reduced by 0x552250(&v, target, realm +0x133,
// power); 0 prints the resisted notice 0x590b40(monster, text 0x1b3, handle)
// when byte 0x68510c is set, else 0x52bb60(monster, v, source, 0, 1, 0, 0, 0)
// runs natively (shield, hit message, last-source copy, death or HP loss with
// 0x52c070 and 0x52beb0), and id 0x50 also drains stamina 0x52c070(m, v, 0)
// and +0x2f9 by v/2. The status kind applies 0x523c00(id, 0x12, 9999, 0,
// source, 1) unless template +0x1be is set or 0x5520d0(target, realm of spell
// 0x5d, power, 0x12) resists. A monster at HP 0 ends the whole tick. Id 0x4c
// then computes an amount u = +0x44 * level + +0x4d (9999 passes unchanged;
// else u+1, one R(4): 0 bumps u > 1, 1 bumps u <= 2; plus the bonus
// percentage), reduces it through 0x552250 with the realm of spell 0x4c, maps
// 1 to 2, skips 0, and for statuses 4, 7, 0x11 applies 0x523c00(id, status, u,
// status 7 ? u : 0, source, 1) unless 0x5520d0 resists.
//
// Party target: the opening line 0x58aad0(0xc, text 0x1ae, record +0x9b) when
// the notice byte is set on a damage slot; members 0..7 with presence byte
// [0x685178]+0x106*i and primary state +0xb01 < 0x12 are processed with the
// local target record {1, i}: damage as above through 0x52a890(i, v, 0, 1, 0,
// 0, 0) (the resisted notice is 0x590950(i, text 0x1b3)), id 0x50 adds
// 0x52b6d0(i, k, v, 1) for k = 0..5 and 0x52af50(i, v, 0, 0); status 0x12 via
// 0x522fe0(i, 0x12, 9999, 0, 0, 1); id 0x4c as above with one R(4) per
// member, 0x552250 on the slot's own target record and 0x522fe0(i, status, u,
// status 7 ? u : 0, 0, 1). Without the notice byte and with hits: the summary
// 0x58aad0(0xc, text 0x1af, record +0x9b, hits, total / hits) (unsigned).
// The source record is 0x536150's: {0, -1, -1, 0 ...}.
//
// Address stubs (re/emu/cases/005526f0.json): D4 0x5520d0 (return from the
// case's statusResisted inputs), 0x52a890, 0x52b6d0, 0x52af50, 0x522fe0,
// 0x523c00, 0x52beb0, 0x4e4280, 0x4e46f0 (return input ignoredReturn; the
// caller ignores eax); 排除 0x58aad0, 0x58b410, 0x590b40, 0x48f650, 0x4acd80,
// 0x4e4db0, 0x4e3af0, 0x5182e0. Outputs are their call counts and every
// call's recorded arguments. 0x517970 and 0x552250 run natively (diceNode and
// the five-input resistanceQuantityReductionNode). Emu replays 0x5520d0's
// returns; F runs statusResistRoll on the world's records (F-only inputs:
// difficulty, growth, reverse, monster primary state and class). F panics when
// it would read a monster or character that a stubbed D4 callee already
// changed (the stub writes nothing, so F's read would be stale); HP is not
// among those writes.
//
// Fixture (setup, fixed): as 0x550c10's (UI mode 0x68ec78 = 0, UI object
// 0x68edcc = 0, actor flag +0x1dc bit 10, empty visual collection).
// Caps (panic past them): spell ids 0..0x5f, realms 0..5, party indices -1..7,
// monster ids 100..101, the stubs' recorded call counts, the 0x1000-byte heap
// window.
func ongoingSpellTickNode(in []int64, r Rand) []int64 {
	return mcaRun(&ostSpec, true, ostBody)(in, r)
}

func ostEmu(in []int64, r Rand) []int64 { return mcaRun(&ostSpec, false, ostBody)(in, r) }

func init() {
	Nodes["005526f0"] = Node{Addr: "005526f0", In: ostIn, Out: ostOut, F: ongoingSpellTickNode, Emu: ostEmu}
}

const (
	ostSlots   = 0x20000000 // the case's first argument (object spells)
	ostTarget  = 0x20000100 // the case's second argument (object target)
	ostRules   = 0x20040000 // fixture [0x65be1c] (object rules)
	ostRecords = 0x60       // fixture cap: spell records 0..0x5f
	// 0x5526f0's stack (emulator entry esp 0x10008000): the 0x536150 source
	// record and the local target record {1, i}, both passed by address.
	ostSrc   = 0x10007fcc
	ostLocal = 0x10007f7c
	// 0x590950 passes its stack prefix buffer and the slot after it to
	// 0x58aad0; measured from the emulator at this call depth.
	ostPrefix = 0x10005f20
	// 0x52c070 builds its own 0x536150 source record on its stack and passes
	// its address to 0x523c00; measured from the emulator at the two call
	// depths (inside 0x52bb60, and straight from 0x5526f0 for spell 0x50).
	ostFatigueSrcHit  = 0x10007eb8
	ostFatigueSrcTick = 0x10007eec
	// Spell records whose realm field 0x5526f0 reads by fixed offset
	// (record + 0x133): spell 0x4c (+0x85e7) and spell 0x5d (+0xa396).
	ostRealm4c = 0x4c*0x1bf + 0x133
	ostRealm5d = 0x5d*0x1bf + 0x133
)

// ostDecoys gives every spell record distinct values (the case's setup loop),
// so a read of the wrong record shows up as a difference.
func ostDecoys(s mstSpec) mstSpec {
	for i := int32(0); i < ostRecords; i++ {
		r := ostRules + uint32(i)*0x1bf
		s.set = append(s.set,
			mstSet{r + 0x44, 4, uint32((i*7 + 1) % 5)},
			mstSet{r + 0x4d, 4, uint32((i * 3) % 20)},
			mstSet{r + 0x49, 4, uint32((i*11)%50 - 10)},
			mstSet{r + 0x56, 4, uint32((i*13)%40 - 10)},
			mstSet{r + 0x137, 4, uint32(6 + i%3)},
			mstSet{r + 0x51, 2, uint32(i%15 - 3)},
			mstSet{r + 0x53, 1, uint32(i % 4)},
			mstSet{r + 0x54, 1, uint32(i%9 + 1)},
			mstSet{r + 0x133, 4, uint32(i % 6)})
	}
	return s
}

type ostState struct {
	w    *mcaWorld
	mon  map[uint32]bool // monsters a stubbed D4 callee changed
	char [8]bool         // party members a stubbed D4 callee changed
}

func (s *ostState) monClean(m uint32) {
	if s.w.real && s.mon[m] {
		panic("ongoing spell tick: monster read after a stubbed D4 callee changed it")
	}
}

func (s *ostState) charClean(i uint32) {
	if s.w.real && i < 8 && s.char[i] {
		panic("ongoing spell tick: character read after a stubbed D4 callee changed it")
	}
}

func ostCall(w *mcaWorld, a uint32, args ...uint32) uint32 {
	st := ostStubs[a]
	k := w.u32(st.cnt)
	for _, r := range st.rec {
		if k >= r[3] {
			panic(fmt.Sprintf("ongoing spell tick: stub %#x called more than %d times (fixture record cap)", a, r[3]))
		}
		w.put(r[2]+k*r[1], int(r[1]), uint64(args[r[0]-1]))
	}
	w.put(st.cnt, 4, uint64(k+1))
	return k
}

func ostReplay(w *mcaWorld, a, k uint32) uint32 {
	st := ostStubs[a]
	if st.retN > 0 {
		if k >= st.retN {
			panic(fmt.Sprintf("ongoing spell tick: stub %#x return past its %d values", a, st.retN))
		}
		return w.u32(st.ret + 4*k)
	}
	return w.u32(st.ret)
}

func ostStub(w *mcaWorld, a uint32, args ...uint32) uint32 {
	return ostReplay(w, a, ostCall(w, a, args...))
}

func ostRecord(w *mcaWorld, id uint32) uint32 {
	if id >= ostRecords {
		panic("ongoing spell tick: spell id past the fixture's 0x60 records")
	}
	return w.u32(0x65be1c) + id*0x1bf
}

// ostChar is the party record for index i (-1 addresses the record before
// the roster, as the original's unchecked index does).
func (s *ostState) ostChar(i uint32) uint32 {
	if int32(i) < -1 || int32(i) > 7 {
		panic("ongoing spell tick: party index outside -1..7 (fixture cap)")
	}
	s.charClean(i)
	return s.w.u32(0x685174) + uint32(int32(i)*0x1862)
}

// ostPower is 0x4ff790(0, id, level).
func ostPower(w *mcaWorld, rec, level uint32) uint32 {
	p := int32(level) + (int32(w.u32(rec+0x49))/2+int32(w.u32(rec+0x56)))/2
	if w.u32(rec+0x137) == 7 {
		p -= 3
	}
	return uint32(max(p, 0))
}

func ostPercent(v, p uint32) uint32 { return v + v*p/100 } // 0x517f90

func ostDice(w *mcaWorld, d uint32) uint32 {
	return uint32(int32(diceNode([]int64{int64(int16(d)), int64(byte(d >> 16)), int64(byte(d >> 24))}, w.r)[0]))
}

// ostResist is 0x552250(&v, record, realm, power) on a record of type typ
// (3: monster id mid; otherwise party index idx).
func (s *ostState) ostResist(v, typ, idx, mid, realm, power uint32) uint32 {
	w := s.w
	if v == 0x270f {
		return v
	}
	if realm > 5 {
		panic("ongoing spell tick: realm past 5 (fixture cap)")
	}
	var res, level uint32
	if typ == 3 {
		m := w.findMonster(mid)
		s.monClean(m)
		t := w.template(m)
		res = uint32(int32(int8(w.u8(m+0x217+realm))) + int32(w.u8(t+0x176+realm)))
		level = uint32(w.u8(t + 0x24f))
	} else {
		c := s.ostChar(idx)
		res, level = w.u32(c+0xee1+16*realm), w.u32(c+0x89)
	}
	return uint32(resistanceQuantityReductionNode([]int64{int64(v), int64(res), int64(level), int64(power), int64(typ)}, w.r)[0])
}

// ostRoll is 0x5520d0(record, realm, power, status): the resisted byte.
func (s *ostState) ostRoll(typ, idx, mid, realm, power, status uint32) byte {
	w := s.w
	arg := uint32(ostLocal)
	if typ == 3 {
		arg = ostTarget
	}
	k := ostCall(w, 0x5520d0, arg, realm, power, status)
	if !w.real {
		return byte(ostReplay(w, 0x5520d0, k))
	}
	v := make([]int64, len(srrIn))
	v[0], v[1], v[2], v[3], v[4], v[5], v[6] = int64(typ), int64(idx), int64(mid), int64(realm), int64(power), int64(status), int64(w.u32(0x6850d5))
	if typ == 3 {
		m := w.findMonster(mid)
		s.monClean(m)
		t := w.template(m)
		v[17] = int64(w.u32(m + 8))
		for j := uint32(0); j < 6; j++ {
			v[18+j], v[25+j] = int64(w.u8(t+0x176+j)), int64(int8(w.u8(m+0x217+j)))
		}
		v[24], v[31], v[32] = int64(w.u8(t+0x24f)), int64(w.u32(m+0x107)), int64(w.u8(m+0x16))
	} else {
		c := s.ostChar(idx)
		v[7] = int64(w.u32(c + 0x89))
		for j := uint32(0); j < 6; j++ {
			v[8+j] = int64(w.u32(c + 0xee1 + 16*j))
		}
		v[14], v[15], v[16] = int64(w.u32(c+0xb01)), int64(w.u8(c+0x6f5)), int64(w.u32(c+0xa35))
	}
	return byte(statusResistRoll(v, w.r, true)[0])
}

// ostAmount is the id 0x4c amount before its reduction.
func ostAmount(w *mcaWorld, rec, level, bonus uint32) uint32 {
	u := w.u32(rec+0x44)*level + w.u32(rec+0x4d)
	if u == 0x270f {
		return u
	}
	u++
	switch w.r.R(4) { // 0x40efa0(4)
	case 0:
		if u > 1 {
			u++
		}
	case 1:
		if u <= 2 {
			u++
		}
	}
	return ostPercent(u, bonus)
}

func ostBody(w *mcaWorld) {
	s := &ostState{w: w, mon: map[uint32]bool{}}
	notice := w.u8(0x68510c)
	for slot := uint32(0); slot < 9; slot++ {
		p := ostSlots + slot*0x11
		if w.u8(p) == 0 {
			continue
		}
		id, level, bonus := w.u32(p+1), w.u32(p+5), w.u32(p+9)
		rec := ostRecord(w, id)
		power := ostPercent(ostPower(w, rec, level), bonus>>1)
		if id-0x30 > 0x2d {
			continue
		}
		var dice uint32
		var dmg, stat bool
		switch t := w.u32(0x552eac + 4*uint32(w.u8(0x552eb8+id-0x30))); t {
		case 0x5527a7:
			dice = w.u32(rec + 0x51)
			dice = dice&^0xff0000 | uint32(byte(dice>>16)*byte(level))<<16
			dmg = true
		case 0x5527cd:
			stat = true
		case 0x552e83:
			continue
		default:
			panic(fmt.Sprintf("ongoing spell tick: jump target %#x", t))
		}
		switch w.u32(ostTarget) {
		case 2:
			s.party(id, rec, level, bonus, power, dice, dmg, stat, notice)
		case 3:
			if !s.monster(id, rec, level, bonus, power, dice, dmg, stat) {
				return
			}
		default:
			panic("ongoing spell tick: target type not 2 or 3 (0x5527ec asserts)")
		}
	}
}

// monster is the type-3 branch; false ends the tick (target at HP 0).
func (s *ostState) monster(id, rec, level, bonus, power, dice uint32, dmg, stat bool) bool {
	w := s.w
	mid := w.u32(ostTarget + 8)
	m := w.findMonster(mid)
	if dmg {
		v := ostPercent(ostDice(w, dice), bonus)
		v = s.ostResist(v, 3, 0, mid, w.u32(rec+0x133), power)
		if v == 0 {
			if w.u8(0x68510c) != 0 {
				h, ok := w.monsterHandle(mid)
				if !ok {
					panic("ongoing spell tick: monster id not found (0x4e5550 asserts)")
				}
				ostStub(w, 0x590b40, w.monsterRecord(h), srdMsg(w, 0x1b3), h)
			}
		} else {
			s.hit(m, v)
			if id == 0x50 {
				s.fatigue(m, v, ostFatigueSrcTick)
				s.monClean(m)
				w.put(m+0x2f9, 4, uint64(w.u32(m+0x2f9)-v>>1))
			}
		}
	}
	if stat && w.u8(w.template(m)+0x1be) == 0 && s.ostRoll(3, w.u32(ostTarget+4), mid, w.u32(w.u32(0x65be1c)+ostRealm5d), power, 0x12) == 0 {
		ostStub(w, 0x523c00, mid, 0x12, 0x270f, 0, ostSrc, 1)
		s.mon[m] = true
	}
	if w.u32(m+0x2b) == 0 {
		return false
	}
	if id != 0x4c {
		return true
	}
	realm := w.u32(w.u32(0x65be1c) + ostRealm4c)
	u := s.ostResist(ostAmount(w, rec, level, bonus), 3, 0, mid, realm, power)
	if u == 0 {
		return true
	}
	if u == 1 {
		u = 2
	}
	for _, c := range [3]uint32{4, 7, 0x11} {
		if s.ostRoll(3, w.u32(ostTarget+4), mid, w.u32(w.u32(0x65be1c)+ostRealm4c), power, c) == 0 {
			ostStub(w, 0x523c00, mid, c, u, map[bool]uint32{true: u}[c == 7], ostSrc, 1)
			s.mon[m] = true
		}
	}
	return true
}

// party is the type-2 branch over members 0..7.
func (s *ostState) party(id, rec, level, bonus, power, dice uint32, dmg, stat bool, notice byte) {
	w := s.w
	if notice != 0 && !stat {
		ostStub(w, 0x58aad0, 0xc, srdMsg(w, 0x1ae), rec+0x9b, 0, 0)
	}
	var total, hits uint32
	for i := uint32(0); i < 8; i++ {
		if w.u8(w.u32(0x685178)+0x106*i) == 0 {
			continue
		}
		s.charClean(i)
		if w.u32(w.u32(0x685174)+0x1862*i+0xb01) >= 0x12 {
			continue
		}
		if dmg {
			v := ostPercent(ostDice(w, dice), bonus)
			v = s.ostResist(v, 1, i, 0, w.u32(rec+0x133), power)
			if v == 0 {
				if w.u8(0x68510c) != 0 {
					ostNotice(w, i, srdMsg(w, 0x1b3))
				}
			} else {
				ostStub(w, 0x52a890, i, v, 0, 1, 0, 0, 0)
				s.char[i] = true
				total += v
				hits++
				if id == 0x50 {
					for k := uint32(0); k < 6; k++ {
						ostStub(w, 0x52b6d0, i, k, v, 1)
					}
					ostStub(w, 0x52af50, i, v, 0, 0)
				}
			}
		}
		if stat && s.ostRoll(1, i, 0, w.u32(rec+0x133), power, 0x12) == 0 {
			ostStub(w, 0x522fe0, i, 0x12, 0x270f, 0, 0, 1)
			s.char[i] = true
		}
		if id != 0x4c {
			continue
		}
		realm := w.u32(w.u32(0x65be1c) + ostRealm4c)
		u := s.ostResist(ostAmount(w, rec, level, bonus), w.u32(ostTarget), w.u32(ostTarget+4), w.u32(ostTarget+8), realm, power)
		if u == 0 {
			continue
		}
		if u == 1 {
			u = 2
		}
		for _, c := range [3]uint32{4, 7, 0x11} {
			if s.ostRoll(1, i, 0, w.u32(w.u32(0x65be1c)+ostRealm4c), power, c) == 0 {
				ostStub(w, 0x522fe0, i, c, u, map[bool]uint32{true: u}[c == 7], 0, 1)
				s.char[i] = true
			}
		}
	}
	if notice == 0 && hits > 0 {
		ostStub(w, 0x58aad0, 0xc, srdMsg(w, 0x1af), rec+0x9b, hits, total/hits)
	}
}

// ostNotice is 0x590950(index, text) at this call depth (srdNotice's body).
func ostNotice(w *mcaWorld, idx, f uint32) {
	t := []rune(srdFormat(w, f) + "\x00\x00")
	name := w.charBase(idx) + 5
	ostStub(w, 0x58aad0, 8, 0x64bf5c, name, ostPrefix, ostPrefix+4)
	n := uint32(len([]rune(srdWstr(w, name))))
	if t[0] == 0x27 {
		n++
		if t[1] == 0x73 {
			n++
		}
	}
	ostStub(w, 0x58b410, uint32(w.u8(w.u32(0x685178)+idx*0x106+0xf1)), 0, n, 0xffffffff)
}

// ostName is 0x4e5150(m, 0, 0) (srdName's body).
func ostName(w *mcaWorld, m uint32) uint32 {
	t := w.template(m)
	if w.u16(t+0x187) == 0x18d {
		srdPutW(w, 0x6875c3, srdFormat(w, 0x614b44, w.charBase(w.u32(0x6875ef))+5))
		return 0x6875c3
	}
	if w.u32(m+4) == 0 {
		if !w.act15(w.u32(m + 0xc)) {
			ostStub(w, 0x5182e0, 1, 0x614b20, w.u32(m))
		}
		return t + 0x60
	}
	if w.u8(w.group(w.u32(m+4))+0x2c) == 0 {
		return t + 0x60
	}
	return t
}

// hit is 0x52bb60(m, dmg, source, 0, 1, 0, 0, 0): srdHit with notice set,
// no result object and 0x536150's source record (type 0: no party credit,
// who 9, its flag bytes clear so it is copied to +0x304).
func (s *ostState) hit(m, dmg uint32) {
	w := s.w
	if w.u32(m+0x2b) == 0 {
		return
	}
	s.monClean(m)
	if w.u32(m+0xc7) != 0 {
		msg, shield := srdMsg(w, 0x193), w.u16(m+0xc5)
		if shield >= dmg {
			ostStub(w, 0x590b40, m, msg, dmg)
			w.put(m+0xc5, 2, uint64(shield-dmg))
			return
		}
		ostStub(w, 0x590b40, m, msg, shield)
		dmg -= shield
		// 0x524390(id, 2): clear buff slot 2, 0x4acd80, rebuild.
		w.put(m+0xbf, 4, 0)
		w.put(m+0xc3, 4, 0)
		w.put(m+0xc7, 4, 0)
		ostStub(w, 0x4acd80, w.u32(m+0xc), 0x12, 0)
		w.rebuild(w.u32(m))
	}
	if w.u8(0x683f94) != 0 || w.u8(m+0x2ab) != 0 {
		ostStub(w, 0x48f650, m, 0, 1)
		ostStub(w, 0x58aad0, 9, srdMsg(w, 0x256), ostName(w, m), dmg, 0x689b34)
	}
	for i, x := range [13]uint32{0, 0xffffffff, 0xffffffff} {
		w.put(m+0x304+4*uint32(i), 4, uint64(x))
	}
	if dmg >= w.u32(m+0x2b) {
		w.put(m+0x2b, 4, 0)
		s.downed(m)
	} else {
		w.put(m+0x2b, 4, uint64(w.u32(m+0x2b)-dmg))
		s.fatigue(m, 2*dmg/3, ostFatigueSrcHit)
		ostStub(w, 0x52beb0, m, ostSrc, 0)
		s.mon[m] = true
	}
	// 0x56a2a0 is idle with UI object 0x68edcc = 0; 0x4e5a50 allocates its
	// 0x14-byte collection and finds the fixture's empty one; 0x4c6c30 is idle
	// with actor flag +0x1dc bit 10.
	if w.u32(0x68edcc) != 0 {
		panic("ongoing spell tick: UI object outside the fixture")
	}
	if a := w.u32(m + 0xc); a != 0 {
		w.alloc(0x14)
		if w.u32(a+0x1dc)&0x400 == 0 {
			panic("ongoing spell tick: actor flag outside the fixture")
		}
	}
}

// downed is 0x4e4690(m, 1).
func (s *ostState) downed(m uint32) {
	w := s.w
	if w.act15(w.u32(m + 0xc)) {
		return
	}
	ostStub(w, 0x4e4db0, m, 0x15, 1)
	ostStub(w, 0x4e4280, m)
	ostStub(w, 0x4e46f0, m, 1)
	s.mon[m] = true
	h, ok := w.monsterHandle(w.u32(m))
	if !ok {
		panic("ongoing spell tick: monster id not found (0x4e5550 asserts)")
	}
	ostStub(w, 0x4e3af0, h, 0)
}

// fatigue is 0x52c070(m, cost, 0); src is its source record's address.
func (s *ostState) fatigue(m, cost, src uint32) {
	w := s.w
	if w.u32(m+0x2b) == 0 {
		return
	}
	s.monClean(m)
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
		panic("ongoing spell tick: max SP 0 divides by zero (0x52c0b8)")
	}
	w.put(m+0x242, 4, uint64(staminaFatigueLevel(sp, most)))
	if sp == 0 && w.u32(m+0x9b) < 0x270f {
		ostStub(w, 0x523c00, w.u32(m), 0x11, 0x270f, 0, src, 1)
		s.mon[m] = true
	}
}

// Code below is generated from the same description as re/emu/cases/005526f0.json
// (the spell record decoys come from ostDecoys).

var ostSpec = ostDecoys(mstSpec{
	set: []mstSet{
		{0x65be1c, 4, 0x20040000},
		{0x685174, 4, 0x20012000},
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
		{0x2000264c, 4, 0x20003040},
		{0x200026b8, 4, 0x20003000},
		{0x200026bc, 4, 0x20003010},
		{0x200026cc, 4, 0x20003020},
		{0x20002958, 4, 0x20003050},
		{0x20012005, 2, 0x43},
		{0x20012007, 2, 0x0},
		{0x200090f1, 1, 0xa0},
		{0x20013867, 2, 0x43},
		{0x20013869, 2, 0x43},
		{0x2001386b, 2, 0x0},
		{0x200091f7, 1, 0xa1},
		{0x200150c9, 2, 0x43},
		{0x200150cb, 2, 0x43},
		{0x200150cd, 2, 0x43},
		{0x200150cf, 2, 0x0},
		{0x200092fd, 1, 0xa2},
		{0x2001692b, 2, 0x43},
		{0x2001692d, 2, 0x43},
		{0x2001692f, 2, 0x43},
		{0x20016931, 2, 0x43},
		{0x20016933, 2, 0x0},
		{0x20009403, 1, 0xa3},
		{0x2001818d, 2, 0x43},
		{0x2001818f, 2, 0x43},
		{0x20018191, 2, 0x43},
		{0x20018193, 2, 0x43},
		{0x20018195, 2, 0x43},
		{0x20018197, 2, 0x0},
		{0x20009509, 1, 0xa4},
		{0x200199ef, 2, 0x43},
		{0x200199f1, 2, 0x43},
		{0x200199f3, 2, 0x43},
		{0x200199f5, 2, 0x43},
		{0x200199f7, 2, 0x43},
		{0x200199f9, 2, 0x43},
		{0x200199fb, 2, 0x0},
		{0x2000960f, 1, 0xa5},
		{0x2001b251, 2, 0x43},
		{0x2001b253, 2, 0x43},
		{0x2001b255, 2, 0x43},
		{0x2001b257, 2, 0x43},
		{0x2001b259, 2, 0x43},
		{0x2001b25b, 2, 0x43},
		{0x2001b25d, 2, 0x43},
		{0x2001b25f, 2, 0x0},
		{0x20009715, 1, 0xa6},
		{0x2001cab3, 2, 0x43},
		{0x2001cab5, 2, 0x43},
		{0x2001cab7, 2, 0x43},
		{0x2001cab9, 2, 0x43},
		{0x2001cabb, 2, 0x43},
		{0x2001cabd, 2, 0x43},
		{0x2001cabf, 2, 0x43},
		{0x2001cac1, 2, 0x43},
		{0x2001cac3, 2, 0x0},
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
		{0x200069dc, 4, 0x400},
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
		{0x20006bdc, 4, 0x400},
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
		{0x20003024, 2, 0x0},
	},
	in: []mstFld{
		{0x68510c, 1, false, -1, 0, 0},
		{0x683f94, 1, false, -1, 0, 0},
		{0x6875ef, 4, false, -1, 0, 0},
		{0x6850d5, 4, false, -1, 0, 0},
		{0x20000100, 4, false, -1, 0, 0},
		{0x20000104, 4, true, -1, 0, 0},
		{0x20000108, 4, false, -1, 0, 0},
		{0x20000000, 1, false, -1, 0, 0},
		{0x20000001, 4, false, -1, 0, 0},
		{0x20000005, 4, false, -1, 0, 0},
		{0x20000009, 4, false, -1, 0, 0},
		{0x20040044, 4, false, 8, 96, 0x1bf},
		{0x2004004d, 4, false, 8, 96, 0x1bf},
		{0x20040049, 4, true, 8, 96, 0x1bf},
		{0x20040056, 4, true, 8, 96, 0x1bf},
		{0x20040137, 4, false, 8, 96, 0x1bf},
		{0x20040051, 2, true, 8, 96, 0x1bf},
		{0x20040053, 1, false, 8, 96, 0x1bf},
		{0x20040054, 1, false, 8, 96, 0x1bf},
		{0x20040133, 4, false, 8, 96, 0x1bf},
		{0x20000011, 1, false, -1, 0, 0},
		{0x20000012, 4, false, -1, 0, 0},
		{0x20000016, 4, false, -1, 0, 0},
		{0x2000001a, 4, false, -1, 0, 0},
		{0x20040044, 4, false, 21, 96, 0x1bf},
		{0x2004004d, 4, false, 21, 96, 0x1bf},
		{0x20040049, 4, true, 21, 96, 0x1bf},
		{0x20040056, 4, true, 21, 96, 0x1bf},
		{0x20040137, 4, false, 21, 96, 0x1bf},
		{0x20040051, 2, true, 21, 96, 0x1bf},
		{0x20040053, 1, false, 21, 96, 0x1bf},
		{0x20040054, 1, false, 21, 96, 0x1bf},
		{0x20040133, 4, false, 21, 96, 0x1bf},
		{0x20000022, 1, false, -1, 0, 0},
		{0x20000023, 4, false, -1, 0, 0},
		{0x20000027, 4, false, -1, 0, 0},
		{0x2000002b, 4, false, -1, 0, 0},
		{0x20040044, 4, false, 34, 96, 0x1bf},
		{0x2004004d, 4, false, 34, 96, 0x1bf},
		{0x20040049, 4, true, 34, 96, 0x1bf},
		{0x20040056, 4, true, 34, 96, 0x1bf},
		{0x20040137, 4, false, 34, 96, 0x1bf},
		{0x20040051, 2, true, 34, 96, 0x1bf},
		{0x20040053, 1, false, 34, 96, 0x1bf},
		{0x20040054, 1, false, 34, 96, 0x1bf},
		{0x20040133, 4, false, 34, 96, 0x1bf},
		{0x20000033, 1, false, -1, 0, 0},
		{0x20000034, 4, false, -1, 0, 0},
		{0x20000038, 4, false, -1, 0, 0},
		{0x2000003c, 4, false, -1, 0, 0},
		{0x20040044, 4, false, 47, 96, 0x1bf},
		{0x2004004d, 4, false, 47, 96, 0x1bf},
		{0x20040049, 4, true, 47, 96, 0x1bf},
		{0x20040056, 4, true, 47, 96, 0x1bf},
		{0x20040137, 4, false, 47, 96, 0x1bf},
		{0x20040051, 2, true, 47, 96, 0x1bf},
		{0x20040053, 1, false, 47, 96, 0x1bf},
		{0x20040054, 1, false, 47, 96, 0x1bf},
		{0x20040133, 4, false, 47, 96, 0x1bf},
		{0x20000044, 1, false, -1, 0, 0},
		{0x20000045, 4, false, -1, 0, 0},
		{0x20000049, 4, false, -1, 0, 0},
		{0x2000004d, 4, false, -1, 0, 0},
		{0x20040044, 4, false, 60, 96, 0x1bf},
		{0x2004004d, 4, false, 60, 96, 0x1bf},
		{0x20040049, 4, true, 60, 96, 0x1bf},
		{0x20040056, 4, true, 60, 96, 0x1bf},
		{0x20040137, 4, false, 60, 96, 0x1bf},
		{0x20040051, 2, true, 60, 96, 0x1bf},
		{0x20040053, 1, false, 60, 96, 0x1bf},
		{0x20040054, 1, false, 60, 96, 0x1bf},
		{0x20040133, 4, false, 60, 96, 0x1bf},
		{0x20000055, 1, false, -1, 0, 0},
		{0x20000056, 4, false, -1, 0, 0},
		{0x2000005a, 4, false, -1, 0, 0},
		{0x2000005e, 4, false, -1, 0, 0},
		{0x20040044, 4, false, 73, 96, 0x1bf},
		{0x2004004d, 4, false, 73, 96, 0x1bf},
		{0x20040049, 4, true, 73, 96, 0x1bf},
		{0x20040056, 4, true, 73, 96, 0x1bf},
		{0x20040137, 4, false, 73, 96, 0x1bf},
		{0x20040051, 2, true, 73, 96, 0x1bf},
		{0x20040053, 1, false, 73, 96, 0x1bf},
		{0x20040054, 1, false, 73, 96, 0x1bf},
		{0x20040133, 4, false, 73, 96, 0x1bf},
		{0x20000066, 1, false, -1, 0, 0},
		{0x20000067, 4, false, -1, 0, 0},
		{0x2000006b, 4, false, -1, 0, 0},
		{0x2000006f, 4, false, -1, 0, 0},
		{0x20040044, 4, false, 86, 96, 0x1bf},
		{0x2004004d, 4, false, 86, 96, 0x1bf},
		{0x20040049, 4, true, 86, 96, 0x1bf},
		{0x20040056, 4, true, 86, 96, 0x1bf},
		{0x20040137, 4, false, 86, 96, 0x1bf},
		{0x20040051, 2, true, 86, 96, 0x1bf},
		{0x20040053, 1, false, 86, 96, 0x1bf},
		{0x20040054, 1, false, 86, 96, 0x1bf},
		{0x20040133, 4, false, 86, 96, 0x1bf},
		{0x20000077, 1, false, -1, 0, 0},
		{0x20000078, 4, false, -1, 0, 0},
		{0x2000007c, 4, false, -1, 0, 0},
		{0x20000080, 4, false, -1, 0, 0},
		{0x20040044, 4, false, 99, 96, 0x1bf},
		{0x2004004d, 4, false, 99, 96, 0x1bf},
		{0x20040049, 4, true, 99, 96, 0x1bf},
		{0x20040056, 4, true, 99, 96, 0x1bf},
		{0x20040137, 4, false, 99, 96, 0x1bf},
		{0x20040051, 2, true, 99, 96, 0x1bf},
		{0x20040053, 1, false, 99, 96, 0x1bf},
		{0x20040054, 1, false, 99, 96, 0x1bf},
		{0x20040133, 4, false, 99, 96, 0x1bf},
		{0x20000088, 1, false, -1, 0, 0},
		{0x20000089, 4, false, -1, 0, 0},
		{0x2000008d, 4, false, -1, 0, 0},
		{0x20000091, 4, false, -1, 0, 0},
		{0x20040044, 4, false, 112, 96, 0x1bf},
		{0x2004004d, 4, false, 112, 96, 0x1bf},
		{0x20040049, 4, true, 112, 96, 0x1bf},
		{0x20040056, 4, true, 112, 96, 0x1bf},
		{0x20040137, 4, false, 112, 96, 0x1bf},
		{0x20040051, 2, true, 112, 96, 0x1bf},
		{0x20040053, 1, false, 112, 96, 0x1bf},
		{0x20040054, 1, false, 112, 96, 0x1bf},
		{0x20040133, 4, false, 112, 96, 0x1bf},
		{0x20009000, 1, false, -1, 0, 0},
		{0x20012b01, 4, false, -1, 0, 0},
		{0x20012089, 4, false, -1, 0, 0},
		{0x20012ee1, 4, false, -1, 0, 0},
		{0x20012ef1, 4, false, -1, 0, 0},
		{0x20012f01, 4, false, -1, 0, 0},
		{0x20012f11, 4, false, -1, 0, 0},
		{0x20012f21, 4, false, -1, 0, 0},
		{0x20012f31, 4, false, -1, 0, 0},
		{0x200126f5, 1, false, -1, 0, 0},
		{0x20012a35, 4, false, -1, 0, 0},
		{0x20009106, 1, false, -1, 0, 0},
		{0x20014363, 4, false, -1, 0, 0},
		{0x200138eb, 4, false, -1, 0, 0},
		{0x20014743, 4, false, -1, 0, 0},
		{0x20014753, 4, false, -1, 0, 0},
		{0x20014763, 4, false, -1, 0, 0},
		{0x20014773, 4, false, -1, 0, 0},
		{0x20014783, 4, false, -1, 0, 0},
		{0x20014793, 4, false, -1, 0, 0},
		{0x20013f57, 1, false, -1, 0, 0},
		{0x20014297, 4, false, -1, 0, 0},
		{0x2000920c, 1, false, -1, 0, 0},
		{0x20015bc5, 4, false, -1, 0, 0},
		{0x2001514d, 4, false, -1, 0, 0},
		{0x20015fa5, 4, false, -1, 0, 0},
		{0x20015fb5, 4, false, -1, 0, 0},
		{0x20015fc5, 4, false, -1, 0, 0},
		{0x20015fd5, 4, false, -1, 0, 0},
		{0x20015fe5, 4, false, -1, 0, 0},
		{0x20015ff5, 4, false, -1, 0, 0},
		{0x200157b9, 1, false, -1, 0, 0},
		{0x20015af9, 4, false, -1, 0, 0},
		{0x20009312, 1, false, -1, 0, 0},
		{0x20017427, 4, false, -1, 0, 0},
		{0x200169af, 4, false, -1, 0, 0},
		{0x20017807, 4, false, -1, 0, 0},
		{0x20017817, 4, false, -1, 0, 0},
		{0x20017827, 4, false, -1, 0, 0},
		{0x20017837, 4, false, -1, 0, 0},
		{0x20017847, 4, false, -1, 0, 0},
		{0x20017857, 4, false, -1, 0, 0},
		{0x2001701b, 1, false, -1, 0, 0},
		{0x2001735b, 4, false, -1, 0, 0},
		{0x20009418, 1, false, -1, 0, 0},
		{0x20018c89, 4, false, -1, 0, 0},
		{0x20018211, 4, false, -1, 0, 0},
		{0x20019069, 4, false, -1, 0, 0},
		{0x20019079, 4, false, -1, 0, 0},
		{0x20019089, 4, false, -1, 0, 0},
		{0x20019099, 4, false, -1, 0, 0},
		{0x200190a9, 4, false, -1, 0, 0},
		{0x200190b9, 4, false, -1, 0, 0},
		{0x2001887d, 1, false, -1, 0, 0},
		{0x20018bbd, 4, false, -1, 0, 0},
		{0x2000951e, 1, false, -1, 0, 0},
		{0x2001a4eb, 4, false, -1, 0, 0},
		{0x20019a73, 4, false, -1, 0, 0},
		{0x2001a8cb, 4, false, -1, 0, 0},
		{0x2001a8db, 4, false, -1, 0, 0},
		{0x2001a8eb, 4, false, -1, 0, 0},
		{0x2001a8fb, 4, false, -1, 0, 0},
		{0x2001a90b, 4, false, -1, 0, 0},
		{0x2001a91b, 4, false, -1, 0, 0},
		{0x2001a0df, 1, false, -1, 0, 0},
		{0x2001a41f, 4, false, -1, 0, 0},
		{0x20009624, 1, false, -1, 0, 0},
		{0x2001bd4d, 4, false, -1, 0, 0},
		{0x2001b2d5, 4, false, -1, 0, 0},
		{0x2001c12d, 4, false, -1, 0, 0},
		{0x2001c13d, 4, false, -1, 0, 0},
		{0x2001c14d, 4, false, -1, 0, 0},
		{0x2001c15d, 4, false, -1, 0, 0},
		{0x2001c16d, 4, false, -1, 0, 0},
		{0x2001c17d, 4, false, -1, 0, 0},
		{0x2001b941, 1, false, -1, 0, 0},
		{0x2001bc81, 4, false, -1, 0, 0},
		{0x2000972a, 1, false, -1, 0, 0},
		{0x2001d5af, 4, false, -1, 0, 0},
		{0x2001cb37, 4, false, -1, 0, 0},
		{0x2001d98f, 4, false, -1, 0, 0},
		{0x2001d99f, 4, false, -1, 0, 0},
		{0x2001d9af, 4, false, -1, 0, 0},
		{0x2001d9bf, 4, false, -1, 0, 0},
		{0x2001d9cf, 4, false, -1, 0, 0},
		{0x2001d9df, 4, false, -1, 0, 0},
		{0x2001d1a3, 1, false, -1, 0, 0},
		{0x2001d4e3, 4, false, -1, 0, 0},
		{0x20010827, 4, false, -1, 0, 0},
		{0x2001167f, 4, false, -1, 0, 0},
		{0x2001168f, 4, false, -1, 0, 0},
		{0x2001169f, 4, false, -1, 0, 0},
		{0x200116af, 4, false, -1, 0, 0},
		{0x200116bf, 4, false, -1, 0, 0},
		{0x200116cf, 4, false, -1, 0, 0},
		{0x2000602b, 4, false, -1, 0, 0},
		{0x20006027, 4, false, -1, 0, 0},
		{0x200060c5, 2, false, -1, 0, 0},
		{0x200060c7, 4, false, -1, 0, 0},
		{0x2000602f, 4, false, -1, 0, 0},
		{0x20006033, 4, false, -1, 0, 0},
		{0x2000609b, 4, false, -1, 0, 0},
		{0x200062ab, 1, false, -1, 0, 0},
		{0x20006004, 4, false, -1, 0, 0},
		{0x200070a4, 1, false, -1, 0, 0},
		{0x20005187, 2, false, -1, 0, 0},
		{0x200062f9, 4, false, -1, 0, 0},
		{0x20006217, 1, true, -1, 0, 0},
		{0x20006218, 1, true, -1, 0, 0},
		{0x20006219, 1, true, -1, 0, 0},
		{0x2000621a, 1, true, -1, 0, 0},
		{0x2000621b, 1, true, -1, 0, 0},
		{0x2000621c, 1, true, -1, 0, 0},
		{0x20006107, 4, false, -1, 0, 0},
		{0x20006016, 1, false, -1, 0, 0},
		{0x200051be, 1, false, -1, 0, 0},
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
		{0x200066ab, 1, false, -1, 0, 0},
		{0x20006404, 4, false, -1, 0, 0},
		{0x200078a4, 1, false, -1, 0, 0},
		{0x20005587, 2, false, -1, 0, 0},
		{0x200066f9, 4, false, -1, 0, 0},
		{0x20006617, 1, true, -1, 0, 0},
		{0x20006618, 1, true, -1, 0, 0},
		{0x20006619, 1, true, -1, 0, 0},
		{0x2000661a, 1, true, -1, 0, 0},
		{0x2000661b, 1, true, -1, 0, 0},
		{0x2000661c, 1, true, -1, 0, 0},
		{0x20006507, 4, false, -1, 0, 0},
		{0x20006416, 1, false, -1, 0, 0},
		{0x200055be, 1, false, -1, 0, 0},
		{0x20005576, 1, false, -1, 0, 0},
		{0x20005577, 1, false, -1, 0, 0},
		{0x20005578, 1, false, -1, 0, 0},
		{0x20005579, 1, false, -1, 0, 0},
		{0x2000557a, 1, false, -1, 0, 0},
		{0x2000557b, 1, false, -1, 0, 0},
		{0x2000564f, 1, false, -1, 0, 0},
		{0x2000352c, 1, false, -1, 0, 0},
		{0x20003020, 2, false, -1, 0, 0},
		{0x20003022, 2, false, -1, 0, 0},
		{0x20030000, 4, false, -1, 0, 0},
		{0x20030004, 4, false, -1, 0, 0},
		{0x20030008, 4, false, -1, 0, 0},
		{0x2003000c, 4, false, -1, 0, 0},
		{0x20030010, 4, false, -1, 0, 0},
		{0x20030014, 4, false, -1, 0, 0},
		{0x20030018, 4, false, -1, 0, 0},
		{0x2003001c, 4, false, -1, 0, 0},
		{0x20030020, 4, false, -1, 0, 0},
		{0x20030024, 4, false, -1, 0, 0},
		{0x20030028, 4, false, -1, 0, 0},
		{0x2003002c, 4, false, -1, 0, 0},
		{0x20030030, 4, false, -1, 0, 0},
		{0x20030034, 4, false, -1, 0, 0},
		{0x20030038, 4, false, -1, 0, 0},
		{0x2003003c, 4, false, -1, 0, 0},
		{0x20030040, 4, false, -1, 0, 0},
		{0x20030044, 4, false, -1, 0, 0},
		{0x20030048, 4, false, -1, 0, 0},
		{0x2003004c, 4, false, -1, 0, 0},
		{0x20030050, 4, false, -1, 0, 0},
		{0x20030054, 4, false, -1, 0, 0},
		{0x20030058, 4, false, -1, 0, 0},
		{0x2003005c, 4, false, -1, 0, 0},
		{0x20030060, 4, false, -1, 0, 0},
		{0x20030064, 4, false, -1, 0, 0},
		{0x20030068, 4, false, -1, 0, 0},
		{0x2003006c, 4, false, -1, 0, 0},
		{0x20030070, 4, false, -1, 0, 0},
		{0x20030074, 4, false, -1, 0, 0},
		{0x20030078, 4, false, -1, 0, 0},
		{0x2003007c, 4, false, -1, 0, 0},
		{0x20030080, 4, false, -1, 0, 0},
		{0x20030084, 4, false, -1, 0, 0},
		{0x20030088, 4, false, -1, 0, 0},
		{0x2003008c, 4, false, -1, 0, 0},
		{0x20030090, 4, false, -1, 0, 0},
		{0x20030094, 4, false, -1, 0, 0},
		{0x20030098, 4, false, -1, 0, 0},
		{0x2003009c, 4, false, -1, 0, 0},
		{0x200300a0, 4, false, -1, 0, 0},
		{0x200300a4, 4, false, -1, 0, 0},
		{0x200300a8, 4, false, -1, 0, 0},
		{0x200300ac, 4, false, -1, 0, 0},
		{0x200300b0, 4, false, -1, 0, 0},
		{0x200300b4, 4, false, -1, 0, 0},
		{0x200300b8, 4, false, -1, 0, 0},
		{0x200300bc, 4, false, -1, 0, 0},
		{0x200300c0, 4, false, -1, 0, 0},
		{0x200300c4, 4, false, -1, 0, 0},
		{0x200300c8, 4, false, -1, 0, 0},
		{0x200300cc, 4, false, -1, 0, 0},
		{0x200300d0, 4, false, -1, 0, 0},
		{0x200300d4, 4, false, -1, 0, 0},
		{0x200300d8, 4, false, -1, 0, 0},
		{0x200300dc, 4, false, -1, 0, 0},
		{0x200300e0, 4, false, -1, 0, 0},
		{0x200300e4, 4, false, -1, 0, 0},
		{0x200300e8, 4, false, -1, 0, 0},
		{0x200300ec, 4, false, -1, 0, 0},
		{0x200300f0, 4, false, -1, 0, 0},
		{0x200300f4, 4, false, -1, 0, 0},
		{0x200300f8, 4, false, -1, 0, 0},
		{0x200300fc, 4, false, -1, 0, 0},
		{0x20030100, 4, false, -1, 0, 0},
	},
	out: []mstFld{
		{0x20030104, 4, false, -1, 0, 0},
		{0x20030108, 4, false, -1, 0, 0},
		{0x2003010c, 4, false, -1, 0, 0},
		{0x20030110, 4, false, -1, 0, 0},
		{0x20030114, 4, false, -1, 0, 0},
		{0x20030118, 4, false, -1, 0, 0},
		{0x2003011c, 4, false, -1, 0, 0},
		{0x20030120, 4, false, -1, 0, 0},
		{0x20030124, 4, false, -1, 0, 0},
		{0x20030128, 4, false, -1, 0, 0},
		{0x2003012c, 4, false, -1, 0, 0},
		{0x20030130, 4, false, -1, 0, 0},
		{0x20030134, 4, false, -1, 0, 0},
		{0x20030138, 4, false, -1, 0, 0},
		{0x2003013c, 4, false, -1, 0, 0},
		{0x20030140, 4, false, -1, 0, 0},
		{0x20030144, 4, false, -1, 0, 0},
		{0x20030148, 4, false, -1, 0, 0},
		{0x2003014c, 4, false, -1, 0, 0},
		{0x20030150, 4, false, -1, 0, 0},
		{0x20030154, 4, false, -1, 0, 0},
		{0x20030158, 4, false, -1, 0, 0},
		{0x2003015c, 4, false, -1, 0, 0},
		{0x20030160, 4, false, -1, 0, 0},
		{0x20030164, 4, false, -1, 0, 0},
		{0x20030168, 4, false, -1, 0, 0},
		{0x2003016c, 4, false, -1, 0, 0},
		{0x20030170, 4, false, -1, 0, 0},
		{0x20030174, 4, false, -1, 0, 0},
		{0x20030178, 4, false, -1, 0, 0},
		{0x2003017c, 4, false, -1, 0, 0},
		{0x20030180, 4, false, -1, 0, 0},
		{0x20030184, 4, false, -1, 0, 0},
		{0x20030188, 4, false, -1, 0, 0},
		{0x2003018c, 4, false, -1, 0, 0},
		{0x20030190, 4, false, -1, 0, 0},
		{0x20030194, 4, false, -1, 0, 0},
		{0x20030198, 4, false, -1, 0, 0},
		{0x2003019c, 4, false, -1, 0, 0},
		{0x200301a0, 4, false, -1, 0, 0},
		{0x200301a4, 4, false, -1, 0, 0},
		{0x200301a8, 4, false, -1, 0, 0},
		{0x200301ac, 4, false, -1, 0, 0},
		{0x200301b0, 4, false, -1, 0, 0},
		{0x200301b4, 4, false, -1, 0, 0},
		{0x200301b8, 4, false, -1, 0, 0},
		{0x200301bc, 4, false, -1, 0, 0},
		{0x200301c0, 4, false, -1, 0, 0},
		{0x200301c4, 4, false, -1, 0, 0},
		{0x200301c8, 4, false, -1, 0, 0},
		{0x200301cc, 4, false, -1, 0, 0},
		{0x200301d0, 4, false, -1, 0, 0},
		{0x200301d4, 4, false, -1, 0, 0},
		{0x200301d8, 4, false, -1, 0, 0},
		{0x200301dc, 4, false, -1, 0, 0},
		{0x200301e0, 4, false, -1, 0, 0},
		{0x200301e4, 4, false, -1, 0, 0},
		{0x200301e8, 4, false, -1, 0, 0},
		{0x200301ec, 4, false, -1, 0, 0},
		{0x200301f0, 4, false, -1, 0, 0},
		{0x200301f4, 4, false, -1, 0, 0},
		{0x200301f8, 4, false, -1, 0, 0},
		{0x200301fc, 4, false, -1, 0, 0},
		{0x20030200, 4, false, -1, 0, 0},
		{0x20030204, 4, false, -1, 0, 0},
		{0x20030208, 4, false, -1, 0, 0},
		{0x2003020c, 4, false, -1, 0, 0},
		{0x20030210, 4, false, -1, 0, 0},
		{0x20030214, 4, false, -1, 0, 0},
		{0x20030218, 4, false, -1, 0, 0},
		{0x2003021c, 4, false, -1, 0, 0},
		{0x20030220, 4, false, -1, 0, 0},
		{0x20030224, 4, false, -1, 0, 0},
		{0x20030228, 4, false, -1, 0, 0},
		{0x2003022c, 4, false, -1, 0, 0},
		{0x20030230, 4, false, -1, 0, 0},
		{0x20030234, 4, false, -1, 0, 0},
		{0x20030238, 4, false, -1, 0, 0},
		{0x2003023c, 4, false, -1, 0, 0},
		{0x20030240, 4, false, -1, 0, 0},
		{0x20030244, 4, false, -1, 0, 0},
		{0x20030248, 4, false, -1, 0, 0},
		{0x2003024c, 4, false, -1, 0, 0},
		{0x20030250, 4, false, -1, 0, 0},
		{0x20030254, 4, false, -1, 0, 0},
		{0x20030258, 4, false, -1, 0, 0},
		{0x2003025c, 4, false, -1, 0, 0},
		{0x20030260, 4, false, -1, 0, 0},
		{0x20030264, 4, false, -1, 0, 0},
		{0x20030268, 4, false, -1, 0, 0},
		{0x2003026c, 4, false, -1, 0, 0},
		{0x20030270, 4, false, -1, 0, 0},
		{0x20030274, 4, false, -1, 0, 0},
		{0x20030278, 4, false, -1, 0, 0},
		{0x2003027c, 4, false, -1, 0, 0},
		{0x20030280, 4, false, -1, 0, 0},
		{0x20030284, 4, false, -1, 0, 0},
		{0x20030288, 4, false, -1, 0, 0},
		{0x2003028c, 4, false, -1, 0, 0},
		{0x20030290, 4, false, -1, 0, 0},
		{0x20030294, 4, false, -1, 0, 0},
		{0x20030298, 4, false, -1, 0, 0},
		{0x2003029c, 4, false, -1, 0, 0},
		{0x200302a0, 4, false, -1, 0, 0},
		{0x200302a4, 4, false, -1, 0, 0},
		{0x200302a8, 4, false, -1, 0, 0},
		{0x200302ac, 4, false, -1, 0, 0},
		{0x200302b0, 4, false, -1, 0, 0},
		{0x200302b4, 4, false, -1, 0, 0},
		{0x200302b8, 4, false, -1, 0, 0},
		{0x200302bc, 4, false, -1, 0, 0},
		{0x200302c0, 4, false, -1, 0, 0},
		{0x200302c4, 4, false, -1, 0, 0},
		{0x200302c8, 4, false, -1, 0, 0},
		{0x200302cc, 4, false, -1, 0, 0},
		{0x200302d0, 4, false, -1, 0, 0},
		{0x200302d4, 4, false, -1, 0, 0},
		{0x200302d8, 4, false, -1, 0, 0},
		{0x200302dc, 4, false, -1, 0, 0},
		{0x200302e0, 4, false, -1, 0, 0},
		{0x200302e4, 4, false, -1, 0, 0},
		{0x200302e8, 4, false, -1, 0, 0},
		{0x200302ec, 4, false, -1, 0, 0},
		{0x200302f0, 4, false, -1, 0, 0},
		{0x200302f4, 4, false, -1, 0, 0},
		{0x200302f8, 4, false, -1, 0, 0},
		{0x200302fc, 4, false, -1, 0, 0},
		{0x20030300, 4, false, -1, 0, 0},
		{0x20030304, 4, false, -1, 0, 0},
		{0x20030308, 4, false, -1, 0, 0},
		{0x2003030c, 4, false, -1, 0, 0},
		{0x20030310, 4, false, -1, 0, 0},
		{0x20030314, 4, false, -1, 0, 0},
		{0x20030318, 4, false, -1, 0, 0},
		{0x2003031c, 4, false, -1, 0, 0},
		{0x20030320, 4, false, -1, 0, 0},
		{0x20030324, 4, false, -1, 0, 0},
		{0x20030328, 4, false, -1, 0, 0},
		{0x2003032c, 4, false, -1, 0, 0},
		{0x20030330, 4, false, -1, 0, 0},
		{0x20030334, 4, false, -1, 0, 0},
		{0x20030338, 4, false, -1, 0, 0},
		{0x2003033c, 4, false, -1, 0, 0},
		{0x20030340, 4, false, -1, 0, 0},
		{0x20030344, 4, false, -1, 0, 0},
		{0x20030348, 4, false, -1, 0, 0},
		{0x2003034c, 4, false, -1, 0, 0},
		{0x20030350, 4, false, -1, 0, 0},
		{0x20030354, 4, false, -1, 0, 0},
		{0x20030358, 4, false, -1, 0, 0},
		{0x2003035c, 4, false, -1, 0, 0},
		{0x20030360, 4, false, -1, 0, 0},
		{0x20030364, 4, false, -1, 0, 0},
		{0x20030368, 4, false, -1, 0, 0},
		{0x2003036c, 4, false, -1, 0, 0},
		{0x20030370, 4, false, -1, 0, 0},
		{0x20030374, 4, false, -1, 0, 0},
		{0x20030378, 4, false, -1, 0, 0},
		{0x2003037c, 4, false, -1, 0, 0},
		{0x20030380, 4, false, -1, 0, 0},
		{0x20030384, 4, false, -1, 0, 0},
		{0x20030388, 4, false, -1, 0, 0},
		{0x2003038c, 4, false, -1, 0, 0},
		{0x20030390, 4, false, -1, 0, 0},
		{0x20030394, 4, false, -1, 0, 0},
		{0x20030398, 4, false, -1, 0, 0},
		{0x2003039c, 4, false, -1, 0, 0},
		{0x200303a0, 4, false, -1, 0, 0},
		{0x200303a4, 4, false, -1, 0, 0},
		{0x200303a8, 4, false, -1, 0, 0},
		{0x200303ac, 4, false, -1, 0, 0},
		{0x200303b0, 4, false, -1, 0, 0},
		{0x200303b4, 4, false, -1, 0, 0},
		{0x200303b8, 4, false, -1, 0, 0},
		{0x200303bc, 4, false, -1, 0, 0},
		{0x200303c0, 4, false, -1, 0, 0},
		{0x200303c4, 4, false, -1, 0, 0},
		{0x200303c8, 4, false, -1, 0, 0},
		{0x200303cc, 4, false, -1, 0, 0},
		{0x200303d0, 4, false, -1, 0, 0},
		{0x200303d4, 4, false, -1, 0, 0},
		{0x200303d8, 4, false, -1, 0, 0},
		{0x200303dc, 4, false, -1, 0, 0},
		{0x200303e0, 4, false, -1, 0, 0},
		{0x200303e4, 4, false, -1, 0, 0},
		{0x200303e8, 4, false, -1, 0, 0},
		{0x200303ec, 4, false, -1, 0, 0},
		{0x200303f0, 4, false, -1, 0, 0},
		{0x200303f4, 4, false, -1, 0, 0},
		{0x200303f8, 4, false, -1, 0, 0},
		{0x200303fc, 4, false, -1, 0, 0},
		{0x20030400, 4, false, -1, 0, 0},
		{0x20030404, 4, false, -1, 0, 0},
		{0x20030408, 4, false, -1, 0, 0},
		{0x2003040c, 4, false, -1, 0, 0},
		{0x20030410, 4, false, -1, 0, 0},
		{0x20030414, 4, false, -1, 0, 0},
		{0x20030418, 4, false, -1, 0, 0},
		{0x2003041c, 4, false, -1, 0, 0},
		{0x20030420, 4, false, -1, 0, 0},
		{0x20030424, 4, false, -1, 0, 0},
		{0x20030428, 4, false, -1, 0, 0},
		{0x2003042c, 4, false, -1, 0, 0},
		{0x20030430, 4, false, -1, 0, 0},
		{0x20030434, 4, false, -1, 0, 0},
		{0x20030438, 4, false, -1, 0, 0},
		{0x2003043c, 4, false, -1, 0, 0},
		{0x20030440, 4, false, -1, 0, 0},
		{0x20030444, 4, false, -1, 0, 0},
		{0x20030448, 4, false, -1, 0, 0},
		{0x2003044c, 4, false, -1, 0, 0},
		{0x20030450, 4, false, -1, 0, 0},
		{0x20030454, 4, false, -1, 0, 0},
		{0x20030458, 4, false, -1, 0, 0},
		{0x2003045c, 4, false, -1, 0, 0},
		{0x20030460, 4, false, -1, 0, 0},
		{0x20030464, 4, false, -1, 0, 0},
		{0x20030468, 4, false, -1, 0, 0},
		{0x2003046c, 4, false, -1, 0, 0},
		{0x20030470, 4, false, -1, 0, 0},
		{0x20030474, 4, false, -1, 0, 0},
		{0x20030478, 4, false, -1, 0, 0},
		{0x2003047c, 4, false, -1, 0, 0},
		{0x20030480, 4, false, -1, 0, 0},
		{0x20030484, 4, false, -1, 0, 0},
		{0x20030488, 4, false, -1, 0, 0},
		{0x2003048c, 4, false, -1, 0, 0},
		{0x20030490, 4, false, -1, 0, 0},
		{0x20030494, 4, false, -1, 0, 0},
		{0x20030498, 4, false, -1, 0, 0},
		{0x2003049c, 4, false, -1, 0, 0},
		{0x200304a0, 4, false, -1, 0, 0},
		{0x200304a4, 4, false, -1, 0, 0},
		{0x200304a8, 4, false, -1, 0, 0},
		{0x200304ac, 4, false, -1, 0, 0},
		{0x200304b0, 4, false, -1, 0, 0},
		{0x200304b4, 4, false, -1, 0, 0},
		{0x200304b8, 4, false, -1, 0, 0},
		{0x200304bc, 4, false, -1, 0, 0},
		{0x200304c0, 4, false, -1, 0, 0},
		{0x200304c4, 4, false, -1, 0, 0},
		{0x200304c8, 4, false, -1, 0, 0},
		{0x200304cc, 4, false, -1, 0, 0},
		{0x200304d0, 4, false, -1, 0, 0},
		{0x200304d4, 4, false, -1, 0, 0},
		{0x200304d8, 4, false, -1, 0, 0},
		{0x200304dc, 4, false, -1, 0, 0},
		{0x200304e0, 4, false, -1, 0, 0},
		{0x200304e4, 4, false, -1, 0, 0},
		{0x200304e8, 4, false, -1, 0, 0},
		{0x200304ec, 4, false, -1, 0, 0},
		{0x200304f0, 4, false, -1, 0, 0},
		{0x200304f4, 4, false, -1, 0, 0},
		{0x200304f8, 4, false, -1, 0, 0},
		{0x200304fc, 4, false, -1, 0, 0},
		{0x20030500, 4, false, -1, 0, 0},
		{0x20030504, 4, false, -1, 0, 0},
		{0x20030508, 4, false, -1, 0, 0},
		{0x2003050c, 4, false, -1, 0, 0},
		{0x20030510, 4, false, -1, 0, 0},
		{0x20030514, 4, false, -1, 0, 0},
		{0x20030518, 4, false, -1, 0, 0},
		{0x2003051c, 4, false, -1, 0, 0},
		{0x20030520, 4, false, -1, 0, 0},
		{0x20030524, 4, false, -1, 0, 0},
		{0x20030528, 4, false, -1, 0, 0},
		{0x2003052c, 4, false, -1, 0, 0},
		{0x20030530, 4, false, -1, 0, 0},
		{0x20030534, 4, false, -1, 0, 0},
		{0x20030538, 4, false, -1, 0, 0},
		{0x2003053c, 4, false, -1, 0, 0},
		{0x20030540, 4, false, -1, 0, 0},
		{0x20030544, 4, false, -1, 0, 0},
		{0x20030548, 4, false, -1, 0, 0},
		{0x2003054c, 4, false, -1, 0, 0},
		{0x20030550, 4, false, -1, 0, 0},
		{0x20030554, 4, false, -1, 0, 0},
		{0x20030558, 4, false, -1, 0, 0},
		{0x2003055c, 4, false, -1, 0, 0},
		{0x20030560, 4, false, -1, 0, 0},
		{0x20030564, 4, false, -1, 0, 0},
		{0x20030568, 4, false, -1, 0, 0},
		{0x2003056c, 4, false, -1, 0, 0},
		{0x20030570, 4, false, -1, 0, 0},
		{0x20030574, 4, false, -1, 0, 0},
		{0x20030578, 4, false, -1, 0, 0},
		{0x2003057c, 4, false, -1, 0, 0},
		{0x20030580, 4, false, -1, 0, 0},
		{0x20030584, 4, false, -1, 0, 0},
		{0x20030588, 4, false, -1, 0, 0},
		{0x2003058c, 4, false, -1, 0, 0},
		{0x20030590, 4, false, -1, 0, 0},
		{0x20030594, 4, false, -1, 0, 0},
		{0x20030598, 4, false, -1, 0, 0},
		{0x2003059c, 4, false, -1, 0, 0},
		{0x200305a0, 4, false, -1, 0, 0},
		{0x200305a4, 4, false, -1, 0, 0},
		{0x200305a8, 4, false, -1, 0, 0},
		{0x200305ac, 4, false, -1, 0, 0},
		{0x200305b0, 4, false, -1, 0, 0},
		{0x200305b4, 4, false, -1, 0, 0},
		{0x200305b8, 4, false, -1, 0, 0},
		{0x200305bc, 4, false, -1, 0, 0},
		{0x200305c0, 4, false, -1, 0, 0},
		{0x200305c4, 4, false, -1, 0, 0},
		{0x200305c8, 4, false, -1, 0, 0},
		{0x200305cc, 4, false, -1, 0, 0},
		{0x200305d0, 4, false, -1, 0, 0},
		{0x200305d4, 4, false, -1, 0, 0},
		{0x200305d8, 4, false, -1, 0, 0},
		{0x200305dc, 4, false, -1, 0, 0},
		{0x200305e0, 4, false, -1, 0, 0},
		{0x200305e4, 4, false, -1, 0, 0},
		{0x200305e8, 4, false, -1, 0, 0},
		{0x200305ec, 4, false, -1, 0, 0},
		{0x200305f0, 4, false, -1, 0, 0},
		{0x200305f4, 4, false, -1, 0, 0},
		{0x200305f8, 4, false, -1, 0, 0},
		{0x200305fc, 4, false, -1, 0, 0},
		{0x20030600, 4, false, -1, 0, 0},
		{0x20030604, 4, false, -1, 0, 0},
		{0x20030608, 4, false, -1, 0, 0},
		{0x2003060c, 4, false, -1, 0, 0},
		{0x20030610, 4, false, -1, 0, 0},
		{0x20030614, 4, false, -1, 0, 0},
		{0x20030618, 4, false, -1, 0, 0},
		{0x2003061c, 4, false, -1, 0, 0},
		{0x20030620, 4, false, -1, 0, 0},
		{0x20030624, 4, false, -1, 0, 0},
		{0x20030628, 4, false, -1, 0, 0},
		{0x2003062c, 4, false, -1, 0, 0},
		{0x20030630, 4, false, -1, 0, 0},
		{0x20030634, 4, false, -1, 0, 0},
		{0x20030638, 4, false, -1, 0, 0},
		{0x2003063c, 4, false, -1, 0, 0},
		{0x20030640, 4, false, -1, 0, 0},
		{0x20030644, 4, false, -1, 0, 0},
		{0x20030648, 4, false, -1, 0, 0},
		{0x2003064c, 4, false, -1, 0, 0},
		{0x20030650, 4, false, -1, 0, 0},
		{0x20030654, 4, false, -1, 0, 0},
		{0x20030658, 4, false, -1, 0, 0},
		{0x2003065c, 4, false, -1, 0, 0},
		{0x20030660, 4, false, -1, 0, 0},
		{0x20030664, 4, false, -1, 0, 0},
		{0x20030668, 4, false, -1, 0, 0},
		{0x2003066c, 4, false, -1, 0, 0},
		{0x20030670, 4, false, -1, 0, 0},
		{0x20030674, 4, false, -1, 0, 0},
		{0x20030678, 4, false, -1, 0, 0},
		{0x2003067c, 4, false, -1, 0, 0},
		{0x20030680, 4, false, -1, 0, 0},
		{0x20030684, 4, false, -1, 0, 0},
		{0x20030688, 4, false, -1, 0, 0},
		{0x2003068c, 4, false, -1, 0, 0},
		{0x20030690, 4, false, -1, 0, 0},
		{0x20030694, 4, false, -1, 0, 0},
		{0x20030698, 4, false, -1, 0, 0},
		{0x2003069c, 4, false, -1, 0, 0},
		{0x200306a0, 4, false, -1, 0, 0},
		{0x200306a4, 4, false, -1, 0, 0},
		{0x200306a8, 4, false, -1, 0, 0},
		{0x200306ac, 4, false, -1, 0, 0},
		{0x200306b0, 4, false, -1, 0, 0},
		{0x200306b4, 4, false, -1, 0, 0},
		{0x200306b8, 4, false, -1, 0, 0},
		{0x200306bc, 4, false, -1, 0, 0},
		{0x200306c0, 4, false, -1, 0, 0},
		{0x200306c4, 4, false, -1, 0, 0},
		{0x200306c8, 4, false, -1, 0, 0},
		{0x200306cc, 4, false, -1, 0, 0},
		{0x200306d0, 4, false, -1, 0, 0},
		{0x200306d4, 4, false, -1, 0, 0},
		{0x200306d8, 4, false, -1, 0, 0},
		{0x200306dc, 4, false, -1, 0, 0},
		{0x200306e0, 4, false, -1, 0, 0},
		{0x200306e4, 4, false, -1, 0, 0},
		{0x200306e8, 4, false, -1, 0, 0},
		{0x200306ec, 4, false, -1, 0, 0},
		{0x200306f0, 4, false, -1, 0, 0},
		{0x200306f4, 4, false, -1, 0, 0},
		{0x200306f8, 4, false, -1, 0, 0},
		{0x200306fc, 4, false, -1, 0, 0},
		{0x20030700, 4, false, -1, 0, 0},
		{0x20030704, 4, false, -1, 0, 0},
		{0x20030708, 4, false, -1, 0, 0},
		{0x2003070c, 4, false, -1, 0, 0},
		{0x20030710, 4, false, -1, 0, 0},
		{0x20030714, 4, false, -1, 0, 0},
		{0x20030718, 4, false, -1, 0, 0},
		{0x2003071c, 4, false, -1, 0, 0},
		{0x20030720, 4, false, -1, 0, 0},
		{0x20030724, 4, false, -1, 0, 0},
		{0x20030728, 4, false, -1, 0, 0},
		{0x2003072c, 4, false, -1, 0, 0},
		{0x20030730, 4, false, -1, 0, 0},
		{0x20030734, 4, false, -1, 0, 0},
		{0x20030738, 4, false, -1, 0, 0},
		{0x2003073c, 4, false, -1, 0, 0},
		{0x20030740, 4, false, -1, 0, 0},
		{0x20030744, 4, false, -1, 0, 0},
		{0x20030748, 4, false, -1, 0, 0},
		{0x2003074c, 4, false, -1, 0, 0},
		{0x20030750, 4, false, -1, 0, 0},
		{0x20030754, 4, false, -1, 0, 0},
		{0x20030758, 4, false, -1, 0, 0},
		{0x2003075c, 4, false, -1, 0, 0},
		{0x20030760, 4, false, -1, 0, 0},
		{0x20030764, 4, false, -1, 0, 0},
		{0x20030768, 4, false, -1, 0, 0},
		{0x2003076c, 4, false, -1, 0, 0},
		{0x20030770, 4, false, -1, 0, 0},
		{0x20030774, 4, false, -1, 0, 0},
		{0x20030778, 4, false, -1, 0, 0},
		{0x2003077c, 4, false, -1, 0, 0},
		{0x20030780, 4, false, -1, 0, 0},
		{0x20030784, 4, false, -1, 0, 0},
		{0x20030788, 4, false, -1, 0, 0},
		{0x2003078c, 4, false, -1, 0, 0},
		{0x20030790, 4, false, -1, 0, 0},
		{0x20030794, 4, false, -1, 0, 0},
		{0x20030798, 4, false, -1, 0, 0},
		{0x2003079c, 4, false, -1, 0, 0},
		{0x200307a0, 4, false, -1, 0, 0},
		{0x200307a4, 4, false, -1, 0, 0},
		{0x200307a8, 4, false, -1, 0, 0},
		{0x200307ac, 4, false, -1, 0, 0},
		{0x200307b0, 4, false, -1, 0, 0},
		{0x200307b4, 4, false, -1, 0, 0},
		{0x200307b8, 4, false, -1, 0, 0},
		{0x200307bc, 4, false, -1, 0, 0},
		{0x200307c0, 4, false, -1, 0, 0},
		{0x200307c4, 4, false, -1, 0, 0},
		{0x200307c8, 4, false, -1, 0, 0},
		{0x200307cc, 4, false, -1, 0, 0},
		{0x200307d0, 4, false, -1, 0, 0},
		{0x200307d4, 4, false, -1, 0, 0},
		{0x200307d8, 4, false, -1, 0, 0},
		{0x200307dc, 4, false, -1, 0, 0},
		{0x200307e0, 4, false, -1, 0, 0},
		{0x200307e4, 4, false, -1, 0, 0},
		{0x200307e8, 4, false, -1, 0, 0},
		{0x200307ec, 4, false, -1, 0, 0},
		{0x200307f0, 4, false, -1, 0, 0},
		{0x200307f4, 4, false, -1, 0, 0},
		{0x200307f8, 4, false, -1, 0, 0},
		{0x200307fc, 4, false, -1, 0, 0},
		{0x20030800, 4, false, -1, 0, 0},
		{0x20030804, 4, false, -1, 0, 0},
		{0x20030808, 4, false, -1, 0, 0},
		{0x2003080c, 4, false, -1, 0, 0},
		{0x20030810, 4, false, -1, 0, 0},
		{0x20030814, 4, false, -1, 0, 0},
		{0x20030818, 4, false, -1, 0, 0},
		{0x2003081c, 4, false, -1, 0, 0},
		{0x20030820, 4, false, -1, 0, 0},
		{0x20030824, 4, false, -1, 0, 0},
		{0x20030828, 4, false, -1, 0, 0},
		{0x2003082c, 4, false, -1, 0, 0},
		{0x20030830, 4, false, -1, 0, 0},
		{0x20030834, 4, false, -1, 0, 0},
		{0x20030838, 4, false, -1, 0, 0},
		{0x2003083c, 4, false, -1, 0, 0},
		{0x20030840, 4, false, -1, 0, 0},
		{0x20030844, 4, false, -1, 0, 0},
		{0x20030848, 4, false, -1, 0, 0},
		{0x2003084c, 4, false, -1, 0, 0},
		{0x20030850, 4, false, -1, 0, 0},
		{0x20030854, 4, false, -1, 0, 0},
		{0x20030858, 4, false, -1, 0, 0},
		{0x2003085c, 4, false, -1, 0, 0},
		{0x20030860, 4, false, -1, 0, 0},
		{0x20030864, 4, false, -1, 0, 0},
		{0x20030868, 4, false, -1, 0, 0},
		{0x2003086c, 4, false, -1, 0, 0},
		{0x20030870, 4, false, -1, 0, 0},
		{0x20030874, 4, false, -1, 0, 0},
		{0x20030878, 4, false, -1, 0, 0},
		{0x2003087c, 4, false, -1, 0, 0},
		{0x20030880, 4, false, -1, 0, 0},
		{0x20030884, 4, false, -1, 0, 0},
		{0x20030888, 4, false, -1, 0, 0},
		{0x2003088c, 4, false, -1, 0, 0},
		{0x20030890, 4, false, -1, 0, 0},
		{0x20030894, 4, false, -1, 0, 0},
		{0x20030898, 4, false, -1, 0, 0},
		{0x2003089c, 4, false, -1, 0, 0},
		{0x200308a0, 4, false, -1, 0, 0},
		{0x200308a4, 4, false, -1, 0, 0},
		{0x200308a8, 4, false, -1, 0, 0},
		{0x200308ac, 4, false, -1, 0, 0},
		{0x200308b0, 4, false, -1, 0, 0},
		{0x200308b4, 4, false, -1, 0, 0},
		{0x200308b8, 4, false, -1, 0, 0},
		{0x200308bc, 4, false, -1, 0, 0},
		{0x200308c0, 4, false, -1, 0, 0},
		{0x200308c4, 4, false, -1, 0, 0},
		{0x200308c8, 4, false, -1, 0, 0},
		{0x200308cc, 4, false, -1, 0, 0},
		{0x200308d0, 4, false, -1, 0, 0},
		{0x200308d4, 4, false, -1, 0, 0},
		{0x200308d8, 4, false, -1, 0, 0},
		{0x200308dc, 4, false, -1, 0, 0},
		{0x200308e0, 4, false, -1, 0, 0},
		{0x200308e4, 4, false, -1, 0, 0},
		{0x200308e8, 4, false, -1, 0, 0},
		{0x200308ec, 4, false, -1, 0, 0},
		{0x200308f0, 4, false, -1, 0, 0},
		{0x200308f4, 4, false, -1, 0, 0},
		{0x200308f8, 4, false, -1, 0, 0},
		{0x200308fc, 4, false, -1, 0, 0},
		{0x20030900, 4, false, -1, 0, 0},
		{0x20030904, 4, false, -1, 0, 0},
		{0x20030908, 4, false, -1, 0, 0},
		{0x2003090c, 4, false, -1, 0, 0},
		{0x20030910, 4, false, -1, 0, 0},
		{0x20030914, 4, false, -1, 0, 0},
		{0x20030918, 4, false, -1, 0, 0},
		{0x2003091c, 4, false, -1, 0, 0},
		{0x20030920, 4, false, -1, 0, 0},
		{0x20030924, 4, false, -1, 0, 0},
		{0x20030928, 4, false, -1, 0, 0},
		{0x2003092c, 4, false, -1, 0, 0},
		{0x20030930, 4, false, -1, 0, 0},
		{0x20030934, 4, false, -1, 0, 0},
		{0x20030938, 4, false, -1, 0, 0},
		{0x2003093c, 4, false, -1, 0, 0},
		{0x20030940, 4, false, -1, 0, 0},
		{0x20030944, 4, false, -1, 0, 0},
		{0x20030948, 4, false, -1, 0, 0},
		{0x2003094c, 4, false, -1, 0, 0},
		{0x20030950, 4, false, -1, 0, 0},
		{0x20030954, 4, false, -1, 0, 0},
		{0x20030958, 4, false, -1, 0, 0},
		{0x2003095c, 4, false, -1, 0, 0},
		{0x20030960, 4, false, -1, 0, 0},
		{0x20030964, 4, false, -1, 0, 0},
		{0x20030968, 4, false, -1, 0, 0},
		{0x2003096c, 4, false, -1, 0, 0},
		{0x20030970, 4, false, -1, 0, 0},
		{0x20030974, 4, false, -1, 0, 0},
		{0x20030978, 4, false, -1, 0, 0},
		{0x2003097c, 4, false, -1, 0, 0},
		{0x20030980, 4, false, -1, 0, 0},
		{0x20030984, 4, false, -1, 0, 0},
		{0x20030988, 4, false, -1, 0, 0},
		{0x2003098c, 4, false, -1, 0, 0},
		{0x20030990, 4, false, -1, 0, 0},
		{0x20030994, 4, false, -1, 0, 0},
		{0x20030998, 4, false, -1, 0, 0},
		{0x2003099c, 4, false, -1, 0, 0},
		{0x200309a0, 4, false, -1, 0, 0},
		{0x200309a4, 4, false, -1, 0, 0},
		{0x200309a8, 4, false, -1, 0, 0},
		{0x200309ac, 4, false, -1, 0, 0},
		{0x200309b0, 4, false, -1, 0, 0},
		{0x200309b4, 4, false, -1, 0, 0},
		{0x200309b8, 4, false, -1, 0, 0},
		{0x200309bc, 4, false, -1, 0, 0},
		{0x200309c0, 4, false, -1, 0, 0},
		{0x200309c4, 4, false, -1, 0, 0},
		{0x200309c8, 4, false, -1, 0, 0},
		{0x200309cc, 4, false, -1, 0, 0},
		{0x200309d0, 4, false, -1, 0, 0},
		{0x200309d4, 4, false, -1, 0, 0},
		{0x200309d8, 4, false, -1, 0, 0},
		{0x200309dc, 4, false, -1, 0, 0},
		{0x200309e0, 4, false, -1, 0, 0},
		{0x200309e4, 4, false, -1, 0, 0},
		{0x200309e8, 4, false, -1, 0, 0},
		{0x200309ec, 4, false, -1, 0, 0},
		{0x200309f0, 4, false, -1, 0, 0},
		{0x200309f4, 4, false, -1, 0, 0},
		{0x200309f8, 4, false, -1, 0, 0},
		{0x200309fc, 4, false, -1, 0, 0},
		{0x20030a00, 4, false, -1, 0, 0},
		{0x20030a04, 4, false, -1, 0, 0},
		{0x20030a08, 4, false, -1, 0, 0},
		{0x20030a0c, 4, false, -1, 0, 0},
		{0x20030a10, 4, false, -1, 0, 0},
		{0x20030a14, 4, false, -1, 0, 0},
		{0x20030a18, 4, false, -1, 0, 0},
		{0x20030a1c, 4, false, -1, 0, 0},
		{0x20030a20, 4, false, -1, 0, 0},
		{0x20030a24, 4, false, -1, 0, 0},
		{0x20030a28, 4, false, -1, 0, 0},
		{0x20030a2c, 4, false, -1, 0, 0},
		{0x20030a30, 4, false, -1, 0, 0},
		{0x20030a34, 4, false, -1, 0, 0},
		{0x20030a38, 4, false, -1, 0, 0},
		{0x20030a3c, 4, false, -1, 0, 0},
		{0x20030a40, 4, false, -1, 0, 0},
		{0x20030a44, 4, false, -1, 0, 0},
		{0x20030a48, 4, false, -1, 0, 0},
		{0x20030a4c, 4, false, -1, 0, 0},
		{0x20030a50, 4, false, -1, 0, 0},
		{0x20030a54, 4, false, -1, 0, 0},
		{0x20030a58, 4, false, -1, 0, 0},
		{0x20030a5c, 4, false, -1, 0, 0},
		{0x20030a60, 4, false, -1, 0, 0},
		{0x20030a64, 4, false, -1, 0, 0},
		{0x20030a68, 4, false, -1, 0, 0},
		{0x20030a6c, 4, false, -1, 0, 0},
		{0x20030a70, 4, false, -1, 0, 0},
		{0x20030a74, 4, false, -1, 0, 0},
		{0x20030a78, 4, false, -1, 0, 0},
		{0x20030a7c, 4, false, -1, 0, 0},
		{0x20030a80, 4, false, -1, 0, 0},
		{0x20030a84, 4, false, -1, 0, 0},
		{0x20030a88, 4, false, -1, 0, 0},
		{0x20030a8c, 4, false, -1, 0, 0},
		{0x20030a90, 4, false, -1, 0, 0},
		{0x20030a94, 4, false, -1, 0, 0},
		{0x20030a98, 4, false, -1, 0, 0},
		{0x20030a9c, 4, false, -1, 0, 0},
		{0x20030aa0, 4, false, -1, 0, 0},
		{0x20030aa4, 4, false, -1, 0, 0},
		{0x20030aa8, 4, false, -1, 0, 0},
		{0x20030aac, 4, false, -1, 0, 0},
		{0x20030ab0, 4, false, -1, 0, 0},
		{0x20030ab4, 4, false, -1, 0, 0},
		{0x20030ab8, 4, false, -1, 0, 0},
		{0x20030abc, 4, false, -1, 0, 0},
		{0x20030ac0, 4, false, -1, 0, 0},
		{0x20030ac4, 4, false, -1, 0, 0},
		{0x20030ac8, 4, false, -1, 0, 0},
		{0x20030acc, 4, false, -1, 0, 0},
		{0x20030ad0, 4, false, -1, 0, 0},
		{0x20030ad4, 4, false, -1, 0, 0},
		{0x20030ad8, 4, false, -1, 0, 0},
		{0x20030adc, 4, false, -1, 0, 0},
		{0x20030ae0, 4, false, -1, 0, 0},
		{0x20030ae4, 4, false, -1, 0, 0},
		{0x20030ae8, 4, false, -1, 0, 0},
		{0x20030aec, 4, false, -1, 0, 0},
		{0x20030af0, 4, false, -1, 0, 0},
		{0x20030af4, 4, false, -1, 0, 0},
		{0x20030af8, 4, false, -1, 0, 0},
		{0x20030afc, 4, false, -1, 0, 0},
		{0x20030b00, 4, false, -1, 0, 0},
		{0x20030b04, 4, false, -1, 0, 0},
		{0x20030b08, 4, false, -1, 0, 0},
		{0x20030b0c, 4, false, -1, 0, 0},
		{0x20030b10, 4, false, -1, 0, 0},
		{0x20030b14, 4, false, -1, 0, 0},
		{0x20030b18, 4, false, -1, 0, 0},
		{0x20030b1c, 4, false, -1, 0, 0},
		{0x20030b20, 4, false, -1, 0, 0},
		{0x20030b24, 4, false, -1, 0, 0},
		{0x20030b28, 4, false, -1, 0, 0},
		{0x20030b2c, 4, false, -1, 0, 0},
		{0x20030b30, 4, false, -1, 0, 0},
		{0x20030b34, 4, false, -1, 0, 0},
		{0x20030b38, 4, false, -1, 0, 0},
		{0x20030b3c, 4, false, -1, 0, 0},
		{0x20030b40, 4, false, -1, 0, 0},
		{0x20030b44, 4, false, -1, 0, 0},
		{0x20030b48, 4, false, -1, 0, 0},
		{0x20030b4c, 4, false, -1, 0, 0},
		{0x20030b50, 4, false, -1, 0, 0},
		{0x20030b54, 4, false, -1, 0, 0},
		{0x20030b58, 4, false, -1, 0, 0},
		{0x20030b5c, 4, false, -1, 0, 0},
		{0x20030b60, 4, false, -1, 0, 0},
		{0x20030b64, 4, false, -1, 0, 0},
		{0x20030b68, 4, false, -1, 0, 0},
		{0x20030b6c, 4, false, -1, 0, 0},
		{0x20030b70, 4, false, -1, 0, 0},
		{0x20030b74, 4, false, -1, 0, 0},
		{0x20030b78, 4, false, -1, 0, 0},
		{0x20030b7c, 4, false, -1, 0, 0},
		{0x20030b80, 4, false, -1, 0, 0},
		{0x20030b84, 4, false, -1, 0, 0},
		{0x20030b88, 4, false, -1, 0, 0},
		{0x20030b8c, 4, false, -1, 0, 0},
		{0x20030b90, 4, false, -1, 0, 0},
		{0x20030b94, 4, false, -1, 0, 0},
		{0x20030b98, 4, false, -1, 0, 0},
		{0x20030b9c, 4, false, -1, 0, 0},
		{0x20030ba0, 4, false, -1, 0, 0},
		{0x20030ba4, 4, false, -1, 0, 0},
		{0x20030ba8, 4, false, -1, 0, 0},
		{0x20030bac, 4, false, -1, 0, 0},
		{0x20030bb0, 4, false, -1, 0, 0},
		{0x20030bb4, 4, false, -1, 0, 0},
		{0x20030bb8, 4, false, -1, 0, 0},
		{0x20030bbc, 4, false, -1, 0, 0},
		{0x20030bc0, 4, false, -1, 0, 0},
		{0x20030bc4, 4, false, -1, 0, 0},
		{0x20030bc8, 4, false, -1, 0, 0},
		{0x20030bcc, 4, false, -1, 0, 0},
		{0x20030bd0, 4, false, -1, 0, 0},
		{0x20030bd4, 4, false, -1, 0, 0},
		{0x20030bd8, 4, false, -1, 0, 0},
		{0x20030bdc, 4, false, -1, 0, 0},
		{0x20030be0, 4, false, -1, 0, 0},
		{0x20030be4, 4, false, -1, 0, 0},
		{0x20030be8, 4, false, -1, 0, 0},
		{0x20030bec, 4, false, -1, 0, 0},
		{0x20030bf0, 4, false, -1, 0, 0},
		{0x20030bf4, 4, false, -1, 0, 0},
		{0x20030bf8, 4, false, -1, 0, 0},
		{0x20030bfc, 4, false, -1, 0, 0},
		{0x20030c00, 4, false, -1, 0, 0},
		{0x20030c04, 4, false, -1, 0, 0},
		{0x20030c08, 4, false, -1, 0, 0},
		{0x20030c0c, 4, false, -1, 0, 0},
		{0x20030c10, 4, false, -1, 0, 0},
		{0x20030c14, 4, false, -1, 0, 0},
		{0x20030c18, 4, false, -1, 0, 0},
		{0x20030c1c, 4, false, -1, 0, 0},
		{0x20030c20, 4, false, -1, 0, 0},
		{0x20030c24, 4, false, -1, 0, 0},
		{0x20030c28, 4, false, -1, 0, 0},
		{0x20030c2c, 4, false, -1, 0, 0},
		{0x20030c30, 4, false, -1, 0, 0},
		{0x20030c34, 4, false, -1, 0, 0},
		{0x20030c38, 4, false, -1, 0, 0},
		{0x20030c3c, 4, false, -1, 0, 0},
		{0x20030c40, 4, false, -1, 0, 0},
		{0x20030c44, 4, false, -1, 0, 0},
		{0x20030c48, 4, false, -1, 0, 0},
		{0x20030c4c, 4, false, -1, 0, 0},
		{0x20030c50, 4, false, -1, 0, 0},
		{0x20030c54, 4, false, -1, 0, 0},
		{0x20030c58, 4, false, -1, 0, 0},
		{0x20030c5c, 4, false, -1, 0, 0},
		{0x20030c60, 4, false, -1, 0, 0},
		{0x20030c64, 4, false, -1, 0, 0},
		{0x20030c68, 4, false, -1, 0, 0},
		{0x20030c6c, 4, false, -1, 0, 0},
		{0x20030c70, 4, false, -1, 0, 0},
		{0x20030c74, 4, false, -1, 0, 0},
		{0x20030c78, 4, false, -1, 0, 0},
		{0x20030c7c, 4, false, -1, 0, 0},
		{0x20030c80, 4, false, -1, 0, 0},
		{0x20030c84, 4, false, -1, 0, 0},
		{0x20030c88, 4, false, -1, 0, 0},
		{0x20030c8c, 4, false, -1, 0, 0},
		{0x20030c90, 4, false, -1, 0, 0},
		{0x20030c94, 4, false, -1, 0, 0},
		{0x20030c98, 4, false, -1, 0, 0},
		{0x20030c9c, 4, false, -1, 0, 0},
		{0x20030ca0, 4, false, -1, 0, 0},
		{0x20030ca4, 4, false, -1, 0, 0},
		{0x20030ca8, 4, false, -1, 0, 0},
		{0x20030cac, 4, false, -1, 0, 0},
		{0x20030cb0, 4, false, -1, 0, 0},
		{0x20030cb4, 4, false, -1, 0, 0},
		{0x20030cb8, 4, false, -1, 0, 0},
		{0x20030cbc, 4, false, -1, 0, 0},
		{0x20030cc0, 4, false, -1, 0, 0},
		{0x20030cc4, 4, false, -1, 0, 0},
		{0x20030cc8, 4, false, -1, 0, 0},
		{0x20030ccc, 4, false, -1, 0, 0},
		{0x20030cd0, 4, false, -1, 0, 0},
		{0x20030cd4, 4, false, -1, 0, 0},
		{0x20030cd8, 4, false, -1, 0, 0},
		{0x20030cdc, 4, false, -1, 0, 0},
		{0x20030ce0, 4, false, -1, 0, 0},
		{0x20030ce4, 4, false, -1, 0, 0},
		{0x20030ce8, 4, false, -1, 0, 0},
		{0x20030cec, 4, false, -1, 0, 0},
		{0x20030cf0, 4, false, -1, 0, 0},
		{0x20030cf4, 4, false, -1, 0, 0},
		{0x20030cf8, 4, false, -1, 0, 0},
		{0x20030cfc, 4, false, -1, 0, 0},
		{0x20030d00, 4, false, -1, 0, 0},
		{0x20030d04, 4, false, -1, 0, 0},
		{0x20030d08, 4, false, -1, 0, 0},
		{0x20030d0c, 4, false, -1, 0, 0},
		{0x20030d10, 4, false, -1, 0, 0},
		{0x20030d14, 4, false, -1, 0, 0},
		{0x20030d18, 4, false, -1, 0, 0},
		{0x20030d1c, 4, false, -1, 0, 0},
		{0x20030d20, 4, false, -1, 0, 0},
		{0x20030d24, 4, false, -1, 0, 0},
		{0x20030d28, 4, false, -1, 0, 0},
		{0x20030d2c, 4, false, -1, 0, 0},
		{0x20030d30, 4, false, -1, 0, 0},
		{0x20030d34, 4, false, -1, 0, 0},
		{0x20030d38, 4, false, -1, 0, 0},
		{0x20030d3c, 4, false, -1, 0, 0},
		{0x20030d40, 4, false, -1, 0, 0},
		{0x20030d44, 4, false, -1, 0, 0},
		{0x20030d48, 4, false, -1, 0, 0},
		{0x20030d4c, 4, false, -1, 0, 0},
		{0x20030d50, 4, false, -1, 0, 0},
		{0x20030d54, 4, false, -1, 0, 0},
		{0x20030d58, 4, false, -1, 0, 0},
		{0x20030d5c, 4, false, -1, 0, 0},
		{0x20030d60, 4, false, -1, 0, 0},
		{0x20030d64, 4, false, -1, 0, 0},
		{0x20030d68, 4, false, -1, 0, 0},
		{0x20030d6c, 4, false, -1, 0, 0},
		{0x20030d70, 4, false, -1, 0, 0},
		{0x20030d74, 4, false, -1, 0, 0},
		{0x20030d78, 4, false, -1, 0, 0},
		{0x20030d7c, 4, false, -1, 0, 0},
		{0x20030d80, 4, false, -1, 0, 0},
		{0x20030d84, 4, false, -1, 0, 0},
		{0x20030d88, 4, false, -1, 0, 0},
		{0x20030d8c, 4, false, -1, 0, 0},
		{0x20030d90, 4, false, -1, 0, 0},
		{0x20030d94, 4, false, -1, 0, 0},
		{0x20030d98, 4, false, -1, 0, 0},
		{0x20030d9c, 4, false, -1, 0, 0},
		{0x20030da0, 4, false, -1, 0, 0},
		{0x20030da4, 4, false, -1, 0, 0},
		{0x20030da8, 4, false, -1, 0, 0},
		{0x20030dac, 4, false, -1, 0, 0},
		{0x20030db0, 4, false, -1, 0, 0},
		{0x20030db4, 4, false, -1, 0, 0},
		{0x20030db8, 4, false, -1, 0, 0},
		{0x20030dbc, 4, false, -1, 0, 0},
		{0x20030dc0, 4, false, -1, 0, 0},
		{0x20030dc4, 4, false, -1, 0, 0},
		{0x20030dc8, 4, false, -1, 0, 0},
		{0x20030dcc, 4, false, -1, 0, 0},
		{0x20030dd0, 4, false, -1, 0, 0},
		{0x20030dd4, 4, false, -1, 0, 0},
		{0x20030dd8, 4, false, -1, 0, 0},
		{0x20030ddc, 4, false, -1, 0, 0},
		{0x20030de0, 4, false, -1, 0, 0},
		{0x20030de4, 4, false, -1, 0, 0},
		{0x20030de8, 4, false, -1, 0, 0},
		{0x20030dec, 4, false, -1, 0, 0},
		{0x20030df0, 4, false, -1, 0, 0},
		{0x20030df4, 4, false, -1, 0, 0},
		{0x20030df8, 4, false, -1, 0, 0},
		{0x20030dfc, 4, false, -1, 0, 0},
		{0x20030e00, 4, false, -1, 0, 0},
		{0x20030e04, 4, false, -1, 0, 0},
		{0x20030e08, 4, false, -1, 0, 0},
		{0x20030e0c, 4, false, -1, 0, 0},
		{0x20030e10, 4, false, -1, 0, 0},
		{0x20030e14, 4, false, -1, 0, 0},
		{0x20030e18, 4, false, -1, 0, 0},
		{0x20030e1c, 4, false, -1, 0, 0},
		{0x20030e20, 4, false, -1, 0, 0},
		{0x20030e24, 4, false, -1, 0, 0},
		{0x20030e28, 4, false, -1, 0, 0},
		{0x20030e2c, 4, false, -1, 0, 0},
		{0x20030e30, 4, false, -1, 0, 0},
		{0x20030e34, 4, false, -1, 0, 0},
		{0x20030e38, 4, false, -1, 0, 0},
		{0x20030e3c, 4, false, -1, 0, 0},
		{0x20030e40, 4, false, -1, 0, 0},
		{0x20030e44, 4, false, -1, 0, 0},
		{0x20030e48, 4, false, -1, 0, 0},
		{0x20030e4c, 4, false, -1, 0, 0},
		{0x20030e50, 4, false, -1, 0, 0},
		{0x20030e54, 4, false, -1, 0, 0},
		{0x20030e58, 4, false, -1, 0, 0},
		{0x20030e5c, 4, false, -1, 0, 0},
		{0x20030e60, 4, false, -1, 0, 0},
		{0x20030e64, 4, false, -1, 0, 0},
		{0x20030e68, 4, false, -1, 0, 0},
		{0x20030e6c, 4, false, -1, 0, 0},
		{0x20030e70, 4, false, -1, 0, 0},
		{0x20030e74, 4, false, -1, 0, 0},
		{0x20030e78, 4, false, -1, 0, 0},
		{0x20030e7c, 4, false, -1, 0, 0},
		{0x20030e80, 4, false, -1, 0, 0},
		{0x20030e84, 4, false, -1, 0, 0},
		{0x20030e88, 4, false, -1, 0, 0},
		{0x20030e8c, 4, false, -1, 0, 0},
		{0x20030e90, 4, false, -1, 0, 0},
		{0x20030e94, 4, false, -1, 0, 0},
		{0x20030e98, 4, false, -1, 0, 0},
		{0x20030e9c, 4, false, -1, 0, 0},
		{0x20030ea0, 4, false, -1, 0, 0},
		{0x20030ea4, 4, false, -1, 0, 0},
		{0x20030ea8, 4, false, -1, 0, 0},
		{0x20030eac, 4, false, -1, 0, 0},
		{0x20030eb0, 4, false, -1, 0, 0},
		{0x20030eb4, 4, false, -1, 0, 0},
		{0x20030eb8, 4, false, -1, 0, 0},
		{0x20030ebc, 4, false, -1, 0, 0},
		{0x20030ec0, 4, false, -1, 0, 0},
		{0x20030ec4, 4, false, -1, 0, 0},
		{0x20030ec8, 4, false, -1, 0, 0},
		{0x20030ecc, 4, false, -1, 0, 0},
		{0x20030ed0, 4, false, -1, 0, 0},
		{0x20030ed4, 4, false, -1, 0, 0},
		{0x20030ed8, 4, false, -1, 0, 0},
		{0x20030edc, 4, false, -1, 0, 0},
		{0x20030ee0, 4, false, -1, 0, 0},
		{0x20030ee4, 4, false, -1, 0, 0},
		{0x20030ee8, 4, false, -1, 0, 0},
		{0x20030eec, 4, false, -1, 0, 0},
		{0x20030ef0, 4, false, -1, 0, 0},
		{0x20030ef4, 4, false, -1, 0, 0},
		{0x20030ef8, 4, false, -1, 0, 0},
		{0x20030efc, 4, false, -1, 0, 0},
		{0x20030f00, 4, false, -1, 0, 0},
		{0x20030f04, 4, false, -1, 0, 0},
		{0x20030f08, 4, false, -1, 0, 0},
		{0x20030f0c, 4, false, -1, 0, 0},
		{0x20030f10, 4, false, -1, 0, 0},
		{0x20030f14, 4, false, -1, 0, 0},
		{0x20030f18, 4, false, -1, 0, 0},
		{0x20030f1c, 4, false, -1, 0, 0},
		{0x20030f20, 4, false, -1, 0, 0},
		{0x20030f24, 4, false, -1, 0, 0},
		{0x20030f28, 4, false, -1, 0, 0},
		{0x20030f2c, 4, false, -1, 0, 0},
		{0x20030f30, 4, false, -1, 0, 0},
		{0x20030f34, 4, false, -1, 0, 0},
		{0x20030f38, 4, false, -1, 0, 0},
		{0x20030f3c, 4, false, -1, 0, 0},
		{0x20030f40, 4, false, -1, 0, 0},
		{0x20030f44, 4, false, -1, 0, 0},
		{0x20030f48, 4, false, -1, 0, 0},
		{0x20030f4c, 4, false, -1, 0, 0},
		{0x20030f50, 4, false, -1, 0, 0},
		{0x20030f54, 4, false, -1, 0, 0},
		{0x20030f58, 4, false, -1, 0, 0},
		{0x20030f5c, 4, false, -1, 0, 0},
		{0x20030f60, 4, false, -1, 0, 0},
		{0x20030f64, 4, false, -1, 0, 0},
		{0x20030f68, 4, false, -1, 0, 0},
		{0x20030f6c, 4, false, -1, 0, 0},
		{0x20030f70, 4, false, -1, 0, 0},
		{0x20030f74, 4, false, -1, 0, 0},
		{0x20030f78, 4, false, -1, 0, 0},
		{0x20030f7c, 4, false, -1, 0, 0},
		{0x20030f80, 4, false, -1, 0, 0},
		{0x20030f84, 4, false, -1, 0, 0},
		{0x20030f88, 4, false, -1, 0, 0},
		{0x20030f8c, 4, false, -1, 0, 0},
		{0x20030f90, 4, false, -1, 0, 0},
		{0x20030f94, 4, false, -1, 0, 0},
		{0x20030f98, 4, false, -1, 0, 0},
		{0x20030f9c, 4, false, -1, 0, 0},
		{0x20030fa0, 4, false, -1, 0, 0},
		{0x20030fa4, 4, false, -1, 0, 0},
		{0x20030fa8, 4, false, -1, 0, 0},
		{0x20030fac, 4, false, -1, 0, 0},
		{0x20030fb0, 4, false, -1, 0, 0},
		{0x20030fb4, 4, false, -1, 0, 0},
		{0x20030fb8, 4, false, -1, 0, 0},
		{0x20030fbc, 4, false, -1, 0, 0},
		{0x20030fc0, 4, false, -1, 0, 0},
		{0x20030fc4, 4, false, -1, 0, 0},
		{0x20030fc8, 4, false, -1, 0, 0},
		{0x20030fcc, 4, false, -1, 0, 0},
		{0x20030fd0, 4, false, -1, 0, 0},
		{0x20030fd4, 4, false, -1, 0, 0},
		{0x20030fd8, 4, false, -1, 0, 0},
		{0x20030fdc, 4, false, -1, 0, 0},
		{0x20030fe0, 4, false, -1, 0, 0},
		{0x20030fe4, 4, false, -1, 0, 0},
		{0x20030fe8, 4, false, -1, 0, 0},
		{0x20030fec, 4, false, -1, 0, 0},
		{0x20030ff0, 4, false, -1, 0, 0},
		{0x20030ff4, 4, false, -1, 0, 0},
		{0x20030ff8, 4, false, -1, 0, 0},
		{0x20030ffc, 4, false, -1, 0, 0},
		{0x20031000, 4, false, -1, 0, 0},
		{0x20031004, 4, false, -1, 0, 0},
		{0x20031008, 4, false, -1, 0, 0},
		{0x2003100c, 4, false, -1, 0, 0},
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
		{0x200310b8, 4, false, -1, 0, 0},
		{0x200310bc, 4, false, -1, 0, 0},
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
		{0x20031100, 4, false, -1, 0, 0},
		{0x20031104, 4, false, -1, 0, 0},
		{0x20031108, 4, false, -1, 0, 0},
		{0x2003110c, 4, false, -1, 0, 0},
		{0x20031110, 4, false, -1, 0, 0},
		{0x20031114, 4, false, -1, 0, 0},
		{0x20031118, 4, false, -1, 0, 0},
		{0x2003111c, 4, false, -1, 0, 0},
		{0x20031120, 4, false, -1, 0, 0},
		{0x20031124, 4, false, -1, 0, 0},
		{0x20031128, 4, false, -1, 0, 0},
		{0x2003112c, 4, false, -1, 0, 0},
		{0x20031130, 4, false, -1, 0, 0},
		{0x20031134, 4, false, -1, 0, 0},
		{0x20031138, 4, false, -1, 0, 0},
		{0x2003113c, 4, false, -1, 0, 0},
		{0x20031140, 4, false, -1, 0, 0},
		{0x20031144, 4, false, -1, 0, 0},
		{0x20031148, 4, false, -1, 0, 0},
		{0x2003114c, 4, false, -1, 0, 0},
		{0x20031150, 4, false, -1, 0, 0},
		{0x20031154, 4, false, -1, 0, 0},
		{0x20031158, 4, false, -1, 0, 0},
		{0x2003115c, 4, false, -1, 0, 0},
		{0x20031160, 4, false, -1, 0, 0},
		{0x20031164, 4, false, -1, 0, 0},
		{0x20031168, 4, false, -1, 0, 0},
		{0x2003116c, 4, false, -1, 0, 0},
		{0x20031170, 4, false, -1, 0, 0},
		{0x20031174, 4, false, -1, 0, 0},
		{0x20031178, 4, false, -1, 0, 0},
		{0x2003117c, 4, false, -1, 0, 0},
		{0x20031180, 4, false, -1, 0, 0},
		{0x20031184, 4, false, -1, 0, 0},
		{0x20031188, 4, false, -1, 0, 0},
		{0x2003118c, 4, false, -1, 0, 0},
		{0x20031190, 4, false, -1, 0, 0},
		{0x20031194, 4, false, -1, 0, 0},
		{0x20031198, 4, false, -1, 0, 0},
		{0x2003119c, 4, false, -1, 0, 0},
		{0x200311a0, 4, false, -1, 0, 0},
		{0x200311a4, 4, false, -1, 0, 0},
		{0x200311a8, 4, false, -1, 0, 0},
		{0x200311ac, 4, false, -1, 0, 0},
		{0x200311b0, 4, false, -1, 0, 0},
		{0x200311b4, 4, false, -1, 0, 0},
		{0x200311b8, 4, false, -1, 0, 0},
		{0x200311bc, 4, false, -1, 0, 0},
		{0x200311c0, 4, false, -1, 0, 0},
		{0x200311c4, 4, false, -1, 0, 0},
		{0x200311c8, 4, false, -1, 0, 0},
		{0x200311cc, 4, false, -1, 0, 0},
		{0x200311d0, 4, false, -1, 0, 0},
		{0x200311d4, 4, false, -1, 0, 0},
		{0x200311d8, 4, false, -1, 0, 0},
		{0x200311dc, 4, false, -1, 0, 0},
		{0x200311e0, 4, false, -1, 0, 0},
		{0x200311e4, 4, false, -1, 0, 0},
		{0x200311e8, 4, false, -1, 0, 0},
		{0x200311ec, 4, false, -1, 0, 0},
		{0x200311f0, 4, false, -1, 0, 0},
		{0x200311f4, 4, false, -1, 0, 0},
		{0x200311f8, 4, false, -1, 0, 0},
		{0x200311fc, 4, false, -1, 0, 0},
		{0x20031200, 4, false, -1, 0, 0},
		{0x20031204, 4, false, -1, 0, 0},
		{0x20031208, 4, false, -1, 0, 0},
		{0x2003120c, 4, false, -1, 0, 0},
		{0x20031210, 4, false, -1, 0, 0},
		{0x20031214, 4, false, -1, 0, 0},
		{0x20031218, 4, false, -1, 0, 0},
		{0x2003121c, 1, false, -1, 0, 0},
		{0x2003121d, 1, false, -1, 0, 0},
		{0x2003121e, 1, false, -1, 0, 0},
		{0x2003121f, 1, false, -1, 0, 0},
		{0x20031220, 1, false, -1, 0, 0},
		{0x20031221, 1, false, -1, 0, 0},
		{0x20031222, 1, false, -1, 0, 0},
		{0x20031223, 1, false, -1, 0, 0},
		{0x20031224, 1, false, -1, 0, 0},
		{0x20031225, 1, false, -1, 0, 0},
		{0x20031226, 1, false, -1, 0, 0},
		{0x20031227, 1, false, -1, 0, 0},
		{0x20031228, 1, false, -1, 0, 0},
		{0x20031229, 1, false, -1, 0, 0},
		{0x2003122a, 1, false, -1, 0, 0},
		{0x2003122b, 1, false, -1, 0, 0},
		{0x2003122c, 1, false, -1, 0, 0},
		{0x2003122d, 1, false, -1, 0, 0},
		{0x2003122e, 1, false, -1, 0, 0},
		{0x2003122f, 1, false, -1, 0, 0},
		{0x20031230, 1, false, -1, 0, 0},
		{0x20031231, 1, false, -1, 0, 0},
		{0x20031232, 1, false, -1, 0, 0},
		{0x20031233, 1, false, -1, 0, 0},
		{0x20031234, 1, false, -1, 0, 0},
		{0x20031235, 1, false, -1, 0, 0},
		{0x20031236, 1, false, -1, 0, 0},
		{0x20031237, 1, false, -1, 0, 0},
		{0x20031238, 1, false, -1, 0, 0},
		{0x20031239, 1, false, -1, 0, 0},
		{0x2003123a, 1, false, -1, 0, 0},
		{0x2003123b, 1, false, -1, 0, 0},
		{0x2003123c, 1, false, -1, 0, 0},
		{0x2003123d, 1, false, -1, 0, 0},
		{0x2003123e, 1, false, -1, 0, 0},
		{0x2003123f, 1, false, -1, 0, 0},
		{0x20031240, 1, false, -1, 0, 0},
		{0x20031241, 1, false, -1, 0, 0},
		{0x20031242, 1, false, -1, 0, 0},
		{0x20031243, 1, false, -1, 0, 0},
		{0x20031244, 4, false, -1, 0, 0},
		{0x20031248, 4, false, -1, 0, 0},
		{0x2003124c, 4, false, -1, 0, 0},
		{0x20031250, 4, false, -1, 0, 0},
		{0x20031254, 4, false, -1, 0, 0},
		{0x20031258, 4, false, -1, 0, 0},
		{0x2003125c, 4, false, -1, 0, 0},
		{0x20031260, 4, false, -1, 0, 0},
		{0x20031264, 4, false, -1, 0, 0},
		{0x20031268, 4, false, -1, 0, 0},
		{0x2003126c, 4, false, -1, 0, 0},
		{0x20031270, 4, false, -1, 0, 0},
		{0x20031274, 4, false, -1, 0, 0},
		{0x20031278, 4, false, -1, 0, 0},
		{0x2003127c, 4, false, -1, 0, 0},
		{0x20031280, 4, false, -1, 0, 0},
		{0x20031284, 4, false, -1, 0, 0},
		{0x20031288, 4, false, -1, 0, 0},
		{0x2003128c, 4, false, -1, 0, 0},
		{0x20031290, 4, false, -1, 0, 0},
		{0x20031294, 4, false, -1, 0, 0},
		{0x20031298, 4, false, -1, 0, 0},
		{0x2003129c, 4, false, -1, 0, 0},
		{0x200312a0, 4, false, -1, 0, 0},
		{0x200312a4, 4, false, -1, 0, 0},
		{0x200312a8, 1, false, -1, 0, 0},
		{0x200312a9, 1, false, -1, 0, 0},
		{0x200312aa, 4, false, -1, 0, 0},
		{0x200312ae, 4, false, -1, 0, 0},
		{0x200312b2, 4, false, -1, 0, 0},
		{0x200312b6, 4, false, -1, 0, 0},
		{0x200312ba, 4, false, -1, 0, 0},
		{0x200312be, 4, false, -1, 0, 0},
		{0x200312c2, 4, false, -1, 0, 0},
		{0x200312c6, 4, false, -1, 0, 0},
		{0x200312ca, 4, false, -1, 0, 0},
		{0x200312ce, 4, false, -1, 0, 0},
		{0x200312d2, 4, false, -1, 0, 0},
		{0x200312d6, 4, false, -1, 0, 0},
		{0x200312da, 4, false, -1, 0, 0},
		{0x200312de, 4, false, -1, 0, 0},
		{0x200312e2, 4, false, -1, 0, 0},
		{0x200312e6, 4, false, -1, 0, 0},
		{0x200312ea, 4, false, -1, 0, 0},
		{0x200312ee, 4, false, -1, 0, 0},
		{0x200312f2, 4, false, -1, 0, 0},
		{0x200312f6, 4, false, -1, 0, 0},
		{0x200312fa, 4, false, -1, 0, 0},
		{0x200312fe, 4, false, -1, 0, 0},
		{0x20031302, 4, false, -1, 0, 0},
		{0x20031306, 4, false, -1, 0, 0},
		{0x2003130a, 4, false, -1, 0, 0},
		{0x2003130e, 4, false, -1, 0, 0},
		{0x20031312, 4, false, -1, 0, 0},
		{0x20031316, 4, false, -1, 0, 0},
		{0x2003131a, 4, false, -1, 0, 0},
		{0x2003131e, 4, false, -1, 0, 0},
		{0x20031322, 4, false, -1, 0, 0},
		{0x20031326, 4, false, -1, 0, 0},
		{0x2003132a, 4, false, -1, 0, 0},
		{0x2003132e, 4, false, -1, 0, 0},
		{0x20031332, 4, false, -1, 0, 0},
		{0x20031336, 4, false, -1, 0, 0},
		{0x2003133a, 4, false, -1, 0, 0},
		{0x2003133e, 4, false, -1, 0, 0},
		{0x20031342, 4, false, -1, 0, 0},
		{0x20031346, 4, false, -1, 0, 0},
		{0x2003134a, 4, false, -1, 0, 0},
		{0x2003134e, 4, false, -1, 0, 0},
		{0x20031352, 4, false, -1, 0, 0},
		{0x20031356, 4, false, -1, 0, 0},
		{0x2003135a, 4, false, -1, 0, 0},
		{0x2003135e, 4, false, -1, 0, 0},
		{0x20031362, 4, false, -1, 0, 0},
		{0x20031366, 4, false, -1, 0, 0},
		{0x2003136a, 4, false, -1, 0, 0},
		{0x2003136e, 4, false, -1, 0, 0},
		{0x20031372, 4, false, -1, 0, 0},
		{0x20031376, 4, false, -1, 0, 0},
		{0x2003137a, 4, false, -1, 0, 0},
		{0x2003137e, 4, false, -1, 0, 0},
		{0x20031382, 4, false, -1, 0, 0},
		{0x20031386, 4, false, -1, 0, 0},
		{0x2003138a, 4, false, -1, 0, 0},
		{0x2003138e, 4, false, -1, 0, 0},
		{0x20031392, 4, false, -1, 0, 0},
		{0x20031396, 4, false, -1, 0, 0},
		{0x2003139a, 4, false, -1, 0, 0},
		{0x2003139e, 4, false, -1, 0, 0},
		{0x200313a2, 4, false, -1, 0, 0},
		{0x200313a6, 4, false, -1, 0, 0},
		{0x200313aa, 4, false, -1, 0, 0},
		{0x200313ae, 4, false, -1, 0, 0},
		{0x200313b2, 4, false, -1, 0, 0},
		{0x200313b6, 4, false, -1, 0, 0},
		{0x200313ba, 4, false, -1, 0, 0},
		{0x200313be, 4, false, -1, 0, 0},
		{0x200313c2, 4, false, -1, 0, 0},
		{0x200313c6, 4, false, -1, 0, 0},
		{0x200313ca, 4, false, -1, 0, 0},
		{0x200313ce, 4, false, -1, 0, 0},
		{0x200313d2, 4, false, -1, 0, 0},
		{0x200313d6, 4, false, -1, 0, 0},
		{0x200313da, 4, false, -1, 0, 0},
		{0x200313de, 4, false, -1, 0, 0},
		{0x200313e2, 4, false, -1, 0, 0},
		{0x200313e6, 4, false, -1, 0, 0},
		{0x200313ea, 4, false, -1, 0, 0},
		{0x200313ee, 4, false, -1, 0, 0},
		{0x200313f2, 4, false, -1, 0, 0},
		{0x200313f6, 4, false, -1, 0, 0},
		{0x200313fa, 4, false, -1, 0, 0},
		{0x200313fe, 4, false, -1, 0, 0},
		{0x20031402, 4, false, -1, 0, 0},
		{0x20031406, 4, false, -1, 0, 0},
		{0x2003140a, 4, false, -1, 0, 0},
		{0x2003140e, 4, false, -1, 0, 0},
		{0x20031412, 4, false, -1, 0, 0},
		{0x20031416, 4, false, -1, 0, 0},
		{0x2003141a, 4, false, -1, 0, 0},
		{0x2003141e, 4, false, -1, 0, 0},
		{0x20031422, 4, false, -1, 0, 0},
		{0x20031426, 4, false, -1, 0, 0},
		{0x2003142a, 4, false, -1, 0, 0},
		{0x2003142e, 4, false, -1, 0, 0},
		{0x20031432, 4, false, -1, 0, 0},
		{0x20031436, 4, false, -1, 0, 0},
		{0x2003143a, 4, false, -1, 0, 0},
		{0x2003143e, 4, false, -1, 0, 0},
		{0x20031442, 4, false, -1, 0, 0},
		{0x20031446, 4, false, -1, 0, 0},
		{0x2003144a, 4, false, -1, 0, 0},
		{0x2003144e, 4, false, -1, 0, 0},
		{0x20031452, 4, false, -1, 0, 0},
		{0x20031456, 4, false, -1, 0, 0},
		{0x2003145a, 4, false, -1, 0, 0},
		{0x2003145e, 4, false, -1, 0, 0},
		{0x20031462, 4, false, -1, 0, 0},
		{0x20031466, 4, false, -1, 0, 0},
		{0x2003146a, 4, false, -1, 0, 0},
		{0x2003146e, 4, false, -1, 0, 0},
		{0x20031472, 4, false, -1, 0, 0},
		{0x20031476, 4, false, -1, 0, 0},
		{0x2003147a, 4, false, -1, 0, 0},
		{0x2003147e, 4, false, -1, 0, 0},
		{0x20031482, 4, false, -1, 0, 0},
		{0x20031486, 4, false, -1, 0, 0},
		{0x2003148a, 4, false, -1, 0, 0},
		{0x2003148e, 4, false, -1, 0, 0},
		{0x20031492, 4, false, -1, 0, 0},
		{0x20031496, 4, false, -1, 0, 0},
		{0x2003149a, 4, false, -1, 0, 0},
		{0x2003149e, 4, false, -1, 0, 0},
		{0x200314a2, 4, false, -1, 0, 0},
		{0x200314a6, 4, false, -1, 0, 0},
		{0x200314aa, 4, false, -1, 0, 0},
		{0x200314ae, 4, false, -1, 0, 0},
		{0x200314b2, 4, false, -1, 0, 0},
		{0x200314b6, 4, false, -1, 0, 0},
		{0x200314ba, 4, false, -1, 0, 0},
		{0x200314be, 4, false, -1, 0, 0},
		{0x200314c2, 4, false, -1, 0, 0},
		{0x200314c6, 4, false, -1, 0, 0},
		{0x200314ca, 4, false, -1, 0, 0},
		{0x200314ce, 4, false, -1, 0, 0},
		{0x200314d2, 4, false, -1, 0, 0},
		{0x200314d6, 4, false, -1, 0, 0},
		{0x200314da, 4, false, -1, 0, 0},
		{0x200314de, 4, false, -1, 0, 0},
		{0x200314e2, 4, false, -1, 0, 0},
		{0x200314e6, 4, false, -1, 0, 0},
		{0x200314ea, 4, false, -1, 0, 0},
		{0x200314ee, 4, false, -1, 0, 0},
		{0x200314f2, 4, false, -1, 0, 0},
		{0x200314f6, 4, false, -1, 0, 0},
		{0x200314fa, 4, false, -1, 0, 0},
		{0x200314fe, 4, false, -1, 0, 0},
		{0x20031502, 4, false, -1, 0, 0},
		{0x20031506, 4, false, -1, 0, 0},
		{0x2003150a, 4, false, -1, 0, 0},
		{0x2003150e, 4, false, -1, 0, 0},
		{0x20031512, 4, false, -1, 0, 0},
		{0x20031516, 4, false, -1, 0, 0},
		{0x2003151a, 4, false, -1, 0, 0},
		{0x2003151e, 4, false, -1, 0, 0},
		{0x20031522, 4, false, -1, 0, 0},
		{0x20031526, 4, false, -1, 0, 0},
		{0x2003152a, 4, false, -1, 0, 0},
		{0x2003152e, 4, false, -1, 0, 0},
		{0x20031532, 4, false, -1, 0, 0},
		{0x20031536, 4, false, -1, 0, 0},
		{0x2003153a, 4, false, -1, 0, 0},
		{0x2003153e, 4, false, -1, 0, 0},
		{0x20031542, 4, false, -1, 0, 0},
		{0x20031546, 4, false, -1, 0, 0},
		{0x2003154a, 4, false, -1, 0, 0},
		{0x2003154e, 4, false, -1, 0, 0},
		{0x20031552, 4, false, -1, 0, 0},
		{0x20031556, 4, false, -1, 0, 0},
		{0x2003155a, 4, false, -1, 0, 0},
		{0x2003155e, 4, false, -1, 0, 0},
		{0x20031562, 4, false, -1, 0, 0},
		{0x20031566, 4, false, -1, 0, 0},
		{0x2003156a, 4, false, -1, 0, 0},
		{0x2003156e, 4, false, -1, 0, 0},
		{0x20031572, 4, false, -1, 0, 0},
		{0x20031576, 4, false, -1, 0, 0},
		{0x2003157a, 4, false, -1, 0, 0},
		{0x2003157e, 4, false, -1, 0, 0},
		{0x20031582, 4, false, -1, 0, 0},
		{0x20031586, 4, false, -1, 0, 0},
		{0x2003158a, 4, false, -1, 0, 0},
		{0x2003158e, 4, false, -1, 0, 0},
		{0x20031592, 4, false, -1, 0, 0},
		{0x20031596, 4, false, -1, 0, 0},
		{0x2003159a, 4, false, -1, 0, 0},
		{0x2003159e, 4, false, -1, 0, 0},
		{0x200315a2, 4, false, -1, 0, 0},
		{0x200315a6, 4, false, -1, 0, 0},
		{0x200315aa, 4, false, -1, 0, 0},
		{0x200315ae, 4, false, -1, 0, 0},
		{0x200315b2, 4, false, -1, 0, 0},
		{0x200315b6, 4, false, -1, 0, 0},
		{0x200315ba, 4, false, -1, 0, 0},
		{0x200315be, 4, false, -1, 0, 0},
		{0x200315c2, 4, false, -1, 0, 0},
		{0x200315c6, 4, false, -1, 0, 0},
		{0x200315ca, 4, false, -1, 0, 0},
		{0x200315ce, 4, false, -1, 0, 0},
		{0x200315d2, 4, false, -1, 0, 0},
		{0x200315d6, 4, false, -1, 0, 0},
		{0x200315da, 4, false, -1, 0, 0},
		{0x200315de, 4, false, -1, 0, 0},
		{0x200315e2, 4, false, -1, 0, 0},
		{0x200315e6, 4, false, -1, 0, 0},
		{0x200315ea, 4, false, -1, 0, 0},
		{0x200315ee, 4, false, -1, 0, 0},
		{0x200315f2, 4, false, -1, 0, 0},
		{0x200315f6, 4, false, -1, 0, 0},
		{0x200315fa, 4, false, -1, 0, 0},
		{0x200315fe, 4, false, -1, 0, 0},
		{0x20031602, 4, false, -1, 0, 0},
		{0x20031606, 4, false, -1, 0, 0},
		{0x2003160a, 4, false, -1, 0, 0},
		{0x2003160e, 4, false, -1, 0, 0},
		{0x20031612, 4, false, -1, 0, 0},
		{0x20031616, 4, false, -1, 0, 0},
		{0x2003161a, 4, false, -1, 0, 0},
		{0x2003161e, 4, false, -1, 0, 0},
		{0x20031622, 4, false, -1, 0, 0},
		{0x20031626, 4, false, -1, 0, 0},
		{0x2003162a, 4, false, -1, 0, 0},
		{0x2003162e, 4, false, -1, 0, 0},
		{0x20031632, 4, false, -1, 0, 0},
		{0x20031636, 4, false, -1, 0, 0},
		{0x2003163a, 4, false, -1, 0, 0},
		{0x2003163e, 4, false, -1, 0, 0},
		{0x20031642, 4, false, -1, 0, 0},
		{0x20031646, 4, false, -1, 0, 0},
		{0x2003164a, 4, false, -1, 0, 0},
		{0x2003164e, 4, false, -1, 0, 0},
		{0x20031652, 4, false, -1, 0, 0},
		{0x20031656, 4, false, -1, 0, 0},
		{0x2003165a, 4, false, -1, 0, 0},
		{0x2003165e, 4, false, -1, 0, 0},
		{0x20031662, 4, false, -1, 0, 0},
		{0x20031666, 4, false, -1, 0, 0},
		{0x2003166a, 4, false, -1, 0, 0},
		{0x2003166e, 4, false, -1, 0, 0},
		{0x20031672, 4, false, -1, 0, 0},
		{0x20031676, 4, false, -1, 0, 0},
		{0x2003167a, 4, false, -1, 0, 0},
		{0x2003167e, 4, false, -1, 0, 0},
		{0x20031682, 4, false, -1, 0, 0},
		{0x20031686, 4, false, -1, 0, 0},
		{0x2003168a, 4, false, -1, 0, 0},
		{0x2003168e, 4, false, -1, 0, 0},
		{0x20031692, 4, false, -1, 0, 0},
		{0x20031696, 4, false, -1, 0, 0},
		{0x2003169a, 4, false, -1, 0, 0},
		{0x2003169e, 4, false, -1, 0, 0},
		{0x200316a2, 4, false, -1, 0, 0},
		{0x200316a6, 4, false, -1, 0, 0},
		{0x200316aa, 4, false, -1, 0, 0},
		{0x200316ae, 4, false, -1, 0, 0},
		{0x200316b2, 4, false, -1, 0, 0},
		{0x200316b6, 4, false, -1, 0, 0},
		{0x200316ba, 4, false, -1, 0, 0},
		{0x200316be, 4, false, -1, 0, 0},
		{0x200316c2, 4, false, -1, 0, 0},
		{0x200316c6, 4, false, -1, 0, 0},
		{0x200316ca, 4, false, -1, 0, 0},
		{0x200316ce, 4, false, -1, 0, 0},
		{0x200316d2, 4, false, -1, 0, 0},
		{0x200316d6, 4, false, -1, 0, 0},
		{0x200316da, 4, false, -1, 0, 0},
		{0x200316de, 4, false, -1, 0, 0},
		{0x200316e2, 4, false, -1, 0, 0},
		{0x200316e6, 4, false, -1, 0, 0},
		{0x200316ea, 4, false, -1, 0, 0},
		{0x200316ee, 4, false, -1, 0, 0},
		{0x200316f2, 4, false, -1, 0, 0},
		{0x200316f6, 4, false, -1, 0, 0},
		{0x200316fa, 4, false, -1, 0, 0},
		{0x200316fe, 4, false, -1, 0, 0},
		{0x20031702, 4, false, -1, 0, 0},
		{0x20031706, 4, false, -1, 0, 0},
		{0x2003170a, 4, false, -1, 0, 0},
		{0x2003170e, 4, false, -1, 0, 0},
		{0x20031712, 4, false, -1, 0, 0},
		{0x20031716, 4, false, -1, 0, 0},
		{0x2003171a, 4, false, -1, 0, 0},
		{0x2003171e, 4, false, -1, 0, 0},
		{0x20031722, 4, false, -1, 0, 0},
		{0x20031726, 4, false, -1, 0, 0},
		{0x2003172a, 4, false, -1, 0, 0},
		{0x2003172e, 4, false, -1, 0, 0},
		{0x20031732, 4, false, -1, 0, 0},
		{0x20031736, 4, false, -1, 0, 0},
		{0x2003173a, 4, false, -1, 0, 0},
		{0x2003173e, 4, false, -1, 0, 0},
		{0x20031742, 4, false, -1, 0, 0},
		{0x20031746, 4, false, -1, 0, 0},
		{0x2003174a, 4, false, -1, 0, 0},
		{0x2003174e, 4, false, -1, 0, 0},
		{0x20031752, 4, false, -1, 0, 0},
		{0x20031756, 4, false, -1, 0, 0},
		{0x2003175a, 4, false, -1, 0, 0},
		{0x2003175e, 4, false, -1, 0, 0},
		{0x20031762, 4, false, -1, 0, 0},
		{0x20031766, 4, false, -1, 0, 0},
		{0x2003176a, 4, false, -1, 0, 0},
		{0x2003176e, 4, false, -1, 0, 0},
		{0x20031772, 4, false, -1, 0, 0},
		{0x20031776, 4, false, -1, 0, 0},
		{0x2003177a, 4, false, -1, 0, 0},
		{0x2003177e, 4, false, -1, 0, 0},
		{0x20031782, 4, false, -1, 0, 0},
		{0x20031786, 4, false, -1, 0, 0},
		{0x2003178a, 4, false, -1, 0, 0},
		{0x2003178e, 4, false, -1, 0, 0},
		{0x20031792, 4, false, -1, 0, 0},
		{0x20031796, 4, false, -1, 0, 0},
		{0x2003179a, 4, false, -1, 0, 0},
		{0x2003179e, 4, false, -1, 0, 0},
		{0x200317a2, 4, false, -1, 0, 0},
		{0x200317a6, 4, false, -1, 0, 0},
		{0x200317aa, 4, false, -1, 0, 0},
		{0x200317ae, 4, false, -1, 0, 0},
		{0x200317b2, 4, false, -1, 0, 0},
		{0x200317b6, 4, false, -1, 0, 0},
		{0x200317ba, 4, false, -1, 0, 0},
		{0x200317be, 4, false, -1, 0, 0},
		{0x200317c2, 4, false, -1, 0, 0},
		{0x200317c6, 4, false, -1, 0, 0},
		{0x200317ca, 4, false, -1, 0, 0},
		{0x200317ce, 4, false, -1, 0, 0},
		{0x200317d2, 4, false, -1, 0, 0},
		{0x200317d6, 4, false, -1, 0, 0},
		{0x200317da, 4, false, -1, 0, 0},
		{0x200317de, 4, false, -1, 0, 0},
		{0x200317e2, 4, false, -1, 0, 0},
		{0x200317e6, 4, false, -1, 0, 0},
		{0x200317ea, 4, false, -1, 0, 0},
		{0x200317ee, 4, false, -1, 0, 0},
		{0x200317f2, 4, false, -1, 0, 0},
		{0x200317f6, 4, false, -1, 0, 0},
		{0x200317fa, 4, false, -1, 0, 0},
		{0x200317fe, 4, false, -1, 0, 0},
		{0x20031802, 4, false, -1, 0, 0},
		{0x20031806, 4, false, -1, 0, 0},
		{0x2003180a, 4, false, -1, 0, 0},
		{0x2003180e, 4, false, -1, 0, 0},
		{0x20031812, 4, false, -1, 0, 0},
		{0x20031816, 4, false, -1, 0, 0},
		{0x2003181a, 4, false, -1, 0, 0},
		{0x2003181e, 4, false, -1, 0, 0},
		{0x20031822, 4, false, -1, 0, 0},
		{0x20031826, 4, false, -1, 0, 0},
		{0x2003182a, 4, false, -1, 0, 0},
		{0x2003182e, 4, false, -1, 0, 0},
		{0x20031832, 4, false, -1, 0, 0},
		{0x20031836, 4, false, -1, 0, 0},
		{0x2003183a, 4, false, -1, 0, 0},
		{0x2003183e, 4, false, -1, 0, 0},
		{0x20031842, 4, false, -1, 0, 0},
		{0x20031846, 4, false, -1, 0, 0},
		{0x2003184a, 4, false, -1, 0, 0},
		{0x2003184e, 4, false, -1, 0, 0},
		{0x20031852, 4, false, -1, 0, 0},
		{0x20031856, 4, false, -1, 0, 0},
		{0x2003185a, 4, false, -1, 0, 0},
		{0x2003185e, 4, false, -1, 0, 0},
		{0x20031862, 4, false, -1, 0, 0},
		{0x20031866, 4, false, -1, 0, 0},
		{0x2003186a, 4, false, -1, 0, 0},
		{0x2003186e, 4, false, -1, 0, 0},
		{0x20031872, 4, false, -1, 0, 0},
		{0x20031876, 4, false, -1, 0, 0},
		{0x2003187a, 4, false, -1, 0, 0},
		{0x2003187e, 4, false, -1, 0, 0},
		{0x20031882, 4, false, -1, 0, 0},
		{0x20031886, 4, false, -1, 0, 0},
		{0x2003188a, 4, false, -1, 0, 0},
		{0x2003188e, 4, false, -1, 0, 0},
		{0x20031892, 4, false, -1, 0, 0},
		{0x20031896, 4, false, -1, 0, 0},
		{0x2003189a, 4, false, -1, 0, 0},
		{0x2003189e, 4, false, -1, 0, 0},
		{0x200318a2, 4, false, -1, 0, 0},
		{0x200318a6, 4, false, -1, 0, 0},
		{0x200318aa, 4, false, -1, 0, 0},
		{0x200318ae, 4, false, -1, 0, 0},
		{0x200318b2, 4, false, -1, 0, 0},
		{0x200318b6, 4, false, -1, 0, 0},
		{0x200318ba, 4, false, -1, 0, 0},
		{0x200318be, 4, false, -1, 0, 0},
		{0x2000602b, 4, false, -1, 0, 0},
		{0x200060c5, 2, false, -1, 0, 0},
		{0x200060c7, 4, false, -1, 0, 0},
		{0x200060bf, 4, false, -1, 0, 0},
		{0x200060c3, 4, false, -1, 0, 0},
		{0x20006033, 4, false, -1, 0, 0},
		{0x20006242, 4, false, -1, 0, 0},
		{0x200062f9, 4, false, -1, 0, 0},
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
		{0x200066f9, 4, false, -1, 0, 0},
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
		{0x6875c3, 4, false, -1, 0, 0},
		{0x6875c7, 4, false, -1, 0, 0},
		{0x6875cb, 4, false, -1, 0, 0},
		{0x6875cf, 4, false, -1, 0, 0},
		{0x6875d3, 4, false, -1, 0, 0},
		{0x6875d7, 4, false, -1, 0, 0},
		{0x6875db, 4, false, -1, 0, 0},
		{0x6875df, 4, false, -1, 0, 0},
	},
})

var ostIn = []string{"notice", "messageAll", "aliasCharacter", "difficulty", "targetType", "targetIndex", "targetMonster", "active0", "spell0", "level0", "bonus0", "baseCount0", "baseAdd0", "powerA0", "powerB0", "spellClass0", "diceZ0", "diceN0", "diceS0", "realm0", "active1", "spell1", "level1", "bonus1", "baseCount1", "baseAdd1", "powerA1", "powerB1", "spellClass1", "diceZ1", "diceN1", "diceS1", "realm1", "active2", "spell2", "level2", "bonus2", "baseCount2", "baseAdd2", "powerA2", "powerB2", "spellClass2", "diceZ2", "diceN2", "diceS2", "realm2", "active3", "spell3", "level3", "bonus3", "baseCount3", "baseAdd3", "powerA3", "powerB3", "spellClass3", "diceZ3", "diceN3", "diceS3", "realm3", "active4", "spell4", "level4", "bonus4", "baseCount4", "baseAdd4", "powerA4", "powerB4", "spellClass4", "diceZ4", "diceN4", "diceS4", "realm4", "active5", "spell5", "level5", "bonus5", "baseCount5", "baseAdd5", "powerA5", "powerB5", "spellClass5", "diceZ5", "diceN5", "diceS5", "realm5", "active6", "spell6", "level6", "bonus6", "baseCount6", "baseAdd6", "powerA6", "powerB6", "spellClass6", "diceZ6", "diceN6", "diceS6", "realm6", "active7", "spell7", "level7", "bonus7", "baseCount7", "baseAdd7", "powerA7", "powerB7", "spellClass7", "diceZ7", "diceN7", "diceS7", "realm7", "active8", "spell8", "level8", "bonus8", "baseCount8", "baseAdd8", "powerA8", "powerB8", "spellClass8", "diceZ8", "diceN8", "diceS8", "realm8", "present0", "condition0", "charLevel0", "charRes0_0", "charRes0_1", "charRes0_2", "charRes0_3", "charRes0_4", "charRes0_5", "growth0", "reverse0", "present1", "condition1", "charLevel1", "charRes1_0", "charRes1_1", "charRes1_2", "charRes1_3", "charRes1_4", "charRes1_5", "growth1", "reverse1", "present2", "condition2", "charLevel2", "charRes2_0", "charRes2_1", "charRes2_2", "charRes2_3", "charRes2_4", "charRes2_5", "growth2", "reverse2", "present3", "condition3", "charLevel3", "charRes3_0", "charRes3_1", "charRes3_2", "charRes3_3", "charRes3_4", "charRes3_5", "growth3", "reverse3", "present4", "condition4", "charLevel4", "charRes4_0", "charRes4_1", "charRes4_2", "charRes4_3", "charRes4_4", "charRes4_5", "growth4", "reverse4", "present5", "condition5", "charLevel5", "charRes5_0", "charRes5_1", "charRes5_2", "charRes5_3", "charRes5_4", "charRes5_5", "growth5", "reverse5", "present6", "condition6", "charLevel6", "charRes6_0", "charRes6_1", "charRes6_2", "charRes6_3", "charRes6_4", "charRes6_5", "growth6", "reverse6", "present7", "condition7", "charLevel7", "charRes7_0", "charRes7_1", "charRes7_2", "charRes7_3", "charRes7_4", "charRes7_5", "growth7", "reverse7", "charLevelM1", "charResM1_0", "charResM1_1", "charResM1_2", "charResM1_3", "charResM1_4", "charResM1_5", "hp0", "maxHp0", "shield0", "shieldDuration0", "maxStamina0", "stamina0", "exhaustion0", "messages0", "group0", "actorState0", "templateKind0", "mana0", "monMod0_0", "monMod0_1", "monMod0_2", "monMod0_3", "monMod0_4", "monMod0_5", "monCondition0", "monClass0", "immune0", "tplRes0_0", "tplRes0_1", "tplRes0_2", "tplRes0_3", "tplRes0_4", "tplRes0_5", "tplLevel0", "hp1", "maxHp1", "shield1", "shieldDuration1", "maxStamina1", "stamina1", "exhaustion1", "messages1", "group1", "actorState1", "templateKind1", "mana1", "monMod1_0", "monMod1_1", "monMod1_2", "monMod1_3", "monMod1_4", "monMod1_5", "monCondition1", "monClass1", "immune1", "tplRes1_0", "tplRes1_1", "tplRes1_2", "tplRes1_3", "tplRes1_4", "tplRes1_5", "tplLevel1", "groupNamed", "resistedText0", "resistedText1", "statusResisted0", "statusResisted1", "statusResisted2", "statusResisted3", "statusResisted4", "statusResisted5", "statusResisted6", "statusResisted7", "statusResisted8", "statusResisted9", "statusResisted10", "statusResisted11", "statusResisted12", "statusResisted13", "statusResisted14", "statusResisted15", "statusResisted16", "statusResisted17", "statusResisted18", "statusResisted19", "statusResisted20", "statusResisted21", "statusResisted22", "statusResisted23", "statusResisted24", "statusResisted25", "statusResisted26", "statusResisted27", "statusResisted28", "statusResisted29", "statusResisted30", "statusResisted31", "statusResisted32", "statusResisted33", "statusResisted34", "statusResisted35", "statusResisted36", "statusResisted37", "statusResisted38", "statusResisted39", "statusResisted40", "statusResisted41", "statusResisted42", "statusResisted43", "statusResisted44", "statusResisted45", "statusResisted46", "statusResisted47", "statusResisted48", "statusResisted49", "statusResisted50", "statusResisted51", "statusResisted52", "statusResisted53", "statusResisted54", "statusResisted55", "statusResisted56", "statusResisted57", "statusResisted58", "statusResisted59", "statusResisted60", "statusResisted61", "statusResisted62", "statusResisted63", "ignoredReturn"}

var ostOut = []string{"statusResistCalls", "statusResistArg1_0", "statusResistArg1_1", "statusResistArg1_2", "statusResistArg1_3", "statusResistArg1_4", "statusResistArg1_5", "statusResistArg1_6", "statusResistArg1_7", "statusResistArg1_8", "statusResistArg1_9", "statusResistArg1_10", "statusResistArg1_11", "statusResistArg1_12", "statusResistArg1_13", "statusResistArg1_14", "statusResistArg1_15", "statusResistArg1_16", "statusResistArg1_17", "statusResistArg1_18", "statusResistArg1_19", "statusResistArg1_20", "statusResistArg1_21", "statusResistArg1_22", "statusResistArg1_23", "statusResistArg1_24", "statusResistArg1_25", "statusResistArg1_26", "statusResistArg1_27", "statusResistArg1_28", "statusResistArg1_29", "statusResistArg1_30", "statusResistArg1_31", "statusResistArg1_32", "statusResistArg1_33", "statusResistArg1_34", "statusResistArg1_35", "statusResistArg1_36", "statusResistArg1_37", "statusResistArg1_38", "statusResistArg1_39", "statusResistArg1_40", "statusResistArg1_41", "statusResistArg1_42", "statusResistArg1_43", "statusResistArg1_44", "statusResistArg1_45", "statusResistArg1_46", "statusResistArg1_47", "statusResistArg1_48", "statusResistArg1_49", "statusResistArg1_50", "statusResistArg1_51", "statusResistArg1_52", "statusResistArg1_53", "statusResistArg1_54", "statusResistArg1_55", "statusResistArg1_56", "statusResistArg1_57", "statusResistArg1_58", "statusResistArg1_59", "statusResistArg1_60", "statusResistArg1_61", "statusResistArg1_62", "statusResistArg1_63", "statusResistArg2_0", "statusResistArg2_1", "statusResistArg2_2", "statusResistArg2_3", "statusResistArg2_4", "statusResistArg2_5", "statusResistArg2_6", "statusResistArg2_7", "statusResistArg2_8", "statusResistArg2_9", "statusResistArg2_10", "statusResistArg2_11", "statusResistArg2_12", "statusResistArg2_13", "statusResistArg2_14", "statusResistArg2_15", "statusResistArg2_16", "statusResistArg2_17", "statusResistArg2_18", "statusResistArg2_19", "statusResistArg2_20", "statusResistArg2_21", "statusResistArg2_22", "statusResistArg2_23", "statusResistArg2_24", "statusResistArg2_25", "statusResistArg2_26", "statusResistArg2_27", "statusResistArg2_28", "statusResistArg2_29", "statusResistArg2_30", "statusResistArg2_31", "statusResistArg2_32", "statusResistArg2_33", "statusResistArg2_34", "statusResistArg2_35", "statusResistArg2_36", "statusResistArg2_37", "statusResistArg2_38", "statusResistArg2_39", "statusResistArg2_40", "statusResistArg2_41", "statusResistArg2_42", "statusResistArg2_43", "statusResistArg2_44", "statusResistArg2_45", "statusResistArg2_46", "statusResistArg2_47", "statusResistArg2_48", "statusResistArg2_49", "statusResistArg2_50", "statusResistArg2_51", "statusResistArg2_52", "statusResistArg2_53", "statusResistArg2_54", "statusResistArg2_55", "statusResistArg2_56", "statusResistArg2_57", "statusResistArg2_58", "statusResistArg2_59", "statusResistArg2_60", "statusResistArg2_61", "statusResistArg2_62", "statusResistArg2_63", "statusResistArg3_0", "statusResistArg3_1", "statusResistArg3_2", "statusResistArg3_3", "statusResistArg3_4", "statusResistArg3_5", "statusResistArg3_6", "statusResistArg3_7", "statusResistArg3_8", "statusResistArg3_9", "statusResistArg3_10", "statusResistArg3_11", "statusResistArg3_12", "statusResistArg3_13", "statusResistArg3_14", "statusResistArg3_15", "statusResistArg3_16", "statusResistArg3_17", "statusResistArg3_18", "statusResistArg3_19", "statusResistArg3_20", "statusResistArg3_21", "statusResistArg3_22", "statusResistArg3_23", "statusResistArg3_24", "statusResistArg3_25", "statusResistArg3_26", "statusResistArg3_27", "statusResistArg3_28", "statusResistArg3_29", "statusResistArg3_30", "statusResistArg3_31", "statusResistArg3_32", "statusResistArg3_33", "statusResistArg3_34", "statusResistArg3_35", "statusResistArg3_36", "statusResistArg3_37", "statusResistArg3_38", "statusResistArg3_39", "statusResistArg3_40", "statusResistArg3_41", "statusResistArg3_42", "statusResistArg3_43", "statusResistArg3_44", "statusResistArg3_45", "statusResistArg3_46", "statusResistArg3_47", "statusResistArg3_48", "statusResistArg3_49", "statusResistArg3_50", "statusResistArg3_51", "statusResistArg3_52", "statusResistArg3_53", "statusResistArg3_54", "statusResistArg3_55", "statusResistArg3_56", "statusResistArg3_57", "statusResistArg3_58", "statusResistArg3_59", "statusResistArg3_60", "statusResistArg3_61", "statusResistArg3_62", "statusResistArg3_63", "statusResistArg4_0", "statusResistArg4_1", "statusResistArg4_2", "statusResistArg4_3", "statusResistArg4_4", "statusResistArg4_5", "statusResistArg4_6", "statusResistArg4_7", "statusResistArg4_8", "statusResistArg4_9", "statusResistArg4_10", "statusResistArg4_11", "statusResistArg4_12", "statusResistArg4_13", "statusResistArg4_14", "statusResistArg4_15", "statusResistArg4_16", "statusResistArg4_17", "statusResistArg4_18", "statusResistArg4_19", "statusResistArg4_20", "statusResistArg4_21", "statusResistArg4_22", "statusResistArg4_23", "statusResistArg4_24", "statusResistArg4_25", "statusResistArg4_26", "statusResistArg4_27", "statusResistArg4_28", "statusResistArg4_29", "statusResistArg4_30", "statusResistArg4_31", "statusResistArg4_32", "statusResistArg4_33", "statusResistArg4_34", "statusResistArg4_35", "statusResistArg4_36", "statusResistArg4_37", "statusResistArg4_38", "statusResistArg4_39", "statusResistArg4_40", "statusResistArg4_41", "statusResistArg4_42", "statusResistArg4_43", "statusResistArg4_44", "statusResistArg4_45", "statusResistArg4_46", "statusResistArg4_47", "statusResistArg4_48", "statusResistArg4_49", "statusResistArg4_50", "statusResistArg4_51", "statusResistArg4_52", "statusResistArg4_53", "statusResistArg4_54", "statusResistArg4_55", "statusResistArg4_56", "statusResistArg4_57", "statusResistArg4_58", "statusResistArg4_59", "statusResistArg4_60", "statusResistArg4_61", "statusResistArg4_62", "statusResistArg4_63", "charDamageCalls", "charDamageArg1_0", "charDamageArg1_1", "charDamageArg1_2", "charDamageArg1_3", "charDamageArg1_4", "charDamageArg1_5", "charDamageArg1_6", "charDamageArg1_7", "charDamageArg1_8", "charDamageArg1_9", "charDamageArg1_10", "charDamageArg1_11", "charDamageArg1_12", "charDamageArg1_13", "charDamageArg1_14", "charDamageArg1_15", "charDamageArg1_16", "charDamageArg1_17", "charDamageArg1_18", "charDamageArg1_19", "charDamageArg1_20", "charDamageArg1_21", "charDamageArg1_22", "charDamageArg1_23", "charDamageArg1_24", "charDamageArg1_25", "charDamageArg1_26", "charDamageArg1_27", "charDamageArg1_28", "charDamageArg1_29", "charDamageArg1_30", "charDamageArg1_31", "charDamageArg1_32", "charDamageArg1_33", "charDamageArg1_34", "charDamageArg1_35", "charDamageArg1_36", "charDamageArg1_37", "charDamageArg1_38", "charDamageArg1_39", "charDamageArg1_40", "charDamageArg1_41", "charDamageArg1_42", "charDamageArg1_43", "charDamageArg1_44", "charDamageArg1_45", "charDamageArg1_46", "charDamageArg1_47", "charDamageArg1_48", "charDamageArg1_49", "charDamageArg1_50", "charDamageArg1_51", "charDamageArg1_52", "charDamageArg1_53", "charDamageArg1_54", "charDamageArg1_55", "charDamageArg1_56", "charDamageArg1_57", "charDamageArg1_58", "charDamageArg1_59", "charDamageArg1_60", "charDamageArg1_61", "charDamageArg1_62", "charDamageArg1_63", "charDamageArg1_64", "charDamageArg1_65", "charDamageArg1_66", "charDamageArg1_67", "charDamageArg1_68", "charDamageArg1_69", "charDamageArg1_70", "charDamageArg1_71", "charDamageArg2_0", "charDamageArg2_1", "charDamageArg2_2", "charDamageArg2_3", "charDamageArg2_4", "charDamageArg2_5", "charDamageArg2_6", "charDamageArg2_7", "charDamageArg2_8", "charDamageArg2_9", "charDamageArg2_10", "charDamageArg2_11", "charDamageArg2_12", "charDamageArg2_13", "charDamageArg2_14", "charDamageArg2_15", "charDamageArg2_16", "charDamageArg2_17", "charDamageArg2_18", "charDamageArg2_19", "charDamageArg2_20", "charDamageArg2_21", "charDamageArg2_22", "charDamageArg2_23", "charDamageArg2_24", "charDamageArg2_25", "charDamageArg2_26", "charDamageArg2_27", "charDamageArg2_28", "charDamageArg2_29", "charDamageArg2_30", "charDamageArg2_31", "charDamageArg2_32", "charDamageArg2_33", "charDamageArg2_34", "charDamageArg2_35", "charDamageArg2_36", "charDamageArg2_37", "charDamageArg2_38", "charDamageArg2_39", "charDamageArg2_40", "charDamageArg2_41", "charDamageArg2_42", "charDamageArg2_43", "charDamageArg2_44", "charDamageArg2_45", "charDamageArg2_46", "charDamageArg2_47", "charDamageArg2_48", "charDamageArg2_49", "charDamageArg2_50", "charDamageArg2_51", "charDamageArg2_52", "charDamageArg2_53", "charDamageArg2_54", "charDamageArg2_55", "charDamageArg2_56", "charDamageArg2_57", "charDamageArg2_58", "charDamageArg2_59", "charDamageArg2_60", "charDamageArg2_61", "charDamageArg2_62", "charDamageArg2_63", "charDamageArg2_64", "charDamageArg2_65", "charDamageArg2_66", "charDamageArg2_67", "charDamageArg2_68", "charDamageArg2_69", "charDamageArg2_70", "charDamageArg2_71", "charDamageArg4_0", "charDamageArg4_1", "charDamageArg4_2", "charDamageArg4_3", "charDamageArg4_4", "charDamageArg4_5", "charDamageArg4_6", "charDamageArg4_7", "charDamageArg4_8", "charDamageArg4_9", "charDamageArg4_10", "charDamageArg4_11", "charDamageArg4_12", "charDamageArg4_13", "charDamageArg4_14", "charDamageArg4_15", "charDamageArg4_16", "charDamageArg4_17", "charDamageArg4_18", "charDamageArg4_19", "charDamageArg4_20", "charDamageArg4_21", "charDamageArg4_22", "charDamageArg4_23", "charDamageArg4_24", "charDamageArg4_25", "charDamageArg4_26", "charDamageArg4_27", "charDamageArg4_28", "charDamageArg4_29", "charDamageArg4_30", "charDamageArg4_31", "charDamageArg4_32", "charDamageArg4_33", "charDamageArg4_34", "charDamageArg4_35", "charDamageArg4_36", "charDamageArg4_37", "charDamageArg4_38", "charDamageArg4_39", "charDamageArg4_40", "charDamageArg4_41", "charDamageArg4_42", "charDamageArg4_43", "charDamageArg4_44", "charDamageArg4_45", "charDamageArg4_46", "charDamageArg4_47", "charDamageArg4_48", "charDamageArg4_49", "charDamageArg4_50", "charDamageArg4_51", "charDamageArg4_52", "charDamageArg4_53", "charDamageArg4_54", "charDamageArg4_55", "charDamageArg4_56", "charDamageArg4_57", "charDamageArg4_58", "charDamageArg4_59", "charDamageArg4_60", "charDamageArg4_61", "charDamageArg4_62", "charDamageArg4_63", "charDamageArg4_64", "charDamageArg4_65", "charDamageArg4_66", "charDamageArg4_67", "charDamageArg4_68", "charDamageArg4_69", "charDamageArg4_70", "charDamageArg4_71", "charRealmDamageCalls", "charRealmDamageArg1_0", "charRealmDamageArg1_1", "charRealmDamageArg1_2", "charRealmDamageArg1_3", "charRealmDamageArg1_4", "charRealmDamageArg1_5", "charRealmDamageArg1_6", "charRealmDamageArg1_7", "charRealmDamageArg1_8", "charRealmDamageArg1_9", "charRealmDamageArg1_10", "charRealmDamageArg1_11", "charRealmDamageArg1_12", "charRealmDamageArg1_13", "charRealmDamageArg1_14", "charRealmDamageArg1_15", "charRealmDamageArg1_16", "charRealmDamageArg1_17", "charRealmDamageArg1_18", "charRealmDamageArg1_19", "charRealmDamageArg1_20", "charRealmDamageArg1_21", "charRealmDamageArg1_22", "charRealmDamageArg1_23", "charRealmDamageArg1_24", "charRealmDamageArg1_25", "charRealmDamageArg1_26", "charRealmDamageArg1_27", "charRealmDamageArg1_28", "charRealmDamageArg1_29", "charRealmDamageArg1_30", "charRealmDamageArg1_31", "charRealmDamageArg1_32", "charRealmDamageArg1_33", "charRealmDamageArg1_34", "charRealmDamageArg1_35", "charRealmDamageArg1_36", "charRealmDamageArg1_37", "charRealmDamageArg1_38", "charRealmDamageArg1_39", "charRealmDamageArg1_40", "charRealmDamageArg1_41", "charRealmDamageArg1_42", "charRealmDamageArg1_43", "charRealmDamageArg1_44", "charRealmDamageArg1_45", "charRealmDamageArg1_46", "charRealmDamageArg1_47", "charRealmDamageArg2_0", "charRealmDamageArg2_1", "charRealmDamageArg2_2", "charRealmDamageArg2_3", "charRealmDamageArg2_4", "charRealmDamageArg2_5", "charRealmDamageArg2_6", "charRealmDamageArg2_7", "charRealmDamageArg2_8", "charRealmDamageArg2_9", "charRealmDamageArg2_10", "charRealmDamageArg2_11", "charRealmDamageArg2_12", "charRealmDamageArg2_13", "charRealmDamageArg2_14", "charRealmDamageArg2_15", "charRealmDamageArg2_16", "charRealmDamageArg2_17", "charRealmDamageArg2_18", "charRealmDamageArg2_19", "charRealmDamageArg2_20", "charRealmDamageArg2_21", "charRealmDamageArg2_22", "charRealmDamageArg2_23", "charRealmDamageArg2_24", "charRealmDamageArg2_25", "charRealmDamageArg2_26", "charRealmDamageArg2_27", "charRealmDamageArg2_28", "charRealmDamageArg2_29", "charRealmDamageArg2_30", "charRealmDamageArg2_31", "charRealmDamageArg2_32", "charRealmDamageArg2_33", "charRealmDamageArg2_34", "charRealmDamageArg2_35", "charRealmDamageArg2_36", "charRealmDamageArg2_37", "charRealmDamageArg2_38", "charRealmDamageArg2_39", "charRealmDamageArg2_40", "charRealmDamageArg2_41", "charRealmDamageArg2_42", "charRealmDamageArg2_43", "charRealmDamageArg2_44", "charRealmDamageArg2_45", "charRealmDamageArg2_46", "charRealmDamageArg2_47", "charRealmDamageArg3_0", "charRealmDamageArg3_1", "charRealmDamageArg3_2", "charRealmDamageArg3_3", "charRealmDamageArg3_4", "charRealmDamageArg3_5", "charRealmDamageArg3_6", "charRealmDamageArg3_7", "charRealmDamageArg3_8", "charRealmDamageArg3_9", "charRealmDamageArg3_10", "charRealmDamageArg3_11", "charRealmDamageArg3_12", "charRealmDamageArg3_13", "charRealmDamageArg3_14", "charRealmDamageArg3_15", "charRealmDamageArg3_16", "charRealmDamageArg3_17", "charRealmDamageArg3_18", "charRealmDamageArg3_19", "charRealmDamageArg3_20", "charRealmDamageArg3_21", "charRealmDamageArg3_22", "charRealmDamageArg3_23", "charRealmDamageArg3_24", "charRealmDamageArg3_25", "charRealmDamageArg3_26", "charRealmDamageArg3_27", "charRealmDamageArg3_28", "charRealmDamageArg3_29", "charRealmDamageArg3_30", "charRealmDamageArg3_31", "charRealmDamageArg3_32", "charRealmDamageArg3_33", "charRealmDamageArg3_34", "charRealmDamageArg3_35", "charRealmDamageArg3_36", "charRealmDamageArg3_37", "charRealmDamageArg3_38", "charRealmDamageArg3_39", "charRealmDamageArg3_40", "charRealmDamageArg3_41", "charRealmDamageArg3_42", "charRealmDamageArg3_43", "charRealmDamageArg3_44", "charRealmDamageArg3_45", "charRealmDamageArg3_46", "charRealmDamageArg3_47", "charFatigueCalls", "charFatigueArg1_0", "charFatigueArg1_1", "charFatigueArg1_2", "charFatigueArg1_3", "charFatigueArg1_4", "charFatigueArg1_5", "charFatigueArg1_6", "charFatigueArg1_7", "charFatigueArg2_0", "charFatigueArg2_1", "charFatigueArg2_2", "charFatigueArg2_3", "charFatigueArg2_4", "charFatigueArg2_5", "charFatigueArg2_6", "charFatigueArg2_7", "charConditionCalls", "charConditionArg1_0", "charConditionArg1_1", "charConditionArg1_2", "charConditionArg1_3", "charConditionArg1_4", "charConditionArg1_5", "charConditionArg1_6", "charConditionArg1_7", "charConditionArg1_8", "charConditionArg1_9", "charConditionArg1_10", "charConditionArg1_11", "charConditionArg1_12", "charConditionArg1_13", "charConditionArg1_14", "charConditionArg1_15", "charConditionArg1_16", "charConditionArg1_17", "charConditionArg1_18", "charConditionArg1_19", "charConditionArg1_20", "charConditionArg1_21", "charConditionArg1_22", "charConditionArg1_23", "charConditionArg1_24", "charConditionArg1_25", "charConditionArg1_26", "charConditionArg1_27", "charConditionArg1_28", "charConditionArg1_29", "charConditionArg1_30", "charConditionArg1_31", "charConditionArg1_32", "charConditionArg1_33", "charConditionArg1_34", "charConditionArg1_35", "charConditionArg1_36", "charConditionArg1_37", "charConditionArg1_38", "charConditionArg1_39", "charConditionArg1_40", "charConditionArg1_41", "charConditionArg1_42", "charConditionArg1_43", "charConditionArg1_44", "charConditionArg1_45", "charConditionArg1_46", "charConditionArg1_47", "charConditionArg1_48", "charConditionArg1_49", "charConditionArg1_50", "charConditionArg1_51", "charConditionArg1_52", "charConditionArg1_53", "charConditionArg1_54", "charConditionArg1_55", "charConditionArg1_56", "charConditionArg1_57", "charConditionArg1_58", "charConditionArg1_59", "charConditionArg1_60", "charConditionArg1_61", "charConditionArg1_62", "charConditionArg1_63", "charConditionArg2_0", "charConditionArg2_1", "charConditionArg2_2", "charConditionArg2_3", "charConditionArg2_4", "charConditionArg2_5", "charConditionArg2_6", "charConditionArg2_7", "charConditionArg2_8", "charConditionArg2_9", "charConditionArg2_10", "charConditionArg2_11", "charConditionArg2_12", "charConditionArg2_13", "charConditionArg2_14", "charConditionArg2_15", "charConditionArg2_16", "charConditionArg2_17", "charConditionArg2_18", "charConditionArg2_19", "charConditionArg2_20", "charConditionArg2_21", "charConditionArg2_22", "charConditionArg2_23", "charConditionArg2_24", "charConditionArg2_25", "charConditionArg2_26", "charConditionArg2_27", "charConditionArg2_28", "charConditionArg2_29", "charConditionArg2_30", "charConditionArg2_31", "charConditionArg2_32", "charConditionArg2_33", "charConditionArg2_34", "charConditionArg2_35", "charConditionArg2_36", "charConditionArg2_37", "charConditionArg2_38", "charConditionArg2_39", "charConditionArg2_40", "charConditionArg2_41", "charConditionArg2_42", "charConditionArg2_43", "charConditionArg2_44", "charConditionArg2_45", "charConditionArg2_46", "charConditionArg2_47", "charConditionArg2_48", "charConditionArg2_49", "charConditionArg2_50", "charConditionArg2_51", "charConditionArg2_52", "charConditionArg2_53", "charConditionArg2_54", "charConditionArg2_55", "charConditionArg2_56", "charConditionArg2_57", "charConditionArg2_58", "charConditionArg2_59", "charConditionArg2_60", "charConditionArg2_61", "charConditionArg2_62", "charConditionArg2_63", "charConditionArg3_0", "charConditionArg3_1", "charConditionArg3_2", "charConditionArg3_3", "charConditionArg3_4", "charConditionArg3_5", "charConditionArg3_6", "charConditionArg3_7", "charConditionArg3_8", "charConditionArg3_9", "charConditionArg3_10", "charConditionArg3_11", "charConditionArg3_12", "charConditionArg3_13", "charConditionArg3_14", "charConditionArg3_15", "charConditionArg3_16", "charConditionArg3_17", "charConditionArg3_18", "charConditionArg3_19", "charConditionArg3_20", "charConditionArg3_21", "charConditionArg3_22", "charConditionArg3_23", "charConditionArg3_24", "charConditionArg3_25", "charConditionArg3_26", "charConditionArg3_27", "charConditionArg3_28", "charConditionArg3_29", "charConditionArg3_30", "charConditionArg3_31", "charConditionArg3_32", "charConditionArg3_33", "charConditionArg3_34", "charConditionArg3_35", "charConditionArg3_36", "charConditionArg3_37", "charConditionArg3_38", "charConditionArg3_39", "charConditionArg3_40", "charConditionArg3_41", "charConditionArg3_42", "charConditionArg3_43", "charConditionArg3_44", "charConditionArg3_45", "charConditionArg3_46", "charConditionArg3_47", "charConditionArg3_48", "charConditionArg3_49", "charConditionArg3_50", "charConditionArg3_51", "charConditionArg3_52", "charConditionArg3_53", "charConditionArg3_54", "charConditionArg3_55", "charConditionArg3_56", "charConditionArg3_57", "charConditionArg3_58", "charConditionArg3_59", "charConditionArg3_60", "charConditionArg3_61", "charConditionArg3_62", "charConditionArg3_63", "charConditionArg4_0", "charConditionArg4_1", "charConditionArg4_2", "charConditionArg4_3", "charConditionArg4_4", "charConditionArg4_5", "charConditionArg4_6", "charConditionArg4_7", "charConditionArg4_8", "charConditionArg4_9", "charConditionArg4_10", "charConditionArg4_11", "charConditionArg4_12", "charConditionArg4_13", "charConditionArg4_14", "charConditionArg4_15", "charConditionArg4_16", "charConditionArg4_17", "charConditionArg4_18", "charConditionArg4_19", "charConditionArg4_20", "charConditionArg4_21", "charConditionArg4_22", "charConditionArg4_23", "charConditionArg4_24", "charConditionArg4_25", "charConditionArg4_26", "charConditionArg4_27", "charConditionArg4_28", "charConditionArg4_29", "charConditionArg4_30", "charConditionArg4_31", "charConditionArg4_32", "charConditionArg4_33", "charConditionArg4_34", "charConditionArg4_35", "charConditionArg4_36", "charConditionArg4_37", "charConditionArg4_38", "charConditionArg4_39", "charConditionArg4_40", "charConditionArg4_41", "charConditionArg4_42", "charConditionArg4_43", "charConditionArg4_44", "charConditionArg4_45", "charConditionArg4_46", "charConditionArg4_47", "charConditionArg4_48", "charConditionArg4_49", "charConditionArg4_50", "charConditionArg4_51", "charConditionArg4_52", "charConditionArg4_53", "charConditionArg4_54", "charConditionArg4_55", "charConditionArg4_56", "charConditionArg4_57", "charConditionArg4_58", "charConditionArg4_59", "charConditionArg4_60", "charConditionArg4_61", "charConditionArg4_62", "charConditionArg4_63", "monConditionCalls", "monConditionArg1_0", "monConditionArg1_1", "monConditionArg1_2", "monConditionArg1_3", "monConditionArg1_4", "monConditionArg1_5", "monConditionArg1_6", "monConditionArg1_7", "monConditionArg1_8", "monConditionArg1_9", "monConditionArg1_10", "monConditionArg1_11", "monConditionArg1_12", "monConditionArg1_13", "monConditionArg1_14", "monConditionArg1_15", "monConditionArg1_16", "monConditionArg1_17", "monConditionArg1_18", "monConditionArg1_19", "monConditionArg1_20", "monConditionArg1_21", "monConditionArg1_22", "monConditionArg1_23", "monConditionArg1_24", "monConditionArg1_25", "monConditionArg1_26", "monConditionArg1_27", "monConditionArg1_28", "monConditionArg1_29", "monConditionArg1_30", "monConditionArg1_31", "monConditionArg1_32", "monConditionArg1_33", "monConditionArg1_34", "monConditionArg1_35", "monConditionArg1_36", "monConditionArg1_37", "monConditionArg1_38", "monConditionArg1_39", "monConditionArg2_0", "monConditionArg2_1", "monConditionArg2_2", "monConditionArg2_3", "monConditionArg2_4", "monConditionArg2_5", "monConditionArg2_6", "monConditionArg2_7", "monConditionArg2_8", "monConditionArg2_9", "monConditionArg2_10", "monConditionArg2_11", "monConditionArg2_12", "monConditionArg2_13", "monConditionArg2_14", "monConditionArg2_15", "monConditionArg2_16", "monConditionArg2_17", "monConditionArg2_18", "monConditionArg2_19", "monConditionArg2_20", "monConditionArg2_21", "monConditionArg2_22", "monConditionArg2_23", "monConditionArg2_24", "monConditionArg2_25", "monConditionArg2_26", "monConditionArg2_27", "monConditionArg2_28", "monConditionArg2_29", "monConditionArg2_30", "monConditionArg2_31", "monConditionArg2_32", "monConditionArg2_33", "monConditionArg2_34", "monConditionArg2_35", "monConditionArg2_36", "monConditionArg2_37", "monConditionArg2_38", "monConditionArg2_39", "monConditionArg3_0", "monConditionArg3_1", "monConditionArg3_2", "monConditionArg3_3", "monConditionArg3_4", "monConditionArg3_5", "monConditionArg3_6", "monConditionArg3_7", "monConditionArg3_8", "monConditionArg3_9", "monConditionArg3_10", "monConditionArg3_11", "monConditionArg3_12", "monConditionArg3_13", "monConditionArg3_14", "monConditionArg3_15", "monConditionArg3_16", "monConditionArg3_17", "monConditionArg3_18", "monConditionArg3_19", "monConditionArg3_20", "monConditionArg3_21", "monConditionArg3_22", "monConditionArg3_23", "monConditionArg3_24", "monConditionArg3_25", "monConditionArg3_26", "monConditionArg3_27", "monConditionArg3_28", "monConditionArg3_29", "monConditionArg3_30", "monConditionArg3_31", "monConditionArg3_32", "monConditionArg3_33", "monConditionArg3_34", "monConditionArg3_35", "monConditionArg3_36", "monConditionArg3_37", "monConditionArg3_38", "monConditionArg3_39", "monConditionArg4_0", "monConditionArg4_1", "monConditionArg4_2", "monConditionArg4_3", "monConditionArg4_4", "monConditionArg4_5", "monConditionArg4_6", "monConditionArg4_7", "monConditionArg4_8", "monConditionArg4_9", "monConditionArg4_10", "monConditionArg4_11", "monConditionArg4_12", "monConditionArg4_13", "monConditionArg4_14", "monConditionArg4_15", "monConditionArg4_16", "monConditionArg4_17", "monConditionArg4_18", "monConditionArg4_19", "monConditionArg4_20", "monConditionArg4_21", "monConditionArg4_22", "monConditionArg4_23", "monConditionArg4_24", "monConditionArg4_25", "monConditionArg4_26", "monConditionArg4_27", "monConditionArg4_28", "monConditionArg4_29", "monConditionArg4_30", "monConditionArg4_31", "monConditionArg4_32", "monConditionArg4_33", "monConditionArg4_34", "monConditionArg4_35", "monConditionArg4_36", "monConditionArg4_37", "monConditionArg4_38", "monConditionArg4_39", "monConditionArg5_0", "monConditionArg5_1", "monConditionArg5_2", "monConditionArg5_3", "monConditionArg5_4", "monConditionArg5_5", "monConditionArg5_6", "monConditionArg5_7", "monConditionArg5_8", "monConditionArg5_9", "monConditionArg5_10", "monConditionArg5_11", "monConditionArg5_12", "monConditionArg5_13", "monConditionArg5_14", "monConditionArg5_15", "monConditionArg5_16", "monConditionArg5_17", "monConditionArg5_18", "monConditionArg5_19", "monConditionArg5_20", "monConditionArg5_21", "monConditionArg5_22", "monConditionArg5_23", "monConditionArg5_24", "monConditionArg5_25", "monConditionArg5_26", "monConditionArg5_27", "monConditionArg5_28", "monConditionArg5_29", "monConditionArg5_30", "monConditionArg5_31", "monConditionArg5_32", "monConditionArg5_33", "monConditionArg5_34", "monConditionArg5_35", "monConditionArg5_36", "monConditionArg5_37", "monConditionArg5_38", "monConditionArg5_39", "monConditionArg6_0", "monConditionArg6_1", "monConditionArg6_2", "monConditionArg6_3", "monConditionArg6_4", "monConditionArg6_5", "monConditionArg6_6", "monConditionArg6_7", "monConditionArg6_8", "monConditionArg6_9", "monConditionArg6_10", "monConditionArg6_11", "monConditionArg6_12", "monConditionArg6_13", "monConditionArg6_14", "monConditionArg6_15", "monConditionArg6_16", "monConditionArg6_17", "monConditionArg6_18", "monConditionArg6_19", "monConditionArg6_20", "monConditionArg6_21", "monConditionArg6_22", "monConditionArg6_23", "monConditionArg6_24", "monConditionArg6_25", "monConditionArg6_26", "monConditionArg6_27", "monConditionArg6_28", "monConditionArg6_29", "monConditionArg6_30", "monConditionArg6_31", "monConditionArg6_32", "monConditionArg6_33", "monConditionArg6_34", "monConditionArg6_35", "monConditionArg6_36", "monConditionArg6_37", "monConditionArg6_38", "monConditionArg6_39", "hitResponseCalls", "hitResponseArg1_0", "hitResponseArg1_1", "hitResponseArg1_2", "hitResponseArg1_3", "hitResponseArg1_4", "hitResponseArg1_5", "hitResponseArg1_6", "hitResponseArg1_7", "hitResponseArg1_8", "hitResponseArg3_0", "hitResponseArg3_1", "hitResponseArg3_2", "hitResponseArg3_3", "hitResponseArg3_4", "hitResponseArg3_5", "hitResponseArg3_6", "hitResponseArg3_7", "hitResponseArg3_8", "removeStateCalls", "removeStateArg1_0", "removeStateArg1_1", "killRewardsCalls", "killRewardsArg1_0", "killRewardsArg1_1", "killRewardsArg2_0", "killRewardsArg2_1", "messageCalls", "messageArg1_0", "messageArg1_1", "messageArg1_2", "messageArg1_3", "messageArg1_4", "messageArg1_5", "messageArg1_6", "messageArg1_7", "messageArg1_8", "messageArg1_9", "messageArg1_10", "messageArg1_11", "messageArg1_12", "messageArg1_13", "messageArg1_14", "messageArg1_15", "messageArg1_16", "messageArg1_17", "messageArg1_18", "messageArg1_19", "messageArg1_20", "messageArg1_21", "messageArg1_22", "messageArg1_23", "messageArg1_24", "messageArg1_25", "messageArg1_26", "messageArg1_27", "messageArg1_28", "messageArg1_29", "messageArg1_30", "messageArg1_31", "messageArg1_32", "messageArg1_33", "messageArg1_34", "messageArg1_35", "messageArg1_36", "messageArg1_37", "messageArg1_38", "messageArg1_39", "messageArg1_40", "messageArg1_41", "messageArg1_42", "messageArg1_43", "messageArg1_44", "messageArg1_45", "messageArg1_46", "messageArg1_47", "messageArg2_0", "messageArg2_1", "messageArg2_2", "messageArg2_3", "messageArg2_4", "messageArg2_5", "messageArg2_6", "messageArg2_7", "messageArg2_8", "messageArg2_9", "messageArg2_10", "messageArg2_11", "messageArg2_12", "messageArg2_13", "messageArg2_14", "messageArg2_15", "messageArg2_16", "messageArg2_17", "messageArg2_18", "messageArg2_19", "messageArg2_20", "messageArg2_21", "messageArg2_22", "messageArg2_23", "messageArg2_24", "messageArg2_25", "messageArg2_26", "messageArg2_27", "messageArg2_28", "messageArg2_29", "messageArg2_30", "messageArg2_31", "messageArg2_32", "messageArg2_33", "messageArg2_34", "messageArg2_35", "messageArg2_36", "messageArg2_37", "messageArg2_38", "messageArg2_39", "messageArg2_40", "messageArg2_41", "messageArg2_42", "messageArg2_43", "messageArg2_44", "messageArg2_45", "messageArg2_46", "messageArg2_47", "messageArg3_0", "messageArg3_1", "messageArg3_2", "messageArg3_3", "messageArg3_4", "messageArg3_5", "messageArg3_6", "messageArg3_7", "messageArg3_8", "messageArg3_9", "messageArg3_10", "messageArg3_11", "messageArg3_12", "messageArg3_13", "messageArg3_14", "messageArg3_15", "messageArg3_16", "messageArg3_17", "messageArg3_18", "messageArg3_19", "messageArg3_20", "messageArg3_21", "messageArg3_22", "messageArg3_23", "messageArg3_24", "messageArg3_25", "messageArg3_26", "messageArg3_27", "messageArg3_28", "messageArg3_29", "messageArg3_30", "messageArg3_31", "messageArg3_32", "messageArg3_33", "messageArg3_34", "messageArg3_35", "messageArg3_36", "messageArg3_37", "messageArg3_38", "messageArg3_39", "messageArg3_40", "messageArg3_41", "messageArg3_42", "messageArg3_43", "messageArg3_44", "messageArg3_45", "messageArg3_46", "messageArg3_47", "messageArg4_0", "messageArg4_1", "messageArg4_2", "messageArg4_3", "messageArg4_4", "messageArg4_5", "messageArg4_6", "messageArg4_7", "messageArg4_8", "messageArg4_9", "messageArg4_10", "messageArg4_11", "messageArg4_12", "messageArg4_13", "messageArg4_14", "messageArg4_15", "messageArg4_16", "messageArg4_17", "messageArg4_18", "messageArg4_19", "messageArg4_20", "messageArg4_21", "messageArg4_22", "messageArg4_23", "messageArg4_24", "messageArg4_25", "messageArg4_26", "messageArg4_27", "messageArg4_28", "messageArg4_29", "messageArg4_30", "messageArg4_31", "messageArg4_32", "messageArg4_33", "messageArg4_34", "messageArg4_35", "messageArg4_36", "messageArg4_37", "messageArg4_38", "messageArg4_39", "messageArg4_40", "messageArg4_41", "messageArg4_42", "messageArg4_43", "messageArg4_44", "messageArg4_45", "messageArg4_46", "messageArg4_47", "messageArg5_0", "messageArg5_1", "messageArg5_2", "messageArg5_3", "messageArg5_4", "messageArg5_5", "messageArg5_6", "messageArg5_7", "messageArg5_8", "messageArg5_9", "messageArg5_10", "messageArg5_11", "messageArg5_12", "messageArg5_13", "messageArg5_14", "messageArg5_15", "messageArg5_16", "messageArg5_17", "messageArg5_18", "messageArg5_19", "messageArg5_20", "messageArg5_21", "messageArg5_22", "messageArg5_23", "messageArg5_24", "messageArg5_25", "messageArg5_26", "messageArg5_27", "messageArg5_28", "messageArg5_29", "messageArg5_30", "messageArg5_31", "messageArg5_32", "messageArg5_33", "messageArg5_34", "messageArg5_35", "messageArg5_36", "messageArg5_37", "messageArg5_38", "messageArg5_39", "messageArg5_40", "messageArg5_41", "messageArg5_42", "messageArg5_43", "messageArg5_44", "messageArg5_45", "messageArg5_46", "messageArg5_47", "portraitCalls", "portraitArg1_0", "portraitArg1_1", "portraitArg1_2", "portraitArg1_3", "portraitArg1_4", "portraitArg1_5", "portraitArg1_6", "portraitArg1_7", "portraitArg1_8", "portraitArg1_9", "portraitArg1_10", "portraitArg1_11", "portraitArg1_12", "portraitArg1_13", "portraitArg1_14", "portraitArg1_15", "portraitArg1_16", "portraitArg1_17", "portraitArg1_18", "portraitArg1_19", "portraitArg1_20", "portraitArg1_21", "portraitArg1_22", "portraitArg1_23", "portraitArg1_24", "portraitArg1_25", "portraitArg1_26", "portraitArg1_27", "portraitArg1_28", "portraitArg1_29", "portraitArg1_30", "portraitArg1_31", "portraitArg3_0", "portraitArg3_1", "portraitArg3_2", "portraitArg3_3", "portraitArg3_4", "portraitArg3_5", "portraitArg3_6", "portraitArg3_7", "portraitArg3_8", "portraitArg3_9", "portraitArg3_10", "portraitArg3_11", "portraitArg3_12", "portraitArg3_13", "portraitArg3_14", "portraitArg3_15", "portraitArg3_16", "portraitArg3_17", "portraitArg3_18", "portraitArg3_19", "portraitArg3_20", "portraitArg3_21", "portraitArg3_22", "portraitArg3_23", "portraitArg3_24", "portraitArg3_25", "portraitArg3_26", "portraitArg3_27", "portraitArg3_28", "portraitArg3_29", "portraitArg3_30", "portraitArg3_31", "monsterMessageCalls", "monsterMessageArg1_0", "monsterMessageArg1_1", "monsterMessageArg1_2", "monsterMessageArg1_3", "monsterMessageArg1_4", "monsterMessageArg1_5", "monsterMessageArg1_6", "monsterMessageArg1_7", "monsterMessageArg1_8", "monsterMessageArg2_0", "monsterMessageArg2_1", "monsterMessageArg2_2", "monsterMessageArg2_3", "monsterMessageArg2_4", "monsterMessageArg2_5", "monsterMessageArg2_6", "monsterMessageArg2_7", "monsterMessageArg2_8", "monsterMessageArg3_0", "monsterMessageArg3_1", "monsterMessageArg3_2", "monsterMessageArg3_3", "monsterMessageArg3_4", "monsterMessageArg3_5", "monsterMessageArg3_6", "monsterMessageArg3_7", "monsterMessageArg3_8", "hitSoundCalls", "hitSoundArg1_0", "hitSoundArg1_1", "hitSoundArg1_2", "hitSoundArg1_3", "hitSoundArg1_4", "hitSoundArg1_5", "hitSoundArg1_6", "hitSoundArg1_7", "hitSoundArg1_8", "shieldVisualCalls", "shieldVisualArg1_0", "shieldVisualArg1_1", "shieldVisualArg1_2", "shieldVisualArg1_3", "shieldVisualArg1_4", "shieldVisualArg1_5", "shieldVisualArg1_6", "shieldVisualArg1_7", "shieldVisualArg1_8", "shieldVisualArg2_0", "shieldVisualArg2_1", "shieldVisualArg2_2", "shieldVisualArg2_3", "shieldVisualArg2_4", "shieldVisualArg2_5", "shieldVisualArg2_6", "shieldVisualArg2_7", "shieldVisualArg2_8", "deathAnimationCalls", "deathAnimationArg1_0", "deathAnimationArg1_1", "deathAnimationArg2_0", "deathAnimationArg2_1", "deathNotifyCalls", "deathNotifyArg1_0", "deathNotifyArg1_1", "nameLogCalls", "nameLogArg1_0", "nameLogArg1_1", "nameLogArg1_2", "nameLogArg1_3", "nameLogArg1_4", "nameLogArg1_5", "nameLogArg1_6", "nameLogArg1_7", "nameLogArg1_8", "nameLogArg3_0", "nameLogArg3_1", "nameLogArg3_2", "nameLogArg3_3", "nameLogArg3_4", "nameLogArg3_5", "nameLogArg3_6", "nameLogArg3_7", "nameLogArg3_8", "hp0", "shield0", "shieldDuration0", "shieldPower0", "shieldWord0", "stamina0", "fatigue0", "mana0", "rebuild470", "rebuild4f0", "rebuild2470", "rebuild24b0", "lastSource0_0", "lastSource0_1", "lastSource0_2", "lastSource0_3", "lastSource0_4", "lastSource0_5", "lastSource0_6", "lastSource0_7", "lastSource0_8", "lastSource0_9", "lastSource0_10", "lastSource0_11", "lastSource0_12", "modifiers0_0", "modifiers0_1", "modifiers0_2", "modifiers0_3", "modifiers0_4", "modifiers0_5", "modifiers0_6", "modifiers0_7", "modifiers0_8", "modifiers0_9", "modifiers0_10", "modifiers0_11", "modifiers0_12", "modifiers0_13", "modifiers0_14", "modifiers0_15", "modifiers0_16", "modifiers0_17", "modifiers0_18", "modifiers0_19", "modifiers0_20", "modifiers0_21", "modifiers0_22", "modifiers0_23", "modifiers0_24", "modifiers0_25", "hp1", "shield1", "shieldDuration1", "shieldPower1", "shieldWord1", "stamina1", "fatigue1", "mana1", "rebuild471", "rebuild4f1", "rebuild2471", "rebuild24b1", "lastSource1_0", "lastSource1_1", "lastSource1_2", "lastSource1_3", "lastSource1_4", "lastSource1_5", "lastSource1_6", "lastSource1_7", "lastSource1_8", "lastSource1_9", "lastSource1_10", "lastSource1_11", "lastSource1_12", "modifiers1_0", "modifiers1_1", "modifiers1_2", "modifiers1_3", "modifiers1_4", "modifiers1_5", "modifiers1_6", "modifiers1_7", "modifiers1_8", "modifiers1_9", "modifiers1_10", "modifiers1_11", "modifiers1_12", "modifiers1_13", "modifiers1_14", "modifiers1_15", "modifiers1_16", "modifiers1_17", "modifiers1_18", "modifiers1_19", "modifiers1_20", "modifiers1_21", "modifiers1_22", "modifiers1_23", "modifiers1_24", "modifiers1_25", "aliasText0", "aliasText1", "aliasText2", "aliasText3", "aliasText4", "aliasText5", "aliasText6", "aliasText7"}

var ostStubs = map[uint32]*mcaStub{
	0x5520d0: {ret: 0x20030000, retN: 64, rec: [][4]uint32{{1, 4, 0x20030108, 64}, {2, 4, 0x20030208, 64}, {3, 4, 0x20030308, 64}, {4, 4, 0x20030408, 64}}, cnt: 0x20030104},
	0x52a890: {ret: 0x20030100, retN: 0, rec: [][4]uint32{{1, 4, 0x2003050c, 72}, {2, 4, 0x2003062c, 72}, {4, 4, 0x2003074c, 72}}, cnt: 0x20030508},
	0x52b6d0: {ret: 0x20030100, retN: 0, rec: [][4]uint32{{1, 4, 0x20030870, 48}, {2, 4, 0x20030930, 48}, {3, 4, 0x200309f0, 48}}, cnt: 0x2003086c},
	0x52af50: {ret: 0x20030100, retN: 0, rec: [][4]uint32{{1, 4, 0x20030ab4, 8}, {2, 4, 0x20030ad4, 8}}, cnt: 0x20030ab0},
	0x522fe0: {ret: 0x20030100, retN: 0, rec: [][4]uint32{{1, 4, 0x20030af8, 64}, {2, 4, 0x20030bf8, 64}, {3, 4, 0x20030cf8, 64}, {4, 4, 0x20030df8, 64}}, cnt: 0x20030af4},
	0x523c00: {ret: 0x20030100, retN: 0, rec: [][4]uint32{{1, 4, 0x20030efc, 40}, {2, 4, 0x20030f9c, 40}, {3, 4, 0x2003103c, 40}, {4, 4, 0x200310dc, 40}, {5, 4, 0x2003117c, 40}, {6, 1, 0x2003121c, 40}}, cnt: 0x20030ef8},
	0x52beb0: {ret: 0x20030100, retN: 0, rec: [][4]uint32{{1, 4, 0x20031248, 9}, {3, 4, 0x2003126c, 9}}, cnt: 0x20031244},
	0x4e4280: {ret: 0x20030100, retN: 0, rec: [][4]uint32{{1, 4, 0x20031294, 2}}, cnt: 0x20031290},
	0x4e46f0: {ret: 0x20030100, retN: 0, rec: [][4]uint32{{1, 4, 0x200312a0, 2}, {2, 1, 0x200312a8, 2}}, cnt: 0x2003129c},
	0x58aad0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200312ae, 48}, {2, 4, 0x2003136e, 48}, {3, 4, 0x2003142e, 48}, {4, 4, 0x200314ee, 48}, {5, 4, 0x200315ae, 48}}, cnt: 0x200312aa},
	0x58b410: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20031672, 32}, {3, 4, 0x200316f2, 32}}, cnt: 0x2003166e},
	0x590b40: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20031776, 9}, {2, 4, 0x2003179a, 9}, {3, 4, 0x200317be, 9}}, cnt: 0x20031772},
	0x48f650: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200317e6, 9}}, cnt: 0x200317e2},
	0x4acd80: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2003180e, 9}, {2, 4, 0x20031832, 9}}, cnt: 0x2003180a},
	0x4e4db0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2003185a, 2}, {2, 4, 0x20031862, 2}}, cnt: 0x20031856},
	0x4e3af0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2003186e, 2}}, cnt: 0x2003186a},
	0x5182e0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2003187a, 9}, {3, 4, 0x2003189e, 9}}, cnt: 0x20031876},
}
