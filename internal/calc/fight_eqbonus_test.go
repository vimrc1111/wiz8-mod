package calc

import "testing"

// 0x50e980：手持物品的 +0xb1/+0xb2 技能加成與 +0xb3/+0xb4 屬性加成（Bloodlust 劍 +5、Fang 屬性 0 +10）。
func TestFightEquipBonus(t *testing.T) {
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build 原版檔：", err)
	}
	it := func(id, off int) int { return int(int8(w.items[0x10d*id+off])) }
	if w.items[0x10d*660+0xb1] != 0 || it(660, 0xb2) != 5 || w.items[0x10d*59+0xb3] != 0 || it(59, 0xb4) != 10 {
		t.Fatal("前提：Bloodlust 劍 +5、Fang 屬性 0 +10")
	}
	sc := FightScenario{Race: 8, Class: 7, Level: 28, Prio: []int{0, 4, 6, 5}, Skills: map[int]int{0: 100, 4: 100, 17: 100},
		Main: &FightWeapon{Item: 660}, Off: &FightWeapon{Item: 59}, Monster: 564, Difficulty: 2}
	c, err := w.fightChar(sc)
	if err != nil {
		t.Fatal(err)
	}
	if want := EffectiveSkill(w.x, 7, 0, 100) + 5; c.skill[0] != want {
		t.Errorf("sword skill=%d, want %d", c.skill[0], want)
	}
	base, err := PlanAttrs(w.x, 8, 7, 28, sc.Prio)
	if err != nil {
		t.Fatal(err)
	}
	if want := min(125, base[0]+10); c.attr[0] != want || base[0]+10 > 125 {
		t.Errorf("STR=%d, want %d (base %d)", c.attr[0], want, base[0])
	}
}
