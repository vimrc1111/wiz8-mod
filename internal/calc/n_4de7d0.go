package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// quest_status_target_select 0x4de7d0 (build/wiz8.asm:298525..298650), no arguments.
//
// Priority per party member m (character record *0x685174, stride 0x1862; presence
// record *0x685178, stride 0x106): 10 when the presence byte is 0 or m < 2, else by the
// class dword +0x69 through the switch at 0x4de80b: classes 0..14 index the byte table
// 0x4de930, which picks a target from the dword table 0x4de91c (both read from the
// player's exe); the targets store immediates (0x4de812 -> 2, 0x4de81a -> 3, 0x4de822
// -> 4, 0x4de82a -> 5, 0x4de832 -> 1). Class above 14 -> 1. With the shipped exe:
// Fighter 0 -> 2, Rogue 7 -> 3, Mage 14 -> 4, Priest 10 -> 5, others 1. The lowest priority p wins; p == 10 returns without
// effect. Otherwise k = R(count of members at p) + 1 picks the k-th of them in member
// order; it is stored at 0x68760c, then 0x522fe0(m, 0x13, 0x270f, 0, 0, 1)
// (character_condition_apply, D4) and 0x5061a0(0x33, 1, 0) (story fact 0x33 := 1).
// The fallback scan at 0x4de8b6 (count 0) is unreachable: the minimum is always held
// by at least one member (notes/re/cov-exempt.d/4de7d0.tsv); calc panics there.
//
// 0x5061a0 runs natively in the case: fact byte 0x689b78+0x33 := 1; when it changed,
// 0x5588f0(0x33) appends (map *0x686a70, 0x33, value) to the event log *0x68de40
// (count +4, capacity +8, buffer +0xc of 12-byte entries; a full log is reallocated
// to count+1 on the emulator's bump heap 0x50000000 and the old entries are copied).
// The value is the low byte of the first 0x506280(0x33) return; the upper three
// bytes are the stack slot's residue: the return address 0x5061f1 of 0x5061a0's
// sprintf call when 0x33 < *0x683f8c (signed), else the residue of earlier stack use
// below the caller (0 in the emulator; calc returns 0, see the final report). Unless
// byte 0x68de44 is set, the journal notice follows: threshold 2/1/0 for
// *0x6850d5 = 0/1/2, else 0x33; when the signed byte +0x37 of the 6836ac entry 0x33
// (stride 0x1d8) is not above it, a second 0x506280(0x33) picks the word +0x38 (non-zero
// low byte) or +0x100; non-zero word and battle object *0x68edcc non-null: byte
// 0x683f97 set -> 0x5289b0(8, 0) (0x24-byte message, +0 = -1, +0xc kind, +0x10 arg,
// +0x20 = *0x68c4ac, appended to the queue 0x68c4c0/c4/c8 grown like the event log),
// else 0x58ac00(3, *(*0x68c09c+0x1d28), 2, obj+0x2d4 - obj+0x2cc, 0) (0x58fb30
// inlined) and 0x408860(0x61c5cc, 0). Then 0x506670(0x33, 1), and when byte 0x688290
// is set 0x58aad0(5, 0x619a38, entry+4) (debug log; the wcscpy/sprintf texts into
// 0x5061a0's locals are not outputs).
//
// Emu address-stubs 0x522fe0 and 0x506670 (D4) and 0x506280, 0x58aad0, 0x58ac00,
// 0x408860 (排除); returns come from case inputs and nothing after the calls reads
// the D4 callees' returns or writes, so F shares this body. Recorded arguments and
// counts are outputs.
//
// Fixture caps (calc panics past them): event-log and queue buffers are the
// fixtures (4 entries each) or the heap; heap output window 128 bytes.
const (
	qtsArena = 0x20000000
	qtsLvl   = qtsArena + 0x10000
	qtsEvBuf = qtsArena + 0xE900
	qtsQBuf  = qtsArena + 0xEA00
	qtsHeap  = 0x50000000
	qtsHeapN = 32 // dwords
	qtsFact  = 0x33
	qtsFixN  = 4
)

var qtsIn = func() []string {
	var s []string
	for m := 0; m < 8; m++ {
		s = append(s, fmt.Sprintf("pres%d", m))
	}
	for m := 0; m < 8; m++ {
		s = append(s, fmt.Sprintf("cls%d", m))
	}
	s = append(s, "fact", "fcnt", "dbg", "evCnt", "evCap")
	for k := 0; k < 12; k++ {
		s = append(s, fmt.Sprintf("ev%d", k))
	}
	s = append(s, "map", "dis", "lvl37", "lvl38", "lvl100", "r280a", "r280b", "jflag", "batPtr", "qmode", "qCnt", "qCap")
	for k := 0; k < 4; k++ {
		s = append(s, fmt.Sprintf("q%d", k))
	}
	return append(s, "qStamp", "b2cc", "b2d4", "t1d28", "r522", "r670", "sel")
}()

