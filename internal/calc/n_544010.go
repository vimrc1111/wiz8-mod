package calc

import "strconv"

// The fixture supplies eight independent party records, an active list with
// zero to two distinct monster records, and an existing candidate list.
// Reach uses the existing 0x51a2f0/0x51a510 composition and runtime exe tables
// in n_545cf0.go. Adventure mode is fixed (0x683f94=0); this node does no IO.
func monsterFumbleTargetListNode(in []int64, r Rand) []int64 {
	if len(in) != 85 {
		panic("monster fumble: expected 85 input fields")
	}
	for i, v := range in {
		limit := int64(4294967295)
		switch i {
		case 5, 7, 8, 10, 30, 33, 36, 39, 42, 45, 48, 51, 56, 57, 58, 59, 60, 61, 62, 64, 65, 66, 67, 68, 82:
			limit = 255
		case 11:
			limit = 2
		case 12:
			limit = 16
		case 13:
			limit = 4
		case 55:
			limit = 3
		case 63:
			limit = 1
		case 83:
			limit = 2
		case 84:
			// M+0x2e5 indexes the spell table unchecked (0x51a3e1, 0x51a64a);
			// the only caller 540170 passes attack mode 0..8.
			limit = 8
		}
		if i == 69 {
			if v < -1 || v > 3 {
				panic("monster fumble: spell range must be -1..3")
			}
		} else if v < 0 || v > limit {
			panic("monster fumble: input field out of range")
		}
	}
	if in[13] > in[12] {
		panic("monster fumble: count exceeds capacity")
	}
	// 0x544085/0x544152/0x5441ce allocate 32-byte candidate records.
	// These addresses are emulator fixture locations (emu.py HEAP and case
	// candidateEntries), not game data. Observe allocations including padding,
	// live list copies and overwritten slots without replacing any child call.
	const heapBase int64 = 0x50000000
	const entriesBase int64 = 0x20000000 + 110032
	heap := make([]int64, 128)
	old := append([]int64(nil), in[14:30]...)
	used := int64(0)
	alloc := func(bytes int64) int64 {
		at := heapBase + used
		used += max(int64(8), (bytes+7)&^7) // emu.py bump allocation alignment.
		if used > 512 {
			panic("monster fumble: fixture heap snapshot exceeded")
		}
		return at
	}
	read := func(ptr int64, i int64) int64 {
		if ptr == entriesBase {
			return old[i]
		}
		return heap[(ptr-heapBase)/4+i]
	}
	write := func(ptr int64, i int64, value int64) {
		if ptr == entriesBase {
			old[i] = value
		} else {
			heap[(ptr-heapBase)/4+i] = value
		}
	}
	capacity, count, entries := in[12], in[13], entriesBase
	appendTarget := func(kind, character, monster int64) {
		record := alloc(32)
		write(record, 0, kind)
		write(record, 1, int64(uint32(character)))
		write(record, 2, int64(uint32(monster)))
		// 0x5e2480 expands a full list by five, retaining only live entries.
		if count >= capacity {
			grown := alloc((capacity + 5) * 4)
			for i := int64(0); i < count; i++ {
				write(grown, i, read(entries, i))
			}
			entries = grown
			capacity += 5
		}
		write(entries, count, record)
		count++
	}
	reach := make([]int64, 48)
	copy(reach[20:48], in[54:82])
	// 0x544025..0x5440fe iterate all eight slots in ascending order.
	for slot := int64(0); slot < 8; slot++ {
		at := 30 + 3*slot
		if in[at] == 0 || in[at+1] == 0 || in[at+2] >= 18 {
			continue
		}
		if in[0] == 1 && in[1] == slot {
			continue
		}
		if interceptReach(reach, true) {
			appendTarget(1, slot, -1)
		}
	}
	// 0x54411c..0x54423f: active, nonzero HP and visible gate all monsters.
	for monster := int64(0); monster < in[11]; monster++ {
		at := 5 + 3*monster
		if in[at] == 0 || in[at+1] == 0 || in[at+2] == 0 {
			continue
		}
		if monster == 0 {
			// Self is eligible without reach or original-target exclusion (0x544148).
			appendTarget(3, -1, in[3])
			continue
		}
		// Preserve the source-ID comparison at 0x5441aa, even when the original
		// target is a different monster: it does not compare the candidate's ID.
		if in[0] == 3 && in[2] == in[3] {
			continue
		}
		if in[5] == 0 && in[82] == 0 {
			panic("monster fumble: reach child requires active source")
		}
		// 0x51a519 excludes the source template's special flag before relation lookup.
		if in[82] == 0 && interceptReach(reach, false) {
			appendTarget(3, -1, in[4])
		}
	}
	out := []int64{capacity, count, entries}
	out = append(out, old...)
	return append(out, heap...)
}

func init() {
	in := []string{
		"originalTargetType",
		"originalCharacterTarget",
		"originalMonsterTarget",
		"actorID",
		"victimID",
		"actorActive",
		"actorCurrentHP",
		"actorVisible",
		"victimActive",
		"victimCurrentHP",
		"victimVisible",
		"monsterCount",
		"candidateCapacity",
		"candidateCount",
	}
	for i := 0; i < 16; i++ {
		in = append(in, "oldEntry"+strconv.Itoa(i))
	}
	for i := 0; i < 8; i++ {
		for _, name := range []string{"characterSlotActive", "characterCurrentHP", "characterMainStatus"} {
			in = append(in, name+strconv.Itoa(i))
		}
	}
	in = append(in, []string{
		"actionType",
		"weaponRange",
		"alternateHand",
		"alternateAction",
		"characterReachEnabled",
		"characterReach0",
		"characterReach1",
		"characterReach2",
		"characterReach3",
		"relationCount",
		"relationActive",
		"monsterReach0",
		"monsterReach1",
		"monsterReach2",
		"monsterReach3",
		"spellRange",
		"actorXBits",
		"actorYBits",
		"actorZBits",
		"victimXBits",
		"victimYBits",
		"victimZBits",
		"partyXBits",
		"partyYBits",
		"partyZBits",
		"actorRadiusBits",
		"victimRadiusBits",
		"partyRadiusBits",
		"actorSpecialTemplate",
		"attackIndex",
		"spellIndex",
	}...)
	out := []string{"capacity", "count", "entriesPointer"}
	for i := 0; i < 16; i++ {
		out = append(out, "oldEntry"+strconv.Itoa(i))
	}
	for i := 0; i < 128; i++ {
		out = append(out, "allocatedWord"+strconv.Itoa(i))
	}
	Nodes["00544010"] = Node{Addr: "00544010", In: in, Out: out, F: monsterFumbleTargetListNode}
}
