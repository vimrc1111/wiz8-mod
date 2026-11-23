package calc

import (
	"fmt"
	"math"
)

// combat_difficulty_rating (exe 0x4e6ce0). Input layout (the case file lists the same names):
// globals/lists first, then 8 monsters x6, 3 templates x3, 8 party slots x7, 8 prior flags,
// 4 NPC table entries x6, 4 NPC list members x4.
const (
	cdMonList, cdMonCount, cdBattlePtr, cdBattleFlag, cdLastTime, cdClock, cdPrev = 0, 1, 2, 3, 4, 5, 6
	cdNpcTab, cdNpcTabN, cdNpcList, cdNpcListN, cdCounter, cdVoice                = 7, 8, 9, 10, 11, 14
	cdMon, cdTpl, cdChar, cdFlag, cdNpc, cdMember, cdInputs                       = 18, 66, 75, 131, 139, 163, 179
	cdMaxMonsters, cdMaxNpc, cdMaxSpecies                                         = 8, 4, 3 // fixture caps (arena capacity)
)

func init() {
	names := []string{"monsterListPointer", "monsterCount", "battlePointer", "battleFlag", "lastTime", "clock", "previousState",
		"npcTablePointer", "npcTableCount", "npcListPointer", "npcListCount"}
	for k := 0; k < 3; k++ {
		names = append(names, fmt.Sprintf("counter[%d]", k))
	}
	names = append(names, "voiceEasy", "voiceMid", "voiceHard", "voiceArg4")
	for k := 0; k < 8; k++ {
		for _, n := range []string{"monsterPresent", "monsterKind", "monsterHp", "monsterMaxHp", "monsterState", "monsterSpecies"} {
			names = append(names, fmt.Sprintf("%s[%d]", n, k))
		}
	}
	for j := 0; j < 3; j++ {
		for _, n := range []string{"templateStrengthA", "templateStrengthB", "templateClass"} {
			names = append(names, fmt.Sprintf("%s[%d]", n, j))
		}
	}
	for k := 0; k < 8; k++ {
		for _, n := range []string{"partyPresent", "npcId", "charhp", "charstatus", "charlevel", "charflag49", "charflag4d"} {
			names = append(names, fmt.Sprintf("%s[%d]", n, k))
		}
	}
	for k := 0; k < 8; k++ {
		names = append(names, fmt.Sprintf("priorFlag[%d]", k))
	}
	for e := 0; e < 4; e++ {
		for _, n := range []string{"npcEntry", "npcBlocked", "npcClass", "npcKind", "npcPriorF0", "npcPriorF1"} {
			names = append(names, fmt.Sprintf("%s[%d]", n, e))
		}
	}
	for l := 0; l < 4; l++ {
		for _, n := range []string{"listSpecies", "listNpca28", "listNpca29", "listNpca2a"} {
			names = append(names, fmt.Sprintf("%s[%d]", n, l))
		}
	}
	outs := []string{"state", "counter[0]", "counter[1]", "counter[2]", "messageArg1", "messageArg2", "messageArg3", "messageCalls",
		"voiceArg1", "voiceArg2", "voiceArg3", "voiceArg4", "voiceCalls"}
	for e := 0; e < 4; e++ {
		outs = append(outs, fmt.Sprintf("npcF0[%d]", e), fmt.Sprintf("npcF1[%d]", e))
	}
	for k := 0; k < 8; k++ {
		outs = append(outs, fmt.Sprintf("flag[%d]", k))
	}
	Nodes["004e6ce0"] = Node{Addr: "004e6ce0", In: names, Out: outs, F: combatDifficultyRatingNode}
}

