package calc

// monster_damage 0x542960(M, dice, *count): damage of one monster attack. M is the
// attacker monster record (kind word at +0x2e5, template slot at +0x8), the dice
// word sits at dice+0x17, *count receives the number of dice rolled.
// The target comes from the combat global G (*0x6836a8): kind +0x8c5 (1 = character
// +0x8c9 into the character array *0x685174, stride 0x1862; anything else = monster
// id +0x8cd looked up in the monster list, its template read through 0x4e5720).
// Dice count: 1, 2 when the target is a bleeding/flagged one (bl, attribute byte or
// G+0x9a4), up to +3 from a percent roll against the attacker template's level scale
// (G+0x9a5), +1 for attacker kind 3, -1 again when the target flag is set and
// G+0x9a4 is clear. Damage is the dice sum (at least 1) cut by the target's
// absorption: kind 3 uses monster +0x1e1 (signed) + template +0xcf, every other
// non-character kind reads the template as if it were a character record
// (+0xed9 dword, +0x71b byte), a character target reads its own record.
//
// Emu address-stubs the D4 callees 0x517970 (dice), 0x4ed2d0 (posture) and
// 0x553f10 (natural skill growth): only their call counts and recorded arguments
// are output, the returns come from the dd*, r4d* and r553 cells (0x553f10's
// return is never read). F calls the real Go callees (diceNode,
// characterPostureEligibilityNode) on the F-only cells fhp/fst/fmain/foff/fdone
// (the target's life, status, posture words and the action-completed flag for
// 0x4ed2d0) and on the dice record cells dz/dn/ds at dice+0x17..0x1a.
//
// Native in the emulator and modelled here: 0x4ed2b0 (G+0x918 ? G+0x90c : G+0x910),
// 0x4e5550/0x4e5620/0x4e5720/0x5100b0/0x5101b0 (monster id -> list entry -> template
// slot; the id table check of 0x5100b0 is satisfied by the fixture),
// 0x547a00 + 0x5e1c22 (level scale of the attacker template byte +0x24f, ftol).
//
// Replay caps (calc panics past them): character index 0..7, template slot 0..2.
// Original assert paths (0x542990, 0x542a43) panic; 0x542c75 (negative result after
// the clamps) cannot be reached and panics.

const (
	mdnTgt = iota
	mdnTchar
	mdnMid
	mdnG90c
	mdnG910
	mdnG918
	mdnF9a4
	mdnF9a5
	mdnKind
	mdnMtA
	mdnMtT
	mdnMk
	mdnMattr
	mdnMmod
	mdnTcf
	mdnTed9
	mdnT71b
	mdnTlv
	mdnC17b5
	mdnCed9
	mdnC71b
	mdnCbase
	mdnTp0
	mdnTp1
	mdnTp2
	mdnDd0
	mdnDd1
	mdnDd2
	mdnDd3
	mdnDd4
	mdnDd5
	mdnR4d0
	mdnR4d1
	mdnR553
	mdnDz
	mdnDn
	mdnDs
	mdnFhp
	mdnFst
	mdnFmain
	mdnFoff
	mdnFdone
)

const (
	mdnORet = iota
	mdnOGout
	mdnOGpad0
	mdnOGpad1
	mdnOC517
	mdnOC4ed
	mdnOC553
	mdnOA517   // six cells
	mdnOA4ed1  = mdnOA517 + 6
	mdnOA4ed2  = mdnOA4ed1 + 2
	mdnOA4ed3  = mdnOA4ed2 + 2
	mdnOA553   = mdnOA4ed3 + 2
	mdnOutSize = mdnOA553 + 4
)

var mdnNames = []string{
	"tgt", "tchar", "mid", "g90c", "g910", "g918", "f9a4", "f9a5", "kind", "mtA", "mtT", "mk", "mattr", "mmod",
	"tcf", "ted9", "t71b", "tlv", "c17b5", "ced9", "c71b", "cbase", "tp0", "tp1", "tp2",
	"dd0", "dd1", "dd2", "dd3", "dd4", "dd5", "r4d0", "r4d1", "r553", "dz", "dn", "ds",
	"fhp", "fst", "fmain", "foff", "fdone",
}

