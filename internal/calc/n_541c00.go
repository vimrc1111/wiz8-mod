package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

// player_arhit (0x541c00): attacker to-hit rating ARhit, notes/re/combat-2.md:26.
// Inputs 0..27 are exactly defhitNode's (index 1 is the shared attack mode, which
// the original passes both to 0x542e10 and 0x542ee0); the rest are this
// node's own fields. The fixture keeps the attacker slot (who 0..3) and the
// target character slot (characterIndex 4..7) distinct so their records never alias.
const (
	arTargetType = 0
	arMode       = 1
	arTargetChar = 2
	arAmbush     = 4 // G+0x9a4, same byte defhit reads
	arSurprise   = 5 // G+0x9a5
	arOwn        = 28
	arInputs     = arOwn + arLevels + 16 // 73
)

// Offsets within the own block (in[arOwn+k]).
const (
	arWho = iota
	arHand
	arSkip
	arB0
	arB1
	arJ0
	arJ1
	arKind0
	arKind1
	arB1Skill // C+0x1771
	arFatigue
	arT94
	arSpd
	arA0
	arA1
	arA2
	arCap
	arClass
	arRace
	arDifficulty
	arReverse
	arT0
	arT1
	arT2
	arT3
	arM0
	arM1
	arM2
	arM3
	arLevels // 16 entries
)

var arHitModeDispatch, arHitModeErr = loadArHitModeDispatch()

