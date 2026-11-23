package fact

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

const factRecordSize, factTextSize = 472, 200

var factTextOffsets = [2]int{56, 256}

type Record struct {
	Raw  string   `json:"raw"`
	Text []string `json:"text"`
}

type Database struct {
	Records []Record `json:"records"`
}

// Only units before the first NUL are text; trailing buffer residue is opaque.
func decodeFactText(field []byte) (string, error) {
	units := make([]uint16, 0, factTextSize/2)
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

func DecodeFile(in string, out io.Writer) error {
	raw, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	if len(raw) < 4 {
		return fmt.Errorf("file too short")
	}
	count := uint64(binary.LittleEndian.Uint32(raw))
	if uint64(len(raw)) != 4+count*factRecordSize {
		return fmt.Errorf("record count does not match file length")
	}
	db := Database{Records: make([]Record, int(count))}
	for i := range db.Records {
		r := raw[4+i*factRecordSize : 4+(i+1)*factRecordSize]
		e := Record{Raw: hex.EncodeToString(r), Text: make([]string, 2)}
		for j, off := range factTextOffsets {
			e.Text[j], err = decodeFactText(r[off : off+factTextSize])
			if err != nil {
				return fmt.Errorf("record %d text %d: %w", i, j, err)
			}
		}
		db.Records[i] = e
	}
	// Validate every record before emitting anything to stdout.
	return json.NewEncoder(out).Encode(db)
}

func EncodeFile(in, out string) error {
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("expected JSON object")
	}
	// RawMessage keeps null text elements distinct from empty strings.
	var db struct {
		Records []struct {
			Raw  string            `json:"raw"`
			Text []json.RawMessage `json:"text"`
		} `json:"records"`
	}
	if err := json.Unmarshal(data, &db); err != nil {
		return err
	}
	count := uint64(len(db.Records))
	if count == 0 {
		return fmt.Errorf("expected records")
	}
	if count > 0xffffffff || 4+count*factRecordSize > uint64(int(^uint(0)>>1)) {
		return fmt.Errorf("too many records")
	}
	encoded := make([]byte, 4, int(4+count*factRecordSize))
	binary.LittleEndian.PutUint32(encoded, uint32(count))
	for i, e := range db.Records {
		r, err := hex.DecodeString(e.Raw)
		if err != nil || len(r) != factRecordSize {
			return fmt.Errorf("record %d: raw must be 472 bytes of hex", i)
		}
		if len(e.Text) != 2 {
			return fmt.Errorf("record %d: text must contain exactly two strings", i)
		}
		for j, off := range factTextOffsets {
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
			if len(units) > 99 {
				return fmt.Errorf("record %d text %d: exceeds 99 UTF-16 units", i, j)
			}
			field := r[off : off+factTextSize]
			original, err := decodeFactText(field)
			if err != nil {
				return fmt.Errorf("record %d text %d: %w", i, j, err)
			}
			if s != original {
				clear(field)
				for k, u := range units {
					binary.LittleEndian.PutUint16(field[2*k:], u)
				}
			}
		}
		encoded = append(encoded, r...)
	}
	// Invalid input must not create or truncate the destination.
	return os.WriteFile(out, encoded, 0o644)
}
