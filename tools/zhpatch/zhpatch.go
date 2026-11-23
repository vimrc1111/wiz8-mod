package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"wiz8-mod/internal/dbsname"
	"wiz8-mod/internal/fact"
	"wiz8-mod/internal/glossary"
	"wiz8-mod/internal/nsf"
	"wiz8-mod/internal/slf"
	"wiz8-mod/internal/stringdata"
	"wiz8-mod/internal/strlib"
)

// Only selected archive members are read; the game archives are never extracted
// into or modified in their source directory.
func readMember(f *os.File, m slf.Manifest, name string) ([]byte, slf.Entry, error) {
	for _, e := range m.Entries {
		if e.Name == name {
			b, err := io.ReadAll(io.NewSectionReader(f, int64(e.Offset), int64(e.Size)))
			return b, e, err
		}
	}
	return nil, slf.Entry{}, fmt.Errorf("%s: archive member not found", name)
}

// Validate every row against the original before replacing anything. In
// particular, duplicate IDs must not hash an earlier row's translated text.
func applyTranslations(path string, texts []*string, terms map[string]string) error {
	rows, err := glossary.ReadTranslations(path)
	if err != nil {
		return err
	}
	expanded := make([]string, len(rows))
	for i, row := range rows {
		if row.ID < 0 || row.ID >= len(texts) {
			return fmt.Errorf("%s: id=%d outside source range", path, row.ID)
		}
		hash := sha256.Sum256([]byte(*texts[row.ID]))
		if row.Sha != hex.EncodeToString(hash[:]) {
			return fmt.Errorf("%s: id=%d src_sha256 與原文不符", path, row.ID)
		}
		var problems []string
		expanded[i], _, problems = glossary.ExpandCodes(row.Zh, terms)
		if len(problems) != 0 {
			return fmt.Errorf("%s: id=%d: %s", path, row.ID, strings.Join(problems, "; "))
		}
	}
	for i, row := range rows {
		*texts[row.ID] = expanded[i]
	}
	return nil
}

// Retain the existing JSON adapters and their validation as the common codec
// boundary. No external processes or runtime dependencies are required.
func translate(raw []byte, name, trPath, work, dst string, terms map[string]string) error {
	decode, encode := strlib.DecodeFile, strlib.EncodeFile
	var document any
	switch {
	case name == `STRINGS\STRINGDATA.DAT`:
		decode, encode = stringdata.DecodeFile, stringdata.EncodeFile
		document = new([]string)
	case strings.HasSuffix(name, ".NSF"):
		decode, encode = nsf.DecodeFile, nsf.EncodeFile
		document = new(nsf.Script)
	case strings.HasSuffix(name, "FACT.DBS"):
		decode, encode = fact.DecodeFile, fact.EncodeFile
		document = new(fact.Database)
	case strings.HasSuffix(name, "ITEMS.DBS"), strings.HasSuffix(name, "MONSTERS.DBS"), strings.HasSuffix(name, "SPELLTABLES.DBS"):
		kind := "items"
		if strings.HasSuffix(name, "MONSTERS.DBS") {
			kind = "monsters"
		} else if strings.HasSuffix(name, "SPELLTABLES.DBS") {
			kind = "spells"
		}
		decode = func(in string, out io.Writer) error { return dbsname.DecodeFile(kind, in, out) }
		encode = func(in, out string) error { return dbsname.EncodeFile(kind, in, out) }
		document = new(dbsname.Database)
	default:
		document = new(strlib.Library)
	}
	input := filepath.Join(work, "source")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		return err
	}
	var decoded bytes.Buffer
	if err := decode(input, &decoded); err != nil {
		return err
	}
	if err := json.Unmarshal(decoded.Bytes(), document); err != nil {
		return err
	}
	var texts []*string
	switch d := document.(type) {
	case *[]string:
		for i := range *d {
			texts = append(texts, &(*d)[i])
		}
	case *strlib.Library:
		for i := range d.Entries {
			texts = append(texts, &d.Entries[i].Text)
		}
	case *nsf.Script:
		for i := range d.Segments {
			for j := range d.Segments[i].Text {
				texts = append(texts, &d.Segments[i].Text[j])
			}
		}
	case *fact.Database:
		for i := range d.Records {
			for j := range d.Records[i].Text {
				texts = append(texts, &d.Records[i].Text[j])
			}
		}
	case *dbsname.Database:
		for i := range d.Records {
			for j := range d.Records[i].Text {
				texts = append(texts, &d.Records[i].Text[j])
			}
		}
	}
	if err := applyTranslations(trPath, texts, terms); err != nil {
		return err
	}
	b, err := json.Marshal(document)
	if err != nil {
		return err
	}
	input = filepath.Join(work, "translated.json")
	if err := os.WriteFile(input, b, 0o600); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	return encode(input, dst)
}

