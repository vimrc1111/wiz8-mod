package calc

import (
	"fmt"
	"regexp"
	"strconv"
)

// Node 0x4ec1e0 status_action_override: which forced action a status-afflicted
// actor takes. Parameter struct: type@0 (1 character, 2 monster), index@4,
// monster id@8. Interface: re/emu/cases/004ec1e0.json.
//
// Input vector: saoOwn fields below, then the 877 inputs of weaponSetSwapAllowedNode
// minus partySlot, showMessage and rebuildEffects (those are supplied here), then
// the monster block (saoMonsterNames): the monster record's own fields, the
// inputs of monsterAttackTargetAvailableNode that are not already party fields,
// five template caches and the spell-record kind word of 150 spells.
const (
	saoType      = 0
	saoSlot      = 1
	saoMonsterID = 2
	saoFloor     = 3  // C+0xa11, +0xa19, +0xa2d, +0xa31, +0xa39 of the party slot
	saoActKind   = 8  // queue+1+slot*262 (0x4ec2xx: ==0xb)
	saoSlotFlag  = 9  // combat+0x98+slot*0xd4
	saoCombat    = 10 // byte 0x683f94
	saoMainAvail = 11 // C+0x114d
	saoOffAvail  = 12 // C+0x11a8
	saoPresent   = 13
	saoHP        = 21
	saoStatus    = 29
	saoCharm     = 37
	saoUnavail   = 45
	saoTeam      = 53
	saoOrder     = 61
	saoItemPre   = 64 // items[-1] range byte (+0x47)
	saoOwn       = 65
)

// saoD maps an index of the weapon-swap vector to this node's vector.
func saoD(j int) int {
	if j < 508 {
		return saoOwn + j - 1
	}
	return saoOwn + j - 3
}

// saoSwapVector rebuilds the 880-field weapon-swap input: no message, rebuild on.
func saoSwapVector(in []int64) []int64 {
	v := make([]int64, 880)
	v[0], v[509] = in[saoSlot], 1
	for j := 1; j < 880; j++ {
		if j != 508 && j != 509 {
			v[j] = in[saoD(j)]
		}
	}
	return v
}

// saoState is the hand/availability state read after (or without) the swap.
type saoState struct {
	itemID  [2]int64 // hands 6, 7 (equipment IDs, -1 empty)
	modes   [2]int64
	avail   [2]int64 // C+0x114d, C+0x11a8
	unavail int64
}

func saoRange(in []int64, id int64) int64 {
	if id == -1 {
		return in[saoItemPre]
	}
	if id < 0 || id > 11 {
		panic("status action override: item ID outside fixture definitions")
	}
	return in[saoD(540+5*int(id)+3)]
}

// saoHands builds the 53d310 / 518e30 hand arguments: item indices 0 and 1
// stand for hands 6 and 7, their range bytes read from items[id].
func saoHands(in []int64, st saoState) []int64 {
	return []int64{st.modes[0], st.avail[0], st.modes[1], st.avail[1], 0, 1,
		saoRange(in, st.itemID[0]), saoRange(in, st.itemID[1]), 0, 0}
}

func saoAnyHand(in []int64, st saoState, r Rand) bool {
	h := saoHands(in, st)
	v := []int64{in[saoSlot], h[0], h[1], h[2], h[3], h[4], h[5], h[6], h[7], h[8], h[9]}
	return anyHandAttackAvailableNode(v, r)[0] != 0
}

