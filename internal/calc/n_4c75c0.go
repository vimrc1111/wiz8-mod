package calc

// monsterMissileReleaseNode is monster_missile_release (exe 0x4c75c0): the release
// event of a monster ranged attack. It reads the monster record M (found by the handle
// at this+0x1e4) and its template, then either
//   - (battle flag 0x683f94 set, battle+0x7b0 == 2, battle+0x7b8 == M; 0x4c760f..0x4c7630)
//     takes the attack slot from M's template pointer (+0x10, dword at +0x11), or
//   - (otherwise, 0x4c7635..) sets M+0x2ba = 1, picks a party member with 0x518150
//     (args 1,1,-1,-1), stores it at M+0x2be and -1 at M+0x2c2 (returns if none), then
//     takes the first of the 3 template attack slots (record = template + 0xe7 + 0x22*i,
//     active byte +0, flag word +0x15 & 0x110) and rolls M+0x2e5 with 0x542db0.
//
// Both paths set M+0x2df = 1 and call the projectile launcher 0x544630 with 7 stack
// arguments. That launcher is an engine process outside the formula boundary, so this
// node outputs what it would be called with: arg1 (attack class, 0x4c7723..0x4c7734),
// arg5 byte, arg6, arg7, the 12-dword attack pack that arg4 points to (dead frame at
// 0x4c75c0 locals +0x1c) and the 13-dword target record of arg2 (0x53be50; locals +0x4c),
// plus the call count (0 only when no party member exists).
// On the flag path arg7 is the 0x542720 result (0x32 otherwise).
//
// Modelled fixture identities (not rule data): M sits at 0x20010400 (the original compares
// the battle pointer with M's address), M's handle is 0xfffffffe, and a target with the
// same handle as M is not modelled (list handles are unique). The pack and target record
// are read from the dead frame; the original never writes the pack when no member is
// found, and the emulator stack starts zeroed, so it is 0 then.
const (
	mmrMonsterAddr = 0x20010400
	mmrBattleAddr  = 0x20000000 + 58000 // nonzero battle pointer fed to 0x542720's battle flag.
	mmrHandle      = 0xfffffffe
)

var mmrNames = func() []string {
	n := []string{"combat", "battle7b0", "battle7b8", "classCount", "attackerTemplate", "playerSlot", "targetType",
		"characterIndex", "monsterID", "mode", "released", "ambushDefenseMask", "monsterAC",
		"ambush", "surprise", "completedActionSelector", "completedPosture", "pendingPosture", "sceneFlags", "difficulty",
		"attackerStance", "attackerFatigue", "attackerStat0", "attackerStat1", "attackerStat2", "attackerStat3",
		"templateExtra", "attackerCap", "attackerKind", "monsterDefense", "monsterHitDefense", "fatigue",
		"monsterPosture", "monsterDefenseSuppressed", "templateIndex", "targetStat0", "targetStat1", "targetStat2",
		"targetStat3", "speedDefense", "shieldDefense", "characterDefenseSuppressed", "characterDefHit",
		"charStat0", "charStat1", "charStat2", "charStat3", "actionCompleted", "mainAction", "offAction"}
	for _, p := range []string{"present", "hp", "status"} {
		for s := 0; s < 8; s++ {
			n = append(n, p+string(rune('0'+s)))
		}
	}
	n = append(n, "tLevel")
	for s := 0; s < 3; s++ {
		for _, f := range []string{"active", "mask", "type", "b3", "bonus", "d5", "d9", "dd", "d11", "d17", "b1b", "cls"} {
			n = append(n, "s"+string(rune('0'+s))+f)
		}
	}
	return n
}()

var mmrIndex = func() map[string]int {
	m := map[string]int{}
	for i, n := range mmrNames {
		m[n] = i
	}
	return m
}()

