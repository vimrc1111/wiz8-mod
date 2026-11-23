package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	Nodes["00536570"] = Node{Addr: "00536570", In: append(append([]string{}, cgInNames...), "a_w", "a_m", "a_p3"),
		Out: append(append([]string{}, cgOutNames...), "ret"), F: characterTargetFallbackNode}
}

func characterTargetFallbackNode(in []int64, r Rand) []int64 {
	return cgRun(in, 3, 1, func(w *cgW, a []int64) int64 { return w.target(a[0], a[1], a[2]) & 0xff })
}

// target is 0x536570(x, m, p3): store the monster target for source m.
func (w *cgW) target(x, m, p3 int64) int64 {
	S, C, D := w.S(x), w.C(x), w.D(x)
	if w.u8(S) == 0 || w.i32(C+0xb11) == 0 || w.u32(C+0xb01) >= 0x12 {
		return 0
	}
	if m == 6 {
		m = w.resolve(x)
	}
	k := w.kind(x, m)
	a, b, c, d := w.act(x, k)
	if c == 0 {
		panic("cg: 0x536570 assert")
	}
	cls := w.actionClass(a, b, d)
	before := w.i32(c)
	done := false
	if w.f94() {
		if m == 0 {
			w.p32(D+0xac, w.u32(D+0xac)+1)
		}
		auto := a == 0 || a == 1
		if (auto || a == 7 || a == 8) && cls == 2 {
			u6 := int64(-1)
			if id := w.i32(c + 8); id != -1 {
				if e := w.find(id, false); e != -1 {
					u6 = w.i32(w.entry(e) + 4)
				}
			}
			r := w.st(cgS53c720, x, u6, k)
			if r == -1 && auto {
				r = w.autoFire(x, u6, k)
			}
			if r != -1 {
				w.store(x, cgRec{3, -1, r, -1}, m)
				// 0x52fe80(w,0) (p3 low byte) is presentation only.
				done = true
				if m == 0 {
					if w.u8(0x68510c) != 0 {
						w.notify(x)
					}
					a2, b2, c2, d2 := w.act(x, w.kind(x, 1))
					_ = b2
					s := w.st(cgS536a20, a2, b2, d2)
					if !(w.kindOK(c2, s) && w.reach(x, 2, 1)) {
						w.store(x, w.rec(c), 1)
					}
				}
			}
		}
	}
	if !done {
		w.store(x, cgRec{0, -1, -1, -1}, m)
	}
	if w.f94() && k == 1 && w.i32(c) != before {
		w.refresh(x)
	}
	if done {
		return 1
	}
	return 0
}

// actionClass maps an action (kind a, spell b, extra d) to the target class
// kindOK tests: attacks 2, defend 4, spell via spellClass, item spell via its
// item flag (the class switch of 0x536570 and 0x536400).
func (w *cgW) actionClass(a, b, d int64) int64 {
	switch a {
	case 0, 1:
		return 2
	case 2:
		return 4
	case 5:
		return 1
	case 7:
		return w.spellClass(b, 0, 6)
	case 8:
		if p := w.u32(d + 4); p != 0 && w.i32(p) != -1 {
			it := w.u32(0x68516c) + w.i32(p)*0x10d
			if sp := w.u8(it + 0x63); sp != 0 {
				kind := int64(0)
				if t := w.u8(it + 0x3e); t == 0x10 || t == 0x15 || t == 0x16 {
					kind = 1
				}
				return w.spellClass(sp, kind, 6)
			}
		}
	}
	return 0
}

// spellClass is 0x5010f0(spell, itemflag, ctx) with 0x4ac9d0.
func (w *cgW) spellClass(spell, itemflag, ctx int64) int64 {
	if spell == 0 {
		return 0
	}
	kind := w.u32(w.u32(0x65be1c) + spell*0x1bf + 0x137)
	if kind == 1 && itemflag != 0 {
		kind = 0
	}
	if kind > 10 {
		panic("cg: 0x5010f0 spell kind above 10")
	}
	if itemSpellTargetsErr != nil {
		panic(itemSpellTargetsErr)
	}
	switch t := itemSpellTargets[kind]; t {
	case -1:
		if spell == 0x58 {
			return 7
		}
		return 1
	case -2:
		if w.uip() != 0 && ctx != 3 && ctx != 4 {
			return 0
		}
		return 6
	default:
		return t
	}
}

// reach is 0x519180(x, a2, ctx): whether the stored target of source ctx can be reached.
func (w *cgW) reach(x, a2, ctx int64) bool {
	rec := w.rptr(x, ctx)
	kind := w.kind(x, ctx)
	switch w.i32(rec) {
	case 3:
		e := w.find(w.i32(rec+8), false)
		if e == -1 {
			return false
		}
		return w.st(cgS5194e0, x, a2, w.entry(e), kind, 0)&0xff != 0
	case 1:
		s := w.i32(rec + 4)
		if int64(int8(x)) == s {
			return true
		}
		r := w.st(cgS5199f0, x, a2, kind) // 0x5199f0: reach of the action (-1 none, 0 out of range)
		if r == -1 {
			return false
		}
		return r != 0 || w.st(cgS51b000, x, s)&0xff == 0 // 0x51b000: formation penalty
	case 6:
		return w.coordReach(x, kind, rec)
	case 4:
		return false // 0x5100b0 finds nothing in the empty group table
	}
	return true
}

