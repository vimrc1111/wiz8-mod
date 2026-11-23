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

type confusedLocationConstants struct{ scale, center, jitter, invalidHeight, zero float64 }

var confusedLocationData, confusedLocationErr = loadConfusedLocationConstants()

// Load scalar operands from the player's executable before entering the pure
// rule. PE mapping follows exe.go; no executable bytes or tables are embedded.
func loadConfusedLocationConstants() (confusedLocationConstants, error) {
	var t confusedLocationConstants
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("confused location: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	for _, c := range []struct {
		va   uint32
		size int
		dst  *float64
	}{
		{0x5ebc40, 4, &t.scale}, {0x5ec3d8, 8, &t.center},
		{0x5ec8d0, 8, &t.jitter}, {0x5ed7d0, 4, &t.invalidHeight}, {0x5ebb40, 8, &t.zero},
	} {
		off, err := peOffset(b, c.va)
		if err != nil {
			return t, err
		}
		if off > len(b)-c.size {
			return t, fmt.Errorf("confused location: truncated constant %#x", c.va)
		}
		if c.size == 8 {
			*c.dst = math.Float64frombits(binary.LittleEndian.Uint64(b[off:]))
		} else {
			*c.dst = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[off:])))
		}
	}
	return t, nil
}

// Fixture: 150 indexed spell records (0x96), a party shape and one monster resolved
// by the original ID list. The collision and navigation indices contain no
// geometry. The ray child executes and leaves the candidate unchanged; the
// height child returns candidate Y (0x434f20, 0x420c17). Positions are independent inputs.
func confusedSpellRandomLocationNode(in []int64, r Rand) []int64 {
	if len(in) != 9 {
		panic("confused location: expected 9 inputs")
	}
	if confusedLocationErr != nil {
		panic(confusedLocationErr)
	}
	if in[0] < 0 || in[0] > 149 || in[1] < -1 || in[1] > 3 || (in[2] != 1 && in[2] != 2) {
		panic("confused location: invalid spell record, range or source kind")
	}
	t := confusedLocationData
	// Range lengths are literal returns at 0x51a9b7..0x51a9d0.
	lengths := [5]float64{0, 2, 4, 25, 50}
	distance := lengths[in[1]+1] * t.scale
	position := [3]float64{}
	start := 3
	if in[2] == 2 {
		start = 6
	}
	for i := range position {
		if in[start+i] < 0 || in[start+i] > 4294967295 {
			panic("confused location: position exceeds f32 storage")
		}
		position[i] = float64(math.Float32frombits(uint32(in[start+i])))
		if math.IsNaN(position[i]) || math.IsInf(position[i], 0) {
			panic("confused location: position must be finite")
		}
	}
	// Original intermediate arithmetic has x87's 64-bit significand. Explicit
	// f64 and f32 stores matter for tiny offsets and halfway rounding.
	ext := func(v float64) *big.Float { return new(big.Float).SetPrec(64).SetFloat64(v) }
	round32 := func(v *big.Float) float64 { f, _ := v.Float32(); return float64(f) }
	candidate := [3]float64{}
	for attempt := 0; attempt < 5; attempt++ { // retry cap at 0x4ff114.
		// Bounds 2001,2001,1001 and coordinate order X,Z,Y: 0x4ff03c..0x4ff09d.
		draws := [3]int{r.R(2001), 0, r.R(2001)}
		draws[1] = r.R(1001)
		for _, i := range []int{0, 2, 1} {
			offset := ext(float64(draws[i]))
			if i != 1 {
				offset.Sub(offset, ext(t.center))
			}
			offset.Mul(offset, ext(distance))
			offset.Mul(offset, ext(t.jitter))
			offset.Add(offset, ext(position[i]))
			if i != 1 {
				v, _ := offset.Float64()
				offset = ext(v)
			} // 0x4ff06b/0x4ff099.
			candidate[i] = round32(offset)
		}
		if candidate[1] != t.invalidHeight {
			break
		}
	}
	delta := [3]float64{}
	for i := range delta {
		delta[i] = round32(new(big.Float).SetPrec(64).Sub(ext(position[i]), ext(candidate[i])))
	}
	square := func(v float64) *big.Float { return new(big.Float).SetPrec(64).Mul(ext(v), ext(v)) }
	// Sum order Z squared, X squared, Y squared: 0x4ff151..0x4ff173.
	squared := square(delta[2])
	squared.Add(squared, square(delta[0]))
	squared.Add(squared, square(delta[1]))
	length := new(big.Float).SetPrec(64).Sqrt(squared)
	if length.Cmp(ext(distance)) > 0 {
		factor := ext(1)
		if squared.Cmp(ext(t.zero)) != 0 {
			factor.Quo(ext(distance), length)
		}
		// The original adds the scaled source-minus-candidate vector to the source;
		// it does not interpolate towards the candidate (0x4ff11d..0x4ff1eb).
		for i := range candidate {
			scaled := new(big.Float).SetPrec(64).Mul(ext(delta[i]), factor)
			if i != 2 {
				scaled = ext(round32(scaled))
			} // Z remains extended until addition.
			scaled.Add(scaled, ext(position[i]))
			candidate[i] = round32(scaled)
		}
	}
	return []int64{int64(math.Float32bits(float32(candidate[0]))), int64(math.Float32bits(float32(candidate[1]))), int64(math.Float32bits(float32(candidate[2])))}
}

func init() {
	Nodes["004fef90"] = Node{Addr: "004fef90", In: []string{"spellIndex", "spellRange", "sourceKind", "partyXBits", "partyYBits", "partyZBits", "monsterXBits", "monsterYBits", "monsterZBits"}, Out: []string{"targetXBits", "targetYBits", "targetZBits"}, F: confusedSpellRandomLocationNode}
}
