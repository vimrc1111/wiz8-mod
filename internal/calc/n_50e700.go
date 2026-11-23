package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// 0x50edc0 uses the spell index map at 0x50ef28 and nine destinations at
// 0x50ef04. Read the player's executable through exe.go's PE mapping.
var partyRebuildDispatch, partyRebuildDispatchErr = loadPartyRebuildDispatch()

func loadPartyRebuildDispatch() ([38]uint32, error) {
	var result [38]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return result, fmt.Errorf("party rebuild: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	read := func(va uint32, n int) ([]byte, error) {
		off, err := peOffset(b, va)
		if err != nil {
			return nil, err
		}
		if n > len(b)-off {
			return nil, fmt.Errorf("party rebuild: truncated table %#x", va)
		}
		return b[off : off+n], nil
	}
	indices, err := read(0x50ef28, 38)
	if err != nil {
		return result, err
	}
	targets, err := read(0x50ef04, 36)
	if err != nil {
		return result, err
	}
	for i, k := range indices {
		if k > 8 {
			return result, fmt.Errorf("party rebuild: invalid dispatch index")
		}
		result[i] = binary.LittleEndian.Uint32(targets[4*int(k):])
		switch result[i] {
		case 0x50ee07, 0x50ee39, 0x50ee6b, 0x50ee90, 0x50ee96, 0x50eeb3, 0x50eece, 0x50eeef, 0x50eef3:
		default:
			return result, fmt.Errorf("party rebuild: unknown destination %#x", result[i])
		}
	}
	return result, nil
}

func init() {
	in := []string{"combatActive"}
	records := func(name string, n int, mult bool) {
		for j := 0; j < n; j++ {
			for _, field := range []string{"active", "spell", "strength"} {
				in = append(in, fmt.Sprintf("%s[%d].%s", name, j, field))
			}
			if mult {
				in = append(in, fmt.Sprintf("%s[%d].multiplier", name, j))
			}
		}
	}
	records("sharedSpell", 12, true)
	records("combatSpell", 9, false)
	records("combatAura", 6, true)
	var out []string
	for j := 0; j < 103; j++ {
		out = append(out, fmt.Sprintf("partyModifier[%d]", j))
	}
	for s := 0; s < 8; s++ {
		for _, f := range []string{"present", "inParty", "hp", "primaryState", "class", "race"} {
			in = append(in, fmt.Sprintf("character[%d].%s", s, f))
		}
		for _, f := range []string{"equipment", "status", "initialTotal"} {
			for j := 0; j < 103; j++ {
				in = append(in, fmt.Sprintf("character[%d].%s[%d]", s, f, j))
			}
		}
		for j := 0; j < 103; j++ {
			out = append(out, fmt.Sprintf("character[%d].total[%d]", s, j))
		}
	}
	Nodes["0050e700"] = Node{Addr: "0050e700", In: in, Out: out, F: partyEffectRebuildNode}
}

