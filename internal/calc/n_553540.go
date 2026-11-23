package calc

// periodicMonsterPhysicalDamageNode combines the periodic dice, physical
// absorption and monster HP entry. The arena uses a loaded template, no
// equipment, a non-crediting source, and an inactive visual cycle. Other status
// effects are zero except unconscious duration, fixed at the indefinite 9999
// sentinel (0x52c0f8), so stamina exhaustion cannot create a new status here.
// Sources: 0x553540, 0x5459b0, 0x52bb60 and 0x52c070 in the player's exe.
func periodicMonsterPhysicalDamageNode(in []int64, r Rand) []int64 {
	if len(in) != 13 {
		panic("periodic monster damage: expected 13 inputs")
	}
	limits := [][2]int64{{-2147483648, 2147483647}, {-32768, 32767}, {0, 255}, {0, 255}, {-128, 127}, {0, 255}, {0, 4294967295}, {0, 4294967295}, {0, 65535}, {0, 4294967295}, {1, 4294967295}, {0, 4294967295}, {0, 4294967295}}
	for i, bounds := range limits {
		if in[i] < bounds[0] || in[i] > bounds[1] {
			panic("periodic monster damage: input outside its record width (maxStamina must be positive)")
		}
	}
	power := int32(in[0])
	hp, total := uint32(in[6]), uint32(in[7])
	shield, duration := uint16(in[8]), uint32(in[9])
	maxStamina, stamina, fatigue := uint32(in[10]), uint32(in[11]), uint32(in[12])
	out := func() []int64 {
		return []int64{int64(hp), int64(total), int64(shield), int64(duration), int64(stamina), int64(fatigue)}
	}
	if power == 0 {
		return out()
	}
	// 0x553561..0x55356b: the product retains only the low byte;
	// the zero guard above still examines all four bytes of the effect power.
	n := uint8(uint8(power) * uint8(in[2]))
	damage := int32(diceNode([]int64{in[1], int64(n), in[3]}, r)[0])
	absorption := int32(int8(in[4])) + int32(uint8(in[5]))
	if absorption != 0 {
		// 0x5459cb..0x5459eb: signed percentage rounding, including
		// the 32-bit product and addition before division toward zero.
		damage = ((100-absorption)*damage + 50) / 100
	}
	if damage < 0 {
		damage = 0
	}
	if damage == 0 {
		return out()
	}
	// 0x5535bc: count the pre-shield damage even for an already dead monster.
	total += uint32(damage)
	if hp == 0 {
		return out()
	}
	remaining := uint32(damage)
	if duration != 0 {
		if uint32(shield) >= remaining {
			shield -= uint16(remaining)
			return out()
		}
		remaining -= uint32(shield)
		// 0x5243ba: remove the entire damage-shield effect on overflow.
		shield, duration = 0, 0
	}
	if remaining >= hp {
		hp = 0
		return out()
	}
	hp -= remaining
	// 0x52be5d..0x52be6c: unsigned, wrapping double then divide by 3.
	cost := (remaining * 2) / 3
	if cost >= stamina {
		stamina = 0
	} else {
		stamina -= cost
	}
	// 0x52c0ad..0x52c0ee: the product wraps before unsigned division;
	// threshold comparisons interpret the deficit as a signed integer.
	deficit := int32(100 - stamina*100/maxStamina)
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
	Nodes["00553540"] = Node{
		Addr: "00553540",
		In: []string{"power", "Z", "N", "S", "dynamicAbsorption", "baseAbsorption",
			"HP", "partyDamage", "damageShield", "shieldDuration", "maxStamina", "stamina", "fatigue"},
		Out: []string{"HP", "partyDamage", "damageShield", "shieldDuration", "stamina", "fatigue"},
		F:   periodicMonsterPhysicalDamageNode,
	}
}
