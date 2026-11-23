package calc

import (
	"math"
	"testing"
)

// Rogue's extra strikes reach 0x5420b0's R(100) < 20*count-20 under the damage dice;
// enumerated 100 ways it cost ~700k nodes per Fight (I4: Rogue rank files 40x slower).
func TestFightRogueStrikeBranches(t *testing.T) {
	w := fightTestWorld(t)
	attrs := [7]int{100, 100, 100, 100, 100, 100, 100}
	skills := map[int]int{}
	for id := 0; id < 41; id++ {
		skills[id] = 100
	}
	s := FightScenario{Race: 0, Class: 7, Level: 28, Prio: []int{0}, Attrs: &attrs, Haste: true, Skills: skills, Main: &FightWeapon{Item: 62}, Off: &FightWeapon{Item: 25}, Monster: 564, Difficulty: 1}
	fightVisits.Store(0)
	kv, err := w.Fight(s, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n := fightVisits.Load(); n > 100000 {
		t.Fatalf("enumerated %d nodes, want <= 100000", n)
	}
	// Same distribution as the 100-way R(100) (value from calc fight before the change).
	found := false
	for _, v := range kv {
		found = found || v.Key == "round.damage"
		if v.Key == "round.damage" && math.Abs(v.Val-435.839541) > 1e-6 {
			t.Fatalf("round.damage %.9f want 435.839541", v.Val)
		}
	}
	if !found {
		t.Fatal("missing round.damage")
	}
}
