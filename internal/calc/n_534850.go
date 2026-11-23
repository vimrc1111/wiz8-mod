package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
)

// Node 0x534850: does monster M have a target in reach, either a party
// character or another monster. Constants and the special-class table are read
// from the player's Wiz8.exe at load time; nothing from the exe is embedded.
type monsterTargetInRangeTables struct {
	templateScale, scaleMul, lengthMul, floorBase, floorExtra float64     // 0x5ed4f0, 0x5ebc64, 0x5ebc40, 0x5ec360, 0x5ee77c
	special                                                   [256]uint32 // 0x61eefc, 8-byte stride
	lengthDest                                                [5]uint32   // 0x51aa10 jump table of 0x51a9a0
}

var monsterTargetInRangeData, monsterTargetInRangeDataErr = loadMonsterTargetInRangeTables()

func loadMonsterTargetInRangeTables() (monsterTargetInRangeTables, error) {
	var t monsterTargetInRangeTables
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("monster target in range: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	read := func(va uint32) (uint32, error) {
		off, e := peOffset(b, va)
		if e != nil {
			return 0, e
		}
		if off > len(b)-4 {
			return 0, fmt.Errorf("monster target in range: truncated table %#x", va)
		}
		return binary.LittleEndian.Uint32(b[off:]), nil
	}
	for _, c := range []struct {
		va  uint32
		dst *float64
	}{{0x5ed4f0, &t.templateScale}, {0x5ebc64, &t.scaleMul}, {0x5ebc40, &t.lengthMul}, {0x5ec360, &t.floorBase}, {0x5ee77c, &t.floorExtra}} {
		x, e := read(c.va)
		if e != nil {
			return t, e
		}
		*c.dst = float64(math.Float32frombits(x))
	}
	for i := range t.special {
		if t.special[i], err = read(0x61eefc + 8*uint32(i)); err != nil {
			return t, err
		}
	}
	for i := range t.lengthDest {
		if t.lengthDest[i], err = read(0x51aa10 + 4*uint32(i)); err != nil {
			return t, err
		}
	}
	return t, nil
}

var monsterTargetInRangeInputs = func() []string {
	n := []string{"monsterPointer", "skipList", "wantedRelation", "useRange", "monsterListPointer", "monsterCount", "entry0", "entry1", "entry2", "entry3",
		"mSpecies", "mDisposition", "mAllegiance", "mState", "mFlag", "mXBits", "mYBits", "mZBits", "mRadiusBits", "relationCount"}
	for k := 0; k < 3; k++ {
		n = append(n, fmt.Sprintf("relation%dId", k), fmt.Sprintf("relation%dActive", k), fmt.Sprintf("relation%dReach", k))
	}
	for k := 0; k < 4; k++ {
		for _, f := range []string{"Id", "Species", "Active", "Visible", "Disposition", "HP", "Allegiance", "Status", "XBits", "YBits", "ZBits", "RadiusBits"} {
			n = append(n, fmt.Sprintf("mon%d%s", k, f))
		}
	}
	n = append(n, "partyXBits", "partyYBits", "partyZBits", "partyRadiusBits")
	for c := 0; c < 8; c++ {
		n = append(n, fmt.Sprintf("slot%dActive", c), fmt.Sprintf("character%dHP", c), fmt.Sprintf("character%dStatus", c))
	}
	for t := 0; t < 5; t++ {
		n = append(n, fmt.Sprintf("tpl%dScaleBits", t))
		for j := 0; j < 3; j++ {
			n = append(n, fmt.Sprintf("tpl%dAttack%dPresent", t, j), fmt.Sprintf("tpl%dAttack%dRange", t, j))
		}
		n = append(n, fmt.Sprintf("tpl%dSpecial", t), fmt.Sprintf("tpl%dGroup", t))
		for i := 0; i < 10; i++ {
			n = append(n, fmt.Sprintf("tpl%dSpell%dId", t, i), fmt.Sprintf("tpl%dSpell%dKind", t, i), fmt.Sprintf("tpl%dSpell%dRange", t, i))
		}
	}
	for k := 0; k < 8; k++ {
		n = append(n, fmt.Sprintf("ret%d", k))
	}
	// F-only: C+0xa35 of each character, read by the real 0x546f10. Emu's
	// address stub ignores them (they sit in a scratch area).
	for c := 0; c < 8; c++ {
		n = append(n, fmt.Sprintf("charm%d", c))
	}
	return n
}()

