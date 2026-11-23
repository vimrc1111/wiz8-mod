package calc

import "fmt"

// character_condition_decay 0x5236a0(who, cond, amount): reduce state slot `cond`
// (character record +0xa01+4*cond, record at *0x685174, stride 0x1862) of party
// member `who` by `amount` (build/wiz8.asm:395248..395340).
//   - the presence byte (record at *0x685178, stride 0x106, byte 0) must be set
//     and the slot non-zero, else srAssertFail (blocks 5236c4, 523707);
//   - slot <= amount (unsigned): character_condition_remove 0x523330(who, cond, 1)
//     and return;
//   - cond 7 (poison; the slot is the poison time): strength +0xb09 loses
//     q = strength*amount/time (32-bit wrapping product, unsigned division); when
//     q is 0 one R(100) is drawn and the loss is 1 if it is below
//     strength*amount*100/time (32-bit wrapping), else 0;
//   - finally slot -= amount.
//
// Emu address-stubs the D4 callee 0x523330 (composition ruling 2026-10-09): the
// slot stays unchanged and only the last call's arguments and the count are
// observable. F also runs the real characterConditionRemoveNode and uses its
// slot output.
const (
	ccdMembers = 8
	ccdSlots   = 20 // callee 0x523330 bound; callers pass constants or slot loop indexes
	ccdIn      = 4 + ccdSlots + 1
)

func init() {
	var in, out []string
	in = append(in, "who", "cond", "amount", "pres")
	for k := 0; k < ccdSlots; k++ {
		in = append(in, fmt.Sprintf("s%d", k))
		out = append(out, fmt.Sprintf("s%d", k))
	}
	in = append(in, "b09")
	out = append(out, "b09", "n", "lastWho", "lastCond", "lastNotify")
	Nodes["005236a0"] = Node{Addr: "005236a0", In: in, Out: out,
		F:   characterConditionDecayNode,
		Emu: func(in []int64, r Rand) []int64 { return ccdRun(in, r, false) }}
}

func characterConditionDecayNode(in []int64, r Rand) []int64 { return ccdRun(in, r, true) }

func ccdRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != ccdIn {
		panic("characterConditionDecay: wrong input count")
	}
	who, cond, amt := int(in[0]), int(in[1]), uint32(in[2])
	if who < 0 || who >= ccdMembers || cond < 0 || cond >= ccdSlots {
		panic("characterConditionDecay: who/cond out of range") // not a value any caller passes
	}
	if in[3]&0xff == 0 {
		panic("characterConditionDecay: assert, member absent") // 0x5236c4
	}
	slot := make([]int64, ccdSlots)
	copy(slot, in[4:4+ccdSlots])
	b09 := uint32(in[4+ccdSlots])
	state := uint32(slot[cond])
	if state == 0 {
		panic("characterConditionDecay: assert, slot is empty") // 0x523707
	}
	var n, lWho, lCond, lNotify int64
	if state <= amt {
		n, lWho, lCond, lNotify = 1, int64(who), int64(cond), 1
		if real {
			slot = ccdRemove(slot, who, cond, r)
		}
	} else {
		if cond == 7 {
			p := b09 * amt
			q := p / state
			if q == 0 {
				roll := uint32(r.R(100))
				if roll < p*100/state {
					q = 1
				}
			}
			b09 -= q
		}
		slot[cond] = int64(state - amt)
	}
	return append(slot, int64(b09), n, lWho, lCond, lNotify)
}

// ccdRemove runs the real 0x523330 for (who, cond, notify=1) on a world made of
// the member's presence byte and state words, zero elsewhere.
func ccdRemove(slot []int64, who, cond int, r Rand) []int64 {
	v := make([]int64, ccrLen)
	v[ccrCharBase], v[ccrWho], v[ccrCond], v[ccrNotify] = pcc0CharPtr, int64(who), int64(cond), 1
	v[ccrPres+who], v[ccrWhoPres] = 1, 1
	copy(v[ccrSlot:], slot)
	return characterConditionRemoveNode(v, r)[:ccdSlots]
}
