package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

// movementRetentionScale is the float32 at exe 0x5ec3b8, read from the
// player's own Wiz8.exe (peOffset, exe.go); it is not embedded.
var movementRetentionScale, movementRetentionScaleErr = loadMovementRetentionScale()

func loadMovementRetentionScale() (float64, error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return 0, fmt.Errorf("movement retention: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	offset, err := peOffset(data, 0x5ec3b8)
	if err != nil {
		return 0, err
	}
	if len(data)-offset < 4 {
		return 0, fmt.Errorf("movement retention: scale truncated")
	}
	v := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset:])))
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("movement retention: nonfinite scale")
	}
	return v, nil
}

const movementRetentionMembers = 8 // exe 0x4f07d9: ebp stride 0x1862 up to 0xc310

func init() {
	in := []string{"remainingMovePct", "movementEventMode", "partyActionMode", "currentPhase", "partyFlags"}
	out := []string{"partyFlags"}
	for i := 0; i < movementRetentionMembers; i++ {
		for _, n := range []string{"present", "hp", "status", "invalid", "actionPhase", "previousPhase", "queueSlot"} {
			in = append(in, fmt.Sprintf("%s%d", n, i))
		}
		for _, n := range []string{"actionPhase", "invalid", "previousPhase", "queueSlot"} {
			out = append(out, fmt.Sprintf("%s%d", n, i))
		}
	}
	Nodes["004f06b0"] = Node{Addr: "004f06b0", In: in, Out: out, F: movementActionRetentionChanceNode}
}

// movementActionRetentionChanceNode: after movement, each present member that
// is alive (hp != 0) with main status < 15 keeps its action with probability
// limit% (R(100) < limit, then the action is rebased by 0x4eceb0), otherwise
// the action is voided. Inputs: 5 shared values, then per member present(u8),
// hp(u32), status(u32), invalid(u8), actionPhase, previousPhase, queueSlot(u32).
// remainingMovePct must be 0..100 (exe 0x4f06ce/0x4f06f2 assert otherwise).
func movementActionRetentionChanceNode(in []int64, r Rand) []int64 {
	if len(in) != 5+7*movementRetentionMembers {
		panic("movementActionRetentionChance: wrong input count")
	}
	if movementRetentionScaleErr != nil {
		panic(movementRetentionScaleErr)
	}
	pct := in[0]
	if pct < 0 || pct > 100 {
		panic("movementActionRetentionChance: remainingMovePct outside 0..100 (srAssertFail)")
	}
	// exe 0x4f071f..0x4f0724: miss = 100 - pct; mode 2 scales it by
	// 0x5ec3b8 and truncates (_ftol), 0x4f0742 compares unsigned.
	miss := uint32(100 - pct)
	if in[2] == 2 {
		miss = uint32(int64(float64(miss) * movementRetentionScale))
	}
	limit := int64(0)
	if miss < 100 {
		limit = int64(100 - miss) // exe 0x4f074d..0x4f0754
	}
	out := make([]int64, 1, 1+4*movementRetentionMembers)
	out[0] = in[4]
	for i := 0; i < movementRetentionMembers; i++ {
		m := in[5+7*i : 12+7*i]
		present, hp, status, invalid := m[0], m[1], m[2], m[3]
		phase, previous, slot := m[4], m[5], m[6]
		// exe 0x4f0762..0x4f0778: present byte, hp != 0, status < 15 (all unsigned).
		if present != 0 && hp != 0 && status < 15 {
			keep := false
			if invalid == 0 {
				keep = int64(r.R(100)) < limit // exe 0x4f0794..0x4f07a2
			}
			if keep {
				res := actionPhaseRebaseNode([]int64{in[2], in[3], phase, previous}, r)
				phase, previous = res[0], res[1]
			} else { // exe 0x4f07af..0x4f07bf
				invalid, phase, slot = 1, 0, 0xffffffff
			}
		}
		out = append(out, phase, invalid, previous, slot)
	}
	// exe 0x562a50: in movement mode 7 the shared flags gain 0x1000ff.
	if in[1] == 7 {
		out[0] = int64(uint32(out[0]) | 0x1000ff)
	}
	return out
}
