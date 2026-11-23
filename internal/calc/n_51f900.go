package calc

import "fmt"

// item_stack_merge 0x51f900(a, b *item, flag *u8): move as much of b's count (+0x4) as fits
// into a, up to the definition's stack limit (+0x6b) (build/wiz8.asm:389744..389810).
// Steps: a.id == -1 returns 0; 0x51fb40 normalizes a first (itemStackNormalizeNode runs
// natively in the emulator, so it is called here); a definition kind (+0x66) other than 1
// returns 0; a.qty above the limit is the assert at 0x51f956, which cannot happen after the
// normalize (it stops the emulator, so it is cov-exempt and panics here). Merging needs
// equal ids, equal byte +6 and spare room. A source left non-empty sets *flag and returns 0.
// An emptied source: the hand struct (0x6874cb) is cleared via 0x55f1e0 and the held-source
// markers 0x6840c0/4/5 are set to -1; any other source is zeroed with id -1 and the
// equipment refresh 0x520d10 is called with (b, 0, 1); a source that is a shared-pack slot
// (0x685191.. 12 bytes each, count at 0x686901) also closes the gap and calls the shared
// weight refresh 0x4edd20. Both refreshes are address-stubbed in the case; Out carries their
// call counts and the last 0x520d10 arguments. 0x55f1e0 releases the cursor graphic by calling
// 0x427ab0/0x427ff0 (stubbed, args and counts recorded) when 0x683fdb is not -1.
// Memory model: instances live in arena objects A, B, the hand struct or the first ismPool
// pack slots; any other pointer, a pack slot past the fixture cap, or an item id past the
// fixture definition table (ids 0..15, same as 0x51fb40) panics. Instance a in an unused pack
// slot (index >= count) holding a live item panics: 0x51fb40 may overwrite it.
const (
	ismPool    = 16 // fixture pack slots modelled (game: 500)
	ismPA      = 0x20000000
	ismPB      = 0x20000020
	ismPF      = 0x20000040
	ismHand    = 0x6874cb
	ismPack    = 0x685191
	ismPackEnd = 0x6868f5
)

type ismInst struct {
	id   int32
	tail uint64 // bytes +4..+11: qty, side count, +6, +7, u32 @+8
}

func ismFields(prefix string) []string {
	var s []string
	for _, f := range []string{"id", "qty", "b5", "b6", "b7", "w8"} {
		s = append(s, prefix+"."+f)
	}
	return s
}

func init() {
	in := []string{"p1", "p2", "pFlag", "id1"}
	in = append(in, ismFields("A")...)
	in = append(in, ismFields("B")...)
	in = append(in, ismFields("hand")...)
	in = append(in, "handMark")
	for k := 0; k < ismPool; k++ {
		in = append(in, ismFields(fmt.Sprintf("pack%d", k))...)
	}
	in = append(in, "packCount", "mark0", "mark4", "mark5", "cursorGfx", "cursorB", "cursorC", "cursorTex", "tblWord",
		"defType", "defMax", "ret520d10", "ret4edd20", "flagByte")
	out := []string{"ret", "flagByte", "A.id", "A.tail", "B.id", "B.tail", "hand.id", "hand.tail", "handMark"}
	for k := 0; k < ismPool; k++ {
		out = append(out, fmt.Sprintf("pack%d.id", k), fmt.Sprintf("pack%d.tail", k))
	}
	out = append(out, "packCount", "mark0", "mark4", "mark5", "cursorGfx", "cursorB", "cursorC",
		"n4edd20", "n520d10", "m520d10a1", "m520d10a2", "m520d10a3",
		"n427ab0", "m427ab0a1", "m427ab0a2", "m427ab0a3", "m427ab0a4", "n427ff0")
	Nodes["0051f900"] = Node{Addr: "0051f900", In: in, Out: out, F: itemStackMergeNode}
}

func ismTail(q, s, b6, b7 uint8, w8 uint32) uint64 {
	return uint64(q) | uint64(s)<<8 | uint64(b6)<<16 | uint64(b7)<<24 | uint64(w8)<<32
}

