package calc

import "fmt"

// 階段 8.2 骨架（票 R）：型別照此，函式由執行方填實作。

// Skill ids used by the melee hand cache (formulas.md F07, F14; growth.md).
const (
	SkillSword       = 0
	SkillDagger      = 4
	SkillCloseCombat = 16 // H+9 for melee
	SkillDualWeapons = 18
	SkillPowerStrike = 34 // STR 100 advanced skill
)

// Build is a single-class character: Level is both total and class level.
// Attr and Skill are raw values; gear and spell modifiers are not modelled.
type Build struct {
	Class, Level int
	Attr         [7]int  // STR INT PIE VIT DEX SPD SEN
	Skill        [41]int // raw skills, index as C+0x19f+38*s
}

// Weapon is one hand's item and its weapon skill id (0 sword, 4 dagger, ...).
type Weapon struct {
	Item  Item
	Skill int
}

// HandCache mirrors 0x4ee220's per-hand fields (formulas.md F07–F14, F28).
type HandCache struct {
	K, M, A, V, BARraw int // H+0xd, M, F08 a, F09 v, H+0x11
	HitMod, DmgPct     int // H+0x21, H+0x29
}

// Target is the defender side of C04–C06, already resolved (C05 defHit/defPen).
type Target struct {
	DefHit, DefPen int
	Attr4Sum       int // INT+DEX+SPD+SEN (monster M+0x248..0x24b)
	DA             int
}

type BuildResult struct {
	Main, Off        HandCache
	MainHit, MainPen int
	OffHit, OffPen   int
	MainRound        MeleeResult
	OffRound         *MeleeResult // nil without an offhand
	ExpectedDamage   float64      // main + off
}

// PlanAttrs allocates creation and level-up points greedily by prio.
func PlanAttrs(x *ExeTables, race, class, level int, prio []int) ([7]int, error) {
	var a [7]int
	if x == nil || race < 0 || race >= len(x.Race) || class < 0 || class >= len(x.ClassMin) || level < 1 {
		return a, fmt.Errorf("calc: invalid tables, race, class or level")
	}
	var seen [7]bool
	order := make([]int, 0, len(a))
	for _, i := range prio {
		if i < 0 || i >= len(a) {
			return a, fmt.Errorf("calc: attribute priority %d must be 0..6", i)
		}
		if !seen[i] {
			order = append(order, i)
			seen[i] = true
		}
	}
	budget := 60
	for i := range a {
		a[i] = max(x.Race[race][i], x.ClassMin[class][i])
		if a[i] < 0 || a[i] > 100 {
			return a, fmt.Errorf("calc: initial attribute %d outside 0..100", i)
		}
		budget -= max(0, x.ClassMin[class][i]-x.Race[race][i])
		if !seen[i] {
			order = append(order, i)
		}
	}
	if budget < 0 {
		return a, fmt.Errorf("calc: class minimum exceeds creation budget")
	}
	allocate := func(points, cap int) int {
		spent := 0
		for _, i := range order {
			n := min(cap, 100-a[i], points)
			a[i] += n
			points -= n
			spent += n
		}
		return spent
	}
	allocate(budget, min(budget, max(3, (budget+2)/3)))
	for l := 2; l <= level; l++ {
		if allocate(6, 3) == 0 {
			break // All attributes are capped; later levels cannot change them.
		}
	}
	return a, nil
}

// PlanSkills allocates raw skill points greedily by skillPrio at each level.
// Automatic initial skills are [未取得] and start at 0 (notes/re/growth.md:20).
// Unspent points are discarded each level; advanced skills require raw attrs >=100.
func PlanSkills(x *ExeTables, race, class, level int, attrPrio, skillPrio []int) ([41]int, error) {
	var skills [41]int
	var seen [41]bool
	for _, s := range skillPrio {
		if s < 0 || s >= len(skills) {
			return skills, fmt.Errorf("calc: skill priority %d must be 0..40", s)
		}
		if seen[s] {
			return skills, fmt.Errorf("calc: duplicate skill priority %d", s)
		}
		seen[s] = true
	}
	// Validate even an empty skill priority and delegate attribute rules to PlanAttrs.
	attrs, err := PlanAttrs(x, race, class, 1, attrPrio)
	if err != nil {
		return skills, err
	}
	if level < 1 {
		return skills, fmt.Errorf("calc: level must be at least 1")
	}
	for lv := 1; lv <= level; lv++ {
		points, cap := 15, 5
		if lv > 1 {
			attrs, err = PlanAttrs(x, race, class, lv, attrPrio)
			if err != nil {
				return skills, err
			}
			points, cap = 9, 3
		}
		for _, s := range skillPrio {
			if x.Avail[s][class] == 0 || (s >= 34 && attrs[s-34] < 100) {
				continue
			}
			n := min(points, cap, 75-skills[s])
			skills[s] += n
			points -= n
		}
	}
	return skills, nil
}

