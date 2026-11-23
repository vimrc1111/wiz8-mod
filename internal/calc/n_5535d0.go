package calc

import "fmt"

// characterPhysicalDamageNode is 0x5535d0 character_physical_damage (who, unused, dmg)
// (build/wiz8.asm:460923..). Member C = *0x685174 + who*0x1862. 0x545950(C, dmg)
// reduces dmg by the character's D = i32 at C+0xed9: D = 0 keeps dmg, else
// trunc(((100-D)*dmg+50)/100) in wrapping i32; a negative result is 0 (0x545988..0x54598c).
// A nonzero byte C+0x71b calls 0x553f10(C, 0x25, 1, 0) first (0x545998). A reduced
// damage of 0 ends the node. Else, with mode = byte 0x68510c:
//   - mode nonzero: 0x52a890(who, dmg, 0, 1, 1, null, 1), the return is ignored (0x553649..);
//   - mode zero: 0x52a890(who, dmg, 0, 0, 0, &killList, 0) on the stack kill list (zeroed
//     record, list ctor 0x516950 at +0x58: count +0x5c, capacity +0x60, data +0x64); its
//     return value is added to *0x6836a8+0xa1e, then each pending death event is appended
//     to the party list (count +0xa02, capacity +0xa06, data +0xa0a; growth takes count+1
//     entries from operator new, 0 keeps the old list and drops the event).
//
// Callees. 0x553f10 and 0x52a890 are D4 formula callees address-stubbed in Emu; the
// 0x52a890 return comes from the case input stubRet, the event count and event data from
// stubEventCount and stubEvent0, 0x553f10's return is ignored (caller reads no eax; ret
// 0x5a5a5a5a in a scratch copy of the case gave diff=0). Emu records both callees' call
// counts and arguments. The kill pointer (a stack address in the original) is not
// recorded: Emu only sees that the list writes reach the event loop (null is skipped). 0x516950
// and the allocators run natively. F calls the real characterDamageApplyNode on a callee
// world made of the w_ inputs (every 0x52a890 input this node does not fix) and feeds its
// return, its event count and its event back; the event pointer is the callee's new-record
// identity and F uses stubEvent0 for it.
//
// Fixture caps (calc panics past them): who 0..7; event count 0..8, capacity 0..9 (data
// fixtures hold 9 entries); one stack event at most, since 0x52a890 appends at most one
// death record per call (so the list move loop's shift never runs); operator new result 0
// or the fixture.
const (
	cphdChars   = 0x20000000
	cphdPdata   = 0x20015000
	cphdNewbuf  = 0x20015100
	cphdEntries = 9
)

func cphdNames() (in, out []string) {
	in = []string{"who", "dmg", "D", "growthFlag", "specialMode", "partyDamage", "eventCount", "eventCapacity"}
	for i := 0; i < cphdEntries; i++ {
		in = append(in, fmt.Sprintf("pe%d", i))
	}
	for i := 0; i < cphdEntries; i++ {
		in = append(in, fmt.Sprintf("nb%d", i))
	}
	in = append(in, "newRet", "stubRet", "stubEventCount", "stubEvent0")
	cin, _ := cdmNames()
	for _, n := range cin {
		if !cphdFixed[n] {
			in = append(in, "w_"+n)
		}
	}
	out = []string{"partyDamage", "eventCount", "eventCapacity", "eventData"}
	for i := 0; i < cphdEntries; i++ {
		out = append(out, fmt.Sprintf("pe%d", i))
	}
	for i := 0; i < cphdEntries; i++ {
		out = append(out, fmt.Sprintf("nb%d", i))
	}
	out = append(out, "n52a890", "a52a890_who", "a52a890_dmg", "a52a890_a3", "a52a890_a4", "a52a890_a5", "a52a890_a7",
		"n553f10", "a553f10_1", "a553f10_2", "a553f10_3", "a553f10_4")
	return
}

// cphdFixed are the 0x52a890 inputs this node determines itself.
var cphdFixed = map[string]bool{"who": true, "dmg": true, "a3": true, "a4": true, "a5": true, "kill": true, "a7": true,
	"kl0": true, "kl4": true, "kl4c": true, "kl5c": true, "kl60": true, "kl64": true}

var cphdX = func() map[string]int {
	in, _ := cphdNames()
	m := map[string]int{}
	for i, n := range in {
		m[n] = i
	}
	return m
}()

func init() {
	in, out := cphdNames()
	Nodes["005535d0"] = Node{Addr: "005535d0", In: in, Out: out,
		F:   characterPhysicalDamageNode,
		Emu: func(in []int64, r Rand) []int64 { return characterPhysicalDamage(in, r, false) }}
}

func characterPhysicalDamageNode(in []int64, r Rand) []int64 {
	return characterPhysicalDamage(in, r, true)
}

