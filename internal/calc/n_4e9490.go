package calc

import "fmt"

// combat_action_selection 0x4e9490() (build/wiz8.asm:313142..313510), cdecl, no result.
//
// cb = *0x6836a8, slots S = *0x685178 (stride 0x106: +0 present, +1 action, +0x3d
// alternate action), chars C = *0x685174 (stride 0x1862: +0xb11 HP, +0xb01 main status),
// combat area i = cb+0x18+0xd4*i (+0 phase, byte +0x34 handled). cb+0x7b0 set on entry
// asserts (0x4e94a6). Loop (0x4e94ca) while nobody was picked:
//   - A (cb+0x910 == 0 or cb+0x918 == 3): the first present slot whose area phase equals
//     cb+8 and that stands (HP != 0, status < 0xf) is picked (0x7b0 := 1, 0x7b4 := i);
//     fallen ones get action -1, phase 0 and, once, 0x562a50((1<<i)|0x100000).
//   - B: the same over the monster list *0x683fad (0x5e2c70 count, 0x4e5620 entry):
//     an active monster (+0x15) whose group [+0x10] phase equals cb+8 and that can act
//     (+0x2b != 0, +0x107 < 0xf, byte +0x24e == 0) is picked (0x7b8 := monster);
//     otherwise +0x2e1 := -1, group phase 0 and, once per group, 0x562a50(0x100000).
//   - C (+0x910 != 0, +0x918 == 0, +0x914 == cb+8): the first slot 0x524a10 accepts runs
//     0x4efc00 (movement phase enter); none: 0x58ac00(0xc, text +0x8a8, -1, -1, 0) and
//     0x4f0630. Either way the round's movement turn is taken (local flag).
//   - D (nothing picked, no turn): with +0x918 == 2 the same scan runs 0x4efda0 instead
//     (no turn flag); then unless (0x683fce set and 0x4f00c0 refuses) cb+8 += 0x4f0500.
//   - E/F (picked or turn): the picked monster's +0x2e1 in {1, 8, -1}, or the picked
//     slot's current action (+1 when its area is handled, else +0x3d) in {4, 5, -1}
//     (0x4e79a0 decides when 0x6850d2 is set), skips the timer; with 0x6850d2 clear a
//     false 0x4e79a0(slot, kind, 0, 0) also skips it. Timer: with hostiles 0x683fa5 or
//     charmed 0x6850be and options bit 6 (0x41f140), cb+0x7ac := clock + 0x6850dd,
//     capped at 800 while cb+0xa60 is set (not when 0x6850d2 and the turn flag). cb+0xa60 := 0.
//     The loop repeats while nothing is picked, 0x683fce is clear, no turn and cb+8 <= 100.
//
// After the loop with cb+8 > 100: cb+0xa50/0xa51 := 0; in combat (0x683f94) each present
// standing slot (status <= 0xc) with action 9 counts in cb+0xa51 and the first one runs
// 0x560e10(i, 0). With none, 0x4e9b20() and 0x4e9f90(1); a false defeat check with
// 0x6850d2 set runs 0x4e8370().
//
// Address stubs. D4 formulas whose return is read: 0x524a10, 0x4f0500, 0x4f00c0, 0x4e79a0,
// 0x4e9f90 (Emu replays the case's r* inputs; F runs characterActionEligibilityNode,
// combatPhaseStepNode, combatMovementPhaseLimitNode, pcActionEligibilityNode and
// combatDefeatResolutionNode, the latter two on the node's slots plus the F-only pa*/df_*
// inputs). D4 0x4e9b20, 0x4e8370: return ignored (0x5a5a5a5a check), count only.
// 排除 0x4f0630 (its unconditional cb+0x918 := 3 and 0x683fce := 0 are the stub's writes),
// 0x562a50, 0x58ac00, 0x408860. Every call's arguments and count are outputs.
// Native (modelled below): 0x4efc00, 0x4efda0, 0x560e10, 0x41f140/0x41f160/0x41efe0
// (options dword *0x652dac), 0x406bf0 (clock 0x6eb70c), 0x5553c0 (equal argument: only
// 0x686a40 := [0x686a44]), 0x5a1ea0/0x5a1dd0 (fixture vobj returns at once).
// Fixtures: monster list of at most caselMons records, monsters 0 and 1 share group 0,
// 2 and 3 have groups 1 and 2; UI flags 0x683f95..0x683f9a, 0x68eddc and 0x659664 are 0.
const (
	caselChars  = 0x20000000
	caselChar   = 0x1862
	caselText   = 0x7e000000 // case fixture: text pointer +x of *0x68c09c holds caselText+x
	caselMon0   = 0x20012000 // monster k at caselMon0+caselMonSz*k
	caselMonSz  = 0x300
	caselMons   = 4
	caselGroups = 3
	caselList   = 0x20011000
	caselOpts   = 0x20011100
	caselSound  = 0x6178b4 // 0x4efc93 / 0x4efdf0
	caselNone   = 0xffffffff
)

