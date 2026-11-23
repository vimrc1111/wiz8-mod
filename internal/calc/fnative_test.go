package calc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fNativeRow struct {
	emuRow
	Compare []bool `json:"compare"`
}

func replayFNative(node Node, row fNativeRow) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("F replay: %v", p)
		}
	}()
	if len(row.In) != len(node.In) || len(row.Out) != len(node.Out) || len(row.Compare) != len(node.Out) {
		return fmt.Errorf("input/output/mask arity")
	}
	r := &scriptRand{trace: row.RNG}
	got := node.F(row.In, r)
	if r.pos != len(r.trace) {
		return fmt.Errorf("unused RNG calls: %d", len(r.trace)-r.pos)
	}
	if len(got) != len(row.Out) {
		return fmt.Errorf("F output arity")
	}
	compared := 0
	for i, yes := range row.Compare {
		if yes {
			compared++
			if got[i] != row.Out[i] {
				return fmt.Errorf("%s: got %d want %d", node.Out[i], got[i], row.Out[i])
			}
		}
	}
	if compared == 0 {
		return fmt.Errorf("empty comparison mask")
	}
	return nil
}

func TestFNative(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(emuDirectory(), "../emu-f/*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Skip("no build/emu-f/*.jsonl")
	}
	for _, file := range files {
		addr := strings.TrimSuffix(filepath.Base(file), ".jsonl")
		t.Run(addr, func(t *testing.T) {
			proof := strings.TrimSuffix(file, ".jsonl") + ".d3f"
			if e := os.Remove(proof); e != nil && !os.IsNotExist(e) {
				t.Fatal(e)
			}
			f, e := os.Open(file)
			if e != nil {
				t.Fatal(e)
			}
			defer f.Close()
			scan := bufio.NewScanner(f)
			scan.Buffer(make([]byte, 4096), 16*1024*1024)
			cases, diff := 0, 0
			var first error
			for scan.Scan() {
				cases++
				var row fNativeRow
				e := json.Unmarshal(scan.Bytes(), &row)
				if e == nil {
					e = replayFNative(Nodes[addr], row)
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
			if e := os.WriteFile(proof, []byte(fmt.Sprintf("cases=%d diff=%d\n", cases, diff)), 0644); e != nil {
				t.Fatal(e)
			}
			if first != nil {
				t.Errorf("cases=%d diff=%d: %v", cases, diff, first)
			}
		})
	}
}

func TestFNativeContract(t *testing.T) {
	node := Node{In: []string{"x"}, Out: []string{"value", "record"}, F: func(in []int64, r Rand) []int64 { return []int64{int64(r.R(3)), 99} }, Emu: func([]int64, Rand) []int64 { panic("must use F") }}
	good := fNativeRow{emuRow: emuRow{In: []int64{0}, RNG: [][2]int{{3, 2}}, Out: []int64{2, 0}}, Compare: []bool{true, false}}
	if e := replayFNative(node, good); e != nil {
		t.Fatal(e)
	}
	for _, trace := range [][][2]int{nil, {{3, 2}, {3, 1}}, {{4, 2}}, {{3, 3}}} {
		bad := good
		bad.RNG = trace
		if replayFNative(node, bad) == nil {
			t.Fatalf("accepted bad RNG %v", trace)
		}
	}
	bad := good
	bad.Compare = nil
	if replayFNative(node, bad) == nil {
		t.Fatal("accepted missing mask")
	}
	bad = good
	bad.Out = []int64{1, 0}
	if replayFNative(node, bad) == nil {
		t.Fatal("accepted F difference")
	}
}