// The observable rule interface is the complete party buffer and eight complete
// character total buffers. 0x4ed9d0 executes in the emulator; its additional
// derived character caches are outside this interface, as in n_50e650.go.
// C+0xb11 and C+0xb0d share each HP input (full-health fixture), following
// n_50db50.go and 0x52a38b. Raw attributes are 50, initial class is Fighter,
// class levels are zero, and inventory has zero weight. These fixed child inputs are
// irrelevant to buffer outputs in this context; derived HP adjusts with its
// maximum without invoking death/status/notification transitions.
// The effect display is absent in the case: 0x483e50/0x483e30 have no observable writes.
func partyEffectRebuildNode(in []int64, r Rand) []int64 {
	if len(in) != 2620 {
		panic("party rebuild: expected 2620 fields")
	}
	if partyRebuildDispatchErr != nil {
		panic(partyRebuildDispatchErr)
	}
	if statusRebuildTablesErr != nil {
		panic(statusRebuildTablesErr)
	}
	for _, v := range in {
		if v < 0 || v > 0xffffffff {
			panic("party rebuild: expected unsigned fields")
		}
	}
	var party [103]uint8
	// 0x517f60: byte power, u32 multiply/add wrapping, rounded percentage,
	// finally truncated to a byte. The caller's power arithmetic wraps first.
	scale := func(power uint8, m int64) uint8 { return uint8((uint32(power)*(uint32(m)+100) + 50) / 100) }
	for j := 0; j < 12; j++ {
		rec := in[1+4*j : 1+4*j+4]
		if rec[0] > 255 || rec[2] > 255 {
			panic("party rebuild: spell byte overflow")
		}
		if rec[0] == 0 || rec[1] < 8 || rec[1] > 45 {
			continue
		}
		p := uint8(rec[2])
		m := rec[3]
		switch partyRebuildDispatch[rec[1]-8] {
		case 0x50ee07:
			party[1] += scale(uint8((uint32(p)+1)/2), m)
		case 0x50ee39:
			party[5] += scale(uint8((uint32(p)+1)/2), m)
		case 0x50ee6b:
			party[7] += scale(p*4+7, m)
		case 0x50ee90:
			party[0x46] = 1
		case 0x50ee96:
			party[0x47] = scale(p, m) * 2
		case 0x50eeb3:
			party[0x48] = scale(p, m)
		case 0x50eece:
			party[0x49] = scale((p+5)*5, m)
		case 0x50eeef:
			party[0x4a] = 1
		}
	}
	// 0x50e731 / 0x50ef50: nine combat spell records and six aura records.
	if in[0] != 0 {
		for j := 0; j < 9; j++ {
			rec := in[49+3*j : 49+3*j+3]
			if rec[0] != 0 && rec[1] == 49 {
				party[5] -= uint8(rec[2])
			}
		}
		for j := 0; j < 6; j++ {
			rec := in[76+4*j : 76+4*j+4]
			if rec[0] == 0 {
				continue
			}
			p := uint8(rec[2])
			m := rec[3]
			switch rec[1] {
			case 2:
				v := scale(2, m)
				party[4] += v
				party[1] += v
			case 53:
				v := scale(p*7, m)
				for k := 0x3c; k < 0x40; k++ {
					party[k] += v
				}
			case 59:
				v := scale(p*10, m)
				party[0x40] += v
				party[0x41] += v
			}
		}
	}
	// 0x50e77f: scan all eight slots (character stride 0x1862, presence stride
	// 0x106), requiring nonzero HP and zero primary state before feature 10.
	for s := 0; s < 8; s++ {
		c := in[100+315*s : 100+315*s+315]
		if c[0] > 255 || c[1] > 255 || c[4] > 14 || c[5] > 15 {
			panic("party rebuild: invalid character fields")
		}
		if c[0] == 0 || c[2] == 0 || c[3] != 0 {
			continue
		}
		has := false
		for _, f := range statusRebuildTables.character.ClassFeat[c[4]] {
			has = has || f == 10
		}
		for _, f := range statusRebuildTables.raceFeatures[c[5]] {
			has = has || f == 10
		}
		if has {
			party[0x42] = 1
			party[0x43] = 1
			party[0x44] = 1
			break
		}
	}
	out := make([]int64, 0, 927)
	for _, v := range party {
		out = append(out, int64(v))
	}
	// 0x50f090: byte sums, flag union and unsigned strength maxima. The tail
	// +0x4c..+0x66 is untouched by merge and stays zero after initialization.
	merge := func(dst *[103]uint8, src []int64) {
		for k := 0; k < 0x4c; k++ {
			v := uint8(src[k])
			switch k {
			case 0x42, 0x43, 0x44, 0x45, 0x46, 0x4a:
				if v != 0 {
					dst[k] = 1
				}
			case 0x47, 0x48, 0x49:
				dst[k] = max(dst[k], v)
			default:
				dst[k] += v
			}
		}
	}
	for s := 0; s < 8; s++ {
		c := in[100+315*s : 100+315*s+315]
		for _, v := range c[6:] {
			if v > 255 {
				panic("party rebuild: modifier byte overflow")
			}
		}
		if c[0] == 0 {
			out = append(out, c[212:315]...)
			continue
		}
		var total [103]uint8
		merge(&total, c[6:109])
		merge(&total, c[109:212])
		if c[1] != 0 {
			merge(&total, out[:103])
		}
		for _, v := range total {
			out = append(out, int64(v))
		}
	}
	return out
}
