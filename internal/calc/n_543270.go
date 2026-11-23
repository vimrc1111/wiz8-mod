package calc

import "fmt"

// attackEffectsNode 0x543270 attack_effects(source, target, effect,
// durationMessage, message, stats): the target's primary state is read from
// the character record (+0xb01, kind 1) or the monster found by id (+0x107,
// kind 3). For each of the sixteen chance bytes effect +8.. (stopping once the
// state reaches 0x12) a non-zero chance c rolls R(100) < c, then computes the
// duration 0x5519c0 (below 2: dice reset 0x517950 on effect +4 and 0x551a20 +
// 1), clamped to 1..12, and dispatches on the slot:
//   - 0,1,3,4,6,7,8,14,15: 0x551ba0 with a fixed state/realm (slot 4 lasts 9999);
//   - 2: its own duration (penalty effect +0x1c, unclamped);
//   - 5: 0x551ba0 state 0x12; on failure to resist, a log line (0x58aad0, 8 or
//     9 by source kind), for a character source R(100) < 50 a 0x52e690 event
//     with the cookie 0x68c510, otherwise with more than one hostile monster
//     (0x683fa5) one with 0x5ee668 whose record gets +0x30 = 800 and +0x34 =
//     tick; a character target also gets the 0x590950 "%s!" notice;
//   - 9, 10: only when 0x53bf10 (source kind 2): 9 drains (0x551ba0 state 0x13
//     then 0x524780 for characters, state 0x12 for monsters), 10 state 0xd for
//     characters;
//   - 11, 12, 13: 0x5520d0 resist check, a dice amount, 0x552250 reduction
//     through its value pointer, then hit points 0x52b7e0 (characters), stamina
//     0x52c070 (monsters, modelled here as in spellStaminaDrain) or 0x52af50
//     (characters), spell points 0x52b590 (characters; the active character
//     0x68518d also 0x593330);
//
// then re-reads the state. With a stats block, per-slot success counts and the
// three amounts accumulate in a local block and are folded into stats at the
// end: a slot-5 success clears stats +0x68/+0x6c/+0x70 and +0xc..+0x54 except
// +0x50, which adds the slot-5 count; otherwise everything adds. +0xc counts
// runs that applied an amount.
//
// The case address-stubs the D4 callees 0x551ba0, 0x5519c0, 0x551a20,
// 0x5520d0, 0x552250 (its value write comes from inputs), 0x52b7e0, 0x52af50,
// 0x52b590, 0x524780, 0x523c00 and the 排除 0x52e690, 0x58aad0, 0x58b410,
// 0x593330; call counts and recorded arguments are outputs. Emu replays the
// stubs; F runs effectDurationNode, effectQuantityNode,
// statusImmunityAndResistNode, statusResistRoll,
// resistanceQuantityReductionNode, characterHpDeficitApplyNode (its +0xb01 is
// fed back as the state) and monsterConditionApplyNode (stamina, condition 17,
// main condition and fatigue fed back), with F-only "f." (0x551ba0 world) and
// "g." (0x52b7e0 world) inputs.
//
// Fixture caps (panic past them): party index 0..7, four monster records with
// ids below 0xffffff00 (decoys), template index 0..2, the 0x52e690 event
// record at the case object (or 0), the stats block at the case object (or 0).
// Asserts 0x54330b (character index -1), 0x543338 (absent member), 0x54338a
// (target kind not 1/3), 0x5433a6 (monster id -1) panic
// (notes/re/cov-exempt.d/543270.tsv), as does 0x53bf10's kind-2 id -1 assert.
func attackEffectsNode(in []int64, r Rand) []int64 { return aeRun(in, r, true) }

func attackEffectsEmu(in []int64, r Rand) []int64 { return aeRun(in, r, false) }

const (
	aeArena    = 0x20000000
	aeStats    = aeArena + 0x100   // case object "stats"
	aeEvent    = aeArena + 0x200   // case object "event"
	aeChars    = aeArena + 0x10000 // case object "chars", stride 0x1862
	aeCharSz   = 0x1862
	aeTick     = 4660 // case "tick" (GetTickCount)
	aeLogFmt   = 0x64bf5c
	aeNoticeFm = 0x6175c0
	aeEvArgA   = 0x5ed8c8
	aeEvArgB   = 0x5ed914
	aeEvCookie = 0x5ee668
	aeSirCalls = 13
	aeDurCalls = 20
	aeRrCalls  = 3
	aeStatsN   = 22 // +0xc..+0x54, +0x68, +0x6c, +0x70
)

