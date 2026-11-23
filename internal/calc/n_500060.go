package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

// item_spell_learning_points 0x500060(P = character record, Q = item pointer):
// the item id is the dword at *Q, its spell id the byte at items[id]+0x63
// (items table pointer 0x68516c, stride 269; spell table pointer 0x65be1c,
// stride 447, 114 records). When character P may learn the spell (0x4ffcb0) the
// node marks it learned and recounts points (0x4ffe70 -> 0x4f96a0, 0x52a540),
// reports skill growth through 0x553f10 (0x14 with 2*level, 0x1c+realm with
// level, and the four school skills 0x18..0x1b when spell byte +0x5a is set),
// consumes the item (0x520070) and calls 0x52e690; otherwise it only shows the
// failure text (0x517a90, 0x55f260). Offsets are record offsets from the exe.
//
// Emu address-stubs the D4 callees 0x553f10, 0x52e690, 0x55f260, 0x55f1e0,
// 0x4edd20, 0x4ff7f0 and 0x520d10 (composition ruling 2026-10-09): the school
// id of 0x4ff7f0 and the three dwords 0x520d10 leaves in *Q come from inputs
// (x, w0..w2). F calls the real spellSchoolSkillSelectionNode and the real
// 0x520d10 Go function (a world of zeros, where refreshEquip returns at once and
// never writes *Q, so *Q stays (-1,0,0)).
//
// Deviations: the learn/failure text (wide-string formatting through the
// realm format table) is not modelled, only its single 0x55f260 call; the
// ground list holds 8 of its 500 entries (panic past that); item ids >= 819 and
// spell ids 0 (assert, cov-exempt block 500084) or >= 114 panic; the 0x547940
// feature test for 0x1f is looked up in the exe tables and panics if it ever
// matches; 0x4ffe70's ebx argument is not read.
const (
	isplItems     = 819
	isplSpells    = 114
	isplGround    = 8
	isplGroundVA  = 0x685191 // ground list base, entry = 3 dwords (0x520070)
	isplGroundEnd = 0x6868f5
	isplCursorVA  = 0x6874cb // cursor item (0x520070)
)

type isplExe struct {
	cap125, scale, half *big.Float // 0x5ec3f8 f32, 0x5ecbb4 f32, 0x5ebe80 f64
	classFeat           [15][3]uint32
	raceFeat            [11][5]uint32
}

var isplData, isplErr = isplLoad()

func isplLoad() (t isplExe, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return t, fmt.Errorf("item spell learning: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	at := func(va uint32, n int) ([]byte, error) {
		o, e := peOffset(b, va)
		if e != nil || o+n > len(b) {
			return nil, fmt.Errorf("item spell learning: bad table %#x", va)
		}
		return b[o : o+n], nil
	}
	f32 := func(va uint32) *big.Float {
		p, e := at(va, 4)
		if e != nil {
			err = e
			return nil
		}
		return new(big.Float).SetPrec(64).SetFloat64(float64(math.Float32frombits(binary.LittleEndian.Uint32(p))))
	}
	t.cap125, t.scale = f32(0x5ec3f8), f32(0x5ecbb4)
	if p, e := at(0x5ebe80, 8); e == nil {
		t.half = new(big.Float).SetPrec(64).SetFloat64(math.Float64frombits(binary.LittleEndian.Uint64(p)))
	} else {
		err = e
	}
	for c := range t.classFeat { // 0x61507c, stride 12
		p, e := at(0x61507c+12*uint32(c), 12)
		if e != nil {
			return t, e
		}
		for k := range t.classFeat[c] {
			t.classFeat[c][k] = binary.LittleEndian.Uint32(p[4*k:])
		}
	}
	for r := range t.raceFeat { // 0x615130, stride 20
		p, e := at(0x615130+20*uint32(r), 20)
		if e != nil {
			return t, e
		}
		for k := range t.raceFeat[r] {
			t.raceFeat[r][k] = binary.LittleEndian.Uint32(p[4*k:])
		}
	}
	return t, err
}

var isplInNames, isplOutNames, isplIdx = isplNames()

