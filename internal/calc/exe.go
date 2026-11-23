package calc

import (
	"encoding/binary"
	"fmt"
	"os"
)

// ExeTables are the character tables read from the player's own Wiz8.exe
// (notes/re/growth.md, formulas.md F05/F06, abilities.md). Row order is the
// exe's: races Human..Mook (Lizardman=6), classes Fighter..Mage (Fighter=0).
type ExeTables struct {
	Race      [11][7]int // initial STR INT PIE VIT DEX SPD SEN
	ClassMin  [15][7]int
	MainSkill [15]int     // 0x6161dc: class primary skill (bonus max(1,b/4))
	ClassFeat [15][3]int  // 0x61507c: feature codes, -1 empty
	RaceFeat  [11][5]int  // 0x615130: feature codes, -1 empty
	SkillAttr [41][2]int  // 0x6155b0 +8/+12: controlling attributes
	Avail     [41][15]int // 0x615840: [skill][class], non-zero = usable
	PMult     [15]int     // F06 weights from 0x4ee210 index + jump targets
}

// HasFeature is 0x547940(C, code) for a party member: the class row, then the race row
// (code 0x1c also reads the per-character flag +0xa79, not modelled here).
func (t *ExeTables) HasFeature(class, race, code int) bool {
	for _, f := range t.ClassFeat[class] {
		if f == code {
			return true
		}
	}
	for _, f := range t.RaceFeat[race] {
		if f == code {
			return true
		}
	}
	return false
}

const (
	Fighter   = 0
	Lizardman = 6
)

// peOffset maps a VA to a file offset through the section table.
func peOffset(b []byte, va uint32) (int, error) {
	if len(b) < 0x40 {
		return 0, fmt.Errorf("calc: not a PE file")
	}
	pe := int(binary.LittleEndian.Uint32(b[0x3c:]))
	if pe+0xf8 > len(b) || string(b[pe:pe+4]) != "PE\x00\x00" {
		return 0, fmt.Errorf("calc: not a PE file")
	}
	n := int(binary.LittleEndian.Uint16(b[pe+6:]))
	opt := int(binary.LittleEndian.Uint16(b[pe+20:]))
	base := binary.LittleEndian.Uint32(b[pe+24+28:])
	rva := va - base
	for s := pe + 24 + opt; s+40 <= len(b) && n > 0; s, n = s+40, n-1 {
		v, size := binary.LittleEndian.Uint32(b[s+12:]), binary.LittleEndian.Uint32(b[s+16:])
		if rva >= v && rva < v+size {
			off := int(rva - v + binary.LittleEndian.Uint32(b[s+20:]))
			if off >= len(b) {
				break
			}
			return off, nil
		}
	}
	return 0, fmt.Errorf("calc: VA %#x not in file", va)
}

func LoadExe(path string) (*ExeTables, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t ExeTables
	at := func(va uint32, n int) ([]byte, error) {
		o, err := peOffset(b, va)
		if err != nil {
			return nil, err
		}
		if o+n > len(b) {
			return nil, fmt.Errorf("calc: %s: table %#x truncated", path, va)
		}
		return b[o : o+n], nil
	}
	ints := func(va uint32, dst []int) error {
		r, err := at(va, 4*len(dst))
		for i := range dst {
			if err == nil {
				dst[i] = int(i32(r[4*i:]))
			}
		}
		return err
	}
	for r := range t.Race {
		if err := ints(0x614cf0+28*uint32(r), t.Race[r][:]); err != nil {
			return nil, err
		}
	}
	for p := range t.ClassMin {
		if err := ints(0x614e24+28*uint32(p), t.ClassMin[p][:]); err != nil {
			return nil, err
		}
		if err := ints(0x61507c+12*uint32(p), t.ClassFeat[p][:]); err != nil {
			return nil, err
		}
	}
	for r := range t.RaceFeat {
		if err := ints(0x615130+20*uint32(r), t.RaceFeat[r][:]); err != nil {
			return nil, err
		}
	}
	if err := ints(0x6161dc, t.MainSkill[:]); err != nil {
		return nil, err
	}
	for s := range t.SkillAttr {
		if err := ints(0x6155b8+16*uint32(s), t.SkillAttr[s][:]); err != nil {
			return nil, err
		}
		if err := ints(0x615840+60*uint32(s), t.Avail[s][:]); err != nil {
			return nil, err
		}
	}
	// F06: idx byte per class selects a jump target that must be `mov ebp,imm32`.
	idx, err := at(0x4ee210, 15)
	if err != nil {
		return nil, err
	}
	jt, err := at(0x4ee204, 12)
	if err != nil {
		return nil, err
	}
	for p, k := range idx {
		if k > 2 {
			return nil, fmt.Errorf("calc: F06 index %d out of range", k)
		}
		code, err := at(binary.LittleEndian.Uint32(jt[4*k:]), 5)
		if err != nil {
			return nil, err
		}
		if code[0] != 0xbd {
			return nil, fmt.Errorf("calc: F06 target is not mov ebp,imm32 (unknown exe build?)")
		}
		t.PMult[p] = int(i32(code[1:]))
	}
	for r, row := range t.Race {
		sum := 0
		for _, v := range row {
			sum += v
		}
		if sum < 280 || sum > 330 {
			return nil, fmt.Errorf("calc: race %d sum %d implausible (unknown exe build?)", r, sum)
		}
	}
	return &t, nil
}
