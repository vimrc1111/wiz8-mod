package calc

// monsterPhysicalDamageNode models a loaded monster and a non-crediting damage
// source with inactive presentation. The case fixes reactive conditions except
// indefinite unconsciousness (9999 at 0x52c0f8), as in 00553540.json. No exe
// table contributes to this projection: absorption is supplied from the loaded
// template. Event pointers are allocation identities; count/capacity are outputs.
func monsterPhysicalDamageNode(in []int64, r Rand) []int64 {
	if len(in) != 13 {
		panic("monster physical damage: expected 13 inputs")
	}
	limits := [][2]int64{{-2147483648, 2147483647}, {-128, 127}, {0, 255}, {0, 4294967295}, {0, 4294967295}, {0, 65535}, {0, 4294967295}, {1, 4294967295}, {0, 4294967295}, {0, 4294967295}, {0, 255}, {0, 4}, {0, 8}}
	for i, bound := range limits {
		if in[i] < bound[0] || in[i] > bound[1] {
			panic("monster physical damage: invalid field width or arena list bound")
		}
	}
	if in[11] > in[12] {
		panic("monster physical damage: list count exceeds capacity")
	}
	hp, total := uint32(in[3]), uint32(in[4])
	shield, duration := uint16(in[5]), uint32(in[6])
	stamina, fatigue := uint32(in[8]), uint32(in[9])
	count, capacity := in[11], in[12]
	out := func() []int64 {
		return []int64{int64(hp), int64(total), int64(shield), int64(duration), int64(stamina), int64(fatigue), count, capacity}
	}
	damage := int32(in[0])
	absorption := int32(in[1] + in[2]) // 0x5459b8/0x5459c1: i8 + u8.
	if absorption != 0 {
		// 0x5459cb..0x5459eb: wrapping i32 product/add, division toward zero.
		damage = ((100-absorption)*damage + 50) / 100
	}
	if damage <= 0 || hp == 0 {
		return out()
	}
	remaining := uint32(damage)
	if duration != 0 {
		if uint32(shield) >= remaining {
			shield -= uint16(remaining)
			return out()
		}
		remaining -= uint32(shield)
		shield, duration = 0, 0 // 0x5243ba removes overflowing damage shield.
	}
	if in[10] == 0 {
		total += remaining // 0x553819: returned post-shield damage, including overkill.
	}
	if remaining >= hp {
		hp = 0 // 0x52bd77: inactive presentation skips the later death cleanup.
		if in[10] == 0 {
			// 0x52bd9d..0x52be44 emits one death record; 0x55386a..0x5538d1
			// transfers it, enlarging capacity by exactly one when necessary.
			count++
			if count > capacity {
				capacity = count
			}
		}
		return out()
	}
	hp -= remaining
	cost := (remaining * 2) / 3 // 0x52be5d..0x52be6c: wrapping u32 double.
	if cost >= stamina {
		stamina = 0
	} else {
		stamina -= cost
	}
	// 0x52c0ad..0x52c0ee: wrapping percentage, signed deficit thresholds.
	deficit := int32(100 - stamina*100/uint32(in[7]))
	switch {
	case deficit < 50:
		fatigue = 0
	case deficit < 70:
		fatigue = 1
	case deficit < 85:
		fatigue = 2
	case deficit < 95:
		fatigue = 3
	default:
		fatigue = 4
	}
	return out()
}

func init() {
	Nodes["00553770"] = Node{Addr: "00553770",
		In:  []string{"damage", "dynamicAbsorption", "baseAbsorption", "HP", "partyDamage", "damageShield", "shieldDuration", "maxStamina", "stamina", "fatigue", "specialMode", "eventCount", "eventCapacity"},
		Out: []string{"HP", "partyDamage", "damageShield", "shieldDuration", "stamina", "fatigue", "eventCount", "eventCapacity"}, F: monsterPhysicalDamageNode}
}
