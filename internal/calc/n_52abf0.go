package calc

import "fmt"

// 0x52abf0 character_death_apply (build/wiz8.asm:405135): party member `who`
// dies. Its presence record (*0x685178, stride 0x106) must be set. Bumps the
// death count (character +0x9fd), calls 0x523330(who, slot, 0) for every state
// slot 0..0x11 except 10 whose word (+0xa01+4*slot) is non-zero, zeroes +0xb11
// and +0xb1d, resets the two action records (+0x1d, +0x4d). In combat (0x683f94)
// it queues `who` on the death list (0x4ecdd0: dword list at *0x6836a8+0x8e8,
// count at +0x908), then 0x52f110 picks one other present, living member with
// state < 0x12 (starting at member 2 when who < 2; R(n) when several qualify) and
// calls the excluded 0x52e690 for it, stamping the returned event with 0xbb8 and
// GetTickCount. In combat it also clears the targeting selection (0x6836a8+0x7b0/
// +0x7b4), sets the member's slot +1 to -1, the context record's +0x18 to 0 and
// byte +0x4c to 1, and runs 0x4e82e0 (= the 0x4e7590 per-member reset: pending
// claims, record +0x4d, presentation refresh, 0x4e8000(who, +0x6d, +0x71, 0, 0)).
// At the end the member's slot +0xfa object id (-1 = none) is looked up by
// 0x50b800 and the found object's word +4 is set to 1.
//
// Emu address-stubs 0x523330, 0x4e8000 (D4 formulas) and 0x52e690, 0x4c5eb0
// (excluded): arguments of the last call and the call count are outputs
// (r523, r52e, stub, mk). F also runs the real 0x523330 on the node's cells (its
// slot output is fed back) and the real cgW.setAction (the four-slot fixture of
// n_520d10.go applies there). 0x408860 returns at once (sound flag = exe initial
// value 0), 0x2b6 is a mis-decoded address (notes/re/iface/52abf0.tsv).
//
// Constants (Wiz8.exe .data initial values): 0x5ee5d0=0x12, 0x5ee5d4=0x13,
// 0x5ee5d8=0x14 (event kind for who>=2 / who>=2 with +0x75 set / who<2),
// 0x5ed8cc=1, 0x5ed914=0x7f (52e690 arguments 4 and 5). GetTickCount is the
// case's fixed tick (4660, as in 0x528ff0).
//
// Fixture caps (calc panics past them): who 0..7; death list count 0..7 (eight
// dwords at +0x8e8); pending list as 0x4e7590 (four ids, two monsters, the one
// UI object); object table 0x689f94 of at most four entries, ids below -1
// unsupported; its pointers, the entry pointers and the 0x52e690 event pointer
// are the fixture objects or 0.
const (
	cdaR523   = paiStub + 4096
	cdaR52e   = paiStub + 4224
	cdaEvRet  = paiStub + 4352
	cdaTbl    = paiStub + 4608
	cdaArr    = paiStub + 4736
	cdaEnt    = paiStub + 5120 // four entries, stride 256
	cdaEv     = paiStub + 6400
	cdaTick   = 4660
	cdaKindA  = 0x12 // exe 0x5ee5d0
	cdaKindB  = 0x13 // exe 0x5ee5d4
	cdaKindLo = 0x14 // exe 0x5ee5d8
	cdaArg4   = 1    // exe 0x5ed8cc
	cdaArg5   = 0x7f // exe 0x5ed914
)

type cdaCell struct {
	name   string
	at     int64
	size   int // negative = signed
	stride int64
	idx    bool // address = at + stride*who
}

func (c cdaCell) addr(who int64) int64 {
	if c.idx {
		return c.at + c.stride*who
	}
	return c.at
}

var cdaIn, cdaOut = cdaLayout()

