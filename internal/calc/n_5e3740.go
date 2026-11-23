package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Candidate count, minimum level and inclusive level window come from
// 0x5e375d, 0x5e3753..0x5e3758 and 0x5e3775 respectively.
const trapTypeCount = 15
const trapTypeMinimumLevel int32 = 1
const trapTypeLevelWindow int32 = 4

var trapTypeThresholds, trapTypeLoadErr = loadTrapTypeThresholds()

func loadTrapTypeThresholds() ([trapTypeCount]int32, error) {
	var thresholds [trapTypeCount]int32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return thresholds, fmt.Errorf("trap type: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return thresholds, err
	}
	// 0x6504ac holds one signed 32-bit minimum level per candidate type.
	offset, err := peOffset(b, 0x6504ac)
	if err != nil {
		return thresholds, err
	}
	if len(b)-offset < 4*trapTypeCount {
		return thresholds, fmt.Errorf("trap type: truncated level threshold table")
	}
	for i := range thresholds {
		thresholds[i] = int32(binary.LittleEndian.Uint32(b[offset+4*i:]))
	}
	return thresholds, nil
}

// Pure after initialization. The argument selects a valid trigger object;
// only its i32 level at +0x36c is read, and +0x37c receives the chosen type.
// Like the original, this rejection loop has no retry cap: a level with no
// qualifying table entry has no terminating result. The defensive check at
// 0x5e374b only skips an invalid object pointer equal to -0x368.
func trapTypeRollNode(in []int64, r Rand) []int64 {
	// Signed field width at 0x5e3750.
	if len(in) != 1 || in[0] < -2147483648 || in[0] > 2147483647 {
		panic("trap type: expected i32 triggerTrapLevel")
	}
	if trapTypeLoadErr != nil {
		panic(trapTypeLoadErr)
	}
	level := max(trapTypeMinimumLevel, int32(in[0]))
	for {
		candidate := r.R(trapTypeCount)
		threshold := trapTypeThresholds[candidate]
		// Keep the original signed comparisons and 32-bit addition semantics.
		if threshold <= level && threshold+trapTypeLevelWindow >= level {
			return []int64{int64(candidate)}
		}
	}
}

func init() {
	Nodes["005e3740"] = Node{
		Addr: "005e3740", In: []string{"triggerTrapLevel"},
		Out: []string{"triggerTrapType"}, F: trapTypeRollNode,
	}
}
