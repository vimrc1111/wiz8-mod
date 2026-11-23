package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var partyTargetRangeDispatch, partyTargetRangeDispatchErr = loadPartyTargetRangeDispatch()

// Read the action-kind dispatch (exe 0x5198fc) from the player's executable.
// F performs no IO. The destinations must match the outcome classes that
// actionAttackRangeNode (0x5199f0 table) already reproduces: kinds 0/1 hand
// range, 7 spell range, 8 item spell range, 2 long range, 5 short range,
// 3/4/6 unavailable.
func loadPartyTargetRangeDispatch() ([9]uint32, error) {
	var t [9]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("party target range: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	o, err := peOffset(b, 0x5198fc)
	if err != nil {
		return t, err
	}
	if len(b)-o < 4*len(t) {
		return t, fmt.Errorf("party target range: truncated dispatch")
	}
	want := [9]uint32{0x51981d, 0x51981d, 0x5198f4, 0x519866, 0x519866, 0x519874, 0x519866, 0x51982d, 0x519851}
	for i := range t {
		t[i] = binary.LittleEndian.Uint32(b[o+4*i:])
		if t[i] != want[i] {
			return t, fmt.Errorf("party target range: unexpected dispatch destination %#x for kind %d", t[i], i)
		}
	}
	return t, nil
}

// Inputs 0..50 are exactly actionAttackRangeNode's (slot, hand, context and
// the records 0x4e77b0/0x519ac0/0x5207e0 read); then the other party slot,
// eight per-character formation quadrants (0x687525, stride 12), three center
// slots (0x68751d) and eight per-character formation-unavailable flags
// (C+0x17b5). Reach classes from the action node: -1 unusable, 0 short, >0
// long; 0x5198fc maps to the same classes (kind 2 long, 5 short, 3/4/6 none).
func partyTargetAttackRangeNode(in []int64, r Rand) []int64 {
	if len(in) != 71 {
		panic("party target attack range: expected 71 inputs")
	}
	if partyTargetRangeDispatchErr != nil {
		panic(partyTargetRangeDispatchErr)
	}
	if in[51] < 0 || in[51] > 7 {
		panic("party target attack range: other slot outside party slots")
	}
	for _, v := range in[52:60] {
		if v < 0 || v > 4 { // five formation areas, 4 is the center.
			panic("party target attack range: quadrant outside 0..4")
		}
	}
	for _, v := range in[60:63] {
		if v != 255 && (v < 0 || v > 7) { // 255 is an empty center slot.
			panic("party target attack range: center slot outside party slots")
		}
	}
	for _, v := range in[63:71] {
		if v < 0 || v > 255 {
			panic("party target attack range: flag is not u8")
		}
	}
	a, b := in[0], in[51]
	if a == b { // 0x5197ce.
		return []int64{1}
	}
	reach := actionAttackRangeNode(in[:51], r)[0]
	if reach != 0 {
		if reach == -1 {
			return []int64{0}
		}
		return []int64{1}
	}
	qa, qb := in[52+a], in[52+b]
	d := qa - qb
	if d < 0 {
		d = -d
	}
	if qa == qb || qa == 4 || qb == 4 || d != 2 {
		return []int64{1}
	}
	for _, c := range in[60:63] {
		if c != 255 && in[63+c] == 0 { // an available center member blocks.
			return []int64{0}
		}
	}
	return []int64{1}
}

func init() {
	Nodes["005197c0"] = Node{Addr: "005197c0", In: []string{
		"characterSlot", "hand", "attackContext", "characterPresent", "characterHP", "characterCondition", "characterInParty",
		"mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "mainItemIndex", "offItemIndex",
		"item0Range", "item1Range", "item2Range", "item3Range", "item0Spell", "item1Spell", "item2Spell", "item3Spell",
		"mainActionKind", "offActionKind", "battleActionKind", "mainSpell", "offSpell", "queuedSpell", "battleSpell", "specialBaseSpell", "specialOverrideSpell",
		"spell0Range", "spell1Range", "spell2Range", "spell3Range", "spell4Range", "spell5Range", "spell6Range", "spell7Range", "spell8Range", "spell9Range",
		"combatMode", "specialSpellMode", "specialItemMode", "selectedCharacterSlot", "phaseType", "battleSpecialFlag", "itemEntriesPointer", "selectedItemIndex", "specialItemOverridePointer", "specialBaseItemIndex", "specialOverrideItemIndex",
		"targetSlot",
		"character0Quadrant", "character1Quadrant", "character2Quadrant", "character3Quadrant", "character4Quadrant", "character5Quadrant", "character6Quadrant", "character7Quadrant",
		"centerSlot0", "centerSlot1", "centerSlot2",
		"character0FormationUnavailable", "character1FormationUnavailable", "character2FormationUnavailable", "character3FormationUnavailable", "character4FormationUnavailable", "character5FormationUnavailable", "character6FormationUnavailable", "character7FormationUnavailable",
	}, Out: []string{"canAttack"}, F: partyTargetAttackRangeNode}
}
