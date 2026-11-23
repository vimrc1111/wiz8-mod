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

var formationRangeConstants, formationRangeError = loadFormationRangeConstants()

// Load angle endpoints, wrap period, bias and degree scale from the player's
// executable before calling the pure rule. PE translation is shared with exe.go.
func loadFormationRangeConstants() ([5]float64, error) {
	var t [5]float64
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("formation range: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	for i, va := range []uint32{0x5ec3fc, 0x5ed1e8, 0x5ec014, 0x5ec318, 0x5ee798} {
		size := 4
		if i == 3 {
			size = 8
		}
		off, err := peOffset(b, va)
		if err != nil {
			return t, err
		}
		if off > len(b)-size {
			return t, fmt.Errorf("formation range: truncated constant %#x", va)
		}
		if size == 8 {
			t[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[off:]))
		} else {
			t[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[off:])))
		}
	}
	return t, nil
}

func init() {
	names := []string{"character", "character0Quadrant", "character1Quadrant", "character2Quadrant", "character3Quadrant", "character4Quadrant", "character5Quadrant", "character6Quadrant", "character7Quadrant", "monsterXBits", "monsterYBits", "monsterZBits", "partyXBits", "partyYBits", "partyZBits", "partyHeadingDegrees"}
	for q := 0; q < 5; q++ {
		for s := 0; s < 3; s++ {
			names = append(names, fmt.Sprintf("quadrant%dSlot%d", q, s))
		}
	}
	for c := 0; c < 8; c++ {
		names = append(names, fmt.Sprintf("character%dFormationUnavailable", c))
	}
	Nodes["0051aec0"] = Node{Addr: "0051aec0", In: names, Out: []string{"rangePenalty"}, F: formationRangePenaltyNode}
}

// Position sources: monster scene object +0x118 (0x4534c0 via 0x4e5af0),
// party world object +0x8c (0x421070), heading 0x686a40 (0x555f66).
// Runtime slots start at 0x687511; flags at [0x685174]+0x17b5 with
// character stride 0x1862. These mutable records are not exe tables.
func formationRangePenaltyNode(in []int64, r Rand) []int64 {
	// Five regions, three slots, center 4, empty 255: 0x51af0d..0x51af4f,
	// 0x51afa3..0x51afd2; eight records: 0x55494e..0x554964.
	const regions, slots, center, empty, partySize = 5, 3, 4, 255, 8
	const slotsStart, flagsStart = 16, 31
	if len(in) != 39 || in[0] < 0 || in[0] >= partySize || in[15] < 0 || in[15] >= 360 {
		panic("formation range: expected character 0..7, u8 quadrant, heading 0..359 and position/formation fields")
	}
	if formationRangeError != nil {
		panic(formationRangeError)
	}
	f := func(i int) float64 {
		if in[i] < 0 || in[i] > math.MaxUint32 {
			panic("formation range: expected float32 bits")
		}
		v := float64(math.Float32frombits(uint32(in[i])))
		if math.IsInf(v, 0) || math.IsNaN(v) {
			panic("formation range: expected finite position")
		}
		return v
	}
	dx, dz := f(9)-f(12), f(11)-f(14)
	f(10)
	f(13) // Y is copied by the children, but does not affect planar direction.
	for _, quadrant := range in[1:9] {
		if quadrant < 0 || quadrant > 255 {
			panic("formation range: expected u8 character quadrant")
		}
	}
	for _, slot := range in[slotsStart:flagsStart] {
		if slot != empty && (slot < 0 || slot >= partySize) {
			panic("formation range: expected character slot 0..7 or empty 255")
		}
	}
	for _, flag := range in[flagsStart:] {
		if flag < 0 || flag > 255 {
			panic("formation range: expected u8 unavailable flag")
		}
	}
	t := formationRangeConstants
	angle := t[1]
	if dz == 0 {
		if dx > 0 {
			angle = t[0]
		}
	} else {
		// 0x4be420 stores the planar angle to float32 before 0x517c60.
		angle = float64(float32(math.Atan2(dx, dz)))
	}
	// 0x517c90 wraps with an x87 64-bit significand; 0x555f58 converts
	// to degrees and 0x5e1c22 truncates. Keep extended precision at boundaries.
	operand := func(v float64) *big.Float { return new(big.Float).SetPrec(64).SetFloat64(v) }
	a := operand(angle)
	a.Add(a, operand(t[3]))
	for a.Sign() < 0 {
		a.Add(a, operand(t[2]))
	}
	for a.Cmp(operand(t[2])) >= 0 {
		a.Sub(a, operand(t[2]))
	}
	a.Mul(a, operand(t[4]))
	degrees, _ := a.Int64()
	relative := degrees - in[15]
	if relative < 0 {
		relative += 360
	} // full turn and half-sector: 0x555f6e/0x555f73.
	quadrant := int((relative + 45) % 360 / 90) // sector width: 0x555f7e..0x555f93.
	characterQuadrant := in[1+int(in[0])]
	if int64(quadrant) == characterQuadrant {
		return []int64{0}
	}
	occupied := func(q int) bool {
		for _, slot := range in[slotsStart+q*slots : slotsStart+(q+1)*slots] {
			if slot != empty && in[flagsStart+int(slot)] == 0 {
				return true
			}
		}
		return false
	}
	penalty := int64(0)
	if occupied(quadrant) {
		penalty = 1
	}
	// The character quadrant is signed in the separation test (0x51af79).
	separation := int64(quadrant) - int64(int8(characterQuadrant))
	if separation < 0 {
		separation = -separation
	}
	if characterQuadrant != center && separation == 2 {
		if penalty != 0 {
			return []int64{2}
		}
		if occupied(center) {
			penalty = 1
		}
	}
	// Only AL is defined as the penalty (0x51afea and caller 0x519659..0x51966a).
	return []int64{penalty}
}
