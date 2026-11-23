package calc

import "math"

// Fixture wiring: eight independently indexed character/action records, one
// group in either active/reserve collection, two distinct monster members,
// and two item records. Contexts 0/1 read independent action kinds; their
// action indices and item spell IDs share the selected spell input. Context 3
// uses the queued spell, 4 the item, 5 the fixed long range, 7 the battle spell,
// and 8 the default main hand. Context 2 selects the special spell/item;
// context 6 resolves to context 0 with mode 0 and selected character -1.
// Feedback executes against a populated duplicate-message mask; its return
// and UI bookkeeping do not participate in the target validity flag.
func monsterGroupTargetValidNode(in []int64, _ Rand) []int64 {
	if len(in) != 34 {
		panic("monster group target: expected 34 inputs")
	}
	result := func(v bool) []int64 {
		if v {
			return []int64{1}
		}
		return []int64{0}
	}
	// 0x5100b0 searches both collections by ID before 0x5101b0 resolves
	// the resulting index. No group activity/HP/visibility test is performed.
	if (in[4] == 0 && in[5] == 0) || uint32(in[1]) != uint32(in[6]) {
		return result(false)
	}
	if queuedSpellRuleErr != nil {
		panic(queuedSpellRuleErr)
	}
	t := queuedSpellRuleData                                // Runtime-loaded 0x603ac8, 0x5ebb34, 0x5ebc40.
	alive := in[7] != 0 && in[8] != 0 && uint32(in[9]) < 18 // 0x53c270.
	rangeClass := int64(-1)
	if alive {
		weapon := func(kind, available, reach int64) int64 {
			if available == 0 {
				return -1
			}
			// 0x519b53: attack kinds 1/3 use I+0x47; others are short.
			if kind == 1 || kind == 3 {
				return reach
			}
			return 0
		}
		category := int64(0)
		switch in[2] {
		case 0, 6:
			category = in[32]
		case 1:
			category = in[33]
		case 2, 3, 7:
			category = 7
		case 4:
			category = 8
		case 5:
			category = 2
		case 8:
			category = 0
		default:
			panic("monster group target: invalid attack context")
		}
		// The special item route also reads the same embedded spell ID.
		// 0x519a3d selects a weapon by action kind, rather than action index.
		switch category {
		case 0:
			rangeClass = weapon(in[25], in[26], in[29])
		case 1:
			rangeClass = weapon(in[27], in[28], in[30])
		case 2:
			rangeClass = 2 // 0x519a8c.
		case 5:
			rangeClass = 0 // 0x519a87.
		case 7, 8:
			if in[10] != 0 { // 0x519a4e, 0x52084f: zero has no spell range.
				rangeClass = in[11]
			}
		}

	}
	// Literal lengths from 0x51a9b7..0x51a9d0, scaled by the user's exe.
	var units float64
	switch rangeClass {
	case -1:
		units = 0
	case 0:
		units = 2
	case 1:
		units = 4
	case 2:
		units = 25
	case 3:
		units = 50
	default:
		panic("monster group target: invalid range class")
	}
	reach := float32(units * t.scale) // 0x537859 stores ST0 as float32.
	f := func(i int) float64 { return float64(math.Float32frombits(uint32(in[i]))) }
	partyY := float64(float32(f(14) - t.height)) // 0x4c7cf4.
	distance := func(x, y, z, radius int) float32 {
		dx, dy, dz := f(x)-f(13), f(y)-partyY, f(z)-f(15)
		d := math.Sqrt(dz*dz+dy*dy+dx*dx) - f(radius) - f(20)
		// 0x4c7d35 clamps below-minimum and unordered x87 comparisons.
		if !(d >= t.minimum) {
			d = t.minimum
		}
		return float32(d) // 0x5378e9, before the inclusive comparison.
	}
	for i := int64(0); i < in[12]; i++ {
		d := distance(16, 17, 18, 19)
		if i != 0 {
			d = distance(21, 22, 23, 24)
		}
		if d <= reach || math.IsNaN(float64(d)) || math.IsNaN(float64(reach)) {
			return result(true) // 0x5378f7: equal/less/unordered succeeds.
		}
	}
	return result(false)
}

func init() {
	Nodes["00519920"] = Node{
		Addr: "00519920",
		In:   []string{"characterSlot", "targetGroupId", "attackContext", "showFailure", "activeGroupCount", "reserveGroupCount", "storedGroupId", "characterPresent", "characterHP", "characterCondition", "spellId", "spellRange", "memberCount", "partyX", "partyY", "partyZ", "member1X", "member1Y", "member1Z", "member1Radius", "partyRadius", "member2X", "member2Y", "member2Z", "member2Radius", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainWeaponRange", "offWeaponRange", "specialUsesSpell", "mainActionKind", "offActionKind"},
		Out:  []string{"valid"}, F: monsterGroupTargetValidNode,
	}
}
