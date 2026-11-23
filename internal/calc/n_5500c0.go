package calc

import (
	"fmt"
	"math"
)

// summonSelectionNode is 0x5500c0 summon_selection(effect), cdecl, no return
// value: the summon spell's effect record P picks a summon category weighted
// by the caster's four summoning skills, spawns the category's monster group
// for the effect power, gives it the caster's side and, with a chance, a
// bonus buff on the summoned leader.
//
// Flow (exe addresses):
//   - caster: 0x53bea0(P+0x5c, 0) is "character" iff [P+0x5c] == 1 (asserts on
//     [P+0x60] == -1); 0x53bf10(P+0x5c, 0) is "monster" iff [P+0x5c] == 2
//     (asserts on [P+0x64] == -1).
//   - character whose message flag 0x5248a0(idx, 0) (byte C+0x181f) is set:
//     byte 0x68510c set -> 0x590950(idx, [[0x68c09c]+0x6a0]); else
//     0x5905f0(0x619788, -1), 0x5905c0(1, -1), 0x5905f0([[0x68c09c]+0x6a0], -1).
//     Then P+0x124 = 1 and return (0x550119 / 0x550154).
//   - monster caster mon = 0x4e5840([P+0x64]); mon+0x344 != -1 returns at once
//     (0x550197).
//   - weights 0x14 each (0x5501a1); a character adds C+0x5cb/+0x5f1/+0x617/+0x63d
//     (0x5501ea); byte 0x652da4 zeroes weight 0 (0x55022b); r = 0x40efa0(sum);
//     category = first i with r < w[i] (unsigned), else r -= w[i] (0x550246).
//     Falling out of the loop (0x550257) needs r >= sum: unreachable.
//   - side bl: character (C+0xa35 == 0) + 1 (0x550298); monster byte mon+0x16;
//     else 1; byte P+0x78 flips it to (bl == 1) + 1 (0x5502cb).
//   - grp = 0x50f1a0(dword 0x616f4c[cat + 4*[P+0xd0]], 1, P+0xa0, 1, 0, 1); NULL
//     asserts (0x55030c). rec = 0x4e5840(grp+0x9f).
//   - facing: 0x54ff20(f32 P+0xa0, P+0xa4, P+0xa8, bl dword, word [rec]) leaves
//     ST0; stored as f32 and passed to 0x4538f0 (this [rec+0xc]+0x18).
//   - 0x547570(grp, bl, 0); every member r (0x5e2c80(grp+8, i), i < grp+4) with
//     r+0x2da == 0 gets r+0x2da = (bl == 2 ? 1 : 2) and 0x4acd80([r+0xc], 0x27, 1).
//   - 0x505810(); character: 0x524780([P+0x60], 0, [rec]); monster:
//     mon+0x344 = [rec] (0x5504ea).
//   - bonus: [P+0xd4] != 0 and 0x40efa0(100) < [P+0xd4] (unsigned): jump table
//     0x550770[cat]. 0x550651 (cat 0, 1): spell s = 0x38 / 0x3d, duration from the
//     spell record, effect id from the byte table 0x5507a0[s-0x13] and jump table
//     0x550780 (each target starts with opcode byte 0xb8 and the id dword; 0x550718 asserts),
//     0x5242b0([rec], effect, power, duration, 0). 0x550541 (cat 2, 3): s = 0x1a /
//     0x20, slot = index of s in the dwords 0x616d84..0x616db4 (absent:
//     0x55058d, unreachable for these two), slot record rec+0x10f+0x11*slot; a
//     slot not holding s gets 0x4acd80([rec+0xc], dword 0x60cffc[8*s], 1); the
//     slot becomes {1, s, power, duration at +0xd}; 0x50e8c0([rec]).
//   - duration: d = per*power + base (spell record [0x65be1c]+0x1bf*s, +0x44 and
//     +0x4d, u32); d != 0x270f: d+1, k = 0x40efa0(4), k == 0 and d > 1 or k == 1
//     and d <= 2: d+1; 0x517f90(&d, pct) with pct = +0x24 of the struct 0x543260
//     zeroes (12 dwords), so it leaves d unchanged.
//   - 0x511ce0(grp, 0); 0x50f720(grp); P+0x125 = 1.
//
// Native helpers modelled here: 0x590950 (message text; vswprintf of the
// effect's format with no arguments, the name length plus one for a leading '
// and one more for a following s, 0x58b410 slot byte S+0xf1), 0x5905c0,
// 0x5248a0, 0x53bea0, 0x53bf10, 0x543260, 0x517f90, 0x50e8c0
// (monsterCacheRebuild, n_523c00.go), and through gsRun (n_556b10.go) 0x547570,
// 0x505810, 0x511ce0, 0x50f720 and the record/group lookups.
//
// Emu (what TestEmu replays) is re/emu/cases/005500c0.json: the D4 callees
// 0x50f1a0, 0x5242b0, 0x524780, 0x5049c0, 0x53bf80, 0x546e70, 0x538e00, 0x4eb7f0,
// 0x531540, 0x547510 and the excluded 0x54ff20, 0x4538f0, 0x4acd80, 0x5905f0,
// 0x58aad0, 0x58b410, 0x593330, 0x4e4390, 0x52e5c0 are address stubs; call
// counts and recorded arguments are outputs, returns (spawned group, ST0,
// 0x547510) are case inputs. 0x58aad0 records arguments 1..3 only (0x590950
// passes two stack buffer addresses as 4 and 5); 0x54ff20's fourth argument is
// recorded as its low byte (the upper bytes are uninitialised stack).
//
// F runs the real D4 callees monsterGroupSpawnNode (written back to the
// fixture group as in 0x556b10's F, members are the fixture records of member
// slots 0..count-1) and combatOpeningConditionNode on the F-only inputs.
// 0x5242b0 (monsterEffectApplyNode, writes rec+0xa7 effect records nothing here
// reads afterwards) and 0x524780 are counted and recorded in F as in Emu.
//
// Fixture caps (calc panics past them): character index 0..7, monster ids 1..8,
// member count 0..4, template ids of the cache 0x6840c7 (0..0x3f, 0x94..0x99),
// '%' in the message format, the spawned group is the fixture group, a spawned
// member that is the monster caster (F). Every assert path panics.
const (
	smnEff    = gsArena
	smnGrp    = gsArena + 0x200 // g1, g2, g3 at +0x100 each
	smnML     = gsArena + 0x600
	smnGLN    = gsArena + 0x680
	smnGLS    = gsArena + 0x6c0
	smnNL     = gsArena + 0x700
	smnSL     = gsArena + 0x780
	smnFmt    = gsArena + 0x800
	smnRec    = gsArena + 0x1000 // + 0x400*k, id k+1, template id k
	smnVis    = gsArena + 0x3000 // + 0x200*k
	smnSt     = gsArena + 0x4000 // + 0x100*k
	smnCr     = gsArena + 0x4800 // + 0x200*k, record +0x10
	smnTpl    = gsArena + 0x6000 // + 0x400*s
	smnT      = gsArena + 0x7000
	smnCmb    = gsArena + 0xb000
	smnUI     = gsArena + 0xbc00
	smnOpt    = gsArena + 0xbd00
	smnFOnly  = gsArena + 0xd000
	smnSpells = gsArena + 0xe000
	smnChr    = gsArena + 0x15000 // + 0x1862*idx
	smnPC     = gsArena + 0x22000 // + 0x106*idx
)