var monsterTargetInRangeOutputs = func() []string {
	n := []string{"inRange", "stubCalls"}
	for k := 0; k < 8; k++ {
		n = append(n, fmt.Sprintf("stubCharacter%d", k))
	}
	for k := 0; k < 8; k++ {
		n = append(n, fmt.Sprintf("stubMonster%d", k))
	}
	return n
}()

func init() {
	Nodes["00534850"] = Node{Addr: "00534850", In: monsterTargetInRangeInputs, Out: monsterTargetInRangeOutputs,
		F:   monsterTargetInRangeNode,
		Emu: func(in []int64, r Rand) []int64 { return monsterTargetInRange(in, r, false) }}
}

// Emulator fixture addresses (arena base 0x20000000), not game data: the
// monster under test and the list header; a list entry is the monster itself
// or its own fixture record.
const (
	monsterTargetInRangeSelf = 0x20000000 + 0x28000
	monsterTargetInRangeList = 0x20000000 + 0x26000
)

func monsterTargetInRangeEntry(k int) int64 { return 0x20000000 + 0x28400 + 0x400*int64(k) }

func monsterTargetInRangeNode(in []int64, r Rand) []int64 { return monsterTargetInRange(in, r, true) }

// monsterTargetInRange: Emu (real=false) takes each 0x546f10 return from
// ret0..7 as the address stub does; F (real=true) runs the real
// targetDispositionNode on (character, charm, M+0x16) instead.
func monsterTargetInRange(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(monsterTargetInRangeInputs) {
		panic("monster target in range: unexpected input count")
	}
	if monsterTargetInRangeDataErr != nil {
		panic(monsterTargetInRangeDataErr)
	}
	t := monsterTargetInRangeData
	v := make(map[string]int64, len(in))
	for i, name := range monsterTargetInRangeInputs {
		v[name] = in[i]
	}
	u := func(name string, max int64) int64 {
		x := v[name]
		if x < 0 || x > max {
			panic("monster target in range: " + name + " out of range")
		}
		return x
	}
	const u8, u32 = 255, math.MaxUint32
	arg := u("monsterPointer", u32)
	if arg != monsterTargetInRangeSelf {
		panic("monster target in range: fixture monster pointer")
	}
	skipList, wanted, useRange := u("skipList", u32), u("wantedRelation", u8), u("useRange", u8)
	listPtr := u("monsterListPointer", u32)
	if listPtr != 0 && listPtr != monsterTargetInRangeList {
		panic("monster target in range: list pointer must be null or the fixture list")
	}
	count := u("monsterCount", u32)
	if listPtr == 0 {
		count = 0
	}
	if count > 4 || u("relationCount", u32) > 3 {
		panic("monster target in range: fixture holds at most 4 monsters and 3 relations")
	}
	f := monsterAttackTargetFloat
	pos := func(p string) [3]float64 { return [3]float64{f(v[p+"XBits"]), f(v[p+"YBits"]), f(v[p+"ZBits"])} }

	// Reach threshold (0x534861..0x5348b6); only built when the caller asks.
	var limit *big.Float
	if useRange != 0 {
		limit = monsterTargetInRangeLimit(v, t, monsterAttackTargetTemplate(u("mSpecies", u32)))
	}
	selfPos, selfRadius := pos("m"), f(v["mRadiusBits"])
	within := func(d *big.Float) bool { return d.Cmp(limit) <= 0 } // false only when d is above the limit

	var calls []int64 // every 0x546f10 call: character index, then the monster pointer
	result := func(x int64) []int64 {
		out := []int64{x, int64(len(calls))}
		var chars, mons [8]int64
		for i, c := range calls {
			chars[i], mons[i] = c, arg
		}
		out = append(out, chars[:]...)
		return append(out, mons[:]...)
	}
	// 0x53492a / 0x5349c2: the wanted relation matches 3 (any), the same value,
	// or 4 with any nonzero value.
	related := func(got int64, want int64) bool {
		got &= 0xff
		return want == 3 || got == want || (want == 4 && got != 0)
	}

	// 0x5348ba: party characters, only for a monster in state 1 with its flag set.
	if u("mState", u8) == 1 && u("mFlag", u8) != 0 {
		party := [3]float64{f(v["partyXBits"]), f(v["partyYBits"]), f(v["partyZBits"])}
		if useRange == 0 || within(monsterAttackTargetDistance(selfPos, party, true, selfRadius, f(v["partyRadiusBits"]))) {
			for c := 0; c < 8; c++ {
				if u(fmt.Sprintf("slot%dActive", c), u8) == 0 || u(fmt.Sprintf("character%dHP", c), u32) == 0 || u(fmt.Sprintf("character%dStatus", c), u32) >= 0x12 {
					continue
				}
				got := u(fmt.Sprintf("ret%d", len(calls)), u32)
				if real {
					charm := int64(int32(u(fmt.Sprintf("charm%d", c), u32)))
					got = targetDispositionNode([]int64{int64(c), charm, u("mDisposition", u8)}, r)[0]
				}
				calls = append(calls, int64(c))
				if related(got, wanted) {
					return result(1)
				}
			}
		}
	}
	if skipList != 0 {
		return result(0)
	}

	// 0x534987: other monsters of the list, the list count re-read each pass.
	for i := 0; i < int(count); i++ {
		p := fmt.Sprintf("mon%d", i)
		switch v[fmt.Sprintf("entry%d", i)] {
		case monsterTargetInRangeSelf:
			continue
		case monsterTargetInRangeEntry(i):
		default:
			panic("monster target in range: list entry must be the monster itself or its fixture record")
		}
		if u(p+"Active", u8) == 0 || u(p+"HP", u32) == 0 || u(p+"Status", u32) >= 0x12 || u(p+"Visible", u8) == 0 {
			continue
		}
		if !related(monsterTargetInRangeStance(v, t, "m", p), wanted) {
			continue
		}
		// 0x505d20: the first relation record naming this monster decides.
		rel := -1
		for k := 0; k < int(v["relationCount"]); k++ {
			if v[fmt.Sprintf("relation%dId", k)] == v[p+"Id"] {
				rel = k
				break
			}
		}
		if rel < 0 || u(fmt.Sprintf("relation%dActive", rel), u8) != 1 || u(fmt.Sprintf("relation%dReach", rel), u8) == 0 {
			continue
		}
		if useRange == 0 || within(monsterAttackTargetDistance(selfPos, pos(p), false, selfRadius, f(v[p+"RadiusBits"]))) {
			return result(1)
		}
	}
	return result(0)
}

