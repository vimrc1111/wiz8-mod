package calc

import "strconv"

// exe 0x445500 treasure_generation (thiscall, ecx = scene object). Fixture
// caps, each a panic when reached (never a silent answer): 3 ITEMTABLES
// records, names of 32 bytes (NUL after byte 32), a generated-item list of 4,
// one container object plus two chained items. The *Base/offset constants
// mirror re/emu/cases/00445500.json's arena (0x20000000).
//
// Callees: 4f88a0 (table lookup by name) and 4f8340 (append to a container
// chain) are modelled here; 4f6c50 -> 51c020/4f6b90/4f93d0 (container
// creation) is counted through its record. 4f88f0 (item_pool_selection) and
// 517970 (dice_roll) are D4 formula nodes, address-stubbed in the case: their
// results (list size, dice values) are inputs and their call arguments are
// outputs. srand (0x5eb1d8) is recorded likewise.
const (
	tcBase      = 0x20000000
	tcW         = 32
	tcTables    = 3
	tcList      = 4
	tcCont      = tcBase + 0x1000
	tcE0        = tcBase + 0x1100
	tcE1        = tcBase + 0x1200
	tcL0        = tcBase + 0x1300 // l_k = tcL0 + 0x100*k
	tcPool0     = tcBase + 0x2000 // table t = tcPool0 + 0x200*t
	tcItemDice  = 0x1cd           // table +0x1cd, exe 0x4455a8; notes/re/loot.md:13
	tcMoneyDice = 0x1d5           // table +0x1d5, exe 0x445620
	tcBoxItem   = 0x23c           // container item id, exe 0x4455d5
	// input indexes
	tcDone, tcSeed, tcCont0, tcTableCount, tcDiceItem, tcDiceMoney = 0, 1, 2, 3, 4, 5
	tcListCount, tcRecID, tcMoney0, tcOrdinal, tcFirst, tcE0Next   = 6, 7, 8, 9, 10, 11
	tcE1Next, tcL0Next, tcName, tcTableName                        = 12, 13, 17, 17 + tcW
	tcExtra                                                        = tcTableName + tcTables*tcW // F-only inputs start here
	tcDiceIn                                                       = 6                          // per table: item Z,N,S then money Z,N,S
)

func treasureNames() (in, out []string) {
	in = []string{"doneFlag", "seed", "contPtr", "tableCount", "diceItem", "diceMoney", "listCount", "recId", "money0", "ordinal", "cont.first", "e0.next", "e1.next"}
	for k := 0; k < tcList; k++ {
		in = append(in, "l"+strconv.Itoa(k)+".next")
	}
	for k := 0; k < tcW; k++ {
		in = append(in, "name"+strconv.Itoa(k))
	}
	for t := 0; t < tcTables; t++ {
		for k := 0; k < tcW; k++ {
			in = append(in, "table"+strconv.Itoa(t)+".name"+strconv.Itoa(k))
		}
	}
	for t := 0; t < tcTables; t++ {
		for _, d := range []string{"item", "money"} {
			for _, f := range []string{"Z", "N", "S"} {
				in = append(in, "f.t"+strconv.Itoa(t)+"."+d+f)
			}
		}
	}
	ip, _ := ipNames() // party levels, pools, item definitions of 0x4f88f0 (its recId is this node's recId)
	for _, n := range ip[ipIn0 : len(ip)-1] {
		in = append(in, "f."+n)
	}
	out = []string{"srand.calls", "srand.seed", "pool.calls", "pool.sel", "pool.K", "dice.calls", "dice.arg0", "dice.arg1",
		"done", "money", "ordinal", "mk.calls", "mk.a2", "mk.a3", "mk.a4", "mk.a5", "place.calls", "cont.first", "e0.next", "e1.next"}
	for k := 0; k < tcList; k++ {
		out = append(out, "l"+strconv.Itoa(k)+".next")
	}
	return
}

func init() {
	in, out := treasureNames()
	Nodes["00445500"] = Node{Addr: "00445500", In: in, Out: out, F: treasureGenerationNode,
		Emu: func(in []int64, r Rand) []int64 { return treasureNode(in, r, false) }}
}

// tcCString: bytes up to the first NUL (the fixture's byte 32 is always NUL).
func ipNamesIn() []string { in, _ := ipNames(); return in }

func tcCString(b []int64) []byte {
	var s []byte
	for _, c := range b {
		if c < 0 || c > 255 {
			panic("treasureGeneration: name bytes are u8")
		}
		if c == 0 {
			break
		}
		s = append(s, byte(c))
	}
	return s
}

// tcFold is the C-locale tolower of _stricmp (exe 0x4f88c2 via 0x5eb1e8): only A-Z fold.
func tcFold(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

func tcEqualFold(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if tcFold(a[i]) != tcFold(b[i]) {
			return false
		}
	}
	return true
}

func treasureGenerationNode(in []int64, r Rand) []int64 { return treasureNode(in, r, true) }

