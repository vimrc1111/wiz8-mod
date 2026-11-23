package calc

import "strconv"

// Input layout of randomPartyAttackTargetNode (exe 0x51b0a0). Index helpers keep the
// eight-slot groups readable; every group is eight consecutive inputs.
const (
	rpatSlot     = 0  // arg1: attacker slot 0..7 (0x51b0a4)
	rpatDesired  = 1  // arg2: wanted disposition, compared as a byte (0x51b110..0x51b117)
	rpatPresence = 2  // action record byte +0 per slot (0x51b0dd)
	rpatHP       = 10 // character +0xb11 per slot (0x51b0ec)
	rpatStatus   = 18 // character +0xb01 per slot (0x51b0fb)
	rpatCharm    = 26 // character +0xa35 per slot (0x547010)
	rpatNotDown  = 34 // character +0x17b5 per slot (0x51b23d)
	rpatTeam     = 42 // team byte table 0x687525, stride 12 per slot (0x51b1e4)
	rpatOrder    = 50 // three bytes at 0x68751d (0x51b221)
	rpatKind     = 53 // attacker action record +1 (0x4e77fa)
	rpatSpell    = 54 // attacker action record +5
	rpatListPtr  = 55 // attacker action record +0x19: pointer to an item index, or null
	rpatListItem = 56 // item index the pointer refers to
	rpatHand     = 57 // mainKind, mainAvail, offKind, offAvail, mainItem, offItem
	rpatRange    = 63 // item record +0x47 for items 0..3
	rpatItemSpl  = 67 // item record +0x63 for items 0..3
	rpatSpellRng = 71 // spell record +0x12f for spell ids 1..3
	rpatInputs   = 74
)

func init() {
	names := []string{"attackerSlot", "desiredDisposition"}
	for _, g := range []string{"presence", "hp", "status", "charm", "notDowned", "team"} {
		for i := 0; i < 8; i++ {
			names = append(names, g+strconv.Itoa(i))
		}
	}
	names = append(names, "partyOrder0", "partyOrder1", "partyOrder2",
		"actionKind", "actionSpell", "actionItemListPtr", "actionItemIndex",
		"mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex",
		"item0Range", "item1Range", "item2Range", "item3Range",
		"item0Spell", "item1Spell", "item2Spell", "item3Spell",
		"spell1Range", "spell2Range", "spell3Range")
	Nodes["0051b0a0"] = Node{Addr: "0051b0a0", In: names, Out: []string{"targetSlot"}, F: randomPartyAttackTargetNode}
}

