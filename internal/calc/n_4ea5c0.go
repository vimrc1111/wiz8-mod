package calc

import "fmt"

// ---- 0x4ea5c0: a party member's action preparation ----
//
// Runs when a character's turn is prepared. In combat it first takes the queued
// next action (unless the character already holds one), lets the status
// override (0x4ec1e0) cancel it, force a melee/ranged attack on a random target
// (0x51b0a0) or lift a condition (0x523330), then commits the action: melee
// swings (0x53d870), breath (0x4ebce0), turn undead (0x547cb0), guard
// (0x545aa0), prayer (0x547fe0), the spell loop (0x4fa4d0, repeated while the
// caster can afford another cast), item use (0x51dde0). An action that cannot
// start falls back to "defend" in combat or to none, and the fatigue cost of the
// action (0x52b2f0, 0x52af50) is charged at the end.
//
// It runs over the same memory image as the emulator case (mstWorld, built by
// gen from re/emu/cases/004ea5c0.json's description). Registered-out callees
// (0x5554a0, 0x555560, 0x5182e0, 0x562a50, 0x5471d0, 0x58aad0, 0x58b410) are
// address stubs whose records and counts are outputs. The D4 callees are address
// stubs too: Emu replays them from case inputs, F calls their Go nodes on their
// parked worlds (cpaPark*) fed with the values this world hands them and uses
// their return value and pointer-out value; their other side effects are not
// copied back (they live in the callee's own world).

const (
	cpaStrTable = 0x20006000 // fixture [0x68c09c]: string pointer table
	cpaStrCount = 0x520      // fixture cap: string ids 0..0x51f
	cpaText     = 0x20007800 // fixture wide texts for the ids in cpaTexts, 8 bytes each
	cpaSlots    = 3          // fixture cap: party slots 0..2
	cpaSpells   = 8          // fixture cap: spell records 0..7
	cpaL7c      = 0x7e000100 // 0x4ea5c0 stack local: the action cost (written by 0x547cb0 / 0x51dde0)
	cpaL6c      = 0x7e000104 // 0x4ea5c0 stack local: one cast's cost (written by 0x4fa4d0)
	cpaLocal81  = 0x10007f7f // 0x4ea5c0 stack byte read uninitialised outside combat (case input)
)

// cpaTexts are the string ids whose format text the fixture carries.
var cpaTexts = []uint32{0x3a0, 0x3a1, 0x3a2, 0x3a3, 0x3a4, 0x3a5, 0x3a6, 0x3a7, 0x3a8, 0x3a9, 0x3aa, 0x3ab, 0x3ac, 0x234}

func cpaStrings(s mstSpec) mstSpec {
	for i := uint32(0); i < cpaStrCount; i++ {
		v := mcaStrValue + i
		for k, t := range cpaTexts {
			if t == i {
				v = cpaText + 8*uint32(k)
			}
		}
		s.set = append(s.set, mstSet{cpaStrTable + 4*i, 4, v})
	}
	return s
}

type cpaWorld struct {
	*mcaWorld
	who, c, q uint32
}

func (p *cpaWorld) str(i uint32) uint32 {
	if i >= cpaStrCount {
		panic("character action prepare: string id past the fixture's 0x520")
	}
	return p.u32(p.u32(0x68c09c) + 4*i)
}

// d4 is D4 callee a: Emu replays its case stub, F runs f on park with live put
// in by name (names from the callee's registration) and returns its outputs.
// Live values follow the callee's field signedness (unsigned dwords as uint32).
func (p *cpaWorld) d4(a uint32, f func([]int64, Rand) []int64, park []int64, live map[string]int64, args ...uint32) (uint32, []int64) {
	k := p.call(a, args...)
	if !p.real {
		return p.replay(a, k, args...), nil
	}
	o := f(mprFeed(park, Nodes[fmt.Sprintf("%08x", a)].In, live), p.r)
	return uint32(o[0]), o
}

func (p *cpaWorld) eligible() bool {
	r, _ := p.d4(0x4e79a0, pcActionEligibilityNode, cpaPark4e79a0[:], map[string]int64{"characterSlot": int64(p.who)}, p.who, 0, 0, 0)
	return byte(r) != 0
}

func (p *cpaWorld) budget() {
	p.d4(0x53d680, attackBudgetPrepareNode, cpaPark53d680[:], map[string]int64{"who": int64(p.who)}, p.who)
}

func (p *cpaWorld) override() int32 {
	r, _ := p.d4(0x4ec1e0, statusActionOverrideNode, cpaPark4ec1e0[:], map[string]int64{"nodeType": 1, "partySlot": int64(p.who)})
	return int32(r)
}

func (p *cpaWorld) condRemove() {
	p.d4(0x523330, characterConditionRemoveNode, cpaPark523330[:], map[string]int64{"who": int64(p.who), "cond": 0xe, "notify": 1}, p.who, 0xe, 1)
}

func (p *cpaWorld) attackRange(h uint32) int32 {
	r, _ := p.d4(0x519ac0, charAttackRangeNode, cpaPark519ac0[:], map[string]int64{"hand": int64(h)}, p.c, h)
	return int32(r)
}

func (p *cpaWorld) target(l uint32) int32 {
	r, _ := p.d4(0x51b0a0, randomPartyAttackTargetNode, cpaPark51b0a0[:],
		map[string]int64{"attackerSlot": int64(p.who), "desiredDisposition": int64(l)}, p.who, l)
	return int32(r)
}

func (p *cpaWorld) allowed(a int32, spell uint32) bool {
	r, _ := p.d4(0x547310, offensiveActionAllowedNode, cpaPark547310[:],
		map[string]int64{"kind": int64(uint32(a)), "spellArg": int64(int32(spell)), "spellTypeAtArg": int64(int32(p.u32(p.q + 0x15)))},
		p.c, uint32(a), spell, p.q+0x15)
	return byte(r) != 0
}

func (p *cpaWorld) swings(mode int32) bool {
	r, _ := p.d4(0x53d870, attackSwingsNode, cpaPark53d870[:], map[string]int64{"who": int64(p.who), "mode": int64(mode)}, p.who, uint32(mode))
	return byte(r) != 0
}

func (p *cpaWorld) breath() bool {
	r, _ := p.d4(0x4ebce0, breathTargetPrepareNode, cpaPark4ebce0[:], map[string]int64{"characterIndex": int64(p.who)}, p.who)
	return byte(r) != 0
}

// turnUndead is 0x547cb0(who, &l7c, 1); its return value is ignored.
func (p *cpaWorld) turnUndead() {
	_, o := p.d4(0x547cb0, turnUndeadPowerNode, cpaPark547cb0[:], map[string]int64{"idx": int64(p.who), "mode": 1}, p.who, cpaL7c, 1)
	if o != nil {
		p.put(cpaL7c, 4, uint64(o[1]))
	}
}

func (p *cpaWorld) guard() bool {
	r, _ := p.d4(0x545aa0, guardAttackAvailableNode, cpaPark545aa0[:], map[string]int64{"characterSlot": int64(p.who)}, p.who)
	return byte(r) != 0
}

func (p *cpaWorld) prayer() uint32 {
	r, _ := p.d4(0x547fe0, battlePrayerNode, cpaPark547fe0[:], map[string]int64{"idx": int64(p.who)}, p.who)
	return r
}

// cast is 0x4fa4d0(who, spell, power, &l6c, 0).
func (p *cpaWorld) cast(spell, pl uint32) uint32 {
	r, o := p.d4(0x4fa4d0, characterSpellCastNode, cpaPark4fa4d0[:],
		map[string]int64{"idx": int64(p.who), "spell": int64(spell), "PL": int64(pl), "flag": 0}, p.who, spell, pl, cpaL6c, 0)
	if o != nil {
		p.put(cpaL6c, 4, uint64(o[1]))
	}
	return r
}

// itemUse is 0x51dde0(C, item, &l7c). Its world takes the character and item
// as pointers into its own arena, so F runs it on the parked world as is.
func (p *cpaWorld) itemUse() uint32 {
	r, o := p.d4(0x51dde0, itemUsePowerSelectionNode, cpaPark51dde0[:], map[string]int64{}, p.c, p.u32(p.q+0x19), cpaL7c)
	if o != nil {
		p.put(cpaL7c, 4, uint64(o[1]))
	}
	return r
}

func (p *cpaWorld) fatigueCost(a int32) uint32 {
	r, _ := p.d4(0x52b2f0, characterActionFatigueCostNode, cpaPark52b2f0[:], map[string]int64{"who": int64(p.who), "action": int64(uint32(a))}, p.who, uint32(a))
	return r
}

func (p *cpaWorld) fatigue(cost uint32) {
	p.d4(0x52af50, characterFatigueApplyNode, cpaPark52af50[:],
		map[string]int64{"who": int64(p.who), "amount": int64(int32(cost)), "scale": 1, "target": 0}, p.who, cost, 1, 0)
}

// msg is 0x590950(who, fmt, ...): the format's first two characters decide how
// many characters past the name the 0x58b410 colour span covers.
func (p *cpaWorld) msg(f uint32) {
	c0, c1 := p.u16(f), uint32(0)
	if c0 != 0 {
		c1 = p.u16(f + 2)
		if c0 == '%' || c1 == '%' {
			panic("character action prepare: a conversion in the first two format characters is outside the fixture")
		}
	}
	p.stub(0x58aad0, 8, 0x64bf5c, p.c+5)
	n := uint32(0)
	for p.u16(p.c+5+2*n) != 0 {
		n++
	}
	if c0 == 0x27 {
		n++
		if c1 == 0x73 {
			n++
		}
	}
	p.stub(0x58b410, uint32(p.u8(p.q+0xf1)), 0, n, 0xffffffff)
}

func (p *cpaWorld) setAction(a int32) { p.put(p.q+1, 4, uint64(uint32(a))) }

const (
	cpaT = iota // l82 decides
	cpaX        // the action could not start
	cpaM        // defend
	cpaE        // closing block
)

