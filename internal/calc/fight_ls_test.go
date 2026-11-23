package calc

import (
	"math"
	"testing"
)

// LS（F02–F04）：武士主手劍每回合以 scale(L,12)/100 抽中，抽中時第一次主手攻擊 +4 揮（L≥20）。
func TestFightLightningStrikes(t *testing.T) {
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build 原版檔：", err)
	}
	swings := func(class, main int) float64 {
		t.Helper()
		kv, err := w.Fight(FightScenario{Race: 4, Class: class, Level: 28, Prio: []int{0, 4, 6, 5}, Skills: map[int]int{0: 100},
			Main: &FightWeapon{Item: main}, Monster: 564, Difficulty: 2}, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range kv {
			if v.Key == "round.swings" {
				return v.Val
			}
		}
		t.Fatal("missing round.swings")
		return 0
	}
	if !w.x.HasFeature(4, 4, 0x10) || w.x.HasFeature(0, 4, 0x10) {
		t.Fatal("前提：武士有能力 0x10、戰士沒有")
	}
	// Muramasa（41）同一把劍：武士比戰士多 p×4（攻擊數相同時）；p＝lsLevelScale(28)/100。
	p := float64(lsLevelScale(28)) / 100
	if p != 0.12 {
		t.Fatalf("lsLevelScale(28)=%g, want 12", p*100)
	}
	c4, err := w.fightChar(FightScenario{Race: 4, Class: 4, Level: 28, Prio: []int{0, 4, 6, 5}, Skills: map[int]int{0: 100}, Main: &FightWeapon{Item: 41}, Monster: 564, Difficulty: 2})
	if err != nil {
		t.Fatal(err)
	}
	hs, err := c4.hands(newFightReg())
	if err != nil {
		t.Fatal(err)
	}
	base := float64(hv(hs, 0, hAttacks)) * float64(hv(hs, 0, hSwings)+1) / 2
	if got := swings(4, 41); math.Abs(got-(base+p*4)) > 1e-3 {
		t.Errorf("Samurai Muramasa swings=%g, want %g (base %g + %g×4)", got, base+p*4, base, p)
	}
}
