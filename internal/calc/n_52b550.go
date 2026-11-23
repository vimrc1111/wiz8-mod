package calc

import "fmt"

func init() {
	in := []string{"drainAmount", "showPrompt"}
	// 0x52b576/0x52b57d/0x52b583: eight party slots, portrait stride 0x106.
	for c := 0; c < 8; c++ {
		in = append(in, fmt.Sprintf("character%dActive", c))
	}
	for c := 0; c < 8; c++ {
		in = append(in, fmt.Sprintf("character%dCurrentHP", c))
	}
	var sp []string
	// 0x52b607: six realms per character, composed through 0x52b590.
	for c := 0; c < 8; c++ {
		for realm := 0; realm < 6; realm++ {
			sp = append(sp, fmt.Sprintf("character%dCurrentSP%d", c, realm))
		}
	}
	in = append(in, sp...)
	in = append(in, "previewOnly", "uiMode", "characterRefreshFlags", "combatRefreshFlags", "combatUIPointer")
	out := append(append([]string{}, sp...), "characterRefreshFlags", "combatRefreshFlags")
	Nodes["0052b550"] = Node{Addr: "0052b550", In: in, Out: out, F: partySpDrainNode}
}

// partySpDrainNode passes the same demand to each active party slot in order.
// Inputs contain independent active flags, HP and all realm stocks; the child
// also composes preview/message behavior and character/combat UI invalidation.
func partySpDrainNode(in []int64, r Rand) []int64 {
	if len(in) != 71 {
		panic("party_sp_drain: expected 71 inputs")
	}
	for _, i := range []int{0, 68, 69, 70} {
		if in[i] < 0 || in[i] > 4294967295 {
			panic("party_sp_drain: expected unsigned DWORD")
		}
	}
	for _, value := range append(append([]int64{}, in[1:10]...), in[66]) {
		if value < 0 || value > 255 {
			panic("party_sp_drain: expected byte flag")
		}
	}
	for _, value := range append(append([]int64{}, in[10:66]...), in[67]) {
		if value < -2147483648 || value > 2147483647 {
			panic("party_sp_drain: expected signed DWORD representation")
		}
	}
	out := append(append([]int64{}, in[18:66]...), in[68], in[69])
	// 0x52b565 tests each active byte for nonzero; 0x52b56e calls the child.
	// The caller preserves the original demand and notification flag for all slots.
	for c := 0; c < 8; c++ {
		if in[2+c] == 0 {
			continue
		}
		child := []int64{int64(c), in[0], in[1], in[10+c]}
		child = append(child, out[:48]...)
		child = append(child, in[66], in[67], out[48], out[49], in[70])
		out = characterRandomRealmSpDrainNode(child, r)
	}
	return out
}
