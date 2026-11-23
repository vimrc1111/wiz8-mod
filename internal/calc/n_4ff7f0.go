package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type schoolSelectionTables struct {
	classAdjustment [15]int32
	classMask       [15]byte
	threshold       [17]uint32
	required        [6]int32
}

var schoolSelectionData, schoolSelectionErr = loadSchoolSelectionTables()

// Tables are read at runtime from the player's executable via exe.go's PE mapper.
func loadSchoolSelectionTables() (schoolSelectionTables, error) {
	var t schoolSelectionTables
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("school selection: set WIZ8_EXE")
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
			return 0, fmt.Errorf("school selection: truncated table %#x", va)
		}
		return binary.LittleEndian.Uint32(b[off:]), nil
	}
	for i := range t.classAdjustment {
		v, e := read(0x616310 + uint32(4*i))
		if e != nil {
			return t, e
		}
		t.classAdjustment[i] = int32(v)
		off, e := peOffset(b, 0x61634c+uint32(i))
		if e != nil {
			return t, e
		}
		t.classMask[i] = b[off]
	}
	for i := range t.threshold {
		v, e := read(0x616df0 + uint32(4*i))
		if e != nil {
			return t, e
		}
		t.threshold[i] = v
	}
	// 0x4acbf8 dispatches levels 2..7 to literal minimum-level returns.
	// Read destinations and their immediate operands, rather than embed an exe table.
	for i := range t.required {
		va, e := read(0x4acbf8 + uint32(4*i))
		if e != nil {
			return t, e
		}
		off, e := peOffset(b, va)
		if e != nil {
			return t, e
		}
		if off > len(b)-6 || b[off] != 0xb8 || b[off+5] != 0xc3 {
			return t, fmt.Errorf("school selection: unknown level return %#x", va)
		}
		t.required[i] = int32(binary.LittleEndian.Uint32(b[off+1:]))
	}
	return t, nil
}

// Interface follows 0x4ff7f0..0x4ffbcc. Effective skills are signed;
// weighted skills, thresholds and actionProgress scaling preserve uint32 wrapping.
// Character slot zero is fixture wiring; party lookup 0x517ec0 reads presence
// and resolves that same character before selecting its combat record.
func spellSchoolSkillSelectionNode(in []int64, _ Rand) []int64 {
	if len(in) != 50 {
		panic("school selection: expected 50 inputs")
	}
	for i, v := range in {
		lo, hi := int64(0), int64(4294967295)
		if i == 9 || i == 10 || (i >= 17 && i < 29 && (i-17)%3 == 1) {
			lo, hi = -2147483648, 2147483647
		}
		if i == 5 || i == 6 || i == 7 || i == 8 || i == 13 || i == 16 || (i >= 17 && i < 29 && (i-17)%3 == 0) {
			hi = 255
		}
		if v < lo || v > hi {
			panic("school selection: field exceeds storage width")
		}
	}
	if in[0] > 149 || in[11] > 5 || in[12] > 14 || in[14] > 2 || in[16] == 0 {
		panic("school selection: invalid spell, realm, profession, difficulty or presence")
	}
	// 0x4ff810 and 0x4ff8c2 inspect only the low byte of these arguments.
	casting, training := byte(in[1]) != 0, byte(in[2]) != 0
	if casting && in[4] != 0 && in[5] != 0 {
		return []int64{26}
	} // 0x4ff83f.
	mask := byte(0)
	if in[6] != 0 {
		mask |= 1
	}
	if in[7] != 0 {
		mask |= 2
	}
	if in[5] != 0 {
		mask |= 4
	}
	if in[8] != 0 {
		mask |= 8
	}
	highest, trainable := -1, -1
	highestValue, trainableValue := int32(-1), int32(-1)
	for i := 0; i < 4; i++ {
		if mask&(1<<i) == 0 {
			continue
		}
		skill := int32(in[18+3*i])
		if skill > highestValue {
			highest, highestValue = 24+i, skill
		}
		if training && in[17+3*i] != 0 && skill > trainableValue {
			trainable, trainableValue = 24+i, skill
		}
	}
	if highest == -1 {
		panic("school selection: no usable school")
	}
	if !training || trainable == -1 || highest == trainable {
		return []int64{int64(highest)}
	}
	if !casting {
		return []int64{int64(trainable)}
	}
	if schoolSelectionErr != nil {
		panic(schoolSelectionErr)
	}
	t := schoolSelectionData
	index := int32(in[9])/2 + int32(in[10])
	if index > 16 {
		index = 16
	} // 0x4ff960..0x4ff976.
	if index < 0 {
		panic("school selection: negative difficulty table index")
	}
	weighted := (uint32(in[19+3*(trainable-24)]) + 4*uint32(in[29+in[11]])) / 5
	threshold := t.threshold[index] * uint32(in[3]) / 7 // 0x4ff982..0x4ff994.
	risk := uint32(0)
	if weighted < threshold {
		risk = (70*threshold - 70*weighted) / threshold // 0x4ff9a5..0x4ff9c5.
		if int32(risk) < 0 {
			risk = 0
		} else if int32(risk) > 100 {
			risk = 100
		}
	}
	class := int(in[12])
	level := int32(-1)
	if t.classAdjustment[class] != -255 {
		level = int32(uint32(in[35+class]) + uint32(t.classAdjustment[class]))
	}
	for i := 0; i < 15; i++ {
		if in[35+i] == 0 || i == class || t.classMask[i]&mask == 0 || t.classAdjustment[i] == -255 {
			continue
		}
		effective := int32(uint32(in[35+i]) + uint32(t.classAdjustment[i]))
		if effective > 0 {
			level += effective
		}
	}
	required := int32(1) // 0x4acbef: default, including spell level 0,1 and >7.
	if in[10] >= 2 && in[10] <= 7 {
		required = t.required[in[10]-2]
	}
	deficit := int32(uint32(required) - uint32(level) - 1 + uint32(in[3]))
	if deficit > 0 {
		risk += uint32(in[10]) * uint32(deficit)
	} // 0x4ffadb..0x4ffae8.
	// 0x4ffaf2..0x4ffb7b shares the exact wrapping progress formula
	// with 0x501910. Reuse its verified pure adapter, including the
	// difficulty thresholds and unsigned division.
	risk = uint32(combatSpellRiskPhaseScalingNode([]int64{0, in[13], in[14], in[15], int64(risk)}, nil)[0])
	if risk == 0 {
		highest = trainable
	}
	return []int64{int64(highest)}
}

func init() {
	names := []string{"spellIndex", "casting", "training", "powerLevel", "verbalCastingBlocked", "alchemyUsable", "wizardryUsable", "divinityUsable", "psionicUsable", "spellCost", "spellLevel", "realm", "currentClass", "battle", "difficulty", "actionProgress", "characterPresent"}
	for i := 0; i < 4; i++ {
		for _, s := range []string{"Enabled", "Effective", "Base"} {
			names = append(names, fmt.Sprintf("school%d%s", i, s))
		}
	}
	for i := 0; i < 6; i++ {
		names = append(names, fmt.Sprintf("realm%dEffective", i))
	}
	for i := 0; i < 15; i++ {
		names = append(names, fmt.Sprintf("classLevel%d", i))
	}
	Nodes["004ff7f0"] = Node{Addr: "004ff7f0", In: names, Out: []string{"skillIndex"}, F: spellSchoolSkillSelectionNode}
}