var caselGroupOf = [caselMons]int{0, 0, 1, 2}

var caselStubs = []struct{ addr, args, n int }{
	{0x524a10, 1, 8}, {0x4f0500, 0, 101}, {0x4f00c0, 0, 1}, {0x4e79a0, 4, 1}, {0x4e9b20, 0, 1},
	{0x4e9f90, 1, 1}, {0x4e8370, 0, 1}, {0x4f0630, 0, 1}, {0x562a50, 1, 11}, {0x58ac00, 5, 2},
	{0x408860, 2, 1}}

// caselPcaeOwn are the pcActionEligibilityNode inputs this node supplies itself: slot,
// attack context, combat flag, the 0x683f95/96/97/98/99 fixtures, the slot's two actions,
// both fixed call parameters and present/HP/status of every slot (indices 30..53).
func caselPcaeOwn(k int) bool {
	switch k {
	case 0, 1, 7, 8, 9, 10, 11, 24, 25, 27, 28, 29:
		return true
	}
	return k >= 30 && k < 54
}

const caselPcaeN = 264

func caselDefOwn(n string) bool {
	if n == "a" {
		return true
	}
	for m := 0; m < 8; m++ {
		if n == fmt.Sprint("pres", m) || n == fmt.Sprint("hp", m) || n == fmt.Sprint("b01_", m) {
			return true
		}
	}
	return false
}

