package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

var spellAreaConstants, spellAreaConstantsErr = loadSpellAreaConstants()

// Scalar geometry constants are read from the player's executable, using the
// same PE address mapping as exe.go. No original table is embedded here.
func loadSpellAreaConstants() (map[uint32]float64, error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := make(map[uint32]float64)
	for _, va := range []uint32{0x5ebc40, 0x5ebc64, 0x5ec3fc, 0x5ed1e8, 0x5ec2a8, 0x5ec29c, 0x5ec35c, 0x5ec014, 0x5ec318, 0x5ecd88} {
		off, err := peOffset(b, va)
		if err != nil {
			return nil, err
		}
		size := 4
		if va == 0x5ec318 {
			size = 8
		}
		if off+size > len(b) {
			return nil, fmt.Errorf("spell area: truncated scalar at %#x", va)
		}
		if size == 8 {
			out[va] = math.Float64frombits(binary.LittleEndian.Uint64(b[off:]))
		} else {
			out[va] = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[off:])))
		}
	}
	return out, nil
}

// The arena describes eight party members, one monster (ID 123), one group
// (ID 456), and an empty collision world. Original child functions traverse
// those records. Coordinates are IEEE754 float32 bit patterns.
func spellAreaTargetSelectionNode(in []int64, _ Rand) []int64 {
	if len(in) != len(spellAreaInputs) {
		panic("spell area: input arity")
	}
	if spellAreaConstantsErr != nil {
		panic(spellAreaConstantsErr)
	}
	v := make(map[string]int64, len(in))
	for i, n := range spellAreaInputs {
		v[n] = in[i]
	}
	f := func(n string) float64 { return float64(math.Float32frombits(uint32(v[n]))) }
	round := func(x float64) float64 { return float64(float32(x)) }
	vec := func(prefix string) [3]float64 {
		return [3]float64{f(prefix + "XBits"), f(prefix + "YBits"), f(prefix + "ZBits")}
	}
	dist := func(a, b [3]float64) float64 {
		dx, dy, dz := a[0]-b[0], a[1]-b[1], a[2]-b[2]
		return math.Sqrt(dz*dz + dy*dy + dx*dx)
	}
	c := spellAreaConstants
	source, center := vec("source"), vec("source")
	party, aim, target, monster := vec("party"), vec("partyAim"), vec("target"), vec("monster")
	special := v["sourceSpecial"] != 0 || v["sourceEnvironmental"] != 0
	sk := v["sourceKind"]
	isCharacter := func(indirect bool) bool { return sk == 1 || (indirect && sk == 3 && v["sourceCharacter"] != -1) }
	isMonster := func(indirect bool) bool { return sk == 2 || (indirect && sk == 3 && v["sourceMonster"] != -1) }
	if sk == 1 {
		source, center = aim, party
	}
	if sk == 2 {
		source, center = monster, monster
		source[1] = round(source[1] + f("monsterEyeHeightBits"))
		if v["monsterMagicMouth"] != 0 && v["sourceUseCenter"] == 0 && v["spellID"] != 119 {
			// Sprite caster vertex is the scaled bounding-box mid-height:
			// original virtual dispatch 0x4c7bd3 -> 0x4c3e60.
			lo := round(f("monsterBoundsMinYBits") * f("monsterModelScaleBits"))
			hi := round(f("monsterBoundsMaxYBits") * f("monsterModelScaleBits"))
			source[1] = round(monster[1] + (hi-lo)*c[0x5ecd88])
		}
	}
	monsterPartySight := v["monsterPartySight2"]
	if sk == 2 && v["monsterMagicMouth"] != 0 && v["sourceUseCenter"] == 0 && v["spellID"] != 119 {
		monsterPartySight = v["monsterPartySight3"]
	}
	monsters := make([]int64, int(v["monsterInitialCount"]))
	for i := range monsters {
		monsters[i] = 123
	}
	characters := make([]int64, int(v["characterInitialCount"]))
	for i := range characters {
		characters[i] = 6
	}
	faction := int64(0)
	partyHit := false
	validMonster := v["monsterPresent"] != 0 && v["monsterHP"] != 0 && v["monsterRemoved"] == 0 && v["monsterImmune"] == 0
	monsterEye := monster
	monsterEye[1] = round(monsterEye[1] + f("monsterEyeHeightBits"))
	azimuth := func(a, b [3]float64) float64 {
		x, z := b[0]-a[0], b[2]-a[2]
		if z == 0 {
			if x > 0 {
				return c[0x5ec3fc]
			}
			return c[0x5ed1e8]
		}
		return round(math.Atan2(x, z))
	}
	elevation := func(a, b [3]float64) float64 {
		dy := round(b[1] - a[1])
		d := round(dist(a, b))
		if d == 0 {
			return 0
		}
		return round(round(math.Acos(dy/d)) - c[0x5ec2a8])
	}
	wrap := func(x float64) float64 {
		x += c[0x5ec318]
		for x < 0 {
			x += c[0x5ec014]
		}
		for x >= c[0x5ec014] {
			x -= c[0x5ec014]
		}
		return x
	}
	cone := func(p [3]float64, radius, sourceRadius float64) bool {
		if math.Floor(dist(p, source)-radius-sourceRadius) > c[0x5ec35c] {
			return false
		}
		a, b := azimuth(source, target), azimuth(source, p)
		if round(wrap(round(a-b))) >= c[0x5ec29c] && wrap(round(b-a)) >= c[0x5ec29c] {
			return false
		}
		a, b = elevation(source, target), elevation(source, p)
		return round(wrap(round(a-b))) < c[0x5ec29c] || wrap(round(b-a)) < c[0x5ec29c]
	}
	radius := round((f("areaPerPowerBits")*float64(uint32(v["powerLevel"])) + f("areaBaseBits")) * c[0x5ebc40])
	addAreaMonster := func(origin [3]float64, r float64) {
		if r != 0 && validMonster && (faction == 3 || faction == v["monsterFaction"]) && dist(origin, monster)-f("monsterRadiusBits") <= r {
			monsters = append(monsters, 123)
		}
	}
	addConeMonster := func() {
		sr := 0.0
		if sk == 1 {
			sr = f("partyRadiusBits")
		}
		if sk == 2 {
			sr = f("monsterRadiusBits")
		}
		// Character cone visibility uses monster +0x28c; the fixture initially
		// has no cached visibility. Monster self visibility has no pair record.
		visible := sk == 3 || (sk == 1 && v["monsterRayVisible"] != 0)
		if validMonster && visible && (faction == 3 || faction == v["monsterFaction"]) && cone(monsterEye, f("monsterRadiusBits"), sr) {
			monsters = append(monsters, 123)
		}
	}
	fallback := func() bool {
		return len(monsters) == 0 && (v["battleObject"] == 0 || v["battleSuppressFallback"] == 0) && sk == 1 && faction == 1 && !(v["monsterVisible"] != 0 && v["monsterFaction"] == 1 && v["monsterHP"] > 0 && v["monsterCondition"] < 14)
	}
	typ, id := v["spellTargetType"], v["spellID"]
	// Diagnostic returns, including 0x4fdaf6, precede the final monster
	// filter and preserve existing output lists.
	returned := false
	if sk == 2 && v["monsterFaction"] != 1 && v["monsterFaction"] != 2 && (typ == 2 || typ == 5 || typ == 6) {
		typ = -1
		returned = true
	}
	switch typ {
	case 0, 1, 3:
		if v["targetKind"] == 1 || (typ == 1 && v["targetKind"] == 9) {
			characters = append(characters, v["targetCharacter"])
		} else if v["targetKind"] == 3 {
			monsters = append(monsters, v["targetMonster"])
		} else if typ != 0 {
			returned = true
		}
	case 4:
		if v["targetKind"] == 2 {
			partyHit = true
		} else if v["targetKind"] == 4 && v["groupMemberCount"] > 0 && v["groupFaction"] == v["monsterFaction"] {
			monsters = append(monsters, 123)
		} else if v["targetKind"] != 4 {
			returned = true
		}
	case 2:
		faction = 2
		if sk == 1 {
			partyHit = true
			if radius > 0 {
				radius = round(radius + f("partyRadiusBits"))
			}
		} else if sk == 2 {
			faction = v["monsterFaction"]
			partyHit = faction == 2 && dist(center, party) <= radius && monsterPartySight != 0
			radius = round(radius + f("monsterRadiusBits"))
		} else {
			partyHit = dist(center, party) <= radius
		}
		addAreaMonster(center, radius)
	case 5:
		if sk == 1 {
			faction = 1
		} else if sk == 2 {
			if v["monsterFaction"] == 2 {
				faction = 1
			} else {
				faction = 2
				partyHit = cone(aim, f("partyRadiusBits"), f("monsterRadiusBits")) && monsterPartySight != 0
			}
		} else {
			faction = 3
			if special {
				if v["sourceCharacter"] != -1 || (v["sourceMonster"] != -1 && v["monsterFaction"] == 2) {
					faction = 2
				} else if v["sourceMonster"] != -1 {
					faction = 1
				}
			}
			partyHit = faction != 1 && cone(aim, f("partyRadiusBits"), 0)
		}
		addConeMonster()
		if fallback() {
			faction = 3
			addConeMonster()
		}
	case 6, 7:
		if typ == 7 {
			lengths := map[int64]float64{-1: 0, 0: 2, 1: 4, 2: 25, 3: 50} // 0x51aa4e..0x51aa67.
			radius = round(lengths[v["spellRange"]] * c[0x5ebc40])
			if sk == 1 {
				radius = round(radius + f("partySpellReachBits"))
			}
			if sk == 2 {
				radius = round(radius + f("monsterSpellReachBits"))
			}
			target = source
		}
		if isCharacter(true) || (isMonster(true) && v["monsterFaction"] == 2) {
			faction = 1
			if special {
				faction = 2
			}
		} else if isMonster(true) {
			faction = 2
			if special {
				faction = 1
			}
		} else {
			faction = 3
		}
		origin := center
		if typ == 6 {
			origin = target
			origin[1] = round(origin[1] - c[0x5ebc64])
		}
		partyHit = faction != 1 && dist(origin, party) <= radius
		addAreaMonster(origin, radius)
		if fallback() {
			faction = 3
			addAreaMonster(origin, radius)
		}
	}
	if partyHit && id != 22 && id != 77 { // Party exclusions: 0x4fddcd/0x4fddd6.
		harmful := id != 3 && id != 41 && typ >= 3 && typ <= 7 // 0x5474da..0x5474f7.
		for i := 0; i < 8; i++ {
			p := fmt.Sprintf("character%d", i)
			if v[p+"present"] != 0 && v[p+"HP"] != 0 && v[p+"condition"] < 18 && (v[p+"magicProtection"] == 0 || !harmful || faction == 3) {
				characters = append(characters, int64(i))
			}
		}
	}
	// Replay list storage as well as logical results. The original grows to
	// exactly count+1 (0x4addf0, 0x4fde5d); the emulator's allocator aligns
	// each allocation to eight bytes at 0x50000000 (re/emu/emu.py).
	memory := map[int64]int64{0x20000100: 123, 0x20000140: 6}
	heap := int64(0x50000000)
	buildList := func(ids []int64, initial, capacity, ptr int64) (int64, int64) {
		for i := initial; i < int64(len(ids)); i++ {
			if i+1 > capacity {
				next := heap
				heap += (4*(i+1) + 7) &^ 7
				for j := int64(0); j < i; j++ {
					memory[next+4*j] = memory[ptr+4*j]
				}
				ptr, capacity = next, i+1
			}
			memory[ptr+4*i] = ids[i]
		}
		return capacity, ptr
	}
	mc, mp := buildList(monsters, v["monsterInitialCount"], v["monsterCapacity"], 0x20000100)
	cc, cp := buildList(characters, v["characterInitialCount"], v["characterCapacity"], 0x20000140)
	// 0x501b70 removes invalid or spell-incompatible monsters in place;
	// the now-unused backing slots retain the previous IDs.
	species := v["monsterSpecies"]
	compatible := validMonster && (id != 22 || species == 20 || species == 21) && (id != 77 || species == 20 || species == 21 || species == 28 || v["monsterSpecialStatus"] != 0) && (id != 129 || species == 20)
	if !returned && !compatible {
		monsters = nil
	}
	out := []int64{int64(len(monsters)), int64(len(characters))}
	for _, ptr := range []int64{0x20000100, 0x20000140} {
		for i := int64(0); i < 8; i++ {
			out = append(out, memory[ptr+4*i])
		}
	}
	for _, x := range target {
		out = append(out, int64(math.Float32bits(float32(x))))
	}
	out = append(out, mc, cc, mp, cp)
	for i := int64(0); i < 64; i++ {
		out = append(out, memory[0x50000000+4*i])
	}

	return out
}

