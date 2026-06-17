// SPDX-License-Identifier: GPL-2.0-only

package dxf

import "oblikovati.org/kernel/exchange/drawing"

// decodeEntities walks an ENTITIES section's pairs, splitting them into per-entity groups
// (each starts at a code-0 type marker) and decoding each. A type with no decoder is
// skipped; a decode error is collected as a warning so the rest of the drawing survives.
func decodeEntities(pairs []pair) ([]drawing.Entity, []string) {
	var out []drawing.Entity
	var warns []string
	for _, g := range splitEntities(pairs) {
		e, err := decodeEntity(g.name, g.body)
		if err != nil {
			warns = append(warns, err.Error())
			continue
		}
		if e != nil {
			out = append(out, e)
		}
	}
	return out, warns
}

// entityGroup is one entity's name (the code-0 value) and the group pairs that follow it.
type entityGroup struct {
	name string
	body []pair
}

// splitEntities breaks a pair list into entity groups on each code-0 marker.
func splitEntities(pairs []pair) []entityGroup {
	var groups []entityGroup
	for i := 0; i < len(pairs); {
		if pairs[i].code != 0 {
			i++
			continue
		}
		name := pairs[i].text()
		j := i + 1
		for j < len(pairs) && pairs[j].code != 0 {
			j++
		}
		groups = append(groups, entityGroup{name: name, body: pairs[i+1 : j]})
		i = j
	}
	return groups
}

// decodeEntity dispatches one entity by its DXF type name. It returns (nil, nil) for a type
// with no decoder yet, so the caller skips it.
func decodeEntity(name string, body []pair) (drawing.Entity, error) {
	switch name {
	case "LINE":
		return decodeLine(indexByCode(body))
	case "CIRCLE":
		return decodeCircle(indexByCode(body))
	case "POINT":
		return decodePoint(indexByCode(body))
	case "ARC":
		return decodeArc(indexByCode(body))
	case "ELLIPSE":
		return decodeEllipse(indexByCode(body))
	default:
		return nil, nil
	}
}

// decodeLine reads LINE: start point (10/20/30) and end point (11/21/31).
func decodeLine(m map[int]pair) (drawing.Entity, error) {
	start, err := coord(m, 10, 20, 30)
	if err != nil {
		return nil, err
	}
	end, err := coord(m, 11, 21, 31)
	if err != nil {
		return nil, err
	}
	return &drawing.Line{Handle: handleOf(m), Start: start, End: end}, nil
}

// decodeCircle reads CIRCLE: centre (10/20/30), radius (40), extrusion normal (210/220/230).
func decodeCircle(m map[int]pair) (drawing.Entity, error) {
	center, err := coord(m, 10, 20, 30)
	if err != nil {
		return nil, err
	}
	radius, err := optFloat(m, 40)
	if err != nil {
		return nil, err
	}
	return &drawing.Circle{Handle: handleOf(m), Center: center, Radius: radius, Normal: normalOf(m)}, nil
}

// decodePoint reads POINT: position (10/20/30).
func decodePoint(m map[int]pair) (drawing.Entity, error) {
	pos, err := coord(m, 10, 20, 30)
	if err != nil {
		return nil, err
	}
	return &drawing.Point{Handle: handleOf(m), Position: pos}, nil
}

// decodeArc reads ARC: centre (10/20/30), radius (40), start/end angles (50/51, in DEGREES
// — converted to the model's radians), extrusion normal (210/220/230).
func decodeArc(m map[int]pair) (drawing.Entity, error) {
	center, err := coord(m, 10, 20, 30)
	if err != nil {
		return nil, err
	}
	radius, err := optFloat(m, 40)
	if err != nil {
		return nil, err
	}
	start, err := optFloat(m, 50)
	if err != nil {
		return nil, err
	}
	end, err := optFloat(m, 51)
	if err != nil {
		return nil, err
	}
	return &drawing.Arc{
		Handle: handleOf(m), Center: center, Radius: radius,
		StartAngle: degToRad(start), EndAngle: degToRad(end), Normal: normalOf(m),
	}, nil
}

// decodeEllipse reads ELLIPSE: centre (10/20/30), major-axis endpoint relative to centre
// (11/21/31), axis ratio (40), start/end parametric angles (41/42, already in RADIANS —
// unlike ARC, no conversion), extrusion normal (210/220/230).
func decodeEllipse(m map[int]pair) (drawing.Entity, error) {
	center, err := coord(m, 10, 20, 30)
	if err != nil {
		return nil, err
	}
	major, err := coord(m, 11, 21, 31)
	if err != nil {
		return nil, err
	}
	ratio, err := optFloat(m, 40)
	if err != nil {
		return nil, err
	}
	start, err := optFloat(m, 41)
	if err != nil {
		return nil, err
	}
	end, err := optFloat(m, 42)
	if err != nil {
		return nil, err
	}
	return &drawing.Ellipse{
		Handle: handleOf(m), Center: center, MajorAxis: major, AxisRatio: ratio,
		StartAngle: start, EndAngle: end, Normal: normalOf(m),
	}, nil
}

// indexByCode maps an entity's group pairs by code (first occurrence wins). Suitable for
// entities whose codes do not repeat (LINE/CIRCLE/POINT/ARC/ELLIPSE); LWPOLYLINE and
// SPLINE decoders walk the ordered body directly because their codes repeat.
func indexByCode(b []pair) map[int]pair {
	m := make(map[int]pair, len(b))
	for _, p := range b {
		if _, dup := m[p.code]; !dup {
			m[p.code] = p
		}
	}
	return m
}

// coord reads a 3D coordinate from the three given codes, each defaulting to 0 when absent.
func coord(m map[int]pair, cx, cy, cz int) ([3]float64, error) {
	x, err := optFloat(m, cx)
	if err != nil {
		return [3]float64{}, err
	}
	y, err := optFloat(m, cy)
	if err != nil {
		return [3]float64{}, err
	}
	z, err := optFloat(m, cz)
	if err != nil {
		return [3]float64{}, err
	}
	return [3]float64{x, y, z}, nil
}

// optFloat reads a real for code, or 0 when the code is absent.
func optFloat(m map[int]pair, code int) (float64, error) {
	if p, ok := m[code]; ok {
		return p.float()
	}
	return 0, nil
}

// handleOf returns the entity's hex handle (code 5), or 0 when absent.
func handleOf(m map[int]pair) uint64 {
	if p, ok := m[5]; ok {
		return p.handle()
	}
	return 0
}

// normalOf reads the extrusion direction (210/220/230), defaulting to +Z (a 2D entity).
func normalOf(m map[int]pair) [3]float64 {
	if _, ok := m[210]; !ok {
		return [3]float64{0, 0, 1}
	}
	n, err := coord(m, 210, 220, 230)
	if err != nil {
		return [3]float64{0, 0, 1}
	}
	return n
}
