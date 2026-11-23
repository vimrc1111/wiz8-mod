package calc

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestFightHasteAttrs(t *testing.T) {
	w := fightTestWorld(t)
	base := FightScenario{Race: 0, Class: 0, Level: 28, Prio: []int{0, 4, 5, 3}, Skills: map[int]int{0: 100, 3: 100, 16: 100, 18: 100}, Main: &FightWeapon{Item: 62}, Off: &FightWeapon{Item: 25}, Monster: 564, Difficulty: 1}
	fight := func(s FightScenario) []FightKV {
		t.Helper()
		kv, err := w.Fight(s, nil)
		if err != nil {
			t.Fatal(err)
		}
		return kv
	}
	// Fight sums map-enumerated branches; compare every cell at TestRank tolerance.
	equal := func(t *testing.T, got, want []FightKV) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("output length %d want %d", len(got), len(want))
		}
		for i, v := range want {
			if got[i].Key != v.Key || math.IsNaN(got[i].Val) || math.Abs(got[i].Val-v.Val) > 1e-12*math.Max(1, math.Abs(v.Val)) {
				t.Fatalf("cell %d: %+v want %+v", i, got[i], v)
			}
		}
	}

	func() {
		attrs := [7]int{100, 100, 100, 100, 100, 97, 100}
		s := base
		s.Attrs, s.Haste = &attrs, true
		before := attrs
		got := fight(s)
		if attrs != before {
			t.Fatal("Fight mutated Attrs")
		}
		attrs[5], s.Haste = 125, false
		equal(t, got, fight(s))
		c, err := w.fightChar(s)
		if err != nil {
			t.Fatal(err)
		}
		reg := newFightReg()
		if _, err := c.hands(reg); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(reg.rows["004ee220"]["speed"].val, "Attrs") || strings.Contains(reg.rows["004ee220"]["meleeExpert"].val, "PlanAttrs") {
			t.Fatal("override provenance missing")
		}
		s.Haste = true
		c, err = w.fightChar(s)
		if err != nil {
			t.Fatal(err)
		}
		reg = newFightReg()
		if _, err := c.hands(reg); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(reg.rows["004ee220"]["speed"].val, "Haste") {
			t.Fatal("Haste provenance missing")
		}
	}()
	func() {
		// Public Attrs rejects 130; exercise the transformation above its input bound.
		attrs := [7]int{100, 100, 100, 100, 100, 130, 100}
		c := &fightChar{s: FightScenario{Attrs: &attrs, Haste: true}, attr: fightHasteAttrs(attrs, true)}
		reg := newFightReg()
		b := reg.bind("004ee220")
		c.attrIn(b, "speed", 5)
		if got := b.in[b.idx["speed"]]; got != 130 {
			t.Fatalf("SPD cell = %d want 130", got)
		}
	}()
	func() {
		// Without Haste the SPD override stays as given (and differs from the hasted fight).
		attrs := [7]int{100, 100, 100, 100, 100, 97, 100}
		s := base
		s.Attrs = &attrs
		c, err := w.fightChar(s)
		if err != nil {
			t.Fatal(err)
		}
		if c.attr[5] != 97 {
			t.Fatalf("SPD %d without Haste want 97", c.attr[5])
		}
	}()
	func() {
		want := fight(base)
		s := base
		s.Attrs = nil
		equal(t, fight(s), want)
		attrs, err := PlanAttrs(w.x, base.Race, base.Class, base.Level, base.Prio)
		if err != nil {
			t.Fatal(err)
		}
		s.Attrs = &attrs
		equal(t, fight(s), want)
	}()
	func() {
		tmpl := RankTemplate{Races: []int{0}, Classes: []int{0}, Attrs: []int{0, 4}, Levels: []int{1, 10, 28}, SkillPrio: []int{0, 3, 16, 18}, Train: map[int]int{0: 100, 3: 100, 16: 100, 18: 100}, Main: base.Main, Off: base.Off, Monster: base.Monster, Difficulty: 1, AttrMin: 100, Haste: true}
		rows, err := Rank(w, tmpl)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Fatalf("got %d rows want 2", len(rows))
		}
		for _, row := range rows {
			if len(row.Damage) != len(tmpl.Levels) {
				t.Fatal("missing level columns")
			}
			for k, lv := range tmpl.Levels {
				attrs, err := PlanAttrs(w.x, row.Race, row.Class, lv, row.Prio)
				if err != nil {
					t.Fatal(err)
				}
				for i := range attrs {
					if attrs[i] < 100 {
						attrs[i] = 100
					}
				}
				if attrs[5] < 125 {
					attrs[5] = 125
				}
				raw, err := PlanSkills(w.x, row.Race, row.Class, lv, row.Prio, row.SkillPrio)
				if err != nil {
					t.Fatal(err)
				}
				skills := map[int]int{}
				for id, v := range raw {
					skills[id] = min(100, v+tmpl.Train[id])
				}
				s := base
				s.Race, s.Class, s.Level, s.Prio, s.Attrs, s.Skills = row.Race, row.Class, lv, row.Prio, &attrs, skills
				found := false
				for _, kv := range fight(s) {
					if kv.Key == "round.damage" {
						found = true
						got := row.Damage[k]
						if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-kv.Val) > 1e-12*math.Max(1, math.Abs(kv.Val)) {
							t.Fatalf("prio %v level %d damage %.17g want %.17g", row.Prio, lv, got, kv.Val)
						}
					}
				}
				if !found {
					t.Fatal("missing round.damage")
				}
			}
		}
		_, _, jobs, cells, err := rankPlan(w, tmpl)
		if err != nil {
			t.Fatal(err)
		}
		if len(jobs) != len(tmpl.Levels) || cells != 2*len(tmpl.Levels) {
			t.Fatalf("modified attrs not deduplicated: jobs %d cells %d", len(jobs), cells)
		}
		for _, job := range jobs {
			if !job.scenario.Haste || job.scenario.Attrs == nil || job.scenario.Attrs[5] != 125 {
				t.Fatal("rank did not forward modified attrs and Haste")
			}
		}
		for _, bad := range []int{-1, 126} {
			tmpl.AttrMin = bad
			if _, err := Rank(w, tmpl); err == nil {
				t.Fatalf("AttrMin %d accepted", bad)
			}
		}
	}()
	func() {
		for i := 0; i < 7; i++ {
			for _, bad := range []int{-1, 126, 130} {
				attrs := [7]int{100, 100, 100, 100, 100, 97, 100}
				attrs[i] = bad
				s := base
				s.Attrs, s.Haste = &attrs, true
				if _, err := w.Fight(s, nil); err == nil {
					t.Fatalf("Attrs[%d]=%d accepted", i, bad)
				}
			}
		}
	}()
	func() {
		d := json.NewDecoder(strings.NewReader(`{"Attrs":[100,100,100,100,100,97,100],"Haste":true}`))
		d.DisallowUnknownFields()
		var s FightScenario
		if err := d.Decode(&s); err != nil {
			t.Fatal(err)
		}
		if s.Attrs == nil || s.Attrs[5] != 97 || !s.Haste {
			t.Fatal("new JSON fields not decoded")
		}
		d = json.NewDecoder(strings.NewReader(`{"Attrs":[100,100,100,100,100,97,100],"Unknown":true}`))
		d.DisallowUnknownFields()
		if err := d.Decode(&s); err == nil {
			t.Fatal("unknown JSON field accepted")
		}
		if _, err := ReadRankTemplate(strings.NewReader(`{"AttrMin":100,"Haste":true,"SkillPrio":[]}`)); err != nil {
			t.Fatal(err)
		}
	}()
}
