// shot: 截圖量測（不看圖，只出數字），給 wiz8-run.sh 與覆蓋規則判定用。
//
//	shot stats <png>                     → JSON：sha256、尺寸、亮度平均／標準差、uniform
//	shot diff <a.png> <b.png> [x,y,w,h]  → JSON：區域內變動像素數、平均絕對差
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

type Stats struct {
	SHA256  string  `json:"sha256"`
	W       int     `json:"w"`
	H       int     `json:"h"`
	Mean    float64 `json:"mean"`
	Stddev  float64 `json:"stddev"`
	Uniform bool    `json:"uniform"` // 亮度極差 ≤ 2：全黑／單色，擷取多半失敗
}

type DiffResult struct {
	Rect    [4]int  `json:"rect"`
	Pixels  int     `json:"pixels"`
	Changed int     `json:"changed"` // 任一通道不同的像素數
	MeanAbs float64 `json:"mean_abs"`
}

func load(p string) (image.Image, string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, "", err
	}
	img, err := png.Decode(strings.NewReader(string(b)))
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", p, err)
	}
	h := sha256.Sum256(b)
	return img, hex.EncodeToString(h[:]), nil
}

func rgb(img image.Image, x, y int) (int, int, int) {
	r, g, b, _ := img.At(x, y).RGBA()
	return int(r >> 8), int(g >> 8), int(b >> 8)
}

func stats(img image.Image, sum string) Stats {
	bd := img.Bounds()
	var s, ss float64
	lo, hi := 255.0, 0.0
	n := float64(bd.Dx() * bd.Dy())
	for y := bd.Min.Y; y < bd.Max.Y; y++ {
		for x := bd.Min.X; x < bd.Max.X; x++ {
			r, g, b := rgb(img, x, y)
			l := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
			s += l
			ss += l * l
			lo, hi = math.Min(lo, l), math.Max(hi, l)
		}
	}
	mean := s / n
	return Stats{sum, bd.Dx(), bd.Dy(), mean, math.Sqrt(math.Max(0, ss/n-mean*mean)), hi-lo <= 2}
}

func diff(a, b image.Image, rect [4]int) (DiffResult, error) {
	if a.Bounds() != b.Bounds() {
		return DiffResult{}, fmt.Errorf("尺寸不同：%v vs %v", a.Bounds(), b.Bounds())
	}
	bd := a.Bounds()
	r := image.Rect(rect[0], rect[1], rect[0]+rect[2], rect[1]+rect[3])
	if rect == [4]int{} {
		r = bd
	}
	if !r.In(bd) || r.Empty() {
		return DiffResult{}, fmt.Errorf("區域 %v 不在圖內 %v", r, bd)
	}
	d := DiffResult{Rect: [4]int{r.Min.X, r.Min.Y, r.Dx(), r.Dy()}, Pixels: r.Dx() * r.Dy()}
	var sum float64
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			r1, g1, b1 := rgb(a, x, y)
			r2, g2, b2 := rgb(b, x, y)
			ad := abs(r1-r2) + abs(g1-g2) + abs(b1-b2)
			if ad > 0 {
				d.Changed++
			}
			sum += float64(ad) / 3
		}
	}
	d.MeanAbs = sum / float64(d.Pixels)
	return d, nil
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func parseRect(s string) ([4]int, error) {
	var r [4]int
	f := strings.Split(s, ",")
	if len(f) != 4 {
		return r, fmt.Errorf("區域要 x,y,w,h：%q", s)
	}
	for i, v := range f {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return r, fmt.Errorf("區域要非負整數：%q", s)
		}
		r[i] = n
	}
	return r, nil
}

func run(args []string, out io.Writer) (int, error) {
	enc := json.NewEncoder(out)
	switch {
	case len(args) == 2 && args[0] == "stats":
		img, sum, err := load(args[1])
		if err != nil {
			return 2, err
		}
		return 0, enc.Encode(stats(img, sum))
	case (len(args) == 3 || len(args) == 4) && args[0] == "diff":
		var rect [4]int
		if len(args) == 4 {
			var err error
			if rect, err = parseRect(args[3]); err != nil {
				return 2, err
			}
		}
		a, _, err := load(args[1])
		if err != nil {
			return 2, err
		}
		b, _, err := load(args[2])
		if err != nil {
			return 2, err
		}
		d, err := diff(a, b, rect)
		if err != nil {
			return 2, err
		}
		return 0, enc.Encode(d)
	}
	return 2, fmt.Errorf("usage: shot stats <png> | shot diff <a.png> <b.png> [x,y,w,h]")
}

func main() {
	code, err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "shot:", err)
	}
	os.Exit(code)
}