func (p *cpaWorld) prepare() {
	who, c, q := p.who, p.c, p.q
	p.put(cpaL7c, 4, 0xffffffff)
	l78 := int32(-1)
	l82 := true
	l81 := p.u8(cpaLocal81)
	p.stub(0x5554a0, who)
	combat := p.u8(0x683f94) != 0
	gm := p.u32(0x6836a8) + 0xd4*who
	if combat {
		if p.u32(0x6836a8) == 0 {
			panic("character action prepare: combat without a battle record")
		}
		l81 = p.u8(gm + 0x4c)
		if l81 == 0 {
			for _, o := range [][2]uint32{{1, 0x3d}, {5, 0x41}, {0x15, 0x45}, {0x19, 0x49}} {
				p.put(q+o[0], 4, uint64(p.u32(q+o[1])))
			}
			p.put(q+9, 4, 0xffffffff)
			for i := uint32(0); i < 8; i++ {
				p.put(q+0x1d+4*i, 4, uint64(p.u32(q+0x4d+4*i)))
			}
			p.put(gm+0xbf, 1, 0)
			if p.u32(q+1) <= 1 && p.eligible() {
				p.budget()
			}
		} else if p.u32(q+1) <= 1 && p.u8(gm+0x99) != 0 {
			for _, o := range []uint32{0x58, 0x5c, 0x80} {
				p.put(gm+o, 4, 0)
			}
			p.setAction(-1)
		}
	}
	a := int32(p.u32(q + 1))
	spell := p.u32(q + 5)
	if combat {
		if a != 4 && a != 5 {
			p.put(gm+0x4c, 1, 1)
			p.stub(0x562a50, 1<<who|0x100000)
		}
		if !p.eligible() {
			a = -1
			p.setAction(a)
		}
		l78 = p.override()
		if l78 != -1 {
			p.msg(p.str(p.u16(0x61e658 + 2*uint32(l78))))
			if uint32(l78) <= 10 {
				l := uint32(0)
				switch dst := p.u32(0x4eac3c + 4*uint32(p.u8(0x4eac4c+uint32(l78)))); dst {
				case 0x4ea804:
					a = -1
					p.setAction(a)
					if l78 == 8 || l78 == 9 {
						panic("character action prepare: the code table sends 8 and 9 to their own arms (0x4ea823 unreachable)")
					}
				case 0x4ea825:
					a = -1
					p.setAction(a)
					p.condRemove()
				case 0x4ea847:
					p.put(gm+0x98, 1, 1)
					l = 2
					r := int32(-1)
					for h := uint32(0); h < 2; h++ {
						if p.u8(c+0x114d+0x5b*h) != 0 {
							if v := p.attackRange(h); v > r {
								r = v
							}
						}
					}
					a = 0
					if r <= 1 {
						a = 1
					}
					p.setAction(a)
				case 0x4ea888:
					l = 1
					a = 0
					p.setAction(a)
				default:
					panic(fmt.Sprintf("character action prepare: unknown override destination %#x", dst))
				}
				if l != 0 {
					if t := p.target(l); t == -1 {
						p.stub(0x5182e0, 1, 0x61750c, c+5, 0)
						a = -1
						p.setAction(a)
					} else {
						if l81 == 0 {
							p.budget()
						}
						for i, v := range []uint32{1, uint32(t), 0xffffffff, 0xffffffff, 0, 0, 0, 0} {
							p.put(q+0x1d+4*uint32(i), 4, uint64(v))
						}
					}
				}
			}
		}
		if p.eligible() {
			p.stub(0x555560, who, q+0x1d)
		}
	}
	if p.allowed(a, spell) && (!combat || (l78 == -1 && p.u8(gm+0x98) == 0)) {
		p.stub(0x5471d0, 0, q+0x1d)
	}
	st := cpaT
	switch a {
	case -1:
		st = cpaX
	case 0:
		l82 = p.swings(-1)
	case 1:
		l82 = p.swings(3)
	case 2:
		l82 = p.breath()
	case 3:
		st = cpaX
		if p.u32(0x683fa5) != 0 {
			p.turnUndead()
			if p.u32(cpaL7c) != 0xffffffff {
				st = cpaE
			}
		}
	case 4:
		if p.u32(0x6836a8) == 0 {
			panic("character action prepare: defend reads the battle record, which is null")
		}
		st = cpaE
		if p.u8(gm+0xbc) != 0 {
			st = cpaM
		}
	case 5:
		l82 = p.guard()
	case 6:
		r := p.prayer()
		p.put(cpaL7c, 4, uint64(r))
		st = cpaX
		if r != 0 {
			st = cpaE
		}
	case 7:
		pl := p.u32(q + 0x15)
		p.put(cpaL7c, 4, 0)
		for {
			res := p.cast(spell, pl)
			if res == 0 {
				l82 = false
			}
			l7c := p.u32(cpaL7c) + p.u32(cpaL6c)
			p.put(cpaL7c, 4, uint64(l7c))
			if res != 2 || pl != 8 {
				break
			}
			if spell >= cpaSpells {
				panic("character action prepare: spell record past the fixture's 8")
			}
			rec := p.u32(0x65be1c) + 0x1bf*spell
			if int32(p.u32(c+0xb1d)) < int32(p.u32(rec+0x56))+1+int32(l7c) ||
				int32(p.u32(c+0xb45+4*p.u32(rec+0x133))) < int32(p.u32(rec+0x49)) {
				break
			}
		}
	case 8:
		st = cpaX
		if p.itemUse() != 0 {
			st = cpaE
		}
	case 9:
		st = cpaE
	default:
		p.stub(0x5182e0, 1, 0x6174cc, who, p.str(p.u16(0x61e988+2*uint32(a))))
		st = cpaE
	}
	if st == cpaT {
		st = cpaX
		if l82 {
			st = cpaE
		}
	}
	if st == cpaX {
		st = cpaE
		if l78 == -1 && l81 == 0 {
			a = 4
			st = cpaM
		} else {
			a = -1
		}
		p.setAction(a)
	}
	if st == cpaM && p.u8(0x68510c) != 0 {
		p.msg(p.str(0x234))
	}
	if g := p.u32(0x6836a8); g != 0 {
		if a != 4 && a != 5 {
			p.put(g+0xa55, 1, 0)
		}
		if p.allowed(a, spell) && p.u32(q+0x1d)-3 <= 3 {
			p.put(g+0xa5c, 4, 0)
		}
	}
	if uint32(a) > 1 {
		cost := p.u32(cpaL7c)
		if cost == 0xffffffff {
			cost = p.fatigueCost(a)
			p.put(cpaL7c, 4, uint64(cost))
		}
		p.fatigue(cost)
	}
	if combat {
		p.put(p.u32(0x6836a8)+0x7b0, 4, 2)
	}
}

func cpaRun(real bool) func([]int64, Rand) []int64 {
	return func(in []int64, r Rand) []int64 {
		if len(in) != len(cpaSpec.in) {
			panic("character action prepare: wrong input count")
		}
		w := &mcaWorld{mstWorld: mstBuild(&cpaSpec, in), real: real, tab: cpaStubs}
		w.r = r
		who := mcaArg(w, 0)
		if who >= cpaSlots {
			panic("character action prepare: party slot past the fixture's 3")
		}
		p := &cpaWorld{mcaWorld: w, who: who, c: w.u32(0x685174) + 0x1862*who, q: w.u32(0x685178) + 0x106*who}
		p.prepare()
		out := make([]int64, len(cpaSpec.out))
		for i, f := range cpaSpec.out {
			out[i] = w.read(f, in)
		}
		return out
	}
}

func characterActionPrepareNode(in []int64, r Rand) []int64 { return cpaRun(true)(in, r) }

func cpaEmu(in []int64, r Rand) []int64 { return cpaRun(false)(in, r) }

func init() {
	Nodes["004ea5c0"] = Node{Addr: "004ea5c0", In: cpaIn, Out: cpaOut, F: characterActionPrepareNode, Emu: cpaEmu}
}

// Code below is generated from the same description as re/emu/cases/004ea5c0.json.

