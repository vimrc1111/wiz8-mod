package slf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	slfHeaderSize = 0x214
	slfRecordSize = 0x118
	manifestName  = ".slf.json"
)

type Entry struct {
	Name   string `json:"name"`
	Offset uint32 `json:"offset"`
	Size   uint32 `json:"size"`
	State  uint32 `json:"state"`
	Time   uint64 `json:"time"`
	Extra  uint32 `json:"extra"`
}

type Manifest struct {
	Name    string    `json:"name"`
	Dir     string    `json:"dir"`
	Fields  [4]uint32 `json:"fields"`
	Entries []Entry   `json:"entries"`
}

func slfString(b []byte) (string, error) {
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return "", fmt.Errorf("檔名沒有 NUL 結尾")
	}
	return string(b[:i]), nil
}

func ReadArchive(f *os.File) (Manifest, error) {
	var m Manifest
	info, err := f.Stat()
	if err != nil {
		return m, err
	}
	if info.Size() < slfHeaderSize {
		return m, fmt.Errorf("檔案小於 0x214 bytes")
	}
	var h [slfHeaderSize]byte
	if _, err := f.ReadAt(h[:], 0); err != nil {
		return m, err
	}
	if m.Name, err = slfString(h[:0x100]); err != nil {
		return m, err
	}
	if m.Dir, err = slfString(h[0x100:0x200]); err != nil {
		return m, err
	}
	n := int64(binary.LittleEndian.Uint32(h[0x200:]))
	dir := info.Size() - n*slfRecordSize
	if dir < slfHeaderSize {
		return m, fmt.Errorf("entry 數 %d 放不下", n)
	}
	for i := range m.Fields {
		m.Fields[i] = binary.LittleEndian.Uint32(h[0x204+i*4:])
	}
	r := io.NewSectionReader(f, dir, n*slfRecordSize)
	for i := int64(0); i < n; i++ {
		var b [slfRecordSize]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return m, err
		}
		name, err := slfString(b[:0x100])
		if err != nil {
			return m, fmt.Errorf("entry %d: %w", i, err)
		}
		e := Entry{name, binary.LittleEndian.Uint32(b[0x100:]), binary.LittleEndian.Uint32(b[0x104:]),
			binary.LittleEndian.Uint32(b[0x108:]), binary.LittleEndian.Uint64(b[0x10c:]), binary.LittleEndian.Uint32(b[0x114:])}
		if int64(e.Offset) < slfHeaderSize || int64(e.Offset)+int64(e.Size) > dir {
			return m, fmt.Errorf("%s: 資料範圍越界", name)
		}
		m.Entries = append(m.Entries, e)
	}
	var end int64 = slfHeaderSize
	for _, i := range dataOrder(m.Entries) {
		e := m.Entries[i]
		// Empty ranges occupy no bytes, even if their offset is inside another range.
		if e.Size == 0 {
			continue
		}
		if int64(e.Offset) < end {
			return m, fmt.Errorf("%s: 資料範圍重疊", e.Name)
		}
		end = int64(e.Offset) + int64(e.Size)
	}
	return m, nil
}

func dataOrder(es []Entry) []int {
	order := make([]int, len(es))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := es[order[i]], es[order[j]]
		if a.Offset == b.Offset {
			return a.Size < b.Size
		}
		return a.Offset < b.Offset
	})
	return order
}

func ListArchive(path string, out io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	m, err := ReadArchive(f)
	if err != nil {
		return err
	}
	for _, e := range m.Entries {
		if _, err := fmt.Fprintf(out, "%d\t%d\t%s\n", e.Offset, e.Size, e.Name); err != nil {
			return err
		}
	}
	return nil
}

