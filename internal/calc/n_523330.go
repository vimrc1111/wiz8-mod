package calc

import "fmt"

// character_condition_remove 0x523330: clear state slot `cond` (character
// +0xa01+4*cond) of party member `who`, refresh the main state, reset the
// matching secondary fields and tell the UI (build/wiz8.asm:394989..).
//
// Callers (17 sites, build/wiz8.asm calls of 0x523330) pass `who` as a party
// index 0..7 (loops end at presence offset 0x830 = 8*0x106) and `cond` either as
// a constant 0..0x13 or as the loop index of the 20 state slots (limits 0x12/0xa
// at 0x56c75a, 0x52ac5a, 0x54e092), so who >= 8 or cond >= 20 panics here.
//
// Callees: 0x5237e0 runs as the reused characterPrimaryConditionNode; 0x4e79a0,
// 0x50e650, 0x52c480, 0x536570, 0x538e00 (inside 0x538d10), 0x52f790 and 0x58aad0
// (inside the 0x590950 notice) are stubbed in the emulator case, so their
// arguments and call counts are part of the output. 0x53a930 (portrait/text
// refresh after the tail test) and the notice text itself have no declared
// output here. F additionally rebuilds the resource cells through ccrWorld;
// Emu preserves the case's 50e650 address stub. The six existing F callers
// continue to use characterConditionRemoveNode and its original interface.
const (
	ccrChars     = 8
	ccrSlots     = 20
	ccrListA     = 12 // character +0xf5d, stride 12
	ccrListB     = 8  // character +0x1029, stride 12
	ccrBackCap   = 6  // fixture cap on backpack records (0x685191, stride 12); more panics
	ccrForms     = 3 * pcFormLen
	ccrCharBase  = 0
	ccrWho       = 1
	ccrCond      = 2
	ccrNotify    = 3
	ccrCombat    = 4
	ccrG7500     = 5
	ccrPres      = 6
	ccrWhoPres   = 14
	ccrSlot      = 15
	ccrB01       = 35
	ccrB11       = 36
	ccrB09       = 37
	ccrB15       = 38
	ccrB21       = 39
	ccrG686a70   = 40
	ccrG6840bb   = 41
	ccrCurF      = 42
	ccrCurID     = 43
	ccrLists     = 44
	ccrBpn       = ccrLists + ccrChars*(ccrListA+ccrListB)
	ccrBp        = ccrBpn + 1
	ccrF1ce      = ccrBp + ccrBackCap
	ccrRec98     = ccrF1ce + 1
	ccrP4d       = ccrRec98 + 1
	ccrP1d       = ccrP4d + 8
	ccrRet52     = ccrP1d + 8
	ccrRet4e7    = ccrRet52 + 1
	ccrPC        = ccrRet4e7 + 1 // interfaceMode.. of the 0x5237e0 node
	ccrLen       = ccrPC + 5 + 2*pcMembers + ccrForms
	ccrCharSize  = 0x1862
	ccrItemWard  = 0x243 // item searched for condition 0xb (0x523444)
	ccrItemKeep  = 0x239 // item the holder must carry (0x52348b)
	ccrNoticeFmt = 0x64bf5c
)

