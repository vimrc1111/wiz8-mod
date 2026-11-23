package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// dialogue_greeting_status_reset 0x577290(sayFlag): greeting step of the dialogue box
// (build/wiz8.asm:507712..507900). G = *0x649f1c, D = *(G+0x1d4) (dialogue record),
// E = *(D+6) (definition record), W = *(G+0x1b0) (keyword widget).
//
// Flow (all conditions are inputs):
//  1. D kind byte (+0x2e) == 0x2a and D flag bytes +0x1a, +0x25 both set: 0x50a3c0 finds
//     the monster with id D+0x16 (list *0x683fad, then *0x683fb9; a miss asserts, panic
//     here) and 0x4c7f10(2 args: 0x619c54, 1) runs on it.
//  2. sayFlag != 0, D+0x1c != 0 and E+0x2f0 == 0 -> return; int32(G+0x25c) > 0 -> return.
//  3. D+0x1d == 0: post event 1 and return. Else post event 0 and clear D+0x1d.
//     Posting is 0x528830(kind,0,0,0): a 0x24-byte event {kind, ..., stamp *0x68c4ac at
//     +0x20} is the first heap block, appended to the queue (*0x68c4c8, count 0x68c4c0,
//     capacity 0x68c4c4); count+1 > capacity (signed) reallocates to count+1 words.
//  4. E+0x54, +0x2ea and +0x56 all zero: for each of the 8 party slots (byte slot i at
//     *0x685178 + 0x106*i non-zero) whose character record c = *0x685174 + 0x1862*i has
//     u32 +0xb11 > 0 and u32 +0xb01 < 0xf (unsigned), call 0x553f10(c, 0x16, 0xf, 0).
//  5. byte 0x683f97 != 0 and G+0x252 == 0: 0x571660(E+4, -1, 1); else E+0x54 == 0 and
//     (E+0x2ea == 0 or D+0x25 != 0): 0x5775d0(E+4, -1).
//
// 0x571660 and 0x5775d0 strip "!?@#$,.\":" (0x528f60; 0x571660 in place, 0x5775d0 on a
// copy), then pick a category bl by case-folding (A-Z only, 0x402920) compares: monster
// name list A (count *0x683f78, base *0x68516c, stride 0x10d) -> 0, list B (count
// *0x683f88, base *0x6836a0 + 4, stride 0x309) -> 1, keyword pointer array at 0x649f8c
// (skipped if the first is empty; stops at the first empty string) -> 1, UI list (count
// *0x649f20, strings *0x68c09c[*(0x649f24+4j)]) -> 2, else 3. 0x5775d0 reaches the UI
// list through 0x575020 (same loop).
//
// 0x571660 then runs 0x55e0c0(name, 0xffffff00|bl, 1) on W: an existing item whose text
// equals the name -> nothing more. Otherwise 0x5d16c0 appends a new item (0x64 bytes,
// vtable 0x5ef894, text = the empty string at 0x689b34 (BSS, never stored) + name,
// then laid out by 0x4f35b0, see dgsrWrap) to the item vector (count W+0x84,
// capacity W+0x88, data W+0x8c; growth to count+1 words), W+6 = 1, and 0x562a50 sets
// bit 31 of *(*0x68edcc + 0xf4) when *0x68ec78 == 7 and the pointer is non-zero. With more
// than one item the first item whose text equals the string at exe 0x647ba0 (" [No
// Keywords]") is removed (shift, count-1, destructor). Then, if G+0x1d8 is neither 0xff
// nor bl: G+0x1d8 = bl, 0x55e7c0(bl) and 0x575390(); 0x408860(0x64a0a8, 0) (the flag
// argument 1 of 0x571660 is tested, not bl: asm 0x5717dd reads the third argument),
// 0x55e2c0 and 0x55e1e0 follow. The getter 0x55e2b0 and the vtable[1] calls on W, G+0x134
// and G+0x138 only touch widget display state, which the node does not read back.
//
// 0x5775d0 searches the string vector (count G+0x1e0, capacity G+0x1e4, data G+0x1e8)
// for the stripped copy; if absent it allocates a zeroed 0xca-byte record, copies the
// string in, stores bl at +0xc8 and appends (growth to count+1 words).
//
// Emu address-stubs the D4/excluded callees 0x4c7f10, 0x408860, 0x55e7c0, 0x575390,
// 0x55e2c0, 0x55e1e0 and 0x553f10 (callers ignore eax; each returns 0x5a5a5a5a) and
// records call counts and arguments as outputs. The vtable slots point at a bare ret
// (0x41a861); wcscspn is the emu builtin, so the new item text goes through the real
// 0x4f35b0 layout (dgsrWrap; inputs layoutLeft/layoutRight 0..255, glyphAdvance/glyphPad
// 0..3). A removed item's destructor leaves vptr 0x5ed5b8 in its heap block.
// Heap: bump allocator from 0x50000000, size max(8, (n+7)&~7).
//
// Fixture caps (panic past them): queue count 0..3 and capacity 0..4; list A/B, UI, string
// vector and item vector counts 0..2 with capacity 0..4; monster id 0xffffff00..0xffffff03
// (A holds ..00, ..01 and B ..02, ..03); battle pointer 0 or the case "battle" object;
// every string used must end in a NUL within 16 characters.
//
// Not observed (unobserved plumbing): 0x4c7f10's ecx (monster record +0xc), the new
// item's body beyond its vtable word, the item vector's selection state (0x5d24a0,
// 0x5d21f0, 0x4f3cf0, L+0x4c/0x50/0x14/0x18), and the widget vtable calls.
const (
	dgsrArena   = 0x20000000
	dgsrHeap    = 0x50000000 // emu builtin bump heap
	dgsrQueue   = dgsrArena + 0xff00
	dgsrItem0   = dgsrArena + 0x10b00
	dgsrItem1   = dgsrArena + 0x10c00
	dgsrWData   = dgsrArena + 0x10a00
	dgsrVData   = dgsrArena + 0x10000
	dgsrVRec0   = dgsrArena + 0x10100
	dgsrVRec1   = dgsrArena + 0x10200
	dgsrBattle  = dgsrArena + 0x10400
	dgsrChars   = dgsrArena + 0x2000 // eight records, stride 0x1862
	dgsrStrLen  = 16                 // characters per fixture string
	dgsrMaxPart = 8
)

