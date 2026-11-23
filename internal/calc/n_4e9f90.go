package calc

import (
	"encoding/binary"
	"fmt"
)

// combat_defeat_resolution 0x4e9f90(force) (build/wiz8.asm:313978..314170), cdecl, returns al.
//
// survivor := 0x4eeec0 (low byte); 0x546e70 recounts hostile (0x683fa5) / charmed
// (0x6850be) monsters and may set the opening-blocked byte cb+0xa54 (cb = *0x6836a8).
// No survivor (0x4ea161): 0x58ac00(0xc, text +0x8c0, 0, -1, 0); for each slot m with
// presence byte *0x685178+0x106*m set and (HP +0xb11 != 0 or main status +0xb01 < 0x12):
// 0x522fe0(m, 0x12, 9999, 0, 0, 1); then 0x4ea310(0), return 1.
// Survivor: unless (force byte != 0 and both counts 0), idle rounds byte cb+0xa56 < 2
// returns 0. Then kills = cb+0xc:
//   - kills != 0: 0x58ac00(0xc, text +0x8b0, 0, -1, 0); 0x58ab60(0xc, 0, text +0x8b4,
//     kills, kills == 1 ? 0x6174c0 "kill" : 0x6174b4 "kills"); 0x48fc10(0x6174a4
//     "CombatWin.MPL", 0, 1); 0x48f9e0; 0x42b770(1, 0); with byte 0x6850b4 set,
//     0x517fb0(0, 0, -1, &slot, 1, 0) picks one present standing member (as in
//     rsrVoiceRandom) and 0x52e690(record, [0x5ee614], [0x5ed8d8], [0x5ed8cc],
//     [0x5ed914]); a non-null event gets +0x30 := 1500 (0x4ea0be) and +0x34 :=
//     GetTickCount.
//   - kills == 0 and cb+0xa54 set: 0x42b770(1, 1).
//   - else 0x58ac00(0xc, text +0x8b8, 0, -1, 0) and, when 0x4ed550 (low byte) reports
//     pending effects, 0x58ac00(0xc, text +0x8bc, 0, -1, 0) and 0x42b770(1, 1).
//     All end with 0x4ea310(0) and return 1.
//
// Address stubs: D4 0x4eeec0, 0x546e70, 0x4ed550 (results read later: Emu replays the
// case inputs r4eeec0, w683fa5/w6850be/wa54, r4ed550; F runs partySurvivalCheckNode,
// hostileMonsterCountNode, combatPendingEffectsNode on the node's slots plus the F-only
// f*/m*/c*/b*/pm* inputs), D4 0x4ea310 and 0x522fe0 (nothing read back; the
// 0x5a5a5a5a return check showed eax ignored) and the 排除 helpers 0x58ac00, 0x58ab60,
// 0x48fc10, 0x42b770, 0x52e690 (return r52e690 is the event pointer or 0). Every call's
// arguments and count are outputs. GetTickCount returns the case input `tick`.
// 0x517fb0 and 0x48f9e0 run natively; 0x48f9e0 returns at once while the music flag
// byte 0x65ba7e is 0 (the case fixture: .data virtual tail).
const (
	cdresChars   = 0x20000000 // case object chars
	cdresChar    = 0x1862
	cdresEvent   = 0x20010000 // case object ev: the only non-null 0x52e690 return
	cdresText    = 0x7e000000 // case fixture: text pointer +x of *0x68c09c holds cdresText+x
	cdresDelay   = 1500       // 0x4ea0be
	cdresKill    = 0x6174c0   // 0x4ea008
	cdresKills   = 0x6174b4   // 0x4ea00f
	cdresWinSong = 0x6174a4   // 0x4ea02f
	cdresCond    = 0x12       // 0x4ea1b5
	cdresDur     = 9999       // 0x4ea1b0
)

var cdresStubs = []struct{ addr, args, n int }{
	{0x4eeec0, 0, 1}, {0x546e70, 0, 1}, {0x4ed550, 0, 1}, {0x4ea310, 1, 1}, {0x522fe0, 6, 8},
	{0x58ac00, 5, 2}, {0x58ab60, 5, 1}, {0x48fc10, 3, 1}, {0x42b770, 2, 1}, {0x52e690, 5, 1}}

