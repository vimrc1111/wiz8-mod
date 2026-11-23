package calc

import "fmt"

// shop_stock_add 0x55a7b0 (merchant, itemId, count): add `count` of item
// definition `itemId` to the merchant's stock list (merchant+0xa, created on
// first use). The list is {data pointer, capacity, count}; an entry is 0x14
// bytes: +4 item id, +8 quantity byte, +0x10 piece-count byte. Item class
// (definition +0x3e) 4 appends `count` separate entries (+8=0x19, +0x10=1,
// loop count is the full u32); any other class searches the list for an entry
// with the same id (first match, null slots skipped) and else appends one,
// then kind (definition +0x66) 1 adds the low byte of count to +8 and sets
// +0x10=1, other kinds set +8=1 and add the low byte of count to +0x10.
// Returns the entry index, the item id when the class-4 loop runs zero times,
// or 0xffffffff for a null merchant.
//
// Callees: the list helpers 0x5e22c0 (new list: 16-byte header, 40-byte data
// array, capacity 10), 0x5e2480 (append, grows the array by 5 slots when
// count >= capacity, signed compare), 0x5e2870 (get), 0x5e2c70 (count) and
// the allocators run natively in the case; here they are transliterated onto
// the sparse memory of n_520d10.go. 0x51c020 (D4, notes/re/iface) is
// address-stubbed in the case: arguments 1-5 of the last call and the call
// count are outputs (stub0..5); its writes into entry+4..+0xf are therefore not
// modelled (F is the stubbed body, the callee's Go node is confined to its own
// fixture and cannot take a heap entry). Its return is ignored by the caller.
//
// The emulator heap is a bump allocator from 0x50000000 (sizes rounded up to 8,
// min 8, never reused, zeroed per path); the case dumps its first 0x600 bytes
// (h0..h383), so every allocation and stale array is compared.
//
// Fixture caps (calc panics past them): merchant and list pointers are the
// two fixture objects or null, item ids 0..15, existing entries <= 6, list
// count after any append <= 32 (so a class-4 count above 32 panics), capacity
// <= 0x7fffffff (it is a signed allocation count), heap use <= 0x600 bytes.
const (
	ssaArena = 0x20000000
	ssaMer   = ssaArena
	ssaItems = ssaArena + 0x1000
	ssaList  = ssaArena + 0x3000
	ssaData  = ssaArena + 0x3100
	ssaEnts  = ssaArena + 0x3400
	ssaStub  = ssaArena + 0x4000
	ssaHeap  = 0x50000000
	ssaWin   = 0x600
	ssaItemN = 16
	ssaEntN  = 6
	ssaSlots = 64
	ssaMaxN  = 32
)

var ssaIn, ssaOut = func() (in, out []string) {
	in = []string{"merchant", "id", "cnt", "listp", "cls", "kind", "cap", "count", "stubRet"}
	for i := 0; i < ssaEntN; i++ {
		in = append(in, fmt.Sprintf("d%d", i))
	}
	for i := 0; i < ssaEntN; i++ {
		in = append(in, fmt.Sprintf("e%d.id", i), fmt.Sprintf("e%d.q", i), fmt.Sprintf("e%d.c", i))
	}
	out = []string{"ret", "merList"}
	for _, o := range []int{0, 4, 8, 0xc, 0x10} {
		out = append(out, fmt.Sprintf("mer%x", o))
	}
	for _, o := range []int{0, 4, 8, 0xc} {
		out = append(out, fmt.Sprintf("list%x", o))
	}
	for k := 0; k < ssaSlots; k++ {
		out = append(out, fmt.Sprintf("data%d", k))
	}
	for i := 0; i < ssaEntN; i++ {
		for _, o := range []int{0, 4, 8, 0xc, 0x10} {
			out = append(out, fmt.Sprintf("ent%d.%x", i, o))
		}
	}
	for k := 0; k < 6; k++ {
		out = append(out, fmt.Sprintf("stub%d", k))
	}
	for k := 0; k < ssaWin/4; k++ {
		out = append(out, fmt.Sprintf("h%d", k))
	}
	return
}()

