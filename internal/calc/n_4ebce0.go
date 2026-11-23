package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

var breathPrepareScalars, breathPrepareScalarError = loadBreathPrepareScalars()

func loadBreathPrepareScalars() (map[uint32]float64, error) {
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
	// Target-distance allowance (0x5193f2) and radians-to-degrees
	// conversion (0x555704) come from the player's executable.
	for _, va := range []uint32{0x5ebb38, 0x5ee798, 0x603ac8} {
		off, err := peOffset(b, va)
		if err != nil {
			return nil, err
		}
		if off+4 > len(b) {
			return nil, fmt.Errorf("breath prepare: truncated scalar %#x", va)
		}
		out[va] = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[off:])))
	}
	return out, nil
}

func breathTargetPrepareNode(in []int64, r Rand) []int64 {
	if len(in) != len(breathPrepareInputs) {
		panic("breath prepare: input arity")
	}
	if breathPrepareScalarError != nil {
		panic(breathPrepareScalarError)
	}
	v := make(map[string]int64, len(in))
	for i, n := range breathPrepareInputs {
		v[n] = in[i]
	}
	out := make([]int64, len(breathPrepareOutputs))
	out[1] = v["effectBefore"]
	for i := 0; i < 8; i++ {
		out[14+i] = v[fmt.Sprintf("character%dFacingBefore", i)]
	}
	f := func(n string) float64 { return float64(math.Float32frombits(uint32(v[n]))) }
	round := func(x float64) float64 { return float64(float32(x)) }
	vec := func(prefix string) [3]float64 {
		return [3]float64{f(prefix + "XBits"), f(prefix + "YBits"), f(prefix + "ZBits")}
	}
	target, origin := vec("target"), vec("partyAim")
	// The action queue contains action 2 (breath) in every character slot;
	// 0x53a8d0 resolves it to spell 119. A coordinate target is checked by
	// 0x519180 before 0x4fd030 selects the actual monster list.
	if v["targetKind"] == 6 {
		dx, dy, dz := target[0]-origin[0], target[1]-origin[1], target[2]-origin[2]
		if math.Sqrt(dx*dx+dy*dy+dz*dz)-f("partyRadiusBits") > spellAreaConstants[0x5ec35c]+breathPrepareScalars[0x5ebb38] {
			return out
		}
	}
	// In exploration mode, 0x5194e0 requires the monster's range cache,
	// a clear breath ray and range class zero (two distance units).
	// The combat/interface mode globals are explicitly zero in the arena.
	if v["targetKind"] == 3 {
		if v["monsterRangeCacheReady"] != 1 || v["monsterRayVisible"] == 0 {
			return out
		}
		monster := vec("monster")
		dx, dy, dz := monster[0]-origin[0], monster[1]-round(origin[1]-breathPrepareScalars[0x603ac8]), monster[2]-origin[2]
		if math.Sqrt(dx*dx+dy*dy+dz*dz)-f("monsterSpellReachBits")-f("partySpellReachBits") > 2*spellAreaConstants[0x5ebc40] { // 0x5196cf.
			return out
		}
	}
	child := make([]int64, len(spellAreaInputs))
	for i, n := range spellAreaInputs {
		child[i] = v[n]
	}
	for i, n := range spellAreaInputs {
		switch n {
		case "spellID":
			child[i] = 119 // 0x4ebd91.
		case "sourceKind":
			child[i] = 1 // Character descriptor, 0x53be00.
		case "sourceCharacter":
			child[i] = v["characterIndex"]
		case "sourceMonster":
			child[i] = -1
		case "battleObject":
			child[i] = 1
		case "monsterCapacity", "characterCapacity":
			child[i] = 5 // 0x4ebd19/0x4ebd3e.
		}
	}
	selected := spellAreaTargetSelectionNode(child, r)
	if selected[0] == 0 {
		return out
	}
	out[0] = 1
	// Replay the fixture's allocator, rather than returning an arena pointer.
	// Two five-entry buffers (0x4ebd04/0x4ebd24) precede the cached effect
	// copies. Allocation sizes are constructor requests in 0x4ab580,
	// 0x457390, 0x456230, 0x452470 and 0x44edf0. The first copy has no
	// animation frames, so 0x4adc06 requests the default copy.
	heap := int64(0x50000000) // re/emu/emu.py allocator origin.
	alloc := func(size int64) int64 { p := heap; heap += (size + 7) &^ 7; return p }
	alloc(20)
	alloc(20)
	for count := int64(6); count <= selected[1]; count++ {
		alloc(count * 4)
	}
	for _, size := range []int64{504, 96, 120, 20, 312, 4, 100, 8} {
		alloc(size)
	}
	out[1] = alloc(504)
	for i, x := range origin {
		out[2+i] = int64(math.Float32bits(float32(x)))
	}
	direction := [3]float64{}
	for i := range direction {
		direction[i] = round(target[i] - origin[i])
	}
	length := math.Sqrt(direction[0]*direction[0] + direction[1]*direction[1] + direction[2]*direction[2])
	if length != 0 {
		for i := range direction {
			direction[i] = round(direction[i] / length)
		}
	}
	yaw := math.Atan2(direction[0], direction[2])
	cy, sy := round(math.Cos(yaw)), round(math.Sin(yaw))
	matrix := [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1} // Identity, 0x467310.
	if yaw != 0 {
		matrix = [9]float64{cy, 0, sy, 0, 1, 0, -sy, 0, cy}
	}
	y, z := direction[1], round(matrix[5]*direction[1]+matrix[2]*direction[0]+matrix[8]*direction[2])
	pitch := -math.Atan2(y, z)
	if pitch != 0 {
		cp, sp := round(math.Cos(pitch)), round(math.Sin(pitch))
		rotation := [9]float64{1, 0, 0, 0, cp, -sp, 0, sp, cp}
		var product [9]float64
		for row := 0; row < 3; row++ {
			for col := 0; col < 3; col++ {
				product[row*3+col] = round(matrix[row*3]*rotation[col] + matrix[row*3+2]*rotation[6+col] + matrix[row*3+1]*rotation[3+col])
			}
		}
		matrix = product
	}
	for i, x := range matrix {
		out[5+i] = int64(math.Float32bits(float32(x)))
	}
	if v["targetKind"] == 6 || v["targetKind"] == 3 {
		facingTarget := target
		if v["targetKind"] == 3 {
			facingTarget = vec("monster")
		}
		dx, dz := facingTarget[0]-origin[0], facingTarget[2]-origin[2]
		angle := round(math.Atan2(dx, dz))
		if dz == 0 {
			angle = spellAreaConstants[0x5ed1e8]
			if dx > 0 {
				angle = spellAreaConstants[0x5ec3fc]
			}
		}
		angle += spellAreaConstants[0x5ec318]
		for angle < 0 {
			angle += spellAreaConstants[0x5ec014]
		}
		for angle >= spellAreaConstants[0x5ec014] {
			angle -= spellAreaConstants[0x5ec014]
		}
		degrees := int64(angle*breathPrepareScalars[0x5ee798]) - v["partyHeading"]
		if degrees < 0 {
			degrees += 360
		}
		out[14+int(v["characterIndex"])] = ((degrees + 45) % 360) / 90 // 0x555712..0x555768.
	}
	return out
}

