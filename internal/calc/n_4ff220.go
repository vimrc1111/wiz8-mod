package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var confusedRedirectDispatch, confusedRedirectDispatchErr = loadConfusedRedirectDispatch()

// The eight target-kind destinations are read from the user's executable,
// at 0x4ff388. peOffset is shared with exe.go; no table is embedded.
func loadConfusedRedirectDispatch() ([8]uint32, error) {
	var table [8]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return table, fmt.Errorf("confused redirect: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return table, err
	}
	off, err := peOffset(b, 0x4ff388)
	if err != nil {
		return table, err
	}
	if off > len(b)-32 {
		return table, fmt.Errorf("confused redirect: truncated dispatch")
	}
	for i := range table {
		table[i] = binary.LittleEndian.Uint32(b[off+4*i:])
	}
	return table, nil
}

// The first 58 inputs share the random-target interface. The next seven
// describe the location child, followed by three preserved target coordinates,
// two confusion durations, class/race, and fifteen class levels. Records are
// selected by sourceCharacterIndex/sourceMonsterID in the replay fixture; the
// slot of the other source kind may hold -1.
func confusedSpellRedirectChanceNode(in []int64, r Rand) []int64 {
	if len(in) != 87 {
		panic("confused redirect: expected 87 inputs")
	}
	for i, v := range in {
		if i == 58 || i == 70 || i == 71 {
			continue
		}
		// The unused source slot holds the -1 sentinel (0x53be00 / 0x53be50).
		if (i == 3 || i == 4) && v == -1 {
			continue
		}
		if v < 0 || v > 4294967295 {
			panic("confused redirect: field outside u32")
		}
		if i >= 10 && i < 18 || i >= 18 && i < 58 && ((i-18)%5 == 2 || (i-18)%5 == 3) {
			if v > 255 {
				panic("confused redirect: flag outside u8")
			}
		}
	}
	if in[0] > 149 || in[3] > 7 || in[4] > 7 || in[9] > 8 || in[58] < -1 || in[58] > 3 || in[70] < -1 || in[70] > 14 || in[71] < -1 || in[71] > 15 {
		panic("confused redirect: invalid record index, range, class or race")
	}
	if in[2] == 1 && in[3] < 0 || in[2] == 2 && in[4] < 0 {
		panic("confused redirect: source sentinel in the active source slot")
	}
	out := append([]int64(nil), in[5:9]...)
	out = append(out, in[65:68]...)
	switch in[2] {
	case 1:
		// Character confusion duration and self-target guard: 0x4ff28f..0x4ff2a9.
		if in[68] == 0 || in[5] == 1 && in[6] == in[3] {
			return out
		}
		if breathTablesErr != nil {
			panic(breathTablesErr)
		}
		hasFeature := false
		if in[70] != -1 {
			for _, f := range breathTables.class[in[70]] {
				hasFeature = hasFeature || f == 7
			}
		}
		if in[71] != -1 {
			for _, f := range breathTables.race[in[71]] {
				hasFeature = hasFeature || f == 7
			}
		}
		// Ability 7, maximum 50: 0x4ff2af..0x4ff2c6. The low-level
		// curve uses executable f32 operands loaded by statusRebuildTables.
		if hasFeature {
			if in[70] == -1 {
				panic("confused redirect: ability scaling requires a current class")
			}
			level := in[72+in[70]]
			threshold := int64(50)
			if level <= 20 {
				if statusRebuildTablesErr != nil {
					panic(statusRebuildTablesErr)
				}
				t := statusRebuildTables
				threshold = int64((2*float64(level) + t.featureBase) * 50 * t.featureScale)
			}
			if uint32(r.R(100)) < uint32(threshold) {
				return out
			}
		}
	case 2:
		// The resolved monster must exist, even when confusion is zero: 0x4e5840.
		found := false
		for i := 0; i < int(in[9]); i++ {
			found = found || in[18+5*i] == in[4]
		}
		if !found {
			panic("confused redirect: source monster absent from ID list")
		}
		if in[69] == 0 || in[5] == 3 && in[7] == in[4] {
			return out
		}
	default:
		return out
	}
	// Preserve the 20 unchanged outcomes of R(100): 0x4ff2e4..0x4ff2f1.
	if r.R(100) < 20 {
		return out
	}
	if in[1] < 1 || in[1] > 8 {
		return out
	}
	if confusedRedirectDispatchErr != nil {
		panic(confusedRedirectDispatchErr)
	}
	switch confusedRedirectDispatch[in[1]-1] {
	case 0x4ff313:
		child := append([]int64(nil), in[:58]...)
		for _, i := range []int{3, 4} {
			child[i] = int64(uint32(child[i])) // 0x4fedc0 reads the sentinel as u32.
		}
		selected := confusedSpellRandomTargetNode(child, r)
		if selected[0] == 0 {
			return out
		}
		copy(out[:4], selected[1:])
	case 0x4ff324:
		loc := []int64{in[0], in[58], in[2]}
		loc = append(loc, in[59:65]...)
		copy(out[4:], confusedSpellRandomLocationNode(loc, r))
	case 0x4ff381:
		return out
	default:
		panic("confused redirect: unknown executable dispatch destination")
	}
	// 0x590950/0x590b40 report the confused caster. Their UI/log side
	// effects do not modify the seven target words returned by this interface.
	return out
}

func init() {
	inputs := []string{"spellIndex", "spellTargetKind", "sourceKind", "sourceCharacterIndex", "sourceMonsterID", "targetKind", "targetCharacterIndex", "targetMonsterID", "targetGroup", "monsterCount"}
	for i := 0; i < 8; i++ {
		inputs = append(inputs, fmt.Sprintf("partyPresent%d", i))
	}
	for i := 0; i < 8; i++ {
		for _, suffix := range []string{"ID", "Group", "Present", "Active", "HP"} {
			inputs = append(inputs, fmt.Sprintf("monster%d%s", i, suffix))
		}
	}
	inputs = append(inputs, "spellRange", "partyXBits", "partyYBits", "partyZBits", "monsterXBits", "monsterYBits", "monsterZBits", "targetXBits", "targetYBits", "targetZBits", "characterConfusion", "monsterConfusion", "currentClass", "race")
	for i := 0; i < 15; i++ {
		inputs = append(inputs, fmt.Sprintf("classLevel%d", i))
	}
	Nodes["004ff220"] = Node{Addr: "004ff220", In: inputs, Out: []string{"targetKind", "targetCharacterIndex", "targetMonsterID", "targetGroup", "targetXBits", "targetYBits", "targetZBits"}, F: confusedSpellRedirectChanceNode}
}
