package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Node 0x53d870 (attack_swings): one swing of one hand. Input and output cell
// names are the case file's (re/emu/cases/0053d870.json).
const (
	swWho        = 0
	swMode       = 1
	swFlag510c   = 2
	swHand       = 3
	swSwings0    = 4
	swSwings1    = 5
	swLsFlag     = 6
	swKind0      = 7
	swKind1      = 8
	swEnabled0   = 9
	swEnabled1   = 10
	swHmode0     = 11
	swHmode1     = 12
	swDice0      = 13
	swDice1      = 14
	swUmask0     = 15
	swUmask1     = 16
	swItem6      = 17
	swItem7      = 18
	swA15        = 19
	swCls        = 20
	swRace       = 21
	swLevel      = 22
	swInParty    = 23
	swMainAction = 24
	swTtype      = 25
	swTchar      = 26
	swTmon       = 27
	swTgrp       = 28
	swIt0cls     = 29
	swIt0sub     = 30
	swIt0range   = 31
	swIt0bonus   = 32
	swIt0wmask   = 33
	swIt1cls     = 34
	swIt1sub     = 35
	swIt1range   = 36
	swIt1bonus   = 37
	swIt1wmask   = 38
	swIt2cls     = 39
	swIt2sub     = 40
	swIt2range   = 41
	swIt2bonus   = 42
	swIt2wmask   = 43
	swIt3cls     = 44
	swIt3sub     = 45
	swIt3range   = 46
	swIt3bonus   = 47
	swIt3wmask   = 48
	swTpresent   = 49
	swThp        = 50
	swTstatus    = 51
	swTmainA     = 52
	swToffA      = 53
	swTcur       = 54
	swTinterf    = 55
	swTatt       = 56
	swMid        = 57
	swMhp        = 58
	swM33        = 59
	swM107       = 60
	swM87        = 61
	swM2e1       = 62
	swM24b       = 63
	swMexcl      = 64
	swMactive    = 65
	swMen        = 66
	swMphys      = 67
	swMspell     = 68
	swMcnt       = 69
	swCombat     = 70
	swPx         = 71
	swPy         = 72
	swPz         = 73
	swMx         = 74
	swMy         = 75
	swMz         = 76
	swMrad       = 77
	swPrad       = 78
	swMhead      = 79
	swF0_0       = 80
	swF0_1       = 81
	swF0_2       = 82
	swF0_3       = 83
	swF0_4       = 84
	swF0_5       = 85
	swF0_6       = 86
	swF0_7       = 87
	swF3_0       = 88
	swF3_1       = 89
	swF3_2       = 90
	swF3_3       = 91
	swF3_4       = 92
	swF3_5       = 93
	swF3_6       = 94
	swF3_7       = 95
)

const (
	soRet                 = 0
	soT68                 = 1
	soT70                 = 2
	soT74                 = 3
	soT78                 = 4
	soT7c                 = 5
	soTa5                 = 6
	soTswings0            = 7
	soTswings1            = 8
	soTls                 = 9
	soFlag9a4             = 10
	soFlag9a5             = 11
	soMode0               = 12
	soMode1               = 13
	soMcntOut             = 14
	soF3o0                = 15
	soF3o1                = 16
	soF3o2                = 17
	soF3o3                = 18
	soF3o4                = 19
	soF3o5                = 20
	soF3o6                = 21
	soF3o7                = 22
	soC_mainAction        = 23
	soC_mainSpell         = 24
	soC_mainParameter     = 25
	soC_mainTargetWord0   = 26
	soC_mainTargetWord1   = 27
	soC_mainTargetWord2   = 28
	soC_mainTargetWord3   = 29
	soC_mainTargetWord4   = 30
	soC_mainTargetWord5   = 31
	soC_mainTargetWord6   = 32
	soC_mainTargetWord7   = 33
	soC_actionCompleted   = 34
	soC_deferredDefense   = 35
	soC_refreshMask       = 36
	soC_interceptAttempts = 37
)

