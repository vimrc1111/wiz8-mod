package calc

import "encoding/binary"

// lockCodeLoadInitNode models the Trigger lock-record loader at exe 0x43ccf0
// (lock_code_load_init). The loader reads a save stream (0x404ea0, stubbed in
// the case: ret ignored by the caller; it copies min(want, remaining) bytes, so
// short reads leave the destination's other bytes untouched). Environment
// assumptions shared with re/emu/cases/0043ccf0.json:
//   - the registry lookup (SR.DLL srRegistry::find, 0x43cdcd) is an input: findPtr
//     nonzero means every record names the one registered Trigger, zero means none does;
//   - unregistered records construct a Trigger via 0x43ba10 (treasureSeedInitNode,
//     same fixed clock) which is then destroyed, so only R(30000), nextTriggerID and
//     the stream consumption are observable;
//   - uninitialised stack slots (0x43cd0c.. locals) read as zero (emulator stack).
//
// Fixture caps: at most lockLoadMaxRecords records, stream size lockLoadStream;
// the record count is the stream's second dword (exe 0x43cd3b).
const (
	lockLoadStream     = 329
	lockLoadMaxRecords = 8
	lockLoadScalars    = 22 // inputs before the stream header
)

func init() {
	in := []string{"handle", "findPtr", "rec368", "rec36c", "rec370"}
	for i := 0; i < 8; i++ {
		in = append(in, "lock"+string(rune('0'+i)))
	}
	in = append(in, "rec37c", "rec380", "rec384", "rec388", "linkPtr", "linkType", "linkFlags", "linkWord",
		"nextTriggerID", "hdrA", "hdrB", "hdrC")
	for p := 12; p < lockLoadStream; p++ {
		in = append(in, "s"+itoa(p))
	}
	out := []string{"ret", "readCalls", "lastHandle", "lastSize", "nextTriggerID", "rec368", "rec36c", "rec370",
		"lock0", "lock1", "lock2", "lock3", "lock4", "lock5", "lock6", "lock7",
		"rec37c", "rec380", "rec384", "rec388", "linkFlags", "linkWord"}
	Nodes["0043ccf0"] = Node{Addr: "0043ccf0", In: in, Out: out, F: lockCodeLoadInitNode}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

func lockCodeLoadInitNode(in []int64, r Rand) []int64 {
	if len(in) != lockLoadStream-12+lockLoadScalars+3 {
		panic("lock_code_load_init: wrong input count")
	}
	stream := make([]byte, lockLoadStream)
	binary.LittleEndian.PutUint32(stream[0:], uint32(int32(in[22])))
	binary.LittleEndian.PutUint32(stream[4:], uint32(int32(in[23])))
	binary.LittleEndian.PutUint32(stream[8:], uint32(int32(in[24])))
	for p := 12; p < lockLoadStream; p++ {
		stream[p] = byte(in[25+p-12])
	}
	var (
		pos, calls, lastSize int64
		lastHandle           int64
	)
	read := func(dst []byte) { // 0x404ea0: copy what remains, never fail loudly
		n := len(dst)
		if rest := lockLoadStream - int(pos); n > rest {
			n = rest
		}
		copy(dst[:n], stream[pos:])
		pos += int64(n)
		calls++
		lastSize, lastHandle = int64(len(dst)), in[0]
	}
	i32 := func(b []byte) int32 { return int32(binary.LittleEndian.Uint32(b)) }
	put := func(b []byte, v int32) { binary.LittleEndian.PutUint32(b, uint32(v)) }

	// the found record (all records share it) and the link object behind rec+0x234
	var r368, r36c, r37c, r380, r384, r388 [4]byte
	put(r368[:], int32(in[2]))
	put(r36c[:], int32(in[3]))
	r370 := []byte{byte(in[4])}
	locks := make([]byte, 8)
	for i := range locks {
		locks[i] = byte(in[5+i])
	}
	put(r37c[:], int32(in[13]))
	put(r380[:], int32(in[14]))
	put(r384[:], int32(in[15]))
	put(r388[:], int32(in[16]))
	linkFlags, linkWord := byte(in[19]), uint16(in[20])
	nextID := in[21]
	found, hasLink := in[1] != 0, in[17] != 0

	var hdrA, hdrB, hdrC [4]byte // uninitialised locals read as zero if the stream is empty
	read(hdrA[:])
	read(hdrB[:])
	read(hdrC[:])
	a, count, c := i32(hdrA[:]), i32(hdrB[:]), i32(hdrC[:])
	if count > lockLoadMaxRecords {
		panic("lock_code_load_init: record count past fixture cap")
	}
	var l90, l9c, l98, l94 [4]byte // per-form version words persist across records
	var name [128]byte
	var scratch4 [4]byte
	var scratch1 [1]byte
	var scratchLocks [8]byte
	for n := int32(0); n < count; n++ {
		read(name[:])
		if found {
			if c == 0 && a > 1 {
				read(l90[:])
				read(r370)
				read(locks)
				read(r384[:])
				if i32(l90[:]) > 1 {
					read(r388[:])
				}
			} else {
				read(l9c[:])
				read(r368[:])
				read(r36c[:])
				read(r370)
				read(locks)
				read(r37c[:])
				read(r380[:])
				if i32(l9c[:]) > 1 {
					read(r384[:])
				}
				if c != 0 {
					if i32(r368[:]) == 1 {
						for i := range locks {
							locks[i] = byte(r.R(4)) // 0x40efa0(4)
						}
						d := i32(r36c[:])
						if d < 2 { // clamp 2..8 (0x43cf0d..0x43cf26)
							d = 2
						} else if d > 8 {
							d = 8
						}
						put(r384[:], d*3)
					}
					put(r388[:], -1)
					r370[0] = 0
				}
			}
			if hasLink && in[18] == 10 && (linkFlags&4 == 0 || linkWord == 0xffff) {
				var z byte
				if r370[0] == 0 {
					z = 4
				}
				linkFlags = z | linkFlags&0xfb
				linkWord = uint16(binary.LittleEndian.Uint32(r380[:]))
			}
			continue
		}
		nextID = treasureSeedInitNode([]int64{nextID}, r)[2] // 0x43ba10 then destroyed
		if c == 0 && a > 1 {
			read(l98[:])
			read(scratch1[:])
			read(scratchLocks[:])
			read(scratch4[:])
			if i32(l98[:]) > 1 {
				read(scratch4[:])
			}
		} else {
			read(l94[:])
			read(scratch4[:])
			read(scratch4[:])
			read(scratch1[:])
			read(scratchLocks[:])
			read(scratch4[:])
			read(scratch4[:])
			if i32(l94[:]) > 1 {
				read(scratch4[:])
			}
		}
	}
	o := []int64{1, calls, lastHandle, lastSize, nextID, int64(i32(r368[:])), int64(i32(r36c[:])), int64(r370[0])}
	for _, b := range locks {
		o = append(o, int64(b))
	}
	return append(o, int64(i32(r37c[:])), int64(i32(r380[:])), int64(i32(r384[:])), int64(i32(r388[:])),
		int64(linkFlags), int64(linkWord))
}
