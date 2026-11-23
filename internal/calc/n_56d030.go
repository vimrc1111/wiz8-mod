package calc

// dialogue_party_representative 0x56d030: opening a dialogue with an NPC object
// (the single argument, arena address 0x20001000 in the case). Records the
// linked monster id and the object in the dialogue panel (0x649f1c), clears the
// object's flags, chooses the representative party slot, refreshes each present
// member's portrait expression and drops a stale "attitude cached" flag.
//
// Callees: 0x50a3c0 resolves the monster record of a linked and available
// object (its first word is the id, 0x50a3c0..0x50a3f6); 0x525110 (second
// argument 0) resets the selection globals and, for an unlinked object, claims
// the NPC entry whose template id equals the object id (0x509cd0 with
// availability 0 and id -1, 0x509e1e..0x509e2c); 0x50a500 is the NPC attitude
// (npcAttitudeNode, reused); 0x52f000 is inlined as in characterPrimaryCondition
// (portrait expression 1 only for states below 0xf with the gate clear).
// 0x4f34d0 and 0x4f4650 only lay out the dialogue text widget and are not
// modelled: their effects are excluded from the outputs.
const (
	dialogueRepObject   = 0x20001000 // arena address of the argument object in the case
	dialogueRepMembers  = 8          // character stride 0x1862, presence stride 0x106, portrait stride 0x118
	dialogueRepNone     = 0xffffffff // "no monster" / unset marker
	dialogueRepPortrait = 0xf        // expression threshold on the primary state (0x52f031)
)

func init() {
	Nodes["0056d030"] = Node{Addr: "0056d030", In: []string{
		"baseAttitude", "monsterLinked", "monsterAvailable", "monsterAttitudeModifier",
		"monsterAttitudeOverride", "faction", "fixedFactionAttitude", "alliedFaction",
		"factionAttitude[0]", "factionAttitude[1]", "factionAttitude[2]", "factionAttitude[3]",
		"factionAttitude[4]", "factionAttitude[5]", "factionAttitude[6]", "factionAttitude[7]",
		"factionAttitude[8]", "factionAttitude[9]", "factionAttitude[10]", "factionAttitude[11]",
		"factionAttitude[12]", "factionAttitude[13]", "factionAttitude[14]", "factionAttitude[15]",
		"factionAttitude[16]", "factionAttitude[17]", "factionAttitude[18]", "factionAttitude[19]",
		"factionAttitude[20]", "npcListCount", "slot0.currentHP", "slot0.npcIndex", "slot1.currentHP",
		"slot1.npcIndex", "npcEntry0", "npc0.unavailable", "npc0.faction", "npcEntry1",
		"npc1.unavailable", "npc1.faction", "monsterID", "panelMode", "dialogOpen", "textArg",
		"portraitGate", "npc.1c", "npc.1d", "npc.22", "npc.23", "npc.84", "npc.ef", "npcId", "decoyId1",
		"panel.f8", "panel.f4", "panel.1d4", "panel.23d", "charStat0", "charStat1", "charStat2",
		"charStat3", "charStat4", "charStat5", "charStat6", "charStat7", "state0", "state1", "state2",
		"state3", "state4", "state5", "state6", "state7", "present0", "present1", "present2", "present3",
		"present4", "present5", "present6", "present7", "expr0", "expr1", "expr2", "expr3", "expr4",
		"expr5", "expr6", "expr7", "flag99_0", "flag99_1", "flag99_2", "flag99_3", "flag99_4",
		"flag99_5", "flag99_6", "flag99_7", "npc.0", "e0.16", "e0.25", "e0.1a", "e1.16", "e1.25",
		"e1.1a", "g68c4f5", "g68c49e", "g68c4a0", "g68c498", "g68c4ac", "g68c4a8",
	}, Out: []string{
		"panel.f8", "panel.f4", "panel.1d4", "panel.23d", "npc.1c", "npc.22", "npc.23", "npc.84",
		"expr0", "expr1", "expr2", "expr3", "expr4", "expr5", "expr6", "expr7", "flag99_0", "flag99_1",
		"flag99_2", "flag99_3", "flag99_4", "flag99_5", "flag99_6", "flag99_7", "g68c4f5", "g68c49e",
		"g68c4a0", "g68c498", "g68c4ac", "g68c4a8", "e0.16", "e0.25", "e0.1a", "e1.16", "e1.25", "e1.1a",
	}, F: dialoguePartyRepresentativeNode}
}

