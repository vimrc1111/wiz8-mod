// 驗收測試（cc 寫，實作方不得改）。格式依據：notes/research/2026-10-07-A1-slf-patch-format.md。
// 真實檔案測試在找不到遊戲目錄時 t.Skip；目錄可用 WIZ8_DIR／WIZ8_UNINSTALL_DIR 覆寫。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	hdrSize = 0x214
	recSize = 0x118
)

type fixFile struct {
	name string // 反斜線分隔，與檔內相同
	body []byte
}

// buildSLF：測試自己的 SLF 寫入器（與實作無關），版面同 A1：檔頭、資料、EOF 目錄。
func buildSLF(name, dir string, files []fixFile) []byte {
	var b bytes.Buffer
	h := make([]byte, hdrSize)
	copy(h[0:], name)
	copy(h[0x100:], dir)
	binary.LittleEndian.PutUint32(h[0x200:], uint32(len(files)))
	binary.LittleEndian.PutUint32(h[0x204:], uint32(len(files)))
	binary.LittleEndian.PutUint32(h[0x208:], 0x200FFFF)
	binary.LittleEndian.PutUint32(h[0x20C:], 1)
	b.Write(h)
	offs := make([]int, len(files))
	for i, f := range files {
		offs[i] = b.Len()
		b.Write(f.body)
	}
	for i, f := range files {
		r := make([]byte, recSize)
		copy(r, f.name)
		binary.LittleEndian.PutUint32(r[0x100:], uint32(offs[i]))
		binary.LittleEndian.PutUint32(r[0x104:], uint32(len(f.body)))
		binary.LittleEndian.PutUint64(r[0x10C:], 0x01C07F0000000000+uint64(i))
		b.Write(r)
	}
	return b.Bytes()
}

// oracleEntry／oracle：測試自己讀目錄（不呼叫實作），用來核對實作的 list 與 extract。
type oracleEntry struct {
	name      string
	off, size int64
}

func oracle(t *testing.T, raw []byte) []oracleEntry {
	t.Helper()
	if len(raw) < hdrSize {
		t.Fatalf("oracle: 檔案 %d bytes 小於檔頭", len(raw))
	}
	n := int64(binary.LittleEndian.Uint32(raw[0x200:]))
	dir := int64(len(raw)) - n*recSize
	if dir < hdrSize {
		t.Fatalf("oracle: entry 數 %d 放不下", n)
	}
	var es []oracleEntry
	for i := int64(0); i < n; i++ {
		r := raw[dir+i*recSize : dir+(i+1)*recSize]
		nm := string(r[:bytes.IndexByte(append(r[:0x100:0x100], 0), 0)])
		es = append(es, oracleEntry{nm,
			int64(binary.LittleEndian.Uint32(r[0x100:])), int64(binary.LittleEndian.Uint32(r[0x104:]))})
	}
	return es
}

// checkBounds：每個 entry 在 [檔頭末, 目錄起點) 內，彼此不重疊。由測試計算，不採信工具。
func checkBounds(t *testing.T, label string, raw []byte, es []oracleEntry) {
	t.Helper()
	dir := int64(len(raw)) - int64(len(es))*recSize
	s := append([]oracleEntry(nil), es...)
	sort.Slice(s, func(i, j int) bool { return s[i].off < s[j].off })
	for i, e := range s {
		if e.off < hdrSize || e.off+e.size > dir {
			t.Errorf("%s: %s 越界 [%d,%d) 資料區 [%d,%d)", label, e.name, e.off, e.off+e.size, hdrSize, dir)
		}
		if i > 0 && s[i-1].off+s[i-1].size > e.off {
			t.Errorf("%s: %s 與 %s 重疊", label, s[i-1].name, e.name)
		}
	}
}

func runOK(t *testing.T, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	code, err := run(args, &out)
	if code != 0 || err != nil {
		t.Fatalf("slf %v: code=%d err=%v out=%s", args, code, err, out.String())
	}
	return out.String()
}