// Fixture constants of the two child vectors (the case's global/arena
// defaults, not executable bytes): 0x53d590 (279 inputs) and 0x5194e0 (78).
var (
	swBase53 = [279]int64{4, 0, 537145912, -1, 0, -1, 0, 0, 0, 0, 4, 1, 1, 536936448, 0, 1, 16, 0, 1, 301, 302, 303, 304, 1, 0, 0, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 1, 10, 0, 0, 0, 0, 0, 100, 20, 0, 1, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 5, 1, 10, 10, 10, 10, 10, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 65535, 4294967295, 3, 255, 0, 255, 0, 255, 0, 255, 0, 1, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 1, 1, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 101, 102, 103, 536936448, 201, 202, 203, 536936448, 77, 301, 302, 303, 304, 0, 0}
	swBase78 = [78]int64{0, 0, 0, 0, 1, 1, 1, 1, 0, 0, 0, 1, 10, 0, 0, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0, 1, 1, 1, 3, 4, 5, 0, 4, 4294967295, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0}
)

// Tables read from the player's Wiz8.exe at runtime: the two relation jump
// tables (0x555f0c, 0x555f20) and the slot table (0x51c8e0).
type attackSwingTables struct {
	relOther [5]int // other quadrant 0..3 -> relation, 4 -> consult relOwn
	relOwn   [4]int
	slot     [4]int64 // weapon slot 6..9 -> companion slot
}

var attackSwingTab, attackSwingErr = loadAttackSwingTables()

func loadAttackSwingTables() (attackSwingTables, error) {
	var t attackSwingTables
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("attack swings: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	rd := func(va uint32, n int, f func(i int, dest uint32) error) error {
		o, err := peOffset(b, va)
		if err != nil {
			return err
		}
		if len(b)-o < 4*n {
			return fmt.Errorf("attack swings: truncated table %#x", va)
		}
		for i := 0; i < n; i++ {
			if err := f(i, binary.LittleEndian.Uint32(b[o+4*i:])); err != nil {
				return err
			}
		}
		return nil
	}
	// Return blocks 0x555ece/ed3/ee7/eec answer 0/1/2/3; 0x555ed8 consults the second table.
	rel := map[uint32]int{0x555ece: 0, 0x555ed3: 1, 0x555ee7: 2, 0x555eec: 3}
	if err := rd(0x555f0c, 5, func(i int, d uint32) error {
		if v, ok := rel[d]; ok && i < 4 {
			t.relOther[i] = v
		} else if d == 0x555ed8 && i == 4 {
			t.relOther[i] = -1
		} else {
			return fmt.Errorf("attack swings: unknown relation destination %#x", d)
		}
		return nil
	}); err != nil {
		return t, err
	}
	if err := rd(0x555f20, 4, func(i int, d uint32) error {
		v, ok := rel[d]
		if !ok {
			return fmt.Errorf("attack swings: unknown relation destination %#x", d)
		}
		t.relOwn[i] = v
		return nil
	}); err != nil {
		return t, err
	}
	// 0x51c8b0 maps slots 6..9 through 0x51c8c3/c9/cf/d5 to 7, 6, 9, 8.
	slot := map[uint32]int64{0x51c8c3: 7, 0x51c8c9: 6, 0x51c8cf: 9, 0x51c8d5: 8}
	err = rd(0x51c8e0, 4, func(i int, d uint32) error {
		v, ok := slot[d]
		if !ok {
			return fmt.Errorf("attack swings: unknown slot destination %#x", d)
		}
		t.slot[i] = v
		return nil
	})
	return t, err
}

