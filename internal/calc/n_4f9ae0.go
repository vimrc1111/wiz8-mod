package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

var spellTrainingScanNames = []string{"character0_buff5", "character1_buff6", "character2_buff7", "partyBuffScan3", "partyBuffScan4", "partyBuffScan5", "partyBuffScan6", "partyBuffScan7"}

func init() {
	Nodes["004f9ae0"] = Node{Addr: "004f9ae0", In: []string{
		"spell",
		"powerLevel",
		"durationPerPower",
		"durationBase",
		"targetType",
		"characterIndex",
		"combat",
		"combatRound",
		"groupPointer",
		"clock",
		"lockOpened",
		"cooldown0",
		"cooldown1",
		"cooldown2",
		"cooldown3",
		"cooldown4",
		"cooldown5",
		"cooldown6",
		"cooldown7",
		"cooldown8",
		"cooldown9",
		"cooldown10",
		"cooldown11",
		"cooldown12",
		"cooldown13",
		"partyEffect0",
		"partyEffect1",
		"partyEffect2",
		"partyEffect3",
		"partyEffect4",
		"partyEffect5",
		"partyEffect6",
		"partyEffect7",
		"partyEffect8",
		"partyEffect9",
		"partyEffect10",
		"partyEffect11",
		"groupEffect0",
		"groupEffect1",
		"groupEffect2",
		"groupEffect3",
		"groupEffect4",
		"groupEffect5",
		"groupEffect6",
		"groupEffect7",
		"groupEffect8",
		"groupObstacle0",
		"groupObstacle1",
		"groupObstacle2",
		"groupObstacle3",
		"groupObstacle4",
		"groupObstacle5",
		"groupObstacle6",
		"groupObstacle7",
		"groupObstacle8",
		"character0_active",
		"character0_mainStatus",
		"character0_maxHP",
		"character0_HP",
		"character0_maxStamina",
		"character0_stamina",
		"character0_status1",
		"character0_status2",
		"character0_status3",
		"character0_status4",
		"character0_status5",
		"character0_status6",
		"character0_status7",
		"character0_status8",
		"character0_status9",
		"character0_status10",
		"character0_status11",
		"character0_status12",
		"character0_status13",
		"character0_status14",
		"character0_status15",
		"character0_status16",
		"character0_status17",
		"character0_status18",
		"character0_buff1",
		"character0_buff2",
		"character0_buff3",
		"character0_buff4",
		"character0_buff5",
		"character0_buff6",
		"character0_buff7",
		"character1_active",
		"character1_mainStatus",
		"character1_maxHP",
		"character1_HP",
		"character1_maxStamina",
		"character1_stamina",
		"character1_status1",
		"character1_status2",
		"character1_status3",
		"character1_status4",
		"character1_status5",
		"character1_status6",
		"character1_status7",
		"character1_status8",
		"character1_status9",
		"character1_status10",
		"character1_status11",
		"character1_status12",
		"character1_status13",
		"character1_status14",
		"character1_status15",
		"character1_status16",
		"character1_status17",
		"character1_status18",
		"character1_buff1",
		"character1_buff2",
		"character1_buff3",
		"character1_buff4",
		"character1_buff5",
		"character1_buff6",
		"character1_buff7",
		"character2_active",
		"character2_mainStatus",
		"character2_maxHP",
		"character2_HP",
		"character2_maxStamina",
		"character2_stamina",
		"character2_status1",
		"character2_status2",
		"character2_status3",
		"character2_status4",
		"character2_status5",
		"character2_status6",
		"character2_status7",
		"character2_status8",
		"character2_status9",
		"character2_status10",
		"character2_status11",
		"character2_status12",
		"character2_status13",
		"character2_status14",
		"character2_status15",
		"character2_status16",
		"character2_status17",
		"character2_status18",
		"character2_buff1",
		"character2_buff2",
		"character2_buff3",
		"character2_buff4",
		"character2_buff5",
		"character2_buff6",
		"character2_buff7",
		"character3_active",
		"character3_mainStatus",
		"character3_maxHP",
		"character3_HP",
		"character3_maxStamina",
		"character3_stamina",
		"character3_status1",
		"character3_status2",
		"character3_status3",
		"character3_status4",
		"character3_status5",
		"character3_status6",
		"character3_status7",
		"character3_status8",
		"character3_status9",
		"character3_status10",
		"character3_status11",
		"character3_status12",
		"character3_status13",
		"character3_status14",
		"character3_status15",
		"character3_status16",
		"character3_status17",
		"character3_status18",
		"character3_buff1",
		"character3_buff2",
		"character3_buff3",
		"character3_buff4",
		"character3_buff5",
		"character3_buff6",
		"character3_buff7",
		"character4_active",
		"character4_mainStatus",
		"character4_maxHP",
		"character4_HP",
		"character4_maxStamina",
		"character4_stamina",
		"character4_status1",
		"character4_status2",
		"character4_status3",
		"character4_status4",
		"character4_status5",
		"character4_status6",
		"character4_status7",
		"character4_status8",
		"character4_status9",
		"character4_status10",
		"character4_status11",
		"character4_status12",
		"character4_status13",
		"character4_status14",
		"character4_status15",
		"character4_status16",
		"character4_status17",
		"character4_status18",
		"character4_buff1",
		"character4_buff2",
		"character4_buff3",
		"character4_buff4",
		"character4_buff5",
		"character4_buff6",
		"character4_buff7",
		"character5_active",
		"character5_mainStatus",
		"character5_maxHP",
		"character5_HP",
		"character5_maxStamina",
		"character5_stamina",
		"character5_status1",
		"character5_status2",
		"character5_status3",
		"character5_status4",
		"character5_status5",
		"character5_status6",
		"character5_status7",
		"character5_status8",
		"character5_status9",
		"character5_status10",
		"character5_status11",
		"character5_status12",
		"character5_status13",
		"character5_status14",
		"character5_status15",
		"character5_status16",
		"character5_status17",
		"character5_status18",
		"character5_buff1",
		"character5_buff2",
		"character5_buff3",
		"character5_buff4",
		"character5_buff5",
		"character5_buff6",
		"character5_buff7",
		"character6_active",
		"character6_mainStatus",
		"character6_maxHP",
		"character6_HP",
		"character6_maxStamina",
		"character6_stamina",
		"character6_status1",
		"character6_status2",
		"character6_status3",
		"character6_status4",
		"character6_status5",
		"character6_status6",
		"character6_status7",
		"character6_status8",
		"character6_status9",
		"character6_status10",
		"character6_status11",
		"character6_status12",
		"character6_status13",
		"character6_status14",
		"character6_status15",
		"character6_status16",
		"character6_status17",
		"character6_status18",
		"character6_buff1",
		"character6_buff2",
		"character6_buff3",
		"character6_buff4",
		"character6_buff5",
		"character6_buff6",
		"character6_buff7",
		"character7_active",
		"character7_mainStatus",
		"character7_maxHP",
		"character7_HP",
		"character7_maxStamina",
		"character7_stamina",
		"character7_status1",
		"character7_status2",
		"character7_status3",
		"character7_status4",
		"character7_status5",
		"character7_status6",
		"character7_status7",
		"character7_status8",
		"character7_status9",
		"character7_status10",
		"character7_status11",
		"character7_status12",
		"character7_status13",
		"character7_status14",
		"character7_status15",
		"character7_status16",
		"character7_status17",
		"character7_status18",
		"character7_buff1",
		"character7_buff2",
		"character7_buff3",
		"character7_buff4",
		"character7_buff5",
		"character7_buff6",
		"character7_buff7",
		"partyBuffScan3",
		"partyBuffScan4",
		"partyBuffScan5",
		"partyBuffScan6",
		"partyBuffScan7",
		"monster_maxHP",
		"monster_HP",
		"monster_maxStamina",
		"monster_stamina",
		"monster_status1",
		"monster_status2",
		"monster_status3",
		"monster_status4",
		"monster_status5",
		"monster_status6",
		"monster_status7",
		"monster_status8",
		"monster_status9",
		"monster_status10",
		"monster_status11",
		"monster_status12",
		"monster_status13",
		"monster_status14",
		"monster_status15",
		"monster_status16",
		"monster_status17",
		"monster_status18",
		"monster_buff1",
		"monster_buff2",
		"monster_buff3",
		"monster_buff4",
		"monster_buff5",
		"monster_buff6",
		"monster_buff7",
		"equipmentItem0",
		"curseRemoved0",
		"equipmentItem1",
		"curseRemoved1",
		"equipmentItem2",
		"curseRemoved2",
		"equipmentItem3",
		"curseRemoved3",
		"equipmentItem4",
		"curseRemoved4",
		"equipmentItem5",
		"curseRemoved5",
		"equipmentItem6",
		"curseRemoved6",
		"equipmentItem7",
		"curseRemoved7",
		"equipmentItem8",
		"curseRemoved8",
		"equipmentItem9",
		"curseRemoved9",
		"equipmentItem10",
		"curseRemoved10",
		"equipmentItem11",
		"curseRemoved11",
		"itemCursed0",
		"itemDifficulty0",
		"itemCursed1",
		"itemDifficulty1",
		"itemCursed2",
		"itemDifficulty2",
		"itemCursed3",
		"itemDifficulty3",
		"itemCursed4",
		"itemDifficulty4",
		"itemCursed5",
		"itemDifficulty5",
		"itemCursed6",
		"itemDifficulty6",
		"itemCursed7",
		"itemDifficulty7",
		"itemCursed8",
		"itemDifficulty8",
		"itemCursed9",
		"itemDifficulty9",
		"itemCursed10",
		"itemDifficulty10",
		"itemCursed11",
		"itemDifficulty11",
	}, Out: []string{"eligible", "cooldown0", "cooldown1", "cooldown2", "cooldown3", "cooldown4", "cooldown5", "cooldown6", "cooldown7", "cooldown8", "cooldown9", "cooldown10", "cooldown11", "cooldown12", "cooldown13"}, F: spellTrainingEligibilityNode}
}

