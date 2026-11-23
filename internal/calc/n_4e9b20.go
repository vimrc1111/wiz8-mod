package calc

// 0x4e9b20 round_stamina_recovery: the once-per-round upkeep. It refreshes the
// party action queue (0x4f0560), advances the clock by one round (0x482a20),
// counts idle rounds while no group engages, plays a party voice (0x52f240 /
// 0x52f1d0), regenerates stamina of resting / sleeping characters and
// monsters (2 dice of maxStamina/20 - 1 sides, 0x517950 / 0x517970), marks
// downed characters, counts down monster team timers, and finally turns
// monsters toward their target.
//
// Address stubs. D4 formulas (Emu replays the case; F runs the Go node where
// the caller reads its effect): 0x4eee80 (xpEligibleCountNode), 0x4e79a0
// (pcActionEligibilityNode), 0x52b1c0 (characterStaminaHealNode), 0x4e99c0
// (combatSkillGrowthNode). 0x5545f0, 0x502010, 0x5044d0, 0x4ee9d0, 0x535200,
// 0x4e4ab0, 0x53bf80, 0x536400 and 0x5049c0 are D4 too; their return is
// ignored and this node reads none of their writes, so F records the call
// only. 排除 stubs record their arguments (0x531920 and 0x51ac30 return /
// write the case's inputs). The dice 0x517970 runs natively (diceNode).
// 0x5479b0's x87 arithmetic runs at extended precision (mtr helpers).
//
// Fixture: the lighting objects [0x65a160] / [0x65ad84] are null (their
// 0x4b8850 calls are presentation); calc panics when they are not.

func init() {
	Nodes["004e9b20"] = Node{Addr: "004e9b20", In: rsrIn, Out: rsrOut, F: roundStaminaRecoveryNode, Emu: rsrEmu}
}

func roundStaminaRecoveryNode(in []int64, r Rand) []int64 { return rsrRun(true)(in, r) }

func rsrEmu(in []int64, r Rand) []int64 { return rsrRun(false)(in, r) }

func rsrRun(real bool) func([]int64, Rand) []int64 {
	return mcaRun(&rsrSpec, real, func(w *mcaWorld) {
		w.tab = rsrStubs
		rsrBody(w)
	})
}

const (
	rsrRoundTicks = 0x1d4c0    // 0x4e9b20 pushes 120000 ms to 0x482a20
	rsrDayTicks   = 0x5265c00  // 0x482a49: 86400000 ms per day
	rsrHaste      = 0x40533333 // 0x4e9d33: f32 3.3, 0x5479b0's multiplier
	rsrLevelBase  = 0x5ee800   // 0x5479e0: f32 added to 2*level
	rsrLevelScale = 0x5ed490   // 0x5479ea: f32 final factor
	rsrClassTab   = 0x61507c   // 0x547940: 3 dwords per profession
	rsrRaceTab    = 0x615130   // 0x547940: 5 dwords per race
	rsrClasses    = 15         // professions 0..14 (the race table follows at 0x61507c+15*12)
	rsrRaces      = 11         // races 0..10
	rsrVoiceOne   = 0x5ee5e0   // 0x52f240: voice id dword
	rsrVoiceAll   = 0x5ee5dc   // 0x52f1d0: voice id dword
	rsrVoiceA     = 0x5ed8cc   // 0x52f1d0 / 0x52f240: fourth 0x52e690 argument
	rsrVoiceB     = 0x5ed914   // fifth 0x52e690 argument
	rsrFacing     = 0x7e000100 // 0x51ac30's output record (a stack local; argument not recorded)
)

func rsrS(w *mcaWorld, i uint32) uint32 { return w.u32(0x685178) + 0x106*i }

func rsrBody(w *mcaWorld) {
	cb := w.u32(0x6836a8)
	w.put(cb+0xa62, 1, 0)
	if w.u8(0x683fce) != 0 {
		w.stub(0x4f0630)
	}
	if w.u8(0x6850d2) == 0 && w.u32(cb+0x90c) != 0 {
		rsrQueue(w, cb)
	}
	w.stub(0x5545f0, 0x683feb, 0x687511)
	if w.u32(0x683fa5) != 0 {
		w.stub(0x58aad0, 0xc, w.u32(w.u32(0x68c09c)+0x8ac), w.u32(cb+4))
		w.stub(0x408860, 0x617480, 0)
	}
	w.put(cb+0xa52, 1, 0)
	w.put(cb+0xa53, 1, 0)
	switch {
	case w.u8(cb+0xa55) == 0:
		w.put(cb+0xa56, 1, 0)
	case !rsrEngaged(w):
		w.put(cb+0xa56, 1, uint64(w.u8(cb+0xa56)+1))
	}
	rsrClock(w)
	if w.u8(0x652da4) != 0 {
		w.stub(0x5044d0, 10)
	}
	if ui := w.u32(0x68edcc); w.u32(0x68ec78) == 7 && ui != 0 { // 0x5699b0
		w.put(ui+0xf4, 4, uint64(w.u32(ui+0xf4)|0x100))
	}
	w.put(cb, 1, 0)
	w.put(cb+1, 1, 1)
	if w.u32(cb+0x908) != 0 {
		switch rsrXpCount(w) {
		case 0:
		case 1:
			rsrVoiceLast(w)
		default:
			if w.u32(cb+0x908) > 1 {
				rsrVoiceRandom(w)
			}
		}
	}
	for i := uint32(0); i < 8; i++ {
		rsrChar(w, cb, i)
	}
	mons := w.u32(0x683fad)
	for i := uint32(0); i < w.listCount(mons); i++ {
		m := w.monsterRecord(i)
		if w.u8(m+0x15) == 0 {
			continue
		}
		if p := w.u32(m + 0x107); w.u32(m+0x2e1) == 1 || p == 0xf || p == 0x11 {
			mtrSp(w, m, rsrDice(w, w.u32(m+0x2f))) // 0x52c140(m, v, 0)
		}
		if tm := w.u32(m + 0x10); w.u8(tm+0x146) != 0 && w.u32(m+0x2e1) != 3 {
			w.put(tm+0x146, 1, uint64(w.u8(tm+0x146)-1))
		}
	}
	w.stub(0x562a50, 0x1000ff)
	w.stub(0x534300)
	w.stub(0x535200)
	w.stub(0x5354e0)
	w.stub(0x4e4ab0)
	for _, mode := range [2]uint32{1, 0} { // 0x505780, 0x5057d0
		for i := uint32(0); i < w.listCount(mons); i++ {
			w.stub(0x5049c0, w.monsterRecord(i), mode, 0)
		}
		if mode == 1 && w.u8(0x683f94) != 0 {
			w.stub(0x593330)
			w.stub(0x53bf80)
		}
	}
	w.stub(0x536400)
	for i := uint32(0); i < w.listCount(mons); i++ {
		m := w.monsterRecord(i)
		if w.u8(m+0x15) == 0 || w.u32(m+0x2b) == 0 || w.u32(m+0x107) >= 0xe || w.u32(m+0x87) != 0 {
			continue
		}
		w.stub(0x51ac30, m, rsrFacing, 3)
		switch w.u32(rsrFacing) {
		case 2:
			w.stub(0x4c6240, w.u32(m+0xc), 0)
		case 3:
			w.stub(0x4c62c0, w.u32(m+0xc), rsrActor(w, w.u32(rsrFacing+8)), 0)
		}
	}
}

