// 票 Q 驗收（cc 寫，執行方不改）：期望值依 notes/re/formulas.md、formulas-2.md、combat-2.md、abilities.md 手算。
// 機率一律是均勻近似（R(n) 視為 0..n-1 等機率），不是 F01 的精確映射。
package calc

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

func near(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// combat-2.md C02：f32 常數 0.0099999998，L=20 是 11 不是 12。
func TestScale(t *testing.T) {
	for _, c := range []struct{ L, m, want int }{
		{1, 12, 7}, {10, 12, 9}, {19, 12, 11}, {20, 12, 11}, {21, 12, 12}, {28, 12, 12},
		{20, 100, 99}, {21, 100, 100}, {1, 5, 3}, {20, 5, 4}, {21, 5, 5},
	} {
		if got := Scale(c.L, c.m); got != c.want {
			t.Errorf("Scale(%d,%d) = %d, want %d", c.L, c.m, got, c.want)
		}
	}
}

// formulas.md F20：sides=0 每顆貢獻 1；base 每組都加。formulas-2.md：N*PL 截成 byte。
func TestDice(t *testing.T) {
	near(t, "3+2d0", DiceDist(Dice{3, 2, 0}).Mean(), 5)
	near(t, "1d7", DiceDist(Dice{0, 1, 7}).Mean(), 4)
	near(t, "2 groups of 5+0d0", Groups(Dice{5, 0, 0}, 2).Mean(), 10)
	near(t, "2 groups of 1d2", Groups(Dice{0, 1, 2}, 2)[4], 0.25)
	near(t, "spell 40d1 PL7 wraps to 24", SpellDamage(Dice{0, 40, 1}, 7, 0).Mean(), 24)
	near(t, "Energy Blast PL7", SpellDamage(Dice{0, 1, 7}, 7, 0).Mean(), 28)
}

// combat-2.md C07：hit=(r≤5) 或 (r≤95 且 r≤A)；傷害資格同一顆骰再比 ARpen。
func TestSwingProb(t *testing.T) {
	near(t, "HitProb(200)", HitProb(200), 0.95)
	near(t, "HitProb(-50)", HitProb(-50), 0.05)
	near(t, "HitProb(60)", HitProb(60), 0.60)
	near(t, "GateProb(200,50)", GateProb(200, 50), 0.50)
	near(t, "GateProb(40,90)", GateProb(40, 90), 0.40)
	near(t, "GateProb(1,1)", GateProb(1, 1), 0.05)
}

// formulas.md F23＋abilities.md：先下限 1，吸收後可歸零；DA=0 保留 D1。
func TestPhysDamage(t *testing.T) {
	one := func(v int) Dist { return Dist{v: 1} }
	near(t, "B0 floor 1", PhysDamage(one(0), 0, 0).Mean(), 1)
	near(t, "B0 DA60 → 0", PhysDamage(one(0), 0, 60).Mean(), 0)
	near(t, "B2 DA25", PhysDamage(one(2), 0, 25).Mean(), 2)
	near(t, "B10 +15%", PhysDamage(one(10), 15, 0).Mean(), 12)
	near(t, "B10 +15% DA50", PhysDamage(one(10), 15, 50).Mean(), 6)
}

// formulas.md F24、formulas-2.md:19：Δ=floor(K/25)+[R(25)<K mod 25]，期望門檻。
func TestKillThreshold(t *testing.T) {
	near(t, "Cane+125", KillThreshold(15, 125), 20)
	near(t, "Cane+100", KillThreshold(15, 100), 19)
	near(t, "Cane+110", KillThreshold(15, 110), 19.4)
	near(t, "bow+peacemakers+125", KillThreshold(6+20, 125), 31)
	near(t, "byte wrap", KillThreshold(250+10, 0), 4)
}

// formulas.md F08/F09/F04、combat-2.md C02。
func TestRoundCounts(t *testing.T) {
	for _, c := range []struct{ a, main, off int }{{49, 1, 1}, {50, 2, 1}, {75, 2, 2}, {100, 3, 2}} {
		if m, o := Attacks(c.a); m != c.main || o != c.off {
			t.Errorf("Attacks(%d) = %d,%d, want %d,%d", c.a, m, o, c.main, c.off)
		}
	}
	for _, c := range []struct{ v, s int }{{66, 1}, {67, 2}, {99, 2}, {100, 3}} {
		if got := SwingCap(c.v); got != c.s {
			t.Errorf("SwingCap(%d) = %d, want %d", c.v, got, c.s)
		}
	}
	near(t, "LS extra L28", LSExtraSwings(28), 0.12*4)
	near(t, "LS extra L20", LSExtraSwings(20), 0.11*4)
	near(t, "LS extra L5", LSExtraSwings(5), float64(Scale(5, 12))/100*3)
	near(t, "LS extra L14", LSExtraSwings(14), float64(Scale(14, 12))/100*(3+5.0/11))
}

// formulas-2.md 法術強度、combat-2.md C12。
func TestSpell(t *testing.T) {
	if got := SpellStrength(1, 1, 2, 1, false); got != 2 {
		t.Errorf("SpellStrength(1,1,2,1) = %d, want 2", got)
	}
	if got := SpellStrength(1, 1, 2, 1, true); got != 0 {
		t.Errorf("SpellStrength type7 = %d, want 0 (max 0)", got)
	}
	for _, c := range []struct {
		skill int
		on    bool
		b     int
	}{{100, true, 26}, {0, true, 1}, {100, false, 0}} {
		if got := PowerCastB(c.skill, c.on); got != c.b {
			t.Errorf("PowerCastB(%d,%v) = %d, want %d", c.skill, c.on, got, c.b)
		}
	}
	near(t, "1d2 b26", SpellDamage(Dice{0, 1, 2}, 1, 26).Mean(), 1.5)
	near(t, "1d2 b50", SpellDamage(Dice{0, 1, 2}, 1, 50).Mean(), 2)
}

// combat-2.md C14：e=Rj+3(Lt−S)；damage=v−floor(v·FR/100)，不是 floor(v·(100−FR)/100)。
func TestResist(t *testing.T) {
	one := func(v int) Dist { return Dist{v: 1} }
	near(t, "e≤0", Resist(one(100), 0, 0, 5).Mean(), 100)
	near(t, "e=1", Resist(one(100), 1, 0, 0).Mean(), 99)
	near(t, "e=3", Resist(one(100), 3, 0, 0).Mean(), 97)
	near(t, "e=3 v=1 keeps 1", Resist(one(1), 3, 0, 0).Mean(), 1)
	near(t, "e=300 caps 100", Resist(one(50), 300, 0, 0).Mean(), 0)
	near(t, "9999 passes", Resist(one(9999), 300, 0, 0).Mean(), 9999)
}

func u16name(s string, units int) []byte {
	b := make([]byte, 2*units)
	for i, u := range utf16.Encode([]rune(s)) {
		binary.LittleEndian.PutUint16(b[2*i:], u)
	}
	return b
}

// combat-2.md C11：SPELLTABLES 檔頭 count,version；磁碟 stride 704；F+o＝記錄+0x101+o。
func TestLoadSpellsSynthetic(t *testing.T) {
	b := make([]byte, 8+704)
	binary.LittleEndian.PutUint32(b, 1)
	binary.LittleEndian.PutUint32(b[4:], 1)
	r := b[8:]
	copy(r[0x19c:], u16name("Zap", 64))
	copy(r[0x152:], []byte{0xfe, 0xff, 3, 6})
	binary.LittleEndian.PutUint32(r[0x14a:], 4)
	binary.LittleEndian.PutUint32(r[0x157:], 2)
	binary.LittleEndian.PutUint32(r[0x234:], 5)
	binary.LittleEndian.PutUint32(r[0x230:], 3)
	binary.LittleEndian.PutUint32(r[0x238:], 6)
	p := filepath.Join(t.TempDir(), "s.dbs")
	os.WriteFile(p, b, 0o600)
	s, err := LoadSpells(p)
	if err != nil {
		t.Fatal(err)
	}
	want := Spell{Name: "Zap", Dice: Dice{-2, 3, 6}, ManaPerPL: 4, Level: 2, Realm: 5, Range: 3, Type: 6}
	if len(s) != 1 || s[0] != want {
		t.Fatalf("got %+v, want %+v", s, want)
	}
	os.WriteFile(p, b[:len(b)-1], 0o600)
	if _, err := LoadSpells(p); err == nil {
		t.Error("short file must error")
	}
}

// 原版資料驗證：repo 不含原檔，WIZ8_DBS 指向玩家自有 DATABASES 目錄才跑（cc 本機 2026-10-08 讀過的值）。
func TestRealData(t *testing.T) {
	dir := os.Getenv("WIZ8_DBS")
	if dir == "" {
		t.Skip("WIZ8_DBS 未設：原版資料驗證未跑")
	}
	sp, err := LoadSpells(filepath.Join(dir, "SPELLTABLES.DBS"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		i           int
		name        string
		d           Dice
		mana, level int32
	}{{5, "Energy Blast", Dice{0, 1, 7}, 2, 1}, {10, "Mind Stab", Dice{1, 1, 7}, 3, 1}, {36, "Fireball", Dice{0, 2, 4}, 6, 3}} {
		s := sp[c.i]
		if s.Name != c.name || s.Dice != c.d || s.ManaPerPL != c.mana || s.Level != c.level {
			t.Errorf("spell %d = %+v", c.i, s)
		}
	}
	it, err := LoadItems(filepath.Join(dir, "ITEMS.DBS"))
	if err != nil {
		t.Fatal(err)
	}
	for id, k := range map[int]uint8{29: 15, 131: 20, 771: 6, 139: 8, 70: 0, 739: 0} {
		if it[id].Effects[5] != k {
			t.Errorf("item %d kill = %d, want %d", id, it[id].Effects[5], k)
		}
	}
	if it[29].Name != "Cane of Corpus" || it[139].Dice != (Dice{5, 5, 5}) {
		t.Errorf("items: %+v / %+v", it[29], it[139])
	}
	mo, err := LoadMonsters(filepath.Join(dir, "MONSTERS.DBS"))
	if err != nil {
		t.Fatal(err)
	}
	m := mo[204]
	if m.Name != "Don Barlone# 20" || m.HP != (Dice{232, 5, 30}) || m.AC != -20 || m.Resist != [6]uint8{99, 99, 99, 99, 80, 80} || m.Level != 20 || m.Absorb != 0 {
		t.Errorf("monster 204 = %+v", m)
	}
}

// 階段 8.1：exe 表讀取（WIZ8_EXE 指向玩家自有 Wiz8.exe 才跑；期望值為 cc 2026-10-08 讀表結果，種族與 jeffludwig 交叉核對一致）。
func TestExeTables(t *testing.T) {
	p := os.Getenv("WIZ8_EXE")
	if p == "" {
		t.Skip("WIZ8_EXE 未設：exe 表驗證未跑")
	}
	x, err := LoadExe(p)
	if err != nil {
		t.Fatal(err)
	}
	if x.Race[Lizardman] != [7]int{60, 25, 25, 70, 40, 50, 30} || x.ClassMin[Fighter] != [7]int{55, 0, 0, 50, 50, 0, 0} {
		t.Errorf("race/class rows: %v / %v", x.Race[Lizardman], x.ClassMin[Fighter])
	}
	if x.PMult != [15]int{4, 4, 4, 4, 4, 4, 4, 3, 3, 3, 2, 2, 2, 2, 2} || x.MainSkill[Fighter] != 16 {
		t.Errorf("PMult %v, MainSkill[Fighter] %d", x.PMult, x.MainSkill[Fighter])
	}
	if x.SkillAttr[0] != [2]int{0, 4} || x.Avail[18][Fighter] == 0 || x.Avail[14][Fighter] != 0 {
		t.Errorf("skill tables: attr0 %v avail18 %d avail14 %d", x.SkillAttr[0], x.Avail[18][Fighter], x.Avail[14][Fighter])
	}
	if _, err := LoadExe(filepath.Join(filepath.Dir(p), "PATCH.001")); err == nil {
		t.Error("non-PE file must error")
	}
}
