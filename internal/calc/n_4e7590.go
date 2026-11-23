package calc

import "fmt"

// 0x4e7590 combat_pc_action_init: for each of the eight party slots whose
// record byte +0 is set, reset the slot's action sources and the character's
// action flags. The world is the sparse cgW memory of n_520d10.go; the cells below mirror
// re/emu/cases/004e7590.json one to one (same order).
//
// Emu stubs the D4 callee 0x4e8000 (composition ruling 2026-10-09): the call is
// only recorded (arguments 1-5, call count). F calls the real cgW.setAction as
// well. 0x4c5eb0 (selection-marker thunk, registered 排除) is address-stubbed too:
// every (slot, monster id, flag) call from 0x53ac30 / 0x539630 is recorded
// (arguments 1-3, call count), so monster marker state is not a node input.
// Fixture caps (calc panics past them): two monsters, four pending ids per
// slot, the one UI object.

const (
	paiChars = cgA
	paiSubs  = cgA + 65536
	paiCtx   = cgA + 69632
	paiUI    = cgA + 86016
	paiStub  = cgA + 94208
	paiG69c  = cgA + 110592
	paiPend  = cgA + 118784
	paiMon   = cgA + 262144
)

type paiCell struct {
	name string
	at   int64
	size int // negative = signed
}

var paiMonEntry = [2]int64{512, 576}
var paiMonRec = [2]int64{4096, 5120}
var paiMonInner = [2]int64{8192, 10240}

var paiIn, paiOut = paiLayout()

func paiLayout() (in []paiCell, out []paiCell) {
	add := func(n string, at int64, s int) { in = append(in, paiCell{n, at, s}) }
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
	for x := int64(0); x < 8; x++ {
		S, C, D := paiSubs+0x106*x, paiChars+0x1862*x, paiCtx+0xd4*x
		add(fmt.Sprintf("s%d.act", x), S, 1)
		for _, o := range []int64{1, 5, 9} {
			add(fmt.Sprintf("s%d.f%d", x, o), S+o, -4)
		}
		for r, base := range []int64{0x1d, 0x4d} {
			for i := int64(0); i < 8; i++ {
				add(fmt.Sprintf("s%d.r%d.%d", x, r, i), S+base+4*i, -4)
			}
		}
		add(fmt.Sprintf("s%d.a6d", x), S+0x6d, -4)
		add(fmt.Sprintf("s%d.a71", x), S+0x71, -4)
		add(fmt.Sprintf("c%d.b01", x), C+0xb01, 4)
		add(fmt.Sprintf("c%d.b11", x), C+0xb11, -4)
		add(fmt.Sprintf("c%d.f1817", x), C+0x1817, 4)
		add(fmt.Sprintf("c%d.f181b", x), C+0x181b, 4)
		add(fmt.Sprintf("c%d.f181f", x), C+0x181f, 1)
		add(fmt.Sprintf("d%d.4c", x), D+0x4c, 1)
		add(fmt.Sprintf("d%d.9c", x), D+0x9c, -4)
		add(fmt.Sprintf("d%d.a0", x), D+0xa0, -4)
		add(fmt.Sprintf("d%d.a4", x), D+0xa4, 1)
		add(fmt.Sprintf("pc%d", x), 0x683794+0x118*x, -4)
		for i := int64(0); i < 4; i++ {
			add(fmt.Sprintf("pi%d.%d", x, i), paiPend+16*x+4*i, -4)
		}
	}
	out = append(out, in...)
	for i := int64(0); i < 5; i++ {
		out = append(out, paiCell{fmt.Sprintf("stub.a%d", i+1), paiStub + 4*i, -4})
	}
	out = append(out, paiCell{"stub.n", paiStub + 32, 4})
	for i := int64(0); i < 3; i++ {
		out = append(out, paiCell{fmt.Sprintf("mk.a%d", i+1), paiStub + 40 + 4*i, -4})
	}
	out = append(out, paiCell{"mk.n", paiStub + 64, 4})
	return
}

func paiNames(c []paiCell) []string {
	n := make([]string, len(c))
	for i := range c {
		n[i] = c[i].name
	}
	return n
}

func init() {
	Nodes["004e7590"] = Node{Addr: "004e7590", In: paiNames(paiIn), Out: paiNames(paiOut),
		F: combatPcActionInitNode, Emu: combatPcActionInitEmu}
}