var smnCnt = []int64{0x50f1a0, 0x54ff20, 0x4538f0, 0x4acd80, 0x5242b0, 0x524780, 0x5905f0, 0x58aad0, 0x58b410,
	0x5049c0, 0x593330, 0x53bf80, 0x546e70, 0x538e00, 0x4eb7f0, 0x531540, 0x547510, 0x4e4390, 0x52e5c0}

// smnRecSlot sizes: 0x54ff20 argument 4 is a byte slot, the rest dwords.
var smnRecs = func() (r []gsRecSlot) {
	span := func(callee, at, n int64) {
		for a := int64(1); a <= n; a++ {
			r = append(r, gsRecSlot{callee, a, at + 4*(a-1), 0})
		}
	}
	span(0x50f1a0, 0x100, 6)
	span(0x54ff20, 0x120, 5) // arg 4 at 0x12c is recorded as a byte (0x12d..0x12f stay 0)
	r = append(r, gsRecSlot{0x4538f0, 1, 0x140, 0},
		gsRecSlot{0x4acd80, 1, 0x150, 5}, gsRecSlot{0x4acd80, 2, 0x170, 5}, gsRecSlot{0x4acd80, 3, 0x190, 5})
	span(0x5242b0, 0x1b0, 5)
	span(0x524780, 0x1d0, 3)
	r = append(r, gsRecSlot{0x5905f0, 1, 0x1e0, 2}, gsRecSlot{0x5905f0, 2, 0x1e8, 2})
	span(0x58aad0, 0x1f0, 3)
	span(0x58b410, 0x200, 4)
	r = append(r, gsRecSlot{0x5049c0, 1, 0x210, 12}, gsRecSlot{0x5049c0, 2, 0x240, 12},
		gsRecSlot{0x4eb7f0, 1, 0x270, 4}, gsRecSlot{0x531540, 1, 0x280, 4}, gsRecSlot{0x4e4390, 1, 0x290, 4})
	span(0x52e5c0, 0x2a0, 4)
	return
}()

type smnField struct {
	name   string
	size   int
	at     int64
	signed bool
	index  string // input naming the record index (at + stride*index); < 0: not written
	stride int64
	count  int64
	also   []int64
}

