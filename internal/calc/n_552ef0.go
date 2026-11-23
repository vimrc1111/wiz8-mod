package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"unicode/utf16"
)

var ongoingAreaSpellNames = func() []string {
	names := []string{"flag"}
	for i := 0; i < 9; i++ {
		names = append(names, fmt.Sprintf("active_%d", i), fmt.Sprintf("type_%d", i), fmt.Sprintf("power_%d", i))
	}
	names = append(names, "Z", "N", "S", "realm", "radiusBits", "areaCount", "id_0", "id_1", "id_2", "id_3")
	realms := []string{"fire", "water", "air", "earth", "mental", "divine"}
	for k := 0; k < 3; k++ {
		for _, n := range []string{"hp", "maxStamina", "stamina", "damageShield", "shieldDuration", "fatigue", "monsterActive"} {
			names = append(names, fmt.Sprintf("%s_%d", n, k))
		}
		for _, n := range realms {
			names = append(names, fmt.Sprintf("resistanceModifier_%d_%s", k, n))
		}
	}
	for k := 0; k < 3; k++ {
		for _, n := range realms {
			names = append(names, fmt.Sprintf("templateResistance_%d_%s", k, n))
		}
		names = append(names, fmt.Sprintf("templateLevel_%d", k))
	}
	return names
}()

func init() {
	var out []string
	for k := 0; k < 3; k++ {
		for _, n := range []string{"hp", "stamina", "damageShield", "shieldDuration", "fatigue"} {
			out = append(out, fmt.Sprintf("%s_%d", n, k))
		}
		for _, n := range []string{"fire", "water", "air", "earth", "mental", "divine"} {
			out = append(out, fmt.Sprintf("resistanceModifier_%d_%s", k, n))
		}
	}
	out = append(out, "messageGate", "areaQueries")
	for i := 0; i < 9; i++ {
		out = append(out, fmt.Sprintf("areaRangeBits_%d", i))
	}
	out = append(out, "logLines", "monsterMessages")
	for i := 0; i < 10; i++ {
		out = append(out, fmt.Sprintf("summary_%d", i))
	}
	Nodes["00552ef0"] = Node{Addr: "00552ef0", In: ongoingAreaSpellNames, Out: out, F: ongoingAreaSpellDamageNode}
}