func build(game, tr, out string) error {
	rows, err := glossary.ReadTSV(filepath.Join(tr, "glossary.tsv"), 4)
	if err != nil {
		return err
	}
	terms, err := glossary.FromRows(rows)
	if err != nil {
		return err
	}
	libs, err := glossary.ReadTSV(filepath.Join(tr, "lib.tsv"), 2)
	if err != nil {
		return err
	}
	patch, err := os.Open(filepath.Join(game, "Patches", "PATCH.001"))
	if err != nil {
		return err
	}
	defer patch.Close()
	m, err := slf.ReadArchive(patch)
	if err != nil {
		return err
	}
	const stringName = `STRINGS\STRINGDATA.DAT`
	raw, base, err := readMember(patch, m, stringName)
	if err != nil {
		return err
	}
	data, err := os.Open(filepath.Join(game, "Data", "DATA.SLF"))
	if err != nil {
		return err
	}
	defer data.Close()
	dm, err := slf.ReadArchive(data)
	if err != nil {
		return err
	}
	m.Entries = []slf.Entry{base}
	for _, row := range libs {
		e := base
		e.Name, e.Offset, e.Size = row[0], 0, 0
		m.Entries = append(m.Entries, e)
	}
	paths, err := slf.EntryPaths(m.Entries)
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "zhpatch-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	packDir := filepath.Join(work, "pack")
	if err := translate(raw, stringName, filepath.Join(tr, "stringdata.json"), work, filepath.Join(packDir, paths[0]), terms); err != nil {
		return fmt.Errorf("%s: %w", stringName, err)
	}
	for i, row := range libs {
		raw, _, err := readMember(data, dm, row[0])
		if err != nil {
			return err
		}
		if err := translate(raw, row[0], filepath.Join(tr, row[1]), work, filepath.Join(packDir, paths[i+1]), terms); err != nil {
			return fmt.Errorf("%s: %w", row[0], err)
		}
	}
	// jq ascii_downcase changes only ASCII letters, preserving all other bytes.
	lower := func(s string) string {
		return strings.Map(func(r rune) rune {
			if r >= 'A' && r <= 'Z' {
				return r + ('a' - 'A')
			}
			return r
		}, s)
	}
	sort.SliceStable(m.Entries, func(i, j int) bool {
		return lower(m.Entries[i].Name) < lower(m.Entries[j].Name)
	})
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(packDir, ".slf.json"), b, 0o600); err != nil {
		return err
	}
	// Keep the final destination untouched until packing and readback succeed.
	staged, err := os.CreateTemp(filepath.Dir(out), ".zhpatch-*")
	if err != nil {
		return err
	}
	stagedPath := staged.Name()
	defer os.Remove(stagedPath)
	if err := staged.Close(); err != nil {
		return err
	}
	if err := slf.PackArchive(packDir, stagedPath); err != nil {
		return err
	}
	if err := verifyArchive(stagedPath, packDir); err != nil {
		return err
	}
	return os.Rename(stagedPath, out)
}

func verifyArchive(path, source string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	m, err := slf.ReadArchive(f)
	if err != nil {
		return err
	}
	paths, err := slf.EntryPaths(m.Entries)
	if err != nil {
		return err
	}
	for i, e := range m.Entries {
		want, err := os.ReadFile(filepath.Join(source, paths[i]))
		if err != nil {
			return err
		}
		got, _, err := readMember(f, m, e.Name)
		if err != nil {
			return err
		}
		if !bytes.Equal(want, got) {
			return fmt.Errorf("%s: packed member differs from encoded data", e.Name)
		}
	}
	return nil
}
