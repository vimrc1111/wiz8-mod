package calc

import (
	"math"
	"math/big"
)

func init() {
	names := []string{"characterSlot", "characterCharm", "attackContext", "monsterListPointer", "monsterListCount"}
	shared := []string{"combatPhase", "combatMode", "specialSpellMode", "specialItemMode", "characterPresent", "characterHP", "characterCondition", "mainAttackKind", "mainAttackAvailable", "offAttackKind", "offAttackAvailable", "characterInParty", "mainItemIndex", "offItemIndex", "item0Range", "item1Range", "item2Range", "item3Range", "spellId", "spellRange", "item0Spell", "item1Spell", "item2Spell", "item3Spell", "mainActionKind", "offActionKind", "battleActionKind", "item0SpellRange", "item1SpellRange", "item2SpellRange", "item3SpellRange", "partyX", "partyY", "partyZ", "partyRadius", "character0Quadrant", "character1Quadrant", "character2Quadrant", "character3Quadrant", "character4Quadrant", "character5Quadrant", "character6Quadrant", "character7Quadrant", "partyHeadingDegrees", "quadrant0Slot0", "quadrant0Slot1", "quadrant0Slot2", "quadrant1Slot0", "quadrant1Slot1", "quadrant1Slot2", "quadrant2Slot0", "quadrant2Slot1", "quadrant2Slot2", "quadrant3Slot0", "quadrant3Slot1", "quadrant3Slot2", "quadrant4Slot0", "quadrant4Slot1", "quadrant4Slot2", "character0FormationUnavailable", "character1FormationUnavailable", "character2FormationUnavailable", "character3FormationUnavailable", "character4FormationUnavailable", "character5FormationUnavailable", "character6FormationUnavailable", "character7FormationUnavailable"}
	names = append(names, shared...)
	for j := 0; j < attackSearchMonsters; j++ {
		for _, f := range []string{"Id", "MaxHp", "Hp", "ActiveA", "ActiveB", "Disposition", "Species", "OrderKey", "TargetEnabled", "PhysicalAllowed", "SpellAllowed", "X", "Y", "Z", "Radius"} {
			names = append(names, "mon"+string(rune('0'+j))+f)
		}
	}
	for s := 0; s < attackSearchMonsters; s++ {
		for i := 0; i < 3; i++ {
			d := string(rune('0' + i))
			names = append(names, "tpl"+string(rune('0'+s))+"Active"+d, "tpl"+string(rune('0'+s))+"Range"+d)
		}
	}
	Nodes["0053c720"] = Node{Addr: "0053c720", In: names, Out: []string{"targetId"}, F: attackTargetSearchNode}
}

// Fixture caps: the list at 0x683fad holds at most four monsters and four
// cached templates (0x6840c7 + 4*species); anything larger panics.
const (
	attackSearchMonsters = 4
	attackSearchListPtr  = 0x20000000 + 300000 // fixture address of the monster list object
	attackSearchShared   = 67
)

type attackSearchRow struct {
	id, key uint32
	flag    bool
	dist    float32
}

// attackTargetSearchNode: 0x53c720 builds one 0x1c-byte row per eligible
// monster (0x53c78a..0x53c8bc) and sorts them with 0x53c920; the reply is the
// first row's id, or -1 (0x53c8e6). Inputs: 5 header fields, 67 shared
// 0x5194e0 fields, 15 fields per monster, 6 per template.
func attackTargetSearchNode(in []int64, r Rand) []int64 {
	const perMon, perTpl, hdr = 15, 6, 5
	if len(in) != hdr+attackSearchShared+attackSearchMonsters*(perMon+perTpl) {
		panic("attack target search: wrong input count")
	}
	if in[3] != 0 && in[3] != attackSearchListPtr {
		panic("attack target search: list pointer must be 0 or the fixture list")
	}
	// 0x5e2c70: a null list gives count 0 (0x53c740 -> -1).
	if in[3] == 0 || in[4] == 0 {
		return []int64{-1}
	}
	if in[4] < 0 || in[4] > attackSearchMonsters {
		panic("attack target search: monster count beyond fixture cap")
	}
	shared := in[hdr : hdr+attackSearchShared]
	monBase := hdr + attackSearchShared
	tplBase := monBase + attackSearchMonsters*perMon
	var rows []attackSearchRow
	for j := 0; j < int(in[4]); j++ {
		m := in[monBase+j*perMon : monBase+(j+1)*perMon]
		// 0x53c78a..0x53c7ab: M+0x14, M+0x15, unsigned M+0x2b nonzero.
		if m[3] == 0 || m[4] == 0 || uint32(m[2]) == 0 {
			continue
		}
		if m[3] > 255 || m[4] > 255 || m[2] < 0 || m[2] > math.MaxUint32 {
			panic("attack target search: bad monster flag/hp")
		}
		// 0x546f10 (charm and M+0x16) must say 1.
		if targetDispositionNode([]int64{in[0], in[1], m[5]}, r)[0]&0xff != 1 {
			continue
		}
		v := make([]int64, 78)
		v[0], v[1], v[2], v[3] = in[0], 2, in[2], 0
		v[4], v[5], v[6] = m[8], m[9], m[10]
		copy(v[7:41], shared[0:34])
		v[41], v[42], v[43], v[44] = m[11], m[12], m[13], m[14]
		copy(v[45:], shared[34:])
		if monsterTargetAttackRangeNode(v, r)[0]&0xff == 0 {
			continue
		}
		if m[1] == 0 { // 0x53c874 srAssertFail on M+0x27 == 0.
			panic("attack target search: M+0x27 is zero")
		}
		row := attackSearchRow{id: uint32(m[0]), key: uint32(m[7])}
		if m[6] < 0 || m[6] >= attackSearchMonsters {
			panic("attack target search: species beyond fixture cap")
		}
		row.flag = attackSearchFlag(in, tplBase+int(m[6])*perTpl, shared, m)
		row.dist = attackSearchDistance(shared, m)
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return []int64{-1}
	}
	// msvcrt qsort shortsort (n <= 8): repeatedly move the strict maximum to the end.
	for hi := len(rows) - 1; hi > 0; hi-- {
		mx := 0
		for p := 1; p <= hi; p++ {
			if attackSearchCmp(rows[p], rows[mx]) > 0 {
				mx = p
			}
		}
		rows[mx], rows[hi] = rows[hi], rows[mx]
	}
	return []int64{int64(int32(rows[0].id))}
}

