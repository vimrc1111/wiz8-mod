package calc

// treasureSeedInitTick is the fixed GetTickCount environment in
// re/emu/cases/0043ba10.json. The current harness cannot vary tick per input.
const treasureSeedInitTick uint32 = 4294960000

func init() {
	Nodes["0043ba10"] = Node{
		Addr: "0043ba10",
		In:   []string{"nextTriggerID"},
		Out:  []string{"treasureSeed", "triggerID", "nextTriggerID", "treasureGenerated", "triggerFlags"},
		F:    treasureSeedInitNode,
	}
}

// treasureSeedInitNode models Trigger construction in the case's fixed clock
// environment. All preexisting Trigger fields used here are overwritten before
// being read. SR class registration has no effect on these rule outputs.
func treasureSeedInitNode(in []int64, r Rand) []int64 {
	if len(in) != 1 || in[0] < 0 || in[0] > 4294967295 {
		panic("treasure_seed_init: expected u32 nextTriggerID")
	}
	// 0x43bbaa supplies 30000 to R; 0x43bbc9 adds the clock modulo 2^32.
	seed := uint32(r.R(30000)) + treasureSeedInitTick
	// 0x43bbdd..0x43bbef assigns the ID, then increments the global modulo 2^32.
	id := uint32(in[0])
	// 0x43bb2a clears treasureGenerated; 0x43bb8e sets triggerFlags bit 0x10
	// after 0x43ba88 clears the flags. Seed use: notes/re/loot.md:35.
	return []int64{int64(seed), int64(id), int64(id + 1), 0, 16}
}
