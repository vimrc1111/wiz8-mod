package calc

import (
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// fightBranch is raised by fightRand when the replay prefix runs out.
type fightBranch struct {
	key string
	w   []float64 // probability of each choice index
}

// crtWeights is the CRT rand()%n mapping NodeDist uses: raw 0..32767,
// v = (raw*n)/32767 % n, each raw value weight 1/32768.
var crtCache = map[int][]float64{}

func crtWeights(n int) []float64 {
	crtMu.Lock()
	defer crtMu.Unlock()
	return crtWeightsLocked(n)
}

func crtWeightsLocked(n int) []float64 {
	if w, ok := crtCache[n]; ok {
		return w
	}
	w := make([]float64, max(1, n))
	for raw := uint64(0); raw <= 32767; raw++ {
		v := 0
		if n > 0 {
			v = int(uint32(raw*uint64(n))/32767) % n
		}
		w[v] += 1.0 / 32768
	}
	crtCache[n] = w
	return w
}

var diceCache = map[[2]int][]float64{}

// diceSumWeights is the distribution of the sum of m CRT-weighted R(s) calls.
func diceSumWeights(m, s int) []float64 {
	crtMu.Lock()
	defer crtMu.Unlock()
	k := [2]int{m, s}
	if w, ok := diceCache[k]; ok {
		return w
	}
	w := []float64{1}
	if s > 0 {
		one := crtWeightsLocked(s)
		for i := 0; i < m; i++ {
			nw := make([]float64, len(w)+s-1)
			for a, pa := range w {
				for b, pb := range one {
					nw[a+b] += pa * pb
				}
			}
			w = nw
		}
	}
	diceCache[k] = w
	return w
}

// fightRand replays a prefix of choice indexes. R(n) branches over CRT weights;
// DiceTotal (used by weapon damage's real path) branches once over the whole
// convolution instead of once per die, which is the same distribution.
type fightRand struct {
	prefix []int
	pos    int
}

func (r *fightRand) next(key string, w func() []float64) int {
	if r.pos == len(r.prefix) {
		panic(fightBranch{key, w()})
	}
	v := r.prefix[r.pos]
	r.pos++
	return v
}

func (r *fightRand) R(n int) int {
	if n < 0 || n > 65535 {
		panic("fight: R(n) outside 0..65535")
	}
	if n <= 1 {
		return 0
	}
	return r.next("R", func() []float64 { return crtWeights(n) })
}

// Below reports R(n) < c as one two-way branch (same distribution as comparing R(n)).
func (r *fightRand) Below(n, c int) bool {
	if n <= 1 {
		return c > 0
	}
	return r.next("B", func() []float64 {
		var lo float64
		for v, q := range crtWeights(n) {
			if v < c {
				lo += q
			}
		}
		return []float64{1 - lo, lo}
	}) == 1
}

// DiceTotal returns count rolls of the dice word Z/N/S summed (diceNode semantics).
func (r *fightRand) DiceTotal(count, z, n, s int64) int64 {
	base := count * (z + n)
	m := int(count * n)
	if m == 0 || s <= 1 {
		return base
	}
	return base + int64(r.next("D", func() []float64 { return diceSumWeights(m, int(s)) }))
}

// fightVisits counts enumerated nodes across all fightExpect calls (tests bound branch blowup).
var fightVisits atomic.Int64

var errFightLimit = errors.New("fight: enumeration limit exceeded")

var crtMu sync.Mutex

// fightExpect enumerates every RNG path of f and returns the probability-weighted
// sum of f's returned vector (complete distribution, no sampling). Subtrees below
// the first two branch points run in parallel; f must not share mutable state.
func fightExpect(f func(r Rand) []float64) ([]float64, error) {
	var (
		mu     sync.Mutex
		parts  = map[[2]int][]float64{}
		first  error
		visits atomic.Int64
		wg     sync.WaitGroup
	)
	sem := make(chan struct{}, runtime.NumCPU())
	fail := func(err error) {
		mu.Lock()
		if first == nil {
			first = err
		}
		mu.Unlock()
	}
	var walk func(prefix []int, p float64)
	walk = func(prefix []int, p float64) {
		fightVisits.Add(1)
		if visits.Add(1) > 20000000 || len(prefix) > 512 {
			fail(errFightLimit)
			return
		}
		mu.Lock()
		stop := first != nil
		mu.Unlock()
		if stop {
			return
		}
		var br *fightBranch
		var out []float64
		var err error
		func() {
			defer func() {
				if x := recover(); x != nil {
					if b, ok := x.(fightBranch); ok {
						br = &b
					} else {
						err = fmt.Errorf("%v", x)
					}
				}
			}()
			out = f(&fightRand{prefix: prefix})
		}()
		if err != nil {
			fail(err)
			return
		}
		if br == nil {
			// Bucket by the parallel prefix: each bucket is filled by one goroutine
			// in a fixed order, so the float sums are reproducible run to run.
			k := [2]int{-1, -1}
			copy(k[:], prefix)
			mu.Lock()
			part := parts[k]
			if part == nil {
				part = make([]float64, len(out))
				parts[k] = part
			}
			for i, v := range out {
				part[i] += p * v
			}
			mu.Unlock()
			return
		}
		for v, q := range br.w {
			if q <= 0 {
				continue
			}
			child := append(prefix[:len(prefix):len(prefix)], v)
			if len(prefix) < 2 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					sem <- struct{}{}
					walk(child, p*q)
					<-sem
				}()
			} else {
				walk(child, p*q)
			}
		}
	}
	walk(nil, 1)
	wg.Wait()
	keys := slices.Collect(maps.Keys(parts))
	slices.SortFunc(keys, func(a, b [2]int) int { return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1])) })
	var sum []float64
	for _, k := range keys {
		if sum == nil {
			sum = make([]float64, len(parts[k]))
		}
		for i, v := range parts[k] {
			sum[i] += v
		}
	}
	return sum, first
}

// fightQ4Effects is the Q4 boundary row of 0x543270 (re/emu/cases/00543270.json row 81),
// one value per Nodes["00543270"].In name in order; calc fight overrides the cells it models.
const fightQ4Effects = `0 1 536871168 1 0 2513873324 3 0 3958627719 0 156 184 3 68 81 1 0 2 150 34 9276 1 -21241 214 159 0 0
0 0 0 19 0 0 0 0 0 0 0 0 0 0 43 1 50 -4 11 0 2 7 2402464987 842 472 632 931 765 518 124 465 145 839
822 864 27 661 669 833 756 63 337 606 589 652 79 805 3391413659 71 26 0 1 0 1 0 1 1 0 0 1 1 0 1 20 1
9 16 6 0 5 20 16 20 11 0 7 9 1 5 7 14 11 8 9 12 0 1 10 13 12 10 5 10 3 14 11 5 13 2 4 7 6 0 0 0 0 8
13 46 1230137050 4086375950 893252631 2775586159 3402614605 927822497 0 0 -1 -1 0 20 0 0 1 1 10 0 0
0 0 0 0 0 0 100 100 100 100 100 100 22 0 0 0 0 0 0 0 20 1 5 65 66 0 1 7 65 66 0 0 50 10 0 0 0 0 0 0
0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 9 100 5 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1 1 1 1 1 396 20 19 0 48 0
0 48 0 0 48 0 0 48 0 0 48 0 0 48 0 0 48 0 0 48 0 0 48 0 0 52 0 0 0 52 0 0 0 52 0 0 0 52 0 0 0 52 0 0
0 52 0 0 0 0 10 5 536870912 536940544 0 0 0 0 0 537133056 0 0 0 537137152 0 0 0 537138176 0 1 1 1 1
1 1 1 1 1 -1 -1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
0 0 0 0 0 0 0 0 536956928 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 255 255 255 255 255 255 255 255 255
255 255 255 255 255 255 0 0 0 0 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0
255 0 255 0 255 0 255 0 255 0 255 255 255 255 255 255 255 255 255 255 255 255 255 255 255 0 0 0 0 0
255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255
255 255 255 255 255 255 255 255 255 255 255 255 255 255 0 0 0 0 0 255 0 255 0 255 0 255 0 255 0 255
0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1
-1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1
-1 -1 -1 -1 -1 4660 100 10 1000 0 -1 3 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 10 0 1 7 8 9 1 2
3 536870912 536940544 0 0 0 0 0 537133056 0 0 0 537137152 0 0 0 537138176 0 1 1 1 1 1 1 1 1 1 -1 -1
0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
536956928 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 255 255 255 255 255 255 255 255 255 255 255 255 255
255 255 0 0 0 0 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255
0 255 0 255 0 255 255 255 255 255 255 255 255 255 255 255 255 255 255 255 0 0 0 0 0 255 0 255 0 255
0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 255 255 255 255
255 255 255 255 255 255 255 255 255 255 0 0 0 0 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0
255 0 255 0 255 0 255 0 255 0 255 0 255 0 255 0 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1
-1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1
-1 100 100 0 0 0 0`

