package calc

import "fmt"

// 0x545e50 attack_intercept(attacker descriptor, target descriptor): collects the
// party members and monsters guarding the target, rolls each one's interception
// chance (candidate score minus attacker score plus 50, clamped to 5..95), picks
// one success at random, retargets the attack to it and spends its intercept and
// stamina. World layout: re/emu/cases/00545e50.json.
//
// Address stubs (case "stubs"): the D4 eligibility callees 0x524a10, 0x4ed2d0,
// 0x545c20 and 0x545cf0 (their returns are read, so Emu replays the stub returns
// and F runs their Go nodes); the D4 effect callees 0x52af50 and 0x523c00 (their
// returns are ignored and their writes are not read again here); the 排除
// presentation helpers 0x590b40, 0x58aad0, 0x58b410, 0x562a50 and 0x5182e0.
// The call count and every call's arguments are outputs.
//
// Fixture caps (panic past them): party index 0..7, active monster list up to 4
// records, template ids 0..1, weapon item ids 0..3, speaker index 0..7, wide
// strings 0x1000 units, message formats with %s only, at most 8 (4 for 0x545cf0,
// 1 for 0x52af50 / 0x523c00 / 0x5182e0) recorded calls per stub.

const (
	aiArena  = 0x20000000
	aiTgt    = aiArena + 0x0
	aiAtk    = aiArena + 0x40
	aiScr    = aiArena + 0x100 // stub returns, counts and records
	aiListA  = aiArena + 0x1000
	aiEntA   = aiArena + 0x1020
	aiListB  = aiArena + 0x1040
	aiEntB   = aiArena + 0x1060
	aiGl1    = aiArena + 0x1080
	aiGent1  = aiArena + 0x10a0
	aiGl2    = aiArena + 0x10c0
	aiGent2  = aiArena + 0x10e0
	aiGrp    = aiArena + 0x1100
	aiVis    = aiArena + 0x1200
	aiActor  = aiArena + 0x1400
	aiTxt    = aiArena + 0x1800 // fixture text pointer table
	aiStr    = aiArena + 0x2800
	aiItems  = aiArena + 0x3000
	aiTpl    = aiArena + 0x4000
	aiMcd    = aiArena + 0x5000
	aiMon    = aiArena + 0x6000
	aiG      = aiArena + 0x8000
	aiP      = aiArena + 0x9000
	aiChars  = aiArena + 0x10000
	aiFonly  = aiArena + 0x20000 // F-only callee worlds (no reader in Emu)
	aiCharSz = 0x1862
	aiSlotSz = 0x106
	aiCombSz = 0xd4
	aiMonSz  = 0x400
	aiTplSz  = 0x297
	aiItemSz = 0x10d
	aiMcdSz  = 0x200
	aiItemsN = 4
	aiWideN  = 0x1000

	aiMsgOn    = 0x68510c // byte: failed intercepts are announced
	aiTextTab  = 0x68c09c // text pointer table: +0x864 failed, +0x868 succeeded
	aiNameBuf  = 0x6875c3 // 0x4e5150 name buffer for template name kind 0x18d
	aiSpeaker  = 0x6875ef // party index whose name 0x4e5150 formats
	aiAlFormat = 0x614b44
	aiLogFmt   = 0x614b20
	aiRaceTab  = 0x61d284 // 18 race bytes that halve a monster's fatigue penalty
	aiQuote    = 0x689b34
	aiSpace    = 0x60cc74
	aiCharFmt  = 0x64bf5c
)

// aiStubs mirrors the case's address stubs (emu.py: record n, ret {at, n}, count).
var aiStubs = map[uint32]*mcaStub{}

var aiIn, aiOut []string
var aiSpec mstSpec
var aiAlso = map[int]uint32{} // input index -> extra address written with the same value
var aiOwn int                 // first F-only input

func init() {
	aiLayout()
	Nodes["00545e50"] = Node{Addr: "00545e50", In: aiIn, Out: aiOut, F: attackInterceptNode,
		Emu: func(in []int64, r Rand) []int64 { return aiRun(in, r, false) }}
}