// saoAttack is 0x518e30(slot, 8, 1, flag): the party-target loop with the monster loop skipped.
func saoAttack(in []int64, st saoState, flag int64, r Rand) bool {
	h := saoHands(in, st)
	v := make([]int64, ataInputs)
	v[ataSlot], v[ataCtx], v[ataSkip], v[ataFlag] = in[saoSlot], 8, 1, flag
	v[ataCombat], v[ataSlotFlag] = in[saoCombat], in[saoSlotFlag]
	for t := 0; t < 8; t++ {
		v[ataPresent+t], v[ataHP+t], v[ataStatus+t] = in[saoPresent+t], in[saoHP+t], in[saoStatus+t]
		v[ataCharm+t], v[ataTeam+t] = in[saoCharm+t], in[saoTeam+t]
		v[ataUnavail+t] = in[saoUnavail+t]
	}
	v[ataUnavail+int(in[saoSlot])] = st.unavail
	copy(v[ataOrder:ataOrder+3], in[saoOrder:saoOrder+3])
	v[ataMainKind], v[ataMainAvail], v[ataOffKind], v[ataOffAvail] = h[0], h[1], h[2], h[3]
	v[ataMainItem], v[ataOffItem] = h[4], h[5]
	copy(v[ataItemRange:ataItemRange+4], h[6:10])
	return attackTargetAvailableNode(v, r)[0] != 0
}

// saoFeature7 is 0x547940(C, 7): class row C+0x69 and race row C+0x71 (-1 skips).
func saoFeature7(class, race int64) bool {
	if breathTablesErr != nil {
		panic(breathTablesErr)
	}
	if class < -1 || class > 14 || race < -1 || race > 15 {
		panic("status action override: class or race outside the feature tables")
	}
	has := false
	if class != -1 {
		for _, f := range breathTables.class[class] {
			has = has || f == 7
		}
	}
	if race != -1 {
		for _, f := range breathTables.race[race] {
			has = has || f == 7
		}
	}
	return has
}

func saoSigned32(v int64) int64 {
	if v < -1 || v > 4294967295 {
		panic("status action override: value outside u32")
	}
	return int64(int32(uint32(v)))
}

func statusActionOverrideNode(in []int64, r Rand) []int64 {
	if len(in) != len(saoInputNames) {
		panic("status action override: expected " + strconv.Itoa(len(saoInputNames)) + " inputs")
	}
	switch in[saoType] {
	case 1:
		return []int64{saoCharacter(in, r)}
	case 2:
		return []int64{saoMonster(in, r)}
	}
	panic("status action override: type other than 1 or 2 asserts at 0x4ec1e0 (Combat.cpp:4520)")
}

// saoLadder is the shared tail of both types after the per-type fields are known.
// Fields are the u32 words at iVar6+0x10, +0x18, +0x2c, +0x30, +0x34, +0x38.
type saoFields struct {
	f10, f18, f2c, f30, f34, f38 int64
	str                          int64 // local_4 (strength / M+0x247)
	cVar2, local_b               bool
	b9, b10                      bool
	mon                          bool
	tpl0xcb                      int64
	charAbility                  bool // character: 0x547940(C,7) set
}

func saoLadder(f saoFields, r Rand) (int64, bool) {
	if f.f10 != 0 && r.R(100) < 25 {
		return 3, true
	}
	if f.f38 != 0 {
		if r.R(100) < int(uint32(f.str)/3) {
			return 7, true
		}
		return 0, true
	}
	if f.f30 != 0 && r.R(100) < 20 {
		if !f.mon && !f.charAbility || f.mon && f.tpl0xcb != 0xc {
			return 2, true
		}
	}
	if f.f2c != 0 && r.R(100) < 80 {
		if f.cVar2 && !f.local_b && r.R(100) < 40 {
			return 8, true
		}
		return int64(r.R(3)) + 4, true
	}
	if f.f18 != 0 {
		if r.R(100) < 25 && !f.b9 {
			return 1, true
		}
		if f.mon && r.R(100) < 40 {
			return 0xc, true
		}
	}
	if f.b10 {
		return 0xb, true
	}
	if f.f34 == 0 {
		return -1, true
	}
	return 0, false
}

