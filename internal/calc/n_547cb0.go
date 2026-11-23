package calc

import "fmt"

// turnUndeadPowerNode is 0x547cb0 turn_undead(idx, costOut, mode) for party
// character idx (build/wiz8.asm:445259..445480).
//
//   - Two empty id lists (characters, monsters) get 20-byte operator new
//     buffers (0x5e1ce0), vtable 0x5ebfe0; both are freed on every return.
//   - mode low byte nonzero: 0x547bf0(idx) must return al != 0, else a
//     message is shown (0x590950) and 0 is returned. Otherwise the battle
//     slot byte *0x6836a8 + 0xd4*idx + 0xb1 (turn undead used) is set to 1.
//   - 0x53be00 builds the character source {1, idx, -1, zeros}; the node sets
//     source byte +0x1a, and +0x18 when the mode low byte is zero.
//   - Power P (int32, wrapping) is the class level words C+0xb5 + C+0xbd of
//     the character record *0x685174 + 0x1862*idx, plus 10 when the mode low
//     byte is zero (a message 0x590950 is shown otherwise).
//   - 0x536170 initialises the target {0, -1, -1, -1, zero coords, 0}.
//     0x4fd030(0x81, 1, &src, &tgt, &monsters, &characters, 0) fills both
//     lists and the target coordinates.
//   - No monsters: return 0. Otherwise 0x5905c0(1, -1) sets byte 0x69b7b8.
//     q = P/N (idiv). q == 0: while N > P, R(N) picks an entry, later entries
//     shift down one (the last slot keeps its stale id) and N drops; power
//     level 1. Otherwise power level is q when unsigned q < 7, else 7.
//   - 0x4fb4c0(0x81, &src, &tgt, level, 0, 0, 0, 0, 0, 0, &monsters); its
//     return is ignored (overwritten before use; ret 0x5a5a5a5a in a scratch
//     copy keeps the diff 0).
//   - costOut nonzero: *costOut = 0x52b2f0(idx, 3). Returns the final N.
//
// Stubs (case): 0x547bf0, 0x4fd030, 0x4fb4c0 and 0x52b2f0 are D4 formula
// callees, address-stubbed, each called at most once; their non-pointer
// arguments and call counts are outputs. 0x53be00, 0x536170, 0x590950,
// 0x5905c0, 0x5e1ce0 and 0x5e1c10 run natively (message renderer absent).
//
// Emu (what TestEmu replays) takes availRet, monCount/mon0..8, the target
// coordinates and costRet from the stub inputs. F runs turnUndeadAvailableNode,
// spellAreaTargetSelectionNode and characterActionFatigueCostNode instead,
// with the call's own values plus F-only world inputs (tu_*, sa_*, class,
// race, maxFatigue, fatigueStatus) parked where the original does not read
// them. Additional co-located monster records share sa_* scalar traits,
// with IDs 123+i; sa_extraMonsterCount defaults to zero for older callers.
// List/group maximum: [not obtained]; see notes/re/iface/547cb0.tsv.
// The fallback of nine covers the known normal-play count above eight.
//
// P in (-N, 0) never ends in the original (N > P forever once N reaches 0):
// calc panics there. Fixture caps (calc panics past them): idx 0..7 (eight
// records), monster count <= 9 (nine-entry list object), costOut 0 or the
// case object.
const (
	tupArena     = 0x20000000
	tupChars     = tupArena           // case object "chars" (*0x685174)
	tupBattle    = tupArena + 0x11000 // case object "G" (*0x6836a8)
	tupCostOut   = tupArena + 0x14400 // case object "costOut"
	tupSpell     = 0x81               // 0x547e1d, 0x547ee5
	tupMaxIDs    = 9                  // Reachable fallback; original group maximum not obtained.
	tupListSlots = 9                  // Schema stays fixed when testing the runtime limit.
	tupMaxLvl    = 7                  // 0x547ebf..0x547ec4
)

var tupFixed = map[string]int64{
	"spellID": tupSpell, "powerLevel": 1, // 0x4fd030 args
	"sourceKind": 1, "sourceMonster": -1, "sourceSpecial": 0, "sourceEnvironmental": 0, "sourceUseCenter": 0, // 0x53be00
	"targetKind": 0, "targetCharacter": -1, "targetMonster": -1, "targetGroup": -1, // 0x536170
	"targetXBits": 0, "targetYBits": 0, "targetZBits": 0, "ignoreLineOfSight": 0,
	"battleObject":        1, // *0x6836a8 is dereferenced by the node itself
	"monsterInitialCount": 0, "characterInitialCount": 0,
	"monsterCapacity": 5, "characterCapacity": 5, // 0x5e1ce0(0x14) buffers
}