func cdaLayout() (in, out []cdaCell) {
	add := func(n string, at int64, s int) { in = append(in, cdaCell{n, at, s, 0, false}) }
	idx := func(n string, at int64, s int, stride int64) { in = append(in, cdaCell{n, at, s, stride, true}) }
	add("f94", 0x683f94, 1)
	add("f95", 0x683f95, 1)
	add("f96", 0x683f96, 1)
	add("sel", 0x68518d, -4)
	add("mode", 0x68ec78, 4)
	add("uip", 0x68edcc, 4)
	add("ui.f4", paiUI+0xf4, 4)
	add("ui.2f0", paiUI+0x2f0, -4)
	add("ui.2f8", paiUI+0x2f8, 1)
	add("g69c.f8", paiG69c+0xf8, 4)
	add("mon.n", paiMon+8, -4)
	for k := int64(0); k < 2; k++ {
		add(fmt.Sprintf("mon%d.id", k), paiMon+paiMonEntry[k], -4)
		add(fmt.Sprintf("mon%d.mask", k), paiMon+paiMonInner[k]+0x5bc, 1)
	}
	add("tblp", 0x689f94, 4)
	add("tbl.n", cdaTbl+4, -4)
	for k := int64(0); k < 4; k++ {
		add(fmt.Sprintf("arr%d", k), cdaArr+4*k, 4)
	}
	for k := int64(0); k < 4; k++ {
		add(fmt.Sprintf("ent%d.w4", k), cdaEnt+256*k+4, 2)
		add(fmt.Sprintf("ent%d.f7", k), cdaEnt+256*k+0xc7, 1)
	}
	add("evp", cdaEvRet, 4)
	for _, o := range []int64{0x2c, 0x30, 0x34, 0x38} {
		add(fmt.Sprintf("ev.%x", o), cdaEv+o, 4)
	}
	add("d.7b0", paiCtx+0x7b0, 4)
	add("d.7b4", paiCtx+0x7b4, -4)
	for i := int64(0); i < 8; i++ {
		add(fmt.Sprintf("d.l%d", i), paiCtx+0x8e8+4*i, 4)
	}
	add("d.cnt", paiCtx+0x908, 4)
	for j := int64(0); j < 8; j++ {
		add(fmt.Sprintf("p%d", j), paiSubs+0x106*j, 1)
		add(fmt.Sprintf("h%d", j), paiChars+0x1862*j+0xb11, 4)
		add(fmt.Sprintf("t%d", j), paiChars+0x1862*j+0xb01, 4)
		add(fmt.Sprintf("pc%d", j), 0x683794+0x118*j, -4)
	}
	idx("s.f1", paiSubs+1, -4, 0x106)
	for i := int64(0); i < 8; i++ {
		idx(fmt.Sprintf("s.r0.%d", i), paiSubs+0x1d+4*i, -4, 0x106)
	}
	for i := int64(0); i < 8; i++ {
		idx(fmt.Sprintf("s.r1.%d", i), paiSubs+0x4d+4*i, -4, 0x106)
	}
	idx("s.a6d", paiSubs+0x6d, -4, 0x106)
	idx("s.a71", paiSubs+0x71, -4, 0x106)
	idx("s.fa", paiSubs+0xfa, -4, 0x106)
	idx("c.9fd", paiChars+0x9fd, 4, 0x1862)
	idx("c.b1d", paiChars+0xb1d, 4, 0x1862)
	idx("c.75", paiChars+0x75, 4, 0x1862)
	for i := int64(0); i < 18; i++ {
		idx(fmt.Sprintf("c.s%d", i), paiChars+0xa01+4*i, 4, 0x1862)
	}
	idx("d.18", paiCtx+0x18, 4, 0xd4)
	idx("d.4c", paiCtx+0x4c, 1, 0xd4)
	for i := int64(0); i < 4; i++ {
		idx(fmt.Sprintf("pi.%d", i), paiPend+4*i, -4, 16)
	}
	for _, c := range in {
		if len(c.name) < 3 || c.name[:3] != "c.s" {
			out = append(out, c)
		}
	}
	rec := func(p string, base int64, n int, cnt int64) {
		for i := 0; i < n; i++ {
			out = append(out, cdaCell{fmt.Sprintf("%s.a%d", p, i+1), base + 4*int64(i), -4, 0, false})
		}
		out = append(out, cdaCell{p + ".n", cnt, 4, 0, false})
	}
	rec("r523", cdaR523, 3, cdaR523+16)
	rec("r52e", cdaR52e, 5, cdaR52e+32)
	rec("stub", paiStub, 5, paiStub+32)
	rec("mk", paiStub+40, 3, paiStub+64)
	return
}

func init() {
	names := []string{"who"}
	for _, c := range cdaIn {
		names = append(names, c.name)
	}
	var outs []string
	for _, c := range cdaOut {
		outs = append(outs, c.name)
	}
	Nodes["0052abf0"] = Node{Addr: "0052abf0", In: names, Out: outs,
		F: characterDeathApplyNode, Emu: characterDeathApplyEmu}
}

