package calc

import "fmt"

func init() {
	inputs := []string{"monsterID", "condition", "decay", "currentHP"}
	outputs := []string{}
	// 0x524019..0x52402c: twenty condition durations, highest index 19.
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("conditionDuration[%d]", i)
		inputs = append(inputs, name)
		outputs = append(outputs, name)
	}
	inputs = append(inputs, "poisonStrength", "primaryCondition")
	outputs = append(outputs, "poisonStrength", "primaryCondition")
	Nodes["00524110"] = Node{Addr: "00524110", In: inputs, Out: outputs, F: monsterConditionDecayNode}
}

// monsterConditionDecayNode returns every condition duration, poison strength,
// and the primary condition. The case supplies a loaded, inactive visual object,
// empty group/effect lists and a loaded template; presentation and derived combat
// caches rebuilt by the removal helper are outside this output interface.
// No exe table contributes to these outputs: priority is the highest active index
// (0x524021), and poison is condition 7 (0x52417f).
func monsterConditionDecayNode(in []int64, r Rand) []int64 {
	if len(in) != 26 {
		panic("monster condition decay: expected 26 fields")
	}
	for _, v := range in {
		if v < 0 || v > 0xffffffff {
			panic("monster condition decay: fields must fit u32")
		}
	}
	if in[1] >= 20 {
		panic("monster condition decay: condition outside 0..19")
	}
	out := append([]int64(nil), in[4:]...)
	if in[3] == 0 { // 0x524132: current HP zero skips decay.
		return out
	}
	condition := int(in[1])
	duration, decay := uint32(out[condition]), uint32(in[2])
	if duration == 0 { // 0x52414c: the original stops at srAssertFail.
		panic("monster condition decay: selected condition must be active")
	}
	if duration <= decay {
		out[condition] = 0
		if condition == 7 { // 0x5240c1: removing poison also clears strength.
			out[20] = 0
		}
		out[21] = 0
		for i := 19; i > 0; i-- {
			if out[i] != 0 {
				out[21] = int64(i)
				break
			}
		}
		return out
	}
	if condition == 7 {
		strength := uint32(out[20])
		// 0x52418c / 0x5241a5: both products retain only their low 32 bits.
		product := strength * decay
		reduction := product / duration
		if reduction == 0 {
			// 0x52419a: R(100) occurs even for zero strength or zero decay.
			roll := uint32(r.R(100))
			// 0x5241a8..0x5241b3: percent multiplication wraps before division.
			if roll < (product*100)/duration {
				reduction = 1
			}
		}
		out[20] = int64(strength - reduction)
	}
	out[condition] = int64(duration - decay)
	return out
}
