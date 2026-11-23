package calc

import "fmt"

func init() {
	names := []string{"monsterBusy", "monsterPartyScan", "monsterDisposition", "monsterTemplateID", "monsterCharmDuration", "skipHostileScan"}
	for i := 0; i < 8; i++ {
		for _, n := range []string{"slotActive", "currentHP", "mainStatus", "charm"} {
			names = append(names, fmt.Sprintf("%s[%d]", n, i))
		}
	}
	names = append(names, "listCount")
	for k := 0; k < 3; k++ {
		for _, n := range []string{"id", "templateID", "disposition", "active", "visible", "currentHP", "status", "charmDuration"} {
			names = append(names, fmt.Sprintf("obj%d.%s", k, n))
		}
	}
	for t := 0; t < 4; t++ {
		names = append(names, fmt.Sprintf("template%d.faction", t))
	}
	names = append(names, "relationCount")
	for j := 0; j < 4; j++ {
		names = append(names, fmt.Sprintf("relation%d.id", j), fmt.Sprintf("relation%d.blocks", j))
	}
	Nodes["00534690"] = Node{Addr: "00534690", In: names, Out: []string{"actionAllowed"}, F: monsterActionClearanceNode}
}

// monsterActionClearanceNode: 0x534690. Four loaded templates, three other
// listed monsters (the acting monster is entry 1 of 4) and four relation
// records are the case's arena capacity, not game limits. Party scan covers
// all eight slots (0x5346f9..0x534705). Template IDs are 0..3 or the special
// 0x224 (0x546f88); other IDs would load a template from disk and are not
// modelled.
func monsterActionClearanceNode(in []int64, r Rand) []int64 {
	if len(in) != 6+32+1+24+4+9 {
		panic("monster action clearance: expected 76 fields")
	}
	pos := 0
	take := func(max int64) int64 {
		v := in[pos]
		pos++
		if v < 0 || v > max {
			panic("monster action clearance: field out of range")
		}
		return v
	}
	const u32 = 0xffffffff
	busy, partyScan, disp := take(255), take(255), take(255)
	tmpl, charmDur, skip := take(u32), take(u32), take(u32)
	type charRec struct{ slot, hp, status, charm int64 }
	var chars [8]charRec
	for i := range chars {
		chars[i].slot, chars[i].hp, chars[i].status = take(255), take(u32), take(u32)
		pos++ // charm is i32
		chars[i].charm = in[pos-1]
		if chars[i].charm < -2147483648 || chars[i].charm > 2147483647 {
			panic("monster action clearance: charm out of range")
		}
	}
	count := take(4)
	type monRec struct{ id, tmpl, disp, active, visible, hp, status, charm int64 }
	var objs [3]monRec
	for k := range objs {
		o := &objs[k]
		o.id, o.tmpl, o.disp, o.active = take(u32), take(u32), take(255), take(255)
		o.visible, o.hp, o.status, o.charm = take(255), take(u32), take(u32), take(u32)
	}
	var faction [4]int64
	for t := range faction {
		faction[t] = take(u32)
	}
	relCount := take(4)
	var relID, relBlocks [4]int64
	for j := range relID {
		relID[j], relBlocks[j] = take(u32), take(255)
	}

	// 0x53469c: an acting monster is never cleared.
	if busy != 0 {
		return []int64{0}
	}
	// 0x5346a9..0x5346ed: with the party flag, a usable character (present,
	// HP > 0, main status < 18) whose effective allegiance toward this monster
	// is hostile (code 1, after charm swap by 0x546f10) vetoes the action.
	if partyScan != 0 {
		for i, c := range chars {
			if c.slot == 0 || c.hp == 0 || c.status >= 0x12 {
				continue
			}
			if targetDispositionNode([]int64{int64(i), c.charm, disp}, r)[0] == 1 {
				return []int64{0}
			}
		}
	}
	if skip != 0 { // 0x53470d
		return []int64{1}
	}
	// 0x546f80 returning 1: different nonzero allegiances, neither template ID
	// the special one, unless both templates share a nonzero group and the
	// two monsters agree on being charmed (0x546fd9..0x546ff9).
	tmplFaction := func(id int64) int64 {
		if id < 0 || id > 3 {
			panic("monster action clearance: template not loaded")
		}
		return faction[id]
	}
	hostile := func(o monRec) bool {
		if tmpl == 0x224 || o.tmpl == 0x224 {
			return false
		}
		if disp == o.disp || disp == 0 || o.disp == 0 {
			return false
		}
		a, b := tmplFaction(tmpl), tmplFaction(o.tmpl)
		if a == 0 || b == 0 || a != b {
			return true
		}
		return (charmDur != 0) != (o.charm != 0)
	}
	entries := []int{0, -1, 1, 2} // -1: the acting monster itself (0x53472f)
	for e := int64(0); e < count; e++ {
		if entries[e] < 0 {
			continue
		}
		o := objs[entries[e]]
		if o.active == 0 || o.hp == 0 || o.status >= 0x12 || o.visible == 0 || !hostile(o) {
			continue
		}
		// 0x505d20: first relation record naming this monster decides.
		for j := int64(0); j < relCount; j++ {
			if relID[j] == o.id {
				if relBlocks[j] != 0 {
					return []int64{0}
				}
				break
			}
		}
	}
	return []int64{1}
}
