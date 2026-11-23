package calc

func init() {
	names := []string{"value", "Rj", "Lt", "S", "targetType", "realm", "characterIndex"}
	for _, n := range []string{"water", "air", "earth", "mental", "divine"} {
		names = append(names, "characterResistance_"+n)
	}
	for _, n := range []string{"fire", "water", "air", "earth", "mental", "divine"} {
		names = append(names, "templateResistance_"+n)
	}
	names = append(names, "templateLevel")
	for _, n := range []string{"fire", "water", "air", "earth", "mental", "divine"} {
		names = append(names, "monsterResistanceModifier_"+n)
	}
	names = append(names, "monsterID", "templateIndex")
	Nodes["00552250"] = Node{Addr: "00552250", In: names, Out: []string{"value"}, F: resistanceQuantityReductionNode}
}

// resistanceQuantityReductionNode uses independently supplied record fields.
// The five-input form preserves the original adapter's public calling convention:
// Rj and Lt are already selected and combined there. The 27-input replay form
// selects the realm from character or monster records (combat-2.md:71).
// No executable data table is read by this formula or its loaded-record lookups.
func resistanceQuantityReductionNode(in []int64, r Rand) []int64 {
	if len(in) != 5 && len(in) != 27 {
		panic("resistance quantity reduction: expected 5 or 27 inputs")
	}
	for _, i := range []int{0, 1, 2, 3, 4} {
		if in[i] < 0 || in[i] > 0xffffffff {
			panic("resistance quantity reduction: expected u32 fields")
		}
	}
	v := uint32(in[0])
	resistance, level, strength := uint32(in[1]), uint32(in[2]), uint32(in[3])
	if len(in) == 27 {
		if in[5] < 0 || in[5] > 5 || in[6] < 0 || in[6] > 7 || in[26] < 0 || in[26] > 2 {
			panic("resistance quantity reduction: invalid record index")
		}
		for i := 7; i < 12; i++ {
			if in[i] < 0 || in[i] > 0xffffffff {
				panic("resistance quantity reduction: expected u32 character resistance")
			}
		}
		for i := 12; i <= 18; i++ {
			if in[i] < 0 || in[i] > 255 {
				panic("resistance quantity reduction: expected u8 template field")
			}
		}
		for i := 19; i <= 24; i++ {
			if in[i] < -128 || in[i] > 127 {
				panic("resistance quantity reduction: expected i8 monster modifier")
			}
		}
		if in[25] < 0 || in[25] > 0xffffffff {
			panic("resistance quantity reduction: expected u32 monster ID")
		}
		j := int(in[5]) // Six realms, combat-2.md:71.
		if in[4] == 3 { // Monster target tag, 0x552268.
			resistance = uint32(int32(in[12+j]) + int32(in[19+j]))
			level = uint32(in[18])
		} else if j != 0 {
			resistance = uint32(in[6+j])
		}
	}
	// Sentinel 9999 at 0x552255; effective resistance uses wrapping u32
	// arithmetic then a signed positive check at 0x5522f0 (formulas-2.md:45).
	er := int32(resistance + 3*(level-strength))
	if v == 9999 || er <= 0 {
		return []int64{int64(v)}
	}
	m := er + 1
	h := m / 2
	first := uint32(r.R(int(uint32(m - h))))
	second := uint32(r.R(int(uint32(h))))
	fr := min(uint32(100), uint32(h)+first+second) // 0x552315: cap at 100.
	// 0x552324 truncates the product to 32 bits before unsigned /100.
	// Subtract the reduction rather than rounding the remaining percentage.
	return []int64{int64(v - (v*fr)/100)}
}
