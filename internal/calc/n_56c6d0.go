package calc

import (
	"fmt"
	"strconv"
)

// dialogue_status_cleanup 0x56c6d0(dlg, state, event, flag, open): status pass when a
// dialogue with object dlg starts (build/wiz8.asm:494715..494990). G = *0x649f1c (dialogue
// panel), E = *(dlg+6) (definition record), R = G+0x1d4 (panel's dialogue record).
//
//  1. byte 0x68f0f9 = 0, G+0x25c = 0.
//  2. dlg kind byte (+0x2e) 0xf or 0x12: party slots 0 and 1 only (presence offset limit
//     0x20c = 2*0x106, 0x56c78c). A present slot (byte *0x685178 + 0x106*i) resolves its
//     NPC with 0x50b800(id at slot+0xfa) (list *0x689f94: id < count signed -> entry id,
//     else entry 0; a null entry or one with +0xc7 set gives 0 and the original then
//     faults reading +0x2e: panic). NPC kind 0x10 or 0x11 and character (*0x685174 +
//     0x1862*i) u32 +0xb01 >= 0xf (unsigned): for every state slot s = 0..0x12 whose u32
//     +0xa01+4s is non-zero, 0x523330(i, s, 0); for s = 0x12 also +0xb11 = 10.
//  3. *0x683fd7 voice queue: 0x52e460 (int32 +0x44 > 0) -> 0x52e3b0.
//  4. byte 0x683f9b == 0: 0x56d030(dlg, flag).
//  5. low byte of open != 0: 0x56cad0(dlg, state, 1), return (0x56ca42 reads the stack after
//     one push, so the second argument is state, not event).
//  6. E+0x54 == 0 and E+0x2ea == 0: attitude class 0x50a500(dlg) (0x50a280 = npcAttitudeNode,
//     signed byte < 0x21 -> 2, < 0x42 -> 1, else 0) == 2 and E+0x56 == 0: post 0x18,
//     return; otherwise low byte of flag == 0: 0x56cad0(dlg, state, 0), return; else
//     post (event == -1 ? 0 : event) and the tail.
//     E+0x2ea != 0: state != 0 copies 12 bytes to G+0x1ed, byte 0x6874ca != 0 sets byte
//     G+0x1f9 = 1, 0x575810(G+0x1ed); state == 0: event == -1 -> 0x577290(1), else post
//     event. Then the tail.
//     E+0x54 != 0 (E+0x2ea == 0): post (event == -1 ? 0 : event) and the tail.
//  7. Tail: R kind not 0x84/0x85 and m = 0x50a3c0(R) != 0: 0x4c6240(*(m+0xc), 0). 0x56aa30.
//     byte G+0x252 = 1, byte 0x683f97 = 1, G+0x25c = 0. m = 0x50a3c0(R) != 0:
//     0x476f90(&position, 0) (camera; the float position from *(m+0xc) is not observed).
//
// 0x50a3c0(rec): rec+0x1a and +0x25 both set -> monster with id rec+0x16 (list *0x683fad,
// then group list *0x683fb9; a miss asserts: panic). Posting is 0x528830(kind,0,0,0), the
// queue of dialogueGreetingStatusResetNode (heap event {kind, stamp *0x68c4ac at +0x20},
// count+1 > capacity (signed) reallocates to count+1 words).
//
// Emu address-stubs the D4 callees 0x523330, 0x56d030, 0x577290 and the excluded 0x4c6240,
// 0x56aa30, 0x476f90, 0x56cad0, 0x575810, 0x52e3b0 (every caller ignores eax; each returns
// 0x5a5a5a5a) and records call counts and arguments. F runs the real 0x523330 per call on a
// callee world of the node's presence, state words, +0xb01 and +0xb11 (formations with
// nobody placed, zero elsewhere) and
// feeds its slot output back; and the real 0x56d030, whose panel+0x1d4 and 0x68c4ac
// outputs (the object) replace R and the event stamp. Its other inputs are zero: they do
// not reach those two outputs. 0x577290's writes are not read by this node; F does not
// run it.
//
// Fixture caps (panic past them): NPC list count 1..2 with entries partyNPC0/1 (the
// 0050a280 adapter's arena addresses), slot NPC id >= 0; monster ids 0x100 (mon0) and
// 0x101 (mon1), group list empty; R is dlg or the case "alt" record; state is 0 or the
// case "vec" record; queue count 0..3, capacity 0..4.
const (
	dscArena    = 0x20000000
	dscD        = dscArena + 0x800 // case object D (argument dlg)
	dscAlt      = dscArena + 0xa00 // case object alt (other panel record)
	dscG        = dscArena         // case object G (*0x649f1c)
	dscChars    = dscArena + 0x2000
	dscNpcList  = dscArena + 0x5100
	dscNpc0     = 0x200055f0 // partyNPC0, as fixed in npcAttitudeNode
	dscNpc1     = 0x200059d8 // partyNPC1
	dscBody0    = dscArena + 0x6800
	dscBody1    = dscArena + 0x6a00
	dscVec      = dscArena + 0x6d00
	dscQueue    = dscArena + 0x6e00
	dscHeap     = 0x50000000 // emu builtin bump heap
	dscMon0     = 0x100      // fixture monster ids
	dscSlots    = 2          // 0x56c78c
	dscStates   = 0x13       // 0x56c771: slots 0..0x12
	dscRevive   = 0xf        // 0x56c72e
	dscReviveHP = 10         // 0x56c765
	dscCalls    = dscSlots * dscStates
)