var smnSpec = func() (f []smnField) {
	add := func(n string, size int, at int64) { f = append(f, smnField{name: n, size: size, at: at}) }
	sadd := func(n string, size int, at int64) { f = append(f, smnField{name: n, size: size, at: at, signed: true}) }
	iadd := func(n string, size int, at int64, ix string, stride, count int64) {
		f = append(f, smnField{name: n, size: size, at: at, index: ix, stride: stride, count: count})
	}
	add("casterType", 4, smnEff+0x5c)
	sadd("charIdx", 4, smnEff+0x60)
	sadd("casterMonId", 4, smnEff+0x64)
	add("reverse", 1, smnEff+0x78)
	for i, n := range []string{"posX", "posY", "posZ"} {
		add(n, 4, smnEff+0xa0+4*int64(i))
	}
	add("power", 4, smnEff+0xd0)
	add("bonusChance", 4, smnEff+0xd4)
	for i, off := range []int64{0x5cb, 0x5f1, 0x617, 0x63d} {
		iadd(fmt.Sprint("skill", i), 4, smnChr+off, "charIdx", 0x1862, 8)
	}
	iadd("chrA35", 4, smnChr+0xa35, "charIdx", 0x1862, 8)
	iadd("chrMessage", 1, smnChr+0x181f, "charIdx", 0x1862, 8)
	iadd("pcSlot", 1, smnPC+0xf1, "charIdx", 0x106, 8)
	add("messageMode", 1, 0x68510c)
	add("noFirstCategory", 1, 0x652da4)
	add("fmt0", 2, smnFmt)
	add("fmt1", 2, smnFmt+2)
	add("suffix0", 2, 0x689b34)
	f = append(f, smnField{name: "memberCount", size: 4, at: smnML + 8, also: []int64{smnGrp + 4}})
	for k := int64(0); k < 4; k++ {
		add(fmt.Sprint("member", k), 4, smnML+0x10+4*k)
	}
	for _, x := range []struct {
		n    string
		off  int64
		size int
	}{{"grpSpecies", 0x18, 4}, {"grpActive", 0x28, 1}, {"grpRemoved", 0x29, 1}, {"grpDisposition", 0x2a, 1},
		{"grpIdentified", 0x2c, 1}, {"grpMonsterId", 0x9f, 4}, {"grpAlert", 0xc8, 1}, {"grpAlertTicks", 0xc9, 1}} {
		add(x.n, x.size, smnGrp+x.off)
	}
	for k := int64(1); k <= 2; k++ {
		gk := smnGrp + 0x100*k
		add(fmt.Sprintf("g%dCount", k), 4, gk+4)
		add(fmt.Sprintf("g%dActive", k), 1, gk+0x28)
		add(fmt.Sprintf("g%dRemoved", k), 1, gk+0x29)
		add(fmt.Sprintf("g%dDisposition", k), 1, gk+0x2a)
	}
	for k := int64(0); k < 8; k++ {
		r, v, s, c := smnRec+0x400*k, smnVis+0x200*k, smnSt+0x100*k, smnCr+0x200*k
		add(fmt.Sprintf("rec%dPresent", k), 1, r+0x14)
		add(fmt.Sprintf("rec%dAwake", k), 1, r+0x15)
		add(fmt.Sprintf("rec%dDisposition", k), 1, r+0x16)
		add(fmt.Sprintf("rec%dHp", k), 4, r+0x2b)
		add(fmt.Sprintf("rec%dPhase", k), 4, r+0x107)
		add(fmt.Sprintf("rec%dHostile", k), 1, r+0x28a)
		add(fmt.Sprintf("rec%dTarget", k), 4, r+0x2e1)
		add(fmt.Sprintf("rec%dSummoned", k), 4, r+0x2da)
		sadd(fmt.Sprintf("rec%dOwner", k), 4, r+0x344)
		add(fmt.Sprintf("vis%dFrozen", k), 1, v+0x332)
		add(fmt.Sprintf("vis%dBusy", k), 1, v+0xb4)
		add(fmt.Sprintf("vis%dFlags", k), 4, v+0x1dc)
		add(fmt.Sprintf("st%dState", k), 1, s+0xa4)
		add(fmt.Sprintf("st%dState2", k), 1, s+0xa7)
		add(fmt.Sprintf("cr%dCmd", k), 4, c)
		add(fmt.Sprintf("cr%dBusy", k), 1, c+4)
	}
	// the spawned leader's record (index grpMonsterId: record id-1)
	lead, leadCr := int64(smnRec-0x400), int64(smnCr-0x200)
	for _, sl := range []int64{0, 6} {
		e := lead + 0x10f + 0x11*sl
		for _, x := range []struct {
			n    string
			off  int64
			size int
		}{{"Flag", 0, 1}, {"Spell", 1, 4}, {"Power", 5, 4}, {"Mid", 9, 4}, {"Dur", 0xd, 4}} {
			iadd(fmt.Sprintf("leadSlot%d%s", sl, x.n), x.size, e+x.off, "grpMonsterId", 0x400, 9)
		}
	}
	for _, x := range []struct {
		n   string
		off int64
	}{{"leadRule27", 0x27}, {"leadRule2f", 0x2f}, {"leadRule10b", 0x10b}} {
		iadd(x.n, 4, lead+x.off, "grpMonsterId", 0x400, 9)
	}
	for _, x := range []struct {
		n    string
		off  int64
		size int
	}{{"leadBuffFlag", 0x3e, 1}, {"leadBuffSpell", 0x3f, 4}, {"leadBuffPower", 0x43, 1}, {"leadAuraFlag", 0xd7, 1},
		{"leadAuraSpell", 0xd8, 4}, {"leadAuraPower", 0xdc, 1}, {"leadAuraMult", 0xe0, 4}} {
		iadd(x.n, x.size, leadCr+x.off, "grpMonsterId", 0x200, 9)
	}
	add("combatActive", 1, 0x683f94)
	add("reportGate", 1, 0x687500)
	add("frozen", 1, 0x68f105)
	add("mode", 4, 0x68ec78)
	add("worldTime", 4, 0x686a48)
	add("curChar", 4, 0x6875ef)
	add("uiFlags", 4, smnUI+0xf4)
	add("combatRound", 4, smnCmb+4)
	add("combatPhaseMode", 4, smnCmb+0x7b0)
	add("combatActor", 4, smnCmb+0x7b8)
	add("combatOpeningBlocked", 1, smnCmb+0xa54)
	add("optPaused", 1, smnOpt+0x9c)
	for s := int64(0); s < 3; s++ {
		t := smnTpl + 0x400*s
		add(fmt.Sprintf("tpl%dNameId", s), 2, t+0x187)
		add(fmt.Sprintf("tpl%dNoRemove", s), 1, t+0x24a)
		for i := int64(0); i < 5; i++ {
			add(fmt.Sprintf("tpl%dAttr%d", s, i), 1, t+0xd1+i)
		}
	}
	add("spawnRet", 4, gsStub+0x80)
	add("facingBits", 8, gsStub+0x88)
	for k := int64(0); k < 3; k++ {
		add(fmt.Sprint("openRet", k), 4, gsStub+0xb4+4*k)
	}
	for _, s := range []int64{0x38, 0x3d, 0x20, 0x1a} {
		add(fmt.Sprintf("spell%xPerPower", s), 4, smnSpells+0x1bf*s+0x44)
		add(fmt.Sprintf("spell%xBase", s), 4, smnSpells+0x1bf*s+0x4d)
	}
	// F-only: read by the real 0x50f1a0 in F, never by the stubbed emulator run.
	for s := int64(0); s < 3; s++ {
		t := smnTpl + 0x400*s
		for _, x := range []struct {
			n    string
			off  int64
			size int
		}{{"hpBase", 0xd6, 2}, {"hpDice", 0xd8, 1}, {"hpSides", 0xd9, 1}, {"spBase", 0xda, 2}, {"spDice", 0xdc, 1},
			{"spSides", 0xdd, 1}, {"special", 0x26a, 1}, {"templateFlags", 0xd0, 1}} {
			add(fmt.Sprint(x.n, s), x.size, t+x.off)
		}
	}
	add("nextMonsterId", 4, 0x6874be)
	add("nextGroupId", 4, 0x6874ba)
	add("reportEnabled", 1, 0x689b32)
	for i, n := range []string{"battleFlags", "animationFrames", "unbornCount"} {
		add(n, 4, smnFOnly+4*int64(i))
	}
	return
}()

