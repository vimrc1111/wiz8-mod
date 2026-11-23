package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var monsterNormalizeDispatch, monsterNormalizeDispatchErr = loadMonsterNormalizeDispatch()

// Dispatch tables are loaded once from the user's executable; F is pure.
func loadMonsterNormalizeDispatch() ([3][9]uint32, error) {
	var tables [3][9]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return tables, fmt.Errorf("monster range: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return tables, err
	}
	// Attack animation, missing-animation fallback, and composed range getter.
	for k, va := range []uint32{0x540128, 0x54014c, 0x51a7d8} {
		o, err := peOffset(b, va)
		if err != nil {
			return tables, err
		}
		if o > len(b)-36 {
			return tables, fmt.Errorf("monster range: truncated dispatch %#x", va)
		}
		for i := range tables[k] {
			tables[k][i] = binary.LittleEndian.Uint32(b[o+4*i:])
		}
	}
	return tables, nil
}

// Fixture uses loaded templates, original scene vtable and an idle previous
// animation (255). Selected records are indexed, not duplicated across slots.
func monsterAttackRangeNormalizeNode(in []int64, r Rand) []int64 {
	if len(in) != 22 || in[0] < 0 || in[0] > 8 || in[1] < 0 || in[1] > 2 || in[7] < 0 || in[7] > 3 {
		panic("monster range: invalid attack kind or record index")
	}
	if monsterNormalizeDispatchErr != nil {
		panic(monsterNormalizeDispatchErr)
	}
	var animation int64
	// Animation constants are the immediate values at these original targets.
	switch monsterNormalizeDispatch[0][in[0]] {
	case 0x540019:
		animation = 9
	case 0x54000d:
		animation = 10
	case 0x540021:
		animation = 11
	case 0x54002d:
		animation = 12
	case 0x540015:
		animation = 13
	case 0x540025:
		animation = 14
	case 0x54001d:
		animation = 15
	case 0x540029:
		animation = 16
	case 0x540011:
		animation = 17
	default:
		panic("monster range: unknown animation dispatch")
	}
	if uint32(in[3]) == 0 { // 0x4c3770: selected animation variant count.
		switch monsterNormalizeDispatch[1][animation-9] {
		case 0x54005a:
			animation = 6 // generic melee animation, 0x54005a.
		case 0x540061:
			animation = 7 // generic ranged animation, 0x540061.
		default:
			panic("monster range: unknown fallback dispatch")
		}
	}
	rangeAfter := int64(uint8(in[2]))
	if animation == 17 || animation == 13 || animation == 7 {
		if rangeAfter <= 1 {
			rangeAfter = 3 // 0x5400ab: normalize ranged attacks.
		}
	} else if rangeAfter >= 2 {
		rangeAfter = 0 // 0x540093: normalize melee attacks.
	}
	actualRange := int64(-1) // 0x51a7d4: unsupported attack mode.
	mode := uint32(in[4])
	if mode <= 8 {
		switch monsterNormalizeDispatch[2][mode] {
		case 0x51a74a:
			actualRange = rangeAfter
		case 0x51a7aa:
			actualRange = int64(int32(in[6])) // spell definition +0x12f.
		case 0x51a7cb:
			actualRange = 2
		case 0x51a7d1:
			actualRange = 0
		case 0x51a7d4:
		default:
			panic("monster range: unknown range dispatch")
		}
	}
	state := int64(uint8(in[5]))
	if actualRange >= 2 {
		state = 0 // 0x5400cf: clear ranged-state byte.
	}
	// 0x4e4ead..0x4e4ed4 starts animation with policy 1, clears frame;
	// scene flag bit 5 prevents only the selected-animation byte assignment.
	selected := animation
	if uint32(in[8])&32 != 0 {
		selected = 255
	}
	// 0x4b6749..0x4b675f records model access and advances the u32 counter.
	counter := uint32(in[12])
	out := []int64{rangeAfter, state, selected, 0, 0, 1, 1,
		int64(counter + 1), int64(counter), int64(uint32(in[11]))}
	// 0x536170 clears the eight-word target record; words 1..3 become -1.
	return append(out, 0, 0xffffffff, 0xffffffff, 0xffffffff, 0, 0, 0, 0)
}

func init() {
	Nodes["0053ffe0"] = Node{
		Addr: "0053ffe0",
		In:   []string{"attackKind", "attackSlot", "range", "animationVariants", "attackMode", "rangedState", "spellRange", "spellIndex", "sceneFlags", "animationFrame", "animationStart", "animationTimeBits", "animationCounter", "targetWord0", "targetWord1", "targetWord2", "targetWord3", "targetWord4", "targetWord5", "targetWord6", "targetWord7", "combatActive"},
		Out:  []string{"rangeAfter", "rangedStateAfter", "animationAfter", "animationFrameAfter", "animationStartAfter", "animationPending", "animationPolicy", "animationCounterAfter", "animationAccessCounter", "animationTimeAfter", "targetWordAfter0", "targetWordAfter1", "targetWordAfter2", "targetWordAfter3", "targetWordAfter4", "targetWordAfter5", "targetWordAfter6", "targetWordAfter7"},
		F:    monsterAttackRangeNormalizeNode,
	}
}
