package calc

import "fmt"

// monster_drop_generation 0x4f8cb0 (cdecl: monster, container item id). Builds
// the drop list of a dead monster (notes/re/loot.md:19, :29) into a local
// pointer list (0x4f9580, 5 slots), then:
//   - NPC template (template +0xd0 bit 0): the NPC record 0x50a4a0 picks
//     (monster +0x2f1 indexes the NPC list at *0x689f94, else record 0). Every
//     slot with item +0x30+2j != -1 and weight +0x115+j == 0 drops; then
//     K = dice(+0x10e), at most the remaining slots, more slots drop by
//     R(40) slot / R(sum of remaining weights) acceptance; dropped slots
//     become -1. Money is the dword +0x80.
//   - Other templates: 8 slots at template +0x1f3 (10 bytes: kind, u16 gate,
//     u16 id, chance, dice); a slot fires when its gate is nonzero and
//     R(100) <= chance; dice(slot+6) times it drops item id (kind 0) or
//     calls item_pool_selection 0x4f88f0(list, id, count) (kind 1). Money is
//     dice(template +0x243).
//   - money != 0: 0x51bea0 adds it to the gold 0x685189 and to the dword
//     0x686b7d+33*[0x686a70] when that index is below 0x2f.
//   - more than one object: a container (item arg2, or 0x23c when arg2 is -1;
//     placed with kind 2 or 3) joins the world list and the objects are chained
//     on its +0x31; exactly one: it joins the world list *0x683fb5 at the
//     monster position.
//
// Objects come from 0x4f6b90: operator new(0xad), 0x520070 reset of the item
// instance at +9, ordinal 0x6874c2 at +0, position at +0x15, kind at +0x25,
// -1 at +0x39, 0x4f93d0, then 0x51fe30 moves the 0x51c020 instance in (byte
// +0xa cleared). Lists grow by 0x4addf0 (to count+1) and 0x5e2480 (cap+5).
// The emulator's bump heap (notes/re/emu.md) is modelled byte for byte and the
// first mdgHeapWords words are output.
//
// Callees: 0x517970 (dice), 0x51c020 (item instance), 0x4f88f0 (item pool),
// 0x520d10 (D4 formulas) and 0x4f93d0 (registered 排除) are address stubs in
// re/emu/cases/004f8cb0.json. Dice returns, the instance bytes and the list
// that 0x4f88f0 hands back (count, cap, the fixture array larr) are inputs;
// arguments and counts are outputs (stack addresses are not recorded). The
// rest runs natively and is transliterated here. Fixture caps (panic): 3
// templates, 2 NPC records, world list cap 8, item-pool list cap 8, 41
// objects, 40 item-pool calls, party index 0x686a70 >= -1 (the game stores only -1 or
// non-negative values there, see n_55b730.go). Assert-only paths panic.
const (
	mdgBase      = 0x20000000
	mdgHeap      = 0x50000000 // emu bump heap
	mdgHeapWords = 0x1000
	mdgMon       = mdgBase + 0x0
	mdgTmpl      = mdgBase + 0x1000 // stride 0x300
	mdgNPC       = mdgBase + 0x2000 // stride 0x200
	mdgGlist     = mdgBase + 0x3000
	mdgGdata     = mdgBase + 0x3100
	mdgLarr      = mdgBase + 0x3800
	mdgL0        = mdgBase + 0x4000 // l_k = mdgL0 + 0x100*k
	mdgTemplates = 3
	mdgNPCs      = 2
	mdgSlots     = 40 // NPC item slots, exe 0x4f8e02
	mdgDropSlots = 8  // template drop slots, exe 0x4f8f6d
	mdgListCap   = 8  // larr and gdata entries
	mdgMaxObj    = 41
	mdgMaxPool   = 40
	mdgDice      = 9
	mdgBoxItem   = 0x23c // exe 0x4f90b5
	mdgObjSize   = 0xad  // exe 0x4f6b91
	mdgNoNPC     = 0xfa  // template +0xcd, exe 0x50a4bb
)

