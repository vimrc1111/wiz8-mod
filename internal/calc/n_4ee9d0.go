package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type armorExeTables struct {
	weights       [5]byte
	body, fatigue [5]uint32
}

var armorTables, armorTablesErr = loadArmorTables()

// The five body-location percentages live at exe 0x616308. Keep the
// player's table outside the source tree; loading precedes pure evaluation.
func loadArmorTables() (armorExeTables, error) {
	var tables armorExeTables
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return tables, err
	}
	o, err := peOffset(b, 0x616308)
	if err != nil {
		return tables, err
	}
	if len(b)-o < len(tables.weights) {
		return tables, fmt.Errorf("armor classes: truncated body weights")
	}
	copy(tables.weights[:], b[o:o+len(tables.weights)])
	// Dispatch tables select code-defined constants; neither table is embedded.
	for va, dst := range map[uint32]*[5]uint32{0x4eed68: &tables.body, 0x52a800: &tables.fatigue} {
		o, err := peOffset(b, va)
		if err != nil {
			return tables, err
		}
		if len(b)-o < 20 {
			return tables, fmt.Errorf("armor classes: truncated dispatch %#x", va)
		}
		for i := range dst {
			dst[i] = binary.LittleEndian.Uint32(b[o+4*i:])
		}
	}
	return tables, nil
}