// Input indexes (aeIn order).
const (
	aeDurMsg = iota
	aeMsg
	aeStatsOut
	aeSrcKind
	aeSrcIndex
	aeSrcID
	aeTgtKind
	aeTgtIndex
	aeMonID
	aeMatchSlot
	aePresence
	aeColor
	aeCharState
	aeName0
	aeName1
	aeTplIndex
	aeMonState
	aeMonHP
	aeMonMaxSP
	aeMonSP
	aeMonCond17
	aeMonFatigue
	aeQBase
	aeQCount
	aeQSides
	aeChance
	aeStrength = aeChance + 16
)

const (
	aePenalty = aeStrength + 1 + iota
	aePower
	aeDurBonus
	aeDurBase
	aeDurPer
	aeHostile
	aeActive
	aeCookie
	aeStat
)

const (
	aeEv30    = aeStat + aeStatsN
	aeEv34    = aeEv30 + 1
	aeText383 = aeEv30 + 2
	aeText0   = aeEv30 + 3
	aeText1   = aeEv30 + 4
	aeRSir    = aeEv30 + 5
	aeRDur    = aeRSir + aeSirCalls
	aeRQty    = aeRDur + aeDurCalls
	aeRRr     = aeRQty + aeDurCalls
	aeWRq     = aeRRr + aeRrCalls
	aeRRq     = aeWRq + aeRrCalls
	aeRHp     = aeRRq + 1
	aeRFat    = aeRHp + 1
	aeRSp     = aeRFat + 1
	aeRDrain  = aeRSp + 1
	aeRCond   = aeRDrain + 1
	aeREvent  = aeRCond + 1
	aeF       = aeREvent + 1 // F-only inputs: "f." then "g."
)

var aeIn, aeOut, aeSirF, aeHdaF = aeLayout()

// aeSirOverride are the 0x551ba0 world inputs this node supplies itself.
var aeSirOverride = map[string]bool{"matchSlot": true, "characterPrimaryState": true, "memberNameColor": true, "templateIndex": true, "monsterPrimaryState": true}

var aeHdaOverride = map[string]bool{"who": true, "amount": true, "flag": true, "pres": true, "b01": true}

func aeLayout() (in, out []string, sirF, hdaF []int) {
	in = []string{"durationMessage", "message", "statsOut", "sourceKind", "sourceIndex", "sourceID",
		"targetKind", "targetIndex", "monsterID", "matchSlot", "presence", "memberNameColor",
		"characterPrimaryState", "characterName0", "characterName1", "templateIndex", "monsterPrimaryState",
		"monsterHP", "monsterMaxStamina", "monsterStamina", "monsterCondition17", "monsterFatigue",
		"quantityBase", "quantityDiceCount", "quantityDieSides"}
	for i := 0; i < 16; i++ {
		in = append(in, fmt.Sprintf("chance%d", i))
	}
	in = append(in, "effectStrength", "effectPenalty", "effectPower", "durationBonusPercent", "durationBase", "durationPerPower",
		"hostileMonsterCount", "activeCharacterIndex", "damageEventCookie")
	stats := aeStatNames()
	for _, s := range stats {
		in = append(in, s)
	}
	in = append(in, "event_30", "event_34", "text383Ptr", "text913_0", "text913_1")
	seq := func(p string, n int) {
		for k := 0; k < n; k++ {
			in = append(in, fmt.Sprintf("%s_%d", p, k))
		}
	}
	seq("r551ba0", aeSirCalls)
	seq("r5519c0", aeDurCalls)
	seq("r551a20", aeDurCalls)
	seq("r5520d0", aeRrCalls)
	seq("w552250", aeRrCalls)
	in = append(in, "r552250", "r52b7e0", "r52af50", "r52b590", "r524780", "r523c00", "r52e690")
	if len(in) != aeF {
		panic("attack effects: input layout drift")
	}
	sirF = make([]int, len(sirIn)-sirSlot)
	for j, n := range sirIn[sirSlot:] {
		sirF[j] = -1
		if !aeSirOverride[n] {
			sirF[j] = len(in)
			in = append(in, "f."+n)
		}
	}
	h := hdaNames()
	hdaF = make([]int, len(h))
	for j, n := range h {
		hdaF[j] = -1
		if !aeHdaOverride[n] {
			hdaF[j] = len(in)
			in = append(in, "g."+n)
		}
	}

	out = []string{"quantityBase", "quantityDiceCount", "quantityDieSides"}
	out = append(out, stats...)
	out = append(out, "event_30", "event_34", "monsterStamina", "monsterFatigue")
	rec := func(fn string, args []int, n int) {
		out = append(out, "c"+fn)
		for _, a := range args {
			if n == 0 {
				out = append(out, fmt.Sprintf("a%s_%d", fn, a))
				continue
			}
			for k := 0; k < n; k++ {
				out = append(out, fmt.Sprintf("a%s_%d_%d", fn, a, k))
			}
		}
	}
	rec("551ba0", []int{3, 4, 5, 6, 7, 8, 9, 10}, aeSirCalls)
	rec("5519c0", nil, 0)
	rec("551a20", nil, 0)
	rec("5520d0", []int{2, 3, 4}, aeRrCalls)
	rec("552250", []int{3, 4}, aeRrCalls)
	rec("52b7e0", []int{1, 2, 3}, 0)
	rec("52af50", []int{1, 2, 3, 4}, 0)
	rec("52b590", []int{1, 2, 3}, 0)
	rec("524780", []int{1, 2, 3}, 0)
	rec("523c00", []int{1, 2, 3, 4, 6}, 0)
	rec("52e690", []int{1, 2, 3, 4, 5}, 0)
	rec("58aad0", []int{1, 2}, 2)
	rec("58b410", []int{1, 2, 3, 4}, 0)
	rec("593330", nil, 0)
	return
}

