package calc

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
)

// monsterArhitNode is the monster attack-rating hit value (exe 0x542720).
// Inputs 0..27 are defhitNode's (the target is the attacker's record at
// M+0x2ba); the rest are this node's own fields, in the order listed in init.
func monsterArhitNode(in []int64, r Rand) []int64 {
	if len(in) != 50 {
		panic("monster arhit: expected 50 inputs")
	}
	mode, kind := in[1], in[0]
	if mode < 0 || mode > 8 { // 0x542e10 asserts outside the 9-entry mode table.
		panic("monster arhit: attack mode outside 0..8")
	}
	battle, stance, attackType, bonus := in[28], int32(in[29]), in[30], int32(in[31])
	fatigue, self := in[32], in[33:37]
	templateExtra, capFlag, attackerKind, difficulty, skip := uint32(in[37]), in[38], in[39], in[40], in[41]
	charStat, targetStat := in[42:46], in[46:50]

	// V = 5*M+0x1dc + A+4, then the attack-mode hit modifier
	// (table at 0x542e98 -> immediates of 0x542e2b..0x542e55; combat-2.md:34).
	hitMode := [9]int32{0, -10, 0, -30, -5, 5, -10, 10, 0}
	v := 5*stance + bonus + hitMode[mode]
	if battle != 0 { // Battle flags G+0x9a4 / G+0x9a5 add 10 each (0x542765..0x542782).
		if in[4] != 0 {
			v += 10
		}
		if in[5] != 0 {
			v += 10
		}
	}
	// 0x52a7d0: fatigue 0..4 -> 0,5,10,20,40, larger values pass through.
	s := int32(uint32(fatigue))
	if fatigue <= 4 {
		s = [5]int32{0, 5, 10, 20, 40}[fatigue]
	}
	if halvedAttackTypes()[byte(attackType)] { // 18 bytes at 0x61d284, read from the player's exe.
		s = int32(uint32(s) >> 1)
	}
	v -= s
	if templateExtra != 0 { // template +0x141 (0x5427b4..0x542804).
		t := int32(uint32(self[2]) - (templateExtra*5+10)*5 + uint32(bonus))
		if t < 0 {
			t = 0
		}
		// 50 - t^0.75 (constants 0x5ee7b8 = 50.0, 0x5ebf40 = 0.75), truncated, floored at 0.
		x := int32(50 - math.Pow(float64(t), 0.75))
		if x < 0 {
			x = 0
		}
		v -= x
	}
	if skip == 0 {
		h := int32(defhitNode(in[:28], r)[0])
		v -= 5 * h
		mine := int32(self[0] + self[1] + self[2] + self[3])
		var other int32
		switch kind {
		case 3: // target monster's four stat bytes (+0x248..+0x24b).
			other = int32(targetStat[0] + targetStat[1] + targetStat[2] + targetStat[3])
		case 1: // target character's four i32 fields (+0xfd,+0x139,+0x14d,+0x161).
			other = int32(uint32(charStat[0]) + uint32(charStat[1]) + uint32(charStat[2]) + uint32(charStat[3]))
		default: // 0x542ee0 asserts first; unreachable.
			panic("monster arhit: target kind not 1 or 3")
		}
		v += (mine - other) / 10
	}
	if capFlag != 0 && v > 10 { // M+0x87 caps the value at 10 (0x542926..0x54293e).
		v = 10
	}
	// 0x55ccb0: percentage by M+0x16 and difficulty (0x6850d5); 32-bit product.
	pct := int32(100)
	switch {
	case attackerKind == 1 && difficulty == 0, attackerKind == 2 && difficulty == 2:
		pct = 60
	case attackerKind == 1 && difficulty == 2, attackerKind == 2 && difficulty == 0:
		pct = 140
	}
	if pct != 100 {
		v = int32(uint32(v)*uint32(pct)) / 100
	}
	return []int64{int64(v)}
}

var halvedTypes map[byte]bool

func halvedAttackTypes() map[byte]bool {
	if halvedTypes != nil {
		return halvedTypes
	}
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	off, err := peOffset(b, 0x61d284)
	if err != nil || off+18 > len(b) {
		panic("monster arhit: cannot read table at 0x61d284")
	}
	halvedTypes = map[byte]bool{}
	for _, x := range b[off : off+18] {
		halvedTypes[x] = true
	}
	return halvedTypes
}

func init() {
	in := []string{
		"targetType", "mode", "characterIndex", "combat", "ambush", "surprise",
		"speedDefense", "shieldDefense", "characterDefenseSuppressed", "monsterDefense",
		"monsterHitDefense", "fatigue", "monsterPosture", "monsterDefenseSuppressed",
		"ambushDefenseMask", "completedActionSelector", "completedPosture", "pendingPosture",
		"sceneFlags", "HP", "status", "actionCompleted", "mainAction", "offAction",
		"monsterID", "templateIndex", "characterDefHit", "monsterAC",
		"battlePointer", "attackerStance", "attackType", "attackBonus", "attackerFatigue",
		"attackerStat0", "attackerStat1", "attackerStat2", "attackerStat3",
		"templateExtra", "attackerCap", "attackerKind", "difficulty", "skipTarget",
		"charStat0", "charStat1", "charStat2", "charStat3",
		"targetStat0", "targetStat1", "targetStat2", "targetStat3",
	}
	Nodes["00542720"] = Node{Addr: "00542720", In: in, Out: []string{"arhit"}, F: monsterArhitNode}
}
