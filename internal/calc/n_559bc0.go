package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Tables of the player's own Wiz8.exe read by 0x559bc0: availability class
// (0x6155b4, 16-byte rows), skill x class availability (0x615840), primary skill
// per class (0x6161dc) and four feature skills per class (0x616218). Nothing is embedded.
type npcSkillConvExeTables struct {
	gate  [41]int32
	avail [41][15]int32
	main  [15]int32
	feat  [15][4]int32
}

var npcSkillConvExe, npcSkillConvExeErr = loadNpcSkillConvExe()

func loadNpcSkillConvExe() (t npcSkillConvExeTables, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	i32at := func(va uint32) (int32, error) {
		o, e := peOffset(b, va)
		if e != nil || o+4 > len(b) {
			return 0, fmt.Errorf("calc npc skill conversion: table %#x unreadable", va)
		}
		return int32(binary.LittleEndian.Uint32(b[o:])), nil
	}
	rd := func(dst *int32, va uint32) {
		if err == nil {
			*dst, err = i32at(va)
		}
	}
	for s := range t.gate {
		rd(&t.gate[s], 0x6155b4+16*uint32(s))
		for c := range t.avail[s] {
			rd(&t.avail[s][c], 0x615840+4*uint32(s*15+c))
		}
	}
	for c := range t.main {
		rd(&t.main[c], 0x6161dc+4*uint32(c))
		for k := range t.feat[c] {
			rd(&t.feat[c][k], 0x616218+16*uint32(c)+4*uint32(k))
		}
	}
	return t, err
}

func init() {
	names := []string{"skill", "class"}
	for i := 0; i < 6; i++ {
		names = append(names, fmt.Sprintf("domain%d", i))
	}
	for o := 0x178; o < 0x19a; o++ {
		names = append(names, fmt.Sprintf("src%x", o))
	}
	names = append(names, "src239")
	Nodes["00559bc0"] = Node{Addr: "00559bc0", In: names, Out: []string{"skillInit"}, F: npcSkillConversionNode}
}

// npcSkillConversionNode is exe 0x559bc0: an NPC skill's starting value, at most 20.
// In: skill index (0..40), class (character +0x69, 0..14), the six u32 domain values
// (character +0xe5d..+0xe74, which are also the +0xded+skill*4 slots for skills 0x1c..0x21),
// the source record bytes +0x178..+0x199 and its type byte +0x239.
// Skills 0x22..0x28 are class 3 in the exe table and return 0 before the dispatch; any other
// skill that reaches an out-of-range case would call srAssertFail (0x559cbb, 0x559f91), so it panics.
func npcSkillConversionNode(in []int64, r Rand) []int64 {
	if len(in) != 43 {
		panic("npc skill conversion: expected 43 inputs")
	}
	if npcSkillConvExeErr != nil {
		panic(npcSkillConvExeErr)
	}
	tb := &npcSkillConvExe
	skill, class := in[0], in[1]
	if skill < 0 || skill > 40 || class < 0 || class > 14 {
		panic("npc skill conversion: skill or class outside the exe tables")
	}
	for i, v := range in[2:] {
		hi := int64(255)
		if i < 6 {
			hi = 0xffffffff
		}
		if v < 0 || v > hi {
			panic("npc skill conversion: field width overflow")
		}
	}
	dom := in[2:8]
	b := func(off int) int64 { return in[8+off-0x178] }
	max := func(vs ...int64) int64 {
		m := vs[0]
		for _, v := range vs[1:] {
			if v > m {
				m = v
			}
		}
		return m
	}
	cap100 := func(v int64) int64 { return min(v, 100) }

	// 0x559bcf..0x559bf5: class 3 returns 0; class 2 needs the skill available to the class.
	switch tb.gate[skill] {
	case 3:
		return []int64{0}
	case 2:
		if tb.avail[skill][class] != 1 {
			return []int64{0}
		}
	}
	var v int64
	switch skill {
	case 9:
		v = max(b(0x17d), b(0x17e))
	case 0x10:
		v = cap100(max(b(0x178), b(0x179), b(0x17a), b(0x17b), b(0x17c), b(0x180), b(0x181)))
	case 0x11:
		v = cap100(max(b(0x17f), b(0x17e), b(0x17d)))
	case 0x12:
		if tb.avail[0x12][class] != 1 {
			break
		}
		v = cap100(max(b(0x178), b(0x179), b(0x17a), b(0x17b), b(0x17c)))
		// 0x559ec6..0x559eed: halved unless the class's primary skill or one of its four features is 0x12.
		if tb.main[class] != 0x12 {
			found := false
			for _, f := range tb.feat[class] {
				found = found || f == 0x12
			}
			if !found {
				v >>= 1
			}
		}
	case 0x14:
		v = (b(0x193) + b(0x190)*4) * 20 / 100
	case 0x17:
	case 0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x21:
		if d := dom[skill-0x1c]; d != 0 {
			if t := in[42]; t == 5 || t == 0xd || t == 4 {
				v = b(0x195)
			} else {
				v = b(0x186)
			}
			if d < 3 {
				v = (d + 1) * v / 3
			}
			v = cap100(v)
		}
	default:
		// Direct lookup: the source record byte 0x178 + slot (0x559c13..0x559cb9).
		slot := map[int64]int{0: 1, 1: 2, 2: 4, 3: 3, 4: 0, 5: 4, 6: 8, 7: 0x12, 8: 7, 10: 0x10, 11: 0x11,
			12: 0xd, 13: 0xf, 14: 9, 15: 0xc, 0x13: 0x21, 0x15: 0x19, 0x16: 0x1c, 0x18: 0x20, 0x19: 0x1e,
			0x1a: 0x1d, 0x1b: 0x1f}
		s, ok := slot[skill]
		if !ok {
			panic("npc skill conversion: srAssertFail (invalid skill)")
		}
		v = b(0x178 + s)
		if skill >= 0x18 && skill <= 0x1b {
			if class == 0xc && (skill == 0x1a || skill == 0x1b) && v == 0 {
				v = (b(0x198) + b(0x196)) / 2
			}
			var sum uint32
			for _, d := range dom {
				sum += uint32(d)
			}
			switch {
			case sum == 0:
				v /= int64(r.R(5)) + 5
			case sum < 5:
				v = int64(sum) * v / 5
			}
		}
	}
	return []int64{min(int64(uint32(2*v)/10), 20)}
}
