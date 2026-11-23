package calc

func init() {
	Nodes["0051cea0"] = Node{Addr: "0051cea0",
		In: []string{"decoyFb1Flags", "decoyFc9Flags", "decoyChar0fa9", "decoyChar0fb5", "decoyChar0fc1", "decoyChar0fcd",
			"itemIdx", "itemType", "itemFlags", "itemSlotByte", "slot", "ignoreOccupied",
			"heldA", "heldB", "heldC", "heldD",
			"heldAFlags", "heldCFlags"},
		Out: []string{"allowed"}, F: itemEquipSlotAllowedNode}
}

// Character record fields (i32 item indices, -1 = empty): +0xfa5, +0xfb1,
// +0xfbd, +0xfc9. Item definitions live at global 0x68516c, stride 269
// (+0x41 flags; bit 2 marks a two-hand item). Inputs: itemIdx/Type/Flags/
// SlotByte describe the tested item; heldA=+0xfa5, heldB=+0xfb1,
// heldC=+0xfbd, heldD=+0xfc9; the two Flags inputs are the +0x41 bytes of the
// records at +0xfa5 and +0xfbd. The first six inputs are unread decoys.
// When item indices coincide the item table holds one record, so the later
// write in input order wins (idx flags, then heldA's, then heldC's), as in
// the original's memory.
func itemEquipSlotAllowedNode(in []int64, r Rand) []int64 {
	if len(in) != 18 {
		panic("item equip slot allowed: expected eighteen inputs (first six are unread decoys)")
	}
	idx, typ, flags, slotByte := in[6], in[7], in[8], in[9]
	slot, ignore := in[10]&0xff, in[11]&0xff != 0
	heldA, heldB, heldC, heldD := int32(in[12]), int32(in[13]), int32(in[14]), int32(in[15])
	heldAFlags, heldCFlags := in[16]&0xff, in[17]&0xff
	// The record at index x has these flags after the harness's writes.
	flagsOf := func(x int32, base int64) int64 {
		switch {
		case x >= 0 && x == heldC:
			return heldCFlags
		case x >= 0 && x == heldA:
			return heldAFlags
		}
		return base
	}
	twoHand := func(x int32) bool { return x != -1 && flagsOf(x, 0)&4 != 0 }
	b := func(v bool) int64 {
		if v {
			return 1
		}
		return 0
	}
	e := b(heldB == -1 || ignore)
	f := b(heldD == -1 || ignore)
	g := b(!twoHand(heldA) || ignore)
	h := b(!twoHand(heldC) || ignore)
	// Same record the mask function reads for the tested item.
	itemFlags := flagsOf(int32(idx), flags&0xff)
	m := itemEquipSlotMaskNode([]int64{idx, typ, itemFlags, slotByte, e, f, g, h, 0, 0, 0, 0, 0, 0}, r)
	return []int64{b((int64(1)<<(slot&0x1f))&m[0]&0xffff != 0)}
}
