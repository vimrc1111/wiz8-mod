package calc

import (
	"fmt"
	"math"
	"testing"
)

// N2: weapon effect slots 1 (paralyze), 2 (poison), 6 (KO) and the kill-hi upper bound.
// Goldens follow notes/re/weapon-effects.md: per-hit success = c/100*(1-a/100), with a
// from its 目標怪抗性表 (Expert, L28; X = immune), and the Fighter KO bonus of ko-paralyze.md.
func TestRankEffects(t *testing.T) {
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build/Wiz8.exe 或 DATABASES：", err)
	}
	resist := map[int][7]float64{114: {1: 50, 2: 33, 6: 84}, 192: {1: 46, 2: -1, 6: -1}} // -1 = X
	prio := []int{0, 4, 5, 3}
	// Pick, from the user's tables, a one-handed melee weapon with only slot k set that race 0 Fighter can use.
	pick := func(k int) *FightWeapon {
		for id := 0; id < w.nItem; id++ {
			r := w.items[id*0x10d : (id+1)*0x10d]
			if r[0x47] > 1 || r[0x46] == 0xff || r[0x50+k] == 0 {
				continue
			}
			clean := true
			for _, j := range []int{1, 2, 5, 6} {
				if j != k && r[0x50+j] != 0 {
					clean = false
				}
			}
			l := RankLoadout{Main: &FightWeapon{Item: id, Skill: int(r[0x46])}}
			if reason, _ := loadoutOracle(w, 0, 0, l); !clean || reason != "" {
				continue
			}
			return l.Main
		}
		t.Fatalf("no Fighter melee weapon with only slot %d", k)
		return nil
	}
	attrs, err := PlanAttrs(w.x, 0, 0, 28, prio)
	if err != nil {
		t.Fatal(err)
	}
	fight := func(class, monster int, main *FightWeapon) map[string]float64 {
		t.Helper()
		kv, err := w.Fight(FightScenario{Race: 0, Class: class, Level: 28, Prio: prio, Attrs: &attrs, Skills: map[int]int{0: 100, 34: 58}, Main: main, Monster: monster, Difficulty: 2}, nil)
		if err != nil {
			t.Fatal(err)
		}
		m := map[string]float64{}
		for _, v := range kv {
			m[v.Key] = v.Val
		}
		return m
	}
	near := func(what string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("FAIL %s: got %.12g want %.12g", what, got, want)
		}
	}
	key := map[int]string{1: "paralyze", 2: "poison", 6: "ko"}
	// R(100) < x under the game's CRT-weighted R (c/100 in weapon-effects.md is its uniform reading).
	below := func(x float64) float64 {
		s := 0.0
		for _, q := range crtWeights(100)[:int(x)] {
			s += q
		}
		return s
	}
	for _, k := range []int{1, 2} {
		wpn := pick(k)
		c := float64(w.items[wpn.Item*0x10d+0x50+k])
		for _, mon := range []int{114, 192} {
			want := 0.0
			if a := resist[mon][k]; a >= 0 {
				want = below(c) * (1 - below(a))
			}
			// Lord (class 1) has no ability 0x13, so slot 6 stays the weapon's 0.
			m := fight(1, mon, wpn)
			near(key[k]+" per hit", m["main."+key[k]], want)
			if m["main.hit"] <= 0 {
				t.Fatalf("FAIL fixture never hits monster %d", mon)
			}
			if mon == 114 && !(m["round."+key[k]] > 0) {
				t.Fatalf("FAIL round.%s = %g, want > 0", key[k], m["round."+key[k]])
			}
		}
	}
	// Poison pool: E+0x1c = striking hand rec[0x60] (+ linked off slot when T74≠-1).
	wpn := pick(2)
	near("poison_pool", fight(1, 114, wpn)["round.poison_pool"], float64(w.items[wpn.Item*0x10d+0x60]))
	// Fighter KO (ability 0x13, L28 > 20 → +5) on a weapon without slot 6; Lord gets none.
	plain := pick(1)
	near("fighter ko", fight(0, 114, plain)["main.ko"], below(float64(fighterKO(28)))*(1-below(resist[114][6])))
	near("lord ko", fight(1, 114, plain)["main.ko"], 0)
	near("fighter ko immune", fight(0, 192, plain)["main.ko"], 0)
	for n, want := range map[int]int64{1: 3, 20: 4, 21: 5, 28: 5} { // (2n+60)*5*0.01f truncated; f32 0.01 makes n=20 give 4
		if got := fighterKO(n); got != want {
			t.Fatalf("FAIL fighterKO(%d)=%d want %d", n, got, want)
		}
	}

	// kill-hi: independent round-by-round sum against rankKillHi; equals kill with no effects.
	hi := func(h, d, p, s, q, pool float64) float64 {
		var sum, e, surv float64 = 0, 0, 1
		dis, poi := 0.0, 0.0
		for sum < h {
			dis = 1 - (1-dis)*(1-s)
			poi = 1 - (1-poi)*(1-q)
			sum += d*(1+dis) + pool*poi
			e += surv
			surv *= 1 - p
		}
		return 1 / e
	}
	for _, c := range []struct{ h, d, p, par, poi, ko, pool float64 }{
		{100, 10, 0, .3, 0, 0, 0}, {100, 10, .05, .1, .2, .3, 4}, {57.5, 3, 0, 0, .5, 0, 12}, {200, 0, .1, 0, .4, 0, 6},
	} {
		s := 1 - (1-c.par)*(1-c.ko)
		near("kill-hi", rankKillHi(c.h, c.d, c.p, [4]float64{c.par, c.poi, c.ko, c.pool}), hi(c.h, c.d, c.p, s, c.poi, c.pool))
		if lo := rankKill(c.h, c.d, c.p); rankKillHi(c.h, c.d, c.p, [4]float64{c.par, c.poi, c.ko, c.pool}) < lo {
			t.Fatalf("FAIL kill-hi < kill for %+v", c)
		}
	}
	near("kill-hi no effects", rankKillHi(100, 7, .05, [4]float64{}), rankKill(100, 7, .05))
	tmpl := RankTemplate{Races: []int{0}, Classes: []int{0}, Attrs: prio, Levels: []int{28}, SkillPrio: []int{0, 34}, Monsters: []int{114, 192}, Difficulty: 2, Loadouts: []RankLoadout{{Name: "p", Main: pick(1)}, {Name: "q", Main: pick(2)}}}
	var lo, up []RankRow
	for _, m := range []string{"kill", "kill-hi"} {
		tmpl.Metric = m
		rows, err := Rank(w, tmpl)
		if err != nil {
			t.Fatal(err)
		}
		if m == "kill" {
			lo = rows
		} else {
			up = rows
		}
	}
	if len(lo) == 0 || len(lo) != len(up) {
		t.Fatalf("FAIL rows %d/%d", len(lo), len(up))
	}
	low := map[string]float64{}
	for _, r := range lo {
		low[fmt.Sprint(r.Loadout, r.Command, r.Race, r.Class, r.Prio, r.SkillPrio)] = r.Damage[0]
	}
	gain := false
	for _, r := range up {
		l, ok := low[fmt.Sprint(r.Loadout, r.Command, r.Race, r.Class, r.Prio, r.SkillPrio)]
		if !ok || r.Damage[0] < l {
			t.Fatalf("FAIL kill-hi %g < kill %g (%s)", r.Damage[0], l, r.Loadout)
		}
		gain = gain || r.Damage[0] > l
	}
	if !gain {
		t.Fatal("FAIL kill-hi never above kill: effects not reaching rank")
	}
}
