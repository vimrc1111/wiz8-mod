package calc

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

// Synthetic tables: only race 10/class 2 is affordable; no game table data.
func partyTables(b int) *ExeTables {
	x := &ExeTables{}
	for c := range x.ClassMin {
		x.ClassMin[c] = [7]int{100, 100, 100, 100, 100, 100, 100}
	}
	x.Race[10] = [7]int{9, 11, 13, 15, 17, 19, 1}
	x.ClassMin[2] = x.Race[10]
	x.ClassMin[2][6] += 60 - b
	return x
}

func partyMetrics(t *testing.T, s string) []PartyMetric {
	t.Helper()
	m, err := ParsePartyMetrics(s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// Exhaustive oracle deliberately enumerates partial allocations too, and
// compares metric vectors pairwise instead of using production pruning.
func partyOracle(x *ExeTables, ms []PartyMetric) map[string]bool {
	values := map[string][]int{}
	var indices []int
	for i := 0; i < 7; i++ {
		for _, m := range ms {
			if m.Attr == i {
				indices = append(indices, i)
				break
			}
		}
	}
	for r := 0; r < 11; r++ {
		for c := 0; c < 15; c++ {
			a := [7]int{}
			b := 60
			for i := 0; i < 7; i++ {
				a[i] = max(x.Race[r][i], x.ClassMin[c][i])
				b -= max(0, x.ClassMin[c][i]-x.Race[r][i])
			}
			if b < 0 {
				continue
			}
			k := min(b, max(3, (b+2)/3))
			var walk func(int, int)
			walk = func(j, left int) {
				if j == len(indices) {
					v := make([]int, len(ms))
					for n, m := range ms {
						v[n] = a[m.Attr]
						if m.Level {
							v[n] = -(1 + (100-a[m.Attr]+2)/3)
						}
					}
					values[fmt.Sprint(v)] = v
					return
				}
				i := indices[j]
				base := a[i]
				for add := 0; add <= k && add <= left && base+add <= 100; add++ {
					a[i] = base + add
					walk(j+1, left-add)
				}
				a[i] = base
			}
			walk(0, b)
		}
	}
	front := map[string]bool{}
	for key, v := range values {
		dominated := false
		for other, w := range values {
			if key == other {
				continue
			}
			all := true
			for i := range v {
				if w[i] < v[i] {
					all = false
					break
				}
			}
			if all {
				dominated = true
				break
			}
		}
		if !dominated {
			front[key] = true
		}
	}
	return front
}

func checkParty(t *testing.T, x *ExeTables, ms []PartyMetric) []PartyPoint {
	t.Helper()
	got, err := Party(x, ms)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]bool{}
	for _, p := range got {
		if p.Race < 0 || p.Race > 10 || p.Class < 0 || p.Class > 14 || p.Gender < 0 || p.Gender > 1 || p.Class == 2 && p.Gender != 1 {
			t.Fatalf("invalid identity: %+v", p)
		}
		b := 60
		for i := 0; i < 7; i++ {
			b -= max(0, x.ClassMin[p.Class][i]-x.Race[p.Race][i])
		}
		if b < 0 {
			t.Fatalf("negative budget: %+v", p)
		}
		spent := 0
		k := min(b, max(3, (b+2)/3))
		for i, a := range p.Attrs {
			add := a - max(x.Race[p.Race][i], x.ClassMin[p.Class][i])
			used := false
			for _, m := range ms {
				used = used || m.Attr == i
			}
			if add < 0 || add > k || a > 100 || !used && add != 0 {
				t.Fatalf("invalid allocation: %+v", p)
			}
			spent += add
		}
		if spent > b {
			t.Fatalf("overspent: %+v", p)
		}
		v := make([]int, len(ms))
		for n, m := range ms {
			v[n] = m.Value(p.Attrs)
			if m.Level {
				v[n] = -v[n]
			}
		}
		key := fmt.Sprint(v)
		if values[key] {
			t.Fatalf("duplicate vector %s", key)
		}
		values[key] = true
	}
	if want := partyOracle(x, ms); !reflect.DeepEqual(values, want) {
		t.Fatalf("front=%v want=%v", values, want)
	}
	return got
}

func TestPartyFrontier(t *testing.T) {
	rng := rand.New(rand.NewSource(17))
	for trial := 0; trial < 18; trial++ {
		x := partyTables(trial % 9)
		// Competing races and classes test global, mixed-direction dominance.
		x.Race[9] = x.Race[10]
		x.Race[9][0] += rng.Intn(5)
		x.Race[9][1] -= rng.Intn(5)
		x.ClassMin[14] = x.ClassMin[2]
		x.ClassMin[14][0] += rng.Intn(4)
		for _, s := range []string{"attr:0", "l100:1", "attr:0,l100:1", "l100:0,l100:1", "attr:0,attr:1,attr:2,attr:3", "attr:0,l100:0,attr:0"} {
			t.Run(fmt.Sprintf("%d/%s", trial, s), func(t *testing.T) { checkParty(t, x, partyMetrics(t, s)) })
		}
	}
}

func TestPartyLimits(t *testing.T) {
	for _, b := range []int{0, 1, 2, 3, 4, 8, 10, 59, 60} {
		x := partyTables(b)
		p := checkParty(t, x, partyMetrics(t, "attr:0"))
		want := 9 + min(b, max(3, (b+2)/3))
		if len(p) != 1 || p[0].Attrs[0] != want || p[0].Class != 2 || p[0].Gender != 1 {
			t.Fatalf("B=%d: %v want attr=%d", b, p, want)
		}
	}
	x := partyTables(6)
	x.Race[10][0] = 99
	x.ClassMin[2][0] = 99
	p := checkParty(t, x, partyMetrics(t, "attr:0,l100:0"))
	if len(p) != 1 || p[0].Attrs[0] != 100 {
		t.Fatalf("100 cap: %v", p)
	}
	x = partyTables(-1)
	if p := checkParty(t, x, partyMetrics(t, "attr:0")); len(p) != 0 {
		t.Fatalf("negative B: %v", p)
	}
}

func TestPartyInvalid(t *testing.T) {
	for _, s := range []string{"", "attr", "attr:-1", "attr:7", "attr:00", "attr:+1", "foo:0", "attr:1,", " attr:0", "l100:6:0"} {
		if _, err := ParsePartyMetrics(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
	if _, err := Party(nil, partyMetrics(t, "attr:0")); err == nil {
		t.Error("nil tables accepted")
	}
	if _, err := Party(partyTables(0), nil); err == nil {
		t.Error("no metrics accepted")
	}
	x := partyTables(0)
	x.Race[0][0] = 101
	if _, err := Party(x, partyMetrics(t, "attr:0")); err == nil {
		t.Error("invalid table accepted")
	}
}
