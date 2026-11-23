package calc

import "fmt"

// spellStatusFallbackNode is 0x54dbe0 (cdecl, void): a spell context (arg1)
// tries to inflict a status on every character, then every monster, listed in
// it. Arguments: arg2 status (for 0x551ba0), arg3 status (for 0x551eb0), arg4
// chance in percent, arg5 penalty (0x551ba0), arg6 penalty (0x551eb0).
//
// Context fields: +0 spell id (realm = the u32 at spell record +0x133; records
// at *0x65be1c, stride 0x1bf), +0x5c source {kind, index at +0x60}, +0xc8
// strength, +0xd0 level, +0xd4 bonus percent, +0xd8 base duration, +0xdc
// duration per level, +0xe4 monster count (i32) with ids at *(+0xec), +0xf4
// character count (i32) with indexes at *(+0xfc).
//
// 0x53bea0(source, 0) is true when the source kind is 1 (index -1 asserts
// there); sourceIndex = +0x60 then, else -1. Per target, in list order:
//   - R(100) >= chance (u32) skips it;
//   - d = perLevel*level + base (u32 wrap); unless d == 9999 (0x270f): d++,
//     R(4): 0 with d > 1 or 1 with d <= 2 adds 1, then 0x517f90 adds
//     u32(d*bonus)/100 (u32 product, unsigned division);
//   - a character target is {1, index}; index -1 asserts (block 54dcaa). A
//     monster target is {3, stale index, id}: 0x4e5550(0x4d8, file, id, 1) and
//     0x4e5620 find its record, id = record+0. The target struct is never
//     cleared, so a monster keeps the last character index (0 on a fresh stack)
//     and characters carry id 0;
//   - 0x551ba0(source, target, arg2, realm, strength, arg5, d, 0, 1, 0); a
//     nonzero low byte calls 0x551eb0(target, arg3, realm, strength, arg6, d,
//     sourceIndex, 0, 1), whose return is ignored.
//
// Stubs (case): 0x551ba0 and 0x551eb0 are D4 formula callees, address-stubbed;
// the outputs are each call's arguments and the call counts. 0x551ba0 call k
// returns resistResult_k, 0x551eb0 returns durationResult (ignored). The list
// helpers 0x4e5550/0x4e5620 run natively; every monster id is mirrored into both
// monster lists, so the lookup always hits.
//
// Emu (what TestEmu replays) uses the stubbed returns. F calls
// statusImmunityAndResistNode and statusDurationAfterResistNode with the
// F-only world block f.* (0x551ba0's inputs from its matchSlot on; 0x551eb0
// gets the tail it shares), one world for every target.
//
// Asserts (calc panics): source kind 1 with index -1, character index -1
// (cov-exempt 54dcaa). A null monster record (cov-exempt 54de27) needs an
// assert to pass first (0x4e5550 on a missing id, 0x4e5620 on a null entry); the
// fixture lists hold no null entry and every id, so the inputs cannot express
// it (fixture assumption: lookup always hits). Fixture caps (panics): spell
// id < 256, character and monster counts <= 8, monster list A count <= 8.
const (
	sfbSpell = iota
	sfbRealm
	sfbStatus
	sfbDurStatus
	sfbChance
	sfbPenalty
	sfbAltPenalty
	sfbSrcKind
	sfbSrcIndex
	sfbStrength
	sfbLevel
	sfbBonus
	sfbBase
	sfbPerLevel
	sfbMonCount
	sfbCharCount
	sfbListA
	sfbChar
	sfbMonID    = sfbChar + sfbTargets
	sfbResist   = sfbMonID + sfbTargets
	sfbDurRes   = sfbResist + sfbCalls
	sfbF        = sfbDurRes + 1
	sfbTargets  = 8
	sfbCalls    = 2 * sfbTargets
	sfbSpells   = 256
	sfbNoSource = 0xffffffff
	sfbFixed    = 9999 // 0x54dc47
)

var sfbResistOut = []string{"source", "status", "realm", "strength", "penalty", "duration", "message", "durationMessage", "durationFlagB", "target.kind", "target.index", "target.monsterID"}
var sfbDurOut = []string{"durationStatus", "realm", "strength", "penalty", "duration", "sourceIndex", "flagB", "message", "target.kind", "target.index", "target.monsterID"}

func init() {
	in := []string{"spell", "realm", "status", "durationStatus", "chance", "penalty", "altPenalty", "sourceKind", "sourceIndex",
		"strength", "level", "bonusPercent", "baseDuration", "durationPerLevel", "monsterCount", "characterCount", "monsterListCount"}
	for k := 0; k < sfbTargets; k++ {
		in = append(in, fmt.Sprintf("character%d", k))
	}
	for k := 0; k < sfbTargets; k++ {
		in = append(in, fmt.Sprintf("monsterID%d", k))
	}
	for k := 0; k < sfbCalls; k++ {
		in = append(in, fmt.Sprintf("resistResult%d", k))
	}
	in = append(in, "durationResult")
	for _, n := range sirIn[sirSlot:] {
		in = append(in, "f."+n)
	}
	out := []string{"resistCalls"}
	for _, f := range sfbResistOut {
		for k := 0; k < sfbCalls; k++ {
			out = append(out, fmt.Sprintf("resist%d.%s", k, f))
		}
	}
	out = append(out, "durationCalls")
	for _, f := range sfbDurOut {
		for k := 0; k < sfbCalls; k++ {
			out = append(out, fmt.Sprintf("duration%d.%s", k, f))
		}
	}
	Nodes["0054dbe0"] = Node{Addr: "0054dbe0", In: in, Out: out, F: spellStatusFallbackNode,
		Emu: func(in []int64, r Rand) []int64 { return sfbRun(in, r, false) }}
}

