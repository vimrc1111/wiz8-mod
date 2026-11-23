package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

// Interface order follows re/emu/cases/0050e650.json. Array lengths are
// from 0x517ec0 (eight slots), 0x50eac0 (20 states), 0x50ecc0 (eight buffs),
// 0x50e650 (103-byte buffers), and 0x5539e0 / 0x553a60 (7 / 41 rules).
func init() {
	inputs := []string{"partySlot", "inParty"}
	appendArray := func(dst *[]string, name string, count int) {
		for i := 0; i < count; i++ {
			*dst = append(*dst, fmt.Sprintf("%s[%d]", name, i))
		}
	}
	appendArray(&inputs, "npcRegionPenaltyEnabled", 8)
	inputs = append(inputs, "class", "race", "gender", "level")
	appendArray(&inputs, "classLevel", 15)
	attributes := []string{"Strength", "Intelligence", "Piety", "Vitality", "Dexterity", "Speed", "Senses"}
	for _, name := range attributes {
		inputs = append(inputs, "raw"+name)
	}
	appendArray(&inputs, "rawSkill", 41)
	appendArray(&inputs, "statusDuration", 20)
	inputs = append(inputs, "poisonStrength")
	for i := 0; i < 8; i++ {
		for _, name := range []string{"timedBuffStrength", "timedBuffMultiplier", "timedBuffDuration"} {
			inputs = append(inputs, fmt.Sprintf("%s[%d]", name, i))
		}
	}
	inputs = append(inputs, "controllerActive", "controllerRegion", "currentRegion", "controllerLevel")
	appendArray(&inputs, "equipmentRuleModifiers", 103)
	appendArray(&inputs, "partyRuleModifiers", 103)
	for i := 0; i < 8; i++ {
		inputs = append(inputs, fmt.Sprintf("backpackItemWeight[%d]", i), fmt.Sprintf("backpackItemCount[%d]", i))
	}
	var outputs []string
	appendArray(&outputs, "statusRuleModifiers", 103)
	appendArray(&outputs, "totalRuleModifiers", 103)
	for _, name := range attributes {
		outputs = append(outputs, "effective"+name)
	}
	appendArray(&outputs, "effectiveSkill", 41)
	outputs = append(outputs, "loadCapacity", "carriedWeight", "loadCategory")
	Nodes["0050e650"] = Node{Addr: "0050e650", In: inputs, Out: outputs, F: statusEffectRebuildNode}
}

// The interface returns both complete modifier buffers and the rebuilt effective
// attributes, skills and carrying rules. Other caches rebuilt by 0x4ed9d0 are
// outside this output interface. All variable inputs to these outputs are explicit.
// Layout: 0x50e650 uses character stride 0x1862 and 103-byte buffers;
// 0x5539e0 / 0x553a60 rebuild seven attributes / 41 skills;
// 0x4ed9d0 / 0x4edc60 rebuild capacity, weight and encumbrance.
var statusRebuildTables, statusRebuildTablesErr = loadStatusRebuildTables()

type statusRebuildExe struct {
	character                 *ExeTables
	raceFeatures              [16][5]int32
	dispatch                  [17]uint32
	featureBase, featureScale float64
}

