package calc

import "fmt"

// party_conditions_clear 0x524720: for condition slot 0..0x11 except 10, for
// party member 0..7 whose presence byte (record at *0x685178, stride 0x106,
// byte 0) is set, call character_condition_remove 0x523330(member, slot, 1)
// when the character's state word +0xa01+4*slot (record at *0x685174, stride
// 0x1862) is non-zero (build/wiz8.asm:396756..). Slots 0x12 and 0x13 and the
// presence record's other bytes are never read.
//
// Emu address-stubs the D4 callee 0x523330 (composition ruling 2026-10-09): only
// the last call's arguments and the call count are observable. F also runs the
// real characterConditionRemoveNode for every call, on a callee world made of
// the node's presence bytes and state words and zero elsewhere, and feeds its
// slot output back (the callee only clears slot `cond`, which the loop never
// rereads, so the call sequence is unchanged).
const (
	pcc0Slots   = 18 // slots 0..0x11 (0x524772)
	pcc0Skip    = 10 // 0x524726
	pcc0Members = 8  // presence offset 0x830 = 8*0x106 (0x524769)
	pcc0CharPtr = 0x20000000
)

func init() {
	var in []string
	for j := 0; j < pcc0Members; j++ {
		in = append(in, fmt.Sprintf("pres%d", j))
	}
	for w := 0; w < pcc0Members; w++ {
		for c := 0; c < pcc0Slots; c++ {
			in = append(in, fmt.Sprintf("s%d_%d", w, c))
		}
	}
	Nodes["00524720"] = Node{Addr: "00524720", In: in,
		Out: []string{"n", "lastWho", "lastCond", "lastNotify"},
		F:   func(in []int64, r Rand) []int64 { return partyConditionsClear(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return partyConditionsClear(in, r, false) }}
}

func partyConditionsClearNode(in []int64, r Rand) []int64 { return partyConditionsClear(in, r, true) }

func partyConditionsClear(in []int64, r Rand, real bool) []int64 {
	if len(in) != pcc0Members+pcc0Members*pcc0Slots {
		panic("party_conditions_clear: wrong input count")
	}
	in = append([]int64(nil), in...) // pccRemove feeds slot outputs back
	var n, who, cond, notify int64
	for c := 0; c < pcc0Slots; c++ {
		if c == pcc0Skip {
			continue
		}
		for w := 0; w < pcc0Members; w++ {
			if in[w]&0xff == 0 || uint32(in[pcc0Members+w*pcc0Slots+c]) == 0 {
				continue
			}
			if real {
				pccRemove(in, w, c, r)
			}
			n, who, cond, notify = n+1, int64(w), int64(c), 1
		}
	}
	return []int64{n, who, cond, notify}
}

// pccRemove runs the real 0x523330 for (w, c, notify=1) and writes back its slot output.
func pccRemove(in []int64, w, c int, r Rand) {
	v := make([]int64, ccrLen)
	v[ccrCharBase], v[ccrWho], v[ccrCond], v[ccrNotify] = pcc0CharPtr, int64(w), int64(c), 1
	for j := 0; j < pcc0Members; j++ {
		v[ccrPres+j] = in[j] & 0xff
	}
	v[ccrWhoPres] = in[w] & 0xff
	for k := 0; k < pcc0Slots; k++ {
		v[ccrSlot+k] = in[pcc0Members+w*pcc0Slots+k]
	}
	out := characterConditionRemoveNode(v, r)
	for k := 0; k < pcc0Slots; k++ {
		in[pcc0Members+w*pcc0Slots+k] = out[k]
	}
}
