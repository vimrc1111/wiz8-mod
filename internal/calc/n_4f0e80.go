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

var noiseAlertData, noiseAlertDataErr = loadNoiseAlertData()

// Read the probability coefficient, hearing-radius conversion and nearest-node
// tolerances from the player's executable, using exe.go's PE address mapping.
func loadNoiseAlertData() ([4]float64, error) {
	var t [4]float64
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("noise alert: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	for i, va := range []uint32{0x5ec128, 0x5ed4f0, 0x5ebc64, 0x5ec150} {
		o, err := peOffset(b, va)
		if err != nil {
			return t, err
		}
		width := 4
		if i == 3 {
			width = 8
		}
		if len(b)-o < width {
			return t, fmt.Errorf("noise alert: truncated constant at %#x", va)
		}
		if width == 8 {
			t[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[o:]))
		} else {
			t[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[o:])))
		}
		if math.IsNaN(t[i]) || math.IsInf(t[i], 0) {
			return t, fmt.Errorf("noise alert: nonfinite constant at %#x", va)
		}
	}
	return t, nil
}

// x87 arithmetic uses a 64-bit significand until an explicit float32 store.
func noiseFloat(v float64) *big.Float { return new(big.Float).SetPrec(64).SetFloat64(v) }
func noiseDistance(a, b [3]float64) *big.Float {
	squares := [3]*big.Float{}
	for i := range a {
		d := noiseFloat(a[i])
		d.Sub(d, noiseFloat(b[i]))
		squares[i] = noiseFloat(0).Mul(d, d)
	}
	sum := noiseFloat(0).Add(squares[2], squares[1])
	sum.Add(sum, squares[0])
	return sum.Sqrt(sum)
}
func noiseTrunc(v *big.Float) int32 {
	n, _ := v.Int(nil)
	if !n.IsInt64() {
		panic("noise alert: distance outside CRT conversion range")
	}
	return int32(n.Int64())
}

// The fixture contains one active group and a separate leader group, with
// loaded templates and empty member lists. Alert dispatch therefore returns
// without setting the group flag (0x50f8a8..0x50f8b8). Navigation has two
// coincident nodes in one spatial cell; either hash bucket may be empty.
// Inputs retain raw float32 coordinate bits and both monsters' prior memories.
func noiseAlertProbabilityNode(in []int64, r Rand) []int64 {
	if len(in) != 33 {
		panic("noise alert: expected 33 inputs")
	}
	if noiseAlertDataErr != nil {
		panic(noiseAlertDataErr)
	}
	for i, v := range in {
		signed := i == 2 || i == 3 || i >= 16 && i <= 25 || i == 28 || i == 29
		if signed {
			if v < -2147483648 || v > 2147483647 {
				panic("noise alert: expected i32 input")
			}
		} else if v < 0 || v > 4294967295 {
			panic("noise alert: expected u32 input")
		}
	}
	for _, i := range []int{0, 4, 5, 7, 8, 9, 26, 30, 31, 32} {
		if in[i] > 255 {
			panic("noise alert: expected byte field")
		}
	}
	if in[1] > 1 || in[6] != 0 && in[6] != 102 || in[28] < -1 || in[28] > 0 || in[29] < -1 || in[29] > 0 {
		panic("noise alert: group or navigation record absent from fixture")
	}
	f := func(i int) float64 {
		v := float64(math.Float32frombits(uint32(in[i])))
		if math.IsNaN(v) || math.IsInf(v, 0) {
			panic("noise alert: expected finite coordinate or radius")
		}
		return v
	}
	out := append([]int64{}, in[16:26]...)
	out = append(out, in[5], 0)
	if in[0] != 0 || in[1] == 0 || in[7] != 0 {
		return out
	}
	mode := int32(in[3])
	if mode == 1 {
		if in[4] != 0 && in[5] != 0 {
			return out
		}
	} else if mode == 0 && in[8] != 1 {
		return out
	}
	event := [3]float64{f(10), f(11), f(12)}
	position := [3]float64{f(13), f(14), f(15)}
	intensity := int32(in[2])
	residual := intensity - noiseTrunc(noiseDistance(position, event))
	if residual <= 0 {
		return out
	}
	// 0x4f0f9e: exactly one R(100); equality fails the probability test.
	roll := r.R(100)
	threshold := noiseFloat(float64(residual))
	threshold.Mul(threshold, noiseFloat(noiseAlertData[0]))
	threshold.Add(threshold, noiseFloat(float64(in[9])))
	if threshold.Cmp(noiseFloat(float64(roll))) <= 0 {
		return out
	}
	target := 0
	if in[6] != 0 {
		if in[26] != 0 {
			return out
		}
		target = 5
	}
	if mode == 1 && in[4] != 0 {
		// 0x4604ee..0x46050b stores the positive float32 intensity as the
		// search limit; both returns restore 0x501502f9 (0x4605c9/0x46062b).
		out[11] = 0x501502f9
		if !noiseNavigation(position, event, in[28], in[29]) {
			return out
		}
		radius := noiseFloat(f(27))
		radius.Mul(radius, noiseFloat(noiseAlertData[1]))
		// 0x4f1076..0x4f1082 scales the truncated hearing radius by 1000,
		// with signed 32-bit wrap; the path length has a float32 store.
		limit := noiseTrunc(radius) * 1000
		length, _ := noiseDistance(position, event).Float32()
		if float64(limit) < float64(length) {
			return out
		}
	} else if residual <= int32(out[target+1]) {
		return out
	}
	out[target] = int64(intensity)
	out[target+1] = int64(residual)
	for i := 0; i < 3; i++ {
		out[target+2+i] = int64(int32(in[10+i]))
	}
	return out
}

