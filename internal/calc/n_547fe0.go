package calc

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
)

// battlePrayerNode is 0x547fe0 battle_prayer(idx) for party character idx
// (build/wiz8.asm:445702; notes/re/abilities.md:35).
//
//   - Prologue = prayerAvailableNode (0x547f40 checks, done inline here): the
//     class/race tables grant ability 11, *0x6836a8 != 0, the battle slot byte
//     G+0xb0+0xd4*idx is 0, and a list monster is present, active, hostile (1)
//     with nonzero HP. Otherwise 0 is returned with no writes.
//   - Empty weight cache (*0x68d84c == 0): cum[j] += w[u] for u = 0..13,
//     j = u..14, int32 wrapping; w is the 14-dword table 0x61dd38 (read from
//     the player's exe at run time). cum[14] is 0x68d84c.
//   - 0x53be00 builds the source {1, idx, -1, ...}; source bytes +0x18 and
//     +0x1a are set. 0x536170 initialises the target {0, -1, -1, -1, 0...}.
//   - cost = 0x52b2f0(idx, 6). Stamina st (C+0xb1d) < cost (signed): if
//     R(cost) > st the node returns cost without marking the slot.
//   - Quiet byte 0x68510c == 0: 0x5905c0(1, -1) twice (byte 0x69b7b8 = 1)
//     around message 0x619788.
//   - roll = R(cum[14]); G+4 < 4: roll += 15*G4 - 60; G+4 > 8: roll += R(10);
//     difficulty byte 0x6850b4: 0 -> -10, 2 -> +10. k = first j in 0..13 with
//     roll <= cum[j] (signed), none 0. P = u32 level word C+0x8d+4*class / 3 + 2.
//   - Results are tried from k-1 down to 0 (a failed result tries the next
//     lower one; -1 is the default message). Each result's own rules are in
//     bprRun. A finished result sets the slot byte to 1 and returns its number
//     (0xffffffff for the default).
//
// Stubs (case): 0x4fb4c0, 0x5246c0, 0x50e700, 0x52b2f0 and 0x547cb0 are D4
// formula callees, address-stubbed; 0x5905f0, 0x562a50, 0x58ac00 and
// 0x408860 are excluded engine flow. Their arguments and call counts are
// outputs. The returns of 0x4fb4c0, 0x5246c0 and 0x50e700 are ignored
// (0x5a5a5a5a in a scratch copy keeps diff 0). 0x52b2f0 and 0x547cb0 returns
// are case inputs (costRet, tuRet) in Emu; F calls
// characterActionFatigueCostNode and turnUndeadPowerNode (mode 0, no cost
// pointer) with F-only inputs instead. The spell effects of 0x4fb4c0 are not
// fed forward (its D4 model stubs its effect handlers).
//
// Message renderer absent (0x68edcc = 0), screen mode 0x68ec78 = 0 (0x5af2d0
// does nothing) and sound flag 0x650e50 = 0 (0x40a910 returns 0, so 0x408860
// is called) are fixed in the case.
//
// Fixture caps (calc panics past them): idx 0..7 (eight records), list count
// <= 8 (eight entries), current map >= -1 (0x686b7d + 33*map is written for
// maps below 47; the case maps -1..46), range class -1..3 (0x51a9a0 asserts
// otherwise), gender -1..1 (the 0x61e434 rows), stale ebx 0..7 when it is
// dereferenced (case 4 after case 5).
const (
	bprArena      = 0x20000000
	bprText       = bprArena + 0x11000 // case object "text": message k at +8k
	bprBuf        = 0x689fd0           // 0x517a90 output buffer
	bprA8         = 0x10007f78         // &local at entry esp - 0x88 (0x4fb4c0 arg 8)
	bprListPtr    = 0x10007f88         // &id list object (0x548b5b)
	bprPrelude    = 0x619788           // 0x5481c0
	bprCharStride = 0x1862             // character record size (0x547ff1)
	bprCastRecN   = 6
	bprMsgRecN    = 4
)

type bprTables struct {
	weights [14]int32
	gender  [3]uint16 // 0x61e434 + 8*g, g = -1..1
	format  []uint16  // 0x61a788 wide "%s %d %s."
}

