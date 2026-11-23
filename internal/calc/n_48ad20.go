package calc

import "fmt"

// encounterGenerateNode is 0x48ad20 encounter_generate(pos), thiscall on the
// encounter generator gen (ecx): pick an encounter table row, spawn the leader
// group and up to two companion groups, register the leader as active.
//
// Flow (exe addresses):
//   - gen u32 +0 bit0 set, or i32 gen+0x1c == -1: return 0 (0x48ad6c, 0x48ad7c).
//   - table T = [0x65ba2c][idx] when idx < i32 [0x65ba24] (signed), else
//     [0x65ba2c][0] (0x48ad85).
//   - 0x48b9a0(T, &list) (thiscall, callee pops 8) fills the result list
//     (header on the stack, data from new(0x14) at 0x48ad4d, capacity 5);
//     0 returns 0. sel = i32 list data[0].
//   - species = u16 [T+0xc][sel] if sel < i32 [T+4], else element 0; msf =
//     [T+0x4c][sel] if sel < i32 [T+0x44], else element 0 (0x48adc3..).
//   - L = 0x4e57c0(species) (template cache 0x6840c7+4*species); L+0x267
//     nonzero: 0x5182e0(1, 0x60a890, species), return 0. 0x4e5b50(species) is
//     called; its return only feeds a dead accumulator (0x48ae7c).
//   - companions k=0,1, entry e = L+0xc5+3k (i16 species, u8 chance at +2):
//     i16 > 0 and 0x40efe0(chance) succeeds -> slot k = e, 0x4e5b50(species);
//     else slot k = 0.
//   - gen bit5: 0x535ad0(u8 L+0x25f) nonzero (attitude record byte
//     i8 [0x68d6e8+14f] >= 0x22) returns 0; then story facts 0x30, 0x22, 0x31
//     (0x506280 -> byte 0x689b78+fact, debug log 0x688290 off) all zero
//     return 0.
//   - n = 0x48bc30(L) (thiscall ecx=gen, callee pops 4); grp = 0x50f1a0(species,
//     n, &pos, 0, 0, 1); grp+0xc3 = 1; 0x50ff40(grp, gen+0xc).
//   - active list 0x65ba10 (count 0x65ba14, capacity 0x65ba18, data 0x65ba1c):
//     grp not found in data[0..count) is appended, growing first through 0x4addf0
//     (new(4*(count+1)), copy, capacity = count+1) when count+1 > capacity.
//   - v = 0x4e5950(u32 grp+0x9f) (monster record by id: normal list 0x683fad,
//     then special list 0x683fb9; returns record+0xc's word, 0 if id -1 or not
//     found); v nonzero: 0x4c7f10 (ecx=v) (msf if msf and msf[0] != 0, else
//     0x60a848; 1).
//   - per nonzero slot: T' = 0x4e57c0(i16 species); T'+0x267 nonzero:
//     0x5182e0(1, 0x60a7c0, species, leaderSpecies), return 0. n' =
//     0x517970(T'+0xc1), ebx += n'; g = 0x50f1a0(species, n', &pos, 0, 0, 1);
//     g+0xc3 = 1; 0x50ff40(g, gen+0xc); 0x50fc20(grp, g).
//   - log when u8 [0x689b32] != 0, u8 [0x6505a8] == 0, u32 [0x68ec78] != 4:
//     0x58aad0(7, 0x60a768, gen+0x24, u32 grp+4, a, ebx, ebx==1 ? 0x60a83c :
//     0x60a830, level) where a = 0x510180(grp) = 0x4e57c0(u32 grp+0x18), +0x30
//     when u32 grp+4 != 1, and level = u8 [T+0x3c][sel] (element 0 when sel >=
//     i32 [T+0x34]). Return 1.
//
// Emu (what TestEmu replays) is re/emu/cases/0048ad20.json: the D4 formula
// callees 0x48b9a0, 0x48bc30, 0x517970, 0x50f1a0 and the excluded 0x50ff40,
// 0x50fc20, 0x4c7f10, 0x5182e0, 0x58aad0 are address stubs; the case outputs
// each call count and the recorded arguments (stack residue arguments are not
// recorded: the leader 0x5182e0 call has no 4th argument). Stub returns are
// case inputs: selectCount (0x48b9a0, which also stores the selbox address as
// the list data, so sel is an input), groupSize (0x48bc30), chumCount0/1
// (0x517970 calls in order), groupRet0..2 (0x50f1a0 calls in order). The
// spawned groups' +4, +0x18 and +0x9f are case inputs (the stub writes nothing).
//
// F runs the real callees: encounterTableSelectNode (capacity 5; sel = first
// result index), encounterGroupSizeNode, diceNode and monsterGroupSpawnNode,
// whose memberCount, lastMonsterId and next ids are written back (group +4,
// +0x9f, globals 0x6874ba / 0x6874be; +0x18 = the species, the node's
// templateId input is 0 because its case only filled template slot 0). F reads
// the F-only inputs (table weight/time arrays, template dice and spawn fields,
// party, difficulty, counters). F's address model is the Emu's: group k at the
// fixture's G_k, and the active-list grow buffer at 0x50000018.
//
// Gaps (known, F only unless noted):
//   - 0x40efe0 is modelled as R(100) < chance (Emu and F); the emulator's
//     intercept records [100, v], the original's own bucketing of rand() is not
//     reproduced here.
//   - the real 0x50f1a0 inserts its monsters into the list 0x683fad that
//     0x4e5950 searches; the fixture lists are not updated.
//   - 0x48b9a0's own grow allocations are not placed in the heap model, so a
//     table select growing past 5 results would move the active-list buffer.
//   - pos and the party position are constants (geometry is out of scope,
//     MG 2026-10-11); 0x50ff40 / 0x50fc20 (placement) are excluded.
//
// Fixture caps (calc panics past them): a table index outside 0..2 when it is
// read (tableCount <= 3), records/lengths <= 8, sel within 0..7 when indexed,
// template species 0..2 (others need the database file: emulator limitation,
// and 0x4e57c0 asserts at >= 1000), group pointers G0..G2, active count 0..4,
// list pointers 0 or the fixture lists, list counts <= 2. Asserts (cov-exempt
// 48ae1f, 48b06e: template NULL; 48af4e: group size 0) panic; so do 0x535ad0's
// assert (faction >= 21) and 0x510180 on a missing template.
const (
	egArena  = 0x20000000
	egGen    = egArena
	egPos    = egArena + 0x100
	egTArr   = egArena + 0x200
	egTab    = egArena + 0x300 // + 0x100*t
	egSpc    = egArena + 0x600 // + 0x80*t: species u16 x8, level +0x20, msf pointers +0x40
	egWgt    = egArena + 0x780 // + 0x40*t: weight x8, time +0x20
	egStr    = egArena + 0x900 // + 0x100*t + 0x10*k
	egSelBox = egArena + 0xc00
	egTpl    = egArena + 0x1000 // + 0x400*s
	egGrp    = egArena + 0x2000 // + 0x200*k
	egAct    = egArena + 0x2800
	egNL     = egArena + 0x2900
	egSL     = egArena + 0x2a00
	egOpts   = egArena + 0x2b00
	egStub   = egArena + 0x3000
	egParty  = egArena + 0x4000
	egFOnly  = egArena + 0x4100
	egHeap   = 0x50000018 // emu bump heap after new(0x14) at 0x50000000
	egCache  = 0x6840c7   // template cache, 4*species
	egAtt    = 0x68d6e8   // attitude records, 14 bytes
	egFacts  = 0x689b78   // story fact bytes
	egActCnt = 0x65ba14
	egActCap = 0x65ba18
	egActDat = 0x65ba1c
)

