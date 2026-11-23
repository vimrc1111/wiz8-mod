package calc

import "testing"

func TestPlanSkills(t *testing.T) {
	x, err := LoadExe("../../build/Wiz8.exe")
	if err != nil {
		t.Skip("沒有 build/Wiz8.exe：", err)
	}
	// Pick IDs at runtime; no exe tables are copied into the test.
	var available []int
	unavailable := -1
	for s := 0; s < 34; s++ {
		if x.Avail[s][Fighter] != 0 {
			available = append(available, s)
		} else if unavailable == -1 {
			unavailable = s
		}
	}
	if len(available) < 4 || unavailable == -1 || x.Avail[34][Fighter] == 0 {
		t.Fatal("need four ordinary usable skills, one unusable skill and skill 34")
	}
	k := available[:4]
	attrPrio := []int{0, 4, 5, 3}
	// Hand arithmetic: L24 = 5+23*3 = 74. At L25 the first
	// three spend 1 each, the fourth spends 3, and 3 points are discarded.
	for _, tc := range []struct {
		name  string
		level int
		prio  []int
		want  [4]int
	}{
		{"creation", 1, k, [4]int{5, 5, 5, 0}},
		{"level_up", 2, k, [4]int{8, 8, 8, 0}},
		{"before_cap", 24, k, [4]int{74, 74, 74, 0}},
		{"cap_spill", 25, k, [4]int{75, 75, 75, 3}},
		{"discard_remainder", 26, k, [4]int{75, 75, 75, 6}},
		{"all_capped", 50, k, [4]int{75, 75, 75, 75}},
		{"unavailable", 2, []int{unavailable, k[0], k[1], k[2]}, [4]int{8, 8, 8, 0}},
		{"single_skill_discards", 2, k[:1], [4]int{8, 0, 0, 0}},
		{"empty_priority", 2, nil, [4]int{}},
		{"priority_order", 1, []int{k[3], k[2], k[1], k[0]}, [4]int{0, 5, 5, 5}},
	} {
		var want [41]int
		for i, s := range k {
			want[s] = tc.want[i]
		}
		got, err := PlanSkills(x, Lizardman, Fighter, tc.level, attrPrio, tc.prio)
		if err != nil || got != want {
			t.Errorf("%s: got %v, %v; want %v", tc.name, got, err, want)
		}
	}

	// Derive the unlock level from PlanAttrs, not from a literal level.
	unlock := 0
	for lv := 1; lv <= 100; lv++ {
		a, err := PlanAttrs(x, Lizardman, Fighter, lv, attrPrio)
		if err != nil {
			t.Fatal(err)
		}
		if a[0] >= 100 {
			unlock = lv
			break
		}
	}
	if unlock <= 1 {
		t.Fatal("need STR to reach 100 after creation")
	}
	for _, tc := range []struct {
		name   string
		offset int
		want   int
	}{
		{"before_unlock", -1, 0},
		{"unlock", 0, 3},
		{"after_unlock", 1, 6},
	} {
		var want [41]int
		want[34] = tc.want
		got, err := PlanSkills(x, Lizardman, Fighter, unlock+tc.offset, attrPrio, []int{34})
		if err != nil || got != want {
			t.Errorf("%s: got %v, %v; want %v", tc.name, got, err, want)
		}
	}

	for _, tc := range []struct {
		name               string
		tables             *ExeTables
		race, class, level int
		attrs, skills      []int
	}{
		{"nil_tables", nil, 0, Fighter, 1, nil, nil},
		{"negative_race", x, -1, Fighter, 1, nil, nil},
		{"race_range", x, len(x.Race), Fighter, 1, nil, nil},
		{"negative_class", x, 0, -1, 1, nil, nil},
		{"class_range", x, 0, len(x.ClassMin), 1, nil, nil},
		{"zero_level", x, 0, Fighter, 0, nil, nil},
		{"negative_level", x, 0, Fighter, -1, nil, nil},
		{"negative_attr", x, 0, Fighter, 1, []int{-1}, nil},
		{"attr_range", x, 0, Fighter, 1, []int{7}, nil},
		{"negative_skill", x, 0, Fighter, 1, nil, []int{-1}},
		{"skill_range", x, 0, Fighter, 1, nil, []int{41}},
		{"duplicate_skill", x, 0, Fighter, 1, nil, []int{k[0], k[0]}},
		{"duplicate_unavailable", x, 0, Fighter, 1, nil, []int{unavailable, unavailable}},
		{"invalid_after_budget", x, 0, Fighter, 1, nil, []int{k[0], k[1], k[2], 41}},
	} {
		if _, err := PlanSkills(tc.tables, tc.race, tc.class, tc.level, tc.attrs, tc.skills); err == nil {
			t.Errorf("%s: expected error", tc.name)
		}
	}
}
