package calc

import (
	"encoding/binary"
	"fmt"
	"math"
)

// groupSpawnSelectionNode is 0x556b10 group_spawn_selection(kind, src, posobj),
// cdecl, no return value: spawn a monster group of the species chosen by kind
// at posobj+0x10, give it the source monster's disposition, try to place it
// and either register it or remove it again.
//
// Flow (exe addresses):
//   - species: kind-0x18 below 8 (unsigned, 0x556b17) indexes the jump table
//     0x556c98: 0x18..0x1d -> 0x99, 0x95, 0x96, 0x97, 0x98, 0x94; 0x1f -> 0xc;
//     0x1e and every other kind use the kind itself (0x556b56).
//   - n = 0x517970(0x4e57c0(species)+0xc1); grp = 0x50f1a0(species, n,
//     posobj+0x10, 0, 0, 1); NULL asserts (0x556b89, line 0x258).
//   - 0x4e5840(0x25a, file, grp+0x9f, 1) (result unused); ebx =
//     0x4e5840(0x25b, file, src+8, 1); 0x547570(grp, ebx+0x16 == 2 ? 2 : 1, 0).
//   - byte ebx+0x34c == 1: 0x511050(grp, d, 0, 1) for d = 0, 0x44bb8000,
//     0x453b8000 until al != 0; else f = (float)0x4be5c0(posobj+0x10, 1, 0, 0, 0)
//     and al = 0x510cc0(grp, posobj+0x10, f, 1, ...).
//   - al != 0: 0x505810(); 0x511ce0(grp, 0); 0x50f720(grp). Else 0x50f5d0(grp);
//     0x58ac00(9, [[0x68c09c]+0x9a8], -1, -1, 0).
//
// Native helpers modelled here (they run inside the emulator too):
// 0x547570 disposition set (with 0x511850, 0x505ea0, 0x510280, 0x5477d0,
// 0x565420), 0x505810, 0x511ce0, 0x50f5d0, 0x50f720 (with 0x57dbb0, 0x511be0)
// and the record/group list lookups 0x4e5550 / 0x4e5620 / 0x5100b0 / 0x5101b0.
// Exe data read at runtime: the word table 0x61ec0c (disposition -> string
// index, 0x5475f0) and the dwords 0x5ee60c / 0x5ed8c8 (0x52e5c0 arguments).
//
// Emu (what TestEmu replays) is re/emu/cases/00556b10.json: the D4 callees
// 0x517970, 0x50f1a0, 0x511050, 0x5049c0, 0x53bf80, 0x546e70, 0x538e00,
// 0x4eb7f0, 0x531540, 0x547510 and the excluded 0x4be5c0, 0x510cc0, 0x4e4390,
// 0x58ac00, 0x58aad0, 0x593330, 0x4e3af0, 0x52e5c0 are address stubs; each
// call count and recorded argument is an output, returns are case inputs.
// 0x538e00 is reached through 0x538db0 / 0x538d60 with a stack struct, so
// only its count is recorded.
//
// F runs the real D4 callees diceNode, monsterGroupSpawnNode,
// monsterGroupRepositionNode and combatOpeningConditionNode on the F-only
// inputs; the spawn's count, id, species, identified flag, presence flags,
// last monster id and next ids are written back to the fixture group, and the
// member list becomes the fixture records named by member slots 0..count-1,
// re-identified as the new monster ids. The D4 callees whose return 0x556b10
// discards (0x5049c0, 0x53bf80, 0x546e70, 0x538e00, 0x4eb7f0, 0x531540) are
// only counted and recorded in F as in Emu. Gaps (F only): the real 0x50f1a0
// also runs 0x547570 itself and inserts its monsters into the list 0x683fad;
// neither is modelled.
//
// Fixture caps (calc panics past them): species 0..0x3f and 0x94..0x99
// (template cache), monster ids 1..8, member count 0..4, character index
// 0..1, the spawned group is the fixture group. The 0x50f1a0 NULL assert
// (cov-exempt 0x556b89) and every other assert path panic.
const (
	gsArena = 0x20000000
	gsPos   = gsArena
	gsSrc   = gsArena + 0x40
	gsGrp   = gsArena + 0x100 // g1, g2, g3 at +0x100 each
	gsML    = gsArena + 0x500
	gsGLN   = gsArena + 0x580
	gsGLS   = gsArena + 0x5c0
	gsNL    = gsArena + 0x600
	gsSL    = gsArena + 0x680
	gsRec   = gsArena + 0x1000 // + 0x400*k
	gsVis   = gsArena + 0x3000 // + 0x200*k
	gsSt    = gsArena + 0x4000 // + 0x100*k
	gsO10   = gsArena + 0x4800 // + 0x10*k
	gsTpl   = gsArena + 0x5000 // + 0x400*s
	gsT     = gsArena + 0x6000
	gsCmb   = gsArena + 0x7000
	gsUI    = gsArena + 0x7c00
	gsOpt   = gsArena + 0x7d00
	gsChr   = gsArena + 0x8000
	gsStub  = gsArena + 0xc000
	gsFOnly = gsArena + 0xd000
	gsName  = 0x6875c3 // 0x510280's swprintf buffer
	gsFile  = 0x61f1b8
)

