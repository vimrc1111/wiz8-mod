package calc

func init() {
	Nodes["00551a20"] = Node{
		Addr: "00551a20",
		In:   []string{"quantityBase", "quantityDiceCount", "quantityDieSides", "quantityBonusPercent"},
		Out:  []string{"quantity"},
		F:    effectQuantityNode,
	}
}

// effectQuantityNode consumes the effect's quantity dice (+4..+7) and
// percentage bonus (+0x24); no character records or tables are read.
func effectQuantityNode(in []int64, r Rand) []int64 {
	if len(in) != 4 || in[3] < -2147483648 || in[3] > 2147483647 {
		panic("effect quantity: expected i16 base, u8 count/sides and i32 bonus percent")
	}
	value := diceNode(in[:3], r)[0]
	// 9999 bypasses the bonus (0x551a31..0x551a3a;
	// notes/re/formulas-2.md:35), including when reached by a random roll.
	if value == 9999 {
		return []int64{value}
	}
	// 0x517f99..0x517faa uses the low 32 bits of the product as an
	// unsigned percentage numerator. The divisor is 100 (0x517f9e..0x517fa5).
	// SpellDamage's wide, signed floor differs for negative/overflowing products.
	product := uint32(value) * uint32(in[3])
	quantity := uint32(value) + product/100
	return []int64{int64(int32(quantity))}
}