// rsrQueue is 0x4f0560(-1).
func rsrQueue(w *mcaWorld, cb uint32) {
	if w.u8(0x683f94) == 0 {
		panic("round stamina recovery: 0x4f0560 asserts while messages are off (0x4f0569)")
	}
	w.put(cb+0x90c, 4, 0)
	for i := uint32(0); i < 8; i++ {
		if byte(rsrEligible(w, i)) != 0 {
			w.stub(0x53a930, i, rsrS(w, i)+0x4d)
		}
	}
	if (w.u32(cb+0x910) != 0 && w.u32(cb+0x918) != 3) || w.u32(cb+0x90c) != 0 {
		ui := w.u32(0x68edcc)
		w.put(ui+0x2dc, 4, 100)
		w.put(ui+0x2e0, 4, 100)
		return // 0x5a1dd0: the view object's vtable slot 1 (fixture: a bare ret 4)
	}
	w.stub(0x5a1890)
}

// rsrEligible is 0x4e79a0(i, 1, 0, 0).
func rsrEligible(w *mcaWorld, x uint32) uint32 {
	k := w.call(0x4e79a0, x)
	if !w.real {
		return w.replay(0x4e79a0, k, x)
	}
	v := append([]int64(nil), lsPark4e79a0[:]...)
	v[0], v[1] = int64(x), 1
	for j := uint32(0); j < 8; j++ {
		c := w.charBase(j)
		v[30+3*j], v[31+3*j], v[32+3*j] = int64(w.u8(rsrS(w, j))), int64(w.u32(c+0xb11)), int64(w.u32(c+0xb01))
	}
	c := w.charBase(x)
	v[54], v[55] = int64(int32(w.u32(c+0x69))), int64(int32(w.u32(c+0x71)))
	v[58], v[59] = int64(w.u32(c+0xb19)), int64(int32(w.u32(c+0xb1d))) // widths: u32, i32
	return uint32(pcActionEligibilityNode(v, w.r)[0])
}

// rsrEngaged is 0x4e9b20's group scan: a group that is ready, active and
// engaging (0x531920) stops the idle-round count.
func rsrEngaged(w *mcaWorld) bool {
	l := w.u32(0x683fb1)
	for i := uint32(0); i < w.listCount(l); i++ {
		g := w.listAt(l, i) // 0x5101b0
		if g == 0 {
			panic("round stamina recovery: null group record")
		}
		if w.u8(g+0x28) != 0 && w.u8(g+0x29) != 0 && byte(w.stub(0x531920, g)) != 0 {
			return true
		}
	}
	return false
}

// rsrClock is 0x482a20(120000).
func rsrClock(w *mcaWorld) {
	t := w.u32(0x6874f7) + rsrRoundTicks
	if t >= rsrDayTicks {
		w.put(0x687595, 4, uint64(w.u32(0x687595)+1))
	}
	w.put(0x6874f7, 4, uint64(t%rsrDayTicks))
	if w.u32(0x68ec78) == 7 {
		w.stub(0x502010, rsrRoundTicks)
	}
	w.put(0x65b9a8, 4, rsrTick) // GetTickCount
	if w.u32(0x65a160) != 0 || w.u32(0x65ad84) != 0 {
		panic("round stamina recovery: lighting objects are outside the fixture")
	}
	sky := uint32(uint64((t%rsrDayTicks)/1000<<8) * 0xc22e4507 >> 48)
	for p := uint32(0x65a168); p < 0x65a174; p += 4 {
		if o := w.u32(p); o != 0 {
			w.put(o+0x58, 4, uint64(sky))
		}
	}
}

// rsrXpCount is 0x4eee80.
func rsrXpCount(w *mcaWorld) uint32 {
	k := w.call(0x4eee80)
	if !w.real {
		return w.replay(0x4eee80, k)
	}
	v := make([]int64, 0, 24)
	for j := uint32(0); j < 8; j++ {
		c := w.charBase(j)
		v = append(v, int64(w.u8(rsrS(w, j))), int64(int32(w.u32(c+0xb11))), int64(w.u32(c+0xb01)))
	}
	return uint32(xpEligibleCountNode(v, w.r)[0])
}

// rsrVoiceLast is 0x52f240: the last present, unmuted character speaks.
func rsrVoiceLast(w *mcaWorld) {
	who := -1
	for i := uint32(0); i < 8; i++ {
		if w.u8(rsrS(w, i)) != 0 && w.u32(w.charBase(i)+0xa49) == 0 {
			who = int(i)
		}
	}
	if who >= 0 {
		w.stub(0x52e690, w.charBase(uint32(who)), w.u32(rsrVoiceOne), 0, w.u32(rsrVoiceA), w.u32(rsrVoiceB))
	}
}

// rsrVoiceRandom is 0x52f1d0: 0x517fb0(0, 0, -1, &buf, 1, 0) picks one
// present, living, standing character at random.
func rsrVoiceRandom(w *mcaWorld) {
	var list []uint32
	for i := uint32(0); i < 8; i++ {
		c := w.charBase(i)
		if w.u8(rsrS(w, i)) != 0 && w.u32(c+0xb11) != 0 && w.u32(c+0xb01) < 0x12 {
			list = append(list, i)
		}
	}
	if len(list) == 0 {
		return
	}
	who := list[0]
	if len(list) > 1 {
		who = list[w.r.R(len(list))]
	}
	w.stub(0x52e690, w.charBase(who), w.u32(rsrVoiceAll), 0, w.u32(rsrVoiceA), w.u32(rsrVoiceB))
}

// rsrDice is 0x517950(buf, 2, t, 0) then 0x517970: 2 dice of byte(t) sides,
// t = max/20 - 1 raised to 1 (max/20 == 0 wraps to 0xffffffff).
func rsrDice(w *mcaWorld, max uint32) uint32 {
	t := max/20 - 1
	if t < 1 {
		t = 1
	}
	return uint32(diceNode([]int64{0, 2, int64(byte(t))}, w.r)[0])
}