// stub call-count slots, case order
var gsCnt = []int64{0x517970, 0x50f1a0, 0x511050, 0x4be5c0, 0x510cc0, 0x5049c0, 0x593330, 0x53bf80, 0x546e70,
	0x538e00, 0x4eb7f0, 0x531540, 0x547510, 0x4e4390, 0x58ac00, 0x58aad0, 0x4e3af0, 0x52e5c0}

type gsRecSlot struct{ callee, arg, at, n int64 }

var gsRecs = func() (r []gsRecSlot) {
	r = append(r, gsRecSlot{0x517970, 1, 0x100, 0})
	for a := int64(1); a <= 6; a++ {
		r = append(r, gsRecSlot{0x50f1a0, a, 0x110 + 4*(a-1), 0})
	}
	r = append(r, gsRecSlot{0x511050, 1, 0x130, 3}, gsRecSlot{0x511050, 2, 0x140, 3},
		gsRecSlot{0x4be5c0, 1, 0x150, 0}, gsRecSlot{0x4be5c0, 2, 0x154, 0})
	for a := int64(1); a <= 4; a++ {
		r = append(r, gsRecSlot{0x510cc0, a, 0x160 + 4*(a-1), 0})
	}
	r = append(r, gsRecSlot{0x5049c0, 1, 0x180, 12}, gsRecSlot{0x5049c0, 2, 0x1c0, 12},
		gsRecSlot{0x4eb7f0, 1, 0x200, 4}, gsRecSlot{0x531540, 1, 0x210, 4}, gsRecSlot{0x4e4390, 1, 0x220, 4})
	for a := int64(1); a <= 5; a++ {
		r = append(r, gsRecSlot{0x58ac00, a, 0x230 + 4*(a-1), 0})
	}
	for a := int64(1); a <= 5; a++ {
		r = append(r, gsRecSlot{0x58aad0, a, 0x250 + 4*(a-1), 0})
	}
	r = append(r, gsRecSlot{0x4e3af0, 1, 0x270, 4}, gsRecSlot{0x4e3af0, 2, 0x280, 4})
	for a := int64(1); a <= 4; a++ {
		r = append(r, gsRecSlot{0x52e5c0, a, 0x290 + 4*(a-1), 0})
	}
	return
}()

type gsField struct {
	name string
	size int
	at   int64
}

