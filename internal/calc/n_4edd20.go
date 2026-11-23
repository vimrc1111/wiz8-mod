package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
)

// shared_encumbrance 0x4edd20: split the shared backpack weight over the party
// by remaining capacity, then refresh each present member's total weight and
// load category (notes/re/formulas.md F11; build/wiz8.asm:319150..).
// The three rebuild calls on a category change (0x4ee000 base attack, 0x4ee220
// hands, 0x4ee9d0 armor classes) are separate nodes and are not recomposed
// here: Out holds only the fields this function itself writes.
func init() {
	Nodes["004edd20"] = Node{
		Addr: "004edd20",
		In:   []string{"gameStarted", "combat", "deferred", "otherMode", "interfaceMode", "battle", "itemCount", "refreshMask", "screenFlags", "sharedItem0", "sharedQuantity0", "sharedItem1", "sharedQuantity1", "unitWeight0", "unitWeight1", "pc0.present", "pc0.status", "pc0.personalWeight", "pc0.capacity", "pc0.sharedWeight", "pc0.totalWeight", "pc0.loadCategory", "pc1.present", "pc1.status", "pc1.personalWeight", "pc1.capacity", "pc1.sharedWeight", "pc1.totalWeight", "pc1.loadCategory", "pc2.present", "pc2.status", "pc2.personalWeight", "pc2.capacity", "pc2.sharedWeight", "pc2.totalWeight", "pc2.loadCategory", "pc3.present", "pc3.status", "pc3.personalWeight", "pc3.capacity", "pc3.sharedWeight", "pc3.totalWeight", "pc3.loadCategory", "pc4.present", "pc4.status", "pc4.personalWeight", "pc4.capacity", "pc4.sharedWeight", "pc4.totalWeight", "pc4.loadCategory", "pc5.present", "pc5.status", "pc5.personalWeight", "pc5.capacity", "pc5.sharedWeight", "pc5.totalWeight", "pc5.loadCategory", "pc6.present", "pc6.status", "pc6.personalWeight", "pc6.capacity", "pc6.sharedWeight", "pc6.totalWeight", "pc6.loadCategory", "pc7.present", "pc7.status", "pc7.personalWeight", "pc7.capacity", "pc7.sharedWeight", "pc7.totalWeight", "pc7.loadCategory"},
		Out:  []string{"pc0.sharedWeight", "pc0.totalWeight", "pc0.loadCategory", "pc1.sharedWeight", "pc1.totalWeight", "pc1.loadCategory", "pc2.sharedWeight", "pc2.totalWeight", "pc2.loadCategory", "pc3.sharedWeight", "pc3.totalWeight", "pc3.loadCategory", "pc4.sharedWeight", "pc4.totalWeight", "pc4.loadCategory", "pc5.sharedWeight", "pc5.totalWeight", "pc5.loadCategory", "pc6.sharedWeight", "pc6.totalWeight", "pc6.loadCategory", "pc7.sharedWeight", "pc7.totalWeight", "pc7.loadCategory", "deferred", "screenFlags", "refreshMask"},
		F:    sharedEncumbranceNode,
	}
}

const (
	encPartySize   = 8    // loop bound 8 slots (0x4ede50)
	encStatusLimit = 0x12 // member takes part only while status < 0x12 (0x4ede2b)
	encGlobalIn    = 15   // inputs before the per-member groups
	encMemberIn    = 7    // present status personal capacity shared total category
	encSlotsModel  = 2    // shared slots provided by the case (0x685191, 0x68519d)
)

var encFloats, encFloatsErr = loadEncFloats()

// Percent multiplier (float32 at 0x5ebcd0) and scan baseline (float32 at
// 0x5ed510) are read from the player's own exe, never embedded.
func loadEncFloats() (f [2]float32, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return f, err
	}
	for i, va := range []uint32{0x5ebcd0, 0x5ed510} {
		o, err := peOffset(b, va)
		if err != nil {
			return f, err
		}
		if len(b)-o < 4 {
			return f, fmt.Errorf("shared encumbrance: truncated float at %#x", va)
		}
		f[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))
	}
	return f, nil
}