var smnOuts = func() (o []smnField) {
	add := func(n string, size int, at int64) { o = append(o, smnField{name: n, size: size, at: at}) }
	for i, a := range smnCnt {
		add(fmt.Sprintf("calls%x", a), 4, gsStub+4*int64(i))
	}
	for _, r := range smnRecs {
		size := 4
		if r.callee == 0x54ff20 && r.arg == 4 {
			size = 1
		}
		if r.n == 0 {
			add(fmt.Sprintf("arg%x_%d", r.callee, r.arg), size, gsStub+r.at)
			continue
		}
		for k := int64(0); k < r.n; k++ {
			add(fmt.Sprintf("arg%x_%d_%d", r.callee, r.arg, k), size, gsStub+r.at+4*k)
		}
	}
	add("messageShown", 1, smnEff+0x124)
	add("summonDone", 1, smnEff+0x125)
	add("messageOverlay", 1, 0x69b7b8)
	for _, x := range []struct {
		n    string
		off  int64
		size int
	}{{"grpDispositionAfter", 0x2a, 1}, {"grpCaFlag", 0xca, 1}, {"grpDispositionTime", 0xcb, 4},
		{"grpRemovedAfter", 0x29, 1}, {"grpAlertAfter", 0xc8, 1}, {"grpAlertTicksAfter", 0xc9, 1}} {
		add(x.n, x.size, smnGrp+x.off)
	}
	for k := int64(0); k < 8; k++ {
		r := smnRec + 0x400*k
		add(fmt.Sprintf("rec%dDispositionAfter", k), 1, r+0x16)
		add(fmt.Sprintf("rec%dTargetAfter", k), 4, r+0x2e1)
		add(fmt.Sprintf("rec%dSummonedAfter", k), 4, r+0x2da)
		o = append(o, smnField{name: fmt.Sprintf("rec%dOwnerAfter", k), size: 4, at: r + 0x344, signed: true})
		add(fmt.Sprintf("cr%dCmdAfter", k), 4, smnCr+0x200*k)
		add(fmt.Sprintf("vis%dBusyAfter", k), 1, smnVis+0x200*k+0xb4)
		for _, sl := range []int64{0, 6} {
			e := r + 0x10f + 0x11*sl
			for _, x := range []struct {
				n    string
				off  int64
				size int
			}{{"Flag", 0, 1}, {"Spell", 1, 4}, {"Power", 5, 4}, {"Dur", 0xd, 4}} {
				add(fmt.Sprintf("rec%dSlot%d%sAfter", k, sl, x.n), x.size, e+x.off)
			}
		}
		for i := int64(0); i < 26; i++ {
			add(fmt.Sprintf("rec%dModifier%d", k, i), 4, r+0x1db+4*i)
		}
		add(fmt.Sprintf("rec%dRecovery", k), 4, r+0x247)
		add(fmt.Sprintf("rec%dRecovery4", k), 1, r+0x24b)
		add(fmt.Sprintf("rec%dAttr47", k), 4, r+0x47)
		add(fmt.Sprintf("rec%dAttr4f", k), 4, r+0x4f)
	}
	add("uiFlagsAfter", 4, smnUI+0xf4)
	for k := int64(0); k < 8; k++ {
		add(fmt.Sprint("nameBuf", k), 2, gsName+2*k)
	}
	return
}()

