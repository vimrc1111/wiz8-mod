package calc

import "fmt"

// game_time_rule_ticks 0x502010(dt) (build/wiz8.asm:347699..348009), cdecl.
//
// Clock: c = u32(0x686a4c + dt); 0x686a48 += c/1000, 0x686a4c = c%1000.
// Rule ticks: 0x6874fb += dt. With byte 0x685071 clear, n = 0x6874fb/120000; n != 0
// takes n*120000 off and calls 0x502d00(n) (game_turns_passed). With it set, n is the
// stack dword the function never writes (entry esp-0xc), the case input `junk`.
// Hourly: 0x68750d += dt, h = 0x68750d/3600000; h != 0 takes h*3600000 off and calls
// 0x51e920(slot, h) (item_charge_increment) for every 12-byte item slot holding
// item 0x266: the 12 slots at +0xf5d and the 8 at +0x1029 of each present member
// (presence byte *0x685178 + 0x106*m, record *0x685174 + 0x1862*m), then the first
// u32 0x686901 entries of the party pack at 0x685191.
// Drain: when byte 0x683f94 (combat) is clear or n != 0, and byte 0x652da4 is set:
// in combat d = 4 and 0x6875a6 -= 4*2500 (n == 0 there is the assert block 0x5021e3,
// unreachable: combat with n == 0 never gets past 0x5021b9); else 0x6875a6 += dt and
// d = 0x6875a6/2500; d != 0 takes d*2500 off and calls 0x5044d0(d)
// (environment_stamina_hp_drain). Otherwise 0x6875a6, byte 0x6875a3 and 0x687608 are
// cleared. Recovery: in combat 0x6875f3 := 0; else 0x6875f3 += dt, e = /1400 and e != 0
// takes e*1400 off and calls 0x504670(e) (party_stamina_recovery).
// Auto-rest (out of combat, byte 0x683fc5 clear): no member passes 0x524a10 (0x4e7ca0),
// 0x4eeec0 holds, byte 0x6875a3 clear, 0x41f010 false, 0x41f070 true, 0x561440 true.
// Then 0x56aab0 when byte 0x6840bc is set; 0x4e7ca0 again (same answer: nothing
// between the calls touches members, so its true branch 0x50232e is unreachable);
// byte 0x683fc6 := 1, 0x58ac00(0xc, text +0x1e50, -1, -1, 0), 0x576030(0,0,0,-1,-1),
// 0x52db30, for each present member 0x52f000(portrait 0x6836b8 + 0x118*m, 2), 0x598c70,
// 0x562a50(0xff), byte 0x683fc5 := 1, 0x4f2380(0x137), 0x4f2040(0x137), 0x683fc7 := 0,
// 0x56b4e0, word 0x683fcb := 0, 0x48fc10(0x619844, 0, 1), 0x6850b0 := 0.
// Wake (0x5023ed): bytes 0x683fc5 and 0x683fc6 set, 0x4e7ca0 true and word 0x683fcb
// == 1: 0x482750 stores 12.0f at 0x60a390, 0x452f50(0), 0x43aa20, 0x48cbe0, 0x56b5f0,
// word 0x683fcb := 2, 0x50da00, 0x42b770(1, 1).
//
// Native helpers modelled here: 0x41f010 (scene *0x652dac: null or flag bit 0 clear
// -> 0; else 1 when a float at +0x88/+0x8c/+0x90 compares unequal and ordered to
// 0.0f), 0x41f070 (null -> 0, else flag bit 4), 0x561440 (bytes 0x683f94..0x683f99 all
// zero), 0x482750, 0x52f000 (byte +0x99 := 0, +0x8d := 2: the requested 2 and the
// b01 >= 0xf / 0x683fc5 override agree).
//
// Address stubs: D4 0x502d00, 0x51e920, 0x5044d0, 0x504670, 0x4eeec0 (returns from
// case inputs r*; only 0x4eeec0's is read) and the 排除 helpers; every call's
// arguments and the counts are outputs (ecx of the thiscalls 0x52db30 / 0x43aa20 is
// not recordable). Emu replays them. F runs the real 0x4eeec0 (partySurvivalCheckNode)
// on the node's presence, +0xb11 and +0xb01, and the real 0x5044d0
// (environmentStaminaHpDrainNode) on the node's values plus the F-only inputs g687500,
// g68510c, t768, race*, sta*, then feeds 0x6875a3 / 0x687608 / 0x6875a6 forward.
// 0x502d00, 0x51e920 and 0x504670 write nothing the node reads later, so F does not
// run them.
//
// Fixture cap (calc panics past it): the pack count 0x686901 is at most 500, the
// table's capacity (0x685191..0x6868f5, as in n_521240.go).
const (
	gtrM       = 8
	gtrSlots   = 20 // 12 at +0xf5d, 8 at +0x1029
	gtrPack    = 500
	gtrCalls   = gtrM*gtrSlots + gtrPack
	gtrItem    = 0x266   // 0x502111
	gtrRuleDiv = 120000  // 0x502062 (0x45e7b273 >> 47)
	gtrHourDiv = 3600000 // 0x5020aa (0x95217cb1 >> 53)
	gtrDrain   = 2500    // 0x502209 (0xd1b71759 >> 43), 0x502223
	gtrRecDiv  = 1400    // 0x502257 (0x5d9f7391 >> 41)
	gtrChars   = 0x20000000
	gtrChar    = 0x1862
	gtrPortr   = 0x6836b8
	gtrPackA   = 0x685191
)

