package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// The shared buff slot order lives at 0x616d84 (12 u32 entries).
// Read the user's executable using exe.go's PE mapping; never embed the table.
var partyBuffSlots, partyBuffSlotsErr = loadPartyBuffSlots()

func loadPartyBuffSlots() ([12]uint32, error) {
	var slots [12]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return slots, fmt.Errorf("party buff: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return slots, err
	}
	off, err := peOffset(b, 0x616d84)
	if err != nil {
		return slots, err
	}
	if len(b)-off < 48 {
		return slots, fmt.Errorf("party buff: truncated slot table")
	}
	for i := range slots {
		slots[i] = binary.LittleEndian.Uint32(b[off+4*i:])
	}
	return slots, nil
}

func init() {
	in := []string{"spell", "power", "bonusPercent", "baseDuration", "durationPerPower", "targetCount", "applyParty", "applied"}
	var out []string
	for _, name := range []string{"party", "monster0", "monster1"} {
		for j := 0; j < 12; j++ {
			for _, f := range []string{"active", "spell", "power", "multiplier", "duration"} {
				s := fmt.Sprintf("%s[%d].%s", name, j, f)
				in = append(in, s)
				out = append(out, s)
			}
		}
	}
	out = append(out, "applied", "refreshFlags")
	in = append(in, "screen", "refreshFlags")
	for _, name := range []string{"partyModifiers", "monster0Modifiers", "monster1Modifiers"} {
		for j := 0; j < 103; j++ {
			s := fmt.Sprintf("%s[%d]", name, j)
			in = append(in, s)
			out = append(out, s)
		}
	}
	Nodes["0054cbf0"] = Node{Addr: "0054cbf0", In: in, Out: out, F: partyBuffDurationNode}
}

// The case has two distinct, loaded monster records and no present characters,
// combat auras, or pre-existing monster status/condition modifiers. The output
// interface covers all shared slots and modifier bytes, including skipped records.
// Render resources use case import stubs; visual list allocation is outside the
// rule outputs. Derived monster attack caches from 0x502c50/0x4e5d00 likewise
// lie outside this interface. The original children still execute in the emulator.
func partyBuffDurationNode(in []int64, r Rand) []int64 {
	if len(in) != 499 {
		panic("party buff: expected 499 fields")
	}
	if partyBuffSlotsErr != nil {
		panic(partyBuffSlotsErr)
	}
	for _, v := range in {
		if v < 0 || v > 0xffffffff {
			panic("party buff: expected u32 inputs")
		}
	}
	count := int32(in[5])
	if count > 2 {
		panic("party buff: fixture has two monster targets")
	}
	slot := -1
	for i, spell := range partyBuffSlots {
		if spell == uint32(in[0]) {
			slot = i
			break
		}
	}
	if slot < 0 {
		panic("party buff: invalid shared spell")
	}
	// 0x54cbfa..0x54cc61: duration arithmetic wraps at every u32 operation.
	// Sentinel 9999 (0x54cc11) bypasses both RNG and percentage adjustment.
	duration := uint32(in[3]) + uint32(in[4])*uint32(in[1])
	if duration != 9999 {
		duration++
		roll := r.R(4) // 0x54cc22.
		if (roll == 0 && duration > 1) || (roll == 1 && duration <= 2) {
			duration++
		}
		// 0x517f90: unsigned floor of the wrapped product divided by 100.
		duration += (duration * uint32(in[2])) / 100
	}
	out := append([]int64(nil), in[8:188]...)
	applied, refresh := in[7], uint32(in[189])
	caches := append([]int64(nil), in[190:]...)
	update := func(record int) {
		off := record*60 + 5*slot
		// 0x54cd54..0x54cd81 / 0x49f3ba: a new or replaced monster
		// visual consumes R(20), even though its phase is not a rule output.
		if record != 0 && (out[off] == 0 || out[off+1] != in[0]) {
			r.R(20)
		}
		out[off], out[off+1], out[off+2], out[off+4] = 1, in[0], in[1], int64(duration)
		applied = 1
		// Compose the existing 0x50edc0 shared-spell dispatch through 0x50e700's
		// adapter with no characters/combat. 0x50e8c0 uses the same dispatch after
		// clearing its buffer; fixed zero status/condition records contribute zero.
		rebuild := make([]int64, 2620)
		for j := 0; j < 12; j++ {
			rec := out[record*60+j*5 : record*60+j*5+5]
			rebuild[1+j*4], rebuild[2+j*4], rebuild[3+j*4], rebuild[4+j*4] = rec[0], rec[1], int64(uint8(rec[2])), rec[3]
		}
		copy(caches[record*103:(record+1)*103], partyEffectRebuildNode(rebuild, r)[:103])
	}
	if in[6] != 0 {
		update(0)
		// 0x562a50: screen mode 7 and the fixture's non-null display object.
		if in[188] == 7 {
			refresh |= 0x800100
		}
	}
	for i := int32(0); i < count; i++ {
		update(int(i) + 1)
	}
	out = append(out, applied, int64(refresh))
	return append(out, caches...)
}