var gsSpec = func() (f []gsField) {
	add := func(n string, size int, at int64) { f = append(f, gsField{n, size, at}) }
	add("kind", 4, 0) // arg 0
	add("srcId", 4, gsSrc+8)
	g := int64(gsGrp)
	for _, x := range []struct {
		n    string
		size int
		off  int64
	}{{"grpCount", 4, 4}, {"grpSpecies", 4, 0x18}, {"grpActive", 1, 0x28}, {"grpRemoved", 1, 0x29},
		{"grpDisposition", 1, 0x2a}, {"grpIdentified", 1, 0x2c}, {"grpMonsterId", 4, 0x9f},
		{"grpAlert", 1, 0xc8}, {"grpAlertTicks", 1, 0xc9}} {
		add(x.n, x.size, g+x.off)
	}
	add("memberCount", 4, gsML+8)
	for k := int64(0); k < 4; k++ {
		add(fmt.Sprint("member", k), 4, gsML+0x10+4*k)
	}
	for k := int64(1); k <= 2; k++ {
		gk := gsGrp + 0x100*k
		add(fmt.Sprintf("g%dCount", k), 4, gk+4)
		add(fmt.Sprintf("g%dActive", k), 1, gk+0x28)
		add(fmt.Sprintf("g%dRemoved", k), 1, gk+0x29)
		add(fmt.Sprintf("g%dDisposition", k), 1, gk+0x2a)
	}
	for k := int64(0); k < 8; k++ {
		r, v, s, o := gsRec+0x400*k, gsVis+0x200*k, gsSt+0x100*k, gsO10+0x10*k
		add(fmt.Sprintf("rec%dPresent", k), 1, r+0x14)
		add(fmt.Sprintf("rec%dAwake", k), 1, r+0x15)
		add(fmt.Sprintf("rec%dDisposition", k), 1, r+0x16)
		add(fmt.Sprintf("rec%dHp", k), 4, r+0x2b)
		add(fmt.Sprintf("rec%dPhase", k), 4, r+0x107)
		add(fmt.Sprintf("rec%dHostile", k), 1, r+0x28a)
		add(fmt.Sprintf("rec%dPlaced", k), 1, r+0x34c)
		add(fmt.Sprintf("rec%dTarget", k), 4, r+0x2e1)
		add(fmt.Sprintf("vis%dFrozen", k), 1, v+0x332)
		add(fmt.Sprintf("vis%dBusy", k), 1, v+0xb4)
		add(fmt.Sprintf("vis%dFlags", k), 4, v+0x1dc)
		add(fmt.Sprintf("st%dState", k), 1, s+0xa4)
		add(fmt.Sprintf("st%dState2", k), 1, s+0xa7)
		add(fmt.Sprintf("o10%dCmd", k), 4, o)
		add(fmt.Sprintf("o10%dBusy", k), 1, o+4)
	}
	add("combatActive", 1, 0x683f94)
	add("reportGate", 1, 0x687500)
	add("frozen", 1, 0x68f105)
	add("mode", 4, 0x68ec78)
	add("worldTime", 4, 0x686a48)
	add("charIdx", 4, 0x6875ef)
	add("uiFlags", 4, gsUI+0xf4)
	add("combatRound", 4, gsCmb+4)
	add("combatPhaseMode", 4, gsCmb+0x7b0)
	add("combatActor", 4, gsCmb+0x7b8)
	add("combatOpeningBlocked", 1, gsCmb+0xa54)
	add("optPaused", 1, gsOpt+0x9c)
	for s := int64(0); s < 3; s++ {
		add(fmt.Sprintf("tpl%dNameId", s), 2, gsTpl+0x400*s+0x187)
		add(fmt.Sprintf("tpl%dNoRemove", s), 1, gsTpl+0x400*s+0x24a)
	}
	add("diceRet", 4, gsStub+0x80)
	add("spawnRet", 4, gsStub+0x84)
	for k := int64(0); k < 3; k++ {
		add(fmt.Sprint("placeRet", k), 4, gsStub+0x88+4*k)
	}
	add("facingBits", 8, gsStub+0x98)
	add("groupPlaceRet", 4, gsStub+0xa0)
	for k := int64(0); k < 4; k++ {
		add(fmt.Sprint("removeRet", k), 4, gsStub+0xa4+4*k)
	}
	for k := int64(0); k < 3; k++ {
		add(fmt.Sprint("openRet", k), 4, gsStub+0xb4+4*k)
	}
	// F-only: read by the real D4 callees in F, never by the stubbed emulator run.
	for s := int64(0); s < 3; s++ {
		t := gsTpl + 0x400*s
		for _, x := range []struct {
			n    string
			off  int64
			size int
		}{{"sizeZ", 0xc1, 2}, {"sizeN", 0xc3, 1}, {"sizeS", 0xc4, 1}, {"hpBase", 0xd6, 2}, {"hpDice", 0xd8, 1},
			{"hpSides", 0xd9, 1}, {"spBase", 0xda, 2}, {"spDice", 0xdc, 1}, {"spSides", 0xdd, 1},
			{"special", 0x26a, 1}, {"templateFlags", 0xd0, 1}} {
			add(fmt.Sprint(x.n, s), x.size, t+x.off)
		}
	}
	for i, n := range []string{"posX", "posY", "posZ"} {
		add(n, 4, gsPos+0x10+4*int64(i))
	}
	add("nextMonsterId", 4, 0x6874be)
	add("nextGroupId", 4, 0x6874ba)
	add("reportEnabled", 1, 0x689b32)
	fo := int64(gsFOnly)
	fadd := func(n string) { add(n, 4, fo); fo += 4 }
	for _, n := range []string{"battleFlags", "animationFrames", "unbornCount", "partyX", "partyY", "partyZ",
		"partyHeading", "partyRadius", "mainRadius", "headingTarget0", "headingTarget1", "headingTarget2",
		"sceneRadius0", "sceneRadius1", "sceneRadius2", "sceneRadius3", "sceneRadius4", "placementClearance",
		"modelPosition0", "modelPosition1", "modelPosition2"} {
		fadd(n)
	}
	return
}()