func isplNames() (ins, outs []string, idx map[string]int) {
	add := func(f string, a ...any) { ins = append(ins, fmt.Sprintf(f, a...)) }
	ins = append(ins, "q", "id", "spell", "lvl", "f48", "f5a", "f11f", "f120", "cls", "race", "pa", "pb")
	for k := 0; k < 15; k++ {
		add("cl%d", k)
	}
	for _, p := range []struct {
		n string
		c int
	}{{"sb", 4}, {"se", 4}, {"u", 6}, {"xx", 6}, {"old", 6}, {"oldD", 6}, {"fl", isplSpells}, {"rl", isplSpells}, {"rv", isplSpells}} {
		for k := 0; k < p.c; k++ {
			add("%s%d", p.n, k)
		}
	}
	ins = append(ins, "x", "w0", "w1", "w2", "ga", "gb", "gc", "gcnt")
	for j := 0; j < isplGround; j++ {
		for f := 0; f < 3; f++ {
			add("e%d_%d", j, f)
		}
	}
	ins = append(ins, "cin0", "cin4", "cin5")
	idx = map[string]int{}
	for i, n := range ins {
		idx[n] = i
	}
	o := func(f string, a ...any) { outs = append(outs, fmt.Sprintf(f, a...)) }
	for _, p := range []struct {
		n string
		c int
	}{{"ofl", isplSpells}, {"ocnt", 6}} {
		for k := 0; k < p.c; k++ {
			o("%s%d", p.n, k)
		}
	}
	outs = append(outs, "ope7d")
	for _, p := range []string{"opts", "odlt"} {
		for k := 0; k < 6; k++ {
			o("%s%d", p, k)
		}
	}
	for a := 2; a <= 4; a++ {
		for k := 0; k < 6; k++ {
			o("t%d_%d", a, k)
		}
	}
	outs = append(outs, "n553", "m2", "m3", "m4", "m5", "n52e", "n55f260", "n55f1e0", "n4edd20", "n520d10", "q520", "f520", "n4ff", "s4ff",
		"ocur0", "ocur4", "ocur5", "ocid", "oi0", "oi1", "oi2")
	for j := 0; j < isplGround; j++ {
		for f := 0; f < 3; f++ {
			o("oe%d_%d", j, f)
		}
	}
	outs = append(outs, "ogcnt")
	return
}

