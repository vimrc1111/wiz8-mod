package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// character_equipment_removal 0x5223a0(char): move every equipped item that is
// not locked out of the equipment slots (the 12 slots at char+0xf5d, 12 bytes each; a
// parallel set of 8 at char+0x1029) into the shared pack (build/wiz8.asm:
// 5223a0..522637, 521ef0, 51fe30). Character record stride 0x1862 at *0x685174.
//
// Loop 1 (slot k < 12, item id at +0): skip an empty slot (-1). The slot is
// processed when the definition byte +0x8c is 0, or the record byte +0xa is set, or
// the gate dword gate[k] (0x648c5c, read from the exe) is -1; otherwise only when
// char+0xa49 is non-zero. Loop 2 (k < 8) processes the record of the second set when
// the first set's slot k still holds an item; the definition passed to the message
// call is that of the first set's id.
// A processed record goes through 0x521ef0(rec, 0, 0):
//   - definition kind +0x66 == 1 and a pack of pkN entries: 0x51f900(pk[idx], rec, &flag)
//     for idx = 0.. until one returns a non-zero low byte (merged; 0x4edd20 follows
//     unconditionally and the record at idx is post-processed). A partial merge sets
//     the flag byte; a set flag runs 0x4edd20 once after the loop;
//   - else when pkN < 500 the pack shifts up by one entry, entry 0 receives the
//     record through 0x51fe30(pk0, rec, 0, 1) (source cleared to id -1, 0x520d10 on
//     source and destination), pkN++, 0x4edd20, and entry 0 is post-processed;
//   - post-processing: mode 6 sets m69+0xfc |= 0x7fc00000; story flags by item id
//     (0x522640); voice lines by item id (0x5227d0, a2 = 1);
//   - a full pack without a merge returns 0 and the record goes to the hand: the held
//     item is saved when the hand mark is set, the markers become (-1, 0xff, -1),
//     0x55f1e0 clears the hand, 0x51fe30(hand, rec, 0, 1) fills it (0x55f160 raises
//     the cursor graphic, cursor state 7), 0x4f7610(0) or 0x55f260(strs[0x13bc], 0, 1, 0)
//     by definition +0x41 bit 1, the saved hand is restored, 0x58aad0(0, strs[0x1f4c],
//     def of the first set's id) reports.
//
// Emu address-stubs the D4 callee 0x51f900 (ret and flag byte from case inputs) and
// 0x4edd20 (only counted), per decisions/2026-10-09-compositional-stubs.md.
// F calls the real itemStackMergeNode for 0x51f900 and applies its effects; 0x4edd20
// has no output in this node, so F only counts it (its body needs the party state
// that is not part of this node).
// Registered 排除 stubs: 0x4f7610, 0x55f260, 0x58aad0, 0x52e690, 0x427ab0, 0x427ff0,
// 0x518150 (ret from input) and 0x520d10; arguments and counts are outputs.
//
// Caps (calc panics past them): item ids 0..639 (definition table, icon cache),
// icon handle < 1000, pack entries observed: 8 (a merge at idx >= 8 panics), the
// 0x51f900 call budget 520 (a pack scan longer than that), event log 24 entries,
// NPC list 2, char index < 8.
const (
	cerSlots1  = 12
	cerSlots2  = 8
	cerPK      = 8
	cerNEV     = 24
	cerNRet    = 520
	cerIDs     = 640
	cerHandles = 1000
	cerChars   = 8
	cerNPCs    = 2
	cerFlags   = 12
	cerPackCap = 500
	cerChar    = 0x20001000 // arena base of the character table
	cerCharStr = 0x1862
	cerDefs    = 0x20020000
	cerDefStr  = 269
	cerHandPtr = 0x6874cb
	cerPackPtr = 0x685191
	cerGate    = 0x648c5c // 12 dwords: slot gate values
	cerIconW   = 0x6448cc // u16 per icon handle, stride 8
)

