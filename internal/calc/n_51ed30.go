package calc

import "fmt"

// thrown_stack_hand_split 0x51ed30 (cdecl, arg0 character, arg1 direction):
// split the stack held in one of the character's two hand records
// (build/wiz8.asm:388728..388864; hand A = character +0xfa5, hand B = +0xfb1,
// 12 bytes each: i32 item id (-1 = empty), count byte +4, side byte +5, flag
// bytes +6 and +0xb, +0xa cleared on every move).
//   - direction 6: source is hand B; item record +0x3e must be 2. Count 1 swaps
//     the two hands via 0x51fd20; otherwise ceil(count/2) pieces go to hand A.
//   - direction 7: source is hand A; same type test. Count 1 is a no-op;
//     otherwise floor(count/2) pieces go to hand B.
//   - any other direction, an empty source or a record of another type: no-op.
//
// Callees: 0x51c020 (D4, run natively; its F is called directly, the dice
// return is the case input diceRet), 0x520d10 (D4, address stub: call count and
// owner/flag arguments are outputs; its first argument is a stack temporary in
// most calls so it is not compared). 0x51fd20 / 0x51fe30 (hand swap / record
// move) and 0x517ec0 (party slot lookup) run natively and are reproduced here.
//
// Fixture caps: item records thsItems (ids 0..3, 0x683f78 = 4 in the case) and
// eight party slots; ids outside panic. srAssertFail paths panic like the
// emulator stops: character not a party slot (0x517ec0), a move into a hand
// that is not empty (0x51fe30, so the split target / dst hand must be empty),
// dice maximum >= 256 and stack kind > 4 (inside 0x51c020). The party flag
// (+4) is a case constant 1. Hand addresses (heap character records) never lie
// in the pack range 0x685191..0x6868f5 or at the held-item instance, so the
// compaction tail of 0x51fe30 is not reached.
const (
	thsStride = 0x1862
	thsParty  = 8
	thsItems  = 4
	thsInst   = 0x20004000 // the inst object used by itemStackChargeInitNode
)

var thsIn = func() []string {
	n := []string{"ch", "op", "who", "partyBase", "itemsBase", "a0", "a1", "a2", "b0", "b1", "b2"}
	for j := 0; j < thsItems; j++ {
		for _, f := range []string{"type", "attr", "kind", "z", "n", "s"} {
			n = append(n, fmt.Sprintf("i%d.%s", j, f))
		}
	}
	n = append(n, "diceRet")
	for k := 0; k < thsParty; k++ {
		n = append(n, fmt.Sprintf("busy%d", k))
	}
	return n
}()

var thsOut = func() []string {
	n := []string{"a0", "a1", "a2", "b0", "b1", "b2"}
	for k := 0; k < thsParty; k++ {
		n = append(n, fmt.Sprintf("busy%d", k))
	}
	return append(n, "refreshCalls", "refreshOwner", "refreshFlag", "diceArg", "diceCalls")
}()

func init() {
	Nodes["0051ed30"] = Node{Addr: "0051ed30", In: thsIn, Out: thsOut, F: thrownStackHandSplitNode}
}

type thsHand [12]byte

func (h *thsHand) id() int64 {
	return int64(int32(uint32(h[0]) | uint32(h[1])<<8 | uint32(h[2])<<16 | uint32(h[3])<<24))
}
func (h *thsHand) clear() {
	*h = thsHand{0xff, 0xff, 0xff, 0xff}
}
func thsDword(h *thsHand, k int) int64 {
	return int64(uint32(h[4*k]) | uint32(h[4*k+1])<<8 | uint32(h[4*k+2])<<16 | uint32(h[4*k+3])<<24)
}
func thsSet(h *thsHand, k int, v int64) {
	for i := 0; i < 4; i++ {
		h[4*k+i] = byte(v >> (8 * uint(i)))
	}
}