// ---- calc fight: one character's melee round against one monster ----

// FightNodes are the node addresses calc fight runs through F, in call order.
var FightNodes = []string{"004ee220", "0053d870", "00541c00", "00541ec0", "005420b0", "00543270"}

type FightWeapon struct{ Item, Skill int }

// FightScenario is the build scenario plus the monster (MONSTERS.DBS id) and difficulty.
type FightScenario struct {
	Race, Class, Level  int
	Prio                []int
	Attrs               *[7]int
	Haste               bool
	Skills              map[int]int
	Main, Off           *FightWeapon
	Mode, OffMode       int
	Monster, Difficulty int
}

// FightWorld is the player's own exe and databases, read only.
type FightWorld struct {
	x           *ExeTables
	items, mons []byte
	nItem, nMon int
	fumA, fumB  float64 // f64 @0x5ee7a8, @0x5ebe80 (53e250 fumble chance cubic)
}

func LoadFightWorld(exe, dbs string) (*FightWorld, error) {
	x, err := LoadExe(exe)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(exe)
	if err != nil {
		return nil, err
	}
	f64 := func(va uint32) (float64, error) {
		o, err := peOffset(b, va)
		if err != nil || o+8 > len(b) {
			return 0, fmt.Errorf("calc fight: exe constant %#x unreadable", va)
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(b[o:])), nil
	}
	w := &FightWorld{x: x}
	if w.fumA, err = f64(0x5ee7a8); err != nil {
		return nil, err
	}
	if w.fumB, err = f64(0x5ebe80); err != nil {
		return nil, err
	}
	if w.items, w.nItem, err = records(filepath.Join(dbs, "ITEMS.DBS"), 4, 0x10d); err != nil {
		return nil, err
	}
	if w.mons, w.nMon, err = records(filepath.Join(dbs, "MONSTERS.DBS"), 4, 0x297); err != nil {
		return nil, err
	}
	return w, nil
}

// ItemID finds an ITEMS.DBS id by exact name (-1 when absent).
func (w *FightWorld) ItemID(name string) int {
	for i := 0; i < w.nItem; i++ {
		if nameUTF16(w.items[i*0x10d:i*0x10d+0x3c]) == name {
			return i
		}
	}
	return -1
}

func field(rec []byte, off, size int, signed bool) int64 {
	var u uint64
	for i := size - 1; i >= 0; i-- {
		u = u<<8 | uint64(rec[off+i])
	}
	if signed {
		shift := 64 - 8*size
		return int64(u<<shift) >> shift
	}
	return int64(u)
}

// ---- input registry: every node input is set by name with its provenance ----

type fightRow struct{ src, val, why string }

type fightReg struct {
	mu   sync.Mutex
	rows map[string]map[string]fightRow
	err  error
}

func newFightReg() *fightReg { return &fightReg{rows: map[string]map[string]fightRow{}} }

type fightBind struct {
	reg  *fightReg
	node string
	idx  map[string]int
	in   []int64
	set  []bool
}

func (g *fightReg) bind(node string) *fightBind {
	n := Nodes[node]
	b := &fightBind{reg: g, node: node, idx: map[string]int{}, in: make([]int64, len(n.In)), set: make([]bool, len(n.In))}
	for i, s := range n.In {
		b.idx[s] = i
	}
	return b
}

func (b *fightBind) record(name string, row fightRow, v int64) {
	g := b.reg
	g.mu.Lock()
	defer g.mu.Unlock()
	i, ok := b.idx[name]
	switch {
	case !ok:
		g.fail(fmt.Errorf("calc fight: %s has no input %q", b.node, name))
		return
	case b.set[i]:
		g.fail(fmt.Errorf("calc fight: %s input %s set twice", b.node, name))
		return
	}
	b.in[i], b.set[i] = v, true
	m := g.rows[b.node]
	if m == nil {
		m = map[string]fightRow{}
		g.rows[b.node] = m
	}
	if old, ok := m[name]; ok && old != row {
		g.fail(fmt.Errorf("calc fight: %s input %s provenance differs between calls (%v vs %v)", b.node, name, old, row))
	}
	m[name] = row
}

func (g *fightReg) fail(err error) {
	if g.err == nil {
		g.err = err
	}
}

// put sets a modelled input; loc says where its value comes from.
func (b *fightBind) put(name, src, loc string, v int64) { b.record(name, fightRow{src, loc, ""}, v) }

// fix sets an input the model holds constant; why is required.
func (b *fightBind) fix(name string, v int64, why string) {
	b.record(name, fightRow{"fixed", strconv.FormatInt(v, 10), why}, v)
}

func (b *fightBind) fixAll(why string, v int64, names ...string) {
	for _, n := range names {
		b.fix(n, v, why)
	}
}

func seqNames(f string, lo, hi int) []string {
	var s []string
	for i := lo; i <= hi; i++ {
		s = append(s, fmt.Sprintf(f, i))
	}
	return s
}

func (b *fightBind) done() ([]int64, error) {
	b.reg.mu.Lock()
	defer b.reg.mu.Unlock()
	if b.reg.err != nil {
		return nil, b.reg.err
	}
	for i, ok := range b.set {
		if !ok {
			return nil, fmt.Errorf("calc fight: %s input %s unset", b.node, Nodes[b.node].In[i])
		}
	}
	return b.in, nil
}

func single(node string, in []int64) ([]int64, error) { return singleNode(Nodes[node], in) }

func singleNode(n Node, in []int64) ([]int64, error) {
	node := n.Addr
	d, err := NodeDist(n, in)
	if err != nil {
		return nil, err
	}
	if len(d) != 1 {
		return nil, fmt.Errorf("calc fight: %s gave %d outcomes, the model needs one", node, len(d))
	}
	return d[0].Out, nil
}

// ---- the fight ----

type FightKV struct {
	Key string
	Val float64
}

const (
	whyNoRuntime = "戰鬥中執行期狀態未建模（防禦／疲勞／姿勢等取開戰初值 0）"
	whyTargetChr = "目標是怪物，角色目標欄不讀"
	whyEmuOnly   = "Emu 重播格，F 路徑不讀"
	whyGeometry  = "幾何不在數值範圍（MG 2026-10-11）：固定正面相鄰 1 格"
	whyIndex     = "世界表索引：calc fight 直接給該筆的欄值"
)

type fightChar struct {
	s     FightScenario
	attr  [7]int
	skill [41]int
	main  []byte
	off   []byte // nil when no offhand
	tmpl  []byte
}

