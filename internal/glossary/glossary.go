package glossary

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Translation struct {
	ID  int    `json:"id"`
	Sha string `json:"src_sha256"`
	Zh  string `json:"zh"`
}

func ReadJSON(path string, target any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(b))) == 0 || strings.TrimSpace(string(b))[0] != '[' {
		return fmt.Errorf("%s: 須為 JSON 陣列", path)
	}
	if err := json.Unmarshal(b, target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func ReadTranslations(path string) ([]Translation, error) {
	var rows []struct {
		ID  *int    `json:"id"`
		Sha *string `json:"src_sha256"`
		Zh  *string `json:"zh"`
	}
	if err := ReadJSON(path, &rows); err != nil {
		return nil, err
	}
	result := make([]Translation, 0, len(rows))
	for i, row := range rows {
		if row.ID == nil || row.Sha == nil || row.Zh == nil {
			return nil, fmt.Errorf("%s: 第 %d 筆缺少 id、src_sha256 或 zh", path, i+1)
		}
		result = append(result, Translation{*row.ID, *row.Sha, *row.Zh})
	}
	return result, nil
}

func ReadTSV(path string, columns int) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows [][]string
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 16*1024*1024)
	for line := 1; s.Scan(); line++ {
		text := s.Text()
		if strings.TrimSpace(text) == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Split(text, "\t")
		if len(fields) != columns {
			return nil, fmt.Errorf("%s:%d: 須有 %d 欄", path, line, columns)
		}
		rows = append(rows, fields)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return rows, nil
}

func ExpandCodes(zh string, glossary map[string]string) (string, string, []string) {
	var expanded, raw strings.Builder
	var problems []string
	for len(zh) > 0 {
		open, close := strings.Index(zh, "{{"), strings.Index(zh, "}}")
		if close >= 0 && (open < 0 || close < open) {
			problems = append(problems, "不成對的 }}")
			expanded.WriteString(zh[:close+2])
			raw.WriteString(zh[:close])
			zh = zh[close+2:]
			continue
		}
		if open < 0 {
			expanded.WriteString(zh)
			raw.WriteString(zh)
			break
		}
		expanded.WriteString(zh[:open])
		raw.WriteString(zh[:open])
		zh = zh[open+2:]
		end := strings.Index(zh, "}}")
		if end < 0 {
			problems = append(problems, "不成對的 {{")
			expanded.WriteString("{{" + zh)
			break
		}
		code := zh[:end]
		if strings.Contains(code, "{{") {
			problems = append(problems, "巢狀或不成對的 {{")
		}
		if value, ok := glossary[code]; ok {
			expanded.WriteString(value)
		} else {
			problems = append(problems, fmt.Sprintf("未知術語 %q", code))
			expanded.WriteString("{{" + code + "}}")
		}
		zh = zh[end+2:]
	}
	return expanded.String(), raw.String(), problems
}

// FromRows validates the four-column glossary read by ReadTSV.
func FromRows(rows [][]string) (map[string]string, error) {
	result := make(map[string]string)
	for _, row := range rows {
		if len(row) != 4 {
			return nil, fmt.Errorf("術語須有 4 欄")
		}
		if row[0] == "" || row[1] == "" {
			return nil, fmt.Errorf("術語英文、中文不可空白")
		}
		if _, exists := result[row[0]]; exists {
			return nil, fmt.Errorf("重複術語 %q", row[0])
		}
		result[row[0]] = row[1]
	}
	return result, nil
}