var gsOuts = func() (o []gsField) {
	add := func(n string, size int, at int64) { o = append(o, gsField{n, size, at}) }
	for i, a := range gsCnt {
		add(fmt.Sprintf("calls%x", a), 4, gsStub+4*int64(i))
	}
	for _, r := range gsRecs {
		if r.n == 0 {
			add(fmt.Sprintf("arg%x_%d", r.callee, r.arg), 4, gsStub+r.at)
			continue
		}
		for k := int64(0); k < r.n; k++ {
			add(fmt.Sprintf("arg%x_%d_%d", r.callee, r.arg, k), 4, gsStub+r.at+4*k)
		}
	}
	add("grpDispositionAfter", 1, gsGrp+0x2a)
	add("grpCaFlag", 1, gsGrp+0xca)
	add("grpDispositionTime", 4, gsGrp+0xcb)
	add("grpRemovedAfter", 1, gsGrp+0x29)
	add("grpAlertAfter", 1, gsGrp+0xc8)
	add("grpAlertTicksAfter", 1, gsGrp+0xc9)
	for k := int64(0); k < 8; k++ {
		add(fmt.Sprintf("rec%dDispositionAfter", k), 1, gsRec+0x400*k+0x16)
		add(fmt.Sprintf("rec%dTargetAfter", k), 4, gsRec+0x400*k+0x2e1)
		add(fmt.Sprintf("o10%dCmdAfter", k), 4, gsO10+0x10*k)
		add(fmt.Sprintf("vis%dBusyAfter", k), 1, gsVis+0x200*k+0xb4)
	}
	add("uiFlagsAfter", 4, gsUI+0xf4)
	for k := int64(0); k < 8; k++ {
		add(fmt.Sprint("nameBuf", k), 2, gsName+2*k)
	}
	return
}()

var gsIn, gsOutNames = func() (in, out []string) {
	for _, f := range gsSpec {
		in = append(in, f.name)
	}
	for _, o := range gsOuts {
		out = append(out, o.name)
	}
	return
}()

func init() {
	Nodes["00556b10"] = Node{Addr: "00556b10", In: gsIn, Out: gsOutNames, F: groupSpawnSelectionNode,
		Emu: func(in []int64, r Rand) []int64 { return gsNode(in, r, false) }}
}

func groupSpawnSelectionNode(in []int64, r Rand) []int64 { return gsNode(in, r, true) }

func gsNode(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(gsSpec) {
		panic("group spawn selection: wrong input count")
	}
	w := &cgW{m: map[int64]byte{}}
	gsSetup(w)
	for i, f := range gsSpec[1:] {
		w.wr(f.at, f.size, in[i+1])
	}
	e := &gsRun{w: w, r: r, real: real}
	e.run(in[0])
	var out []int64
	for _, o := range gsOuts {
		out = append(out, w.rd(o.at, o.size))
	}
	return out
}

// gsSetup is the case's setup.
func gsSetup(w *cgW) {
	for _, x := range [][2]int64{{0x683fad, gsNL}, {0x683fb9, gsSL}, {0x683fb1, gsGLN}, {0x683fbd, gsGLS},
		{0x68c09c, gsT}, {0x6836a8, gsCmb}, {0x68edcc, gsUI}, {0x659c0c, gsOpt}, {0x685174, gsChr}} {
		w.p32(x[0], x[1])
	}
	for _, l := range []struct {
		obj, cnt int64
		items    []int64
	}{{gsNL, 6, []int64{gsRec, gsRec + 0x400, gsRec + 0x800, gsRec + 0xc00, gsRec + 0x1000, gsRec + 0x1400}},
		{gsSL, 2, []int64{gsRec + 0x1800, gsRec + 0x1c00}},
		{gsGLN, 3, []int64{gsGrp, gsGrp + 0x100, gsGrp + 0x200}},
		{gsGLS, 1, []int64{gsGrp + 0x300}}} {
		w.p32(l.obj, l.obj+0x10)
		w.p32(l.obj+4, 8)
		w.p32(l.obj+8, l.cnt)
		for i := int64(0); i < 8; i++ {
			v := 0xdead0100 + i
			if i < int64(len(l.items)) {
				v = l.items[i]
			}
			w.p32(l.obj+0x10+4*i, v)
		}
	}
	w.p32(gsML, gsML+0x10)
	w.p32(gsML+4, 8)
	for i := int64(4); i < 8; i++ {
		w.p32(gsML+0x10+4*i, 9+i)
	}
	w.p32(gsGrp+8, gsML)
	for k := int64(0); k < 4; k++ {
		w.p32(gsGrp+0x100*k, 0x51+k)
	}
	w.p8(gsGrp+0xca, 0x5a)
	w.p32(gsGrp+0xcb, 0x77777777)
	for k := int64(0); k < 8; k++ {
		r := gsRec + 0x400*k
		w.p32(r, k+1)
		w.p32(r+0xc, gsVis+0x200*k)
		w.p32(r+0x10, gsO10+0x10*k)
		w.p32(gsVis+0x200*k+0x1d8, gsSt+0x100*k)
	}
	for sp := int64(0); sp < 0x9a; sp++ {
		if sp < 0x40 || sp >= 0x94 {
			w.p32(0x6840c7+4*sp, gsTpl+0x400*(sp%3))
		}
	}
	for _, off := range []int64{0x75c, 0x760, 0x800, 0x804, 0x9a8} {
		w.p32(gsT+off, 0x54000000+off)
	}
	for idx, name := range []string{"Rex", "Vi"} {
		for i, ch := range name + "\x00" {
			w.wr(gsChr+6242*int64(idx)+5+2*int64(i), 2, int64(ch))
		}
	}
	for k := int64(0); k < 8; k++ {
		w.wr(gsName+2*k, 2, 0)
	}
}