var spellAreaInputs = []string{"sourceXBits", "sourceYBits", "sourceZBits", "targetXBits", "targetYBits", "targetZBits", "partyXBits", "partyYBits", "partyZBits", "partyAimXBits", "partyAimYBits", "partyAimZBits", "partyRadiusBits", "spellID", "spellTargetType", "areaBaseBits", "areaPerPowerBits", "spellRange", "powerLevel", "sourceKind", "sourceCharacter", "sourceMonster", "sourceSpecial", "sourceEnvironmental", "sourceUseCenter", "targetKind", "targetCharacter", "targetMonster", "targetGroup", "ignoreLineOfSight", "monsterPresent", "monsterVisible", "monsterFaction", "monsterHP", "monsterRemoved", "monsterCondition", "monsterMagicMouth", "monsterPartySight2", "monsterPartySight3", "monsterImmune", "monsterSpecies", "monsterSpecialStatus", "monsterXBits", "monsterYBits", "monsterZBits", "monsterRadiusBits", "monsterEyeHeightBits", "monsterSpellReachBits", "partySpellReachBits", "groupFaction", "groupMemberCount", "character0present", "character0HP", "character0condition", "character0magicProtection", "character1present", "character1HP", "character1condition", "character1magicProtection", "character2present", "character2HP", "character2condition", "character2magicProtection", "character3present", "character3HP", "character3condition", "character3magicProtection", "character4present", "character4HP", "character4condition", "character4magicProtection", "character5present", "character5HP", "character5condition", "character5magicProtection", "character6present", "character6HP", "character6condition", "character6magicProtection", "character7present", "character7HP", "character7condition", "character7magicProtection", "battleObject", "battleSuppressFallback", "monsterRayVisible", "monsterInitialCount", "monsterCapacity", "characterInitialCount", "characterCapacity", "monsterModelScaleBits", "monsterBoundsMinYBits", "monsterBoundsMaxYBits"}

