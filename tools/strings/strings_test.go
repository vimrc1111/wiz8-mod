// 驗收測試（cc 寫，實作方不准改）。格式 oracle 在本檔自寫，不採信工具輸出。
// 真實檔案測試在找不到遊戲目錄時 t.Skip；目錄可用 WIZ8_DIR 覆寫。
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

// 格式 oracle：u32 筆數；每筆 u32 位元組長度＋該長度的 u16，每個 u16 存 0x9696−c（含結尾 NUL）。
func buildDat(ss []string) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint32(len(ss)))
	for _, s := range ss {
		u := append(utf16.Encode([]rune(s)), 0)
		binary.Write(&b, binary.LittleEndian, uint32(2*len(u)))
		for _, c := range u {
			binary.Write(&b, binary.LittleEndian, uint16(0x9696-c))
		}
	}
	return b.Bytes()
}

func writeTmp(t *testing.T, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func decode(t *testing.T, p string) []string {
	t.Helper()
	var out bytes.Buffer
	if code, err := run([]string{"decode", p}, &out); code != 0 || err != nil {
		t.Fatalf("decode %s: code=%d err=%v", p, code, err)
	}
	var ss []string
	if err := json.Unmarshal(out.Bytes(), &ss); err != nil {
		t.Fatalf("decode 輸出不是 JSON 字串陣列：%v", err)
	}
	return ss
}

func roundTrip(t *testing.T, raw []byte) {
	t.Helper()
	var out bytes.Buffer
	if code, err := run([]string{"decode", writeTmp(t, "in.dat", raw)}, &out); code != 0 || err != nil {
		t.Fatalf("decode: code=%d err=%v", code, err)
	}
	dst := filepath.Join(t.TempDir(), "out.dat")
	if code, err := run([]string{"encode", writeTmp(t, "in.json", out.Bytes()), dst}, &bytes.Buffer{}); code != 0 || err != nil {
		t.Fatalf("encode: code=%d err=%v", code, err)
	}
	got, _ := os.ReadFile(dst)
	if !bytes.Equal(got, raw) {
		t.Fatalf("encode(decode(x)) != x：%d vs %d bytes", len(got), len(raw))
	}
}

var fixture = []string{"Zoom In (ENTER)", "", "新隊伍", "𠀀 surrogate pair", "tab\tquote\"back\\slash"}

func TestFixtureDecode(t *testing.T) {
	got := decode(t, writeTmp(t, "f.dat", buildDat(fixture)))
	if len(got) != len(fixture) {
		t.Fatalf("筆數 %d，應 %d", len(got), len(fixture))
	}
	for i := range fixture {
		if got[i] != fixture[i] {
			t.Errorf("[%d] = %q，應 %q", i, got[i], fixture[i])
		}
	}
}

func TestFixtureRoundTrip(t *testing.T) { roundTrip(t, buildDat(fixture)) }

// encode 是翻譯管線的入口：JSON 改了字，輸出要照 oracle 編。
func TestEncodeFromJSON(t *testing.T) {
	ss := []string{"新隊伍", "Exit"}
	j, _ := json.Marshal(ss)
	dst := filepath.Join(t.TempDir(), "o.dat")
	if code, err := run([]string{"encode", writeTmp(t, "s.json", j), dst}, &bytes.Buffer{}); code != 0 || err != nil {
		t.Fatalf("encode: code=%d err=%v", code, err)
	}
	if got, _ := os.ReadFile(dst); !bytes.Equal(got, buildDat(ss)) {
		t.Fatalf("encode 輸出與 oracle 不同")
	}
}

func TestRejectsMalformed(t *testing.T) {
	good := buildDat([]string{"ab", "c"})
	cases := map[string][]byte{
		"short":       good[:2],
		"count-extra": func() []byte { b := bytes.Clone(good); binary.LittleEndian.PutUint32(b, 3); return b }(),
		"len-oob":     func() []byte { b := bytes.Clone(good); binary.LittleEndian.PutUint32(b[4:], 0x7FFFFFFE); return b }(),
		"len-odd":     func() []byte { b := bytes.Clone(good); binary.LittleEndian.PutUint32(b[4:], 5); return b }(),
		"trailing":    append(bytes.Clone(good), 0),
		"no-nul": func() []byte { // 第一筆最後一個 unit 不是 NUL
			b := bytes.Clone(good)
			binary.LittleEndian.PutUint16(b[8+4:], 0x9696-'x')
			return b
		}(),
		"lone-surrogate": buildRaw([][]uint16{{0xD800, 0}}),
		// grok 稽核 C（8yPB F1）：拿掉內部 NUL 檢查的實作原本全綠
		"inner-nul":     buildRaw([][]uint16{{'a', 0, 'b', 0}}),
		"len0-sibling":  buildRaw([][]uint16{{}, {'A', 0}}),
		"lone-low-surr": buildRaw([][]uint16{{0xDC00, 0}}),
	}
	for name, raw := range cases {
		var out bytes.Buffer
		code, err := run([]string{"decode", writeTmp(t, name+".dat", raw)}, &out)
		if code == 0 || err == nil || out.Len() != 0 {
			t.Errorf("%s: 應拒絕且無輸出，得 code=%d err=%v out=%d bytes", name, code, err, out.Len())
		}
	}
	for name, j := range map[string]string{"not-array": `{"a":1}`, "not-string": `[1]`, "has-nul": `["a\u0000b"]`} {
		code, err := run([]string{"encode", writeTmp(t, name+".json", []byte(j)), filepath.Join(t.TempDir(), "o.dat")}, &bytes.Buffer{})
		if code == 0 || err == nil {
			t.Errorf("encode %s: 應拒絕，得 code=%d", name, code)
		}
	}
}

func buildRaw(recs [][]uint16) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint32(len(recs)))
	for _, u := range recs {
		binary.Write(&b, binary.LittleEndian, uint32(2*len(u)))
		for _, c := range u {
			binary.Write(&b, binary.LittleEndian, uint16(0x9696-c))
		}
	}
	return b.Bytes()
}

