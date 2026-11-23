package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Layout follows 0x54d100 and its children: nine 17-byte effects,
// 100 synthetic spell definitions (447-byte stride), two target monsters,
// 103-byte modifier buffers. Characters and visual attachment lists are empty
// fixtures; the original rebuild and visual cleanup functions still execute.
// Outputs include complete effect records, rebuilt modifiers and monster
// attributes. Rendering state and secondary floating combat caches are outside
// this rule interface. Monster status 19 is absent: it is a character NPC state.
var dispelCombatSlots, dispelCombatSlotsErr = loadDispelCombatSlots()

func loadDispelCombatSlots() ([9]uint32, error) {
	var t [9]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("dispel: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	// 0x616db4..0x616dd8: combat effect slot spell IDs; never embed the table.
	off, err := peOffset(b, 0x616db4)
	if err != nil {
		return t, err
	}
	if len(b)-off < 36 {
		return t, fmt.Errorf("dispel: truncated slot table")
	}
	for i := range t {
		t[i] = binary.LittleEndian.Uint32(b[off+4*i:])
		if t[i] >= 100 {
			return t, fmt.Errorf("dispel: unknown spell ID")
		}
	}
	return t, nil
}

func init() {
	in := []string{"power", "bonus", "partyTarget", "monsterCount", "changed"}
	out := []string{"changed"}
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
	Nodes["0054d100"] = Node{Addr: "0054d100", In: in, Out: out, F: dispelCombatSpellNode}
}

// Rebuild the party or monster shared sources via the existing exact node.
// All eight character slots are absent in this fixture, so no character-derived
// modifier or cache participates. 0x50e8c0 uses the same shared/aura rules.
func dispelSharedModifiers(shared, aura, effects []int64, combat int64, r Rand) []int64 {
	a := make([]int64, 2620)
	a[0] = combat
	copy(a[1:49], shared)
	copy(a[76:100], aura)
	for i := 0; i < 9; i++ {
		copy(a[49+3*i:52+3*i], effects[5*i:5*i+3])
		// Rebuild reads only the low byte of the u32 effect power.
		a[51+3*i] = int64(uint8(a[51+3*i]))
	}
	return partyEffectRebuildNode(a, r)[:103]
}

func dispelMonsterModifiers(m []int64, r Rand) []int64 {
	// Per monster: presence, 45 effect fields, 20 durations, poison,
	// 24 buff fields, 48 shared fields, 24 aura fields, 103 modifiers, 5 attrs.
	out := dispelSharedModifiers(m[91:139], m[139:163], m[1:46], m[0], r)
	add := func(k int, v int64) { out[k] = int64(uint8(out[k] + v)) }
	for i := 3; i < 19; i++ {
		if m[46+i] == 0 {
			continue
		}
		// 0x50eac0, with null character: feature/NPC branches do not apply.
		switch statusRebuildTables.dispatch[i-3] {
		case 0x50eafb:
			add(1, -2)
			add(0x4b, -2)
		case 0x50eb11:
			add(1, -5)
			for k := 0xc; k < 0x3c; k++ {
				add(k, -20)
			}
		case 0x50eb4a:
			add(1, -5)
			add(0x4b, -4)
		case 0x50eb60:
			add(1, -3)
			add(0x4b, -2)
		case 0x50eb76:
			add(0x11, -50)
		case 0x50eb7f:
			add(1, -2)
			add(0x4b, -2)
			add(8, m[66])
		case 0x50eba1:
			add(0x12, -50)
			out[0x45] = 1
		case 0x50ebe1:
			add(0x10, -50)
			out[0x45] = 1
		case 0x50ebe5:
			out[0x45] = 1
		case 0x50ebea:
			add(0xd, -50)
		}
	}
	// 0x50ecc0 / 0x517f60: byte power arithmetic, rounded percentage.
	scale := func(v uint8, b int64) int64 { return int64(uint8((uint32(v)*(uint32(b)+100) + 50) / 100)) }
	for i := 5; i < 8; i++ {
		b := m[67+3*i : 70+3*i]
		if b[2] == 0 {
			continue
		}
		p := uint8(b[0])
		switch i {
		case 5:
			add(0x11, scale(p*10, b[1]))
		case 6:
			for k := 0xc; k < 0x13; k++ {
				add(k, scale(p*5, b[1]))
			}
		case 7:
			add(6, scale(p*8, b[1]))
			add(5, scale(p, b[1]))
		}
	}
	return out
}

func dispelCombatSpellNode(in []int64, r Rand) []int64 {
	if len(in) != 872 {
		panic("dispel: expected 872 inputs")
	}
	if dispelCombatSlotsErr != nil {
		panic(dispelCombatSlotsErr)
	}
	if statusRebuildTablesErr != nil {
		panic(statusRebuildTablesErr)
	}
	for _, v := range in {
		if v < 0 || v > 0xffffffff {
			panic("dispel: expected u32 fields")
		}
	}
	if in[3] > 2 || in[4] > 255 {
		panic("dispel: invalid target count or changed byte")
	}
	a := append([]int64(nil), in...)
	power := uint32(a[0])
	// 0x54d139: 9999 bypasses both randomness and scaling. 0x517f90:
	// multiplication wraps to u32 before unsigned truncating division by 100.
	if power != 9999 {
		power++
		roll := r.R(4)
		if roll == 0 && power > 1 || roll == 1 && power <= 2 {
			power++
		}
		power += uint32(power*uint32(a[1])) / 100
	}
	// Clear only fields explicitly cleared by 0x54d205 / 0x54d2cf;
	// the reserved +9 word remains unchanged, including for inactive effects.
	reduce := func(e []int64, amount uint32) bool {
		if uint32(e[4]) > amount {
			e[4] = int64(uint32(e[4]) - amount)
			return false
		}
		e[0], e[1], e[2], e[4] = 0, 0, 0, 0
		return true
	}
	for slot, id := range dispelCombatSlots {
		// 0x54d18f dispatch: only these five combat spell types are dispellable.
		switch id {
		case 48, 76, 80, 81, 93:
		default:
			continue
		}
		// Spell 72's definition level is at +0x7e0e (72*447+0x56).
		amount := uint32(a[5+72]) - uint32(a[5+int(id)]) + power
		if int32(amount) < 1 {
			continue
		}
		e := a[177+5*slot : 182+5*slot]
		if a[2] != 0 && e[0] != 0 {
			if reduce(e, amount) {
				copy(a[222:325], dispelSharedModifiers(a[105:153], a[153:177], a[177:222], 1, r))
			}
			a[4] = 1
		}
		for j := 0; j < int(a[3]); j++ {
			m := a[325+271*j : 596+271*j]
			e = m[1+5*slot : 6+5*slot]
			if e[0] == 0 {
				continue
			}
			if reduce(e, amount) {
				copy(m[163:266], dispelMonsterModifiers(m, r))
				// 0x4e5d00: template attributes plus signed modifier bytes, clamp 1..125.
				for k, off := range []int{0, 1, 4, 5, 6} {
					m[266+k] = int64(min(125, max(1, int(a[867+k])+int(int8(m[163+12+off])))))
				}
			}
			a[4] = 1
		}
	}
	out := []int64{a[4]}
	out = append(out, a[177:325]...)
	for j := 0; j < 2; j++ {
		m := a[325+271*j : 596+271*j]
		out = append(out, m[1:46]...)
		out = append(out, m[163:271]...)
	}
	return out
}
