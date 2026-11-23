package calc

// randomPartyMemberNode picks the (n+1)th eligible party slot, n = R(8), scanning
// round-robin from slot 0 (0x518150). Args: selA/selB (2 = loosened), excludeSlot,
// classKey (signed byte, -1 = no filter). Per slot: presence byte (table 0x685178,
// stride 0x106), flag 0xb11, level 0xb01, key 0x75 (characters 0x685174, stride 0x1862).
// Retries with selB then selA promoted to 2; returns -1 when nobody qualifies.
func randomPartyMemberNode(in []int64, r Rand) []int64 {
	if len(in) != 36 {
		panic("random party member: expected 36 inputs")
	}
	a, b, exclude, key := uint32(in[0]), uint32(in[1]), in[2], in[3]
	present, flag, level, slotKey := in[4:12], in[12:20], in[20:28], in[28:36]
	for {
		skip := r.R(8)
		slot := 0
		for step := 0; step <= 8; step++ {
			i := slot
			ok := present[i] != 0 && int64(i) != exclude
			if ok && !(uint32(flag[i]) != 0 && uint32(level[i]) < 0x12) && a != 2 {
				ok = false
			}
			if ok && key != -1 && key == slotKey[i] {
				ok = false
			}
			if ok && uint32(level[i]) >= 0xf && b != 2 {
				ok = false
			}
			if ok {
				if skip == 0 {
					return []int64{int64(i)}
				}
				skip--
				step = 0
			}
			slot = (slot + 1) % 8
		}
		if b == 1 {
			b = 2
		} else if a == 1 {
			a = 2
		} else {
			return []int64{-1}
		}
	}
}

func init() {
	in := []string{"selA", "selB", "excludeSlot", "classKey"}
	for _, p := range []string{"present", "flag11_", "level01_", "key75_"} {
		for s := 0; s < 8; s++ {
			in = append(in, p+string(rune('0'+s)))
		}
	}
	Nodes["00518150"] = Node{Addr: "00518150", In: in, Out: []string{"slot"}, F: randomPartyMemberNode}
}