func (w *FightWorld) fightChar(s FightScenario) (*fightChar, error) {
	if s.Main == nil && s.Off != nil {
		return nil, errors.New("calc fight: unarmed Main cannot have Off")
	}
	// Mode 0 is the default weapon attack; unarmed hands use the punch mode.
	if s.Main == nil && s.Mode == 0 {
		s.Mode = 6
	}
	if s.Main == nil && s.OffMode == 0 {
		s.OffMode = 6
	}
	if s.Monster < 0 || s.Monster >= w.nMon {
		return nil, errors.New("calc fight: Monster outside MONSTERS.DBS")
	}
	if s.Difficulty < 0 || s.Difficulty > 3 {
		return nil, errors.New("calc fight: Difficulty must be 0..3")
	}
	var attr [7]int
	var err error
	if s.Attrs == nil {
		attr, err = PlanAttrs(w.x, s.Race, s.Class, s.Level, s.Prio)
		if err != nil {
			return nil, err
		}
	} else {
		if w.x == nil || s.Race < 0 || s.Race >= len(w.x.Race) || s.Class < 0 || s.Class >= len(w.x.ClassMin) || s.Level < 1 {
			return nil, errors.New("calc fight: invalid tables, race, class or level")
		}
		attr = *s.Attrs
		for _, v := range attr {
			if v < 0 || v > 125 {
				return nil, errors.New("calc fight: Attrs values must be 0..125")
			}
		}
	}
	c := &fightChar{s: s, tmpl: w.mons[s.Monster*0x297 : (s.Monster+1)*0x297]}
	for k, raw := range s.Skills {
		if k < 0 || k > 40 || raw < 0 || raw > 125 {
			return nil, errors.New("calc fight: Skills id 0..40, value 0..125")
		}
	}
	item := func(id int) ([]byte, error) {
		if id < 0 || id >= w.nItem {
			return nil, errors.New("calc fight: Item outside ITEMS.DBS")
		}
		return w.items[id*0x10d : (id+1)*0x10d], nil
	}
	if s.Main != nil {
		if c.main, err = item(s.Main.Item); err != nil {
			return nil, err
		}
	}
	if s.Off != nil {
		if c.off, err = item(s.Off.Item); err != nil {
			return nil, err
		}
	}
	// 0x50e980: each worn item adds i8 +0xb2 to skill +0xb1 and i8 +0xb4 to attribute +0xb3 (0xff = none);
	// 0x553a60 / 0x5539e0 add the sums after the favoured-skill bonus and clamp to 0..125 / 1..125.
	var sb [41]int
	var ab [7]int
	for _, d := range [][]byte{c.main, c.off} {
		if d == nil {
			continue
		}
		if k := d[0xb1]; k != 0xff {
			sb[k] += int(int8(d[0xb2]))
		}
		if a := d[0xb3]; a != 0xff {
			ab[a] += int(int8(d[0xb4]))
		}
	}
	for a := range attr {
		attr[a] = min(125, max(1, attr[a]+ab[a]))
	}
	c.attr = fightHasteAttrs(attr, s.Haste)
	for k := range c.skill {
		raw, ok := s.Skills[k]
		if !ok && sb[k] == 0 {
			continue
		}
		if k == w.x.MainSkill[s.Class] {
			raw += max(1, raw/4)
		}
		c.skill[k] = min(125, max(0, raw+sb[k]))
	}
	return c, nil
}

func (c *fightChar) mode(h int) int64 {
	if h == 0 {
		return int64(c.s.Mode)
	}
	return int64(c.s.OffMode)
}

// itemCell reads a hand's item field; an empty hand reads 0.
func (c *fightChar) defID(h int) int64 {
	if h == 0 && c.s.Main != nil {
		return int64(c.s.Main.Item)
	}
	if h == 1 && c.s.Off != nil {
		return int64(c.s.Off.Item)
	}
	return 0
}

func (c *fightChar) itemCell(b *fightBind, name string, h, off, size int, signed bool) {
	rec, who := c.main, "Main"
	if h == 1 {
		rec, who = c.off, "Off"
	}
	v := int64(0)
	if rec != nil {
		v = field(rec, off, size, signed)
	}
	if h == 0 && rec == nil {
		b.put(name, "scenario", "Main 空手（空物品格＝0）", v)
	} else {
		b.put(name, "dbs", fmt.Sprintf("ITEMS[%s]+%#x（無則 0）", who, off), v)
	}
}

func (c *fightChar) classLevels(b *fightBind, f string) {
	for k := 0; k < 15; k++ {
		v := int64(0)
		if k == c.s.Class {
			v = int64(c.s.Level)
		}
		b.put(fmt.Sprintf(f, k), "scenario", "Level（k==Class，其餘 0）", v)
	}
}

func (c *fightChar) tmplStat(b *fightBind, name string, k int) {
	b.put(name, "dbs", fmt.Sprintf("clamp(MONSTERS[Monster]+%#x,1,125)", 0xd2+k), min(125, max(1, int64(c.tmpl[0xd2+k]))))
}

func (c *fightChar) tmplByte(b *fightBind, name string, off int, signed bool) {
	b.put(name, "dbs", fmt.Sprintf("MONSTERS[Monster]+%#x", off), field(c.tmpl, off, 1, signed))
}

var attrNames = [7]string{"STR", "INT", "PIE", "VIT", "DEX", "SPD", "SEN"}

func fightHasteAttrs(attr [7]int, haste bool) [7]int {
	if haste {
		attr[5] = max(attr[5], 125)
	}
	return attr
}

func (c *fightChar) attrSource(a int) string {
	source := "PlanAttrs"
	if c.s.Attrs != nil {
		source = "Attrs"
	}
	if a == 5 && c.s.Haste {
		source += "＋Haste（max(SPD,125)）"
	}
	return source
}

func (c *fightChar) attrIn(b *fightBind, name string, a int) {
	b.put(name, "scenario", attrNames[a]+"（"+c.attrSource(a)+"）", int64(c.attr[a]))
}

// hands runs 0x4ee220 to its fixed point (it reads its own previous outputs).
func (c *fightChar) hands(g *fightReg) ([]int64, error) {
	prior := make([]int64, 37)
	for iter := 0; iter < 8; iter++ {
		b := g.bind("004ee220")
		b.fix("inParty", 1, "角色在隊伍中")
		b.put("class", "scenario", "Class", int64(c.s.Class))
		b.put("race", "scenario", "Race", int64(c.s.Race))
		c.attrIn(b, "strength", 0)
		c.attrIn(b, "dexterity", 4)
		c.attrIn(b, "speed", 5)
		c.attrIn(b, "senses", 6)
		b.fix("loadCategory", 0, "負重未建模（取無負重）")
		re := int64(0)
		if c.attr[6] >= 100 {
			re = 1
		}
		b.put("rangedExpert", "scenario", "SEN≥100（"+c.attrSource(6)+"）", re)
		me := int64(0)
		if c.attr[0] >= 100 {
			me = 1
		}
		b.put("meleeExpert", "scenario", "STR≥100（"+c.attrSource(0)+"）", me)
		c.classLevels(b, "classLevel%d")
		for k := 0; k <= 40; k++ {
			b.put(fmt.Sprintf("skill%d", k), "scenario", "EffectiveSkill(Skills[k])（未列 0）", int64(c.skill[k]))
		}
		if c.main == nil {
			b.put("itemID0", "scenario", "Main 空手（-1）", -1)
		} else {
			b.fix("itemID0", 0, whyIndex+"（主手＝0；0x51c930 另取 Main.Item 定義 ID）")
		}
		b.fix("charges0", 1, "武器不耗次數")
		off := int64(-1)
		if c.off != nil {
			off = 1
		}
		b.put("itemID1", "scenario", "Off（有 1，無 -1）", off)
		b.fix("charges1", 1, "武器不耗次數")
		for h := 0; h < 2; h++ {
			p := fmt.Sprintf("item%d", h)
			c.itemCell(b, p+"type", h, 62, 1, false)
			c.itemCell(b, p+"subtype", h, 63, 2, false)
			c.itemCell(b, p+"weaponSkill", h, 70, 1, true)
			c.itemCell(b, p+"range", h, 71, 1, false)
			c.itemCell(b, p+"initiative", h, 72, 1, true)
			c.itemCell(b, p+"toHit", h, 73, 1, true)
			c.itemCell(b, p+"damage", h, 74, 2, true)
			c.itemCell(b, p+"attacks", h, 78, 2, false)
			c.itemCell(b, p+"chargeType", h, 102, 1, false)
		}
		b.fixAll("非 NPC、無任務旗標", 0, "npcUnavailable", "npcPersonality", "quest68")
		for h := 0; h < 2; h++ {
			for j, f := range attacksHandFields[4:] {
				b.put(fmt.Sprintf("prior%d%s", h, f), "node", fmt.Sprintf("004ee220.hand%d%s（前次輸出，不動點）", h, f), prior[18*h+4+j])
			}
		}
		in, err := b.done()
		if err != nil {
			return nil, err
		}
		defs := [2]int64{c.defID(0), c.defID(1)}
		n := Nodes["004ee220"]
		n.F = func(in []int64, r Rand) []int64 { return calcAttacks(in, r, &defs) }
		out, err := singleNode(n, in)
		if err != nil {
			return nil, err
		}
		if fmt.Sprint(out) == fmt.Sprint(prior) {
			return out, nil
		}
		prior = out
	}
	return nil, errors.New("calc fight: 004ee220 has no fixed point within 8 rounds")
}

