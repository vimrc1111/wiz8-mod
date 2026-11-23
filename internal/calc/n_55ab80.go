package calc

import "fmt"

// shopOptionalRestockNode is 0x55ab80 shop_optional_restock(merchant): top up
// the merchant's stock of the shop's optional offers.
//
// merchant+6 is the shop record, record+0x2ca its offer list (list header
// {data pointer, capacity, count at +8}; an entry is {i32 item id, u8 quantity
// at +4, u8 fixed flag at +5}). A null or empty list does nothing. Per offer,
// in list order:
//   - an offer with a nonzero fixed flag is skipped (those are stocked by
//     0x55a630);
//   - item id 0x1fc (508) needs the story fact 0x15f: 0x506280 -> 0x5080f0 reads
//     the byte at 0x689b78+0x15f, zero skips the offer (debug logging 0x688290 is
//     off and not an input);
//   - the quantity is that of the FIRST offer with the same item id (+4 byte),
//     zero skips;
//   - cur is the +0x10 byte of the first non-null entry of the stock list
//     (merchant+0xa, null list = empty) whose +4 id equals the item id, else 0;
//     cur > quantity/2 skips;
//   - class = low byte of 0x55aad0(merchant, id), signed; 0..4 give the chance
//     0/25/50/75/100, anything else (a byte past 4, or negative) 0. Roll R(100),
//     it succeeds when R(100) < chance (the roll is made even for chance 0);
//   - on success delta = u8(quantity - cur); R(3): 0 gives u8(delta+delta/2), 1
//     gives delta-delta/2, 2 leaves it; a zero delta skips (u8 wrap: 171 -> 0),
//     else 0x55a7b0(merchant, id, delta) adds stock.
//
// Always returns 1. Stubs (case): 0x55a7b0 and 0x55aad0 are D4 formula callees,
// address-stubbed; the case outputs the arguments of every call (up to 4) and
// the call count. Their returns are case inputs: ret7b0 (ignored by the caller)
// and cls0..cls3 (call k reads cls_k; 0x55aad0 itself only returns 0..3, the
// wider domain is the stub's). Because the stub adds nothing, the stock list
// stays as set up for the whole loop (the original grows it between offers).
// The list helpers 0x5e2c70 (count; null = 0) and 0x5e2870 (get; null or index
// >= count gives 0) run natively and are folded in.
//
// Unreachable in the original: the flag test at 0x55aca1 (the flag is zero past
// the first test; cov-exempt 55aca6) and the "id not found in the offer list"
// exit (the entry itself is in the list; cov-exempt 55ac2e).
//
// Emu and F share this body. Emu (what TestEmu replays) is the case with the two
// stubs above. F calls the real callees instead: shopStockAddNode (0x55a7b0)
// on a callee world built from the node's stock list, so stock really grows
// between offers, and partyAverageLevelNode plus sisClassOf (0x55aad0, which
// n_55aad0.go's two-slot node cannot cover). F ignores cls0..cls3 and ret7b0 and
// reads the F-only inputs: ic/ik/il_j are the item definition's class (+0x3e),
// kind (+0x66) and level (+0x85) of offer j's item id (the stubbed callees do
// not read them in the emulator, so the case parks them in a scratch object),
// pa/pt_k the party slots (presence byte, level at character +0x89).
// F's extra caps (panics): stock entries after growth <= 6 and at most 32
// entries per class-4 add (shopStockAddNode's fixture), at most 16 distinct
// item ids (ids are remapped onto the callee's 16 records).
//
// Fixture caps (calc panics past them): offer list count <= 4, stock list count
// <= 6, list pointers 0 or the fixture lists, stock data slots 0 or their
// fixture entry (offer slots are always the fixture entries).
const (
	sorArena = 0x20000000
	sorMer   = sorArena
	sorRec   = sorArena + 0x100
	sorOList = sorArena + 0x200
	sorOData = sorArena + 0x300
	sorOEnts = sorArena + 0x400
	sorSList = sorArena + 0x600
	sorSData = sorArena + 0x700
	sorSEnts = sorArena + 0x800
	sorStub  = sorArena + 0x1000
	sorIDef  = sorArena + 0x1400
	sorFact  = 0x689b78 + 0x15f
	sorOffN  = 4
	sorStkN  = 6
)

