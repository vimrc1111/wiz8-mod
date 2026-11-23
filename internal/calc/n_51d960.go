package calc

import "fmt"

// invalid_equipment_removal 0x51d960 (cdecl, arg0 character; build/wiz8.asm:387030..387148):
// every equipped item (12 records of 12 bytes at character +0xf5d: i32 id, dword
// +4 whose low byte is the stack count, dword +8) that 0x51d610 declares unusable is
// reported (0x51b5c0 + 0x590950 -> 0x58aad0) and moved: 0x520d10 on a temporary, 0x51fd20
// (hand/slot swap with the empty temporary -> 0x51fe30 move), then 0x51c300 puts the item
// into the backpack (merge via 0x51f900 for stackable items, else first free slot of
// 8 at character +0x1029) and, if that fails, 0x521ef0 puts it into the shared pack
// (12-byte records at 0x685191, count at 0x686901). Which requirement is unmet (stat pair
// +0x7d/+0x7f, skill pair +0x81/+0x83) only selects the message text, so the requirement
// fields are case inputs that drive the branches but reach no output.
//
// Callees: 0x51d610 and 0x51f900 (D4) are address-stubbed in the case; their returns are
// the case inputs r610_n / r900_n (call sequence), recorded arguments and counts are
// outputs. 0x520d10, 0x4edd20, 0x518150 (D4) are recorded the same way, 0x52e690 and 0x58aad0
// (registered 排除) too. 0x51bfd0 (D4) is run through itemWeightNode. 0x517ec0, 0x517f30,
// 0x51fd20, 0x51fe30, 0x51c300, 0x521ef0, 0x5227d0, 0x522640, 0x4edc60, 0x590950 and
// 0x58b410 run natively in the case and are reproduced here. F is the same body as Emu: the
// real 0x51d610 / 0x51f900 need profession, race and merge inputs this world does not
// carry, so F also takes their returns from r610_n / r900_n (composition gap).
//
// Fixture caps (calc panics past them): item ids -1..3 (four definitions at itemsBase);
// 24 calls of 0x51d610, 330 calls of 0x51f900, shared count 2 on entry (14 records out),
// level 0 (division by zero in 0x5227d0). The extension records b8..b11 are the
// backpack-indexed writes at character +0x1089..: a merge into equipment slot k uses
// index k in 0x4edc60's follow-up (0x51c300 tail), so ids there are 0..3. 0x51f900's
// out-flag is not modeled (the stub never sets it). A character that is not one of
// the eight party records returns at once (0x517f30); the party flag +4 is the
// case constant 1.
const (
	ierStride  = 0x1862
	ierParty   = 8
	ierDefs    = 4
	ierSharedC = 2
	ierShared  = 14
	ierN610    = 24
	ierN900    = 12*20 + (2*12 + 11*12/2)
	ierStats   = 8
	ierSkills  = 41
	ierMsgArg1 = 8        // push at 0x5909dc
	ierMsgArg2 = 0x64bf5c // push at 0x5909d7
)

var ierDefFields = []string{"stack", "s0", "m0", "s1", "m1", "k0", "n0", "k1", "n1", "price", "weight"}

var ierIn, ierOut = func() (in, out []string) {
	in = []string{"ch", "who", "partyBase", "itemsBase", "mode", "level", "wOld", "wLimit", "evtPrice", "evtOver", "evtElse",
		"argA", "argB", "ui", "r518", "sharedCount"}
	for i := 0; i < ierStats; i++ {
		in = append(in, fmt.Sprintf("st%d", i))
	}
	for i := 0; i < ierSkills; i++ {
		in = append(in, fmt.Sprintf("sk%d", i))
	}
	rec := func(p string, n int) (l []string) {
		for s := 0; s < n; s++ {
			for d := 0; d < 3; d++ {
				l = append(l, fmt.Sprintf("%s%d.%d", p, s, d))
			}
		}
		return
	}
	in = append(in, rec("e", 12)...)
	in = append(in, rec("b", 12)...)
	in = append(in, rec("sh", ierSharedC)...)
	for k := 0; k < ierParty; k++ {
		in = append(in, fmt.Sprintf("busy%d", k))
	}
	for j := 0; j < ierDefs; j++ {
		for _, f := range ierDefFields {
			in = append(in, fmt.Sprintf("t%d.%s", j, f))
		}
	}
	in = append(in, "tm1.price")
	for i := 0; i < ierN610; i++ {
		in = append(in, fmt.Sprintf("r610_%d", i))
	}
	for i := 0; i < ierN900; i++ {
		in = append(in, fmt.Sprintf("r900_%d", i))
	}
	out = append(out, rec("e", 12)...)
	out = append(out, rec("b", 12)...)
	out = append(out, rec("sh", ierShared)...)
	out = append(out, "shCount")
	for k := 0; k < ierParty; k++ {
		out = append(out, fmt.Sprintf("busy%d", k))
	}
	out = append(out, "wNew", "ui", "c610", "a610", "c520", "o520", "f520", "c4edd20", "c518", "a518", "c900", "c52e",
		"x1", "x2", "x3", "x4", "x5", "c58a", "y1", "y2", "y3")
	return
}()