type gsRun struct {
	w    *cgW
	r    Rand
	real bool
	n    map[int64]int64
	cnt  []int64     // count slots (nil: gsCnt); 0x5500c0 shares these helpers with its own case
	recs []gsRecSlot // record slots (nil: gsRecs)
}

// call counts a stubbed call and records its arguments (args[i] is stack
// argument i+1) at the case's record slots.
func (e *gsRun) call(callee int64, args ...int64) int64 {
	if e.n == nil {
		e.n = map[int64]int64{}
	}
	k := e.n[callee]
	cnt, recs := gsCnt, gsRecs
	if e.cnt != nil {
		cnt, recs = e.cnt, e.recs
	}
	for _, s := range recs {
		if s.callee != callee || s.arg > int64(len(args)) {
			continue
		}
		at := gsStub + s.at
		if s.n != 0 {
			if k >= s.n {
				panic(fmt.Sprintf("group spawn selection: %#x called more than the case's %d recorded calls", callee, s.n))
			}
			at += 4 * k
		}
		e.w.p32(at, args[s.arg-1]&0xffffffff)
	}
	e.n[callee] = k + 1
	for i, a := range cnt {
		if a == callee {
			e.w.p32(gsStub+4*int64(i), k+1)
		}
	}
	return k
}

// seq is a stub's k-th return from the case's "ret": {"at", "n"} slots.
func (e *gsRun) seq(at, k, n int64) int64 {
	if k >= n {
		panic("group spawn selection: stub return sequence exhausted (emulator stops)")
	}
	return e.w.u32(gsStub + at + 4*k)
}

func (e *gsRun) template(sp int64) int64 {
	if sp >= 1000 {
		panic("group spawn selection: 0x4e57c0 asserts on species >= 1000")
	}
	if !(sp >= 0 && sp < 0x40 || sp >= 0x94 && sp <= 0x99) {
		panic("group spawn selection: species outside the fixture template cache (database file load)")
	}
	return e.w.u32(0x6840c7 + 4*sp)
}

// recIndex is 0x4e5550(id, flag 1): index in the normal list 0x683fad, or
// 0x2710+i in the special list 0x683fb9.
func (e *gsRun) recIndex(id int64) int64 {
	w := e.w
	for i := int64(0); i < w.u32(w.u32(0x683fad)+8); i++ {
		if w.u32(e.recPtr(i)) == id {
			return i
		}
	}
	sl := w.u32(0x683fb9)
	for i := int64(0); i < w.u32(sl+8); i++ {
		if w.u32(w.u32(w.u32(sl)+4*i)) == id {
			return i + 0x2710
		}
	}
	panic(fmt.Sprintf("group spawn selection: monster id %d not found (0x4e5550 asserts; fixture ids 1..8)", id))
}

// recPtr is 0x4e5620(idx).
func (e *gsRun) recPtr(i int64) int64 {
	w := e.w
	l := w.u32(0x683fad)
	if i >= 0x2710 && i < 0x4e20 {
		l, i = w.u32(0x683fb9), i-0x2710
	}
	if i >= w.u32(l+8) {
		panic("group spawn selection: 0x4e5620 index past the list (assert)")
	}
	p := w.u32(w.u32(l) + 4*i)
	if p == 0 {
		panic("group spawn selection: 0x4e5620 NULL record (assert)")
	}
	return p
}

func (e *gsRun) rec(id int64) int64 { return e.recPtr(e.recIndex(id)) } // 0x4e5840(.., 1)

// members returns the record pointers of grp's member list (0x5e2c70/0x5e2c80).
func (e *gsRun) members(g int64) []int64 {
	w := e.w
	l := w.u32(g + 8)
	if l == 0 {
		return nil
	}
	var p []int64
	for i := int64(0); i < w.u32(l+8); i++ {
		p = append(p, e.rec(w.u32(w.u32(l)+4*i)))
	}
	return p
}

// act15 is 0x4ca4c0 on the record's visual object.
func (e *gsRun) act15(rec int64) bool {
	st := e.w.u32(e.w.u32(rec+0xc) + 0x1d8)
	return int8(e.w.u8(st+0xa4)) == 0x15 || e.w.u8(st+0xa7) == 0x15
}

