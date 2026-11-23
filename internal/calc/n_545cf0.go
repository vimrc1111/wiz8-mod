package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

type interceptRules struct {
	character, monster     [9]uint32
	height, scale, minimum float64
}

var interceptData, interceptDataErr = loadInterceptRules()

// Load immutable dispatch tables and geometry constants before replay. The
// pure node does no IO; PE translation is shared with exe.go.
func loadInterceptRules() (interceptRules, error) {
	var t interceptRules
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("monster intercept: set WIZ8_EXE")
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
			return 0, fmt.Errorf("monster intercept: truncated table %#x", va)
		}
		return binary.LittleEndian.Uint32(b[off:]), nil
	}
	for _, table := range []struct {
		va  uint32
		dst *[9]uint32
	}{{0x51a4d8, &t.character}, {0x51a6f0, &t.monster}} {
		for i := range table.dst {
			v, e := read(table.va + 4*uint32(i))
			if e != nil {
				return t, e
			}
			table.dst[i] = v
		}
	}
	for _, c := range []struct {
		va  uint32
		dst *float64
	}{{0x603ac8, &t.height}, {0x5ebc40, &t.scale}, {0x5ebb34, &t.minimum}} {
		v, e := read(c.va)
		if e != nil {
			return t, e
		}
		*c.dst = float64(math.Float32frombits(v))
	}
	return t, nil
}

func init() {
	Nodes["00545cf0"] = Node{Addr: "00545cf0", In: []string{
		"targetType", "characterIndex", "characterSlotActive", "characterCurrentHP", "characterMaxHP", "characterStatus", "characterCharm", "characterLevel", "characterInterceptState", "actorLevel", "actorDisposition", "victimActive", "victimCurrentHP", "victimMaxHP", "victimVisible", "victimSpecialTemplate", "victimLevel", "victimDisposition", "actorInterceptState", "actorSpecialTemplate", "actionType", "weaponRange", "alternateHand", "alternateAction", "characterReachEnabled", "characterReach0", "characterReach1", "characterReach2", "characterReach3", "relationCount", "relationActive", "monsterReach0", "monsterReach1", "monsterReach2", "monsterReach3", "spellRange", "actorXBits", "actorYBits", "actorZBits", "victimXBits", "victimYBits", "victimZBits", "partyXBits", "partyYBits", "partyZBits", "actorRadiusBits", "victimRadiusBits", "partyRadiusBits",
	}, Out: []string{"interceptAvailable"}, F: monsterInterceptAvailableNode}
}

// Fixture resolves one interceptor and one distinct target monster through
// real active lists and loaded templates. Character selection uses all eight
// slots. Action index is zero (primary attack); combat formation adjustment
// at 0x51a41c is disabled by the fixture's adventure-mode flag 0x683f94=0.
// Gates, action context, range, HP, levels and geometry are independent inputs.
func monsterInterceptAvailableNode(in []int64, r Rand) []int64 {
	if len(in) != 48 {
		panic("monster intercept: expected 48 fields")
	}
	for i, v := range in {
		if i == 6 || i == 35 {
			if v < -2147483648 || v > 2147483647 {
				panic("monster intercept: i32 field out of range")
			}
			continue
		}
		limit := int64(4294967295)
		switch i {
		case 1:
			limit = 7
		case 2, 8, 9, 10, 11, 14, 15, 16, 17, 18, 19, 21, 22, 23, 24, 25, 26, 27, 28, 30, 31, 32, 33, 34:
			limit = 255
		case 29:
			limit = 1
		}
		if v < 0 || v > limit {
			panic("monster intercept: field out of range")
		}
	}
	no := []int64{0}
	kind := in[0]
	if kind != 1 && kind != 3 {
		return no
	} // 0x545d09 / 0x545d8b.
	var hp, maxHP, level, state int64
	if kind == 1 {
		// 0x53c270 reads selected presence, nonzero HP and unsigned status <18.
		slots := make([]int64, 25)
		slots[0] = in[1]
		j := 1 + 3*int(in[1])
		slots[j], slots[j+1], slots[j+2] = in[2], int64(int32(in[3])), in[5]
		if characterTargetableNode(slots, r)[0] == 0 {
			return no
		}
		if targetDispositionNode([]int64{in[1], in[6], in[10]}, r)[0] != 2 {
			return no
		}
		if !interceptReach(in, true) {
			return no
		}
		hp, maxHP, level, state = in[3], in[4], in[7], in[8]
	} else {
		// 0x545db1..0x545dde: live, visible and ordinary target template.
		if in[11] == 0 || in[12] == 0 || in[14] == 0 || in[15] != 0 {
			return no
		}
		// 0x546f85..0x546fa9: equal dispositions return friendly=2.
		// The fixture's template IDs 0 and 1 cannot be the special ID 548;
		// unequal dispositions only produce neutral/hostile, never friendly.
		if in[10] != in[17] {
			return no
		}
		if in[19] != 0 || !interceptReach(in, false) {
			return no
		}
		hp, maxHP, level, state = in[12], in[13], in[16], in[18]
	}
	if maxHP == 0 {
		panic("monster intercept: original divides by zero")
	}
	// 0x545d6d / 0x545e16 truncate the product to u32 before unsigned divide.
	// 0x545e31 compares levels unsigned; 0x545e35 uses the strict 40% cutoff.
	percent := uint32(hp) * 100 / uint32(maxHP)
	if uint32(level) > uint32(in[9]) && (percent < 40 || state != 0) {
		return []int64{1}
	}
	return no
}