var gtrStubs = []struct{ addr, args, n int }{
	{0x502d00, 1, 1}, {0x51e920, 2, gtrCalls}, {0x5044d0, 1, 1}, {0x504670, 1, 1}, {0x4eeec0, 0, 1},
	{0x56aab0, 0, 1}, {0x58ac00, 5, 1}, {0x576030, 5, 1}, {0x52db30, 0, 1}, {0x598c70, 0, 1},
	{0x562a50, 1, 1}, {0x4f2380, 1, 1}, {0x4f2040, 1, 1}, {0x56b4e0, 0, 1}, {0x48fc10, 3, 1},
	{0x452f50, 1, 1}, {0x43aa20, 0, 1}, {0x48cbe0, 0, 1}, {0x56b5f0, 0, 1}, {0x50da00, 0, 1}, {0x42b770, 2, 1}}

var gtrOutG = []string{"o686a48", "o686a4c", "o6874fb", "o68750d", "o6875a6", "o6875a3", "o687608", "o6875f3",
	"o683fc5", "o683fc6", "o683fc7", "o683fcb", "o6850b0", "o60a390"}

func gtrNames() (in, out []string) {
	add := func(l *[]string, f string, a ...any) { *l = append(*l, fmt.Sprintf(f, a...)) }
	in = []string{"dt", "junk", "g686a4c", "g686a48", "g6874fb", "g685071", "g68750d"}
	for _, p := range []string{"pres", "hp", "b01_"} {
		for m := 0; m < gtrM; m++ {
			add(&in, "%s%d", p, m)
		}
	}
	for m := 0; m < gtrM; m++ {
		for k := 0; k < gtrSlots; k++ {
			add(&in, "it%d_%d", m, k)
		}
	}
	in = append(in, "packN")
	for k := 0; k < gtrPack; k++ {
		add(&in, "pack%d", k)
	}
	in = append(in, "g683f94", "g652da4", "g6875a6", "g6875a3", "g687608", "g6875f3", "g683fc5", "g683fc6",
		"g683fc7", "g683fcb", "g6840bc", "g6850b0", "g60a390", "g683f95", "g683f96", "g683f97", "g683f98", "g683f99",
		"scenePtr", "sceneFlags", "sf88", "sf8c", "sf90", "t1e50")
	for _, p := range []string{"p8d_", "p99_"} {
		for m := 0; m < gtrM; m++ {
			add(&in, "%s%d", p, m)
		}
	}
	in = append(in, "r502d00", "r51e920", "r5044d0", "r504670", "r4eeec0", "g687500", "g68510c", "t768")
	for _, p := range []string{"race", "sta"} {
		for m := 0; m < gtrM; m++ {
			add(&in, "%s%d", p, m)
		}
	}
	out = append(out, gtrOutG...)
	for _, p := range []string{"op8d_", "op99_"} {
		for m := 0; m < gtrM; m++ {
			add(&out, "%s%d", p, m)
		}
	}
	for _, s := range gtrStubs {
		add(&out, "n%x", s.addr)
		for a := 1; a <= s.args; a++ {
			if s.n == 1 {
				add(&out, "a%x_%d", s.addr, a)
				continue
			}
			for c := 0; c < s.n; c++ {
				add(&out, "a%x_%d_%d", s.addr, a, c)
			}
		}
	}
	return
}

func init() {
	in, out := gtrNames()
	Nodes["00502010"] = Node{Addr: "00502010", In: in, Out: out, F: gameTimeRuleTicksNode,
		Emu: func(in []int64, r Rand) []int64 { return gtrRun(in, r, false) }}
}

func gameTimeRuleTicksNode(in []int64, r Rand) []int64 { return gtrRun(in, r, true) }

