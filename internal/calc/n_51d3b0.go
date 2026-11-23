package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Interface: re/emu/cases/0051d3b0.json. The first 508 fields share
// 0x50e5c0; twelve item definitions have stride 269 (notes/re/loot.md:9).

func init() {
	inputs := append([]string(nil), Nodes["0050e5c0"].In...)
	inputs = append(inputs, "showMessage", "rebuildEffects", "combatMessageContext", "curseOverride")
	for slot := 6; slot <= 9; slot++ {
		for _, field := range []string{"instanceState5", "identified", "instanceState7", "instanceState8", "instanceState9", "curseExempt", "curseActivation"} {
			inputs = append(inputs, fmt.Sprintf("handSlot[%d].%s", slot, field))
		}
	}
	for item := 0; item < 12; item++ {
		for _, field := range []string{"type", "subtype", "handFlags", "range", "cursed"} {
			inputs = append(inputs, fmt.Sprintf("handItem[%d].%s", item, field))
		}
	}
	for item := 0; item < 12; item++ {
		inputs = append(inputs, fmt.Sprintf("handItem[%d].name", item))
	}
	inputs = append(inputs, "categoryName")
	for i := 0; i < 8; i++ {
		inputs = append(inputs, fmt.Sprintf("nameBuffer[%d]", i))
	}
	// Rebuild outputs 103..359 are the cached fields absent from its inputs.
	inputs = append(inputs, Nodes["0050e5c0"].Out[103:360]...)
	inputs = append(inputs, "handMode[0]", "handMode[1]")
	outputs := []string{"allowed"}
	for slot := 6; slot <= 9; slot++ {
		outputs = append(outputs, fmt.Sprintf("handSlot[%d].itemID", slot), fmt.Sprintf("handSlot[%d].quantity", slot))
		for i := 5; i <= 11; i++ {
			outputs = append(outputs, fmt.Sprintf("handSlot[%d].state%d", slot, i))
		}
	}
	for i := 0; i < 8; i++ {
		outputs = append(outputs, fmt.Sprintf("nameBuffer[%d]", i))
	}
	outputs = append(outputs, Nodes["0050e5c0"].Out...)
	outputs = append(outputs, "handMode[0]", "handMode[1]")
	Nodes["0051d3b0"] = Node{Addr: "0051d3b0", In: inputs, Out: outputs, F: weaponSetSwapAllowedNode}
}

