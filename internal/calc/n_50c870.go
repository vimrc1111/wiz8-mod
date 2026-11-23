package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	npcRecruitRecSlots   = 16 // modelled quest-item records (0x685191, stride 12)
	npcRecruitPartySlots = 8
	npcRecruitListA      = 12 // character +0xf5d, stride 12
	npcRecruitListB      = 8  // character +0x1029, stride 12
)

// Input layout (names double as the emulator case's `in` order).
func npcRecruitNames() []string {
	n := []string{"kind", "npcFlag57", "npcMask68", "npcThreshold6f", "selector",
		"f8b", "f20d", "f1a7", "fbf", "f1ce", "g6874ca", "g6874cb", "recCount"}
	for k := 0; k < npcRecruitRecSlots; k++ {
		n = append(n, fmt.Sprintf("recKey%d", k))
	}
	for _, p := range []string{"present", "level", "hp", "grade"} {
		for i := 0; i < npcRecruitPartySlots; i++ {
			n = append(n, fmt.Sprintf("%s%d", p, i))
		}
	}
	for i := 0; i < npcRecruitPartySlots; i++ {
		for k := 0; k < npcRecruitListA; k++ {
			n = append(n, fmt.Sprintf("listA%d_%d", i, k))
		}
	}
	for i := 0; i < npcRecruitPartySlots; i++ {
		for k := 0; k < npcRecruitListB; k++ {
			n = append(n, fmt.Sprintf("listB%d_%d", i, k))
		}
	}
	return n
}

type npcRecruitExeData struct {
	slotTable [48]int8 // 0x6044e2 + 107*r (r 0..0x2f)
	keys      []int32  // 0x619dfc, stride 12, until key -1
	masks     []uint32 // 0x619e00, same stride
}

var npcRecruitExe, npcRecruitExeErr = loadNpcRecruitExe()

// Tables come from the player's own Wiz8.exe; nothing is embedded.
func loadNpcRecruitExe() (npcRecruitExeData, error) {
	var t npcRecruitExeData
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	at := func(va uint32, n int) ([]byte, error) {
		o, e := peOffset(b, va)
		if e != nil {
			return nil, e
		}
		if o+n > len(b) {
			return nil, fmt.Errorf("calc npcRecruit: truncated table %#x", va)
		}
		return b[o : o+n], nil
	}
	for r := range t.slotTable {
		v, e := at(0x6044e2+107*uint32(r), 1)
		if e != nil {
			return t, e
		}
		t.slotTable[r] = int8(v[0])
	}
	for i := uint32(0); ; i++ {
		v, e := at(0x619dfc+12*i, 8)
		if e != nil {
			return t, e
		}
		k := int32(binary.LittleEndian.Uint32(v))
		if k == -1 {
			return t, nil
		}
		if i > 64 {
			return t, fmt.Errorf("calc npcRecruit: unterminated table 0x619dfc")
		}
		t.keys = append(t.keys, k)
		t.masks = append(t.masks, binary.LittleEndian.Uint32(v[4:]))
	}
}

func init() {
	Nodes["0050c870"] = Node{Addr: "0050c870", In: npcRecruitNames(), Out: []string{"recruitable"}, F: npcRecruitLevelGateNode}
}

