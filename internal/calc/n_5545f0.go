package calc

import "fmt"

func init() {
	var in, out []string
	for c := 0; c < 8; c++ {
		in = append(in,
			fmt.Sprintf("character%dPresent", c), fmt.Sprintf("character%dPoolEnabled", c), fmt.Sprintf("character%dMainStatus", c),
			fmt.Sprintf("previous%dRegion", c), fmt.Sprintf("previous%dSlot", c),
			fmt.Sprintf("current%dRegion", c), fmt.Sprintf("current%dPrevRegion", c), fmt.Sprintf("current%dSlot", c),
			fmt.Sprintf("current%dFacing", c), fmt.Sprintf("current%dExtraA", c), fmt.Sprintf("current%dExtraB", c))
	}
	for _, side := range []string{"previous", "current"} {
		for w := 0; w < 33; w++ {
			out = append(out, fmt.Sprintf("%sWord%d", side, w))
		}
	}
	Nodes["005545f0"] = Node{Addr: "005545f0", In: in, Out: out, F: formationReconcileNode}
}

// Formation block (0x554580..0x5545ad): 33 dwords = 15 slot bytes (3 per region,
// 5 regions), 5 region counts at +15, then 8 member records of 12 bytes at +20
// (region, previous region, slot, facing, 8 bytes carried verbatim); empty is 255,
// fresh facing is 4 (0x5545b7). Party record stride 0x1862 (+0xb01 status,
// +0xb11 pool flag), presence stride 0x106 (0x554639..0x554654).
// Inputs per member: present byte, pool flag u32, main status u32, previous
// region/slot (the "previous" block), and the whole 12-byte "current" record.
// Outputs: both blocks after the function copies the rebuilt block into each
// (0x554935..0x55497c), 33 little-endian dwords each.
//
// Out-of-structure indexing in the original (region >= 5 or slot >= 3 used as a
// table index, which reads or writes outside the block) has no defined result, so
// calc panics there; srAssertFail paths (0x554bd0 slot/count checks) also panic.
func formationReconcileNode(in []int64, r Rand) []int64 {
	if len(in) != 88 {
		panic("formation reconcile: expected 88 inputs")
	}
	if restoreFacingError != nil {
		panic(restoreFacingError)
	}
	widths := [11]int64{255, 1<<32 - 1, 1<<32 - 1, 255, 255, 255, 255, 255, 255, 1<<32 - 1, 1<<32 - 1}
	for i, v := range in {
		if v < 0 || v > widths[i%11] {
			panic("formation reconcile: input out of field width")
		}
	}
	// presence is read as a byte
	var cur [8][12]byte
	var nf [132]byte
	for i := 0; i < 15; i++ {
		nf[i] = 255
	}
	for c := 0; c < 8; c++ {
		nf[20+12*c], nf[21+12*c], nf[22+12*c], nf[23+12*c] = 255, 255, 255, 4
	}
	for c := 0; c < 8; c++ {
		m := in[11*c : 11*c+11]
		cur[c] = [12]byte{byte(m[5]), byte(m[6]), byte(m[7]), byte(m[8]),
			byte(m[9]), byte(m[9] >> 8), byte(m[9] >> 16), byte(m[9] >> 24),
			byte(m[10]), byte(m[10] >> 8), byte(m[10] >> 16), byte(m[10] >> 24)}
	}
	for c := 0; c < 8; c++ {
		m := in[11*c : 11*c+11]
		if m[0] == 0 || m[1] == 0 || m[2] >= 13 {
			continue
		}
		q, s := byte(m[3]), byte(m[4])
		differ := q != cur[c][0]
		frMove(&nf, c, q, s, differ)
		if !differ {
			nf[20+12*c+3] = cur[c][3]
		}
	}
	for c := 0; c < 8; c++ {
		m := in[11*c : 11*c+11]
		if m[0] == 0 || (m[1] != 0 && m[2] <= 12) {
			continue
		}
		p := 20 + 12*c
		if m[2] >= 18 {
			copy(nf[p:p+12], cur[c][:])
			continue
		}
		region, slot := int(cur[c][0]), int(cur[c][2])
		if region >= 5 || slot >= 3 {
			panic("formation reconcile: current record region/slot outside the formation block")
		}
		if nf[15+region] < 3 {
			if nf[3*region+slot] != 255 {
				slot = -1
				for s := 0; s < 3; s++ {
					if nf[3*region+s] == 255 {
						slot = s
						break
					}
				}
				if slot < 0 {
					panic("formation reconcile: region count below 3 but no empty slot")
				}
			}
			frMove(&nf, c, byte(region), byte(slot), false)
			nf[p+3] = cur[c][3]
			continue
		}
		placed := false
		for _, q := range []int{4, 2, 3} { // 0x55486b..0x55487c
			for s := 0; s < 3; s++ {
				if nf[3*q+s] == 255 {
					frMove(&nf, c, byte(q), byte(s), true)
					placed = true
					break
				}
			}
			if placed {
				break
			}
		}
		if !placed {
			panic("formation reconcile: no empty slot in regions 4, 2, 3")
		}
	}
	for q := 0; q < 5; q++ {
		frCompact(&nf, q)
	}
	out := make([]int64, 66)
	for w := 0; w < 33; w++ {
		v := int64(nf[4*w]) | int64(nf[4*w+1])<<8 | int64(nf[4*w+2])<<16 | int64(nf[4*w+3])<<24
		out[w], out[33+w] = v, v
	}
	return out
}

// frMove is 0x554bd0 with the notify and compact arguments off (the callers here
// never turn them on; notification is gated by the scene flag 0x68517c).
func frMove(f *[132]byte, who int, q, s byte, turn bool) {
	if who < 0 || who >= 8 {
		panic("formation reconcile: invalid member")
	}
	p := 20 + 12*who
	if old := f[p]; old != 255 {
		if old >= 5 || f[p+2] >= 3 {
			panic("formation reconcile: source outside the formation block")
		}
		i := 3*int(old) + int(f[p+2])
		if f[i] == 255 || f[15+old] == 0 { // 0x554c02, 0x554c27 assertions
			panic("formation reconcile: inconsistent source")
		}
		f[i] = 255
		f[15+old]--
	}
	f[p], f[p+2] = q, s
	if q == 255 {
		return
	}
	if q >= 5 || s >= 3 {
		panic("formation reconcile: destination outside the formation block")
	}
	i := 3*int(q) + int(s)
	if f[i] != 255 || f[15+q] >= 3 { // 0x554c70, 0x554c8f assertions
		panic("formation reconcile: inconsistent destination")
	}
	f[i] = byte(who)
	f[15+q]++
	if turn && restoreFacing[q] != 255 {
		f[p+3] = restoreFacing[q]
	}
}

// frCompact is 0x554dd0: slide members of one region toward the front.
func frCompact(f *[132]byte, q int) {
	switch f[15+q] {
	case 1:
		for s := 0; s < 3; s++ {
			if f[3*q+s] != 255 {
				frMove(f, int(f[3*q+s]), byte(q), 0, true)
				return
			}
		}
	case 2:
		hole := -1
		for s := 2; s >= 0; s-- {
			if f[3*q+s] == 255 {
				hole = s
			} else if hole != -1 {
				frMove(f, int(f[3*q+s]), byte(q), byte(hole), true)
				hole = s
			}
		}
	}
}