func saoCharacter(in []int64, r Rand) int64 {
	who := int(in[saoSlot])
	if who < 0 || who > 7 {
		panic("status action override: party slot -1 asserts (Combat.cpp:4499), 8.. is outside the party array")
	}
	pre := saoState{
		itemID:  [2]int64{saoSigned32(in[saoD(282)]), saoSigned32(in[saoD(283)])},
		modes:   [2]int64{in[saoD(878)], in[saoD(879)]},
		avail:   [2]int64{in[saoMainAvail], in[saoOffAvail]},
		unavail: in[saoUnavail+who],
	}
	f := saoFields{f10: in[saoFloor], f18: in[saoFloor+1], f2c: in[saoFloor+2], f30: in[saoFloor+3], f38: in[saoFloor+4],
		f34: in[saoCharm+who], str: in[saoD(827)], b9: in[saoActKind] == 0xb}
	f.local_b = in[saoSlotFlag] != 0
	var localC bool
	if saoAnyHand(in, pre, r) {
		localC = saoAttack(in, pre, 0, r)
		f.cVar2 = saoAttack(in, pre, 1, r)
	}
	f.charAbility = saoFeature7(saoSigned32(in[saoD(2)]), saoSigned32(in[saoD(3)]))
	if v, done := saoLadder(f, r); done {
		return v
	}
	if localC {
		return 9
	}
	// 0x51f8d0: nonzero only when all four hand records are empty.
	empty := true
	for h := 0; h < 4; h++ {
		empty = empty && saoSigned32(in[saoD(282+h)]) == -1
	}
	if empty {
		return 10
	}
	for h := 0; h < 2; h++ {
		id := pre.itemID[h]
		cursed := int64(0)
		if id < -1 || id > 11 {
			panic("status action override: item ID outside fixture definitions")
		}
		if id != -1 {
			cursed = in[saoD(540+5*int(id)+4)]
		}
		if itemRemovalAllowedNode([]int64{int64(6 + h), id & 0xffffffff, cursed, in[saoD(517+7*h)], in[saoD(511)]}, r)[0] == 0 {
			return 10
		}
	}
	out := weaponSetSwapAllowedNode(saoSwapVector(in), r)
	post := saoState{itemID: [2]int64{out[1], out[10]}, modes: [2]int64{out[421], out[422]}, unavail: out[320]}
	post.avail = saoAvail(in, post.itemID, [2]int64{out[3], out[12]}, r)
	if saoAnyHand(in, post, r) && saoAttack(in, post, 0, r) {
		return 9 // 0x590950 only formats a message
	}
	return 10
}

// Fixture list (re/emu/cases/004ec1e0.json): active list at 0x20000000+106496
// holding the monster of ID saoFixtureMonsterID first.
const (
	saoFixtureMonsterID = 77
	saoListAddress      = 0x20000000 + 106496
)

var (
	saoReTpl  = regexp.MustCompile(`^tpl(\d)Attack(Present|Range)$`)
	saoReChar = regexp.MustCompile(`^character(\d)(Quadrant|CurrentHP|Status|Charm|FormationUnavailable)$`)
	saoReSlot = regexp.MustCompile(`^slot(\d)Active$`)
	saoReForm = regexp.MustCompile(`^formationSlot(\d+)$`)
	// Inputs of monsterAttackTargetAvailableNode that this node supplies from elsewhere.
	saoReDrop = regexp.MustCompile(`^(attackIndex|allyFlag|monsterListPointer|combatMode|spellRange|mSpellFlag|tpl\dAttack(Present|Range)|character\d(Quadrant|CurrentHP|Status|Charm|FormationUnavailable)|slot\dActive|formationSlot1[234])$`)
)