// characterDeathApplyNode runs the real 0x523330 and 0x4e8000 for each call.
func characterDeathApplyNode(in []int64, r Rand) []int64 { return cdaExec(in, r, true) }

// characterDeathApplyEmu replays the case: the four callees are address-stubbed.
func characterDeathApplyEmu(in []int64, r Rand) []int64 { return cdaExec(in, r, false) }

func cdaExec(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(cdaIn)+1 {
		panic("character_death_apply: wrong input count")
	}
	who := in[0]
	if who < 0 || who >= 8 {
		panic("character_death_apply: member outside the eight-slot fixture")
	}
	w := &cgW{m: map[int64]byte{}}
	w.p32(0x685174, paiChars)
	w.p32(0x685178, paiSubs)
	w.p32(0x6836a8, paiCtx)
	w.p32(0x69c0f4, paiG69c)
	w.p32(0x683fad, paiMon)
	w.p32(paiMon, paiMon+256)
	w.p32(paiMon+256, paiMon+paiMonEntry[0])
	w.p32(paiMon+260, paiMon+paiMonEntry[1])
	for k := range paiMonEntry {
		w.p32(paiMon+paiMonEntry[k]+0xc, paiMon+paiMonRec[k])
		w.p32(paiMon+paiMonRec[k]+0x1d8, paiMon+paiMonInner[k])
	}
	for x := int64(0); x < 8; x++ {
		w.p32(0x68379c+0x118*x, paiPend+16*x)
	}
	w.p32(cdaTbl+0xc, cdaArr)
	for i, c := range cdaIn {
		s := c.size
		if s < 0 {
			s = -s
		}
		w.wr(c.addr(who), s, in[i+1])
	}
	if n := w.i32(paiMon + 8); n < 0 || n > 2 {
		panic("character_death_apply: monster list outside the two-monster fixture")
	}
	if u := w.uip(); u != 0 && u != paiUI {
		panic("character_death_apply: UI object pointer outside the one-object fixture")
	}
	S, C, D := w.u32(0x685178)+0x106*who, w.u32(0x685174)+0x1862*who, w.u32(0x6836a8)+0xd4*who
	if w.u8(S) == 0 {
		panic("character_death_apply: absent party member asserts at 0x52ac27")
	}
	w.p32(C+0x9fd, w.u32(C+0x9fd)+1)
	for i := int64(0); i < 0x12; i++ {
		if i == 10 || w.u32(C+0xa01+4*i) == 0 {
			continue
		}
		cdaStub(w, cdaR523, cdaR523+16, who, i, 0)
		if real {
			cdaRemove(w, who, i, r)
		}
	}
	w.p32(C+0xb11, 0)
	w.p32(C+0xb1d, 0)
	w.putRec(S+0x1d, cgRec{0, -1, -1, -1}) // 0x536170
	w.putRec(S+0x4d, cgRec{0, -1, -1, -1})
	combat := w.f94()
	if combat { // 0x4ecdd0
		D0 := w.u32(0x6836a8)
		n := w.u32(D0 + 0x908)
		if n > 7 {
			panic("character_death_apply: death list outside its eight entries")
		}
		w.p32(D0+0x8e8+4*n, who)
		w.p32(D0+0x908, n+1)
	}
	cdaEvent(w, who, r) // 0x52f110
	if combat {
		D0 := w.u32(0x6836a8)
		if w.i32(D0+0x7b4) == who {
			w.p32(D0+0x7b0, 0)
			w.p32(D0+0x7b4, -1)
		}
		w.p32(S+1, -1)
		w.p32(D+0x18, 0)
		w.p8(D+0x4c, 1)
		cdaCancel(w, who, real) // 0x4e82e0
	}
	if id := w.i32(S + 0xfa); id != -1 {
		if e := cdaObject(w, id); e != 0 { // 0x50b800
			w.wr(e+4, 2, 1)
		}
	}
	out := make([]int64, len(cdaOut))
	for i, c := range cdaOut {
		if c.size < 0 {
			out[i] = w.rs(c.addr(who), -c.size)
		} else {
			out[i] = w.rd(c.addr(who), c.size)
		}
	}
	return out
}

// cdaStub records one stubbed call: the first len(args) arguments and the count.
func cdaStub(w *cgW, base, count int64, args ...int64) {
	for i, v := range args {
		w.p32(base+4*int64(i), v)
	}
	w.p32(count, w.u32(count)+1)
}

