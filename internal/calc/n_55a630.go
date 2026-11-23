package calc

import "fmt"

// shopInitialStockNode is 0x55a630 shop_initial_stock(merchant): roll the
// merchant's starting stock from its shop record's offer list.
//
// merchant+6 is the shop record, record+0x2ca its offer list (a 0x5e22c0-style
// list of entry pointers; an entry is {i32 item id, u8 quantity at +4, u8 fixed
// flag at +5}). A null offer list or an empty one clears merchant+0xa (the
// stock list pointer) and returns 1. Otherwise merchant+0xa is created when
// null, and every offer whose item id is below the global item count (0x683f78,
// signed) rolls `quantity` percent dice: the chance is 100 when the fixed flag
// is set, else the stock class from 0x55aad0 maps 0..4 to 0/25/50/75/100
// (jump table 0x55a798; a class past 4 would give 0 but 0x55aad0 returns 0..3).
// A roll succeeds when R(100) < chance. The u8 success count, when nonzero, goes
// to 0x55a7b0(merchant, id, count); then 0x55b9c0(merchant) re-sorts the stock
// list; finally the global 0x686a48 is stored at merchant+0xe and +0x12.
//
// 0x55a7b0 is address-stubbed in the case (a formula callee already at D4): its
// arguments of the last call and its call count are outputs, it adds nothing to
// the stock list, and its return is a case input the caller ignores.
// 0x55aad0 and its party average run natively; the party is eight slots, and an
// empty party faults in the original (division by zero at 0x4ef456), so the
// fixture always has slot 0 active and calc panics if the class is needed with
// no active slot.
// 0x55b9c0 runs natively: it copies the stock list entries (0x14 bytes each) to
// a new buffer, qsorts them with comparator 0x55baf0 (which passes entry+4 to
// 0x520600), empties the old list (0x5e27c0 until empty, then 0x5e2440 and
// 0x5e23e0), builds a new list with 0x5e22c0 and re-adds a fresh 0x14-byte copy
// of every sorted entry. The emulator heap is a bump allocator (see
// n_55a7b0.go), so the case compares its first 0x1c0 bytes.
//
// Fixture caps (calc panics past them): the offer list has at most 4 entries and
// the stock list at most 8 (MSVCRT qsort then is the shortsort of emu.py), item
// ids are 0..15 (the case's 16 item records), list pointers are 0 or the fixture
// list objects.
const (
	sisArena = 0x20000000
	sisMer   = sisArena
	sisItems = sisArena + 0x1000
	sisMList = sisArena + 0x3000
	sisMData = sisArena + 0x3100
	sisMEnts = sisArena + 0x3400
	sisStub  = sisArena + 0x4000
	sisRec   = sisArena + 0x5000
	sisSList = sisArena + 0x6000
	sisSData = sisArena + 0x6100
	sisSEnts = sisArena + 0x6200
	sisItemN = 16
	sisOffN  = 4
	sisStkN  = 8
	sisHeapW = 0x1c0
)

// sisWorld is the sparse memory and bump heap of n_55a7b0.go.
type sisWorld struct{ ssaWorld }

// sisField is one case input: name, byte size and the address it is written to.
type sisField struct {
	name string
	size int
	at   func(v map[string]int64) int64
}

func sisFixed(a int64) func(map[string]int64) int64 {
	return func(map[string]int64) int64 { return a }
}

// sisItemAt is the item record field `off` of the record whose id input is `id`.
func sisItemAt(id string, off int64) func(map[string]int64) int64 {
	return func(v map[string]int64) int64 {
		if v[id] < 0 || v[id] >= sisItemN {
			panic("shop_initial_stock: item id past the 16-record fixture")
		}
		return sisItems + v[id]*0x10d + off
	}
}