func init() {
	var in, out []string
	for _, f := range smnSpec {
		in = append(in, f.name)
	}
	for _, f := range smnOuts {
		out = append(out, f.name)
	}
	Nodes["005500c0"] = Node{Addr: "005500c0", In: in, Out: out, F: summonSelectionNode,
		Emu: func(in []int64, r Rand) []int64 { return smnNode(in, r, false) }}
}

func summonSelectionNode(in []int64, r Rand) []int64 { return smnNode(in, r, true) }

func smnNode(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(smnSpec) {
		panic("summon selection: wrong input count")
	}
	w := &cgW{m: map[int64]byte{}}
	smnSetup(w)
	named := map[string]int64{}
	for i, f := range smnSpec {
		named[f.name] = in[i]
	}
	for i, f := range smnSpec {
		at := f.at
		if f.index != "" {
			k := named[f.index]
			if k < 0 {
				continue
			}
			if k >= f.count {
				panic(fmt.Sprintf("summon selection: %s=%d outside the fixture's %d records", f.index, k, f.count))
			}
			at += f.stride * k
		}
		w.wr(at, f.size, in[i])
		for _, a := range f.also {
			w.wr(a, f.size, in[i])
		}
	}
	e := &gsRun{w: w, r: r, real: real, cnt: smnCnt, recs: smnRecs}
	smnRun(e)
	var out []int64
	for _, o := range smnOuts {
		if o.signed {
			out = append(out, w.rs(o.at, o.size))
		} else {
			out = append(out, w.rd(o.at, o.size))
		}
	}
	return out
}

