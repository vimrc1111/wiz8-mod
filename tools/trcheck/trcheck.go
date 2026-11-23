package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
	gloss "wiz8-mod/internal/glossary"
)

type translation = gloss.Translation

type term struct{ en, zh string }
type classification struct {
	id            int
	class, reason string
}
type pronounTemplate struct {
	id, position int
	group        string
}

var printfToken = regexp.MustCompile(`%[-+ #0]*[0-9]*(\.[0-9]+)?(h|l)?[diouxXcs%]`)

func readJSON(path string, target any) error { return gloss.ReadJSON(path, target) }

func readTranslations(path string) ([]translation, error) { return gloss.ReadTranslations(path) }

func readTSV(path string, columns int) ([][]string, error) { return gloss.ReadTSV(path, columns) }

func parseID(value string) (int, error) {
	id, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("無效整數 %q", value)
	}
	return id, nil
}

// expandCodes returns both the expanded text and the text outside codes.
// Only input codes are scanned; glossary values are never expanded recursively.
func expandCodes(zh string, glossary map[string]string) (string, string, []string) {
	return gloss.ExpandCodes(zh, glossary)
}

func runTrcheck(args []string, out io.Writer) int {
	inputError := func(err error) int {
		fmt.Fprintln(out, "trcheck:", err)
		return 2
	}
	if len(args) == 0 || (args[0] != "check" && args[0] != "expand") {
		return inputError(fmt.Errorf("用法：trcheck check|expand <旗標>"))
	}
	fs := flag.NewFlagSet("trcheck "+args[0], flag.ContinueOnError)
	fs.SetOutput(out)
	trPath := fs.String("tr", "", "譯文 JSON")
	glossPath := fs.String("glossary", "", "術語 TSV")
	var srcPath, classesPath, pronounsPath, pctPath, keywordsPath *string
	pronounForms := new(bool)
	if args[0] == "check" {
		srcPath = fs.String("src", "", "原文 JSON")
		classesPath = fs.String("classes", "", "分類 TSV")
		pronounsPath = fs.String("pronouns", "", "代名詞 TSV")
		pctPath = fs.String("pct", "", "百分號字面 TSV")
		keywordsPath = fs.String("keywords", "", "關鍵字清單（一行一個；NSF 對白用，可省略）")
		pronounForms = fs.Bool("pronoun-forms", true, "檢查 STRINGDATA 724–732 代名詞形式（字串庫等非 STRINGDATA 來源關掉）")
	}
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *trPath == "" || *glossPath == "" {
		return inputError(fmt.Errorf("須提供 -tr 與 -glossary，且不得有額外參數"))
	}
	trs, err := readTranslations(*trPath)
	if err != nil {
		return inputError(err)
	}
	glossRows, err := readTSV(*glossPath, 4)
	if err != nil {
		return inputError(err)
	}
	glossary, err := gloss.FromRows(glossRows)
	if err != nil {
		return inputError(err)
	}
	var terms []term
	for _, row := range glossRows {
		terms = append(terms, term{row[0], row[1]})
	}

	if args[0] == "expand" {
		type expandedRow struct {
			ID int    `json:"id"`
			Zh string `json:"zh"`
		}
		result := make([]expandedRow, 0, len(trs))
		failed := false
		for _, tr := range trs {
			zh, _, problems := expandCodes(tr.Zh, glossary)
			for _, problem := range problems {
				fmt.Fprintf(out, "FAIL code id=%d: %s\n", tr.ID, problem)
				failed = true
			}
			result = append(result, expandedRow{tr.ID, zh})
		}
		if failed {
			return 1
		}
		if err := json.NewEncoder(out).Encode(result); err != nil {
			return inputError(err)
		}
		return 0
	}
	if *srcPath == "" || *classesPath == "" || *pronounsPath == "" || *pctPath == "" {
		return inputError(fmt.Errorf("check 須提供 -src、-classes、-pronouns、-pct"))
	}
	var srcValues []*string
	if err := readJSON(*srcPath, &srcValues); err != nil {
		return inputError(err)
	}
	src := make([]string, len(srcValues))
	for id, value := range srcValues {
		if value == nil {
			return inputError(fmt.Errorf("原文 id=%d 須為字串", id))
		}
		src[id] = *value
	}
	classRows, err := readTSV(*classesPath, 3)
	if err != nil {
		return inputError(err)
	}
	var classes []classification
	for _, row := range classRows {
		id, err := parseID(row[0])
		if err != nil {
			return inputError(err)
		}
		classes = append(classes, classification{id, row[1], row[2]})
	}
	pronRows, err := readTSV(*pronounsPath, 3)
	if err != nil {
		return inputError(err)
	}
	var templates []pronounTemplate
	for _, row := range pronRows {
		id, err := parseID(row[0])
		if err != nil {
			return inputError(err)
		}
		position, err := parseID(row[1])
		if err != nil {
			return inputError(err)
		}
		if position < 1 || (row[2] != "obj" && row[2] != "poss" && row[2] != "refl") {
			return inputError(fmt.Errorf("無效代名詞位置或組：%v", row))
		}
		templates = append(templates, pronounTemplate{id, position, row[2]})
	}
	pctRows, err := readTSV(*pctPath, 2)
	if err != nil {
		return inputError(err)
	}
	pct := make(map[int]bool)
	for _, row := range pctRows {
		id, err := parseID(row[0])
		if err != nil {
			return inputError(err)
		}
		pct[id] = true
	}
	var keywords []string
	if *keywordsPath != "" {
		rows, err := readTSV(*keywordsPath, 1)
		if err != nil {
			return inputError(err)
		}
		for _, row := range rows {
			keywords = append(keywords, row[0])
		}
	}
	return checkTranslations(out, src, trs, classes, glossary, terms, templates, pct, *pronounForms, keywords)
}

