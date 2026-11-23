package calc

import (
	"math"
	"testing"
)

// N5 遠程：全能槍彈藥相容取定義 ID、箭的 +0x60 進中毒池、0x11 遠程 Critical。
func TestFightRanged(t *testing.T) {
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build 原版檔：", err)
	}
	get := func(s FightScenario, key string) float64 {
		t.Helper()
		kv, err := w.Fight(s, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range kv {
			if v.Key == key {
				return v.Val
			}
		}
		t.Fatalf("missing %s", key)
		return 0
	}
	sc := func(class, main, off int) FightScenario {
		return FightScenario{Race: 0, Class: class, Level: 28, Prio: []int{4, 6, 5, 0}, Skills: map[int]int{7: 100, 8: 100, 17: 100},
			Main: &FightWeapon{Item: main}, Off: &FightWeapon{Item: off}, Monster: 564, Difficulty: 1}
	}
	b0x60 := func(id int) float64 { return float64(w.items[0x10d*id+0x60]) }

	// Omnigun Mk 12 (610) + Musket Balls (141)：0x51c930 特殊集合只認 599..610 定義 ID。
	if v := get(sc(8, 610, 141), "round.swings"); v <= 0 {
		t.Errorf("Omnigun Mk 12 + Musket Balls swings=%g, want >0", v)
	}
	// Strong Bow (771) + Viper Arrows (129)：池＝弓 +0x60 加箭 +0x60（T74≠-1）。
	if got, want := get(sc(3, 771, 129), "round.poison_pool"), b0x60(771)+b0x60(129); math.Abs(got-want) > 1e-9 || want == 0 { // 加權平均，浮點容差
		t.Errorf("Strong Bow + Viper pool=%g, want %g (>0)", got, want)
	}
	// 多箭：0x53dac7 讀 ITEMS+0xad 加進 t68（build/wiz8.asm:432039），每次攻擊箭數＝R(dice)+1＋箭數欄。
	// 期望＝attacks×((dice+1)/2＋ITEMS[739]+0xad)；Tripleshot 欄值 2，拿掉加項即差 attacks×2。
	{
		s := sc(3, 739, 139)
		c, err := w.fightChar(s)
		if err != nil {
			t.Fatal(err)
		}
		hs, err := c.hands(newFightReg())
		if err != nil {
			t.Fatal(err)
		}
		extra := float64(w.items[0x10d*739+0xad])
		want := float64(hv(hs, 0, hAttacks)) * (float64(hv(hs, 0, hSwings)+1)/2 + extra)
		if got := get(s, "round.swings"); extra != 2 || math.Abs(got-want) > 1e-3 {
			t.Errorf("Tripleshot swings=%g, want %g (ITEMS+0xad=%g, want 2)", got, want, extra)
		}
	}
	// 0x11 wsk 8：需能力 4、取 skill 17；遊俠有能力 4 時觸發率 >0（skill 19 未給）。
	if !w.x.HasFeature(3, 0, 4) {
		t.Fatal("前提：遊俠人類應有能力 4")
	}
	// 觸發率基值來自彈藥本身，所以比 skill 17＝100 與＝0：分支在才會拉開。
	hi, lo := sc(3, 771, 131), sc(3, 771, 131)
	lo.Skills = map[int]int{7: 100, 8: 100, 17: 0}
	if a, b := get(hi, "main.instakill_trigger"), get(lo, "main.instakill_trigger"); a <= b {
		t.Errorf("Ranger bow instakill_trigger skill17=100 %g, want > skill17=0 %g", a, b)
	}
}
