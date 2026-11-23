package calc

import "fmt"

// item_consume_one 0x528ff0(item *, matchById u8, itemId i32): use up one unit of an item.
// The item is found by pointer (or, when matchById is set, by id) first among the
// eight present characters' backpack slots, then among the party pack slots.
// A hit lowers the slot's quantity byte (never below 0); at 0 the slot is cleared
// (0x520070 for a character slot, 0x521cd0 for a pack slot, which also compacts
// the pack and refreshes the shared weight 0x4edd20). A notice is queued: UI
// refresh 0x5adb10 when panel mode is 4, else the message box 0x576030 plus the
// notice state globals and a sound request (0x408860, a no-op with sound off).
// With no hit, a held-item panel (flag @+0x1f9, id @+0x1ed) matching *item lowers
// its quantity; at 0 the flag is cleared and 0x55f1e0 empties the hand struct and
// releases the cursor graphic (0x5493e0/0x549390 read tables, 0x427ab0/0x427ff0
// are UI).
//
// Fixture and environment (see the case _notes): characters at arena 0x20006000
// stride 0x1862, backpack slots at +0x1029 (8 x 12 bytes), present flags at
// arena 0x20005000 stride 0x106, party pack 8 slots at 0x685191 (count @0x686901),
// hand struct at 0x6874cb. 0x650e20=1, 0x62c464=1 (cursor graphic loaded),
// 0x650e50=0, GetTickCount = 4660. Stubbed in the case: 0x576030, 0x5adb10,
// 0x427ab0, 0x427ff0 (exclusions) and 0x520d10, 0x4edd20 (D4 formulas; their
// returns come from inputs and are unused). Pointers outside the modelled
// memory and a party count above 8 panic.
const (
	icnChars    = 8
	icnSlots    = 8
	icnPack     = 8
	icnCharBase = 0x20006000
	icnCharSize = 0x1862
	icnSlotOff  = 0x1029
	icnPackBase = 0x685191
	icnHand     = 0x6874cb
	icnTick     = 4660
)

var icnIn, icnOut []string

func init() {
	icnIn = []string{"ptr", "flag", "itemId", "panelMode", "heldId", "heldFlag", "noHeldClear", "cursorGfx", "cursorB", "cursorC",
		"tblWord", "cursorTex", "notifA", "notifB", "notifDur", "notifT1", "notifT2", "handMark", "handId", "handQw", "handAux", "partyCount"}
	for c := 0; c < icnChars; c++ {
		icnIn = append(icnIn, fmt.Sprintf("present%d", c))
	}
	icnOut = []string{"heldFlag", "cursorGfx", "cursorB", "cursorC", "notifA", "notifB", "notifDur", "notifT1", "notifT2", "handMark", "handId", "handQw", "handAux", "partyCount"}
	for c := 0; c < icnChars; c++ {
		for j := 0; j < icnSlots; j++ {
			for _, w := range []string{"Id", "Qw", "Aux"} {
				icnIn = append(icnIn, fmt.Sprintf("c%ds%d%s", c, j, w))
				icnOut = append(icnOut, fmt.Sprintf("c%ds%d%s", c, j, w))
			}
		}
	}
	for s := 0; s < icnPack; s++ {
		for _, w := range []string{"Id", "Qw", "Aux"} {
			icnIn = append(icnIn, fmt.Sprintf("p%d%s", s, w))
			icnOut = append(icnOut, fmt.Sprintf("p%d%s", s, w))
		}
	}
	icnIn = append(icnIn, "ret520d10", "ret4edd20")
	icnOut = append(icnOut, "m576030a1", "m576030a3", "m576030a4", "m576030a5", "n576030", "m5adb10a1", "n5adb10",
		"m427ab0a1", "m427ab0a2", "m427ab0a3", "m427ab0a4", "n427ab0", "n427ff0", "m520d10a1", "m520d10a2", "m520d10a3", "n520d10", "n4edd20")
	Nodes["00528ff0"] = Node{Addr: "00528ff0", In: icnIn, Out: icnOut, F: itemConsumeOneNode}
}