// cdaRemove runs the real 0x523330(who, slot, 0) on the node's cells and feeds its slot output back.
func cdaRemove(w *cgW, who, slot int64, r Rand) {
	C := w.u32(0x685174) + 0x1862*who
	v := make([]int64, ccrLen)
	v[ccrCharBase], v[ccrWho], v[ccrCond], v[ccrNotify] = paiChars, who, slot, 0
	if w.f94() {
		v[ccrCombat] = 1
	}
	for j := int64(0); j < 8; j++ {
		v[ccrPres+int(j)] = w.u8(paiSubs + 0x106*j)
	}
	v[ccrWhoPres] = w.u8(paiSubs + 0x106*who)
	for k := 0; k < 18; k++ {
		v[ccrSlot+k] = w.u32(C + 0xa01 + 4*int64(k))
	}
	v[ccrB01], v[ccrB11] = w.u32(C+0xb01), w.u32(C+0xb11)
	out := characterConditionRemoveNode(v, r)
	for k := 0; k < 18; k++ {
		w.p32(C+0xa01+4*int64(k), out[k])
	}
}

// cdaEvent is 0x52f110(who): one other present, living member with state < 0x12
// (members 2.. only when who < 2), chosen by R(n) when several qualify, gets a
// 0x52e690 event; the returned event (if any) is stamped with 0xbb8 and the tick.
func cdaEvent(w *cgW, who int64, r Rand) {
	kind, first := int64(cdaKindA), int64(0)
	if who < 2 {
		kind, first = cdaKindLo, 2
	} else if w.u32(w.u32(0x685174)+0x1862*who+0x75) != 0 {
		kind = cdaKindB
	}
	var elig []int64
	for j := first; j < 8; j++ {
		C := w.u32(0x685174) + 0x1862*j
		if w.u8(w.u32(0x685178)+0x106*j) != 0 && j != who && w.u32(C+0xb11) > 0 && w.u32(C+0xb01) < 0x12 {
			elig = append(elig, j)
		}
	}
	if len(elig) == 0 {
		return
	}
	pick := elig[0]
	if len(elig) > 1 { // 0x40efa0
		pick = elig[r.R(len(elig))]
	}
	cdaStub(w, cdaR52e, cdaR52e+32, w.u32(0x685174)+0x1862*pick, kind, 0, cdaArg4, cdaArg5)
	ev := w.u32(cdaEvRet)
	if ev == 0 {
		return
	}
	if ev != cdaEv {
		panic("character_death_apply: 0x52e690 event pointer outside the one-event fixture")
	}
	w.p32(ev+0x30, 0xbb8)
	w.p32(ev+0x34, cdaTick)
}

// cdaCancel is 0x4e82e0(who): drop pending claims, reset record +0x4d, refresh the
// presentation and call 0x4e8000(who, +0x6d, +0x71, 0, 0).
func cdaCancel(w *cgW, who int64, real bool) {
	S := w.u32(0x685178) + 0x106*who
	if w.resolve(who) != 2 { // 0x53bc10
		paiClearMasks(w, who, w.rec(S+0x4d)) // 0x53ac30
	}
	w.putRec(S+0x4d, cgRec{0, -1, -1, -1})
	w.refresh(who) // 0x55ee30
	if w.sel() == who {
		w.ui(0x20000) // 0x565420
	}
	action, id := w.i32(S+0x6d), w.i32(S+0x71)
	paiStubCall(w, who, action, id)
	if real {
		w.setAction(who, action, id, 0, 0)
	}
}

// cdaObject is 0x50b800(id): entry id of the object table at *0x689f94 (entry 0
// when id >= the count), 0 when absent or flagged (+0xc7).
func cdaObject(w *cgW, id int64) int64 {
	t := w.u32(0x689f94)
	if t == 0 {
		return 0
	}
	if t != cdaTbl {
		panic("character_death_apply: object table pointer outside the fixture")
	}
	n := w.i32(t + 4)
	if n < 0 || n > 4 {
		panic("character_death_apply: object table outside its four entries")
	}
	i := int64(0)
	if id < n {
		if id < 0 {
			panic("character_death_apply: object id below -1 outside the fixture")
		}
		i = id
	}
	e := w.u32(cdaArr + 4*i)
	if e == 0 {
		return 0
	}
	if e < cdaEnt || e >= cdaEnt+4*256 || (e-cdaEnt)%256 != 0 {
		panic("character_death_apply: object entry pointer outside the fixture")
	}
	if w.u8(e+0xc7) != 0 {
		return 0
	}
	return e
}
