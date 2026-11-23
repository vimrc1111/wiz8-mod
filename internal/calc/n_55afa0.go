package calc

import "fmt"

// shopTimedRestockNode is 0x55afa0 shop_timed_restock(merchant, force): the
// merchant's periodic stock upkeep. Returns 1 when the daily restock ran, else 0.
//
// merchant+6 is the shop record (record+0x2ca its offer list), merchant+0xa the
// stock list, merchant+0xe and +0x12 two u32 time stamps; the clock is the u32 at
// 0x686a48 (now) and force is the low byte of the second argument. A list is
// {data pointer, capacity, count at +8}; an offer is {i32 item id, u8 quantity at
// +4, u8 fixed flag at +5}; a stock entry is {u32 gate at +0, i32 item id at +4,
// u8 stack quantity at +8, u8 second quantity at +9, u8 piece count at +0x10}.
// Item definitions are 0x10d bytes apiece from the pointer at 0x68516c: class
// +0x3e, flags +0x41, kind +0x66, level +0x85.
//
//  1. Every stock entry with a nonzero piece count is normalized by 0x522e80
//     (itemCountNormalizationNode on the entry's id, kind and +8/+9); then, if
//     its kind is 1 and the piece count exceeds the stack quantity, the stack
//     quantity becomes the piece count and the piece count 1.
//  2. If now - merchant+0xe (u32) exceeds 0xa8c0 or force is nonzero: that stamp
//     becomes now, and every offer with a nonzero fixed flag is topped up. The
//     quantity is that of the FIRST offer with the same id (zero skips); cur is
//     the piece count of the first non-null stock entry with that id, else 0;
//     cur > quantity/2 skips; delta = u8(quantity - cur); R(3): 0 gives
//     u8(delta + delta/2), 1 gives delta - delta/2, 2 leaves it; a zero delta
//     skips, else 0x55a7b0(merchant, id, delta) adds stock.
//  3. If now - merchant+0x12 (u32) is below 0x15180 and force is zero: return 0.
//     Otherwise 0x55ae70(merchant) (stock outflow) and 0x55ab80(merchant) (optional
//     restock) run, then every stock entry with piece count 0 is removed from the
//     list by 0x5e27c0 (data pointers shift down, count - 1; after a removal the
//     scan steps back one slot unless it is at slot 0, so a removal at slot 0
//     skips the entry that shifts into it), and both stamps become now.
//
// The list helpers 0x5e2870/0x5e2c70/0x5e27c0 and 0x522e80 run natively in the
// case; the delete 0x5e1c10 of a removed entry is a no-op (the entry memory stays).
//
// Emu and F share this body. Emu (what TestEmu replays) is the case with 0x55a7b0,
// 0x55ae70 and 0x55ab80 address-stubbed (D4 formula callees): their returns are the
// case inputs ret7b0, ret70, ret80 (the caller ignores eax), the case outputs the
// arguments of every 0x55a7b0 call (up to 4), of the 0x55ae70 and 0x55ab80 calls and
// the three call counts, and the stubs change no memory, so the stock list stays as
// set up through the whole run. F calls the real callees on the same memory:
// shopStockAddNode through sorRealAdd (0x55a7b0), shopStockTurnoverNode (0x55ae70)
// and sorRun with real=true (0x55ab80, its item class from the party average and the
// definition level). F reads the F-only inputs: gate sg_i of stock entry i, class
// dc_t, flags df_t and level dl_t of item definition t (kind dk_t is read by the
// node itself), the story fact (0x689cd7) and the party slots pa_k/pt_k; Emu ignores
// them.
//
// Fixture caps (calc panics past them): offer list count <= 4, stock list count
// <= 6, list pointers 0 or the fixture lists, stock data slots 0 or their fixture
// entry (offer slots are always the fixture entries), item ids -1..5 wherever a
// definition is read. F's extra caps: stock count <= 3 and offer count <= 3 at the
// 0x55ae70 call, eligible stock entries with ids 0..5 there, offer ids -1..5 when
// the list is non-empty, plus the caps of sorRealAdd and sorRun (stock entries after
// growth <= 6, at most 16 distinct ids).
const (
	ssrRst     = sorArena + 0x2000 // stub records (outputs of the address stubs)
	ssrDefs    = sorArena + 0x3000 // item definitions for ids -1..5
	ssrDefSize = 0x10d
	ssrDefN    = 7
	ssrNow     = 0x686a48
	ssrDefBase = 0x68516c
)

type ssrField struct {
	name string
	size int
	at   int64
}

