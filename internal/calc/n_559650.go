package calc

import "fmt"

// npc_starting_equipment 0x559650(C, src): give party member C its starting gear.
//
// Order: 0x520310 clears the twelve equipment and eight backpack records
// (id -1, 0x520d10 per record); the two ten-slot source lists (src+0x40, stride
// 0x78, entry 12 bytes, u16 id first) are resolved against the item table (first
// row whose +0x3c word equals the id): 0x128 and cheap items are equipped at
// once, rows priced 3000..4999 (list A) and 1000..2999 (list B) are collected and
// one of A and two of B, richest first, are equipped afterwards (the pick uses the
// halved price unless 0x520750 keeps the item, but remembers the unhalved one);
// the class table row (race 5 uses row 15) is walked for six default items
// (0x51c4e0); a character without a weapon slot (6) gets a class default; the
// final 0x50e540 rebuilds the character (0x4ed9d0 via 0x50e5c0).
//
// Fixture (calc panics past each): item ids -1..1023 (the table object holds 1024
// rows; rows outside the twelve exposed ones read as all-zero rows), item count
// +0x683f78 <= 1024, class table index 0..15, six exposed pack records, pack count
// 0x1f4 or more only when nothing is scanned or inserted, slot results 0..11.
//
// Address stubs of the case: the D4 callees 0x51d610, 0x51c5a0, 0x51f900,
// 0x51b5c0, 0x518150 (returns from the sequence inputs q*), 0x520d10, 0x4edd20,
// 0x52e690, 0x51d960, 0x51c020, 0x5201b0; their argument records and call counts
// are outputs. 0x50e5c0 and 0x4ed9d0 run natively; calc reuses
// equipmentEffectRebuildNode through an identity slot remap. 0x5213c0 and the
// rest of 0x51c300 / 0x51c280 / 0x521ef0 / 0x51fe30 / 0x5227d0 / 0x4edc60 are
// modelled inline.
const (
	n559Stride  = 0x1862     // 0x520310: character stride
	n559CBase   = 0x20010000 // case object "chars"
	n559MaxDefs = 1024       // item count the case table object can hold (zero rows past the exposed ones)
	n559MaxID   = 1023       // last item id of the 1024-row table object (exe class table reaches 0x301)
	n559Packed  = 6          // exposed pack records
	n559PackMax = 0x1f4      // 0x521f91
	n559None    = 0xffffffff // empty record id
)

// Row field indices, in the interface order of rows.
const (
	n559Id3c = iota
	n559Type
	n559F3f
	n559F41
	n559F66
	n559Price
	n559W8a
	n559F8c
	n559W4e
	n559B62
	n559B48
	n559B49
	n559B6c
	n559B6d
	n559B6e
	n559B6f
	n559B70
	n559B71
	n559B72
	n559B73
	n559B74
	n559Bb1
	n559Bb2
	n559Bb3
	n559Bb4
	n559RowN
)

var n559RowNames = [n559RowN]string{"id3c", "type", "f3f", "f41", "f66", "price", "w8a", "f8c", "w4e", "b62",
	"b48", "b49", "b6c", "b6d", "b6e", "b6f", "b70", "b71", "b72", "b73", "b74", "bb1", "bb2", "bb3", "bb4"}

var n559RowIdx = [12]uint32{0, 1, 2, 3, 4, 5, 6, 7, 0x12, 0x16, 0x52, 0x128}

// stub sequence: explicit inputs, and the number of entries the case reserves.
var n559Seqs = []struct {
	name        string
	explicit, n int
}{{"51d610", 64, 64}, {"51c5a0", 8, 8}, {"51f900", 64, 16384}, {"51b5c0", 64, 64}, {"518150", 48, 48}}