// call-count slots in the stub object, case order
var egCnt = []int64{0x48b9a0, 0x48bc30, 0x517970, 0x50f1a0, 0x50ff40, 0x50fc20, 0x4c7f10, 0x5182e0, 0x58aad0}

type egField struct {
	name string
	size int
	at   int64
}

var egSpec = func() (f []egField) {
	add := func(n string, size int, at int64) { f = append(f, egField{n, size, at}) }
	add("genFlags", 4, egGen)
	add("tableIdx", 4, egGen+0x1c)
	add("tableCount", 4, 0x65ba24)
	add("selectCount", 4, egStub+0x40)
	add("sel", 4, egSelBox)
	add("recordCount", 4, egTab+4)
	add("msfLen", 4, egTab+0x44)
	add("levelLen", 4, egTab+0x34)
	for k := int64(0); k < 8; k++ {
		add(fmt.Sprint("species", k), 2, egSpc+2*k)
	}
	for k := int64(0); k < 8; k++ {
		add(fmt.Sprint("level", k), 1, egSpc+0x20+k)
	}
	for k := int64(0); k < 8; k++ {
		add(fmt.Sprint("msfChar", k), 1, egStr+0x10*k)
	}
	for s := int64(0); s < 3; s++ {
		t := egTpl + 0x400*s
		add(fmt.Sprint("deleted", s), 1, t+0x267)
		add(fmt.Sprint("faction", s), 1, t+0x25f)
		add(fmt.Sprint("chum0Species", s), 2, t+0xc5)
		add(fmt.Sprint("chum0Chance", s), 1, t+0xc7)
		add(fmt.Sprint("chum1Species", s), 2, t+0xc8)
		add(fmt.Sprint("chum1Chance", s), 1, t+0xca)
	}
	for i := int64(0); i < 21; i++ {
		add(fmt.Sprint("attitude", i), 1, egAtt+14*i)
	}
	for _, e := range []int64{0x30, 0x22, 0x31} {
		add(fmt.Sprintf("fact%x", e), 1, egFacts+e)
	}
	add("groupSize", 4, egStub+0x44)
	for k := int64(0); k < 2; k++ {
		add(fmt.Sprint("chumCount", k), 4, egStub+0x48+4*k)
	}
	for k := int64(0); k < 3; k++ {
		add(fmt.Sprint("groupRet", k), 4, egStub+0x50+4*k)
	}
	for k := int64(0); k < 3; k++ {
		g := egGrp + 0x200*k
		add(fmt.Sprint("groupCount", k), 4, g+4)
		add(fmt.Sprint("groupSpecies", k), 4, g+0x18)
		add(fmt.Sprint("groupMonsterId", k), 4, g+0x9f)
	}
	add("activeCount", 4, egActCnt)
	add("activeCap", 4, egActCap)
	for k := int64(0); k < 5; k++ {
		add(fmt.Sprint("active", k), 4, egAct+4*k)
	}
	for _, l := range []struct {
		n      string
		g, obj int64
	}{{"nl", 0x683fad, egNL}, {"sl", 0x683fb9, egSL}} {
		add(l.n+"Ptr", 4, l.g)
		add(l.n+"Count", 4, l.obj+8)
		for k := int64(0); k < 2; k++ {
			add(fmt.Sprintf("%sId%d", l.n, k), 4, l.obj+0x40+0x20*k)
			add(fmt.Sprintf("%sMonster%d", l.n, k), 4, l.obj+0x40+0x20*k+0xc)
		}
	}
	add("reportEnabled", 1, 0x689b32)
	add("logSuppressed", 1, 0x6505a8)
	add("mode", 4, 0x68ec78)
	// F-only: read by the real callees, never by the stubbed emulator run.
	add("weightLen", 4, egTab+0x14)
	add("timeLen", 4, egTab+0x24)
	for k := int64(0); k < 8; k++ {
		add(fmt.Sprint("weight", k), 1, egWgt+k)
	}
	for k := int64(0); k < 8; k++ {
		add(fmt.Sprint("time", k), 1, egWgt+0x20+k)
	}
	for s := int64(0); s < 3; s++ {
		t := egTpl + 0x400*s
		for _, x := range []struct {
			n    string
			off  int64
			size int
		}{{"monsterLevel", 0x251, 1}, {"sizeZ", 0xc1, 2}, {"sizeN", 0xc3, 1}, {"sizeS", 0xc4, 1},
			{"hpBase", 0xd6, 2}, {"hpDice", 0xd8, 1}, {"hpSides", 0xd9, 1},
			{"spBase", 0xda, 2}, {"spDice", 0xdc, 1}, {"spSides", 0xdd, 1},
			{"special", 0x26a, 1}, {"templateFlags", 0xd0, 1}, {"nameId", 0x187, 2}} {
			add(fmt.Sprint(x.n, s), x.size, t+x.off)
		}
	}
	add("gameTime", 4, 0x6874f7)
	for k := int64(0); k < 6; k++ {
		add(fmt.Sprintf("slot%dActive", k+2), 1, egParty+8*k)
		add(fmt.Sprintf("slot%dTotalLevel", k+2), 4, egParty+8*k+4)
	}
	add("difficulty", 4, egFOnly)
	add("nextMonsterId", 4, 0x6874be)
	add("nextGroupId", 4, 0x6874ba)
	add("worldTime", 4, 0x686a48)
	add("battleFlags", 4, egFOnly+0xc)
	add("animationFrames", 1, egFOnly+4)
	add("unbornCount", 4, egFOnly+8)
	return
}()

