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

// stamina_recovery 0x504730(who, ticks): one recovery tick batch for party
// member `who` (record at *0x685174, stride 0x1862; build/wiz8.asm:350886..),
// notes/re/abilities.md:23.
//
// The signed byte +0x177a is the stamina change per tick. Positive and stamina
// +0xb1d below the maximum +0xb19 (signed): heal 0x52b1c0(who, rate*ticks, 0).
// Negative: spend 0x52af50(who, -(rate*ticks), 0, 0). Both products are 32-bit.
// Then the fractional rate f is chosen: 1.0 when byte 0x683fc5 is set, else by
// game mode 0x687509 (3: 0.5; 0 or 2: 0.1; others: 0.0); times 0.25 when +0xa09
// is non-zero; when 0x547940(C, 0) says the character has feature 0 (class row
// +0x69 or race row +0x71, tables 0x61507c/0x615130) f becomes f*1.5, or, when f
// is 0 and byte 0x683f94 is set, 0x5479b0(C, 0, 3.2f) = (2L+60)*3.2*0.01 for the
// class level L at +0x8d+4*class (L <= 20, else 3.2). While f > 0 and stamina is
// below the maximum, the x87 sum ticks*(+0xb71)*f + (+0xb75) is kept in +0xb75
// as binary32, its truncation (0x5e1c22) heals 0x52b1c0(who, int, 0), and the
// fraction left is +0xb75 minus the truncated stored value (zero-extended low
// 32 bits). Stamina at or above the maximum afterwards zeroes +0xb75.
//
// Emu address-stubs the D4 callees 0x52b1c0 (at most twice) and 0x52af50 (once)
// (composition ruling 2026-10-09): the caller ignores their eax, so the outputs
// are the call counts and every call's arguments. The node reads +0xb1d after
// them, so F also runs the real characterStaminaHealNode and
// characterFatigueApplyNode and feeds the new +0xb1d forward. F-only inputs:
// state +0xb01 and hit points +0xb11 (the only other callee fields that decide
// the new stamina; the callees' notice, fatigue and state outputs are not read here).
//
// Fixture caps (calc panics past them): who 0..7, class -1..14, race -1..15.
// Non-finite or int64-overflowing float chains fault in the original truncation
// helper and panic here. The unreachable-by-tables case (a level read for a
// class without feature 0) is still modelled: the level slot is an input.
const (
	strcChars   = 0x20000000
	strcMembers = 8
	strcIn      = 11 + 15 + 3 + 2 // who..levelM1, level0..14, three globals, state, hp
)

const (
	strcWho = iota
	strcTicks
	strcCur
	strcMax
	strcFlag
	strcRate
	strcB71
	strcB75
	strcClass
	strcRace
	strcLevelM1
	strcLevel0
	strcG5fc5 = strcLevel0 + 15
	strcMode  = strcG5fc5 + 1
	strcG3f94 = strcMode + 1
	strcState = strcG3f94 + 1
	strcHP    = strcState + 1
)

type strcFloats struct{ zero, one, tenth, half, quarter, threeHalves float32 }

var strcConst, strcConstErr = strcLoad()

// strcLoad reads the six binary32 constants of 0x504730 from the player's Wiz8.exe.
func strcLoad() (c strcFloats, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return c, fmt.Errorf("stamina recovery: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	for i, p := range []*float32{&c.zero, &c.one, &c.tenth, &c.half, &c.quarter, &c.threeHalves} {
		va := []uint32{0x5ebb34, 0x5ebb38, 0x5ebc3c, 0x5ebc7c, 0x5ebcc8, 0x5ec3b8}[i]
		o, e := peOffset(b, va)
		if e != nil || o+4 > len(b) {
			return c, fmt.Errorf("stamina recovery: constant %#x not in exe", va)
		}
		*p = math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))
	}
	return c, nil
}

func init() {
	names := []string{"who", "ticks", "currentStamina", "maximumStamina", "recoveryFlag", "staminaRate", "rateScale",
		"fractionBits", "classIndex", "raceIndex", "levelM1"}
	for k := 0; k < 15; k++ {
		names = append(names, fmt.Sprintf("level%d", k))
	}
	names = append(names, "g683fc5", "g687509", "g683f94", "state", "hitPoints")
	// positional names of the case file's outputs
	out := []string{"fractionBits", "currentStamina", "n52b1c0", "a52b1c0_0_0", "a52b1c0_0_1", "a52b1c0_0_2",
		"a52b1c0_1_0", "a52b1c0_1_1", "a52b1c0_1_2", "n52af50", "a52af50_0", "a52af50_1", "a52af50_2", "a52af50_3"}
	Nodes["00504730"] = Node{Addr: "00504730", In: names, Out: out,
		F:   func(in []int64, r Rand) []int64 { return staminaRecoveryNode(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return staminaRecoveryNode(in, r, false) }}
}

func strcBig(x float32) *big.Float {
	if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
		panic("stamina recovery: non-finite binary32 operand")
	}
	return chdF(float64(x))
}

