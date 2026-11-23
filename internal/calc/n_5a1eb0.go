package calc

import (
	"fmt"
	"math"
	"math/big"
)

// movement_stamina_cost 0x5a1eb0(deltaA*, deltaB*), build/wiz8.asm:563998..564200.
// First 0x41f170 (movement step source, not registered as a formula, runs
// natively in emu): when the object at *0x652dac is null it returns false and
// 0x5a1eb0 returns 0. Otherwise it copies the object's floats +0x24/+0x28 to
// *deltaA/*deltaB, adds them to the float accumulators at 0x687501/0x687505
// (x87 add, stored as float32), zeroes the two object dwords and returns true
// when either float is above 0.001 (float32 0x3a83126f at 0x5ebc84). The caller
// ignores the 0.001 result: 0x5a1ec7 tests it, so false returns 0 early.
//
// Then for each of the eight party members (party record at *0x685174, stride
// 0x1862; active-flag/fatigue record at *0x685178, stride 0x106): skipped
// unless the active byte Q+0 is non-zero, C+0xb1d (signed) > 0 and C+0xb01
// (unsigned) < 15. The step cost is *deltaB, plus 0.5 * *deltaA (0x5ebc7c) when
// byte 0x6874ff or byte 0x683f94 is non-zero, multiplied by the load factor
// from C+0xbc9 (jump table 0x5a2068: 1, 1.25, 1.5, 2, 3; any other level is
// the srAssertFail path 0x5a1f84, which stops the game so calc panics), doubled
// when byte 0x683f94 is non-zero, times 1.5 (0x5ec3b8) when C+0xa09 is non-zero.
// Q+0xf6 (float32) += factor * cost. Every x87 step keeps a 64-bit significand
// and each store to memory rounds to float32. When the exact sum is above 2500.0
// (0x5ebfdc) the sum times 0.0004 (float32 0x39d1b717 at 0x5eecd0) is truncated
// (_ftol, low 32 bits) to n, 0x52af50(who, n, 0, 0) is called and Q+0xf6 becomes
// float32(Q+0xf6 - uint32(n*2500)) (32-bit imul wrap, zero-extended fild).
//
// Emu address-stubs 0x52af50 (D4, composition ruling 2026-10-09): the caller
// ignores its eax and reads nothing it writes (no later read of C+0xb1d), so the
// outputs are the call count and every call's four arguments (each member calls
// it at most once). Fixture caps (calc panics past them): the pointer at
// 0x652dac is 0 or the case object; float inputs are finite; a sum whose scaled
// truncation leaves int64 is the emulator's _ftol limit; load level above 4 on a
// reached member panics (cov-exempt 5a1f84).
const (
	moveStaminaObj     = 0x20011000 // case object holding the step floats
	moveStaminaMembers = 8
	moveStaminaInputs  = 9 + 6*moveStaminaMembers
)

func init() {
	in := []string{"stepSource", "stepSourceA", "stepSourceB", "deltaAInit", "deltaBInit", "flag6874ff", "flag683f94", "accum687501", "accum687505"}
	out := []string{"result", "deltaA", "deltaB", "stepSourceAAfter", "stepSourceBAfter", "accum687501", "accum687505"}
	for i := 0; i < moveStaminaMembers; i++ {
		for _, f := range []string{"active", "fatigueAccum", "stamina", "moveBlock", "loadLevel", "burdenState"} {
			in = append(in, fmt.Sprintf("m%d_%s", i, f))
		}
		out = append(out, fmt.Sprintf("m%d_fatigueAccum", i))
	}
	out = append(out, "n52af50")
	for a := 1; a <= 4; a++ {
		for i := 0; i < moveStaminaMembers; i++ {
			out = append(out, fmt.Sprintf("a52af50_%d_%d", a, i))
		}
	}
	Nodes["005a1eb0"] = Node{Addr: "005a1eb0", In: in, Out: out, F: movementStaminaCostNode}
}

