package calc

import (
	"fmt"
	"strconv"
)

// specialAbilityEffectNode 0x5560a0 special_ability_effect(ability, source, group,
// _, characterCount, _, characterList, _, monsterCount, _, monsterList) (cdecl),
// build/wiz8.asm:464819.
//
// Per ability a (0..31, the size of the exe tables) two effect slots s carry a
// type t = u32 0x61eefc+8a+4s, a sub-kind = u32 0x61effc+8a+4s, and the ability
// has one realm u32 0x61f0fc+4a. The notify byte is 0x68510c.
//
// Source power: a type-1 source (party index source +4) uses level L = u32
// C+0x89 and stamina/maximum stamina C+0xb1d/C+0xb19; a type-2 source (monster id
// source +8, found by 0x4e5840) uses the template byte +0x24f and monster
// +0x33/+0x2f. With s = L + min(L, 15) (u32): power = max(1, u32(st*s)/max) and both
// dice = max(1, u32(st*(s+5))/max) (unsigned division). For a type-2 source the
// exe dwords 0x61eefc / 0x61ef04 equal to 5 would double dice 0 / dice 1; they are
// 0 and 1, so calc panics if it ever reads 5. Other source types leave power and
// dice unset (stack garbage): calc panics once they are used.
//
// Type 6 in slot 0 hands the whole ability to 0x556b10(ability, source, group).
//
// Characters (count i32, ids in list order; id -1 asserts): target record
// {1, id, -1}. Per slot: slot 0 is skipped on R(100) < 10 for a type-2 source,
// slot 1 on R(100) < 50; t outside 1..5 is skipped.
//   - t1: roll = R(dice), extra = sub 7 ? R(dice)+1 : 0; 0x551ba0(source, target,
//     sub, realm, power, extra, roll, notify, notify, 0); a zero low byte without
//     notify counts a hit.
//   - t2/t3/t4/t5: x = R(d)+R(d)+R(d); amount x/3+1 (t2, t3, t5) or x+1 (t4);
//     0x552340 reduces it through 0x552250 (resistanceQuantityReductionNode), not
//     for t5 with ability 10; a positive amount goes to 0x52b7e0(id, amount, 1),
//     6 x 0x52b6d0(id, realm, amount, 1), 0x52af50(id, amount, 0, 0) or
//     0x52a890(id, amount, 0, notify, 0, 0, 0) and, without notify, into the slot's
//     total and hit count; zero runs 0x552070.
//
// Monsters (count i32, ids; id -1 asserts; record via 0x4e5840): target {3, -1,
// id}; slot 0 is skipped on R(100) < 10 for every source, slot 1 on R(100) < 50.
//   - t1: 0x551ba0 with roll max(1, dice/3) and extra sub 7 ? R(2)+1 : 0.
//   - t2: x/3+1, 0x552340, 0x52bb60(monster, amount, source, 0, 1, 0, 0, 0).
//   - t3: x/6+1, 0x552340, SP +0x2f9 -= amount>>1 (u32, wraps).
//   - t4: x, 0x552340, 0x52c070(monster, amount, 0).
//   - t5: x/3+1, 0x552340 unless ability 10, 0x52bb60(monster, amount, source, 0,
//     notify, 0, 0, 0).
//
// 0x552340 / 0x552070 notices (byte 0x68510c set; 0x552340 only for a zero
// amount): monster target 0x590b40(record, message 435); character target
// 0x590950(id, message 435), which prints kind 8 through 0x58aad0.
// 0x52bb60 and 0x52c070 run natively, modelled as in n_54ec80.go (report block 0
// and argument 4 = 0 here, so the shield notice is always message 0x193).
//
// Summary without notify: hits in slot 0 == 0 (u32(2*hits)) prints message 421 via
// 0x5905f0. Otherwise per slot with hits: slot 1 first prints "," (0x61a3c4); t1
// formats "%ld %s" (0x619778) with hits and the status name (message u16
// 0x61e574+8*sub), t2..t5 format messages 449/450/451/409 with hits and u32
// total/hits; 0x517a90 writes into 0x689fd0 (vswprintf, wide) and 0x5905f0 prints
// it; 0x5905c0(1, -1) sets the pending byte 0x69b7b8.
//
// Stubs (re/emu/cases/005560a0.json): D4 callees 0x551ba0 (sequenced return
// resisted_k), 0x52b7e0, 0x52b6d0, 0x52af50, 0x52a890, 0x556b10, 0x52beb0, 0x523c00
// and engine callees 0x4e4280, 0x4e46f0 are address-stubbed (their other returns are
// ignored: 0x5a5a5a5a check); 0x5905f0, 0x590b40, 0x58aad0, 0x48f650, 0x4e4db0,
// 0x4e3af0 are excluded presentation callees. Emu replays the stubs. F runs
// statusImmunityAndResistNode for 0x551ba0 on the F-only world f.* (as n_54dbe0.go)
// and monsterConditionApplyNode for 0x523c00 (as n_54ec80.go). The character
// callees and 0x556b10 write nothing this node reads or outputs and stay recorded
// calls in F too.
//
// Fixture caps (panic past them): two targets per list, party indices 0..7,
// monster ids 100..101, templates 0..2, message ids < 1024, stub call counts.
// srAssertFail paths and the unreachable blocks of
// notes/re/cov-exempt.d/5560a0.tsv panic.
func specialAbilityEffectNode(in []int64, r Rand) []int64 { return saeRun(in, r, true) }

