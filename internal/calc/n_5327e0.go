package calc

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Cycle group 0x533320 / 0x5327e0 / 0x534ef0 (monster spell target selection).
// The three members share one memory-image model: every input is written into a
// byte image at the address the emulator case writes it to (setup first, then the
// inputs in order, as emu.py does), and the node bodies below read that image
// through the same pointers the original follows. Immutable exe data (spell id
// tables, class vtables) is read from the player's own Wiz8.exe; nothing from it
// is embedded. The generated tables at the end of each member file are produced
// from the same description as the case JSON, so the two cannot drift.

const (
	mstArena      = 0x20000000
	mstArenaSize  = 0x100000
	mstStackBase  = 0x7f000000 // virtual stack argument slots; slot n at +4+4n
	mstCharStride = 0x1862
	mstSpellSize  = 0x1bf
	mstMaxChar    = 8
	mstSpellCap   = 256
	mstTemplates  = 2 // fixture cap: template ids 0..1 are preloaded in the cache
	mstMonsters   = 3 // fixture cap: monster ids 0..2 exist
)

// mstFld places one input (or reads one output). idx >= 0 names the input whose
// value indexes a record array (count records of stride bytes); a negative index
// value skips the write, an index past count is an error, as in emu.py.
type mstFld struct {
	addr   uint32
	size   uint8
	signed bool
	idx    int32
	count  uint32
	stride uint32
}

type mstSet struct {
	addr uint32
	size uint8
	val  uint32
}

// mstStubAt describes one address-stubbed callee: where the return value input
// sits, which stack arguments (1-based) are copied where, and the call counter.
type mstStubAt struct {
	ret uint32
	rec [][3]uint32 // arg, size, addr
	cnt uint32
}

type mstSpec struct {
	set  []mstSet
	in   []mstFld
	out  []mstFld
	stub []mstStubAt // power (0x500330), area (0x4fd030), log (0x5182e0), ui (0x4d9080), disp (0x546f10), rand (0x518150), penalty (0x51aec0)
	// areaChars / areaMons are the constant vectors the 0x4fd030 address stub
	// writes in the case (character indexes, monster ids).
	areaChars, areaMons []uint32
}

const (
	mstPower = iota
	mstArea
	mstLog
	mstUI
	mstDisp
	mstRand
	mstPenalty
)

const (
	mstHeap       = 0x50000000
	mstHeapWindow = 0x1000 // fixture cap: allocations stay inside the observed window
)

// alloc is 0x5e1ce0 on the emulator heap: 8-byte rounded, minimum 8, zero filled,
// never freed (0x5e1c10 does nothing observable). Past the window calc panics.
func (w *mstWorld) alloc(n uint32) uint32 {
	sz := (n + 7) &^ 7
	if sz < 8 {
		sz = 8
	}
	if w.heap+sz > mstHeapWindow {
		panic("monster spell target: heap allocations past the fixture window")
	}
	a := mstHeap + w.heap
	w.heap += sz
	return a
}

const mstRet = 0xfffffff0 // out field marker: the function's return value

var mstExe struct {
	once sync.Once
	b    []byte
	err  error
}

func mstImage() []byte {
	mstExe.once.Do(func() {
		path := os.Getenv("WIZ8_EXE")
		if path == "" {
			_, source, _, ok := runtime.Caller(0)
			if !ok {
				mstExe.err = fmt.Errorf("monster spell target: set WIZ8_EXE")
				return
			}
			path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
		}
		mstExe.b, mstExe.err = os.ReadFile(path)
	})
	if mstExe.err != nil {
		panic(mstExe.err)
	}
	return mstExe.b
}

type mstWorld struct {
	arena []byte
	ov    map[uint32]byte
	spec  *mstSpec
	r     Rand
	heap  uint32  // bump-allocator offset into the emulator heap (base mstHeap)
	extra []int64 // trailing node inputs only the real 0x4fd030 uses (dead in Emu)
}

func (w *mstWorld) u8(a uint32) byte {
	if a >= mstArena && a < mstArena+mstArenaSize {
		return w.arena[a-mstArena]
	}
	if v, ok := w.ov[a]; ok {
		return v
	}
	b := mstImage()
	if off, err := peOffset(b, a); err == nil {
		return b[off]
	}
	return 0
}

func (w *mstWorld) u32(a uint32) uint32 {
	return uint32(w.u8(a)) | uint32(w.u8(a+1))<<8 | uint32(w.u8(a+2))<<16 | uint32(w.u8(a+3))<<24
}

func (w *mstWorld) put(a uint32, size int, v uint64) {
	for i := 0; i < size; i++ {
		x := byte(v >> (8 * i))
		if p := a + uint32(i); p >= mstArena && p < mstArena+mstArenaSize {
			w.arena[p-mstArena] = x
		} else {
			w.ov[p] = x
		}
	}
}

func mstFits(v int64, size uint8, signed bool) bool {
	bits := uint(size) * 8
	if signed {
		return v >= -(1<<(bits-1)) && v < 1<<(bits-1)
	}
	return v >= 0 && (bits == 64 || v < 1<<bits)
}

func (w *mstWorld) address(f mstFld, in []int64) (uint32, bool) {
	if f.idx < 0 {
		return f.addr, true
	}
	i := in[f.idx]
	if i < 0 {
		return 0, false
	}
	if uint32(i) >= f.count || i > 1<<32 {
		panic("monster spell target: record index past the fixture count")
	}
	return f.addr + uint32(i)*f.stride, true
}

func (w *mstWorld) write(f mstFld, v int64, in []int64) {
	if !mstFits(v, f.size, f.signed) {
		panic("monster spell target: input does not fit its field")
	}
	if a, ok := w.address(f, in); ok {
		w.put(a, int(f.size), uint64(v))
	}
}

func (w *mstWorld) read(f mstFld, in []int64) int64 {
	a, ok := w.address(f, in)
	if !ok {
		panic("monster spell target: output index is negative")
	}
	var v uint64
	for i := 0; i < int(f.size); i++ {
		v |= uint64(w.u8(a+uint32(i))) << (8 * i)
	}
	if f.signed && f.size < 8 && v&(1<<(8*uint(f.size)-1)) != 0 {
		return int64(v) - 1<<(8*uint(f.size))
	}
	return int64(v)
}

// mstBuild applies setup, then the inputs in order (emu.py Machine.__init__).
func mstBuild(spec *mstSpec, in []int64) *mstWorld {
	w := &mstWorld{arena: make([]byte, mstArenaSize), ov: map[uint32]byte{}, spec: spec}
	for _, s := range spec.set {
		w.put(s.addr, int(s.size), uint64(s.val))
	}
	for i, f := range spec.in {
		w.write(f, in[i], in)
	}
	return w
}

// note is the shared stub layer: it copies the recorded call arguments (1-based
// stack positions) and bumps the call counter, whichever implementation runs.
func (w *mstWorld) note(k int, args ...uint32) {
	s := &w.spec.stub[k]
	for _, r := range s.rec {
		w.put(r[2], int(r[1]), uint64(args[r[0]-1]))
	}
	if s.cnt != 0 {
		w.put(s.cnt, 4, uint64(w.u32(s.cnt)+1))
	}
}

// stubRet is what an address stub returns: the u32 at its ret input position.
func (w *mstWorld) stubRet(k int) uint32 { return w.u32(w.spec.stub[k].ret) }

// ---- helpers shared by the member bodies (the callees that run natively) ----

func (w *mstWorld) spellRecord(spell uint32) uint32 {
	if spell >= mstSpellCap {
		panic("monster spell target: spell id past the fixture's 256 spell records")
	}
	return w.u32(0x65be1c) + spell*mstSpellSize
}

// mstSpellType is 0x4ac9d0: the spell record's +0x137 dword, 0 when it is 1 and
// the flag is set.
func (w *mstWorld) spellType(spell uint32, flag bool) uint32 {
	v := w.u32(w.spellRecord(spell) + 0x137)
	if v == 1 && flag {
		return 0
	}
	return v
}

func (w *mstWorld) charBase(i uint32) uint32 {
	if i >= mstMaxChar {
		panic("monster spell target: party index past 7")
	}
	return w.u32(0x685174) + i*mstCharStride
}

// listCount is 0x5e2c70, listAt is 0x5e2870 (entries pointer at +0, count at +8).
func (w *mstWorld) listCount(l uint32) uint32 {
	if l == 0 {
		return 0
	}
	return w.u32(l + 8)
}

func (w *mstWorld) listAt(l uint32, i uint32) uint32 {
	if l == 0 || int32(i) >= int32(w.u32(l+8)) {
		return 0
	}
	return w.u32(w.u32(l) + i*4)
}

// monsterRecord is 0x4e5620: indexes 0x2710..0x4e1f address the reserve list.
func (w *mstWorld) monsterRecord(i uint32) uint32 {
	list, k := w.u32(0x683fad), i
	if i >= 0x2710 && i < 0x4e20 {
		list, k = w.u32(0x683fb9), i-0x2710
	}
	if k >= w.listCount(list) { // 0x4e5651 / 0x4e56b9: srAssertFail.
		panic("monster spell target: monster handle past its list")
	}
	r := w.listAt(list, k)
	if r == 0 {
		panic("monster spell target: null monster record")
	}
	return r
}

// monsterHandle is 0x4e5550 without the assert: it searches the monster list then
// the reserve list (handle + 0x2710) for the record whose +0 equals id.
func (w *mstWorld) monsterHandle(id uint32) (uint32, bool) {
	for _, base := range [2]struct{ list, add uint32 }{{w.u32(0x683fad), 0}, {w.u32(0x683fb9), 0x2710}} {
		for i := uint32(0); i < w.listCount(base.list); i++ {
			r := w.listAt(base.list, i)
			if r == 0 {
				panic("monster spell target: null monster record")
			}
			if w.u32(r) == id {
				return i + base.add, true
			}
		}
	}
	return 0, false
}

// findMonster is 0x4e5840 with the assert flag set.
func (w *mstWorld) findMonster(id uint32) uint32 {
	h, ok := w.monsterHandle(id)
	if !ok {
		panic("monster spell target: monster id not found (0x4e5840 asserts)")
	}
	return w.monsterRecord(h)
}

// template is 0x4e5720: the template cache is a direct array at 0x6840c7. A miss
// would load the template file (0x54a8a0); the fixture preloads template ids
// below mstTemplates and cannot model a miss.
func (w *mstWorld) template(m uint32) uint32 {
	if m == 0 {
		panic("monster spell target: null monster")
	}
	id := w.u32(m + 8)
	if id >= 0x3e8 {
		panic("monster spell target: template id past 0x3e7 (0x4e5749 asserts)")
	}
	if id >= mstTemplates {
		panic("monster spell target: template cache miss is outside the fixture (file load)")
	}
	t := w.u32(0x6840c7 + 4*id)
	if t == 0 {
		panic("monster spell target: template cache entry is empty")
	}
	return t
}

// animFlag is 0x4c5a80 on the one object class the fixture builds (vtable 0x5ed22c,
// slot 6): the dword at state+((id+0xb)<<4) is nonzero, state = obj+0x1d8's pointer.
func (w *mstWorld) animFlag(obj, id uint32) bool {
	if obj == 0 {
		return false
	}
	if w.u32(obj) != 0x5ed22c {
		panic("monster spell target: object class is outside the fixture")
	}
	if id >= 0x1b {
		panic("monster spell target: anim flag id past 0x1a (assert)")
	}
	return w.u32(w.u32(obj+0x1d8)+(id+0xb)<<4) != 0
}

// spellBlockedByKind is 0x4fb1d0 (monster m, spell id), true when it returns 1:
// m+0x77 nonzero and (the spell has no +0x48 flag, or the template's +0xcb kind
// is outside 4..5 and 0xd).
func (w *mstWorld) spellBlockedByKind(m, spell uint32) bool {
	if w.u32(m+0x77) == 0 {
		return false
	}
	if w.u8(w.spellRecord(spell)+0x48) == 0 {
		return true
	}
	k := w.u8(w.template(m) + 0xcb)
	return !(k >= 4 && (k <= 5 || k == 0xd))
}

// monsterCanCast is 0x532550 with flag clear (the only form 0x5327e0 uses).
func (w *mstWorld) monsterCanCast(m, spell uint32) bool {
	if spell == 0 {
		return false
	}
	rec := w.spellRecord(spell)
	if w.u8(rec+0x126) == 0 {
		return false
	}
	if w.spellBlockedByKind(m, spell) {
		return false
	}
	w.note(mstUI, m)
	if byte(w.stubRet(mstUI)) != 0 {
		return false
	}
	if spell == 0x3c && w.u32(m+0x344) != 0xffffffff {
		return false
	}
	if w.u32(rec+0x133) == 0 && w.u8(0x652da4) != 0 {
		return false
	}
	return true
}

// mstCallees are the callees that differ between production (the real Go
// functions) and the emulator replay (address stubs that read their return from
// the case input and record their arguments).
type mstCallees struct {
	power func(w *mstWorld, m, tmpl, spell uint32) uint32
	area  func(w *mstWorld, spell, pl uint32) (mons, chars []uint32)
	disp  func(w *mstWorld, i, m uint32) uint32
	// rand (0x518150) and pen (0x51aec0) are only reached from 0x533320.
	rand func(w *mstWorld, a, b, c, d uint32) uint32
	pen  func(w *mstWorld, ch, m uint32) uint32
}

func (w *mstWorld) logCall(arg3 uint32) { w.note(mstLog, 0, 0, arg3) }

// ok is 0x5327e0 (monster m, spell id, target q = {type, character, monster id}).
// Type 1 is a party character, 3 a monster, anything else the area path.
func (w *mstWorld) ok(c *mstCallees, m, spell, qt, q1, q2 uint32) bool {
	if w.spellType(spell, false) == 8 {
		return true
	}
	var maxHP, hp, st, maxSt, a, b, ch, tgt, tmpl uint32
	switch qt {
	case 1:
		ch = w.charBase(q1)
		maxHP, hp, st, maxSt = w.u32(ch+0xb0d), w.u32(ch+0xb11), w.u32(ch+0xb1d), w.u32(ch+0xb19)
		a, b = ch+0xa01, ch+0xa65
	case 3:
		tgt = w.findMonster(q2)
		tmpl = w.template(tgt)
		maxSt, st, hp, maxHP = w.u32(tgt+0x2f), w.u32(tgt+0x33), w.u32(tgt+0x2b), w.u32(tgt+0x27)
		a, b = tgt+0x57, tgt+0xa7
		if w.u8(m+0x16) == 1 && w.u8(tgt+0x16) == 2 && w.u8(m+0x15) != 0 && w.u8(w.u32(m+0x10)+0x152) != 0 {
			return false
		}
	default:
		return w.okArea(c, m, spell)
	}
	t := func(tbl uint32, n uint32, f func(k uint32) (uint32, bool)) (hit bool) {
		for k := uint32(0); k < n; k++ {
			if spell == w.u32(tbl+4*k) {
				if v, used := f(k); used && v != 0 {
					return true
				}
			}
		}
		return false
	}
	ctx := w.u32(0x6836a8)
	// slot reads shared by the three status-table groups: characters read the battle
	// context, monsters (only when +0x15 is set) their extra record at +0x10.
	slot := func(ctxOff, monOff uint32) func(k uint32) (uint32, bool) {
		return func(k uint32) (uint32, bool) {
			if qt == 1 {
				return w.u32(ctx + ctxOff + 0x11*k), true
			}
			if w.u8(tgt+0x15) == 0 {
				return 0, false
			}
			return w.u32(w.u32(tgt+0x10) + monOff + 0x11*k), true
		}
	}
	nz := func(p uint32) bool { return w.u32(p) != 0 }
	switch spell {
	case 1, 4, 5, 9, 10, 0x19, 0x1c, 0x24, 0x2a, 0x2b, 0x2f, 0x32, 0x34, 0x37, 0x39, 0x3c, 0x3f, 0x42, 0x46, 0x47,
		0x4e, 0x4f, 0x53, 0x56, 0x57, 0x5a, 0x5b, 0x5c, 0x5e, 0x5f, 0x60, 0x61, 0x62, 99, 0x65, 0x77:
	case 2, 0x35, 0x3b:
		if w.u8(0x683f94) == 0 {
			return true
		}
		return !t(0x616dd8, 9, slot(0x867, 0xe4))
	case 6, 0x44:
		if hp == maxHP {
			return false
		}
	case 7:
		return !nz(a + 0xc)
	case 0xb, 0x25, 0x43:
		return !nz(a + 0x40)
	case 0xc:
		return !nz(a + 0x3c)
	case 0xd, 0x2c:
		if st == maxSt {
			return false
		}
	case 0xe:
		return !nz(a + 0x18)
	case 0xf:
		return !nz(a + 0x30)
	case 0x10:
		if !nz(a+0xc) && !nz(a+0x10) && !nz(a+0x18) && !nz(a+0x3c) && !nz(a+0x30) {
			return false
		}
	case 0x14, 0x1a, 0x20, 0x28:
		if t(0x616d84, 12, func(k uint32) (uint32, bool) {
			if qt == 1 {
				return w.u32(0x68692c + 0x11*k), true
			}
			return w.u32(tgt + 0x11c + 0x11*k), true
		}) {
			return false
		}
		if spell == 0x14 && qt == 3 && w.u8(tmpl+0xe0) > 0x3b {
			return false
		}
	case 0x15:
		return !nz(b + 0x20)
	case 0x1b:
		return !nz(b + 0x2c)
	case 0x1d:
		return !nz(a + 0x14)
	case 0x1f:
		return !nz(a + 0x38)
	case 0x22:
		return nz(a + 0x40)
	case 0x23:
		return nz(a + 0x1c)
	case 0x2e:
		return w.okRegen(c, qt, ch, tgt, tmpl, a, spell)
	case 0x30, 0x31, 0x4c, 0x50, 0x51, 0x5d:
		if w.u8(0x683f94) == 0 {
			return true
		}
		return !t(0x616db4, 9, slot(0x7ce, 0x4b))
	case 0x36:
		return !nz(b + 0x38)
	case 0x38:
		return !nz(b + 0x44)
	case 0x3d:
		return !nz(b + 0x50)
	case 0x41:
		return !nz(b + 0x5c)
	case 0x45:
		return !nz(a + 0x24)
	case 0x48:
		if w.u8(0x683f94) == 0 {
			return false
		}
		sl := slot(0x7ce, 0x4b)
		for k := uint32(0); k < 9; k++ {
			if w.u32(0x616db4+4*k) != 0x31 {
				if v, used := sl(k); used && v != 0 {
					return true
				}
			}
		}
		return false
	case 0x4a:
		if !nz(a+0x2c) && !nz(a+0x34) {
			return false
		}
	case 0x4d:
		if qt == 1 {
			return false
		}
		if k := w.u8(tmpl + 0xcb); k != 0x14 && k != 0x15 && k != 0x1c && w.u32(tgt+0x2da) == 0 {
			return false
		}
	case 0x55, 0x18:
		if spell == 0x55 && !nz(a+0x18) {
			return true
		}
		return !nz(a + 0x2c)
	case 0x59:
		return !nz(a + 0x34)
	default:
		w.logCall(spell)
	}
	return true
}

