package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

// monsterActivationStatsNode reproduces activation of the resident model described
// by 004e3c70.json. Model resources and SR imports are supplied by that case.
func monsterActivationStatsNode(in []int64, r Rand) []int64 {
	if len(in) != 262 {
		panic("monster activation: expected 262 inputs")
	}
	out := []int64{in[0], in[11], in[4], in[5], in[13], in[29], in[25], in[26], in[27], in[28], in[16], in[37], in[3], in[38], in[39], in[40], in[12], in[47], in[48], in[49]}
	out = append(out, in[50:153]...)  // Existing 103-byte condition modifier cache.
	out = append(out, in[158:163]...) // Existing five effective attributes.
	out = append(out, in[163:165]...) // Existing HP/SP recovery coefficients.
	// 0x4e3ca5: already enabled monsters retain every observed field.
	if in[0] != 0 {
		return out
	}
	out[0] = 1
	out[1] = int64(uint32(in[11]) + 1) // 0x4e3fba: unsigned counter wrap.
	if in[9] != 0 {
		out[4] = int64(uint32(in[13]) | 0x10000000)
	} // 0x4e4039.
	// 0x4c6350 collects three primitive types in every resident cycle/frame.
	// The resident case supports cycles 0, 1, 7, 13 and 25 (other cycles are absent).
	parts := int(in[30]) * (3 + int(in[260]) + int(in[261])) * 3
	align := func(n int) int64 { return int64((n + 7) &^ 7) }
	// 0x4c6a50 starts its collection with five pointers (20 bytes); 0x4c6536
	// grows capacity by one pointer. The emulator's allocator aligns to 8 bytes.
	collection := align(5 * 4)
	for n := 6; n <= parts; n++ {
		collection += align(n * 4)
	}
	allocation := collection
	if in[3] != 0 {
		monsterActivationRebuildRules(in, out)
		// 0x4c5f00's position update collects from an initially empty list.
		for n := 1; n <= parts; n++ {
			allocation += align(n * 4)
		}
		out[6], out[7], out[8], out[9] = 1, 0, 1, 0 // 0x51b420 initializes primary and secondary capabilities.
		// 0x51b455..0x51b4d2: enabled attack groups contribute their maximum count.
		most := int64(-1)
		for i := 18; i <= 22; i += 2 {
			if in[i] != 0 && in[i+1] > most {
				most = in[i+1]
			}
		}
		// 0x4c7960 and 0x4712d0 resolve event codes 5 (multiple) and 6 (ranged).
		hasEvent := func(code int64) bool { return in[34] == code || in[35] == code }
		if most >= 2 && hasEvent(5) {
			out[7] = 1
		}
		if in[24] > 0 && in[261] != 0 && hasEvent(6) {
			out[9] = 1
		}
		out[17] = 0                                          // 0x4e3e94: identical MEMORYSTATUS readings have zero delta.
		out[11] = int64(uint32(in[8])*65536 + uint32(in[1])) // 0x4e3e6d.
		out[12] = 0
		out[13] = 0
		out[14] = 0
		out[15] = 0
		if in[14] != 0 {
			out[18], out[19] = 0, 1
		} // 0x4e3dad and 0x4e3dba.
		if in[14] == 0 {
			out[15] = int64(r.R(int(in[30]))) // 0x4e3e09.
			out[14] = 1                       // 0x4a8460 selects idle mode 3; frame selection uses resident cycle 1.
			if in[7] == 0 {
				out[13] = 1
			}
		}
	}
	appearances := int64(int32(in[15])) // 0x4e3fbf: signed comparison.
	if appearances > 1 {
		// 0x4e3fcf..0x4e3fe7 uses wrapping signed 32-bit product and signed
		// division toward zero; only the upper damage-appearance bound is clamped.
		numerator := int32((uint32(in[4]) - uint32(in[5])) * uint32(in[15]))
		stage := int64(numerator) / int64(int32(in[4]))
		if stage >= appearances-1 {
			stage = appearances - 1
		}
		out[10] = stage
		allocation += collection // 0x4c6990 gathers primitives again before writing.
	}
	// 0x5e22c0 allocates the association header first; 0x50000000 is the
	// documented replay heap base (re/emu/emu.py), not a game address constant.
	out[5] = 0x50000000 + allocation
	return out
}

