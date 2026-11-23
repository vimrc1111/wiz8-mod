package calc

import "strconv"

func init() {
	in := []string{"awarenessCheckEnabled"}
	// exe 0x4ed206: eight party slots, including both companion slots.
	for i := 0; i < 8; i++ {
		for _, field := range []string{"slotActive", "currentHP", "mainStatus", "effectiveSEN"} {
			in = append(in, field+strconv.Itoa(i))
		}
	}
	Nodes["004ed1c0"] = Node{Addr: "004ed1c0", In: in, Out: []string{"awarenessCheckFailed"}, F: partyAwarenessCheckNode}
}

// partyAwarenessCheckNode reads a mode byte and eight activity/HP/status/SEN
// records. The mode's event name is unconfirmed; the caller consumes a byte
// result (exe 0x4ecfff). No executable data tables are read by this node.
func partyAwarenessCheckNode(in []int64, r Rand) []int64 {
	if len(in) != 33 {
		panic("partyAwarenessCheck: expected mode and eight slot records")
	}
	if in[0] < 0 || in[0] > 255 {
		panic("partyAwarenessCheck: expected u8 mode")
	}
	for i := 1; i < len(in); i += 4 {
		if in[i] < 0 || in[i] > 255 {
			panic("partyAwarenessCheck: expected u8 slot activity")
		}
		for _, v := range in[i+1 : i+4] {
			if v < 0 || v > 4294967295 {
				panic("partyAwarenessCheck: expected u32 HP/status/SEN bits")
			}
		}
	}
	// exe 0x4ed1c8..0x4ed1d3: disabled mode returns zero without RNG.
	if in[0] == 0 {
		return []int64{0}
	}
	for i := 1; i < len(in); i += 4 {
		// exe 0x4ed1e8..0x4ed1f7: first present, nonzero-HP slot with
		// unsigned main status below the literal cutoff 11 is selected.
		if in[i] != 0 && in[i+1] != 0 && in[i+2] < 11 {
			// exe 0x4ed212..0x4ed223: retain the singleton selection R(1),
			// followed by R(100), even at a deterministic SEN threshold.
			r.R(1)
			roll := int64(r.R(100))
			// exe 0x4ed23d..0x4ed24b: unsigned SEN divided by two;
			// equality is failure. A success returns the byte zero.
			if roll < in[i+3]/2 {
				return []int64{0}
			}
			return []int64{1}
		}
	}
	// exe 0x4ed20b: no eligible slot returns one without RNG.
	return []int64{1}
}