// aeStatOff is the byte offset of stats input/output k.
func aeStatOff(k int) uint32 {
	if k < 19 {
		return 0xc + 4*uint32(k)
	}
	return 0x68 + 4*uint32(k-19)
}

func aeStatNames() []string {
	s := make([]string, aeStatsN)
	for k := range s {
		s[k] = fmt.Sprintf("stats_%02x", aeStatOff(k))
	}
	return s
}

func init() {
	Nodes["00543270"] = Node{Addr: "00543270", In: aeIn, Out: aeOut, F: attackEffectsNode, Emu: attackEffectsEmu}
}

func aeRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(aeIn) {
		panic("attack effects: wrong input count")
	}
	u := func(i int) uint32 { return uint32(in[i]) }
	statsOut := u(aeStatsOut)
	if statsOut != 0 && statsOut != aeStats {
		panic("attack effects: stats block outside the case object")
	}
	kind, idx, monID := u(aeTgtKind), u(aeTgtIndex), u(aeMonID)
	var status uint32
	switch kind {
	case 1:
		if idx == 0xffffffff {
			panic("attack effects: character index -1 (assert 0x54330b)")
		}
		if idx > 7 {
			panic("attack effects: party index past 0..7")
		}
		if u(aePresence)&0xff == 0 {
			panic("attack effects: absent party member (assert 0x543338)")
		}
		status = u(aeCharState)
	case 3:
		if monID == 0xffffffff {
			panic("attack effects: monster id -1 (assert 0x5433a6)")
		}
		if monID >= 0xffffff00 || u(aeMatchSlot) > 3 {
			panic("attack effects: monster id or record past the case's four records")
		}
		status = u(aeMonState)
	default:
		panic("attack effects: target kind not 1 or 3 (assert 0x54338a)")
	}
	// Tracked effect dice (+4 i16, +6, +7) and monster stamina record.
	qBase, qCount, qSides := uint16(u(aeQBase)), uint8(u(aeQCount)), uint8(u(aeQSides))
	monSP, monC17, monFat := u(aeMonSP), u(aeMonCond17), u(aeMonFatigue)
	charState := u(aeCharState)
	strength := u(aeStrength)

	out := make([]int64, len(aeOut))
	o := 3 + aeStatsN + 4 // first stub output
	oSir, oDur := o, o+1+8*aeSirCalls
	oQty := oDur + 1
	oRr := oQty + 1
	oRq := oRr + 1 + 3*aeRrCalls
	oHp := oRq + 1 + 2*aeRrCalls
	oFat := oHp + 4
	oSp := oFat + 5
	oDrain := oSp + 4
	oCond := oDrain + 4
	oEvent := oCond + 6
	oLog := oEvent + 6
	oColor := oLog + 1 + 2*2
	oActive := oColor + 5
	if oActive+1 != len(out) {
		panic("attack effects: output layout drift")
	}
	call := func(base, n, k int, args ...uint32) { // count at base; arg j of call k at base+1+j*n+k
		out[base]++
		if n == 0 {
			n, k = 1, 0
		}
		for j, a := range args {
			out[base+1+j*n+k] = int64(a)
		}
	}

	nSir, nDur, nQty, nRr, nRq, nLog := 0, 0, 0, 0, 0, 0
	duration := func() uint32 {
		v := u(aeRDur + nDur)
		if real {
			v = uint32(effectDurationNode([]int64{in[aePower], int64(int32(u(aeDurBonus))), in[aeDurBase], in[aeDurPer]}, r)[0])
		}
		nDur++
		return v
	}
	quantity := func(count, sides, base uint32) uint32 { // 0x517950 then 0x551a20
		qCount, qSides, qBase = uint8(count), uint8(sides), uint16(base)
		v := u(aeRQty + nQty)
		if real {
			qin := []int64{int64(int16(qBase)), int64(qCount), int64(qSides), int64(int32(u(aeDurBonus)))}
			// calc fight's enumerator rolls all dice as one convolution branch; the
			// summed value re-enters 0x551a20 as a dice word with no dice (same result).
			if dt, ok := r.(interface {
				DiceTotal(count, z, n, s int64) int64
			}); ok && qin[0]+qin[1]*qin[2] <= 32767 {
				qin[0], qin[1], qin[2] = dt.DiceTotal(1, qin[0], qin[1], qin[2]), 0, 0
			}
			v = uint32(effectQuantityNode(qin, r)[0])
		}
		nQty++
		return v
	}
	q := strength/3 + 1
	amount := func(count, sides, base uint32) uint32 {
		d := duration()
		if d < 2 {
			d = quantity(count, sides, base) + 1
		}
		return d
	}
	srcIdx := u(aeSrcIndex)
	sir := func(st, realm, pen, dur, a8, a9, a10 uint32) bool { // 0x551ba0; true = not resisted
		if nSir >= aeSirCalls {
			panic("attack effects: 0x551ba0 past its 13 case returns")
		}
		call(oSir, aeSirCalls, nSir, st, realm, strength, pen, dur, a8, a9, a10)
		res := u(aeRSir + nSir)
		if real {
			v := make([]int64, len(sirIn))
			v[sirStatus], v[sirRealm], v[sirStrength], v[sirPenalty], v[sirDuration] = int64(st), int64(realm), int64(strength), int64(pen), int64(dur)
			v[sirMessage], v[sirDurMessage], v[sirDurFlagB] = int64(a8), int64(a9), int64(a10)
			v[sirSrcKind], v[sirSrcIndex] = in[aeSrcKind], int64(int32(srcIdx))
			v[sirKind], v[sirIdx], v[sirMonID] = int64(kind), int64(int32(idx)), int64(monID)
			ms := in[aeMonState]
			if kind == 3 {
				ms = int64(status)
			}
			aeWorld(v[sirSlot:], aeSirF, in, map[string]int64{"matchSlot": in[aeMatchSlot], "characterPrimaryState": int64(charState),
				"memberNameColor": in[aeColor], "templateIndex": in[aeTplIndex], "monsterPrimaryState": ms}, sirIn[sirSlot:])
			res = uint32(statusImmunityAndResistNode(v, r)[0])
		}
		nSir++
		return res&0xff == 0
	}
	fv := func(n string) int64 { // F-only 0x551ba0 world value by name
		for j, m := range sirIn[sirSlot:] {
			if m == n && aeSirF[j] >= 0 {
				return in[aeSirF[j]]
			}
		}
		panic("attack effects: F-only input " + n + " missing")
	}
	realms := []string{"fire", "water", "air", "earth", "mental", "divine"}
	resistCheck := func(realm uint32) bool { // 0x5520d0; true = resisted
		call(oRr, aeRrCalls, nRr, realm, strength, 0)
		res := u(aeRRr + nRr)
		if real {
			v := []int64{int64(kind), int64(idx), int64(monID), int64(realm), int64(strength), 0, fv("difficulty"), fv("characterLevel")}
			for _, n := range realms {
				v = append(v, fv("characterResistance_"+n))
			}
			v = append(v, int64(charState), fv("characterSkillGrowthFlag"), fv("characterReverseDifficulty"), in[aeTplIndex])
			for _, n := range realms {
				v = append(v, fv("templateResistance_"+n))
			}
			v = append(v, fv("templateLevel"))
			for _, n := range realms {
				v = append(v, fv("monsterResistanceModifier_"+n))
			}
			ms := in[aeMonState]
			if kind == 3 {
				ms = int64(status)
			}
			v = append(v, ms, fv("monsterDifficultyClass"), 0)
			res = uint32(statusResistRoll(v, r, true)[0])
		}
		nRr++
		return res&0xff != 0
	}
	reduce := func(d, realm uint32) uint32 { // 0x552250(&d, target, realm, strength)
		call(oRq, aeRrCalls, nRq, realm, strength)
		res := u(aeWRq + nRq)
		if real {
			ci := int64(idx)
			if kind != 1 {
				ci = 0
			}
			v := []int64{int64(d), fv("characterResistance_fire"), fv("characterLevel"), int64(strength), int64(kind), int64(realm), ci}
			for _, n := range realms[1:] {
				v = append(v, fv("characterResistance_"+n))
			}
			for _, n := range realms {
				v = append(v, fv("templateResistance_"+n))
			}
			v = append(v, fv("templateLevel"))
			for _, n := range realms {
				v = append(v, fv("monsterResistanceModifier_"+n))
			}
			v = append(v, int64(monID), in[aeTplIndex])
			res = uint32(resistanceQuantityReductionNode(v, r)[0])
		}
		nRq++
		return res
	}
	logLine := func(a1, a2 uint32) { // 0x58aad0
		call(oLog, 2, nLog, a1, a2)
		nLog++
	}
	img := &mstWorld{}

	var local [0x74 / 4]uint32 // the 0xa2-byte block, present iff statsOut != 0
	add := func(off, v uint32) {
		if statsOut != 0 {
			local[off/4] += v
		}
	}
	applied := false
	ev30, ev34 := u(aeEv30), u(aeEv34)
	b2u := func(b bool) uint32 {
		if b {
			return 1
		}
		return 0
	}
	durMsg, msg := u(aeDurMsg), u(aeMsg)
	generic := map[int][3]uint32{0: {0xf, 2, 0x44}, 1: {0x10, 1, 0x48}, 3: {9, 5, 0x2c}, 4: {2, 1, 0x10},
		6: {0x11, 3, 0x4c}, 7: {0xc, 0, 0x38}, 8: {6, 4, 0x20}, 14: {4, 2, 0x18}, 15: {0xb, 4, 0x34}}
	srcKind := u(aeSrcKind)
	sourceIsKind2 := func() bool { // 0x53bf10(source, 0)
		if srcKind != 2 {
			return false
		}
		if u(aeSrcID) == 0xffffffff {
			panic("attack effects: source kind 2 with id -1 (0x53bf10 asserts)")
		}
		return true
	}
	below := func(n int, c uint32) bool { // R(n) < c; calc fight's enumerator branches it two ways
		if b, ok := r.(interface{ Below(n, c int) bool }); ok && real {
			return b.Below(n, int(min(c, uint32(n))))
		}
		return uint32(r.R(n)) < c
	}
	for i := 0; i < 16 && status < 0x12; i++ {
		c := u(aeChance+i) & 0xff
		if c == 0 {
			continue
		}
		if below(100, c) {
			d := amount(q, 2, 1)
			dur := d
			if int32(d) < 1 {
				dur = 1
			} else if d > 12 {
				dur = 12
			}
			if g, ok := generic[i]; ok {
				if i == 4 {
					dur = 0x270f
				}
				add(g[2], b2u(sir(g[0], g[1], 0, dur, msg, durMsg, 0)))
			}
			switch i {
			case 2:
				d2 := amount(3*q, 2, u(aePenalty))
				add(0x24, b2u(sir(7, 1, u(aePenalty), d2, msg, durMsg, 0)))
			case 5:
				if !sir(0x12, 5, 0, 0x270f, msg, 0, 0) {
					break
				}
				text := u(aeText383)
				if srcKind == 1 {
					logLine(8, text)
					cc := aeChars + srcIdx*aeCharSz
					if below(100, 50) {
						call(oEvent, 0, 0, cc, u(aeCookie), 0, img.u32(aeEvArgA), img.u32(aeEvArgB))
					} else if u(aeHostile) > 1 {
						call(oEvent, 0, 0, cc, img.u32(aeEvCookie), 0, img.u32(aeEvArgA), img.u32(aeEvArgB))
						switch ev := u(aeREvent); ev {
						case 0:
						case aeEvent:
							ev30, ev34 = 0x320, aeTick
						default:
							panic("attack effects: 0x52e690 event outside the case object")
						}
					}
				} else {
					logLine(9, text)
				}
				if kind == 1 {
					aeNotice(in, img, logLine, func(a ...uint32) { call(oColor, 0, 0, a...) })
				}
				add(0x50, 1)
			case 9:
				if !sourceIsKind2() {
					break
				}
				if kind == 1 {
					if sir(0x13, 5, 0, 0x270f, msg, durMsg, 1) {
						add(0x54, 1)
						call(oDrain, 0, 0, idx, 1, u(aeSrcID))
					}
				} else if kind == 3 {
					sir(0x12, 5, 0, 0x270f, msg, durMsg, 0)
				}
			case 10:
				if sourceIsKind2() && kind == 1 {
					add(0x3c, b2u(sir(0xd, 5, 0, dur, msg, durMsg, 0)))
				}
			case 11, 12, 13:
				realm := map[int]uint32{11: 3, 12: 1, 13: 5}[i]
				if kind != 1 && i != 12 {
					break
				}
				if resistCheck(realm) {
					break
				}
				var a uint32
				switch i {
				case 11:
					a = amount(q, 2, 0)
				case 12:
					a = amount(2*q, 5, 3*q)
				case 13:
					a = amount(q, 6, q)
				}
				a = reduce(a, realm)
				if a == 0 {
					break
				}
				switch {
				case i == 11:
					call(oHp, 0, 0, idx, a, 1)
					if real {
						v := make([]int64, len(aeHdaF))
						for j, n := range hdaNames() {
							switch n {
							case "who":
								v[j] = int64(idx)
							case "amount":
								v[j] = int64(a)
							case "flag":
								v[j] = 1
							case "pres":
								v[j] = in[aePresence]
							case "b01":
								v[j] = int64(charState)
							default:
								v[j] = in[aeHdaF[j]]
							}
						}
						charState = uint32(characterHpDeficitApplyNode(v, r)[4])
					}
				case i == 12 && kind == 3:
					monSP, monC17, monFat, status = aeStaminaDrain(in, r, real, a, monSP, monC17, monFat, status, fv("templateCategory"),
						func(a ...uint32) { call(oCond, 0, 0, a...) })
				case i == 12:
					call(oFat, 0, 0, idx, a, 0, 0)
				default:
					call(oSp, 0, 0, idx, a, 1)
					if idx == u(aeActive) {
						out[oActive]++
					}
				}
				off := map[int]uint32{11: 0x68, 12: 0x6c, 13: 0x70}[i]
				add(off, a)
				if statsOut != 0 {
					applied = true
				}
			}
		}
		if kind == 1 { // re-read the state
			status = charState
		}
	}
	stats := make([]uint32, aeStatsN)
	for k := range stats {
		stats[k] = u(aeStat + k)
	}
	if statsOut != 0 {
		if applied {
			local[0xc/4]++
		}
		if local[0x50/4] != 0 {
			for k := range stats {
				switch off := aeStatOff(k); {
				case off == 0x50:
					stats[k] += local[0x50/4]
				default:
					stats[k] = 0
				}
			}
		} else {
			for k := range stats {
				stats[k] += local[aeStatOff(k)/4]
			}
		}
	}
	out[0], out[1], out[2] = int64(qBase), int64(qCount), int64(qSides)
	for k, v := range stats {
		out[3+k] = int64(v)
	}
	out[3+aeStatsN], out[3+aeStatsN+1] = int64(ev30), int64(ev34)
	out[3+aeStatsN+2], out[3+aeStatsN+3] = int64(monSP), int64(monFat)
	out[oDur], out[oQty] = int64(nDur), int64(nQty)
	return out
}