func interceptReach(in []int64, character bool) bool {
	if interceptDataErr != nil {
		panic(interceptDataErr)
	}
	t := interceptData
	action, weapon := in[20], in[21]
	// 0x51a308 / 0x51a55d choose an enable byte from the action context.
	gate := 0
	switch action {
	case 0:
		if weapon == 2 || weapon == 3 {
			if in[22] != 0 {
				gate = 1
			}
		}
	case 2:
		gate = 2
		if in[23] != 0 {
			gate = 3
		}
	case 3:
		gate = 2
	}
	if character {
		if in[24] != 1 || in[25+gate] == 0 {
			return false
		}
	} else if in[29] == 0 || in[30] != 1 || in[31+gate] == 0 {
		return false
	}
	if action > 8 {
		return false
	}
	dispatch := t.monster[action]
	if character {
		dispatch = t.character[action]
	}
	var rangeClass int64
	// The destinations name semantic branches; table entries stay in Wiz8.exe.
	switch dispatch {
	case 0x51a40e, 0x51a6e6:
		return false // Unsupported action context: no legal reach.
	case 0x51a38e, 0x51a5e0:
		rangeClass = weapon // primary template attack +0xea.
	case 0x51a3e1, 0x51a64a:
		rangeClass = in[35] // selected spell +0x12f, stride 447.
	case 0x51a403, 0x51a66d:
		rangeClass = 2
	case 0x51a40a, 0x51a674:
		rangeClass = 0
	default:
		panic("monster intercept: unknown reach dispatch")
	}
	if rangeClass == -1 {
		return false
	}
	// 0x51a464..0x51a47d and 0x51a646..0x51a68d, in game range units.
	var reach float64
	switch rangeClass {
	case 0:
		reach = 2
	case 1:
		reach = 4
	case 2:
		reach = 25
	case 3:
		reach = 50
	default:
		panic("monster intercept: invalid range class")
	}
	f := func(i int) float64 { return float64(math.Float32frombits(uint32(in[i]))) }
	var dx, dy, dz, radius float64
	if character {
		// 0x4c7cf4 stores the adjusted party height to float32 before subtraction.
		py := float64(float32(f(43) - t.height))
		dx, dy, dz = f(36)-f(42), f(37)-py, f(38)-f(44)
		radius = f(45) + f(47)
	} else {
		dx, dy, dz = f(36)-f(39), f(37)-f(40), f(38)-f(41)
		radius = f(45) + f(46)
	}
	distance := math.Sqrt((dz*dz+dy*dy)+dx*dx) - radius
	if distance < t.minimum {
		distance = t.minimum
	}
	// The original rejects unordered as well as beyond-range comparisons.
	return distance <= reach*t.scale
}
