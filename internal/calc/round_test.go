package calc

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestMeleeRoundComposition(t *testing.T) {
	// a=100 gives 3 main attacks; v=100 gives E[swings]=2 per attack.
	// LS L28 adds .12*4=.48 once. Shared gate min(60,40)/100=.4.
	// Two independent 1d2 groups have mean 3; no damage rounding changes it.
	in := MeleeInput{ARhit: 60, ARpen: 40, A: 100, V: 100, Dice: Dice{N: 1, S: 2}, G: 2, SlotSum: 7, Skill: 110, LSLevel: 28}
	r := MeleeRound(in)
	near(t, "main swings", r.ExpectedSwings, 6.48)
	near(t, "main effective", r.ExpectedEffectiveSwings, 2.592)
	near(t, "main damage", r.ExpectedDamage, 7.776)
	// Threshold is 7+4+.4=11.4, conditional and unweighted by the .4 gate.
	near(t, "conditional threshold", r.KillThresholdPerSwingBeforeResistance, 11.4)
	if r.MissRetargetingSupported {
		t.Fatal("C08 must be marked unsupported")
	}
	// Offhand has 2 attacks, each E=2; no LS, even with LSLevel supplied.
	in.OffHand = true
	r = MeleeRound(in)
	near(t, "off swings", r.ExpectedSwings, 4)
	near(t, "off effective", r.ExpectedEffectiveSwings, 1.6)
	near(t, "off damage", r.ExpectedDamage, 4.8)
	// 1 damage with DA60 rounds to zero after the minimum-one stage.
	in.Dice = Dice{Z: 1}
	in.G = 1
	in.DA = 60
	near(t, "absorbed round", MeleeRound(in).ExpectedDamage, 0)
}

func TestBuildStrengthAndDualPenaltyEdges(t *testing.T) {
	x := fighterTables()
	b := Build{Class: Fighter, Level: 1, Attr: [7]int{40, 45, 45, 45, 45, 45, 20}}
	w := Weapon{Skill: SkillSword}
	m, o := Hands(x, b, w, nil)
	if o != nil || m.DmgPct != -10 || m.HitMod != -2 {
		t.Fatalf("low STR/DEX/SEN: main=%+v off=%v", m, o)
	}
	// Low STR penalties apply equally to both hands. DEX -5 truncates to 0.
	m, o = Hands(x, b, w, &w)
	if m.DmgPct != -10 || o.DmgPct != -10 || m.HitMod != -2 || o.HitMod != -2 {
		t.Fatalf("dual low STR: main=%+v off=%+v", m, o)
	}
	// Effective Dual Weapons above 100 makes the signed quotient negative:
	// tz((100-125)/4)=-6, yielding M=-4/-14.
	b.Skill[SkillDualWeapons] = 125
	m, o = Hands(x, b, w, &w)
	if m.M != -4 || o.M != -14 {
		t.Fatalf("dual signed truncation: M=%d/%d", m.M, o.M)
	}
}

func TestBuildHandModesIndependent(t *testing.T) {
	x := fighterTables()
	b := Build{Class: Fighter, Level: 1, Attr: [7]int{50, 50, 50, 50, 50, 50, 50}}
	w := Weapon{Item: Item{Dice: Dice{Z: 10}}, Skill: SkillSword}
	normal := MeleeBuild(x, b, w, &w, 0, 0, Target{Attr4Sum: 200})
	mainBerserk := MeleeBuild(x, b, w, &w, 3, 0, Target{Attr4Sum: 200})
	if mainBerserk.OffHit != normal.OffHit || mainBerserk.OffPen != normal.OffPen || *mainBerserk.OffRound != *normal.OffRound {
		t.Fatal("main mode must not propagate to offhand")
	}
	offBerserk := MeleeBuild(x, b, w, &w, 0, 3, Target{Attr4Sum: 200})
	if offBerserk.MainRound != normal.MainRound || offBerserk.OffHit != normal.OffHit-30 || offBerserk.OffPen != normal.OffPen-30 {
		t.Fatal("offhand mode must only adjust its own hand")
	}
	// Each offhand swing now rolls two fixed-ten groups, gated by its own ratings.
	near(t, "offhand berserk groups", offBerserk.OffRound.ExpectedDamage,
		20*offBerserk.OffRound.ExpectedSwings*GateProb(offBerserk.OffHit, offBerserk.OffPen))
}