// aeWorld fills a callee world slice from F-only inputs, with overrides by name.
func aeWorld(dst []int64, idx []int, in []int64, over map[string]int64, names []string) {
	for j, n := range names {
		if idx[j] >= 0 {
			dst[j] = in[idx[j]]
		} else if v, ok := over[n]; ok {
			dst[j] = v
		} else {
			panic("attack effects: world input " + n + " missing")
		}
	}
}

// aeNotice is 0x590950(targetIndex, 0x6175c0, text 913): the log line
// 0x58aad0(8, 0x64bf5c, name) and 0x58b410(color, 0, n, -1), where n is the
// name length plus one for a leading apostrophe and one more for "'s".
func aeNotice(in []int64, img *mstWorld, logLine func(a1, a2 uint32), color func(...uint32)) {
	var text []uint16
	if c := uint16(in[aeText0]); c != 0 {
		text = append(text, c)
		if c := uint16(in[aeText1]); c != 0 {
			text = append(text, c, 'x') // case string continues with 'x'
		}
	}
	var buf []uint16
	f := sirWide(img, aeNoticeFm)
	for i := 0; i < len(f); i++ {
		if f[i] != '%' {
			buf = append(buf, f[i])
			continue
		}
		if i+1 >= len(f) || f[i+1] != 's' {
			panic("attack effects: notice format conversion other than one %s")
		}
		buf = append(buf, text...)
		i++
	}
	at := func(i int) uint16 {
		if i < len(buf) {
			return buf[i]
		}
		return 0
	}
	logLine(8, aeLogFmt)
	n := uint32(0)
	if uint16(in[aeName0]) != 0 {
		n = 1
		if uint16(in[aeName1]) != 0 {
			n = 2 // case names stop at +9
		}
	}
	if at(0) == '\'' {
		n++
		if at(1) == 's' {
			n++
		}
	}
	color(uint32(in[aeColor])&0xff, 0, n, 0xffffffff)
}