func gtrRun(in []int64, r Rand, real bool) []int64 {
	names, outNames := gtrNames()
	if len(in) != len(names) {
		panic(fmt.Sprintf("game_time_rule_ticks: expected %d inputs, got %d", len(names), len(in)))
	}
	ix := make(map[string]int, len(names))
	for i, n := range names {
		ix[n] = i
	}
	u := func(n string) uint32 { return uint32(in[ix[n]]) }
	b := func(n string) uint32 { return uint32(in[ix[n]]) & 0xff }
	g := map[string]uint32{}
	for _, n := range gtrOutG {
		g[n] = u("g" + n[1:])
	}
	g["o6875a3"] &= 0xff
	g["o683fc5"] &= 0xff
	g["o683fc6"] &= 0xff
	g["o683fcb"] &= 0xffff
	var p8d, p99 [gtrM]uint32
	pres, hp, b01 := make([]uint32, gtrM), make([]uint32, gtrM), make([]uint32, gtrM)
	for m := 0; m < gtrM; m++ {
		p8d[m], p99[m] = u(fmt.Sprint("p8d_", m)), b(fmt.Sprint("p99_", m))
		pres[m], hp[m], b01[m] = b(fmt.Sprint("pres", m)), u(fmt.Sprint("hp", m)), u(fmt.Sprint("b01_", m))
	}
	type rec struct {
		n    int
		args [][]uint32
	}
	stub := map[int]*rec{}
	limit := map[int]int{}
	for _, s := range gtrStubs {
		stub[s.addr], limit[s.addr] = &rec{}, s.n
	}
	call := func(a int, args ...uint32) {
		s := stub[a]
		if s.n == limit[a] {
			panic(fmt.Sprintf("game_time_rule_ticks: stub %#x called more than %d times", a, limit[a]))
		}
		s.n++
		s.args = append(s.args, args)
	}
	eligible := func() bool { // 0x4e7ca0
		for m := 0; m < gtrM; m++ {
			if characterActionEligibilityNode([]int64{int64(m), int64(pres[m]), int64(hp[m]), int64(b01[m])}, r)[0] != 0 {
				return true
			}
		}
		return false
	}

	dt := u("dt")
	c := g["o686a4c"] + dt
	g["o686a48"] += c / 1000
	g["o686a4c"] = c % 1000

	g["o6874fb"] += dt
	var n uint32
	if b("g685071") == 0 {
		n = g["o6874fb"] / gtrRuleDiv
		if n != 0 {
			g["o6874fb"] -= n * gtrRuleDiv
			call(0x502d00, n)
		}
	} else {
		n = u("junk")
	}

	g["o68750d"] += dt
	if h := g["o68750d"] / gtrHourDiv; h != 0 {
		g["o68750d"] -= h * gtrHourDiv
		for m := 0; m < gtrM; m++ {
			if pres[m] == 0 {
				continue
			}
			for k := 0; k < gtrSlots; k++ {
				off := uint32(0xf5d + 12*k)
				if k >= 12 {
					off = uint32(0x1029 + 12*(k-12))
				}
				if u(fmt.Sprintf("it%d_%d", m, k)) == gtrItem {
					call(0x51e920, gtrChars+gtrChar*uint32(m)+off, h)
				}
			}
		}
		cnt := u("packN")
		if cnt > gtrPack {
			panic("game_time_rule_ticks: pack count beyond the 500-entry table")
		}
		for k := uint32(0); k < cnt; k++ {
			if u(fmt.Sprint("pack", k)) == gtrItem {
				call(0x51e920, gtrPackA+12*k, h)
			}
		}
	}

	f94 := b("g683f94")
	if (f94 == 0 || n != 0) && b("g652da4") != 0 {
		var d uint32
		if f94 != 0 {
			if n == 0 { // 0x5021b9 already sent combat with n == 0 to the clear branch
				panic("game_time_rule_ticks: unreachable assert block 0x5021e3")
			}
			d = 4
			g["o6875a6"] -= d * gtrDrain
		} else {
			g["o6875a6"] += dt
			d = g["o6875a6"] / gtrDrain
			g["o6875a6"] -= d * gtrDrain
		}
		if d != 0 {
			call(0x5044d0, d)
			if real {
				gtrDrainReal(in, ix, g, d, pres, hp, b01, r)
			}
		}
	} else {
		g["o6875a6"], g["o6875a3"], g["o687608"] = 0, 0, 0
	}

	if f94 != 0 {
		g["o6875f3"] = 0
	} else {
		g["o6875f3"] += dt
		if e := g["o6875f3"] / gtrRecDiv; e != 0 {
			g["o6875f3"] -= e * gtrRecDiv
			call(0x504670, e)
		}
	}
	if f94 == 0 && g["o683fc5"] == 0 && gtrRestReady(in, ix, g, eligible, call, real, pres, hp, b01, r) {
		if b("g6840bc") != 0 {
			call(0x56aab0)
		}
		if eligible() {
			panic("game_time_rule_ticks: unreachable block 0x50232e (0x4e7ca0 changed between calls)")
		}
		g["o683fc6"] = 1
		call(0x58ac00, 0xc, u("t1e50"), 0xffffffff, 0xffffffff, 0)
		call(0x576030, 0, 0, 0, 0xffffffff, 0xffffffff)
		call(0x52db30)
		for m := 0; m < gtrM; m++ {
			if pres[m] != 0 { // 0x52f000(rec, 2)
				p99[m], p8d[m] = 0, 2
			}
		}
		call(0x598c70)
		call(0x562a50, 0xff)
		g["o683fc5"] = 1
		call(0x4f2380, 0x137)
		call(0x4f2040, 0x137)
		g["o683fc7"] = 0
		call(0x56b4e0)
		g["o683fcb"] = 0
		call(0x48fc10, 0x619844, 0, 1)
		g["o6850b0"] = 0
	}
	if g["o683fc5"] != 0 && g["o683fc6"] != 0 && eligible() && g["o683fcb"] == 1 {
		g["o60a390"] = 0x41400000 // 0x482750(12.0f)
		call(0x452f50, 0)
		call(0x43aa20)
		call(0x48cbe0)
		call(0x56b5f0)
		g["o683fcb"] = 2
		call(0x50da00)
		call(0x42b770, 1, 1)
	}

	out := make([]int64, 0, len(outNames))
	for _, n := range gtrOutG {
		out = append(out, int64(g[n]))
	}
	for m := 0; m < gtrM; m++ {
		out = append(out, int64(p8d[m]))
	}
	for m := 0; m < gtrM; m++ {
		out = append(out, int64(p99[m]))
	}
	for _, s := range gtrStubs {
		st := stub[s.addr]
		out = append(out, int64(st.n))
		for a := 0; a < s.args; a++ {
			if s.n == 1 {
				v := int64(0)
				if st.n > 0 {
					v = int64(st.args[st.n-1][a])
				}
				out = append(out, v)
				continue
			}
			for k := 0; k < s.n; k++ {
				v := int64(0)
				if k < st.n {
					v = int64(st.args[k][a])
				}
				out = append(out, v)
			}
		}
	}
	return out
}

