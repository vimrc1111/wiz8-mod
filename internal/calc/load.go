package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"unicode/utf16"
)

func records(path string, header, stride int) ([]byte, int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	if len(b) < header {
		return nil, 0, fmt.Errorf("calc: %s: short header", path)
	}
	count := binary.LittleEndian.Uint32(b)
	expected := uint64(header) + uint64(count)*uint64(stride)
	if uint64(len(b)) != expected {
		return nil, 0, fmt.Errorf("calc: %s: length %d, want %d for %d records", path, len(b), expected, count)
	}
	return b[header:], int(count), nil
}
func nameUTF16(b []byte) string {
	units := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u := binary.LittleEndian.Uint16(b[i:])
		if u == 0 {
			break
		}
		units = append(units, u)
	}
	return string(utf16.Decode(units))
}
func readDice(b []byte) Dice { return Dice{int16(binary.LittleEndian.Uint16(b)), b[2], b[3]} }
func i32(b []byte) int32     { return int32(binary.LittleEndian.Uint32(b)) }
func LoadSpells(path string) ([]Spell, error) {
	b, n, err := records(path, 8, 704)
	if err != nil {
		return nil, err
	}
	out := make([]Spell, n)
	for i := range out {
		r := b[i*704 : (i+1)*704]
		out[i] = Spell{Name: nameUTF16(r[0x19c:0x21c]), Dice: readDice(r[0x152:]), ManaPerPL: i32(r[0x14a:]), Level: i32(r[0x157:]), Realm: binary.LittleEndian.Uint32(r[0x234:]), Range: i32(r[0x230:]), Type: i32(r[0x238:])}
	}
	return out, nil
}
func LoadItems(path string) ([]Item, error) {
	b, n, err := records(path, 4, 0x10d)
	if err != nil {
		return nil, err
	}
	out := make([]Item, n)
	for i := range out {
		r := b[i*0x10d : (i+1)*0x10d]
		out[i] = Item{Name: nameUTF16(r[:0x3c]), Dice: readDice(r[0x4a:]), SpeedMod: int8(r[0x48]), HitMod: int8(r[0x49]), ModeMask: binary.LittleEndian.Uint16(r[0x4e:]), ClassMask: binary.LittleEndian.Uint16(r[0x76:])}
		copy(out[i].Effects[:], r[0x50:0x60])
	}
	return out, nil
}
func LoadMonsters(path string) ([]Monster, error) {
	b, n, err := records(path, 4, 0x297)
	if err != nil {
		return nil, err
	}
	out := make([]Monster, n)
	for i := range out {
		r := b[i*0x297 : (i+1)*0x297]
		out[i] = Monster{Name: nameUTF16(r[:0x30]), HP: readDice(r[0xd6:]), AC: int8(r[0x15d]), Level: r[0x24f], Absorb: r[0xcf]}
		copy(out[i].Attr[:], r[0xd1:0xd6])
		copy(out[i].Resist[:], r[0x176:0x17c])
	}
	return out, nil
}
