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

// Dispatch tables of 0x519c00 and its 0x51a510 child are read from the player's
// Wiz8.exe at load time (0x519f40, 0x519f64, 0x51a6f0, 0x51a714); the code
// below maps their destination addresses to meanings, never the table bytes.
type monsterAttackTargetTables struct {
	partyAct, partyLen, monsterAct, monsterLen [9]uint32 // Len tables use the first five entries.
}

var monsterAttackTargetData, monsterAttackTargetDataErr = loadMonsterAttackTargetTables()

func loadMonsterAttackTargetTables() (monsterAttackTargetTables, error) {
	var t monsterAttackTargetTables
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("monster attack target: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	for _, table := range []struct {
		va  uint32
		n   int
		dst *[9]uint32
	}{{0x519f40, 9, &t.partyAct}, {0x519f64, 5, &t.partyLen}, {0x51a6f0, 9, &t.monsterAct}, {0x51a714, 5, &t.monsterLen}} {
		for i := 0; i < table.n; i++ {
			off, e := peOffset(b, table.va+4*uint32(i))
			if e != nil {
				return t, e
			}
			if off > len(b)-4 {
				return t, fmt.Errorf("monster attack target: truncated table %#x", table.va)
			}
			table.dst[i] = binary.LittleEndian.Uint32(b[off:])
		}
	}
	return t, nil
}

var monsterAttackTargetInputs = func() []string {
	n := []string{"attackIndex", "allyFlag", "monsterListPointer", "monsterCount", "combatMode", "partyHeading"}
	for c := 0; c < 8; c++ {
		n = append(n, fmt.Sprintf("character%dQuadrant", c))
	}
	for j := 0; j < 15; j++ {
		n = append(n, fmt.Sprintf("formationSlot%d", j))
	}
	for c := 0; c < 8; c++ {
		for _, f := range []string{"slot%dActive", "character%dCurrentHP", "character%dStatus", "character%dCharm", "character%dFormationUnavailable"} {
			n = append(n, fmt.Sprintf(f, c))
		}
	}
	n = append(n, "partyXBits", "partyYBits", "partyZBits", "partyRadiusBits",
		"mTemplateId", "mVisible", "mOwnerVisible", "mDisposition", "mAllegianceWord", "mAction", "mSpellIndex", "mAttackEnabled",
		"mReach0", "mReach1", "mReach2", "mReach3", "mAltFlag", "mSpellFlag", "relationCount")
	for k := 0; k < 3; k++ {
		n = append(n, fmt.Sprintf("relation%dId", k), fmt.Sprintf("relation%dActive", k))
		for q := 0; q < 4; q++ {
			n = append(n, fmt.Sprintf("relation%dReach%d", k, q))
		}
	}
	n = append(n, "mXBits", "mYBits", "mZBits", "mRadiusBits")
	for j := 1; j <= 3; j++ {
		for _, f := range []string{"Id", "TemplateId", "Active", "Visible", "Disposition", "CurrentHP", "AllegianceWord", "Status", "XBits", "YBits", "ZBits", "RadiusBits"} {
			n = append(n, fmt.Sprintf("mon%d%s", j, f))
		}
	}
	for k := 0; k < 5; k++ {
		for _, f := range []string{"Special", "Group", "AttackPresent", "AttackRange"} {
			n = append(n, fmt.Sprintf("tpl%d%s", k, f))
		}
	}
	return append(n, "spellRange")
}()

func init() {
	Nodes["00519c00"] = Node{Addr: "00519c00", In: monsterAttackTargetInputs, Out: []string{"targetAvailable"}, F: monsterAttackTargetAvailableNode}
}

type monsterAttackTargetRecord struct {
	id, template, disposition, hp, word, status int64
	active, visible                             bool
	pos                                         [3]float64
	radius                                      float64
}

// Destination addresses that name the meaning of a dispatch-table entry.
type monsterAttackTargetDispatch struct {
	act                [9]uint32
	weapon, spell, two uint32
	zero, none         uint32
	length             [9]uint32
	lengthDest         [5]uint32 // destinations for length 0, 2, 4, 25, 50
}

