package stringdata

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const stringKey uint16 = 0x9696

func decodeData(b []byte) ([]string, error) {
	if len(b) < 4 {
		return nil, fmt.Errorf("檔案缺少筆數")
	}
	n := int64(binary.LittleEndian.Uint32(b))
	end := int64(len(b))
	// Every record requires a length field and at least one NUL unit.
	if n > (end-4)/6 {
		return nil, fmt.Errorf("筆數 %d 超過實際內容", n)
	}
	ss := make([]string, 0, int(n))
	pos := int64(4)
	for i := int64(0); i < n; i++ {
		if end-pos < 4 {
			return nil, fmt.Errorf("第 %d 筆缺少長度", i)
		}
		length := int64(binary.LittleEndian.Uint32(b[pos:]))
		pos += 4
		if length == 0 || length%2 != 0 || length > end-pos {
			return nil, fmt.Errorf("第 %d 筆長度 %d 無效或越界", i, length)
		}
		u := make([]uint16, int(length/2))
		for j := range u {
			u[j] = stringKey - binary.LittleEndian.Uint16(b[pos+int64(j)*2:])
		}
		pos += length
		if u[len(u)-1] != 0 {
			return nil, fmt.Errorf("第 %d 筆沒有 NUL 結尾", i)
		}
		u = u[:len(u)-1]
		for j := 0; j < len(u); j++ {
			c := u[j]
			switch {
			case c == 0:
				return nil, fmt.Errorf("第 %d 筆含內部 NUL", i)
			case c >= 0xD800 && c <= 0xDBFF:
				if j+1 >= len(u) || u[j+1] < 0xDC00 || u[j+1] > 0xDFFF {
					return nil, fmt.Errorf("第 %d 筆含孤立 surrogate", i)
				}
				j++
			case c >= 0xDC00 && c <= 0xDFFF:
				return nil, fmt.Errorf("第 %d 筆含孤立 surrogate", i)
			}
		}
		ss = append(ss, string(utf16.Decode(u)))
	}
	if pos != end {
		return nil, fmt.Errorf("檔案含尾隨資料")
	}
	return ss, nil
}

func DecodeFile(path string, out io.Writer) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	ss, err := decodeData(b)
	if err != nil {
		return err
	}
	// Validate the entire file before emitting any JSON.
	return json.NewEncoder(out).Encode(ss)
}

func encodeJSON(b []byte) ([]byte, error) {
	if !utf8.Valid(b) {
		return nil, fmt.Errorf("JSON 含非法 UTF-8")
	}
	b = bytes.TrimSpace(b)
	if len(b) == 0 || b[0] != '[' {
		return nil, fmt.Errorf("JSON 必須是字串陣列")
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, err
	}
	if uint64(len(entries)) > math.MaxUint32 {
		return nil, fmt.Errorf("筆數超過 u32 範圍")
	}
	data := binary.LittleEndian.AppendUint32(nil, uint32(len(entries)))
	for i, entry := range entries {
		if len(entry) == 0 || entry[0] != '"' {
			return nil, fmt.Errorf("第 %d 筆不是字串", i)
		}
		var s string
		if err := json.Unmarshal(entry, &s); err != nil {
			return nil, fmt.Errorf("第 %d 筆: %w", i, err)
		}
		if !utf8.ValidString(s) || strings.ContainsRune(s, 0) {
			return nil, fmt.Errorf("第 %d 筆含 NUL 或非法 UTF-8", i)
		}
		u := utf16.Encode([]rune(s))
		length := (uint64(len(u)) + 1) * 2
		if length > math.MaxUint32 {
			return nil, fmt.Errorf("第 %d 筆長度超過 u32 範圍", i)
		}
		data = binary.LittleEndian.AppendUint32(data, uint32(length))
		for _, c := range u {
			data = binary.LittleEndian.AppendUint16(data, stringKey-c)
		}
		data = binary.LittleEndian.AppendUint16(data, stringKey)
	}
	return data, nil
}

func EncodeFile(src, dest string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	data, err := encodeJSON(b)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}