func dialoguePartyRepresentativeNode(in []int64, r Rand) []int64 {
	if len(in) != 110 {
		panic("dialogue party representative: expected 110 inputs")
	}
	const (
		iLinked, iAvail, iCount, iMonster = 1, 2, 29, 40
		iNpc1c, iNpc1d, iNpcEf            = 45, 46, 50
		iCharStat, iState, iPresent       = 57, 65, 73
		iExpr, iFlag, iObj0, iEntry       = 81, 89, 97, 98
		iGate                             = 44
	)
	out := make([]int64, 36)
	u := func(i int) int64 { return int64(uint32(in[i])) }
	b := func(i int) int64 { return int64(uint8(in[i])) }
	// panel +0xf8: monster id of the linked, available object, else -1.
	out[0] = dialogueRepNone
	if in[iLinked] != 0 && in[iAvail] != 0 {
		out[0] = u(iMonster)
	}
	out[2] = dialogueRepObject // panel +0x1d4
	out[5], out[6] = 0, 0      // object +0x22, +0x23
	out[7] = b(49)             // object +0x84, cleared only when +0x1d is set
	if in[iNpc1d] != 0 {
		out[7] = 0
	}
	out[3] = 0 // panel +0x23d
	// 0x525110 globals; the NPC entry claim happens only for an unlinked object.
	out[24], out[25], out[26], out[27] = 0, 0, 0, dialogueRepNone
	out[28], out[29] = dialogueRepObject, u(iObj0)
	for e := 0; e < 2; e++ {
		out[30+3*e], out[31+3*e], out[32+3*e] = u(iEntry+3*e), b(iEntry+3*e+1), b(iEntry+3*e+2)
	}
	if in[iLinked] == 0 {
		if in[34] == 0 {
			panic("dialogue party representative: NPC entry list is empty")
		}
		out[30], out[31], out[32] = dialogueRepNone, 0, 1 // first entry, matched by template id
	}
	// representative: strict unsigned compare against an initial 0xffffffff never
	// succeeds, so the first present member stays selected (0x56d0d6..0x56d148).
	rep, best := int64(-1), uint32(0xffffffff)
	for k := 0; k < dialogueRepMembers; k++ {
		if uint8(in[iPresent+k]) != 0 {
			if rep == -1 {
				rep = int64(k)
			}
			if best < uint32(in[iCharStat+k]) {
				best, rep = uint32(in[iCharStat+k]), int64(k)
			}
		}
	}
	out[1] = int64(uint32(rep))
	// 0x52f000 for every present member.
	for k := 0; k < dialogueRepMembers; k++ {
		out[8+k], out[16+k] = u(iExpr+k), b(iFlag+k)
		if uint8(in[iPresent+k]) != 0 {
			out[16+k] = 0
			out[8+k] = 2
			if uint32(in[iState+k]) < dialogueRepPortrait && uint8(in[iGate]) == 0 {
				out[8+k] = 1
			}
		}
	}
	// 0x50a500: a cached attitude that no longer matches clears the flag.
	out[4] = b(iNpc1c)
	if in[iNpc1c] != 0 {
		a := make([]int64, 44)
		copy(a, in[:29])
		a[29], a[30] = 1, in[iCount]
		a[31], a[32], a[33] = in[iPresent], in[30], in[31]
		a[34], a[35], a[36] = in[iPresent+1], in[32], in[33]
		copy(a[37:43], in[34:40])
		a[43] = in[iMonster]
		// 0x50a500 classifies the signed attitude byte: below 0x21 is 2, below 0x42 is 1, else 0.
		class := int64(0)
		switch att := int8(npcAttitudeNode(a, r)[0]); {
		case att < 0x21:
			class = 2
		case att < 0x42:
			class = 1
		}
		if b(iNpcEf) != class {
			out[4] = 0
		}
	}
	return out
}
