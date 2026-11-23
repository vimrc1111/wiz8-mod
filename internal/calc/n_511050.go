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

type monsterRepositionConstants struct{ jitter, offset, half, zero, one, period, bias float64 }

var monsterRepositionData, monsterRepositionError = loadMonsterRepositionConstants()

// Load before replay so the rule itself performs no IO. PE address translation
// is shared with exe.go; constants remain in the user's executable.
func loadMonsterRepositionConstants() (monsterRepositionConstants, error) {
	var t monsterRepositionConstants
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("monster reposition: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	for _, c := range []struct {
		va   uint32
		dst  *float64
		size int
	}{
		{0x5ed828, &t.jitter, 4}, {0x5ec04c, &t.offset, 4}, {0x5ebb30, &t.half, 4},
		{0x5ebb34, &t.zero, 4}, {0x5ebb38, &t.one, 4}, {0x5ec014, &t.period, 4}, {0x5ec318, &t.bias, 8},
	} {
		off, e := peOffset(b, c.va)
		if e != nil {
			return t, e
		}
		if off > len(b)-c.size {
			return t, fmt.Errorf("monster reposition: truncated constant %#x", c.va)
		}
		if c.size == 8 {
			*c.dst = math.Float64frombits(binary.LittleEndian.Uint64(b[off:]))
		} else {
			*c.dst = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[off:])))
		}
	}
	return t, nil
}

func init() {
	Nodes["00511050"] = Node{Addr: "00511050", In: []string{
		"offsetEnabled", "requestedDistanceBits", "partyXBits", "partyYBits", "partyZBits", "partyHeadingBits", "partyRadiusBits", "groupActive", "groupMemberCount", "mainRadiusBits",
		"linkedGroup1", "linkedRadius1Bits", "linkedGroup2", "linkedRadius2Bits", "linkedGroup3", "linkedRadius3Bits", "linkedGroup4", "linkedRadius4Bits",
		"headingTarget0Bits", "headingTarget1Bits", "headingTarget2Bits",
		"sceneRadius0Bits", "sceneRadius1Bits", "sceneRadius2Bits", "sceneRadius3Bits", "sceneRadius4Bits",
		"linkedMemberCount1", "linkedMemberCount2", "linkedMemberCount3", "linkedMemberCount4", "partyPlacementClearanceBits",
		"originalModelPosition0Bits", "originalModelPosition1Bits", "originalModelPosition2Bits",
	}, Out: []string{"candidateXBits", "candidateYBits", "candidateZBits", "distanceArgumentBits", "radiusArgumentBits", "placementSucceeded", "modelPosition0Bits", "modelPosition1Bits", "modelPosition2Bits"}, F: monsterGroupRepositionNode}
}

// The fixture has one main member and four optional linked groups, all resolved
// through real lists. An empty navigation hash rejects placement, leaving the
// caller's candidate vector observable in its surviving stack frame. This is
// the candidate sent to the scene, not a successfully installed model position.
// Sources: 0x511050, 0x510cc0, 0x437980, 0x437f30 and 0x463040.
func monsterGroupRepositionNode(in []int64, r Rand) []int64 {
	if len(in) != 34 {
		panic("monster reposition: expected 34 inputs")
	}
	if monsterRepositionError != nil {
		panic(monsterRepositionError)
	}
	for i, v := range in {
		limit := int64(4294967295)
		if i == 0 || i == 7 {
			limit = 255
		}
		if i == 8 || (i >= 26 && i <= 29) {
			limit = 1
		}
		if v < 0 || v > limit {
			panic("monster reposition: field outside fixture width")
		}
	}
	for i := 0; i < 4; i++ {
		if in[10+2*i] > 4 {
			panic("monster reposition: linked group absent from fixture")
		}
	}
	f := func(i int) float64 { return float64(math.Float32frombits(uint32(in[i]))) }
	bits := func(v float64) int64 { return int64(math.Float32bits(float32(v))) }
	round := func(v float64) float64 { return float64(float32(v)) }
	t := monsterRepositionData
	x, y, z := f(2), f(3), f(4)
	distance, radius := in[1], in[0]
	if in[0] != 0 {
		// 0x5110a8: exactly one R(1000), independent of scene acceptance.
		angle80 := new(big.Float).SetPrec(64).SetFloat64(f(5))
		operand := func(v float64) *big.Float { return new(big.Float).SetPrec(64).SetFloat64(v) }
		angle80.Add(angle80, operand(round(float64(r.R(1000))*t.jitter)))
		angle80.Add(angle80, operand(t.offset))
		angle80.Sub(angle80, operand(t.half))
		angle32, _ := angle80.Float32()
		angle := float64(angle32)
		// Preserve x87 64-bit significands before rounding to float32;
		// small positive headings can resolve an otherwise exact halfway sum.
		// 0x517c90: bias, then wrap against the executable's float32 period.
		angle80.SetFloat64(angle)
		angle80.Add(angle80, operand(t.bias))
		angle, _ = angle80.Float64()
		if math.IsInf(angle, 0) || math.IsNaN(angle) {
			panic("monster reposition: heading must be finite")
		}
		for angle80.Cmp(operand(t.zero)) < 0 {
			angle80.Add(angle80, operand(t.period))
		}
		for angle80.Cmp(operand(t.period)) >= 0 {
			angle80.Sub(angle80, operand(t.period))
		}
		angle32, _ = angle80.Float32()
		angle = float64(angle32)
		maxRadius := f(9)
		// 0x511102: four linked slots; unordered comparisons retain the old radius.
		for i := 0; i < 4; i++ {
			if id := in[10+2*i]; id != 0 && f(9+2*int(id)) > maxRadius {
				maxRadius = f(9 + 2*int(id))
			}
		}
		radius = bits(maxRadius)
		d := f(1)
		if d <= t.zero || math.IsNaN(d) {
			d = round(f(6) + maxRadius + t.one)
		}
		distance = bits(d)
		// 0x5111b4..0x5111f5: cosine is saved as double, sine displacement
		// as float32, then each coordinate is saved as float32.
		x = round(x + round(d*math.Sin(angle)))
		z = round(z + d*math.Cos(angle))
	}
	// Empty navigation buckets make both scene placement paths return false.
	// Neither the main model nor any linked model is updated (0x510f09).
	return []int64{bits(x), bits(y), bits(z), distance, radius, 0, in[31], in[32], in[33]}
}
