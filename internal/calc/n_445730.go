package calc

func init() {
	Nodes["00445730"] = Node{
		Addr: "00445730",
		In:   []string{"lockType", "difficulty", "code0", "code1", "code2", "code3", "code4", "code5", "code6", "code7", "attemptsRemaining", "lockAttemptIndex", "unlocked"},
		Out:  []string{"code0", "code1", "code2", "code3", "code4", "code5", "code6", "code7", "attemptsRemaining", "lockAttemptIndex", "unlocked"},
		F:    lockCodeInitNode,
	}
}

// lockCodeInitNode initializes the lock's code and attempt state. Initial output
// fields are inputs because types other than 1 retain their code and budget.
func lockCodeInitNode(in []int64, r Rand) []int64 {
	if len(in) != 13 || in[0] < 0 || in[0] > 4294967295 {
		panic("lockCodeInit: expected u32 lock type and 12 state fields")
	}
	for _, i := range []int{1, 10, 11} {
		if in[i] < -2147483648 || in[i] > 2147483647 {
			panic("lockCodeInit: difficulty, attempts and index must be i32")
		}
	}
	for _, i := range []int{2, 3, 4, 5, 6, 7, 8, 9, 12} {
		if in[i] < 0 || in[i] > 255 {
			panic("lockCodeInit: codes and unlocked flag must be u8")
		}
	}
	out := append([]int64(nil), in[2:]...)
	// Type 1, eight code bytes and R(4): exe 0x445733..0x44574d.
	if in[0] == 1 {
		for i := 0; i < 8; i++ {
			out[i] = int64(r.R(4))
		}
		// Difficulty limits 2 and 8, multiplier 3: exe 0x44574f..0x445782.
		out[8] = min(int64(8), max(int64(2), in[1])) * 3
	}
	// Unselected index -1 and locked flag 0: exe 0x445785..0x44578c;
	// the difficulty-below-2 path repeats these resets at 0x445768..0x44576f.
	out[9], out[10] = -1, 0
	return out
}
