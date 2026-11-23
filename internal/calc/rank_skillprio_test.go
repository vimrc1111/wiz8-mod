package calc

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestRankSkillPrios(t *testing.T) {
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build/Wiz8.exe 或 DATABASES：", err)
	}
	tmpl := RankTemplate{Races: []int{0}, Classes: []int{0}, Attrs: []int{0, 4}, Levels: []int{1, 10}, SkillSet: []int{-1, -2, 16}, Loadouts: []RankLoadout{{Name: "pair", Main: &FightWeapon{Item: 62, Skill: 0}, Off: &FightWeapon{Item: 25, Skill: 3}}}, Monster: 564, Difficulty: 1, Train: map[int]int{16: 7}}
	// Enumerate independently with three distinct index loops, not rank helpers.
	var ps [][]int
	for a := 0; a < 3; a++ {
		for b := 0; b < 3; b++ {
			for c := 0; c < 3; c++ {
				if a != b && b != c && a != c {
					ps = append(ps, []int{tmpl.SkillSet[a], tmpl.SkillSet[b], tmpl.SkillSet[c]})
				}
			}
		}
	}
	// Recursive exchange order puts the reversed final suffix first.
	ps[4], ps[5] = ps[5], ps[4]
	if len(ps) != 6 {
		t.Fatal("oracle permutation count")
	}
	check := func(tmpl RankTemplate) {
		t.Helper()
		rows, _, fights, cells, err := rankWorkers(w, tmpl, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 12 || cells != 24 {
			t.Fatalf("rows=%d cells=%d", len(rows), cells)
		}
		unique := map[string]bool{}
		rowID := 0
		for _, p := range ps {
			var sprio []int
			for _, id := range p {
				if id == -1 {
					id = int(w.items[tmpl.Loadouts[0].Main.Item*269+70])
				}
				if id == -2 {
					id = int(w.items[tmpl.Loadouts[0].Off.Item*269+70])
				}
				found := false
				for _, prev := range sprio {
					if prev == id {
						found = true
					}
				}
				if !found {
					sprio = append(sprio, id)
				}
			}
			for _, prio := range [][]int{{0, 4}, {4, 0}} {
				r := rows[rowID]
				rowID++
				if r.Race != 0 || r.Class != 0 || r.Loadout != "pair" || !reflect.DeepEqual(r.Prio, prio) || !reflect.DeepEqual(r.SkillPrio, sprio) {
					t.Fatalf("row %d metadata %+v", rowID-1, r)
				}
				for col, lv := range tmpl.Levels {
					raw, err := PlanSkills(w.x, 0, 0, lv, prio, sprio)
					if err != nil {
						t.Fatal(err)
					}
					attrs, err := PlanAttrs(w.x, 0, 0, lv, prio)
					if err != nil {
						t.Fatal(err)
					}
					skills := map[int]int{}
					for id, v := range raw {
						raw[id] = min(100, v+tmpl.Train[id])
						skills[id] = raw[id]
					}
					l := tmpl.Loadouts[0]
					if !requirementsOracle(w, l, attrs, skills) {
						if !math.IsInf(r.Damage[col], -1) {
							t.Fatal("expected NA")
						}
						continue
					}
					// Full independently serialized Fight input, including level even when
					// saturated attributes and trained skills agree across levels.
					key := fmt.Sprintf("%d/%d/%d/%v/%v/%v/%v/%d/%d/%d/%d", 0, 0, lv, attrs, raw, *l.Main, *l.Off, tmpl.Mode, tmpl.OffMode, tmpl.Monster, tmpl.Difficulty)
					unique[key] = true
					kv, err := w.Fight(FightScenario{Race: 0, Class: 0, Level: lv, Prio: prio, Skills: skills, Main: l.Main, Off: l.Off, Mode: tmpl.Mode, OffMode: tmpl.OffMode, Monster: tmpl.Monster, Difficulty: tmpl.Difficulty}, nil)
					if err != nil {
						t.Fatal(err)
					}
					found := false
					for _, v := range kv {
						if v.Key == "round.damage" {
							found = true
							if math.Abs(r.Damage[col]-v.Val) > 1e-12*math.Max(1, math.Abs(v.Val)) {
								t.Fatalf("row %d level %d damage %.17g want %.17g", rowID-1, lv, r.Damage[col], v.Val)
							}
						}
					}
					if !found {
						t.Fatal("missing round.damage")
					}
				}
			}
		}
		if fights != len(unique) || fights >= cells {
			t.Fatalf("fights=%d want=%d cells=%d", fights, len(unique), cells)
		}
		var serial, parallel bytes.Buffer
		if err := WriteRankWorkers(&serial, w, tmpl, 0, 1); err != nil {
			t.Fatal(err)
		}
		if err := WriteRankWorkers(&parallel, w, tmpl, 0, 4); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(serial.Bytes(), parallel.Bytes()) {
			t.Fatalf("-j 1 / -j 4 differ:\n%s\n%s", serial.String(), parallel.String())
		}
		if !strings.Contains(serial.String(), fmt.Sprintf("# fights %d %d\n", len(unique), cells)) {
			t.Fatal("Fight counts missing")
		}
		if !strings.HasPrefix(serial.String(), "race\tclass\tloadout\tcommand\tprio\tsprio\t") {
			t.Fatal("sprio column missing")
		}
	}
	check(tmpl)
	// Force identical attributes and skills at two distinct levels, making an
	// omitted level in the cache key observable even if damage happens to tie.
	tmpl.Levels = []int{100, 101}
	tmpl.Train = map[int]int{}
	for id := 0; id < 41; id++ {
		tmpl.Train[id] = 100
	}
	check(tmpl)
	// Explicit priorities preserve input order, including equal expanded rows.
	tmpl.SkillSet = nil
	tmpl.SkillPrios = ps
	var explicit bytes.Buffer
	if err := WriteRankWorkers(&explicit, w, tmpl, 0, 4); err != nil {
		t.Fatal(err)
	}
	tmpl.SkillPrios = nil
	tmpl.SkillSet = []int{-1, -2, 16}
	var permuted bytes.Buffer
	if err := WriteRankWorkers(&permuted, w, tmpl, 0, 1); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(explicit.Bytes(), permuted.Bytes()) {
		t.Fatal("SkillPrios differs from independently enumerated SkillSet")
	}
	same := RankLoadout{Main: tmpl.Loadouts[0].Main, Off: tmpl.Loadouts[0].Main}
	got, err := rankExpandSkills(w, same, []int{-1, -2, 16})
	want := []int{int(w.items[same.Main.Item*269+70]), 16}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("placeholder dedup %v %v", got, err)
	}
	if _, err := rankExpandSkills(w, RankLoadout{Main: same.Main}, []int{-2}); err == nil {
		t.Fatal("accepted off placeholder without offhand")
	}
}

func TestRankSkillPrioValidation(t *testing.T) {
	for _, input := range []string{`{}`, `{"SkillPrio":null}`, `{"SkillPrio":[],"SkillSet":[]}`, `{"SkillPrio":[],"SkillPrios":null}`, `{"SkillSet":[0,0]}`, `{"SkillSet":[-3]}`, `{"SkillPrios":[[41]]}`, `{"SkillPrios":[[16,16]]}`} {
		if _, err := ReadRankTemplate(strings.NewReader(input)); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	for _, input := range []string{`{"SkillPrio":[]}`, `{"SkillSet":[]}`, `{"SkillPrios":[]}`, `{"SkillPrios":[[-1,-2,16],[16,-2,-1]]}`} {
		if _, err := ReadRankTemplate(strings.NewReader(input)); err != nil {
			t.Fatalf("rejected %s: %v", input, err)
		}
	}
}
