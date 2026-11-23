package calc

import "fmt"

// 0x551eb0 status_duration_after_resist(target, cond, realm, strength, penalty,
// duration, unused, flagB, message): the resisted duration of a status, the
// condition-5 buff cancellation for characters (0x523b30) and monsters (0x524400),
// then 0x522fe0 (character) or 0x523c00 (target type 3). World layout:
// re/emu/cases/00551eb0.json. 0x552250, 0x523b30, 0x523c00 and 0x522fe0 are D4
// callees and address stubs; 0x58aad0, 0x58b410, 0x4acd80 and 0x5182e0 are 排除
// stubs. 0x524400, 0x50e8c0 (mcaWorld.rebuild), 0x4e5150 (mcaWorld.name) and
// 0x590950 run natively. Emu replays the stubs; F runs 0x552250 (five-input form,
// Rj/Lt as F-only inputs) and 0x522fe0 (on F-only world cells, as 0x52b7e0 does).
// 0x523b30 and 0x523c00 leave eax unused here, and nothing after them in this node
// reads what they write.

const (
	sdrStub     = 0x20001600 // case object "stub": 0x552250's written duration at +0
	sdrMonster  = 0x20000400 // case object "M": its id mirrors the target's monster id (case "also")
	sdrTextID   = 0x61e59a   // u16 text id of the resist notice (861)
	sdrTextTab  = 0x68c09c   // text pointer table
	sdrNoticeFm = 0x6175c0   // character notice format ("%s!")
	sdrMonFmt   = 0x614a3c   // monster notice format, 0x58aad0 arg 2
	sdrQuote    = 0x689b34   // 0x590950 suffix after ' or :
	sdrSpace    = 0x60cc74   // 0x590950 suffix otherwise
	sdrCharFmt  = 0x64bf5c   // 0x590950's 0x58aad0 format
	sdrLogFmt   = 0x614b20   // 0x4e5150's 0x5182e0 log format
	sdrDurArg   = 5          // duration is stack argument 5 (0-based)
	sdrFx       = 180        // first F-only input (fRj, fLt, then the 0x522fe0 world)
	sdrWideCap  = 0x1000     // fixture cap on wide string length
)

const (
	sdrK552250 = iota // sdrSpec.stub order
	sdrK523b30
	sdrK523c00
	sdrK522fe0
	sdrK58aad0
	sdrK58b410
	sdrK4acd80
	sdrK5182e0
)

func init() {
	if sdrIn[sdrFx] != "fRj" || len(sdrIn)-sdrFx-2 != ccaWLen {
		panic("status duration after resist: F-only input layout")
	}
	Nodes["00551eb0"] = Node{Addr: "00551eb0", In: sdrIn, Out: sdrOut, F: statusDurationAfterResistNode, Emu: sdrEmu}
}

func statusDurationAfterResistNode(in []int64, r Rand) []int64 { return sdrRun(in, r, true) }

func sdrEmu(in []int64, r Rand) []int64 { return sdrRun(in, r, false) }

// The notice overrides belong only to this node's case. Parents keep the
// pre-G2 field order and use the same unoverridden memory reads as before G2.
func sdrNoticeFixture(name string) bool {
	switch name {
	case "noticeFmt0", "noticeFmt1", "suffix0":
		return true
	}
	return false
}

var sdrParentFields = func() []int {
	var fields []int
	for i, name := range sdrIn {
		if !sdrNoticeFixture(name) {
			fields = append(fields, i)
		}
	}
	return fields
}()

func sdrParentInput(world []int64) []int64 {
	if len(world) != len(sdrParentFields)-sirSdrSkip {
		panic("status duration after resist: wrong parent world count")
	}
	v := make([]int64, len(sdrIn))
	for i, field := range sdrParentFields[sirSdrSkip:] {
		v[field] = world[i]
	}
	w := &mcaWorld{mstWorld: &mstWorld{}}
	for i, name := range sdrIn {
		if sdrNoticeFixture(name) {
			v[i] = int64(w.u16(sdrSpec.in[i].addr))
		}
	}
	return v
}

// sdrWide reads a NUL-terminated UTF-16 string (exe bytes outside the arena).
func sdrWide(w *mcaWorld, a uint32) []uint16 {
	var s []uint16
	for ; ; a += 2 {
		c := uint16(w.u16(a))
		if c == 0 {
			return s
		}
		if len(s) == sdrWideCap {
			panic("status duration after resist: wide string past the fixture cap")
		}
		s = append(s, c)
	}
}

// sdrNotice is 0x590950(idx, 0x6175c0, text): format, build a separate separator, log via
// 0x58aad0, then 0x58b410 with the name length (+1 after ', +1 more for 's).
func sdrNotice(w *mcaWorld, idx, text uint32) {
	var buf []uint16
	fm := sdrWide(w, sdrNoticeFm)
	for i := 0; i < len(fm); i++ {
		if fm[i] != '%' {
			buf = append(buf, fm[i])
			continue
		}
		if i+1 >= len(fm) || fm[i+1] != 's' {
			panic(fmt.Sprintf("status duration after resist: format conversion at %d is not %%s", i))
		}
		buf = append(buf, sdrWide(w, text)...)
		i++
	}
	at := func(i int) uint16 {
		if i < len(buf) {
			return buf[i]
		}
		return 0
	}
	suffix := uint32(sdrSpace)
	if at(0) == '\'' || at(0) == ':' {
		suffix = sdrQuote
	}
	// Model the separate two-character local in virtual stack memory. The fixture
	// starts this uninitialized local at zero; wcscat never changes buf.
	separatorAt := uint32(mstStackBase - 0x2008)
	formattedAt := separatorAt + 4
	for i, ch := range append(append([]uint16(nil), buf...), 0) {
		w.put(formattedAt+2*uint32(i), 2, uint64(ch))
	}
	separator := append(sdrWide(w, separatorAt), sdrWide(w, suffix)...)
	for i, ch := range append(separator, 0) {
		w.put(separatorAt+2*uint32(i), 2, uint64(ch))
	}
	c := w.charBase(idx)
	w.note(sdrK58aad0, 8, sdrCharFmt, c+5, separatorAt, formattedAt)
	n := uint32(len(sdrWide(w, c+5)))
	if at(0) == '\'' {
		n++
		if at(1) == 's' {
			n++
		}
	}
	w.note(sdrK58b410, uint32(w.u8(w.u32(0x685178)+0x106*idx+0xf1)), 0, n, 0xffffffff)
}

// sdrMonsterName is 0x4e5150(m, 0, 0) with its 0x5182e0 log (actor not in state 0x15).
func sdrMonsterName(w *mcaWorld, m uint32) uint32 {
	if w.u16(w.template(m)+0x187) != 0x18d && w.u32(m+4) == 0 && !w.act15(w.u32(m+0xc)) {
		w.note(sdrK5182e0, 1, sdrLogFmt, w.u32(m))
	}
	return w.name(m)
}

// sdrCancelMonsterBuff is 0x524400(id, 5, duration): shorten buff 5 or clear it,
// stop effect 0x15 (0x4acd80) and rebuild (0x50e8c0).
func sdrCancelMonsterBuff(w *mcaWorld, id, dur uint32) {
	m := w.findMonster(id)
	if s := w.u32(m + 0xeb); s > dur {
		w.put(m+0xeb, 4, uint64(s-dur))
		return
	}
	w.put(m+0xe3, 4, 0)
	w.put(m+0xe7, 4, 0)
	w.put(m+0xeb, 4, 0)
	w.note(sdrK4acd80, w.u32(m+0xc), 0x15, 0)
	w.rebuild(id)
}

func sdrRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(sdrSpec.in) {
		panic("status duration after resist: wrong input count")
	}
	w := &mcaWorld{mstWorld: mstBuild(&sdrSpec, in)}
	w.r = r
	arg := func(n uint32) uint32 { return w.u32(mstStackBase + 4 + 4*n) }
	tgt, cond, realm, strength, penalty, flagB, msg := arg(0), arg(1), arg(2), arg(3), arg(4), arg(7), arg(8)
	typ, idx, mon := w.u32(tgt), w.u32(tgt+4), w.u32(tgt+8)
	w.put(sdrMonster, 4, uint64(mon))
	al := sdrApply(w, in, real, tgt, cond, realm, strength, penalty, flagB, msg, typ, idx, mon)
	out := []int64{int64(al)}
	for _, f := range sdrSpec.out {
		out = append(out, w.read(f, in))
	}
	return out
}

func sdrApply(w *mcaWorld, in []int64, real bool, tgt, cond, realm, strength, penalty, flagB, msg, typ, idx, mon uint32) byte {
	durAt := uint32(mstStackBase + 4 + 4*sdrDurArg)
	w.note(sdrK552250, durAt, tgt, realm, strength)
	dur := w.u32(sdrStub)
	if real {
		dur = uint32(resistanceQuantityReductionNode([]int64{int64(w.u32(durAt)), in[sdrFx], in[sdrFx+1], int64(strength), int64(typ)}, w.r)[0])
	}
	if dur == 0 {
		return 0
	}
	text := w.u32(w.u32(sdrTextTab) + 4*w.u16(sdrTextID))
	if cond == 5 {
		var c uint32
		if typ == 1 {
			if idx >= mstMaxChar {
				panic("status duration after resist: party index past 7")
			}
			c = w.u32(w.charBase(idx) + 0xaa9) // buff 5 duration
			if c != 0 {
				w.note(sdrK523b30, idx, 5, dur)
			}
		} else {
			c = w.u32(w.findMonster(mon) + 0xeb)
			if c != 0 {
				sdrCancelMonsterBuff(w, mon, dur)
			}
		}
		if c != 0 {
			if e := dur - c; int32(e) > 0 {
				dur = e
			} else {
				if msg&0xff != 0 {
					if typ == 1 {
						sdrNotice(w, idx, text)
					} else {
						w.note(sdrK58aad0, 9, sdrMonFmt, sdrMonsterName(w, w.findMonster(mon)))
					}
				}
				return 1
			}
		}
	}
	if dur == 1 {
		dur = 2
	}
	if penalty != 0 && cond != 7 {
		panic("status duration after resist: penalty without condition 7 (0x523046 / 0x523c15 assert)")
	}
	if typ == 3 {
		w.note(sdrK523c00, mon, cond, dur, penalty, 0, msg)
		return 1
	}
	if idx >= mstMaxChar || cond >= ccaSlots {
		panic("status duration after resist: member or state outside 0x522fe0's eight members and 20 slots")
	}
	pres := int64(w.u8(w.u32(0x685178) + 0x106*idx))
	if pres == 0 {
		panic("status duration after resist: assert, member absent (0x52301e)")
	}
	w.note(sdrK522fe0, idx, cond, dur, penalty, flagB, msg)
	if !real {
		return byte(w.stubRet(sdrK522fe0))
	}
	v := append([]int64{int64(idx), int64(cond), int64(dur), int64(penalty), int64(flagB), int64(msg)}, in[sdrFx+2:]...)
	v[6+ccaWPres+int(idx)], v[6+ccaWWhoPres] = pres, pres
	return byte(characterConditionApplyNode(v, w.r)[0])
}

// Code below is generated from the same description as re/emu/cases/00551eb0.json.