// npcRecruitLevelGateNode follows exe 0x50c870 (npc_recruit_level_gate), with
// callees folded in: 0x42b740 -> 0x4d9700 (selector remap by quest flags
// 0x5b/0x1ce) -> table 0x6044e2; 0x506280 -> 0x5080f0, which for ids
// 0x8b,0x20d,0x1a7,0xbf,0x1ce returns byte 0x689b78+id and for id 0x5b counts
// how many of items 0x242..0x244 are held (0x508480 -> 0x521060 -> 0x520f90,
// called with mode 2, no out pointers). Debug logging (0x688290) is off.
// ponytail: the 0x685191 record list is modelled with npcRecruitRecSlots
// entries; recCount above that panics rather than guess. Game capacity is
// ~499; raise the constant (and the case) if bigger lists must be diffed.
func npcRecruitLevelGateNode(in []int64, r Rand) []int64 {
	if npcRecruitExeErr != nil {
		panic(npcRecruitExeErr)
	}
	names := npcRecruitNames()
	if len(in) != len(names) {
		panic("npcRecruitLevelGate: wrong input count")
	}
	pos := 0
	next := func() int64 { v := in[pos]; pos++; return v }
	kind, flag57, mask68, thr := next(), next(), uint32(next()), uint8(next())
	_ = r
	sel := int32(next())
	f8b, f20d, f1a7, fbf, f1ce := next() != 0, next() != 0, next() != 0, next() != 0, next() != 0
	g74ca, g74cb, recCount := next() != 0, uint32(next()), uint32(next())
	if recCount > npcRecruitRecSlots {
		panic(fmt.Sprintf("npcRecruitLevelGate: recCount %d exceeds modelled cap %d", recCount, npcRecruitRecSlots))
	}
	var rec [npcRecruitRecSlots]uint32
	for k := range rec {
		rec[k] = uint32(next())
	}
	var present [8]bool
	var level, hp, grade [8]uint32
	for i := range present {
		present[i] = next() != 0
	}
	for i := range level {
		level[i] = uint32(next())
	}
	for i := range hp {
		hp[i] = uint32(next())
	}
	for i := range grade {
		grade[i] = uint32(next())
	}
	var listA [8][npcRecruitListA]uint32
	var listB [8][npcRecruitListB]uint32
	for i := range listA {
		for k := range listA[i] {
			listA[i][k] = uint32(next())
		}
	}
	for i := range listB {
		for k := range listB[i] {
			listB[i][k] = uint32(next())
		}
	}
	if flag57 == 0 {
		return []int64{0}
	}
	b2i := func(b bool) []int64 {
		if b {
			return []int64{1}
		}
		return []int64{0}
	}

	// exe 0x521060 (mode 2, no match pointers): has item id?
	has := func(id uint32) bool {
		if g74ca && g74cb == id {
			return true
		}
		for i := range present {
			if !present[i] {
				continue
			}
			for _, v := range listA[i] {
				if v == id {
					return true
				}
			}
			for _, v := range listB[i] {
				if v == id {
					return true
				}
			}
		}
		for k := uint32(0); k < recCount; k++ {
			if rec[k] == id {
				return true
			}
		}
		return false
	}
	held := 0
	for id := uint32(0x242); id <= 0x244; id++ {
		if has(id) {
			held++
		}
	}
	// exe 0x4d9700 remaps selector 0x10/0x26 by flags 0x5b (held >= 2) and 0x1ce.
	remapped := sel
	if sel == 0x10 || sel == 0x26 {
		remapped = 0x26
		if held >= 2 || f1ce {
			remapped = 0x10
		}
	}
	// exe 0x42b740: 0..0x2f indexes the table, anything else gives 0.
	var key int32
	if remapped >= 0 && remapped <= 0x2f {
		key = int32(npcRecruitExe.slotTable[remapped])
	}
	for i, k := range npcRecruitExe.keys {
		if k == key {
			if uint32(mask68)&npcRecruitExe.masks[i] != 0 {
				return b2i(false)
			}
			break
		}
	}
	// exe 0x50c8ba..: kind-specific quest flags (0x50c93e repeats the 0x4a
	// check on the same flag, so it adds nothing).
	switch {
	case kind == 0x2b && f8b, kind == 0x4f && !f20d, kind == 0x4a && !f1a7,
		(kind == 0x10 || kind == 0x11) && fbf:
		return b2i(false)
	}
	if thr == 0 {
		return b2i(true)
	}
	// exe 0x50c96e: mean level (unsigned, 32-bit wrap) of present members with
	// hp > 0 and grade < 15 against the NPC threshold.
	var count, sum uint32
	for i := range present {
		if present[i] && hp[i] > 0 && grade[i] < 15 {
			count++
			sum += level[i]
		}
	}
	var avg uint32
	if count > 0 {
		avg = sum / count
	}
	return b2i(avg >= uint32(thr))
}