// combatDifficultyRatingNode: sum of monster strength x HP% -> threat level 1..50 (0x556000),
// party average level (NPC members already on the NPC list are skipped) -> state 0/1/2
// (0x4e6ee8..0x4e6eff), then the message call (0x48fc10) and, by the 0x4e6f41 gate, the
// voice call (0x52e5c0), both registered 排除 and replaced by address stubs in the case:
// their recorded args and call counts are outputs here. Then the usage counter, the previous
// state byte, NPC object flags +0xf0/+0xf1 and the 0x6837a1 table.
// Differences from the original: a counted monster with HP>0 and max HP 0 divides by zero there
// and panics here; a party without counted members reads an uninitialised stack slot at 0x4e6ed9
// (0 under the emulator, modelled as 0); list/table sizes beyond the arena caps panic.
func combatDifficultyRatingNode(in []int64, r Rand) []int64 {
	if len(in) != cdInputs {
		panic("combat difficulty: wrong input count")
	}
	u := func(i int) uint32 { return uint32(in[i]) }
	for _, v := range in {
		if v < 0 || v > 0xffffffff {
			panic("combat difficulty: field width overflow")
		}
	}
	if in[cdMonCount] > cdMaxMonsters || in[cdNpcTabN] > cdMaxNpc || in[cdNpcListN] > cdMaxNpc {
		panic("combat difficulty: list count beyond fixture cap")
	}
	species := func(v int64) int {
		if v >= cdMaxSpecies {
			panic("combat difficulty: species beyond fixture template cap")
		}
		return int(v)
	}
	// Template record fields (0x4e6780: +0x26b if above zero else +0x181; +0x25f class).
	strength := func(j int) uint32 {
		if b := u(cdTpl + 3*j + 1); b > 0 {
			return b
		}
		return u(cdTpl + 3*j)
	}
	// Monster strength x HP%, 0x4e6d01..0x4e6d5c.
	var sum uint32
	if in[cdMonList] != 0 {
		for i := 0; i < int(in[cdMonCount]); i++ {
			m := in[cdMon+6*i : cdMon+6*i+6]
			if m[0] == 0 || m[1] != 1 || m[2] == 0 || m[4] >= 0x12 {
				continue
			}
			if m[3] == 0 {
				panic("combat difficulty: original divides by zero (monster max HP 0)")
			}
			pct := uint32(m[2]) * 100 / uint32(m[3]) // u32 wrap of HP*100, 0x4e6d31..0x4e6d3f
			sum += strength(species(m[5])) * pct / 100
		}
	}
	// 0x556000: ftol((sum^0.29 - 3.5) * 0.5), low byte; negative -> 0, above 50 -> 50; 0 -> 1.
	t := int8(int32((math.Pow(float64(sum), 0.28999999165534973) - 3.5) * 0.5))
	level := uint32(1)
	switch {
	case t < 0:
	case t > 50:
		level = 50
	case t > 0:
		level = uint32(t)
	}
	// NPC object lookup 0x50b800: index at or past the count (signed) reads slot 0.
	npc := func(id int64) int {
		if in[cdNpcTab] == 0 {
			return -1
		}
		if int32(id) < 0 {
			panic("combat difficulty: negative NPC index beyond fixture")
		}
		i := 0
		if id < in[cdNpcTabN] {
			i = int(id)
		}
		e := in[cdNpc+6*i : cdNpc+6*i+6]
		if e[0] == 0 || e[1] != 0 {
			return -1
		}
		return i
	}
	// NPC list member 0x5101b0/0x510180: template class +0x25f of the member's species.
	onList := func(class uint32) bool {
		if in[cdNpcList] == 0 {
			return false
		}
		for l := 0; l < int(in[cdNpcListN]); l++ {
			m := in[cdMember+4*l : cdMember+4*l+4]
			if m[1] != 0 && m[2] != 0 && m[3] == 1 && u(cdTpl+3*species(m[0])+2) == class {
				return true
			}
		}
		return false
	}
	var levels, count uint32
	for k := 0; k < 8; k++ {
		c := in[cdChar+7*k : cdChar+7*k+7]
		if c[0] == 0 || c[2] == 0 || c[3] >= 0x12 {
			continue
		}
		if k < 2 {
			if n := npc(c[1]); n >= 0 {
				class := uint32(in[cdNpc+6*n+2])
				if class != 0 && class != 1 && onList(class) {
					continue
				}
			}
		}
		count++
		levels += uint32(c[4])
	}
	avg := uint32(0)
	if count > 0 {
		avg = uint32(int32(float64(levels)/math.Pow(float64(count)*6, 0.5) + 0.5)) // 0x4e6ea9..0x4e6ed2
	}
	d := int32(avg*100/level - 100)
	state := int64(1)
	if d >= 20 {
		state = 0
	} else if d <= -20 {
		state = 2
	}
	msg := [3]int64{0x617320, 0x617330, 0x61733c}[state]
	// Voice gate 0x4e6f41..0x4e6f6e.
	voice := in[cdBattlePtr] != 0 && in[cdBattleFlag] != 0
	if !voice {
		voice = u(cdLastTime) <= u(cdClock) || state != in[cdPrev]
	}
	out := make([]int64, 0, 13+8+8)
	counters := [3]int64{in[cdCounter], in[cdCounter+1], in[cdCounter+2]}
	counters[state] = int64(uint32(counters[state] + 1))
	out = append(out, state, counters[0], counters[1], counters[2], msg, 0, 1, 1)
	if voice {
		out = append(out, in[cdVoice+int(state)], 0xffffffff, 0, in[cdVoice+3], 1)
	} else {
		out = append(out, 0, 0, 0, 0, 0)
	}
	// NPC flags 0x4e6fcc..0x4e702a, entries mutated in order (two slots may share an object).
	f := make([][2]int64, 4)
	for e := range f {
		f[e] = [2]int64{in[cdNpc+6*e+4], in[cdNpc+6*e+5]}
	}
	for k := 0; k < 2; k++ {
		if in[cdChar+7*k] == 0 {
			continue
		}
		n := npc(in[cdChar+7*k+1])
		if n < 0 {
			continue
		}
		switch kind := in[cdNpc+6*n+3]; {
		case in[cdChar+7*k+3] == 0x12:
			f[n][1] = 0
		case kind == 0x18 || kind == 0x10 || kind == 0x11:
			f[n][0] = 1
		default:
			f[n][1] = 1
		}
	}
	for e := range f {
		out = append(out, f[e][0], f[e][1])
	}
	// 0x6837a1 table, 0x4e703e..0x707f.
	for k := 0; k < 8; k++ {
		c := in[cdChar+7*k : cdChar+7*k+7]
		v := in[cdFlag+k]
		if c[0] != 0 && c[5] == 0 {
			v = 0
			if c[6] != 0 {
				v = 1
			}
		}
		out = append(out, v)
	}
	return out
}