func attackInterceptNode(in []int64, r Rand) []int64 { return aiRun(in, r, true) }

func aiLayout() {
	fld := func(name string, at uint32, size uint8, signed bool) {
		aiIn = append(aiIn, name)
		aiSpec.in = append(aiSpec.in, mstFld{addr: at, size: size, signed: signed, idx: -1})
	}
	set := func(at uint32, size uint8, v uint32) { aiSpec.set = append(aiSpec.set, mstSet{at, size, v}) }
	out := func(name string, at uint32, size uint8) {
		aiOut = append(aiOut, name)
		aiSpec.out = append(aiSpec.out, mstFld{addr: at, size: size, idx: -1})
	}
	fld("attackerKind", aiAtk, 4, false)
	fld("attackerIndex", aiAtk+4, 4, true)
	fld("attackerMonsterID", aiAtk+8, 4, false)
	aiAlso[len(aiIn)-1] = aiMon + 5*aiMonSz
	for k := uint32(0); k < 8; k++ {
		fld(fmt.Sprintf("target_%d", k), aiTgt+4*k, 4, false)
		if k == 2 {
			aiAlso[len(aiIn)-1] = aiMon + 4*aiMonSz
		}
	}
	fld("monsterCount", aiListA+8, 4, false)
	for c := uint32(0); c < 8; c++ {
		ch, pp, gg := aiChars+aiCharSz*c, aiP+aiSlotSz*c, aiG+aiCombSz*c
		n := func(s string) string { return fmt.Sprintf("c%d.%s", c, s) }
		fld(n("slotActive"), pp, 1, false)
		fld(n("mainStatus"), ch+0xb01, 4, false)
		fld(n("currentHP"), ch+0xb11, 4, false)
		fld(n("interceptOn"), ch+0x114d, 1, false)
		fld(n("interceptMode"), ch+0x114e, 4, false)
		fld(n("interceptBonusA"), ch+0x115a, 4, false)
		fld(n("interceptBudget"), ch+0x115e, 4, false)
		fld(n("interceptBonusB"), ch+0x116a, 4, false)
		fld(n("speed"), ch+0x14d, 4, false)
		fld(n("fatigueLevel"), ch+0x169e, 4, false)
		fld(n("interceptModifier"), ch+0x1771, 1, true)
		fld(n("weaponItem"), ch+0xfa5, 4, false)
		fld(n("nameColor"), pp+0xf1, 1, false)
		fld(n("posture"), pp+1, 4, false)
		fld(n("postureA"), pp+5, 4, true)
		fld(n("postureB"), pp+9, 4, true)
		for k := uint32(0); k < 8; k++ {
			fld(n(fmt.Sprintf("guard_%d", k)), pp+0x1d+4*k, 4, false)
		}
		for k := uint32(0); k < 8; k++ {
			fld(n(fmt.Sprintf("pendingGuard_%d", k)), pp+0x4d+4*k, 4, false)
		}
		fld(n("guardSet"), gg+0x4c, 1, false)
		fld(n("interceptsUsed"), gg+0xb8, 4, false)
	}
	for k := uint32(0); k < aiItemsN; k++ {
		fld(fmt.Sprintf("item%d.weight", k), aiItems+aiItemSz*k+0x8a, 2, false)
	}
	for s := uint32(0); s < 6; s++ {
		m := aiMon + aiMonSz*s
		n := func(x string) string { return fmt.Sprintf("m%d.%s", s, x) }
		if s < 4 {
			fld(n("id"), m, 4, false)
		}
		fld(n("group"), m+4, 4, false)
		fld(n("template"), m+8, 4, false)
		fld(n("active"), m+0x14, 1, false)
		fld(n("visible"), m+0x15, 1, false)
		fld(n("hp"), m+0x2b, 4, false)
		fld(n("staminaMax"), m+0x2f, 4, false)
		fld(n("stamina"), m+0x33, 4, false)
		fld(n("state11Duration"), m+0x9b, 4, false)
		fld(n("mainState"), m+0x107, 4, false)
		fld(n("interceptModifier"), m+0x1dc, 1, true)
		fld(n("fatigueLevel"), m+0x242, 4, false)
		fld(n("speed"), m+0x24a, 1, false)
		fld(n("action"), m+0x2e1, 4, false)
		for k := uint32(0); k < 8; k++ {
			fld(n(fmt.Sprintf("guard_%d", k)), m+0x2ba+4*k, 4, false)
		}
		fld(n("interceptsUsed"), aiMcd+aiMcdSz*s+0x147, 4, false)
	}
	for t := uint32(0); t < 2; t++ {
		b := aiTpl + aiTplSz*t
		n := func(x string) string { return fmt.Sprintf("t%d.%s", t, x) }
		fld(n("flags"), b+0xd0, 1, false)
		fld(n("interceptLimit"), b+0xe5, 1, false)
		fld(n("interceptOn"), b+0xe7, 1, false)
		fld(n("race"), b+0xe8, 1, false)
		fld(n("interceptBonus"), b+0xeb, 1, false)
		fld(n("nameKind"), b+0x187, 2, false)
	}
	for g := uint32(0); g < 2; g++ {
		fld(fmt.Sprintf("group%d.named", g), aiGrp+0x40*g+0x2c, 1, false)
	}
	fld("messagesEnabled", aiMsgOn, 1, false)
	fld("attackBonusA", aiG+0x9a4, 1, false)
	fld("attackBonusB", aiG+0x9a5, 1, false)
	fld("speakerIndex", aiSpeaker, 4, false)
	fld("actorStateA", aiActor+0xa4, 1, true)
	fld("actorStateB", aiActor+0xa7, 1, false)
	rets := []struct {
		name     string
		off, cnt uint32
	}{{"r524a10", 0, 8}, {"r4ed2d0", 0x20, 8}, {"r545c20", 0x40, 8}, {"r545cf0", 0x60, 4}}
	for _, r := range rets {
		for k := uint32(0); k < r.cnt; k++ {
			fld(fmt.Sprintf("%s_%d", r.name, k), aiScr+r.off+4*k, 4, false)
		}
	}
	aiOwn = len(aiIn)
	// F-only: the parked worlds of the D4 callees, by their input position.
	for _, c := range []struct {
		a string
		n int
	}{{"4ed2d0", 279}, {"545c20", 108}, {"545cf0", 48}} {
		for i := 0; i < c.n; i++ { // case: 8-byte cells from aiFonly, never read by the world
			aiIn = append(aiIn, fmt.Sprintf("f%s_%d", c.a, i))
		}
	}

	// Stubs: counts and records laid out from scr+0x100, outputs in that order.
	out("result", 0, 1)
	stubs := []struct {
		a    uint32
		ret  int // index into rets, -1 = returns 0
		args []uint32
		n    uint32
	}{{0x524a10, 0, []uint32{1}, 8}, {0x4ed2d0, 1, []uint32{1, 2, 3}, 8}, {0x545c20, 2, []uint32{1, 2}, 8},
		{0x545cf0, 3, []uint32{1, 2}, 4}, {0x52af50, -1, []uint32{1, 2, 3, 4}, 1}, {0x523c00, -1, []uint32{1, 2, 3, 4, 6}, 1},
		{0x590b40, -1, []uint32{1, 2}, 4}, {0x58aad0, -1, []uint32{1, 2, 3}, 8}, {0x58b410, -1, []uint32{1, 2, 3, 4}, 8},
		{0x562a50, -1, []uint32{1}, 8}, {0x5182e0, -1, []uint32{1, 2, 3}, 1}}
	p := uint32(aiScr + 0x100)
	for _, s := range stubs {
		st := &mcaStub{cnt: p}
		if s.ret >= 0 {
			st.ret, st.retN = aiScr+rets[s.ret].off, s.n
		}
		out(fmt.Sprintf("n%x", s.a), p, 4)
		p += 4
		for _, a := range s.args {
			st.rec = append(st.rec, [4]uint32{a, 4, p, s.n})
			for k := uint32(0); k < s.n; k++ {
				out(fmt.Sprintf("a%x_%d_%d", s.a, a, k), p+4*k, 4)
			}
			p += 4 * s.n
		}
		aiStubs[s.a] = st
	}
	for k := uint32(0); k < 8; k++ {
		out(fmt.Sprintf("target_%d", k), aiTgt+4*k, 4)
	}
	for c := uint32(0); c < 8; c++ {
		pp, gg := aiP+aiSlotSz*c, aiG+aiCombSz*c
		out(fmt.Sprintf("c%d.interceptsUsed", c), gg+0xb8, 4)
		out(fmt.Sprintf("c%d.guardSet", c), gg+0x4c, 1)
		out(fmt.Sprintf("c%d.posture", c), pp+1, 4)
		out(fmt.Sprintf("c%d.postureA", c), pp+5, 4)
		out(fmt.Sprintf("c%d.postureB", c), pp+9, 4)
		for k := uint32(0); k < 8; k++ {
			out(fmt.Sprintf("c%d.guard_%d", c, k), pp+0x1d+4*k, 4)
		}
	}
	out("attackBonusA", aiG+0x9a4, 1)
	out("attackBonusB", aiG+0x9a5, 1)
	for s := uint32(0); s < 6; s++ {
		m := aiMon + aiMonSz*s
		out(fmt.Sprintf("m%d.stamina", s), m+0x33, 4)
		out(fmt.Sprintf("m%d.fatigueLevel", s), m+0x242, 4)
		out(fmt.Sprintf("m%d.interceptsUsed", s), aiMcd+aiMcdSz*s+0x147, 4)
	}
	for k := uint32(0); k < 24; k++ {
		out(fmt.Sprintf("message_%d", k), aiG+0x6b8+2*k, 2)
	}
	for k := uint32(0); k < 16; k++ {
		out(fmt.Sprintf("nameBuffer_%d", k), aiNameBuf+2*k, 2)
	}

	// Setup: world wiring and fixture text.
	for _, g := range [][2]uint32{{0x685174, aiChars}, {0x685178, aiP}, {0x6836a8, aiG}, {0x68516c, aiItems},
		{0x683fad, aiListA}, {0x683fb9, aiListB}, {0x683fb1, aiGl1}, {0x683fbd, aiGl2}, {aiTextTab, aiTxt}} {
		set(g[0], 4, g[1])
	}
	for t := uint32(0); t < 2; t++ {
		set(0x6840c7+4*t, 4, aiTpl+aiTplSz*t)
	}
	set(aiListA, 4, aiEntA)
	set(aiListB, 4, aiEntB)
	set(aiListB+8, 4, 2)
	for s := uint32(0); s < 6; s++ {
		e := aiEntA + 4*s
		if s >= 4 {
			e = aiEntB + 4*(s-4)
		}
		set(e, 4, aiMon+aiMonSz*s)
		set(aiMon+aiMonSz*s+0xc, 4, aiVis)
		set(aiMon+aiMonSz*s+0x10, 4, aiMcd+aiMcdSz*s)
	}
	for g, l := range [][2]uint32{{aiGl1, aiGent1}, {aiGl2, aiGent2}} {
		set(l[0], 4, l[1])
		set(l[0]+8, 4, 1)
		set(l[1], 4, aiGrp+0x40*uint32(g))
		set(aiGrp+0x40*uint32(g), 4, uint32(g)+1)
	}
	set(aiVis+0x1d8, 4, aiActor)
	wide := func(at uint32, s string) {
		for i, c := range s + "\x00" {
			set(at+2*uint32(i), 2, uint32(c))
		}
	}
	set(aiTxt+0x864, 4, aiStr)
	wide(aiStr, "%s'")
	set(aiTxt+0x868, 4, aiStr+0x40)
	wide(aiStr+0x40, "'s %s")
	for c, s := range []string{"A", "Bo", "Cid", "Dana", "Ed", "Fay", "Gus", "Hal"} {
		wide(aiChars+aiCharSz*uint32(c)+5, s)
	}
	for t, s := range [][2]string{{"Orcs", "Orc"}, {"Imps", "Imp"}} {
		wide(aiTpl+aiTplSz*uint32(t), s[0])
		wide(aiTpl+aiTplSz*uint32(t)+0x60, s[1])
	}
}