func thrownStackHandSplitNode(in []int64, r Rand) []int64 {
	if len(in) != len(thsIn) {
		panic("thrown_stack_hand_split: wrong input count")
	}
	at := map[string]int{}
	for i, n := range thsIn {
		at[n] = i
	}
	v := func(n string) int64 { return in[at[n]] }
	ch, op, who := v("ch")&0xffffffff, v("op")&0xffffffff, v("who")
	if who < 0 || who >= thsParty || ch != (v("partyBase")+who*thsStride)&0xffffffff {
		panic("thrown_stack_hand_split: srAssertFail 0x517ec0, character is not a party slot")
	}
	var A, B thsHand
	for k := 0; k < 3; k++ {
		thsSet(&A, k, v(fmt.Sprintf("a%d", k)))
		thsSet(&B, k, v(fmt.Sprintf("b%d", k)))
	}
	busy := make([]int64, thsParty)
	for k := range busy {
		busy[k] = v(fmt.Sprintf("busy%d", k))
	}
	var refreshCalls, diceArg, diceCalls int64
	// 0x51fe30(dst, src): dst must be empty, src must hold an item.
	move := func(dst, src *thsHand) {
		if dst.id() != -1 {
			panic("thrown_stack_hand_split: srAssertFail 0x51fe30, destination record not empty")
		}
		if src.id() == -1 {
			panic("thrown_stack_hand_split: srAssertFail 0x51fe30, source record empty")
		}
		src[10] = 0
		*dst = *src
		src.clear()
		refreshCalls += 2 // 0x520d10(src, character, 1), then (dst, character, 1)
	}
	item := func(idx int64) func(string) int64 {
		if idx < 0 || idx >= thsItems {
			panic("thrown_stack_hand_split: item record past fixture cap")
		}
		return func(f string) int64 { return v(fmt.Sprintf("i%d.%s", idx, f)) }
	}
	// split the source hand: the new record is built by 0x51c020, then moved
	// into dst, and the source is refreshed.
	split := func(src, dst *thsHand, bl int64) {
		idx := src.id()
		iv := make([]int64, len(iscIn))
		set := func(n string, x int64) {
			for i, name := range iscIn {
				if name == n {
					iv[i] = x
					return
				}
			}
			panic("thrown_stack_hand_split: no 0x51c020 input " + n)
		}
		set("p", thsInst)
		set("id", idx)
		set("forceFlag6", int64(src[6]))
		set("setFlagB", int64(src[11]))
		set("itemsBase", v("itemsBase"))
		set("itemCount", thsItems)
		for j := 0; j < thsItems; j++ {
			ij := item(int64(j))
			set(fmt.Sprintf("i%d.kind", j), ij("kind"))
			set(fmt.Sprintf("i%d.z", j), ij("z"))
			set(fmt.Sprintf("i%d.n", j), ij("n"))
			set(fmt.Sprintf("i%d.s", j), ij("s"))
			set(fmt.Sprintf("i%d.attr", j), ij("attr"))
		}
		set("diceRet", v("diceRet"))
		o := itemStackChargeInitNode(iv, r)
		get := func(n string) int64 {
			for i, name := range iscOut {
				if name == n {
					return o[i]
				}
			}
			panic("thrown_stack_hand_split: no 0x51c020 output " + n)
		}
		var buf thsHand
		thsSet(&buf, 0, get("inst0"))
		thsSet(&buf, 1, get("inst4"))
		thsSet(&buf, 2, get("inst8"))
		diceArg, diceCalls = get("diceArg"), get("diceCalls")
		refreshCalls += get("refreshCalls")
		src[4] -= byte(bl)
		buf[4] = byte(bl)
		move(dst, &buf)
		refreshCalls++ // 0x520d10(source hand, character, 1)
	}
	// 0x51fd20(h1, h2): swap through a temporary; busy flag of the slot ends 0.
	swap := func(h1, h2 *thsHand) {
		if h1.id() == -1 && h2.id() == -1 {
			panic("thrown_stack_hand_split: srAssertFail 0x51fd20, both records empty")
		}
		switch {
		case h1.id() == -1:
			move(h1, h2)
		case h2.id() == -1:
			move(h2, h1)
		default:
			var tmp thsHand
			tmp.clear()
			refreshCalls++ // 0x520d10(tmp, 0, 1)
			move(&tmp, h1)
			move(h1, h2)
			move(h2, &tmp)
		}
		busy[who] = 0
	}

	switch op {
	case 6:
		if idx := B.id(); idx != -1 && item(idx)("type")&0xff == 2 {
			if cnt := int64(B[4]); cnt == 1 {
				swap(&A, &B)
			} else {
				split(&B, &A, cnt-cnt>>1)
			}
		}
	case 7:
		if idx := A.id(); idx != -1 && item(idx)("type")&0xff == 2 {
			if cnt := int64(A[4]); cnt != 1 {
				split(&A, &B, cnt>>1)
			}
		}
	}
	out := []int64{thsDword(&A, 0), thsDword(&A, 1), thsDword(&A, 2), thsDword(&B, 0), thsDword(&B, 1), thsDword(&B, 2)}
	out = append(out, busy...)
	owner, flag := int64(0), int64(0)
	if refreshCalls > 0 {
		owner, flag = ch, 1
	}
	// the last 0x520d10 call is (hand or temp, character, 1); the first
	// argument is not compared (stack temporary), see the file comment.
	return append(out, refreshCalls, owner, flag, diceArg, diceCalls)
}
