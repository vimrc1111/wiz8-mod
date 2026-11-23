package calc

// Node 0x536400: for each party slot x in 0..7 whose stored action (source 1) can
// no longer reach its target, store the monster target again (0x536570). It
// runs over the 520d10 group's byte world (n_520d10.go): In = the group's cells
// (cgIn) followed by cg6InX, Out = the group's memory cells, cg6OutX, the
// group's stub records, cg6OutS. 0x4e79a0 (gate), 0x5199f0 and 0x51b000 (kind 1
// reach) and 0x434f20 (map ray of a coordinate target) are address-stubbed.
// Slots 4..7 have no catalogued cells: their existence bytes read 0 and the
// loop skips them (a non-zero byte there panics in who).

func init() {
	in := append(append([]string{}, cgInNames...), cg6InNamesX...)
	out := append(append(append([]string{}, cgOutNames[:cg6OutSplit]...), cg6OutNamesX[:len(cg6OutX)]...), cgOutNames[cg6OutSplit:]...)
	out = append(out, cg6OutNamesX[len(cg6OutX):]...)
	Nodes["00536400"] = Node{Addr: "00536400", In: in, Out: out, F: partyTargetRevalidateNode}
}

func partyTargetRevalidateNode(in []int64, r Rand) []int64 {
	if len(in) != len(cgIn)+len(cg6InX) {
		panic("cg: wrong input count")
	}
	w := &cgW{m: map[int64]byte{}}
	for _, c := range cgInit {
		w.wr(c.a, c.s, c.v)
	}
	for _, c := range cg6Init {
		w.wr(c.a, c.s, c.v)
	}
	for i, c := range append(append([]cgCell{}, cgIn...), cg6InX...) {
		w.wr(c.a, abs(c.s), in[i])
	}
	for x := int64(0); x < 8; x++ {
		if w.u8(w.u32(0x685178)+0x106*x) != 0 {
			w.revalidate(x)
		}
	}
	read := func(cs []cgCell) (o []int64) {
		for _, c := range cs {
			if c.s < 0 {
				o = append(o, w.rs(c.a, -c.s))
			} else {
				o = append(o, w.rd(c.a, c.s))
			}
		}
		return
	}
	var out []int64
	for _, cs := range [][]cgCell{cgOut[:cg6OutSplit], cg6OutX, cgOut[cg6OutSplit:], cg6OutS} {
		out = append(out, read(cs)...)
	}
	return out
}

// revalidate is one iteration of the loop: a living slot whose stored action
// still names a target of the right class but cannot reach it gets a new
// monster target.
func (w *cgW) revalidate(x int64) {
	C := w.C(x)
	if w.i32(C+0xb11) == 0 || w.u32(C+0xb01) >= 0xd {
		return
	}
	if w.st(cgS4e79a0, x, 1, 1, 0)&0xff == 0 {
		return
	}
	a, b, c, d := w.act(x, 1)
	if w.kindOK(c, w.actionClass(a, b, d)) && w.reach(x, 2, 1) {
		return
	}
	w.target(x, 1, 0)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// Cells after the 520d10 group's catalog: the three new stub return cells, the spell range field, the camera and the view-distance cell.
var cg6InX = []cgCell{{0x20017150, -4}, {0x20017154, -4}, {0x20017158, -4}, {0x2001e2ee, -4}, {0x2001e4ad, -4}, {0x2001e66c, -4}, {0x2001e82b, -4}, {0x2001e9ea, -4}, {0x2001eba9, -4}, {0x2001ed68, -4}, {0x2001ef27, -4}, {0x2001f0e6, -4}, {0x20027ad7, -4}, {0x2002b0f8, -4}, {0x2003308c, -4}, {0x20033090, -4}, {0x20033094, -4}, {0x20034084, -4}}
var cg6OutX = []cgCell{{0x2001e2ee, -4}, {0x2001e4ad, -4}, {0x2001e66c, -4}, {0x2001e82b, -4}, {0x2001e9ea, -4}, {0x2001eba9, -4}, {0x2001ed68, -4}, {0x2001ef27, -4}, {0x2001f0e6, -4}, {0x20027ad7, -4}, {0x2002b0f8, -4}, {0x2003308c, -4}, {0x20033090, -4}, {0x20033094, -4}, {0x20034084, -4}}
var cg6OutS = []cgCell{{0x20017850, 4}, {0x20017708, -4}, {0x2001770c, -4}, {0x20017710, -4}, {0x20017714, -4}, {0x20017718, -4}, {0x20017854, 4}, {0x20017740, -4}, {0x20017744, -4}, {0x20017748, -4}, {0x20017858, 4}, {0x20017780, -4}, {0x20017784, -4}}
var cg6Init = []cgInitCell{{0x65a0f8, 4, 0x20033000}, {0x659c0c, 4, 0x20034000}}

const cg6OutSplit = 408 // cgOut: this many catalogued memory cells, then the stub records
var cg6InNamesX = []string{"sr_434f20", "sr_5199f0", "sr_51b000", "sp1.f", "sp2.f", "sp3.f", "sp4.f", "sp5.f", "sp6.f", "sp7.f", "sp8.f", "sp9.f", "sp88.f", "sp119.f", "geo.a0", "geo.a1", "geo.a2", "geo2.v"}
var cg6OutNamesX = []string{"sp1.f", "sp2.f", "sp3.f", "sp4.f", "sp5.f", "sp6.f", "sp7.f", "sp8.f", "sp9.f", "sp88.f", "sp119.f", "geo.a0", "geo.a1", "geo.a2", "geo2.v", "cnt_434f20", "rec_434f20_3", "rec_434f20_4", "rec_434f20_5", "rec_434f20_6", "rec_434f20_7", "cnt_5199f0", "rec_5199f0_1", "rec_5199f0_2", "rec_5199f0_3", "cnt_51b000", "rec_51b000_1", "rec_51b000_2"}
