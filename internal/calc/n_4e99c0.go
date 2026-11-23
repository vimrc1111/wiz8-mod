package calc

import (
	"fmt"
	"strconv"
)

// 0x4e99c0 combat_skill_growth(who): after a fight, each hand's swing counter (record
// G+0xd4*who+0x60+0x10*h, G = *0x6836a8) pays out that many practice rolls, then the
// pending-growth flags of skills 6, 0xb and 0x26 are cashed in.
//
// Per hand h (0, 1) with u32 count n = [+0], skill a = [+4], skill b = [+8], flag f = [+0xc]:
// threshold t = 75 if f != 0, else 50 if a == 14 else 100 (exe 0x4e9a0a..0x4e9a2d).
// Repeat n times: if R(100) < t, pick R(f != 0 ? 3 : 2) -> skill a, b or 0x12, call
// 0x553f10(C, skill, 1, 0); when that skill equals a and b == 0x10 and a second R(100) < 25,
// call it once more. Then n is decremented (exe 0x4e9ab0), so it ends at 0.
// Afterwards every flag k in 0..40 of G+0xd4*who+0xc0+k that is non-zero and k is 6, 0xb or
// 0x26 calls 0x553f10(C, k, 2, 0) and is cleared.
//
// C = *0x685174 + 0x1862*who (wraps at 32 bits). 0x553f10 is a D4 callee, address-stubbed in
// the case: its return (never read, eax is overwritten) and its writes (character record, not
// the combat record) are not read back, so the output is its call count and recorded arguments.
//
// Replay caps (calc panics past them): who 0..7 (record count), n <= 16 per hand (record
// would otherwise need an unbounded call log).
const (
	csgRecords = 8
	csgMaxN    = 16
	csgFlags   = 41
	csgCalls   = 4*csgMaxN + 3 // 2 hands * n * 2 calls + 3 flag calls
	csgInCount = 2 + 2*4 + csgFlags
)

func init() {
	var inN, outN []string
	inN = append(inN, "who", "charBase")
	for j := 0; j < 2; j++ {
		for _, f := range []string{"cnt", "id", "alt", "flag"} {
			inN = append(inN, f+strconv.Itoa(j))
		}
	}
	for k := 0; k < csgFlags; k++ {
		inN = append(inN, "pend"+strconv.Itoa(k))
	}
	outN = append(outN, "cnt0", "cnt1")
	for k := 0; k < csgFlags; k++ {
		outN = append(outN, "pend"+strconv.Itoa(k))
	}
	outN = append(outN, "callCount")
	for a := 1; a <= 4; a++ {
		for c := 0; c < csgCalls; c++ {
			outN = append(outN, fmt.Sprintf("arg%d_%d", a, c))
		}
	}
	Nodes["004e99c0"] = Node{Addr: "004e99c0", In: inN, Out: outN, F: combatSkillGrowthNode}
}

func combatSkillGrowthNode(in []int64, r Rand) []int64 {
	if len(in) != csgInCount {
		panic("combat skill growth: wrong input count")
	}
	for _, x := range in {
		if x < 0 || x > 1<<32-1 {
			panic("combat skill growth: input outside u32")
		}
	}
	who, base := in[0], in[1]
	if who >= csgRecords {
		panic("combat skill growth: character index outside the 8 records")
	}
	cnt := [2]int64{in[2], in[6]}
	id := [2]int64{in[3], in[7]}
	alt := [2]int64{in[4], in[8]}
	flag := [2]int64{in[5], in[9]}
	var pend [csgFlags]int64
	copy(pend[:], in[10:])
	if flag[0] > 255 || flag[1] > 255 {
		panic("combat skill growth: byte flag outside u8")
	}
	for _, p := range pend {
		if p > 255 {
			panic("combat skill growth: byte flag outside u8")
		}
	}
	if cnt[0] > csgMaxN || cnt[1] > csgMaxN {
		panic("combat skill growth: swing count past the fixture cap")
	}
	c := (base + 0x1862*who) & 0xffffffff
	var args [4][csgCalls]int64
	calls := 0
	call := func(skill, repeat int64) {
		args[0][calls], args[1][calls], args[2][calls], args[3][calls] = c, skill, repeat, 0
		calls++
	}
	for h := 0; h < 2; h++ {
		thr, pick := int64(100), 2
		if flag[h] != 0 {
			thr, pick = 75, 3
		} else if id[h] == 14 {
			thr = 50
		}
		for ; cnt[h] > 0; cnt[h]-- {
			if int64(r.R(100)) >= thr {
				continue
			}
			skill := int64(0x12)
			switch r.R(pick) {
			case 0:
				skill = id[h]
			case 1:
				skill = alt[h]
			}
			call(skill, 1)
			if skill == id[h] && alt[h] == 0x10 && r.R(100) < 25 {
				call(skill, 1)
			}
		}
	}
	for k := 0; k < csgFlags; k++ {
		if pend[k] != 0 && (k == 6 || k == 0xb || k == 0x26) {
			call(int64(k), 2)
			pend[k] = 0
		}
	}
	out := []int64{cnt[0], cnt[1]}
	out = append(out, pend[:]...)
	out = append(out, int64(calls))
	for a := range args {
		out = append(out, args[a][:]...)
	}
	return out
}