const (
	hType = iota
	hEnabled
	hWeaponSkill
	hCombatSkill
	hCombinedSkill
	hBAR
	hAttacks
	hSwings
	hInitiative
	hToHit
	hDamageFlat
	hDamagePercent
	hDamageDice
	hAttackModes
	hEffect0
)

func hv(hs []int64, h, f int) int64 { return hs[18*h+f] }

func handSrc(b *fightBind, name string, hs []int64, h, f int) {
	b.put(name, "node", fmt.Sprintf("004ee220.hand%d%s", h, attacksHandFields[f]), hv(hs, h, f))
}

func handSrc2(b *fightBind, f2 string, hs []int64, f int) {
	for h := 0; h < 2; h++ {
		handSrc(b, fmt.Sprintf(f2, h), hs, h, f)
	}
}

func f32bits(v float32) int64 { return int64(math.Float32bits(v)) }

// swings runs 0x53d870 for one attack of hand h.
// ls is the T+0x82 flag the round preparation 0x4e8370 rolled (F02/F03).
func (c *fightChar) swings(g *fightReg, hs []int64, h int, ls int64) ([]NodeOutcome, error) {
	b := g.bind("0053d870")
	b.fix("who", 0, "攻擊者佔隊伍格 0")
	b.put("mode", "scenario", "Mode／OffMode（依手）", c.mode(h))
	b.fix("flag510c", 0, "非紀錄重播")
	b.put("hand", "scenario", "手（0 主、1 副）", int64(h))
	handSrc2(b, "swings%d", hs, hAttacks)
	b.put("lsFlag", "scenario", "LS 旗標：0x4e8370 以 lsChance 抽中（Fight 兩值加權）", ls)
	handSrc2(b, "kind%d", hs, hType)
	handSrc2(b, "enabled%d", hs, hEnabled)
	handSrc2(b, "hmode%d", hs, hWeaponSkill)
	handSrc2(b, "dice%d", hs, hSwings)
	handSrc2(b, "umask%d", hs, hAttackModes)
	if c.main == nil {
		b.put("item6", "scenario", "Main 空手（2＝空物品格；子節點域不收 -1）", 2)
	} else {
		b.fix("item6", 0, whyIndex+"（主手＝0）")
	}
	off := int64(2) // child 0x519ac0's domain is 0..3: no offhand points at the empty record 2
	if c.off != nil {
		off = 1
	}
	b.put("item7", "scenario", "Off（有 1，無 2＝空物品格；子節點 519ac0 域不收 -1）", off)
	b.fix("a15", 0, "未建模：C+0xa15 禁 LS 狀態取 0")
	b.put("cls", "scenario", "Class", int64(c.s.Class))
	b.put("race", "scenario", "Race", int64(c.s.Race))
	b.put("level", "scenario", "當前職業等級 [C+0x8d+4·C+0x69]（build/wiz8.asm:432059）", int64(c.s.Level))
	b.fix("inParty", 1, "角色在隊伍中")
	b.fix("mainAction", 0, "行動＝近戰攻擊")
	b.fix("ttype", 3, "目標類別＝怪物")
	b.fix("tchar", 4, whyTargetChr+"（任一非攻擊者格）")
	b.fix("tmon", 0, whyIndex+"（怪物清單 0）")
	b.fix("tgrp", -1, "單一目標，無群組")
	for k := 0; k < 4; k++ {
		p := fmt.Sprintf("it%d", k)
		if k < 2 {
			c.itemCell(b, p+"cls", k, 62, 1, false)
			c.itemCell(b, p+"sub", k, 63, 2, false)
			c.itemCell(b, p+"range", k, 71, 1, false)
			c.itemCell(b, p+"bonus", k, 173, 4, false)
			c.itemCell(b, p+"wmask", k, 78, 2, false)
		} else {
			b.fixAll("裝備表只用主副手兩格", 0, p+"cls", p+"sub", p+"range", p+"bonus", p+"wmask")
		}
	}
	b.fix("tpresent", 1, whyTargetChr)
	b.fix("thp", 10, whyTargetChr)
	b.fix("tstatus", 0, whyTargetChr)
	b.fix("tmainA", 0, whyTargetChr)
	b.fix("toffA", 0, whyTargetChr)
	b.fix("tcur", 20, whyTargetChr)
	b.fix("tinterf", 0, whyTargetChr)
	b.fix("tatt", 0, whyTargetChr)
	b.fix("mid", 0, whyIndex+"（怪物清單 0）")
	b.fix("mhp", 1, "怪物存活（回合內死亡未建模）")
	b.fix("m33", 1, whyNoRuntime)
	b.fix("m107", 0, whyNoRuntime)
	b.fix("m87", 0, whyNoRuntime)
	b.fix("m2e1", 0, whyNoRuntime+"（姿勢正常）")
	b.fix("m24b", 100, whyNoRuntime)
	b.fix("mexcl", 0, whyNoRuntime)
	b.fix("mactive", 1, "怪物在場")
	b.fixAll("怪物可被攻擊", 1, "men", "mphys", "mspell")
	b.fix("mcnt", 0, whyNoRuntime)
	b.fix("combat", 1, "戰鬥中")
	b.fix("px", f32bits(10), whyGeometry)
	b.fix("py", 0, whyGeometry)
	b.fix("pz", f32bits(10), whyGeometry)
	b.fix("mx", f32bits(11), whyGeometry)
	b.fix("my", 0, whyGeometry)
	b.fix("mz", f32bits(10), whyGeometry)
	b.fix("mrad", f32bits(1), whyGeometry)
	b.fix("prad", 0, whyGeometry)
	b.fix("mhead", 0, whyGeometry)
	b.fixAll(whyGeometry+"（隊形格）", 0, seqNames("f0_%d", 0, 7)...)
	b.fixAll(whyGeometry+"（隊形格）", 0, seqNames("f3_%d", 0, 7)...)
	in, err := b.done()
	if err != nil {
		return nil, err
	}
	return NodeDist(Nodes["0053d870"], in)
}

// defenseCells are the target-side runtime cells 0x541c00 and 0x541ec0 share.
var defenseCells = []string{"speedDefense", "shieldDefense", "characterDefenseSuppressed", "monsterDefense", "monsterHitDefense",
	"fatigue", "monsterPosture", "monsterDefenseSuppressed", "completedActionSelector", "completedPosture", "pendingPosture",
	"sceneFlags", "HP", "status", "actionCompleted", "mainAction", "offAction", "characterDefHit"}

func (c *fightChar) arCommon(b *fightBind, h int, f4, f5 int64) {
	c.arCommonMode(b, h, f4, f5, "scenario", "Mode／OffMode（依手）", c.mode(h))
}