var cpaSpec = cpaStrings(mstSpec{
	set: []mstSet{
		{0x685174, 4, 0x20000400},
		{0x685178, 4, 0x20000000},
		{0x68c09c, 4, 0x20006000},
		{0x65be1c, 4, 0x20008000},
		{0x20000405, 2, 0x41},
		{0x20000407, 2, 0x42},
		{0x20000409, 2, 0x0},
		{0x20000001, 4, 0xd0000001},
		{0x20000005, 4, 0xd0000005},
		{0x20000009, 4, 0xd0000009},
		{0x20000015, 4, 0xd0000015},
		{0x20000019, 4, 0xd0000019},
		{0x2000003d, 4, 0xd000003d},
		{0x20000041, 4, 0xd0000041},
		{0x20000045, 4, 0xd0000045},
		{0x20000049, 4, 0xd0000049},
		{0x2000001d, 4, 0xd000001d},
		{0x20000021, 4, 0xd0000021},
		{0x20000025, 4, 0xd0000025},
		{0x20000029, 4, 0xd0000029},
		{0x2000002d, 4, 0xd000002d},
		{0x20000031, 4, 0xd0000031},
		{0x20000035, 4, 0xd0000035},
		{0x20000039, 4, 0xd0000039},
		{0x2000004d, 4, 0xd000004d},
		{0x20000051, 4, 0xd0000051},
		{0x20000055, 4, 0xd0000055},
		{0x20000059, 4, 0xd0000059},
		{0x2000005d, 4, 0xd000005d},
		{0x20000061, 4, 0xd0000061},
		{0x20000065, 4, 0xd0000065},
		{0x20000069, 4, 0xd0000069},
		{0x200000f1, 1, 0x31},
		{0x2000504c, 1, 0x4c},
		{0x200050bf, 1, 0x4f},
		{0x20005098, 1, 0x48},
		{0x20005099, 1, 0x49},
		{0x200050bc, 1, 0x4c},
		{0x20005058, 4, 0xd1000058},
		{0x2000505c, 4, 0xd100005c},
		{0x20005080, 4, 0xd1000080},
		{0x20000f1d, 4, 0x100},
		{0x20000f45, 4, 0x200},
		{0x20000f49, 4, 0x201},
		{0x20000f4d, 4, 0x202},
		{0x20000f51, 4, 0x203},
		{0x2000154d, 1, 0x1},
		{0x200015a8, 1, 0x1},
		{0x20001c67, 2, 0x41},
		{0x20001c69, 2, 0x42},
		{0x20001c6b, 2, 0x43},
		{0x20001c6d, 2, 0x44},
		{0x20001c6f, 2, 0x0},
		{0x20000107, 4, 0xd0010001},
		{0x2000010b, 4, 0xd0010005},
		{0x2000010f, 4, 0xd0010009},
		{0x2000011b, 4, 0xd0010015},
		{0x2000011f, 4, 0xd0010019},
		{0x20000143, 4, 0xd001003d},
		{0x20000147, 4, 0xd0010041},
		{0x2000014b, 4, 0xd0010045},
		{0x2000014f, 4, 0xd0010049},
		{0x20000123, 4, 0xd001001d},
		{0x20000127, 4, 0xd0010021},
		{0x2000012b, 4, 0xd0010025},
		{0x2000012f, 4, 0xd0010029},
		{0x20000133, 4, 0xd001002d},
		{0x20000137, 4, 0xd0010031},
		{0x2000013b, 4, 0xd0010035},
		{0x2000013f, 4, 0xd0010039},
		{0x20000153, 4, 0xd001004d},
		{0x20000157, 4, 0xd0010051},
		{0x2000015b, 4, 0xd0010055},
		{0x2000015f, 4, 0xd0010059},
		{0x20000163, 4, 0xd001005d},
		{0x20000167, 4, 0xd0010061},
		{0x2000016b, 4, 0xd0010065},
		{0x2000016f, 4, 0xd0010069},
		{0x200001f7, 1, 0x32},
		{0x20005120, 1, 0x5c},
		{0x20005193, 1, 0x5f},
		{0x2000516c, 1, 0x58},
		{0x2000516d, 1, 0x59},
		{0x20005190, 1, 0x5c},
		{0x2000512c, 4, 0xd1010058},
		{0x20005130, 4, 0xd101005c},
		{0x20005154, 4, 0xd1010080},
		{0x2000277f, 4, 0x101},
		{0x200027a7, 4, 0x210},
		{0x200027ab, 4, 0x211},
		{0x200027af, 4, 0x212},
		{0x200027b3, 4, 0x213},
		{0x20002daf, 1, 0x1},
		{0x20002e0a, 1, 0x1},
		{0x200034c9, 2, 0x41},
		{0x200034cb, 2, 0x0},
		{0x2000020d, 4, 0xd0020001},
		{0x20000211, 4, 0xd0020005},
		{0x20000215, 4, 0xd0020009},
		{0x20000221, 4, 0xd0020015},
		{0x20000225, 4, 0xd0020019},
		{0x20000249, 4, 0xd002003d},
		{0x2000024d, 4, 0xd0020041},
		{0x20000251, 4, 0xd0020045},
		{0x20000255, 4, 0xd0020049},
		{0x20000229, 4, 0xd002001d},
		{0x2000022d, 4, 0xd0020021},
		{0x20000231, 4, 0xd0020025},
		{0x20000235, 4, 0xd0020029},
		{0x20000239, 4, 0xd002002d},
		{0x2000023d, 4, 0xd0020031},
		{0x20000241, 4, 0xd0020035},
		{0x20000245, 4, 0xd0020039},
		{0x20000259, 4, 0xd002004d},
		{0x2000025d, 4, 0xd0020051},
		{0x20000261, 4, 0xd0020055},
		{0x20000265, 4, 0xd0020059},
		{0x20000269, 4, 0xd002005d},
		{0x2000026d, 4, 0xd0020061},
		{0x20000271, 4, 0xd0020065},
		{0x20000275, 4, 0xd0020069},
		{0x200002fd, 1, 0x33},
		{0x200051f4, 1, 0x6c},
		{0x20005267, 1, 0x6f},
		{0x20005240, 1, 0x68},
		{0x20005241, 1, 0x69},
		{0x20005264, 1, 0x6c},
		{0x20005200, 4, 0xd1020058},
		{0x20005204, 4, 0xd102005c},
		{0x20005228, 4, 0xd1020080},
		{0x20003fe1, 4, 0x102},
		{0x20004009, 4, 0x220},
		{0x2000400d, 4, 0x221},
		{0x20004011, 4, 0x222},
		{0x20004015, 4, 0x223},
		{0x20004611, 1, 0x1},
		{0x2000466c, 1, 0x1},
		{0x20005a55, 1, 0x55},
		{0x20005a5c, 4, 0xdecada5c},
		{0x200057b0, 4, 0xdecad7b0},
	},
	in: []mstFld{
		{addr: 0x7f000004, size: 4, signed: false, idx: -1},
		{addr: 0x683f94, size: 1, signed: false, idx: -1},
		{addr: 0x683fa5, size: 4, signed: false, idx: -1},
		{addr: 0x68510c, size: 1, signed: false, idx: -1},
		{addr: 0x6836a8, size: 4, signed: false, idx: -1},
		{addr: 0x10007f7f, size: 1, signed: false, idx: -1},
		{addr: 0x20000001, size: 4, signed: true, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000005, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000015, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000019, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x2000001d, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000021, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x2000003d, size: 4, signed: true, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000041, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000045, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000049, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x2000004d, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000051, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000055, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000059, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x2000005d, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000061, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000065, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x20000069, size: 4, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x200000f1, size: 1, signed: false, idx: 0, count: 3, stride: 0x106},
		{addr: 0x2000504c, size: 1, signed: false, idx: 0, count: 3, stride: 0xd4},
		{addr: 0x20005098, size: 1, signed: false, idx: 0, count: 3, stride: 0xd4},
		{addr: 0x20005099, size: 1, signed: false, idx: 0, count: 3, stride: 0xd4},
		{addr: 0x200050bc, size: 1, signed: false, idx: 0, count: 3, stride: 0xd4},
		{addr: 0x20000f1d, size: 4, signed: true, idx: 0, count: 3, stride: 0x1862},
		{addr: 0x20000f45, size: 4, signed: true, idx: 0, count: 3, stride: 0x1862},
		{addr: 0x20000f49, size: 4, signed: true, idx: 0, count: 3, stride: 0x1862},
		{addr: 0x20000f4d, size: 4, signed: true, idx: 0, count: 3, stride: 0x1862},
		{addr: 0x20000f51, size: 4, signed: true, idx: 0, count: 3, stride: 0x1862},
		{addr: 0x2000154d, size: 1, signed: false, idx: 0, count: 3, stride: 0x1862},
		{addr: 0x200015a8, size: 1, signed: false, idx: 0, count: 3, stride: 0x1862},
		{addr: 0x20008056, size: 4, signed: true, idx: -1},
		{addr: 0x20008133, size: 4, signed: false, idx: -1},
		{addr: 0x20008049, size: 4, signed: true, idx: -1},
		{addr: 0x20008215, size: 4, signed: true, idx: -1},
		{addr: 0x200082f2, size: 4, signed: false, idx: -1},
		{addr: 0x20008208, size: 4, signed: true, idx: -1},
		{addr: 0x200083d4, size: 4, signed: true, idx: -1},
		{addr: 0x200084b1, size: 4, signed: false, idx: -1},
		{addr: 0x200083c7, size: 4, signed: true, idx: -1},
		{addr: 0x20008593, size: 4, signed: true, idx: -1},
		{addr: 0x20008670, size: 4, signed: false, idx: -1},
		{addr: 0x20008586, size: 4, signed: true, idx: -1},
		{addr: 0x20008752, size: 4, signed: true, idx: -1},
		{addr: 0x2000882f, size: 4, signed: false, idx: -1},
		{addr: 0x20008745, size: 4, signed: true, idx: -1},
		{addr: 0x20008911, size: 4, signed: true, idx: -1},
		{addr: 0x200089ee, size: 4, signed: false, idx: -1},
		{addr: 0x20008904, size: 4, signed: true, idx: -1},
		{addr: 0x20008ad0, size: 4, signed: true, idx: -1},
		{addr: 0x20008bad, size: 4, signed: false, idx: -1},
		{addr: 0x20008ac3, size: 4, signed: true, idx: -1},
		{addr: 0x20008c8f, size: 4, signed: true, idx: -1},
		{addr: 0x20008d6c, size: 4, signed: false, idx: -1},
		{addr: 0x20008c82, size: 4, signed: true, idx: -1},
		{addr: 0x20007800, size: 2, signed: false, idx: -1},
		{addr: 0x20007802, size: 2, signed: false, idx: -1},
		{addr: 0x20007808, size: 2, signed: false, idx: -1},
		{addr: 0x2000780a, size: 2, signed: false, idx: -1},
		{addr: 0x20007810, size: 2, signed: false, idx: -1},
		{addr: 0x20007812, size: 2, signed: false, idx: -1},
		{addr: 0x20007818, size: 2, signed: false, idx: -1},
		{addr: 0x2000781a, size: 2, signed: false, idx: -1},
		{addr: 0x20007820, size: 2, signed: false, idx: -1},
		{addr: 0x20007822, size: 2, signed: false, idx: -1},
		{addr: 0x20007828, size: 2, signed: false, idx: -1},
		{addr: 0x2000782a, size: 2, signed: false, idx: -1},
		{addr: 0x20007830, size: 2, signed: false, idx: -1},
		{addr: 0x20007832, size: 2, signed: false, idx: -1},
		{addr: 0x20007838, size: 2, signed: false, idx: -1},
		{addr: 0x2000783a, size: 2, signed: false, idx: -1},
		{addr: 0x20007840, size: 2, signed: false, idx: -1},
		{addr: 0x20007842, size: 2, signed: false, idx: -1},
		{addr: 0x20007848, size: 2, signed: false, idx: -1},
		{addr: 0x2000784a, size: 2, signed: false, idx: -1},
		{addr: 0x20007850, size: 2, signed: false, idx: -1},
		{addr: 0x20007852, size: 2, signed: false, idx: -1},
		{addr: 0x20007858, size: 2, signed: false, idx: -1},
		{addr: 0x2000785a, size: 2, signed: false, idx: -1},
		{addr: 0x20007860, size: 2, signed: false, idx: -1},
		{addr: 0x20007862, size: 2, signed: false, idx: -1},
		{addr: 0x20007868, size: 2, signed: false, idx: -1},
		{addr: 0x2000786a, size: 2, signed: false, idx: -1},
		{addr: 0x20009060, size: 4, signed: false, idx: -1},
		{addr: 0x20009064, size: 4, signed: false, idx: -1},
		{addr: 0x20009068, size: 4, signed: false, idx: -1},
		{addr: 0x20009088, size: 4, signed: true, idx: -1},
		{addr: 0x200090a0, size: 4, signed: true, idx: -1},
		{addr: 0x200090a4, size: 4, signed: true, idx: -1},
		{addr: 0x200090bc, size: 4, signed: true, idx: -1},
		{addr: 0x200090cc, size: 4, signed: false, idx: -1},
		{addr: 0x200090d0, size: 4, signed: false, idx: -1},
		{addr: 0x200090f8, size: 4, signed: false, idx: -1},
		{addr: 0x20009108, size: 4, signed: false, idx: -1},
		{addr: 0x20009114, size: 4, signed: true, idx: -1},
		{addr: 0x20009124, size: 4, signed: false, idx: -1},
		{addr: 0x20009130, size: 4, signed: true, idx: -1},
		{addr: 0x2000913c, size: 4, signed: false, idx: -1},
		{addr: 0x20009140, size: 4, signed: false, idx: -1},
		{addr: 0x20009144, size: 4, signed: false, idx: -1},
		{addr: 0x20009148, size: 4, signed: false, idx: -1},
		{addr: 0x2000914c, size: 4, signed: false, idx: -1},
		{addr: 0x20009150, size: 4, signed: false, idx: -1},
		{addr: 0x20009154, size: 4, signed: false, idx: -1},
		{addr: 0x20009158, size: 4, signed: false, idx: -1},
		{addr: 0x2000915c, size: 4, signed: false, idx: -1},
		{addr: 0x20009160, size: 4, signed: false, idx: -1},
		{addr: 0x20009164, size: 4, signed: false, idx: -1},
		{addr: 0x20009168, size: 4, signed: false, idx: -1},
		{addr: 0x2000916c, size: 4, signed: false, idx: -1},
		{addr: 0x20009170, size: 4, signed: false, idx: -1},
		{addr: 0x20009174, size: 4, signed: false, idx: -1},
		{addr: 0x20009178, size: 4, signed: false, idx: -1},
		{addr: 0x2000917c, size: 4, signed: false, idx: -1},
		{addr: 0x20009180, size: 4, signed: false, idx: -1},
		{addr: 0x20009184, size: 4, signed: false, idx: -1},
		{addr: 0x20009188, size: 4, signed: false, idx: -1},
		{addr: 0x2000918c, size: 4, signed: false, idx: -1},
		{addr: 0x20009190, size: 4, signed: false, idx: -1},
		{addr: 0x20009194, size: 4, signed: false, idx: -1},
		{addr: 0x20009198, size: 4, signed: false, idx: -1},
		{addr: 0x2000919c, size: 4, signed: false, idx: -1},
		{addr: 0x200091a0, size: 4, signed: false, idx: -1},
		{addr: 0x200091a4, size: 4, signed: true, idx: -1},
		{addr: 0x200091a8, size: 4, signed: true, idx: -1},
		{addr: 0x200091ac, size: 4, signed: true, idx: -1},
		{addr: 0x200091b0, size: 4, signed: true, idx: -1},
		{addr: 0x200091b4, size: 4, signed: true, idx: -1},
		{addr: 0x200091b8, size: 4, signed: true, idx: -1},
		{addr: 0x200091bc, size: 4, signed: true, idx: -1},
		{addr: 0x200091c0, size: 4, signed: true, idx: -1},
		{addr: 0x200091c4, size: 4, signed: true, idx: -1},
		{addr: 0x200091c8, size: 4, signed: true, idx: -1},
		{addr: 0x200091cc, size: 4, signed: true, idx: -1},
		{addr: 0x200091d0, size: 4, signed: true, idx: -1},
		{addr: 0x200091d4, size: 4, signed: true, idx: -1},
		{addr: 0x200091d8, size: 4, signed: true, idx: -1},
		{addr: 0x200091dc, size: 4, signed: true, idx: -1},
		{addr: 0x200091e0, size: 4, signed: true, idx: -1},
		{addr: 0x200091e4, size: 4, signed: true, idx: -1},
		{addr: 0x200091e8, size: 4, signed: true, idx: -1},
		{addr: 0x200091ec, size: 4, signed: true, idx: -1},
		{addr: 0x200091f0, size: 4, signed: true, idx: -1},
		{addr: 0x200091f4, size: 4, signed: true, idx: -1},
		{addr: 0x200091f8, size: 4, signed: true, idx: -1},
		{addr: 0x200091fc, size: 4, signed: true, idx: -1},
		{addr: 0x20009200, size: 4, signed: true, idx: -1},
		{addr: 0x20009204, size: 4, signed: true, idx: -1},
		{addr: 0x20009208, size: 4, signed: true, idx: -1},
		{addr: 0x200093b0, size: 4, signed: false, idx: -1},
		{addr: 0x200093b4, size: 4, signed: true, idx: -1},
		{addr: 0x200093c4, size: 4, signed: false, idx: -1},
	},
	out: []mstFld{
		{addr: 0x20000001, size: 4, signed: true, idx: -1},
		{addr: 0x20000005, size: 4, signed: false, idx: -1},
		{addr: 0x20000009, size: 4, signed: false, idx: -1},
		{addr: 0x20000015, size: 4, signed: false, idx: -1},
		{addr: 0x20000019, size: 4, signed: false, idx: -1},
		{addr: 0x2000001d, size: 4, signed: false, idx: -1},
		{addr: 0x20000021, size: 4, signed: false, idx: -1},
		{addr: 0x20000025, size: 4, signed: false, idx: -1},
		{addr: 0x20000029, size: 4, signed: false, idx: -1},
		{addr: 0x2000002d, size: 4, signed: false, idx: -1},
		{addr: 0x20000031, size: 4, signed: false, idx: -1},
		{addr: 0x20000035, size: 4, signed: false, idx: -1},
		{addr: 0x20000039, size: 4, signed: false, idx: -1},
		{addr: 0x2000504c, size: 1, signed: false, idx: -1},
		{addr: 0x200050bf, size: 1, signed: false, idx: -1},
		{addr: 0x20005098, size: 1, signed: false, idx: -1},
		{addr: 0x20005058, size: 4, signed: false, idx: -1},
		{addr: 0x2000505c, size: 4, signed: false, idx: -1},
		{addr: 0x20005080, size: 4, signed: false, idx: -1},
		{addr: 0x20000107, size: 4, signed: true, idx: -1},
		{addr: 0x2000010b, size: 4, signed: false, idx: -1},
		{addr: 0x2000010f, size: 4, signed: false, idx: -1},
		{addr: 0x2000011b, size: 4, signed: false, idx: -1},
		{addr: 0x2000011f, size: 4, signed: false, idx: -1},
		{addr: 0x20000123, size: 4, signed: false, idx: -1},
		{addr: 0x20000127, size: 4, signed: false, idx: -1},
		{addr: 0x2000012b, size: 4, signed: false, idx: -1},
		{addr: 0x2000012f, size: 4, signed: false, idx: -1},
		{addr: 0x20000133, size: 4, signed: false, idx: -1},
		{addr: 0x20000137, size: 4, signed: false, idx: -1},
		{addr: 0x2000013b, size: 4, signed: false, idx: -1},
		{addr: 0x2000013f, size: 4, signed: false, idx: -1},
		{addr: 0x20005120, size: 1, signed: false, idx: -1},
		{addr: 0x20005193, size: 1, signed: false, idx: -1},
		{addr: 0x2000516c, size: 1, signed: false, idx: -1},
		{addr: 0x2000512c, size: 4, signed: false, idx: -1},
		{addr: 0x20005130, size: 4, signed: false, idx: -1},
		{addr: 0x20005154, size: 4, signed: false, idx: -1},
		{addr: 0x2000020d, size: 4, signed: true, idx: -1},
		{addr: 0x20000211, size: 4, signed: false, idx: -1},
		{addr: 0x20000215, size: 4, signed: false, idx: -1},
		{addr: 0x20000221, size: 4, signed: false, idx: -1},
		{addr: 0x20000225, size: 4, signed: false, idx: -1},
		{addr: 0x20000229, size: 4, signed: false, idx: -1},
		{addr: 0x2000022d, size: 4, signed: false, idx: -1},
		{addr: 0x20000231, size: 4, signed: false, idx: -1},
		{addr: 0x20000235, size: 4, signed: false, idx: -1},
		{addr: 0x20000239, size: 4, signed: false, idx: -1},
		{addr: 0x2000023d, size: 4, signed: false, idx: -1},
		{addr: 0x20000241, size: 4, signed: false, idx: -1},
		{addr: 0x20000245, size: 4, signed: false, idx: -1},
		{addr: 0x200051f4, size: 1, signed: false, idx: -1},
		{addr: 0x20005267, size: 1, signed: false, idx: -1},
		{addr: 0x20005240, size: 1, signed: false, idx: -1},
		{addr: 0x20005200, size: 4, signed: false, idx: -1},
		{addr: 0x20005204, size: 4, signed: false, idx: -1},
		{addr: 0x20005228, size: 4, signed: false, idx: -1},
		{addr: 0x20005a55, size: 1, signed: false, idx: -1},
		{addr: 0x20005a5c, size: 4, signed: false, idx: -1},
		{addr: 0x200057b0, size: 4, signed: false, idx: -1},
		{addr: 0x20009000, size: 4, signed: false, idx: -1},
		{addr: 0x20009004, size: 4, signed: false, idx: -1},
		{addr: 0x20009008, size: 4, signed: false, idx: -1},
		{addr: 0x2000900c, size: 4, signed: false, idx: -1},
		{addr: 0x20009010, size: 4, signed: false, idx: -1},
		{addr: 0x20009014, size: 4, signed: false, idx: -1},
		{addr: 0x20009018, size: 4, signed: false, idx: -1},
		{addr: 0x2000901c, size: 4, signed: false, idx: -1},
		{addr: 0x20009020, size: 4, signed: false, idx: -1},
		{addr: 0x20009024, size: 4, signed: false, idx: -1},
		{addr: 0x20009028, size: 4, signed: false, idx: -1},
		{addr: 0x2000902c, size: 4, signed: false, idx: -1},
		{addr: 0x20009030, size: 4, signed: false, idx: -1},
		{addr: 0x20009034, size: 4, signed: false, idx: -1},
		{addr: 0x20009038, size: 4, signed: false, idx: -1},
		{addr: 0x2000903c, size: 4, signed: false, idx: -1},
		{addr: 0x20009040, size: 4, signed: false, idx: -1},
		{addr: 0x20009044, size: 4, signed: false, idx: -1},
		{addr: 0x20009048, size: 4, signed: false, idx: -1},
		{addr: 0x2000904c, size: 4, signed: false, idx: -1},
		{addr: 0x20009050, size: 1, signed: false, idx: -1},
		{addr: 0x20009051, size: 1, signed: false, idx: -1},
		{addr: 0x20009054, size: 4, signed: false, idx: -1},
		{addr: 0x20009058, size: 4, signed: false, idx: -1},
		{addr: 0x2000905c, size: 4, signed: false, idx: -1},
		{addr: 0x2000906c, size: 4, signed: false, idx: -1},
		{addr: 0x20009070, size: 4, signed: false, idx: -1},
		{addr: 0x20009074, size: 4, signed: false, idx: -1},
		{addr: 0x20009078, size: 4, signed: false, idx: -1},
		{addr: 0x2000907c, size: 4, signed: false, idx: -1},
		{addr: 0x20009080, size: 4, signed: false, idx: -1},
		{addr: 0x20009084, size: 4, signed: false, idx: -1},
		{addr: 0x2000908c, size: 4, signed: false, idx: -1},
		{addr: 0x20009090, size: 4, signed: false, idx: -1},
		{addr: 0x20009094, size: 4, signed: false, idx: -1},
		{addr: 0x20009098, size: 4, signed: false, idx: -1},
		{addr: 0x2000909c, size: 4, signed: false, idx: -1},
		{addr: 0x200090a8, size: 4, signed: false, idx: -1},
		{addr: 0x200090ac, size: 4, signed: false, idx: -1},
		{addr: 0x200090b0, size: 4, signed: false, idx: -1},
		{addr: 0x200090b4, size: 4, signed: false, idx: -1},
		{addr: 0x200090b8, size: 4, signed: false, idx: -1},
		{addr: 0x200090c0, size: 4, signed: false, idx: -1},
		{addr: 0x200090c4, size: 1, signed: false, idx: -1},
		{addr: 0x200090c8, size: 4, signed: false, idx: -1},
		{addr: 0x200090d4, size: 4, signed: false, idx: -1},
		{addr: 0x200090d8, size: 4, signed: false, idx: -1},
		{addr: 0x200090dc, size: 4, signed: false, idx: -1},
		{addr: 0x200090e0, size: 4, signed: false, idx: -1},
		{addr: 0x200090e4, size: 4, signed: false, idx: -1},
		{addr: 0x200090e8, size: 4, signed: false, idx: -1},
		{addr: 0x200090ec, size: 4, signed: false, idx: -1},
		{addr: 0x200090f0, size: 4, signed: false, idx: -1},
		{addr: 0x200090f4, size: 4, signed: false, idx: -1},
		{addr: 0x200090fc, size: 4, signed: false, idx: -1},
		{addr: 0x20009100, size: 4, signed: false, idx: -1},
		{addr: 0x20009104, size: 4, signed: false, idx: -1},
		{addr: 0x2000910c, size: 4, signed: false, idx: -1},
		{addr: 0x20009110, size: 4, signed: false, idx: -1},
		{addr: 0x20009118, size: 4, signed: false, idx: -1},
		{addr: 0x2000911c, size: 4, signed: false, idx: -1},
		{addr: 0x20009120, size: 4, signed: false, idx: -1},
		{addr: 0x20009128, size: 4, signed: false, idx: -1},
		{addr: 0x2000912c, size: 4, signed: false, idx: -1},
		{addr: 0x20009134, size: 4, signed: false, idx: -1},
		{addr: 0x20009138, size: 4, signed: false, idx: -1},
		{addr: 0x2000920c, size: 4, signed: false, idx: -1},
		{addr: 0x20009210, size: 4, signed: false, idx: -1},
		{addr: 0x20009214, size: 4, signed: false, idx: -1},
		{addr: 0x20009218, size: 4, signed: false, idx: -1},
		{addr: 0x2000921c, size: 4, signed: false, idx: -1},
		{addr: 0x20009220, size: 4, signed: false, idx: -1},
		{addr: 0x20009224, size: 4, signed: false, idx: -1},
		{addr: 0x20009228, size: 4, signed: false, idx: -1},
		{addr: 0x2000922c, size: 4, signed: false, idx: -1},
		{addr: 0x20009230, size: 4, signed: false, idx: -1},
		{addr: 0x20009234, size: 4, signed: false, idx: -1},
		{addr: 0x20009238, size: 4, signed: false, idx: -1},
		{addr: 0x2000923c, size: 4, signed: false, idx: -1},
		{addr: 0x20009240, size: 4, signed: false, idx: -1},
		{addr: 0x20009244, size: 4, signed: false, idx: -1},
		{addr: 0x20009248, size: 4, signed: false, idx: -1},
		{addr: 0x2000924c, size: 4, signed: false, idx: -1},
		{addr: 0x20009250, size: 4, signed: false, idx: -1},
		{addr: 0x20009254, size: 4, signed: false, idx: -1},
		{addr: 0x20009258, size: 4, signed: false, idx: -1},
		{addr: 0x2000925c, size: 4, signed: false, idx: -1},
		{addr: 0x20009260, size: 4, signed: false, idx: -1},
		{addr: 0x20009264, size: 4, signed: false, idx: -1},
		{addr: 0x20009268, size: 4, signed: false, idx: -1},
		{addr: 0x2000926c, size: 4, signed: false, idx: -1},
		{addr: 0x20009270, size: 4, signed: false, idx: -1},
		{addr: 0x20009274, size: 4, signed: false, idx: -1},
		{addr: 0x20009278, size: 4, signed: false, idx: -1},
		{addr: 0x2000927c, size: 4, signed: false, idx: -1},
		{addr: 0x20009280, size: 4, signed: false, idx: -1},
		{addr: 0x20009284, size: 4, signed: false, idx: -1},
		{addr: 0x20009288, size: 4, signed: false, idx: -1},
		{addr: 0x2000928c, size: 4, signed: false, idx: -1},
		{addr: 0x20009290, size: 4, signed: false, idx: -1},
		{addr: 0x20009294, size: 4, signed: false, idx: -1},
		{addr: 0x20009298, size: 4, signed: false, idx: -1},
		{addr: 0x2000929c, size: 4, signed: false, idx: -1},
		{addr: 0x200092a0, size: 4, signed: false, idx: -1},
		{addr: 0x200092a4, size: 4, signed: false, idx: -1},
		{addr: 0x200092a8, size: 4, signed: false, idx: -1},
		{addr: 0x200092ac, size: 4, signed: false, idx: -1},
		{addr: 0x200092b0, size: 4, signed: false, idx: -1},
		{addr: 0x200092b4, size: 4, signed: false, idx: -1},
		{addr: 0x200092b8, size: 4, signed: false, idx: -1},
		{addr: 0x200092bc, size: 4, signed: false, idx: -1},
		{addr: 0x200092c0, size: 4, signed: false, idx: -1},
		{addr: 0x200092c4, size: 4, signed: false, idx: -1},
		{addr: 0x200092c8, size: 4, signed: false, idx: -1},
		{addr: 0x200092cc, size: 4, signed: false, idx: -1},
		{addr: 0x200092d0, size: 4, signed: false, idx: -1},
		{addr: 0x200092d4, size: 4, signed: false, idx: -1},
		{addr: 0x200092d8, size: 4, signed: false, idx: -1},
		{addr: 0x200092dc, size: 4, signed: false, idx: -1},
		{addr: 0x200092e0, size: 4, signed: false, idx: -1},
		{addr: 0x200092e4, size: 4, signed: false, idx: -1},
		{addr: 0x200092e8, size: 4, signed: false, idx: -1},
		{addr: 0x200092ec, size: 4, signed: false, idx: -1},
		{addr: 0x200092f0, size: 4, signed: false, idx: -1},
		{addr: 0x200092f4, size: 4, signed: false, idx: -1},
		{addr: 0x200092f8, size: 4, signed: false, idx: -1},
		{addr: 0x200092fc, size: 4, signed: false, idx: -1},
		{addr: 0x20009300, size: 4, signed: false, idx: -1},
		{addr: 0x20009304, size: 4, signed: false, idx: -1},
		{addr: 0x20009308, size: 4, signed: false, idx: -1},
		{addr: 0x2000930c, size: 4, signed: false, idx: -1},
		{addr: 0x20009310, size: 4, signed: false, idx: -1},
		{addr: 0x20009314, size: 4, signed: false, idx: -1},
		{addr: 0x20009318, size: 4, signed: false, idx: -1},
		{addr: 0x2000931c, size: 4, signed: false, idx: -1},
		{addr: 0x20009320, size: 4, signed: false, idx: -1},
		{addr: 0x20009324, size: 4, signed: false, idx: -1},
		{addr: 0x20009328, size: 4, signed: false, idx: -1},
		{addr: 0x2000932c, size: 4, signed: false, idx: -1},
		{addr: 0x20009330, size: 4, signed: false, idx: -1},
		{addr: 0x20009334, size: 4, signed: false, idx: -1},
		{addr: 0x20009338, size: 4, signed: false, idx: -1},
		{addr: 0x2000933c, size: 4, signed: false, idx: -1},
		{addr: 0x20009340, size: 4, signed: false, idx: -1},
		{addr: 0x20009344, size: 4, signed: false, idx: -1},
		{addr: 0x20009348, size: 4, signed: false, idx: -1},
		{addr: 0x2000934c, size: 4, signed: false, idx: -1},
		{addr: 0x20009350, size: 4, signed: false, idx: -1},
		{addr: 0x20009354, size: 4, signed: false, idx: -1},
		{addr: 0x20009358, size: 4, signed: false, idx: -1},
		{addr: 0x2000935c, size: 4, signed: false, idx: -1},
		{addr: 0x20009360, size: 4, signed: false, idx: -1},
		{addr: 0x20009364, size: 4, signed: false, idx: -1},
		{addr: 0x20009368, size: 4, signed: false, idx: -1},
		{addr: 0x2000936c, size: 4, signed: false, idx: -1},
		{addr: 0x20009370, size: 4, signed: false, idx: -1},
		{addr: 0x20009374, size: 4, signed: false, idx: -1},
		{addr: 0x20009378, size: 4, signed: false, idx: -1},
		{addr: 0x2000937c, size: 4, signed: false, idx: -1},
		{addr: 0x20009380, size: 4, signed: false, idx: -1},
		{addr: 0x20009384, size: 4, signed: false, idx: -1},
		{addr: 0x20009388, size: 4, signed: false, idx: -1},
		{addr: 0x2000938c, size: 4, signed: false, idx: -1},
		{addr: 0x20009390, size: 4, signed: false, idx: -1},
		{addr: 0x20009394, size: 4, signed: false, idx: -1},
		{addr: 0x20009398, size: 4, signed: false, idx: -1},
		{addr: 0x2000939c, size: 4, signed: false, idx: -1},
		{addr: 0x200093a0, size: 4, signed: false, idx: -1},
		{addr: 0x200093a4, size: 4, signed: false, idx: -1},
		{addr: 0x200093a8, size: 4, signed: false, idx: -1},
		{addr: 0x200093ac, size: 4, signed: false, idx: -1},
		{addr: 0x200093b8, size: 4, signed: false, idx: -1},
		{addr: 0x200093bc, size: 4, signed: false, idx: -1},
		{addr: 0x200093c0, size: 4, signed: false, idx: -1},
		{addr: 0x200093c8, size: 4, signed: false, idx: -1},
		{addr: 0x200093cc, size: 4, signed: false, idx: -1},
		{addr: 0x200093d0, size: 4, signed: false, idx: -1},
		{addr: 0x200093d4, size: 4, signed: false, idx: -1},
		{addr: 0x200093d8, size: 4, signed: false, idx: -1},
		{addr: 0x200093dc, size: 4, signed: false, idx: -1},
		{addr: 0x200093e0, size: 4, signed: false, idx: -1},
		{addr: 0x200093e4, size: 4, signed: false, idx: -1},
	},
})