func init() {
	var in, out []string
	add := func(l *[]string, f string, a ...any) { *l = append(*l, fmt.Sprintf(f, a...)) }
	for _, n := range []string{"charBase", "who", "cond", "notify", "combat", "g7500"} {
		in = append(in, n)
	}
	for j := 0; j < ccrChars; j++ {
		add(&in, "pres%d", j)
	}
	in = append(in, "whoPres")
	for k := 0; k < ccrSlots; k++ {
		add(&in, "s%d", k)
	}
	in = append(in, "b01", "b11", "b09", "b15", "b21", "g686a70", "g6840bb", "curF", "curId")
	for j := 0; j < ccrChars; j++ {
		for k := 0; k < ccrListA; k++ {
			add(&in, "it%d_%d", j, k)
		}
		for k := 0; k < ccrListB; k++ {
			add(&in, "eq%d_%d", j, k)
		}
	}
	in = append(in, "bpn")
	for k := 0; k < ccrBackCap; k++ {
		add(&in, "bp%d", k)
	}
	in = append(in, "f1ce", "rec98")
	for k := 0; k < 8; k++ {
		add(&in, "p4d%d", k)
	}
	for k := 0; k < 8; k++ {
		add(&in, "p1d%d", k)
	}
	in = append(in, "ret52", "ret4e7", "interfaceMode", "battle", "battleMask", "screenFlags", "portraitGate")
	for c := 0; c < pcMembers; c++ {
		add(&in, "expr%d", c)
	}
	for c := 0; c < pcMembers; c++ {
		add(&in, "flag99_%d", c)
	}
	for _, t := range "ABC" {
		for i := 0; i < pcFormLen; i++ {
			add(&in, "%c.f%d", t, i)
		}
	}
	for k := 0; k < ccrSlots; k++ {
		add(&out, "s%d", k)
	}
	out = append(out, "b09", "b15", "b21", "g6840bb")
	for k := 0; k < 8; k++ {
		add(&out, "p1d%d", k)
	}
	for c := 0; c < pcMembers; c++ {
		add(&out, "expr%d", c)
	}
	for c := 0; c < pcMembers; c++ {
		add(&out, "flag99_%d", c)
	}
	out = append(out, "battleMask", "screenFlags")
	for _, t := range "ABC" {
		for i := 0; i < pcFormLen; i++ {
			add(&out, "%c.f%d", t, i)
		}
	}
	out = append(out, "b01",
		"r4e79a0_1", "r4e79a0_2", "r4e79a0_3", "r4e79a0_4", "n4e79a0",
		"r50e650_1", "n50e650", "r52f790_1", "r52f790_2", "n52f790", "n52c480",
		"r536570_1", "r536570_2", "r536570_3", "n536570", "r538e00_2", "n538e00",
		"r58aad0_1", "r58aad0_2", "r58aad0_3", "n58aad0")
	Nodes["00523330"] = Node{Addr: "00523330", In: in, Out: out, F: characterConditionRemoveNode, Emu: characterConditionRemoveEmu}
}

func characterConditionRemoveNode(in []int64, r Rand) []int64 {
	return ccrRun(in, r, newCCRWorld(in))
}

func characterConditionRemoveEmu(in []int64, r Rand) []int64 {
	return ccrRun(in, r, nil)
}

