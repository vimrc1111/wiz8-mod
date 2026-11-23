package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writePNG(t *testing.T, img image.Image) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "x.png")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return p
}

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func runJSON(t *testing.T, v any, args ...string) {
	t.Helper()
	var buf bytes.Buffer
	if code, err := run(args, &buf); code != 0 || err != nil {
		t.Fatalf("%v: code=%d err=%v", args, code, err)
	}
	if err := json.Unmarshal(buf.Bytes(), v); err != nil {
		t.Fatal(err)
	}
}

func TestStatsUniformAndNot(t *testing.T) {
	black := writePNG(t, solid(8, 8, color.RGBA{0, 0, 0, 255}))
	var s Stats
	runJSON(t, &s, "stats", black)
	if !s.Uniform || s.W != 8 || s.H != 8 || s.Mean != 0 {
		t.Fatalf("黑圖應 uniform：%+v", s)
	}
	img := solid(8, 8, color.RGBA{0, 0, 0, 255})
	img.Set(3, 3, color.RGBA{255, 255, 255, 255})
	runJSON(t, &s, "stats", writePNG(t, img))
	if s.Uniform || s.Stddev == 0 {
		t.Fatalf("有一個白點不應 uniform：%+v", s)
	}
}

func TestDiffRect(t *testing.T) {
	a := solid(10, 10, color.RGBA{10, 10, 10, 255})
	b := solid(10, 10, color.RGBA{10, 10, 10, 255})
	b.Set(7, 7, color.RGBA{40, 10, 10, 255})
	pa, pb := writePNG(t, a), writePNG(t, b)
	var d DiffResult
	runJSON(t, &d, "diff", pa, pb)
	if d.Changed != 1 || d.Pixels != 100 || d.MeanAbs != 0.1 {
		t.Fatalf("全圖：%+v", d)
	}
	runJSON(t, &d, "diff", pa, pb, "0,0,5,5")
	if d.Changed != 0 || d.Pixels != 25 {
		t.Fatalf("不含變動點的區域：%+v", d)
	}
	if code, _ := run([]string{"diff", pa, pb, "8,8,5,5"}, &bytes.Buffer{}); code != 2 {
		t.Fatalf("越界區域應拒絕，code=%d", code)
	}
	if code, _ := run([]string{"diff", pa, writePNG(t, solid(9, 10, color.RGBA{}))}, &bytes.Buffer{}); code != 2 {
		t.Fatalf("尺寸不同應拒絕，code=%d", code)
	}
}
