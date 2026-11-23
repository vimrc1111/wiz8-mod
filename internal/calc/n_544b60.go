package calc

import "fmt"

// ranged_attack_effect_pack 0x544b60: pack the shooter's launch data and hand it
// to the projectile launcher 0x544630 together with the attacker's ARhit.
// Per call: class = i8 at weapon item +0xcc (>= item count -> 0, 0x544b9d); the 12-dword
// block {0, weapon +0x4a, 16 effect-chance bytes of the first item (+0x50, or the
// bytewise sum with the second item when descr+0x74 != -1), level+min(level,15), sum of
// the items' +0x60 bytes, 0,0,0,0} (0x544bd1..0x544ca9); ARhit from playerArhitNode
// (0x541c00, skip flag 0) and the launcher call (0x544cd9..0x544d13).
//
// Replay domain: 0x544630 returns early (stores 1 at battle +0x8c4 and copies the 8-dword
// target record to +0x8c5) only when the target kind is 1 and the class flag
// (table 0x65bde0, stride 0x1e5, +0x154) is 0 (build/wiz8.asm 0x544630 prologue). Every other
// kind or flag builds a missile object (0x4a2d30 -> 0x4a28d0 -> entity system), which the
// emulator cannot run; the case fixes target kind 1 and an all-zero flag table, and the node
// panics outside it.
// The launcher call arguments are read back from the dead stack frame of the original,
// so the node outputs: class, arhit, targetRange (6th launcher argument, the caller's 4th),
// the 12-dword block, launch flag, 8-dword target copy.
const (
	rapItems    = 0x10d // item record stride, global 0x68516c (0x544b76..0x544b86)
	rapSlot     = 12    // inventory slot stride in the shooter record at +0xf5d (0x544c18..0x544c1b)
	rapSlotBase = 0xf5d
	rapLanes    = 16 // 0x544c6a: 16 bytes at item +0x50
)

type rapMem map[[2]int64]byte

func (m rapMem) put(rec, off, size, v int64) {
	for i := int64(0); i < size; i++ {
		m[[2]int64{rec, off + i}] = byte(uint64(v) >> (8 * i))
	}
}

func (m rapMem) get(rec, off, size int64) (v int64) {
	for i := size - 1; i >= 0; i-- {
		v = v<<8 | int64(m[[2]int64{rec, off + i}])
	}
	return v
}

var rapIn = func() []string {
	n := []string{"who", "hand", "targetRange", "slotA", "slotB", "weaponItem", "level", "itemA", "itemB", "itemCount", "weaponClass", "weaponDword"}
	for _, t := range "AB" {
		for k := 0; k < rapLanes; k++ {
			n = append(n, fmt.Sprintf("item%c.b%d", t, k))
		}
		n = append(n, fmt.Sprintf("item%c.t", t))
	}
	for i := 3; i < 8; i++ {
		n = append(n, fmt.Sprintf("targetRecord%d", i))
	}
	return n
}()

func rangedAttackEffectPackNode(in []int64, r Rand) []int64 {
	arh, ok := Nodes["00541c00"]
	if !ok {
		panic("ranged attack pack: node 00541c00 not registered")
	}
	at := map[string]int{}
	names := append(append([]string{}, rapIn...), rapShared(arh.In)...)
	names = append(names, "mode0", "mode1")
	if len(in) != len(names) {
		panic(fmt.Sprintf("ranged attack pack: expected %d inputs", len(names)))
	}
	for i, n := range names {
		at[n] = i
	}
	g := func(n string) int64 { return in[at[n]] }
	if g("targetType") != 1 {
		panic("ranged attack pack: outside replay domain (target kind must be 1)")
	}
	hand := g("hand")
	// 541c00 argument vector: shared names by lookup, own skip flag 0, mode of the chosen hand.
	v := make([]int64, len(arh.In))
	for i, n := range arh.In {
		switch n {
		case "skipDefense":
		case "mode":
			v[i] = g(fmt.Sprintf("mode%d", hand))
		default:
			v[i] = g(n)
		}
	}
	arhit := playerArhitNode(v, r)[0]

	slots := map[int64]int64{} // shooter record words at +0xf5d+slot*12, written in input order
	slots[g("slotA")] = g("itemA")
	if g("slotB") >= 0 {
		slots[g("slotB")] = g("itemB")
	}
	items := rapMem{}
	wi := g("weaponItem")
	items.put(wi, 0xcc, 1, g("weaponClass"))
	items.put(wi, 0x4a, 4, g("weaponDword"))
	for _, t := range "AB" {
		rec := g("item" + string(t))
		for k := 0; k < rapLanes; k++ {
			items.put(rec, 0x50+int64(k), 1, g(fmt.Sprintf("item%c.b%d", t, k)))
		}
		items.put(rec, 0x60, 1, g(fmt.Sprintf("item%c.t", t)))
	}

	cls := uint32(int32(int8(items.get(wi, 0xcc, 1)))) // movsx, then unsigned compare 0x544b90
	if cls >= uint32(g("itemCount")) {
		cls = 0 // 0x544b9d..0x544bae reports DATA ERROR (0x5182e0) and resets
	}
	a := slots[g("slotA")]
	var lanes [rapLanes]byte
	sum := items.get(a, 0x60, 1)
	for k := range lanes {
		lanes[k] = byte(items.get(a, 0x50+int64(k), 1))
	}
	if g("slotB") != -1 { // 0x544c3c
		b := slots[g("slotB")]
		for k := range lanes {
			lanes[k] += byte(items.get(b, 0x50+int64(k), 1))
		}
		sum += items.get(b, 0x60, 1)
	}
	lv := uint32(g("level"))
	capped := lv
	if capped >= 0xf { // 0x544be5..0x544bec
		capped = 0xf
	}
	pack := [12]uint32{1: uint32(items.get(wi, 0x4a, 4)), 6: lv + capped, 7: uint32(sum)}
	for i := 0; i < 4; i++ {
		pack[2+i] = uint32(lanes[4*i]) | uint32(lanes[4*i+1])<<8 | uint32(lanes[4*i+2])<<16 | uint32(lanes[4*i+3])<<24
	}

	out := []int64{int64(cls), int64(int32(arhit)), int64(int32(g("targetRange")))}
	for _, p := range pack {
		out = append(out, int64(p))
	}
	out = append(out, 1, g("targetType"), g("characterIndex"), g("monsterID"))
	for i := 3; i < 8; i++ {
		out = append(out, g(fmt.Sprintf("targetRecord%d", i)))
	}
	return out
}

// 541c00's inputs minus the ones this node derives itself (who, hand, skip flag, mode).
func rapShared(all []string) (r []string) {
	for _, n := range all {
		switch n {
		case "who", "hand", "skipDefense", "mode":
		default:
			r = append(r, n)
		}
	}
	return r
}

func init() {
	arh, ok := Nodes["00541c00"]
	if !ok {
		panic("ranged attack pack: node 00541c00 must register first")
	}
	in := append(append([]string{}, rapIn...), rapShared(arh.In)...)
	in = append(in, "mode0", "mode1")
	out := []string{"class", "arhit", "targetRange"}
	for i := 0; i < 12; i++ {
		out = append(out, fmt.Sprintf("pack%d", i))
	}
	out = append(out, "launchFlag")
	for i := 0; i < 8; i++ {
		out = append(out, fmt.Sprintf("targetCopy%d", i))
	}
	Nodes["00544b60"] = Node{Addr: "00544b60", In: in, Out: out, F: rangedAttackEffectPackNode}
}
