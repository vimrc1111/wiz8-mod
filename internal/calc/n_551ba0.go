package calc

// status_immunity_and_resist 0x551ba0(src, T, status, realm, strength, penalty,
// duration, message, durationMessage, durationFlagB): does target T shrug off a
// status. T is a target record (kind at +0: 1 = character, otherwise a monster
// looked up by the id at +8; +4 character index); src is the source record.
//
//	character: primary state u32 C+0xb01.
//	monster:   0x4e5840 finds the live monster by id, 0x4e5720 maps its template
//	           (monster +8, cache 0x6840c7). Template byte +0x1be set and status 0x12
//	           resists; template byte +0xcb matching a category row of the exe table
//	           0x6171a8 (3 rows of 0x52 bytes: category byte, 20 status dwords) whose
//	           list holds the status resists. Primary state is monster +0x107.
//	primary >= 0x12 (unsigned) resists. Status 0x13: a monster target (kind 3)
//	resists, any other kind resists when C+0xa4d is non-zero.
//	0x5520d0(T, realm, strength, status) non-zero: resisted, notice.
//	character, status 2..15: exe jump bytes 0x551e98 / targets 0x551e84 select
//	  0x551cd6 feature 3 (notice text +0x600 when message) else 0x551d27,
//	  0x551d27 feature 0x1e resists silently,
//	  0x551d58 duration 0x270f goes on, else 0x551d60,
//	  0x551d60 feature 0xe resists (notice text +0x604 with the status name, text id
//	  u16 at exe 0x61e570+8*status, when message); features are 0x547940.
//	otherwise 0x551eb0(T, status, realm, strength, penalty, duration, k, durationFlagB,
//	  durationMessage), k = source character index when T is kind 3 and 0x53bea0(src, 0)
//	  (source kind 1; index -1 asserts at 0x53beb1), else -1. Non-zero: not resisted (0).
//	resisted: when message and byte 0x68510c, monster targets (kind 3) go to 0x590b40
//	  (monster record, text +0x6cc), others to 0x590950(index, text +0x6cc). Result 1.
//
// 0x590950(index, fmt, ...) runs natively: %s formatting into a buffer, a suffix
// (exe 0x689b34 after ' or :, else 0x60cc74), 0x58aad0(8, 0x64bf5c, name C+5), then
// 0x58b410(member byte *0x685178+0x106*index+0xf1, 0, name length (+1 after ', +1 more
// for 's), -1). 0x58aad0, 0x58b410 and 0x590b40 are 排除 address stubs.
//
// Emu address-stubs the D4 callees 0x5520d0 (return input resistResult) and 0x551eb0
// (return input durationResult); their arguments and call counts are outputs. F runs
// statusResistRoll on the same target inputs and statusDurationAfterResistNode on the
// node's arguments plus F-only inputs (sdr.*: 0x551eb0's own world after its ten
// argument and target fields, dead in Emu), using the low byte of each.
//
// Fixture caps (panic past them): character index 0..7, match slot 0..3 (the live
// monster is slot matchSlot of two 2-entry lists, the other slots hold decoy ids
// >= 0xffffff00), monster id < 0xffffff00, template index 0..2, class -1..14 and race
// -1..15 (feature table rows), wide strings 0x1000 units, formats with %s only.
// World layout: re/emu/cases/00551ba0.json (text strings and names are fixture data).

import "fmt"

const (
	sirArena     = 0x20000000
	sirTgt       = sirArena + 0x0
	sirSrc       = sirArena + 0x40
	sirScratch   = sirArena + 0x100 // stub returns, records and counters
	sirListA     = sirArena + 0x200
	sirEntA      = sirArena + 0x220
	sirListB     = sirArena + 0x240
	sirEntB      = sirArena + 0x260
	sirTxt       = sirArena + 0x1000 // fixture text pointer table
	sirStr       = sirArena + 0x2000
	sirMon       = sirArena + 0x4000
	sirTemplate  = sirArena + 0x6000
	sirSlots     = sirArena + 0x7000
	sirChars     = sirArena + 0x10000
	sirFonly     = sirArena + 0x20000 // F-only 0x551eb0 world cells (no reader in Emu)
	sirMonSz     = 1024
	sirTplSz     = 663
	sirSlotSz    = 0x106 // 0x590950 member stride
	sirChar      = 0x1862
	sirTemplates = 3
	sirSlotsN    = 4
	sirDecoy     = 0xffffff00
	sirWideCap   = 0x1000

	sirCatTable = 0x6171aa // first category row's status list (byte at -2), rows of 0x52 up to 0x6172a0
	sirCatEnd   = 0x6172a0
	sirJumpSel  = 0x551e98 // switch byte per status-2
	sirJumpTab  = 0x551e84
	sirTextIDs  = 0x61e570 // u16 text id per status, stride 8
	sirTextTab  = 0x68c09c
	sirMsgOn    = 0x68510c
	sirQuote    = 0x689b34
	sirSpace    = 0x60cc74
	sirCharFmt  = 0x64bf5c
	sirSdrSkip  = 10 // 0x551eb0 inputs before its world (sdrIn order)
)