func init() {
	Nodes["00500060"] = Node{Addr: "00500060", In: isplInNames, Out: isplOutNames,
		F:   func(in []int64, r Rand) []int64 { return isplRun(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return isplRun(in, r, false) }}
}

func itemSpellLearningPointsNode(in []int64, r Rand) []int64 { return isplRun(in, r, true) }

func isplMaxU(a, b uint32) uint32 {
	if b > a {
		return b
	}
	return a
}

func isplRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(isplInNames) {
		panic("item_spell_learning_points: wrong input count")
	}
	if isplErr != nil {
		panic(isplErr)
	}
	g := func(n string) int64 { return in[isplIdx[n]] }
	gi := func(f string, k int) int64 { return in[isplIdx[fmt.Sprintf(f, k)]] }
	o := map[string]int64{}
	set := func(f string, k int, v int64) { o[fmt.Sprintf(f, k)] = v }

	// Q is the cursor item, a ground-list entry or an ordinary item record.
	q := g("q")
	ground := -1
	switch {
	case q == isplCursorVA:
	case q >= isplGroundVA && q <= isplGroundEnd:
		if (q-isplGroundVA)%12 != 0 || (q-isplGroundVA)/12 >= isplGround {
			panic("item_spell_learning_points: ground entry outside the 8-entry fixture")
		}
		ground = int((q - isplGroundVA) / 12)
	}
	id := g("id")
	if ground >= 0 {
		id = in[isplIdx[fmt.Sprintf("e%d_0", ground)]]
	}
	if id != g("id") || id < 0 || id >= isplItems {
		panic("item_spell_learning_points: item id outside the fixture")
	}
	spell := int(g("spell"))
	if spell == 0 {
		panic("item_spell_learning_points: spell id 0 (assert at 0x500084)")
	}
	if spell >= isplSpells {
		panic("item_spell_learning_points: spell id outside the 114-record table")
	}
	lvl, realm := uint32(g("lvl")), uint32(gi("rl%d", spell))
	if realm > 5 {
		panic("item_spell_learning_points: realm outside 0..5")
	}

	// defaults: memory the node leaves alone
	for i := 0; i < isplSpells; i++ {
		set("ofl%d", i, gi("fl%d", i))
	}
	for b := 0; b < 6; b++ {
		set("opts%d", b, gi("old%d", b))
		set("odlt%d", b, gi("oldD%d", b))
	}
	for j := 0; j < isplGround; j++ {
		for f := 0; f < 3; f++ {
			o[fmt.Sprintf("oe%d_%d", j, f)] = in[isplIdx[fmt.Sprintf("e%d_%d", j, f)]]
		}
	}
	o["ogcnt"] = g("gcnt")
	o["ocur0"], o["ocur4"], o["ocur5"], o["ocid"], o["oi0"] = g("cin0"), g("cin4"), g("cin5"), g("id"), g("id")
	o["n55f260"] = 1 // 0x55f260 runs once on both paths

	// 0x4ffcb0
	c := uint32(g("cls"))
	if c > 14 {
		panic("item_spell_learning_points: class outside 0..14")
	}
	race := uint32(g("race"))
	if race != 0xffffffff && race > 10 {
		panic("item_spell_learning_points: race outside -1, 0..10")
	}
	t := schoolSelectionData
	if schoolSelectionErr != nil {
		panic(schoolSelectionErr)
	}
	m := byte(0)
	if g("f120") != 0 {
		m |= 8
	}
	if g("f11f") != 0 {
		m |= 2
	}
	if g("f5a") != 0 {
		m |= 1
	}
	if g("f48") != 0 {
		m |= 4
	}
	learn := false
	if t.classMask[c]&m != 0 {
		for _, f := range isplData.classFeat[c] { // 0x547940(P, 0x1f)
			if f == 0x1f {
				panic("item_spell_learning_points: feature 0x1f branch of 0x547940 not modelled")
			}
		}
		if race != 0xffffffff {
			for _, f := range isplData.raceFeat[race] {
				if f == 0x1f {
					panic("item_spell_learning_points: feature 0x1f branch of 0x547940 not modelled")
				}
			}
		}
		lv := func(k int) int32 { // 0x4ff3b0
			if t.classAdjustment[k] == -255 {
				return -1
			}
			return int32(uint32(gi("cl%d", k)) + uint32(t.classAdjustment[k]))
		}
		sum := lv(int(c))
		if sum > 0 {
			for k := 0; k < 15; k++ {
				if gi("cl%d", k) != 0 && uint32(k) != c && t.classMask[k]&m != 0 {
					if v := lv(k); v > 0 {
						sum += v
					}
				}
			}
		}
		L := uint32(0)
		for e := 7; e >= 1; e-- {
			need := int32(1) // 0x4acb40: levels 2..7 from the exe, others 1
			if e >= 2 {
				need = t.required[e-2]
			}
			if sum >= need {
				L = uint32(e)
				break
			}
		}
		o["n4ff"], o["s4ff"] = 1, int64(spell)
		var sk int64
		if real {
			sk = isplSchool(in, spell, int(realm), r)
		} else {
			sk = g("x")
		}
		if sk < 24 || sk > 27 {
			panic("item_spell_learning_points: school outside 24..27")
		}
		v := uint32(gi("sb%d", int(sk-24))) + uint32(gi("u%d", int(realm)))/10
		pr := v/15 + 1
		if L < pr {
			pr = L
		}
		learn = pr >= lvl
	}

	if !learn {
		return isplOut(o)
	}

	// 0x4ffe70: mark learned, recount (0x4f96a0) and recompute points (0x52a540)
	flag := func(i int) int64 {
		if i == spell {
			return 1
		}
		return gi("fl%d", i)
	}
	set("ofl%d", spell, 1)
	var cnt [6]uint32
	var rmax [6]uint32
	for i := 0; i < isplSpells; i++ {
		if f := flag(i); f == 1 || f == 2 {
			rl := gi("rl%d", i)
			if rl < 0 || rl > 5 {
				panic("item_spell_learning_points: realm outside 0..5")
			}
			cnt[rl]++
			rmax[rl] = isplMaxU(rmax[rl], uint32(gi("rv%d", i)))
		}
	}
	for b := 0; b < 6; b++ {
		set("ocnt%d", b, int64(cnt[b]))
	}
	s := []uint32{uint32(g("sb0")), uint32(g("sb1")), uint32(g("sb2")), uint32(g("sb3"))}
	sort.Slice(s, func(i, j int) bool { return s[i] > s[j] })
	// 0x52a5f0..0x52a61c: x87 sum of the shifted dwords and signed s0, no 32-bit wrap
	tt := int64(s[3]>>3) + int64(s[2]>>2) + int64(s[1]>>1) + int64(int32(s[0]))
	tf := new(big.Float).SetPrec(64).SetInt64(tt)
	if tf.Cmp(isplData.cap125) > 0 {
		tf = new(big.Float).SetPrec(64).Copy(isplData.cap125)
	}
	pa, pb := uint32(g("pa")), int32(g("pb"))
	maxPts := uint32(0)
	for b := 0; b < 6; b++ {
		st := new(big.Float).SetPrec(64).SetUint64(uint64(uint32(3 * gi("xx%d", b))))
		st.Add(st, new(big.Float).SetPrec(64).SetInt64(int64(pb)))
		st.Add(st, tf)
		st.Mul(st, isplData.scale)
		st.Mul(st, new(big.Float).SetPrec(64).SetInt64(int64(int32(cnt[b]+pa+1))))
		st.Add(st, isplData.half)
		i64, _ := st.Int64()
		pts := uint32(i64)
		maxPts = isplMaxU(maxPts, pts)
		nw := uint32(0)
		if cnt[b] > 0 {
			nw = pts
		}
		nw = isplMaxU(nw, rmax[b])
		set("opts%d", b, int64(nw))
		if old := uint32(gi("old%d", b)); nw != old {
			set("odlt%d", b, int64(uint32(gi("oldD%d", b))+nw-old))
		}
	}
	o["ope7d"] = int64(maxPts)

	// 0x553f10 growth reports, in call order
	rec := 0
	grow := func(a2, a3 int64) {
		o[fmt.Sprintf("t2_%d", rec)], o[fmt.Sprintf("t3_%d", rec)], o[fmt.Sprintf("t4_%d", rec)] = a2, a3, 0
		rec++
	}
	grow(0x14, int64(2*lvl))
	grow(int64(0x1c+realm), int64(lvl))
	if g("f5a") != 0 {
		for k := int64(0x18); k <= 0x1b; k++ {
			grow(k, int64(lvl))
		}
	}
	o["n553"] = int64(rec)

	// 0x520070(Q, P, 1)
	switch {
	case q == isplCursorVA:
		o["ocur0"], o["ocur4"], o["ocur5"] = 0xffffffff, 0xff, 0xffff
		o["n55f1e0"] = 1
	default:
		w := [3]int64{g("w0"), g("w1"), g("w2")} // what 0x520d10 leaves in *Q
		if real {
			w = isplRefresh(q, r)
		}
		o["n520d10"], o["q520"], o["f520"] = 1, q, 1
		cntg := g("gcnt")
		if cntg > isplGround {
			panic("item_spell_learning_points: ground list beyond the 8-entry fixture")
		}
		if ground < 0 {
			o["oi0"], o["oi1"], o["oi2"] = w[0], w[1], w[2]
			break
		}
		for f := 0; f < 3; f++ {
			o[fmt.Sprintf("oe%d_%d", ground, f)] = w[f]
		}
		if int64(ground) >= cntg || w[0] != 0xffffffff {
			break
		}
		ent := func(j, f int) int64 { return o[fmt.Sprintf("oe%d_%d", j, f)] }
		for j := ground + 1; j < int(cntg); j++ {
			for f := 0; f < 3; f++ {
				o[fmt.Sprintf("oe%d_%d", j-1, f)] = ent(j, f)
			}
		}
		last := int(cntg) - 1
		o[fmt.Sprintf("oe%d_0", last)], o[fmt.Sprintf("oe%d_1", last)], o[fmt.Sprintf("oe%d_2", last)] = 0xffffffff, 0, 0
		o["ogcnt"] = cntg - 1
		o["n4edd20"] = 1
	}
	// 0x52e690(P, [0x68c510], 0, [0x5ed8c8], [0x5ed914])
	o["n52e"], o["m2"], o["m3"], o["m4"], o["m5"] = 1, g("ga"), 0, g("gb"), g("gc")
	return isplOut(o)
}

