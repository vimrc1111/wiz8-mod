package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// status_resist_roll 0x5520d0(T, realm, strength, status): does the target resist
// a status effect. T is a target record (kind at +0: 1 = character, 3 = monster;
// +4 character index; +8 monster id).
//
//	level = character +0x89 | monster template +0x24f (u8)
//	Rj    = character +0xee1+16*realm | template +0x176+realm (u8) + live monster +0x217+realm (i8)
//	a     = 3*(level-strength) + bonus[status] + Rj          (32-bit wrap; bonus table: exe 0x6172a0)
//	a     = difficulty(a) -> clamp to [5,95] (signed) -> resist when R(100) < a
//
// A primary state word (character +0xb01 / monster +0x107) >= 0x12 (unsigned)
// resists at once with no roll. A resisting character whose byte +0x6f5 is
// non-zero also calls natural_skill_growth 0x553f10(char, 0x24, 2, 0).
//
// Difficulty: character -> 0x55cc00 (D4 node, difficultyNode); monster -> 0x55ccb0
// (unregistered helper, run natively in emu): percent by monster byte +0x16 and
// the difficulty global 0x6850d5, 32-bit product then signed /100.
// Emu replays the address stubs of 0x55cc00 (its write comes from input
// "difficultyResult") and 0x553f10 (args and call count are outputs); F runs the
// real difficultyNode. 0x553f10's effects are never read back, so it needs no F branch.
//
// Fixture caps (panic past them): character index 0..7, template index 0..2,
// realm 0..5, status 0..0x13 (highest dynamic caller bound, 0x551c58), monster id != 0xffffffff.
// Invalid target kind and character index -1 reach srAssertFail (0x552136, 0x552152): panic.
const (
	srrCharBase   = 0x20001000 // case object "character"; record stride 0x1862 (0x55216f..0x552188)
	srrCharStride = 0x1862
	srrBonusVA    = 0x6172a0
	srrMaxStatus  = 0x13
)

var srrIn = func() []string {
	n := []string{"targetKind", "characterIndex", "monsterID", "realm", "strength", "status", "difficulty", "characterLevel"}
	realms := []string{"fire", "water", "air", "earth", "mental", "divine"}
	for _, r := range realms {
		n = append(n, "characterResistance_"+r)
	}
	n = append(n, "characterPrimaryState", "characterSkillGrowthFlag", "characterReverseDifficulty", "templateIndex")
	for _, r := range realms {
		n = append(n, "templateResistance_"+r)
	}
	n = append(n, "templateLevel")
	for _, r := range realms {
		n = append(n, "monsterResistanceModifier_"+r)
	}
	return append(n, "monsterPrimaryState", "monsterDifficultyClass", "difficultyResult")
}()

