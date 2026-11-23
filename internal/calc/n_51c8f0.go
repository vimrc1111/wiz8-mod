package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type partnerItemTables struct {
	classification [121]byte
	targets        [6]uint32
	specialClass   [12]byte
	specialTargets [4]uint32
	partners       map[uint16]uint16
}

var partnerTables, partnerTablesErr = loadPartnerItemTables()

func loadPartnerItemTables() (partnerItemTables, error) {
	var t partnerItemTables
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("compatible partner items: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	at := func(va uint32, n int) ([]byte, error) {
		off, err := peOffset(b, va)
		if err != nil {
			return nil, err
		}
		if n > len(b)-off {
			return nil, fmt.Errorf("compatible partner items: truncated table at %#x", va)
		}
		return b[off : off+n], nil
	}
	// Classification bounds: 0x51c976..0x51c984; special ID bounds:
	// 0x51ca64..0x51ca75. All dispatch data come from the user's executable.
	for _, spec := range []struct {
		va  uint32
		dst []byte
	}{
		{0x51cba8, t.classification[:]}, {0x51cc34, t.specialClass[:]},
	} {
		raw, err := at(spec.va, len(spec.dst))
		if err != nil {
			return t, err
		}
		copy(spec.dst, raw)
	}
	for _, spec := range []struct {
		va  uint32
		dst []uint32
	}{
		{0x51cb90, t.targets[:]}, {0x51cc24, t.specialTargets[:]},
	} {
		raw, err := at(spec.va, 4*len(spec.dst))
		if err != nil {
			return t, err
		}
		for i := range spec.dst {
			spec.dst[i] = binary.LittleEndian.Uint32(raw[4*i:])
		}
	}
	// Zero-terminated subtype pairs, four bytes per row: 0x51ca22..0x51ca53.
	t.partners = make(map[uint16]uint16)
	for va := uint32(0x616e6c); ; va += 4 {
		raw, err := at(va, 4)
		if err != nil {
			return t, err
		}
		key := binary.LittleEndian.Uint16(raw)
		if key == 0 {
			break
		}
		if _, exists := t.partners[key]; !exists {
			t.partners[key] = binary.LittleEndian.Uint16(raw[2:])
		}
	}
	return t, nil
}

func init() {
	Nodes["0051c8f0"] = Node{Addr: "0051c8f0",
		In:  []string{"rangedItemIndex", "otherItemIndex", "rangedSubtype", "rangedRange", "otherType", "otherSubtype", "otherRange"},
		Out: []string{"compatible"}, F: compatiblePartnerItemsNode}
}

// Pure after table initialization. Definition stride 269 and database count
// 819 are documented in notes/re/loot.md:9. The case populates the selected
// records, including aliasing: the other item's fields are written last.
func compatiblePartnerItemsNode(in []int64, _ Rand) []int64 {
	if len(in) != 7 {
		panic("compatible partner items: expected seven inputs")
	}
	for i, v := range in {
		bound := int64(255)
		if i < 2 {
			bound = 818
		}
		if i == 2 || i == 5 {
			bound = 65535
		}
		if v < 0 || v > bound {
			panic("compatible partner items: invalid item index or field width")
		}
	}
	if partnerTablesErr != nil {
		panic(partnerTablesErr)
	}
	subtype, itemRange := uint16(in[2]), in[3]
	otherType, otherSubtype, otherRange := in[4], uint16(in[5]), in[6]
	if in[0] == in[1] {
		subtype, itemRange = otherSubtype, otherRange
	}
	compatible := false
	// First subtype 11 and span 120: 0x51c976..0x51c979.
	if subtype >= 11 && subtype <= 131 {
		class := partnerTables.classification[subtype-11]
		if int(class) >= len(partnerTables.targets) {
			panic("compatible partner items: invalid classification")
		}
		switch target := partnerTables.targets[class]; target {
		case 0x51c991:
			compatible = otherSubtype == 22 // 0x51c99e: arrow partner.
		case 0x51c9b1:
			compatible = otherSubtype == 23 // 0x51c9be: bolt partner.
		case 0x51c9d1:
			compatible = otherSubtype == 83 // 0x51c9de: stone partner.
		case 0x51c9f1:
			// Other types above 3 bypass range matching (0x51ca01);
			// smaller types require equal +0x47 range bytes (0x51ca07).
			if otherType > 3 || itemRange == otherRange {
				partner, found := partnerTables.partners[subtype]
				compatible = !found || otherSubtype == partner
			}
		case 0x51ca64:
			// Special definition IDs start at 599, span 11: 0x51ca64..0x51ca6a.
			if in[0] >= 599 && in[0] <= 610 {
				class := partnerTables.specialClass[in[0]-599]
				if int(class) >= len(partnerTables.specialTargets) {
					panic("compatible partner items: invalid special classification")
				}
				target := partnerTables.specialTargets[class]
				// The four increasingly inclusive sets follow 0x51ca82,
				// 0x51caab, 0x51cae2 and 0x51cb30 respectively.
				compatible = otherSubtype == 23 || otherSubtype == 113
				switch target {
				case 0x51ca82:
				case 0x51caab:
					compatible = compatible || otherSubtype == 112
				case 0x51cae2, 0x51cb30:
					compatible = compatible || otherSubtype == 112 || otherSubtype == 21 || otherSubtype == 126 || (otherType == 2 && otherSubtype == 0)
					if target == 0x51cb30 {
						compatible = compatible || otherSubtype == 22 || otherSubtype == 83
					}
				default:
					panic(fmt.Sprintf("compatible partner items: unknown special target %#x", target))
				}
			}
		case 0x51cb87:
		default:
			panic(fmt.Sprintf("compatible partner items: unknown target %#x", target))
		}
	}
	if compatible {
		return []int64{1}
	}
	return []int64{0}
}
