package calc

import (
	"fmt"
	"strings"
)

// shop_buy_transaction 0x55b7e0(merchant, idx, qty, noPay, outP): buy qty units of
// the merchant stock entry idx (build/wiz8.asm:55b7e0..55b9b1).
//   - the entry is the idx-th pointer of the list at merchant+0xa (0x5e2870: a null
//     list, idx >= count or a null slot give 0, and the function returns 0);
//   - loop (acc, qty are bytes): 0x51c020(inst, id, 1, 1, 0) builds an item instance
//     (id and 8 tail bytes come from the callee), then by definition kind +0x66:
//     kind 0 takes one unit, kind 1 takes bl = min(qty-acc, def+0x6b) (0 -> 1) and
//     stores it in tail byte 0, any other kind copies entry byte +0x9 to tail byte 1
//     and takes one. The instance goes through 0x522090: 0x521ef0(inst, 0, 0), else
//     the hand path of 0x5223a0 (without its 0x58aad0 report). acc += bl while acc < qty;
//   - 0x51c020(inst, id, 0, 1, 0) again, price = 0x55b5e0(merchant, inst, 1); the
//     entry is reduced by acc through 0x55ae00; noPay == 0 pays (0x51bf40: gold = 0
//     when price > gold, else gold -= price); outP != 0 receives (entry qty & 0xff) -
//     (acc & 0xff), the first read of the entry byte +0x8;
//   - cleanup: every list entry whose byte +0x10 is 0 is removed from the list
//     (0x5e27c0 shifts the pointers and decrements the count, 0x5e1c10 frees it);
//     after a removal at index i > 0 the scan steps back one, at i == 0 it does not,
//     so the entry that slid into slot 0 is skipped.
//
// 0x521ef0 (pack insertion), the story and voice hooks it triggers (0x522640,
// 0x5227d0), the cursor and the pack are modelled by cerSt of n_5223a0.go, which is
// called as it is; the hand path is shopToHand, a copy of cerSt.toHand without the
// 0x58aad0 call, with the same caps: 8 quest-flag ids from its set, event log <= 24,
// NPC list <= 2, debug-log globals fixed 0, item ids 0..639, pack scan <= 8 entries.
//
// D4 address stubs in emu mode, results from case inputs: 0x51c020 (instance id and tail),
// 0x55b5e0 (price), 0x518150 (character slot, shared with n_5223a0.go) and 0x51f900;
// 0x51b5c0 (display name, registered 排除) is counted. F ignores priceRet, instId,
// inst4, inst8 and r518150 and calls the real itemStackChargeInitNode (both 0x51c020
// calls, dice rolled with diceNode), shopTransactionPriceNode, randomPartyMemberNode
// (inside shopRemove, before the voice hook) and itemStackMergeNode, reading the F-only
// inputs "f.*" (sbtFNames). F's extra caps (panics): one callee item record (the
// callee id is remapped to 0), assert-only paths of 0x51c020 (stack kind >= 5, dice
// maximum >= 256). The branches cov-exempt in this node (0x55b879, 0x55b8ab, 0x55b91c) panic.
//
// Fixture caps (calc panics past them): list count and data slots 4, a list or data
// pointer outside the fixture addresses, entry ids 0..639, 32 results of 0x51f900,
// negative idx or count. Not compared: the stack addresses passed to 0x520d10 (arg 1)
// and 0x51f900 (arg 2).
const (
	sbtN      = 4 // list entries and data slots in the fixture
	sbtSeq    = 32
	sbtList   = 0x20090100
	sbtEnt    = 0x20090400
	sbtEntStr = 0x40
	sbtOutP   = 0x20091100
	sbtInst   = 0x10007000 // stack address of the item instance, never compared
)

var sbtShared = func() []string {
	var n []string
	for _, s := range cerNames(true) {
		if strings.HasPrefix(s, "t1") || strings.HasPrefix(s, "t2") || s == "argp" || s == "ci" || s == "a49" {
			continue
		}
		var k int
		if strings.HasPrefix(s, "r51f") {
			if _, err := fmt.Sscanf(s, "r51f%d", &k); err == nil && k >= sbtSeq {
				continue
			}
		}
		n = append(n, s)
	}
	return n
}()

