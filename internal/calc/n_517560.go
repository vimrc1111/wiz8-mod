package calc

import (
	"fmt"
	"math"
	"math/big"
)

// 0x517560 exploration_detection_range(this = target holder): the party member with the best
// search chance in the eight character slots is found, and that chance (percent) is turned
// into a distance; if the party is closer than that to the holder's world position the node
// returns the finder's slot, otherwise -1.
//
// Per slot k (flag byte k*0x106 of *0x685178 set, C = *0x685174 + 0x1862*k): C+0xb11 != 0 and
// C+0xb01 < 13. Ability 0xc is 0x547940(C, 0xc), potionFeature in n_548e60.go. Without the
// ability and without debug byte 0x6874ff the slot is skipped. Otherwise v = C+0x3dd >> 1, >> 1
// again when **0x652dac has bit 0x100; v == 0 skips the bonus unless debug is on and that bit is
// clear. Bonus by u = C+0x161 (u32 wrap, the exe's multiply-shift divisions are done with the
// same constants): u <= 50 +u/5; <= 80 +10+(u-50)/3; <= 90 +u-60; < 100 +2u-150; 100 +50;
// > 100 +50+(10u-1000)/25. v is capped at 100 and replaces the best (strictly greater, u32).
// Byte 0x687499 (cheat) then forces the slot to win with 100% when nothing won yet or R(2) != 0
// (0x40efa0), clearing the "train" flag.
//
// range = f32(f32 best * 10000 * 0.01) (x87 64-bit significand; constants exe 0x61a364,
// 0x5ed490). The holder's position comes from 0x517080: node A (packed floats at +0x15 with
// y + 100, or, when it has a child, the child's box position via 0x49fba0), node B (0x48d050 ->
// getLocation, FLT_MAX when it has no node), model M (bounding-box centre via 0x44d5c0 and
// 0x4a1710). The party position is the global vector read by 0x421070. The distance is the
// x87 square root of dz^2+dy^2+dx^2 (64-bit significand); dist < range (NaN counts as less)
// returns the slot and, when the slot is a real winner, trains skill 0xf by 5 on it
// (0x553f10, address-stubbed: its eax is overwritten, its writes are not read back).
//
// Fixture contract: pointers are 0 or the case's object address (any other value panics);
// the model's assert fields are fixed (type byte 2, mesh loaded: flag 4). 4a1710's animated-model
// path (model byte 5 != 0 on a cache miss) and its mesh loading are not modelled and panic.
const (
	edrArena  = 0x20000000
	edrAObj   = edrArena + 0x17540
	edrBObj   = edrArena + 0x17580
	edrB2Obj  = edrArena + 0x176c0
	edrXObj   = edrArena + 0x176d0
	edrMObj   = edrArena + 0x176e0
	edrKObj   = edrArena + 0x17810
	edrCFlags = edrArena + 0x17960
	edrSlotN  = 7
	edrParty  = 3 + 8*edrSlotN
	edrPA     = edrParty + 3
	edrPB     = edrPA + 1
	edrPM     = edrPB + 1
	edrPb     = edrPM + 1 // child pointer of node A, then ax ay az
	edrC1     = edrPb + 4 // c1x c1y c1z c16c c178
	edrPX     = edrC1 + 5
	edrGl     = edrPX + 1
	edrPK     = edrGl + 3
	edrFrame  = edrPK + 1
	edrM5     = edrFrame + 1
	edrPC     = edrM5 + 1
	edrCHit   = edrPC + 1
	edrCv     = edrCHit + 1 // cv1 xyz, cv2 xyz
	edrMesh   = edrCv + 6   // min xyz, max xyz
	edrIn     = edrMesh + 6
)

func init() {
	n := []string{"dbg", "rngOn", "sdl"}
	for k := 0; k < 8; k++ {
		for _, f := range []string{"sel", "cls", "race", "b01_", "b11_", "s3dd", "s161_"} {
			n = append(n, fmt.Sprint(f, k))
		}
	}
	for _, f := range []string{"px", "py", "pz", "pA", "pB", "pM", "pb", "ax", "ay", "az", "c1x", "c1y", "c1z", "c16c", "c178", "pX",
		"glx", "gly", "glz", "pK", "frame", "m5", "pC", "chit", "cv1x", "cv1y", "cv1z", "cv2x", "cv2y", "cv2z",
		"mnx", "mny", "mnz", "mxx", "mxy", "mxz"} {
		n = append(n, f)
	}
	Nodes["00517560"] = Node{Addr: "00517560", In: n, Out: []string{"ret", "calls", "arg1", "arg2", "arg3", "arg4"}, F: explorationDetectionRangeNode}
}

func edrExt(f float32) *big.Float { // nil is NaN
	if f != f {
		return nil
	}
	return new(big.Float).SetPrec(64).SetFloat64(float64(f))
}

func edrNew() *big.Float { return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven) }

func edrAdd(a, b *big.Float) *big.Float {
	if a == nil || b == nil || (a.IsInf() && b.IsInf() && a.Signbit() != b.Signbit()) {
		return nil
	}
	return edrNew().Add(a, b)
}

func edrSub(a, b *big.Float) *big.Float {
	if a == nil || b == nil || (a.IsInf() && b.IsInf() && a.Signbit() == b.Signbit()) {
		return nil
	}
	return edrNew().Sub(a, b)
}

func edrMul(a, b *big.Float) *big.Float {
	if a == nil || b == nil || (a.IsInf() && b.Sign() == 0) || (b.IsInf() && a.Sign() == 0) {
		return nil
	}
	return edrNew().Mul(a, b)
}

func edrStore(a *big.Float) float32 { // fstp dword: one rounding from the 64-bit significand
	if a == nil {
		return float32(math.NaN())
	}
	f, _ := a.Float32()
	return f
}