type monsterActivationExe struct {
	effectIndex  [38]uint8
	effectTarget [9]uint32
	recovery     [5]float64
}

var monsterActivationTables, monsterActivationTablesErr = loadMonsterActivationTables()

// Tables remain in the player's executable. peOffset is shared with exe.go;
// initialization performs the I/O before the pure node is invoked.
func loadMonsterActivationTables() (monsterActivationExe, error) {
	var t monsterActivationExe
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("monster activation: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	read := func(va uint32, count int) ([]byte, error) {
		off, err := peOffset(b, va)
		if err != nil {
			return nil, err
		}
		if off < 0 || count > len(b)-off {
			return nil, fmt.Errorf("monster activation: truncated table %#x", va)
		}
		return b[off : off+count], nil
	}
	// 0x50edf8 / 0x50ee00: effect codes 8..45 select these two tables.
	row, err := read(0x50ef28, len(t.effectIndex))
	if err != nil {
		return t, err
	}
	copy(t.effectIndex[:], row)
	row, err = read(0x50ef04, 4*len(t.effectTarget))
	if err != nil {
		return t, err
	}
	for i := range t.effectTarget {
		t.effectTarget[i] = binary.LittleEndian.Uint32(row[4*i:])
	}
	for _, index := range t.effectIndex {
		if int(index) >= len(t.effectTarget) {
			return t, fmt.Errorf("monster activation: invalid effect destination")
		}
	}
	// 0x502c50 reads the HP scale, intercept, common scale, SP scale and
	// recovery suppression multiplier, respectively, as float32 constants.
	for i, va := range []uint32{0x5ebca4, 0x5ed4f0, 0x5ed7ec, 0x5ec390, 0x5ec3b8} {
		row, err := read(va, 4)
		if err != nil {
			return t, err
		}
		t.recovery[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(row)))
	}
	return t, nil
}

