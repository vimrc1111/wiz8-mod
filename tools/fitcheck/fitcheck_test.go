// 驗收測試（cc 寫，實作方不准改）。fixture 全是自造位元組與字串，不含原文、不含遊戲檔。
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 自造字表 "AB ab." 與對應字形（offX, width, height）。
var chars = []uint16{'A', 'B', ' ', 'a', 'b', '.'}
var glyphs = [][3]int{{0, 7, 11}, {1, 6, 11}, {0, 3, 11}, {0, 5, 9}, {0, 5, 13}, {0, 2, 11}}

// exe：前後塞雜訊；中間是 `66 c7 00 imm16`＋`83 c0 02` 串，中段夾一個 `83 c4 08`，結尾以別的指令斷開。
func exe() []byte {
	var b bytes.Buffer
	b.Write([]byte{0x90, 0x55, 0x66, 0xc7, 0x00}) // 不完整的假開頭
	b.Write([]byte{0x8b, 0xec})
	for i, c := range chars {
		b.Write([]byte{0x66, 0xc7, 0x00, byte(c), byte(c >> 8)})
		if i == 0 {
			b.Write([]byte{0x83, 0xc4, 0x08})
		}
		b.Write([]byte{0x83, 0xc0, 0x02})
	}
	b.Write([]byte{0x5e, 0xc3, 0x66, 0xc7, 0x00, 'Z', 0})
	return b.Bytes()
}

// sti：STCI 64 位元組檔頭（0x18 色數 u32、0x1c 子圖數 u16）＋調色盤＋每子圖 16 位元組（offset u32, size u32, offX i16, offY i16, height u16, width u16）。
func sti(gs [][3]int) []byte {
	h := make([]byte, 64)
	copy(h, "STCI")
	binary.LittleEndian.PutUint32(h[0x10:], 0x28)
	binary.LittleEndian.PutUint32(h[0x18:], 256)
	binary.LittleEndian.PutUint16(h[0x1c:], uint16(len(gs)))
	b := append(h, make([]byte, 768)...)
	for _, g := range gs {
		e := make([]byte, 16)
		binary.LittleEndian.PutUint16(e[8:], uint16(int16(g[0])))
		binary.LittleEndian.PutUint16(e[12:], uint16(g[2]))
		binary.LittleEndian.PutUint16(e[14:], uint16(g[1]))
		b = append(b, e...)
	}
	return b
}

type fx struct {
	src  []string
	tr   map[int]string // id → 展開後中文
	over string         // 登記表內容
	font []byte
	exe  []byte
}

func base() *fx {
	return &fx{
		src:  []string{"AAAA", "ab ab", "", "B."},
		tr:   map[int]string{0: "中中", 1: "中"},
		font: sti(glyphs),
		exe:  exe(),
	}
}

