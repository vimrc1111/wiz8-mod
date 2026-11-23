package calc

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

// Column offsets of the search lattice: five binary32 values at 0x605ae8,
// read from the player's own Wiz8.exe (no table is embedded here).
var spawnSearchColumns, spawnSearchColumnsErr = func() (t [5]float64, err error) {
	path := os.Getenv("WIZ8_EXE")
	if path == "" {
		_, source, _, _ := runtime.Caller(0)
		path = filepath.Join(filepath.Dir(source), "../../build/Wiz8.exe")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	off, err := peOffset(b, 0x605ae8)
	if err != nil {
		return t, err
	}
	for i := range t {
		t[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[off+4*i:])))
	}
	return t, nil
}()

// Scene fixture as in 0x437f30: empty geometry bounded above by groundCeiling,
// one spatial cell holding an optional monster, four navigation cells.
// Float inputs and coordinate outputs are IEEE binary32 bit patterns.
// The output buffer holds the 50 positions that rows*columns can ever place.
func init() {
	names := []string{"placed"}
	for i := 0; i < 50; i++ {
		for _, axis := range []string{"X", "Y", "Z"} {
			names = append(names, "position"+string(rune('0'+i/10))+string(rune('0'+i%10))+axis)
		}
	}
	Nodes["00437980"] = Node{Addr: "00437980", In: []string{
		"centerX", "centerY", "centerZ", "heading", "spacing", "count",
		"avoidActors", "preserveHeight", "groundCeiling",
		"partyX", "partyY", "partyZ", "partyRadius", "monsterClass",
		"monsterX", "monsterY", "monsterZ", "monsterRadius",
		"navFloor", "navTolerance", "navHorizontalStep", "navVerticalStep",
		"navMinX", "navMinY", "navMinZ", "navMaxX", "navMaxY", "navMaxZ",
		"navCell1", "navCell2", "navCell3",
	}, Out: names, F: spawnPositionSearchNode}
}

