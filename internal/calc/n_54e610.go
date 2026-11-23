package calc

// randomSpellEffectNode is 0x54e610 random_spell_effect(effect), reached only
// from the effect dispatcher at 0x54c4dd, which ignores eax.
//
// Duration (0x54e615..0x54e66b): d = effect +0xdc * +0xd0 + +0xd8 (32-bit
// wrap). Unless d == 9999 (0x54e628): d+1, then R(4) (0x40efa0): roll 0 adds 1
// when d > 1, roll 1 adds 1 when d <= 2 (unsigned); then 0x517f90 adds
// floor(uint32(d * +0xd4) / 100). The result is written to the argument slot
// and never read again, so it is an output only; with 9999 the slot keeps the
// effect pointer.
// Target type +0x90 must be 1 (character); anything else hits srAssertFail
// (0x54e677, cov-exempt), and calc panics.
// Effect choice R(100) (0x54e691): < 40 status 0x54d5c0(effect, 4, 100, 0),
// < 80 stamina 0x54f520(effect), < 90 status kind 0xf, < 95 status kind 9,
// else timed buff 0x54e3f0(effect, 6). The chosen callee's eax is returned.
//
// Address stubs (re/emu/cases/0054e610.json): D4 0x54d5c0, 0x54f520, 0x54e3f0,
// each returning a case input; outputs are their call counts and arguments.
// No Emu/F split: all three are tail calls, so nothing in this node reads
// their returns or writes; the callees' own rules are their nodes' F.
func randomSpellEffectNode(in []int64, r Rand) []int64 {
	const effect = 0x20000000 // the case's argument (object effect)
	if len(in) != 8 {
		panic("random spell effect: wrong input count")
	}
	if uint32(in[4]) != 1 {
		panic("random spell effect: target type is not character (srAssertFail at 0x54e677)")
	}
	d := uint32(in[0])*uint32(in[1]) + uint32(in[2])
	if d != 9999 {
		d++
		roll := r.R(4)
		if (roll == 0 && d > 1) || (roll == 1 && d <= 2) {
			d++
		}
		d += d * uint32(in[3]) / 100
	} else {
		d = effect
	}
	out := make([]int64, 12)
	out[1] = int64(d)
	status := func(kind int64) {
		out[0] = in[5]
		out[2], out[3], out[4], out[5], out[6] = 1, effect, kind, 100, 0
	}
	switch p := r.R(100); {
	case p < 40:
		status(4)
	case p < 80:
		out[0], out[7], out[8] = in[6], 1, effect
	case p < 90:
		status(0xf)
	case p < 95:
		status(9)
	default:
		out[0], out[9], out[10], out[11] = in[7], 1, effect, 6
	}
	return out
}

func init() {
	Nodes["0054e610"] = Node{Addr: "0054e610",
		In: []string{"durationPerPower", "power", "durationBase", "percentBonus", "targetType",
			"statusRet", "staminaRet", "buffRet"},
		Out: []string{"eax", "duration", "status.count", "status.effect", "status.kind",
			"status.chance", "status.arg4", "stamina.count", "stamina.effect",
			"buff.count", "buff.effect", "buff.kind"},
		F: randomSpellEffectNode}
}
