package calc

import "fmt"

func init() {
	inputs := []string{"effectIndex", "newStrength", "newDuration", "newPercent"}
	var outputs []string
	// 0x5242d7..0x524313: eight 12-byte effect records at monster+0xa7.
	for i := 0; i < 8; i++ {
		inputs = append(inputs, fmt.Sprintf("oldStrength%d", i), fmt.Sprintf("oldPercent%d", i), fmt.Sprintf("oldDuration%d", i))
		outputs = append(outputs, fmt.Sprintf("strength%d", i), fmt.Sprintf("percent%d", i), fmt.Sprintf("duration%d", i))
	}
	inputs = append(inputs, "oldAbsorptionPool", "Z", "N", "S")
	outputs = append(outputs, "absorptionPool")
	Nodes["005242b0"] = Node{Addr: "005242b0", In: inputs, Out: outputs, F: monsterEffectApplyNode}
}

// This interface exposes all eight effect records and the absorption pool.
// Presentation objects and the separately rebuilt modifier caches are outside
// these outputs; neither feeds back into the exposed effect records.
func monsterEffectApplyNode(in []int64, r Rand) []int64 {
	if len(in) != 32 || in[0] < 0 || in[0] > 7 {
		panic("monster effect apply: expected eight effect records and index 0..7")
	}
	for i, v := range in[1:29] {
		if i == 2 {
			if v < -2147483648 || v > 2147483647 {
				panic("monster effect apply: percentage must fit i32")
			}
			continue
		}
		limit := int64(0xffffffff)
		if i == 27 || (i >= 3 && (i-3)%3 == 1) {
			limit = 65535
		}
		if v < 0 || v > limit {
			panic("monster effect apply: field overflow")
		}
	}
	out := append([]int64(nil), in[4:29]...)
	k := int(in[0]) * 3
	oldDuration := out[k+2]
	// 0x5242e8: unsigned comparison accepts ties and rejects weaker effects.
	if uint32(in[1]) < uint32(out[k]) {
		return out
	}
	out[k], out[k+1], out[k+2] = in[1], int64(uint16(in[3])), in[2]
	// 0x52431b..0x52434c: effect 2 uses the rules+0x24fc dice supplied as
	// Z/N/S inputs. The percentage product wraps to u32 before unsigned
	// division by 100; both the base and final pool are truncated to u16.
	if in[0] == 2 {
		base := uint32(diceNode(in[29:32], r)[0]) * uint32(in[1])
		percent := uint32(uint16(base)) * uint32(in[3])
		out[24] = int64(uint16(base + percent/100))
	}
	// 0x49f3ba: creating a newly visible effect consumes R(20) for the
	// presentation phase even though it does not change the rule outputs.
	if oldDuration == 0 {
		r.R(20)
	}
	return out
}
