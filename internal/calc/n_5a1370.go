package calc

// 0x5a1370 spell_cast_power_submit: the spell panel's power button. It checks
// the panel's spell against the selected character's current target
// descriptor (0x537270: 0x53b7f0 descriptor, 0x5010f0 class, 0x537160
// compatibility), then commits the queued action (0x53a830) and queues the
// cast (0x4f9a20: 0x517ec0 slot, 0x4e7cc0 action set, descriptor copy).
//
// Address stubs: 0x536190 and 0x53c270 are D4 (F runs combatTargetValidNode /
// characterTargetableNode, Emu replays the case's r536190 / r53c270);
// 0x4ea5c0 and 0x4e8000 are D4 whose return is ignored, recorded only (their
// other effects are not copied back, as in n_4ea5c0.go); 0x569a50, 0x52e690
// and 0x4c5eb0 are 排除 and recorded. The panel's presentation tail (0x58aa20,
// 0x59f2b0, 0x560e10) runs natively in the case and is not compared; their
// own 排除 callees are count-only stubs there.
func init() {
	Nodes["005a1370"] = Node{Addr: "005a1370", In: scpIn, Out: scpOut, F: spellCastPowerSubmitNode, Emu: scpEmu}
}

func spellCastPowerSubmitNode(in []int64, r Rand) []int64 { return scpRun(true)(in, r) }

func scpEmu(in []int64, r Rand) []int64 { return scpRun(false)(in, r) }

func scpRun(real bool) func([]int64, Rand) []int64 {
	return mcaRun(&scpSpec, real, func(w *mcaWorld) { w.tab = scpStubs; scpSubmit(w) })
}

const (
	scpPanelPtr  = 0x69bf3c // spell panel object: +0xf8 caster, +0x100 spell, +0x108 power, +0x10c mode, +0xc5a/+0xc5b flags
	scpSel       = 0x68518d // selected party slot
	scpChars     = 0x685174 // character records, stride 0x1862
	scpQueue     = 0x685178 // per-slot action records, stride 0x106
	scpDialog    = 0x68edd0 // 0x56aa20: nonzero while a modal dialog is up
	scpUI        = 0x68edcc // combat UI object (0x53b7f0, 0x5010f0; 0x58aa20 / 0x59f2b0 dereference it)
	scpStrings   = 0x68c09c // string table (0x569a50 message ids at +0x1e90 / +0x1e94)
	scpPhase     = 0x68ec78 // game phase; 7 = combat round (0x53b7f0)
	scpCombat    = 0x683f94 // combat flag (0x53b7f0, 0x53a830, 0x4e7cc0)
	scpSpecSpell = 0x683f95 // special-target spell flag (0x53b7f0 source 2)
	scpSpecItem  = 0x683f96 // special-target item flag (0x53b7f0 source 2)
	scpRefresh   = 0x683f9b // 0x5a1370: clear the message panel first (0x58aa20)
	scpPartyOK   = 0x685116 // 0x537270: class 2 passes while set
	scpSpecDesc  = 0x68408b // source-2 descriptor (8 dwords); 0x6840ab its target, +4 the item
	scpSpecItemP = 0x6840af // source-2 item instance (0x4e77b0 stores 0x59e0d0 here)
	scpHand1     = 0x69b9a4 // 0x59e0d0: first choice
	scpHand0     = 0x69b9a0 // 0x59e0d0: fallback
	scpItemTypes = 0x683f78 // 0x520880: item type count
	scpItemDefs  = 0x68516c // 0x520880: item definitions, stride 0x10d
	scpList      = 0x683fad // 0x53aeb0: monster object list (data +0, count +8)
	scpCharSize  = 0x1862
	scpQSize     = 0x106
	scpSpellSize = 0x1bf
	scpSpellCap  = 150 // fixture cap: spell ids 0..149
	scpItemCap   = 8   // fixture cap: item type ids 0..7
	scpListCap   = 3   // fixture cap: three listed monsters
)

func scpQ(w *mcaWorld, slot uint32) uint32 { return w.u32(scpQueue) + slot*scpQSize }

func scpChar(w *mcaWorld, slot uint32) uint32 { return w.u32(scpChars) + slot*scpCharSize }

// scpSource is the descriptor source shared by 0x53b7f0 and 0x53bc90 (ctx 6).
func scpSource(w *mcaWorld, slot uint32) int {
	ui := w.u32(scpUI)
	if w.u32(scpPhase) == 7 && ui != 0 && w.u32(ui+0x2f0) != 0xffffffff {
		switch a, f := w.u32(ui+0x2f0), w.u8(ui+0x2f8); {
		case a == 7 && f != 0:
			return 3
		case a == 8 && f != 0:
			return 4
		}
		return 7
	}
	if slot == w.u32(scpSel) && (w.u8(scpSpecSpell) != 0 || w.u8(scpSpecItem) != 0) {
		return 2
	}
	if w.u8(scpCombat) != 0 {
		return 1
	}
	return 0
}

// scpDesc is 0x53b7f0(slot, 6): the slot's current target descriptor.
func scpDesc(w *mcaWorld, slot uint32) uint32 {
	q := scpQ(w, slot)
	switch scpSource(w, slot) {
	case 0:
		return q + 0x1d
	case 1:
		return q + 0x4d
	case 2:
		return scpSpecDesc
	case 3:
		return q + 0x81
	case 4:
		return q + 0xa9
	}
	return 0
}

// scpClass is 0x5010f0(spell, 0, 6) through 0x4ac9d0 and the 0x501190 dispatch.
func scpClass(w *mcaWorld, spell uint32) uint32 {
	if spell == 0 {
		return 0
	}
	if spell >= scpSpellCap {
		panic("spell cast power submit: spell id past the fixture's spell records")
	}
	kind := w.u32(w.u32(0x65be1c) + spell*scpSpellSize + 0x137)
	if kind > 10 {
		panic("spell cast power submit: spell target kind above 10 (0x501167 asserts)")
	}
	if itemSpellTargetsErr != nil {
		panic(itemSpellTargetsErr)
	}
	switch t := itemSpellTargets[kind]; t {
	case -1:
		if spell == 0x58 {
			return 7
		}
		return 1
	case -2:
		if w.u32(scpUI) != 0 {
			return 0
		}
		return 6
	default:
		return uint32(t)
	}
}

// scpCompatible is 0x537160(q, cls).
func scpCompatible(w *mcaWorld, q, cls uint32) bool {
	if q == 0 {
		return false
	}
	if cls-1 > 7 {
		return true
	}
	if currentTargetChecksErr != nil {
		panic(currentTargetChecksErr)
	}
	kind := w.u32(q)
	set := func(off uint32) bool { return w.u32(q+off) != 0xffffffff }
	var call bool
	switch currentTargetChecks[cls-1] {
	case 0x53717c:
		call = (kind == 1 && set(4)) || (kind == 3 && set(8))
	case 0x5371a4:
		call = kind == 9 && set(4)
	case 0x5371b9:
		call = (kind == 4 && set(0xc)) || kind == 2
	case 0x5371cf:
		call = kind == 7 && w.u32(q+0x1c) != 0
	case 0x5371e5:
		call = kind == 6
	case 0x5371f4:
		return true
	}
	return call && mpnValid(w, q)
}

// scpAllowed is 0x537270(slot, spell).
func scpAllowed(w *mcaWorld, slot, spell uint32) bool {
	desc := scpDesc(w, slot)
	cls := scpClass(w, spell)
	if cls == 2 && w.u8(scpPartyOK) != 0 {
		return true
	}
	return scpCompatible(w, desc, cls)
}

// scpTargetable is 0x53c270(slot).
func scpTargetable(w *mcaWorld, slot uint32) bool {
	k := w.call(0x53c270, slot)
	if !w.real {
		return byte(w.replay(0x53c270, k)) != 0
	}
	x := []int64{int64(slot)}
	for i := uint32(0); i < 8; i++ {
		c := scpChar(w, i)
		x = append(x, int64(w.u8(scpQ(w, i))), int64(int32(w.u32(c+0xb11))), int64(w.u32(c+0xb01)))
	}
	return characterTargetableNode(x, w.r)[0] != 0
}

