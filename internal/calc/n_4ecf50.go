package calc

import "fmt"

// combatSurprisePendingMonsters is the fixture cap (precedent 4ed550/4f0c80): the case
// lays out 8 monster records; a larger count panics rather than answer wrongly.
const combatSurprisePendingMonsters = 8

// Input layout: combatMode(0x683f94), skipPartyRoll(u8 arg0), surpriseBlockFlag(0x68749d),
// ambushFlag(0x6874ff), scenePointer(0x652dac), sceneFlags(u32 at scene+0), monsterListPointer,
// monsterCount, awarenessMode(0x683fc5), battle +0x1/+0xa52/+0xa53, 8 x (slotActive, currentHP,
// mainStatus, effectiveSEN) for 0x4ed1c0, then 8 x (present, active, hp, level, state, f28a, f34c).
const (
	csParty   = 12
	csMon     = csParty + 32
	csMonSize = 7
)

func init() {
	in := []string{"combatMode", "skipPartyRoll", "surpriseBlockFlag", "ambushFlag", "scenePointer", "sceneFlags",
		"monsterListPointer", "monsterCount", "awarenessMode", "battleByte1", "battleFlag52", "battleFlag53"}
	for i := 0; i < 8; i++ {
		for _, f := range []string{"slotActive", "currentHP", "mainStatus", "effectiveSEN"} {
			in = append(in, fmt.Sprintf("%s%d", f, i))
		}
	}
	for i := 0; i < combatSurprisePendingMonsters; i++ {
		for _, f := range []string{"present", "active", "hp", "field107", "state", "field28a", "field34c"} {
			in = append(in, fmt.Sprintf("monster%d%s", i, f))
		}
	}
	Nodes["004ecf50"] = Node{Addr: "004ecf50", In: in, Out: []string{"battleFlag52", "battleFlag53", "battleByte1"}, F: combatSurpriseFlagsNode}
}

// combatSurpriseFlagsNode (exe 0x4ecf50) sets the two lose-the-round flags at battle +0xa52/+0xa53
// and clears +0x1 when the first is set. The final log call (0x58ac00, window refresh) is not
// an output here; with the window object absent it is a no-op in the emulator.
// Combat mode 0 hits srAssertFail (exe 0x4ecf64), so it panics.
func combatSurpriseFlagsNode(in []int64, r Rand) []int64 {
	if len(in) != csMon+combatSurprisePendingMonsters*csMonSize {
		panic("combatSurpriseFlags: wrong input count")
	}
	for i, v := range in {
		max := int64(255)
		switch {
		case i == 4 || i == 6 || i == 7 || i == 5:
			max = 4294967295
		case i >= csParty && i < csMon:
			if (i-csParty)%4 != 0 {
				max = 4294967295
			}
		case i >= csMon:
			switch (i - csMon) % csMonSize {
			case 2, 3:
				max = 4294967295
			}
		}
		if v < 0 || v > max {
			panic("combatSurpriseFlags: input out of range")
		}
	}
	if in[0] == 0 {
		panic("combatSurpriseFlags: combat mode 0 asserts (exe 0x4ecf64)")
	}
	n := in[7]
	if in[6] == 0 { // exe 0x5e2c70: null list is zero monsters.
		n = 0
	}
	if n > combatSurprisePendingMonsters {
		panic("combatSurpriseFlags: only 8 monster fixture records")
	}
	// exe 0x4ecfb7..0x4ecfe2 and 0x4ed08a..0x4ed0b5: first listed monster that is present,
	// active, alive (unsigned hp != 0), field +0x107 below 16 (unsigned), state byte 1 and
	// the given flag byte 1.
	scan := func(flag int) bool {
		for i := int64(0); i < n; i++ {
			m := in[csMon+int(i)*csMonSize:]
			if m[0] != 0 && m[1] != 0 && m[2] != 0 && m[3] < 16 && m[4] == 1 && m[5+flag] == 1 {
				return true
			}
		}
		return false
	}
	a52, a53, b1 := in[10], in[11], in[9]
	if in[1] == 0 {
		if in[2] == 0 {
			if scan(0) {
				// exe 0x4ecffa: 0x4ed1c0 reads the mode and eight party slots.
				if partyAwarenessCheckNode(append([]int64{in[8]}, in[csParty:csMon]...), r)[0] != 0 {
					a52 = 1
				}
			} else {
				a52 = 1
			}
		}
		if a52 == 0 {
			// exe 0x41efb0: scene pointer non-null and bit 8 of its first word.
			if in[4] != 0 && in[5]>>8&1 != 0 {
				if r.R(100) < 20 {
					a52 = 1
				}
			} else if in[3] != 0 {
				if r.R(100) < 20 {
					a52 = 1
				}
			}
		}
	}
	if !scan(1) && in[8] == 0 {
		a53 = 1
	}
	if a52 != 0 && a53 != 0 {
		a52, a53 = 0, 0
	}
	if a52 != 0 {
		b1 = 0
	}
	return []int64{a52, a53, b1}
}
