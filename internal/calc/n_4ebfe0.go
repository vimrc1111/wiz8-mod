package calc

import "fmt"

// breathStaminaCostNode is 0x4ebfe0 breath_stamina_cost(idx): the breath
// attack of party character idx (build/wiz8.asm:316579..316770).
//
//   - Two empty id lists are built first, each with a 20-byte buffer from
//     operator new (0x4ec001/0x4ec021; heap 0x50000000 and 0x50000018).
//   - The effect object is *(*0x6836a8 + 0xa4c). A zero byte at effect+0x1e4
//     frees both buffers and returns 2 (0x4ec07e).
//   - Otherwise effect+0x1e6 = 1 (0x4ec09a) and 0x53be00 builds the character
//     source {1, idx, -1, zeros}. 0x4fd030(0x77, 0, &src, slot+0x1d, &monsters,
//     &characters, 0) fills the lists; slot is *0x685178 + 0x106*idx.
//   - 0x547120(&src, &monsters) runs natively. Its 0x547080 is
//     oppositeDispositionNode: for a character source it is 1 iff the charm
//     word C+0xa35 is 0. Then every monster id other than -1 (the character
//     source's id, 0x54712c) is found with 0x4e5550/0x4e5620. A record whose
//     group word +4 is nonzero calls 0x5471d0 (registered 排除). The count is
//     reread each iteration and 0x5471d0 never sees the list.
//   - By-value copies of the monster list (inline loop 0x4ec0ed..0x4ec139),
//     then the character list (0x4ed900), get new(count*4) buffers. Capacity
//     is set to count.
//   - 0x5560a0(0x1e, &src, slot+0x1d, characterCopy, monsterCopy).
//   - 0x52af50(idx, u32(C+0xb19)/5, 0, 0), with u32 max stamina C+0xb19
//     (mul 0xcccccccd, shr 2 at 0x4ec189..0x4ec199).
//   - Both list buffers are freed; the function returns 3 (0x4ec1cc).
//
// Stubs (case): 0x4fd030, 0x5560a0 and 0x52af50 are D4 formula callees,
// address-stubbed, each called at most once. Their arguments other than stack
// pointers and their call counts are outputs. The caller overwrites eax after
// each of them (0x4ec0e0, 0x4ec17a, 0x4ec1a3); with ret 0x5a5a5a5a in a
// scratch copy the diff stays 0. 0x5471d0 (排除) is stubbed with only its call
// count: its arguments are both stack pointers.
//
// The 0x4fd030 stub writes the list counts (charCount, monCount) and points
// the list buffers at the case's entry objects. This stands for the original
// growing a list past its five-entry buffer; buffer addresses are not outputs.
// The heap words after the two list buffers hold the copies, and they are
// outputs.
//
// Emu (what TestEmu replays) takes the lists from charCount/char0..7 and
// monCount/mon0..7.
//
// F runs the real spellAreaTargetSelectionNode instead. Its inputs are the
// call's own values (spell 0x77, power 0, character source idx, combat
// pointer nonzero, two empty lists of capacity 5) plus the F-only world
// inputs. Those are the slot's target words at slot+0x1d, the combat byte
// +0xa54, and the rest parked in a scratch object. F feeds its monster and
// character ids forward. Its monster ids (123) are the fixture's first
// record. It also replays the callee's buffer growth (n_4fd030.go buildList)
// on the shared heap.
//
// Fixture caps (calc panics past them): idx 0..7 (eight character records),
// list counts <= 8, monster ids 123..125 (the three fixture records; any
// other id hits the not-found assert in 0x4e5550).
const (
	bscArena  = 0x20000000
	bscSlots  = bscArena + 0x10000 // case object "slots" (*0x685178)
	bscHeap   = 0x50000000         // re/emu/emu.py operator new origin
	bscListVT = 0x5ebfe0           // id-list vtable, 0x4ebffc
	bscMaxIDs = 8
	bscNoID   = 0xffffffff // ebp = -1 for a character source, 0x54712c
)

var bscFixed = map[string]int64{
	"spellID": 0x77, "powerLevel": 0, // 0x4ec0d4, 0x4ec0d3
	"sourceKind": 1, "sourceMonster": -1, "sourceSpecial": 0, "sourceEnvironmental": 0, "sourceUseCenter": 0, // 0x53be00
	"battleObject":        1, // *0x6836a8 is the combat object the effect hangs off
	"monsterInitialCount": 0, "characterInitialCount": 0,
	"monsterCapacity": 5, "characterCapacity": 5, // 0x4ec012..0x4ec044
}

var bscIn, bscOut = bscNames()

