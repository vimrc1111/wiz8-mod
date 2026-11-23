package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
)

// Node 0x541630: a monster starts an attack. Outputs are the rear flag
// (G+0x9a4), the special flag (G+0x9a5), the two interception counters and the
// eight party facing bytes. The battle text, name lookups and window refreshes
// are not modelled: they write only the message buffer and the UI. The victim
// turn of a monster (0x4c62c0) moves the victim's facing; that effect is not
// modelled either and is not an output.

var monsterFlagsConstants, monsterFlagsError = loadMonsterFlagsConstants()

// Frontal tolerance f32 0x5ebcf4 and the rear threshold f64 0x5ee858 come from
// the player's executable, like the angle constants in n_51aec0.go.
func loadMonsterFlagsConstants() ([2]float64, error) {
	var t [2]float64
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("monster attack flags: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	for i, va := range []uint32{0x5ebcf4, 0x5ee858} {
		size := 4 + 4*i
		off, err := peOffset(b, va)
		if err != nil {
			return t, err
		}
		if off > len(b)-size {
			return t, fmt.Errorf("monster attack flags: truncated constant %#x", va)
		}
		if i == 0 {
			t[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[off:])))
		} else {
			t[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[off:]))
		}
	}
	return t, nil
}

// Input positions after the 279 inputs shared with node 0x53d590.
const (
	mfTargetType     = 280
	mfMonsterType    = 281
	mfAttackIndex    = 282
	mfRangeType      = 283
	mfTemplateClass  = 284
	mfAttack         = 286 // 3 records of E0..E3, 12 values
	mfSpellRange     = 298 // 6 values
	mfSwingCount     = 304
	mfVictimAlive    = 305
	mfVictimAwake    = 306
	mfVictimLevel    = 307
	mfVictimStun     = 308
	mfVictimTurnable = 309
	mfVictimTurns    = 310
	mfVictimAttempts = 311
	mfAttackerX      = 312
	mfAttackerZ      = 313
	mfVictimX        = 314
	mfVictimZ        = 315
	mfPartyX         = 316
	mfPartyZ         = 317
	mfAttackerFacing = 318
	mfVictimFacing   = 319
	mfPartyHeading   = 320
	mfFace           = 321 // 8 bytes
	mfInputs         = 331
)

