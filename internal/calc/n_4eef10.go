package calc

import "strconv"

func init() {
	in := []string{"xp", "announceSide", "statIndex"}
	for i := 0; i < 8; i++ {
		for _, field := range []string{"slotActive", "currentHP", "mainStatus", "xpPool"} {
			in = append(in, field+strconv.Itoa(i))
		}
	}
	var out []string
	for i := 0; i < 8; i++ {
		out = append(out, "xpPool"+strconv.Itoa(i))
	}
	// exe 0x686b79 stride 0x21: 47 entries, the first dword of each is the one touched.
	for i := 0; i < 47; i++ {
		in = append(in, "statEntry"+strconv.Itoa(i))
		out = append(out, "statEntry"+strconv.Itoa(i))
	}
	in = append(in, "awardedFlag", "queueCount", "queueCap")
	out = append(out, "awardedFlag", "queueCount", "queueCap", "eventXp", "eventSide")
	Nodes["004eef10"] = Node{Addr: "004eef10", In: in, Out: out, F: xpAwardSaturatingNode}
}

// xpAwardSaturatingNode adds xp to every eligible party member's XP pool (saturating at
// 0xffffffff), bumps the per-roster stat entry, flags the award and queues a type-0x17 event
// carrying xp and the announce side. In: xp, announceSide, statIndex, 8x(slotActive,
// currentHP, mainStatus, xpPool), 47 statEntry, awardedFlag, queueCount, queueCap.
// Reads no executable data table. The event's message text is not a rule output.
func xpAwardSaturatingNode(in []int64, r Rand) []int64 {
	if len(in) != 3+32+47+3 {
		panic("xpAward: unexpected input count")
	}
	xp, side, idx := in[0], in[1]&0xff, in[2]
	if xp < 0 || xp > 0xffffffff || idx < -2147483648 || idx > 2147483647 {
		panic("xpAward: expected u32 xp and i32 statIndex")
	}
	if idx < 0 {
		panic("xpAward: negative statIndex addresses memory outside the modeled 47-entry table")
	}
	out := make([]int64, 0, 8+47+5)
	pool := make([]int64, 8)
	stat := append([]int64(nil), in[35:82]...)
	flag := in[82]
	for i := 0; i < 8; i++ {
		active, hp, status, cur := in[3+4*i], in[4+4*i], in[5+4*i], in[6+4*i]
		pool[i] = cur
		// exe 0x4eef29/0x4eef36/0x4eef3a/0x4eef44: slot present, HP nonzero,
		// unsigned status below 18, xp nonzero.
		if active&0xff == 0 || hp == 0 || status >= 0x12 || xp == 0 {
			continue
		}
		// exe 0x4eef4f-0x4eef5f: unsigned wrap saturates to 0xffffffff.
		if sum := cur + xp; sum <= 0xffffffff {
			pool[i] = sum
		} else {
			pool[i] = 0xffffffff
		}
		// exe 0x4eef6f: signed index below 0x2f; unlike the pool, a wrapping sum leaves the entry unchanged.
		if idx < 0x2f {
			if sum := stat[idx] + xp; sum <= 0xffffffff {
				stat[idx] = sum
			}
		}
		flag = 1
	}
	out = append(out, pool...)
	out = append(out, stat...)
	// exe 0x528a80 (queue push): count+1, capacity grows to count+1 when smaller (signed).
	count, capacity := int32(in[83]), int32(in[84])
	if count+1 > capacity {
		capacity = count + 1
	}
	count++
	// exe 0x4eefb9-0x4eeffd: event payload is (xp, side byte).
	return append(out, flag, int64(count), int64(capacity), xp, side)
}
