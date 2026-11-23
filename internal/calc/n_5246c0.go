package calc

import "fmt"

// party_condition_remove 0x5246c0: for each of the 8 party slots whose presence
// byte (table at global 0x685178, stride 0x106) is non-zero and whose state
// slot `cond` (character +0xa01+4*cond, stride 0x1862 from global 0x685174) is
// non-zero, call 0x523330(member, cond, 1). build/wiz8.asm:396710.
//
// 0x523330 (D4 formula) is address-stubbed in the emulator case, so its
// arguments (last call) and call count are the outputs. Callers pass cond as a
// constant (0xd, 0); the character has 20 state slots, so cond >= 20 would read
// other record fields and panics.
const (
	pcrMembers = 8
	pcrSlots   = 20
)

func init() {
	in := []string{"cond"}
	for j := 0; j < pcrMembers; j++ {
		in = append(in, fmt.Sprintf("pres%d", j))
	}
	for j := 0; j < pcrMembers; j++ {
		in = append(in, fmt.Sprintf("slot%d", j))
	}
	Nodes["005246c0"] = Node{Addr: "005246c0", In: in, Out: []string{"n", "who", "cond", "flag"}, F: partyConditionRemoveNode}
}

func partyConditionRemoveNode(in []int64, r Rand) []int64 {
	if len(in) != 1+2*pcrMembers {
		panic("partyConditionRemove: wrong input count")
	}
	cond := in[0]
	if cond < 0 || cond >= pcrSlots {
		panic("partyConditionRemove: cond outside the 20 state slots")
	}
	var n, who, c, flag int64
	for j := 0; j < pcrMembers; j++ {
		if in[1+j] != 0 && uint32(in[1+pcrMembers+j]) != 0 {
			n, who, c, flag = n+1, int64(j), cond, 1
		}
	}
	return []int64{n, who, c, flag}
}
