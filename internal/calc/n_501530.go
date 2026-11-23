package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

type queuedSpellRules struct {
	compatibility                                           [8]uint32
	height, eyeHeight, scale, tolerance, unlimited, minimum float64
}

var queuedSpellRuleData, queuedSpellRuleErr = loadQueuedSpellRules()

// Dispatch data and floating constants remain in the player's executable.
// The node itself performs no IO and has no mutable state.
func loadQueuedSpellRules() (queuedSpellRules, error) {
	var t queuedSpellRules
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("queued spell: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	read := func(va uint32) (uint32, error) {
		o, err := peOffset(b, va)
		if err != nil {
			return 0, err
		}
		if len(b)-o < 4 {
			return 0, fmt.Errorf("queued spell: truncated data at %#x", va)
		}
		return binary.LittleEndian.Uint32(b[o:]), nil
	}
	for i := range t.compatibility {
		v, err := read(0x5371f8 + 4*uint32(i))
		if err != nil {
			return t, err
		}
		switch v {
		case 0x53717c, 0x5371a4, 0x5371b9, 0x5371cf, 0x5371e5, 0x5371f4:
		default:
			return t, fmt.Errorf("queued spell: unknown compatibility destination %#x", v)
		}
		t.compatibility[i] = v
	}
	for _, c := range []struct {
		va  uint32
		dst *float64
	}{
		{0x603ac8, &t.height}, {0x5ebc64, &t.eyeHeight},
		{0x5ebc40, &t.scale}, {0x5ebb38, &t.tolerance},
		{0x5ec35c, &t.unlimited}, {0x5ebb34, &t.minimum},
	} {
		v, err := read(c.va)
		if err != nil {
			return t, err
		}
		*c.dst = float64(math.Float32frombits(v))
	}
	return t, nil
}