// gtrRestReady is the auto-rest gate 0x5022a2..0x5022ea, evaluated in order.
func gtrRestReady(in []int64, ix map[string]int, g map[string]uint32, eligible func() bool,
	call func(int, ...uint32), real bool, pres, hp, b01 []uint32, r Rand) bool {
	u := func(n string) uint32 { return uint32(in[ix[n]]) }
	if eligible() {
		return false
	}
	call(0x4eeec0)
	ok := u("r4eeec0")&0xff != 0
	if real {
		v := make([]int64, 0, 3*gtrM)
		for m := 0; m < gtrM; m++ {
			v = append(v, int64(pres[m]), int64(hp[m]), int64(b01[m]))
		}
		ok = partySurvivalCheckNode(v, r)[0] != 0
	}
	if !ok || g["o6875a3"] != 0 {
		return false
	}
	scene, flags := u("scenePtr"), u("sceneFlags")
	if scene != 0 && flags&1 != 0 { // 0x41f010
		for _, f := range []string{"sf88", "sf8c", "sf90"} {
			if x := u(f); x&0x7fffffff != 0 && !(x&0x7f800000 == 0x7f800000 && x&0x7fffff != 0) {
				return false
			}
		}
	}
	if scene == 0 || flags&0x10 == 0 { // 0x41f070
		return false
	}
	for _, a := range []string{"g683f94", "g683f95", "g683f96", "g683f97", "g683f98", "g683f99"} { // 0x561440
		if u(a)&0xff != 0 {
			return false
		}
	}
	return true
}

// gtrDrainReal runs the real 0x5044d0 on the node's values (F only) and writes back
// 0x687608, 0x6875a3 and 0x6875a6.
func gtrDrainReal(in []int64, ix map[string]int, g map[string]uint32, d uint32, pres, hp, b01 []uint32, r Rand) {
	v := []int64{int64(d), in[ix["g687500"]], int64(g["o687608"]), int64(g["o6875a3"]), int64(g["o6875a6"]),
		in[ix["g683f94"]], in[ix["g68510c"]], in[ix["t768"]]}
	for m := 0; m < gtrM; m++ {
		v = append(v, int64(pres[m]))
	}
	for m := 0; m < gtrM; m++ {
		v = append(v, int64(hp[m]), int64(b01[m]), in[ix[fmt.Sprint("race", m)]], in[ix[fmt.Sprint("sta", m)]])
		for k := 0; k < 12; k++ {
			v = append(v, in[ix[fmt.Sprintf("it%d_%d", m, k)]])
		}
	}
	o := environmentStaminaHpDrainNode(v, r)
	g["o687608"], g["o6875a3"], g["o6875a6"] = uint32(o[0]), uint32(o[1]), uint32(o[2])
}
