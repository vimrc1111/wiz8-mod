package calc

import "fmt"

// post_rest_status_duration 0x5029e0 (build/wiz8.asm:348438..348545), no arguments.
//
// Loop 1 runs when the party record pointer 0x6836a8 is non-null and its byte
// +0xa52 is non-zero. For member m = 0..7 whose presence byte (record 0x685178,
// stride 0x106, byte 0) is non-zero and whose main state +0xb01 (record 0x685174,
// stride 0x1862) is below 0x12: v = R(100) - 20 - floor(u32(C+0x161 * 70) / 100)
// as int32. v > 0 calls 0x522fe0(m, 0xf, v/30+1, 0, 0, 0) (character_condition_apply,
// state 0xf); otherwise 0x590950 formats the notice text (the text-table entry at
// 0x68c09c + 4*(u16 0x61e5e8) is only a printf argument) and calls the message
// routine 0x58aad0(8, 0x64bf5c, record+5) once.
//
// Loop 2 runs for every present member: 0x52f000 sets the portrait record's
// (0x6836b8 + 0x118*m) byte +0x99 to 0 and its expression +0x8d to 1, or 2 when
// +0xb01 >= 0xf or byte 0x683fc5 is set; R(2) != 0 stores 0x406bf0(R(5000)+5000)
// (= that plus u32 0x6eb70c) at +0x95; byte +0x9a becomes 1; 0x562a50(1<<m) ORs
// the bit into the battle object's (0x68edcc) +0xf4 when 0x68ec78 is 7 and the
// object exists. Last, 0x598c10 calls 0x5db8d0(widget, 1) on twelve widgets
// (pointers at 0x69b858/5c, 0x69b8e4, 0x69b8d8/dc/e0, 0x69b884..98): byte +0x35
// := 1 and, when the widget index +0x1c is 0..0x18f and table 0x6e1240 holds an
// object for it, that object's flag word +0x60 gets bit 0 and the widget byte
// +0x38 becomes 1 (only when the bit was clear).
//
// Emu address-stubs the D4 callee 0x522fe0 (return ignored by the caller: no output
// reads it) and the message routine 0x58aad0 (as in n_522fe0.go): every call's
// arguments and the call counts are outputs. The text box colour helper 0x58b410
// (inside 0x590950) runs natively and is not an output. The rest of the callees
// run natively.
//
// F calls the real characterConditionApplyNode path (ccaApply) per call on a
// callee world: the F-only inputs ft* (one W template shared by all members, class
// and race included), fs<m>_<k> (state slots of member m), fb09/fb0d/fb11 (+0xb09,
// +0xb0d, +0xb11); b01 and the presence bytes come from the node inputs. It feeds
// the new +0xb01 forward, which changes the expression 0x52f000 writes.
//
// Fixture caps (calc panics past them): party and battle pointers 0 or the
// fixtures, character base the fixture, and battle mode 7 with a null battle
// object when a member fails the roll (0x58b410 dereferences 0x68edcc; the
// original faults). The widget pointers and the table 0x6e1240 are fixed: table
// entries exist for indexes 0..11 only; the widgets are twelve distinct objects.
const (
	prsM        = 8
	prsW        = 12
	prsArena    = 0x20000000
	prsChars    = prsArena
	prsParty    = prsArena + 0x11000
	prsBat      = prsArena + 0x15000
	prsCharSize = 0x1862
	prsMsgBuf   = 0x64bf5c // sprintf buffer handed to 0x58aad0
	prsMsgArg1  = 8
)

type prsCall struct{ a [6]int64 }

