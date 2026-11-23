// 驗收測試（cc 寫，實作方不准改）。格式 oracle 在本檔自寫，不採信工具輸出。
// 真實檔案測試在找不到遊戲目錄時 t.Skip；目錄可用 WIZ8_DIR 覆寫。
// 真實檔的計數與雜湊由 cc 以獨立 Python 掃描取得（tickets/K-fact-tool.md「計數來源」）。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

const recSize, f0, f1, fLen = 472, 56, 256, 200

type rec struct {
	Raw  string   `json:"raw"`
	Text []string `json:"text"`
}

type fact struct {
	Records []rec `json:"records"`
}

// oracle：欄位起點 off 的文字＝第一個 u16 NUL 之前的 UTF-16LE。
func field(r []byte, off int) string {
	f := r[off : off+fLen]
	var u []uint16
	for i := 0; i < fLen; i += 2 {
		c := binary.LittleEndian.Uint16(f[i:])
		if c == 0 {
			break
		}
		u = append(u, c)
	}
	return string(utf16.Decode(u))
}

func putField(r []byte, off int, s string) {
	clear(r[off : off+fLen])
	for i, c := range utf16.Encode([]rune(s)) {
		binary.LittleEndian.PutUint16(r[off+2*i:], c)
	}
}

func file(rs ...[]byte) []byte {
	b := binary.LittleEndian.AppendUint32(nil, uint32(len(rs)))
	for _, r := range rs {
		b = append(b, r...)
	}
	return b
}

// 合成記錄：u32 序號、名稱、兩段文字、u32 尾欄；dirty 時名稱與兩欄 NUL 之後留殘渣（原檔有 99／42／6 筆這樣）。
func mkRec(id uint32, name, t0, t1 string, tail uint32, dirty bool) []byte {
	r := make([]byte, recSize)
	binary.LittleEndian.PutUint32(r, id)
	copy(r[4:56], name)
	putField(r, f0, t0)
	putField(r, f1, t1)
	binary.LittleEndian.PutUint32(r[468:], tail)
	if dirty {
		r[55] = 2
		copy(r[f0+fLen-20:], []byte{'h', 0, 'e', 0, ' ', 0, 'M', 0})
		copy(r[f1+fLen-8:], []byte{'x', 0, 'y', 0})
	}
	return r
}

var fixRecs = [][]byte{
	mkRec(0, "FACT_NONE", "", "", 0, false),
	mkRec(1, "FACT_A", "Gave Antone a gem", "", 0x15, true),
	mkRec(2, "FACT_B", "Find the key.", "Got the key!", 0, false),
}

func fixture() []byte { return file(fixRecs...) }

func want(rs [][]byte) fact {
	var f fact
	for _, r := range rs {
		f.Records = append(f.Records, rec{hex.EncodeToString(r), []string{field(r, f0), field(r, f1)}})
	}
	return f
}

