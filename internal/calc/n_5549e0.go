package calc

func init() {
	Nodes["005549e0"] = Node{
		Addr: "005549e0",
		In:   []string{"currentHP", "primaryStatus"},
		Out:  []string{"formationMovable"},
		F:    formationMovableNode,
	}
}

// formationMovableNode consumes the selected character's current HP (+0xb11)
// and primary status (+0xb01). The character index only selects the record;
// it does not otherwise affect the predicate (0x5549e0..0x5549fc).
func formationMovableNode(in []int64, r Rand) []int64 {
	// Both fields are unsigned 32-bit values at 0x5549fc and 0x554a06.
	const maxField = 4294967295
	if len(in) != 2 || in[0] < 0 || in[0] > maxField || in[1] < 0 || in[1] > maxField {
		panic("formation movable: expected u32 current HP and primary status")
	}
	// The status cutoff is 13 at 0x554a06; HP zero fails at 0x554a02.
	const statusCutoff = 13
	if in[0] != 0 && in[1] < statusCutoff {
		return []int64{1}
	}
	return []int64{0}
}
