// 票 R 驗收（cc 寫，執行方不改）。期望值由 cc 以獨立參考實作（scratchpad ref8.py，照 formulas.md F06–F14、
// growth.md 配點、combat-2.md C04/C06）算出，G2 另手算：u=⌊2(125+200)/3⌋=216，K=108，a=⌊320/3⌋=106，
// v=⌊(100−50+112+108)/3⌋=90，BARraw=60+⌊(50+216+112)/3⌋=186，J=5+5+0+(1+⌊58/20⌋)=13，Δ=17→+1，
// ARhit=186+65−30−125+1=97。
package calc

import (
	"os"
	"path/filepath"
	"testing"
)

// 只放計算用到的列（種族 Human=0、Lizardman=6，職業 Fighter=0）。
func fighterTables() *ExeTables {
	x := &ExeTables{}
	x.Race[0] = [7]int{45, 45, 45, 45, 45, 45, 45}
	x.Race[Lizardman] = [7]int{60, 25, 25, 70, 40, 50, 30}
	x.ClassMin[Fighter] = [7]int{55, 0, 0, 50, 50, 0, 0}
	x.MainSkill[Fighter] = SkillCloseCombat
	x.PMult[Fighter] = 4
	return x
}

var prio = []int{0, 4, 5, 3} // STR, DEX, SPD, VIT；其餘依索引序

// growth.md：創角 B=60−D、每項 ≤K=min(B,max(3,⌈B/3⌉))；每級 6 點、每項 ≤3、裸值 ≤100。
func TestPlanAttrs(t *testing.T) {
	x := fighterTables()
	for _, c := range []struct {
		race, L int
		want    [7]int
	}{
		{Lizardman, 1, [7]int{77, 25, 25, 70, 67, 66, 30}},
		{Lizardman, 5, [7]int{89, 25, 25, 70, 79, 66, 30}},
		{Lizardman, 28, [7]int{100, 49, 43, 100, 100, 100, 30}},
		{0, 1, [7]int{69, 45, 45, 50, 64, 57, 45}},
		{0, 28, [7]int{100, 52, 45, 95, 100, 100, 45}},
	} {
		got, err := PlanAttrs(x, c.race, Fighter, c.L, prio)
		if err != nil || got != c.want {
			t.Errorf("PlanAttrs(race %d, L%d) = %v, %v; want %v", c.race, c.L, got, err, c.want)
		}
	}
	x.ClassMin[Fighter] = [7]int{100, 100, 0, 0, 0, 0, 0} // D=110>60
	if _, err := PlanAttrs(x, 0, Fighter, 1, prio); err == nil {
		t.Error("unpayable class minimum must error")
	}
}

// abilities.md：主技能 b+max(1,⌊b/4⌋)，夾 0..125；非主技能原值。
func TestEffectiveSkill(t *testing.T) {
	x := fighterTables()
	for _, c := range []struct{ s, raw, want int }{
		{SkillCloseCombat, 100, 125}, {SkillCloseCombat, 0, 1}, {SkillCloseCombat, 10, 12}, {SkillSword, 100, 100},
	} {
		if got := EffectiveSkill(x, Fighter, c.s, c.raw); got != c.want {
			t.Errorf("EffectiveSkill(%d,%d) = %d, want %d", c.s, c.raw, got, c.want)
		}
	}
}

func fighter(t *testing.T, x *ExeTables, race, L int, sk map[int]int) Build {
	t.Helper()
	a, err := PlanAttrs(x, race, Fighter, L, prio)
	if err != nil {
		t.Fatal(err)
	}
	b := Build{Class: Fighter, Level: L, Attr: a}
	for s, v := range sk {
		b.Skill[s] = v
	}
	return b
}

type golden struct {
	main, off      HandCache
	mh, mp, oh, op int
	total          float64
}

func checkBuild(t *testing.T, name string, r BuildResult, g golden, dual bool) {
	t.Helper()
	if r.Main != g.main || r.MainHit != g.mh || r.MainPen != g.mp {
		t.Errorf("%s main = %+v hit %d pen %d; want %+v hit %d pen %d", name, r.Main, r.MainHit, r.MainPen, g.main, g.mh, g.mp)
	}
	if dual && (r.Off != g.off || r.OffHit != g.oh || r.OffPen != g.op || r.OffRound == nil) {
		t.Errorf("%s off = %+v hit %d pen %d; want %+v hit %d pen %d", name, r.Off, r.OffHit, r.OffPen, g.off, g.oh, g.op)
	}
	if !dual && r.OffRound != nil {
		t.Errorf("%s: single weapon must have nil OffRound", name)
	}
	near(t, name+" total", r.ExpectedDamage, g.total)
}