var egOuts = func() (o []egField) {
	add := func(n string, size int, at int64) { o = append(o, egField{n, size, at}) }
	for i, a := range egCnt {
		add(fmt.Sprintf("calls%x", a), 4, egStub+4*int64(i))
	}
	add("tableArg", 4, egStub+0x100)
	add("sizeTemplateArg", 4, egStub+0x110)
	for k := int64(0); k < 2; k++ {
		add(fmt.Sprint("diceArg", k), 4, egStub+0x120+4*k)
	}
	for _, x := range []struct {
		n    string
		at   int64
		size int64
	}{{"spawnSpecies", 0x140, 3}, {"spawnCount", 0x160, 3}, {"placeGroup", 0x180, 3}, {"placeTarget", 0x1a0, 3},
		{"attachLeader", 0x1c0, 2}, {"attachChum", 0x1e0, 2}} {
		for k := int64(0); k < x.size; k++ {
			add(fmt.Sprint(x.n, k), 4, egStub+x.at+4*k)
		}
	}
	add("soundName", 4, egStub+0x200)
	add("soundFlag", 4, egStub+0x204)
	add("deletedFormat", 4, egStub+0x210)
	add("deletedSpecies", 4, egStub+0x214)
	for a := int64(3); a <= 8; a++ {
		add(fmt.Sprint("logArg", a), 4, egStub+0x220+4*(a-3))
	}
	for k := int64(0); k < 3; k++ {
		add(fmt.Sprint("groupActive", k), 1, egGrp+0x200*k+0xc3)
	}
	add("activeCountAfter", 4, egActCnt)
	add("activeCapAfter", 4, egActCap)
	add("activeDataAfter", 4, egActDat)
	for k := int64(0); k < 5; k++ {
		add(fmt.Sprint("activeSlot", k), 4, egAct+4*k)
	}
	for k := int64(0); k < 5; k++ {
		add(fmt.Sprint("grownSlot", k), 4, egHeap+4*k)
	}
	return
}()