var bprTab, bprTabErr = bprLoad()

func bprLoad() (bprTables, error) {
	var t bprTables
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("battle prayer: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	at := func(va uint32, n int) ([]byte, error) {
		o, err := peOffset(b, va)
		if err != nil {
			return nil, err
		}
		if len(b)-o < n {
			return nil, fmt.Errorf("battle prayer: table %#x truncated", va)
		}
		return b[o : o+n], nil
	}
	w, err := at(0x61dd38, 56)
	if err != nil {
		return t, err
	}
	for i := range t.weights {
		t.weights[i] = int32(binary.LittleEndian.Uint32(w[4*i:]))
	}
	for g := -1; g <= 1; g++ {
		x, err := at(uint32(0x61e434+8*g), 2)
		if err != nil {
			return t, err
		}
		t.gender[g+1] = binary.LittleEndian.Uint16(x)
	}
	for va := uint32(0x61a788); ; va += 2 {
		x, err := at(va, 2)
		if err != nil {
			return t, err
		}
		u := binary.LittleEndian.Uint16(x)
		if u == 0 {
			break
		}
		t.format = append(t.format, u)
	}
	return t, nil
}

var bprIn, bprOut = bprNames()

func bprNames() ([]string, []string) {
	in := []string{"idx", "battlePtr", "used", "cls", "race", "gender"}
	for k := 0; k < 16; k++ {
		in = append(in, fmt.Sprintf("lvl%d", k))
	}
	in = append(in, "fatigueStatus")
	for j := 0; j < 8; j++ {
		for _, f := range []string{"slot", "cond", "hpMax", "hp", "stMax", "st", "a2d", "a35", "a85", "name"} {
			in = append(in, fmt.Sprintf("%s%d", f, j))
		}
	}
	in = append(in, "g4", "g14")
	for k := 0; k < 18; k++ {
		in = append(in, fmt.Sprintf("eff%dFlag", k), fmt.Sprintf("eff%dId", k))
		if k < 9 {
			in = append(in, fmt.Sprintf("eff%dF5", k), fmt.Sprintf("eff%dFd", k))
		}
	}
	for k := 0; k < 12; k++ {
		in = append(in, fmt.Sprintf("glob%dFlag", k), fmt.Sprintf("glob%dId", k))
	}
	for j := 0; j < 14; j++ {
		in = append(in, fmt.Sprintf("cum%d", j))
	}
	in = append(in, "cumTotal", "difficulty", "quiet", "msgContinue", "stale84", "stale20", "gold", "curMap")
	for m := -1; m < 47; m++ {
		in = append(in, fmt.Sprintf("acc[%d]", m))
	}
	in = append(in, "monCount")
	for i := 0; i < 8; i++ {
		for _, f := range []string{"Id", "Present", "Active", "Disp", "HP", "F6f", "X", "Y", "Z", "R"} {
			in = append(in, fmt.Sprintf("mon%d%s", i, f))
		}
	}
	in = append(in, "partyX", "partyY", "partyZ", "partyR", "rc12", "rc6", "costRet", "tuRet")
	for _, n := range bprSaNames() {
		in = append(in, "sa_"+n)
	}

	out := []string{"result", "usedAfter", "g14"}
	for k := 0; k < 9; k++ {
		for _, f := range []string{"Flag", "Id", "F5", "Fd"} {
			out = append(out, fmt.Sprintf("eff%d%s", k, f))
		}
	}
	for j := 0; j < 14; j++ {
		out = append(out, fmt.Sprintf("cum%d", j))
	}
	out = append(out, "cumTotal", "msgContinue", "gold")
	for m := -1; m < 47; m++ {
		out = append(out, fmt.Sprintf("acc[%d]", m))
	}
	for k := 0; k < 8; k++ {
		out = append(out, fmt.Sprintf("buf%d", k))
	}
	for k := 0; k < 13; k++ {
		out = append(out, fmt.Sprintf("src%d", k))
	}
	for k := 0; k < 8; k++ {
		out = append(out, fmt.Sprintf("tgt%d", k))
	}
	out = append(out, "listCount", "list0", "s4fb4c0_calls")
	for _, a := range bprCastArgs {
		for k := 0; k < bprCastRecN; k++ {
			out = append(out, fmt.Sprintf("s4fb4c0_a%d_%d", a, k))
		}
	}
	out = append(out, "s5905f0_calls")
	for k := 0; k < bprMsgRecN; k++ {
		out = append(out, fmt.Sprintf("s5905f0_a1_%d", k))
	}
	out = append(out, "s52b2f0_calls", "s52b2f0_a1", "s52b2f0_a2", "s547cb0_calls", "s547cb0_a1", "s547cb0_a2", "s547cb0_a3", "s5246c0_calls")
	for k := 0; k < 18; k++ {
		out = append(out, fmt.Sprintf("s5246c0_a1_%d", k))
	}
	out = append(out, "s50e700_calls", "s562a50_calls")
	for k := 0; k < 9; k++ {
		out = append(out, fmt.Sprintf("s562a50_a1_%d", k))
	}
	out = append(out, "s58ac00_calls", "s58ac00_a1", "s58ac00_a2", "s408860_calls", "s58aad0_calls")
	for k := 0; k < 2; k++ {
		out = append(out, fmt.Sprintf("s58aad0_a1_%d", k), fmt.Sprintf("s58aad0_a2_%d", k), fmt.Sprintf("s58aad0_a3_%d", k))
	}
	out = append(out, "s58b410_calls")
	for k := 0; k < 2; k++ {
		out = append(out, fmt.Sprintf("s58b410_a2_%d", k), fmt.Sprintf("s58b410_a4_%d", k))
	}
	return in, out
}