var dgsrIn, dgsrOut []string

var dgsrIsString = map[string]bool{}

func dgsrAppendStrings(dst []string, base string) []string {
	for w := 0; w < 4; w++ {
		dst = append(dst, base+strconv.Itoa(w))
		dgsrIsString[base+strconv.Itoa(w)] = true
	}
	return dst
}

func init() {
	n := []string{"sayFlag", "dlgKind", "dlgSoundID", "dlgSoundFlagA", "dlgSoundFlagB", "dlgHasDef", "dlgFirstGreeting",
		"defTalkLock", "defFlag54", "defFlag2ea", "defFlag56", "gameBusy", "gameFlag252", "gameLastCategory", "modeFlag683f97"}
	for _, p := range []string{"slotActive", "statusStrength", "statusCap"} {
		for i := 0; i < dgsrMaxPart; i++ {
			n = append(n, p+strconv.Itoa(i))
		}
	}
	n = append(n, "queueCount", "queueCapacity", "queueEntry0", "queueEntry1", "queueEntry2", "queueEntry3", "eventStamp")
	n = dgsrAppendStrings(n, "name")
	n = append(n, "listACount")
	n = dgsrAppendStrings(dgsrAppendStrings(n, "listA0"), "listA1")
	n = append(n, "listBCount")
	n = dgsrAppendStrings(dgsrAppendStrings(n, "listB0"), "listB1")
	n = dgsrAppendStrings(dgsrAppendStrings(n, "keyword0"), "keyword1")
	n = append(n, "uiCount")
	n = dgsrAppendStrings(dgsrAppendStrings(n, "uiName0"), "uiName1")
	n = append(n, "vecCount", "vecCapacity")
	n = dgsrAppendStrings(dgsrAppendStrings(n, "vecName0"), "vecName1")
	n = append(n, "widgetCount", "widgetCapacity")
	n = dgsrAppendStrings(dgsrAppendStrings(n, "widgetText0"), "widgetText1")
	n = append(n, "widgetKey", "layoutLeft", "layoutRight", "glyphAdvance", "glyphPad")
	n = append(n, "modeValue", "battlePtr", "battleFlags")
	dgsrIn = n
	o := []string{"eventKind", "eventStamp", "queueCount", "queueCapacity", "queuePointer",
		"queueEntry0", "queueEntry1", "queueEntry2", "queueEntry3", "dlgFirstGreeting", "statusCalls"}
	for i := 0; i < dgsrMaxPart; i++ {
		o = append(o, "statusTarget"+strconv.Itoa(i))
	}
	o = append(o, "statusCode", "statusValue", "statusExtra", "soundCalls", "soundArg1", "soundArg2",
		"hintCalls", "hintArg1", "hintArg2", "categoryCalls", "categoryArg", "refreshCalls", "drawCalls", "layoutCalls")
	o = dgsrAppendStrings(o, "name")
	o = append(o, "gameLastCategory", "widgetCount",
		"widgetEntry0", "widgetEntry1", "widgetEntry2", "widgetEntry3", "widgetCapacity", "widgetPointer", "widgetMarked", "battleFlags",
		"vecCount", "vecCapacity", "vecPointer", "vecEntry0", "vecEntry1", "vecEntry2", "vecEntry3",
		"vecGrown0", "vecGrown1", "vecGrown2", "vecGrown3", "heapWord28", "recordCategory")
	dgsrOut = o
	Nodes["00577290"] = Node{Addr: "00577290", In: dgsrIn, Out: dgsrOut, F: dialogueGreetingStatusResetNode}
}

