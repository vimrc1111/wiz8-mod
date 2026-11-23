package calc

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
)

func TestFightUnarmed(t *testing.T) {
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build 原版檔：", err)
	}
	exe, err := os.ReadFile("../../build/Wiz8.exe")
	if err != nil {
		t.Fatal(err)
	}
	attrs := [7]int{100, 100, 100, 100, 100, 100, 100}
	s := FightScenario{Race: 0, Class: 6, Level: 28, Attrs: &attrs, Haste: true, Skills: map[int]int{14: 100, 16: 100, 18: 100, 19: 100, 34: 100}, Monster: 564, Difficulty: 1}
	damage := func(s FightScenario) float64 {
		t.Helper()
		kv, err := w.Fight(s, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range kv {
			if v.Key == "round.damage" {
				if math.IsNaN(v.Val) || math.IsInf(v.Val, 0) || v.Val <= 0 {
					t.Fatalf("invalid damage %g", v.Val)
				}
				return v.Val
			}
		}
		t.Fatal("missing round.damage")
		return 0
	}
	hands := func(s FightScenario) ([]int64, *fightReg) {
		t.Helper()
		c, err := w.fightChar(s)
		if err != nil {
			t.Fatal(err)
		}
		g := newFightReg()
		hs, err := c.hands(g)
		if err != nil {
			t.Fatal(err)
		}
		return hs, g
	}
	hs, g := hands(s)
	eff := EffectiveSkill(w.x, 6, 14, 100)
	wantEff := 100
	if w.x.MainSkill[6] == 14 {
		wantEff += max(1, 100/4)
	} else {
		t.Logf("規格疑慮：Monk 主技能為 %d，並非 14", w.x.MainSkill[6])
	}
	if eff != wantEff {
		t.Fatalf("effective skill %d want %d", eff, wantEff)
	}
	// The hand output names its weapon skill; the effective value is its input.
	if hv(hs, 0, hWeaponSkill) != 14 {
		t.Fatalf("unarmed weapon skill %d", hv(hs, 0, hWeaponSkill))
	}
	c, _ := w.fightChar(s)
	if c.skill[int(hv(hs, 0, hWeaponSkill))] != eff {
		t.Fatal("hand effective skill differs")
	}
	wantCombined := int64(2 * (EffectiveSkill(w.x, 6, 16, 100) + 2*eff) / 3 * 10 / 20)
	if hv(hs, 0, hCombinedSkill) != wantCombined {
		t.Fatalf("combined skill %d want %d", hv(hs, 0, hCombinedSkill), wantCombined)
	}
	if !strings.Contains(g.rows["004ee220"]["itemID0"].val, "Main 空手") {
		t.Fatal("missing Main 空手 provenance")
	}
	off, err := peOffset(exe, 0x6172f0+uint32(eff/11)*4)
	if err != nil || off+4 > len(exe) {
		t.Fatalf("dice table: %v", err)
	}
	word := binary.LittleEndian.Uint32(exe[off : off+4])
	wantDice := word&0xffff0000 | uint32(uint16(word)+2)
	if hv(hs, 0, hDamageDice) != int64(wantDice) {
		t.Fatalf("dice %d want %d", hv(hs, 0, hDamageDice), wantDice)
	}
	if hv(hs, 0, hAttackModes) != 96 {
		t.Fatalf("trained modes %d want 96", hv(hs, 0, hAttackModes))
	}
	low := s
	low.Skills = map[int]int{14: 3}
	lowHS, _ := hands(low)
	if EffectiveSkill(w.x, 6, 14, 3) >= 5 || hv(lowHS, 0, hAttackModes) != 32 {
		t.Fatal("low skill modes must be 32")
	}
	damage(s)
	// I11: both hands empty → 004ee220 enables a second-hand punch; Fight must swing it.
	// MG panel 2026-10-13 (Monk, martial arts 125): kick 20–44 ×3, punch 12–30 ×2.
	var dw [4]byte
	for h, want := range [][3]int64{{20, 44, 3}, {12, 30, 2}} {
		binary.LittleEndian.PutUint32(dw[:], uint32(hv(hs, h, hDamageDice)))
		d := readDice(dw[:])
		pct := 100 + hv(hs, h, hDamagePercent)
		lo, hi := (int64(d.Z)+int64(d.N))*pct/100, (int64(d.Z)+int64(d.N)*int64(d.S))*pct/100
		if hv(hs, h, hEnabled) != 1 || lo != want[0] || hi != want[1] || hv(hs, h, hAttacks) != want[2] {
			t.Fatalf("hand%d range %d–%d ×%d want %v", h, lo, hi, hv(hs, h, hAttacks), want)
		}
	}
	hasOff := func(s FightScenario) bool {
		t.Helper()
		kv, err := w.Fight(s, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range kv {
			if strings.HasPrefix(v.Key, "off.") {
				return true
			}
		}
		return false
	}
	if !hasOff(s) {
		t.Fatal("unarmed punch hand not swung")
	}
	for _, wp := range []struct{ main, off int }{{81, -1}, {501, -1}, {59, 25}} {
		ws := s
		ws.Main = &FightWeapon{Item: wp.main}
		if wp.off >= 0 {
			ws.Off = &FightWeapon{Item: wp.off}
		}
		if hasOff(ws) != (wp.off >= 0) {
			t.Fatalf("weapon %v: offhand swing must follow the offhand item", wp)
		}
	}
	for _, input := range []string{`{}`, `{"Main":null}`} {
		var decoded FightScenario
		if err := json.Unmarshal([]byte(input), &decoded); err != nil || decoded.Main != nil {
			t.Fatal("unarmed JSON rejected")
		}
	}
	bad := s
	bad.Off = &FightWeapon{Item: 0}
	if _, err := w.Fight(bad, nil); err == nil {
		t.Fatal("accepted unarmed Main with Off")
	}
	tmpl := RankTemplate{Races: []int{0}, Classes: []int{6}, Attrs: []int{0, 4}, Levels: []int{1, 10, 28}, Loadouts: []RankLoadout{{Name: "unarmed"}}, AttrMin: 100, Haste: true, Train: map[int]int{14: 100, 16: 100, 19: 100, 34: 100}, SkillPrio: []int{14, 16, 19, 34}, Monster: 564, Difficulty: 1}
	rows, err := Rank(w, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows %d want 2", len(rows))
	}
	for _, row := range rows {
		for k, lv := range tmpl.Levels {
			a, err := PlanAttrs(w.x, row.Race, row.Class, lv, row.Prio)
			if err != nil {
				t.Fatal(err)
			}
			for id := range a {
				a[id] = max(a[id], 100)
			}
			raw, err := PlanSkills(w.x, row.Race, row.Class, lv, row.Prio, row.SkillPrio)
			if err != nil {
				t.Fatal(err)
			}
			skills := map[int]int{}
			for id, v := range raw {
				skills[id] = min(100, v+tmpl.Train[id])
			}
			test := s
			test.Level, test.Prio, test.Attrs, test.Skills = lv, row.Prio, &a, skills
			want := damage(test)
			if math.IsNaN(row.Damage[k]) || math.IsInf(row.Damage[k], 0) || math.Abs(row.Damage[k]-want) > 1e-12*math.Max(1, math.Abs(want)) {
				t.Fatalf("rank level %d damage %g want %g", lv, row.Damage[k], want)
			}
		}
	}
	// Independently weight the two unarmed modes for the Attack command.
	tmpl.Commands = []string{"attack"}
	attackRows, err := Rank(w, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	for i, row := range attackRows {
		for k, lv := range tmpl.Levels {
			a, _ := PlanAttrs(w.x, row.Race, row.Class, lv, row.Prio)
			for id := range a {
				a[id] = max(a[id], 100)
			}
			raw, _ := PlanSkills(w.x, row.Race, row.Class, lv, row.Prio, row.SkillPrio)
			skills := map[int]int{}
			for id, v := range raw {
				skills[id] = min(100, v+tmpl.Train[id])
			}
			test := s
			test.Level, test.Attrs, test.Skills = lv, &a, skills
			// I11: the punch hand draws its own mode independently.
			want := 0.0
			for _, m := range [][2]float64{{5, 5.0 / 9}, {6, 4.0 / 9}} {
				for _, o := range [][2]float64{{5, 5.0 / 9}, {6, 4.0 / 9}} {
					test.Mode, test.OffMode = int(m[0]), int(o[0])
					want += m[1] * o[1] * damage(test)
				}
			}
			if math.Abs(attackRows[i].Damage[k]-want) > 1e-12*math.Max(1, math.Abs(want)) {
				t.Fatal("unarmed Attack distribution differs")
			}
		}
	}
	unavailable := -1
	for class := range w.x.ClassMin {
		if w.x.Avail[14][class] == 0 {
			if _, err := PlanAttrs(w.x, 0, class, 1, tmpl.Attrs); err == nil {
				unavailable = class
				break
			}
		}
	}
	if unavailable < 0 {
		t.Fatal("no unavailable skill 14 class")
	}
	tmpl.Commands = nil
	tmpl.Classes = []int{unavailable}
	na, skipped, err := rank(w, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if skipped["weapon-skill"] != 1 || len(na) != 2 {
		t.Fatalf("unavailable rows %d skipped %v", len(na), skipped)
	}
	for _, row := range na {
		for _, d := range row.Damage {
			if !math.IsInf(d, -1) {
				t.Fatal("unavailable class must be NA")
			}
		}
	}
	tmpl.Classes = []int{6}
	tmpl.Loadouts[0].Off = &FightWeapon{Item: 0}
	if _, err := Rank(w, tmpl); err == nil {
		t.Fatal("rank accepted unarmed Main with Off")
	}
	parsed, err := ReadRankTemplate(strings.NewReader(`{"SkillPrio":[-1],"Loadouts":[{"Name":"unarmed"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := rankExpandSkills(w, parsed.Loadouts[0], parsed.SkillPrio)
	if err != nil || len(expanded) != 1 || expanded[0] != 14 {
		t.Fatal("unarmed main skill placeholder")
	}
}
