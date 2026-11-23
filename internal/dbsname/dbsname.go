// Package dbsname preserves fixed-size DBS records while editing name fields.
package dbsname

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf16"
)

type Record struct {
	Raw  string   `json:"raw"`
	Text []string `json:"text"`
}

type Database struct {
	Head    string   `json:"head,omitempty"`
	Records []Record `json:"records"`
}

// maxText: the last units of an ITEMS/MONSTERS name field hold record data
// (items units 30–31, monsters unit 23), so new text plus its NUL must end before them.
type layout struct {
	header, stride, width, maxText int
	offsets                        []int
}

func kindLayout(kind string) (layout, error) {
	switch kind {
	case "items":
		return layout{4, 0x10d, 32, 29, []int{0}}, nil
	case "monsters":
		return layout{4, 0x297, 24, 22, []int{0, 0x30, 0x60, 0x90}}, nil
	case "spells":
		return layout{8, 0x2c0, 64, 63, []int{0x19c}}, nil
	default:
		return layout{}, fmt.Errorf("unknown kind %q: expected items, monsters or spells", kind)
	}
}

// Only units before the first NUL are text; trailing residue is opaque.
func decodeText(field []byte) (string, error) {
	units := make([]uint16, 0, len(field)/2)
	terminated := false
	for i := 0; i < len(field); i += 2 {
		u := binary.LittleEndian.Uint16(field[i:])
		if u == 0 {
			terminated = true
			break
		}
		units = append(units, u)
	}
	if !terminated {
		return "", fmt.Errorf("missing terminating NUL")
	}
	for i := 0; i < len(units); i++ {
		switch u := units[i]; {
		case u >= 0xd800 && u <= 0xdbff:
			if i+1 >= len(units) || units[i+1] < 0xdc00 || units[i+1] > 0xdfff {
				return "", fmt.Errorf("isolated high surrogate")
			}
			i++
		case u >= 0xdc00 && u <= 0xdfff:
			return "", fmt.Errorf("isolated low surrogate")
		}
	}
	return string(utf16.Decode(units)), nil
}

func DecodeFile(kind, in string, out io.Writer) error {
	l, err := kindLayout(kind)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	if len(raw) < l.header {
		return fmt.Errorf("file too short")
	}
	count := uint64(binary.LittleEndian.Uint32(raw))
	if uint64(len(raw)) != uint64(l.header)+count*uint64(l.stride) {
		return fmt.Errorf("record count does not match file length")
	}
	db := Database{Head: hex.EncodeToString(raw[4:l.header]), Records: make([]Record, int(count))}
	for i := range db.Records {
		r := raw[l.header+i*l.stride : l.header+(i+1)*l.stride]
		e := Record{Raw: hex.EncodeToString(r), Text: make([]string, len(l.offsets))}
		for j, off := range l.offsets {
			e.Text[j], err = decodeText(r[off : off+2*l.width])
			if err != nil {
				return fmt.Errorf("record %d text %d: %w", i, j, err)
			}
		}
		db.Records[i] = e
	}
	// Validate the entire database before emitting JSON.
	return json.NewEncoder(out).Encode(db)
}

func EncodeFile(kind, in, out string) error {
	l, err := kindLayout(kind)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("expected JSON object")
	}
	// RawMessage distinguishes null fields from empty strings.
	var db struct {
		Head    string `json:"head"`
		Records []struct {
			Raw  string            `json:"raw"`
			Text []json.RawMessage `json:"text"`
		} `json:"records"`
	}
	if err := json.Unmarshal(data, &db); err != nil {
		return err
	}
	if db.Records == nil {
		return fmt.Errorf("expected records array")
	}
	head, err := hex.DecodeString(db.Head)
	if err != nil || len(head) != l.header-4 {
		return fmt.Errorf("head must be %d bytes of hex", l.header-4)
	}
	count := uint64(len(db.Records))
	if count > 0xffffffff || uint64(l.header)+count*uint64(l.stride) > uint64(int(^uint(0)>>1)) {
		return fmt.Errorf("too many records")
	}
	encoded := make([]byte, l.header, int(uint64(l.header)+count*uint64(l.stride)))
	binary.LittleEndian.PutUint32(encoded, uint32(count))
	copy(encoded[4:], head)
	for i, e := range db.Records {
		r, err := hex.DecodeString(e.Raw)
		if err != nil || len(r) != l.stride {
			return fmt.Errorf("record %d: raw must be %d bytes of hex", i, l.stride)
		}
		if len(e.Text) != len(l.offsets) {
			return fmt.Errorf("record %d: text must contain exactly %d strings", i, len(l.offsets))
		}
		for j, off := range l.offsets {
			var s string
			t := bytes.TrimSpace(e.Text[j])
			if len(t) == 0 || t[0] != '"' {
				return fmt.Errorf("record %d text %d: expected string", i, j)
			}
			if err := json.Unmarshal(t, &s); err != nil {
				return fmt.Errorf("record %d text %d: %w", i, j, err)
			}
			if strings.ContainsRune(s, 0) {
				return fmt.Errorf("record %d text %d: internal NUL", i, j)
			}
			units := utf16.Encode([]rune(s))
			field := r[off : off+2*l.width]
			original, err := decodeText(field)
			if err != nil {
				return fmt.Errorf("record %d text %d: %w", i, j, err)
			}
			if s != original {
				if len(units) > l.maxText {
					return fmt.Errorf("record %d text %d: exceeds %d UTF-16 units", i, j, l.maxText)
				}
				// Text plus one NUL; bytes after it stay as in the original (record data lives there).
				for k, u := range append(units, 0) {
					binary.LittleEndian.PutUint16(field[2*k:], u)
				}
			}
		}
		encoded = append(encoded, r...)
	}
	// Invalid input must not create or truncate the destination.
	return os.WriteFile(out, encoded, 0o644)
}