// Read the 9-entry HitMode jump table (0x542e98) from the player's executable.
func loadArHitModeDispatch() ([9]uint32, error) {
	var t [9]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("player arhit: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	o, err := peOffset(b, 0x542e98)
	if err != nil {
		return t, err
	}
	if len(b)-o < 4*len(t) {
		return t, fmt.Errorf("player arhit: truncated HitMode dispatch")
	}
	for i := range t {
		t[i] = binary.LittleEndian.Uint32(b[o+4*i:])
		switch t[i] {
		case 0x542e2b, 0x542e34, 0x542e3d, 0x542e43, 0x542e4c, 0x542e55:
		default:
			return t, fmt.Errorf("player arhit: unknown HitMode destination %#x", t[i])
		}
	}
	return t, nil
}

func playerArhitNode(in []int64, r Rand) []int64 {
	if len(in) != arInputs {
		panic("player arhit: expected 73 inputs")
	}
	if arHitModeErr != nil {
		panic(arHitModeErr)
	}
	if breathTablesErr != nil {
		panic(breathTablesErr)
	}
	if defenseFatigueError != nil {
		panic(defenseFatigueError)
	}
	own := in[arOwn:]
	mode, skip := in[arMode], own[arSkip]
	if mode < 0 || mode > 8 { // 0x542e1f: modes above 8 hit the assert
		panic("player arhit: mode outside 0..8")
	}
	hand := own[arHand]
	if hand < 0 || hand > 1 || own[arWho] < 0 || own[arWho] > 3 || in[arTargetChar] < 4 || in[arTargetChar] > 7 {
		panic("player arhit: expected hand 0..1, attacker slot 0..3, target slot 4..7")
	}
	if skip < 0 || skip > 255 {
		panic("player arhit: skip flag is one byte")
	}

	// 0x541c47..0x541c61: B + 5J + 5*i8(C+0x1771) in 32-bit arithmetic.
	b := int32(own[arB0+int(hand)])
	j := int32(own[arJ0+int(hand)])
	kind := own[arKind0+int(hand)]
	skill := int32(int8(own[arB1Skill]))
	a := b + 5*j + 5*skill

	// 0x542e10(0, mode): HitMode, destination -> immediate (exe 0x542e2b..0x542e55).
	a += map[uint32]int32{0x542e2b: 10, 0x542e34: 5, 0x542e3d: 0, 0x542e43: -5, 0x542e4c: -10, 0x542e55: -30}[arHitModeDispatch[mode]]
	if in[arAmbush] != 0 { // 0x541c86
		a += 10
	}
	if in[arSurprise] != 0 { // 0x541c9c
		a += 10
	}

	// 0x52a7d0 fatigue category (dispatch shared with defense adjustment), > 4 returns itself.
	fh := uint32(own[arFatigue])
	if fh <= 4 {
		fh = map[uint32]uint32{0x52a7e0: 0, 0x52a7e3: 5, 0x52a7e9: 10, 0x52a7ef: 20, 0x52a7f5: 40}[defenseFatigueDispatch[fh]]
	}
	if kind == 7 { // 0x541cbd: shr eax,1
		fh >>= 1
	}
	a -= int32(fh)

	if t := uint32(own[arT94]); t > 0 { // 0x541cc8: unsigned compare
		x := int32(own[arSpd]) - int32(25*t+50) + b // 0x541cd2..0x541ce5, 32-bit wrap
		if x < 0 {
			x = 0
		}
		// 0x541cf5..0x541d13: _CIpow(x, 0.75) (exe f64 @0x5ebf40), 50 - pow (f64 @0x5ee7b8), _ftol, clamp at 0.
		p := int32(int64(50 - math.Pow(float64(x), 0.75)))
		if p < 0 {
			p = 0
		}
		a -= p
	}

	if byte(skip) == 0 { // 0x541d19: low byte of the fourth argument
		hit := int32(defhitNode(in[:28], r)[0])
		a += -5 * hit // 0x541d4b..0x541d54
		mine := int32(own[arA0]) + int32(own[arA1]) + int32(own[arA2]) + int32(own[arSpd])
		var theirs int32
		switch in[arTargetType] {
		case 3: // monster: u8 M+0x248..0x24b
			theirs = int32(own[arM0] + own[arM1] + own[arM2] + own[arM3])
		case 1: // character: C+0xfd, +0x139, +0x14d, +0x161
			theirs = int32(own[arT0]) + int32(own[arT1]) + int32(own[arT2]) + int32(own[arT3])
		default:
			panic("player arhit: target kind other than 1 or 3 asserts in 0x542ee0")
		}
		a += (mine - theirs) / 10 // 0x541e2d..0x541e44: signed truncating division by 10
	}

	if uint32(own[arCap]) > 0 && a > 10 { // 0x541e4a, 0x541e5d
		has7 := false // 0x547940(C, 7): class row C+0x69, race row C+0x71, -1 skips
		if c := own[arClass]; c != -1 {
			for _, f := range breathTables.class[c] {
				has7 = has7 || f == 7
			}
		}
		if rc := own[arRace]; rc != -1 {
			for _, f := range breathTables.race[rc] {
				has7 = has7 || f == 7
			}
		}
		if !has7 {
			a = 10
		} else {
			// 0x5479b0: level at C+0x8d+4*class; > 20 returns m, else (2L+60)*m*f32(0.01).
			m := float32(a - 10)
			l := uint32(own[arLevels+int(own[arClass])+1])
			v := int64(m)
			if l <= 20 {
				// Product is exact in x87 extended precision: (2L+60)*m*5368709/2^29,
				// f32 0.01 = 5368709/536870912 (notes/re/combat-2.md:13).
				v = (int64(2*l) + 60) * int64(m) * 5368709 / 536870912
			}
			a = int32(v) + 10 // _ftol keeps the low 32 bits
		}
	}
	a = int32(difficultyNode([]int64{0, 0, 0, 0, 0, 0, 0, 0, int64(a), own[arDifficulty], own[arReverse], own[arWho]}, r)[0])
	return []int64{int64(a)}
}

func init() {
	in := []string{
		"targetType", "mode", "characterIndex", "combat", "ambush", "surprise",
		"speedDefense", "shieldDefense", "characterDefenseSuppressed", "monsterDefense",
		"monsterHitDefense", "fatigue", "monsterPosture", "monsterDefenseSuppressed",
		"ambushDefenseMask", "completedActionSelector", "completedPosture", "pendingPosture",
		"sceneFlags", "HP", "status", "actionCompleted", "mainAction", "offAction",
		"monsterID", "templateIndex", "characterDefHit", "monsterAC",
		"who", "hand", "skipDefense",
		"handB0", "handB1", "handJ0", "handJ1", "handKind0", "handKind1", "skillBonus", "fatigueCategory",
		"battleRecordT94", "speed", "attackerStat0", "attackerStat1", "attackerStat2",
		"levelCapState", "currentClass", "race", "difficulty", "reverseDifficultyState",
		"targetCharStat0", "targetCharStat1", "targetCharSpeed", "targetCharStat3",
		"monsterStat0", "monsterStat1", "monsterStat2", "monsterStat3",
	}
	for k := -1; k < 15; k++ {
		in = append(in, fmt.Sprintf("classLevel%d", k))
	}
	Nodes["00541c00"] = Node{Addr: "00541c00", In: in, Out: []string{"arhit"}, F: playerArhitNode}
}
