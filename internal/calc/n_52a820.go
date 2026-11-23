package calc

// partyDamageRollNode 0x52a820 party_damage_roll(dice*, kill, a4) (build/wiz8.asm:404752..):
// for each of the eight party slots whose presence byte (array *0x685178, stride 0x106,
// first byte) is nonzero and whose character state dword at +0xb01 (array *0x685174,
// stride 0x1862, unsigned) is below 0x12, roll the dice {Z, N, S} (0x517970) and hand
// (slot, roll, 0, a4, 0, kill, 0) to character_damage_apply 0x52a890. a4 and kill are
// passed through as whole 32-bit words.
//
// 0x52a890 is a D4 formula node, address-stubbed in the emulator case: the caller
// ignores its return value (stub ret 0 here; ret 0x5a5a5a5a in a scratch copy of the
// case gave the same diff=0), so no Emu/F split is needed. The case outputs are the
// call count and every call's seven arguments. The dice run for real, so every roll
// goes through diceNode and the rng trace.
//
// Inputs: Z(i16), N(u8), S(u8), kill(u32), a4(u32), presence byte of slots 0..7,
// state dword +0xb01 of slots 0..7 (u32).
// Outputs: call count, then for each of the seven arguments the values of calls
// 0..7 (unused cells 0; the roll is signed 32-bit).
func partyDamageRollNode(in []int64, r Rand) []int64 {
	if len(in) != 21 {
		panic("party damage roll: expected 21 inputs")
	}
	if in[3] < 0 || in[3] > 0xffffffff || in[4] < 0 || in[4] > 0xffffffff {
		panic("party damage roll: kill/a4 outside u32")
	}
	for i := 5; i < 13; i++ {
		if in[i] < 0 || in[i] > 255 {
			panic("party damage roll: presence byte outside 0..255")
		}
	}
	for i := 13; i < 21; i++ {
		if in[i] < 0 || in[i] > 0xffffffff {
			panic("party damage roll: state dword outside u32")
		}
	}
	out := make([]int64, 1+7*8)
	for who := int64(0); who < 8; who++ {
		if in[5+who] == 0 || in[13+who] >= 0x12 {
			continue
		}
		k := out[0]
		out[0]++
		out[1+k] = who
		out[9+k] = int64(int32(diceNode(in[:3], r)[0]))
		out[17+k] = 0
		out[25+k] = in[4]
		out[33+k] = 0
		out[41+k] = in[3]
		out[49+k] = 0
	}
	return out
}

func init() {
	n := []string{"damageCallCount"}
	for _, p := range []string{"who", "dmg", "a3", "a4", "a5", "kill", "a7"} {
		for k := 0; k < 8; k++ {
			n = append(n, p+string(rune('0'+k)))
		}
	}
	in := []string{"Z", "N", "S", "kill", "a4"}
	for k := 0; k < 8; k++ {
		in = append(in, "present"+string(rune('0'+k)))
	}
	for k := 0; k < 8; k++ {
		in = append(in, "state"+string(rune('0'+k)))
	}
	Nodes["0052a820"] = Node{Addr: "0052a820", In: in, Out: n, F: partyDamageRollNode}
}
