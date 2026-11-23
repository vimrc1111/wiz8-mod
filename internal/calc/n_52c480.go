package calc

import "strconv"

// Eight slots, main-condition cutoff 18, initial priority 999 and character
// stride 0x1862: 0x52c4c7, 0x52c4a7, 0x52c495 and 0x52c4c1 respectively.
// C+0xf21 is named for its item-target selection use at 0x52346b; its UI
// attribute name is unverified. No executable data table is used here.
const conditionItemTargetSlots = 8
const conditionItemTargetConditionLimit = 18
const conditionItemTargetPriorityLimit = 999
const conditionItemTargetCharacterStride = 0x1862

// Inputs are the character-array pointer, initial ECX and, per slot, the
// presence byte, unsigned main condition and unsigned item-target priority.
// Initial ECX is temporary storage only: any selected slot replaces it;
// otherwise the original returns null (0x52c482, 0x52c4b8, 0x52c4d7).
// The case's dynamic arena offsets follow characterBase so relocated records
// and the returned pointer are checked together. Address arithmetic is u32.
func conditionItemTargetSelectNode(in []int64, r Rand) []int64 {
	if len(in) != 2+3*conditionItemTargetSlots {
		panic("condition item target: expected base, ECX and eight character triples")
	}
	for i, v := range in {
		limit := int64(4294967295) // u32 fields at 0x52c4a7 and 0x52c4b0.
		if i >= 2 && (i-2)%3 == 0 {
			limit = 255 // u8 presence at 0x52c4a2.
		}
		if v < 0 || v > limit {
			panic("condition item target: expected u32 base/ECX/condition/priority and u8 presence")
		}
	}
	best, selected := int64(conditionItemTargetPriorityLimit), -1
	for slot := 0; slot < conditionItemTargetSlots; slot++ {
		i := 2 + 3*slot
		if in[i] != 0 && in[i+1] < conditionItemTargetConditionLimit && in[i+2] < best {
			best, selected = in[i+2], slot
		}
	}
	if selected < 0 {
		return []int64{0}
	}
	return []int64{int64(uint32(in[0]) + uint32(selected*conditionItemTargetCharacterStride))}
}

func init() {
	names := []string{"characterBase", "initialEcX"}
	for slot := 0; slot < conditionItemTargetSlots; slot++ {
		suffix := strconv.Itoa(slot)
		names = append(names, "present"+suffix, "mainCondition"+suffix, "itemTargetPriority"+suffix)
	}
	Nodes["0052c480"] = Node{
		Addr: "0052c480", In: names, Out: []string{"targetCharacter"}, F: conditionItemTargetSelectNode,
	}
}
