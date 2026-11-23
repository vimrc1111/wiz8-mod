package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

func init() {
	Nodes["00500330"] = Node{Addr: "00500330",
		In:  []string{"monsterBonus", "templateBonus", "templateId", "spellId", "spellCostPerLevel", "spellRealmLevel"},
		Out: []string{"powerLevel"}, F: monsterSpellPowerLevelNode}
}

var monsterSpellPLExe struct {
	once sync.Once
	b    []byte
	err  error
}

// monsterSpellPLRisk reads the u32 at VA 0x616df0+4*idx of the player's own
// Wiz8.exe (the per-level threshold table, idx 0..16 after the clamp at
// 0x5003aa). Negative idx reads the bytes before the table, as the original does.
func monsterSpellPLRisk(idx int32) uint32 {
	monsterSpellPLExe.once.Do(func() {
		path := os.Getenv("WIZ8_EXE")
		if path == "" {
			_, source, _, ok := runtime.Caller(0)
			if !ok {
				monsterSpellPLExe.err = fmt.Errorf("monster spell power level: set WIZ8_EXE")
				return
			}
			path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
		}
		monsterSpellPLExe.b, monsterSpellPLExe.err = os.ReadFile(path)
	})
	if monsterSpellPLExe.err != nil {
		panic(monsterSpellPLExe.err)
	}
	off, err := peOffset(monsterSpellPLExe.b, 0x616df0+uint32(idx)*4)
	if err != nil || off+4 > len(monsterSpellPLExe.b) {
		panic("monster spell power level: threshold table address unmapped")
	}
	return binary.LittleEndian.Uint32(monsterSpellPLExe.b[off:])
}

// Monster spell power level 1..7 (0x500330..0x50045d). Inputs: monster field
// +0x2f9, the monster template's +0x26f (both i32, summed with u32 wrap and
// floored at 0), the spell id (0x77 returns 0), and the spell record's +0x49 and
// +0x56 (i32). templateId only selects which template record is read in the
// replay fixture. The 0x4e5150/0x5182e0 calls at 0x500372 (template +0x26f == 0)
// only format an error message and do not change the result; the loop recomputes
// the same sum every pass, so it is computed once. The loop (0x500357..0x500401)
// raises the base level edi from 1 while the risk estimate stays at or below 8;
// then edi-1 (if >1) is nudged by R(4): 0 lowers (if >2), 1 raises (if <7).
func monsterSpellPowerLevelNode(in []int64, r Rand) []int64 {
	if len(in) != 6 {
		panic("monster spell power level: expected six inputs")
	}
	for i, v := range in {
		lo, hi := int64(-1<<31), int64(1<<31-1)
		if i == 2 || i == 3 {
			lo, hi = 0, 1<<32-1
		}
		if v < lo || v > hi {
			panic("monster spell power level: input out of 32-bit range")
		}
	}
	if uint32(in[3]) == 0x77 {
		return []int64{0}
	}
	sum := int32(uint32(in[0]) + uint32(in[1]))
	if sum < 0 {
		sum = 0
	}
	esi := uint32(sum)
	idx := int32(in[4])/2 + int32(in[5])
	if idx > 0x10 {
		idx = 0x10
	}
	table := monsterSpellPLRisk(idx)
	edi := uint32(1)
	for {
		thr := table * edi / 7
		if esi < thr {
			q := (thr*70 - esi*70) / thr
			if int32(q) >= 0 && q > 8 {
				break
			}
		}
		edi++
		if edi >= 8 {
			break
		}
	}
	if edi > 1 {
		edi--
	}
	pl := edi
	switch r.R(4) {
	case 0:
		if pl > 2 {
			pl--
		}
	case 1:
		if pl < 7 {
			pl++
		}
	}
	return []int64{int64(pl)}
}
