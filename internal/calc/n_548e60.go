package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// 0x548e60 potion_creation(C): a character C who is alive, has C+0xb01 < 0x11 and
// owns class/race feature 0x12 (0x547940: tables 0x61507c rows of 3 for C+0x69, then
// 0x615130 rows of 5 for C+0x71; -1 = no row) makes one potion. The potion id is
// the (R(n)+1)-th row of the level table 0x616fe0 (id, min, max; max -1 = open)
// whose range holds the signed level C+0xb9, n = number of such rows. The new item
// instance (12 bytes on the stack) is filled by 0x51c020 (stub: id, 1, 1 args),
// its quantity byte is capped at 1, the message 0x58aad0 is posted, 0x51c280 puts the
// instance in the first free backpack slot C+0x1029+12k (0x51c300), else at the front
// of the shared pack 0x685191 (0x521ef0). C+0xb65 = 0x168 whenever a potion was made.
//
// Backpack slot: the instance is copied (byte +0xa cleared), the source becomes
// id -1 and zero, 0x520d10 runs on source and slot, 0x4edc60 re-sums the 20 carried
// weights (0x51bfd0) into C+0xbb9 and warns (0x52e690) when the old weight was
// below the capacity C+0xbc5 and the new one exceeds it (unsigned), 0x4edd20 follows
// when the weight changed, 0x5227d0 warns (and sets slot byte +9 bit 0) when the potion
// definition weight / C+0x89 exceeds 0x1f4 and slot byte +6 is set. Mode 6 sets
// bit 2<<k (0x4edc60 path) or 0x7fc00000 (pack path) in the bits word.
// Pack path: with kind 1 (+0x66) the pack is scanned by 0x51f900, a pack of 0x1f4
// or more takes nothing, else the pack shifts up one slot and 0x5227d0 draws R(8)
// (0x518150) unless the mode is 5.
//
// Callees: 0x517ec0, 0x547940, 0x51c280, 0x51c300, 0x521ef0, 0x51fe30, 0x4edc60,
// 0x5227d0, 0x522640, 0x51b5c0, 0x518150 run natively in the case and are modelled
// here. 0x51c020, 0x51bfd0, 0x51d610, 0x51f900, 0x4edd20, 0x520d10 (D4 formulas) are
// address stubs in the case: weights, usability and merge result come from the
// inputs (0x51f900: one result per call, inputs m0..m19 for the 0x51c300 scan of 12 worn then
// 8 backpack slots, p0..p5 for the first pack slots of the 0x521ef0 scan; only the low byte
// counts and the first nonzero one merges into that slot: no move, the weight/mode tail runs
// on slot C+0x1029+12k with k the scan index, worn k 8..11 land in four decoy slots behind the
// backpack; a pack merge refreshes, sets the mode bits and draws R(8)), the last arguments and the call counts of
// the stubs are outputs, as are 0x58aad0 and 0x52e690 (registered 排除). The real
// Go callees are not composed (50baf0 precedent): the instance is a stack address
// outside their fixtures. Display strings (0x51b5c0 name cache) are not modelled.
//
// Fixture facts (re/emu/cases/00548e60.json): C is one of the 8 roster records
// (stride 0x1862; who selects it, the other seven hold decoys), roster byte +4 is 1,
// the bits word starts at 0x401, backpack and pack tail words are the fixed
// patterns below, every table item has the same kind and weight. Caps (calc
// panics past them): slot item ids -1..0x1d1 on merges (0x5227d0 special ids start at 0x1d2),
// pack scan results past p5 are zero, pack count 0..5 or 0x1f4..0x1f5 on the pack path, level >= 1,
// C+0x69 -1..14, C+0x71 -1..15. C+0x89 == 0 on the 0x5227d0 path is a #DE and
// panics. Constants (messages, tables, 0x5ee5a8 ...) are read from the player's exe.
const (
	potionChars    = 0x20001000 // case object "chars" (arena 0x20000000 + 0x1000)
	potionStride   = 0x1862
	potionInst     = 0x10007ff4 // stack instance of the case
	potionMsgBuf   = 0x68c0b4   // buffer returned by 0x51b5c0
	potionPack     = 0x685191
	potionBitsIn   = 0x401
	potionPackMax  = 5
	potionPackFull = 0x1f4
)