var ssrSpec = func() (f []ssrField) {
	add := func(n string, size int, at int64) { f = append(f, ssrField{n, size, at}) }
	add("force", 0, 0) // the second argument, not memory
	add("now", 4, ssrNow)
	add("stampA", 4, sorMer+0xe)
	add("stampB", 4, sorMer+0x12)
	add("fact", 1, sorFact)
	add("ret7b0", 4, ssrRst+0x20)
	add("ret70", 4, ssrRst+0x24)
	add("ret80", 4, ssrRst+0x28)
	add("olistp", 4, sorRec+0x2ca)
	add("ocount", 4, sorOList+8)
	for j := int64(0); j < sorOffN; j++ {
		s := fmt.Sprint(j)
		add("oid"+s, 4, sorOEnts+0x20*j)
		add("oq"+s, 1, sorOEnts+0x20*j+4)
		add("of"+s, 1, sorOEnts+0x20*j+5)
	}
	add("slistp", 4, sorMer+0xa)
	add("scount", 4, sorSList+8)
	for i := int64(0); i < sorStkN; i++ {
		s := fmt.Sprint(i)
		add("sd"+s, 4, sorSData+4*i)
		add("sg"+s, 4, sorSEnts+0x20*i)
		add("sid"+s, 4, sorSEnts+0x20*i+4)
		add("sq"+s, 1, sorSEnts+0x20*i+8)
		add("sr"+s, 1, sorSEnts+0x20*i+9)
		add("sc"+s, 1, sorSEnts+0x20*i+0x10)
	}
	for t := int64(0); t < ssrDefN; t++ {
		s := fmt.Sprint(t - 1)
		add("dc"+s, 1, ssrDefs+ssrDefSize*t+0x3e)
		add("df"+s, 1, ssrDefs+ssrDefSize*t+0x41)
		add("dk"+s, 1, ssrDefs+ssrDefSize*t+0x66)
		add("dl"+s, 1, ssrDefs+ssrDefSize*t+0x85)
	}
	for k := int64(0); k < 8; k++ {
		add(fmt.Sprint("pa", k), 1, sorArena+0x20000+0x106*k)
		add(fmt.Sprint("pt", k), 4, sorArena+0x10000+0x1862*k+0x89)
	}
	return
}()

type ssrOut struct {
	name string
	size int
	at   int64
}

var ssrOuts = func() (o []ssrOut) {
	add := func(n string, size int, at int64) { o = append(o, ssrOut{n, size, at}) }
	add("c7b0", 4, ssrRst+0x10)
	for a := int64(1); a <= 3; a++ {
		for k := int64(0); k < 4; k++ {
			add(fmt.Sprintf("a%d_%d", a, k), 4, ssrRst+0x100+0x20*(a-1)+4*k)
		}
	}
	add("c70", 4, ssrRst+0x14)
	add("c70a", 4, ssrRst+0x200)
	add("c80", 4, ssrRst+0x18)
	add("c80a", 4, ssrRst+0x220)
	for _, x := range []struct {
		o int64
		s int
	}{{0, 4}, {4, 2}, {0xa, 4}, {0xe, 4}, {0x12, 4}, {0x16, 4}} {
		add(fmt.Sprintf("mer%x", x.o), x.s, sorMer+x.o)
	}
	for _, x := range []int64{0x2c6, 0x2ca, 0x2ce} {
		add(fmt.Sprintf("rec%x", x), 4, sorRec+x)
	}
	for _, x := range []int64{0, 4, 8, 0xc} {
		add(fmt.Sprintf("ol%x", x), 4, sorOList+x)
	}
	for k := int64(0); k < 12; k++ {
		add(fmt.Sprintf("od%d", k), 4, sorOData+4*k)
	}
	for j := int64(0); j < sorOffN; j++ {
		for _, x := range []int64{0, 4, 8, 0xc, 0x10} {
			add(fmt.Sprintf("oe%d.%x", j, x), 4, sorOEnts+0x20*j+x)
		}
	}
	for _, x := range []int64{0, 4, 8, 0xc} {
		add(fmt.Sprintf("sl%x", x), 4, sorSList+x)
	}
	for k := int64(0); k < 12; k++ {
		add(fmt.Sprintf("sdd%d", k), 4, sorSData+4*k)
	}
	for i := int64(0); i < sorStkN; i++ {
		for _, x := range []int64{0, 4, 8, 0xc, 0x10} {
			add(fmt.Sprintf("se%d.%x", i, x), 4, sorSEnts+0x20*i+x)
		}
	}
	return
}()

var ssrIn, ssrOutNames = func() (in, out []string) {
	for _, f := range ssrSpec {
		in = append(in, f.name)
	}
	out = []string{"ret"}
	for _, o := range ssrOuts {
		out = append(out, o.name)
	}
	return
}()

