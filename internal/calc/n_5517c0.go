package calc

import "fmt"

// spell_remove_curse 0x5517c0(effect): the Remove Curse effect.
//   - First 0x54df00(effect, 9, 0) clears condition 9 from the targets; its
//     low byte is the default result (0x5517db).
//   - Only effect kind +0x90 == 1 continues (0x5517e6), else that result.
//   - 0x520bc0(character +0x94, power +0xd0, percent +0xd4) tries the
//     character's cursed equipment: 0 returns 0 (0x5518f7); a value outside
//     1..2 (signed, 0x551816/0x55181f) returns the default result.
//   - Unless the silent byte 0x68510c is set: +0x125 = 1, the prefix
//     0x619788 (+0x124 clear) or 0x619794 goes to 0x5905f0(prefix, -1), and
//     0x5905c0(1, -1) sets byte 0x69b7b8 = 1 (the -1 colour is not stored).
//   - The text [*0x68c09c+0x6f0] (all removed, result 2) or +0x6f4 (some
//     removed, result 1) is formatted by 0x517a90 into 0x689fd0 with the
//     character's name (character record *0x685174 + id*0x1862 + 5) and sent to
//     0x5905f0(0x689fd0, -1); +0x124 = 1. Result 2 returns the default
//     result, result 1 returns 0 (0x5518e7).
//
// Emu address-stubs the D4 callees 0x54df00 and 0x520bc0 (ruling 2026-10-09;
// returns from inputs cureRet/removeRet, args and counts are outputs) and the
// 排除 0x5905f0 (args of both calls and count). 0x5905c0 and 0x517a90 run
// natively. The case's texts are the fixture strings "A%s" (+0x6f0) and "B%s"
// (+0x6f4); names are two UTF-16 units per character, then a terminator.
// F calls cureAndResurrectionNode with cond 9, flag 0 and the shared fields
// (power, percent, kind, character, +0x124, +0x125, 0x68510c, 0x69b7b8) plus
// the F-only inputs cureIn<i> (its input i), and feeds its result, +0x124,
// +0x125 and 0x69b7b8 forward; then removeCursePowerApplyNode with the
// character, power, percent and F-only removeIn<i>, using its result.
//
// Fixture cap (panics past it): character ids 0..7.
const (
	srcArena   = 0x20000000 // effect object
	srcText    = 0x689fd0   // 0x517a90's buffer
	srcPrefix1 = 0x619788   // 0x551840
	srcPrefix2 = 0x619794   // 0x551847
	srcShared  = 10 + 16    // node inputs before the F-only ones
)

// srcCureShared maps 0x54df00's shared input slots to this node's inputs (-1:
// cond 9, -2: flag 0).
var srcCureShared = map[int]int{0: -1, 1: -2, 2: 3, 3: 4, 22: 1, 23: 2, 25: 5, 26: 6, 27: 0, 28: 7}

const (
	srcCureFOnly   = carIn - 10
	srcRemoveFOnly = 48 // 0x520bc0 inputs 3..50
	srcIn          = srcShared + srcCureFOnly + srcRemoveFOnly
)

func init() {
	in := []string{"silent", "kind", "character", "power", "percent", "eff124", "eff125", "g69b7b8", "cureRet", "removeRet"}
	for k := 0; k < 8; k++ {
		in = append(in, fmt.Sprintf("name%d_0", k), fmt.Sprintf("name%d_1", k))
	}
	for i := 0; i < carIn; i++ {
		if _, ok := srcCureShared[i]; !ok {
			in = append(in, fmt.Sprintf("cureIn%d", i))
		}
	}
	for i := 3; i < 3+srcRemoveFOnly; i++ {
		in = append(in, fmt.Sprintf("removeIn%d", i))
	}
	out := []string{"ret", "eff124", "eff125", "g69b7b8",
		"cure.count", "cure.arg1", "cure.arg2", "cure.arg3",
		"remove.count", "remove.arg1", "remove.arg2", "remove.arg3",
		"message.count", "message.arg1.call0", "message.arg1.call1", "message.arg2.call0", "message.arg2.call1",
		"text0", "text1", "text2", "text3"}
	Nodes["005517c0"] = Node{Addr: "005517c0", In: in, Out: out, F: spellRemoveCurseNode,
		Emu: func(in []int64, r Rand) []int64 { return srcRun(in, r, false) }}
}

func spellRemoveCurseNode(in []int64, r Rand) []int64 { return srcRun(in, r, true) }

func srcRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != srcIn {
		panic("remove curse spell: wrong input count")
	}
	if in[2] < 0 || in[2] > 7 {
		panic("remove curse spell: fixture caps character ids at 0..7")
	}
	u := func(i int) uint32 { return uint32(in[i]) }
	silent, kind, char, power, percent := u(0)&0xff, u(1), u(2), u(3), u(4)
	eff124, eff125, g69b7b8 := u(5)&0xff, u(6)&0xff, u(7)&0xff
	bl := u(8) & 0xff
	out := make([]int64, 21)
	if real {
		v := make([]int64, carIn)
		j := srcShared
		for i := range v {
			s, ok := srcCureShared[i]
			switch {
			case !ok:
				v[i] = in[j]
				j++
			case s == -1:
				v[i] = 9
			case s >= 0:
				v[i] = in[s]
			}
		}
		c := cureAndResurrectionNode(v, r)
		bl, eff124, eff125, g69b7b8 = uint32(c[0])&0xff, uint32(c[1]), uint32(c[2]), uint32(c[3])
	}
	out[4], out[5], out[6], out[7] = 1, srcArena, 9, 0
	ret := bl
	edi := int32(u(9))
	if kind == 1 {
		if real {
			v := []int64{int64(char), int64(power), int64(int32(percent))}
			v = append(v, in[srcShared+srcCureFOnly:]...)
			edi = int32(removeCursePowerApplyNode(v, r)[0])
		}
		out[8], out[9], out[10], out[11] = 1, int64(char), int64(power), int64(percent)
		switch {
		case edi == 0:
			ret = 0
		case edi == 1 || edi == 2:
			var msg []uint32
			if silent == 0 {
				eff125 = 1
				p := uint32(srcPrefix2)
				if eff124 == 0 {
					p = srcPrefix1
				}
				msg = append(msg, p)
				g69b7b8 = 1
			}
			// 0x517a90: "A%s"/"B%s" with the name, NUL-terminated; units past
			// the terminator keep the buffer's earlier (zero) contents.
			text := []int64{'B'}
			if edi == 2 {
				text[0] = 'A'
			}
			for w := 0; w < 2 && in[10+2*int(char)+w] != 0; w++ {
				text = append(text, in[10+2*int(char)+w])
			}
			copy(out[17:], append(text, 0))
			msg = append(msg, srcText)
			out[12] = int64(len(msg))
			for k, p := range msg {
				out[13+k], out[15+k] = int64(p), 0xffffffff
			}
			eff124 = 1
			if edi == 1 {
				ret = 0
			}
		}
	}
	out[0], out[1], out[2], out[3] = int64(ret), int64(eff124), int64(eff125), int64(g69b7b8)
	return out
}
