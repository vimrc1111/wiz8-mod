package calc

func init() {
	Nodes["00537380"] = Node{Addr: "00537380", In: []string{
		"characterSlot", "battlePhase", "battleObject", "battleActionKind", "battleSpell", "battleSpecialFlag",
		"selectedCharacterSlot", "combatMode", "specialSpellMode", "specialItemMode",
		"characterPresent", "characterHP", "characterCondition",
		"mainActionKind", "mainSpell", "mainItemEntries", "offActionKind", "offSpell", "offItemEntries",
		"queuedSpell", "queuedItemEntries", "specialBaseSpell", "specialOverrideSpell", "specialOverrideItemEntries",
		"mainItemIndex", "offItemIndex", "queuedItemIndex", "specialBaseItemIndex", "specialOverrideItemIndex",
		"mainItemSpell", "offItemSpell", "queuedItemSpell", "specialBaseItemSpell", "specialOverrideItemSpell",
		"mainItemType", "offItemType", "queuedItemType", "specialBaseItemType", "specialOverrideItemType",
		"mainItemSpellTarget", "offItemSpellTarget", "queuedItemSpellTarget", "specialBaseItemSpellTarget", "specialOverrideItemSpellTarget",
		"mainSpellTarget", "offSpellTarget", "queuedSpellTarget", "battleSpellTarget", "specialBaseSpellTarget", "specialOverrideSpellTarget",
	}, Out: []string{"targetType"}, F: characterActionTargetTypeNode}
}

// characterActionTargetTypeNode (exe 0x537380) for party slot in[0] (stack argument, 0..7).
// Inputs 0..9: slot, 0x68ec78 phase, 0x68edcc battle object pointer (only its nullness is read), battle
// +0x2f0/+0x2f4/+0x2f8, 0x68518d, bytes 0x683f94/95/96. 10..12: queue record +0 (u8), character +0xb11 (i32),
// +0xb01 (u32). 13..23: queue record +1/+5/+0x19, +0x3d/+0x41/+0x49, +0x75/+0xa5, special object +0x100/+0x104,
// 0x69b9a4. 24..28: the item index each item-entry list points at (queue main/off/queued, 0x69b9a0 base,
// 0x69b9a4 override). 29..43: item record +0x63 / +0x3e of that index, and the target field (+0x137) of
// the spell record that item spell names. 44..49: target field of the spell record each spell source names.
// Record fields are read from the sparse memory the inputs fill: the last writer to a record in input order wins,
// exactly the order the fixture writes them. Pointer inputs are present when nonzero (the lists they point at
// are the index inputs); 0x69b9a0 is always present.
// The spell database has 150 records (notes/re/combat-2.md C11) and the item database 819 (notes/re/loot.md);
// used spell ids are accepted over 0..255 and used item indices over -1..818.
func characterActionTargetTypeNode(in []int64, r Rand) []int64 {
	if len(in) != 50 {
		panic("character action target: expected 50 inputs")
	}
	u32 := func(v int64) bool { return v >= 0 && v <= 4294967295 }
	i32 := func(v int64) bool { return v >= -2147483648 && v <= 2147483647 }
	for i, v := range in {
		ok := true
		switch {
		case i == 0:
			ok = v >= 0 && v <= 7
		case i == 5 || i == 7 || i == 8 || i == 9 || i == 10 || i >= 29 && i <= 38:
			ok = v >= 0 && v <= 255
		case i == 1 || i == 2 || i == 6 || i == 12 || i == 15 || i == 18 || i == 20 || i == 23:
			ok = u32(v)
		default:
			ok = i32(v)
		}
		if !ok {
			panic("character action target: input outside its storage width")
		}
	}
	slot := in[0]
	// 0x537380..0x5373fb: action source context, same rule as 0x53bc90 resolves for 0x4e77b0.
	ctx := int64(0)
	switch {
	case in[1] == 7 && in[2] != 0 && in[3] != -1:
		ctx = 7
		if in[5] != 0 && in[3] == 7 {
			ctx = 3
		} else if in[5] != 0 && in[3] == 8 {
			ctx = 4
		}
	case slot == in[6] && (in[8] != 0 || in[9] != 0):
		ctx = 2
	case in[7] != 0:
		ctx = 1
	}
	if ctx == 0 { // 0x537408 dispatch entry 0.
		return []int64{0}
	}
	// 0x4e77b0 with context 6 resolved above: action kind, spell id and item-entry list.
	kind, spell := int64(0), int64(0)
	list, source := false, -1 // list: entries pointer present; source: index input 24+source.
	switch ctx {
	case 1:
		kind, spell, list, source = in[16], in[17], in[18] != 0, 1
	case 2:
		if in[8] != 0 {
			kind, spell = 7, in[21]
			if in[22] != 0 {
				spell = in[22]
			}
		} else {
			kind, list, source = 8, true, 3 // 0x59e0d0: override list when 0x69b9a4 is set, else base.
			if in[23] != 0 {
				source = 4
			}
		}
	case 3:
		kind, spell = 7, in[19]
	case 4:
		kind, list, source = 8, in[20] != 0, 2
	case 7:
		kind, spell = in[3], in[4]
		if kind == 8 {
			// 0x4e78cf leaves the list pointer slot NULL; 0x5374a0 dereferences it: the original faults.
			panic("character action target: battle item action without special flag dereferences NULL")
		}
	default:
		kind, spell, list, source = in[13], in[14], in[15] != 0, 0
	}
	// 0x53c270: slot present, HP nonzero, condition below 18; kinds 10/11 survive an unusable slot.
	targetable := make([]int64, 25)
	targetable[0] = slot
	copy(targetable[1+3*slot:], []int64{in[10], in[11], in[12]})
	if characterTargetableNode(targetable, r)[0] == 0 && kind != 10 && kind != 11 {
		kind = -1
	}
	// 0x537518 dispatch on kind (unsigned bound 8).
	switch kind {
	case 0, 1:
		return []int64{2}
	case 2:
		return []int64{4}
	case 5:
		return []int64{1}
	case 7:
		if spell < 0 || spell > 255 {
			panic("character action target: spell id outside the spell table")
		}
		return itemSpellTargetTypeNode([]int64{1, 0, spell, 0, characterActionSpellTarget(in, spell), in[2]}, r)
	case 8:
		if !list {
			return []int64{0}
		}
		index := in[24+source]
		if index == -1 {
			return []int64{0}
		}
		if index < 0 || index > 818 {
			panic("character action target: item index outside the item table")
		}
		itemSpell, itemType := int64(0), int64(0)
		for k := 0; k < 5; k++ { // fixture write order
			if in[24+k] == index {
				itemSpell, itemType = in[29+k], in[34+k]
			}
		}
		if itemSpell == 0 { // 0x5374cd
			return []int64{0}
		}
		return itemSpellTargetTypeNode([]int64{1, index, itemSpell, itemType, characterActionSpellTarget(in, itemSpell), in[2]}, r)
	}
	return []int64{0}
}

// characterActionSpellTarget reads spell record id's target field from the fixture memory (last writer wins).
func characterActionSpellTarget(in []int64, id int64) int64 {
	ids := [11]int64{in[29], in[30], in[31], in[32], in[33], in[14], in[17], in[19], in[4], in[21], in[22]}
	target := int64(0)
	for k, v := range ids {
		if v == id {
			target = in[39+k]
		}
	}
	return target
}