// 關鍵字在原文要是獨立詞（兩側非英數）；在譯文兩側只能是 ASCII 非英數或頭尾，不貼中文（遊戲靠英文字詞辨識關鍵字）。
const asciiNonAlnum = `\x00-\x2f\x3a-\x40\x5b-\x60\x7b-\x7f`

func keywordPatterns(kw string) (src, zh *regexp.Regexp) {
	q := regexp.QuoteMeta(kw)
	return regexp.MustCompile(`(?i)(^|[^A-Za-z0-9])` + q + `($|[^A-Za-z0-9])`),
		regexp.MustCompile(`(?i)(^|[` + asciiNonAlnum + `])` + q + `($|[` + asciiNonAlnum + `])`)
}

func checkTranslations(out io.Writer, src []string, trs []translation, classes []classification, glossary map[string]string, terms []term, templates []pronounTemplate, pct map[int]bool, pronounForms bool, keywords []string) int {
	failed := false
	fail := func(rule string, id int, message string) {
		fmt.Fprintf(out, "FAIL %s id=%d: %s\n", rule, id, message)
		failed = true
	}
	byID := make(map[int]string)
	expanded := make(map[int]string)
	for _, tr := range trs {
		if _, exists := byID[tr.ID]; exists {
			fail("coverage", tr.ID, "重複譯文")
		}
		byID[tr.ID] = tr.Zh
	}
	classByID := make(map[int]string)
	translated, kept := 0, 0
	for _, c := range classes {
		if c.id < 0 || c.id >= len(src) {
			fail("classes", c.id, "id 超出原文範圍")
		}
		if _, exists := classByID[c.id]; exists {
			fail("classes", c.id, "重複分類")
		}
		classByID[c.id] = c.class
		switch c.class {
		case "ui", "combat", "system":
		case "keep":
			if strings.TrimSpace(c.reason) == "" {
				fail("classes", c.id, "keep 缺少理由")
			}
		default:
			fail("classes", c.id, "不合法的 class")
		}
		_, has := byID[c.id]
		if c.class == "keep" && has {
			fail("coverage", c.id, "keep 不得有譯文")
		} else if c.class != "keep" && !has {
			fail("coverage", c.id, "缺少譯文")
		}
	}
	for id := range src {
		class, exists := classByID[id]
		if !exists {
			fail("classes", id, "缺少分類")
		}
		if class == "keep" {
			kept++
		} else if _, has := byID[id]; has {
			translated++
		}
	}
	for _, tr := range trs {
		zh, raw, problems := expandCodes(tr.Zh, glossary)
		expanded[tr.ID] = zh
		for _, problem := range problems {
			fail("code", tr.ID, problem)
		}
		for _, t := range terms {
			if t.en != t.zh && utf8.RuneCountInString(t.zh) >= 2 && strings.Contains(raw, t.zh) {
				fail("raw-term", tr.ID, fmt.Sprintf("術語 %q 須使用代號", t.zh))
			}
		}
		if tr.ID < 0 || tr.ID >= len(src) {
			fail("coverage", tr.ID, "譯文 id 超出原文範圍")
			continue
		}
		hash := sha256.Sum256([]byte(src[tr.ID]))
		if tr.Sha != hex.EncodeToString(hash[:]) {
			fail("sha", tr.ID, "src_sha256 與原文不符")
		}
		want := printfToken.FindAllString(src[tr.ID], -1)
		got := printfToken.FindAllString(zh, -1)
		if pct[tr.ID] {
			want = nil
		}
		if !slices.Equal(want, got) {
			fail("printf", tr.ID, fmt.Sprintf("格式記號不符：原文 %q，譯文 %q", want, got))
		}
		for _, kw := range keywords {
			if s, z := keywordPatterns(kw); s.MatchString(src[tr.ID]) && !z.MatchString(zh) {
				fail("keyword", tr.ID, fmt.Sprintf("關鍵字 %q 須保留英文、兩側不貼中文", kw))
			}
		}
	}
	for id := 724; pronounForms && id <= 732; id++ {
		if _, has := byID[id]; !has {
			fail("pronoun", id, "缺少代名詞譯文")
		}
	}
	for k := 0; pronounForms && k < 3; k++ {
		obj, hasObj := byID[724+k]
		for _, form := range []struct {
			id     int
			suffix string
		}{{727 + k, "的"}, {730 + k, "自己"}} {
			if value, has := byID[form.id]; has && hasObj && value != obj+form.suffix {
				fail("pronoun", form.id, "代名詞形式不符")
			}
		}
	}
	for _, template := range templates {
		zh, has := expanded[template.id]
		if !has {
			fail("pronoun", template.id, "模板缺少譯文")
			continue
		}
		// Count actual %s tokens, excluding escaped %%s and other specifiers.
		var positions []int
		for _, span := range printfToken.FindAllStringIndex(zh, -1) {
			if zh[span[0]:span[1]] == "%s" {
				positions = append(positions, span[0])
			}
		}
		if len(positions) < template.position {
			fail("pronoun", template.id, "模板的 %s 數量不足")
			continue
		}
		start := map[string]int{"obj": 724, "poss": 727, "refl": 730}[template.group]
		at := positions[template.position-1]
		for k := 0; k < 3; k++ {
			form, has := byID[start+k]
			if !has {
				continue // Missing forms were reported above.
			}
			result := zh[:at] + form + zh[at+2:]
			if strings.Contains(result, "的的") || strings.Contains(result, "自己自己") {
				fail("pronoun", template.id, "代入後含的的或自己自己")
				break
			}
		}
	}
	if failed {
		return 1
	}
	fmt.Fprintf(out, "ok trcheck: %d 條；譯 %d、不翻 %d\n", len(src), translated, kept)
	return 0
}
