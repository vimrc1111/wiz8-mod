package calc

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// 0x58a060 trap_inspect_chance(panel): the trap-inspection panel (ecx) rolls, for the
// character panel+0x1c, whether each of the trap's eight parts is recognised.
//
//  1. panel+0x150 = 0; 0x553f10(character, 10, 1, 0) lets the trap skill (10) grow
//     naturally. Its writes to the character that this function reads afterwards are the
//     usable flag (byte +0x319) and the effective skill (dword +0x31f).
//  2. 0x589d90 refreshes the panel widgets (presentation only; nothing it writes is read
//     here, the case runs it natively over a fixed widget fixture and the character 0x68518d).
//  3. skill = -1 unless 0x524a10(c) holds (characterActionEligibilityNode) and the flag
//     or the effective skill is nonzero; otherwise skill = +0x31f, plus 6*panel+0x14c when
//     it is >= 0 (all 32-bit wrapping).
//  4. chance = ((skill - T[max(panel+0x38,0)]) * 3) / 2 (toward zero) + (11 - difficulty) * 10,
//     with T the dword table at exe 0x6504e8 and difficulty the dword at 0x6850d5, clamped to
//     0..99.
//  5. for part i = 0..7 one R(100): below the chance the cell panel+0x4c+4*(8c+i) gets
//     0x5e3730(panel+0x34, i) & 0xff (byte at exe 0x650434 + 8*row + i), else 2.
//     Then byte panel+0x44+c = 1 and the widget block pointer +0x80 = &cell(c, 0).
//  6. per part i, button i's +0xb8 gets the cell when it is nonzero or when
//     0x5e3730(block+0x78, i) is nonzero (the cell is then 0), else 0xffffffff; every button
//     gets 0x4f4650(0) (its display refresh, which sets button +0xa0 to 1; the child
//     widget pointer is null in the case, so nothing else is touched).
//
// Emu address-stubs the D4 callee 0x553f10 (composition ruling 2026-10-09): the case records its
// four arguments and the call count, and its two writes read above come from the inputs
// postUsableFlag and postSkill; its return value is ignored by this function. F runs the real
// naturalSkillGrowthNode on the F-only inputs sg_* (the 553 inputs of the character, the
// same names as in n_553f10.go without slot/charAddr/s/n/quiet, which are c, the character
// address, 10, 1 and 0) and feeds its f19d10 and eff10 outputs forward instead. F also spends
// its R(100) draws first, as the original does inside 0x553f10.
//
// 0x524a10 runs natively in the case (its character index, slot byte, HP and status are inputs).
// 0x40efa0 is the case's RNG. 0x5e3730 and 0x4f4650 run natively; 0x589d90 is the
// non-formula widget refresh.
//
// Fixture caps (calc panics past them): c must be one of the eight party slots (the panel
// arrays are indexed with it unchecked). Table reads outside the exe panic.
//
// Output cells that the function never writes are the case's fixture constants (ticDecoy*).
const (
	ticArena     = 0x20000000
	ticStride    = 0x1862
	ticPanel     = ticArena + 0x16000 // the case's panel object
	ticRowBase   = 0x4c
	ticRowDecoy  = 0xdead0000 // cell k = 8*c+i before the call: ticRowDecoy+k
	ticFlagDecoy = 0x70       // byte panel+0x44+k before the call: ticFlagDecoy+k
	ticSkill     = 10         // 0x58a08b..0x58a090: second argument of 0x553f10
)

var ticSharedNames = []string{"c", "trapType", "trapTier", "skillBonus", "classRow", "slotActive",
	"currentHP", "mainStatus", "difficulty", "postUsableFlag", "postSkill"}

const (
	ticC = iota
	ticTrapType
	ticTrapTier
	ticBonus
	ticClassRow
	ticActive
	ticHP
	ticStatus
	ticDifficulty
	ticPostFlag
	ticPostSkill
	ticShared
)

var (
	ticExeOnce  sync.Once
	ticExeBytes []byte
	ticExeErr   error
	ticSgNames  []string // F-only inputs: the 553 inputs minus the five call constants
	ticSgIdx    []int    // for every 553 input its index in the node input (-1: constant)
	ticOutF19d  = -1     // index of f19d10 / eff10 in the 553 outputs
	ticOutEff   = -1
)

func init() {
	var all []string
	(&n553In{}).bind(func(name string, _ *int64, _ int) { all = append(all, name) })
	for _, n := range all {
		switch n {
		case "slot", "charAddr", "s", "n", "quiet":
			ticSgIdx = append(ticSgIdx, -1)
		default:
			ticSgIdx = append(ticSgIdx, ticShared+len(ticSgNames))
			ticSgNames = append(ticSgNames, "sg_"+n)
		}
	}
	k := 0
	(&n553Out{}).bind(func(name string, _ *int64) {
		switch name {
		case "f19d10":
			ticOutF19d = k
		case "eff10":
			ticOutEff = k
		}
		k++
	})
	names := append(append([]string{}, ticSharedNames...), ticSgNames...)
	out := []string{"stubCalls", "growArg1", "growArg2", "growArg3", "growArg4", "rowPtr", "p150"}
	for k := 0; k < 64; k++ {
		out = append(out, fmt.Sprintf("row%d_%d", k/8, k%8))
	}
	for k := 0; k < 8; k++ {
		out = append(out, fmt.Sprintf("flag%d", k))
	}
	for k := 0; k < 8; k++ {
		out = append(out, fmt.Sprintf("b8_%d", k))
	}
	for k := 0; k < 8; k++ {
		out = append(out, fmt.Sprintf("a0_%d", k))
	}
	Nodes["0058a060"] = Node{Addr: "0058a060", In: names, Out: out,
		F:   trapInspectChanceNode,
		Emu: func(in []int64, r Rand) []int64 { return ticRun(in, r, false) }}
}