func init() {
	Nodes["004ebce0"] = Node{Addr: "004ebce0", In: breathPrepareInputs, Out: breathPrepareOutputs, F: breathTargetPrepareNode}
}

var breathPrepareInputs = []string{"characterIndex", "targetXBits", "targetYBits", "targetZBits", "partyXBits", "partyYBits", "partyZBits", "partyAimXBits", "partyAimYBits", "partyAimZBits", "partyRadiusBits", "spellTargetType", "areaBaseBits", "areaPerPowerBits", "spellRange", "targetKind", "targetCharacter", "targetMonster", "targetGroup", "monsterPresent", "monsterVisible", "monsterFaction", "monsterHP", "monsterRemoved", "monsterCondition", "monsterMagicMouth", "monsterPartySight2", "monsterPartySight3", "monsterImmune", "monsterSpecies", "monsterSpecialStatus", "monsterXBits", "monsterYBits", "monsterZBits", "monsterRadiusBits", "monsterEyeHeightBits", "monsterSpellReachBits", "partySpellReachBits", "groupFaction", "groupMemberCount", "character0present", "character0HP", "character0condition", "character0magicProtection", "character1present", "character1HP", "character1condition", "character1magicProtection", "character2present", "character2HP", "character2condition", "character2magicProtection", "character3present", "character3HP", "character3condition", "character3magicProtection", "character4present", "character4HP", "character4condition", "character4magicProtection", "character5present", "character5HP", "character5condition", "character5magicProtection", "character6present", "character6HP", "character6condition", "character6magicProtection", "character7present", "character7HP", "character7condition", "character7magicProtection", "battleSuppressFallback", "monsterRayVisible", "monsterModelScaleBits", "monsterBoundsMinYBits", "monsterBoundsMaxYBits", "alternateMessage", "effectBefore", "partyHeading", "character0FacingBefore", "character1FacingBefore", "character2FacingBefore", "character3FacingBefore", "character4FacingBefore", "character5FacingBefore", "character6FacingBefore", "character7FacingBefore", "monsterRangeCacheReady"}
var breathPrepareOutputs = []string{"prepared", "effect", "effectOriginXBits", "effectOriginYBits", "effectOriginZBits", "effectOrientation0Bits", "effectOrientation1Bits", "effectOrientation2Bits", "effectOrientation3Bits", "effectOrientation4Bits", "effectOrientation5Bits", "effectOrientation6Bits", "effectOrientation7Bits", "effectOrientation8Bits", "character0FacingAfter", "character1FacingAfter", "character2FacingAfter", "character3FacingAfter", "character4FacingAfter", "character5FacingAfter", "character6FacingAfter", "character7FacingAfter"}
