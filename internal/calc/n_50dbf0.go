package calc

import "fmt"

// npcRegionStatPenaltyNode composes the party-slot lookup and the region penalty.
// Inputs include both pointer identities, C+4 (in party), all eight Q+fe flags,
// and the caller's buffer +0..+3c. Pointer relocation preserves the result.
// Sources: 0x517ec0 (party base 0x685174, eight records of 0x1862 bytes),
// 0x50dbf0 (NPC state base 0x685178, records of 0x106 bytes, flag +0xfe).
// These are record layout constants; this node reads no executable data table.
func npcRegionStatPenaltyNode(in []int64, r Rand) []int64 {
	const (
		partySlots      = 8      // 0x517eff
		characterStride = 0x1862 // 0x517ef9
		bufferSize      = 0x3d   // 0x50dc21..0x50dc47, plus adjacent sentinel +0x3c
		bufferInput     = 3 + partySlots
		attributeStart  = 0xc // 0x50dc21
		attributeCount  = 7   // 0x50dc2d
		skillCount      = 41  // 0x50dc44
		penalty         = 20  // 0x50dc26 and 0x50dc3d
	)
	if len(in) != bufferInput+bufferSize {
		panic("npc region penalty: expected character address, party base, in-party flag, eight penalty flags and 61 buffer bytes")
	}
	for _, address := range in[:2] {
		if address < 0 || address > 0xffffffff {
			panic("npc region penalty: pointer must be u32")
		}
	}
	for _, value := range in[2:] {
		if value < 0 || value > 255 {
			panic("npc region penalty: fields must be u8")
		}
	}
	if in[2] == 0 {
		panic("npc region penalty: character must be in party (0x517ed3 assertion)")
	}
	// The original lookup compares pointers after each u32 stride increment.
	slot := -1
	for i := 0; i < partySlots; i++ {
		if uint32(in[0]) == uint32(in[1])+uint32(i*characterStride) {
			slot = i
			break
		}
	}
	if slot < 0 {
		panic("npc region penalty: character must match a party slot (0x517f04 assertion)")
	}
	out := append([]int64(nil), in[bufferInput:]...)
	if in[3+slot] != 0 {
		for i := attributeStart; i < attributeStart+attributeCount+skillCount; i++ {
			out[i] = int64(uint8(out[i] - penalty))
		}
	}
	return out
}

func init() {
	inputs := []string{"characterAddress", "partyCharacterBase", "inParty"}
	for i := 0; i < 8; i++ {
		inputs = append(inputs, fmt.Sprintf("npcRegionPenaltyEnabled[%d]", i))
	}
	var outputs []string
	attributes := []string{"strength", "intelligence", "piety", "vitality", "dexterity", "speed", "senses"}
	for offset := 0; offset <= 0x3c; offset++ {
		name := fmt.Sprintf("callerBuffer[%d]", offset)
		if offset >= 0xc && offset <= 0x12 {
			name = attributes[offset-0xc]
		} else if offset >= 0x13 && offset <= 0x3b {
			name = fmt.Sprintf("skill[%d]", offset-0x13)
		}
		inputs = append(inputs, name)
		outputs = append(outputs, name)
	}
	Nodes["0050dbf0"] = Node{Addr: "0050dbf0", In: inputs, Out: outputs, F: npcRegionStatPenaltyNode}
}