// group is 0x5101b0(0x5100b0(id, 1)).
func (e *gsRun) group(id int64) int64 {
	w := e.w
	for _, l := range []int64{w.u32(0x683fb1), w.u32(0x683fbd)} {
		for i := int64(0); i < w.u32(l+8); i++ {
			if g := w.u32(w.u32(l) + 4*i); w.u32(g) == id {
				return g
			}
		}
	}
	panic("group spawn selection: group id not found (0x5100b0 asserts)")
}

func gsExeWord(va uint32) int64  { return int64(binary.LittleEndian.Uint16(ticRead(va, 2))) }
func gsExeDword(va uint32) int64 { return int64(binary.LittleEndian.Uint32(ticRead(va, 4))) }

func (e *gsRun) ui() {
	w := e.w
	if w.i32(0x68ec78) == 7 && w.u32(0x68edcc) != 0 { // 0x565420
		p := w.u32(0x68edcc) + 0xf4
		w.p32(p, w.u32(p)|0x20000)
	}
}

// name is 0x510280(grp).
func (e *gsRun) name(g int64) int64 {
	w := e.w
	t := e.template(w.u32(g + 0x18))
	dl := int64(0)
	if w.u32(g+4) != 1 {
		dl = 1
	}
	if w.rd(t+0x187, 2) == 0x18d {
		// swprintf(0x6875c3, L"Al-%s" (0x614b44), name of character [0x6875ef])
		idx := w.u32(0x6875ef)
		if idx > 1 {
			panic("group spawn selection: character index outside the fixture's 0..1")
		}
		src := w.u32(0x685174) + 6242*idx + 5
		s := []int64{'A', 'l', '-'}
		for i := int64(0); ; i++ {
			c := w.rd(src+2*i, 2)
			s = append(s, c)
			if c == 0 {
				break
			}
		}
		for i, c := range s {
			w.wr(gsName+2*int64(i), 2, c)
		}
		return gsName
	}
	if w.u8(g+0x2c) != 0 {
		return t + 0x30*dl
	}
	return t + 0x30*(dl+2)
}

// setDisposition is 0x547570(grp, disp, 0); grp's link slots are 0.
func (e *gsRun) setDisposition(g, disp int64) {
	w := e.w
	all := true // 0x511850
	for _, m := range e.members(g) {
		all = all && e.act15(m)
	}
	if all {
		return
	}
	if w.u8(w.u32(e.rec(w.u32(g+0x9f))+0xc)+0x332) != 0 {
		return
	}
	old := w.u8(g + 0x2a)
	if old == disp {
		return
	}
	w.p8(g+0x2a, disp)
	w.p8(g+0xca, 0)
	for _, m := range e.members(g) { // 0x505ea0
		if w.u8(m+0x14) != 0 && !e.act15(m) && w.u32(m+0x2b) != 0 && w.u32(m+0x107) < 0xc && w.u8(m+0x28a) == 1 {
			T := w.u32(0x68c09c)
			c := int64(0)
			if w.u32(g+4) != 1 {
				c = 1
			}
			e.call(0x58aad0, 9, 0x61dd24, e.name(g), w.u32(T+0x75c+4*c), w.u32(T+4*gsExeWord(uint32(0x61ec0c+2*disp))))
			break
		}
	}
	w.p32(g+0xcb, w.u32(0x686a48))
	if old != 0 {
		e.call(0x538e00) // 0x538db0(grp id, 1)
	}
	for _, m := range e.members(g) {
		e.memberDisposition(m, disp)
	}
	if w.u32(g+0xa3) != 0 || w.u32(g+0xa7) != 0 || w.u32(g+0xab) != 0 {
		panic("group spawn selection: linked groups are outside the fixture")
	}
	e.ui()
}

// memberDisposition is 0x5477d0(rec, disp).
func (e *gsRun) memberDisposition(m, al int64) {
	w := e.w
	bl := w.u8(m + 0x16)
	if bl == al || w.u8(w.u32(m+0xc)+0x332) != 0 {
		return
	}
	cl := w.u8(m + 0x15)
	w.p8(m+0x16, al)
	if cl != 0 && (al == 1 || bl == 1) {
		e.call(0x546e70)
	}
	if bl != 0 {
		e.call(0x538e00) // 0x538d60(rec id, 1)
	}
	if w.u8(m+0x15) == 0 || w.u32(m+0x2b) == 0 || w.u8(m+0x16) == 0 {
		return
	}
	cmb := w.u32(0x6836a8)
	if w.u8(0x683f94) != 0 && w.u32(cmb+0x7b0) != 1 && w.u32(cmb+0x7b8) == m {
		e.call(0x4eb7f0, m)
		w.p32(m+0x2e1, 0xffffffff)
		w.p32(w.u32(m+0x10), 0)
		return
	}
	if w.u8(w.u32(m+0x10)+4) == 0 {
		e.call(0x531540, m)
		return
	}
	w.p32(m+0x2e1, 0xffffffff)
	w.p32(w.u32(m+0x10), 0)
}

