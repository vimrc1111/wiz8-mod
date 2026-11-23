package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type attacksExe struct {
	classes  *ExeTables
	dice     [12]uint32
	combat   [15]uint32
	strength [41]uint32
}

var attacksTables, attacksTablesErr = loadAttacksExe()

// Tables are read from the user's PE, before evaluating the pure rule.
// 0x6172f0 supplies martial-arts dice for effective skill 0..125;
// 0x4ee958/0x4ee94c classify skills and 0x4ee984/0x4ee97c select
// item subtypes that bypass strength modifiers. No original table is embedded.
func loadAttacksExe() (attacksExe, error) {
	var t attacksExe
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	var err error
	t.classes, err = LoadExe(path)
	if err != nil {
		return t, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	read := func(va uint32, size int) ([]byte, error) {
		o, e := peOffset(b, va)
		if e != nil {
			return nil, e
		}
		if size > len(b)-o {
			return nil, fmt.Errorf("calc attacks: truncated table %#x", va)
		}
		return b[o : o+size], nil
	}
	raw, err := read(0x6172f0, len(t.dice)*4)
	if err != nil {
		return t, err
	}
	for i := range t.dice {
		t.dice[i] = binary.LittleEndian.Uint32(raw[4*i:])
	}
	for _, spec := range []struct {
		selectors, targets uint32
		dst                []uint32
	}{{0x4ee958, 0x4ee94c, t.combat[:]}, {0x4ee984, 0x4ee97c, t.strength[:]}} {
		idx, e := read(spec.selectors, len(spec.dst))
		if e != nil {
			return t, e
		}
		for i, k := range idx {
			v, e := read(spec.targets+4*uint32(k), 4)
			if e != nil {
				return t, e
			}
			spec.dst[i] = binary.LittleEndian.Uint32(v)
		}
	}
	return t, nil
}

// The case models two independently editable item records (IDs 0 and 1),
// including aliases, and the selected party member's NPC record. Addressing
// globals only locate these records. Debug logging is disabled in the fixture.
// Arithmetic follows F06..F12/F26/F28 (notes/re/formulas.md); unsigned and
// signed 32-bit conversions preserve the original intermediate rounding.
func calcAttacksNode(in []int64, r Rand) []int64 { return calcAttacks(in, r, nil) }

// calcAttacks: defs, when set, are the hands' ITEMS definition IDs. The emu
// fixture's item table has two entries, so there itemID is also the definition
// ID; the fight model keeps itemID as the 0/1 slot and passes the real IDs here
// for 0x51c930, whose Omnigun sets key on definition IDs 599..610.
func calcAttacks(in []int64, r Rand, defs *[2]int64) []int64 {
	names := Nodes["004ee220"].In
	if len(in) != len(names) {
		panic("calc attacks: input arity")
	}
	if attacksTablesErr != nil {
		panic(attacksTablesErr)
	}
	v := make(map[string]int64, len(in))
	for i, n := range names {
		v[n] = in[i]
	}
	u := func(x int64) int64 { return int64(uint32(x)) }
	s := func(x int64) int64 { return int64(int32(x)) }
	skill := func(id int64) int64 { return v[fmt.Sprintf("skill%d", id)] }
	item := func(id int64, field string) int64 { return v[fmt.Sprintf("item%d%s", id, field)] }
	var h [2][18]int64
	ids := [2]int64{v["itemID0"], v["itemID1"]}
	for hand := 0; hand < 2; hand++ {
		if ids[hand] < -1 || ids[hand] > 1 {
			panic("calc attacks: item ID outside fixture")
		}
		for j, n := range attacksHandFields {
			if j >= 4 {
				h[hand][j] = v[fmt.Sprintf("prior%d%s", hand, n)]
			}
		}
		typ := int64(0)
		if ids[hand] != -1 {
			k := item(ids[hand], "type")
			if k < 0 || k >= 6 {
				panic("calc attacks: invalid item type")
			}
			if handTypeDispatchErr != nil {
				panic(handTypeDispatchErr)
			}
			switch handTypeDispatch[k] {
			case 0x520248:
				typ = 1
			case 0x52024f:
				typ = 1
				if hand == 1 && ids[0] != -1 && item(ids[0], "subtype") == 131 {
					typ = 3
				}
			case 0x52027e:
				typ = 3
			case 0x520285:
				typ = 2
			default:
				panic("calc attacks: unknown hand classification")
			}
		}
		h[hand][0] = typ
		h[hand][1] = 1
		h[hand][2] = 14
		if ids[hand] != -1 {
			h[hand][2] = item(ids[hand], "weaponSkill")
		}
	}
	bonus := func(id int64) bool { return weaponDamageBonusItemNode([]int64{id, item(id, "subtype")}, r)[0] != 0 }
	for hand := 0; hand < 2; hand++ {
		a := &h[hand]
		other := 1 - hand
		k := a[2]
		target := uint32(0x4ee319)
		if k >= 0 && k < 15 {
			target = attacksTables.combat[k]
		}
		switch target {
		case 0x4ee2b8:
			a[3] = 16
		case 0x4ee2c1:
			a[3] = 17
			id, oid := ids[hand], ids[other]
			if bonus(id) {
				compatible := false
				if oid != -1 {
					pid, poid := id, oid
					if defs != nil {
						pid, poid = defs[hand], defs[other]
					}
					compatible = compatiblePartnerItemsNode([]int64{pid, poid, item(id, "subtype"), item(id, "range"), item(oid, "type"), item(oid, "subtype"), item(oid, "range")}, r)[0] != 0
				}
				if !compatible {
					a[1] = 0
				}
			}
			if item(id, "chargeType") == 4 && v[fmt.Sprintf("charges%d", hand)] == 0 {
				a[1] = 0
			}
		case 0x4ee319:
			a[3] = -1
			a[1] = 0
		default:
			panic(fmt.Sprintf("calc attacks: unexpected combat dispatch %#x", target))
		}
		if hand == 1 {
			if a[3] != h[0][3] || a[0] == 2 || a[0] == 3 || (ids[0] != -1 && item(ids[0], "subtype") == 131) || (a[0] == 0 && h[0][0] != 0) {
				a[1] = 0
			}
		}
	}
	dual := int64(0)
	if h[0][0] == 1 && h[0][1] != 0 && h[1][0] == 1 && h[1][1] != 0 {
		dual = 1
	}
	p := int64(0)
	if v["class"] != -1 {
		for j, m := range attacksTables.classes.PMult {
			p = u(p + v[fmt.Sprintf("classLevel%d", j)]*int64(m))
		}
	}
	str, dex, spd, sen := v["strength"], v["dexterity"], v["speed"], v["senses"]
	load := v["loadCategory"]
	if load < 0 || load > 4 {
		panic("calc attacks: original asserts invalid load category")
	}
	// Encumbrance constants originate in 0x4ee475..0x4ee49d and
	// 0x4ee887..0x4ee8ae, not an embedded executable data table.
	penalty := int64(0)
	if load > 0 {
		penalty = -(1 << uint(load-1))
	}
	for hand := 0; hand < 2; hand++ {
		a, o := &h[hand], &h[1-hand]
		if a[1] == 0 {
			continue
		}
		total := u(u(skill(a[3])+2*skill(a[2]))*2) / 3
		den := int64(20)
		if o[0] == 1 && o[1] != 0 {
			total = u(total + u(skill(o[2]))/2)
			den = 25
		}
		if dual != 0 {
			total = u(total + skill(18))
			den += 10
		}
		a[4] = u(total*10) / den
		e := penalty * 15
		if a[2] == 7 {
			e /= 2
		}
		m := int64(0)
		if dual != 0 {
			m = -10 - int64(hand)*10 - s(100-skill(18))/4
		}
		a[5] = s(m+u(dex)/2+a[4]*2+p)/3 + 60
		// NPC race 15 with personality 32 and unset quest 68: 0x4ee538..0x4ee58e.
		if v["inParty"] != 0 && v["race"] == 15 && v["npcUnavailable"] == 0 && v["npcPersonality"] == 32 && v["quest68"] == 0 {
			a[5] /= 2
		}
		attack := s(u(dex+spd)/2+m+p+e+a[4]) / 3
		a[6] = 1
		if hand == 0 {
			if attack >= 50 {
				a[6] = 2
			}
			if attack >= 100 {
				a[6] = 3
			}
		} else if attack >= 75 {
			a[6] = 2
		}
		// Initiative is read before being reset (0x4ee5ec then 0x4ee64f).
		swing := s(spd+a[8]*10+m+p+e+a[4]) / 3
		a[7] = 1
		if swing >= 67 {
			a[7] = 2
		}
		if swing >= 100 {
			a[7] = 3
		}
		if ids[0] != -1 && item(ids[0], "subtype") == 144 {
			a[7] = 1
		}
		a[8], a[9], a[10], a[11] = 0, 0, 0, 0
		id := ids[hand]
		if id == -1 {
			a[8] = a[4] / 10
			a[13] = 32
			if skill(a[2]) >= 5 {
				a[13] = 96
			}
			wi := skill(a[2]) / 11
			if wi < 0 || wi >= int64(len(attacksTables.dice)) {
				panic("calc attacks: martial-arts skill outside 0..125")
			}
			dice := attacksTables.dice[wi]
			if hand == 0 {
				dice = (dice & 0xffff0000) | uint32(uint16(dice)+2)
			}
			a[12] = int64(dice)
			a[14], a[15], a[16], a[17] = 0, 0, 0, 0
			if str >= 50 {
				a[15] = int64(uint8((str-50)/5)) << 16
			}
		} else {
			a[8] = item(id, "initiative")
			a[9] = item(id, "toHit")
			if o[0] == 3 && ids[1-hand] != -1 {
				a[9] += item(ids[1-hand], "toHit")
			}
			if bonus(id) {
				a[11] = item(id, "damage") * 10
			}
		}
		divisor := int64(1)
		skipStrength := false
		if id != -1 && item(id, "attacks")&0xfe6f == 0 {
			subtype := item(id, "subtype")
			// Dispatch interval starts at subtype 104, ends at 144 (0x4ee774).
			if subtype >= 104 && subtype <= 144 {
				skipStrength = attacksTables.strength[subtype-104] == 0x4ee7fe
			}
			if !skipStrength {
				divisor = 2
			}
		}
		if !skipStrength {
			if str < 50 {
				a[9] -= (50 - str) / (divisor * 10)
				a[11] -= (50 - str) / divisor
			} else if str > 50 {
				divisor *= int64(hand + 1)
				a[9] += (str - 50) / (divisor * 10)
				a[11] += u(str*2-100) / divisor
			}
		}
		if dex > 50 {
			a[9] += (dex - 50) / 10
		} else if dex < 50 {
			a[9] -= (50 - dex) / 10
		}
		if sen > 70 {
			a[9] += (sen - 70) / 10
		} else if sen < 30 {
			a[9] -= (30 - sen) / 10
		}
		hitPenalty := penalty
		if hand == 1 {
			hitPenalty = hitPenalty * 3 / 2
		}
		if a[2] == 7 {
			hitPenalty /= 2
		}
		a[9] += hitPenalty
		if v["rangedExpert"] != 0 && a[3] == 17 {
			a[9] += skill(40)/20 + 1
		}
		if v["meleeExpert"] != 0 && a[3] == 16 {
			a[9] += skill(34)/20 + 1
		}
	}
	out := make([]int64, 0, 37)
	for _, a := range h {
		for _, value := range a {
			out = append(out, u(value))
		}
	}
	return append(out, dual)
}

var attacksHandFields = []string{"type", "enabled", "weaponSkill", "combatSkill", "combinedSkill", "BAR", "attacks", "swings", "initiative", "toHit", "damageFlat", "damagePercent", "damageDice", "attackModes", "effect0", "effect1", "effect2", "effect3"}

func init() {
	Nodes["004ee220"] = Node{Addr: "004ee220",
		In:  []string{"inParty", "class", "race", "strength", "dexterity", "speed", "senses", "loadCategory", "rangedExpert", "meleeExpert", "classLevel0", "classLevel1", "classLevel2", "classLevel3", "classLevel4", "classLevel5", "classLevel6", "classLevel7", "classLevel8", "classLevel9", "classLevel10", "classLevel11", "classLevel12", "classLevel13", "classLevel14", "skill0", "skill1", "skill2", "skill3", "skill4", "skill5", "skill6", "skill7", "skill8", "skill9", "skill10", "skill11", "skill12", "skill13", "skill14", "skill15", "skill16", "skill17", "skill18", "skill19", "skill20", "skill21", "skill22", "skill23", "skill24", "skill25", "skill26", "skill27", "skill28", "skill29", "skill30", "skill31", "skill32", "skill33", "skill34", "skill35", "skill36", "skill37", "skill38", "skill39", "skill40", "itemID0", "charges0", "itemID1", "charges1", "item0type", "item0subtype", "item0weaponSkill", "item0range", "item0initiative", "item0toHit", "item0damage", "item0attacks", "item0chargeType", "item1type", "item1subtype", "item1weaponSkill", "item1range", "item1initiative", "item1toHit", "item1damage", "item1attacks", "item1chargeType", "npcUnavailable", "npcPersonality", "quest68", "prior0combinedSkill", "prior0BAR", "prior0attacks", "prior0swings", "prior0initiative", "prior0toHit", "prior0damageFlat", "prior0damagePercent", "prior0damageDice", "prior0attackModes", "prior0effect0", "prior0effect1", "prior0effect2", "prior0effect3", "prior1combinedSkill", "prior1BAR", "prior1attacks", "prior1swings", "prior1initiative", "prior1toHit", "prior1damageFlat", "prior1damagePercent", "prior1damageDice", "prior1attackModes", "prior1effect0", "prior1effect1", "prior1effect2", "prior1effect3"},
		Out: []string{"hand0type", "hand0enabled", "hand0weaponSkill", "hand0combatSkill", "hand0combinedSkill", "hand0BAR", "hand0attacks", "hand0swings", "hand0initiative", "hand0toHit", "hand0damageFlat", "hand0damagePercent", "hand0damageDice", "hand0attackModes", "hand0effect0", "hand0effect1", "hand0effect2", "hand0effect3", "hand1type", "hand1enabled", "hand1weaponSkill", "hand1combatSkill", "hand1combinedSkill", "hand1BAR", "hand1attacks", "hand1swings", "hand1initiative", "hand1toHit", "hand1damageFlat", "hand1damagePercent", "hand1damageDice", "hand1attackModes", "hand1effect0", "hand1effect1", "hand1effect2", "hand1effect3", "dualWeapons"}, F: calcAttacksNode}
}