func caselNames() (in, out []string) {
	add := func(l *[]string, f string, a ...any) { *l = append(*l, fmt.Sprintf(f, a...)) }
	in = []string{"g7b0", "g7b4", "g7b8", "phase", "g910", "g914", "g918", "ga60", "g7ac", "g91c",
		"ga50", "ga51", "ga55"}
	for i := 0; i < 8; i++ {
		for _, p := range []string{"pres", "act", "alt", "hp", "st", "aph", "afl"} {
			add(&in, "%s%d", p, i)
		}
	}
	in = append(in, "monList", "monCount")
	for k := 0; k < caselMons; k++ {
		for _, p := range []string{"mOn", "mAble", "mSt", "mBusy", "mAct"} {
			add(&in, "%s%d", p, k)
		}
	}
	for j := 0; j < caselGroups; j++ {
		add(&in, "grPh%d", j)
		add(&in, "grFl%d", j)
	}
	in = append(in, "g683fce", "g683fd3", "g683fcf", "g6850d2", "g683fa5", "g6850be", "g6850dd",
		"optPtr", "opts", "clock", "g686a44", "g686a40", "g683f94", "r4f0500")
	for k := 0; k < 8; k++ {
		add(&in, "r524a10_%d", k)
	}
	in = append(in, "r4f00c0", "r4e79a0", "r4e9f90")
	for k := 0; k < caselPcaeN; k++ {
		if !caselPcaeOwn(k) {
			add(&in, "pa%d", k)
		}
	}
	dn, _ := cdresNames()
	for _, n := range dn {
		if !caselDefOwn(n) {
			add(&in, "df_%s", n)
		}
	}
	out = []string{"phase", "g7b0", "g7b4", "g7b8", "g7ac", "ga50", "ga51", "ga55", "ga60", "g918", "g91c"}
	for i := 0; i < 8; i++ {
		add(&out, "act%d", i)
		add(&out, "aph%d", i)
		add(&out, "afl%d", i)
	}
	for k := 0; k < caselMons; k++ {
		add(&out, "mAct%d", k)
	}
	for j := 0; j < caselGroups; j++ {
		add(&out, "grPh%d", j)
		add(&out, "grFl%d", j)
	}
	out = append(out, "g683fce", "g683fd3", "opts", "g686a40", "g68ed1c", "g68ed20", "g68ed24", "g68ed10", "g68eddc")
	for _, s := range caselStubs {
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
	in, out := caselNames()
	Nodes["004e9490"] = Node{Addr: "004e9490", In: in, Out: out, F: combatActionSelectionNode,
		Emu: func(in []int64, r Rand) []int64 { return caselRun(in, r, false) }}
}

func combatActionSelectionNode(in []int64, r Rand) []int64 { return caselRun(in, r, true) }

func caselRun(in []int64, r Rand, real bool) []int64 {
	names, outNames := caselNames()
	if len(in) != len(names) {
		panic(fmt.Sprintf("combat_action_selection: expected %d inputs, got %d", len(names), len(in)))
	}
	ix := make(map[string]int, len(names))
	for i, n := range names {
		ix[n] = i
	}
	u := func(n string) uint32 { return uint32(in[ix[n]]) }
	ui := func(p string, i int) uint32 { return u(fmt.Sprint(p, i)) }

	type rec struct {
		n    int
		args [][]uint32
	}
	stub := map[int]*rec{}
	limit := map[int]int{}
	for _, s := range caselStubs {
		stub[s.addr], limit[s.addr] = &rec{}, s.n
	}
	call := func(a int, args ...uint32) int {
		s := stub[a]
		if s.n == limit[a] {
			panic(fmt.Sprintf("combat_action_selection: stub %#x called more than %d times", a, limit[a]))
		}
		s.n++
		s.args = append(s.args, args)
		return s.n - 1
	}

	if u("g7b0") != 0 {
		panic("combat_action_selection: cb+0x7b0 set on entry asserts (0x4e94a6)")
	}
	// Entry invariant: whoever clears cb+0x7b0 also clears cb+0x7b8 (0x4e72ba, 0x4e936b,
	// 0x4e7f93, 0x4e4371).
	if u("g7b8") != 0 {
		panic("combat_action_selection: cb+0x7b8 nonzero while cb+0x7b0 is clear")
	}
	g7b0, g7b4, g7b8 := uint32(0), u("g7b4"), uint32(0)
	if int32(g7b4) < -1 || int32(g7b4) > 7 {
		panic("combat_action_selection: cb+0x7b4 outside -1..7")
	}
	phase, g910, g914, g918 := u("phase"), u("g910"), u("g914"), u("g918")
	a60, g7ac, g91c := u("ga60")&0xff, u("g7ac"), u("g91c")
	a50, a51, a55 := u("ga50")&0xff, u("ga51")&0xff, u("ga55")&0xff
	var pres, act, alt, hp, st, aph, afl [8]uint32
	for i := 0; i < 8; i++ {
		pres[i], act[i], alt[i] = ui("pres", i)&0xff, ui("act", i), ui("alt", i)
		hp[i], st[i], aph[i], afl[i] = ui("hp", i), ui("st", i), ui("aph", i), ui("afl", i)&0xff
	}
	count := uint32(0)
	switch u("monList") {
	case 0:
	case caselList:
		count = u("monCount")
	default:
		panic("combat_action_selection: monster list pointer outside the case object")
	}
	if count > caselMons {
		panic(fmt.Sprintf("combat_action_selection: fixture holds at most %d monsters", caselMons))
	}
	var mOn, mAble, mSt, mBusy, mAct [caselMons]uint32
	for k := 0; k < caselMons; k++ {
		mOn[k], mAble[k], mSt[k] = ui("mOn", k)&0xff, ui("mAble", k), ui("mSt", k)
		mBusy[k], mAct[k] = ui("mBusy", k)&0xff, ui("mAct", k)
	}
	var grPh, grFl [caselGroups]uint32
	for j := 0; j < caselGroups; j++ {
		grPh[j], grFl[j] = ui("grPh", j), ui("grFl", j)&0xff
	}
	fce, fd3 := u("g683fce")&0xff, u("g683fd3")
	d2 := u("g6850d2")&0xff != 0
	optOn := false
	switch u("optPtr") {
	case 0:
	case caselOpts:
		optOn = true
	default:
		panic("combat_action_selection: options pointer outside the case object")
	}
	opts, clock, a44, a40 := u("opts"), u("clock"), u("g686a44"), u("g686a40")
	ed1c, ed20, ed24, ed10 := uint32(0), uint32(0), uint32(0), uint32(0)

	bit6 := func() bool { return optOn && opts>>6&1 != 0 } // 0x41f140
	eligible := func(i int) bool {                         // 0x524a10(i)
		k := call(0x524a10, uint32(i))
		if real {
			return characterActionEligibilityNode([]int64{int64(i), int64(pres[i]), int64(hp[i]), int64(st[i])}, r)[0] != 0
		}
		return u(fmt.Sprint("r524a10_", k))&0xff != 0
	}
	// movement presentation shared by 0x4efc00 and 0x4efda0 once the mode is 1 or 2.
	present := func() {
		if bit6() && g918 == 2 {
			call(0x58ac00, 8, caselText+0x874, caselNone, caselNone, 0)
		}
		call(0x408860, caselSound, 0)
		if optOn {
			opts &^= 0x40 // 0x41f160
			if g910 == 2 {
				opts |= 0x100 // 0x41efe0
			}
		}
		if d2 {
			g91c = clock + 1000 // 0x406bf0(1000)
		}
		a40 = a44 // 0x5553c0([0x686a44])
	}
	enter := func() { // 0x4efc00
		if g910 == 1 || g910 == 2 {
			fce = 1
			call(0x58ac00, 8, caselText+0x86c, caselNone, caselNone, 0)
			fd3 = 0
			present()
		}
		g918, a55 = 1, 0
	}
	resume := func() { // 0x4efda0
		if g910 == 1 || g910 == 2 {
			present()
		}
		g918 = 1
	}
	nobody := func() {
		call(0x58ac00, 0xc, caselText+0x8a8, caselNone, caselNone, 0)
		call(0x4f0630)
		g918, fce = 3, 0
	}
	pcae := func(slot, kind uint32) bool { // 0x4e79a0(slot, kind, 0, 0)
		call(0x4e79a0, slot, kind, 0, 0)
		if !real {
			return u("r4e79a0")&0xff != 0
		}
		v := make([]int64, caselPcaeN)
		for k := range v {
			if !caselPcaeOwn(k) {
				v[k] = in[ix[fmt.Sprint("pa", k)]]
			}
		}
		v[0], v[1], v[7] = int64(slot), int64(kind), int64(u("g683f94")&0xff)
		v[10], v[11] = int64(int32(act[slot])), int64(int32(alt[slot]))
		for m := 0; m < 8; m++ {
			v[30+3*m], v[31+3*m], v[32+3*m] = int64(pres[m]), int64(hp[m]), int64(st[m])
		}
		return pcActionEligibilityNode(v, r)[0]&0xff != 0
	}

	moved := false
	for {
		if g910 == 0 || g918 == 3 {
			for i := 0; i < 8; i++ {
				if pres[i] == 0 || aph[i] != phase {
					continue
				}
				if hp[i] != 0 && st[i] < 0xf {
					g7b0, g7b4 = 1, uint32(i)
					break
				}
				act[i], aph[i] = caselNone, 0
				if afl[i] == 0 {
					afl[i] = 1
					call(0x562a50, 1<<i|0x100000)
				}
			}
		}
		if g7b0 == 0 {
			for k := 0; uint32(k) < count; k++ {
				g := caselGroupOf[k]
				if mOn[k] == 0 || grPh[g] != phase {
					continue
				}
				if mAble[k] != 0 && mSt[k] < 0xf && mBusy[k] == 0 {
					g7b0, g7b8 = 1, caselMon0+caselMonSz*uint32(k)
					break
				}
				mAct[k], grPh[g] = caselNone, 0
				if grFl[g] == 0 {
					grFl[g] = 1
					call(0x562a50, 0x100000)
				}
			}
		}
		if g7b0 == 0 && g910 != 0 && g918 == 0 && g914 == phase {
			found := false
			for i := 0; i < 8 && !found; i++ {
				found = eligible(i)
			}
			if found {
				enter()
			} else {
				nobody()
			}
			moved = true
		}
		if g7b0 == 0 && !moved {
			if g918 == 2 {
				found := false
				for i := 0; i < 8 && !found; i++ {
					found = eligible(i)
				}
				if found {
					resume()
				} else {
					nobody()
				}
			}
			advance := true
			if fce != 0 {
				call(0x4f00c0)
				if real {
					advance = combatMovementPhaseLimitNode([]int64{int64(phase), int64(g910), int64(g914),
						int64(g918), int64(fce), int64(fd3), int64(u("g683fcf"))}, r)[0]&0xff != 0
				} else {
					advance = u("r4f00c0")&0xff != 0
				}
			}
			if advance {
				call(0x4f0500)
				step := u("r4f0500")
				if real {
					step = uint32(combatPhaseStepNode([]int64{int64(g910)}, r)[0])
				}
				phase += step
			}
		} else {
			// 0x4e9756: skip decides whether the timer block is bypassed.
			skip, timer := false, true
			switch {
			case g7b0 == 0:
			case g7b8 != 0:
				v := mAct[(g7b8-caselMon0)/caselMonSz]
				bl := v == 1 || v == 8 || v == caselNone
				skip = a60 != 0 || bl
			case g7b4 == caselNone:
			default:
				slot, kind, a := g7b4, uint32(1), alt[g7b4]
				if afl[slot] != 0 {
					kind, a = 0, act[slot]
				}
				bl := false
				if a == 4 || a == 5 || a == caselNone {
					bl = !d2 || pcae(slot, kind)
				}
				if !d2 && !pcae(g7b4, kind) {
					timer = false
				}
				skip = timer && bl
			}
			if timer && !skip && (u("g683fa5") != 0 || u("g6850be") != 0) && bit6() {
				d := u("g6850dd")
				if a60 != 0 && !(d2 && moved) && d >= 0x320 {
					d = 0x320
				}
				g7ac = d + clock // 0x406bf0
			}
			a60 = 0
		}
		if !(g7b0 == 0 && fce == 0 && !moved && phase <= 100) {
			break
		}
	}

	if phase > 100 {
		a50, a51 = 0, 0
		if u("g683f94")&0xff != 0 {
			for i := 0; i < 8; i++ {
				if pres[i] == 0 || hp[i] == 0 || st[i] > 0xc || act[i] != 9 {
					continue
				}
				a51 = (a51 + 1) & 0xff
				if a50 == 0 {
					a50 = 1
					// 0x560e10(i, 0) with the UI fixtures: current character and screen 6.
					ed1c, ed20, ed24, ed10 = uint32(i), caselChars+caselChar*uint32(i), 0, 6
				}
			}
		}
		if a50 == 0 {
			call(0x4e9b20)
			call(0x4e9f90, 1)
			defeat := u("r4e9f90")&0xff != 0
			if real {
				dn, _ := cdresNames()
				v := make([]int64, len(dn))
				for k, n := range dn {
					switch {
					case n == "a":
						v[k] = 1
					case caselDefOwn(n):
						var m int
						var p string
						fmt.Sscanf(n[len(n)-1:], "%d", &m)
						p = n[:len(n)-1]
						switch p {
						case "pres":
							v[k] = int64(pres[m])
						case "hp":
							v[k] = int64(hp[m])
						default:
							v[k] = int64(st[m])
						}
					default:
						v[k] = in[ix["df_"+n]]
					}
				}
				defeat = combatDefeatResolutionNode(v, r)[0]&0xff != 0
			}
			if !defeat && d2 {
				call(0x4e8370)
			}
		}
	}

	out := make([]int64, 0, len(outNames))
	out = append(out, int64(phase), int64(g7b0), int64(int32(g7b4)), int64(g7b8), int64(g7ac),
		int64(a50), int64(a51), int64(a55), int64(a60), int64(g918), int64(g91c))
	for i := 0; i < 8; i++ {
		out = append(out, int64(int32(act[i])), int64(aph[i]), int64(afl[i]))
	}
	for k := 0; k < caselMons; k++ {
		out = append(out, int64(int32(mAct[k])))
	}
	for j := 0; j < caselGroups; j++ {
		out = append(out, int64(grPh[j]), int64(grFl[j]))
	}
	out = append(out, int64(fce), int64(fd3), int64(opts), int64(a40), int64(ed1c), int64(ed20),
		int64(ed24), int64(ed10), 0)
	for _, s := range caselStubs {
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
