// 驗收測試（cc 寫，實作方不准改）。格式 oracle 在本檔自寫，不採信工具輸出。
// 真實檔案測試在找不到遊戲目錄時 t.Skip；目錄可用 WIZ8_DIR 覆寫。
// 真實檔的計數由 cc 以獨立掃描取得（tickets/J-nsf-tool.md「計數來源」）。
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf16"
)

type seg struct {
	Raw  *string  `json:"raw,omitempty"`
	Text []string `json:"text,omitempty"`
	Kw   *string  `json:"kw,omitempty"`
}

type nsf struct {
	Segments []seg `json:"segments"`
}

func raw(b ...[]byte) seg {
	s := hex.EncodeToString(bytes.Join(b, nil))
	return seg{Raw: &s}
}
func txt(c ...string) seg { return seg{Text: c} }
func kw(s string) seg     { return seg{Kw: &s} }

// 格式 oracle：段的位元組。raw＝原樣；text＝u8 chunk 數＋每 chunk（u16 unit 數＋UTF-16LE）；kw＝u16 位元組數＋ASCII。
func segBytes(s seg) []byte {
	var b bytes.Buffer
	switch {
	case s.Raw != nil:
		r, _ := hex.DecodeString(*s.Raw)
		b.Write(r)
	case s.Kw != nil:
		binary.Write(&b, binary.LittleEndian, uint16(len(*s.Kw)))
		b.WriteString(*s.Kw)
	default:
		b.WriteByte(byte(len(s.Text)))
		for _, c := range s.Text {
			u := utf16.Encode([]rune(c))
			binary.Write(&b, binary.LittleEndian, uint16(len(u)))
			binary.Write(&b, binary.LittleEndian, u)
		}
	}
	return b.Bytes()
}

func build(ss []seg) []byte {
	var b bytes.Buffer
	for _, s := range ss {
		b.Write(segBytes(s))
	}
	return b.Bytes()
}

func le32(v uint32) []byte { return binary.LittleEndian.AppendUint32(nil, v) }

// 合成檔（R7 附錄 A 版式）：檔頭＋檔前導＋節點前導＋文本、關鍵字定義、13B／18B 記錄、雙 chunk 文本（含 U+0092、LF）。
var fixture = []seg{
	raw(le32(0xffffffff), le32(2), []byte{0, 0, 0x28, 0x23, 0x2f, 0},
		[]byte{1}, le32(0x2f), le32(0x4b), []byte{5, 0, 0}),
	txt("Hi! I am Marten."),
	raw([]byte{1}, le32(0x4c), le32(0), le32(0x4c0001)),
	kw("MARTEN"),
	raw([]byte{3}, le32(765), le32(1), le32(9),
		[]byte{2}, le32(0x2f), le32(0x4c), []byte{7, 0, 0}),
	txt("First part,\nsecond line", "It\u0092s the end."),
	raw([]byte{8}, make([]byte, 17)),
}

