package calc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// 0x553f10 natural_skill_growth(C, skill, repeat, quiet): for every repeat a percent roll
// raises the skill's success counter; eight successes raise the raw skill by one. A raise
// recomputes the usable-skill flags (0x553cd0 / 0x553d90), the effective skill value, runs
// the equipment check (0x51d960, address-stubbed in the case) and the full character
// rebuild (0x4ed9d0), then either flags the character for a later announcement or queues
// the type-0x14 notification event (0x528a80).
//
// Callees run natively in the case and are modelled here: 0x553d90 (skill usable),
// 0x553cd0 (flag refresh), 0x5b1af0 / 0x5b1b30 (character-creation bookkeeping,
// 0x557c90 / 0x557d20), 0x561440, 0x56aa20, 0x517ec0, 0x528a80, 0x547940 (potionFeature
// in n_548e60.go) and the 0x553a60 part of 0x4ed9d0. The message text built by 0x554170
// is not a rule output. 0x5539e0 (attribute clamp, aptitude rebuild) and the rest of
// 0x4ed9d0 are outside this output interface, as in n_50e700.go.
const (
	n553CharBase   = 0x20000000 // emu arena start: the case places the eight character records here
	n553CharStride = 6242       // 0x1862, exe 0x517ef9
	n553Skills     = 41         // exe 0x554175 loop bound 0x29
	n553Classes    = 15         // exe 0x4ed9d8 compares class with 0xf
	n553Spells     = 113        // exe 0x553cfc loop count 0x71
	n553Stub       = 64         // recorded 0x51d960 calls in the case
	n553HeapEvent  = 0x50000400 // first 0x400 bytes are the message buffer, then the 0x11-byte payload
)

type n553Exe struct {
	avail [n553Skills][n553Classes]uint32 // 0x615840
	main  [n553Classes]uint32             // 0x6161dc
	owned [n553Classes][4]int32           // 0x616218
	ctrl  [n553Classes]int32              // 0x616310
	kind  [n553Skills]uint32              // 0x6155b0, first dword of each 16-byte row
	jump  [28]byte                        // 0x553ec4
}

var n553Tables, n553TablesErr = n553LoadExe()

func n553LoadExe() (n553Exe, error) {
	var t n553Exe
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("skill growth: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	read := func(va uint32, n int) ([]byte, error) {
		off, err := peOffset(b, va)
		if err != nil {
			return nil, err
		}
		if n > len(b)-off {
			return nil, fmt.Errorf("skill growth: truncated table %#x", va)
		}
		return b[off : off+n], nil
	}
	u32at := func(va uint32) (uint32, error) {
		r, err := read(va, 4)
		if err != nil {
			return 0, err
		}
		return binary.LittleEndian.Uint32(r), nil
	}
	for s := 0; s < n553Skills; s++ {
		for c := 0; c < n553Classes; c++ {
			if t.avail[s][c], err = u32at(0x615840 + 4*uint32(15*s+c)); err != nil {
				return t, err
			}
		}
		if t.kind[s], err = u32at(0x6155b0 + 16*uint32(s)); err != nil {
			return t, err
		}
	}
	for c := 0; c < n553Classes; c++ {
		if t.main[c], err = u32at(0x6161dc + 4*uint32(c)); err != nil {
			return t, err
		}
		v, err := u32at(0x616310 + 4*uint32(c))
		if err != nil {
			return t, err
		}
		t.ctrl[c] = int32(v)
		for k := 0; k < 4; k++ {
			v, err := u32at(0x616218 + 16*uint32(c) + 4*uint32(k))
			if err != nil {
				return t, err
			}
			t.owned[c][k] = int32(v)
		}
	}
	j, err := read(0x553ec4, len(t.jump))
	if err != nil {
		return t, err
	}
	copy(t.jump[:], j)
	// Jump targets 0x553eb8[0..2]: type 1 is the "needs base points" check at 0x553e8d,
	// the others return true (0x553eb0).
	for i, want := range []uint32{0x553eb0, 0x553e8d, 0x553eb0} {
		got, err := u32at(0x553eb8 + 4*uint32(i))
		if err != nil {
			return t, err
		}
		if got != want {
			return t, fmt.Errorf("skill growth: unknown jump target %#x", got)
		}
	}
	for _, k := range t.jump {
		if k > 2 {
			return t, fmt.Errorf("skill growth: jump index %d out of range", k)
		}
	}
	return t, nil
}