// register is 0x505810(); 0x511ce0(grp, 0); 0x50f720(grp).
func (e *gsRun) register(g int64) {
	e.refresh()
	e.release(g)
}

// refresh is 0x505810.
func (e *gsRun) refresh() {
	w := e.w
	nl := w.u32(0x683fad)
	for _, flag := range []int64{1, 0} {
		for i := int64(0); i < w.u32(nl+8); i++ {
			e.call(0x5049c0, e.recPtr(i), flag, 0)
		}
		if flag == 1 && w.u8(0x683f94) != 0 {
			e.call(0x593330)
			e.call(0x53bf80)
		}
	}
}

// release is 0x511ce0(grp, 0); 0x50f720(grp).
func (e *gsRun) release(g int64) {
	w := e.w
	for _, m := range e.members(g) {
		w.p8(w.u32(m+0xc)+0xb4, 0)
	}
	// 0x50f720
	if w.u8(0x687500) != 0 && w.u8(g+0x2a) == 1 && w.u8(0x683f94) == 0 {
		return
	}
	if w.u8(0x68f105) != 0 || w.u8(g+0x29) != 0 || w.u8(e.template(w.u32(g+0x18))+0x24a) != 0 {
		return
	}
	ms := e.members(g)
	for _, m := range ms {
		if w.u8(w.u32(m+0xc)+0xb4) != 0 {
			return
		}
	}
	if w.u8(w.u32(0x659c0c)+0x9c) != 0 {
		return
	}
	for _, m := range ms {
		if w.u32(w.u32(m+0xc)+0x1dc)>>5&1 != 0 {
			return
		}
	}
	live := false
	for _, m := range ms {
		live = live || !e.act15(m)
	}
	if !live {
		return
	}
	if w.u32(g+0xa3) != 0 {
		panic("group spawn selection: linked groups are outside the fixture")
	}
	for _, m := range ms {
		e.call(0x4e4390, m)
	}
	w.p8(g+0x29, 1)
	cmb := w.u32(0x6836a8)
	if w.u8(g+0x2a) == 1 && w.u8(0x683f94) != 0 && w.u32(cmb+4) > 0 {
		n := int64(0)
		gl := w.u32(0x683fb1)
		for i := int64(0); i < w.u32(gl+8); i++ {
			x := w.u32(w.u32(gl) + 4*i)
			if w.u8(x+0x28) == 0 || w.u8(x+0x29) == 0 || w.u32(x+4) == 0 {
				continue
			}
			if w.u8(x+0x2a) == 1 {
				n++
				continue
			}
			k := e.call(0x547510)
			ok := e.seq(0xb4, k, 3)
			if e.real {
				ok = combatOpeningConditionNode([]int64{w.u8(0x683f94), w.u8(cmb + 0xa54), w.u32(cmb + 4)}, e.r)[0]
			}
			if ok&0xff != 0 {
				n++
			}
		}
		if n > 1 {
			e.call(0x52e5c0, gsExeDword(0x5ee60c), 0xffffffff, 0, gsExeDword(0x5ed8c8))
		}
	}
	for _, off := range []int64{0xa7, 0xab, 0xaf, 0xb3} {
		if w.u32(g+off) != 0 {
			panic("group spawn selection: linked groups are outside the fixture")
		}
	}
	e.ui()
	// 0x511be0(grp id, 0)
	x := e.group(w.u32(g))
	if w.u8(x+0xc8) != 0 {
		w.p8(x+0xc8, 0)
		w.p8(x+0xc9, 0)
	}
}

// remove is 0x50f5d0(grp): 0x4e3af0(idx, 1) from the last member down until
// one returns al == 0.
func (e *gsRun) remove(g int64) {
	w := e.w
	l := w.u32(g + 8)
	cnt := int64(0)
	if l != 0 {
		cnt = w.u32(l + 8)
	}
	for i := int64(int32(cnt)) - 1; i >= 0; i-- {
		idx := e.recIndex(w.u32(w.u32(l) + 4*i))
		k := e.call(0x4e3af0, idx, 1)
		if e.seq(0xa4, k, 4)&0xff == 0 {
			break
		}
	}
}

