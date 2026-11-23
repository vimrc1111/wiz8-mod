package calc

import (
	"fmt"
	"math"
	"math/big"
)

// Input/output layout of the 0x4f0af0 node (indices into in/out).
const (
	cmlUIFields     = 10 // fields of the movement UI object at +0x24,0x28,0x2c,0x30,0x4c,0x50,0x54,0x7c,0x80,0x84
	cmlInSlots      = 21 // 8 x (active, excluded, strength, bonus, duration)
	cmlInHPInit     = cmlInSlots + 40
	cmlInActions    = cmlInHPInit + 16
	cmlInMonsters   = cmlInActions + 16 // listPointer, count, 8 x (active, action, done)
	cmlInCount      = cmlInMonsters + 2 + 24
	cmlOutUIFields  = 6
	cmlOutCtxFlags  = cmlOutUIFields + cmlUIFields
	cmlOutSlotAct   = cmlOutCtxFlags + 1
	cmlOutMonsterAc = cmlOutSlotAct + 8
	cmlOutCount     = cmlOutMonsterAc + 8
)

func init() {
	in := []string{"newPartyActionMode", "partyActionMode", "partyActionProgress", "partyActionPhase", "partyThreshold",
		"gameState", "movementUiPointer", "movementUiFlags"}
	out := []string{"partyActionMode", "partyActionProgress", "partyActionPhase", "partyThreshold", "movementDistanceLimit", "movementUiFlags"}
	uiNames := []string{"uiReset24", "uiReset28", "uiMovementDistance", "uiReset30", "uiReset4c", "uiReset50", "uiReset54", "uiReset7c", "uiReset80", "uiReset84"}
	in = append(in, uiNames...)
	out = append(out, uiNames...)
	in = append(in, "movementDistanceLimit", "effectsContextPointer", "effectsContextFlags")
	out = append(out, "effectsContextFlags")
	for i := 0; i < 8; i++ {
		in = append(in, fmt.Sprintf("slot%dActive", i), fmt.Sprintf("slot%dMovementExcludedCondition", i),
			fmt.Sprintf("slot%dMovementBuffStrength", i), fmt.Sprintf("slot%dMovementBuffBonusPercent", i), fmt.Sprintf("slot%dMovementBuffDuration", i))
	}
	for i := 0; i < 8; i++ {
		in = append(in, fmt.Sprintf("slot%dCurrentHP", i), fmt.Sprintf("slot%dInitiative", i))
	}
	for i := 0; i < 8; i++ {
		in = append(in, fmt.Sprintf("slot%dAction", i), fmt.Sprintf("slot%dDone", i))
		out = append(out, fmt.Sprintf("slot%dAction", i))
	}
	in = append(in, "monsterListPointer", "monsterCount")
	for i := 0; i < 8; i++ {
		in = append(in, fmt.Sprintf("monster%dActive", i), fmt.Sprintf("monster%dAction", i), fmt.Sprintf("monster%dDone", i))
		out = append(out, fmt.Sprintf("monster%dAction", i))
	}
	Nodes["004f0af0"] = Node{Addr: "004f0af0", In: in, Out: out, F: combatMovementDistanceLimitNode}
}

