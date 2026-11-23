package calc

import "fmt"

func init() {
	in := []string{"characterIndex", "drainAmount", "showPrompt", "currentHP"}
	var sp []string
	// Eight party slots (0x52b57d), six realms (0x52b607).
	for c := 0; c < 8; c++ {
		for realm := 0; realm < 6; realm++ {
			sp = append(sp, fmt.Sprintf("character%dCurrentSP%d", c, realm))
		}
	}
	in = append(in, sp...)
	in = append(in, "previewOnly", "uiMode", "characterRefreshFlags", "combatRefreshFlags", "combatUIPointer")
	out := append(append([]string{}, sp...), "characterRefreshFlags", "combatRefreshFlags")
	Nodes["0052b590"] = Node{Addr: "0052b590", In: in, Out: out, F: characterRandomRealmSpDrainNode}
}

// characterRandomRealmSpDrainNode composes SP subtraction with UI invalidation.
// Text-only children format messages but do not alter SP or refresh flags.
// The realm-name table at 0x61e518 belongs only to that presentation path;
// the emulator reads it from the user's executable without a calc-side copy.
func characterRandomRealmSpDrainNode(in []int64, r Rand) []int64 {
	if len(in) != 57 || in[0] < 0 || in[0] > 7 {
		panic("character_random_realm_sp_drain: expected 57 inputs and party slot 0..7")
	}
	for _, i := range []int{1, 54, 55, 56} {
		if in[i] < 0 || in[i] > 4294967295 {
			panic("character_random_realm_sp_drain: expected unsigned DWORD")
		}
	}
	for _, i := range []int{2, 52} {
		if in[i] < 0 || in[i] > 255 {
			panic("character_random_realm_sp_drain: expected byte flag")
		}
	}
	for i := 3; i <= 53; i++ {
		if i == 52 {
			continue
		}
		if in[i] < -2147483648 || in[i] > 2147483647 {
			panic("character_random_realm_sp_drain: expected signed DWORD representation")
		}
	}
	out := append([]int64{}, in[4:52]...)
	characterFlags, combatFlags := uint32(in[54]), uint32(in[55])
	if in[3] != 0 && in[52] == 0 {
		remaining := uint32(in[1])
		// 0x52b5ed sets the 50-attempt cap; empty/negative realms consume attempts.
		for attempts := 0; attempts < 50 && remaining != 0; attempts++ {
			slot := int(in[0])*6 + r.R(6)
			current := int32(out[slot])
			if current <= 0 {
				continue
			}
			// 0x52b62b compares the demand and positive stock as unsigned.
			drain := min(remaining, uint32(current))
			out[slot] = int64(int32(uint32(current) - drain))
			remaining -= drain
			// Composed 0x55ee30/0x562a50: UI modes 6/7, dirty bit 0x100
			// (0x55ee57), or selected character bit (0x55ee46/0x562a6c).
			if in[53] == 6 {
				characterFlags |= 0x100
			} else if in[53] == 7 && in[56] != 0 {
				combatFlags |= uint32(1) << uint(in[0])
			}
		}
	}
	return append(out, int64(characterFlags), int64(combatFlags))
}