// Static effect lists and the ratio limit belong to the player's executable.
// No extracted tables are embedded here. Address translation is shared with exe.go.
type spellTrainingTables struct {
	party, group, cooldown []uint32
	buff                   map[uint32]uint32
	ratio                  float64
}

var spellTrainingData, spellTrainingDataErr = loadSpellTrainingTables()

func loadSpellTrainingTables() (spellTrainingTables, error) {
	t := spellTrainingTables{buff: make(map[uint32]uint32)}
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("spell training: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	read := func(va uint32, n int) ([]byte, error) {
		off, e := peOffset(b, va)
		if e != nil {
			return nil, e
		}
		if off+n > len(b) {
			return nil, fmt.Errorf("spell training: truncated table at %#x", va)
		}
		return b[off : off+n], nil
	}
	for _, x := range []struct {
		va  uint32
		n   int
		dst *[]uint32
	}{
		{0x616d84, 12, &t.party}, {0x616dd8, 9, &t.group}, {0x616e34, 14, &t.cooldown},
	} {
		data, e := read(x.va, 4*x.n)
		if e != nil {
			return t, e
		}
		for i := 0; i < x.n; i++ {
			*x.dst = append(*x.dst, binary.LittleEndian.Uint32(data[4*i:]))
		}
	}
	data, err := read(0x5ebcc8, 4)
	if err != nil {
		return t, err
	}
	t.ratio = float64(math.Float32frombits(binary.LittleEndian.Uint32(data)))
	// 0x551900: spell-19 indexes a byte dispatch table; each accepted target
	// returns its effect slot as a 32-bit immediate (0x55191b..0x55193f).
	for _, spell := range []uint32{19, 21, 27, 54, 61, 65} {
		index, e := read(0x551984+spell-19, 1)
		if e != nil {
			return t, e
		}
		target, e := read(0x551964+4*uint32(index[0]), 4)
		if e != nil {
			return t, e
		}
		address := binary.LittleEndian.Uint32(target)
		if address < 0x55191b || address > 0x55193f {
			return t, fmt.Errorf("spell training: unsupported buff dispatch")
		}
		value, e := read(address+1, 4)
		if e != nil {
			return t, e
		}
		t.buff[spell] = binary.LittleEndian.Uint32(value)
	}
	return t, nil
}