// smnSetup is the case's setup.
func smnSetup(w *cgW) {
	for _, x := range [][2]int64{{0x683fad, smnNL}, {0x683fb9, smnSL}, {0x683fb1, smnGLN}, {0x683fbd, smnGLS},
		{0x68c09c, smnT}, {0x6836a8, smnCmb}, {0x68edcc, smnUI}, {0x659c0c, smnOpt}, {0x685174, smnChr},
		{0x685178, smnPC}, {0x65be1c, smnSpells}} {
		w.p32(x[0], x[1])
	}
	for _, l := range []struct {
		obj   int64
		items []int64
	}{{smnNL, []int64{smnRec, smnRec + 0x400, smnRec + 0x800, smnRec + 0xc00, smnRec + 0x1000, smnRec + 0x1400}},
		{smnSL, []int64{smnRec + 0x1800, smnRec + 0x1c00}},
		{smnGLN, []int64{smnGrp, smnGrp + 0x100, smnGrp + 0x200}},
		{smnGLS, []int64{smnGrp + 0x300}}} {
		w.p32(l.obj, l.obj+0x10)
		w.p32(l.obj+4, 8)
		w.p32(l.obj+8, int64(len(l.items)))
		for i := int64(0); i < 8; i++ {
			v := 0xdead0100 + i
			if i < int64(len(l.items)) {
				v = l.items[i]
			}
			w.p32(l.obj+0x10+4*i, v)
		}
	}
	w.p32(smnML, smnML+0x10)
	w.p32(smnML+4, 8)
	for i := int64(4); i < 8; i++ {
		w.p32(smnML+0x10+4*i, 9+i)
	}
	w.p32(smnGrp+8, smnML)
	for k := int64(0); k < 4; k++ {
		w.p32(smnGrp+0x100*k, 0x51+k)
	}
	w.p8(smnGrp+0xca, 0x5a)
	w.p32(smnGrp+0xcb, 0x77777777)
	for k := int64(0); k < 8; k++ {
		r := smnRec + 0x400*k
		w.p32(r, k+1)
		w.p32(r+8, k)
		w.p32(r+0xc, smnVis+0x200*k)
		w.p32(r+0x10, smnCr+0x200*k)
		w.p32(smnVis+0x200*k+0x1d8, smnSt+0x100*k)
		for _, s := range []int64{0, 6} { // decoys: only the leader's slots are case inputs
			e := r + 0x10f + 0x11*s
			w.p8(e, 1)
			w.p32(e+1, 0x60+k)
			w.p32(e+5, 0x30+k)
			w.p32(e+9, 0x40+k)
			w.p32(e+0xd, 0x50+k)
		}
		w.p32(r+0x27, 3+k)
		w.p32(r+0x2f, 5+k)
		w.p32(r+0x10b, 7+k)
		c := smnCr + 0x200*k
		w.p8(c+0x3e, 1)
		w.p32(c+0x3f, 0x31)
		w.p8(c+0x43, 2+k)
		w.p8(c+0xd7, 1)
		w.p32(c+0xd8, 0x35)
		w.p8(c+0xdc, 1+k)
		w.p32(c+0xe0, 10*k)
	}
	for sp := int64(0); sp < 0x9a; sp++ {
		if sp < 0x40 || sp >= 0x94 {
			w.p32(0x6840c7+4*sp, smnTpl+0x400*(sp%3))
		}
	}
	for _, off := range []int64{0x75c, 0x760, 0x7fc, 0x800, 0x804, 0x9a8} {
		w.p32(smnT+off, 0x54000000+off)
	}
	w.p32(smnT+0x6a0, smnFmt)
	for i, name := range []string{"Rex", "Vi", "Ulrich", "Myles", "Saxx", "Sparkle", "Lucinda", "Glumph"} {
		c := smnChr + 0x1862*int64(i)
		for j, ch := range name + "\x00" {
			w.wr(c+5+2*int64(j), 2, int64(ch))
		}
		for n, off := range []int64{0x5cb, 0x5f1, 0x617, 0x63d} {
			w.p32(c+off, 200+10*int64(i)+int64(n))
		}
		w.p32(c+0xa35, 0)
		w.p8(c+0x181f, 1)
		w.p8(smnPC+0x106*int64(i)+0xf1, 0x60+int64(i))
	}
	for _, s := range []int64{0x38, 0x3d, 0x20, 0x1a} { // decoys on the neighbouring spell records
		for _, d := range []int64{-1, 1} {
			w.p32(smnSpells+0x1bf*(s+d)+0x44, 900+s+d)
			w.p32(smnSpells+0x1bf*(s+d)+0x4d, 800+s+d)
		}
	}
	for k := int64(0); k < 8; k++ {
		w.wr(gsName+2*k, 2, 0)
	}
}

