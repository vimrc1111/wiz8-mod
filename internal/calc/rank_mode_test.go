package calc

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestRankModes(t *testing.T) {
	// Hand-counted numbers of q in 0..8 for each sorted enabled mode.
	counts := [][]int{{9}, {5, 4}, {3, 3, 3}, {3, 2, 2, 2}, {2, 2, 2, 2, 1}, {2, 2, 2, 1, 1, 1}, {2, 2, 1, 1, 1, 1, 1}, {2, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 1, 1, 1, 1, 1, 1, 1}}
	for k, nums := range counts {
		mask := uint16(1<<(k+1)) - 1
		got, err := RankModeDistribution(&mask, 0)
		if err != nil {
			t.Fatal(err)
		}
		for m := range got {
			want := 0.0
			if m < len(nums) {
				want = float64(nums[m]) / 9
			}
			if got[m] != want {
				t.Fatalf("B15 k=%d mode=%d got %g want %g", k+1, m, got[m], want)
			}
		}
	}
	sparse := uint16(1<<1 | 1<<4 | 1<<8)
	got, err := RankModeDistribution(&sparse, 100)
	if err != nil || got != [9]float64{1: 1.0 / 3, 4: 1.0 / 3, 8: 1.0 / 3} {
		t.Fatalf("sparse: %v %v", got, err)
	}
	for _, skill := range []int{0, 4, 5, 125} {
		got, err := RankModeDistribution(nil, skill)
		want := [9]float64{5: 1}
		if skill >= 5 {
			want = [9]float64{5: 5.0 / 9, 6: 4.0 / 9}
		}
		if err != nil || got != want {
			t.Fatalf("unarmed skill %d: %v %v", skill, got, err)
		}
	}
	zero := uint16(0)
	if _, err := RankModeDistribution(&zero, 0); err == nil {
		t.Fatal("empty mask accepted")
	}
	for _, input := range []string{
		`{"SkillPrio":[],"Commands":["attack"],"Mode":0}`,
		`{"SkillPrio":[],"commands":["attack"],"offmode":0}`,
		`{"SkillPrio":[],"Commands":["invalid"]}`,
		`{"SkillPrio":[],"Commands":["attack","attack"]}`,
	} {
		if _, err := ReadRankTemplate(strings.NewReader(input)); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	for _, input := range []string{`{"SkillPrio":[],"Commands":null}`, `{"SkillPrio":[],"Commands":[]}`, `{"SkillPrio":[],"Mode":0}`} {
		tmpl, err := ReadRankTemplate(strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(input, "null") && !reflect.DeepEqual(tmpl.Commands, []string{"attack", "berserk"}) {
			t.Fatal("default commands")
		}
		if strings.Contains(input, "[]}") && tmpl.Commands == nil {
			t.Fatal("empty commands became legacy")
		}
	}
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build/Wiz8.exe 或 DATABASES：", err)
	}
	train := map[int]int{}
	for id := 0; id < 41; id++ {
		train[id] = 100
	}
	tmpl := RankTemplate{Attrs: []int{0, 4}, Levels: []int{10, 28}, SkillPrio: []int{0, 16, 18, 34}, Train: train, Commands: []string{"attack", "berserk"}, Monster: 564, Difficulty: 1}
	// Read all candidate fields at runtime; never bake original item records into tests.
	modes := func(wp *FightWeapon) []int {
		mask := binary.LittleEndian.Uint16(w.items[wp.Item*0x10d+0x4e:])
		var result []int
		for m := 0; m < 9; m++ {
			if mask&(1<<m) != 0 {
				result = append(result, m)
			}
		}
		return result
	}
	candidates := func(off bool) []*FightWeapon {
		var result []*FightWeapon
		for id := 0; id < w.nItem; id++ {
			r := w.items[id*0x10d : (id+1)*0x10d]
			wp := &FightWeapon{Item: id, Skill: int(r[0x46])}
			if r[0x47] > 1 || wp.Skill > 40 || len(modes(wp)) < 2 || (!off && r[0x41]&4 != 0) || (off && r[0x41]&8 == 0) {
				continue
			}
			result = append(result, wp)
		}
		return result
	}
	mains, offs := candidates(false), candidates(true)
	t.Logf("runtime candidates main=%d off=%d", len(mains), len(offs))
	find := func(berserk bool) (int, int, RankLoadout) {
		for class := 0; class < 15; class++ {
			capable := false
			for _, f := range w.x.ClassFeat[class] {
				capable = capable || f == 0x14
			}
			if capable != berserk {
				continue
			}
			for race := 0; race < 11; race++ {
				if _, err := PlanAttrs(w.x, race, class, 1, tmpl.Attrs); err != nil {
					continue
				}
				for _, main := range mains {
					for _, off := range offs {
						l := RankLoadout{Name: "dynamic", Main: main, Off: off}
						if reason, _ := loadoutOracle(w, race, class, l); reason != "" {
							continue
						}
						valid := true
						for _, prio := range [][]int{{0, 4}, {4, 0}} {
							for _, lv := range tmpl.Levels {
								attrs, err := PlanAttrs(w.x, race, class, lv, prio)
								if err != nil || !requirementsOracle(w, l, attrs, train) {
									valid = false
								}
							}
						}
						if valid {
							return race, class, l
						}
					}
				}
			}
		}
		t.Fatalf("no runtime multi-mode pair (berserk=%v)", berserk)
		return 0, 0, RankLoadout{}
	}
	race, class, l := find(true)
	tmpl.Races, tmpl.Classes = []int{race}, []int{class}
	tmpl.Loadouts = []RankLoadout{l, {Name: "single", Main: l.Main}}
	rows, skipped, err := rank(w, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 8 || len(skipped) != 0 {
		t.Fatalf("rows=%d skipped=%v", len(rows), skipped)
	}
	// Independent B15 oracle explicitly assigns the nine q outcomes to modes.
	weights := func(ms []int) map[int]float64 {
		counts := map[int]int{}
		for q := 0; q < 9; q++ {
			counts[ms[q%len(ms)]]++
		}
		result := map[int]float64{}
		for m, n := range counts {
			result[m] = float64(n) / 9
		}
		return result
	}
	for _, row := range rows {
		var loadout RankLoadout
		for _, candidate := range tmpl.Loadouts {
			if candidate.Name == row.Loadout {
				loadout = candidate
			}
		}
		if row.Command != "attack" && row.Command != "berserk" {
			t.Fatalf("command %q", row.Command)
		}
		for col, lv := range tmpl.Levels {
			raw, err := PlanSkills(w.x, row.Race, row.Class, lv, row.Prio, row.SkillPrio)
			if err != nil {
				t.Fatal(err)
			}
			skills := map[int]int{}
			for id, v := range raw {
				skills[id] = min(100, v+train[id])
			}
			main := weights(modes(loadout.Main))
			if row.Command == "berserk" {
				main = map[int]float64{3: 1}
			}
			off := map[int]float64{0: 1}
			if loadout.Off != nil {
				off = weights(modes(loadout.Off))
			}
			want := 0.0
			// Fixed mode order keeps the oracle's sum reproducible.
			for i := 0; i < 9; i++ {
				for j := 0; j < 9; j++ {
					if main[i]*off[j] == 0 {
						continue
					}
					kv, err := w.Fight(FightScenario{Race: row.Race, Class: row.Class, Level: lv, Prio: row.Prio, Skills: skills, Main: loadout.Main, Off: loadout.Off, Mode: i, OffMode: j, Monster: tmpl.Monster, Difficulty: tmpl.Difficulty}, nil)
					if err != nil {
						t.Fatal(err)
					}
					found := false
					for _, v := range kv {
						if v.Key == "round.damage" {
							want += main[i] * off[j] * v.Val
							found = true
						}
					}
					if !found {
						t.Fatal("no round.damage")
					}
				}
			}
			if math.IsNaN(row.Damage[col]) || math.IsInf(row.Damage[col], 0) || math.Abs(row.Damage[col]-want) > 1e-12*math.Max(1, math.Abs(want)) {
				t.Fatalf("%s %s level %d got %.17g want %.17g", row.Loadout, row.Command, lv, row.Damage[col], want)
			}
		}
	}
	var serial, parallel bytes.Buffer
	if err := WriteRankWorkers(&serial, w, tmpl, 0, 1); err != nil {
		t.Fatal(err)
	}
	if err := WriteRankWorkers(&parallel, w, tmpl, 0, 4); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(serial.Bytes(), parallel.Bytes()) {
		t.Fatal("weighted worker outputs differ")
	}
	if !strings.HasPrefix(serial.String(), "race\tclass\tloadout\tcommand\tprio\tsprio\t") {
		t.Fatal("command header")
	}
	for _, row := range rows {
		if !strings.Contains(serial.String(), "\t"+row.Command+"\t") {
			t.Fatal("command missing in TSV")
		}
	}
	race, class, l = find(false)
	tmpl.Races, tmpl.Classes, tmpl.Loadouts = []int{race}, []int{class}, []RankLoadout{l}
	rows, skipped, err = rank(w, tmpl)
	if err != nil || len(rows) != 2 || skipped["berserk"] != 1 {
		t.Fatalf("non-berserk rows=%d skipped=%v err=%v", len(rows), skipped, err)
	}
	for _, row := range rows {
		if row.Command != "attack" {
			t.Fatal("non-berserk class has berserk row")
		}
	}
}
