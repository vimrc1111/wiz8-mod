package calc

func init() {
	Nodes["0053c990"] = Node{Addr: "0053c990", In: append(append([]string{}, cgInNames...), "a_w", "a_u6", "a_m"),
		Out: append(append([]string{}, cgOutNames...), "ret"), F: characterAutoAttackConfigurationNode}
}

func characterAutoAttackConfigurationNode(in []int64, r Rand) []int64 {
	return cgRun(in, 3, 4, func(w *cgW, a []int64) int64 { return w.autoFire(a[0], a[1], a[2]) })
}

// autoFire is 0x53c990(x, u6, m): fire the auto-attack toggle; returns the
// monster id chosen or -1.
func (w *cgW) autoFire(x, u6, m int64) int64 {
	S, C, D := w.S(x), w.C(x), w.D(x)
	cb := w.u32(0x6836a8)
	if w.u8(0x685115) == 0 || w.u8(0x6837cb+0x118*x) != 0 || w.u8(S+0xf5) != 0 ||
		w.u8(cb) != 0 && w.u8(D+0x4c) != 0 && w.u32(D+0x18) != 0 ||
		w.st(cgS51d180, C+0xfbd)&0xff != 0 || w.st(cgS51d180, C+0xfc9)&0xff != 0 ||
		w.st(cgS51d3b0, x, 0, 0)&0xff != 1 {
		return -1
	}
	r := w.st(cgS53c720, x, u6, m)
	ret := r
	if r == -1 {
		if w.st(cgS51d3b0, x, 0, 0)&0xff != 0 {
			return -1
		}
		w.notify(x)
	} else {
		w.notify(x)
		ctx := m
		if m == 6 {
			ctx = w.resolve(x)
		}
		w.kind(x, ctx)
		rec := w.rec(w.rptr(x, ctx))
		if rec[0] == 3 {
			if e := w.find(rec[2], false); e != -1 && w.st(cgS5194e0, x, 2, w.entry(e), m, 0)&0xff != 0 {
				ret = rec[2]
			}
		}
	}
	w.p8(S+0x105, w.u8(S+0x105)^1)
	w.refreshEquip(C+0xfa5, C, 1)
	w.refreshEquip(C+0xfb1, C, 1)
	return ret
}