var cpaIn = []string{"who", "combat", "g683fa5", "g68510c", "G", "local81", "Q.action", "Q.spell", "Q.power", "Q.item", "Q.rec0", "Q.rec1", "Q.next.action", "Q.next.spell", "Q.next.power", "Q.next.item", "Q.next.rec0", "Q.next.rec1", "Q.next.rec2", "Q.next.rec3", "Q.next.rec4", "Q.next.rec5", "Q.next.rec6", "Q.next.rec7", "Q.colour", "Gm.f4c", "Gm.f98", "Gm.f99", "Gm.fbc", "C.fb1d", "C.school0", "C.school1", "C.school2", "C.school3", "C.hand0", "C.hand1", "sp0.f56", "sp0.f133", "sp0.f49", "sp1.f56", "sp1.f133", "sp1.f49", "sp2.f56", "sp2.f133", "sp2.f49", "sp3.f56", "sp3.f133", "sp3.f49", "sp4.f56", "sp4.f133", "sp4.f49", "sp5.f56", "sp5.f133", "sp5.f49", "sp6.f56", "sp6.f133", "sp6.f49", "sp7.f56", "sp7.f133", "sp7.f49", "text3a0.c0", "text3a0.c1", "text3a1.c0", "text3a1.c1", "text3a2.c0", "text3a2.c1", "text3a3.c0", "text3a3.c1", "text3a4.c0", "text3a4.c1", "text3a5.c0", "text3a5.c1", "text3a6.c0", "text3a6.c1", "text3a7.c0", "text3a7.c1", "text3a8.c0", "text3a8.c1", "text3a9.c0", "text3a9.c1", "text3aa.c0", "text3aa.c1", "text3ab.c0", "text3ab.c1", "text3ac.c0", "text3ac.c1", "text234.c0", "text234.c1", "r4e79a0_0", "r4e79a0_1", "r4e79a0_2", "r4ec1e0", "r519ac0_0", "r519ac0_1", "r51b0a0", "r547310_0", "r547310_1", "r53d870", "r4ebce0", "w547cb0_cost", "r545aa0", "r547fe0", "r4fa4d0_0", "r4fa4d0_1", "r4fa4d0_2", "r4fa4d0_3", "r4fa4d0_4", "r4fa4d0_5", "r4fa4d0_6", "r4fa4d0_7", "r4fa4d0_8", "r4fa4d0_9", "r4fa4d0_10", "r4fa4d0_11", "r4fa4d0_12", "r4fa4d0_13", "r4fa4d0_14", "r4fa4d0_15", "r4fa4d0_16", "r4fa4d0_17", "r4fa4d0_18", "r4fa4d0_19", "r4fa4d0_20", "r4fa4d0_21", "r4fa4d0_22", "r4fa4d0_23", "r4fa4d0_24", "r4fa4d0_25", "w4fa4d0_cost_0", "w4fa4d0_cost_1", "w4fa4d0_cost_2", "w4fa4d0_cost_3", "w4fa4d0_cost_4", "w4fa4d0_cost_5", "w4fa4d0_cost_6", "w4fa4d0_cost_7", "w4fa4d0_cost_8", "w4fa4d0_cost_9", "w4fa4d0_cost_10", "w4fa4d0_cost_11", "w4fa4d0_cost_12", "w4fa4d0_cost_13", "w4fa4d0_cost_14", "w4fa4d0_cost_15", "w4fa4d0_cost_16", "w4fa4d0_cost_17", "w4fa4d0_cost_18", "w4fa4d0_cost_19", "w4fa4d0_cost_20", "w4fa4d0_cost_21", "w4fa4d0_cost_22", "w4fa4d0_cost_23", "w4fa4d0_cost_24", "w4fa4d0_cost_25", "r51dde0", "w51dde0_cost", "r52b2f0"}