// aeStaminaDrain is the native 0x52c070(monster, amount, 0), as in
// spellStaminaDrain: returns stamina, condition 17, fatigue and main condition.
func aeStaminaDrain(in []int64, r Rand, real bool, a, sp, c17, fat, state uint32, tplKind int64, cond func(...uint32)) (uint32, uint32, uint32, uint32) {
	if uint32(in[aeMonHP]) == 0 { // 0x52c07b
		return sp, c17, fat, state
	}
	if a != 0 {
		if uint32(in[aeTplIndex]) > 2 {
			panic("attack effects: template index past the case's three templates (0x4e5720)")
		}
		if a >= sp {
			sp = 0
		} else {
			sp -= a
		}
	}
	maxSP := uint32(in[aeMonMaxSP])
	if maxSP == 0 {
		panic("attack effects: maximum stamina 0 divides by zero (0x52c0b8)")
	}
	fat = staminaFatigueLevel(sp, maxSP)
	if sp != 0 || c17 >= 0x270f {
		return sp, c17, fat, state
	}
	cond(uint32(in[aeMonID]), 0x11, 0x270f, 0, 1)
	if !real {
		return sp, c17, fat, state
	}
	x := append([]int64(nil), sdrApplyParked...)
	set := func(name string, v int64) {
		for i, n := range mcaSpecApplyIn {
			if n == name {
				x[i] = v
				return
			}
		}
		panic("attack effects: parked input " + name + " missing")
	}
	set("cond", 0x11)
	set("duration", 0x270f)
	set("penalty", 0)
	set("message", 1)
	set("M.HP", in[aeMonHP])
	set("M.maxSP", int64(maxSP))
	set("M.SP", int64(sp))
	set("M.cond17", int64(c17))
	set("M.mainCond", int64(state))
	set("tpl.kind", tplKind)
	set("a5.type", 0)
	set("a5.char", -1)
	set("a5.monster", -1)
	set("a5.f1b", 0)
	set("a5.f1c", 0)
	set("a5.f1e", 0)
	o := monsterConditionApplyNode(x, r)
	phased := false
	for i, n := range mcaOutNames {
		switch n {
		case "M.SP":
			sp = uint32(o[i])
		case "M.cond17":
			c17 = uint32(o[i])
		case "M.mainCond":
			state = uint32(o[i])
		case "s52c240_calls":
			phased = o[i] != 0
		}
	}
	if phased {
		for i, n := range mcaOutNames {
			if n == "M.fatigue" {
				fat = uint32(o[i])
			}
		}
	}
	return sp, c17, fat, state
}