// spawn is 0x50f1a0(species, n, posobj+0x10, 0, 0, 1).
func (e *gsRun) spawn(sp, n int64) int64 {
	w := e.w
	e.call(0x50f1a0, sp, n, gsPos+0x10, 0, 0, 1)
	if !e.real {
		g := w.u32(gsStub + 0x84)
		if g == 0 {
			panic("group spawn selection: 0x50f1a0 returned NULL (assert 0x556b89)")
		}
		if g != gsGrp {
			panic("group spawn selection: spawned group is not the fixture group")
		}
		return g
	}
	t, g := e.template(sp), int64(gsGrp)
	out := monsterGroupSpawnNode([]int64{
		w.rs(t+0xd6, 2), w.u8(t + 0xd8), w.u8(t + 0xd9), w.rs(t+0xda, 2), w.u8(t + 0xdc), w.u8(t + 0xdd),
		w.u8(t + 0x26a), w.u32(0x6874be), w.u32(gsPos + 0x10), w.u32(gsPos + 0x14), w.u32(gsPos + 0x18), 0, 0, 0,
		w.i32(0x68ec78), w.u32(gsFOnly), n & 0xffffffff, 0, 0, 1, w.u8(t + 0xd0), w.u32(0x6874ba),
		w.u32(0x686a48), w.u8(0x689b32), 0, w.rd(t+0x187, 2), w.u8(gsFOnly + 4), w.u32(gsFOnly + 8),
	}, e.r)
	cnt := out[2]
	if cnt > 4 {
		panic("group spawn selection: spawned member count past the fixture's 4 member slots")
	}
	// the new monsters are the fixture records named by member slots 0..cnt-1
	first := out[19] - cnt + 1
	var recs []int64
	seen := map[int64]bool{e.rec(w.u32(gsSrc + 8)): true}
	for k := int64(0); k < cnt; k++ {
		rp := e.rec(w.u32(gsML + 0x10 + 4*k))
		if seen[rp] {
			panic("group spawn selection: spawned members must be distinct fixture records other than the source monster")
		}
		seen[rp] = true
		recs = append(recs, rp)
	}
	for k, rp := range recs {
		w.p32(rp, first+int64(k))
		w.p32(gsML+0x10+4*int64(k), first+int64(k))
	}
	w.p32(gsML+8, cnt)
	w.p32(g, out[1])
	w.p32(g+4, cnt)
	w.p32(g+0x18, sp)
	w.p8(g+0x28, out[11])
	w.p8(g+0x29, out[12])
	w.p8(g+0x2c, out[14])
	w.p32(g+0x9f, out[19])
	w.p32(0x6874ba, out[21])
	w.p32(0x6874be, out[22])
	return g
}

// place is the 0x511050 attempt d (Emu: the case's k-th return; F: the real
// reposition, whose scene placement always fails in its model).
func (e *gsRun) place(g, d int64) int64 {
	k := e.call(0x511050, g, d, 0, 1)
	if !e.real {
		return e.seq(0x88, k, 3) & 0xff
	}
	w := e.w
	f := func(off int64) int64 { return w.u32(gsFOnly + off) }
	in := []int64{1, d, f(0xc), f(0x10), f(0x14), f(0x18), f(0x1c), w.u8(g + 0x28), w.u32(g + 4), f(0x20),
		0, 0, 0, 0, 0, 0, 0, 0, f(0x24), f(0x28), f(0x2c),
		f(0x30), f(0x34), f(0x38), f(0x3c), f(0x40), 0, 0, 0, 0, f(0x44), f(0x48), f(0x4c), f(0x50)}
	return monsterGroupRepositionNode(in, e.r)[5] & 0xff
}

func (e *gsRun) run(kind int64) {
	w := e.w
	sp := kind
	if k := uint32(kind - 0x18); k < 8 { // 0x556c98
		sp = []int64{0x99, 0x95, 0x96, 0x97, 0x98, 0x94, kind, 0xc}[k]
	}
	t := e.template(sp)
	k := e.call(0x517970, t+0xc1)
	n := w.u32(gsStub + 0x80)
	if e.real {
		n = diceNode([]int64{w.rs(t+0xc1, 2), w.u8(t + 0xc3), w.u8(t + 0xc4)}, e.r)[0] & 0xffffffff
	}
	_ = k
	g := e.spawn(sp, n)
	e.rec(w.u32(g + 0x9f))
	src := e.rec(w.u32(gsSrc + 8))
	disp := int64(1)
	if w.u8(src+0x16) == 2 {
		disp = 2
	}
	e.setDisposition(g, disp)
	var al int64
	if w.u8(src+0x34c) == 1 {
		for _, d := range []int64{0, 0x44bb8000, 0x453b8000} {
			if al = e.place(g, d); al != 0 {
				break
			}
		}
	} else {
		e.call(0x4be5c0, gsPos+0x10, 1, 0, 0, 0)
		f := int64(math.Float32bits(float32(math.Float64frombits(uint64(w.rd(gsStub+0x98, 8))))))
		e.call(0x510cc0, g, gsPos+0x10, f, 1)
		al = w.u32(gsStub+0xa0) & 0xff
	}
	if al != 0 {
		e.register(g)
		return
	}
	e.remove(g)
	e.call(0x58ac00, 9, w.u32(w.u32(0x68c09c)+0x9a8), 0xffffffff, 0xffffffff, 0)
}