// The "no keywords" placeholder item text lives at exe 0x647ba0 (UTF-16, NUL-terminated).
var dgsrPlaceholder, dgsrPlaceholderErr = dgsrLoadPlaceholder()

func dgsrLoadPlaceholder() ([]uint16, error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return nil, fmt.Errorf("dialogue greeting: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	off, err := peOffset(b, 0x647ba0)
	if err != nil {
		return nil, err
	}
	var s []uint16
	for ; off+2 <= len(b); off += 2 {
		c := binary.LittleEndian.Uint16(b[off:])
		if c == 0 {
			return s, nil
		}
		s = append(s, c)
	}
	return nil, fmt.Errorf("dialogue greeting: unterminated placeholder text")
}

// dgsrFold lowers A-Z only (0x402920).
func dgsrFold(c uint16) uint16 {
	if c >= 'A' && c <= 'Z' {
		return c + 0x20
	}
	return c
}

func dgsrFoldEqual(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if dgsrFold(a[i]) != dgsrFold(b[i]) {
			return false
		}
	}
	return true
}

type dgsrString [dgsrStrLen]uint16

// text returns the characters before the first NUL; a fixture string without NUL panics.
func (s *dgsrString) text() []uint16 {
	for i, c := range s {
		if c == 0 {
			return s[:i]
		}
	}
	panic("dialogue greeting: string without NUL in the first 16 characters")
}

// dgsrStrip removes ! ? @ # $ , . " : and copies the rest back over the string including
// the new NUL; characters after the new NUL keep their old values (0x528f60).
func dgsrStrip(s *dgsrString) {
	t := s.text()
	k := 0
	var tmp dgsrString
	for _, c := range t {
		switch c {
		case '!', '?', '@', '#', '$', ',', '.', '"', ':':
		default:
			tmp[k] = c
			k++
		}
	}
	copy(s[:k+1], tmp[:k+1])
}

type dgsrHeapMem struct {
	next int64
	mem  map[int64]byte
}

func (h *dgsrHeapMem) alloc(n int64) int64 {
	p := h.next
	h.next += max(8, (n+7)&^7)
	return p
}
func (h *dgsrHeapMem) wr(addr int64, size int, v int64) {
	for i := 0; i < size; i++ {
		h.mem[addr+int64(i)] = byte(v >> (8 * i))
	}
}
func (h *dgsrHeapMem) rd(addr int64, size int) int64 {
	var v int64
	for i := size - 1; i >= 0; i-- {
		v = v<<8 | int64(h.mem[addr+int64(i)])
	}
	return v
}

