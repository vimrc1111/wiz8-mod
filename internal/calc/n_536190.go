package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var combatTargetDispatch, combatTargetDispatchErr = loadCombatTargetDispatch()

// The nine target-kind destinations at 0x5363d4 remain in the user's exe.
// Initialization uses exe.go's PE mapping; the node itself performs no IO.
func loadCombatTargetDispatch() ([9]uint32, error) {
	var table [9]uint32 // 0x5361a2: targetKind-1 has unsigned upper bound 8.
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return table, fmt.Errorf("combat target: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return table, err
	}
	offset, err := peOffset(b, 0x5363d4)
	if err != nil {
		return table, err
	}
	if len(b)-offset < len(table)*4 {
		return table, fmt.Errorf("combat target: truncated dispatch")
	}
	for i := range table {
		destination := binary.LittleEndian.Uint32(b[offset+4*i:])
		// Semantic branch entry addresses, from 0x536190..0x5363d1.
		switch destination {
		case 0x5361b2, 0x53621c, 0x536291, 0x53631f, 0x53639f, 0x5363cd:
			table[i] = destination
		default:
			return table, fmt.Errorf("combat target: unknown branch %#x", destination)
		}
	}
	return table, nil
}

// Inputs describe eight selectable character slots and two lists per entity
// kind. Each list contains a fixed, ineligible decoy ID 0xfffffffe at slot 0
// and an independent input record at slot 1. Counts select 0, 1 or 2 entries.
// The secondary list is searched only after the primary list misses, matching
// 0x4e5550/0x5100b0; both lists are dereferenced by the original child functions.
func combatTargetValidNode(in []int64, _ Rand) []int64 {
	if len(in) != 29 {
		panic("combat target: expected 29 inputs")
	}
	for i, v := range in {
		if v < 0 || v > 4294967295 {
			panic("combat target: expected u32 inputs")
		}
		switch i {
		case 3, 15, 20, 24, 28:
			if v > 255 {
				panic("combat target: expected u8 presence/activity")
			}
		case 11, 16, 21, 25:
			if v > 2 {
				panic("combat target: fixture list count exceeds 2")
			}
		}
	}
	if in[0] == 0 {
		return []int64{0}
	} // 0x536195: NULL target is false.
	kind := uint32(in[1]) - 1
	if kind > 8 {
		return []int64{1}
	} // 0x5361a2: other kinds default true.
	if combatTargetDispatchErr != nil {
		panic(combatTargetDispatchErr)
	}
	valid := true
	switch combatTargetDispatch[kind] {
	case 0x5361b2, 0x53621c:
		if in[2] >= 8 {
			panic("combat target: character index outside eight-slot fixture")
		}
		if combatTargetDispatch[kind] == 0x5361b2 {
			// 0x5361e2..0x536211: present, nonzero HP and unsigned status <18.
			valid = in[3] != 0 && in[4] != 0 && in[5] < 18
		} else {
			// 0x53624c..0x536286: dead and eligible for resurrection.
			valid = in[3] != 0 && in[4] == 0 && in[6] != 0 && in[7] == 0
		}
	case 0x536291:
		if in[8] == 4294967295 {
			panic("combat target: original asserts invalid monster ID")
		}
		record := combatTargetRecord(in, in[8], 11, 16)
		// 0x5362f6..0x536314: HP !=0, removed state ==0, activity byte !=0.
		valid = record >= 0 && in[record+2] != 0 && in[record+3] == 0 && in[record+4] != 0
	case 0x53631f:
		if in[9] == 4294967295 {
			panic("combat target: original asserts invalid group ID")
		}
		record := combatTargetRecord(in, in[9], 21, 25)
		// 0x536387..0x536398: member count and activity byte must be nonzero.
		valid = record >= 0 && in[record+2] != 0 && in[record+3] != 0
	case 0x53639f:
		// 0x5363c3: item definition -1 is the empty-item sentinel.
		valid = in[10] != 4294967295
	}
	if valid {
		return []int64{1}
	}
	return []int64{0}
}

func combatTargetRecord(in []int64, id int64, primary, secondary int) int {
	for _, offset := range []int{primary, secondary} {
		if in[offset] > 0 && id == 4294967294 {
			return -1 // Fixed decoys have zero HP/member count; first match wins.
		}
		if in[offset] == 2 && id == in[offset+1] {
			return offset
		}
	}
	return -1 // Original lookup returns -1; target validation returns false.
}

func init() {
	Nodes["00536190"] = Node{
		Addr: "00536190",
		In:   []string{"targetPointer", "targetKind", "characterIndex", "characterPresent", "characterHP", "characterPrimaryStatus", "resurrectionAvailable", "resurrectionBlocked", "monsterID", "groupID", "itemDefinitionID", "monsterPrimaryCount", "monsterPrimaryID", "monsterPrimaryHP", "monsterPrimaryExcludedState", "monsterPrimaryActive", "monsterSecondaryCount", "monsterSecondaryID", "monsterSecondaryHP", "monsterSecondaryExcludedState", "monsterSecondaryActive", "groupPrimaryCount", "groupPrimaryID", "groupPrimaryMembers", "groupPrimaryActive", "groupSecondaryCount", "groupSecondaryID", "groupSecondaryMembers", "groupSecondaryActive"},
		Out:  []string{"valid"}, F: combatTargetValidNode,
	}
}
