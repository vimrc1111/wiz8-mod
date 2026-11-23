package calc

import (
	"fmt"
	"math"
)

func init() {
	in := []string{"gameTime"}
	for s := 2; s <= 7; s++ {
		in = append(in, fmt.Sprintf("slot%dActive", s), fmt.Sprintf("slot%dTotalLevel", s))
	}
	in = append(in, "recordCount", "levelLen", "weightLen", "timeLen")
	for _, a := range []string{"level", "weight", "time"} {
		for i := 0; i < encounterTableSelectMaxRecords; i++ {
			in = append(in, fmt.Sprintf("%sByte%d", a, i))
		}
	}
	in = append(in, "resultCapacity")
	out := []string{"resultCount", "resultCapacityAfter", "resultPointer", "returnCount"}
	for i := 0; i < encounterTableSelectListWords; i++ {
		out = append(out, fmt.Sprintf("initialBuffer%d", i))
	}
	for i := 0; i < encounterTableSelectHeapWords; i++ {
		out = append(out, fmt.Sprintf("grownBuffers%d", i))
	}
	Nodes["0048b9a0"] = Node{Addr: "0048b9a0", In: in, Out: out, F: encounterTableSelectNode}
}

const (
	// Fixture caps (not game limits): more records panic instead of
	// returning a silently wrong answer. 8 records need at most 8 grows,
	// 8 * 8 .. 32 bytes of bump heap = 40 words.
	encounterTableSelectMaxRecords = 8
	encounterTableSelectListWords  = 16
	encounterTableSelectHeapWords  = 40
	encounterTableSelectListBase   = 0x20000500 // fixture address of the caller's initial buffer
	encounterTableSelectHeapBase   = 0x50000000 // emu bump heap start
	encounterTableSelectDecoy      = 0x7100     // fixture decoy word k of the initial buffer
)