func prsNames() (in, out []string) {
	add := func(l *[]string, f string, a ...any) { *l = append(*l, fmt.Sprintf(f, a...)) }
	in = append(in, "partyPtr", "a52")
	for _, p := range []string{"pres", "b01_", "a161_"} {
		for i := 0; i < prsM; i++ {
			add(&in, "%s%d", p, i)
		}
	}
	in = append(in, "idxLo", "idxHi", "tickBase", "mode", "batPtr", "batMask", "gate")
	for _, p := range []string{"expr", "time", "f99_", "f9a_"} {
		for i := 0; i < prsM; i++ {
			add(&in, "%s%d", p, i)
		}
	}
	for _, p := range []string{"widIdx", "wid35_", "wid38_", "wobj60_"} {
		for k := 0; k < prsW; k++ {
			add(&in, "%s%d", p, k)
		}
	}
	in = append(in, "charBase")
	for k := 0; k < ccaWLen; k++ {
		add(&in, "ft%d", k)
	}
	for i := 0; i < prsM; i++ {
		for k := 0; k < ccaSlots; k++ {
			add(&in, "fs%d_%d", i, k)
		}
	}
	for i := 0; i < prsM; i++ {
		add(&in, "fb09_%d", i)
		add(&in, "fb0d_%d", i)
		add(&in, "fb11_%d", i)
	}
	out = append(out, "n522")
	for a := 1; a <= 6; a++ {
		for k := 0; k < prsM; k++ {
			add(&out, "c522_%d_%d", a, k)
		}
	}
	out = append(out, "n58")
	for a := 1; a <= 3; a++ {
		for k := 0; k < prsM; k++ {
			add(&out, "c58_%d_%d", a, k)
		}
	}
	for _, p := range []string{"oexpr", "otime", "of99_", "of9a_"} {
		for i := 0; i < prsM; i++ {
			add(&out, "%s%d", p, i)
		}
	}
	out = append(out, "obatMask")
	for _, p := range []string{"owid35_", "owid38_", "owobj60_"} {
		for k := 0; k < prsW; k++ {
			add(&out, "%s%d", p, k)
		}
	}
	return
}

func init() {
	in, out := prsNames()
	Nodes["005029e0"] = Node{Addr: "005029e0", In: in, Out: out,
		F:   postRestStatusDurationNode,
		Emu: func(in []int64, r Rand) []int64 { return postRestStatusDuration(in, r, false) }}
}

func postRestStatusDurationNode(in []int64, r Rand) []int64 {
	return postRestStatusDuration(in, r, true)
}

func postRestStatusDuration(in []int64, r Rand, real bool) []int64 {
	o, _, _ := prsRun(in, r, real)
	return o
}

// prsRun also returns every member's +0xb01 after loop 1 and, with real, the
// callee world (its b11 holds the +0xb11 that 0x522fe0 leaves behind).
func prsRun(in []int64, r Rand, real bool) ([]int64, []int64, *prsWorld) {
	in0, _ := prsNames()
	if len(in) != len(in0) {
		panic("post_rest_status_duration: wrong input count")
	}
	// input index bases (same order as prsNames)
	const (
		iParty = 0
		iA52   = 1
		iPres  = 2
		iB01   = iPres + prsM
		iA161  = iB01 + prsM
		iIdxLo = iA161 + prsM
		iIdxHi = iIdxLo + 1
		iTick  = iIdxHi + 1
		iMode  = iTick + 1
		iBat   = iMode + 1
		iBatM  = iBat + 1
		iGate  = iBatM + 1
		iExpr  = iGate + 1
		iTime  = iExpr + prsM
		iF99   = iTime + prsM
		iF9a   = iF99 + prsM
		iWIdx  = iF9a + prsM
		iW35   = iWIdx + prsW
		iW38   = iW35 + prsW
		iWObj  = iW38 + prsW
		iChar  = iWObj + prsW
		iFt    = iChar + 1
		iFs    = iFt + ccaWLen
		iFb    = iFs + prsM*ccaSlots
	)
	if in[iParty] != 0 && in[iParty] != prsParty || in[iBat] != 0 && in[iBat] != prsBat || in[iChar] != prsChars {
		panic("post_rest_status_duration: pointer outside the fixture")
	}
	pres := func(m int) bool { return in[iPres+m]&0xff != 0 }
	b01 := make([]int64, prsM)
	for m := range b01 {
		b01[m] = in[iB01+m] & 0xffffffff
	}
	var c522 []prsCall
	var c58 []prsCall
	var w *prsWorld
	if real {
		w = newPrsWorld(in[iFt:iFs], in[iFs:iFb], in[iFb:], pres, in[iChar])
	}
	if in[iParty] != 0 && in[iA52]&0xff != 0 {
		for m := 0; m < prsM; m++ {
			if !pres(m) || uint32(b01[m]) >= 0x12 {
				continue
			}
			x := uint32(in[iA161+m]) * 70 / 100
			v := int32(uint32(r.R(100)) + (0xffffffec - x))
			if v > 0 {
				t := int64(v/30 + 1)
				c522 = append(c522, prsCall{[6]int64{int64(m), 0xf, t, 0, 0, 0}})
				if real {
					b01[m] = w.apply(m, t, b01[m], r)
				}
			} else {
				if uint32(in[iMode]) == 7 && in[iBat] == 0 {
					panic("post_rest_status_duration: battle mode with a null battle object (0x58b410 faults)")
				}
				c58 = append(c58, prsCall{[6]int64{prsMsgArg1, prsMsgBuf, prsChars + int64(m)*prsCharSize + 5}})
			}
		}
	}
	expr, tm, f99, f9a := make([]int64, prsM), make([]int64, prsM), make([]int64, prsM), make([]int64, prsM)
	for m := 0; m < prsM; m++ {
		expr[m], tm[m], f99[m], f9a[m] = in[iExpr+m]&0xffffffff, in[iTime+m]&0xffffffff, in[iF99+m]&0xff, in[iF9a+m]&0xff
	}
	batMask := in[iBatM] & 0xffffffff
	for m := 0; m < prsM; m++ {
		if !pres(m) {
			continue
		}
		f99[m] = 0
		if uint32(b01[m]) >= 0xf || in[iGate]&0xff != 0 {
			expr[m] = 2
		} else {
			expr[m] = 1
		}
		if r.R(2) != 0 {
			tm[m] = int64(uint32(r.R(5000) + 5000 + int(uint32(in[iTick]))))
		}
		f9a[m] = 1
		if uint32(in[iMode]) == 7 && in[iBat] != 0 {
			batMask |= 1 << uint(m)
		}
	}
	w35, w38, w60 := make([]int64, prsW), make([]int64, prsW), make([]int64, prsW)
	for k := 0; k < prsW; k++ {
		w35[k], w38[k], w60[k] = in[iW35+k]&0xff, in[iW38+k]&0xff, in[iWObj+k]&0xffffffff
	}
	for k := 0; k < prsW; k++ {
		w35[k] = 1
		idx := int32(in[iWIdx+k])
		if idx >= 0 && idx < prsW && w60[idx]&1 == 0 { // table 0x6e1240 holds objects for 0..11 only
			w60[idx] |= 1
			w38[k] = 1
		}
	}
	var o []int64
	o = append(o, int64(len(c522)))
	for a := 0; a < 6; a++ {
		for k := 0; k < prsM; k++ {
			o = append(o, prsRec(c522, k, a))
		}
	}
	o = append(o, int64(len(c58)))
	for a := 0; a < 3; a++ {
		for k := 0; k < prsM; k++ {
			o = append(o, prsRec(c58, k, a))
		}
	}
	for _, l := range [][]int64{expr, tm, f99, f9a} {
		o = append(o, l...)
	}
	o = append(o, batMask)
	return append(append(append(o, w35...), w38...), w60...), b01, w
}