type aiCand struct{ kind, idx, id uint32 }

type aiWorld struct {
	*mcaWorld
	in []int64
}

func aiRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(aiIn) {
		panic("attack intercept: wrong input count")
	}
	w := &aiWorld{&mcaWorld{mstWorld: mstBuild(&aiSpec, in), real: real}, in}
	w.r = r
	for i, a := range aiAlso {
		f := aiSpec.in[i]
		w.put(a, int(f.size), uint64(in[i]))
	}
	res := w.body()
	out := make([]int64, len(aiSpec.out))
	out[0] = int64(res)
	for i, f := range aiSpec.out[1:] {
		out[i+1] = w.read(f, in)
	}
	return out
}

// stub records call k of address stub a and returns its replayed return value.
func (w *aiWorld) stub(a uint32, args ...uint32) uint32 {
	s := aiStubs[a]
	k := w.u32(s.cnt)
	for _, r := range s.rec {
		if k >= r[3] {
			panic(fmt.Sprintf("attack intercept: stub %#x called more than %d times (fixture record cap)", a, r[3]))
		}
		w.put(r[2]+k*r[1], int(r[1]), uint64(args[r[0]-1]))
	}
	w.put(s.cnt, 4, uint64(k+1))
	if s.retN == 0 {
		return 0
	}
	if k >= s.retN {
		panic(fmt.Sprintf("attack intercept: stub %#x return past its %d values", a, s.retN))
	}
	return w.u32(s.ret + 4*k)
}

