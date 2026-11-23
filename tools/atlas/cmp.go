package main

import (
	"fmt"
	"image"
	"io"
	"sort"
	"strings"
)

// Cmp 是 stage4 §4.3 的機檢：在 640×480 原生格點上比「實測字形遮罩」與「期望字形遮罩」。
//   - 實測 M：同一畫面、同一位置，有字的幀（a）與空字串幀（bg）不同的原生像素。
//   - 期望 E：圖集字形畫在 log 記下的 (X, Y)，第 j 字在 X + j·h（CJK 記錄 ox=oy=0、寬＝高）。
//
// 截圖是拉伸後的整個螢幕：原生像素 (x, y) 取樣在其中心 ((x+.5)·sx, (y+.5)·sy)。
// nGlide 雙線性拉伸（×6、×4.5）：取樣點離原生像素中心 ≤ 0.5/sx、0.5/sy 原生像素，鄰點混入權重 ≤ 1−(1−.5/sx)(1−.5/sy)（約 0.185）。
// 沒變的像素差值 ≤ 0.185×鄰點真反差，而那個鄰點量到的差值 ≥ 0.815×真反差 → 沒變的 d ≤ 0.227×鄰點 d。
// 門檻取局部：d > 0 且 4·d > 3×3 鄰域最大 d。全域「最大值一半」在字色貼近背景處會漏（加入隊伍，2026-10-07 run 033741）。
// ponytail: 鄰接墨點的反差比超過約 3.6× 時，低反差的墨會被當成暈而漏判（xor 變大、偏嚴，不會假綠）；要更準就改成逐點反解雙線性權重。
type Cmp struct {
	X, Y, Margin int
	Glyphs       []Bitmap
}

type CmpResult struct {
	Measured, Expected, Xor int
	Mask                    string // 每列一行：'#' 兩者皆有、'+' 只有實測、'-' 只有期望、'.' 皆無
	Ink, Bg                 [3]int // 實測遮罩像素在 a、bg 上的各通道中位數（8 位元）；字色判讀見 notes/zh-patch-log.md:193
}

func (c Cmp) Run(a, bg image.Image) CmpResult {
	h := c.Glyphs[0].H
	w := len(c.Glyphs) * h
	sx := float64(a.Bounds().Dx()) / 640
	sy := float64(a.Bounds().Dy()) / 480
	x0, y0, cw, ch := c.X-c.Margin, c.Y-c.Margin, w+2*c.Margin, h+2*c.Margin
	d := make([]int, cw*ch)
	av, bv := make([][3]int, cw*ch), make([][3]int, cw*ch)
	for i := range d {
		px, py := int((float64(x0+i%cw)+.5)*sx), int((float64(y0+i/cw)+.5)*sy)
		ar, ag, ab, _ := a.At(px, py).RGBA()
		br, bgg, bb, _ := bg.At(px, py).RGBA()
		av[i], bv[i] = [3]int{int(ar >> 8), int(ag >> 8), int(ab >> 8)}, [3]int{int(br >> 8), int(bgg >> 8), int(bb >> 8)}
		d[i] = maxInt(absInt(int(ar)-int(br)), absInt(int(ag)-int(bgg)), absInt(int(ab)-int(bb)))
	}
	var inkA, inkB [3][]int
	var r CmpResult
	var mask strings.Builder
	for y := y0; y < y0+ch; y++ {
		for x := x0; x < x0+cw; x++ {
			v, nb := d[(y-y0)*cw+x-x0], 0
			for j := 0; j < 9; j++ {
				if nx, ny := x-x0+j%3-1, y-y0+j/3-1; nx >= 0 && nx < cw && ny >= 0 && ny < ch {
					nb = maxInt(nb, d[ny*cw+nx])
				}
			}
			m := v > 0 && 4*v > nb
			e := false
			if gx, gy := x-c.X, y-c.Y; gx >= 0 && gx < w && gy >= 0 && gy < h {
				g := c.Glyphs[gx/h]
				e = g.Pix[gy*g.W+gx%h] != 0
			}
			k := byte('.')
			switch {
			case m && e:
				k = '#'
			case m:
				k = '+'
			case e:
				k = '-'
			}
			if m {
				r.Measured++
				for c := 0; c < 3; c++ {
					inkA[c] = append(inkA[c], av[(y-y0)*cw+x-x0][c])
					inkB[c] = append(inkB[c], bv[(y-y0)*cw+x-x0][c])
				}
			}
			if e {
				r.Expected++
			}
			if m != e {
				r.Xor++
			}
			mask.WriteByte(k)
		}
		mask.WriteByte('\n')
	}
	r.Mask = mask.String()
	for c := 0; c < 3; c++ {
		r.Ink[c], r.Bg[c] = median(inkA[c]), median(inkB[c])
	}
	return r
}

func (r CmpResult) Print(w io.Writer) {
	fmt.Fprintf(w, "measured=%d expected=%d xor=%d ink=%d,%d,%d bg=%d,%d,%d\n", r.Measured, r.Expected, r.Xor,
		r.Ink[0], r.Ink[1], r.Ink[2], r.Bg[0], r.Bg[1], r.Bg[2])
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func maxInt(v ...int) int {
	m := v[0]
	for _, x := range v[1:] {
		if x > m {
			m = x
		}
	}
	return m
}

func median(v []int) int {
	if len(v) == 0 {
		return -1
	}
	v = append([]int(nil), v...)
	sort.Ints(v)
	return v[len(v)/2]
}