func init() {
	Nodes["0055afa0"] = Node{Addr: "0055afa0", In: ssrIn, Out: ssrOutNames, F: shopTimedRestockNode,
		Emu: func(in []int64, r Rand) []int64 { return ssrNode(in, r, false) }}
}

func shopTimedRestockNode(in []int64, r Rand) []int64 { return ssrNode(in, r, true) }

func ssrNode(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(ssrIn) {
		panic("shop_timed_restock: wrong input count")
	}
	w := &cgW{m: map[int64]byte{}}
	sorSetup(w)
	ssrSetup(w)
	// The case's inputs, in input order (a later write to the same byte wins).
	for i, f := range ssrSpec {
		if f.size > 0 {
			w.wr(f.at, f.size, in[i])
		}
	}
	ssrCheck(w)
	out := []int64{ssrRun(w, r, real, in[0]&0xff)}
	for _, o := range ssrOuts {
		out = append(out, w.rd(o.at, o.size))
	}
	return out
}

// ssrSetup writes the decoys around the fields the rule reads: the neighbour
// bytes of every item definition field (a wrong stride reads another
// definition's input).
func ssrSetup(w *cgW) {
	for t := int64(0); t < ssrDefN; t++ {
		for _, o := range []int64{0x3d, 0x3f, 0x40, 0x42, 0x65, 0x67, 0x84, 0x86} {
			w.p8(ssrDefs+ssrDefSize*t+o, 0xa0+o+t)
		}
	}
}

// ssrCheck panics past the fixture caps.
func ssrCheck(w *cgW) {
	if p := w.u32(sorRec + 0x2ca); p != 0 && p != sorOList {
		panic("shop_timed_restock: offer list pointer is not the fixture object")
	}
	if p := w.u32(sorMer + 0xa); p != 0 && p != sorSList {
		panic("shop_timed_restock: stock list pointer is not the fixture object")
	}
	if w.u32(sorRec+0x2ca) != 0 && w.u32(sorOList+8) > sorOffN {
		panic("shop_timed_restock: offer list count past the fixture")
	}
	if w.u32(sorMer+0xa) != 0 && w.u32(sorSList+8) > sorStkN {
		panic("shop_timed_restock: stock list count past the fixture")
	}
	for i := int64(0); i < sorStkN; i++ {
		if p := w.u32(sorSData + 4*i); p != 0 && p != sorSEnts+0x20*i {
			panic("shop_timed_restock: stock slot pointer is not the fixture entry")
		}
	}
}

// ssrDef reads byte off of item definition id (-1..5).
func ssrDef(w *cgW, id, off int64) int64 {
	if id < -1 || id >= ssrDefN-1 {
		panic("shop_timed_restock: item id outside the -1..5 definition fixture")
	}
	return w.u8(ssrDefs + ssrDefSize*(id+1) + off)
}