var sbtOutShared = func() []string {
	var n []string
	skip := map[string]bool{"n58aad0": true, "n58aad0a1": true, "n58aad0a2": true, "n58aad0a3": true, "n520d10a1": true, "n51f900a2": true}
	for _, s := range cerNames(false) {
		if strings.HasPrefix(s, "t1") || strings.HasPrefix(s, "t2") || skip[s] {
			continue
		}
		n = append(n, s)
	}
	return n
}()

func sbtInNames() []string {
	n := []string{"idx", "qty", "noPay", "outP", "listP", "count"}
	for j := 0; j < sbtN; j++ {
		n = append(n, fmt.Sprintf("p%d", j))
	}
	for j := 0; j < sbtN; j++ {
		for _, f := range []string{"id", "q", "b9", "u", "kind", "max", "d41", "ih"} {
			n = append(n, fmt.Sprintf("L%d%s", j, f))
		}
	}
	n = append(n, "priceRet", "instId", "inst4", "inst8", "gold", "outInit")
	n = append(n, sbtShared...)
	return append(n, sbtFNames()...)
}

// sbtFNames are the F-only inputs (the case parks them in a scratch object the emulator never reads):
// per list entry the dice (z, n, s) of 0x51c020, the definition type (+0x3e) and base value (+0x86) of
// 0x55b5e0; the merchant's two rates; the party's active/life/status/skill; and the presence, flag,
// level and key bytes of the 8 slots that 0x518150 scans.
func sbtFNames() []string {
	var n []string
	for j := 0; j < sbtN; j++ {
		for _, f := range []string{"z", "n", "s", "type", "base"} {
			n = append(n, fmt.Sprintf("f.L%d%s", j, f))
		}
	}
	n = append(n, "f.rateLow", "f.rateHigh")
	for k := 0; k < 8; k++ {
		for _, f := range []string{"Active", "Life", "Status", "Skill"} {
			n = append(n, fmt.Sprintf("f.m%d%s", k, f))
		}
	}
	for _, p := range []string{"pres", "flag", "lvl", "key"} {
		for k := 0; k < 8; k++ {
			n = append(n, fmt.Sprintf("f.%s%d", p, k))
		}
	}
	return n
}

// sbtInit is one real 0x51c020(inst, id, maxRoll, 1, 0) on the item record of entry j (the callee's
// fixture has one record, so the id is remapped to 0 and the caller keeps the real id); it returns the
// instance dwords at +4 and +8. The dice are rolled here, like 0x517970 does for a real query.
func sbtInit(v map[string]int64, j int, e *sbtEntry, maxRoll int64, r Rand) (uint32, uint32) {
	at := map[string]int{}
	for i, n := range iscIn {
		at[n] = i
	}
	in := make([]int64, len(iscIn))
	f := func(s string) int64 { return v[fmt.Sprintf("f.L%d%s", j, s)] }
	in[at["p"]], in[at["maxRoll"]], in[at["forceFlag6"]], in[at["itemCount"]] = iscInst, maxRoll, 1, 1
	in[at["i0.kind"]], in[at["i0.z"]], in[at["i0.n"]], in[at["i0.s"]] = int64(e.kind), f("z"), f("n"), f("s")
	if maxRoll == 0 && e.kind != 0 {
		in[at["diceRet"]] = diceNode([]int64{f("z"), f("n"), f("s")}, r)[0]
	}
	o := itemStackChargeInitNode(in, r)
	return uint32(o[1]), uint32(o[2])
}