var egIn, egOutNames = func() (in, out []string) {
	for _, f := range egSpec {
		in = append(in, f.name)
	}
	out = []string{"ret"}
	for _, o := range egOuts {
		out = append(out, o.name)
	}
	return
}()

func init() {
	Nodes["0048ad20"] = Node{Addr: "0048ad20", In: egIn, Out: egOutNames, F: encounterGenerateNode,
		Emu: func(in []int64, r Rand) []int64 { return egNode(in, r, false) }}
}

func encounterGenerateNode(in []int64, r Rand) []int64 { return egNode(in, r, true) }

func egNode(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(egSpec) {
		panic("encounter_generate: wrong input count")
	}
	w := &cgW{m: map[int64]byte{}}
	egSetup(w)
	for i, f := range egSpec {
		w.wr(f.at, f.size, in[i])
	}
	e := &egRun{w: w, r: r, real: real}
	out := []int64{e.run()}
	for _, o := range egOuts {
		out = append(out, w.rd(o.at, o.size))
	}
	return out
}

// egSetup is the case's setup: pointers, the two constant tables, decoys.
func egSetup(w *cgW) {
	w.p8(0x688290, 0)
	w.p32(0x65bdf4, 0)
	w.p32(0x65a118, egOpts)
	w.p8(egOpts+0x10, 1)
	w.p32(0x65ba2c, egTArr)
	w.p32(egActDat, egAct)
	for _, k := range []int64{1, 2, 3, 4, 5, 6, 8, 9, 10} {
		w.p32(egGen+4*k, 0x47454e00+k)
	}
	for i, v := range []int64{0x11, 0x22, 0x33} {
		w.p32(egPos+4*int64(i), v)
	}
	for t := int64(0); t < 3; t++ {
		tb, sp, wg, st := egTab+0x100*t, egSpc+0x80*t, egWgt+0x40*t, egStr+0x100*t
		w.p32(egTArr+4*t, tb)
		w.p32(tb+0xc, sp)
		w.p32(tb+0x3c, sp+0x20)
		w.p32(tb+0x4c, sp+0x40)
		w.p32(tb+0x1c, wg)
		w.p32(tb+0x2c, wg+0x20)
		for k := int64(0); k < 8; k++ {
			p := st + 0x10*k
			if k == 3 {
				p = 0
			}
			w.p32(sp+0x40+4*k, p)
			w.p32(st+0x10*k+1, 0x46534d2e) // ".MSF"
			w.p8(st+0x10*k+5, 0)
		}
	}
	for t, c := range map[int64]struct {
		rc, lev, msf, wl, tl int64
		species              [8]int64
	}{1: {3, 3, 3, 3, 3, [8]int64{1, 2, 0, 2, 1, 0, 2, 1}}, 2: {8, 6, 8, 8, 5, [8]int64{2, 0, 1, 1, 2, 0, 0, 2}}} {
		tb, sp, wg, st := egTab+0x100*t, egSpc+0x80*t, egWgt+0x40*t, egStr+0x100*t
		w.p32(tb+4, c.rc)
		w.p32(tb+0x34, c.lev)
		w.p32(tb+0x44, c.msf)
		w.p32(tb+0x14, c.wl)
		w.p32(tb+0x24, c.tl)
		for k := int64(0); k < 8; k++ {
			w.wr(sp+2*k, 2, c.species[k])
			w.p8(sp+0x20+k, 10*t+k+1)
			w.p8(wg+k, []int64{3, 7, 20, 70}[(k+t)%4])
			w.p8(wg+0x20+k, (k+t)%3)
			ch := int64(0)
			if (k+t)%2 != 0 {
				ch = 0x41
			}
			w.p8(st+0x10*k, ch)
		}
	}
	for s := int64(0); s < 3; s++ {
		w.p32(egCache+4*s, egTpl+0x400*s)
		w.p8(egGrp+0x200*s+0xc3, 0x5a)
	}
	for f := int64(0); f < 21; f++ {
		w.p8(egAtt+14*f+1, 0x40)
	}
	for _, e := range []int64{0x21, 0x23, 0x2f, 0x32} {
		w.p8(egFacts+e, 1)
	}
	for k := int64(5); k < 8; k++ {
		w.p32(egAct+4*k, 0xdead0000+k)
	}
	for _, l := range []int64{egNL, egSL} {
		w.p32(l, l+0x10)
		w.p32(l+4, 4)
		for k := int64(0); k < 4; k++ {
			v := 0xdead0100 + k
			if k < 2 {
				v = l + 0x40 + 0x20*k
			}
			w.p32(l+0x10+4*k, v)
		}
		for k := int64(0); k < 2; k++ {
			w.p32(l+0x40+0x20*k+4, 0xc0de0000+k)
			w.p32(l+0x40+0x20*k+8, 0xc0de0010+k)
		}
	}
}