const (
	sirKResist = iota // 0x5520d0
	sirKDur           // 0x551eb0
	sirKMon           // 0x590b40
	sirKLog           // 0x58aad0
	sirKColor         // 0x58b410
)

// Input indexes (sirIn order).
const (
	sirStatus = iota
	sirRealm
	sirStrength
	sirPenalty
	sirDuration
	sirMessage
	sirDurMessage
	sirDurFlagB
	sirSrcKind
	sirSrcIndex
	sirKind
	sirIdx
	sirMonID
	sirSlot
	sirCharPrimary
	sirStatus13
	sirClass
	sirRace
	sirColor
	sirTplIndex
	sirImmune12
	sirCategory
	sirMonPrimary
	sirResist
	sirDurResult
	sirMsgEnabled
	sirDifficulty
	sirCharLevel
	sirCharRes
)

const (
	sirGrowth   = sirCharRes + 6
	sirReverse  = sirGrowth + 1
	sirTplRes   = sirGrowth + 2
	sirTplLevel = sirTplRes + 6
	sirMonMod   = sirTplLevel + 1
	sirMonClass = sirMonMod + 6
	sirOwn      = sirMonClass + 1 // first F-only 0x551eb0 input
)

var sirIn, sirSpec = sirLayout()

func sirLayout() ([]string, mstSpec) {
	var names []string
	var s mstSpec
	fld := func(name string, addr uint32, size uint8, signed bool) {
		names = append(names, name)
		s.in = append(s.in, mstFld{addr: addr, size: size, signed: signed, idx: -1})
	}
	rec := func(name string, base, off uint32, size uint8, signed bool, idx int32, count, stride uint32) {
		names = append(names, name)
		s.in = append(s.in, mstFld{addr: base + off, size: size, signed: signed, idx: idx, count: count, stride: stride})
	}
	arg := func(n uint32) uint32 { return mstStackBase + 4 + 4*n }
	for i, n := range []string{"status", "realm", "strength", "penalty", "duration", "message", "durationMessage", "durationFlagB"} {
		fld(n, arg(uint32(i)+2), 4, false)
	}
	fld("sourceKind", sirSrc, 4, false)
	fld("sourceIndex", sirSrc+4, 4, true)
	fld("targetKind", sirTgt, 4, false)
	fld("characterIndex", sirTgt+4, 4, true)
	fld("monsterID", sirTgt+8, 4, false)
	fld("matchSlot", sirScratch+0xe0, 4, false)
	ch := func(n string, off uint32, size uint8, signed bool) {
		rec(n, sirChars, off, size, signed, sirIdx, 8, sirChar)
	}
	mon := func(n string, off uint32, size uint8, signed bool) {
		rec(n, sirMon, off, size, signed, sirSlot, sirSlotsN, sirMonSz)
	}
	tpl := func(n string, off uint32, size uint8) {
		rec(n, sirTemplate, off, size, false, sirTplIndex, sirTemplates, sirTplSz)
	}
	ch("characterPrimaryState", 0xb01, 4, false)
	ch("characterStatus13Block", 0xa4d, 4, false)
	ch("characterClass", 0x69, 4, true)
	ch("characterRace", 0x71, 4, true)
	rec("memberNameColor", sirSlots, 0xf1, 1, false, sirIdx, 8, sirSlotSz)
	mon("templateIndex", 8, 4, false)
	tpl("templateStatus12Immune", 0x1be, 1)
	tpl("templateCategory", 0xcb, 1)
	mon("monsterPrimaryState", 0x107, 4, false)
	fld("resistResult", sirScratch, 4, false)
	fld("durationResult", sirScratch+4, 4, false)
	fld("messagesEnabled", sirMsgOn, 1, false)
	// 0x5520d0 inputs only F reads, at their real places.
	fld("difficulty", 0x6850d5, 4, false)
	ch("characterLevel", 0x89, 4, false)
	realms := []string{"fire", "water", "air", "earth", "mental", "divine"}
	for k, n := range realms {
		ch("characterResistance_"+n, 0xee1+16*uint32(k), 4, false)
	}
	ch("characterSkillGrowthFlag", 0x6f5, 1, false)
	ch("characterReverseDifficulty", 0xa35, 4, false)
	for k, n := range realms {
		tpl("templateResistance_"+n, 0x176+uint32(k), 1)
	}
	tpl("templateLevel", 0x24f, 1)
	for k, n := range realms {
		mon("monsterResistanceModifier_"+n, 0x217+uint32(k), 1, true)
	}
	mon("monsterDifficultyClass", 0x16, 1, false)
	if len(names) != sirOwn {
		panic("status immunity and resist: input layout")
	}
	for i, field := range sdrParentFields[sirSdrSkip:] {
		f := sdrSpec.in[field]
		fld("sdr."+sdrIn[field], sirFonly+4*uint32(i), 4, f.signed)
	}

	// Stubs: return at +0 / +4, records and counters in the scratch object.
	st := func(ret, cnt, base uint32, args ...uint32) {
		a := mstStubAt{ret: ret, cnt: sirScratch + cnt}
		for k, n := range args {
			a.rec = append(a.rec, [3]uint32{n, 4, sirScratch + base + 4*uint32(k)})
		}
		s.stub = append(s.stub, a)
	}
	st(sirScratch, 0x10, 0x14, 1, 2, 3, 4)
	st(sirScratch+4, 0x40, 0x44, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	st(0, 0x80, 0x84, 1, 2)
	st(0, 0xa0, 0xa4, 1, 2, 3)
	st(0, 0xc0, 0xc4, 1, 2, 3, 4)
	for _, a := range s.stub {
		s.out = append(s.out, mstFld{addr: a.cnt, size: 4, idx: -1})
		for _, r := range a.rec {
			s.out = append(s.out, mstFld{addr: r[2], size: 4, idx: -1})
		}
	}

	set := func(a uint32, size uint8, v uint32) { s.set = append(s.set, mstSet{a, size, v}) }
	set(arg(0), 4, sirSrc) // case "args": ["@src", "@tgt", 0, ...]
	set(arg(1), 4, sirTgt)
	set(0x685174, 4, sirChars)
	set(0x685178, 4, sirSlots)
	set(0x683fad, 4, sirListA)
	set(0x683fb9, 4, sirListB)
	set(sirListA, 4, sirEntA)
	set(sirListA+8, 4, 2)
	set(sirListB, 4, sirEntB)
	set(sirListB+8, 4, 2)
	for k := uint32(0); k < sirSlotsN; k++ {
		ent := uint32(sirEntA)
		if k >= 2 {
			ent = sirEntB
		}
		set(ent+4*(k%2), 4, sirMon+sirMonSz*k)
		set(sirMon+sirMonSz*k, 4, sirDecoy+k) // decoy ids; the monsterID input overwrites slot matchSlot
	}
	for k := uint32(0); k < sirTemplates; k++ {
		set(0x6840c7+4*k, 4, sirTemplate+sirTplSz*k)
	}
	// decoys for the double-indexed fields: inputs overwrite only the selected monster / template
	for k := uint32(0); k < sirTemplates; k++ {
		set(sirTemplate+sirTplSz*k+0x1be, 1, 1)
		set(sirTemplate+sirTplSz*k+0xcb, 1, 20)
	}
	for k := uint32(0); k < sirSlotsN; k++ {
		set(sirMon+sirMonSz*k+8, 4, (k+1)%sirTemplates)
		set(sirMon+sirMonSz*k+0x107, 4, 0x12)
	}
	set(sirTextTab, 4, sirTxt)
	// fixture text (the game's text table is runtime data): 0x180 = +0x600,
	// 0x181 = +0x604, 0x1b3 = +0x6cc, 884 / 892 = the text ids of statuses 11 / 13.
	wide := func(a uint32, t string) {
		for i, c := range t + "\x00" {
			set(a+2*uint32(i), 2, uint32(c))
		}
	}
	off := uint32(0)
	for _, t := range []struct {
		id  uint32
		txt string
	}{{0x180, ":A"}, {0x181, "%s!"}, {0x1b3, "'t"}, {884, "'s x"}, {892, "Zz"}} {
		set(sirTxt+4*t.id, 4, sirStr+off)
		wide(sirStr+off, t.txt)
		off += 0x40
	}
	for c, n := range []string{"A", "Bo", "Cid", "Dana", "Ed", "Fay", "Gus", "Hal"} {
		wide(sirChars+sirChar*uint32(c)+5, n)
	}
	return names, s
}

var sirOut = func() []string {
	o := []string{"resisted"}
	for _, s := range []struct {
		n string
		k int
	}{{"resist", 4}, {"duration", 9}, {"monsterMessage", 2}, {"log", 3}, {"nameColor", 4}} {
		o = append(o, s.n+"Calls")
		for k := 1; k <= s.k; k++ {
			o = append(o, fmt.Sprintf("%s_a%d", s.n, k))
		}
	}
	return o
}()

func init() {
	Nodes["00551ba0"] = Node{Addr: "00551ba0", In: sirIn, Out: sirOut,
		F:   statusImmunityAndResistNode,
		Emu: func(in []int64, r Rand) []int64 { return sirRun(in, r, false) }}
}

func statusImmunityAndResistNode(in []int64, r Rand) []int64 { return sirRun(in, r, true) }

func sirRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(sirIn) {
		panic("status immunity and resist: wrong input count")
	}
	if in[sirMonID] >= sirDecoy {
		panic("status immunity and resist: monster id in the fixture's decoy range")
	}
	// The monster record id mirrors the target's id (case "also" on slot matchSlot).
	w := mstBuild(&sirSpec, in)
	w.r = r
	if a, ok := w.address(mstFld{addr: sirMon, size: 4, idx: sirSlot, count: sirSlotsN, stride: sirMonSz}, in); ok {
		w.put(a, 4, uint64(in[sirMonID]))
	}
	out := []int64{int64(sirBody(w, in, real))}
	for _, f := range sirSpec.out {
		out = append(out, w.read(f, in))
	}
	return out
}

