package calc

import "fmt"

func init() {
	names := []string{"outputPointer", "movementAverage"}
	for slot := 0; slot < 8; slot++ {
		for _, field := range []string{"Active", "MovementExcludedCondition", "MovementBuffStrength", "MovementBuffBonusPercent", "MovementBuffDuration"} {
			names = append(names, fmt.Sprintf("slot%d%s", slot, field))
		}
	}
	Nodes["004f0010"] = Node{Addr: "004f0010", In: names,
		Out: []string{"success", "movementAverage"}, F: partyMovementEffectAverageNode}
}

// partyMovementEffectAverageNode models the optional output pointer and its
// previous value, followed by all eight party records. Pointer values describe
// presence only; the function never dereferences memory. No exe table is used.
func partyMovementEffectAverageNode(in []int64, _ Rand) []int64 {
	// Eight slots, character stride 0x1862 and party-slot stride 0x106:
	// exe 0x4f007d..0x4f008f. Fields are u8/u32/u8/u16/u32 respectively.
	const slots = 8
	if len(in) != 2+slots*5 {
		panic("party movement: expected output pointer, previous average and eight records")
	}
	for i, v := range in {
		limit := int64(4294967295)
		if i >= 2 {
			switch (i - 2) % 5 {
			case 0, 2:
				limit = 255
			case 3:
				limit = 65535
			}
		}
		if v < 0 || v > limit {
			panic("party movement: field outside original unsigned width")
		}
	}
	average := in[1]
	var sum, count uint32
	for slot := 0; slot < slots; slot++ {
		member := in[2+slot*5 : 2+(slot+1)*5]
		if member[0] == 0 || member[1] != 0 {
			continue
		}
		// A missing movement buff fails immediately without writing the output
		// (exe 0x4f0039..0x4f0042 and 0x4f00a9).
		if member[4] == 0 {
			return []int64{0, average}
		}
		// Strength multiplier 10 and byte truncation: exe 0x4f0044..0x4f0056.
		value := uint32(uint8(member[2] * 10))
		// Child 0x517f90 adds floor(unsigned low-32-bit product / 100).
		// Division constant 100: exe 0x517f9e..0x517fa5. These widths differ
		// from SpellDamage's signed wide percentage arithmetic.
		value += (value * uint32(member[3])) / 100
		sum += value
		count++
	}
	// Null output is a successful query even with no eligible members
	// (exe 0x4f0091..0x4f00a4). A nonnull empty query faults in the original.
	if in[0] != 0 {
		if count == 0 {
			panic("party movement: original division by zero")
		}
		average = int64(sum / count)
	}
	return []int64{1, average}
}