func movementStaminaCostNode(in []int64, r Rand) []int64 {
	if len(in) != moveStaminaInputs {
		panic(fmt.Sprintf("movement_stamina_cost: expected %d inputs, got %d", moveStaminaInputs, len(in)))
	}
	ext := func(v float64) *big.Float { return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).SetFloat64(v) }
	ld := func(bits int64) *big.Float {
		f := math.Float32frombits(uint32(bits))
		if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
			panic("movement_stamina_cost: non-finite float input")
		}
		return ext(float64(f))
	}
	add := func(a, b *big.Float) *big.Float { return ext(0).Add(a, b) }
	sub := func(a, b *big.Float) *big.Float { return ext(0).Sub(a, b) }
	mul := func(a, b *big.Float) *big.Float { return ext(0).Mul(a, b) }
	st := func(v *big.Float) int64 { f, _ := v.Float32(); return int64(math.Float32bits(f)) }

	out := make([]int64, 0, 7+moveStaminaMembers+1+4*moveStaminaMembers)
	ptr, a, b := in[0], in[1], in[2]
	acc1, acc2 := in[7], in[8]
	if ptr != 0 && ptr != moveStaminaObj {
		panic("movement_stamina_cost: step source pointer is neither null nor the case object")
	}
	var q [moveStaminaMembers]int64
	for i := range q {
		q[i] = in[9+6*i+1]
	}
	result := int64(0)
	dA, dB := in[3], in[4]
	if ptr != 0 {
		dA, dB = a, b
		acc1 = st(add(ld(acc1), ld(a))) // 0x41f1c0..0x41f1ca
		acc2 = st(add(ld(acc2), ld(b))) // 0x41f1d0..0x41f1da
		thr := ext(float64(math.Float32frombits(0x3a83126f)))
		if ld(a).Cmp(thr) > 0 || ld(b).Cmp(thr) > 0 { // 0x41f18e..0x41f1b0
			result = 1
		}
		a, b = 0, 0 // object dwords zeroed
	}
	var args [4][moveStaminaMembers]int64
	calls := int64(0)
	if result != 0 {
		half, limit, scale, burden := ext(0.5), ext(2500), ext(float64(math.Float32frombits(0x39d1b717))), ext(1.5)
		flagA, flagB := in[5]&0xff != 0, in[6]&0xff != 0
		base := ld(dB)
		cost := st(base)
		if flagA || flagB {
			cost = st(add(mul(ld(dA), half), base)) // 0x5a1f2c..0x5a1f38
		}
		loads := [5]float64{1, 1.25, 1.5, 2, 3} // jump table 0x5a2068
		for i := 0; i < moveStaminaMembers; i++ {
			m := in[9+6*i : 9+6*i+6]
			if m[0]&0xff == 0 || int32(m[2]) <= 0 || uint32(m[3]) >= 0xf {
				continue
			}
			if uint32(m[4]) > 4 {
				panic("movement_stamina_cost: load level outside 0..4 (srAssertFail)")
			}
			f := loads[uint32(m[4])]
			if flagB {
				f += f
			}
			factor := ext(f)
			if uint32(m[5]) != 0 {
				factor = mul(factor, burden) // exact: stored as float32 unchanged
			}
			sum := add(mul(factor, ld(cost)), ld(q[i])) // 0x5a1fcc..0x5a1fd4
			q[i] = st(sum)                              // fst 0x5a1fe0
			if sum.Cmp(limit) <= 0 {
				continue
			}
			t, _ := mul(sum, scale).Int(nil)
			if !t.IsInt64() {
				panic("movement_stamina_cost: _ftol result outside int64")
			}
			n := uint32(t.Int64())
			args[0][calls], args[1][calls], args[2][calls], args[3][calls] = int64(i), int64(n), 0, 0
			calls++
			q[i] = st(sub(ld(q[i]), ext(float64(n*2500)))) // lea chain *2500, 32-bit, fild qword
		}
	}
	out = append(out, result, dA, dB, a, b, acc1, acc2)
	out = append(out, q[:]...)
	out = append(out, calls)
	for k := range args {
		out = append(out, args[k][:]...)
	}
	return out
}