var cpaOut = []string{"Q0.action", "Q0.spell", "Q0.f9", "Q0.power", "Q0.item", "Q0.rec0", "Q0.rec1", "Q0.rec2", "Q0.rec3", "Q0.rec4", "Q0.rec5", "Q0.rec6", "Q0.rec7", "Gm0.f4c", "Gm0.fbf", "Gm0.f98", "Gm0.f58", "Gm0.f5c", "Gm0.f80", "Q1.action", "Q1.spell", "Q1.f9", "Q1.power", "Q1.item", "Q1.rec0", "Q1.rec1", "Q1.rec2", "Q1.rec3", "Q1.rec4", "Q1.rec5", "Q1.rec6", "Q1.rec7", "Gm1.f4c", "Gm1.fbf", "Gm1.f98", "Gm1.f58", "Gm1.f5c", "Gm1.f80", "Q2.action", "Q2.spell", "Q2.f9", "Q2.power", "Q2.item", "Q2.rec0", "Q2.rec1", "Q2.rec2", "Q2.rec3", "Q2.rec4", "Q2.rec5", "Q2.rec6", "Q2.rec7", "Gm2.f4c", "Gm2.fbf", "Gm2.f98", "Gm2.f58", "Gm2.f5c", "Gm2.f80", "G.fa55", "G.fa5c", "G.f7b0", "s5554a0_a1_0", "s5554a0_calls", "s555560_a1_0", "s555560_a2_0", "s555560_calls", "s5182e0_a2_0", "s5182e0_a3_0", "s5182e0_a4_0", "s5182e0_calls", "s562a50_a1_0", "s562a50_calls", "s5471d0_a2_0", "s5471d0_calls", "s58aad0_a1_0", "s58aad0_a1_1", "s58aad0_a2_0", "s58aad0_a2_1", "s58aad0_a3_0", "s58aad0_a3_1", "s58aad0_calls", "s58b410_a1_0", "s58b410_a1_1", "s58b410_a3_0", "s58b410_a3_1", "s58b410_calls", "s4e79a0_a1_0", "s4e79a0_a1_1", "s4e79a0_a1_2", "s4e79a0_calls", "s53d680_a1_0", "s53d680_a1_1", "s53d680_calls", "s4ec1e0_calls", "s523330_a1_0", "s523330_a2_0", "s523330_a3_0", "s523330_calls", "s519ac0_a1_0", "s519ac0_a1_1", "s519ac0_a2_0", "s519ac0_a2_1", "s519ac0_calls", "s51b0a0_a1_0", "s51b0a0_a2_0", "s51b0a0_calls", "s547310_a1_0", "s547310_a1_1", "s547310_a2_0", "s547310_a2_1", "s547310_a3_0", "s547310_a3_1", "s547310_a4_0", "s547310_a4_1", "s547310_calls", "s53d870_a1_0", "s53d870_a2_0", "s53d870_calls", "s4ebce0_a1_0", "s4ebce0_calls", "s547cb0_a1_0", "s547cb0_a3_0", "s547cb0_calls", "s545aa0_a1_0", "s545aa0_calls", "s547fe0_a1_0", "s547fe0_calls", "s4fa4d0_a1_0", "s4fa4d0_a1_1", "s4fa4d0_a1_2", "s4fa4d0_a1_3", "s4fa4d0_a1_4", "s4fa4d0_a1_5", "s4fa4d0_a1_6", "s4fa4d0_a1_7", "s4fa4d0_a1_8", "s4fa4d0_a1_9", "s4fa4d0_a1_10", "s4fa4d0_a1_11", "s4fa4d0_a1_12", "s4fa4d0_a1_13", "s4fa4d0_a1_14", "s4fa4d0_a1_15", "s4fa4d0_a1_16", "s4fa4d0_a1_17", "s4fa4d0_a1_18", "s4fa4d0_a1_19", "s4fa4d0_a1_20", "s4fa4d0_a1_21", "s4fa4d0_a1_22", "s4fa4d0_a1_23", "s4fa4d0_a1_24", "s4fa4d0_a1_25", "s4fa4d0_a2_0", "s4fa4d0_a2_1", "s4fa4d0_a2_2", "s4fa4d0_a2_3", "s4fa4d0_a2_4", "s4fa4d0_a2_5", "s4fa4d0_a2_6", "s4fa4d0_a2_7", "s4fa4d0_a2_8", "s4fa4d0_a2_9", "s4fa4d0_a2_10", "s4fa4d0_a2_11", "s4fa4d0_a2_12", "s4fa4d0_a2_13", "s4fa4d0_a2_14", "s4fa4d0_a2_15", "s4fa4d0_a2_16", "s4fa4d0_a2_17", "s4fa4d0_a2_18", "s4fa4d0_a2_19", "s4fa4d0_a2_20", "s4fa4d0_a2_21", "s4fa4d0_a2_22", "s4fa4d0_a2_23", "s4fa4d0_a2_24", "s4fa4d0_a2_25", "s4fa4d0_a3_0", "s4fa4d0_a3_1", "s4fa4d0_a3_2", "s4fa4d0_a3_3", "s4fa4d0_a3_4", "s4fa4d0_a3_5", "s4fa4d0_a3_6", "s4fa4d0_a3_7", "s4fa4d0_a3_8", "s4fa4d0_a3_9", "s4fa4d0_a3_10", "s4fa4d0_a3_11", "s4fa4d0_a3_12", "s4fa4d0_a3_13", "s4fa4d0_a3_14", "s4fa4d0_a3_15", "s4fa4d0_a3_16", "s4fa4d0_a3_17", "s4fa4d0_a3_18", "s4fa4d0_a3_19", "s4fa4d0_a3_20", "s4fa4d0_a3_21", "s4fa4d0_a3_22", "s4fa4d0_a3_23", "s4fa4d0_a3_24", "s4fa4d0_a3_25", "s4fa4d0_a5_0", "s4fa4d0_a5_1", "s4fa4d0_a5_2", "s4fa4d0_a5_3", "s4fa4d0_a5_4", "s4fa4d0_a5_5", "s4fa4d0_a5_6", "s4fa4d0_a5_7", "s4fa4d0_a5_8", "s4fa4d0_a5_9", "s4fa4d0_a5_10", "s4fa4d0_a5_11", "s4fa4d0_a5_12", "s4fa4d0_a5_13", "s4fa4d0_a5_14", "s4fa4d0_a5_15", "s4fa4d0_a5_16", "s4fa4d0_a5_17", "s4fa4d0_a5_18", "s4fa4d0_a5_19", "s4fa4d0_a5_20", "s4fa4d0_a5_21", "s4fa4d0_a5_22", "s4fa4d0_a5_23", "s4fa4d0_a5_24", "s4fa4d0_a5_25", "s4fa4d0_calls", "s51dde0_a1_0", "s51dde0_a2_0", "s51dde0_calls", "s52b2f0_a1_0", "s52b2f0_a2_0", "s52b2f0_calls", "s52af50_a1_0", "s52af50_a2_0", "s52af50_a3_0", "s52af50_a4_0", "s52af50_calls"}