func check(t *testing.T, f *fx) (int, string) {
	t.Helper()
	d := t.TempDir()
	put := func(n string, b []byte) string {
		p := filepath.Join(d, n)
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	src, _ := json.Marshal(f.src)
	type e struct {
		ID int    `json:"id"`
		ZH string `json:"zh"`
	}
	var tr []e
	for id := 0; id < len(f.src)+10; id++ {
		if z, ok := f.tr[id]; ok {
			tr = append(tr, e{id, z})
		}
	}
	trb, _ := json.Marshal(tr)
	var out bytes.Buffer
	rc := run([]string{"-exe", put("x.exe", f.exe), "-font", put("f.sti", f.font), "-src", put("src.json", src),
		"-tr", put("tr.json", trb), "-over", put("over.tsv", []byte(f.over))}, &out)
	return rc, out.String()
}

func want(t *testing.T, f *fx, rc int, lines ...string) string {
	t.Helper()
	got, out := check(t, f)
	if got != rc {
		t.Fatalf("要 rc=%d；得 rc=%d：\n%s", rc, got, out)
	}
	for _, l := range lines {
		if !strings.Contains("\n"+out, "\n"+l) {
			t.Fatalf("缺 %q：\n%s", l, out)
		}
	}
	return out
}

// 寬度：英文 = Σ(offX+width)；中文字前進 = 字型最大字高（13）。
// id0 "AAAA"=28 vs "中中"=26 → 寬度內；id1 "ab ab"=5+5+3+5+5=23 vs "中"=13 → 寬度內。
func TestGood(t *testing.T) {
	want(t, base(), 0, "ok fitcheck: 2 條；寬度內 2、登記溢出 0")
}

func TestOver(t *testing.T) {
	f := base()
	f.tr[3] = "中中" // "B." = 7+2=9 < 26
	want(t, f, 1, "FAIL over id=3: zh 26px > en 9px")
}

func TestOffXCounts(t *testing.T) {
	f := base()
	f.src[3] = "BBBB" // (1+6)*4 = 28
	f.tr[3] = "中中"    // 26
	want(t, f, 0, "ok fitcheck: 3 條")
}

func TestRegistered(t *testing.T) {
	f := base()
	f.tr[3] = "中中"
	f.over = "# 註解\n3\t長文在可換行框內\n"
	want(t, f, 0, "ok fitcheck: 3 條；寬度內 2、登記溢出 1")
}

func TestRegisteredNeedsReason(t *testing.T) {
	f := base()
	f.tr[3] = "中中"
	f.over = "3\t\n"
	want(t, f, 1, "FAIL over-reason id=3")
}

func TestStale(t *testing.T) {
	f := base()
	f.over = "1\t舊登記\n" // id1 已在寬度內
	want(t, f, 1, "FAIL stale id=1")
}

// 字表沒有的字：U+2000 以上算中文前進；以下照遊戲行為退回字形 0（'A'）。
func TestUnknownChars(t *testing.T) {
	f := base()
	f.src[3] = "AAAA" // 28
	f.tr[3] = "x「」"   // 'x' 不在表 → 7；「」各 13 → 33
	want(t, f, 1, "FAIL over id=3: zh 33px > en 28px")
}

// ASCII 在表內的照表量（中文裡夾英文關鍵字、數字）。
func TestMixedASCII(t *testing.T) {
	f := base()
	f.src[3] = "AAAA" // 28
	f.tr[3] = "中 ab"  // 13+3+5+5 = 26
	want(t, f, 0, "ok fitcheck: 3 條")
}

// 原文空字串而譯文非空：寬 0 也照比。
func TestEmptySourceOver(t *testing.T) {
	f := base()
	f.tr[2] = "中"
	want(t, f, 1, "FAIL over id=2: zh 13px > en 0px")
}

func TestBadSTI(t *testing.T) {
	f := base()
	f.font = []byte("NOPE")
	if rc, out := check(t, f); rc != 2 {
		t.Fatalf("壞字型檔要 rc=2；得 %d：\n%s", rc, out)
	}
}

func TestNoTable(t *testing.T) {
	f := base()
	f.exe = []byte{0x90, 0x90, 0xc3}
	if rc, out := check(t, f); rc != 2 {
		t.Fatalf("找不到字表要 rc=2；得 %d：\n%s", rc, out)
	}
}

// 字表比字形多（遊戲字表 252 格、字型 154 個子圖）：超出的格不可當錯，量到時寬 0。
func TestTableLongerThanFont(t *testing.T) {
	f := base()
	f.font = sti(glyphs[:4]) // 'b'、'.' 沒字形
	f.src[3] = "bb"          // 0
	f.tr[3] = "b"
	want(t, f, 0, "ok fitcheck: 3 條")
}

// 明示換行：譯文 LF 數不得多於原文（框高固定）。寬度照全句總和比（LF 本身寬 0）。
func TestExplicitLines(t *testing.T) {
	f := base()
	f.src[3] = "AAAA\nAAAA" // 56，1 個 LF
	f.tr[3] = "中\n中\n中"     // 39，2 個 LF
	want(t, f, 1, "FAIL lines id=3: zh 2 LF > en 1 LF")
}

// 中文字框不小於 12：字型最高 9 時中文前進仍是 12。
func TestCJKMin12(t *testing.T) {
	f := base()
	f.font = sti([][3]int{{0, 7, 9}, {1, 6, 9}, {0, 3, 9}, {0, 5, 9}, {0, 5, 9}, {0, 2, 9}})
	f.src[3] = "B." // 9
	f.tr[3] = "中"   // 12
	want(t, f, 1, "FAIL over id=3: zh 12px > en 9px")
}

// Journal uses a fixed viewport and the hook's Sub[0] height (28 clamped to 24),
// independent of the English length and of the font's tallest subimage.
func TestJournalRun(t *testing.T) {
	for _, tc := range []struct {
		name, zh, over string
		rc             int
		output         string
	}{
		{"over", strings.Repeat("中", 18), "", 1, "FAIL journal id=0 zh=432 max=430\n"},
		{"equal", strings.Repeat("中", 17) + "AAA ", "", 0, "ok journal: 1 條；寬度內 1、登記溢出 0"}, // 408+21+1=430
		{"registered", strings.Repeat("中", 18), "# comment\n0\t待縮短\n", 0, "登記溢出 1"},
		{"reason", strings.Repeat("中", 18), "0\t \n", 1, "FAIL journal id=0 zh=432 max=430\n"},
		{"stale", "中", "0\t舊登記\n", 1, "FAIL journal-stale id=0\n"},
		{"absent", "中", "1\t舊登記\n", 1, "FAIL journal-stale id=1\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			put := func(name string, data []byte) string {
				p := filepath.Join(d, name)
				if err := os.WriteFile(p, data, 0o600); err != nil {
					t.Fatal(err)
				}
				return p
			}
			tr, err := json.Marshal([]map[string]any{{"id": 0, "zh": tc.zh}})
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			rc := run([]string{"-journal", "-exe", put("exe", exe()),
				"-font", put("font", sti([][3]int{{0, 7, 28}, {1, 6, 30}, {0, 1, 10}})),
				"-src", put("src", []byte(`["", ""]`)), "-tr", put("tr", tr),
				"-over", put("over", []byte(tc.over))}, &out)
			if rc != tc.rc || !strings.Contains(out.String(), tc.output) {
				t.Fatalf("rc=%d output=%q; want rc=%d containing %q", rc, out.String(), tc.rc, tc.output)
			}
		})
	}
}

func TestJournalHookMetrics(t *testing.T) {
	f, err := parseFont(sti([][3]int{{1, 6, 9}, {0, 5, 30}}))
	if err != nil {
		t.Fatal(err)
	}
	table := map[rune]int{'A': 0, 'B': 1, 'x': 2}
	for _, tc := range []struct {
		text string
		want int
	}{
		{"中…B", 29},        // First height 9 -> 12, not maximum height 30.
		{"x?", 14},         // ext_font fills missing original glyphs with glyph zero.
		{"\U00020000", 14}, // Atlas contains BMP scalars; each surrogate falls back to glyph zero.
		{"A\x00中", 7},
		{"\n", 7}, // Journal's drawing entry does not introduce a new line.
	} {
		if got := f.journalWidth(tc.text, table); got != tc.want {
			t.Errorf("journalWidth(%q)=%d, want %d", tc.text, got, tc.want)
		}
	}
}
