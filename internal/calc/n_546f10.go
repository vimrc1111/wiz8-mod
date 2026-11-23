package calc

func init() {
	Nodes["00546f10"] = Node{
		Addr: "00546f10",
		In:   []string{"characterIndex", "characterCharm", "monsterDisposition"},
		Out:  []string{"targetDisposition"}, F: targetDispositionNode,
	}
}

// targetDispositionNode uses the selected character's C+0xa35 and the
// monster's M+0x16. The party pointer at 0x685174 has record stride 0x1862
// (0x546f10..0x546f29); the fixture supplies that selected field directly.
func targetDispositionNode(in []int64, r Rand) []int64 {
	if len(in) != 3 || in[0] < 0 || in[0] > 7 || in[1] < -2147483648 || in[1] > 2147483647 || in[2] < 0 || in[2] > 255 {
		panic("target disposition: expected party index 0..7, i32 charm and u8 disposition")
	}
	disposition := in[2]
	// 0x546f30: every nonzero charm value switches allegiance, including
	// negative values. With zero charm, 0x546f38 returns the original byte.
	if in[1] != 0 {
		switch disposition {
		case 0: // Neutral remains neutral (0x546f6a).
		case 1: // Hostile becomes friendly (0x546f70).
			disposition = 2
		case 2: // Friendly becomes hostile (0x546f6d).
			disposition = 1
		default:
			// 0x546f50 reports an invalid allegiance through srAssertFail.
			panic("target disposition: charmed character requires disposition 0..2")
		}
	}
	// Only the low byte is the return value; upper EAX bits are incidental.
	return []int64{disposition}
}
