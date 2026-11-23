package calc

import "fmt"

// Input layout (313): monsterListPointer, monsterCount, then 8 x (present, 8 effects),
// 9 battle flags A (stride 0x11 from +0x7c1), 6 battle flags B (from +0x85a), then
// 8 x (present, active, 8 effects, 9 A bytes, 6 B bytes, hp, field9f, field2da).
const combatPendingMonsters = 8

func init() {
	in := []string{"monsterListPointer", "monsterCount"}
	for i := 0; i < 8; i++ {
		in = append(in, fmt.Sprintf("char%dPresent", i))
		for j := 0; j < 8; j++ {
			in = append(in, fmt.Sprintf("char%dEffect%d", i, j))
		}
	}
	for k := 0; k < 9; k++ {
		in = append(in, fmt.Sprintf("battleA%d", k))
	}
	for k := 0; k < 6; k++ {
		in = append(in, fmt.Sprintf("battleB%d", k))
	}
	for i := 0; i < combatPendingMonsters; i++ {
		in = append(in, fmt.Sprintf("monster%dPresent", i), fmt.Sprintf("monster%dActive", i))
		for j := 0; j < 8; j++ {
			in = append(in, fmt.Sprintf("monster%dEffect%d", i, j))
		}
		for k := 0; k < 9; k++ {
			in = append(in, fmt.Sprintf("monster%dA%d", i, k))
		}
		for k := 0; k < 6; k++ {
			in = append(in, fmt.Sprintf("monster%dB%d", i, k))
		}
		in = append(in, fmt.Sprintf("monster%dHP", i), fmt.Sprintf("monster%dField9f", i), fmt.Sprintf("monster%dField2da", i))
	}
	Nodes["004ed550"] = Node{Addr: "004ed550", In: in, Out: []string{"pending"}, F: combatPendingEffectsNode}
}

// combatPendingEffectsNode (exe 0x4ed550): 1 if any combat effect is still pending, else 0.
// Pure OR of independent tests (no writes, no RNG), so scan order does not matter:
// present characters' 8 u32 effect words (+0xa6d, stride 12) non-zero; the 9 battle bytes
// (+0x7c1, stride 17) and 6 (+0x85a, stride 17); per present+active monster the 9 bytes
// (+0x3e) and 6 bytes (+0xd7) of the record at monster+0x10; per listed monster (no presence test) the 8 u32 words
// (+0xaf, stride 12); per present monster alive (u32 HP at +0x2b non-zero, unsigned) with
// field +0x9f zero and field +0x2da non-zero. Null list pointer = 0 monsters (0x5e2c70).
// Only the first 8 monster records exist in the case fixture (precedent: 4f0c80); count > 8 panics.
func combatPendingEffectsNode(in []int64, r Rand) []int64 {
	if len(in) != 2+8*9+9+6+combatPendingMonsters*28 {
		panic("combatPendingEffects: wrong input count")
	}
	n := in[1]
	if in[0] == 0 {
		n = 0
	}
	if n < 0 || n > combatPendingMonsters {
		panic("combatPendingEffects: only 8 monster fixture records")
	}
	nz := func(s []int64) bool {
		for _, v := range s {
			if v != 0 {
				return true
			}
		}
		return false
	}
	p := 2
	for i := 0; i < 8; i++ {
		if in[p] != 0 && nz(in[p+1:p+9]) {
			return []int64{1}
		}
		p += 9
	}
	if nz(in[p : p+15]) {
		return []int64{1}
	}
	p += 15
	for i := 0; i < int(n); i++ {
		m := in[p+i*28 : p+(i+1)*28]
		present, active := m[0] != 0, m[1] != 0
		if nz(m[2:10]) {
			return []int64{1}
		}
		if present && active && nz(m[10:25]) {
			return []int64{1}
		}
		if present && m[25] != 0 && m[26] == 0 && m[27] != 0 {
			return []int64{1}
		}
	}
	return []int64{0}
}