// scpAction is 0x4e77b0(slot, 6, &action, &b, NULL, &target): the slot's
// current action and its target record (the item instance sits at +4).
func scpAction(w *mcaWorld, slot uint32) (action, target uint32) {
	q := scpQ(w, slot)
	switch scpSource(w, slot) {
	case 0:
		action, target = w.u32(q+1), q+0x15
	case 1:
		action, target = w.u32(q+0x3d), q+0x45
	case 2:
		if w.u8(scpSpecSpell) != 0 {
			action = 7
		} else {
			hand := w.u32(scpHand1) // 0x59e0d0
			if hand == 0 {
				hand = w.u32(scpHand0)
			}
			w.put(scpSpecItemP, 4, uint64(hand))
			action = 8
		}
		target = scpSpecDesc + 0x20
	case 3:
		action, target = 7, q+0x79
	case 4:
		action, target = 8, q+0xa1
	case 7:
		action = w.u32(w.u32(scpUI) + 0x2f0)
	}
	if !scpTargetable(w, slot) && action != 10 && action != 11 {
		return 0xffffffff, 0
	}
	return action, target
}

// scpItemCheck is 0x520880(item instance): only its asserts matter (the
// caller ignores the item's byte it returns).
func scpItemCheck(w *mcaWorld, p uint32) {
	if p == 0 {
		return
	}
	id := int32(w.u32(p))
	switch {
	case id == -1:
		panic("spell cast power submit: item instance without a type (0x520880 asserts)")
	case id >= int32(w.u32(scpItemTypes)):
		panic("spell cast power submit: item type past the type count (0x520880 asserts)")
	case id < 0 || id >= scpItemCap:
		panic("spell cast power submit: item type outside the fixture's item records")
	}
}

// scpCommit is 0x53a830: the selected slot's queued action is committed.
func scpCommit(w *mcaWorld) {
	sel := w.u32(scpSel)
	action, target := scpAction(w, sel)
	if action == 8 { // an item use: its instance must be typed
		if target == 0 {
			panic("spell cast power submit: item use without a target record (0x53a869 reads address 4)")
		}
		scpItemCheck(w, w.u32(target+4))
	}
	dst := scpQ(w, sel) + 0x1d
	if w.u8(scpCombat) != 0 {
		dst = scpQ(w, sel) + 0x4d
	}
	for i := uint32(0); i < 32; i += 4 {
		w.put(dst+i, 4, uint64(w.u32(scpSpecDesc+i)))
	}
	// 0x52fe80(sel, 1)
	w.stub(0x52e690, scpChar(w, sel), w.u32(0x68c50c), 0, w.u32(0x5ed8c8), w.u32(0x5ed914))
}

// scpSlotOf is 0x517ec0(character): its party slot.
func scpSlotOf(w *mcaWorld, c uint32) uint32 {
	if w.u8(c+4) == 0 {
		panic("spell cast power submit: caster record not present (0x517ec0 asserts)")
	}
	for i := uint32(0); i < 8; i++ {
		if scpChar(w, i) == c {
			return i
		}
	}
	panic("spell cast power submit: caster not in the party (0x517ec0 asserts)")
}

// scpClearMarks is 0x53aeb0(slot): drop the slot's bit from every listed
// monster's selection marks.
func scpClearMarks(w *mcaWorld, slot uint32) {
	v := w.u32(scpList)
	count := func() int32 { // 0x5e2c70
		if v == 0 {
			return 0
		}
		return int32(w.u32(v + 8))
	}
	if n := count(); n < 0 || n > scpListCap {
		panic("spell cast power submit: monster list past the fixture's records")
	}
	for i := int32(0); i < count(); i++ {
		e := w.u32(w.u32(v) + 4*uint32(i)) // 0x4e5620 / 0x5e2870
		if e == 0 {
			panic("spell cast power submit: null listed monster (0x4e5620 asserts)")
		}
		s := w.u32(e + 0xc)
		if w.u8(e+0x14) == 0 || s == 0 {
			continue
		}
		mk := w.u32(s+0x1d8) + 0x5bc // 0x4c5820 / 0x4c5840
		bit := byte(1) << slot
		if m := w.u8(mk); m&bit != 0 {
			w.put(mk, 1, uint64(m&^bit))
			w.stub(0x4c5eb0, slot, w.u32(e), 0)
		}
	}
}

// scpSetAction is 0x4e7cc0(slot, 7, spell, &{power, 0}, 0, 1).
func scpSetAction(w *mcaWorld, slot, spell, power uint32) {
	q := scpQ(w, slot)
	if w.u8(scpCombat) != 0 {
		w.stub(0x4e8000, slot, 7, spell, 0, 1) // via 0x4e7ee0; arg 4 (a stack record) is not recorded
		w.put(q+0xd0, 1, 7)
		return
	}
	w.put(q+1, 4, 7)
	w.put(q+5, 4, uint64(spell))
	w.put(q+9, 4, 0xffffffff)
	w.put(q+0x15, 4, uint64(power))
	w.put(q+0x19, 4, 0)
	w.stub(0x4ea5c0, slot)
	scpClearMarks(w, slot)
}

// scpQueueCast is 0x4f9a20(caster, spell, power).
func scpQueueCast(w *mcaWorld, c, spell, power uint32) {
	slot := scpSlotOf(w, c)
	scpSetAction(w, slot, spell, power)
	desc := scpDesc(w, slot)
	q := scpQ(w, slot)
	w.put(q+0x79, 4, uint64(power))
	w.put(q+0x75, 4, uint64(spell))
	w.put(q+0x7d, 4, 0)
	if desc == 0 {
		panic("spell cast power submit: no target descriptor to copy (0x4f9a8f reads address 0)")
	}
	for i := uint32(0); i < 32; i += 4 {
		w.put(q+0x81+i, 4, uint64(w.u32(desc+i)))
	}
}

func scpSubmit(w *mcaWorld) {
	p := w.u32(scpPanelPtr)
	spell := w.u32(p + 0x100)
	dialog := func() bool { return w.u32(scpDialog) != 0 } // 0x56aa20
	ok := false
	if spell != 0 && (w.u32(p+0x108) != 0xffffffff || w.u32(p+0x10c) == 3) {
		ok = scpAllowed(w, w.u32(scpSel), spell)
	}
	switch {
	case spell == 0x4b && w.u8(p+0xc5b) == 0 && !dialog():
		if w.u8(scpChar(w, w.u32(scpSel))+0x1861) != 0 {
			w.stub(0x569a50, w.u32(w.u32(scpStrings)+0x1e90), 0x5a0ae0, 1, 1)
		}
	case spell == 0x49 && w.u8(p+0xc5b) == 0 && !dialog():
		w.stub(0x569a50, w.u32(w.u32(scpStrings)+0x1e94), 0x5a0ae0, 1, 1)
	}
	if !ok || dialog() {
		return
	}
	w.put(p+0xc5a, 1, 1)
	scpCommit(w)
	w.put(p+0xc5a, 1, 0)
	caster := w.u32(p + 0xf8)
	if spell == 0x17 {
		scpSlotOf(w, caster) // then 0x560e10 (screen switch, presentation)
		return
	}
	if w.u32(scpUI) == 0 {
		panic("spell cast power submit: null combat UI object (0x58aa20 / 0x59f2b0 dereference it)")
	}
	scpQueueCast(w, caster, spell, w.u32(p+0x108)+1) // then 0x59f2b0 (presentation)
}

// Code below is generated from the same description as re/emu/cases/005a1370.json.

