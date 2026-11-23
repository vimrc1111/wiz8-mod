package calc

import "fmt"

// item_stack_normalize 0x51fb40: fold an item instance's two count bytes
// according to the item definition's kind, then split a stack above the
// definition's stack limit into new shared-pool instances
// (notes/re/formulas.md; build/wiz8.asm:389974..390052).
// Instance layout (12 bytes): id i32, qty u8 @+4, side count u8 @+5, bytes @+6,+7,
// u32 @+8. Definition fields: kind u8 @+0x66, stack limit u8 @+0x6b.
// The per-slot instance mover 0x51fe30 runs natively in the emulator; its model
// here is a 12-byte copy with byte +0xa cleared. The shared-weight refresh 0x4edd20
// and the equipment refresh 0x520d10 are address-stubbed in the case: Out carries
// their call counts and the last 0x520d10 arguments (slot address, 0, 1).
func init() {
	in := []string{"id", "qty", "b5", "b6", "b7", "w8", "defType", "defMax", "slotCount"}
	out := []string{"instId", "instTail"}
	for k := 0; k < isnSlots; k++ {
		out = append(out, fmt.Sprintf("slot%d.id", k), fmt.Sprintf("slot%d.tail", k))
	}
	out = append(out, "slotCount", "sharedRefreshCalls", "actionRefreshCalls", "actionRefreshArg1", "actionRefreshArg2", "actionRefreshArg3")
	Nodes["0051fb40"] = Node{Addr: "0051fb40", In: in, Out: out, F: itemStackNormalizeNode}
}

const (
	isnSlots   = 500                // shared instance slots 0x685191..0x686901, 12 bytes each (cap 0x1f4 at 0x51fc1a)
	isnSlotAt  = 0x685191           // slot array base
	isnMaxDefs = 15                 // fixture definition table: ids 0..15; calc panics past it
	isnClr     = uint64(0xff) << 48 // instance byte +0xa lives in tail bits 48..55
)

func itemStackNormalizeNode(in []int64, r Rand) []int64 {
	if len(in) != 9 {
		panic("item stack normalize: wrong input count")
	}
	id, qty, b5 := int32(in[0]), uint8(in[1]), uint8(in[2])
	kind, limit, count := uint8(in[6]), uint8(in[7]), uint32(in[8])
	if id < -1 || id > isnMaxDefs {
		panic("item stack normalize: item id outside the fixture definition table")
	}
	tail := func(q, s uint8) uint64 {
		return uint64(q) | uint64(s)<<8 | uint64(uint8(in[3]))<<16 | uint64(uint8(in[4]))<<24 | uint64(uint32(in[5]))<<32
	}
	slotID := make([]int32, isnSlots)
	slotTail := make([]uint64, isnSlots)
	var shared, action, last uint32
	arg3 := uint32(0)
	if id != -1 {
		switch {
		case kind == 1: // 0x51fba1: side count moves into an empty main count
			if b5 != 0 {
				if qty == 0 {
					qty = b5
				}
				b5 = 0
			}
		case kind >= 2 && kind <= 4: // 0x51fb84: main count moves to the side count
			if qty != 0 {
				if b5 == 0 {
					b5 = qty
				}
				qty = 0
			}
		}
		if kind == 1 {
			if limit == 0 {
				qty = 0
			}
			for limit != 0 && qty > limit {
				take := qty - limit
				if take > limit {
					take = limit
				}
				if count >= isnSlots { // pool full: clamp to the limit
					qty = limit
					break
				}
				slotID[count] = id
				slotTail[count] = tail(take, b5) &^ isnClr
				last, arg3 = isnSlotAt+12*count, 1
				count++
				shared++
				action += 2
				qty -= take
			}
		}
	}
	o := []int64{int64(int32(in[0])), int64(tail(qty, b5))}
	for k := 0; k < isnSlots; k++ {
		o = append(o, int64(slotID[k]), int64(slotTail[k]))
	}
	return append(o, int64(count), int64(shared), int64(action), int64(last), 0, int64(arg3))
}
