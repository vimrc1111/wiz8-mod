package calc

import "fmt"

// The replay fixture has 150 loaded spell records (0x96, rejected above by 0x5001e0), eight party slots and room
// for eight independent monster records. spellTargetKind is the selected loaded
// record's value, not a table copied from the executable (0x4ac9e8).
func confusedSpellRandomTargetNode(in []int64, r Rand) []int64 {
	if len(in) != 58 {
		panic("confused spell target: expected 58 inputs")
	}
	for i, v := range in {
		limit := int64(4294967295) // u32 fields read by 0x4fedc0 and its children.
		if i >= 10 && i < 18 || i >= 18 && ((i-18)%5 == 2 || (i-18)%5 == 3) {
			limit = 255 // byte flags: 0x4fedd3, 0x4fee1a, 0x4fee24.
		}
		if v < 0 || v > limit {
			panic("confused spell target: field outside storage width")
		}
	}
	if in[0] > 149 || in[9] > 8 {
		panic("confused spell target: fixture record index or list count outside capacity")
	}
	// These source sentinels trigger the original assertion children, not a
	// target-selection result (0x53beab and 0x53bf1b).
	if in[2] == 1 && in[3] == 4294967295 || in[2] == 2 && in[4] == 4294967295 {
		panic("confused spell target: invalid source sentinel")
	}
	out := []int64{0, in[5], in[6], in[7], in[8]}
	type candidate struct {
		character        bool
		index, id, group int64
	}
	choices := make([]candidate, 0, 16)
	// Eight slots, stride 0x106 and loop endpoint 0x830: 0x4fedee.
	for i := 0; i < 8; i++ {
		if in[10+i] != 0 && !(in[2] == 1 && in[3] == int64(i)) {
			choices = append(choices, candidate{character: true, index: int64(i)})
		}
	}
	for i := 0; i < int(in[9]); i++ {
		m := in[18+5*i : 23+5*i]
		// Present, active and unsigned HP must all be nonzero (0x4fee1a..
		// 0x4fee30). Source kind 2 excludes matching IDs (0x53bf10).
		if m[2] != 0 && m[3] != 0 && m[4] != 0 && !(in[2] == 2 && in[4] == m[0]) {
			choices = append(choices, candidate{id: m[0], group: m[1]})
		}
	}
	if len(choices) == 0 {
		return out // 0x4fef84: failure preserves every target field.
	}
	c := choices[r.R(len(choices))] // 0x4fee64: one draw, in list order.
	out[0] = 1
	if c.character {
		if in[1] == 4 { // Group spell target code: 0x4fef21.
			out[1] = 2 // Whole party: 0x4fef2d.
		} else {
			out[1], out[2] = 1, c.index // Single character: 0x4fef37.
		}
	} else if in[1] == 4 { // Group spell target code: 0x4fef56.
		out[1], out[4] = 4, c.group // Monster group: 0x4fef5f.
	} else {
		out[1], out[3] = 3, c.id // Single monster: 0x4fef72.
	}
	return out
}

func init() {
	in := []string{"spellIndex", "spellTargetKind", "sourceKind", "sourceCharacterIndex", "sourceMonsterID",
		"targetKind", "targetCharacterIndex", "targetMonsterID", "targetGroup", "monsterCount"}
	for i := 0; i < 8; i++ {
		in = append(in, fmt.Sprintf("partyPresent%d", i))
	}
	for i := 0; i < 8; i++ {
		for _, suffix := range []string{"ID", "Group", "Present", "Active", "HP"} {
			in = append(in, fmt.Sprintf("monster%d%s", i, suffix))
		}
	}
	Nodes["004fedc0"] = Node{Addr: "004fedc0", In: in,
		Out: []string{"success", "targetKind", "targetCharacterIndex", "targetMonsterID", "targetGroup"},
		F:   confusedSpellRandomTargetNode}
}
