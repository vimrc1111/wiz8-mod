package calc

import (
	"fmt"
	"strconv"
)

// The runtime fixture has two characters, six independent item definitions,
// and six shared inventory slots. Each character exposes all twelve equipment
// and eight backpack slots. Slot words retain quantity and unrelated instance
// flags so partial destruction cannot accidentally clear the latter.
// Layout: 0x551139 (character stride 0x1862), 0x551154/0x55122f
// (slot starts 0xf5d/0x1029), 0x551221 (12-byte slots), 0x55116a
// (269-byte definitions, category at +0x3e).
const (
	destructionHeader = 16
	destructionSlots  = 46
	destructionHands  = destructionHeader + 3*destructionSlots
)

func init() {
	in := []string{"power", "targetCount", "target0", "target1", "silent", "messageStarted", "effectReported", "appendMode", "partyCount", "clearedItemCategory"}
	for i := 0; i < 6; i++ {
		in = append(in, fmt.Sprintf("item%dCategory", i))
	}
	for i := 0; i < destructionSlots; i++ {
		for _, suffix := range []string{"Item", "QuantityState", "Aux"} {
			in = append(in, fmt.Sprintf("slot%d%s", i, suffix))
		}
	}
	for ch := 0; ch < 2; ch++ {
		for hand := 0; hand < 2; hand++ {
			in = append(in, fmt.Sprintf("character%dHand%d", ch, hand))
		}
	}
	out := []string{"messageStarted", "effectReported", "appendMode", "partyCount"}
	out = append(out, in[destructionHeader:]...)
	for i := 0; i < 8; i++ {
		out = append(out, fmt.Sprintf("messageChar%d", i))
	}
	Nodes["005510b0"] = Node{Addr: "005510b0", In: in, Out: out, F: spellItemDestructionNode}
}

// 0x68517c=0 and 0x68ec78=0 in the fixture disable live character bookkeeping
// and message rendering. The original removal helpers, shared-slot compaction,
// hand reset and message formatter still execute. Message formats are synthetic
// runtime inputs A%u/B%u/C%u, not text extracted from the executable.
// No executable data table is needed by this rule in that context.
func spellItemDestructionNode(in []int64, r Rand) []int64 {
	if len(in) != destructionHands+4 {
		panic("item destruction: expected 158 inputs")
	}
	if in[0] < -2147483648 || in[0] > 2147483647 || in[1] < -2147483648 || in[1] > 2 {
		panic("item destruction: expected i32 power and at most two targets")
	}
	for _, v := range in[2:4] {
		if v < 0 || v > 1 {
			panic("item destruction: target outside two-character fixture")
		}
	}
	for _, i := range []int{4, 5, 6, 7, 9, 10, 11, 12, 13, 14, 15} {
		if in[i] < 0 || in[i] > 255 {
			panic("item destruction: expected byte flag or category")
		}
	}
	if in[8] < 0 || in[8] > 6 {
		panic("item destruction: shared inventory exceeds fixture")
	}
	for s := 0; s < destructionSlots; s++ {
		i := destructionHeader + 3*s
		if in[i] < -1 || in[i] > 5 {
			panic("item destruction: item outside runtime definitions")
		}
		for _, v := range in[i+1 : i+3] {
			if v < 0 || v > 4294967295 {
				panic("item destruction: expected u32 instance state")
			}
		}
	}
	for _, v := range in[destructionHands:] {
		if v < 0 || v > 4294967295 {
			panic("item destruction: expected u32 hand type")
		}
	}
	a := append([]int64(nil), in...)
	// 0x5510b7: the percentage threshold wraps at 32 bits, then the comparison
	// at 0x5511ae is unsigned. Negative powers are not clamped to zero.
	threshold := uint32(a[0]) * 10
	var lost [3]uint32
	category := func(id int64) int64 {
		if id == -1 {
			// 0x5511e9/0x5512c7/0x5513c5 reread the ID after removal.
			// The cleared ID reads allocation padding before the definitions.
			return a[9]
		}
		return a[10+id]
	}
	process := func(slot int) {
		i := destructionHeader + 3*slot
		if a[i] == -1 {
			return
		}
		kind := category(a[i])
		// Destructible category codes: 0x551181..0x55118f.
		if kind != 16 && kind != 19 && kind != 20 {
			return
		}
		quantity := uint32(uint8(a[i+1]))
		var n uint32
		for j := uint32(0); j < quantity; j++ {
			if uint32(r.R(100)) < threshold { // 0x5511a0, one roll per unit.
				n++
			}
		}
		if n != 0 {
			if n == quantity {
				// 0x5200b0..0x5200c4 clears all twelve instance bytes.
				a[i], a[i+1], a[i+2] = -1, 0, 0
				if slot < 40 {
					ch, equipped := slot/20, slot%20
					if equipped == 6 || equipped == 7 {
						// 0x520d26/0x520d3d and 0x52021d/0x52029e:
						// only the cleared hand's type becomes zero.
						a[destructionHands+2*ch+equipped-6] = 0
					}
				} else {
					// 0x52012e..0x520193 shifts remaining shared slots left,
					// clears the old tail, and decrements the live count.
					end := destructionHeader + 3*(40+int(a[8]))
					copy(a[i:end-3], a[i+3:end])
					a[end-3], a[end-2], a[end-1] = -1, 0, 0
					a[8]--
				}
			} else {
				// 0x5511e4/0x5512c2/0x5513c0 writes only the quantity byte.
				a[i+1] = int64(uint32(a[i+1])&0xffffff00 | (quantity - n))
			}
		}
		// Counting deliberately uses the *new* category, including the next
		// shared item after compaction, rather than the destroyed item's type.
		switch category(a[i]) {
		case 16:
			lost[0] += n
		case 20:
			lost[1] += n
		case 19:
			lost[2] += n
		}
	}
	if a[1] > 0 {
		for target := 0; target < int(a[1]); target++ {
			first := 20 * int(a[2+target])
			for slot := first; slot < first+20; slot++ {
				process(slot)
			}
		}
		// 0x5513fd advances even after removal, skipping the shifted successor.
		for slot := 0; slot < int(a[8]); slot++ {
			process(40 + slot)
		}
	}
	var message [8]int64
	if lost[0]+lost[1]+lost[2] != 0 {
		for k, n := range lost {
			if n == 0 {
				continue
			}
			// 0x551442/0x55146a/0x5514d2; 0x5905c0 retains this byte.
			a[7] = 0
			if a[4] == 0 {
				a[7] = 1
			}
			// 0x517a90 formats into 0x689fd0. Preserve bytes beyond the
			// new terminator, matching the CRT formatter's scratch buffer.
			text := string(rune('A'+k)) + strconv.FormatUint(uint64(n), 10)
			for i, ch := range text {
				message[i] = int64(ch)
			}
			message[len(text)] = 0
			a[5] = 1 // 0x5514de: message has started.
		}
		a[6] = 1 // 0x5514ef: effect has a report.
	}
	out := []int64{a[5], a[6], a[7], a[8]}
	out = append(out, a[destructionHeader:]...)
	return append(out, message[:]...)
}
