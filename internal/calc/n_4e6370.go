package calc

import "math"

const monsterRerollSlots, monsterRerollFields = 4, 11

// monsterHpSpRerollNode re-rolls HP/SP for the monsters of the normal list (exe 0x683fad)
// and then the special list (0x683fb9); case wiring gives each list at most 2 members
// (arena capacity): slots 0,1 are list A members 0,1 and slots 2,3 are list B members 0,1.
// Inputs: listA pointer, listA count (list +0x8), listB pointer, listB count, then per slot
// priorMark (value of all six output fields before the call), template pointer (cache entry
// 0x6840c7+4*species), HP dice Z/N/S (template +0xd6/+0xd8/+0xd9), SP dice Z/N/S
// (+0xda/+0xdc/+0xdd) and the model size fields f32 bits (+0x5f0 base, +0x5f4 min, +0x5f8 max).
// A null list pointer counts as zero members (0x5e2c70). Per monster, in order: HP dice
// (0x517970) -> +0x27/+0x2b; SP dice -> +0x2f/+0x33; stamina tier of (SP,SP) -> +0x242;
// size scale (monsterHpScaleNode, 0x4e65d0) stored as f32 -> +0x24f; then the model refresh calls.
// Precondition: the species template is already cached (template pointer nonzero). A cache
// miss makes 0x54a8a0 read the species template file; that path is outside this node.
// The model refresh calls (0x4c57a0, 0x4c5810, 0x4c5ed0) only touch animation/scene state, not
// the six outputs. Differences from the original: a zero SP roll divides by zero there and panics here.
func monsterHpSpRerollNode(in []int64, r Rand) []int64 {
	if len(in) != 4+monsterRerollSlots*monsterRerollFields {
		panic("monster hp/sp reroll: wrong input count")
	}
	count := func(p, n int64) int {
		if p == 0 {
			return 0
		}
		if n < 0 || n > 2 {
			panic("monster hp/sp reroll: list count outside arena capacity 0..2")
		}
		return int(n)
	}
	todo := [monsterRerollSlots]bool{}
	for i := 0; i < count(in[0], in[1]); i++ {
		todo[i] = true
	}
	for i := 0; i < count(in[2], in[3]); i++ {
		todo[2+i] = true
	}
	out := make([]int64, 0, monsterRerollSlots*6)
	for k := 0; k < monsterRerollSlots; k++ {
		s := in[4+k*monsterRerollFields : 4+(k+1)*monsterRerollFields]
		if !todo[k] {
			out = append(out, s[0], s[0], s[0], s[0], s[0], s[0])
			continue
		}
		if s[1] == 0 {
			panic("monster hp/sp reroll: template not cached")
		}
		hp := uint32(diceNode(s[2:5], r)[0])
		sp := uint32(diceNode(s[5:8], r)[0])
		if sp == 0 {
			panic("monster hp/sp reroll: original SP division by zero")
		}
		// 0x52a780(sp, sp): u32 wrap multiply, unsigned divide, thresholds 50/70/85/95.
		depletion := int32(100 - (sp*100)/sp)
		tier := int64(0)
		switch {
		case depletion >= 95:
			tier = 4
		case depletion >= 85:
			tier = 3
		case depletion >= 70:
			tier = 2
		case depletion >= 50:
			tier = 1
		}
		bits := monsterHpScaleNode([]int64{s[8], s[9], s[10], int64(hp), s[2], s[3], s[4]}, r)[0]
		size := int64(math.Float32bits(float32(math.Float64frombits(uint64(bits)))))
		out = append(out, int64(hp), int64(hp), int64(sp), int64(sp), tier, size)
	}
	return out
}

func init() {
	names := []string{"listAPointer", "listACount", "listBPointer", "listBCount"}
	outs := []string{}
	for k := 0; k < monsterRerollSlots; k++ {
		p := "[" + string(rune('0'+k)) + "]"
		for _, n := range []string{"priorMark", "templatePointer", "hpZ", "hpN", "hpS", "spZ", "spN", "spS", "baseSizeBits", "sizeMinBits", "sizeMaxBits"} {
			names = append(names, n+p)
		}
		for _, n := range []string{"hp", "hpMax", "sp", "spMax", "spTier", "sizeBits"} {
			outs = append(outs, n+p)
		}
	}
	Nodes["004e6370"] = Node{Addr: "004e6370", In: names, Out: outs, F: monsterHpSpRerollNode}
}
