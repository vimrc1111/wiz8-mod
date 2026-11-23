package calc

import "fmt"

// monster_action_random_pick 0x5323f0 (cdecl, 4 stack args: monster, a, b,
// flag): rebuilds the monster's candidate list with 0x531ce0(monster, a,
// b), then picks one entry with R(count) and copies it into the monster:
// action type -> +0x2e1, attack index -> +0x2e5 (types 0 and 1), target block
// of 8 dwords -> +0x2ba; type 0 also copies entry +8 to combat record +0x11 and,
// when the low byte of flag is set, the template byte +0xe5 into combat +5/+9.
// Returns al = 1 on a pick, 0 for an empty or null list.
//
// Emu address-stubs the D4 callee 0x531ce0 (composition ruling 2026-10-09): its
// return is ignored by the caller, but it rebuilds the list, so the post-call
// list (count and entries) and the monster/target words it leaves behind are
// case inputs, and the stub's arguments and call count are outputs. F instead
// runs the real monsterActionCandidatesNode on a fixed world (the 533 trailing
// inputs) and feeds its recorded 0x532360 calls forward as the list entries
// and its +0x2e1/target words as the old monster state; so F and Emu differ
// wherever the case's own list differs from that world's.
//
// Fixture caps (panic past them): template id 0..3 (the template cache table
// at 0x6840c7 is filled for those slots; an empty slot loads from a file),
// emu list count 0..8. A negative count is not a list size; 0x5e2870 would
// return null for it (block 0x532440, unreachable for legal lists).
const (
	rpkNT        = 4
	rpkNE        = 8
	rpkListPtr   = 0x20000000 + 8320 // case 'hdr' object
	rpkMonPtr    = 0x20000000 + 8448 // case 'mon' object
	rpkParked    = 533               // inputs of 0x531ce0
	rpkEntryW    = 11
	rpkIn        = 112 // inputs before the parked world
	rpkCandRolls = 11  // 0x531ce0 output: n_roll
)

func init() {
	var in, out []string
	in = append(in, "candA", "candB", "flag", "templateId")
	for i := 0; i < rpkNT; i++ {
		in = append(in, fmt.Sprintf("template%dActionByte", i))
	}
	in = append(in, "listPtr", "count")
	for j := 0; j < rpkNE; j++ {
		for w := 0; w < rpkEntryW; w++ {
			in = append(in, fmt.Sprintf("entry%dWord%d", j, w))
		}
	}
	in = append(in, "oldActionType", "oldAttackIndex")
	for k := 0; k < 8; k++ {
		in = append(in, fmt.Sprintf("oldTarget%d", k))
	}
	in = append(in, "oldAttacks5", "oldAttacks9", "oldAttackKind", "candidatesRet")
	for i := 0; i < rpkParked; i++ {
		in = append(in, fmt.Sprintf("parked%d", i))
	}
	out = append(out, "actionType", "attackIndex")
	for k := 0; k < 8; k++ {
		out = append(out, fmt.Sprintf("target%d", k))
	}
	out = append(out, "attacks5", "attacks9", "attackKind", "picked", "candMonster", "candA", "candB", "candCalls")
	Nodes["005323f0"] = Node{Addr: "005323f0", In: in, Out: out,
		F:   func(in []int64, r Rand) []int64 { return monsterActionRandomPick(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return monsterActionRandomPick(in, r, false) }}
}

func monsterActionRandomPickNode(in []int64, r Rand) []int64 {
	return monsterActionRandomPick(in, r, true)
}

// rpkRollEntry is the record 0x532360 builds for roll call (a, b, c, d, e).
func rpkRollEntry(a, b, c, d, e int64) []int64 {
	w := []int64{a & 0xffffffff, b & 0xffffffff, 0, d & 0xffffffff, 0xffffffff, 0xffffffff, 0xffffffff, 0, 0, 0, 0}
	if a == 0 {
		w[2] = c & 0xffffffff
	}
	switch d {
	case 1:
		w[4] = e & 0xffffffff
	case 3:
		w[5] = e & 0xffffffff
	}
	return w
}

func monsterActionRandomPick(in []int64, r Rand, real bool) []int64 {
	if len(in) != rpkIn+rpkParked {
		panic("monster action random pick: wrong input count")
	}
	u := func(i int) int64 { return in[i] & 0xffffffff }
	ca, cb, flag, tid := u(0), u(1), u(2), u(3)
	listPtr, count := u(8), int64(int32(in[9]))
	entries := make([][]int64, 0, rpkNE)
	for j := 0; j < rpkNE; j++ {
		e := make([]int64, rpkEntryW)
		for w := range e {
			e[w] = u(10 + j*rpkEntryW + w)
		}
		entries = append(entries, e)
	}
	o := 10 + rpkNE*rpkEntryW
	type mon struct{ e1, e5 int64 }
	old := mon{u(o), u(o + 1)}
	tgt := make([]int64, 8)
	for k := range tgt {
		tgt[k] = u(o + 2 + k)
	}
	c5, c9, c11 := u(o+10), u(o+11), u(o+12)
	if real {
		// 0x531ce0 runs on its own world; the recorded 0x532360 calls are the list.
		p := in[rpkIn:]
		cand := monsterActionCandidatesNode(p, r)
		n := int(cand[rpkCandRolls])
		old.e1 = cand[0] & 0xffffffff
		for k := range tgt {
			tgt[k] = cand[1+k] & 0xffffffff
		}
		entries = entries[:0]
		for k := 0; k < n; k++ {
			entries = append(entries, rpkRollEntry(cand[19+k], cand[115+k], cand[211+k], cand[307+k], cand[403+k]))
		}
		count = int64(n)
	} else if count > rpkNE {
		panic("monster action random pick: fixture holds at most 8 list entries")
	}
	if tid >= rpkNT { // 0x4e5720: template cache slots 0..3 only
		panic("monster action random pick: template id past the fixture's cached slots")
	}
	if count < 0 {
		panic("monster action random pick: negative list count")
	}
	if listPtr == 0 { // 0x5e2c70: null list is zero entries
		count = 0
	} else if listPtr != rpkListPtr {
		panic("monster action random pick: list pointer is not the fixture list")
	}
	picked := int64(0)
	if count != 0 {
		e := entries[r.R(int(count))] // 0x40efa0 then 0x5e2870
		old.e1 = e[0]
		switch e[0] {
		case 0:
			c11, old.e5 = e[2], e[1]
			if flag&0xff != 0 {
				c5 = u(4 + int(tid))
				c9 = c5
			}
		case 1:
			old.e5 = e[1]
		}
		copy(tgt, e[3:11])
		picked = 1
	}
	out := []int64{old.e1, old.e5}
	out = append(out, tgt...)
	return append(out, c5, c9, c11, picked, rpkMonPtr, ca, cb, 1)
}