func smnRun(e *gsRun) {
	w := e.w
	P := int64(smnEff)
	typ := w.u32(P + 0x5c)
	isChar := func() bool { // 0x53bea0(P+0x5c, 0)
		if typ == 1 && w.i32(P+0x60) == -1 {
			panic("summon selection: character caster without index (0x53bea0 asserts)")
		}
		return typ == 1
	}
	isMonster := func() bool { // 0x53bf10(P+0x5c, 0)
		if typ == 2 && w.i32(P+0x64) == -1 {
			panic("summon selection: monster caster without id (0x53bf10 asserts)")
		}
		return typ == 2
	}
	chr := func() int64 {
		idx := w.u32(P + 0x60)
		if idx > 7 {
			panic("summon selection: character index outside the fixture's 0..7")
		}
		return w.u32(0x685174) + 0x1862*idx
	}
	if isChar() && w.u8(chr()+0x181f) != 0 { // 0x5248a0(idx, 0)
		if w.u8(0x68510c) != 0 {
			smnMessage(e, w.u32(P+0x60), w.u32(w.u32(0x68c09c)+0x6a0))
		} else {
			e.call(0x5905f0, 0x619788, 0xffffffff)
			w.p8(0x69b7b8, 1) // 0x5905c0(1, -1): -1 leaves 0x64bd54
			e.call(0x5905f0, w.u32(w.u32(0x68c09c)+0x6a0), 0xffffffff)
		}
		w.p8(P+0x124, 1)
		return
	}
	var mon int64
	if isMonster() {
		mon = e.rec(w.u32(P + 0x64))
		if w.u32(mon+0x344) != 0xffffffff {
			return
		}
	}
	var wt [4]uint32
	for i := range wt {
		wt[i] = 0x14
	}
	if isChar() {
		for i, off := range []int64{0x5cb, 0x5f1, 0x617, 0x63d} {
			wt[i] = uint32(w.u32(chr()+off)) + 0x14
		}
	}
	if w.u8(0x652da4) != 0 {
		wt[0] = 0
	}
	x := uint32(e.r.R(int(wt[0] + wt[1] + wt[2] + wt[3])))
	cat := int64(-1)
	for i, v := range wt {
		if x < v {
			cat = int64(i)
			break
		}
		x -= v
	}
	if cat < 0 {
		panic("summon selection: random value past the weight sum (0x550257)")
	}
	var bl int64 = 1
	if isChar() {
		bl = 1
		if w.u32(chr()+0xa35) == 0 {
			bl = 2
		}
	} else if isMonster() {
		bl = w.u8(mon + 0x16)
	}
	if w.u8(P+0x78) != 0 { // 0x5502cb
		if bl == 1 {
			bl = 2
		} else {
			bl = 1
		}
	}
	power := w.u32(P + 0xd0)
	sp := gsExeDword(uint32(0x616f4c + 4*(cat+4*power))) // summon species by power row, category column
	g := smnSpawn(e, sp, mon)
	rec := e.rec(w.u32(g + 0x9f))
	e.call(0x54ff20, w.u32(P+0xa0), w.u32(P+0xa4), w.u32(P+0xa8), bl&0xff, w.rd(rec, 2))
	facing := int64(math.Float32bits(float32(math.Float64frombits(uint64(w.rd(gsStub+0x88, 8))))))
	e.call(0x4538f0, facing)
	e.setDisposition(g, bl&0xff)
	for i := int64(0); i < w.u32(g+4); i++ { // 0x5e2c80(grp+8, i)
		m := e.rec(w.u32(w.u32(w.u32(g+8)) + 4*i))
		if w.u32(m+0x2da) == 0 {
			v := int64(2)
			if bl == 2 {
				v = 1
			}
			w.p32(m+0x2da, v)
			e.call(0x4acd80, w.u32(m+0xc), 0x27, 1)
		}
	}
	e.refresh()
	if isChar() {
		e.call(0x524780, w.u32(P+0x60), 0, w.u32(rec))
	} else if isMonster() {
		w.p32(mon+0x344, w.u32(rec))
	}
	if c := w.u32(P + 0xd4); c != 0 && int64(uint32(e.r.R(100))) < c {
		smnBonus(e, cat, power, rec)
	}
	e.release(g) // 0x550750
	w.p8(P+0x125, 1)
}

// smnSpawn is 0x50f1a0(species, 1, P+0xa0, 1, 0, 1).
func smnSpawn(e *gsRun, sp, mon int64) int64 {
	w := e.w
	P := int64(smnEff)
	e.call(0x50f1a0, sp, 1, P+0xa0, 1, 0, 1)
	if !e.real {
		g := w.u32(gsStub + 0x80)
		if g == 0 {
			panic("summon selection: 0x50f1a0 returned NULL (assert 0x55030c)")
		}
		if g != smnGrp {
			panic("summon selection: spawned group is not the fixture group")
		}
		return g
	}
	t, g := e.template(sp), int64(smnGrp)
	out := monsterGroupSpawnNode([]int64{
		w.rs(t+0xd6, 2), w.u8(t + 0xd8), w.u8(t + 0xd9), w.rs(t+0xda, 2), w.u8(t + 0xdc), w.u8(t + 0xdd),
		w.u8(t + 0x26a), w.u32(0x6874be), w.u32(P + 0xa0), w.u32(P + 0xa4), w.u32(P + 0xa8), 0, 0, 0,
		w.i32(0x68ec78), w.u32(smnFOnly), 1, 1, 0, 1, w.u8(t + 0xd0), w.u32(0x6874ba),
		w.u32(0x686a48), w.u8(0x689b32), 0, w.rd(t+0x187, 2), w.u8(smnFOnly + 4), w.u32(smnFOnly + 8),
	}, e.r)
	cnt := out[2]
	if cnt > 4 {
		panic("summon selection: spawned member count past the fixture's 4 member slots")
	}
	// the new monsters are the fixture records named by member slots 0..cnt-1
	first := out[19] - cnt + 1
	seen := map[int64]bool{}
	if mon != 0 {
		seen[mon] = true
	}
	var recs []int64
	for k := int64(0); k < cnt; k++ {
		rp := e.rec(w.u32(smnML + 0x10 + 4*k))
		if seen[rp] {
			panic("summon selection: spawned members must be distinct fixture records other than the monster caster")
		}
		seen[rp] = true
		recs = append(recs, rp)
	}
	for k, rp := range recs {
		w.p32(rp, first+int64(k))
		w.p32(smnML+0x10+4*int64(k), first+int64(k))
	}
	w.p32(smnML+8, cnt)
	w.p32(g, out[1])
	w.p32(g+4, cnt)
	w.p32(g+0x18, sp)
	w.p8(g+0x28, out[11])
	w.p8(g+0x29, out[12])
	w.p8(g+0x2c, out[14])
	w.p32(g+0x9f, out[19])
	w.p32(0x6874ba, out[21])
	w.p32(0x6874be, out[22])
	return g
}

