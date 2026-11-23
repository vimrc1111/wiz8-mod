package calc

import "fmt"

// itemSpellUsageCountsNode is 0x501d20. Inputs, in order: charIndex (arg 2),
// gameStarted, itemDefinitionCount, profession, race, gender, instrumentLevel,
// deviceLevel, 7 attributes, 41 skills, 150 initial spell counters (table at
// 0x687614, stride 16), 150 initial use counters (table at 0x687610, stride 16),
// then 20 slots (12 equipment instances from character +0xf5d, 8 pack instances
// from +0x1029, stride 12) of itemSpellSlotFields values each.
// Outputs: the 150 spell counters, then the 150 use counters.
//
// Spell ids are limited to 0..149 (the 150-entry counter table 0x687614..0x687f74
// and the 150-byte local flag buffer); the game asserts spell numbers < 0x96
// elsewhere (0x54733c). charIndex selects a 16-byte record of the use table;
// the table has 150 records (0x687610..0x687f74), so that is its input range.
//
// Item definition records are shared by index and spell-keyed character and spell
// table entries by spell id: when two slots name the same definition index (or the
// same spell id) the later slot's values are what memory holds. The loops below
// replay that, then each slot runs 0x51d800 through itemActivationAllowedNode.
const (
	itemSpellSlotFields = 20
	itemSpellSlotBase   = 356
	itemSpellSlots      = 20
	itemSpellEquipSlots = 12
	itemSpellCharBase   = 0x20000000 // arena base the case uses for the character
)

func itemSpellUsageCountsNode(in []int64, r Rand) []int64 {
	if len(in) != itemSpellSlotBase+itemSpellSlots*itemSpellSlotFields {
		panic("item spell usage: wrong input count")
	}
	charIndex := in[0]
	if charIndex < 0 || charIndex >= 150 {
		panic("item spell usage: character index outside the 150-record table")
	}
	avail := make([]int64, 150)
	use := make([]int64, 150)
	copy(avail, in[56:206])
	copy(use, in[206:356])

	slot := func(k int) []int64 { return in[itemSpellSlotBase+k*itemSpellSlotFields:][:itemSpellSlotFields] }
	// Slot field indexes: 0 def, 1 charges, 2..17 definition record (masks,
	// requirements, kind, spell, type, flags, charge mode), 18 learned, 19 level.
	defs := map[int64][]int64{}
	learned, level := map[int64]int64{}, map[int64]int64{}
	for k := 0; k < itemSpellSlots; k++ {
		s := slot(k)
		if s[14] < 0 || s[14] > 149 {
			panic("item spell usage: spell id outside the counter table")
		}
		if s[0] >= 0 {
			defs[s[0]] = s[2:18]
		}
		learned[s[14]] = s[18]
		level[s[14]] = s[19]
	}

	var flag [150]bool
	for k := 0; k < itemSpellSlots; k++ {
		s := slot(k)
		if s[0] < 0 {
			continue // empty instance: 0x501d51 / 0x501db0
		}
		d := defs[s[0]]
		spell := d[12]
		if spell == 0 {
			continue // definition without a spell: 0x501d6f / 0x501dd0
		}
		addr := int64(itemSpellCharBase)
		if k < itemSpellEquipSlots {
			addr += 0xf5d + 12*int64(k)
		} else {
			addr += 0x1029 + 12*int64(k-itemSpellEquipSlots)
		}
		vec := []int64{s[0], in[3], in[4], in[5]}
		vec = append(vec, d[0:11]...)   // masks, 4 requirement pairs
		vec = append(vec, d[11], d[12]) // kind, spell
		vec = append(vec, in[6], in[7], learned[spell], level[spell])
		vec = append(vec, in[8:56]...) // 7 attributes, 41 skills
		vec = append(vec, d[13], in[1], d[14], d[15], s[1], addr, itemSpellCharBase, in[2])
		if itemActivationAllowedNode(vec, r)[0] == 0 {
			continue
		}
		// Charge modes 4 and 2 need a nonzero instance charge byte: 0x501d81..0x501d91.
		if (d[15] == 4 || d[15] == 2) && s[1] == 0 {
			continue
		}
		flag[spell] = true
	}
	for i := range avail {
		if flag[i] {
			avail[i] = int64(uint32(avail[i] + 1))
		}
	}
	use[charIndex] = int64(uint32(use[charIndex] + 1))
	return append(avail, use...)
}

func init() {
	names := []string{"charIndex", "gameStarted", "itemDefinitionCount", "profession", "race", "gender", "instrumentLevel", "deviceLevel",
		"strength", "intelligence", "piety", "vitality", "dexterity", "speed", "senses"}
	for i := 0; i < 41; i++ {
		names = append(names, fmt.Sprintf("skill%d", i))
	}
	var out []string
	for i := 0; i < 150; i++ {
		names = append(names, fmt.Sprintf("spellCountInit%d", i))
		out = append(out, fmt.Sprintf("spellCount%d", i))
	}
	for i := 0; i < 150; i++ {
		names = append(names, fmt.Sprintf("useCountInit%d", i))
		out = append(out, fmt.Sprintf("useCount%d", i))
	}
	fields := []string{"DefIndex", "Charges", "ProfessionMask", "RaceMask", "GenderMask", "AttributeId0", "AttributeMinimum0",
		"AttributeId1", "AttributeMinimum1", "SkillId0", "SkillMinimum0", "SkillId1", "SkillMinimum1", "ItemKind", "SpellId",
		"ItemType", "ActivationFlags", "ChargeMode", "LearnedState", "SpellLevel"}
	for k := 0; k < itemSpellSlots; k++ {
		for _, f := range fields {
			names = append(names, fmt.Sprintf("slot%d%s", k, f))
		}
	}
	Nodes["00501d20"] = Node{Addr: "00501d20", In: names, Out: out, F: itemSpellUsageCountsNode}
}
