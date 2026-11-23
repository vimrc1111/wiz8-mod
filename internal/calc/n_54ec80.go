package calc

import "fmt"

// spellDrainResourcesNode 0x54ec80 spell_drain_resources(effect), build/wiz8.asm:454827.
//
// The spell's realm is the u32 at rules +0x133 + 447*spell (rules record *0x65be1c,
// spell id effect +0) and goes to 0x552250 as its realm argument; the power level
// effect +0xc8 is 0x552250's strength argument; the notify byte is 0x68510c. The source record
// effect +0x5c (type, party index, monster id, flags +0x1b/+0x1c/+0x1e) is checked
// by 0x53bea0(source, 1): type 1 with index -1, and type 3 with index != -1 and
// both +0x1b and +0x1c clear, are srAssertFail paths; calc panics.
//
// Characters (count effect +0xf4, i32; list +0xfc): roll the effect dice (0x517970
// on effect +0xb4); unless the roll is the 9999 sentinel add the percentage bonus
// effect +0xd4 (0x517f90: u32 product /100); reduce it by 0x552250 (target type 1);
// a zero result with notify set prints through 0x590950; cap at hit points +0xb11
// (unsigned); a non-zero amount goes to 0x52a890(who, amount, 0, notify, 0,
// notify ? 0 : effect+0x126, 0) and into the total.
//
// Monsters (count +0xe4; list +0xec of ids): the same roll and 0x552250 (type 3);
// a zero result with notify set prints through 0x590b40; cap at +0x2b; a non-zero
// amount runs 0x52bb60(monster, amount, source, 0, notify, 0, notify ? 0 :
// effect+0x126, 0), modelled here, and the capped amount joins the total.
// 0x52bb60: a dead monster is untouched; an active shield (+0xc7 != 0, pool +0xc5
// u16) absorbs the blow (0x590b40; a pool >= amount loses the amount, counts a hit
// in the report block +0x12a and returns; a smaller pool is spent and 0x524390
// (run natively) clears +0xbf, +0xc3, +0xc5 and +0xc7 and rebuilds the monster cache
// through 0x50e8c0, modelled by monsterCacheRebuild: modifier bytes +0x1db (the realm
// resistances among them), attributes +0x247, recoveries +0x47/+0x4f); the report block then adds the amount to
// +0x126 and a hit to +0x12a. A visible monster (byte 0x683f94 or +0x2ab) runs
// 0x48f650 and with notify prints kind 8 (type-1 source) or 9 through 0x58aad0.
// With source flags +0x1b/+0x1c/+0x1e clear the 13 source dwords are copied to
// +0x304, and a type-1 source with credit flag +0x15 earns the amount in the credit
// record (*(+0x10) +0x1a + 4*index). A lethal amount (>= +0x2b) zeroes +0x2b; with
// the report block a 0x6c-byte death record {3, name} is appended to its list
// (count +0x182, capacity +0x186 grown to count+1, data +0x18a); 0x4e4690(monster,
// notify) runs the death visuals 0x4e4db0, 0x4e4280, 0x4e46f0, 0x4e3af0 unless the
// render state byte +0xa4 (i8) or +0xa7 is already 0x15. Otherwise +0x2b falls by
// the amount, 0x52c070(monster, u32(2*amount)/3, report) lowers stamina +0x33,
// sets fatigue +0x242 (staminaFatigueLevel) and at stamina 0 with +0x9b < 9999
// calls 0x523c00(id, 0x11, 9999, 0, source, notify) and counts report +0x172;
// then 0x52beb0(monster, source, 0).
//
// Without notify 0x54e710 reports: hits +0x12a > 0 set the message flags +0x124 and
// +0x125 and the pending-message byte 0x69b7b8, as does any of the 18 counters
// +0x12e; every death record is popped (type 1: 0x590950, which prints kind 8 via
// 0x58aad0; type 3: kind 9; other types nothing).
//
// Afterwards a type-1 source feeds the caster: spell 0x54 picks 0x52c3b0's party
// member and, when present and its positive current SP sum (0x52a730, i32) is below
// its maximum SP sum (0x52a760), calls 0x52b910(member, total); spell 0x52 picks
// 0x52c350's member and, when present and +0xb11 < +0xb0d, calls 0x52add0(member,
// u32(7*total)/10, notify). A type-2 source (0x53bf10; monster id -1 is the
// srAssertFail 0x53bf21) drains into the monster: spell 0x54 adds total/2 to SP
// +0x2f9; spell 0x52 heals through 0x52bfd0 (alive, not full, non-zero heal: u32
// sum capped at +0x27; notify prints kind 9).
//
// Stubs (re/emu/cases/0054ec80.json): D4 formula callees 0x52a890, 0x52add0,
// 0x52b910, 0x52c350, 0x52c3b0, 0x52beb0, 0x523c00 and the engine callees 0x4e4280,
// 0x4e46f0 are address-stubbed (0x4e46f0's second argument is a byte, recorded with
// size 1); 0x52c350/0x52c3b0 return case inputs, the others'
// returns are ignored (0x5a5a5a5a check). 0x590b40, 0x58aad0, 0x48f650, 0x4e4db0 and
// 0x4e3af0 are excluded presentation callees. Emu replays the stubs. F runs
// characterDamageApplyNode for 0x52a890 (feeding +0xb11, the absorb pool and the
// report block forward, and appending {1, who} death records),
// lowestHpRatioTargetNode/lowestSpRatioTargetNode on the post-damage party, and
// monsterConditionApplyNode for 0x523c00 (stamina, +0x9b and fatigue forward).
// 0x52beb0, 0x52add0, 0x52b910, 0x4e4280 and 0x4e46f0 write nothing this node reads
// or outputs and stay recorded calls in F too.
//
// Fixture caps (panic past them): spell realms 0..5 (0x552250 reads six realm
// fields), two targets per list, party indices 0..7, monster
// ids 100..101, templates 0..2, four death records on entry, 12 recorded 0x58aad0
// calls. srAssertFail paths panic (notes/re/cov-exempt.d/54ec80.tsv).
func spellDrainResourcesNode(in []int64, r Rand) []int64 { return spellDrainResources(in, r, true) }