// EffectiveSkill applies the class primary-skill bonus and the 125 cap.
func EffectiveSkill(x *ExeTables, class, s, raw int) int {
	if s == x.MainSkill[class] {
		raw += max(1, raw/4)
	}
	return min(125, max(0, raw))
}

// Hands rebuilds both hand caches; off is nil for a single weapon.
func Hands(x *ExeTables, b Build, main Weapon, off *Weapon) (HandCache, *HandCache) {
	eff := func(s int) int { return EffectiveSkill(x, b.Class, s, b.Skill[s]) }
	str, dex, spd, sen := b.Attr[0], b.Attr[4], b.Attr[5], b.Attr[6]
	p := x.PMult[b.Class] * b.Level
	cache := func(w Weapon, other *Weapon, offHand bool) HandCache {
		ws := eff(w.Skill)
		u, d := 2*(eff(SkillCloseCombat)+2*ws)/3, 20
		var h HandCache
		if other != nil {
			// F07: C+0x12b5 (dual flag) also adds Dual Weapons and 10 to d (0x4ee44e).
			u += eff(other.Skill)/2 + eff(SkillDualWeapons)
			d = 35
			h.M = -10 - (100-eff(SkillDualWeapons))/4
			if offHand {
				h.M -= 10
			}
		}
		h.K = 10 * u / d
		// Encumbrance E and e are zero: load is outside the build model.
		h.A = ((dex+spd)/2 + h.M + p + h.K) / 3
		h.V = (spd + 10*int(w.Item.SpeedMod) + h.M + p + h.K) / 3
		h.BARraw = 60 + (dex/2+2*h.K+p+h.M)/3
		h.HitMod = int(w.Item.HitMod) + (dex-50)/10
		if str > 50 {
			c := 1
			if offHand {
				c = 2
			}
			h.DmgPct = (2*str - 100) / c
			h.HitMod += (str - 50) / (10 * c)
		} else if str < 50 {
			h.DmgPct = str - 50
			h.HitMod -= (50 - str) / 10
		}
		if sen > 70 {
			h.HitMod += (sen - 70) / 10
		} else if sen < 30 {
			h.HitMod -= (30 - sen) / 10
		}
		if str >= 100 {
			h.HitMod += 1 + eff(SkillPowerStrike)/20
		}
		// F28's special-item 10*Z damage branch is not modelled.
		return h
	}
	m := cache(main, off, false)
	if off == nil {
		return m, nil
	}
	o := cache(*off, &main, true)
	return m, &o
}

// MeleeBuild combines Hands, AttackValues and MeleeRound for one round.
// mode applies to the main hand; the offhand uses offMode (backstab.md: not inherited).
func MeleeBuild(x *ExeTables, b Build, main Weapon, off *Weapon, mode, offMode int, t Target) BuildResult {
	m, o := Hands(x, b, main, off)
	r := BuildResult{Main: m}
	delta := b.Attr[1] + b.Attr[4] + b.Attr[5] + b.Attr[6] - t.Attr4Sum
	round := func(w Weapon, h HandCache, mode int, offHand bool) (int, int, MeleeResult) {
		hit, pen := AttackValues(AttackInput{
			BARraw: h.BARraw, HitMod: h.HitMod, Mode: mode, WeaponSkill: w.Skill,
			SPD: b.Attr[5], Delta: delta, DeltaActual: delta,
			DefHit: t.DefHit, DefPen: t.DefPen, Difficulty: 1,
		})
		g := 1
		if mode == 3 {
			g++
		}
		return hit, pen, MeleeRound(MeleeInput{
			ARhit: hit, ARpen: pen, A: h.A, V: h.V, Dice: w.Item.Dice,
			G: g, BonusPct: h.DmgPct, DA: t.DA, OffHand: offHand,
			SlotSum: int(w.Item.Effects[5]),
		})
	}
	r.MainHit, r.MainPen, r.MainRound = round(main, m, mode, false)
	r.ExpectedDamage = r.MainRound.ExpectedDamage
	if o != nil {
		r.Off = *o
		var result MeleeResult
		r.OffHit, r.OffPen, result = round(*off, *o, offMode, true)
		r.OffRound = &result
		r.ExpectedDamage += result.ExpectedDamage
	}
	return r
}