func ccrRun(in []int64, r Rand, world *ccrWorld) []int64 {
	if npcRecruitExeErr != nil {
		panic(npcRecruitExeErr)
	}
	if len(in) != ccrLen {
		panic("characterConditionRemove: wrong input count")
	}
	base, who, cond := in[ccrCharBase], int(in[ccrWho]), int(in[ccrCond])
	if who < 0 || who >= ccrChars || cond < 0 || cond >= ccrSlots {
		panic("characterConditionRemove: who/cond out of range") // not a value any caller passes
	}
	notify, combat := in[ccrNotify] != 0, in[ccrCombat] != 0
	var present [ccrChars]bool
	for j := range present {
		present[j] = in[ccrPres+j] != 0
	}
	present[who] = in[ccrWhoPres] != 0 // same presence byte, written last
	slot := make([]int64, ccrSlots)
	copy(slot, in[ccrSlot:ccrSlot+ccrSlots])
	b01, b09, b15, b21 := in[ccrB01], in[ccrB09], in[ccrB15], in[ccrB21]
	g6840bb := in[ccrG6840bb]
	p1d := append([]int64(nil), in[ccrP1d:ccrP1d+8]...)
	pc := append([]int64(nil), in[ccrPC:]...) // 0x5237e0 inputs after combat
	var res []int64                           // 0x5237e0 output when it ran
	var r4e, r536, r58 [4]int64
	var n4e, n50, n52f, n52c, n536, n538, n58 int64
	var a50, a52f1, a52f2, a538 int64

	finish := func() []int64 {
		o := append([]int64(nil), slot...)
		o = append(o, b09, b15, b21, g6840bb)
		o = append(o, p1d...)
		if res != nil {
			o = append(o, res[1:]...)
			b01 = res[0]
		} else {
			o = append(o, in[ccrPC+5:ccrPC+5+2*pcMembers]...)
			o = append(o, in[ccrPC+2], in[ccrPC+3])
			o = append(o, in[ccrPC+5+2*pcMembers:]...)
		}
		o = append(o, b01)
		o = append(o, r4e[0], r4e[1], r4e[2], r4e[3], n4e, a50, n50, a52f1, a52f2, n52f, n52c)
		o = append(o, r536[0], r536[1], r536[2], n536, a538, n538, r58[0], r58[1], r58[2], n58)
		return o
	}

	if uint32(slot[cond]) == 0 {
		if in[ccrG7500] != 0 {
			return finish() // 0x523364: nothing to remove
		}
		panic("characterConditionRemove: assert, slot is empty") // 0x5233c1
	}
	if !present[who] {
		panic("characterConditionRemove: assert, member absent") // 0x523393
	}
	b11, oldB01 := uint32(in[ccrB11]), uint32(b01)
	bVar7 := b11 > 0 && oldB01 < 0xd // 0x523400..0x52341f

	// Item search of exe 0x521060 (mode 2, no match pointer) over cursor, present
	// members (list A then B) and backpack.
	var lists [ccrChars][ccrListA + ccrListB]uint32
	for j := 0; j < ccrChars; j++ {
		for k := 0; k < ccrListA+ccrListB; k++ {
			lists[j][k] = uint32(in[ccrLists+j*(ccrListA+ccrListB)+k])
		}
	}
	bpn := uint32(in[ccrBpn])
	if bpn > ccrBackCap {
		panic(fmt.Sprintf("characterConditionRemove: backpack count %d exceeds modelled cap %d", bpn, ccrBackCap))
	}
	cursor := func(id uint32) bool { return in[ccrCurF]&0xff != 0 && uint32(in[ccrCurID]) == id }
	inList := func(j int, id uint32, withB bool) bool {
		n := ccrListA
		if withB {
			n += ccrListB
		}
		for _, v := range lists[j][:n] {
			if v == id {
				return true
			}
		}
		return false
	}
	// returns where: -2 not found, -1 cursor, 0..7 member, 8 backpack
	locate := func(id uint32) int {
		if cursor(id) {
			return -1
		}
		for j := 0; j < ccrChars; j++ {
			if present[j] && inList(j, id, true) {
				return j
			}
		}
		for k := uint32(0); k < bpn; k++ {
			if uint32(in[ccrBp+int(k)]) == id {
				return 8
			}
		}
		return -2
	}
	// exe 0x42b740 with the 0x4d9700 remap of 0x10/0x26.
	keyOf := func(sel int32) uint8 {
		if sel == 0x10 || sel == 0x26 {
			held := 0
			for id := uint32(0x242); id <= 0x244; id++ {
				if locate(id) != -2 {
					held++
				}
			}
			sel = 0x26
			if held >= 2 || in[ccrF1ce]&0xff != 0 {
				sel = 0x10
			}
		}
		if sel < 0 || sel > 0x2f {
			return 0
		}
		return uint8(npcRecruitExe.slotTable[sel])
	}

	switch cond {
	case 9: // 0x5234a3
		if uint32(slot[10]) != 0 {
			if k := keyOf(int32(in[ccrG686a70])); k != 9 && k != 0xa {
				return finish()
			}
		}
	case 0xb: // 0x523436
		if at := locate(ccrItemWard); at != -2 && at != -1 {
			holder := at
			if at == 8 {
				n52c++
				d := in[ccrRet52] - base
				if d < 0 || d%ccrCharSize != 0 || d/ccrCharSize >= ccrChars {
					panic("characterConditionRemove: 0x52c480 result is not a party member")
				}
				holder = int(d / ccrCharSize)
			}
			if holder == who && !inList(who, ccrItemKeep, false) {
				return finish()
			}
		}
	}

	if notify { // 0x590950 -> 0x58aad0(8, fmt, name)
		r58 = [4]int64{8, ccrNoticeFmt, base + int64(who)*ccrCharSize + 5}
		n58 = 1
	}
	slot[cond] = 0
	v := make([]int64, pcForm+3*pcFormLen)
	v[0] = int64(who)
	copy(v[1:1+pcSlots], slot)
	v[pcOld], v[pcCombat] = b01, in[ccrCombat]
	copy(v[pcMode:], pc[:5+2*pcMembers+ccrForms])
	res = characterPrimaryConditionNode(v, r)

	switch cond { // jump table 0x523674
	case 1:
		b21, b15 = 0, 0
	case 7:
		b09 = 0
	case 9, 12:
		g6840bb = 1
	case 0xb:
		if combat && in[ccrRec98]&0xff != 0 {
			copy(p1d, in[ccrP4d:ccrP4d+8])
		}
	case 0xd:
		n538, a538 = 1, 1                          // 0x538d10(who, 1) -> 0x538e00
		r536, n536 = [4]int64{int64(who), 1, 0}, 2 // last of (who,0,0), (who,1,0)
	}
	a50, n50 = int64(who), 1
	if world != nil {
		ch := world.char(who)
		for k, value := range slot {
			world.m.w32(ch+0xa01+4*uint32(k), uint32(value))
		}
		world.m.w32(ch+0xb01, uint32(res[0]))
		world.m.w32(ch+0xb09, uint32(b09))
		world.m.w32(ch+0xb15, uint32(b15))
		world.m.w32(ch+0xb21, uint32(b21))
		world.rebuild(who, r)
		for k := range slot {
			slot[k] = int64(world.m.r32(ch + 0xa01 + 4*uint32(k)))
		}
		b11 = world.m.r32(ch + 0xb11)
		b09, b15, b21 = int64(world.m.r32(ch+0xb09)), int64(world.m.r32(ch+0xb15)), int64(world.m.r32(ch+0xb21))
		res[0] = int64(world.m.r32(ch + 0xb01))
	}
	a52f1, a52f2, n52f = base+int64(who)*ccrCharSize, int64(cond), 1
	if !bVar7 && b11 > 0 && uint32(res[0]) < 0xd && combat { // 0x523614..0x523643
		r4e, n4e = [4]int64{int64(who), 1, 0, 0}, 1
		// 0x53a930 follows when the 0x4e79a0 result is non-zero; it has no declared output.
	}
	return finish()
}

