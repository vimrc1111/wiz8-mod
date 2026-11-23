package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// Inputs of attackTargetAvailableNode (exe 0x518e30). Groups of eight are the
// eight party slots; ataMon is three fixture monster records (ataMonFields each).
const (
	ataSlot       = 0  // arg1: attacker slot (0x518e3b)
	ataCtx        = 1  // arg2: attack context, passed to 0x4e77b0 / 0x5194e0
	ataSkip       = 2  // arg3: 1 skips the monster loop (0x518e78)
	ataFlag       = 3  // arg4: byte, the fallback disposition request (0x518e61)
	ataCombat     = 4  // byte 0x683f94
	ataSlotFlag   = 5  // G+0x98+slot*0xd4 byte (0x518e54), G = *0x6836a8
	ataBlocked    = 6  // G+0xa54 byte (0x547510)
	ataRound      = 7  // G+4 u32 (0x547510)
	ataPhase      = 8  // u32 0x683fe7 (0x5194e0)
	ataSpellMode  = 9  // byte 0x683f95
	ataItemMode   = 10 // byte 0x683f96
	ataCount      = 11 // 0x5e2c70 of the monster list
	ataPresent    = 12 // queue +0 per slot, stride 0x106 (0x518f5d)
	ataHP         = 20 // C+0xb11 per slot (0x518f67)
	ataStatus     = 28 // C+0xb01 per slot (0x518f76)
	ataCharm      = 36 // C+0xa35 per slot (0x546f10, 0x547010)
	ataUnavail    = 44 // C+0x17b5 per slot (0x5190cd)
	ataTeam       = 52 // 0x687525 + 12*slot (0x519065)
	ataOrder      = 60 // 0x68751d..0x68751f (0x5190b1)
	ataQuad       = 63 // 0x687511..0x68751c: formation quadrants 0..3, three slots each (0x51aec0)
	ataHeading    = 75 // party heading degrees 0x686a40 (0x51aec0)
	ataMainKind   = 76 // attacker record fields below are the selected slot's
	ataMainAvail  = 77
	ataOffKind    = 78
	ataOffAvail   = 79
	ataMainItem   = 80
	ataOffItem    = 81
	ataItemRange  = 82 // item records 0..3, +0x47
	ataItemSpell  = 86 // item records 0..3, +0x63
	ataSpellID    = 90 // queued action spell id (queue +5)
	ataSpellRange = 91 // spell record +0x12f of that id
	ataListItem   = 92 // item index the action item list points at (0x5207e0 arg)
	ataMainAction = 93 // queue +1
	ataOffAction  = 94 // queue +0x3d
	ataBattleKind = 95 // battle action kind
	ataItemSpRng  = 96 // spell record +0x12f of item 0..3's spell
	ataParty      = 100
	ataMon        = 104
	ataMonFields  = 11
	ataMonsters   = 3
	ataInputs     = ataMon + ataMonsters*ataMonFields
)

var ataDispatch, ataDispatchErr = loadAtaDispatch()

