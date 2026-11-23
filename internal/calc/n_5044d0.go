package calc

import "fmt"

// environment_stamina_hp_drain 0x5044d0(ticks): per-tick environmental drain on
// the party (build/wiz8.asm:350612..350654). Nothing happens while byte 0x687500
// is non-zero. Otherwise, for each of the eight members whose presence byte
// (*0x685178 + 0x106*i) is non-zero, hit points +0xb11 non-zero, state +0xb01 < 0x12
// (unsigned), race +0x71 != 0xf and who does not hold item 0x1e5 in the 12 slots at
// +0xf5d+12k (0x520f90(C, 0x1e5, 0, 0, 0), mode 0), the dice record {Z=0, N=low byte
// of ticks, S=low byte of 0x687608/6+2} is rolled by 0x517970 (diceNode). A roll of
// 0 skips the spend. Else with stamina +0xb1d (signed): stamina >= roll spends the
// roll with 0x52af50(i, roll, 0, 0); otherwise, stamina > 0 first spends all of it
// with 0x52af50(i, stamina, 0, 0) and the rest of the roll goes down; then
// 0x52a890(i, rest, 0, f, 0, 0, 0) with f = 1 only when both bytes 0x683f94 and
// 0x68510c are non-zero. After each qualifying member, when byte 0x6875a3 is clear it
// is set and 0x58ac00(8, *(*0x68c09c + 0x768), -1, -1, 0) is called once. If any
// member qualified, 0x687608 += ticks (wrapping); else 0x687608, 0x6875a3 and 0x6875a6
// are cleared.
//
// Emu address-stubs the D4 callees 0x52af50 and 0x52a890 and the excluded
// presentation helper 0x58ac00: the node never reads their eax or writes (the stamina
// and hit points it reads belong to the member before its own calls), so ret 0x5a5a5a5a
// in a scratch copy stays diff=0 and the outputs are each stub's call count and every
// call's arguments. No Emu/F split is needed for the same reason. 0x517970 and 0x520f90
// run natively. No fixture caps: every input is the full width of its field.
const (
	esdMembers = 8
	esdItem    = 0x1e5
	esdFatigue = 16 // 0x52af50 calls: two per member
	esdDamage  = 8
)

func esdNames() (in, out []string) {
	in = []string{"ticks", "g687500", "g687608", "g6875a3", "g6875a6", "g683f94", "g68510c", "ui768"}
	for i := 0; i < esdMembers; i++ {
		in = append(in, fmt.Sprintf("member%dPresent", i))
	}
	for i := 0; i < esdMembers; i++ {
		in = append(in, fmt.Sprintf("member%dHp", i), fmt.Sprintf("member%dState", i), fmt.Sprintf("member%dRace", i), fmt.Sprintf("member%dStamina", i))
		for k := 0; k < 12; k++ {
			in = append(in, fmt.Sprintf("member%dItem%d", i, k))
		}
	}
	out = []string{"g687608", "g6875a3", "g6875a6", "n52af50"}
	for _, s := range []struct {
		pre   string
		args  int
		calls int
	}{{"52af50", 4, esdFatigue}, {"52a890", 7, esdDamage}, {"58ac00", 5, 1}} {
		if s.pre != "52af50" {
			out = append(out, "n"+s.pre)
		}
		for k := 1; k <= s.args; k++ {
			for c := 0; c < s.calls; c++ {
				out = append(out, fmt.Sprintf("a%s_%d_%d", s.pre, k, c))
			}
		}
	}
	return
}

func init() {
	in, out := esdNames()
	Nodes["005044d0"] = Node{Addr: "005044d0", In: in, Out: out, F: environmentStaminaHpDrainNode}
}

func environmentStaminaHpDrainNode(in []int64, r Rand) []int64 {
	names, _ := esdNames()
	if len(in) != len(names) {
		panic(fmt.Sprintf("environment stamina hp drain: expected %d inputs, got %d", len(names), len(in)))
	}
	ticks, g7500, g7608 := uint32(in[0]), in[1]&0xff, uint32(in[2])
	g75a3, g75a6, f94, f10c, ui := in[3]&0xff, uint32(in[4]), in[5]&0xff, in[6]&0xff, uint32(in[7])
	var fat [4][esdFatigue]int64
	var dmg [7][esdDamage]int64
	var msg [5]int64
	var nFat, nDmg, nMsg int64
	finish := func() []int64 {
		o := []int64{int64(g7608), g75a3, int64(g75a6), nFat}
		for k := range fat {
			o = append(o, fat[k][:]...)
		}
		o = append(o, nDmg)
		for k := range dmg {
			o = append(o, dmg[k][:]...)
		}
		o = append(o, nMsg)
		return append(o, msg[:]...)
	}
	if g7500 != 0 { // 0x5044e1
		return finish()
	}
	spend := func(who int, amount int32) { // 0x52af50(who, amount, 0, 0)
		for k, v := range []int64{int64(who), int64(uint32(amount)), 0, 0} {
			fat[k][nFat] = v
		}
		nFat++
	}
	any := false
	for i := 0; i < esdMembers; i++ {
		base := 8 + esdMembers + i*16
		m := in[base : base+16]
		if in[8+i]&0xff == 0 || uint32(m[0]) == 0 || uint32(m[1]) >= 0x12 || uint32(m[2]) == 0xf {
			continue
		}
		held := false // 0x520f90 mode 0: the 12 slots at +0xf5d, 12 bytes apart
		for k := 0; k < 12; k++ {
			held = held || uint32(m[4+k]) == esdItem
		}
		if held {
			continue
		}
		any = true
		roll := int32(diceNode([]int64{0, int64(ticks & 0xff), int64((g7608/6 + 2) & 0xff)}, r)[0])
		if roll != 0 {
			sta := int32(m[3])
			if sta >= roll { // 0x504595
				spend(i, roll)
			} else {
				if sta > 0 {
					spend(i, sta)
					roll -= sta
				}
				flag := int64(0)
				if f94 != 0 && f10c != 0 {
					flag = 1
				}
				for k, v := range []int64{int64(i), int64(uint32(roll)), 0, flag, 0, 0, 0} {
					dmg[k][nDmg] = v
				}
				nDmg++
			}
		}
		if g75a3 == 0 { // 0x5045df
			g75a3 = 1
			msg = [5]int64{8, int64(ui), 0xffffffff, 0xffffffff, 0}
			nMsg++
		}
	}
	if any {
		g7608 += ticks
	} else {
		g7608, g75a3, g75a6 = 0, 0, 0
	}
	return finish()
}