// Validate names before touching the filesystem; Root also confines operations
// when an existing directory contains symbolic links or changes concurrently.
func EntryPaths(es []Entry) ([]string, error) {
	paths := make([]string, len(es))
	seen := map[string]bool{manifestName: true}
	for i, e := range es {
		name := strings.ReplaceAll(e.Name, `\`, "/")
		if name == "" || strings.HasPrefix(name, "/") || strings.ContainsRune(name, 0) {
			return nil, fmt.Errorf("%q: 不安全的檔名", e.Name)
		}
		for _, part := range strings.Split(name, "/") {
			if part == ".." || strings.Contains(part, ":") {
				return nil, fmt.Errorf("%q: 不安全的路徑段", e.Name)
			}
		}
		p := filepath.Clean(filepath.FromSlash(name))
		if p == "." || !filepath.IsLocal(p) || seen[p] {
			return nil, fmt.Errorf("%q: 不安全或重複的輸出路徑", e.Name)
		}
		paths[i], seen[p] = p, true
	}
	for _, p := range paths {
		for parent := filepath.Dir(p); parent != "."; parent = filepath.Dir(parent) {
			if seen[parent] {
				return nil, fmt.Errorf("%s: 父目錄與檔案衝突", p)
			}
		}
	}
	return paths, nil
}

func checkExisting(root *os.Root, p string) error {
	for q := p; q != "."; q = filepath.Dir(q) {
		info, err := root.Lstat(q)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if q == p && info.Mode().IsRegular() || q != p && info.IsDir() {
			continue
		}
		return fmt.Errorf("%s: 路徑不是一般檔案或目錄（拒絕 symlink）", q)
	}
	return nil
}

func ExtractArchive(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	m, err := ReadArchive(f)
	if err != nil {
		return err
	}
	paths, err := EntryPaths(m.Entries)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	root, err := os.OpenRoot(dest)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, p := range append(paths, manifestName) {
		if err := checkExisting(root, p); err != nil {
			return fmt.Errorf("%s: %w", dest, err)
		}
	}
	for i, p := range paths {
		if err := root.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		dst, err := root.OpenFile(p, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		e := m.Entries[i]
		_, copyErr := io.Copy(dst, io.NewSectionReader(f, int64(e.Offset), int64(e.Size)))
		closeErr := dst.Close()
		if copyErr != nil {
			return fmt.Errorf("%s: %w", p, copyErr)
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return root.WriteFile(manifestName, append(b, '\n'), 0o644)
}

func validString(s string) bool {
	return len(s) < 0x100 && !strings.ContainsRune(s, 0)
}

func PackArchive(src, dest string) error {
	root, err := os.OpenRoot(src)
	if err != nil {
		return err
	}
	defer root.Close()
	b, err := root.ReadFile(manifestName)
	if err != nil {
		return err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("%s: %w", manifestName, err)
	}
	if !validString(m.Name) || !validString(m.Dir) || uint64(len(m.Entries)) > math.MaxUint32 {
		return fmt.Errorf("%s: 無效的檔頭", manifestName)
	}
	paths, err := EntryPaths(m.Entries)
	if err != nil {
		return err
	}
	order := dataOrder(m.Entries)
	pos := int64(slfHeaderSize)
	for _, i := range order {
		e := &m.Entries[i]
		if !validString(e.Name) {
			return fmt.Errorf("%q: 檔名超過 255 bytes", e.Name)
		}
		if err := checkExisting(root, paths[i]); err != nil {
			return err
		}
		info, err := root.Stat(paths[i])
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > math.MaxUint32 || pos > math.MaxUint32 {
			return fmt.Errorf("%s: 不是一般檔案或超過 u32 範圍", paths[i])
		}
		e.Offset, e.Size = uint32(pos), uint32(info.Size())
		pos += info.Size()
	}
	// Write alongside the destination, then rename only after a complete archive.
	f, err := os.CreateTemp(filepath.Dir(dest), ".slf-pack-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	var h [slfHeaderSize]byte
	copy(h[:0x100], m.Name)
	copy(h[0x100:0x200], m.Dir)
	binary.LittleEndian.PutUint32(h[0x200:], uint32(len(m.Entries)))
	for i, v := range m.Fields {
		binary.LittleEndian.PutUint32(h[0x204+i*4:], v)
	}
	if _, err := f.Write(h[:]); err != nil {
		return err
	}
	for _, i := range order {
		in, err := root.Open(paths[i])
		if err != nil {
			return err
		}
		_, copyErr := io.CopyN(f, in, int64(m.Entries[i].Size))
		closeErr := in.Close()
		if copyErr != nil {
			return fmt.Errorf("%s: %w", paths[i], copyErr)
		}
		if closeErr != nil {
			return closeErr
		}
	}
	for _, e := range m.Entries {
		var b [slfRecordSize]byte
		copy(b[:0x100], e.Name)
		binary.LittleEndian.PutUint32(b[0x100:], e.Offset)
		binary.LittleEndian.PutUint32(b[0x104:], e.Size)
		binary.LittleEndian.PutUint32(b[0x108:], e.State)
		binary.LittleEndian.PutUint64(b[0x10c:], e.Time)
		binary.LittleEndian.PutUint32(b[0x114:], e.Extra)
		if _, err := f.Write(b[:]); err != nil {
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), dest); err != nil {
		return err
	}
	return nil
}
