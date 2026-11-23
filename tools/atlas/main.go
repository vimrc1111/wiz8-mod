// atlas: 從 OFL 字型（Noto Sans TC）產生 CJK 點陣圖集，輸出給 hook/ddraw 編進 dll 的 C 原始碼（stage4 §4.1）。
// 圖集只含我們從字型光柵化的 1-bit 字形，不含任何遊戲資料；遊戲字型的字高、字色在 dll 執行時量（notes/zh-patch-log.md:23）。
//
//	atlas [-px <點陣字型.ttf>] c <字型.otf> <輸出.c> <字表檔…>   字表檔裡所有 U+3000 以上的字元取聯集（其餘忽略），依碼位排序
//	atlas [-px <點陣字型.ttf>] cmp <字型.otf> <有字.png> <空字串.png> <x> <y> <字高> <字串> <xor 上限>
//	    §4.3 機檢（cmp.go）：印 measured／expected／xor 與遮罩；xor 超過上限 exit 1
package main

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"strconv"

	"golang.org/x/image/font/sfnt"
)

func run(args []string, out io.Writer) (int, error) {
	var px *sfnt.Font
	if len(args) > 2 && args[0] == "-px" {
		var err error
		if px, err = LoadFont(args[1]); err != nil {
			return 2, fmt.Errorf("%s: %w", args[1], err)
		}
		args = args[2:]
	}
	if len(args) == 9 && args[0] == "cmp" {
		return runCmp(px, args[1:], out)
	}
	if len(args) < 4 || args[0] != "c" {
		return 2, fmt.Errorf("usage: atlas c <字型.otf> <輸出.c> <字表檔…>")
	}
	face, err := LoadFont(args[1])
	if err != nil {
		return 2, fmt.Errorf("%s: %w", args[1], err)
	}
	chars, err := CharsFromFiles(args[3:])
	if err != nil {
		return 2, err
	}
	a, err := Build(face, px, chars, Default)
	if err != nil {
		return 2, err
	}
	f, err := os.Create(args[2])
	if err != nil {
		return 2, err
	}
	if err := a.WriteC(f); err != nil {
		f.Close()
		return 2, fmt.Errorf("%s: %w", args[2], err)
	}
	asciiFont := px
	if asciiFont == nil {
		asciiFont = face
	}
	if err := WriteASCII(f, asciiFont); err != nil {
		f.Close()
		return 2, err
	}
	if err := f.Close(); err != nil {
		return 2, err
	}
	fmt.Fprintf(out, "atlas: %d 字 × 字高 %d–%d → %s\n", len(a.Chars), a.Sizes[0], a.Sizes[len(a.Sizes)-1], args[2])
	return 0, nil
}

func runCmp(px *sfnt.Font, args []string, out io.Writer) (int, error) {
	f, err := LoadFont(args[0])
	if err != nil {
		return 2, fmt.Errorf("%s: %w", args[0], err)
	}
	var n [3]int
	for i := range n {
		if n[i], err = strconv.Atoi(args[3+i]); err != nil {
			return 2, fmt.Errorf("%q 不是整數", args[3+i])
		}
	}
	max, err := strconv.Atoi(args[7])
	if err != nil {
		return 2, fmt.Errorf("%q 不是整數", args[7])
	}
	var imgs [2]image.Image
	for i := range imgs {
		b, err := os.Open(args[1+i])
		if err != nil {
			return 2, err
		}
		imgs[i], err = png.Decode(b)
		b.Close()
		if err != nil {
			return 2, fmt.Errorf("%s: %w", args[1+i], err)
		}
	}
	if imgs[0].Bounds() != imgs[1].Bounds() {
		return 2, fmt.Errorf("兩張圖尺寸不同：%v %v", imgs[0].Bounds(), imgs[1].Bounds())
	}
	c := Cmp{X: n[0], Y: n[1], Margin: 3}
	for _, r := range args[6] {
		g, err := Glyph(f, px, r, n[2], Default)
		if err != nil {
			return 2, err
		}
		for i, v := range g.Pix {
			if v == 2 {
				g.Pix[i] = 0 // 陰影不算墨
			}
		}
		c.Glyphs = append(c.Glyphs, g)
	}
	if len(c.Glyphs) == 0 {
		return 2, fmt.Errorf("空字串")
	}
	res := c.Run(imgs[0], imgs[1])
	fmt.Fprint(out, res.Mask)
	res.Print(out)
	if res.Xor > max {
		return 1, fmt.Errorf("xor %d > 上限 %d", res.Xor, max)
	}
	return 0, nil
}

func main() {
	code, err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "atlas:", err)
	}
	os.Exit(code)
}
