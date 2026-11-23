package calc

import (
	"strconv"
	"strings"
)

// ---- 0x5326f0: pick one monster spell target at random ----
//
// The original allocates a 5-record (0xa0 bytes, 0x5e1ce0) candidate list, lets
// 0x533320 fill it for (monster, spell id), and with no candidate returns 0. Otherwise it
// draws R(count) (0x40efa0), copies that 32-byte record into the monster's eight
// target words at +0x2ba and returns 1. The candidate builder is reused as is (its
// fixture: monster id 0..2, spell id 0..255, state arms 2 and 3).

const (
	mstRndList  = 0x7f000100 // the original's stack-local list (count +4, cap +8, data +0xc)
	mstRndOut   = 0x2002af2a // monster 0 + 0x2ba, the eight target words
	mstRndDecoy = 0x5a000000 // prior contents of the target words, so "not written" shows
)

// mstRndDropIn are the candidate builder's list inputs (lst.n, lst.cap); here the list is
// built by the function itself, so they are not inputs.
var mstRndDropIn = [2]int{280, 281}

func mstRunRnd(w *mstWorld, c *mstCallees) bool {
	mon, spell := w.u32(mstStackBase+4), w.u32(mstStackBase+8)
	data := w.alloc(0xa0) // 0x5e1ce0; never null on the emulator heap
	w.put(mstRndList+4, 4, 0)
	w.put(mstRndList+8, 4, 5)
	w.put(mstRndList+0xc, 4, uint64(data))
	w.cand(c, mon, spell, mstRndList)
	n := w.u32(mstRndList + 4)
	if n == 0 {
		return false
	}
	pick := int32(w.r.R(int(n)))
	src := w.u32(mstRndList + 0xc) // the builder may have grown the list
	if pick < int32(n) {
		src += uint32(pick) << 5
	}
	for k := uint32(0); k < 8; k++ {
		w.put(mon+0x2ba+4*k, 4, uint64(w.u32(src+4*k)))
	}
	return true
}

func mstRndSpec() (mstSpec, []string, []string) {
	s, in, out := mstSpec_cand, mstNames_cand_in(), mstNames_cand_out()
	var f []mstFld
	var nin, nout []string
	remap := func(i int32) int32 {
		r := i
		for _, d := range mstRndDropIn {
			if int32(d) < i {
				r--
			}
		}
		return r
	}
	for i, x := range s.in {
		if i == mstRndDropIn[0] || i == mstRndDropIn[1] {
			continue
		}
		if x.idx >= 0 {
			x.idx = remap(x.idx)
		}
		f = append(f, x)
		nin = append(nin, in[i])
	}
	s.in = f
	var o []mstFld
	for i, x := range s.out {
		if strings.HasPrefix(out[i], "lst.") || strings.HasPrefix(out[i], "lbuf") {
			continue
		}
		if x.idx >= 0 {
			x.idx = remap(x.idx)
		}
		o = append(o, x)
		nout = append(nout, out[i])
	}
	s.set = append([]mstSet(nil), s.set...)
	for k := uint32(0); k < 8; k++ {
		s.set = append(s.set, mstSet{mstRndOut + 4*k, 4, mstRndDecoy + k})
		o = append(o, mstFld{addr: mstRndOut + 4*k, size: 4, idx: -1})
		nout = append(nout, "tgt"+strconv.Itoa(int(k)))
	}
	s.out = append(o, mstFld{addr: mstRet, size: 1, idx: -1})
	return s, nin, append(nout, "ret")
}

var mstSpec_rnd, mstNames_rnd_in, mstNames_rnd_out = mstRndSpec()

// monsterSpellTargetRandomNode is 0x5326f0 with the real callees.
func monsterSpellTargetRandomNode(in []int64, r Rand) []int64 {
	return mstNode(&mstSpec_rnd, &mstCandReal, mstRunRnd, len(spellAreaInputs)+mstCandExtra)(in, r)
}

func mstEmuRnd(in []int64, r Rand) []int64 {
	return mstNode(&mstSpec_rnd, &mstCandEmu, mstRunRnd, len(spellAreaInputs)+mstCandExtra)(in, r)
}

func init() {
	Nodes["005326f0"] = Node{Addr: "005326f0", In: mstNames_rnd_in, Out: mstNames_rnd_out,
		F: monsterSpellTargetRandomNode, Emu: mstEmuRnd}
}