// attackSwingsNode domain (panics outside it): attacker party slot 0..3,
// target character 4..7, target type 1 (character) or 3 (monster), main
// action 0..5, items 0..3 or -1, formation quadrants 0..4, last quadrants
// 0..3, finite float32 coordinates. A monster target needs tmon == mid and
// attack range <= 1. Messages, sound and the display refreshes are not state.
func attackSwingsNode(in []int64, r Rand) []int64 {
	if len(in) != 96 {
		panic("attack swings: expected 96 inputs")
	}
	if attackSwingErr != nil {
		panic(attackSwingErr)
	}
	if prayerTablesErr != nil {
		panic(prayerTablesErr)
	}
	g := func(i int) int64 { return in[i] }
	who, hand := g(swWho), int(g(swHand))
	if who < 0 || who > 3 { // 0x53d87c asserts who >= 8; party only here.
		panic("attack swings: expected attacker slot 0..3")
	}
	if hand < 0 || hand > 1 { // 0x53d8ed asserts.
		panic("attack swings: expected hand 0/1")
	}
	out := make([]int64, 38)
	out[soMode0], out[soMode1] = 1, 77 // fixture cells the node may overwrite
	out[soTswings0], out[soTswings1], out[soTls] = g(swSwings0), g(swSwings1), g(swLsFlag)
	out[soMcntOut] = g(swMcnt)
	for k := 0; k < 8; k++ {
		out[soF3o0+k] = g(swF3_0 + k)
	}
	// 0x53d590 child vector: fixture defaults plus this row's target cells.
	cv := swBase53
	t := g(swTchar)
	if t < 0 || t > 7 {
		panic("attack swings: expected target character 0..7")
	}
	cv[0] = t
	cv[27+3*t], cv[28+3*t], cv[29+3*t] = g(swTpresent), g(swThp), g(swTstatus)
	cv[9], cv[10], cv[56] = g(swTmainA), g(swToffA), g(swTcur)
	cv[277], cv[278] = g(swTinterf), g(swTatt)
	// Child outputs when it is not called: its own input cells (see n_53d590.go).
	childOut := []int64{cv[9], cv[11], cv[272], cv[273], cv[274], cv[275], cv[276],
		cv[264], cv[265], cv[266], cv[267], cv[261], cv[262], cv[263], cv[278]}
	copy(out[soC_mainAction:], childOut)
	finish := func(ret int64) []int64 {
		for i := 1; i < len(out); i++ {
			out[i] = int64(uint32(out[i]))
		}
		out[0] = ret
		return out
	}
	if g(swEnabled0+hand) == 0 { // 0x53d8f7..: hand disabled
		out[soTswings0+hand] = 0
		return finish(0)
	}
	rangeIn := []int64{int64(hand), g(swInParty), g(swKind0), g(swEnabled0), g(swKind1), g(swEnabled1), g(swItem6), g(swItem7),
		g(swIt0range), g(swIt1range), g(swIt2range), g(swIt3range)}
	v1 := charAttackRangeNode(rangeIn, r)[0]
	if v1 == -1 {
		out[soTswings0+hand] = 0
		return finish(0)
	}
	item := g(swItem6 + hand)
	if item < -1 || item > 3 {
		panic("attack swings: expected equipped item -1..3")
	}
	itemCell := func(idx int64, field int) int64 { return g(swIt0cls + 5*int(idx) + field) }
	mode := g(swMode)
	if mode == -1 {
		wm := int64(0)
		if item >= 0 {
			wm = itemCell(item, 4)
		}
		mode = chooseHandAttackModeNode([]int64{int64(hand), g(swKind0 + hand), g(swUmask0 + hand), item, wm}, r)[0]
	}
	out[soMode0+hand] = mode
	sw := g(swSwings0 + hand)
	if sw == 0 {
		return finish(0)
	}
	out[soTswings0+hand] = sw - 1

	// 0x536f60(who, hand, 0): the main action's target category must be valid
	// and in reach.
	act := g(swMainAction)
	if act < 0 || act > 5 {
		panic("attack swings: expected main action 0..5")
	}
	cat := [6]int{2, 2, 4, 0, 0, 1}[act]
	ttype := g(swTtype)
	if ttype != 1 && ttype != 3 {
		panic("attack swings: expected target type 1 or 3") // 0x53df06 asserts
	}
	if ttype == 3 && (g(swTmon) != g(swMid) || g(swTmon) != 0) {
		panic("attack swings: monster target must be list id 0")
	}
	valid := true
	switch cat {
	case 1, 2:
		if ttype == 1 {
			valid = g(swTpresent) != 0 && g(swThp) != 0 && uint32(g(swTstatus)) < 18
		} else {
			valid = g(swMhp) != 0 && uint32(g(swMexcl)) == 0 && g(swMactive) != 0
		}
	case 4:
		valid = false // spatial check never passes a character or monster target
	}
	if valid {
		if ttype == 3 {
			m := swBase78
			m[0], m[1] = int64(who), int64(hand)
			m[4], m[5], m[6] = g(swMen), g(swMphys), g(swMspell)
			m[8] = g(swCombat)
			m[14], m[15], m[16], m[17] = g(swKind0), g(swEnabled0), g(swKind1), g(swEnabled1)
			m[18], m[19], m[20] = g(swInParty), g(swItem6), g(swItem7)
			for k := 0; k < 4; k++ {
				m[21+k] = g(swIt0range + 5*k)
			}
			m[31] = act
			for k := 0; k < 3; k++ {
				m[38+k], m[41+k] = g(swPx+k), g(swMx+k)
			}
			m[44], m[45] = g(swMrad), g(swPrad)
			for k := 0; k < 8; k++ {
				m[46+k] = g(swF0_0 + k)
			}
			valid = monsterTargetAttackRangeNode(m[:], r)[0] != 0
		} else if t != who {
			var rr int64
			switch act { // 0x519a9c dispatch of 0x5199f0
			case 0, 1:
				rr = v1
			case 2:
				rr = 2
			case 5:
				rr = 0
			default:
				rr = -1
			}
			if rr == -1 {
				valid = false
			} else if rr == 0 {
				pf := []int64{who, t}
				for k := 0; k < 8; k++ {
					pf = append(pf, g(swF0_0+k))
				}
				pf = append(pf, 255, 255, 255)
				pf = append(pf, make([]int64, 8)...)
				valid = partyFormationRangePenaltyNode(pf, r)[0] == 0
			}
		}
	}
	if !valid {
		return finish(0)
	}

	// Swing roll.
	var t70, t74 int64 = 6, -1
	if hand == 1 {
		t70 = 7
	} else {
		sub := int64(0)
		if item >= 0 {
			sub = itemCell(item, 1)
		}
		if weaponDamageBonusItemNode([]int64{item, sub}, r)[0] != 0 {
			t74 = attackSwingTab.slot[0]
		}
	}
	out[soT70], out[soT74] = t70, t74
	d := g(swDice0 + hand)
	t68 := int64(uint32(r.R(int(d)) + 1))
	// 0x547940(ch, 9): the class or race feature rows hold ability 9.
	has9 := false
	if c := g(swCls); c >= 0 {
		for _, f := range prayerTables.class[c] {
			has9 = has9 || f == 9
		}
	}
	if rc := g(swRace); rc >= 0 {
		for _, f := range prayerTables.race[rc] {
			has9 = has9 || f == 9
		}
	}
	hm := g(swHmode0 + hand)
	if has9 && v1 <= 0 && (hm == 0 || hm == 4) {
		out[soFlag9a5] = 1
	}
	if item != -1 {
		t68 += itemCell(item, 3)
	}
	if g(swLsFlag) != 0 && hand == 0 {
		if g(swHmode0) == 0 && g(swA15) == 0 {
			if g(swLevel)+int64(r.R(11)) >= 20 {
				t68 += 4
			} else {
				t68 += 3
			}
		}
		out[soTls] = 0
	}
	out[soT68] = t68
	out[soT78] = item
	out[soT7c] = item
	if t74 != -1 {
		switch t74 {
		case 6:
			out[soT7c] = g(swItem6)
		case 7:
			out[soT7c] = g(swItem7)
		default:
			panic("attack swings: companion slot outside 6/7")
		}
	}

	// Direction flags.
	if ttype == 3 && v1 <= 1 { // 0x53de3e: v1>1 skips the direction block
		mf := make([]int64, mfInputs)
		mf[mfAttackerX], mf[mfAttackerZ] = g(swPx), g(swPz)
		mf[mfVictimX], mf[mfVictimZ], mf[mfVictimFacing] = g(swMx), g(swMz), g(swMhead)
		if !monsterFrontal(mf) {
			skip := false
			if g(swMhp) != 0 && g(swM33) != 0 && uint32(g(swM107)) < 14 && uint32(g(swM87)) == 0 && g(swM2e1) == 1 {
				cnt := uint32(g(swMcnt))
				skip = cnt == 0 || uint32(r.R(100)) < uint32(g(swM24b))-25*cnt
				out[soMcntOut] = int64(cnt + 1)
			}
			if !skip && monsterRear(mf) {
				out[soFlag9a4] = 1
			}
		}
	} else if v1 <= 1 {
		f0 := func(c int64) int {
			v := g(swF0_0 + int(c))
			if v < 0 || v > 4 {
				panic("attack swings: formation quadrant outside 0..4")
			}
			return int(v)
		}
		rel := func(a, b int64) int {
			own, oth := f0(a), f0(b)
			if own == oth {
				return 4
			}
			if res := attackSwingTab.relOther[oth]; res >= 0 {
				return res
			}
			if own > 3 {
				panic("attack swings: relation of two centre quadrants")
			}
			return attackSwingTab.relOwn[own]
		}
		last := g(swF3_0 + int(t))
		rl := rel(t, who)
		if !(rl == 4 || int64(rl) == last) {
			// 0x53d590(target): an intercept attempt may consume R(100).
			res := Nodes["0053d590"].F(cv[:], r)
			copy(out[soC_mainAction:], res[1:])
			if res[0] != 0 {
				if rl != 4 && last != int64(rl) {
					out[soF3o0+int(t)] = int64(rl)
				}
			} else if rl != 4 {
				diff := int64(rl) - last
				if diff < 0 {
					diff = -diff
				}
				if diff == 2 {
					out[soFlag9a4] = 1
				}
			}
		}
	}
	if v1 >= 2 {
		out[soTa5] = 1
		return finish(1)
	}
	r.R(3) // attack sound pick
	return finish(1)
}

