package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// lockPinChanceTable is the 8-entry per-mille chance table at 0x64ba90
// (indexed by spell power), read from the user's Wiz8.exe.
var lockPinChanceTable, lockPinChanceError = loadLockPinChanceTable()

func loadLockPinChanceTable() ([8]uint32, error) {
	var t [8]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("lock pin chance: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	off, err := peOffset(b, 0x64ba90)
	if err != nil {
		return t, err
	}
	if off > len(b)-32 {
		return t, fmt.Errorf("lock pin chance: truncated table")
	}
	for i := range t {
		t[i] = binary.LittleEndian.Uint32(b[off+4*i:])
	}
	return t, nil
}

func init() {
	in := []string{"power", "unlockFlag", "pinCount"}
	for i := 0; i < 8; i++ {
		in = append(in, fmt.Sprintf("slot%d", i))
	}
	in = append(in, "charIndex", "slotActive", "currentHP", "mainStatus", "toolByte", "toolValue",
		"learnedSpell", "stamina", "strength")
	for i := 0; i < 6; i++ {
		in = append(in, fmt.Sprintf("realmSP%d", i))
	}
	in = append(in, "spellRealm", "spellCost", "difficulty", "uiCount", "anchor20", "anchor24", "uiMode")
	var out []string
	for i := 0; i < 8; i++ {
		out = append(out, fmt.Sprintf("slotOut%d", i))
	}
	for i := 0; i < 8; i++ {
		out = append(out, fmt.Sprintf("slotByte%d", i))
	}
	for i := 0; i < 8; i++ {
		for _, o := range []string{"4", "34", "35", "36", "37", "38"} {
			out = append(out, fmt.Sprintf("pin%d_%s", i, o))
		}
	}
	out = append(out, "pinsDirty", "A_5", "A_34", "C_5", "C_34", "D20_4", "D20_a4", "D24_4", "D24_a4",
		"dlgState", "processFlags")
	Nodes["005871a0"] = Node{Addr: "005871a0", In: in, Out: out, F: lockSpellPinChanceNode}
}

// lockSpellPinChanceNode: the pin-lock spell dialog. Case wiring (not game data):
// untouched output cells are prefilled 0x5a (pin and slot bytes, A and C flags;
// the A dirty byte starts 0),
// 0x33 (D20/D24 +0xa4) and 0x10 (process flags); the sound player and the
// 0x585ed0 text refresh write nothing the outputs observe and are not modelled.
// The spell-0x27 school selection (0x4ff7f0) result is unused by this function.
func lockSpellPinChanceNode(in []int64, r Rand) []int64 {
	if len(in) != 33 {
		panic("lockSpellPinChance: expected 33 inputs")
	}
	power, unlock, n := int32(in[0]), in[1] != 0, int32(in[2])
	var slot [8]int32
	for i := range slot {
		slot[i] = int32(in[3+i])
	}
	ci, slotActive, hp, status := in[11], in[12], in[13], in[14]
	toolByte, toolValue := in[15], int32(in[16])
	learned, stamina, str := in[17], int32(in[18]), uint32(in[19])
	var realmSP [6]int32
	for i := range realmSP {
		realmSP[i] = int32(in[20+i])
	}
	spellRealm, spellCost, difficulty := in[26], int32(in[27]), int32(in[28])
	uiCount, anchor20, anchor24, uiMode := int32(in[29]), int32(in[30]), int32(in[31]), in[32]
	// The permutation lives in an 8-entry stack array; more pins smash the return
	// address, and the pin pointer table also holds 8 children.
	if n > 8 || uiCount > 8 || spellRealm > 5 {
		panic("lockSpellPinChance: pin count, container count or realm beyond the 8/8/6 slots")
	}
	var slotByte [8]byte
	var pins [8][6]byte // +4 +0x34 +0x35 +0x36 +0x37 +0x38
	for i := range slotByte {
		slotByte[i] = 0x5a
		pins[i] = [6]byte{0x5a, 0x5a, 0x5a, 0x5a, 0x5a, 0x5a}
	}
	pinsDirty, a5, a34, c5, c34 := byte(0), byte(0x5a), byte(0x5a), byte(0x5a), byte(0x5a)
	const (
		f4, f34, f35, f36, f37, f38 = 0, 1, 2, 3, 4, 5
	)
	var perm [8]int32
	for i := int32(0); i < n; i++ {
		perm[i] = i
	}
	for k := int32(0); k < n; k++ {
		a, b := r.R(int(n)), r.R(int(n))
		if a != b {
			perm[a], perm[b] = perm[b], perm[a]
		}
	}
	m := power + 1
	if m > n {
		m = n
	}
	changed := false
	if m > 0 {
		if lockPinChanceError != nil {
			panic(lockPinChanceError)
		}
		if power < 0 || power > 7 {
			panic("lockSpellPinChance: power outside the 8-entry chance table")
		}
		chance := lockPinChanceTable[power]
		for j := int32(0); j < m; j++ {
			idx := perm[j]
			if uint32(r.R(1000)) >= chance {
				continue
			}
			if unlock {
				if slot[idx] == -1 {
					continue
				}
				slotByte[idx], slot[idx] = 0, -1
				pins[idx][f37], pins[idx][f34], pins[idx][f36] = 0, 0, 1
				pinsDirty = 1
			} else {
				if slot[idx] == -1 {
					pins[idx][f38], pins[idx][f35] = 0, 1
					pinsDirty = 1
				}
				pins[idx][f37], slotByte[idx], slot[idx] = 1, 1, int32(ci)
			}
			changed = true
		}
	}
	process := int64(0x10)
	vcall := func(flag *byte, flag2 *byte) {
		*flag, *flag2 = 1, 1
		if uiMode == 7 {
			process |= 0x80000000
		}
	}
	if changed && unlock {
		vcall(&a5, &a34)
	}
	eligible := characterActionEligibilityNode([]int64{ci, slotActive, hp, status}, r)[0] != 0
	tool := false
	if eligible && (toolByte != 0 || toolValue != 0) {
		tool = toolValue >= 0
	}
	flag := byte(0)
	if tool {
		flag = 1
	}
	for i := int32(0); i < uiCount; i++ {
		pins[i][f4] = flag
	}
	canSpend := false
	if eligible && learned == 1 {
		canSpend = realmSP[spellRealm] >= spellCost
	}
	d20 := setPinAnchor(canSpend, anchor20)
	strong := false
	if eligible && stamina >= 0x50 && str > 0x32 {
		c := n + difficulty - 1
		if c < 2 {
			c = 2
		} else if c > 8 {
			c = 8
		}
		strong = int32((str-0x32)/(uint32(1)<<uint(c-2))) >= 0
	}
	d24 := setPinAnchor(strong, anchor24)
	// 0x585ed0 refresh: its view of the dialog calls the same slot-1 method.
	b5, b34 := byte(0), byte(0)
	vcall(&b5, &b34)
	vcall(&c5, &c34)
	o := make([]int64, 0, 75)
	for i := 0; i < 8; i++ {
		o = append(o, int64(slot[i]))
	}
	for i := 0; i < 8; i++ {
		o = append(o, int64(slotByte[i]))
	}
	for i := 0; i < 8; i++ {
		for _, v := range pins[i] {
			o = append(o, int64(v))
		}
	}
	o = append(o, int64(pinsDirty), int64(a5), int64(a34), int64(c5), int64(c34))
	o = append(o, d20[:]...)
	o = append(o, d24[:]...)
	return append(o, 7, process)
}

// setPinAnchor is virtual method 1 of the D20/D24 flag widgets (0x4f5410): it
// stores the flag, and when the widget's anchor field is not -1 also its state
// 4 (set) or 7 (clear). The state cell is prefilled 0x33 by the case wiring.
func setPinAnchor(set bool, anchor int32) [2]int64 {
	flag, state := int64(0), int64(0x33)
	if set {
		flag = 1
	}
	if anchor != -1 {
		state = 7
		if set {
			state = 4
		}
	}
	return [2]int64{flag, state}
}