// story flag ids touched by 0x522640/0x5227d0, in the order of the case's flag inputs.
var cerFlagIDs = [cerFlags]int{0x30, 0x166, 0x22, 0x165, 0x31, 0x167, 0x182, 0x31e, 0x242, 0x24e, 0x24f, 0xe5}

var cerStubNames = []string{"4f7610", "55f260", "58aad0", "52e690", "427ab0", "427ff0", "518150", "520d10", "4edd20", "51f900"}
var cerStubArgs = [][]int{{1}, {1, 2, 3, 4}, {1, 2, 3}, {1, 2, 3, 4, 5}, {2, 3, 4}, {}, {}, {1, 2, 3}, {}, {1, 2}}

var cerExe, cerExeErr = cerLoadExe()

type cerExeTab struct {
	gate               [cerSlots1]uint32
	c5ed914, c5ed8c8   uint32 // 0x5ed914, 0x5ed8c8: trailing voice-call constants
	v239, v242, v243   uint32 // 0x5ee6fc, 0x5ee640, 0x5ee644
	v244a, v244b, v264 uint32 // 0x5ee648, 0x5ee64c, 0x5ee690
	iconW              [cerHandles]uint16
}

func cerLoadExe() (t cerExeTab, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	u32 := func(va uint32) uint32 {
		o, e := peOffset(b, va)
		if e != nil || len(b)-o < 4 {
			err = fmt.Errorf("equipment removal: cannot read %#x", va)
			return 0
		}
		return binary.LittleEndian.Uint32(b[o:])
	}
	for k := range t.gate {
		t.gate[k] = u32(cerGate + 4*uint32(k))
	}
	t.c5ed914, t.c5ed8c8 = u32(0x5ed914), u32(0x5ed8c8)
	t.v239, t.v242, t.v243 = u32(0x5ee6fc), u32(0x5ee640), u32(0x5ee644)
	t.v244a, t.v244b, t.v264 = u32(0x5ee648), u32(0x5ee64c), u32(0x5ee690)
	for h := range t.iconW {
		o, e := peOffset(b, cerIconW+8*uint32(h))
		if e != nil || len(b)-o < 2 {
			err = fmt.Errorf("equipment removal: cannot read icon word %d", h)
			return t, err
		}
		t.iconW[h] = binary.LittleEndian.Uint16(b[o:])
	}
	return t, err
}

type cerRec struct {
	id   int32
	tail uint64 // record bytes +4..+11 (w1 | w2<<32)
}

type cerNPC struct {
	sub int32
	b2e uint8
	hid uint8
	w   [10]int64
}

// slot index u: 0..11 first set, 12..19 second set.
type cerSt struct {
	t1              [cerSlots1]cerRec
	t2              [cerSlots2]cerRec
	d8c             [cerSlots1]uint8
	d41, d66        [cerSlots1 + cerSlots2]uint8
	ih              [cerSlots1 + cerSlots2]uint32
	a49, c0b01      uint32
	hand            cerRec
	hMark           uint8
	m0              uint32
	m4              uint8
	m5              uint16
	cur, cur2, cur3 uint32
	pkN             uint32
	pk              [cerPK]cerRec
	pkD66, pkD6b    [cerPK]uint8
	mode, m69fc     uint32
	b73c            uint8
	g6a48           uint32
	g75fa           uint8
	g7603           uint32
	flag            [cerFlags]uint8
	evStamp         uint32
	evCnt, evCap    uint32
	evLive          bool
	ev              [cerNEV][3]uint32
	npcN            int
	npc             [cerNPCs]cerNPC
	q0, q106        uint8
	qi1, qi2        int32
	s13bc, s1f4c    uint32
	r518150, flagW  uint32
	seq             [cerNRet]uint32
	n51f            int // 0x51f900 calls so far (cumulative over the run)
	flagLocal       uint8
	argp            uint32
	real            bool
	cnt             [10]uint32
	arg             [10][6]uint32
}