var sdrSpec = mstSpec{
	set: []mstSet{
		{0x7f000004, 4, 0x20000000},
		{0x685174, 4, 0x20010000},
		{0x685178, 4, 0x20003000},
		{0x683fad, 4, 0x20001500},
		{0x683fb9, 4, 0x20001540},
		{0x683fb1, 4, 0x20001580},
		{0x683fbd, 4, 0x200015c0},
		{0x6840c7, 4, 0x20000c00},
		{0x68c09c, 4, 0x20002000},
		{0x6875ef, 4, 0x0},
		{0x20001500, 4, 0x20001510},
		{0x20001508, 4, 0x2},
		{0x20001510, 4, 0x20000800},
		{0x20001514, 4, 0x20000400},
		{0x20001540, 4, 0x20001550},
		{0x20001548, 4, 0x0},
		{0x20001580, 4, 0x20001590},
		{0x20001588, 4, 0x2},
		{0x20001590, 4, 0x20001400},
		{0x20001594, 4, 0x20001440},
		{0x200015c0, 4, 0x200015d0},
		{0x200015c8, 4, 0x0},
		{0x20001400, 4, 0x1},
		{0x20001440, 4, 0x2},
		{0x20000800, 4, 0x7},
		{0x20000808, 4, 0x0},
		{0x2000080c, 4, 0x20000f00},
		{0x20000810, 4, 0x20001200},
		{0x20000408, 4, 0x0},
		{0x2000040c, 4, 0x20000f00},
		{0x20000410, 4, 0x20001200},
		{0x200010d8, 4, 0x20001100},
		{0x200008e3, 4, 0x31},
		{0x200008e7, 4, 0x32},
		{0x200008eb, 4, 0x33},
		{0x200005db, 1, 0x80},
		{0x200005dc, 1, 0x81},
		{0x200005dd, 1, 0x82},
		{0x200005de, 1, 0x83},
		{0x200005df, 1, 0x84},
		{0x200005e0, 1, 0x85},
		{0x200005e1, 1, 0x86},
		{0x200005e2, 1, 0x87},
		{0x200005e3, 1, 0x88},
		{0x200005e4, 1, 0x89},
		{0x200005e5, 1, 0x8a},
		{0x200005e6, 1, 0x8b},
		{0x200005e7, 1, 0x8c},
		{0x200005e8, 1, 0x8d},
		{0x200005e9, 1, 0x8e},
		{0x200005ea, 1, 0x8f},
		{0x200005eb, 1, 0x90},
		{0x200005ec, 1, 0x91},
		{0x200005ed, 1, 0x92},
		{0x200005ee, 1, 0x93},
		{0x200005ef, 1, 0x94},
		{0x200005f0, 1, 0x95},
		{0x200005f1, 1, 0x96},
		{0x200005f2, 1, 0x97},
		{0x200005f3, 1, 0x98},
		{0x200005f4, 1, 0x99},
		{0x200005f5, 1, 0x9a},
		{0x200005f6, 1, 0x9b},
		{0x200005f7, 1, 0x9c},
		{0x200005f8, 1, 0x9d},
		{0x200005f9, 1, 0x9e},
		{0x200005fa, 1, 0x9f},
		{0x200005fb, 1, 0xa0},
		{0x200005fc, 1, 0xa1},
		{0x200005fd, 1, 0xa2},
		{0x200005fe, 1, 0xa3},
		{0x200005ff, 1, 0xa4},
		{0x20000600, 1, 0xa5},
		{0x20000601, 1, 0xa6},
		{0x20000602, 1, 0xa7},
		{0x20000603, 1, 0xa8},
		{0x20000604, 1, 0xa9},
		{0x20000605, 1, 0xaa},
		{0x20000606, 1, 0xab},
		{0x20000607, 1, 0xac},
		{0x20000608, 1, 0xad},
		{0x20000609, 1, 0xae},
		{0x2000060a, 1, 0xaf},
		{0x2000060b, 1, 0xb0},
		{0x2000060c, 1, 0xb1},
		{0x2000060d, 1, 0xb2},
		{0x2000060e, 1, 0xb3},
		{0x2000060f, 1, 0xb4},
		{0x20000610, 1, 0xb5},
		{0x20000611, 1, 0xb6},
		{0x20000612, 1, 0xb7},
		{0x20000613, 1, 0xb8},
		{0x20000614, 1, 0xb9},
		{0x20000615, 1, 0xba},
		{0x20000616, 1, 0xbb},
		{0x20000617, 1, 0xbc},
		{0x20000618, 1, 0xbd},
		{0x20000619, 1, 0xbe},
		{0x2000061a, 1, 0xbf},
		{0x2000061b, 1, 0xc0},
		{0x2000061c, 1, 0xc1},
		{0x2000061d, 1, 0xc2},
		{0x2000061e, 1, 0xc3},
		{0x2000061f, 1, 0xc4},
		{0x20000620, 1, 0xc5},
		{0x20000621, 1, 0xc6},
		{0x20000622, 1, 0xc7},
		{0x20000623, 1, 0xc8},
		{0x20000624, 1, 0xc9},
		{0x20000625, 1, 0xca},
		{0x20000626, 1, 0xcb},
		{0x20000627, 1, 0xcc},
		{0x20000628, 1, 0xcd},
		{0x20000629, 1, 0xce},
		{0x2000062a, 1, 0xcf},
		{0x2000062b, 1, 0xd0},
		{0x2000062c, 1, 0xd1},
		{0x2000062d, 1, 0xd2},
		{0x2000062e, 1, 0xd3},
		{0x2000062f, 1, 0xd4},
		{0x20000630, 1, 0xd5},
		{0x20000631, 1, 0xd6},
		{0x20000632, 1, 0xd7},
		{0x20000633, 1, 0xd8},
		{0x20000634, 1, 0xd9},
		{0x20000635, 1, 0xda},
		{0x20000636, 1, 0xdb},
		{0x20000637, 1, 0xdc},
		{0x20000638, 1, 0xdd},
		{0x20000639, 1, 0xde},
		{0x2000063a, 1, 0xdf},
		{0x2000063b, 1, 0xe0},
		{0x2000063c, 1, 0xe1},
		{0x2000063d, 1, 0xe2},
		{0x2000063e, 1, 0xe3},
		{0x2000063f, 1, 0xe4},
		{0x20000640, 1, 0xe5},
		{0x20000641, 1, 0xe6},
		{0x20000647, 1, 0xe0},
		{0x20000648, 1, 0xe1},
		{0x20000649, 1, 0xe2},
		{0x2000064a, 1, 0xe3},
		{0x2000064b, 1, 0xe4},
		{0x20000447, 4, 0x11111111},
		{0x2000044f, 4, 0x22222222},
		{0x20002d74, 4, 0x20001800},
		{0x20010aa9, 4, 0x7000},
		{0x20010005, 2, 0x5a},
		{0x20010007, 2, 0x59},
		{0x20003000, 1, 0x50},
		{0x200030f1, 1, 0xa0},
		{0x2001230b, 4, 0x7001},
		{0x20011867, 2, 0x5a},
		{0x20011869, 2, 0x59},
		{0x20003106, 1, 0x51},
		{0x200031f7, 1, 0xa1},
		{0x20013b6d, 4, 0x7002},
		{0x200130c9, 2, 0x5a},
		{0x200130cb, 2, 0x59},
		{0x2000320c, 1, 0x52},
		{0x200032fd, 1, 0xa2},
		{0x200153cf, 4, 0x7003},
		{0x2001492b, 2, 0x5a},
		{0x2001492d, 2, 0x59},
		{0x20003312, 1, 0x53},
		{0x20003403, 1, 0xa3},
		{0x20016c31, 4, 0x7004},
		{0x2001618d, 2, 0x5a},
		{0x2001618f, 2, 0x59},
		{0x20003418, 1, 0x54},
		{0x20003509, 1, 0xa4},
		{0x20018493, 4, 0x7005},
		{0x200179ef, 2, 0x5a},
		{0x200179f1, 2, 0x59},
		{0x2000351e, 1, 0x55},
		{0x2000360f, 1, 0xa5},
		{0x20019cf5, 4, 0x7006},
		{0x20019251, 2, 0x5a},
		{0x20019253, 2, 0x59},
		{0x20003624, 1, 0x56},
		{0x20003715, 1, 0xa6},
		{0x2001b557, 4, 0x7007},
		{0x2001aab3, 2, 0x5a},
		{0x2001aab5, 2, 0x59},
		{0x2000372a, 1, 0x57},
		{0x2000381b, 1, 0xa7},
	},
	in: []mstFld{
		{addr: 0x7f000008, size: 4, signed: false, idx: -1},                          // cond
		{addr: 0x7f00000c, size: 4, signed: false, idx: -1},                          // realm
		{addr: 0x7f000010, size: 4, signed: false, idx: -1},                          // strength
		{addr: 0x7f000014, size: 4, signed: false, idx: -1},                          // penalty
		{addr: 0x7f000018, size: 4, signed: false, idx: -1},                          // duration
		{addr: 0x7f000020, size: 4, signed: false, idx: -1},                          // flagB
		{addr: 0x7f000024, size: 4, signed: false, idx: -1},                          // message
		{addr: 0x20000000, size: 4, signed: false, idx: -1},                          // target.type
		{addr: 0x20000004, size: 4, signed: true, idx: -1},                           // charIdx (-1: monster target, 543fc1)
		{addr: 0x20000008, size: 4, signed: false, idx: -1},                          // monId
		{addr: 0x20001600, size: 4, signed: false, idx: -1},                          // r552250
		{addr: 0x20001604, size: 4, signed: false, idx: -1},                          // r522fe0
		{addr: 0x20010aa9, size: 4, signed: false, idx: 8, count: 8, stride: 0x1862}, // C.buff5.dur
		{addr: 0x20010005, size: 2, signed: false, idx: 8, count: 8, stride: 0x1862}, // C.name0
		{addr: 0x20010007, size: 2, signed: false, idx: 8, count: 8, stride: 0x1862}, // C.name1
		{addr: 0x20010009, size: 2, signed: false, idx: 8, count: 8, stride: 0x1862}, // C.name2
		{addr: 0x20003000, size: 1, signed: false, idx: 8, count: 8, stride: 0x106},  // P.present
		{addr: 0x200030f1, size: 1, signed: false, idx: 8, count: 8, stride: 0x106},  // P.f1
		{addr: 0x20001800, size: 2, signed: false, idx: -1},                          // text0
		{addr: 0x20001802, size: 2, signed: false, idx: -1},                          // text1
		{addr: 0x20000404, size: 4, signed: false, idx: -1},                          // M.group
		{addr: 0x20000415, size: 1, signed: false, idx: -1},                          // M.inCombat
		{addr: 0x20000427, size: 4, signed: false, idx: -1},                          // M.maxHP
		{addr: 0x2000042f, size: 4, signed: false, idx: -1},                          // M.maxSP
		{addr: 0x20000457, size: 4, signed: false, idx: -1},                          // M.cond0
		{addr: 0x2000045b, size: 4, signed: false, idx: -1},                          // M.cond1
		{addr: 0x2000045f, size: 4, signed: false, idx: -1},                          // M.cond2
		{addr: 0x20000463, size: 4, signed: false, idx: -1},                          // M.cond3
		{addr: 0x20000467, size: 4, signed: false, idx: -1},                          // M.cond4
		{addr: 0x2000046b, size: 4, signed: false, idx: -1},                          // M.cond5
		{addr: 0x2000046f, size: 4, signed: false, idx: -1},                          // M.cond6
		{addr: 0x20000473, size: 4, signed: false, idx: -1},                          // M.cond7
		{addr: 0x20000477, size: 4, signed: false, idx: -1},                          // M.cond8
		{addr: 0x2000047b, size: 4, signed: false, idx: -1},                          // M.cond9
		{addr: 0x2000047f, size: 4, signed: false, idx: -1},                          // M.cond10
		{addr: 0x20000483, size: 4, signed: false, idx: -1},                          // M.cond11
		{addr: 0x20000487, size: 4, signed: false, idx: -1},                          // M.cond12
		{addr: 0x2000048b, size: 4, signed: false, idx: -1},                          // M.cond13
		{addr: 0x2000048f, size: 4, signed: false, idx: -1},                          // M.cond14
		{addr: 0x20000493, size: 4, signed: false, idx: -1},                          // M.cond15
		{addr: 0x20000497, size: 4, signed: false, idx: -1},                          // M.cond16
		{addr: 0x2000049b, size: 4, signed: false, idx: -1},                          // M.cond17
		{addr: 0x2000049f, size: 4, signed: false, idx: -1},                          // M.cond18
		{addr: 0x200004a7, size: 1, signed: false, idx: -1},                          // M.buff0.power
		{addr: 0x200004ab, size: 2, signed: false, idx: -1},                          // M.buff0.mult
		{addr: 0x200004af, size: 4, signed: false, idx: -1},                          // M.buff0.dur
		{addr: 0x200004b3, size: 1, signed: false, idx: -1},                          // M.buff1.power
		{addr: 0x200004b7, size: 2, signed: false, idx: -1},                          // M.buff1.mult
		{addr: 0x200004bb, size: 4, signed: false, idx: -1},                          // M.buff1.dur
		{addr: 0x200004bf, size: 1, signed: false, idx: -1},                          // M.buff2.power
		{addr: 0x200004c3, size: 2, signed: false, idx: -1},                          // M.buff2.mult
		{addr: 0x200004c7, size: 4, signed: false, idx: -1},                          // M.buff2.dur
		{addr: 0x200004cb, size: 1, signed: false, idx: -1},                          // M.buff3.power
		{addr: 0x200004cf, size: 2, signed: false, idx: -1},                          // M.buff3.mult
		{addr: 0x200004d3, size: 4, signed: false, idx: -1},                          // M.buff3.dur
		{addr: 0x200004d7, size: 1, signed: false, idx: -1},                          // M.buff4.power
		{addr: 0x200004db, size: 2, signed: false, idx: -1},                          // M.buff4.mult
		{addr: 0x200004df, size: 4, signed: false, idx: -1},                          // M.buff4.dur
		{addr: 0x200004e3, size: 1, signed: false, idx: -1},                          // M.buff5.power
		{addr: 0x200004e7, size: 2, signed: false, idx: -1},                          // M.buff5.mult
		{addr: 0x200004eb, size: 4, signed: false, idx: -1},                          // M.buff5.dur
		{addr: 0x200004ef, size: 1, signed: false, idx: -1},                          // M.buff6.power
		{addr: 0x200004f3, size: 2, signed: false, idx: -1},                          // M.buff6.mult
		{addr: 0x200004f7, size: 4, signed: false, idx: -1},                          // M.buff6.dur
		{addr: 0x200004fb, size: 1, signed: false, idx: -1},                          // M.buff7.power
		{addr: 0x200004ff, size: 2, signed: false, idx: -1},                          // M.buff7.mult
		{addr: 0x20000503, size: 4, signed: false, idx: -1},                          // M.buff7.dur
		{addr: 0x2000050b, size: 4, signed: false, idx: -1},                          // M.attackPenalty
		{addr: 0x2000050f, size: 1, signed: false, idx: -1},                          // M.eff0.active
		{addr: 0x20000510, size: 4, signed: false, idx: -1},                          // M.eff0.type
		{addr: 0x20000514, size: 1, signed: false, idx: -1},                          // M.eff0.power
		{addr: 0x20000518, size: 4, signed: false, idx: -1},                          // M.eff0.mult
		{addr: 0x20000520, size: 1, signed: false, idx: -1},                          // M.eff1.active
		{addr: 0x20000521, size: 4, signed: false, idx: -1},                          // M.eff1.type
		{addr: 0x20000525, size: 1, signed: false, idx: -1},                          // M.eff1.power
		{addr: 0x20000529, size: 4, signed: false, idx: -1},                          // M.eff1.mult
		{addr: 0x20000531, size: 1, signed: false, idx: -1},                          // M.eff2.active
		{addr: 0x20000532, size: 4, signed: false, idx: -1},                          // M.eff2.type
		{addr: 0x20000536, size: 1, signed: false, idx: -1},                          // M.eff2.power
		{addr: 0x2000053a, size: 4, signed: false, idx: -1},                          // M.eff2.mult
		{addr: 0x20000542, size: 1, signed: false, idx: -1},                          // M.eff3.active
		{addr: 0x20000543, size: 4, signed: false, idx: -1},                          // M.eff3.type
		{addr: 0x20000547, size: 1, signed: false, idx: -1},                          // M.eff3.power
		{addr: 0x2000054b, size: 4, signed: false, idx: -1},                          // M.eff3.mult
		{addr: 0x20000553, size: 1, signed: false, idx: -1},                          // M.eff4.active
		{addr: 0x20000554, size: 4, signed: false, idx: -1},                          // M.eff4.type
		{addr: 0x20000558, size: 1, signed: false, idx: -1},                          // M.eff4.power
		{addr: 0x2000055c, size: 4, signed: false, idx: -1},                          // M.eff4.mult
		{addr: 0x20000564, size: 1, signed: false, idx: -1},                          // M.eff5.active
		{addr: 0x20000565, size: 4, signed: false, idx: -1},                          // M.eff5.type
		{addr: 0x20000569, size: 1, signed: false, idx: -1},                          // M.eff5.power
		{addr: 0x2000056d, size: 4, signed: false, idx: -1},                          // M.eff5.mult
		{addr: 0x20000575, size: 1, signed: false, idx: -1},                          // M.eff6.active
		{addr: 0x20000576, size: 4, signed: false, idx: -1},                          // M.eff6.type
		{addr: 0x2000057a, size: 1, signed: false, idx: -1},                          // M.eff6.power
		{addr: 0x2000057e, size: 4, signed: false, idx: -1},                          // M.eff6.mult
		{addr: 0x20000586, size: 1, signed: false, idx: -1},                          // M.eff7.active
		{addr: 0x20000587, size: 4, signed: false, idx: -1},                          // M.eff7.type
		{addr: 0x2000058b, size: 1, signed: false, idx: -1},                          // M.eff7.power
		{addr: 0x2000058f, size: 4, signed: false, idx: -1},                          // M.eff7.mult
		{addr: 0x20000597, size: 1, signed: false, idx: -1},                          // M.eff8.active
		{addr: 0x20000598, size: 4, signed: false, idx: -1},                          // M.eff8.type
		{addr: 0x2000059c, size: 1, signed: false, idx: -1},                          // M.eff8.power
		{addr: 0x200005a0, size: 4, signed: false, idx: -1},                          // M.eff8.mult
		{addr: 0x200005a8, size: 1, signed: false, idx: -1},                          // M.eff9.active
		{addr: 0x200005a9, size: 4, signed: false, idx: -1},                          // M.eff9.type
		{addr: 0x200005ad, size: 1, signed: false, idx: -1},                          // M.eff9.power
		{addr: 0x200005b1, size: 4, signed: false, idx: -1},                          // M.eff9.mult
		{addr: 0x200005b9, size: 1, signed: false, idx: -1},                          // M.eff10.active
		{addr: 0x200005ba, size: 4, signed: false, idx: -1},                          // M.eff10.type
		{addr: 0x200005be, size: 1, signed: false, idx: -1},                          // M.eff10.power
		{addr: 0x200005c2, size: 4, signed: false, idx: -1},                          // M.eff10.mult
		{addr: 0x200005ca, size: 1, signed: false, idx: -1},                          // M.eff11.active
		{addr: 0x200005cb, size: 4, signed: false, idx: -1},                          // M.eff11.type
		{addr: 0x200005cf, size: 1, signed: false, idx: -1},                          // M.eff11.power
		{addr: 0x200005d3, size: 4, signed: false, idx: -1},                          // M.eff11.mult
		{addr: 0x20000cd1, size: 1, signed: false, idx: -1},                          // tpl.attr0
		{addr: 0x20000cd2, size: 1, signed: false, idx: -1},                          // tpl.attr1
		{addr: 0x20000cd3, size: 1, signed: false, idx: -1},                          // tpl.attr2
		{addr: 0x20000cd4, size: 1, signed: false, idx: -1},                          // tpl.attr3
		{addr: 0x20000cd5, size: 1, signed: false, idx: -1},                          // tpl.attr4
		{addr: 0x20000d87, size: 2, signed: false, idx: -1},                          // tpl.species
		{addr: 0x200011a4, size: 1, signed: false, idx: -1},                          // actor.fa4
		{addr: 0x200011a7, size: 1, signed: false, idx: -1},                          // actor.fa7
		{addr: 0x2000123e, size: 1, signed: false, idx: -1},                          // T.sp0.active
		{addr: 0x2000123f, size: 4, signed: false, idx: -1},                          // T.sp0.code
		{addr: 0x20001243, size: 1, signed: false, idx: -1},                          // T.sp0.value
		{addr: 0x2000124f, size: 1, signed: false, idx: -1},                          // T.sp1.active
		{addr: 0x20001250, size: 4, signed: false, idx: -1},                          // T.sp1.code
		{addr: 0x20001254, size: 1, signed: false, idx: -1},                          // T.sp1.value
		{addr: 0x20001260, size: 1, signed: false, idx: -1},                          // T.sp2.active
		{addr: 0x20001261, size: 4, signed: false, idx: -1},                          // T.sp2.code
		{addr: 0x20001265, size: 1, signed: false, idx: -1},                          // T.sp2.value
		{addr: 0x20001271, size: 1, signed: false, idx: -1},                          // T.sp3.active
		{addr: 0x20001272, size: 4, signed: false, idx: -1},                          // T.sp3.code
		{addr: 0x20001276, size: 1, signed: false, idx: -1},                          // T.sp3.value
		{addr: 0x20001282, size: 1, signed: false, idx: -1},                          // T.sp4.active
		{addr: 0x20001283, size: 4, signed: false, idx: -1},                          // T.sp4.code
		{addr: 0x20001287, size: 1, signed: false, idx: -1},                          // T.sp4.value
		{addr: 0x20001293, size: 1, signed: false, idx: -1},                          // T.sp5.active
		{addr: 0x20001294, size: 4, signed: false, idx: -1},                          // T.sp5.code
		{addr: 0x20001298, size: 1, signed: false, idx: -1},                          // T.sp5.value
		{addr: 0x200012a4, size: 1, signed: false, idx: -1},                          // T.sp6.active
		{addr: 0x200012a5, size: 4, signed: false, idx: -1},                          // T.sp6.code
		{addr: 0x200012a9, size: 1, signed: false, idx: -1},                          // T.sp6.value
		{addr: 0x200012b5, size: 1, signed: false, idx: -1},                          // T.sp7.active
		{addr: 0x200012b6, size: 4, signed: false, idx: -1},                          // T.sp7.code
		{addr: 0x200012ba, size: 1, signed: false, idx: -1},                          // T.sp7.value
		{addr: 0x200012c6, size: 1, signed: false, idx: -1},                          // T.sp8.active
		{addr: 0x200012c7, size: 4, signed: false, idx: -1},                          // T.sp8.code
		{addr: 0x200012cb, size: 1, signed: false, idx: -1},                          // T.sp8.value
		{addr: 0x200012d7, size: 1, signed: false, idx: -1},                          // T.aura0.active
		{addr: 0x200012d8, size: 4, signed: false, idx: -1},                          // T.aura0.code
		{addr: 0x200012dc, size: 1, signed: false, idx: -1},                          // T.aura0.power
		{addr: 0x200012e0, size: 4, signed: false, idx: -1},                          // T.aura0.mult
		{addr: 0x200012e8, size: 1, signed: false, idx: -1},                          // T.aura1.active
		{addr: 0x200012e9, size: 4, signed: false, idx: -1},                          // T.aura1.code
		{addr: 0x200012ed, size: 1, signed: false, idx: -1},                          // T.aura1.power
		{addr: 0x200012f1, size: 4, signed: false, idx: -1},                          // T.aura1.mult
		{addr: 0x200012f9, size: 1, signed: false, idx: -1},                          // T.aura2.active
		{addr: 0x200012fa, size: 4, signed: false, idx: -1},                          // T.aura2.code
		{addr: 0x200012fe, size: 1, signed: false, idx: -1},                          // T.aura2.power
		{addr: 0x20001302, size: 4, signed: false, idx: -1},                          // T.aura2.mult
		{addr: 0x2000130a, size: 1, signed: false, idx: -1},                          // T.aura3.active
		{addr: 0x2000130b, size: 4, signed: false, idx: -1},                          // T.aura3.code
		{addr: 0x2000130f, size: 1, signed: false, idx: -1},                          // T.aura3.power
		{addr: 0x20001313, size: 4, signed: false, idx: -1},                          // T.aura3.mult
		{addr: 0x2000131b, size: 1, signed: false, idx: -1},                          // T.aura4.active
		{addr: 0x2000131c, size: 4, signed: false, idx: -1},                          // T.aura4.code
		{addr: 0x20001320, size: 1, signed: false, idx: -1},                          // T.aura4.power
		{addr: 0x20001324, size: 4, signed: false, idx: -1},                          // T.aura4.mult
		{addr: 0x2000132c, size: 1, signed: false, idx: -1},                          // T.aura5.active
		{addr: 0x2000132d, size: 4, signed: false, idx: -1},                          // T.aura5.code
		{addr: 0x20001331, size: 1, signed: false, idx: -1},                          // T.aura5.power
		{addr: 0x20001335, size: 4, signed: false, idx: -1},                          // T.aura5.mult
		{addr: 0x2000142c, size: 1, signed: false, idx: -1},                          // grpA.f2c
		{addr: 0x2000146c, size: 1, signed: false, idx: -1},                          // grpB.f2c
		{addr: 0x6175c0, size: 2, signed: false, idx: -1},                            // noticeFmt0
		{addr: 0x6175c2, size: 2, signed: false, idx: -1},                            // noticeFmt1
		{addr: 0x689b34, size: 2, signed: false, idx: -1},                            // suffix0
		{addr: 0x20001900, size: 4, signed: false, idx: -1},                          // fRj
		{addr: 0x20001904, size: 4, signed: false, idx: -1},                          // fLt
		{addr: 0x20004000, size: 4, signed: true, idx: -1},                           // ft0
		{addr: 0x20004004, size: 4, signed: true, idx: -1},                           // ft1
		{addr: 0x20004008, size: 4, signed: true, idx: -1},                           // ft2
		{addr: 0x2000400c, size: 4, signed: true, idx: -1},                           // ft3
		{addr: 0x20004010, size: 4, signed: true, idx: -1},                           // ft4
		{addr: 0x20004014, size: 4, signed: true, idx: -1},                           // ft5
		{addr: 0x20004018, size: 4, signed: true, idx: -1},                           // ft6
		{addr: 0x2000401c, size: 4, signed: true, idx: -1},                           // ft7
		{addr: 0x20004020, size: 4, signed: true, idx: -1},                           // ft8
		{addr: 0x20004024, size: 4, signed: true, idx: -1},                           // ft9
		{addr: 0x20004028, size: 4, signed: true, idx: -1},                           // ft10
		{addr: 0x2000402c, size: 4, signed: true, idx: -1},                           // ft11
		{addr: 0x20004030, size: 4, signed: true, idx: -1},                           // ft12
		{addr: 0x20004034, size: 4, signed: true, idx: -1},                           // ft13
		{addr: 0x20004038, size: 4, signed: true, idx: -1},                           // ft14
		{addr: 0x2000403c, size: 4, signed: true, idx: -1},                           // ft15
		{addr: 0x20004040, size: 4, signed: true, idx: -1},                           // ft16
		{addr: 0x20004044, size: 4, signed: true, idx: -1},                           // ft17
		{addr: 0x20004048, size: 4, signed: true, idx: -1},                           // ft18
		{addr: 0x2000404c, size: 4, signed: true, idx: -1},                           // ft19
		{addr: 0x20004050, size: 4, signed: true, idx: -1},                           // ft20
		{addr: 0x20004054, size: 4, signed: true, idx: -1},                           // ft21
		{addr: 0x20004058, size: 4, signed: true, idx: -1},                           // ft22
		{addr: 0x2000405c, size: 4, signed: true, idx: -1},                           // ft23
		{addr: 0x20004060, size: 4, signed: true, idx: -1},                           // ft24
		{addr: 0x20004064, size: 4, signed: true, idx: -1},                           // ft25
		{addr: 0x20004068, size: 4, signed: true, idx: -1},                           // ft26
		{addr: 0x2000406c, size: 4, signed: true, idx: -1},                           // ft27
		{addr: 0x20004070, size: 4, signed: true, idx: -1},                           // ft28
		{addr: 0x20004074, size: 4, signed: true, idx: -1},                           // ft29
		{addr: 0x20004078, size: 4, signed: true, idx: -1},                           // ft30
		{addr: 0x2000407c, size: 4, signed: true, idx: -1},                           // ft31
		{addr: 0x20004080, size: 4, signed: true, idx: -1},                           // ft32
		{addr: 0x20004084, size: 4, signed: true, idx: -1},                           // ft33
		{addr: 0x20004088, size: 4, signed: true, idx: -1},                           // ft34
		{addr: 0x2000408c, size: 4, signed: true, idx: -1},                           // ft35
		{addr: 0x20004090, size: 4, signed: true, idx: -1},                           // ft36
		{addr: 0x20004094, size: 4, signed: true, idx: -1},                           // ft37
		{addr: 0x20004098, size: 4, signed: true, idx: -1},                           // ft38
		{addr: 0x2000409c, size: 4, signed: true, idx: -1},                           // ft39
		{addr: 0x200040a0, size: 4, signed: true, idx: -1},                           // ft40
		{addr: 0x200040a4, size: 4, signed: true, idx: -1},                           // ft41
		{addr: 0x200040a8, size: 4, signed: true, idx: -1},                           // ft42
		{addr: 0x200040ac, size: 4, signed: true, idx: -1},                           // ft43
		{addr: 0x200040b0, size: 4, signed: true, idx: -1},                           // ft44
		{addr: 0x200040b4, size: 4, signed: true, idx: -1},                           // ft45
		{addr: 0x200040b8, size: 4, signed: true, idx: -1},                           // ft46
		{addr: 0x200040bc, size: 4, signed: true, idx: -1},                           // ft47
		{addr: 0x200040c0, size: 4, signed: true, idx: -1},                           // ft48
		{addr: 0x200040c4, size: 4, signed: true, idx: -1},                           // ft49
		{addr: 0x200040c8, size: 4, signed: true, idx: -1},                           // ft50
		{addr: 0x200040cc, size: 4, signed: true, idx: -1},                           // ft51
		{addr: 0x200040d0, size: 4, signed: true, idx: -1},                           // ft52
		{addr: 0x200040d4, size: 4, signed: true, idx: -1},                           // ft53
		{addr: 0x200040d8, size: 4, signed: true, idx: -1},                           // ft54
		{addr: 0x200040dc, size: 4, signed: true, idx: -1},                           // ft55
		{addr: 0x200040e0, size: 4, signed: true, idx: -1},                           // ft56
		{addr: 0x200040e4, size: 4, signed: true, idx: -1},                           // ft57
		{addr: 0x200040e8, size: 4, signed: true, idx: -1},                           // ft58
		{addr: 0x200040ec, size: 4, signed: true, idx: -1},                           // ft59
		{addr: 0x200040f0, size: 4, signed: true, idx: -1},                           // ft60
		{addr: 0x200040f4, size: 4, signed: true, idx: -1},                           // ft61
		{addr: 0x200040f8, size: 4, signed: true, idx: -1},                           // ft62
		{addr: 0x200040fc, size: 4, signed: true, idx: -1},                           // ft63
		{addr: 0x20004100, size: 4, signed: true, idx: -1},                           // ft64
		{addr: 0x20004104, size: 4, signed: true, idx: -1},                           // ft65
		{addr: 0x20004108, size: 4, signed: true, idx: -1},                           // ft66
		{addr: 0x2000410c, size: 4, signed: true, idx: -1},                           // ft67
		{addr: 0x20004110, size: 4, signed: true, idx: -1},                           // ft68
		{addr: 0x20004114, size: 4, signed: true, idx: -1},                           // ft69
		{addr: 0x20004118, size: 4, signed: true, idx: -1},                           // ft70
		{addr: 0x2000411c, size: 4, signed: true, idx: -1},                           // ft71
		{addr: 0x20004120, size: 4, signed: true, idx: -1},                           // ft72
		{addr: 0x20004124, size: 4, signed: true, idx: -1},                           // ft73
		{addr: 0x20004128, size: 4, signed: true, idx: -1},                           // ft74
		{addr: 0x2000412c, size: 4, signed: true, idx: -1},                           // ft75
		{addr: 0x20004130, size: 4, signed: true, idx: -1},                           // ft76
		{addr: 0x20004134, size: 4, signed: true, idx: -1},                           // ft77
		{addr: 0x20004138, size: 4, signed: true, idx: -1},                           // ft78
		{addr: 0x2000413c, size: 4, signed: true, idx: -1},                           // ft79
		{addr: 0x20004140, size: 4, signed: true, idx: -1},                           // ft80
		{addr: 0x20004144, size: 4, signed: true, idx: -1},                           // ft81
		{addr: 0x20004148, size: 4, signed: true, idx: -1},                           // ft82
		{addr: 0x2000414c, size: 4, signed: true, idx: -1},                           // ft83
		{addr: 0x20004150, size: 4, signed: true, idx: -1},                           // ft84
		{addr: 0x20004154, size: 4, signed: true, idx: -1},                           // ft85
		{addr: 0x20004158, size: 4, signed: true, idx: -1},                           // ft86
		{addr: 0x2000415c, size: 4, signed: true, idx: -1},                           // ft87
		{addr: 0x20004160, size: 4, signed: true, idx: -1},                           // ft88
		{addr: 0x20004164, size: 4, signed: true, idx: -1},                           // ft89
		{addr: 0x20004168, size: 4, signed: true, idx: -1},                           // ft90
		{addr: 0x2000416c, size: 4, signed: true, idx: -1},                           // ft91
		{addr: 0x20004170, size: 4, signed: true, idx: -1},                           // ft92
		{addr: 0x20004174, size: 4, signed: true, idx: -1},                           // ft93
		{addr: 0x20004178, size: 4, signed: true, idx: -1},                           // ft94
		{addr: 0x2000417c, size: 4, signed: true, idx: -1},                           // ft95
		{addr: 0x20004180, size: 4, signed: true, idx: -1},                           // ft96
		{addr: 0x20004184, size: 4, signed: true, idx: -1},                           // ft97
		{addr: 0x20004188, size: 4, signed: true, idx: -1},                           // ft98
		{addr: 0x2000418c, size: 4, signed: true, idx: -1},                           // ft99
		{addr: 0x20004190, size: 4, signed: true, idx: -1},                           // ft100
		{addr: 0x20004194, size: 4, signed: true, idx: -1},                           // ft101
		{addr: 0x20004198, size: 4, signed: true, idx: -1},                           // ft102
		{addr: 0x2000419c, size: 4, signed: true, idx: -1},                           // ft103
		{addr: 0x200041a0, size: 4, signed: true, idx: -1},                           // ft104
		{addr: 0x200041a4, size: 4, signed: true, idx: -1},                           // ft105
		{addr: 0x200041a8, size: 4, signed: true, idx: -1},                           // ft106
		{addr: 0x200041ac, size: 4, signed: true, idx: -1},                           // ft107
		{addr: 0x200041b0, size: 4, signed: true, idx: -1},                           // ft108
		{addr: 0x200041b4, size: 4, signed: true, idx: -1},                           // ft109
		{addr: 0x200041b8, size: 4, signed: true, idx: -1},                           // ft110
		{addr: 0x200041bc, size: 4, signed: true, idx: -1},                           // ft111
		{addr: 0x200041c0, size: 4, signed: true, idx: -1},                           // ft112
		{addr: 0x200041c4, size: 4, signed: true, idx: -1},                           // ft113
		{addr: 0x200041c8, size: 4, signed: true, idx: -1},                           // ft114
		{addr: 0x200041cc, size: 4, signed: true, idx: -1},                           // ft115
		{addr: 0x200041d0, size: 4, signed: true, idx: -1},                           // ft116
		{addr: 0x200041d4, size: 4, signed: true, idx: -1},                           // ft117
		{addr: 0x200041d8, size: 4, signed: true, idx: -1},                           // ft118
		{addr: 0x200041dc, size: 4, signed: true, idx: -1},                           // ft119
		{addr: 0x200041e0, size: 4, signed: true, idx: -1},                           // ft120
		{addr: 0x200041e4, size: 4, signed: true, idx: -1},                           // ft121
		{addr: 0x200041e8, size: 4, signed: true, idx: -1},                           // ft122
		{addr: 0x200041ec, size: 4, signed: true, idx: -1},                           // ft123
		{addr: 0x200041f0, size: 4, signed: true, idx: -1},                           // ft124
		{addr: 0x200041f4, size: 4, signed: true, idx: -1},                           // ft125
		{addr: 0x200041f8, size: 4, signed: true, idx: -1},                           // ft126
		{addr: 0x200041fc, size: 4, signed: true, idx: -1},                           // ft127
		{addr: 0x20004200, size: 4, signed: true, idx: -1},                           // ft128
		{addr: 0x20004204, size: 4, signed: true, idx: -1},                           // ft129
		{addr: 0x20004208, size: 4, signed: true, idx: -1},                           // ft130
		{addr: 0x2000420c, size: 4, signed: true, idx: -1},                           // ft131
		{addr: 0x20004210, size: 4, signed: true, idx: -1},                           // ft132
		{addr: 0x20004214, size: 4, signed: true, idx: -1},                           // ft133
		{addr: 0x20004218, size: 4, signed: true, idx: -1},                           // ft134
		{addr: 0x2000421c, size: 4, signed: true, idx: -1},                           // ft135
		{addr: 0x20004220, size: 4, signed: true, idx: -1},                           // ft136
		{addr: 0x20004224, size: 4, signed: true, idx: -1},                           // ft137
		{addr: 0x20004228, size: 4, signed: true, idx: -1},                           // ft138
		{addr: 0x2000422c, size: 4, signed: true, idx: -1},                           // ft139
		{addr: 0x20004230, size: 4, signed: true, idx: -1},                           // ft140
		{addr: 0x20004234, size: 4, signed: true, idx: -1},                           // ft141
		{addr: 0x20004238, size: 4, signed: true, idx: -1},                           // ft142
		{addr: 0x2000423c, size: 4, signed: true, idx: -1},                           // ft143
		{addr: 0x20004240, size: 4, signed: true, idx: -1},                           // ft144
		{addr: 0x20004244, size: 4, signed: true, idx: -1},                           // ft145
		{addr: 0x20004248, size: 4, signed: true, idx: -1},                           // ft146
		{addr: 0x2000424c, size: 4, signed: true, idx: -1},                           // ft147
		{addr: 0x20004250, size: 4, signed: true, idx: -1},                           // ft148
		{addr: 0x20004254, size: 4, signed: true, idx: -1},                           // ft149
		{addr: 0x20004258, size: 4, signed: true, idx: -1},                           // ft150
		{addr: 0x2000425c, size: 4, signed: true, idx: -1},                           // ft151
		{addr: 0x20004260, size: 4, signed: true, idx: -1},                           // ft152
		{addr: 0x20004264, size: 4, signed: true, idx: -1},                           // ft153
		{addr: 0x20004268, size: 4, signed: true, idx: -1},                           // ft154
		{addr: 0x2000426c, size: 4, signed: true, idx: -1},                           // ft155
		{addr: 0x20004270, size: 4, signed: true, idx: -1},                           // ft156
		{addr: 0x20004274, size: 4, signed: true, idx: -1},                           // ft157
		{addr: 0x20004278, size: 4, signed: true, idx: -1},                           // ft158
		{addr: 0x2000427c, size: 4, signed: true, idx: -1},                           // ft159
		{addr: 0x20004280, size: 4, signed: true, idx: -1},                           // ft160
		{addr: 0x20004284, size: 4, signed: true, idx: -1},                           // ft161
		{addr: 0x20004288, size: 4, signed: true, idx: -1},                           // ft162
		{addr: 0x2000428c, size: 4, signed: true, idx: -1},                           // ft163
		{addr: 0x20004290, size: 4, signed: true, idx: -1},                           // ft164
		{addr: 0x20004294, size: 4, signed: true, idx: -1},                           // ft165
		{addr: 0x20004298, size: 4, signed: true, idx: -1},                           // ft166
		{addr: 0x2000429c, size: 4, signed: true, idx: -1},                           // ft167
		{addr: 0x200042a0, size: 4, signed: true, idx: -1},                           // ft168
		{addr: 0x200042a4, size: 4, signed: true, idx: -1},                           // ft169
		{addr: 0x200042a8, size: 4, signed: true, idx: -1},                           // ft170
		{addr: 0x200042ac, size: 4, signed: true, idx: -1},                           // ft171
		{addr: 0x200042b0, size: 4, signed: true, idx: -1},                           // ft172
		{addr: 0x200042b4, size: 4, signed: true, idx: -1},                           // ft173
		{addr: 0x200042b8, size: 4, signed: true, idx: -1},                           // ft174
		{addr: 0x200042bc, size: 4, signed: true, idx: -1},                           // ft175
		{addr: 0x200042c0, size: 4, signed: true, idx: -1},                           // ft176
		{addr: 0x200042c4, size: 4, signed: true, idx: -1},                           // ft177
		{addr: 0x200042c8, size: 4, signed: true, idx: -1},                           // ft178
		{addr: 0x200042cc, size: 4, signed: true, idx: -1},                           // ft179
		{addr: 0x200042d0, size: 4, signed: true, idx: -1},                           // ft180
		{addr: 0x200042d4, size: 4, signed: true, idx: -1},                           // ft181
		{addr: 0x200042d8, size: 4, signed: true, idx: -1},                           // ft182
		{addr: 0x200042dc, size: 4, signed: true, idx: -1},                           // ft183
		{addr: 0x200042e0, size: 4, signed: true, idx: -1},                           // ft184
		{addr: 0x200042e4, size: 4, signed: true, idx: -1},                           // ft185
		{addr: 0x200042e8, size: 4, signed: true, idx: -1},                           // ft186
		{addr: 0x200042ec, size: 4, signed: true, idx: -1},                           // ft187
		{addr: 0x200042f0, size: 4, signed: true, idx: -1},                           // ft188
		{addr: 0x200042f4, size: 4, signed: true, idx: -1},                           // ft189
		{addr: 0x200042f8, size: 4, signed: true, idx: -1},                           // ft190
		{addr: 0x200042fc, size: 4, signed: true, idx: -1},                           // ft191
		{addr: 0x20004300, size: 4, signed: true, idx: -1},                           // ft192
		{addr: 0x20004304, size: 4, signed: true, idx: -1},                           // ft193
		{addr: 0x20004308, size: 4, signed: true, idx: -1},                           // ft194
		{addr: 0x2000430c, size: 4, signed: true, idx: -1},                           // ft195
		{addr: 0x20004310, size: 4, signed: true, idx: -1},                           // ft196
		{addr: 0x20004314, size: 4, signed: true, idx: -1},                           // ft197
		{addr: 0x20004318, size: 4, signed: true, idx: -1},                           // ft198
		{addr: 0x2000431c, size: 4, signed: true, idx: -1},                           // ft199
		{addr: 0x20004320, size: 4, signed: true, idx: -1},                           // ft200
		{addr: 0x20004324, size: 4, signed: true, idx: -1},                           // ft201
		{addr: 0x20004328, size: 4, signed: true, idx: -1},                           // ft202
		{addr: 0x2000432c, size: 4, signed: true, idx: -1},                           // ft203
		{addr: 0x20004330, size: 4, signed: true, idx: -1},                           // ft204
		{addr: 0x20004334, size: 4, signed: true, idx: -1},                           // ft205
		{addr: 0x20004338, size: 4, signed: true, idx: -1},                           // ft206
		{addr: 0x2000433c, size: 4, signed: true, idx: -1},                           // ft207
		{addr: 0x20004340, size: 4, signed: true, idx: -1},                           // ft208
		{addr: 0x20004344, size: 4, signed: true, idx: -1},                           // ft209
		{addr: 0x20004348, size: 4, signed: true, idx: -1},                           // ft210
		{addr: 0x2000434c, size: 4, signed: true, idx: -1},                           // ft211
		{addr: 0x20004350, size: 4, signed: true, idx: -1},                           // ft212
		{addr: 0x20004354, size: 4, signed: true, idx: -1},                           // ft213
		{addr: 0x20004358, size: 4, signed: true, idx: -1},                           // ft214
		{addr: 0x2000435c, size: 4, signed: true, idx: -1},                           // ft215
		{addr: 0x20004360, size: 4, signed: true, idx: -1},                           // ft216
		{addr: 0x20004364, size: 4, signed: true, idx: -1},                           // ft217
		{addr: 0x20004368, size: 4, signed: true, idx: -1},                           // ft218
		{addr: 0x2000436c, size: 4, signed: true, idx: -1},                           // ft219
		{addr: 0x20004370, size: 4, signed: true, idx: -1},                           // ft220
		{addr: 0x20004374, size: 4, signed: true, idx: -1},                           // ft221
		{addr: 0x20004378, size: 4, signed: true, idx: -1},                           // ft222
		{addr: 0x2000437c, size: 4, signed: true, idx: -1},                           // ft223
		{addr: 0x20004380, size: 4, signed: true, idx: -1},                           // ft224
		{addr: 0x20004384, size: 4, signed: true, idx: -1},                           // ft225
		{addr: 0x20004388, size: 4, signed: true, idx: -1},                           // ft226
		{addr: 0x2000438c, size: 4, signed: true, idx: -1},                           // ft227
		{addr: 0x20004390, size: 4, signed: true, idx: -1},                           // ft228
		{addr: 0x20004394, size: 4, signed: true, idx: -1},                           // ft229
		{addr: 0x20004398, size: 4, signed: true, idx: -1},                           // ft230
		{addr: 0x2000439c, size: 4, signed: true, idx: -1},                           // ft231
		{addr: 0x200043a0, size: 4, signed: true, idx: -1},                           // ft232
		{addr: 0x200043a4, size: 4, signed: true, idx: -1},                           // ft233
		{addr: 0x200043a8, size: 4, signed: true, idx: -1},                           // ft234
		{addr: 0x200043ac, size: 4, signed: true, idx: -1},                           // ft235
		{addr: 0x200043b0, size: 4, signed: true, idx: -1},                           // ft236
		{addr: 0x200043b4, size: 4, signed: true, idx: -1},                           // ft237
		{addr: 0x200043b8, size: 4, signed: true, idx: -1},                           // ft238
		{addr: 0x200043bc, size: 4, signed: true, idx: -1},                           // ft239
		{addr: 0x200043c0, size: 4, signed: true, idx: -1},                           // ft240
		{addr: 0x200043c4, size: 4, signed: true, idx: -1},                           // ft241
		{addr: 0x200043c8, size: 4, signed: true, idx: -1},                           // ft242
		{addr: 0x200043cc, size: 4, signed: true, idx: -1},                           // ft243
		{addr: 0x200043d0, size: 4, signed: true, idx: -1},                           // ft244
		{addr: 0x200043d4, size: 4, signed: true, idx: -1},                           // ft245
		{addr: 0x200043d8, size: 4, signed: true, idx: -1},                           // ft246
		{addr: 0x200043dc, size: 4, signed: true, idx: -1},                           // ft247
		{addr: 0x200043e0, size: 4, signed: true, idx: -1},                           // ft248
		{addr: 0x200043e4, size: 4, signed: true, idx: -1},                           // ft249
		{addr: 0x200043e8, size: 4, signed: true, idx: -1},                           // ft250
		{addr: 0x200043ec, size: 4, signed: true, idx: -1},                           // ft251
		{addr: 0x200043f0, size: 4, signed: true, idx: -1},                           // ft252
		{addr: 0x200043f4, size: 4, signed: true, idx: -1},                           // ft253
		{addr: 0x200043f8, size: 4, signed: true, idx: -1},                           // ft254
		{addr: 0x200043fc, size: 4, signed: true, idx: -1},                           // ft255
		{addr: 0x20004400, size: 4, signed: true, idx: -1},                           // ft256
		{addr: 0x20004404, size: 4, signed: true, idx: -1},                           // ft257
		{addr: 0x20004408, size: 4, signed: true, idx: -1},                           // ft258
		{addr: 0x2000440c, size: 4, signed: true, idx: -1},                           // ft259
		{addr: 0x20004410, size: 4, signed: true, idx: -1},                           // ft260
		{addr: 0x20004414, size: 4, signed: true, idx: -1},                           // ft261
		{addr: 0x20004418, size: 4, signed: true, idx: -1},                           // ft262
		{addr: 0x2000441c, size: 4, signed: true, idx: -1},                           // ft263
		{addr: 0x20004420, size: 4, signed: true, idx: -1},                           // ft264
		{addr: 0x20004424, size: 4, signed: true, idx: -1},                           // ft265
		{addr: 0x20004428, size: 4, signed: true, idx: -1},                           // ft266
		{addr: 0x2000442c, size: 4, signed: true, idx: -1},                           // ft267
		{addr: 0x20004430, size: 4, signed: true, idx: -1},                           // ft268
		{addr: 0x20004434, size: 4, signed: true, idx: -1},                           // ft269
		{addr: 0x20004438, size: 4, signed: true, idx: -1},                           // ft270
		{addr: 0x2000443c, size: 4, signed: true, idx: -1},                           // ft271
		{addr: 0x20004440, size: 4, signed: true, idx: -1},                           // ft272
		{addr: 0x20004444, size: 4, signed: true, idx: -1},                           // ft273
		{addr: 0x20004448, size: 4, signed: true, idx: -1},                           // ft274
		{addr: 0x2000444c, size: 4, signed: true, idx: -1},                           // ft275
		{addr: 0x20004450, size: 4, signed: true, idx: -1},                           // ft276
		{addr: 0x20004454, size: 4, signed: true, idx: -1},                           // ft277
		{addr: 0x20004458, size: 4, signed: true, idx: -1},                           // ft278
		{addr: 0x2000445c, size: 4, signed: true, idx: -1},                           // ft279
		{addr: 0x20004460, size: 4, signed: true, idx: -1},                           // ft280
		{addr: 0x20004464, size: 4, signed: true, idx: -1},                           // ft281
		{addr: 0x20004468, size: 4, signed: true, idx: -1},                           // ft282
		{addr: 0x2000446c, size: 4, signed: true, idx: -1},                           // ft283
		{addr: 0x20004470, size: 4, signed: true, idx: -1},                           // ft284
		{addr: 0x20004474, size: 4, signed: true, idx: -1},                           // ft285
		{addr: 0x20004478, size: 4, signed: true, idx: -1},                           // ft286
		{addr: 0x2000447c, size: 4, signed: true, idx: -1},                           // ft287
		{addr: 0x20004480, size: 4, signed: true, idx: -1},                           // ft288
		{addr: 0x20004484, size: 4, signed: true, idx: -1},                           // ft289
		{addr: 0x20004488, size: 4, signed: true, idx: -1},                           // ft290
		{addr: 0x2000448c, size: 4, signed: true, idx: -1},                           // ft291
		{addr: 0x20004490, size: 4, signed: true, idx: -1},                           // ft292
		{addr: 0x20004494, size: 4, signed: true, idx: -1},                           // ft293
		{addr: 0x20004498, size: 4, signed: true, idx: -1},                           // ft294
		{addr: 0x2000449c, size: 4, signed: true, idx: -1},                           // ft295
		{addr: 0x200044a0, size: 4, signed: true, idx: -1},                           // ft296
		{addr: 0x200044a4, size: 4, signed: true, idx: -1},                           // ft297
		{addr: 0x200044a8, size: 4, signed: true, idx: -1},                           // ft298
		{addr: 0x200044ac, size: 4, signed: true, idx: -1},                           // ft299
		{addr: 0x200044b0, size: 4, signed: true, idx: -1},                           // ft300
		{addr: 0x200044b4, size: 4, signed: true, idx: -1},                           // ft301
		{addr: 0x200044b8, size: 4, signed: true, idx: -1},                           // ft302
		{addr: 0x200044bc, size: 4, signed: true, idx: -1},                           // ft303
		{addr: 0x200044c0, size: 4, signed: true, idx: -1},                           // ft304
	},
	out: []mstFld{
		{addr: 0x20001610, size: 4, signed: false, idx: -1}, // s552250.calls
		{addr: 0x20001614, size: 4, signed: false, idx: -1}, // s552250.a2
		{addr: 0x20001618, size: 4, signed: false, idx: -1}, // s552250.a3
		{addr: 0x2000161c, size: 4, signed: false, idx: -1}, // s552250.a4
		{addr: 0x20001620, size: 4, signed: false, idx: -1}, // s523b30.calls
		{addr: 0x20001624, size: 4, signed: false, idx: -1}, // s523b30.a1
		{addr: 0x20001628, size: 4, signed: false, idx: -1}, // s523b30.a2
		{addr: 0x2000162c, size: 4, signed: false, idx: -1}, // s523b30.a3
		{addr: 0x20001630, size: 4, signed: false, idx: -1}, // s523c00.calls
		{addr: 0x20001634, size: 4, signed: false, idx: -1}, // s523c00.a1
		{addr: 0x20001638, size: 4, signed: false, idx: -1}, // s523c00.a2
		{addr: 0x2000163c, size: 4, signed: false, idx: -1}, // s523c00.a3
		{addr: 0x20001640, size: 4, signed: false, idx: -1}, // s523c00.a4
		{addr: 0x20001644, size: 4, signed: false, idx: -1}, // s523c00.a6
		{addr: 0x20001650, size: 4, signed: false, idx: -1}, // s522fe0.calls
		{addr: 0x20001654, size: 4, signed: false, idx: -1}, // s522fe0.a1
		{addr: 0x20001658, size: 4, signed: false, idx: -1}, // s522fe0.a2
		{addr: 0x2000165c, size: 4, signed: false, idx: -1}, // s522fe0.a3
		{addr: 0x20001660, size: 4, signed: false, idx: -1}, // s522fe0.a4
		{addr: 0x20001664, size: 4, signed: false, idx: -1}, // s522fe0.a5
		{addr: 0x20001668, size: 4, signed: false, idx: -1}, // s522fe0.a6
		{addr: 0x20001670, size: 4, signed: false, idx: -1}, // s58aad0.calls
		{addr: 0x20001674, size: 4, signed: false, idx: -1}, // s58aad0.a1
		{addr: 0x20001678, size: 4, signed: false, idx: -1}, // s58aad0.a2
		{addr: 0x2000167c, size: 4, signed: false, idx: -1}, // s58aad0.a3
		{addr: 0x20001680, size: 4, signed: false, idx: -1}, // s58b410.calls
		{addr: 0x20001684, size: 4, signed: false, idx: -1}, // s58b410.a1
		{addr: 0x20001688, size: 4, signed: false, idx: -1}, // s58b410.a2
		{addr: 0x2000168c, size: 4, signed: false, idx: -1}, // s58b410.a3
		{addr: 0x20001690, size: 4, signed: false, idx: -1}, // s58b410.a4
		{addr: 0x200016a0, size: 4, signed: false, idx: -1}, // s4acd80.calls
		{addr: 0x200016a4, size: 4, signed: false, idx: -1}, // s4acd80.a1
		{addr: 0x200016a8, size: 4, signed: false, idx: -1}, // s4acd80.a2
		{addr: 0x200016ac, size: 4, signed: false, idx: -1}, // s4acd80.a3
		{addr: 0x200016b0, size: 4, signed: false, idx: -1}, // s5182e0.calls
		{addr: 0x200016b4, size: 4, signed: false, idx: -1}, // s5182e0.a1
		{addr: 0x200016b8, size: 4, signed: false, idx: -1}, // s5182e0.a2
		{addr: 0x200016bc, size: 4, signed: false, idx: -1}, // s5182e0.a3
		{addr: 0x200004e3, size: 4, signed: false, idx: -1}, // M.buff5.power
		{addr: 0x200004e7, size: 4, signed: false, idx: -1}, // M.buff5.mult
		{addr: 0x200004eb, size: 4, signed: false, idx: -1}, // M.buff5.dur
		{addr: 0x200005db, size: 1, signed: false, idx: -1}, // M.mod00
		{addr: 0x200005dc, size: 1, signed: false, idx: -1}, // M.mod01
		{addr: 0x200005dd, size: 1, signed: false, idx: -1}, // M.mod02
		{addr: 0x200005de, size: 1, signed: false, idx: -1}, // M.mod03
		{addr: 0x200005df, size: 1, signed: false, idx: -1}, // M.mod04
		{addr: 0x200005e0, size: 1, signed: false, idx: -1}, // M.mod05
		{addr: 0x200005e1, size: 1, signed: false, idx: -1}, // M.mod06
		{addr: 0x200005e2, size: 1, signed: false, idx: -1}, // M.mod07
		{addr: 0x200005e3, size: 1, signed: false, idx: -1}, // M.mod08
		{addr: 0x200005e4, size: 1, signed: false, idx: -1}, // M.mod09
		{addr: 0x200005e5, size: 1, signed: false, idx: -1}, // M.mod0a
		{addr: 0x200005e6, size: 1, signed: false, idx: -1}, // M.mod0b
		{addr: 0x200005e7, size: 1, signed: false, idx: -1}, // M.mod0c
		{addr: 0x200005e8, size: 1, signed: false, idx: -1}, // M.mod0d
		{addr: 0x200005e9, size: 1, signed: false, idx: -1}, // M.mod0e
		{addr: 0x200005ea, size: 1, signed: false, idx: -1}, // M.mod0f
		{addr: 0x200005eb, size: 1, signed: false, idx: -1}, // M.mod10
		{addr: 0x200005ec, size: 1, signed: false, idx: -1}, // M.mod11
		{addr: 0x200005ed, size: 1, signed: false, idx: -1}, // M.mod12
		{addr: 0x200005ee, size: 1, signed: false, idx: -1}, // M.mod13
		{addr: 0x200005ef, size: 1, signed: false, idx: -1}, // M.mod14
		{addr: 0x200005f0, size: 1, signed: false, idx: -1}, // M.mod15
		{addr: 0x200005f1, size: 1, signed: false, idx: -1}, // M.mod16
		{addr: 0x200005f2, size: 1, signed: false, idx: -1}, // M.mod17
		{addr: 0x200005f3, size: 1, signed: false, idx: -1}, // M.mod18
		{addr: 0x200005f4, size: 1, signed: false, idx: -1}, // M.mod19
		{addr: 0x200005f5, size: 1, signed: false, idx: -1}, // M.mod1a
		{addr: 0x200005f6, size: 1, signed: false, idx: -1}, // M.mod1b
		{addr: 0x200005f7, size: 1, signed: false, idx: -1}, // M.mod1c
		{addr: 0x200005f8, size: 1, signed: false, idx: -1}, // M.mod1d
		{addr: 0x200005f9, size: 1, signed: false, idx: -1}, // M.mod1e
		{addr: 0x200005fa, size: 1, signed: false, idx: -1}, // M.mod1f
		{addr: 0x200005fb, size: 1, signed: false, idx: -1}, // M.mod20
		{addr: 0x200005fc, size: 1, signed: false, idx: -1}, // M.mod21
		{addr: 0x200005fd, size: 1, signed: false, idx: -1}, // M.mod22
		{addr: 0x200005fe, size: 1, signed: false, idx: -1}, // M.mod23
		{addr: 0x200005ff, size: 1, signed: false, idx: -1}, // M.mod24
		{addr: 0x20000600, size: 1, signed: false, idx: -1}, // M.mod25
		{addr: 0x20000601, size: 1, signed: false, idx: -1}, // M.mod26
		{addr: 0x20000602, size: 1, signed: false, idx: -1}, // M.mod27
		{addr: 0x20000603, size: 1, signed: false, idx: -1}, // M.mod28
		{addr: 0x20000604, size: 1, signed: false, idx: -1}, // M.mod29
		{addr: 0x20000605, size: 1, signed: false, idx: -1}, // M.mod2a
		{addr: 0x20000606, size: 1, signed: false, idx: -1}, // M.mod2b
		{addr: 0x20000607, size: 1, signed: false, idx: -1}, // M.mod2c
		{addr: 0x20000608, size: 1, signed: false, idx: -1}, // M.mod2d
		{addr: 0x20000609, size: 1, signed: false, idx: -1}, // M.mod2e
		{addr: 0x2000060a, size: 1, signed: false, idx: -1}, // M.mod2f
		{addr: 0x2000060b, size: 1, signed: false, idx: -1}, // M.mod30
		{addr: 0x2000060c, size: 1, signed: false, idx: -1}, // M.mod31
		{addr: 0x2000060d, size: 1, signed: false, idx: -1}, // M.mod32
		{addr: 0x2000060e, size: 1, signed: false, idx: -1}, // M.mod33
		{addr: 0x2000060f, size: 1, signed: false, idx: -1}, // M.mod34
		{addr: 0x20000610, size: 1, signed: false, idx: -1}, // M.mod35
		{addr: 0x20000611, size: 1, signed: false, idx: -1}, // M.mod36
		{addr: 0x20000612, size: 1, signed: false, idx: -1}, // M.mod37
		{addr: 0x20000613, size: 1, signed: false, idx: -1}, // M.mod38
		{addr: 0x20000614, size: 1, signed: false, idx: -1}, // M.mod39
		{addr: 0x20000615, size: 1, signed: false, idx: -1}, // M.mod3a
		{addr: 0x20000616, size: 1, signed: false, idx: -1}, // M.mod3b
		{addr: 0x20000617, size: 1, signed: false, idx: -1}, // M.mod3c
		{addr: 0x20000618, size: 1, signed: false, idx: -1}, // M.mod3d
		{addr: 0x20000619, size: 1, signed: false, idx: -1}, // M.mod3e
		{addr: 0x2000061a, size: 1, signed: false, idx: -1}, // M.mod3f
		{addr: 0x2000061b, size: 1, signed: false, idx: -1}, // M.mod40
		{addr: 0x2000061c, size: 1, signed: false, idx: -1}, // M.mod41
		{addr: 0x2000061d, size: 1, signed: false, idx: -1}, // M.mod42
		{addr: 0x2000061e, size: 1, signed: false, idx: -1}, // M.mod43
		{addr: 0x2000061f, size: 1, signed: false, idx: -1}, // M.mod44
		{addr: 0x20000620, size: 1, signed: false, idx: -1}, // M.mod45
		{addr: 0x20000621, size: 1, signed: false, idx: -1}, // M.mod46
		{addr: 0x20000622, size: 1, signed: false, idx: -1}, // M.mod47
		{addr: 0x20000623, size: 1, signed: false, idx: -1}, // M.mod48
		{addr: 0x20000624, size: 1, signed: false, idx: -1}, // M.mod49
		{addr: 0x20000625, size: 1, signed: false, idx: -1}, // M.mod4a
		{addr: 0x20000626, size: 1, signed: false, idx: -1}, // M.mod4b
		{addr: 0x20000627, size: 1, signed: false, idx: -1}, // M.mod4c
		{addr: 0x20000628, size: 1, signed: false, idx: -1}, // M.mod4d
		{addr: 0x20000629, size: 1, signed: false, idx: -1}, // M.mod4e
		{addr: 0x2000062a, size: 1, signed: false, idx: -1}, // M.mod4f
		{addr: 0x2000062b, size: 1, signed: false, idx: -1}, // M.mod50
		{addr: 0x2000062c, size: 1, signed: false, idx: -1}, // M.mod51
		{addr: 0x2000062d, size: 1, signed: false, idx: -1}, // M.mod52
		{addr: 0x2000062e, size: 1, signed: false, idx: -1}, // M.mod53
		{addr: 0x2000062f, size: 1, signed: false, idx: -1}, // M.mod54
		{addr: 0x20000630, size: 1, signed: false, idx: -1}, // M.mod55
		{addr: 0x20000631, size: 1, signed: false, idx: -1}, // M.mod56
		{addr: 0x20000632, size: 1, signed: false, idx: -1}, // M.mod57
		{addr: 0x20000633, size: 1, signed: false, idx: -1}, // M.mod58
		{addr: 0x20000634, size: 1, signed: false, idx: -1}, // M.mod59
		{addr: 0x20000635, size: 1, signed: false, idx: -1}, // M.mod5a
		{addr: 0x20000636, size: 1, signed: false, idx: -1}, // M.mod5b
		{addr: 0x20000637, size: 1, signed: false, idx: -1}, // M.mod5c
		{addr: 0x20000638, size: 1, signed: false, idx: -1}, // M.mod5d
		{addr: 0x20000639, size: 1, signed: false, idx: -1}, // M.mod5e
		{addr: 0x2000063a, size: 1, signed: false, idx: -1}, // M.mod5f
		{addr: 0x2000063b, size: 1, signed: false, idx: -1}, // M.mod60
		{addr: 0x2000063c, size: 1, signed: false, idx: -1}, // M.mod61
		{addr: 0x2000063d, size: 1, signed: false, idx: -1}, // M.mod62
		{addr: 0x2000063e, size: 1, signed: false, idx: -1}, // M.mod63
		{addr: 0x2000063f, size: 1, signed: false, idx: -1}, // M.mod64
		{addr: 0x20000640, size: 1, signed: false, idx: -1}, // M.mod65
		{addr: 0x20000641, size: 1, signed: false, idx: -1}, // M.mod66
		{addr: 0x20000647, size: 1, signed: false, idx: -1}, // M.attr0
		{addr: 0x20000648, size: 1, signed: false, idx: -1}, // M.attr1
		{addr: 0x20000649, size: 1, signed: false, idx: -1}, // M.attr2
		{addr: 0x2000064a, size: 1, signed: false, idx: -1}, // M.attr3
		{addr: 0x2000064b, size: 1, signed: false, idx: -1}, // M.attr4
		{addr: 0x20000447, size: 4, signed: false, idx: -1}, // M.hpRecovery
		{addr: 0x2000044f, size: 4, signed: false, idx: -1}, // M.spRecovery
		{addr: 0x200008eb, size: 4, signed: false, idx: -1}, // M2.buff5.dur
	},
	stub: []mstStubAt{
		{ret: 0x0, rec: [][3]uint32{{2, 4, 0x20001614}, {3, 4, 0x20001618}, {4, 4, 0x2000161c}}, cnt: 0x20001610},                                                                    // 0x552250
		{ret: 0x0, rec: [][3]uint32{{1, 4, 0x20001624}, {2, 4, 0x20001628}, {3, 4, 0x2000162c}}, cnt: 0x20001620},                                                                    // 0x523b30
		{ret: 0x0, rec: [][3]uint32{{1, 4, 0x20001634}, {2, 4, 0x20001638}, {3, 4, 0x2000163c}, {4, 4, 0x20001640}, {6, 4, 0x20001644}}, cnt: 0x20001630},                            // 0x523c00
		{ret: 0x20001604, rec: [][3]uint32{{1, 4, 0x20001654}, {2, 4, 0x20001658}, {3, 4, 0x2000165c}, {4, 4, 0x20001660}, {5, 4, 0x20001664}, {6, 4, 0x20001668}}, cnt: 0x20001650}, // 0x522fe0
		{ret: 0x0, rec: [][3]uint32{{1, 4, 0x20001674}, {2, 4, 0x20001678}, {3, 4, 0x2000167c}}, cnt: 0x20001670},                                                                    // 0x58aad0
		{ret: 0x0, rec: [][3]uint32{{1, 4, 0x20001684}, {2, 4, 0x20001688}, {3, 4, 0x2000168c}, {4, 4, 0x20001690}}, cnt: 0x20001680},                                                // 0x58b410
		{ret: 0x0, rec: [][3]uint32{{1, 4, 0x200016a4}, {2, 4, 0x200016a8}, {3, 4, 0x200016ac}}, cnt: 0x200016a0},                                                                    // 0x4acd80
		{ret: 0x0, rec: [][3]uint32{{1, 4, 0x200016b4}, {2, 4, 0x200016b8}, {3, 4, 0x200016bc}}, cnt: 0x200016b0},                                                                    // 0x5182e0
	},
}

