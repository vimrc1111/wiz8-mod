package calc

import "strconv"

// Fumble candidate list (0x543dc0). The fixture holds eight party records, two
// loaded monster records, and an existing candidate list of up to 16 entries.
// Both range gates run the original callees: party members use the 0x5197c0
// composition (partyTargetAttackRangeNode) and monsters use the 0x5194e0
// composition (interceptMonsterAttackRange in n_545c20.go), always with attack
// context 0, hand as given, and no failure feedback (0x543f76..0x543f81,
// 0x543e78..0x543e81). List growth mirrors the 0x5e2480 model in n_544010.go.
func fumbleTargetListNode(in []int64, r Rand) []int64 {
	names := Nodes["00543dc0"].In
	if len(in) != len(names) {
		panic("fumble target list: wrong input count")
	}
	f := make(map[string]int64, len(in))
	for i, name := range names {
		f[name] = in[i]
	}
	attacker, capacity, count := f["characterSlot"], f["candidateCapacity"], f["candidateCount"]
	if attacker < 0 || attacker > 7 {
		panic("fumble target list: attacker is not a party slot")
	}
	if capacity < 0 || capacity > 16 || count < 0 || count > capacity {
		panic("fumble target list: fixture holds 16 entries and count may not exceed capacity")
	}
	n := func(name string, i int64) string { return name + strconv.FormatInt(i, 10) }
	spellRange := func(spell int64) int64 {
		if spell < 0 || spell > 9 {
			panic("fumble target list: fixture holds spell records 0..9")
		}
		return f[n("spell", spell)+"Range"]
	}
	// 0x543dfa/0x54406d style gate of party members: present, nonzero HP, condition below 18.
	partyArg := func(name string, slot int64) int64 {
		switch name {
		case "characterSlot", "hand":
			return f[name]
		case "characterPresent":
			return f[n("slotActive", attacker)]
		case "characterHP":
			return f[n("currentHP", attacker)]
		case "characterCondition":
			return f[n("mainStatus", attacker)]
		case "mainSpell":
			return f["mainActionSpell"]
		case "targetSlot":
			return slot
		case "centerSlot0", "centerSlot1", "centerSlot2":
			return f["quadrant4Slot"+name[len(name)-1:]]
		case "attackContext", "offActionKind", "battleActionKind", "offSpell", "queuedSpell", "battleSpell",
			"specialBaseSpell", "specialOverrideSpell", "phaseType", "battleSpecialFlag",
			"specialItemOverridePointer", "specialBaseItemIndex", "specialOverrideItemIndex":
			return 0 // Context 0 never reads these.
		case "selectedCharacterSlot":
			return 0 // Only context 6 reads it; the callee's domain is 0..7.
		}
		v, ok := f[name]
		if !ok {
			panic("fumble target list: no input for party callee field " + name)
		}
		return v
	}
	monsterArg := func(name string, m int64) int64 {
		switch name {
		case "attackContext", "showFailure", "offActionKind", "battleActionKind":
			return 0
		case "monsterTargetEnabled", "monsterPhysicalAllowed", "monsterSpellAllowed", "monsterX", "monsterY", "monsterZ", "monsterRadius":
			return f[n(name, m)]
		case "characterPresent":
			return f[n("slotActive", attacker)]
		case "characterHP":
			return f[n("currentHP", attacker)]
		case "characterCondition":
			return f[n("mainStatus", attacker)]
		case "spellId":
			return f["mainActionSpell"]
		case "spellRange":
			return spellRange(f["mainActionSpell"])
		case "item0SpellRange", "item1SpellRange", "item2SpellRange", "item3SpellRange":
			return spellRange(f["item"+name[4:5]+"Spell"])
		}
		v, ok := f[name]
		if !ok {
			panic("fumble target list: no input for monster callee field " + name)
		}
		return v
	}
	partyReach := func(slot int64) bool {
		calleeNames := Nodes["005197c0"].In
		args := make([]int64, len(calleeNames))
		for i, name := range calleeNames {
			args[i] = partyArg(name, slot)
		}
		return partyTargetAttackRangeNode(args, r)[0] != 0
	}
	monsterReach := func(m int64) bool {
		calleeNames := Nodes["005194e0"].In
		args := make([]int64, len(calleeNames))
		for i, name := range calleeNames {
			args[i] = monsterArg(name, m)
		}
		// Action kind 8 reads the item picked through the attacker's item list
		// (0x5199f0 model: no list means no item, unusable); the shared adapter
		// only knows item 0.
		if args[31] == 8 {
			if f["itemEntriesPointer"] == 0 {
				args[27] = 0
			} else {
				spell := f[n("item", f["selectedItemIndex"])+"Spell"]
				args[27], args[34] = spell, spellRange(spell)
			}
		}
		return interceptMonsterAttackRange(args, r)[0] != 0
	}

	// Fixture list state: the 16-entry candidate array and the bump heap
	// (emu.py HEAP 0x50000000, 8-byte aligned, 128-word snapshot).
	const heapBase int64 = 0x50000000
	const entriesBase int64 = 0x20000000 + 230032
	heap := make([]int64, 128)
	old := make([]int64, 16)
	for i := range old {
		old[i] = f[n("oldEntry", int64(i))]
	}
	used := int64(0)
	alloc := func(bytes int64) int64 {
		at := heapBase + used
		used += max(int64(8), (bytes+7)&^7)
		if used > 512 {
			panic("fumble target list: fixture heap snapshot exceeded")
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
	entries := entriesBase
	appendTarget := func(kind, character, monster int64) {
		record := alloc(32) // 32-byte candidate record: 0x543e10, 0x543e8d, 0x543f8f.
		write(record, 0, kind)
		write(record, 1, int64(uint32(character)))
		write(record, 2, int64(uint32(monster)))
		// 0x5e2480 expands a full list by five entries, keeping live ones.
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

	// 0x543dd5..0x543f06: party slots in ascending order.
	for slot := int64(0); slot < 8; slot++ {
		if f[n("slotActive", slot)] == 0 || f[n("currentHP", slot)] == 0 || f[n("mainStatus", slot)] >= 18 {
			continue
		}
		if slot == attacker { // 0x543e0c: the attacker is always a candidate.
			appendTarget(1, slot, -1)
			continue
		}
		if f["originalTargetType"] == 1 && f["originalCharacterTarget"] == slot { // 0x543e6b.
			continue
		}
		if partyReach(slot) {
			appendTarget(1, slot, -1)
		}
	}
	// 0x543f25..0x544000: loaded monsters; active, nonzero HP and visible.
	for m := int64(0); m < f["monsterCount"]; m++ {
		if f[n("monsterActive", m)] == 0 || f[n("monsterHP", m)] == 0 || f[n("monsterVisible", m)] == 0 {
			continue
		}
		id := f[n("monsterID", m)]
		if f["originalTargetType"] == 3 && f["originalMonsterTarget"] == id { // 0x543f67.
			continue
		}
		if monsterReach(m) {
			appendTarget(3, -1, id)
		}
	}
	out := []int64{capacity, count, entries}
	out = append(out, old...)
	return append(out, heap...)
}

func init() {
	in := []string{
		"characterSlot", "hand", "candidateCapacity", "candidateCount", "oldEntry0", "oldEntry1",
		"oldEntry2", "oldEntry3", "oldEntry4", "oldEntry5", "oldEntry6", "oldEntry7",
		"oldEntry8", "oldEntry9", "oldEntry10", "oldEntry11", "oldEntry12", "oldEntry13",
		"oldEntry14", "oldEntry15", "originalTargetType", "originalCharacterTarget", "originalMonsterTarget", "monsterCount",
		"monsterID0", "monsterActive0", "monsterVisible0", "monsterHP0", "monsterTargetEnabled0", "monsterPhysicalAllowed0",
		"monsterSpellAllowed0", "monsterX0", "monsterY0", "monsterZ0", "monsterRadius0", "monsterID1",
		"monsterActive1", "monsterVisible1", "monsterHP1", "monsterTargetEnabled1", "monsterPhysicalAllowed1", "monsterSpellAllowed1",
		"monsterX1", "monsterY1", "monsterZ1", "monsterRadius1", "partyX", "partyY",
		"partyZ", "partyRadius", "slotActive0", "currentHP0", "mainStatus0", "slotActive1",
		"currentHP1", "mainStatus1", "slotActive2", "currentHP2", "mainStatus2", "slotActive3",
		"currentHP3", "mainStatus3", "slotActive4", "currentHP4", "mainStatus4", "slotActive5",
		"currentHP5", "mainStatus5", "slotActive6", "currentHP6", "mainStatus6", "slotActive7",
		"currentHP7", "mainStatus7", "combatPhase", "combatMode", "specialSpellMode", "specialItemMode",
		"characterInParty", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex",
		"offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range", "item0Spell",
		"item1Spell", "item2Spell", "item3Spell", "mainActionKind", "mainActionSpell", "itemEntriesPointer",
		"selectedItemIndex", "spell0Range", "spell1Range", "spell2Range", "spell3Range", "spell4Range",
		"spell5Range", "spell6Range", "spell7Range", "spell8Range", "spell9Range", "character0Quadrant",
		"character1Quadrant", "character2Quadrant", "character3Quadrant", "character4Quadrant", "character5Quadrant", "character6Quadrant",
		"character7Quadrant", "partyHeadingDegrees", "quadrant0Slot0", "quadrant0Slot1", "quadrant0Slot2", "quadrant1Slot0",
		"quadrant1Slot1", "quadrant1Slot2", "quadrant2Slot0", "quadrant2Slot1", "quadrant2Slot2", "quadrant3Slot0",
		"quadrant3Slot1", "quadrant3Slot2", "quadrant4Slot0", "quadrant4Slot1", "quadrant4Slot2", "character0FormationUnavailable",
		"character1FormationUnavailable", "character2FormationUnavailable", "character3FormationUnavailable", "character4FormationUnavailable", "character5FormationUnavailable", "character6FormationUnavailable",
		"character7FormationUnavailable",
	}
	out := []string{"capacity", "count", "entriesPointer"}
	for i := 0; i < 16; i++ {
		out = append(out, "oldEntry"+strconv.Itoa(i))
	}
	for i := 0; i < 128; i++ {
		out = append(out, "allocatedWord"+strconv.Itoa(i))
	}
	Nodes["00543dc0"] = Node{Addr: "00543dc0", In: in, Out: out, F: fumbleTargetListNode}
}
