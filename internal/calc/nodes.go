package calc

import (
	"fmt"
	"sort"
)

// Rand supplies one original R(n) call, including R(0)=0.
type Rand interface{ R(n int) int }
type Node struct {
	Addr    string
	In, Out []string
	F       func(in []int64, r Rand) []int64
	// Emu, when set, is what TestEmu replays: the case address-stubs D4 formula
	// callees (decisions/2026-10-09-compositional-stubs.md), so their returns come
	// from inputs. F must share Emu's body and call the real callee Go functions.
	Emu func(in []int64, r Rand) []int64
}

var Nodes = map[string]Node{
	"00517970": {Addr: "00517970", In: []string{"Z", "N", "S"}, Out: []string{"value"}, F: diceNode},
	"00552250": {Addr: "00552250", In: []string{"value", "Rj", "Lt", "S", "targetType"}, Out: []string{"value"}, F: resistNode},
}

// singleton extracts a deterministic specialization of an existing formula.
func singleton(d Dist) int {
	if len(d) != 1 {
		panic("node: expected deterministic formula")
	}
	for v := range d {
		return v
	}
	panic("node: empty distribution")
}

func diceNode(in []int64, r Rand) []int64 {
	if len(in) != 3 || in[0] < -32768 || in[0] > 32767 || in[1] < 0 || in[1] > 255 || in[2] < 0 || in[2] > 255 {
		panic("dice: expected i16 Z, u8 N, u8 S")
	}
	// DiceDist with S=0 includes Z and every die's fixed +1. Replay only the random parts.
	value := singleton(DiceDist(Dice{Z: int16(in[0]), N: uint8(in[1])}))
	for i := int64(0); i < in[1]; i++ {
		value += r.R(int(in[2]))
	}
	return []int64{int64(value)}
}

func resistNode(in []int64, r Rand) []int64 {
	if len(in) != 5 || in[0] < 0 || in[0] > 42949672 || in[1] < 0 || in[1] > 255 || in[2] < 0 || in[2] > 255 || in[3] < 0 || in[3] > 255 || in[4] < 1 || in[4] > 3 {
		panic("resist: expected value 0..42949672, Rj/Lt/S 0..255, targetType 1..3")
	}
	v, resistance, level, strength := int(in[0]), int(in[1]), int(in[2]), int(in[3])
	er := resistance + 3*(level-strength)
	if v == 9999 || er <= 0 {
		return []int64{int64(singleton(Resist(Dist{v: 1}, resistance, level, strength)))}
	}
	h := (er + 1) / 2
	// Preserve the two original calls even when resistance is certainly capped.
	fr := min(100, h+r.R(er+1-h)+r.R(h))
	// ER=1 makes Resist's FR exactly 1. Its reduction on v*fr is floor(v*fr/100).
	// This composition shares Resist's rounding and sentinel behavior; product=9999
	// needs a different deterministic specialization because 9999 is a sentinel.
	product := v * fr
	reduced := 0
	if product == 9999 {
		// 9999/100 has the same floor as 9998/100, which is not a sentinel.
		reduced = 9998 - singleton(Resist(Dist{9998: 1}, 1, 0, 0))
	} else {
		reduced = product - singleton(Resist(Dist{product: 1}, 1, 0, 0))
	}
	return []int64{int64(v - reduced)}
}

// NodeOutcome describes the joint distribution of all outputs.
type NodeOutcome struct {
	Out         []int64
	Probability float64
}
type nodeBranch struct{ n int }
type prefixRand struct {
	values []int
	pos    int
}

func (r *prefixRand) R(n int) int {
	if n < 0 || n > 65535 {
		panic("NodeDist: R(n) outside 0..65535")
	}
	if r.pos == len(r.values) {
		panic(nodeBranch{n})
	}
	v := r.values[r.pos]
	r.pos++
	if v < 0 || v >= max(1, n) {
		panic("NodeDist: invalid replay prefix")
	}
	return v
}
func runPrefix(node Node, in []int64, prefix []int) (out []int64, branch *nodeBranch, err error) {
	defer func() {
		if p := recover(); p != nil {
			if b, ok := p.(nodeBranch); ok {
				branch = &b
			} else {
				err = fmt.Errorf("node %s: %v", node.Addr, p)
			}
		}
	}()
	r := &prefixRand{values: prefix}
	out = node.F(append([]int64(nil), in...), r)
	if r.pos != len(prefix) {
		return nil, nil, fmt.Errorf("node %s: unused RNG prefix", node.Addr)
	}
	if len(out) != len(node.Out) {
		return nil, nil, fmt.Errorf("node %s: output arity", node.Addr)
	}
	return out, nil, nil
}

// NodeDist is named separately from the pre-existing scalar distribution type Dist.
// It enumerates complete paths, weighting each R(n) by all 32768 CRT rand values.
// Resource limits return an error, never a partial or sampled distribution.
func NodeDist(node Node, in []int64) ([]NodeOutcome, error) {
	if len(in) != len(node.In) {
		return nil, fmt.Errorf("node %s needs %d inputs", node.Addr, len(node.In))
	}
	outcomes := map[string]*NodeOutcome{}
	weights := map[int][]int{}
	visits := 0
	var walk func([]int, float64) error
	walk = func(prefix []int, probability float64) error {
		visits++
		if visits > 1000000 || len(prefix) > 256 {
			return fmt.Errorf("NodeDist: enumeration limit exceeded")
		}
		out, branch, err := runPrefix(node, in, prefix)
		if err != nil {
			return err
		}
		if branch == nil {
			key := fmt.Sprint(out)
			if outcomes[key] == nil {
				outcomes[key] = &NodeOutcome{Out: append([]int64(nil), out...)}
			}
			outcomes[key].Probability += probability
			return nil
		}
		n := branch.n
		counts, ok := weights[n]
		if !ok {
			counts = make([]int, max(1, n))
			for raw := uint64(0); raw <= 32767; raw++ {
				v := 0
				if n > 0 {
					v = int(uint32(raw*uint64(n))/32767) % n
				}
				counts[v]++
			}
			weights[n] = counts
		}
		for v, count := range counts {
			if count > 0 {
				if err := walk(append(prefix, v), probability*float64(count)/32768); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(nil, 1); err != nil {
		return nil, err
	}
	result := make([]NodeOutcome, 0, len(outcomes))
	for _, out := range outcomes {
		result = append(result, *out)
	}
	sort.Slice(result, func(i, j int) bool {
		for k, v := range result[i].Out {
			if v != result[j].Out[k] {
				return v < result[j].Out[k]
			}
		}
		return false
	})
	return result, nil
}