var sisSpec = func() (f []sisField) {
	add := func(n string, size int, at func(map[string]int64) int64) { f = append(f, sisField{n, size, at}) }
	add("limit", 4, sisFixed(0x683f78))
	add("stamp", 4, sisFixed(0x686a48))
	add("stubRet", 4, sisFixed(sisStub+0x20))
	add("listp", 4, sisFixed(sisMer+0xa))
	add("srcp", 4, sisFixed(sisRec+0x2ca))
	add("scount", 4, sisFixed(sisSList+8))
	for j := int64(0); j < sisOffN; j++ {
		s := fmt.Sprint(j)
		add("sid"+s, 4, sisFixed(sisSEnts+0x20*j))
		add("sq"+s, 1, sisFixed(sisSEnts+0x20*j+4))
		add("sf"+s, 1, sisFixed(sisSEnts+0x20*j+5))
		add("slv"+s, 1, sisItemAt("sid"+s, 0x85))
	}
	for k := int64(0); k < 8; k++ {
		s := fmt.Sprint(k)
		add("sa"+s, 1, sisFixed(sisArena+0x20000+0x106*k))
		add("tl"+s, 4, sisFixed(sisArena+0x10000+0x1862*k+0x89))
	}
	add("mcount", 4, sisFixed(sisMList+8))
	for i := int64(0); i < sisStkN; i++ {
		s := fmt.Sprint(i)
		add("mid"+s, 4, sisFixed(sisMEnts+0x20*i+4))
		add("mb8_"+s, 1, sisFixed(sisMEnts+0x20*i+8))
		add("mb9_"+s, 1, sisFixed(sisMEnts+0x20*i+9))
		add("mb10_"+s, 1, sisFixed(sisMEnts+0x20*i+10))
		add("mcls"+s, 1, sisItemAt("mid"+s, 0x3e))
		add("mw"+s, 2, sisItemAt("mid"+s, 0x3f))
		add("md"+s, 4, sisItemAt("mid"+s, 0x86))
	}
	return
}()

var sisIn, sisOut = func() (in, out []string) {
	for _, f := range sisSpec {
		in = append(in, f.name)
	}
	out = []string{"ret", "merList", "stampA", "stampB", "stubMer", "stubId", "stubCnt", "stubCalls"}
	for _, o := range []int{0, 4, 8, 0xc} {
		out = append(out, fmt.Sprintf("mlist%x", o))
	}
	for k := 0; k < 12; k++ {
		out = append(out, fmt.Sprintf("mdata%d", k))
	}
	for i := 0; i < sisStkN; i++ {
		for _, o := range []int{0, 4, 8, 0xc, 0x10} {
			out = append(out, fmt.Sprintf("ment%d.%x", i, o))
		}
	}
	for _, o := range []int{0, 4, 8, 0xc} {
		out = append(out, fmt.Sprintf("slist%x", o))
	}
	for k := 0; k < sisHeapW/4; k++ {
		out = append(out, fmt.Sprintf("h%d", k))
	}
	return
}()

func init() {
	Nodes["0055a630"] = Node{Addr: "0055a630", In: sisIn, Out: sisOut, F: shopInitialStockNode}
}

func shopInitialStockNode(in []int64, r Rand) []int64 {
	if len(in) != len(sisIn) {
		panic("shop_initial_stock: wrong input count")
	}
	v := map[string]int64{}
	for i, n := range sisIn {
		v[n] = in[i]
	}
	s := &sisWorld{ssaWorld{cgW: cgW{m: map[int64]byte{}}}}
	s.sisSetup()
	// The case's inputs, in input order (a later write to the same byte wins).
	for _, f := range sisSpec {
		s.wr(f.at(v), f.size, v[f.name])
	}
	if lp := s.u32(sisMer + 0xa); lp != 0 && lp != sisMList {
		panic("shop_initial_stock: stock list pointer is not the fixture object")
	}
	if sp := s.u32(sisRec + 0x2ca); sp != 0 && sp != sisSList {
		panic("shop_initial_stock: offer list pointer is not the fixture object")
	}
	s.roll(r)

	out := []int64{1}
	rd := func(a int64) { out = append(out, s.u32(a)) }
	rd(sisMer + 0xa)
	rd(sisMer + 0xe)
	rd(sisMer + 0x12)
	for _, o := range []int64{0, 4, 8, 0x10} {
		rd(sisStub + o)
	}
	for _, o := range []int64{0, 4, 8, 0xc} {
		rd(sisMList + o)
	}
	for k := int64(0); k < 12; k++ {
		rd(sisMData + 4*k)
	}
	for i := int64(0); i < sisStkN; i++ {
		for _, o := range []int64{0, 4, 8, 0xc, 0x10} {
			rd(sisMEnts + 0x20*i + o)
		}
	}
	for _, o := range []int64{0, 4, 8, 0xc} {
		rd(sisSList + o)
	}
	for k := int64(0); k < sisHeapW/4; k++ {
		rd(ssaHeap + 4*k)
	}
	return out
}