func sirCharBase(w *mstWorld, idx uint32) uint32 {
	if idx >= 8 {
		panic("status immunity and resist: character index outside 0..7")
	}
	return w.u32(0x685174) + idx*sirChar
}

// sirTemplateOf is 0x4e5720 on the preloaded cache 0x6840c7 (a miss loads a file).
func sirTemplateOf(w *mstWorld, m uint32) uint32 {
	id := w.u32(m + 8)
	if id >= sirTemplates {
		panic("status immunity and resist: template id past the fixture's cache")
	}
	return w.u32(0x6840c7 + 4*id)
}

// sirFeature is 0x547940(C, code) for code != 0x1c.
func sirFeature(w *mstWorld, c uint32, code int32) bool {
	v := make([]int64, 58)
	v[54], v[55] = int64(int32(w.u32(c+0x69))), int64(int32(w.u32(c+0x71)))
	if v[54] < -1 || v[54] > 14 || v[55] < -1 || v[55] > 15 {
		panic("status immunity and resist: class or race outside the feature tables")
	}
	return pcActionFeature(v, code)
}

func sirWide(w *mstWorld, a uint32) []uint16 {
	var s []uint16
	for ; ; a += 2 {
		c := uint16(w.u8(a)) | uint16(w.u8(a+1))<<8
		if c == 0 {
			return s
		}
		if len(s) == sirWideCap {
			panic("status immunity and resist: wide string past the fixture cap")
		}
		s = append(s, c)
	}
}

