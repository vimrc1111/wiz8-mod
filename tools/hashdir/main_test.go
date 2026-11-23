package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func scanJSON(t *testing.T, root string) string {
	t.Helper()
	var buf bytes.Buffer
	if code, err := run([]string{"scan", root}, &buf); code != 0 || err != nil {
		t.Fatalf("scan: code=%d err=%v", code, err)
	}
	p := filepath.Join(t.TempDir(), "m.json")
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestScanKnownHashAndSlashPaths(t *testing.T) {
	root := t.TempDir()
	write(t, root, "Data/a.slf", "abc")
	m, err := scan(root)
	if err != nil {
		t.Fatal(err)
	}
	want := Manifest{"Data/a.slf": {Size: 3, SHA256: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"}}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("got %v want %v", m, want)
	}
}

func TestDiffRoundTrip(t *testing.T) {
	root := t.TempDir()
	write(t, root, "keep.txt", "same")
	write(t, root, "mod.exe", "v1")
	write(t, root, "gone.dll", "x")
	before := scanJSON(t, root)

	var out bytes.Buffer
	if code, _ := run([]string{"diff", before, before}, &out); code != 0 {
		t.Fatalf("identical manifests: code=%d out=%s", code, out.String())
	}

	write(t, root, "mod.exe", "v2") // 同長度，只有內容變
	os.Remove(filepath.Join(root, "gone.dll"))
	write(t, root, "sub/new.ini", "n")
	after := scanJSON(t, root)

	out.Reset()
	code, err := run([]string{"diff", before, after}, &out)
	if code != 1 || err != nil {
		t.Fatalf("code=%d err=%v", code, err)
	}
	want := "A sub/new.ini\nD gone.dll\nM mod.exe\nsummary: 1 added, 1 removed, 1 changed, 3 non-noise\n"
	if out.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestDiffBadJSONFails(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(p, []byte("not json"), 0o644)
	code, err := run([]string{"diff", p, p}, &bytes.Buffer{})
	if code != 2 || err == nil || !strings.Contains(err.Error(), "bad.json") {
		t.Fatalf("code=%d err=%v", code, err)
	}
}

func TestDiffNoise(t *testing.T) {
	root := t.TempDir()
	write(t, root, "Wiz8.CFG", "a")
	write(t, root, "Data/x.slf", "a")
	before := scanJSON(t, root)
	write(t, root, "Wiz8.CFG", "b")
	write(t, root, "Saves/s1/a.sav", "s")
	write(t, root, "Data/y.CFG", "c") // *.CFG 不跨目錄：子目錄的 CFG 不算雜訊
	after := scanJSON(t, root)
	var out bytes.Buffer
	code, _ := run([]string{"diff", "-noise", "Saves/,*.CFG", before, after}, &out)
	want := "A Data/y.CFG\nnoise A Saves/s1/a.sav\nnoise M Wiz8.CFG\nsummary: 2 added, 0 removed, 1 changed, 1 non-noise\n"
	if code != 1 || out.String() != want {
		t.Fatalf("code=%d got:\n%s", code, out.String())
	}
	os.Remove(filepath.Join(root, "Data/y.CFG"))
	after = scanJSON(t, root)
	if code, _ := run([]string{"diff", "-noise", "Saves/,*.CFG", before, after}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("只剩雜訊應 exit 0，code=%d", code)
	}
}