func rsrChar(w *mcaWorld, cb, i uint32) {
	s, c := rsrS(w, i), w.charBase(i)
	if w.u8(s) == 0 {
		return
	}
	if st := w.u32(c + 0xb01); w.u32(s+1) == 4 || st == 0xf || st == 0x11 {
		v := rsrDice(w, w.u32(c+0xb19))
		if rsrSlowed(w, c) {
			x := mtrMul(rsrLevelFactor(w, c), mtrF64(float64(int32(v))))
			v = uint32(mtrFtol(x))
		}
		low := uint32(0)
		if int32(w.u32(c+0xb1d)) < int32(w.u32(c+0xb19)) {
			low = 1
		}
		rsrHeal(w, i, v, w.u32(0x685174)&0xffffff00|low)
	}
	rsrGrowth(w, i)
	down := w.u32(c+0xb11) == 0 || w.u32(c+0xb01) >= 0xf
	w.put(cb+0x4c+0xd4*i, 1, uint64(mstB(down)))
	w.stub(0x4ee9d0, c)
}

// rsrSlowed is 0x547940(C, 0): a zero dword in the profession's or race's row.
func rsrSlowed(w *mcaWorld, c uint32) bool {
	if cls := int32(w.u32(c + 0x69)); cls != -1 {
		if cls < 0 || cls >= rsrClasses {
			panic("round stamina recovery: profession outside 0..14")
		}
		for k := uint32(0); k < 3; k++ {
			if w.u32(rsrClassTab+12*uint32(cls)+4*k) == 0 {
				return true
			}
		}
	}
	if race := int32(w.u32(c + 0x71)); race != -1 {
		if race < 0 || race >= rsrRaces {
			panic("round stamina recovery: race outside 0..10")
		}
		for k := uint32(0); k < 5; k++ {
			if w.u32(rsrRaceTab+20*uint32(race)+4*k) == 0 {
				return true
			}
		}
	}
	return false
}

// rsrLevelFactor is 0x5479b0(C, 0, 3.3): the profession's level slot (+0x8d +
// 4*profession, so profession -1 reads +0x89) above 20 gives 3.3, else
// (2*level + [0x5ee800]) * 3.3 * [0x5ed490].
func rsrLevelFactor(w *mcaWorld, c uint32) mtrX {
	lv := w.u32(c + 0x8d + 4*w.u32(c+0x69))
	if lv > 20 {
		return mtrF32(rsrHaste)
	}
	x := mtrAdd(mtrF64(float64(2*lv)), mtrF32(w.u32(rsrLevelBase)))
	return mtrMul(mtrMul(x, mtrF32(rsrHaste)), mtrF32(w.u32(rsrLevelScale)))
}

// rsrHeal is 0x52b1c0(i, v, notify).
func rsrHeal(w *mcaWorld, i, v, notify uint32) {
	k := w.call(0x52b1c0, i, v, notify)
	if !w.real {
		w.replay(0x52b1c0, k, i, v, notify)
		return
	}
	c := w.charBase(i)
	o := characterStaminaHealNode([]int64{int64(w.u32(0x685174)), int64(i), int64(v), int64(notify),
		int64(w.u32(c + 0xb01)), int64(w.u32(c + 0xb11)), int64(w.u32(c + 0xb19)), int64(w.u32(c + 0xb1d)),
		int64(w.u32(c + 0x169e)), int64(w.u32(c + 0xa45)), 0, 0}, w.r)
	w.put(c+0xb1d, 4, uint64(o[0]))
	w.put(c+0x169e, 4, uint64(o[1]))
}

// rsrGrowth is 0x4e99c0(i).
func rsrGrowth(w *mcaWorld, i uint32) {
	k := w.call(0x4e99c0, i)
	if !w.real {
		w.replay(0x4e99c0, k, i)
		return
	}
	d := w.u32(0x6836a8) + 0xd4*i
	v := []int64{int64(i), int64(w.u32(0x685174))}
	for h := uint32(0); h < 2; h++ {
		b := d + 0x60 + 0x10*h
		v = append(v, int64(w.u32(b)), int64(w.u32(b+4)), int64(w.u32(b+8)), int64(w.u8(b+0xc)))
	}
	for f := uint32(0); f < csgFlags; f++ {
		v = append(v, int64(w.u8(d+0xc0+f)))
	}
	o := combatSkillGrowthNode(v, w.r)
	w.put(d+0x60, 4, uint64(o[0]))
	w.put(d+0x70, 4, uint64(o[1]))
	for f := uint32(0); f < csgFlags; f++ {
		w.put(d+0xc0+f, 1, uint64(o[2+f]))
	}
}

// rsrActor is 0x4e5950(id): the monster's actor (+0xc), 0 for id -1 or an
// unknown id.
func rsrActor(w *mcaWorld, id uint32) uint32 {
	if id == 0xffffffff {
		return 0
	}
	h, ok := w.monsterHandle(id)
	if !ok {
		return 0
	}
	return w.u32(w.monsterRecord(h) + 0xc)
}

// Code below is generated from the same description as re/emu/cases/004e9b20.json.

const rsrTick = 0x12345678 // the case's GetTickCount value