// G1：Lizardman L28 雙持；主手 a=86（≥50）、v=109（≥100），副手 a=77（≥75）、v=87（≥67）。
// F07 雙持旗標 C+0x12b5：u 再加 Dual Weapons、d=35（Wiz8.exe 0x4ee44e；grok 稽核 cXan F1）。
func TestBuildDualL28(t *testing.T) {
	x := fighterTables()
	b := fighter(t, x, Lizardman, 28, map[int]int{SkillSword: 100, SkillDagger: 29, SkillCloseCombat: 100, SkillDualWeapons: 29})
	main := Weapon{Item{Dice: Dice{0, 5, 8}, SpeedMod: 7, HitMod: 2}, SkillSword}
	off := &Weapon{Item{Dice: Dice{2, 1, 12}, SpeedMod: 3, HitMod: 1}, SkillDagger}
	r := MeleeBuild(x, b, main, off, 0, 0, Target{DefHit: 10, DefPen: 10, Attr4Sum: 200})
	checkBuild(t, "G1", r, golden{
		main: HandCache{K: 74, M: -27, A: 86, V: 109, BARraw: 154, HitMod: 13, DmgPct: 100},
		off:  HandCache{K: 57, M: -37, A: 77, V: 87, BARraw: 139, HitMod: 9, DmgPct: 50},
		mh:   176, mp: 176, oh: 141, op: 141, total: 171 + 37.05,
	}, true)
}

// G2：Human L28 雙手武器狂暴（mode 3：命中／穿透 −30、多一組骰）；a=106（≥100）、v=90；DA 20。
func TestBuildBerserkL28(t *testing.T) {
	x := fighterTables()
	b := fighter(t, x, 0, 28, map[int]int{SkillSword: 100, SkillCloseCombat: 100, SkillPowerStrike: 58})
	main := Weapon{Item{Dice: Dice{5, 5, 8}, SpeedMod: -5}, SkillSword}
	r := MeleeBuild(x, b, main, nil, 3, 0, Target{DefHit: 25, DefPen: 25, Attr4Sum: 280, DA: 20})
	checkBuild(t, "G2", r, golden{
		main: HandCache{K: 108, M: 0, A: 106, V: 90, BARraw: 186, HitMod: 13, DmgPct: 100},
		mh:   97, mp: 97, total: 376.2,
	}, false)
}

// G3：Human L5 雙持，低於所有門檻（a<50、v<67），STR 81 副手傷害／命中減半。
func TestBuildDualL5(t *testing.T) {
	x := fighterTables()
	b := fighter(t, x, 0, 5, map[int]int{SkillSword: 20, SkillDagger: 10, SkillCloseCombat: 15, SkillDualWeapons: 10})
	main := Weapon{Item{Dice: Dice{0, 2, 6}}, SkillSword}
	off := &Weapon{Item{Dice: Dice{0, 1, 4}}, SkillDagger}
	r := MeleeBuild(x, b, main, off, 0, 0, Target{DefHit: 10, DefPen: 10, Attr4Sum: 200})
	checkBuild(t, "G3", r, golden{
		main: HandCache{K: 15, M: -32, A: 23, V: 20, BARraw: 78, HitMod: 5, DmgPct: 62},
		off:  HandCache{K: 12, M: -42, A: 18, V: 15, BARraw: 73, HitMod: 3, DmgPct: 31},
		mh:   55, mp: 55, oh: 40, op: 40, total: 7.518055555555557,
	}, true)
}

// 原版資料：WIZ8_EXE＋WIZ8_DBS 都設才跑。*Light* *Sword*（Init +7、HitMod +6）＋Canezou Dagger（Init −1、HitMod +2），比兩手完整快取（grok 稽核 cXan F2）。
func TestBuildRealData(t *testing.T) {
	exe, dbs := os.Getenv("WIZ8_EXE"), os.Getenv("WIZ8_DBS")
	if exe == "" || dbs == "" {
		t.Skip("WIZ8_EXE／WIZ8_DBS 未設：原版資料驗證未跑")
	}
	x, err := LoadExe(exe)
	if err != nil {
		t.Fatal(err)
	}
	items, err := LoadItems(filepath.Join(dbs, "ITEMS.DBS"))
	if err != nil {
		t.Fatal(err)
	}
	find := func(name string) Item {
		for _, it := range items {
			if it.Name == name {
				return it
			}
		}
		t.Fatalf("item %q not found", name)
		return Item{}
	}
	b := fighter(t, x, Lizardman, 28, map[int]int{SkillSword: 100, SkillDagger: 29, SkillCloseCombat: 100, SkillDualWeapons: 29})
	m, o := Hands(x, b, Weapon{find("*Light* *Sword*"), SkillSword}, &Weapon{find("Canezou Dagger"), SkillDagger})
	wm := HandCache{K: 74, M: -27, A: 86, V: 109, BARraw: 154, HitMod: 17, DmgPct: 100}
	wo := HandCache{K: 57, M: -37, A: 77, V: 74, BARraw: 139, HitMod: 10, DmgPct: 50}
	if o == nil || m != wm || *o != wo {
		t.Errorf("Light Sword main = %+v, off %v; want %+v, %+v", m, o, wm, wo)
	}
}