// Recorded 0x4fb4c0 arguments (2 and 3 are the fixed source/target pointers).
var bprCastArgs = []int{1, 4, 5, 6, 7, 8, 9, 10, 11}

// sa_* inputs feed spellAreaTargetSelectionNode inside turnUndeadPowerNode (F only).
func bprSaNames() []string {
	var s []string
	for _, n := range spellAreaInputs {
		if _, ok := tupFixed[n]; !ok && n != "sourceCharacter" {
			s = append(s, n)
		}
	}
	return s
}

type bprStop struct{}

func battlePrayerNode(in []int64, r Rand) []int64 { return bprRun(in, r, true) }

func bprRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(bprIn) {
		panic("battle prayer: input arity")
	}
	if bprTabErr != nil {
		panic(bprTabErr)
	}
	v := make(map[string]int64, len(in))
	for i, n := range bprIn {
		v[n] = in[i]
	}
	o := make([]int64, len(bprOut))
	pos := make(map[string]int, len(bprOut))
	for i, n := range bprOut {
		pos[n] = i
	}
	set := func(name string, x int64) {
		i, ok := pos[name]
		if !ok {
			panic("battle prayer: unknown output " + name)
		}
		o[i] = x
	}
	u32 := func(x int64) int64 { return int64(uint32(x)) }
	idx := v["idx"]
	if idx < 0 || idx > 7 {
		panic("battle prayer: character index past the eight fixture records")
	}
	n := v["monCount"]
	if n < 0 || n > 8 {
		panic("battle prayer: monster count past the eight-entry list")
	}

	// Pass-through state.
	set("usedAfter", v["used"])
	set("g14", v["g14"])
	eff := make([][4]int64, 9)
	for k := range eff {
		eff[k] = [4]int64{v[fmt.Sprintf("eff%dFlag", k)], v[fmt.Sprintf("eff%dId", k)], v[fmt.Sprintf("eff%dF5", k)], v[fmt.Sprintf("eff%dFd", k)]}
	}
	cum := make([]int64, 15)
	for j := 0; j < 14; j++ {
		cum[j] = v[fmt.Sprintf("cum%d", j)]
	}
	cum[14] = v["cumTotal"]
	msgContinue := v["msgContinue"]
	gold := v["gold"]
	acc := make([]int64, 48)
	for m := range acc {
		acc[m] = v[fmt.Sprintf("acc[%d]", m-1)]
	}
	buf := make([]uint16, 16)
	src := []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	tgt := []int64{0, 0, 0, 0, 0, 0, 0, 0}
	listCount, list0 := int64(0), int64(0)
	casts := [][]int64{}
	msgs := []int64{}
	var cost52, tu547 []int64
	remove := []int64{}
	rebuilds, flagsSet := 0, 0
	var log58 []int64
	sounds := 0
	notices := 0 // 0x590950(idx, text) runs natively; its 排除 callees are stubbed

	flush := func(result int64) []int64 {
		set("result", u32(result))
		for k := range eff {
			for f, name := range []string{"Flag", "Id", "F5", "Fd"} {
				set(fmt.Sprintf("eff%d%s", k, name), eff[k][f])
			}
		}
		for j := 0; j < 14; j++ {
			set(fmt.Sprintf("cum%d", j), u32(cum[j]))
		}
		set("cumTotal", u32(cum[14]))
		set("msgContinue", msgContinue)
		set("gold", gold)
		for m := range acc {
			set(fmt.Sprintf("acc[%d]", m-1), acc[m])
		}
		for k := 0; k < 8; k++ {
			set(fmt.Sprintf("buf%d", k), int64(buf[2*k])|int64(buf[2*k+1])<<16)
		}
		for k, x := range src {
			set(fmt.Sprintf("src%d", k), x)
		}
		for k, x := range tgt {
			set(fmt.Sprintf("tgt%d", k), x)
		}
		set("listCount", listCount)
		set("list0", list0)
		set("s4fb4c0_calls", int64(len(casts)))
		for k, c := range casts {
			if k >= bprCastRecN {
				panic("battle prayer: more 0x4fb4c0 calls than the record slots")
			}
			for i, a := range bprCastArgs {
				set(fmt.Sprintf("s4fb4c0_a%d_%d", a, k), c[i])
			}
		}
		set("s5905f0_calls", int64(len(msgs)))
		for k, m := range msgs {
			if k >= bprMsgRecN {
				panic("battle prayer: more 0x5905f0 calls than the record slots")
			}
			set(fmt.Sprintf("s5905f0_a1_%d", k), m)
		}
		if cost52 != nil {
			set("s52b2f0_calls", 1)
			set("s52b2f0_a1", cost52[0])
			set("s52b2f0_a2", cost52[1])
		}
		if tu547 != nil {
			set("s547cb0_calls", 1)
			set("s547cb0_a1", tu547[0])
			set("s547cb0_a2", tu547[1])
			set("s547cb0_a3", tu547[2])
		}
		set("s5246c0_calls", int64(len(remove)))
		for k, x := range remove {
			set(fmt.Sprintf("s5246c0_a1_%d", k), x)
		}
		set("s50e700_calls", int64(rebuilds))
		set("s562a50_calls", int64(flagsSet))
		for k := 0; k < flagsSet; k++ {
			set(fmt.Sprintf("s562a50_a1_%d", k), 0x800100) // 0x5524cb
		}
		if log58 != nil {
			set("s58ac00_calls", 1)
			set("s58ac00_a1", log58[0])
			set("s58ac00_a2", log58[1])
		}
		set("s408860_calls", int64(sounds))
		set("s58aad0_calls", int64(notices))
		set("s58b410_calls", int64(notices))
		for k := 0; k < notices; k++ {
			// 0x5909de: 0x58aad0(8, 0x64bf5c, name C+5, ...); 0x590a27: 0x58b410(colour, 0, len, -1)
			set(fmt.Sprintf("s58aad0_a1_%d", k), 8)
			set(fmt.Sprintf("s58aad0_a2_%d", k), 0x64bf5c)
			set(fmt.Sprintf("s58aad0_a3_%d", k), bprArena+bprCharStride*idx+5)
			set(fmt.Sprintf("s58b410_a4_%d", k), 0xffffffff)
		}
		return o
	}

	// Prologue: 0x547f40's rule on the same record, battle slot and list.
	avail := []int64{idx, bprArena, v["cls"], v["race"], v["battlePtr"], v["used"], 1, n}
	for i := 0; i < 8; i++ {
		for _, f := range []string{"Present", "Active", "Disp", "HP"} {
			avail = append(avail, v[fmt.Sprintf("mon%d%s", i, f)])
		}
	}
	if prayerAvailableNode(avail, r)[0] == 0 {
		return flush(0)
	}

	if cum[14] == 0 {
		for u := 0; u < 14; u++ {
			for j := u; j < 15; j++ {
				cum[j] = int64(int32(cum[j]) + bprTab.weights[u])
			}
		}
	}
	src = []int64{1, idx, 0xffffffff, 0, 0, 0, 0x10001, 0, 0, 0, 0, 0, 0}
	tgtInit := func() { tgt = []int64{0, 0xffffffff, 0xffffffff, 0xffffffff, 0, 0, 0, 0} }
	tgtInit()

	ch := func(f string, j int64) int64 { return v[fmt.Sprintf("%s%d", f, j)] }
	notices++ // message *0x68c09c+0x5d0
	cost52 = []int64{idx, 6}
	cost := int64(int32(v["costRet"]))
	if real {
		cost = int64(int32(characterActionFatigueCostNode([]int64{idx, 6, 0, 0, 0, 0, 0, 0, -1, -1, -1, -1, 0, 0, 0, 0, u32(ch("stMax", idx)), v["fatigueStatus"]}, r)[0]))
	}
	st := int64(int32(ch("st", idx)))
	if st < cost {
		if int64(int32(r.R(int(uint32(cost))))) > st {
			notices++ // message +0x5d4
			return flush(cost)
		}
	}
	if v["quiet"] == 0 {
		msgContinue = 1
		msgs = append(msgs, bprPrelude)
	}
	roll := int32(r.R(int(uint32(cum[14]))))
	if g4 := uint32(v["g4"]); g4 < 4 {
		roll += 15*int32(g4) - 60
	} else if g4 > 8 {
		roll += int32(r.R(10))
	}
	switch v["difficulty"] {
	case 0:
		roll -= 10
	case 2:
		roll += 10
	}
	k := int64(0)
	for j := 0; j < 14; j++ {
		if roll <= int32(cum[j]) {
			k = int64(j)
			break
		}
	}
	cls := v["cls"]
	P := u32(u32(v[fmt.Sprintf("lvl%d", cls+1)])/3 + 2)
	Pstore := P
	msg := func(off int64) int64 { return bprText + 8*(off/4) }
	cast := func(id, power, a5, a6, a7, a11 int64) {
		casts = append(casts, []int64{id, u32(power), a5, a6, a7, bprA8, 0, 0, a11})
	}
	elig := func(j int64) bool {
		return ch("slot", j) != 0 && u32(ch("hp", j)) != 0 && u32(ch("cond", j)) < 0x12
	}
	mon := func(i int64, f string) int64 { return v[fmt.Sprintf("mon%d%s", i, f)] }
	monElig := func(i int64) bool {
		return mon(i, "Present") != 0 && mon(i, "Active") != 0 && mon(i, "Disp") == 1 && mon(i, "HP") != 0
	}
	inRange := func(i int64, class int64) bool {
		if monsterTargetInRangeDataErr != nil {
			panic(monsterTargetInRangeDataErr)
		}
		if class < -1 || class > 3 {
			panic("battle prayer: range class assertion in 0x51a9a0")
		}
		length := int64(-1)
		for x, d := range []uint32{0x51a9b7, 0x51a9bb, 0x51a9c2, 0x51a9c9, 0x51a9d0} {
			if monsterTargetInRangeData.lengthDest[class+1] == d {
				length = []int64{0, 2, 4, 25, 50}[x]
			}
		}
		if length < 0 {
			panic("battle prayer: unknown length dispatch")
		}
		ext, op := monsterAttackTargetExt, monsterAttackTargetOp
		lim, _ := op((*big.Float).Mul, new(big.Float).SetPrec(64).SetInt64(length), ext(monsterTargetInRangeData.lengthMul)).Float32()
		f := monsterAttackTargetFloat
		d := monsterAttackTargetDistance(
			[3]float64{f(mon(i, "X")), f(mon(i, "Y")), f(mon(i, "Z"))},
			[3]float64{f(v["partyX"]), f(v["partyY"]), f(v["partyZ"])},
			true, f(mon(i, "R")), f(v["partyR"]))
		// fcomp + test ah,0x41: within unless the distance is greater.
		return ext(float64(lim)).Cmp(d) >= 0
	}
	vsw := func(format []uint16, args ...any) {
		w := []uint16{}
		for i := 0; i < len(format); i++ {
			if format[i] != '%' || i+1 == len(format) {
				w = append(w, format[i])
				continue
			}
			i++
			switch format[i] {
			case 's':
				w = append(w, args[0].([]uint16)...)
			case 'd':
				for _, c := range fmt.Sprint(args[0].(int64)) {
					w = append(w, uint16(c))
				}
			default:
				panic("battle prayer: unexpected format conversion")
			}
			args = args[1:]
		}
		w = append(w, 0)
		if len(w) > len(buf) {
			panic("battle prayer: formatted text past the eight compared dwords")
		}
		copy(buf, w)
	}
	text := func(slot int64) []uint16 { return bprMsgText(slot) }
	cstr := func(j int64) []uint16 { // name: u16 at C+5, NUL at C+7 (case)
		if u := uint16(ch("name", j)); u != 0 {
			return []uint16{u}
		}
		return nil
	}

	local84 := u32(v["stale84"])
	slot20 := v["stale20"]
	bVar1, bVar2 := false, false
	res := k
	for {
		res--
		if res < 0 {
			msgs = append(msgs, msg(0x5d8))
			res = 0xffffffff
			break
		}
		done, brk := false, true
		switch res {
		case 12:
			brk = false
			P >>= 1
			Pstore = P
			bVar1 = false
			local84 = 0
			for i := int64(0); i < n; i++ {
				if monElig(i) && inRange(i, int64(int32(v["rc12"]))) {
					msgs = append(msgs, msg(0x5e4))
					local84 = 1
					tgtInit()
					cast(0x60, P, 0, 0, 0, 0)
					break
				}
			}
		case 11:
			local84 = 0
			for i := int64(0); i < n && !done; i++ {
				if monElig(i) && u32(mon(i, "F6f")) == 0 {
					g := v["gender"]
					if g < -1 || g > 1 {
						panic("battle prayer: gender past the 0x61e434 rows")
					}
					vsw(text(0x5f0/4), text(int64(bprTab.gender[g+1])))
					msgs = append(msgs, bprBuf)
					tgt[0] = 5
					cast(0x75, P, 0, 0, 0, 0)
					done = true
				}
			}
		case 10:
			for _, id := range []int64{0x28, 0x14, 0x20, 0x1a} {
				if !bprGlobHas(v, id) {
					tgt[0], tgt[1] = 2, 0xffffffff
					msgs = append(msgs, msg(0x5ec))
					for _, c := range []int64{0x28, 0x14, 0x20, 0x1a} {
						cast(c, P, 0, 0, 0, 0)
					}
					done = true
					break
				}
			}
		case 9:
			if !bprEffHas(v, eff, 0x3b) || !bprEffHas(v, eff, 0x35) {
				tgt[0], tgt[1] = 2, 0xffffffff
				msgs = append(msgs, msg(0x5ec))
				cast(0x3b, P, 0, 0, 0, 0)
				cast(0x35, P, 0, 0, 0, 0)
				done = true
			}
		case 8:
			msgs = append(msgs, msg(0x5e4))
			for c := int64(0); c < 18; c++ {
				remove = append(remove, c)
			}
			for k := range eff {
				if eff[k][0] != 0 {
					eff[k] = [4]int64{}
					rebuilds++
					flagsSet++
				}
			}
			done = true
		case 7:
			tu547 = []int64{idx, 0, 0}
			ret := int64(int32(v["tuRet"]))
			if real {
				ret = bprTurnUndead(v, r)
				if ret > 0 {
					msgContinue = 1
				}
			}
			bVar1 = ret > 0
		case 6:
			local84 = 0
			for i := int64(0); i < n; i++ {
				if monElig(i) && inRange(i, int64(int32(v["rc6"]))) {
					local84++
				}
			}
			if local84 != 0 {
				e := int64(r.R(int(local84))) + 1
				for i := int64(0); i < n && !done; i++ {
					if monElig(i) && inRange(i, int64(int32(v["rc6"]))) {
						e--
						if e == 0 {
							msgs = append(msgs, msg(0x5e4))
							tgt[0], tgt[2] = 3, u32(mon(i, "Id"))
							if Pstore >= 7 {
								Pstore = 7
							}
							if listCount+1 > 5 { // capacity 5 from 0x548023; growth 0x548af9..0x548b3f is unreachable
								panic("battle prayer: id list growth (unreachable)")
							}
							listCount, list0 = 1, u32(mon(i, "Id"))
							cast(0x19, Pstore, 0, 0, 0, bprListPtr)
							done = true
						}
					}
				}
				if !done {
					panic("battle prayer: case 6 walk missed a counted monster (0x548aad, unreachable)")
				}
			}
		case 5: // records G+0x85a: eff 9..14
			found := false
			for c := 9; c < 15 && !found; c++ {
				found = v[fmt.Sprintf("eff%dFlag", c)] != 0 && u32(v[fmt.Sprintf("eff%dId", c)]) == 2
			}
			switch {
			case found:
				bVar2 = true
			case bVar2:
				panic("battle prayer: case 5 entered with the flag set (0x548496, unreachable)")
			default:
				msgs = append(msgs, msg(0x5ec))
				cast(2, P, 0, 0, 0, 0)
				done = true
			}
		case 4:
			if P > 7 {
				if P&^1 < 14 {
					P >>= 1
				} else {
					P = 7
				}
				tgtInit()
				tgt[0] = 2
				msgs = append(msgs, msg(0x5e4))
				cast(0x44, P, 0, 0, 1, 0)
				done = true
				break
			}
			for j := int64(0); j < 8; j++ {
				hp := u32(ch("hp", j))
				if !elig(j) || hp >= u32(ch("hpMax", j)) {
					continue
				}
				if bVar2 {
					if slot20 < 0 || slot20 > 7 {
						panic("battle prayer: stale ebx past the eight character records")
					}
					if hp < u32(ch("hp", slot20)) {
						slot20 = j
					}
				} else {
					slot20, bVar2 = j, true
				}
			}
			if bVar2 {
				tgtInit()
				tgt[0], tgt[1] = 1, u32(slot20)
				msgs = append(msgs, msg(0x5e4))
				cast(6, Pstore, 0, 0, 0, 0)
				done = true
			} else {
				P = Pstore
			}
		case 3:
			bVar2 = false
			local84 = 0
			for j := int64(0); j < 8; j++ {
				if elig(j) && (u32(ch("a2d", j)) != 0 || u32(ch("a35", j)) != 0) {
					local84++
					bVar2 = true
					P = Pstore
				}
			}
			if bVar2 {
				e := u32(int64(r.R(int(local84))) + 1)
				slot20 = e
				for j := int64(0); j < 8 && !done; j++ {
					if elig(j) && (u32(ch("a2d", j)) != 0 || u32(ch("a35", j)) != 0) {
						e--
						slot20 = e
						if e == 0 {
							msgs = append(msgs, msg(0x5e4))
							tgt[0], tgt[1] = 1, j
							if P > 7 {
								cast(0x4a, 7, 0, 0, 1, 0)
							} else {
								cast(0x4a, P, 0, 0, 0, 0)
							}
							done = true
						}
					}
				}
				if !done {
					panic("battle prayer: case 3 walk missed a counted character (0x548464, unreachable)")
				}
			}
		case 2:
			bVar2 = false
			for j := int64(0); j < 8; j++ {
				if elig(j) && u32(ch("a85", j)) == 0 {
					local84 = u32(local84 + 1)
					bVar2 = true
					P = Pstore
				}
			}
			if bVar2 {
				e := u32(int64(r.R(int(local84))) + 1)
				slot20 = e
				for j := int64(0); j < 8 && !done; j++ {
					if elig(j) && u32(ch("a85", j)) == 0 {
						e = u32(e - 1)
						slot20 = e
						if e == 0 {
							vsw(text(0x5e8/4), cstr(j))
							msgs = append(msgs, bprBuf)
							tgt[0], tgt[1] = 1, j
							if P >= 7 {
								P = 7
							}
							cast(0x15, P, 0, 0, 0, 0)
							done = true
						}
					}
				}
			}
		case 1:
			bVar2 = false
			for j := int64(0); j < 8; j++ {
				if ch("slot", j) != 0 && u32(ch("hp", j)) != 0 && int32(ch("st", j)) < int32(ch("stMax", j)) {
					msgs = append(msgs, msg(0x5e4))
					cast(0x2c, 7, 0, 0, 1, 0)
					done = true
					break
				}
			}
		case 0:
			if r.R(2) != 0 {
				msgs = append(msgs, msg(0x5dc))
				// 0x51bea0(100, 1): gold, per-map gold, log line, sound.
				gold = u32(gold + 100)
				m := int64(int32(v["curMap"]))
				if m < -1 {
					panic("battle prayer: current map below -1 writes outside the case")
				}
				if m < 0x2f {
					acc[m+1] = u32(acc[m+1] + 100)
				}
				vsw(bprTab.format, text(0x57c/4), int64(100), text(0x580/4))
				log58 = []int64{8, bprBuf}
				sounds = 1
			} else {
				msgs = append(msgs, msg(0x5e0))
				set("g14", u32(v["g14"]+10))
			}
			done = true
		}
		if done {
			break
		}
		if brk && bVar1 {
			break
		}
	}
	_ = local84
	set("usedAfter", 1)
	return flush(cost) // 0x548db2: eax is the cost local, not the result number
}

