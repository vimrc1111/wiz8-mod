package calc

// npcAttitudeNode models byte attitudes and the mutable faction reputation
// state at 0x68d6e8 (21 records, stride 14; 0x535c50 / 0x535d30).
// Each record's attitude is an explicit input: it is save-game state, not a
// constant exe table. No original bytes or extracted table values are stored.
// Arena references in this adapter match 0050a280.json; absent list entries
// are zero. NPC indices at/above the count select entry zero (0x50a360).
func npcAttitudeNode(in []int64, r Rand) []int64 {
	if len(in) != 44 {
		panic("NPC attitude: expected 44 inputs")
	}
	result := func(v byte) []int64 { return []int64{int64(v)} }
	base := byte(in[0])
	local, saved := base, base
	if in[1] != 0 && in[2] != 0 {
		local += byte(in[3]) // byte addition at 0x50a2cb wraps modulo 256.
		saved = local        // 0x50a2cf preserves the combined individual attitude.
		if uint32(in[4]) != 0 {
			local, saved = 100, 100
		} // 0x50a2d5.
	}
	faction := int(in[5])
	if faction < 0 || faction > 20 {
		panic("NPC attitude: faction outside 0..20")
	}
	if faction == 0 {
		return result(local)
	}
	attitude := int8(in[8+faction])
	if in[6] != 0 || attitude < 34 {
		return result(byte(attitude))
	} // 0x535cab..0x535cbb.
	// The final fallback uses the combined individual attitude saved before
	// faction lookup (0x50a2cf / 0x50a3a1).
	ally := int32(int8(in[7]))
	if ally != 0 {
		for slot := 0; slot < 2; slot++ { // two NPC slots: 0x50a39c.
			start := 31 + slot*3
			if in[start] == 0 || uint32(in[start+1]) == 0 || in[29] == 0 {
				continue
			}
			index := int32(in[start+2])
			if index >= int32(in[30]) {
				index = 0
			}
			if index < 0 || index > 1 {
				panic("NPC attitude: invalid list index")
			}
			record := 37 + int(index)*3
			expected := int64(0x200055f0) // partyNPC0 arena address; case objects.
			if index == 1 {
				expected = 0x200059d8
			} // partyNPC1 arena address.
			if in[record] == 0 {
				continue
			}
			if in[record] != expected {
				panic("NPC attitude: invalid NPC reference")
			}
			if in[record+1] == 0 && int32(uint8(in[record+2])) == ally {
				return result(50)
			} // 0x50a3ad.
		}
	}
	return result(saved)
}

func init() {
	Nodes["0050a280"] = Node{Addr: "0050a280", In: []string{
		"baseAttitude",
		"monsterLinked",
		"monsterAvailable",
		"monsterAttitudeModifier",
		"monsterAttitudeOverride",
		"faction",
		"fixedFactionAttitude",
		"alliedFaction",
		"factionAttitude[0]",
		"factionAttitude[1]",
		"factionAttitude[2]",
		"factionAttitude[3]",
		"factionAttitude[4]",
		"factionAttitude[5]",
		"factionAttitude[6]",
		"factionAttitude[7]",
		"factionAttitude[8]",
		"factionAttitude[9]",
		"factionAttitude[10]",
		"factionAttitude[11]",
		"factionAttitude[12]",
		"factionAttitude[13]",
		"factionAttitude[14]",
		"factionAttitude[15]",
		"factionAttitude[16]",
		"factionAttitude[17]",
		"factionAttitude[18]",
		"factionAttitude[19]",
		"factionAttitude[20]",
		"npcListPointer",
		"npcListCount",
		"slot0.active",
		"slot0.currentHP",
		"slot0.npcIndex",
		"slot1.active",
		"slot1.currentHP",
		"slot1.npcIndex",
		"npcEntry0",
		"npc0.unavailable",
		"npc0.faction",
		"npcEntry1",
		"npc1.unavailable",
		"npc1.faction",
		"monsterID",
	}, Out: []string{"attitude"}, F: npcAttitudeNode}
}