// coordConsts are the scalars 0x519180 uses for a coordinate target, read from
// the player's own Wiz8.exe at load time (not embedded): the far range 0x5ec35c,
// the range step 0x5ebc40, the two height offsets 0x603ac8 and 0x5ebc64, the
// slack 0x5ebb38, and the range-step table behind the jump at 0x5194c0
// (index F+1 for spell field F in -1..3).
var coordConsts, coordConstsErr = loadCoordConsts()

type coordConst struct {
	far, step, dropCamera, dropTarget, slack float32
	steps                                    [5]int64
}

func loadCoordConsts() (c coordConst, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return c, fmt.Errorf("coordinate target: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	at := func(va uint32) []byte {
		if err != nil {
			return make([]byte, 4)
		}
		var o int
		if o, err = peOffset(b, va); err == nil && o+4 > len(b) {
			err = fmt.Errorf("coordinate target: truncated at %#x", va)
		}
		if err != nil {
			return make([]byte, 4)
		}
		return b[o:]
	}
	f := func(va uint32) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(at(va))) }
	c.far, c.step, c.dropCamera, c.dropTarget, c.slack = f(0x5ec35c), f(0x5ebc40), f(0x603ac8), f(0x5ebc64), f(0x5ebb38)
	for i := range c.steps {
		// Jump targets 0x51933d..0x519356 load 0, 2, 4, 25, 50 into the step count.
		switch d := binary.LittleEndian.Uint32(at(0x5194c0 + 4*uint32(i))); d {
		case 0x51933d:
			c.steps[i] = 0
		case 0x519341:
			c.steps[i] = 2
		case 0x519348:
			c.steps[i] = 4
		case 0x51934f:
			c.steps[i] = 0x19
		case 0x519356:
			c.steps[i] = 0x32
		default:
			if err == nil {
				err = fmt.Errorf("coordinate target: unknown range destination %#x", d)
			}
		}
	}
	return c, err
}

func (w *cgW) f32(a int64) float32 { return math.Float32frombits(uint32(w.u32(a))) }

// coordReach is the record kind 6 branch of 0x519180: the target point P (floats
// at rec+0x10) is reachable when the camera A (0x421070, copied from the dwords
// at *0x65a0f8+0x8c) is within the spell's range of it, and for the ranged
// spell classes 6 and 8 when the map-ray query 0x434f20 (address-stubbed: its
// return cell decides, its internal block rolls are not run) does not block.
// The arithmetic is x87 extended (64-bit mantissa, nearest-even) with float32
// stores where the original stores a dword; inputs must be finite.
func (w *cgW) coordReach(x, kind, rec int64) bool {
	if coordConstsErr != nil {
		panic(coordConstsErr)
	}
	ext := func(v float32) *big.Float { return new(big.Float).SetPrec(64).SetFloat64(float64(v)) }
	op := func(f func(z, a, b *big.Float) *big.Float, a, b *big.Float) *big.Float {
		return f(new(big.Float).SetPrec(64), a, b)
	}
	narrow := func(v *big.Float) *big.Float { f, _ := v.Float32(); return ext(f) } // fstp to a dword
	px, py, pz := ext(w.f32(rec+0x10)), ext(w.f32(rec+0x14)), ext(w.f32(rec+0x18))
	g := w.u32(0x65a0f8) + 0x8c
	ax, ay, az := ext(w.f32(g)), ext(w.f32(g+4)), ext(w.f32(g+8))
	// 0x53a8d0: the spell behind the stored action.
	a, b, _, d := w.act(x, kind)
	spell := int64(0)
	switch a {
	case 2:
		spell = 0x77
	case 7:
		spell = b
	case 8:
		spell = w.itemSpell(w.u32(d + 4))
	}
	if spell == 0 {
		panic("cg: 0x519180 assert")
	}
	base := w.u32(0x65be1c) + spell*0x1bf
	var reach *big.Float
	ray := false
	qy := py
	switch w.u32(base + 0x137) { // 0x4ac9d0 with item flag 0
	case 5:
		reach = ext(coordConsts.far)
	case 6, 8:
		f := (w.u32(base+0x12f) + 1) & 0xffffffff // F+1 as an unsigned index
		if f > 4 {
			panic("cg: 0x519180 assert")
		}
		reach = op((*big.Float).Mul, new(big.Float).SetPrec(64).SetInt64(coordConsts.steps[f]), ext(coordConsts.step))
		ray = true
		ay = narrow(op((*big.Float).Sub, ay, ext(coordConsts.dropCamera)))
		qy = narrow(op((*big.Float).Sub, py, ext(coordConsts.dropTarget)))
	default:
		panic("cg: 0x519180 assert")
	}
	dx, dy, dz := op((*big.Float).Sub, px, ax), op((*big.Float).Sub, qy, ay), op((*big.Float).Sub, pz, az)
	sum := op((*big.Float).Add, op((*big.Float).Mul, dz, dz), op((*big.Float).Mul, dy, dy))
	sum = op((*big.Float).Add, sum, op((*big.Float).Mul, dx, dx))
	dist := new(big.Float).SetPrec(64).Sqrt(sum)
	v := ext(w.f32(w.u32(0x659c0c) + 0x84))
	if op((*big.Float).Add, reach, ext(coordConsts.slack)).Cmp(op((*big.Float).Sub, dist, v)) < 0 {
		return false
	}
	if !ray {
		return true
	}
	// 0x434f20(from camera, to P, 1, -3, -3, 1, 0): zero in the low word means no block.
	return w.st(cgS434f20, 0, 0, 1, -3, -3, 1, 0)&0xffff == 0
}