// sisSetup writes the case's fixed setup: globals, pointers and the decoy bytes
// around every field the rule reads.
func (s *sisWorld) sisSetup() {
	s.p32(0x68516c, sisItems)
	s.p32(sisMer, 0x11110000)
	s.wr(sisMer+4, 2, 0x2222)
	s.p32(sisMer+6, sisRec)
	s.p32(sisMer+0xe, 0x33333333)
	s.p32(sisMer+0x12, 0x44444444)
	s.p32(sisMer+0x16, 0x55555555)
	s.p32(sisRec+0x2c6, 0x66666666)
	s.p32(sisRec+0x2ce, 0x77777777)
	s.p32(sisSList, sisSData)
	s.p32(sisSList+4, 10)
	s.p32(sisSList+0xc, 0x7e7e7e7e)
	s.p32(sisMList, sisMData)
	s.p32(sisMList+4, 10)
	s.p32(sisMList+0xc, 0x7e7e7e7e)
	for k := int64(0); k < 12; k++ {
		sd, md := int64(0xdead0000+k), int64(0xdead0000+k)
		if k < sisOffN {
			sd = sisSEnts + 0x20*k
		}
		if k < sisStkN {
			md = sisMEnts + 0x20*k
		}
		s.p32(sisSData+4*k, sd)
		s.p32(sisMData+4*k, md)
	}
	for j := int64(0); j < sisOffN; j++ {
		b := sisSEnts + 0x20*j
		s.wr(b+6, 2, 0xa5a5)
		s.p32(b+8, 0xb1b1b100+j)
	}
	for i := int64(0); i < sisStkN; i++ {
		b := sisMEnts + 0x20*i
		s.p32(b, 0xa5a5a500+i)
		s.p32(b+8, 0xc4000000)
		s.p32(b+0xc, 0xc3c3c300+i)
		s.p32(b+0x10, 0xd1d1d100+i)
	}
	for k := int64(0); k < sisItemN; k++ {
		b := sisItems + 0x10d*k
		s.p8(b+0x3e, 0x70+k)
		s.wr(b+0x3f, 2, 0x1000+k)
		s.p8(b+0x85, 0x31+k)
		s.p32(b+0x86, 0x9000+k)
	}
}

func (s *sisWorld) roll(r Rand) {
	offers := s.u32(sisRec + 0x2ca)
	if offers == 0 || s.u32(offers+8) == 0 {
		s.p32(sisMer+0xa, 0)
		return
	}
	if s.u32(offers+8) > sisOffN {
		panic("shop_initial_stock: offer list count past the fixture")
	}
	if s.u32(sisMer+0xa) == 0 {
		s.p32(sisMer+0xa, s.listNew())
	}
	// The loop bound is read once (exe 0x55a683); the list is not changed.
	n := s.u32(offers + 8)
	for i := int64(0); i < n; i++ {
		e := s.listGet(offers, i)
		if e == 0 {
			panic("shop_initial_stock: null offer entry")
		}
		id := s.i32(e)
		if id >= s.i32(0x683f78) {
			continue
		}
		qty, fixed := s.u8(e+4), s.u8(e+5)
		succ := int64(0)
		for k := int64(0); k < qty; k++ {
			class := int64(4)
			if fixed == 0 {
				class = s.stockClass(id)
			}
			chance := []int64{0, 25, 50, 75, 100}[class]
			if int64(r.R(100)) < chance {
				succ = (succ + 1) & 0xff
			}
		}
		if succ != 0 {
			// Stubbed 0x55a7b0(merchant, id, count): last call's arguments, call count.
			for a, x := range []int64{sisMer, id & 0xffffffff, succ} {
				s.p32(sisStub+4*int64(a), x)
			}
			s.p32(sisStub+0x10, s.u32(sisStub+0x10)+1)
		}
	}
	s.resortStock()
	s.p32(sisMer+0x12, s.u32(0x686a48))
	s.p32(sisMer+0xe, s.u32(0x686a48))
}

// stockClass is 0x55aad0(merchant, id): the item level against the party average.
func (s *sisWorld) stockClass(id int64) int64 {
	if id < 0 || id >= sisItemN {
		panic("shop_initial_stock: item id past the 16-record fixture")
	}
	pair := make([]int64, 0, 16)
	for k := int64(0); k < 8; k++ {
		pair = append(pair, s.u8(sisArena+0x20000+0x106*k), s.u32(sisArena+0x10000+0x1862*k+0x89))
	}
	avg := partyAverageLevelNode(pair, nil)[0]
	return sisClassOf(avg, s.rs(sisItems+id*0x10d+0x85, 1))
}

