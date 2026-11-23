package calc

import "fmt"

// shop_sold_stock_merge 0x55a930 (build/wiz8.asm:471310): find or create the
// merchant's stock entry (merchant+0xa list, created by 0x5e22c0 when null)
// for a sold item and add the sold count to one of its two counters. The item
// definition +0x3e == 4 skips the search (always a new entry); a new entry is
// 0x14 bytes from operator new, filled by initNewItem 0x51c020 (called as
// (entry+4, id, 1, 1, 0)), byte +0x8 cleared, then appended by 0x5e2480. After
// that item definition +0x66 == 1 means: +0x10 is raised to at least 1 and
// the count is added to +0x8; otherwise +0x8 is raised to at least 1 and the
// count is added to +0x10 (both u8 adds, upper bytes of the dwords kept).
// Returns the entry's list index, or -1 for a null merchant or negative item.
//
// Callees: 0x51c020 is a D4 formula but writes the new instance's memory, so
// it cannot be an address stub: the case runs it natively (its own callee
// 0x520d10, D4, is the address stub whose arguments and call count are
// outputs) and calc calls itemStackChargeInitNode. 0x5e22c0/0x5e2480/0x5e2870/
// 0x5e2c70 (list) and operator new are modelled here; the emulator bump heap
// (0x50000000, sizes rounded up to 8, minimum 8) is reproduced by sssmAlloc.
//
// Fixture caps (calc panics past them): item ids 0..3 (the 0x51c020 case
// models four item records), list count and capacity 0..12, list array pointers
// non-null (the game never stores null). Entry words +0x0/+0xc are never
// written; decoys sit there so a stray write shows in the outputs.
const (
	sssmItems   = 4
	sssmEnts    = 12
	sssmMer     = 0x20001000
	sssmLst     = 0x20003000
	sssmArr     = 0x20003100
	sssmEnt     = 0x20003200
	sssmHeap    = 0x50000000
	sssmHeapDw  = 24
	sssmInst    = 0x20004000 // 0x51c020 case object "inst"; stands for entry+4
	sssmEntSize = 0x14
)

var sssmIn = func() []string {
	n := []string{"merchant", "soldIdx", "count", "listPtr", "cap", "n"}
	for k := 0; k < sssmItems; k++ {
		for _, f := range []string{"cls", "kind", "z", "n", "s"} {
			n = append(n, fmt.Sprintf("i%d.%s", k, f))
		}
	}
	for j := 0; j < sssmEnts; j++ {
		for _, f := range []string{"item", "q8", "q10"} {
			n = append(n, fmt.Sprintf("e%d.%s", j, f))
		}
	}
	return append(n, "refreshRet")
}()

var sssmOut = func() []string {
	n := []string{"eax", "listPtr", "lst0", "cap", "n"}
	for i := 0; i < sssmEnts; i++ {
		n = append(n, fmt.Sprintf("arr%d", i))
	}
	for j := 0; j < sssmEnts; j++ {
		for _, o := range []int{0, 4, 8, 0xc, 0x10} {
			n = append(n, fmt.Sprintf("e%d.%x", j, o))
		}
	}
	for k := 0; k < sssmHeapDw; k++ {
		n = append(n, fmt.Sprintf("heap%d", k))
	}
	return append(n, "rec0", "rec1", "rec2", "rec3")
}()

func init() {
	Nodes["0055a930"] = Node{Addr: "0055a930", In: sssmIn, Out: sssmOut, F: shopSoldStockMergeNode}
}

type sssmMem struct {
	b    map[int64]byte
	heap int64
}

func (m *sssmMem) put(a int64, n int, v int64) {
	for i := 0; i < n; i++ {
		m.b[a+int64(i)] = byte(v >> (8 * uint(i)))
	}
}

func (m *sssmMem) rd(a int64, n int) int64 {
	var v int64
	for i := n - 1; i >= 0; i-- {
		x, ok := m.b[a+int64(i)]
		if !ok && !(a+int64(i) >= sssmHeap && a+int64(i) < sssmHeap+0x100000) {
			panic(fmt.Sprintf("shop_sold_stock_merge: read of unmodelled address %#x", a+int64(i)))
		}
		v = v<<8 | int64(x)
	}
	return v
}

func (m *sssmMem) alloc(n int64) int64 {
	a := m.heap
	if n < 1 {
		n = 1
	}
	s := (n + 7) &^ 7
	if s < 8 {
		s = 8
	}
	m.heap += s
	return a
}