func init() {
	Nodes["005520d0"] = Node{Addr: "005520d0", In: srrIn,
		Out: []string{"resisted", "growthCalls", "growthCharacter", "growthSkill", "growthRepeat", "growthQuiet", "difficultyCalls", "difficultySlot"},
		F:   func(in []int64, r Rand) []int64 { return statusResistRoll(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return statusResistRoll(in, r, false) }}
}

func statusResistRollNode(in []int64, r Rand) []int64 { return statusResistRoll(in, r, true) }

var srrBonus, srrBonusErr = srrLoadBonus()

// The status bonus table stays in the player's executable.
func srrLoadBonus() (t [srrMaxStatus + 1]int32, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("status resist roll: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	o, err := peOffset(b, srrBonusVA)
	if err != nil {
		return t, err
	}
	if len(b)-o < 4*len(t) {
		return t, fmt.Errorf("status resist roll: truncated bonus table")
	}
	for i := range t {
		t[i] = int32(binary.LittleEndian.Uint32(b[o+4*i:]))
	}
	return t, nil
}

func statusResistRoll(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(srrIn) {
		panic("status resist roll: wrong input count")
	}
	if srrBonusErr != nil {
		panic(srrBonusErr)
	}
	const (
		iKind, iWho, iMonID, iRealm, iStrength, iStatus, iDiff, iCharLevel = 0, 1, 2, 3, 4, 5, 6, 7
		iCharRes, iCharPrimary, iGrowth, iReverse, iTplIndex               = 8, 14, 15, 16, 17
		iTplRes, iTplLevel, iMonMod, iMonPrimary, iMonClass, iDiffResult   = 18, 24, 25, 31, 32, 33
	)
	for i, v := range in {
		lo, hi := int64(0), int64(0xffffffff)
		if i >= iMonMod && i < iMonMod+6 || i == iDiffResult {
			lo, hi = -128, 127
			if i == iDiffResult {
				lo, hi = -0x80000000, 0x7fffffff
			}
		} else if i >= iTplRes && i <= iTplLevel || i == iMonClass || i == iGrowth {
			hi = 255
		}
		if v < lo || v > hi {
			panic("status resist roll: input out of field range")
		}
	}
	kind, realm, status := in[iKind], int(in[iRealm]), in[iStatus]
	if kind != 1 && kind != 3 {
		panic("status resist roll: target kind not 1 or 3 (assert 0x552136)")
	}
	if realm > 5 || status > srrMaxStatus {
		panic("status resist roll: realm or status beyond the case table")
	}
	who := in[iWho]
	var level, rj, primary uint32
	if kind == 3 {
		if in[iTplIndex] > 2 || in[iMonID] == 0xffffffff {
			panic("status resist roll: template index or monster id beyond the case lists")
		}
		level = uint32(in[iTplLevel])
		rj = uint32(int32(in[iMonMod+realm]) + int32(in[iTplRes+realm]))
		primary = uint32(in[iMonPrimary])
	} else {
		if who == 0xffffffff {
			panic("status resist roll: character index -1 (assert 0x552152)")
		}
		if who > 7 {
			panic("status resist roll: character index beyond the case roster")
		}
		level, rj, primary = uint32(in[iCharLevel]), uint32(in[iCharRes+realm]), uint32(in[iCharPrimary])
	}
	out := make([]int64, 8)
	if primary >= 0x12 { // 0x5521a1: already past the state range, resists with no roll
		out[0] = 1
		return out
	}
	a := int32(3*(level-uint32(in[iStrength])) + uint32(srrBonus[status]) + rj)
	if kind == 3 {
		pct := int32(100) // 0x55ccb0
		switch d, k := uint32(in[iDiff]), in[iMonClass]; {
		case k == 1 && d == 0, k == 2 && d == 2:
			pct = 60
		case k == 1 && d == 2, k == 2 && d == 0:
			pct = 140
		}
		if pct != 100 {
			a = int32(uint32(a)*uint32(pct)) / 100
		}
	} else {
		out[6], out[7] = 1, who // 0x55cc00(who, &a)
		if real {
			a = int32(difficultyNode([]int64{0, 0, 0, 0, 0, 0, 0, 0, int64(a), int64(int32(uint32(in[iDiff]))), int64(int32(uint32(in[iReverse]))), who}, r)[0])
		} else {
			a = int32(in[iDiffResult])
		}
	}
	a = max(5, min(95, a)) // 0x5179d0(&a, 5, 0x5f), signed
	resisted := false
	if bl, ok := r.(interface{ Below(n, c int) bool }); ok && real { // same distribution, one two-way branch
		resisted = bl.Below(100, int(a))
	} else {
		resisted = int32(r.R(100)) < a
	}
	if !resisted {
		return out
	}
	out[0] = 1
	if kind == 1 && in[iGrowth] != 0 { // character byte +0x6f5 (0x552227)
		out[1], out[2], out[3], out[4], out[5] = 1, srrCharBase+who*srrCharStride, 0x24, 2, 0
	}
	return out
}
