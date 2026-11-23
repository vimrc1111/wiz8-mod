package calc

import "fmt"

// Flat layout of the 0x552530 node. Offsets are exe field offsets; see
// notes/re/iface/552530.tsv. Party: 9 effect records at P+0x7c1 and 6 aura
// records at P+0x85a (17 bytes each: active u8, spell u32 +1, strength u32 +5,
// reserved u32 +9, duration u32 +0xd). Monster: same records at status area
// +0x3e (9) and +0xd7 (6).
const (
	cbShared   = 2
	cbPEffect  = cbShared + 48
	cbPAura    = cbPEffect + 45
	cbPMod     = cbPAura + 30
	cbChars    = cbPMod + 103
	cbTemplate = cbChars + 8*315
	cbMon      = cbTemplate + 5
	cbMonLen   = 278
	cbIn       = cbMon + 3*cbMonLen

	cbOutChars = 45 + 30 + 103
	cbOutMon   = cbOutChars + 8*103
	cbOutMonLn = 45 + 30 + 103 + 5
	cbOut      = cbOutMon + 3*cbOutMonLn
)

func init() {
	in := []string{"combatActive", "monsterCount"}
	for i := 0; i < 12; i++ {
		for _, f := range []string{"active", "spell", "strength", "multiplier"} {
			in = append(in, fmt.Sprintf("sharedSpell[%d].%s", i, f))
		}
	}
	var out []string
	recs := func(dst *[]string, name string, n int) {
		for i := 0; i < n; i++ {
			for _, f := range []string{"active", "spell", "strength", "reserved", "duration"} {
				*dst = append(*dst, fmt.Sprintf("%s[%d].%s", name, i, f))
			}
		}
	}
	recs(&in, "partyEffect", 9)
	recs(&in, "partyAura", 6)
	recs(&out, "partyEffect", 9)
	recs(&out, "partyAura", 6)
	for j := 0; j < 103; j++ {
		in = append(in, fmt.Sprintf("partyModifier[%d]", j))
		out = append(out, fmt.Sprintf("partyModifier[%d]", j))
	}
	for s := 0; s < 8; s++ {
		for _, f := range []string{"present", "inParty", "hp", "primaryState", "class", "race"} {
			in = append(in, fmt.Sprintf("character[%d].%s", s, f))
		}
		for _, f := range []string{"equipment", "status", "initialTotal"} {
			for j := 0; j < 103; j++ {
				in = append(in, fmt.Sprintf("character[%d].%s[%d]", s, f, j))
			}
		}
	}
	for s := 0; s < 8; s++ {
		for j := 0; j < 103; j++ {
			out = append(out, fmt.Sprintf("character[%d].total[%d]", s, j))
		}
	}
	for k := 0; k < 5; k++ {
		in = append(in, fmt.Sprintf("templateAttribute[%d]", k))
	}
	for j := 0; j < 3; j++ {
		p := fmt.Sprintf("m%d", j)
		in = append(in, p+".present", p+".inCombat")
		recs(&in, p+".effect", 9)
		for k := 0; k < 20; k++ {
			in = append(in, fmt.Sprintf("%s.status[%d]", p, k))
		}
		in = append(in, p+".poison")
		for k := 0; k < 8; k++ {
			for _, f := range []string{"power", "bonus", "duration"} {
				in = append(in, fmt.Sprintf("%s.buff[%d].%s", p, k, f))
			}
		}
		for k := 0; k < 12; k++ {
			for _, f := range []string{"active", "spell", "power", "bonus"} {
				in = append(in, fmt.Sprintf("%s.shared[%d].%s", p, k, f))
			}
		}
		recs(&in, p+".aura", 6)
		for k := 0; k < 103; k++ {
			in = append(in, fmt.Sprintf("%s.modifier[%d]", p, k))
		}
		for k := 0; k < 5; k++ {
			in = append(in, fmt.Sprintf("%s.attribute[%d]", p, k))
		}
		recs(&out, p+".effect", 9)
		recs(&out, p+".aura", 6)
		for k := 0; k < 103; k++ {
			out = append(out, fmt.Sprintf("%s.modifier[%d]", p, k))
		}
		for k := 0; k < 5; k++ {
			out = append(out, fmt.Sprintf("%s.attribute[%d]", p, k))
		}
	}
	Nodes["00552530"] = Node{Addr: "00552530", In: in, Out: out, F: clearCombatBuffsNode}
}