func init() {
	Nodes["0053d870"] = Node{Addr: "0053d870", In: []string{"who", "mode", "flag510c", "hand", "swings0", "swings1", "lsFlag", "kind0", "kind1", "enabled0", "enabled1", "hmode0", "hmode1", "dice0", "dice1", "umask0", "umask1", "item6", "item7", "a15", "cls", "race", "level", "inParty", "mainAction", "ttype", "tchar", "tmon", "tgrp", "it0cls", "it0sub", "it0range", "it0bonus", "it0wmask", "it1cls", "it1sub", "it1range", "it1bonus", "it1wmask", "it2cls", "it2sub", "it2range", "it2bonus", "it2wmask", "it3cls", "it3sub", "it3range", "it3bonus", "it3wmask", "tpresent", "thp", "tstatus", "tmainA", "toffA", "tcur", "tinterf", "tatt", "mid", "mhp", "m33", "m107", "m87", "m2e1", "m24b", "mexcl", "mactive", "men", "mphys", "mspell", "mcnt", "combat", "px", "py", "pz", "mx", "my", "mz", "mrad", "prad", "mhead", "f0_0", "f0_1", "f0_2", "f0_3", "f0_4", "f0_5", "f0_6", "f0_7", "f3_0", "f3_1", "f3_2", "f3_3", "f3_4", "f3_5", "f3_6", "f3_7"},
		Out: []string{"ret", "T68", "T70", "T74", "T78", "T7c", "Ta5", "Tswings0", "Tswings1", "Tls", "flag9a4", "flag9a5", "mode0", "mode1", "mcntOut", "f3o0", "f3o1", "f3o2", "f3o3", "f3o4", "f3o5", "f3o6", "f3o7", "c_mainAction", "c_mainSpell", "c_mainParameter", "c_mainTargetWord0", "c_mainTargetWord1", "c_mainTargetWord2", "c_mainTargetWord3", "c_mainTargetWord4", "c_mainTargetWord5", "c_mainTargetWord6", "c_mainTargetWord7", "c_actionCompleted", "c_deferredDefense", "c_refreshMask", "c_interceptAttempts"}, F: attackSwingsNode}
}
