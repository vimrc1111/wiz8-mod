package calc

import (
	"math"
	"math/big"
)

func init() {
	Nodes["0052ba80"] = Node{
		Addr: "0052ba80",
		In:   []string{"monsterCount", "centerXBits", "centerYBits", "centerZBits", "radiusBits", "Z", "N", "S", "monsterActive", "monsterXBits", "monsterYBits", "monsterZBits", "monsterHP", "resultDamage", "resultHits", "monsterMaxStamina", "monsterStamina", "damageShield", "shieldDuration", "fatigue"},
		Out:  []string{"monsterHP", "resultDamage", "resultHits", "monsterStamina", "damageShield", "shieldDuration", "fatigue"},
		F:    radiusMonsterDamageRollNode,
	}
}

// radiusMonsterDamageRollNode replays the case's single loaded monster, with
// no combat UI or reactive status effects. Arena links and
// the non-character damage source are fixed in 0052ba80.json. No exe table is
// needed for the HP/result/stamina projection in this context.
func radiusMonsterDamageRollNode(in []int64, r Rand) []int64 {
	if len(in) != 20 || in[0] < 0 || in[0] > 1 {
		panic("radius damage: expected a zero or one monster list and 20 inputs")
	}
	for _, i := range []int{1, 2, 3, 4, 9, 10, 11, 12, 13, 14, 15, 16, 18, 19} {
		if in[i] < 0 || in[i] > math.MaxUint32 {
			panic("radius damage: expected u32 fields and binary32 bits")
		}
	}
	if in[8] < 0 || in[8] > 255 || in[15] == 0 || in[17] < 0 || in[17] > 65535 {
		panic("radius damage: expected u8 active flag and nonzero maximum stamina")
	}
	hp, total, hits, stamina := uint32(in[12]), uint32(in[13]), uint32(in[14]), uint32(in[16])
	shield, duration, fatigue := uint16(in[17]), uint32(in[18]), uint32(in[19])
	out := func() []int64 {
		return []int64{int64(hp), int64(total), int64(hits), int64(stamina), int64(shield), int64(duration), int64(fatigue)}
	}
	if in[0] != 0 && in[8] != 0 && radius52ba80Contains(in) {
		// 0x52bb13 rolls even for a target with zero HP. 0x52bb60 interprets
		// that signed dice result as an unsigned damage amount.
		damage := uint32(diceNode(in[5:8], r)[0])
		if hp != 0 {
			// 0x52bb72..0x52bbf3 absorbs before accumulating damage.
			// Full absorption counts a hit but contributes no damage.
			if duration != 0 {
				if uint32(shield) >= damage {
					shield -= uint16(damage)
					hits++
					return out()
				}
				damage -= uint32(shield)
				// 0x5243ba removes the entire shield effect on overflow.
				shield, duration = 0, 0
			}
			// 0x52bbf9 accumulates the full roll, including overkill and zero.
			total += damage
			hits++
			if damage >= hp {
				hp = 0
				// 0x4e42e5/0x4e42e8 clears HP and stamina on death.
				stamina = 0
			} else if damage != 0 {
				hp -= damage
				// 0x52be5d and 0x52c070: unsigned 32-bit doubling wraps
				// before division by 3; drain saturates at current stamina.
				drain := (damage * 2) / 3
				if drain >= stamina {
					stamina = 0
				} else {
					stamina -= drain
				}
				// 0x52c0ad..0x52c0ee: wrapping u32 percentage, signed
				// deficit comparisons against 50, 70, 85 and 95.
				deficit := int32(100 - stamina*100/uint32(in[15]))
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
			}
		}
	}
	return out()
}

func radius52ba80Contains(in []int64) bool {
	decode := func(v int64) float64 { return float64(math.Float32frombits(uint32(v))) }
	radius := decode(in[4])
	var delta [3]float64
	for axis := range delta {
		delta[axis] = decode(in[1+axis]) - decode(in[9+axis])
	}
	// 0x52bb08 accepts unordered comparisons, as well as distance <= radius.
	if math.IsNaN(radius) || math.IsNaN(delta[0]) || math.IsNaN(delta[1]) || math.IsNaN(delta[2]) {
		return true
	}
	if math.IsInf(radius, 1) {
		return true
	}
	if radius < 0 || math.IsInf(delta[0], 0) || math.IsInf(delta[1], 0) || math.IsInf(delta[2], 0) {
		return false
	}
	// 0x52bade..0x52bb08 uses x87 extended intermediates (64 significand
	// bits under the emulator's 0x037f control word). Preserve each rounding
	// and the original Z-square + Y-square + X-square order, including sqrt.
	f := func() *big.Float { return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven) }
	var square [3]*big.Float
	for axis := range square {
		d := f().Sub(f().SetFloat64(decode(in[1+axis])), f().SetFloat64(decode(in[9+axis])))
		square[axis] = f().Mul(d, d)
	}
	sum := f().Add(square[2], square[1])
	sum.Add(sum, square[0])
	return f().Sqrt(sum).Cmp(f().SetFloat64(radius)) <= 0
}