// ongoingAreaSpellDamageNode replays the nine ongoing-effect slots of 0x552ef0
// against the case's three loaded monsters (IDs 0x100..0x102, template k for
// monster k; record-count cap). The 539e70 area query is an excluded stub whose
// list comes from the case (areaCount, id_0..id_3); its fourth argument, the
// query range, is reproduced bit for bit. 58aad0 log lines and 590b40 monster
// messages are counted. The summary is the UTF-16 text 517a90 formats into
// 0x689fd0 with the case's format strings "A%x" (+0x668) and "B%x/%x"
// (+0x664); untouched units keep the case's 0xdddd fill.
// Sources: 0x552ef0, 0x51a9a0, 0x517970, 0x552250, 0x52bb60, 0x52c070 in the
// player's exe; 0x5ebc40 (range scale) is read from the exe at run time.
func ongoingAreaSpellDamageNode(in []int64, r Rand) []int64 {
	if len(in) != len(ongoingAreaSpellNames) {
		panic("ongoing area spell: wrong input count")
	}
	get := func(i int, lo, hi int64) int64 {
		if in[i] < lo || in[i] > hi {
			panic(fmt.Sprintf("ongoing area spell: %s outside its record width", ongoingAreaSpellNames[i]))
		}
		return in[i]
	}
	flag := get(0, 0, 255) != 0
	z, n, s := get(28, -32768, 32767), uint8(get(29, 0, 255)), get(30, 0, 255)
	realm := get(31, 0, 5) // Six realm resistances (combat-2.md:71); 552250 indexes the template by it.
	radiusBits := uint32(get(32, 0, math.MaxUint32))
	count := int32(get(33, math.MinInt32, math.MaxInt32))
	if count > 4 {
		panic("ongoing area spell: the case supplies at most four area-list entries")
	}
	type monster struct {
		hp, maxStamina, stamina, duration, fatigue uint32
		shield                                     uint16
		mods                                       [6]int64
		active                                     bool
	}
	var mons [3]monster
	var tmpl [3][7]int64
	for k := 0; k < 3; k++ {
		b := 38 + 13*k
		mons[k] = monster{hp: uint32(get(b, 0, math.MaxUint32)), maxStamina: uint32(get(b+1, 1, math.MaxUint32)),
			stamina: uint32(get(b+2, 0, math.MaxUint32)), shield: uint16(get(b+3, 0, 65535)),
			duration: uint32(get(b+4, 0, math.MaxUint32)), fatigue: uint32(get(b+5, 0, math.MaxUint32)), active: get(b+6, 0, 255) != 0}
		for j := 0; j < 6; j++ {
			mons[k].mods[j] = get(b+7+j, -128, 127)
		}
		for j := 0; j < 7; j++ {
			tmpl[k][j] = get(77+7*k+j, 0, 255)
		}
	}
	var ids [4]int64
	for j := range ids {
		ids[j] = get(34+j, 0, math.MaxUint32)
	}
	rangeBits := ongoingAreaRange(radiusBits)

	var gate, queries, logs, messages uint32
	var ranges [9]uint32
	summary := make([]uint16, 20)
	for i := range summary {
		summary[i] = 0xdddd
	}
	// Accumulator at esp+0x60 is zeroed once (0x552f8e) and spans all slots.
	var total, hits uint32
	var events []uint32
	for slot := 0; slot < 9; slot++ {
		active := get(1+3*slot, 0, 255)
		typ := get(2+3*slot, 0, math.MaxUint32)
		power := uint32(get(3+3*slot, 0, math.MaxUint32))
		if active == 0 || typ != 0x3e { // 0x552fa0, 0x552fa8.
			continue
		}
		logs++   // 0x552fcc: 58aad0(8, …).
		gate = 1 // 0x552fd5: 5905c0(1, -1) sets 0x69b7b8.
		ranges[queries] = rangeBits
		queries++ // 0x553038: 539e70(…, range, 1, 0).
		for i := int32(0); i < count; i++ {
			if i >= count { // 0x55304e/0x55305b: bounds fallback of the list accessor.
				panic("ongoing area spell: list index outside the area list")
			}
			id := ids[i]
			if id < 0x100 || id > 0x102 {
				// 0x553082: an ID absent from the loaded lists stops in srAssertFail.
				panic("ongoing area spell: area-list ID outside the case's loaded monsters")
			}
			k := int(id - 0x100)
			m := &mons[k]
			// 0x5530ab..0x5530b7: the dice count keeps only the low byte of N*power.
			roll := uint32(diceNode([]int64{z, int64(n * uint8(power)), s}, r)[0])
			q := []int64{int64(roll), 0, 0, int64(power * 3), 3, realm, 0, 0, 0, 0, 0, 0}
			q = append(q, tmpl[k][:]...)
			q = append(q, m.mods[:]...)
			q = append(q, id, int64(k))
			targetType := q[4] // 0x5530cd: the descriptor is always a monster.
			damage := uint32(resistanceQuantityReductionNode(q, r)[0])
			if damage == 0 {
				if flag {
					if targetType != 3 {
						panic("ongoing area spell: 0x553148 needs a non-monster target")
					}
					messages++ // 0x55313e: 590b40.
				}
				continue
			}
			// 0x52bb60(mon, damage, src, 0, flag, 0, flag ? NULL : &acc, 0).
			if m.hp == 0 {
				continue
			}
			if m.duration != 0 {
				messages++ // 0x52bba6/0x52bbd1: 590b40.
				if uint32(m.shield) >= damage {
					m.shield -= uint16(damage)
					if !flag {
						hits++
					}
					continue
				}
				damage -= uint32(m.shield)
				// 0x52bbeb: 524390 removes the shield effect, then 50e8c0
				// rebuilds the modifier block 0x1db..0x241 from the monster's
				// condition, effect and equipment records, which the case fixes
				// at zero (unconscious duration 9999): every resistance
				// modifier becomes 0 (asserted by the resistanceModifier outputs).
				m.shield, m.duration = 0, 0
				m.mods = [6]int64{}
			}
			if !flag {
				total += damage
				hits++
			}
			if damage >= m.hp {
				m.hp = 0
				if m.active { // 0x4e42ab: 4e4280 (via 4e4690) clears stamina of an active monster.
					m.stamina = 0
				}
				if !flag {
					events = append(events, 3) // 0x52bdab: a type-3 kill event on the accumulator list.
				}
				continue
			}
			m.hp -= damage
			// 0x52be5d..0x52be6c: unsigned wrapping double, then /3.
			if drain := (damage * 2) / 3; drain >= m.stamina {
				m.stamina = 0
			} else {
				m.stamina -= drain
			}
			// 0x52c0ad..0x52c0ee: wrapping percentage, signed deficit thresholds.
			deficit := int32(100 - m.stamina*100/m.maxStamina)
			switch {
			case deficit < 50:
				m.fatigue = 0
			case deficit < 70:
				m.fatigue = 1
			case deficit < 85:
				m.fatigue = 2
			case deficit < 95:
				m.fatigue = 3
			default:
				m.fatigue = 4
			}
		}
		if flag { // 0x5531ab.
			continue
		}
		var text string
		switch {
		case hits == 0: // 0x5531da: a fixed message, no formatting.
		case count == 1: // 0x5531f3.
			text = fmt.Sprintf("A%x", total)
		default: // 0x553215: unsigned division.
			text = fmt.Sprintf("B%x/%x", hits, total/hits)
		}
		if text != "" {
			u := append(utf16.Encode([]rune(text)), 0)
			copy(summary, u)
		}
		// 0x553246..0x5532ef drains the kill events; only type 3 is ever queued.
		for _, t := range events {
			if t != 3 {
				panic("ongoing area spell: 0x55328c needs a non-kill event")
			}
			logs++ // 0x5532d4: 58aad0(9, …).
		}
		events = events[:0]
	}
	out := make([]int64, 0, 56)
	for _, m := range mons {
		out = append(out, int64(m.hp), int64(m.stamina), int64(m.shield), int64(m.duration), int64(m.fatigue))
		out = append(out, m.mods[:]...)
	}
	out = append(out, int64(gate), int64(queries))
	for _, b := range ranges {
		out = append(out, int64(b))
	}
	out = append(out, int64(logs), int64(messages))
	for i := 0; i < 20; i += 2 {
		out = append(out, int64(uint32(summary[i])|uint32(summary[i+1])<<16))
	}
	return out
}

// ongoingAreaRange is the float32 range 0x552fe1..0x553026 passes to 539e70:
// 51a9a0(1) (jump-table entry 0x51a9c2 gives 4) times 0x5ebc40, stored as
// binary32, plus the party radius, summed on x87 with 64 significand bits and
// stored as binary32.
func ongoingAreaRange(radiusBits uint32) uint32 {
	scale := math.Float32frombits(binary.LittleEndian.Uint32(ticRead(0x5ebc40, 4)))
	reach := float64(float32(4 * float64(scale)))
	radius := math.Float32frombits(radiusBits)
	if math.IsNaN(float64(radius)) {
		return radiusBits | 0x400000 // Quiet NaN keeps its payload.
	}
	f := func() *big.Float { return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven) }
	sum := f().Add(f().SetFloat64(float64(radius)), f().SetFloat64(reach))
	v, _ := sum.Float32()
	return math.Float32bits(v)
}
