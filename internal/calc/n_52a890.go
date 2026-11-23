package calc

import "fmt"

// character_damage_apply 0x52a890(who, dmg, a3, a4, a5, kill, a7) (build/wiz8.asm:404778..):
// party member `who` (record *0x685174 + 0x1862*who, presence byte *0x685178 +
// 0x106*who) takes `dmg`. a3, a4, a5, a7 are byte flags (only the low byte counts);
// `kill` is a kill-list object or null.
//
// A member whose presence byte is 0 hits srAssertFail (block 0x52a8ca): calc panics.
// +0xb11 == 0 returns 0. Byte 0x687500 set: 0x590950 prints a notice, return 0.
// Absorb pool (+0xa83 u16, active when +0xa85 != 0): a pool >= dmg takes the whole
// blow (notice, pool -= dmg, kill[4]++ when kill is set, return 0); a smaller pool
// is spent (notice), dmg falls by it, 0x523a80(who, 2) runs and a second notice
// follows. Then 0x52af50(who, floor(2*dmg/3), 0, kill) (32-bit 2*dmg), and with
// kill: kill[0] += dmg, kill[4]++. Flag a4 prints a notice (a7: 0x590950; else a5:
// 0x58aad0(9, text, dmg); else 0x590950 with the a3-selected text). dmg >= +0xb11 is
// lethal: a member with feature 2 (0x547940) and +0xa45 < 7 is saved by 0x547a50(who)
// (portrait update 0x59ac40, return the old +0xb11); otherwise dmg := +0xb11.
// +0xb11 -= dmg; 0x59ac40(who, dmg) updates the portrait record (0x6836b8 + 0x118*who);
// a survivor with byte 0x683fc5 clear runs the voice cue 0x52f2c0. At +0xb11 == 0
// with a kill list a 0x6c-byte record {1, who, 0...} is allocated and appended to
// the list (count +0x5c, capacity +0x60, data +0x64; growth reallocates to n+1
// entries; both allocators may fail, then nothing is appended) and 0x522fe0(who,
// 0x12, 0x270f, 0, 0, kill == null) runs in every death. Last, with state slot
// +0xa3d > 0 and a3 clear, R(100) < u32(+0x161)/2 + 0x32 calls
// 0x523330(who, 0xf, 1).
//
// Callees. D4 formula, address-stubbed in Emu (their returns are ignored by the
// caller; proof: ret 0x5a5a5a5a in a scratch copy of the case gave diff=0): 0x52af50,
// 0x523a80, 0x522fe0, 0x523330, 0x547a50. Excluded, address-stubbed: 0x58aad0,
// 0x52e690. Run natively: 0x590950 (its text-box helper 0x58b410 dereferences the
// battle object 0x68edcc in mode 7: calc panics with mode 7 and no battle object),
// 0x59ac40, 0x562a50, 0x406bf0, 0x52f2c0 with 0x517ec0 (member index) and 0x52dd90
// (bit `member` of byte table[idx], table at *(0x683fd7 + 0x68)). 0x547940 is modelled
// by ccaS.feat. 0x59ac40 calls 0x561a20 (excluded, address-stubbed, its return and its
// writes into the battle object are ignored) when 0x6850ce != 0 and the battle byte
// +0x109+who is clear; it then sets portrait +0xcf = 1 and +0xa3 = -1.
//
// Emu replays the stubs: the stubbed callees change nothing and the case outputs are
// call counts and arguments. F calls the real group functions on a callee world:
// characterFatigueApplyNode for 0x52af50, ccaApply (0x522fe0) and chdRun (0x547a50);
// +0xa45 (slot 0x11), +0xa3d (slot 0xf), +0xb11, +0xb01 flow forward. F-only inputs:
// fa* (fatigue cells of the member, gate cells of 0x52e690), fab* (the +0xb11 and
// +0x161 that 0x523a80 leaves behind: its Go node outputs neither, and 0x523a80 is
// not run in F), ft* (the W cells of ccaApply, as in n_5029e0.go). 0x50e650's rebuild
// of +0x161 inside 0x522fe0 is not modelled in F.
//
// Fixture caps (calc panics past them): who 0..7; kill pointer 0 or the fixture; list
// count 0..15 (data fixtures hold 16 entries); allocator returns 0 or the fixtures;
// 0x52e690 returns 0 or the two voice fixtures; class -1..14 and race -1..15; battle
// mode 7 with a null battle object, 0x6850ce set with a null battle object (0x59ad04 faults); zero +0xb0d at 0x52f2c0 (divide fault).
const (
	cdmArena   = 0x20000000
	cdmChars   = cdmArena
	cdmStrs    = cdmArena + 77824
	cdmKl      = cdmArena + 0x28000
	cdmKlData  = cdmArena + 0x28100
	cdmNewBuf  = cdmArena + 0x28200
	cdmNewObj  = cdmArena + 0x28300
	cdmVoice1  = cdmArena + 0x28600
	cdmVoice2  = cdmArena + 0x28700
	cdmMsgBuf  = 0x64bf5c
	cdmTick    = 4660 // GetTickCount of the case
	cdmMembers = 8
	cdmListMax = 16
	cdmNewObjW = 27
)