type egRun struct {
	w    *cgW
	r    Rand
	real bool
	n    map[int64]int64 // stub call counts by callee address
}

// call counts a stubbed call and records its arguments at the stub's record
// slots (each slot stride 4 per call).
func (e *egRun) call(callee int64, rec map[int64]int64) int64 {
	if e.n == nil {
		e.n = map[int64]int64{}
	}
	k := e.n[callee]
	for at, v := range rec {
		e.w.p32(egStub+at+4*k, v)
	}
	e.n[callee] = k + 1
	for i, a := range egCnt {
		if a == callee {
			e.w.p32(egStub+4*int64(i), k+1)
		}
	}
	return k
}

// template is 0x4e57c0 / 0x4e5b50's cache lookup.
func (e *egRun) template(sp int64) int64 {
	if sp < 0 || sp > 2 {
		panic("encounter_generate: template species outside the fixture cache 0..2 (database file load / assert >= 1000)")
	}
	return e.w.u32(egCache + 4*sp)
}

func (e *egRun) group(g int64) int64 {
	if g != egGrp && g != egGrp+0x200 && g != egGrp+0x400 {
		panic("encounter_generate: group pointer is not a fixture group")
	}
	return g
}

func (e *egRun) party() []int64 {
	p := make([]int64, 0, 12)
	for k := int64(0); k < 6; k++ {
		p = append(p, e.w.u8(egParty+8*k), e.w.u32(egParty+8*k+4))
	}
	return p
}