// smnBonus is the bonus spell on the summoned leader rec (jump table 0x550770).
func smnBonus(e *gsRun, cat, power, rec int64) {
	w := e.w
	duration := func(s int64) int64 { // spell record [0x65be1c]+0x1bf*s
		sr := w.u32(0x65be1c) + 0x1bf*s
		d := uint32(w.u32(sr+0x44)*power + w.u32(sr+0x4d))
		if d != 0x270f {
			d++
			switch k := e.r.R(4); {
			case k == 0 && d > 1, k == 1 && d <= 2:
				d++
			}
		}
		return int64(d) // 0x517f90(&d, 0) keeps it
	}
	switch gsExeDword(uint32(0x550770 + 4*cat)) {
	case 0x550651:
		s := []int64{0x38, 0x3d}[cat] // 0x550651: category 0 / 1
		d := duration(s)
		idx := s - 0x13
		if uint64(idx) > 0x2e {
			panic("summon selection: spell outside the effect table (assert 0x550718)")
		}
		tgt := uint32(gsExeDword(uint32(0x550780 + 4*int64(ticRead(uint32(0x5507a0+idx), 1)[0]))))
		if ticRead(tgt, 1)[0] != 0xb8 { // each target starts with opcode byte 0xb8 and the effect id dword
			panic("summon selection: spell without effect id (assert 0x550718)")
		}
		e.call(0x5242b0, w.u32(rec), gsExeDword(tgt+1), power, d, 0)
	case 0x550541:
		s := []int64{0, 0, 0x1a, 0x20}[cat] // 0x550541: category 2 / 3
		d := duration(s)
		slot := int64(-1)
		for i := int64(0); i < 12; i++ { // dwords 0x616d84..0x616db0
			if gsExeDword(uint32(0x616d84+4*i)) == s {
				slot = i
				break
			}
		}
		if slot < 0 {
			panic("summon selection: spell without a buff slot (0x55058d)")
		}
		r := rec + 0x10f + 0x11*slot
		if w.u8(r) == 0 || w.u32(r+1) != s {
			e.call(0x4acd80, w.u32(rec+0xc), gsExeDword(uint32(0x60cffc+8*s)), 1)
		}
		w.p8(r, 1)
		w.p32(r+1, s)
		w.p32(r+5, power)
		w.p32(r+0xd, d)
		// 0x50e8c0([rec])
		monsterCacheRebuild(func(a uint32) byte { return byte(w.u8(int64(a))) },
			func(a uint32) uint32 { return uint32(w.u32(int64(a))) },
			func(a uint32, n int, v uint64) { w.wr(int64(a), n, int64(v)) },
			uint32(rec), uint32(e.template(w.u32(rec+8))))
	default:
		panic("summon selection: unknown bonus case in jump table 0x550770")
	}
}

// smnMessage is 0x590950(idx, format): the "<name> summons" line.
func smnMessage(e *gsRun, idx, fmtp int64) {
	w := e.w
	if idx > 7 {
		panic("summon selection: character index outside the fixture's 0..7")
	}
	var buf []int64 // vswprintf(buf, format, no arguments)
	for i := int64(0); ; i++ {
		c := w.rd(fmtp+2*i, 2)
		if c == 0 {
			break
		}
		if c == '%' {
			panic("summon selection: '%' in the message format (vswprintf arguments are outside the fixture)")
		}
		if i >= 0xfff {
			panic("summon selection: message format past the 0x2000-byte buffer")
		}
		buf = append(buf, c)
	}
	// The ' / : choice copies 0x689b34 or L" " 0x60cc74 into a separate two-character
	// local (0x58aad0's fourth argument); the length test below reads only the
	// formatted text (its fifth argument).
	buf = append(buf, 0, 0)
	name := w.u32(0x685174) + 0x1862*idx + 5
	e.call(0x58aad0, 8, 0x64bf5c, name)
	n := int64(0)
	for w.rd(name+2*n, 2) != 0 {
		n++
	}
	if buf[0] == 0x27 {
		n++
		if buf[1] == 0x73 {
			n++
		}
	}
	e.call(0x58b410, w.u8(w.u32(0x685178)+0x106*idx+0xf1), 0, n, 0xffffffff)
}