// callee runs a D4 eligibility callee: F calls its Go node on its F-only world
// with this call's arguments, Emu replays the stub return.
func (w *aiWorld) callee(a uint32, real func() bool, args ...uint32) bool {
	ret := w.stub(a, args...)
	if w.real {
		return real()
	}
	return byte(ret) != 0
}

func (w *aiWorld) parked(skip, n int) []int64 {
	v := make([]int64, n)
	copy(v, w.in[aiOwn+skip:aiOwn+skip+n])
	return v
}

func (w *aiWorld) charBase(i uint32) uint32 {
	if i >= 8 {
		panic("attack intercept: party index past 7")
	}
	return w.u32(0x685174) + i*aiCharSz
}

func (w *aiWorld) wide(a uint32) []uint16 {
	var s []uint16
	for ; ; a += 2 {
		c := uint16(w.u16(a))
		if c == 0 {
			return s
		}
		if len(s) == aiWideN {
			panic("attack intercept: wide string past the fixture cap")
		}
		s = append(s, c)
	}
}

func (w *aiWorld) putWide(a uint32, s []uint16) {
	for i, c := range append(s, 0) {
		w.put(a+2*uint32(i), 2, uint64(c))
	}
}

// swprintf with %s conversions only (fixture formats).
func (w *aiWorld) swprintf(dst, fm uint32, args ...[]uint16) {
	var buf []uint16
	f := w.wide(fm)
	for i := 0; i < len(f); i++ {
		if f[i] != '%' {
			buf = append(buf, f[i])
			continue
		}
		if i+1 >= len(f) || f[i+1] != 's' || len(args) == 0 {
			panic("attack intercept: format conversion other than %s with an argument")
		}
		buf = append(buf, args[0]...)
		args = args[1:]
		i++
	}
	w.putWide(dst, buf)
}

