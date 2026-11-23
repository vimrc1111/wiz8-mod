package main

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// 字型檔：官方 notofonts/noto-cjk Sans2.004 `Sans/SubsetOTF/TC/NotoSansTC-Regular.otf`（下載與比對見 assets/LICENSES）。
const fontSHA = "5bab0cb3c1cf89dde07c4a95a4054b195afbcfe784d69d75c340780712237537"

func testFont(t *testing.T) string {
	p := os.Getenv("WIZ8_FONT")
	if p == "" {
		home, _ := os.UserHomeDir()
		p = filepath.Join(home, ".local/share/wiz8-mod/fonts/noto-sans-tc-2.004/NotoSansTC-Regular.otf")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("字型不在（設 WIZ8_FONT）：%v", err)
	}
	if s := sha256.Sum256(b); hex.EncodeToString(s[:]) != fontSHA {
		t.Fatalf("%s 不是預期的 Noto Sans TC 2.004", p)
	}
	return p
}

func glyphHash(t *testing.T, chars string, h int, p Params) string {
	f, err := LoadFont(testFont(t))
	if err != nil {
		t.Fatal(err)
	}
	s := sha256.New()
	for _, r := range chars {
		bm, err := Rasterize(f, r, h, p)
		if err != nil {
			t.Fatal(err)
		}
		s.Write(bm.Pix)
	}
	return hex.EncodeToString(s.Sum(nil))
}

// 固定字元、固定參數的 bitmap 雜湊；改光柵參數（Default）會紅。
func TestGlyphHash(t *testing.T) {
	for _, c := range []struct {
		chars string
		h     int
		want  string
	}{
		{"新隊伍", 12, "7f816eacd44744bf5d7fb06ab370186481d05c3f04c80800f91c5bf3fb9efe66"},
		{"召集你的隊伍", 18, "737e833e659cca67940fdd333febe15fac17d7346da20c02080634543d63b044"},
	} {
		if got := glyphHash(t, c.chars, c.h, Default); got != c.want {
			t.Errorf("%s h=%d: %s, want %s", c.chars, c.h, got, c.want)
		}
	}
}

