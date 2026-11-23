package calc

import "fmt"

func init() {
	in := []string{"spell", "baseDuration", "durationPerPower", "casterType", "casterIndex", "messageStarted", "contribution", "messageEnabled", "templateLevel", "npcDisabled", "power", "bonus", "partyTarget", "monsterCount", "changed"}
	out := []string{"messageStarted", "changed", "contribution", "messageEnabled"}
	array := func(dst *[]string, name string, n int) {
		for i := 0; i < n; i++ {
			*dst = append(*dst, fmt.Sprintf("%s[%d]", name, i))
		}
	}
	records := func(dst *[]string, name string, n int, fields []string) {
		for i := 0; i < n; i++ {
			for _, f := range fields {
				*dst = append(*dst, fmt.Sprintf("%s[%d].%s", name, i, f))
			}
		}
	}
	array(&in, "spellLevel", 100)
	records(&in, "sharedSpell", 12, []string{"active", "spell", "strength", "multiplier"})
	records(&in, "combatAura", 6, []string{"active", "spell", "strength", "multiplier"})
	effectFields := []string{"active", "spell", "strength", "reserved", "duration"}
	records(&in, "partyEffect", 9, effectFields)
	records(&out, "partyEffect", 9, effectFields)
	array(&in, "partyModifier", 103)
	array(&out, "partyModifier", 103)
	for j := 0; j < 2; j++ {
		name := fmt.Sprintf("monster%d", j)
		in = append(in, name+".inCombat")
		records(&in, name+".effect", 9, effectFields)
		records(&out, name+".effect", 9, effectFields)
		array(&in, name+".status", 20)
		in = append(in, name+".poison")
		records(&in, name+".buff", 8, []string{"power", "bonus", "duration"})
		records(&in, name+".shared", 12, []string{"active", "spell", "power", "bonus"})
		records(&in, name+".aura", 6, []string{"active", "spell", "power", "bonus"})
		array(&in, name+".modifier", 103)
		array(&out, name+".modifier", 103)
		array(&in, name+".attribute", 5)
		array(&out, name+".attribute", 5)
	}
	array(&in, "templateAttribute", 5)
	for j := 0; j < 2; j++ {
		array(&in, fmt.Sprintf("monster%d.contribution", j), 8)
		array(&out, fmt.Sprintf("monster%d.contribution", j), 8)
	}
	in = append(in, "casterNpcCharacterFlag", "casterNpcPartyFlag")
	Nodes["0054cdd0"] = Node{Addr: "0054cdd0", In: in, Out: out, F: combatSpellDurationNode}
}

// The fixture uses two distinct monsters, eight absent character slots and
// disabled NPC loading. Shared/aura/status/buff fields are explicit inputs.
// Combat slot IDs come from the existing runtime PE loader for 0x616db4.
// Rendering resources and secondary attack caches are outside rule outputs.
func combatSpellDurationNode(in []int64, r Rand) []int64 {
	if len(in) != 900 {
		panic("combat duration: expected 900 inputs")
	}
	if dispelCombatSlotsErr != nil {
		panic(dispelCombatSlotsErr)
	}
	if statusRebuildTablesErr != nil {
		panic(statusRebuildTablesErr)
	}
	a := append([]int64(nil), in...)
	b := a[10:882]
	if int32(b[3]) > 2 {
		panic("combat duration: fixture has two targets")
	}
	slot := -1
	for k, id := range dispelCombatSlots {
		if uint32(a[0]) == id {
			slot = k
			break
		}
	}
	if slot < 0 {
		panic("combat duration: invalid combat spell")
	}
	// 0x54ce0d..0x54ce68: u32 multiply/add, permanent sentinel,
	// R(4) adjustment, then 0x517f90's wrapped percentage product.
	duration := uint32(a[1]) + uint32(a[2])*uint32(b[0])
	if duration != 9999 {
		duration++
		roll := r.R(4)
		if roll == 0 && duration > 1 || roll == 1 && duration <= 2 {
			duration++
		}
		duration += duration * uint32(b[1]) / 100
	}
	set := func(e []int64) {
		e[0], e[1], e[2], e[4] = 1, a[0], b[0], int64(duration)
		b[4] = 1
	}
	if b[2] != 0 {
		set(b[177+5*slot : 182+5*slot])
		copy(b[222:325], dispelSharedModifiers(b[105:153], b[153:177], b[177:222], 1, r))
		// 0x4eee80 contributes zero with all eight characters absent.
	}
	for j := int32(0); j < int32(b[3]); j++ {
		m := b[325+271*int(j) : 596+271*int(j)]
		if m[0] == 0 {
			// 0x50f720 returns without loading disabled NPC records (+0x29).
			if a[9] == 0 {
				panic("combat duration: NPC loading requires active fixture")
			}
			continue
		}
		e := m[1+5*slot : 6+5*slot]
		if e[0] == 0 || e[1] != a[0] {
			r.R(20)
		} // 0x49f3ba visual phase.
		set(e)
		copy(m[163:266], dispelMonsterModifiers(m, r))
		for k, off := range []int{0, 1, 4, 5, 6} {
			m[266+k] = int64(min(125, max(1, int(b[867+k])+int(int8(m[163+12+off])))))
		}
		// 0x53bea0: type 1 is a direct character; type 3 requires
		// a valid character index and either NPC character/party flag.
		if a[3] == 1 || a[3] == 3 && uint32(a[4]) != 0xffffffff && (a[898] != 0 || a[899] != 0) {
			if a[4] > 7 {
				panic("combat duration: caster index outside fixture")
			}
			k := 882 + 8*int(j) + int(a[4])
			a[k] = int64(uint32(a[k]) + uint32(a[8])*uint32(b[0]))
		}
		a[6] = int64(uint32(a[6]) + 1) // 0x54d070, one successful monster.
	}
	// 0x54cec0..0x54d0e4: any contribution starts the message group.
	if uint32(a[6]) != 0 {
		a[5] = 1
	}
	out := []int64{a[5], b[4], a[6], a[7]}
	out = append(out, b[177:325]...)
	for j := 0; j < 2; j++ {
		m := b[325+271*j : 596+271*j]
		out = append(out, m[1:46]...)
		out = append(out, m[163:271]...)
	}
	return append(out, a[882:898]...)
}