// shopRemove is cerSt.remove (n_5223a0.go) with the one real call it cannot make: before the voice hook,
// 0x518150(0, 0, -1, -1) picks the party slot (F only; Emu keeps r518150).
func (s *cerSt) shopRemove(rec *cerRec, ptr uint32, u int, r Rand, v map[string]int64, real bool) bool {
	if rec.id < 0 || rec.id >= cerIDs {
		panic("equipment removal: item id outside the definition table cap (or an empty second-set slot, 0x51fe30 assert)")
	}
	merged, idx := false, 0
	s.flagLocal = 0
	if s.d66[u] == 1 {
		for i := uint32(0); i < s.pkN; i++ {
			if s.merge(i, rec, ptr, r) {
				merged, idx = true, int(i)
				break
			}
		}
	}
	switch {
	case merged:
		s.call(8)
	default:
		if s.flagLocal != 0 {
			s.call(8)
		}
		if s.pkN >= cerPackCap {
			return false
		}
		old := s.pk
		for i := 0; i < int(s.pkN) && i+1 < cerPK; i++ {
			s.pk[i+1] = old[i]
		}
		s.pk[0] = cerRec{-1, 0}
		s.move(&s.pk[0], cerPackPtr, rec, ptr, u)
		s.pkN++
		s.call(8)
		idx = 0
	}
	if s.mode == 6 {
		s.m69fc |= 0x7fc00000
	}
	if idx >= cerPK {
		panic("equipment removal: merged pack entry past the observed pool")
	}
	pk := &s.pk[idx]
	s.story(pk)
	if real && s.mode != 5 {
		in := []int64{0, 0, -1, -1}
		for _, p := range []string{"pres", "flag", "lvl", "key"} {
			for k := 0; k < 8; k++ {
				in = append(in, v[fmt.Sprintf("f.%s%d", p, k)])
			}
		}
		s.r518150 = uint32(int32(randomPartyMemberNode(in, r)[0]))
	}
	s.voice(pk)
	return true
}

func sbtOutNames() []string {
	n := []string{"al", "gold", "outV", "count"}
	for j := 0; j < sbtN; j++ {
		n = append(n, fmt.Sprintf("p%d", j))
	}
	for j := 0; j < sbtN; j++ {
		n = append(n, fmt.Sprintf("L%dq", j), fmt.Sprintf("L%du", j))
	}
	n = append(n, "n51c020", "n51c020a2", "n51c020a3", "n51c020a4", "n51c020a5", "n55b5e0", "n55b5e0a1", "n55b5e0a3", "n51b5c0", "n51b5c0a2")
	return append(n, sbtOutShared...)
}

func init() {
	Nodes["0055b7e0"] = Node{Addr: "0055b7e0", In: sbtInNames(), Out: sbtOutNames(),
		F:   shopBuyTransactionNode,
		Emu: func(in []int64, r Rand) []int64 { return sbtRun(in, r, false) }}
}

func shopBuyTransactionNode(in []int64, r Rand) []int64 { return sbtRun(in, r, true) }

type sbtEntry struct {
	id        int32
	q, b9, u  uint8
	kind, max uint8
	d41       uint8
	ih        uint32
}

// shopToHand is cerSt.toHand (n_5223a0.go) without the 0x58aad0 report: the hand path of 0x522090.
func (s *cerSt) shopToHand(rec *cerRec, ptr uint32, u int) {
	bl := s.hMark != 0
	saved := s.hand
	s.m0, s.m4, s.m5 = 0xffffffff, 0xff, 0xffff
	s.clearHand()
	s.move(&s.hand, cerHandPtr, rec, ptr, u)
	if s.d41[u]&2 == 0 {
		s.call(0, 0)
	} else {
		s.call(1, s.s13bc, 0, 1, 0)
	}
	if bl {
		s.hand = saved
	}
}

