package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
)

// visibility_awareness_update 0x5049c0(group, partyMode, arg3), cdecl.
//
// Party mode (partyMode low byte != 0): the group's object sees the party when
// the party is inside the camera clip range and the sight rule 0x505a40 (light,
// facing, weakest party stat) reaches the distance; the party notices the group
// when any able character's sight rule reaches it (stealth, character stats).
// Noticing plays a voice (NPC greeting or a 2-way random shout), starts the
// group's notice timers and records both positions. Monster mode: refreshes the
// group's per-monster sight list (+0x2b6), appending new entries on the heap.
//
// Composition (coordinator ruling 2026-10-11, MG numeric-only scope): the
// registered exclusions 0x4c4810 0x4c4870 0x4c4920 0x4c4a20 0x4c4af0 0x4c4b70
// 0x539870 0x4c77f0 0x4c78e0 0x4c4f80 0x4c5150 0x50cf70, the LOS ray 0x434f20,
// the voice player 0x52e690, the message queue 0x58ac00 and the D4 formula
// 0x518150 are address stubs in the case: their results and documented writes
// come from case inputs, their arguments and call counts are outputs. Distance
// and heading (0x4c7cb0, 0x4c7dd0, 0x4be420, 0x517ce0) run natively and reuse the
// shared helpers. F runs the real randomPartyMemberNode for 0x518150; Emu returns
// the stub result like the emulator.
//
// Fixture layout mirrors re/emu/cases/005049c0.json: arena objects at
// 0x20000000+offset, the bump heap at 0x50000000 (8-aligned, zero-filled).
const (
	vauArena = 0x20000000
	vauGrp   = vauArena + 0x0
	vauMon0  = vauArena + 0x400
	vauMon1  = vauArena + 0x800
	vauSobj  = vauArena + 0x1000
	vauMo0   = vauArena + 0x1200
	vauMo1   = vauArena + 0x1400
	vauTpl0  = vauArena + 0x2000 // template id t at +0x300*t (cache 0x6840c7+4*t)
	vauStub  = vauArena + 0x3100
	vauView  = vauArena + 0x4200
	vauPsh   = vauArena + 0x4300
	vauClock = vauArena + 0x4500
	vauNpcl  = vauArena + 0x4e00
	vauNpcd  = vauArena + 0x4e20
	vauNpc0  = vauArena + 0x4f00
	vauNpc1  = vauArena + 0x5000
	vauNsub0 = vauArena + 0x5100
	vauNsub1 = vauArena + 0x5200
	vauGl    = vauArena + 0x5300
	vauGdata = vauArena + 0x5310
	vauLst   = vauArena + 0x5400
	vauLdata = vauArena + 0x5410
	vauEnt0  = vauArena + 0x5500 // ent k at +0x40*k
	vauEntD  = vauArena + 0x55c0
	vauVoice = vauArena + 0x5600
	vauPres  = vauArena + 0x6000 // presence record, stride 0x106 (*0x685178)
	vauChars = vauArena + 0x7000 // characters, stride 0x1862 (*0x685174)
	vauHeap  = 0x50000000
	vauTick  = 4660       // case "tick": GetTickCount (0x5eb02c) value
	vauText  = 0x11110774 // fixture [[0x68c09c]+0x774]; +4 for +0x778
	vauTable = 0x68d6f2   // 0x536110 flag table, stride 14 (zero in the exe image)
	vauHeapN = 0xe0       // heap bytes exposed as outputs h00..hdc
)

type vauField struct {
	name   string
	at     uint32 // 0 for call arguments
	size   int
	signed bool
}

type vauStubDef struct {
	addr  string
	rec   []vauRec
	count uint32
}

type vauRec struct {
	arg, n int
	at     uint32
}

var vauIn, vauOut, vauStubs = vauLayout()

