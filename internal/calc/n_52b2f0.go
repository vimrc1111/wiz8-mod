package calc

func init() {
	names := []string{"who", "action", "actionIdx", "actionData0", "actionData1", "actionData2", "actionData3", "equipIdx",
		"equipItem0", "equipItem1", "equipItem2", "equipItem3", "itemWeight0", "itemWeight1", "itemWeight2", "itemWeight3", "maxFatigue", "fatigueStatus"}
	Nodes["0052b2f0"] = Node{Addr: "0052b2f0", In: names, Out: []string{"fatigueCost"}, F: characterActionFatigueCostNode}
}

// characterActionFatigueCostNode (exe 0x52b2f0): in = who, action, selected action slot
// index, the four action-data words of the slot record, selected equip slot index,
// the four equipment item ids (signed, -1 empty), the weight of items 0..3, max fatigue
// and fatigue status. Only the selected slot of each group is read, as in the original.
// Action 0/1: action data 5 -> R(3)+2, 6 -> R(3)+3 (exe 0x52b3cb, 0x52b3bc); otherwise an
// empty equip slot -> R(4)+3, else w=weight/40+1 -> R(w)+w+1; action 1 doubles the result
// (0x52b3d8). Action 3/6: max(maxFatigue/5, 20) (0x52b3e2). Action 8: R(2)+1; 9: R(4)+3;
// others 0. A non-zero fatigue status doubles the result (0x52b454).
func characterActionFatigueCostNode(in []int64, r Rand) []int64 {
	if len(in) != 18 || in[0] < 0 || in[0] > 7 || in[1] < 0 || in[2] < 0 || in[2] > 3 || in[7] < 0 || in[7] > 3 ||
		in[16] < 0 || in[16] > 0xffffffff || in[17] < 0 || in[17] > 0xffffffff {
		panic("character action fatigue cost: bad input")
	}
	action, kind := in[1], in[3+in[2]]
	item := in[8+in[7]]
	if item < -1 || item > 3 {
		panic("character action fatigue cost: item id out of range")
	}
	var v int64
	switch action {
	case 0, 1:
		switch kind {
		case 5:
			v = int64(r.R(3)) + 2
		case 6:
			v = int64(r.R(3)) + 3
		default:
			if item == -1 {
				v = int64(r.R(4)) + 3
			} else {
				w := in[12+item]/40 + 1
				v = int64(r.R(int(w))) + w + 1
			}
		}
		if action == 1 {
			v *= 2
		}
	case 3, 6:
		v = max(in[16]/5, 20)
	case 8:
		v = int64(r.R(2)) + 1
	case 9:
		v = int64(r.R(4)) + 3
	}
	if in[17] != 0 {
		v *= 2
	}
	return []int64{v}
}