var rsrSpec = mcaStrings(mstSpec{
	set: []mstSet{
		{0x68c09c, 4, 0x20003000},
		{0x6836a8, 4, 0x20006000},
		{0x685178, 4, 0x20005000},
		{0x685174, 4, 0x20010000},
		{0x683fad, 4, 0x2000b300},
		{0x683fb1, 4, 0x2000b340},
		{0x683fb9, 4, 0x0},
		{0x683fbd, 4, 0x2000b3a0},
		{0x6840c7, 4, 0x20008000},
		{0x6840cb, 4, 0x20008400},
		{0x68edcc, 4, 0x2000d000},
		{0x69bf4c, 4, 0x2000c500},
		{0x2000c500, 4, 0x2000c520},
		{0x2000c524, 4, 0x5aa400},
		{0x65a160, 4, 0x0},
		{0x65ad84, 4, 0x0},
		{0x65a168, 4, 0x2000c600},
		{0x65a16c, 4, 0x0},
		{0x65a170, 4, 0x2000c700},
		{0x2000b300, 4, 0x2000b320},
		{0x2000b340, 4, 0x2000b360},
		{0x2000b3a0, 4, 0x2000b360},
		{0x2000b3a8, 4, 0x2},
		{0x2000b360, 4, 0x2000b400},
		{0x2000b400, 4, 0x1c8},
		{0x2000b408, 4, 0x2000b380},
		{0x2000b42c, 1, 0x1},
		{0x2000b320, 4, 0x20009000},
		{0x20009000, 4, 0x64},
		{0x2000900c, 4, 0x20009800},
		{0x20009010, 4, 0x2000c000},
		{0x200099d8, 4, 0x2000a000},
		{0x20008187, 2, 0x18c},
		{0x2000817c, 1, 0x7f},
		{0x200080ce, 1, 0x7e},
		{0x20009027, 4, 0x5a},
		{0x2000b364, 4, 0x2000b500},
		{0x2000b500, 4, 0x1c9},
		{0x2000b508, 4, 0x2000b380},
		{0x2000b52c, 1, 0x1},
		{0x2000b324, 4, 0x20009400},
		{0x20009400, 4, 0x65},
		{0x2000940c, 4, 0x20009c00},
		{0x20009410, 4, 0x2000c200},
		{0x20009dd8, 4, 0x2000a800},
		{0x20008587, 2, 0x18c},
		{0x2000857c, 1, 0x7e},
		{0x200084ce, 1, 0x7d},
		{0x20009427, 4, 0x5b},
		{0x20010089, 4, 0x60},
		{0x2001008d, 4, 0x61},
		{0x20010091, 4, 0x62},
		{0x20010095, 4, 0x63},
		{0x20010099, 4, 0x64},
		{0x2001009d, 4, 0x65},
		{0x200100a1, 4, 0x66},
		{0x200100a5, 4, 0x67},
		{0x200100a9, 4, 0x68},
		{0x200100ad, 4, 0x69},
		{0x200100b1, 4, 0x6a},
		{0x200100b5, 4, 0x6b},
		{0x200100b9, 4, 0x6c},
		{0x200100bd, 4, 0x6d},
		{0x200100c1, 4, 0x6e},
		{0x200100c5, 4, 0x6f},
		{0x200118eb, 4, 0x70},
		{0x200118ef, 4, 0x71},
		{0x200118f3, 4, 0x72},
		{0x200118f7, 4, 0x73},
		{0x200118fb, 4, 0x74},
		{0x200118ff, 4, 0x75},
		{0x20011903, 4, 0x76},
		{0x20011907, 4, 0x77},
		{0x2001190b, 4, 0x78},
		{0x2001190f, 4, 0x79},
		{0x20011913, 4, 0x7a},
		{0x20011917, 4, 0x7b},
		{0x2001191b, 4, 0x7c},
		{0x2001191f, 4, 0x7d},
		{0x20011923, 4, 0x7e},
		{0x20011927, 4, 0x7f},
		{0x2001314d, 4, 0x80},
		{0x20013151, 4, 0x81},
		{0x20013155, 4, 0x82},
		{0x20013159, 4, 0x83},
		{0x2001315d, 4, 0x84},
		{0x20013161, 4, 0x85},
		{0x20013165, 4, 0x86},
		{0x20013169, 4, 0x87},
		{0x2001316d, 4, 0x88},
		{0x20013171, 4, 0x89},
		{0x20013175, 4, 0x8a},
		{0x20013179, 4, 0x8b},
		{0x2001317d, 4, 0x8c},
		{0x20013181, 4, 0x8d},
		{0x20013185, 4, 0x8e},
		{0x20013189, 4, 0x8f},
		{0x200149af, 4, 0x90},
		{0x200149b3, 4, 0x91},
		{0x200149b7, 4, 0x92},
		{0x200149bb, 4, 0x93},
		{0x200149bf, 4, 0x94},
		{0x200149c3, 4, 0x95},
		{0x200149c7, 4, 0x96},
		{0x200149cb, 4, 0x97},
		{0x200149cf, 4, 0x98},
		{0x200149d3, 4, 0x99},
		{0x200149d7, 4, 0x9a},
		{0x200149db, 4, 0x9b},
		{0x200149df, 4, 0x9c},
		{0x200149e3, 4, 0x9d},
		{0x200149e7, 4, 0x9e},
		{0x200149eb, 4, 0x9f},
		{0x20016211, 4, 0xa0},
		{0x20016215, 4, 0xa1},
		{0x20016219, 4, 0xa2},
		{0x2001621d, 4, 0xa3},
		{0x20016221, 4, 0xa4},
		{0x20016225, 4, 0xa5},
		{0x20016229, 4, 0xa6},
		{0x2001622d, 4, 0xa7},
		{0x20016231, 4, 0xa8},
		{0x20016235, 4, 0xa9},
		{0x20016239, 4, 0xaa},
		{0x2001623d, 4, 0xab},
		{0x20016241, 4, 0xac},
		{0x20016245, 4, 0xad},
		{0x20016249, 4, 0xae},
		{0x2001624d, 4, 0xaf},
		{0x20017a73, 4, 0xb0},
		{0x20017a77, 4, 0xb1},
		{0x20017a7b, 4, 0xb2},
		{0x20017a7f, 4, 0xb3},
		{0x20017a83, 4, 0xb4},
		{0x20017a87, 4, 0xb5},
		{0x20017a8b, 4, 0xb6},
		{0x20017a8f, 4, 0xb7},
		{0x20017a93, 4, 0xb8},
		{0x20017a97, 4, 0xb9},
		{0x20017a9b, 4, 0xba},
		{0x20017a9f, 4, 0xbb},
		{0x20017aa3, 4, 0xbc},
		{0x20017aa7, 4, 0xbd},
		{0x20017aab, 4, 0xbe},
		{0x20017aaf, 4, 0xbf},
		{0x200192d5, 4, 0xc0},
		{0x200192d9, 4, 0xc1},
		{0x200192dd, 4, 0xc2},
		{0x200192e1, 4, 0xc3},
		{0x200192e5, 4, 0xc4},
		{0x200192e9, 4, 0xc5},
		{0x200192ed, 4, 0xc6},
		{0x200192f1, 4, 0xc7},
		{0x200192f5, 4, 0xc8},
		{0x200192f9, 4, 0xc9},
		{0x200192fd, 4, 0xca},
		{0x20019301, 4, 0xcb},
		{0x20019305, 4, 0xcc},
		{0x20019309, 4, 0xcd},
		{0x2001930d, 4, 0xce},
		{0x20019311, 4, 0xcf},
		{0x2001ab37, 4, 0xd0},
		{0x2001ab3b, 4, 0xd1},
		{0x2001ab3f, 4, 0xd2},
		{0x2001ab43, 4, 0xd3},
		{0x2001ab47, 4, 0xd4},
		{0x2001ab4b, 4, 0xd5},
		{0x2001ab4f, 4, 0xd6},
		{0x2001ab53, 4, 0xd7},
		{0x2001ab57, 4, 0xd8},
		{0x2001ab5b, 4, 0xd9},
		{0x2001ab5f, 4, 0xda},
		{0x2001ab63, 4, 0xdb},
		{0x2001ab67, 4, 0xdc},
		{0x2001ab6b, 4, 0xdd},
		{0x2001ab6f, 4, 0xde},
		{0x2001ab73, 4, 0xdf},
	},
	in: []mstFld{
		{0x683fce, 1, false, -1, 0, 0},
		{0x6850d2, 1, false, -1, 0, 0},
		{0x2000690c, 4, false, -1, 0, 0},
		{0x683f94, 1, false, -1, 0, 0},
		{0x683fa5, 4, false, -1, 0, 0},
		{0x20006004, 4, false, -1, 0, 0},
		{0x20006a55, 1, false, -1, 0, 0},
		{0x20006a56, 1, false, -1, 0, 0},
		{0x2000b348, 4, false, -1, 0, 0},
		{0x2000b428, 1, false, -1, 0, 0},
		{0x2000b429, 1, false, -1, 0, 0},
		{0x2000b414, 4, false, -1, 0, 0},
		{0x2000b528, 1, false, -1, 0, 0},
		{0x2000b529, 1, false, -1, 0, 0},
		{0x2000b514, 4, false, -1, 0, 0},
		{0x652da4, 1, false, -1, 0, 0},
		{0x68ec78, 4, false, -1, 0, 0},
		{0x6874f7, 4, false, -1, 0, 0},
		{0x687595, 4, false, -1, 0, 0},
		{0x2000d0f4, 4, false, -1, 0, 0},
		{0x20006908, 4, false, -1, 0, 0},
		{0x20006910, 4, false, -1, 0, 0},
		{0x20006918, 4, false, -1, 0, 0},
		{0x20005000, 1, false, -1, 0, 0},
		{0x20005001, 4, false, -1, 0, 0},
		{0x20010b01, 4, false, -1, 0, 0},
		{0x20010b11, 4, true, -1, 0, 0},
		{0x20010b19, 4, false, -1, 0, 0},
		{0x20010b1d, 4, false, -1, 0, 0},
		{0x20010069, 4, true, -1, 0, 0},
		{0x20010071, 4, true, -1, 0, 0},
		{0x2001008d, 4, false, -1, 0, 0},
		{0x20010089, 4, false, -1, 0, 0},
		{0x20010a49, 4, false, -1, 0, 0},
		{0x2001169e, 4, false, -1, 0, 0},
		{0x20010a45, 4, false, -1, 0, 0},
		{0x20005106, 1, false, -1, 0, 0},
		{0x20005107, 4, false, -1, 0, 0},
		{0x20012363, 4, false, -1, 0, 0},
		{0x20012373, 4, true, -1, 0, 0},
		{0x2001237b, 4, false, -1, 0, 0},
		{0x2001237f, 4, false, -1, 0, 0},
		{0x200118cb, 4, true, -1, 0, 0},
		{0x200118d3, 4, true, -1, 0, 0},
		{0x200118f3, 4, false, -1, 0, 0},
		{0x200118eb, 4, false, -1, 0, 0},
		{0x200122ab, 4, false, -1, 0, 0},
		{0x20012f00, 4, false, -1, 0, 0},
		{0x200122a7, 4, false, -1, 0, 0},
		{0x2000520c, 1, false, -1, 0, 0},
		{0x2000520d, 4, false, -1, 0, 0},
		{0x20013bc5, 4, false, -1, 0, 0},
		{0x20013bd5, 4, true, -1, 0, 0},
		{0x20013bdd, 4, false, -1, 0, 0},
		{0x20013be1, 4, false, -1, 0, 0},
		{0x2001312d, 4, true, -1, 0, 0},
		{0x20013135, 4, true, -1, 0, 0},
		{0x20013159, 4, false, -1, 0, 0},
		{0x2001314d, 4, false, -1, 0, 0},
		{0x20013b0d, 4, false, -1, 0, 0},
		{0x20014762, 4, false, -1, 0, 0},
		{0x20013b09, 4, false, -1, 0, 0},
		{0x20005312, 1, false, -1, 0, 0},
		{0x20005313, 4, false, -1, 0, 0},
		{0x20015427, 4, false, -1, 0, 0},
		{0x20015437, 4, true, -1, 0, 0},
		{0x2001543f, 4, false, -1, 0, 0},
		{0x20015443, 4, false, -1, 0, 0},
		{0x2001498f, 4, true, -1, 0, 0},
		{0x20014997, 4, true, -1, 0, 0},
		{0x200149bf, 4, false, -1, 0, 0},
		{0x200149af, 4, false, -1, 0, 0},
		{0x2001536f, 4, false, -1, 0, 0},
		{0x20015fc4, 4, false, -1, 0, 0},
		{0x2001536b, 4, false, -1, 0, 0},
		{0x20005418, 1, false, -1, 0, 0},
		{0x20005419, 4, false, -1, 0, 0},
		{0x20016c89, 4, false, -1, 0, 0},
		{0x20016c99, 4, true, -1, 0, 0},
		{0x20016ca1, 4, false, -1, 0, 0},
		{0x20016ca5, 4, false, -1, 0, 0},
		{0x200161f1, 4, true, -1, 0, 0},
		{0x200161f9, 4, true, -1, 0, 0},
		{0x20016225, 4, false, -1, 0, 0},
		{0x20016211, 4, false, -1, 0, 0},
		{0x20016bd1, 4, false, -1, 0, 0},
		{0x20017826, 4, false, -1, 0, 0},
		{0x20016bcd, 4, false, -1, 0, 0},
		{0x2000551e, 1, false, -1, 0, 0},
		{0x2000551f, 4, false, -1, 0, 0},
		{0x200184eb, 4, false, -1, 0, 0},
		{0x200184fb, 4, true, -1, 0, 0},
		{0x20018503, 4, false, -1, 0, 0},
		{0x20018507, 4, false, -1, 0, 0},
		{0x20017a53, 4, true, -1, 0, 0},
		{0x20017a5b, 4, true, -1, 0, 0},
		{0x20017a8b, 4, false, -1, 0, 0},
		{0x20017a73, 4, false, -1, 0, 0},
		{0x20018433, 4, false, -1, 0, 0},
		{0x20019088, 4, false, -1, 0, 0},
		{0x2001842f, 4, false, -1, 0, 0},
		{0x20005624, 1, false, -1, 0, 0},
		{0x20005625, 4, false, -1, 0, 0},
		{0x20019d4d, 4, false, -1, 0, 0},
		{0x20019d5d, 4, true, -1, 0, 0},
		{0x20019d65, 4, false, -1, 0, 0},
		{0x20019d69, 4, false, -1, 0, 0},
		{0x200192b5, 4, true, -1, 0, 0},
		{0x200192bd, 4, true, -1, 0, 0},
		{0x200192f1, 4, false, -1, 0, 0},
		{0x200192d5, 4, false, -1, 0, 0},
		{0x20019c95, 4, false, -1, 0, 0},
		{0x2001a8ea, 4, false, -1, 0, 0},
		{0x20019c91, 4, false, -1, 0, 0},
		{0x2000572a, 1, false, -1, 0, 0},
		{0x2000572b, 4, false, -1, 0, 0},
		{0x2001b5af, 4, false, -1, 0, 0},
		{0x2001b5bf, 4, true, -1, 0, 0},
		{0x2001b5c7, 4, false, -1, 0, 0},
		{0x2001b5cb, 4, false, -1, 0, 0},
		{0x2001ab17, 4, true, -1, 0, 0},
		{0x2001ab1f, 4, true, -1, 0, 0},
		{0x2001ab57, 4, false, -1, 0, 0},
		{0x2001ab37, 4, false, -1, 0, 0},
		{0x2001b4f7, 4, false, -1, 0, 0},
		{0x2001c14c, 4, false, -1, 0, 0},
		{0x2001b4f3, 4, false, -1, 0, 0},
		{0x2000b308, 4, false, -1, 0, 0},
		{0x20009015, 1, false, -1, 0, 0},
		{0x20009014, 1, false, -1, 0, 0},
		{0x200092ab, 1, false, -1, 0, 0},
		{0x2000924e, 1, false, -1, 0, 0},
		{0x200092e1, 4, false, -1, 0, 0},
		{0x20009107, 4, false, -1, 0, 0},
		{0x20009093, 4, false, -1, 0, 0},
		{0x2000909b, 4, false, -1, 0, 0},
		{0x2000902b, 4, false, -1, 0, 0},
		{0x2000902f, 4, false, -1, 0, 0},
		{0x20009033, 4, false, -1, 0, 0},
		{0x20009242, 4, false, -1, 0, 0},
		{0x20009087, 4, false, -1, 0, 0},
		{0x20009004, 4, false, -1, 0, 0},
		{0x20009008, 4, false, -1, 0, 0},
		{0x2000c146, 1, false, -1, 0, 0},
		{0x2000a0a7, 1, false, -1, 0, 0},
		{0x20009415, 1, false, -1, 0, 0},
		{0x20009414, 1, false, -1, 0, 0},
		{0x200096ab, 1, false, -1, 0, 0},
		{0x2000964e, 1, false, -1, 0, 0},
		{0x200096e1, 4, false, -1, 0, 0},
		{0x20009507, 4, false, -1, 0, 0},
		{0x20009493, 4, false, -1, 0, 0},
		{0x2000949b, 4, false, -1, 0, 0},
		{0x2000942b, 4, false, -1, 0, 0},
		{0x2000942f, 4, false, -1, 0, 0},
		{0x20009433, 4, false, -1, 0, 0},
		{0x20009642, 4, false, -1, 0, 0},
		{0x20009487, 4, false, -1, 0, 0},
		{0x20009404, 4, false, -1, 0, 0},
		{0x20009408, 4, false, -1, 0, 0},
		{0x2000c346, 1, false, -1, 0, 0},
		{0x2000a8a7, 1, false, -1, 0, 0},
		{0x20004000, 4, false, -1, 0, 0},
		{0x20004004, 4, false, -1, 0, 0},
		{0x20004010, 4, false, -1, 0, 0},
		{0x20004014, 4, false, -1, 0, 0},
		{0x20004018, 4, false, -1, 0, 0},
		{0x2000401c, 4, false, -1, 0, 0},
		{0x20004020, 4, false, -1, 0, 0},
		{0x20004024, 4, false, -1, 0, 0},
		{0x20004028, 4, false, -1, 0, 0},
		{0x2000402c, 4, false, -1, 0, 0},
		{0x20004030, 4, false, -1, 0, 0},
		{0x20004034, 4, false, -1, 0, 0},
		{0x20004038, 4, false, -1, 0, 0},
		{0x2000403c, 4, false, -1, 0, 0},
		{0x20004040, 4, false, -1, 0, 0},
		{0x20004044, 4, false, -1, 0, 0},
	},
	out: []mstFld{
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
		{0x20006000, 1, false, -1, 0, 0},
		{0x20006001, 1, false, -1, 0, 0},
		{0x2000690c, 4, false, -1, 0, 0},
		{0x20006a52, 1, false, -1, 0, 0},
		{0x20006a53, 1, false, -1, 0, 0},
		{0x20006a56, 1, false, -1, 0, 0},
		{0x20006a62, 1, false, -1, 0, 0},
		{0x2000604c, 1, false, -1, 0, 0},
		{0x20006120, 1, false, -1, 0, 0},
		{0x200061f4, 1, false, -1, 0, 0},
		{0x200062c8, 1, false, -1, 0, 0},
		{0x2000639c, 1, false, -1, 0, 0},
		{0x20006470, 1, false, -1, 0, 0},
		{0x20006544, 1, false, -1, 0, 0},
		{0x20006618, 1, false, -1, 0, 0},
		{0x6874f7, 4, false, -1, 0, 0},
		{0x687595, 4, false, -1, 0, 0},
		{0x65b9a8, 4, false, -1, 0, 0},
		{0x2000c658, 4, false, -1, 0, 0},
		{0x2000c758, 4, false, -1, 0, 0},
		{0x2000d0f4, 4, false, -1, 0, 0},
		{0x2000d2dc, 4, false, -1, 0, 0},
		{0x2000d2e0, 4, false, -1, 0, 0},
		{0x20010b1d, 4, false, -1, 0, 0},
		{0x2001169e, 4, false, -1, 0, 0},
		{0x2001237f, 4, false, -1, 0, 0},
		{0x20012f00, 4, false, -1, 0, 0},
		{0x20013be1, 4, false, -1, 0, 0},
		{0x20014762, 4, false, -1, 0, 0},
		{0x20015443, 4, false, -1, 0, 0},
		{0x20015fc4, 4, false, -1, 0, 0},
		{0x20016ca5, 4, false, -1, 0, 0},
		{0x20017826, 4, false, -1, 0, 0},
		{0x20018507, 4, false, -1, 0, 0},
		{0x20019088, 4, false, -1, 0, 0},
		{0x20019d69, 4, false, -1, 0, 0},
		{0x2001a8ea, 4, false, -1, 0, 0},
		{0x2001b5cb, 4, false, -1, 0, 0},
		{0x2001c14c, 4, false, -1, 0, 0},
		{0x20009033, 4, false, -1, 0, 0},
		{0x20009242, 4, false, -1, 0, 0},
		{0x20009107, 4, false, -1, 0, 0},
		{0x2000909b, 4, false, -1, 0, 0},
		{0x2000924e, 1, false, -1, 0, 0},
		{0x2000c146, 1, false, -1, 0, 0},
		{0x2000b414, 4, false, -1, 0, 0},
		{0x20009433, 4, false, -1, 0, 0},
		{0x20009642, 4, false, -1, 0, 0},
		{0x20009507, 4, false, -1, 0, 0},
		{0x2000949b, 4, false, -1, 0, 0},
		{0x2000964e, 1, false, -1, 0, 0},
		{0x2000c346, 1, false, -1, 0, 0},
		{0x2000b514, 4, false, -1, 0, 0},
	},
})

