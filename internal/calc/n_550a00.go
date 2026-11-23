package calc

// monster_control_duration 0x550a00(effect): extend how long a monster stays under
// control. The effect record's target is always a monster (target kind +0x90 = 3, else
// srAssertFail 0x550a1f; id +0x98 = -1 asserts at 0x550a3e: panic here).
//
//	flag78 != 0: only a chance roll. mean = (template +0xd2 + +0xd5) >> 1; R(100) < mean
//	             queues one event (0x528830(0x16), count at 0x68c4c0 is empty at entry);
//	             nothing else changes.
//	else:  resisted = low byte of 0x5520d0(target, realm 4, strength, status 0)
//	       resisted: with the notice flag 0x68510c set the resist message is shown, no state changes
//	       not resisted: v = 7*power; v += uint32(v*percentBonus)/100 (0x517f90);
//	                     v = 0x552250(v, target monster, realm 4, strength); n = low byte of v (i8)
//	       n > monster control +0x2de (i8): from 0 also 0x4acd80(visual, 0x26, 1); control = n;
//	                     notice flag clear: effect +0x125 = 1; then the control message (0x576030).
//
// Emu replays the address stubs: 0x5520d0's return is input "resistRet", its arguments and
// call count are outputs. F runs the real statusResistRollNode and feeds its result forward.
// 0x552250 runs natively in the case and is the same real function in Emu and F.
// 0x4acd80 and 0x576030 return values are never read; their arguments and counts are outputs.
//
// Fixture caps (panic past them): template index 0..2, monster id != 0xffffffff.
// Fixed by the case, not read by the rule: visual object address, event queue empty,
// message window state silent, monster +4 name override 0, template +0x187 != 0x18d.
const (
	mcdEffect = 0x20000000 // case object "effect"
	mcdVisual = 0x2001adb0 // case object "visual" (monster +0xc)
)

var mcdIn = []string{"flag78", "monsterID", "strength", "power", "percentBonus", "noticeByteBefore", "noticeFlag",
	"controlBefore", "templateIndex", "templateControlA", "templateControlB", "templateMentalResistance",
	"templateLevel", "monsterMentalModifier", "resistRet", "monsterPrimaryState", "monsterDifficultyClass", "difficulty"}

func init() {
	Nodes["00550a00"] = Node{Addr: "00550a00", In: mcdIn,
		Out: []string{"controlAfter", "noticeByteAfter", "eventCount", "resistCalls", "resistArg1", "resistArg2", "resistArg3", "resistArg4",
			"visualCalls", "visualArg1", "visualArg2", "visualArg3", "messageCalls", "messageArg1", "messageArg3", "messageArg4", "messageArg5"},
		F:   monsterControlDurationNode,
		Emu: func(in []int64, r Rand) []int64 { return mcdRun(in, r, false) }}
}

func monsterControlDurationNode(in []int64, r Rand) []int64 { return mcdRun(in, r, true) }

func mcdRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(mcdIn) {
		panic("monster control duration: wrong input count")
	}
	for i, v := range in {
		lo, hi := int64(0), int64(0xffffffff)
		switch i {
		case 0, 5, 6, 9, 10, 11, 12, 16:
			hi = 255
		case 7, 13:
			lo, hi = -128, 127
		}
		if v < lo || v > hi {
			panic("monster control duration: input out of field range")
		}
	}
	if in[1] == 0xffffffff {
		panic("monster control duration: monster id -1 (assert 0x550a3e)")
	}
	if in[8] > 2 {
		panic("monster control duration: template index beyond the case lists")
	}
	out := make([]int64, 17)
	out[0], out[1] = in[7], in[5]
	if in[0] != 0 {
		if int64(r.R(100)) < (in[9]+in[10])>>1 {
			out[2] = 1
		}
		return out
	}
	resisted := in[14]&0xff != 0
	if real {
		sr := make([]int64, 34)
		sr[0], sr[2], sr[3], sr[4], sr[6] = 3, in[1], 4, in[2], in[17]
		sr[17], sr[22], sr[24], sr[29] = in[8], in[11], in[12], in[13]
		sr[31], sr[32] = in[15], in[16]
		resisted = statusResistRollNode(sr, r)[0] != 0
	}
	out[3], out[4], out[5], out[6], out[7] = 1, mcdEffect+0x90, 4, in[2], 0
	if resisted {
		return out
	}
	v := uint32(in[3]) * 7
	v += uint32(v*uint32(in[4])) / 100 // 0x517f90
	rq := make([]int64, 27)
	rq[0], rq[3], rq[4], rq[5] = int64(v), in[2], 3, 4
	rq[16], rq[18], rq[23], rq[25], rq[26] = in[11], in[12], in[13], in[1], in[8]
	n := int64(int8(resistanceQuantityReductionNode(rq, r)[0]))
	if n <= in[7] {
		return out
	}
	if in[7] == 0 {
		out[8], out[9], out[10], out[11] = 1, mcdVisual, 0x26, 1
	}
	out[0] = n
	if in[6] == 0 {
		out[1] = 1
	}
	out[12], out[13], out[14], out[15], out[16] = 1, 1, 0, 0xffffffff, 0x47
	return out
}