func ssrRun(w *cgW, r Rand, real bool, force int64) int64 {
	now := w.u32(ssrNow)
	// 1. normalize the stock entries with pieces
	for i, n := int64(0), sorCount(w, w.u32(sorMer+0xa)); i < n; i++ {
		e := sorGet(w, w.u32(sorMer+0xa), i)
		if e == 0 || w.u8(e+0x10) == 0 {
			continue
		}
		id := w.rs(e+4, 4)
		kind := ssrDef(w, id, 0x66) // read for every entry (the exe reads it after 0x522e80)
		c := itemCountNormalizationNode([]int64{id, kind, w.u8(e + 8), w.u8(e + 9)}, nil)
		w.p8(e+8, c[0])
		w.p8(e+9, c[1])
		if kind == 1 && w.u8(e+0x10) > w.u8(e+8) {
			w.p8(e+8, w.u8(e+0x10))
			w.p8(e+0x10, 1)
		}
	}
	// 2. timed top-up of the fixed offers
	if (now-w.u32(sorMer+0xe))&0xffffffff > 0xa8c0 || force != 0 {
		w.p32(sorMer+0xe, now)
		offers := w.u32(w.u32(sorMer+6) + 0x2ca)
		calls := int64(0)
		for k, n := int64(0), sorCount(w, offers); k < n; k++ {
			oe := sorGet(w, offers, k)
			if w.u8(oe+5) == 0 {
				continue
			}
			id := w.u32(oe)
			qty, first := int64(0), int64(-1)
			for j, c := int64(0), sorCount(w, offers); j < c; j++ {
				if x := sorGet(w, offers, j); w.u32(x) == id {
					qty, first = w.u8(x+4), j
					break
				}
			}
			if first < 0 {
				panic("shop_timed_restock: offer id not in the offer list") // unreachable
			}
			if qty == 0 {
				continue
			}
			cur := int64(0)
			stock := w.u32(sorMer + 0xa)
			for q, c := int64(0), sorCount(w, stock); q < c; q++ {
				if s := sorGet(w, stock, q); s != 0 && w.u32(s+4) == id {
					cur = w.u8(s + 0x10)
					break
				}
			}
			if cur > qty>>1 {
				continue
			}
			delta := (qty - cur) & 0xff
			switch r.R(3) {
			case 0:
				delta = (delta + delta>>1) & 0xff
			case 1:
				delta -= delta >> 1
			}
			if delta == 0 {
				continue
			}
			// 0x55a7b0(merchant, id, delta)
			for a, x := range []int64{sorMer, id, delta} {
				w.p32(ssrRst+0x100+0x20*int64(a)+4*calls, x)
			}
			calls++
			w.p32(ssrRst+0x10, calls)
			if real {
				ssrRealDefs(w)
				sorRealAdd(w, r, id, delta, first)
			}
		}
	}
	// 3. daily restock
	if (now-w.u32(sorMer+0x12))&0xffffffff < 0x15180 && force == 0 {
		return 0
	}
	w.p32(ssrRst+0x200, sorMer) // 0x55ae70(merchant)
	w.p32(ssrRst+0x14, 1)
	if real {
		ssrRealTurnover(w, r)
	}
	w.p32(ssrRst+0x220, sorMer) // 0x55ab80(merchant)
	w.p32(ssrRst+0x18, 1)
	if real {
		ssrRealDefs(w)
		sorRun(w, r, true)
	}
	// entries without pieces leave the list (0x5e27c0, then the delete)
	stock := w.u32(sorMer + 0xa)
	for i, n := int64(0), sorCount(w, stock); i < n; i++ {
		if e := sorGet(w, stock, i); e != 0 && w.u8(e+0x10) == 0 {
			c, d := w.i32(stock+8), w.u32(stock)
			for j := i; j < c-1; j++ {
				w.p32(d+4*j, w.u32(d+4*(j+1)))
			}
			w.p32(stock+8, c-1)
			if i > 0 {
				i--
			}
			n = sorCount(w, stock)
		}
	}
	w.p32(sorMer+0x12, now)
	w.p32(sorMer+0xe, w.u32(ssrNow))
	return 1
}

// ssrRealDefs fills the 0x55ab80 / 0x55a7b0 callees' per-offer definition
// scratch (class, kind, level) from the offer's item definition.
func ssrRealDefs(w *cgW) {
	if w.u32(sorRec+0x2ca) == 0 {
		return
	}
	for j, n := int64(0), w.u32(sorOList+8); j < n; j++ {
		id := w.rs(sorOEnts+0x20*j, 4)
		w.p8(sorIDef+0x10*j, ssrDef(w, id, 0x3e))
		w.p8(sorIDef+0x10*j+1, ssrDef(w, id, 0x66))
		w.p8(sorIDef+0x10*j+2, ssrDef(w, id, 0x85))
	}
}

// ssrRealTurnover is 0x55ae70(merchant) run by shopStockTurnoverNode: its stock
// entries and customer (offer) entries, and the flags and stack types of
// definitions 0..5; the new quantity and piece count are written back.
func ssrRealTurnover(w *cgW, r Rand) {
	stock := w.u32(sorMer + 0xa)
	offers := w.u32(w.u32(sorMer+6) + 0x2ca)
	sn, on := sorCount(w, stock), sorCount(w, offers)
	if sn > 3 || on > 3 {
		panic("shop_timed_restock: F: the 0x55ae70 node covers at most 3 stock and 3 offer entries")
	}
	in := make([]int64, 34)
	in[0], in[1], in[2], in[3] = stock, sn, offers, on
	for k := int64(0); k < sn; k++ {
		e := sorGet(w, stock, k)
		if e == 0 { // a null slot is skipped like a stock entry without pieces
			continue
		}
		gate, item, units := w.u32(e), w.u32(e+4), w.u8(e+0x10)
		if units != 0 && gate == 0 && item > 5 {
			panic("shop_timed_restock: F: the 0x55ae70 node covers item ids 0..5")
		}
		copy(in[4+4*k:], []int64{gate, item, w.u8(e + 8), units})
	}
	for j := int64(0); j < on; j++ {
		oe := sorGet(w, offers, j)
		in[16+2*j], in[17+2*j] = w.u32(oe), w.u8(oe+5)
	}
	for t := int64(0); t < 6; t++ {
		in[22+t], in[28+t] = ssrDef(w, t, 0x41), ssrDef(w, t, 0x66)
	}
	out := shopStockTurnoverNode(in, r)
	for k := int64(0); k < sn; k++ {
		if e := sorGet(w, stock, k); e != 0 {
			w.p8(e+8, out[2*k])
			w.p8(e+0x10, out[2*k+1])
		}
	}
}
