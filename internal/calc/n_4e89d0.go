package calc

import "fmt"

// combatInitiativePhaseNode is 0x4e89d0 combat_initiative_phase: recompute every
// party member's and every monster's initiative phase for the round.
//
// Party member i (0..7; character record C = *0x685174 stride 0x1862, presence
// table S = *0x685178 stride 0x106, combat state B = *0x6836a8, phase record
// B+i*0xd4) is "ready" when S[i] byte 0 is set, character +0xb11 (HP, u32) is
// non-zero, +0xb01 (u32) is below 15, and byte B+0xa52 is zero (exe 0x4e89e2..0x4e8a1d).
// A ready member: v = R(4) + i32 C+0xe81; when S[i]+0x3d (dword) is 0 or 1 the
// best hand h = chooseAttackHandNode (0x53d390, native) is added when not -1:
// v += i32 at C+0x1166+0x5b*h; v is clamped as a signed value to -10..89 (0x5179d0,
// exe 0x4e8a7b); phase = 90 - v (B+i*0xd4+0x18, always 1..100); then 0x4f0480
// rounds it (combatPhaseRoundingNode, native, mode B+0x910, floor 10). With
// *0x683fa5 non-zero and byte C+0x767 non-zero, R(5)==0 calls 0x553f10(C record,
// 0x27, 1, 0) (stubbed in the case: the call count and its four arguments are
// outputs). A member who is not ready gets S[i]+1 dword = -1, phase 0 and B+i*0xd4+0x4c = 1.
// After the party loop 0x4efe70 runs when B+0x910 != 0 (stubbed in the case:
// only the call count is an output, its +0x914 write is not modelled).
// Each monster j (list *0x683fad, count at +8, null = 0) with byte +0x15 set is
// ready when dword +0x2b != 0, dword +0x107 < 15 and the flag byte is zero
// (B+0xa53 when byte +0x16 == 1, else B+0xa52): v = R(4) + u8 template(+8)+0xe4,
// clamped to -10..89, written 90 - v through pointer +0x10 and rounded by 0x4f0480;
// otherwise dword +0x2e1 = -1, [+0x10] = 0 and byte [+0x10]+4 = 1.
//
// Fixture caps (calc panics past them): at most 8 monsters, list pointer 0 or
// the case list, monster template ids 0..3, hand item index 0..3 (chooseAttackHandNode).
// 0x40efa0 is the Rand R; 0x4e5620/0x4e5720/0x5e2c70 run natively and are folded in.
const (
	cipGlobals  = 10
	cipSlot     = 18
	cipMon      = 8
	cipMonsters = 8
	cipTmpl     = 4
	cipListPtr  = 0x20000000 + 80000 // case object "list"
	cipCharPtr  = 0x20000000         // case object "characters"
	cipGrowth   = 8                  // 553f10 calls recorded: one per party slot
)

func init() {
	in := []string{"combatFlag683fa5", "partyActionMode", "flagA52", "flagA53", "monsterListPointer", "monsterCount",
		"item0Range", "item1Range", "item2Range", "item3Range"}
	out := []string{}
	for i := 0; i < 8; i++ {
		for _, f := range []string{"active", "currentHP", "limit0xb01", "initiative", "slotKind", "growthFlag",
			"mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex",
			"mainAttackCount", "mainInitiative", "offAttackCount", "offInitiative", "slotWord1", "phaseFlag4c"} {
			in = append(in, fmt.Sprintf("slot%d_%s", i, f))
		}
		out = append(out, fmt.Sprintf("slot%d_phase", i), fmt.Sprintf("slot%d_phaseFlag4c", i), fmt.Sprintf("slot%d_slotWord1", i))
	}
	for j := 0; j < cipMonsters; j++ {
		for _, f := range []string{"present", "state", "hp", "field107", "templateId", "phaseInit", "phaseByte4", "field2e1"} {
			in = append(in, fmt.Sprintf("monster%d_%s", j, f))
		}
		out = append(out, fmt.Sprintf("monster%d_phase", j), fmt.Sprintf("monster%d_phaseByte4", j), fmt.Sprintf("monster%d_field2e1", j))
	}
	for t := 0; t < cipTmpl; t++ {
		in = append(in, fmt.Sprintf("template%d_rollBonus", t))
	}
	out = append(out, "growthCalls")
	for k := 0; k < cipGrowth; k++ {
		for _, f := range []string{"who", "skill", "amount", "flag"} {
			out = append(out, fmt.Sprintf("growth%d_%s", k, f))
		}
	}
	out = append(out, "partyActionCalls")
	Nodes["004e89d0"] = Node{Addr: "004e89d0", In: in, Out: out, F: combatInitiativePhaseNode}
}