// 0x546f80 between the monster record a and b (field prefixes in v):
// 0 neutral or special, 2 same disposition, 1 hostile.
func monsterTargetInRangeStance(v map[string]int64, t monsterTargetInRangeTables, a, b string) int64 {
	if v[a+"Species"] == 0x224 || v[b+"Species"] == 0x224 {
		return 0
	}
	if v[a+"Disposition"] == v[b+"Disposition"] {
		return 2
	}
	if v[a+"Disposition"] == 0 || v[b+"Disposition"] == 0 {
		return 0
	}
	g := func(p string) int64 { return v[fmt.Sprintf("tpl%dGroup", monsterAttackTargetTemplate(v[p+"Species"]))] }
	if g(a) == 0 || g(b) == 0 || g(a) != g(b) {
		return 1
	}
	if (v[a+"Allegiance"] != 0) == (v[b+"Allegiance"] != 0) {
		return 0
	}
	return 1
}

// monsterTargetInRangeLimit is the threshold built at 0x534861..0x5348b6.
// Every x87 step keeps a 64-bit significand; the stores are float32.
func monsterTargetInRangeLimit(v map[string]int64, t monsterTargetInRangeTables, tpl int) *big.Float {
	ext, op := monsterAttackTargetExt, monsterAttackTargetOp
	f32 := func(x *big.Float) *big.Float { r, _ := x.Float32(); return ext(float64(r)) }
	// 0x4e5990 scaled by 0x5ebc64 and stored.
	scale := monsterAttackTargetFloat(v[fmt.Sprintf("tpl%dScaleBits", tpl)])
	base := f32(op((*big.Float).Mul, op((*big.Float).Mul, ext(scale), ext(t.templateScale)), ext(t.scaleMul)))
	class := monsterTargetInRangeClass(v, t, tpl)
	// 0x51a9a0: reach class -1..3 to a length, times 0x5ebc40.
	length := int64(-1)
	for i, d := range []uint32{0x51a9b7, 0x51a9bb, 0x51a9c2, 0x51a9c9, 0x51a9d0} {
		if t.lengthDest[class+1] == d {
			length = []int64{0, 2, 4, 25, 50}[i]
		}
	}
	if length < 0 {
		panic("monster target in range: unknown length dispatch")
	}
	scaled := op((*big.Float).Mul, new(big.Float).SetPrec(64).SetInt64(length), ext(t.lengthMul))
	reach := f32(op((*big.Float).Add, scaled, base))
	// 0x51b300 + 0x5ee77c is the floor; the larger one is the limit (equal: the floor).
	floor := op((*big.Float).Add, ext(t.floorBase), ext(t.floorExtra))
	if floor.Cmp(reach) < 0 {
		return reach
	}
	return f32(floor)
}

