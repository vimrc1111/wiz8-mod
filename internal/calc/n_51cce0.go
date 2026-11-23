package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// The classification interval and return targets come from 0x51cd03..0x51cd1f.
// The selection table itself is loaded from the player's exe, never embedded.
const weaponBonusFirstClass = 11
const weaponBonusLastClass = 131

var weaponBonusClasses, weaponBonusLoadErr = loadWeaponBonusClasses()

func loadWeaponBonusClasses() ([121]bool, error) {
	var classes [121]bool
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		// Match the emulator's repository-local, user-supplied executable.
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return classes, fmt.Errorf("weapon damage bonus: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return classes, fmt.Errorf("weapon damage bonus: %w", err)
	}
	read := func(va uint32, size int) ([]byte, error) {
		offset, err := peOffset(b, va)
		if err != nil {
			return nil, err
		}
		if size > len(b)-offset {
			return nil, fmt.Errorf("weapon damage bonus: table %#x truncated", va)
		}
		return b[offset : offset+size], nil
	}
	// 0x51cd20: two dispatch targets; 0x51cd28: one selector per class.
	targets, err := read(0x51cd20, 8)
	if err != nil {
		return classes, err
	}
	selectors, err := read(0x51cd28, len(classes))
	if err != nil {
		return classes, err
	}
	for i, selector := range selectors {
		if selector > 1 {
			return classes, fmt.Errorf("weapon damage bonus: invalid selector at class %d", i+weaponBonusFirstClass)
		}
		target := binary.LittleEndian.Uint32(targets[4*int(selector):])
		switch target {
		case 0x51cd1a: // Special weapon: import damage percentage points (F28).
			classes[i] = true
		case 0x51cd1d: // Ordinary item: no percentage-point import.
		default:
			return classes, fmt.Errorf("weapon damage bonus: unknown return target %#x", target)
		}
	}
	return classes, nil
}

// weaponDamageBonusItemNode is pure after package initialization. itemIndex is
// an i32 record index (-1 means no item); itemClass is u16 I+0x3f. Record stride
// 269 and the base pointer 0x68516c affect addressing only (formulas.md:9).
// The observable result is AL, the boolean consumed by F28 (formulas.md:71);
// upper EAX bits are not part of that interface.
func weaponDamageBonusItemNode(in []int64, r Rand) []int64 {
	if len(in) != 2 || in[0] < -1 || in[0] > 2147483647 || in[1] < 0 || in[1] > 65535 {
		panic("weapon damage bonus: expected itemIndex -1..2147483647 and u16 itemClass")
	}
	if in[0] == -1 || in[1] < weaponBonusFirstClass || in[1] > weaponBonusLastClass {
		return []int64{0}
	}
	if weaponBonusLoadErr != nil {
		panic(weaponBonusLoadErr)
	}
	if weaponBonusClasses[in[1]-weaponBonusFirstClass] {
		return []int64{1}
	}
	return []int64{0}
}

func init() {
	Nodes["0051cce0"] = Node{
		Addr: "0051cce0", In: []string{"itemIndex", "itemClass"},
		Out: []string{"importsDamageBonus"}, F: weaponDamageBonusItemNode,
	}
}
