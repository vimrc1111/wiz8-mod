package calc

// Inputs describe the selected item and character records, followed by the
// instance identification byte. Only AL is defined by 0x500010..0x500056.
func itemMagicUsableNode(in []int64, r Rand) []int64 {
	if len(in) != 67 {
		panic("item magic usable: expected 67 inputs")
	}
	if in[0] < 0 || in[0] > 4294967295 || in[15] < 0 || in[15] > 255 || in[66] < 0 || in[66] > 255 {
		panic("item magic usable: invalid index or byte field")
	}
	// Kind 3 is a scroll (0x500030); +6 is identification, not charges
	// (0x500039, with name selection evidence at 0x51b5dc..0x51b656).
	if in[15] != 3 || in[66] == 0 {
		return []int64{0}
	}
	// The scroll path never reads instrument/device levels or spell level.
	// Reinsert those unused inputs to share the 0x51d610 qualification formula.
	use := make([]int64, 69)
	copy(use[:17], in[:17])
	use[19] = in[17]
	copy(use[21:], in[18:66])
	// The child's 0..3 selector is only an arena addressing limit; record
	// contents are already inputs. Normalize without narrowing item indices.
	use[0] = 0
	return itemUseRequirementsNode(use, r)
}

func init() {
	Nodes["00500010"] = Node{Addr: "00500010", In: []string{"itemIndex", "profession", "race", "gender", "professionMask", "raceMask", "genderMask", "attributeId0", "attributeMinimum0", "attributeId1", "attributeMinimum1", "skillId0", "skillMinimum0", "skillId1", "skillMinimum1", "itemKind", "spellId", "learnedState", "strength", "intelligence", "piety", "vitality", "dexterity", "speed", "senses", "skill0", "skill1", "skill2", "skill3", "skill4", "skill5", "skill6", "skill7", "skill8", "skill9", "skill10", "skill11", "skill12", "skill13", "skill14", "skill15", "skill16", "skill17", "skill18", "skill19", "skill20", "skill21", "skill22", "skill23", "skill24", "skill25", "skill26", "skill27", "skill28", "skill29", "skill30", "skill31", "skill32", "skill33", "skill34", "skill35", "skill36", "skill37", "skill38", "skill39", "skill40", "identified"}, Out: []string{"magicUsable"}, F: itemMagicUsableNode}
}
