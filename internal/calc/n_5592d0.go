package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// 0x5592d0 npc_attribute_initialization. Attribute slot p of the target is the
// u32 at target+0xe5+0x14*p (0x559419, 0x5594c0). Slots 0..6 are the seven
// attributes; the random top-up loop (0x559560..0x559586) walks its cursor
// past slot 6 into later character fields without a bound, so inputs and
// outputs carry slots 0..npcAttrInitSlots-1. Slot 9 is target+0x199.
const (
	npcAttrInitSlots = 64 // fixture cap: calc panics if the cursor reaches it
	npcAttrInitIn    = 2 + 6 + 1 + (npcAttrInitSlots - 7)
)

type npcAttrInitTable struct {
	race    [11][7]uint32 // 0x614cf0, 7 u32 per race
	classMn [15][7]uint32 // 0x614e24, 7 u32 per class
	primary [15][3]uint32 // 0x614fc8, 3 u32 per class (slot numbers)
}

var npcAttrInitTables, npcAttrInitTablesErr = loadNpcAttrInitTables()

func loadNpcAttrInitTables() (t npcAttrInitTable, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	u32s := func(va uint32, dst []uint32) error {
		o, e := peOffset(b, va)
		if e != nil {
			return e
		}
		if len(b)-o < 4*len(dst) {
			return fmt.Errorf("npc attribute init: table %#x truncated", va)
		}
		for i := range dst {
			dst[i] = binary.LittleEndian.Uint32(b[o+4*i:])
		}
		return nil
	}
	for i := range t.race {
		if err = u32s(0x614cf0+28*uint32(i), t.race[i][:]); err != nil {
			return
		}
	}
	for i := range t.classMn {
		if err = u32s(0x614e24+28*uint32(i), t.classMn[i][:]); err != nil {
			return
		}
		if err = u32s(0x614fc8+12*uint32(i), t.primary[i][:]); err != nil {
			return
		}
		for _, k := range t.primary[i] {
			if k > 6 { // the original indexes a 7-dword local array
				return t, fmt.Errorf("npc attribute init: primary slot %d out of range", k)
			}
		}
	}
	return
}

// Inputs: class (+0x69, 0..14), race (+0x71, 0..10), six source bytes
// (+0x170..+0x175), old +0xe1, then initial slots 7..npcAttrInitSlots-1.
// Outputs: slots 0..npcAttrInitSlots-1, then +0xe1.
func npcAttributeInitializationNode(in []int64, r Rand) []int64 {
	if npcAttrInitTablesErr != nil {
		panic(npcAttrInitTablesErr)
	}
	if len(in) != npcAttrInitIn {
		panic("npc attribute init: input count")
	}
	t := &npcAttrInitTables
	class, race := in[0], in[1]
	if class < 0 || class > 14 || race < 0 || race > 10 {
		panic("npc attribute init: class or race beyond exe tables")
	}
	var w [7]int64 // 0x559414 local weights; w[6] is the six-attribute mean
	sum := int64(0)
	for i := 0; i < 6; i++ {
		if in[2+i] < 0 || in[2+i] > 255 {
			panic("npc attribute init: source attribute width")
		}
		w[i] = in[2+i]
		sum += w[i]
	}
	var slot [npcAttrInitSlots]uint32
	for i := 7; i < npcAttrInitSlots; i++ {
		v := in[9+i-7]
		if v < 0 || v > 4294967295 {
			panic("npc attribute init: slot width")
		}
		slot[i] = uint32(v)
	}
	e1 := uint32(in[8])
	w[6] = sum / 6           // 0x5593d7..0x5593e3, signed divide by 6
	div := sum + w[6] + 0x78 // 0x5593e8..0x5593ed, before the primary bonus
	for _, k := range t.primary[class] {
		w[k] += 40 // 0x559404..0x559417
	}
	raise := int64(0)
	for j := 0; j < 7; j++ {
		slot[j] = t.race[race][j]                    // 0x559427..0x559445
		if mn := t.classMn[class][j]; slot[j] < mn { // 0x55944d..0x559477
			raise += int64(mn - slot[j])
			slot[j] = mn
		}
	}
	rem := 0x46 - raise // 0x559479
	if rem < 0 {
		need := -rem
		for done := int64(0); done < need; { // 0x55949a..0x5594f9
			j := r.R(7)
			if t.classMn[class][j] != 0 && slot[j] > t.race[race][j] {
				slot[j]--
				done++
			}
		}
		slot[9] = uint32(int32(rem)) // target+0x199
		e1 = 1
	} else {
		used := int64(0)
		for j := 0; j < 7; j++ { // 0x559525..0x55953c
			q := uint32(rem * w[j] / div)
			if room := 100 - slot[j]; q >= room { // unsigned compare, jb at 0x55953a
				q = room
			}
			slot[j] += q
			used += int64(q)
		}
		rem -= used
		left := 3 * rem
		for p := int64(0); p < left; { // 0x559564..0x559586; cursor ignores the draw
			r.R(7)
			if p >= npcAttrInitSlots {
				panic("npc attribute init: cursor beyond fixture slots")
			}
			if slot[p] < 100 {
				slot[p]++
				rem--
				left -= 3
				if rem == 0 {
					break
				}
			}
			p++
		}
	}
	out := make([]int64, 0, npcAttrInitSlots+1)
	for _, v := range slot {
		out = append(out, int64(v))
	}
	return append(out, int64(e1))
}

func init() {
	in := []string{"class", "race"}
	for i := 0; i < 6; i++ {
		in = append(in, fmt.Sprintf("sourceAttr%d", i))
	}
	in = append(in, "initializedFlagOld")
	out := []string{}
	for i := 0; i < npcAttrInitSlots; i++ {
		out = append(out, fmt.Sprintf("slot%d", i))
		if i >= 7 {
			in = append(in, fmt.Sprintf("slotOld%d", i))
		}
	}
	out = append(out, "initializedFlag")
	Nodes["005592d0"] = Node{Addr: "005592d0", In: in, Out: out, F: npcAttributeInitializationNode}
}
