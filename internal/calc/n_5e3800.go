package calc

import "strconv"

// trapSpellTargetsNode is 0x5e3800 trap_spell_targets(x, y, z, spell, level,
// count), cdecl, called from 0x5e3ab0 (trap_power_target_count). Exe
// 0x5e3800..0x5e3aa6.
//
// Source record (0x536150, 13 dwords) = {kind 3, -1, -1, x, y, z, 0...}.
// Target record = party position *0x659c0c +0x100..+0x108 (0x4534c0 copy).
// Spell class (0x4ac9d0, flag 0) = spell record *0x65be1c + spell*0x1bf
// +0x137. Class 8: target kind 6 (only +0 and +0x10..+0x18 are written; the
// rest of that stack record is never set by the original and is the fixture
// stack's zero), cast with no target list. Otherwise target is the 0x536170
// record {0, -1, -1, -1, 0...} with kind 2 and the party position, and the
// id list {vt 0x5ebfe0, count, cap 5, data} gets every party slot k (0..7)
// with occupied byte *0x685178 + k*0x106 non-zero, current HP C+0xb11 non-zero
// and condition C+0xb01 < 0x12 (u32), growing by one entry (0x4addf0) past 5.
// count is clamped to the eligible number (signed; the arg slot is
// overwritten); then while the list is longer, R(len) picks an entry and,
// unless it equals the selected character dword 0x68518d, removes it by
// shifting the tail down. 0x4fb4c0 (D4) is an address stub: its return is
// case input castResult and nothing here reads it or its writes, so the
// call's arguments and count are outputs (no Emu/F split).
//
// Heap model (emu bump heap at 0x50000000, sizes rounded to 8, free no-op):
// the 5-entry block, then one block per grow; the 28 heap dwords are outputs.
// Fixture caps: spell id < 256 (spell table fixture rows); count <= 0 hits the
// srAssertFail at 0x5e3851 (notes/re/cov-exempt.d/5e3800.tsv) and panics.
const (
	tstSource = 0x10007fc0 // entry esp 0x10008000 - 0x40
	tstTarget = 0x10007fa0 // entry esp - 0x60
	tstList   = 0x10007f90 // entry esp - 0x70
	tstHeap   = 0x50000000
	tstVT     = 0x5ebfe0
)

func trapSpellTargetsNode(in []int64, r Rand) []int64 {
	if len(in) != 36 {
		panic("trap spell targets: expected 36 inputs")
	}
	if int32(in[5]) <= 0 {
		panic("trap spell targets: target count <= 0 asserts at 0x5e3851")
	}
	if uint32(in[3]) >= 256 {
		panic("trap spell targets: spell id outside the fixture spell table (fixture cap)")
	}
	out := make([]int64, 66)
	m1 := int64(0xffffffff)
	out[0], out[1], out[2], out[3], out[4] = 1, int64(uint32(in[3])), tstSource, tstTarget, int64(uint32(in[4]))
	out[12] = int64(int32(in[5]))
	src := out[13:26]
	src[0], src[1], src[2] = 3, m1, m1
	src[3], src[4], src[5] = int64(uint32(in[0])), int64(uint32(in[1])), int64(uint32(in[2]))
	tgt := out[26:34]
	tgt[4], tgt[5], tgt[6] = int64(uint32(in[7])), int64(uint32(in[8])), int64(uint32(in[9]))
	list := out[34:38]
	list[0], list[2], list[3] = tstVT, 5, tstHeap
	heap := out[38:66]
	if int32(in[6]) == 8 {
		tgt[0] = 6
		return out
	}
	tgt[0], tgt[1], tgt[2], tgt[3] = 2, m1, m1, m1
	out[10] = tstList

	word := func(a int64) *int64 { return &heap[(a-tstHeap)/4] }
	next := int64(tstHeap + 0x18)
	data, capa, n := int64(tstHeap), int64(5), int64(0)
	eligible := int64(0)
	for k := 0; k < 8; k++ {
		if uint8(in[11+k]) == 0 || uint32(in[19+k]) == 0 || uint32(in[27+k]) >= 0x12 {
			continue
		}
		if n+1 > capa { // 0x4addf0: new block of n+1 entries, copy, free old
			nd := next
			next += max(8, (4*(n+1)+7)&^7)
			for j := int64(0); j < n; j++ {
				*word(nd + 4*j) = *word(data + 4*j)
			}
			data, capa = nd, n+1
		}
		*word(data + 4*n) = int64(k)
		n++
		eligible++
	}
	keep := int64(int32(in[5]))
	if keep >= eligible {
		keep = eligible
		out[12] = eligible
	}
	sel := int64(uint32(in[10]))
	for n > keep {
		i := int64(r.R(int(n)))
		if i < 0 || i >= n {
			panic("trap spell targets: R(n) outside 0..n-1 (block 0x5e3a09, cov-exempt)")
		}
		if *word(data + 4*i) == sel {
			continue
		}
		for j := i; j < n-1; j++ {
			*word(data + 4*j) = *word(data + 4*j + 4)
		}
		n--
	}
	list[1], list[2], list[3] = n, capa, data
	return out
}

func init() {
	in := []string{"posXBits", "posYBits", "posZBits", "spell", "level", "targetCount", "spellClass", "partyXBits", "partyYBits", "partyZBits", "selectedCharacter"}
	for _, f := range []string{"slotOccupied", "currentHP", "condition"} {
		for k := 0; k < 8; k++ {
			in = append(in, f+string(rune('0'+k)))
		}
	}
	in = append(in, "castResult")
	out := []string{"cast.count", "cast.spell", "cast.source", "cast.target", "cast.level", "cast.bonus", "cast.fail", "cast.keep", "cast.out", "cast.mode", "cast.list2", "cast.list1", "targetCountArg"}
	for k := 0; k < 13; k++ {
		out = append(out, "source."+strconv.Itoa(k))
	}
	for k := 0; k < 8; k++ {
		out = append(out, "target."+strconv.Itoa(k))
	}
	out = append(out, "list.vt", "list.count", "list.cap", "list.data")
	for k := 0; k < 28; k++ {
		out = append(out, "heap"+strconv.Itoa(k))
	}
	Nodes["005e3800"] = Node{Addr: "005e3800", In: in, Out: out, F: trapSpellTargetsNode}
}