var mdgIn, mdgOut = func() (in, out []string) {
	in = []string{"monArg", "p2", "tid", "npcIdx", "pos0", "pos1", "pos2", "nlist.count", "ordinal", "gold", "partyIdx"}
	for i := -1; i < 47; i++ {
		in = append(in, fmt.Sprintf("acc%d", i))
	}
	in = append(in, "glist.cap", "glist.count", "t.flags", "t.cd")
	for j := 0; j < mdgDropSlots; j++ {
		for _, f := range []string{"kind", "ne", "id", "c", "Z", "N", "S"} {
			in = append(in, fmt.Sprintf("s%d.%s", j, f))
		}
	}
	in = append(in, "m.Z", "m.N", "m.S")
	for n := 0; n < mdgNPCs; n++ {
		in = append(in, fmt.Sprintf("npc%d.money", n), fmt.Sprintf("npc%d.Z", n), fmt.Sprintf("npc%d.N", n), fmt.Sprintf("npc%d.S", n))
		for j := 0; j < mdgSlots; j++ {
			in = append(in, fmt.Sprintf("npc%d.item%d", n, j))
		}
		for j := 0; j < mdgSlots; j++ {
			in = append(in, fmt.Sprintf("npc%d.w%d", n, j))
		}
	}
	for k := 0; k < mdgDice; k++ {
		in = append(in, fmt.Sprintf("dice%d", k))
	}
	in = append(in, "inst0", "inst4", "inst8", "pool.count", "pool.cap", "f.itemsBase")
	for j := 0; j < iscItems; j++ {
		for _, f := range []string{"kind", "z", "n", "s", "attr"} {
			in = append(in, fmt.Sprintf("f.i%d.%s", j, f))
		}
	}

	out = []string{"ordinal", "gold"}
	for i := -1; i < 47; i++ {
		out = append(out, fmt.Sprintf("acc%d", i))
	}
	out = append(out, "glist.data", "glist.cap", "glist.count")
	for k := 0; k < mdgListCap; k++ {
		out = append(out, fmt.Sprintf("gdata%d", k))
	}
	for k := 0; k < mdgListCap; k++ {
		out = append(out, fmt.Sprintf("larr%d", k))
	}
	for k := 0; k < mdgListCap; k++ {
		for _, off := range []int{0x15, 0x19, 0x1d, 0x31} {
			out = append(out, fmt.Sprintf("l%d.%x", k, off))
		}
	}
	for n := 0; n < mdgNPCs; n++ {
		for j := 0; j < mdgSlots; j++ {
			out = append(out, fmt.Sprintf("npc%d.item%d", n, j))
		}
	}
	out = append(out, "dice.calls")
	for k := 0; k < mdgDice; k++ {
		out = append(out, fmt.Sprintf("dice.arg%d", k))
	}
	out = append(out, "mk.calls")
	for a := 2; a <= 5; a++ {
		for k := 0; k < mdgMaxObj; k++ {
			out = append(out, fmt.Sprintf("mk.a%d.%d", a, k))
		}
	}
	out = append(out, "pool.calls")
	for a := 2; a <= 3; a++ {
		for k := 0; k < mdgMaxPool; k++ {
			out = append(out, fmt.Sprintf("pool.a%d.%d", a, k))
		}
	}
	out = append(out, "place.calls")
	for k := 0; k < mdgMaxObj; k++ {
		out = append(out, fmt.Sprintf("place.a1.%d", k))
	}
	out = append(out, "refresh.calls")
	for a := 2; a <= 3; a++ {
		for k := 0; k < 3*mdgMaxObj; k++ {
			out = append(out, fmt.Sprintf("refresh.a%d.%d", a, k))
		}
	}
	for w := 0; w < mdgHeapWords; w++ {
		out = append(out, fmt.Sprintf("heap%d", w))
	}
	return
}()

func init() {
	Nodes["004f8cb0"] = Node{Addr: "004f8cb0", In: mdgIn, Out: mdgOut, F: monsterDropGenerationNode,
		Emu: func(in []int64, r Rand) []int64 { return monsterDrop(in, r, false) }}
}

// monsterDropGenerationNode is F: dice rolls run diceNode on the dice records
// the node passes, and item instances run itemStackChargeInitNode on the f.*
// item definitions (its own charge dice rolled with diceNode). The item-pool
// branch panics in F: itemPoolSelectionNode reports counts and the last id,
// not the objects it appends, which this node chains and outputs.
func monsterDropGenerationNode(in []int64, r Rand) []int64 { return monsterDrop(in, r, true) }

// mdgMem is the modelled memory: arena fixture fields and the heap window.
type mdgMem map[int64]byte