// strcLevelScale is 0x5479b0(C, 0, 3.2f) for class level l.
func strcLevelScale(l uint32) float32 {
	m := math.Float32frombits(0x40533333) // immediate pushed at 0x504835
	if l > 20 {
		return m
	}
	if statusRebuildTablesErr != nil {
		panic(statusRebuildTablesErr)
	}
	two := chdAdd(chdF(float64(l)), chdF(float64(l)))
	v := chdMul(chdMul(chdAdd(two, chdF(statusRebuildTables.featureBase)), chdF(float64(m))), chdF(statusRebuildTables.featureScale))
	f, _ := v.Float32()
	return f
}

func staminaRecoveryNode(in []int64, r Rand, real bool) []int64 {
	if len(in) != strcIn {
		panic(fmt.Sprintf("stamina recovery: expected %d inputs, got %d", strcIn, len(in)))
	}
	if strcConstErr != nil {
		panic(strcConstErr)
	}
	if breathTablesErr != nil {
		panic(breathTablesErr)
	}
	who := in[strcWho]
	if who < 0 || who >= strcMembers {
		panic("stamina recovery: member outside the eight fixture records")
	}
	cls, race := in[strcClass], in[strcRace]
	if cls < -1 || cls > 14 || race < -1 || race > 15 {
		panic("stamina recovery: class or race outside the table rows")
	}
	c := strcConst
	ticks := uint32(in[strcTicks])
	cur, max := int32(in[strcCur]), int32(in[strcMax])
	rate := int32(int8(in[strcRate]))
	b75 := uint32(in[strcB75])
	state, hp := in[strcState], in[strcHP]

	var n52b int64
	var a52b [2][3]int64
	var n52a int64
	var a52a [4]int64
	heal := func(h int32) { // 0x52b1c0(who, h, 0)
		if n52b >= 2 {
			panic("stamina recovery: third heal call")
		}
		a52b[n52b] = [3]int64{who, int64(h), 0}
		n52b++
		if real {
			v := []int64{strcChars, who, int64(h), 0, state, hp, int64(max), int64(cur), 0, 0, 0, 0}
			cur = int32(characterStaminaHealNode(v, r)[0])
		}
	}
	out := func() []int64 {
		o := []int64{int64(b75), int64(cur), n52b}
		for k := 0; k < 2; k++ {
			o = append(o, a52b[k][:]...)
		}
		o = append(o, n52a)
		return append(o, a52a[:]...)
	}

	flag := uint32(in[strcFlag]) != 0 // 0x504752..0x504762: unsigned > 0
	step := rate * int32(ticks)       // imul, 32-bit
	if rate > 0 {
		if cur < max { // 0x504776
			heal(step)
		}
	} else if rate < 0 {
		n52a = 1 // 0x52af50(who, -step, 0, 0)
		a52a = [4]int64{who, int64(-step), 0, 0}
		if real {
			v := []int64{who, int64(-step), 0, 0, int64(cur), hp, int64(max), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
			cur = int32(characterFatigueApplyNode(v, r)[0])
		}
	}

	var f float32
	switch {
	case in[strcG5fc5]&0xff != 0: // 0x5047af
		f = c.one
	case uint32(in[strcMode]) == 3:
		f = c.half
	case uint32(in[strcMode]) == 0 || uint32(in[strcMode]) == 2:
		f = c.tenth
	default:
		f = c.zero
	}
	if flag {
		f = float32(float64(f) * float64(c.quarter)) // exact product, one rounding
	}
	// 0x547940(C, 0): feature code 0 in the class row or the race row.
	has0 := false
	if cls != -1 {
		for _, x := range breathTables.class[cls] {
			has0 = has0 || x == 0
		}
	}
	if race != -1 {
		for _, x := range breathTables.race[race] {
			has0 = has0 || x == 0
		}
	}
	if has0 {
		if f != c.zero { // 0x504814
			f = float32(float64(f) * float64(c.threeHalves))
		} else if in[strcG3f94]&0xff != 0 {
			slot := strcLevelM1
			if cls != -1 {
				slot = strcLevel0 + int(cls)
			}
			f = strcLevelScale(uint32(in[slot]))
		}
	}
	if f > c.zero && cur < max { // 0x504848, 0x504859
		sum := chdAdd(chdMul(chdMul(chdF(float64(ticks)), strcBig(math.Float32frombits(uint32(in[strcB71])))), strcBig(f)),
			strcBig(math.Float32frombits(b75)))
		stored, _ := sum.Float32() // fst to +0xb75
		b75 = math.Float32bits(stored)
		heal(int32(chdFtol(sum))) // truncation of the unrounded x87 value
		whole := chdFtol(strcBig(stored))
		left := new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Sub(strcBig(stored), chdF(float64(whole)))
		stored, _ = left.Float32()
		b75 = math.Float32bits(stored)
	}
	if cur >= max { // 0x5048bc
		b75 = 0
	}
	return out()
}
