package calc

import "fmt"

// item_use_consumption 0x51e760 (arg0 owner, arg1 item instance, arg2 force):
// spend one unit or one charge of an item instance and, when it runs out,
// remove it (build/wiz8.asm:388177..388345).
//   - item record +0x66 = 1: consume the count byte at instance +4;
//     otherwise the charge byte at instance +5 (record +0x66 = 0 only counts
//     when record +0x3e = 0x16).
//   - exhausted: the held-item instance 0x6874cb is cleared by 0x55f1e0 (run
//     natively); any other instance is zeroed, its id set to -1 and 0x520d10
//     is called; an entry of the instance table (0x685191, 12 bytes each,
//     count at 0x686901) is then compacted and 0x4edd20 is called.
//
// Callees: 0x520750 (retention test) and 0x520d10, 0x4edd20 are D4 formula
// nodes and 0x427ab0 / 0x427ff0 are presentation helpers (registered 排除):
// the case address-stubs them, so the node outputs their call counts and
// recorded arguments. 0x5493e0 / 0x549390 / 0x5e1da0 run natively.
// Fixture caps: table entries itemUseTab, item records itemUseItems; any
// pointer or id outside them panics.
const (
	itemUseTab    = 5
	itemUseItems  = 4
	itemUseCursor = 0x6874cb
	itemUseTabLo  = 0x685191
	itemUseTabHi  = 0x6868f5
	itemUseExtLo  = 0x10000000 // arena objects: instance fields come from the ext inputs
)

type itemUseInst struct{ id, d4, d8 int64 } // d4 holds the count byte (bits 0..7) and charge byte (bits 8..15)

func init() {
	in := []string{"a0", "inst", "force", "n", "ext.id", "ext.d4", "ext.d8", "cur.id", "cur.d4", "cur.d8", "cur.flag"}
	out := []string{"ext.id", "ext.d4", "ext.d8", "cur.id", "cur.d4", "cur.d8", "cur.flag"}
	for k := 0; k < itemUseTab; k++ {
		for _, f := range []string{"id", "d4", "d8"} {
			in = append(in, fmt.Sprintf("t%d.%s", k, f))
			out = append(out, fmt.Sprintf("t%d.%s", k, f))
		}
	}
	for i := 0; i < itemUseItems; i++ {
		in = append(in, fmt.Sprintf("it%d.b66", i), fmt.Sprintf("it%d.b3e", i))
	}
	in = append(in, "g6840c0", "g6840c4", "g6840c5", "g683fdb", "g683fdf", "g683fe3", "retain", "cursorWord", "cursorHandle")
	out = append(out, "n", "g6840c0", "g6840c4", "g6840c5", "g683fdb", "g683fdf", "g683fe3",
		"cnt520750", "arg520750.1", "cnt520d10", "arg520d10.1", "arg520d10.2", "arg520d10.3",
		"cnt427ab0", "arg427ab0.1", "arg427ab0.2", "arg427ab0.3", "arg427ab0.4", "cnt427ff0", "cnt4edd20")
	Nodes["0051e760"] = Node{Addr: "0051e760", In: in, Out: out, F: itemUseConsumptionNode}
}

func itemUseConsumptionNode(in []int64, r Rand) []int64 {
	if len(in) != 43 {
		panic("itemUseConsumption: expected 43 inputs")
	}
	p := 0
	next := func() int64 { p++; return in[p-1] }
	owner, inst, force, n := next(), next(), next(), next()
	var ext, cur itemUseInst
	ext = itemUseInst{next(), next(), next()}
	cur = itemUseInst{next(), next(), next()}
	curFlag := next()
	var tab [itemUseTab]itemUseInst
	for k := range tab {
		tab[k] = itemUseInst{next(), next(), next()}
	}
	var b66, b3e [itemUseItems]int64
	for i := range b66 {
		b66[i], b3e[i] = next()&0xff, next()&0xff
	}
	g40c0, g40c4, g40c5 := next(), next(), next()
	g3fdb, g3fdf, g3fe3 := next(), next(), next()
	retain, curWord, curHandle := next(), next(), next()

	var cnt520750, arg520750, cnt520d10, cnt427ab0, cnt427ff0, cnt4edd20 int64
	var a520d10 [3]int64
	var a427ab0 [4]int64

	slot := -1 // table index of the instance, -1 when it is not a table entry
	var it *itemUseInst
	switch {
	case inst == itemUseCursor:
		it = &cur
	case inst >= itemUseTabLo && inst <= itemUseTabHi:
		if (inst-itemUseTabLo)%12 != 0 || (inst-itemUseTabLo)/12 >= itemUseTab {
			panic("itemUseConsumption: instance pointer outside the modelled table")
		}
		slot = int((inst - itemUseTabLo) / 12)
		it = &tab[slot]
	case inst >= itemUseExtLo:
		it = &ext
	default:
		panic("itemUseConsumption: instance pointer not modelled")
	}
	if n < 0 || n > itemUseTab {
		panic("itemUseConsumption: table count outside the modelled table")
	}

	finish := func() []int64 {
		o := []int64{ext.id, ext.d4, ext.d8, cur.id, cur.d4, cur.d8, curFlag}
		for _, e := range tab {
			o = append(o, e.id, e.d4, e.d8)
		}
		return append(o, n, g40c0, g40c4, g40c5, g3fdb, g3fdf, g3fe3,
			cnt520750, arg520750, cnt520d10, a520d10[0], a520d10[1], a520d10[2],
			cnt427ab0, a427ab0[0], a427ab0[1], a427ab0[2], a427ab0[3], cnt427ff0, cnt4edd20)
	}

	if it.id == -1 {
		return finish()
	}
	if it.id < 0 || it.id >= itemUseItems {
		panic("itemUseConsumption: item id outside the modelled records")
	}
	id := it.id
	if b66[id] == 0 && b3e[id] != 0x16 {
		return finish()
	}
	if b66[id] == 1 { // count byte
		if it.d4&0xff != 0 {
			it.d4 = it.d4&^0xff | (it.d4&0xff - 1)
		}
		if it.d4&0xff != 0 {
			return finish()
		}
	} else { // charge byte
		c := it.d4 >> 8 & 0xff
		if c == 0 {
			if force&0xff == 0 {
				return finish()
			}
		} else {
			c--
			it.d4 = it.d4&^0xff00 | c<<8
			if c != 0 {
				return finish()
			}
			cnt520750, arg520750 = 1, id
			if retain&0xff != 0 {
				return finish()
			}
		}
	}

	// removal
	if inst == itemUseCursor {
		g40c0, g40c4, g40c5 = -1, 0xff, 0xffff
		// 0x55f1e0 (native): clear the held-item state, refresh the cursor when a mode is active
		cur = itemUseInst{-1, 0, 0}
		curFlag = 0
		if g3fdb != -1 {
			a427ab0 = [4]int64{curHandle & 0xffffffff, curWord & 0xffff, 0, 0}
			cnt427ab0, cnt427ff0 = 1, 1
			g3fdb, g3fdf, g3fe3 = -1, 0, 0
		}
	} else {
		*it = itemUseInst{-1, 0, 0}
		cnt520d10, a520d10 = 1, [3]int64{inst, owner & 0xffffffff, 1}
	}
	if slot < 0 || n == 0 { // not a table entry (cursor 0x6874cb and arena instances are outside the table range)
		return finish()
	}
	if int64(slot) >= n || tab[slot].id != -1 {
		return finish()
	}
	copy(tab[slot:n-1], tab[slot+1:n])
	tab[n-1] = itemUseInst{-1, 0, 0}
	n--
	cnt4edd20 = 1
	return finish()
}
