package calc

import "fmt"

func init() {
	inputs := []string{"monsterListPointer", "monsterCount", "combatActive", "combatOpeningBlocked", "uiPointer", "combatDisplayDirty", "sceneDisplayDirty", "previousHostileCount", "previousCharmedCount"}
	// Eight independent records are the case's arena capacity, not a game limit.
	for i := 0; i < 8; i++ {
		for _, name := range []string{"monsterPresent", "monsterActive", "monsterDisposition", "monsterCharmDuration"} {
			inputs = append(inputs, fmt.Sprintf("%s[%d]", name, i))
		}
	}
	Nodes["00546e70"] = Node{Addr: "00546e70", In: inputs,
		Out: []string{"hostileMonsterCount", "charmedMonsterCount", "combatOpeningBlocked", "combatDisplayDirty", "sceneDisplayDirty"}, F: hostileMonsterCountNode}
}

// Reproduce the counts and the 0x5653f0 tail call's display invalidation.
// List lookup/count children (0x4e5620, 0x5e2870, 0x5e2c70) only select the
// supplied records. No executable data table is read by this composition.
func hostileMonsterCountNode(in []int64, r Rand) []int64 {
	if len(in) != 41 || in[1] < 0 || in[1] > 8 {
		panic("hostile monster count: expected count 0..8 and eight records")
	}
	for i, v := range in {
		limit := int64(0xffffffff)
		if i >= 2 && i <= 6 && i != 4 || i >= 9 && (i-9)%4 != 3 {
			limit = 255
		}
		if v < 0 || v > limit {
			panic("hostile monster count: field width overflow")
		}
	}
	// 0x546e77 / 0x546e81 reset both u32 counters, ignoring previous values.
	hostile, charmed := int64(0), int64(0)
	if in[0] != 0 { // 0x5e2c74: a null list has zero members.
		for i := int64(0); i < in[1]; i++ {
			m := in[9+4*i : 13+4*i]
			// 0x546e9f / 0x546ea9 require both u8 presence and active flags.
			if m[0] == 0 || m[1] == 0 {
				continue
			}
			if m[2] == 1 { // Enemy disposition code at 0x546eb0.
				hostile++
			}
			if m[3] != 0 { // u32 charm duration M+0x8b at 0x546ebc.
				charmed++
			}
		}
	}
	openingBlocked, combatDirty, sceneDirty := in[3], in[5], in[6]
	if in[2] != 0 && hostile != 0 {
		openingBlocked = 1 // 0x546ef9; otherwise preserve G+0xa54.
	}
	if in[4] != 0 { // 0x5653f5: absent UI receives no writes.
		if in[2] != 0 {
			combatDirty = 1 // 0x565405 writes UI+0x2c0 only during combat.
		}
		sceneDirty = 1 // 0x565410 always writes UI+0x2c8 for an existing UI.
	}
	return []int64{hostile, charmed, openingBlocked, combatDirty, sceneDirty}
}