func saeEmu(in []int64, r Rand) []int64 { return saeRun(in, r, false) }

const (
	saeArena    = 0x20000000 // case object src at arena +0
	saeMonster  = saeArena + 0x28000
	saeMonStep  = 0x800
	saeText     = saeArena + 0x20000 // fixture message k at +32k
	saeTextStep = 32
	saeTextCap  = 1024
	saeTypes    = 0x61eefc
	saeSubs     = 0x61effc
	saeRealms   = 0x61f0fc
	saeTextIDs  = 0x61e574
	saeComma    = 0x61a3c4
	saeFmt      = 0x619778 // "%ld %s"
	saeBuffer   = 0x689fd0
	saeBufWords = 24
	saeNotice   = 435 // 0x552340 / 0x552070 notice (0x68c09c +0x6cc)
	saeNoHits   = 421 // +0x694
)

var saeSummaryText = [6]uint32{0, 0, 449, 450, 451, 409} // +0x704, +0x708, +0x70c, +0x664 per type

var saeStubs = []struct {
	addr  string
	args  []int
	sizes []int
	n     int
}{
	{"551ba0", []int{3, 4, 5, 6, 7, 8, 9, 10, 26, 27, 28}, []int{4, 4, 4, 4, 4, 1, 1, 4, 4, 4, 4}, 8},
	{"52b7e0", []int{1, 2, 3}, []int{4, 4, 4}, 2},
	{"52b6d0", []int{1, 2, 3, 4}, []int{4, 4, 4, 4}, 12},
	{"52af50", []int{1, 2, 3, 4}, []int{4, 4, 4, 4}, 2},
	{"52a890", []int{1, 2, 3, 4, 5, 6, 7}, []int{4, 4, 4, 1, 4, 4, 4}, 2},
	{"556b10", []int{1, 2, 3}, []int{4, 4, 4}, 1},
	{"5905f0", []int{1, 2}, []int{4, 4}, 3},
	{"52beb0", []int{1, 2, 3}, []int{4, 4, 4}, 4},
	{"4e4280", []int{1}, []int{4}, 2},
	{"4e46f0", []int{1, 2}, []int{4, 1}, 2},
	{"523c00", []int{1, 2, 3, 4, 6}, []int{4, 4, 4, 4, 4}, 4},
	{"590b40", []int{1, 2}, []int{4, 4}, 8},
	{"58aad0", []int{1}, []int{4}, 12},
	{"48f650", []int{1}, []int{4}, 4},
	{"4e4db0", []int{1, 2}, []int{4, 4}, 2},
	{"4e3af0", []int{1}, []int{4}, 2},
}