func (c *fightChar) arCommonMode(b *fightBind, h int, f4, f5 int64, src, why string, mode int64) {
	b.fix("targetType", 3, "目標類別＝怪物")
	b.put("mode", src, why, mode)
	b.fix("characterIndex", 4, whyTargetChr+"（任一非攻擊者格）")
	b.fix("combat", 1, "戰鬥中")
	b.put("ambush", "node", "0053d870.flag9a4", f4)
	b.put("surprise", "node", "0053d870.flag9a5", f5)
	b.fixAll(whyNoRuntime, 0, defenseCells...)
	c.tmplByte(b, "ambushDefenseMask", 0xd0, false)
	b.fix("monsterID", 0, whyIndex+"（怪物清單 0）")
	b.fix("templateIndex", 0, whyIndex+"（模板欄直接給）")
	c.tmplByte(b, "monsterAC", 0x15d, true)
	b.fix("who", 0, "攻擊者佔隊伍格 0")
	b.put("hand", "scenario", "手（0 主、1 副）", int64(h))
	b.fix("reverseDifficultyState", 0, "未建模：反向難度狀態")
	b.put("difficulty", "scenario", "Difficulty", int64(c.s.Difficulty))
}

// arhit is 0x541c00 (skip: 53e250 calls it with 0 for the chance and 1 for the fumble term).
func (c *fightChar) arhit(g *fightReg, hs []int64, h int, f4, f5, skip int64) (int64, error) {
	b := g.bind("00541c00")
	c.arCommon(b, h, f4, f5)
	b.put("skipDefense", "node", "0053e250 膠合（命中 0／失手項 1）", skip)
	handSrc2(b, "handB%d", hs, hBAR)
	handSrc2(b, "handJ%d", hs, hToHit)
	handSrc2(b, "handKind%d", hs, hWeaponSkill)
	b.fix("skillBonus", 0, "未建模：暫時技能加成")
	b.fix("fatigueCategory", 0, "未建模：疲勞（取精神飽滿）")
	b.fix("battleRecordT94", 0, "未建模：戰鬥紀錄 T94")
	c.attrIn(b, "speed", 5)
	c.attrIn(b, "attackerStat0", 1)
	c.attrIn(b, "attackerStat1", 4)
	c.attrIn(b, "attackerStat2", 6)
	b.fix("levelCapState", 0, "未建模：等級上限狀態")
	b.put("currentClass", "scenario", "Class", int64(c.s.Class))
	b.put("race", "scenario", "Race", int64(c.s.Race))
	b.fixAll(whyTargetChr, 0, "targetCharStat0", "targetCharStat1", "targetCharSpeed", "targetCharStat3")
	for k := 0; k < 4; k++ {
		c.tmplStat(b, fmt.Sprintf("monsterStat%d", k), k)
	}
	b.put("classLevel-1", "scenario", "Level", int64(c.s.Level))
	c.classLevels(b, "classLevel%d")
	in, err := b.done()
	if err != nil {
		return 0, err
	}
	out, err := single("00541c00", in)
	if err != nil {
		return 0, err
	}
	return int64(int32(out[0])), nil
}

// arpen binds 0x541ec0; mode<0 uses the hand's own mode, otherwise 53e250's
// T74 path mode (0x53ecff: 0x542ca0(C,1), drawn from the linked off-slot item).
func (c *fightChar) arpen(g *fightReg, hs []int64, h int, f4, f5, loc, mode int64) (int64, error) {
	b := g.bind("00541ec0")
	if mode < 0 {
		c.arCommon(b, h, f4, f5)
	} else {
		c.arCommonMode(b, h, f4, f5, "node", "0053e250 T74≠-1：0x542ca0(C,1) 自副槽物品 +0x4e 遮罩抽", mode)
	}
	b.put("hitLocation", "node", "0053e250 膠合（部位骰）", loc)
	b.fix("monsterPenDefense", 0, whyNoRuntime)
	for k := 0; k < 9; k++ {
		c.tmplByte(b, fmt.Sprintf("monsterModeAC%d", k), 365+k, true)
	}
	for k := 0; k < 7; k++ {
		c.tmplByte(b, fmt.Sprintf("monsterLocAC%d", k), 358+k, true)
	}
	b.fixAll(whyTargetChr, 0, seqNames("characterLocDef%d", 0, 6)...)
	handSrc2(b, "hand%dBase", hs, hBAR)
	handSrc2(b, "hand%dSkill", hs, hToHit)
	handSrc2(b, "hand%dBonus", hs, hDamageFlat)
	handSrc2(b, "hand%dKind", hs, hWeaponSkill)
	b.fixAll("未建模：暫時加成", 0, "bonusA", "bonusB")
	b.fix("staminaGrade", 0, "未建模：疲勞（取精神飽滿）")
	c.attrIn(b, "attackerINT", 1)
	c.attrIn(b, "attackerDEX", 4)
	c.attrIn(b, "attackerSPD", 5)
	c.attrIn(b, "attackerSEN", 6)
	b.fixAll(whyTargetChr, 0, "targetINT", "targetDEX", "targetSPD", "targetSEN")
	for k, n := range []string{"monsterAttrA", "monsterAttrB", "monsterAttrC", "monsterAttrD"} {
		c.tmplStat(b, n, k)
	}
	in, err := b.done()
	if err != nil {
		return 0, err
	}
	out, err := single("00541ec0", in)
	if err != nil {
		return 0, err
	}
	return int64(int32(out[0])), nil
}

// damageIn binds 0x5420b0 for one strike of hand h.
func (c *fightChar) damageIn(g *fightReg, hs []int64, h int, f4, f5, t70, t74 int64) ([]int64, error) {
	b := g.bind("005420b0")
	b.fix("who", 0, "攻擊者佔隊伍格 0")
	b.put("hand", "scenario", "手（0 主、1 副）", int64(h))
	b.put("mode", "scenario", "Mode／OffMode（依手）", c.mode(h))
	b.fix("tgt", 3, "目標類別＝怪物")
	b.fix("tchar", 4, whyTargetChr+"（任一非攻擊者格）")
	b.fix("mid", 0, whyIndex+"（怪物清單 0）")
	b.fixAll("目標是怪物，角色姿勢選擇格不讀", 0, "g90c", "g910", "g918")
	b.put("f9a4", "node", "0053d870.flag9a4", f4)
	b.put("f9a5", "node", "0053d870.flag9a5", f5)
	b.put("s70", "node", "0053d870.T70", t70)
	b.put("s74", "node", "0053d870.T74", t74)
	b.put("lvi", "scenario", "Class", int64(c.s.Class))
	b.put("lvm1", "scenario", "Level", int64(c.s.Level))
	c.classLevels(b, "lv%d")
	for k := 0; k < 12; k++ {
		switch k {
		case 6:
			if c.main == nil {
				b.put("it6", "scenario", "Main 空手（2＝空物品格）", 2)
			} else {
				b.fix("it6", 0, whyIndex+"（主手＝物品格 0）")
			}
		case 7:
			b.fix("it7", 1, whyIndex+"（副手＝物品格 1，無副手時該格為空）")
		default:
			b.fix(fmt.Sprintf("it%d", k), 2, whyIndex+"（其他裝備格指向空物品格 2）")
		}
	}
	handSrc2(b, "hf%d", hs, hType)
	for h := 0; h < 2; h++ { // 004ee220 emits the raw dword; 5420b0 reads it signed (stat penalty, e.g. -5)
		b.put(fmt.Sprintf("hb%d", h), "node", fmt.Sprintf("004ee220.hand%ddamagePercent", h), int64(int32(hv(hs, h, hDamagePercent))))
	}
	handSrc2(b, "hd%d", hs, hDamageDice)
	b.fix("pwr", 0, "未建模：暫時傷害加成")
	b.fixAll(whyTargetChr, 0, "tA", "tDA", "tS")
	b.fix("mtidx", 0, whyIndex+"（模板欄直接給）")
	b.fix("mA", 0, whyNoRuntime)
	b.fix("mK", 0, whyNoRuntime+"（姿勢正常）")
	b.fix("mD", 0, whyNoRuntime)
	c.tmplByte(b, "tcb", 0xcb, false)
	c.tmplByte(b, "tcf", 0xcf, false)
	c.itemCell(b, "idice0", 0, 74, 4, false)
	c.itemCell(b, "idice1", 1, 74, 4, false)
	b.fixAll("空物品格", 0, seqNames("idice%d", 2, 7)...)
	for k := 0; k < 2; k++ {
		rec, who := c.main, "Main"
		if k == 1 {
			rec, who = c.off, "Off"
		}
		v := int64(255)
		if rec != nil {
			v = int64(rec[0x61])
		}
		b.put(fmt.Sprintf("ib%d", k), "dbs", fmt.Sprintf("ITEMS[%s]+0x61（無則 255）", who), v)
	}
	b.fixAll("空物品格", 255, seqNames("ib%d", 2, 7)...)
	b.fix("g68c510", 0, "事件參數，不影響傷害")
	b.fix("cbase", 0, "角色表基址，只出現在事件參數")
	b.fixAll(whyEmuOnly, 0, seqNames("dd%d", 0, 7)...)
	b.fixAll(whyEmuOnly, 0, "r51", "r4d0", "r4d1")
	c.itemCell(b, "fcls0", 0, 63, 2, false)
	c.itemCell(b, "fcls1", 1, 63, 2, false)
	b.fixAll("空物品格", 0, seqNames("fcls%d", 2, 7)...)
	b.fixAll(whyTargetChr, 0, "fhp", "fst", "fmain", "foff", "fdone")
	return b.done()
}