func cerNames(in bool) []string {
	var n []string
	add := func(f string, a ...any) { n = append(n, fmt.Sprintf(f, a...)) }
	if in {
		add("argp")
		add("ci")
		for k := 0; k < cerSlots1; k++ {
			for _, f := range []string{"id", "w1", "w2", "d8c", "d41", "d66", "ih"} {
				add("t1%s%d", f, k)
			}
		}
		for k := 0; k < cerSlots2; k++ {
			for _, f := range []string{"id", "w1", "w2", "d41", "d66", "ih"} {
				add("t2%s%d", f, k)
			}
		}
		add("a49")
		add("c0b01")
		for _, f := range []string{"hId", "hW1", "hW2", "hMark", "m0", "m4", "m5", "cur", "cur2", "cur3", "pkN"} {
			add("%s", f)
		}
		for k := 0; k < cerPK; k++ {
			add("pk%did", k)
			add("pk%dw1", k)
			add("pk%dw2", k)
		}
		for k := 0; k < cerPK; k++ {
			add("pk%dd66", k)
			add("pk%dd6b", k)
		}
		for _, f := range []string{"mode", "m69fc", "b73c", "g6a48", "g75fa", "g7603"} {
			add("%s", f)
		}
		for _, id := range cerFlagIDs {
			add("f%x", id)
		}
		add("evStamp")
		add("evCnt")
		add("evCap")
		add("npcN")
		for j := 0; j < cerNPCs; j++ {
			add("e%dsub", j)
			add("e%db2e", j)
			add("e%dhid", j)
			for c := 0; c < 10; c++ {
				add("e%dw%d", j, c)
			}
		}
		for _, f := range []string{"q0", "qi1", "q106", "qi2", "s13bc", "s1f4c", "r518150", "flagW"} {
			add("%s", f)
		}
		for k := 0; k < cerNRet; k++ {
			add("r51f%d", k)
		}
		return n
	}
	for k := 0; k < cerSlots1; k++ {
		add("t1id%d", k)
		add("t1t%d", k)
	}
	for k := 0; k < cerSlots2; k++ {
		add("t2id%d", k)
		add("t2t%d", k)
	}
	for _, f := range []string{"hId", "hT", "hMark", "m0", "m4", "m5", "cur", "cur2", "cur3", "pkN"} {
		add("%s", f)
	}
	for k := 0; k < cerPK; k++ {
		add("pk%did", k)
		add("pk%dt", k)
	}
	for _, f := range []string{"m69fc", "g75fa", "g7603"} {
		add("%s", f)
	}
	for _, id := range cerFlagIDs {
		add("f%x", id)
	}
	add("evCnt")
	add("evCap")
	for k := 0; k < cerNEV; k++ {
		add("ev%dstamp", k)
		add("ev%did", k)
		add("ev%dbl", k)
	}
	for j := 0; j < cerNPCs; j++ {
		for c := 0; c < 10; c++ {
			add("e%dw%d", j, c)
		}
	}
	for j, a := range cerStubNames {
		add("n%s", a)
		for _, i := range cerStubArgs[j] {
			add("n%sa%d", a, i)
		}
	}
	return n
}

var cerInLen = len(cerNames(true))

func init() {
	Nodes["005223a0"] = Node{Addr: "005223a0", In: cerNames(true), Out: cerNames(false),
		F:   characterEquipmentRemovalNode,
		Emu: func(in []int64, r Rand) []int64 { return cerRun(in, r, false) }}
}

func characterEquipmentRemovalNode(in []int64, r Rand) []int64 { return cerRun(in, r, true) }

func cerTail(w1, w2 uint32) uint64 { return uint64(w1) | uint64(w2)<<32 }