func writeTmp(t *testing.T, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func decodeRaw(t *testing.T, b []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	if code, err := run([]string{"decode", writeTmp(t, "in.nsf", b)}, &out); code != 0 {
		t.Fatalf("decode exit %d：%v", code, err)
	}
	return out.Bytes()
}

func decodeSegs(t *testing.T, b []byte) []seg {
	t.Helper()
	var n nsf
	if err := json.Unmarshal(decodeRaw(t, b), &n); err != nil {
		t.Fatal(err)
	}
	return n.Segments
}

func encodeSegs(t *testing.T, ss []seg) []byte {
	t.Helper()
	j, _ := json.Marshal(nsf{ss})
	out := filepath.Join(t.TempDir(), "out.nsf")
	if code, err := run([]string{"encode", writeTmp(t, "in.json", j), out}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("encode exit %d：%v", code, err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func roundTrip(t *testing.T, name string, b []byte) []seg {
	t.Helper()
	ss := decodeSegs(t, b)
	if got := encodeSegs(t, ss); !bytes.Equal(got, b) {
		t.Fatalf("%s：往返不一致（%d → %d bytes）", name, len(b), len(got))
	}
	if !bytes.Equal(build(ss), b) {
		t.Fatalf("%s：段依 oracle 重組不等於原檔", name)
	}
	return ss
}

func TestFixtureDecode(t *testing.T) {
	got := decodeSegs(t, build(fixture))
	a, _ := json.Marshal(got)
	w, _ := json.Marshal(fixture)
	if !bytes.Equal(a, w) {
		t.Fatalf("decode\n得 %s\n應 %s", a, w)
	}
}

func TestFixtureRoundTrip(t *testing.T) { roundTrip(t, "fixture", build(fixture)) }

// 改長度：每個 chunk 前後加中文，encode 後重解要得到改後文本，非文本段不變。
func TestEncodeResizes(t *testing.T) {
	ss := decodeSegs(t, build(fixture))
	want := make([]seg, len(ss))
	for i, s := range ss {
		want[i] = s
		if s.Text != nil {
			c := make([]string, len(s.Text))
			for j, x := range s.Text {
				c[j] = "中文" + x + "😀"
			}
			want[i] = txt(c...)
		}
	}
	b := encodeSegs(t, want)
	if !bytes.Equal(b, build(want)) {
		t.Fatal("encode 結果不等於 oracle 重組")
	}
	a, _ := json.Marshal(decodeSegs(t, b))
	w, _ := json.Marshal(want)
	if !bytes.Equal(a, w) {
		t.Fatalf("改長度後重解\n得 %s\n應 %s", a, w)
	}
}

func TestRejectsMalformed(t *testing.T) {
	good := build(fixture)
	for name, b := range map[string][]byte{
		"太短":       good[:6],
		"魔數錯":      append([]byte{0, 0, 0, 0}, good[4:]...),
		"文本截斷":     good[:30],
		"chunk 越界": append(append([]byte{}, good[:26]...), 1, 0xff, 0xff, 'A', 0),
	} {
		var out bytes.Buffer
		if code, _ := run([]string{"decode", writeTmp(t, "bad.nsf", b)}, &out); code != 2 || out.Len() != 0 {
			t.Errorf("decode %s：exit %d、stdout %d bytes，應 exit 2 且 stdout 空", name, code, out.Len())
		}
	}
	long := strings.Repeat("x", 65536)
	empty, badHex, okHex, nonASCII := "", "zz", "00", "中"
	longKw := strings.Repeat("A", 65536) // grok 稽核 m0ft F1：拿掉 kw 長度上限的實作原本仍全綠
	for name, s := range map[string]seg{
		"空段":         {},
		"兩種欄位":       {Raw: &okHex, Kw: &nonASCII},
		"raw 非 hex":  {Raw: &badHex},
		"5 chunk":    txt("a", "b", "c", "d", "e"),
		"空 chunk":    txt("a", ""),
		"chunk 過長":   txt(long),
		"kw 空":       {Kw: &empty},
		"kw 非 ASCII": {Kw: &nonASCII},
		"kw 過長":      {Kw: &longKw},
		"0 chunk":    {Text: []string{}},
	} {
		ss := append(append([]seg{}, fixture[:1]...), s)
		j, _ := json.Marshal(nsf{ss})
		out := filepath.Join(t.TempDir(), "out.nsf")
		if code, _ := run([]string{"encode", writeTmp(t, "bad.json", j), out}, &bytes.Buffer{}); code != 2 {
			t.Errorf("encode %s：exit %d，應 2", name, code)
		}
		if _, err := os.Stat(out); err == nil {
			t.Errorf("encode %s：拒絕時不應寫出檔案", name)
		}
	}
}

// 從 Data\DATA.SLF（A1 格式：檔尾 n×0x118，n 在 +0x200）取名稱符合 keep 的各筆。
func realFiles(t *testing.T, keep func(string) bool) map[string][]byte {
	t.Helper()
	dir := os.Getenv("WIZ8_DIR")
	if dir == "" {
		dir = "/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"
	}
	f, err := os.Open(filepath.Join(dir, "Data", "DATA.SLF"))
	if err != nil {
		t.Skipf("skip：找不到遊戲（%v）", err)
	}
	defer f.Close()
	st, _ := f.Stat()
	var nb [4]byte
	f.ReadAt(nb[:], 0x200)
	n := int64(binary.LittleEndian.Uint32(nb[:]))
	tbl := make([]byte, n*0x118)
	if _, err := f.ReadAt(tbl, st.Size()-n*0x118); err != nil {
		t.Fatal(err)
	}
	got := map[string][]byte{}
	for i := range n {
		r := tbl[i*0x118:]
		name := string(r[:bytes.IndexByte(r[:0x100], 0)])
		if keep(name) {
			b := make([]byte, binary.LittleEndian.Uint32(r[0x104:]))
			if _, err := f.ReadAt(b, int64(binary.LittleEndian.Uint32(r[0x100:]))); err != nil {
				t.Fatal(err)
			}
			got[name] = b
		}
	}
	return got
}

// 原檔裡 6 個 unit 以上的可列印 UTF-16LE 連續段（任意對齊：檔內無對齊要求，R7）。
var run6 = regexp.MustCompile(`(?:[\x20-\x7e\t\n]\x00){6,}`)

// 165 檔：往返 byte 一致；文本段 10320、關鍵字段 4590（cc 獨立掃描）；
// 每個 6 unit 以上的 UTF-16LE 連續段都落在某個文本段內（漏抓＝遊戲裡留英文）；RPC_DRAZIC 0x4cad 是文本段起點。
func TestRealAll(t *testing.T) {
	got := realFiles(t, func(s string) bool { return strings.HasSuffix(s, ".NSF") })
	if len(got) != 165 {
		t.Fatalf("DATA.SLF 有 %d 個 .NSF，應 165", len(got))
	}
	nt, nk, drazic := 0, 0, false
	for name, b := range got {
		ss := roundTrip(t, name, b)
		in := make([]bool, len(b))
		at := 0
		for _, s := range ss {
			n := len(segBytes(s))
			if s.Text != nil {
				nt++
				for i := at; i < at+n; i++ {
					in[i] = true
				}
				if strings.HasSuffix(name, `\RPC_DRAZIC.NSF`) && at == 0x4cad {
					drazic = true
				}
			}
			if s.Kw != nil {
				nk++
			}
			at += n
		}
		for _, m := range run6.FindAllIndex(b, -1) {
			if s, e := m[0], m[1]; !in[s] || !in[e-1] {
				t.Errorf("%s：0x%x 起 %d units 的 UTF-16 文字不在任何文本段內", name, s, (e-s)/2)
			}
		}
	}
	if nt != 10320 || nk != 4590 {
		t.Errorf("文本段 %d／關鍵字段 %d，應 10320／4590", nt, nk)
	}
	if !drazic {
		t.Error("RPC_DRAZIC.NSF 0x4cad 沒有被解成文本段")
	}
}