var (
	fightQ4Once sync.Once
	fightQ4Vec  []int64
)

// effectSlots are the chance slots fed to 0x543270: 1 paralysis, 2 poison, 5 instakill, 6 KO
// (notes/re/weapon-effects.md); the other slots stay 0.
var effectSlots = [...]int{1, 2, 5, 6}

// fighterKO is 53f1c7: ability 0x13 (class table 0x61507c: Fighter only) adds
// _ftol(5479b0(C,0x13,5.0)) to E slot 6; 5479b0 is v for n>20, else (2n+60)*v*0.01f.
func fighterKO(n int) int64 {
	if n > 20 {
		return 5
	}
	return int64(float64(2*n+60) * 5 * float64(float32(0.01)))
}

// effectsIn binds 0x543270 for one hit; ch[5] already includes the Critical Strike term.
func (c *fightChar) effectsIn(g *fightReg, ch [16]int64, pool int64) ([]int64, error) {
	fightQ4Once.Do(func() {
		for _, s := range strings.Fields(fightQ4Effects) {
			v, _ := strconv.ParseInt(s, 10, 64)
			fightQ4Vec = append(fightQ4Vec, v)
		}
	})
	b := g.bind("00543270")
	b.fixAll("53e250 攻擊包清零後不寫（數量、威力、持續）", 0, "quantityBase", "quantityDiceCount", "quantityDieSides",
		"effectPower", "durationBonusPercent", "durationBase", "durationPerPower")
	for k := 0; k < 16; k++ {
		n := fmt.Sprintf("chance%d", k)
		switch k {
		case 5:
			b.put(n, "dbs", "ITEMS[該手武器]+0x55（徒手：4ee220 effect 位元組 5）＋Critical Strike 項（技能 19／25，餘數以 R(25) 進位）", ch[k])
		case 1, 2:
			b.put(n, "dbs", fmt.Sprintf("ITEMS[該手武器]+%#x（徒手：4ee220 effect 位元組 %d）", 0x50+k, k), ch[k])
		case 6:
			b.put(n, "dbs", "ITEMS[該手武器]+0x56（徒手：4ee220 effect 位元組 6）＋戰士 fighterKO(Level)", ch[k])
		default:
			b.fix(n, 0, "未建模：槽 1、2、5、6 以外的特效槽")
		}
	}
	b.put("effectStrength", "scenario", "Level＋min(Level,15)", int64(c.s.Level+min(c.s.Level, 15)))
	b.put("effectPenalty", "dbs", "ITEMS[該手]+0x60＋（T74≠-1）ITEMS[副槽]+0x60（無武器 0；weapon-effects.md E+0x1c）", pool)
	b.put("f.difficulty", "scenario", "Difficulty", int64(c.s.Difficulty))
	b.fix("f.monsterDifficultyClass", 1, "敵對怪 E+0x16＝1（notes/re/monsters-live.md:81；0x55ccb0 依此套難度倍率）")
	for k, el := range []string{"fire", "water", "air", "earth", "mental", "divine"} {
		c.tmplByte(b, "f.templateResistance_"+el, 0x176+k, false)
	}
	c.tmplByte(b, "f.templateLevel", 0x24f, false)
	c.tmplByte(b, "f.templateStatus12Immune", 0x1be, false)
	c.tmplByte(b, "f.templateCategory", 0xcb, false)
	for i, n := range Nodes["00543270"].In {
		if !b.set[i] {
			b.fix(n, fightQ4Vec[i], "calc-qa Q4 情境列（re/emu/cases/00543270.json boundary 81 改欄）")
		}
	}
	return b.done()
}

type fightStrike struct {
	hit, dmg, gout, trig, kill float64    // dmg, gout, trig, kill and status are per hit
	pool                       int64      // E+0x1c of this strike
	status                     [3]float64 // successes of slots 1, 2, 6 (0x543270 stats 0x48, 0x24, 0x4c)
}

// poisonPool is 0x543270's E+0x1c: the striking hand's item u8 +0x60, plus the linked
// off-slot item's when T74≠-1 (0x53efe3..0x53f02a); an empty hand reads 0.
func (c *fightChar) poisonPool(h int, t74 int64) int64 {
	p := int64(0)
	if r := c.handRec(h); r != nil {
		p += int64(r[0x60])
	}
	if t74 != -1 && c.off != nil {
		p += int64(c.off[0x60])
	}
	return p
}

func (c *fightChar) handRec(h int) []byte {
	if h == 1 {
		return c.off
	}
	return c.main
}

// offModes is 0x542ca0(C,1) as a distribution: R(9) ordinal cycled over the set bits
// 0..8 of the off-slot item's u16 +0x4e mask.
func (c *fightChar) offModes() (map[int64]float64, error) {
	if c.off == nil {
		return nil, errors.New("calc fight: T74≠-1 但副手無物品")
	}
	m := int(c.off[0x4e]) | int(c.off[0x4f])<<8
	var bits []int64
	for k := 0; k < 9; k++ {
		if m>>k&1 != 0 {
			bits = append(bits, int64(k))
		}
	}
	if len(bits) == 0 {
		return nil, errors.New("calc fight: 副手物品攻擊模式遮罩為 0（0x542ca0 斷言路徑）")
	}
	out := map[int64]float64{}
	for q, w := range crtWeights(9) {
		out[bits[q%len(bits)]] += w
	}
	return out, nil
}

var (
	effMu    sync.Mutex
	effCache = map[string][5]float64{}
)

// effectOut are the 0x543270 stats offsets read back: 551ba0 calls, then the success
// counts of slots 5 (0x50), 1 (0x48), 2 (0x24) and 6 (0x4c). A slot-5 success zeroes the others.
var effectOut = [...]string{"c551ba0", "stats_50", "stats_48", "stats_24", "stats_4c"}

