package calc

import "fmt"

// all_effect_remove 0x524540 (build/wiz8.asm:396606..): for timed effect k =
// 0..7, first every party member j = 0..7 whose presence byte (record at
// *0x685178, stride 0x106, byte 0) is set and whose effect duration
// C+0xa6d+0xc*k (record at *0x685174, stride 0x1862) is non-zero gets the three
// u32 fields C+0xa65/+0xa69/+0xa6d of effect k zeroed, its primary effect
// C+0xb05 rewritten (highest index 7..1 with non-zero duration, else 0; scan
// starts at C+0xac1), the interface refresh 0x55ee30(j), the effect/ability
// panel refresh 0x562a50(0x200000), 0x562a50(0x8000) when *0x68ec78 == 7,
// and the status rebuild 0x50e650(j); k == 6 sets *0x6840bb = 1. Then every
// entry i of the monster list *0x683fad (0x4e5620, count 0x5e2c70) whose
// duration M+0xaf+0xc*k is non-zero is looked up again by its id M+0 through
// 0x4e5550 (first entry with that id), and that entry gets effect k's three
// u32 fields at M+0xa7 zeroed, 0x4acd80(M+0xc, k+0x10, 0), the monster
// modifier rebuild 0x50e8c0(id) and, for k == 6, 0x505760(M), which calls
// 0x5049c0(M, 1, 0) and 0x5049c0(M, 0, 0) (build/wiz8.asm:352017).
//
// Emu address-stubs the D4 formulas 0x50e650 and 0x5049c0 and the excluded
// presentation call 0x4acd80; their arguments and call counts are outputs.
// None of their results is read back here (the node rereads only durations,
// ids and the visual pointer), so F needs no Emu split. 0x50e8c0 runs natively
// in the emulator; like 005242b0, its monster modifier caches are outside this
// interface (this node never reads them).
// 0x55ee30: mode 6 sets bit 0x100 in *0x69c0f4+0xf8, mode 7 passes 1<<j to
// 0x562a50; 0x562a50 ORs into *0x68edcc+0xf4 only in mode 7 with a non-null
// object (build/wiz8.asm:478051, 482405).
// Fixture caps (re/emu/cases/00524540.json): monster count <= aerMonsters and
// list / combat UI pointers 0 or the fixture objects; anything else panics.
const (
	aerMembers   = 8
	aerEffects   = 8
	aerMonsters  = 4
	aerArena     = 0x20000000
	aerUI        = aerArena + 0xe000 // refresh object: +0xf4 combat, +0xf8 inventory
	aerList      = aerArena + 0xe800
	aerMonBase   = aerArena + 0x11000
	aerMonStride = 0x800
	aerIn0       = 5
	aerPerMember = 2 + 3*aerEffects
	aerListAt    = aerIn0 + aerMembers*aerPerMember
	aerMon0      = aerListAt + 2
	aerPerMon    = 2 + 3*aerEffects
	aerInLen     = aerMon0 + aerMonsters*aerPerMon
	aerIcons     = aerEffects * aerMonsters
	aerAware     = 2 * aerMonsters
)

func init() {
	in := []string{"interfaceMode", "combatUiPtr", "combatRefreshFlags", "inventoryRefreshFlags", "magicBuffRefreshPending"}
	var out []string
	fields := []string{"strength", "multiplier", "duration"}
	for j := 0; j < aerMembers; j++ {
		in = append(in, fmt.Sprintf("pres%d", j), fmt.Sprintf("primary%d", j))
		out = append(out, fmt.Sprintf("primary%d", j))
		for k := 0; k < aerEffects; k++ {
			for _, f := range fields {
				in = append(in, fmt.Sprintf("c%d_e%d_%s", j, k, f))
				out = append(out, fmt.Sprintf("c%d_e%d_%s", j, k, f))
			}
		}
	}
	in = append(in, "monsterListPtr", "monsterCount")
	for m := 0; m < aerMonsters; m++ {
		in = append(in, fmt.Sprintf("m%d_id", m), fmt.Sprintf("m%d_visual", m))
		for k := 0; k < aerEffects; k++ {
			for _, f := range fields {
				in = append(in, fmt.Sprintf("m%d_e%d_%s", m, k, f))
				out = append(out, fmt.Sprintf("m%d_e%d_%s", m, k, f))
			}
		}
	}
	out = append(out, "combatRefreshFlags", "inventoryRefreshFlags", "magicBuffRefreshPending", "rebuildCalls")
	for i := 0; i < aerMembers*aerEffects; i++ {
		out = append(out, fmt.Sprintf("rebuildWho%d", i))
	}
	out = append(out, "iconCalls")
	for a := 1; a <= 3; a++ {
		for i := 0; i < aerIcons; i++ {
			out = append(out, fmt.Sprintf("icon%d_arg%d", i, a))
		}
	}
	out = append(out, "awareCalls")
	for a := 1; a <= 3; a++ {
		for i := 0; i < aerAware; i++ {
			out = append(out, fmt.Sprintf("aware%d_arg%d", i, a))
		}
	}
	Nodes["00524540"] = Node{Addr: "00524540", In: in, Out: out, F: allEffectRemoveNode}
}