// Message k of the case: one UTF-16 unit 0x100+k, except the "%s" formats
// 0x5e8/4 and 0x5f0/4 (case setup).
func bprMsgText(k int64) []uint16 {
	if k == 0x5e8/4 || k == 0x5f0/4 {
		return []uint16{'%', 's'}
	}
	return []uint16{uint16(0x100 + k)}
}

// 0x501250: nine combat-effect records G+0x7c1, then nine at G+0x85a.
func bprEffHas(v map[string]int64, eff [][4]int64, id int64) bool {
	for k := 0; k < 18; k++ {
		flag, x := v[fmt.Sprintf("eff%dFlag", k)], v[fmt.Sprintf("eff%dId", k)]
		if k < 9 {
			flag, x = eff[k][0], eff[k][1]
		}
		if flag != 0 && uint32(x) == uint32(id) {
			return true
		}
	}
	return false
}

// 0x5012b0: twelve shared-effect records 0x68691f / 0x686920, stride 0x11.
func bprGlobHas(v map[string]int64, id int64) bool {
	for k := 0; k < 12; k++ {
		if v[fmt.Sprintf("glob%dFlag", k)] != 0 && uint32(v[fmt.Sprintf("glob%dId", k)]) == uint32(id) {
			return true
		}
	}
	return false
}

// 0x547cb0(idx, 0, 0) through its D4 model; levels C+0xb5 / C+0xbd are
// level words 11 and 13 (C+0x89+4k).
func bprTurnUndead(v map[string]int64, r Rand) int64 {
	x := map[string]int64{"idx": v["idx"], "levelB5": int64(int32(v["lvl11"])), "levelBD": int64(int32(v["lvl13"])),
		"class": v["cls"], "race": v["race"]}
	for _, n := range bprSaNames() {
		x["sa_"+n] = v["sa_"+n]
	}
	child := make([]int64, len(tupIn))
	for i, n := range tupIn {
		child[i] = x[n]
	}
	return int64(int32(turnUndeadPowerNode(child, r)[0]))
}

func init() {
	Nodes["00547fe0"] = Node{Addr: "00547fe0", In: bprIn, Out: bprOut,
		F:   battlePrayerNode,
		Emu: func(in []int64, r Rand) []int64 { return bprRun(in, r, false) }}
}
