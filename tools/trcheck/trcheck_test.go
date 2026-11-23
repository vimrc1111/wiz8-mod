// 驗收測試（cc 寫，實作方不准改）。fixture 全是自造字串，不含原文。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fx：一組輸入。src 0..N-1；預設 733 條（含代名詞 721–732），除了 tr 有的都歸 keep。
type fx struct {
	src     []string
	tr      map[int]string // id → zh（含代號）
	classes map[int][2]string
	gloss   string
	prons   string
	pct     string
	sha     map[int]string // 覆寫 src_sha256
}

func sum(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

func base() *fx {
	f := &fx{tr: map[int]string{}, classes: map[int][2]string{}, sha: map[int]string{}}
	for i := 0; i < 733; i++ {
		f.src = append(f.src, fmt.Sprintf("filler %d", i))
	}
	en := []string{"him", "her", "it", "his", "her", "its", "himself", "herself", "itself"}
	zh := []string{"他", "她", "牠", "他的", "她的", "牠的", "他自己", "她自己", "牠自己"}
	for k := range en {
		f.src[724+k] = en[k]
		f.tr[724+k] = zh[k]
	}
	f.src[1] = "Fighter"
	f.tr[1] = "{{Fighter}}"
	f.src[2] = "%s hits %s for %d damage!"
	f.tr[2] = "%s 擊中 %s，造成 %d 點傷害！"
	f.src[3] = "%s raises %s shield."
	f.tr[3] = "%s 舉起%s盾。"
	f.src[4] = "Over 70% of capacity"
	f.tr[4] = "超過 70%的載重"
	f.src[5] = "%ld gold, %hs"
	f.tr[5] = "%ld 金幣，%hs"
	f.gloss = "# 註解\nFighter\t戰士\t職業\t1\nMook\tMook\t種族\t9\n"
	f.prons = "# 模板\t%s 位置\t組\n3\t2\tposs\n"
	f.pct = "# 原文的 % 是字面\n4\t說明文字\n"
	return f
}

func (f *fx) write(t *testing.T) []string {
	t.Helper()
	d := t.TempDir()
	put := func(n string, b []byte) string {
		p := filepath.Join(d, n)
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	sb, _ := json.Marshal(f.src)
	type ent struct {
		ID  int    `json:"id"`
		Sha string `json:"src_sha256"`
		Zh  string `json:"zh"`
	}
	var es []ent
	var ids []int
	for id := range f.tr {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		s := sum(f.src[id])
		if o, ok := f.sha[id]; ok {
			s = o
		}
		es = append(es, ent{id, s, f.tr[id]})
	}
	tb, _ := json.Marshal(es)
	var c strings.Builder
	c.WriteString("# id\tclass\treason\n")
	for i := range f.src {
		cl, ok := f.classes[i]
		if !ok {
			if _, has := f.tr[i]; has {
				cl = [2]string{"ui", ""}
			} else {
				cl = [2]string{"keep", "填充"}
			}
		}
		if cl[0] == "-" {
			continue // 測試用：整條拿掉
		}
		fmt.Fprintf(&c, "%d\t%s\t%s\n", i, cl[0], cl[1])
	}
	return []string{"check",
		"-src", put("src.json", sb), "-tr", put("tr.json", tb), "-classes", put("classes.tsv", []byte(c.String())),
		"-glossary", put("glossary.tsv", []byte(f.gloss)), "-pronouns", put("pronouns.tsv", []byte(f.prons)),
		"-pct", put("pct.tsv", []byte(f.pct))}
}

func check(t *testing.T, f *fx) (int, string) {
	t.Helper()
	var out bytes.Buffer
	rc := run(f.write(t), &out)
	return rc, out.String()
}

// wantFail：退出碼 1，且有一行以 "FAIL <rule> id=<id>" 開頭。
func wantFail(t *testing.T, f *fx, rule string, id int) {
	t.Helper()
	rc, out := check(t, f)
	want := fmt.Sprintf("FAIL %s id=%d", rule, id)
	if rc != 1 || !strings.Contains("\n"+out, "\n"+want) {
		t.Fatalf("要 rc=1 且有 %q；得 rc=%d：\n%s", want, rc, out)
	}
}

func TestGood(t *testing.T) {
	rc, out := check(t, base())
	if rc != 0 || !strings.HasPrefix(out, "ok trcheck:") {
		t.Fatalf("好輸入要 rc=0、ok 行；得 rc=%d：\n%s", rc, out)
	}
}

func TestClassesMissing(t *testing.T) {
	f := base()
	f.classes[7] = [2]string{"-", ""}
	wantFail(t, f, "classes", 7)
}

func TestClassesBadClass(t *testing.T) {
	f := base()
	f.classes[7] = [2]string{"misc", "x"}
	wantFail(t, f, "classes", 7)
}

func TestKeepNeedsReason(t *testing.T) {
	f := base()
	f.classes[7] = [2]string{"keep", ""}
	wantFail(t, f, "classes", 7)
}

func TestUntranslated(t *testing.T) {
	f := base()
	f.classes[7] = [2]string{"system", ""}
	wantFail(t, f, "coverage", 7)
}

func TestKeepTranslated(t *testing.T) {
	f := base()
	f.classes[2] = [2]string{"keep", "x"}
	wantFail(t, f, "coverage", 2)
}

func TestShaMismatch(t *testing.T) {
	f := base()
	f.sha[2] = strings.Repeat("0", 64)
	wantFail(t, f, "sha", 2)
}

func TestPrintfOrder(t *testing.T) {
	f := base()
	f.src[6], f.tr[6] = "%s has %d", "%d 個 %s"
	wantFail(t, f, "printf", 6)
}

func TestPrintfCount(t *testing.T) {
	f := base()
	f.tr[2] = "%s 擊中 %s！"
	wantFail(t, f, "printf", 2)
}

func TestPrintfType(t *testing.T) {
	f := base()
	f.tr[5] = "%d 金幣，%hs"
	wantFail(t, f, "printf", 5)
}

func TestPrintfPercentEscape(t *testing.T) {
	f := base()
	f.src[6], f.tr[6] = "%d%% bonus", "加成 %d％"
	wantFail(t, f, "printf", 6)
}

func TestPctLiteralNeedsRegistry(t *testing.T) {
	f := base()
	f.pct = ""
	wantFail(t, f, "printf", 4)
}

func TestPctLiteralNoNewSpec(t *testing.T) {
	f := base()
	f.tr[4] = "超過 70%s 的載重"
	wantFail(t, f, "printf", 4)
}

func TestUnknownCode(t *testing.T) {
	f := base()
	f.tr[1] = "{{Fightr}}"
	wantFail(t, f, "code", 1)
}

func TestBrokenCode(t *testing.T) {
	f := base()
	f.tr[1] = "{{Fighter}"
	wantFail(t, f, "code", 1)
}

func TestRawTerm(t *testing.T) {
	f := base()
	f.tr[1] = "戰士"
	wantFail(t, f, "raw-term", 1)
}

// 單字術語（力、火…）會出現在一般詞裡，raw-term 只管 ≥2 字的中文欄
func TestRawTermSingleCharExempt(t *testing.T) {
	f := base()
	f.gloss += "Fire\t火\t元素\t9\n"
	f.tr[1] = "火把"
	rc, out := check(t, f)
	if rc != 0 {
		t.Fatalf("單字術語不該觸發 raw-term；得 rc=%d：\n%s", rc, out)
	}
}

func TestPronounForms(t *testing.T) {
	f := base()
	f.tr[728] = "她们的"
	wantFail(t, f, "pronoun", 728)
}

// -pronoun-forms=false（字串庫）：724–732 缺譯、形式不符都不檢。
func TestPronounFormsOff(t *testing.T) {
	f := base()
	f.tr[728] = "她们的"
	delete(f.tr, 724)
	f.classes[724] = [2]string{"keep", "x"}
	var out bytes.Buffer
	if rc := run(append(f.write(t), "-pronoun-forms=false"), &out); rc != 0 {
		t.Fatalf("關掉代名詞形式檢查應 rc=0；得 rc=%d：\n%s", rc, out.String())
	}
}

func TestPronounTemplateDoubled(t *testing.T) {
	f := base()
	f.tr[3] = "%s 舉起%s的盾。" // poss 代入後「他的的盾」
	wantFail(t, f, "pronoun", 3)
}

func TestPronounTemplateUntranslated(t *testing.T) {
	f := base()
	delete(f.tr, 3)
	f.classes[3] = [2]string{"keep", "x"}
	wantFail(t, f, "pronoun", 3)
}

func TestExpand(t *testing.T) {
	f := base()
	args := f.write(t)
	var out bytes.Buffer
	// expand 只吃 -tr 與 -glossary
	if rc := run([]string{"expand", args[3], args[4], args[7], args[8]}, &out); rc != 0 {
		t.Fatalf("expand rc=%d：%s", rc, out.String())
	}
	var got []struct {
		ID int    `json:"id"`
		Zh string `json:"zh"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("expand 輸出不是 JSON：%v\n%s", err, out.String())
	}
	m := map[int]string{}
	for _, e := range got {
		m[e.ID] = e.Zh
	}
	if len(got) != len(f.tr) || m[1] != "戰士" || m[2] != f.tr[2] {
		t.Fatalf("expand 結果不對：%v", m)
	}
}

// -keywords（NSF 對白，stage7 7.2）：原文以獨立詞出現的關鍵字（不分大小寫），譯文要保留英文，且兩側是空白、ASCII 標點或字串頭尾（不貼中文）。
func kwArgs(t *testing.T, f *fx, kws string) []string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "kw.txt")
	if err := os.WriteFile(p, []byte(kws), 0o644); err != nil {
		t.Fatal(err)
	}
	return append(f.write(t), "-keywords", p)
}

func kwFixture() *fx {
	f := base()
	f.src[10] = "Go to Arnika's tower, ask about the Dark Savant."
	f.tr[10] = "去 Arnika's 塔（ TOWER ），問問 Dark Savant 。"
	f.src[11] = "Arnikan ale is not a keyword hit."
	f.tr[11] = "Arnika 人的麥酒。"
	return f
}

const kwList = "# 一行一個\nARNIKA\nTOWER\nDARK SAVANT\n"

func TestKeywordsGood(t *testing.T) {
	var out bytes.Buffer
	if rc := run(kwArgs(t, kwFixture(), kwList), &out); rc != 0 {
		t.Fatalf("關鍵字保留應 rc=0；得 rc=%d：\n%s", rc, out.String())
	}
}

func TestKeywordsMissing(t *testing.T) {
	for name, zh := range map[string]string{
		"譯掉":   "去 Arnika's 塔，問問 Dark Savant 。",
		"貼中文":  "去Arnika's 塔（ TOWER ），問問 Dark Savant 。",
		"拆開片語": "去 Arnika's 塔（ TOWER ），問問 Dark 的 Savant 。",
	} {
		f := kwFixture()
		f.tr[10] = zh
		var out bytes.Buffer
		rc := run(kwArgs(t, f, kwList), &out)
		if rc != 1 || !strings.Contains(out.String(), "FAIL keyword id=10") {
			t.Errorf("%s：要 rc=1 且 FAIL keyword id=10；得 rc=%d：\n%s", name, rc, out.String())
		}
	}
}
