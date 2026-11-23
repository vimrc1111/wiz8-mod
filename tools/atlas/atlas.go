package main

import (
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"sort"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Params 是光柵參數；改任何一個都會改變 bitmap（atlas_test 的雜湊要跟著紅）。
type Params struct {
	HMin, HMax int     // 產生的字高範圍（像素）；遊戲字型字高在範圍外時 dll 取最近的一個
	Threshold  uint8   // 覆蓋率 ≥ Threshold/255 算有墨
	BaselineEm float64 // 基線距字框頂端（em 比例）；Noto CJK 的表意字框是 +0.88／−0.12 em
}

var Default = Params{HMin: 7, HMax: 24, Threshold: 128, BaselineEm: 0.88}

// Bitmap 是一個字形：W×H，Pix[y*W+x] 為 0（空）、1（墨）、2（陰影）。
type Bitmap struct {
	W, H int
	Pix  []uint8
}

// Atlas：Glyphs[i][j] 是 Chars[j] 在字高 Sizes[i] 的字形，Cells[i] 見方（全形）。
type Atlas struct {
	Chars  []rune
	Sizes  []int
	Cells  []int
	Glyphs [][]Bitmap
}

func LoadFont(path string) (*sfnt.Font, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return opentype.Parse(b)
}

func CharsFromFiles(paths []string) ([]rune, error) {
	set := map[rune]bool{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		for _, r := range string(b) {
			if r >= 0x2000 && r <= 0xffff { // U+2000 起遊戲字表沒有（…），一律由 dll 畫
				set[r] = true
			}
		}
	}
	var rs []rune
	for r := range set {
		rs = append(rs, r)
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i] < rs[j] })
	return rs, nil
}

// Rasterize 把 r 畫進 h×h 的框，基線在 round(h·BaselineEm)，覆蓋率二值化。
func Rasterize(f *sfnt.Font, r rune, h int, p Params) (Bitmap, error) {
	if idx, err := f.GlyphIndex(nil, r); err != nil || idx == 0 {
		return Bitmap{}, fmt.Errorf("字型沒有 U+%04X", r)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: float64(h), DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return Bitmap{}, err
	}
	defer face.Close()
	img := image.NewAlpha(image.Rect(0, 0, h, h))
	d := font.Drawer{Dst: img, Src: image.Opaque, Face: face,
		Dot: fixed.Point26_6{X: 0, Y: fixed.I(int(math.Round(float64(h) * p.BaselineEm)))}}
	d.DrawString(string(r))
	bm := Bitmap{W: h, H: h, Pix: make([]uint8, h*h)}
	for i, a := range img.Pix {
		if a >= p.Threshold {
			bm.Pix[i] = 1
		}
	}
	return bm, nil
}

// 點陣字型（MG 2026-10-07：中文不小於 12 級字，寧可略出框也要清楚；取代 5.4 E 版 BoutiqueBitmap9x9）：Fusion Pixel 12px zh-TW，
// 表意字墨 11×11（em 第 3–13 列），加右下 1px 陰影佔 12×12；字高 ≥ 24 用 2 倍。字框邊長 Cell(h)，比字高大的部分由 dll 上下置中、超出原行高。
const PixEm, PixInk, PixTop = 12, 11, 3

func pixScale(h int) int {
	if h >= 2*PixEm {
		return 2
	}
	return 1
}

// Cell：有點陣字型時字形框的邊長（≥ 字高）；沒有點陣字型時＝字高。
func Cell(px *sfnt.Font, h int) int {
	if px == nil {
		return h
	}
	return max(h, PixInk*pixScale(h)+1)
}

func RasterizePixel(f *sfnt.Font, r rune, h int) (Bitmap, error) {
	if idx, err := f.GlyphIndex(nil, r); err != nil || idx == 0 {
		return Bitmap{}, fmt.Errorf("點陣字型沒有 U+%04X", r)
	}
	s, c := pixScale(h), Cell(f, h)
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: float64(PixEm * s), DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return Bitmap{}, err
	}
	defer face.Close()
	img := image.NewAlpha(image.Rect(0, 0, c, c))
	o := (c - PixInk*s - 1) / 2 // 墨＋陰影置中
	d := font.Drawer{Dst: img, Src: image.Opaque, Face: face,
		Dot: fixed.Point26_6{X: fixed.I(o), Y: fixed.I(o-PixTop*s) + face.Metrics().Ascent}}
	d.DrawString(string(r))
	bm := Bitmap{W: c, H: c, Pix: make([]uint8, c*c)}
	for i, a := range img.Pix {
		if a >= 128 {
			bm.Pix[i] = 1
		}
	}
	for y := c - 1; y > 0; y-- {
		for x := c - 1; x > 0; x-- {
			if bm.Pix[y*c+x] == 0 && bm.Pix[(y-1)*c+x-1] == 1 {
				bm.Pix[y*c+x] = 2
			}
		}
	}
	return bm, nil
}

