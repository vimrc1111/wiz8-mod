package calc

func init() {
	Nodes["0055ae00"] = Node{
		Addr: "0055ae00",
		In:   []string{"stockListPointer", "stockIndex", "stockCount", "itemIndex", "stackType", "stackQuantity", "stockUnits", "removeQuantity"},
		Out:  []string{"removed", "stackQuantity", "stockUnits"},
		F:    shopStockRemoveNode,
	}
}

// shopStockRemoveNode: in = stockListPointer, stockIndex (i32), stockCount (i32),
// itemIndex, stackType (u8, item record +0x66), stackQuantity (u8, entry +0x8),
// stockUnits (u8, entry +0x10), removeQuantity (u8). Out = removed (u8), the new
// stackQuantity and stockUnits. Entry lookup (exe 0x5e2870) is null for a null
// list or stockIndex >= stockCount; the original compares signed, indices are >= 0.
func shopStockRemoveNode(in []int64, r Rand) []int64 {
	list, idx, count := in[0], in[1], in[2]
	stackType, qty, units, rm := in[4], in[5], in[6], in[7]
	// exe 0x55ae05: removing zero, or a missing entry, reports failure and changes nothing.
	if rm == 0 || list == 0 || idx >= count {
		return []int64{0, qty, units}
	}
	if stackType == 1 { // exe 0x55ae40: stacked item, quantity at entry +0x8
		if qty < rm {
			qty = rm
		}
		qty -= rm
		if qty == 0 { // exe 0x55ae54: emptied stack clears the units field
			units = 0
		}
	} else { // exe 0x55ae5c: otherwise units at entry +0x10, clamped at zero
		if units < rm {
			rm = units
		}
		units -= rm
	}
	return []int64{1, qty, units}
}
