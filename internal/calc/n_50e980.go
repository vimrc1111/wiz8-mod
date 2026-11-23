package calc

import "fmt"

const (
	equipmentSlots       = 12 // 0x50ea70; C+0xf5d, stride 12 at 0x50ea6d
	equipmentRecords     = 12 // synthetic input record count, not a game limit
	equipmentFields      = 15 // I+48,49,6c..74,b1..b4 (0x50e9e1..0x50ea61)
	equipmentBufferSize  = 66 // output through +0x41 at 0x50ea9c
	equipmentBufferInput = equipmentSlots + equipmentRecords*equipmentFields
)

// equipmentEffectSumNode consumes slot IDs, synthetic item records and the
// caller's byte buffer. Item data is mutable and supplied by the caller (global
// 0x68516c, record stride 0x10d at 0x50e9c2..0x50e9d5); no exe table is read.
// Each item is initiative, hit, HP/stamina/SP recovery, skill ID/delta,
// attribute ID/delta, then six signed realm resistances.
func equipmentEffectSumNode(in []int64, r Rand) []int64 {
	if len(in) != equipmentBufferInput+equipmentBufferSize {
		panic("equipment: input arity")
	}
	for _, id := range in[:equipmentSlots] {
		if id < -1 || id >= equipmentRecords {
			panic("equipment: slot must be empty or identify an input item")
		}
	}
	for i := 0; i < equipmentRecords; i++ {
		item := in[equipmentSlots+i*equipmentFields : equipmentSlots+(i+1)*equipmentFields]
		for j, v := range item {
			hi := int64(127)
			lo := int64(-128)
			if j == 5 {
				lo, hi = -1, 40
			} // 41 skills, buffer +13..3b (0x50ea21)
			if j == 7 {
				lo, hi = -1, 6
			} // seven attributes, buffer +c..12 (0x50ea41)
			if v < lo || v > hi {
				panic("equipment: invalid item field")
			}
		}
	}
	for _, v := range in[equipmentBufferInput:] {
		if v < 0 || v > 255 {
			panic("equipment: buffer must contain bytes")
		}
	}
	out := append([]int64(nil), in[equipmentBufferInput:]...)
	add := func(offset int, delta int64) { out[offset] = int64(uint8(out[offset] + delta)) }
	var resistance [6]int64 // 0x50e98d: zeroed independently of caller buffer
	for slot, id := range in[:equipmentSlots] {
		if slot == 8 || slot == 9 || id == -1 {
			continue
		} // 0x50e9a4..0x50e9bc
		item := in[equipmentSlots+int(id)*equipmentFields : equipmentSlots+(int(id)+1)*equipmentFields]
		if slot != 6 && slot != 7 {
			add(0, item[0])
			add(1, item[1])
		} // 0x50e9d7..0x50e9f2
		for j := 0; j < 3; j++ {
			add(9+j, item[2+j])
		} // 0x50e9f5..0x50ea13
		if item[5] != -1 {
			add(0x13+int(item[5]), item[6])
		} // 0x50ea16..0x50ea34
		if item[7] != -1 {
			add(0xc+int(item[7]), item[8])
		} // 0x50ea36..0x50ea54
		for j := range resistance {
			resistance[j] += item[9+j]
		} // 0x50ea5c
	}
	for j, v := range resistance {
		out[0x3c+j] = int64(uint8(min(int64(125), max(int64(-125), v)))) // 0x50ea7f..0x50ea9c
	}
	return out
}

func equipmentBufferName(offset int) string {
	switch offset {
	case 0:
		return "initiativeModifier"
	case 1:
		return "toHitModifier"
	case 9:
		return "hitPointRecoveryModifier"
	case 10:
		return "staminaRecoveryModifier"
	case 11:
		return "spellPointRecoveryModifier"
	}
	if offset >= 12 && offset < 19 {
		return []string{"strength", "intelligence", "piety", "vitality", "dexterity", "speed", "senses"}[offset-12] + "Modifier"
	}
	if offset >= 19 && offset < 60 {
		return fmt.Sprintf("skillModifier[%d]", offset-19)
	}
	if offset >= 60 {
		return fmt.Sprintf("realmResistance[%d]", offset-60)
	}
	return fmt.Sprintf("callerBuffer[%d]", offset)
}

func init() {
	var inputs, outputs []string
	for slot := 0; slot < equipmentSlots; slot++ {
		inputs = append(inputs, fmt.Sprintf("equipmentItemID[%d]", slot))
	}
	fields := []string{"initiative", "toHit", "hitPointRecovery", "staminaRecovery", "spellPointRecovery", "skillID", "skillDelta", "attributeID", "attributeDelta", "resistance[0]", "resistance[1]", "resistance[2]", "resistance[3]", "resistance[4]", "resistance[5]"}
	for item := 0; item < equipmentRecords; item++ {
		for _, field := range fields {
			inputs = append(inputs, fmt.Sprintf("item[%d].%s", item, field))
		}
	}
	for offset := 0; offset < equipmentBufferSize; offset++ {
		name := equipmentBufferName(offset)
		inputs = append(inputs, "initial."+name)
		outputs = append(outputs, name)
	}
	Nodes["0050e980"] = Node{Addr: "0050e980", In: inputs, Out: outputs, F: equipmentEffectSumNode}
}