func saeNames() (in, out []string) {
	a := func(l *[]string, f string, x ...any) { *l = append(*l, fmt.Sprintf(f, x...)) }
	in = append(in, "ability", "groupArgument", "notify", "characterCount", "characterTarget0", "characterTarget1",
		"monsterCount", "monsterTarget0", "monsterTarget1")
	in = append(in, sdresSourceNames...)
	in = append(in, "messagePending", "allMonstersVisible")
	for k := 0; k < 8; k++ {
		for _, f := range []string{"slotUsed", "level", "maxStamina", "stamina", "fireResistance"} {
			a(&in, "%s%d", f, k)
		}
		for _, n := range sdresRealmNames {
			a(&in, "%sResistance%d", n, k)
		}
	}
	for t := 0; t < 3; t++ {
		for j := 0; j < 6; j++ {
			a(&in, "templateResistance%dRealm%d", t, j)
		}
		a(&in, "templateLevel%d", t)
		a(&in, "templateKind%d", t)
		for k := 0; k < 5; k++ {
			a(&in, "templateAttribute%dIndex%d", t, k)
		}
	}
	for i := 0; i < 2; i++ {
		for _, f := range []string{"monsterTemplate", "monsterHp", "monsterMaxHp", "monsterMaxStamina", "monsterStamina",
			"monsterFatigue", "monsterShieldTimer", "monsterShieldPower", "monsterShield", "monsterShieldDuration",
			"monsterExhaustionDuration", "monsterCreditFlag", "monsterVisible", "monsterSp"} {
			a(&in, "%s%d", f, i)
		}
		for j := 0; j < 6; j++ {
			a(&in, "monsterResistanceModifier%dRealm%d", i, j)
		}
		for k := 0; k < 8; k++ {
			a(&in, "monsterCredit%dCharacter%d", i, k)
		}
		a(&in, "monsterVisualState%d", i)
		a(&in, "monsterVisualNext%d", i)
		for _, s := range sdresStatus {
			a(&in, "monsterStatus%dIndex%d", i, s)
		}
		a(&in, "monsterPoison%d", i)
		for b := 5; b < 8; b++ {
			for _, f := range []string{"Power", "Multiplier", "Duration"} {
				a(&in, "monsterBuff%dSlot%d%s", i, b, f)
			}
		}
		for e := 0; e < 12; e++ {
			for _, f := range []string{"Active", "Type", "Power", "Multiplier"} {
				a(&in, "monsterEffect%dRecord%d%s", i, e, f)
			}
		}
		for k := 0; k < 9; k++ {
			for _, f := range []string{"Active", "Code", "Value"} {
				a(&in, "monsterCombatSpell%dRecord%d%s", i, k, f)
			}
		}
		for k := 0; k < 6; k++ {
			for _, f := range []string{"Active", "Code", "Power", "Multiplier"} {
				a(&in, "monsterAura%dRecord%d%s", i, k, f)
			}
		}
	}
	for k := 0; k < 8; k++ {
		a(&in, "resisted%d", k)
	}
	for _, n := range sirIn[sirSlot:] {
		in = append(in, "f."+n)
	}

	out = append(out, "messagePending")
	for w := 0; w < saeBufWords; w++ {
		a(&out, "summaryText%d", w)
	}
	for i := 0; i < 2; i++ {
		for _, f := range []string{"monsterHp", "monsterStamina", "monsterFatigue", "monsterShieldTimer",
			"monsterShieldPower", "monsterShield", "monsterShieldDuration", "monsterSp"} {
			a(&out, "%s%d", f, i)
		}
		for k := 0; k < 8; k++ {
			a(&out, "monsterCredit%dCharacter%d", i, k)
		}
		for w := 0; w < 13; w++ {
			a(&out, "monsterLastAttacker%dWord%d", i, w)
		}
		for k := 0; k < 103; k++ {
			a(&out, "monsterModifier%dByte%d", i, k)
		}
		for k := 0; k < 5; k++ {
			a(&out, "monsterAttribute%dIndex%d", i, k)
		}
		a(&out, "monsterHpRecovery%d", i)
		a(&out, "monsterSpRecovery%d", i)
	}
	for _, s := range saeStubs {
		a(&out, "n%s", s.addr)
		for _, g := range s.args {
			for k := 0; k < s.n; k++ {
				if s.n > 1 {
					a(&out, "a%s_%d_%d", s.addr, g, k)
				} else {
					a(&out, "a%s_%d", s.addr, g)
				}
			}
		}
	}
	return
}

var saeIn, saeOut = saeNames()

var saeInIdx, saeOutIdx = func() (map[string]int, map[string]int) {
	mi, mo := map[string]int{}, map[string]int{}
	for i, n := range saeIn {
		mi[n] = i
	}
	for i, n := range saeOut {
		mo[n] = i
	}
	return mi, mo
}()

func init() {
	Nodes["005560a0"] = Node{Addr: "005560a0", In: saeIn, Out: saeOut, F: specialAbilityEffectNode, Emu: saeEmu}
}

