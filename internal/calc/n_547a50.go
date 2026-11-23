package calc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
)

// cheat_death_hp 0x547a50(who) (build/wiz8.asm:~437500): the member survives a
// deadly blow. Shows a notice, then applies state 0x11 (0x522fe0, same group) for
// a time derived from the member's class rank n (character +0x8d+4*class, class -1
// reads +0x89): f = 5.0 for n > 20, else (2n + 60.0) * 5.0 * 0.01; the state time
// is slot 0xa45 - ftol(f) + 7. Afterwards +0xb11 is set from +0xb0d (the first time
// the combat record byte +0xbe of the member is clear: (R(t) + 50) * +0xb0d / 100
// with t = ftol(+0xb0d * 0.5 * rankScale), then the byte is set) or, when the record
// table 0x6836a8 is null or the byte is set, to R(t) + 1 with t = ftol(+0xb11 *
// rankScale). rankScale is (2n + 60) * 0.01 for n <= 20 and 1 above.
//
// Constants from the exe: 5.0 at 0x5ebc28, 0.5 at 0x5ebc7c (binary32), 60.0 at
// 0x5ee800 and 0.01 at 0x5ed490 (loaded by statusRebuildTables). The float chain
// runs at x87 extended precision; 0x5e1c22 (_ftol) truncates to int64 and keeps
// the low 32 bits. The group is run by the 0x522fe0 code in this package, so the
// stubs and outputs are those of 0x522fe0.
//
// Fixture caps (calc panics past them): who 0..7, class -1..14; the record table
// pointer is null or the fixture object.
var chdConst, chdConstErr = chdLoad()

func chdLoad() (c [2]*big.Float, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return c, fmt.Errorf("cheat death hp: set WIZ8_EXE")
		}
		path = filepath.Join(filepath.Dir(file), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	for i, va := range []uint32{0x5ebc28, 0x5ebc7c} {
		o, e := peOffset(b, va)
		if e != nil || o+4 > len(b) {
			return c, fmt.Errorf("cheat death hp: constant %#x not in exe", va)
		}
		c[i] = chdF(float64(math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))))
	}
	return c, nil
}

func chdF(x float64) *big.Float {
	return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).SetFloat64(x)
}

func chdMul(a, b *big.Float) *big.Float {
	return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Mul(a, b)
}

func chdAdd(a, b *big.Float) *big.Float {
	return new(big.Float).SetPrec(64).SetMode(big.ToNearestEven).Add(a, b)
}

// chdFtol is 0x5e1c22: truncate toward zero, low 32 bits of the int64.
func chdFtol(x *big.Float) uint32 {
	i, acc := x.Int64()
	if acc != big.Exact && (x.Cmp(chdF(math.MaxInt64)) >= 0 || x.Cmp(chdF(math.MinInt64)) <= 0) {
		panic("cheat death hp: ftol overflow")
	}
	return uint32(i)
}

func init() {
	in := append([]string{"who"}, ccaWNames()...)
	Nodes["00547a50"] = Node{Addr: "00547a50", In: in, Out: ccaOutNames(false), F: cheatDeathHpNode}
}

func cheatDeathHpNode(in []int64, r Rand) []int64 {
	if chdConstErr != nil {
		panic(chdConstErr)
	}
	s := ccaNew(in, 1, r, true)
	who := in[0]
	if who < 0 || who >= ccaChars {
		panic("cheat death hp: who outside the eight members")
	}
	chdRun(s, who)
	return s.output(false, 0)
}

// chdRank is the class rank n: +0x8d+4*class (class -1 reads +0x89).
func chdRank(s *ccaS) uint32 {
	cls := s.g(ccaWCls)
	switch {
	case cls == -1:
		return s.u(ccaWNm1)
	case cls >= 0 && cls <= 14:
		return s.u(ccaWN + int(cls))
	}
	panic("cheat death hp: class outside -1..14")
}

// chdScale applies (2n+60)*0.01 to f when n <= 20.
func chdScale(f *big.Float, n uint32) *big.Float {
	if n > 20 {
		return f
	}
	two := chdAdd(chdF(float64(n)), chdF(float64(n)))
	k := chdAdd(two, chdF(float64(statusRebuildTables.featureBase)))
	return chdMul(chdMul(f, k), chdF(statusRebuildTables.featureScale))
}

func chdRun(s *ccaS, who int64) {
	s.notice(who)
	n := chdRank(s)
	f := chdConst[0]
	if n <= 20 {
		two := chdAdd(chdF(float64(n)), chdF(float64(n)))
		f = chdMul(chdMul(chdAdd(two, chdF(statusRebuildTables.featureBase)), chdConst[0]), chdF(statusRebuildTables.featureScale))
	}
	tm := s.u(ccaWS+17) - chdFtol(f) + 7
	ccaApply(s, who, 0x11, tm, 0, true)
	rp := s.g(ccaWRecPtr)
	if rp != 0 && rp != ccaRecObj {
		panic("cheat death hp: record table pointer outside the fixture")
	}
	if rp == 0 || s.g(ccaWRecBe+int(who))&0xff != 0 {
		t := chdFtol(chdScale(chdF(float64(s.u(ccaWB11))), n))
		s.put(ccaWB11, int64(uint32(s.r.R(int(t)))+1))
		return
	}
	b0d := s.u(ccaWB0d)
	t := chdFtol(chdScale(chdMul(chdF(float64(b0d)), chdConst[1]), n))
	x := (uint32(s.r.R(int(t))) + 0x32) * b0d
	s.put(ccaWB11, int64(uint32((uint64(x)*0x51eb851f)>>37)))
	s.put(ccaWRecBe+int(who), 1)
}
