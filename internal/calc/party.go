package calc

import (
	"fmt"
	"strings"
)

// PartyMetric is an attribute to maximize or an independent level to minimize.
// Name is retained verbatim for the TSV header.
type PartyMetric struct {
	Name  string
	Attr  int
	Level bool
}

func ParsePartyMetrics(s string) ([]PartyMetric, error) {
	var metrics []PartyMetric
	for _, name := range strings.Split(s, ",") {
		parts := strings.Split(name, ":")
		if len(parts) != 2 || (parts[0] != "attr" && parts[0] != "l100") || len(parts[1]) != 1 || parts[1][0] < '0' || parts[1][0] > '6' {
			return nil, fmt.Errorf("party -metric 必須為 attr:i 或 l100:i（i=0..6），多項以逗號分隔；收到 %q", name)
		}
		metrics = append(metrics, PartyMetric{name, int(parts[1][0] - '0'), parts[0] == "l100"})
	}
	return metrics, nil
}

func (m PartyMetric) Value(attrs [7]int) int {
	if m.Level {
		return 1 + (max(0, 100-attrs[m.Attr])+2)/3
	}
	return attrs[m.Attr]
}

type PartyPoint struct {
	Race, Class, Gender int
	Attrs               [7]int
}

type partyStart struct {
	point  PartyPoint
	budget int
	cap    [7]int
}

// Party returns one witness for each globally nondominated metric vector.
// Only referenced attributes receive free points. Tables remain caller-owned.
func Party(t *ExeTables, metrics []PartyMetric) ([]PartyPoint, error) {
	if t == nil || len(metrics) == 0 {
		return nil, fmt.Errorf("party: tables and metrics are required")
	}
	var used, exact [7]bool
	for _, m := range metrics {
		if m.Attr < 0 || m.Attr > 6 {
			return nil, fmt.Errorf("party: attribute must be 0..6")
		}
		used[m.Attr] = true
		exact[m.Attr] = exact[m.Attr] || !m.Level
	}
	for _, rows := range [][][7]int{t.Race[:], t.ClassMin[:]} {
		for _, row := range rows {
			for _, a := range row {
				if a < 0 || a > 100 {
					return nil, fmt.Errorf("party: table attribute outside 0..100")
				}
			}
		}
	}
	var idx []int
	for i := range used {
		if used[i] {
			idx = append(idx, i)
		}
	}
	var starts []partyStart
	for r, race := range t.Race {
		for c, minimum := range t.ClassMin {
			s := partyStart{point: PartyPoint{Race: r, Class: c}, budget: 60}
			for i := range race {
				s.point.Attrs[i] = max(race[i], minimum[i])
				s.budget -= max(0, minimum[i]-race[i])
			}
			if s.budget < 0 {
				continue
			}
			k := min(s.budget, max(3, (s.budget+2)/3))
			for _, i := range idx {
				s.cap[i] = min(k, 100-s.point.Attrs[i])
			}
			for g := 0; g < 2; g++ {
				if c == 2 && g != 1 {
					continue
				}
				s.point.Gender = g
				starts = append(starts, s)
			}
		}
	}
	// Normalize each attribute to the minimum value giving the same metrics.
	// If attr and l100 both name it, attr is the stronger (exact) constraint.
	key := func(a [7]int) (v [7]int) {
		for _, i := range idx {
			v[i] = a[i]
			if !exact[i] {
				v[i] = 100 - 3*((100-a[i]+2)/3)
			}
		}
		return
	}
	// Test dominance without constructing the competing allocations: pay the
	// minimum cost to match every coordinate, then afford one strict improvement.
	dominated := func(v [7]int) bool {
		for _, s := range starts {
			cost, feasible, strict := 0, true, false
			var need [7]int
			for _, i := range idx {
				need[i] = max(0, v[i]-s.point.Attrs[i])
				if need[i] > s.cap[i] {
					feasible = false
					break
				}
				cost += need[i]
			}
			if !feasible || cost > s.budget {
				continue
			}
			for _, i := range idx {
				step := 1
				if !exact[i] {
					step = 3
				}
				next := max(0, v[i]+step-s.point.Attrs[i])
				if next <= s.cap[i] && cost-need[i]+next <= s.budget {
					strict = true
					break
				}
			}
			if strict {
				return true
			}
		}
		return false
	}
	var front []PartyPoint
	seen := make(map[[7]int]bool)
	// Equivalent starting regions (including genders) need only one witness.
	regions := make(map[[15]int]bool)
	for _, s := range starts {
		var region [15]int
		region[14] = s.budget
		for _, i := range idx {
			region[i], region[7+i] = s.point.Attrs[i], s.cap[i]
		}
		if regions[region] {
			continue
		}
		regions[region] = true
		var suffix [8]int
		for j := len(idx) - 1; j >= 0; j-- {
			suffix[j] = suffix[j+1] + s.cap[idx[j]]
		}
		a := s.point.Attrs
		var visit func(int, int)
		visit = func(j, left int) {
			upper := a
			for _, i := range idx[j:] {
				upper[i] += min(left, s.cap[i])
			}
			if dominated(key(upper)) {
				return
			}
			if j == len(idx) {
				v := key(a)
				if !seen[v] {
					seen[v] = true
					p := s.point
					p.Attrs = a
					front = append(front, p)
				}
				return
			}
			i := idx[j]
			for add := min(left, s.cap[i]); add >= max(0, left-suffix[j+1]); add-- {
				a[i] = s.point.Attrs[i] + add
				visit(j+1, left-add)
			}
			a[i] = s.point.Attrs[i]
		}
		// Any partial allocation extends to one spending this amount, with no
		// metric worsened. Thus this boundary contains every Pareto value vector.
		visit(0, min(s.budget, suffix[0]))
	}
	return front, nil
}
