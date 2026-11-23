package calc

// Interface prefix matches 0x50e650's 348 inputs. Buff strength and multiplier
// retain all four bytes here because 0x523b68 clears three complete u32 fields.
// The suffix supplies decay controls and the 257 existing ability/cache values
// which 0x523be7 leaves untouched. Layout sources: 0x523b30, 0x50e650,
// and notes/re/iface/50e650.tsv.
func init() {
	rebuild := Nodes["0050e650"]
	inputs := append([]string(nil), rebuild.In...)
	inputs = append(inputs, "buffIndex", "decayAmount", "lastActiveBuff", "gameMode",
		"attributeRefreshPending", "adventureRefreshMask", "inventoryRefreshMask")
	for _, name := range rebuild.Out {
		inputs = append(inputs, "initial"+string(name[0]-'a'+'A')+name[1:])
	}
	outputs := append([]string(nil), rebuild.In[98:122]...)
	outputs = append(outputs, "lastActiveBuff", "attributeRefreshPending", "adventureRefreshMask", "inventoryRefreshMask")
	outputs = append(outputs, rebuild.Out...)
	Nodes["00523b30"] = Node{Addr: "00523b30", In: inputs, Out: outputs, F: characterEffectDecayNode}
}

func characterEffectDecayNode(in []int64, r Rand) []int64 {
	// Counts and indices follow the interface documented above; eight 12-byte
	// buffs start at C+0xa65, their durations at C+0xa6d (0x523b51).
	if len(in) != 612 {
		panic("character effect decay: expected 612 interface fields")
	}
	for _, v := range in {
		if v < 0 || v > 0xffffffff {
			panic("character effect decay: fields must fit u32")
		}
	}
	if in[0] > 7 || in[348] > 7 || in[352] > 255 {
		panic("character effect decay: invalid party slot, buff index or refresh byte")
	}
	buffs := append([]int64(nil), in[98:122]...)
	last, refresh, adventure, inventory := in[350], in[352], uint32(in[353]), uint32(in[354])
	caches := append([]int64(nil), in[355:]...)
	index := int(in[348])
	duration, decay := uint32(buffs[3*index+2]), uint32(in[349])
	if duration > decay {
		buffs[3*index+2] = int64(duration - decay)
	} else {
		buffs[3*index], buffs[3*index+1], buffs[3*index+2] = 0, 0, 0
		// 0x523b8a: scan backward, always accepting slot zero even if empty.
		last = 0
		for j := 7; j > 0; j-- {
			if buffs[3*j+2] != 0 {
				last = int64(j)
				break
			}
		}
		// 0x55ee30 and 0x562a50: inventory/adventure dirty flags. The
		// rebuild's final 0x55ee30 repeats the slot bit, an idempotent union.
		switch in[351] {
		case 6:
			inventory |= 0x100 // 0x55ee57: inventory character panel.
		case 7:
			adventure |= uint32(1)<<uint(in[0]) | 0x200000 | 0x8000 // 0x523bb7.
		}
		rebuildIn := append([]int64(nil), in[:348]...)
		copy(rebuildIn[98:122], buffs)
		for j := 0; j < 8; j++ {
			// 0x50ecc0 reads strength as u8 and multiplier as u16.
			rebuildIn[98+3*j] &= 0xff
			rebuildIn[99+3*j] &= 0xffff
		}
		caches = statusEffectRebuildNode(rebuildIn, r)
		if index == 6 {
			refresh = 1 // 0x523bde: attribute-affecting buff expiration.
		}
	}
	out := append(buffs, last, refresh, int64(adventure), int64(inventory))
	return append(out, caches...)
}
