package calc

import "fmt"

// spell_timed_buff 0x54e3f0(effect, type): duration = effect+0xdc * effect+0xd0
// + effect+0xd8 (u32 wrap, exe 0x54e3f7..0x54e404). Unless it is the 9999
// sentinel (0x54e40a) the duration gets +1, one R(4) (0 bumps it when >1, 1 when
// <=2; 0x54e422..0x54e43a) and the child 0x517f90 percentage
// d += d*effect+0xd4/100 (low 32 bits of the product, unsigned division).
// Then every character id in the array at effect+0xfc (count effect+0xf4,
// signed; <=0 runs nothing) and every monster id at effect+0xec (count
// effect+0xe4) gets the buff. For type 5 with the target's opposite-effect
// time (character +0xa15 = condition slot 5, monster +0x6b) non-zero the
// target's time is decayed by the full duration (0x5236a0 / 0x524110) and the
// remainder (duration - time, signed > 0) is applied as type 5; otherwise the
// buff applies with the given type and the whole duration (0x523940 / 0x5242b0,
// args: id, type, effect+0xd0, duration, effect+0xd4).
//
// Emu address-stubs the four D4 callees (composition ruling 2026-10-09) and
// returns every call's arguments (up to stbMax calls each, call k in cell k)
// and the call count of each; their return values are ignored by every call
// site. F instead runs the real decay callees 5236a0 (ccdRun) and 524110
// (monsterConditionDecayNode) and feeds the new +0xa15 / +0x6b back, so a target
// id repeated in one array sees the decayed time. The apply callees 523940 and
// 5242b0 write effect records and pools only, never +0xa15 / +0x6b or anything
// else this node reads, so F does not run them (their outputs are the recorded
// args). Real decay needs the member present (else its assert: panic) and the
// monster's current HP (+0x2b; zero leaves the time unchanged).
// Monster ids resolve through 0x4e5550/0x4e5620: first match in the main list,
// else in the reserve list; an unknown id and a character id of -1 end in
// srAssertFail, so they panic here (blocks 0x54e486, 0x54e578).
// Fixture caps: at most stbMax ids per array, character ids 0..7.
const (
	stbMax     = 8
	stbMembers = 8
	stbMain    = 5 // main monster list records
	stbRes     = 2 // reserve monster list records
	stbRecs    = stbMain + stbRes
)

func init() {
	in := []string{"type", "power", "percent", "bonusBase", "bonusPerPower", "charCount"}
	for j := 0; j < stbMax; j++ {
		in = append(in, fmt.Sprintf("charId%d", j))
	}
	in = append(in, "monsterCount")
	for j := 0; j < stbMax; j++ {
		in = append(in, fmt.Sprintf("monsterId%d", j))
	}
	for k := 0; k < stbMembers; k++ {
		in = append(in, fmt.Sprintf("charCond5_%d", k))
	}
	for k := 0; k < stbMain; k++ {
		in = append(in, fmt.Sprintf("listMonsterId%d", k))
	}
	for k := 0; k < stbRes; k++ {
		in = append(in, fmt.Sprintf("reserveMonsterId%d", k))
	}
	for k := 0; k < stbMain; k++ {
		in = append(in, fmt.Sprintf("listMonsterCond5_%d", k))
	}
	for k := 0; k < stbRes; k++ {
		in = append(in, fmt.Sprintf("reserveMonsterCond5_%d", k))
	}
	for k := 0; k < stbMembers; k++ {
		in = append(in, fmt.Sprintf("charPresent%d", k))
	}
	for k := 0; k < stbMain; k++ {
		in = append(in, fmt.Sprintf("listMonsterHp%d", k))
	}
	for k := 0; k < stbRes; k++ {
		in = append(in, fmt.Sprintf("reserveMonsterHp%d", k))
	}
	var out []string
	for _, c := range []struct {
		name string
		args int
	}{{"charDecay", 3}, {"charApply", 5}, {"monDecay", 3}, {"monApply", 5}} {
		for i := 1; i <= c.args; i++ {
			for k := 0; k < stbMax; k++ {
				out = append(out, fmt.Sprintf("%s.arg%d.call%d", c.name, i, k))
			}
		}
		out = append(out, c.name+".count")
	}
	Nodes["0054e3f0"] = Node{Addr: "0054e3f0", In: in, Out: out, F: spellTimedBuffNode,
		Emu: func(in []int64, r Rand) []int64 { return stbRun(in, r, false) }}
}

