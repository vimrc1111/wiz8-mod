package calc

import "fmt"

func init() {
	in := []string{"partySlot", "characterBase", "class", "race", "battlePointer", "turnUndeadUsed", "monsterListPointer", "monsterCount"}
	for i := 0; i < 8; i++ {
		for _, name := range []string{"monsterPresent", "monsterActive", "monsterDisposition", "monsterHP", "templateIndex"} {
			in = append(in, fmt.Sprintf("%s[%d]", name, i))
		}
	}
	for i := 0; i < 3; i++ {
		in = append(in, fmt.Sprintf("templateType[%d]", i))
	}
	Nodes["00547bf0"] = Node{Addr: "00547bf0", In: in, Out: []string{"turnUndeadAvailable"}, F: turnUndeadAvailableNode}
}

// The selected character uses stride 6242 (0x547bfd..0x547c0a), and
// battle usage uses stride 212 and offset 177 (0x547c2a..0x547c34).
// Eight list entries and three independently populated loaded templates are
// represented by this adapter. Template selection is M+8 (0x4e5746).
func turnUndeadAvailableNode(in []int64, r Rand) []int64 {
	if len(in) != 51 || in[0] < 0 || in[0] > 7 || in[2] < -1 || in[2] > 14 || in[3] < -1 || in[3] > 15 || in[7] < 0 || in[7] > 8 {
		panic("turn undead: expected slot 0..7, class -1..14, race -1..15, count 0..8, eight monsters and three templates")
	}
	for i, v := range in {
		if i == 2 || i == 3 {
			continue
		}
		byteField := i == 5 || i >= 48 || i >= 8 && (i-8)%5 < 3
		if v < 0 || v > 0xffffffff || byteField && v > 255 || i >= 8 && i < 48 && (i-8)%5 == 4 && v > 2 {
			panic("turn undead: field width or template index overflow")
		}
	}
	if prayerTablesErr != nil {
		panic(prayerTablesErr)
	}
	// Share the runtime exe loader: class features at 0x61507c and race
	// features at 0x615130. Ability 17 is requested at 0x547c0d.
	ability := false
	if in[1] != 0 || in[0] != 0 {
		if in[2] != -1 {
			for _, code := range prayerTables.class[in[2]] {
				ability = ability || code == 17
			}
		}
		if in[3] != -1 {
			for _, code := range prayerTables.race[in[3]] {
				ability = ability || code == 17
			}
		}
	}
	if !ability || in[4] == 0 || in[5] != 0 || in[6] == 0 {
		return []int64{0}
	}
	for i := 0; i < int(in[7]); i++ {
		m := in[8+5*i : 13+5*i]
		// 0x547c5a..0x547c8a: nonzero present/active, disposition 1,
		// nonzero unsigned HP; template category 20 at D+203.
		if m[0] != 0 && m[1] != 0 && m[2] == 1 && m[3] != 0 && in[48+int(m[4])] == 20 {
			return []int64{1}
		}
	}
	return []int64{0}
}
