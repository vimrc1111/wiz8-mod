package calc

// partyStaminaHealRollNode 0x52b160: for each of the eight party slots whose
// presence byte (record array at 0x685178, stride 0x106, first byte) is
// nonzero, roll the dice {Z, N, S} (0x517970) and hand (slot, roll, 0) to the
// stamina heal 0x52b1c0.
//
// 0x52b1c0 is a D4 formula node and is address-stubbed in the emulator case:
// its call count and the arguments of the last call are the outputs. The
// dice are run for real, so every roll goes through diceNode and the rng trace.
//
// Inputs: Z(i16), N(u8), S(u8), presence byte of slots 0..7.
// Outputs: call count, then per call k (0..7, unused cells 0) the slot, roll (i32)
// and notify flag (always 0) passed to 0x52b1c0.
func partyStaminaHealRollNode(in []int64, r Rand) []int64 {
	if len(in) != 11 {
		panic("party stamina heal roll: expected eleven inputs")
	}
	for _, p := range in[3:] {
		if p < 0 || p > 255 {
			panic("party stamina heal roll: presence byte outside 0..255")
		}
	}
	out := make([]int64, 25)
	for who := int64(0); who < 8; who++ {
		if in[3+who] == 0 {
			continue
		}
		k := out[0]
		out[0]++
		out[1+k] = who
		out[9+k] = int64(int32(diceNode(in[:3], r)[0]))
		out[17+k] = 0
	}
	return out
}

var outNames = func() []string {
	n := []string{"healCallCount"}
	for _, p := range []string{"who", "heal", "notify"} {
		for k := 0; k < 8; k++ {
			n = append(n, p+string(rune('0'+k)))
		}
	}
	return n
}()

func init() {
	Nodes["0052b160"] = Node{
		Addr: "0052b160",
		In: []string{"Z", "N", "S", "present0", "present1", "present2", "present3",
			"present4", "present5", "present6", "present7"},
		Out: outNames,
		F:   partyStaminaHealRollNode,
	}
}
