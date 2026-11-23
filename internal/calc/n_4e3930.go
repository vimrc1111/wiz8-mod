package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

var monsterHeadingConstants, monsterHeadingError = loadMonsterHeadingConstants()

// Heading endpoint constants belong to the user's executable, not a copied table.
// The loader follows exe.go; loading happens before the pure node is called.
func loadMonsterHeadingConstants() ([2]uint32, error) {
	var result [2]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return result, fmt.Errorf("monster heading: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	// 0x4be60d and 0x4be617 select these float32 constants when Z displacement is zero.
	for i, va := range []uint32{0x5ec3fc, 0x5ed1e8} {
		off, err := peOffset(b, va)
		if err != nil {
			return result, err
		}
		if off > len(b)-4 {
			return result, fmt.Errorf("monster heading: truncated constant %#x", va)
		}
		result[i] = binary.LittleEndian.Uint32(b[off:])
	}
	return result, nil
}

// monsterHpSpInitNode initializes one member of a group in an empty scene index.
// List storage and world grid cell size (1) are case wiring. All source group,
// template dice, position, party position, identifier and battle flag values are inputs.
// Finite float32 positions are supplied as their raw bits. A zero SP roll is an
// original division fault at 0x52a78f, so it has no normal output to reproduce.
func monsterHpSpInitNode(in []int64, r Rand) []int64 {
	if len(in) != 21 {
		panic("monster HP/SP init: expected 21 inputs")
	}
	if monsterHeadingError != nil {
		panic(monsterHeadingError)
	}
	for i, v := range in {
		switch i {
		case 0, 3:
			if v < -32768 || v > 32767 {
				panic("monster HP/SP init: expected i16 dice base")
			}
		case 1, 2, 4, 5, 6, 12:
			if v < 0 || v > 255 {
				panic("monster HP/SP init: expected byte")
			}
		default:
			if v < 0 || v > math.MaxUint32 {
				panic("monster HP/SP init: expected u32")
			}
		}
	}
	position := func(i int) float64 {
		v := float64(math.Float32frombits(uint32(in[i])))
		if math.IsNaN(v) || math.IsInf(v, 0) {
			panic("monster HP/SP init: nonfinite world position")
		}
		return v
	}
	dx, dz := position(16)-position(13), position(18)-position(15)
	position(14)
	position(17)
	var heading uint32
	if dz == 0 {
		heading = monsterHeadingConstants[1]
		if dx > 0 {
			heading = monsterHeadingConstants[0]
		}
	} else {
		// 0x4be621 computes the quadrant-aware X/Z angle and 0x4e39cb stores float32.
		heading = math.Float32bits(float32(math.Atan2(dx, dz)))
	}
	hp := uint32(diceNode(in[:3], r)[0])
	sp := uint32(diceNode(in[3:6], r)[0])
	if sp == 0 {
		panic("monster HP/SP init: original SP division by zero")
	}
	// 0x52a780: multiplication wraps at u32, division is unsigned, depletion
	// thresholds are 50/70/85/95; 0x4e39f0 supplies current=max=the SP roll.
	depletion := int32(100 - (sp*100)/sp)
	tier := uint32(0)
	switch {
	case depletion >= 95:
		tier = 4
	case depletion >= 85:
		tier = 3
	case depletion >= 70:
		tier = 2
	case depletion >= 50:
		tier = 1
	}
	// 0x4e395d..0x4e397c reserves a nonzero identifier, wrapping the counter.
	id := uint32(in[7])
	next := id + 1
	if id == 0 {
		id = next
		next++
	}
	normal, special := int64(1), int64(0)
	if in[6] != 0 {
		normal, special = 0, 1
	}
	flags := uint32(in[20])
	// 0x565420 marks battle scene content dirty only in mode 7; the case has a battle object.
	if in[19] == 7 {
		flags |= 0x20000
	}
	// Initialization constants: float32 -1 at 0x4e399a, invalid scene handle -1
	// at 0x4e3a56, and zero state fields throughout 0x4e3951..0x4e3a6b.
	return []int64{
		int64(hp), int64(hp), int64(sp), int64(sp), int64(tier), int64(heading), int64(id),
		in[8], in[11], in[12], in[13], in[14], in[15], 0xbf800000, 0xffffffff,
		0, 0, 0, 0, 0, 0, 0, 0,
		int64(next), int64(uint32(in[9]) + 1), int64(uint32(in[10]) + 1), 1,
		normal, special, int64(flags),
	}
}

func init() {
	Nodes["004e3930"] = Node{
		Addr: "004e3930",
		In:   []string{"hpBase", "hpDice", "hpSides", "spBase", "spDice", "spSides", "specialMonster", "nextMonsterId", "groupId", "groupMembers", "groupTotal", "templateId", "groupKind", "positionX", "positionY", "positionZ", "partyX", "partyY", "partyZ", "mode", "battleFlags"},
		Out:  []string{"maxHP", "HP", "maxSP", "SP", "spDepletion", "heading", "monsterId", "groupIdCopy", "templateIdCopy", "groupKindCopy", "positionXCopy", "positionYCopy", "positionZCopy", "actionTimeUnset", "sceneHandle", "mainCondition", "conditionCompanionState", "presence", "enabled", "actionControlFlag", "actionSecondaryFlag", "auxiliaryActionState", "auxiliaryActionFlag", "nextMonsterIdAfter", "groupMembersAfter", "groupTotalAfter", "groupIdCount", "normalCount", "specialCount", "battleFlagsAfter"},
		F:    monsterHpSpInitNode,
	}
}
