package calc

import "fmt"

func init() {
	inputs := []string{"currentHP"}
	for i := 0; i < 20; i++ { // Twenty durations at 0x524019..0x52403b.
		inputs = append(inputs, fmt.Sprintf("conditionDuration[%d]", i))
	}
	inputs = append(inputs, "primaryCondition", "poisonStrength", "senses", "reactionState", "responseSuppressed", "resisted", "sourceKind", "characterIndex", "sourceMonsterID", "sourceDecaySuppressionA", "sourceDecaySuppressionB", "sourceDecaySuppressionC", "targetDisposition", "characterCharm", "sourceDisposition", "sourceTemplateID", "targetTemplateID", "sourceFaction", "targetFaction", "sourceCharmDuration")
	outputs := append([]string{}, inputs[1:23]...)
	outputs = append(outputs, "reactionState", "responseSuppressed")
	Nodes["0052beb0"] = Node{Addr: "0052beb0", In: inputs, Out: outputs, F: monsterHitConditionResponseNode}
}

// monsterHitConditionResponseNode projects all condition durations, the primary
// condition, poison strength and reaction flags. The case supplies loaded visual
// and template records, an empty group/effect list, and inactive presentation.
// The visual controller's +0x74 is nonzero, so responseSuppressed is preserved
// by 0x4e6062. Derived combat caches and presentation are outside this interface,
// as in monsterConditionDecayNode. No exe data table contributes to this projection;
// diagnostic strings and visual dispatch tables remain in the emulator's PE map.
func monsterHitConditionResponseNode(in []int64, r Rand) []int64 {
	if len(in) != 41 {
		panic("monster hit response: expected 41 fields")
	}
	for i, v := range in {
		if i == 34 { // Selected character's signed charm field, C+0xa35.
			if v < -2147483648 || v > 2147483647 {
				panic("monster hit response: charm must fit i32")
			}
		} else if v < 0 || v > 0xffffffff {
			panic("monster hit response: expected u32 fields")
		}
	}
	for _, i := range []int{23, 25, 26, 30, 31, 32, 33, 35} {
		if in[i] > 255 {
			panic("monster hit response: byte overflow")
		}
	}
	if in[28] > 7 { // Party record count at 0x52b57d; stride at 0x546f1e.
		panic("monster hit response: character index outside 0..7")
	}
	out := append([]int64{}, in[1:23]...)
	remove := func(condition int) {
		out[condition] = 0
		out[20] = 0
		// 0x524021: highest active condition, including index zero fallback.
		for i := 19; i > 0; i-- {
			if out[i] != 0 {
				out[20] = int64(i)
				break
			}
		}
	}
	// 0x52bed6..0x52beef: R(100) < 50 + floor(Senses/2).
	// 0x523f66 skips removal for a dead target, but the roll still occurs.
	if out[15] != 0 && in[26] == 0 {
		if r.R(100) < 50+int(in[23])/2 && in[0] != 0 {
			remove(15)
		}
	}
	reaction := in[24]
	if reaction == 1 { // 0x52befe / 0x4e6085: cancel pending response.
		reaction = 0
	}
	// 0x53bea0 / 0x53bf10 with argument zero accept source kinds 1 / 2.
	eligible := (in[27] == 1 || in[27] == 2) && in[30] == 0 && in[31] == 0 && in[32] == 0 && in[26] == 0 && out[13] != 0
	if eligible {
		friendly := false
		if in[27] == 1 {
			friendly = targetDispositionNode([]int64{in[28], in[34], in[33]}, r)[0] == 2
		} else {
			// 0x546f8d..0x546fa8: template 548 is exempt; equal disposition
			// returns friendly (2). Later faction/charm checks only return 0/1,
			// so their supplied fields cannot enable the decay at 0x52bfb1.
			friendly = in[36] != 548 && in[37] != 548 && in[35] == in[33]
		}
		if friendly {
			// 0x52bf83 / 0x52bfb9: condition 13 decays by exactly one.
			decayInput := []int64{123, 13, 1, in[0]} // ID 123 is fixture wiring.
			decayInput = append(decayInput, out[:20]...)
			decayInput = append(decayInput, out[21], out[20])
			decayed := monsterConditionDecayNode(decayInput, r)
			copy(out[:20], decayed[:20])
			out[21], out[20] = decayed[20], decayed[21]
		}
	}
	return append(out, reaction, in[25])
}