func shopSoldStockMergeNode(in []int64, r Rand) []int64 {
	if len(in) != len(sssmIn) {
		panic("shop_sold_stock_merge: wrong input count")
	}
	at := map[string]int{}
	for i, n := range sssmIn {
		at[n] = i
	}
	v := func(n string) int64 { return in[at[n]] }
	m := &sssmMem{b: map[int64]byte{}, heap: sssmHeap}
	m.put(sssmMer+0xa, 4, v("listPtr"))
	m.put(sssmLst, 4, sssmArr)
	m.put(sssmLst+4, 4, v("cap"))
	m.put(sssmLst+8, 4, v("n"))
	for i := int64(0); i < sssmEnts; i++ {
		m.put(sssmArr+4*i, 4, sssmEnt+sssmEntSize*i)
		e := sssmEnt + sssmEntSize*i
		m.put(e, 4, 0xA5A50000+i)
		m.put(e+4, 4, v(fmt.Sprintf("e%d.item", i)))
		m.put(e+8, 4, v(fmt.Sprintf("e%d.q8", i)))
		m.put(e+0xc, 4, 0x5A5A0000+i)
		m.put(e+0x10, 4, v(fmt.Sprintf("e%d.q10", i)))
	}
	var rec [4]int64
	eax := int64(-1)
	merchant := v("merchant") & 0xffffffff
	idx := int64(int32(v("soldIdx")))
	if merchant != 0 && idx >= 0 {
		eax = sssmRun(m, v, idx, v("count")&0xff, &rec)
	}
	out := []int64{int64(int32(eax)), m.rd(sssmMer+0xa, 4), m.rd(sssmLst, 4), int64(int32(m.rd(sssmLst+4, 4))), int64(int32(m.rd(sssmLst+8, 4)))}
	for i := int64(0); i < sssmEnts; i++ {
		out = append(out, m.rd(sssmArr+4*i, 4))
	}
	for j := int64(0); j < sssmEnts; j++ {
		for _, o := range []int64{0, 4, 8, 0xc, 0x10} {
			out = append(out, m.rd(sssmEnt+sssmEntSize*j+o, 4))
		}
	}
	for k := int64(0); k < sssmHeapDw; k++ {
		out = append(out, m.rd(sssmHeap+4*k, 4))
	}
	return append(out, rec[:]...)
}

func sssmRun(m *sssmMem, v func(string) int64, idx, cnt int64, rec *[4]int64) int64 {
	if idx >= sssmItems {
		panic("shop_sold_stock_merge: item id past fixture cap")
	}
	item := func(id int64, f string) int64 { return v(fmt.Sprintf("i%d.%s", id, f)) }
	if m.rd(sssmMer+0xa, 4) == 0 { // 0x5e22c0: header 0xc bytes, array 0x28 bytes
		h := m.alloc(0xc)
		a := m.alloc(0x28)
		m.put(h, 4, a)
		m.put(h+4, 4, 10)
		m.put(h+8, 4, 0)
		m.put(sssmMer+0xa, 4, h)
	}
	list := m.rd(sssmMer+0xa, 4)
	count := func() int64 { return int64(int32(m.rd(list+8, 4))) }
	entry := func(i int64) int64 { // 0x5e2870
		if i >= count() {
			return 0
		}
		return m.rd(m.rd(list, 4)+4*i, 4)
	}
	if c := count(); c < 0 || c > sssmEnts || int64(int32(m.rd(list+4, 4))) < 0 || int64(int32(m.rd(list+4, 4))) > sssmEnts {
		panic("shop_sold_stock_merge: list count or capacity past fixture cap")
	}
	var e, index int64
	found := false
	if item(idx, "cls")&0xff != 4 {
		for i := int64(0); i < count(); i++ {
			p := entry(i)
			if p != 0 && m.rd(p+4, 4) == idx {
				e, index, found = p, i, true
				break
			}
		}
	}
	if !found {
		e = m.alloc(sssmEntSize)
		for o := int64(0); o <= 0x10; o += 4 {
			m.put(e+o, 4, 0)
		}
		ii := make([]int64, len(iscIn))
		at := map[string]int{}
		for i, n := range iscIn {
			at[n] = i
		}
		ii[at["p"]], ii[at["id"]], ii[at["maxRoll"]], ii[at["forceFlag6"]] = sssmInst, idx, 1, 1
		ii[at["itemCount"]] = sssmItems
		for k := int64(0); k < sssmItems; k++ {
			for _, f := range []string{"kind", "z", "n", "s"} {
				ii[at[fmt.Sprintf("i%d.%s", k, f)]] = item(k, f)
			}
		}
		res := itemStackChargeInitNode(ii, nil)
		for i, n := range iscOut {
			switch n {
			case "inst0", "inst4", "inst8":
				m.put(e+4+4*int64(i), 4, res[i])
			}
		}
		rec[0], rec[1], rec[2], rec[3] = e+4, 0, 1, 1 // 0x520d10(p, 0, 1) once
		m.put(e+8, 1, 0)
		// 0x5e2480
		cap := int64(int32(m.rd(list+4, 4)))
		n := count()
		if n >= cap {
			na := m.alloc(cap*4 + 0x14)
			for i := int64(0); i < n; i++ {
				m.put(na+4*i, 4, m.rd(m.rd(list, 4)+4*i, 4))
			}
			m.put(list, 4, na)
			m.put(list+4, 4, cap+5)
		}
		m.put(m.rd(list, 4)+4*n, 4, e)
		m.put(list+8, 4, n+1)
		index = n
	}
	id := m.rd(e+4, 4)
	if id >= sssmItems {
		panic("shop_sold_stock_merge: item id past fixture cap")
	}
	if item(id, "kind")&0xff != 1 {
		if m.rd(e+8, 1) == 0 {
			m.put(e+8, 1, 1)
		}
		m.put(e+0x10, 1, m.rd(e+0x10, 1)+cnt)
	} else {
		if m.rd(e+0x10, 1) == 0 {
			m.put(e+0x10, 1, 1)
		}
		m.put(e+8, 1, m.rd(e+8, 1)+cnt)
	}
	return index
}