var scpSpec = mstSpec{
	set: []mstSet{
		{0x685174, 4, 0x20000000},
		{0x685178, 4, 0x20010000},
		{0x65be1c, 4, 0x20011000},
		{0x69bf3c, 4, 0x20023000},
		{0x68c09c, 4, 0x20026000},
		{0x68516c, 4, 0x2002a000},
		{0x2002b100, 4, 0x2002b120},
		{0x2002b120, 4, 0x2002b200},
		{0x2002b4d8, 4, 0x2002c000},
		{0x2002b124, 4, 0x2002b220},
		{0x2002b6d8, 4, 0x2002c600},
		{0x2002b128, 4, 0x2002b240},
		{0x2002b8d8, 4, 0x2002cc00},
		{0x20000004, 1, 0x1},
		{0x20001861, 1, 0x1},
		{0x20010000, 1, 0x1},
		{0x20000b11, 4, 0x5},
		{0x2001001d, 4, 0xd0001d0},
		{0x20010021, 4, 0xd0001d1},
		{0x20010025, 4, 0xd0001d2},
		{0x20010029, 4, 0xd0001d3},
		{0x2001002d, 4, 0xd0001d4},
		{0x20010031, 4, 0xd0001d5},
		{0x20010035, 4, 0xd0001d6},
		{0x20010039, 4, 0xd0001d7},
		{0x2001004d, 4, 0xd0004d0},
		{0x20010051, 4, 0xd0004d1},
		{0x20010055, 4, 0xd0004d2},
		{0x20010059, 4, 0xd0004d3},
		{0x2001005d, 4, 0xd0004d4},
		{0x20010061, 4, 0xd0004d5},
		{0x20010065, 4, 0xd0004d6},
		{0x20010069, 4, 0xd0004d7},
		{0x20010081, 4, 0xd000810},
		{0x20010085, 4, 0xd000811},
		{0x20010089, 4, 0xd000812},
		{0x2001008d, 4, 0xd000813},
		{0x20010091, 4, 0xd000814},
		{0x20010095, 4, 0xd000815},
		{0x20010099, 4, 0xd000816},
		{0x2001009d, 4, 0xd000817},
		{0x200100a9, 4, 0xd000a90},
		{0x200100ad, 4, 0xd000a91},
		{0x200100b1, 4, 0xd000a92},
		{0x200100b5, 4, 0xd000a93},
		{0x200100b9, 4, 0xd000a94},
		{0x200100bd, 4, 0xd000a95},
		{0x200100c1, 4, 0xd000a96},
		{0x200100c5, 4, 0xd000a97},
		{0x20010001, 4, 0x60},
		{0x20010005, 4, 0x70},
		{0x2001003d, 4, 0x80},
		{0x20010041, 4, 0x90},
		{0x20010075, 4, 0xa0},
		{0x20010079, 4, 0xb0},
		{0x20001866, 1, 0x1},
		{0x200030c3, 1, 0x1},
		{0x20010106, 1, 0x1},
		{0x20002373, 4, 0x5},
		{0x20010123, 4, 0xd0101d0},
		{0x20010127, 4, 0xd0101d1},
		{0x2001012b, 4, 0xd0101d2},
		{0x2001012f, 4, 0xd0101d3},
		{0x20010133, 4, 0xd0101d4},
		{0x20010137, 4, 0xd0101d5},
		{0x2001013b, 4, 0xd0101d6},
		{0x2001013f, 4, 0xd0101d7},
		{0x20010153, 4, 0xd0104d0},
		{0x20010157, 4, 0xd0104d1},
		{0x2001015b, 4, 0xd0104d2},
		{0x2001015f, 4, 0xd0104d3},
		{0x20010163, 4, 0xd0104d4},
		{0x20010167, 4, 0xd0104d5},
		{0x2001016b, 4, 0xd0104d6},
		{0x2001016f, 4, 0xd0104d7},
		{0x20010187, 4, 0xd010810},
		{0x2001018b, 4, 0xd010811},
		{0x2001018f, 4, 0xd010812},
		{0x20010193, 4, 0xd010813},
		{0x20010197, 4, 0xd010814},
		{0x2001019b, 4, 0xd010815},
		{0x2001019f, 4, 0xd010816},
		{0x200101a3, 4, 0xd010817},
		{0x200101af, 4, 0xd010a90},
		{0x200101b3, 4, 0xd010a91},
		{0x200101b7, 4, 0xd010a92},
		{0x200101bb, 4, 0xd010a93},
		{0x200101bf, 4, 0xd010a94},
		{0x200101c3, 4, 0xd010a95},
		{0x200101c7, 4, 0xd010a96},
		{0x200101cb, 4, 0xd010a97},
		{0x20010107, 4, 0x61},
		{0x2001010b, 4, 0x71},
		{0x20010143, 4, 0x81},
		{0x20010147, 4, 0x91},
		{0x2001017b, 4, 0xa1},
		{0x2001017f, 4, 0xb1},
		{0x200030c8, 1, 0x1},
		{0x20004925, 1, 0x1},
		{0x2001020c, 1, 0x1},
		{0x20003bd5, 4, 0x5},
		{0x20010229, 4, 0xd0201d0},
		{0x2001022d, 4, 0xd0201d1},
		{0x20010231, 4, 0xd0201d2},
		{0x20010235, 4, 0xd0201d3},
		{0x20010239, 4, 0xd0201d4},
		{0x2001023d, 4, 0xd0201d5},
		{0x20010241, 4, 0xd0201d6},
		{0x20010245, 4, 0xd0201d7},
		{0x20010259, 4, 0xd0204d0},
		{0x2001025d, 4, 0xd0204d1},
		{0x20010261, 4, 0xd0204d2},
		{0x20010265, 4, 0xd0204d3},
		{0x20010269, 4, 0xd0204d4},
		{0x2001026d, 4, 0xd0204d5},
		{0x20010271, 4, 0xd0204d6},
		{0x20010275, 4, 0xd0204d7},
		{0x2001028d, 4, 0xd020810},
		{0x20010291, 4, 0xd020811},
		{0x20010295, 4, 0xd020812},
		{0x20010299, 4, 0xd020813},
		{0x2001029d, 4, 0xd020814},
		{0x200102a1, 4, 0xd020815},
		{0x200102a5, 4, 0xd020816},
		{0x200102a9, 4, 0xd020817},
		{0x200102b5, 4, 0xd020a90},
		{0x200102b9, 4, 0xd020a91},
		{0x200102bd, 4, 0xd020a92},
		{0x200102c1, 4, 0xd020a93},
		{0x200102c5, 4, 0xd020a94},
		{0x200102c9, 4, 0xd020a95},
		{0x200102cd, 4, 0xd020a96},
		{0x200102d1, 4, 0xd020a97},
		{0x2001020d, 4, 0x62},
		{0x20010211, 4, 0x72},
		{0x20010249, 4, 0x82},
		{0x2001024d, 4, 0x92},
		{0x20010281, 4, 0xa2},
		{0x20010285, 4, 0xb2},
		{0x2000492a, 1, 0x1},
		{0x20006187, 1, 0x1},
		{0x20010312, 1, 0x1},
		{0x20005437, 4, 0x5},
		{0x2001032f, 4, 0xd0301d0},
		{0x20010333, 4, 0xd0301d1},
		{0x20010337, 4, 0xd0301d2},
		{0x2001033b, 4, 0xd0301d3},
		{0x2001033f, 4, 0xd0301d4},
		{0x20010343, 4, 0xd0301d5},
		{0x20010347, 4, 0xd0301d6},
		{0x2001034b, 4, 0xd0301d7},
		{0x2001035f, 4, 0xd0304d0},
		{0x20010363, 4, 0xd0304d1},
		{0x20010367, 4, 0xd0304d2},
		{0x2001036b, 4, 0xd0304d3},
		{0x2001036f, 4, 0xd0304d4},
		{0x20010373, 4, 0xd0304d5},
		{0x20010377, 4, 0xd0304d6},
		{0x2001037b, 4, 0xd0304d7},
		{0x20010393, 4, 0xd030810},
		{0x20010397, 4, 0xd030811},
		{0x2001039b, 4, 0xd030812},
		{0x2001039f, 4, 0xd030813},
		{0x200103a3, 4, 0xd030814},
		{0x200103a7, 4, 0xd030815},
		{0x200103ab, 4, 0xd030816},
		{0x200103af, 4, 0xd030817},
		{0x200103bb, 4, 0xd030a90},
		{0x200103bf, 4, 0xd030a91},
		{0x200103c3, 4, 0xd030a92},
		{0x200103c7, 4, 0xd030a93},
		{0x200103cb, 4, 0xd030a94},
		{0x200103cf, 4, 0xd030a95},
		{0x200103d3, 4, 0xd030a96},
		{0x200103d7, 4, 0xd030a97},
		{0x20010313, 4, 0x63},
		{0x20010317, 4, 0x73},
		{0x2001034f, 4, 0x83},
		{0x20010353, 4, 0x93},
		{0x20010387, 4, 0xa3},
		{0x2001038b, 4, 0xb3},
		{0x2000618c, 1, 0x1},
		{0x200079e9, 1, 0x1},
		{0x20010418, 1, 0x1},
		{0x20006c99, 4, 0x5},
		{0x20010435, 4, 0xd0401d0},
		{0x20010439, 4, 0xd0401d1},
		{0x2001043d, 4, 0xd0401d2},
		{0x20010441, 4, 0xd0401d3},
		{0x20010445, 4, 0xd0401d4},
		{0x20010449, 4, 0xd0401d5},
		{0x2001044d, 4, 0xd0401d6},
		{0x20010451, 4, 0xd0401d7},
		{0x20010465, 4, 0xd0404d0},
		{0x20010469, 4, 0xd0404d1},
		{0x2001046d, 4, 0xd0404d2},
		{0x20010471, 4, 0xd0404d3},
		{0x20010475, 4, 0xd0404d4},
		{0x20010479, 4, 0xd0404d5},
		{0x2001047d, 4, 0xd0404d6},
		{0x20010481, 4, 0xd0404d7},
		{0x20010499, 4, 0xd040810},
		{0x2001049d, 4, 0xd040811},
		{0x200104a1, 4, 0xd040812},
		{0x200104a5, 4, 0xd040813},
		{0x200104a9, 4, 0xd040814},
		{0x200104ad, 4, 0xd040815},
		{0x200104b1, 4, 0xd040816},
		{0x200104b5, 4, 0xd040817},
		{0x200104c1, 4, 0xd040a90},
		{0x200104c5, 4, 0xd040a91},
		{0x200104c9, 4, 0xd040a92},
		{0x200104cd, 4, 0xd040a93},
		{0x200104d1, 4, 0xd040a94},
		{0x200104d5, 4, 0xd040a95},
		{0x200104d9, 4, 0xd040a96},
		{0x200104dd, 4, 0xd040a97},
		{0x20010419, 4, 0x64},
		{0x2001041d, 4, 0x74},
		{0x20010455, 4, 0x84},
		{0x20010459, 4, 0x94},
		{0x2001048d, 4, 0xa4},
		{0x20010491, 4, 0xb4},
		{0x200079ee, 1, 0x1},
		{0x2000924b, 1, 0x1},
		{0x2001051e, 1, 0x1},
		{0x200084fb, 4, 0x5},
		{0x2001053b, 4, 0xd0501d0},
		{0x2001053f, 4, 0xd0501d1},
		{0x20010543, 4, 0xd0501d2},
		{0x20010547, 4, 0xd0501d3},
		{0x2001054b, 4, 0xd0501d4},
		{0x2001054f, 4, 0xd0501d5},
		{0x20010553, 4, 0xd0501d6},
		{0x20010557, 4, 0xd0501d7},
		{0x2001056b, 4, 0xd0504d0},
		{0x2001056f, 4, 0xd0504d1},
		{0x20010573, 4, 0xd0504d2},
		{0x20010577, 4, 0xd0504d3},
		{0x2001057b, 4, 0xd0504d4},
		{0x2001057f, 4, 0xd0504d5},
		{0x20010583, 4, 0xd0504d6},
		{0x20010587, 4, 0xd0504d7},
		{0x2001059f, 4, 0xd050810},
		{0x200105a3, 4, 0xd050811},
		{0x200105a7, 4, 0xd050812},
		{0x200105ab, 4, 0xd050813},
		{0x200105af, 4, 0xd050814},
		{0x200105b3, 4, 0xd050815},
		{0x200105b7, 4, 0xd050816},
		{0x200105bb, 4, 0xd050817},
		{0x200105c7, 4, 0xd050a90},
		{0x200105cb, 4, 0xd050a91},
		{0x200105cf, 4, 0xd050a92},
		{0x200105d3, 4, 0xd050a93},
		{0x200105d7, 4, 0xd050a94},
		{0x200105db, 4, 0xd050a95},
		{0x200105df, 4, 0xd050a96},
		{0x200105e3, 4, 0xd050a97},
		{0x2001051f, 4, 0x65},
		{0x20010523, 4, 0x75},
		{0x2001055b, 4, 0x85},
		{0x2001055f, 4, 0x95},
		{0x20010593, 4, 0xa5},
		{0x20010597, 4, 0xb5},
		{0x20009250, 1, 0x1},
		{0x2000aaad, 1, 0x1},
		{0x20010624, 1, 0x1},
		{0x20009d5d, 4, 0x5},
		{0x20010641, 4, 0xd0601d0},
		{0x20010645, 4, 0xd0601d1},
		{0x20010649, 4, 0xd0601d2},
		{0x2001064d, 4, 0xd0601d3},
		{0x20010651, 4, 0xd0601d4},
		{0x20010655, 4, 0xd0601d5},
		{0x20010659, 4, 0xd0601d6},
		{0x2001065d, 4, 0xd0601d7},
		{0x20010671, 4, 0xd0604d0},
		{0x20010675, 4, 0xd0604d1},
		{0x20010679, 4, 0xd0604d2},
		{0x2001067d, 4, 0xd0604d3},
		{0x20010681, 4, 0xd0604d4},
		{0x20010685, 4, 0xd0604d5},
		{0x20010689, 4, 0xd0604d6},
		{0x2001068d, 4, 0xd0604d7},
		{0x200106a5, 4, 0xd060810},
		{0x200106a9, 4, 0xd060811},
		{0x200106ad, 4, 0xd060812},
		{0x200106b1, 4, 0xd060813},
		{0x200106b5, 4, 0xd060814},
		{0x200106b9, 4, 0xd060815},
		{0x200106bd, 4, 0xd060816},
		{0x200106c1, 4, 0xd060817},
		{0x200106cd, 4, 0xd060a90},
		{0x200106d1, 4, 0xd060a91},
		{0x200106d5, 4, 0xd060a92},
		{0x200106d9, 4, 0xd060a93},
		{0x200106dd, 4, 0xd060a94},
		{0x200106e1, 4, 0xd060a95},
		{0x200106e5, 4, 0xd060a96},
		{0x200106e9, 4, 0xd060a97},
		{0x20010625, 4, 0x66},
		{0x20010629, 4, 0x76},
		{0x20010661, 4, 0x86},
		{0x20010665, 4, 0x96},
		{0x20010699, 4, 0xa6},
		{0x2001069d, 4, 0xb6},
		{0x2000aab2, 1, 0x1},
		{0x2000c30f, 1, 0x1},
		{0x2001072a, 1, 0x1},
		{0x2000b5bf, 4, 0x5},
		{0x20010747, 4, 0xd0701d0},
		{0x2001074b, 4, 0xd0701d1},
		{0x2001074f, 4, 0xd0701d2},
		{0x20010753, 4, 0xd0701d3},
		{0x20010757, 4, 0xd0701d4},
		{0x2001075b, 4, 0xd0701d5},
		{0x2001075f, 4, 0xd0701d6},
		{0x20010763, 4, 0xd0701d7},
		{0x20010777, 4, 0xd0704d0},
		{0x2001077b, 4, 0xd0704d1},
		{0x2001077f, 4, 0xd0704d2},
		{0x20010783, 4, 0xd0704d3},
		{0x20010787, 4, 0xd0704d4},
		{0x2001078b, 4, 0xd0704d5},
		{0x2001078f, 4, 0xd0704d6},
		{0x20010793, 4, 0xd0704d7},
		{0x200107ab, 4, 0xd070810},
		{0x200107af, 4, 0xd070811},
		{0x200107b3, 4, 0xd070812},
		{0x200107b7, 4, 0xd070813},
		{0x200107bb, 4, 0xd070814},
		{0x200107bf, 4, 0xd070815},
		{0x200107c3, 4, 0xd070816},
		{0x200107c7, 4, 0xd070817},
		{0x200107d3, 4, 0xd070a90},
		{0x200107d7, 4, 0xd070a91},
		{0x200107db, 4, 0xd070a92},
		{0x200107df, 4, 0xd070a93},
		{0x200107e3, 4, 0xd070a94},
		{0x200107e7, 4, 0xd070a95},
		{0x200107eb, 4, 0xd070a96},
		{0x200107ef, 4, 0xd070a97},
		{0x2001072b, 4, 0x67},
		{0x2001072f, 4, 0x77},
		{0x20010767, 4, 0x87},
		{0x2001076b, 4, 0x97},
		{0x2001079f, 4, 0xa7},
		{0x200107a3, 4, 0xb7},
	},
	in: []mstFld{
		{0x68518d, 4, true, -1, 0, 0},
		{0x200230f8, 4, false, -1, 0, 0},
		{0x20023100, 4, false, -1, 0, 0},
		{0x20023108, 4, true, -1, 0, 0},
		{0x2002310c, 4, false, -1, 0, 0},
		{0x20023c5b, 1, false, -1, 0, 0},
		{0x20023c5a, 1, false, -1, 0, 0},
		{0x68edd0, 4, false, -1, 0, 0},
		{0x685116, 1, false, -1, 0, 0},
		{0x20011137, 4, false, 2, 150, 0x1bf},
		{0x68edcc, 4, false, -1, 0, 0},
		{0x683fad, 4, false, -1, 0, 0},
		{0x68ec78, 4, false, -1, 0, 0},
		{0x200282f0, 4, true, -1, 0, 0},
		{0x200282f8, 1, false, -1, 0, 0},
		{0x683f94, 1, false, -1, 0, 0},
		{0x683f95, 1, false, -1, 0, 0},
		{0x683f96, 1, false, -1, 0, 0},
		{0x683f9b, 1, false, -1, 0, 0},
		{0x20001861, 1, false, 0, 8, 0x1862},
		{0x20027e90, 4, false, -1, 0, 0},
		{0x20027e94, 4, false, -1, 0, 0},
		{0x2001001d, 4, false, 0, 8, 0x106},
		{0x20010021, 4, true, 0, 8, 0x106},
		{0x20010025, 4, true, 0, 8, 0x106},
		{0x20010029, 4, true, 0, 8, 0x106},
		{0x2001002d, 4, false, 0, 8, 0x106},
		{0x20010031, 4, false, 0, 8, 0x106},
		{0x20010035, 4, false, 0, 8, 0x106},
		{0x20010039, 4, false, 0, 8, 0x106},
		{0x2001004d, 4, false, 0, 8, 0x106},
		{0x20010051, 4, true, 0, 8, 0x106},
		{0x20010055, 4, true, 0, 8, 0x106},
		{0x20010059, 4, true, 0, 8, 0x106},
		{0x2001005d, 4, false, 0, 8, 0x106},
		{0x20010061, 4, false, 0, 8, 0x106},
		{0x20010065, 4, false, 0, 8, 0x106},
		{0x20010069, 4, false, 0, 8, 0x106},
		{0x68408b, 4, false, -1, 0, 0},
		{0x68408f, 4, true, -1, 0, 0},
		{0x684093, 4, true, -1, 0, 0},
		{0x684097, 4, true, -1, 0, 0},
		{0x68409b, 4, false, -1, 0, 0},
		{0x68409f, 4, false, -1, 0, 0},
		{0x6840a3, 4, false, -1, 0, 0},
		{0x6840a7, 4, false, -1, 0, 0},
		{0x20010081, 4, false, 0, 8, 0x106},
		{0x20010085, 4, true, 0, 8, 0x106},
		{0x20010089, 4, true, 0, 8, 0x106},
		{0x2001008d, 4, true, 0, 8, 0x106},
		{0x20010091, 4, false, 0, 8, 0x106},
		{0x20010095, 4, false, 0, 8, 0x106},
		{0x20010099, 4, false, 0, 8, 0x106},
		{0x2001009d, 4, false, 0, 8, 0x106},
		{0x200100a9, 4, false, 0, 8, 0x106},
		{0x200100ad, 4, true, 0, 8, 0x106},
		{0x200100b1, 4, true, 0, 8, 0x106},
		{0x200100b5, 4, true, 0, 8, 0x106},
		{0x200100b9, 4, false, 0, 8, 0x106},
		{0x200100bd, 4, false, 0, 8, 0x106},
		{0x200100c1, 4, false, 0, 8, 0x106},
		{0x200100c5, 4, false, 0, 8, 0x106},
		{0x20010001, 4, true, 0, 8, 0x106},
		{0x2001003d, 4, true, 0, 8, 0x106},
		{0x20010005, 4, true, 0, 8, 0x106},
		{0x20010075, 4, true, 0, 8, 0x106},
		{0x20010019, 4, false, 0, 8, 0x106},
		{0x20010049, 4, false, 0, 8, 0x106},
		{0x2001007d, 4, false, 0, 8, 0x106},
		{0x200100a5, 4, false, 0, 8, 0x106},
		{0x6840af, 4, false, -1, 0, 0},
		{0x69b9a4, 4, false, -1, 0, 0},
		{0x69b9a0, 4, false, -1, 0, 0},
		{0x683f78, 4, true, -1, 0, 0},
		{0x2002b000, 4, true, -1, 0, 0},
		{0x2002b010, 4, true, -1, 0, 0},
		{0x2002b108, 4, true, -1, 0, 0},
		{0x2002b200, 4, false, -1, 0, 0},
		{0x2002b20c, 4, false, -1, 0, 0},
		{0x2002b214, 1, false, -1, 0, 0},
		{0x2002c5bc, 1, false, -1, 0, 0},
		{0x2002b220, 4, false, -1, 0, 0},
		{0x2002b22c, 4, false, -1, 0, 0},
		{0x2002b234, 1, false, -1, 0, 0},
		{0x2002cbbc, 1, false, -1, 0, 0},
		{0x2002b240, 4, false, -1, 0, 0},
		{0x2002b24c, 4, false, -1, 0, 0},
		{0x2002b254, 1, false, -1, 0, 0},
		{0x2002d1bc, 1, false, -1, 0, 0},
		{0x20029000, 4, false, -1, 0, 0},
		{0x20029004, 4, false, -1, 0, 0},
		{0x20010000, 1, false, 0, 8, 0x106},
		{0x20000b11, 4, true, 0, 8, 0x1862},
		{0x20000b01, 4, false, 0, 8, 0x1862},
	},
	out: []mstFld{
		{0x20029100, 4, false, -1, 0, 0},
		{0x20029104, 4, false, -1, 0, 0},
		{0x20029110, 4, false, -1, 0, 0},
		{0x20029114, 4, false, -1, 0, 0},
		{0x20029120, 4, false, -1, 0, 0},
		{0x20029124, 4, false, -1, 0, 0},
		{0x20029130, 4, false, -1, 0, 0},
		{0x20029134, 4, false, -1, 0, 0},
		{0x20029138, 4, false, -1, 0, 0},
		{0x2002913c, 4, false, -1, 0, 0},
		{0x20029140, 4, false, -1, 0, 0},
		{0x20029150, 4, false, -1, 0, 0},
		{0x20029154, 4, false, -1, 0, 0},
		{0x20029158, 4, false, -1, 0, 0},
		{0x2002915c, 4, false, -1, 0, 0},
		{0x20029160, 4, false, -1, 0, 0},
		{0x20029170, 4, false, -1, 0, 0},
		{0x20029174, 4, false, -1, 0, 0},
		{0x20029178, 4, false, -1, 0, 0},
		{0x2002917c, 4, false, -1, 0, 0},
		{0x20029180, 4, false, -1, 0, 0},
		{0x20029184, 4, false, -1, 0, 0},
		{0x20029190, 4, false, -1, 0, 0},
		{0x20029194, 4, false, -1, 0, 0},
		{0x20029198, 4, false, -1, 0, 0},
		{0x2002919c, 4, false, -1, 0, 0},
		{0x200291a0, 4, false, -1, 0, 0},
		{0x200291a4, 4, false, -1, 0, 0},
		{0x200291a8, 4, false, -1, 0, 0},
		{0x200291ac, 4, false, -1, 0, 0},
		{0x200291b0, 4, false, -1, 0, 0},
		{0x200291b4, 4, false, -1, 0, 0},
		{0x20010001, 4, false, -1, 0, 0},
		{0x20010005, 4, false, -1, 0, 0},
		{0x20010009, 4, false, -1, 0, 0},
		{0x20010015, 4, false, -1, 0, 0},
		{0x20010019, 4, false, -1, 0, 0},
		{0x200100d0, 1, false, -1, 0, 0},
		{0x20010075, 4, false, -1, 0, 0},
		{0x20010079, 4, false, -1, 0, 0},
		{0x2001007d, 4, false, -1, 0, 0},
		{0x20010081, 4, false, -1, 0, 0},
		{0x20010085, 4, false, -1, 0, 0},
		{0x20010089, 4, false, -1, 0, 0},
		{0x2001008d, 4, false, -1, 0, 0},
		{0x20010091, 4, false, -1, 0, 0},
		{0x20010095, 4, false, -1, 0, 0},
		{0x20010099, 4, false, -1, 0, 0},
		{0x2001009d, 4, false, -1, 0, 0},
		{0x2001001d, 4, false, -1, 0, 0},
		{0x20010021, 4, false, -1, 0, 0},
		{0x20010025, 4, false, -1, 0, 0},
		{0x20010029, 4, false, -1, 0, 0},
		{0x2001002d, 4, false, -1, 0, 0},
		{0x20010031, 4, false, -1, 0, 0},
		{0x20010035, 4, false, -1, 0, 0},
		{0x20010039, 4, false, -1, 0, 0},
		{0x2001004d, 4, false, -1, 0, 0},
		{0x20010051, 4, false, -1, 0, 0},
		{0x20010055, 4, false, -1, 0, 0},
		{0x20010059, 4, false, -1, 0, 0},
		{0x2001005d, 4, false, -1, 0, 0},
		{0x20010061, 4, false, -1, 0, 0},
		{0x20010065, 4, false, -1, 0, 0},
		{0x20010069, 4, false, -1, 0, 0},
		{0x20010107, 4, false, -1, 0, 0},
		{0x2001010b, 4, false, -1, 0, 0},
		{0x2001010f, 4, false, -1, 0, 0},
		{0x2001011b, 4, false, -1, 0, 0},
		{0x2001011f, 4, false, -1, 0, 0},
		{0x200101d6, 1, false, -1, 0, 0},
		{0x2001017b, 4, false, -1, 0, 0},
		{0x2001017f, 4, false, -1, 0, 0},
		{0x20010183, 4, false, -1, 0, 0},
		{0x20010187, 4, false, -1, 0, 0},
		{0x2001018b, 4, false, -1, 0, 0},
		{0x2001018f, 4, false, -1, 0, 0},
		{0x20010193, 4, false, -1, 0, 0},
		{0x20010197, 4, false, -1, 0, 0},
		{0x2001019b, 4, false, -1, 0, 0},
		{0x2001019f, 4, false, -1, 0, 0},
		{0x200101a3, 4, false, -1, 0, 0},
		{0x20010123, 4, false, -1, 0, 0},
		{0x20010127, 4, false, -1, 0, 0},
		{0x2001012b, 4, false, -1, 0, 0},
		{0x2001012f, 4, false, -1, 0, 0},
		{0x20010133, 4, false, -1, 0, 0},
		{0x20010137, 4, false, -1, 0, 0},
		{0x2001013b, 4, false, -1, 0, 0},
		{0x2001013f, 4, false, -1, 0, 0},
		{0x20010153, 4, false, -1, 0, 0},
		{0x20010157, 4, false, -1, 0, 0},
		{0x2001015b, 4, false, -1, 0, 0},
		{0x2001015f, 4, false, -1, 0, 0},
		{0x20010163, 4, false, -1, 0, 0},
		{0x20010167, 4, false, -1, 0, 0},
		{0x2001016b, 4, false, -1, 0, 0},
		{0x2001016f, 4, false, -1, 0, 0},
		{0x2001020d, 4, false, -1, 0, 0},
		{0x20010211, 4, false, -1, 0, 0},
		{0x20010215, 4, false, -1, 0, 0},
		{0x20010221, 4, false, -1, 0, 0},
		{0x20010225, 4, false, -1, 0, 0},
		{0x200102dc, 1, false, -1, 0, 0},
		{0x20010281, 4, false, -1, 0, 0},
		{0x20010285, 4, false, -1, 0, 0},
		{0x20010289, 4, false, -1, 0, 0},
		{0x2001028d, 4, false, -1, 0, 0},
		{0x20010291, 4, false, -1, 0, 0},
		{0x20010295, 4, false, -1, 0, 0},
		{0x20010299, 4, false, -1, 0, 0},
		{0x2001029d, 4, false, -1, 0, 0},
		{0x200102a1, 4, false, -1, 0, 0},
		{0x200102a5, 4, false, -1, 0, 0},
		{0x200102a9, 4, false, -1, 0, 0},
		{0x20010229, 4, false, -1, 0, 0},
		{0x2001022d, 4, false, -1, 0, 0},
		{0x20010231, 4, false, -1, 0, 0},
		{0x20010235, 4, false, -1, 0, 0},
		{0x20010239, 4, false, -1, 0, 0},
		{0x2001023d, 4, false, -1, 0, 0},
		{0x20010241, 4, false, -1, 0, 0},
		{0x20010245, 4, false, -1, 0, 0},
		{0x20010259, 4, false, -1, 0, 0},
		{0x2001025d, 4, false, -1, 0, 0},
		{0x20010261, 4, false, -1, 0, 0},
		{0x20010265, 4, false, -1, 0, 0},
		{0x20010269, 4, false, -1, 0, 0},
		{0x2001026d, 4, false, -1, 0, 0},
		{0x20010271, 4, false, -1, 0, 0},
		{0x20010275, 4, false, -1, 0, 0},
		{0x20010313, 4, false, -1, 0, 0},
		{0x20010317, 4, false, -1, 0, 0},
		{0x2001031b, 4, false, -1, 0, 0},
		{0x20010327, 4, false, -1, 0, 0},
		{0x2001032b, 4, false, -1, 0, 0},
		{0x200103e2, 1, false, -1, 0, 0},
		{0x20010387, 4, false, -1, 0, 0},
		{0x2001038b, 4, false, -1, 0, 0},
		{0x2001038f, 4, false, -1, 0, 0},
		{0x20010393, 4, false, -1, 0, 0},
		{0x20010397, 4, false, -1, 0, 0},
		{0x2001039b, 4, false, -1, 0, 0},
		{0x2001039f, 4, false, -1, 0, 0},
		{0x200103a3, 4, false, -1, 0, 0},
		{0x200103a7, 4, false, -1, 0, 0},
		{0x200103ab, 4, false, -1, 0, 0},
		{0x200103af, 4, false, -1, 0, 0},
		{0x2001032f, 4, false, -1, 0, 0},
		{0x20010333, 4, false, -1, 0, 0},
		{0x20010337, 4, false, -1, 0, 0},
		{0x2001033b, 4, false, -1, 0, 0},
		{0x2001033f, 4, false, -1, 0, 0},
		{0x20010343, 4, false, -1, 0, 0},
		{0x20010347, 4, false, -1, 0, 0},
		{0x2001034b, 4, false, -1, 0, 0},
		{0x2001035f, 4, false, -1, 0, 0},
		{0x20010363, 4, false, -1, 0, 0},
		{0x20010367, 4, false, -1, 0, 0},
		{0x2001036b, 4, false, -1, 0, 0},
		{0x2001036f, 4, false, -1, 0, 0},
		{0x20010373, 4, false, -1, 0, 0},
		{0x20010377, 4, false, -1, 0, 0},
		{0x2001037b, 4, false, -1, 0, 0},
		{0x20010419, 4, false, -1, 0, 0},
		{0x2001041d, 4, false, -1, 0, 0},
		{0x20010421, 4, false, -1, 0, 0},
		{0x2001042d, 4, false, -1, 0, 0},
		{0x20010431, 4, false, -1, 0, 0},
		{0x200104e8, 1, false, -1, 0, 0},
		{0x2001048d, 4, false, -1, 0, 0},
		{0x20010491, 4, false, -1, 0, 0},
		{0x20010495, 4, false, -1, 0, 0},
		{0x20010499, 4, false, -1, 0, 0},
		{0x2001049d, 4, false, -1, 0, 0},
		{0x200104a1, 4, false, -1, 0, 0},
		{0x200104a5, 4, false, -1, 0, 0},
		{0x200104a9, 4, false, -1, 0, 0},
		{0x200104ad, 4, false, -1, 0, 0},
		{0x200104b1, 4, false, -1, 0, 0},
		{0x200104b5, 4, false, -1, 0, 0},
		{0x20010435, 4, false, -1, 0, 0},
		{0x20010439, 4, false, -1, 0, 0},
		{0x2001043d, 4, false, -1, 0, 0},
		{0x20010441, 4, false, -1, 0, 0},
		{0x20010445, 4, false, -1, 0, 0},
		{0x20010449, 4, false, -1, 0, 0},
		{0x2001044d, 4, false, -1, 0, 0},
		{0x20010451, 4, false, -1, 0, 0},
		{0x20010465, 4, false, -1, 0, 0},
		{0x20010469, 4, false, -1, 0, 0},
		{0x2001046d, 4, false, -1, 0, 0},
		{0x20010471, 4, false, -1, 0, 0},
		{0x20010475, 4, false, -1, 0, 0},
		{0x20010479, 4, false, -1, 0, 0},
		{0x2001047d, 4, false, -1, 0, 0},
		{0x20010481, 4, false, -1, 0, 0},
		{0x2001051f, 4, false, -1, 0, 0},
		{0x20010523, 4, false, -1, 0, 0},
		{0x20010527, 4, false, -1, 0, 0},
		{0x20010533, 4, false, -1, 0, 0},
		{0x20010537, 4, false, -1, 0, 0},
		{0x200105ee, 1, false, -1, 0, 0},
		{0x20010593, 4, false, -1, 0, 0},
		{0x20010597, 4, false, -1, 0, 0},
		{0x2001059b, 4, false, -1, 0, 0},
		{0x2001059f, 4, false, -1, 0, 0},
		{0x200105a3, 4, false, -1, 0, 0},
		{0x200105a7, 4, false, -1, 0, 0},
		{0x200105ab, 4, false, -1, 0, 0},
		{0x200105af, 4, false, -1, 0, 0},
		{0x200105b3, 4, false, -1, 0, 0},
		{0x200105b7, 4, false, -1, 0, 0},
		{0x200105bb, 4, false, -1, 0, 0},
		{0x2001053b, 4, false, -1, 0, 0},
		{0x2001053f, 4, false, -1, 0, 0},
		{0x20010543, 4, false, -1, 0, 0},
		{0x20010547, 4, false, -1, 0, 0},
		{0x2001054b, 4, false, -1, 0, 0},
		{0x2001054f, 4, false, -1, 0, 0},
		{0x20010553, 4, false, -1, 0, 0},
		{0x20010557, 4, false, -1, 0, 0},
		{0x2001056b, 4, false, -1, 0, 0},
		{0x2001056f, 4, false, -1, 0, 0},
		{0x20010573, 4, false, -1, 0, 0},
		{0x20010577, 4, false, -1, 0, 0},
		{0x2001057b, 4, false, -1, 0, 0},
		{0x2001057f, 4, false, -1, 0, 0},
		{0x20010583, 4, false, -1, 0, 0},
		{0x20010587, 4, false, -1, 0, 0},
		{0x20010625, 4, false, -1, 0, 0},
		{0x20010629, 4, false, -1, 0, 0},
		{0x2001062d, 4, false, -1, 0, 0},
		{0x20010639, 4, false, -1, 0, 0},
		{0x2001063d, 4, false, -1, 0, 0},
		{0x200106f4, 1, false, -1, 0, 0},
		{0x20010699, 4, false, -1, 0, 0},
		{0x2001069d, 4, false, -1, 0, 0},
		{0x200106a1, 4, false, -1, 0, 0},
		{0x200106a5, 4, false, -1, 0, 0},
		{0x200106a9, 4, false, -1, 0, 0},
		{0x200106ad, 4, false, -1, 0, 0},
		{0x200106b1, 4, false, -1, 0, 0},
		{0x200106b5, 4, false, -1, 0, 0},
		{0x200106b9, 4, false, -1, 0, 0},
		{0x200106bd, 4, false, -1, 0, 0},
		{0x200106c1, 4, false, -1, 0, 0},
		{0x20010641, 4, false, -1, 0, 0},
		{0x20010645, 4, false, -1, 0, 0},
		{0x20010649, 4, false, -1, 0, 0},
		{0x2001064d, 4, false, -1, 0, 0},
		{0x20010651, 4, false, -1, 0, 0},
		{0x20010655, 4, false, -1, 0, 0},
		{0x20010659, 4, false, -1, 0, 0},
		{0x2001065d, 4, false, -1, 0, 0},
		{0x20010671, 4, false, -1, 0, 0},
		{0x20010675, 4, false, -1, 0, 0},
		{0x20010679, 4, false, -1, 0, 0},
		{0x2001067d, 4, false, -1, 0, 0},
		{0x20010681, 4, false, -1, 0, 0},
		{0x20010685, 4, false, -1, 0, 0},
		{0x20010689, 4, false, -1, 0, 0},
		{0x2001068d, 4, false, -1, 0, 0},
		{0x2001072b, 4, false, -1, 0, 0},
		{0x2001072f, 4, false, -1, 0, 0},
		{0x20010733, 4, false, -1, 0, 0},
		{0x2001073f, 4, false, -1, 0, 0},
		{0x20010743, 4, false, -1, 0, 0},
		{0x200107fa, 1, false, -1, 0, 0},
		{0x2001079f, 4, false, -1, 0, 0},
		{0x200107a3, 4, false, -1, 0, 0},
		{0x200107a7, 4, false, -1, 0, 0},
		{0x200107ab, 4, false, -1, 0, 0},
		{0x200107af, 4, false, -1, 0, 0},
		{0x200107b3, 4, false, -1, 0, 0},
		{0x200107b7, 4, false, -1, 0, 0},
		{0x200107bb, 4, false, -1, 0, 0},
		{0x200107bf, 4, false, -1, 0, 0},
		{0x200107c3, 4, false, -1, 0, 0},
		{0x200107c7, 4, false, -1, 0, 0},
		{0x20010747, 4, false, -1, 0, 0},
		{0x2001074b, 4, false, -1, 0, 0},
		{0x2001074f, 4, false, -1, 0, 0},
		{0x20010753, 4, false, -1, 0, 0},
		{0x20010757, 4, false, -1, 0, 0},
		{0x2001075b, 4, false, -1, 0, 0},
		{0x2001075f, 4, false, -1, 0, 0},
		{0x20010763, 4, false, -1, 0, 0},
		{0x20010777, 4, false, -1, 0, 0},
		{0x2001077b, 4, false, -1, 0, 0},
		{0x2001077f, 4, false, -1, 0, 0},
		{0x20010783, 4, false, -1, 0, 0},
		{0x20010787, 4, false, -1, 0, 0},
		{0x2001078b, 4, false, -1, 0, 0},
		{0x2001078f, 4, false, -1, 0, 0},
		{0x20010793, 4, false, -1, 0, 0},
		{0x6840af, 4, false, -1, 0, 0},
		{0x20023c5a, 1, false, -1, 0, 0},
		{0x2002c5bc, 1, false, -1, 0, 0},
		{0x2002cbbc, 1, false, -1, 0, 0},
		{0x2002d1bc, 1, false, -1, 0, 0},
	},
}