// saoMonsterNames is the monster block that follows the weapon-swap fields.
var saoMonsterNames = func() []string {
	n := []string{"mon.f10", "mon.f18", "mon.f2c", "mon.f30", "mon.f38", "mon.str", "mon.noAttack", "mon.flag2fd"}
	for _, name := range monsterAttackTargetInputs {
		if !saoReDrop.MatchString(name) {
			n = append(n, name)
		}
	}
	for k := 0; k < 5; k++ {
		n = append(n, fmt.Sprintf("tpl%d.cb", k), fmt.Sprintf("tpl%d.e5", k), fmt.Sprintf("tpl%d.e6", k))
		for i := 0; i < 3; i++ {
			n = append(n, fmt.Sprintf("tpl%d.present%d", k, i), fmt.Sprintf("tpl%d.range%d", k, i), fmt.Sprintf("tpl%d.fc%d", k, i))
		}
	}
	for k := 0; k < 150; k++ {
		if k != 3 && k != 0x29 {
			n = append(n, fmt.Sprintf("spellValue%d", k))
		}
	}
	return n
}()

var saoIdx = func() map[string]int {
	m := map[string]int{}
	for i, n := range saoInputNames {
		m[n] = i
	}
	return m
}()

// saoTargetVector builds the 0x519c00 argument vector for attack index i and
// ally flag hand. 0x53d4b0 zeroes M+0x2e1 around the call, so the action is 0
// (weapon attack) and the spell index is never read.
func saoTargetVector(in []int64, tpl, i int, hand int64) []int64 {
	v := make([]int64, len(monsterAttackTargetInputs))
	at := func(name string) int64 { return in[saoIdx[name]] }
	num := func(s string) int { x, _ := strconv.Atoi(s); return x }
	for j, name := range monsterAttackTargetInputs {
		switch {
		case name == "attackIndex":
			v[j] = int64(i)
		case name == "allyFlag":
			v[j] = hand
		case name == "monsterListPointer":
			v[j] = saoListAddress
		case name == "combatMode":
			v[j] = in[saoCombat]
		case name == "mAction" || name == "mSpellIndex" || name == "spellRange" || name == "mSpellFlag":
			v[j] = 0
		case saoReTpl.MatchString(name):
			g := saoReTpl.FindStringSubmatch(name)
			f := map[string]string{"Present": "present", "Range": "range"}[g[2]]
			v[j] = at(fmt.Sprintf("tpl%s.%s%d", g[1], f, i))
		case saoReChar.MatchString(name):
			g := saoReChar.FindStringSubmatch(name)
			c := num(g[1])
			switch g[2] {
			case "Quadrant":
				v[j] = in[saoTeam+c]
			case "CurrentHP":
				v[j] = in[saoHP+c]
			case "Status":
				v[j] = in[saoStatus+c]
			case "Charm":
				v[j] = saoSigned32(in[saoCharm+c])
			default:
				v[j] = in[saoUnavail+c]
			}
		case saoReSlot.MatchString(name):
			v[j] = in[saoPresent+num(saoReSlot.FindStringSubmatch(name)[1])]
		case saoReForm.MatchString(name) && num(saoReForm.FindStringSubmatch(name)[1]) >= 12:
			v[j] = in[saoOrder+num(saoReForm.FindStringSubmatch(name)[1])-12]
		default:
			v[j] = at(name)
		}
	}
	return v
}

// saoMonsterAttack is 0x53d4b0 for attack i of the template cached at slot tpl:
// 1 no usable attack (missing data logs through 0x5182e0), 0 a target exists, 3 none.
func saoMonsterAttack(in []int64, tpl, i int, hand int64, r Rand) int64 {
	t := func(f string) int64 { return in[saoIdx[fmt.Sprintf("tpl%d.%s", tpl, f)]] }
	if t(fmt.Sprintf("present%d", i))&0xff == 0 || t(fmt.Sprintf("fc%d", i))&0xffff == 0 || t("e5")&0xff == 0 || t("e6")&0xff == 0 {
		return 1
	}
	if monsterAttackTargetAvailableNode(saoTargetVector(in, tpl, i, hand), r)[0] != 0 {
		return 0
	}
	return 3
}