func edrBits(v int64) float32 { return math.Float32frombits(uint32(v)) }

func edrPtr(v, obj int64) bool {
	if v != 0 && v != obj {
		panic("exploration detection range: pointer is neither null nor the fixture object")
	}
	return v != 0
}

// edrHalf is 0x517080's centre of two box corners: x is stored from the x87 sum, y and z are
// rounded to float32 after the add and halved again.
func edrHalf(a, b [3]float32) [3]float32 {
	half := edrExt(0.5)
	var t [3]float32
	for i := range t {
		s := edrAdd(edrExt(a[i]), edrExt(b[i]))
		if i > 0 {
			s = edrExt(edrStore(s))
		}
		t[i] = edrStore(edrMul(s, half))
	}
	return t
}

func edrTarget(in []int64) [3]float32 {
	var t [3]float32
	f := func(i int) float32 { return edrBits(in[i]) }
	switch {
	case edrPtr(in[edrPA], edrAObj):
		if edrPtr(in[edrPb], edrBObj) { // 0x49fba0: box position, y raised by 0.66 of the box height
			d := edrSub(edrExt(f(edrC1+4)), edrExt(f(edrC1+3)))
			y := edrAdd(edrMul(d, edrExt(0.66000003)), edrExt(f(edrC1+1)))
			return [3]float32{f(edrC1), edrStore(y), f(edrC1 + 2)}
		}
		return [3]float32{f(edrPb + 1), edrStore(edrAdd(edrExt(f(edrPb+2)), edrExt(100))), f(edrPb + 3)}
	case edrPtr(in[edrPB], edrB2Obj):
		if edrPtr(in[edrPX], edrXObj) {
			return [3]float32{f(edrGl), f(edrGl + 1), f(edrGl + 2)}
		}
		m := math.Float32frombits(0x7f7fffff)
		return [3]float32{m, m, m}
	case edrPtr(in[edrPM], edrMObj):
		if !edrPtr(in[edrPK], edrKObj) {
			return t
		}
		hit := edrPtr(in[edrPC], edrCFlags) && in[edrCHit] != 0
		if !hit && in[edrM5] != 0 {
			panic("exploration detection range: 0x4a1710 animated-model path is not modelled")
		}
		var lo, hi [3]float32
		for i := range lo {
			if hit { // cached box of this frame
				lo[i], hi[i] = f(edrCv+i), f(edrCv+3+i)
			} else { // loaded static mesh box (0x4b6640)
				lo[i], hi[i] = f(edrMesh+i), f(edrMesh+3+i)
			}
		}
		return edrHalf(lo, hi)
	}
	return t
}

// edrBonus is the u32 wrap-around bonus ladder at exe 0x517600..0x517666.
func edrBonus(v, u uint32) uint32 {
	switch {
	case u <= 0x32:
		return v + uint32(uint64(u)*0xcccccccd>>34)
	case u <= 0x50:
		return v + uint32(uint64(u-0x32)*0xaaaaaaab>>33) + 10
	case u <= 0x5a:
		return v + u - 0x3c
	case u < 0x64:
		return v + u*2 - 0x96
	case u == 0x64:
		return v + 0x32
	}
	return v + uint32(uint64(u*10-0x3e8)*0x51eb851f>>35) + 0x32
}

func explorationDetectionRangeNode(in []int64, r Rand) []int64 {
	if len(in) != edrIn {
		panic("exploration detection range: wrong input count")
	}
	for i, v := range in {
		lo, hi := int64(0), int64(math.MaxUint32)
		switch {
		case i < 2 || i == edrFrame || i == edrM5 || i == edrCHit || i >= 3 && i < edrParty && (i-3)%edrSlotN == 0:
			hi = 255 // byte fields
		case i >= 3 && i < edrParty && (i-3)%edrSlotN < 3:
			lo, hi = math.MinInt32, math.MaxInt32 // class, race: signed
		}
		if v < lo || v > hi {
			panic("exploration detection range: input outside its field width")
		}
	}
	dbg, rngOn, sdl := in[0] != 0, in[1] != 0, in[2]&0x100 != 0
	best, bp, train := uint32(0), int64(-1), false
	for k := 0; k < 8; k++ {
		s := in[3+k*edrSlotN:]
		if s[0] != 0 && s[4] != 0 && s[3] < 13 {
			if potionFeature(s[1], s[2], 0xc) || dbg {
				v := uint32(s[5]) >> 1
				if sdl {
					v >>= 1
				}
				if v != 0 || dbg && !sdl {
					v = edrBonus(v, uint32(s[6]))
				}
				if v >= 100 {
					v = 100
				}
				if v > best {
					best, bp, train = v, int64(k), true
				}
			}
			if rngOn && (bp == -1 || r.R(2) != 0) {
				bp, best, train = int64(k), 100, false
			}
		}
	}
	rng := edrStore(edrMul(edrMul(edrExt(float32(best)), edrExt(10000)), edrExt(0.0099999998)))
	t := edrTarget(in)
	var sq [3]*big.Float
	for i := range sq {
		d := edrSub(edrExt(edrBits(in[edrParty+i])), edrExt(t[i]))
		sq[i] = edrMul(d, d)
	}
	dist := edrAdd(edrAdd(sq[2], sq[1]), sq[0])
	if dist != nil {
		dist.Sqrt(dist)
	}
	if dist != nil && dist.Cmp(edrExt(rng)) >= 0 {
		return []int64{-1, 0, 0, 0, 0, 0}
	}
	if !train {
		return []int64{bp, 0, 0, 0, 0, 0}
	}
	return []int64{bp, 1, (edrArena + 0x1862*bp) & 0xffffffff, 0xf, 5, 0}
}
