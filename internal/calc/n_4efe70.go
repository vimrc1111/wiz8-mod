package calc

import "fmt"

func init() {
	names := []string{"partyActionMode"}
	for slot := 0; slot < 8; slot++ {
		names = append(names, fmt.Sprintf("slot%dActive", slot), fmt.Sprintf("slot%dCurrentHP", slot), fmt.Sprintf("slot%dInitiative", slot))
	}
	Nodes["004efe70"] = Node{Addr: "004efe70", In: names,
		Out: []string{"partyActionPhase"}, F: partyActionPhaseNode}
}

// partyActionPhaseNode: in = mode, then (active, currentHP, initiative) for 8 slots.
func partyActionPhaseNode(in []int64, r Rand) []int64 {
	if len(in) != 25 {
		panic("party action phase: expected mode and eight slot triples")
	}
	// Slowest value starts at 90 (exe 0x4efe7c); each live member adds R(4) (0x4efe9c) to initiative, wrapping in 32 bits.
	slowest := int32(90)
	for s := 0; s < 8; s++ {
		active, hp, init := in[1+3*s], in[2+3*s], in[3+3*s]
		if active == 0 || hp == 0 {
			continue
		}
		if v := int32(r.R(4)) + int32(init); v < slowest {
			slowest = v
		}
	}
	// Signed clamp to [-10, 89]: call 0x5179d0 with bounds 0xfffffff6 and 0x59.
	if slowest > 89 {
		slowest = 89
	} else if slowest < -10 {
		slowest = -10
	}
	phase := uint32(90 - slowest)
	// Modes 1 and 2 round to nearest 10 (half up) then clamp unsigned to [10, 100]: exe 0x4eff22..0x4eff3a, call 0x5179f0.
	if m := in[0]; m == 1 || m == 2 {
		phase = phase + 5 - (phase+5)%10
		if phase > 100 {
			phase = 100
		} else if phase < 10 {
			phase = 10
		}
	}
	return []int64{int64(phase)}
}