var sdrIn = []string{"cond", "realm", "strength", "penalty", "duration", "flagB", "message", "target.type", "charIdx", "monId", "r552250", "r522fe0", "C.buff5.dur", "C.name0", "C.name1", "C.name2", "P.present", "P.f1", "text0", "text1", "M.group", "M.inCombat", "M.maxHP", "M.maxSP", "M.cond0", "M.cond1", "M.cond2", "M.cond3", "M.cond4", "M.cond5", "M.cond6", "M.cond7", "M.cond8", "M.cond9", "M.cond10", "M.cond11", "M.cond12", "M.cond13", "M.cond14", "M.cond15", "M.cond16", "M.cond17", "M.cond18", "M.buff0.power", "M.buff0.mult", "M.buff0.dur", "M.buff1.power", "M.buff1.mult", "M.buff1.dur", "M.buff2.power", "M.buff2.mult", "M.buff2.dur", "M.buff3.power", "M.buff3.mult", "M.buff3.dur", "M.buff4.power", "M.buff4.mult", "M.buff4.dur", "M.buff5.power", "M.buff5.mult", "M.buff5.dur", "M.buff6.power", "M.buff6.mult", "M.buff6.dur", "M.buff7.power", "M.buff7.mult", "M.buff7.dur", "M.attackPenalty", "M.eff0.active", "M.eff0.type", "M.eff0.power", "M.eff0.mult", "M.eff1.active", "M.eff1.type", "M.eff1.power", "M.eff1.mult", "M.eff2.active", "M.eff2.type", "M.eff2.power", "M.eff2.mult", "M.eff3.active", "M.eff3.type", "M.eff3.power", "M.eff3.mult", "M.eff4.active", "M.eff4.type", "M.eff4.power", "M.eff4.mult", "M.eff5.active", "M.eff5.type", "M.eff5.power", "M.eff5.mult", "M.eff6.active", "M.eff6.type", "M.eff6.power", "M.eff6.mult", "M.eff7.active", "M.eff7.type", "M.eff7.power", "M.eff7.mult", "M.eff8.active", "M.eff8.type", "M.eff8.power", "M.eff8.mult", "M.eff9.active", "M.eff9.type", "M.eff9.power", "M.eff9.mult", "M.eff10.active", "M.eff10.type", "M.eff10.power", "M.eff10.mult", "M.eff11.active", "M.eff11.type", "M.eff11.power", "M.eff11.mult", "tpl.attr0", "tpl.attr1", "tpl.attr2", "tpl.attr3", "tpl.attr4", "tpl.species", "actor.fa4", "actor.fa7", "T.sp0.active", "T.sp0.code", "T.sp0.value", "T.sp1.active", "T.sp1.code", "T.sp1.value", "T.sp2.active", "T.sp2.code", "T.sp2.value", "T.sp3.active", "T.sp3.code", "T.sp3.value", "T.sp4.active", "T.sp4.code", "T.sp4.value", "T.sp5.active", "T.sp5.code", "T.sp5.value", "T.sp6.active", "T.sp6.code", "T.sp6.value", "T.sp7.active", "T.sp7.code", "T.sp7.value", "T.sp8.active", "T.sp8.code", "T.sp8.value", "T.aura0.active", "T.aura0.code", "T.aura0.power", "T.aura0.mult", "T.aura1.active", "T.aura1.code", "T.aura1.power", "T.aura1.mult", "T.aura2.active", "T.aura2.code", "T.aura2.power", "T.aura2.mult", "T.aura3.active", "T.aura3.code", "T.aura3.power", "T.aura3.mult", "T.aura4.active", "T.aura4.code", "T.aura4.power", "T.aura4.mult", "T.aura5.active", "T.aura5.code", "T.aura5.power", "T.aura5.mult", "grpA.f2c", "grpB.f2c", "noticeFmt0", "noticeFmt1", "suffix0", "fRj", "fLt", "ft0", "ft1", "ft2", "ft3", "ft4", "ft5", "ft6", "ft7", "ft8", "ft9", "ft10", "ft11", "ft12", "ft13", "ft14", "ft15", "ft16", "ft17", "ft18", "ft19", "ft20", "ft21", "ft22", "ft23", "ft24", "ft25", "ft26", "ft27", "ft28", "ft29", "ft30", "ft31", "ft32", "ft33", "ft34", "ft35", "ft36", "ft37", "ft38", "ft39", "ft40", "ft41", "ft42", "ft43", "ft44", "ft45", "ft46", "ft47", "ft48", "ft49", "ft50", "ft51", "ft52", "ft53", "ft54", "ft55", "ft56", "ft57", "ft58", "ft59", "ft60", "ft61", "ft62", "ft63", "ft64", "ft65", "ft66", "ft67", "ft68", "ft69", "ft70", "ft71", "ft72", "ft73", "ft74", "ft75", "ft76", "ft77", "ft78", "ft79", "ft80", "ft81", "ft82", "ft83", "ft84", "ft85", "ft86", "ft87", "ft88", "ft89", "ft90", "ft91", "ft92", "ft93", "ft94", "ft95", "ft96", "ft97", "ft98", "ft99", "ft100", "ft101", "ft102", "ft103", "ft104", "ft105", "ft106", "ft107", "ft108", "ft109", "ft110", "ft111", "ft112", "ft113", "ft114", "ft115", "ft116", "ft117", "ft118", "ft119", "ft120", "ft121", "ft122", "ft123", "ft124", "ft125", "ft126", "ft127", "ft128", "ft129", "ft130", "ft131", "ft132", "ft133", "ft134", "ft135", "ft136", "ft137", "ft138", "ft139", "ft140", "ft141", "ft142", "ft143", "ft144", "ft145", "ft146", "ft147", "ft148", "ft149", "ft150", "ft151", "ft152", "ft153", "ft154", "ft155", "ft156", "ft157", "ft158", "ft159", "ft160", "ft161", "ft162", "ft163", "ft164", "ft165", "ft166", "ft167", "ft168", "ft169", "ft170", "ft171", "ft172", "ft173", "ft174", "ft175", "ft176", "ft177", "ft178", "ft179", "ft180", "ft181", "ft182", "ft183", "ft184", "ft185", "ft186", "ft187", "ft188", "ft189", "ft190", "ft191", "ft192", "ft193", "ft194", "ft195", "ft196", "ft197", "ft198", "ft199", "ft200", "ft201", "ft202", "ft203", "ft204", "ft205", "ft206", "ft207", "ft208", "ft209", "ft210", "ft211", "ft212", "ft213", "ft214", "ft215", "ft216", "ft217", "ft218", "ft219", "ft220", "ft221", "ft222", "ft223", "ft224", "ft225", "ft226", "ft227", "ft228", "ft229", "ft230", "ft231", "ft232", "ft233", "ft234", "ft235", "ft236", "ft237", "ft238", "ft239", "ft240", "ft241", "ft242", "ft243", "ft244", "ft245", "ft246", "ft247", "ft248", "ft249", "ft250", "ft251", "ft252", "ft253", "ft254", "ft255", "ft256", "ft257", "ft258", "ft259", "ft260", "ft261", "ft262", "ft263", "ft264", "ft265", "ft266", "ft267", "ft268", "ft269", "ft270", "ft271", "ft272", "ft273", "ft274", "ft275", "ft276", "ft277", "ft278", "ft279", "ft280", "ft281", "ft282", "ft283", "ft284", "ft285", "ft286", "ft287", "ft288", "ft289", "ft290", "ft291", "ft292", "ft293", "ft294", "ft295", "ft296", "ft297", "ft298", "ft299", "ft300", "ft301", "ft302", "ft303", "ft304"}

