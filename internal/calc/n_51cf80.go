package calc

func init() {
	Nodes["0051cf80"] = Node{Addr: "0051cf80",
		In: []string{"idx", "itemType", "itemFlags", "itemSlotByte", "handA", "handB", "handC", "handD",
			"decoy3d", "decoy3f", "decoy40", "decoy42", "decoy45", "decoy47"},
		Out: []string{"slotMask"}, F: itemEquipSlotMaskNode}
}

// Item definition at global 0x68516c, stride 269 (+0x3e type, +0x41 flags,
// +0x46 slot byte). hand A..D are the caller's byte flags (low byte only).
// Result is the 16-bit mask returned in AX. For types 0..3 a slot byte of
// 0xff only prints a data-error message (0x5182e0, formatting only) and
// yields 0.
func itemEquipSlotMaskNode(in []int64, r Rand) []int64 {
	if len(in) != 14 {
		panic("item equip slot mask: expected fourteen inputs (last six are unread decoy bytes)")
	}
	typ, flags, slot := in[1]&0xff, in[2]&0xff, in[3]&0xff
	a, b, c, d := in[4]&0xff != 0, in[5]&0xff != 0, in[6]&0xff != 0, in[7]&0xff != 0
	m := int64(0)
	switch typ {
	case 0, 1, 2, 3:
		if slot == 0xff {
			return []int64{0}
		}
		if flags&4 != 0 { // flag bit 2
			if a {
				m = 0x40
			}
			if b {
				m |= 0x100
			}
		} else {
			m = 0x140
			if flags&8 != 0 { // flag bit 3
				if c {
					m = 0x1c0
				}
				if d {
					m |= 0x200
				}
			}
		}
	case 4, 5:
		if c {
			m = 0x80
		}
		if d {
			m |= 0x200
		}
	case 6:
		m = 0x10
	case 7:
		m = 0x400
	case 8:
		m = 1
	case 9:
		m = 0x20
	case 10:
		m = 0x800
	case 11:
		m = 6
	case 12:
		m = 8
	}
	return []int64{m}
}
