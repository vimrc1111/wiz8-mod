package calc

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestRank(t *testing.T) {
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build/Wiz8.exe 或 DATABASES：", err)
	}
	tmpl := RankTemplate{Races: []int{0, 4}, Classes: []int{0, 14}, Attrs: []int{0, 4}, Levels: []int{1, 10}, SkillPrio: []int{0, 16, 18, 34}, Train: map[int]int{0: 100, 16: 7}, Main: &FightWeapon{Item: 62, Skill: 0}, Off: &FightWeapon{Item: 25, Skill: 3}, Monster: 564, Difficulty: 1}
	rows, err := Rank(w, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	// Independently enumerate the two permutations and count legal combinations.
	var want []RankRow
	skipped := map[string]int{}
	loadout := RankLoadout{Name: "-", Main: tmpl.Main, Off: tmpl.Off}
	for _, race := range tmpl.Races {
		for _, class := range tmpl.Classes {
			if _, err := PlanAttrs(w.x, race, class, 1, tmpl.Attrs); err != nil {
				skipped["creation"]++
				continue
			}
			reason, note := loadoutOracle(w, race, class, loadout)
			if reason != "" {
				skipped[reason]++
				continue
			}
			for _, prio := range [][]int{{0, 4}, {4, 0}} {
				r := RankRow{Race: race, Class: class, Loadout: "-", Note: note, Prio: prio, SkillPrio: tmpl.SkillPrio}
				for _, lv := range tmpl.Levels {
					raw, err := PlanSkills(w.x, race, class, lv, prio, tmpl.SkillPrio)
					if err != nil {
						t.Fatal(err)
					}
					skills := map[int]int{}
					for s, v := range raw {
						v += tmpl.Train[s]
						if v > 100 {
							v = 100
						}
						skills[s] = v
					}
					attrs, err := PlanAttrs(w.x, race, class, lv, prio)
					if err != nil {
						t.Fatal(err)
					}
					if !requirementsOracle(w, loadout, attrs, skills) {
						r.Damage = append(r.Damage, math.Inf(-1))
						continue
					}
					scenario := FightScenario{Race: race, Class: class, Level: lv, Prio: prio, Skills: skills, Main: tmpl.Main, Off: tmpl.Off, Mode: tmpl.Mode, OffMode: tmpl.OffMode, Monster: tmpl.Monster, Difficulty: tmpl.Difficulty}
					kv, err := w.Fight(scenario, nil)
					if err != nil {
						t.Fatal(err)
					}
					found := false
					for _, v := range kv {
						if v.Key == "round.damage" {
							r.Damage = append(r.Damage, v.Val)
							found = true
						}
					}
					if !found {
						t.Fatal("Fight has no round.damage")
					}
				}
				want = append(want, r)
			}
		}
	}
	if len(want) == 0 {
		t.Fatal("fixture must have legal combinations")
	}
	if len(rows) != len(want) {
		t.Fatalf("legal row count: got %d want %d (skipped %v)", len(rows), len(want), skipped)
	}
	for i := range want {
		if rows[i].Loadout != want[i].Loadout || rows[i].Note != want[i].Note || rows[i].Race != want[i].Race || rows[i].Class != want[i].Class || !reflect.DeepEqual(rows[i].Prio, want[i].Prio) || !reflect.DeepEqual(rows[i].SkillPrio, want[i].SkillPrio) || len(rows[i].Damage) != len(want[i].Damage) {
			t.Fatalf("row %d: got %+v want %+v", i, rows[i], want[i])
		}
		for k, d := range want[i].Damage {
			if rows[i].Damage[k] != d && (math.IsInf(rows[i].Damage[k], -1) || math.IsInf(d, -1) || math.Abs(rows[i].Damage[k]-d) > 1e-12*math.Max(1, math.Abs(d))) {
				t.Fatalf("row %d level column %d: got %.17g want %.17g", i, k, rows[i].Damage[k], d)
			}
		}
	}
	// O(n²) independent dominance oracle, including equality without dominance.
	oracle := func(rs []RankRow) []RankRow {
		var front []RankRow
		for i, a := range rs {
			dominated := false
			for j, b := range rs {
				if i == j {
					continue
				}
				losses, gains := 0, 0
				for k := range a.Damage {
					if b.Damage[k] < a.Damage[k] {
						losses++
					}
					if b.Damage[k] > a.Damage[k] {
						gains++
					}
				}
				if losses == 0 && gains > 0 {
					dominated = true
				}
			}
			if !dominated {
				front = append(front, a)
			}
		}
		return front
	}
	if got := RankPareto(rows); !reflect.DeepEqual(got, oracle(rows)) {
		t.Fatalf("Pareto got %v want %v", got, oracle(rows))
	}
	equal := []RankRow{{Race: 0, Damage: []float64{1, 2}}, {Race: 1, Damage: []float64{1, 2}}, {Race: 2, Damage: []float64{0, 2}}, {Race: 3, Damage: []float64{2, 1}}}
	if !reflect.DeepEqual(RankPareto(equal), oracle(equal)) {
		t.Fatal("equal points or tradeoffs removed")
	}
	key := func(r RankRow) string {
		p := make([]string, len(r.Prio))
		for i, v := range r.Prio {
			p[i] = strconv.Itoa(v)
		}
		sp := make([]string, len(r.SkillPrio))
		for i, v := range r.SkillPrio {
			sp[i] = strconv.Itoa(v)
		}
		return fmt.Sprintf("%d\t%d\t-\t\t%s\t%s", r.Race, r.Class, strings.Join(p, ","), strings.Join(sp, ","))
	}
	for _, top := range []int{0, 1} {
		var out bytes.Buffer
		if err := writeRankRows(&out, tmpl.Levels, rows, skipped, top); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
		if lines[0] != "race\tclass\tloadout\tcommand\tprio\tsprio\tdmg_L1\tdmg_L10\tnote" {
			t.Fatalf("header: %s", lines[0])
		}
		gotSkipped := map[string]int{}
		for _, line := range lines {
			if strings.HasPrefix(line, "# skipped ") {
				var reason string
				var n int
				if _, err := fmt.Sscanf(line, "# skipped %s %d", &reason, &n); err != nil {
					t.Fatal(err)
				}
				gotSkipped[reason] = n
			}
		}
		if !reflect.DeepEqual(gotSkipped, skipped) {
			t.Fatalf("skip counts %v want %v", gotSkipped, skipped)
		}
		expected := rows
		if top == 1 {
			best := rows[0]
			for _, r := range rows[1:] {
				if r.Damage[1] > best.Damage[1] {
					best = r
				}
			}
			expected = []RankRow{best}
		}
		check := func(line string, r RankRow) {
			fields := strings.Split(line, "\t")
			if len(fields) != 9 || strings.Join(fields[:6], "\t") != key(r) || fields[8] != r.Note {
				t.Fatalf("TSV row %q want %v", line, r)
			}
			for k, d := range r.Damage {
				if math.IsInf(d, -1) {
					if fields[6+k] != "NA" {
						t.Fatal("missing NA")
					}
					continue
				}
				v, err := strconv.ParseFloat(fields[6+k], 64)
				if err != nil || math.Abs(v-d) > 1e-12*math.Max(1, math.Abs(d)) {
					t.Fatalf("TSV damage %q want %.17g", fields[6+k], d)
				}
			}
		}
		for i, r := range expected {
			check(lines[i+1], r)
		}
		idx := 1 + len(expected)
		if lines[idx] != "" || lines[idx+1] != "# pareto" {
			t.Fatalf("missing Pareto delimiter: %q", out.String())
		}
		front := oracle(rows)
		if len(lines) != idx+2+len(front)+len(skipped) {
			t.Fatalf("TSV section sizes: %q", out.String())
		}
		for i, r := range front {
			check(lines[idx+2+i], r)
		}
	}
	var live bytes.Buffer
	if err := WriteRank(&live, w, tmpl, 0); err != nil {
		t.Fatal(err)
	}
	sections := strings.Split(live.String(), "\n# pareto\n")
	if len(sections) != 2 {
		t.Fatal("missing live Pareto section")
	}
	var emitted []RankRow
	for _, line := range strings.Split(strings.TrimSuffix(sections[0], "\n"), "\n")[1:] {
		f := strings.Split(line, "\t")
		race, _ := strconv.Atoi(f[0])
		class, _ := strconv.Atoi(f[1])
		r := RankRow{Race: race, Class: class, Loadout: f[2], Note: f[len(f)-1]}
		for _, s := range strings.Split(f[4], ",") {
			v, _ := strconv.Atoi(s)
			r.Prio = append(r.Prio, v)
		}
		for _, s := range strings.Split(f[5], ",") {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatal(err)
			}
			r.SkillPrio = append(r.SkillPrio, v)
		}
		for _, s := range f[6 : len(f)-1] {
			if s == "NA" {
				r.Damage = append(r.Damage, math.Inf(-1))
				continue
			}
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				t.Fatal(err)
			}
			r.Damage = append(r.Damage, v)
		}
		emitted = append(emitted, r)
	}
	front := oracle(emitted)
	lines := strings.Split(strings.TrimSpace(sections[1]), "\n")
	if len(lines) != len(front)+len(skipped)+1 {
		t.Fatal("live Pareto count differs from emitted scores")
	}
	for i, r := range front {
		if !strings.HasPrefix(lines[i], key(r)+"\t") {
			t.Fatal("live Pareto membership differs")
		}
	}
}

