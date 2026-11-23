package calc

import "fmt"

// Fixture caps (calc panics past them, never a silent answer): shared pack
// records 14 (count at exe 0x686901; records at 0x685191, 12 bytes each), item
// definitions 4 (instance ids -1..3; the case maps ids to a 4-entry table at 0x68516c).
const (
	irrChars   = 8  // roster loop end 0x830 / 0x106 (exe 0x522f1f)
	irrEquip   = 12 // equipment records at character +0xf5d, 12 bytes (exe 0x522f3a)
	irrPack    = 8  // backpack records at character +0x1029 (exe 0x522f62)
	irrSlots   = irrEquip + irrPack
	irrShared  = 14 // fixture cap on the shared pack count
	irrDefs    = 4  // fixture cap on item definition ids
	irrCharSz  = 0x1862
	irrHead    = 14 // packN, charsBase, 8 active flags, 4 definition types
	irrOutHead = 9  // 8 recorded 0x51d960 arguments + call count
)

func init() {
	in := []string{"packN", "charsBase"}
	for i := 0; i < irrChars; i++ {
		in = append(in, fmt.Sprintf("active%d", i))
	}
	for d := 0; d < irrDefs; d++ {
		in = append(in, fmt.Sprintf("type%d", d))
	}
	var slotNames []string
	for i := 0; i < irrChars; i++ {
		for s := 0; s < irrEquip; s++ {
			slotNames = append(slotNames, fmt.Sprintf("c%deq%d", i, s))
		}
		for s := 0; s < irrPack; s++ {
			slotNames = append(slotNames, fmt.Sprintf("c%dbp%d", i, s))
		}
	}
	for k := 0; k < irrShared; k++ {
		slotNames = append(slotNames, fmt.Sprintf("pack%d", k))
	}
	var out []string
	for k := 0; k < irrChars; k++ {
		out = append(out, fmt.Sprintf("call%d_arg", k))
	}
	out = append(out, "call_count")
	for _, n := range slotNames {
		in = append(in, n+"_id", n+"_c4", n+"_c5")
		out = append(out, n+"_c4", n+"_c5")
	}
	for i := 0; i < irrChars; i++ {
		in = append(in, fmt.Sprintf("unusable%d", i))
	}
	Nodes["00522ef0"] = Node{Addr: "00522ef0", In: in, Out: out, F: inventoryRuleRefreshNode,
		Emu: func(in []int64, r Rand) []int64 { return inventoryRuleRefresh(in, r, false) }}
}

// inventoryRuleRefreshNode is exe 0x522ef0 with the real 0x51d960 composed in.
func inventoryRuleRefreshNode(in []int64, r Rand) []int64 { return inventoryRuleRefresh(in, r, true) }

// inventoryRuleRefresh is exe 0x522ef0: for every active roster entry
// (flag byte != 0, exe 0x522efb) call 0x51d960 with the character pointer
// (charsBase + i*0x1862); then, for each active character, normalize the 12
// equipment and 8 backpack records whose id is not -1 with 0x522e80 (exe
// 0x522f4c, 0x522f74), and finally the first shared-pack records (exe 0x522fb5).
//
// Emu (real=false) matches the case, which address-stubs 0x51d960: its argument
// and call count are outputs and its item moves are not modeled. F (real=true)
// runs the real invalidEquipmentRemovalNode world for each call in order, so the
// later loops read the moved equipment, backpack, and the shared pack (records
// and count at 0x685191 / 0x686901) it grew. The callee's own inputs that this
// node does not carry are fixed: mode 5 (no pricing), level 1, weight limit 0,
// no stackable definitions, no busy flags. Which equipment 0x51d610 declares
// unusable is the unusableN input (bit s = equipment slot s), the composition
// gap named in n_51d960.go. F panics past 0x51d960's caps (shared pack 14 records,
// and this node's 14-record shared table).
func inventoryRuleRefresh(in []int64, r Rand, real bool) []int64 {
	n := irrHead + 3*(irrChars*irrSlots+irrShared) + irrChars
	if len(in) != n {
		panic("inventoryRuleRefresh: bad input length")
	}
	packN, base := in[0], in[1]
	if packN < 0 || packN > irrShared {
		panic("inventoryRuleRefresh: shared pack count past fixture cap")
	}
	types := in[2+irrChars : irrHead]
	slots := append([]int64(nil), in[irrHead:n-irrChars]...) // id,c4,c5 per slot; F moves items in it
	active := func(i int) bool { return in[2+i]&0xff != 0 }
	out := make([]int64, irrOutHead, irrOutHead+2*(irrChars*irrSlots+irrShared))
	norm := func(k int) (int64, int64) { // slot k: id, c4, c5 at slots[3k..]
		id, c4, c5 := slots[3*k], slots[3*k+1], slots[3*k+2]
		if id == -1 {
			o := itemCountNormalizationNode([]int64{id, 0, c4, c5}, r)
			return o[0], o[1]
		}
		if id < 0 || id >= irrDefs {
			panic("inventoryRuleRefresh: item id past fixture cap")
		}
		o := itemCountNormalizationNode([]int64{id, types[id], c4, c5}, r)
		return o[0], o[1]
	}
	shared := irrChars * irrSlots // first shared record in slots
	count := int(packN)
	calls := 0
	for i := 0; i < irrChars; i++ {
		if !active(i) {
			continue
		}
		out[calls] = int64(uint32(base + int64(i)*irrCharSz))
		calls++
		if real {
			count = irrMove(in, slots, i, base, count, shared)
		}
	}
	out[8] = int64(calls)
	for i := 0; i < irrChars; i++ {
		for s := 0; s < irrSlots; s++ {
			k := i*irrSlots + s
			if !active(i) {
				out = append(out, slots[3*k+1], slots[3*k+2])
				continue
			}
			c4, c5 := norm(k)
			out = append(out, c4, c5)
		}
	}
	for k := 0; k < irrShared; k++ {
		kk := shared + k
		if k >= count {
			out = append(out, slots[3*kk+1], slots[3*kk+2])
			continue
		}
		c4, c5 := norm(kk)
		out = append(out, c4, c5)
	}
	return out
}