var cdmPortF = [...]struct {
	name      string
	off, size int
	in        bool
}{{"9c", 0x9c, 1, true}, {"9d", 0x9d, 1, true}, {"9f", 0x9f, 4, true}, {"a3", 0xa3, 4, false}, {"a7", 0xa7, 4, false},
	{"ab", 0xab, 1, false}, {"bd", 0xbd, 1, true}, {"ca", 0xca, 4, false}, {"cf", 0xcf, 1, false}, {"d0", 0xd0, 1, true}}

// cdmPortInit is the case's setup value of a portrait field the node does not read.
func cdmPortInit(k, f int) int64 {
	return int64(0x21+k*3+cdmPortF[f].off) & (1<<(8*uint(cdmPortF[f].size)) - 1)
}

func cdmNames() (in, out []string) {
	a := func(l *[]string, f string, x ...any) { *l = append(*l, fmt.Sprintf(f, x...)) }
	in = append(in, "who", "dmg", "a3", "a4", "a5", "kill", "a7", "pres", "hp", "maxHp", "b01", "a85", "a83", "a3d", "a45",
		"sen", "cls", "race", "g7500", "g3fc5", "mode", "batPtr", "batF4", "g6850ce", "bt109", "tick", "vc0", "vc1", "vc2", "vb3", "vb2", "vb1c",
		"mallocRet", "newRet", "kl0", "kl4", "kl4c", "kl5c", "kl60", "kl64")
	for i := 0; i < cdmListMax; i++ {
		a(&in, "kd%d", i)
	}
	for i := 0; i < cdmListMax; i++ {
		a(&in, "nb%d", i)
	}
	in = append(in, "r52e0", "r52e1")
	for k := 0; k < cdmMembers; k++ {
		for _, f := range cdmPortF {
			if f.in {
				a(&in, "p%s_%d", f.name, k)
			}
		}
	}
	in = append(in, "fa_stamina", "fa_staMax", "fa_enc", "fa_a15", "fa_aa9", "fa_level", "fa_flag", "fa_g8c8", "fa_g914", "fa_g598",
		"fab_hp", "fab_sen")
	for k := 0; k < ccaWLen; k++ {
		a(&in, "ft%d", k)
	}
	out = append(out, "ret", "hp", "a83", "kl0", "kl4", "kl4c", "kl5c", "kl60", "kl64")
	for i := 0; i < cdmListMax; i++ {
		a(&out, "kd%d", i)
	}
	for i := 0; i < cdmListMax; i++ {
		a(&out, "nb%d", i)
	}
	for i := 0; i < cdmNewObjW; i++ {
		a(&out, "no%d", i)
	}
	out = append(out, "batF4", "v1_30", "v1_34", "v2_30", "v2_34")
	for k := 0; k < cdmMembers; k++ {
		for _, f := range cdmPortF {
			a(&out, "o%s_%d", f.name, k)
		}
	}
	out = append(out, "n52af50", "a52af50_1", "a52af50_2", "a52af50_3", "a52af50_4", "n523a80", "a523a80_1", "a523a80_2", "n522fe0")
	for i := 1; i <= 6; i++ {
		a(&out, "a522fe0_%d", i)
	}
	out = append(out, "n523330", "a523330_1", "a523330_2", "a523330_3", "n547a50", "a547a50_1", "n58aad0")
	for i := 1; i <= 3; i++ {
		for k := 0; k < 4; k++ {
			a(&out, "a58aad0_%d_%d", i, k)
		}
	}
	out = append(out, "n52e690")
	for i := 1; i <= 5; i++ {
		for k := 0; k < 2; k++ {
			a(&out, "a52e690_%d_%d", i, k)
		}
	}
	out = append(out, "n561a20", "a561a20_1")
	return
}