func writeTmp(t *testing.T, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func decode(t *testing.T, b []byte) fact {
	t.Helper()
	var out bytes.Buffer
	if code, err := run([]string{"decode", writeTmp(t, "in.dbs", b)}, &out); code != 0 {
		t.Fatalf("decode exit %d：%v", code, err)
	}
	var f fact
	if err := json.Unmarshal(out.Bytes(), &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func encode(t *testing.T, f fact) []byte {
	t.Helper()
	j, _ := json.Marshal(f)
	out := filepath.Join(t.TempDir(), "out.dbs")
	if code, err := run([]string{"encode", writeTmp(t, "in.json", j), out}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("encode exit %d：%v", code, err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestFixtureDecode(t *testing.T) {
	a, _ := json.Marshal(decode(t, fixture()))
	w, _ := json.Marshal(want(fixRecs))
	if !bytes.Equal(a, w) {
		t.Fatalf("decode\n得 %s\n應 %s", a, w)
	}
}

func TestFixtureRoundTrip(t *testing.T) {
	if got := encode(t, decode(t, fixture())); !bytes.Equal(got, fixture()) {
		t.Fatal("往返不一致（未改的欄位要連殘渣原樣保留）")
	}
}

// 改字：改過的欄位＝新字 UTF-16LE＋NUL 補滿 200 bytes（殘渣清掉）；沒改的欄位與其他位元組原樣。
func TestEncodeChanges(t *testing.T) {
	f := decode(t, fixture())
	f.Records[1].Text[0] = "交給 Antone 一顆寶石😀"
	f.Records[2].Text[1] = strings.Repeat("字", 99)
	exp := make([][]byte, len(fixRecs))
	for i, r := range fixRecs {
		exp[i] = append([]byte{}, r...)
	}
	putField(exp[1], f0, f.Records[1].Text[0])
	putField(exp[2], f1, f.Records[2].Text[1])
	if got := encode(t, f); !bytes.Equal(got, file(exp...)) {
		t.Fatal("encode 結果不等於 oracle")
	}
}

func TestRejectsMalformed(t *testing.T) {
	good := fixture()
	noNul := append([]byte{}, good...)
	for i := 4 + recSize + f0; i < 4+recSize+f0+fLen; i++ {
		noNul[i] = 'A'
	}
	lone := append([]byte{}, good...)
	binary.LittleEndian.PutUint16(lone[4+recSize+f0:], 0xd800) // 落單代理對
	for name, b := range map[string][]byte{
		"太短":      good[:3],
		"大小不合":    good[:len(good)-1],
		"多一筆宣告":   append(binary.LittleEndian.AppendUint32(nil, 4), good[4:]...),
		"欄位無 NUL": noNul,
		"落單代理對":   lone,
	} {
		var out bytes.Buffer
		if code, _ := run([]string{"decode", writeTmp(t, "bad.dbs", b)}, &out); code != 2 || out.Len() != 0 {
			t.Errorf("decode %s：exit %d、stdout %d bytes，應 exit 2 且 stdout 空", name, code, out.Len())
		}
	}
	base := want(fixRecs)
	for name, mod := range map[string]func(*fact){
		"raw 非 hex":  func(f *fact) { f.Records[0].Raw = "zz" },
		"raw 長度錯":    func(f *fact) { f.Records[0].Raw = f.Records[0].Raw[2:] },
		"text 一段":    func(f *fact) { f.Records[0].Text = f.Records[0].Text[:1] },
		"text 三段":    func(f *fact) { f.Records[0].Text = append(f.Records[0].Text, "") },
		"超過 99 unit": func(f *fact) { f.Records[0].Text[0] = strings.Repeat("a", 98) + "😀" },
		"含 NUL":      func(f *fact) { f.Records[0].Text[1] = "a\x00b" },
		"沒有記錄":       func(f *fact) { f.Records = nil },
	} {
		f := want(fixRecs)
		f.Records = append([]rec{}, base.Records...)
		f.Records[0] = rec{base.Records[0].Raw, append([]string{}, base.Records[0].Text...)}
		mod(&f)
		j, _ := json.Marshal(f)
		out := filepath.Join(t.TempDir(), "out.dbs")
		if code, _ := run([]string{"encode", writeTmp(t, "bad.json", j), out}, &bytes.Buffer{}); code != 2 {
			t.Errorf("encode %s：exit %d，應 2", name, code)
		}
		if _, err := os.Stat(out); err == nil {
			t.Errorf("encode %s：拒絕時不應寫出檔案", name)
		}
	}
}

// 從 Data\DATA.SLF（A1 格式：檔尾 n×0x118，n 在 +0x200）取 DATABASES\FACT.DBS。
func realFact(t *testing.T) []byte {
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
	for i := range n {
		r := tbl[i*0x118:]
		if string(r[:bytes.IndexByte(r[:0x100], 0)]) == `DATABASES\FACT.DBS` {
			b := make([]byte, binary.LittleEndian.Uint32(r[0x104:]))
			if _, err := f.ReadAt(b, int64(binary.LittleEndian.Uint32(r[0x100:]))); err != nil {
				t.Fatal(err)
			}
			return b
		}
	}
	t.Fatal(`DATA.SLF 沒有 DATABASES\FACT.DBS`)
	return nil
}

// 807 筆；欄 0 非空 264、欄 1 非空 73；最長 57 unit；全部文字依序以 \n 接起的 sha256（cc 獨立掃描）；往返 byte 一致。
func TestRealFact(t *testing.T) {
	b := realFact(t)
	f := decode(t, b)
	if len(f.Records) != 807 {
		t.Fatalf("%d 筆，應 807", len(f.Records))
	}
	var all []string
	n0, n1, mx := 0, 0, 0
	for i, r := range f.Records {
		w := b[4+i*recSize : 4+(i+1)*recSize]
		if r.Raw != hex.EncodeToString(w) {
			t.Fatalf("第 %d 筆 raw 不等於原檔該段", i)
		}
		if r.Text[0] != field(w, f0) || r.Text[1] != field(w, f1) {
			t.Fatalf("第 %d 筆文字不等於 oracle", i)
		}
		if r.Text[0] != "" {
			n0++
		}
		if r.Text[1] != "" {
			n1++
		}
		for _, s := range r.Text {
			mx = max(mx, len(utf16.Encode([]rune(s))))
		}
		all = append(all, r.Text...)
	}
	h := sha256.Sum256([]byte(strings.Join(all, "\n")))
	if n0 != 264 || n1 != 73 || mx != 57 || hex.EncodeToString(h[:]) != "6c826be391cc7d75b15099fde323f9d990253a54650344a18ce6fb116573286d" {
		t.Errorf("欄 0 %d／欄 1 %d／最長 %d／雜湊 %x，應 264／73／57／6c826be3…", n0, n1, mx, h[:4])
	}
	if !bytes.Equal(encode(t, f), b) {
		t.Fatal("往返不一致")
	}
}
