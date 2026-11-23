package strlib

import (
	"bytes"
	"encoding/binary"
	hexencoding "encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf16"
)

type Entry struct {
	A    uint32 `json:"a"`
	B    uint32 `json:"b"`
	Text string `json:"text"`
	Nul  bool   `json:"nul"`
}

type Library struct {
	Header  string  `json:"header"`
	Tail    string  `json:"tail"`
	Entries []Entry `json:"entries"`
}

func DecodeFile(path string, out io.Writer) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	l, err := decodeLibrary(raw)
	if err != nil {
		return err
	}
	// Validate the entire library before emitting any JSON.
	return json.NewEncoder(out).Encode(l)
}

func decodeLibrary(raw []byte) (Library, error) {
	l := Library{Entries: make([]Entry, 0)}
	size := int64(len(raw))
	if size < 13 {
		return l, fmt.Errorf("file too short")
	}
	if raw[4] > 1 {
		return l, fmt.Errorf("invalid header flag %d", raw[4])
	}
	l.Header = hexencoding.EncodeToString(raw[:5])
	count := int64(binary.LittleEndian.Uint32(raw[size-8:]))
	end := int64(binary.LittleEndian.Uint32(raw[size-4:]))
	table := size - 8 - 4*count
	tailStart := table
	l.Tail = "dbs"
	if end != 0 {
		l.Tail = "msg"
		tailStart -= 4
	}
	// Every entry needs at least its three u32 fields; do not allocate
	// from an untrusted count before proving it fits in the file.
	if tailStart < 5 || count > (tailStart-5)/12 {
		return l, fmt.Errorf("entry count or offset table out of bounds")
	}
	if l.Tail == "msg" {
		if end != tailStart || binary.LittleEndian.Uint32(raw[tailStart:]) != 0xffffffff {
			return l, fmt.Errorf("invalid msg tail")
		}
	}
	pos := int64(5)
	for i := int64(0); i < count; i++ {
		offset := int64(binary.LittleEndian.Uint32(raw[table+4*i:]))
		if offset != pos || tailStart-pos < 12 {
			return l, fmt.Errorf("entry %d: offset chain mismatch or truncated entry", i)
		}
		e := Entry{
			A: binary.LittleEndian.Uint32(raw[pos:]),
			B: binary.LittleEndian.Uint32(raw[pos+4:]),
		}
		n := int64(binary.LittleEndian.Uint32(raw[pos+8:]))
		pos += 12
		if n > 2000 || n*2 > tailStart-pos {
			return l, fmt.Errorf("entry %d: unit count out of bounds", i)
		}
		units := make([]uint16, int(n))
		for j := range units {
			u := binary.LittleEndian.Uint16(raw[pos+int64(j)*2:])
			if raw[4] == 1 {
				u = uint16(0x9696) - u
			}
			units[j] = u
		}
		pos += n * 2
		if n > 0 {
			if units[len(units)-1] != 0 {
				return l, fmt.Errorf("entry %d: missing terminating NUL", i)
			}
			e.Nul = true
			units = units[:len(units)-1]
		}
		for j := 0; j < len(units); j++ {
			u := units[j]
			switch {
			case u == 0:
				return l, fmt.Errorf("entry %d: internal NUL", i)
			case u >= 0xd800 && u <= 0xdbff:
				if j+1 >= len(units) || units[j+1] < 0xdc00 || units[j+1] > 0xdfff {
					return l, fmt.Errorf("entry %d: isolated high surrogate", i)
				}
				j++
			case u >= 0xdc00 && u <= 0xdfff:
				return l, fmt.Errorf("entry %d: isolated low surrogate", i)
			}
		}
		e.Text = string(utf16.Decode(units))
		l.Entries = append(l.Entries, e)
	}
	if pos != tailStart {
		return l, fmt.Errorf("last entry does not end at tail")
	}
	return l, nil
}

func EncodeFile(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("expected JSON object")
	}
	var l Library
	if err := json.Unmarshal(raw, &l); err != nil {
		return err
	}
	encoded, err := encodeLibrary(l)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, encoded, 0o644)
}

func encodeLibrary(l Library) ([]byte, error) {
	header, err := hexencoding.DecodeString(l.Header)
	if err != nil || len(header) != 5 {
		return nil, fmt.Errorf("header must be 5 bytes of hex")
	}
	if header[4] > 1 {
		return nil, fmt.Errorf("invalid header flag %d", header[4])
	}
	if l.Tail != "dbs" && l.Tail != "msg" {
		return nil, fmt.Errorf("tail must be dbs or msg")
	}
	count := int64(len(l.Entries))
	size := int64(5) + 8 + 4*count
	if l.Tail == "msg" {
		size += 4
	}
	for i, e := range l.Entries {
		if strings.ContainsRune(e.Text, 0) || (e.Text != "" && !e.Nul) {
			return nil, fmt.Errorf("entry %d: internal NUL or missing terminating NUL", i)
		}
		n := 0
		for _, r := range e.Text {
			n += utf16.RuneLen(r)
		}
		if e.Nul {
			n++
		}
		if n > 2000 {
			return nil, fmt.Errorf("entry %d: unit count exceeds 2000", i)
		}
		size += 12 + int64(n)*2
	}
	if count > 0xffffffff || size > 0xffffffff || size > int64(int(^uint(0)>>1)) {
		return nil, fmt.Errorf("library too large for u32 offsets")
	}
	raw := make([]byte, 0, int(size))
	raw = append(raw, header...)
	offsets := make([]uint32, len(l.Entries))
	for i, e := range l.Entries {
		offsets[i] = uint32(len(raw))
		units := utf16.Encode([]rune(e.Text))
		if e.Nul {
			units = append(units, 0)
		}
		raw = binary.LittleEndian.AppendUint32(raw, e.A)
		raw = binary.LittleEndian.AppendUint32(raw, e.B)
		raw = binary.LittleEndian.AppendUint32(raw, uint32(len(units)))
		for _, u := range units {
			if header[4] == 1 {
				u = uint16(0x9696) - u
			}
			raw = binary.LittleEndian.AppendUint16(raw, u)
		}
	}
	end := uint32(0)
	if l.Tail == "msg" {
		end = uint32(len(raw))
		raw = binary.LittleEndian.AppendUint32(raw, 0xffffffff)
	}
	for _, offset := range offsets {
		raw = binary.LittleEndian.AppendUint32(raw, offset)
	}
	raw = binary.LittleEndian.AppendUint32(raw, uint32(count))
	raw = binary.LittleEndian.AppendUint32(raw, end)
	return raw, nil
}
