package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var combatBuffSlots, combatBuffSlotsErr = loadCombatBuffSlots()

func loadCombatBuffSlots() ([9]uint32, error) {
	var slots [9]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return slots, fmt.Errorf("combat buff: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return slots, err
	}
	// 0x54d41d..0x54d42f: nine candidate IDs starting at 0x616dd8.
	off, err := peOffset(b, 0x616dd8)
	if err != nil {
		return slots, err
	}
	if len(b)-off < 36 {
		return slots, fmt.Errorf("combat buff: truncated slot table")
	}
	for i := range slots {
		slots[i] = binary.LittleEndian.Uint32(b[off+4*i:])
	}
	return slots, nil
}

func init() {
	in := []string{"spell", "power", "bonus", "baseDuration", "perPowerDuration", "partyTarget", "monsterCount", "changed", "uiDirty"}
	out := []string{"changed", "uiDirty"}
	records := func(dst *[]string, name string, n int, fields ...string) {
		for i := 0; i < n; i++ {
			for _, f := range fields {
				*dst = append(*dst, fmt.Sprintf("%s[%d].%s", name, i, f))
			}
		}
	}
	array := func(dst *[]string, name string, n int) {
		for i := 0; i < n; i++ {
			*dst = append(*dst, fmt.Sprintf("%s[%d]", name, i))
		}
	}
	records(&in, "sharedSpell", 12, "active", "spell", "strength", "multiplier")
	records(&in, "partyEffect", 9, "active", "spell", "strength", "reserved", "duration")
	array(&in, "partyModifier", 103)
	fields := []string{"active", "spell", "strength", "multiplier", "duration"}
	records(&in, "partyAura", 9, fields...)
	records(&out, "partyAura", 9, fields...)
	array(&out, "partyModifier", 103)
	for j := 0; j < 2; j++ {
		name := fmt.Sprintf("monster%d", j)
		in = append(in, name+".inCombat")
		records(&in, name+".effect", 9, "active", "spell", "strength", "reserved", "duration")
		array(&in, name+".status", 20)
		in = append(in, name+".poison")
		records(&in, name+".buff", 8, "power", "bonus", "duration")
		records(&in, name+".shared", 12, "active", "spell", "power", "bonus")
		records(&in, name+".aura", 9, fields...)
		records(&out, name+".aura", 9, fields...)
		array(&in, name+".modifier", 103)
		array(&out, name+".modifier", 103)
		array(&in, name+".attribute", 5)
		array(&out, name+".attribute", 5)
	}
	array(&in, "templateAttribute", 5)
	Nodes["0054d380"] = Node{Addr: "0054d380", In: in, Out: out, F: combatBuffDurationNode}
}

// The fixture has two independently addressed monsters, no party characters,
// and an already active encounter group. A monster outside combat remains
// outside combat when 0x50f720 sees group+0x29 set. Visual attachment imports
// are case stubs; their pixels and handles do not enter the rule outputs.
// Nine complete aura records preserve the +9 multiplier and all untouched
// slots; child rebuilds read the first six (0x50ef50). The other combat spell
// records, status/buff sources, modifiers and template attributes are inputs.
func combatBuffDurationNode(in []int64, r Rand) []int64 {
	if len(in) != 839 {
		panic("combat buff: expected 839 fields")
	}
	if combatBuffSlotsErr != nil {
		panic(combatBuffSlotsErr)
	}
	for _, v := range in {
		if v < 0 || v > 0xffffffff {
			panic("combat buff: expected u32 fields")
		}
	}
	if in[6] > 2 {
		panic("combat buff: expected at most two fixture monsters")
	}
	a := append([]int64(nil), in...)
	// 0x54d3b5: low-32-bit multiply/add. Sentinel 9999 bypasses both calls.
	duration := uint32(a[4])*uint32(a[1]) + uint32(a[3])
	if duration != 9999 {
		duration++
		roll := r.R(4)
		if roll == 0 && duration > 1 || roll == 1 && duration <= 2 {
			duration++
		}
		// 0x517f90: unsigned quotient after wrapping the percentage product.
		duration += duration * uint32(a[2]) / 100
	}
	slot := -1
	for i, id := range combatBuffSlots {
		if uint32(a[0]) == id {
			slot = i
			break
		}
	}
	if slot < 0 {
		panic("combat buff: spell lacks a legal combat aura slot")
	}
	write := func(e []int64) { e[0], e[1], e[2], e[4] = 1, a[0], a[1], int64(duration) }
	// Adapt complete aura records to the exact shared rebuild node, retaining
	// the low-byte power and the separate multiplier word at +9.
	aura := func(full []int64) []int64 {
		v := make([]int64, 24)
		for i := 0; i < 6; i++ {
			copy(v[4*i:4*i+4], full[5*i:5*i+4])
			v[4*i+2] = int64(uint8(v[4*i+2]))
		}
		return v
	}
	if a[5] != 0 {
		write(a[205+5*slot : 210+5*slot])
		copy(a[102:205], dispelSharedModifiers(a[9:57], aura(a[205:250]), a[57:102], 1, r))
		// 0x562a50: active combat UI receives the notification mask 0x800100.
		a[8] = int64(uint32(a[8]) | 0x800100)
		a[7] = 1
	}
	for j := 0; j < int(a[6]); j++ {
		m := a[250+292*j : 542+292*j]
		if m[0] == 0 {
			continue
		}
		// 0x54d55c -> 0x4acd80 -> 0x49f3ba: a new attachment consumes
		// R(20), even though its animation phase has no rule output.
		e := m[139+5*slot : 144+5*slot]
		if e[0] == 0 || e[1] != a[0] {
			r.R(20)
		}
		write(m[139+5*slot : 144+5*slot])
		old := make([]int64, 271)
		copy(old[:139], m[:139])
		copy(old[139:163], aura(m[139:184]))
		copy(old[163:], m[184:])
		copy(m[184:287], dispelMonsterModifiers(old, r))
		// 0x4e5d00: template + signed attribute modifier, clamped 1..125.
		for k, off := range []int{0, 1, 4, 5, 6} {
			m[287+k] = int64(min(125, max(1, int(a[834+k])+int(int8(m[184+12+off])))))
		}
		a[7] = 1
	}
	out := []int64{a[7], a[8]}
	out = append(out, a[205:250]...)
	out = append(out, a[102:205]...)
	for j := 0; j < 2; j++ {
		m := a[250+292*j : 542+292*j]
		out = append(out, m[139:292]...)
	}
	return out
}