// mmrArhitOrder is the input order of the 0x542720 node (n_542720.go).
var mmrArhitOrder = []string{
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

func monsterMissileReleaseNode(in []int64, r Rand) []int64 {
	if len(in) != len(mmrNames) {
		panic("monster missile release: wrong input count")
	}
	v := func(n string) int64 { return in[mmrIndex[n]] }
	slot := func(s int, f string) int64 { return v("s" + string(rune('0'+s)) + f) }

	m2ba, m2be, m2c2 := uint32(v("targetType")), uint32(v("characterIndex")), uint32(v("monsterID"))
	m2df, m2e5 := uint32(byte(v("released"))), uint32(v("mode"))
	flagPath := byte(v("combat")) != 0 && uint32(v("battle7b0")) == 2 && uint32(v("battle7b8")) == mmrMonsterAddr

	var selected int // attack slot index; 3 = none active
	if flagPath {
		selected = int(uint32(v("playerSlot")))
		if selected >= 3 { // 0x4c76fd: srAssertFail.
			panic("monster missile release: attack slot outside 0..2 (0x4c76fd asserts)")
		}
	} else {
		m2ba = 1
		pick := make([]int64, 36)
		pick[0], pick[1], pick[2], pick[3] = 1, 1, -1, -1
		for s := 0; s < 8; s++ {
			sfx := string(rune('0' + s))
			pick[4+s], pick[12+s], pick[20+s] = v("present"+sfx), v("hp"+sfx), v("status"+sfx)
		}
		party := randomPartyMemberNode(pick, r)[0]
		m2be, m2c2 = uint32(party), 0xffffffff
		if party == -1 {
			return mmrOut(m2ba, m2be, m2c2, m2df, m2e5, 0, nil)
		}
		selected = 3
		for s := 0; s < 3; s++ {
			if byte(slot(s, "active")) != 0 && uint16(slot(s, "mask"))&0x110 != 0 {
				selected = s
				m2e5 = uint32(chooseAttackModeNode([]int64{int64(uint16(slot(s, "mask")))}, r)[0])
				break
			}
		}
	}
	m2df = 1

	call := &mmrCall{arg6: 3, arg5: 1, arg7: 0x32}
	if flagPath {
		call.arg5 = 0
	}
	if selected < 3 {
		cls := int32(int8(slot(selected, "cls")))
		if uint32(cls) >= uint32(v("classCount")) { // 0x4c772b..0x4c774c: out of range class -> 0.
			cls = 0
		}
		call.arg1 = uint32(cls)
		level := uint32(byte(v("tLevel")))
		call.arg6 = uint32(byte(slot(selected, "b3")))
		call.pack = [12]uint32{0, uint32(slot(selected, "d17")), uint32(slot(selected, "d5")), uint32(slot(selected, "d9")),
			uint32(slot(selected, "dd")), uint32(slot(selected, "d11")), level + min(level, 15), uint32(byte(slot(selected, "b1b")))}
		if flagPath {
			ci := int(int32(uint32(v("characterIndex"))))
			vec := make([]int64, 50)
			for i, n := range mmrArhitOrder {
				switch n {
				case "HP", "status":
					if ci >= 0 && ci < 8 {
						vec[i] = v(map[string]string{"HP": "hp", "status": "status"}[n] + string(rune('0'+ci)))
					}
				case "attackType":
					vec[i] = slot(selected, "type")
				case "attackBonus":
					vec[i] = slot(selected, "bonus")
				case "characterIndex": // unused for a monster target; the sub-nodes insist on 0..7.
					vec[i] = v(n)
					if vec[i] < 0 && v("targetType") == 3 {
						vec[i] = 0
					}
				case "skipTarget":
					vec[i] = 0
				case "battlePointer":
					vec[i] = mmrBattleAddr
				default:
					vec[i] = v(n)
				}
			}
			call.arg7 = uint32(monsterArhitNode(vec, r)[0]) // arg7 is the 0x542720 result (also logged via 0x4ed260).
		}
	}
	return mmrOut(m2ba, m2be, m2c2, m2df, m2e5, 1, call)
}

type mmrCall struct {
	arg1, arg5, arg6, arg7 uint32
	pack                   [12]uint32
}

func mmrOut(m2ba, m2be, m2c2, m2df, m2e5 uint32, calls int64, c *mmrCall) []int64 {
	if c == nil {
		c = &mmrCall{}
	}
	out := []int64{int64(m2ba), int64(m2be), int64(m2c2), int64(m2df), int64(m2e5), calls}
	if calls == 0 {
		out = append(out, 0, 0, 0, 0)
	} else {
		out = append(out, int64(c.arg1), int64(c.arg5), int64(c.arg6), int64(c.arg7))
	}
	for _, p := range c.pack {
		out = append(out, int64(p))
	}
	// 0x53be50 target record: kind 2, index -1, handle of M, rest zero.
	rec := [13]uint32{2, 0xffffffff, mmrHandle}
	for _, x := range rec {
		out = append(out, int64(x))
	}
	return out
}

func init() {
	Nodes["004c75c0"] = Node{Addr: "004c75c0", In: mmrNames, F: monsterMissileReleaseNode, Out: func() []string {
		o := []string{"m2ba", "m2be", "m2c2", "m2df", "m2e5", "calls", "a1", "a5", "a6", "a7"}
		for k := 0; k < 12; k++ {
			o = append(o, "pack"+string(rune('a'+k)))
		}
		for k := 0; k < 13; k++ {
			o = append(o, "rec"+string(rune('a'+k)))
		}
		return o
	}()}
}
