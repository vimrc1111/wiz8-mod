package calc

import (
	"math"
	"testing"
)

// fightKills：一次攻擊的揮擊都打同一隻，打死後該次剩下的揮擊作廢（MG 2026-11-01）。
func TestFightKills(t *testing.T) {
	one := func(n int, q, k float64) []fightAtkOut { return []fightAtkOut{{1, n, q, k}} }
	for _, c := range []struct {
		name   string
		h, dph float64
		atks   [][]fightAtkOut
		want   float64
	}{
		// 一下就死：每次攻擊至多殺 1 隻，3 次攻擊×2 揮＝3 隻，不是 6 隻。
		{"overkill lost", 88, 100, [][]fightAtkOut{one(2, 1, 0), one(2, 1, 0), one(2, 1, 0)}, 3},
		// LS 7 揮第一下就死，其餘 6 下作廢。
		{"LS 7 swings", 88, 100, [][]fightAtkOut{one(7, 1, 0)}, 1},
		// 兩下才死、每次 1 揮：傷害跨攻擊累積，3 次攻擊殺 1 隻。
		{"carry across attacks", 150, 100, [][]fightAtkOut{one(1, 1, 0), one(1, 1, 0), one(1, 1, 0)}, 1},
		// 命中 0.5、1 揮、一下死：每次攻擊 0.5 隻。
		{"miss", 88, 100, [][]fightAtkOut{one(1, 0.5, 0), one(1, 0.5, 0)}, 1},
		// 即死：需 3 下，1 揮、命中 1、即死 0.5：第一次攻擊 0.5 隻。
		{"instakill", 250, 100, [][]fightAtkOut{one(1, 1, 0.5)}, 0.5},
	} {
		if got := fightKills(c.h, c.dph, c.atks); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("%s: fightKills=%v want %v", c.name, got, c.want)
		}
	}
}
