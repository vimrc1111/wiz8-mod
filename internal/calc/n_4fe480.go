package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type autoPowerRules struct {
	category       [10]uint32
	spell          [95]uint32
	skillThreshold [17]uint32
	classOffset    [15]int32
	classMask      [15]byte
}

var autoPowerTables, autoPowerError = loadAutoPowerRules()

// Immutable dispatch and skill tables come from the user's executable, through
// exe.go's PE address mapping. The formula below has no IO or mutable state.
func loadAutoPowerRules() (autoPowerRules, error) {
	var t autoPowerRules
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("auto power: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	read := func(va uint32, n int) ([]byte, error) {
		o, e := peOffset(b, va)
		if e != nil {
			return nil, e
		}
		if len(b)-o < n {
			return nil, fmt.Errorf("auto power: truncated table %#x", va)
		}
		return b[o : o+n], nil
	}
	dispatch := func(index, jump uint32, dst []uint32) error {
		idx, e := read(index, len(dst))
		if e != nil {
			return e
		}
		for i, k := range idx {
			v, e := read(jump+4*uint32(k), 4)
			if e != nil {
				return e
			}
			dst[i] = binary.LittleEndian.Uint32(v)
		}
		return nil
	}
	if err = dispatch(0x4fe6ac, 0x4fe698, t.category[:]); err != nil {
		return t, err
	}
	if err = dispatch(0x4fe6dc, 0x4fe6b8, t.spell[:]); err != nil {
		return t, err
	}
	for i := range t.skillThreshold {
		v, e := read(0x616df0+uint32(i*4), 4)
		if e != nil {
			return t, e
		}
		t.skillThreshold[i] = binary.LittleEndian.Uint32(v)
	}
	for i := range t.classOffset {
		v, e := read(0x616310+uint32(i*4), 4)
		if e != nil {
			return t, e
		}
		t.classOffset[i] = int32(binary.LittleEndian.Uint32(v))
	}
	v, e := read(0x61634c, 15)
	if e != nil {
		return t, e
	}
	copy(t.classMask[:], v)
	return t, nil
}

// Field order is the case input order; arrays are caster skills, target statuses
// and the twelve equipment/definition records used by the real children.
var spellAutoPowerInputs = []string{"casterIndex", "spellID", "bonusPercent", "targetType", "targetIndex", "spellCategory", "spellRealm", "spellCost", "spellLevel", "mageUsable", "priestUsable", "alchemistUsable", "psionicUsable", "effectBase", "effectDice", "effectSides", "casterClass", "classLevel0", "classLevel1", "classLevel2", "classLevel3", "classLevel4", "classLevel5", "classLevel6", "classLevel7", "classLevel8", "classLevel9", "classLevel10", "classLevel11", "classLevel12", "classLevel13", "classLevel14", "casterNausea", "magicLearned0", "magicSkill0", "magicLearned1", "magicSkill1", "magicLearned2", "magicSkill2", "magicLearned3", "magicSkill3", "effectiveMagicSkill0", "effectiveMagicSkill1", "effectiveMagicSkill2", "effectiveMagicSkill3", "realmSkill0", "spellPoints0", "realmSkill1", "spellPoints1", "realmSkill2", "spellPoints2", "realmSkill3", "spellPoints3", "realmSkill4", "spellPoints4", "realmSkill5", "spellPoints5", "combatActive", "difficulty", "actionProgress", "maxHP", "currentHP", "maxStamina", "currentStamina", "characterStatus2", "monsterStatus2", "characterStatus4", "monsterStatus4", "characterStatus6", "monsterStatus6", "characterStatus7", "monsterStatus7", "characterStatus9", "monsterStatus9", "characterStatus11", "monsterStatus11", "characterStatus12", "monsterStatus12", "characterStatus13", "monsterStatus13", "characterStatus15", "monsterStatus15", "characterStatus16", "monsterStatus16", "identifyItem", "equipmentItem0", "curseRemoved0", "equipmentItem1", "curseRemoved1", "equipmentItem2", "curseRemoved2", "equipmentItem3", "curseRemoved3", "equipmentItem4", "curseRemoved4", "equipmentItem5", "curseRemoved5", "equipmentItem6", "curseRemoved6", "equipmentItem7", "curseRemoved7", "equipmentItem8", "curseRemoved8", "equipmentItem9", "curseRemoved9", "equipmentItem10", "curseRemoved10", "equipmentItem11", "curseRemoved11", "itemCursed0", "itemDifficulty0", "itemCursed1", "itemDifficulty1", "itemCursed2", "itemDifficulty2", "itemCursed3", "itemDifficulty3", "itemCursed4", "itemDifficulty4", "itemCursed5", "itemDifficulty5", "itemCursed6", "itemDifficulty6", "itemCursed7", "itemDifficulty7", "itemCursed8", "itemDifficulty8", "itemCursed9", "itemDifficulty9", "itemCursed10", "itemDifficulty10", "itemCursed11", "itemDifficulty11"}