// combatMovementDistanceLimitNode: entering a new party action mode (non-zero u32,
// exe 0x4f0af0..0x4f0afa asserts otherwise) does nothing when it equals the current
// mode. Otherwise it stores the mode, clears the progress field, derives the
// movement distance limit float, resets the movement UI object, re-rolls the party
// phase (0x4efe70), rounds actions (0x4f0c80) and rounds the threshold.
// Inputs outside the original's legal domain (srAssertFail: zero mode, threshold
// that ends outside 1..100) and a nonempty-output 0x4f0010 query with no eligible
// member (original divide fault) panic like the callee nodes.
func combatMovementDistanceLimitNode(in []int64, r Rand) []int64 {
	if len(in) != cmlInCount {
		panic("combatMovementDistanceLimit: wrong input count")
	}
	for _, v := range in {
		if v < -2147483648 || v > 4294967295 {
			panic("combatMovementDistanceLimit: input outside 32 bits")
		}
	}
	n := uint32(in[0])
	if n == 0 {
		panic("combatMovementDistanceLimit: zero party action mode (srAssertFail)")
	}
	out := make([]int64, cmlOutCount)
	copy(out, in[1:5]) // mode, progress, phase, threshold
	out[4] = in[18]    // previous distance limit
	out[5] = in[7]
	copy(out[cmlOutUIFields:], in[8:8+cmlUIFields])
	out[cmlOutCtxFlags] = in[20]
	for i := 0; i < 8; i++ {
		out[cmlOutSlotAct+i] = in[cmlInActions+2*i]
		out[cmlOutMonsterAc+i] = in[cmlInMonsters+2+3*i+1]
	}
	// exe 0x4f0b1a: already in this mode, nothing happens.
	if uint32(in[1]) == n {
		return out
	}
	out[0], out[1] = int64(n), 0 // exe 0x4f0b26, 0x4f0b31

	// exe 0x4f0b47..0x4f0b58: base multiplier 1.0 (0x3f800000), 1.5 (0x3fc00000) in mode 2.
	mult := new(big.Float).SetPrec(64).SetFloat64(1.0)
	if n == 2 {
		mult.SetFloat64(1.5)
	}
	// 0x4f0010 with a stack output slot: on success the average is added as
	// fild(u32) * f32 at exe 0x5ed490 (0.01f) + base, in 80-bit x87 precision.
	q := []int64{1, 0}
	for s := 0; s < 8; s++ {
		q = append(q, in[cmlInSlots+5*s:cmlInSlots+5*s+5]...)
	}
	// Active flag at index 0 of each record is shared with the phase node below.
	if res := partyMovementEffectAverageNode(q, r); res[0]&0xff != 0 {
		c := new(big.Float).SetPrec(64).SetFloat64(float64(float32(0.009999999776482582))) // exe 0x5ed490
		t := new(big.Float).SetPrec(64).SetUint64(uint64(uint32(res[1])))
		t.Mul(t, c)
		mult.Add(t, mult)
	}
	// exe 0x4f0b95..0x4f0ba1: times f32 at exe 0x5ec0a8 (10000.0f), stored as float32 at 0x683fcf.
	mult.Mul(mult, new(big.Float).SetPrec(64).SetFloat64(10000))
	f, _ := mult.Float32()
	bits := int64(math.Float32bits(f))
	out[4] = bits

	// 0x41eee0(limit bits, 0, mode==2): resets the movement UI object when present.
	if in[6] != 0 {
		flags := uint32(in[7]) | 0x20
		flags &^= 0x80
		if n == 2 {
			flags |= 0x100
		} else {
			flags &^= 0x100
		}
		out[5] = int64(flags)
		for i := 0; i < cmlUIFields; i++ {
			out[cmlOutUIFields+i] = 0
		}
		out[cmlOutUIFields+2] = bits
	}

	// 0x4efe70 recomputes the phase from the new mode.
	hp := []int64{int64(n)}
	for s := 0; s < 8; s++ {
		hp = append(hp, in[cmlInSlots+5*s], in[cmlInHPInit+2*s], in[cmlInHPInit+2*s+1])
	}
	phase := uint32(partyActionPhaseNode(hp, r)[0])
	// exe 0x4f0bc8..0x4f0be4: add the threshold (32-bit), cap at 100 (unsigned).
	thr := uint32(in[4])
	phase += thr
	if phase > 100 {
		phase = 100
	}
	// 0x4f0c80 rounds the phase, party slots and monsters.
	rin := []int64{int64(n), int64(thr), int64(phase)}
	rin = append(rin, in[cmlInActions:cmlInActions+16]...)
	rin = append(rin, in[cmlInMonsters:cmlInMonsters+26]...)
	ro := combatActorPhaseRoundingNode(rin, r)
	out[2] = ro[0]
	copy(out[cmlOutSlotAct:], ro[1:9])
	copy(out[cmlOutMonsterAc:], ro[9:17])

	// exe 0x4f0bf9..0x4f0c3d: mode is non-zero, so the threshold is rounded like the
	// phase: +5, drop the remainder mod 10, ceil the old value to a multiple of 10
	// (32-bit wraps) and clamp with 0x5179f0 (cap 100 tested first).
	v := thr + 5
	v -= v % 10
	lower := thr
	if rem := lower % 10; rem != 0 {
		lower += 10 - rem
	}
	if v > 100 {
		v = 100
	} else if v < lower {
		v = lower
	}
	// exe 0x4f0c3f..0x4f0c49: srAssertFail unless 1..100.
	if v == 0 || v > 100 {
		panic("combatMovementDistanceLimit: threshold outside 1..100 (srAssertFail)")
	}
	out[3] = int64(v)

	// 0x562a50(0x100000): in game state 7 with a context object, set that flag.
	if in[5] == 7 && in[19] != 0 {
		out[cmlOutCtxFlags] = int64(uint32(in[20]) | 0x100000)
	}
	return out
}