var dscIn, dscOut []string
var dscIdx = map[string]int{}

func init() {
	n := []string{"a1", "a2", "a3", "a4", "vec0", "vec1", "vec2"}
	for _, p := range []string{"dlg", "alt"} {
		n = append(n, p+".kind", p+".linked", p+".available", p+".monsterID")
	}
	n = append(n, "dlg.baseAttitude", "panel.1d4", "panel.1ed0", "panel.1ed1", "panel.1ed2", "panel.1f9",
		"panel.252", "panel.25c", "g68f0f9", "g683f97", "g683f9b", "g6874ca", "voice.44", "eventStamp",
		"queueCount", "queueCapacity", "queueEntry0", "queueEntry1", "queueEntry2", "queueEntry3",
		"def.54", "def.56", "def.2ea", "def.5f", "def.6e")
	for k := 0; k < 21; k++ {
		n = append(n, fmt.Sprintf("factionAttitude[%d]", k))
	}
	n = append(n, "mon0.2de", "mon0.8b", "mon1.2de", "mon1.8b", "npcListCount",
		"slot0.active", "slot0.npcIndex", "slot1.active", "slot1.npcIndex")
	for s := 0; s < 2; s++ {
		n = append(n, fmt.Sprintf("npc%d.kind", s), fmt.Sprintf("npc%d.unavailable", s), fmt.Sprintf("npc%d.faction", s))
	}
	for c := 0; c < dscSlots; c++ {
		n = append(n, fmt.Sprintf("char%d.b01", c), fmt.Sprintf("char%d.b11", c))
		for k := 0; k < dscStates; k++ {
			n = append(n, fmt.Sprintf("char%d.s%d", c, k))
		}
	}
	o := []string{"g68f0f9", "panel.25c", "panel.252", "g683f97", "panel.1ed0", "panel.1ed1", "panel.1ed2",
		"panel.1f9", "char0.b11", "char1.b11", "eventKind", "eventStamp", "queueCount", "queueCapacity",
		"queuePointer", "queueEntry0", "queueEntry1", "queueEntry2", "queueEntry3", "heap28", "heap2c", "heap30",
		"heap34", "n523330"}
	for _, a := range []string{"who", "cond"} {
		for k := 0; k < dscCalls; k++ {
			o = append(o, "r523330_"+a+strconv.Itoa(k))
		}
	}
	o = append(o, "r523330_flag", "n56d030", "r56d030_1", "r56d030_2", "n577290", "r577290_1",
		"n4c6240", "r4c6240_1", "r4c6240_2", "n56aa30", "n476f90", "r476f90_2",
		"n56cad0", "r56cad0_1", "r56cad0_2", "r56cad0_3", "n575810", "r575810_1", "n52e3b0")
	dscIn, dscOut = n, o
	for i, s := range n {
		dscIdx[s] = i
	}
	Nodes["0056c6d0"] = Node{Addr: "0056c6d0", In: dscIn, Out: dscOut,
		F:   dialogueStatusCleanupNode,
		Emu: func(in []int64, r Rand) []int64 { return dscRun(in, r, false) }}
}