func init() {
	Nodes["004fe480"] = Node{Addr: "004fe480", In: spellAutoPowerInputs, Out: []string{"autoPowerLevel"}, F: spellAutoPowerLevelNode}
}

// Inputs are loaded spell, caster, selected target and item records. Distinct
// arena slots keep caster/target fields independent in the replay fixtures.
// Character stride 0x1862, action stride 0x106: 0x4fe480..0x4fe501.
func spellAutoPowerLevelNode(in []int64, r Rand) []int64 {
	names := spellAutoPowerInputs
	if len(in) != len(names) {
		panic("auto power: input arity")
	}
	if autoPowerError != nil {
		panic(autoPowerError)
	}
	v := make(map[string]int64, len(in))
	for i, n := range names {
		v[n] = in[i]
	}
	u := func(n string) uint32 { return uint32(v[n]) }
	skill := func(i int) uint32 { return u(fmt.Sprint("magicSkill", i)) }
	mask := byte(0)
	for i, n := range []string{"mageUsable", "priestUsable", "alchemistUsable", "psionicUsable"} {
		if v[n] != 0 {
			mask |= 1 << i
		}
	}
	effectiveLevel := func() int32 {
		cls := int(v["casterClass"])
		if cls < 0 || cls >= 15 {
			panic("auto power: invalid class")
		}
		level := func(i int) int32 {
			if autoPowerTables.classOffset[i] == -255 {
				return -1
			}
			return int32(v[fmt.Sprint("classLevel", i)]) + autoPowerTables.classOffset[i]
		}
		total := level(cls)
		for i := 0; i < 15; i++ {
			if i != cls && v[fmt.Sprint("classLevel", i)] != 0 && autoPowerTables.classMask[i]&mask != 0 {
				if n := level(i); n > 0 {
					total += n
				}
			}
		}
		return total
	}
	risk := func(pl, selected uint32) uint32 {
		// 0x4fe269..0x4fe3e1: unsigned weighted skill, level penalty,
		// then the original combat phase scaling node.
		realm := u("spellRealm")
		if realm > 5 {
			panic("auto power: invalid realm")
		}
		expertise := (u(fmt.Sprint("effectiveMagicSkill", int(selected)-24)) + 4*u(fmt.Sprint("realmSkill", realm))) / 5
		difficulty := int32(v["spellCost"])/2 + int32(v["spellLevel"])
		difficulty = min(difficulty, 16)
		if difficulty < 0 {
			panic("auto power: negative skill table index")
		}
		threshold := autoPowerTables.skillThreshold[difficulty] * pl / 7
		failure := uint32(0)
		if expertise < threshold {
			failure = (70*threshold - 70*expertise) / threshold
			if int32(failure) < 0 {
				failure = 0
			}
			if failure > 100 {
				failure = 100
			}
		}
		required := int32(1)
		level := u("spellLevel")
		if level >= 2 && level <= 7 {
			if itemUseThresholdErr != nil {
				panic(itemUseThresholdErr)
			}
			required = int32(itemUseThresholds[level-2])
		}
		penalty := required - effectiveLevel() + int32(pl) - 1
		if penalty > 0 {
			failure += uint32(int32(v["spellLevel"]) * penalty)
		}
		return uint32(combatSpellRiskPhaseScalingNode([]int64{v["casterIndex"], v["combatActive"], v["difficulty"], v["actionProgress"], int64(failure)}, r)[0])
	}
	selectSkill := func(pl uint32) uint32 {
		// 0x4ff81a: nauseated casters use Alchemy when the spell permits it.
		if v["casterNausea"] != 0 && v["alchemistUsable"] != 0 {
			return 26
		}
		highest, learned := uint32(0), uint32(0)
		best, known := int32(-1), int32(-1)
		for i := 0; i < 4; i++ {
			if mask&(1<<i) == 0 {
				continue
			}
			s := int32(skill(i))
			if s > best {
				best = s
				highest = uint32(24 + i)
			}
			if v[fmt.Sprint("magicLearned", i)] != 0 && s > known {
				known = s
				learned = uint32(24 + i)
			}
		}
		if highest == 0 {
			panic("auto power: spell has no usable magic skill")
		}
		if learned != 0 && learned != highest && risk(pl, learned) == 0 {
			return learned
		}
		return highest
	}
	healing := func(spell uint32) uint32 {
		// 0x4fe1c7..0x4fe223: only character targets have HP/stamina gaps.
		if v["targetType"] != 1 {
			return 1
		}
		gap := int32(0)
		switch spell {
		case 6:
			gap = int32(u("maxHP") - u("currentHP"))
		case 13:
			gap = int32(u("maxStamina") - u("currentStamina"))
		case 100:
			gap = int32(u("maxHP") - u("currentHP"))
			if gap == 0 {
				gap = int32(u("maxStamina") - u("currentStamina"))
			}
		}
		if gap <= 0 {
			return 1
		}
		for pl := uint32(1); pl < 8; pl++ {
			failure := risk(pl, selectSkill(pl))
			if failure > 10 {
				return max(1, pl-1)
			} // 0x4fe3ed, 0x4fe462.
			// 0x4fe3f8..0x4fe444: dice count wraps at eight bits; average
			// is truncated toward zero and must strictly exceed the deficit.
			dice := int32(uint8(u("effectDice") * pl))
			mean := (2*int32(int16(v["effectBase"])) + dice*(int32(u("effectSides"))+1)) / 2
			if mean > gap {
				return pl
			}
		}
		return 8 // 0x4fe44b: all seven levels exhausted.
	}
	status := func(duration uint32) uint32 {
		if duration == 9999 {
			return 1
		} // Permanent-status sentinel: 0x4fdf61.
		best, cost := uint32(0), uint32(0)
		for pl := uint32(1); pl < 8; pl++ {
			if int32(u(fmt.Sprint("spellPoints", u("spellRealm")))) < int32(u("spellCost")*pl) {
				break
			}
			failure := risk(pl, selectSkill(pl))
			// 0x4fe14f..0x4fe17b: expected reduction, duration divided by expected reduction and upward rounding,
			// preserving wrapping products.
			expected := pl*pl - pl*pl*failure/100
			if expected == 0 {
				panic("auto power: original division by zero")
			}
			casts := duration / expected
			if duration%expected != 0 {
				casts++
			}
			spent := u("spellCost") * casts * pl
			if spent < cost || pl == 1 {
				cost = spent
				best = pl
			}
		}
		return best
	}
	spell := u("spellID")
	category := u("spellCategory")
	if category < 1 || category > 10 {
		return []int64{1}
	}
	result := uint32(1)
	switch dest := autoPowerTables.category[category-1]; dest {
	case 0x4fe52f:
		if spell < 6 || spell > 100 {
			break
		}
		prefix := "characterStatus"
		if v["targetType"] == 3 {
			prefix = "monsterStatus"
		}
		remaining := func(i int) uint32 { return u(fmt.Sprint(prefix, i)) }
		switch dest = autoPowerTables.spell[spell-6]; dest {
		case 0x4fe54a:
			result = healing(spell)
		case 0x4fe55e:
			result = status(max(remaining(4), remaining(6), remaining(15), remaining(12)))
		case 0x4fe590:
			result = status(remaining(16))
		case 0x4fe5a7:
			result = status(remaining(7))
		case 0x4fe5be:
			result = status(remaining(2))
		case 0x4fe5d5:
			result = status(max(remaining(11), remaining(13), remaining(15)))
		case 0x4fe5fe:
			result = status(remaining(9))
			if v["targetType"] == 1 {
				curse := []int64{v["targetIndex"]}
				for i := 0; i < 12; i++ {
					curse = append(curse, v[fmt.Sprint("equipmentItem", i)], v[fmt.Sprint("curseRemoved", i)])
				}
				for i := 0; i < 12; i++ {
					curse = append(curse, v[fmt.Sprint("itemCursed", i)], v[fmt.Sprint("itemDifficulty", i)])
				}
				result = max(result, uint32(removeCursePowerRequiredNode(curse, r)[0]))
			}
		case 0x4fe685:
		default:
			panic(fmt.Sprintf("auto power: unknown spell dispatch %#x", dest))
		}
	case 0x4fe646: // Party healing passes no individual target (0x4fe654).
		result = 1
	case 0x4fe662:
		if spell == 23 {
			idx := v["identifyItem"]
			requirement := int64(0)
			if idx >= 0 {
				requirement = v[fmt.Sprint("itemDifficulty", idx)]
			}
			result = uint32(identifyPowerRequiredNode([]int64{idx, requirement, v["bonusPercent"]}, r)[0])
		}
	case 0x4fe685:
	default:
		panic(fmt.Sprintf("auto power: unknown category dispatch %#x", dest))
	}
	// Unsigned minimum 1: 0x4fe680..0x4fe68f.
	return []int64{int64(max(1, result))}
}