func TestRankTemplate(t *testing.T) {
	for _, s := range []string{`null`, `{"Unknown":1}`, `{} {}`, `{"Main":{"Item":1,"Unknown":1}}`} {
		if _, err := ReadRankTemplate(strings.NewReader(s)); err == nil {
			t.Fatalf("accepted %s", s)
		}
	}
	if _, err := ReadRankTemplate(strings.NewReader(`{"Attrs":[0,4],"Main":{"Item":62,"Skill":0},"Train":{"16":10},"SkillPrio":[]}`)); err != nil {
		t.Fatal(err)
	}
}

// TestFightNegativeDamagePercent: race 0 class 7 L1 with prio [4,5,3,0] gets a -5 damage percent from 004ee220.
func TestFightNegativeDamagePercent(t *testing.T) {
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build/Wiz8.exe 或 DATABASES：", err)
	}
	// Exercise the original combat regression directly: Rank now filters this
	// fixture's class mask before combat, so it no longer reaches the damage path.
	tmpl := RankTemplate{Races: []int{0}, Classes: []int{7}, Attrs: []int{4, 5, 3, 0}, Levels: []int{1}, SkillPrio: []int{0, 16, 18, 34}, Main: &FightWeapon{Item: 62, Skill: 0}, Off: &FightWeapon{Item: 25, Skill: 3}, Monster: 564, Difficulty: 1}
	count := 0
	var visit func(int)
	visit = func(i int) {
		if i < len(tmpl.Attrs) {
			for j := i; j < len(tmpl.Attrs); j++ {
				tmpl.Attrs[i], tmpl.Attrs[j] = tmpl.Attrs[j], tmpl.Attrs[i]
				visit(i + 1)
				tmpl.Attrs[i], tmpl.Attrs[j] = tmpl.Attrs[j], tmpl.Attrs[i]
			}
			return
		}
		raw, err := PlanSkills(w.x, 0, 7, 1, tmpl.Attrs, tmpl.SkillPrio)
		if err != nil {
			t.Fatal(err)
		}
		skills := map[int]int{}
		for s, v := range raw {
			skills[s] = v
		}
		kv, err := w.Fight(FightScenario{Race: 0, Class: 7, Level: 1, Prio: tmpl.Attrs, Skills: skills, Main: tmpl.Main, Off: tmpl.Off, Monster: tmpl.Monster, Difficulty: tmpl.Difficulty}, nil)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, v := range kv {
			if v.Key == "round.damage" {
				found = true
				if math.IsNaN(v.Val) || math.IsInf(v.Val, 0) {
					t.Fatal("nonfinite damage")
				}
			}
		}
		if !found {
			t.Fatal("no round.damage")
		}
		count++
	}
	visit(0)
	if count != 24 {
		t.Fatalf("evaluated %d permutations want 24", count)
	}
}