// saoMonsterReady is 0x53d450: 0 as soon as one attack has a target, else the largest status.
func saoMonsterReady(in []int64, tpl int, hand int64, r Rand) int64 {
	if in[saoIdx["mon.noAttack"]]&0xff != 0 {
		return 1
	}
	best := int64(0)
	for i := 0; i < 3; i++ {
		v := saoMonsterAttack(in, tpl, i, hand, r)
		if v == 0 {
			return 0
		}
		if v > best {
			best = v
		}
	}
	return best
}

// saoMonsterCanAct is 0x547440 (kind, spell): attack kinds 0 and 3 always, kind 2 when the
// spell record's kind word (0x4ac9d0) is 3..7; spells 3 and 0x29 never.
func saoMonsterCanAct(in []int64, kind, spell int64) bool {
	switch kind {
	case 0, 3:
		return true
	case 2:
		s := int64(int32(uint32(spell)))
		if s >= 0x96 {
			panic("status action override: spell index >= 0x96 asserts (0x547440)")
		}
		if s < 0 {
			panic("status action override: negative spell index is outside the spell table")
		}
		if s == 3 || s == 0x29 {
			return false
		}
		w := int64(int32(uint32(in[saoIdx[fmt.Sprintf("spellValue%d", s)]])))
		return w >= 3 && w <= 7
	}
	return false
}

func saoMonster(in []int64, r Rand) int64 {
	at := func(name string) int64 { return in[saoIdx[name]] }
	switch id := saoSigned32(in[saoMonsterID]); {
	case id == -1:
		panic("status action override: monster ID -1 asserts (Combat.cpp:4521)")
	case id != saoFixtureMonsterID:
		panic("status action override: monster ID outside the fixture list (only 77)")
	}
	if at("monsterCount") < 1 || at("monsterCount") > 4 {
		panic("status action override: the fixture list holds 1..4 monsters, the first is the monster")
	}
	tpl := monsterAttackTargetTemplate(at("mTemplateId"))
	saoMonsterReady(in, tpl, 0, r) // result unused by 0x4ec1e0 (only the calls' checks)
	cVar2 := saoMonsterReady(in, tpl, 1, r) == 0
	f := saoFields{f10: at("mon.f10"), f18: at("mon.f18"), f2c: at("mon.f2c"), f30: at("mon.f30"), f38: at("mon.f38"),
		f34: at("mAllegianceWord"), str: at("mon.str") & 0xff, cVar2: cVar2, local_b: at("mOwnerVisible")&0xff != 0,
		b9: at("mAction") == 4, b10: at("mon.flag2fd") == 1, mon: true, tpl0xcb: at(fmt.Sprintf("tpl%d.cb", tpl)) & 0xff}
	if v, done := saoLadder(f, r); done {
		return v
	}
	if saoMonsterCanAct(in, at("mAction"), at("mSpellIndex")) {
		return 9
	}
	return -1
}

// saoInputNames: own fields by name; the 877 weapon-swap fields keep their
// index in re/emu/cases/0051d3b0.json (names are in Nodes["0051d3b0"].In, which
// is registered after this file's init).
var saoInputNames = func() []string {
	n := []string{"nodeType", "partySlot", "monsterId", "floor10", "floor18", "floor2c", "floor30", "floor38", "actionKind", "combatSlotFlag", "combatMode", "mainAvail", "offAvail"}
	for _, g := range []string{"present", "hp", "status", "charm", "unavail", "team"} {
		for t := 0; t < 8; t++ {
			n = append(n, g+strconv.Itoa(t))
		}
	}
	n = append(n, "partyOrder0", "partyOrder1", "partyOrder2", "itemPre.range")
	for j := 1; j < 880; j++ {
		if j != 508 && j != 509 {
			n = append(n, fmt.Sprintf("swap[%d]", j))
		}
	}
	n = append(n, saoMonsterNames...)
	for k := 0; k < 12; k++ {
		n = append(n, fmt.Sprintf("handItem[%d].weaponSkill", k), fmt.Sprintf("handItem[%d].chargeType", k))
	}
	return n
}()

