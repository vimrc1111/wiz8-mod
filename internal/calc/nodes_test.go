package calc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type emuRow struct {
	In      []int64  `json:"in"`
	RNG     [][2]int `json:"rng"`
	Out     []int64  `json:"out"`
	Sampled bool     `json:"sampled"`
}
type scriptRand struct {
	trace [][2]int
	pos   int
}

func (r *scriptRand) R(n int) int {
	if r.pos >= len(r.trace) {
		panic("extra RNG call")
	}
	call := r.trace[r.pos]
	r.pos++
	if call[0] != n {
		panic(fmt.Sprintf("RNG n: got %d want %d", n, call[0]))
	}
	if call[1] < 0 || call[1] >= max(1, n) {
		panic("invalid RNG value")
	}
	return call[1]
}
func replay(node Node, row emuRow) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("replay: %v", p)
		}
	}()
	if len(row.In) != len(node.In) || len(row.Out) != len(node.Out) {
		return fmt.Errorf("input/output arity")
	}
	r := &scriptRand{trace: row.RNG}
	got := node.F(row.In, r)
	if r.pos != len(r.trace) {
		return fmt.Errorf("unused RNG calls: %d", len(r.trace)-r.pos)
	}
	if !reflect.DeepEqual(got, row.Out) {
		return fmt.Errorf("out=%v want=%v", got, row.Out)
	}
	return nil
}
func emuDirectory() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "../../build/emu")
}
func replayFile(path string, node Node) (cases, diff int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 4096), 16*1024*1024)
	var first error
	for scan.Scan() {
		cases++
		var row emuRow
		e := json.Unmarshal(scan.Bytes(), &row)
		if e == nil {
			e = replay(node, row)
		}
		if e != nil {
			diff++
			if first == nil {
				first = fmt.Errorf("line %d: %w", cases, e)
			}
		}
	}
	if e := scan.Err(); e != nil {
		diff++
		first = e
	}
	if cases == 0 {
		diff++
		first = fmt.Errorf("empty JSONL")
	}
	return cases, diff, first
}
func TestEmu(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(emuDirectory(), "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Skip("SKIP: no build/emu/*.jsonl")
	}
	for _, file := range files {
		addr := strings.TrimSuffix(filepath.Base(file), ".jsonl")
		t.Run(addr, func(t *testing.T) {
			proof := strings.TrimSuffix(file, ".jsonl") + ".d3"
			if err := os.Remove(proof); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			node, ok := Nodes[addr]
			if !ok {
				node = Node{Addr: addr, F: func([]int64, Rand) []int64 { panic("unregistered node") }}
			}
			if node.Emu != nil {
				node.F = node.Emu
			}
			cases, diff, err := replayFile(file, node)
			if err != nil && diff == 0 {
				diff = 1
			}
			if e := os.WriteFile(proof, []byte(fmt.Sprintf("cases=%d diff=%d\n", cases, diff)), 0644); e != nil {
				t.Fatal(e)
			}
			if err != nil {
				t.Errorf("cases=%d diff=%d: %v", cases, diff, err)
			}
		})
	}
}
func TestEmuWrong(t *testing.T) {
	if os.Getenv("EMU_WRONG") != "1" {
		t.Skip("SKIP: set EMU_WRONG=1")
	}
	node := Nodes["00517970"]
	original := node.F
	node.F = func(in []int64, r Rand) []int64 { out := original(in, r); out[0]++; return out }
	cases, diff, err := replayFile(filepath.Join(emuDirectory(), "00517970.jsonl"), node)
	if cases == 0 || diff == 0 {
		t.Fatalf("wrong node escaped: cases=%d diff=%d err=%v", cases, diff, err)
	}
	// All valid rows must differ specifically because the output is incremented.
	if diff != cases || err == nil || !strings.Contains(err.Error(), "out=") {
		t.Fatalf("unexpected failure: cases=%d diff=%d err=%v", cases, diff, err)
	}
}
func TestReplayRNGContract(t *testing.T) {
	node := Nodes["00517970"]
	good := emuRow{In: []int64{2, 1, 6}, RNG: [][2]int{{6, 3}}, Out: []int64{6}}
	if err := replay(node, good); err != nil {
		t.Fatal(err)
	}
	for _, trace := range [][][2]int{nil, {{5, 3}}, {{6, 3}, {6, 0}}, {{6, 6}}} {
		row := good
		row.RNG = trace
		if replay(node, row) == nil {
			t.Errorf("accepted %v", trace)
		}
	}
}
func TestNodeDistWeights(t *testing.T) {
	dist, err := NodeDist(Nodes["00517970"], []int64{0, 2, 6})
	if err != nil {
		t.Fatal(err)
	}
	counts := make([]int, 6)
	for raw := 0; raw <= 32767; raw++ {
		counts[(raw*6/32767)%6]++
	}
	want := map[int64]float64{}
	for a, ca := range counts {
		for b, cb := range counts {
			want[int64(2+a+b)] += float64(ca*cb) / (32768 * 32768)
		}
	}
	sum := 0.0
	for _, out := range dist {
		if math.Abs(out.Probability-want[out.Out[0]]) > 1e-14 {
			t.Errorf("%v", out)
		}
		sum += out.Probability
	}
	if math.Abs(sum-1) > 1e-14 {
		t.Fatalf("mass=%g", sum)
	}
	one, err := NodeDist(Nodes["00517970"], []int64{-3, 2, 0})
	if err != nil || len(one) != 1 || one[0].Out[0] != -1 || one[0].Probability != 1 {
		t.Fatalf("R(0): %v %v", one, err)
	}
}
func TestResistNodeComposition(t *testing.T) {
	// Compare the adapter's complete uniform path distribution with the existing formula.
	for _, v := range []int{0, 1, 101, 9999, 42949672} {
		for _, er := range []int{0, 1, 2, 3, 98, 100, 198, 199, 200} {
			got := make(Dist)
			if er == 0 || v == 9999 {
				row := Nodes["00552250"].F([]int64{int64(v), int64(er), 0, 0, 1}, &scriptRand{})
				got[int(row[0])] = 1
			} else {
				h := (er + 1) / 2
				a := er + 1 - h
				for x := 0; x < a; x++ {
					for y := 0; y < h; y++ {
						r := &scriptRand{trace: [][2]int{{a, x}, {h, y}}}
						row := Nodes["00552250"].F([]int64{int64(v), int64(er), 0, 0, 3}, r)
						if r.pos != 2 {
							t.Fatal("missing capped RNG calls")
						}
						got[int(row[0])] += 1 / float64(a*h)
					}
				}
			}
			want := Resist(Dist{v: 1}, er, 0, 0)
			for out, p := range got {
				if math.Abs(p-want[out]) > 1e-11 {
					t.Fatalf("v=%d ER=%d out=%d got=%g want=%g", v, er, out, p, want[out])
				}
			}
			for out, p := range want {
				if math.Abs(p-got[out]) > 1e-11 {
					t.Fatal("missing output")
				}
			}
		}
	}
}
func TestNodeDistLimits(t *testing.T) {
	_, err := NodeDist(Nodes["00517970"], []int64{0, 256, 6})
	if err == nil {
		t.Fatal("accepted invalid dice")
	}
	node := Node{Addr: "test", F: func(_ []int64, r Rand) []int64 {
		for {
			r.R(1)
		}
	}}
	if _, err := NodeDist(node, nil); err == nil {
		t.Fatal("unbounded RNG enumeration")
	}
}

// 5ba740's F feeds 5a6340 an item pointer fixed at iicDsrItem; it must stay
// the "item" object of 005a6340.json or F silently reads the wrong object.
func TestIicDsrItemMatchesCase(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(emuDirectory(), "../../re/emu/cases/005a6340.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c struct{ Objects map[string]json.Number }
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	off, err := c.Objects["item"].Int64()
	if err != nil || iicArena+off != iicDsrItem {
		t.Fatalf("005a6340.json item offset %v (%v), F uses %#x", c.Objects["item"], err, iicDsrItem-iicArena)
	}
}