func trapInspectChanceNode(in []int64, r Rand) []int64 { return ticRun(in, r, true) }

func ticLoadExe() ([]byte, error) {
	ticExeOnce.Do(func() {
		path := os.Getenv("WIZ8_EXE")
		if path == "" {
			_, source, _, ok := runtime.Caller(0)
			if !ok {
				ticExeErr = fmt.Errorf("trap inspect: set WIZ8_EXE")
				return
			}
			path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
		}
		ticExeBytes, ticExeErr = os.ReadFile(path)
	})
	return ticExeBytes, ticExeErr
}

// ticRead returns the n bytes of the exe image at VA va (a table in .data); panics outside the file.
func ticRead(va uint32, n int) []byte {
	b, err := ticLoadExe()
	if err != nil {
		panic(err)
	}
	off, err := peOffset(b, va)
	if err != nil || off+n > len(b) {
		panic(fmt.Sprintf("trap inspect: exe table read at %#x outside the file", va))
	}
	return b[off : off+n]
}

// tic5e3730 is 0x5e3730(row, i): the byte at 0x650434 + 8*row + i (32-bit address).
func tic5e3730(row, i uint32) int64 {
	return int64(ticRead(0x650434+row*8+i, 1)[0])
}

func ticRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != ticShared+len(ticSgNames) {
		panic("trap inspect chance: wrong input count")
	}
	rng := func(name string, v, lo, hi int64) int64 {
		if v < lo || v > hi {
			panic(fmt.Sprintf("trap inspect chance: input %s=%d outside %d..%d", name, v, lo, hi))
		}
		return v
	}
	c := rng("c", in[ticC], 0, 7) // the panel arrays are indexed with it unchecked
	trapType := uint32(rng("trapType", in[ticTrapType], 0, 1<<32-1))
	tier := int32(rng("trapTier", in[ticTrapTier], -1<<31, 1<<31-1))
	bonus := int32(rng("skillBonus", in[ticBonus], -1<<31, 1<<31-1))
	classRow := uint32(rng("classRow", in[ticClassRow], 0, 1<<32-1))
	active := rng("slotActive", in[ticActive], 0, 255)
	hp := rng("currentHP", in[ticHP], 0, 1<<32-1)
	status := rng("mainStatus", in[ticStatus], 0, 1<<32-1)
	diff := int32(rng("difficulty", in[ticDifficulty], -1<<31, 1<<31-1))
	flag := rng("postUsableFlag", in[ticPostFlag], 0, 255)
	skillWord := int32(rng("postSkill", in[ticPostSkill], -1<<31, 1<<31-1))

	charPtr := int64(ticArena + ticStride*c)
	if real { // F: the real 0x553f10 on the character, its R(100) draws come first
		v := make([]int64, len(ticSgIdx))
		for i, j := range ticSgIdx {
			if j >= 0 {
				v[i] = in[j]
			}
		}
		i := 0
		(&n553In{}).bind(func(name string, _ *int64, _ int) {
			switch name {
			case "slot":
				v[i] = c
			case "charAddr":
				v[i] = charPtr
			case "s":
				v[i] = ticSkill
			case "n":
				v[i] = 1
			case "quiet":
				v[i] = 0
			}
			i++
		})
		g := naturalSkillGrowthNode(v, r)
		flag = g[ticOutF19d] & 0xff
		skillWord = int32(g[ticOutEff])
	}

	// 0x58a0a2..0x58a109: the skill as the function sees it.
	skill := int32(-1)
	elig := characterActionEligibilityNode([]int64{c, active, hp, status}, r)[0]
	if elig != 0 && (flag != 0 || skillWord != 0) {
		skill = skillWord
		if skill > -1 {
			skill += bonus * 6
		}
	}
	if tier < 0 {
		tier = 0
	}
	tbl := int32(uint32(ticRead(0x6504e8+4*uint32(tier), 4)[0]) | uint32(ticRead(0x6504e8+4*uint32(tier)+1, 1)[0])<<8 |
		uint32(ticRead(0x6504e8+4*uint32(tier)+2, 1)[0])<<16 | uint32(ticRead(0x6504e8+4*uint32(tier)+3, 1)[0])<<24)
	x := (skill - tbl) * 3
	if x < 0 {
		x++ // 0x58a113: cdq / sub eax,edx before the arithmetic shift
	}
	chance := x>>1 + (11-diff)*10
	if chance < 0 {
		chance = 0
	} else if chance > 99 {
		chance = 99
	}

	var cell [8]int64
	for i := 0; i < 8; i++ {
		if int32(r.R(100)) < chance {
			cell[i] = tic5e3730(trapType, uint32(i))
		} else {
			cell[i] = 2
		}
	}

	out := []int64{1, charPtr, ticSkill, 1, 0, ticPanel + ticRowBase + 32*c, 0}
	for k := int64(0); k < 64; k++ {
		if k/8 == c {
			out = append(out, cell[k%8])
		} else {
			out = append(out, ticRowDecoy+k)
		}
	}
	for k := int64(0); k < 8; k++ {
		if k == c {
			out = append(out, 1)
		} else {
			out = append(out, ticFlagDecoy+k)
		}
	}
	for i := uint32(0); i < 8; i++ {
		b8 := int64(0xffffffff)
		if cell[i] != 0 || tic5e3730(classRow, i) != 0 {
			b8 = cell[i]
		}
		out = append(out, b8)
	}
	for i := 0; i < 8; i++ {
		out = append(out, 1) // 0x4f4650: button +0xa0 = 1
	}
	return out
}