var rsrIn = []string{"partyMoving", "queueBusy", "queueDirty", "messageAll", "roundMessage", "combatRound", "engageCheck", "idleRounds", "groupCount", "group0Ready", "group0Active", "group0Living", "group1Ready", "group1Active", "group1Living", "environment", "screen", "dayTime", "dayCount", "uiFlags", "voiceGate", "queueTarget", "queueMode", "c0present", "c0posture", "c0state", "c0hp", "c0maxStamina", "c0stamina", "c0profession", "c0race", "c0level", "c0levelNone", "c0voiceMute", "c0fatigue", "c0exhausted", "c1present", "c1posture", "c1state", "c1hp", "c1maxStamina", "c1stamina", "c1profession", "c1race", "c1level", "c1levelNone", "c1voiceMute", "c1fatigue", "c1exhausted", "c2present", "c2posture", "c2state", "c2hp", "c2maxStamina", "c2stamina", "c2profession", "c2race", "c2level", "c2levelNone", "c2voiceMute", "c2fatigue", "c2exhausted", "c3present", "c3posture", "c3state", "c3hp", "c3maxStamina", "c3stamina", "c3profession", "c3race", "c3level", "c3levelNone", "c3voiceMute", "c3fatigue", "c3exhausted", "c4present", "c4posture", "c4state", "c4hp", "c4maxStamina", "c4stamina", "c4profession", "c4race", "c4level", "c4levelNone", "c4voiceMute", "c4fatigue", "c4exhausted", "c5present", "c5posture", "c5state", "c5hp", "c5maxStamina", "c5stamina", "c5profession", "c5race", "c5level", "c5levelNone", "c5voiceMute", "c5fatigue", "c5exhausted", "c6present", "c6posture", "c6state", "c6hp", "c6maxStamina", "c6stamina", "c6profession", "c6race", "c6level", "c6levelNone", "c6voiceMute", "c6fatigue", "c6exhausted", "c7present", "c7posture", "c7state", "c7hp", "c7maxStamina", "c7stamina", "c7profession", "c7race", "c7level", "c7levelNone", "c7voiceMute", "c7fatigue", "c7exhausted", "monsterCount", "m0teamMember", "m0present", "m0inCombat", "m0visualFlag", "m0reaction", "m0primary", "m0dur15", "m0dur17", "m0hp", "m0maxSp", "m0sp", "m0fatigue", "m0engaged", "m0group", "m0template", "m0teamTimer", "m0anim", "m1teamMember", "m1present", "m1inCombat", "m1visualFlag", "m1reaction", "m1primary", "m1dur15", "m1dur17", "m1hp", "m1maxSp", "m1sp", "m1fatigue", "m1engaged", "m1group", "m1template", "m1teamTimer", "m1anim", "engageRet0", "engageRet1", "xpCount", "ignoredReturn", "eligibleRet0", "eligibleRet1", "eligibleRet2", "eligibleRet3", "eligibleRet4", "eligibleRet5", "eligibleRet6", "eligibleRet7", "targetKind0", "targetKind1", "targetId0", "targetId1"}