// sirNotice is 0x590950(idx, fm, args...).
func sirNotice(w *mstWorld, idx, fm uint32, args ...uint32) {
	var buf []uint16
	f := sirWide(w, fm)
	for i := 0; i < len(f); i++ {
		if f[i] != '%' {
			buf = append(buf, f[i])
			continue
		}
		if i+1 >= len(f) || f[i+1] != 's' || len(args) == 0 {
			panic("status immunity and resist: format conversion other than %s with an argument")
		}
		buf = append(buf, sirWide(w, args[0])...)
		args = args[1:]
		i++
	}
	at := func(i int) uint16 {
		if i < len(buf) {
			return buf[i]
		}
		return 0
	}
	suffix := uint32(sirSpace)
	if at(0) == '\'' || at(0) == ':' {
		suffix = sirQuote
	}
	buf = append(buf, sirWide(w, suffix)...)
	c := sirCharBase(w, idx)
	w.note(sirKLog, 8, sirCharFmt, c+5)
	n := uint32(len(sirWide(w, c+5)))
	if at(0) == '\'' {
		n++
		if at(1) == 's' {
			n++
		}
	}
	w.note(sirKColor, uint32(w.u8(w.u32(0x685178)+sirSlotSz*idx+0xf1)), 0, n, 0xffffffff)
}

