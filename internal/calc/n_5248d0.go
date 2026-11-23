package calc

import "fmt"

// condition_dependency_remove 0x5248d0: for dependency kind b = 0, 1 whose bit
// is set in the source's flag byte (+0x24c), scan the 8 party members' source
// records (record at character +0x1817 + 0x11*b, stride 0x1862: u32 tag, u32
// source id, byte flag; build/wiz8.asm:396883..). A record matches when its tag
// equals *0x686a70 and its source id equals *source (+0). For kind 1 only
// members whose state word +0xa4d (= +0xa01 + 4*0x13) is non-zero are scanned.
// A match zeroes the record and, for kind 1 with the member present
// (character +4 non-zero), calls character_condition_remove 0x523330(member,
// 0x13, 1). If kind 0 matched nothing, the first monster of the list at
// *0x683fad (0x4e5620 entry i, count by 0x5e2c70) whose +0x344 equals the
// source id gets +0x344 = 0xffffffff.
//
// Emu address-stubs the D4 callee 0x523330 (composition ruling 2026-10-09):
// its return is unused here, only the last call's arguments and the call count
// are outputs. F does not run the real callee, because the callee's effects are
// not read back by this node.
// Fixture caps: monster count <= cdrMonsters; more (or a list pointer other than
// 0 / the fixture list, or a non-present entry) panics.
const (
	cdrMembers  = 8
	cdrMonsters = 8
	cdrList     = 0x20000000 + 0x10800 // fixture list address (the only non-NULL pointer modelled)
	cdrCond     = 0x13                 // 0x524905
	cdrIn0      = 5                    // inputs before the per-member block
	cdrPerW     = 2 + 6
)

func init() {
	in := []string{"sourceId", "dependFlags", "sourceTag", "monsterListPtr", "monsterCount"}
	for w := 0; w < cdrMembers; w++ {
		in = append(in, fmt.Sprintf("present%d", w), fmt.Sprintf("state13_%d", w))
		for e := 0; e < 2; e++ {
			in = append(in, fmt.Sprintf("rec%d_%d_src", e, w), fmt.Sprintf("rec%d_%d_tgt", e, w), fmt.Sprintf("rec%d_%d_flag", e, w))
		}
	}
	for k := 0; k < cdrMonsters; k++ {
		in = append(in, fmt.Sprintf("mon%d_link", k))
	}
	var out []string
	for w := 0; w < cdrMembers; w++ {
		for e := 0; e < 2; e++ {
			out = append(out, fmt.Sprintf("rec%d_%d_src", e, w), fmt.Sprintf("rec%d_%d_tgt", e, w), fmt.Sprintf("rec%d_%d_flag", e, w))
		}
	}
	for k := 0; k < cdrMonsters; k++ {
		out = append(out, fmt.Sprintf("mon%d_link", k))
	}
	out = append(out, "calls", "lastWho", "lastCond", "lastNotify")
	Nodes["005248d0"] = Node{Addr: "005248d0", In: in, Out: out, F: conditionDependencyRemoveNode}
}

func conditionDependencyRemoveNode(in []int64, r Rand) []int64 {
	if len(in) != cdrIn0+cdrMembers*cdrPerW+cdrMonsters {
		panic("condition_dependency_remove: wrong input count")
	}
	in = append([]int64(nil), in...) // records are cleared in place
	id, flags, tag, listPtr, count := in[0]&0xffffffff, in[1]&0xff, in[2]&0xffffffff, in[3]&0xffffffff, in[4]&0xffffffff
	if listPtr != 0 && listPtr != cdrList {
		panic("condition_dependency_remove: monster list pointer outside the fixture")
	}
	if listPtr != 0 && count > cdrMonsters {
		panic("condition_dependency_remove: monster count above fixture cap")
	}
	m := func(w int) []int64 { return in[cdrIn0+w*cdrPerW : cdrIn0+(w+1)*cdrPerW] } // present, state, 2x(src,tgt,flag)
	mon := append([]int64(nil), in[cdrIn0+cdrMembers*cdrPerW:]...)
	var calls, who, cond, notify int64
	for b := 0; b < 2; b++ {
		if flags>>uint(b)&1 == 0 {
			continue
		}
		found := false
		for w := 0; w < cdrMembers; w++ {
			c := m(w)
			rec := c[2+3*b : 5+3*b]
			if b == 1 && uint32(c[1]) == 0 { // 0x524928 jbe: unsigned > 0 required
				continue
			}
			if uint32(rec[0]) != uint32(tag) || uint32(rec[1]) != uint32(id) {
				continue
			}
			found = true
			rec[0], rec[1], rec[2] = 0, 0, 0
			if b == 1 && c[0]&0xff != 0 {
				calls, who, cond, notify = calls+1, int64(w), cdrCond, 1
			}
		}
		if !found && b == 0 && listPtr != 0 {
			for i := int64(0); i < count; i++ {
				if uint32(mon[i]) == uint32(id) {
					mon[i] = 0xffffffff
					break
				}
			}
		}
	}
	var out []int64
	for w := 0; w < cdrMembers; w++ {
		out = append(out, m(w)[2:]...)
	}
	out = append(out, mon...)
	return append(out, calls, who, cond, notify)
}
