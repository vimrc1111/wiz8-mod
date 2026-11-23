package calc

import "fmt"

// item_charge_transfer 0x51e9f0 (cdecl; arg0 owner, arg1 target instance,
// arg2 source instance; build/wiz8.asm:388433..388492).
// Moves up to min(source amount, room left on the target) units from the
// source item instance to the target instance and consumes them from the source
// one call of 0x51e760 (itemUseConsumptionNode) at a time.
//   - instance: +0 item id (i32), +4 count byte, +5 charge byte.
//   - item record (table pointer 0x68516c, stride 269): +0x66 = 1 means the
//     count byte (+4) is the live amount, otherwise the charge byte (+5);
//     capacity = u8 +0x69 * u8 +0x6a + i16 +0x67.
//   - amount = source live byte; room = capacity - target live byte, computed in
//     32 bits unsigned (a negative room wraps and so never limits); the moved
//     amount is room when room < amount, else amount.
//   - the target charge byte (+5) gets += amount (mod 256), even when its live
//     byte is the count byte; then 0x51e760(owner, source, 0) runs amount times.
//
// The D4 callee 0x51e760 is modelled as the case address stub (composition
// ruling 2026-10-09): its return (input "ret", unused by the caller) is fixed;
// the last arguments (owner, source pointer, force 0) and the call count are
// outputs. The callee's own effect on the source is not applied here.
// Fixture caps (calc panics past them): item ids -1..3, the two instances A/B
// (pointers ictA/ictB, may alias each other).
const (
	ictRecs = 4 // item ids 0..3, plus the record before the table for id -1
	ictA    = 0x20001000
	ictB    = 0x20001100
)

type ictInst struct{ id, cnt, chg int64 }
type ictRec struct{ b66, s67, b69, b6a int64 }

func init() {
	in := []string{"owner", "tp", "sp", "A.id", "A.cnt", "A.chg", "B.id", "B.cnt", "B.chg", "ret"}
	for k := -1; k < ictRecs; k++ {
		for _, f := range []string{"b66", "s67", "b69", "b6a"} {
			in = append(in, fmt.Sprintf("r%d.%s", k, f))
		}
	}
	Nodes["0051e9f0"] = Node{Addr: "0051e9f0", In: in,
		Out: []string{"A.cnt", "A.chg", "B.cnt", "B.chg", "cnt", "rec.owner", "rec.sp", "rec.force"},
		F:   itemChargeTransferNode}
}

func itemChargeTransferNode(in []int64, r Rand) []int64 {
	if len(in) != 30 {
		panic("itemChargeTransfer: expected 30 inputs")
	}
	owner, tp, sp := in[0]&0xffffffff, in[1], in[2]
	inst := [2]ictInst{{in[3], in[4] & 0xff, in[5] & 0xff}, {in[6], in[7] & 0xff, in[8] & 0xff}}
	var rec [ictRecs + 1]ictRec // index = item id + 1
	for i := range rec {
		o := 10 + 4*i
		rec[i] = ictRec{in[o] & 0xff, in[o+1], in[o+2] & 0xff, in[o+3] & 0xff}
	}
	slot := func(p int64) int {
		switch p {
		case ictA:
			return 0
		case ictB:
			return 1
		}
		panic("itemChargeTransfer: instance pointer not modelled")
	}
	record := func(id int64) ictRec {
		if id < -1 || id >= ictRecs {
			panic("itemChargeTransfer: item id outside the modelled records")
		}
		return rec[id+1]
	}
	si, ti := slot(sp), slot(tp)
	live := func(i *ictInst) int64 {
		if record(i.id).b66 == 1 {
			return i.cnt
		}
		return i.chg
	}
	tr := record(inst[ti].id)
	amount := live(&inst[si])
	room := uint32(tr.b69*tr.b6a+int64(int16(tr.s67))) - uint32(live(&inst[ti]))
	if int64(room) < amount {
		amount = int64(room)
	}
	inst[ti].chg = (inst[ti].chg + amount) & 0xff

	var calls, recOwner, recSp int64
	for k := int64(0); k < amount; k++ {
		calls, recOwner, recSp = calls+1, owner, sp
	}
	return []int64{inst[0].cnt, inst[0].chg, inst[1].cnt, inst[1].chg, calls, recOwner, recSp, 0}
}