func init() {
	var in []string
	add := func(f string, a ...any) { in = append(in, fmt.Sprintf(f, a...)) }
	in = append(in, "who", "C", "mode", "started", "defCount", "c4", "c69", "c71", "c75", "c89", "wOld", "wCap", "v558")
	for k := 0; k < 15; k++ {
		add("cl%d", k)
	}
	for k := 0; k < 7; k++ {
		add("ra%d", k)
	}
	for k := 0; k < 41; k++ {
		add("rs%d", k)
	}
	in = append(in, "tb")
	for k := 0; k < 103; k++ {
		add("st%d", k)
	}
	for k := 0; k < 103; k++ {
		add("pm%d", k)
	}
	for r := 0; r < 16; r++ {
		for k := 0; k < 6; k++ {
			add("ct%d_%d", r, k)
		}
	}
	for z := 0; z < 4; z++ {
		for _, o := range []int{0, 4, 8} {
			add("z%d_%d", z, o)
		}
	}
	in = append(in, "s1a3", "s1c9", "s215", "s261", "i4", "i8", "pkN")
	for j := 0; j < 6; j++ {
		add("pk%d", j)
		add("pk%d_4", j)
		add("pk%d_8", j)
	}
	for g := 0; g < 2; g++ {
		for k := 0; k < 10; k++ {
			add("s%d_%d", g, k)
		}
	}
	for r := range n559RowIdx {
		for _, f := range n559RowNames {
			add("r%d_%s", r, f)
		}
	}
	for _, q := range n559Seqs {
		for i := 0; i < q.explicit; i++ {
			add("q%s_%d", q.name, i)
		}
	}
	var out []string
	addo := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	for s := 0; s < 12; s++ {
		for _, o := range []int{0, 4, 8} {
			addo("e%d_%d", s, o)
		}
	}
	for s := 0; s < 12; s++ {
		for _, o := range []int{0, 4, 8} {
			addo("b%d_%d", s, o)
		}
	}
	for j := 0; j < 6; j++ {
		for _, o := range []int{0, 4, 8} {
			addo("p%d_%d", j, o)
		}
	}
	out = append(out, "packN", "wNew", "capNew", "bits", "k0New", "k1New", "k3New", "k5New")
	for _, l := range n559LogSpecs {
		for _, a := range l.args {
			for k := 0; k < l.n; k++ {
				addo("x%s_a%d_%d", l.name, a, k)
			}
		}
		addo("x%s_cnt", l.name)
	}
	Nodes["00559650"] = Node{Addr: "00559650", In: in, Out: out,
		F:   func(in []int64, r Rand) []int64 { return npcStartingEquipment(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return npcStartingEquipment(in, r, false) }}
}

// Output order of the stubbed callees: argument records (1-based argument
// numbers, n slots each) then the call count.
var n559LogSpecs = []struct {
	name string
	args []int
	n    int
}{{"51d610", []int{2}, 64}, {"51c5a0", []int{2}, 8}, {"51f900", nil, 0}, {"51b5c0", []int{2}, 64},
	{"520d10", []int{2, 3}, 96}, {"4edd20", nil, 0}, {"518150", []int{4}, 48}, {"52e690", []int{1, 2, 3, 4, 5}, 64},
	{"51d960", nil, 0}, {"51c020", []int{2}, 48}}

type n559Rec struct{ id, d4, d8 uint32 }

// n559Seq replays a stub's sequenced return: call k reads entry k.
type n559Seq struct {
	v    []int64
	n, k int
}

func (q *n559Seq) next() int64 {
	if q.k >= q.n {
		panic("npc_starting_equipment: stub sequence past the case")
	}
	var x int64
	if q.k < len(q.v) {
		x = q.v[q.k]
	}
	q.k++
	return x
}

// n559Log keeps the arguments of every call of a stub (capped at n) and the count.
type n559Log struct {
	n    int
	cols [][]uint32
	cnt  int
}

func newN559Log(args, n int) *n559Log {
	l := &n559Log{n: n, cols: make([][]uint32, args)}
	for i := range l.cols {
		l.cols[i] = make([]uint32, n)
	}
	return l
}

func (l *n559Log) add(v ...int64) {
	if len(l.cols) > 0 && l.cnt >= l.n {
		panic("npc_starting_equipment: stub call record past the case")
	}
	for i := range l.cols {
		l.cols[i][l.cnt] = uint32(v[i])
	}
	l.cnt++
}

type n559 struct {
	real                               bool
	who                                int64
	c                                  uint32
	mode, started, defCount            int64
	c4, class, race, gender            int64
	level, w, cap, v558                uint32
	classLevel                         [15]int64
	rawAttr                            [7]int64
	rawSkill                           [41]int64
	tb                                 int64
	st, pm                             [103]int64
	ct                                 [16][6]int64
	eq, bp                             [12]n559Rec
	pack                               [n559Packed + 1]n559Rec
	packN                              int64
	skill                              [4]uint32 // +0x1a3, +0x1c9, +0x215, +0x261
	i4, i8                             uint32
	src                                [2][10]uint16
	rows                               [12][n559RowN]int64
	zero                               [n559RowN]int64
	bits                               uint32
	seq                                [5]*n559Seq
	lg                                 map[string]*n559Log
	listA, listB                       [20]uint16
	nA, nB                             int
	l51d610, l51c5a0, l51b5c0, l518150 *n559Log
}

func npcStartingEquipmentNode(in []int64, r Rand) []int64 { return npcStartingEquipment(in, r, true) }

func npcStartingEquipment(in []int64, _ Rand, real bool) []int64 {
	s := &n559{real: real, bits: 0x401, lg: map[string]*n559Log{}}
	p := 0
	next := func() int64 { x := in[p]; p++; return x }
	s.who = next()
	c := uint32(next())
	if s.who < 0 || s.who > 7 || int64(c) != n559CBase+s.who*n559Stride {
		panic("npc_starting_equipment: C is not the fixture character of `who`")
	}
	s.c = c
	s.mode, s.started, s.defCount = next(), next(), next()
	s.c4, s.class, s.race, s.gender = next(), next(), next(), next()
	s.level, s.w, s.cap, s.v558 = uint32(next()), uint32(next()), uint32(next()), uint32(next())
	for k := range s.classLevel {
		s.classLevel[k] = next()
	}
	for k := range s.rawAttr {
		s.rawAttr[k] = next()
	}
	for k := range s.rawSkill {
		s.rawSkill[k] = next()
	}
	s.tb = next()
	for k := range s.st {
		s.st[k] = next()
	}
	for k := range s.pm {
		s.pm[k] = next()
	}
	for r := range s.ct {
		for k := range s.ct[r] {
			s.ct[r][k] = next()
		}
	}
	var z [4]n559Rec
	for k := range z {
		z[k] = n559Rec{uint32(next()), uint32(next()), uint32(next())}
	}
	for k := range s.skill {
		s.skill[k] = uint32(next())
	}
	s.i4, s.i8 = uint32(next()), uint32(next())
	s.packN = next()
	for k := 0; k < 6; k++ {
		s.pack[k] = n559Rec{uint32(next()), uint32(next()), uint32(next())}
	}
	for g := range s.src {
		for k := range s.src[g] {
			s.src[g][k] = uint16(next())
		}
	}
	for r := range s.rows {
		for f := range s.rows[r] {
			s.rows[r][f] = next()
		}
	}
	for i, q := range n559Seqs {
		sq := &n559Seq{n: q.n, v: in[p : p+q.explicit]}
		p += q.explicit
		s.seq[i] = sq
	}
	if p != len(in) {
		panic("npc_starting_equipment: wrong input count")
	}
	for i := 8; i < 12; i++ {
		s.bp[i] = z[i-8]
	}
	for _, l := range n559LogSpecs {
		s.lg[l.name] = newN559Log(len(l.args), l.n)
	}
	s.l51d610, s.l51c5a0, s.l51b5c0, s.l518150 = s.lg["51d610"], s.lg["51c5a0"], s.lg["51b5c0"], s.lg["518150"]
	s.run()
	return s.out()
}

func (s *n559) out() []int64 {
	var o []int64
	rec := func(r n559Rec) { o = append(o, int64(r.id), int64(r.d4), int64(r.d8)) }
	for _, r := range s.eq {
		rec(r)
	}
	for _, r := range s.bp {
		rec(r)
	}
	for j := 0; j < 6; j++ {
		rec(s.pack[j])
	}
	o = append(o, s.packN, int64(s.w), int64(s.cap), int64(s.bits))
	for _, k := range s.skill {
		o = append(o, int64(k))
	}
	for _, l := range n559LogSpecs {
		lg := s.lg[l.name]
		for a := range l.args {
			for k := 0; k < l.n; k++ {
				o = append(o, int64(lg.cols[a][k]))
			}
		}
		o = append(o, int64(lg.cnt))
	}
	return o
}

// row is the item table row of id; only the twelve exposed rows carry data.
func (s *n559) row(id uint32) *[n559RowN]int64 {
	if id > n559MaxID {
		panic("npc_starting_equipment: item id outside the 1024-row fixture table")
	}
	for i, x := range n559RowIdx {
		if x == id {
			return &s.rows[i]
		}
	}
	return &s.zero
}

// scanRow is row for the id search, which also reads rows past the cap (all zero).
func (s *n559) scanRow(i int64) *[n559RowN]int64 {
	if i > n559MaxID {
		return &s.zero
	}
	return s.row(uint32(i))
}

// find is the table search of 0x559650: first row whose +0x3c word is id.
func (s *n559) find(id uint16) (uint32, bool) {
	if s.defCount < 0 || s.defCount > n559MaxDefs {
		panic("npc_starting_equipment: item count past the fixture table")
	}
	for i := int64(0); i < s.defCount; i++ {
		if uint16(s.scanRow(i)[n559Id3c]) == id {
			return uint32(i), true
		}
	}
	return 0, false
}

func (s *n559) tmp(idx uint32) n559Rec { // 0x51c020
	s.lg["51c020"].add(int64(idx))
	return n559Rec{idx, s.i4, s.i8}
}

func (s *n559) cnt(name string) { s.lg[name].cnt++ }

func (s *n559) clear(r *n559Rec) { // 0x520310 per record
	*r = n559Rec{id: n559None}
	s.lg["520d10"].add(int64(s.c), 1)
}

func (s *n559) run() {
	for i := range s.eq {
		s.clear(&s.eq[i])
	}
	for i := 0; i < 8; i++ {
		s.clear(&s.bp[i])
	}
	for g := 0; g < 2; g++ {
		for _, id := range s.src[g] {
			if id == 0 {
				continue
			}
			idx, ok := s.find(id)
			if !ok {
				continue
			}
			row := s.row(idx)
			if idx != 0x128 {
				if row[n559F3f] == 0x84 || row[n559Price] >= 5000 {
					continue
				}
				if row[n559Price] >= 3000 {
					s.listA[s.nA] = id
					s.nA++
					continue
				}
				if row[n559Price] >= 1000 {
					s.listB[s.nB] = id
					s.nB++
					continue
				}
			}
			t := s.tmp(idx)
			if row[n559F8c] == 0 {
				s.c280(&t, g == 0)
			} else {
				s.c300(&t, false)
			}
		}
	}
	s.pickBest(s.listA[:], &s.nA, 1)
	s.pickBest(s.listB[:], &s.nB, 2)
	// Class defaults (0x61635c, six dwords per class; race 5 uses row 15).
	rowc := int64(15)
	if s.race != 5 {
		rowc = s.class
	}
	if rowc < 0 || rowc > 15 {
		panic("npc_starting_equipment: class table row outside 0..15")
	}
	for _, d := range s.ct[rowc] {
		if d == -1 {
			continue
		}
		if d < 0 || d > n559MaxID {
			panic("npc_starting_equipment: class table item id outside the fixture table")
		}
		t := s.tmp(uint32(d))
		rw := s.row(uint32(d))
		slot := itemDefaultEquipSlotNode([]int64{d, rw[n559Type], s.started}, nil)[0]
		if slot == -1 {
			if !s.has(rw[n559F3f]) {
				s.c300(&t, true)
			}
			continue
		}
		if slot == 6 && rw[n559F41]&8 != 0 {
			slot = 7
		}
		if s.eq[slot].id == n559None {
			s.c300(&t, true)
		}
	}
	if s.eq[6].id == n559None {
		var id uint32
		switch s.class {
		case 0:
			id = 7
			if s.skill[0] < s.skill[1] {
				id = 0x12
			}
		case 10:
			id = 0x16
			if s.skill[2] <= s.skill[3] {
				id = 0x52
			}
		}
		if s.class == 0 || s.class == 10 {
			t := s.tmp(id)
			s.c300(&t, true)
		}
	}
	s.rebuild()
}

// pickBest equips up to capN (min of count and limit) entries of the list, best
// first: the value compared is the price, halved unless 0x520750 keeps the item,
// but the one remembered for the next comparison is the unhalved price.
func (s *n559) pickBest(list []uint16, n *int, limit int) {
	capN := min(*n, limit)
	for capN > 0 {
		best, bestVal := -1, int64(0)
		for i := 0; i < *n; i++ {
			if int16(list[i]) <= 0 {
				continue
			}
			idx, _ := s.find(list[i])
			rw := s.row(idx)
			val := rw[n559Price]
			if depletedItemRetentionNode([]int64{int64(idx), rw[n559W4e], rw[n559B62], rw[n559B6c], rw[n559B6d], rw[n559B6e],
				rw[n559B6f], rw[n559B70], rw[n559B71], rw[n559B72], rw[n559B73], rw[n559B74], rw[n559F8c]}, nil)[0] == 0 {
				val /= 2
			}
			if bestVal < val {
				bestVal, best = rw[n559Price], i
			}
		}
		if best == -1 {
			panic("npc_starting_equipment: no pick, the original loops forever")
		}
		idx, _ := s.find(list[best])
		t := s.tmp(idx)
		if s.row(idx)[n559F8c] == 0 {
			s.c280(&t, true)
		} else {
			s.c300(&t, false)
		}
		capN--
		list[best] = 0
	}
}

// has is 0x5213c0(C, f3f, 0, 2): an equipped or backpack item with this +0x3f word.
func (s *n559) has(f3f int64) bool {
	for _, set := range [][]n559Rec{s.eq[:], s.bp[:8]} {
		for _, r := range set {
			if r.id != n559None && uint16(s.row(r.id)[n559F3f]) == uint16(f3f) {
				return true
			}
		}
	}
	return false
}

// c280 is 0x51c280(C, item, 0, 0, a5).
func (s *n559) c280(item *n559Rec, a5 bool) bool {
	return s.c300(item, a5) || s.insert(item)
}

func (s *n559) put(dst, src *n559Rec, c uint32) { // 0x51fe30(dst, src, c, 1)
	src.d8 &^= 0xff0000
	*dst = *src
	*src = n559Rec{id: n559None}
	s.lg["520d10"].add(int64(c), 1)
	s.lg["520d10"].add(int64(c), 1)
}

func (s *n559) useOK(id uint32) bool { // 0x51d610(C, id)
	s.l51d610.add(int64(id))
	return s.seq[0].next()&0xff != 0
}

func (s *n559) merge() bool { // 0x51f900(record, item, 0)
	s.cnt("51f900")
	return s.seq[2].next()&0xff != 0
}

// c300 is 0x51c300(C, item, arg3, 0, 0).
func (s *n559) c300(item *n559Rec, arg3 bool) bool {
	if arg3 && s.useOK(item.id) {
		s.l51c5a0.add(int64(item.id))
		slot := int64(int32(s.seq[1].next()))
		if slot < -1 || slot > 11 {
			panic("npc_starting_equipment: slot result outside -1..11")
		}
		if slot != -1 {
			ok := false
			if s.eq[slot].id == n559None {
				s.put(&s.eq[slot], item, s.c)
				ok = true
			} else {
				ok = s.merge()
			}
			if ok {
				if s.c4 != 0 {
					s.rebuild()
				}
				return true
			}
		}
	}
	s.l51b5c0.add(1)
	s.seq[3].next() // result lands in a stack slot the test below never reads (it tests arg5, always 0)
	esi := -1
	if s.row(item.id)[n559F66] == 1 {
		for i := 0; i < 12 && esi < 0; i++ {
			if s.merge() {
				esi = i
			}
		}
		for i := 0; i < 8 && esi < 0; i++ {
			if s.merge() {
				esi = i
			}
		}
	}
	if esi < 0 {
		for i := 0; i < 8 && esi < 0; i++ {
			if s.bp[i].id == n559None {
				esi = i
			}
		}
		if esi < 0 {
			return false
		}
		s.put(&s.bp[esi], item, 0)
	}
	if s.weigh() {
		s.cnt("4edd20")
	}
	if s.mode == 6 {
		s.bits |= 2 << (uint(esi) & 31)
	}
	s.potion(&s.bp[esi], false)
	return true
}

// insert is 0x521ef0(item, 0, 0): put the item at the head of the pack list.
func (s *n559) insert(item *n559Rec) bool {
	s.l51b5c0.add(1)
	s.seq[3].next() // result unused here
	hit := -1
	if s.row(item.id)[n559F66] == 1 {
		for j := int64(0); j < s.packN && hit < 0; j++ {
			if j >= n559Packed {
				panic("npc_starting_equipment: pack record past the six exposed")
			}
			if s.merge() {
				hit = int(j)
			}
		}
	}
	bl := false
	if hit >= 0 {
		bl = true
		s.cnt("4edd20")
	} else {
		if s.packN >= n559PackMax {
			return false
		}
		if s.packN > n559Packed {
			panic("npc_starting_equipment: pack shift past the six exposed records")
		}
		for j := s.packN; j > 0; j-- {
			s.pack[j] = s.pack[j-1]
		}
		s.pack[0] = n559Rec{id: n559None}
		s.put(&s.pack[0], item, 0)
		s.packN++
		s.cnt("4edd20")
		bl = true
		hit = 0
	}
	if s.mode == 6 {
		s.bits |= 0x7fc00000
	}
	s.potion(&s.pack[hit], true)
	return bl
}

// weigh is 0x4edc60: recompute the carried weight, warn when it newly exceeds
// the capacity, report whether it changed.
func (s *n559) weigh() bool {
	old := s.w
	var sum uint32
	for _, set := range [][]n559Rec{s.eq[:], s.bp[:8]} {
		for _, r := range set {
			if r.id != n559None {
				sum += max(1, r.d4&0xff) * uint32(s.row(r.id)[n559W8a])
			}
		}
	}
	s.w = sum
	if old < s.cap && sum > s.cap {
		msg := int64(s.v558)
		if s.mode == 6 {
			msg = potionMsgWeight()
		}
		s.warn(msg)
	}
	return old != sum
}

func potionMsgWeight() int64 {
	if potionExeErr != nil {
		panic(potionExeErr)
	}
	return potionExe.msgWeight
}

func (s *n559) warn(msg int64) { // 0x52e690(C, msg, 0, [0x5ed8c8], [0x5ed914])
	if potionExeErr != nil {
		panic(potionExeErr)
	}
	s.lg["52e690"].add(int64(s.c), msg, 0, potionExe.a4, potionExe.a5)
}

// potion is 0x5227d0(record, a2, C) (C is the character only when a2 is false) for item ids below the special ones.
func (s *n559) potion(r *n559Rec, a2 bool) {
	if s.mode == 5 {
		return
	}
	if a2 {
		s.l518150.add(-1)
		if int32(s.seq[4].next()) == -1 {
			return
		}
	}
	if r.d8>>8&1 != 0 {
		return
	}
	switch r.id {
	case 0x239, 0x242, 0x243, 0x244, 0x264, 0x27c:
		panic("npc_starting_equipment: special item id in 0x5227d0 is outside the 1024-row fixture")
	}
	if a2 {
		return
	}
	if !s.useOK(r.id) {
		return
	}
	// An empty record (id -1) only gets here through a stubbed merge hit; the
	// fixture arena reads zero outside the item table, so its price is 0.
	var price uint32
	if r.id != n559None {
		price = uint32(s.row(r.id)[n559Price])
	}
	q := price / s.level
	if q > 500 && r.d4>>16&0xff != 0 {
		if potionExeErr != nil {
			panic(potionExeErr)
		}
		s.lg["52e690"].add(int64(s.c), potionExe.msgOver, 0, potionExe.a4, potionExe.a5)
		r.d8 |= 0x100
	}
}

// rebuild is 0x50e540 -> 0x4ed9d0. Its other effects (ability caches,
// notifications) are not exposed; capacity and the four effective skills are.
func (s *n559) rebuild() {
	if s.class >= 15 || s.race >= 16 || s.gender == -1 {
		return
	}
	if s.class < 0 {
		panic("npc_starting_equipment: negative class")
	}
	s.level = min(s.level, 50)
	for k := range s.classLevel {
		s.classLevel[k] = min(s.classLevel[k], 50)
	}
	s.lg["51d960"].cnt += 48
	in := make([]int64, 508)
	in[0], in[1], in[2], in[3], in[4], in[5] = s.who, s.c4, s.class, s.race, s.gender, int64(s.level)
	copy(in[6:], s.classLevel[:])
	copy(in[21:], s.rawAttr[:])
	copy(in[28:], s.rawSkill[:])
	in[69] = s.tb
	copy(in[70:], s.st[:])
	copy(in[173:], s.pm[:])
	for i, r := range s.eq {
		in[276+i] = -1
		if r.id == n559None {
			continue
		}
		rw := s.row(r.id)
		in[276+i] = int64(i)
		for k, f := range []int{n559B48, n559B49, n559B6c, n559B6d, n559B6e, n559Bb1, n559Bb2, n559Bb3, n559Bb4, n559B6f, n559B70, n559B71, n559B72, n559B73, n559B74} {
			in[288+i*15+k] = rw[f]
		}
		in[468+i] = rw[n559W8a]
		in[480+i] = int64(max(1, r.d4&0xff))
	}
	for k, r := range s.bp[:8] {
		if r.id != n559None {
			in[492+2*k] = s.row(r.id)[n559W8a]
			in[493+2*k] = int64(max(1, r.d4&0xff))
		}
	}
	rb := equipmentEffectRebuildNode(in, nil)
	oldCap := s.cap
	s.cap = uint32(rb[357])
	for k, i := range []int{0, 1, 3, 5} {
		s.skill[k] = uint32(rb[316+i])
	}
	al := s.weigh()
	if s.started != 0 && (oldCap != s.cap || al) {
		s.cnt("4edd20")
	}
}