func cerParse(in []int64, s *cerSt) {
	p := 0
	next := func() int64 { p++; return in[p-1] }
	s.argp = uint32(next())
	ci := next()
	if ci < 0 || ci >= cerChars || s.argp != uint32(cerChar+cerCharStr*ci) {
		panic("equipment removal: character pointer outside the modelled table")
	}
	for k := 0; k < cerSlots1; k++ {
		id := int32(next())
		s.t1[k] = cerRec{id, cerTail(uint32(next()), uint32(next()))}
		s.d8c[k], s.d41[k], s.d66[k] = uint8(next()), uint8(next()), uint8(next())
		s.ih[k] = uint32(next())
	}
	for k := 0; k < cerSlots2; k++ {
		id := int32(next())
		s.t2[k] = cerRec{id, cerTail(uint32(next()), uint32(next()))}
		s.d41[12+k], s.d66[12+k] = uint8(next()), uint8(next())
		s.ih[12+k] = uint32(next())
	}
	s.a49, s.c0b01 = uint32(next()), uint32(next())
	s.hand.id = int32(next())
	w1, w2 := uint32(next()), uint32(next())
	s.hand.tail = cerTail(w1, w2)
	s.hMark, s.m0, s.m4, s.m5 = uint8(next()), uint32(next()), uint8(next()), uint16(next())
	s.cur, s.cur2, s.cur3, s.pkN = uint32(next()), uint32(next()), uint32(next()), uint32(next())
	for k := 0; k < cerPK; k++ {
		s.pk[k].id = int32(next())
		s.pk[k].tail = cerTail(uint32(next()), uint32(next()))
	}
	for k := 0; k < cerPK; k++ {
		s.pkD66[k], s.pkD6b[k] = uint8(next()), uint8(next())
	}
	s.mode, s.m69fc, s.b73c, s.g6a48 = uint32(next()), uint32(next()), uint8(next()), uint32(next())
	s.g75fa, s.g7603 = uint8(next()), uint32(next())
	for k := range s.flag {
		s.flag[k] = uint8(next())
	}
	s.evStamp, s.evCnt, s.evCap = uint32(next()), uint32(next()), uint32(next())
	s.npcN = int(next())
	for j := range s.npc {
		s.npc[j].sub, s.npc[j].b2e, s.npc[j].hid = int32(next()), uint8(next()), uint8(next())
		for c := range s.npc[j].w {
			s.npc[j].w[c] = next()
		}
	}
	s.q0, s.qi1, s.q106, s.qi2 = uint8(next()), int32(next()), uint8(next()), int32(next())
	s.s13bc, s.s1f4c, s.r518150, s.flagW = uint32(next()), uint32(next()), uint32(next()), uint32(next())
	for k := range s.seq {
		s.seq[k] = uint32(next())
	}
	if p != len(in) {
		panic("equipment removal: input layout")
	}
	if s.npcN < 0 || s.npcN > cerNPCs {
		panic("equipment removal: NPC list above the fixture cap")
	}
	if s.evCnt > cerNEV {
		panic("equipment removal: event log above the fixture cap")
	}
	s.evLive = true
}

func cerRun(in []int64, r Rand, real bool) []int64 {
	if cerExeErr != nil {
		panic(cerExeErr)
	}
	if len(in) != cerInLen {
		panic("equipment removal: wrong input count")
	}
	s := &cerSt{real: real}
	cerParse(in, s)
	for k := 0; k < cerSlots1; k++ { // 0x5223b6 loop 1
		rec := &s.t1[k]
		if rec.id == -1 {
			continue
		}
		if !(s.d8c[k] == 0 || uint8(rec.tail>>48) != 0 || cerExe.gate[k] == 0xffffffff || s.a49 != 0) {
			continue
		}
		ptr := s.argp + 0xf5d + 12*uint32(k)
		if s.remove(rec, ptr, k, r) {
			continue
		}
		s.toHand(rec, ptr, k, rec.id)
	}
	for k := 0; k < cerSlots2; k++ { // 0x52251d loop 2
		edi := s.t1[k].id
		if edi == -1 {
			continue
		}
		rec := &s.t2[k]
		ptr := s.argp + 0x1029 + 12*uint32(k)
		if s.remove(rec, ptr, 12+k, r) {
			continue
		}
		s.toHand(rec, ptr, 12+k, edi)
	}
	return s.out()
}

func (s *cerSt) call(j int, a ...uint32) {
	s.cnt[j]++
	for i, v := range a {
		s.arg[j][i+1] = v
	}
}