// randomPartyAttackTargetNode lists the party slots the attacker could hit with the
// wanted disposition, then picks one with R(count); -1 when the list is empty.
// Domain notes: slots 0..7; party-order bytes are slots 0..7 or 0xff; spell ids,
// item indices and item spell ids that are actually used must name one of the
// synthetic records (spell ids 1..3, items 0..3), as in the sibling range nodes.
func randomPartyAttackTargetNode(in []int64, r Rand) []int64 {
	if len(in) != rpatInputs {
		panic("random party attack target: expected 74 inputs")
	}
	u := func(i int, max int64) {
		if in[i] < 0 || in[i] > max {
			panic("random party attack target: input outside its storage width")
		}
	}
	u(rpatSlot, 7)
	u(rpatDesired, 255)
	for i := 0; i < 8; i++ {
		u(rpatPresence+i, 255)
		u(rpatHP+i, 4294967295)
		u(rpatStatus+i, 4294967295)
		u(rpatCharm+i, 4294967295)
		u(rpatNotDown+i, 255)
		u(rpatTeam+i, 255)
	}
	for i := 0; i < 3; i++ {
		if in[rpatOrder+i] != 255 { // 0xff ends a party-order entry (0x51b227)
			u(rpatOrder+i, 7)
		}
	}
	u(rpatKind, 4294967295)
	u(rpatSpell, 4294967295)
	u(rpatListPtr, 4294967295)

	slot := int(in[rpatSlot])
	// 0x53c270 for the attacker: present, HP nonzero, status below 18.
	targetableIn := make([]int64, 25)
	targetableIn[0] = int64(slot)
	for i := 0; i < 8; i++ {
		targetableIn[1+3*i] = in[rpatPresence+i]
		targetableIn[2+3*i] = int64(int32(uint32(in[rpatHP+i])))
		targetableIn[3+3*i] = in[rpatStatus+i]
	}
	attackerOK := characterTargetableNode(targetableIn, r)[0] != 0

	handIn := func(hand int64) []int64 {
		return append([]int64{int64(slot), hand}, in[rpatHand:rpatHand+6]...)
	}
	withItems := func(head []int64) []int64 { return append(head, in[rpatRange:rpatRange+4]...) }
	spellRange := func(id int64) int64 {
		if id < 1 || id > 3 {
			panic("random party attack target: spell id outside the synthetic spell records")
		}
		return int64(int32(uint32(in[rpatSpellRng+int(id)-1])))
	}

	var list []int64
	for t := 0; t < 8; t++ {
		// 0x51b0cb..0x51b101: other slot, record present, HP nonzero, status below 18.
		if t == slot || in[rpatPresence+t] == 0 || in[rpatHP+t] == 0 || in[rpatStatus+t] >= 18 {
			continue
		}
		// 0x547010 returns 2 for same charm state, 1 otherwise; only its low byte is compared.
		disp := characterDispositionNode([]int64{in[rpatCharm+slot], in[rpatCharm+t]}, r)[0]
		if disp != in[rpatDesired] {
			continue
		}
		for hand := int64(0); hand < 2; hand++ {
			if handAttackAvailableNode(withItems(handIn(hand)), r)[0] == 0 { // 0x53d2a0
				continue
			}
			// 0x4e77b0 with context 0, then 0x53c270: kinds 10/11 and an unusable attacker both end up >8.
			kind := in[rpatKind]
			if !attackerOK || kind > 8 {
				continue
			}
			rng := int64(0) // 0x51b1ca: -1 skip, nonzero hit, zero falls to the formation test
			switch kind {
			case 2:
				rng = 1
			case 3, 4, 6:
				continue
			case 0, 1:
				rng = charAttackRangeNode(withItems(append([]int64{hand, 1}, in[rpatHand:rpatHand+6]...)), r)[0] // 0x519ac0
			case 7:
				if in[rpatSpell] == 0 {
					continue
				}
				rng = spellRange(in[rpatSpell]) // spell record +0x12f
			case 8:
				rng = -1 // 0x5207e0: null pointer
				if in[rpatListPtr] != 0 {
					idx := in[rpatListItem]
					if idx < 0 || idx > 3 {
						panic("random party attack target: item index outside the synthetic items")
					}
					if id := in[rpatItemSpl+int(idx)]; id != 0 { // item record +0x63
						rng = spellRange(id)
					}
				}
			}
			if rng == -1 {
				continue
			}
			if rng == 0 { // 0x51b1d5, kind 5 jumps here directly
				if !formationAllows(in, slot, t) {
					continue
				}
			}
			list = append(list, int64(t))
			break
		}
	}
	if len(list) == 0 {
		return []int64{-1}
	}
	return []int64{list[r.R(len(list))]}
}

// formationAllows is 0x51b1db..0x51b258: true when the target is added.
func formationAllows(in []int64, slot, t int) bool {
	a, b := in[rpatTeam+slot], int64(int8(in[rpatTeam+t]))
	if a == in[rpatTeam+t] || a == 4 || b == 4 {
		return true
	}
	if d := a - b; d != 2 && d != -2 {
		return true
	}
	count := 0
	for j := 0; j < 3; j++ {
		o := in[rpatOrder+j]
		if o != 255 && in[rpatNotDown+int(o)] == 0 {
			count++
		}
	}
	return count == 0
}
