package calc

// defpenNode is the penetration defense: defHit (node 0x542ee0) plus a
// target-kind specific base-vs-location difference plus the adjustment U
// (node 0x5468d0), so U is counted twice (notes/re/combat-2.md:28).
// Inputs 0..27 are defhitNode's; the rest are this node's own fields.
func defpenNode(in []int64, r Rand) []int64 {
	if len(in) != 53 {
		panic("defpen: expected 53 inputs")
	}
	loc, mode := in[28], in[1]
	if loc < 0 || loc > 6 { // Hit location 0..6, notes/re/combat-2.md:28.
		panic("defpen: hit location outside 0..6")
	}
	hit := int32(defhitNode(in[:28], r)[0])
	u := int32(defenseAdjustmentNode(in[:26], r)[0])
	var diff int32
	if in[0] == 3 {
		if mode > 8 { // Template mode table has 9 entries (+0x16d), combat-2.md:113.
			panic("defpen: monster attack mode outside 0..8")
		}
		// Runtime +0x1e0 - template mode entry - template location entry + template AC (+0x15d); exe 0x543002..0x54302d.
		diff = int32(in[29]) - int32(in[30+mode]) - int32(in[39+loc]) + int32(in[27])
	} else {
		// Character location entry (C+0xec1+4*loc) minus C+0xe85; exe 0x543064..0x543073.
		diff = int32(in[46+loc]) - int32(in[26])
	}
	return []int64{int64(u + diff + hit)} // 32-bit wrap as in the original adds.
}

func init() {
	in := []string{
		"targetType", "mode", "characterIndex", "combat", "ambush", "surprise",
		"speedDefense", "shieldDefense", "characterDefenseSuppressed", "monsterDefense",
		"monsterHitDefense", "fatigue", "monsterPosture", "monsterDefenseSuppressed",
		"ambushDefenseMask", "completedActionSelector", "completedPosture", "pendingPosture",
		"sceneFlags", "HP", "status", "actionCompleted", "mainAction", "offAction",
		"monsterID", "templateIndex", "characterDefHit", "monsterAC",
		"hitLocation", "monsterPenDefense",
	}
	for k := 0; k < 9; k++ {
		in = append(in, "monsterModeAC"+string(rune('0'+k)))
	}
	for k := 0; k < 7; k++ {
		in = append(in, "monsterLocAC"+string(rune('0'+k)))
	}
	for k := 0; k < 7; k++ {
		in = append(in, "characterLocDef"+string(rune('0'+k)))
	}
	Nodes["00542fc0"] = Node{Addr: "00542fc0", In: in, Out: []string{"defPen"}, F: defpenNode}
}