func spellTimedBuffNode(in []int64, r Rand) []int64 { return stbRun(in, r, true) }

// stbCalls collects one callee's calls: cell k of each argument is call k's.
type stbCalls struct {
	args [5][stbMax]uint32
	n    int
}

func (c *stbCalls) add(a ...uint32) {
	for i, v := range a {
		c.args[i][c.n] = v
	}
	c.n++
}

func (c *stbCalls) emit(out []int64, na int) []int64 {
	for i := 0; i < na; i++ {
		for k := 0; k < stbMax; k++ {
			out = append(out, int64(c.args[i][k]))
		}
	}
	return append(out, int64(c.n))
}

func stbRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != 7+2*stbMax+stbMembers+2*stbRecs+stbMembers+stbRecs {
		panic("spell_timed_buff: wrong input count")
	}
	for _, v := range in {
		if v < -2147483648 || v > 0xffffffff {
			panic("spell_timed_buff: fields must fit 32 bits")
		}
	}
	typ, power, pct := uint32(in[0]), uint32(in[1]), uint32(in[2])
	dur := uint32(in[4])*power + uint32(in[3])
	if dur != 9999 {
		dur++
		switch r.R(4) { // 0x54e41a
		case 0:
			if dur > 1 {
				dur++
			}
		case 1:
			if dur <= 2 {
				dur++
			}
		}
		dur += dur * pct / 100 // 0x517f90
	}
	nChar, chars := int32(in[5]), in[6:6+stbMax]
	nMon, mons := int32(in[6+stbMax]), in[7+stbMax:7+2*stbMax]
	if nChar > stbMax || nMon > stbMax {
		panic("spell_timed_buff: fixture caps target arrays at 8 ids")
	}
	o := 7 + 2*stbMax
	cond := append([]int64(nil), in[o:o+stbMembers]...)
	o += stbMembers
	ids := in[o : o+stbRecs]
	o += stbRecs
	t5 := append([]int64(nil), in[o:o+stbRecs]...)
	o += stbRecs
	present := in[o : o+stbMembers]
	o += stbMembers
	hp := in[o : o+stbRecs]
	var cd, ca, md, ma stbCalls
	for j := int32(0); j < nChar; j++ {
		id := uint32(chars[j])
		if id >= stbMembers {
			panic("spell_timed_buff: assert or character outside the 8-member fixture") // 0x54e486
		}
		if typ == 5 && cond[id] != 0 {
			old := uint32(cond[id])
			cd.add(id, typ, dur)
			if real {
				if present[id]&0xff == 0 {
					panic("spell_timed_buff: assert, member absent (0x5236a0)")
				}
				v := make([]int64, ccdIn)
				v[0], v[1], v[2], v[3], v[4+5] = int64(id), 5, int64(dur), 1, cond[id]
				cond[id] = ccdRun(v, r, true)[5]
			}
			if rem := dur - old; int32(rem) > 0 {
				ca.add(id, 5, power, rem, pct)
			}
			continue
		}
		ca.add(id, typ, power, dur, pct)
	}
	for j := int32(0); j < nMon; j++ {
		id, k := uint32(mons[j]), -1
		for i := 0; i < stbRecs && k < 0; i++ {
			if uint32(ids[i]) == id {
				k = i
			}
		}
		if k < 0 {
			panic("spell_timed_buff: monster id in neither list (srAssertFail in 0x4e5550)")
		}
		if typ == 5 && t5[k] != 0 {
			old := uint32(t5[k])
			md.add(id, typ, dur)
			if real {
				v := make([]int64, 26)
				v[0], v[1], v[2], v[3], v[4+5] = int64(id), 5, int64(dur), hp[k], t5[k]
				t5[k] = monsterConditionDecayNode(v, r)[5]
			}
			if rem := dur - old; int32(rem) > 0 {
				ma.add(id, 5, power, rem, pct)
			}
			continue
		}
		ma.add(id, typ, power, dur, pct)
	}
	var out []int64
	out = cd.emit(out, 3)
	out = ca.emit(out, 5)
	out = md.emit(out, 3)
	return ma.emit(out, 5)
}
