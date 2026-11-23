package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// cure_and_resurrection 0x54df00(effect, cond, flag): remove condition `cond`
// from every target of a cure/resurrection spell effect; returns 1 when every
// target was fully cured, else 0.
//   - amount = power*power (effect+0xd0, u32 wrap), then the child 0x517f90
//     adds amount*effect+0xd4/100 (low 32 bits of the product, unsigned).
//   - Characters (ids at effect+0xfc, count effect+0xf4, signed; <=0 runs
//     nothing): slot t = character +0xa01+4*cond. t==0 skips. t!=9999 and no
//     flag: decay by amount through 0x5236a0; a slot still non-zero counts the
//     target (count++, result 0, effect+0x125=1; with byte 0x68510c set the
//     0x590950 notice), a cleared slot sets +0x124 and +0x125. Otherwise one
//     R(100) against 100 (cond 0x12 or 1 or flag) or power*15: below removes it
//     through 0x523330(id, cond, 1) and sets +0x124/+0x125; at or above sets
//     result 0. For cond 0x12 a removal also sets +0xb11 = max(1, +0xb0d*s/100)
//     and +0xb1d = max(1, +0xb19*s/100), s = (+0x111 + +0x125)>>2 (u32), and
//     clears word +4 of the object 0x50b800 returns for the member's object id
//     (presence record +0xfa, -1 none). Every removal then draws R(100); below
//     50 calls 0x52e690(character, [0x68c55c], 0, [0x5ed8c8], [0x5ed914]).
//   - Monsters (ids at effect+0xec, count effect+0xe4): duration t =
//     monster +0x57+4*cond. t==0 skips; t!=9999, t>amount and no flag: t -=
//     amount, count++, result 0, +0x125=1. Otherwise R(100) against 100 (cond
//     0x12 or 1, t<=amount or flag) or power*15: below runs 0x523f40(id, cond)
//     and sets +0x125, else result 0.
//   - Tail (byte 0x68510c clear and count>0): byte 0x69b7b8 set becomes 1 via
//     0x5905c0. count==1 with effect+0x90 == 1 / 3 sends the single-target notice
//     (0x590950 / 0x590b40); every tail sets +0x124.
//
// Text pointers come from the table at *0x68c09c: entry 0x6c4/4 and entry
// word(0x61e576+8*cond), the word table read from the player's Wiz8.exe.
//
// Emu address-stubs the D4 callees 0x5236a0 and 0x523330 (ruling 2026-10-09)
// and the 排除 callees 0x52e690 and 0x590b40; their args (up to carMax calls,
// call k in cell k) and counts are outputs. Both call sites ignore eax. The
// 0x5236a0 stub writes the post-decay slot from input decayPost<k> (call k).
// F runs ccdRun (0x5236a0) and the shared ccrRun implementation of 0x523330.
// The removal world retains condition slots, effective attributes, HP/stamina
// maxima and current values across nested deaths and repeated target ids.
// Resurrection reads those rebuilt cells; g686a70 is the condition-9 selector.
// 0x523f40 runs natively in both modes: with HP +0x2b zero nothing changes,
// else the duration is cleared, condition 7 also clears +0x10b and +0x107
// becomes the highest active index 19..1 (monsterConditionDecayNode with an
// unbounded decay).
//
// Fixture caps (panic past them): <=8 ids per array, character ids 0..7,
// monster ids found in a 3-record main list then a 1-record reserve list, the
// object list 0 or the 4-entry fixture list, object ids -1..3 within the list.
// Character id -1 and an unknown monster id end in srAssertFail (0x54df75,
// 0x54e1e4); indexes past the count (0x54df68, 0x54e1bb) cannot occur.
const (
	carMax      = 8
	carMembers  = 8
	carMain     = 3
	carMons     = carMain + 1
	carObjs     = 4
	carCharF    = 29 // pres, objId, s0..s19, b09, 111, 125, b0d, b19, b11, b1d
	carMonF     = 24 // id, hp, d0..d19, poison, primary
	carChars    = 46
	carMonBase  = carChars + carMembers*carCharF
	carPost     = carMonBase + carMons*carMonF
	carG686a70  = carPost + carMax // F only: 0x523330's condition-9 item-use selector
	carIn       = carG686a70 + 1
	carArena    = 0x20000000
	carCharSize = 0x1862
	carMonAddr  = carArena + 0xf000 // fixture monster k at +0x400*k
	carObjAddr  = carArena + 0x16200
	carObjList  = carArena + 0x16100
	carText     = carArena + 0x17000 // fixture "X" string per text index
)