var mdnOutNames = []string{
	"ret", "gout", "gpad0", "gpad1", "c517", "c4ed", "c553",
	"a517_0", "a517_1", "a517_2", "a517_3", "a517_4", "a517_5",
	"a4ed1_0", "a4ed1_1", "a4ed2_0", "a4ed2_1", "a4ed3_0", "a4ed3_1",
	"a553_1", "a553_2", "a553_3", "a553_4",
}

const (
	mdnCharStride = 0x1862     // character record stride
	mdnDiceArg    = 0x20004017 // dice+0x17 of the case's dice object (arena 0x20000000 + 0x4000)
	mdnMaxDice    = 6          // 1 + 1 + 3 + 1
)

// Storage width of each input cell, narrowed by the replay caps (record counts and ID ranges).
var mdnRange = [][2]int64{
	{0, 4294967295},           // tgt
	{-1, 7},                   // tchar
	{0, 4294967295},           // mid
	{0, 4294967295},           // g90c
	{0, 4294967295},           // g910
	{0, 4294967295},           // g918
	{0, 255},                  // f9a4
	{0, 255},                  // f9a5
	{0, 4294967295},           // kind
	{0, 2},                    // mtA
	{0, 2},                    // mtT
	{0, 4294967295},           // mk
	{0, 255},                  // mattr
	{-128, 127},               // mmod
	{0, 255},                  // tcf
	{-2147483648, 2147483647}, // ted9
	{0, 255},                  // t71b
	{0, 255},                  // tlv
	{0, 255},                  // c17b5
	{-2147483648, 2147483647}, // ced9
	{0, 255},                  // c71b
	{0x20010000, 0x20010000},  // cbase
	{0x20006000, 0x20006000},  // tp0
	{0x20007000, 0x20007000},  // tp1
	{0x20008000, 0x20008000},  // tp2
	{-2147483648, 2147483647}, // dd0
	{-2147483648, 2147483647}, // dd1
	{-2147483648, 2147483647}, // dd2
	{-2147483648, 2147483647}, // dd3
	{-2147483648, 2147483647}, // dd4
	{-2147483648, 2147483647}, // dd5
	{0, 4294967295},           // r4d0
	{0, 4294967295},           // r4d1
	{0, 4294967295},           // r553
	{-32768, 32767},           // dz
	{0, 255},                  // dn
	{0, 255},                  // ds
	{0, 4294967295},           // fhp
	{0, 4294967295},           // fst
	{0, 4294967295},           // fmain
	{0, 4294967295},           // foff
	{0, 4294967295},           // fdone
}

