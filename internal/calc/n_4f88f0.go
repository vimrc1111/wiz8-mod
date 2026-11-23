package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
)

// exe 0x4f88f0 item_pool_selection. Fixture caps (panic past them, never a
// silent answer): ITEMTABLES pools 2 (sel & 0xffff), item definitions 64
// (ids read at +0x86), out-list count/cap 64. Slot count 40 and the 7 level
// triples are fixed by the exe (0x4f8a98 end 0x1cc, 0x4f8b41 end 0x617d38).
const (
	ipPools = 2
	ipItems = 64
	ipSlots = 40
	ipList  = 64
	ipTrip  = 7
	ipIn0   = 5 // sel, K, list.count, list.cap, ordinal
)

var (
	ipOnce  sync.Once
	ipTable [ipTrip][3]uint32
	ipErr   error
)

// ipTriples reads the player's own Wiz8.exe at 0x617ce0 (a, b, c per triple).
func ipTriples() [ipTrip][3]uint32 {
	ipOnce.Do(func() {
		path := os.Getenv("WIZ8_EXE")
		if path == "" {
			_, src, _, ok := runtime.Caller(0)
			if !ok {
				ipErr = fmt.Errorf("itemPoolSelection: set WIZ8_EXE")
				return
			}
			path = filepath.Join(filepath.Dir(src), "../../build/Wiz8.exe")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			ipErr = err
			return
		}
		o, err := peOffset(b, 0x617ce0)
		if err != nil || len(b)-o < ipTrip*12 {
			ipErr = fmt.Errorf("itemPoolSelection: triples unreadable: %v", err)
			return
		}
		for i := range ipTable {
			for k := range ipTable[i] {
				ipTable[i][k] = binary.LittleEndian.Uint32(b[o+12*i+4*k:])
			}
		}
	})
	if ipErr != nil {
		panic(ipErr)
	}
	return ipTable
}

func ipNames() (in, out []string) {
	in = []string{"sel", "K", "list.count", "list.cap", "ordinal"}
	for i := 0; i < 8; i++ {
		in = append(in, "slotActive"+strconv.Itoa(i), "totalLevel"+strconv.Itoa(i))
	}
	for p := 0; p < ipPools; p++ {
		for j := 0; j < ipSlots; j++ {
			s := "p" + strconv.Itoa(p) + ".s" + strconv.Itoa(j)
			in = append(in, s+".cls", s+".id", s+".w")
		}
		in = append(in, "p"+strconv.Itoa(p)+".flag")
	}
	for k := 0; k < ipItems; k++ {
		in = append(in, "item"+strconv.Itoa(k)+".v86")
	}
	in = append(in, "recId") // value the 0x51c020 stub writes into its out record
	out = []string{"ret", "list.count", "list.cap", "ordinal", "mk.calls", "place.calls", "mk.a2", "mk.a3", "mk.a4", "mk.a5"}
	return
}

func init() {
	in, out := ipNames()
	Nodes["004f88f0"] = Node{Addr: "004f88f0", In: in, Out: out, F: itemPoolSelectionNode}
}

// itemPoolSelectionNode: 0x51c020 (item record build) is address-stubbed in the
// case; its record (arg2 = item id, args 3..5 = 0) and call count are outputs,
// only the last id is kept by the stub. 0x4f93d0 (terrain placement of the new
// item object) is stubbed likewise and only counted. The record is not read by
// any rule here, so F and the emulator share this body.
func itemPoolSelectionNode(in []int64, r Rand) []int64 {
	nIn, _ := ipNames()
	if len(in) != len(nIn) {
		panic("itemPoolSelection: wrong input count")
	}
	sel, k, count, capa, ordinal := in[0], in[1], in[2], in[3], in[4]
	if sel < 0 || sel > 0xffffffff || k < 0 || k > 0xffffffff || ordinal < 0 || ordinal > 0xffffffff {
		panic("itemPoolSelection: sel/K/ordinal are u32")
	}
	if sel&0xffff >= ipPools {
		panic("itemPoolSelection: fixture holds 2 pools")
	}
	if count < 0 || capa < count || capa > ipList {
		panic("itemPoolSelection: out list needs 0 <= count <= cap <= 64")
	}
	level := uint32(partyAverageLevelNode(in[ipIn0:ipIn0+16], r)[0]) // exe 0x4f88f7
	base := ipIn0 + 16 + int(sel&0xffff)*(ipSlots*3+1)
	flag := in[base+ipSlots*3] != 0
	cls := func(j int) bool { return in[base+3*j] != 0 }
	id := func(j int) int64 { return in[base+3*j+1] }
	w := func(j int) int { return int(in[base+3*j+2]) }
	defs := ipIn0 + 16 + ipPools*(ipSlots*3+1)
	tri := ipTriples()

	var mkCalls, lastID int64
	make1 := func(j int) {
		mkCalls++
		lastID = id(j)
		ordinal = (ordinal + 1) & 0xffffffff // 0x6874c2, 0x4f6b90
		if capa < count+1 {                  // 0x4addf0 grows to exactly n
			capa = count + 1
		}
		count++
	}
	var a []int // list A; its capacity never shows in an output
	var wsum int
	for j := 0; j < ipSlots; j++ {
		switch {
		case !cls(j):
		case w(j) == 0:
			make1(j)
		case !flag:
			a = append(a, j)
		default:
			if id(j) >= ipItems {
				panic("itemPoolSelection: fixture holds 64 item definitions")
			}
			v := uint32(in[defs+int(id(j))])
			for _, t := range tri {
				if level >= t[0] && v >= t[1] && v < t[2] {
					a = append(a, j)
					break
				}
			}
		}
	}
	if int32(len(a)) < int32(k) && flag { // 0x4f8a31 signed compare
		a = a[:0]
		for j := 0; j < ipSlots; j++ {
			if cls(j) {
				a = append(a, j)
			}
		}
	}
	for _, j := range a {
		wsum += w(j)
	}
	for done := int64(0); k != 0 && done < k && len(a) > 0; {
		i := r.R(len(a))
		if r.R(wsum) <= w(a[i]) { // 0x4f8baf: reject when w < roll
			make1(a[i])
			a = append(a[:i], a[i+1:]...)
			done++
		}
	}
	return []int64{count, count, capa, int64(ordinal), mkCalls, mkCalls, lastID, 0, 0, 0}
}