func spellStatusFallbackNode(in []int64, r Rand) []int64 { return sfbRun(in, r, true) }

func sfbRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != sfbF+len(sirIn)-sirSlot {
		panic("spell status fallback: wrong input count")
	}
	u := func(i int) uint32 { return uint32(in[i]) }
	if u(sfbSpell) >= sfbSpells {
		panic("spell status fallback: spell id past the fixture's 256 records")
	}
	chars, mons := int32(u(sfbCharCount)), int32(u(sfbMonCount))
	if chars > sfbTargets || mons > sfbTargets || u(sfbListA) > sfbTargets {
		panic("spell status fallback: list count past the fixture's 8")
	}
	srcIndex := uint32(sfbNoSource)
	if u(sfbSrcKind) == 1 { // 0x53bea0
		if int32(u(sfbSrcIndex)) == -1 {
			panic("spell status fallback: source character index -1 (0x53bea0 asserts)")
		}
		srcIndex = u(sfbSrcIndex)
	}
	resist := make([][]uint32, len(sfbResistOut))
	dur := make([][]uint32, len(sfbDurOut))
	for i := range resist {
		resist[i] = make([]uint32, sfbCalls)
	}
	for i := range dur {
		dur[i] = make([]uint32, sfbCalls)
	}
	nr, nd := 0, 0
	var tIdx, tID uint32 // the target struct is never cleared (fresh stack: 0); skipped targets leave it
	hit := func(kind, entry uint32) {
		if uint32(r.R(100)) >= u(sfbChance) {
			return
		}
		d := u(sfbPerLevel)*u(sfbLevel) + u(sfbBase)
		if d != sfbFixed {
			d++
			if roll := r.R(4); roll == 0 && d > 1 || roll == 1 && d <= 2 {
				d++
			}
			d += d * u(sfbBonus) / 100 // 0x517f90
		}
		if kind == 3 {
			tID = entry // the record found has this id at +0
		} else if tIdx = entry; entry == 0xffffffff {
			panic("spell status fallback: character index -1 (assert at 0x54dcaa)")
		}
		for i, v := range []uint32{0x2000005c, u(sfbStatus), u(sfbRealm), u(sfbStrength), u(sfbPenalty), d, 0, 1, 0, kind, tIdx, tID} {
			resist[i][nr] = v
		}
		var resisted byte
		if real {
			v := make([]int64, len(sirIn))
			v[sirStatus], v[sirRealm], v[sirStrength], v[sirPenalty], v[sirDuration] = in[sfbStatus], in[sfbRealm], in[sfbStrength], in[sfbPenalty], int64(d)
			v[sirMessage], v[sirDurMessage], v[sirDurFlagB] = 0, 1, 0
			v[sirSrcKind], v[sirSrcIndex] = in[sfbSrcKind], int64(int32(u(sfbSrcIndex)))
			v[sirKind], v[sirIdx], v[sirMonID] = int64(kind), int64(int32(tIdx)), int64(tID)
			copy(v[sirSlot:], in[sfbF:])
			resisted = byte(statusImmunityAndResistNode(v, r)[0])
		} else {
			resisted = byte(in[sfbResist+nr])
		}
		nr++
		if resisted == 0 {
			return
		}
		for i, v := range []uint32{u(sfbDurStatus), u(sfbRealm), u(sfbStrength), u(sfbAltPenalty), d, srcIndex, 0, 1, kind, tIdx, tID} {
			dur[i][nd] = v
		}
		nd++
		if real {
			v := sdrParentInput(in[sfbF+sirOwn-sirSlot:])
			v[0], v[1], v[2], v[3], v[4] = in[sfbDurStatus], in[sfbRealm], in[sfbStrength], in[sfbAltPenalty], int64(d)
			v[5], v[6], v[7], v[8], v[9] = 0, 1, int64(kind), int64(tIdx), int64(tID)
			statusDurationAfterResistNode(v, r) // return ignored
		}
	}
	for k := int32(0); k < chars; k++ {
		hit(1, u(sfbChar+int(k)))
	}
	for k := int32(0); k < mons; k++ {
		hit(3, u(sfbMonID+int(k)))
	}
	out := []int64{int64(nr)}
	for _, f := range resist {
		for _, v := range f {
			out = append(out, int64(v))
		}
	}
	out = append(out, int64(nd))
	for _, f := range dur {
		for _, v := range f {
			out = append(out, int64(v))
		}
	}
	return out
}