func bscNames() ([]string, []string) {
	in := []string{"idx", "effectFlag", "effectBusyBefore", "charm", "maxStamina", "charCount"}
	for k := 0; k < bscMaxIDs; k++ {
		in = append(in, fmt.Sprintf("char%d", k))
	}
	in = append(in, "monCount")
	for k := 0; k < bscMaxIDs; k++ {
		in = append(in, fmt.Sprintf("mon%d", k))
	}
	in = append(in, "group123", "group124", "group125", "battleSuppressFallback",
		"targetKind", "targetCharacter", "targetMonster", "targetGroup", "targetXBits", "targetYBits", "targetZBits")
	have := map[string]bool{}
	for _, n := range in {
		have[n] = true
	}
	for _, n := range spellAreaInputs {
		if _, ok := bscFixed[n]; !ok && n != "sourceCharacter" && !have[n] {
			in = append(in, n)
		}
	}
	out := []string{"result", "effectBusy", "s4fd030_calls", "s4fd030_a1", "s4fd030_a2", "s4fd030_a4", "s4fd030_a7",
		"s5471d0_calls", "s5560a0_calls"}
	for _, a := range []int{1, 3, 4, 5, 6, 7, 8, 9, 10, 11} {
		out = append(out, fmt.Sprintf("s5560a0_a%d", a))
	}
	out = append(out, "s52af50_calls", "s52af50_a1", "s52af50_a2", "s52af50_a3", "s52af50_a4")
	for i := 0; i < 16; i++ {
		out = append(out, fmt.Sprintf("heap%02x", 0x30+4*i))
	}
	return in, out
}

func breathStaminaCostNode(in []int64, r Rand) []int64 { return bscRun(in, r, true) }

func bscRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(bscIn) {
		panic("breath stamina cost: input arity")
	}
	v := make(map[string]int64, len(in))
	for i, n := range bscIn {
		v[n] = in[i]
	}
	idx := v["idx"]
	if idx < 0 || idx > 7 {
		panic("breath stamina cost: character index past the eight fixture records")
	}
	o := make([]int64, len(bscOut))
	o[1] = v["effectBusyBefore"]
	if v["effectFlag"]&0xff == 0 {
		o[0] = 2
		return o
	}
	o[1] = 1
	slot := bscSlots + 0x106*idx + 0x1d
	o[2], o[3], o[4], o[5], o[6] = 1, 0x77, 0, slot, 0
	heap := int64(bscHeap + 0x30) // after the two 20-byte list buffers
	list := func(count string, entry string) []int64 {
		n := v[count]
		if n < 0 || n > bscMaxIDs {
			panic("breath stamina cost: list count past the eight-entry fixture")
		}
		ids := make([]int64, n)
		for k := range ids {
			ids[k] = v[fmt.Sprintf("%s%d", entry, k)]
		}
		return ids
	}
	var chars, mons []int64
	if real {
		child := make([]int64, len(spellAreaInputs))
		for i, n := range spellAreaInputs {
			if x, ok := bscFixed[n]; ok {
				child[i] = x
			} else if n == "sourceCharacter" {
				child[i] = idx
			} else {
				child[i] = v[n]
			}
		}
		a := spellAreaTargetSelectionNode(child, r)
		mons, chars = append([]int64(nil), a[2:2+a[0]]...), append([]int64(nil), a[10:10+a[1]]...)
		// n_4fd030.go buildList: growing past capacity 5 takes a new buffer
		// per entry, monsters first.
		for _, n := range []int64{a[0], a[1]} {
			for i := int64(5); i < n; i++ {
				heap += (4*(i+1) + 7) &^ 7
			}
		}
	} else {
		chars, mons = list("charCount", "char"), list("monCount", "mon")
	}
	if oppositeDispositionNode([]int64{1, idx, v["charm"], 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, r)[0] != 0 {
		for _, id := range mons {
			if id == bscNoID {
				continue
			}
			if id < 123 || id > 125 {
				panic("breath stamina cost: monster id outside the fixture records")
			}
			if v[fmt.Sprintf("group%d", id)] != 0 {
				o[7]++
			}
		}
	}
	words := map[int64]int64{}
	alloc := func(ids []int64) int64 {
		p := heap
		heap += max(8, (4*int64(len(ids))+7)&^7) // re/emu/emu.py allocation rounding
		for k, id := range ids {
			words[p+4*int64(k)] = id
		}
		return p
	}
	monData := alloc(mons)
	charData := alloc(chars)
	nc, nm := int64(len(chars)), int64(len(mons))
	copy(o[8:], []int64{1, 0x1e, slot, bscListVT, nc, nc, charData, bscListVT, nm, nm, monData})
	copy(o[19:], []int64{1, idx, (v["maxStamina"] & 0xffffffff) / 5, 0, 0})
	for i := int64(0); i < 16; i++ {
		o[24+i] = words[bscHeap+0x30+4*i]
	}
	o[0] = 3
	return o
}

func init() {
	Nodes["004ebfe0"] = Node{Addr: "004ebfe0", In: bscIn, Out: bscOut,
		F:   breathStaminaCostNode,
		Emu: func(in []int64, r Rand) []int64 { return bscRun(in, r, false) }}
}