func vauLayout() ([]vauField, []vauField, map[string]*vauStubDef) {
	var in, out []vauField
	o := func(base uint32, off int) uint32 { return base + uint32(off) }
	add := func(name string, at uint32, size int) { in = append(in, vauField{name, at, size, false}) }
	add("gate", 0x68ec78, 4)
	in = append(in, vauField{"arg2", 0, 4, false}, vauField{"arg3", 0, 4, false})
	for _, f := range []struct{ off, size int }{{0x14, 1}, {0x15, 1}, {0x16, 1}, {0x2b, 4}, {0x87, 4}, {0x107, 4}, {0x24b, 1}, {0x28a, 1}, {0x28e, 4}, {0x34c, 1}, {0x354, 4}, {0x37a, 1}, {0x37c, 1}, {0x0, 4}, {0x8, 4}} {
		add(fmt.Sprintf("s%x", f.off), o(vauGrp, f.off), f.size)
	}
	add("ox", vauSobj+0x118, 4)
	add("oy", vauSobj+0x11c, 4)
	add("oz", vauSobj+0x120, 4)
	add("oh", vauSobj+0x190, 4)
	add("orad", vauSobj+0x18c, 4)
	add("ofac", vauSobj+0xec, 4)
	add("clipLo", vauStub+520, 4)
	add("clipHi", vauStub+524, 4)
	add("pf", vauView+4, 4)
	add("px", vauView+0x8c, 4)
	add("py", vauView+0x90, 4)
	add("pz", vauView+0x94, 4)
	add("prad", vauPsh+0x174, 4)
	add("qx", vauPsh+0x100, 4)
	add("qy", vauPsh+0x104, 4)
	add("qz", vauPsh+0x108, 4)
	for t := 0; t < 3; t++ {
		for _, f := range []int{0xcb, 0xcd, 0xd0, 0x248, 0x24f, 0x25f} {
			add(fmt.Sprintf("t%d_%x", t, f), o(vauTpl0+0x300*uint32(t), f), 1)
		}
	}
	for i := 0; i < 8; i++ {
		add(fmt.Sprintf("c%d_pres", i), o(vauPres, i*0x106), 1)
		for _, f := range []struct{ off, size int }{{0x69, 4}, {0x89, 4}, {0x161, 4}, {0x3dd, 1}, {0xa31, 4}, {0xb01, 4}, {0xb11, 4}} {
			add(fmt.Sprintf("c%d_%x", i, f.off), o(vauChars, i*0x1862+f.off), f.size)
		}
	}
	for _, g := range []struct {
		name string
		at   uint32
		size int
	}{{"g683f94", 0x683f94, 1}, {"g68749b", 0x68749b, 1}, {"g68749d", 0x68749d, 1}, {"g6874ff", 0x6874ff, 1},
		{"g60a390", 0x60a390, 4}, {"g683f97", 0x683f97, 1}, {"g686a70", 0x686a70, 4}, {"time", 0x686a48, 4},
		{"t6c", 0x689b6c, 4}, {"t70", 0x689b70, 4}, {"t74", 0x689b74, 4}, {"g683fc5", 0x683fc5, 1}, {"fact2ee", 0x689e66, 1}} {
		add(g.name, g.at, g.size)
	}
	add("clock", vauClock+0x30, 4)
	in = append(in, vauField{"ncount", vauNpcl + 4, 4, true})
	for k := 0; k < 2; k++ {
		npc, sub := uint32(vauNpc0+0x100*k), uint32(vauNsub0+0x100*k)
		add(fmt.Sprintf("n%d_54", k), sub+0x54, 1)
		add(fmt.Sprintf("n%d_58", k), sub+0x58, 4)
		for _, f := range []int{0x2d, 0x2e, 0xe9} {
			add(fmt.Sprintf("n%d_%x", k, f), o(npc, f), 1)
		}
	}
	add("gcount", vauGl+8, 4)
	for k := 0; k < 2; k++ {
		mon, mo := uint32(vauMon0+0x400*k), uint32(vauMo0+0x200*k)
		for _, f := range []struct{ off, size int }{{0x0, 4}, {0x8, 4}, {0x14, 1}, {0x15, 1}, {0x2b, 4}} {
			add(fmt.Sprintf("m%d_%x", k, f.off), o(mon, f.off), f.size)
		}
		for _, f := range []struct {
			n   string
			off int
		}{{"x", 0x118}, {"y", 0x11c}, {"z", 0x120}, {"h", 0x190}, {"rad", 0x18c}} {
			add(fmt.Sprintf("mo%d_%s", k, f.n), o(mo, f.off), 4)
		}
	}
	in = append(in, vauField{"lcap", vauLst + 4, 4, true}, vauField{"lcount", vauLst + 8, 4, true})
	for k := 0; k < 3; k++ {
		for j := 0; j < 13; j++ {
			add(fmt.Sprintf("e%d_%x", k, 4*j), uint32(vauEnt0+0x40*k+4*j), 4)
		}
	}
	add("dId", vauEntD, 4)
	ret := map[string]uint32{}
	sret := func(name string, off int, signed bool) {
		ret[name] = vauStub + uint32(off)
		in = append(in, vauField{name, vauStub + uint32(off), 4, signed})
	}
	for _, s := range []struct {
		n   string
		off int
	}{{"r4810", 600}, {"w4870a", 604}, {"w4870b", 608}, {"r4920", 612}, {"w4a20a", 616}, {"w4a20b", 620},
		{"r4af0_0", 624}, {"r4af0_1", 628}, {"w4b70a", 632}, {"w4b70b", 636}, {"r539870", 640}} {
		sret(s.n, s.off, false)
	}
	for k := 0; k < 6; k++ {
		sret(fmt.Sprintf("r434f20_%d", k), 644+4*k, false)
	}
	sret("r518150", 672, true)
	sret("r52e690", 676, false)

	put := func(name string, at uint32, size int) { out = append(out, vauField{name, at, size, false}) }
	for _, f := range []int{0x24d, 0x28a, 0x28b, 0x28c, 0x28d, 0x2aa, 0x2ab, 0x34c, 0x34d, 0x34e, 0x34f, 0x350, 0x353, 0x370} {
		put(fmt.Sprintf("s%x", f), o(vauGrp, f), 1)
	}
	for _, f := range []int{0x28e, 0x292, 0x296, 0x29a, 0x29e, 0x2a2, 0x2a6, 0x354, 0x358, 0x35c, 0x360, 0x364, 0x368, 0x36c} {
		put(fmt.Sprintf("s%x", f), o(vauGrp, f), 4)
	}
	for k := 0; k < 3; k++ {
		for j := 0; j < 13; j++ {
			put(fmt.Sprintf("e%d_%x", k, 4*j), uint32(vauEnt0+0x40*k+4*j), 4)
		}
	}
	for j := 0; j < 8; j++ {
		put(fmt.Sprintf("ld%d", j), uint32(vauLdata+4*j), 4)
	}
	put("Lldata", vauLst, 4)
	put("Llcap", vauLst+4, 4)
	put("Llcount", vauLst+8, 4)
	for j := 0; j < vauHeapN; j += 4 {
		put(fmt.Sprintf("h%02x", j), uint32(vauHeap+j), 4)
	}
	put("n0_2d", vauNpc0+0x2d, 1)
	put("n1_2d", vauNpc1+0x2d, 1)
	put("v30", vauVoice+0x30, 4)
	put("v34", vauVoice+0x34, 4)
	for k := -128; k < 128; k++ {
		put(fmt.Sprintf("m536_%d", k+128), uint32(vauTable+14*k), 1)
	}
	put("t6c", 0x689b6c, 4)
	put("t70", 0x689b70, 4)
	put("t74", 0x689b74, 4)

	stubs := map[string]*vauStubDef{}
	next := uint32(vauStub + 800)
	stub := func(addr string, recs ...struct {
		arg, n int
		name   string
	}) {
		s := &vauStubDef{addr: addr}
		for _, r := range recs {
			s.rec = append(s.rec, vauRec{r.arg, r.n, next})
			for k := 0; k < r.n; k++ {
				name := r.name
				if r.n > 1 {
					name = fmt.Sprintf("%s_%d", r.name, k)
				}
				put(name, next+uint32(4*k), 4)
			}
			next += uint32(4 * r.n)
		}
		s.count = next
		put("n_"+addr, next, 4)
		next += 4
		stubs[addr] = s
	}
	type rc = struct {
		arg, n int
		name   string
	}
	stub("4c4810")
	stub("4c4870")
	stub("4c4920", rc{1, 1, "a4920"})
	stub("4c4a20", rc{3, 1, "a4a20_3"})
	stub("4c4af0", rc{1, 2, "a4af0"})
	stub("4c4b70", rc{1, 2, "a4b70"})
	stub("539870", rc{1, 1, "a539870"})
	stub("4c77f0")
	stub("4c78e0")
	stub("434f20", rc{3, 6, "a434_3"}, rc{4, 6, "a434_4"}, rc{5, 6, "a434_5"}, rc{6, 6, "a434_6"}, rc{7, 6, "a434_7"})
	stub("4c4f80", rc{1, 1, "a4f80"})
	stub("4c5150", rc{1, 1, "a5150"})
	stub("50cf70", rc{1, 1, "a50cf70"}, rc{2, 1, "a50cf70_2"})
	stub("52e690", rc{1, 1, "a52e_1"}, rc{2, 1, "a52e_2"}, rc{3, 1, "a52e_3"}, rc{4, 1, "a52e_4"}, rc{5, 1, "a52e_5"})
	stub("58ac00", rc{1, 1, "a58ac_1"}, rc{2, 1, "a58ac_2"}, rc{3, 1, "a58ac_3"}, rc{4, 1, "a58ac_4"}, rc{5, 1, "a58ac_5"})
	stub("518150", rc{1, 1, "a5181_1"}, rc{2, 1, "a5181_2"}, rc{3, 1, "a5181_3"}, rc{4, 1, "a5181_4"})
	for name, at := range ret {
		stubs["ret:"+name] = &vauStubDef{addr: name, count: at}
	}
	return in, out, stubs
}