func init() {
	Nodes["0051d960"] = Node{Addr: "0051d960", In: ierIn, Out: ierOut, F: invalidEquipmentRemovalNode,
		Emu: invalidEquipmentRemovalNode}
}

type ierRec [3]uint32

func (r ierRec) id() int64 { return int64(int32(r[0])) }

var ierEmpty = ierRec{0xffffffff, 0, 0}

type ierWorld struct {
	v          func(string) int64
	ch         uint32
	who        int
	mode       int64
	eq, bp     [12]ierRec
	sh         [ierShared]ierRec
	shCount    int
	busy       [ierParty]int64
	ui, wNew   uint32
	seq610     int
	seq900     int
	c610, a610 int64
	c520, o520 int64
	f520       int64
	c4e, c518  int64
	a518, c900 int64
	c52e       int64
	x          [5]int64
	c58a       int64
}

func (w *ierWorld) def(id int64, f string) int64 {
	if id == -1 && f == "price" {
		return w.v("tm1.price")
	}
	if id < 0 || id >= ierDefs {
		panic("invalid_equipment_removal: item id past fixture cap")
	}
	return w.v(fmt.Sprintf("t%d.%s", id, f))
}

// 0x51d610 stub: the return is the next r610_n; the last argument id is recorded.
func (w *ierWorld) usable(id int64) bool {
	if w.seq610 >= ierN610 {
		panic("invalid_equipment_removal: 0x51d610 call sequence past fixture cap")
	}
	r := w.v(fmt.Sprintf("r610_%d", w.seq610))
	w.seq610++
	w.c610++
	w.a610 = int64(uint32(id))
	return r&0xff != 0
}

// 0x51f900 stub: the return is the next r900_n.
func (w *ierWorld) merges() bool {
	if w.seq900 >= ierN900 {
		panic("invalid_equipment_removal: 0x51f900 call sequence past fixture cap")
	}
	r := w.v(fmt.Sprintf("r900_%d", w.seq900))
	w.seq900++
	w.c900++
	return r&0xff != 0
}

func (w *ierWorld) refresh(owner uint32) {
	w.c520++
	w.o520, w.f520 = int64(owner), 1
}

// 0x51fe30(dst, src, owner, 1): dst empty, src holds an item.
func (w *ierWorld) move(dst, src *ierRec, owner uint32) {
	if dst.id() != -1 || src.id() == -1 {
		panic("invalid_equipment_removal: srAssertFail 0x51fe30")
	}
	src[2] &^= 0xff0000
	*dst = *src
	*src = ierEmpty
	w.refresh(owner)
	w.refresh(owner)
}

// 0x4edc60: recompute the carried weight; a crossing of the limit raises 0x52e690.
func (w *ierWorld) weight() bool {
	var sum int64
	for _, r := range append(append([]ierRec{}, w.eq[:]...), w.bp[:8]...) {
		id := r.id()
		unit := int64(0)
		if id != -1 {
			unit = w.def(id, "weight") & 0xffff
		}
		sum += itemWeightNode([]int64{id, int64(r[1] & 0xff), unit}, nil)[0]
	}
	old, limit := w.wNew, uint32(w.v("wLimit"))
	w.wNew = uint32(sum)
	if old < limit && w.wNew > limit {
		evt := w.v("evtElse")
		if w.mode == 6 {
			evt = w.v("evtOver")
		}
		w.raise(evt)
	}
	return old != w.wNew
}

func (w *ierWorld) raise(evt int64) {
	w.c52e++
	w.x = [5]int64{int64(w.ch), evt & 0xffffffff, 0, w.v("argA") & 0xffffffff, w.v("argB") & 0xffffffff}
}

// 0x5227d0(r, flag, owner) for ids -1..3: flag 0 prices the held item, flag 1 asks 0x518150.
func (w *ierWorld) price(r *ierRec, flag int64) {
	if w.mode == 5 {
		return
	}
	if flag != 0 {
		w.c518++
		w.a518 = 0xffffffff
		return
	}
	if r[2]>>8&1 != 0 {
		return
	}
	if !w.usable(r.id()) {
		return
	}
	level := w.v("level") & 0xffffffff
	if level == 0 {
		panic("invalid_equipment_removal: level 0 divides in 0x5227d0")
	}
	if uint32(w.def(r.id(), "price"))/uint32(level) > 0x1f4 && r[1]>>16&0xff != 0 {
		w.raise(w.v("evtPrice"))
		r[2] |= 1 << 8
	}
}

// 0x51c300(character, tmp): true when the item was placed.
func (w *ierWorld) toBackpack(tmp *ierRec) bool {
	esi := -1
	if w.def(tmp.id(), "stack")&0xff == 1 {
		for k := 0; k < 12 && esi < 0; k++ {
			if w.merges() {
				esi = k
			}
		}
		for k := 0; k < 8 && esi < 0; k++ {
			if w.merges() {
				esi = k
			}
		}
	}
	if esi < 0 {
		for k := 0; k < 8; k++ {
			if w.bp[k].id() == -1 {
				esi = k
				break
			}
		}
		if esi < 0 {
			return false
		}
		w.move(&w.bp[esi], tmp, 0)
	}
	if w.weight() {
		w.c4e++
	}
	if w.mode == 6 {
		w.ui |= 2 << uint(esi)
	}
	w.price(&w.bp[esi], 0)
	return true
}

