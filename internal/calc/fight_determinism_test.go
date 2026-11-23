package calc

import (
	"bytes"
	"testing"
)

// fightExpect sums leaf probabilities from parallel subtrees; the result must
// not depend on goroutine completion order (rank ties and dedup rely on it).
func TestFightDeterministic(t *testing.T) {
	w, err := LoadFightWorld("../../build/Wiz8.exe", "../../build/data-x/DATABASES")
	if err != nil {
		t.Skip("沒有 build/Wiz8.exe 或 DATABASES：", err)
	}
	train := map[int]int{}
	for s := 0; s <= 40; s++ {
		train[s] = 100
	}
	tmpl := RankTemplate{Races: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, Classes: []int{7}, Attrs: []int{0}, Levels: []int{28}, SkillPrio: []int{}, Train: train, AttrMin: 100, Haste: true, Loadouts: []RankLoadout{{Name: "ls", Main: &FightWeapon{Item: 62}}}, Monster: 564, Difficulty: 1}
	var first []byte
	for i := 0; i < 6; i++ {
		var b bytes.Buffer
		if err := WriteRankWorkers(&b, w, tmpl, 0, 4); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = b.Bytes()
		} else if !bytes.Equal(first, b.Bytes()) {
			t.Fatalf("run %d differs from run 0:\n%s\n%s", i, first, b.Bytes())
		}
	}
}
