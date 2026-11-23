package calc

// Fixture wiring (case 00535200): seven group records and seven monster
// records. Slots 0-2 are the active lists (IDs 1-3, entries beyond the input
// count are not visited); slots 3-6 are the reserve lists (IDs 1-4). Lookups
// return the first match, active before reserve (0x4e5550, 0x5100b0). Group
// slot layout is 14 inputs: flag28, flag29, flag2a, v0xa3, kind, state, count,
// link0-3, memberCount, member0-1; monster slot layout is 3 inputs.
const (
	mgasGroups   = 7
	mgasGroupLen = 14
	mgasMonBase  = mgasGroups * mgasGroupLen
	mgasBattle   = mgasMonBase + 7*3
)

func monsterGroupActionStateNode(in []int64, _ Rand) []int64 {
	if len(in) != mgasBattle+3 {
		panic("monster group action state: wrong input count")
	}
	var f [mgasBattle + 3]int64
	copy(f[:], in)
	field := func(g, k int) *int64 { return &f[g*mgasGroupLen+k] }
	activeGroups, activeMons := int(f[mgasBattle+1]), int(f[mgasBattle+2])
	// 0x5100b0 / 0x4e5550: active list (IDs 1-3), then reserve (IDs 1-4).
	// A miss is the original's assertion, not a legal state.
	lookup := func(id int64, active int, what string) int {
		for i := 0; i < active && i < 3; i++ {
			if id == int64(i+1) {
				return i
			}
		}
		if id >= 1 && id <= 4 {
			return 3 + int(id) - 1
		}
		panic("monster group action state: " + what + " ID not in replay fixture")
	}
	// 0x511be0 body at 0x511c20..0x511c45 (state byte, counter byte).
	apply := func(g int, flag int64) {
		st, ct := field(g, 5), field(g, 6)
		if *st&0xff != flag {
			*st, *ct = flag, 0
		}
		if *st&0xff != 0 {
			*ct = (*ct + 1) & 0xff
		}
	}
	// 0x511be0: kinds 0x1b6/0x234 with flag 1 leave everything untouched.
	update := func(id, flag int64) {
		g := lookup(id, activeGroups, "group")
		if k := *field(g, 4); (k == 0x1b6 || k == 0x234) && flag == 1 {
			return
		}
		apply(g, flag)
		for l := 7; l < 11; l++ {
			if lid := *field(g, l); lid != 0 {
				apply(lookup(lid, activeGroups, "linked group"), flag)
			}
		}
	}
	// 0x5352b0..0x5352c4: member monster flags +0x14/+0x15 set and level < 3.
	danger := func(g int) bool {
		n := int(*field(g, 11))
		for i := 0; i < n && i < 2; i++ {
			m := lookup(*field(g, 12+i), activeMons, "monster")
			b := mgasMonBase + m*3
			if f[b]&0xff != 0 && f[b+1]&0xff != 0 && uint32(f[b+2]) < 3 {
				return true
			}
		}
		return false
	}
	if uint32(f[mgasBattle]) >= 3 { // 0x535215: battle value +0xa5c below 3 returns.
		for g := 0; g < activeGroups && g < 3; g++ {
			if *field(g, 0)&0xff == 0 || *field(g, 1)&0xff == 0 || *field(g, 2)&0xff != 1 || uint32(*field(g, 3)) != 0 {
				continue
			}
			flag := int64(1)
			hit := danger(g)
			for l := 7; l < 11 && !hit; l++ {
				if lid := *field(g, l); lid != 0 {
					hit = danger(lookup(lid, activeGroups, "linked group"))
				}
			}
			if hit || *field(g, 6)&0xff >= 3 { // 0x535386: counter byte below 3 gives flag 1.
				flag = 0
			}
			update(int64(g+1), flag)
		}
	}
	out := make([]int64, 0, 14)
	for g := 0; g < mgasGroups; g++ {
		out = append(out, *field(g, 5)&0xff, *field(g, 6)&0xff)
	}
	return out
}

func init() {
	Nodes["00535200"] = Node{
		Addr: "00535200",
		In:   []string{"g0flag28", "g0flag29", "g0flag2a", "g0v0xa3", "g0kind", "g0state", "g0count", "g0link0", "g0link1", "g0link2", "g0link3", "g0memberCount", "g0member0", "g0member1", "g1flag28", "g1flag29", "g1flag2a", "g1v0xa3", "g1kind", "g1state", "g1count", "g1link0", "g1link1", "g1link2", "g1link3", "g1memberCount", "g1member0", "g1member1", "g2flag28", "g2flag29", "g2flag2a", "g2v0xa3", "g2kind", "g2state", "g2count", "g2link0", "g2link1", "g2link2", "g2link3", "g2memberCount", "g2member0", "g2member1", "g3flag28", "g3flag29", "g3flag2a", "g3v0xa3", "g3kind", "g3state", "g3count", "g3link0", "g3link1", "g3link2", "g3link3", "g3memberCount", "g3member0", "g3member1", "g4flag28", "g4flag29", "g4flag2a", "g4v0xa3", "g4kind", "g4state", "g4count", "g4link0", "g4link1", "g4link2", "g4link3", "g4memberCount", "g4member0", "g4member1", "g5flag28", "g5flag29", "g5flag2a", "g5v0xa3", "g5kind", "g5state", "g5count", "g5link0", "g5link1", "g5link2", "g5link3", "g5memberCount", "g5member0", "g5member1", "g6flag28", "g6flag29", "g6flag2a", "g6v0xa3", "g6kind", "g6state", "g6count", "g6link0", "g6link1", "g6link2", "g6link3", "g6memberCount", "g6member0", "g6member1", "m0flag14", "m0flag15", "m0level", "m1flag14", "m1flag15", "m1level", "m2flag14", "m2flag15", "m2level", "m3flag14", "m3flag15", "m3level", "m4flag14", "m4flag15", "m4level", "m5flag14", "m5flag15", "m5level", "m6flag14", "m6flag15", "m6level", "battleCount", "activeGroupCount", "activeMonsterCount"},
		Out: []string{"g0state", "g0count", "g1state", "g1count", "g2state", "g2count", "g3state", "g3count",
			"g4state", "g4count", "g5state", "g5count", "g6state", "g6count"},
		F: monsterGroupActionStateNode,
	}
}