func dialogueGreetingStatusResetNode(in []int64, r Rand) []int64 {
	if len(in) != len(dgsrIn) {
		panic("dialogue greeting: wrong input count")
	}
	if dgsrPlaceholderErr != nil {
		panic(dgsrPlaceholderErr)
	}
	at := map[string]int{}
	for i, n := range dgsrIn {
		at[n] = i
	}
	v := func(n string) int64 { return in[at[n]] }
	for i, x := range in {
		n := dgsrIn[i]
		lo, hi := int64(0), int64(0xffffffff)
		switch {
		case dgsrIsString[n]:
			hi = 1<<63 - 1
		case dgsrIsByte[n]:
			hi = 255
		case dgsrIsSigned[n]:
			lo, hi = -0x80000000, 0x7fffffff
		}
		if x < lo || x > hi {
			panic("dialogue greeting: input out of field range")
		}
	}
	str := func(base string) dgsrString {
		var s dgsrString
		for w := 0; w < 4; w++ {
			x := v(base + strconv.Itoa(w))
			if w == 3 && x>>48 >= 0x8000 {
				panic("dialogue greeting: string character beyond fixture range")
			}
			for k := 0; k < 4; k++ {
				s[4*w+k] = uint16(x >> (16 * k))
			}
		}
		return s
	}
	limit := func(n string, hi int64) int64 {
		x := v(n)
		if x < 0 || x > hi {
			panic("dialogue greeting: " + n + " beyond the case fixture")
		}
		return x
	}
	qCount, qCap := limit("queueCount", 3), limit("queueCapacity", 4)
	listA, listB, uiCount := limit("listACount", 2), limit("listBCount", 2), limit("uiCount", 2)
	vecCount, vecCap := limit("vecCount", 2), limit("vecCapacity", 4)
	wCount, wCap := limit("widgetCount", 2), limit("widgetCapacity", 4)
	layoutL, layoutR := limit("layoutLeft", 255), limit("layoutRight", 255)
	glyphAdv, glyphPad := limit("glyphAdvance", 3), limit("glyphPad", 3)

	o := map[string]int64{}
	h := &dgsrHeapMem{next: dgsrHeap, mem: map[int64]byte{}}
	h.wr(dgsrWData, 4, dgsrItem0)
	h.wr(dgsrWData+4, 4, dgsrItem1)
	h.wr(dgsrVData, 4, dgsrVRec0)
	h.wr(dgsrVData+4, 4, dgsrVRec1)
	q := [4]int64{v("queueEntry0"), v("queueEntry1"), v("queueEntry2"), v("queueEntry3")}
	queuePtr := int64(dgsrQueue)
	first, lastCat, battleFlags := v("dlgFirstGreeting"), v("gameLastCategory"), v("battleFlags")
	var widgetMarked int64
	name := str("name")
	wData, vecData := int64(dgsrWData), int64(dgsrVData)

	finish := func() []int64 {
		o["eventKind"], o["eventStamp"] = h.rd(dgsrHeap, 4), h.rd(dgsrHeap+0x20, 4)
		o["queueCount"], o["queueCapacity"], o["queuePointer"] = qCount, qCap, queuePtr
		for i := int64(0); i < 4; i++ {
			s := strconv.Itoa(int(i))
			o["queueEntry"+s] = q[i]
			o["widgetEntry"+s] = h.rd(dgsrWData+4*i, 4)
			o["vecEntry"+s] = h.rd(dgsrVData+4*i, 4)
			o["vecGrown"+s] = h.rd(dgsrHeap+0xf8+4*i, 4)
			var x int64
			for k := int64(0); k < 4; k++ {
				x |= int64(name[4*i+k]) << (16 * k)
			}
			o["name"+s] = x
		}
		o["dlgFirstGreeting"], o["gameLastCategory"], o["battleFlags"] = first, lastCat, battleFlags
		o["widgetCount"], o["widgetCapacity"], o["widgetPointer"], o["widgetMarked"] = wCount, wCap, wData, widgetMarked
		o["vecCount"], o["vecCapacity"], o["vecPointer"] = vecCount, vecCap, vecData
		o["heapWord28"], o["recordCategory"] = h.rd(dgsrHeap+0x28, 4), h.rd(dgsrHeap+0xf0, 1)
		res := make([]int64, len(dgsrOut))
		for i, n := range dgsrOut {
			res[i] = o[n]
		}
		return res
	}
	call := func(counter string, args ...int64) { // address-stubbed callee: count and arguments
		o[counter+"Calls"]++
		for i, a := range args {
			o[counter+"Arg"+strconv.Itoa(i+1)] = a
		}
	}
	post := func(kind int64) { // 0x528830(kind, 0, 0, 0)
		ev := h.alloc(0x24)
		h.wr(ev, 4, kind)
		h.wr(ev+0x20, 4, v("eventStamp"))
		n := int32(qCount)
		if n+1 > int32(qCap) { // signed, 0x528880
			p := h.alloc(4 * int64(n+1))
			for i := int32(0); i < n; i++ {
				h.wr(p+4*int64(i), 4, q[i])
			}
			h.wr(p+4*int64(n), 4, ev)
			queuePtr, qCap = p, int64(n+1)
		} else {
			q[n] = ev
		}
		qCount++
	}
	// category search shared by 0x571660 and 0x5775d0 (bl); uiViaHelper is 0x575020.
	category := func(s *dgsrString) int64 {
		t := s.text()
		for i := int64(0); i < listA; i++ { // list A, strings at +0 of 0x10d-byte records
			e := str("listA" + strconv.Itoa(int(i)))
			if dgsrFoldEqual(t, e.text()) {
				return 0
			}
		}
		for i := int64(0); i < listB; i++ { // list B, strings at +4 of 0x309-byte records
			e := str("listB" + strconv.Itoa(int(i)))
			if dgsrFoldEqual(t, e.text()) {
				return 1
			}
		}
		kw := [3]*dgsrString{}
		for i := 0; i < 2; i++ { // keyword pointer array 0x649f8c, ends at the first empty string
			k := str("keyword" + strconv.Itoa(i))
			kw[i] = &k
		}
		for i := 0; i < 2 && kw[i][0] != 0; i++ {
			if dgsrFoldEqual(t, kw[i].text()) {
				return 1
			}
		}
		for i := int64(0); i < uiCount; i++ { // UI list: index array 0x649f24 holds 0, 1
			e := str("uiName" + strconv.Itoa(int(i)))
			if dgsrFoldEqual(t, e.text()) {
				return 2
			}
		}
		return 3
	}

	// 1. monster record for the dialogue (0x50a3c0) and its 0x4c7f10 call
	if v("dlgKind") == 0x2a && v("dlgSoundFlagA") != 0 && v("dlgSoundFlagB") != 0 {
		if id := v("dlgSoundID"); id < 0xffffff00 || id > 0xffffff03 {
			panic("dialogue greeting: monster id beyond the case fixture (lookup assert)")
		}
		call("sound", 0x619c54, 1)
	}
	// 2. early returns
	if v("sayFlag") != 0 && v("dlgHasDef") != 0 && v("defTalkLock") == 0 {
		return finish()
	}
	if int32(v("gameBusy")) > 0 {
		return finish()
	}
	// 3. event
	if first == 0 {
		post(1)
		return finish()
	}
	post(0)
	first = 0
	// 4. status loop (0x553f10 is address-stubbed; its eax is ignored)
	if v("defFlag54") == 0 && v("defFlag2ea") == 0 && v("defFlag56") == 0 {
		for i := int64(0); i < dgsrMaxPart; i++ {
			s := strconv.Itoa(int(i))
			if v("slotActive"+s) != 0 && v("statusStrength"+s) > 0 && v("statusCap"+s) < 0xf {
				k := o["statusCalls"]
				if k >= dgsrMaxPart {
					panic("dialogue greeting: more status calls than recorded")
				}
				o["statusTarget"+strconv.Itoa(int(k))] = dgsrChars + 0x1862*i
				o["statusCalls"]++
				o["statusCode"], o["statusValue"], o["statusExtra"] = 0x16, 0xf, 0
			}
		}
	}
	// 5. keyword registration
	if v("modeFlag683f97") != 0 && v("gameFlag252") == 0 { // 0x571660(E+4, -1, 1)
		if len(name.text()) == 0 {
			return finish()
		}
		dgsrStrip(&name)
		bl := category(&name)
		nt := name.text()
		texts := [3][]uint16{}
		for i := int64(0); i < wCount; i++ {
			t := str("widgetText" + strconv.Itoa(int(i)))
			texts[i] = t.text()
		}
		created := true
		for i := int64(0); i < wCount; i++ { // 0x55e0c0: duplicate item text
			if dgsrFoldEqual(texts[i], nt) {
				created = false
			}
		}
		if created {
			item := h.alloc(0x64)
			h.wr(item, 4, 0x5ef894)       // item vtable
			h.alloc(int64(2*len(nt) + 2)) // 0x5d1050: text buffer malloc(2*wcslen+2)
			texts[wCount] = dgsrWrap(nt, uint32(layoutR-layoutL), int32(glyphAdv+glyphPad))

			if wCount+1 > wCap { // 0x5d16c0 growth to count+1 words
				p := h.alloc(4 * (wCount + 1))
				for i := int64(0); i < wCount; i++ {
					h.wr(p+4*i, 4, h.rd(wData+4*i, 4))
				}
				wData, wCap = p, wCount+1
			}
			h.wr(wData+4*wCount, 4, item)
			wCount++
			widgetMarked = 1                                // 0x4f2f00
			if v("modeValue") == 7 && v("battlePtr") != 0 { // 0x562a50
				if v("battlePtr") != dgsrBattle {
					panic("dialogue greeting: battle pointer beyond the case fixture")
				}
				battleFlags |= 0x80000000
			}
			if wCount > 1 { // drop the " [No Keywords]" placeholder item
				for i := int64(0); i < wCount; i++ {
					if dgsrFoldEqual(texts[i], dgsrPlaceholder) {
						h.wr(h.rd(wData+4*i, 4), 4, 0x5ed5b8) // item destructor resets the vptr to the base class
						for j := i; j < wCount-1; j++ {       // 0x5d1820
							h.wr(wData+4*j, 4, h.rd(wData+4*j+4, 4))
						}
						wCount--
						break
					}
				}
			}
			if lastCat != 0xff && lastCat != bl {
				lastCat = bl
				o["categoryCalls"]++ // 0x55e7c0(bl)
				o["categoryArg"] = bl
				o["refreshCalls"]++ // 0x575390
			}
			call("hint", 0x64a0a8, 0) // third argument (1) tested at 0x5717dd, not bl
			o["drawCalls"]++
			o["layoutCalls"]++
		}
		return finish()
	}
	if v("defFlag54") != 0 || v("defFlag2ea") != 0 && v("dlgSoundFlagB") == 0 {
		return finish()
	}
	// 0x5775d0(E+4, -1): works on a stripped copy; the definition name is unchanged
	buf := name
	dgsrStrip(&buf)
	bl := category(&buf)
	bt := buf.text()
	for i := int64(0); i < vecCount; i++ {
		e := str("vecName" + strconv.Itoa(int(i)))
		if dgsrFoldEqual(e.text(), bt) {
			return finish()
		}
	}
	rec := h.alloc(0xca)
	for i, c := range bt {
		h.wr(rec+2*int64(i), 2, int64(c))
	}
	h.wr(rec+0xc8, 1, bl)
	if vecCount+1 > vecCap {
		p := h.alloc(4 * (vecCount + 1))
		for i := int64(0); i < vecCount; i++ {
			h.wr(p+4*i, 4, h.rd(vecData+4*i, 4))
		}
		vecData, vecCap = p, vecCount+1
	}
	h.wr(vecData+4*vecCount, 4, rec)
	vecCount++
	return finish()
}

