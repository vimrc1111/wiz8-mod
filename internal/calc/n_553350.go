package calc

import "fmt"

// periodicCharacterPhysicalDamageNode is 0x553350 periodic_character_physical_damage
// (who, unused, &power) (build/wiz8.asm:460779..). A power dword of 0 returns at once.
// Otherwise the dice record {Z i16, N u8, S u8} at *0x65be1c + 0x2f76 gets
// N' = low byte of int8(power) * int8(N) (0x5533ea), is rolled by 0x517970 (diceNode,
// i32 result), and 0x545950 reduces it by the character's D = i32 at C+0xed9
// (C = *0x685174 + who*0x1862): D = 0 keeps the roll, else trunc(((100-D)*roll+50)/100)
// in wrapping i32; a negative result is 0 (0x545988..0x54598c). A nonzero byte C+0x71b
// calls 0x553f10(C, 0x25, 1, 0) first (0x545998). A reduced damage of 0 ends the node.
// Else 0x52a890(who, dmg, 0, flag, flag, &killList, 0) runs with flag = byte 0x68510c and
// the stack kill list (zeroed record, 0x517... list ctor 0x516950 at +0x58: count +0x5c,
// capacity +0x60, data +0x64); kill[0] is added to *0x6836a8+0xa1a, then each pending
// death event is appended to the party list (count +0xa02, capacity +0xa06, data +0xa0a;
// growth takes count+1 entries from operator new, 0 keeps the old list and drops the event).
//
// Callees. 0x553f10 and 0x52a890 are D4 formula callees address-stubbed in Emu: the
// caller ignores both returns (ret 0x5a5a5a5a in a scratch copy of the case gave
// diff=0). Emu records 0x553f10's count and arguments (its writes are never read back
// by this node) and 0x52a890's count and arguments, and the stub writes kill[0],
// the event count and the event list data from case inputs. 0x516950, 0x517970 and the
// allocators run natively. F calls the real characterDamageApply on a callee world made
// of the w_ inputs (every 0x52a890 input this node does not fix) and feeds its kill[0],
// its event count and its event back; the event pointer is the callee's new-record
// identity and F uses stubEvent0 for it.
//
// Fixture caps (calc panics past them): who 0..7; event count 0..8, capacity 0..9
// (data fixtures hold 9 entries); one stack event at most, since 0x52a890 appends at
// most one death record per call (so the list move loop's shift never runs); operator
// new result 0 or the fixture.
const (
	pcpdChars   = 0x20000000
	pcpdPdata   = 0x20015000
	pcpdNewbuf  = 0x20015100
	pcpdEntries = 9
)

func pcpdNames() (in, out []string) {
	in = []string{"who", "power", "Z", "N", "S", "D", "growthFlag", "specialMode", "partyDamage", "eventCount", "eventCapacity"}
	for i := 0; i < pcpdEntries; i++ {
		in = append(in, fmt.Sprintf("pe%d", i))
	}
	for i := 0; i < pcpdEntries; i++ {
		in = append(in, fmt.Sprintf("nb%d", i))
	}
	in = append(in, "newRet", "stubKillDamage", "stubEventCount", "stubEvent0")
	cin, _ := cdmNames()
	for _, n := range cin {
		if !pcpdFixed[n] {
			in = append(in, "w_"+n)
		}
	}
	out = []string{"partyDamage", "eventCount", "eventCapacity", "eventData"}
	for i := 0; i < pcpdEntries; i++ {
		out = append(out, fmt.Sprintf("pe%d", i))
	}
	for i := 0; i < pcpdEntries; i++ {
		out = append(out, fmt.Sprintf("nb%d", i))
	}
	out = append(out, "n52a890", "a52a890_who", "a52a890_dmg", "a52a890_a3", "a52a890_a4", "a52a890_a5", "a52a890_a7",
		"n553f10", "a553f10_1", "a553f10_2", "a553f10_3", "a553f10_4")
	return
}

// pcpdFixed are the 0x52a890 inputs this node determines itself.
var pcpdFixed = map[string]bool{"who": true, "dmg": true, "a3": true, "a4": true, "a5": true, "kill": true, "a7": true,
	"kl0": true, "kl4": true, "kl4c": true, "kl5c": true, "kl60": true, "kl64": true}

var pcpdX = func() map[string]int {
	in, _ := pcpdNames()
	m := map[string]int{}
	for i, n := range in {
		m[n] = i
	}
	return m
}()

func init() {
	in, out := pcpdNames()
	Nodes["00553350"] = Node{Addr: "00553350", In: in, Out: out,
		F:   periodicCharacterPhysicalDamageNode,
		Emu: func(in []int64, r Rand) []int64 { return periodicCharacterPhysicalDamage(in, r, false) }}
}