func spellDrainResourcesEmu(in []int64, r Rand) []int64 { return spellDrainResources(in, r, false) }

const (
	sdresArena   = 0x20000000
	sdresSource  = sdresArena + 0x5c  // effect +0x5c (case object effect at arena +0)
	sdresReport  = sdresArena + 0x126 // effect +0x126
	sdresMonster = sdresArena + 0x20000
	sdresMonStep = 0x800
	sdresDecoy   = 0xd0d00000 // case setup of monster +0x304 words
)

var sdresSourceNames = []string{"sourceType", "sourceCharacter", "sourceMonster", "source68", "source6c", "source70",
	"source74", "source76", "sourceFlag1b", "sourceFlag1c", "source79", "sourceFlag1e", "source7b",
	"source7c", "source80", "source84", "source88", "source8c"}

var sdresRealmNames = []string{"water", "air", "earth", "mental", "divine"}

// sdresStubs: address, recorded args with their byte size, recorded calls.
var sdresStubs = []struct {
	addr  string
	args  []int
	sizes []int
	n     int
}{
	{"52a890", []int{1, 2, 4, 6}, []int{4, 4, 1, 4}, 2},
	{"52add0", []int{1, 2, 3}, []int{4, 4, 1}, 1},
	{"52b910", []int{1, 2}, []int{4, 4}, 1},
	{"52c350", nil, nil, 1},
	{"52c3b0", nil, nil, 1},
	{"52beb0", []int{1, 2, 3}, []int{4, 4, 4}, 2},
	{"4e4280", []int{1}, []int{4}, 2},
	{"4e46f0", []int{1, 2}, []int{4, 1}, 2},
	{"523c00", []int{1, 2, 3, 4, 6}, []int{4, 4, 4, 4, 4}, 2},
	{"590b40", []int{1}, []int{4}, 2},
	{"58aad0", []int{1}, []int{4}, 12},
	{"48f650", []int{1}, []int{4}, 2},
	{"4e4db0", []int{1, 2}, []int{4, 4}, 2},
	{"4e3af0", []int{1}, []int{4}, 2},
}

