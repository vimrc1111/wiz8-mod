package calc

import "fmt"

func init() {
	in := append([]string{}, cgInNames...)
	for k := 0; k < atuSeq; k++ {
		in = append(in, fmt.Sprintf("sq%d", k))
	}
	out := append(append([]string{}, cgOutNames...), "cnt_4c5eb0", "rec_4c5eb0_1", "rec_4c5eb0_2", "rec_4c5eb0_3")
	Nodes["0053bf80"] = Node{Addr: "0053bf80", In: in, Out: out, F: partyAutoTargetUpdateNode}
}

// atuSeq is the fixture cap on 0x4e79a0 calls per run (case "ret":{at,n}); calc
// panics on the call after the last supplied return, as the emulator does.
const atuSeq = 48

func partyAutoTargetUpdateNode(in []int64, r Rand) []int64 {
	if len(in) != len(cgIn)+atuSeq {
		panic("0x53bf80: wrong input count")
	}
	w := cgWorld(in)
	w.seq = in[len(cgIn):]
	w.autoTargetUpdate()
	out := cgRead(w)
	const base = cgA + 0x17000
	out = append(out, w.u32(base+0x800+4*cgS4c5eb0))
	for k := int64(0); k < 3; k++ {
		out = append(out, w.u32(base+0x200+0x40*cgS4c5eb0+4*k))
	}
	return out
}

// autoTargetUpdate is 0x53bf80(): for each of the eight party slots (exe loop
// bound 0xc310 = 8 * 0x1862), re-pick the stored attack target. Slots 4..7 are
// outside the four-slot fixture: their cells are not inputs and read as absent
// (S[0] = 0), so the loop skips them; a present one panics (fixture cap, see n_520d10.go).
func (w *cgW) autoTargetUpdate() {
	for x := int64(0); x < 8; x++ {
		S := w.u32(0x685178) + 0x106*x
		C := w.u32(0x685174) + 0x1862*x
		if w.u8(S) == 0 {
			continue
		}
		w.who(x) // slots 4..7 are outside the four-slot fixture: panic if one is present
		if w.u32(C+0xb11) == 0 && w.u32(C+0xb01) >= 0x12 {
			continue
		}
		p := w.rptr(x, 6) // 0x53b7f0
		if w.st(cgS4e79a0, x, 6, 0, 0)&0xff != 0 {
			w.release(x, w.rec(p)) // 0x53a930
		} else if w.st(cgS4e79a0, x, 6, 1, 0)&0xff != 0 {
			w.target(x, 6, 0) // 0x536570
		} else if w.i32(p) != 0 {
			w.store(x, cgRec{0, -1, -1, -1}, 6)
		}
		// 0x524a10 reads S[0], HP and the main status again, after the calls above.
		if characterActionEligibilityNode([]int64{x, w.u8(S), w.u32(C + 0xb11), w.u32(C + 0xb01)}, nil)[0] == 0 {
			continue
		}
		if k := w.i32(S + 0x3d); k != 0 && k != 1 {
			continue
		}
		if w.u8(S+0x105) == 0 && w.st(cgS4e79a0, x, 1, 0, 0)&0xff != 0 {
			continue
		}
		u6 := int64(-1)
		if id := w.i32(S + 0x55); id != -1 {
			if e := w.find(id, false); e != -1 {
				u6 = w.i32(w.entry(e) + 4)
			}
		}
		if r := w.autoFire(x, u6, 1); r != -1 { // 0x53c990
			w.store(x, cgRec{3, -1, r, -1}, 1)
		}
	}
}