func init() {
	Nodes["0055a7b0"] = Node{Addr: "0055a7b0", In: ssaIn, Out: ssaOut, F: shopStockAddNode}
}

type ssaWorld struct {
	cgW
	top int64 // bump heap use
}

func (s *ssaWorld) alloc(n int64) int64 {
	sz := (n + 7) &^ 7
	if sz < 8 {
		sz = 8
	}
	at := ssaHeap + s.top
	if s.top += sz; s.top > ssaWin {
		panic("shop_stock_add: heap use past the 0x600-byte fixture window")
	}
	return at
}

// listNew is 0x5e22c0 (the memory-failure and assert blocks cannot run).
func (s *ssaWorld) listNew() int64 {
	l := s.alloc(0xc)
	s.p32(l+8, 0)
	s.p32(l, 0)
	arr := s.alloc(0x28)
	s.p32(l, arr)
	s.p32(l+4, 0xa)
	s.p32(l+8, 0)
	return l
}

// listGet is 0x5e2870: null list or index >= count (signed) gives 0.
func (s *ssaWorld) listGet(l, i int64) int64 {
	if l == 0 || i >= s.i32(l+8) {
		return 0
	}
	return s.u32(s.u32(l) + 4*i)
}

// listAdd is 0x5e2480: returns the new entry's index.
func (s *ssaWorld) listAdd(l, e int64) int64 {
	if s.i32(l+8) >= s.i32(l+4) {
		arr := s.alloc((s.u32(l+4)*4 + 0x14) & 0xffffffff)
		for i := int64(0); i < s.i32(l+8); i++ {
			s.p32(arr+4*i, s.u32(s.u32(l)+4*i))
		}
		s.p32(l, arr)
		s.p32(l+4, s.u32(l+4)+5)
	}
	c := s.u32(l + 8)
	s.p32(s.u32(l)+4*c, e)
	s.p32(l+8, c+1)
	return c
}

func shopStockAddNode(in []int64, r Rand) []int64 {
	if len(in) != len(ssaIn) {
		panic("shop_stock_add: wrong input count")
	}
	v := map[string]int64{}
	for i, n := range ssaIn {
		v[n] = in[i] & 0xffffffff
	}
	s := &ssaWorld{cgW: cgW{m: map[int64]byte{}}}
	// The case's setup: item records (decoy class/kind in every slot), merchant
	// decoys, list header, data-array decoys and entry decoys.
	s.p32(0x68516c, ssaItems)
	for k := int64(0); k < ssaItemN; k++ {
		cls, kind := int64(4), int64(2)
		if k%2 == 1 {
			cls, kind = 1+k%3, 1
		}
		s.p8(ssaItems+k*0x10d+0x3e, cls)
		s.p8(ssaItems+k*0x10d+0x66, kind)
	}
	for j, o := range []int64{0, 4, 8, 0xc, 0x10} {
		s.p32(ssaMer+o, 0x11111111*int64(j+1))
	}
	s.p32(ssaList, ssaData)
	s.p32(ssaList+0xc, 0x7e7e7e7e)
	for k := int64(ssaEntN); k < ssaSlots; k++ {
		s.p32(ssaData+4*k, 0xdead0000+k)
	}
	for i := int64(0); i < ssaEntN; i++ {
		b := ssaEnts + 0x20*i
		s.p32(b, 0xa5a5a500+i)
		s.p32(b+8, 0xb1b1b100)
		s.p32(b+0xc, 0xc3c3c3c3)
		s.p32(b+0x10, 0xd1d1d100)
	}
	// The case's inputs.
	id := v["id"]
	if id >= ssaItemN {
		panic("shop_stock_add: item id past the 16-record fixture")
	}
	if v["merchant"] != 0 && v["merchant"] != ssaMer {
		panic("shop_stock_add: merchant pointer is not the fixture object")
	}
	if v["listp"] != 0 && v["listp"] != ssaList {
		panic("shop_stock_add: list pointer is not the fixture object")
	}
	if v["cap"] > 0x7fffffff || v["count"] > ssaEntN {
		panic("shop_stock_add: list capacity or count past the fixture")
	}
	s.p32(ssaMer+10, v["listp"])
	s.p8(ssaItems+id*0x10d+0x3e, v["cls"])
	s.p8(ssaItems+id*0x10d+0x66, v["kind"])
	s.p32(ssaList+4, v["cap"])
	s.p32(ssaList+8, v["count"])
	for i := int64(0); i < ssaEntN; i++ {
		d := v[fmt.Sprintf("d%d", i)]
		if d != 0 && d != ssaEnts+0x20*i {
			panic("shop_stock_add: slot pointer is not the fixture entry")
		}
		s.p32(ssaData+4*i, d)
		b := ssaEnts + 0x20*i
		s.p32(b+4, v[fmt.Sprintf("e%d.id", i)])
		s.p8(b+8, v[fmt.Sprintf("e%d.q", i)])
		s.p8(b+0x10, v[fmt.Sprintf("e%d.c", i)])
	}
	// v["stubRet"] is the stubbed 0x51c020's return; 0x55a7b0 discards it.

	ret := int64(0xffffffff)
	if v["merchant"] != 0 {
		ret = s.run(id, v["cnt"])
	}
	var out []int64
	rd := func(a int64) { out = append(out, s.u32(a)) }
	out = append(out, ret)
	rd(ssaMer + 10)
	for _, o := range []int64{0, 4, 8, 0xc, 0x10} {
		rd(ssaMer + o)
	}
	for _, o := range []int64{0, 4, 8, 0xc} {
		rd(ssaList + o)
	}
	for k := int64(0); k < ssaSlots; k++ {
		rd(ssaData + 4*k)
	}
	for i := int64(0); i < ssaEntN; i++ {
		for _, o := range []int64{0, 4, 8, 0xc, 0x10} {
			rd(ssaEnts + 0x20*i + o)
		}
	}
	for k := int64(0); k < 6; k++ {
		rd(ssaStub + 4*k)
	}
	for k := int64(0); k < ssaWin/4; k++ {
		rd(ssaHeap + 4*k)
	}
	return out
}

