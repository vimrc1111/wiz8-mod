package calc

import (
	"os"
	"testing"
)

// fightTestWorld loads the player's own exe and databases (build/ in the repo), or skips.
func fightTestWorld(t *testing.T) *FightWorld {
	t.Helper()
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build/Wiz8.exe 或 DATABASES：", err)
	}
	return w
}

// notes/calc-fight-inputs.tsv is generated from the registry over the Q4 single-weapon
// scenario and a dual-wield scenario; UPDATE=1 rewrites it.
func TestFightInputsTSV(t *testing.T) {
	w := fightTestWorld(t)
	q4 := FightScenario{Race: 0, Class: 4, Level: 28, Prio: []int{0, 4, 5, 3}, Skills: map[int]int{0: 100, 16: 100, 19: 100, 34: 58},
		Main: &FightWeapon{Item: 41}, Monster: 411, Difficulty: 1}
	dual := q4
	dual.Skills = map[int]int{0: 100, 16: 100, 18: 100, 19: 100, 34: 58}
	dual.Off = &FightWeapon{Item: 41}
	got, err := w.FightInputsTSV([]FightScenario{q4, dual})
	if err != nil {
		t.Fatal(err)
	}
	const path = "../../notes/calc-fight-inputs.tsv"
	if os.Getenv("UPDATE") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Fatal("notes/calc-fight-inputs.tsv 與登記不符；UPDATE=1 go test -run TestFightInputsTSV ./internal/calc 重產")
	}
}

// DiceTotal (one branch over the convolution) must give the same distribution as
// 5420b0's per-die path through diceNode.
func TestFightDiceTotal(t *testing.T) {
	for _, c := range []struct{ count, z, n, s int64 }{{1, 0, 2, 6}, {2, 3, 1, 8}, {3, -2, 2, 4}, {1, 5, 0, 9}} {
		a, err := fightExpect(func(r Rand) []float64 {
			v := r.(*fightRand).DiceTotal(c.count, c.z, c.n, c.s)
			return []float64{float64(v), float64(v * v)}
		})
		if err != nil {
			t.Fatal(err)
		}
		b, err := fightExpect(func(r Rand) []float64 {
			var v int64
			for k := int64(0); k < c.count; k++ {
				v += diceNode([]int64{c.z, c.n, c.s}, r)[0]
			}
			return []float64{float64(v), float64(v * v)}
		})
		if err != nil {
			t.Fatal(err)
		}
		for i := range a {
			if d := a[i] - b[i]; d > 1e-9 || d < -1e-9 {
				t.Fatalf("%+v moment %d: DiceTotal %v, per-die %v", c, i, a[i], b[i])
			}
		}
	}
}