// notice is 0x590950(idx, text) with no format arguments.
func (w *aiWorld) notice(idx, text uint32) {
	var buf []uint16
	for _, c := range w.wide(text) {
		if c == '%' {
			panic("attack intercept: format conversion other than %s with an argument")
		}
		buf = append(buf, c)
	}
	at := func(i int) uint16 {
		if i < len(buf) {
			return buf[i]
		}
		return 0
	}
	c := w.charBase(idx)
	w.stub(0x58aad0, 8, aiCharFmt, c+5)
	n := uint32(len(w.wide(c + 5)))
	if at(0) == '\'' {
		n++
		if at(1) == 's' {
			n++
		}
	}
	w.stub(0x58b410, uint32(w.u8(w.u32(0x685178)+aiSlotSz*idx+0xf1)), 0, n, 0xffffffff)
}

// name is 0x4e5150(m, 0, 0).
func (w *aiWorld) name(m uint32) uint32 {
	t := w.template(m)
	if w.u16(t+0x187) == 0x18d {
		w.swprintf(aiNameBuf, aiAlFormat, w.wide(w.charBase(w.u32(aiSpeaker))+5))
		return aiNameBuf
	}
	if w.u32(m+4) == 0 {
		if !w.act15(w.u32(m + 0xc)) {
			w.stub(0x5182e0, 1, aiLogFmt, w.u32(m))
		}
		return t + 0x60
	}
	if w.u8(w.group(w.u32(m+4))+0x2c) == 0 {
		return t + 0x60
	}
	return t
}