// sorField is one case input: name, byte size and address.
type sorField struct {
	name string
	size int
	at   int64
}

var sorSpec = func() (f []sorField) {
	add := func(n string, size int, at int64) { f = append(f, sorField{n, size, at}) }
	add("fact", 1, sorFact)
	add("ret7b0", 4, sorStub+0x20)
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
		add("sid"+s, 4, sorSEnts+0x20*i+4)
		add("sc"+s, 1, sorSEnts+0x20*i+0x10)
	}
	for k := int64(0); k < sorOffN; k++ {
		add(fmt.Sprint("cls", k), 4, sorStub+0x40+4*k)
	}
	for j := int64(0); j < sorOffN; j++ {
		s := fmt.Sprint(j)
		add("ic"+s, 1, sorIDef+0x10*j)
		add("ik"+s, 1, sorIDef+0x10*j+1)
		add("il"+s, 1, sorIDef+0x10*j+2)
	}
	for k := int64(0); k < 8; k++ {
		add(fmt.Sprint("pa", k), 1, sorArena+0x20000+0x106*k)
		add(fmt.Sprint("pt", k), 4, sorArena+0x10000+0x1862*k+0x89)
	}
	return
}()

// sorOutAt lists the unchanged words echoed after the stub records, in the
// case's output order: name, address (size 4 unless noted in sorOutSize).
type sorOut struct {
	name string
	size int
	at   int64
}

