package calc

import "fmt"

// party_stamina_recovery 0x504670(ticks): sets the game mode global 0x687509 from
// the engine flag word (pointer 0x652dac; null: mode 3; bit 8 set: 1; else bit 9
// set: 0; else 3), then for each of the eight party slots whose presence byte
// (*0x685178 + 0x106*i) is non-zero decides whether 0x504730(i, ticks) runs.
// A member is skipped when state +0xb01 >= 0x12 (unsigned) and either +0xa49 is
// non-zero or the dependency flag 0x5248a0(i, 1) (byte +0x1830) is zero; and when
// byte 0x6875a3 is non-zero but 0x520f90(C, 0x1e5, 0, 0, 0) finds no item 0x1e5
// in the 12 slots at +0xf5d+12k (mode 0: the +0x1029 slots are not scanned).
// Emu address-stubs the D4 callee 0x504730: its outputs are the call count and
// each call's (who, ticks); nothing later in the node reads its writes or eax.
//
// Fixture cap (calc panics past it): the engine pointer is 0 or the arena
// object address.
const (
	psrEngAddr = 0x20000000 + 61440
	psrMembers = 8
	psrPerMem  = 3 + 12 // state, exempt flag, dependency flag, 12 item slots
	psrFixed   = 4 + psrMembers
	psrIn      = psrFixed + psrMembers*psrPerMem
	psrItem    = 0x1e5
)

func init() {
	in := []string{"ticks", "engineFlagsPointer", "engineFlags", "itemCheckEnabled"}
	for i := 0; i < psrMembers; i++ {
		in = append(in, fmt.Sprintf("member%dPresent", i))
	}
	for i := 0; i < psrMembers; i++ {
		in = append(in, fmt.Sprintf("member%dState", i), fmt.Sprintf("member%dStateExemptFlag", i), fmt.Sprintf("member%dDependencyFlag", i))
		for k := 0; k < 12; k++ {
			in = append(in, fmt.Sprintf("member%dItem%d", i, k))
		}
	}
	out := []string{"gameMode", "recoveryCalls"}
	for k := 0; k < psrMembers; k++ {
		out = append(out, fmt.Sprintf("call%dWho", k))
	}
	for k := 0; k < psrMembers; k++ {
		out = append(out, fmt.Sprintf("call%dTicks", k))
	}
	Nodes["00504670"] = Node{Addr: "00504670", In: in, Out: out, F: partyStaminaRecoveryNode}
}

func partyStaminaRecoveryNode(in []int64, r Rand) []int64 {
	if len(in) != psrIn {
		panic(fmt.Sprintf("party stamina recovery: expected %d inputs, got %d", psrIn, len(in)))
	}
	if p := in[1]; p != 0 && p != psrEngAddr {
		panic("party stamina recovery: engine flag pointer is neither null nor the fixture object")
	}
	mode := int64(3)
	if in[1] != 0 {
		flags := uint32(in[2])
		switch {
		case flags>>8&1 != 0:
			mode = 1
		case flags>>9&1 != 0:
			mode = 0
		}
	}
	out := make([]int64, 2+2*psrMembers)
	out[0] = mode
	ticks := int64(uint32(in[0]))
	for i := 0; i < psrMembers; i++ {
		if in[4+i]&0xff == 0 {
			continue
		}
		m := in[psrFixed+i*psrPerMem:]
		if uint32(m[0]) >= 0x12 && (uint32(m[1]) != 0 || m[2]&0xff == 0) {
			continue
		}
		if in[3]&0xff != 0 {
			found := false
			for k := 0; k < 12; k++ {
				found = found || uint32(m[3+k]) == psrItem
			}
			if !found {
				continue
			}
		}
		n := out[1]
		out[2+n], out[2+psrMembers+n] = int64(i), ticks
		out[1]++
	}
	return out
}