func cipClamp(v int32) int32 {
	// 0x5179d0: upper bound 0x59 first, then lower bound -10 (signed).
	if v > 0x59 {
		return 0x59
	} else if v < -10 {
		return -10
	}
	return v
}

func combatInitiativePhaseNode(in []int64, r Rand) []int64 {
	if len(in) != cipGlobals+8*cipSlot+cipMonsters*cipMon+cipTmpl {
		panic("combatInitiativePhase: wrong input count")
	}
	u := func(i int, max int64) int64 {
		if in[i] < 0 || in[i] > max {
			panic("combatInitiativePhase: input out of range")
		}
		return in[i]
	}
	const u8, u32 = 255, 4294967295
	combatFlag, mode := u(0, u32), u(1, u32)
	flagA52, flagA53 := u(2, u8), u(3, u8)
	listPtr, count := u(4, u32), u(5, u32)
	out := make([]int64, 0, 82)
	growth := make([]int64, 4*cipGrowth)
	var nGrowth int64
	for i := 0; i < 8; i++ {
		s := in[cipGlobals+i*cipSlot : cipGlobals+(i+1)*cipSlot]
		if s[0]&0xff != 0 && uint32(s[1]) != 0 && uint32(s[2]) < 15 && flagA52 == 0 {
			// exe 0x4e8a23..0x4e8a37: R(4) plus the signed initiative, 32-bit wrap.
			v := int32(r.R(4)) + int32(s[3])
			if k := uint32(s[4]); k == 0 || k == 1 {
				best := chooseAttackHandNode([]int64{int64(i), s[6], s[7], s[8], s[9], s[10], s[11],
					in[6], in[7], in[8], in[9], s[12], s[13], s[14], s[15]}, r)[0]
				if best != -1 {
					// exe 0x4e8a5e..0x4e8a77: record + 0x5b*hand, field 0x1166 (main s[13], off s[15]).
					v += int32(s[13+2*int(best)])
				}
			}
			v = cipClamp(v)
			phase := combatPhaseRoundingNode([]int64{mode, int64(uint32(0x5a - v)), 10}, r)[0]
			out = append(out, phase, s[17], s[16])
			if uint32(combatFlag) != 0 && s[5]&0xff != 0 && r.R(5) == 0 {
				// 0x553f10(character record, 0x27, 1, 0): arguments and count only.
				copy(growth[nGrowth*4:], []int64{cipCharPtr + int64(i)*0x1862, 0x27, 1, 0})
				nGrowth++
			}
		} else {
			out = append(out, 0, 1, 0xffffffff)
		}
	}
	var partyCalls int64
	if mode != 0 {
		partyCalls = 1 // 0x4efe70 at exe 0x4e8b41
	}
	n := count
	if listPtr == 0 { // exe 0x5e2c70: null list is zero monsters.
		n = 0
	} else if listPtr != cipListPtr {
		panic("combatInitiativePhase: list pointer must be 0 or the case list")
	}
	if n > cipMonsters {
		panic("combatInitiativePhase: only 8 monster fixture records")
	}
	base := cipGlobals + 8*cipSlot
	for j := 0; j < cipMonsters; j++ {
		m := in[base+j*cipMon : base+(j+1)*cipMon]
		phase, b4, f2e1 := m[5], m[6], m[7]
		if int64(j) < n && m[0]&0xff != 0 {
			flag := flagA52
			if m[1]&0xff == 1 {
				flag = flagA53
			}
			if uint32(m[2]) != 0 && uint32(m[3]) < 15 && flag == 0 {
				if uint32(m[4]) >= cipTmpl {
					panic("combatInitiativePhase: template id past the 4 fixture templates")
				}
				v := cipClamp(int32(r.R(4)) + int32(in[base+cipMonsters*cipMon+int(m[4])]&0xff))
				phase = combatPhaseRoundingNode([]int64{mode, int64(uint32(0x5a - v)), 10}, r)[0]
			} else {
				phase, b4, f2e1 = 0, 1, 0xffffffff
			}
		}
		out = append(out, phase, b4, f2e1)
	}
	out = append(out, nGrowth)
	out = append(out, growth...)
	return append(out, partyCalls)
}
