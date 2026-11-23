package calc

import "fmt"

// party_item_identification 0x5209f0(item instance): when the instance's flag
// byte +6 is zero, visit party members 0..7. A member takes part when its
// presence byte (record at *0x685178, stride 0x106, byte 0) is non-zero, its
// +0xb11 word is non-zero and its state word +0xb01 is below 11 (all unsigned,
// record at *0x685174, stride 0x1862); each such member calls
// itemIdentificationNode 0x5208f0(member, item). The result is the low byte of
// the first call that returned a non-zero low byte; later members are still
// visited (their results are ignored once one succeeded). A non-zero +6 returns
// 0 without any call.
//
// Emu address-stubs the D4 callee 0x5208f0: its return is the case input retK
// (call k reads ret_k), and the outputs are every call's two arguments and the
// call count. F runs the real itemIdentificationNode for each call on the F-only
// inputs fItemId, fReq (the item definition's +0x85 byte) and fScore_j (member
// j's +0x49b), feeding its result forward into the first-success rule.
const (
	pi9Members  = 8
	pi9CharBase = 0x20000000 // emu arena start; the case puts the character records at offset 0
	pi9CharStep = 0x1862
	pi9Inst     = 0x20011000 // the case's inst object (arena offset 0x11000)
)

// input positions
const (
	pi9Flag6  = 0
	pi9Pres   = 1
	pi9Pools  = pi9Pres + pi9Members
	pi9State  = pi9Pools + pi9Members
	pi9Ret    = pi9State + pi9Members
	pi9ItemID = pi9Ret + pi9Members
	pi9Req    = pi9ItemID + 1
	pi9Score  = pi9Req + 1
	pi9Len    = pi9Score + pi9Members
)

func init() {
	in := []string{"flag6"}
	for _, p := range []string{"pres", "pools", "state", "ret"} {
		for j := 0; j < pi9Members; j++ {
			in = append(in, fmt.Sprintf("%s%d", p, j))
		}
	}
	in = append(in, "fItemId", "fReq")
	for j := 0; j < pi9Members; j++ {
		in = append(in, fmt.Sprintf("fScore%d", j))
	}
	out := []string{"result", "calls"}
	for k := 0; k < pi9Members; k++ {
		out = append(out, fmt.Sprintf("callChar%d", k))
	}
	for k := 0; k < pi9Members; k++ {
		out = append(out, fmt.Sprintf("callItem%d", k))
	}
	Nodes["005209f0"] = Node{Addr: "005209f0", In: in, Out: out,
		F:   func(in []int64, r Rand) []int64 { return partyItemIdentification(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return partyItemIdentification(in, r, false) }}
}

func partyItemIdentificationNode(in []int64, r Rand) []int64 {
	return partyItemIdentification(in, r, true)
}

func partyItemIdentification(in []int64, r Rand, real bool) []int64 {
	if len(in) != pi9Len {
		panic("party_item_identification: wrong input count")
	}
	for i, v := range in {
		hi := int64(1<<32 - 1) // pools, state, ret, scores are words
		if i <= pi9Pres+pi9Members-1 || i == pi9Req {
			hi = 255 // flag6, presence bytes, definition byte
		}
		if i == pi9ItemID {
			hi = 818 // fixture cap: ITEMS.DBS record count (notes/re/loot.md:9)
		}
		if v < 0 || v > hi {
			panic(fmt.Sprintf("party_item_identification: input %d outside its memory width", i))
		}
	}
	o := make([]int64, 2+2*pi9Members)
	if in[pi9Flag6] != 0 { // 0x5209fb
		return o
	}
	var bl, calls int64
	for j := 0; j < pi9Members; j++ {
		if in[pi9Pres+j] == 0 || in[pi9Pools+j] == 0 || in[pi9State+j] >= 0xb { // 0x520a0f, 0x520a24, 0x520a28
			continue
		}
		var res int64
		if real {
			v := []int64{int64(j), pi9CharBase + pi9CharStep*int64(j), in[pi9ItemID], in[pi9Req],
				in[pi9Pools+j], in[pi9State+j], in[pi9Score+j], 0, 0, 0, 0, 0, 0, 0, 0}
			res = itemIdentificationNode(v, r)[7]
		} else {
			res = in[pi9Ret+int(calls)] & 0xff // call k reads ret_k; only the low byte (bl) is kept
		}
		o[2+calls], o[2+pi9Members+calls] = pi9CharBase+pi9CharStep*int64(j), pi9Inst
		calls++
		if bl == 0 { // 0x520a31: once bl is set the later results are dropped
			bl = res
		}
	}
	o[0], o[1] = bl, calls
	return o
}