var potionIn = strings.Fields(`
	who C alive0 alive1 alive2 alive3 alive4 alive5 alive6 alive7 lvlB01_0 lvlB01_1 lvlB01_2 lvlB01_3
	lvlB01_4 lvlB01_5 lvlB01_6 lvlB01_7 level c69 c71 c89 wOld wCap c65 bk0 bk1 bk2 bk3 bk4 bk5 bk6 bk7
	mode v558 ctx60c packN pk0 pk1 pk2 pk3 pk4 pk5 kind weight r51d i4 i5 i6 i7 i8 i9 i10 i11 wFirst
	wWorn wBack0 wBack m0 m1 m2 m3 m4 m5 m6 m7 m8 m9 m10 m11 m12 m13 m14 m15 m16 m17 m18 m19 p0 p1 p2 p3 p4 p5`)

var potionOut = strings.Fields(`
	c65Out rcId rcA3 rcA4 rcA5 rcInst rcCalls msgA1 msgA2 msgA3 msgA4 msgCalls warnA1 warnA2 warnA3
	warnA4 warnA5 warnCalls refreshCalls resetA1 resetA2 resetA3 resetCalls weighCalls usableCalls
	mergeCalls inst0 inst4 inst8 wNew bits packNOut pk0_0 pk0_4 pk0_8 pk1_0 pk1_4 pk1_8 pk2_0 pk2_4
	pk2_8 pk3_0 pk3_4 pk3_8 pk4_0 pk4_4 pk4_8 pk5_0 pk5_4 pk5_8 pk6_0 pk6_4 pk6_8 bk0_0 bk0_4 bk0_8
	bk1_0 bk1_4 bk1_8 bk2_0 bk2_4 bk2_8 bk3_0 bk3_4 bk3_8 bk4_0 bk4_4 bk4_8 bk5_0 bk5_4 bk5_8 bk6_0
	bk6_4 bk6_8 bk7_0 bk7_4 bk7_8 bk8_0 bk8_4 bk8_8 bk9_0 bk9_4 bk9_8 bk10_0 bk10_4 bk10_8 bk11_0 bk11_4 bk11_8
`)

func init() {
	Nodes["00548e60"] = Node{Addr: "00548e60", In: potionIn, Out: potionOut, F: potionCreationNode}
}

type potionExeData struct {
	table                      [][3]int32
	msgWeight, msgOver, a4, a5 int64 // 0x5ee5a8, 0x5ee664, 0x5ed8c8, 0x5ed914
}

var potionExe, potionExeErr = loadPotionExe()

func loadPotionExe() (d potionExeData, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return d, fmt.Errorf("potion creation: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return d, err
	}
	get := func(va uint32) (int32, error) {
		o, err := peOffset(b, va)
		if err != nil || len(b)-o < 4 {
			return 0, fmt.Errorf("potion creation: exe value %#x unreadable", va)
		}
		return int32(binary.LittleEndian.Uint32(b[o:])), nil
	}
	for va := uint32(0x616fe0); ; va += 12 {
		var e [3]int32
		for i := range e {
			if e[i], err = get(va + 4*uint32(i)); err != nil {
				return d, err
			}
		}
		if e[0] == -1 {
			break
		}
		if len(d.table) > 64 {
			return d, fmt.Errorf("potion creation: level table not terminated")
		}
		d.table = append(d.table, e)
	}
	for i, va := range []uint32{0x5ee5a8, 0x5ee664, 0x5ed8c8, 0x5ed914} {
		x, err := get(va)
		if err != nil {
			return d, err
		}
		*[]*int64{&d.msgWeight, &d.msgOver, &d.a4, &d.a5}[i] = int64(x)
	}
	return d, nil
}

// potionFeature is 0x547940(C, code): class row (3) then race row (5).
func potionFeature(c69, c71 int64, code int32) bool {
	if c69 < -1 || c69 > 14 || c71 < -1 || c71 > 15 {
		panic("potion creation: class/race index outside the exe feature tables")
	}
	if c69 != -1 {
		for _, f := range breathTables.class[c69] {
			if f == code {
				return true
			}
		}
	}
	if c71 != -1 {
		for _, f := range breathTables.race[c71] {
			if f == code {
				return true
			}
		}
	}
	return false
}

// potionSlot is a 12-byte item record: id, then words at +4 and +8.
type potionSlot struct{ id, d4, d8 int64 }