func cdresNames() (in, out []string) {
	add := func(l *[]string, f string, a ...any) { *l = append(*l, fmt.Sprintf(f, a...)) }
	in = []string{"a", "r4eeec0", "w683fa5", "w6850be", "wa54", "a54", "a56", "kills", "g6850b4",
		"r4ed550", "r52e690", "tick", "ev30", "ev34"}
	for _, p := range []string{"pres", "hp", "b01_"} {
		for m := 0; m < 8; m++ {
			add(&in, "%s%d", p, m)
		}
	}
	in = append(in, "fMonList", "fMonCount", "fCombat", "fUi", "fCombatDirty", "fSceneDirty")
	for i := 0; i < 8; i++ {
		add(&in, "mPres%d", i)
		add(&in, "mAct%d", i)
		add(&in, "mDisp%d", i)
		add(&in, "mCharm%d", i)
	}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			add(&in, "cEff%d_%d", i, j)
		}
	}
	for k := 0; k < 9; k++ {
		add(&in, "bA%d", k)
	}
	for k := 0; k < 6; k++ {
		add(&in, "bB%d", k)
	}
	for i := 0; i < 8; i++ {
		add(&in, "pmPres%d", i)
		add(&in, "pmAct%d", i)
		for j := 0; j < 8; j++ {
			add(&in, "pmEff%d_%d", i, j)
		}
		for k := 0; k < 9; k++ {
			add(&in, "pmA%d_%d", i, k)
		}
		for k := 0; k < 6; k++ {
			add(&in, "pmB%d_%d", i, k)
		}
		add(&in, "pmHP%d", i)
		add(&in, "pm9f_%d", i)
		add(&in, "pm2da_%d", i)
	}
	out = []string{"eax", "ev30", "ev34", "o683fa5", "o6850be", "oa54"}
	for _, s := range cdresStubs {
		add(&out, "n%x", s.addr)
		for a := 1; a <= s.args; a++ {
			if s.n == 1 {
				add(&out, "a%x_%d", s.addr, a)
				continue
			}
			for c := 0; c < s.n; c++ {
				add(&out, "a%x_%d_%d", s.addr, a, c)
			}
		}
	}
	return
}

func init() {
	in, out := cdresNames()
	Nodes["004e9f90"] = Node{Addr: "004e9f90", In: in, Out: out, F: combatDefeatResolutionNode,
		Emu: func(in []int64, r Rand) []int64 { return cdresRun(in, r, false) }}
}

func combatDefeatResolutionNode(in []int64, r Rand) []int64 { return cdresRun(in, r, true) }

// cdresExe32 reads a dword the function loads from the user's Wiz8.exe (.rdata voice ids).
func cdresExe32(va uint32) uint32 {
	b := mstImage()
	off, err := peOffset(b, va)
	if err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint32(b[off:])
}