// irrMove runs the real 0x51d960 world for character i on slots (in place),
// with the shared pack taken from and written back to the shared records.
// It returns the new shared count.
func irrMove(in, slots []int64, i int, base int64, count, shared int) int {
	at := map[string]int{}
	for k, nm := range ierIn {
		at[nm] = k
	}
	v := make([]int64, len(ierIn))
	set := func(nm string, x int64) { v[at[nm]] = x }
	set("ch", int64(uint32(base+int64(i)*irrCharSz)))
	set("who", int64(i))
	set("partyBase", base)
	set("mode", 5)
	set("level", 1)
	set("sharedCount", 0)
	mask := in[len(in)-irrChars+i]
	seq := 0
	rec := func(p string, s int, id, c4, c5 int64) {
		set(fmt.Sprintf("%s%d.0", p, s), id)
		set(fmt.Sprintf("%s%d.1", p, s), c4&0xff|(c5&0xff)<<8)
	}
	for s := 0; s < 12; s++ {
		k := i*irrSlots + s
		rec("e", s, slots[3*k], slots[3*k+1], slots[3*k+2])
		if slots[3*k] != -1 {
			if mask>>uint(s)&1 == 0 { // usable: 0x51d610 returns non-zero
				set(fmt.Sprintf("r610_%d", seq), 1)
			}
			seq++
		}
	}
	for s := 0; s < 12; s++ {
		if s < irrPack {
			k := i*irrSlots + irrEquip + s
			rec("b", s, slots[3*k], slots[3*k+1], slots[3*k+2])
		} else {
			rec("b", s, -1, 0, 0)
		}
	}
	w := ierWorldFrom(v)
	w.shCount = count
	for k := 0; k < irrShared; k++ {
		kk := shared + k
		w.sh[k] = ierEmpty
		if k < count {
			w.sh[k] = ierRec{uint32(slots[3*kk]), uint32(slots[3*kk+1]&0xff | (slots[3*kk+2]&0xff)<<8), 0}
		}
	}
	for k := irrShared; k < ierShared; k++ {
		w.sh[k] = ierEmpty
	}
	w.run()
	if w.shCount > irrShared {
		panic("inventoryRuleRefresh: shared pack grew past this node's 14-record table")
	}
	put := func(k int, x ierRec) {
		slots[3*k], slots[3*k+1], slots[3*k+2] = x.id(), int64(x[1]&0xff), int64(x[1]>>8&0xff)
	}
	for s := 0; s < 12; s++ {
		put(i*irrSlots+s, w.eq[s])
	}
	for s := 0; s < irrPack; s++ {
		put(i*irrSlots+irrEquip+s, w.bp[s])
	}
	for k := 0; k < w.shCount; k++ {
		put(shared+k, w.sh[k])
	}
	return w.shCount
}
