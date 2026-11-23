// 驗收測試（cc 寫，實作方不准改）。格式 oracle 在本檔自寫，不採信工具輸出。
// 真實檔案測試在找不到遊戲目錄時 t.Skip；目錄可用 WIZ8_DIR 覆寫。
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

type entry struct {
	A    uint32 `json:"a"`
	B    uint32 `json:"b"`
	Text string `json:"text"`
	Nul  bool   `json:"nul"`
}

type lib struct {
	Header  string  `json:"header"`
	Tail    string  `json:"tail"`
	Entries []entry `json:"entries"`
}

// 格式 oracle：5B 檔頭（u32 版本＋u8 旗標；旗標 1＝每個 unit 存 0x9696−c）；
// 每筆 u32 a＋u32 b＋u32 unit 數＋UTF-16LE（nul 時含結尾 NUL）；
// 尾段 tail="dbs"：每筆一個 u32 絕對位移（指向該筆的 a）、u32 筆數、u32 0；
// tail="msg"：u32 0xffffffff、位移表、u32 筆數、u32＝最後一筆結束處（即 0xffffffff 的位移）。
func build(hdr []byte, tail string, es []entry) []byte {
	var b bytes.Buffer
	b.Write(hdr)
	var offs []uint32
	for _, e := range es {
		offs = append(offs, uint32(b.Len()))
		u := utf16.Encode([]rune(e.Text))
		if e.Nul {
			u = append(u, 0)
		}
		if hdr[4] == 1 {
			for i := range u {
				u[i] = 0x9696 - u[i]
			}
		}
		binary.Write(&b, binary.LittleEndian, []uint32{e.A, e.B, uint32(len(u))})
		binary.Write(&b, binary.LittleEndian, u)
	}
	end := uint32(b.Len())
	last := uint32(0)
	if tail == "msg" {
		binary.Write(&b, binary.LittleEndian, uint32(0xffffffff))
		last = end
	}
	binary.Write(&b, binary.LittleEndian, offs)
	binary.Write(&b, binary.LittleEndian, []uint32{uint32(len(es)), last})
	return b.Bytes()
}

var hdr = []byte{1, 0, 0, 0, 0}
var hdrMsg = []byte{1, 0, 0, 0, 1}

var fixture = []entry{
	{1, 0xffffff00, "", true},  // 空的寫法一：一個 NUL
	{1, 0xffffff00, "", false}, // 空的寫法二：長度 0
	{1, 0xffffff00, "A sharp blade.", true},
	{0x572d39, 0xffffffff, "", false},
	{7, 8, "中文 😀 測試", true}, // 非 BMP 走 surrogate pair
}

