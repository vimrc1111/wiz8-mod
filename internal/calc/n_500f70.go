package calc

import (
	"math"
	"math/big"
)

// spellDeathFollowupEffectNode is 0x500f70 spell_death_followup_effect(res),
// called once from 0x500d5b (eax ignored) after an effect result whose spell
// is 0x4f. Exe 0x500f70..0x5010e2.
//
// Target kind res+0x90: 3 = monster (record by id res+0x98 via 0x4e5840),
// 1 = party member res+0x94; any other kind does nothing. A living target
// (monster +0x2b HP or character +0xb11 HP non-zero) also does nothing.
// A dead target fires spell 0x76 at a position target record {kind 6, ...,
// +0x10 x, y, z} on the stack (exe 0x500fef, 0x5010b0..0x5010ce):
//
//	monster: x, z from its visual (+0xc) +0x118 / +0x120; y is the ground
//	  height 0x420bd0 returns (排除, address stub; st0 is case input
//	  groundHeight, stored as float32 at 0x501033). The lifted y the stub
//	  receives (+0x11c plus +0x190) is not observable. Message 0x590b40
//	  (排除, stub) gets (monster, text [*0x68c09c+0x654]).
//	character: x, y, z from the party view 0x421070 (*0x65a0f8 +0x8c..);
//	  y -= float32 at exe 0x603ac8 (x87 extended, stored float32, 0x50105d).
//	  0x590950 runs natively: its 0x58aad0(8, 0x64bf5c, name C+5, ...) and
//	  0x58b410(slot +0xf1 colour, 0, len, -1) are 排除 stubs.
//
// Unless message mode byte 0x68510c is set, 0x5905c0(1, -1) writes 1 to byte
// 0x69b7b8 (0x64bd54 kept). Then 0x4fb4c0(0x76, res+0x5c, &target,
// res+0xd0, res+0xd4, 0 x6): D4, address stub whose return (case input
// castResult) is this function's eax, which the only caller ignores; nothing
// here reads it or the callee's writes, so there is no Emu/F split and the
// call's arguments and count are the outputs.
//
// Fixture (re/emu/cases/00500f70.json): monster at slot 0..3 of two 2-entry
// lists (*0x683fad, *0x683fb9) with decoy ids >= 0xffffff00 elsewhere; party
// index 0..7. Outside these calc panics (an unknown monster id asserts in
// 0x4e5840 in the original).
const (
	sdfRes    = 0x20000000 // case object res
	sdfText   = 0x20002000 // fixture message text (zstr)
	sdfMon    = 0x20005000 // monster slot k at sdfMon + k*0x400
	sdfChars  = 0x20010000 // *0x685174, stride 0x1862
	sdfTarget = 0x10007fe0 // entry esp 0x10008000 - 0x20: the target record
	sdfGround = 0x10007fd4 // entry esp - 0x2c: the position 0x420bd0 gets
)

func spellDeathFollowupEffectNode(in []int64, r Rand) []int64 {
	if len(in) != 21 {
		panic("spell death followup: expected 21 inputs")
	}
	out := make([]int64, 32)
	out[26], out[27] = in[15], in[16] // 0x69b7b8, 0x64bd54 unchanged by default
	var x, y, z uint32
	switch uint32(in[0]) {
	case 3:
		if in[3] < 0 || in[3] > 3 || uint32(in[2]) >= 0xffffff00 {
			panic("spell death followup: monster outside the fixture lists (fixture cap)")
		}
		if uint32(in[4]) != 0 {
			return out
		}
		x, z = uint32(in[5]), uint32(in[7])
		y = math.Float32bits(float32(math.Float64frombits(uint64(in[9]))))
		out[0], out[1], out[2] = 1, sdfGround, 0
		out[3], out[4], out[5] = 1, sdfMon+in[3]*0x400, sdfText
	case 1:
		if in[1] < 0 || in[1] > 7 {
			panic("spell death followup: party index outside 0..7")
		}
		if uint32(in[10]) != 0 {
			return out
		}
		if interceptDataErr != nil {
			panic(interceptDataErr)
		}
		x, z = uint32(in[11]), uint32(in[13])
		y = sdfLower(uint32(in[12]), float32(interceptData.height))
		out[6], out[7], out[8], out[9] = 1, 8, 0x64bf5c, sdfChars+in[1]*0x1862+5
		out[10], out[11], out[12], out[13] = 1, int64(uint8(in[20])), 0, 0xffffffff
	default:
		return out
	}
	if uint8(in[14]) == 0 {
		out[26] = 1
	}
	out[14], out[15], out[16], out[17] = 1, 0x76, sdfRes+0x5c, sdfTarget
	out[18], out[19] = int64(uint32(in[17])), int64(uint32(in[18]))
	out[28], out[29], out[30], out[31] = 6, int64(x), int64(y), int64(z)
	return out
}

// sdfLower is fld y; fsub c (64-bit mantissa, round to nearest); fstp float32.
func sdfLower(y uint32, c float32) uint32 {
	f := math.Float32frombits(y)
	if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
		return math.Float32bits(float32(float64(f) - float64(c)))
	}
	d := new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Sub(chdF(float64(f)), chdF(float64(c)))
	v, _ := d.Float32()
	return math.Float32bits(v)
}

func init() {
	Nodes["00500f70"] = Node{Addr: "00500f70", In: []string{
		"targetType", "characterIndex", "monsterID", "matchSlot", "monsterCurrentHP", "monsterXBits", "monsterYBits", "monsterZBits", "monsterLiftBits", "groundHeightBits",
		"characterCurrentHP", "partyXBits", "partyYBits", "partyZBits", "messageMode", "messagePending", "messageColor", "effectLevel", "effectBonus", "castResult", "characterNameColor",
	}, Out: []string{
		"ground.count", "ground.position", "ground.flag", "monsterMessage.count", "monsterMessage.monster", "monsterMessage.text",
		"nameLog.count", "nameLog.kind", "nameLog.buffer", "nameLog.name", "nameColor.count", "nameColor.color", "nameColor.start", "nameColor.end",
		"cast.count", "cast.spell", "cast.source", "cast.target", "cast.level", "cast.bonus", "cast.fail", "cast.keep", "cast.out", "cast.mode", "cast.list2", "cast.list1",
		"messagePending", "messageColor", "targetKind", "targetXBits", "targetYBits", "targetZBits",
	}, F: spellDeathFollowupEffectNode}
}