func characterPhysicalDamage(in []int64, r Rand, real bool) []int64 {
	names, outNames := cphdNames()
	if len(in) != len(names) {
		panic("character physical damage: wrong input count")
	}
	g := func(n string) int64 { return in[cphdX[n]] }
	who := g("who")
	if who < 0 || who > 7 {
		panic("character physical damage: member outside the eight fixture records")
	}
	count, capacity := g("eventCount"), g("eventCapacity")
	if count < 0 || count > 8 || capacity < 0 || capacity > cphdEntries {
		panic("character physical damage: party event list outside the fixture (count 0..8, capacity 0..9)")
	}
	if v := g("newRet"); v != 0 && v != cphdNewbuf {
		panic("character physical damage: operator new result outside the fixture")
	}
	if g("dmg") < -2147483648 || g("dmg") > 2147483647 || g("D") < -2147483648 || g("D") > 2147483647 {
		panic("character physical damage: i32 field out of range")
	}
	pe, nb := make([]uint32, cphdEntries), make([]uint32, cphdEntries)
	for i := range pe {
		pe[i], nb[i] = uint32(g(fmt.Sprintf("pe%d", i))), uint32(g(fmt.Sprintf("nb%d", i)))
	}
	partyDmg := uint32(g("partyDamage"))
	dataPtr := int64(cphdPdata)
	var n52, n553 int64
	var a52 [6]int64
	var a553 [4]int64
	finish := func() []int64 {
		x := []int64{int64(partyDmg), count, capacity, dataPtr}
		for _, v := range pe {
			x = append(x, int64(v))
		}
		for _, v := range nb {
			x = append(x, int64(v))
		}
		x = append(x, n52)
		x = append(x, a52[:]...)
		x = append(x, n553)
		x = append(x, a553[:]...)
		if len(x) != len(outNames) {
			panic("character physical damage: output count")
		}
		return x
	}
	// 0x545950
	red := int32(g("dmg"))
	if d := int32(g("D")); d != 0 {
		red = ((100-d)*red + 50) / 100
	}
	if red < 0 {
		red = 0
	}
	if g("growthFlag")&0xff != 0 { // 0x545998: 0x553f10(C, 0x25, 1, 0)
		n553, a553 = 1, [4]int64{cphdChars + who*0x1862, 0x25, 1, 0}
	}
	if red == 0 {
		return finish()
	}
	if g("specialMode")&0xff != 0 { // 0x553649: return ignored, no kill list
		n52, a52 = 1, [6]int64{who, int64(uint32(red)), 0, 1, 1, 1}
		if real {
			cphdCallee(in, who, uint32(red), 0, 1, 1, 0, 1, r)
		}
		return finish()
	}
	n52, a52 = 1, [6]int64{who, int64(uint32(red)), 0, 0, 0, 0}
	ret, evCount, ev0 := uint32(g("stubRet")), g("stubEventCount"), uint32(g("stubEvent0"))
	if real {
		ret, evCount = cphdCallee(in, who, uint32(red), 0, 0, 0, cdmKl, 0, r)
	}
	partyDmg += ret // 0x553680
	if evCount < 0 || evCount > 1 {
		panic("character physical damage: more than one pending death event")
	}
	if evCount == 1 { // 0x553698..0x553735: the list move; one event never shifts.
		if capacity < count+1 {
			if g("newRet") == 0 {
				return finish() // 0x5536fe: old list kept, event dropped
			}
			copy(nb[:count], pe[:count])
			dataPtr, capacity = cphdNewbuf, count+1
			nb[count] = ev0
		} else {
			pe[count] = ev0
		}
		count++
	}
	return finish()
}

// cphdCallee runs the real 0x52a890 for (who, dmg, a3, a4, a5, kill, a7) on a callee world
// whose stack kill list (when kill is set) is empty with capacity 5, and returns its
// return value and the number of pending events.
func cphdCallee(in []int64, who int64, dmg uint32, a3, a4, a5, kill, a7 int64, r Rand) (uint32, int64) {
	cin, cout := cdmNames()
	v := make([]int64, len(cin))
	fixed := map[string]int64{"who": who, "dmg": int64(dmg), "a3": a3, "a4": a4, "a5": a5, "kill": kill, "a7": a7,
		"kl0": 0, "kl4": 0, "kl4c": 0, "kl5c": 0, "kl60": 5, "kl64": cdmKlData}
	for i, n := range cin {
		if f, ok := fixed[n]; ok {
			v[i] = f
		} else {
			v[i] = in[cphdX["w_"+n]]
		}
	}
	out := characterDamageApplyNode(v, r)
	o := map[string]int64{}
	for i, n := range cout {
		o[n] = out[i]
	}
	return uint32(o["ret"]), o["kl5c"]
}