func monsterActivationRebuildRules(in, out []int64) {
	if statusRebuildTablesErr != nil {
		panic(statusRebuildTablesErr)
	}
	if monsterActivationTablesErr != nil {
		panic(monsterActivationTablesErr)
	}
	var modifiers [103]uint8 // 0x50e8e1..0x50e8f4 clears the entire cache.
	add := func(offset, value int) { modifiers[offset] += uint8(value) }
	// 0x50eac0 is called with no character: racial features and character
	// controller status do not apply. Status 19 requires a character pointer
	// and has no successful monster replay; its case duration is zero.
	for i := 3; i < 19; i++ {
		if in[166+i] == 0 {
			continue
		}
		switch statusRebuildTables.dispatch[i-3] {
		case 0x50eafb:
			add(1, -2)
			add(0x4b, -2)
		case 0x50eb11:
			add(1, -5)
			for j := 0xc; j < 0x3c; j++ {
				add(j, -20)
			}
		case 0x50eb4a:
			add(1, -5)
			add(0x4b, -4)
		case 0x50eb60:
			add(1, -3)
			add(0x4b, -2)
		case 0x50eb76:
			add(0x11, -50)
		case 0x50eb7f:
			add(1, -2)
			add(0x4b, -2)
			add(8, int(in[186]))
		case 0x50eba1:
			add(0x12, -50)
			modifiers[0x45] = 1
		case 0x50ebe1:
			add(0x10, -50)
			modifiers[0x45] = 1
		case 0x50ebe5:
			modifiers[0x45] = 1
		case 0x50ebea:
			add(0xd, -50)
		}
	}
	// 0x517f60: numerator wraps to u32 before the unsigned division and
	// the result wraps to u8. Effect multipliers are full u32, unlike buffs.
	scale := func(value uint8, multiplier int64) uint8 {
		numerator := uint32(value)*(uint32(multiplier)+100) + 50
		return uint8(numerator / 100)
	}
	// 0x50ecc0: records 5,6,7 contribute to the monster's modifier cache.
	for i := 5; i < 8; i++ {
		at := 187 + 3*i
		if in[at+2] == 0 {
			continue
		}
		power, multiplier := uint8(in[at]), in[at+1]
		switch i {
		case 5:
			add(0x11, int(scale(power*10, multiplier)))
		case 6:
			amount := scale(power*5, multiplier)
			for j := 0xc; j < 0x13; j++ {
				add(j, int(amount))
			}
		case 7:
			add(6, int(scale(power*8, multiplier)))
			add(5, int(scale(power, multiplier)))
		}
	}
	// 0x50edc0 processes twelve active effect records in their stored order.
	for i := 0; i < 12; i++ {
		at := 211 + 4*i
		if in[at] == 0 || uint32(in[at+1])-8 >= 38 {
			continue
		}
		target := monsterActivationTables.effectTarget[monsterActivationTables.effectIndex[in[at+1]-8]]
		power, multiplier := uint8(in[at+2]), in[at+3]
		switch target {
		case 0x50ee07:
			add(1, int(scale(uint8((uint32(power)+1)/2), multiplier)))
		case 0x50ee39:
			add(5, int(scale(uint8((uint32(power)+1)/2), multiplier)))
		case 0x50ee6b:
			add(7, int(scale(power*4+7, multiplier)))
		case 0x50ee90:
			modifiers[0x46] = 1
		case 0x50ee96:
			modifiers[0x47] = scale(power, multiplier) * 2
		case 0x50eeb3:
			modifiers[0x48] = scale(power, multiplier)
		case 0x50eece:
			modifiers[0x49] = scale((power+5)*5, multiplier)
		case 0x50eeef:
			modifiers[0x4a] = 1
		case 0x50eef3: // This code has no monster modifier.
		default:
			panic(fmt.Sprintf("monster activation: unknown effect destination %#x", target))
		}
	}
	for i, v := range modifiers {
		out[20+i] = int64(v)
	}
	// 0x4e5d00 / notes/re/combat-2.md:120: five template attributes,
	// signed byte modifiers, and the inclusive 1..125 effective range.
	for i, offset := range []int{0, 1, 4, 5, 6} {
		out[123+i] = min(int64(125), max(int64(1), in[153+i]+int64(int8(modifiers[0xc+offset]))))
	}
	// 0x502c50 treats both maxima as zero-extended u32 and the two cache
	// corrections as i8. Final stores round to float32; expose exact bits.
	t := monsterActivationTables.recovery
	hp := (float64(uint32(in[4]))*t[0]+t[1])*t[2] + float64(int8(modifiers[9]))
	sp := (float64(uint32(in[165]))*t[3]+t[1])*t[2] + float64(int8(modifiers[10]))
	if modifiers[0x42] != 0 {
		hp *= t[4]
	}
	if modifiers[0x43] != 0 {
		sp *= t[4]
	}
	out[128] = int64(math.Float32bits(float32(hp)))
	out[129] = int64(math.Float32bits(float32(sp)))
}