// spawn is 0x50f1a0(species, n, &pos, 0, 0, 1).
func (e *egRun) spawn(sp, n int64) int64 {
	k := e.call(0x50f1a0, map[int64]int64{0x140: sp, 0x160: n})
	if !e.real {
		return e.group(e.w.u32(egStub + 0x50 + 4*k))
	}
	w, t, g := e.w, e.template(sp), egGrp+0x200*k
	out := monsterGroupSpawnNode([]int64{
		w.rs(t+0xd6, 2), w.u8(t + 0xd8), w.u8(t + 0xd9), w.rs(t+0xda, 2), w.u8(t + 0xdc), w.u8(t + 0xdd),
		w.u8(t + 0x26a), w.u32(0x6874be), w.u32(egPos), w.u32(egPos + 4), w.u32(egPos + 8), 0, 0, 0,
		w.i32(0x68ec78), w.u32(egFOnly + 0xc), n & 0xffffffff, 0, 0, 1, w.u8(t + 0xd0), w.u32(0x6874ba),
		w.u32(0x686a48), w.u8(0x689b32), 0, w.rd(t+0x187, 2), w.u8(egFOnly + 4), w.u32(egFOnly + 8),
	}, e.r)
	w.p32(g+4, out[2])
	w.p32(g+0x18, sp)
	w.p32(g+0x9f, out[19])
	w.p32(0x6874ba, out[21])
	w.p32(0x6874be, out[22])
	return g
}

// find is 0x4e5950(id): word +0xc of the monster record with that id.
func (e *egRun) find(id int64) int64 {
	w := e.w
	if id == 0xffffffff {
		return 0
	}
	for _, l := range []struct{ g, obj int64 }{{0x683fad, egNL}, {0x683fb9, egSL}} {
		p := w.u32(l.g)
		if p == 0 {
			continue
		}
		if p != l.obj || w.u32(p+8) > 2 {
			panic("encounter_generate: monster list is not the fixture list or count past 2")
		}
		for i := int64(0); i < w.u32(p+8); i++ {
			if r := w.u32(w.u32(p) + 4*i); w.u32(r) == id {
				return w.u32(r + 0xc)
			}
		}
	}
	return 0
}

