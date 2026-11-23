package calc

// Inputs reuse the posture fixture, excluding the two fixed call parameters,
// followed by C+0xa31 and the selected combat record's +0xa8 attempt counter.
// The child retains its exe-backed tables and eight independent party records.
func playerTurnInterceptNode(in []int64, r Rand) []int64 {
	if len(in) != 279 || in[0] < 0 || in[0] > 7 {
		panic("player turn intercept: expected 279 inputs and party slot 0..7")
	}
	for _, v := range in[277:] {
		if v < 0 || v > 4294967295 {
			panic("player turn intercept: expected u32 status and attempts")
		}
	}
	child := make([]int64, 279)
	copy(child, in[:261])
	copy(child[263:], in[261:277])
	child[261], child[262] = 4, 1 // Defense, apply enabled: exe 0x53d5f7..0x53d5fc.
	out := []int64{0, child[9], child[11], child[274]}
	out = append(out, child[275:279]...)
	out = append(out, child[266:270]...)
	out = append(out, child[263], child[264], child[265], in[278])
	slot := int(in[0])
	// Parent gates are nonzero HP/stamina, unsigned status <14 and no
	// interference, independently of child eligibility (exe 0x53d5c2..0x53d5f5).
	if child[28+3*slot] == 0 || child[56] == 0 || uint32(child[29+3*slot]) >= 14 || in[277] != 0 {
		return out
	}
	posture := characterPostureEligibilityNode(child, r)
	if posture[0] == 0 {
		child[261] = 5 // Protect fallback: exe 0x53d608..0x53d617.
		posture = characterPostureEligibilityNode(child, r)
	}
	copy(out, posture)
	if posture[0] == 0 {
		return out
	}
	attempts := uint32(in[278])
	out[15] = int64(attempts + 1) // u32 counter, exe 0x53d625/0x53d65a.
	if attempts == 0 {
		out[0] = 1 // First eligible attempt needs no RNG, exe 0x53d623..0x53d632.
		return out
	}
	// Both the subtraction and comparison are unsigned 32-bit, including
	// underflow: no clamp at zero (exe 0x53d640..0x53d65b).
	threshold := uint32(child[210]) - 25*attempts
	out[0] = 0
	if uint32(r.R(100)) < threshold { // Bound 100: exe 0x53d646.
		out[0] = 1
	}
	return out
}

func init() {
	child := Nodes["004ed2d0"]
	names := append([]string{}, child.In[:261]...)
	names = append(names, child.In[263:]...)
	names = append(names, "interferenceStatus", "interceptAttempts")
	outputs := append([]string{}, child.Out...)
	outputs[0] = "intercepted"
	outputs = append(outputs, "interceptAttempts")
	Nodes["0053d590"] = Node{Addr: "0053d590", In: names, Out: outputs, F: playerTurnInterceptNode}
}
