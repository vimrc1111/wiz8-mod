package calc

// monsterActionRollNode composes action creation, target initialization and
// list append. Record layout: 0x532372..0x5323ce; target defaults: 0x536170.
// The list's existing entries are explicit inputs, including the unused slot
// that is overwritten when appending without growth (0x5e2480..0x5e252d).
func monsterActionRollNode(in []int64, r Rand) []int64 {
	if len(in) != 16 {
		panic("monster action: expected five action/target fields, capacity, count, nine entries")
	}
	for _, v := range in[:5] {
		if v < -2147483648 || v > 2147483647 {
			panic("monster action: action/target fields must be i32")
		}
	}
	capacity, count := in[5], in[6]
	if capacity < 0 || capacity > 8 || count < 0 || count > capacity {
		panic("monster action: expected 0 <= count <= capacity <= 8")
	}
	for _, v := range in[7:] {
		if v < 0 || v > 4294967295 {
			panic("monster action: entries must be u32")
		}
	}
	// Twelve zeroed words; target initialization sets its three ID slots to
	// absent (-1). 0x53238f stores the attack parameter only for action type 0.
	record := []int64{in[0], in[1], 0, in[3], -1, -1, -1, 0, 0, 0, 0, 0}
	if in[0] == 0 {
		record[2] = in[2]
	}
	// Character target kind 1 and monster target kind 3: 0x5323ac..0x5323be.
	switch in[3] {
	case 1:
		record[4] = in[4]
	case 3:
		record[5] = in[4]
	}
	// 0x5323c1..0x5323ce: one R(100), plus one, stored as a byte.
	record[11] = int64(uint8(r.R(100) + 1))
	// These addresses describe the case harness, not game tables: emu.py's
	// HEAP starts at 0x50000000; the first allocation is 48 bytes (0x532361).
	// The case places the original entry buffer at ARENA+1024.
	const actionPointer int64 = 0x50000000
	const grownPointer int64 = actionPointer + 48
	const originalPointer int64 = 0x20000400
	oldEntries := append([]int64(nil), in[7:]...)
	grownEntries := make([]int64, 9)
	entriesPointer := originalPointer
	if count == capacity {
		// 0x5e24b6 and 0x5e250c: expand by five slots, copy live entries.
		capacity += 5
		entriesPointer = grownPointer
		copy(grownEntries[:count], oldEntries[:count])
		grownEntries[count] = actionPointer
	} else {
		oldEntries[count] = actionPointer
	}
	out := append(record, capacity, count+1, entriesPointer, count)
	out = append(out, oldEntries...)
	return append(out, grownEntries...)
}

func init() {
	Nodes["00532360"] = Node{
		Addr: "00532360",
		In: []string{"actionType", "attackIndex", "attackParameter", "targetType", "targetID", "capacity", "count",
			"entry0", "entry1", "entry2", "entry3", "entry4", "entry5", "entry6", "entry7", "entry8"},
		Out: []string{"actionType", "attackIndex", "attackParameter", "targetType", "characterTarget", "monsterTarget", "otherTarget",
			"targetReserved0", "targetReserved1", "targetReserved2", "targetReserved3", "actionRoll", "capacity", "count", "entriesPointer", "appendIndex",
			"oldEntry0", "oldEntry1", "oldEntry2", "oldEntry3", "oldEntry4", "oldEntry5", "oldEntry6", "oldEntry7", "oldEntry8",
			"grownEntry0", "grownEntry1", "grownEntry2", "grownEntry3", "grownEntry4", "grownEntry5", "grownEntry6", "grownEntry7", "grownEntry8"},
		F: monsterActionRollNode,
	}
}
