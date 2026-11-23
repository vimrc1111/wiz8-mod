package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Feature 28 and the stamina divisor 5: 0x4ebc9e, 0x4ebcb5..0x4ebcc8;
// feature table dimensions: 0x547955..0x54798d (abilities.md:11).
const breathFeatureCode = 28
const breathStaminaDivisor = 5

type breathFeatureTables struct {
	class [15][3]int32
	race  [16][5]int32
}

var breathTables, breathTablesErr = loadBreathFeatureTables()

func loadBreathFeatureTables() (breathFeatureTables, error) {
	var tables breathFeatureTables
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return tables, fmt.Errorf("breath eligibility: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return tables, err
	}
	read := func(va uint32, dst []int32) error {
		offset, err := peOffset(b, va)
		if err != nil {
			return err
		}
		if 4*len(dst) > len(b)-offset {
			return fmt.Errorf("breath eligibility: table %#x truncated", va)
		}
		for i := range dst {
			dst[i] = int32(binary.LittleEndian.Uint32(b[offset+4*i:]))
		}
		return nil
	}
	// Original feature tables at 0x61507c and 0x615130; no exe data embedded.
	for i := range tables.class {
		if err := read(0x61507c+12*uint32(i), tables.class[i][:]); err != nil {
			return tables, err
		}
	}
	for i := range tables.race {
		if err := read(0x615130+20*uint32(i), tables.race[i][:]); err != nil {
			return tables, err
		}
	}
	return tables, nil
}

// Pure after table loading. characterIndex selects one of eight arena records
// (stride 6242: 0x4ebc84..0x4ebc99, growth.md:67); it does not change the rule.
// Output is AL, the boolean consumed by 0x501860, not unspecified upper EAX.
func breathStaminaEligibilityNode(in []int64, r Rand) []int64 {
	if len(in) != 6 || in[0] < 0 || in[0] > 7 || in[1] < -1 || in[1] > 14 || in[2] < -1 || in[2] > 15 || in[3] < 0 || in[3] > 4294967295 || in[4] < 0 || in[4] > 4294967295 || in[5] < -2147483648 || in[5] > 2147483647 {
		panic("breath eligibility: expected slot 0..7, class -1..14, race -1..15, u32 effect/maxStamina, i32 currentStamina")
	}
	if breathTablesErr != nil {
		panic(breathTablesErr)
	}
	hasFeature := in[3] != 0 // C+0xa79, unsigned nonzero: 0x547994..0x54799c.
	if in[1] != -1 {
		for _, feature := range breathTables.class[in[1]] {
			hasFeature = hasFeature || feature == breathFeatureCode
		}
	}
	if in[2] != -1 {
		for _, feature := range breathTables.race[in[2]] {
			hasFeature = hasFeature || feature == breathFeatureCode
		}
	}
	// u32 maximum is divided before the signed comparison (abilities.md:45).
	if hasFeature && in[5] >= in[4]/breathStaminaDivisor {
		return []int64{1}
	}
	return []int64{0}
}

func init() {
	Nodes["004ebc80"] = Node{
		Addr: "004ebc80",
		In:   []string{"characterIndex", "currentClass", "race", "breathEffect", "maxStamina", "currentStamina"},
		Out:  []string{"breathEligible"}, F: breathStaminaEligibilityNode,
	}
}