// ccrWorld carries the rule cells shared by removal, rebuilding and death.
// Legacy callers supply only the removal interface; their other record cells
// start at zero. Callers with HP/stamina cells keep this world across targets.
// The original party and condition arrays have eight and twenty entries.
type ccrWorld struct {
	m       n4Mem
	base    uint32
	inputs  [ccrChars][]int64
	present [ccrChars]uint32
	voice   func(...uint32)
	died    func(int)
}

func newCCRWorld(in []int64) *ccrWorld {
	if len(in) != ccrLen {
		panic("characterConditionRemove: wrong input count")
	}
	w := &ccrWorld{m: n4Mem{}, base: uint32(in[ccrCharBase])}
	w.m.w32(0x685174, w.base)
	for j := 0; j < ccrChars; j++ {
		w.present[j] = uint32(in[ccrPres+j]) & 255
		w.m.w8(w.char(j)+4, w.present[j])
	}
	who := int(in[ccrWho])
	if who < 0 || who >= ccrChars {
		panic("characterConditionRemove: who out of range")
	}
	w.present[who] = uint32(in[ccrWhoPres]) & 255
	w.m.w8(w.char(who)+4, w.present[who])
	w.inputs[who] = append([]int64(nil), in...)
	for i, off := range []uint32{0xb01, 0xb11, 0xb09, 0xb15, 0xb21} {
		w.m.w32(w.char(who)+off, uint32(in[ccrB01+i]))
	}
	for i := 0; i < ccrSlots; i++ {
		w.m.w32(w.char(who)+0xa01+4*uint32(i), uint32(in[ccrSlot+i]))
	}
	return w
}

