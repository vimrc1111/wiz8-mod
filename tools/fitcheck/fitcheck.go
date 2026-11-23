package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
)

type translation struct {
	ID *int    `json:"id"`
	ZH *string `json:"zh"`
}

type fontMetrics struct {
	advances    []int
	height      int
	firstHeight int
}

func parseFont(data []byte) (fontMetrics, error) {
	var f fontMetrics
	if len(data) < 64 || string(data[:4]) != "STCI" {
		return f, fmt.Errorf("invalid STI header")
	}
	start := 64
	if binary.LittleEndian.Uint32(data[0x18:]) == 256 {
		start += 768
	}
	n := int(binary.LittleEndian.Uint16(data[0x1c:]))
	if len(data) < start+n*16 {
		return f, fmt.Errorf("truncated STI palette or subimages")
	}
	f.advances = make([]int, n)
	for i := range f.advances {
		entry := data[start+i*16:]
		f.advances[i] = int(int16(binary.LittleEndian.Uint16(entry[8:]))) + int(binary.LittleEndian.Uint16(entry[14:]))
		if i == 0 {
			f.firstHeight = int(binary.LittleEndian.Uint16(entry[12:]))
		}
		if h := int(binary.LittleEndian.Uint16(entry[12:])); h > f.height {
			f.height = h
		}
	}
	return f, nil
}

// A complete store and increment form one entry. Stack adjustments may
// appear inside an entry or between entries, as in the executable fixture.
func tableEntry(data []byte, pos int) (rune, int, bool) {
	if pos+5 > len(data) || !bytes.Equal(data[pos:pos+3], []byte{0x66, 0xc7, 0x00}) {
		return 0, pos, false
	}
	c := rune(binary.LittleEndian.Uint16(data[pos+3:]))
	next := skipStackAdjustments(data, pos+5)
	if next+3 > len(data) || !bytes.Equal(data[next:next+3], []byte{0x83, 0xc0, 0x02}) {
		return 0, pos, false
	}
	return c, next + 3, true
}

func skipStackAdjustments(data []byte, pos int) int {
	for pos+3 <= len(data) && bytes.Equal(data[pos:pos+3], []byte{0x83, 0xc4, 0x08}) {
		pos += 3
	}
	return pos
}

func parseTable(data []byte) (map[rune]int, error) {
	for start := 0; start < len(data); start++ {
		if _, _, ok := tableEntry(data, start); !ok {
			continue
		}
		table := make(map[rune]int)
		for pos, i := start, 0; ; i++ {
			c, next, ok := tableEntry(data, pos)
			if !ok {
				break
			}
			if _, exists := table[c]; !exists {
				table[c] = i
			}
			pos = skipStackAdjustments(data, next)
		}
		return table, nil
	}
	return nil, fmt.Errorf("character table not found")
}

func (f fontMetrics) width(s string, table map[rune]int) int {
	w := 0
	for _, c := range s {
		if c == '\n' {
			continue
		}
		i, found := table[c]
		if !found && c >= 0x2000 {
			w += max(f.height, cjkMin) // dll 字框不小於 12（tools/atlas Cell）
			continue
		}
		// A missing low code point uses glyph zero; a known code point
		// outside the font's subimage count has zero advance.
		if i < len(f.advances) {
			w += f.advances[i]
		}
	}
	return w
}

func parseOver(data []byte) (map[int]string, error) {
	over := make(map[int]string)
	for line, s := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(s) == "" || strings.HasPrefix(s, "#") {
			continue
		}
		idText, reason, ok := strings.Cut(s, "\t")
		id, err := strconv.Atoi(strings.TrimSpace(idText))
		if !ok || err != nil || id < 0 {
			return nil, fmt.Errorf("invalid overflow row %d", line+1)
		}
		if _, exists := over[id]; exists {
			return nil, fmt.Errorf("duplicate overflow id %d", id)
		}
		over[id] = strings.TrimSpace(reason)
	}
	return over, nil
}

// Journal events: x=193+5, exclusive right edge=193+435 (0x005bdd00).
// See notes/re/journal.md. This is independent of the source sentence width.
const journalMax = 430

func (f fontMetrics) journalWidth(s string, table map[rune]int) int {
	// proxy.c ext_font uses Sub[0].h, clamped to atlas sizes 7..24;
	// atlas Cell then makes the square at least 12px. Journal h=28 -> 24.
	cell := max(cjkMin, min(f.firstHeight, 24))
	w := 0
	for _, unit := range utf16.Encode([]rune(s)) {
		if unit == 0 {
			break // The drawing entry consumes a NUL-terminated UTF-16 string.
		}
		i, found := table[rune(unit)]
		if !found && unit >= 0x2000 && (unit < 0xd800 || unit > 0xdfff) {
			w += cell
			continue
		}
		// ext_font fills the unused original slots with copies of glyph zero.
		if i >= len(f.advances) {
			i = 0
		}
		w += f.advances[i]
	}
	return w
}

