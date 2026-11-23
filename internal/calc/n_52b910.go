package calc

// characterSpHealDistributeNode restores the six realm pools and returns the
// count passed to the notification (0x590950). Notification formatting has no
// rule-state effect; the replay uses empty names and an absent renderer.
func characterSpHealDistributeNode(in []int64, r Rand) []int64 {
	if len(in) != 14 || in[0] < 0 || in[0] > 7 {
		panic("SP heal: expected party index and thirteen u32 inputs")
	}
	for _, v := range in[1:] {
		if v < 0 || v > 4294967295 {
			panic("SP heal: input outside u32")
		}
	}
	// Six realms, maximum +0xb25 and current +0xb45 (0x52b93b..0x52b955).
	// Subtraction, comparisons and pool increments all use unsigned 32 bits.
	type pool struct {
		realm int
		gap   uint32
	}
	var pools [6]pool
	out := make([]int64, 7)
	for i := range pools {
		pools[i] = pool{i, uint32(in[2+i]) - uint32(in[8+i])}
		out[i] = in[8+i]
	}
	// 0x52b8e0 compares descending unsigned gaps. For six 8-byte records,
	// MSVCRT qsort uses shortsort (notes/re/emu.md, qsort): select the first
	// smallest remaining gap and swap it to the end. Preserve equal-key order.
	for end := len(pools) - 1; end > 0; end-- {
		smallest := 0
		for i := 1; i <= end; i++ {
			if pools[i].gap < pools[smallest].gap {
				smallest = i
			}
		}
		pools[smallest], pools[end] = pools[end], pools[smallest]
	}
	remaining := uint64(in[1])
	for remaining > 0 && pools[0].gap > 0 {
		// 0x52b97c..0x52b9ce reduces the leading equal-gap group in its
		// fixed sorted order. Batch complete rounds down to the next group;
		// this also handles full-width budgets without billions of iterations.
		k := 1
		for k < len(pools) && pools[k].gap == pools[0].gap {
			k++
		}
		next := uint32(0)
		if k < len(pools) {
			next = pools[k].gap
		}
		rounds := min(uint64(pools[0].gap-next), remaining/uint64(k))
		if rounds == 0 {
			// An incomplete final round visits the first remaining pools.
			k = int(remaining)
			rounds = 1
		}
		for i := 0; i < k; i++ {
			p := &pools[i]
			p.gap -= uint32(rounds)
			out[p.realm] = int64(uint32(out[p.realm]) + uint32(rounds))
		}
		remaining -= rounds * uint64(k)
	}
	// The last next-gap read sees the adjacent character-index argument
	// (0x52b991); its equality only changes whether the loop restarts via
	// 0x52b9c3 or 0x52b9cc, with identical pool updates and termination.
	out[6] = in[1] - int64(remaining)
	return out
}

func init() {
	Nodes["0052b910"] = Node{
		Addr: "0052b910",
		In: []string{"characterIndex", "healAmount", "fireMaximumSP", "waterMaximumSP",
			"airMaximumSP", "earthMaximumSP", "mentalMaximumSP", "divineMaximumSP",
			"fireCurrentSP", "waterCurrentSP", "airCurrentSP", "earthCurrentSP", "mentalCurrentSP", "divineCurrentSP"},
		Out: []string{"fireCurrentSP", "waterCurrentSP", "airCurrentSP", "earthCurrentSP", "mentalCurrentSP", "divineCurrentSP", "actualRestoredSP"},
		F:   characterSpHealDistributeNode,
	}
}