// prsRec is the stub record of call k, argument a: slots of calls that did not
// happen keep the case's zero (the stub memory starts at zero).
func prsRec(c []prsCall, k, a int) int64 {
	if k < len(c) {
		return c[k].a[a]
	}
	return 0
}

// prsWorld is F's callee world for the real 0x522fe0.
type prsWorld struct {
	w        []int64 // shared W template (all W cells)
	slots    [prsM][ccaSlots]int64
	b09      [prsM]int64
	b0d, b11 [prsM]int64
	pres     [prsM]int64
	base     int64
}

func newPrsWorld(ft, fs, fb []int64, pres func(int) bool, base int64) *prsWorld {
	w := &prsWorld{w: append([]int64(nil), ft...), base: base}
	for m := 0; m < prsM; m++ {
		copy(w.slots[m][:], fs[m*ccaSlots:(m+1)*ccaSlots])
		w.b09[m], w.b0d[m], w.b11[m] = fb[3*m], fb[3*m+1], fb[3*m+2]
		if pres(m) {
			w.pres[m] = 1
		}
	}
	return w
}

// apply runs ccaApply(m, 0xf, t) and returns member m's new +0xb01.
func (p *prsWorld) apply(m int, t, b01 int64, r Rand) int64 {
	v := make([]int64, 6+ccaWLen)
	v[0], v[1], v[2] = int64(m), 0xf, t
	copy(v[6:], p.w)
	c := v[6:]
	c[ccaWBase] = p.base
	c[ccaWWhoPres] = p.pres[m]
	for j := 0; j < prsM; j++ {
		c[ccaWPres+j] = p.pres[j]
	}
	copy(c[ccaWS:ccaWS+ccaSlots], p.slots[m][:])
	c[ccaWB01], c[ccaWB09], c[ccaWB0d], c[ccaWB11] = b01, p.b09[m], p.b0d[m], p.b11[m]
	s := ccaNew(v, 6, r, true)
	ccaApply(s, int64(m), 0xf, uint32(t), 0, false)
	copy(p.w, s.v[6:])
	copy(p.slots[m][:], s.v[6+ccaWS:6+ccaWS+ccaSlots])
	p.b09[m], p.b11[m] = s.g(ccaWB09), s.g(ccaWB11)
	return int64(s.u(ccaWB01))
}
