package calc

import "fmt"

// partyItemCountNode is 0x521240: how many copies of item id `itemId` the party holds, plus the
// address of the first holder slot and the first holding character (both optional out pointers).
// Layout evidence: build/wiz8.asm:392082-392161 (this function) and 392014-392069 (0x5211a0).
const (
	picCharCount   = 8      // loop 0 .. 0x830 step 0x106, build/wiz8.asm:392137-392140
	picCharStride  = 0x1862 // character record stride, notes/re/formulas.md:9
	picBagBase     = 0xf5d  // 12 bag slots, 12 bytes each: id u32 at +0, count u8 at +4 (asm 392022-392027)
	picBagSlots    = 12
	picEquipBase   = 0x1029 // 8 slots, id u32 at +0; count is the u8 at slot-0xc8 (asm 392045-392050)
	picEquipSlots  = 8
	picSlotStride  = 12
	picPendingAddr = 0x6874cb // pending-pickup record: flag 0x6874ca, id 0x6874cb, count 0x6874cf
	picPartyBase   = 0x685191 // party item table, 12 bytes per entry (asm 392147)
	picPartyMax    = 500      // 0x685191..0x6868f5 inclusive entry starts (asm 384388-384391), count at 0x686901
	picPartyScope  = 2        // searchScope that also scans the party table (asm 392141)

	picHead = 11 // itemId, 2 out pointers, scope, 2 out inits, pending flag/id/count, charArrayBase, partyCount
	picChar = 1 + picBagSlots*2 + picEquipSlots
)

// picSlotCount: count byte of a slot, 0 counts as one item (asm 392027-392033).
func picSlotCount(b int64) uint32 {
	if b&0xff == 0 {
		return 1
	}
	return uint32(b & 0xff)
}

// partyItemCountNode inputs: itemId, firstSlotOutPtr, firstOwnerOutPtr (0 = caller passed null),
// searchScope, firstSlotOutInit, firstOwnerOutInit (what the out locations held before the call),
// pendingFlag, pendingItemId, pendingCount, charArrayBase, partyItemCount; then per character
// active, bagId[12], bagCount[12], equipId[8]; then partyId/partyCnt pairs for 500 entries.
// Outputs: total, final *firstSlotOutPtr, final *firstOwnerOutPtr.
// partyItemCount above 500 would read past the table (not a legal state) and panics.
func partyItemCountNode(in []int64, r Rand) []int64 {
	if len(in) != picHead+picCharCount*picChar+picPartyMax*2 {
		panic("party item count: wrong input count")
	}
	id := uint32(in[0])
	p1, p2 := uint32(in[1]), uint32(in[2])
	scope := uint32(in[3])
	v1, v2 := uint32(in[4]), uint32(in[5])
	n := uint32(in[10])
	if n > picPartyMax {
		panic("party item count: partyItemCount beyond the 500-entry table")
	}
	var total uint32
	if p2 != 0 {
		v2 = 0
	}
	if p1 != 0 {
		v1 = 0
	}
	// pending-pickup record
	if in[6]&0xff != 0 && uint32(in[7]) == id {
		total = uint32(in[8] & 0xff)
		if p1 != 0 && v1 == 0 {
			v1 = picPendingAddr
		}
	}
	base := uint32(in[9])
	chars := in[picHead:]
	for c := 0; c < picCharCount; c++ {
		ch := chars[c*picChar : (c+1)*picChar]
		if ch[0]&0xff == 0 {
			continue
		}
		cbase := base + uint32(c)*picCharStride
		var got uint32
		note := func(slot uint32, cnt uint32) {
			got += cnt
			if p1 != 0 && v1 == 0 {
				v1 = slot
			}
		}
		bagIDs, bagCnt, eqIDs := ch[1:13], ch[13:25], ch[25:33]
		for k := 0; k < picBagSlots; k++ {
			if uint32(bagIDs[k]) == id {
				note(cbase+picBagBase+uint32(k)*picSlotStride, picSlotCount(bagCnt[k]))
			}
		}
		if scope != 0 {
			for k := 0; k < picEquipSlots; k++ {
				if uint32(eqIDs[k]) == id {
					// count byte lives at slot-0xc8 = bag slot k's count byte
					note(cbase+picEquipBase+uint32(k)*picSlotStride, picSlotCount(bagCnt[k]))
				}
			}
		}
		if got != 0 {
			total += got
			if p2 != 0 && v2 == 0 {
				v2 = cbase
			}
		}
	}
	if scope == picPartyScope {
		party := in[picHead+picCharCount*picChar:]
		for k := uint32(0); k < n; k++ {
			if uint32(party[2*k]) == id {
				total += picSlotCount(party[2*k+1])
				if p1 != 0 && v1 == 0 {
					v1 = picPartyBase + k*picSlotStride
				}
			}
		}
	}
	return []int64{int64(total), int64(v1), int64(v2)}
}

func init() {
	in := []string{"itemId", "firstSlotOutPtr", "firstOwnerOutPtr", "searchScope", "firstSlotOutInit", "firstOwnerOutInit",
		"pendingFlag", "pendingItemId", "pendingCount", "charArrayBase", "partyItemCount"}
	for c := 0; c < picCharCount; c++ {
		in = append(in, fmt.Sprintf("active[%d]", c))
		for k := 0; k < picBagSlots; k++ {
			in = append(in, fmt.Sprintf("bagId[%d][%d]", c, k))
		}
		for k := 0; k < picBagSlots; k++ {
			in = append(in, fmt.Sprintf("bagCount[%d][%d]", c, k))
		}
		for k := 0; k < picEquipSlots; k++ {
			in = append(in, fmt.Sprintf("equipId[%d][%d]", c, k))
		}
	}
	for k := 0; k < picPartyMax; k++ {
		in = append(in, fmt.Sprintf("partyId[%d]", k), fmt.Sprintf("partyCnt[%d]", k))
	}
	Nodes["00521240"] = Node{Addr: "00521240", In: in, Out: []string{"total", "firstSlotOut", "firstOwnerOut"}, F: partyItemCountNode}
}