// Glyph：Cell(px, h) 見方。有點陣字型就用它；點陣字型缺字時用 Noto 畫在框內置中（無陰影）。
func Glyph(f, px *sfnt.Font, r rune, h int, p Params) (Bitmap, error) {
	if px == nil {
		return Rasterize(f, r, h, p)
	}
	if bm, err := RasterizePixel(px, r, h); err == nil {
		return bm, nil
	}
	c, n := Cell(px, h), PixInk*pixScale(h)
	g, err := Rasterize(f, r, n, p)
	if err != nil {
		return Bitmap{}, err
	}
	bm, o := Bitmap{W: c, H: c, Pix: make([]uint8, c*c)}, (c-n)/2
	for y := 0; y < n; y++ {
		copy(bm.Pix[(y+o)*c+o:], g.Pix[y*n:(y+1)*n])
	}
	return bm, nil
}

func Build(f, px *sfnt.Font, chars []rune, p Params) (*Atlas, error) {
	a := &Atlas{Chars: chars}
	for h := p.HMin; h <= p.HMax; h++ {
		row := make([]Bitmap, len(chars))
		for j, r := range chars {
			bm, err := Glyph(f, px, r, h, p)
			if err != nil {
				return nil, err
			}
			row[j] = bm
		}
		a.Sizes = append(a.Sizes, h)
		a.Cells = append(a.Cells, Cell(px, h))
		a.Glyphs = append(a.Glyphs, row)
	}
	return a, nil
}

// WriteC 輸出 dll 用的表：c＝atlas_cell[h-HMIN]，字形 j 在字高 h 的第 y 列從 atlas_bits[atlas_off[h-HMIN] + (j*c + y)*((c+7)/8)] 開始，
// MSB 在左；atlas_shadow 同版面，記陰影點。
func (a *Atlas) WriteC(w io.Writer) error {
	fmt.Fprintf(w, "// 由 tools/atlas 產生，勿手改。\n#define ATLAS_N %d\n#define ATLAS_HMIN %d\n#define ATLAS_HMAX %d\n",
		len(a.Chars), a.Sizes[0], a.Sizes[len(a.Sizes)-1])
	fmt.Fprint(w, "static const unsigned short atlas_chars[ATLAS_N] = {")
	for j, r := range a.Chars {
		if j%16 == 0 {
			fmt.Fprint(w, "\n")
		}
		fmt.Fprintf(w, "0x%04x,", r)
	}
	var bits, shadow []byte
	var offs []int
	for i := range a.Sizes {
		offs = append(offs, len(bits))
		stride := (a.Cells[i] + 7) / 8
		for _, bm := range a.Glyphs[i] {
			for y := 0; y < bm.H; y++ {
				row, sh := make([]byte, stride), make([]byte, stride)
				for x := 0; x < bm.W; x++ {
					switch bm.Pix[y*bm.W+x] {
					case 1:
						row[x/8] |= 0x80 >> (x % 8)
					case 2:
						sh[x/8] |= 0x80 >> (x % 8)
					}
				}
				bits = append(bits, row...)
				shadow = append(shadow, sh...)
			}
		}
	}
	fmt.Fprint(w, "\n};\nstatic const unsigned char atlas_cell[] = {")
	for _, c := range a.Cells {
		fmt.Fprintf(w, "%d,", c)
	}
	fmt.Fprint(w, "};\nstatic const unsigned int atlas_off[] = {")
	for _, o := range offs {
		fmt.Fprintf(w, "%d,", o)
	}
	fmt.Fprint(w, "};\nstatic const unsigned char atlas_bits[] = {")
	for i, b := range bits {
		if i%32 == 0 {
			fmt.Fprint(w, "\n")
		}
		fmt.Fprintf(w, "%d,", b)
	}
	fmt.Fprint(w, "\n};\nstatic const unsigned char atlas_shadow[] = {")
	for i, b := range shadow {
		if i%32 == 0 {
			fmt.Fprint(w, "\n")
		}
		fmt.Fprintf(w, "%d,", b)
	}
	_, err := fmt.Fprint(w, "\n};\n")
	return err
}

// WriteASCII appends a separate 12px proportional table; CJK tables stay byte-identical.
func WriteASCII(w io.Writer, f *sfnt.Font) error {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: 12, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return err
	}
	defer face.Close()
	var bits []byte
	var advances []int
	for r := rune(0x20); r <= 0x7e; r++ {
		advance, ok := face.GlyphAdvance(r)
		if !ok {
			return fmt.Errorf("ASCII missing U+%04X", r)
		}
		advances = append(advances, max(1, advance.Ceil()))
		bm, err := RasterizePixel(f, r, 12)
		if err != nil {
			return err
		}
		for y := 0; y < 12; y++ {
			row := [2]byte{}
			for x := 0; x < 12; x++ {
				if bm.Pix[y*bm.W+x] == 1 {
					row[x/8] |= 0x80 >> (x % 8)
				}
			}
			bits = append(bits, row[:]...)
		}
	}
	fmt.Fprint(w, "static const unsigned char ascii_adv[95] = {")
	for _, a := range advances {
		fmt.Fprintf(w, "%d,", a)
	}
	fmt.Fprint(w, "};\nstatic const unsigned char ascii_bits[95 * 24] = {")
	for i, b := range bits {
		if i%24 == 0 {
			fmt.Fprint(w, "\n")
		}
		fmt.Fprintf(w, "%d,", b)
	}
	_, err = fmt.Fprint(w, "\n};\n")
	return err
}