func init() {
	Nodes["004fd030"] = Node{Addr: "004fd030", In: spellAreaInputs, Out: []string{"monsterCount", "characterCount", "monster0", "monster1", "monster2", "monster3", "monster4", "monster5", "monster6", "monster7", "character0", "character1", "character2", "character3", "character4", "character5", "character6", "character7", "resultTargetXBits", "resultTargetYBits", "resultTargetZBits", "monsterCapacityAfter", "characterCapacityAfter", "monsterBufferAfter", "characterBufferAfter", "allocatedListWord0", "allocatedListWord1", "allocatedListWord2", "allocatedListWord3", "allocatedListWord4", "allocatedListWord5", "allocatedListWord6", "allocatedListWord7", "allocatedListWord8", "allocatedListWord9", "allocatedListWord10", "allocatedListWord11", "allocatedListWord12", "allocatedListWord13", "allocatedListWord14", "allocatedListWord15", "allocatedListWord16", "allocatedListWord17", "allocatedListWord18", "allocatedListWord19", "allocatedListWord20", "allocatedListWord21", "allocatedListWord22", "allocatedListWord23", "allocatedListWord24", "allocatedListWord25", "allocatedListWord26", "allocatedListWord27", "allocatedListWord28", "allocatedListWord29", "allocatedListWord30", "allocatedListWord31", "allocatedListWord32", "allocatedListWord33", "allocatedListWord34", "allocatedListWord35", "allocatedListWord36", "allocatedListWord37", "allocatedListWord38", "allocatedListWord39", "allocatedListWord40", "allocatedListWord41", "allocatedListWord42", "allocatedListWord43", "allocatedListWord44", "allocatedListWord45", "allocatedListWord46", "allocatedListWord47", "allocatedListWord48", "allocatedListWord49", "allocatedListWord50", "allocatedListWord51", "allocatedListWord52", "allocatedListWord53", "allocatedListWord54", "allocatedListWord55", "allocatedListWord56", "allocatedListWord57", "allocatedListWord58", "allocatedListWord59", "allocatedListWord60", "allocatedListWord61", "allocatedListWord62", "allocatedListWord63"}, F: spellAreaTargetSelectionNode}
}
