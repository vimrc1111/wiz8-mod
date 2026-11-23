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

// weapon_damage 0x5420b0(who, hand, mode, *count, *maxed): damage of one swing
// of one hand, as a mix of the attacker record (C = *0x685174 + 0x1862*who), the
// target (monster at G+0x8cd, or character G+0x8c9 when G+0x8c5 is not 3) and
// the weapon dice. Output: eax = damage, *count = number of dice rolled,
// *maxed = (c2 - N*c1) * max * count <= total (x87 extended).
//
// Emu address-stubs the D4 callees 0x517970 (dice), 0x51cce0, 0x4ed2d0 and 0x553f10
// and the 排除 0x52e690: only their call counts and recorded arguments are
// output, the returns come from the dd*, r51, r4d* cells. F calls the real Go
// callees (diceNode, weaponDamageBonusItemNode, characterPostureEligibilityNode;
// 0x553f10's return and writes are never read here) on the F-only cells fcls*
// (item class u16 of record r) and fhp/fst/fmain/foff/fdone (the target's
// life, status, posture words and the action-completed flag for 0x4ed2d0).
//
// Native in the emulator and modelled here: 0x4ed2b0 (G+0x918 ? G+0x90c : G+0x910),
// 0x4e5550/0x4e5620/0x4e5720 (monster id -> list entry 1 -> template mtidx),
// 0x51c8b0 (jump table of the exe), 0x5479b0 + 0x5e1c22 (level scale, ftol).
//
// Replay caps (calc panics past them): item record index 0..7, equipment slot
// s70 0..11 and s74 -1..11, class index -1..14, monster template index 0..2.
// Original assert paths (0x542100, 0x54218b, 0x5423b9, 0x542457, 0x542507) panic.

const (
	wdnWho     = 0
	wdnHand    = 1
	wdnMode    = 2
	wdnTgt     = 3
	wdnTchar   = 4
	wdnMid     = 5
	wdnG90c    = 6
	wdnG910    = 7
	wdnG918    = 8
	wdnF9a4    = 9
	wdnF9a5    = 10
	wdnS70     = 11
	wdnS74     = 12
	wdnLvi     = 13
	wdnLvm1    = 14
	wdnLv0     = 15
	wdnLv1     = 16
	wdnLv2     = 17
	wdnLv3     = 18
	wdnLv4     = 19
	wdnLv5     = 20
	wdnLv6     = 21
	wdnLv7     = 22
	wdnLv8     = 23
	wdnLv9     = 24
	wdnLv10    = 25
	wdnLv11    = 26
	wdnLv12    = 27
	wdnLv13    = 28
	wdnLv14    = 29
	wdnIt0     = 30
	wdnIt1     = 31
	wdnIt2     = 32
	wdnIt3     = 33
	wdnIt4     = 34
	wdnIt5     = 35
	wdnIt6     = 36
	wdnIt7     = 37
	wdnIt8     = 38
	wdnIt9     = 39
	wdnIt10    = 40
	wdnIt11    = 41
	wdnHf0     = 42
	wdnHf1     = 43
	wdnHb0     = 44
	wdnHb1     = 45
	wdnHd0     = 46
	wdnHd1     = 47
	wdnPwr     = 48
	wdnTA      = 49
	wdnTDA     = 50
	wdnTS      = 51
	wdnMtidx   = 52
	wdnMA      = 53
	wdnMK      = 54
	wdnMD      = 55
	wdnTcb     = 56
	wdnTcf     = 57
	wdnIdice0  = 58
	wdnIdice1  = 59
	wdnIdice2  = 60
	wdnIdice3  = 61
	wdnIdice4  = 62
	wdnIdice5  = 63
	wdnIdice6  = 64
	wdnIdice7  = 65
	wdnIb0     = 66
	wdnIb1     = 67
	wdnIb2     = 68
	wdnIb3     = 69
	wdnIb4     = 70
	wdnIb5     = 71
	wdnIb6     = 72
	wdnIb7     = 73
	wdnG68c510 = 74
	wdnCbase   = 75
	wdnDd0     = 76
	wdnDd1     = 77
	wdnDd2     = 78
	wdnDd3     = 79
	wdnDd4     = 80
	wdnDd5     = 81
	wdnDd6     = 82
	wdnDd7     = 83
	wdnR51     = 84
	wdnR4d0    = 85
	wdnR4d1    = 86
	wdnFcls0   = 87
	wdnFcls1   = 88
	wdnFcls2   = 89
	wdnFcls3   = 90
	wdnFcls4   = 91
	wdnFcls5   = 92
	wdnFcls6   = 93
	wdnFcls7   = 94
	wdnFhp     = 95
	wdnFst     = 96
	wdnFmain   = 97
	wdnFoff    = 98
	wdnFdone   = 99
)

