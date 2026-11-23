package calc

import "math"

// AttackInput contains the resolved C04/C06 fields. Fatigue is f (not raw
// stamina), Delta compares the original target, and DeltaActual compares the
// actual penetration target. DefHit/DefPen already include C05's modifiers.
// Difficulty is 0=easy, 1=normal, 2=hard; other values leave ratings unchanged.
type AttackInput struct {
	BARraw, HitMod, CharacterHitMod    int // B, J, b1
	PenMod, CharacterPenMod            int // H+0x25, C+0x1772
	Mode, WeaponSkill, Fatigue         int
	SPD, SlowStacks                    int
	Ambush, Surprise                   bool // G+0x9a4, G+0x9a5
	DefHit, DefPen, Delta, DeltaActual int
	SkipHitTarget                      bool // fourth argument low byte != 0
	Blind, BlindFeature7               bool
	ClassLevel, Difficulty             int
	ReverseDifficulty                  bool // C+0xa35 > 0
}

// AttackValues combines C04/C06, preserving all truncation points and applying
// difficulty after the blind adjustment. It handles melee modes 0..8.
func AttackValues(in AttackInput) (A, P int) {
	hitMode := [9]int{0, -10, 0, -30, -5, 5, -10, 10, 0}
	penMode := [9]int{0, 10, -5, -30, 5, -5, 0, -10, 10}
	if in.Mode < 0 || in.Mode >= len(hitMode) {
		panic("calc: melee mode must be 0..8")
	}
	fh, fp := in.Fatigue, in.Fatigue
	if in.WeaponSkill == 7 {
		fh = int(math.Floor(float64(fh) / 2))
		fp = 0
	}
	A = in.BARraw + 5*in.HitMod + 5*in.CharacterHitMod + hitMode[in.Mode] - fh
	if in.Ambush {
		A += 10
	}
	if in.Surprise {
		A += 10
	}
	if in.SlowStacks > 0 {
		x := max(0, in.SPD+in.BARraw-25*in.SlowStacks-50)
		A -= max(0, int(50-math.Pow(float64(x), 0.75)))
	}
	if !in.SkipHitTarget {
		A += -5*in.DefHit + in.Delta/10
	}
	if in.Blind && A > 10 {
		if in.BlindFeature7 {
			A = 10 + Scale(in.ClassLevel, A-10)
		} else {
			A = 10
		}
	}
	P = in.BARraw + 5*in.HitMod + 5*in.PenMod + 5*in.CharacterHitMod + 5*in.CharacterPenMod + penMode[in.Mode] - fp - 5*in.DefPen + in.DeltaActual/10
	difficulty := in.Difficulty
	if in.ReverseDifficulty && (difficulty == 0 || difficulty == 2) {
		difficulty = 2 - difficulty
	}
	adjust := func(x int) int {
		switch difficulty {
		case 0:
			return 140 * x / 100
		case 2:
			return 60 * x / 100
		default:
			return x
		}
	}
	return adjust(A), adjust(P)
}

// MeleeInput describes one enabled hand; add two MeleeRound results for dual
// wielding, supplying each hand's own ratings, a/v, dice and bonuses. OffHand
// selects F08's offhand budget and disables LS. G is the final group count,
// including mode 3 and any externally resolved F21/F22 bonuses.
type MeleeInput struct {
	ARhit, ARpen    int
	A, V            int // F08 a, F09 v; these are already combined ratings.
	Dice            Dice
	G, BonusPct, DA int
	SlotSum, Skill  int // skill 0 when the skill-based Kill path is ineligible.
	LSLevel         int // current Samurai class level; 0 disables LS.
	OffHand         bool
}
type MeleeResult struct {
	ExpectedSwings                        float64
	ExpectedEffectiveSwings               float64 // swings passing the shared hit/penetration gate.
	ExpectedDamage                        float64
	KillThresholdPerSwingBeforeResistance float64 // byte threshold, not percent probability of death.
	MissRetargetingSupported              bool    // always false: C08 is outside this model.
}

// MeleeRound assumes the full attack budget runs without death or interruption.
// The Kill threshold is conditional on reaching the effect stage; it is not
// weighted by hit or damage and is never interpreted as final Kill success.
func MeleeRound(in MeleeInput) MeleeResult {
	main, off := Attacks(in.A)
	attacks := main
	if in.OffHand {
		attacks = off
	}
	swings := float64(attacks) * float64(SwingCap(in.V)+1) / 2
	if !in.OffHand && in.LSLevel > 0 {
		swings += LSExtraSwings(in.LSLevel)
	}
	effective := swings * GateProb(in.ARhit, in.ARpen)
	return MeleeResult{
		ExpectedSwings: swings, ExpectedEffectiveSwings: effective,
		ExpectedDamage:                        effective * PhysDamage(Groups(in.Dice, in.G), in.BonusPct, in.DA).Mean(),
		KillThresholdPerSwingBeforeResistance: KillThreshold(in.SlotSum, in.Skill),
	}
}

type SpellResult struct {
	Damage            Dist
	ExpectedDamage    float64
	Cost, Strength    int
	UnsupportedReason string
}

// SpellCast models one target using S0 without id-specific C12 deductions.
// Spell has no id field: use SpellCastID when the database index is known.
// L is the effective caster level, not necessarily total character level.
func SpellCast(sp Spell, L, PL, b int, target Monster) SpellResult {
	return SpellCastID(sp, -1, L, PL, b, target)
}

// SpellCastID additionally applies C12's id-specific strength deductions.
// Target resistances are template values; apply runtime modifiers before calling.
func SpellCastID(sp Spell, id, L, PL, b int, target Monster) SpellResult {
	s := SpellStrength(L, PL, int(sp.ManaPerPL), int(sp.Level), sp.Type == 7)
	deduction := 0
	switch id {
	case 0x46:
		deduction = 2
	case 0x57:
		deduction = 4
	case 0x5a, 0x5d, 0x5e:
		deduction = 6
	}
	s = max(0, s-deduction)
	s += int(math.Floor(float64(s) * math.Floor(float64(b)/2) / 100))
	out := SpellResult{Cost: int(sp.ManaPerPL) * PL, Strength: s}
	if sp.Realm >= uint32(len(target.Resist)) {
		out.UnsupportedReason = "spell realm must be 0..5"
		return out
	}
	out.Damage = Resist(SpellDamage(sp.Dice, PL, b), int(target.Resist[sp.Realm]), int(target.Level), s)
	out.ExpectedDamage = out.Damage.Mean()
	return out
}