var tupIn, tupOut = tupNames()

func tupNames() ([]string, []string) {
	in := []string{"idx", "costOut", "mode", "availRet", "usedBefore", "levelB5", "levelBD", "monCount"}
	for k := 0; k < 8; k++ {
		in = append(in, fmt.Sprintf("mon%d", k))
	}
	in = append(in, "costRet", "costBefore", "messageContinueBefore", "areaTargetXBits", "areaTargetYBits", "areaTargetZBits",
		"class", "race", "tu_monsterListPointer", "tu_monsterCount")
	for i := 0; i < 8; i++ {
		for _, n := range []string{"monsterPresent", "monsterActive", "monsterDisposition", "monsterHP", "templateIndex"} {
			in = append(in, fmt.Sprintf("tu_%s[%d]", n, i))
		}
	}
	for i := 0; i < 3; i++ {
		in = append(in, fmt.Sprintf("tu_templateType[%d]", i))
	}
	in = append(in, "maxFatigue", "fatigueStatus")
	for _, n := range spellAreaInputs {
		if _, ok := tupFixed[n]; !ok && n != "sourceCharacter" {
			in = append(in, "sa_"+n)
		}
	}
	out := []string{"result", "cost", "usedAfter", "messageContinue"}
	for k := 0; k < 8; k++ {
		out = append(out, fmt.Sprintf("list%d", k))
	}
	out = append(out, "s547bf0_calls", "s547bf0_a1", "s4fd030_calls", "s4fd030_a1", "s4fd030_a2", "s4fd030_a7", "s4fb4c0_calls")
	for _, a := range []int{1, 4, 5, 6, 7, 8, 9, 10} {
		out = append(out, fmt.Sprintf("s4fb4c0_a%d", a))
	}
	out = append(out, "s52b2f0_calls", "s52b2f0_a1", "s52b2f0_a2")
	for k := 0; k < 13; k++ {
		out = append(out, fmt.Sprintf("src%d", k))
	}
	for k := 0; k < 8; k++ {
		out = append(out, fmt.Sprintf("tgt%d", k))
	}
	// Append slots to preserve the positional inputs used by parked callers.
	for k := 8; k < tupListSlots; k++ {
		in = append(in, fmt.Sprintf("mon%d", k))
		out = append(out, fmt.Sprintf("list%d", k))
	}
	in = append(in, "sa_extraMonsterCount")
	return in, out
}

func turnUndeadPowerNode(in []int64, r Rand) []int64 { return tupRun(in, r, true) }