func (m mdgMem) put(a int64, n int, v int64) {
	for i := 0; i < n; i++ {
		m[a+int64(i)] = byte(v >> (8 * uint(i)))
	}
}
func (m mdgMem) wr(a int64, n int, v int64) {
	for i := 0; i < n; i++ {
		if _, ok := m[a+int64(i)]; !ok {
			panic(fmt.Sprintf("monster_drop_generation: write to unmodelled address %#x", a+int64(i)))
		}
	}
	m.put(a, n, v)
}
func (m mdgMem) rd(a int64) int64 {
	var v int64
	for i := 3; i >= 0; i-- {
		b, ok := m[a+int64(i)]
		if !ok {
			panic(fmt.Sprintf("monster_drop_generation: read of unmodelled address %#x", a+int64(i)))
		}
		v = v<<8 | int64(b)
	}
	return v
}

func monsterDrop(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(mdgIn) {
		panic("monster_drop_generation: wrong input count")
	}
	at := make(map[string]int, len(mdgIn))
	for i, n := range mdgIn {
		at[n] = i
	}
	rng := func(n string, lo, hi int64) int64 {
		v := in[at[n]]
		if v < lo || v > hi {
			panic(fmt.Sprintf("monster_drop_generation: %s out of range", n))
		}
		return v
	}
	u32 := func(n string) int64 { return rng(n, 0, 0xffffffff) }
	i32 := func(n string) int64 { return rng(n, -0x80000000, 0x7fffffff) }
	u8 := func(n string) int64 { return rng(n, 0, 0xff) }
	u16 := func(n string) int64 { return rng(n, 0, 0xffff) }
	i16 := func(n string) int64 { return rng(n, -0x8000, 0x7fff) }
	i8 := func(n string) int64 { return rng(n, -0x80, 0x7f) }

	m := mdgMem{}
	for a := int64(0); a < 4*mdgHeapWords; a++ {
		m[mdgHeap+a] = 0
	}
	for k := int64(0); k < mdgListCap; k++ {
		l := mdgL0 + 0x100*k
		m.put(mdgLarr+4*k, 4, l)
		m.put(mdgGdata+4*k, 4, 0x66660000+k) // case setup decoys
		m.put(l+0x15, 4, 0x11110000+k)
		m.put(l+0x19, 4, 0x22220000+k)
		m.put(l+0x1d, 4, 0x33330000+k)
		m.put(l+0x21, 0x10, 0) // +0x21..+0x30 are not written by any path
		m.put(l+0x31, 4, 0x44440000+k)
	}
	gcap, gcount := i32("glist.cap"), i32("glist.count")
	if gcount < 0 || gcount > gcap || gcap > mdgListCap {
		panic("monster_drop_generation: fixture world list holds 0 <= count <= cap <= 8")
	}
	m.put(mdgGlist, 4, mdgGdata)
	m.put(mdgGlist+4, 4, gcap)
	m.put(mdgGlist+8, 4, gcount)

	ordinal, gold, party := u32("ordinal"), u32("gold"), i32("partyIdx")
	var acc [48]int64
	for i := range acc {
		acc[i] = u32(fmt.Sprintf("acc%d", i-1))
	}
	var npcItem [mdgNPCs][mdgSlots]int64
	for n := 0; n < mdgNPCs; n++ {
		for j := 0; j < mdgSlots; j++ {
			npcItem[n][j] = i16(fmt.Sprintf("npc%d.item%d", n, j))
		}
	}
	pos := [3]int64{u32("pos0"), u32("pos1"), u32("pos2")}
	p2 := u32("p2")
	for _, n := range []string{"t.flags", "t.cd", "m.N", "m.S", "npc0.N", "npc0.S", "npc1.N", "npc1.S"} {
		u8(n)
	}
	for _, n := range []string{"m.Z", "npc0.Z", "npc1.Z"} {
		i16(n)
	}

	var heapTop int64 = mdgHeap
	alloc := func(n int64) int64 {
		p := heapTop
		heapTop += max(8, (n+7)&^7)
		if heapTop > mdgHeap+4*mdgHeapWords {
			panic("monster_drop_generation: heap past the output window")
		}
		return p
	}

	var diceRet [mdgDice]int64
	for k := range diceRet {
		diceRet[k] = i32(fmt.Sprintf("dice%d", k))
	}
	var diceArgs []int64
	// dice: 0x517970 on the record at ptr whose Z,N,S are the named inputs.
	dice := func(ptr int64, z, n, s string) int64 {
		k := len(diceArgs)
		if k >= mdgDice {
			panic("monster_drop_generation: more than 9 dice calls")
		}
		diceArgs = append(diceArgs, ptr)
		if real {
			return int64(int32(diceNode([]int64{in[at[z]], in[at[n]], in[at[s]]}, r)[0]))
		}
		return diceRet[k]
	}

	inst := [3]int64{u32("inst0"), u32("inst4"), u32("inst8")}
	var mkArgs [][4]int64
	// mk: 0x51c020(&inst, id, 0, 0, 0); returns the 12 instance bytes.
	mk := func(id int64) [12]byte {
		if len(mkArgs) >= mdgMaxObj {
			panic("monster_drop_generation: more than 41 objects")
		}
		mkArgs = append(mkArgs, [4]int64{id, 0, 0, 0})
		v := inst
		if real {
			v = mdgRealInstance(in, at, id, r)
		}
		var b [12]byte
		for i := 0; i < 12; i++ {
			b[i] = byte(v[i/4] >> (8 * uint(i%4)))
		}
		return b
	}

	var placeArgs, refreshArgs []int64
	refresh := func() { refreshArgs = append(refreshArgs, 0) } // 0x520d10(p, 0, 1)

	// worldAdd: 0x5e2480(*0x683fb5, p).
	worldAdd := func(p int64) {
		data, cp, cnt := m.rd(mdgGlist), int64(int32(m.rd(mdgGlist+4))), int64(int32(m.rd(mdgGlist+8)))
		if cnt >= cp {
			nb := alloc((cp*4 + 0x14) & 0xffffffff)
			for i := int64(0); i < cnt; i++ {
				m.wr(nb+4*i, 4, m.rd(data+4*i))
			}
			data = nb
			m.wr(mdgGlist, 4, nb)
			m.wr(mdgGlist+4, 4, cp+5)
		}
		m.wr(data+4*cnt, 4, p)
		m.wr(mdgGlist+8, 4, cnt+1)
	}

	// place: 0x4f6b90(&inst, &pos, kind, addWorld).
	place := func(b [12]byte, kind int64, addWorld bool) int64 {
		o := alloc(mdgObjSize)
		m.wr(o+9, 4, -1) // 0x520070: instance reset, then 0x520d10
		m.wr(o+13, 8, 0)
		refresh()
		m.wr(o, 4, ordinal)
		ordinal = (ordinal + 1) & 0xffffffff
		m.wr(o+8, 1, 0)
		m.wr(o+4, 4, 0)
		for i, v := range pos {
			m.wr(o+0x15+4*int64(i), 4, v)
		}
		m.wr(o+0x39, 4, -1)
		if len(placeArgs) >= mdgMaxObj {
			panic("monster_drop_generation: more than 41 objects")
		}
		placeArgs = append(placeArgs, o) // 0x4f93d0(o)
		m.wr(o+0x25, 4, kind)
		// 0x51fe30(o+9, &inst, 0, 1)
		if b[0]&b[1]&b[2]&b[3] == 0xff {
			panic("monster_drop_generation: srAssertFail 0x51fe30, source instance empty")
		}
		b[0xa] = 0
		for i, x := range b {
			m.wr(o+9+int64(i), 1, int64(x))
		}
		refresh() // source reset
		refresh() // destination
		if addWorld {
			worldAdd(o)
		}
		return o
	}

	// the local list (0x4f9580(5): 0x5e1ce0 of 0x14 bytes)
	ldata, lcount, lcap := alloc(0x14), int64(0), int64(5)
	appendObj := func(o int64) {
		if lcount+1 > lcap { // 0x4addf0(count+1)
			nb := alloc(((lcount + 1) * 4) & 0xffffffff)
			for i := int64(0); i < lcount; i++ {
				m.wr(nb+4*i, 4, m.rd(ldata+4*i))
			}
			ldata, lcap = nb, lcount+1
		}
		m.wr(ldata+4*lcount, 4, o)
		lcount++
	}
	drop := func(id int64) { appendObj(place(mk(id), 3, false)) }

	var poolArgs [][2]int64
	pool := func(id, q int64) { // 0x4f88f0(&list, id, q)
		if real {
			panic("monster_drop_generation: F does not model 0x4f88f0's appended objects")
		}
		if len(poolArgs) >= mdgMaxPool {
			panic("monster_drop_generation: more than 40 item-pool calls")
		}
		poolArgs = append(poolArgs, [2]int64{id, q})
		c, p := i32("pool.count"), i32("pool.cap")
		if c < 0 || c > p || p > mdgListCap {
			panic("monster_drop_generation: fixture item-pool list holds 0 <= count <= cap <= 8")
		}
		ldata, lcount, lcap = mdgLarr, c, p
	}

	var money int64
	if mon := u32("monArg"); mon != 0 {
		if mon != mdgMon {
			panic("monster_drop_generation: fixture holds one monster")
		}
		tid := u32("tid") // 0x4e5720
		if tid >= 1000 {
			panic("monster_drop_generation: srAssertFail template id below 1000")
		}
		if tid >= mdgTemplates {
			panic("monster_drop_generation: fixture holds 3 templates")
		}
		tmpl := mdgTmpl + 0x300*tid
		if u8("t.flags")&1 != 0 {
			if u8("t.cd") == mdgNoNPC { // 0x50a4a0 returns 0, assert at 0x4f8d33
				panic("monster_drop_generation: srAssertFail NPC record missing")
			}
			sel := i32("npcIdx")
			if sel >= i32("nlist.count") {
				sel = 0
			}
			if sel < 0 || sel >= mdgNPCs {
				panic("monster_drop_generation: fixture holds 2 NPC records")
			}
			item := &npcItem[sel]
			var w [mdgSlots]int64
			for j := range w {
				w[j] = i8(fmt.Sprintf("npc%d.w%d", sel, j))
			}
			for j := 0; j < mdgSlots; j++ { // 0x4f8d5c
				if item[j] != -1 && w[j] == 0 {
					drop(item[j] & 0xffffffff)
					item[j] = -1
				}
			}
			var left, sum int64
			for j := 0; j < mdgSlots; j++ {
				if item[j] != -1 {
					left++
					sum += w[j]
				}
			}
			if left > 0 {
				p := fmt.Sprintf("npc%d.", sel)
				k := dice(mdgNPC+0x200*sel+0x10e, p+"Z", p+"N", p+"S") & 0xffffffff
				k = min(k, left) // unsigned compare, exe 0x4f8e6b
				for done := int64(0); done < k; {
					j := r.R(mdgSlots)
					if item[j] == -1 {
						continue
					}
					rv := r.R(int(sum & 0xffffffff))
					if w[j] < int64(int16(rv)) { // 16-bit signed compare, exe 0x4f8eae
						continue
					}
					drop(item[j] & 0xffffffff)
					done++
					item[j] = -1
				}
			}
			money = i32(fmt.Sprintf("npc%d.money", sel))
		} else {
			for j := 0; j < mdgDropSlots; j++ {
				s := fmt.Sprintf("s%d.", j)
				kind, ne, id, c := u8(s+"kind"), u16(s+"ne"), u16(s+"id"), u8(s+"c")
				u8(s + "N")
				u8(s + "S")
				i16(s + "Z")
				if ne == 0 || int64(r.R(100)) > c {
					continue
				}
				q := dice(tmpl+0x1f3+10*int64(j)+6, s+"Z", s+"N", s+"S")
				if q <= 0 {
					continue
				}
				switch kind {
				case 0:
					for i := int64(0); i < q; i++ {
						drop(id)
					}
				case 1:
					for i := int64(0); i < q; i++ {
						pool(id, q)
					}
				}
			}
			money = dice(tmpl+0x243, "m.Z", "m.N", "m.S")
		}
		if money != 0 { // 0x51bea0(money, 1)
			gold = (gold + money) & 0xffffffff
			if party < 0x2f {
				if party < -1 {
					panic("monster_drop_generation: fixture party index below -1")
				}
				acc[party+1] = (acc[party+1] + money) & 0xffffffff
			}
		}
		if lcount > 1 {
			kind := int64(3)
			id := p2
			if p2 == 0xffffffff {
				kind, id = 2, mdgBoxItem
			}
			box := place(mk(id), kind, true)
			for k := int64(0); k < lcount; k++ { // chain on +0x31, exe 0x4f9133
				o := m.rd(ldata + 4*k)
				e := box
				for steps := 0; ; steps++ {
					nx := m.rd(e + 0x31)
					if nx == 0 {
						break
					}
					if steps > mdgMaxObj+mdgListCap {
						panic("monster_drop_generation: chain leaves the fixture")
					}
					e = nx
				}
				m.wr(e+0x31, 4, o)
				m.wr(o+0x31, 4, 0)
			}
		} else if lcount == 1 {
			o := m.rd(ldata)
			worldAdd(o)
			for i, v := range pos {
				m.wr(o+0x15+4*int64(i), 4, v)
			}
		}
	}

	out := []int64{ordinal, gold}
	out = append(out, acc[:]...)
	out = append(out, m.rd(mdgGlist), int64(int32(m.rd(mdgGlist+4))), int64(int32(m.rd(mdgGlist+8))))
	for k := int64(0); k < mdgListCap; k++ {
		out = append(out, m.rd(mdgGdata+4*k))
	}
	for k := int64(0); k < mdgListCap; k++ {
		out = append(out, m.rd(mdgLarr+4*k))
	}
	for k := int64(0); k < mdgListCap; k++ {
		for _, off := range []int64{0x15, 0x19, 0x1d, 0x31} {
			out = append(out, m.rd(mdgL0+0x100*k+off))
		}
	}
	for n := 0; n < mdgNPCs; n++ {
		out = append(out, npcItem[n][:]...)
	}
	seq := func(n int, v func(k int) int64, cnt int) {
		for k := 0; k < n; k++ {
			if k < cnt {
				out = append(out, v(k))
			} else {
				out = append(out, 0)
			}
		}
	}
	out = append(out, int64(len(diceArgs)))
	seq(mdgDice, func(k int) int64 { return diceArgs[k] }, len(diceArgs))
	out = append(out, int64(len(mkArgs)))
	for a := 0; a < 4; a++ {
		seq(mdgMaxObj, func(k int) int64 { return mkArgs[k][a] }, len(mkArgs))
	}
	out = append(out, int64(len(poolArgs)))
	for a := 0; a < 2; a++ {
		seq(mdgMaxPool, func(k int) int64 { return poolArgs[k][a] & 0xffffffff }, len(poolArgs))
	}
	out = append(out, int64(len(placeArgs)))
	seq(mdgMaxObj, func(k int) int64 { return placeArgs[k] }, len(placeArgs))
	out = append(out, int64(len(refreshArgs)))
	seq(3*mdgMaxObj, func(int) int64 { return 0 }, len(refreshArgs))
	seq(3*mdgMaxObj, func(int) int64 { return 1 }, len(refreshArgs))
	for w := int64(0); w < mdgHeapWords; w++ {
		out = append(out, m.rd(mdgHeap+4*w))
	}
	return out
}