var sorOuts = func() (o []sorOut) {
	add := func(n string, size int, at int64) { o = append(o, sorOut{n, size, at}) }
	add("c7b0", 4, sorStub+0x10)
	for a := int64(1); a <= 3; a++ {
		for k := int64(0); k < 4; k++ {
			add(fmt.Sprintf("a%d_%d", a, k), 4, sorStub+0x100+0x20*(a-1)+4*k)
		}
	}
	add("caad0", 4, sorStub+0x14)
	for a := int64(1); a <= 2; a++ {
		for k := int64(0); k < 4; k++ {
			add(fmt.Sprintf("b%d_%d", a, k), 4, sorStub+0x200+0x20*(a-1)+4*k)
		}
	}
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

var sorIn, sorOutNames = func() (in, out []string) {
	for _, f := range sorSpec {
		in = append(in, f.name)
	}
	out = []string{"ret"}
	for _, o := range sorOuts {
		out = append(out, o.name)
	}
	return
}()

func init() {
	Nodes["0055ab80"] = Node{Addr: "0055ab80", In: sorIn, Out: sorOutNames, F: shopOptionalRestockNode,
		Emu: func(in []int64, r Rand) []int64 { return sorNode(in, r, false) }}
}

func shopOptionalRestockNode(in []int64, r Rand) []int64 { return sorNode(in, r, true) }

func sorNode(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(sorIn) {
		panic("shop_optional_restock: wrong input count")
	}
	w := &cgW{m: map[int64]byte{}}
	sorSetup(w)
	// The case's inputs, in input order (a later write to the same byte wins).
	for i, f := range sorSpec {
		w.wr(f.at, f.size, in[i])
	}
	sorCheck(w)
	sorRun(w, r, real)
	out := []int64{1}
	for _, o := range sorOuts {
		out = append(out, w.rd(o.at, o.size))
	}
	return out
}

// sorSetup writes the case's fixed setup: pointers and decoys around every
// field the rule reads.
func sorSetup(w *cgW) {
	w.p32(sorMer, 0x11110000)
	w.wr(sorMer+4, 2, 0x2222)
	w.p32(sorMer+6, sorRec)
	w.p32(sorMer+0xe, 0x33333333)
	w.p32(sorMer+0x12, 0x44444444)
	w.p32(sorMer+0x16, 0x55555555)
	w.p32(sorRec+0x2c6, 0x66666666)
	w.p32(sorRec+0x2ce, 0x77777777)
	w.p32(sorOList, sorOData)
	w.p32(sorOList+4, 10)
	w.p32(sorOList+0xc, 0x7e7e7e7e)
	w.p32(sorSList, sorSData)
	w.p32(sorSList+4, 10)
	w.p32(sorSList+0xc, 0x7e7e7e7e)
	for k := int64(0); k < 12; k++ {
		v := int64(0xdead0000 + k)
		if k < sorOffN {
			v = sorOEnts + 0x20*k
		}
		w.p32(sorOData+4*k, v)
		if k >= sorStkN {
			w.p32(sorSData+4*k, 0xdead0000+k)
		}
	}
	for j := int64(0); j < sorOffN; j++ {
		b := sorOEnts + 0x20*j
		w.wr(b+6, 2, 0xa5a5)
		w.p32(b+8, 0xb1b1b100+j)
		w.p32(b+0xc, 0xc3c3c300+j)
		w.p32(b+0x10, 0xd1d1d100+j)
	}
	for i := int64(0); i < sorStkN; i++ {
		b := sorSEnts + 0x20*i
		w.p32(b, 0xa5a5a500+i)
		w.p32(b+8, 0xc4000000+i)
		w.p32(b+0xc, 0xc3c3c300+i)
		w.p32(b+0x10, 0xd1d1d100+i)
	}
}

// sorCheck panics past the fixture caps.
func sorCheck(w *cgW) {
	if p := w.u32(sorRec + 0x2ca); p != 0 && p != sorOList {
		panic("shop_optional_restock: offer list pointer is not the fixture object")
	}
	if p := w.u32(sorMer + 0xa); p != 0 && p != sorSList {
		panic("shop_optional_restock: stock list pointer is not the fixture object")
	}
	if w.u32(sorRec+0x2ca) != 0 && w.u32(sorOList+8) > sorOffN {
		panic("shop_optional_restock: offer list count past the fixture")
	}
	if w.u32(sorMer+0xa) != 0 && w.u32(sorSList+8) > sorStkN {
		panic("shop_optional_restock: stock list count past the fixture")
	}
	for i := int64(0); i < sorStkN; i++ {
		if p := w.u32(sorSData + 4*i); p != 0 && p != sorSEnts+0x20*i {
			panic("shop_optional_restock: stock slot pointer is not the fixture entry")
		}
	}
}

// sorCount is 0x5e2c70; sorGet is 0x5e2870 (index >= signed count gives 0).
func sorCount(w *cgW, l int64) int64 {
	if l == 0 {
		return 0
	}
	return w.u32(l + 8)
}

func sorGet(w *cgW, l, i int64) int64 {
	if l == 0 || i >= w.i32(l+8) {
		return 0
	}
	return w.u32(w.u32(l) + 4*i)
}

func sorRun(w *cgW, r Rand, real bool) {
	offers := w.u32(w.u32(sorMer+6) + 0x2ca)
	n := sorCount(w, offers) // read once (exe 0x55ab92)
	var calls7b0, callsAad0 int64
	for i := int64(0); i < n; i++ {
		e := sorGet(w, offers, i)
		if w.u8(e+5) != 0 {
			continue
		}
		id := w.u32(e)
		if id == 0x1fc && w.u8(sorFact) == 0 {
			continue
		}
		qty, found, first := int64(0), false, int64(0)
		for j := int64(0); j < n && !found; j++ {
			if x := sorGet(w, offers, j); w.u32(x) == id {
				qty, found, first = w.u8(x+4), true, j
			}
		}
		if !found {
			panic("shop_optional_restock: offer id not in the offer list") // unreachable
		}
		if qty == 0 {
			continue
		}
		stock := w.u32(sorMer + 0xa)
		cur := int64(0)
		for k, c := int64(0), sorCount(w, stock); k < c; k++ {
			if s := sorGet(w, stock, k); s != 0 && w.u32(s+4) == id {
				cur = w.u8(sorGet(w, stock, k) + 0x10)
				break
			}
		}
		if cur > qty>>1 {
			continue
		}
		// Stubbed 0x55aad0(merchant, id): the k-th call returns cls_k.
		w.p32(sorStub+0x200+4*callsAad0, sorMer)
		w.p32(sorStub+0x220+4*callsAad0, id)
		class := int64(int8(w.u32(sorStub + 0x40 + 4*callsAad0)))
		if real {
			class = sorRealClass(w, first)
		}
		callsAad0++
		w.p32(sorStub+0x14, callsAad0)
		chance := int64(0)
		if class >= 0 && class <= 4 {
			chance = 25 * class
		}
		if int64(r.R(100)) >= chance {
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
		// Stubbed 0x55a7b0(merchant, id, count).
		for a, x := range []int64{sorMer, id, delta} {
			w.p32(sorStub+0x100+0x20*int64(a)+4*calls7b0, x)
		}
		calls7b0++
		w.p32(sorStub+0x10, calls7b0)
		if real {
			sorRealAdd(w, r, id, delta, first)
		}
	}
}

// sorRealClass is 0x55aad0(merchant, id) for the item definition of offer j.
func sorRealClass(w *cgW, j int64) int64 {
	pair := make([]int64, 0, 16)
	for k := int64(0); k < 8; k++ {
		pair = append(pair, w.u8(sorArena+0x20000+0x106*k), w.u32(sorArena+0x10000+0x1862*k+0x89))
	}
	return sisClassOf(partyAverageLevelNode(pair, nil)[0], w.rs(sorIDef+0x10*j+2, 1))
}

// sorRealAdd is 0x55a7b0(merchant, id, count) run by shopStockAddNode on a
// callee world made of the node's stock list; the result is written back.
func sorRealAdd(w *cgW, r Rand, id, cnt, offer int64) {
	list := w.u32(sorMer + 0xa)
	n := sorCount(w, list)
	// ids -> the callee's 16 records, by equality only
	ids := map[int64]int64{}
	rev := []int64{}
	m := func(x int64) int64 {
		if v, ok := ids[x]; ok {
			return v
		}
		if len(rev) >= ssaItemN {
			panic("shop_optional_restock: more than 16 distinct item ids for the 0x55a7b0 callee")
		}
		ids[x] = int64(len(rev))
		rev = append(rev, x)
		return ids[x]
	}
	cid := m(id)
	v := map[string]int64{"merchant": ssaMer, "id": cid, "cnt": cnt, "cls": w.u8(sorIDef + 0x10*offer), "kind": w.u8(sorIDef + 0x10*offer + 1),
		"cap": 10, "count": n}
	if list != 0 {
		v["listp"] = ssaList
	}
	for i := int64(0); i < n; i++ {
		d := w.u32(sorSData + 4*i)
		if d != 0 {
			v["d"+fmt.Sprint(i)] = ssaEnts + 0x20*i
			v[fmt.Sprintf("e%d.id", i)] = m(w.u32(d + 4))
		}
		v[fmt.Sprintf("e%d.q", i)] = w.u8(sorSEnts + 0x20*i + 8)
		v[fmt.Sprintf("e%d.c", i)] = w.u8(sorSEnts + 0x20*i + 0x10)
	}
	vec := make([]int64, len(ssaIn))
	for i, name := range ssaIn {
		vec[i] = v[name]
	}
	out := shopStockAddNode(vec, nil)
	at := map[string]int64{}
	for i, name := range ssaOut {
		at[name] = out[i]
	}
	cnt2 := at["list8"]
	if cnt2 > sorStkN {
		panic("shop_optional_restock: stock list grew past the 6-entry fixture")
	}
	if list == 0 {
		w.p32(sorMer+0xa, sorSList)
		w.p32(sorSList, sorSData)
		w.p32(sorSList+4, 10)
	}
	word := func(p, o int64) int64 {
		if p >= ssaHeap {
			return at[fmt.Sprintf("h%d", (p-ssaHeap+o)/4)]
		}
		return at[fmt.Sprintf("ent%d.%x", (p-ssaEnts)/0x20, o)]
	}
	for k := int64(0); k < cnt2; k++ {
		p := at[fmt.Sprintf("data%d", k)]
		if k < n {
			if p != 0 {
				w.p32(sorSEnts+0x20*k+4, rev[word(p, 4)])
				w.p8(sorSEnts+0x20*k+8, word(p, 8))
				w.p8(sorSEnts+0x20*k+0x10, word(p, 0x10))
			}
			continue
		}
		b := sorSEnts + 0x20*k
		w.p32(sorSData+4*k, b)
		w.p32(b, word(p, 0))
		w.p32(b+4, rev[word(p, 4)])
		w.p32(b+8, word(p, 8))
		w.p32(b+0xc, word(p, 0xc))
		w.p32(b+0x10, word(p, 0x10))
	}
	w.p32(sorSList+8, cnt2)
}
