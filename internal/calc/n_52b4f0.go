package calc

func init() {
	sp := make([]string, 0, 48)
	for character := 0; character < 8; character++ {
		for realm := 0; realm < 6; realm++ {
			sp = append(sp, "character"+string(rune('0'+character))+"CurrentSP"+string(rune('0'+realm)))
		}
	}
	in := append([]string{"characterIndex", "realmIndex", "healAmount"}, sp...)
	in = append(in, "maximumSP0", "maximumSP1", "maximumSP2", "maximumSP3", "maximumSP4", "maximumSP5",
		"uiMode", "characterRefreshFlags", "combatRefreshFlags", "combatUIPointer")
	out := append(append([]string{}, sp...), "characterRefreshFlags", "combatRefreshFlags")
	Nodes["0052b4f0"] = Node{Addr: "0052b4f0", In: in, Out: out, F: characterRealmSpHealNode}
}

// characterRealmSpHealNode replays SP healing and the composed UI invalidation.
// Inputs carry every party slot's six realm SP values; only the slot at
// characterIndex/realmIndex changes, and returning all 48 checks the rest are kept.
// SP inputs/outputs use signed representations of the complete DWORD bit patterns.
func characterRealmSpHealNode(in []int64, _ Rand) []int64 {
	if len(in) != 61 {
		panic("character_realm_sp_heal: expected 61 inputs")
	}
	// Eight party slots: the caller's party loop at 0x52b57d/0x52b583.
	// Six SP realms: notes/re/combat-2.md C17; fields at 0x52b517/0x52b529.
	const characters, realms = 8, 6
	if in[0] < 0 || in[0] >= characters || in[1] < 0 || in[1] >= realms {
		panic("character_realm_sp_heal: invalid character or realm index")
	}
	for _, value := range in[2:57] {
		if value < -2147483648 || value > 2147483647 {
			panic("character_realm_sp_heal: expected signed DWORD representation")
		}
	}
	for _, value := range in[58:61] {
		if value < 0 || value > 4294967295 {
			panic("character_realm_sp_heal: expected unsigned DWORD")
		}
	}
	// combatUIPointer is a synthetic arena address in 0052b4f0.json.
	// Its identity is irrelevant to calc; 0x562a5e tests only pointer presence.
	out := append(make([]int64, 0, characters*realms+2), in[3:3+characters*realms]...)
	slot := int(in[0])*realms + int(in[1])
	// 0x52b525 wraps the sum to a DWORD. 0x52b537 compares it as signed.
	current := int32(uint32(in[3+slot]) + uint32(in[2]))
	maximum := int32(in[51+int(in[1])])
	if current > maximum {
		current = maximum
	}
	out[slot] = int64(current)
	characterFlags, combatFlags := uint32(in[58]), uint32(in[59])
	// UI modes 6/7: 0x55ee35/0x55ee3a. Character refresh bit 0x100:
	// 0x55ee57; combat refresh character mask: 0x55ee46 and 0x562a6c.
	if in[57] == 6 {
		characterFlags |= 0x100
	} else if in[57] == 7 && in[60] != 0 {
		combatFlags |= uint32(1) << uint(in[0])
	}
	return append(out, int64(characterFlags), int64(combatFlags))
}
