package calc

func init() {
	Nodes["0052b6d0"] = Node{
		Addr: "0052b6d0",
		In: []string{"characterIndex", "realmIndex", "drainAmount", "notifyFlag", "notificationMode",
			"spPoolsEnabled", "fireCurrentSP", "waterCurrentSP", "airCurrentSP", "earthCurrentSP",
			"mentalCurrentSP", "divineCurrentSP"},
		Out: []string{"fireCurrentSP", "waterCurrentSP", "airCurrentSP", "earthCurrentSP",
			"mentalCurrentSP", "divineCurrentSP"},
		F: characterRealmSpDrainNode,
	}
}

// characterRealmSpDrainNode drains one realm SP pool of the selected
// character (exe 0x52b6d0). Inputs: index, realm 0..5, u32 amount, notify
// byte, notification mode byte (0x687500), pool-enabled word (+0xb11) and the
// six u32 current-SP words at +0xb45. Output is the six current-SP words.
// The notification calls (0x58aad0 / 0x590950) and refresh (0x55ee30) have no
// effect on these words.
func characterRealmSpDrainNode(in []int64, r Rand) []int64 {
	if len(in) != 12 || in[0] < 0 || in[0] > 7 || in[1] < 0 || in[1] > 5 {
		panic("realm SP drain: expected character 0..7, realm 0..5 and ten more inputs")
	}
	for _, v := range in[2:] {
		if v < 0 || v > 4294967295 {
			panic("realm SP drain: input outside u32")
		}
	}
	out := append([]int64(nil), in[6:]...)
	cur := uint32(in[6+in[1]])
	amt := uint32(in[2])
	// 0x52b6fb, 0x52b70f: pools disabled or empty pool -> nothing.
	if in[5] == 0 || cur == 0 {
		return out
	}
	// 0x52b717: nonzero mode only notifies, no pool write.
	if in[4]&0xff != 0 {
		return out
	}
	// 0x52b74e: a pool with the sign bit set is left alone.
	if int32(cur) < 0 {
		return out
	}
	// 0x52b756..0x52b75e: clamp the amount to the pool (unsigned compare;
	// the pool is non-negative here).
	if amt >= cur {
		amt = cur
	}
	// 0x52b762: zero amount does nothing.
	if amt == 0 {
		return out
	}
	// 0x52b787..0x52b791: the 0x52b76d assert is unreachable since amt <= cur.
	out[in[1]] = int64(cur - amt)
	return out
}