// saeFixtureText is the case's message k (fixture strings, not game text).
func saeFixtureText(k uint32) []uint16 {
	switch k {
	case saeNotice:
		return []uint16{':'}
	case 409, 449, 450, 451:
		return []uint16{'%', 'l', 'd', uint16(0x4000 + k), '%', 'l', 'd'}
	}
	return []uint16{uint16(0x4000 + k)}
}

func saeDecimal(v uint32) []uint16 {
	var w []uint16
	for _, c := range strconv.FormatInt(int64(int32(v)), 10) { // %ld
		w = append(w, uint16(c))
	}
	return w
}

func saeRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(saeIn) {
		panic("special ability effect: wrong input count")
	}
	g := func(n string) int64 {
		i, ok := saeInIdx[n]
		if !ok {
			panic("special ability effect: no input " + n)
		}
		return in[i]
	}
	u := func(n string) uint32 { return uint32(g(n)) }
	gf := func(f string, x ...any) int64 { return g(fmt.Sprintf(f, x...)) }
	uf := func(f string, x ...any) uint32 { return uint32(gf(f, x...)) }
	out := make([]int64, len(saeOut))
	set := func(n string, v int64) {
		i, ok := saeOutIdx[n]
		if !ok {
			panic("special ability effect: no output " + n)
		}
		out[i] = v
	}
	calls := map[string]int{}
	stub := func(addr string, args ...int64) int {
		for _, s := range saeStubs {
			if s.addr != addr {
				continue
			}
			k := calls[addr]
			if k >= s.n {
				panic(fmt.Sprintf("special ability effect: 0x%s called more than %d times (fixture record cap)", addr, s.n))
			}
			calls[addr] = k + 1
			for j, arg := range s.args {
				v := args[arg-1] & (1<<(8*uint(s.sizes[j])) - 1)
				if s.n > 1 {
					set(fmt.Sprintf("a%s_%d_%d", addr, arg, k), v)
				} else {
					set(fmt.Sprintf("a%s_%d", addr, arg), v)
				}
			}
			return k
		}
		panic("special ability effect: unknown stub " + addr)
	}
	msgPtr := func(k uint32) int64 {
		if k >= saeTextCap {
			panic("special ability effect: message id past the fixture's 1024 messages")
		}
		return saeArena + 0x20000 + saeTextStep*int64(k)
	}

	ability := u("ability")
	if ability > 31 {
		panic("special ability effect: ability past the 32-entry exe tables (0x61eefc)")
	}
	typ := func(s uint32) uint32 { return uint32(gsExeDword(saeTypes + 8*ability + 4*s)) }
	sub := func(s uint32) uint32 { return uint32(gsExeDword(saeSubs + 8*ability + 4*s)) }
	realm := gsExeDword(saeRealms + 4*ability)
	notify := u("notify")&0xff != 0
	nflag := int64(0)
	if notify {
		nflag = 1
	}
	srcType, srcChar, srcMon := u("sourceType"), u("sourceCharacter"), u("sourceMonster")
	f1b, f1c, f1e := u("sourceFlag1b")&0xff != 0, u("sourceFlag1c")&0xff != 0, u("sourceFlag1e")&0xff != 0

	var level, maxSt, st, fire [8]uint32
	var res [8][5]uint32
	for k := 0; k < 8; k++ {
		level[k], maxSt[k], st[k], fire[k] = uf("level%d", k), uf("maxStamina%d", k), uf("stamina%d", k), uf("fireResistance%d", k)
		for j, n := range sdresRealmNames {
			res[k][j] = uf("%sResistance%d", n, k)
		}
	}
	var mon [2]sdresMon
	for i := range mon {
		m := &mon[i]
		m.tpl, m.hp, m.maxHp = uf("monsterTemplate%d", i), uf("monsterHp%d", i), uf("monsterMaxHp%d", i)
		m.maxSt, m.st, m.fat = uf("monsterMaxStamina%d", i), uf("monsterStamina%d", i), uf("monsterFatigue%d", i)
		m.shTimer, m.shPower, m.shield = uf("monsterShieldTimer%d", i), uf("monsterShieldPower%d", i), uf("monsterShield%d", i)
		m.shDur, m.exh = uf("monsterShieldDuration%d", i), uf("monsterExhaustionDuration%d", i)
		m.creditFlag, m.visible, m.sp = uf("monsterCreditFlag%d", i)&0xff, uf("monsterVisible%d", i)&0xff, uf("monsterSp%d", i)
		for j := 0; j < 6; j++ {
			m.mods[j] = gf("monsterResistanceModifier%dRealm%d", i, j)
			m.mod[0x3c+j] = uint8(m.mods[j])
		}
		for k := 0; k < 8; k++ {
			m.credit[k] = uf("monsterCredit%dCharacter%d", i, k)
		}
		m.visState, m.visNext = int8(gf("monsterVisualState%d", i)), uint8(gf("monsterVisualNext%d", i))
		for _, s := range sdresStatus {
			m.status[s] = uf("monsterStatus%dIndex%d", i, s)
		}
		m.status[17], m.poison = m.exh, uf("monsterPoison%d", i)
		if m.tpl > 2 {
			panic("special ability effect: monster template outside the three fixtures")
		}
		for k := range m.tplAttr {
			m.tplAttr[k] = gf("templateAttribute%dIndex%d", m.tpl, k)
		}
		for b := 5; b < 8; b++ {
			for f, n := range []string{"Power", "Multiplier", "Duration"} {
				m.buff[b][f] = gf("monsterBuff%dSlot%d%s", i, b, n)
			}
		}
		for e := range m.eff {
			for f, n := range []string{"Active", "Type", "Power", "Multiplier"} {
				m.eff[e][f] = gf("monsterEffect%dRecord%d%s", i, e, n)
			}
		}
		for k := range m.combatSpell {
			for f, n := range []string{"Active", "Code", "Value"} {
				m.combatSpell[k][f] = uf("monsterCombatSpell%dRecord%d%s", i, k, n)
			}
		}
		for k := range m.aura {
			for f, n := range []string{"Active", "Code", "Power", "Multiplier"} {
				m.aura[k][f] = uf("monsterAura%dRecord%d%s", i, k, n)
			}
		}
		for w := range m.last {
			m.last[w] = sdresDecoy + 0x100*uint32(i) + uint32(w)
		}
	}
	allVisible := u("allMonstersVisible")&0xff != 0
	pending := u("messagePending") & 0xff
	monOf := func(id uint32) (*sdresMon, int, int64) {
		if id < 100 || id > 101 {
			panic("special ability effect: monster id outside the fixtures 100..101")
		}
		j := int(id - 100)
		return &mon[j], j, int64(saeMonster + saeMonStep*j)
	}

	// Source power and dice (unset for other source types).
	var power uint32
	var dice [2]uint32
	powered := true
	scale := func(l, cur, max uint32) {
		s := l + min(l, 15)
		if max == 0 {
			panic("special ability effect: maximum stamina 0 divides by zero")
		}
		power, dice[0] = max1((cur*s)/max), max1((cur*(s+5))/max)
		dice[1] = dice[0]
	}
	switch srcType {
	case 1:
		if srcChar > 7 {
			panic("special ability effect: source party index outside 0..7")
		}
		scale(level[srcChar], st[srcChar], maxSt[srcChar])
	case 2:
		m, _, _ := monOf(srcMon)
		scale(uint32(gf("templateLevel%d", m.tpl)), m.st, m.maxSt)
		if gsExeDword(saeTypes) == 5 || gsExeDword(saeTypes+8) == 5 {
			panic("special ability effect: exe type table entry 5 doubles the dice (cov-exempt 5561d1/5561e2)")
		}
	default:
		powered = false
	}

	if typ(0) == 6 {
		stub("556b10", int64(ability), saeArena, g("groupArgument"))
		return saeFinish(out, set, mon, pending, nil, calls)
	}

	// 0x552250 through resistanceQuantityReductionNode's 27-input form.
	resist := func(v uint32, kind int64, idx uint32, id uint32) uint32 {
		x := make([]int64, 27)
		x[0], x[3], x[4], x[5] = int64(v), int64(power), kind, realm
		if kind == 1 {
			x[1], x[2], x[6] = int64(fire[idx]), int64(level[idx]), int64(idx)
			for j := 0; j < 5; j++ {
				x[7+j] = int64(res[idx][j])
			}
		} else {
			m, _, _ := monOf(id)
			for j := 0; j < 6; j++ {
				x[12+j] = gf("templateResistance%dRealm%d", m.tpl, j)
				x[19+j] = m.mods[j]
			}
			x[18], x[25], x[26] = gf("templateLevel%d", m.tpl), int64(id), int64(m.tpl)
		}
		return uint32(resistanceQuantityReductionNode(x, r)[0])
	}
	// 0x552070's notice (also 0x552340's for a zero amount).
	notice := func(kind int64, id uint32) {
		if !notify {
			return
		}
		if kind == 3 {
			_, _, mp := monOf(id)
			stub("590b40", mp, msgPtr(saeNotice))
		} else {
			stub("58aad0", 8) // 0x590950
		}
	}
	reduce := func(v uint32, kind int64, idx, id uint32) uint32 {
		v = resist(v, kind, idx, id)
		if v == 0 {
			notice(kind, id)
		}
		return v
	}
	resisted := func(kind int64, idx, id uint32, s uint32, extra, roll uint32) bool {
		k := stub("551ba0", saeArena, 0, int64(sub(s)), realm, int64(power), int64(extra), int64(roll), nflag, nflag, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, kind, int64(idx), int64(id))
		if !real {
			return byte(gf("resisted%d", k)) != 0
		}
		v := make([]int64, len(sirIn))
		v[sirStatus], v[sirRealm], v[sirStrength], v[sirPenalty], v[sirDuration] = int64(sub(s)), realm, int64(power), int64(extra), int64(roll)
		v[sirMessage], v[sirDurMessage], v[sirDurFlagB] = nflag, nflag, 0
		v[sirSrcKind], v[sirSrcIndex] = int64(srcType), int64(int32(srcChar))
		v[sirKind], v[sirIdx], v[sirMonID] = kind, int64(int32(idx)), int64(id)
		if kind == 1 {
			v[sirMonID] = 0 // 0x551ba0 reads +8 only for monster targets
		} else {
			v[sirIdx] = 0 // and +4 only for character targets (the record holds -1)
		}
		copy(v[sirSlot:], in[saeInIdx["f."+sirIn[sirSlot]]:])
		return byte(statusImmunityAndResistNode(v, r)[0]) != 0
	}
	var hits, total [2]uint32
	need := func() {
		if !powered {
			panic("special ability effect: source type other than 1/2 uses unset power and dice")
		}
	}
	sum3 := func(d uint32) uint32 {
		x := uint32(r.R(int(d)))
		x += uint32(r.R(int(d)))
		return x + uint32(r.R(int(d)))
	}
	count := func(s int, v uint32) {
		if !notify {
			total[s] += v
			hits[s]++
		}
	}

	charCount := int32(g("characterCount"))
	if charCount > 2 {
		panic("special ability effect: more than two character targets (fixture cap)")
	}
	for i := 0; i < int(charCount); i++ {
		id := uf("characterTarget%d", i)
		if id == 0xffffffff {
			panic("special ability effect: character id -1 is srAssertFail (cov-exempt 556253)")
		}
		if id > 7 {
			panic("special ability effect: character index outside party slots 0..7")
		}
		for s := uint32(0); s < 2; s++ {
			if s == 1 && r.R(100) < 50 || s == 0 && srcType == 2 && r.R(100) < 10 {
				continue
			}
			t := typ(s)
			if t-1 > 4 {
				continue
			}
			need()
			d := dice[s]
			var v uint32
			switch t {
			case 1:
				roll := uint32(r.R(int(d)))
				extra := uint32(0)
				if sub(s) == 7 {
					extra = uint32(r.R(int(d))) + 1
				}
				if !resisted(1, id, 0xffffffff, s, extra, roll) && !notify {
					hits[s]++
				}
				continue
			case 2, 3, 5:
				v = sum3(d)/3 + 1
			case 4:
				v = sum3(d) + 1
			}
			if t != 5 || ability != 10 {
				v = reduce(v, 1, id, 0)
			}
			if v == 0 {
				notice(1, 0)
				continue
			}
			switch t {
			case 2:
				stub("52b7e0", int64(id), int64(v), 1)
			case 3:
				for k := 0; k < 6; k++ {
					stub("52b6d0", int64(id), realm, int64(v), 1)
				}
			case 4:
				stub("52af50", int64(id), int64(v), 0, 0)
			case 5:
				stub("52a890", int64(id), int64(v), 0, nflag, 0, 0, 0)
			}
			count(int(s), v)
		}
	}

	monCount := int32(g("monsterCount"))
	if monCount > 2 {
		panic("special ability effect: more than two monster targets (fixture cap)")
	}
	srcWords := func() [13]uint32 {
		var w [13]uint32
		for j := 0; j < 6; j++ {
			w[j] = u(sdresSourceNames[j])
		}
		w[6] = u("source74") | u("source76")<<16 | u("sourceFlag1b")<<24
		w[7] = u("sourceFlag1c") | u("source79")<<8 | u("sourceFlag1e")<<16 | u("source7b")<<24
		for j, n := range []string{"source7c", "source80", "source84", "source88", "source8c"} {
			w[8+j] = u(n)
		}
		return w
	}
	// 0x52c070(monster, amount, 0): stamina, fatigue and exhaustion.
	tire := func(m *sdresMon, id uint32, amt uint32) {
		if m.hp == 0 {
			return
		}
		if amt != 0 {
			if amt >= m.st {
				m.st = 0
			} else {
				m.st -= amt
			}
		}
		if m.maxSt == 0 {
			panic("special ability effect: maximum stamina 0 divides by zero (0x52c0b8)")
		}
		m.fat = staminaFatigueLevel(m.st, m.maxSt)
		if m.st != 0 || m.exh >= 0x270f {
			return
		}
		stub("523c00", int64(id), 0x11, 0x270f, 0, 0, 1)
		if !real {
			return
		}
		x := append([]int64(nil), sdrApplyParked...)
		setx := func(name string, v int64) {
			for i, n := range mcaSpecApplyIn {
				if n == name {
					x[i] = v
					return
				}
			}
			panic("special ability effect: parked input " + name + " missing")
		}
		for n, val := range map[string]int64{"cond": 0x11, "duration": 0x270f, "penalty": 0, "message": 1,
			"M.HP": int64(m.hp), "M.maxSP": int64(m.maxSt), "M.SP": int64(m.st), "M.cond17": int64(m.exh),
			"tpl.kind": gf("templateKind%d", m.tpl), "a5.type": 0, "a5.char": -1, "a5.monster": -1,
			"a5.f1b": 0, "a5.f1c": 0, "a5.f1e": 0} {
			setx(n, val)
		}
		o := monsterConditionApplyNode(x, r)
		var phased bool
		var fat uint32
		for i, n := range mcaOutNames {
			switch n {
			case "M.SP":
				m.st = uint32(o[i])
			case "M.cond17":
				m.exh = uint32(o[i])
			case "M.fatigue":
				fat = uint32(o[i])
			case "s52c240_calls":
				phased = o[i] != 0
			}
		}
		if phased {
			m.fat = fat
		}
	}
	// 0x52bb60(monster, amount, source, 0, print, 0, 0, 0) with no report block.
	damage := func(id uint32, amt uint32, print bool) {
		m, j, mp := monOf(id)
		if m.hp == 0 {
			return
		}
		if m.shDur != 0 {
			stub("590b40", mp, msgPtr(0x193)) // argument 4 (0 here) would pick 0x192
			if m.shield >= amt {
				m.shield = (m.shield - amt) & 0xffff
				return
			}
			amt -= m.shield
			m.shTimer, m.shPower, m.shield, m.shDur = 0, 0, 0, 0 // 0x524390(id, 2)
			sdresRebuild(m, j)
		}
		if allVisible || m.visible != 0 {
			stub("48f650", mp)
			if print {
				kind := int64(9)
				if srcType == 1 {
					kind = 8
				}
				stub("58aad0", kind)
			}
		}
		if !f1c && !f1b && !f1e {
			m.last = srcWords()
			if m.creditFlag != 0 && srcType == 1 {
				if srcChar > 7 {
					panic("special ability effect: credit index outside party slots 0..7")
				}
				m.credit[srcChar] += amt
			}
		}
		if amt >= m.hp {
			m.hp = 0
			if m.visState != 0x15 && m.visNext != 0x15 { // 0x4e4690(monster, 1)
				stub("4e4db0", mp, 0x15)
				stub("4e4280", mp)
				stub("4e46f0", mp, 1)
				stub("4e3af0", int64(j))
			}
			return
		}
		m.hp -= amt
		tire(m, id, (2*amt)/3)
		stub("52beb0", mp, saeArena, 0)
	}
	for i := 0; i < int(monCount); i++ {
		id := uf("monsterTarget%d", i)
		if id == 0xffffffff {
			panic("special ability effect: monster id -1 is srAssertFail (cov-exempt 556604)")
		}
		m, _, _ := monOf(id)
		for s := uint32(0); s < 2; s++ {
			if s == 1 && r.R(100) < 50 || s == 0 && r.R(100) < 10 {
				continue
			}
			t := typ(s)
			if t-1 > 4 {
				continue
			}
			need()
			d := dice[s]
			var v uint32
			switch t {
			case 1:
				extra := uint32(0)
				if sub(s) == 7 {
					extra = uint32(r.R(2)) + 1
				}
				if !resisted(3, 0xffffffff, id, s, extra, max1(d/3)) && !notify {
					hits[s]++
				}
				continue
			case 2, 5:
				v = sum3(d)/3 + 1
			case 3:
				v = sum3(d)/6 + 1
			case 4:
				v = sum3(d)
			}
			if t != 5 || ability != 10 {
				v = reduce(v, 3, 0, id)
			}
			if v == 0 {
				notice(3, id)
				continue
			}
			switch t {
			case 2:
				damage(id, v, true)
			case 3:
				m.sp -= v >> 1
			case 4:
				tire(m, id, v)
			case 5:
				damage(id, v, notify)
			}
			count(int(s), v)
		}
	}

	var buf []uint16
	if !notify {
		if 2*hits[0] == 0 {
			stub("5905f0", msgPtr(saeNoHits), 0xffffffff)
		} else {
			buf = make([]uint16, saeBufWords)
			for s := uint32(0); s < 2; s++ {
				c := hits[s]
				if c == 0 {
					continue
				}
				if s == 1 {
					stub("5905f0", saeComma, 0xffffffff)
				}
				t := typ(s)
				if t-1 <= 4 {
					var text []uint16
					if t == 1 {
						id := uint32(gsExeWord(saeTextIDs + 8*sub(s)))
						msgPtr(id)
						// "%ld %s" (0x619778)
						text = append(append(saeDecimal(c), ' '), saeFixtureText(id)...)
					} else {
						text = saeFormat(saeFixtureText(saeSummaryText[t]), c, total[s]/c)
					}
					text = append(text, 0)
					if len(text) > saeBufWords {
						panic("special ability effect: summary text past the 24 observed words")
					}
					copy(buf, text)
					stub("5905f0", saeBuffer, 0xffffffff)
				}
				pending = 1 // 0x5905c0(1, -1)
			}
		}
	}
	return saeFinish(out, set, mon, pending, buf, calls)
}