// Fixture template cache (0x6840c7 + 4*id): ids 0..3 and the special id 548
// (0x224) are pre-loaded; any other id would reach the file loader (0x4e5772).
func monsterAttackTargetTemplate(id int64) int {
	switch id {
	case 0, 1, 2, 3:
		return int(id)
	case 548:
		return 4
	}
	panic("monster attack target: template id outside fixture cache (0..3, 548)")
}

func monsterAttackTargetFloat(bits int64) float64 {
	if bits < 0 || bits > math.MaxUint32 {
		panic("monster attack target: expected float32 bits")
	}
	f := float64(math.Float32frombits(uint32(bits)))
	if math.IsInf(f, 0) || math.IsNaN(f) {
		panic("monster attack target: expected finite position")
	}
	return f
}

// x87 mode of the original: every operation rounds to a 64-bit significand.
func monsterAttackTargetExt(f float64) *big.Float { return new(big.Float).SetPrec(64).SetFloat64(f) }

func monsterAttackTargetOp(f func(z, x, y *big.Float) *big.Float, x, y *big.Float) *big.Float {
	return f(new(big.Float).SetPrec(64), x, y)
}

// Surface distance of 0x4c7cb0 (party: height-adjusted party position, two
// radii) and 0x4c7dd0 (two monsters, both radii), clamped below at 0x5ebb34.
// a is the monster; b the party or the other monster.
func monsterAttackTargetDistance(a, b [3]float64, partyHeight bool, ra, rb float64) *big.Float {
	if interceptDataErr != nil {
		panic(interceptDataErr)
	}
	ext, op := monsterAttackTargetExt, monsterAttackTargetOp
	by := b[1]
	if partyHeight {
		// 0x4c7ce1..0x4c7cf4 stores the height-adjusted party Y as float32.
		adjusted, _ := op((*big.Float).Sub, ext(b[1]), ext(interceptData.height)).Float32()
		by = float64(adjusted)
	}
	dx, dy, dz := op((*big.Float).Sub, ext(a[0]), ext(b[0])), op((*big.Float).Sub, ext(a[1]), ext(by)), op((*big.Float).Sub, ext(a[2]), ext(b[2]))
	sq := func(x *big.Float) *big.Float { return op((*big.Float).Mul, x, x) }
	// Order of the original sum: (z*z + y*y) + x*x.
	sum := op((*big.Float).Add, op((*big.Float).Add, sq(dz), sq(dy)), sq(dx))
	d := new(big.Float).SetPrec(64).Sqrt(sum)
	d = op((*big.Float).Sub, d, ext(ra))
	d = op((*big.Float).Sub, d, ext(rb))
	if d.Cmp(ext(interceptData.minimum)) < 0 {
		d = ext(interceptData.minimum)
	}
	return d
}

// length * 500.0 (0x5ebc40) must not be below the distance (0x519e58..0x519e69,
// 0x51a6c9..0x51a6da); the original comparison is false for unordered values.
func monsterAttackTargetWithin(length int64, d *big.Float) bool {
	limit := new(big.Float).SetPrec(64).Mul(new(big.Float).SetPrec(64).SetInt64(length), monsterAttackTargetExt(interceptData.scale))
	return limit.Cmp(d) >= 0
}

// class: -1 none, else 0..3 (assertion otherwise); returns game range units.
func (m monsterAttackTargetDispatch) lengthOf(class int64) int64 {
	if class < 0 || class > 3 {
		panic("monster attack target: range class assertion")
	}
	switch m.length[class+1] {
	case m.lengthDest[1]:
		return 2
	case m.lengthDest[2]:
		return 4
	case m.lengthDest[3]:
		return 25
	case m.lengthDest[4]:
		return 50
	}
	panic("monster attack target: unknown length dispatch")
}