// Kind dispatch at exe 0x519150 (kinds 0..8), read from the player's Wiz8.exe.
// Destinations: 0x519009 hand range, 0x51913a long range (available),
// 0x5190e6 none, 0x51905e short range, 0x519015 spell, 0x51903d item spell.
func loadAtaDispatch() ([9]uint32, error) {
	var t [9]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("attack target available: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	o, err := peOffset(b, 0x519150)
	if err != nil {
		return t, err
	}
	if len(b)-o < 4*len(t) {
		return t, fmt.Errorf("attack target available: truncated dispatch")
	}
	want := [9]uint32{0x519009, 0x519009, 0x51913a, 0x5190e6, 0x5190e6, 0x51905e, 0x5190e6, 0x519015, 0x51903d}
	for i := range t {
		t[i] = binary.LittleEndian.Uint32(b[o+4*i:])
		if t[i] != want[i] {
			return t, fmt.Errorf("attack target available: unexpected dispatch destination %#x for kind %d", t[i], i)
		}
	}
	return t, nil
}

func init() {
	names := []string{"attackerSlot", "attackContext", "skipMonsters", "fallbackDisposition", "combatMode", "slotCombatFlag", "combatOpeningBlocked", "combatRound", "combatPhase", "specialSpellMode", "specialItemMode", "monsterCount"}
	for _, g := range []string{"present", "hp", "status", "charm", "formationUnavailable", "team"} {
		for i := 0; i < 8; i++ {
			names = append(names, g+strconv.Itoa(i))
		}
	}
	names = append(names, "partyOrder0", "partyOrder1", "partyOrder2")
	for q := 0; q < 4; q++ {
		for s := 0; s < 3; s++ {
			names = append(names, fmt.Sprintf("quadrant%dSlot%d", q, s))
		}
	}
	names = append(names, "partyHeadingDegrees", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex",
		"item0Range", "item1Range", "item2Range", "item3Range", "item0Spell", "item1Spell", "item2Spell", "item3Spell",
		"spellId", "spellRange", "actionItemIndex", "mainActionKind", "offActionKind", "battleActionKind",
		"item0SpellRange", "item1SpellRange", "item2SpellRange", "item3SpellRange",
		"partyX", "partyY", "partyZ", "partyRadius")
	for m := 0; m < ataMonsters; m++ {
		for _, f := range []string{"active", "hp", "status", "disposition", "attackEnabled", "physicalAllowed", "spellAllowed", "x", "y", "z", "radius"} {
			names = append(names, "monster"+strconv.Itoa(m)+f)
		}
	}
	Nodes["00518e30"] = Node{Addr: "00518e30", In: names, Out: []string{"canAttack"}, F: attackTargetAvailableNode}
}

// attackTargetAvailableNode: does the attacker have any attackable target?
// Monster loop (skipped when arg3 == 1): 0x5194e0 per hand. Party loop (only
// contexts 0 and 8): other party members of the wanted disposition.
// Fixture caps (panic past them): three monsters, four items, spell ids below
// 150 with distinct non-zero ids, an action item list that always points at a
// valid item index, party-order bytes 0..7 or 0xff.
func attackTargetAvailableNode(in []int64, r Rand) []int64 {
	if len(in) != ataInputs {
		panic("attack target available: expected " + strconv.Itoa(ataInputs) + " inputs")
	}
	if ataDispatchErr != nil {
		panic(ataDispatchErr)
	}
	u := func(i int, max int64) {
		if in[i] < 0 || in[i] > max {
			panic("attack target available: input " + strconv.Itoa(i) + " outside its storage width")
		}
	}
	u(ataSlot, 7) // eight party slots: array strides at 0x518e4a, 0x518fc1
	for _, i := range []int{ataCtx, ataSkip, ataRound, ataPhase} {
		u(i, 4294967295)
	}
	for _, i := range []int{ataFlag, ataCombat, ataSlotFlag, ataBlocked, ataSpellMode, ataItemMode} {
		u(i, 255)
	}
	u(ataCount, ataMonsters)
	for i := 0; i < 8; i++ {
		u(ataPresent+i, 255)
		u(ataHP+i, 4294967295)
		u(ataStatus+i, 4294967295)
		u(ataCharm+i, 4294967295)
		u(ataUnavail+i, 255)
		u(ataTeam+i, 255)
	}
	for j := 0; j < 3; j++ {
		if in[ataOrder+j] != 255 { // 0xff ends an entry (0x5190b7); movsx makes other bytes a negative index
			u(ataOrder+j, 7)
		}
	}
	for _, i := range []int{ataListItem} {
		u(i, 3) // 0x5207e0 asserts index < item count (0x683f78); fixture has four items
	}
	for m := 0; m < ataMonsters; m++ {
		b := ataMon + m*ataMonFields
		for i, max := range []int64{255, 4294967295, 4294967295, 255, 255, 255, 255, 4294967295, 4294967295, 4294967295, 4294967295} {
			u(b+i, max) // M+0x14 u8, +0x2b u32, +0x107 u32, +0x16 u8, +0x28a..c u8, scene x/y/z/radius float bits
		}
	}
	ids := map[int64]bool{}
	for _, i := range []int{ataSpellID, ataItemSpell, ataItemSpell + 1, ataItemSpell + 2, ataItemSpell + 3} {
		u(i, 149) // spell table of the fixture
		if in[i] != 0 {
			if ids[in[i]] {
				panic("attack target available: spell id shared by two inputs")
			}
			ids[in[i]] = true
		}
	}

	slot := int(in[ataSlot])
	// 0x518e44..0x518e6b: combat flag and the slot's own flag force request 1;
	// otherwise the byte argument. Result is 1 or 2.
	want := in[ataFlag]
	if in[ataCombat] != 0 && in[ataSlotFlag] != 0 {
		want = 1
	}
	if want != 0 {
		want = 2
	} else {
		want = 1
	}
	opening := combatOpeningConditionNode([]int64{in[ataCombat], in[ataBlocked], in[ataRound]}, r)[0] != 0 // 0x547510
	charm := func(i int) int64 { return int64(int32(uint32(in[ataCharm+i]))) }

	handAvail := func(h int) bool {
		return handAttackAvailableNode([]int64{int64(slot), int64(h), in[ataMainKind], in[ataMainAvail], in[ataOffKind], in[ataOffAvail],
			in[ataMainItem], in[ataOffItem], in[ataItemRange], in[ataItemRange+1], in[ataItemRange+2], in[ataItemRange+3]}, r)[0] != 0 // 0x53d2a0
	}
	itemSpell := in[ataItemSpell+int(in[ataListItem])]
	itemSpellRange := in[ataItemSpRng+int(in[ataListItem])]

	// Monster loop, 0x518e78..0x518f1a.
	if in[ataSkip] != 1 {
		for m := 0; m < int(in[ataCount]); m++ {
			f := in[ataMon+m*ataMonFields : ataMon+(m+1)*ataMonFields]
			if f[0] == 0 || f[1] == 0 || f[2] >= 0x12 { // M+0x14, M+0x2b, M+0x107
				continue
			}
			if !opening && targetDispositionNode([]int64{int64(slot), charm(slot), f[3]}, r)[0] != want {
				continue
			}
			for h := 0; h < 2; h++ {
				if !handAvail(h) {
					continue
				}
				v := make([]int64, 78) // 0x5194e0 inputs
				v[0], v[1], v[2], v[3] = int64(slot), int64(h), in[ataCtx], 0
				copy(v[4:7], f[4:7])
				v[7], v[8], v[9], v[10] = in[ataPhase], in[ataCombat], in[ataSpellMode], in[ataItemMode]
				v[11] = in[ataPresent+slot]
				v[12], v[13] = in[ataHP+slot], in[ataStatus+slot]
				copy(v[14:18], in[ataMainKind:ataMainKind+4])
				v[18] = 1 // in-party flag: read only for a disabled hand, which 0x53d2a0 excluded
				v[19], v[20] = in[ataMainItem], in[ataOffItem]
				copy(v[21:25], in[ataItemRange:ataItemRange+4])
				v[25], v[26] = in[ataSpellID], in[ataSpellRange]
				v[27], v[34] = itemSpell, itemSpellRange // action item list refers to this item (0x5207e0)
				v[31], v[32], v[33] = in[ataMainAction], in[ataOffAction], in[ataBattleKind]
				copy(v[38:41], in[ataParty:ataParty+3])
				copy(v[41:44], f[7:10])
				v[44], v[45] = f[10], in[ataParty+3]
				copy(v[46:54], in[ataTeam:ataTeam+8])
				v[54] = in[ataHeading]
				copy(v[55:67], in[ataQuad:ataQuad+12])
				copy(v[67:70], in[ataOrder:ataOrder+3])
				copy(v[70:78], in[ataUnavail:ataUnavail+8])
				if monsterTargetAttackRangeNode(v, r)[0] != 0 {
					return []int64{1}
				}
			}
		}
	}

	// Party loop, 0x518f1c..0x519144: contexts 0 and 8 only.
	if in[ataCtx] != 0 && in[ataCtx] != 8 {
		return []int64{0}
	}
	usable := characterTargetableNode(func() []int64 {
		t := []int64{int64(slot)}
		for i := 0; i < 8; i++ {
			t = append(t, in[ataPresent+i], int64(int32(uint32(in[ataHP+i]))), in[ataStatus+i])
		}
		return t
	}(), r)[0] != 0 // 0x53c270 inside 0x4e77b0
	for t := 0; t < 8; t++ {
		// 0x518f4b..0x518f7e. t == slot is skipped here, which also keeps the
		// (char)slot == t early return at 0x518fb4 unreachable for slots 0..7.
		if t == slot || in[ataPresent+t] == 0 || in[ataHP+t] == 0 || in[ataStatus+t] >= 0x12 {
			continue
		}
		if characterDispositionNode([]int64{in[ataCharm+slot], in[ataCharm+t]}, r)[0] != want { // 0x547010
			continue
		}
		for h := 0; h < 2; h++ {
			if !handAvail(h) {
				continue
			}
			// 0x4e77b0: context 0 is the main action record, context 8 the plain hand attack.
			kind := int64(0)
			if in[ataCtx] == 0 {
				kind = in[ataMainAction]
			}
			if !usable && kind != 10 && kind != 11 {
				kind = 4294967295
			}
			if kind > 8 {
				continue
			}
			rng := int64(0)
			switch ataDispatch[kind] {
			case 0x51913a: // kind 2: long range, always available
				return []int64{1}
			case 0x5190e6: // kinds 3, 4, 6
				continue
			case 0x519009:
				rng = charAttackRangeNode([]int64{int64(h), 1, in[ataMainKind], in[ataMainAvail], in[ataOffKind], in[ataOffAvail],
					in[ataMainItem], in[ataOffItem], in[ataItemRange], in[ataItemRange+1], in[ataItemRange+2], in[ataItemRange+3]}, r)[0] // 0x519ac0
			case 0x519015:
				if in[ataSpellID] == 0 {
					continue
				}
				rng = int64(int32(uint32(in[ataSpellRange]))) // spell record +0x12f
			case 0x51903d:
				rng = -1 // 0x5207e0: zero item spell
				if itemSpell != 0 {
					rng = int64(int32(uint32(itemSpellRange)))
				}
			case 0x51905e: // kind 5: straight to the formation test
			}
			if rng == -1 {
				continue
			}
			if rng != 0 || ataFormationAllows(in, slot, t) {
				return []int64{1}
			}
		}
	}
	return []int64{0}
}

// ataFormationAllows is 0x51905e..0x5190e2: true when the target counts (return 1).
func ataFormationAllows(in []int64, slot, t int) bool {
	a, b := in[ataTeam+slot], int64(int8(in[ataTeam+t]))
	if a == in[ataTeam+t] || a == 4 || b == 4 {
		return true
	}
	if d := a - b; d != 2 && d != -2 {
		return true
	}
	count := 0
	for j := 0; j < 3; j++ {
		if o := in[ataOrder+j]; o != 255 && in[ataUnavail+int(o)] == 0 {
			count++
		}
	}
	return count == 0
}