// The case owns eight independent character records, a loaded monster, and
// twelve item definitions. The node depends only on these inputs and immutable
// executable tables, consumes no RNG and returns AL plus all 14 cooldowns.
// Spell record stride 0x1bf and duration operands +0x44/+0x4d: 0x4f9aee..0x4f9b18.
// Character/monster status bases +0xa01/+0x57: 0x4f9fca..0x4fa24d.
// Vitals and party eligibility: 0x4f9d6a..0x4f9fc0. Status indices below follow
// those arrays, without asserting unverified display names for each status.
func spellTrainingEligibilityNode(in []int64, r Rand) []int64 {
	names := Nodes["004f9ae0"].In
	if len(in) != len(names) {
		panic("spell training: wrong input count")
	}
	if spellTrainingDataErr != nil {
		panic(spellTrainingDataErr)
	}
	values := make(map[string]int64, len(in))
	for i, n := range names {
		values[n] = in[i]
	}
	get := func(n string) int64 { return values[n] }
	spell := uint32(get("spell"))
	who := get("characterIndex")
	if spell > 127 || who < 0 || who > 7 {
		panic("spell training: spell/character outside fixture")
	}
	typ := get("targetType")
	combat := get("combat") != 0
	round := get("combatRound") != 0
	// Multiplication/addition retain the original 32-bit wrap and signed divisor.
	duration := int32(get("durationPerPower"))*int32(get("powerLevel")) + int32(get("durationBase"))
	if duration != 9999 {
		duration++
	} // 0x4f9b0e sentinel, 0x4f9b18 increment.
	ratioOK := func(remaining int64) bool {
		// 0x4f9bc2 and 0x4f9d42 accept unordered as well as <=. Thus 0/0
		// accepts; positive/0 rejects. No float32 rounding occurs in the x87 path.
		if duration == 0 {
			return remaining == 0
		}
		return float64(uint32(remaining))/float64(duration) <= spellTrainingData.ratio
	}
	out := make([]int64, 15)
	for i := 0; i < 14; i++ {
		out[i+1] = get(fmt.Sprintf("cooldown%d", i))
	}
	find := func(table []uint32) int {
		for i, s := range table {
			if s == spell {
				return i
			}
		}
		return -1
	}
	cooldown := func() bool {
		i := find(spellTrainingData.cooldown)
		if i < 0 {
			return true
		}
		now := uint32(get("clock"))
		eligible := uint32(out[i+1]) <= now // 0x406c00 uses unsigned deadline comparison.
		out[i+1] = int64(now + 180000)      // 0x4fa307/0x4fa37c, 0x406bf0; wraps at 32 bits.
		return eligible
	}
	prefix := "monster_"
	if typ == 1 {
		prefix = fmt.Sprintf("character%d_", who)
	}
	status := func(s int) bool { return get(fmt.Sprintf("%sstatus%d", prefix, s)) != 0 }
	eligible := true
	switch spell {
	case 2, 53, 59, 62:
		eligible = combat && round
		if eligible {
			if i := find(spellTrainingData.group); i >= 0 {
				eligible = ratioOK(get(fmt.Sprintf("groupEffect%d", i)))
			}
		}
	case 19, 21, 27, 54, 61, 65:
		eligible = combat && round
		if eligible {
			eligible = ratioOK(get(fmt.Sprintf("%sbuff%d", prefix, spellTrainingData.buff[spell])))
		}
	case 20, 26, 32, 40:
		if combat {
			if i := find(spellTrainingData.party); i >= 0 {
				eligible = ratioOK(get(fmt.Sprintf("partyEffect%d", i)))
			}
		} else {
			eligible = cooldown()
		}
	case 8, 17, 33, 45, 64, 73, 75:
		eligible = cooldown()
	case 30, 38:
		if get("groupPointer") == 0 || !round {
			eligible = cooldown()
		}
	case 6:
		eligible = get(prefix+"HP") != get(prefix+"maxHP")
	case 13:
		eligible = combat && get(prefix+"stamina") != get(prefix+"maxStamina")
	case 16:
		eligible = status(3) || status(4) || status(6) || status(15) || status(12)
	case 23:
		eligible = get("lockOpened") == 0
	case 34:
		eligible = status(16)
	case 35:
		eligible = status(7)
	case 51:
		eligible = status(2)
	case 74:
		eligible = status(11) || status(13)
	case 88:
		// 0x4fa139 tests target type 9, unlike the other character selectors.
		if typ == 9 {
			prefix = fmt.Sprintf("character%d_", who)
		} else {
			prefix = "monster_"
		}
		eligible = status(18)
	case 120:
		eligible = status(1)
	case 100:
		eligible = uint32(get(prefix+"HP")) < uint32(get(prefix+"maxHP")) || uint32(get(prefix+"stamina")) < uint32(get(prefix+"maxStamina"))
		for i := 1; i < 18; i++ {
			eligible = eligible || status(i)
		} // 0x4f9f5f/0x4f9fbb, excludes slot 18.
	case 44, 68:
		eligible = false
		if spell == 68 || combat {
			for i := 0; i < 8; i++ {
				p := fmt.Sprintf("character%d_", i)
				current, maximum := "HP", "maxHP"
				if spell == 44 {
					current, maximum = "stamina", "maxStamina"
				}
				if get(p+"active") != 0 && uint32(get(p+"mainStatus")) < 19 && uint32(get(p+current)) < uint32(get(p+maximum)) {
					eligible = true
					break
				}
			}
		}
	case 56:
		eligible = combat && round
		if eligible && typ == 2 {
			eligible = false
			// 0x4f9d57 advances by 0x186e, deliberately not the normal 0x1862.
			for i, n := range spellTrainingScanNames {
				if get(fmt.Sprintf("character%d_active", i)) != 0 && ratioOK(get(n)) {
					eligible = true
				}
			}
		}
	case 72:
		eligible = false
		if get("groupPointer") != 0 {
			for i := 0; i < 9; i++ {
				if get(fmt.Sprintf("groupObstacle%d", i)) != 0 {
					eligible = true
					break
				}
			}
		}
	case 58:
		if !combat {
			eligible = cooldown()
		}
		if eligible {
			eligible = status(9)
			if !eligible && typ == 1 {
				equipment := []int64{who}
				for i := 0; i < 12; i++ {
					equipment = append(equipment, get(fmt.Sprintf("equipmentItem%d", i)), get(fmt.Sprintf("curseRemoved%d", i)))
				}
				for i := 0; i < 12; i++ {
					equipment = append(equipment, get(fmt.Sprintf("itemCursed%d", i)), get(fmt.Sprintf("itemDifficulty%d", i)))
				}
				eligible = removeCursePowerRequiredNode(equipment, r)[0] != 0
			}
		}
	}
	if eligible {
		out[0] = 1
	}
	return out
}
