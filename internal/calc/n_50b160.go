package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// npc_recruit_skill_state 0x50b160: an NPC joins the party. The template must
// have +0x57 set; the new member slot comes from 0x4ef4a0(npc+0x27, byte
// npc+0x2c) (-1 = no slot, return 0). Every skill with a non-zero base value
// (member +0x19f + 0x26k) gets its mark byte (+0x1b0 + 0x26k) set, then the
// skill availability pass 0x553cd0 re-derives the enable bytes (+0x19d+0x26k)
// for the 41 skills through predicate 0x553d90. When npc+0x1a and npc+0x25 are
// both non-zero the NPC's item record (id npc+0x16, 0x4e5550 / 0x4e5620) hands
// each non-zero item word (+0x57 + 4i, 20 words) to the member through
// 0x522fe0 (0x524250), and the record is removed through 0x4e3af0. Then the NPC
// is marked joined (+0x26 = 1, +0x25 = 0, +0x2b = slot, +0xc5 = +0xe9 = 0),
// 0x55a0a0(npc+0) and 0x524ca0(npc) are called and npc+0 is cleared. Finally
// template +0x12c (byte) is stored at 0x619eac[0] (zero fills 0x619eac[0..4]);
// a non-zero value is added to the first free byte of the five at 0x6875aa
// unless already present. The return is always 1 on this path.
//
// Emu address-stubs 0x4ef4a0, 0x55a0a0, 0x524ca0, 0x4e3af0, 0x5b1af0, 0x5b1b30 (registry class
// 排除) and the D4 callee 0x522fe0 (its return is ignored by 0x524250; this node
// does not read what it writes, so no Emu/F split): the outputs are the
// recorded arguments and call counts. The slot returned by 0x4ef4a0 is an input.
//
// Fixture caps (ID ranges / record counts only; calc panics past them): slot
// -1..7, class 0..14 and race -1..15 (exe table rows), spell realm 0..5 (the
// 553cd0 flag array has 6 bytes; a larger value overwrites the stack), NPC item
// record id 0..7 (ids 0..3 in the normal list at 0x683fad, 4..7 in the special
// list at 0x683fb9, each id one record, so 0x4e5550 returns the id or
// 0x2710 + id - 4), member present flag (+4) set (0x517ec0 calls the assert
// handler 0x5ebaf4 when it is clear, an error path, not a legal state).
// Game mode 0x68ec78 is an input; only 0x553cd0 reads it on this path: mode 3
// calls 0x5b1af0(skill) when a skill turns on and 0x5b1b30(skill) when it turns
// off (stubbed, args and call counts are outputs).
const (
	rssSkills  = 0x29
	rssSpells  = 0x71
	rssItems   = 20
	rssSlots   = 8
	rssClasses = 15
	rssNpc     = 0x20000000 // fixture: npc object at arena offset 0 (emu.md)
	rssNormal  = 4          // item records 0..3 are in the normal list
	rssRecords = 8
	rssAbility = 0x1f // skill/spell ban ability tested by 0x553d90 via 0x547940
)

type rssSpec struct {
	name string
	n    int
}

var rssIn = []rssSpec{
	{"tmpl57", 1}, {"tmpl12c", 1}, {"npc16", 1}, {"npc1a", 1}, {"npc25", 1}, {"npc27", 1}, {"npc2c", 1}, {"npc0", 1},
	{"npc26_init", 1}, {"npc2b_init", 1}, {"npce9_init", 1}, {"npcc5_init", 1},
	{"slot", 1}, {"slotIdx", 1}, {"class", 1}, {"race", 1},
	{"enable", rssSkills}, {"base", rssSkills}, {"mark", rssSkills},
	{"grant", 6}, {"attr", 7}, {"classStat", rssClasses},
	{"spellState", rssSpells}, {"spellRealm", rssSpells},
	{"item", rssItems}, {"item10b", 1}, {"mode", 1},
	{"slot619eac", 5}, {"slot6875aa", 5},
}