// sdresStatus are the monster status durations (M+0x57+4i) the 0x50e8c0 rebuild reads;
// 17 is monsterExhaustionDuration, 19 stays zero (0x50eac0 would dereference a null character).
var sdresStatus = []int{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 18}

// sdresCdmFixed are the 0x52a890 inputs this node supplies; the rest are F-only w_ inputs.
var sdresCdmFixed = map[string]bool{"who": true, "dmg": true, "a3": true, "a4": true, "a5": true, "kill": true,
	"a7": true, "pres": true, "hp": true, "maxHp": true, "b01": true, "kl0": true, "kl4": true, "kl4c": true,
	"kl5c": true, "kl60": true, "kl64": true, "mallocRet": true, "newRet": true}

func sdresNames() (in, out []string) {
	a := func(l *[]string, f string, x ...any) { *l = append(*l, fmt.Sprintf(f, x...)) }
	in = append(in, "spellId", "spellRealm", "powerLevel", "Z", "N", "S", "bonusPercent", "notify", "characterCount",
		"characterTarget0", "characterTarget1", "monsterCount", "monsterTarget0", "monsterTarget1")
	in = append(in, sdresSourceNames...)
	in = append(in, "effectMessageFlag", "effectResultFlag", "killDamage", "killHits")
	for k := 0; k < 18; k++ {
		a(&in, "reportCounter%d", k)
	}
	in = append(in, "deathCount", "deathCapacity")
	for k := 0; k < 4; k++ {
		a(&in, "deathType%d", k)
	}
	for k := 0; k < 4; k++ {
		a(&in, "deathCharacter%d", k)
	}
	in = append(in, "messagePending", "allMonstersVisible")
	for k := 0; k < 8; k++ {
		for _, f := range []string{"slotUsed", "mainStatus", "hp", "maxHp", "fireResistance", "level"} {
			a(&in, "%s%d", f, k)
		}
		for _, n := range sdresRealmNames {
			a(&in, "%sResistance%d", n, k)
		}
		for j := 0; j < 6; j++ {
			a(&in, "maxSp%dRealm%d", k, j)
		}
		for j := 0; j < 6; j++ {
			a(&in, "currentSp%dRealm%d", k, j)
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
	in = append(in, "stubLowestHp", "stubLowestSp")
	cdmIn, _ := cdmNames()
	for _, n := range cdmIn {
		if !sdresCdmFixed[n] && !(len(n) > 2 && n[:2] == "kd") {
			in = append(in, "w_"+n)
		}
	}

	out = append(out, "effectMessageFlag", "effectResultFlag", "killDamage", "killHits", "reportCounter17",
		"deathCount", "deathCapacity", "messagePending")
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
	for _, s := range sdresStubs {
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

var sdresInIdx, sdresOutIdx = func() (map[string]int, map[string]int) {
	in, out := sdresNames()
	mi, mo := map[string]int{}, map[string]int{}
	for i, n := range in {
		mi[n] = i
	}
	for i, n := range out {
		mo[n] = i
	}
	return mi, mo
}()

var sdresCdmIdx, sdresCdmOutIdx = func() (map[string]int, map[string]int) {
	in, out := cdmNames()
	mi, mo := map[string]int{}, map[string]int{}
	for i, n := range in {
		mi[n] = i
	}
	for i, n := range out {
		mo[n] = i
	}
	return mi, mo
}()

func init() {
	in, out := sdresNames()
	Nodes["0054ec80"] = Node{Addr: "0054ec80", In: in, Out: out, F: spellDrainResourcesNode, Emu: spellDrainResourcesEmu}
}

type sdresMon struct {
	tpl, hp, maxHp, maxSt, st, fat, shTimer, shPower, shield, shDur, exh, creditFlag, visible, sp uint32
	mods                                                                                          [6]int64
	credit                                                                                        [8]uint32
	visState                                                                                      int8
	visNext                                                                                       uint8
	last                                                                                          [13]uint32
	// 0x50e8c0 operands and results.
	status       [20]uint32
	poison       uint32
	buff         [8][3]int64
	eff          [12][4]int64
	combatSpell  [9][3]uint32
	aura         [6][4]uint32
	mod          [103]uint8
	attr         [5]int64
	tplAttr      [5]int64
	hpRec, spRec int64
}

// sdresRebuild is 0x524390's tail 0x50e8c0(id): the monster cache rebuild on the
// fields after slot 2 was cleared (slot 2 is not a rebuild operand).
func sdresRebuild(m *sdresMon, j int) {
	// monsterCacheRebuild reads a monster record: lay the fields out in a scratch
	// byte memory (record at 0, credit object at sdresRbCredit, template at sdresRbTpl).
	mem := map[uint32]byte{}
	put := func(a uint32, n int, v uint64) {
		for k := 0; k < n; k++ {
			mem[a+uint32(k)] = byte(v >> (8 * k))
		}
	}
	u8 := func(a uint32) byte { return mem[a] }
	u32 := func(a uint32) uint32 {
		return uint32(mem[a]) | uint32(mem[a+1])<<8 | uint32(mem[a+2])<<16 | uint32(mem[a+3])<<24
	}
	m.status[17] = m.exh
	put(0x27, 4, uint64(m.maxHp))
	put(0x2f, 4, uint64(m.maxSt))
	put(0x10b, 4, uint64(m.poison))
	for i, v := range m.tplAttr {
		put(sdresRbTpl+0xd1+uint32(i), 1, uint64(v))
	}
	for i, v := range m.status {
		put(0x57+4*uint32(i), 4, uint64(v))
	}
	for i, r := range m.buff {
		b := 0xa7 + 12*uint32(i)
		put(b, 1, uint64(r[0]))
		put(b+4, 2, uint64(r[1]))
		put(b+8, 4, uint64(r[2]))
	}
	for i, r := range m.eff {
		e := 0x10f + 0x11*uint32(i)
		put(e, 1, uint64(r[0]))
		put(e+1, 4, uint64(r[1]))
		put(e+5, 1, uint64(r[2]))
		put(e+9, 4, uint64(r[3]))
	}
	if m.creditFlag != 0 {
		put(0x15, 1, 1)
	}
	put(0x10, 4, sdresRbCredit)
	for i, r := range m.combatSpell {
		c := sdresRbCredit + 0x3e + 0x11*uint32(i)
		put(c, 1, uint64(r[0]))
		put(c+1, 4, uint64(r[1]))
		put(c+5, 1, uint64(r[2]))
	}
	for i, r := range m.aura {
		c := sdresRbCredit + 0xd7 + 0x11*uint32(i)
		put(c, 1, uint64(r[0]))
		put(c+1, 4, uint64(r[1]))
		put(c+5, 1, uint64(r[2]))
		put(c+9, 4, uint64(r[3]))
	}
	monsterCacheRebuild(u8, u32, put, 0, sdresRbTpl)
	for k := range m.mod {
		m.mod[k] = u8(0x1db + uint32(k))
	}
	for k := range m.attr {
		m.attr[k] = int64(u8(0x247 + uint32(k)))
	}
	m.hpRec, m.spRec = int64(u32(0x47)), int64(u32(0x4f))
	for k := range m.mods {
		m.mods[k] = int64(int8(m.mod[0x3c+k]))
	}
}

const sdresRbCredit, sdresRbTpl = 0x1000, 0x2000

type sdresDeath struct{ kind, who uint32 }

func spellDrainResources(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(sdresInIdx) {
		panic("spell drain resources: wrong input count")
	}
	g := func(n string) int64 {
		i, ok := sdresInIdx[n]
		if !ok {
			panic("spell drain resources: no input " + n)
		}
		return in[i]
	}
	u := func(n string) uint32 { return uint32(g(n)) }
	gf := func(f string, x ...any) int64 { return g(fmt.Sprintf(f, x...)) }
	uf := func(f string, x ...any) uint32 { return uint32(gf(f, x...)) }

	out := make([]int64, len(sdresOutIdx))
	set := func(n string, v int64) {
		i, ok := sdresOutIdx[n]
		if !ok {
			panic("spell drain resources: no output " + n)
		}
		out[i] = v
	}
	calls := map[string]int{}
	stub := func(addr string, args ...int64) {
		for _, s := range sdresStubs {
			if s.addr != addr {
				continue
			}
			k := calls[addr]
			if k >= s.n {
				panic(fmt.Sprintf("spell drain resources: 0x%s called more than %d times (fixture record cap)", addr, s.n))
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
			return
		}
		panic("spell drain resources: unknown stub " + addr)
	}

	spell := u("spellId")
	if spell > 255 {
		panic("spell drain resources: spell id outside the rules fixture 0..255")
	}
	spellRealm, power := g("spellRealm"), u("powerLevel")
	notify := u("notify")&0xff != 0
	srcType, srcChar, srcMon := u("sourceType"), u("sourceCharacter"), u("sourceMonster")
	f1b, f1c, f1e := u("sourceFlag1b") != 0, u("sourceFlag1c") != 0, u("sourceFlag1e") != 0
	// 0x53bea0(source, 1).
	if srcType == 1 && srcChar == 0xffffffff {
		panic("spell drain resources: type-1 source with index -1 is srAssertFail 0x53beb1")
	}
	if srcType == 3 && srcChar != 0xffffffff && !f1b && !f1c {
		panic("spell drain resources: type-3 source without flags is srAssertFail 0x53beee")
	}

	// Party state.
	var slot, status, hp, maxHp, fire, level [8]uint32
	var res [8][5]uint32
	var maxSp, curSp [8][6]uint32
	for k := 0; k < 8; k++ {
		slot[k], status[k], hp[k], maxHp[k] = uf("slotUsed%d", k)&0xff, uf("mainStatus%d", k), uf("hp%d", k), uf("maxHp%d", k)
		fire[k], level[k] = uf("fireResistance%d", k), uf("level%d", k)
		for j, n := range sdresRealmNames {
			res[k][j] = uf("%sResistance%d", n, k)
		}
		for j := 0; j < 6; j++ {
			maxSp[k][j], curSp[k][j] = uf("maxSp%dRealm%d", k, j), uf("currentSp%dRealm%d", k, j)
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
		}
		for k := 0; k < 8; k++ {
			m.credit[k] = uf("monsterCredit%dCharacter%d", i, k)
		}
		m.visState, m.visNext = int8(gf("monsterVisualState%d", i)), uint8(gf("monsterVisualNext%d", i))
		for j := 0; j < 6; j++ {
			m.mod[0x3c+j] = uint8(m.mods[j])
		}
		for _, s := range sdresStatus {
			m.status[s] = uf("monsterStatus%dIndex%d", i, s)
		}
		m.status[17], m.poison = m.exh, uf("monsterPoison%d", i)
		if m.tpl > 2 {
			panic("spell drain resources: monster template outside the three fixtures")
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

	// Report block effect +0x126.
	killDamage, killHits := u("killDamage"), u("killHits")
	var counters [18]uint32
	for k := range counters {
		counters[k] = uf("reportCounter%d", k)
	}
	msgFlag, resFlag, pending := u("effectMessageFlag")&0xff, u("effectResultFlag")&0xff, u("messagePending")&0xff
	deathCount, deathCap := int32(g("deathCount")), int32(g("deathCapacity"))
	if deathCount < 0 || deathCount > 4 {
		panic("spell drain resources: death list outside the four fixture records")
	}
	var deaths []sdresDeath
	for k := 0; k < int(deathCount); k++ {
		deaths = append(deaths, sdresDeath{uf("deathType%d", k), uf("deathCharacter%d", k)})
	}
	appendDeath := func(d sdresDeath) {
		if n := int32(len(deaths)) + 1; n > deathCap { // 0x52bdcf: grow to count+1
			deathCap = n
		}
		deaths = append(deaths, d)
	}

	// 0x517970 + 0x517f90.
	roll := func() uint32 {
		v := uint32(int32(diceNode([]int64{g("Z"), g("N"), g("S")}, r)[0]))
		if v != 9999 { // 0x54ed1f / 0x54eeb7
			v += (v * u("bonusPercent")) / 100
		}
		return v
	}
	// 0x552250 through resistanceQuantityReductionNode's 27-input form.
	resist := func(v uint32, ttype int64, idx int, tpl int64, mods [6]int64, id int64) uint32 {
		x := make([]int64, 27)
		x[0], x[3], x[4], x[5] = int64(v), int64(power), ttype, spellRealm
		if ttype == 1 {
			x[1], x[2], x[6] = int64(fire[idx]), int64(level[idx]), int64(idx)
			for j := 0; j < 5; j++ {
				x[7+j] = int64(res[idx][j])
			}
		} else {
			if tpl > 2 {
				panic("spell drain resources: monster template outside the three fixtures")
			}
			for j := 0; j < 6; j++ {
				x[12+j] = gf("templateResistance%dRealm%d", tpl, j)
				x[19+j] = mods[j]
			}
			x[18], x[25], x[26] = gf("templateLevel%d", tpl), id, tpl
		}
		return uint32(resistanceQuantityReductionNode(x, r)[0])
	}

	// F-side 0x52a890 world: absorb pool per member flows forward.
	a83 := map[uint32]int64{}
	total := uint32(0)
	report := int64(sdresReport)
	if notify {
		report = 0
	}

	charCount := int32(g("characterCount"))
	if charCount > 2 {
		panic("spell drain resources: more than two character targets (fixture cap)")
	}
	for i := 0; i < int(charCount); i++ {
		idx := uf("characterTarget%d", i)
		if idx == 0xffffffff {
			panic("spell drain resources: character index -1 is srAssertFail 0x54ecfc")
		}
		if idx > 7 {
			panic("spell drain resources: character index outside party slots 0..7")
		}
		v := resist(roll(), 1, int(idx), 0, [6]int64{}, 0)
		if v == 0 && notify {
			stub("58aad0", 8) // 0x590950 prints through 0x58aad0 kind 8
		}
		if v >= hp[idx] {
			v = hp[idx]
		}
		if v == 0 {
			continue
		}
		a4 := int64(0)
		if notify {
			a4 = 1
		}
		stub("52a890", int64(idx), int64(v), 0, a4, 0, report, 0)
		total += v
		if !real {
			continue
		}
		x := make([]int64, len(sdresCdmIdx))
		for n, p := range sdresCdmIdx {
			if j, ok := sdresInIdx["w_"+n]; ok {
				x[p] = in[j]
			}
		}
		if p, ok := a83[idx]; ok {
			x[sdresCdmIdx["a83"]] = p
		}
		kill := int64(0)
		if !notify {
			kill = cdmKl
		}
		for n, val := range map[string]int64{"who": int64(idx), "dmg": int64(v), "a3": 0, "a4": a4, "a5": 0, "kill": kill,
			"a7": 0, "pres": int64(slot[idx]), "hp": int64(hp[idx]), "maxHp": int64(maxHp[idx]), "b01": int64(status[idx]),
			"kl0": int64(killDamage), "kl4": int64(killHits), "kl4c": int64(counters[17]), "kl5c": int64(len(deaths)),
			"kl60": int64(deathCap), "kl64": cdmKlData, "mallocRet": cdmNewObj, "newRet": cdmNewBuf} {
			x[sdresCdmIdx[n]] = val
		}
		o := characterDamageApplyNode(x, r)
		c := func(n string) int64 { return o[sdresCdmOutIdx[n]] }
		hp[idx], a83[idx] = uint32(c("hp")), c("a83")
		if kill != 0 {
			killDamage, killHits, counters[17] = uint32(c("kl0")), uint32(c("kl4")), uint32(c("kl4c"))
			if int(c("kl5c")) > len(deaths) {
				deaths = append(deaths, sdresDeath{1, idx})
			}
			deathCap = int32(c("kl60"))
		}
	}

	monCount := int32(g("monsterCount"))
	if monCount > 2 {
		panic("spell drain resources: more than two monster targets (fixture cap)")
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
	for i := 0; i < int(monCount); i++ {
		id := uf("monsterTarget%d", i)
		if id < 100 || id > 101 {
			panic("spell drain resources: monster id outside the fixtures 100..101 (srAssertFail 0x54ee8f on a missing record)")
		}
		j := int(id - 100)
		m := &mon[j]
		mp := int64(sdresMonster + sdresMonStep*j)
		v := resist(roll(), 3, 0, int64(m.tpl), m.mods, int64(id))
		if v == 0 && notify {
			stub("590b40", mp)
		}
		if v >= m.hp {
			v = m.hp
		}
		if v == 0 {
			continue
		}
		total += v
		// 0x52bb60(monster, v, source, 0, notify, 0, report, 0).
		amt := v
		if m.shDur != 0 {
			stub("590b40", mp)
			if m.shield >= amt {
				m.shield = (m.shield - amt) & 0xffff
				if report != 0 {
					killHits++
				}
				continue
			}
			amt -= m.shield
			m.shTimer, m.shPower, m.shield, m.shDur = 0, 0, 0, 0 // 0x524390(id, 2)
			sdresRebuild(m, j)
		}
		if report != 0 {
			killDamage += amt
			killHits++
		}
		if allVisible || m.visible != 0 {
			stub("48f650", mp)
			if notify {
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
					panic("spell drain resources: credit index outside party slots 0..7")
				}
				m.credit[srcChar] += amt
			}
		}
		if amt >= m.hp {
			m.hp = 0
			if report != 0 {
				appendDeath(sdresDeath{3, 0})
			}
			if m.visState != 0x15 && m.visNext != 0x15 { // 0x4e4690
				flag := int64(0)
				if report == 0 {
					flag = 1
				}
				stub("4e4db0", mp, 0x15)
				stub("4e4280", mp)
				stub("4e46f0", mp, flag)
				stub("4e3af0", int64(j))
			}
			continue
		}
		m.hp -= amt
		// 0x52c070(monster, u32(2*amt)/3, report).
		if d := (2 * amt) / 3; d != 0 {
			if d >= m.st {
				m.st = 0
			} else {
				m.st -= d
			}
		}
		if m.maxSt == 0 {
			panic("spell drain resources: maximum stamina 0 divides by zero (0x52c0b8)")
		}
		m.fat = staminaFatigueLevel(m.st, m.maxSt)
		if m.st == 0 && m.exh < 0x270f {
			msg := int64(0)
			if report == 0 {
				msg = 1
			}
			stub("523c00", int64(id), 0x11, 0x270f, 0, 0, msg)
			if report != 0 {
				counters[17]++
			}
			if real {
				x := append([]int64(nil), sdrApplyParked...)
				setx := func(name string, v int64) {
					for i, n := range mcaSpecApplyIn {
						if n == name {
							x[i] = v
							return
						}
					}
					panic("spell drain resources: parked input " + name + " missing")
				}
				for n, val := range map[string]int64{"cond": 0x11, "duration": 0x270f, "penalty": 0, "message": msg,
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
		}
		stub("52beb0", mp, sdresSource, 0)
	}

	if !notify { // 0x54e710
		if killHits > 0 {
			pending, msgFlag, resFlag = 1, 1, 1
		}
		for _, c := range counters {
			if c > 0 {
				pending = 1
			}
		}
		for _, d := range deaths {
			switch d.kind {
			case 1:
				if d.who > 7 {
					panic("spell drain resources: death record member outside party slots 0..7")
				}
				stub("58aad0", 8) // 0x590950
			case 3:
				stub("58aad0", 9)
			}
		}
		deaths = deaths[:0]
	}

	switch srcType {
	case 1:
		switch spell {
		case 0x54:
			c := uint32(g("stubLowestSp"))
			if real {
				x := make([]int64, 0, 112)
				for k := 0; k < 8; k++ {
					x = append(x, int64(slot[k]), int64(status[k]))
					for j := 0; j < 6; j++ {
						x = append(x, int64(maxSp[k][j]))
					}
					for j := 0; j < 6; j++ {
						x = append(x, int64(int32(curSp[k][j])))
					}
				}
				c = uint32(lowestSpRatioTargetNode(x, r)[0])
			}
			stub("52c3b0")
			if c > 7 {
				panic("spell drain resources: 0x52c3b0 member outside party slots 0..7")
			}
			if slot[c] == 0 {
				break
			}
			var cur, max int32
			for j := 0; j < 6; j++ {
				if s := int32(curSp[c][j]); s > 0 { // 0x52a730
					cur += s
				}
				max += int32(maxSp[c][j]) // 0x52a760
			}
			if cur < max {
				stub("52b910", int64(c), int64(total))
			}
		case 0x52:
			c := uint32(g("stubLowestHp"))
			if real {
				x := make([]int64, 0, 32)
				for k := 0; k < 8; k++ {
					x = append(x, int64(slot[k]), int64(status[k]), int64(hp[k]), int64(maxHp[k]))
				}
				c = uint32(lowestHpRatioTargetNode(x, r)[0])
			}
			stub("52c350")
			if c > 7 {
				panic("spell drain resources: 0x52c350 member outside party slots 0..7")
			}
			if slot[c] != 0 && hp[c] < maxHp[c] {
				n := int64(0)
				if notify {
					n = 1
				}
				stub("52add0", int64(c), int64((7*total)/10), n)
			}
		}
	case 2:
		if srcMon == 0xffffffff {
			panic("spell drain resources: type-2 source with monster id -1 is srAssertFail 0x53bf21")
		}
		if srcMon < 100 || srcMon > 101 {
			panic("spell drain resources: source monster outside the fixtures 100..101 (srAssertFail 0x54f121 on a missing record)")
		}
		m := &mon[srcMon-100]
		switch spell {
		case 0x54:
			m.sp += total >> 1
		case 0x52: // 0x52bfd0
			heal := (7 * total) / 10
			if m.hp != 0 && m.hp != m.maxHp && heal != 0 {
				m.hp += heal
				if m.hp > m.maxHp {
					m.hp = m.maxHp
				}
				if notify {
					stub("58aad0", 9)
				}
			}
		}
	}

	set("effectMessageFlag", int64(msgFlag))
	set("effectResultFlag", int64(resFlag))
	set("killDamage", int64(killDamage))
	set("killHits", int64(killHits))
	set("reportCounter17", int64(counters[17]))
	set("deathCount", int64(len(deaths)))
	set("deathCapacity", int64(deathCap))
	set("messagePending", int64(pending))
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
	for _, s := range sdresStubs {
		set("n"+s.addr, int64(calls[s.addr]))
	}
	return out
}