func periodicCharacterPhysicalDamageNode(in []int64, r Rand) []int64 {
	return periodicCharacterPhysicalDamage(in, r, true)
}

func periodicCharacterPhysicalDamage(in []int64, r Rand, real bool) []int64 {
	names, outNames := pcpdNames()
	if len(in) != len(names) {
		panic("periodic character physical damage: wrong input count")
	}
	g := func(n string) int64 { return in[pcpdX[n]] }
	who := g("who")
	if who < 0 || who > 7 {
		panic("periodic character physical damage: member outside the eight fixture records")
	}
	count, capacity := g("eventCount"), g("eventCapacity")
	if count < 0 || count > 8 || capacity < 0 || capacity > pcpdEntries {
		panic("periodic character physical damage: party event list outside the fixture (count 0..8, capacity 0..9)")
	}
	if v := g("newRet"); v != 0 && v != pcpdNewbuf {
		panic("periodic character physical damage: operator new result outside the fixture")
	}
	if g("power") < -2147483648 || g("power") > 2147483647 || g("D") < -2147483648 || g("D") > 2147483647 {
		panic("periodic character physical damage: i32 field out of range")
	}
	pe, nb := make([]uint32, pcpdEntries), make([]uint32, pcpdEntries)
	for i := range pe {
		pe[i], nb[i] = uint32(g(fmt.Sprintf("pe%d", i))), uint32(g(fmt.Sprintf("nb%d", i)))
	}
	partyDmg := uint32(g("partyDamage"))
	dataPtr := int64(pcpdPdata)
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
			panic("periodic character physical damage: output count")
		}
		return x
	}
	p := int32(g("power"))
	if p == 0 { // 0x55338e
		return finish()
	}
	// 0x5533ea: imul byte, the low byte of the signed product becomes the dice count.
	nDice := int64(uint8(int8(p) * int8(g("N"))))
	roll := int32(diceNode([]int64{g("Z"), nDice, g("S")}, r)[0])
	// 0x545950
	red := roll
	if d := int32(g("D")); d != 0 {
		red = ((100-d)*roll + 50) / 100
	}
	if red < 0 {
		red = 0
	}
	if g("growthFlag")&0xff != 0 { // 0x545998: 0x553f10(C, 0x25, 1, 0)
		n553, a553 = 1, [4]int64{pcpdChars + who*0x1862, 0x25, 1, 0}
	}
	if red == 0 {
		return finish()
	}
	flag := g("specialMode") & 0xff
	n52, a52 = 1, [6]int64{who, int64(uint32(red)), 0, flag, flag, 0}
	killDmg, evCount, ev0 := uint32(g("stubKillDamage")), g("stubEventCount"), uint32(g("stubEvent0"))
	if real {
		killDmg, evCount, ev0 = pcpdCallee(in, who, uint32(red), flag, ev0, r)
	}
	partyDmg += killDmg // 0x55344c
	if evCount < 0 || evCount > 1 {
		panic("periodic character physical damage: more than one pending death event")
	}
	if evCount == 1 { // 0x553466..0x553500: the list move; one event never shifts.
		if capacity < count+1 {
			if g("newRet") == 0 {
				return finish() // 0x5534c9: old list kept, event dropped
			}
			copy(nb[:count], pe[:count])
			dataPtr, capacity = pcpdNewbuf, count+1
			nb[count] = ev0
		} else {
			pe[count] = ev0
		}
		count++
	}
	return finish()
}

// pcpdCallee runs the real 0x52a890 for (who, dmg, 0, flag, flag, kill, 0) on a callee
// world whose stack kill list is empty with capacity 5, and returns kill[0], the number
// of pending events and the event pointer (stubEvent0 stands for the new record).
func pcpdCallee(in []int64, who int64, dmg uint32, flag int64, ev0 uint32, r Rand) (uint32, int64, uint32) {
	cin, cout := cdmNames()
	v := make([]int64, len(cin))
	fixed := map[string]int64{"who": who, "dmg": int64(dmg), "a3": 0, "a4": flag, "a5": flag, "kill": cdmKl, "a7": 0,
		"kl0": 0, "kl4": 0, "kl4c": 0, "kl5c": 0, "kl60": 5, "kl64": cdmKlData}
	for i, n := range cin {
		if f, ok := fixed[n]; ok {
			v[i] = f
		} else {
			v[i] = in[pcpdX["w_"+n]]
		}
	}
	out := characterDamageApplyNode(v, r)
	o := map[string]int64{}
	for i, n := range cout {
		o[n] = out[i]
	}
	return uint32(o["kl0"]), o["kl5c"], ev0
}