var rssOut = []rssSpec{
	{"ret", 1}, {"enable", rssSkills}, {"mark", rssSkills},
	{"npc26", 1}, {"npc2b", 1}, {"npc25", 1}, {"npce9", 1}, {"npcc5", 1}, {"npc0", 1},
	{"slotArg1", 1}, {"slotArg2", 1}, {"slotCalls", 1},
	{"clearArg1", 1}, {"clearCalls", 1},
	{"npcArg1", 1}, {"npcCalls", 1},
	{"removeArg1", 1}, {"removeArg2", 1}, {"removeCalls", 1},
	{"applyArg1_", rssItems}, {"applyArg2_", rssItems}, {"applyArg3_", rssItems}, {"applyArg4_", rssItems},
	{"applyArg5_", rssItems}, {"applyArg6_", rssItems}, {"applyCalls", 1},
	{"uiAddArg", rssSkills}, {"uiAddCalls", 1}, {"uiDelArg", rssSkills}, {"uiDelCalls", 1},
	{"slot619eac", 5}, {"slot6875aa", 5},
}

func rssNames(s []rssSpec) []string {
	var o []string
	for _, f := range s {
		if f.n == 1 {
			o = append(o, f.name)
			continue
		}
		for i := 0; i < f.n; i++ {
			o = append(o, fmt.Sprintf("%s%d", f.name, i))
		}
	}
	return o
}

func init() {
	Nodes["0050b160"] = Node{Addr: "0050b160", In: rssNames(rssIn), Out: rssNames(rssOut), F: npcRecruitSkillStateNode}
}

// rssTable holds the exe tables 0x553d90 reads: 0x615840 (via LoadExe), 0x6161dc,
// 0x616218 (4 skills per class), 0x616310, and the jump table of 0x553d90's
// last switch (byte index at 0x553ec4 per skill 0..27, targets at 0x553eb8).
type rssTable struct {
	exe   *ExeTables
	extra [rssClasses][4]int32
	stat  [rssClasses]int32
	jump  [28]uint32
}

var rssExe, rssExeErr = rssLoad()