var rsrOut = []string{"s4f0630_calls", "s4e79a0_calls", "s4e79a0_a1_0", "s4e79a0_a1_1", "s4e79a0_a1_2", "s4e79a0_a1_3", "s4e79a0_a1_4", "s4e79a0_a1_5", "s4e79a0_a1_6", "s4e79a0_a1_7", "s53a930_calls", "s53a930_a1_0", "s53a930_a1_1", "s53a930_a1_2", "s53a930_a1_3", "s53a930_a1_4", "s53a930_a1_5", "s53a930_a1_6", "s53a930_a1_7", "s53a930_a2_0", "s53a930_a2_1", "s53a930_a2_2", "s53a930_a2_3", "s53a930_a2_4", "s53a930_a2_5", "s53a930_a2_6", "s53a930_a2_7", "s5a1890_calls", "s5545f0_calls", "s5545f0_a1_0", "s5545f0_a2_0", "s58aad0_calls", "s58aad0_a1_0", "s58aad0_a1_1", "s58aad0_a1_2", "s58aad0_a2_0", "s58aad0_a2_1", "s58aad0_a2_2", "s58aad0_a3_0", "s58aad0_a3_1", "s58aad0_a3_2", "s408860_calls", "s408860_a1_0", "s408860_a2_0", "s531920_calls", "s531920_a1_0", "s531920_a1_1", "s502010_calls", "s502010_a1_0", "s5044d0_calls", "s5044d0_a1_0", "s4eee80_calls", "s52e690_calls", "s52e690_a1_0", "s52e690_a2_0", "s52e690_a3_0", "s52e690_a4_0", "s52e690_a5_0", "s52b1c0_calls", "s52b1c0_a1_0", "s52b1c0_a1_1", "s52b1c0_a1_2", "s52b1c0_a1_3", "s52b1c0_a1_4", "s52b1c0_a1_5", "s52b1c0_a1_6", "s52b1c0_a1_7", "s52b1c0_a2_0", "s52b1c0_a2_1", "s52b1c0_a2_2", "s52b1c0_a2_3", "s52b1c0_a2_4", "s52b1c0_a2_5", "s52b1c0_a2_6", "s52b1c0_a2_7", "s52b1c0_a3_0", "s52b1c0_a3_1", "s52b1c0_a3_2", "s52b1c0_a3_3", "s52b1c0_a3_4", "s52b1c0_a3_5", "s52b1c0_a3_6", "s52b1c0_a3_7", "s4e99c0_calls", "s4e99c0_a1_0", "s4e99c0_a1_1", "s4e99c0_a1_2", "s4e99c0_a1_3", "s4e99c0_a1_4", "s4e99c0_a1_5", "s4e99c0_a1_6", "s4e99c0_a1_7", "s4ee9d0_calls", "s4ee9d0_a1_0", "s4ee9d0_a1_1", "s4ee9d0_a1_2", "s4ee9d0_a1_3", "s4ee9d0_a1_4", "s4ee9d0_a1_5", "s4ee9d0_a1_6", "s4ee9d0_a1_7", "s562a50_calls", "s562a50_a1_0", "s534300_calls", "s535200_calls", "s5354e0_calls", "s4e4ab0_calls", "s593330_calls", "s53bf80_calls", "s536400_calls", "s5049c0_calls", "s5049c0_a1_0", "s5049c0_a1_1", "s5049c0_a1_2", "s5049c0_a1_3", "s5049c0_a2_0", "s5049c0_a2_1", "s5049c0_a2_2", "s5049c0_a2_3", "s51ac30_calls", "s51ac30_a1_0", "s51ac30_a1_1", "s51ac30_a3_0", "s51ac30_a3_1", "s4c6240_calls", "s4c6240_a1_0", "s4c6240_a1_1", "s4c6240_a2_0", "s4c6240_a2_1", "s4c62c0_calls", "s4c62c0_a1_0", "s4c62c0_a1_1", "s4c62c0_a2_0", "s4c62c0_a2_1", "s4c62c0_a3_0", "s4c62c0_a3_1", "s4acd80_calls", "s4acd80_a1_0", "s4acd80_a1_1", "s4acd80_a2_0", "s4acd80_a2_1", "s4c5a00_calls", "s4c5a00_a1_0", "s4c5a00_a1_1", "s4c5a00_a2_0", "s4c5a00_a2_1", "s4c5aa0_calls", "s4c5aa0_a1_0", "s4c5aa0_a1_1", "s4c5aa0_a2_0", "s4c5aa0_a2_1", "s4e4db0_calls", "s4e4db0_a1_0", "s4e4db0_a1_1", "s4e4db0_a2_0", "s4e4db0_a2_1", "s4e4db0_a3_0", "s4e4db0_a3_1", "busy0", "busy1", "queueDirty", "a52", "a53", "idleRounds", "a62", "c0down", "c1down", "c2down", "c3down", "c4down", "c5down", "c6down", "c7down", "dayTime", "dayCount", "lastTick", "sky0", "sky1", "uiFlags", "ui2dc", "ui2e0", "c0stamina", "c0fatigue", "c1stamina", "c1fatigue", "c2stamina", "c2fatigue", "c3stamina", "c3fatigue", "c4stamina", "c4fatigue", "c5stamina", "c5fatigue", "c6stamina", "c6fatigue", "c7stamina", "c7fatigue", "m0sp", "m0fatigue", "m0primary", "m0dur17", "m0visualFlag", "m0teamTimer", "group0Living", "m1sp", "m1fatigue", "m1primary", "m1dur17", "m1visualFlag", "m1teamTimer", "group1Living"}

