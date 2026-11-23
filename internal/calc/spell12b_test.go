package calc

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// G4（後續票 4b）：驗 4fa5cc 豁免前提——記憶體法術記錄 +0x12b 的模式值只有 0..3。
// 對應關係見 notes/re/misc.md:77：記憶體 +0x12b ＝ DBS 記錄 +0x22c 的 u32。
// 讀檔的檔頭／stride（8／704）沿用 LoadSpells 的方式，不改 load.go。
func TestSpellMode12bDomain(t *testing.T) {
	dir := os.Getenv("WIZ8_DBS")
	if dir == "" {
		t.Skip("WIZ8_DBS 未設：原版資料驗證未跑")
	}
	if !filepath.IsAbs(dir) {
		// go test 的工作目錄是套件目錄，相對路徑以 repo 根（go.mod 所在）為準。
		root, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		for {
			if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
				break
			}
			parent := filepath.Dir(root)
			if parent == root {
				break
			}
			root = parent
		}
		dir = filepath.Join(root, dir)
	}
	b, err := os.ReadFile(filepath.Join(dir, "SPELLTABLES.DBS"))
	if err != nil {
		t.Fatal(err)
	}
	const header, stride, modeOff = 8, 704, 0x22c
	if len(b) < header {
		t.Fatalf("SPELLTABLES.DBS 長 %d，小於檔頭 %d", len(b), header)
	}
	count := int(binary.LittleEndian.Uint32(b))
	if uint64(len(b)) != uint64(header)+uint64(count)*uint64(stride) {
		t.Fatalf("SPELLTABLES.DBS 長 %d，與 %d 筆 x stride %d 不合", len(b), count, stride)
	}
	hist := make(map[uint32]int)
	for i := 0; i < count; i++ {
		r := b[header+i*stride : header+(i+1)*stride]
		v := binary.LittleEndian.Uint32(r[modeOff:])
		hist[v]++
		if v > 3 {
			t.Errorf("法術 %d：模式值 %d 不在 0..3", i, v)
		}
	}
	t.Logf("法術模式直方圖（共 %d 筆）：0:%d 1:%d 2:%d 3:%d 相異值 %d",
		count, hist[0], hist[1], hist[2], hist[3], len(hist))
}