// encounterTableSelectNode picks the encounter-table rows (exe 0x48b9a0) whose
// level/weight fit this roll's category and time slot, into the result list at
// result+0xc/+4 (count)/+8 (capacity).
// Table: record count at +4; three u8 arrays as (length, pointer): level
// (+0x34/+0x3c, 0x48ba44), weight (+0x14/+0x1c, 0x48ba65), time slot
// (+0x24/+0x2c, 0x48bad0). A read at index >= length returns element 0
// (exe 0x48ba32..0x48ba3c). Input order: gameTime (global 0x6874f7), the 12
// party-average inputs (partyAverageLevelFloatNode), recordCount, three lengths,
// 3x8 array bytes, initial result capacity.
// The result list grows to exactly count+1 through 0x4addf0, whose allocator
// is the emulator's 8-byte-aligned bump heap (re/emu/emu.py); that is modelled
// here by address so list contents are compared after growth.
func encounterTableSelectNode(in []int64, r Rand) []int64 {
	const max = encounterTableSelectMaxRecords
	if len(in) != 1+12+4+3*max+1 {
		panic("encounter table select: input arity")
	}
	t, n, capacity := in[0], in[13], in[len(in)-1]
	lens := in[14:17]
	arr := [3][]int64{in[17 : 17+max], in[17+max : 17+2*max], in[17+2*max : 17+3*max]}
	if t < 0 || t > 1<<32-1 || n < -1<<31 || n > max || capacity < -1<<31 || capacity > 1<<31-1 {
		panic("encounter table select: record count above fixture cap or value out of range")
	}
	for _, l := range lens {
		if l < -1<<31 || l > 1<<31-1 {
			panic("encounter table select: length out of i32")
		}
	}
	for _, a := range arr {
		for _, b := range a {
			if b < 0 || b > 255 {
				panic("encounter table select: byte out of range")
			}
		}
	}
	// Out of range index reads element 0 (exe 0x48ba32.. pattern).
	at := func(k int, i int64) int64 {
		if i < lens[k] {
			return arr[k][i]
		}
		return arr[k][0]
	}
	// Time window (0x48b9b2/0x48b9b9, unsigned): inside it the "night" flag
	// is set and table time slot 0 matches; outside, slot 1 matches.
	timeSlot := int64(1)
	if t > 0x112a880 && t <= 0x4b87f00 {
		timeSlot = 0
	}
	roll := int64(r.R(100)) // 0x48b9d7
	avg := math.Float64frombits(uint64(partyAverageLevelFloatNode(in[1:13], r)[0]))
	// float32 slot at 0x48b9f6: floor(avg + 0.49f), 0.49f at exe 0x5eca8c.
	level := float64(float32(math.Floor(avg + float64(float32(0.49)))))
	var category int64
	switch {
	case roll <= 3:
		category = 3
	case roll <= 10:
		category = 7
	case roll <= 30:
		category = 20
	default:
		category = 70
	}

	mem := map[int64]int64{}
	for k := int64(0); k < encounterTableSelectListWords; k++ {
		mem[encounterTableSelectListBase+4*k] = encounterTableSelectDecoy + k
	}
	var count int64
	ptr, heap := int64(encounterTableSelectListBase), int64(encounterTableSelectHeapBase)
	// exe 0x48bafc..0x48bb13 and 0x4addf0: grow to count+1, then append.
	// The allocator-failure return of 0x4addf0 cannot happen (emu heap
	// exhaustion is an error), see cov-exempt.
	appendIndex := func(i int64) {
		if count+1 > capacity {
			size := 4 * (count + 1)
			if size%8 != 0 {
				size += 8 - size%8
			}
			if size < 8 {
				size = 8
			}
			for j := int64(0); j < count; j++ {
				mem[heap+4*j] = mem[ptr+4*j]
			}
			ptr, capacity = heap, count+1
			heap += size
		}
		mem[ptr+4*count] = i
		count++
	}
	timeOK := func(i int64) bool { k := at(2, i); return k == 2 || k == timeSlot }
	for i := int64(0); i < n; i++ { // 0x48ba32 loop
		// level / party level ratio: x87 fild/fdiv by the float32 slot (0x48ba44..0x48ba54).
		y := math.Abs(1 - float64(at(0, i))/level)
		if y >= 0.5 { // not-below also false for NaN, which proceeds (fcom flags)
			continue
		}
		// 0x5eca88 = 1.8f, thresholds 3.0 / 10.0 / 30.0 (0x5ec52c, 0x5ebc88, 0x5eca84).
		p := (1 - y*float64(float32(1.8))) * float64(at(1, i))
		var c int64
		switch {
		case !(p > 3):
			c = 3
		case !(p > 10):
			c = 7
		case !(p > 30):
			c = 20
		default:
			c = 70
		}
		if c == category && timeOK(i) {
			appendIndex(i)
		}
	}
	if count == 0 { // fallback pass at 0x48bb26: no level test
		for i := int64(0); i < n; i++ {
			if at(1, i) == category && timeOK(i) {
				appendIndex(i)
			}
		}
	}
	if count > 1 { // 0x48bb9f: move one random entry to the front
		if k := int64(r.R(int(count))); k != 0 {
			e := mem[ptr+4*k]
			for j := k; j < count-1; j++ {
				mem[ptr+4*j] = mem[ptr+4*(j+1)]
			}
			count--
			for j := count; j >= 1; j-- {
				mem[ptr+4*j] = mem[ptr+4*(j-1)]
			}
			mem[ptr] = e
			count++
		}
	}
	out := []int64{count, capacity, ptr, count}
	for k := int64(0); k < encounterTableSelectListWords; k++ {
		out = append(out, mem[encounterTableSelectListBase+4*k])
	}
	for k := int64(0); k < encounterTableSelectHeapWords; k++ {
		out = append(out, mem[encounterTableSelectHeapBase+4*k])
	}
	return out
}