func writeTmp(t *testing.T, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func decodeRaw(t *testing.T, raw []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	if code, err := run([]string{"decode", writeTmp(t, "in.dbs", raw)}, &out); code != 0 || err != nil {
		t.Fatalf("decode: code=%d err=%v", code, err)
	}
	return out.Bytes()
}

func encodeJSON(t *testing.T, j []byte) []byte {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "out.dbs")
	if code, err := run([]string{"encode", writeTmp(t, "in.json", j), dst}, &bytes.Buffer{}); code != 0 || err != nil {
		t.Fatalf("encode: code=%d err=%v", code, err)
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func roundTrip(t *testing.T, raw []byte) {
	t.Helper()
	if got := encodeJSON(t, decodeRaw(t, raw)); !bytes.Equal(got, raw) {
		t.Fatalf("往返不一致：%d → %d bytes", len(raw), len(got))
	}
}

func TestFixtureDecode(t *testing.T) {
	for _, c := range []struct {
		hdr  []byte
		tail string
	}{{hdr, "dbs"}, {hdrMsg, "msg"}} {
		var l lib
		if err := json.Unmarshal(decodeRaw(t, build(c.hdr, c.tail, fixture)), &l); err != nil {
			t.Fatalf("decode 輸出不是 {header, tail, entries}：%v", err)
		}
		if want := hex(c.hdr); l.Header != want || l.Tail != c.tail {
			t.Errorf("header／tail = %q／%q，應 %q／%q（header 為 5 bytes 小寫 hex）", l.Header, l.Tail, want, c.tail)
		}
		if len(l.Entries) != len(fixture) {
			t.Fatalf("%s 筆數 %d，應 %d", c.tail, len(l.Entries), len(fixture))
		}
		for i, w := range fixture {
			if l.Entries[i] != w {
				t.Errorf("%s [%d] = %+v，應 %+v", c.tail, i, l.Entries[i], w)
			}
		}
	}
}

func hex(b []byte) string {
	const d = "0123456789abcdef"
	s := ""
	for _, c := range b {
		s += string(d[c>>4]) + string(d[c&15])
	}
	return s
}

func TestFixtureRoundTrip(t *testing.T) {
	roundTrip(t, build(hdr, "dbs", fixture))
	roundTrip(t, build(hdrMsg, "msg", fixture))
	roundTrip(t, build(hdrMsg, "dbs", fixture)) // 旗標與尾段各自獨立
	roundTrip(t, build(hdr, "msg", nil))
}

// 改長度後位移表（msg 另有結尾位移）要重算：以 oracle 重建的結果為準。
func TestEncodeRecomputesOffsets(t *testing.T) {
	es := append([]entry(nil), fixture...)
	es[2].Text = "一把鋒利的刀刃，比原文長得多也短得多都要對。"
	es[4].Text = "x"
	for _, c := range []struct {
		hdr  []byte
		tail string
	}{{hdr, "dbs"}, {hdrMsg, "msg"}} {
		j, _ := json.Marshal(lib{hex(c.hdr), c.tail, es})
		if got, want := encodeJSON(t, j), build(c.hdr, c.tail, es); !bytes.Equal(got, want) {
			t.Errorf("%s：encode 與 oracle 不一致：%d vs %d bytes", c.tail, len(got), len(want))
		}
	}
}

func TestRejectsMalformed(t *testing.T) {
	good := build(hdr, "dbs", []entry{{1, 2, "ab", true}, {1, 2, "", false}})
	goodMsg := build(hdrMsg, "msg", []entry{{1, 2, "ab", true}, {1, 2, "", false}})
	put := func(src []byte, off int, v uint32) []byte {
		b := bytes.Clone(src)
		if off < 0 {
			off += len(b)
		}
		binary.LittleEndian.PutUint32(b[off:], v)
		return b
	}
	cases := map[string][]byte{
		"short":         good[:7],
		"hdr-flag-2":    func() []byte { b := bytes.Clone(good); b[4] = 2; return b }(), // 旗標只認 0／1
		"dbs-last":      put(good, -4, 1),
		"msg-last":      put(goodMsg, -4, 0), // msg 結尾位移須等於最後一筆結束處
		"msg-sentinel":  put(goodMsg, -20, 0xfffffffe),
		"count-big":     put(good, -8, 3),
		"count-small":   put(good, -8, 1),
		"offset-wrong":  put(good, -16, 6),
		"nunits-oob":    put(good, 5+8, 0x7FFFFFFF),
		"no-nul":        build(hdr, "dbs", []entry{{1, 2, "ab", false}}),
		"no-nul-msg":    build(hdrMsg, "msg", []entry{{1, 2, "ab", false}}),
		"inner-nul":     build(hdr, "dbs", []entry{{1, 2, "a\x00b", true}}),
		"inner-nul-msg": build(hdrMsg, "msg", []entry{{1, 2, "a\x00b", true}}),
		"lone-surr":     rawUnits([]uint16{0xD800, 0}),
		"lone-low-surr": rawUnits([]uint16{0xDC00, 0}),
		"over-2000":     build(hdr, "dbs", []entry{{1, 2, strings.Repeat("a", 2000), true}}), // 2001 units，遊戲讀取器拒收
	}
	// gap：最後一筆與尾段之間夾 2 bytes 垃圾（位移鏈銜接不上）
	g := build(hdr, "dbs", []entry{{1, 2, "ab", true}})
	cases["gap"] = append(append(append([]byte(nil), g[:len(g)-12]...), 0, 0), g[len(g)-12:]...)
	for name, raw := range cases {
		var out bytes.Buffer
		code, err := run([]string{"decode", writeTmp(t, name+".dbs", raw)}, &out)
		if code == 0 || err == nil || out.Len() != 0 {
			t.Errorf("decode %s: 應拒絕且無輸出，得 code=%d err=%v out=%d bytes", name, code, err, out.Len())
		}
	}
	long := strings.Repeat("a", 2000)
	pre := `{"header":"0100000000","tail":"dbs","entries":`
	for name, j := range map[string]string{
		"not-object":     `[1]`,
		"hdr-short":      `{"header":"01000000","tail":"dbs","entries":[]}`,
		"hdr-flag-2":     `{"header":"0100000002","tail":"dbs","entries":[]}`,
		"hdr-not-hex":    `{"header":"zz00000000","tail":"dbs","entries":[]}`,
		"tail-unknown":   `{"header":"0100000000","tail":"x","entries":[]}`,
		"tail-missing":   `{"header":"0100000000","entries":[]}`,
		"text-no-nul":    pre + `[{"a":1,"b":2,"text":"ab","nul":false}]}`,
		"has-nul":        pre + `[{"a":1,"b":2,"text":"a\u0000b","nul":true}]}`,
		"over-2000":      pre + `[{"a":1,"b":2,"text":"` + long + `","nul":true}]}`,
		"a-out-of-range": pre + `[{"a":4294967296,"b":2,"text":"","nul":false}]}`,
	} {
		code, err := run([]string{"encode", writeTmp(t, name+".json", []byte(j)), filepath.Join(t.TempDir(), "o.dbs")}, &bytes.Buffer{})
		if code == 0 || err == nil {
			t.Errorf("encode %s: 應拒絕，得 code=%d", name, code)
		}
	}
	// 邊界：2000 units（1999 字＋NUL）要收
	j, _ := json.Marshal(lib{"0100000000", "dbs", []entry{{1, 2, long[:1999], true}}})
	encodeJSON(t, j)
}

func rawUnits(u []uint16) []byte {
	var b bytes.Buffer
	b.Write(hdr)
	binary.Write(&b, binary.LittleEndian, []uint32{1, 2, uint32(len(u))})
	binary.Write(&b, binary.LittleEndian, u)
	binary.Write(&b, binary.LittleEndian, []uint32{5, 1, 0})
	return b.Bytes()
}

// 從 Data\DATA.SLF（A1 格式：檔尾 n×0x118，n 在 +0x200）取名稱符合 keep 的各筆，只讀需要的區段。
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

func countLib(t *testing.T, raw []byte) (n, nonEmpty int, tail string) {
	t.Helper()
	var l lib
	if err := json.Unmarshal(decodeRaw(t, raw), &l); err != nil {
		t.Fatal(err)
	}
	for _, e := range l.Entries {
		if e.Text != "" {
			nonEmpty++
		}
	}
	return len(l.Entries), nonEmpty, l.Tail
}

// 筆數／非空數：cc 以獨立 Python 解析抄下（notes/research/2026-10-07-R6-dbs-msg-format.md 與 tickets/I-strlib-tool.md）。
func TestRealDesc(t *testing.T) {
	want := map[string][2]int{
		`DATABASES\ITEMDESC.DBS`:    {1000, 326},
		`DATABASES\SPELLDESC.DBS`:   {160, 101},
		`DATABASES\SPELLEFFECT.DBS`: {160, 0},
	}
	got := realFiles(t, func(s string) bool { _, ok := want[s]; return ok })
	if len(got) != len(want) {
		t.Fatalf("DATA.SLF 找到 %d 檔，應 %d", len(got), len(want))
	}
	for name, raw := range got {
		roundTrip(t, raw)
		if n, ne, tail := countLib(t, raw); n != want[name][0] || ne != want[name][1] || tail != "dbs" {
			t.Errorf("%s：筆數 %d／非空 %d／tail %q，應 %d／%d／dbs", name, n, ne, tail, want[name][0], want[name][1])
		}
	}
}

// 51 個 MSG（MESSAGES\ 15、QUOTES\PCS\ 36）：全部往返 byte 一致；總筆數 6669、非空 4308、tail 皆 msg。
func TestRealMsg(t *testing.T) {
	got := realFiles(t, func(s string) bool { return strings.HasSuffix(s, ".MSG") })
	if len(got) != 51 {
		t.Fatalf("DATA.SLF 有 %d 個 .MSG，應 51", len(got))
	}
	n, ne := 0, 0
	for name, raw := range got {
		roundTrip(t, raw)
		a, b, tail := countLib(t, raw)
		if tail != "msg" {
			t.Errorf("%s：tail %q，應 msg", name, tail)
		}
		n, ne = n+a, ne+b
	}
	if n != 6669 || ne != 4308 {
		t.Errorf("總筆數 %d／非空 %d，應 6669／4308", n, ne)
	}
}