func init() {
	names := func(f []vauField) []string {
		var s []string
		for _, x := range f {
			s = append(s, x.name)
		}
		return s
	}
	Nodes["005049c0"] = Node{Addr: "005049c0", In: names(vauIn), Out: names(vauOut),
		F:   func(in []int64, r Rand) []int64 { return visibilityAwarenessUpdate(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return visibilityAwarenessUpdate(in, r, false) }}
}

func visibilityAwarenessUpdateNode(in []int64, r Rand) []int64 {
	return visibilityAwarenessUpdate(in, r, true)
}

// vauMem is the sparse little-endian memory of the fixture; unset bytes read 0.
type vauMem struct {
	b    map[uint32]byte
	heap uint32
	r    Rand
	real bool
	in   []int64
}

func (m *vauMem) u8(a uint32) uint32 { return uint32(m.b[a]) }
func (m *vauMem) u32(a uint32) uint32 {
	return uint32(m.b[a]) | uint32(m.b[a+1])<<8 | uint32(m.b[a+2])<<16 | uint32(m.b[a+3])<<24
}
func (m *vauMem) w8(a, v uint32) { m.b[a] = byte(v) }
func (m *vauMem) w32(a, v uint32) {
	for k := uint32(0); k < 4; k++ {
		m.b[a+k] = byte(v >> (8 * k))
	}
}
func (m *vauMem) f32(a uint32) float64 {
	v := float64(math.Float32frombits(m.u32(a)))
	if math.IsInf(v, 0) || math.IsNaN(v) {
		panic(fmt.Sprintf("visibility awareness: non-finite float32 at %#x", a))
	}
	return v
}
func (m *vauMem) ret(name string) uint32 { return m.u32(vauStubs["ret:"+name].count) }

// call records the stub arguments (record n>1: call k at +4k) and bumps the count.
func (m *vauMem) call(addr string, args ...uint32) {
	s := vauStubs[addr]
	k := m.u32(s.count)
	for _, r := range s.rec {
		at := r.at
		if r.n > 1 {
			if int(k) >= r.n {
				panic("visibility awareness: stub " + addr + " called more often than the case records")
			}
			at += 4 * k
		}
		m.w32(at, args[r.arg-1])
	}
	m.w32(s.count, k+1)
}

// malloc: emu bump allocator, max(8, (n+7)&^7) bytes, zero-filled.
func (m *vauMem) malloc(n uint32) uint32 {
	at := m.heap
	m.heap += max(8, (n+7)&^7)
	return at
}

func vauExeU32(va uint32) uint32 { return binary.LittleEndian.Uint32(ticRead(va, 4)) }
func vauExeF32(va uint32) float64 {
	return float64(math.Float32frombits(binary.LittleEndian.Uint32(ticRead(va, 4))))
}
func vauExeF64(va uint32) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(ticRead(va, 8)))
}