func (s *ssaWorld) run(id, cnt int64) int64 {
	if s.u32(ssaMer+10) == 0 {
		s.p32(ssaMer+10, s.listNew())
	}
	list := s.u32(ssaMer + 10)
	rec := s.u32(0x68516c) + id*0x10d
	cls, kind := s.u8(rec+0x3e), s.u8(rec+0x66)
	loops := int64(1)
	if cls == 4 {
		loops = cnt
	}
	if loops == 0 {
		return id
	}
	if loops > ssaMaxN {
		panic("shop_stock_add: class-4 loop count past the 32-entry fixture")
	}
	var idx int64
	for k := int64(0); k < loops; k++ {
		var ent int64
		found := false
		if cls != 4 {
			n := s.u32(list + 8)
			for i := int64(0); i < n; i++ {
				e := s.listGet(list, i)
				if e != 0 && s.u32(e+4) == id {
					idx, ent, found = i, e, true
					break
				}
			}
		}
		if !found {
			ent = s.alloc(0x14)
			for o := int64(0); o <= 0x10; o += 4 {
				s.p32(ent+o, 0)
			}
			// 0x51c020(ent+4, id, 1, 1, 0): recorded, not run.
			for a, x := range []int64{ent + 4, id, 1, 1, 0} {
				s.p32(ssaStub+4*int64(a), x)
			}
			s.p32(ssaStub+20, s.u32(ssaStub+20)+1)
			s.p8(ent+8, 0)
			if s.u32(list+8) >= ssaMaxN {
				panic("shop_stock_add: list count past the 32-entry fixture")
			}
			idx = s.listAdd(list, ent)
		}
		switch {
		case cls == 4:
			s.p8(ent+8, 0x19)
			s.p8(ent+0x10, 1)
		case kind == 1:
			s.p8(ent+8, s.u8(ent+8)+cnt)
			s.p8(ent+0x10, 1)
		default:
			s.p8(ent+0x10, s.u8(ent+0x10)+cnt)
			s.p8(ent+8, 1)
		}
	}
	return idx
}