func tupRun(in []int64, r Rand, real bool) []int64 {
	// Existing parent fixtures describe one monster and eight backing slots.
	if len(in) == 143 {
		in = append(append([]int64(nil), in...), make([]int64, len(tupIn)-143)...)
	}
	if len(in) != len(tupIn) {
		panic("turn undead power: input arity")
	}
	v := make(map[string]int64, len(in))
	for i, n := range tupIn {
		v[n] = in[i]
	}
	idx := v["idx"]
	if idx < 0 || idx > 7 {
		panic("turn undead power: character index past the eight fixture records")
	}
	if v["costOut"] != 0 && v["costOut"] != tupCostOut {
		panic("turn undead power: cost pointer is neither null nor the case object")
	}
	o := make([]int64, len(tupOut))
	set := func(name string, x int64) {
		for i, n := range tupOut {
			if n == name {
				o[i] = x
				return
			}
		}
		panic("turn undead power: unknown output " + name)
	}
	list := make([]int64, tupListSlots)
	for k := range list {
		list[k] = v[fmt.Sprintf("mon%d", k)]
	}
	flush := func(result int64) []int64 {
		for k, id := range list {
			set(fmt.Sprintf("list%d", k), id)
		}
		set("result", result)
		return o
	}
	set("cost", v["costBefore"])
	set("usedAfter", v["usedBefore"])
	set("messageContinue", v["messageContinueBefore"])
	mode := v["mode"] & 0xff
	if mode != 0 {
		set("s547bf0_calls", 1)
		set("s547bf0_a1", idx)
		avail := v["availRet"] & 0xff
		if real {
			child := []int64{idx, tupChars, v["class"], v["race"], tupBattle, v["usedBefore"], v["tu_monsterListPointer"], v["tu_monsterCount"]}
			for i := 0; i < 8; i++ {
				for _, n := range []string{"monsterPresent", "monsterActive", "monsterDisposition", "monsterHP", "templateIndex"} {
					child = append(child, v[fmt.Sprintf("tu_%s[%d]", n, i)])
				}
			}
			for i := 0; i < 3; i++ {
				child = append(child, v[fmt.Sprintf("tu_templateType[%d]", i)])
			}
			avail = turnUndeadAvailableNode(child, r)[0]
		}
		if avail == 0 {
			return flush(0)
		}
		set("usedAfter", 1)
	}
	src := []int64{1, idx, 0xffffffff, 0, 0, 0, 0x10000, 0, 0, 0, 0, 0, 0}
	power := int64(int32(v["levelB5"] + v["levelBD"]))
	if mode == 0 {
		src[6] |= 1
		power = int64(int32(power + 10))
	}
	for k, x := range src {
		set(fmt.Sprintf("src%d", k), x)
	}
	tgt := []int64{0, 0xffffffff, 0xffffffff, 0xffffffff, v["areaTargetXBits"], v["areaTargetYBits"], v["areaTargetZBits"], 0}
	set("s4fd030_calls", 1)
	set("s4fd030_a1", tupSpell)
	set("s4fd030_a2", 1)
	n := v["monCount"]
	if real {
		child := make([]int64, len(spellAreaInputs))
		for i, name := range spellAreaInputs {
			if x, ok := tupFixed[name]; ok {
				child[i] = x
			} else if name == "sourceCharacter" {
				child[i] = idx
			} else {
				child[i] = v["sa_"+name]
			}
		}
		a := spellAreaTargetSelectionNode(child, r)
		n = a[0]
		// Each additional world record shares the scalar monster's geometry
		// and traits, but has a distinct ID. Selection is independent per
		// record because the child always starts with empty target lists.
		// This extends the one-record world without using the stub monCount.
		extra := v["sa_extraMonsterCount"]
		if extra < 0 || extra >= tupListSlots {
			panic("turn undead power: extra world monsters past the fixture")
		}
		n *= 1 + extra
		if n > tupMaxIDs {
			panic("turn undead power: area selection past the monster list limit")
		}
		for i := int64(0); i < n; i++ {
			list[i] = a[2] + i
		}
		copy(tgt[4:7], a[18:21])
	}
	if n < 0 || n > tupMaxIDs {
		panic("turn undead power: monster count past the monster list limit")
	}
	for k, x := range tgt {
		set(fmt.Sprintf("tgt%d", k), x)
	}
	if n == 0 {
		return flush(0)
	}
	set("messageContinue", 1)
	level := power / n
	if level == 0 {
		if power < 0 {
			panic("turn undead power: negative power above -count loops forever in the original")
		}
		for n > power {
			k := int64(r.R(int(n)))
			copy(list[k:n-1], list[k+1:n])
			n--
		}
		level = 1
	} else if uint32(level) >= tupMaxLvl {
		level = tupMaxLvl
	}
	set("s4fb4c0_calls", 1)
	set("s4fb4c0_a1", tupSpell)
	set("s4fb4c0_a4", level)
	if v["costOut"] != 0 {
		set("s52b2f0_calls", 1)
		set("s52b2f0_a1", idx)
		set("s52b2f0_a2", 3)
		cost := v["costRet"]
		if real {
			cost = characterActionFatigueCostNode([]int64{idx, 3, 0, 0, 0, 0, 0, 0, -1, -1, -1, -1, 0, 0, 0, 0, v["maxFatigue"], v["fatigueStatus"]}, r)[0]
		}
		set("cost", cost&0xffffffff)
	}
	return flush(n)
}

func init() {
	Nodes["00547cb0"] = Node{Addr: "00547cb0", In: tupIn, Out: tupOut,
		F:   turnUndeadPowerNode,
		Emu: func(in []int64, r Rand) []int64 { return tupRun(in, r, false) }}
}