const (
	wdnORet     = 0
	wdnOGout    = 1
	wdnOGpad0   = 2
	wdnOGpad1   = 3
	wdnOFout    = 4
	wdnOFpad0   = 5
	wdnOFpad1   = 6
	wdnOFpad2   = 7
	wdnOC517    = 8
	wdnOC51c    = 9
	wdnOC4ed    = 10
	wdnOC553    = 11
	wdnOC52e    = 12
	wdnOA4ed1_0 = 13
	wdnOA4ed1_1 = 14
	wdnOA4ed2_0 = 15
	wdnOA4ed2_1 = 16
	wdnOA4ed3_0 = 17
	wdnOA4ed3_1 = 18
	wdnOA553_1  = 19
	wdnOA553_2  = 20
	wdnOA553_3  = 21
	wdnOA553_4  = 22
	wdnOA52e_1  = 23
	wdnOA52e_2  = 24
	wdnOA52e_3  = 25
	wdnOA52e_4  = 26
	wdnOA52e_5  = 27
	wdnOA51c_1  = 28
)

var wdnNames = []string{
	"who",
	"hand",
	"mode",
	"tgt",
	"tchar",
	"mid",
	"g90c",
	"g910",
	"g918",
	"f9a4",
	"f9a5",
	"s70",
	"s74",
	"lvi",
	"lvm1",
	"lv0",
	"lv1",
	"lv2",
	"lv3",
	"lv4",
	"lv5",
	"lv6",
	"lv7",
	"lv8",
	"lv9",
	"lv10",
	"lv11",
	"lv12",
	"lv13",
	"lv14",
	"it0",
	"it1",
	"it2",
	"it3",
	"it4",
	"it5",
	"it6",
	"it7",
	"it8",
	"it9",
	"it10",
	"it11",
	"hf0",
	"hf1",
	"hb0",
	"hb1",
	"hd0",
	"hd1",
	"pwr",
	"tA",
	"tDA",
	"tS",
	"mtidx",
	"mA",
	"mK",
	"mD",
	"tcb",
	"tcf",
	"idice0",
	"idice1",
	"idice2",
	"idice3",
	"idice4",
	"idice5",
	"idice6",
	"idice7",
	"ib0",
	"ib1",
	"ib2",
	"ib3",
	"ib4",
	"ib5",
	"ib6",
	"ib7",
	"g68c510",
	"cbase",
	"dd0",
	"dd1",
	"dd2",
	"dd3",
	"dd4",
	"dd5",
	"dd6",
	"dd7",
	"r51",
	"r4d0",
	"r4d1",
	"fcls0",
	"fcls1",
	"fcls2",
	"fcls3",
	"fcls4",
	"fcls5",
	"fcls6",
	"fcls7",
	"fhp",
	"fst",
	"fmain",
	"foff",
	"fdone",
}