// 0x53c920: key ascending, flagged first, then float32 distance ascending.
func attackSearchCmp(a, b attackSearchRow) int {
	switch {
	case a.key < b.key:
		return -1
	case a.key > b.key:
		return 1
	case a.flag && !b.flag:
		return -1
	case !a.flag && b.flag:
		return 1
	case a.dist < b.dist:
		return -1
	case a.dist == b.dist:
		return 0
	}
	return 1
}

// 0x51a800(template, 1): max range byte of active entries with range <= 1.
func attackSearchTemplateRange(in []int64, base int) int64 {
	best := int64(-1)
	for i := 0; i < 3; i++ {
		act, rng := in[base+2*i], in[base+2*i+1]
		if act < 0 || act > 255 || rng < 0 || rng > 255 {
			panic("attack target search: bad template byte")
		}
		if act != 0 && rng <= 1 && rng > best {
			best = rng
		}
	}
	return best
}

func attackSearchFlag(in []int64, tpl int, shared, m []int64) bool {
	if attackSearchTemplateRange(in, tpl) == -1 {
		return false
	}
	q := attackSearchQuadrant(shared, m)
	if q == 2 {
		return true
	}
	// 0x51ae60 for the monster's quadrant, then for the center region 4.
	free := func(q int64) bool {
		v := make([]int64, 24)
		v[0] = q
		copy(v[1:], shared[44:67]) // 15 quadrant slots and 8 unavailable flags
		return formationQuadrantAvailableNode(v, nil)[0]&0xff != 0
	}
	return !free(q) && free(4)
}

// 0x4e5af0: monster direction relative to the party heading (0x686a40), as in
// 0x51aec0 (0x4be420 angle, 0x517c90 wrap, 0x555f58 degrees, 0x555f6e..0x555f93).
func attackSearchQuadrant(shared, m []int64) int64 {
	if formationRangeError != nil {
		panic(formationRangeError)
	}
	heading := shared[43]
	if heading < 0 || heading >= 360 {
		panic("attack target search: heading must be 0..359")
	}
	f := monsterAttackTargetFloat
	dx, dz := f(m[11])-f(shared[31]), f(m[13])-f(shared[33])
	f(m[12])
	f(shared[32])
	t := formationRangeConstants
	angle := t[1]
	if dz == 0 {
		if dx > 0 {
			angle = t[0]
		}
	} else {
		angle = float64(float32(math.Atan2(dx, dz)))
	}
	op := func(v float64) *big.Float { return new(big.Float).SetPrec(64).SetFloat64(v) }
	a := op(angle)
	a.Add(a, op(t[3]))
	for a.Sign() < 0 {
		a.Add(a, op(t[2]))
	}
	for a.Cmp(op(t[2])) >= 0 {
		a.Sub(a, op(t[2]))
	}
	a.Mul(a, op(t[4]))
	degrees, _ := a.Int64()
	rel := degrees - heading
	if rel < 0 {
		rel += 360
	}
	return (rel + 45) % 360 / 90
}

// 0x4c7cb0 stored as float32 (0x53c848, 0x53c8a9).
func attackSearchDistance(shared, m []int64) float32 {
	f := monsterAttackTargetFloat
	d := monsterAttackTargetDistance([3]float64{f(m[11]), f(m[12]), f(m[13])}, [3]float64{f(shared[31]), f(shared[32]), f(shared[33])}, true, f(m[14]), f(shared[34]))
	v, _ := d.Float32()
	return v
}