// Inputs 0..278 retain characterPostureEligibilityNode's fixture layout.
// Its request/apply inputs are ignored: this caller checks 4 then 5 without
// applying either posture (0x4ee9f2, 0x4eea05). The last input is the arena
// character address, checked against the selected record, not a rule value.
// The fixture retains the child's empty monster lists and shared spell data.
func calcArmorClassesNode(in []int64, r Rand) []int64 {
	if len(in) != 305 {
		panic("armor classes: expected 305 inputs")
	}
	if in[0] < 0 || in[0] > 7 || in[304] != 0x20001000+6242*in[0] {
		panic("armor classes: character address does not match party slot")
	}
	if armorTablesErr != nil {
		panic(armorTablesErr)
	}
	if breathTablesErr != nil {
		panic(breathTablesErr)
	}
	child := append([]int64(nil), in[:279]...)
	// Item definitions are shared with the action checker; the later armor
	// definition fields are authoritative if they overlap its selected item.
	if child[14] >= 0 && child[14] < 4 {
		child[16] = in[296+2*child[14]]
	}
	child[262] = 0
	defending := false
	if in[6] != 0 {
		child[261] = 4
		defending = characterPostureEligibilityNode(child, r)[0] != 0
		if !defending {
			child[261] = 5
			defending = characterPostureEligibilityNode(child, r)[0] != 0
		}
	}
	// Equipment record stride 12, item stride 269: 0x4eea5a..0x4eea91.
	gear := [12]int64{in[286], in[287], in[288], in[289], in[290], in[291], in[61], in[62], in[292], in[293], in[294], in[295]}
	item := func(slot int) (int64, int32) {
		id := gear[slot]
		if id == -1 {
			return -1, 0
		}
		if id < 0 || id > 3 {
			panic("armor classes: item outside loaded definitions")
		}
		return in[296+2*id], int32(int8(in[297+2*id]))
	}
	var parts [13]int32 // C+0xe8d..0xebd, cleared at 0x4eea18.
	for _, slot := range []int{1, 2, 3, 6, 7} {
		kind, ac := item(slot)
		if kind == 5 {
			parts[3] += ac
		} else {
			parts[4] += ac
		}
	}
	if uint32(in[29+3*in[0]]) < 18 { // 0x4eea93: inactive statuses skip personal bonuses.
		feature := false
		if in[51] >= 0 && in[51] < 15 {
			for _, f := range breathTables.class[in[51]] {
				feature = feature || f == 22
			}
		}
		if in[52] >= 0 && in[52] < 16 {
			for _, f := range breathTables.race[in[52]] {
				feature = feature || f == 22
			}
		}
		if feature {
			parts[0] = 2
		} // Feature 0x16, exe 0x4eeaa0; abilities.md:39.
		speed := uint32(in[209])
		// Speed thresholds: 0x4eeab3..0x4eeadf.
		if speed >= 80 {
			parts[1]++
		}
		if speed >= 90 {
			parts[1]++
		}
		if speed < 20 {
			parts[1]--
		}
		if speed < 10 {
			parts[1]--
		}
		parts[2] = int32(uint32(in[222]) / 10) // Stealth, 0x4eeae5.
		if in[279] != 0 {
			parts[11] = int32(uint32(in[249])/20 + 1)
		} // Reflexion, 0x4eeb0b.
		if parts[3] > 0 {
			bonus := int32(uint32(in[217]) / 25)
			cap := parts[3]
			if defending {
				bonus = int32(uint32(in[217]) / 15)
				cap = (3 * cap) / 2
			}
			parts[3] += min(bonus, cap) // Shield skill cap, 0x4eeb28..0x4eeb6e.
		}
		parts[5], parts[6], parts[8] = int32(int8(in[282])), int32(int8(in[283])), int32(int8(in[285]))
		if defending {
			parts[10] = 2
		} // 0x4eeba2.
		switch uint32(in[280]) { // Encumbrance levels, 0x4eeba9..0x4eebd7.
		case 2:
			parts[7] = -1
		case 3:
			parts[7] = -2
		case 4:
			parts[7] = -4
		}
		fatigue := uint32(in[281])
		// Fatigue conversion immediates in 0x52a7e0..0x52a7fb; unknown codes pass through.
		if fatigue < 5 {
			switch armorTables.fatigue[fatigue] {
			case 0x52a7e0:
				fatigue = 0
			case 0x52a7e3:
				fatigue = 5
			case 0x52a7e9:
				fatigue = 10
			case 0x52a7ef:
				fatigue = 20
			case 0x52a7f5:
				fatigue = 40
			default:
				panic("armor classes: unknown fatigue dispatch")
			}
		}
		parts[9] = -int32(fatigue / 10)
	}
	var base int32
	for i := 0; i < 12; i++ {
		if i != 6 {
			base += parts[i]
		}
	} // 0x4eec25: body magic added separately.
	if in[284] != 0 && base > -5 {
		base = -5
	} // 0x4eec3b: vulnerable AC ceiling.
	var body [5]int32
	var weighted int32
	// Location-to-slot choices are code constants at 0x4eec8a..0x4eecb2.
	for i, target := range armorTables.body {
		var slot int
		switch target {
		case 0x4eec8a:
			slot = 0
		case 0x4eec94:
			slot = 4
		case 0x4eec9e:
			slot = 10
		case 0x4eeca8:
			slot = 5
		case 0x4eecb2:
			slot = 11
		default:
			panic("armor classes: unknown body dispatch")
		}
		_, ac := item(slot)
		body[i] = base + parts[6] + ac
		weighted += int32(armorTables.weights[i]) * body[i]
	}
	// Round signed percentage half away from zero, with the original i32
	// accumulator and addition overflow, 0x4eed23..0x4eed5c.
	if weighted < 0 {
		weighted = (weighted - 50) / 100
	} else {
		weighted = (weighted + 50) / 100
	}
	out := []int64{int64(base), int64(weighted)}
	for _, p := range parts {
		out = append(out, int64(p))
	}
	for _, ac := range body {
		out = append(out, int64(ac))
	}
	return out
}

func init() {
	names := append([]string(nil), Nodes["004ed2d0"].In...)
	names = append(names, "reflexEnabled", "encumbrance", "fatigue", "magicAC", "bodyMagicAC", "vulnerable", "effectAC",
		"equipment0", "equipment1", "equipment2", "equipment3", "equipment4", "equipment5", "equipment8", "equipment9", "equipment10", "equipment11",
		"armorType0", "armorAC0", "armorType1", "armorAC1", "armorType2", "armorAC2", "armorType3", "armorAC3", "characterAddress")
	Nodes["004ee9d0"] = Node{Addr: "004ee9d0", In: names, Out: []string{
		"baseAC", "weightedAC", "abilityAC", "speedAC", "stealthAC", "shieldAC", "accessoryAC", "magicAC", "bodyMagicAC", "encumbranceAC", "effectAC", "fatigueAC", "postureAC", "reflexAC", "reservedAC",
		"bodyAC0", "bodyAC1", "bodyAC2", "bodyAC3", "bodyAC4",
	}, F: calcArmorClassesNode}
}