var carTextIndex, carTextIndexErr = loadCarTextIndex()

// loadCarTextIndex reads the 20 words at exe 0x61e576 (stride 8): the message
// table index per condition.
func loadCarTextIndex() ([20]uint16, error) {
	var t [20]uint16
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("cure_and_resurrection: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	o, err := peOffset(b, 0x61e576)
	if err != nil {
		return t, err
	}
	if len(b)-o < 8*len(t) {
		return t, fmt.Errorf("cure_and_resurrection: text index table truncated")
	}
	for i := range t {
		t[i] = binary.LittleEndian.Uint16(b[o+8*i:])
	}
	return t, nil
}

// carTextAddr is the fixture string the case's 0x68c09c table holds for index i.
func carTextAddr(i int) uint32 {
	switch {
	case i == 433 || i == 434:
		return carText + 4*uint32(i-433)
	case i >= 843 && i <= 919:
		return carText + 4*uint32(i-840)
	}
	panic(fmt.Sprintf("cure_and_resurrection: text index %d outside the fixture table", i))
}

func init() {
	var in, out []string
	add := func(l *[]string, f string, a ...any) { *l = append(*l, fmt.Sprintf(f, a...)) }
	in = append(in, "cond", "flag", "power", "percent", "charCount")
	for j := 0; j < carMax; j++ {
		add(&in, "charId%d", j)
	}
	in = append(in, "monsterCount")
	for j := 0; j < carMax; j++ {
		add(&in, "monsterId%d", j)
	}
	in = append(in, "noticeKind", "noticeChar", "noticeMonster", "eff124", "eff125",
		"g68510c", "g69b7b8", "g68c55c", "g5ed8c8", "g5ed914", "objListPtr", "objCount")
	for k := 0; k < carObjs; k++ {
		add(&in, "objEntry%d", k)
	}
	for k := 0; k < carObjs; k++ {
		add(&in, "obj%dHidden", k)
		add(&in, "obj%dWord4", k)
	}
	for k := 0; k < carMembers; k++ {
		add(&in, "pres%d", k)
		add(&in, "objId%d", k)
		for s := 0; s < 20; s++ {
			add(&in, "c%ds%d", k, s)
		}
		for _, f := range []string{"b09", "111", "125", "b0d", "b19", "b11", "b1d"} {
			add(&in, "c%d_%s", k, f)
		}
	}
	for k := 0; k < carMons; k++ {
		add(&in, "m%dId", k)
		add(&in, "m%dHp", k)
		for s := 0; s < 20; s++ {
			add(&in, "m%dd%d", k, s)
		}
		add(&in, "m%dPoison", k)
		add(&in, "m%dPrimary", k)
	}
	for k := 0; k < carMax; k++ {
		add(&in, "decayPost%d", k)
	}
	in = append(in, "g686a70")
	out = append(out, "result", "eff124", "eff125", "g69b7b8")
	for k := 0; k < carMembers; k++ {
		add(&out, "c%d_b11", k)
		add(&out, "c%d_b1d", k)
	}
	for k := 0; k < carObjs; k++ {
		add(&out, "obj%dWord4", k)
	}
	for k := 0; k < carMons; k++ {
		for s := 0; s < 20; s++ {
			add(&out, "m%dd%d", k, s)
		}
		add(&out, "m%dPoison", k)
		add(&out, "m%dPrimary", k)
	}
	for _, c := range []struct {
		name    string
		args, n int
	}{{"decay", 3, carMax}, {"remove", 3, carMax}, {"voice", 5, carMax}, {"monsterNotice", 3, 1}} {
		for a := 1; a <= c.args; a++ {
			for k := 0; k < c.n; k++ {
				add(&out, "%s.arg%d.call%d", c.name, a, k)
			}
		}
		out = append(out, c.name+".count")
	}
	for a := 1; a <= 5; a++ {
		for k := carMax; k < carVoiceMax; k++ {
			add(&out, "voice.arg%d.call%d", a, k)
		}
	}
	Nodes["0054df00"] = Node{Addr: "0054df00", In: in, Out: out, F: cureAndResurrectionNode,
		Emu: func(in []int64, r Rand) []int64 { return carRun(in, r, false) }}
}

func cureAndResurrectionNode(in []int64, r Rand) []int64 { return carRun(in, r, true) }

func carRun(in []int64, r Rand, real bool) []int64 {
	if carTextIndexErr != nil {
		panic(carTextIndexErr)
	}
	if len(in) != carIn {
		panic("cure_and_resurrection: wrong input count")
	}
	for _, v := range in {
		if v < -2147483648 || v > 0xffffffff {
			panic("cure_and_resurrection: fields must fit 32 bits")
		}
	}
	u := func(i int) uint32 { return uint32(in[i]) }
	cond := u(0)
	if cond >= 20 {
		panic("cure_and_resurrection: condition outside the 20 slots") // no caller passes it (max 0x12)
	}
	flag := u(1)&0xff != 0
	power := u(2)
	amount := power * power
	amount += amount * u(3) / 100 // 0x517f90
	nChar, nMon := int32(u(4)), int32(u(13))
	if nChar > carMax || nMon > carMax {
		panic("cure_and_resurrection: fixture caps target arrays at 8 ids")
	}
	result := uint32(1)
	eff124, eff125 := u(25)&0xff, u(26)&0xff
	g69b7b8 := u(28) & 0xff
	var count uint32

	type member struct {
		pres, b09, b11, b1d uint32
		objID               int32
		slot                []int64
	}
	var chars [carMembers]member
	for k := range chars {
		o := carChars + k*carCharF
		c := &chars[k]
		c.pres, c.objID = u(o), int32(u(o+1))
		c.slot = append([]int64(nil), in[o+2:o+22]...)
		c.b09, c.b11, c.b1d = u(o+22), u(o+27), u(o+28)
	}
	var word4 [carObjs]uint32
	for k := range word4 {
		word4[k] = u(38+2*k+1) & 0xffff
	}
	var mons [carMons][]int64 // d0..d19, poison, primary
	for k := range mons {
		o := carMonBase + k*carMonF
		mons[k] = append([]int64(nil), in[o+2:o+24]...)
	}
	findMon := func(id uint32) int {
		for k := 0; k < carMons; k++ {
			if u(carMonBase+k*carMonF) == id {
				return k
			}
		}
		panic("cure_and_resurrection: monster id in neither list (srAssertFail)") // 0x54e1e4
	}
	var decay, remove, notice stbCalls
	var voice carVoiceCalls
	var world *ccrWorld
	if real {
		world = newCCRWorld(carRemoveInput(chars[0].slot, 0, cond, chars[0].pres, in[carG686a70]))
		for k := range chars {
			c := &chars[k]
			world.inputs[k] = carRemoveInput(c.slot, uint32(k), cond, c.pres, in[carG686a70])
			world.present[k] = c.pres & 255
			ch := world.char(k)
			world.m.w8(ch+4, 1)
			for j, value := range c.slot {
				world.m.w32(ch+0xa01+4*uint32(j), uint32(value))
			}
			o := carChars + k*carCharF
			for j, off := range []uint32{0xb09, 0x111, 0x125, 0xb0d, 0xb19, 0xb11, 0xb1d} {
				world.m.w32(ch+off, u(o+22+j))
			}
		}
		world.m.w32(0x5ed8c8, u(30))
		world.m.w32(0x5ed914, u(31))
		world.voice = voice.add
		world.died = func(who int) {
			if chars[who].objID != -1 {
				if k := carObject(in, chars[who].objID); k >= 0 {
					word4[k] = 1
				}
			}
		}
	}
	syncMember := func(id uint32) {
		if !real {
			return
		}
		c, ch := &chars[id], world.char(int(id))
		for j := range c.slot {
			c.slot[j] = int64(world.m.r32(ch + 0xa01 + 4*uint32(j)))
		}
		c.b09, c.b11, c.b1d = world.m.r32(ch+0xb09), world.m.r32(ch+0xb11), world.m.r32(ch+0xb1d)
	}
	tblStr := carTextAddr(int(carTextIndex[cond]))

	for j := int32(0); j < nChar; j++ {
		id := u(5 + int(j))
		if id == 0xffffffff {
			panic("cure_and_resurrection: character id -1 (srAssertFail 0x54df75)")
		}
		if id >= carMembers {
			panic("cure_and_resurrection: character outside the 8-member fixture")
		}
		syncMember(id)
		c := &chars[id]
		t := uint32(c.slot[cond])
		if t == 0 {
			continue
		}
		if t != 9999 && !flag {
			decay.add(id, cond, amount)
			if real && t <= amount {
				// 0x5236a0's removal path: only its 0x523330 call changes state.
				if c.pres&0xff == 0 {
					panic("cure_and_resurrection: assert, member absent (0x5236c4)")
				}
				world.remove(int(id), int(cond), 1, r)
				syncMember(id)
			} else if real {
				v := make([]int64, ccdIn)
				v[0], v[1], v[2], v[3] = int64(id), int64(cond), int64(amount), int64(c.pres)
				copy(v[4:], c.slot)
				v[4+ccdSlots] = int64(c.b09)
				o := ccdRun(v, r, true)
				copy(c.slot, o[:ccdSlots])
				c.b09 = uint32(o[ccdSlots])
				ch := world.char(int(id))
				world.m.w32(ch+0xb09, c.b09)
				for k, value := range c.slot {
					world.m.w32(ch+0xa01+4*uint32(k), uint32(value))
				}
			} else {
				c.slot[cond] = in[carPost+decay.n-1]
			}
			if uint32(c.slot[cond]) != 0 {
				count++
				result = 0
				// 0x590950 notice when byte 0x68510c is set: no declared output.
			} else {
				eff124 = 1
			}
			eff125 = 1
			continue
		}
		th := power * 15
		if cond == 0x12 || cond == 1 || flag {
			th = 100
		}
		if uint32(r.R(100)) >= th {
			result = 0
			continue
		}
		remove.add(id, cond, 1)
		if real {
			world.remove(int(id), int(cond), 1, r)
			syncMember(id)
		}
		eff125, eff124 = 1, 1
		if cond == 0x12 {
			s := (u(carChars+int(id)*carCharF+23) + u(carChars+int(id)*carCharF+24)) >> 2
			c.b11 = max(1, u(carChars+int(id)*carCharF+25)*s/100)
			c.b1d = max(1, u(carChars+int(id)*carCharF+26)*s/100)
			if real {
				ch := world.char(int(id))
				s = (world.m.r32(ch+0x111) + world.m.r32(ch+0x125)) >> 2
				c.b11 = max(1, world.m.r32(ch+0xb0d)*s/100)
				c.b1d = max(1, world.m.r32(ch+0xb19)*s/100)
				world.m.w32(ch+0xb11, c.b11)
				world.m.w32(ch+0xb1d, c.b1d)
			}
			if c.objID != -1 {
				if k := carObject(in, c.objID); k >= 0 {
					word4[k] = 0
				}
			}
		}
		if r.R(100) < 50 {
			voice.add(carArena+id*carCharSize, u(29), 0, u(30), u(31))
		}
	}

	for j := int32(0); j < nMon; j++ {
		k := findMon(u(14 + int(j)))
		m := mons[k]
		t := uint32(m[cond])
		if t == 0 {
			continue
		}
		if t != 9999 && t > amount && !flag {
			m[cond] = int64(t - amount)
			count++
			result = 0
			eff125 = 1
			continue
		}
		th := power * 15
		if cond == 0x12 || cond == 1 || t <= amount || flag {
			th = 100
		}
		if uint32(r.R(100)) >= th {
			result = 0
			continue
		}
		// 0x523f40(id, cond): the record found by id again (first match).
		hp := in[carMonBase+k*carMonF+1]
		if hp != 0 && m[19] != 0 && (cond != 19) {
			panic("cure_and_resurrection: 0x523f40 on a living monster with condition 19 faults in 0x50eac0")
		}
		v := make([]int64, 26)
		v[0], v[1], v[2], v[3] = in[carMonBase+k*carMonF], int64(cond), 0xffffffff, hp
		copy(v[4:], m)
		copy(m, monsterConditionDecayNode(v, r))
		eff125 = 1
	}

	if u(27)&0xff == 0 && count > 0 {
		if g69b7b8 != 0 {
			g69b7b8 = 1 // 0x5905f0 notice, then 0x5905c0(1, -1)
		}
		kind := u(22)
		switch {
		case count == 1 && kind == 1:
			// 0x590950(effect+0x94, ...): no declared output.
		case count == 1 && kind == 3:
			k := findMon(u(24))
			notice.add(carMonAddr+0x400*uint32(k), carTextAddr(0x6c4/4), tblStr)
		}
		eff124 = 1
	}

	out := []int64{int64(result), int64(eff124), int64(eff125), int64(g69b7b8)}
	for _, c := range chars {
		out = append(out, int64(c.b11), int64(c.b1d))
	}
	for _, w := range word4 {
		out = append(out, int64(w))
	}
	for _, m := range mons {
		out = append(out, m...)
	}
	out = decay.emit(out, 3)
	out = remove.emit(out, 3)
	out = voice.emit(out, 5)
	out = carEmitOne(out, &notice)
	return voice.emitExtra(out)
}

// carRemoveInput supplies the removal interface for the shared character
// world: empty formations, no items or cursor item, out of combat. The resource
// cells live in ccrWorld and are initialized from this node's character inputs.
func carRemoveInput(slot []int64, who, cond, pres uint32, g686a70 int64) []int64 {
	v := make([]int64, ccrLen)
	v[ccrCharBase], v[ccrWho], v[ccrCond], v[ccrNotify] = pcc0CharPtr, int64(who), int64(cond), 1
	v[ccrPres+int(who)], v[ccrWhoPres] = int64(pres&0xff), int64(pres&0xff)
	v[ccrG686a70] = g686a70
	copy(v[ccrSlot:], slot)
	forms := v[ccrPC+5+2*pcMembers:]
	for t := 0; t < 3; t++ {
		f := forms[t*pcFormLen : (t+1)*pcFormLen]
		for i := 0; i < 15; i++ {
			f[i] = 255
		}
		for c := 0; c < pcMembers; c++ {
			f[20+4*c], f[21+4*c], f[22+4*c] = 255, 255, 255
		}
	}
	return v
}

// carEmitOne emits a callee recorded with n=1 (only call 0's args).
func carEmitOne(out []int64, c *stbCalls) []int64 {
	for i := 0; i < 3; i++ {
		out = append(out, int64(c.args[i][0]))
	}
	return append(out, int64(c.n))
}

// carObject resolves 0x50b800(i) on the fixture object list: the object index
// whose word +4 is cleared, or -1 when the list, the entry or a set byte +0xc7
// yields NULL.
func carObject(in []int64, i int32) int {
	list := uint32(in[32])
	if list == 0 {
		return -1
	}
	if list != carObjList {
		panic("cure_and_resurrection: object list outside the fixture")
	}
	if i >= int32(uint32(in[33])) {
		i = 0 // index past the count reads the first entry
	}
	if i < 0 || i >= carObjs {
		panic("cure_and_resurrection: object id outside the 4-entry fixture list")
	}
	e := uint32(in[34+int(i)])
	if e == 0 {
		return -1
	}
	if e < carObjAddr || (e-carObjAddr)%0x100 != 0 || (e-carObjAddr)/0x100 >= carObjs {
		panic("cure_and_resurrection: object entry outside the fixture objects")
	}
	k := int((e - carObjAddr) / 0x100)
	if in[38+2*k]&0xff != 0 {
		return -1
	}
	return k
}

// Each of eight target visits can produce a death response, a state-change
// voice and a cure voice. Keep the first eight cells in their historical output
// positions; append the additional records so existing composition links hold.
const carVoiceMax = 3 * carMax

type carVoiceCalls struct {
	args [5][carVoiceMax]uint32
	n    int
}

func (c *carVoiceCalls) add(a ...uint32) {
	if len(a) != 5 || c.n >= carVoiceMax {
		panic("cure_and_resurrection: voice record sequence full")
	}
	for i, v := range a {
		c.args[i][c.n] = v
	}
	c.n++
}
func (c *carVoiceCalls) emit(out []int64, argc int) []int64 {
	for a := 0; a < argc; a++ {
		for k := 0; k < carMax; k++ {
			out = append(out, int64(c.args[a][k]))
		}
	}
	return append(out, int64(c.n))
}
func (c *carVoiceCalls) emitExtra(out []int64) []int64 {
	for a := 0; a < 5; a++ {
		for k := carMax; k < carVoiceMax; k++ {
			out = append(out, int64(c.args[a][k]))
		}
	}
	return out
}
