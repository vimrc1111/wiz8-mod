package calc

import "fmt"

// all_condition_remove 0x5244a0(cond): two passes, build/wiz8.asm:396526..396569.
//
// Party: for each of the 8 slots whose presence byte (record at *0x685178,
// stride 0x106) is non-zero and whose state slot character +0xa01+4*cond
// (records at *0x685174, stride 0x1862) is non-zero, call
// character_condition_remove 0x523330(member, cond, 1). 0x523330 is D4 and is
// address-stubbed in the case; its return is ignored here (0x5244d7 only pops
// the arguments) and its effect on member j is never read back (later slots
// are other members), so every call's arguments and the call count are the
// outputs. No Emu/F split is needed.
//
// Monsters: for each entry i < count of the list at *0x683fad (0x5e2c70, NULL
// list counts 0; entry via 0x4e5620) whose duration +0x57+4*cond is non-zero,
// call monster_condition_remove 0x523f40(monster id, cond), which runs natively.
// Modelled projection of 0x523f40 (same as monsterConditionDecayNode): HP
// (+0x2b) zero skips it (0x523f68); otherwise the duration is cleared
// (0x524019), the primary condition +0x107 becomes the highest index 19..1 with
// a non-zero duration, else 0 (0x524021..0x52404b), and cond 7 also clears the
// poison strength +0x10b (0x5240c2). Derived caches and presentation rebuilt by
// it are outside this interface.
//
// Panics (never a silent answer):
//   - cond >= 20: the slots would be other record fields; the sole caller
//     0x4ea341 passes 5.
//   - list pointer other than NULL / the fixture list, or count > aclMonsters
//     (fixture record cap).
//   - a removal on a living monster that still has condition 19 afterwards:
//     0x523f40's cache rebuild 0x50e8c0 -> 0x50eac0 reaches 0x50ebf0 for
//     condition 19 with a NULL character and 0x517ec5 reads address 4, so the
//     original faults.
const (
	aclMembers  = 8
	aclMonsters = 3
	aclSlots    = 20
	aclList     = 0x20000000 + 0x14800 // fixture list object (case objects.list)
	aclMonIn    = 1 + aclSlots + 2     // hp, durations, poison, primary
)

func init() {
	in := []string{"cond", "monsterListPtr", "monsterCount"}
	for j := 0; j < aclMembers; j++ {
		in = append(in, fmt.Sprintf("present%d", j))
	}
	for j := 0; j < aclMembers; j++ {
		in = append(in, fmt.Sprintf("slot%d", j))
	}
	var out []string
	for k := 0; k < aclMonsters; k++ {
		in = append(in, fmt.Sprintf("m%d_hp", k))
		var f []string
		for i := 0; i < aclSlots; i++ {
			f = append(f, fmt.Sprintf("m%d_dur%d", k, i))
		}
		f = append(f, fmt.Sprintf("m%d_poison", k), fmt.Sprintf("m%d_primary", k))
		in, out = append(in, f...), append(out, f...)
	}
	for _, n := range []string{"who", "cond", "flag"} {
		for c := 0; c < aclMembers; c++ {
			out = append(out, fmt.Sprintf("%s%d", n, c))
		}
	}
	out = append(out, "calls")
	Nodes["005244a0"] = Node{Addr: "005244a0", In: in, Out: out, F: allConditionRemoveNode}
}

func allConditionRemoveNode(in []int64, r Rand) []int64 {
	if len(in) != 3+2*aclMembers+aclMonsters*aclMonIn {
		panic("all_condition_remove: wrong input count")
	}
	cond, listPtr, count := in[0], uint32(in[1]), uint32(in[2])
	if cond < 0 || cond >= aclSlots {
		panic("all_condition_remove: cond outside the 20 state slots")
	}
	if listPtr != 0 && listPtr != aclList {
		panic("all_condition_remove: monster list pointer outside the fixture")
	}
	if listPtr == 0 {
		count = 0
	}
	if count > aclMonsters {
		panic("all_condition_remove: monster count above fixture cap")
	}
	var who, conds, flags [aclMembers]int64
	calls := 0
	for j := 0; j < aclMembers; j++ {
		if in[3+j]&0xff != 0 && uint32(in[3+aclMembers+j]) != 0 {
			who[calls], conds[calls], flags[calls] = int64(j), cond, 1
			calls++
		}
	}
	var out []int64
	for k := 0; k < aclMonsters; k++ {
		m := in[3+2*aclMembers+k*aclMonIn : 3+2*aclMembers+(k+1)*aclMonIn]
		hp, dur := uint32(m[0]), append([]int64(nil), m[1:]...) // dur[20], dur[21] = poison, primary
		if uint32(k) < count && uint32(dur[cond]) != 0 && hp != 0 {
			dur[cond] = 0
			if uint32(dur[19]) != 0 {
				panic("all_condition_remove: condition 19 on a monster faults in 0x50eac0")
			}
			dur[21] = 0
			for i := aclSlots - 1; i > 0; i-- {
				if uint32(dur[i]) != 0 {
					dur[21] = int64(i)
					break
				}
			}
			if cond == 7 {
				dur[20] = 0
			}
		}
		out = append(out, dur...)
	}
	out = append(out, who[:]...)
	out = append(out, conds[:]...)
	out = append(out, flags[:]...)
	return append(out, int64(calls))
}
