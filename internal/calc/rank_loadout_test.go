package calc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// These oracles read the fields directly and never call Rank's eligibility helpers.
func loadoutOracle(w *FightWorld, race, class int, l RankLoadout) (string, string) {
	masks := byte(3)
	for _, hand := range []*FightWeapon{l.Main, l.Off} {
		if hand == nil {
			continue
		}
		rec := w.items[hand.Item*269 : (hand.Item+1)*269]
		cm := uint16(rec[118]) | uint16(rec[119])<<8
		rm := uint32(rec[120]) | uint32(rec[121])<<8 | uint32(rec[122])<<16 | uint32(rec[123])<<24
		if cm>>class&1 == 0 {
			return "class-mask", ""
		}
		if rm>>race&1 == 0 {
			return "race-mask", ""
		}
		if int(rec[70]) >= 41 || w.x.Avail[rec[70]][class] == 0 {
			return "weapon-skill", ""
		}
		masks &= rec[124]
	}
	if l.Off != nil {
		if w.items[l.Main.Item*269+65]>>2&1 != 0 {
			return "two-handed", ""
		}
		if w.items[l.Off.Item*269+65]>>3&1 == 0 {
			return "offhand", ""
		}
	}
	if masks == 1 || masks == 2 {
		return "", "此武器組限單一性別"
	}
	return "", ""
}

func requirementsOracle(w *FightWorld, l RankLoadout, a [7]int, s map[int]int) bool {
	for _, hand := range []*FightWeapon{l.Main, l.Off} {
		if hand == nil {
			continue
		}
		rec := w.items[hand.Item*269 : (hand.Item+1)*269]
		for slot := 0; slot < 4; slot++ {
			pos := 125 + 2*slot
			id, minimum := int(rec[pos]), int(rec[pos+1])
			if minimum == 0 || id == 255 {
				continue
			}
			if slot < 2 {
				if id >= 7 || a[id] < minimum {
					return false
				}
			} else if id >= 41 || s[id] < minimum {
				return false
			}
		}
	}
	return true
}