var dgsrIsByte = map[string]bool{"sayFlag": true, "dlgKind": true, "dlgSoundFlagA": true, "dlgSoundFlagB": true, "dlgHasDef": true,
	"dlgFirstGreeting": true, "defTalkLock": true, "defFlag54": true, "defFlag2ea": true, "defFlag56": true, "gameFlag252": true,
	"gameLastCategory": true, "modeFlag683f97": true,
	"slotActive0": true, "slotActive1": true, "slotActive2": true, "slotActive3": true,
	"slotActive4": true, "slotActive5": true, "slotActive6": true, "slotActive7": true}

var dgsrIsSigned = map[string]bool{"gameBusy": true, "queueCount": true, "queueCapacity": true, "uiCount": true, "vecCount": true,
	"vecCapacity": true, "widgetCount": true, "widgetCapacity": true}

// dgsrWrap is 0x4f35b0 on the new item text: split at ' ' and '\n' (wcscspn set at
// 0x617c88); a segment that does not fit (width + line so far >= limit, unsigned) turns the
// previous delimiter into '\n'; every other delimiter keeps its original character (' ' or '\n'). Width is 0x407010 with
// the fixture font (every character maps to glyph 0): len*(advance+pad), truncated to
// int16. limit is item rect +0xc minus +0x4, both copied from the list object at W+0x64.
func dgsrWrap(name []uint16, limit uint32, charW int32) []uint16 {
	t := append([]uint16{}, name...)
	width := func(n int) uint32 { return uint32(int32(int16(int32(n) * charW))) }
	space, line, prev, start := width(1), uint32(0), -1, 0
	for {
		e := start
		for e < len(t) && t[e] != 0x20 && t[e] != 0x0a {
			e++
		}
		w := width(e - start)
		fits := w+line < limit
		if e == len(t) {
			if !fits && prev >= 0 {
				t[prev] = 0x0a
			}
			return t
		}
		if fits {
			line += w + space
			prev = e
		} else {
			if prev >= 0 {
				t[prev] = 0x0a
			}
			line, prev = space+w, -1
		}
		start = e + 1
	}
}
