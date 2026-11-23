// Package calc computes melee and direct spell damage using uniform R(n).
// It does not model ranged attacks, AoE target lists, casting success, Kill
// resistance, status effects, interruptions, or the exact F01 RNG mapping.
package calc

import (
	"math"
	"sort"
)

type Dist map[int]float64

func (d Dist) Mean() float64 {
	keys := make([]int, 0, len(d))
	for v := range d {
		keys = append(keys, v)
	}
	sort.Ints(keys)
	var mean float64
	for _, v := range keys {
		mean += float64(v) * d[v]
	}
	return mean
}

type Dice struct {
	Z    int16
	N, S uint8
}
type Spell struct {
	Name             string
	Dice             Dice
	ManaPerPL, Level int32
	Realm            uint32
	Range, Type      int32
}
type Item struct {
	Name                string
	Dice                Dice
	Effects             [16]uint8
	SpeedMod, HitMod    int8
	ModeMask, ClassMask uint16
}
type Monster struct {
	Name   string
	HP     Dice
	Attr   [5]uint8
	AC     int8
	Resist [6]uint8
	Level  uint8
	Absorb uint8
}

// Scale retains C02's binary32 0.01 constant, with no intermediate f32 write.
func Scale(L, m int) int {
	if uint32(L) > 20 {
		return m
	}
	return int(float64(2*L+60) * float64(float32(m)) * 5368709 / 536870912)
}

// diceSum convolves uniform dice with a sliding window. Zero sides contribute 1.
func diceSum(z, n, s int) Dist {
	if n <= 0 {
		return Dist{z: 1}
	}
	if s <= 1 {
		return Dist{z + n: 1}
	}
	p := []float64{1}
	for i := 0; i < n; i++ {
		next := make([]float64, len(p)+s-1)
		var window float64
		for j := range next {
			if j < len(p) {
				window += p[j]
			}
			if j >= s && j-s < len(p) {
				window -= p[j-s]
			}
			next[j] = math.Max(0, window/float64(s))
		}
		p = next
	}
	out := make(Dist, len(p))
	for j, prob := range p {
		if prob > 0 {
			out[z+n+j] = prob
		}
	}
	return out
}
func DiceDist(d Dice) Dist { return diceSum(int(d.Z), int(d.N), int(d.S)) }

// Groups includes the fixed base once per independent group; g already includes mode 3.
func Groups(d Dice, g int) Dist {
	if g <= 0 {
		return Dist{0: 1}
	}
	return diceSum(int(d.Z)*g, int(d.N)*g, int(d.S))
}
func HitProb(A int) float64     { return float64(max(5, min(95, A))) / 100 }
func GateProb(A, P int) float64 { return HitProb(min(A, P)) }
func transform(d Dist, f func(int) int) Dist {
	out := make(Dist)
	for v, p := range d {
		out[f(v)] += p
	}
	return out
}
func PhysDamage(d Dist, bonusPct, DA int) Dist {
	return transform(d, func(v int) int {
		d1 := max(1, (v*(100+bonusPct)+50)/100)
		if DA == 0 {
			return d1
		}
		return max(0, (d1*(100-DA)+50)/100)
	})
}

// KillThreshold is a byte threshold expectation, not a final Kill probability.
// Slot composition and the subsequent skill addition each wrap at 256 (F24).
func KillThreshold(slotSum, skill int) float64 {
	base := int(uint8(slotSum)) + skill/25
	remainder := skill % 25
	return float64(uint8(base))*float64(25-remainder)/25 + float64(uint8(base+1))*float64(remainder)/25
}
func Attacks(a int) (main, off int) {
	main, off = 1, 1
	if a >= 50 {
		main++
	}
	if a >= 100 {
		main++
	}
	if a >= 75 {
		off++
	}
	return
}
func SwingCap(v int) int {
	s := 1
	if v >= 67 {
		s++
	}
	if v >= 100 {
		s++
	}
	return s
}
func LSExtraSwings(L int) float64 {
	extra := 3.0
	if L >= 20 {
		extra = 4
	} else if L >= 10 {
		extra += float64(L-9) / 11
	}
	return float64(Scale(L, 12)) / 100 * extra
}
func SpellStrength(L, PL, manaPerPL, level int, type7 bool) int {
	s := int(math.Floor(float64(L)/2)) + PL + (manaPerPL/2+level)/2
	if type7 {
		s -= 3
	}
	return max(0, s)
}
func PowerCastB(skill int, on bool) int {
	if !on {
		return 0
	}
	return 1 + int(uint32(skill)/4)
}
func SpellDamage(d Dice, PL, b int) Dist {
	d.N = uint8(int(d.N) * PL)
	return transform(DiceDist(d), func(v int) int {
		if v == 9999 {
			return v
		}
		return v + int(math.Floor(float64(v)*float64(b)/100))
	})
}

// Resist enumerates the exact capped triangular FR distribution (C14).
func Resist(d Dist, Rj, Lt, S int) Dist {
	e := Rj + 3*(Lt-S)
	if e <= 0 {
		return transform(d, func(v int) int { return v })
	}
	m := e + 1
	h := m / 2
	fr := make(Dist)
	if h >= 100 {
		fr[100] = 1
	} else {
		a, c := m-h, max(1, h) // R(0)=0, represented as one outcome.
		var below float64
		for f := h; f < 100; f++ {
			sum := f - h
			count := max(0, min(a-1, sum)-max(0, sum-(c-1))+1)
			p := float64(count) / float64(a) / float64(c)
			if p > 0 {
				fr[f] = p
				below += p
			}
		}
		if below < 1 {
			fr[100] = 1 - below
		}
	}
	out := make(Dist)
	for v, p := range d {
		if v == 9999 {
			out[v] += p
			continue
		}
		for f, q := range fr {
			out[v-int(math.Floor(float64(v)*float64(f)/100))] += p * q
		}
	}
	return out
}