// fatiguePenalty is 0x52a7d0.
func aiFatiguePenalty(x uint32) uint32 {
	switch x {
	case 0:
		return 0
	case 1:
		return 5
	case 2:
		return 10
	case 3:
		return 20
	case 4:
		return 40
	}
	return x
}

func (w *aiWorld) charScore(c uint32) uint32 {
	var bonus uint32
	if w.u8(c+0x114d) != 0 {
		bonus = 5*w.u32(c+0x116a) + w.u32(c+0x115a)
	}
	bonus += 5 * uint32(int32(int8(w.u8(c+0x1771))))
	pen := aiFatiguePenalty(w.u32(c + 0x169e))
	if w.u32(c+0x114e) == 7 {
		pen >>= 1
	}
	return w.u32(c+0x14d) - pen + bonus
}

func (w *aiWorld) monScore(m uint32) uint32 {
	t := w.template(m)
	var bonus uint32
	if w.u8(t+0xe7) != 0 {
		bonus = uint32(w.u8(t + 0xeb))
	}
	bonus += 5 * uint32(int32(int8(w.u8(m+0x1dc))))
	pen := aiFatiguePenalty(w.u32(m + 0x242))
	for k := uint32(0); k < 18; k++ {
		if w.u8(aiRaceTab+k) == w.u8(t+0xe8) {
			pen >>= 1
			break
		}
	}
	return uint32(w.u8(m+0x24a)) - pen + bonus
}

// initTarget is 0x536170: a cleared descriptor (kind 0, indexes -1).
func (w *aiWorld) initTarget(d uint32) {
	for k := uint32(0); k < 8; k++ {
		w.put(d+4*k, 4, 0)
	}
	for k := uint32(1); k < 4; k++ {
		w.put(d+4*k, 4, 0xffffffff)
	}
}

// staminaSpend is 0x52c070(m, amount, 0).
func (w *aiWorld) staminaSpend(m, amount uint32) {
	if w.u32(m+0x2b) == 0 {
		return
	}
	w.template(m)
	sp := w.u32(m + 0x33)
	if amount >= sp {
		sp = 0
	} else {
		sp -= amount
	}
	w.put(m+0x33, 4, uint64(sp))
	max := w.u32(m + 0x2f)
	if max == 0 {
		panic("attack intercept: monster maximum stamina 0 (0x52c0b8 divide fault)")
	}
	w.put(m+0x242, 4, uint64(staminaFatigueLevel(sp, max)))
	if sp == 0 && w.u32(m+0x9b) < 0x270f {
		// 0x523c00(id, 0x11, 0x270f, 0, &source, 1); arg 5 is a stack local.
		w.stub(0x523c00, w.u32(m), 0x11, 0x270f, 0, 0, 1)
	}
}