// icnMem is the modelled memory: character backpack slots, the party pack, the hand struct.
type icnMem map[uint32]byte

func (m icnMem) get(a uint32, n int) uint32 {
	var v uint32
	for i := n - 1; i >= 0; i-- {
		b, ok := m[a+uint32(i)]
		if !ok {
			panic(fmt.Sprintf("item consume: address %#x outside the fixture memory", a+uint32(i)))
		}
		v = v<<8 | uint32(b)
	}
	return v
}

func (m icnMem) put(a uint32, n int, v uint32) {
	for i := 0; i < n; i++ {
		if _, ok := m[a+uint32(i)]; !ok {
			panic(fmt.Sprintf("item consume: address %#x outside the fixture memory", a+uint32(i)))
		}
		m[a+uint32(i)] = byte(v >> (8 * i))
	}
}

func icnSlotAddr(c, j int) uint32 {
	return icnCharBase + uint32(c)*icnCharSize + icnSlotOff + 12*uint32(j)
}

func itemConsumeOneNode(in []int64, r Rand) []int64 {
	if len(in) != len(icnIn) {
		panic("item consume: wrong input count")
	}
	v := map[string]int64{}
	for i, n := range icnIn {
		v[n] = in[i]
	}
	if v["partyCount"] < 0 || v["partyCount"] > icnPack {
		panic("item consume: party pack exceeds the fixture")
	}
	mem := icnMem{}
	fill := func(base uint32, name func(k, w int) string, slots int) {
		for k := 0; k < slots; k++ {
			for w := 0; w < 3; w++ {
				x := uint32(v[name(k, w)])
				for b := 0; b < 4; b++ {
					mem[base+12*uint32(k)+4*uint32(w)+uint32(b)] = byte(x >> (8 * b))
				}
			}
		}
	}
	suf := []string{"Id", "Qw", "Aux"}
	for c := 0; c < icnChars; c++ {
		c := c
		fill(icnCharBase+uint32(c)*icnCharSize+icnSlotOff, func(k, w int) string { return fmt.Sprintf("c%ds%d%s", c, k, suf[w]) }, icnSlots)
	}
	fill(icnPackBase, func(k, w int) string { return fmt.Sprintf("p%d%s", k, suf[w]) }, icnPack)
	fill(icnHand, func(k, w int) string { return []string{"handId", "handQw", "handAux"}[w] }, 1)

	var n576030, n5adb10, n427ab0, n427ff0, n520d10, n4edd20 uint32
	var m520d10 [3]uint32
	var m427ab0 [4]uint32
	var m5adb10 uint32
	var m576030 [4]uint32
	count := uint32(v["partyCount"])
	handMark := uint32(v["handMark"])
	cursor, cursorB, cursorC := uint32(v["cursorGfx"]), uint32(v["cursorB"]), uint32(v["cursorC"])
	heldFlag := uint32(v["heldFlag"])
	notifA, notifB, notifDur := uint32(v["notifA"]), uint32(v["notifB"]), uint32(v["notifDur"])
	notifT1, notifT2 := uint32(v["notifT1"]), uint32(v["notifT2"])
	ptr, flag, id := uint32(v["ptr"]), uint32(v["flag"])&0xff, uint32(v["itemId"])

	// 0x55f1e0: empty the hand struct and release the cursor graphic.
	clearHeld := func() {
		mem.put(icnHand, 4, 0xffffffff)
		mem.put(icnHand+4, 4, 0)
		mem.put(icnHand+8, 4, 0)
		handMark = 0
		if cursor != 0xffffffff {
			n427ab0++
			m427ab0 = [4]uint32{uint32(v["cursorTex"]), uint32(v["tblWord"]) & 0xffff, 0, 0}
			n427ff0++
			cursor, cursorB, cursorC = 0xffffffff, 0, 0
		}
	}
	// 0x528ff0 tail shared by both hits: queue the notice.
	notice := func() {
		if v["panelMode"] == 4 {
			n5adb10++
			m5adb10 = 0
			return
		}
		n576030++
		m576030 = [4]uint32{1, 0, 0xffffffff, 0x47}
		notifB, notifDur, notifT1, notifT2, notifA = 0, 0x7d0, icnTick, icnTick, 1
	}
	dec := func(a uint32) { // quantity byte at a+4, never below 0
		if q := mem.get(a+4, 1); q > 0 {
			mem.put(a+4, 1, q-1)
		}
	}
	match := func(a uint32) bool {
		if mem.get(a, 4) == 0xffffffff {
			return false
		}
		return a == ptr || (flag != 0 && id == mem.get(a, 4))
	}
	done := false
	for c := 0; c < icnChars && !done; c++ {
		if v[fmt.Sprintf("present%d", c)]&0xff == 0 {
			continue
		}
		for j := 0; j < icnSlots; j++ {
			a := icnSlotAddr(c, j)
			if !match(a) {
				continue
			}
			dec(a)
			if mem.get(a+4, 1) == 0 { // 0x520070: clear the slot, refresh the hands
				mem.put(a, 4, 0)
				mem.put(a+4, 4, 0)
				mem.put(a+8, 4, 0)
				mem.put(a, 4, 0xffffffff)
				n520d10++
				m520d10 = [3]uint32{a, icnCharBase + uint32(c)*icnCharSize, 1}
			}
			notice()
			done = true
			break
		}
	}
	for k := uint32(0); k < count && !done; k++ {
		a := icnPackBase + 12*k
		if !match(a) {
			continue
		}
		dec(a)
		if mem.get(a+4, 1) == 0 { // 0x521cd0: clear, compact, shrink, refresh weight
			mem.put(a, 4, 0)
			mem.put(a+4, 4, 0)
			mem.put(a+8, 4, 0)
			mem.put(a, 4, 0xffffffff)
			n520d10++
			m520d10 = [3]uint32{a, 0, 1}
			for s := k; s+1 < count; s++ {
				for b := uint32(0); b < 12; b++ {
					mem[icnPackBase+12*s+b] = mem[icnPackBase+12*(s+1)+b]
				}
			}
			last := icnPackBase + 12*(count-1)
			mem.put(last, 4, 0)
			mem.put(last+4, 4, 0)
			mem.put(last+8, 4, 0)
			mem.put(last, 4, 0xffffffff)
			count--
			n4edd20++
		}
		notice()
		done = true
	}
	if !done && heldFlag&0xff != 0 && ptr != 0 && uint32(v["heldId"]) == mem.get(ptr, 4) {
		dec(ptr)
		if mem.get(ptr+4, 1) == 0 {
			heldFlag = heldFlag &^ 0xff
			if v["noHeldClear"]&0xff == 0 {
				clearHeld()
			}
		}
	}

	o := []int64{int64(heldFlag & 0xff), int64(int32(cursor)), int64(cursorB), int64(cursorC), int64(notifA & 0xff), int64(notifB & 0xff),
		int64(int32(notifDur)), int64(notifT1), int64(notifT2), int64(handMark & 0xff), int64(int32(mem.get(icnHand, 4))), int64(mem.get(icnHand+4, 4)), int64(mem.get(icnHand+8, 4)), int64(count)}
	for c := 0; c < icnChars; c++ {
		for j := 0; j < icnSlots; j++ {
			a := icnSlotAddr(c, j)
			o = append(o, int64(int32(mem.get(a, 4))), int64(mem.get(a+4, 4)), int64(mem.get(a+8, 4)))
		}
	}
	for s := uint32(0); s < icnPack; s++ {
		a := icnPackBase + 12*s
		o = append(o, int64(int32(mem.get(a, 4))), int64(mem.get(a+4, 4)), int64(mem.get(a+8, 4)))
	}
	o = append(o, int64(m576030[0]), int64(m576030[1]), int64(int32(m576030[2])), int64(m576030[3]), int64(n576030), int64(m5adb10), int64(n5adb10))
	for _, x := range m427ab0 {
		o = append(o, int64(x))
	}
	o = append(o, int64(n427ab0), int64(n427ff0))
	for _, x := range m520d10 {
		o = append(o, int64(x))
	}
	return append(o, int64(n520d10), int64(n4edd20))
}