var cpaStubs = map[uint32]*mcaStub{
	0x5554a0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20009000, 1}}, cnt: 0x20009004, wr: []mcaWrite{}},
	0x555560: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20009008, 1}, {2, 4, 0x2000900c, 1}}, cnt: 0x20009010, wr: []mcaWrite{}},
	0x5182e0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x20009014, 1}, {3, 4, 0x20009018, 1}, {4, 4, 0x2000901c, 1}}, cnt: 0x20009020, wr: []mcaWrite{}},
	0x562a50: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20009024, 1}}, cnt: 0x20009028, wr: []mcaWrite{}},
	0x5471d0: {ret: 0x0, retN: 0, rec: [][4]uint32{{2, 4, 0x2000902c, 1}}, cnt: 0x20009030, wr: []mcaWrite{}},
	0x58aad0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20009034, 2}, {2, 4, 0x2000903c, 2}, {3, 4, 0x20009044, 2}}, cnt: 0x2000904c, wr: []mcaWrite{}},
	0x58b410: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 1, 0x20009050, 2}, {3, 4, 0x20009054, 2}}, cnt: 0x2000905c, wr: []mcaWrite{}},
	0x4e79a0: {ret: 0x20009060, retN: 3, rec: [][4]uint32{{1, 4, 0x2000906c, 3}}, cnt: 0x20009078, wr: []mcaWrite{}},
	0x53d680: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x2000907c, 2}}, cnt: 0x20009084, wr: []mcaWrite{}},
	0x4ec1e0: {ret: 0x20009088, retN: 0, rec: [][4]uint32{}, cnt: 0x2000908c, wr: []mcaWrite{}},
	0x523330: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20009090, 1}, {2, 4, 0x20009094, 1}, {3, 4, 0x20009098, 1}}, cnt: 0x2000909c, wr: []mcaWrite{}},
	0x519ac0: {ret: 0x200090a0, retN: 2, rec: [][4]uint32{{1, 4, 0x200090a8, 2}, {2, 4, 0x200090b0, 2}}, cnt: 0x200090b8, wr: []mcaWrite{}},
	0x51b0a0: {ret: 0x200090bc, retN: 0, rec: [][4]uint32{{1, 4, 0x200090c0, 1}, {2, 1, 0x200090c4, 1}}, cnt: 0x200090c8, wr: []mcaWrite{}},
	0x547310: {ret: 0x200090cc, retN: 2, rec: [][4]uint32{{1, 4, 0x200090d4, 2}, {2, 4, 0x200090dc, 2}, {3, 4, 0x200090e4, 2}, {4, 4, 0x200090ec, 2}}, cnt: 0x200090f4, wr: []mcaWrite{}},
	0x53d870: {ret: 0x200090f8, retN: 0, rec: [][4]uint32{{1, 4, 0x200090fc, 1}, {2, 4, 0x20009100, 1}}, cnt: 0x20009104, wr: []mcaWrite{}},
	0x4ebce0: {ret: 0x20009108, retN: 0, rec: [][4]uint32{{1, 4, 0x2000910c, 1}}, cnt: 0x20009110, wr: []mcaWrite{}},
	0x547cb0: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x20009118, 1}, {3, 4, 0x2000911c, 1}}, cnt: 0x20009120, wr: []mcaWrite{{2, 0x0, 4, 0x20009114, 0}}},
	0x545aa0: {ret: 0x20009124, retN: 0, rec: [][4]uint32{{1, 4, 0x20009128, 1}}, cnt: 0x2000912c, wr: []mcaWrite{}},
	0x547fe0: {ret: 0x20009130, retN: 0, rec: [][4]uint32{{1, 4, 0x20009134, 1}}, cnt: 0x20009138, wr: []mcaWrite{}},
	0x4fa4d0: {ret: 0x2000913c, retN: 26, rec: [][4]uint32{{1, 4, 0x2000920c, 26}, {2, 4, 0x20009274, 26}, {3, 4, 0x200092dc, 26}, {5, 4, 0x20009344, 26}}, cnt: 0x200093ac, wr: []mcaWrite{{4, 0x0, 4, 0x200091a4, 26}}},
	0x51dde0: {ret: 0x200093b0, retN: 0, rec: [][4]uint32{{1, 4, 0x200093b8, 1}, {2, 4, 0x200093bc, 1}}, cnt: 0x200093c0, wr: []mcaWrite{{3, 0x0, 4, 0x200093b4, 0}}},
	0x52b2f0: {ret: 0x200093c4, retN: 0, rec: [][4]uint32{{1, 4, 0x200093c8, 1}, {2, 4, 0x200093cc, 1}}, cnt: 0x200093d0, wr: []mcaWrite{}},
	0x52af50: {ret: 0x0, retN: 0, rec: [][4]uint32{{1, 4, 0x200093d4, 1}, {2, 4, 0x200093d8, 1}, {3, 4, 0x200093dc, 1}, {4, 4, 0x200093e0, 1}}, cnt: 0x200093e4, wr: []mcaWrite{}},
}

