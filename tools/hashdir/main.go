// hashdir: 遞迴算目錄 SHA256 → JSON，並比對兩份 JSON。
//
//	hashdir scan <dir>            > out.json
//	hashdir diff <old.json> <new.json>   差異印到 stdout；有差異 exit 1
//	hashdir diff -noise <pat,...> <old> <new>   符合 pat 的差異標 noise、不算差異（pat 結尾 / ＝目錄前綴，否則 path.Match）
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type Entry struct {
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Manifest: 相對路徑（正斜線）→ Entry。encoding/json 依 key 排序輸出，結果可重現。
type Manifest map[string]Entry

func scan(root string) (Manifest, error) {
	m := Manifest{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			if d.IsDir() {
				return nil
			}
			return fmt.Errorf("%s: 非一般檔案（symlink／device），拒絕略過", p)
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		h := sha256.New()
		n, err := io.Copy(h, f)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		m[filepath.ToSlash(rel)] = Entry{Size: n, SHA256: hex.EncodeToString(h.Sum(nil))}
		return nil
	})
	return m, err
}

type Diff struct {
	Added, Removed, Changed []string
}

func (d Diff) Empty() bool { return len(d.Added)+len(d.Removed)+len(d.Changed) == 0 }

func diff(old, new Manifest) Diff {
	var d Diff
	for p, e := range new {
		o, ok := old[p]
		switch {
		case !ok:
			d.Added = append(d.Added, p)
		case o != e:
			d.Changed = append(d.Changed, p)
		}
	}
	for p := range old {
		if _, ok := new[p]; !ok {
			d.Removed = append(d.Removed, p)
		}
	}
	sort.Strings(d.Added)
	sort.Strings(d.Removed)
	sort.Strings(d.Changed)
	return d
}

func isNoise(p string, pats []string) bool {
	for _, pat := range pats {
		if strings.HasSuffix(pat, "/") && strings.HasPrefix(p, pat) {
			return true
		}
		if ok, _ := path.Match(pat, p); ok {
			return true
		}
	}
	return false
}

func load(path string) (Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return m, nil
}

func run(args []string, out io.Writer) (int, error) {
	switch {
	case len(args) == 2 && args[0] == "scan":
		m, err := scan(args[1])
		if err != nil {
			return 2, err
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", " ")
		return 0, enc.Encode(m)
	case len(args) == 3 && args[0] == "diff", len(args) == 5 && args[0] == "diff" && args[1] == "-noise":
		var pats []string
		if len(args) == 5 {
			pats = strings.Split(args[2], ",")
			args = []string{"diff", args[3], args[4]}
		}
		a, err := load(args[1])
		if err != nil {
			return 2, err
		}
		b, err := load(args[2])
		if err != nil {
			return 2, err
		}
		d := diff(a, b)
		real := 0
		for _, g := range []struct {
			tag   string
			paths []string
		}{{"A", d.Added}, {"D", d.Removed}, {"M", d.Changed}} {
			for _, p := range g.paths {
				if isNoise(p, pats) {
					fmt.Fprintln(out, "noise", g.tag, p)
					continue
				}
				real++
				fmt.Fprintln(out, g.tag, p)
			}
		}
		// 通過也留痕（無差異時明說），exit code 給腳本判斷
		fmt.Fprintf(out, "summary: %d added, %d removed, %d changed, %d non-noise\n", len(d.Added), len(d.Removed), len(d.Changed), real)
		if real == 0 {
			return 0, nil
		}
		return 1, nil
	}
	return 2, fmt.Errorf("usage: hashdir scan <dir> | hashdir diff [-noise <pat,...>] <old.json> <new.json>")
}

func main() {
	code, err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hashdir:", err)
	}
	os.Exit(code)
}