func TestMissingGlyph(t *testing.T) {
	f, err := LoadFont(testFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Rasterize(f, 0xE000, 12, Default); err == nil {
		t.Fatal("私用區字元應該報錯")
	}
}

// 合成截圖：背景以原生像素為單位雜色，字形以最近鄰拉伸到 3840×2160 畫上；字形的 8 鄰格加上 15%×墨點反差
// （模擬 nGlide 雙線性：暈的差值＝權重×鄰點的反差，權重上限 0.185）；
// 位置對 → xor 0（暈不能算成字）；錯一格 → xor > 0。
func TestCmpSynthetic(t *testing.T) {
	g := Bitmap{W: 3, H: 3, Pix: []uint8{1, 0, 1, 0, 1, 0, 1, 1, 1}}
	bg := image.NewRGBA(image.Rect(0, 0, 3840, 2160))
	bgv := func(gx, gy int) uint8 { return uint8((gx*31 + gy*17) % 100) }
	for y := 0; y < 2160; y++ {
		for x := 0; x < 3840; x++ {
			v := bgv(x/6, y*2/9)
			bg.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	a := image.NewRGBA(bg.Rect)
	copy(a.Pix, bg.Pix)
	ink := func(gx, gy int) bool { return gx >= 0 && gx < 3 && gy >= 0 && gy < 3 && g.Pix[gy*3+gx] != 0 }
	for y := 0; y < 2160; y++ {
		for x := 0; x < 3840; x++ {
			gx, gy := x/6-100, y*2/9-50
			if ink(gx, gy) {
				a.Set(x, y, color.RGBA{250, 250, 250, 255})
				continue
			}
			d := 0
			for j := 0; j < 9; j++ {
				if nx, ny := gx+j%3-1, gy+j/3-1; ink(nx, ny) {
					d = max(d, (250-int(bgv(nx+100, ny+50)))*15/100)
				}
			}
			v := int(bgv(x/6, y*2/9)) + d
			a.Set(x, y, color.RGBA{uint8(v), uint8(v), uint8(v), 255})
		}
	}
	if r := (Cmp{X: 100, Y: 50, Margin: 2, Glyphs: []Bitmap{g}}).Run(a, bg); r.Xor != 0 || r.Expected != 6 || r.Ink != [3]int{250, 250, 250} {
		t.Errorf("對位：%+v", r)
	}
	if r := (Cmp{X: 101, Y: 50, Margin: 2, Glyphs: []Bitmap{g}}).Run(a, bg); r.Xor == 0 {
		t.Errorf("錯位一格仍 xor 0：%+v", r)
	}
}

// 點陣字型：Fusion Pixel 12px zh-TW 2026.09.25（OFL，assets/LICENSES）；MG 2026-10-07「中文不小於 12 級字」。
const pixSHA = "1dc1db84d42d8d918ea617f5976b5f519e24efcf110d156fc715407357215ceb"

func testPixFont(t *testing.T) string {
	p := os.Getenv("WIZ8_PXFONT")
	if p == "" {
		home, _ := os.UserHomeDir()
		p = filepath.Join(home, ".local/share/wiz8-mod/fonts/fusion-pixel-12px-2026.09.25/fusion-pixel-12px-proportional-zh_tw.ttf")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("點陣字型不在（設 WIZ8_PXFONT）：%v", err)
	}
	if s := sha256.Sum256(b); hex.EncodeToString(s[:]) != pixSHA {
		t.Fatalf("%s 不是預期的 Fusion Pixel 12px zh-TW 2026.09.25", p)
	}
	return p
}

// 點陣字形雜湊（字高 11＝Wiz_Text_Font、24＝2 倍）＋陰影規則：陰影點不壓墨、左上必有墨；框邊長：字高 7／11 都是 12（不小於 12 級字）、17 照字高。
func TestPixelGlyph(t *testing.T) {
	f, err := LoadFont(testFont(t))
	if err != nil {
		t.Fatal(err)
	}
	px, err := LoadFont(testPixFont(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		h    int
		want string
	}{{11, "6bcf9fb7c557eb7e442f997e949282e77b6f204caa5cb370509bc1ac5bf7d895"}, {24, "40fab0a278d3c9ee8b9a4320b34ed69960f2c4dc949243ea3c68022822f92a98"}} {
		s := sha256.New()
		for _, r := range "立伍建隊" {
			bm, err := Glyph(f, px, r, c.h, Default)
			if err != nil {
				t.Fatal(err)
			}
			ink, sh := 0, 0
			for y := 0; y < bm.H; y++ {
				for x := 0; x < bm.W; x++ {
					switch bm.Pix[y*bm.W+x] {
					case 1:
						ink++
					case 2:
						sh++
						if x == 0 || y == 0 || bm.Pix[(y-1)*bm.W+x-1] != 1 {
							t.Errorf("%c h=%d 陰影 (%d,%d) 左上不是墨", r, c.h, x, y)
						}
					}
				}
			}
			if ink == 0 || sh == 0 {
				t.Errorf("%c h=%d 墨 %d 陰影 %d", r, c.h, ink, sh)
			}
			s.Write(bm.Pix)
		}
		if got := hex.EncodeToString(s.Sum(nil)); got != c.want {
			t.Errorf("h=%d: %s, want %s", c.h, got, c.want)
		}
	}
	for h, want := range map[int]int{7: 12, 11: 12, 17: 17, 24: 24} {
		if bm, err := Glyph(f, px, '隊', h, Default); err != nil || bm.W != want || bm.H != want {
			t.Errorf("h=%d 框 %d×%d（%v），要 %d", h, bm.W, bm.H, err, want)
		}
	}
}