func sirBody(w *mstWorld, in []int64, real bool) byte {
	arg := func(n uint32) uint32 { return w.u32(mstStackBase + 4 + 4*n) }
	src, tgt, status, realm, strength := arg(0), arg(1), arg(2), arg(3), arg(4)
	penalty, dur, msg, durMsg, durFlagB := arg(5), arg(6), byte(arg(7)), arg(8), arg(9)
	kind, idx := w.u32(tgt), w.u32(tgt+4)
	var primary uint32
	if kind == 1 {
		primary = w.u32(sirCharBase(w, idx) + 0xb01)
	} else {
		m := w.findMonster(w.u32(tgt + 8))
		t := sirTemplateOf(w, m)
		if w.u8(t+0x1be) != 0 && status == 0x12 {
			return 1
		}
		cat := w.u8(t + 0xcb)
		for row := uint32(sirCatTable); row < sirCatEnd; row += 0x52 {
			if w.u8(row-2) != cat {
				continue
			}
			for k := uint32(0); k < 20; k++ {
				if w.u32(row+4*k) == status {
					return 1
				}
			}
		}
		primary = w.u32(m + 0x107)
	}
	if primary >= 0x12 {
		return 1
	}
	if status == 0x13 {
		if kind == 3 || w.u32(sirCharBase(w, idx)+0xa4d) != 0 {
			return 1
		}
	}
	w.note(sirKResist, tgt, realm, strength, status)
	resisted := byte(w.stubRet(sirKResist))
	if real {
		v := make([]int64, len(srrIn))
		v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7] = int64(kind), int64(idx), in[sirMonID], int64(realm), int64(strength), int64(status), in[sirDifficulty], in[sirCharLevel]
		copy(v[8:14], in[sirCharRes:sirCharRes+6])
		v[14], v[15], v[16], v[17] = in[sirCharPrimary], in[sirGrowth], in[sirReverse], in[sirTplIndex]
		copy(v[18:24], in[sirTplRes:sirTplRes+6])
		v[24] = in[sirTplLevel]
		copy(v[25:31], in[sirMonMod:sirMonMod+6])
		v[31], v[32] = in[sirMonPrimary], in[sirMonClass]
		resisted = byte(statusResistRoll(v, w.r, true)[0])
	}
	if resisted == 0 {
		if kind == 1 && status-2 <= 13 {
			switch target := w.u32(sirJumpTab + 4*uint32(w.u8(sirJumpSel+status-2))); target {
			case 0x551cd6, 0x551d27:
				c := sirCharBase(w, idx)
				if target == 0x551cd6 && sirFeature(w, c, 3) {
					if msg != 0 {
						sirNotice(w, idx, w.u32(w.u32(sirTextTab)+0x600))
					}
					return 1
				}
				if sirFeature(w, c, 0x1e) {
					return 1
				}
			case 0x551d58, 0x551d60:
				if target == 0x551d58 && dur == 0x270f {
					break
				}
				if sirFeature(w, sirCharBase(w, idx), 0xe) {
					if msg != 0 {
						tab := w.u32(sirTextTab)
						id := uint32(w.u8(sirTextIDs+8*status)) | uint32(w.u8(sirTextIDs+8*status+1))<<8
						sirNotice(w, idx, w.u32(tab+0x604), w.u32(tab+4*id))
					}
					return 1
				}
			case 0x551dc1:
			default:
				panic(fmt.Sprintf("status immunity and resist: jump target %#x", target))
			}
		}
		k := uint32(0xffffffff)
		if kind == 3 && w.u32(src) == 1 { // 0x53bea0(src, 0)
			if int32(w.u32(src+4)) == -1 {
				panic("status immunity and resist: source character index -1 (assert 0x53beb1)")
			}
			k = w.u32(src + 4)
		}
		w.note(sirKDur, tgt, status, realm, strength, penalty, dur, k, durFlagB, durMsg)
		applied := byte(w.stubRet(sirKDur))
		if real {
			v := sdrParentInput(in[sirOwn:])
			v[0], v[1], v[2], v[3], v[4] = int64(status), int64(realm), int64(strength), int64(penalty), int64(dur)
			v[5], v[6], v[7], v[8], v[9] = int64(durFlagB), int64(durMsg), int64(kind), int64(int32(idx)), in[sirMonID]
			applied = byte(statusDurationAfterResistNode(v, w.r)[0])
		}
		if applied != 0 {
			return 0
		}
	}
	if msg != 0 && w.u8(sirMsgOn) != 0 {
		tab := w.u32(sirTextTab)
		if kind == 3 {
			h, ok := w.monsterHandle(w.u32(tgt + 8))
			if !ok {
				panic("status immunity and resist: monster id not found (0x4e5550 asserts)")
			}
			w.note(sirKMon, w.monsterRecord(h), w.u32(tab+0x6cc))
		} else {
			sirNotice(w, idx, w.u32(tab+0x6cc))
		}
	}
	return 1
}
