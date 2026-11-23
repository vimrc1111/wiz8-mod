package calc

import "fmt"

// player_arpen 0x541ec0: armor-penetration rating of one attack
// (notes/re/combat-2.md:34; build/wiz8.asm:437222).
// Inputs: who, hand, the 53 inputs of defpenNode (0x542fc0; they carry the
// attack mode, hit location and the current target), then this node's own fields.
const (
	arpenDefpen  = 2  // first defpenNode input
	arpenDefpenN = 53 // defpenNode input count
	arpenOwn     = arpenDefpen + arpenDefpenN
	arpenHandLen = 4 // base, skill, bonus, kind per hand slot
	arpenBonusA  = arpenOwn + 2*arpenHandLen
	arpenGrade   = arpenBonusA + 2
	arpenAttr    = arpenGrade + 1 // attacker INT, DEX, SPD, SEN
	arpenTChar   = arpenAttr + 4  // target character INT, DEX, SPD, SEN
	arpenMonster = arpenTChar + 4 // target monster u8 attributes
	arpenReverse = arpenMonster + 4
	arpenDiff    = arpenReverse + 1
	arpenInputs  = arpenDiff + 1
)

// attackModePenModifier is 0x542e10 with its first argument 1 (table at 0x542ebc).
var attackModePenModifier = [9]int32{0, 10, -5, -30, 5, -5, 0, -10, 10}

// lowStaminaPenaltyGrade is 0x52a7d0: grades 0..4 map to 0,5,10,20,40; anything else is returned unchanged.
func lowStaminaPenaltyGrade(g uint32) int32 {
	if g > 4 {
		return int32(g)
	}
	return [5]int32{0, 5, 10, 20, 40}[g]
}

func playerArpenNode(in []int64, r Rand) []int64 {
	if len(in) != arpenInputs {
		panic("player arpen: wrong input count")
	}
	who, hand := in[0], in[1]
	def := in[arpenDefpen:arpenOwn]
	kind, mode := def[0], def[1]
	if who < 0 || who > 7 || hand < 0 || hand > 1 {
		panic("player arpen: character 0..7 and hand slot 0..1 expected")
	}
	if mode < 0 || mode > 8 { // 0x542e10 stops on a mode outside its 9-entry table
		panic("player arpen: attack mode outside 0..8")
	}
	h := arpenOwn + int(hand)*arpenHandLen
	i := func(k int) int32 { return int32(in[k]) }
	// Hand slot h sits at +91*h (slot index plus 90*index); terms weigh 1, 5, 5, the two attacker bytes 5 each (exe 0x541eeb..0x541f3d), plus the mode modifier.
	v := i(h) + 5*i(h+1) + 5*i(h+2) + 5*i(arpenBonusA) + 5*i(arpenBonusA+1) + attackModePenModifier[mode]
	if uint32(in[h+3]) != 7 { // hand kind 7 skips the stamina penalty (0x541f50..0x541f56)
		v -= lowStaminaPenaltyGrade(uint32(in[arpenGrade]))
	}
	v -= 5 * int32(defpenNode(def, r)[0]) // 0x541f82..0x541f96
	own := i(arpenAttr) + i(arpenAttr+1) + i(arpenAttr+2) + i(arpenAttr+3)
	var target int32
	switch kind {
	case 3: // monster instance +0x248..0x24b, four u8
		for k := 0; k < 4; k++ {
			target += int32(uint8(in[arpenMonster+k]))
		}
	case 1: // target character, four i32
		for k := 0; k < 4; k++ {
			target += i(arpenTChar + k)
		}
	default: // 0x54205b is only reachable after 0x542fc0 stopped on it
		panic("player arpen: target kind outside 1 and 3")
	}
	v += (own - target) / 10 // truncating division (0x54207d..0x542090)
	z := make([]int64, 8)
	return difficultyNode(append(z, int64(v), in[arpenDiff], in[arpenReverse], who), r)
}

func init() {
	in := []string{"who", "hand",
		"targetType", "mode", "characterIndex", "combat", "ambush", "surprise",
		"speedDefense", "shieldDefense", "characterDefenseSuppressed", "monsterDefense",
		"monsterHitDefense", "fatigue", "monsterPosture", "monsterDefenseSuppressed",
		"ambushDefenseMask", "completedActionSelector", "completedPosture", "pendingPosture",
		"sceneFlags", "HP", "status", "actionCompleted", "mainAction", "offAction",
		"monsterID", "templateIndex", "characterDefHit", "monsterAC",
		"hitLocation", "monsterPenDefense",
	}
	for k := 0; k < 9; k++ {
		in = append(in, fmt.Sprint("monsterModeAC", k))
	}
	for k := 0; k < 7; k++ {
		in = append(in, fmt.Sprint("monsterLocAC", k))
	}
	for k := 0; k < 7; k++ {
		in = append(in, fmt.Sprint("characterLocDef", k))
	}
	for h := 0; h < 2; h++ {
		for _, n := range []string{"Base", "Skill", "Bonus", "Kind"} {
			in = append(in, fmt.Sprint("hand", h, n))
		}
	}
	in = append(in, "bonusA", "bonusB", "staminaGrade",
		"attackerINT", "attackerDEX", "attackerSPD", "attackerSEN",
		"targetINT", "targetDEX", "targetSPD", "targetSEN",
		"monsterAttrA", "monsterAttrB", "monsterAttrC", "monsterAttrD",
		"reverseDifficultyState", "difficulty")
	Nodes["00541ec0"] = Node{Addr: "00541ec0", In: in, Out: []string{"arpen"}, F: playerArpenNode}
}
