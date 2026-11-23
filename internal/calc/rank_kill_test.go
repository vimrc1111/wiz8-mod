package calc

import (
	"encoding/binary"
	"math"
	"os"
	"strings"
	"testing"
)

func TestRankKill(t *testing.T) {
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build/Wiz8.exe 或 DATABASES：", err)
	}
	monsters, err := os.ReadFile("../../build/data-x/DATABASES/MONSTERS.DBS")
	if err != nil {
		t.Fatal(err)
	}
	tmpl := RankTemplate{Classes: []int{0}, Attrs: []int{0, 4}, Levels: []int{28}, SkillPrio: []int{0, 16, 19, 34}, Train: map[int]int{0: 100, 16: 100, 19: 100, 34: 100}, AttrMin: 100, Haste: true, Monsters: []int{564, 411}, Metric: "kill", Difficulty: 1}
	// Select a melee weapon and two legal races from the user's tables at runtime.
	for id := 0; id < w.nItem && len(tmpl.Races) < 2; id++ {
		r := w.items[id*0x10d : (id+1)*0x10d]
		mask := binary.LittleEndian.Uint16(r[0x4e:]) & 0x1ff
		if r[0x47] > 1 || r[0x46] != 0 || r[0x55] == 0 || mask == 0 || mask&(mask-1) == 0 { // own instakill: Fighter has no Critical Strike
			continue
		}
		l := RankLoadout{Name: "dynamic", Main: &FightWeapon{Item: id, Skill: int(r[0x46])}}
		var races []int
		for race := 0; race < 11; race++ {
			if _, err := PlanAttrs(w.x, race, 0, 1, tmpl.Attrs); err != nil {
				continue
			}
			if reason, _ := loadoutOracle(w, race, 0, l); reason != "" {
				continue
			}
			if !requirementsOracle(w, l, [7]int{100, 100, 100, 100, 100, 125, 100}, tmpl.Train) {
				continue
			}
			races = append(races, race)
		}
		if len(races) >= 2 {
			tmpl.Races = races[:2]
			tmpl.Loadouts = []RankLoadout{l}
		}
	}
	if len(tmpl.Races) != 2 {
		t.Fatal("no Fighter melee fixture with two legal races")
	}
	// Independent finite survival sum, without using the rank scoring helper.
	kill := func(h, d, p float64) float64 {
		if d <= 0 {
			if p > 0 {
				return p
			}
			return 0
		}
		n := 1
		if h > 0 {
			n = int(math.Ceil(h / d))
		}
		e, survive := 0.0, 1.0
		for r := 0; r < n; r++ {
			e += survive
			survive *= 1 - p
		}
		return 1 / e
	}
	closeScore := func(got, want float64) {
		t.Helper()
		if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > 1e-12*math.Max(1, math.Abs(want)) {
			t.Fatalf("got %.17g want %.17g", got, want)
		}
	}
	oracle := func(row RankRow, commands bool, metric string) float64 {
		t.Helper()
		attrs, err := PlanAttrs(w.x, row.Race, row.Class, 28, row.Prio)
		if err != nil {
			t.Fatal(err)
		}
		for i := range attrs {
			attrs[i] = max(attrs[i], 100)
		}
		attrs[5] = max(attrs[5], 125)
		raw, err := PlanSkills(w.x, row.Race, row.Class, 28, row.Prio, row.SkillPrio)
		if err != nil {
			t.Fatal(err)
		}
		skills := map[int]int{}
		for id, v := range raw {
			if w.x.Avail[id][row.Class] != 0 { // Train skips skills the class lacks
				skills[id] = min(100, v+tmpl.Train[id])
			}
		}
		main, off := [9]float64{0: 1}, [9]float64{0: 1}
		if commands {
			mask := binary.LittleEndian.Uint16(w.items[tmpl.Loadouts[0].Main.Item*0x10d+0x4e:])
			main, err = RankModeDistribution(&mask, 0)
			if err != nil {
				t.Fatal(err)
			}
			if row.Command == "berserk" {
				main = [9]float64{3: 1}
			}
		}
		total := 0.0
		for _, id := range tmpl.Monsters {
			d, p := 0.0, 0.0
			for i, pi := range main {
				for j, pj := range off {
					if pi*pj == 0 {
						continue
					}
					kv, err := w.Fight(FightScenario{Race: row.Race, Class: row.Class, Level: 28, Prio: row.Prio, Attrs: &attrs, Haste: true, Skills: skills, Main: tmpl.Loadouts[0].Main, Mode: i, OffMode: j, Monster: id, Difficulty: tmpl.Difficulty}, nil)
					if err != nil {
						t.Fatal(err)
					}
					foundD, foundP := false, false
					for _, v := range kv {
						switch v.Key {
						case "round.damage":
							d += pi * pj * v.Val
							foundD = true
						case "round.instakill":
							p += pi * pj * v.Val
							foundP = true
						}
					}
					if !foundD || !foundP {
						t.Fatal("missing round damage or instakill")
					}
				}
			}
			if id == 564 && p != 0 {
				t.Fatalf("564 p=%g, want zero", p)
			}
			if id == 411 && !(p > 0) {
				t.Fatalf("411 p=%g, want positive", p)
			}
			if metric == "damage" {
				total += d
				continue
			}
			// Decode HP independently, including the signed constant and fractional mean.
			hp := monsters[4+id*0x297+0xd6:]
			h := float64(int16(binary.LittleEndian.Uint16(hp))) + float64(hp[2])*(float64(hp[3])+1)/2
			total += kill(h, d, p)
		}
		return total / float64(len(tmpl.Monsters))
	}
	for _, metric := range []string{"kill", "damage", ""} {
		tmpl.Metric = metric
		rows, err := Rank(w, tmpl)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 4 {
			t.Fatalf("rows=%d want 4", len(rows))
		}
		for _, row := range rows {
			m := metric
			if m == "" {
				m = "damage"
			}
			closeScore(row.Damage[0], oracle(row, false, m))
		}
	}
	for _, bad := range []RankTemplate{
		func() RankTemplate { b := tmpl; b.Monster = 564; return b }(),
		func() RankTemplate { b := tmpl; b.Metric = "unknown"; return b }(),
		func() RankTemplate { b := tmpl; b.Monsters = []int{411, 411}; return b }(),
		func() RankTemplate { b := tmpl; b.Monsters = []int{-1}; return b }(),
		func() RankTemplate { b := tmpl; b.Monsters = []int{w.nMon}; return b }(),
	} {
		if _, err := Rank(w, bad); err == nil {
			t.Fatalf("accepted invalid template: %+v", bad)
		}
	}
	for _, input := range []string{
		`{"SkillPrio":[],"Metric":"unknown"}`,
		`{"SkillPrio":[],"Monsters":[411],"Monster":564}`,
	} {
		if _, err := ReadRankTemplate(strings.NewReader(input)); err == nil {
			t.Fatalf("accepted invalid JSON: %s", input)
		}
	}
	for _, c := range []struct{ h, d, p, want float64 }{
		{10, 0, .25, .25}, {10, -1, .25, .25}, {10, 0, 0, 0},
		{0, 5, 0, 1}, {-1, 5, .25, 1}, {10, 5, 0, .5},
		{11, 5, 0, 1.0 / 3}, {11, 5, 1, 1}, {11, 5, .25, 1 / (1 + .75 + .75*.75)},
	} {
		closeScore(rankKill(c.h, c.d, c.p), c.want)
	}
	tmpl.Metric = "kill"
	tmpl.Commands = []string{"attack", "berserk"}
	rows, err := Rank(w, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 8 {
		t.Fatalf("command rows=%d want 8", len(rows))
	}
	for _, row := range rows {
		closeScore(row.Damage[0], oracle(row, true, "kill"))
	}
}
