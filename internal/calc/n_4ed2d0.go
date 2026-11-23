package calc

// characterPostureEligibilityNode shares the eight-character, item and spell
// fixture of pcActionEligibilityNode. Its child context is always 1, with both
// flags zero (exe 0x4ed35d). The case exposes the setter's data and UI effects;
// scene targets and populated monster lists remain outside the child fixture.
func characterPostureEligibilityNode(in []int64, r Rand) []int64 {
	if len(in) != 279 {
		panic("character posture eligibility: expected 279 inputs")
	}
	child := make([]int64, 264)
	j := 0
	for i := range child {
		switch i {
		case 1:
			child[i] = 1 // Battle action source, exe 0x4ed361.
		case 24, 25:
			child[i] = 0 // Check target; do not prepare weapons, exe 0x4ed35d.
		default:
			child[i] = in[j]
			j++
		}
	}
	for i, value := range child {
		width := pcActionInputWidths[i]
		bits := uint(absPCActionWidth(width) * 8)
		lo, hi := int64(0), int64((uint64(1)<<bits)-1)
		if width < 0 {
			lo = -(int64(1) << (bits - 1))
			hi = (int64(1) << (bits - 1)) - 1
		}
		if value < lo || value > hi {
			panic("character posture eligibility: child storage width exceeded")
		}
	}
	if child[0] < 0 || child[0] > 7 || in[261] < -2147483648 || in[261] > 2147483647 || in[274] < -2147483648 || in[274] > 2147483647 {
		panic("character posture eligibility: invalid selector or signed word")
	}
	for i := 262; i < 274; i++ {
		hi := int64(4294967295)
		if i < 265 {
			hi = 255
		}
		if in[i] < 0 || in[i] > hi {
			panic("character posture eligibility: storage width exceeded")
		}
	}
	slot, posture := int(child[0]), in[261]
	// The u32 life/status gates are independent of the child's stricter status
	// threshold (exe 0x4ed2ee, 0x4ed2ff versus 0x4e7a0f).
	eligible := child[31+3*slot] != 0 && child[32+3*slot] < 15
	completed := in[263] != 0
	if eligible {
		if completed {
			eligible = child[10] == posture
		} else if child[11] != posture {
			eligible = posture == 4 && pcActionEligibilityNode(child, r)[0] == 0
		}
	}
	for _, value := range in[275:279] {
		if value < 0 || value > 4294967295 {
			panic("character posture eligibility: target storage width exceeded")
		}
	}
	out := []int64{0, child[10], child[12], in[274]}
	out = append(out, in[275:279]...)
	out = append(out, in[266:270]...)
	out = append(out, in[263], in[264], in[265])
	if !eligible {
		return out
	}
	out[0] = 1
	if completed || in[262] == 0 {
		return out
	}
	if posture != 4 && posture != 5 {
		// The original setter stops at srAssertFail for these requests
		// (exe 0x4ed39a..0x4ed3b8); it has no successful replay output.
		panic("character posture eligibility: applying posture requires defense 4 or protect 5")
	}
	// Setter constants and field effects: exe 0x4ed3d1..0x4ed456.
	out[1], out[2], out[3] = posture, 4294967295, -1
	if posture == 5 {
		for i, value := range child[20:24] {
			out[4+i] = int64(uint32(value))
		}
		copy(out[8:12], in[270:274]) // Protect copies all eight target words.
	}
	out[12] = 1
	if posture == 4 && child[11] != posture {
		out[13] = 1
	}
	// UI refresh helper 0x562a50: mode 7 and a live battle object only.
	if child[2] == 7 && child[3] != 0 {
		out[14] = int64(uint32(in[265]) | uint32(1<<uint(slot)) | 0x100000)
	}
	return out
}

func init() {
	// Copy the child's public names, omitting the three fixed call parameters.
	names := make([]string, 0, 279)
	for i, name := range Nodes["004e79a0"].In {
		if i != 1 && i != 24 && i != 25 {
			names = append(names, name)
		}
	}
	names = append(names, "posture", "applyPosture", "actionCompleted", "deferredDefense", "refreshMask",
		"mainTargetTail0", "mainTargetTail1", "mainTargetTail2", "mainTargetTail3",
		"offTargetTail0", "offTargetTail1", "offTargetTail2", "offTargetTail3", "mainParameter",
		"mainTargetHead0", "mainTargetHead1", "mainTargetHead2", "mainTargetHead3")
	Nodes["004ed2d0"] = Node{Addr: "004ed2d0", In: names, Out: []string{
		"eligible", "mainAction", "mainSpell", "mainParameter",
		"mainTargetWord0", "mainTargetWord1", "mainTargetWord2", "mainTargetWord3",
		"mainTargetWord4", "mainTargetWord5", "mainTargetWord6", "mainTargetWord7",
		"actionCompleted", "deferredDefense", "refreshMask",
	}, F: characterPostureEligibilityNode}
}
