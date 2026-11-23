package calc

import "fmt"

// shop_sell_transaction 0x55b730 (build/wiz8.asm:472633): sell one item record to a merchant.
// Item definition (base 0x68516c, 0x10d bytes each, notes/re/loot.md:9) byte +0x41 bit 1 rejects the
// sale; otherwise 0x55b290 must accept the item category. Then a priced temporary item instance is
// built (0x51c020), the price comes from 0x55b5e0, and unless the no-payment argument's low byte is
// nonzero the price is added to the party gold 0x685189 (u32 wrap) by 0x51bea0 (message flag 0), which
// also adds it to the dword at 0x686b7d+0x21*statIdx when the signed 0x686a70 is below 0x2f. The
// sound 0x408860 is presentation. A zero record byte +4 becomes 1, 0x55a930 merges the stock, result 1.
//
// Callees: 0x55b290, 0x51c020, 0x55b5e0, 0x55a930 are D4 formulas and address stubs in the case: each
// return is a case input (acceptRet, priceRet; the others' returns are unused), the recorded arguments
// and call counts are outputs. 0x51c020/0x55b5e0 write and read the stack instance, so the instance
// byte stored from the count argument is not observable. 0x51bea0 and 0x408860 run natively; 0x408860
// returns at its first test with the sound flag 0x650e50 at its .data default.
//
// Fixture caps (calc panics past them): item ids 0..3 (the case models four item records), statIdx
// -1 or above (the game only stores -1 or non-negative values, asm 42a7b5/42a889/42ae2a; other negatives
// write outside the modelled slot window). Slots modelled: -1..46.
const (
	ssMer   = 0x20001000
	ssRec   = 0x20002000
	ssItems = 4
	ssLo    = -1
	ssHi    = 46
)

var ssIn = func() []string {
	n := []string{"idx", "itemFlags", "acceptRet", "priceRet", "sellFlag", "qualByte", "count", "noPay", "gold", "statIdx"}
	for s := ssLo; s <= ssHi; s++ {
		n = append(n, fmt.Sprintf("slot%d", s))
	}
	return n
}()

var ssOut = func() []string {
	n := []string{"al", "sellFlag", "gold"}
	for s := ssLo; s <= ssHi; s++ {
		n = append(n, fmt.Sprintf("slot%d", s))
	}
	return append(n, "acc.mer", "acc.rec", "acc.n", "init.id", "init.zero1", "init.qual", "init.zero2", "init.n",
		"price.mer", "price.zero", "price.n", "merge.mer", "merge.rec", "merge.count", "merge.n")
}()

func init() {
	Nodes["0055b730"] = Node{Addr: "0055b730", In: ssIn, Out: ssOut, F: shopSellTransactionNode}
}

func shopSellTransactionNode(in []int64, r Rand) []int64 {
	if len(in) != len(ssIn) {
		panic("shop_sell_transaction: wrong input count")
	}
	idx, flags, accept, price := in[0], in[1], in[2], in[3]
	sell, qual, count, noPay, gold, stat := in[4], in[5], in[6], in[7], in[8], in[9]
	if idx < 0 || idx >= ssItems {
		panic("shop_sell_transaction: item id past fixture cap")
	}
	if stat < ssLo || stat > 0x7fffffff {
		panic("shop_sell_transaction: stat index outside modelled slot window")
	}
	for i, v := range in {
		if i != 9 && (v < 0 || v > 0xffffffff) {
			panic("shop_sell_transaction: field width overflow")
		}
	}
	slots := append([]int64(nil), in[10:]...)
	var al int64
	rec := make([]int64, 15) // acc(3), init(5), price(3), merge(4)
	if flags&2 == 0 {        // 0x55b74d
		rec[0], rec[1], rec[2] = ssMer, ssRec, 1 // 0x55b290(merchant, record)
		if accept&0xff != 0 {                    // 0x55b763
			rec[3], rec[4], rec[5], rec[6], rec[7] = idx, 0, qual&0xff, 0, 1 // 0x51c020
			rec[8], rec[9], rec[10] = ssMer, 0, 1                            // 0x55b5e0
			if noPay&0xff == 0 {                                             // 0x55b7a3: 0x51bea0(price, 0)
				gold = (gold + price) & 0xffffffff
				if stat < 0x2f {
					slots[stat-ssLo] = (slots[stat-ssLo] + price) & 0xffffffff
				}
			}
			if sell&0xff == 0 { // 0x55b7c4
				sell = 1
			}
			rec[11], rec[12], rec[13], rec[14] = ssMer, ssRec, count, 1 // 0x55a930
			al = 1
		}
	}
	return append(append([]int64{al, sell, gold}, slots...), rec...)
}