// monsterTargetInRangeClass is 0x51a840 for flag 1: the monster's longest
// attack reach class (-1 none, else 0..3 or an assertion).
func monsterTargetInRangeClass(v map[string]int64, t monsterTargetInRangeTables, tpl int) int {
	best := int64(-1)
	for j := 0; j < 3; j++ {
		if v[fmt.Sprintf("tpl%dAttack%dPresent", tpl, j)] != 0 && v[fmt.Sprintf("tpl%dAttack%dRange", tpl, j)] > best {
			best = v[fmt.Sprintf("tpl%dAttack%dRange", tpl, j)]
		}
	}
	class := best
	if sp := v[fmt.Sprintf("tpl%dSpecial", tpl)]; sp != 0 && sp <= 255 && t.special[sp] != 6 && best < 2 {
		class = 2
	}
	// Spell records by id, later fixture writes replacing earlier ones.
	type rec struct{ kind, reach int64 }
	records := map[int64]rec{}
	for u := 0; u < 5; u++ {
		for i := 0; i < 10; i++ {
			p := fmt.Sprintf("tpl%dSpell%d", u, i)
			records[v[p+"Id"]] = rec{int64(int32(v[p+"Kind"])), int64(int32(v[p+"Range"]))}
		}
	}
	for i := 0; i < 10; i++ {
		id := v[fmt.Sprintf("tpl%dSpell%dId", tpl, i)]
		if id < 0 || id >= 150 {
			panic("monster target in range: spell id assertion")
		}
		// 0x5474b0: spells 3 and 0x29 never, others when the kind is 3..7.
		if id == 3 || id == 0x29 {
			continue
		}
		if r := records[id]; r.kind >= 3 && r.kind <= 7 && class < r.reach {
			class = r.reach
		}
	}
	if class < -1 || class > 3 {
		panic("monster target in range: reach class assertion")
	}
	return int(class)
}
