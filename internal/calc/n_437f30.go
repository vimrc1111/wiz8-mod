package calc

import "math"

// The scene fixture uses an empty geometry index bounded above by groundCeiling,
// one spatial cell containing an optional monster, and four navigation cells.
// All float inputs and coordinate outputs are IEEE binary32 bit patterns.
func init() {
	Nodes["00437f30"] = Node{Addr: "00437f30", In: []string{
		"centerX", "centerY", "centerZ", "heading", "spacing", "count",
		"avoidActors", "preserveHeight", "avoidTriggers", "rings", "requireConnection",
		"groundCeiling", "partyX", "partyY", "partyZ", "partyRadius", "monsterClass",
		"monsterX", "monsterY", "monsterZ", "monsterRadius", "navFloor", "navTolerance",
		"navHorizontalStep", "navVerticalStep", "navMinX", "navMinY", "navMinZ",
		"navMaxX", "navMaxY", "navMaxZ", "triggerX", "triggerY", "triggerZ",
		"triggerInnerRadius", "triggerOuterRadius", "triggerFlags", "navCell1", "navCell2", "navCell3",
	}, Out: spawn437f30Outputs(), F: spawnPositionGridNode}
}

func spawn437f30Outputs() []string {
	names := []string{"placed", "centerY"}
	for i := 0; i < 8; i++ {
		for _, axis := range []string{"X", "Y", "Z"} {
			names = append(names, "position"+string(rune('0'+i))+axis)
		}
	}
	return names
}