// combatPcActionInitNode runs the real 0x4e8000 (cgW.setAction) per active slot.
func combatPcActionInitNode(in []int64, r Rand) []int64 {
	return paiExec(in, true)
}

// combatPcActionInitEmu replays the case: 0x4e8000 is address-stubbed.
func combatPcActionInitEmu(in []int64, r Rand) []int64 {
	return paiExec(in, false)
}

func paiExec(in []int64, real bool) []int64 {
	if len(in) != len(paiIn) {
		panic("combat_pc_action_init: wrong input count")
	}
	w := &cgW{m: map[int64]byte{}}
	// Constant pointers (the case's setup).
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
	for i, c := range paiIn {
		s := c.size
		if s < 0 {
			s = -s
		}
		w.wr(c.at, s, in[i])
	}
	if n := w.i32(paiMon + 8); n < 0 || n > 2 {
		panic("combat_pc_action_init: monster list outside the two-monster fixture")
	}
	if u := w.uip(); u != 0 && u != paiUI {
		panic("combat_pc_action_init: UI object pointer outside the one-object fixture")
	}
	for x := int64(0); x < 8; x++ {
		S, C, D := w.u32(0x685178)+0x106*x, w.u32(0x685174)+0x1862*x, w.u32(0x6836a8)+0xd4*x
		if w.u8(S) == 0 {
			continue
		}
		w.p32(S+1, -1)
		w.p32(S+5, -1)
		w.p32(S+9, -1)
		w.putRec(S+0x1d, cgRec{0, -1, -1, -1}) // 0x536170
		if w.resolve(x) != 2 {                 // 0x53bc10
			old := w.rec(S + 0x4d)
			paiClearMasks(w, x, old) // 0x53ac30
		}
		w.putRec(S+0x4d, cgRec{0, -1, -1, -1})
		w.refresh(x) // 0x55ee30
		if w.sel() == x {
			w.ui(0x20000) // 0x565420
		}
		action, id := w.i32(S+0x6d), w.i32(S+0x71)
		paiStubCall(w, x, action, id)
		if real {
			w.setAction(x, action, id, 0, 0)
		}
		w.p8(D+0xa4, 0xff)
		w.p32(D+0x9c, -1)
		w.p32(D+0xa0, -1)
		var flag int64
		if w.i32(C+0xb11) == 0 || w.u32(C+0xb01) >= 0xf {
			flag = 1
		}
		w.p8(D+0x4c, flag)
		w.p8(C+0x181f, 0)
		w.p32(C+0x1817, 0)
		w.p32(C+0x181b, 0)
	}
	out := make([]int64, len(paiOut))
	for i, c := range paiOut {
		if c.size < 0 {
			out[i] = w.rs(c.at, -c.size)
		} else {
			out[i] = w.rd(c.at, c.size)
		}
	}
	return out
}

// paiStubCall records one 0x4e8000(x, action, id, 0, 0) call: arguments 1-5 and the count.
func paiStubCall(w *cgW, x, action, id int64) {
	for i, v := range []int64{x, action, id, 0, 0} {
		w.p32(paiStub+4*int64(i), v)
	}
	w.p32(paiStub+32, w.u32(paiStub+32)+1)
}

// paiMark records one 0x4c5eb0(x, id, flag) call: arguments 1-3 and the count.
func paiMark(w *cgW, x, id, flag int64) {
	for i, v := range []int64{x, id, flag} {
		w.p32(paiStub+40+4*int64(i), v)
	}
	w.p32(paiStub+64, w.u32(paiStub+64)+1)
}

// paiClearMasks is 0x53ac30(x, rec) = cgW.clearMasks plus the 0x4c5eb0 record
// for each monster found (0x539630 clears its mask bit, then calls 0x4c5eb0).
func paiClearMasks(w *cgW, x int64, rec cgRec) {
	if pc := w.i32(w.pcAddr(x)); pc > 0 {
		if pc > 4 {
			panic("combat_pc_action_init: more than four pending ids")
		}
		list := w.u32(0x68379c + 0x118*x)
		for i := int64(0); i < pc; i++ {
			paiMaskOne(w, x, w.i32(list+4*i))
		}
		w.p32(w.pcAddr(x), 0)
		return
	}
	if rec[0] == 3 && rec[2] != -1 {
		paiMaskOne(w, x, rec[2])
	}
}

func paiMaskOne(w *cgW, x, id int64) {
	if w.find(id, false) != -1 {
		w.setMask(x, id, 0)
		paiMark(w, x, id, 0)
	}
}