var scpIn = []string{"sel", "panelChar", "spell", "power", "mode10c", "lockC5b", "submitting", "dialogUp", "partyAllowed", "spellKind", "battlePtr", "listPtr", "phase", "battleAction", "battleFlag", "combat", "specSpell", "specItem", "f9b", "castFlag", "dlg1e90", "dlg1e94", "d0.kind", "d0.char", "d0.mon", "d0.grp", "d0.w10", "d0.w14", "d0.w18", "d0.list", "d1.kind", "d1.char", "d1.mon", "d1.grp", "d1.w10", "d1.w14", "d1.w18", "d1.list", "dg.kind", "dg.char", "dg.mon", "dg.grp", "dg.w10", "dg.w14", "dg.w18", "dg.list", "d3.kind", "d3.char", "d3.mon", "d3.grp", "d3.w10", "d3.w14", "d3.w18", "d3.list", "d4.kind", "d4.char", "d4.mon", "d4.grp", "d4.w10", "d4.w14", "d4.w18", "d4.list", "act0", "act1", "b0", "b3", "item0", "item1", "item3", "item4", "itemG", "hand1", "hand0", "itemTypes", "invA", "invB", "listCount", "m0.id", "m0.scene", "m0.live", "m0.marks", "m1.id", "m1.scene", "m1.live", "m1.marks", "m2.id", "m2.scene", "m2.live", "m2.marks", "r536190", "r53c270", "selActive", "selHP", "selStatus"}