func TestAttackValuesComposition(t *testing.T) {
	// A=80+10-5-30+10+10-9-10+2=58.
	// P=80+10+15-5-10-30-9-20-2=29. Easy difficulty: tz(1.4*x).
	in := AttackInput{BARraw: 80, HitMod: 2, CharacterHitMod: -1, PenMod: 3, CharacterPenMod: -2, Mode: 3, Fatigue: 9, Ambush: true, Surprise: true, DefHit: 2, DefPen: 4, Delta: 29, DeltaActual: -29, Difficulty: 0}
	a, p := AttackValues(in)
	if a != 81 || p != 40 {
		t.Fatalf("easy A,P=%d,%d, want 81,40", a, p)
	}
	// Reverse swaps easy to hard; tz(.6*58)=34, tz(.6*29)=17.
	in.ReverseDifficulty = true
	a, p = AttackValues(in)
	if a != 34 || p != 17 {
		t.Fatalf("reverse A,P=%d,%d, want 34,17", a, p)
	}
	// Skill7: fh=floor(9/2)=4, fp=0. A=63, P=38.
	in.ReverseDifficulty = false
	in.Difficulty = 1
	in.WeaponSkill = 7
	a, p = AttackValues(in)
	if a != 63 || p != 38 {
		t.Fatalf("skill7 A,P=%d,%d, want 63,38", a, p)
	}
	// Blind is applied to hit only, before difficulty; no feature -> A=10.
	in.Blind = true
	in.Difficulty = 0
	a, p = AttackValues(in)
	if a != 14 || p != 53 {
		t.Fatalf("blind A,P=%d,%d, want 14,53", a, p)
	}
	// L20 feature7: scale(20,53)=52, so A=10+52=62 -> easy 86.
	in.BlindFeature7 = true
	in.ClassLevel = 20
	a, _ = AttackValues(in)
	if a != 86 {
		t.Fatalf("blind feature7 A=%d, want 86", a)
	}
}

func TestAttackValuesSlowAndSkip(t *testing.T) {
	// SPD=20,B=55,one stack: x=max(0,20+55-25-50)=0.
	// Slow penalty=50. Target would add -10+2; skip omits both => A=5.
	in := AttackInput{BARraw: 55, SPD: 20, SlowStacks: 1, DefHit: 2, Delta: 29, SkipHitTarget: true, Difficulty: 1}
	a, p := AttackValues(in)
	if a != 5 || p != 55 {
		t.Fatalf("slow/skip A,P=%d,%d, want 5,55", a, p)
	}
	// x=16 => x^.75=8; penalty=42, with target correction -8 => A=5.
	in.SPD = 36
	in.SkipHitTarget = false
	a, _ = AttackValues(in)
	if a != 5 {
		t.Fatalf("pow A=%d, want 5", a)
	}
}

func TestSpellCastComposition(t *testing.T) {
	// S0=floor(10/2)+2+tz((tz(6/2)+2)/2)=9.
	// b50: S=9+floor(9*25/100)=11. Mana=6*2=12.
	// v=100 ->150; e=24+3*(4-11)=3, FR=2,3,4 with p=1/4,1/2,1/4.
	// Damage is 147,146,144, respectively, mean 145.75.
	sp := Spell{Dice: Dice{Z: 100}, ManaPerPL: 6, Level: 2, Realm: 2, Type: 3}
	target := Monster{Level: 4, Resist: [6]uint8{0, 0, 24}}
	r := SpellCast(sp, 10, 2, 50, target)
	near(t, "cast damage", r.ExpectedDamage, 145.75)
	near(t, "damage147", r.Damage[147], .25)
	near(t, "damage146", r.Damage[146], .5)
	near(t, "damage144", r.Damage[144], .25)
	if r.Strength != 11 || r.Cost != 12 || r.UnsupportedReason != "" {
		t.Fatalf("cast=%+v", r)
	}
	// Deduct BEFORE PC: S1=9-d; S=S1+floor(S1*.25).
	for _, c := range []struct{ id, want int }{{0x46, 8}, {0x57, 6}, {0x5a, 3}, {0x5d, 3}, {0x5e, 3}, {5, 11}} {
		if got := SpellCastID(sp, c.id, 10, 2, 50, target).Strength; got != c.want {
			t.Errorf("id %d strength=%d, want %d", c.id, got, c.want)
		}
	}
	// Type7 lowers the base by 3, then id d6 takes it to zero, never negative.
	sp.Type = 7
	if got := SpellCastID(sp, 0x5a, 10, 2, 50, target).Strength; got != 0 {
		t.Errorf("clamped strength=%d", got)
	}
	sp.Realm = 6
	if SpellCast(sp, 10, 2, 50, target).UnsupportedReason == "" {
		t.Fatal("invalid realm must be reported")
	}
}