func rssLoad() (*rssTable, error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return nil, fmt.Errorf("npc recruit: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	t := &rssTable{}
	var err error
	if t.exe, err = LoadExe(path); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	at := func(va uint32, n int) ([]byte, error) {
		o, e := peOffset(b, va)
		if e != nil {
			return nil, e
		}
		if o+n > len(b) {
			return nil, fmt.Errorf("npc recruit: table %#x truncated", va)
		}
		return b[o : o+n], nil
	}
	for c := 0; c < rssClasses; c++ {
		r, e := at(0x616218+16*uint32(c), 16)
		if e != nil {
			return nil, e
		}
		for j := range t.extra[c] {
			t.extra[c][j] = i32(r[4*j:])
		}
		if r, e = at(0x616310+4*uint32(c), 4); e != nil {
			return nil, e
		}
		t.stat[c] = i32(r)
	}
	idx, err := at(0x553ec4, len(t.jump))
	if err != nil {
		return nil, err
	}
	tg, err := at(0x553eb8, 12)
	if err != nil {
		return nil, err
	}
	for s := range t.jump {
		if idx[s] > 2 {
			return nil, fmt.Errorf("npc recruit: jump index %d", idx[s])
		}
		t.jump[s] = binary.LittleEndian.Uint32(tg[4*idx[s]:])
	}
	return t, nil
}

// rssMember is the part of the member record 0x553d90 reads.
type rssMember struct {
	class, race int64
	enable      [rssSkills]int64
	grant       [6]uint32
	attr        [7]uint32
	classStat   [rssClasses]int32
}

// rssAvailable is 0x553d90(member, skill, flags): flags are the six realm bytes.
func rssAvailable(t *rssTable, m *rssMember, s int, flags *[6]bool) bool {
	if t.exe.Avail[s][m.class] == 0 {
		return false
	}
	banned := potionFeature(m.class, m.race, rssAbility) // 0x547940(member, 0x1f)
	if banned && s >= 0x18 && s <= 0x21 {
		return false
	}
	if s >= 0x1c && s <= 0x21 {
		held := false
		for k := 0x18; k <= 0x1b; k++ {
			held = held || m.enable[k] != 0
		}
		if !held {
			return false
		}
		return m.grant[s-0x1c] > 0 || flags[s-0x1c]
	}
	if s == t.exe.MainSkill[m.class] {
		return true
	}
	for _, x := range t.extra[m.class] {
		if int32(s) == x {
			return true
		}
	}
	if s >= 0x22 && s <= 0x28 {
		return m.attr[s-0x22] >= 100
	}
	if s > 0x1b {
		panic("npc recruit: 0x553d90 skill above 0x28")
	}
	switch t.jump[s] {
	case 0x553e8d:
		v := t.stat[m.class]
		if v >= 0 || v <= -0xff {
			panic("npc recruit: 0x553e8d class stat outside the exe table values")
		}
		return m.classStat[m.class]+v > 0
	case 0x553eb0:
		return true
	}
	panic("npc recruit: unknown 0x553d90 switch target")
}

func npcRecruitSkillStateNode(in []int64, r Rand) []int64 {
	if len(in) != len(rssNames(rssIn)) {
		panic("npc recruit: wrong input count")
	}
	if rssExeErr != nil {
		panic(rssExeErr)
	}
	if breathTablesErr != nil {
		panic(breathTablesErr)
	}
	p := 0
	rd := func(n int) []int64 { p += n; return append([]int64(nil), in[p-n:p]...) }
	tmpl57, tmpl12c, npc16, npc1a, npc25 := rd(1)[0], rd(1)[0], rd(1)[0], rd(1)[0], rd(1)[0]
	npc27, npc2c, npc0 := rd(1)[0], rd(1)[0], rd(1)[0]
	n26, n2b, ne9, nc5 := rd(1)[0], rd(1)[0], rd(1)[0], rd(1)[0]
	slot, slotIdx, class, race := rd(1)[0], rd(1)[0], rd(1)[0], rd(1)[0]
	enable, base, mark := rd(rssSkills), rd(rssSkills), rd(rssSkills)
	grant, attr, classStat := rd(6), rd(7), rd(rssClasses)
	state, realm := rd(rssSpells), rd(rssSpells)
	item, item10b, mode := rd(rssItems), rd(1)[0], rd(1)[0]
	g619, g6875 := rd(5), rd(5)

	var o struct {
		ret                                 int64
		slotArg1, slotArg2, slotCalls       int64
		clearArg1, clearCalls               int64
		npcArg1, npcCalls                   int64
		removeArg1, removeArg2, removeCalls int64
		apply                               [6][rssItems]int64
		applyCalls                          int64
		uiAdd, uiDel                        [rssSkills]int64
		uiAddCalls, uiDelCalls              int64
	}
	emit := func() []int64 {
		out := []int64{o.ret}
		out = append(out, enable...)
		out = append(out, mark...)
		out = append(out, n26, n2b, npc25, ne9, nc5, npc0)
		out = append(out, o.slotArg1, o.slotArg2, o.slotCalls, o.clearArg1, o.clearCalls, o.npcArg1, o.npcCalls)
		out = append(out, o.removeArg1, o.removeArg2, o.removeCalls)
		for a := range o.apply {
			out = append(out, o.apply[a][:]...)
		}
		out = append(out, o.applyCalls)
		out = append(out, o.uiAdd[:]...)
		out = append(out, o.uiAddCalls)
		out = append(out, o.uiDel[:]...)
		out = append(out, o.uiDelCalls)
		out = append(out, g619...)
		return append(out, g6875...)
	}
	if tmpl57 == 0 { // 0x50b16e
		return emit()
	}
	o.slotArg1, o.slotArg2, o.slotCalls = npc27, npc2c, 1 // 0x4ef4a0(npc+0x27, byte npc+0x2c)
	if slot == -1 {
		return emit()
	}
	// The fixture reads the member record by slotIdx; it must be the record
	// 0x4ef4a0 returns (slotIdx is 0 when no slot is returned).
	if slotIdx != max(slot, 0) {
		panic("npc recruit: fixture record index differs from the 0x4ef4a0 slot")
	}
	if slot < 0 || slot >= rssSlots || class < 0 || class >= rssClasses || race < -1 || race > 15 || npc16 < 0 || npc16 >= rssRecords {
		panic("npc recruit: slot, class, race or item record id outside the fixture caps")
	}
	for k := 0; k < rssSkills; k++ { // 0x50b1b3
		if uint32(base[k]) != 0 {
			mark[k] = 1
		}
	}
	// 0x553cd0: realm flags from the 0x71 spell states, then the 41 skills in order.
	var flags [6]bool
	for i := 0; i < rssSpells; i++ {
		if uint32(state[i]) == 0xffffffff || uint32(state[i]) == 2 {
			if realm[i] < 0 || realm[i] > 5 {
				panic("npc recruit: spell realm above 5 overwrites the 0x553cd0 stack frame")
			}
			flags[realm[i]] = true
		}
	}
	m := rssMember{class: class, race: race}
	copy(m.enable[:], enable)
	for i := range m.grant {
		m.grant[i] = uint32(grant[i])
	}
	for i := range m.attr {
		m.attr[i] = uint32(attr[i])
	}
	for i := range m.classStat {
		m.classStat[i] = int32(classStat[i])
	}
	for s := 0; s < rssSkills; s++ {
		ok := rssAvailable(rssExe, &m, s, &flags)
		if ok && m.enable[s] == 0 {
			m.enable[s] = 1
			if uint32(mode) == 3 { // 0x553d49: 0x5b1af0(skill)
				o.uiAdd[o.uiAddCalls] = int64(s)
				o.uiAddCalls++
			}
		} else if !ok && m.enable[s] != 0 {
			m.enable[s] = 0
			if uint32(mode) == 3 { // 0x553d60: 0x5b1b30(skill)
				o.uiDel[o.uiDelCalls] = int64(s)
				o.uiDelCalls++
			}
		}
		enable[s] = m.enable[s]
	}
	if npc1a != 0 && npc25 != 0 { // 0x50b1db / 0x50b1e5
		// 0x4e5550 then 0x4e5620: ids 0..3 sit in the normal list, 4..7 in the special list.
		idx := npc16
		if npc16 >= rssNormal {
			idx = 0x2710 + npc16 - rssNormal
		}
		for i := 0; i < rssItems; i++ { // 0x524250
			if uint32(item[i]) != 0 {
				k := o.applyCalls
				for a, v := range []int64{slot, int64(i), item[i], item10b, 0, 0} {
					o.apply[a][k] = v
				}
				o.applyCalls++
			}
		}
		o.removeArg1, o.removeArg2, o.removeCalls = idx, 1, 1 // 0x4e3af0(0x4e5550(record id), 1)
	}
	n2b, n26, npc25, ne9, nc5 = slot&0xff, 1, 0, 0, 0
	o.clearArg1, o.clearCalls = npc0, 1 // 0x55a0a0(npc+0)
	npc0 = 0
	o.npcArg1, o.npcCalls = rssNpc, 1 // 0x524ca0(npc)
	for i := 0; i < 5; i++ {          // 0x50b279: the template byte is the same on every pass
		g619[i] = tmpl12c
		if tmpl12c != 0 {
			break
		}
	}
	o.ret = 1
	if tmpl12c == 0 {
		return emit()
	}
	for i := 0; i < 5; i++ {
		if g6875[i] == tmpl12c {
			return emit()
		}
	}
	for i := 0; i < 5; i++ {
		if g6875[i] == 0 {
			g6875[i] = tmpl12c
			break
		}
	}
	return emit()
}