// clearCombatBuffsNode (exe 0x552530): zero every active combat effect (9) and
// aura (6) record of the party, and of each listed monster that is present and
// in combat; the reserved word (+9) is kept. Each cleared slot rebuilds the
// derived modifiers (party: 0x50e700, monster: 0x50e8c0). A rebuild is a pure
// function of the final slot state, so one rebuild after all clears is the
// same as the original's per-slot rebuilds; without a cleared slot nothing is
// rebuilt and the input buffers pass through. The scene-effect removal
// (0x4acd80) and UI refresh (0x5af2d0, 0x562a50) touch no rule field.
// A monster slot that gets cleared must carry a spell that has a visual
// attachment class in the exe table at 0x60cffc, else 0x4acd80 asserts; that is
// a precondition of the original, not checked here.
func clearCombatBuffsNode(in []int64, r Rand) []int64 {
	if len(in) != cbIn {
		panic("zeroActive combat buffs: wrong input count")
	}
	for _, v := range in {
		if v < 0 || v > 0xffffffff {
			panic("zeroActive combat buffs: expected u32 fields")
		}
	}
	if in[1] > 3 {
		panic("zeroActive combat buffs: monsterCount above 3")
	}
	// zeroActive zeroes active, spell, strength and duration of every active record
	// (0x552539 and 0x5525cd write +0, +1, +5, +0xd) and reports whether any was active.
	zeroActive := func(rec []int64, n int) bool {
		hit := false
		for i := 0; i < n; i++ {
			e := rec[5*i : 5*i+5]
			if e[0] != 0 {
				e[0], e[1], e[2], e[4] = 0, 0, 0, 0
				hit = true
			}
		}
		return hit
	}
	pe := append([]int64(nil), in[cbPEffect:cbPEffect+45]...)
	pa := append([]int64(nil), in[cbPAura:cbPAura+30]...)
	changed := zeroActive(pe, 9)
	changed = zeroActive(pa, 6) || changed
	out := make([]int64, 0, cbOut)
	out = append(out, pe...)
	out = append(out, pa...)
	if changed {
		a := make([]int64, 2620)
		a[0] = in[0]
		copy(a[1:49], in[cbShared:cbShared+48])
		for i := 0; i < 9; i++ {
			copy(a[49+3*i:52+3*i], pe[5*i:5*i+3])
			a[51+3*i] &= 0xff // the rebuild reads the low byte of the power
		}
		for i := 0; i < 6; i++ {
			copy(a[76+4*i:79+4*i], pa[5*i:5*i+3])
			a[78+4*i] &= 0xff
			a[79+4*i] = pa[5*i+3]
		}
		copy(a[100:], in[cbChars:cbChars+8*315])
		out = append(out, partyEffectRebuildNode(a, r)...)
	} else {
		out = append(out, in[cbPMod:cbPMod+103]...)
		for s := 0; s < 8; s++ {
			out = append(out, in[cbChars+315*s+212:cbChars+315*s+315]...)
		}
	}
	for j := 0; j < 3; j++ {
		b := in[cbMon+cbMonLen*j : cbMon+cbMonLen*(j+1)]
		// b: present, inCombat, effects 45, status 20, poison, buffs 24, shared 48,
		// aura 30, modifiers 103, attributes 5.
		eff := append([]int64(nil), b[2:47]...)
		aura := append([]int64(nil), b[140:170]...)
		mod := append([]int64(nil), b[170:273]...)
		attr := append([]int64(nil), b[273:278]...)
		// 0x55258b: only listed (index < count), present and in-combat monsters.
		if int64(j) < in[1] && b[0] != 0 && b[1] != 0 {
			c := zeroActive(eff, 9)
			c = zeroActive(aura, 6) || c
			if c {
				// layout of the 0x54d100 monster model: inCombat, effects, status,
				// poison, buffs, shared, aura as (active, spell, power, bonus),
				// modifiers, attributes.
				m := make([]int64, 271)
				m[0] = b[1]
				copy(m[1:46], eff)
				copy(m[46:139], b[47:140])
				for i := 0; i < 6; i++ {
					m[139+4*i] = aura[5*i]
					m[140+4*i] = aura[5*i+1]
					m[141+4*i] = aura[5*i+2] & 0xff
					m[142+4*i] = aura[5*i+3]
				}
				mod = dispelMonsterModifiers(m, r)
				// 0x4e5d00: template attribute plus signed modifier byte, clamp 1..125.
				for k, off := range []int{0, 1, 4, 5, 6} {
					attr[k] = int64(min(125, max(1, int(in[cbTemplate+k])+int(int8(mod[12+off])))))
				}
			}
		}
		out = append(out, eff...)
		out = append(out, aura...)
		out = append(out, mod...)
		out = append(out, attr...)
	}
	return out
}