func cdresRun(in []int64, r Rand, real bool) []int64 {
	names, outNames := cdresNames()
	if len(in) != len(names) {
		panic(fmt.Sprintf("combat_defeat_resolution: expected %d inputs, got %d", len(names), len(in)))
	}
	ix := make(map[string]int, len(names))
	for i, n := range names {
		ix[n] = i
	}
	u := func(n string) uint32 { return uint32(in[ix[n]]) }
	seg := func(from string, n int) []int64 { return in[ix[from] : ix[from]+n] }
	var pres, hp, b01 [8]uint32
	for m := 0; m < 8; m++ {
		pres[m], hp[m], b01[m] = u(fmt.Sprint("pres", m))&0xff, u(fmt.Sprint("hp", m)), u(fmt.Sprint("b01_", m))
	}
	type rec struct {
		n    int
		args [][]uint32
	}
	stub := map[int]*rec{}
	limit := map[int]int{}
	for _, s := range cdresStubs {
		stub[s.addr], limit[s.addr] = &rec{}, s.n
	}
	call := func(a int, args ...uint32) {
		s := stub[a]
		if s.n == limit[a] {
			panic(fmt.Sprintf("combat_defeat_resolution: stub %#x called more than %d times", a, limit[a]))
		}
		s.n++
		s.args = append(s.args, args)
	}
	ev30, ev34 := u("ev30"), u("ev34")
	hostile, charmed, a54 := uint32(0), uint32(0), u("a54")&0xff
	ret := uint32(1)

	call(0x4eeec0)
	survivor := u("r4eeec0")&0xff != 0
	if real {
		v := make([]int64, 0, 24)
		for m := 0; m < 8; m++ {
			v = append(v, int64(pres[m]), int64(hp[m]), int64(b01[m]))
		}
		survivor = partySurvivalCheckNode(v, r)[0]&0xff != 0
	}
	call(0x546e70)
	if real {
		v := []int64{in[ix["fMonList"]], in[ix["fMonCount"]], in[ix["fCombat"]], int64(a54),
			in[ix["fUi"]], in[ix["fCombatDirty"]], in[ix["fSceneDirty"]], 0, 0}
		v = append(v, seg("mPres0", 32)...)
		o := hostileMonsterCountNode(v, r)
		hostile, charmed, a54 = uint32(o[0]), uint32(o[1]), uint32(o[2])
	} else {
		hostile, charmed, a54 = u("w683fa5"), u("w6850be"), u("wa54")&0xff
	}

	switch {
	case !survivor:
		call(0x58ac00, 0xc, cdresText+0x8c0, 0, 0xffffffff, 0)
		for m := uint32(0); m < 8; m++ {
			if pres[m] != 0 && (hp[m] != 0 || b01[m] < cdresCond) {
				call(0x522fe0, m, cdresCond, cdresDur, 0, 0, 1)
			}
		}
		call(0x4ea310, 0)
	case !(u("a")&0xff != 0 && hostile == 0 && charmed == 0) && u("a56")&0xff < 2:
		ret = 0
	case u("kills") != 0:
		kills := u("kills")
		call(0x58ac00, 0xc, cdresText+0x8b0, 0, 0xffffffff, 0)
		word := uint32(cdresKills)
		if kills == 1 {
			word = cdresKill
		}
		call(0x58ab60, 0xc, 0, cdresText+0x8b4, kills, word)
		call(0x48fc10, cdresWinSong, 0, 1)
		call(0x42b770, 1, 0)
		if u("g6850b4")&0xff != 0 {
			cdresVoice(pres, hp, b01, r, func(who uint32) {
				call(0x52e690, cdresChars+cdresChar*who, cdresExe32(0x5ee614), cdresExe32(0x5ed8d8), cdresExe32(0x5ed8cc), cdresExe32(0x5ed914))
				switch e := u("r52e690"); e {
				case 0:
				case cdresEvent:
					ev30, ev34 = cdresDelay, u("tick")
				default:
					panic("combat_defeat_resolution: 0x52e690 event pointer outside the case object")
				}
			})
		}
		call(0x4ea310, 0)
	case a54 != 0:
		call(0x42b770, 1, 1)
		call(0x4ea310, 0)
	default:
		call(0x58ac00, 0xc, cdresText+0x8b8, 0, 0xffffffff, 0)
		call(0x4ed550)
		pending := u("r4ed550")&0xff != 0
		if real {
			v := []int64{in[ix["fMonList"]], in[ix["fMonCount"]]}
			for m := 0; m < 8; m++ {
				v = append(v, int64(pres[m]))
				v = append(v, seg(fmt.Sprintf("cEff%d_0", m), 8)...)
			}
			v = append(v, seg("bA0", 15)...)
			v = append(v, seg("pmPres0", 8*28)...)
			pending = combatPendingEffectsNode(v, r)[0] != 0
		}
		if pending {
			call(0x58ac00, 0xc, cdresText+0x8bc, 0, 0xffffffff, 0)
			call(0x42b770, 1, 1)
		}
		call(0x4ea310, 0)
	}

	out := make([]int64, 0, len(outNames))
	out = append(out, int64(ret), int64(ev30), int64(ev34), int64(hostile), int64(charmed), int64(a54))
	for _, s := range cdresStubs {
		st := stub[s.addr]
		out = append(out, int64(st.n))
		for a := 0; a < s.args; a++ {
			if s.n == 1 {
				v := int64(0)
				if st.n > 0 {
					v = int64(st.args[st.n-1][a])
				}
				out = append(out, v)
				continue
			}
			for k := 0; k < s.n; k++ {
				v := int64(0)
				if k < st.n {
					v = int64(st.args[k][a])
				}
				out = append(out, v)
			}
		}
	}
	return out
}

// cdresVoice is 0x517fb0(0, 0, -1, &slot, 1, 0) and the caller's test of its count: the
// members with presence, HP != 0 and main status < 0x12 (first pass; the relaxed passes
// only run for argument 1, the single-pick stop otherwise); none -> no call, one -> it,
// several -> R(n) picks (0x51810a).
func cdresVoice(pres, hp, b01 [8]uint32, r Rand, speak func(who uint32)) {
	var list []uint32
	for i := uint32(0); i < 8; i++ {
		if pres[i] != 0 && hp[i] != 0 && b01[i] < cdresCond {
			list = append(list, i)
		}
	}
	switch len(list) {
	case 0:
	case 1:
		speak(list[0])
	default:
		speak(list[r.R(len(list))])
	}
}
