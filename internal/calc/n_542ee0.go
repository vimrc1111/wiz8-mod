package calc

// defhitNode composes the target's base defense with the existing adjustment
// node. Its first 26 inputs are exactly that node's explicit runtime fields.
func defhitNode(in []int64, r Rand) []int64 {
	if len(in) != 28 || in[26] < -2147483648 || in[26] > 2147483647 || in[27] < -128 || in[27] > 127 {
		panic("defhit: expected 26 adjustment inputs, i32 character defense, i8 monster AC")
	}
	adjustment := int32(defenseAdjustmentNode(in[:26], r)[0])
	base := int32(in[26]) // Character defense encoding, exe 0x542f6a; C+0xe85.
	if in[0] == 3 {
		base = 10 - int32(in[27]) // Monster AC encoding, exe 0x542f11..0x542f26.
		if in[13] != 0 && base > -5 {
			base = -5 // Status defense cap, exe 0x542f2d..0x542f38.
		}
	}
	// Original addition returns a signed 32-bit value, including wraparound.
	return []int64{int64(base + adjustment)}
}

// The first 26 inputs are exactly defenseAdjustmentNode's explicit runtime
// fields, in the same order; the last two are this node's own base values.
// Listed explicitly so registration does not depend on Go init order across
// files (n_542ee0.go initializes before n_5468d0.go alphabetically).
func init() {
	Nodes["00542ee0"] = Node{Addr: "00542ee0", In: []string{
		"targetType", "mode", "characterIndex", "combat", "ambush", "surprise",
		"speedDefense", "shieldDefense", "characterDefenseSuppressed", "monsterDefense",
		"monsterHitDefense", "fatigue", "monsterPosture", "monsterDefenseSuppressed",
		"ambushDefenseMask", "completedActionSelector", "completedPosture", "pendingPosture",
		"sceneFlags", "HP", "status", "actionCompleted", "mainAction", "offAction",
		"monsterID", "templateIndex",
		"characterDefHit", "monsterAC",
	}, Out: []string{"defHit"}, F: defhitNode}
}