// 從 PATCH.001 目錄（A1 格式：檔尾 n×0x118，n 在 +0x200）取 STRINGS\STRINGDATA.DAT。
func realStringData(t *testing.T) []byte {
	t.Helper()
	dir := os.Getenv("WIZ8_DIR")
	if dir == "" {
		dir = "/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"
	}
	b, err := os.ReadFile(filepath.Join(dir, "Patches", "PATCH.001"))
	if err != nil {
		t.Skipf("skip：找不到遊戲（%v）", err)
	}
	n := int(binary.LittleEndian.Uint32(b[0x200:]))
	for i := 0; i < n; i++ {
		r := b[len(b)-(n-i)*0x118:]
		if string(r[:bytes.IndexByte(r[:0x100], 0)]) == `STRINGS\STRINGDATA.DAT` {
			off, size := binary.LittleEndian.Uint32(r[0x100:]), binary.LittleEndian.Uint32(r[0x104:])
			return b[off : off+size]
		}
	}
	t.Fatal("PATCH.001 沒有 STRINGS\\STRINGDATA.DAT")
	return nil
}

func TestRealRoundTrip(t *testing.T) { roundTrip(t, realStringData(t)) }

// 期望值：cc 以獨立 Python 實作 decode 後抄下（截圖上的主選單項是 INTROTEXT_*.STI 點陣圖，不在字串庫）。
func TestRealKnownStrings(t *testing.T) {
	ss := decode(t, writeTmp(t, "s.dat", realStringData(t)))
	want := map[int]string{0: "Zoom In (ENTER)", 1: "Zoom Out (BACKSPACE)", 22: "Exit", 1718: "New Game Options", 2411: "A specific item is required here."}
	if len(ss) != 2412 {
		t.Fatalf("筆數 %d，應 2412", len(ss))
	}
	for i, w := range want {
		if ss[i] != w {
			t.Errorf("[%d] = %q，應 %q", i, ss[i], w)
		}
	}
}
