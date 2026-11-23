package nsf

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"unicode"
	"unicode/utf16"
)

type Segment struct {
	Raw  *string  `json:"raw,omitempty"`
	Text []string `json:"text,omitempty"`
	Kw   *string  `json:"kw,omitempty"`
}

type Script struct {
	Segments []Segment `json:"segments"`
}

func DecodeFile(path string, out io.Writer) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s, err := decodeScript(b)
	if err != nil {
		return err
	}
	// Finish validation before writing anything to stdout.
	return json.NewEncoder(out).Encode(s)
}

// readText recognizes complete, nonempty chunks at an unaligned offset.
// Printable Unicode permits translated files to be decoded again; U+0092
// is retained because the original scripts use that control code as text.
func readText(b []byte, pos int) ([]string, int, bool) {
	if pos >= len(b) || b[pos] < 1 || b[pos] > 4 {
		return nil, pos, false
	}
	m := int(b[pos])
	pos++
	chunks := make([]string, 0, m)
	for range m {
		if len(b)-pos < 2 {
			return nil, pos, false
		}
		n := int(binary.LittleEndian.Uint16(b[pos:]))
		pos += 2
		if n == 0 || n > (len(b)-pos)/2 {
			return nil, pos, false
		}
		units := make([]uint16, n)
		for i := range units {
			units[i] = binary.LittleEndian.Uint16(b[pos+2*i:])
		}
		for i := 0; i < n; i++ {
			r := rune(units[i])
			if r >= 0xd800 && r <= 0xdbff {
				if i+1 == n || units[i+1] < 0xdc00 || units[i+1] > 0xdfff {
					return nil, pos, false
				}
				r = utf16.DecodeRune(r, rune(units[i+1]))
				i++
			} else if r >= 0xdc00 && r <= 0xdfff {
				return nil, pos, false
			}
			if r != '\t' && r != '\n' && r != 0x92 && !unicode.IsPrint(r) {
				return nil, pos, false
			}
		}
		chunks = append(chunks, string(utf16.Decode(units)))
		pos += 2 * n
	}
	return chunks, pos, true
}

func printableASCII(b []byte) bool {
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return false
		}
	}
	return true
}

func decodeScript(b []byte) (Script, error) {
	s := Script{Segments: make([]Segment, 0)}
	if len(b) < 26 {
		return s, fmt.Errorf("file too short")
	}
	if binary.LittleEndian.Uint32(b) != 0xffffffff {
		return s, fmt.Errorf("invalid magic")
	}
	chunks, end, ok := readText(b, 26)
	if !ok {
		return s, fmt.Errorf("invalid or truncated first text at offset 26")
	}
	addRaw := func(start, end int) {
		if start < end {
			h := hex.EncodeToString(b[start:end])
			s.Segments = append(s.Segments, Segment{Raw: &h})
		}
	}
	addRaw(0, 26)
	s.Segments = append(s.Segments, Segment{Text: chunks})
	rawStart := end
	for p := end; p < len(b); {
		// Only the matching chunk count is known about the 12-byte
		// node leader. Its other fields, including tail variants, stay raw.
		if len(b)-p > 12 && b[p] >= 1 && b[p] <= 4 && b[p+12] == b[p] {
			if chunks, end, ok := readText(b, p+12); ok {
				addRaw(rawStart, p+12)
				s.Segments = append(s.Segments, Segment{Text: chunks})
				p, rawStart = end, end
				continue
			}
		}
		if len(b)-p >= 15 && b[p] >= 1 && b[p] <= 3 && binary.LittleEndian.Uint32(b[p+5:]) == 0 {
			n := int(binary.LittleEndian.Uint16(b[p+13:]))
			// Single-byte matches are ambiguous quad fields, not reliable
			// keyword definitions (one would consume a text node leader).
			if n >= 2 && n <= len(b)-p-15 && printableASCII(b[p+15:p+15+n]) {
				addRaw(rawStart, p+13)
				k := string(b[p+15 : p+15+n])
				s.Segments = append(s.Segments, Segment{Kw: &k})
				p = p + 15 + n
				rawStart = p
				continue
			}
		}
		p++
	}
	addRaw(rawStart, len(b))
	return s, nil
}

func EncodeFile(src, dst string) error {
	j, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	b, err := encodeScript(j)
	if err != nil {
		return err
	}
	// Validate and build the entire file before opening the destination.
	return os.WriteFile(dst, b, 0o644)
}

func encodeScript(j []byte) ([]byte, error) {
	j = bytes.TrimSpace(j)
	if len(j) == 0 || j[0] != '{' {
		return nil, fmt.Errorf("expected JSON object")
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(j, &obj); err != nil {
		return nil, err
	}
	array := bytes.TrimSpace(obj["segments"])
	if len(array) == 0 || array[0] != '[' {
		return nil, fmt.Errorf("segments must be an array")
	}
	var segments []map[string]json.RawMessage
	if err := json.Unmarshal(array, &segments); err != nil {
		return nil, err
	}
	var b []byte
	for i, s := range segments {
		if len(s) != 1 {
			return nil, fmt.Errorf("segment %d: expected exactly one field", i)
		}
		for kind, value := range s {
			switch kind {
			case "raw", "kw":
				var str string
				if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
					return nil, fmt.Errorf("segment %d: %s must be a string", i, kind)
				}
				if err := json.Unmarshal(value, &str); err != nil {
					return nil, fmt.Errorf("segment %d: %w", i, err)
				}
				if kind == "raw" {
					r, err := hex.DecodeString(str)
					if err != nil {
						return nil, fmt.Errorf("segment %d: %w", i, err)
					}
					b = append(b, r...)
				} else {
					if len(str) == 0 || len(str) > 65535 || !printableASCII([]byte(str)) {
						return nil, fmt.Errorf("segment %d: keyword must be 1–65535 printable ASCII bytes", i)
					}
					b = binary.LittleEndian.AppendUint16(b, uint16(len(str)))
					b = append(b, str...)
				}
			case "text":
				var chunks []string
				if err := json.Unmarshal(value, &chunks); err != nil {
					return nil, fmt.Errorf("segment %d: %w", i, err)
				}
				if len(chunks) < 1 || len(chunks) > 4 {
					return nil, fmt.Errorf("segment %d: text must have 1–4 chunks", i)
				}
				b = append(b, byte(len(chunks)))
				for c, str := range chunks {
					units := utf16.Encode([]rune(str))
					if len(units) == 0 || len(units) > 65535 {
						return nil, fmt.Errorf("segment %d chunk %d: expected 1–65535 UTF-16 units", i, c)
					}
					b = binary.LittleEndian.AppendUint16(b, uint16(len(units)))
					for _, u := range units {
						b = binary.LittleEndian.AppendUint16(b, u)
					}
				}
			default:
				return nil, fmt.Errorf("segment %d: unknown field %q", i, kind)
			}
		}
	}
	return b, nil
}