func init() {
	Nodes["00542960"] = Node{Addr: "00542960", In: mdnNames, Out: mdnOutNames,
		F:   func(in []int64, r Rand) []int64 { return mdnRun(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return mdnRun(in, r, false) }}
}

func monsterDamageNode(in []int64, r Rand) []int64 { return mdnRun(in, r, true) }

// mdnDiv100 is the compiler's signed /100 at 0x542bff..0x542c14 (magic 0x51eb851f).
func mdnDiv100(v int32) int32 { return v / 100 }

// mdnPostureCall is the real 0x4ed2d0(tchar, posture, 1) on a callee world made of the target's F-only cells.
func mdnPostureCall(in []int64, tchar, posture int64, r Rand) bool {
	cv := swBase53
	cv[0] = tchar
	cv[27+3*tchar], cv[28+3*tchar], cv[29+3*tchar] = 1, in[mdnFhp], in[mdnFst]
	cv[9], cv[10], cv[263] = in[mdnFmain], in[mdnFoff], in[mdnFdone]
	cv[261], cv[262] = posture, 1
	return characterPostureEligibilityNode(cv[:], r)[0]&0xff != 0
}

func mdnRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(mdnNames) {
		panic("monster damage: wrong input count")
	}
	for i, v := range in {
		if v < mdnRange[i][0] || v > mdnRange[i][1] {
			panic("monster damage: " + mdnNames[i] + " outside the replay domain")
		}
	}
	if statusRebuildTablesErr != nil {
		panic(statusRebuildTablesErr)
	}
	u := func(i int) uint32 { return uint32(in[i]) }
	out := make([]int64, mdnOutSize)

	tgt := u(mdnTgt)
	var bl, attr, flag bool // bl, byte [esp+0x13], byte [esp+0x24]
	var calls4ed int64
	var rec int64 // [esp+0x1c]: character record or target template
	var absorb int32
	var growth bool
	if tgt == 1 {
		tchar := in[mdnTchar]
		if tchar < 0 {
			panic("monster damage: target character -1 asserts at 0x542990")
		}
		sel := mdnG910
		if u(mdnG918) != 0 {
			sel = mdnG90c
		}
		bl, attr = u(sel) == 2, in[mdnC17b5] != 0
		call := func(posture int64) bool {
			k := calls4ed
			calls4ed++
			out[mdnOA4ed1+int(k)] = tchar
			out[mdnOA4ed2+int(k)] = posture
			out[mdnOA4ed3+int(k)] = 1
			if real {
				return mdnPostureCall(in, tchar, posture, r)
			}
			return in[mdnR4d0+int(k)]&0xff != 0
		}
		flag = call(4) || call(5)
		rec = in[mdnCbase] + tchar*mdnCharStride
		absorb, growth = int32(in[mdnCed9]), in[mdnC71b] != 0
	} else {
		if u(mdnMid) == 0xffffffff {
			panic("monster damage: monster id -1 asserts at 0x542a43")
		}
		k := u(mdnMk)
		bl, attr, flag = k == 4, in[mdnMattr] != 0, k == 1 || k == 8
		rec = in[mdnTp0+int(in[mdnMtT])]
		absorb, growth = int32(in[mdnTed9]), in[mdnT71b] != 0
	}
	out[mdnOC4ed] = calls4ed

	kind := u(mdnKind)
	count := int64(1)
	if kind != 4 && kind != 7 && kind != 8 {
		if bl || attr || in[mdnF9a4] != 0 {
			count = 2
		}
		if in[mdnF9a5] != 0 {
			// 0x547a00(attacker template, 9, 100.0f): template level byte +0x24f, then _ftol.
			esi := chdFtol(chdScale(chdF(100), u(mdnTlv)))
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
	if kind == 3 {
		count++
	}
	if count > 1 && flag && in[mdnF9a4] == 0 {
		count--
	}
	if count > mdnMaxDice {
		panic("monster damage: more dice than the replay records")
	}

	// 0x517970 once per count, on dice+0x17.
	var total int32
	for k := int64(0); k < count; k++ {
		if real {
			total += int32(diceNode([]int64{in[mdnDz], in[mdnDn], in[mdnDs]}, r)[0])
		} else {
			total += int32(in[mdnDd0+int(k)])
		}
		out[mdnOA517+int(k)] = mdnDiceArg
	}
	out[mdnOC517] = count

	dmg := total
	if dmg < 1 {
		dmg = 1
	}
	if tgt == 3 {
		if d := int32(in[mdnMmod]) + int32(in[mdnTcf]); d != 0 {
			dmg = mdnDiv100((100-d)*dmg + 0x32)
			if dmg < 0 {
				dmg = 0
			}
		}
		if dmg < 0 {
			panic("monster damage: unreachable 0x542c75")
		}
	} else {
		if absorb != 0 {
			dmg = mdnDiv100((100-absorb)*dmg + 0x32)
		}
		if dmg < 0 {
			dmg = 0
		}
		if growth { // 0x553f10(record, 0x25, 1, 0): return and writes unread
			out[mdnOC553]++
			for i, v := range []int64{int64(uint32(rec)), 0x25, 1, 0} {
				out[mdnOA553+i] = v
			}
		}
	}

	out[mdnORet] = int64(dmg)
	out[mdnOGout] = count
	out[mdnOGpad0], out[mdnOGpad1] = 0xa5a5a5a5, 0x5a5a5a5a
	return out
}
