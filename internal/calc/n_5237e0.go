package calc

import "fmt"

// character_primary_condition 0x5237e0: pick the highest active state slot
// (index 0x13 down, slots at character +0xa01, 4 bytes each) as the main status
// (+0xb01), keep the three formation tables in step, refresh the portrait
// expression and mark the battle/screen panels (build/wiz8.asm:395349..).
//
// Callees: 0x554e70 is formationRestoreAfterStatusNode (own node, reused); the
// notification gate 0x68517c is disabled in the replay, which also makes the
// shared-weight refresh 0x4edd20 return at once when the status crosses 0x12
// (0x5238a8..0x5238d1), so that call changes nothing here. 0x55ee30/0x562a50
// (panel marks) and 0x52f000 (portrait expression) are inlined below.
const (
	pcSlots     = 20 // slot indexes 0..0x13 (0x52384f)
	pcMembers   = 8  // character stride 0x1862, presence stride 0x106
	pcFormLen   = 52 // inputs of the formation node after character/status
	pcHead      = 1 + pcSlots
	pcOld       = pcHead
	pcCombat    = pcHead + 1
	pcMode      = pcHead + 2
	pcBattle    = pcHead + 3
	pcMask      = pcHead + 4
	pcScreen    = pcHead + 5
	pcGate      = pcHead + 6
	pcExpr      = pcHead + 7
	pcFlag99    = pcExpr + pcMembers
	pcForm      = pcFlag99 + pcMembers
	pcPortrait  = 0xf      // expression threshold on the new status (0x5238ed, 0x52f031)
	pcAbsent    = 2        // expression index used for status 0xf and above (0x52f053)
	pcMode6Bits = 0x100    // screen +0xf8 mark in interface mode 6 (0x55ee57)
	pcMode7Bits = 0x208000 // battle +0xf4 marks 0x200000|0x8000 (0x52391d, 0x523927)
)

func init() {
	var in, out []string
	in = append(in, "who")
	for k := 0; k < pcSlots; k++ {
		in = append(in, fmt.Sprintf("slot%d", k))
	}
	in = append(in, "oldState", "combat", "interfaceMode", "battle", "battleMask", "screenFlags", "portraitGate")
	for c := 0; c < pcMembers; c++ {
		in = append(in, fmt.Sprintf("expr%d", c))
	}
	for c := 0; c < pcMembers; c++ {
		in = append(in, fmt.Sprintf("flag99_%d", c))
	}
	out = append(out, "newState")
	for c := 0; c < pcMembers; c++ {
		out = append(out, fmt.Sprintf("expr%d", c))
	}
	for c := 0; c < pcMembers; c++ {
		out = append(out, fmt.Sprintf("flag99_%d", c))
	}
	out = append(out, "battleMask", "screenFlags")
	for _, t := range "ABC" {
		for i := 0; i < pcFormLen; i++ {
			n := fmt.Sprintf("%c.f%d", t, i)
			in = append(in, n)
			out = append(out, n)
		}
	}
	Nodes["005237e0"] = Node{Addr: "005237e0", In: in, Out: out, F: characterPrimaryConditionNode}
}

func characterPrimaryConditionNode(in []int64, r Rand) []int64 {
	if len(in) != pcForm+3*pcFormLen {
		panic("character primary condition: wrong input count")
	}
	who := int(in[0])
	if who < 0 || who >= pcMembers {
		panic("character primary condition: member out of range")
	}
	state := int64(0) // highest non-zero slot, 0 when none (0x52385b..0x52386e)
	for k := pcSlots - 1; k > 0; k-- {
		if uint32(in[1+k]) != 0 {
			state = int64(k)
			break
		}
	}
	mask, screen := uint32(in[pcMask]), uint32(in[pcScreen])
	expr := append([]int64(nil), in[pcExpr:pcExpr+pcMembers]...)
	flag := append([]int64(nil), in[pcFlag99:pcFlag99+pcMembers]...)
	forms := make([][]int64, 3)
	for t := range forms {
		f := append([]int64{int64(who), state}, in[pcForm+t*pcFormLen:pcForm+(t+1)*pcFormLen]...)
		forms[t] = formationRestoreAfterStatusNode(f, r)
		if t == 0 && in[pcCombat] == 0 { // the other two tables only follow in combat (0x523887)
			forms[1] = append([]int64(nil), in[pcForm+pcFormLen:pcForm+2*pcFormLen]...)
			forms[2] = append([]int64(nil), in[pcForm+2*pcFormLen:]...)
			break
		}
	}
	// 0x55ee30: mode 6 marks the screen, mode 7 marks the member's bit on the battle object.
	switch in[pcMode] {
	case 6:
		screen |= pcMode6Bits
	case 7:
		if in[pcBattle] != 0 {
			mask |= 1 << uint(who&31)
		}
	}
	// 0x5238ed..0x523909: portrait expression follows the new status.
	if (uint32(state) < pcPortrait && expr[who] == pcAbsent) || (uint32(state) >= pcPortrait && expr[who] != pcAbsent) {
		flag[who] = 0 // 0x52f018
		if uint32(state) < pcPortrait && in[pcGate] == 0 {
			expr[who] = 1
		} else {
			expr[who] = pcAbsent
		}
	}
	if in[pcMode] == 7 && in[pcBattle] != 0 {
		mask |= pcMode7Bits
	}
	out := []int64{state}
	out = append(out, expr...)
	out = append(out, flag...)
	out = append(out, int64(mask), int64(screen))
	for _, f := range forms {
		out = append(out, f...)
	}
	return out
}