func loadStatusRebuildTables() (statusRebuildExe, error) {
	var t statusRebuildExe
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("status rebuild: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	var err error
	t.character, err = LoadExe(path)
	if err != nil {
		return t, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	read := func(va uint32, count int) ([]byte, error) {
		offset, err := peOffset(b, va)
		if err != nil {
			return nil, err
		}
		if offset < 0 || count > len(b)-offset {
			return nil, fmt.Errorf("status rebuild: truncated table %#x", va)
		}
		return b[offset : offset+count], nil
	}
	// 0x547940: race feature rows, five entries per race at 0x615130.
	for i := range t.raceFeatures {
		row, err := read(0x615130+uint32(i*20), 20)
		if err != nil {
			return t, err
		}
		for j := range t.raceFeatures[i] {
			t.raceFeatures[i][j] = int32(binary.LittleEndian.Uint32(row[j*4:]))
		}
	}
	// 0x50eaf4: status 3..19 dispatch; load the user's table without copying it.
	row, err := read(0x50ec74, 68)
	if err != nil {
		return t, err
	}
	for i := range t.dispatch {
		t.dispatch[i] = binary.LittleEndian.Uint32(row[i*4:])
		switch t.dispatch[i] {
		case 0x50eafb, 0x50eb11, 0x50eb4a, 0x50eb60, 0x50eb76, 0x50eb7f,
			0x50eba1, 0x50ebe1, 0x50ebe5, 0x50ebea, 0x50ebf0, 0x50ec58:
		default:
			return t, fmt.Errorf("status rebuild: unknown status destination %#x", t.dispatch[i])
		}
	}
	// 0x5479b0 uses float constants at 0x5ee800 and 0x5ed490.
	for i, va := range []uint32{0x5ee800, 0x5ed490} {
		row, err := read(va, 4)
		if err != nil {
			return t, err
		}
		value := float64(math.Float32frombits(binary.LittleEndian.Uint32(row)))
		if i == 0 {
			t.featureBase = value
		} else {
			t.featureScale = value
		}
	}
	return t, nil
}

func statusEffectRebuildNode(in []int64, r Rand) []int64 {
	if len(in) != 348 {
		panic("status rebuild: expected 348 interface fields")
	}
	if statusRebuildTablesErr != nil {
		panic(statusRebuildTablesErr)
	}
	t := statusRebuildTables
	for _, v := range in {
		if v < 0 || v > 0xffffffff {
			panic("status rebuild: fields must fit u32")
		}
	}
	if in[0] > 7 || in[1] == 0 || in[1] > 255 || in[10] > 14 || in[11] > 15 || in[12] > 1 {
		panic("status rebuild: invalid party slot, membership, class, race or gender")
	}
	for _, span := range [][2]int{{2, 10}, {122, 123}, {125, 126}, {126, 332}} {
		for _, v := range in[span[0]:span[1]] {
			if v > 255 {
				panic("status rebuild: byte field overflow")
			}
		}
	}
	for i := 0; i < 8; i++ {
		if in[98+3*i] > 255 || in[99+3*i] > 65535 {
			panic("status rebuild: invalid timed buff")
		}
		if in[332+2*i] > 65535 || in[333+2*i] > 255 {
			panic("status rebuild: invalid item weight/count")
		}
	}
	var status, total [103]uint8
	add := func(offset, value int) { status[offset] += uint8(value) }
	feat := func(code int) bool {
		for _, v := range t.character.ClassFeat[in[10]] {
			if v == code {
				return true
			}
		}
		for _, v := range t.raceFeatures[in[11]] {
			if int(v) == code {
				return true
			}
		}
		return code == 28 && in[100+3] > 0 // 0x547994: second timed buff duration, C+0xa79.
	}
	// 0x50eadb: each nonzero u32 duration activates its status once.
	for i := 3; i < 20; i++ {
		if in[77+i] == 0 {
			continue
		}
		switch t.dispatch[i-3] {
		case 0x50eafb:
			add(1, -2)
			add(0x4b, -2)
		case 0x50eb11:
			add(1, -5)
			for j := 0xc; j < 0x3c; j++ {
				add(j, -20)
			}
		case 0x50eb4a:
			add(1, -5)
			add(0x4b, -4)
		case 0x50eb60:
			add(1, -3)
			add(0x4b, -2)
		case 0x50eb76:
			add(0x11, -50)
		case 0x50eb7f:
			add(1, -2)
			add(0x4b, -2)
			add(8, int(in[97]))
		case 0x50eba1:
			if feat(7) {
				level := in[14+int(in[10])] // Status generation precedes 0x4eda15 level clamping.
				value := 50
				if level <= 20 {
					value = int((float64(2*level) + t.featureBase) * 50 * t.featureScale)
				}
				add(0x12, value-50)
			} else {
				add(0x12, -50)
				status[0x45] = 1
			}
		case 0x50ebe1:
			add(0x10, -50)
			status[0x45] = 1
		case 0x50ebe5:
			status[0x45] = 1
		case 0x50ebea:
			add(0xd, -50)
		case 0x50ebf0:
			if in[122] != 0 {
				value := -5
				if in[123] == in[124] {
					value = -1 - int(uint8(in[125])>>1)
				}
				add(9, value)
			}
		}
	}
	// 0x50ecc0: only buff records 5,6,7 contribute here. The other five
	// still belong to the input record and may supply character features.
	scale := func(value uint8, multiplier int64) uint8 {
		return uint8((uint32(value)*uint32(multiplier+100) + 50) / 100)
	}
	for i := 5; i < 8; i++ {
		if in[100+3*i] == 0 {
			continue
		}
		power, multiplier := uint8(in[98+3*i]), in[99+3*i]
		switch i {
		case 5:
			add(0x11, int(scale(power*10, multiplier)))
		case 6:
			amount := scale(power*5, multiplier)
			for j := 0xc; j < 0x13; j++ {
				add(j, int(amount))
			}
		case 7:
			add(6, int(scale(power*8, multiplier)))
			add(5, int(scale(power, multiplier)))
		}
	}
	// Reuse the existing exact NPC region formula, including relocated slot lookup.
	npc := make([]int64, 72)
	npc[0] = 0x20000000 + in[0]*0x1862
	npc[1] = 0x20000000
	npc[2] = in[1]
	copy(npc[3:11], in[2:10])
	for i := 0; i < 61; i++ {
		npc[11+i] = int64(status[i])
	}
	for i, v := range npcRegionStatPenaltyNode(npc, r) {
		status[i] = uint8(v)
	}
	// 0x50f090: additive fields wrap; six flags form a normalized union;
	// three strength fields select the unsigned maximum. +0x4c..+0x66 stay zero.
	merge := func(src []int64) {
		for i := 0; i < 0x4c; i++ {
			switch i {
			case 0x42, 0x43, 0x44, 0x45, 0x46, 0x4a:
				if src[i] != 0 {
					total[i] = 1
				}
			case 0x47, 0x48, 0x49:
				total[i] = max(total[i], uint8(src[i]))
			default:
				total[i] += uint8(src[i])
			}
		}
	}
	merge(in[126:229])
	statusValues := make([]int64, 103)
	for i, v := range status {
		statusValues[i] = int64(v)
	}
	merge(statusValues)
	merge(in[229:332])
	out := append([]int64(nil), statusValues...)
	for _, v := range total {
		out = append(out, int64(v))
	}
	var attrs [7]int64
	// 0x5539ef: signed byte correction, signed comparison, clamp to 1..125.
	for i := range attrs {
		attrs[i] = int64(min(125, max(1, int32(uint32(in[29+i])+uint32(int32(int8(total[0xc+i])))))))
		out = append(out, attrs[i])
	}
	// 0x553a6f: primary skill bonus is unsigned floor(25*raw/100), minimum one.
	for i := 0; i < 41; i++ {
		value := uint32(in[36+i])
		if i == t.character.MainSkill[in[10]] {
			value += max(uint32(1), value*25/100)
		}
		effective := min(int32(125), max(int32(0), int32(value+uint32(int32(int8(total[0x13+i]))))))
		out = append(out, int64(effective))
	}
	// 0x4edb03: carrying capacity 12*(2*STR+VIT), feature 24 multiplies by 2/3.
	capacity := 12 * (2*attrs[0] + attrs[3])
	if feat(24) {
		capacity = capacity * 2 / 3
	}
	var weight uint32
	for i := 0; i < 8; i++ {
		weight += uint32(in[332+2*i]) * uint32(max(int64(1), in[333+2*i]))
	}
	// 0x4edb6e: u32 multiplication and unsigned integer quotient.
	pct := uint32(weight*100) / uint32(capacity)
	category := int64(4)
	switch {
	case pct < 50:
		category = 0
	case pct < 70:
		category = 1
	case pct < 85:
		category = 2
	case pct <= 100:
		category = 3
	}
	return append(out, capacity, int64(weight), category)
}