// n553In / n553Out list the interface once; the order is the order of the case's in/out.
type n553In struct {
	slot, charAddr, s, n, quiet, class, race, idx75, state int64
	g                                                      [6]int64
	edd0, f506e, f506d, qcount, qcap, evSrc                int64
	raw, apt, cnt, f19d, f1af, f1b0, eff, bonus, ded       [n553Skills]int64
	b8d                                                    [n553Classes]int64
	att                                                    [7]int64
	spv, sch                                               [n553Spells]int64
	kclass, k8d                                            int64
	kraw, keff, k19e, uval, u1bc                           [n553Skills]int64
	u64                                                    int64
	grid                                                   [8][n553Skills]int64
	wf4, wf8                                               int64
}

type n553Out struct {
	raw, cnt, f19d, f1af, f1b0, eff [n553Skills]int64
	stubCount                       int64
	stubArg                         [n553Stub]int64
	grid                            [8][n553Skills]int64
	f506d, qcount, qcap             int64
	evP0, evP1, evP9, evKind, evSrc int64
	kraw, keff, uval, u1bc          [n553Skills]int64
	u64                             int64
	k19e                            [n553Skills]int64
	wf4, wf8                        int64
}

// Field kinds: width and signedness of the value in the original's memory.
const (
	n553U8 = iota
	n553I8
	n553U32
	n553I32
)

func (v *n553In) bind(f func(name string, p *int64, kind int)) {
	f("slot", &v.slot, n553U32)
	f("charAddr", &v.charAddr, n553U32)
	f("s", &v.s, n553I32)
	f("n", &v.n, n553U32)
	f("quiet", &v.quiet, n553U8)
	f("class", &v.class, n553U32)
	f("race", &v.race, n553I32)
	f("idx75", &v.idx75, n553I32)
	f("state", &v.state, n553U32)
	for j := range v.g {
		f("g"+strconv.Itoa(j), &v.g[j], n553U8)
	}
	f("edd0", &v.edd0, n553U32)
	f("f506e", &v.f506e, n553U8)
	f("f506d", &v.f506d, n553U8)
	f("qcount", &v.qcount, n553I32)
	f("qcap", &v.qcap, n553I32)
	f("evSrc", &v.evSrc, n553U32)
	arr := []struct {
		name string
		a    *[n553Skills]int64
		kind int
	}{{"raw", &v.raw, n553U32}, {"apt", &v.apt, n553U32}, {"cnt", &v.cnt, n553U32}, {"f19d", &v.f19d, n553U8},
		{"f1af", &v.f1af, n553U8}, {"f1b0", &v.f1b0, n553U8}, {"eff", &v.eff, n553U32}, {"bonus", &v.bonus, n553I8},
		{"ded", &v.ded, n553U32}}
	for _, x := range arr {
		for k := range x.a {
			f(x.name+strconv.Itoa(k), &x.a[k], x.kind)
		}
	}
	for c := range v.b8d {
		f("b8d"+strconv.Itoa(c), &v.b8d[c], n553U32)
	}
	for i := range v.att {
		f("att"+strconv.Itoa(i), &v.att[i], n553U32)
	}
	for i := range v.spv {
		f("spv"+strconv.Itoa(i), &v.spv[i], n553I32)
	}
	for i := range v.sch {
		f("spsch"+strconv.Itoa(i), &v.sch[i], n553U32)
	}
	f("kclass", &v.kclass, n553U32)
	f("k8d", &v.k8d, n553U32)
	ui := []struct {
		name string
		a    *[n553Skills]int64
		kind int
	}{{"kraw", &v.kraw, n553U32}, {"keff", &v.keff, n553U32}, {"k19e", &v.k19e, n553U8},
		{"uval", &v.uval, n553I32}, {"u1bc", &v.u1bc, n553U32}}
	for _, x := range ui {
		for k := range x.a {
			f(x.name+strconv.Itoa(k), &x.a[k], x.kind)
		}
	}
	f("u64", &v.u64, n553U32)
	for sl := range v.grid {
		for k := range v.grid[sl] {
			f("grid"+strconv.Itoa(sl)+"_"+strconv.Itoa(k), &v.grid[sl][k], n553U8)
		}
	}
	f("wf4", &v.wf4, n553U32)
	f("wf8", &v.wf8, n553U32)
}