func (e *egRun) run() int64 {
	w := e.w
	gen := int64(egGen)
	if w.u32(gen)&1 != 0 || w.i32(gen+0x1c) == -1 {
		return 0
	}
	idx, cnt := w.i32(gen+0x1c), w.i32(0x65ba24)
	if idx >= cnt {
		idx = 0
	}
	if idx < 0 || idx > 2 {
		panic("encounter_generate: table index outside the fixture's 3 tables")
	}
	T := w.u32(w.u32(0x65ba2c) + 4*idx)
	for _, off := range []int64{4, 0x14, 0x24, 0x34, 0x44} {
		if w.i32(T+off) > 8 {
			panic("encounter_generate: table record count or length past the fixture's 8")
		}
	}
	e.call(0x48b9a0, map[int64]int64{0x100: T})
	var sel int64
	if e.real {
		in := append([]int64{w.u32(0x6874f7)}, e.party()...)
		in = append(in, w.i32(T+4), w.i32(T+0x34), w.i32(T+0x14), w.i32(T+0x24))
		for _, a := range []int64{0x3c, 0x1c, 0x2c} {
			for k := int64(0); k < encounterTableSelectMaxRecords; k++ {
				in = append(in, w.u8(w.u32(T+a)+k))
			}
		}
		out := encounterTableSelectNode(append(in, 5), e.r)
		if out[3] == 0 {
			return 0
		}
		if p := out[2]; p == encounterTableSelectListBase {
			sel = out[4]
		} else {
			sel = out[20+(p-encounterTableSelectHeapBase)/4]
		}
	} else {
		if w.u32(egStub+0x40) == 0 {
			return 0
		}
		sel = w.i32(egSelBox)
	}
	at := func(lenOff, arrOff, stride int64) int64 {
		base := w.u32(T + arrOff)
		if sel < w.i32(T+lenOff) {
			if sel < 0 || sel > 7 {
				panic("encounter_generate: selection index outside the fixture arrays")
			}
			return base + stride*sel
		}
		return base
	}
	species := w.rd(at(4, 0xc, 2), 2)
	msf := w.u32(at(0x44, 0x4c, 4))
	L := e.template(species)
	if w.u8(L+0x267) != 0 {
		e.call(0x5182e0, map[int64]int64{0x210: 0x60a890, 0x214: species})
		return 0
	}
	var slot [2]int64
	for k := int64(0); k < 2; k++ {
		ent := L + 0xc5 + 3*k
		if sp := w.rs(ent, 2); sp > 0 && int64(e.r.R(100)) < w.u8(ent+2) {
			e.template(sp)
			slot[k] = ent
		}
	}
	if w.u32(gen)&0x20 != 0 {
		f := w.u8(L + 0x25f)
		if f >= 0x15 {
			panic("encounter_generate: 0x535ad0 asserts on faction >= 21")
		}
		if w.rs(egAtt+14*f, 1) >= 0x22 {
			return 0
		}
		if w.u8(egFacts+0x30) == 0 && w.u8(egFacts+0x22) == 0 && w.u8(egFacts+0x31) == 0 {
			return 0
		}
	}
	e.call(0x48bc30, map[int64]int64{0x110: L})
	n := w.u32(egStub + 0x44)
	if e.real {
		n = encounterGroupSizeNode(append([]int64{w.u8(L + 0x251), w.rs(L+0xc1, 2), w.u8(L + 0xc3), w.u8(L + 0xc4),
			w.i32(egFOnly)}, e.party()...), e.r)[0] & 0xffffffff
	}
	if n == 0 {
		panic("encounter_generate: group size 0 asserts (0x48af4e)")
	}
	grp := e.spawn(species, n)
	w.p8(grp+0xc3, 1)
	e.call(0x50ff40, map[int64]int64{0x180: grp, 0x1a0: gen + 0xc})
	if c := w.i32(egActCnt); c < 0 || c > 4 {
		panic("encounter_generate: active list count outside the fixture's 0..4")
	} else {
		found := false
		for i := int64(0); i < c; i++ {
			found = found || w.u32(w.u32(egActDat)+4*i) == grp
		}
		if !found {
			if c+1 > w.i32(egActCap) { // 0x4addf0(count+1)
				old := w.u32(egActDat)
				for i := int64(0); i < c; i++ {
					w.p32(egHeap+4*i, w.u32(old+4*i))
				}
				w.p32(egActCap, c+1)
				w.p32(egActDat, egHeap)
			}
			w.p32(w.u32(egActDat)+4*c, grp)
			w.p32(egActCnt, c+1)
		}
	}
	if e.find(w.u32(grp+0x9f)) != 0 {
		name := int64(0x60a848)
		if msf != 0 && w.u8(msf) != 0 {
			name = msf
		}
		e.call(0x4c7f10, map[int64]int64{0x200: name, 0x204: 1})
	}
	var ebx int32
	for k := 0; k < 2; k++ {
		if slot[k] == 0 {
			continue
		}
		sp := w.rs(slot[k], 2)
		t := e.template(sp)
		if w.u8(t+0x267) != 0 {
			e.call(0x5182e0, map[int64]int64{0x210: 0x60a7c0, 0x214: sp})
			return 0
		}
		d := e.call(0x517970, map[int64]int64{0x120: t + 0xc1})
		cn := w.u32(egStub + 0x48 + 4*d)
		if e.real {
			cn = diceNode([]int64{w.rs(t+0xc1, 2), w.u8(t + 0xc3), w.u8(t + 0xc4)}, e.r)[0] & 0xffffffff
		}
		ebx += int32(cn)
		g := e.spawn(sp, cn)
		w.p8(g+0xc3, 1)
		e.call(0x50ff40, map[int64]int64{0x180: g, 0x1a0: gen + 0xc})
		e.call(0x50fc20, map[int64]int64{0x1c0: grp, 0x1e0: g})
	}
	if w.u8(0x689b32) != 0 && w.u8(0x6505a8) == 0 && w.u32(0x68ec78) != 4 {
		s := int64(0x60a830)
		if ebx == 1 {
			s = 0x60a83c
		}
		a := e.template(w.u32(grp + 0x18))
		if w.u32(grp+4) != 1 {
			a += 0x30
		}
		lv := w.u8(at(0x34, 0x3c, 1))
		e.call(0x58aad0, map[int64]int64{0x220: gen + 0x24, 0x224: w.u32(grp + 4), 0x228: a, 0x22c: int64(uint32(ebx)), 0x230: s, 0x234: lv})
	}
	return 1
}
