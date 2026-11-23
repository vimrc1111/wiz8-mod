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

var respawnFactors, respawnFactorsErr = loadRespawnFactors()

// Read the jitter fraction (0x5ec040), seconds conversion (0x5ec0a8),
// and positive-duration threshold (0x5ebb34) from the user's executable.
// Loading once outside F keeps the node pure; PE mapping follows exe.go.
func loadRespawnFactors() ([3]float64, error) {
	var result [3]float64
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return result, fmt.Errorf("respawn interval: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	for i, addr := range []uint32{0x5ec040, 0x5ec0a8, 0x5ebb34} {
		offset, err := peOffset(data, addr)
		if err != nil {
			return result, err
		}
		if len(data)-offset < 4 {
			return result, fmt.Errorf("respawn interval: factor %#x truncated", addr)
		}
		result[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[offset:])))
		if math.IsNaN(result[i]) || math.IsInf(result[i], 0) {
			return result, fmt.Errorf("respawn interval: nonfinite factor %#x", addr)
		}
	}
	return result, nil
}

// The timer's three float32 factors can exceed binary64 precision. Match
// x87's 64-bit significand at each product before _ftol truncation.
func respawnDuration(rate, seconds, scale float64) uint32 {
	value := new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).SetFloat64(rate)
	value.Mul(value, new(big.Float).SetPrec(64).SetFloat64(seconds))
	value.Mul(value, new(big.Float).SetPrec(64).SetFloat64(scale))
	integer, _ := value.Int(nil)
	if integer == nil || !integer.IsInt64() {
		panic("respawn interval: timer conversion outside i64")
	}
	return uint32(integer.Int64())
}

// Inputs represent a valid generator and timer. The case allocates at a fixed
// arena address, with an initialized shared clock whose reads return tick 777.
// 0x48b442 allocates 40 bytes; 0x43a4e0 constructs rate 1 and clock kind 0.
func encounterRespawnIntervalNode(in []int64, r Rand) []int64 {
	if len(in) != 14 {
		panic("respawn interval: expected 14 inputs")
	}
	for i, value := range in {
		switch i {
		case 1, 2:
			if value < -32768 || value > 32767 {
				panic("respawn interval: expected i16 interval")
			}
		case 5:
			if value < 0 || value > 65535 {
				panic("respawn interval: expected u16 timer flags")
			}
		case 8:
			if value < 0 || value > 255 {
				panic("respawn interval: expected u8 frozen clock flag")
			}
		default:
			if value < 0 || value > 4294967295 {
				panic("respawn interval: expected u32 field")
			}
		}
	}
	if respawnFactorsErr != nil {
		panic(respawnFactorsErr)
	}
	kind, flags := uint32(in[4]), uint16(in[5])
	secondsBits, rateBits := uint32(in[6]), uint32(in[7])
	clockUsers := uint32(in[13]) // 0x6598cc; constructor increments it.
	pointer := uint32(in[3])
	if pointer == 0 {
		pointer = 0x20000400 // Case arena base 0x20000000 + timer offset 1024.
		kind, flags = 0, 0   // 0x439814..0x43981b; 0x48b496 clears one-shot bit.
		secondsBits, rateBits = math.Float32bits(1), math.Float32bits(1)
		clockUsers++ // 0x43995b..0x43995c registers the new timer.
	}
	base := in[1]
	if uint32(in[0])&8 != 0 {
		base = in[2]
	} // 0x48b49c..0x48b4be, flag bit 3.
	jitter := float64(base) * respawnFactors[0]
	// 0x48b4d0 stores float32 but leaves the extended value for _ftol.
	width := uint32(int64(math.Trunc(jitter))*2 + 1)
	random := uint32(r.R(int(width)))
	seconds := float32(float64(base) + float64(random) - float64(float32(jitter)))
	// 0x439b85..0x439b98 preserves old seconds when the new value is <= 0.
	if float64(seconds) > respawnFactors[2] {
		secondsBits = math.Float32bits(seconds)
	}
	rate := float64(math.Float32frombits(rateBits))
	oldSeconds := float64(math.Float32frombits(secondsBits))
	if math.IsNaN(rate) || math.IsInf(rate, 0) || math.IsNaN(oldSeconds) || math.IsInf(oldSeconds, 0) {
		panic("respawn interval: nonfinite timer field")
	}
	duration := respawnDuration(rate, oldSeconds, respawnFactors[1])
	// 0x43a530 restarts: kind 1 uses the calendar, flag bit 0 uses raw
	// clock reads, otherwise the frozen clock or reads minus pause offset.
	start := uint32(777) // Case getUTime return, not a game rule constant.
	if kind == 1 {
		// 0x43a592..0x43a5bd: day length 86400000, final factor 10.
		start = (uint32(in[11])*86400000 + uint32(in[12])) * 10
	} else if flags&1 == 0 {
		if in[8] != 0 {
			start = uint32(in[9])
		} else {
			start -= uint32(in[10])
		}
	}
	return []int64{int64(pointer), int64(kind), int64(flags), int64(start), int64(start + duration), int64(duration), int64(secondsBits), int64(rateBits), 0, int64(clockUsers)}
}

func init() {
	Nodes["0048b420"] = Node{
		Addr: "0048b420",
		In:   []string{"generatorFlags", "localInterval", "defaultInterval", "timerPointer", "timerClockKind", "timerFlags", "timerSecondsBits", "timerRateBits", "frozenClockEnabled", "frozenClock", "clockOffset", "gameDays", "gameTicks", "clockUsers"},
		Out:  []string{"timerPointer", "timerClockKind", "timerFlags", "timerStart", "timerEnd", "timerDuration", "timerSecondsBits", "timerRateBits", "timerExpired", "clockUsers"},
		F:    encounterRespawnIntervalNode,
	}
}