func run(args []string, out io.Writer) int {
	flags := flag.NewFlagSet("fitcheck", flag.ContinueOnError)
	flags.SetOutput(out)
	exePath := flags.String("exe", "", "game executable")
	fontPath := flags.String("font", "", "STI font")
	srcPath := flags.String("src", "", "source JSON")
	trPath := flags.String("tr", "", "expanded translation JSON")
	overPath := flags.String("over", "", "overflow TSV")
	journal := flags.Bool("journal", false, "check Journal events against their fixed 430px text area")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	failInput := func(err error) int {
		fmt.Fprintf(out, "error fitcheck: %v\n", err)
		return 2
	}
	if flags.NArg() != 0 || *exePath == "" || *fontPath == "" || *srcPath == "" || *trPath == "" || *overPath == "" {
		return failInput(fmt.Errorf("require -exe, -font, -src, -tr and -over; no positional arguments"))
	}
	inputs := make([][]byte, 5)
	for i, path := range []string{*exePath, *fontPath, *srcPath, *trPath, *overPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			return failInput(err)
		}
		inputs[i] = data
	}
	font, err := parseFont(inputs[1])
	if err != nil {
		return failInput(err)
	}
	if *journal && len(font.advances) == 0 {
		return failInput(fmt.Errorf("journal font must contain glyph zero"))
	}
	table, err := parseTable(inputs[0])
	if err != nil {
		return failInput(err)
	}
	var src []*string
	if err := json.Unmarshal(inputs[2], &src); err != nil {
		return failInput(err)
	}
	if src == nil {
		return failInput(fmt.Errorf("source must be a JSON string array"))
	}
	for id, s := range src {
		if s == nil {
			return failInput(fmt.Errorf("source id %d must be a string", id))
		}
	}
	var tr []translation
	if err := json.Unmarshal(inputs[3], &tr); err != nil {
		return failInput(err)
	}
	if tr == nil {
		return failInput(fmt.Errorf("translations must be a JSON array"))
	}
	seen := make(map[int]bool)
	for _, entry := range tr {
		if entry.ID == nil || entry.ZH == nil || *entry.ID < 0 || *entry.ID >= len(src) {
			return failInput(fmt.Errorf("translation requires an in-range id and a zh string"))
		}
		if seen[*entry.ID] {
			return failInput(fmt.Errorf("duplicate translation id %d", *entry.ID))
		}
		seen[*entry.ID] = true
	}
	over, err := parseOver(inputs[4])
	if err != nil {
		return failInput(err)
	}
	problems, within, registered := 0, 0, 0
	report := func(rule string, id int, description string) {
		fmt.Fprintf(out, "FAIL %s id=%d: %s\n", rule, id, description)
		problems++
	}
	actuallyOver := make(map[int]bool)
	for _, entry := range tr {
		id, zh := *entry.ID, *entry.ZH
		if *journal {
			zw := font.journalWidth(zh, table)
			if zw > journalMax {
				actuallyOver[id] = true
				if over[id] != "" {
					registered++
				} else {
					fmt.Fprintf(out, "FAIL journal id=%d zh=%d max=%d\n", id, zw, journalMax)
					problems++
				}
			} else {
				within++
			}
			continue
		}
		en := *src[id]
		zw, ew := font.width(zh, table), font.width(en, table)
		if zw > ew {
			actuallyOver[id] = true
			if _, exists := over[id]; exists {
				registered++
			} else {
				report("over", id, fmt.Sprintf("zh %dpx > en %dpx", zw, ew))
			}
		} else {
			within++
		}
		if zl, el := strings.Count(zh, "\n"), strings.Count(en, "\n"); zl > el {
			report("lines", id, fmt.Sprintf("zh %d LF > en %d LF", zl, el))
		}
	}
	ids := make([]int, 0, len(over))
	for id := range over {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	prefix := ""
	if *journal {
		prefix = "journal-"
	}
	for _, id := range ids {
		if over[id] == "" {
			report(prefix+"over-reason", id, "理由空白")
		}
		if !actuallyOver[id] {
			if *journal {
				fmt.Fprintf(out, "FAIL journal-stale id=%d\n", id)
				problems++
			} else {
				report("stale", id, "未溢出或不在譯文表")
			}
		}
	}
	if problems > 0 {
		return 1
	}
	name := "fitcheck"
	if *journal {
		name = "journal"
	}
	fmt.Fprintf(out, "ok %s: %d 條；寬度內 %d、登記溢出 %d\n", name, len(tr), within, registered)
	return 0
}

// cjkMin：中文字框下限（MG 2026-10-07「不小於 12 級字」；與 tools/atlas Cell 同值）。
const cjkMin = 12
