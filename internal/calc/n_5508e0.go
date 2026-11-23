package calc

// monster_spell_interaction 0x5508e0(e): what happens when a spell effect e lands
// on a monster (build/wiz8.asm:457140..457300).
//
//	e+0x00 spell index (spell table *0x65be1c, stride 0x1bf); its realm is u32 at +0x133
//	e+0x5c first target record {kind, character index}; must be a character (kind 1)
//	       with index != -1, else srAssertFail (0x53beb1 for index -1, 0x55091d for another kind; panic here)
//	e+0x78 byte: non-zero = "monster branch"
//	e+0x90 second target record {kind, ., monster id at +0x98}; kind must be 3 (monster),
//	       else srAssertFail (0x550944, panic here)
//	e+0xc8 effect strength
//
// 0x4e5840 finds the live monster by id (list *0x683fad, then *0x683fb9; a missing id
// asserts) and 0x4e5720 maps its template index (monster +8) to the template record.
//
//	monster branch: half = (template u8 +0xd2 + u8 +0xd5) / 2; R(100) < half posts event 0x16.
//	otherwise: 0x5520d0(e+0x90, realm, strength, 0) resisted -> post event 0x69;
//	           not resisted -> 0x576da0 on the shared flag word (*(*0x649f1c+0x1d4)+0x85):
//	           bit 0 clear: post 0x67 and set bit 0; else bit 1 clear: post 0x68 and set
//	           bit 1; else post 0x69 (flags unchanged).
//
// Posting is 0x528830(id, 0, 0, 0), run natively in emu: a 0x24-byte event {id, stamp
// *0x68c4ac at +0x20} is allocated (first heap block) and appended to the queue
// (*0x68c4c8, count 0x68c4c0, capacity 0x68c4c4); when count+1 > capacity (signed) the
// array is reallocated to count+1 words (second heap block, 0x28 on) and the old
// entries copied. Allocation failure (0x5288a3) is not modelled.
//
// Emu address-stubs the D4 callee 0x5520d0: its return is input resistResult (low byte),
// and its four arguments plus call count are outputs. F runs the real statusResistRoll
// on the monster target instead and ignores resistResult; the monster, template,
// difficulty and realm inputs it needs are the same case inputs (they sit in their real
// places; the stub never reads them).
//
// Fixture caps (panic past them): spell index 0..3, match slot 0..3 (the live monster
// is slot matchSlot of two 2-entry lists; the other slots hold decoy ids >= 0xffffff00),
// monster id < 0xffffff00, template index 0..2, queue count 0..3, capacity 0..4.
const (
	msiArena    = 0x20000000
	msiEffect   = msiArena + 0 // case object "effect"
	msiQueue    = msiArena + 0x8000
	msiHeap     = 0x50000000
	msiGrown    = msiHeap + 0x28
	msiMaxQueue = 3
)

var msiIn = func() []string {
	n := []string{"spellIndex", "spellRealm", "targetCharacterIndex", "monsterBranch", "monsterID", "matchSlot", "strength", "templateIndex"}
	realms := []string{"fire", "water", "air", "earth", "mental", "divine"}
	for _, r := range realms {
		n = append(n, "templateResistance_"+r)
	}
	n = append(n, "templateLevel", "templateByteD2", "templateByteD5")
	for _, r := range realms {
		n = append(n, "monsterResistanceModifier_"+r)
	}
	n = append(n, "monsterPrimaryState", "monsterDifficultyClass", "difficulty", "resistResult", "sharedFlagWord", "queueCount", "queueCapacity",
		"queueEntry0", "queueEntry1", "queueEntry2", "queueEntry3", "eventStamp", "firstTargetKind", "secondTargetKind")
	return n
}()

const (
	msiSpell, msiRealm, msiChar, msiBranch, msiMonID, msiSlot, msiStrength, msiTpl = 0, 1, 2, 3, 4, 5, 6, 7
	msiTplRes, msiTplLevel, msiD2, msiD5, msiMod                                   = 8, 14, 15, 16, 17
	msiPrimary, msiClass, msiDiff, msiResist, msiFlag, msiCount, msiCap, msiQ0     = 23, 24, 25, 26, 27, 28, 29, 30
	msiStamp, msiKind1, msiKind2                                                   = 34, 35, 36
)

