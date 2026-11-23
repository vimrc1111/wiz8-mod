package calc

func init() {
	Nodes["0051b840"] = Node{
		Addr: "0051b840",
		In:   []string{"itemDefinitionIndex", "quantity", "itemType", "stackable", "baseValue"},
		Out:  []string{"itemValue"},
		F:    itemValueNode,
	}
}

// itemValueNode takes the instance quantity and the selected definition's
// fields as inputs. The loaded definition base at 0x68516c and 269-byte stride
// (0x51b846..0x51b858) only locate that record; no exe data table is needed.
func itemValueNode(in []int64, r Rand) []int64 {
	if len(in) != 5 || in[0] < 0 || in[0] > 4294967295 ||
		in[1] < 0 || in[1] > 255 || in[2] < 0 || in[2] > 255 ||
		in[3] < 0 || in[3] > 255 || in[4] < 0 || in[4] > 4294967295 {
		panic("item value: expected u32 definition index/base value and u8 quantity/type/stackable")
	}
	quantity, value := uint32(in[1]), uint32(in[4])
	// Type 4 takes precedence over stacking (0x51b85a). Its value is
	// scaled by 25 with a rounding bias of 24 (0x51b86c..0x51b878).
	// Both the product and the bias wrap at 32 bits before unsigned division.
	if in[2] == 4 {
		return []int64{int64((quantity*value + 24) / 25)}
	}
	// Only flag 1 and quantity > 1 multiply the base value; zero quantity
	// still returns the base value (0x51b87c..0x51b89c).
	if in[3] == 1 && quantity > 1 {
		value *= quantity
	}
	return []int64{int64(value)}
}