func (w *aiWorld) body() byte {
	tgt := uint32(aiTgt)
	g0 := w.u32(0x6836a8)
	same := func(a uint32) bool {
		for k := uint32(0); k < 8; k++ {
			if w.u32(a+4*k) != w.u32(tgt+4*k) {
				return false
			}
		}
		return true
	}
	var cands []aiCand
	for i := uint32(0); i < 8; i++ {
		c, p, g := w.charBase(i), w.u32(0x685178)+i*aiSlotSz, g0+i*aiCombSz
		if !w.callee(0x524a10, func() bool {
			return characterActionEligibilityNode([]int64{int64(i), int64(w.u8(p)), int64(w.u32(c + 0xb11)), int64(w.u32(c + 0xb01))}, w.r)[0]&0xff != 0
		}, i) {
			continue
		}
		if w.u32(c+0xb01) >= 0xc || w.u8(c+0x114d) == 0 {
			continue
		}
		if !w.callee(0x4ed2d0, func() bool {
			v := w.parked(0, 279)
			v[0], v[261], v[262] = int64(i), 5, 0
			return characterPostureEligibilityNode(v, w.r)[0]&0xff != 0
		}, i, 5, 0) {
			continue
		}
		if w.u32(g+0xb8) >= w.u32(c+0x115e) {
			continue
		}
		guard := p + 0x4d
		if w.u8(g+0x4c) != 0 {
			guard = p + 0x1d
		}
		if !same(guard) {
			continue
		}
		if !w.callee(0x545c20, func() bool {
			v := w.parked(279, 108)
			v[0], v[8] = int64(w.u32(tgt)), int64(i)
			if w.u32(tgt) == 1 {
				v[1] = int64(w.u32(tgt + 4))
			}
			return interceptTargetAvailableNode(v, w.r)[0]&0xff != 0
		}, i, tgt) {
			continue
		}
		cands = append(cands, aiCand{1, i, 0xffffffff})
		if w.u8(g+0x4c) == 0 { // 0x4ed390(i, 5)
			w.put(p+1, 4, 5)
			w.put(p+5, 4, 0xffffffff)
			w.put(p+9, 4, 0xffffffff)
			for k := uint32(0); k < 8; k++ {
				w.put(p+0x1d+4*k, 4, uint64(w.u32(p+0x4d+4*k)))
			}
			w.put(g+0x4c, 1, 1)
			w.stub(0x562a50, 1<<i|0x100000)
		}
	}
	for j := uint32(0); j < w.listCount(w.u32(0x683fad)); j++ {
		if j >= 4 {
			panic("attack intercept: active monster list past the fixture's 4 records")
		}
		m := w.monsterRecord(j)
		t := w.template(m)
		if w.u8(m+0x14) == 0 || w.u8(m+0x15) == 0 || w.u32(m+0x2b) == 0 || w.u32(m+0x107) >= 0xc {
			continue
		}
		if w.u8(t+0xd0)&4 == 0 || w.u8(t+0xe7) == 0 || w.u32(m+0x2e1) != 8 {
			continue
		}
		if w.u32(w.u32(m+0x10)+0x147) >= uint32(w.u8(t+0xe5)) || !same(m+0x2ba) {
			continue
		}
		if !w.callee(0x545cf0, func() bool {
			v := w.parked(279+108, 48)
			v[0] = int64(w.u32(tgt))
			if w.u32(tgt) == 1 {
				v[1] = int64(w.u32(tgt + 4))
			}
			return monsterInterceptAvailableNode(v, w.r)[0]&0xff != 0
		}, m, tgt) {
			continue
		}
		cands = append(cands, aiCand{2, 0xffffffff, w.u32(m)})
	}
	if len(cands) == 0 {
		return 0
	}

	var local []uint16 // the target's name
	switch w.u32(tgt) {
	case 1:
		local = w.wide(w.charBase(w.u32(tgt+4)) + 5)
	case 3:
		local = w.wide(w.name(w.findMonster(w.u32(tgt + 8))))
	default:
		panic("attack intercept: target kind other than 1 or 3 (0x54622e asserts)")
	}

	var succ []aiCand
	for i := 0; i < len(cands); i++ {
		if i >= len(cands) {
			panic("attack intercept: candidate index past the list (0x546278 unreachable)")
		}
		c := cands[i]
		var cs, as uint32
		switch c.kind {
		case 1:
			if w.u8(w.charBase(c.idx)+0x114d) == 0 {
				panic("attack intercept: candidate without intercept stance (0x5462be unreachable)")
			}
			cs = w.charScore(w.charBase(c.idx))
		case 2:
			cs = w.monScore(w.findMonster(c.id))
		default:
			panic("attack intercept: candidate kind other than 1 or 2 (0x546389 asserts)")
		}
		switch w.u32(aiAtk) {
		case 1:
			as = w.charScore(w.charBase(w.u32(aiAtk + 4)))
		case 2:
			as = w.monScore(w.findMonster(w.u32(aiAtk + 8)))
		default:
			panic("attack intercept: attacker kind other than 1 or 2 (0x5464c2 asserts)")
		}
		if w.u8(g0+0x9a4) != 0 {
			as += 10
		}
		if w.u8(g0+0x9a5) != 0 {
			as += 10
		}
		chance := int32(cs - as + 50)
		if chance > 0x5f {
			chance = 0x5f
		} else if chance < 5 {
			chance = 5
		}
		if int32(w.r.R(100)) >= chance {
			continue
		}
		if len(succ) >= 8+4 {
			panic("list expand: 546576 (success list past 8 party + 4 monster candidates)")
		}
		succ = append(succ, c)
	}
	text := g0 + 0x6b8
	if len(succ) == 0 {
		if w.u8(aiMsgOn) != 0 {
			for i := 0; i < len(cands); i++ {
				if i >= len(cands) {
					panic("attack intercept: failure-message index past the list (0x546633 unreachable)")
				}
				c := cands[i]
				w.swprintf(text, w.u32(w.u32(aiTextTab)+0x864), local)
				if c.kind == 1 {
					w.notice(c.idx, text)
				} else {
					w.stub(0x590b40, w.findMonster(c.id), text)
				}
			}
		}
		return 0
	}
	if len(succ) == 0 {
		panic("attack intercept: success branch with zero successes (0x5466ee unreachable)")
	}
	k := w.r.R(len(succ))
	if k < 0 || k >= len(succ) {
		panic("attack intercept: success pick past the list (0x546724 unreachable)")
	}
	s := succ[k]
	w.swprintf(text, w.u32(w.u32(aiTextTab)+0x868), local)
	switch s.kind {
	case 1:
		w.notice(s.idx, text)
		w.initTarget(tgt)
		w.put(tgt, 4, 1)
		w.put(tgt+4, 4, uint64(s.idx))
		g := g0 + s.idx*aiCombSz
		w.put(g+0xb8, 4, uint64(w.u32(g+0xb8)+1))
		item := w.u32(w.charBase(s.idx) + 0xfa5)
		if item >= aiItemsN {
			panic("attack intercept: weapon item id past the fixture's 4 item records")
		}
		x := w.u16(w.u32(0x68516c)+aiItemSz*item+0x8a) / 40
		amount := uint32(w.r.R(int(x+1))) + x + 2
		w.stub(0x52af50, s.idx, amount, 1, 0)
	default:
		m := w.findMonster(s.id)
		w.stub(0x590b40, m, text)
		w.initTarget(tgt)
		w.put(tgt, 4, 3)
		w.put(tgt+8, 4, uint64(s.id))
		d := w.u32(m + 0x10)
		w.put(d+0x147, 4, uint64(w.u32(d+0x147)+1))
		w.staminaSpend(m, uint32(w.r.R(3))+2)
	}
	w.put(g0+0x9a4, 1, 0)
	w.put(g0+0x9a5, 1, 0)
	return 1
}