var qtsOut = func() []string {
	s := []string{"sel", "n522"}
	for a := 1; a <= 6; a++ {
		s = append(s, fmt.Sprintf("c522_%d", a))
	}
	s = append(s, "fact", "n670", "c670_1", "c670_2", "n280", "c280_0", "c280_1", "naad", "caad_1", "caad_2", "caad_3", "nac0")
	for a := 1; a <= 5; a++ {
		s = append(s, fmt.Sprintf("cac0_%d", a))
	}
	s = append(s, "n886", "c886_1", "c886_2", "evCnt", "evCap", "evPtr")
	for k := 0; k < 12; k++ {
		s = append(s, fmt.Sprintf("ev%d", k))
	}
	s = append(s, "qCnt", "qCap", "qPtr")
	for k := 0; k < 4; k++ {
		s = append(s, fmt.Sprintf("q%d", k))
	}
	for k := 0; k < qtsHeapN; k++ {
		s = append(s, fmt.Sprintf("heap%d", k))
	}
	return s
}()

var qtsClassPrio, qtsClassErr = qtsLoadClassPrio()

// qtsLoadClassPrio resolves the class switch (index bytes 0x4de930, dword targets
// 0x4de91c) from the player's exe; each target block stores one immediate.
func qtsLoadClassPrio() ([15]uint32, error) {
	var t [15]uint32
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, src, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("quest_status_target_select: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(src), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	oi, err := peOffset(b, 0x4de930)
	if err != nil {
		return t, err
	}
	ot, err := peOffset(b, 0x4de91c)
	if err != nil {
		return t, err
	}
	if len(b)-oi < 15 || len(b)-ot < 20 {
		return t, fmt.Errorf("quest_status_target_select: truncated class switch")
	}
	imm := map[uint32]uint32{0x4de812: 2, 0x4de81a: 3, 0x4de822: 4, 0x4de82a: 5, 0x4de832: 1}
	for c := range t {
		i := b[oi+c]
		if i > 4 {
			return t, fmt.Errorf("quest_status_target_select: class switch index %d", i)
		}
		p, ok := imm[binary.LittleEndian.Uint32(b[ot+4*int(i):])]
		if !ok {
			return t, fmt.Errorf("quest_status_target_select: unknown class switch target")
		}
		t[c] = p
	}
	return t, nil
}

func init() {
	Nodes["004de7d0"] = Node{Addr: "004de7d0", In: qtsIn, Out: qtsOut, F: questStatusTargetSelectNode}
}

// qtsMem holds the fixture buffers and the heap window as dword cells.
type qtsMem struct {
	ev   [12]uint32
	q    [qtsFixN]uint32
	heap [qtsHeapN]uint32
	top  uint32
}

func (m *qtsMem) cell(a uint32) *uint32 {
	switch {
	case a >= qtsEvBuf && a < qtsEvBuf+48 && a%4 == 0:
		return &m.ev[(a-qtsEvBuf)/4]
	case a >= qtsQBuf && a < qtsQBuf+4*qtsFixN && a%4 == 0:
		return &m.q[(a-qtsQBuf)/4]
	case a >= qtsHeap && a < qtsHeap+4*qtsHeapN && a%4 == 0:
		return &m.heap[(a-qtsHeap)/4]
	}
	panic(fmt.Sprintf("quest_status_target_select: access %#x outside the fixture buffers", a))
}

func (m *qtsMem) alloc(n uint32) uint32 {
	a := qtsHeap + m.top
	step := (n + 7) &^ 7
	if step < 8 {
		step = 8
	}
	m.top += step
	if m.top > 4*qtsHeapN {
		panic("quest_status_target_select: heap above the output window")
	}
	return a
}

// grow is the (count, capacity, buffer) append of 0x5588f0 / 0x5289b0: entries of
// w dwords, reallocated to count+1 when count+1 > capacity (signed).
func (m *qtsMem) grow(cnt, capa, buf *uint32, w uint32, e []uint32) {
	c := int32(*cnt)
	if c+1 > int32(*capa) {
		nb := m.alloc(uint32(c+1) * w * 4)
		for i := uint32(0); int32(i) < c; i++ {
			for j := uint32(0); j < w; j++ {
				*m.cell(nb + 4*(w*i+j)) = *m.cell(*buf + 4*(w*i+j))
			}
		}
		*buf, *capa = nb, uint32(c+1)
	}
	for j, v := range e {
		*m.cell(*buf + 4*(w*uint32(c)+uint32(j))) = v
	}
	*cnt = uint32(c + 1)
}

func questStatusTargetSelectNode(in []int64, r Rand) []int64 {
	if len(in) != len(qtsIn) {
		panic("quest_status_target_select: wrong input count")
	}
	v := map[string]int64{}
	for i, n := range qtsIn {
		v[n] = in[i]
	}
	u := func(n string) uint32 { return uint32(v[n]) }
	o := map[string]int64{}
	for _, n := range []string{"sel", "fact", "evCnt", "evCap", "qCnt", "qCap"} {
		o[n] = int64(u(n))
	}
	o["evPtr"], o["qPtr"] = qtsEvBuf, qtsQBuf
	var mem qtsMem
	for k := range mem.ev {
		mem.ev[k] = u(fmt.Sprintf("ev%d", k))
	}
	for k := range mem.q {
		mem.q[k] = u(fmt.Sprintf("q%d", k))
	}

	var prio [8]uint32
	best := uint32(10)
	for m := 0; m < 8; m++ {
		prio[m] = 10
		if m >= 2 && u(fmt.Sprintf("pres%d", m))&0xff != 0 {
			prio[m] = 1
			if c := u(fmt.Sprintf("cls%d", m)); c <= 14 {
				if qtsClassErr != nil {
					panic(qtsClassErr)
				}
				prio[m] = qtsClassPrio[c]
			}
		}
		if prio[m] < best {
			best = prio[m]
		}
	}
	if best != 10 {
		cnt := 0
		for _, p := range prio {
			if p == best {
				cnt++
			}
		}
		if cnt == 0 {
			panic("quest_status_target_select: fallback scan 0x4de8b6 (unreachable)")
		}
		k := r.R(cnt) + 1
		sel := 0
		for ; sel < 8; sel++ {
			if prio[sel] == best {
				if k--; k == 0 {
					break
				}
			}
		}
		o["sel"] = int64(sel)
		o["n522"] = 1
		for a, x := range []int64{int64(sel), 0x13, 0x270f, 0, 0, 1} {
			o[fmt.Sprintf("c522_%d", a+1)] = x
		}
		qtsSetFact(u, o, &mem)
	}

	out := make([]int64, 0, len(qtsOut))
	for _, n := range qtsOut {
		switch {
		case len(n) > 2 && n[:2] == "ev" && n[2] >= '0' && n[2] <= '9':
			var k int
			fmt.Sscanf(n[2:], "%d", &k)
			out = append(out, int64(mem.ev[k]))
		case len(n) == 2 && n[0] == 'q':
			out = append(out, int64(mem.q[n[1]-'0']))
		case len(n) > 4 && n[:4] == "heap":
			var k int
			fmt.Sscanf(n[4:], "%d", &k)
			out = append(out, int64(mem.heap[k]))
		default:
			out = append(out, o[n])
		}
	}
	return out
}

// qtsSetFact is 0x5061a0(0x33, 1, 0).
func qtsSetFact(u func(string) uint32, o map[string]int64, mem *qtsMem) {
	old := u("fact") & 0xff
	o["fact"] = 1
	if old != 1 { // 0x5588f0(0x33)
		o["n280"] = 1
		o["c280_0"] = qtsFact
		cnt, capa, buf := u("evCnt"), u("evCap"), uint32(qtsEvBuf)
		val := u("r280a") & 0xff
		if int32(u("fcnt")) > qtsFact {
			val |= 0x506100 // residue: return address 0x5061f1 of the sprintf call at 0x5061eb
		}
		mem.grow(&cnt, &capa, &buf, 3, []uint32{u("map"), qtsFact, val})
		o["evCnt"], o["evCap"], o["evPtr"] = int64(cnt), int64(capa), int64(buf)
		if u("jflag")&0xff == 0 {
			thr := int32(qtsFact)
			switch u("dis") {
			case 0:
				thr = 2
			case 1:
				thr = 1
			case 2:
				thr = 0
			}
			if int32(int8(u("lvl37"))) <= thr {
				o["n280"] = 2
				o["c280_1"] = qtsFact
				w := u("lvl100") & 0xffff
				if u("r280b")&0xff != 0 {
					w = u("lvl38") & 0xffff
				}
				if w != 0 && u("batPtr") != 0 {
					if u("qmode")&0xff != 0 { // 0x5289b0(8, 0)
						msg := mem.alloc(0x24)
						for i := uint32(0); i < 9; i++ {
							*mem.cell(msg + 4*i) = 0
						}
						*mem.cell(msg) = 0xffffffff
						*mem.cell(msg + 0xc) = 8
						*mem.cell(msg + 0x20) = u("qStamp")
						qc, qa, qb := u("qCnt"), u("qCap"), uint32(qtsQBuf)
						mem.grow(&qc, &qa, &qb, 1, []uint32{msg})
						o["qCnt"], o["qCap"], o["qPtr"] = int64(qc), int64(qa), int64(qb)
					} else {
						o["nac0"] = 1
						for a, x := range []uint32{3, u("t1d28"), 2, u("b2d4") - u("b2cc"), 0} {
							o[fmt.Sprintf("cac0_%d", a+1)] = int64(x)
						}
						o["n886"], o["c886_1"], o["c886_2"] = 1, 0x61c5cc, 0
					}
				}
			}
		}
	}
	o["n670"], o["c670_1"], o["c670_2"] = 1, qtsFact, 1
	if u("dbg")&0xff != 0 {
		o["naad"], o["caad_1"], o["caad_2"], o["caad_3"] = 1, 5, 0x619a38, qtsLvl+qtsFact*0x1d8+4
	}
}