var rsrStubs = map[uint32]*mcaStub{
	0x4f0630: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20004080, wr: []mcaWrite{}},
	0x4e79a0: {ret: 0x20004018, retN: 8, rec: [][4]uint32{{1, 4, 0x20004088, 8}}, cnt: 0x20004084, wr: []mcaWrite{}},
	0x53a930: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200040ac, 8}, {2, 4, 0x200040cc, 8}}, cnt: 0x200040a8, wr: []mcaWrite{}},
	0x5a1890: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x200040ec, wr: []mcaWrite{}},
	0x5545f0: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x200040f4, 1}, {2, 4, 0x200040f8, 1}}, cnt: 0x200040f0, wr: []mcaWrite{}},
	0x58aad0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004100, 3}, {2, 4, 0x2000410c, 3}, {3, 4, 0x20004118, 3}}, cnt: 0x200040fc, wr: []mcaWrite{}},
	0x408860: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004128, 1}, {2, 4, 0x2000412c, 1}}, cnt: 0x20004124, wr: []mcaWrite{}},
	0x531920: {ret: 0x20004000, retN: 2, rec: [][4]uint32{{1, 4, 0x20004134, 2}}, cnt: 0x20004130, wr: []mcaWrite{}},
	0x502010: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x20004140, 1}}, cnt: 0x2000413c, wr: []mcaWrite{}},
	0x5044d0: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x20004148, 1}}, cnt: 0x20004144, wr: []mcaWrite{}},
	0x4eee80: {ret: 0x20004010, retN: 0, rec: [][4]uint32{}, cnt: 0x2000414c, wr: []mcaWrite{}},
	0x52e690: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004154, 1}, {2, 4, 0x20004158, 1}, {3, 4, 0x2000415c, 1}, {4, 4, 0x20004160, 1}, {5, 4, 0x20004164, 1}}, cnt: 0x20004150, wr: []mcaWrite{}},
	0x52b1c0: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x2000416c, 8}, {2, 4, 0x2000418c, 8}, {3, 4, 0x200041ac, 8}}, cnt: 0x20004168, wr: []mcaWrite{}},
	0x4e99c0: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x200041d0, 8}}, cnt: 0x200041cc, wr: []mcaWrite{}},
	0x4ee9d0: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x200041f4, 8}}, cnt: 0x200041f0, wr: []mcaWrite{}},
	0x562a50: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004218, 1}}, cnt: 0x20004214, wr: []mcaWrite{}},
	0x534300: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x2000421c, wr: []mcaWrite{}},
	0x535200: {ret: 0x20004014, retN: 0, rec: [][4]uint32{}, cnt: 0x20004220, wr: []mcaWrite{}},
	0x5354e0: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20004224, wr: []mcaWrite{}},
	0x4e4ab0: {ret: 0x20004014, retN: 0, rec: [][4]uint32{}, cnt: 0x20004228, wr: []mcaWrite{}},
	0x593330: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x2000422c, wr: []mcaWrite{}},
	0x53bf80: {ret: 0x20004014, retN: 0, rec: [][4]uint32{}, cnt: 0x20004230, wr: []mcaWrite{}},
	0x536400: {ret: 0x20004014, retN: 0, rec: [][4]uint32{}, cnt: 0x20004234, wr: []mcaWrite{}},
	0x5049c0: {ret: 0x20004014, retN: 0, rec: [][4]uint32{{1, 4, 0x2000423c, 4}, {2, 4, 0x2000424c, 4}}, cnt: 0x20004238, wr: []mcaWrite{}},
	0x51ac30: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004260, 2}, {3, 4, 0x20004268, 2}}, cnt: 0x2000425c, wr: []mcaWrite{{2, 0x0, 4, 0x20004038, 2}, {2, 0x8, 4, 0x20004040, 2}}},
	0x4c6240: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004274, 2}, {2, 4, 0x2000427c, 2}}, cnt: 0x20004270, wr: []mcaWrite{}},
	0x4c62c0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20004288, 2}, {2, 4, 0x20004290, 2}, {3, 4, 0x20004298, 2}}, cnt: 0x20004284, wr: []mcaWrite{}},
	0x4acd80: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200042a4, 2}, {2, 4, 0x200042ac, 2}}, cnt: 0x200042a0, wr: []mcaWrite{}},
	0x4c5a00: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200042b8, 2}, {2, 4, 0x200042c0, 2}}, cnt: 0x200042b4, wr: []mcaWrite{}},
	0x4c5aa0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200042cc, 2}, {2, 4, 0x200042d4, 2}}, cnt: 0x200042c8, wr: []mcaWrite{}},
	0x4e4db0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200042e0, 2}, {2, 4, 0x200042e8, 2}, {3, 4, 0x200042f0, 2}}, cnt: 0x200042dc, wr: []mcaWrite{}},
}
