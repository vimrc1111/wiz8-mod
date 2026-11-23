package calc

import (
	"encoding/binary"
	"fmt"
)

// identifyItemCastNode is 0x5ba740 identify_item_cast(item): the party member
// in the global 0x69c0ec casts identify (spell 0x17) on an item.
//
// The caster must be 0..7 with a set presence byte (record at *0x685178,
// stride 0x106, byte 0); otherwise the original asserts (blocks 5ba753,
// 5ba773, 5ba7a3, cov-exempt) and calc panics. An item id (+0 dword) of -1
// returns at once with eax = *0x685178 (the slots pointer).
//
// 0x51f2b0 runs natively and is folded in: the item is identifiable when +6
// is 0, or when 0x51c4e0(id) (itemDefaultEquipSlotNode on the definition type
// at *0x68516c + id*269 + 0x3e and gameStarted 0x68517c) is not -1 and +0xb
// is 0. Not identifiable: 0x52e690(chars + caster*0x1862, [0x5ee65c], 0,
// [0x5ed8c8], [0x5ed914]) and return its eax; the caster global is kept.
//
// Identifiable: 0x536170 fills a 32-byte source block natively (zero, +0..+c
// = -1), then +0 = 7 and +0x1c = item; 0x5a6340(caster, 0x17, 8, &block).
// A result of 1 calls 0x5ba110(item, 0), then 0x52e690 as above when item+6
// is 0. Then 0x5a4a00(), ui (*0x69c0f4) +0xf8 |= 0x0fffffff, 0x5b59b0(0),
// caster global = -1, eax = 0x5b59b0's return.
//
// The three .data dwords passed to 0x52e690 are never written; they are read
// from the user's exe (a VA past the section's file data is zero-initialized).
//
// Emu (TestEmu) is the case: 0x5a6340 (D4) and the 排除 callees 0x52e690,
// 0x5ba110, 0x5a4a00, 0x5b59b0 are address-stubbed; their returns are case
// inputs (0x5ba110 returns 0, ignored) and the case records their non-pointer
// arguments and call counts. The source block's contents are not observable
// in the emulator (only its pointer is passed). F runs the real
// directSpellResolutionNode for 0x5a6340 on the source block above, the
// node's caster and item bytes, and the F-only inputs d.* (every other
// directSpellResolutionNode input, parked in a scratch object by the case);
// it ignores r5a6340. 0x5ba110 stays stubbed in F too (排除), so item+6 is
// the input value at the post-cast test.
//
// Fixture caps (calc panics past them): item id -1 or 0..7 (eight definition
// records).
const (
	iicArena    = 0x20000000
	iicSlots    = iicArena + 0x10000
	iicChars    = iicArena
	iicCharSize = 0x1862
	iicDefs     = 8
	iicSpell    = 0x17
	iicPL       = 8
	iicSrcType  = 7
	iicOwn      = 11                // caster pres id b6 bb dtype gs r5a6340 r52e690 r5b59b0 uif8
	iicDsrItem  = iicArena + 0x7000 // the item object of 005a6340.json (offset 28672)
)

var iicDsrPark []int // dsrIn positions of the F-only inputs, in node input order

func init() {
	in := []string{"caster", "pres", "id", "b6", "bb", "dtype", "gs", "r5a6340", "r52e690", "r5b59b0", "uif8"}
	for i, n := range dsrIn {
		switch {
		case n == "idx", n == "spell", n == "PL", n == "item.b6", n == "item.bb", len(n) > 5 && n[:5] == "src8.":
			continue
		}
		iicDsrPark = append(iicDsrPark, i)
		in = append(in, "d."+n)
	}
	Nodes["005ba740"] = Node{Addr: "005ba740", In: in,
		Out: []string{"eax", "caster.after", "uif8.after",
			"n5a6340", "c5a6340.a1", "c5a6340.a2", "c5a6340.a3",
			"n52e690", "c52e690.a1", "c52e690.a2", "c52e690.a3", "c52e690.a4", "c52e690.a5",
			"n5ba110", "c5ba110.a2", "n5a4a00", "n5b59b0", "c5b59b0.a1"},
		F:   func(in []int64, r Rand) []int64 { return identifyItemCast(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return identifyItemCast(in, r, false) }}
}

func identifyItemCastNode(in []int64, r Rand) []int64 { return identifyItemCast(in, r, true) }

func iicExeU32(va uint32) int64 {
	b := mstImage()
	off, err := peOffset(b, va)
	if err != nil {
		return 0 // past the section's file data: zero-initialized, never written
	}
	return int64(binary.LittleEndian.Uint32(b[off:]))
}

func identifyItemCast(in []int64, r Rand, real bool) []int64 {
	if len(in) != iicOwn+len(iicDsrPark) {
		panic("identify_item_cast: wrong input count")
	}
	caster, pres, id, b6, bb := in[0], in[1]&0xff, in[2], in[3]&0xff, in[4]&0xff
	if caster < 0 || caster > 7 || pres == 0 {
		panic(fmt.Sprintf("identify_item_cast: caster %d presence %d asserts", caster, pres))
	}
	out := make([]int64, 18)
	out[1], out[2] = caster, in[10]&0xffffffff
	if id == -1 {
		out[0] = iicSlots
		return out
	}
	if id < 0 || id >= iicDefs {
		panic(fmt.Sprintf("identify_item_cast: item id %d outside fixture 0..%d", id, iicDefs-1))
	}
	notify := func() int64 {
		out[7]++
		out[8], out[9], out[10] = iicChars+caster*iicCharSize, iicExeU32(0x5ee65c), 0
		out[11], out[12] = iicExeU32(0x5ed8c8), iicExeU32(0x5ed914)
		return in[8] & 0xffffffff
	}
	ok := b6 == 0 || (itemDefaultEquipSlotNode([]int64{id, in[5] & 0xff, in[6] & 0xff}, r)[0] != -1 && bb == 0)
	if !ok {
		out[0] = notify()
		return out
	}
	res := in[7] & 0xffffffff
	if real {
		v := make([]int64, len(dsrIn))
		for k, p := range iicDsrPark {
			v[p] = in[iicOwn+k]
		}
		for i, n := range dsrIn {
			switch n {
			case "idx":
				v[i] = caster
			case "spell":
				v[i] = iicSpell
			case "PL":
				v[i] = iicPL
			case "item.b6":
				v[i] = b6
			case "item.bb":
				v[i] = bb
			case "src8.type":
				v[i] = iicSrcType
			case "src8.char", "src8.mon", "src8.grp":
				v[i] = -1
			case "src8.x", "src8.y", "src8.z":
				v[i] = 0
			case "src8.d1c":
				v[i] = iicDsrItem
			}
		}
		res = directSpellResolutionNode(v, r)[0] & 0xffffffff
	}
	out[3], out[4], out[5], out[6] = 1, caster, iicSpell, iicPL
	if res == 1 {
		out[13], out[14] = 1, 0
		if b6 == 0 {
			notify()
		}
	}
	out[15]++
	out[2] |= 0x0fffffff
	out[16], out[17] = 1, 0
	out[0], out[1] = in[9]&0xffffffff, -1
	return out
}
