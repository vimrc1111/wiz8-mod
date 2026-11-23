package calc

import "fmt"

const (
	n5208CharBase   = 0x20000000 // emu arena start: the case places the eight character records here
	n5208CharStride = 6242       // 0x1862, exe 0x517ef9
	n5208Items      = 819        // ITEMS.DBS record count (notes/re/loot.md:9); fixture cap
)

func init() {
	Nodes["005208f0"] = Node{Addr: "005208f0", F: itemIdentificationNode,
		In: []string{"slot", "charAddr", "itemId", "itemRequiredSkill", "charPoolsEnabled", "charState", "charSkillScore",
			"flag6", "flag7", "flag8", "flag9", "flagA", "flagB", "flagC", "trainRet"},
		Out: []string{"flag6", "flag7", "flag8", "flag9", "flagA", "flagB", "flagC", "result",
			"trainCalls", "trainChar", "trainSkill", "trainRepeat", "trainQuiet"}}
}

// itemIdentificationNode is exe 0x5208f0(C, item): a character who has skill pools
// (+0xb11 != 0) in a state below 11 (+0xb01) identifies an item when
// score(+0x49b)/6 reaches the item definition's byte at +0x85. All byte compares are
// signed 8-bit (exe 0x520963, 0x5209ab). Identified items get flags +6/+7/+8, +0xb when the
// level is 3 over the need, and a level within 0..2 of the need trains skill 0x14 (twice,
// quiet) through 0x553f10, which the case address-stubs: the node outputs its recorded
// arguments and call count. The stub's return (trainRet) is ignored by the caller (exe 0x5209da).
// The null item pointer (assert at 0x520902) is not an input: the item is a record.
func itemIdentificationNode(in []int64, r Rand) []int64 {
	if len(in) != 15 {
		panic("item identification: wrong input count")
	}
	slot, addr, id := in[0], in[1], in[2]
	if slot < 0 || slot > 7 || addr != n5208CharBase+n5208CharStride*slot {
		panic("item identification: character is not one of the eight party records")
	}
	if id < 0 || id >= n5208Items {
		panic("item identification: item id outside the fixture's 819 definitions")
	}
	for i, v := range in[3:] {
		hi := int64(255)                 // required skill and the seven flag bytes
		if i >= 1 && i <= 3 || i == 11 { // pools, state, score, stub return: u32
			hi = 1<<32 - 1
		}
		if v < 0 || v > hi {
			panic(fmt.Sprintf("item identification: input %d outside its memory width", i+3))
		}
	}
	req, pools, state, score := in[3], in[4], in[5], in[6]
	flags := append([]int64(nil), in[7:14]...)
	out := func(res, calls int64) []int64 {
		o := append(flags, res, calls, 0, 0, 0, 0)
		if calls != 0 {
			o[9], o[10], o[11], o[12] = addr, 0x14, 2, 1 // push order at 0x5209ce..0x5209d4
		}
		return o
	}
	if pools == 0 || state >= 0xb { // 0x520922, 0x52092a
		return out(0, 0)
	}
	bl := int8(score / 6) // 0x520937..0x520960: low byte of the quotient
	need := int8(req)
	if bl < need { // 0x520963
		return out(0, 0)
	}
	flags[0], flags[1], flags[2] = 1, 1, 1
	if bl >= need+3 { // 0x5209a8: 8-bit add wraps
		flags[5] = 1
	}
	if d := int64(bl) - req; d >= 0 && d <= 2 { // 0x5209c2..0x5209cc: signed bl minus unsigned need
		return out(1, 1)
	}
	return out(1, 0)
}