func (o *n553Out) bind(f func(name string, p *int64)) {
	arr := func(name string, a *[n553Skills]int64) {
		for k := range a {
			f(name+strconv.Itoa(k), &a[k])
		}
	}
	arr("raw", &o.raw)
	arr("cnt", &o.cnt)
	arr("f19d", &o.f19d)
	arr("f1af", &o.f1af)
	arr("f1b0", &o.f1b0)
	arr("eff", &o.eff)
	f("stubCount", &o.stubCount)
	for k := range o.stubArg {
		f("stubArg"+strconv.Itoa(k), &o.stubArg[k])
	}
	for sl := range o.grid {
		for k := range o.grid[sl] {
			f("grid"+strconv.Itoa(sl)+"_"+strconv.Itoa(k), &o.grid[sl][k])
		}
	}
	f("f506d", &o.f506d)
	f("qcount", &o.qcount)
	f("qcap", &o.qcap)
	f("evP0", &o.evP0)
	f("evP1", &o.evP1)
	f("evP9", &o.evP9)
	f("evKind", &o.evKind)
	f("evSrc", &o.evSrc)
	arr("kraw", &o.kraw)
	arr("keff", &o.keff)
	arr("uval", &o.uval)
	arr("u1bc", &o.u1bc)
	f("u64", &o.u64)
	arr("k19e", &o.k19e)
	f("wf4", &o.wf4)
	f("wf8", &o.wf8)
}

func init() {
	var in n553In
	var out n553Out
	var inNames, outNames []string
	in.bind(func(name string, _ *int64, _ int) { inNames = append(inNames, name) })
	out.bind(func(name string, _ *int64) { outNames = append(outNames, name) })
	Nodes["00553f10"] = Node{Addr: "00553f10", In: inNames, Out: outNames, F: naturalSkillGrowthNode}
}

func n553Parse(in []int64) n553In {
	var v n553In
	i := 0
	v.bind(func(name string, p *int64, kind int) {
		if i >= len(in) {
			panic("skill growth: too few inputs")
		}
		x := in[i]
		i++
		lo, hi := int64(0), int64(1<<32-1)
		switch kind {
		case n553U8:
			hi = 255
		case n553I8:
			lo, hi = -128, 127
		case n553I32:
			lo, hi = -1<<31, 1<<31-1
		}
		if x < lo || x > hi {
			panic(fmt.Sprintf("skill growth: input %s=%d outside its memory width", name, x))
		}
		*p = x
	})
	if i != len(in) {
		panic("skill growth: wrong input count")
	}
	return v
}

type n553State struct {
	in        n553In
	t         *n553Exe
	stubCount int64
	ev        bool
}

func n553W(x int64) int64 { return int64(uint32(x)) } // wrap to a stored dword

