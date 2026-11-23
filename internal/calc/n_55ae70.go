package calc

func init() {
	Nodes["0055ae70"] = Node{
		Addr: "0055ae70",
		In: []string{
			"shopStockListPointer", "shopStockCount", "customerStockListPointer", "customerStockCount",
			"stockEntry0Gate", "stockEntry0ItemIndex", "stockEntry0StackQuantity", "stockEntry0StockUnits",
			"stockEntry1Gate", "stockEntry1ItemIndex", "stockEntry1StackQuantity", "stockEntry1StockUnits",
			"stockEntry2Gate", "stockEntry2ItemIndex", "stockEntry2StackQuantity", "stockEntry2StockUnits",
			"customerEntry0ItemIndex", "customerEntry0TradeFlag",
			"customerEntry1ItemIndex", "customerEntry1TradeFlag",
			"customerEntry2ItemIndex", "customerEntry2TradeFlag",
			"itemDef0Flags", "itemDef1Flags", "itemDef2Flags", "itemDef3Flags", "itemDef4Flags", "itemDef5Flags",
			"itemDef0StackType", "itemDef1StackType", "itemDef2StackType", "itemDef3StackType", "itemDef4StackType", "itemDef5StackType",
		},
		Out: []string{
			"stockEntry0StackQuantity", "stockEntry0StockUnits",
			"stockEntry1StackQuantity", "stockEntry1StockUnits",
			"stockEntry2StackQuantity", "stockEntry2StockUnits",
		},
		F: shopStockTurnoverNode,
	}
}

// shopStockTurnoverNode: in = shop stock list pointer and count, the customer's
// list pointer and count (counts >= 0, at most 3 entries each), three stock entries
// (gate u32 at +0, itemIndex u32 at +4, stackQuantity u8 at +0x8, stockUnits u8 at
// +0x10), three customer entries (itemIndex u32 at +0, tradeFlag u8 at +5), and the
// flags (+0x41) and stackType (+0x66) of item definitions 0..5; entry item indices
// are in 0..5. Out = each stock entry's stackQuantity and stockUnits after turnover.
// Each entry that still has units, has gate 0, an item without flag bits 0x12 and no
// customer entry for the same item with tradeFlag 1 rolls percent dice once per unit
// (exe 0x55af49; the roll is always below 100, so every unit goes) and then removes
// that many through the shop-stock-remove node (exe 0x55af74).
func shopStockTurnoverNode(in []int64, r Rand) []int64 {
	stockN, custN := in[1], in[3]
	if in[0] == 0 { // exe 0x5e2c70: null list counts as empty
		stockN = 0
	}
	if in[2] == 0 {
		custN = 0
	}
	out := make([]int64, 0, 6)
	for k := int64(0); k < 3; k++ {
		e := in[4+4*k : 8+4*k]
		gate, item, qty, units := e[0], e[1], e[2], e[3]
		if k < stockN && units != 0 && gate == 0 && in[22+item]&0x12 == 0 && !shopStockTurnoverCustomerHolds(in, custN, item) {
			for i := int64(0); i < units; i++ {
				r.R(100)
			}
			rm := shopStockRemoveNode([]int64{in[0], k, stockN, item, in[28+item], qty, units, units}, r)
			qty, units = rm[1], rm[2]
		}
		out = append(out, qty, units)
	}
	return out
}

// shopStockTurnoverCustomerHolds: a customer entry for the item with tradeFlag 1 blocks it.
func shopStockTurnoverCustomerHolds(in []int64, custN, item int64) bool {
	for j := int64(0); j < custN && j < 3; j++ {
		if in[16+2*j] == item && in[17+2*j] == 1 {
			return true
		}
	}
	return false
}
