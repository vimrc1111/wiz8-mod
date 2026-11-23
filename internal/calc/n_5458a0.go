package calc

// Ability 20: 0x5458c0; the union of current class/race features is
// 0x547940 (notes/re/abilities.md:11). breathTables loads both feature
// tables from the player's executable at runtime, without embedding data.
const berserkFeatureCode = 20

// Inputs describe the selected party slot and four independent item records.
// Character stride 6242: 0x5458a5..0x5458be; hand stride 91:
// 0x54591c. The return is the boolean AL; upper EAX is unspecified.
func berserkAvailableNode(in []int64, r Rand) []int64 {
	if len(in) != 14 || in[0] < 0 || in[0] > 7 || in[1] < -1 || in[1] > 14 || in[2] < -1 || in[2] > 15 {
		panic("berserk: expected party slot 0..7, class -1..14, race -1..15 and 14 inputs")
	}
	if breathTablesErr != nil {
		panic(breathTablesErr)
	}
	// Validate the same hand/item inputs as the original range adapter,
	// including a nonzero in-party flag for the unavailable-hand diagnostic.
	rangeIn := append([]int64{0}, in[3:]...)
	mainRange := charAttackRangeNode(rangeIn, r)[0]
	hasFeature := false
	if in[1] != -1 {
		for _, feature := range breathTables.class[in[1]] {
			hasFeature = hasFeature || feature == berserkFeatureCode
		}
	}
	if in[2] != -1 {
		for _, feature := range breathTables.race[in[2]] {
			hasFeature = hasFeature || feature == berserkFeatureCode
		}
	}
	if !hasFeature {
		return []int64{0}
	}
	// 0x5458f9..0x545922 requires an enabled hand with range != -1.
	// Enabled hands in 0x519ac0 return either zero or an unsigned item byte,
	// so availability alone suffices for this guard.
	if in[5] == 0 && in[7] == 0 {
		return []int64{0}
	}
	// 0x545932 always queries hand 0, even when only hand 1 is enabled.
	// The signed threshold at 0x54593c accepts range -1 in that case.
	if mainRange <= 1 {
		return []int64{1}
	}
	return []int64{0}
}

func init() {
	Nodes["005458a0"] = Node{
		Addr: "005458a0",
		In: []string{"characterIndex", "currentClass", "race", "characterInParty",
			"mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable",
			"mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range"},
		Out: []string{"berserkAvailable"}, F: berserkAvailableNode,
	}
}