// usable is 0x553d90(C, k, p): may this class use skill k. p holds the six school flags
// that 0x553cd0 builds, nil when the caller passes a null pointer (0x553f67).
func (st *n553State) usable(k int, p *[6]bool) bool {
	v, t := &st.in, st.t
	class := int(v.class)
	if t.avail[k][class] == 0 {
		return false
	}
	// exe 0x553db8: 0x547940(C, 0x1f), class row then race row.
	feat := potionFeature(v.class, v.race, 0x1f)
	if feat && k >= 24 && k <= 33 {
		return false
	}
	if k >= 28 && k <= 33 {
		// exe 0x553df1: one of skills 24..27 must have its usable flag set.
		found := false
		for j := 24; j <= 27; j++ {
			if v.f19d[j] != 0 {
				found = true
			}
		}
		if !found {
			return false
		}
		if v.ded[k] != 0 {
			return true
		}
		return p != nil && p[k-28]
	}
	if uint32(k) == t.main[class] {
		return true
	}
	for _, o := range t.owned[class] {
		if int64(o) == int64(k) {
			return true
		}
	}
	if k >= 34 {
		return uint32(v.att[k-34]) >= 100
	}
	if k > 27 {
		return true
	}
	if t.jump[k] == 1 {
		c := t.ctrl[class]
		if c >= 0 || c <= -255 {
			return true
		}
		return int32(uint32(v.b8d[class])+uint32(c)) > 0
	}
	return true
}

// uiAdd is 0x5b1af0(k) = 0x557c90 on the creation-screen copy (K = ui+0x18, U = ui+0x187c).
// The virtual call on the widget at ui+0x1b14 that follows is presentation; the case keeps
// that pointer null.
func (st *n553State) uiAdd(k int) {
	v, t := &st.in, st.t
	v.k19e[k] = 1
	v.uval[k] = 0
	if k >= 24 && k <= 27 {
		c := t.ctrl[v.kclass]
		if c < 0 && c > -255 && uint32(v.k8d)+uint32(c) == 1 {
			if d := int32(5 - uint32(v.kraw[k])); d > 0 {
				v.kraw[k] = 5
				v.keff[k] = n553W(v.keff[k] + int64(d))
				v.u1bc[k] = n553W(v.u1bc[k] + int64(d))
			}
		}
	}
}

// uiSub is 0x5b1b30(k) = 0x557d20 on the same copy.
func (st *n553State) uiSub(k int) {
	v := &st.in
	if d := v.uval[k]; d > 0 {
		v.u64 = n553W(v.u64 + d)
		v.kraw[k] = n553W(v.kraw[k] - d)
		v.uval[k] = 0
	}
	v.keff[k] = v.kraw[k]
}

// refresh is 0x553cd0(C): recompute every skill's usable flag.
func (st *n553State) refresh() {
	v := &st.in
	var arr [6]bool
	for i := range v.spv {
		if v.spv[i] == -1 || v.spv[i] == 2 {
			if v.sch[i] > 5 {
				panic("skill growth: spell school outside the 6 school flags")
			}
			arr[v.sch[i]] = true
		}
	}
	for k := 0; k < n553Skills; k++ {
		ok := st.usable(k, &arr)
		switch {
		case ok && v.f19d[k] == 0:
			v.f19d[k] = 1
			if v.state == 3 {
				st.uiAdd(k)
			}
		case !ok && v.f19d[k] != 0:
			v.f19d[k] = 0
			if v.state == 3 {
				st.uiSub(k)
			}
		}
	}
}

// effective is the shared formula of exe 0x554057..0x554099 and 0x553a60.
func (st *n553State) effective(k int) int64 {
	v := &st.in
	raw := uint32(v.raw[k])
	e := int32(raw)
	if uint32(k) == st.t.main[v.class] {
		b := raw * 25 / 100
		if b == 0 {
			b = 1
		}
		e = int32(raw + b)
	}
	e += int32(v.bonus[k])
	if e > 125 {
		e = 125
	} else if e < 0 {
		e = 0
	}
	return int64(e)
}