func TestDistributionEnumeration(t *testing.T) {
	// Check convolution probabilities, not just mean: 2d3 totals have counts 1,2,3,2,1.
	d := Groups(Dice{Z: -1, N: 1, S: 3}, 2)
	for v, count := range []int{1, 2, 3, 2, 1} {
		near(t, "2 groups distribution", d[v], float64(count)/9)
	}
	// Independent enumeration checks C14's odd/even, capped and zero-R branches.
	for _, e := range []int{1, 2, 3, 4, 99, 100, 198, 199, 300} {
		m := e + 1
		h := m / 2
		want := make(Dist)
		for a := 0; a < m-h; a++ {
			for b := 0; b < max(1, h); b++ {
				fr := min(100, h+a+b)
				want[37-37*fr/100] += 1 / float64((m-h)*max(1, h))
			}
		}
		got := Resist(Dist{37: 1}, e, 0, 0)
		for v, p := range want {
			near(t, "enumerated resistance", got[v], p)
		}
		var mass float64
		for v, p := range got {
			mass += p
			near(t, "no extra resistance outcome", p, want[v])
		}
		near(t, "resistance mass", mass, 1)
	}
	// Final skill increment also wraps a byte: 254+1+.04 => 255*.96+0*.04.
	near(t, "skill byte wrap", KillThreshold(254, 26), 244.8)
	// The reserved 9999 amount bypasses both PC and resistance.
	near(t, "sentinel", Resist(SpellDamage(Dice{Z: 9999}, 7, 50), 300, 0, 0).Mean(), 9999)
}

func TestLoadItemsMonstersSynthetic(t *testing.T) {
	// Invented records test signed fields and UTF-16 decoding without game data.
	item := make([]byte, 4+0x10d)
	binary.LittleEndian.PutUint32(item, 1)
	r := item[4:]
	copy(r, u16name("測試🔧", 32))
	r[0x48] = 0xfd
	r[0x49] = 0xfe
	copy(r[0x4a:], []byte{0xfc, 0xff, 2, 3})
	binary.LittleEndian.PutUint16(r[0x4e:], 0x123)
	binary.LittleEndian.PutUint16(r[0x76:], 0x456)
	for i := 0; i < 16; i++ {
		r[0x50+i] = byte(i + 1)
	}
	path := filepath.Join(t.TempDir(), "items.dbs")
	if err := os.WriteFile(path, item, 0o600); err != nil {
		t.Fatal(err)
	}
	items, err := LoadItems(path)
	if err != nil {
		t.Fatal(err)
	}
	want := Item{Name: "測試🔧", Dice: Dice{-4, 2, 3}, SpeedMod: -3, HitMod: -2, ModeMask: 0x123, ClassMask: 0x456}
	for i := range want.Effects {
		want.Effects[i] = uint8(i + 1)
	}
	if len(items) != 1 || items[0] != want {
		t.Fatalf("items=%+v", items)
	}
	monster := make([]byte, 4+0x297)
	binary.LittleEndian.PutUint32(monster, 1)
	r = monster[4:]
	copy(r, u16name("測試怪", 24))
	copy(r[0xd6:], []byte{0xfb, 0xff, 3, 4})
	copy(r[0xd1:], []byte{2, 3, 4, 5, 6})
	r[0x15d] = 0xf9
	r[0xcf] = 12
	r[0x24f] = 13
	copy(r[0x176:], []byte{7, 8, 9, 10, 11, 12})
	path = filepath.Join(t.TempDir(), "monsters.dbs")
	if err := os.WriteFile(path, monster, 0o600); err != nil {
		t.Fatal(err)
	}
	monsters, err := LoadMonsters(path)
	if err != nil {
		t.Fatal(err)
	}
	wm := Monster{Name: "測試怪", HP: Dice{-5, 3, 4}, Attr: [5]uint8{2, 3, 4, 5, 6}, AC: -7, Resist: [6]uint8{7, 8, 9, 10, 11, 12}, Level: 13, Absorb: 12}
	if len(monsters) != 1 || monsters[0] != wm {
		t.Fatalf("monsters=%+v", monsters)
	}
}

func TestLoadRecordLengthValidation(t *testing.T) {
	for _, c := range []struct {
		name           string
		header, stride int
		load           func(string) error
	}{
		{"spells", 8, 704, func(p string) error { _, err := LoadSpells(p); return err }},
		{"items", 4, 0x10d, func(p string) error { _, err := LoadItems(p); return err }},
		{"monsters", 4, 0x297, func(p string) error { _, err := LoadMonsters(p); return err }},
	} {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "dbs")
			for _, length := range []int{0, c.header - 1, c.header + c.stride - 1, c.header + c.stride + 1} {
				b := make([]byte, length)
				if len(b) >= 4 {
					binary.LittleEndian.PutUint32(b, 1)
				}
				if err := os.WriteFile(path, b, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := c.load(path); err == nil {
					t.Errorf("length %d accepted", length)
				}
			}
			// Max count must not overflow the size check or cause an allocation panic.
			b := make([]byte, c.header)
			binary.LittleEndian.PutUint32(b, ^uint32(0))
			if err := os.WriteFile(path, b, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := c.load(path); err == nil {
				t.Fatal("overflow count accepted")
			}
			// A zero-record file containing just its header is valid.
			if err := os.WriteFile(path, make([]byte, c.header), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := c.load(path); err != nil {
				t.Fatal(err)
			}
		})
	}
}
