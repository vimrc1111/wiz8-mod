package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var actionTargetDispatch, actionTargetDispatchErr = loadActionTargetDispatch()

// Read the action dispatch from the player's executable, using exe.go's PE
// address translation. No original table entries are stored in the repository.
func loadActionTargetDispatch() ([9]uint32, error) {
	var dispatch [9]uint32 // 0x536a24: action kinds through 8.
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return dispatch, fmt.Errorf("action target: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return dispatch, err
	}
	o, err := peOffset(b, 0x536aa8)
	if err != nil {
		return dispatch, err
	}
	if len(b)-o < 4*len(dispatch) {
		return dispatch, fmt.Errorf("action target: truncated dispatch table")
	}
	for i := range dispatch {
		destination := binary.LittleEndian.Uint32(b[o+4*i:])
		switch destination {
		case 0x536a31, 0x536a38, 0x536a4b, 0x536a52, 0x536a59, 0x536aa4:
			dispatch[i] = destination
		default:
			return dispatch, fmt.Errorf("action target: unknown destination %#x", destination)
		}
	}
	return dispatch, nil
}

// Pure after initialization. The fixture supplies four indexed item definitions
// and 256 indexed spell definitions; pointers select records, not rule values.
// Both possible spell records receive spellTarget, since only one is consumed
// in any invocation. Source category 6 never dereferences the battle object.
func actionTargetTypeNode(in []int64, r Rand) []int64 {
	if len(in) != 8 || in[0] < 0 || in[0] > 4294967295 ||
		in[1] < 0 || in[1] > 255 || in[2] < 0 || in[2] > 4294967295 ||
		in[3] < -1 || in[3] > 3 || in[4] < 0 || in[4] > 255 ||
		in[5] < 0 || in[5] > 255 || in[6] < 0 || in[6] > 10 ||
		in[7] < 0 || in[7] > 4294967295 {
		panic("action target: invalid action, spell, item fixture, target kind or battle pointer")
	}
	if in[0] > 8 { // 0x536a24..0x536a28: unsigned action bound.
		return []int64{0}
	}
	if actionTargetDispatchErr != nil {
		panic(actionTargetDispatchErr)
	}
	switch actionTargetDispatch[in[0]] {
	case 0x536a31:
		return []int64{2} // 0x536a31: single enemy target (ordinary attacks).
	case 0x536a4b:
		return []int64{4} // 0x536a4b: enemy group.
	case 0x536a52:
		return []int64{1} // 0x536a52: single ally.
	case 0x536a38:
		// 0x536a3c..0x536a41: no item self-use flag, source category 6.
		// Reuse the composed 5010f0 rule, with a present synthetic instance.
		return itemSpellTargetTypeNode([]int64{1, 0, in[1], 0, in[6], in[7]}, r)
	case 0x536a59:
		// 0x536a59..0x536a9a: wrapper +4 selects the item instance;
		// its index, spell +0x63 and type +0x3e feed the existing rule.
		return itemSpellTargetTypeNode([]int64{in[2], in[3], in[4], in[5], in[6], in[7]}, r)
	default:
		return []int64{0} // 0x536aa4: no target for this action/item.
	}
}

func init() {
	Nodes["00536a20"] = Node{
		Addr: "00536a20",
		In:   []string{"actionKind", "spellID", "itemInstance", "itemIndex", "itemSpellID", "itemType", "spellTarget", "battle"},
		Out:  []string{"targetType"}, F: actionTargetTypeNode,
	}
}