func naturalSkillGrowthNode(in []int64, r Rand) []int64 {
	if n553TablesErr != nil {
		panic(n553TablesErr)
	}
	v := n553Parse(in)
	st := &n553State{in: v, t: &n553Tables}
	v = st.in
	switch {
	case v.slot > 7 || v.charAddr != n553CharBase+n553CharStride*v.slot:
		panic("skill growth: character is not one of the eight party records")
	case v.s < 0 || v.s >= n553Skills:
		panic("skill growth: skill id outside 0..40")
	case v.class >= n553Classes:
		panic("skill growth: class outside 0..14")
	case v.kclass >= n553Classes:
		panic("skill growth: creation class outside 0..14")
	case v.qcount < 0 || v.qcount > 16:
		panic("skill growth: event queue longer than the fixture")
	}
	st.run(r)
	return st.output()
}

func (st *n553State) run(r Rand) {
	v, t := &st.in, st.t
	s := int(v.s)
	if v.n == 0 || t.avail[s][v.class] == 0 {
		return
	}
	v.f1b0[s] = 1
	if v.f19d[s] == 0 {
		if !st.usable(s, nil) {
			return
		}
		v.f19d[s] = 1
		if v.state == 3 {
			st.uiAdd(s)
		}
	}
	changed := false
	for i := int64(0); i < v.n; i++ {
		raw := uint32(v.raw[s])
		if raw >= 100 { // the original skips the rest of the loop without rolling
			break
		}
		chance := (100 - raw) * uint32(v.apt[s]) / 100
		if t.kind[s] == 4 {
			chance >>= 1
		}
		if chance == 0 {
			chance = 1
		}
		if uint32(r.R(100)) >= chance {
			continue
		}
		c := uint32(v.cnt[s]) + 1
		v.cnt[s] = int64(c)
		if c >= 8 {
			v.raw[s] = int64(raw + 1)
			v.cnt[s] = 0
			v.f1af[s] = 1
			changed = true
		}
	}
	if !changed {
		return
	}
	st.refresh()
	v.eff[s] = st.effective(s)
	st.stubCount++ // 0x51d960(C)
	// 0x4ed9d0(C): returns at once unless class < 15, race < 16 and [C+0x75] != -1.
	if v.idx75 != -1 {
		for k := 0; k < n553Skills; k++ {
			v.eff[k] = st.effective(k) // 0x553a60
		}
		st.stubCount += 48 // seven attribute rebuilds and 41 skills, one 0x51d960 each
		// 0x55ee30 (character present, state != 3): window dirty flags.
		switch v.state {
		case 6:
			v.wf8 = int64(uint32(v.wf8) | 0x100)
		case 7:
			v.wf4 = int64(uint32(v.wf4) | 1<<uint(v.slot))
		}
	}
	slot := int(v.slot) // 0x517ec0(C)
	flag := v.g[0] != 0 || v.edd0 != 0 || v.f506e != 0 || v.state != 7 || v.quiet != 0
	for _, g := range v.g {
		flag = flag || g != 0 // 0x561440
	}
	if flag {
		v.grid[slot][s] = 1
		v.f506d = 1
		return
	}
	// 0x528a80(0x14, text, payload): queue one event; the text is not a rule output.
	st.ev = true
	if v.qcount+1 > v.qcap {
		v.qcap = v.qcount + 1
	}
	v.qcount++
}

func (st *n553State) output() []int64 {
	v := &st.in
	o := n553Out{raw: v.raw, cnt: v.cnt, f19d: v.f19d, f1af: v.f1af, f1b0: v.f1b0, eff: v.eff,
		stubCount: st.stubCount, grid: v.grid, f506d: v.f506d, qcount: v.qcount, qcap: v.qcap,
		kraw: v.kraw, keff: v.keff, uval: v.uval, u1bc: v.u1bc, u64: v.u64, k19e: v.k19e,
		wf4: v.wf4, wf8: v.wf8}
	for k := int64(0); k < st.stubCount; k++ {
		o.stubArg[k] = v.charAddr
	}
	if st.ev {
		o.evP0, o.evP1, o.evP9, o.evKind, o.evSrc = 1, v.slot, v.s, 0x14, v.evSrc
	}
	var out []int64
	o.bind(func(_ string, p *int64) { out = append(out, *p) })
	return out
}