// Fixture: caster slot 0, ally slot 1, monster ID 123, group ID 456,
// up to 255 references to that monster, an item target list and an empty world.
// Combat flags 0x683f94..96 and 0x683fe7 are zero (adventure reachability).
// Record selection is wiring; every variable rule field is an explicit input.
func queuedSpellTargetEligibilityNode(in []int64, _ Rand) []int64 {
	if len(in) != 41 {
		panic("queued spell: expected 41 inputs")
	}
	if itemSpellTargetsErr != nil {
		panic(itemSpellTargetsErr)
	}
	if queuedSpellRuleErr != nil {
		panic(queuedSpellRuleErr)
	}
	if in[1] < 0 || in[1] > 10 || in[2] < -1 || in[2] > 3 ||
		in[3] < 0 || in[3] > 9 || in[4] < -1 || in[4] > 1 ||
		in[22] < -1 || in[22] > 1 || in[38] < 0 || in[38] > 255 {
		panic("queued spell: invalid fixture record or spell range")
	}
	for i, v := range in {
		if i == 2 || i == 4 || i == 5 || i == 6 || i == 22 || i == 40 {
			continue
		}
		max := int64(4294967295)
		switch i {
		case 7, 14, 15, 16, 17, 20, 21, 23, 25:
			max = 255
		}
		if v < 0 || v > max {
			panic("queued spell: field exceeds storage width")
		}
	}
	result := func(ok bool) []int64 {
		if ok {
			return []int64{1}
		}
		return []int64{0}
	}
	t := queuedSpellRuleData
	kind := in[3]
	present, hp, condition := in[7], in[8], in[9]
	if in[4] == 0 {
		present, hp, condition = in[25], in[26], in[27]
	}
	casterAlive := in[25] != 0 && in[26] != 0 && in[27] < 18 // 0x53c283..0x53c2ab.
	valid := func() bool {
		switch kind {
		case 1:
			return in[4] != -1 && present != 0 && hp != 0 && condition < 18 // 0x5361b2..0x536217.
		case 9:
			return in[4] != -1 && present != 0 && hp == 0 && in[10] != 0 && in[11] == 0 // 0x53621c..0x53628c.
		case 3:
			return in[5] == 123 && in[12] != 0 && in[13] == 0 && in[14] != 0 // 0x536291..0x53631e.
		case 4:
			return in[6] == 456 && in[39] != 0 && in[17] != 0 // 0x53631f..0x53639e.
		case 7:
			return in[40] != -1 // 0x53639f..0x5363c6.
		default:
			return true // Remaining target kinds have no record qualification.
		}
	}
	// 0x5010f0 with item flag 0 and source category 6. Reuse the existing
	// runtime-loaded target dispatch; spell 88 and battle presence are special.
	category := int64(0)
	if in[0] != 0 {
		category = itemSpellTargets[in[1]]
		if category == -1 {
			category = 1
			if in[0] == 88 {
				category = 7
			}
		}
		if category == -2 {
			category = 0
			if in[24] == 0 {
				category = 6
			}
		}
	}
	compatible := true // 0x53716f..0x537173: category 0 also succeeds.
	if category >= 1 && category <= 8 {
		switch t.compatibility[category-1] {
		case 0x53717c:
			compatible = ((kind == 1 && in[4] != -1) || (kind == 3 && in[5] != -1)) && valid()
		case 0x5371a4:
			compatible = kind == 9 && in[4] != -1 && valid()
		case 0x5371b9:
			compatible = (kind == 2 || (kind == 4 && in[6] != -1)) && valid()
		case 0x5371cf:
			compatible = kind == 7 && valid()
		case 0x5371e5:
			compatible = kind == 6 && valid()
		case 0x5371f4:
			compatible = true
		}
	}
	if !compatible {
		return result(false)
	}
	// 0x5199f0 uses 0x4e77b0: an incapacitated caster has no range.
	rangeClass := in[2]
	if in[0] == 0 || !casterAlive {
		rangeClass = -1
	}
	f := func(i int) float64 { return float64(math.Float32frombits(uint32(in[i]))) }
	round := func(x float64) float64 { return float64(float32(x)) }
	// Range lengths are literal returns at 0x51933d..0x519356 and
	// 0x51a9b7..0x51a9d0; scale is loaded from 0x5ebc40.
	length := func(k int64) float64 {
		switch k {
		case -1:
			return 0
		case 0:
			return 2 * t.scale
		case 1:
			return 4 * t.scale
		case 2:
			return 25 * t.scale
		case 3:
			return 50 * t.scale
		}
		panic("queued spell: invalid range category")
	}
	monsterDistance := func() float64 {
		dx, dy, dz := f(28)-f(19), f(33)-round(f(31)-t.height), f(34)-f(32)
		d := math.Sqrt(dx*dx+dy*dy+dz*dz) - f(35) - f(36)
		return math.Max(t.minimum, d) // 0x4c7cb0..0x4c7d4a.
	}
	switch kind {
	case 1:
		if in[4] == 0 {
			return result(true)
		} // Self target, 0x51920b.
		if rangeClass == -1 {
			return result(false)
		}
		if rangeClass != 0 {
			return result(true)
		}
		// 0x51b000: opposing formations (difference 2), neither in center
		// (4), are separated when a non-disabled front character exists.
		diff := in[20] - in[21]
		if diff < 0 {
			diff = -diff
		}
		return result(!(in[20] != in[21] && in[20] != 4 && in[21] != 4 && diff == 2 && in[22] != -1 && in[23] == 0))
	case 3:
		// Adventure mode fixes effective range to short (0x519520), requires
		// visibility exactly 1 and reachability byte +0x28c nonzero.
		return result(in[5] == 123 && in[15] == 1 && in[16] != 0 && monsterDistance() <= length(0))
	case 4:
		// 0x537780 traverses group members; a missing caster range becomes
		// zero distance, rather than an unconditional failure.
		return result(in[6] == 456 && in[38] != 0 && round(monsterDistance()) <= round(length(rangeClass)))
	case 6:
		if in[0] == 0 || !casterAlive {
			panic("queued spell: coordinate target requires an active caster spell")
		}
		if in[1] != 5 && in[1] != 6 && in[1] != 8 {
			panic("queued spell: coordinate target requires spell target kind 5, 6 or 8")
		}
		limit := t.unlimited
		dx, dy, dz := f(18)-f(19), f(29)-f(31), f(30)-f(32)
		if in[1] != 5 {
			limit = length(in[2]) + t.tolerance
			dy = round(f(29)-t.eyeHeight) - round(f(31)-t.height) // 0x51938e..0x5193a6.
		}
		// Empty scene indices have no obstructing triangles; original
		// 0x434f20 still executes. No import or game function is replaced.
		return result(math.Sqrt(dx*dx+dy*dy+dz*dz)-f(37) <= limit)
	default:
		return result(true) // 0x51943c..0x5194b5.
	}
}

func init() {
	Nodes["00501530"] = Node{
		Addr: "00501530",
		In:   []string{"spellId", "spellTargetType", "spellRange", "targetKind", "characterTarget", "monsterTarget", "groupTarget", "characterPresent", "characterHP", "characterCondition", "characterRecoverable", "characterRecoveryBlocked", "monsterHP", "monsterRemoved", "monsterPresent", "monsterVisible", "monsterReachable", "groupActive", "targetX", "partyX", "casterFormation", "targetFormation", "frontCharacter", "characterFormationDisabled", "battlePresent", "casterPresent", "casterHP", "casterCondition", "monsterX", "targetY", "targetZ", "partyY", "partyZ", "monsterY", "monsterZ", "monsterRadius", "partyRadius", "coordinatePartyRadius", "groupMemberCount", "groupSize", "targetListItem"},
		Out:  []string{"eligible"}, F: queuedSpellTargetEligibilityNode,
	}
}