// listNames：`slf list` 每行最後一欄（tab 分隔）是檔名，第一、二欄是 offset、size（十進位）。
func listNames(t *testing.T, out string) []oracleEntry {
	t.Helper()
	var es []oracleEntry
	for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		f := strings.Split(ln, "\t")
		if len(f) < 3 {
			t.Fatalf("list 行不是 offset<TAB>size<TAB>name：%q", ln)
		}
		off, err1 := strconv.ParseInt(f[0], 10, 64)
		size, err2 := strconv.ParseInt(f[1], 10, 64)
		if err1 != nil || err2 != nil {
			t.Fatalf("list offset／size 不是十進位：%q", ln)
		}
		es = append(es, oracleEntry{f[len(f)-1], off, size})
	}
	return es
}

// checkArchive：list 與 oracle 一致、邊界、extract 後逐 entry sha256 對原檔切片。
func checkArchive(t *testing.T, label, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := oracle(t, raw)
	checkBounds(t, label, raw, want)
	got := listNames(t, runOK(t, "list", path))
	if len(got) != len(want) {
		t.Fatalf("%s: list %d 筆，oracle %d 筆", label, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: 第 %d 筆 list=%+v oracle=%+v", label, i, got[i], want[i])
		}
	}
	out := t.TempDir()
	runOK(t, "extract", path, out)
	for _, e := range want {
		b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(strings.ReplaceAll(e.name, `\`, "/"))))
		if err != nil {
			t.Fatalf("%s: %s 沒解出來：%v", label, e.name, err)
		}
		if sha256.Sum256(b) != sha256.Sum256(raw[e.off:e.off+e.size]) {
			t.Fatalf("%s: %s 解出內容與原檔 [%d,+%d) 不同", label, e.name, e.off, e.size)
		}
	}
}

func writeTmp(t *testing.T, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func fixture() []byte {
	return buildSLF("TEST.SLF", `Data\`, []fixFile{
		{`DATABASES\ITEMS.DBS`, []byte("items")},
		{`STRINGS\STRINGDATA.DAT`, []byte("strings-body")},
		{`LEVEL LOAD\EMPTY.STI`, nil},
		{`TOP.TXT`, []byte("t")},
	})
}

func TestFixtureListExtract(t *testing.T) {
	checkArchive(t, "fixture", writeTmp(t, "TEST.SLF", fixture()))
}

func TestFixturePackRoundTrip(t *testing.T) {
	raw := fixture()
	src := writeTmp(t, "TEST.SLF", raw)
	dir := filepath.Join(t.TempDir(), "x")
	runOK(t, "extract", src, dir)
	dst := filepath.Join(t.TempDir(), "OUT.SLF")
	runOK(t, "pack", dir, dst)
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("pack(extract(fixture)) 與原檔不同：%d vs %d bytes", len(got), len(raw))
	}
}

// 壞檔一律 exit 非 0 且回 error，不 panic、不輸出半套結果。
func TestRejectsMalformed(t *testing.T) {
	good := fixture()
	n := len(good)
	cases := map[string][]byte{
		"short":      good[:0x100],
		"count-huge": func() []byte { b := bytes.Clone(good); binary.LittleEndian.PutUint32(b[0x200:], 0xFFFFFF); return b }(),
		"oob-size": func() []byte {
			b := bytes.Clone(good)
			binary.LittleEndian.PutUint32(b[n-4*recSize+0x104:], 0x7FFFFFFF)
			return b
		}(),
		"into-header": func() []byte {
			b := bytes.Clone(good)
			binary.LittleEndian.PutUint32(b[n-4*recSize+0x100:], 0x10)
			return b
		}(),
		"overlap": func() []byte {
			b := bytes.Clone(good)
			binary.LittleEndian.PutUint32(b[n-3*recSize+0x100:], hdrSize+1)
			return b
		}(),
		// grok 稽核 XrSx：以下兩筆在錯誤實作上原本假綠。
		"name-no-nul": func() []byte {
			b := bytes.Clone(good)
			copy(b[n-4*recSize:n-4*recSize+0x100], bytes.Repeat([]byte("A"), 0x100))
			return b
		}(),
		"size-wraps-u32": func() []byte {
			b := bytes.Clone(good)
			off := binary.LittleEndian.Uint32(b[n-4*recSize+0x100:])
			binary.LittleEndian.PutUint32(b[n-4*recSize+0x104:], -off) // off+size 在 uint32 繞回 0
			return b
		}(),
	}
	for name, raw := range cases {
		p := writeTmp(t, name+".SLF", raw)
		var out bytes.Buffer
		code, err := run([]string{"list", p}, &out)
		if code == 0 || err == nil {
			t.Errorf("%s: 應拒絕，得 code=%d err=%v", name, code, err)
		}
	}
}

// 檔名是不受信任輸入：extract 不得寫出輸出目錄之外。
func TestExtractRejectsTraversal(t *testing.T) {
	for _, bad := range []string{`..\..\evil.txt`, `\abs.txt`, `C:\win.txt`, `A\..\..\x.txt`} {
		raw := buildSLF("T.SLF", "", []fixFile{{bad, []byte("x")}})
		p := writeTmp(t, "T.SLF", raw)
		parent := t.TempDir()
		out := filepath.Join(parent, "out")
		var buf bytes.Buffer
		code, err := run([]string{"extract", p, out}, &buf)
		if code == 0 || err == nil {
			t.Errorf("%q: 應拒絕，得 code=%d", bad, code)
		}
		filepath.WalkDir(parent, func(q string, d os.DirEntry, _ error) error {
			if d != nil && !d.IsDir() {
				t.Errorf("%q: 寫出了 %s", bad, q)
			}
			return nil
		})
	}
}

func gameDirs() (clean, uninstall string) {
	clean = os.Getenv("WIZ8_DIR")
	if clean == "" {
		clean = "/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"
	}
	uninstall = os.Getenv("WIZ8_UNINSTALL_DIR")
	if uninstall == "" {
		uninstall = "/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8_Uninstall"
	}
	return
}

// 原版所有 SLF／PATCH＋FanPatch POL Lang.SLF：list 零錯誤、邊界、extract 逐筆 sha256。
func TestRealArchives(t *testing.T) {
	clean, un := gameDirs()
	if _, err := os.Stat(clean); err != nil {
		t.Skip("找不到遊戲目錄：", clean)
	}
	var paths []string
	filepath.WalkDir(clean, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			b := strings.ToUpper(d.Name())
			if strings.HasSuffix(b, ".SLF") || strings.HasPrefix(b, "PATCH.") && len(b) == 9 {
				paths = append(paths, p)
			}
		}
		return nil
	})
	lang := filepath.Join(un, "Localization/Fan-Patch1.28/POL/Lang.SLF")
	if _, err := os.Stat(lang); err == nil {
		paths = append(paths, lang)
	}
	// 原版至少 5 個 SLF＋PATCH.001（A1）；少了代表掃描壞了，不是通過
	if len(paths) < 6 {
		t.Fatalf("只找到 %d 個封存檔：%v", len(paths), paths)
	}
	// 各封存檔互不相干，平行跑（全部約 1.6 GB，逐個讀解要 50 s 以上）；
	// 但同時整檔讀 /mnt/c 太多會讓 drvfs 回 ENOMEM（CI 偶發紅），同時最多 2 個
	sem := make(chan struct{}, 2)
	for _, p := range paths {
		t.Run(filepath.Base(p), func(t *testing.T) {
			t.Parallel()
			sem <- struct{}{}
			defer func() { <-sem }()
			checkArchive(t, p, p)
		})
	}
}

func TestRealPatchPackRoundTrip(t *testing.T) {
	clean, _ := gameDirs()
	src := filepath.Join(clean, "Patches/PATCH.001")
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Skip("找不到 PATCH.001：", err)
	}
	dir := filepath.Join(t.TempDir(), "x")
	runOK(t, "extract", src, dir)
	dst := filepath.Join(t.TempDir(), "PATCH.001")
	runOK(t, "pack", dir, dst)
	got, _ := os.ReadFile(dst)
	if !bytes.Equal(got, raw) {
		t.Fatalf("pack(extract(PATCH.001)) 與原檔不同：%d vs %d bytes", len(got), len(raw))
	}
}
