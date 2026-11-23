package calc

import "fmt"

// character_fatigue_apply 0x52af50(who, amount, scale, target): spend `amount`
// of party member `who`'s stamina +0xb1d (record at *0x685174, stride 0x1862),
// build/wiz8.asm:405367..405497. Nothing happens unless +0xb1d > 0 (signed),
// +0xb11 != 0 and amount != 0. When the low byte of `scale` is set the amount
// is adjusted by a percentage chosen by the encumbrance level +0xbc9 (0,25,50,
// 100,200 from the jump table at 0x52b144; any other level is the srAssertFail
// path 0x52afd7, which stops the game, so calc panics), minus 25 when +0xa15 is
// non-zero or else plus 25 when +0xaa9 is non-zero; the product is a 32-bit
// imul, /100 is the 0x51eb851f reciprocal (truncating), added to amount. The
// spend is clamped to 0..+0xb1d, +0xb1d falls by it, and the fatigue level
// +0x169e becomes 0..4 from 100-100*stamina/+0xb19 (u32 divide: a zero
// maximum is the original's divide fault, calc panics). A changed level runs
// 0x4ee9d0(char). Stamina still above 0 and a changed level >= 3 sets the
// one-shot flag +0x185b with 0x52e690(char, g5ee598, 0, g5ed8c8, g5ed914) when
// it was clear. Stamina 0 with +0xa45 < 0x270f applies state 0x11 for 0x270f
// turns through 0x522fe0(who, 0x11, 0x270f, 0, 0, target==0) and counts +1 at
// target+0x4c when target is non-null.
//
// Emu address-stubs 0x4ee9d0 and 0x522fe0 (D4, composition ruling 2026-10-09)
// and 0x52e690 (excluded voice event): the caller ignores every eax and reads
// no field they write after the call, so the case feeds distinct return values
// from inputs and the outputs are call counts plus every call's arguments (each
// callee runs at most once). The character argument is the fixture address.
// Fixture caps (calc panics past them): who 0..7 (eight records). The
// level < 2 reset of +0x185b at 0x52b138 cannot be reached (the path needs
// level >= 3) and panics in calc.
const (
	cfaChars    = 0x20000000 // chars object of the case
	cfaCharSize = 0x1862
	cfaMembers  = 8
)

func init() {
	Nodes["0052af50"] = Node{Addr: "0052af50", In: []string{
		"who", "amount", "scale", "target",
		"stamina", "hp", "staminaMax", "encumbrance", "statusA15", "statusAA9", "fatigueLevel", "fatigueFlag", "stateA45",
		"targetCount", "g5ed8c8", "g5ed914", "g5ee598", "ret4ee9d0", "ret522fe0", "ret52e690"},
		Out: []string{"stamina", "fatigueLevel", "fatigueFlag", "targetCount",
			"n4ee9d0", "a4ee9d0_1", "n522fe0", "a522fe0_1", "a522fe0_2", "a522fe0_3", "a522fe0_4", "a522fe0_5", "a522fe0_6",
			"n52e690", "a52e690_1", "a52e690_2", "a52e690_3", "a52e690_4", "a52e690_5"},
		F: characterFatigueApplyNode}
}

func characterFatigueApplyNode(in []int64, r Rand) []int64 {
	if len(in) != 20 {
		panic(fmt.Sprintf("character_fatigue_apply: expected 20 inputs, got %d", len(in)))
	}
	who := in[0]
	if who < 0 || who >= cfaMembers {
		panic("character_fatigue_apply: member outside the eight fixture records")
	}
	amount, scale, target := int32(in[1]), uint8(in[2]), uint32(in[3])
	stamina, hp, staMax := int32(in[4]), uint32(in[5]), uint32(in[6])
	enc, a15, aa9 := uint32(in[7]), uint32(in[8]), uint32(in[9])
	level, flag, a45 := uint32(in[10]), uint8(in[11]), uint32(in[12])
	count := uint32(in[13])
	g8c8, g914, g598 := uint32(in[14]), uint32(in[15]), uint32(in[16])
	char := int64(cfaChars + cfaCharSize*who)

	var n4ee, a4ee, n522, n52e int64
	var a522 [6]int64
	var a52e [5]int64
	out := func() []int64 {
		o := []int64{int64(stamina), int64(level), int64(flag), int64(count), n4ee, a4ee, n522}
		o = append(o, a522[:]...)
		o = append(o, n52e)
		return append(o, a52e[:]...)
	}
	if stamina <= 0 || hp == 0 || amount == 0 { // 0x52af74, 0x52af8b, 0x52af93
		return out()
	}
	spend := amount
	if scale != 0 { // 0x52af9f: only the low byte
		var pct int32
		switch enc { // jump table 0x52b144
		case 0:
			pct = 0
		case 1:
			pct = 25
		case 2:
			pct = 50
		case 3:
			pct = 100
		case 4:
			pct = 200
		default:
			panic("character_fatigue_apply: encumbrance level outside 0..4 (srAssertFail)")
		}
		if a15 != 0 {
			pct -= 25 // 0x52b002
		} else if aa9 != 0 {
			pct += 25 // 0x52b011
		}
		hi := int32((int64(pct*amount) * 0x51eb851f) >> 32) // imul by reciprocal, 0x52b01e
		hi >>= 5
		hi += int32(uint32(hi) >> 31)
		spend = hi + amount
	}
	if spend < 0 {
		spend = 0
	} else if spend > stamina {
		spend = stamina
	}
	left := stamina - spend
	if staMax == 0 {
		panic("character_fatigue_apply: zero maximum stamina (divide fault in the original)")
	}
	ecx := 100 - int32(uint32(left)*100/staMax) // 0x52b04a..0x52b05b, 32-bit
	var newLevel uint32
	switch {
	case ecx < 50:
		newLevel = 0
	case ecx < 70:
		newLevel = 1
	case ecx < 85:
		newLevel = 2
	case ecx < 95:
		newLevel = 3
	default:
		newLevel = 4
	}
	stamina, level = left, newLevel
	if newLevel != uint32(in[10]) {
		n4ee, a4ee = 1, char
	}
	if stamina > 0 {
		if newLevel != uint32(in[10]) && newLevel >= 3 { // 0x52b0f6, 0x52b0fd
			if flag == 0 {
				n52e, a52e = 1, [5]int64{char, int64(g598), 0, int64(g8c8), int64(g914)}
				flag = 1
			}
			if level <= 1 {
				panic("character_fatigue_apply: 0x52b138 is unreachable (level >= 3 here)")
			}
		}
		return out()
	}
	if a45 >= 0x270f { // 0x52b0b7
		return out()
	}
	n522 = 1
	a522 = [6]int64{who, 0x11, 0x270f, 0, 0, 0}
	if target == 0 {
		a522[5] = 1
	}
	if target != 0 {
		count++
	}
	return out()
}