var sdrOut = []string{"al", "s552250.calls", "s552250.a2", "s552250.a3", "s552250.a4", "s523b30.calls", "s523b30.a1", "s523b30.a2", "s523b30.a3", "s523c00.calls", "s523c00.a1", "s523c00.a2", "s523c00.a3", "s523c00.a4", "s523c00.a6", "s522fe0.calls", "s522fe0.a1", "s522fe0.a2", "s522fe0.a3", "s522fe0.a4", "s522fe0.a5", "s522fe0.a6", "s58aad0.calls", "s58aad0.a1", "s58aad0.a2", "s58aad0.a3", "s58b410.calls", "s58b410.a1", "s58b410.a2", "s58b410.a3", "s58b410.a4", "s4acd80.calls", "s4acd80.a1", "s4acd80.a2", "s4acd80.a3", "s5182e0.calls", "s5182e0.a1", "s5182e0.a2", "s5182e0.a3", "M.buff5.power", "M.buff5.mult", "M.buff5.dur", "M.mod00", "M.mod01", "M.mod02", "M.mod03", "M.mod04", "M.mod05", "M.mod06", "M.mod07", "M.mod08", "M.mod09", "M.mod0a", "M.mod0b", "M.mod0c", "M.mod0d", "M.mod0e", "M.mod0f", "M.mod10", "M.mod11", "M.mod12", "M.mod13", "M.mod14", "M.mod15", "M.mod16", "M.mod17", "M.mod18", "M.mod19", "M.mod1a", "M.mod1b", "M.mod1c", "M.mod1d", "M.mod1e", "M.mod1f", "M.mod20", "M.mod21", "M.mod22", "M.mod23", "M.mod24", "M.mod25", "M.mod26", "M.mod27", "M.mod28", "M.mod29", "M.mod2a", "M.mod2b", "M.mod2c", "M.mod2d", "M.mod2e", "M.mod2f", "M.mod30", "M.mod31", "M.mod32", "M.mod33", "M.mod34", "M.mod35", "M.mod36", "M.mod37", "M.mod38", "M.mod39", "M.mod3a", "M.mod3b", "M.mod3c", "M.mod3d", "M.mod3e", "M.mod3f", "M.mod40", "M.mod41", "M.mod42", "M.mod43", "M.mod44", "M.mod45", "M.mod46", "M.mod47", "M.mod48", "M.mod49", "M.mod4a", "M.mod4b", "M.mod4c", "M.mod4d", "M.mod4e", "M.mod4f", "M.mod50", "M.mod51", "M.mod52", "M.mod53", "M.mod54", "M.mod55", "M.mod56", "M.mod57", "M.mod58", "M.mod59", "M.mod5a", "M.mod5b", "M.mod5c", "M.mod5d", "M.mod5e", "M.mod5f", "M.mod60", "M.mod61", "M.mod62", "M.mod63", "M.mod64", "M.mod65", "M.mod66", "M.attr0", "M.attr1", "M.attr2", "M.attr3", "M.attr4", "M.hpRecovery", "M.spRecovery", "M2.buff5.dur"}
