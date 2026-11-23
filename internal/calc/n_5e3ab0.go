package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// trapPowerTargetCountNode is 0x5e3ab0 trap_power_target_count(trap), cdecl,
// called once from 0x589cf9 with a trap object. Exe 0x5e3ab0..0x5e3c7e.
//
// Level L = trap+0x36c (signed) clamped to 1..7; type T = trap+0x37c.
// R(2) == 0: the trap springs (0x4447f0: trap+0x370 = 1 and, when the linked
// object trap+0x234 is non-null with kind byte +4 == 0x0a, bit 2 of its byte +8
// is cleared) and the message is msgobj+0x1ecc; otherwise msgobj+0x1ed0
// (msgobj = *0x68c09c). The text "%s %s" (0x617584) of the trap name
// msgobj[word 0x61e9ec[T]] and that message is formatted into 0x689fd0
// (0x517a90) and queued (0x5299b0: a 0x1d entry, the text, a second 0x1d; only
// the text when bit 6 of **0x652dac is set).
// Count = L - R(L/2). Power = 4; when L > base = 0x6504ac[T], power is
// L - base + 4, and when that is >= 8 (unsigned) the count gains
// R(power - 7) and power becomes 7.
// trap+0x10c must be 2 (srAssertFail at 0x5e3b7b otherwise, cov-exempt).
// Position: trap+0x110 non-null -> model bounds 0x44d5c0 (排除, address stub
// writing lo/hi from case inputs), centre = (lo+hi)*0.5 (double 0x5ebe80) in
// x87 extended precision: x rounds once, y and z store the sum as float32
// first. Null -> 0x421150 (排除, camera point 1000 ahead, address stub writing
// the point). 0x421070 copies the view position into a local that only the
// ray query 0x434f20 (排除, address stub, ret ignored) receives by pointer;
// its seven arguments and call count are outputs. 0x5e3800 (D4) is an
// address stub: arguments (x, y, z, spell 0x650514[T], power, count) and
// count are outputs, its return (case input castRet) is this function's eax;
// nothing here reads its return or writes, so there is no Emu/F split.
//
// Fixture caps: T < 15 (the exe tables hold 15 trap types; 0x6504ac[15..]
// and 0x650514[15..] are other data), linked object = the one fixture record,
// message strings one UTF-16 unit each (0 = empty), queue capacity 64.
const (
	tptTypes   = 15
	tptEsp     = 0x10008000 // emulator entry esp
	tptLink    = 0x20000800 // fixture linked object (case object "link")
	tptQueue   = 64
	tptHalfVA  = 0x5ebe80
	tptBaseVA  = 0x6504ac
	tptSpellVA = 0x650514
)

type tptExeData struct {
	base, spell [tptTypes]int32
	half        float64
}

var tptExe, tptExeErr = tptLoadExe()

func tptLoadExe() (d tptExeData, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return d, fmt.Errorf("trap power: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return d, err
	}
	get := func(va uint32, n int) ([]byte, error) {
		o, err := peOffset(b, va)
		if err != nil || len(b)-o < n {
			return nil, fmt.Errorf("trap power: exe value %#x unreadable", va)
		}
		return b[o : o+n], nil
	}
	for t := 0; t < tptTypes; t++ {
		v, err := get(tptBaseVA+4*uint32(t), 4)
		if err != nil {
			return d, err
		}
		d.base[t] = int32(binary.LittleEndian.Uint32(v))
		if v, err = get(tptSpellVA+4*uint32(t), 4); err != nil {
			return d, err
		}
		d.spell[t] = int32(binary.LittleEndian.Uint32(v))
	}
	v, err := get(tptHalfVA, 8)
	if err != nil {
		return d, err
	}
	d.half = math.Float64frombits(binary.LittleEndian.Uint64(v))
	return d, nil
}

// x87 value loaded from float32 bits: finite as big.Float, or a special.
type tptX87 struct {
	nan  bool
	snan bool   // the float32 was a signalling NaN
	bits uint32 // the NaN's float32 bits, quieted
	inf  int    // +1 / -1
	v    *big.Float
}

func tptLoad(bits uint32) tptX87 {
	f := math.Float32frombits(bits)
	switch {
	case f != f:
		return tptX87{nan: true, snan: bits&0x400000 == 0, bits: bits | 0x400000}
	case math.IsInf(float64(f), 1):
		return tptX87{inf: 1}
	case math.IsInf(float64(f), -1):
		return tptX87{inf: -1}
	}
	return tptX87{v: new(big.Float).SetPrec(64).SetFloat64(float64(f))}
}

// tptAdd is fadd in 64-bit precision with x87 NaN rules: one NaN wins
// (quieted); SNaN and QNaN: the QNaN; otherwise the larger significand, tie ->
// positive sign; inf - inf -> indefinite.
func tptAdd(a, b tptX87) tptX87 {
	switch {
	case a.nan && b.nan && a.snan != b.snan:
		if a.snan {
			return b
		}
		return a
	case a.nan && b.nan:
		fa, fb := a.bits&0x7fffff, b.bits&0x7fffff
		if fa > fb || fa == fb && a.bits>>31 == 0 {
			return a
		}
		return b
	case a.nan:
		return a
	case b.nan:
		return b
	case a.inf != 0 && b.inf != 0 && a.inf != b.inf:
		return tptX87{nan: true, bits: 0xffc00000}
	case a.inf != 0:
		return a
	case b.inf != 0:
		return b
	}
	return tptX87{v: new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Add(a.v, b.v)}
}