// okRegen is the spell 0x2e arm of 0x5327e0.
func (w *mstWorld) okRegen(c *mstCallees, qt, ch, tgt, tmpl, a, spell uint32) bool {
	if w.u32(a+0x20) != 0 {
		return false
	}
	if qt != 1 {
		if k := w.u8(tmpl + 0xcb); k > 3 && (k < 6 || k == 0xd) {
			return false
		}
		if w.u8(tmpl+0xe0) != 0 {
			if !w.animFlag(w.u32(tgt+0xc), 0x19) {
				if w.u8(0x68d524) == 0 {
					w.logCall(tmpl)
					w.put(0x68d524, 1, 1)
				}
			} else {
				for k := uint32(0); k < 10; k++ {
					if w.monsterCanCast(tgt, uint32(w.u8(tmpl+0x14d+k))) {
						return true
					}
				}
			}
		}
		return false
	}
	if w.u32(ch+0x36b) != 0 {
		return true
	}
	found := false
	for k := uint32(0); k < 0x72; k++ {
		if w.u32(ch+0xbcd+4*k) == 1 {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	return !(w.u8(w.spellRecord(spell)+0x48) != 0 && w.u32(ch+0x57f) != 0)
}

// okArea is the q type 0 / 2 / other arm: 0x4fd030 lists the targets the spell
// area covers, each is judged by 0x5327e0 itself, and at least half must pass.
func (w *mstWorld) okArea(c *mstCallees, m, spell uint32) bool {
	tmpl := w.template(m)
	w.alloc(0x14) // the two target vectors (0x5e1ce0)
	w.alloc(0x14)
	pl := c.power(w, m, tmpl, spell)
	mons, chars := c.area(w, spell, pl)
	var n, okc uint32
	for _, e := range chars {
		n++
		if w.ok(c, m, spell, 1, e, 0) {
			okc++
		}
	}
	for _, e := range mons {
		n++
		if w.ok(c, m, spell, 3, 0, e) {
			okc++
		}
	}
	return okc != 0 && okc >= (n+1)>>1
}

// ---- callee implementations ----

// Emu callees read the stub return from the case input and record arguments.
var mstEmuCallees = mstCallees{
	power: func(w *mstWorld, m, tmpl, spell uint32) uint32 {
		w.note(mstPower, m, tmpl, spell)
		return w.stubRet(mstPower)
	},
	area: func(w *mstWorld, spell, pl uint32) ([]uint32, []uint32) {
		w.note(mstArea, spell, pl)
		return w.spec.areaMons, w.spec.areaChars
	},
	disp: func(w *mstWorld, i, m uint32) uint32 {
		w.note(mstDisp, i, m)
		return w.stubRet(mstDisp)
	},
	rand: func(w *mstWorld, a, b, c, d uint32) uint32 {
		w.note(mstRand, a, b, c, d)
		return w.stubRet(mstRand)
	},
	pen: func(w *mstWorld, ch, m uint32) uint32 {
		w.note(mstPenalty, ch, m)
		return w.stubRet(mstPenalty)
	},
}

// Real callees call the registered Go nodes (0x500330, 0x4fd030, 0x546f10) on the
// values the original would hand them; the arguments are noted exactly as in Emu.
var mstRealCallees = mstCallees{
	power: func(w *mstWorld, m, tmpl, spell uint32) uint32 {
		w.note(mstPower, m, tmpl, spell)
		rec := w.spellRecord(spell)
		in := []int64{int64(int32(w.u32(m + 0x2f9))), int64(int32(w.u32(tmpl + 0x26f))), int64(w.u32(m + 8)), int64(spell),
			int64(int32(w.u32(rec + 0x49))), int64(int32(w.u32(rec + 0x56)))}
		return uint32(monsterSpellPowerLevelNode(in, w.r)[0])
	},
	area: func(w *mstWorld, spell, pl uint32) ([]uint32, []uint32) {
		w.note(mstArea, spell, pl)
		if len(w.extra) < len(spellAreaInputs) {
			panic("monster spell target: the real 0x4fd030 needs its own " + fmt.Sprint(len(spellAreaInputs)) + " trailing inputs")
		}
		x := append([]int64(nil), w.extra[:len(spellAreaInputs)]...)
		for i, nm := range spellAreaInputs {
			switch nm {
			case "spellID":
				x[i] = int64(spell)
			case "powerLevel":
				x[i] = int64(pl)
			}
		}
		o := spellAreaTargetSelectionNode(x, w.r)
		list := func(count int64, first int) []uint32 {
			if count < 0 || count > 8 {
				panic("monster spell target: 0x4fd030 target count past 8")
			}
			l := make([]uint32, count)
			for i := range l {
				l[i] = uint32(o[first+i])
			}
			return l
		}
		return list(o[0], 2), list(o[1], 10)
	},
	disp: func(w *mstWorld, i, m uint32) uint32 {
		w.note(mstDisp, i, m)
		return uint32(targetDispositionNode([]int64{int64(i), int64(int32(w.u32(w.charBase(i) + 0xa35))), int64(w.u8(m + 0x16))}, w.r)[0])
	},
	// 0x518150 on the party table: slot presence, hp (+0xb11), level (+0xb01), class key (+0x75).
	rand: func(w *mstWorld, a, b, c, d uint32) uint32 {
		w.note(mstRand, a, b, c, d)
		in := []int64{int64(a), int64(b), int64(int32(c)), int64(int32(d))}
		for _, f := range []func(i uint32) int64{
			func(i uint32) int64 { return int64(w.u8(w.u32(0x685178) + 0x106*i)) },
			func(i uint32) int64 { return int64(w.u32(w.charBase(i) + 0xb11)) },
			func(i uint32) int64 { return int64(w.u32(w.charBase(i) + 0xb01)) },
			func(i uint32) int64 { return int64(int8(w.u8(w.charBase(i) + 0x75))) },
		} {
			for i := uint32(0); i < mstMaxChar; i++ {
				in = append(in, f(i))
			}
		}
		return uint32(int32(randomPartyMemberNode(in, w.r)[0]))
	},
	// 0x51aec0 reads formation globals this fixture keeps constant: they ride in as
	// trailing inputs behind the 0x4fd030 ones; only the character index is live.
	pen: func(w *mstWorld, ch, m uint32) uint32 {
		w.note(mstPenalty, ch, m)
		if len(w.extra) != len(spellAreaInputs)+39 {
			panic("monster spell target: the real 0x51aec0 needs its own 39 trailing inputs")
		}
		x := append([]int64(nil), w.extra[len(spellAreaInputs):]...)
		x[0] = int64(ch)
		return uint32(formationRangePenaltyNode(x, w.r)[0])
	},
}

// mstNode runs one member over the case image. nExtra trailing inputs are kept for
// the real callees (they are absent from the emulator case).
func mstNode(spec *mstSpec, c *mstCallees, run func(w *mstWorld, c *mstCallees) bool, nExtra int) func([]int64, Rand) []int64 {
	return func(in []int64, r Rand) []int64 {
		n := len(spec.in)
		if len(in) != n {
			panic("monster spell target: wrong input count")
		}
		w := mstBuild(spec, in)
		w.r, w.extra = r, in[n-nExtra:]
		ret := int64(0)
		if run(w, c) {
			ret = 1
		}
		out := make([]int64, len(spec.out))
		for i, f := range spec.out {
			if f.addr == mstRet {
				out[i] = ret
			} else {
				out[i] = w.read(f, in)
			}
		}
		return out
	}
}

func mstRunOk(w *mstWorld, c *mstCallees) bool {
	q := w.u32(mstStackBase + 12)
	return w.ok(c, w.u32(mstStackBase+4), w.u32(mstStackBase+8), w.u32(q), w.u32(q+4), w.u32(q+8))
}

// monsterSpellTargetOkNode is 0x5327e0 with the real callees.
func monsterSpellTargetOkNode(in []int64, r Rand) []int64 {
	return mstNode(&mstSpec_ok, &mstRealCallees, mstRunOk, len(spellAreaInputs))(in, r)
}

func mstEmuOk(in []int64, r Rand) []int64 {
	return mstNode(&mstSpec_ok, &mstEmuCallees, mstRunOk, len(spellAreaInputs))(in, r)
}

func init() {
	Nodes["005327e0"] = Node{Addr: "005327e0",
		In:  mstNames_ok_in(),
		Out: mstNames_ok_out(), F: monsterSpellTargetOkNode, Emu: mstEmuOk}
}

var mstSpec_ok = mstSpec{
	set: []mstSet{
		{0x7f000004, 4, 0x20028db0},
		{0x7f00000c, 4, 0x20000140},
		{0x685174, 4, 0x20000280},
		{0x685178, 4, 0x2000c5a0},
		{0x683fad, 4, 0x2002abb0},
		{0x683fb9, 4, 0x2002ac30},
		{0x6840c7, 4, 0x200293b0},
		{0x6840cb, 4, 0x200297b0},
		{0x65be1c, 4, 0x2000cea0},
		{0x6836a8, 4, 0x20029bb0},
		{0x2002abb0, 4, 0x2002abf0},
		{0x2002abb8, 4, 0x3},
		{0x2002ac30, 4, 0x2002ac70},
		{0x2002ac38, 4, 0x1},
		{0x2002abf0, 4, 0x2002acb0},
		{0x2002abf4, 4, 0x2002b0b0},
		{0x2002abf8, 4, 0x2002b4b0},
		{0x2002ac70, 4, 0x2002b8b0},
		{0x2002acb0, 4, 0x0},
		{0x2002acbc, 4, 0x2002c4b0},
		{0x2002acc0, 4, 0x2002bcb0},
		{0x2002c4b0, 4, 0x5ed22c},
		{0x2002c688, 4, 0x2002ccb0},
		{0x2002b0b0, 4, 0x1},
		{0x2002b0bc, 4, 0x2002c6b0},
		{0x2002b0c0, 4, 0x2002beb0},
		{0x2002c6b0, 4, 0x5ed22c},
		{0x2002c888, 4, 0x2002d0b0},
		{0x2002b4b0, 4, 0x2},
		{0x2002b4bc, 4, 0x2002c8b0},
		{0x2002b4c0, 4, 0x2002c0b0},
		{0x2002c8b0, 4, 0x5ed22c},
		{0x2002ca88, 4, 0x2002d4b0},
		{0x2002b8b0, 4, 0x3},
		{0x2002b8bc, 4, 0x2002cab0},
		{0x2002b8c0, 4, 0x2002c2b0},
		{0x2002cab0, 4, 0x5ed22c},
		{0x2002cc88, 4, 0x2002d8b0},
		{0x20028db0, 4, 0x77},
		{0x20028dc0, 4, 0x200291b0},
		{0x2000cfd7, 4, 0x5},
		{0x2000d196, 4, 0x5},
		{0x2000d355, 4, 0x5},
		{0x2000d514, 4, 0x5},
		{0x2000d6d3, 4, 0x5},
		{0x2000d892, 4, 0x5},
		{0x2000da51, 4, 0x5},
		{0x2000dc10, 4, 0x5},
		{0x2000ddcf, 4, 0x5},
		{0x2000df8e, 4, 0x5},
		{0x2000e14d, 4, 0x5},
		{0x2000e30c, 4, 0x5},
		{0x2000e4cb, 4, 0x5},
		{0x2000e68a, 4, 0x5},
		{0x2000e849, 4, 0x5},
		{0x2000ea08, 4, 0x5},
		{0x2000ebc7, 4, 0x5},
		{0x2000ed86, 4, 0x5},
		{0x2000ef45, 4, 0x5},
		{0x2000f104, 4, 0x5},
		{0x2000f2c3, 4, 0x5},
		{0x2000f482, 4, 0x5},
		{0x2000f641, 4, 0x5},
		{0x2000f800, 4, 0x5},
		{0x2000f9bf, 4, 0x5},
		{0x2000fb7e, 4, 0x5},
		{0x2000fd3d, 4, 0x5},
		{0x2000fefc, 4, 0x5},
		{0x200100bb, 4, 0x5},
		{0x2001027a, 4, 0x5},
		{0x20010439, 4, 0x5},
		{0x200105f8, 4, 0x5},
		{0x200107b7, 4, 0x5},
		{0x20010976, 4, 0x5},
		{0x20010b35, 4, 0x5},
		{0x20010cf4, 4, 0x5},
		{0x20010eb3, 4, 0x5},
		{0x20011072, 4, 0x5},
		{0x20011231, 4, 0x5},
		{0x200113f0, 4, 0x5},
		{0x200115af, 4, 0x5},
		{0x2001176e, 4, 0x5},
		{0x2001192d, 4, 0x5},
		{0x20011aec, 4, 0x5},
		{0x20011cab, 4, 0x5},
		{0x20011e6a, 4, 0x5},
		{0x20012029, 4, 0x5},
		{0x200121e8, 4, 0x5},
		{0x200123a7, 4, 0x5},
		{0x20012566, 4, 0x5},
		{0x20012725, 4, 0x5},
		{0x200128e4, 4, 0x5},
		{0x20012aa3, 4, 0x5},
		{0x20012c62, 4, 0x5},
		{0x20012e21, 4, 0x5},
		{0x20012fe0, 4, 0x5},
		{0x2001319f, 4, 0x5},
		{0x2001335e, 4, 0x5},
		{0x2001351d, 4, 0x5},
		{0x200136dc, 4, 0x5},
		{0x2001389b, 4, 0x5},
		{0x20013a5a, 4, 0x5},
		{0x20013c19, 4, 0x5},
		{0x20013dd8, 4, 0x5},
		{0x20013f97, 4, 0x5},
		{0x20014156, 4, 0x5},
		{0x20014315, 4, 0x5},
		{0x200144d4, 4, 0x5},
		{0x20014693, 4, 0x5},
		{0x20014852, 4, 0x5},
		{0x20014a11, 4, 0x5},
		{0x20014bd0, 4, 0x5},
		{0x20014d8f, 4, 0x5},
		{0x20014f4e, 4, 0x5},
		{0x2001510d, 4, 0x5},
		{0x200152cc, 4, 0x5},
		{0x2001548b, 4, 0x5},
		{0x2001564a, 4, 0x5},
		{0x20015809, 4, 0x5},
		{0x200159c8, 4, 0x5},
		{0x20015b87, 4, 0x5},
		{0x20015d46, 4, 0x5},
		{0x20015f05, 4, 0x5},
		{0x200160c4, 4, 0x5},
		{0x20016283, 4, 0x5},
		{0x20016442, 4, 0x5},
		{0x20016601, 4, 0x5},
		{0x200167c0, 4, 0x5},
		{0x2001697f, 4, 0x5},
		{0x20016b3e, 4, 0x5},
		{0x20016cfd, 4, 0x5},
		{0x20016ebc, 4, 0x5},
		{0x2001707b, 4, 0x5},
		{0x2001723a, 4, 0x5},
		{0x200173f9, 4, 0x5},
		{0x200175b8, 4, 0x5},
		{0x20017777, 4, 0x5},
		{0x20017936, 4, 0x5},
		{0x20017af5, 4, 0x5},
		{0x20017cb4, 4, 0x5},
		{0x20017e73, 4, 0x5},
		{0x20018032, 4, 0x5},
		{0x200181f1, 4, 0x5},
		{0x200183b0, 4, 0x5},
		{0x2001856f, 4, 0x5},
		{0x2001872e, 4, 0x5},
		{0x200188ed, 4, 0x5},
		{0x20018aac, 4, 0x5},
		{0x20018c6b, 4, 0x5},
		{0x20018e2a, 4, 0x5},
		{0x20018fe9, 4, 0x5},
		{0x200191a8, 4, 0x5},
		{0x20019367, 4, 0x5},
		{0x20019526, 4, 0x5},
		{0x200196e5, 4, 0x5},
		{0x200198a4, 4, 0x5},
		{0x20019a63, 4, 0x5},
		{0x20019c22, 4, 0x5},
		{0x20019de1, 4, 0x5},
		{0x20019fa0, 4, 0x5},
		{0x2001a15f, 4, 0x5},
		{0x2001a31e, 4, 0x5},
		{0x2001a4dd, 4, 0x5},
		{0x2001a69c, 4, 0x5},
		{0x2001a85b, 4, 0x5},
		{0x2001aa1a, 4, 0x5},
		{0x2001abd9, 4, 0x5},
		{0x2001ad98, 4, 0x5},
		{0x2001af57, 4, 0x5},
		{0x2001b116, 4, 0x5},
		{0x2001b2d5, 4, 0x5},
		{0x2001b494, 4, 0x5},
		{0x2001b653, 4, 0x5},
		{0x2001b812, 4, 0x5},
		{0x2001b9d1, 4, 0x5},
		{0x2001bb90, 4, 0x5},
		{0x2001bd4f, 4, 0x5},
		{0x2001bf0e, 4, 0x5},
		{0x2001c0cd, 4, 0x5},
		{0x2001c28c, 4, 0x5},
		{0x2001c44b, 4, 0x5},
		{0x2001c60a, 4, 0x5},
		{0x2001c7c9, 4, 0x5},
		{0x2001c988, 4, 0x5},
		{0x2001cb47, 4, 0x5},
		{0x2001cd06, 4, 0x5},
		{0x2001cec5, 4, 0x5},
		{0x2001d084, 4, 0x5},
		{0x2001d243, 4, 0x5},
		{0x2001d402, 4, 0x5},
		{0x2001d5c1, 4, 0x5},
		{0x2001d780, 4, 0x5},
		{0x2001d93f, 4, 0x5},
		{0x2001dafe, 4, 0x5},
		{0x2001dcbd, 4, 0x5},
		{0x2001de7c, 4, 0x5},
		{0x2001e03b, 4, 0x5},
		{0x2001e1fa, 4, 0x5},
		{0x2001e3b9, 4, 0x5},
		{0x2001e578, 4, 0x5},
		{0x2001e737, 4, 0x5},
		{0x2001e8f6, 4, 0x5},
		{0x2001eab5, 4, 0x5},
		{0x2001ec74, 4, 0x5},
		{0x2001ee33, 4, 0x5},
		{0x2001eff2, 4, 0x5},
		{0x2001f1b1, 4, 0x5},
		{0x2001f370, 4, 0x5},
		{0x2001f52f, 4, 0x5},
		{0x2001f6ee, 4, 0x5},
		{0x2001f8ad, 4, 0x5},
		{0x2001fa6c, 4, 0x5},
		{0x2001fc2b, 4, 0x5},
		{0x2001fdea, 4, 0x5},
		{0x2001ffa9, 4, 0x5},
		{0x20020168, 4, 0x5},
		{0x20020327, 4, 0x5},
		{0x200204e6, 4, 0x5},
		{0x200206a5, 4, 0x5},
		{0x20020864, 4, 0x5},
		{0x20020a23, 4, 0x5},
		{0x20020be2, 4, 0x5},
		{0x20020da1, 4, 0x5},
		{0x20020f60, 4, 0x5},
		{0x2002111f, 4, 0x5},
		{0x200212de, 4, 0x5},
		{0x2002149d, 4, 0x5},
		{0x2002165c, 4, 0x5},
		{0x2002181b, 4, 0x5},
		{0x200219da, 4, 0x5},
		{0x20021b99, 4, 0x5},
		{0x20021d58, 4, 0x5},
		{0x20021f17, 4, 0x5},
		{0x200220d6, 4, 0x5},
		{0x20022295, 4, 0x5},
		{0x20022454, 4, 0x5},
		{0x20022613, 4, 0x5},
		{0x200227d2, 4, 0x5},
		{0x20022991, 4, 0x5},
		{0x20022b50, 4, 0x5},
		{0x20022d0f, 4, 0x5},
		{0x20022ece, 4, 0x5},
		{0x2002308d, 4, 0x5},
		{0x2002324c, 4, 0x5},
		{0x2002340b, 4, 0x5},
		{0x200235ca, 4, 0x5},
		{0x20023789, 4, 0x5},
		{0x20023948, 4, 0x5},
		{0x20023b07, 4, 0x5},
		{0x20023cc6, 4, 0x5},
		{0x20023e85, 4, 0x5},
		{0x20024044, 4, 0x5},
		{0x20024203, 4, 0x5},
		{0x200243c2, 4, 0x5},
		{0x20024581, 4, 0x5},
		{0x20024740, 4, 0x5},
		{0x200248ff, 4, 0x5},
		{0x20024abe, 4, 0x5},
		{0x20024c7d, 4, 0x5},
		{0x20024e3c, 4, 0x5},
		{0x20024ffb, 4, 0x5},
		{0x200251ba, 4, 0x5},
		{0x20025379, 4, 0x5},
		{0x20025538, 4, 0x5},
		{0x200256f7, 4, 0x5},
		{0x200258b6, 4, 0x5},
		{0x20025a75, 4, 0x5},
		{0x20025c34, 4, 0x5},
		{0x20025df3, 4, 0x5},
		{0x20025fb2, 4, 0x5},
		{0x20026171, 4, 0x5},
		{0x20026330, 4, 0x5},
		{0x200264ef, 4, 0x5},
		{0x200266ae, 4, 0x5},
		{0x2002686d, 4, 0x5},
		{0x20026a2c, 4, 0x5},
		{0x20026beb, 4, 0x5},
		{0x20026daa, 4, 0x5},
		{0x20026f69, 4, 0x5},
		{0x20027128, 4, 0x5},
		{0x200272e7, 4, 0x5},
		{0x200274a6, 4, 0x5},
		{0x20027665, 4, 0x5},
		{0x20027824, 4, 0x5},
		{0x200279e3, 4, 0x5},
		{0x20027ba2, 4, 0x5},
		{0x20027d61, 4, 0x5},
		{0x20027f20, 4, 0x5},
		{0x200280df, 4, 0x5},
		{0x2002829e, 4, 0x5},
		{0x2002845d, 4, 0x5},
		{0x2002861c, 4, 0x5},
		{0x200287db, 4, 0x5},
		{0x2002899a, 4, 0x5},
		{0x20028b59, 4, 0x5},
		{0x20028d18, 4, 0x5},
		{0x2000cee8, 1, 0x2},
		{0x2000d0a7, 1, 0x2},
		{0x2000d266, 1, 0x2},
		{0x2000d425, 1, 0x2},
		{0x2000d5e4, 1, 0x2},
		{0x2000d7a3, 1, 0x2},
		{0x2000d962, 1, 0x2},
		{0x2000db21, 1, 0x2},
		{0x2000dce0, 1, 0x2},
		{0x2000de9f, 1, 0x2},
		{0x2000e05e, 1, 0x2},
		{0x2000e21d, 1, 0x2},
		{0x2000e3dc, 1, 0x2},
		{0x2000e59b, 1, 0x2},
		{0x2000e75a, 1, 0x2},
		{0x2000e919, 1, 0x2},
		{0x2000ead8, 1, 0x2},
		{0x2000ec97, 1, 0x2},
		{0x2000ee56, 1, 0x2},
		{0x2000f015, 1, 0x2},
		{0x2000f1d4, 1, 0x2},
		{0x2000f393, 1, 0x2},
		{0x2000f552, 1, 0x2},
		{0x2000f711, 1, 0x2},
		{0x2000f8d0, 1, 0x2},
		{0x2000fa8f, 1, 0x2},
		{0x2000fc4e, 1, 0x2},
		{0x2000fe0d, 1, 0x2},
		{0x2000ffcc, 1, 0x2},
		{0x2001018b, 1, 0x2},
		{0x2001034a, 1, 0x2},
		{0x20010509, 1, 0x2},
		{0x200106c8, 1, 0x2},
		{0x20010887, 1, 0x2},
		{0x20010a46, 1, 0x2},
		{0x20010c05, 1, 0x2},
		{0x20010dc4, 1, 0x2},
		{0x20010f83, 1, 0x2},
		{0x20011142, 1, 0x2},
		{0x20011301, 1, 0x2},
		{0x200114c0, 1, 0x2},
		{0x2001167f, 1, 0x2},
		{0x2001183e, 1, 0x2},
		{0x200119fd, 1, 0x2},
		{0x20011bbc, 1, 0x2},
		{0x20011d7b, 1, 0x2},
		{0x20011f3a, 1, 0x2},
		{0x200120f9, 1, 0x2},
		{0x200122b8, 1, 0x2},
		{0x20012477, 1, 0x2},
		{0x20012636, 1, 0x2},
		{0x200127f5, 1, 0x2},
		{0x200129b4, 1, 0x2},
		{0x20012b73, 1, 0x2},
		{0x20012d32, 1, 0x2},
		{0x20012ef1, 1, 0x2},
		{0x200130b0, 1, 0x2},
		{0x2001326f, 1, 0x2},
		{0x2001342e, 1, 0x2},
		{0x200135ed, 1, 0x2},
		{0x200137ac, 1, 0x2},
		{0x2001396b, 1, 0x2},
		{0x20013b2a, 1, 0x2},
		{0x20013ce9, 1, 0x2},
		{0x20013ea8, 1, 0x2},
		{0x20014067, 1, 0x2},
		{0x20014226, 1, 0x2},
		{0x200143e5, 1, 0x2},
		{0x200145a4, 1, 0x2},
		{0x20014763, 1, 0x2},
		{0x20014922, 1, 0x2},
		{0x20014ae1, 1, 0x2},
		{0x20014ca0, 1, 0x2},
		{0x20014e5f, 1, 0x2},
		{0x2001501e, 1, 0x2},
		{0x200151dd, 1, 0x2},
		{0x2001539c, 1, 0x2},
		{0x2001555b, 1, 0x2},
		{0x2001571a, 1, 0x2},
		{0x200158d9, 1, 0x2},
		{0x20015a98, 1, 0x2},
		{0x20015c57, 1, 0x2},
		{0x20015e16, 1, 0x2},
		{0x20015fd5, 1, 0x2},
		{0x20016194, 1, 0x2},
		{0x20016353, 1, 0x2},
		{0x20016512, 1, 0x2},
		{0x200166d1, 1, 0x2},
		{0x20016890, 1, 0x2},
		{0x20016a4f, 1, 0x2},
		{0x20016c0e, 1, 0x2},
		{0x20016dcd, 1, 0x2},
		{0x20016f8c, 1, 0x2},
		{0x2001714b, 1, 0x2},
		{0x2001730a, 1, 0x2},
		{0x200174c9, 1, 0x2},
		{0x20017688, 1, 0x2},
		{0x20017847, 1, 0x2},
		{0x20017a06, 1, 0x2},
		{0x20017bc5, 1, 0x2},
		{0x20017d84, 1, 0x2},
		{0x20017f43, 1, 0x2},
		{0x20018102, 1, 0x2},
		{0x200182c1, 1, 0x2},
		{0x20018480, 1, 0x2},
		{0x2001863f, 1, 0x2},
		{0x200187fe, 1, 0x2},
		{0x200189bd, 1, 0x2},
		{0x20018b7c, 1, 0x2},
		{0x20018d3b, 1, 0x2},
		{0x20018efa, 1, 0x2},
		{0x200190b9, 1, 0x2},
		{0x20019278, 1, 0x2},
		{0x20019437, 1, 0x2},
		{0x200195f6, 1, 0x2},
		{0x200197b5, 1, 0x2},
		{0x20019974, 1, 0x2},
		{0x20019b33, 1, 0x2},
		{0x20019cf2, 1, 0x2},
		{0x20019eb1, 1, 0x2},
		{0x2001a070, 1, 0x2},
		{0x2001a22f, 1, 0x2},
		{0x2001a3ee, 1, 0x2},
		{0x2001a5ad, 1, 0x2},
		{0x2001a76c, 1, 0x2},
		{0x2001a92b, 1, 0x2},
		{0x2001aaea, 1, 0x2},
		{0x2001aca9, 1, 0x2},
		{0x2001ae68, 1, 0x2},
		{0x2001b027, 1, 0x2},
		{0x2001b1e6, 1, 0x2},
		{0x2001b3a5, 1, 0x2},
		{0x2001b564, 1, 0x2},
		{0x2001b723, 1, 0x2},
		{0x2001b8e2, 1, 0x2},
		{0x2001baa1, 1, 0x2},
		{0x2001bc60, 1, 0x2},
		{0x2001be1f, 1, 0x2},
		{0x2001bfde, 1, 0x2},
		{0x2001c19d, 1, 0x2},
		{0x2001c35c, 1, 0x2},
		{0x2001c51b, 1, 0x2},
		{0x2001c6da, 1, 0x2},
		{0x2001c899, 1, 0x2},
		{0x2001ca58, 1, 0x2},
		{0x2001cc17, 1, 0x2},
		{0x2001cdd6, 1, 0x2},
		{0x2001cf95, 1, 0x2},
		{0x2001d154, 1, 0x2},
		{0x2001d313, 1, 0x2},
		{0x2001d4d2, 1, 0x2},
		{0x2001d691, 1, 0x2},
		{0x2001d850, 1, 0x2},
		{0x2001da0f, 1, 0x2},
		{0x2001dbce, 1, 0x2},
		{0x2001dd8d, 1, 0x2},
		{0x2001df4c, 1, 0x2},
		{0x2001e10b, 1, 0x2},
		{0x2001e2ca, 1, 0x2},
		{0x2001e489, 1, 0x2},
		{0x2001e648, 1, 0x2},
		{0x2001e807, 1, 0x2},
		{0x2001e9c6, 1, 0x2},
		{0x2001eb85, 1, 0x2},
		{0x2001ed44, 1, 0x2},
		{0x2001ef03, 1, 0x2},
		{0x2001f0c2, 1, 0x2},
		{0x2001f281, 1, 0x2},
		{0x2001f440, 1, 0x2},
		{0x2001f5ff, 1, 0x2},
		{0x2001f7be, 1, 0x2},
		{0x2001f97d, 1, 0x2},
		{0x2001fb3c, 1, 0x2},
		{0x2001fcfb, 1, 0x2},
		{0x2001feba, 1, 0x2},
		{0x20020079, 1, 0x2},
		{0x20020238, 1, 0x2},
		{0x200203f7, 1, 0x2},
		{0x200205b6, 1, 0x2},
		{0x20020775, 1, 0x2},
		{0x20020934, 1, 0x2},
		{0x20020af3, 1, 0x2},
		{0x20020cb2, 1, 0x2},
		{0x20020e71, 1, 0x2},
		{0x20021030, 1, 0x2},
		{0x200211ef, 1, 0x2},
		{0x200213ae, 1, 0x2},
		{0x2002156d, 1, 0x2},
		{0x2002172c, 1, 0x2},
		{0x200218eb, 1, 0x2},
		{0x20021aaa, 1, 0x2},
		{0x20021c69, 1, 0x2},
		{0x20021e28, 1, 0x2},
		{0x20021fe7, 1, 0x2},
		{0x200221a6, 1, 0x2},
		{0x20022365, 1, 0x2},
		{0x20022524, 1, 0x2},
		{0x200226e3, 1, 0x2},
		{0x200228a2, 1, 0x2},
		{0x20022a61, 1, 0x2},
		{0x20022c20, 1, 0x2},
		{0x20022ddf, 1, 0x2},
		{0x20022f9e, 1, 0x2},
		{0x2002315d, 1, 0x2},
		{0x2002331c, 1, 0x2},
		{0x200234db, 1, 0x2},
		{0x2002369a, 1, 0x2},
		{0x20023859, 1, 0x2},
		{0x20023a18, 1, 0x2},
		{0x20023bd7, 1, 0x2},
		{0x20023d96, 1, 0x2},
		{0x20023f55, 1, 0x2},
		{0x20024114, 1, 0x2},
		{0x200242d3, 1, 0x2},
		{0x20024492, 1, 0x2},
		{0x20024651, 1, 0x2},
		{0x20024810, 1, 0x2},
		{0x200249cf, 1, 0x2},
		{0x20024b8e, 1, 0x2},
		{0x20024d4d, 1, 0x2},
		{0x20024f0c, 1, 0x2},
		{0x200250cb, 1, 0x2},
		{0x2002528a, 1, 0x2},
		{0x20025449, 1, 0x2},
		{0x20025608, 1, 0x2},
		{0x200257c7, 1, 0x2},
		{0x20025986, 1, 0x2},
		{0x20025b45, 1, 0x2},
		{0x20025d04, 1, 0x2},
		{0x20025ec3, 1, 0x2},
		{0x20026082, 1, 0x2},
		{0x20026241, 1, 0x2},
		{0x20026400, 1, 0x2},
		{0x200265bf, 1, 0x2},
		{0x2002677e, 1, 0x2},
		{0x2002693d, 1, 0x2},
		{0x20026afc, 1, 0x2},
		{0x20026cbb, 1, 0x2},
		{0x20026e7a, 1, 0x2},
		{0x20027039, 1, 0x2},
		{0x200271f8, 1, 0x2},
		{0x200273b7, 1, 0x2},
		{0x20027576, 1, 0x2},
		{0x20027735, 1, 0x2},
		{0x200278f4, 1, 0x2},
		{0x20027ab3, 1, 0x2},
		{0x20027c72, 1, 0x2},
		{0x20027e31, 1, 0x2},
		{0x20027ff0, 1, 0x2},
		{0x200281af, 1, 0x2},
		{0x2002836e, 1, 0x2},
		{0x2002852d, 1, 0x2},
		{0x200286ec, 1, 0x2},
		{0x200288ab, 1, 0x2},
		{0x20028a6a, 1, 0x2},
		{0x20028c29, 1, 0x2},
		{0x2000cfc6, 1, 0x5},
		{0x2000d185, 1, 0x5},
		{0x2000d344, 1, 0x5},
		{0x2000d503, 1, 0x5},
		{0x2000d6c2, 1, 0x5},
		{0x2000d881, 1, 0x5},
		{0x2000da40, 1, 0x5},
		{0x2000dbff, 1, 0x5},
		{0x2000ddbe, 1, 0x5},
		{0x2000df7d, 1, 0x5},
		{0x2000e13c, 1, 0x5},
		{0x2000e2fb, 1, 0x5},
		{0x2000e4ba, 1, 0x5},
		{0x2000e679, 1, 0x5},
		{0x2000e838, 1, 0x5},
		{0x2000e9f7, 1, 0x5},
		{0x2000ebb6, 1, 0x5},
		{0x2000ed75, 1, 0x5},
		{0x2000ef34, 1, 0x5},
		{0x2000f0f3, 1, 0x5},
		{0x2000f2b2, 1, 0x5},
		{0x2000f471, 1, 0x5},
		{0x2000f630, 1, 0x5},
		{0x2000f7ef, 1, 0x5},
		{0x2000f9ae, 1, 0x5},
		{0x2000fb6d, 1, 0x5},
		{0x2000fd2c, 1, 0x5},
		{0x2000feeb, 1, 0x5},
		{0x200100aa, 1, 0x5},
		{0x20010269, 1, 0x5},
		{0x20010428, 1, 0x5},
		{0x200105e7, 1, 0x5},
		{0x200107a6, 1, 0x5},
		{0x20010965, 1, 0x5},
		{0x20010b24, 1, 0x5},
		{0x20010ce3, 1, 0x5},
		{0x20010ea2, 1, 0x5},
		{0x20011061, 1, 0x5},
		{0x20011220, 1, 0x5},
		{0x200113df, 1, 0x5},
		{0x2001159e, 1, 0x5},
		{0x2001175d, 1, 0x5},
		{0x2001191c, 1, 0x5},
		{0x20011adb, 1, 0x5},
		{0x20011c9a, 1, 0x5},
		{0x20011e59, 1, 0x5},
		{0x20012018, 1, 0x5},
		{0x200121d7, 1, 0x5},
		{0x20012396, 1, 0x5},
		{0x20012555, 1, 0x5},
		{0x20012714, 1, 0x5},
		{0x200128d3, 1, 0x5},
		{0x20012a92, 1, 0x5},
		{0x20012c51, 1, 0x5},
		{0x20012e10, 1, 0x5},
		{0x20012fcf, 1, 0x5},
		{0x2001318e, 1, 0x5},
		{0x2001334d, 1, 0x5},
		{0x2001350c, 1, 0x5},
		{0x200136cb, 1, 0x5},
		{0x2001388a, 1, 0x5},
		{0x20013a49, 1, 0x5},
		{0x20013c08, 1, 0x5},
		{0x20013dc7, 1, 0x5},
		{0x20013f86, 1, 0x5},
		{0x20014145, 1, 0x5},
		{0x20014304, 1, 0x5},
		{0x200144c3, 1, 0x5},
		{0x20014682, 1, 0x5},
		{0x20014841, 1, 0x5},
		{0x20014a00, 1, 0x5},
		{0x20014bbf, 1, 0x5},
		{0x20014d7e, 1, 0x5},
		{0x20014f3d, 1, 0x5},
		{0x200150fc, 1, 0x5},
		{0x200152bb, 1, 0x5},
		{0x2001547a, 1, 0x5},
		{0x20015639, 1, 0x5},
		{0x200157f8, 1, 0x5},
		{0x200159b7, 1, 0x5},
		{0x20015b76, 1, 0x5},
		{0x20015d35, 1, 0x5},
		{0x20015ef4, 1, 0x5},
		{0x200160b3, 1, 0x5},
		{0x20016272, 1, 0x5},
		{0x20016431, 1, 0x5},
		{0x200165f0, 1, 0x5},
		{0x200167af, 1, 0x5},
		{0x2001696e, 1, 0x5},
		{0x20016b2d, 1, 0x5},
		{0x20016cec, 1, 0x5},
		{0x20016eab, 1, 0x5},
		{0x2001706a, 1, 0x5},
		{0x20017229, 1, 0x5},
		{0x200173e8, 1, 0x5},
		{0x200175a7, 1, 0x5},
		{0x20017766, 1, 0x5},
		{0x20017925, 1, 0x5},
		{0x20017ae4, 1, 0x5},
		{0x20017ca3, 1, 0x5},
		{0x20017e62, 1, 0x5},
		{0x20018021, 1, 0x5},
		{0x200181e0, 1, 0x5},
		{0x2001839f, 1, 0x5},
		{0x2001855e, 1, 0x5},
		{0x2001871d, 1, 0x5},
		{0x200188dc, 1, 0x5},
		{0x20018a9b, 1, 0x5},
		{0x20018c5a, 1, 0x5},
		{0x20018e19, 1, 0x5},
		{0x20018fd8, 1, 0x5},
		{0x20019197, 1, 0x5},
		{0x20019356, 1, 0x5},
		{0x20019515, 1, 0x5},
		{0x200196d4, 1, 0x5},
		{0x20019893, 1, 0x5},
		{0x20019a52, 1, 0x5},
		{0x20019c11, 1, 0x5},
		{0x20019dd0, 1, 0x5},
		{0x20019f8f, 1, 0x5},
		{0x2001a14e, 1, 0x5},
		{0x2001a30d, 1, 0x5},
		{0x2001a4cc, 1, 0x5},
		{0x2001a68b, 1, 0x5},
		{0x2001a84a, 1, 0x5},
		{0x2001aa09, 1, 0x5},
		{0x2001abc8, 1, 0x5},
		{0x2001ad87, 1, 0x5},
		{0x2001af46, 1, 0x5},
		{0x2001b105, 1, 0x5},
		{0x2001b2c4, 1, 0x5},
		{0x2001b483, 1, 0x5},
		{0x2001b642, 1, 0x5},
		{0x2001b801, 1, 0x5},
		{0x2001b9c0, 1, 0x5},
		{0x2001bb7f, 1, 0x5},
		{0x2001bd3e, 1, 0x5},
		{0x2001befd, 1, 0x5},
		{0x2001c0bc, 1, 0x5},
		{0x2001c27b, 1, 0x5},
		{0x2001c43a, 1, 0x5},
		{0x2001c5f9, 1, 0x5},
		{0x2001c7b8, 1, 0x5},
		{0x2001c977, 1, 0x5},
		{0x2001cb36, 1, 0x5},
		{0x2001ccf5, 1, 0x5},
		{0x2001ceb4, 1, 0x5},
		{0x2001d073, 1, 0x5},
		{0x2001d232, 1, 0x5},
		{0x2001d3f1, 1, 0x5},
		{0x2001d5b0, 1, 0x5},
		{0x2001d76f, 1, 0x5},
		{0x2001d92e, 1, 0x5},
		{0x2001daed, 1, 0x5},
		{0x2001dcac, 1, 0x5},
		{0x2001de6b, 1, 0x5},
		{0x2001e02a, 1, 0x5},
		{0x2001e1e9, 1, 0x5},
		{0x2001e3a8, 1, 0x5},
		{0x2001e567, 1, 0x5},
		{0x2001e726, 1, 0x5},
		{0x2001e8e5, 1, 0x5},
		{0x2001eaa4, 1, 0x5},
		{0x2001ec63, 1, 0x5},
		{0x2001ee22, 1, 0x5},
		{0x2001efe1, 1, 0x5},
		{0x2001f1a0, 1, 0x5},
		{0x2001f35f, 1, 0x5},
		{0x2001f51e, 1, 0x5},
		{0x2001f6dd, 1, 0x5},
		{0x2001f89c, 1, 0x5},
		{0x2001fa5b, 1, 0x5},
		{0x2001fc1a, 1, 0x5},
		{0x2001fdd9, 1, 0x5},
		{0x2001ff98, 1, 0x5},
		{0x20020157, 1, 0x5},
		{0x20020316, 1, 0x5},
		{0x200204d5, 1, 0x5},
		{0x20020694, 1, 0x5},
		{0x20020853, 1, 0x5},
		{0x20020a12, 1, 0x5},
		{0x20020bd1, 1, 0x5},
		{0x20020d90, 1, 0x5},
		{0x20020f4f, 1, 0x5},
		{0x2002110e, 1, 0x5},
		{0x200212cd, 1, 0x5},
		{0x2002148c, 1, 0x5},
		{0x2002164b, 1, 0x5},
		{0x2002180a, 1, 0x5},
		{0x200219c9, 1, 0x5},
		{0x20021b88, 1, 0x5},
		{0x20021d47, 1, 0x5},
		{0x20021f06, 1, 0x5},
		{0x200220c5, 1, 0x5},
		{0x20022284, 1, 0x5},
		{0x20022443, 1, 0x5},
		{0x20022602, 1, 0x5},
		{0x200227c1, 1, 0x5},
		{0x20022980, 1, 0x5},
		{0x20022b3f, 1, 0x5},
		{0x20022cfe, 1, 0x5},
		{0x20022ebd, 1, 0x5},
		{0x2002307c, 1, 0x5},
		{0x2002323b, 1, 0x5},
		{0x200233fa, 1, 0x5},
		{0x200235b9, 1, 0x5},
		{0x20023778, 1, 0x5},
		{0x20023937, 1, 0x5},
		{0x20023af6, 1, 0x5},
		{0x20023cb5, 1, 0x5},
		{0x20023e74, 1, 0x5},
		{0x20024033, 1, 0x5},
		{0x200241f2, 1, 0x5},
		{0x200243b1, 1, 0x5},
		{0x20024570, 1, 0x5},
		{0x2002472f, 1, 0x5},
		{0x200248ee, 1, 0x5},
		{0x20024aad, 1, 0x5},
		{0x20024c6c, 1, 0x5},
		{0x20024e2b, 1, 0x5},
		{0x20024fea, 1, 0x5},
		{0x200251a9, 1, 0x5},
		{0x20025368, 1, 0x5},
		{0x20025527, 1, 0x5},
		{0x200256e6, 1, 0x5},
		{0x200258a5, 1, 0x5},
		{0x20025a64, 1, 0x5},
		{0x20025c23, 1, 0x5},
		{0x20025de2, 1, 0x5},
		{0x20025fa1, 1, 0x5},
		{0x20026160, 1, 0x5},
		{0x2002631f, 1, 0x5},
		{0x200264de, 1, 0x5},
		{0x2002669d, 1, 0x5},
		{0x2002685c, 1, 0x5},
		{0x20026a1b, 1, 0x5},
		{0x20026bda, 1, 0x5},
		{0x20026d99, 1, 0x5},
		{0x20026f58, 1, 0x5},
		{0x20027117, 1, 0x5},
		{0x200272d6, 1, 0x5},
		{0x20027495, 1, 0x5},
		{0x20027654, 1, 0x5},
		{0x20027813, 1, 0x5},
		{0x200279d2, 1, 0x5},
		{0x20027b91, 1, 0x5},
		{0x20027d50, 1, 0x5},
		{0x20027f0f, 1, 0x5},
		{0x200280ce, 1, 0x5},
		{0x2002828d, 1, 0x5},
		{0x2002844c, 1, 0x5},
		{0x2002860b, 1, 0x5},
		{0x200287ca, 1, 0x5},
		{0x20028989, 1, 0x5},
		{0x20028b48, 1, 0x5},
		{0x20028d07, 1, 0x5},
		{0x2000cfd3, 4, 0x5},
		{0x2000d192, 4, 0x5},
		{0x2000d351, 4, 0x5},
		{0x2000d510, 4, 0x5},
		{0x2000d6cf, 4, 0x5},
		{0x2000d88e, 4, 0x5},
		{0x2000da4d, 4, 0x5},
		{0x2000dc0c, 4, 0x5},
		{0x2000ddcb, 4, 0x5},
		{0x2000df8a, 4, 0x5},
		{0x2000e149, 4, 0x5},
		{0x2000e308, 4, 0x5},
		{0x2000e4c7, 4, 0x5},
		{0x2000e686, 4, 0x5},
		{0x2000e845, 4, 0x5},
		{0x2000ea04, 4, 0x5},
		{0x2000ebc3, 4, 0x5},
		{0x2000ed82, 4, 0x5},
		{0x2000ef41, 4, 0x5},
		{0x2000f100, 4, 0x5},
		{0x2000f2bf, 4, 0x5},
		{0x2000f47e, 4, 0x5},
		{0x2000f63d, 4, 0x5},
		{0x2000f7fc, 4, 0x5},
		{0x2000f9bb, 4, 0x5},
		{0x2000fb7a, 4, 0x5},
		{0x2000fd39, 4, 0x5},
		{0x2000fef8, 4, 0x5},
		{0x200100b7, 4, 0x5},
		{0x20010276, 4, 0x5},
		{0x20010435, 4, 0x5},
		{0x200105f4, 4, 0x5},
		{0x200107b3, 4, 0x5},
		{0x20010972, 4, 0x5},
		{0x20010b31, 4, 0x5},
		{0x20010cf0, 4, 0x5},
		{0x20010eaf, 4, 0x5},
		{0x2001106e, 4, 0x5},
		{0x2001122d, 4, 0x5},
		{0x200113ec, 4, 0x5},
		{0x200115ab, 4, 0x5},
		{0x2001176a, 4, 0x5},
		{0x20011929, 4, 0x5},
		{0x20011ae8, 4, 0x5},
		{0x20011ca7, 4, 0x5},
		{0x20011e66, 4, 0x5},
		{0x20012025, 4, 0x5},
		{0x200121e4, 4, 0x5},
		{0x200123a3, 4, 0x5},
		{0x20012562, 4, 0x5},
		{0x20012721, 4, 0x5},
		{0x200128e0, 4, 0x5},
		{0x20012a9f, 4, 0x5},
		{0x20012c5e, 4, 0x5},
		{0x20012e1d, 4, 0x5},
		{0x20012fdc, 4, 0x5},
		{0x2001319b, 4, 0x5},
		{0x2001335a, 4, 0x5},
		{0x20013519, 4, 0x5},
		{0x200136d8, 4, 0x5},
		{0x20013897, 4, 0x5},
		{0x20013a56, 4, 0x5},
		{0x20013c15, 4, 0x5},
		{0x20013dd4, 4, 0x5},
		{0x20013f93, 4, 0x5},
		{0x20014152, 4, 0x5},
		{0x20014311, 4, 0x5},
		{0x200144d0, 4, 0x5},
		{0x2001468f, 4, 0x5},
		{0x2001484e, 4, 0x5},
		{0x20014a0d, 4, 0x5},
		{0x20014bcc, 4, 0x5},
		{0x20014d8b, 4, 0x5},
		{0x20014f4a, 4, 0x5},
		{0x20015109, 4, 0x5},
		{0x200152c8, 4, 0x5},
		{0x20015487, 4, 0x5},
		{0x20015646, 4, 0x5},
		{0x20015805, 4, 0x5},
		{0x200159c4, 4, 0x5},
		{0x20015b83, 4, 0x5},
		{0x20015d42, 4, 0x5},
		{0x20015f01, 4, 0x5},
		{0x200160c0, 4, 0x5},
		{0x2001627f, 4, 0x5},
		{0x2001643e, 4, 0x5},
		{0x200165fd, 4, 0x5},
		{0x200167bc, 4, 0x5},
		{0x2001697b, 4, 0x5},
		{0x20016b3a, 4, 0x5},
		{0x20016cf9, 4, 0x5},
		{0x20016eb8, 4, 0x5},
		{0x20017077, 4, 0x5},
		{0x20017236, 4, 0x5},
		{0x200173f5, 4, 0x5},
		{0x200175b4, 4, 0x5},
		{0x20017773, 4, 0x5},
		{0x20017932, 4, 0x5},
		{0x20017af1, 4, 0x5},
		{0x20017cb0, 4, 0x5},
		{0x20017e6f, 4, 0x5},
		{0x2001802e, 4, 0x5},
		{0x200181ed, 4, 0x5},
		{0x200183ac, 4, 0x5},
		{0x2001856b, 4, 0x5},
		{0x2001872a, 4, 0x5},
		{0x200188e9, 4, 0x5},
		{0x20018aa8, 4, 0x5},
		{0x20018c67, 4, 0x5},
		{0x20018e26, 4, 0x5},
		{0x20018fe5, 4, 0x5},
		{0x200191a4, 4, 0x5},
		{0x20019363, 4, 0x5},
		{0x20019522, 4, 0x5},
		{0x200196e1, 4, 0x5},
		{0x200198a0, 4, 0x5},
		{0x20019a5f, 4, 0x5},
		{0x20019c1e, 4, 0x5},
		{0x20019ddd, 4, 0x5},
		{0x20019f9c, 4, 0x5},
		{0x2001a15b, 4, 0x5},
		{0x2001a31a, 4, 0x5},
		{0x2001a4d9, 4, 0x5},
		{0x2001a698, 4, 0x5},
		{0x2001a857, 4, 0x5},
		{0x2001aa16, 4, 0x5},
		{0x2001abd5, 4, 0x5},
		{0x2001ad94, 4, 0x5},
		{0x2001af53, 4, 0x5},
		{0x2001b112, 4, 0x5},
		{0x2001b2d1, 4, 0x5},
		{0x2001b490, 4, 0x5},
		{0x2001b64f, 4, 0x5},
		{0x2001b80e, 4, 0x5},
		{0x2001b9cd, 4, 0x5},
		{0x2001bb8c, 4, 0x5},
		{0x2001bd4b, 4, 0x5},
		{0x2001bf0a, 4, 0x5},
		{0x2001c0c9, 4, 0x5},
		{0x2001c288, 4, 0x5},
		{0x2001c447, 4, 0x5},
		{0x2001c606, 4, 0x5},
		{0x2001c7c5, 4, 0x5},
		{0x2001c984, 4, 0x5},
		{0x2001cb43, 4, 0x5},
		{0x2001cd02, 4, 0x5},
		{0x2001cec1, 4, 0x5},
		{0x2001d080, 4, 0x5},
		{0x2001d23f, 4, 0x5},
		{0x2001d3fe, 4, 0x5},
		{0x2001d5bd, 4, 0x5},
		{0x2001d77c, 4, 0x5},
		{0x2001d93b, 4, 0x5},
		{0x2001dafa, 4, 0x5},
		{0x2001dcb9, 4, 0x5},
		{0x2001de78, 4, 0x5},
		{0x2001e037, 4, 0x5},
		{0x2001e1f6, 4, 0x5},
		{0x2001e3b5, 4, 0x5},
		{0x2001e574, 4, 0x5},
		{0x2001e733, 4, 0x5},
		{0x2001e8f2, 4, 0x5},
		{0x2001eab1, 4, 0x5},
		{0x2001ec70, 4, 0x5},
		{0x2001ee2f, 4, 0x5},
		{0x2001efee, 4, 0x5},
		{0x2001f1ad, 4, 0x5},
		{0x2001f36c, 4, 0x5},
		{0x2001f52b, 4, 0x5},
		{0x2001f6ea, 4, 0x5},
		{0x2001f8a9, 4, 0x5},
		{0x2001fa68, 4, 0x5},
		{0x2001fc27, 4, 0x5},
		{0x2001fde6, 4, 0x5},
		{0x2001ffa5, 4, 0x5},
		{0x20020164, 4, 0x5},
		{0x20020323, 4, 0x5},
		{0x200204e2, 4, 0x5},
		{0x200206a1, 4, 0x5},
		{0x20020860, 4, 0x5},
		{0x20020a1f, 4, 0x5},
		{0x20020bde, 4, 0x5},
		{0x20020d9d, 4, 0x5},
		{0x20020f5c, 4, 0x5},
		{0x2002111b, 4, 0x5},
		{0x200212da, 4, 0x5},
		{0x20021499, 4, 0x5},
		{0x20021658, 4, 0x5},
		{0x20021817, 4, 0x5},
		{0x200219d6, 4, 0x5},
		{0x20021b95, 4, 0x5},
		{0x20021d54, 4, 0x5},
		{0x20021f13, 4, 0x5},
		{0x200220d2, 4, 0x5},
		{0x20022291, 4, 0x5},
		{0x20022450, 4, 0x5},
		{0x2002260f, 4, 0x5},
		{0x200227ce, 4, 0x5},
		{0x2002298d, 4, 0x5},
		{0x20022b4c, 4, 0x5},
		{0x20022d0b, 4, 0x5},
		{0x20022eca, 4, 0x5},
		{0x20023089, 4, 0x5},
		{0x20023248, 4, 0x5},
		{0x20023407, 4, 0x5},
		{0x200235c6, 4, 0x5},
		{0x20023785, 4, 0x5},
		{0x20023944, 4, 0x5},
		{0x20023b03, 4, 0x5},
		{0x20023cc2, 4, 0x5},
		{0x20023e81, 4, 0x5},
		{0x20024040, 4, 0x5},
		{0x200241ff, 4, 0x5},
		{0x200243be, 4, 0x5},
		{0x2002457d, 4, 0x5},
		{0x2002473c, 4, 0x5},
		{0x200248fb, 4, 0x5},
		{0x20024aba, 4, 0x5},
		{0x20024c79, 4, 0x5},
		{0x20024e38, 4, 0x5},
		{0x20024ff7, 4, 0x5},
		{0x200251b6, 4, 0x5},
		{0x20025375, 4, 0x5},
		{0x20025534, 4, 0x5},
		{0x200256f3, 4, 0x5},
		{0x200258b2, 4, 0x5},
		{0x20025a71, 4, 0x5},
		{0x20025c30, 4, 0x5},
		{0x20025def, 4, 0x5},
		{0x20025fae, 4, 0x5},
		{0x2002616d, 4, 0x5},
		{0x2002632c, 4, 0x5},
		{0x200264eb, 4, 0x5},
		{0x200266aa, 4, 0x5},
		{0x20026869, 4, 0x5},
		{0x20026a28, 4, 0x5},
		{0x20026be7, 4, 0x5},
		{0x20026da6, 4, 0x5},
		{0x20026f65, 4, 0x5},
		{0x20027124, 4, 0x5},
		{0x200272e3, 4, 0x5},
		{0x200274a2, 4, 0x5},
		{0x20027661, 4, 0x5},
		{0x20027820, 4, 0x5},
		{0x200279df, 4, 0x5},
		{0x20027b9e, 4, 0x5},
		{0x20027d5d, 4, 0x5},
		{0x20027f1c, 4, 0x5},
		{0x200280db, 4, 0x5},
		{0x2002829a, 4, 0x5},
		{0x20028459, 4, 0x5},
		{0x20028618, 4, 0x5},
		{0x200287d7, 4, 0x5},
		{0x20028996, 4, 0x5},
		{0x20028b55, 4, 0x5},
		{0x20028d14, 4, 0x5},
		{0x2000cee9, 4, 0x3},
		{0x2000d0a8, 4, 0x3},
		{0x2000d267, 4, 0x3},
		{0x2000d426, 4, 0x3},
		{0x2000d5e5, 4, 0x3},
		{0x2000d7a4, 4, 0x3},
		{0x2000d963, 4, 0x3},
		{0x2000db22, 4, 0x3},
		{0x2000dce1, 4, 0x3},
		{0x2000dea0, 4, 0x3},
		{0x2000e05f, 4, 0x3},
		{0x2000e21e, 4, 0x3},
		{0x2000e3dd, 4, 0x3},
		{0x2000e59c, 4, 0x3},
		{0x2000e75b, 4, 0x3},
		{0x2000e91a, 4, 0x3},
		{0x2000ead9, 4, 0x3},
		{0x2000ec98, 4, 0x3},
		{0x2000ee57, 4, 0x3},
		{0x2000f016, 4, 0x3},
		{0x2000f1d5, 4, 0x3},
		{0x2000f394, 4, 0x3},
		{0x2000f553, 4, 0x3},
		{0x2000f712, 4, 0x3},
		{0x2000f8d1, 4, 0x3},
		{0x2000fa90, 4, 0x3},
		{0x2000fc4f, 4, 0x3},
		{0x2000fe0e, 4, 0x3},
		{0x2000ffcd, 4, 0x3},
		{0x2001018c, 4, 0x3},
		{0x2001034b, 4, 0x3},
		{0x2001050a, 4, 0x3},
		{0x200106c9, 4, 0x3},
		{0x20010888, 4, 0x3},
		{0x20010a47, 4, 0x3},
		{0x20010c06, 4, 0x3},
		{0x20010dc5, 4, 0x3},
		{0x20010f84, 4, 0x3},
		{0x20011143, 4, 0x3},
		{0x20011302, 4, 0x3},
		{0x200114c1, 4, 0x3},
		{0x20011680, 4, 0x3},
		{0x2001183f, 4, 0x3},
		{0x200119fe, 4, 0x3},
		{0x20011bbd, 4, 0x3},
		{0x20011d7c, 4, 0x3},
		{0x20011f3b, 4, 0x3},
		{0x200120fa, 4, 0x3},
		{0x200122b9, 4, 0x3},
		{0x20012478, 4, 0x3},
		{0x20012637, 4, 0x3},
		{0x200127f6, 4, 0x3},
		{0x200129b5, 4, 0x3},
		{0x20012b74, 4, 0x3},
		{0x20012d33, 4, 0x3},
		{0x20012ef2, 4, 0x3},
		{0x200130b1, 4, 0x3},
		{0x20013270, 4, 0x3},
		{0x2001342f, 4, 0x3},
		{0x200135ee, 4, 0x3},
		{0x200137ad, 4, 0x3},
		{0x2001396c, 4, 0x3},
		{0x20013b2b, 4, 0x3},
		{0x20013cea, 4, 0x3},
		{0x20013ea9, 4, 0x3},
		{0x20014068, 4, 0x3},
		{0x20014227, 4, 0x3},
		{0x200143e6, 4, 0x3},
		{0x200145a5, 4, 0x3},
		{0x20014764, 4, 0x3},
		{0x20014923, 4, 0x3},
		{0x20014ae2, 4, 0x3},
		{0x20014ca1, 4, 0x3},
		{0x20014e60, 4, 0x3},
		{0x2001501f, 4, 0x3},
		{0x200151de, 4, 0x3},
		{0x2001539d, 4, 0x3},
		{0x2001555c, 4, 0x3},
		{0x2001571b, 4, 0x3},
		{0x200158da, 4, 0x3},
		{0x20015a99, 4, 0x3},
		{0x20015c58, 4, 0x3},
		{0x20015e17, 4, 0x3},
		{0x20015fd6, 4, 0x3},
		{0x20016195, 4, 0x3},
		{0x20016354, 4, 0x3},
		{0x20016513, 4, 0x3},
		{0x200166d2, 4, 0x3},
		{0x20016891, 4, 0x3},
		{0x20016a50, 4, 0x3},
		{0x20016c0f, 4, 0x3},
		{0x20016dce, 4, 0x3},
		{0x20016f8d, 4, 0x3},
		{0x2001714c, 4, 0x3},
		{0x2001730b, 4, 0x3},
		{0x200174ca, 4, 0x3},
		{0x20017689, 4, 0x3},
		{0x20017848, 4, 0x3},
		{0x20017a07, 4, 0x3},
		{0x20017bc6, 4, 0x3},
		{0x20017d85, 4, 0x3},
		{0x20017f44, 4, 0x3},
		{0x20018103, 4, 0x3},
		{0x200182c2, 4, 0x3},
		{0x20018481, 4, 0x3},
		{0x20018640, 4, 0x3},
		{0x200187ff, 4, 0x3},
		{0x200189be, 4, 0x3},
		{0x20018b7d, 4, 0x3},
		{0x20018d3c, 4, 0x3},
		{0x20018efb, 4, 0x3},
		{0x200190ba, 4, 0x3},
		{0x20019279, 4, 0x3},
		{0x20019438, 4, 0x3},
		{0x200195f7, 4, 0x3},
		{0x200197b6, 4, 0x3},
		{0x20019975, 4, 0x3},
		{0x20019b34, 4, 0x3},
		{0x20019cf3, 4, 0x3},
		{0x20019eb2, 4, 0x3},
		{0x2001a071, 4, 0x3},
		{0x2001a230, 4, 0x3},
		{0x2001a3ef, 4, 0x3},
		{0x2001a5ae, 4, 0x3},
		{0x2001a76d, 4, 0x3},
		{0x2001a92c, 4, 0x3},
		{0x2001aaeb, 4, 0x3},
		{0x2001acaa, 4, 0x3},
		{0x2001ae69, 4, 0x3},
		{0x2001b028, 4, 0x3},
		{0x2001b1e7, 4, 0x3},
		{0x2001b3a6, 4, 0x3},
		{0x2001b565, 4, 0x3},
		{0x2001b724, 4, 0x3},
		{0x2001b8e3, 4, 0x3},
		{0x2001baa2, 4, 0x3},
		{0x2001bc61, 4, 0x3},
		{0x2001be20, 4, 0x3},
		{0x2001bfdf, 4, 0x3},
		{0x2001c19e, 4, 0x3},
		{0x2001c35d, 4, 0x3},
		{0x2001c51c, 4, 0x3},
		{0x2001c6db, 4, 0x3},
		{0x2001c89a, 4, 0x3},
		{0x2001ca59, 4, 0x3},
		{0x2001cc18, 4, 0x3},
		{0x2001cdd7, 4, 0x3},
		{0x2001cf96, 4, 0x3},
		{0x2001d155, 4, 0x3},
		{0x2001d314, 4, 0x3},
		{0x2001d4d3, 4, 0x3},
		{0x2001d692, 4, 0x3},
		{0x2001d851, 4, 0x3},
		{0x2001da10, 4, 0x3},
		{0x2001dbcf, 4, 0x3},
		{0x2001dd8e, 4, 0x3},
		{0x2001df4d, 4, 0x3},
		{0x2001e10c, 4, 0x3},
		{0x2001e2cb, 4, 0x3},
		{0x2001e48a, 4, 0x3},
		{0x2001e649, 4, 0x3},
		{0x2001e808, 4, 0x3},
		{0x2001e9c7, 4, 0x3},
		{0x2001eb86, 4, 0x3},
		{0x2001ed45, 4, 0x3},
		{0x2001ef04, 4, 0x3},
		{0x2001f0c3, 4, 0x3},
		{0x2001f282, 4, 0x3},
		{0x2001f441, 4, 0x3},
		{0x2001f600, 4, 0x3},
		{0x2001f7bf, 4, 0x3},
		{0x2001f97e, 4, 0x3},
		{0x2001fb3d, 4, 0x3},
		{0x2001fcfc, 4, 0x3},
		{0x2001febb, 4, 0x3},
		{0x2002007a, 4, 0x3},
		{0x20020239, 4, 0x3},
		{0x200203f8, 4, 0x3},
		{0x200205b7, 4, 0x3},
		{0x20020776, 4, 0x3},
		{0x20020935, 4, 0x3},
		{0x20020af4, 4, 0x3},
		{0x20020cb3, 4, 0x3},
		{0x20020e72, 4, 0x3},
		{0x20021031, 4, 0x3},
		{0x200211f0, 4, 0x3},
		{0x200213af, 4, 0x3},
		{0x2002156e, 4, 0x3},
		{0x2002172d, 4, 0x3},
		{0x200218ec, 4, 0x3},
		{0x20021aab, 4, 0x3},
		{0x20021c6a, 4, 0x3},
		{0x20021e29, 4, 0x3},
		{0x20021fe8, 4, 0x3},
		{0x200221a7, 4, 0x3},
		{0x20022366, 4, 0x3},
		{0x20022525, 4, 0x3},
		{0x200226e4, 4, 0x3},
		{0x200228a3, 4, 0x3},
		{0x20022a62, 4, 0x3},
		{0x20022c21, 4, 0x3},
		{0x20022de0, 4, 0x3},
		{0x20022f9f, 4, 0x3},
		{0x2002315e, 4, 0x3},
		{0x2002331d, 4, 0x3},
		{0x200234dc, 4, 0x3},
		{0x2002369b, 4, 0x3},
		{0x2002385a, 4, 0x3},
		{0x20023a19, 4, 0x3},
		{0x20023bd8, 4, 0x3},
		{0x20023d97, 4, 0x3},
		{0x20023f56, 4, 0x3},
		{0x20024115, 4, 0x3},
		{0x200242d4, 4, 0x3},
		{0x20024493, 4, 0x3},
		{0x20024652, 4, 0x3},
		{0x20024811, 4, 0x3},
		{0x200249d0, 4, 0x3},
		{0x20024b8f, 4, 0x3},
		{0x20024d4e, 4, 0x3},
		{0x20024f0d, 4, 0x3},
		{0x200250cc, 4, 0x3},
		{0x2002528b, 4, 0x3},
		{0x2002544a, 4, 0x3},
		{0x20025609, 4, 0x3},
		{0x200257c8, 4, 0x3},
		{0x20025987, 4, 0x3},
		{0x20025b46, 4, 0x3},
		{0x20025d05, 4, 0x3},
		{0x20025ec4, 4, 0x3},
		{0x20026083, 4, 0x3},
		{0x20026242, 4, 0x3},
		{0x20026401, 4, 0x3},
		{0x200265c0, 4, 0x3},
		{0x2002677f, 4, 0x3},
		{0x2002693e, 4, 0x3},
		{0x20026afd, 4, 0x3},
		{0x20026cbc, 4, 0x3},
		{0x20026e7b, 4, 0x3},
		{0x2002703a, 4, 0x3},
		{0x200271f9, 4, 0x3},
		{0x200273b8, 4, 0x3},
		{0x20027577, 4, 0x3},
		{0x20027736, 4, 0x3},
		{0x200278f5, 4, 0x3},
		{0x20027ab4, 4, 0x3},
		{0x20027c73, 4, 0x3},
		{0x20027e32, 4, 0x3},
		{0x20027ff1, 4, 0x3},
		{0x200281b0, 4, 0x3},
		{0x2002836f, 4, 0x3},
		{0x2002852e, 4, 0x3},
		{0x200286ed, 4, 0x3},
		{0x200288ac, 4, 0x3},
		{0x20028a6b, 4, 0x3},
		{0x20028c2a, 4, 0x3},
		{0x2000cef6, 4, 0x2},
		{0x2000d0b5, 4, 0x2},
		{0x2000d274, 4, 0x2},
		{0x2000d433, 4, 0x2},
		{0x2000d5f2, 4, 0x2},
		{0x2000d7b1, 4, 0x2},
		{0x2000d970, 4, 0x2},
		{0x2000db2f, 4, 0x2},
		{0x2000dcee, 4, 0x2},
		{0x2000dead, 4, 0x2},
		{0x2000e06c, 4, 0x2},
		{0x2000e22b, 4, 0x2},
		{0x2000e3ea, 4, 0x2},
		{0x2000e5a9, 4, 0x2},
		{0x2000e768, 4, 0x2},
		{0x2000e927, 4, 0x2},
		{0x2000eae6, 4, 0x2},
		{0x2000eca5, 4, 0x2},
		{0x2000ee64, 4, 0x2},
		{0x2000f023, 4, 0x2},
		{0x2000f1e2, 4, 0x2},
		{0x2000f3a1, 4, 0x2},
		{0x2000f560, 4, 0x2},
		{0x2000f71f, 4, 0x2},
		{0x2000f8de, 4, 0x2},
		{0x2000fa9d, 4, 0x2},
		{0x2000fc5c, 4, 0x2},
		{0x2000fe1b, 4, 0x2},
		{0x2000ffda, 4, 0x2},
		{0x20010199, 4, 0x2},
		{0x20010358, 4, 0x2},
		{0x20010517, 4, 0x2},
		{0x200106d6, 4, 0x2},
		{0x20010895, 4, 0x2},
		{0x20010a54, 4, 0x2},
		{0x20010c13, 4, 0x2},
		{0x20010dd2, 4, 0x2},
		{0x20010f91, 4, 0x2},
		{0x20011150, 4, 0x2},
		{0x2001130f, 4, 0x2},
		{0x200114ce, 4, 0x2},
		{0x2001168d, 4, 0x2},
		{0x2001184c, 4, 0x2},
		{0x20011a0b, 4, 0x2},
		{0x20011bca, 4, 0x2},
		{0x20011d89, 4, 0x2},
		{0x20011f48, 4, 0x2},
		{0x20012107, 4, 0x2},
		{0x200122c6, 4, 0x2},
		{0x20012485, 4, 0x2},
		{0x20012644, 4, 0x2},
		{0x20012803, 4, 0x2},
		{0x200129c2, 4, 0x2},
		{0x20012b81, 4, 0x2},
		{0x20012d40, 4, 0x2},
		{0x20012eff, 4, 0x2},
		{0x200130be, 4, 0x2},
		{0x2001327d, 4, 0x2},
		{0x2001343c, 4, 0x2},
		{0x200135fb, 4, 0x2},
		{0x200137ba, 4, 0x2},
		{0x20013979, 4, 0x2},
		{0x20013b38, 4, 0x2},
		{0x20013cf7, 4, 0x2},
		{0x20013eb6, 4, 0x2},
		{0x20014075, 4, 0x2},
		{0x20014234, 4, 0x2},
		{0x200143f3, 4, 0x2},
		{0x200145b2, 4, 0x2},
		{0x20014771, 4, 0x2},
		{0x20014930, 4, 0x2},
		{0x20014aef, 4, 0x2},
		{0x20014cae, 4, 0x2},
		{0x20014e6d, 4, 0x2},
		{0x2001502c, 4, 0x2},
		{0x200151eb, 4, 0x2},
		{0x200153aa, 4, 0x2},
		{0x20015569, 4, 0x2},
		{0x20015728, 4, 0x2},
		{0x200158e7, 4, 0x2},
		{0x20015aa6, 4, 0x2},
		{0x20015c65, 4, 0x2},
		{0x20015e24, 4, 0x2},
		{0x20015fe3, 4, 0x2},
		{0x200161a2, 4, 0x2},
		{0x20016361, 4, 0x2},
		{0x20016520, 4, 0x2},
		{0x200166df, 4, 0x2},
		{0x2001689e, 4, 0x2},
		{0x20016a5d, 4, 0x2},
		{0x20016c1c, 4, 0x2},
		{0x20016ddb, 4, 0x2},
		{0x20016f9a, 4, 0x2},
		{0x20017159, 4, 0x2},
		{0x20017318, 4, 0x2},
		{0x200174d7, 4, 0x2},
		{0x20017696, 4, 0x2},
		{0x20017855, 4, 0x2},
		{0x20017a14, 4, 0x2},
		{0x20017bd3, 4, 0x2},
		{0x20017d92, 4, 0x2},
		{0x20017f51, 4, 0x2},
		{0x20018110, 4, 0x2},
		{0x200182cf, 4, 0x2},
		{0x2001848e, 4, 0x2},
		{0x2001864d, 4, 0x2},
		{0x2001880c, 4, 0x2},
		{0x200189cb, 4, 0x2},
		{0x20018b8a, 4, 0x2},
		{0x20018d49, 4, 0x2},
		{0x20018f08, 4, 0x2},
		{0x200190c7, 4, 0x2},
		{0x20019286, 4, 0x2},
		{0x20019445, 4, 0x2},
		{0x20019604, 4, 0x2},
		{0x200197c3, 4, 0x2},
		{0x20019982, 4, 0x2},
		{0x20019b41, 4, 0x2},
		{0x20019d00, 4, 0x2},
		{0x20019ebf, 4, 0x2},
		{0x2001a07e, 4, 0x2},
		{0x2001a23d, 4, 0x2},
		{0x2001a3fc, 4, 0x2},
		{0x2001a5bb, 4, 0x2},
		{0x2001a77a, 4, 0x2},
		{0x2001a939, 4, 0x2},
		{0x2001aaf8, 4, 0x2},
		{0x2001acb7, 4, 0x2},
		{0x2001ae76, 4, 0x2},
		{0x2001b035, 4, 0x2},
		{0x2001b1f4, 4, 0x2},
		{0x2001b3b3, 4, 0x2},
		{0x2001b572, 4, 0x2},
		{0x2001b731, 4, 0x2},
		{0x2001b8f0, 4, 0x2},
		{0x2001baaf, 4, 0x2},
		{0x2001bc6e, 4, 0x2},
		{0x2001be2d, 4, 0x2},
		{0x2001bfec, 4, 0x2},
		{0x2001c1ab, 4, 0x2},
		{0x2001c36a, 4, 0x2},
		{0x2001c529, 4, 0x2},
		{0x2001c6e8, 4, 0x2},
		{0x2001c8a7, 4, 0x2},
		{0x2001ca66, 4, 0x2},
		{0x2001cc25, 4, 0x2},
		{0x2001cde4, 4, 0x2},
		{0x2001cfa3, 4, 0x2},
		{0x2001d162, 4, 0x2},
		{0x2001d321, 4, 0x2},
		{0x2001d4e0, 4, 0x2},
		{0x2001d69f, 4, 0x2},
		{0x2001d85e, 4, 0x2},
		{0x2001da1d, 4, 0x2},
		{0x2001dbdc, 4, 0x2},
		{0x2001dd9b, 4, 0x2},
		{0x2001df5a, 4, 0x2},
		{0x2001e119, 4, 0x2},
		{0x2001e2d8, 4, 0x2},
		{0x2001e497, 4, 0x2},
		{0x2001e656, 4, 0x2},
		{0x2001e815, 4, 0x2},
		{0x2001e9d4, 4, 0x2},
		{0x2001eb93, 4, 0x2},
		{0x2001ed52, 4, 0x2},
		{0x2001ef11, 4, 0x2},
		{0x2001f0d0, 4, 0x2},
		{0x2001f28f, 4, 0x2},
		{0x2001f44e, 4, 0x2},
		{0x2001f60d, 4, 0x2},
		{0x2001f7cc, 4, 0x2},
		{0x2001f98b, 4, 0x2},
		{0x2001fb4a, 4, 0x2},
		{0x2001fd09, 4, 0x2},
		{0x2001fec8, 4, 0x2},
		{0x20020087, 4, 0x2},
		{0x20020246, 4, 0x2},
		{0x20020405, 4, 0x2},
		{0x200205c4, 4, 0x2},
		{0x20020783, 4, 0x2},
		{0x20020942, 4, 0x2},
		{0x20020b01, 4, 0x2},
		{0x20020cc0, 4, 0x2},
		{0x20020e7f, 4, 0x2},
		{0x2002103e, 4, 0x2},
		{0x200211fd, 4, 0x2},
		{0x200213bc, 4, 0x2},
		{0x2002157b, 4, 0x2},
		{0x2002173a, 4, 0x2},
		{0x200218f9, 4, 0x2},
		{0x20021ab8, 4, 0x2},
		{0x20021c77, 4, 0x2},
		{0x20021e36, 4, 0x2},
		{0x20021ff5, 4, 0x2},
		{0x200221b4, 4, 0x2},
		{0x20022373, 4, 0x2},
		{0x20022532, 4, 0x2},
		{0x200226f1, 4, 0x2},
		{0x200228b0, 4, 0x2},
		{0x20022a6f, 4, 0x2},
		{0x20022c2e, 4, 0x2},
		{0x20022ded, 4, 0x2},
		{0x20022fac, 4, 0x2},
		{0x2002316b, 4, 0x2},
		{0x2002332a, 4, 0x2},
		{0x200234e9, 4, 0x2},
		{0x200236a8, 4, 0x2},
		{0x20023867, 4, 0x2},
		{0x20023a26, 4, 0x2},
		{0x20023be5, 4, 0x2},
		{0x20023da4, 4, 0x2},
		{0x20023f63, 4, 0x2},
		{0x20024122, 4, 0x2},
		{0x200242e1, 4, 0x2},
		{0x200244a0, 4, 0x2},
		{0x2002465f, 4, 0x2},
		{0x2002481e, 4, 0x2},
		{0x200249dd, 4, 0x2},
		{0x20024b9c, 4, 0x2},
		{0x20024d5b, 4, 0x2},
		{0x20024f1a, 4, 0x2},
		{0x200250d9, 4, 0x2},
		{0x20025298, 4, 0x2},
		{0x20025457, 4, 0x2},
		{0x20025616, 4, 0x2},
		{0x200257d5, 4, 0x2},
		{0x20025994, 4, 0x2},
		{0x20025b53, 4, 0x2},
		{0x20025d12, 4, 0x2},
		{0x20025ed1, 4, 0x2},
		{0x20026090, 4, 0x2},
		{0x2002624f, 4, 0x2},
		{0x2002640e, 4, 0x2},
		{0x200265cd, 4, 0x2},
		{0x2002678c, 4, 0x2},
		{0x2002694b, 4, 0x2},
		{0x20026b0a, 4, 0x2},
		{0x20026cc9, 4, 0x2},
		{0x20026e88, 4, 0x2},
		{0x20027047, 4, 0x2},
		{0x20027206, 4, 0x2},
		{0x200273c5, 4, 0x2},
		{0x20027584, 4, 0x2},
		{0x20027743, 4, 0x2},
		{0x20027902, 4, 0x2},
		{0x20027ac1, 4, 0x2},
		{0x20027c80, 4, 0x2},
		{0x20027e3f, 4, 0x2},
		{0x20027ffe, 4, 0x2},
		{0x200281bd, 4, 0x2},
		{0x2002837c, 4, 0x2},
		{0x2002853b, 4, 0x2},
		{0x200286fa, 4, 0x2},
		{0x200288b9, 4, 0x2},
		{0x20028a78, 4, 0x2},
		{0x20028c37, 4, 0x2},
		{0x20000d8d, 4, 0x4},
		{0x200025ef, 4, 0x4},
		{0x20003e51, 4, 0x4},
		{0x200056b3, 4, 0x4},
		{0x20006f15, 4, 0x4},
		{0x20008777, 4, 0x4},
		{0x20009fd9, 4, 0x4},
		{0x2000b83b, 4, 0x4},
		{0x20000d99, 4, 0x3},
		{0x200025fb, 4, 0x3},
		{0x20003e5d, 4, 0x3},
		{0x200056bf, 4, 0x3},
		{0x20006f21, 4, 0x3},
		{0x20008783, 4, 0x3},
		{0x20009fe5, 4, 0x3},
		{0x2000b847, 4, 0x3},
		{0x20000d9d, 4, 0x2},
		{0x200025ff, 4, 0x2},
		{0x20003e61, 4, 0x2},
		{0x200056c3, 4, 0x2},
		{0x20006f25, 4, 0x2},
		{0x20008787, 4, 0x2},
		{0x20009fe9, 4, 0x2},
		{0x2000b84b, 4, 0x2},
		{0x20000d91, 4, 0x4},
		{0x200025f3, 4, 0x4},
		{0x20003e55, 4, 0x4},
		{0x200056b7, 4, 0x4},
		{0x20006f19, 4, 0x4},
		{0x2000877b, 4, 0x4},
		{0x20009fdd, 4, 0x4},
		{0x2000b83f, 4, 0x4},
		{0x20000c8d, 4, 0x5},
		{0x200024ef, 4, 0x5},
		{0x20003d51, 4, 0x5},
		{0x200055b3, 4, 0x5},
		{0x20006e15, 4, 0x5},
		{0x20008677, 4, 0x5},
		{0x20009ed9, 4, 0x5},
		{0x2000b73b, 4, 0x5},
		{0x20000c91, 4, 0x4},
		{0x200024f3, 4, 0x4},
		{0x20003d55, 4, 0x4},
		{0x200055b7, 4, 0x4},
		{0x20006e19, 4, 0x4},
		{0x2000867b, 4, 0x4},
		{0x20009edd, 4, 0x4},
		{0x2000b73f, 4, 0x4},
		{0x20000c95, 4, 0x4},
		{0x200024f7, 4, 0x4},
		{0x20003d59, 4, 0x4},
		{0x200055bb, 4, 0x4},
		{0x20006e1d, 4, 0x4},
		{0x2000867f, 4, 0x4},
		{0x20009ee1, 4, 0x4},
		{0x2000b743, 4, 0x4},
		{0x20000c99, 4, 0x3},
		{0x200024fb, 4, 0x3},
		{0x20003d5d, 4, 0x3},
		{0x200055bf, 4, 0x3},
		{0x20006e21, 4, 0x3},
		{0x20008683, 4, 0x3},
		{0x20009ee5, 4, 0x3},
		{0x2000b747, 4, 0x3},
		{0x20000c9d, 4, 0x2},
		{0x200024ff, 4, 0x2},
		{0x20003d61, 4, 0x2},
		{0x200055c3, 4, 0x2},
		{0x20006e25, 4, 0x2},
		{0x20008687, 4, 0x2},
		{0x20009ee9, 4, 0x2},
		{0x2000b74b, 4, 0x2},
		{0x20000ca1, 4, 0x2},
		{0x20002503, 4, 0x2},
		{0x20003d65, 4, 0x2},
		{0x200055c7, 4, 0x2},
		{0x20006e29, 4, 0x2},
		{0x2000868b, 4, 0x2},
		{0x20009eed, 4, 0x2},
		{0x2000b74f, 4, 0x2},
		{0x20000ca5, 4, 0x6},
		{0x20002507, 4, 0x6},
		{0x20003d69, 4, 0x6},
		{0x200055cb, 4, 0x6},
		{0x20006e2d, 4, 0x6},
		{0x2000868f, 4, 0x6},
		{0x20009ef1, 4, 0x6},
		{0x2000b753, 4, 0x6},
		{0x20000cad, 4, 0x6},
		{0x2000250f, 4, 0x6},
		{0x20003d71, 4, 0x6},
		{0x200055d3, 4, 0x6},
		{0x20006e35, 4, 0x6},
		{0x20008697, 4, 0x6},
		{0x20009ef9, 4, 0x6},
		{0x2000b75b, 4, 0x6},
		{0x20000cb1, 4, 0x5},
		{0x20002513, 4, 0x5},
		{0x20003d75, 4, 0x5},
		{0x200055d7, 4, 0x5},
		{0x20006e39, 4, 0x5},
		{0x2000869b, 4, 0x5},
		{0x20009efd, 4, 0x5},
		{0x2000b75f, 4, 0x5},
		{0x20000cb5, 4, 0x5},
		{0x20002517, 4, 0x5},
		{0x20003d79, 4, 0x5},
		{0x200055db, 4, 0x5},
		{0x20006e3d, 4, 0x5},
		{0x2000869f, 4, 0x5},
		{0x20009f01, 4, 0x5},
		{0x2000b763, 4, 0x5},
		{0x20000cb9, 4, 0x4},
		{0x2000251b, 4, 0x4},
		{0x20003d7d, 4, 0x4},
		{0x200055df, 4, 0x4},
		{0x20006e41, 4, 0x4},
		{0x200086a3, 4, 0x4},
		{0x20009f05, 4, 0x4},
		{0x2000b767, 4, 0x4},
		{0x20000cbd, 4, 0x4},
		{0x2000251f, 4, 0x4},
		{0x20003d81, 4, 0x4},
		{0x200055e3, 4, 0x4},
		{0x20006e45, 4, 0x4},
		{0x200086a7, 4, 0x4},
		{0x20009f09, 4, 0x4},
		{0x2000b76b, 4, 0x4},
		{0x20000cc1, 4, 0x3},
		{0x20002523, 4, 0x3},
		{0x20003d85, 4, 0x3},
		{0x200055e7, 4, 0x3},
		{0x20006e49, 4, 0x3},
		{0x200086ab, 4, 0x3},
		{0x20009f0d, 4, 0x3},
		{0x2000b76f, 4, 0x3},
		{0x20000d05, 4, 0x3},
		{0x20002567, 4, 0x3},
		{0x20003dc9, 4, 0x3},
		{0x2000562b, 4, 0x3},
		{0x20006e8d, 4, 0x3},
		{0x200086ef, 4, 0x3},
		{0x20009f51, 4, 0x3},
		{0x2000b7b3, 4, 0x3},
		{0x20000d11, 4, 0x2},
		{0x20002573, 4, 0x2},
		{0x20003dd5, 4, 0x2},
		{0x20005637, 4, 0x2},
		{0x20006e99, 4, 0x2},
		{0x200086fb, 4, 0x2},
		{0x20009f5d, 4, 0x2},
		{0x2000b7bf, 4, 0x2},
		{0x20000d1d, 4, 0x5},
		{0x2000257f, 4, 0x5},
		{0x20003de1, 4, 0x5},
		{0x20005643, 4, 0x5},
		{0x20006ea5, 4, 0x5},
		{0x20008707, 4, 0x5},
		{0x20009f69, 4, 0x5},
		{0x2000b7cb, 4, 0x5},
		{0x20000d29, 4, 0x3},
		{0x2000258b, 4, 0x3},
		{0x20003ded, 4, 0x3},
		{0x2000564f, 4, 0x3},
		{0x20006eb1, 4, 0x3},
		{0x20008713, 4, 0x3},
		{0x20009f75, 4, 0x3},
		{0x2000b7d7, 4, 0x3},
		{0x20000d35, 4, 0x2},
		{0x20002597, 4, 0x2},
		{0x20003df9, 4, 0x2},
		{0x2000565b, 4, 0x2},
		{0x20006ebd, 4, 0x2},
		{0x2000871f, 4, 0x2},
		{0x20009f81, 4, 0x2},
		{0x2000b7e3, 4, 0x2},
		{0x20000d41, 4, 0x5},
		{0x200025a3, 4, 0x5},
		{0x20003e05, 4, 0x5},
		{0x20005667, 4, 0x5},
		{0x20006ec9, 4, 0x5},
		{0x2000872b, 4, 0x5},
		{0x20009f8d, 4, 0x5},
		{0x2000b7ef, 4, 0x5},
		{0x200005eb, 4, 0x4},
		{0x20001e4d, 4, 0x4},
		{0x200036af, 4, 0x4},
		{0x20004f11, 4, 0x4},
		{0x20006773, 4, 0x4},
		{0x20007fd5, 4, 0x4},
		{0x20009837, 4, 0x4},
		{0x2000b099, 4, 0x4},
		{0x200007ff, 4, 0x5},
		{0x20002061, 4, 0x5},
		{0x200038c3, 4, 0x5},
		{0x20005125, 4, 0x5},
		{0x20006987, 4, 0x5},
		{0x200081e9, 4, 0x5},
		{0x20009a4b, 4, 0x5},
		{0x2000b2ad, 4, 0x5},
		{0x20000e4d, 4, 0x4},
		{0x200026af, 4, 0x4},
		{0x20003f11, 4, 0x4},
		{0x20005773, 4, 0x4},
		{0x20006fd5, 4, 0x4},
		{0x20008837, 4, 0x4},
		{0x2000a099, 4, 0x4},
		{0x2000b8fb, 4, 0x4},
		{0x20000f2d, 4, 0x2},
		{0x2000278f, 4, 0x2},
		{0x20003ff1, 4, 0x2},
		{0x20005853, 4, 0x2},
		{0x200070b5, 4, 0x2},
		{0x20008917, 4, 0x2},
		{0x2000a179, 4, 0x2},
		{0x2000b9db, 4, 0x2},
		{0x20001011, 4, 0x5},
		{0x20002873, 4, 0x5},
		{0x200040d5, 4, 0x5},
		{0x20005937, 4, 0x5},
		{0x20007199, 4, 0x5},
		{0x200089fb, 4, 0x5},
		{0x2000a25d, 4, 0x5},
		{0x2000babf, 4, 0x5},
		{0x2002acb8, 4, 0x1},
		{0x2002b0b8, 4, 0x1},
		{0x2002b4b8, 4, 0x1},
		{0x2002b8b8, 4, 0x1},
		{0x2002acc5, 1, 0x2},
		{0x2002b0c5, 1, 0x2},
		{0x2002b4c5, 1, 0x2},
		{0x2002b8c5, 1, 0x2},
		{0x2002acc6, 1, 0x3},
		{0x2002b0c6, 1, 0x3},
		{0x2002b4c6, 1, 0x3},
		{0x2002b8c6, 1, 0x3},
		{0x2002acd7, 4, 0x6},
		{0x2002b0d7, 4, 0x6},
		{0x2002b4d7, 4, 0x6},
		{0x2002b8d7, 4, 0x6},
		{0x2002acdb, 4, 0x6},
		{0x2002b0db, 4, 0x6},
		{0x2002b4db, 4, 0x6},
		{0x2002b8db, 4, 0x6},
		{0x2002acdf, 4, 0x5},
		{0x2002b0df, 4, 0x5},
		{0x2002b4df, 4, 0x5},
		{0x2002b8df, 4, 0x5},
		{0x2002ace3, 4, 0x5},
		{0x2002b0e3, 4, 0x5},
		{0x2002b4e3, 4, 0x5},
		{0x2002b8e3, 4, 0x5},
		{0x2002ad13, 4, 0x2},
		{0x2002b113, 4, 0x2},
		{0x2002b513, 4, 0x2},
		{0x2002b913, 4, 0x2},
		{0x2002ad17, 4, 0x2},
		{0x2002b117, 4, 0x2},
		{0x2002b517, 4, 0x2},
		{0x2002b917, 4, 0x2},
		{0x2002ad1b, 4, 0x6},
		{0x2002b11b, 4, 0x6},
		{0x2002b51b, 4, 0x6},
		{0x2002b91b, 4, 0x6},
		{0x2002ad1f, 4, 0x6},
		{0x2002b11f, 4, 0x6},
		{0x2002b51f, 4, 0x6},
		{0x2002b91f, 4, 0x6},
		{0x2002ad23, 4, 0x5},
		{0x2002b123, 4, 0x5},
		{0x2002b523, 4, 0x5},
		{0x2002b923, 4, 0x5},
		{0x2002ad27, 4, 0x4},
		{0x2002b127, 4, 0x4},
		{0x2002b527, 4, 0x4},
		{0x2002b927, 4, 0x4},
		{0x2002ad2b, 4, 0x4},
		{0x2002b12b, 4, 0x4},
		{0x2002b52b, 4, 0x4},
		{0x2002b92b, 4, 0x4},
		{0x2002ad33, 4, 0x3},
		{0x2002b133, 4, 0x3},
		{0x2002b533, 4, 0x3},
		{0x2002b933, 4, 0x3},
		{0x2002ad37, 4, 0x2},
		{0x2002b137, 4, 0x2},
		{0x2002b537, 4, 0x2},
		{0x2002b937, 4, 0x2},
		{0x2002ad3b, 4, 0x6},
		{0x2002b13b, 4, 0x6},
		{0x2002b53b, 4, 0x6},
		{0x2002b93b, 4, 0x6},
		{0x2002ad3f, 4, 0x6},
		{0x2002b13f, 4, 0x6},
		{0x2002b53f, 4, 0x6},
		{0x2002b93f, 4, 0x6},
		{0x2002ad43, 4, 0x5},
		{0x2002b143, 4, 0x5},
		{0x2002b543, 4, 0x5},
		{0x2002b943, 4, 0x5},
		{0x2002ad47, 4, 0x5},
		{0x2002b147, 4, 0x5},
		{0x2002b547, 4, 0x5},
		{0x2002b947, 4, 0x5},
		{0x2002ad77, 4, 0x2},
		{0x2002b177, 4, 0x2},
		{0x2002b577, 4, 0x2},
		{0x2002b977, 4, 0x2},
		{0x2002ad83, 4, 0x2},
		{0x2002b183, 4, 0x2},
		{0x2002b583, 4, 0x2},
		{0x2002b983, 4, 0x2},
		{0x2002ad8f, 4, 0x5},
		{0x2002b18f, 4, 0x5},
		{0x2002b58f, 4, 0x5},
		{0x2002b98f, 4, 0x5},
		{0x2002ad9b, 4, 0x3},
		{0x2002b19b, 4, 0x3},
		{0x2002b59b, 4, 0x3},
		{0x2002b99b, 4, 0x3},
		{0x2002ada7, 4, 0x6},
		{0x2002b1a7, 4, 0x6},
		{0x2002b5a7, 4, 0x6},
		{0x2002b9a7, 4, 0x6},
		{0x2002adb3, 4, 0x4},
		{0x2002b1b3, 4, 0x4},
		{0x2002b5b3, 4, 0x4},
		{0x2002b9b3, 4, 0x4},
		{0x2002adcc, 4, 0x2},
		{0x2002b1cc, 4, 0x2},
		{0x2002b5cc, 4, 0x2},
		{0x2002b9cc, 4, 0x2},
		{0x2002addd, 4, 0x2},
		{0x2002b1dd, 4, 0x2},
		{0x2002b5dd, 4, 0x2},
		{0x2002b9dd, 4, 0x2},
		{0x2002adee, 4, 0x5},
		{0x2002b1ee, 4, 0x5},
		{0x2002b5ee, 4, 0x5},
		{0x2002b9ee, 4, 0x5},
		{0x2002adff, 4, 0x4},
		{0x2002b1ff, 4, 0x4},
		{0x2002b5ff, 4, 0x4},
		{0x2002b9ff, 4, 0x4},
		{0x2002ae10, 4, 0x3},
		{0x2002b210, 4, 0x3},
		{0x2002b610, 4, 0x3},
		{0x2002ba10, 4, 0x3},
		{0x2002ae21, 4, 0x6},
		{0x2002b221, 4, 0x6},
		{0x2002b621, 4, 0x6},
		{0x2002ba21, 4, 0x6},
		{0x2002ae32, 4, 0x5},
		{0x2002b232, 4, 0x5},
		{0x2002b632, 4, 0x5},
		{0x2002ba32, 4, 0x5},
		{0x2002ae43, 4, 0x5},
		{0x2002b243, 4, 0x5},
		{0x2002b643, 4, 0x5},
		{0x2002ba43, 4, 0x5},
		{0x2002ae54, 4, 0x4},
		{0x2002b254, 4, 0x4},
		{0x2002b654, 4, 0x4},
		{0x2002ba54, 4, 0x4},
		{0x2002ae65, 4, 0x2},
		{0x2002b265, 4, 0x2},
		{0x2002b665, 4, 0x2},
		{0x2002ba65, 4, 0x2},
		{0x2002ae76, 4, 0x6},
		{0x2002b276, 4, 0x6},
		{0x2002b676, 4, 0x6},
		{0x2002ba76, 4, 0x6},
		{0x2002ae87, 4, 0x5},
		{0x2002b287, 4, 0x5},
		{0x2002b687, 4, 0x5},
		{0x2002ba87, 4, 0x5},
		{0x2002af8a, 4, 0x3},
		{0x2002b38a, 4, 0x3},
		{0x2002b78a, 4, 0x3},
		{0x2002bb8a, 4, 0x3},
		{0x2002aff4, 4, 0x5},
		{0x2002b3f4, 4, 0x5},
		{0x2002b7f4, 4, 0x5},
		{0x2002bbf4, 4, 0x5},
		{0x2002cef0, 4, 0x2},
		{0x2002d2f0, 4, 0x2},
		{0x2002d6f0, 4, 0x2},
		{0x2002daf0, 4, 0x2},
		{0x2002bd94, 4, 0x2},
		{0x2002bf94, 4, 0x2},
		{0x2002c194, 4, 0x2},
		{0x2002c394, 4, 0x2},
		{0x2002bda5, 4, 0x6},
		{0x2002bfa5, 4, 0x6},
		{0x2002c1a5, 4, 0x6},
		{0x2002c3a5, 4, 0x6},
		{0x2002bdb6, 4, 0x5},
		{0x2002bfb6, 4, 0x5},
		{0x2002c1b6, 4, 0x5},
		{0x2002c3b6, 4, 0x5},
		{0x2002bdc7, 4, 0x3},
		{0x2002bfc7, 4, 0x3},
		{0x2002c1c7, 4, 0x3},
		{0x2002c3c7, 4, 0x3},
		{0x2002bdd8, 4, 0x2},
		{0x2002bfd8, 4, 0x2},
		{0x2002c1d8, 4, 0x2},
		{0x2002c3d8, 4, 0x2},
		{0x2002bde9, 4, 0x2},
		{0x2002bfe9, 4, 0x2},
		{0x2002c1e9, 4, 0x2},
		{0x2002c3e9, 4, 0x2},
		{0x2002bdfa, 4, 0x6},
		{0x2002bffa, 4, 0x6},
		{0x2002c1fa, 4, 0x6},
		{0x2002c3fa, 4, 0x6},
		{0x2002be0b, 4, 0x4},
		{0x2002c00b, 4, 0x4},
		{0x2002c20b, 4, 0x4},
		{0x2002c40b, 4, 0x4},
		{0x2002be1c, 4, 0x3},
		{0x2002c01c, 4, 0x3},
		{0x2002c21c, 4, 0x3},
		{0x2002c41c, 4, 0x3},
	},
	in: []mstFld{
		mstFld{addr: 0x7f000008, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20000140, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20000144, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20000148, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000cfd7, size: 4, signed: false, idx: 0, count: 256, stride: 447},
		mstFld{addr: 0x2000cee8, size: 1, signed: false, idx: 0, count: 256, stride: 447},
		mstFld{addr: 0x2000cfc6, size: 1, signed: false, idx: 0, count: 256, stride: 447},
		mstFld{addr: 0x2000cfd3, size: 4, signed: false, idx: 0, count: 256, stride: 447},
		mstFld{addr: 0x2000cee9, size: 4, signed: true, idx: 0, count: 256, stride: 447},
		mstFld{addr: 0x2000cef6, size: 4, signed: true, idx: 0, count: 256, stride: 447},
		mstFld{addr: 0x20028db8, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20028dc5, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20028dc6, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x200290a9, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20029302, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x683f94, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x652da4, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x68d524, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a417, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a37e, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a428, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a38f, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a439, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a3a0, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a44a, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a3b1, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a45b, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a3c2, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a46c, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a3d3, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a47d, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a3e4, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a48e, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a3f5, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a49f, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002a406, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x68692c, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x68693d, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x68694e, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x68695f, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x686970, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x686981, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x686992, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x6869a3, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x6869b4, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x6869c5, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x6869d6, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x6869e7, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002947b, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20029490, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002961f, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002987b, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20029890, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20029a1f, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x200298fd, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000cee8, size: 1, signed: false, idx: 54, count: 256, stride: 447},
		mstFld{addr: 0x2000cfc6, size: 1, signed: false, idx: 54, count: 256, stride: 447},
		mstFld{addr: 0x2000cfd3, size: 4, signed: false, idx: 54, count: 256, stride: 447},
		mstFld{addr: 0x200298fe, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000cee8, size: 1, signed: false, idx: 58, count: 256, stride: 447},
		mstFld{addr: 0x2000cfc6, size: 1, signed: false, idx: 58, count: 256, stride: 447},
		mstFld{addr: 0x2000cfd3, size: 4, signed: false, idx: 58, count: 256, stride: 447},
		mstFld{addr: 0x200298ff, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000cee8, size: 1, signed: false, idx: 62, count: 256, stride: 447},
		mstFld{addr: 0x2000cfc6, size: 1, signed: false, idx: 62, count: 256, stride: 447},
		mstFld{addr: 0x2000cfd3, size: 4, signed: false, idx: 62, count: 256, stride: 447},
		mstFld{addr: 0x20029900, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000cee8, size: 1, signed: false, idx: 66, count: 256, stride: 447},
		mstFld{addr: 0x2000cfc6, size: 1, signed: false, idx: 66, count: 256, stride: 447},
		mstFld{addr: 0x2000cfd3, size: 4, signed: false, idx: 66, count: 256, stride: 447},
		mstFld{addr: 0x20029901, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000cee8, size: 1, signed: false, idx: 70, count: 256, stride: 447},
		mstFld{addr: 0x2000cfc6, size: 1, signed: false, idx: 70, count: 256, stride: 447},
		mstFld{addr: 0x2000cfd3, size: 4, signed: false, idx: 70, count: 256, stride: 447},
		mstFld{addr: 0x20029902, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000cee8, size: 1, signed: false, idx: 74, count: 256, stride: 447},
		mstFld{addr: 0x2000cfc6, size: 1, signed: false, idx: 74, count: 256, stride: 447},
		mstFld{addr: 0x2000cfd3, size: 4, signed: false, idx: 74, count: 256, stride: 447},
		mstFld{addr: 0x20029903, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000cee8, size: 1, signed: false, idx: 78, count: 256, stride: 447},
		mstFld{addr: 0x2000cfc6, size: 1, signed: false, idx: 78, count: 256, stride: 447},
		mstFld{addr: 0x2000cfd3, size: 4, signed: false, idx: 78, count: 256, stride: 447},
		mstFld{addr: 0x20029904, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000cee8, size: 1, signed: false, idx: 82, count: 256, stride: 447},
		mstFld{addr: 0x2000cfc6, size: 1, signed: false, idx: 82, count: 256, stride: 447},
		mstFld{addr: 0x2000cfd3, size: 4, signed: false, idx: 82, count: 256, stride: 447},
		mstFld{addr: 0x20029905, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000cee8, size: 1, signed: false, idx: 86, count: 256, stride: 447},
		mstFld{addr: 0x2000cfc6, size: 1, signed: false, idx: 86, count: 256, stride: 447},
		mstFld{addr: 0x2000cfd3, size: 4, signed: false, idx: 86, count: 256, stride: 447},
		mstFld{addr: 0x20029906, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000cee8, size: 1, signed: false, idx: 90, count: 256, stride: 447},
		mstFld{addr: 0x2000cfc6, size: 1, signed: false, idx: 90, count: 256, stride: 447},
		mstFld{addr: 0x2000cfd3, size: 4, signed: false, idx: 90, count: 256, stride: 447},
		mstFld{addr: 0x20000d8d, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000d99, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000d9d, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000d91, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000c8d, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000c91, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000c95, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000c99, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000c9d, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000ca1, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000ca5, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000cad, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000cb1, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000cb5, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000cb9, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000cbd, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000cc1, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000d05, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000d11, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000d1d, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000d29, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000d35, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000d41, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x200005eb, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x200007ff, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000e4d, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20000f2d, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x20001011, size: 4, signed: false, idx: 2, count: 8, stride: 6242},
		mstFld{addr: 0x2002acb8, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002acc5, size: 1, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002acc6, size: 1, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002acd7, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002acdb, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002acdf, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ace3, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad13, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad17, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad1b, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad1f, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad23, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad27, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad2b, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad33, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad37, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad3b, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad3f, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad43, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad47, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad77, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad83, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad8f, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ad9b, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ada7, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002adb3, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002adcc, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002addd, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002adee, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002adff, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ae10, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ae21, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ae32, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ae43, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ae54, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ae65, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ae76, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002ae87, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002af8a, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002aff4, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002cef0, size: 4, signed: false, idx: 3, count: 4, stride: 1024},
		mstFld{addr: 0x2002bd94, size: 4, signed: false, idx: 3, count: 4, stride: 512},
		mstFld{addr: 0x2002bda5, size: 4, signed: false, idx: 3, count: 4, stride: 512},
		mstFld{addr: 0x2002bdb6, size: 4, signed: false, idx: 3, count: 4, stride: 512},
		mstFld{addr: 0x2002bdc7, size: 4, signed: false, idx: 3, count: 4, stride: 512},
		mstFld{addr: 0x2002bdd8, size: 4, signed: false, idx: 3, count: 4, stride: 512},
		mstFld{addr: 0x2002bde9, size: 4, signed: false, idx: 3, count: 4, stride: 512},
		mstFld{addr: 0x2002bdfa, size: 4, signed: false, idx: 3, count: 4, stride: 512},
		mstFld{addr: 0x2002be0b, size: 4, signed: false, idx: 3, count: 4, stride: 512},
		mstFld{addr: 0x2002be1c, size: 4, signed: false, idx: 3, count: 4, stride: 512},
		mstFld{addr: 0x2002dcb0, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcb4, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcb8, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcbc, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcc0, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcc4, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcc8, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dccc, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcd0, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcd4, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcd8, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcdc, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dce0, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dce4, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dce8, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcec, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcf0, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcf4, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcf8, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dcfc, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd00, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd04, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd08, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd0c, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd10, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd14, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd18, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd1c, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd20, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd24, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd28, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd2c, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd30, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd34, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd38, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd3c, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd40, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd44, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd48, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd4c, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd50, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd54, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd58, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd5c, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd60, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd64, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd68, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd6c, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd70, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd74, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd78, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd7c, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd80, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd84, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd88, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd8c, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd90, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd94, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd98, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dd9c, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dda0, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dda4, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dda8, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddac, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddb0, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddb4, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddb8, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddbc, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddc0, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddc4, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddc8, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddcc, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddd0, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddd4, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddd8, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dddc, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dde0, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dde4, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002dde8, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddec, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddf0, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddf4, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddf8, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002ddfc, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002de00, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002de04, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002de08, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002de0c, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002de10, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002de14, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002de18, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002de1c, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2002de20, size: 4, signed: true, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x200001c0, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x200001c4, size: 4, signed: false, idx: -1, count: 0, stride: 0},
	},
	out: []mstFld{
		mstFld{addr: mstRet, size: 1, idx: -1},
		mstFld{addr: 0x68d524, size: 1, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20000180, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20000184, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20000188, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000018c, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20000190, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20000194, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x20000198, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x2000019c, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x200001a0, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x200001a4, size: 4, signed: false, idx: -1, count: 0, stride: 0},
		mstFld{addr: 0x200001a8, size: 4, signed: false, idx: -1, count: 0, stride: 0},
	},
	stub: []mstStubAt{
		{ret: 0x200001c0, rec: [][3]uint32{{1, 4, 0x20000180}, {2, 4, 0x20000184}, {3, 4, 0x20000188}}, cnt: 0x2000018c},
		{ret: 0x0, rec: [][3]uint32{{1, 4, 0x20000190}, {2, 4, 0x20000194}}, cnt: 0x20000198},
		{ret: 0x0, rec: [][3]uint32{{3, 4, 0x2000019c}}, cnt: 0x200001a0},
		{ret: 0x200001c4, rec: [][3]uint32{{1, 4, 0x200001a4}}, cnt: 0x200001a8},
		{ret: 0x0, rec: [][3]uint32{}, cnt: 0x0},
	},
	areaChars: []uint32{2, 5},
	areaMons:  []uint32{1, 2},
}

func mstNames_ok_in() []string {
	return []string{"spell", "q0", "q1", "q2", "spell.type", "spell.f48", "spell.f126", "spell.f133", "spell.f49", "spell.f56", "M.templateId", "M.f15", "M.f16", "M.bonus", "M.ext152", "combat", "flag652da4", "flag68d524", "ctx.s867_0", "ctx.s7ce_0", "ctx.s867_1", "ctx.s7ce_1", "ctx.s867_2", "ctx.s7ce_2", "ctx.s867_3", "ctx.s7ce_3", "ctx.s867_4", "ctx.s7ce_4", "ctx.s867_5", "ctx.s7ce_5", "ctx.s867_6", "ctx.s7ce_6", "ctx.s867_7", "ctx.s7ce_7", "ctx.s867_8", "ctx.s7ce_8", "grpA_0", "grpA_1", "grpA_2", "grpA_3", "grpA_4", "grpA_5", "grpA_6", "grpA_7", "grpA_8", "grpA_9", "grpA_10", "grpA_11", "tmpl0.cb", "tmpl0.e0", "tmpl0.f26f", "tmpl1.cb", "tmpl1.e0", "tmpl1.f26f", "tmpl1.spell0", "ts0.f48", "ts0.f126", "ts0.f133", "tmpl1.spell1", "ts1.f48", "ts1.f126", "ts1.f133", "tmpl1.spell2", "ts2.f48", "ts2.f126", "ts2.f133", "tmpl1.spell3", "ts3.f48", "ts3.f126", "ts3.f133", "tmpl1.spell4", "ts4.f48", "ts4.f126", "ts4.f133", "tmpl1.spell5", "ts5.f48", "ts5.f126", "ts5.f133", "tmpl1.spell6", "ts6.f48", "ts6.f126", "ts6.f133", "tmpl1.spell7", "ts7.f48", "ts7.f126", "ts7.f133", "tmpl1.spell8", "ts8.f48", "ts8.f126", "ts8.f133", "tmpl1.spell9", "ts9.f48", "ts9.f126", "ts9.f133", "ch.maxHP", "ch.maxSt", "ch.st", "ch.hp", "ch.Ac", "ch.A10", "ch.A14", "ch.A18", "ch.A1c", "ch.A20", "ch.A24", "ch.A2c", "ch.A30", "ch.A34", "ch.A38", "ch.A3c", "ch.A40", "ch.B20", "ch.B2c", "ch.B38", "ch.B44", "ch.B50", "ch.B5c", "ch.f36b", "ch.f57f", "ch.arr0", "ch.arr38", "ch.arr71", "mon.templateId", "mon.f15", "mon.f16", "mon.maxHP", "mon.hp", "mon.maxSt", "mon.st", "mon.Ac", "mon.A10", "mon.A14", "mon.A18", "mon.A1c", "mon.A20", "mon.A24", "mon.A2c", "mon.A30", "mon.A34", "mon.A38", "mon.A3c", "mon.A40", "mon.B20", "mon.B2c", "mon.B38", "mon.B44", "mon.B50", "mon.B5c", "mon.tbl0", "mon.tbl1", "mon.tbl2", "mon.tbl3", "mon.tbl4", "mon.tbl5", "mon.tbl6", "mon.tbl7", "mon.tbl8", "mon.tbl9", "mon.tbl10", "mon.tbl11", "mon.f2da", "mon.f344", "mon.anim240", "mext.e4_0", "mext.e4_1", "mext.e4_2", "mext.e4_3", "mext.e4_4", "mext.e4_5", "mext.e4_6", "mext.e4_7", "mext.e4_8", "area.sourceXBits", "area.sourceYBits", "area.sourceZBits", "area.targetXBits", "area.targetYBits", "area.targetZBits", "area.partyXBits", "area.partyYBits", "area.partyZBits", "area.partyAimXBits", "area.partyAimYBits", "area.partyAimZBits", "area.partyRadiusBits", "area.spellID", "area.spellTargetType", "area.areaBaseBits", "area.areaPerPowerBits", "area.spellRange", "area.powerLevel", "area.sourceKind", "area.sourceCharacter", "area.sourceMonster", "area.sourceSpecial", "area.sourceEnvironmental", "area.sourceUseCenter", "area.targetKind", "area.targetCharacter", "area.targetMonster", "area.targetGroup", "area.ignoreLineOfSight", "area.monsterPresent", "area.monsterVisible", "area.monsterFaction", "area.monsterHP", "area.monsterRemoved", "area.monsterCondition", "area.monsterMagicMouth", "area.monsterPartySight2", "area.monsterPartySight3", "area.monsterImmune", "area.monsterSpecies", "area.monsterSpecialStatus", "area.monsterXBits", "area.monsterYBits", "area.monsterZBits", "area.monsterRadiusBits", "area.monsterEyeHeightBits", "area.monsterSpellReachBits", "area.partySpellReachBits", "area.groupFaction", "area.groupMemberCount", "area.character0present", "area.character0HP", "area.character0condition", "area.character0magicProtection", "area.character1present", "area.character1HP", "area.character1condition", "area.character1magicProtection", "area.character2present", "area.character2HP", "area.character2condition", "area.character2magicProtection", "area.character3present", "area.character3HP", "area.character3condition", "area.character3magicProtection", "area.character4present", "area.character4HP", "area.character4condition", "area.character4magicProtection", "area.character5present", "area.character5HP", "area.character5condition", "area.character5magicProtection", "area.character6present", "area.character6HP", "area.character6condition", "area.character6magicProtection", "area.character7present", "area.character7HP", "area.character7condition", "area.character7magicProtection", "area.battleObject", "area.battleSuppressFallback", "area.monsterRayVisible", "area.monsterInitialCount", "area.monsterCapacity", "area.characterInitialCount", "area.characterCapacity", "area.monsterModelScaleBits", "area.monsterBoundsMinYBits", "area.monsterBoundsMaxYBits", "ret500330", "ret4d9080"}
}
func mstNames_ok_out() []string {
	return []string{"ret", "flag68d524", "rec_p1", "rec_p2", "rec_p3", "rec_pc", "rec_a1", "rec_a2", "rec_ac", "rec_l3", "rec_lc", "rec_u1", "rec_uc"}
}