// stockCompare is 0x55baf0 -> 0x520600 on entries a and b (their +4 words).
func (s *sisWorld) stockCompare(a, b int64) int64 {
	ia, ib := s.u32(a+4), s.u32(b+4)
	if ia >= sisItemN || ib >= sisItemN {
		panic("shop_initial_stock: stock entry item id past the 16-record fixture")
	}
	ra, rb := sisItems+ia*0x10d, sisItems+ib*0x10d
	switch ca, cb := s.u8(ra+0x3e), s.u8(rb+0x3e); {
	case ca > cb:
		return 1
	case ca < cb:
		return -1
	}
	switch wa, wb := s.rd(ra+0x3f, 2), s.rd(rb+0x3f, 2); {
	case wa > wb:
		return 1
	case wa < wb:
		return -1
	}
	fa, fb := s.u8(a+0xa), s.u8(b+0xa)
	if fa == 0 {
		if fb == 0 {
			return 0
		}
		return 1
	}
	if fb == 0 {
		return -1
	}
	switch da, db := s.u32(ra+0x86), s.u32(rb+0x86); {
	case da < db:
		return 1
	case da > db:
		return -1
	}
	switch qa, qb := s.u8(a+8), s.u8(b+8); {
	case qa < qb:
		return 1
	case qa > qb:
		return -1
	}
	switch ga, gb := s.u8(a+9), s.u8(b+9); {
	case gb > ga:
		return 1
	case gb < ga:
		return -1
	}
	return 0
}

// resortStock is 0x55b9c0(merchant).
func (s *sisWorld) resortStock() {
	list := s.u32(sisMer + 0xa)
	var n int64
	if list != 0 {
		n = s.u32(list + 8)
	}
	if n == 0 {
		return
	}
	if n > sisStkN {
		panic("shop_initial_stock: stock list count past the fixture")
	}
	buf := s.alloc(20 * n)
	for i := int64(0); i < n; i++ {
		e := s.listGet(list, i)
		if e == 0 {
			panic("shop_initial_stock: null stock entry")
		}
		for o := int64(0); o < 20; o += 4 {
			s.p32(buf+20*i+o, s.u32(e+o))
		}
	}
	// MSVCRT qsort for at most 8 elements is the shortsort of emu.py: move the
	// first maximum under the comparator to the end, shrink, repeat.
	for hi := n - 1; hi > 0; hi-- {
		mx := int64(0)
		for p := int64(1); p <= hi; p++ {
			if s.stockCompare(buf+20*p, buf+20*mx) > 0 {
				mx = p
			}
		}
		if mx != hi {
			for o := int64(0); o < 20; o += 4 {
				x, y := s.u32(buf+20*mx+o), s.u32(buf+20*hi+o)
				s.p32(buf+20*mx+o, y)
				s.p32(buf+20*hi+o, x)
			}
		}
	}
	// 0x5e27c0(list, 0) until empty: shift the pointers down, count - 1.
	for s.u32(list+8) != 0 {
		c := s.i32(list + 8)
		for j := int64(0); j < c-1; j++ {
			s.p32(s.u32(list)+4*j, s.u32(s.u32(list)+4*(j+1)))
		}
		s.p32(list+8, c-1)
	}
	s.p32(list, 0) // 0x5e2440 frees the array and nulls it; 0x5e23e0 frees the header
	fresh := s.listNew()
	s.p32(sisMer+0xa, fresh)
	for i := int64(0); i < n; i++ {
		e := s.alloc(0x14)
		for o := int64(0); o < 20; o += 4 {
			s.p32(e+o, s.u32(buf+20*i+o))
		}
		s.listAdd(fresh, e)
	}
}

// sisClassOf is the rule of 0x55aad0 after the party average (same as
// shopStockLevelClassNode, which takes only two party slots).
func sisClassOf(avg, itemLevel int64) int64 {
	a := i8(i8(i8(avg)*100/30) / 5)
	lo := min(max(a, 1), 16)
	hi := min(max(i8(a+4), 4), 20)
	b := i8(itemLevel)
	switch {
	case b > hi:
		return 0
	case b >= lo:
		return 3
	case b+4 >= lo:
		return 2
	}
	return 1
}
