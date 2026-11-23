package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var monsterFatigueDispatch, monsterFatigueDispatchErr = loadMonsterFatigueDispatch()

// The action dispatch stays in the player's executable, not in repo data.
func loadMonsterFatigueDispatch() ([11]uint32, error) {
	var dispatch [11]uint32 // 0x52c251: action+1 has unsigned upper bound 10.
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return dispatch, fmt.Errorf("monster fatigue: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return dispatch, err
	}
	o, err := peOffset(b, 0x52c2ec)
	if err != nil {
		return dispatch, err
	}
	if len(b)-o < 4*len(dispatch) {
		return dispatch, fmt.Errorf("monster fatigue: truncated action dispatch")
	}
	for i := range dispatch {
		destination := binary.LittleEndian.Uint32(b[o+4*i:])
		switch destination {
		case 0x52c25d, 0x52c26e, 0x52c27f, 0x52c28e, 0x52c2dd:
			dispatch[i] = destination
		default:
			return dispatch, fmt.Errorf("monster fatigue: unknown destination %#x", destination)
		}
	}
	return dispatch, nil
}

// Pure after table initialization. All three runtime monster fields are inputs.
func monsterActionFatigueCostNode(in []int64, r Rand) []int64 {
	if len(in) != 3 || in[0] < -2147483648 || in[0] > 2147483647 ||
		in[1] < -2147483648 || in[1] > 2147483647 || in[2] < 0 || in[2] > 4294967295 {
		panic("monster fatigue: expected i32 action/subaction and u32 fatigueDoubleState")
	}
	index := uint32(int32(in[0])) + 1
	if index > 10 { // 0x52c24c..0x52c254; invalid actions assert at 0x52c2b9.
		panic("monster fatigue: original asserts invalid action")
	}
	if monsterFatigueDispatchErr != nil {
		panic(monsterFatigueDispatchErr)
	}
	cost := 0
	switch monsterFatigueDispatch[index] {
	case 0x52c25d:
		cost = r.R(5) + 5 // 0x52c25d..0x52c269.
	case 0x52c26e:
		cost = r.R(3) + 3 // 0x52c26e..0x52c27a.
	case 0x52c27f:
		cost = r.R(2) + 1 // 0x52c27f..0x52c28b.
	case 0x52c28e:
		if in[1] == 3 { // 0x52c28e: subaction selects the larger cost.
			cost = r.R(8) + 5 // 0x52c2a8..0x52c2b4.
		} else {
			cost = r.R(3) + 2 // 0x52c297..0x52c2a3.
		}
	}
	if in[2] != 0 {
		cost *= 2 // 0x52c2dd..0x52c2e9: unsigned nonzero status, including bit 31.
	}
	return []int64{int64(cost)}
}

func init() {
	Nodes["0052c240"] = Node{
		Addr: "0052c240",
		In:   []string{"actionKind", "subactionKind", "fatigueDoubleState"},
		Out:  []string{"fatigueCost"}, F: monsterActionFatigueCostNode,
	}
}