func monsterAttackFlagsNode(in []int64, r Rand) []int64 {
	if len(in) != mfInputs {
		panic("monster attack flags: wrong input count")
	}
	if monsterFlagsError != nil {
		panic(monsterFlagsError)
	}
	if formationRangeError != nil {
		panic(formationRangeError)
	}
	slot := in[0]
	if slot < 0 || slot > 7 {
		panic("monster attack flags: expected party slot 0..7")
	}
	// Attack record pointer D+0xe7+0x22*k, three records (exe 0x541657).
	k := in[mfAttackIndex]
	if k < 0 || k > 2 {
		panic("monster attack flags: expected attack index 0..2")
	}
	e := func(i int) int64 { return in[mfAttack+4*int(k)+i] }
	for _, v := range in[mfAttack : mfAttack+12] {
		if v < 0 || v > 255 {
			panic("monster attack flags: expected u8 attack record bytes")
		}
	}
	face := make([]int64, 8)
	for i := range face {
		face[i] = in[mfFace+i]
		if face[i] < 0 || face[i] > 255 {
			panic("monster attack flags: expected u8 facing")
		}
	}
	for _, i := range []int{mfVictimTurns, mfTemplateClass} {
		if in[i] < 0 || in[i] > 255 {
			panic("monster attack flags: expected u8 field")
		}
	}
	for _, i := range []int{mfVictimLevel, mfVictimStun, mfVictimAttempts} {
		if in[i] < 0 || in[i] > math.MaxUint32 {
			panic("monster attack flags: expected u32 field")
		}
	}

	// 0x51a730: range of the chosen attack, by range type (jump table 0x51a7d8).
	rng := int64(-1)
	if uint32(in[mfRangeType]) <= 8 {
		switch in[mfRangeType] {
		case 0: // attack record range byte E3; E0 zero is an assertion (0x51a785)
			if e(0) == 0 {
				panic("monster attack flags: empty attack record")
			}
			rng = e(3)
		case 2: // spell range dword, 0x1bf bytes per spell, offset 0x12f (0x51a7aa)
			if in[mfMonsterType] < 0 || in[mfMonsterType] > 5 {
				panic("monster attack flags: expected spell id 0..5")
			}
			rng = int64(int32(in[mfSpellRange+int(in[mfMonsterType])]))
		case 3:
			rng = 2
		case 8:
			rng = 0
		}
	}
	rear, special := int64(0), int64(0)
	if in[mfTemplateClass] == 3 && rng <= 0 { // 0x54167f..0x541691
		special = 1
	}
	if special != 0 && e(1) == 0 { // text id assertion, exe 0x541af0
		panic("monster attack flags: special attack without text id")
	}
	interceptAttempts := in[278]
	victimAttempts := in[mfVictimAttempts]
	switch in[mfTargetType] {
	case 1: // character victim (0x5418da)
		if rng <= 1 {
			q := monsterQuadrant(in)
			if q != face[slot] { // 0x555960: not already facing the attacker
				child := playerTurnInterceptNode(in[:279], r)
				interceptAttempts = child[15]
				if child[0] != 0 { // 0x555820: turn toward the attacker
					face[slot] = q
				} else if d := q - int64(int8(face[slot])); d == 2 || d == -2 { // 0x555c60
					rear = 1
				}
			}
		}
	case 3: // monster victim (0x5419a1)
		if rng <= 1 && !monsterFrontal(in) { // 0x555b00
			if in[mfVictimAlive] != 0 && in[mfVictimAwake] != 0 && uint32(in[mfVictimLevel]) < 14 &&
				in[mfVictimStun] == 0 && in[mfVictimTurnable] == 1 { // 0x541a05..0x541a35
				n := uint32(victimAttempts)
				turned := n == 0
				if !turned {
					threshold := uint32(in[mfVictimTurns]) - 25*n // u32, no clamp (0x541a45..0x541a63)
					turned = uint32(r.R(100)) < threshold
				}
				victimAttempts = int64(n + 1)
				if !turned && monsterRear(in) {
					rear = 1
				}
			} else if monsterRear(in) {
				rear = 1
			}
		}
	default:
		panic("monster attack flags: expected victim kind 1 or 3")
	}
	out := []int64{rear, special, interceptAttempts, victimAttempts}
	return append(out, face...)
}

func init() {
	base := Nodes["0053d590"]
	if base.F == nil {
		panic("monster attack flags: node 0x53d590 is not registered")
	}
	names := append([]string{}, base.In...)
	names = append(names, "continuation", "targetType", "monsterType", "attackIndex", "rangeType", "templateClass", "templateDisturb",
		"attackE0k0", "attackE1k0", "attackE2k0", "attackE3k0", "attackE0k1", "attackE1k1", "attackE2k1", "attackE3k1", "attackE0k2", "attackE1k2", "attackE2k2", "attackE3k2",
		"spellRange0", "spellRange1", "spellRange2", "spellRange3", "spellRange4", "spellRange5", "swingCount",
		"victimAlive", "victimAwake", "victimLevel", "victimStun", "victimTurnable", "victimTurns", "victimAttempts",
		"attackerX", "attackerZ", "victimX", "victimZ", "partyX", "partyZ", "attackerFacing", "victimFacing", "partyHeading",
		"face0", "face1", "face2", "face3", "face4", "face5", "face6", "face7", "notifyKind", "verboseMode")
	Nodes["00541630"] = Node{Addr: "00541630", In: names,
		Out: []string{"rearFlag", "specialFlag", "interceptAttemptsOut", "victimAttemptsOut", "face0Out", "face1Out", "face2Out", "face3Out", "face4Out", "face5Out", "face6Out", "face7Out"},
		F:   monsterAttackFlagsNode}
}

func mfBig(v float64) *big.Float { return new(big.Float).SetPrec(64).SetFloat64(v) }