func dialogueStatusCleanupNode(in []int64, r Rand) []int64 { return dscRun(in, r, true) }

func dscRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(dscIn) {
		panic("dialogue status cleanup: wrong input count")
	}
	v := func(n string) int64 {
		i, ok := dscIdx[n]
		if !ok {
			panic("dialogue status cleanup: unknown input " + n)
		}
		return in[i]
	}
	u8 := func(n string) int64 { return v(n) & 0xff }
	u32 := func(n string) int64 { return int64(uint32(v(n))) }
	o := map[string]int64{}
	for _, k := range []string{"panel.252", "g683f97", "panel.1f9"} {
		o[k] = u8(k)
	}
	for _, k := range []string{"panel.1ed0", "panel.1ed1", "panel.1ed2"} {
		o[k] = u32(k)
	}
	var b01, b11 [dscSlots]int64
	var st [dscSlots][20]int64
	for c := 0; c < dscSlots; c++ {
		p := "char" + strconv.Itoa(c) + "."
		b01[c], b11[c] = u32(p+"b01"), u32(p+"b11")
		for k := 0; k < dscStates; k++ {
			st[c][k] = u32(p + "s" + strconv.Itoa(k))
		}
	}
	event, flag := int64(int32(v("a2"))), v("a3")
	panelRec, stamp := u32("panel.1d4"), u32("eventStamp")
	h := &dgsrHeapMem{next: dscHeap, mem: map[int64]byte{}}
	qCount, qCap := v("queueCount"), v("queueCapacity")
	if qCount < 0 || qCount > 3 || qCap < 0 || qCap > 4 {
		panic("dialogue status cleanup: queue outside the fixture caps")
	}
	q := [4]int64{u32("queueEntry0"), u32("queueEntry1"), u32("queueEntry2"), u32("queueEntry3")}
	queuePtr := int64(dscQueue)

	post := func(kind int64) { // 0x528830(kind, 0, 0, 0)
		ev := h.alloc(0x24)
		h.wr(ev, 4, kind)
		h.wr(ev+0x20, 4, stamp)
		n := int32(qCount)
		if n+1 > int32(qCap) { // signed, 0x528880
			p := h.alloc(4 * int64(n+1))
			for i := int32(0); i < n; i++ {
				h.wr(p+4*int64(i), 4, q[i])
			}
			h.wr(p+4*int64(n), 4, ev)
			queuePtr, qCap = p, int64(n+1)
		} else {
			q[n] = ev
		}
		qCount++
	}
	// npcAt is 0x50b800(id) followed by the +0x2e read: the NPC record index 0/1.
	npcAt := func(id int64) int {
		cnt := int32(v("npcListCount"))
		if cnt < 1 || cnt > 2 {
			panic("dialogue status cleanup: NPC list count outside the fixture cap")
		}
		k := int32(0)
		if int32(id) < cnt {
			if int32(id) < 0 {
				panic("dialogue status cleanup: negative NPC id (reads before the list)")
			}
			k = int32(id)
		}
		if u8(fmt.Sprintf("npc%d.unavailable", k)) != 0 {
			panic("dialogue status cleanup: 0x50b800 returned 0 and the original faults on +0x2e")
		}
		return int(k)
	}
	// rec is the prefix of the record at address a (dlg or alt).
	rec := func(a int64) string {
		switch a {
		case dscD:
			return "dlg."
		case dscAlt:
			return "alt."
		}
		panic("dialogue status cleanup: panel record outside the fixture")
	}
	// mon is 0x50a3c0(record): monster index 0/1, or -1 for 0.
	mon := func(p string) int {
		if u8(p+"linked") == 0 || u8(p+"available") == 0 {
			return -1
		}
		id := u32(p + "monsterID")
		if id != dscMon0 && id != dscMon0+1 {
			panic("dialogue status cleanup: srAssertFail 0x4e5550, monster id not found")
		}
		return int(id - dscMon0)
	}
	attitude := func() []int64 { // 0x50a280(dlg) input vector (0050a280.json order)
		a := make([]int64, 44)
		a[0], a[1], a[2] = u8("dlg.baseAttitude"), u8("dlg.linked"), u8("dlg.available")
		if m := mon("dlg."); m >= 0 {
			p := "mon" + strconv.Itoa(m) + "."
			a[3], a[4] = u8(p+"2de"), u32(p+"8b")
		}
		a[5], a[6], a[7] = u8("def.5f"), u8("def.54"), u8("def.6e")
		for k := 0; k < 21; k++ {
			a[8+k] = v(fmt.Sprintf("factionAttitude[%d]", k))
		}
		a[29], a[30] = dscNpcList, v("npcListCount")
		for s := 0; s < dscSlots; s++ {
			p := "slot" + strconv.Itoa(s) + "."
			a[31+3*s], a[32+3*s], a[33+3*s] = u8(p+"active"), b11[s], v(p+"npcIndex")
			a[37+3*s], a[38+3*s], a[39+3*s] = []int64{dscNpc0, dscNpc1}[s], u8(fmt.Sprintf("npc%d.unavailable", s)), u8(fmt.Sprintf("npc%d.faction", s))
		}
		a[43] = u32("dlg.monsterID")
		return a
	}
	call := func(name string, args ...int64) { // address-stubbed callee: count and last arguments
		o["n"+name]++
		for i, x := range args {
			o["r"+name+"_"+strconv.Itoa(i+1)] = int64(uint32(x))
		}
	}

	// 1.
	o["g68f0f9"], o["panel.25c"] = 0, 0
	// 2. revival of present NPC members.
	if k := u8("dlg.kind"); k == 0xf || k == 0x12 {
		for i := 0; i < dscSlots; i++ {
			p := "slot" + strconv.Itoa(i) + "."
			if u8(p+"active") == 0 {
				continue
			}
			nk := u8(fmt.Sprintf("npc%d.kind", npcAt(v(p+"npcIndex"))))
			if (nk != 0x10 && nk != 0x11) || uint32(b01[i]) < dscRevive {
				continue
			}
			for s := 0; s < dscStates; s++ {
				if st[i][s] == 0 {
					continue
				}
				c := o["n523330"]
				if c >= dscCalls {
					panic("dialogue status cleanup: more 0x523330 calls than recorded")
				}
				o["r523330_who"+strconv.Itoa(int(c))], o["r523330_cond"+strconv.Itoa(int(c))] = int64(i), int64(s)
				o["r523330_flag"], o["n523330"] = 0, c+1
				if real {
					w := make([]int64, ccrLen)
					w[ccrCharBase], w[ccrWho], w[ccrCond], w[ccrNotify] = dscChars, int64(i), int64(s), 0
					for j := 0; j < dscSlots; j++ {
						w[ccrPres+j] = u8("slot" + strconv.Itoa(j) + ".active")
					}
					w[ccrWhoPres] = u8(p + "active")
					copy(w[ccrSlot:ccrSlot+20], st[i][:])
					w[ccrB01], w[ccrB11] = b01[i], b11[i]
					for t := 0; t < 3; t++ { // formations: nobody placed (slots and regions 255, counts 0)
						f := w[ccrPC+5+2*pcMembers+t*pcFormLen:][:pcFormLen]
						for k := range f {
							if k < 15 || k >= 20 && (k-20)%4 != 3 {
								f[k] = 0xff
							}
						}
					}
					res := characterConditionRemoveNode(w, r)
					copy(st[i][:], res[:20])
				}
				if s == 0x12 {
					b11[i] = dscReviveHP
				}
			}
		}
	}
	// 3.
	if int32(v("voice.44")) > 0 {
		call("52e3b0")
	}
	// 4.
	if u8("g683f9b") == 0 {
		call("56d030", dscD, flag)
		if real {
			w := make([]int64, 110)
			a := attitude()
			copy(w, a[:29])
			w[29], w[30], w[31], w[32], w[33] = a[30], a[32], a[33], a[35], a[36]
			copy(w[34:40], a[37:43])
			w[40] = a[43]
			w[65], w[66], w[73], w[74] = b01[0], b01[1], a[31], a[34]
			res := dialoguePartyRepresentativeNode(w, r)
			if res[2] != dialogueRepObject || res[28] != dialogueRepObject {
				panic("dialogue status cleanup: 0x56d030 did not record its object")
			}
			panelRec, stamp = dscD, dscD
		}
	}
	finish := func() []int64 {
		o["char0.b11"], o["char1.b11"] = b11[0], b11[1]
		o["eventKind"], o["eventStamp"] = h.rd(dscHeap, 4), h.rd(dscHeap+0x20, 4)
		o["queueCount"], o["queueCapacity"], o["queuePointer"] = qCount, qCap, queuePtr
		for i := 0; i < 4; i++ {
			o["queueEntry"+strconv.Itoa(i)] = q[i]
			o[fmt.Sprintf("heap%x", 0x28+4*i)] = h.rd(dscHeap+0x28+4*int64(i), 4)
		}
		res := make([]int64, len(dscOut))
		for i, n := range dscOut {
			res[i] = o[n]
		}
		return res
	}
	// 5.
	if v("a4")&0xff != 0 {
		call("56cad0", dscD, u32("a1"), 1) // 0x56ca42 reads the state argument after the push
		return finish()
	}
	tail := func() []int64 {
		p := rec(panelRec)
		bodies := []int64{dscBody0, dscBody1}
		if k := u8(p + "kind"); k != 0x84 && k != 0x85 {
			if m := mon(p); m >= 0 {
				call("4c6240", bodies[m], 0)
			}
		}
		call("56aa30")
		o["panel.252"], o["g683f97"], o["panel.25c"] = 1, 1, 0
		if mon(p) >= 0 {
			o["n476f90"], o["r476f90_2"] = o["n476f90"]+1, 0 // first argument: stack position vector
		}
		return finish()
	}
	evOrZero := event
	if event == -1 {
		evOrZero = 0
	}
	// 6.
	e54, e2ea := u8("def.54"), u8("def.2ea")
	switch {
	case e2ea != 0:
		if s := u32("a1"); s != 0 {
			if s != dscVec {
				panic("dialogue status cleanup: state pointer outside the fixture")
			}
			o["panel.1ed0"], o["panel.1ed1"], o["panel.1ed2"] = u32("vec0"), u32("vec1"), u32("vec2")
			if u8("g6874ca") != 0 {
				o["panel.1f9"] = 1
			}
			call("575810", dscG+0x1ed)
		} else if event == -1 {
			call("577290", 1)
		} else {
			post(int64(uint32(event)))
		}
		return tail()
	case e54 == 0:
		cls := int64(0)
		switch att := int8(npcAttitudeNode(attitude(), r)[0]); {
		case att < 0x21:
			cls = 2
		case att < 0x42:
			cls = 1
		}
		if cls == 2 && u8("def.56") == 0 {
			post(0x18)
			return finish()
		}
		if flag&0xff == 0 {
			call("56cad0", dscD, u32("a1"), 0)
			return finish()
		}
	}
	post(int64(uint32(evOrZero)))
	return tail()
}