// remove is 0x521ef0(rec, 0, 0); the result is "the record went to the pack".
func (s *cerSt) remove(rec *cerRec, ptr uint32, u int, r Rand) bool {
	if rec.id < 0 || rec.id >= cerIDs {
		panic("equipment removal: item id outside the definition table cap (or an empty second-set slot, 0x51fe30 assert)")
	}
	merged, idx := false, 0
	s.flagLocal = 0
	if s.d66[u] == 1 {
		for i := uint32(0); i < s.pkN; i++ {
			if s.merge(i, rec, ptr, r) {
				merged, idx = true, int(i)
				break
			}
		}
	}
	switch {
	case merged: // 0x522089 -> 0x521f7f: the refresh runs unconditionally
		s.call(8)
	default:
		if s.flagLocal != 0 { // partial merge
			s.call(8)
		}
		if s.pkN >= cerPackCap { // 0x521f91
			return false
		}
		old := s.pk
		for i := 0; i < int(s.pkN) && i+1 < cerPK; i++ {
			s.pk[i+1] = old[i]
		}
		s.pk[0] = cerRec{-1, 0}
		s.move(&s.pk[0], cerPackPtr, rec, ptr, u)
		s.pkN++
		s.call(8)
		idx = 0
	}
	if s.mode == 6 {
		s.m69fc |= 0x7fc00000
	}
	if idx >= cerPK {
		panic("equipment removal: merged pack entry past the observed pool")
	}
	pk := &s.pk[idx]
	s.story(pk)
	s.voice(pk)
	return true
}

// merge is one 0x51f900(pk[i], rec, &flag) call.
func (s *cerSt) merge(i uint32, rec *cerRec, ptr uint32, r Rand) bool {
	s.call(9, cerPackPtr+12*i, ptr)
	if !s.real {
		if s.n51f >= cerNRet {
			panic("equipment removal: 0x51f900 call budget exhausted")
		}
		ret := s.seq[s.n51f]
		s.n51f++
		s.flagLocal = uint8(s.flagW)
		return uint8(ret) != 0
	}
	if i >= cerPK || s.pkN > ismPool {
		panic("equipment removal: pack entry past the fixture of 0x51f900")
	}
	v := []int64{int64(ismPack + 12*i), ismPB, ismPF, int64(s.pk[i].id)}
	inst := func(id int32, tail uint64) {
		v = append(v, int64(id), int64(tail&0xff), int64(tail>>8&0xff), int64(tail>>16&0xff), int64(tail>>24&0xff), int64(tail>>32))
	}
	inst(-1, 0)
	inst(rec.id, rec.tail)
	inst(-1, 0)
	v = append(v, 0)
	for k := 0; k < ismPool; k++ {
		if k < cerPK {
			inst(s.pk[k].id, s.pk[k].tail)
		} else {
			inst(-1, 0)
		}
	}
	v = append(v, int64(s.pkN), int64(s.m0), int64(s.m4), int64(s.m5), int64(s.cur), int64(s.cur2), int64(s.cur3), 0, 0,
		int64(s.pkD66[i]), int64(s.pkD6b[i]), 0, 0, int64(s.flagLocal))
	o := itemStackMergeNode(v, r)
	rec.id, rec.tail = int32(o[4]), uint64(o[5])
	for k := 0; k < cerPK; k++ {
		s.pk[k] = cerRec{int32(o[9+2*k]), uint64(o[10+2*k])}
	}
	s.pkN = uint32(o[41])
	s.flagLocal = uint8(o[1])
	s.cnt[8] += uint32(o[48])
	if o[49] != 0 {
		a1 := uint32(o[50])
		if a1 == ismPB {
			a1 = ptr
		}
		s.cnt[7] += uint32(o[49])
		s.arg[7][1], s.arg[7][2], s.arg[7][3] = a1, uint32(o[51]), uint32(o[52])
	}
	return uint8(o[0]) != 0
}

