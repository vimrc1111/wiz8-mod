package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

var trapTimerScale, trapTimerScaleErr = loadTrapTimerScale()

// Load the timer conversion factor at 0x5ec0a8 from the user's executable.
// Initialization keeps the node itself pure, following exe.go's PE mapping.
func loadTrapTimerScale() (float64, error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return 0, fmt.Errorf("trap timer: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	offset, err := peOffset(b, 0x5ec0a8)
	if err != nil {
		return 0, err
	}
	if len(b)-offset < 4 {
		return 0, fmt.Errorf("trap timer: conversion factor truncated")
	}
	scale := float64(math.Float32frombits(binary.LittleEndian.Uint32(b[offset:])))
	if math.IsNaN(scale) || math.IsInf(scale, 0) {
		return 0, fmt.Errorf("trap timer: nonfinite conversion factor")
	}
	return scale, nil
}

// trapSpellDisarmChanceNode replays the chance and observable UI/timer writes.
// The case supplies three valid UI handles and a timer with flags=0 and the
// frozen-clock global enabled; clock kind 1 instead uses the game calendar.
// These clock settings do not affect trap success. The second argument is
// unused, and only the low byte of the third argument signals spell failure.
func trapSpellDisarmChanceNode(in []int64, r Rand) []int64 {
	if len(in) != 12 {
		panic("trap spell disarm: expected 12 inputs")
	}
	for i, v := range in {
		if i < 2 {
			if v < -2147483648 || v > 2147483647 {
				panic("trap spell disarm: expected i32 strength/difficulty")
			}
		} else if v < 0 || v > 4294967295 {
			panic("trap spell disarm: expected u32 parameters and object/global fields")
		}
	}
	// 0x58a944..0x58a956: base 50, strength coefficient 5, difficulty
	// coefficient 6. Each operation wraps at 32 bits before the signed test.
	threshold := int32(0)
	if byte(in[2]) == 0 {
		threshold = 50 + 5*int32(in[0]) - 6*int32(in[1])
	}
	// 0x58a958..0x58a976: R(100), strict comparison, state 7/8.
	state := int64(8)
	if int32(r.R(100)) < threshold {
		state = 7
	}
	if trapTimerScaleErr != nil {
		panic(trapTimerScaleErr)
	}
	// 0x58a99f sets 1.5 seconds; 0x439b9b..0x439bb4 converts the
	// float32 rate times seconds times the exe scale, truncating toward zero.
	// The exact products of these binary float32 factors fit in float64 for
	// the supplied 1.5 seconds, preserving the x87 integer conversion.
	rate := float64(math.Float32frombits(uint32(in[8])))
	duration := math.Trunc(rate * 1.5 * trapTimerScale)
	if math.IsNaN(duration) || math.IsInf(duration, 0) || duration < -9223372036854775808.0 || duration >= 9223372036854775808.0 {
		panic("trap timer: duration outside finite i64 conversion")
	}
	ticks := uint32(int64(duration))
	start := uint32(in[5])
	if in[4] == 1 {
		// 0x439dde..0x439e09: day length 86400000 and final factor 10,
		// both derived from the calendar conversion; additions wrap as u32.
		start = (uint32(in[6])*86400000 + uint32(in[7])) * 10
	}
	// 0x4f2220 disables each of the three UI handles. 0x439d80 restarts
	// the timer with the selected clock and previously converted duration.
	return []int64{state, 0, 0, 0, int64(start), int64(start + ticks), int64(ticks), int64(math.Float32bits(1.5))}
}

func init() {
	Nodes["0058a930"] = Node{
		Addr: "0058a930",
		In:   []string{"spellStrength", "trapDifficulty", "failureParameter", "unusedParameter", "timerClockKind", "frozenClock", "gameDays", "gameTicks", "timerRateBits", "panelEnabled", "childEnabled", "buttonEnabled"},
		Out:  []string{"trapState", "panelEnabled", "childEnabled", "buttonEnabled", "timerStart", "timerEnd", "timerDuration", "timerSecondsBits"},
		F:    trapSpellDisarmChanceNode,
	}
}
