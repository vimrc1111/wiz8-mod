package calc

import "fmt"

// Interface order follows re/emu/cases/0050e5c0.json. Character stride
// 0x1862 and 103-byte buffers come from 0x50e5c0; item stride 0x10d
// and twelve equipment slots come from 0x50e980 / 0x4edc60.
func init() {
	inputs := []string{"partySlot", "inParty", "class", "race", "gender", "level"}
	array := func(dst *[]string, name string, count int) {
		for i := 0; i < count; i++ {
			*dst = append(*dst, fmt.Sprintf("%s[%d]", name, i))
		}
	}
	array(&inputs, "classLevel", 15)
	attributes := []string{"Strength", "Intelligence", "Piety", "Vitality", "Dexterity", "Speed", "Senses"}
	for _, name := range attributes {
		inputs = append(inputs, "raw"+name)
	}
	array(&inputs, "rawSkill", 41)
	inputs = append(inputs, "timedBuffDuration[1]")
	array(&inputs, "statusRuleModifiers", 103)
	array(&inputs, "partyRuleModifiers", 103)
	array(&inputs, "equipmentItemID", 12)
	fields := []string{"initiative", "toHit", "hitPointRecovery", "staminaRecovery", "spellPointRecovery", "skillID", "skillDelta", "attributeID", "attributeDelta", "resistance[0]", "resistance[1]", "resistance[2]", "resistance[3]", "resistance[4]", "resistance[5]"}
	for item := 0; item < 12; item++ {
		for _, field := range fields {
			inputs = append(inputs, fmt.Sprintf("item[%d].%s", item, field))
		}
	}
	array(&inputs, "equipmentRecordWeight", 12)
	array(&inputs, "equipmentStackCount", 12)
	for i := 0; i < 8; i++ {
		inputs = append(inputs, fmt.Sprintf("backpackItemWeight[%d]", i), fmt.Sprintf("backpackItemCount[%d]", i))
	}
	var outputs []string
	array(&outputs, "statusRuleModifiers", 103)
	array(&outputs, "equipmentRuleModifiers", 103)
	array(&outputs, "totalRuleModifiers", 103)
	for _, name := range attributes {
		outputs = append(outputs, "effective"+name)
	}
	array(&outputs, "effectiveSkill", 41)
	outputs = append(outputs, "loadCapacity", "carriedWeight", "loadCategory", "rebuilt.level")
	array(&outputs, "rebuilt.classLevel", 15)
	Nodes["0050e5c0"] = Node{Addr: "0050e5c0", In: inputs, Out: outputs, F: equipmentEffectRebuildNode}
}

// Equipment and total modifiers are cleared before rebuilding. Status modifiers
// are preserved. Effective attributes, skills and load rules use the same
// output scope as 0x50e650 (notes/re/iface/50e650.tsv); other ability caches and
// notification/formation side effects are not exposed by this interface.
func equipmentEffectRebuildNode(in []int64, r Rand) []int64 {
	if len(in) != 508 {
		panic("equipment rebuild: expected 508 fields")
	}
	if in[0] < 0 || in[0] > 7 || in[1] < 0 || in[1] > 255 {
		panic("equipment rebuild: invalid party slot or membership")
	}
	for _, v := range in[70:276] {
		if v < 0 || v > 255 {
			panic("equipment rebuild: modifier byte overflow")
		}
	}
	for _, v := range in[468:480] {
		if v < 0 || v > 65535 {
			panic("equipment rebuild: item weight overflow")
		}
	}
	for _, v := range in[480:492] {
		if v < 0 || v > 255 {
			panic("equipment rebuild: stack count overflow")
		}
	}
	// 0x50e5ef..0x50e5f6 supplies a freshly cleared equipment buffer.
	equipmentIn := make([]int64, equipmentBufferInput+equipmentBufferSize)
	copy(equipmentIn, in[276:468])
	equipment := make([]int64, 103)
	copy(equipment, equipmentEffectSumNode(equipmentIn, r))
	total := make([]int64, 103)
	// 0x50f090: addition wraps at eight bits, six flags normalize to one,
	// three strengths take unsigned maxima; trailing bytes are not merged.
	merge := func(src []int64) {
		for i := 0; i < 0x4c; i++ {
			switch i {
			case 0x42, 0x43, 0x44, 0x45, 0x46, 0x4a:
				if src[i] != 0 {
					total[i] = 1
				}
			case 0x47, 0x48, 0x49:
				total[i] = max(total[i], src[i])
			default:
				total[i] = int64(uint8(total[i] + src[i]))
			}
		}
	}
	merge(equipment)
	merge(in[70:173])
	if in[1] != 0 {
		merge(in[173:276])
	}
	// Reuse the exact effective-rule formulas and runtime exe tables from
	// 0x50e650, supplying the already merged modifiers with zero active statuses,
	// zero NPC penalties and zero shared modifiers. Synthetic membership only
	// satisfies that adapter's NPC precondition; it cannot add a party effect.
	rebuild := make([]int64, 348)
	rebuild[0], rebuild[1] = in[0], 1
	copy(rebuild[10:77], in[2:69])
	rebuild[103] = in[69] // C+0xa79: second timed buff duration used by feature lookup.
	copy(rebuild[126:229], total)
	copy(rebuild[332:348], in[492:508])
	rebuilt := statusEffectRebuildNode(rebuild, r)
	out := append([]int64(nil), in[70:173]...)
	out = append(out, equipment...)
	out = append(out, total...)
	out = append(out, rebuilt[206:254]...)
	capacity := rebuilt[254]
	// 0x4edc60 counts all twelve equipment slots (including 8/9, which do
	// not contribute modifiers) plus eight backpack slots. 0x51bfd0 uses
	// u16 item weight times max(1,u8 stack count); duplicate IDs count per slot.
	weight := uint32(rebuilt[255])
	for slot, id := range in[276:288] {
		if id != -1 {
			weight += uint32(in[468+int(id)]) * uint32(max(int64(1), in[480+slot]))
		}
	}
	// 0x4edb6e: unsigned 32-bit multiplication before percentage division.
	pct := weight * 100 / uint32(capacity)
	category := int64(4)
	switch {
	case pct < 50:
		category = 0
	case pct < 70:
		category = 1
	case pct < 85:
		category = 2
	case pct <= 100:
		category = 3
	}
	out = append(out, capacity, int64(weight), category)
	// 0x4eda00..0x4eda1f clamps total level and all fifteen class levels.
	for _, v := range in[5:21] {
		out = append(out, min(v, int64(50)))
	}
	return out
}