func (w *ccrWorld) char(who int) uint32 { return w.base + uint32(who)*ccrCharSize }

func (w *ccrWorld) remove(who, cond int, notify int64, r Rand) []int64 {
	v := append([]int64(nil), w.inputs[who]...)
	if len(v) == 0 {
		panic("condition rebuild: missing removal world")
	}
	ch := w.char(who)
	v[ccrWho], v[ccrCond], v[ccrNotify] = int64(who), int64(cond), notify
	for i, off := range []uint32{0xb01, 0xb11, 0xb09, 0xb15, 0xb21} {
		v[ccrB01+i] = int64(w.m.r32(ch + off))
	}
	for i := 0; i < ccrSlots; i++ {
		v[ccrSlot+i] = int64(w.m.r32(ch + 0xa01 + 4*uint32(i)))
	}
	return ccrRun(v, r, w)
}

// Reuse 50e650's status/attribute rebuild and 4ed9d0's exact x87 resource
// formulas. The latter reports 522fe0 through its call record; dispatch it
// before stamina rebuilding, as in the native call order.
func (w *ccrWorld) rebuild(who int, r Rand) {
	m, ch := w.m, w.char(who)
	v := make([]int64, 348)
	v[0], v[1] = int64(who), int64(m.r8(ch+4))
	v[10], v[11], v[12], v[13] = int64(m.r32(ch+0x69)), int64(m.r32(ch+0x71)), int64(m.r32(ch+0x75)), int64(m.r32(ch+0x89))
	for i := 0; i < 15; i++ {
		v[14+i] = int64(m.r32(ch + 0x8d + 4*uint32(i)))
	}
	for i := 0; i < 7; i++ {
		v[29+i] = int64(m.r32(ch + 0xe5 + 20*uint32(i)))
	}
	for i := 0; i < 41; i++ {
		v[36+i] = int64(m.r32(ch + 0x19f + 38*uint32(i)))
	}
	for i := 0; i < 20; i++ {
		v[77+i] = int64(m.r32(ch + 0xa01 + 4*uint32(i)))
	}
	v[97] = int64(m.r32(ch + 0xb09))
	out := statusEffectRebuildNode(v, r)
	for i := 0; i < 7; i++ {
		m.w32(ch+0xe9+20*uint32(i), uint32(out[206+i]))
	}
	m.w32(ch+0x89, min(m.r32(ch+0x89), 50))
	for i := uint32(0); i < 15; i++ {
		m.w32(ch+0x8d+4*i, min(m.r32(ch+0x8d+4*i), 50))
	}
	// The callee's private call-record cells must not alias party records.
	hp := n4Mem{}
	for a, b := range m {
		hp[a] = b
	}
	c := &rcraC{m: hp, ch: ch}
	hp.w32(rcraS522fe0.count, 0)
	c.f52a2f0()
	m.w32(ch+0xb0d, hp.r32(ch+0xb0d))
	m.w32(ch+0xb11, hp.r32(ch+0xb11))
	if hp.r32(rcraS522fe0.count) != 0 {
		w.applyDeath(who, r)
	}
	c.m = m
	c.f52a3e0()
}