func potionCreationNode(in []int64, r Rand) []int64 {
	if len(in) != len(potionIn) {
		panic("potion creation: wrong input count")
	}
	if potionExeErr != nil {
		panic(potionExeErr)
	}
	if breathTablesErr != nil {
		panic(breathTablesErr)
	}
	v := map[string]int64{}
	for i, n := range potionIn {
		lo, hi := int64(0), int64(0xffffffff)
		switch {
		case strings.HasPrefix(n, "alive") || (n[0] == 'i' && len(n) <= 3):
			hi = 255
		case n == "level" || n == "c69" || n == "c71" || strings.HasPrefix(n, "bk") || strings.HasPrefix(n, "pk"):
			lo, hi = -1<<31, 1<<31-1
		}
		if in[i] < lo || in[i] > hi {
			panic("potion creation: input " + n + " outside its field width")
		}
		v[n] = in[i]
	}
	who, C := v["who"], v["C"]
	if who < 0 || who > 7 || C != potionChars+who*potionStride {
		panic("potion creation: C is not a roster record of the case")
	}
	o := map[string]int64{}
	set := func(n string, x int64) {
		for _, k := range potionOut {
			if k == n {
				o[n] = x
				return
			}
		}
		panic("potion creation: unknown output " + n)
	}
	w := strconv.Itoa(int(who))
	var bk [12]potionSlot // 8 backpack slots, then the 4 record bytes after them (a worn-slot merge index 8..11 lands there)
	for k := range bk {
		id := 100 + 10*who + int64(k)
		if k < 8 {
			id = v["bk"+strconv.Itoa(k)]
		}
		bk[k] = potionSlot{id, 0xAAAA0000 + 256*who + int64(16*k), 0xBBBB0000 + 256*who + int64(16*k)}
	}
	var pk [7]potionSlot
	for j := range pk {
		pk[j] = potionSlot{0, 0xCCCC0000 + int64(16*j), 0xDDDD0000 + int64(16*j)}
		if j < 6 {
			pk[j].id = v["pk"+strconv.Itoa(j)]
		}
	}
	inst := potionSlot{0, v["i4"] | v["i5"]<<8 | v["i6"]<<16 | v["i7"]<<24, v["i8"] | v["i9"]<<8 | v["i10"]<<16 | v["i11"]<<24}
	c65, wNew, bits, packN := v["c65"], v["wOld"], int64(potionBitsIn), v["packN"]
	if v["alive"+w] != 0 && v["lvlB01_"+w] < 0x11 && potionFeature(v["c69"], v["c71"], 0x12) {
		L := int32(v["level"])
		var match []int64
		for _, e := range potionExe.table {
			if L >= e[1] && (e[2] == -1 || L <= e[2]) {
				match = append(match, int64(e[0]))
			}
		}
		if len(match) == 0 {
			panic("potion creation: level outside the table domain (no potion row)")
		}
		pick := r.R(len(match))
		if pick < 0 || pick >= len(match) {
			panic("potion creation: R(n) outside 0..n-1")
		}
		inst.id = match[pick]
		set("rcId", inst.id)
		set("rcA4", 1)
		set("rcA5", 1)
		set("rcInst", potionInst)
		set("rcCalls", 1)
		if inst.d4&0xff > 1 {
			inst.d4 = inst.d4&^0xff | 1
		}
		set("msgA1", who)
		set("msgA2", v["ctx60c"])
		set("msgA3", C+5)
		set("msgA4", potionMsgBuf)
		set("msgCalls", 1)
		var merge, refresh, warns, weighs, usable, resets int64
		warn := func(a2 int64) {
			set("warnA1", C)
			set("warnA2", a2)
			set("warnA3", 0)
			set("warnA4", potionExe.a4)
			set("warnA5", potionExe.a5)
			warns++
		}
		reset := func(a1 int64) { // 0x51fe30 runs 0x520d10 on the source, then on the destination
			set("resetA1", a1)
			set("resetA2", 0)
			set("resetA3", 1)
			resets += 2
		}
		moved := potionSlot{inst.id, inst.d4, inst.d8 &^ 0xff0000}
		mode := v["mode"]
		slotWeight := func(id int64) int64 { // definition weight +0x86 of the slot's item
			for _, e := range potionExe.table {
				if int64(e[0]) == id {
					return v["weight"]
				}
			}
			if id == -1 {
				return 0 // definition -1 lies in the zeroed arena gap below the definitions
			}
			if id < 0 || id > 0x1d1 {
				panic("potion creation: slot item id beyond the fixture cap")
			}
			return 0x7777 + id // the decoy definitions of the case
		}
		// 0x51c44f: the weight/mode tail shared by a free-slot placement and a merge
		// into worn/backpack slot k (the slot address is C+0x1029+12k for both).
		tail := func(k int) {
			sum := (v["wFirst"] + 11*v["wWorn"] + v["wBack0"] + 7*v["wBack"]) & 0xffffffff
			weighs = 20
			if v["wOld"] < v["wCap"] && sum > v["wCap"] {
				m := v["v558"]
				if mode == 6 {
					m = potionExe.msgWeight
				}
				warn(m)
			}
			wNew = sum
			if sum != v["wOld"] {
				refresh++
			}
			if mode == 6 {
				bits |= 2 << uint(k)
			}
			if mode != 5 && bk[k].d8&0x100 == 0 { // 0x5227d0(slot, 0, C)
				usable++
				if v["r51d"]&0xff != 0 {
					if v["c89"] == 0 {
						panic("potion creation: C+0x89 == 0 divides by zero (#DE)")
					}
					if slotWeight(bk[k].id)/v["c89"] > 0x1f4 && bk[k].d4>>16&0xff != 0 {
						warn(potionExe.msgOver)
						bk[k].d8 |= 0x100
					}
				}
			}
		}
		// 0x51f900 merge scan, one stub result per call: the sequence m0..m19 covers the
		// 12 worn then 8 backpack slots of 0x51c300, p0..p5 the first pack slots of
		// 0x521ef0; a nonzero low byte stops the scan. Later pack entries are zero.
		hit := -1
		if v["kind"] == 1 {
			for i := 0; i < 20; i++ {
				merge++
				if v["m"+strconv.Itoa(i)]&0xff != 0 {
					hit = i
					if i >= 12 {
						hit = i - 12
					}
					break
				}
			}
		}
		packHit := func() int {
			if v["kind"] != 1 {
				return -1
			}
			for j := int64(0); j < packN; j++ {
				merge++
				if j < 6 && v["p"+strconv.FormatInt(j, 10)]&0xff != 0 {
					return int(j)
				}
			}
			return -1
		}
		packMerge := func(j int) { // 0x522089 -> 0x521f7f: refresh, mode bits, 0x5227d0(slot, 1, 0)
			if pk[j].id < -1 || pk[j].id > 0x1d1 {
				panic("potion creation: pack item id beyond the fixture cap")
			}
			refresh++
			if mode == 6 {
				bits |= 0x7fc00000
			}
			if mode != 5 {
				r.R(8)
			}
		}
		free := -1
		for k := 0; k < 8; k++ {
			if bk[k].id == -1 {
				free = k
				break
			}
		}
		switch {
		case hit >= 0:
			tail(hit)
		case free >= 0:
			bk[free] = moved
			inst = potionSlot{-1, 0, 0}
			reset(C + 0x1029 + 12*int64(free))
			tail(free)
		case packN >= potionPackFull:
			if packN > potionPackFull+1 {
				panic("potion creation: pack count beyond the fixture cap")
			}
			if j := packHit(); j >= 0 {
				packMerge(j)
			}
		default:
			if packN > potionPackMax {
				panic("potion creation: pack count beyond the fixture cap")
			}
			if j := packHit(); j >= 0 {
				packMerge(j)
				break
			}
			copy(pk[1:packN+1], pk[:packN])
			pk[0] = moved
			inst = potionSlot{-1, 0, 0}
			reset(potionPack)
			packN++
			refresh++
			if mode == 6 {
				bits |= 0x7fc00000
			}
			if mode != 5 {
				r.R(8) // 0x5227d0 -> 0x518150: one draw, the result changes nothing
			}
		}
		c65 = 0x168
		set("weighCalls", weighs)
		set("usableCalls", usable)
		set("mergeCalls", merge)
		set("refreshCalls", refresh)
		set("warnCalls", warns)
		set("resetCalls", resets)
	}
	set("c65Out", c65)
	set("inst0", inst.id&0xffffffff)
	set("inst4", inst.d4)
	set("inst8", inst.d8)
	set("wNew", wNew)
	set("bits", bits)
	set("packNOut", packN)
	for j, s := range pk {
		set(fmt.Sprintf("pk%d_0", j), s.id)
		set(fmt.Sprintf("pk%d_4", j), s.d4)
		set(fmt.Sprintf("pk%d_8", j), s.d8)
	}
	for k, s := range bk {
		set(fmt.Sprintf("bk%d_0", k), s.id)
		set(fmt.Sprintf("bk%d_4", k), s.d4)
		set(fmt.Sprintf("bk%d_8", k), s.d8)
	}
	out := make([]int64, len(potionOut))
	for i, n := range potionOut {
		out[i] = o[n]
	}
	return out
}