var scpOut = []string{"s536190_calls", "s536190_a1_0", "s53c270_calls", "s53c270_a1_0", "s4ea5c0_calls", "s4ea5c0_a1_0", "s4e8000_calls", "s4e8000_a1_0", "s4e8000_a2_0", "s4e8000_a3_0", "s4e8000_a5_0", "s569a50_calls", "s569a50_a1_0", "s569a50_a2_0", "s569a50_a3_0", "s569a50_a4_0", "s52e690_calls", "s52e690_a1_0", "s52e690_a2_0", "s52e690_a3_0", "s52e690_a4_0", "s52e690_a5_0", "s4c5eb0_calls", "s4c5eb0_a1_0", "s4c5eb0_a1_1", "s4c5eb0_a1_2", "s4c5eb0_a2_0", "s4c5eb0_a2_1", "s4c5eb0_a2_2", "s4c5eb0_a3_0", "s4c5eb0_a3_1", "s4c5eb0_a3_2", "q0.act", "q0.spell", "q0.x9", "q0.pw", "q0.it", "q0.cact", "q0.sp75", "q0.pl79", "q0.x7d", "q0.t81_0", "q0.t81_1", "q0.t81_2", "q0.t81_3", "q0.t81_4", "q0.t81_5", "q0.t81_6", "q0.t81_7", "q0.t1d_0", "q0.t1d_1", "q0.t1d_2", "q0.t1d_3", "q0.t1d_4", "q0.t1d_5", "q0.t1d_6", "q0.t1d_7", "q0.t4d_0", "q0.t4d_1", "q0.t4d_2", "q0.t4d_3", "q0.t4d_4", "q0.t4d_5", "q0.t4d_6", "q0.t4d_7", "q1.act", "q1.spell", "q1.x9", "q1.pw", "q1.it", "q1.cact", "q1.sp75", "q1.pl79", "q1.x7d", "q1.t81_0", "q1.t81_1", "q1.t81_2", "q1.t81_3", "q1.t81_4", "q1.t81_5", "q1.t81_6", "q1.t81_7", "q1.t1d_0", "q1.t1d_1", "q1.t1d_2", "q1.t1d_3", "q1.t1d_4", "q1.t1d_5", "q1.t1d_6", "q1.t1d_7", "q1.t4d_0", "q1.t4d_1", "q1.t4d_2", "q1.t4d_3", "q1.t4d_4", "q1.t4d_5", "q1.t4d_6", "q1.t4d_7", "q2.act", "q2.spell", "q2.x9", "q2.pw", "q2.it", "q2.cact", "q2.sp75", "q2.pl79", "q2.x7d", "q2.t81_0", "q2.t81_1", "q2.t81_2", "q2.t81_3", "q2.t81_4", "q2.t81_5", "q2.t81_6", "q2.t81_7", "q2.t1d_0", "q2.t1d_1", "q2.t1d_2", "q2.t1d_3", "q2.t1d_4", "q2.t1d_5", "q2.t1d_6", "q2.t1d_7", "q2.t4d_0", "q2.t4d_1", "q2.t4d_2", "q2.t4d_3", "q2.t4d_4", "q2.t4d_5", "q2.t4d_6", "q2.t4d_7", "q3.act", "q3.spell", "q3.x9", "q3.pw", "q3.it", "q3.cact", "q3.sp75", "q3.pl79", "q3.x7d", "q3.t81_0", "q3.t81_1", "q3.t81_2", "q3.t81_3", "q3.t81_4", "q3.t81_5", "q3.t81_6", "q3.t81_7", "q3.t1d_0", "q3.t1d_1", "q3.t1d_2", "q3.t1d_3", "q3.t1d_4", "q3.t1d_5", "q3.t1d_6", "q3.t1d_7", "q3.t4d_0", "q3.t4d_1", "q3.t4d_2", "q3.t4d_3", "q3.t4d_4", "q3.t4d_5", "q3.t4d_6", "q3.t4d_7", "q4.act", "q4.spell", "q4.x9", "q4.pw", "q4.it", "q4.cact", "q4.sp75", "q4.pl79", "q4.x7d", "q4.t81_0", "q4.t81_1", "q4.t81_2", "q4.t81_3", "q4.t81_4", "q4.t81_5", "q4.t81_6", "q4.t81_7", "q4.t1d_0", "q4.t1d_1", "q4.t1d_2", "q4.t1d_3", "q4.t1d_4", "q4.t1d_5", "q4.t1d_6", "q4.t1d_7", "q4.t4d_0", "q4.t4d_1", "q4.t4d_2", "q4.t4d_3", "q4.t4d_4", "q4.t4d_5", "q4.t4d_6", "q4.t4d_7", "q5.act", "q5.spell", "q5.x9", "q5.pw", "q5.it", "q5.cact", "q5.sp75", "q5.pl79", "q5.x7d", "q5.t81_0", "q5.t81_1", "q5.t81_2", "q5.t81_3", "q5.t81_4", "q5.t81_5", "q5.t81_6", "q5.t81_7", "q5.t1d_0", "q5.t1d_1", "q5.t1d_2", "q5.t1d_3", "q5.t1d_4", "q5.t1d_5", "q5.t1d_6", "q5.t1d_7", "q5.t4d_0", "q5.t4d_1", "q5.t4d_2", "q5.t4d_3", "q5.t4d_4", "q5.t4d_5", "q5.t4d_6", "q5.t4d_7", "q6.act", "q6.spell", "q6.x9", "q6.pw", "q6.it", "q6.cact", "q6.sp75", "q6.pl79", "q6.x7d", "q6.t81_0", "q6.t81_1", "q6.t81_2", "q6.t81_3", "q6.t81_4", "q6.t81_5", "q6.t81_6", "q6.t81_7", "q6.t1d_0", "q6.t1d_1", "q6.t1d_2", "q6.t1d_3", "q6.t1d_4", "q6.t1d_5", "q6.t1d_6", "q6.t1d_7", "q6.t4d_0", "q6.t4d_1", "q6.t4d_2", "q6.t4d_3", "q6.t4d_4", "q6.t4d_5", "q6.t4d_6", "q6.t4d_7", "q7.act", "q7.spell", "q7.x9", "q7.pw", "q7.it", "q7.cact", "q7.sp75", "q7.pl79", "q7.x7d", "q7.t81_0", "q7.t81_1", "q7.t81_2", "q7.t81_3", "q7.t81_4", "q7.t81_5", "q7.t81_6", "q7.t81_7", "q7.t1d_0", "q7.t1d_1", "q7.t1d_2", "q7.t1d_3", "q7.t1d_4", "q7.t1d_5", "q7.t1d_6", "q7.t1d_7", "q7.t4d_0", "q7.t4d_1", "q7.t4d_2", "q7.t4d_3", "q7.t4d_4", "q7.t4d_5", "q7.t4d_6", "q7.t4d_7", "itemG", "submitting", "m0.marks", "m1.marks", "m2.marks"}