func max1(v uint32) uint32 { return max(v, 1) }

// saeFormat expands the two %ld of a fixture summary format.
func saeFormat(f []uint16, a, b uint32) []uint16 {
	var o []uint16
	args := [][]uint16{saeDecimal(a), saeDecimal(b)}
	for i := 0; i < len(f); i++ {
		if i+2 < len(f) && f[i] == '%' && f[i+1] == 'l' && f[i+2] == 'd' {
			o = append(o, args[0]...)
			args = args[1:]
			i += 2
			continue
		}
		o = append(o, f[i])
	}
	return o
}

func saeFinish(out []int64, set func(string, int64), mon [2]sdresMon, pending uint32, buf []uint16, calls map[string]int) []int64 {
	set("messagePending", int64(pending))
	for w, v := range buf {
		set(fmt.Sprintf("summaryText%d", w), int64(v))
	}
	for i, m := range mon {
		for n, v := range map[string]uint32{"monsterHp": m.hp, "monsterStamina": m.st, "monsterFatigue": m.fat,
			"monsterShieldTimer": m.shTimer, "monsterShieldPower": m.shPower & 0xffff, "monsterShield": m.shield & 0xffff,
			"monsterShieldDuration": m.shDur, "monsterSp": m.sp} {
			set(fmt.Sprintf("%s%d", n, i), int64(v))
		}
		for k, v := range m.credit {
			set(fmt.Sprintf("monsterCredit%dCharacter%d", i, k), int64(v))
		}
		for k, v := range m.mod {
			set(fmt.Sprintf("monsterModifier%dByte%d", i, k), int64(v))
		}
		for k, v := range m.attr {
			set(fmt.Sprintf("monsterAttribute%dIndex%d", i, k), v)
		}
		set(fmt.Sprintf("monsterHpRecovery%d", i), m.hpRec)
		set(fmt.Sprintf("monsterSpRecovery%d", i), m.spRec)
		for w, v := range m.last {
			set(fmt.Sprintf("monsterLastAttacker%dWord%d", i, w), int64(v))
		}
	}
	for _, s := range saeStubs {
		set("n"+s.addr, int64(calls[s.addr]))
	}
	return out
}