// cpaPark4e79a0 is the first boundary row of re/emu/cases/004e79a0.json (pc action eligibility).
var cpaPark4e79a0 = [264]int64{0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 1, 1, 536936448, 0, 1, 16, 0, 1, 2, 0, -1, -1, 1, 0, 1, 0, 0, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 0, 0, 0, 0, 100, 20, 0, 1, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 5, 1, 10, 10, 10, 10, 10, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 65535, 4294967295, 3, 255, 0, 255, 0, 255, 0, 255, 0, 1, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 1, 1, 1, 0, 0, 0, 0, 0, 0}

// cpaPark4ebce0 is the first boundary row of re/emu/cases/004ebce0.json (breath target prepare).
var cpaPark4ebce0 = [89]int64{0, 0, 0, 1065353216, 0, 0, 0, 0, 0, 0, 0, 5, 1065353216, 0, 2, 7, 0, 123, 456, 1, 1, 1, 10, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 0, 1, 1065353216, 0, 1073741824, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// cpaPark4ec1e0 is the first boundary row of re/emu/cases/004ec1e0.json (status action override).
var cpaPark4ec1e0 = [1282]int64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 100, 100, 100, 100, 100, 100, 100, 100, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 50, 50, 50, 50, 50, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// cpaPark4fa4d0 is the first boundary row of re/emu/cases/004fa4d0.json (character spell cast).
var cpaPark4fa4d0 = [728]int64{2, 16, 3, 0, 0, 0, 0, 0, 10, 0, 132102, 4, 0, 0, 0, 0, 1, 3, 0, 0, 3, 0, 0, 0, 40, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 30, 0, 0, 100, 100, 100, 100, 100, 100, 0, 0, 0, 0, 19, 19, 1073741824, 1073741824, 1073741824, 1073741824, 0, 0, 0, 0, 0, 0, 19, 19, 1073741824, 1073741824, 1073741824, 1073741824, 0, 0, 0, 0, 0, 0, 19, 19, 1073741824, 1073741824, 1073741824, 1073741824, 0, 0, 0, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 100, 0, 0, 4, 37, 37, 37, 37, 0, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 537067520, 537068032, 537068544, 537069056, 537069568, 537070080, 537070592, 537071104, 537071616, 537072128, 537072640, 537073152, 537073664, 537074176, 537074688, 537075200, 537075712, 537076224, 537076736, 537077248, 537077760, 537078272, 537078784, 537079296, 537079808, 537080320, 537080832, 537081344, 537081856, 537082368, 537082880, 537083392, 537083904, 537084416, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4611686018427387904, 0, 1073741824, 1073741824, 1073741824, 0, 0, 3, 0, 101, 16, 1073741824, 1073741824, 1073741824, 1, 5, 0, 5, 1073741824, 1073741824, 1073741824, 0, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1, 0, 100, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 3, 0, 100, 16, 1073741824, 1073741824, 1073741824, 1073741824, 101, 100, 100, 100, 100, 100, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 5, 100, 100, 100, 100, 100, 100, 0, 0, 0, 0, 4611686018427387904, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 537100544, 537100544, 537100544, 537100544, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 3, 0, 0, 0, 24, 24, 24, 24, 0, 0, 0, 0, 536877568, 536877568, 536877568, 536877568, 3, 1, 101, 16, 1073741824, 1073741824, 1073741824, 536899584, 0, 0, 0, 0, 0, 0, 50, 0, 0, 0, 0, 50, 0, 0, 0, 0, 0, 40, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 10, 0, 0, 0, 0}

// cpaPark51b0a0 is the first boundary row of re/emu/cases/0051b0a0.json (random party attack target).
var cpaPark51b0a0 = [74]int64{3, 255, 255, 1, 255, 1, 255, 255, 0, 1, 5, 2147483648, 5, 0, 1, 1, 2147483648, 2147483648, 0, 17, 19, 19, 0, 0, 0, 4294967295, 0, 4294967295, 1, 0, 5, 5, 5, 4294967295, 0, 255, 1, 0, 0, 0, 0, 0, 127, 255, 5, 0, 4, 0, 0, 2, 7, 3, 0, 9, 3, 536875008, 1, 2, 0, 3, 1, 2, 3, 5, 255, 0, 255, 0, 3, 3, 0, 1, 7, 2147483647}

// cpaPark51dde0 is the first boundary row of re/emu/cases/0051dde0.json (item use power selection).
var cpaPark51dde0 = [704]int64{536913604, 2, 1, 0, 16, 14, 2, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 30, 20, 20, 20, 20, 20, 20, 1, 10, 0, 0, 0, 0, 10, 0, 132102, 4, 0, 0, 0, 0, 1, 3, 0, 0, 3, 0, 0, 0, 40, 0, 0, 0, 0, 5, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 30, 0, 0, 100, 100, 100, 100, 100, 100, 0, 0, 0, 0, 19, 19, 1073741824, 1073741824, 1073741824, 1073741824, 0, 0, 0, 0, 0, 0, 19, 19, 1073741824, 1073741824, 1073741824, 1073741824, 0, 0, 0, 0, 0, 0, 19, 19, 1073741824, 1073741824, 1073741824, 1073741824, 0, 0, 0, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 100, 0, 0, 4, 37, 37, 37, 37, 0, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 537067520, 537068032, 537068544, 537069056, 537069568, 537070080, 537070592, 537071104, 537071616, 537072128, 537072640, 537073152, 537073664, 537074176, 537074688, 537075200, 537075712, 537076224, 537076736, 537077248, 537077760, 537078272, 537078784, 537079296, 537079808, 537080320, 537080832, 537081344, 537081856, 537082368, 537082880, 537083392, 537083904, 537084416, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4611686018427387904, 0, 1073741824, 1073741824, 1073741824, 0, 0, 3, 0, 101, 16, 1073741824, 1073741824, 1073741824, 1, 5, 0, 5, 1073741824, 1073741824, 1073741824, 0, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1, 0, 100, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 1073741824, 3, 0, 100, 16, 1073741824, 1073741824, 1073741824, 1073741824, 101, 100, 100, 100, 100, 100, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 5, 100, 100, 100, 100, 100, 100, 0, 0, 0, 0, 4611686018427387904, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 537100544, 537100544, 537100544, 537100544, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 24, 24, 24, 24, 0, 0, 0, 0, 536877568, 536877568, 536877568, 536877568, 537135632, 3, 1, 101, 16, 1073741824, 1073741824, 1073741824, 536899584, 1, 2, 0, 0, 68, 0, 2, 0, 1, 256, 0, 0, 0, 0, 7, 0, 0, 1, 1, 0, 1, 1, 537100288, 10, 30, 4660, 0, 0, 119, 1, 1, 2, 386912, 349169, 0, 26224, 26225, 908489270, 1735213790, 1689, 2730260571, 210, 1, 174, 5, 242, 11, 209, 4, 201, 1168149043, 372769985, 1913131460, 1032240212, 1665952873, 2328344840, 2256922788, 2608373793, 904456312, 2297930258, 2638112178, 2858682718, 1589125630, 2417470147, 1492668486, 1384987301, 887667169, 99215872, 3522104622, 4258980395, 1876226658, 2115696419, 2763734627, 3780439323, 779167982, 3227229097, 3497940901, 1994567726, 3291518044, 786790051, 3403873696, 3665256095, 2764207923, 2789514912, 4205687869, 2310320242, 3414908137, 1943514022, 4200716945, 1042766289, 1582403645, 3881670847, 217}

// cpaPark523330 is the first boundary row of re/emu/cases/00523330.json (character condition remove).
var cpaPark523330 = [408]int64{536870912, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 536870912, 0, 0, 536956928, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0, 255, 255, 255, 0}

// cpaPark52af50 is the first boundary row of re/emu/cases/0052af50.json (character fatigue apply).
var cpaPark52af50 = [20]int64{0, 10, 1, 536936448, 0, 1, 1000, 2, 0, 0, 0, 0, 0, 5, 286331153, 572662306, 858993459, 0, 0, 0}

// cpaPark52b2f0 is the first boundary row of re/emu/cases/0052b2f0.json (character action fatigue cost).
var cpaPark52b2f0 = [18]int64{0, 0, 0, 0, 0, 0, 0, 0, -1, -1, -1, -1, 0, 0, 0, 0, 1000, 0}

// cpaPark53d680 is the first boundary row of re/emu/cases/0053d680.json (attack budget prepare).
var cpaPark53d680 = [20]int64{1, 1, 0, 0, -2147483648, 3, 0, 0, -2147483648, 2, 3, 0, 0, 7, 9, 11, 12, 13, 14, 15}

// cpaPark53d870 is the first boundary row of re/emu/cases/0053d870.json (attack swings).
var cpaPark53d870 = [96]int64{0, -1, 0, 0, 2, 2, 0, 1, 1, 1, 1, 1, 1, 3, 3, 1, 1, 0, 1, 0, 0, 0, 5, 1, 0, 1, 4, -1, -1, 0, 0, 2, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 10, 0, 0, 0, 20, 0, 0, 0, 1, 1, 0, 0, 1, 100, 0, 1, 1, 1, 1, 0, 0, 1092616192, 0, 1092616192, 1094713344, 0, 1094713344, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// cpaPark545aa0 is the first boundary row of re/emu/cases/00545aa0.json (guard attack available).
var cpaPark545aa0 = [112]int64{3, 1, 0, 1, 100, 1, 0, 1, 0, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 1, 0, 2, 3, 4, 5, 0, 1, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 1, 100, 0, 6, 0, 7, 0, 0, 0, 0, 0}

// cpaPark547310 is the first boundary row of re/emu/cases/00547310.json (offensive action allowed).
var cpaPark547310 = [81]int64{0, 5, 5, 0, 0, 0, 0, 65535, 4294967295, 3, 255, 0, 255, 0, 255, 0, 255, 0, 6, 1, 4, 4, 4, 0, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 255, 0, 0, 0, 1, 536877056, 536870912, 700, 5}

// cpaPark547cb0 is the first boundary row of re/emu/cases/00547cb0.json (turn undead power).
var cpaPark547cb0 = [143]int64{3, 536953856, 0, 0, 0, 2, 1, 3, 4096, 4113, 4130, 4147, 4164, 4181, 4198, 4215, 0, 2004318071, 90, 0, 0, 0, -1, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 19, 19, 19, 0, 0, 0, 0, 0, 0, 0, 0, 1065353216, 0, 0, 0, 0, 1065353216, 0, 2, 1, 1, 1, 10, 0, 0, 0, 1, 1, 0, 19, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 0, 1, 1065353216, 0, 1073741824}

// cpaPark547fe0 is the first boundary row of re/emu/cases/00547fe0.json (battle prayer).
var cpaPark547fe0 = [414]int64{0, 536928256, 0, 10, 0, 0, 0, 3, 6, 9, 12, 15, 18, 21, 24, 27, 30, 9, 36, 39, 42, 45, 0, 1, 0, 20, 20, 30, 30, 0, 0, 1, 65, 1, 0, 20, 20, 30, 30, 0, 0, 1, 66, 1, 0, 20, 20, 30, 30, 0, 0, 1, 67, 1, 0, 20, 20, 30, 30, 0, 0, 1, 68, 1, 0, 20, 20, 30, 30, 0, 0, 1, 69, 1, 0, 20, 20, 30, 30, 0, 0, 1, 70, 1, 0, 20, 20, 30, 30, 0, 0, 1, 71, 1, 0, 20, 20, 30, 30, 0, 0, 1, 72, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1, 1, 0, 0, 0, 0, 1000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 77, 1, 1, 1, 10, 0, 0, 0, 0, 0, 78, 1, 1, 1, 10, 0, 0, 0, 0, 0, 79, 1, 1, 1, 10, 0, 0, 0, 0, 0, 80, 1, 1, 1, 10, 0, 0, 0, 0, 0, 81, 1, 1, 1, 10, 0, 0, 0, 0, 0, 82, 1, 1, 1, 10, 0, 0, 0, 0, 0, 83, 1, 1, 1, 10, 0, 0, 0, 0, 0, 84, 1, 1, 1, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 3, 5, 0, 0, 0, 0, 0, 0, 0, 1065353216, 0, 0, 0, 0, 1065353216, 0, 2, 1, 1, 1, 10, 0, 0, 0, 1, 1, 0, 19, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 1, 10, 0, 0, 0, 1, 1065353216, 0, 1073741824}

// cpaPark519ac0 is the first boundary row of re/emu/cases/00519ac0.json (char attack range).
var cpaPark519ac0 = [12]int64{0, 1, 0, 0, 0, 0, 0, 1, 0, 1, 3, 255}