// 522fe0's death branch on the shared record. Its public node interface does
// not carry the rebuilt resource maxima or the other members' HP. Reuse its
// feature/primary helpers and the death node after clearing states in this
// world, so nested removals see the same maxima and resources.
func (w *ccrWorld) applyDeath(who int, r Rand) {
	m, ch := w.m, w.char(who)
	v := make([]int64, ccaWLen)
	v[ccaWBase] = int64(w.base)
	v[ccaWWhoPres] = int64(w.present[who])
	v[ccaWCls], v[ccaWRace] = int64(m.r32(ch+0x69)), int64(m.r32(ch+0x71))
	s := ccaNew(v, 0, r, true)
	if s.feat(2) {
		panic("condition rebuild: death-survival feature needs its character world")
	}
	old := m.r32(ch + 0xb01)
	changed := m.r32(ch+0xa49) < 9999
	if changed {
		m.w32(ch+0xa49, 9999)
		for i := 0; i < 20; i++ {
			s.put(ccaWS+i, int64(m.r32(ch+0xa01+4*uint32(i))))
		}
		s.put(ccaWB01, int64(old))
		copy(s.v[ccaWPC:ccaWPC+ccaPCLen], w.inputs[who][ccrPC:])
		s.primary(int64(who))
		m.w32(ch+0xb01, s.u(ccaWB01))
	}
	w.rebuild(who, r)
	if !changed {
		return
	}
	for i := 0; i < 18; i++ {
		if i != 10 && m.r32(ch+0xa01+4*uint32(i)) != 0 {
			w.remove(who, i, 0, r)
		}
	}
	death := make([]int64, len(cdaIn)+1)
	death[0] = int64(who)
	for i, f := range cdaIn {
		switch {
		case len(f.name) == 2 && f.name[0] == 'p':
			death[i+1] = int64(w.present[int(f.name[1]-'0')])
		case len(f.name) == 2 && f.name[0] == 'h':
			death[i+1] = int64(m.r32(w.char(int(f.name[1]-'0')) + 0xb11))
		case len(f.name) == 2 && f.name[0] == 't':
			death[i+1] = int64(m.r32(w.char(int(f.name[1]-'0')) + 0xb01))
		case f.name == "s.fa":
			death[i+1] = -1
		case f.name == "c.75":
			death[i+1] = int64(m.r32(ch + 0x75))
		case f.name == "c.b1d":
			death[i+1] = int64(m.r32(ch + 0xb1d))
		}
	}
	out := characterDeathApplyNode(death, r)
	var args [5]uint32
	var calls int64
	for i, f := range cdaOut {
		if f.name == "r52e.n" {
			calls = out[i]
		}
		for a := 0; a < 5; a++ {
			if f.name == fmt.Sprintf("r52e.a%d", a+1) {
				args[a] = uint32(out[i])
			}
		}
	}
	m.w32(ch+0xb11, 0)
	m.w32(ch+0xb1d, 0)
	if calls != 0 && w.voice != nil {
		args[0] = w.base + args[0] - paiChars
		args[4] = m.r32(0x5ed914)
		w.voice(args[:]...)
	}
	if w.died != nil {
		w.died(who)
	}
	if old != m.r32(ch+0xb01) && m.r8(0x68c4fa) == 0 {
		if m.r8(0x6875f8) != 0 {
			m.w8(0x6875f8, 0)
			return
		}
		state := m.r32(ch + 0xb01)
		if state == 18 || state == 17 {
			kind := m.r32([]uint32{0x68c538, 0x68c540, 0x68c564}[r.R(3)])
			arg3, arg4 := uint32(0), m.r32(0x5ed8c8)
			if ccaTab.evd[state-2] == 0x52f479 {
				arg3, arg4 = m.r32(0x5ed8d4), m.r32(0x5ed8cc)
			}
			if w.voice != nil {
				w.voice(ch, kind, arg3, arg4, m.r32(0x5ed914))
			}
		}
	}
}