func init() {
	Nodes["004e3c70"] = Node{Addr: "004e3c70",
		In: []string{
			"monsterEnabled", "monsterId", "species", "modelRefreshNeeded",
			"maxHP", "HP", "headingBits", "idleAnimationSuppressed",
			"templateLevel", "templateSpecialDisplayFlag", "templateNpcIndex", "activeMonsterCount",
			"totalModelActivationMemory", "modelFlags", "unbornCount", "damageAppearanceCount",
			"damageAppearanceBefore", "activationDiagnostics", "attack1Enabled", "attack1Count",
			"attack2Enabled", "attack2Count", "attack3Enabled", "attack3Count",
			"rangedAttackCount", "primaryAttackBefore", "multipleAttackBefore", "secondaryAttackBefore",
			"rangedAttackBefore", "associationBefore", "animationFrames", "cameraXBits",
			"cameraYBits", "cameraZBits", "multipleAttackEventCode", "rangedAttackEventCode",
			"selectedTemplatePointer", "modelLevelIdentityBefore", "animationRunningBefore", "animationCycleBefore",
			"animationFrameBefore", "monsterXBits", "monsterYBits", "monsterZBits",
			"modelXBits", "modelYBits", "modelZBits", "modelActivationMemoryBefore",
			"modelUnbornPresentationBefore", "modelUnbornActivationBefore", "conditionRuleModifierBefore[0]", "conditionRuleModifierBefore[1]",
			"conditionRuleModifierBefore[2]", "conditionRuleModifierBefore[3]", "conditionRuleModifierBefore[4]", "conditionRuleModifierBefore[5]",
			"conditionRuleModifierBefore[6]", "conditionRuleModifierBefore[7]", "conditionRuleModifierBefore[8]", "conditionRuleModifierBefore[9]",
			"conditionRuleModifierBefore[10]", "conditionRuleModifierBefore[11]", "conditionRuleModifierBefore[12]", "conditionRuleModifierBefore[13]",
			"conditionRuleModifierBefore[14]", "conditionRuleModifierBefore[15]", "conditionRuleModifierBefore[16]", "conditionRuleModifierBefore[17]",
			"conditionRuleModifierBefore[18]", "conditionRuleModifierBefore[19]", "conditionRuleModifierBefore[20]", "conditionRuleModifierBefore[21]",
			"conditionRuleModifierBefore[22]", "conditionRuleModifierBefore[23]", "conditionRuleModifierBefore[24]", "conditionRuleModifierBefore[25]",
			"conditionRuleModifierBefore[26]", "conditionRuleModifierBefore[27]", "conditionRuleModifierBefore[28]", "conditionRuleModifierBefore[29]",
			"conditionRuleModifierBefore[30]", "conditionRuleModifierBefore[31]", "conditionRuleModifierBefore[32]", "conditionRuleModifierBefore[33]",
			"conditionRuleModifierBefore[34]", "conditionRuleModifierBefore[35]", "conditionRuleModifierBefore[36]", "conditionRuleModifierBefore[37]",
			"conditionRuleModifierBefore[38]", "conditionRuleModifierBefore[39]", "conditionRuleModifierBefore[40]", "conditionRuleModifierBefore[41]",
			"conditionRuleModifierBefore[42]", "conditionRuleModifierBefore[43]", "conditionRuleModifierBefore[44]", "conditionRuleModifierBefore[45]",
			"conditionRuleModifierBefore[46]", "conditionRuleModifierBefore[47]", "conditionRuleModifierBefore[48]", "conditionRuleModifierBefore[49]",
			"conditionRuleModifierBefore[50]", "conditionRuleModifierBefore[51]", "conditionRuleModifierBefore[52]", "conditionRuleModifierBefore[53]",
			"conditionRuleModifierBefore[54]", "conditionRuleModifierBefore[55]", "conditionRuleModifierBefore[56]", "conditionRuleModifierBefore[57]",
			"conditionRuleModifierBefore[58]", "conditionRuleModifierBefore[59]", "conditionRuleModifierBefore[60]", "conditionRuleModifierBefore[61]",
			"conditionRuleModifierBefore[62]", "conditionRuleModifierBefore[63]", "conditionRuleModifierBefore[64]", "conditionRuleModifierBefore[65]",
			"conditionRuleModifierBefore[66]", "conditionRuleModifierBefore[67]", "conditionRuleModifierBefore[68]", "conditionRuleModifierBefore[69]",
			"conditionRuleModifierBefore[70]", "conditionRuleModifierBefore[71]", "conditionRuleModifierBefore[72]", "conditionRuleModifierBefore[73]",
			"conditionRuleModifierBefore[74]", "conditionRuleModifierBefore[75]", "conditionRuleModifierBefore[76]", "conditionRuleModifierBefore[77]",
			"conditionRuleModifierBefore[78]", "conditionRuleModifierBefore[79]", "conditionRuleModifierBefore[80]", "conditionRuleModifierBefore[81]",
			"conditionRuleModifierBefore[82]", "conditionRuleModifierBefore[83]", "conditionRuleModifierBefore[84]", "conditionRuleModifierBefore[85]",
			"conditionRuleModifierBefore[86]", "conditionRuleModifierBefore[87]", "conditionRuleModifierBefore[88]", "conditionRuleModifierBefore[89]",
			"conditionRuleModifierBefore[90]", "conditionRuleModifierBefore[91]", "conditionRuleModifierBefore[92]", "conditionRuleModifierBefore[93]",
			"conditionRuleModifierBefore[94]", "conditionRuleModifierBefore[95]", "conditionRuleModifierBefore[96]", "conditionRuleModifierBefore[97]",
			"conditionRuleModifierBefore[98]", "conditionRuleModifierBefore[99]", "conditionRuleModifierBefore[100]", "conditionRuleModifierBefore[101]",
			"conditionRuleModifierBefore[102]", "templateStrength", "templateIntelligence", "templateDexterity",
			"templateSpeed", "templateSenses", "effectiveStrengthBefore", "effectiveIntelligenceBefore",
			"effectiveDexterityBefore", "effectiveSpeedBefore", "effectiveSensesBefore", "hpRecoveryCoefficientBitsBefore",
			"spRecoveryCoefficientBitsBefore", "maxSP", "statusDuration[0]", "statusDuration[1]",
			"statusDuration[2]", "statusDuration[3]", "statusDuration[4]", "statusDuration[5]",
			"statusDuration[6]", "statusDuration[7]", "statusDuration[8]", "statusDuration[9]",
			"statusDuration[10]", "statusDuration[11]", "statusDuration[12]", "statusDuration[13]",
			"statusDuration[14]", "statusDuration[15]", "statusDuration[16]", "statusDuration[17]",
			"statusDuration[18]", "statusDuration[19]", "poisonStrength", "timedBuffStrength[0]",
			"timedBuffMultiplier[0]", "timedBuffDuration[0]", "timedBuffStrength[1]", "timedBuffMultiplier[1]",
			"timedBuffDuration[1]", "timedBuffStrength[2]", "timedBuffMultiplier[2]", "timedBuffDuration[2]",
			"timedBuffStrength[3]", "timedBuffMultiplier[3]", "timedBuffDuration[3]", "timedBuffStrength[4]",
			"timedBuffMultiplier[4]", "timedBuffDuration[4]", "timedBuffStrength[5]", "timedBuffMultiplier[5]",
			"timedBuffDuration[5]", "timedBuffStrength[6]", "timedBuffMultiplier[6]", "timedBuffDuration[6]",
			"timedBuffStrength[7]", "timedBuffMultiplier[7]", "timedBuffDuration[7]", "effectActive[0]",
			"effectType[0]", "effectStrength[0]", "effectMultiplier[0]", "effectActive[1]",
			"effectType[1]", "effectStrength[1]", "effectMultiplier[1]", "effectActive[2]",
			"effectType[2]", "effectStrength[2]", "effectMultiplier[2]", "effectActive[3]",
			"effectType[3]", "effectStrength[3]", "effectMultiplier[3]", "effectActive[4]",
			"effectType[4]", "effectStrength[4]", "effectMultiplier[4]", "effectActive[5]",
			"effectType[5]", "effectStrength[5]", "effectMultiplier[5]", "effectActive[6]",
			"effectType[6]", "effectStrength[6]", "effectMultiplier[6]", "effectActive[7]",
			"effectType[7]", "effectStrength[7]", "effectMultiplier[7]", "effectActive[8]",
			"effectType[8]", "effectStrength[8]", "effectMultiplier[8]", "effectActive[9]",
			"effectType[9]", "effectStrength[9]", "effectMultiplier[9]", "effectActive[10]",
			"effectType[10]", "effectStrength[10]", "effectMultiplier[10]", "effectActive[11]",
			"effectType[11]", "effectStrength[11]", "effectMultiplier[11]", "sceneMonsterIdHighWord",
			"multipleAttackCycleSupported", "rangedAttackCycleSupported",
		},
		Out: []string{
			"monsterEnabledAfter", "activeMonsterCountAfter", "maxHPAfter", "HPAfter",
			"modelFlagsAfter", "visibleMonsterAssociationList", "primaryAttackAnimation", "multipleAttackAnimation",
			"secondaryAttackAnimation", "rangedAttackAnimation", "damageAppearanceAfter", "modelLevelIdentityAfter",
			"modelRefreshAfter", "animationRunningAfter", "animationCycleAfter", "animationFrameAfter",
			"totalModelActivationMemoryAfter", "modelActivationMemoryAfter", "modelUnbornPresentationAfter", "modelUnbornActivationAfter",
			"conditionRuleModifierAfter[0]", "conditionRuleModifierAfter[1]", "conditionRuleModifierAfter[2]", "conditionRuleModifierAfter[3]",
			"conditionRuleModifierAfter[4]", "conditionRuleModifierAfter[5]", "conditionRuleModifierAfter[6]", "conditionRuleModifierAfter[7]",
			"conditionRuleModifierAfter[8]", "conditionRuleModifierAfter[9]", "conditionRuleModifierAfter[10]", "conditionRuleModifierAfter[11]",
			"conditionRuleModifierAfter[12]", "conditionRuleModifierAfter[13]", "conditionRuleModifierAfter[14]", "conditionRuleModifierAfter[15]",
			"conditionRuleModifierAfter[16]", "conditionRuleModifierAfter[17]", "conditionRuleModifierAfter[18]", "conditionRuleModifierAfter[19]",
			"conditionRuleModifierAfter[20]", "conditionRuleModifierAfter[21]", "conditionRuleModifierAfter[22]", "conditionRuleModifierAfter[23]",
			"conditionRuleModifierAfter[24]", "conditionRuleModifierAfter[25]", "conditionRuleModifierAfter[26]", "conditionRuleModifierAfter[27]",
			"conditionRuleModifierAfter[28]", "conditionRuleModifierAfter[29]", "conditionRuleModifierAfter[30]", "conditionRuleModifierAfter[31]",
			"conditionRuleModifierAfter[32]", "conditionRuleModifierAfter[33]", "conditionRuleModifierAfter[34]", "conditionRuleModifierAfter[35]",
			"conditionRuleModifierAfter[36]", "conditionRuleModifierAfter[37]", "conditionRuleModifierAfter[38]", "conditionRuleModifierAfter[39]",
			"conditionRuleModifierAfter[40]", "conditionRuleModifierAfter[41]", "conditionRuleModifierAfter[42]", "conditionRuleModifierAfter[43]",
			"conditionRuleModifierAfter[44]", "conditionRuleModifierAfter[45]", "conditionRuleModifierAfter[46]", "conditionRuleModifierAfter[47]",
			"conditionRuleModifierAfter[48]", "conditionRuleModifierAfter[49]", "conditionRuleModifierAfter[50]", "conditionRuleModifierAfter[51]",
			"conditionRuleModifierAfter[52]", "conditionRuleModifierAfter[53]", "conditionRuleModifierAfter[54]", "conditionRuleModifierAfter[55]",
			"conditionRuleModifierAfter[56]", "conditionRuleModifierAfter[57]", "conditionRuleModifierAfter[58]", "conditionRuleModifierAfter[59]",
			"conditionRuleModifierAfter[60]", "conditionRuleModifierAfter[61]", "conditionRuleModifierAfter[62]", "conditionRuleModifierAfter[63]",
			"conditionRuleModifierAfter[64]", "conditionRuleModifierAfter[65]", "conditionRuleModifierAfter[66]", "conditionRuleModifierAfter[67]",
			"conditionRuleModifierAfter[68]", "conditionRuleModifierAfter[69]", "conditionRuleModifierAfter[70]", "conditionRuleModifierAfter[71]",
			"conditionRuleModifierAfter[72]", "conditionRuleModifierAfter[73]", "conditionRuleModifierAfter[74]", "conditionRuleModifierAfter[75]",
			"conditionRuleModifierAfter[76]", "conditionRuleModifierAfter[77]", "conditionRuleModifierAfter[78]", "conditionRuleModifierAfter[79]",
			"conditionRuleModifierAfter[80]", "conditionRuleModifierAfter[81]", "conditionRuleModifierAfter[82]", "conditionRuleModifierAfter[83]",
			"conditionRuleModifierAfter[84]", "conditionRuleModifierAfter[85]", "conditionRuleModifierAfter[86]", "conditionRuleModifierAfter[87]",
			"conditionRuleModifierAfter[88]", "conditionRuleModifierAfter[89]", "conditionRuleModifierAfter[90]", "conditionRuleModifierAfter[91]",
			"conditionRuleModifierAfter[92]", "conditionRuleModifierAfter[93]", "conditionRuleModifierAfter[94]", "conditionRuleModifierAfter[95]",
			"conditionRuleModifierAfter[96]", "conditionRuleModifierAfter[97]", "conditionRuleModifierAfter[98]", "conditionRuleModifierAfter[99]",
			"conditionRuleModifierAfter[100]", "conditionRuleModifierAfter[101]", "conditionRuleModifierAfter[102]", "effectiveStrengthAfter",
			"effectiveIntelligenceAfter", "effectiveDexterityAfter", "effectiveSpeedAfter", "effectiveSensesAfter",
			"hpRecoveryCoefficientBitsAfter", "spRecoveryCoefficientBitsAfter",
		},
		F: monsterActivationStatsNode,
	}
}