func init() {
	in, out := cdmNames()
	Nodes["0052a890"] = Node{Addr: "0052a890", In: in, Out: out,
		F:   characterDamageApplyNode,
		Emu: func(in []int64, r Rand) []int64 { return characterDamageApply(in, r, false) }}
}

func characterDamageApplyNode(in []int64, r Rand) []int64 { return characterDamageApply(in, r, true) }

// cdmX looks an input up by name.
var cdmX = func() map[string]int {
	in, _ := cdmNames()
	m := map[string]int{}
	for i, n := range in {
		m[n] = i
	}
	return m
}()

type cdmR struct { // stub traces
	n52a    int
	a52a    [4]int64
	n523a   int
	a523a   [2]int64
	n522    int
	a522    [6]int64
	n5233   int
	a5233   [3]int64
	n547    int
	a547    int64
	n58     int
	a58     [3][4]int64
	n52e    int
	a52e    [5][2]int64
	n561    int
	a561    int64
	ret     uint32
	hp, a83 uint32
	kl      [6]uint32
	kd, nb  [cdmListMax]uint32
	no      [cdmNewObjW]uint32
	batF4   uint32
	v1, v2  [2]uint32
	port    [cdmMembers][10]int64
}

func characterDamageApply(in []int64, r Rand, real bool) []int64 {
	names, _ := cdmNames()
	if len(in) != len(names) {
		panic("character_damage_apply: wrong input count")
	}
	g := func(n string) int64 { return in[cdmX[n]] }
	u := func(n string) uint32 { return uint32(in[cdmX[n]]) }
	b := func(n string) bool { return in[cdmX[n]]&0xff != 0 }
	who := g("who")
	if who < 0 || who >= cdmMembers {
		panic("character_damage_apply: member outside the eight fixture records")
	}
	if g("pres")&0xff == 0 {
		panic("character_damage_apply: assert, member absent") // 0x52a8ca
	}
	if g("kill") != 0 && g("kill") != cdmKl {
		panic("character_damage_apply: kill list outside the fixture")
	}
	if g("batPtr") != 0 && g("batPtr") != cdmArena+86016 {
		panic("character_damage_apply: battle object outside the fixture")
	}
	if v := g("mallocRet"); v != 0 && v != cdmNewObj {
		panic("character_damage_apply: malloc result outside the fixture")
	}
	if v := g("newRet"); v != 0 && v != cdmNewBuf {
		panic("character_damage_apply: operator new result outside the fixture")
	}
	if g("kl64") != cdmKlData {
		panic("character_damage_apply: list data outside the fixture")
	}
	C := cdmChars + who*0x1862
	o := &cdmR{}
	o.hp, o.a83 = u("hp"), uint32(g("a83")&0xffff)
	for i, n := range []string{"kl0", "kl4", "kl4c", "kl5c", "kl60", "kl64"} {
		o.kl[i] = u(n)
	}
	for i := 0; i < cdmListMax; i++ {
		o.kd[i], o.nb[i] = u(fmt.Sprintf("kd%d", i)), u(fmt.Sprintf("nb%d", i))
	}
	for i := range o.no {
		o.no[i] = 0x7e570000 + uint32(i) // case setup (junk proving the zeroing)
	}
	o.batF4 = u("batF4")
	o.v1, o.v2 = [2]uint32{0x6b1, 0x6b2}, [2]uint32{0x6b3, 0x6b4}
	for k := 0; k < cdmMembers; k++ {
		for f := range cdmPortF {
			o.port[k][f] = cdmPortInit(k, f)
		}
		for f, pf := range cdmPortF {
			if pf.in {
				o.port[k][f] = g(fmt.Sprintf("p%s_%d", pf.name, k)) & (1<<(8*uint(pf.size)) - 1)
			}
		}
	}
	finish := func() []int64 {
		x := []int64{int64(o.ret), int64(o.hp), int64(o.a83)}
		for _, v := range o.kl {
			x = append(x, int64(v))
		}
		for _, v := range o.kd {
			x = append(x, int64(v))
		}
		for _, v := range o.nb {
			x = append(x, int64(v))
		}
		for _, v := range o.no {
			x = append(x, int64(v))
		}
		x = append(x, int64(o.batF4), int64(o.v1[0]), int64(o.v1[1]), int64(o.v2[0]), int64(o.v2[1]))
		for k := range o.port {
			x = append(x, o.port[k][:]...)
		}
		x = append(x, int64(o.n52a))
		x = append(x, o.a52a[:]...)
		x = append(x, int64(o.n523a), o.a523a[0], o.a523a[1], int64(o.n522))
		x = append(x, o.a522[:]...)
		x = append(x, int64(o.n5233))
		x = append(x, o.a5233[:]...)
		x = append(x, int64(o.n547), o.a547, int64(o.n58))
		for a := 0; a < 3; a++ {
			x = append(x, o.a58[a][:]...)
		}
		x = append(x, int64(o.n52e))
		for a := 0; a < 5; a++ {
			x = append(x, o.a52e[a][:]...)
		}
		x = append(x, int64(o.n561), o.a561)
		return x
	}

	// ---- the callee world: ccaS cells (the real callees' state); only F reads ft*
	w := make([]int64, ccaWLen)
	if real {
		copy(w, in[cdmX["ft0"]:])
	}
	w[ccaWBase] = cdmChars
	w[ccaWWhoPres] = g("pres") & 0xff
	w[ccaWCls], w[ccaWRace], w[ccaWA79] = g("cls"), g("race"), 0
	w[ccaWS+15], w[ccaWS+17] = g("a3d"), g("a45")
	w[ccaWB01], w[ccaWB0d], w[ccaWB11] = g("b01"), g("maxHp"), g("hp")
	w[ccaWG7500] = g("g7500") & 0xff
	hp, sen := u("hp"), u("sen")
	cca := func(st int64, tm uint32, flagC bool) {
		w[ccaWB11] = int64(hp)
		s := ccaNew(w, 0, r, true)
		ccaApply(s, who, st, tm, 0, flagC)
		w = s.v
		hp = s.u(ccaWB11)
	}

	// notice is 0x590950: 0x58aad0(8, buf, name) and the text-box helper 0x58b410.
	notice := func() {
		if u("mode") == 7 && g("batPtr") == 0 {
			panic("character_damage_apply: battle mode with a null battle object (0x58b410 faults)")
		}
		cdmRec58(o, 8, cdmMsgBuf, C+5)
	}
	if hp == 0 {
		return finish()
	}
	dmg := u("dmg")
	if b("g7500") {
		notice()
		return finish()
	}
	if u("a85") != 0 {
		pool := uint32(g("a83") & 0xffff)
		if pool >= dmg {
			notice()
			o.a83 = (pool - dmg) & 0xffff
			if g("kill") != 0 {
				o.kl[1]++
			}
			return finish()
		}
		notice()
		dmg -= pool
		o.n523a, o.a523a = 1, [2]int64{who, 2}
		notice()
		if real {
			hp, sen = u("fab_hp"), u("fab_sen")
		}
	}
	// 0x52af50(who, floor(2*dmg/3), 0, kill)
	amount := dmg * 2 / 3
	o.n52a, o.a52a = 1, [4]int64{who, int64(amount), 0, g("kill")}
	if real {
		fin := make([]int64, 20)
		copy(fin, []int64{who, int64(int32(amount)), 0, g("kill"), g("fa_stamina"), int64(hp), g("fa_staMax"), g("fa_enc"),
			g("fa_a15"), g("fa_aa9"), g("fa_level"), g("fa_flag"), int64(uint32(w[ccaWS+17])), int64(o.kl[2]), g("fa_g8c8"), g("fa_g914"), g("fa_g598")})
		fo := characterFatigueApplyNode(fin, r)
		o.kl[2] = uint32(fo[3])
		if fo[6] != 0 { // 0x522fe0(who, 0x11, 0x270f, 0, 0, kill == 0)
			cca(0x11, 0x270f, fo[12] != 0)
		}
	}
	if g("kill") != 0 {
		o.kl[0] += dmg
		o.kl[1]++
	}
	if b("a4") {
		switch {
		case b("a7"):
			notice()
		case b("a5"):
			cdmRec58(o, 9, cdmStrs, int64(dmg))
		default:
			notice()
		}
	}
	// lethal blow
	if dmg >= hp {
		s := ccaNew(w, 0, r, true)
		if s.feat(2) && uint32(w[ccaWS+17]) < 7 {
			old := hp
			o.n547, o.a547 = 1, who
			if real {
				w[ccaWB11] = int64(hp)
				s = ccaNew(w, 0, r, true)
				chdRun(s, who)
				w = s.v
				hp = s.u(ccaWB11)
			}
			o.hp = hp
			cdmPortrait(o, g, u, who, hp, dmg, r)
			o.ret = old
			return finish()
		}
		dmg = hp
	}
	o.hp = hp - dmg
	hpNow := o.hp
	cdmPortrait(o, g, u, who, hpNow, dmg, r)
	if hpNow != 0 && !b("g3fc5") {
		cdmVoice(o, g, u, who, hpNow, r, uint32(w[ccaWB01]))
	}
	if hpNow == 0 && g("kill") != 0 {
		cdmAppend(o, g, who)
	}
	if hpNow == 0 {
		o.n522, o.a522 = 1, [6]int64{who, 0x12, 0x270f, 0, 0, 0}
		if g("kill") == 0 {
			o.a522[5] = 1
		}
		if real {
			cca(0x12, 0x270f, g("kill") == 0)
		}
	}
	if uint32(w[ccaWS+15]) > 0 && !b("a3") {
		if int64(r.R(100)) < int64(sen/2+0x32) {
			o.n5233, o.a5233 = 1, [3]int64{who, 0xf, 1}
		}
	}
	o.ret = dmg
	return finish()
}

