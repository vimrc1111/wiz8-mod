package calc

import "fmt"

// encounter_spawn_order 0x48ca20 (build/wiz8.asm:188845..): dissolve every active
// monster group (group_dissolve 0x510930 on the last active-list entry while the
// active count 0x65ba14 is positive), copy the scene's generator list
// (*0x659ab4 +0xc4; count +4, entries +0xc), shuffle the copy with `count`
// rounds of two R(count) draws swapping the two entries when they differ (only
// when count > 1), then for each generator in shuffled order run the spawn check
// 0x48b200(gen, 1) and, when it passes, encounter_generate 0x48ad20(gen, gen+0xc).
//
// Spawn check with argument 1 (0x48b200, run natively in the emulator): no
// spawn if any block byte 0x65ba48, 0x6840bc, 0x683f94, 0x683f97, 0x68f105 is
// set or the generator's enabled byte +0x44 is clear; if active count >= budget
// (i32 0x60a6c0) call group_cull_by_distance 0x48c8e0 and re-check, no spawn if
// still >=. The distance it computes is unused with argument 1.
//
// Emu address-stubs 0x510930, 0x48c8e0 (both excluded, b90-helpers.tsv) and the
// D4 callee 0x48ad20 (composition ruling 2026-10-09): the active count after each
// call comes from the case's per-call inputs. F instead decrements on dissolve
// and adds the active-count growth of the real encounterGenerateNode (run with
// active count 0 on the template inputs eg_*, the generator's own flags and
// table index); the cull result stays an input (excluded callee).
//
// Fixture caps: active count and generator count 0..8, panic outside.
// An R(count) result outside 0..count-1 panics: the index-clamp blocks it would
// reach are cov-exempt.
const (
	esoCap  = 8
	esoGen  = 0x20000000 // case object gen0; generator k at +0x100*k
	esoAct  = 0x47525000 // case decoy pointer stored at active-list entry k (+k)
	esoHeap = 0x50000000 // emu bump heap: the shuffled copy from 48cfd0's operator new

	esoIAct, esoIDis, esoIBudget, esoIBlock, esoIGenN = 0, 1, 9, 10, 15
	esoIEn, esoIGF, esoICAG, esoICAC, esoITpl         = 16, 24, 40, 48, 56

	esoODisN, esoODis, esoOOrder, esoOCull, esoOGenN, esoOGen, esoOFinal = 0, 1, 9, 17, 18, 19, 27
)

var esoTplSkip = map[string]bool{"genFlags": true, "tableIdx": true, "activeCount": true}

var esoIn = func() (in []string) {
	in = append(in, "activeCount")
	for k := 0; k < esoCap; k++ {
		in = append(in, fmt.Sprint("dissolveNext", k))
	}
	in = append(in, "spawnBudget", "blockA", "blockB", "blockC", "blockD", "blockE", "generatorCount")
	for k := 0; k < esoCap; k++ {
		in = append(in, fmt.Sprint("enabled", k))
	}
	for k := 0; k < esoCap; k++ {
		in = append(in, fmt.Sprint("genFlags", k), fmt.Sprint("tableIdx", k))
	}
	for k := 0; k < esoCap; k++ {
		in = append(in, fmt.Sprint("countAfterGenerate", k))
	}
	for k := 0; k < esoCap; k++ {
		in = append(in, fmt.Sprint("countAfterCull", k))
	}
	for _, n := range egIn {
		if !esoTplSkip[n] {
			in = append(in, "eg_"+n)
		}
	}
	return
}()

var esoOut = func() (out []string) {
	out = append(out, "dissolveCalls")
	for k := 0; k < esoCap; k++ {
		out = append(out, fmt.Sprint("dissolveArg", k))
	}
	for k := 0; k < esoCap; k++ {
		out = append(out, fmt.Sprint("order", k))
	}
	out = append(out, "cullCalls", "generateCalls")
	for k := 0; k < esoCap; k++ {
		out = append(out, fmt.Sprint("generateArg", k))
	}
	return append(out, "activeCountFinal")
}()

func init() {
	Nodes["0048ca20"] = Node{Addr: "0048ca20", In: esoIn, Out: esoOut, F: encounterSpawnOrderNode,
		Emu: func(in []int64, r Rand) []int64 { return esoNode(in, r, false) }}
}

func encounterSpawnOrderNode(in []int64, r Rand) []int64 { return esoNode(in, r, true) }

func esoNode(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(esoIn) {
		panic("encounter_spawn_order: wrong input count")
	}
	out := make([]int64, len(esoOut))
	c := in[esoIAct]
	nd := 0
	for c > 0 {
		if c > esoCap || nd == esoCap {
			panic("encounter_spawn_order: active count beyond fixture cap")
		}
		out[esoODis+nd] = esoAct + c - 1
		if real {
			c--
		} else {
			c = in[esoIDis+nd]
		}
		nd++
	}
	out[esoODisN] = int64(nd)

	n := int(in[esoIGenN])
	if n < 0 || n > esoCap {
		panic("encounter_spawn_order: generator count beyond fixture cap")
	}
	order := make([]int, n)
	for k := range order {
		order[k] = k
	}
	if n > 1 {
		for round := 0; round < n; round++ {
			i, j := r.R(n), r.R(n)
			if i < 0 || i >= n || j < 0 || j >= n {
				panic("encounter_spawn_order: R(n) out of range")
			}
			order[i], order[j] = order[j], order[i]
		}
	}
	for k, g := range order {
		out[esoOOrder+k] = esoGen + 0x100*int64(g)
	}

	blocked := false
	for b := 0; b < 5; b++ {
		blocked = blocked || in[esoIBlock+b]&0xff != 0
	}
	budget, nc, ng := in[esoIBudget], 0, 0
	for _, g := range order {
		if blocked || in[esoIEn+g]&0xff == 0 {
			continue
		}
		if c >= budget {
			c = in[esoICAC+nc]
			nc++
			if c >= budget {
				continue
			}
		}
		out[esoOGen+ng] = esoGen + 0x100*int64(g) + 0xc
		if real {
			c += esoGenerate(in, g, r)
		} else {
			c = in[esoICAG+ng]
		}
		ng++
	}
	out[esoOCull], out[esoOGenN], out[esoOFinal] = int64(nc), int64(ng), c
	return out
}

// esoGenerate runs the real encounterGenerateNode for generator g on the template
// inputs with active count 0 and returns how many groups it added.
func esoGenerate(in []int64, g int, r Rand) int64 {
	v := make([]int64, len(egIn))
	t := esoITpl
	for i, n := range egIn {
		switch n {
		case "genFlags":
			v[i] = in[esoIGF+2*g]
		case "tableIdx":
			v[i] = in[esoIGF+2*g+1]
		case "activeCount":
			v[i] = 0
		default:
			v[i] = in[t]
			t++
		}
	}
	o := encounterGenerateNode(v, r)
	for i, n := range egOutNames {
		if n == "activeCountAfter" {
			return o[i]
		}
	}
	panic("encounter_spawn_order: encounter_generate has no activeCountAfter")
}