var scpStubs = map[uint32]*mcaStub{
	0x536190: {ret: 0x20029000, retN: 0, rec: [][4]uint32{{1, 4, 0x20029104, 1}}, cnt: 0x20029100, wr: []mcaWrite{}},
	0x53c270: {ret: 0x20029004, retN: 0, rec: [][4]uint32{{1, 4, 0x20029114, 1}}, cnt: 0x20029110, wr: []mcaWrite{}},
	0x4ea5c0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20029124, 1}}, cnt: 0x20029120, wr: []mcaWrite{}},
	0x4e8000: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20029134, 1}, {2, 4, 0x20029138, 1}, {3, 4, 0x2002913c, 1}, {5, 4, 0x20029140, 1}}, cnt: 0x20029130, wr: []mcaWrite{}},
	0x569a50: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20029154, 1}, {2, 4, 0x20029158, 1}, {3, 4, 0x2002915c, 1}, {4, 4, 0x20029160, 1}}, cnt: 0x20029150, wr: []mcaWrite{}},
	0x52e690: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20029174, 1}, {2, 4, 0x20029178, 1}, {3, 4, 0x2002917c, 1}, {4, 4, 0x20029180, 1}, {5, 4, 0x20029184, 1}}, cnt: 0x20029170, wr: []mcaWrite{}},
	0x4c5eb0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20029194, 3}, {2, 4, 0x200291a0, 3}, {3, 4, 0x200291ac, 3}}, cnt: 0x20029190, wr: []mcaWrite{}},
	0x52e5c0: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20029300, wr: []mcaWrite{}},
	0x562a50: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20029304, wr: []mcaWrite{}},
	0x53a320: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20029308, wr: []mcaWrite{}},
	0x4f23d0: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x2002930c, wr: []mcaWrite{}},
	0x55ee70: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20029310, wr: []mcaWrite{}},
	0x58f6b0: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20029314, wr: []mcaWrite{}},
	0x562580: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20029318, wr: []mcaWrite{}},
	0x5b1c80: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x2002931c, wr: []mcaWrite{}},
	0x56aab0: {ret: 0x0, retN: 0, rec: [][4]uint32{}, cnt: 0x20029320, wr: []mcaWrite{}},
}