// effects returns E[0x551ba0 calls] and E[stats increase] for effectOut[1:] for one 0x543270 call.
func effects(in []int64) ([5]float64, error) {
	key := fmt.Sprint(in)
	effMu.Lock()
	v, ok := effCache[key]
	effMu.Unlock()
	if ok {
		return v, nil
	}
	n := Nodes["00543270"]
	oi := [5]int{-1, -1, -1, -1, -1}
	ii := [5]int{-1, -1, -1, -1, -1}
	for i, s := range n.Out {
		for k, name := range effectOut {
			if s == name {
				oi[k] = i
			}
		}
	}
	for i, s := range n.In {
		for k, name := range effectOut {
			if k > 0 && s == name {
				ii[k] = i
			}
		}
	}
	for k := range oi {
		if oi[k] < 0 || (k > 0 && ii[k] < 0) {
			return v, fmt.Errorf("calc fight: 00543270 layout lacks %s", effectOut[k])
		}
	}
	e, err := fightExpect(func(r Rand) []float64 {
		o := n.F(append([]int64(nil), in...), r)
		x := []float64{float64(o[oi[0]])}
		for k := 1; k < 5; k++ {
			x = append(x, float64(o[oi[k]]-in[ii[k]]))
		}
		return x
	})
	if err != nil {
		return v, err
	}
	copy(v[:], e)
	effMu.Lock()
	effCache[key] = v
	effMu.Unlock()
	return v, nil
}

// strike is 0x53e250's resolution of one strike (ac < 2) as a distribution.
func (w *FightWorld) strike(c *fightChar, g *fightReg, hs []int64, h int, f4, f5, t70, t74 int64) (fightStrike, error) {
	var st fightStrike
	chance, err := c.arhit(g, hs, h, f4, f5, 0)
	if err != nil {
		return st, err
	}
	c2, err := c.arhit(g, hs, h, f4, f5, 1)
	if err != nil {
		return st, err
	}
	p := int32(0)
	if c2 < 100 {
		x := float64((100 - c2) * (100 - c2) * (100 - c2))
		v := chdFtol(chdAdd(chdMul(chdF(x), chdF(w.fumA)), chdF(w.fumB)))
		p = int32(uint32(1) * v * 10 / 100) // ponytail: fumble list length n=1 (543dc0 not modelled)
	}
	p = min(100, max(0, p))
	modes := map[int64]float64{-1: 1}
	if t74 != -1 {
		if modes, err = c.offModes(); err != nil {
			return st, err
		}
	}
	type penRow struct {
		w   float64
		pen [7]int64
	}
	var pens []penRow
	for mode, mw := range modes {
		r := penRow{w: mw}
		for loc := range r.pen {
			if r.pen[loc], err = c.arpen(g, hs, h, f4, f5, int64(loc), mode); err != nil {
				return st, err
			}
		}
		pens = append(pens, r)
	}
	// Hit location: R(100) against the template's cumulative weights, default 3.
	var locW [7]float64
	for r, q := range crtWeights(100) {
		loc, sum := 3, 0
		for i := 0; i < 7; i++ {
			sum += int(c.tmpl[0x15f+i])
			if r < sum {
				loc = i
				break
			}
		}
		locW[loc] += q
	}
	for i, q := range crtWeights(100) {
		roll := int64(i + 1)
		e1 := roll <= 5
		b4 := int64(100-p) < roll
		if e1 && b4 {
			return st, errors.New("calc fight: 未建模：自動命中且失手（0x545a00 路徑）")
		}
		if roll > 95 || b4 || (!e1 && chance < roll) {
			continue
		}
		for _, pr := range pens {
			for loc, ql := range locW {
				if e1 || pr.pen[loc] >= roll {
					st.hit += q * ql * pr.w
				}
			}
		}
	}
	in, err := c.damageIn(g, hs, h, f4, f5, t70, t74)
	if err != nil {
		return st, err
	}
	dn := Nodes["005420b0"]
	d, err := fightExpect(func(r Rand) []float64 {
		o := dn.F(append([]int64(nil), in...), r)
		pos := 0.0
		if o[wdnORet] > 0 {
			pos = 1
		}
		return []float64{float64(o[wdnORet]), float64(o[wdnOGout]), pos}
	})
	if err != nil {
		return st, err
	}
	st.dmg, st.gout = d[0], d[1]
	// Effects pack: chance5 base plus Critical Strike's v/25 (+1 with R(25) < v%25).
	kind := hv(hs, h, hType)
	rec := c.handRec(h)
	var ch [16]int64
	for _, k := range effectSlots {
		if kind == 0 {
			ch[k] = hv(hs, h, hEffect0+k/4) >> (8 * (k % 4)) & 0xff // effect dwords are chance0..15 bytes
		} else {
			ch[k] = int64(rec[0x50+k])
			if t74 != -1 {
				ch[k] = (ch[k] + int64(c.off[0x50+k])) & 0xff // 0x53efe3: byte add, wraps
			}
		}
	}
	if c.s.Class == 0 {
		ch[6] += fighterKO(c.s.Level)
	}
	pool := c.poisonPool(h, t74)
	st.pool = pool
	base := ch[5]
	var vs []int64
	var ws []float64
	// 0x53f0bc: combatSkill 0x10 always adds skill 19; 0x11 needs ability 4 (skill 17) for
	// weaponSkill 7/8 and for 9 with item W+0x3f == 13, ability 5 (skill 19) for other 9s.
	v, crit := int64(0), false
	feat := func(code, sk int) {
		if w.x.HasFeature(c.s.Class, c.s.Race, code) {
			v, crit = int64(c.skill[sk]), true
		}
	}
	switch hv(hs, h, hCombatSkill) {
	case 0x10:
		v, crit = int64(c.skill[19]), true
	case 0x11:
		switch wsk := hv(hs, h, hWeaponSkill); {
		case wsk == 7 || wsk == 8:
			feat(4, 17)
		case wsk == 9 && rec != nil && int(rec[0x3f])|int(rec[0x40])<<8 == 13:
			feat(4, 17)
		case wsk == 9:
			feat(5, 19)
		}
	}
	switch {
	case crit:
		lo := 0.0
		for k, q := range crtWeights(25) {
			if int64(k) < v%25 {
				lo += q
			}
		}
		if v%25 == 0 {
			lo = 0
		}
		vs, ws = []int64{base + v/25, base + v/25 + 1}, []float64{1 - lo, lo}
	default:
		vs, ws = []int64{base}, []float64{1}
	}
	for i := range vs {
		if ws[i] == 0 {
			continue
		}
		// One 0x543270 call per slot keeps the enumeration small. The slots roll
		// independently and only a slot-5 success (state 0x12) stops or zeroes the
		// others, so slot k counts with probability P_k*(1-P_5).
		var one [16]int64
		one[5] = vs[i] & 0xff
		ein, err := c.effectsIn(g, one, pool)
		if err != nil {
			return st, err
		}
		e, err := effects(ein)
		if err != nil {
			return st, err
		}
		st.trig += ws[i] * d[2] * e[0]
		st.kill += ws[i] * d[2] * e[1]
		for k, slot := range [3]int{1, 2, 6} {
			if ch[slot] == 0 {
				continue
			}
			one = [16]int64{}
			one[slot] = ch[slot]
			if ein, err = c.effectsIn(g, one, pool); err != nil {
				return st, err
			}
			ek, err := effects(ein)
			if err != nil {
				return st, err
			}
			st.status[k] += ws[i] * d[2] * ek[2+k] * (1 - e[1])
		}
	}
	return st, nil
}