func mfF32(in []int64, i int) float64 {
	if in[i] < 0 || in[i] > math.MaxUint32 {
		panic("monster attack flags: expected float32 bits")
	}
	v := float64(math.Float32frombits(uint32(in[i])))
	if math.IsInf(v, 0) || math.IsNaN(v) {
		panic("monster attack flags: expected finite float32")
	}
	return v
}

// 0x4be420 then 0x517c60: planar angle of (dx,dz), stored as float32, plus pi
// wrapped into [0,2pi) at x87 64-bit precision. The result is the extended value.
func mfWrapped(dx, dz float64) *big.Float {
	t := formationRangeConstants
	angle := t[1]
	if dz == 0 {
		if dx > 0 {
			angle = t[0]
		}
	} else {
		angle = float64(float32(math.Atan2(dx, dz)))
	}
	return mfReduce(new(big.Float).SetPrec(64).Add(mfBig(angle), mfBig(t[3])))
}

// 0x517c90..0x517cda (and 0x517d20..0x517d76): subtract/add the f32 period.
func mfReduce(v *big.Float) *big.Float {
	period := mfBig(formationRangeConstants[2])
	for v.Sign() < 0 {
		v.Add(v, period)
	}
	for v.Cmp(period) >= 0 {
		v.Sub(v, period)
	}
	return v
}

// Quadrant of the attacker around the party relative to its heading, shared by
// 0x555960, 0x555820 and 0x555c60 (0x555978..0x5559f1).
func monsterQuadrant(in []int64) int64 {
	if in[mfPartyHeading] < 0 || in[mfPartyHeading] >= 360 {
		panic("monster attack flags: expected heading 0..359")
	}
	dx := mfF32(in, mfAttackerX) - mfF32(in, mfPartyX)
	dz := mfF32(in, mfAttackerZ) - mfF32(in, mfPartyZ)
	mfF32(in, mfAttackerX+0)
	a := mfWrapped(dx, dz)
	a.Mul(a, mfBig(formationRangeConstants[4]))
	degrees, _ := a.Int64()
	relative := degrees - in[mfPartyHeading]
	if relative < 0 {
		relative += 360
	}
	q := (relative + 45) % 360 / 90 // sector width 90, idiv remainder
	if q < 0 || q > 3 {
		panic("monster attack flags: quadrant outside 0..3")
	}
	return q
}

// Bearing from victim to attacker as float32 (0x4be420, 0x517c60) and the victim
// facing (0x453970).
func monsterBearings(in []int64) (alpha, beta float64) {
	dx := mfF32(in, mfAttackerX) - mfF32(in, mfVictimX)
	dz := mfF32(in, mfAttackerZ) - mfF32(in, mfVictimZ)
	a, _ := mfWrapped(dx, dz).Float32()
	return float64(a), mfF32(in, mfVictimFacing)
}

// 0x555b00: the victim already faces the attacker within f32 0x5ebcf4.
func monsterFrontal(in []int64) bool {
	alpha, beta := monsterBearings(in)
	d, _ := new(big.Float).SetPrec(64).Sub(mfBig(alpha), mfBig(beta)).Float32()
	return math.Abs(float64(d)) <= monsterFlagsConstants[0]
}

// 0x555de0 with 0x517ce0: the shortest wrapped angle difference reaches the
// rear threshold f64 0x5ee858.
func monsterRear(in []int64) bool {
	alpha, beta := monsterBearings(in)
	pi := mfBig(formationRangeConstants[3])
	x32, _ := new(big.Float).SetPrec(64).Sub(mfBig(alpha), mfBig(beta)).Float32()
	w32, _ := mfReduce(new(big.Float).SetPrec(64).Add(mfBig(float64(x32)), pi)).Float32()
	y32, _ := new(big.Float).SetPrec(64).Sub(mfBig(beta), mfBig(alpha)).Float32()
	u := mfReduce(new(big.Float).SetPrec(64).Add(mfBig(float64(y32)), pi))
	least := u
	if w := mfBig(float64(w32)); w.Cmp(u) < 0 {
		least = w
	}
	return least.Cmp(mfBig(monsterFlagsConstants[1])) >= 0
}