// move is 0x51fe30(dst, src, 0, 1) with dst empty.
func (s *cerSt) move(dst *cerRec, dstPtr uint32, src *cerRec, srcPtr uint32, u int) {
	if dst.id != -1 || src.id == -1 {
		panic("equipment removal: 0x51fe30 assert")
	}
	src.tail &^= uint64(0xff) << 48
	*dst = *src
	*src = cerRec{-1, 0}
	s.call(7, srcPtr, 0, 1)
	s.call(7, dstPtr, 0, 1)
	if dstPtr == cerHandPtr {
		s.m0, s.m4, s.m5 = 0xffffffff, 0xff, 0xffff
		s.raise(u)
	}
}

// clearHand is 0x55f1e0.
func (s *cerSt) clearHand() {
	s.hand, s.hMark = cerRec{-1, 0}, 0
	if s.cur != 0xffffffff {
		s.call(4, 0, uint32(cerExe.iconW[0]), 0, 0)
		s.call(5)
		s.cur, s.cur2, s.cur3 = 0xffffffff, 0, 0
	}
}

// raise is 0x55f160(0): show the held item as the cursor.
func (s *cerSt) raise(u int) {
	if s.hand.id == -1 {
		return
	}
	s.hMark = 1
	h := s.ih[u]
	if h >= cerHandles {
		panic("equipment removal: icon handle above the cap")
	}
	s.call(4, 0, uint32(cerExe.iconW[h]), 0, 0)
	s.call(5)
	s.cur = 7
}

// toHand puts a record that did not fit the pack into the hand and reports it.
func (s *cerSt) toHand(rec *cerRec, ptr uint32, u int, edi int32) {
	bl := s.hMark != 0
	saved := s.hand
	s.m0, s.m4, s.m5 = 0xffffffff, 0xff, 0xffff
	s.clearHand()
	s.move(&s.hand, cerHandPtr, rec, ptr, u)
	if s.d41[u]&2 == 0 {
		s.call(0, 0)
	} else {
		s.call(1, s.s13bc, 0, 1, 0)
	}
	if bl {
		s.hand = saved
	}
	s.call(2, 0, s.s1f4c, uint32(cerDefs)+uint32(edi)*cerDefStr)
}

func (s *cerSt) get(id int) uint8 {
	for i, f := range cerFlagIDs {
		if f == id {
			return s.flag[i]
		}
	}
	panic("equipment removal: story flag outside the modelled set")
}

// set is 0x5061a0(id, val, 0).
func (s *cerSt) set(id int, val uint8) {
	old := s.get(id)
	for i, f := range cerFlagIDs {
		if f == id {
			s.flag[i] = val
		}
	}
	if val != old { // 0x5588f0: append to the event log
		need := s.evCnt + 1
		if need > s.evCap { // realloc: later entries are unobservable
			s.evCap, s.evLive = need, false
		}
		if s.evLive {
			if s.evCnt >= cerNEV {
				panic("equipment removal: event log above the fixture cap")
			}
			s.ev[s.evCnt] = [3]uint32{s.evStamp, uint32(id), uint32(val)}
		}
		s.evCnt++
	}
	if id == 0x22 && val != 0 { // 0x506670(0x22, 1): first NPC whose sub is 0x48 loses scripted word 0x242
		for j := 0; j < s.npcN; j++ {
			if s.npc[j].sub == 0x48 {
				for c := range s.npc[j].w {
					for q := uint(0); q < 4; q++ {
						if uint16(s.npc[j].w[c]>>(16*q)) == 0x242 {
							s.npc[j].w[c] |= 0xffff << (16 * q)
							return
						}
					}
				}
				return
			}
		}
		panic("equipment removal: 0x506670 assert, no NPC 0x48")
	}
}