func vauB(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// vauTemplate: 0x4e5720 cache 0x6840c7+4*id; the fixture loads ids 0..2.
func vauTemplate(id uint32) uint32 {
	if id > 2 {
		panic("visibility awareness: template id outside fixture cache 0..2")
	}
	return vauTpl0 + 0x300*id
}

// vauPos: object position +0x118 with +0x190 added to Y and stored as float32.
func (m *vauMem) vauPos(obj uint32) [3]uint32 {
	y, _ := monsterAttackTargetOp((*big.Float).Add, monsterAttackTargetExt(m.f32(obj+0x11c)), monsterAttackTargetExt(m.f32(obj+0x190))).Float32()
	return [3]uint32{m.u32(obj + 0x118), math.Float32bits(y), m.u32(obj + 0x120)}
}

func vauF(p [3]uint32) [3]float64 {
	var f [3]float64
	for i, b := range p {
		f[i] = float64(math.Float32frombits(b))
		if math.IsInf(f[i], 0) || math.IsNaN(f[i]) {
			panic("visibility awareness: non-finite position")
		}
	}
	return f
}

// vauClip: srCamera::getClipRange far value as float32; <= 0 or unordered is
// the srAssertFail at 0x504a0e / 0x505a62.
func (m *vauMem) clip() float64 {
	d := math.Float64frombits(uint64(m.u32(vauStub+524))<<32 | uint64(m.u32(vauStub+520)))
	if !(d > 0) {
		panic("visibility awareness: clip range assertion (0x504a0e)")
	}
	c := float64(float32(d))
	if math.IsInf(c, 0) {
		panic("visibility awareness: clip range overflows float32")
	}
	return c
}

// 0x4c7cb0: object (+0x118) to the party ([0x65a0f8]+0x8c), radii +0x18c and [0x659c0c]+0x174.
func (m *vauMem) partyDistance() *big.Float {
	return monsterAttackTargetDistance([3]float64{m.f32(vauSobj + 0x118), m.f32(vauSobj + 0x11c), m.f32(vauSobj + 0x120)},
		[3]float64{m.f32(vauView + 0x8c), m.f32(vauView + 0x90), m.f32(vauView + 0x94)}, true, m.f32(vauSobj+0x18c), m.f32(vauPsh+0x174))
}

// 0x4c7dd0: two objects (+0x118), radii +0x18c.
func (m *vauMem) objectDistance(a, b uint32) *big.Float {
	return monsterAttackTargetDistance([3]float64{m.f32(a + 0x118), m.f32(a + 0x11c), m.f32(a + 0x120)},
		[3]float64{m.f32(b + 0x118), m.f32(b + 0x11c), m.f32(b + 0x120)}, false, m.f32(a+0x18c), m.f32(b+0x18c))
}

func vauF32(x *big.Float) float64 { f, _ := x.Float32(); return float64(f) }

// vauAmp: template +0xcb == 4 gives byte(+0x24f * 5) capped at 0x7d.
func (m *vauMem) amp(tmpl uint32) uint32 {
	if m.u8(tmpl+0xcb) != 4 {
		return 0
	}
	return min(m.u8(tmpl+0x24f)*5&0xff, 0x7d)
}

type vauSight struct {
	a, b                                [3]float64
	facing                              float64
	a7, a8, a9, a10, a11, a12, a13, a14 uint32
	dist                                float64
}

// 0x505a40: sight reach (extended). a = viewer, b = target, facing a6.
func (m *vauMem) sight(s vauSight, clip float64) *big.Float {
	m.clip() // 0x505a43 re-reads the clip range and repeats the assertion
	ext, op := monsterAttackTargetExt, monsterAttackTargetOp
	t := formationRangeConstants
	if formationRangeError != nil {
		panic(formationRangeError)
	}
	// 0x4be420: heading from a to b, float32.
	dx := op((*big.Float).Sub, ext(s.b[0]), ext(s.a[0]))
	dz := op((*big.Float).Sub, ext(s.b[2]), ext(s.a[2]))
	var ang float64
	if dz.Sign() == 0 {
		ang = t[1] // 0x5ed1e8
		if dx.Sign() > 0 {
			ang = t[0] // 0x5ec3fc
		}
	} else {
		x, _ := dx.Float64()
		z, _ := dz.Float64()
		ang = float64(float32(math.Atan2(x, z)))
	}
	// 0x517ce0(facing, ang): smaller wrapped difference, float32 first leg.
	pi := ext(t[3])
	x32, _ := op((*big.Float).Sub, ext(s.facing), ext(ang)).Float32()
	w32, _ := mfReduce(op((*big.Float).Add, ext(float64(x32)), pi)).Float32()
	y32, _ := op((*big.Float).Sub, ext(ang), ext(s.facing)).Float32()
	least := mfReduce(op((*big.Float).Add, ext(float64(y32)), pi))
	if w := ext(float64(w32)); w.Cmp(least) < 0 {
		least = w
	}
	front := least.Cmp(ext(vauExeF64(0x5ec2b0))) < 0 // 0x5ec2b0 f64 0.78539815 (pi/4)
	if s.a9&0xff != 0 && s.a10&0xff == 0 {
		return ext(0) // 0x5ebb34
	}
	v := int32(100)
	if !front && s.a14&0xff == 0 && s.a13 == 0 && s.dist != 0 {
		n := new(big.Float).SetPrec(64).SetUint64(uint64(s.a7 - 2*s.a11))
		q := op((*big.Float).Mul, n, op((*big.Float).Sub, ext(clip), ext(s.dist)))
		q = op((*big.Float).Quo, q, ext(s.dist))
		i, _ := q.Int(nil) // _ftol truncates toward zero
		if !i.IsInt64() {
			panic("visibility awareness: _ftol int64 overflow (0x505b1b)")
		}
		v = int32(uint32(i.Int64()))
	}
	if s.a12 != 0 {
		if s.a13 != 0 {
			v -= int32(5 * s.a12)
		} else {
			v -= int32(10 * s.a12)
		}
	}
	if a8 := int32(s.a8 & 0xff); a8 != 0 {
		v = (a8 + 100) * v / 100
	}
	v = min(max(v, 2), 100) // 0x5179d0(&v, 2, 100)
	res := op((*big.Float).Mul, ext(float64(v)), ext(clip))
	return op((*big.Float).Mul, res, ext(vauExeF32(0x5ed490))) // f32 0.01
}

// 0x50b830: NPC list *0x689f94 (count +4 signed, data +0xc), match [[e+6]+0x58].
func (m *vauMem) npcFind(key uint32) uint32 {
	n := int32(m.u32(vauNpcl + 4))
	for k := int32(0); k < n; k++ {
		e := m.u32(m.u32(vauNpcl+0xc) + 4*uint32(k))
		if m.u32(m.u32(e+6)+0x58) == key {
			return e
		}
	}
	return 0
}

// ftol of the clock (*0x6598bc)+0x30, low 32 bits.
func (m *vauMem) clockMs() uint32 {
	c := m.f32(vauClock + 0x30)
	t := math.Trunc(c)
	if t >= 1<<63 || t < -(1<<63) {
		panic("visibility awareness: _ftol int64 overflow (clock)")
	}
	return uint32(int64(t))
}

// Message cooldown: a non-zero stamp blocks for 0xc8 ms; passing restamps.
func (m *vauMem) cooldown(at uint32) bool {
	if t := m.u32(at); t != 0 && m.clockMs()-t < 0xc8 {
		return false
	}
	m.w32(at, m.clockMs())
	return true
}

// 0x518150(0, 0, -1, -1).
func (m *vauMem) pickMember() int32 {
	m.call("518150", 0, 0, 0xffffffff, 0xffffffff)
	if !m.real {
		return int32(m.ret("r518150"))
	}
	v := []int64{0, 0, -1, -1}
	for _, f := range []int{-1, 0xb11, 0xb01} {
		for i := uint32(0); i < 8; i++ {
			if f < 0 {
				v = append(v, int64(m.u8(vauPres+0x106*i)))
			} else {
				v = append(v, int64(m.u32(vauChars+0x1862*i+uint32(f))))
			}
		}
	}
	v = append(v, make([]int64, 8)...) // key75: classKey -1 never compares it
	return int32(randomPartyMemberNode(v, m.r)[0])
}

// 0x52e690(character, sound, 0, [0x5ed8c8], [0x5ed914]); a voice handle gets
// +0x30 = 0x5dc and +0x34 = GetTickCount.
func (m *vauMem) voice(slot int32, sound uint32) {
	m.call("52e690", vauChars+uint32(slot)*0x1862, sound, 0, vauExeU32(0x5ed8c8), vauExeU32(0x5ed914))
	h := m.ret("r52e690")
	if h == 0 {
		return
	}
	if h != vauVoice {
		panic("visibility awareness: voice handle outside fixture")
	}
	m.w32(h+0x30, 0x5dc)
	m.w32(h+0x34, vauTick)
}

func visibilityAwarenessUpdate(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(vauIn) {
		panic("visibility awareness: wrong input count")
	}
	m := &vauMem{b: map[uint32]byte{}, heap: vauHeap, r: r, real: real, in: in}
	var arg2, arg3 uint32
	for i, f := range vauIn {
		v := in[i]
		lo, hi := int64(0), int64(1)<<(8*f.size)-1
		if f.signed {
			lo, hi = -(int64(1) << 31), int64(1)<<31-1
		}
		if v < lo || v > hi {
			panic("visibility awareness: input " + f.name + " outside its field")
		}
		switch f.name {
		case "arg2":
			arg2 = uint32(v)
		case "arg3":
			arg3 = uint32(v)
		default:
			if f.size == 1 {
				m.w8(f.at, uint32(v))
			} else {
				m.w32(f.at, uint32(v))
			}
		}
	}
	// Fixture caps (counts of the fixture lists).
	if n := int32(m.u32(vauNpcl + 4)); n < 0 || n > 2 {
		panic("visibility awareness: NPC count outside fixture 0..2")
	}
	if m.u32(vauGl+8) > 3 {
		panic("visibility awareness: monster count outside fixture 0..3")
	}
	if c := int32(m.u32(vauLst + 8)); c < 0 || c > 3 {
		panic("visibility awareness: sight list count outside fixture 0..3")
	}
	if c := int32(m.u32(vauLst + 4)); c < 0 || c > 8 {
		panic("visibility awareness: sight list capacity outside fixture 0..8")
	}
	// Fixture pointers (case "setup").
	m.w32(vauGrp+0xc, vauSobj)
	m.w32(vauMon0+0xc, vauMo0)
	m.w32(vauMon1+0xc, vauMo1)
	m.w32(vauNpcl+0xc, vauNpcd)
	for k, p := range []uint32{vauNpc0, vauNpc1, vauNpc0} {
		m.w32(vauNpcd+4*uint32(k), p)
	}
	m.w32(vauNpc0+6, vauNsub0)
	m.w32(vauNpc1+6, vauNsub1)
	m.w32(vauGl, vauGdata)
	m.w32(vauGl+4, 4)
	for k, p := range []uint32{vauMon0, vauGrp, vauMon1, vauMon0} {
		m.w32(vauGdata+4*uint32(k), p)
	}
	m.w32(vauGrp+0x2b6, vauLst)
	m.w32(vauLst, vauLdata)
	for k := uint32(0); k < 8; k++ {
		p := uint32(vauEntD)
		if k < 3 {
			p = vauEnt0 + 0x40*k
		}
		m.w32(vauLdata+4*k, p)
	}

	m.update(arg2, arg3)

	out := make([]int64, len(vauOut))
	for i, f := range vauOut {
		if f.size == 1 {
			out[i] = int64(m.u8(f.at))
		} else {
			out[i] = int64(m.u32(f.at))
		}
	}
	return out
}

func (m *vauMem) update(arg2, arg3 uint32) {
	const g = vauGrp
	if m.u32(0x68ec78) != 7 || m.u8(g+0x14) == 0 || m.u32(g+0x2b) == 0 {
		return
	}
	clip := m.clip()
	tmpl := vauTemplate(m.u32(g + 8))
	mpos := m.vauPos(vauSobj)
	ext := monsterAttackTargetExt
	time := m.u32(0x686a48)
	if arg2&0xff == 0 {
		m.monsters(clip, mpos, time)
		return
	}
	party := [3]uint32{m.u32(vauView + 0x8c), m.u32(vauView + 0x90), m.u32(vauView + 0x94)}
	d := m.partyDistance()
	dist := vauF32(d)
	inRange := d.Cmp(ext(clip)) <= 0
	m.w8(g+0x24d, vauB(inRange))
	m.w8(g+0x353, 0)
	m.w8(g+0x370, 0)
	if inRange {
		m.call("4c4810")
		m.w8(g+0x370, m.ret("r4810"))
		if m.u8(g+0x370) != 0 && m.u32(g+0x2b) != 0 && m.u32(g+0x107) < 0xf {
			// The group's sight of the party.
			light := uint32(0)
			if m.u8(0x683f94) == 0 {
				light = m.u8(0x68749b)
			}
			weakest := uint32(0x270f)
			for i := uint32(0); i < 8; i++ {
				c := vauChars + 0x1862*i
				if m.u8(vauPres+0x106*i) != 0 && m.u32(c+0xb11) != 0 && m.u32(c+0x89) < weakest {
					weakest = m.u32(c + 0x89)
				}
			}
			reach := m.sight(vauSight{a: vauF(mpos), b: vauF(party), facing: m.f32(vauSobj + 0xec),
				a7: m.u8(g + 0x24b), a8: m.amp(tmpl), a9: vauB(m.u32(g+0x87) != 0), a10: vauB(m.u8(tmpl+0xcb) == 0xc),
				a11: weakest, a12: light, a13: m.u8(g + 0x34c), dist: dist}, clip)
			m.w8(g+0x353, vauB(reach.Cmp(ext(dist)) >= 0))
		}
	}
	if m.u8(g+0x353) != 0 {
		m.w8(g+0x34c, 1)
		m.w32(g+0x354, time)
		for k := uint32(0); k < 3; k++ {
			m.w32(g+0x364+4*k, mpos[k])
			m.w32(g+0x358+4*k, party[k])
		}
	} else {
		m.w8(g+0x34c, m.fade(m.u32(g+0x354), time, 0xf0))
	}
	if m.u8(g+0x370) != 0 {
		m.call("4c4870")
		m.w8(g+0x34d, m.ret("w4870a"))
		m.w8(g+0x34f, m.ret("w4870b"))
		if m.u8(g+0x37a) != 0 {
			m.call("4c77f0") // case ret 1: the assertion at 0x504d1a is not reached
			m.w8(g+0x34e, vauB(m.ray(1, m.u32(g), 0xffffffff, 1, 1)&0xffff == 0))
		}
		if m.u8(g+0x37c) != 0 {
			m.call("4c78e0") // case ret 1: the assertion at 0x504d5c is not reached
			m.w8(g+0x350, vauB(m.ray(1, 0xfffffffd, 0xfffffffd, 1, 0)&0xffff != 1))
		}
	} else {
		m.w32(g+0x34d, 0)
	}

	// The party's notice of the group.
	m.w8(g+0x28d, 0)
	m.w8(g+0x2ab, 0)
	m.w8(g+0x2aa, 0)
	found := false
	if m.u8(g+0x24d) != 0 {
		if m.u8(g+0x28a) != 0 {
			arg3 = arg3&^0xff | 1
		}
		m.w8(g+0x2aa, arg3)
		m.call("4c4920", arg3)
		m.w8(g+0x2ab, m.ret("r4920"))
		if m.u8(g+0x2ab) != 0 {
			stealth := uint32(0)
			if m.u8(g+0x15) == 0 {
				stealth = m.u8(tmpl + 0x248)
			}
			pf := m.f32(vauView + 4)
			for i := uint32(0); i < 8 && !found; i++ {
				c := vauChars + 0x1862*i
				if m.u8(vauPres+0x106*i) == 0 || m.u32(c+0xb11) == 0 || m.u32(c+0xb01) >= 0xf {
					continue
				}
				reach := m.sight(vauSight{a: vauF(party), b: vauF(mpos), facing: pf,
					a7: m.u32(c + 0x161), a8: m.u8(c + 0x3dd), a9: vauB(m.u32(c+0xa31) != 0), a10: vauB(m.u32(c+0x69) == 6),
					a11: m.u8(tmpl + 0x24f), a12: stealth, a13: m.u8(g + 0x28a), a14: m.u8(0x68749d), dist: dist}, clip)
				if m.u8(0x6874ff) != 0 {
					reach = monsterAttackTargetOp((*big.Float).Mul, reach, ext(vauExeF32(0x5ed7f8))) // f32 0.6667
				}
				found = ext(dist).Cmp(reach) <= 0
			}
			m.w8(g+0x28d, vauB(found))
		}
	}
	if found {
		m.noticed(tmpl, time)
		for k := uint32(0); k < 3; k++ {
			m.w32(g+0x29e+4*k, party[k])
			m.w32(g+0x292+4*k, mpos[k])
		}
	} else {
		m.unnoticed(tmpl, time)
	}
	if m.u8(g+0x2ab) != 0 {
		m.call("4c4a20", 0, 0, 0)
		m.w8(g+0x28b, m.ret("w4a20a"))
		m.w8(g+0x28c, m.ret("w4a20b"))
	} else {
		m.w8(g+0x28b, 0)
		m.w8(g+0x28c, 0)
	}
}

// fade: a zero stamp gives 0; else 0 once more than lim has passed, 2 before.
func (m *vauMem) fade(stamp, time, lim uint32) uint32 {
	if stamp == 0 || time-stamp > lim {
		return 0
	}
	return 2
}

// ray: 0x434f20 address stub, arguments 3..7 recorded, k-th result from the case.
func (m *vauMem) ray(a3, a4, a5, a6, a7 uint32) uint32 {
	k := m.u32(vauStubs["434f20"].count)
	m.call("434f20", 0, 0, a3, a4, a5, a6, a7)
	return m.ret(fmt.Sprintf("r434f20_%d", k))
}

// 0x504ffb..0x5052e9: the party noticed the group.
func (m *vauMem) noticed(tmpl, time uint32) {
	const g = vauGrp
	if m.f32(0x60a390) == vauExeF32(0x5ec254) { // f32 12.0
		m.w8(uint32(int32(vauTable)+14*int32(int8(m.u8(tmpl+0x25f)))), 1) // 0x536110(id, 1)
		if m.u8(tmpl+0xd0)&1 != 0 {
			e := m.npcFind(m.u8(tmpl + 0xcd))
			if e == 0 {
				panic("visibility awareness: NPC record assertion (0x505055)")
			}
			if m.u8(e+0x2d) == 0 && m.u8(m.u32(e+6)+0x54) == 0 {
				m.call("539870", g)
				if m.ret("r539870")&0xff != 0 {
					// Object +0x118 to the party shape [0x659c0c]+0x100, no radii.
					d := monsterAttackTargetDistance([3]float64{m.f32(vauSobj + 0x118), m.f32(vauSobj + 0x11c), m.f32(vauSobj + 0x120)},
						[3]float64{m.f32(vauPsh + 0x100), m.f32(vauPsh + 0x104), m.f32(vauPsh + 0x108)}, false, 0, 0)
					if d.Cmp(monsterAttackTargetExt(vauExeF64(0x5ec3a0))) < 0 { // f64 25000
						m.w8(e+0x2d, 1)
						if m.u8(0x683f97) == 0 && m.u32(0x686a70) != 4 {
							if s := m.pickMember(); s != -1 {
								sound := vauExeU32(0x5ee620)                     // 38
								if m.u8(e+0x2e) == 0x18 && m.u8(0x689e66) != 0 { // 0x506280(0x2ee): fact byte 0x689b78+0x2ee
									sound = vauExeU32(0x5ee66c) // 57
								}
								m.voice(s, sound)
							}
						}
					}
				}
			}
		} else if t := m.u32(g + 0x28e); t == 0 || time-t > 0x78 {
			m.call("539870", g)
			if b := m.u8(g + 0x16); m.ret("r539870")&0xff == 0 && b <= 1 && m.cooldown(0x689b6c) {
				if s := m.pickMember(); s != -1 {
					sound := vauExeU32(0x5ee698) // 68
					if m.r.R(2) != 0 {           // 0x40efa0(2)
						sound = vauExeU32(0x5ee694) // 67
					}
					m.voice(s, sound)
				}
			}
		}
	}
	if m.u8(g+0x28a) != 1 && m.u8(tmpl+0x248) != 0 {
		m.call("4c4f80", math.Float32bits(5))
		if m.u8(0x683fc5) == 0 && m.cooldown(0x689b70) {
			m.call("58ac00", 8, vauText, 0xffffffff, 0xffffffff, 0)
		}
	}
	m.w8(g+0x28a, 1)
	m.w32(g+0x28e, time)
}

// 0x505335..0x50542a: the party did not notice the group.
func (m *vauMem) unnoticed(tmpl, time uint32) {
	const g = vauGrp
	if m.u8(g+0x28a) == 1 && m.u8(tmpl+0x248) != 0 {
		m.call("4c5150", math.Float32bits(5))
		if m.u8(0x683fc5) == 0 && m.cooldown(0x689b74) {
			m.call("58ac00", 8, vauText+4, 0xffffffff, 0xffffffff, 0)
		}
	}
	m.w8(g+0x28a, m.fade(m.u32(g+0x28e), time, 0x78))
	if m.u8(tmpl+0xd0)&1 != 0 {
		e := m.npcFind(m.u8(tmpl + 0xcd))
		if e == 0 {
			panic("visibility awareness: NPC record assertion (0x5053fa)")
		}
		if m.u8(e+0xe9) != 0 {
			m.call("50cf70", e, 0)
		}
	}
}

// 0x50546b..0x50574d: refresh the group's sight list against every other monster group.
func (m *vauMem) monsters(clip float64, mpos [3]uint32, time uint32) {
	const g = vauGrp
	if m.u8(0x683f94) == 0 {
		return
	}
	ext := monsterAttackTargetExt
	for i := uint32(0); i < m.u32(vauGl+8); i++ {
		o := m.u32(m.u32(vauGl) + 4*i)
		if o == g || m.u8(o+0x15) == 0 || m.u8(o+0x14) == 0 || m.u32(o+0x2b) == 0 {
			continue
		}
		list := m.u32(g + 0x2b6)
		e := uint32(0)
		for k := int32(0); k < int32(m.u32(list+8)); k++ {
			x := m.u32(m.u32(list) + 4*uint32(k))
			if x == 0 {
				return
			}
			if m.u32(x) == m.u32(o) {
				e = x
				break
			}
		}
		if e == 0 {
			e = m.malloc(0x31)
			m.w32(e, m.u32(o))
			// 0x5e2480: grow by 5 when full, then append.
			n, c := int32(m.u32(list+8)), int32(m.u32(list+4))
			if n >= c {
				data := m.malloc(uint32(c*4 + 0x14))
				for j := int32(0); j < n; j++ {
					m.w32(data+4*uint32(j), m.u32(m.u32(list)+4*uint32(j)))
				}
				m.w32(list, data)
				m.w32(list+4, uint32(c+5))
			}
			m.w32(m.u32(list)+4*uint32(n), e)
			m.w32(list+8, uint32(n+1))
		}
		mo := m.u32(o + 0xc)
		opos := m.vauPos(mo)
		m.w8(e+0xb, 0)
		m.w8(e+0x28, 0)
		d := m.objectDistance(vauSobj, mo)
		seen := false
		if d.Cmp(ext(clip)) <= 0 {
			k := m.u32(vauStubs["4c4af0"].count)
			m.call("4c4af0", mo)
			m.w8(e+0x28, m.ret(fmt.Sprintf("r4af0_%d", k)))
			if m.u8(e+0x28) != 0 {
				seen = m.sees(o, e, mpos, opos, vauF32(d))
				m.w8(e+0xb, vauB(seen))
			}
		}
		if seen {
			m.w8(e+4, 1)
			m.w32(e+0xc, time)
			for k := uint32(0); k < 3; k++ {
				m.w32(e+0x1c+4*k, mpos[k])
				m.w32(e+0x10+4*k, opos[k])
			}
		} else {
			m.w8(e+4, m.fade(m.u32(e+0xc), time, 0xf0))
		}
		if m.u8(e+0x28) == 0 {
			m.w32(e+5, 0)
			continue
		}
		m.call("4c4b70", mo, e+5, e+7)
		m.w8(e+5, m.ret("w4b70a"))
		m.w8(e+7, m.ret("w4b70b"))
		if m.u8(g+0x37a) != 0 {
			m.call("4c77f0") // case ret 1: the assertion at 0x50565d is not reached
			m.w8(e+6, vauB(m.ray(1, m.u32(g), m.u32(o), 1, 1)&0xffff == 0))
		}
		if m.u8(g+0x37c) != 0 {
			m.call("4c78e0") // case ret 1: the assertion at 0x50569e is not reached
			m.w8(e+8, vauB(m.ray(1, 0xfffffffd, 0xfffffffd, 1, 0)&0xffff != 1))
		}
	}
}

// 0x5058a0(group, other, entry): the group's sight of another monster group.
func (m *vauMem) sees(o, e uint32, apos, bpos [3]uint32, dist float64) bool {
	const g = vauGrp
	if m.u32(g+0x2b) == 0 || m.u32(g+0x107) >= 0xf {
		return false
	}
	tb := vauTemplate(m.u32(o + 8))
	ta := vauTemplate(m.u32(g + 8))
	stealth := uint32(0)
	if m.u8(o+0x15) == 0 {
		stealth = m.u8(tb + 0x248)
	}
	clip := m.clip()
	reach := m.sight(vauSight{a: vauF(apos), b: vauF(bpos), facing: m.f32(vauSobj + 0xec),
		a7: m.u8(g + 0x24b), a8: m.amp(ta), a9: vauB(m.u32(g+0x87) != 0), a10: vauB(m.u8(ta+0xcb) == 0xc),
		a11: m.u8(tb + 0x24f), a12: stealth, a13: m.u8(e + 4), dist: dist}, clip)
	return reach.Cmp(monsterAttackTargetExt(dist)) >= 0
}
