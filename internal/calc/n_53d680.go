package calc

// attackBudgetPrepareNode picks the character's preferred attack hand and
// copies each usable hand's attack budget into the turn record.
// Inputs: who, per hand (kind, avail, rank, tie), two equipment slot item
// indexes, four item range bytes, five previous turn-record fields.
// Outputs: the five turn-record fields after the call.
func attackBudgetPrepareNode(in []int64, _ Rand) []int64 {
	if len(in) != 20 {
		panic("attack budget prepare: expected 20 inputs")
	}
	// Per-hand record stride 0x5b: kind +0x1149, avail +0x114d, rank +0x115e, tie +0x1166.
	usable := func(hand int) bool {
		if in[2+4*hand] == 0 {
			return false
		}
		// 0x53d6e8/0x53d7aa call 0x519ac0 with the hand index.
		r := charAttackRangeNode([]int64{int64(hand), 1, in[1], in[2], in[5], in[6], in[9], in[10], in[11], in[12], in[13], in[14]}, nil)
		return r[0] != -1
	}
	out := append([]int64(nil), in[15:20]...) // 0x50, 0x54, 0x58, 0x5c, 0x84 of the turn record
	best := int64(-1)
	bestRank, bestTie := int64(0), int64(-999) // 0x53d69f edi=0; 0x53d6b2 = -999
	for h := 0; h < 2; h++ {
		if !usable(h) {
			continue
		}
		rank, tie := in[3+4*h], int64(int32(in[4+4*h]))
		// 0x53d705..0x53d716: rank unsigned greater, or equal with signed tie greater.
		if rank > bestRank || (rank == bestRank && tie > bestTie) {
			best, bestRank, bestTie = int64(h), rank, tie
		}
	}
	if best == -1 {
		return out // 0x53d73d logs a diagnostic and returns without writing
	}
	out[4] = best // 0x53d772
	for h := 0; h < 2; h++ {
		var v int64
		if usable(h) {
			v = in[3+4*h]
		}
		out[h], out[2+h] = v, v // 0x53d7ce, 0x53d7d8
	}
	return out
}

func init() {
	Nodes["0053d680"] = Node{
		Addr: "0053d680",
		In: []string{"who", "kind0", "avail0", "rank0", "tie0", "kind1", "avail1", "rank1", "tie1", "slotItem0", "slotItem1",
			"item0Range", "item1Range", "item2Range", "item3Range", "prevMain0", "prevOff0", "prevMain1", "prevOff1", "prevHand"},
		Out: []string{"outMain0", "outOff0", "outMain1", "outOff1", "outHand"}, F: attackBudgetPrepareNode,
	}
}
