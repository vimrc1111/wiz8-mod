package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	in := []string{"partySlot", "characterBase", "class", "race", "battlePointer", "prayerUsed", "monsterListPointer", "monsterCount"}
	// Eight arena records in the case, in list order, independently populated.
	for i := 0; i < 8; i++ {
		for _, name := range []string{"monsterPresent", "monsterActive", "monsterDisposition", "monsterHP"} {
			in = append(in, fmt.Sprintf("%s[%d]", name, i))
		}
	}
	Nodes["00547f40"] = Node{Addr: "00547f40", In: in, Out: []string{"prayerAvailable"}, F: prayerAvailableNode}
}

type prayerAbilityTables struct {
	class [15][3]int
	race  [16][5]int32
}

var prayerTables, prayerTablesErr = loadPrayerAbilityTables()

// Read the player's tables at runtime; no executable table contents are embedded.
func loadPrayerAbilityTables() (prayerAbilityTables, error) {
	var t prayerAbilityTables
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("prayer: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	character, err := LoadExe(path)
	if err != nil {
		return t, err
	}
	t.class = character.ClassFeat
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	// 0x547979: race features start at 0x615130, five i32 per 20-byte row.
	for i := range t.race {
		o, err := peOffset(b, 0x615130+uint32(i*20))
		if err != nil {
			return t, err
		}
		if len(b)-o < 20 {
			return t, fmt.Errorf("prayer: race table truncated")
		}
		for j := range t.race[i] {
			t.race[i][j] = int32(binary.LittleEndian.Uint32(b[o+j*4:]))
		}
	}
	return t, nil
}

// Inputs describe one selected character and up to eight list records. Party
// stride 6242 and battle stride 212 are from 0x547f40 and 0x547f76; the slot
// affects arena addressing, not the rule. Both identity sentinels are -1
// (0x54794c / 0x547971). Only AL is the boolean return (0x547fd8 / 0x547fdc).
func prayerAvailableNode(in []int64, r Rand) []int64 {
	if len(in) != 40 || in[0] < 0 || in[0] > 7 || in[2] < -1 || in[2] > 14 || in[3] < -1 || in[3] > 15 || in[7] < 0 || in[7] > 8 {
		panic("prayer: expected party slot 0..7, class -1..14, race -1..15, count 0..8 and eight monster records")
	}
	for i, v := range in {
		if i == 2 || i == 3 {
			continue
		}
		if v < 0 || v > 0xffffffff || ((i == 5 || i >= 8 && (i-8)%4 != 3) && v > 255) {
			panic("prayer: field width overflow")
		}
	}
	if prayerTablesErr != nil {
		panic(prayerTablesErr)
	}
	available := false
	// Ability code 11 is requested at 0x547f5d; class rows contain three
	// features at 0x61507c (LoadExe), followed by the race union in 0x547940.
	if in[1] != 0 {
		if in[2] != -1 {
			for _, code := range prayerTables.class[in[2]] {
				available = available || code == 11
			}
		}
		if in[3] != -1 {
			for _, code := range prayerTables.race[in[3]] {
				available = available || code == 11
			}
		}
	}
	if !available || in[4] == 0 || in[5] != 0 || in[6] == 0 {
		return []int64{0}
	}
	for i := 0; i < int(in[7]); i++ {
		m := in[8+4*i : 12+4*i]
		// 0x547fa6..0x547fc2: present, active, hostile (1), nonzero u32 HP.
		if m[0] != 0 && m[1] != 0 && m[2] == 1 && m[3] != 0 {
			return []int64{1}
		}
	}
	return []int64{0}
}