func itemStackMergeNode(in []int64, r Rand) []int64 {
	if len(in) != 4+6*3+1+6*ismPool+14 {
		panic("item stack merge: wrong input count")
	}
	p := 0
	next := func() int64 { p++; return in[p-1] }
	p1, p2, pf := uint32(next()), uint32(next()), uint32(next())
	id1 := int32(next())
	inst := func() ismInst {
		id := int32(next())
		q, s, b6, b7, w8 := uint8(next()), uint8(next()), uint8(next()), uint8(next()), uint32(next())
		return ismInst{id, ismTail(q, s, b6, b7, w8)}
	}
	a, b, hand := inst(), inst(), inst()
	handMark := uint8(next())
	var pack [ismPool]ismInst
	for k := range pack {
		pack[k] = inst()
	}
	count := uint32(next())
	mark0, mark4, mark5 := uint32(next()), uint8(next()), uint16(next())
	cursor, cursorB, cursorC, cursorTex, tblWord := uint32(next()), uint32(next()), uint32(next()), uint32(next()), uint16(next())
	kind, limit := uint8(next()), uint8(next())
	next()
	next()
	flagByte := uint8(next())

	if count > ismPool {
		panic("item stack merge: pack count above the fixture cap")
	}
	resolve := func(ptr uint32) *ismInst {
		switch {
		case ptr == ismPA:
			return &a
		case ptr == ismPB:
			return &b
		case ptr == ismHand:
			return &hand
		case ptr >= ismPack && ptr < ismPack+12*ismPool && (ptr-ismPack)%12 == 0:
			return &pack[(ptr-ismPack)/12]
		}
		panic("item stack merge: pointer outside the modelled memory")
	}
	var n4edd20, n520d10, n427ab0, n427ff0 uint32
	var m520d10 [3]uint32
	var m427ab0 [4]uint32
	ret := uint8(0)
	finish := func() []int64 {
		o := []int64{int64(ret), int64(flagByte), int64(a.id), int64(a.tail), int64(b.id), int64(b.tail),
			int64(hand.id), int64(hand.tail), int64(handMark)}
		for _, s := range pack {
			o = append(o, int64(s.id), int64(s.tail))
		}
		o = append(o, int64(count), int64(mark0), int64(mark4), int64(mark5), int64(cursor), int64(cursorB), int64(cursorC),
			int64(n4edd20), int64(n520d10), int64(m520d10[0]), int64(m520d10[1]), int64(m520d10[2]),
			int64(n427ab0), int64(m427ab0[0]), int64(m427ab0[1]), int64(m427ab0[2]), int64(m427ab0[3]), int64(n427ff0))
		return o
	}

	i1 := resolve(p1)
	if i1.id == -1 { // 0x51f91c
		return finish()
	}
	if i1.id != id1 || id1 < 0 || id1 > isnMaxDefs {
		panic("item stack merge: item id outside the fixture definition table")
	}
	if p1 >= ismPack && p1 < ismPack+12*ismPool && uint32((p1-ismPack)/12) >= count {
		panic("item stack merge: item a in an unused pack slot")
	}
	// 0x51f923: normalize a.
	ni := []int64{int64(i1.id), int64(i1.tail & 0xff), int64(i1.tail >> 8 & 0xff), int64(i1.tail >> 16 & 0xff),
		int64(i1.tail >> 24 & 0xff), int64(i1.tail >> 32), int64(kind), int64(limit), int64(count)}
	no := itemStackNormalizeNode(ni, r)
	i1.tail = uint64(no[1])
	newCount := uint32(no[2*isnSlots+2])
	if newCount > ismPool {
		panic("item stack merge: pack count above the fixture cap")
	}
	for k := count; k < newCount; k++ {
		pack[k] = ismInst{int32(no[2+2*k]), uint64(no[3+2*k])}
	}
	count = newCount
	n4edd20 += uint32(no[2*isnSlots+3])
	n520d10 += uint32(no[2*isnSlots+4])
	m520d10 = [3]uint32{uint32(no[2*isnSlots+5]), uint32(no[2*isnSlots+6]), uint32(no[2*isnSlots+7])}

	if kind != 1 { // 0x51f947
		return finish()
	}
	if uint8(i1.tail) > limit {
		panic("item stack merge: 0x51f956 assert, unreachable after the normalize")
	}
	i2 := resolve(p2)
	if i1.id != i2.id || uint8(i1.tail>>16) != uint8(i2.tail>>16) { // 0x51f9d0, 0x51f9dc
		return finish()
	}
	cl := uint8(i1.tail)
	room := limit - cl
	if room == 0 {
		return finish()
	}
	take := room
	if dl := uint8(i2.tail); dl < take {
		take = dl
	}
	i1.tail = i1.tail&^0xff | uint64(cl+take)
	i2.tail = i2.tail&^0xff | uint64(uint8(i2.tail)-take)
	if uint8(i2.tail) != 0 { // 0x51fb20
		if pf != 0 {
			if pf != ismPF {
				panic("item stack merge: flag pointer outside the modelled memory")
			}
			flagByte = 1
		}
		return finish()
	}
	ret = 1
	if p2 == ismHand { // 0x51fa10
		mark0, mark4, mark5 = 0xffffffff, 0xff, 0xffff
		hand, handMark = ismInst{-1, 0}, 0 // 0x55f1e0
		if cursor != 0xffffffff {
			n427ab0++
			m427ab0 = [4]uint32{cursorTex, uint32(tblWord), 0, 0}
			n427ff0++
			cursor, cursorB, cursorC = 0xffffffff, 0, 0
		}
		return finish()
	}
	*i2 = ismInst{-1, 0} // 0x51fa2b
	n520d10++
	m520d10 = [3]uint32{p2, 0, 1}
	if p2 < ismPack || p2 > ismPackEnd || count == 0 {
		return finish()
	}
	k := uint32(0)
	for ; k < count && p2 != ismPack+12*k; k++ {
	}
	if k == count { // 0x51fa7d
		return finish()
	}
	copy(pack[k:count-1], pack[k+1:count]) // 0x51fa8a
	pack[count-1] = ismInst{-1, 0}
	count--
	n4edd20++
	return finish()
}