// tptMulStore is fmul m64 (64-bit precision) then fstp m32.
func tptMulStore(a tptX87, c float64) uint32 {
	switch {
	case a.nan:
		return a.bits
	case a.inf != 0:
		if c == 0 || c != c || math.IsInf(c, 0) {
			panic("trap power: half constant at 0x5ebe80 is not finite non-zero (unknown exe build?)")
		}
		if (a.inf > 0) == (c > 0) {
			return 0x7f800000
		}
		return 0xff800000
	}
	p := new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Mul(a.v, big.NewFloat(c))
	f, _ := p.Float32()
	return math.Float32bits(f)
}

func tptStore(a tptX87) uint32 { // fstp m32 of a sum
	switch {
	case a.nan:
		return a.bits
	case a.inf > 0:
		return 0x7f800000
	case a.inf < 0:
		return 0xff800000
	}
	f, _ := a.v.Float32()
	return math.Float32bits(f)
}

// input positions
const (
	tptLevel = iota
	tptType
	tptKind
	tptModel
	tptLo // x, y, z
	_
	_
	tptHi // x, y, z
	_
	_
	tptAhead // x, y, z
	_
	_
	tptName
	tptTrigger
	tptResist
	tptQuiet
	tptSprung
	tptLinkPtr
	tptLinkKind
	tptLinkFlags
	tptCastRet
	tptInputs
)

func trapPowerTargetCountNode(in []int64, r Rand) []int64 {
	if len(in) != tptInputs {
		panic(fmt.Sprintf("trap power: expected %d inputs, got %d", tptInputs, len(in)))
	}
	if tptExeErr != nil {
		panic("trap power: " + tptExeErr.Error())
	}
	level := int32(in[tptLevel])
	level = max(1, min(level, 7)) // 0x5e3ac1..0x5e3ad2
	typ := uint32(in[tptType])
	if typ >= tptTypes {
		panic("trap power: trap type past the 15 exe table rows (fixture cap)")
	}
	sprung, linkFlags := in[tptSprung]&0xff, in[tptLinkFlags]&0xff
	msg := in[tptResist]
	if r.R(2) == 0 { // 0x4447f0
		sprung = 1
		switch uint32(in[tptLinkPtr]) {
		case 0:
		case tptLink:
			if in[tptLinkKind]&0xff == 0x0a {
				linkFlags &= 0xfb
			}
		default:
			panic("trap power: linked object is not the fixture record")
		}
		msg = in[tptTrigger]
	}
	var units []int64 // "%s %s"
	if u := in[tptName] & 0xffff; u != 0 {
		units = append(units, u)
	}
	units = append(units, 0x20)
	if u := msg & 0xffff; u != 0 {
		units = append(units, u)
	}
	var buf [6]int64
	copy(buf[:], units)
	msgCount := int64(3)
	if (in[tptQuiet]>>6)&1 != 0 {
		msgCount = 1
	}
	if msgCount > tptQueue {
		panic("trap power: message queue past the fixture capacity")
	}

	count := level - int32(r.R(int(level/2))) // 0x5e3b2c..0x5e3b3b
	power := uint32(4)
	if base := tptExe.base[typ]; level > base {
		power = uint32(level - base + 4)
		if power >= 8 {
			count += int32(r.R(int(power - 7)))
			power = 7
		}
	}
	if in[tptKind]&0xff != 2 {
		panic("trap power: trap+0x10c != 2 asserts at 0x5e3b7b (cov-exempt)")
	}
	var pos [3]uint32
	n44, n42 := int64(0), int64(0)
	if uint32(in[tptModel]) != 0 {
		n44 = 1
		var s [3]tptX87
		for i := range s {
			s[i] = tptAdd(tptLoad(uint32(in[tptLo+i])), tptLoad(uint32(in[tptHi+i])))
		}
		pos[0] = tptMulStore(s[0], tptExe.half)
		for i := 1; i < 3; i++ {
			pos[i] = tptMulStore(tptLoad(tptStore(s[i])), tptExe.half)
		}
	} else {
		n42 = 1
		for i := range pos {
			pos[i] = uint32(in[tptAhead+i])
		}
	}
	out := []int64{in[tptCastRet] & 0xffffffff, 1, int64(pos[0]), int64(pos[1]), int64(pos[2]),
		int64(uint32(tptExe.spell[typ])), int64(power), int64(uint32(count)), n44, n42, 1,
		tptEsp - 0xc, tptEsp - 0x48, 1, 0xfffffffd, 0xfffffffd, 1, 0, msgCount,
		buf[0] | buf[1]<<16, buf[2] | buf[3]<<16, buf[4] | buf[5]<<16, sprung, linkFlags}
	return out
}

func init() {
	in := []string{"trapLevel", "trapType", "trapKind", "modelPtr", "boundLoX", "boundLoY", "boundLoZ",
		"boundHiX", "boundHiY", "boundHiZ", "aheadX", "aheadY", "aheadZ", "nameUnit", "triggerUnit",
		"resistUnit", "msgQuiet", "trapSprung", "linkPtr", "linkKind", "linkFlags", "castRet"}
	out := []string{"eax", "n5e3800", "castX", "castY", "castZ", "castSpell", "castPower", "castCount",
		"n44d5c0", "n421150", "n434f20"}
	for k := 1; k <= 7; k++ {
		out = append(out, "ray"+strconv.Itoa(k))
	}
	out = append(out, "msgCount", "msgW0", "msgW1", "msgW2", "trapSprungOut", "linkFlagsOut")
	Nodes["005e3ab0"] = Node{Addr: "005e3ab0", In: in, Out: out, F: trapPowerTargetCountNode}
}