func init() {
	out := []string{"eventID", "eventStamp", "queueCount", "queueCapacity", "queuePointer",
		"queueEntry0", "queueEntry1", "queueEntry2", "queueEntry3",
		"grownEntry0", "grownEntry1", "grownEntry2", "grownEntry3",
		"sharedFlagWord", "resistCalls", "resist_target", "resist_realm", "resist_strength", "resist_status"}
	Nodes["005508e0"] = Node{Addr: "005508e0", In: msiIn, Out: out,
		F:   func(in []int64, r Rand) []int64 { return msiRun(in, r, true) },
		Emu: func(in []int64, r Rand) []int64 { return msiRun(in, r, false) }}
}

func monsterSpellInteractionNode(in []int64, r Rand) []int64 { return msiRun(in, r, true) }

func msiRun(in []int64, r Rand, real bool) []int64 {
	if len(in) != len(msiIn) {
		panic("monster spell interaction: wrong input count")
	}
	for i, v := range in {
		lo, hi := int64(0), int64(0xffffffff)
		switch {
		case i >= msiMod && i < msiMod+6:
			lo, hi = -128, 127
		case i == msiCount || i == msiCap:
			lo, hi = -0x80000000, 0x7fffffff
		case i == msiBranch || i >= msiTplLevel && i <= msiD5 || i >= msiTplRes && i < msiTplRes+6 || i == msiClass:
			hi = 255
		}
		if v < lo || v > hi {
			panic("monster spell interaction: input out of field range")
		}
	}
	if in[msiSpell] > 3 || in[msiSlot] > 3 || in[msiTpl] > 2 || in[msiMonID] >= 0xffffff00 {
		panic("monster spell interaction: spell index, match slot, template index or monster id beyond the case fixtures")
	}
	if in[msiKind1] != 1 {
		panic("monster spell interaction: first target is not a character (assert 0x55091d)")
	}
	if in[msiKind2] != 3 {
		panic("monster spell interaction: second target is not a monster (assert 0x550944)")
	}
	if in[msiChar] == 0xffffffff {
		panic("monster spell interaction: first target character index -1 (assert 0x53beb1)")
	}
	if in[msiCount] < 0 || in[msiCount] > msiMaxQueue || in[msiCap] < 0 || in[msiCap] > msiMaxQueue+1 {
		panic("monster spell interaction: queue count or capacity beyond the case queue")
	}
	out := make([]int64, 19)
	flag := in[msiFlag]
	var event int64
	if in[msiBranch] != 0 { // 0x55097a
		if int64(r.R(100)) < (in[msiD2]+in[msiD5])/2 { // 0x5509a7, 0x4e5720 template +0xd2/+0xd5
			event = 0x16
		}
	} else {
		resisted := in[msiResist]&0xff != 0 // stub: low byte of the return (0x5509d3)
		out[14], out[15], out[16], out[17], out[18] = 1, msiEffect+0x90, in[msiRealm], in[msiStrength], 0
		if real {
			if in[msiRealm] > 5 {
				panic("monster spell interaction: realm beyond 0..5 (status resist fixture)")
			}
			v := make([]int64, len(srrIn))
			v[0], v[2], v[3], v[4], v[6] = 3, in[msiMonID], in[msiRealm], in[msiStrength], in[msiDiff]
			v[17] = in[msiTpl]
			copy(v[18:24], in[msiTplRes:msiTplRes+6])
			v[24] = in[msiTplLevel]
			copy(v[25:31], in[msiMod:msiMod+6])
			v[31], v[32] = in[msiPrimary], in[msiClass]
			resisted = statusResistRoll(v, r, true)[0] != 0
		}
		if resisted {
			event = 0x69
		} else { // 0x576da0
			switch {
			case flag&1 == 0:
				event, flag = 0x67, flag|1
			case flag&2 == 0:
				event, flag = 0x68, flag|2
			default:
				event = 0x69 // 0x576e10
			}
		}
	}
	out[13] = flag
	count, capacity := int32(in[msiCount]), int32(in[msiCap])
	var q, grown [4]int64
	copy(q[:], in[msiQ0:msiQ0+4])
	ptr := int64(msiQueue)
	if event != 0 { // 0x528830(event, 0, 0, 0)
		out[0], out[1] = event, in[msiStamp]
		if count+1 > capacity { // signed, 0x528880
			for i := int32(0); i < count; i++ {
				grown[i] = q[i]
			}
			grown[count], ptr, capacity = msiHeap, msiGrown, count+1
		} else {
			q[count] = msiHeap
		}
		count++
	}
	out[2], out[3], out[4] = int64(count), int64(capacity), ptr
	copy(out[5:9], q[:])
	copy(out[9:13], grown[:])
	return out
}