func init() {
	Nodes["004ec1e0"] = Node{Addr: "004ec1e0", In: saoInputNames, Out: []string{"action"}, F: statusActionOverrideNode}
}

// saoSwapField is the vector position of a field of the weapon-swap case (resolved at run time:
// Nodes["0051d3b0"] is registered after this file's init).
func saoSwapField(name string) int {
	for j, n := range Nodes["0051d3b0"].In {
		if n == name {
			return saoD(j)
		}
	}
	panic("status action override: no weapon-swap field " + name)
}

// saoAvail is the hand availability bytes (C+0x114d, +0x11a8) that the rebuild
// 0x4ee220 rewrites after a weapon-set swap (0x4ee229..0x4ee37f), from the new
// hand items and their charge bytes. Item fields come from the 12 fixture items.
func saoAvail(in []int64, ids, charges [2]int64, r Rand) [2]int64 {
	item := func(id int64, f string) int64 {
		name := fmt.Sprintf("handItem[%d].%s", id, f)
		if f == "weaponSkill" || f == "chargeType" {
			return in[saoIdx[name]] // fields of this node only (+0x46, +0x66)
		}
		return in[saoSwapField(name)]
	}
	if attacksTablesErr != nil {
		panic(attacksTablesErr)
	}
	if handTypeDispatchErr != nil {
		panic(handTypeDispatchErr)
	}
	var typ, kind [2]int64
	for h := 0; h < 2; h++ {
		if ids[h] < -1 || ids[h] > 11 {
			panic("status action override: item ID outside fixture definitions")
		}
		if ids[h] == -1 {
			continue
		}
		k := item(ids[h], "type")
		if k < 0 || k >= 6 {
			panic("status action override: invalid item type (0x5201b0 asserts)")
		}
		switch handTypeDispatch[k] {
		case 0x520248:
			typ[h] = 1
		case 0x52024f:
			typ[h] = 1
			if h == 1 && ids[0] != -1 && item(ids[0], "subtype") == 131 {
				typ[h] = 3
			}
		case 0x52027e:
			typ[h] = 3
		case 0x520285:
			typ[h] = 2
		default:
			panic("status action override: unknown hand classification")
		}
	}
	avail := [2]int64{1, 1}
	for h := 0; h < 2; h++ {
		skill := int64(14)
		if ids[h] != -1 {
			skill = item(ids[h], "weaponSkill")
		}
		target := uint32(0x4ee319)
		if skill >= 0 && skill < 15 {
			target = attacksTables.combat[skill]
		}
		switch target {
		case 0x4ee2b8:
			kind[h] = 16
		case 0x4ee2c1:
			kind[h] = 17
			id, oid := ids[h], ids[1-h]
			if weaponDamageBonusItemNode([]int64{id, item(id, "subtype")}, r)[0] != 0 {
				ok := oid != -1 && compatiblePartnerItemsNode([]int64{id, oid, item(id, "subtype"), item(id, "range"),
					item(oid, "type"), item(oid, "subtype"), item(oid, "range")}, r)[0] != 0
				if !ok {
					avail[h] = 0
				}
			}
			if item(id, "chargeType") == 4 && charges[h]&0xff == 0 {
				avail[h] = 0
			}
		case 0x4ee319:
			kind[h] = -1
			avail[h] = 0
		default:
			panic("status action override: unexpected combat dispatch")
		}
	}
	if kind[1] != kind[0] || typ[1] == 2 || typ[1] == 3 || (ids[0] != -1 && item(ids[0], "subtype") == 131) || (typ[1] == 0 && typ[0] != 0) {
		avail[1] = 0
	}
	return avail
}
