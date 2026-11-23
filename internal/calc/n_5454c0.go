package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

func init() {
	Nodes["005454c0"] = Node{Addr: "005454c0",
		In:  []string{"originX", "originY", "originZ", "aimX", "aimY", "aimZ", "accuracy", "blindFlag"},
		Out: []string{"aimX", "aimY", "aimZ"}, F: missileAimDispersionNode}
}

var missileAimExe struct {
	once sync.Once
	b    []byte
	err  error
}

// missileAimConst reads the double at VA from the player's own Wiz8.exe.
func missileAimConst(va uint32) float64 {
	missileAimExe.once.Do(func() {
		path := os.Getenv("WIZ8_EXE")
		if path == "" {
			_, source, _, ok := runtime.Caller(0)
			if !ok {
				missileAimExe.err = fmt.Errorf("missile aim: set WIZ8_EXE")
				return
			}
			path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
		}
		missileAimExe.b, missileAimExe.err = os.ReadFile(path)
	})
	if missileAimExe.err != nil {
		panic(missileAimExe.err)
	}
	off, err := peOffset(missileAimExe.b, va)
	if err != nil || off+8 > len(missileAimExe.b) {
		panic("missile aim: constant address unmapped")
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(missileAimExe.b[off:]))
}

// Missile aim dispersion (0x5454c0..0x545891). Inputs: origin and aim point as
// float32 bits (ebx / edi records), accuracy (i32, [ebp+0x10]) and the blind
// flag (low byte of [ebp+0x14]). Output: the aim point written back to edi.
// x87 intermediates are 64-bit significands (control word 0x037f), modelled
// with big.Float; every float32 store rounds. sin/cos are binary64 (the
// emulator's fsin/fcos). Float32 overflow of the intermediate vectors needs
// |coordinates| beyond 2^31 and is not modelled (panics); NaN/Inf are not
// positions.
// Exe constants: 0x5ebb40 0, 0x5ebc30 1, 0x5ebf50 0.002, 0x5ec150 500,
// 0x5ec300 1/180, 0x5ee7a0 pi, 0x5ee7c0 0.01, 0x5ee7c8 1/300, 0x5ee7d0 300,
// 0x5ee7d8 0.08, 0x5ee7e0 1.75.
func missileAimDispersionNode(in []int64, r Rand) []int64 {
	if len(in) != 8 || in[6] < math.MinInt32 || in[6] > math.MaxInt32 || in[7] < 0 || in[7] > 255 {
		panic("missile aim: expected 6 float32 bit patterns, i32 accuracy, u8 flag")
	}
	n := func() *big.Float { return new(big.Float).SetPrec(64) }
	val := func(f float64) *big.Float { return n().SetFloat64(f) }
	lim := math.Ldexp(1, 31)
	ld := func(i int) *big.Float {
		if in[i] < 0 || in[i] > math.MaxUint32 {
			panic("missile aim: expected float32 bits")
		}
		f := float64(math.Float32frombits(uint32(in[i])))
		if math.IsInf(f, 0) || math.IsNaN(f) || math.Abs(f) > lim {
			panic("missile aim: expected finite position within 2^31")
		}
		return val(f)
	}
	mul := func(a, b *big.Float) *big.Float { return n().Mul(a, b) }
	add := func(a, b *big.Float) *big.Float { return n().Add(a, b) }
	sub := func(a, b *big.Float) *big.Float { return n().Sub(a, b) }
	div := func(a, b *big.Float) *big.Float { return n().Quo(a, b) }
	sqrt := func(a *big.Float) *big.Float { return n().Sqrt(a) }
	neg := func(a *big.Float) *big.Float { return n().Neg(a) }
	st32 := func(a *big.Float) *big.Float { // fstp dword, later reloaded
		f, _ := a.Float32()
		if math.IsInf(float64(f), 0) {
			panic("missile aim: float32 overflow")
		}
		return val(float64(f))
	}
	st64 := func(a *big.Float) *big.Float { f, _ := a.Float64(); return val(f) }
	sq := func(a *big.Float) *big.Float { return mul(a, a) }
	zero := val(missileAimConst(0x5ebb40))

	ox, oy, oz := ld(0), ld(1), ld(2)
	tx, ty, tz := ld(3), ld(4), ld(5)
	dx, dy, dz := st32(sub(tx, ox)), st32(sub(ty, oy)), st32(sub(tz, oz))
	// P is perpendicular to d, Q to both (0x5454db..0x545583).
	dz2 := st32(sq(dz))
	p0 := st32(mul(dy, neg(dx)))
	p1 := st32(sub(dz2, mul(dx, neg(dx))))
	p2 := st32(neg(mul(dz, dy)))
	q0 := st32(sub(mul(p1, dz), mul(p2, dy)))
	q1 := st32(sub(mul(p2, dx), mul(dz, p0)))
	q2 := st32(sub(mul(p0, dy), mul(p1, dx)))

	fr := n().SetInt64(int64(uint32(r.R(360))))
	ang := mul(fr, mul(val(missileAimConst(0x5ee7a0)), val(missileAimConst(0x5ec300))))
	f64 := func(a *big.Float) float64 { f, _ := a.Float64(); return f }
	if s := add(add(sq(q1), sq(q2)), sq(q0)); s.Cmp(zero) != 0 {
		k := div(val(math.Sin(f64(ang))), sqrt(s))
		q0, q1, q2 = st32(mul(q0, k)), st32(mul(q1, k)), st32(mul(q2, k))
	}
	if u := add(add(sq(p2), sq(p1)), sq(p0)); u.Cmp(zero) != 0 {
		c := div(val(math.Cos(f64(ang))), sqrt(u))
		p0, p1, p2 = st32(mul(p0, c)), st32(mul(p1, c)), st32(mul(p2, c))
	}
	q0, q1, q2 = st32(add(p0, q0)), st32(add(p1, q1)), st32(add(p2, q2))

	dist := sqrt(add(add(sq(dx), sq(dy)), dz2))
	if dist.Cmp(val(missileAimConst(0x5ebc30))) < 0 {
		return in[3:6] // aim point left as given
	}
	var g, h *big.Float
	if in[7] != 0 {
		h = val(1.75) // [esp+0x28] = 0x3ffc0000:0
		g = div(val(missileAimConst(0x5ec150)), dist)
		if m := val(missileAimConst(0x5ee7e0)); g.Cmp(m) > 0 {
			g = m
		}
	} else {
		h = val(float64(math.Float64frombits(0x3fc999999999999a)))
		g = val(missileAimConst(0x5ee7d8))
	}
	v18 := st64(mul(g, dist))
	t, _ := mul(dist, val(missileAimConst(0x5ebf50))).Int64() // _ftol truncates toward zero
	a := int32(int32(100-t)*int32(in[6])) / 100               // imul + 0x51eb851f magic, sar 5
	v28 := h
	switch {
	case a <= 0:
		v28 = st64(mul(h, dist))
	case a >= 300:
		v28 = v18
	default:
		w := mul(sub(val(missileAimConst(0x5ee7d0)), n().SetInt64(int64(a))), val(missileAimConst(0x5ee7c8)))
		v28 = st64(add(mul(mul(w, sub(h, g)), dist), v18))
	}
	x := int32(r.R(100)) + 1
	m := a
	if m > 95 {
		m = 95
	}
	var f *big.Float
	switch {
	case x <= m:
		f = mul(div(n().SetInt64(int64(x)), n().SetInt64(int64(a))), v18)
	case a > 0:
		f = add(mul(div(n().SetInt64(int64(x-a)), n().SetInt64(int64(x))), sub(v28, v18)), v18)
	case x+a > 0: // int32 wrap cannot occur: x <= 100, a <= 0
		f = add(mul(mul(n().SetInt64(int64(100-(x+a))), val(missileAimConst(0x5ee7c0))), sub(v28, v18)), v18)
	default:
		f = v28
	}
	if s := add(add(sq(q1), sq(q2)), sq(q0)); s.Cmp(zero) != 0 {
		k := div(f, sqrt(s))
		q0, q1, q2 = st32(mul(q0, k)), st32(mul(q1, k)), st32(mul(q2, k))
	}
	// First sum keeps its 64-bit significand until the origin is added (0x54583b..0x54585b).
	rx := st32(add(add(dx, q0), ox))
	ry := st32(add(st32(add(dy, q1)), oy))
	rz := st32(add(st32(add(dz, q2)), oz))
	bits := func(v *big.Float) int64 { f, _ := v.Float32(); return int64(math.Float32bits(f)) }
	return []int64{bits(rx), bits(ry), bits(rz)}
}