func monsterAttackTargetAvailableNode(in []int64, r Rand) []int64 {
	if len(in) != len(monsterAttackTargetInputs) {
		panic("monster attack target: unexpected input count")
	}
	if monsterAttackTargetDataErr != nil {
		panic(monsterAttackTargetDataErr)
	}
	v := make(map[string]int64, len(in))
	for i, name := range monsterAttackTargetInputs {
		v[name] = in[i]
	}
	if v["attackIndex"] < 0 || v["attackIndex"] > 2 || v["mSpellIndex"] < 0 || v["mSpellIndex"] > 2 || v["allyFlag"] < 0 || v["allyFlag"] > 255 {
		panic("monster attack target: attack/spell index must be 0..2 and flag a byte")
	}
	if v["monsterCount"] < 0 || v["monsterCount"] > 4 || v["relationCount"] < 0 || v["relationCount"] > 3 {
		panic("monster attack target: fixture holds at most 4 monsters and 3 relations")
	}
	const listAddress = 0x20000000 + 106496 // emulator fixture list object, not game data.
	if v["monsterListPointer"] != 0 && v["monsterListPointer"] != listAddress {
		panic("monster attack target: list pointer must be null or the fixture list")
	}
	action := v["mAction"] // u32; values above 8 have no reach.
	pos := func(p string) [3]float64 {
		return [3]float64{monsterAttackTargetFloat(v[p+"XBits"]), monsterAttackTargetFloat(v[p+"YBits"]), monsterAttackTargetFloat(v[p+"ZBits"])}
	}
	monster := func(j int) monsterAttackTargetRecord {
		if j == 0 { // the monster itself; id 77 and active come from the fixture.
			return monsterAttackTargetRecord{id: 77, template: v["mTemplateId"], disposition: v["mDisposition"], hp: 1, word: v["mAllegianceWord"],
				active: true, visible: v["mVisible"] != 0, pos: pos("m"), radius: monsterAttackTargetFloat(v["mRadiusBits"])}
		}
		p := fmt.Sprintf("mon%d", j)
		return monsterAttackTargetRecord{id: v[p+"Id"], template: v[p+"TemplateId"], disposition: v[p+"Disposition"], hp: v[p+"CurrentHP"],
			word: v[p+"AllegianceWord"], status: v[p+"Status"], active: v[p+"Active"] != 0, visible: v[p+"Visible"] != 0,
			pos: pos(p), radius: monsterAttackTargetFloat(v[p+"RadiusBits"])}
	}
	self := monster(0)
	selfTpl := monsterAttackTargetTemplate(self.template)
	tplField := func(k int, f string) int64 { return v[fmt.Sprintf("tpl%d%s", k, f)] }

	// 0x519c15..0x519c47: a visible monster with a visible owner looks for
	// friendly targets (2); otherwise the caller flag (nonzero: friendly).
	want := int64(1)
	if (self.visible && v["mOwnerVisible"] != 0) || v["allyFlag"] != 0 {
		want = 2
	}
	// 0x519cae..0x519d13 / 0x51a55d..0x51a5b8: which reach flag applies.
	reachKind := func() int {
		switch action {
		case 0:
			if c := tplField(selfTpl, "AttackRange"); c == 2 || c == 3 {
				if v["mAltFlag"] != 0 {
					return 1
				}
			}
		case 2: // 0x505e80 returns 2 plus the spell flag.
			if v["mSpellFlag"] != 0 {
				return 3
			}
			return 2
		case 3:
			return 2
		}
		return 0
	}
	t := monsterAttackTargetData
	// Destination addresses of 0x519f40/0x519f64 and 0x51a6f0/0x51a714 entries.
	partyD := monsterAttackTargetDispatch{act: t.partyAct, weapon: 0x519d38, spell: 0x519d8b, two: 0x519dae, zero: 0x519db5, none: 0x519db9,
		length: t.partyLen, lengthDest: [5]uint32{0x519e0b, 0x519e0f, 0x519e16, 0x519e1d, 0x519e24}}
	monsterD := monsterAttackTargetDispatch{act: t.monsterAct, weapon: 0x51a5e0, spell: 0x51a64a, two: 0x51a66d, zero: 0x51a674, none: 0x51a6e6,
		length: t.monsterLen, lengthDest: [5]uint32{0x51a646, 0x51a678, 0x51a67f, 0x51a686, 0x51a68d}}
	rangeClass := func(d monsterAttackTargetDispatch) int64 {
		if action > 8 {
			return -1
		}
		switch d.act[action] {
		case d.weapon:
			if v["attackIndex"] >= 3 {
				panic("monster attack target: attack index assertion")
			}
			if tplField(selfTpl, "AttackPresent") == 0 {
				panic("monster attack target: missing attack assertion")
			}
			return tplField(selfTpl, "AttackRange")
		case d.spell:
			return v["spellRange"]
		case d.two:
			return 2
		case d.zero:
			return 0
		case d.none:
			return -1
		}
		panic("monster attack target: unknown reach dispatch")
	}

	// Party loop: eight slots in ascending order (0x519c54..0x519e9f).
	for c := 0; c < 8; c++ {
		cs := func(f string) int64 { return v[fmt.Sprintf(f, c)] }
		if cs("slot%dActive") == 0 || cs("character%dCurrentHP") == 0 || cs("character%dStatus") >= 0x12 {
			continue
		}
		if targetDispositionNode([]int64{int64(c), cs("character%dCharm"), self.disposition}, r)[0] != want {
			continue
		}
		if v["mAttackEnabled"] != 1 || v[fmt.Sprintf("mReach%d", reachKind())] == 0 {
			continue
		}
		class := rangeClass(partyD)
		if class == -1 {
			continue
		}
		if v["combatMode"] != 0 && class >= 0 && class <= 1 {
			fin := []int64{int64(c)}
			for q := 0; q < 8; q++ {
				fin = append(fin, v[fmt.Sprintf("character%dQuadrant", q)])
			}
			fin = append(fin, v["mXBits"], v["mYBits"], v["mZBits"], v["partyXBits"], v["partyYBits"], v["partyZBits"], v["partyHeading"])
			for j := 0; j < 15; j++ {
				fin = append(fin, v[fmt.Sprintf("formationSlot%d", j)])
			}
			for q := 0; q < 8; q++ {
				fin = append(fin, v[fmt.Sprintf("character%dFormationUnavailable", q)])
			}
			penalty := formationRangePenaltyNode(fin, r)[0] & 0xff
			skip := false
			for penalty != 0 { // 0x519dec..0x519df3: each penalty point shortens the class.
				if class == 0 {
					skip = true
					break
				}
				class--
				penalty--
			}
			if skip {
				continue
			}
		}
		party := [3]float64{monsterAttackTargetFloat(v["partyXBits"]), monsterAttackTargetFloat(v["partyYBits"]), monsterAttackTargetFloat(v["partyZBits"])}
		if monsterAttackTargetWithin(partyD.lengthOf(class), monsterAttackTargetDistance(self.pos, party, true, self.radius, monsterAttackTargetFloat(v["partyRadiusBits"]))) {
			return []int64{1}
		}
	}

	// Other monsters (0x519ea5..0x519f2b); the list count is re-read each pass.
	count := v["monsterCount"]
	if v["monsterListPointer"] == 0 {
		count = 0
	}
	for i := 1; i < int(count); i++ { // entry 0 is the monster itself (0x519ecb).
		other := monster(i)
		otherTpl := monsterAttackTargetTemplate(other.template)
		if !other.active || !other.visible || other.hp == 0 || other.status >= 0x12 {
			continue
		}
		// 0x546f80: 2 same disposition, 0 neutral/special, else 1 unless the
		// same template group splits on the allegiance word.
		relation := int64(1)
		switch {
		case selfTpl == 4 || otherTpl == 4:
			relation = 0
		case self.disposition == other.disposition:
			relation = 2
		case self.disposition == 0 || other.disposition == 0:
			relation = 0
		default:
			g, h := tplField(selfTpl, "Group"), tplField(otherTpl, "Group")
			if g != 0 && h != 0 && g == h && (self.word != 0) == (other.word != 0) {
				relation = 0
			}
		}
		if relation != want {
			continue
		}
		// 0x51a510: ordinary template, a relation record naming the other
		// monster, its reach flag for the action, then the range check.
		if tplField(selfTpl, "Special") != 0 {
			continue
		}
		rel := -1
		for k := 0; k < int(v["relationCount"]); k++ {
			if v[fmt.Sprintf("relation%dId", k)] == other.id {
				rel = k
				break
			}
		}
		if rel < 0 || v[fmt.Sprintf("relation%dActive", rel)] != 1 || v[fmt.Sprintf("relation%dReach%d", rel, reachKind())] == 0 {
			continue
		}
		class := rangeClass(monsterD)
		if class == -1 {
			continue
		}
		if monsterAttackTargetWithin(monsterD.lengthOf(class), monsterAttackTargetDistance(self.pos, other.pos, false, self.radius, other.radius)) {
			return []int64{1}
		}
	}
	return []int64{0}
}