var wdnOutNames = []string{
	"ret",
	"gout",
	"gpad0",
	"gpad1",
	"fout",
	"fpad0",
	"fpad1",
	"fpad2",
	"c517",
	"c51c",
	"c4ed",
	"c553",
	"c52e",
	"a4ed1_0",
	"a4ed1_1",
	"a4ed2_0",
	"a4ed2_1",
	"a4ed3_0",
	"a4ed3_1",
	"a553_1",
	"a553_2",
	"a553_3",
	"a553_4",
	"a52e_1",
	"a52e_2",
	"a52e_3",
	"a52e_4",
	"a52e_5",
	"a51c_1",
}

// Storage width of each input cell, narrowed by the replay caps (record counts and ID ranges).
var wdnRange = [][2]int64{
	{0, 7},                    // who
	{0, 1},                    // hand
	{0, 4294967295},           // mode
	{0, 4294967295},           // tgt
	{-1, 7},                   // tchar
	{0, 4294967295},           // mid
	{0, 4294967295},           // g90c
	{0, 4294967295},           // g910
	{0, 4294967295},           // g918
	{0, 255},                  // f9a4
	{0, 255},                  // f9a5
	{0, 11},                   // s70
	{-1, 11},                  // s74
	{-1, 14},                  // lvi
	{0, 4294967295},           // lvm1
	{0, 4294967295},           // lv0
	{0, 4294967295},           // lv1
	{0, 4294967295},           // lv2
	{0, 4294967295},           // lv3
	{0, 4294967295},           // lv4
	{0, 4294967295},           // lv5
	{0, 4294967295},           // lv6
	{0, 4294967295},           // lv7
	{0, 4294967295},           // lv8
	{0, 4294967295},           // lv9
	{0, 4294967295},           // lv10
	{0, 4294967295},           // lv11
	{0, 4294967295},           // lv12
	{0, 4294967295},           // lv13
	{0, 4294967295},           // lv14
	{0, 7},                    // it0
	{0, 7},                    // it1
	{0, 7},                    // it2
	{0, 7},                    // it3
	{0, 7},                    // it4
	{0, 7},                    // it5
	{0, 7},                    // it6
	{0, 7},                    // it7
	{0, 7},                    // it8
	{0, 7},                    // it9
	{0, 7},                    // it10
	{0, 7},                    // it11
	{0, 4294967295},           // hf0
	{0, 4294967295},           // hf1
	{-2147483648, 2147483647}, // hb0
	{-2147483648, 2147483647}, // hb1
	{0, 4294967295},           // hd0
	{0, 4294967295},           // hd1
	{-128, 127},               // pwr
	{0, 255},                  // tA
	{-2147483648, 2147483647}, // tDA
	{0, 255},                  // tS
	{0, 2},                    // mtidx
	{0, 255},                  // mA
	{0, 4294967295},           // mK
	{-128, 127},               // mD
	{0, 255},                  // tcb
	{0, 255},                  // tcf
	{0, 4294967295},           // idice0
	{0, 4294967295},           // idice1
	{0, 4294967295},           // idice2
	{0, 4294967295},           // idice3
	{0, 4294967295},           // idice4
	{0, 4294967295},           // idice5
	{0, 4294967295},           // idice6
	{0, 4294967295},           // idice7
	{0, 255},                  // ib0
	{0, 255},                  // ib1
	{0, 255},                  // ib2
	{0, 255},                  // ib3
	{0, 255},                  // ib4
	{0, 255},                  // ib5
	{0, 255},                  // ib6
	{0, 255},                  // ib7
	{0, 4294967295},           // g68c510
	{0, 4294967295},           // cbase
	{-2147483648, 2147483647}, // dd0
	{-2147483648, 2147483647}, // dd1
	{-2147483648, 2147483647}, // dd2
	{-2147483648, 2147483647}, // dd3
	{-2147483648, 2147483647}, // dd4
	{-2147483648, 2147483647}, // dd5
	{-2147483648, 2147483647}, // dd6
	{-2147483648, 2147483647}, // dd7
	{0, 4294967295},           // r51
	{0, 4294967295},           // r4d0
	{0, 4294967295},           // r4d1
	{0, 65535},                // fcls0
	{0, 65535},                // fcls1
	{0, 65535},                // fcls2
	{0, 65535},                // fcls3
	{0, 65535},                // fcls4
	{0, 65535},                // fcls5
	{0, 65535},                // fcls6
	{0, 65535},                // fcls7
	{0, 4294967295},           // fhp
	{0, 4294967295},           // fst
	{0, 4294967295},           // fmain
	{0, 4294967295},           // foff
	{0, 4294967295},           // fdone
}