func cdmRec58(o *cdmR, a1, a2, a3 int64) {
	if o.n58 >= 4 {
		panic("character_damage_apply: more 0x58aad0 calls than the stub records")
	}
	o.a58[0][o.n58], o.a58[1][o.n58], o.a58[2][o.n58] = a1, a2, a3
	o.n58++
}

// cdmPortrait is 0x59ac40(who, dmg) (with 0x562a50 and 0x406bf0); hp is +0xb11 now.
func cdmPortrait(o *cdmR, g func(string) int64, u func(string) uint32, who int64, hp, dmg uint32, r Rand) {
	if g("g3fc5")&0xff != 0 {
		return
	}
	p := &o.port[who]
	const f9c, f9d, f9f, fa3, fa7, fab, fbd, fca, fcf, fd0 = 0, 1, 2, 3, 4, 5, 6, 7, 8, 9
	bl := false
	if p[f9c]&0xff == 0 {
		p[f9f], p[f9c] = int64(dmg), 1
		if r.R(2) == 0 {
			p[fab] = 0x90
		} else {
			p[fab] = 0x91
		}
		if hp == 0 {
			p[f9d], p[fa7] = 1, 0x1e
		} else {
			p[f9d], p[fa7] = 0, 8
		}
		p[fa3] = 0
		bl = true
		if u("g6850ce") != 0 { // portrait UI flag: 0x561a20(who) unless the battle byte is set
			if g("batPtr") == 0 {
				panic("character_damage_apply: battle byte read through a null battle object (0x59ad04 faults)")
			}
			if g("bt109")&0xff == 0 {
				o.n561, o.a561 = 1, who
				p[fcf], p[fa3] = 1, 0xffffffff
			}
		}
	} else {
		p[f9f] = int64(uint32(p[f9f]) + dmg)
		p[fa3] = 0
		if hp == 0 && p[f9d]&0xff == 0 {
			p[f9d], p[fa7] = 1, 0x1e
		}
	}
	if p[fd0]&0xff == 0 && u("mode") == 7 && g("batPtr") != 0 { // 0x562a50(1 << who)
		o.batF4 |= 1 << uint(who)
	}
	if p[fbd]&0xff != 0 && bl {
		p[fa3] = 0xffffffff
		return
	}
	p[fca] = int64(100 + u("tick")) // 0x406bf0(100)
	p[fca] &= 0xffffffff
}