// 0x521ef0(tmp): the shared pack. A stackable item first tries 0x51f900 on each record; the
// stub never sets the out-flag, so a merge only calls 0x4edd20 and falls to the tail, and
// a miss creates a new first record (the 0x1f4 record limit is above the fixture cap).
func (w *ierWorld) toShared(tmp *ierRec) {
	idx := -1
	if w.def(tmp.id(), "stack")&0xff == 1 {
		for k := 0; k < w.shCount && idx < 0; k++ {
			if w.merges() {
				idx = k
			}
		}
	}
	if idx >= 0 {
		w.c4e++
	} else {
		if w.shCount >= ierShared {
			panic("invalid_equipment_removal: shared pack past fixture cap")
		}
		for k := w.shCount - 1; k >= 0; k-- {
			w.sh[k+1] = w.sh[k]
		}
		w.sh[0] = ierEmpty
		w.move(&w.sh[0], tmp, 0)
		w.shCount++
		w.c4e++
		idx = 0
	}
	if w.mode == 6 {
		w.ui |= 0x7fc00000
	}
	w.price(&w.sh[idx], 1)
}

func invalidEquipmentRemovalNode(in []int64, r Rand) []int64 {
	if len(in) != len(ierIn) {
		panic("invalid_equipment_removal: wrong input count")
	}
	w := ierWorldFrom(in)
	if w.shCount > ierSharedC {
		panic("invalid_equipment_removal: shared count past fixture cap")
	}
	return w.run()
}

// ierWorldFrom builds the callee world from an input vector (no shared-count check).
func ierWorldFrom(in []int64) *ierWorld {
	at := map[string]int{}
	for i, n := range ierIn {
		at[n] = i
	}
	w := &ierWorld{v: func(n string) int64 { return in[at[n]] }}
	w.ch, w.mode = uint32(w.v("ch")), w.v("mode")&0xffffffff
	w.who = int(w.v("who"))
	w.ui = uint32(w.v("ui"))
	w.wNew = uint32(w.v("wOld"))
	w.shCount = int(w.v("sharedCount") & 0xffffffff)
	rec := func(p string, s int) ierRec {
		return ierRec{uint32(w.v(fmt.Sprintf("%s%d.0", p, s))), uint32(w.v(fmt.Sprintf("%s%d.1", p, s))),
			uint32(w.v(fmt.Sprintf("%s%d.2", p, s)))}
	}
	for s := 0; s < 12; s++ {
		w.eq[s], w.bp[s] = rec("e", s), rec("b", s)
	}
	for k := 0; k < ierSharedC; k++ {
		w.sh[k] = rec("sh", k)
	}
	for k := range w.busy {
		w.busy[k] = w.v(fmt.Sprintf("busy%d", k))
	}
	return w
}

// run is the body of 0x51d960 on a built world; the result is the node output.
func (w *ierWorld) run() []int64 {
	slot := -1
	for k := 0; k < ierParty; k++ {
		if w.ch == uint32(w.v("partyBase"))+uint32(k)*ierStride {
			slot = k
		}
	}
	// 0x517f30 returns for a character that is not a party record.
	if slot >= 0 {
		if slot != w.who {
			panic("invalid_equipment_removal: character is a party slot other than who")
		}
		for s := 0; s < 12; s++ {
			if w.eq[s].id() == -1 {
				continue
			}
			if w.usable(w.eq[s].id()) {
				continue
			}
			w.c58a++
			tmp := ierEmpty
			w.refresh(0)
			// 0x51fd20(slot, tmp): tmp is empty, the slot is moved into it.
			w.move(&tmp, &w.eq[s], w.ch)
			w.busy[w.who] = 0
			if !w.toBackpack(&tmp) {
				w.toShared(&tmp)
			}
		}
	}
	out := []int64{}
	put := func(l []ierRec) {
		for _, x := range l {
			out = append(out, int64(x[0]), int64(x[1]), int64(x[2]))
		}
	}
	put(w.eq[:])
	put(w.bp[:])
	put(w.sh[:])
	out = append(out, int64(w.shCount))
	out = append(out, w.busy[:]...)
	y1, y2, y3 := int64(0), int64(0), int64(0)
	if w.c58a > 0 {
		y1, y2, y3 = ierMsgArg1, ierMsgArg2, int64(w.ch+5)
	}
	return append(out, int64(w.wNew), int64(w.ui), w.c610, w.a610, w.c520, w.o520, w.f520, w.c4e, w.c518, w.a518,
		w.c900, w.c52e, w.x[0], w.x[1], w.x[2], w.x[3], w.x[4], w.c58a, y1, y2, y3)
}