// 0x45b120 selects the first coincident nearest node. The fixture's navigation
// lower and upper bounds both equal -1000, so smoothing cannot replace the
// endpoint segment. Spatial grid origin=-1000, width=10000 (case setup).
func noiseNavigation(a, b [3]float64, navBucket, sceneBucket int64) bool {
	if navBucket < 0 || sceneBucket < 0 {
		return false
	}
	for _, p := range [][3]float64{a, b} {
		for _, coordinate := range p {
			if int64((coordinate+1000)/10000) != 0 {
				return false
			}
		}
		d := noiseDistance(p, [3]float64{})
		if float64(noiseTrunc(d)) >= noiseAlertData[2] {
			return false
		}
		// The nearest-node horizontal acceptance omits Y (0x45b283).
		horizontal := noiseDistance([3]float64{p[0], 0, p[2]}, [3]float64{})
		if horizontal.Cmp(noiseFloat(noiseAlertData[3])) >= 0 {
			return false
		}
	}
	return true
}

func init() {
	Nodes["004f0e80"] = Node{
		Addr: "004f0e80",
		In:   []string{"simulationPaused", "groupCount", "intensity", "mode", "combat", "alreadyAlert", "leaderID", "monsterDisabled", "noiseEligibility", "perception", "eventX", "eventY", "eventZ", "modelX", "modelY", "modelZ", "monster_previousIntensity", "monster_previousResidual", "monster_heardX", "monster_heardY", "monster_heardZ", "linkedMonster_previousIntensity", "linkedMonster_previousResidual", "linkedMonster_heardX", "linkedMonster_heardY", "linkedMonster_heardZ", "leaderDisabled", "hearingRadius", "navigationBucket", "sceneBucket", "alertSuppressed", "partyAlertBlocked", "templateAlertBlocked"},
		Out:  []string{"monster_intensity", "monster_residual", "monster_heardX", "monster_heardY", "monster_heardZ", "linkedMonster_intensity", "linkedMonster_residual", "linkedMonster_heardX", "linkedMonster_heardY", "linkedMonster_heardZ", "alert", "navigationLimit"},
		F:    noiseAlertProbabilityNode,
	}
}