// cdmVoice is 0x52f2c0(char): the pain cue of a member who is still alive.
func cdmVoice(o *cdmR, g func(string) int64, u func(string) uint32, who int64, hp uint32, r Rand, b01 uint32) {
	C := cdmChars + who*0x1862
	if u("maxHp") == 0 {
		panic("character_damage_apply: zero maximum hit points (divide fault in 0x52f2c0)")
	}
	pct := hp * 100 / u("maxHp")
	bit := func(name string) bool { return uint32(g(name))>>uint(who)&1 != 0 }
	bl, second := bit("vb3"), bit("vb2") // 0x52dd90 for indexes 3 and 2
	voice := func(a2 int64, a3 int64, a4 int64) {
		if o.n52e >= 2 {
			panic("character_damage_apply: more 0x52e690 calls than the stub records")
		}
		k := o.n52e
		for a, v := range []int64{C, a2, a3, a4, 0x7f} {
			o.a52e[a][k] = v
		}
		o.n52e++
	}
	cue := func(a2, a3, a4 int64) {
		ret := g(fmt.Sprintf("r52e%d", o.n52e))
		voice(a2, a3, a4)
		switch ret {
		case 0:
		case cdmVoice1:
			o.v1 = [2]uint32{0x5dc, cdmTick}
		case cdmVoice2:
			o.v2 = [2]uint32{0x5dc, cdmTick}
		default:
			panic("character_damage_apply: voice object outside the fixture")
		}
	}
	if b01 != 0xf && b01 != 0x11 {
		if pct < 0x14 && !bl {
			cue(3, 1, 2)
		} else if pct < 0x32 {
			idx := int64(2)
			if r.R(2) != 0 || second {
				idx = 0x1c
			}
			cue(idx, 1, 2)
		}
	}
	arr := [3]int64{g("vc0"), g("vc1"), g("vc2")}
	voice(arr[r.R(3)], 1, 1)
}

// cdmAppend appends a new record {1, who, 0...} to the kill list.
func cdmAppend(o *cdmR, g func(string) int64, who int64) {
	if g("mallocRet") == 0 {
		return
	}
	for i := range o.no {
		o.no[i] = 0
	}
	o.no[0], o.no[1] = 1, uint32(who)
	n, capa := int32(o.kl[3]), int32(o.kl[4])
	if n < 0 || n >= cdmListMax {
		panic("character_damage_apply: kill list count outside the fixture (0..15)")
	}
	if n+1 > capa {
		if g("newRet") == 0 {
			return
		}
		copy(o.nb[:], o.kd[:n])
		o.kl[5], o.kl[4] = cdmNewBuf, uint32(n+1)
		o.nb[n] = cdmNewObj
	} else {
		o.kd[n] = cdmNewObj
	}
	o.kl[3]++
}
