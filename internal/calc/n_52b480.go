package calc

func init() {
	in := []string{"characterIndex", "realmIndex", "spendAmount",
		"currentSP0", "currentSP1", "currentSP2", "currentSP3", "currentSP4", "currentSP5",
		"uiMode", "characterRefreshFlags", "combatRefreshFlags", "combatUIPointer"}
	out := make([]string, 0, 50)
	for character := 0; character < 8; character++ {
		for realm := 0; realm < 6; realm++ {
			out = append(out, "character"+string(rune('0'+character))+"CurrentSP"+string(rune('0'+realm)))
		}
	}
	out = append(out, "characterRefreshFlags", "combatRefreshFlags")
	Nodes["0052b480"] = Node{Addr: "0052b480", In: in, Out: out, F: characterSpSpendNode}
}

// characterSpSpendNode composes realm SP spending with UI invalidation.
// Cases duplicate six independent SP values across eight party records and
// return every slot to check selection and preservation. Signed inputs cover
// DWORD bit patterns; insufficient nonzero spending violates the original
// assertion precondition at 0x52b4ac/0x52b4c6 (cov-exempt.d/52b480.tsv).
func characterSpSpendNode(in []int64, _ Rand) []int64 {
	if len(in) != 13 {
		panic("character_sp_spend: expected 13 inputs")
	}
	// Eight party slots: 0x52b57d/0x52b583; six realms: notes/re/combat-2.md C17.
	const characters, realms = 8, 6
	if in[0] < 0 || in[0] >= characters || in[1] < 0 || in[1] >= realms {
		panic("character_sp_spend: invalid character or realm index")
	}
	for _, value := range in[2:10] {
		if value < -2147483648 || value > 2147483647 {
			panic("character_sp_spend: expected signed DWORD representation")
		}
	}
	for _, value := range in[10:13] {
		if value < 0 || value > 4294967295 {
			panic("character_sp_spend: expected unsigned DWORD")
		}
	}
	out := make([]int64, 0, characters*realms+2)
	for character := 0; character < characters; character++ {
		out = append(out, in[3:9]...)
	}
	characterFlags, combatFlags := uint32(in[10]), uint32(in[11])
	// Zero spending at 0x52b49f/0x52b4a5 skips both SP writes and refresh.
	if in[2] != 0 {
		realm := int(in[1])
		if in[3+realm] < in[2] {
			panic("character_sp_spend: insufficient SP violates original assertion")
		}
		// 0x52b4d6 performs DWORD subtraction, including wraparound.
		out[int(in[0])*realms+realm] = int64(int32(uint32(in[3+realm]) - uint32(in[2])))
		// 0x55ee35/0x55ee3a select modes 6/7. 0x55ee57 sets bit
		// 0x100; 0x562a5e checks pointer presence, 0x562a6c sets the mask.
		if in[9] == 6 {
			characterFlags |= 0x100
		} else if in[9] == 7 && in[12] != 0 {
			combatFlags |= uint32(1) << uint(in[0])
		}
	}
	return append(out, int64(characterFlags), int64(combatFlags))
}