// encShare is (rem*K)/cap evaluated in x87 extended precision (64-bit
// mantissa; the emulator keeps control word 0x037f) and stored as float32.
// A zero divisor gives +Inf, -Inf or NaN by the numerator's sign (masked FPU).
func encShare(rem, cap int32) float64 {
	x := new(big.Float).SetPrec(64).SetInt64(int64(rem))
	x.Mul(x, new(big.Float).SetPrec(64).SetFloat64(float64(encFloats[0])))
	if cap == 0 {
		switch x.Sign() {
		case 1:
			return math.Inf(1)
		case -1:
			return math.Inf(-1)
		}
		return math.NaN()
	}
	x.Quo(x, new(big.Float).SetPrec(64).SetInt64(int64(cap)))
	f, _ := x.Float32()
	return float64(f)
}

func sharedEncumbranceNode(in []int64, r Rand) []int64 {
	if encFloatsErr != nil {
		panic(encFloatsErr)
	}
	if len(in) != encGlobalIn+encPartySize*encMemberIn {
		panic("shared encumbrance: wrong input count")
	}
	in = append([]int64(nil), in...)
	out := make([]int64, encPartySize*3+3)
	m := func(k, f int) *int64 { return &in[encGlobalIn+k*encMemberIn+f] }
	const (
		present, status, personal, capacity, shared, total, category = 0, 1, 2, 3, 4, 5, 6
	)
	write := func() { // out = current shared/total/category per member, then flags
		for k := 0; k < encPartySize; k++ {
			out[3*k], out[3*k+1], out[3*k+2] = *m(k, shared), *m(k, total), *m(k, category)
		}
		out[24], out[25], out[26] = in[2], in[8], in[7]
	}
	defer write()
	if in[0] == 0 { // game not started: nothing written (0x4edd20 prologue)
		return out
	}
	if in[1] != 0 { // combat: defer the refresh
		in[2] = 1
		return out
	}
	var rem, cap [encPartySize]int32
	var share [encPartySize]float64
	eligible := func(k int) bool { return *m(k, present) != 0 && uint32(*m(k, status)) < encStatusLimit }
	for k := 0; k < encPartySize; k++ {
		*m(k, shared) = 0
		if eligible(k) {
			cap[k] = int32(uint32(*m(k, capacity)))
			rem[k] = cap[k] - int32(uint32(*m(k, personal)))
			share[k] = encShare(rem[k], cap[k])
		}
	}
	// Shared pool: sum of stack weights (0x51bfd0), halved.
	if in[6] < 0 || in[6] > encSlotsModel {
		panic("shared encumbrance: itemCount outside the two modeled slots")
	}
	var pool uint32
	for s := 0; s < int(in[6]); s++ {
		id, qty := in[9+2*s], in[10+2*s]
		if id == -1 {
			continue
		}
		if id < 0 || id > 1 {
			panic("shared encumbrance: item id outside the two modeled definitions")
		}
		if qty == 0 {
			qty = 1
		}
		pool += uint32(qty) * uint32(in[13+id])
	}
	for n := pool >> 1; n != 0; n-- {
		best, pick := float64(encFloats[1]), -1
		for k := 0; k < encPartySize; k++ {
			// fcom + test C0: taken when best < share or the pair is unordered (NaN).
			if eligible(k) && !(best >= share[k]) {
				best, pick = share[k], k
			}
		}
		if pick < 0 {
			return out // nobody can take a unit: loop two and the tail are skipped
		}
		*m(pick, shared)++
		rem[pick]--
		share[pick] = encShare(rem[pick], cap[pick])
	}
	for k := 0; k < encPartySize; k++ {
		if *m(k, present) == 0 {
			continue
		}
		t := uint32(*m(k, personal)) + uint32(*m(k, shared))
		*m(k, total) = int64(t)
		if uint32(*m(k, capacity)) == 0 {
			panic("shared encumbrance: zero capacity faults in the original (integer divide)")
		}
		q := t * 100 / uint32(*m(k, capacity))
		switch {
		case q < 50:
			*m(k, category) = 0
		case q < 70:
			*m(k, category) = 1
		case q < 85:
			*m(k, category) = 2
		case q <= 100:
			*m(k, category) = 3
		default:
			*m(k, category) = 4
		}
	}
	switch in[4] {
	case 7:
		if in[3] == 0 {
			if in[5] != 0 { // 0x562a50(0xff): only with a live battle object
				in[7] |= 0xff
			}
		}
	case 6:
		in[8] |= 0x2000 | 0x100
	}
	in[2] = 0
	return out
}