// treasureNode: real=false is Emu (the case address-stubs 0x517970 and 0x4f88f0,
// so K, the gold roll and the list size are the inputs diceItem, diceMoney,
// listCount). real=true is F: it runs diceNode for both rolls and
// itemPoolSelectionNode (0x4f88f0) for the list, in the exe's call order, and
// ignores those three inputs. 0x51c020 is not run: nothing here reads its record.
// F reads the f.* inputs (per-table dice records, party levels, pools, item
// definitions; Emu leaves them in scratch memory). F's extra caps (panics): the
// matched table must be one of 4f88f0's 2 pools, the list at most 4 items, and
// 4f88f0's own fixture caps. F's mk/place counters add 4f88f0's item objects.
func treasureNode(in []int64, r Rand, real bool) []int64 {
	nIn, _ := treasureNames()
	if len(in) != len(nIn) {
		panic("treasureGeneration: wrong input count")
	}
	u32 := func(i int) int64 {
		if in[i] < 0 || in[i] > 0xffffffff {
			panic("treasureGeneration: " + nIn[i] + " is u32")
		}
		return in[i]
	}
	i32 := func(i int) int64 {
		if in[i] < -0x80000000 || in[i] > 0x7fffffff {
			panic("treasureGeneration: " + nIn[i] + " is i32")
		}
		return in[i]
	}
	if in[tcDone] < 0 || in[tcDone] > 255 {
		panic("treasureGeneration: doneFlag is u8")
	}
	done, ordinal, money := in[tcDone], u32(tcOrdinal), i32(tcMoney0)
	i32(tcDiceItem)
	i32(tcDiceMoney)
	u32(tcRecID)
	var srandCalls, srandSeed, poolCalls, poolSel, poolK, diceCalls, diceArg0, diceArg1 int64
	var mkCalls, mkA2, placeCalls int64
	// next pointers of the fixture's chained objects (+0x31); container field is cont.first
	first := u32(tcFirst)
	next := map[int64]int64{tcE0: u32(tcE0Next), tcE1: u32(tcE1Next)}
	for k := 0; k < tcList; k++ {
		next[tcL0+0x100*int64(k)] = u32(tcL0Next + k)
	}
	nameLen := len(tcCString(in[tcName : tcName+tcW]))
	if nameLen != 0 && done == 0 { // exe 0x44556a, 0x445576
		srandCalls, srandSeed = 1, u32(tcSeed) // exe 0x445585
		sel := int64(-1)
		count := i32(tcTableCount) // exe 0x4f88a9 signed
		name := tcCString(in[tcName : tcName+tcW])
		for t := int64(0); t < count; t++ {
			if t >= tcTables {
				panic("treasureGeneration: fixture holds 3 tables")
			}
			base := tcTableName + int(t)*tcW
			if tcEqualFold(name, tcCString(in[base:base+tcW])) {
				sel = t
				break
			}
		}
		if sel >= 0 {
			pool := tcPool0 + 0x200*sel
			poolCalls, poolSel, poolK = 1, sel, in[tcDiceItem] // exe 0x4455ba
			diceCalls, diceArg0, diceArg1 = 2, pool+tcItemDice, pool+tcMoneyDice
			n := i32(tcListCount) // the 4f88f0 result list size, exe 0x4455e8 signed
			var poolMk, poolPlace, poolLast int64
			if real {
				d := tcExtra + int(sel)*tcDiceIn
				poolK = int64(int32(diceNode(in[d:d+3], r)[0])) // exe 0x4455ae
				v := make([]int64, len(ipNamesIn()))
				v[0], v[1], v[2], v[3], v[4] = sel, poolK&0xffffffff, 0, 5, ordinal // empty list: count 0, cap 5 (0x5e1ce0 of 0x14)
				copy(v[ipIn0:], in[tcExtra+tcTables*tcDiceIn:])
				v[len(v)-1] = u32(tcRecID)
				o := itemPoolSelectionNode(v, r) // exe 0x4455ba
				n, ordinal, poolMk, poolPlace, poolLast = o[0], o[3], o[4], o[5], o[6]
			}
			cont := u32(tcCont0)
			heapFirst := int64(0) // a container made by 0x4f6c50 lives on the heap
			useArena := true
			switch cont {
			case 0: // exe 0x4455ca
				useArena = false
				mkCalls, mkA2, placeCalls = 1, tcBoxItem, 1
				ordinal = (ordinal + 1) & 0xffffffff // exe 0x4f6bcc
			case tcCont:
			default:
				panic("treasureGeneration: fixture holds one container")
			}
			if n > tcList {
				panic("treasureGeneration: fixture holds 4 list items")
			}
			for k := int64(0); k < n; k++ { // exe 0x4455f2..0x445616, calls 0x4f8340
				item := tcL0 + 0x100*k
				head := &first
				if !useArena {
					head = &heapFirst
				}
				if *head == 0 {
					*head = item
				} else {
					c := *head
					for steps := 0; ; steps++ {
						nx, ok := next[c]
						if !ok || steps > 8 {
							panic("treasureGeneration: chain leaves the fixture")
						}
						if nx == 0 {
							break
						}
						c = nx
					}
					next[c] = item
				}
				next[item] = 0
			}
			done = 1 // exe 0x445635
			money = in[tcDiceMoney]
			if real {
				d := tcExtra + int(sel)*tcDiceIn + 3
				money = int64(int32(diceNode(in[d:d+3], r)[0])) // exe 0x445627
				mkCalls += poolMk
				placeCalls += poolPlace
				if mkA2 == 0 {
					mkA2 = poolLast
				}
			}
		}
	}
	out := []int64{srandCalls, srandSeed, poolCalls, poolSel, poolK, diceCalls, diceArg0, diceArg1,
		done, money, ordinal, mkCalls, mkA2, 0, 0, 0, placeCalls, first, next[tcE0], next[tcE1]}
	for k := 0; k < tcList; k++ {
		out = append(out, next[tcL0+0x100*int64(k)])
	}
	return out
}