// mdgRealInstance runs 0x51c020 on instance id with the f.* item definitions
// (no forced flags, no maximum roll); a charge dice is rolled with diceNode.
func mdgRealInstance(in []int64, at map[string]int, id int64, r Rand) [3]int64 {
	iv := make([]int64, len(iscIn))
	set := func(n string, x int64) {
		for i, name := range iscIn {
			if name == n {
				iv[i] = x
				return
			}
		}
		panic("monster_drop_generation: no 0x51c020 input " + n)
	}
	set("p", iscInst)
	set("id", id)
	set("itemsBase", in[at["f.itemsBase"]])
	set("itemCount", 0x10000) // ids past the 4 fixture records panic in itemStackChargeInitNode
	for j := 0; j < iscItems; j++ {
		for _, f := range []string{"kind", "z", "n", "s", "attr"} {
			set(fmt.Sprintf("i%d.%s", j, f), in[at[fmt.Sprintf("f.i%d.%s", j, f)]])
		}
	}
	if id < iscItems && in[at[fmt.Sprintf("f.i%d.kind", id)]]&0xff != 0 {
		p := fmt.Sprintf("f.i%d.", id)
		set("diceRet", diceNode([]int64{in[at[p+"z"]], in[at[p+"n"]], in[at[p+"s"]]}, r)[0])
	}
	o := itemStackChargeInitNode(iv, r)
	return [3]int64{o[0], o[1], o[2]}
}