// story is 0x522640(rec).
func (s *cerSt) story(rec *cerRec) {
	switch rec.id {
	case 0x243:
		if s.get(0x30) == 0 {
			s.set(0x30, 1)
		}
		if s.get(0x166) != 0 {
			s.set(0x166, 0)
		}
	case 0x242:
		if s.get(0x22) == 0 {
			s.set(0x22, 1)
		}
		if s.get(0x165) != 0 {
			s.set(0x165, 0)
		}
	case 0x244:
		if s.get(0x31) == 0 {
			s.set(0x31, 1)
		}
		if s.get(0x167) != 0 {
			s.set(0x167, 0)
		}
		s.g75fa, s.g7603 = 1, s.g6a48
	case 0x264:
		if s.get(0x182) == 0 {
			s.set(0x182, 1)
			s.set(0x31e, 1)
		}
	case 0x201:
		s.set(0x242, 0)
	case 0x27c:
		s.set(0x24e, 0)
	case 0x1d2:
		if s.get(0x24f) != 0 {
			s.set(0x24f, 0)
		}
	}
}

// voice is 0x5227d0(rec, 1, 0).
func (s *cerSt) voice(rec *cerRec) {
	if s.mode == 5 {
		return
	}
	s.call(6)
	if s.r518150 == 0xffffffff {
		return
	}
	ch := cerChar + s.r518150*cerCharStr
	if rec.tail>>40&1 != 0 {
		return
	}
	var v uint32
	switch rec.id {
	case 0x239:
		v = cerExe.v239
	case 0x242:
		v = cerExe.v242
	case 0x243:
		v = cerExe.v243
	case 0x244:
		if s.b73c != 0 {
			v = cerExe.v244a
		} else {
			v = cerExe.v244b
		}
	case 0x264:
		v = cerExe.v264
	case 0x27c:
		s.set(0xe5, 1)
		s.fault50b8f0()
		if s.get(0x24e) != 0 {
			panic("equipment removal: 0x5227d0 tail past 0x506280(0x24e), cleared by 0x522640 just before")
		}
		return
	default:
		return
	}
	s.call(3, ch, v, 0, cerExe.c5ed8c8, cerExe.c5ed914)
	rec.tail |= 1 << 40
}

// fault50b8f0 models the one effect of 0x50b8f0(7) here: its result is not used
// (both outcomes return), but the original faults on a hidden NPC.
func (s *cerSt) fault50b8f0() {
	elem := func(i int32) *cerNPC {
		if i < 0 {
			panic("equipment removal: negative NPC index")
		}
		e := &s.npc[0]
		if int(i) < s.npcN {
			e = &s.npc[i]
		}
		if e.hid != 0 {
			panic("equipment removal: 0x50b8f0 faults on a hidden NPC")
		}
		return e
	}
	if s.q0 != 0 {
		if e := elem(s.qi1); e.b2e == 7 && s.c0b01 < 0xf {
			return
		}
	}
	if s.q106 != 0 {
		elem(s.qi2)
	}
}

func (s *cerSt) out() []int64 {
	var o []int64
	add := func(v ...int64) { o = append(o, v...) }
	for k := range s.t1 {
		add(int64(s.t1[k].id), int64(s.t1[k].tail))
	}
	for k := range s.t2 {
		add(int64(s.t2[k].id), int64(s.t2[k].tail))
	}
	add(int64(s.hand.id), int64(s.hand.tail), int64(s.hMark), int64(int32(s.m0)), int64(s.m4), int64(s.m5),
		int64(int32(s.cur)), int64(s.cur2), int64(s.cur3), int64(s.pkN))
	for k := range s.pk {
		add(int64(s.pk[k].id), int64(s.pk[k].tail))
	}
	add(int64(s.m69fc), int64(s.g75fa), int64(s.g7603))
	for _, f := range s.flag {
		add(int64(f))
	}
	add(int64(s.evCnt), int64(s.evCap))
	for _, e := range s.ev {
		add(int64(e[0]), int64(e[1]), int64(uint8(e[2])))
	}
	for j := range s.npc {
		add(s.npc[j].w[:]...)
	}
	for j := range cerStubNames {
		add(int64(s.cnt[j]))
		for _, i := range cerStubArgs[j] {
			add(int64(s.arg[j][i]))
		}
	}
	return o
}
