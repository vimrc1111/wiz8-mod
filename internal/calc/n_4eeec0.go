package calc

func init() {
	// Eight party slots, including recruit slots: exe 0x4eeed4.
	names := make([]string, 0, 24)
	for _, slot := range []string{"0", "1", "2", "3", "4", "5", "6", "7"} {
		names = append(names, "slotActive"+slot, "currentHP"+slot, "mainStatus"+slot)
	}
	Nodes["004eeec0"] = Node{Addr: "004eeec0", In: names, Out: []string{"partyHasSurvivor"}, F: partySurvivalCheckNode}
}

// partySurvivalCheckNode reads triples of slot activity, HP bits and main status.
// No tables or RNG calls are used; the original returns a full 32-bit boolean.
func partySurvivalCheckNode(in []int64, r Rand) []int64 {
	// Eight slots (0x4eeed4), each represented by the three fields named in iface.
	if len(in) != 24 {
		panic("partySurvivalCheck: expected eight slotActive/currentHP/mainStatus triples")
	}
	for i := 0; i < len(in); i += 3 {
		// u8 activity at 0x4eeed9, 32-bit HP at 0x4eeede and status at 0x4eeee5.
		if in[i] < 0 || in[i] > 255 || in[i+1] < 0 || in[i+1] > 4294967295 || in[i+2] < 0 || in[i+2] > 4294967295 {
			panic("partySurvivalCheck: expected u8 slotActive and u32 currentHP/mainStatus")
		}
	}
	for i := 0; i < len(in); i += 3 {
		// exe 0x4eeed9..0x4eeeea: active, HP nonzero, unsigned main status < 18.
		if in[i] != 0 && in[i+1] != 0 && in[i+2] < 18 {
			return []int64{1}
		}
	}
	return []int64{0}
}