// Fight returns the round's expectations; reg (may be nil) collects input provenance.
func (w *FightWorld) Fight(s FightScenario, reg *fightReg) ([]FightKV, error) {
	if reg == nil {
		reg = newFightReg()
	}
	c, err := w.fightChar(s)
	if err != nil {
		return nil, err
	}
	hs, err := c.hands(reg)
	if err != nil {
		return nil, err
	}
	var kv []FightKV
	var swings, damage float64
	survive := 1.0
	free := [3]float64{1, 1, 1} // P(no slot 1/2/6 success this round)
	var poolW, poolN float64    // E+0x1c weighted by expected poison successes
	for h := 0; h < 2; h++ {
		if h == 1 && hv(hs, 1, hEnabled) == 0 {
			break // 004ee220 enables the second hand for an offhand weapon or an unarmed punch
		}
		attacks := float64(hv(hs, h, hAttacks))
		cache := map[[4]int64]fightStrike{}
		stats := func(ls int64) (a fightAttack, err error) {
			outs, err := c.swings(reg, hs, h, ls)
			if err != nil {
				return a, err
			}
			for _, o := range outs {
				ret, t68 := o.Out[soRet], int64(int32(o.Out[soT68]))
				if ret == 0 || t68 <= 0 {
					a.atk = append(a.atk, fightAtkOut{p: o.Probability})
					a.alive += o.Probability
					for k := range a.sAlive {
						a.sAlive[k] += o.Probability
					}
					continue
				}
				key := [4]int64{o.Out[soFlag9a4], o.Out[soFlag9a5], int64(int32(o.Out[soT70])), int64(int32(o.Out[soT74]))}
				st, ok := cache[key]
				if !ok {
					if st, err = w.strike(c, reg, hs, h, key[0], key[1], key[2], key[3]); err != nil {
						return a, err
					}
					cache[key] = st
				}
				pt := o.Probability * float64(t68)
				a.atk = append(a.atk, fightAtkOut{o.Probability, int(t68), st.hit, st.kill})
				a.n += pt
				a.hits += pt * st.hit
				a.dmg += pt * st.hit * st.dmg
				a.gout += pt * st.hit * st.gout
				a.trig += pt * st.hit * st.trig
				a.kill += pt * st.hit * st.kill
				a.alive += o.Probability * math.Pow(1-st.hit*st.kill, float64(t68))
				for k := range a.sAlive {
					a.sAlive[k] += o.Probability * math.Pow(1-st.hit*st.status[k], float64(t68))
					a.sHit[k] += pt * st.hit * st.status[k]
				}
				a.poolW += pt * st.hit * st.status[1] * float64(st.pool)
			}
			return a, nil
		}
		a0, err := stats(0)
		if err != nil {
			return nil, err
		}
		// LS (F02–F04): the round flag is rolled once with p1 and spent on the first main-hand
		// attack, so at most one attack runs with lsFlag=1; the node itself drops non-swords.
		p1, a1 := 0.0, a0
		if h == 0 && w.x.HasFeature(c.s.Class, c.s.Race, 0x10) {
			p1 = float64(lsLevelScale(int64(c.s.Level))) / 100
			if a1, err = stats(1); err != nil {
				return nil, err
			}
		}
		lin := func(x0, x1 float64) float64 { return attacks*x0 + p1*(x1-x0) }
		n, hits, dmg := lin(a0.n, a1.n), lin(a0.hits, a1.hits), lin(a0.dmg, a1.dmg)
		gout, trig, kill := lin(a0.gout, a1.gout), lin(a0.trig, a1.trig), lin(a0.kill, a1.kill)
		poolW += lin(a0.poolW, a1.poolW)
		var sHit [3]float64
		for k := range sHit {
			sHit[k] = lin(a0.sHit[k], a1.sHit[k])
		}
		poolN += sHit[1]
		div := func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		}
		p := "main."
		if h == 1 {
			p = "off."
		}
		atks := make([][]fightAtkOut, int(attacks))
		for i := range atks {
			atks[i] = a0.atk
		}
		if len(atks) > 0 && p1 > 0 {
			atks[0] = nil
			for _, o := range a0.atk {
				atks[0] = append(atks[0], fightAtkOut{o.p * (1 - p1), o.n, o.q, o.k})
			}
			for _, o := range a1.atk {
				atks[0] = append(atks[0], fightAtkOut{o.p * p1, o.n, o.q, o.k})
			}
		}
		hp := readDice(c.tmpl[0xd6:])
		kv = append(kv, FightKV{p + "kills", fightKills(float64(hp.Z)+float64(hp.N)*(float64(hp.S)+1)/2, div(dmg, hits), atks)},
			FightKV{p + "swings", n}, FightKV{p + "hit", div(hits, n)}, FightKV{p + "dmg_per_hit", div(dmg, hits)},
			FightKV{p + "dice_groups_mean", div(gout, hits)}, FightKV{p + "instakill_trigger", div(trig, hits)},
			FightKV{p + "instakill", div(kill, hits)}, FightKV{p + "paralyze", div(sHit[0], hits)},
			FightKV{p + "poison", div(sHit[1], hits)}, FightKV{p + "ko", div(sHit[2], hits)})
		swings += n
		damage += dmg
		mix := func(x0, x1 float64) float64 { return math.Pow(x0, attacks-1) * ((1-p1)*x0 + p1*x1) }
		survive *= mix(a0.alive, a1.alive)
		for k := range free {
			free[k] *= mix(a0.sAlive[k], a1.sAlive[k])
		}
	}
	pool := 0.0
	if poolN > 0 {
		pool = poolW / poolN
	}
	out := []FightKV{{"round.swings", swings}}
	out = append(out, kv...)
	return append(out, FightKV{"round.damage", damage}, FightKV{"round.instakill", 1 - survive},
		FightKV{"round.paralyze", 1 - free[0]}, FightKV{"round.poison", 1 - free[1]}, FightKV{"round.ko", 1 - free[2]},
		FightKV{"round.poison_pool", pool}), nil
}

type fightAttack struct {
	n, hits, dmg, gout, trig, kill, alive, poolW float64
	sAlive, sHit                                 [3]float64
	atk                                          []fightAtkOut
}

// fightAtkOut is one outcome of an attack: probability p, n swings, per-swing hit q and per-hit instakill k.
type fightAtkOut struct {
	p    float64
	n    int
	q, k float64
}

// fightKills is the expected monsters killed per round by one hand against a group of identical
// monsters of mean HP h. Every swing of an attack hits the attack's target; the swings left after
// the target dies are lost (MG 2026-11-01), and the next attack starts on a fresh monster. A target
// that survives an attack keeps its damage for the next one. ponytail: a kill needs ceil(h/dph) hits
// at the mean damage per hit, no damage variance; hands are independent (no shared target).
func fightKills(h, dph float64, atks [][]fightAtkOut) float64 {
	if dph <= 0 {
		return 0
	}
	m := int(math.Max(1, math.Ceil(h/dph)))
	st := make([]float64, m+1) // st[r]: P(current target needs r more hits)
	st[m] = 1
	var kills float64
	for _, outs := range atks {
		next := make([]float64, m+1)
		for r := 1; r <= m; r++ {
			if st[r] == 0 {
				continue
			}
			for _, o := range outs {
				cur := make([]float64, m+1)
				cur[r] = st[r] * o.p
				for i := 0; i < o.n; i++ {
					nx := make([]float64, m+1)
					for j := 1; j <= m; j++ {
						x := cur[j]
						if x == 0 {
							continue
						}
						die := o.q * o.k
						if j == 1 {
							die = o.q
						}
						kills += x * die
						next[m] += x * die
						nx[j] += x * (1 - o.q)
						if j > 1 {
							nx[j-1] += x * o.q * (1 - o.k)
						}
					}
					cur = nx
				}
				for j := 1; j <= m; j++ {
					next[j] += cur[j]
				}
			}
		}
		st = next
	}
	return kills
}

// FightInputsTSV renders the provenance of every node input the given scenarios touched.
func (w *FightWorld) FightInputsTSV(ss []FightScenario) (string, error) {
	reg := newFightReg()
	for _, s := range ss {
		if _, err := w.Fight(s, reg); err != nil {
			return "", err
		}
	}
	var sb strings.Builder
	sb.WriteString("node\tinput\tsource\tvalue\treason\n")
	for _, a := range FightNodes {
		rows := reg.rows[a]
		names := Nodes[a].In
		for _, n := range names {
			r, ok := rows[n]
			if !ok {
				return "", fmt.Errorf("calc fight: %s input %s never bound", a, n)
			}
			fmt.Fprintf(&sb, "%s\t%s\t%s\t%s\t%s\n", a, n, r.src, r.val, r.why)
		}
	}
	return sb.String(), nil
}