func spawnPositionSearchNode(in []int64, r Rand) []int64 {
	if len(in) != 31 || in[5] < 0 || in[5] > math.MaxUint32 {
		panic("spawn_position_search: expected 31 inputs and u32 count")
	}
	if spawnSearchColumnsErr != nil {
		panic(spawnSearchColumnsErr)
	}
	f := func(i int) float64 { return float64(math.Float32frombits(uint32(in[i]))) }
	f32 := func(x float64) float64 { return float64(float32(x)) }
	bits := func(x float64) int64 { return int64(math.Float32bits(float32(x))) }
	const (
		ceiling                   = 8
		partyX, partyRadius       = 9, 12
		monsterClass, monsterX    = 13, 14
		monsterRadius             = 17
		navTolerance, navStep     = 19, 20
		navVStep                  = 21
		navMinX, navMinY, navMinZ = 22, 23, 24
		navMaxX, navMaxY, navMaxZ = 25, 26, 27
	)
	count := uint32(in[5])
	cx, cy, cz, spacing := f(0), f(1), f(2), f(4)
	// 0x4379ca, 0x517c94, 0x517cba: half-turn, normalization offset, period
	// (same wrap as 0x437f30).
	a := f32(f(3)+3.141592502593994) + 6.2831852
	const period = 6.283185005187988
	for a < 0 {
		a += period
	}
	for a >= period {
		a -= period
	}
	// 0x437a18..0x437a3f: lattice steps along and across the heading.
	along, across := f32(spacing*math.Cos(a)), f32(spacing*math.Sin(a))
	// 0x4379b0, 0x4379ba: three columns, five when more than ten are requested.
	cols := 3
	if count > 10 {
		cols = 5
	}
	// 0x420bd0/0x433820: without triangle hits, the scene ceiling clips the query.
	ground := func(y float64) float64 { return min(y, f(ceiling)) }
	// 0x437c9e, 0x437cbf, 0x437ce9: minimum, absolute and normal height limits.
	validDrop := func(y, h float64) bool {
		d := math.Abs(f32(y - h))
		return d >= 1 && d <= 5000 && (in[7] != 0 || d <= 2500)
	}
	type point struct{ x, y, z float64 }
	cell := func(p point) (int32, int32, int32) {
		return int32((p.x - f(navMinX)) / f(navStep)), int32((p.y-f(navMinY))/f(navVStep)) + 1, int32((p.z - f(navMinZ)) / f(navStep))
	}
	lookup := func(x, z int32) (uint32, bool) {
		switch uint32(z)<<16 + uint32(x) {
		case 0:
			return uint32(in[18]), true
		case 1:
			return uint32(in[28]), true
		case 65536:
			return uint32(in[29]), true
		case 65537:
			return uint32(in[30]), true
		}
		return 0, false
	}
	nav := func(p point, radius bool) bool {
		x, y, z := cell(p)
		packed, found := lookup(x, z)
		if !found {
			return false
		}
		d := y - int32(packed&65535)
		// 0x462e72 and 0x463051: point tests use twice the signed height tolerance.
		tolerance := int32(int16(in[navTolerance])) * 2
		if d <= -tolerance || d >= tolerance || radius && packed&0x10000000 != 0 {
			return false
		}
		if radius {
			clearance := float64(packed >> 16 & 255)
			if packed&0x1000000 != 0 {
				clearance = 0
			}
			// 0x463248..0x46325b: strict clearance test against f32(spacing*0.5).
			return clearance*500+f(navStep)*0.5 > f32(spacing*0.5)
		}
		return true
	}
	connected := func(p, q point) bool {
		for _, v := range []point{p, q} {
			if v.x < f(navMinX) || v.y < f(navMinY) || v.z < f(navMinZ) || v.x > f(navMaxX) || v.y > f(navMaxY) || v.z > f(navMaxZ) {
				return false
			}
		}
		px, py, pz := cell(p)
		qx, qy, qz := cell(q)
		tolerance := int32(int16(in[navTolerance]))
		valid := func(packed uint32, height int32) bool {
			d := int32(packed&65535) - height
			return packed&0x10000000 == 0 && d > -tolerance && d < tolerance
		}
		if px == qx && pz == qz {
			packed, found := lookup(px, pz)
			return found && valid(packed, py) && valid(packed, qy)
		}
		// 0x45af60: traverse the segment in integer world units. The dominant
		// axis and fractional distance to each cell edge determine its error term.
		step := int32(f(navStep))
		start := [2]int32{int32(p.x - f(navMinX)), int32(p.z - f(navMinZ))}
		end := [2]int32{int32(q.x - f(navMinX)), int32(q.z - f(navMinZ))}
		var delta, sign [2]int32
		var remaining [2]float64
		major := 0
		for axis := 0; axis < 2; axis++ {
			delta[axis] = end[axis] - start[axis]
			remaining[axis] = f32(float64(start[axis]%step) / f(navStep))
			sign[axis] = -1
			if delta[axis] >= 0 {
				sign[axis] = 1
				remaining[axis] = f32(1 - remaining[axis])
			} else {
				delta[axis] = -delta[axis]
			}
			if delta[axis] > delta[major] {
				major = axis
			}
		}
		minor := 1 - major
		slope := int32(f32(float64(delta[minor])/float64(delta[major])) * f(navStep))
		errorTerm := int32(float64(step)*remaining[minor] - float64(slope)*remaining[major])
		steps := (delta[major] + step - 1) / step
		pos := [2]int32{px, pz}
		height := py
		for i := int32(0); i < steps; i++ {
			packed, found := lookup(pos[0], pos[1])
			if !found || !valid(packed, height) {
				return false
			}
			height = int32(packed & 65535)
			if pos[0] == qx && pos[1] == qz {
				break
			}
			axis := major
			if errorTerm < 0 {
				axis = minor
				errorTerm += step
				i--
			} else {
				errorTerm -= slope
			}
			// 0x45aee0: directional bits are N,NE,E,SE,S,SW,W,NW.
			direction := uint32(0)
			if axis == 0 {
				direction = 2
			}
			if sign[axis] < 0 {
				direction += 4
			}
			mask := packed >> 16 & 255
			if packed&0x1000000 != 0 && mask != 0 && mask&(1<<direction) == 0 {
				return false
			}
			pos[axis] += sign[axis]
		}
		// 0x45aa82: the terminal height tolerance is inclusive.
		d := qy - height
		return d >= -tolerance && d <= tolerance
	}
	distance2 := func(p point, first int) float64 {
		x, y, z := f(first)-p.x, f(first+1)-p.y, f(first+2)-p.z
		return z*z + y*y + x*x
	}
	// 0x437a4c..0x437aec: actor search box = center +- spacing*15 (unrounded
	// x87 product). Fixture spatial cell spans [-10000,10000) per axis
	// (0x42f280, truncation toward zero); only class 12 records are collected.
	nearMonster := in[monsterClass] == 12 && in[6] != 0
	for _, v := range []float64{cx, cy, cz} {
		lo := int32((f32(v-spacing*15) + 10000) / 20000)
		hi := int32((f32(v+spacing*15) + 10000) / 20000)
		nearMonster = nearMonster && lo <= 0 && hi >= 0
	}
	out := make([]int64, 151)
	var placed []point
	// Later lattice cells are laid out around the first placed point (0x437e22).
	origin := point{cx, cy, cz}
	for row := 0; row < 10 && uint32(len(placed)) < count; row++ {
		for col := 0; col < cols && uint32(len(placed)) < count; col++ {
			jx, jz := 0.0, 0.0
			if len(placed) != 0 {
				// 0x5ec044 (0.0004) and 0x5ec040 (0.2); only the first jitter spills to binary32
				// (0x437b58), the second stays an x87 extended value.
				jx = f32(float64(r.R(1000))*0.0004000000189989805 - 0.20000000298023224)
				jz = float64(r.R(1000))*0.0004000000189989805 - 0.20000000298023224
			}
			s := jz + spawnSearchColumns[col]
			rf := float64(row) + jx
			dx := f32(-along * s)
			x := f32(f32(origin.x+f32(across*rf)) + dx)
			z := f32(rf*along + origin.z + s*across)
			p := point{x, f32(origin.y + 500), z}
			h := ground(p.y)
			if !validDrop(p.y, h) {
				continue
			}
			p.y = h
			if in[6] == 0 {
				if !nav(p, false) {
					continue
				}
			} else {
				half := f32(spacing * 0.5)
				if !nav(p, true) {
					continue
				}
				limit := f(partyRadius) + half
				if limit*limit > distance2(p, partyX) {
					continue
				}
				blocked := false
				if nearMonster {
					limit = f(monsterRadius) + half
					blocked = distance2(p, monsterX) < limit*limit
				}
				if blocked {
					continue
				}
			}
			if len(placed) != 0 && !connected(p, placed[0]) {
				continue
			}
			if len(placed) == 0 {
				origin = p
			}
			placed = append(placed, p)
		}
	}
	out[0] = int64(len(placed))
	for i, p := range placed {
		if in[7] != 0 {
			p.y = cy
		}
		out[1+3*i], out[2+3*i], out[3+3*i] = bits(p.x), bits(p.y), bits(p.z)
	}
	return out
}