// Records at C+0xfa5/+0xfb1/+0xfbd/+0xfc9 contain an i32 ID and
// eight instance bytes (0x51fd20/0x51fe30). Empty IDs are -1.
func weaponSetSwapAllowedNode(in []int64, r Rand) []int64 {
	if len(in) != 880 {
		panic("weapon swap: expected 880 fields")
	}
	var records [4][9]int64
	for h := 0; h < 4; h++ {
		records[h][0] = in[282+h]
		records[h][1] = in[486+h]
		copy(records[h][2:], in[512+7*h:519+7*h])
		if records[h][0] < -1 || records[h][0] > 11 {
			panic("weapon swap: item ID outside fixture definitions")
		}
	}
	buffer := append([]int64(nil), in[613:621]...)
	prior := append([]int64(nil), in[70:173]...)
	prior = append(prior, in[621:878]...)
	prior = append(prior, in[5:21]...)
	modes := append([]int64(nil), in[878:880]...)
	finish := func(allowed int64) []int64 {
		out := []int64{allowed}
		for _, rec := range records {
			out = append(out, rec[:]...)
		}
		out = append(out, buffer...)
		out = append(out, prior...)
		return append(out, modes...)
	}
	if records[0][0] == -1 && records[1][0] == -1 && records[2][0] == -1 && records[3][0] == -1 {
		return finish(0)
	}
	if curseActivationErr != nil {
		panic(curseActivationErr)
	}
	// Both current hands activate before either blocking test (0x51d449/451).
	for h := 0; h < 2; h++ {
		rec := &records[h]
		if rec[0] == -1 || !curseActivationTable.applicable[6+h] || rec[8] != 0 {
			continue
		}
		rec[8] = 1
		if in[544+5*int(rec[0])] == 0 || rec[7] != 0 {
			continue
		}
		name := in[600+int(rec[0])]
		if rec[3] == 0 {
			name = in[612]
		}
		format := curseActivationTable.name
		if rec[1] >= 2 {
			format = curseActivationTable.quantity
		}
		text := curseActivationName(format, uint16(name), rec[1])
		if len(text) > len(buffer) {
			panic("weapon swap: name exceeds buffer projection")
		}
		copy(buffer, text)
	}
	for h := 0; h < 2; h++ {
		rec := records[h]
		if rec[0] != -1 && in[544+5*int(rec[0])] != 0 && rec[7] == 0 && curseActivationTable.applicable[6+h] && in[511] == 0 {
			return finish(0)
		}
	}
	compatible := func(a, b int64) bool {
		if a == -1 || b == -1 {
			return true
		}
		fields := []int64{a, b}
		fields = append(fields, in[540+5*int(a):544+5*int(a)]...)
		fields = append(fields, in[540+5*int(b):544+5*int(b)]...)
		return compatibleHandItemsNode(fields, r)[0] != 0
	}
	keepMain := records[2][0] == -1 && records[3][0] != -1 && compatible(records[0][0], records[3][0])
	keepOff := records[3][0] == -1 && records[2][0] != -1 && compatible(records[1][0], records[2][0])
	for h, keep := range []bool{keepMain, keepOff} {
		if keep || records[h][0] == -1 && records[h+2][0] == -1 {
			continue
		}
		// Each moved nonempty instance loses its exemption. The vacated record
		// is cleared, so metadata in an empty slot never moves (0x51fea6/51ff21).
		for _, k := range []int{h, h + 2} {
			if records[k][0] == -1 {
				records[k] = [9]int64{-1}
			} else {
				records[k][7] = 0
			}
		}
		records[h], records[h+2] = records[h+2], records[h]
		// 0x520d10 refreshes the active hand mode after each removal/insertion.
		modes[h] = weaponSwapHandMode(h, records, in)

	}
	if uint8(in[509]) != 0 {
		rebuild := append([]int64(nil), in[:508]...)
		for h, rec := range records {
			rebuild[282+h] = rec[0]
			rebuild[486+h] = rec[1]
		}
		prior = equipmentEffectRebuildNode(rebuild, r)
		// Rebuild child 0x4ee220 refreshes both modes, including a kept hand.
		for h := range modes {
			modes[h] = weaponSwapHandMode(h, records, in)
		}
	}
	return finish(1)
}

// Read the dispatch table used by child 0x5201b0 from the user's exe.
// No original table or instruction bytes are retained in this adapter.
var weaponSwapModes, weaponSwapModeErr = loadWeaponSwapModes()

func loadWeaponSwapModes() ([6]uint32, error) {
	var modes [6]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return modes, fmt.Errorf("weapon swap: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return modes, err
	}
	offset, err := peOffset(data, 0x5202f4)
	if err != nil {
		return modes, err
	}
	if len(data)-offset < 24 {
		return modes, fmt.Errorf("weapon swap: truncated hand mode table")
	}
	for i := range modes {
		modes[i] = binary.LittleEndian.Uint32(data[offset+4*i:])
	}
	return modes, nil
}
func weaponSwapHandMode(hand int, records [4][9]int64, in []int64) int64 {
	id := records[hand][0]
	if id == -1 {
		return 0
	}
	if weaponSwapModeErr != nil {
		panic(weaponSwapModeErr)
	}
	kind := in[540+5*int(id)]
	if kind < 0 || kind >= 6 {
		panic("weapon swap: invalid equipped hand type")
	}
	// Semantic branches of 0x5201b0: ordinary weapon, conditional dual mode,
	// ammunition and shield. Mode values come from 0x520248/27e/285.
	switch weaponSwapModes[kind] {
	case 0x520248:
		return 1
	case 0x52024f:
		main := records[0][0]
		if hand == 1 && main != -1 && in[541+5*int(main)] == 131 {
			return 3
		}
		return 1
	case 0x52027e:
		return 3
	case 0x520285:
		return 2
	default:
		panic("weapon swap: unknown hand mode dispatch")
	}
}