func spawnPositionGridNode(in []int64, r Rand) []int64 {
	if len(in) != 40 || in[5] < 0 || in[5] > 8 {
		panic("spawn_position_grid: expected 40 inputs and output capacity 8")
	}
	f := func(i int) float64 { return float64(math.Float32frombits(uint32(in[i]))) }
	f32 := func(x float64) float64 { return float64(float32(x)) }
	bits := func(x float64) int64 { return int64(math.Float32bits(float32(x))) }
	cx, cy, cz, spacing := f(0), f(1), f(2), f(4)
	originalY := cy
	// 0x437f9b, 0x517c94, 0x517cba: half-turn, normalization offset, period.
	a := f32(f(3)+3.141592502593994) + 6.2831852
	const period = 6.283185005187988
	for a < 0 {
		a += period
	}
	for a >= period {
		a -= period
	}
	dx, dz := f32(spacing*math.Cos(a)), f32(spacing*math.Sin(a))
	// 0x420bd0/0x433820: without triangle hits, the scene ceiling clips the query.
	ground := func(y float64) float64 { return min(y, f(11)) }
	// 0x438031, 0x438052, 0x43807a: minimum, absolute and normal height limits.
	validDrop := func(y, h float64) bool {
		d := math.Abs(f32(y - h))
		return d >= 1 && d <= 5000 && (in[7] != 0 || d <= 2500)
	}
	// 0x5ebc40: each ground query is raised by 500 world units.
	cy = f32(cy + 500)
	if h := ground(cy); validDrop(cy, h) {
		cy = h
	}
	type point struct{ x, y, z float64 }
	center := point{cx, cy, cz}
	cell := func(p point) (int32, int32, int32) {
		return int32((p.x - f(25)) / f(23)), int32((p.y-f(26))/f(24)) + 1, int32((p.z - f(27)) / f(23))
	}
	lookup := func(x, z int32) (uint32, bool) {
		switch uint32(z)<<16 + uint32(x) {
		case 0:
			return uint32(in[21]), true
		case 1:
			return uint32(in[37]), true
		case 65536:
			return uint32(in[38]), true
		case 65537:
			return uint32(in[39]), true
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
		tolerance := int32(int16(in[22])) * 2
		if d <= -tolerance || d >= tolerance || radius && packed&0x10000000 != 0 {
			return false
		}
		if radius {
			clearance := float64(packed >> 16 & 255)
			if packed&0x1000000 != 0 {
				clearance = 0
			}
			// 0x463248..0x46325b: strict clearance test.
			return clearance*500+f(23)*0.5 > f32(spacing*0.5)
		}
		return true
	}
	connected := func(p, q point) bool {
		for _, v := range []point{p, q} {
			if v.x < f(25) || v.y < f(26) || v.z < f(27) || v.x > f(28) || v.y > f(29) || v.z > f(30) {
				return false
			}
		}
		px, py, pz := cell(p)
		qx, qy, qz := cell(q)
		tolerance := int32(int16(in[22]))
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
		step := int32(f(23))
		start := [2]int32{int32(p.x - f(25)), int32(p.z - f(27))}
		end := [2]int32{int32(q.x - f(25)), int32(q.z - f(27))}
		var delta, sign [2]int32
		var remaining [2]float64
		major := 0
		for axis := 0; axis < 2; axis++ {
			delta[axis] = end[axis] - start[axis]
			remaining[axis] = f32(float64(start[axis]%step) / f(23))
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
		slope := int32(f32(float64(delta[minor])/float64(delta[major])) * f(23))
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
	distance2 := func(p point, start int) float64 {
		x, y, z := f(start)-p.x, f(start+1)-p.y, f(start+2)-p.z
		return z*z + y*y + x*x
	}
	// 0x51a9bb and 0x51aa05: distance class zero is 2 * 500 world units.
	partyLimit := f32((1000+spacing)*0.5 + f(15))
	// 0x4380a9: actor search half-width = spacing * 15. Fixture spatial cell
	// spans [-10000,10000) on each axis (0x42f280, truncation toward zero).
	nearMonster := in[16] == 12
	for _, v := range []float64{cx, cy, cz} {
		lo := int32((f32(v-f32(spacing*15)) + 10000) / 20000)
		hi := int32((f32(v+f32(spacing*15)) + 10000) / 20000)
		nearMonster = nearMonster && lo <= 0 && hi >= 0
	}
	out := make([]int64, 26)
	out[1] = bits(cy)
	var placed []point
	firstX, firstZ := 9999, 9999 // 0x437f49: first successful lattice index sentinel.
	lo, hi := 0, 0
	for hi <= int(int32(in[9])) && len(placed) < int(in[5]) {
		first := false
		for x := lo; x <= hi && len(placed) < int(in[5]) && !first; x++ {
			for z := lo; z <= hi && len(placed) < int(in[5]) && !first; z++ {
				if x > lo && x < hi && z > lo && z < hi || x == firstX && z == firstZ {
					continue
				}
				jx, jz := 0.0, 0.0
				if len(placed) != 0 {
					// 0x5ec050 and 0x5ebc3c; only the first jitter spills to binary32.
					jx = f32(float64(r.R(1000))*0.00020000000949949026 - 0.10000000149011612)
					jz = float64(r.R(1000))*0.00020000000949949026 - 0.10000000149011612
				}
				xf, zf := float64(x)+jx, float64(z)+jz
				p := point{f32(f32(f32(dz*xf)+cx) + f32(-dx*zf)), f32(cy + 500), f32(dx*xf + cz + dz*zf)}
				h := ground(p.y)
				if !validDrop(p.y, h) {
					continue
				}
				p.y = h
				if !nav(p, in[6] != 0) {
					continue
				}
				if in[8] != 0 {
					d := math.Sqrt(distance2(p, 31))
					if d < f(35) && uint32(in[36])&64 == 0 && d >= f(34) {
						continue
					}
				}
				if in[6] != 0 {
					if distance2(p, 12) < partyLimit*partyLimit {
						continue
					}
					limit := f(20) + f32(spacing*0.5)
					if nearMonster && distance2(p, 17) < limit*limit {
						continue
					}
				}
				ok := in[10] == 0 || connected(p, center)
				if !ok {
					for _, prev := range placed {
						if connected(p, prev) {
							ok = true
							break
						}
					}
				}
				if !ok {
					continue
				}
				if len(placed) == 0 {
					first, firstX, firstZ = true, x, z
				}
				placed = append(placed, p)
			}
		}
		if first {
			hi--
			lo++
		}
		hi++
		lo--
	}
	out[0] = int64(len(placed))
	for i, p := range placed {
		if in[7] != 0 {
			p.y = originalY
		}
		out[2+3*i], out[3+3*i], out[4+3*i] = bits(p.x), bits(p.y), bits(p.z)
	}
	return out
}