func allEffectRemoveNode(in []int64, r Rand) []int64 {
	if len(in) != aerInLen {
		panic("all_effect_remove: wrong input count")
	}
	for i, v := range in {
		limit := int64(0xffffffff)
		if i == 4 || (i >= aerIn0 && i < aerListAt && (i-aerIn0)%aerPerMember == 0) {
			limit = 0xff // dirty byte, presence bytes
		}
		if v < 0 || v > limit {
			panic("all_effect_remove: field out of range")
		}
	}
	in = append([]int64(nil), in...) // fields are cleared in place
	mode, ui, combat, inventory, dirty := in[0], in[1], in[2], in[3], in[4]
	listPtr, count := in[aerListAt], in[aerListAt+1]
	if ui != 0 && ui != aerUI {
		panic("all_effect_remove: combat UI pointer outside the fixture")
	}
	if listPtr != 0 && listPtr != aerList {
		panic("all_effect_remove: monster list pointer outside the fixture")
	}
	if listPtr == 0 {
		count = 0 // 0x5e2c70: null container has no entries
	}
	if count > aerMonsters {
		panic("all_effect_remove: monster count above fixture cap")
	}
	member := func(j int) []int64 { return in[aerIn0+j*aerPerMember : aerIn0+(j+1)*aerPerMember] }
	monster := func(m int) []int64 { return in[aerMon0+m*aerPerMon : aerMon0+(m+1)*aerPerMon] }
	refresh := func(bits int64) { // 0x562a50
		if mode == 7 && ui != 0 {
			combat |= bits
		}
	}
	var rebuild, icons, aware []int64
	for k := 0; k < aerEffects; k++ {
		for j := 0; j < aerMembers; j++ {
			c := member(j)
			if c[0] == 0 || c[2+3*k+2] == 0 {
				continue
			}
			c[2+3*k], c[3+3*k], c[4+3*k] = 0, 0, 0
			c[1] = aerPrimary(c[2:])
			switch mode { // 0x55ee30
			case 6:
				inventory |= 0x100
			case 7:
				refresh(int64(1) << uint(j))
			}
			if mode == 7 { // 0x5245bd
				refresh(0x200000)
				refresh(0x8000)
			}
			rebuild = append(rebuild, int64(j))
			if k == 6 {
				dirty = 1
			}
		}
		for i := 0; i < int(count); i++ {
			if monster(i)[2+3*k+2] == 0 {
				continue
			}
			t := 0 // 0x4e5550: first entry with this id (entry i itself at the latest)
			for monster(t)[0] != monster(i)[0] {
				t++
			}
			m := monster(t)
			m[2+3*k], m[3+3*k], m[4+3*k] = 0, 0, 0
			icons = append(icons, m[1], int64(k+0x10), 0)
			if k == 6 {
				p := int64(aerMonBase + t*aerMonStride)
				aware = append(aware, p, 1, 0, p, 0, 0)
			}
		}
	}
	var out []int64
	for j := 0; j < aerMembers; j++ {
		c := member(j)
		out = append(out, c[1])
		out = append(out, c[2:]...)
	}
	for m := 0; m < aerMonsters; m++ {
		out = append(out, monster(m)[2:]...)
	}
	out = append(out, combat, inventory, dirty, int64(len(rebuild)))
	out = append(out, aerPad(rebuild, aerMembers*aerEffects)...)
	out = append(out, aerArgs(icons, aerIcons)...)
	return append(out, aerArgs(aware, aerAware)...)
}

// aerPrimary is the scan at 0x524583..0x5245a8 over eight (strength,
// multiplier, duration) triples: the highest index 7..1 with a non-zero u32
// duration, else 0 (index 0 is written without regard to its duration).
func aerPrimary(e []int64) int64 {
	for i := 7; i >= 0; i-- {
		if e[3*i+2] != 0 || i == 0 {
			return int64(i)
		}
	}
	panic("all_effect_remove: 0x5245a6 unreachable (scan index below 0)")
}

func aerPad(v []int64, n int) []int64 {
	return append(append([]int64(nil), v...), make([]int64, n-len(v))...)
}

// aerArgs turns per-call (arg1, arg2, arg3) triples into the stub record
// layout: count, then all arg1 slots, all arg2 slots, all arg3 slots.
func aerArgs(calls []int64, n int) []int64 {
	out := []int64{int64(len(calls) / 3)}
	for a := 0; a < 3; a++ {
		col := make([]int64, n)
		for i := 0; i*3 < len(calls); i++ {
			col[i] = calls[3*i+a]
		}
		out = append(out, col...)
	}
	return out
}