const wdnCharStride = 0x1862 // character record stride

type wdnTables struct {
	c1, c2 float32 // 0x5ebcf4, 0x5ebb38
	d1, d2 uint32  // 0x5ed8c8, 0x5ed914: passed to 0x52e690
}

var wdnExe, wdnExeErr = wdnLoad()

func wdnLoad() (t wdnTables, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("weapon damage: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	rd := func(va uint32) (uint32, error) {
		o, err := peOffset(b, va)
		if err != nil || o+4 > len(b) {
			return 0, fmt.Errorf("weapon damage: constant %#x not in exe", va)
		}
		return binary.LittleEndian.Uint32(b[o:]), nil
	}
	var w [4]uint32
	for i, va := range []uint32{0x5ebcf4, 0x5ebb38, 0x5ed8c8, 0x5ed914} {
		if w[i], err = rd(va); err != nil {
			return t, err
		}
	}
	t.c1, t.c2 = math.Float32frombits(w[0]), math.Float32frombits(w[1])
	t.d1, t.d2 = w[2], w[3]
	return t, nil
}

func init() {
	Nodes["005420b0"] = Node{Addr: "005420b0", In: wdnNames, Out: wdnOutNames,
		F:   func(in []int64, r Rand) []int64 { return wdnRun(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return wdnRun(in, r, false) }}
}

func weaponDamageNode(in []int64, r Rand) []int64 { return wdnRun(in, r, true) }

// wdnDiv100 is the compiler's signed /100 at 0x54263b..0x542645 (magic 0x51eb851f).
func wdnDiv100(v int32) int32 { return v / 100 }

// wdnPostureCall is the real 0x4ed2d0(tchar, posture, 1) on a callee world made of the target's F-only cells.
func wdnPostureCall(in []int64, tchar, posture int64, r Rand) bool {
	cv := swBase53
	cv[0] = tchar
	cv[27+3*tchar], cv[28+3*tchar], cv[29+3*tchar] = 1, in[wdnFhp], in[wdnFst]
	cv[9], cv[10], cv[263] = in[wdnFmain], in[wdnFoff], in[wdnFdone]
	cv[261], cv[262] = posture, 1
	return characterPostureEligibilityNode(cv[:], r)[0]&0xff != 0
}

func wdnRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(wdnNames) {
		panic("weapon damage: wrong input count")
	}
	for i, v := range in {
		if v < wdnRange[i][0] || v > wdnRange[i][1] {
			panic("weapon damage: " + wdnNames[i] + " outside the replay domain")
		}
	}
	if wdnExeErr != nil {
		panic(wdnExeErr)
	}
	if attackSwingErr != nil {
		panic(attackSwingErr)
	}
	if statusRebuildTablesErr != nil {
		panic(statusRebuildTablesErr)
	}
	g := func(i int) int64 { return in[i] }
	u := func(i int) uint32 { return uint32(in[i]) }
	who, hand, mode := g(wdnWho), g(wdnHand), u(wdnMode)
	out := make([]int64, len(wdnOutNames))
	cptr := int64(uint32(g(wdnCbase) + who*wdnCharStride))

	var bl, attr, flag bool // bl, byte [esp+0x13], byte [esp+0x34]
	var calls4ed int64
	tchar := g(wdnTchar)
	monster := u(wdnTgt) == 3
	if monster {
		if u(wdnMid) == 0xffffffff {
			panic("weapon damage: monster id -1 asserts at 0x542100")
		}
		k := u(wdnMK)
		bl, attr, flag = k == 4, g(wdnMA) != 0, k == 1 || k == 8
	} else {
		if tchar < 0 {
			panic("weapon damage: target character -1 asserts at 0x54218b")
		}
		sel := wdnG910
		if u(wdnG918) != 0 {
			sel = wdnG90c
		}
		bl, attr = u(sel) == 2, g(wdnTA) != 0
		call := func(posture int64) bool {
			k := calls4ed
			calls4ed++
			out[wdnOA4ed1_0+int(k)] = tchar
			out[wdnOA4ed2_0+int(k)] = posture
			out[wdnOA4ed3_0+int(k)] = 1
			if real {
				return wdnPostureCall(in, tchar, posture, r)
			}
			return g(wdnR4d0+int(k))&0xff != 0
		}
		flag = call(4) || call(5)
	}
	out[wdnOC4ed] = calls4ed

	count := int64(1)
	if mode != 4 && !(mode > 6 && mode <= 8) {
		if bl || attr || g(wdnF9a4) != 0 {
			count = 2
		}
		if g(wdnF9a5) != 0 {
			// 0x5479b0(C, 9, 100.0f): level word of the class index (or -1), then _ftol.
			ci := g(wdnLvi)
			lv := u(wdnLvm1)
			if ci >= 0 {
				lv = u(wdnLv0 + int(ci))
			}
			esi := chdFtol(chdScale(chdF(100), lv))
			roll := uint32(r.R(100))
			add := int64(0)
			if roll+0xf < esi {
				add = 1
				if roll+0x4b < esi {
					add = 2
					if roll+0x5f < esi {
						add = 3
					}
				}
			}
			count += add
		}
	}
	if mode == 3 {
		count++
	}
	if count > 1 && flag && g(wdnF9a4) == 0 {
		count--
	}

	hf := wdnHf0 + int(hand)
	tmplB := u(wdnTcb)
	itemB := func(slot int64) uint32 { // items[C.item[slot]] byte +0x61
		return u(wdnIb0 + int(g(wdnIt0+int(slot))))
	}
	if monster && g(hf) != 0 {
		if b := itemB(g(wdnS70)); b != 0xff && b == tmplB {
			count++
		}
		if g(wdnS74) != -1 {
			if b := itemB(g(wdnS74)); b != 0xff && b == tmplB {
				count++
			}
		}
	}

	// Dice word: Z i16 | N u8 | S u8.
	var dice uint32
	itemDice := func(idx int64) uint32 { return u(wdnIdice0 + int(idx)) }
	switch mode {
	case 5:
		if g(hf) != 0 {
			panic("weapon damage: mode 5 with the hand flag set asserts at 0x542507")
		}
		if g(wdnHf1) != 0 {
			dice = itemDice(g(wdnIt0 + 7))
		} else {
			dice = u(wdnHd1)
		}
	case 6:
		if g(hf) != 0 {
			panic("weapon damage: mode 6 with the hand flag set asserts at 0x542457")
		}
		if g(wdnHf0) != 0 {
			dice = itemDice(wdnPrimary(in, out, real))
		} else {
			dice = u(wdnHd0)
		}
	default:
		if mode != 3 && g(hf) == 0 {
			panic("weapon damage: hand without the flag asserts at 0x5423b9")
		}
		switch {
		case g(hf) == 0:
			dice = u(wdnHd0 + int(hand))
		case hand != 0:
			dice = itemDice(g(wdnIt0 + 7))
		default:
			dice = itemDice(wdnPrimary(in, out, real))
		}
	}

	// 0x517970 once per count.
	var total int32
	z, n, s := int64(int16(dice)), int64(dice>>16&0xff), int64(dice>>24)
	if dt, ok := r.(interface {
		DiceTotal(count, z, n, s int64) int64
	}); ok && real {
		// calc fight's enumerator: the count rolls' sum as one branch (same distribution).
		total = int32(dt.DiceTotal(count, z, n, s))
	} else {
		for k := int64(0); k < count; k++ {
			if real {
				total += int32(diceNode([]int64{z, n, s}, r)[0])
			} else {
				total += int32(g(wdnDd0 + int(k)))
			}
		}
	}
	out[wdnOC517] = count

	maxed := int64(0)
	if n := int64(dice >> 16 & 0xff); n > 0 {
		mx := n*int64(dice>>24) + int64(int16(dice))
		v := chdMul(chdF(float64(n)), chdF(float64(wdnExe.c1)))
		v = new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Sub(chdF(float64(wdnExe.c2)), v)
		v = chdMul(v, chdF(float64(mx)))
		v = chdMul(v, chdF(float64(count)))
		if v.Cmp(chdF(float64(total))) <= 0 {
			maxed = 1
		}
	}

	// R(100) < c only splits two ways; calc fight's enumerator branches it so (Rogue's extra
	// strikes reach here under the damage dice, and 100 ways there cost ~50x per Fight).
	below := func(c int64) bool {
		if b, ok := r.(interface{ Below(n, c int) bool }); ok && real {
			return b.Below(100, int(min(c, 100)))
		}
		return int64(r.R(100)) < c
	}
	if count > 1 && below(20*count-20) {
		out[wdnOC52e]++
		for i, v := range []int64{cptr, g(wdnG68c510), 0, int64(wdnExe.d1), int64(wdnExe.d2)} {
			out[wdnOA52e_1+i] = v
		}
	}

	pwr, hb := g(wdnPwr), g(wdnHb0+int(hand))
	dmg := wdnDiv100((int32(pwr)+int32(hb)+100)*total + 0x32)
	if dmg < 1 {
		dmg = 1
	}
	esi := dmg
	if monster {
		if d := g(wdnMD) + g(wdnTcf); d != 0 {
			esi = wdnDiv100((100-int32(d))*esi + 0x32)
		}
		if esi < 0 {
			esi = 0
		}
		if esi < 0 {
			panic("weapon damage: unreachable 0x5426f6")
		}
	} else {
		if d := int32(g(wdnTDA)); d != 0 {
			esi = wdnDiv100((100-d)*esi + 0x32)
		}
		if esi < 0 {
			esi = 0
		}
		if esi < 0 {
			panic("weapon damage: unreachable 0x5426f6")
		}
		if g(wdnTS) != 0 { // 0x553f10(target, 0x25, 1, 0): return and writes unread
			out[wdnOC553]++
			for i, v := range []int64{int64(uint32(g(wdnCbase) + tchar*wdnCharStride)), 0x25, 1, 0} {
				out[wdnOA553_1+i] = v
			}
		}
	}

	out[wdnORet] = int64(esi)
	out[wdnOGout] = count
	out[wdnOGpad0], out[wdnOGpad1] = 0xa5a5a5a5, 0x5a5a5a5a
	out[wdnOFout] = maxed
	out[wdnOFpad0], out[wdnOFpad1], out[wdnOFpad2] = 0xc3, 0xd2, 0xe1
	return out
}

// wdnPrimary is the weapon-slot item index of 0x542418/0x54249d: the main hand
// item (slot 6), or the companion slot's item (0x51c8b0(6)) when 0x51cce0 says
// the main hand item imports a damage bonus.
func wdnPrimary(in []int64, out []int64, real bool) int64 {
	item := in[wdnIt0+6]
	out[wdnOC51c]++
	out[wdnOA51c_1] = item
	var imports bool
	if real {
		imports = weaponDamageBonusItemNode([]int64{item, in[wdnFcls0+int(item)]}, nil)[0] != 0
	} else {
		imports = in[wdnR51]&0xff != 0
	}
	if imports {
		return in[wdnIt0+int(attackSwingTab.slot[0])]
	}
	return item
}