func isplOut(o map[string]int64) []int64 {
	out := make([]int64, len(isplOutNames))
	for i, n := range isplOutNames {
		out[i] = o[n]
	}
	return out
}

// isplSchool is the real 0x4ff7f0 for (P, spell, casting 0, training 0, power 0).
func isplSchool(in []int64, spell, realm int, r Rand) int64 {
	g := func(n string) int64 { return in[isplIdx[n]] }
	v := make([]int64, 50)
	v[5], v[6], v[7], v[8] = isplNZ(g("f48")), isplNZ(g("f5a")), isplNZ(g("f11f")), isplNZ(g("f120"))
	v[11], v[12], v[16] = int64(realm), g("cls"), 1
	for i := 0; i < 4; i++ {
		v[18+3*i], v[19+3*i] = g(fmt.Sprintf("se%d", i)), g(fmt.Sprintf("sb%d", i))
	}
	for k := 0; k < 15; k++ {
		v[35+k] = g(fmt.Sprintf("cl%d", k))
	}
	return spellSchoolSkillSelectionNode(v, r)[0]
}

func isplNZ(v int64) int64 {
	if v != 0 {
		return 1
	}
	return 0
}

// isplRefresh runs the real 0x520d10 on a zero world after 0x520070 has set *Q
// to (-1,0,0); the function never writes *Q, so those three dwords come back.
func isplRefresh(q int64, r Rand) [3]int64 {
	w := cgWorld(make([]int64, len(cgIn)))
	w.refreshEquip(q, 0x20000000, 1)
	return [3]int64{0xffffffff, 0, 0}
}
