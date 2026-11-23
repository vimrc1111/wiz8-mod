package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var monsterSpellChoiceWeights, monsterSpellChoiceWeightsErr = loadMonsterSpellChoiceWeights()

func loadMonsterSpellChoiceWeights() ([10]uint32, error) {
	var weights [10]uint32 // Ten template spell slots: 0x533282..0x5332c5.
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return weights, fmt.Errorf("monster spell choice: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return weights, err
	}
	offset, err := peOffset(b, 0x61cc14)
	if err != nil {
		return weights, err
	}
	if len(b)-offset < 4*len(weights) {
		return weights, fmt.Errorf("monster spell choice: truncated weights")
	}
	for i := range weights {
		weights[i] = binary.LittleEndian.Uint32(b[offset+4*i:])
	}
	return weights, nil
}

// The replay fixture uses synthetic coordinate-target spell definitions
// (F+0x137=8). 0x5340bf always appends one position target, without a
// target-eligibility filter; position bits do not change its nonempty result.
// Region objects are absent (0x65ba5c=0), so 0x4d9080 returns false whether
// checks are disabled or enabled. Pointers select the explicitly supplied
// monster, loaded template and spell records; no I/O occurs in this function.
func monsterSpellWeightedChoiceNode(in []int64, r Rand) []int64 {
	if len(in) != 48 {
		panic("monster spell choice: expected 48 inputs")
	}
	if monsterSpellChoiceWeightsErr != nil {
		panic(monsterSpellChoiceWeightsErr)
	}
	for i, v := range in {
		lo, hi := int64(0), int64(4294967295)
		if i == 2 {
			lo, hi = -2147483648, 2147483647
		}
		if i == 1 || i == 3 || i == 4 || (i >= 8 && (i-8)%4 != 3) {
			hi = 255
		}
		if v < lo || v > hi {
			panic("monster spell choice: input width")
		}
	}
	var spells [10]int64
	var weights [10]uint32
	var total uint32
	count := 0
	for i := 0; i < 10; i++ {
		k := 8 + 4*i
		spell, enabled, verbal, combatUsable := in[k], in[k+1], in[k+2], in[k+3]
		// 0x53256f/0x53259a: zero spell or disabled monster use.
		if spell == 0 || enabled == 0 {
			continue
		}
		// 0x4fb1d4..0x4fb21d: silence blocks nonverbal spells too unless
		// the spell has its verbal flag and template type is 4, 5, or 13.
		if in[0] != 0 && (verbal == 0 || (in[1] != 4 && in[1] != 5 && in[1] != 13)) {
			continue
		}
		// 0x532604..0x532625: spell 60 requires an absent special target (-1).
		if spell == 60 && in[2] != -1 {
			continue
		}
		// 0x53262c..0x532653: noncombat spells are unavailable in combat.
		if combatUsable == 0 && in[3] != 0 {
			continue
		}
		spells[count], weights[count] = spell, monsterSpellChoiceWeights[i]
		total += weights[count] // Original unsigned 32-bit accumulation.
		count++
	}
	if count == 0 {
		panic("monster spell choice: original asserts no castable spells at 0x5332cb")
	}
	value := uint32(r.R(int(total))) // One 0x40efa0 call after all ten checks.
	for i := 0; i < count; i++ {
		if value < weights[i] {
			return []int64{spells[i]}
		}
		value -= weights[i]
	}
	panic("monster spell choice: original reads beyond initialized choices")
}

func init() {
	names := []string{"silence", "creatureType", "specialSpellTarget", "combat", "regionChecksDisabled", "position0Bits", "position1Bits", "position2Bits"}
	for i := 0; i < 10; i++ {
		for _, name := range []string{"spell", "enabled", "verbal", "combatUsable"} {
			names = append(names, fmt.Sprintf("%s%d", name, i))
		}
	}
	Nodes["00533260"] = Node{Addr: "00533260", In: names, Out: []string{"spellID"}, F: monsterSpellWeightedChoiceNode}
}