func TestRankLoadouts(t *testing.T) {
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build 原版檔：", err)
	}
	// Select weapons by record properties, without persisting any item ids or names.
	picks := map[string]*FightWeapon{}
	for id := 0; id < w.nItem; id++ {
		rec := w.items[id*269 : (id+1)*269]
		if rec[70] >= 41 {
			continue
		}
		cm, rm := binary.LittleEndian.Uint16(rec[118:]), binary.LittleEndian.Uint32(rec[120:])
		if cm == 0 || rm&0x7ff == 0 {
			continue
		}
		if picks["requirement"] == nil && rec[70] == 8 && (rec[130] > 0 || rec[132] > 0) {
			picks["requirement"] = &FightWeapon{Item: id, Skill: int(rec[70])}
		}
		if picks["attribute"] == nil && rec[70] <= 4 && (rec[126] > 0 || rec[128] > 0) {
			picks["attribute"] = &FightWeapon{Item: id, Skill: int(rec[70])}
		}
		if rec[70] > 4 || binary.LittleEndian.Uint16(rec[78:])&1 == 0 {
			continue
		}
		conditions := map[string]bool{
			"base":  rec[65]&4 == 0 && rec[65]&8 != 0 && cm&1 != 0 && rm&1 != 0,
			"race":  rm&(rm-1) == 0,
			"class": cm&0x7fff != 0x7fff,
			"two":   rec[65]&4 != 0,
			"off":   rec[65]&4 == 0 && rec[65]&8 == 0,

			"gender": rec[124] == 1 || rec[124] == 2,
		}
		for kind, ok := range conditions {
			if ok && picks[kind] == nil {
				picks[kind] = &FightWeapon{Item: id, Skill: int(rec[70])}
			}
		}
	}
	for _, kind := range []string{"base", "race", "class", "two", "off", "requirement", "attribute"} {
		if picks[kind] == nil {
			t.Fatalf("no runtime fixture for %s", kind)
		}
	}
	ls := []RankLoadout{
		{Name: "base", Main: picks["base"]},
		{Name: "race", Main: picks["race"]},
		{Name: "class", Main: picks["class"]},
		{Name: "two", Main: picks["two"], Off: picks["base"]},
		{Name: "off", Main: picks["base"], Off: picks["off"]},
		{Name: "requirements-main", Main: picks["requirement"]},
		{Name: "attribute", Main: picks["attribute"]},
		{Name: "requirements-off", Main: picks["base"], Off: picks["requirement"]},
		{Name: "dual", Main: picks["base"], Off: picks["base"]},
	}
	if picks["gender"] != nil {
		ls = append(ls, RankLoadout{Name: "gender", Main: picks["gender"]})
	}
	tmpl := RankTemplate{Races: []int{0, 5, 10}, Classes: []int{0, 7, 14}, Attrs: []int{4}, Levels: []int{1, 28}, Loadouts: ls, Difficulty: 1}
	sawNA, sawNumber, sawTrainGain := false, false, false
	sawAttrNA, sawAttrRecovery := false, false
	attrsOK := func(l RankLoadout, a [7]int) bool {
		for _, hand := range []*FightWeapon{l.Main, l.Off} {
			if hand == nil {
				continue
			}
			rec := w.items[hand.Item*269 : (hand.Item+1)*269]
			for pos := 125; pos <= 127; pos += 2 {
				id, n := int(rec[pos]), int(rec[pos+1])
				if id != 255 && n > 0 && (id >= 7 || a[id] < n) {
					return false
				}
			}
		}
		return true
	}
	for _, trained := range []bool{false, true} {
		tmpl.Train = map[int]int{}
		if trained {
			for skill := 0; skill < 41; skill++ {
				tmpl.Train[skill] = 100
			}
		}
		rows, skipped, err := rank(w, tmpl)
		if err != nil {
			t.Fatal(err)
		}
		var want []RankRow
		counts := map[string]int{}
		for _, race := range tmpl.Races {
			for _, class := range tmpl.Classes {
				for _, l := range ls {
					a, err := PlanAttrs(w.x, race, class, 1, tmpl.Attrs)
					if err != nil {
						counts["creation"]++
						continue
					}
					reason, note := loadoutOracle(w, race, class, l)
					if reason != "" {
						counts[reason]++
						continue
					}
					row := RankRow{Race: race, Class: class, Loadout: l.Name, Note: note, Prio: []int{4}}
					for _, lv := range tmpl.Levels {
						a, err = PlanAttrs(w.x, race, class, lv, tmpl.Attrs)
						if err != nil {
							t.Fatal(err)
						}
						raw, err := PlanSkills(w.x, race, class, lv, tmpl.Attrs, tmpl.SkillPrio)
						if err != nil {
							t.Fatal(err)
						}
						skills := map[int]int{}
						for s, v := range raw {
							skills[s] = min(100, v+tmpl.Train[s])
						}
						if !requirementsOracle(w, l, a, skills) {
							row.Damage = append(row.Damage, math.Inf(-1))
							sawNA = true
							if !attrsOK(l, a) {
								sawAttrNA = true
							}
							continue
						}
						if lv > 1 {
							initial, _ := PlanAttrs(w.x, race, class, 1, tmpl.Attrs)
							if !attrsOK(l, initial) {
								sawAttrRecovery = true
							}
						}
						if trained {
							untrained := map[int]int{}
							for s, v := range raw {
								untrained[s] = v
							}
							if !requirementsOracle(w, l, a, untrained) {
								sawTrainGain = true
							}
						}
						kv, err := w.Fight(FightScenario{Race: race, Class: class, Level: lv, Prio: tmpl.Attrs, Skills: skills, Main: l.Main, Off: l.Off, Monster: tmpl.Monster, Difficulty: tmpl.Difficulty}, nil)
						if err != nil {
							t.Fatal(err)
						}
						found := false
						for _, v := range kv {
							if v.Key == "round.damage" {
								row.Damage = append(row.Damage, v.Val)
								found = true
								sawNumber = true
							}
						}
						if !found {
							t.Fatal("no round.damage")
						}
					}
					want = append(want, row)
				}
			}
		}
		if !reflect.DeepEqual(skipped, counts) {
			t.Fatalf("skipped got %v want %v", skipped, counts)
		}
		for _, reason := range []string{"race-mask", "class-mask", "offhand", "two-handed"} {
			if counts[reason] == 0 {
				t.Fatalf("missing coverage: %s", reason)
			}
		}
		if len(rows) != len(want) {
			t.Fatalf("row count %d want %d", len(rows), len(want))
		}
		for i, r := range rows {
			score := r.Damage
			r.Damage = nil
			expected := want[i]
			expected.Damage = nil
			if !reflect.DeepEqual(r, expected) || len(score) != len(want[i].Damage) {
				t.Fatalf("row %d differs", i)
			}
			for k, d := range score {
				e := want[i].Damage[k]
				if d != e && (math.IsInf(d, -1) || math.IsInf(e, -1) || math.Abs(d-e) > 1e-12*math.Max(1, math.Abs(e))) {
					t.Fatalf("damage row %d col %d: %g want %g", i, k, d, e)
				}
			}
		}
		var out bytes.Buffer
		if err := WriteRank(&out, w, tmpl, 0); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
		if lines[0] != "race\tclass\tloadout\tcommand\tprio\tsprio\tdmg_L1\tdmg_L28\tnote" {
			t.Fatal("TSV header")
		}
		for i, r := range rows {
			f := strings.Split(lines[i+1], "\t")
			if len(f) != 9 || f[2] != r.Loadout || f[5] != "" || f[3] != r.Command || f[8] != r.Note {
				t.Fatal("TSV metadata")
			}
			for k, d := range r.Damage {
				if math.IsInf(d, -1) {
					if f[6+k] != "NA" {
						t.Fatal("missing NA")
					}
					continue
				}
				v, err := strconv.ParseFloat(f[6+k], 64)
				if err != nil || math.Abs(v-d) > 1e-12*math.Max(1, math.Abs(d)) {
					t.Fatal("TSV damage")
				}
			}
		}
		gotCounts := map[string]int{}
		for _, line := range lines {
			if strings.HasPrefix(line, "# skipped ") {
				var reason string
				var n int
				if _, err := fmt.Sscanf(line, "# skipped %s %d", &reason, &n); err != nil {
					t.Fatal(err)
				}
				gotCounts[reason] = n
			}
		}
		if !reflect.DeepEqual(gotCounts, counts) {
			t.Fatal("TSV skipped differs")
		}
	}
	if !sawAttrNA || !sawAttrRecovery {
		t.Fatalf("attribute NA=%v recovery=%v", sawAttrNA, sawAttrRecovery)
	}
	if !sawNA || !sawNumber || !sawTrainGain {
		t.Fatalf("NA=%v numeric=%v Train gain=%v", sawNA, sawNumber, sawTrainGain)
	}
	var topOut bytes.Buffer
	topRows := []RankRow{{Loadout: "na", Damage: []float64{math.Inf(-1)}}, {Loadout: "number", Damage: []float64{0}}}
	if err := writeRankRows(&topOut, []int{1}, topRows, nil, 1); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Split(topOut.String(), "\n")[1], "\tnumber\t") {
		t.Fatal("top must rank numeric above NA")
	}
	empty := tmpl
	empty.Loadouts = []RankLoadout{}
	if rows, err := Rank(w, empty); err != nil || len(rows) != 0 {
		t.Fatal("empty loadout enumeration")
	}
	points := []RankRow{{Damage: []float64{math.Inf(-1), 2}}, {Damage: []float64{0, 2}}, {Damage: []float64{math.Inf(-1), 3}}, {Damage: []float64{0, 2}}}
	if !reflect.DeepEqual(RankPareto(points), points[1:]) {
		t.Fatal("NA Pareto dominance/equality/tradeoff")
	}
	for _, tmpl := range []RankTemplate{
		{Loadouts: ls, Main: picks["base"]}, {Loadouts: ls, Off: picks["base"]}, {Loadouts: []RankLoadout{{Name: "missing", Off: picks["base"]}}}, // no Main is unarmed (I7); unarmed with Off stays invalid
	} {
		tmpl.Levels = []int{1}
		if _, err := Rank(w, tmpl); err == nil {
			t.Fatal("accepted invalid loadout template")
		}
	}
}

func TestRankLoadoutJSON(t *testing.T) {
	for _, input := range []string{
		`{"Loadouts":[{"Name":"x","Main":{"Item":0}}],"Main":null}`,
		`{"Loadouts":[{"Name":"x","Main":{"Item":0}}],"Off":null}`,
		`{"Loadouts":[{"Main":{"Item":0},"Unknown":1}]}`,
	} {
		if _, err := ReadRankTemplate(strings.NewReader(input)); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	parsed, err := ReadRankTemplate(strings.NewReader(`{"SkillPrio":[],"Loadouts":[{"Name":"x","Main":{"Item":0},"Off":null}]}`))
	if err != nil || len(parsed.Loadouts) != 1 || parsed.Loadouts[0].Main == nil || parsed.Loadouts[0].Off != nil {
		t.Fatalf("loadout JSON: %+v %v", parsed, err)
	}
}