func sbtRun(in []int64, r Rand, real bool) []int64 {
	if cerExeErr != nil {
		panic(cerExeErr)
	}
	names := sbtInNames()
	if len(in) != len(names) {
		panic("shop buy: wrong input count")
	}
	v := make(map[string]int64, len(names))
	for i, n := range names {
		v[n] = in[i]
	}
	listP, count, idx := v["listP"], v["count"], v["idx"]
	if listP != 0 && listP != sbtList {
		panic("shop buy: list pointer outside the fixture")
	}
	if count < 0 || count > sbtN {
		panic("shop buy: list count above the fixture cap")
	}
	if idx < 0 {
		panic("shop buy: negative list index")
	}
	var ent [sbtN]sbtEntry
	var data [sbtN]int // entry index, -1 = null
	for j := 0; j < sbtN; j++ {
		f := func(s string) int64 { return v[fmt.Sprintf("L%d%s", j, s)] }
		if f("id") < 0 || f("id") >= cerIDs {
			panic("shop buy: entry id outside the definition table cap")
		}
		ent[j] = sbtEntry{id: int32(f("id")), q: uint8(f("q")), b9: uint8(f("b9")), u: uint8(f("u")),
			kind: uint8(f("kind")), max: uint8(f("max")), d41: uint8(f("d41")), ih: uint32(f("ih"))}
		p := v[fmt.Sprintf("p%d", j)]
		switch {
		case p == 0:
			data[j] = -1
		case p >= sbtEnt && p < sbtEnt+sbtN*sbtEntStr && (p-sbtEnt)%sbtEntStr == 0:
			data[j] = int((p - sbtEnt) / sbtEntStr)
		default:
			panic("shop buy: list slot pointer outside the fixture")
		}
	}
	outP := v["outP"]
	if outP != 0 && outP != sbtOutP {
		panic("shop buy: output pointer outside the fixture")
	}
	gold := uint32(v["gold"])
	var o51c, o55b, o51b [6]uint32
	var n51c, n55b, n51b uint32
	o := map[string]int64{}
	cnt := int(count)
	// 0x5e2870
	get := func(i int64) int {
		if listP == 0 || i >= int64(cnt) {
			return -1
		}
		return data[i]
	}
	pointer := func(j int) int64 {
		if j < 0 {
			return 0
		}
		return int64(sbtEnt + sbtEntStr*j)
	}
	build := func(s *cerSt, outV int64, al int64) []int64 {
		so := s.out()
		on := cerNames(false)
		for i, n := range on {
			o[n] = so[i]
		}
		o["al"], o["gold"], o["outV"], o["count"] = al, int64(gold), outV, int64(cnt)
		for j := 0; j < sbtN; j++ {
			o[fmt.Sprintf("p%d", j)] = pointer(data[j])
			o[fmt.Sprintf("L%dq", j)], o[fmt.Sprintf("L%du", j)] = int64(ent[j].q), int64(ent[j].u)
		}
		o["n51c020"], o["n51c020a2"], o["n51c020a3"], o["n51c020a4"], o["n51c020a5"] = int64(n51c), int64(o51c[2]), int64(o51c[3]), int64(o51c[4]), int64(o51c[5])
		o["n55b5e0"], o["n55b5e0a1"], o["n55b5e0a3"] = int64(n55b), int64(o55b[1]), int64(o55b[3])
		o["n51b5c0"], o["n51b5c0a2"] = int64(n51b), int64(o51b[2])
		res := make([]int64, 0, len(sbtOutNames()))
		for _, n := range sbtOutNames() {
			x, ok := o[n]
			if !ok {
				panic("shop buy: unset output " + n)
			}
			res = append(res, x)
		}
		return res
	}

	// the character-table state shared with n_5223a0.go, as its input vector
	vec := make([]int64, cerInLen)
	pos := map[string]int{}
	for i, n := range cerNames(true) {
		pos[n] = i
	}
	for _, n := range sbtShared {
		vec[pos[n]] = v[n]
	}
	vec[pos["argp"]] = cerChar
	s := &cerSt{real: real}

	j0 := get(idx)
	if j0 < 0 { // 0x55b807: no entry
		cerParse(vec, s)
		return build(s, int64(uint32(v["outInit"])), 0)
	}
	e := &ent[j0]
	for k := int64(0); k < v["pkN"] && k < cerPK; k++ { // the definition byte +0x6b is one cell for both views
		if v[fmt.Sprintf("pk%did", k)] == int64(e.id) && v[fmt.Sprintf("pk%dd6b", k)] != int64(e.max) {
			panic("shop buy: pack slot and list entry alias one definition with different +0x6b")
		}
	}
	vec[pos["t1d410"]], vec[pos["t1d660"]], vec[pos["t1ih0"]] = int64(e.d41), int64(e.kind), int64(e.ih)
	cerParse(vec, s)
	if !real && int32(v["instId"]) != e.id {
		panic("shop buy: 0x51c020 returns the id it was given")
	}
	want := uint8(v["qty"])
	qtyE := e.q
	var acc uint8
	for {
		n51c++
		o51c = [6]uint32{0, 0, uint32(e.id), 1, 1, 0}
		i4, i8 := uint32(v["inst4"]), uint32(v["inst8"])
		if real {
			i4, i8 = sbtInit(v, j0, e, 1, r)
		}
		rec := cerRec{e.id, cerTail(i4, i8)}
		var bl uint8
		switch e.kind {
		case 0:
			bl = 1
		case 1:
			bl = want - acc
			if bl > e.max {
				bl = e.max
			}
			if bl == 0 {
				bl = 1
			}
			rec.tail = rec.tail&^0xff | uint64(bl)
		default:
			rec.tail = rec.tail&^(0xff<<8) | uint64(e.b9)<<8
			bl = 1
		}
		n51b++ // 0x521ef0 starts with 0x51b5c0(inst, 1)
		o51b = [6]uint32{0, 0, 1, 0, 0, 0}
		if !s.shopRemove(&rec, sbtInst, 0, r, v, real) {
			s.shopToHand(&rec, sbtInst, 0)
			if s.hMark == 0 { // 0x55b879: the hand mark is always set after the hand path
				panic("shop buy: hand mark clear after the hand path")
			}
		}
		acc += bl
		if acc >= want {
			break
		}
	}
	if s.n51f > sbtSeq {
		panic("shop buy: 0x51f900 result budget exhausted")
	}
	if acc == 0 { // 0x55b8ab: cov-exempt, acc >= 1 after the first pass
		panic("shop buy: nothing bought after a pass")
	}
	n51c++
	o51c = [6]uint32{0, 0, uint32(e.id), 0, 1, 0}
	n55b++
	o55b = [6]uint32{0, 0x20090000, 0, 1, 0, 0}
	price := uint32(v["priceRet"])
	if real { // 0x51c020(inst, id, 0, 1, 0) then 0x55b5e0(merchant, inst, 1)
		i4, _ := sbtInit(v, j0, e, 0, r)
		pin := []int64{int64(e.id), int64(i4 & 0xff), int64(i4 >> 16 & 0xff), v[fmt.Sprintf("f.L%dtype", j0)], int64(e.kind),
			v[fmt.Sprintf("f.L%dbase", j0)], v["f.rateLow"], v["f.rateHigh"], 1}
		for k := 0; k < 8; k++ {
			for _, f := range []string{"Active", "Life", "Status", "Skill"} {
				pin = append(pin, v[fmt.Sprintf("f.m%d%s", k, f)])
			}
		}
		price = uint32(shopTransactionPriceNode(pin, r)[0])
	}
	ro := shopStockRemoveNode([]int64{1, idx, count, 0, int64(e.kind), int64(e.q), int64(e.u), int64(acc)}, r)
	if ro[0] == 0 { // 0x55b91c: cov-exempt, removing acc >= 1 from a found entry cannot fail
		panic("shop buy: 0x55ae00 failed on a found entry")
	}
	e.q, e.u = uint8(ro[1]), uint8(ro[2])
	if uint8(v["noPay"]) == 0 {
		if price > gold {
			gold = 0
		} else {
			gold -= price
		}
	}
	outV := int64(uint32(v["outInit"]))
	if outP != 0 {
		outV = int64(uint32(qtyE) - uint32(acc))
	}
	// cleanup: drop every entry whose byte +0x10 is 0
	for i, n := 0, cnt; i < n; i++ {
		jj := get(int64(i))
		if jj < 0 || ent[jj].u != 0 {
			continue
		}
		for k := i; k < cnt-1; k++ {
			data[k] = data[k+1]
		}
		cnt--
		n = cnt
		if i > 0 {
			i--
		}
	}
	return build(s, outV, 1)
}
