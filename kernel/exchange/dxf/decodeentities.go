// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/drawing"
)

// decodeModelEntities walks an ENTITIES section, returning the directly-drawn model-space
// geometry and the model-space INSERTs separately (the caller expands the INSERTs against the
// block set). Paper-space entities (code 67 = 1) are skipped, matching the DWG importer which
// brings in model space only. A type with no decoder is skipped; a decode error is collected
// as a warning so the rest of the drawing survives.
func decodeModelEntities(dr *drawing.Drawing, pairs []pair, bs *blockSet, opts exchange.TranslationOptions) ([]drawing.Entity, []*drawing.Insert, []string, error) {
	var geometry []drawing.Entity
	var inserts []*drawing.Insert
	var warns []string
	groups := splitEntities(pairs)
	for i, g := range groups {
		if err := opts.Report("entities", i, len(groups)); err != nil {
			return geometry, inserts, warns, err
		}
		m := indexByCode(g.body)
		if isPaperSpace(m) {
			continue
		}
		if g.name == "INSERT" {
			inserts = append(inserts, decodeInsert(m, bs))
			continue
		}
		e, err := decodeEntity(g.name, g.body)
		if err != nil {
			warns = append(warns, err.Error())
			continue
		}
		geometry = appendDecodedEntity(dr, geometry, e, m)
	}
	return geometry, inserts, warns, nil
}

// isPaperSpace reports whether an entity is in paper space (code 67 = 1).
func isPaperSpace(m map[int]pair) bool {
	if p, ok := m[67]; ok {
		v, _ := p.integer()
		return v == 1
	}
	return false
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
	case "LWPOLYLINE":
		return decodeLwPolyline(body)
	case "SPLINE":
		return decodeSpline(body)
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

// decodeLwPolyline reads LWPOLYLINE, whose codes repeat per vertex, so it walks the ordered
// body. Each vertex is a 10/20 pair; a 42 that follows a vertex is that vertex's bulge (the
// Bulges slice stays parallel to Points, default 0). 70 bit 1 is closed; 38 is elevation;
// 210/220/230 the extrusion normal.
//
//nolint:funlen,gocyclo // sequential ordered-pair walk; the field handling is the format.
func decodeLwPolyline(b []pair) (drawing.Entity, error) {
	p := &drawing.LwPolyline{Normal: [3]float64{0, 0, 1}}
	var ferr error
	f := readFloat(&ferr)
	var nx, ny, x float64
	for _, pr := range b {
		switch pr.code {
		case 5:
			p.Handle = pr.handle()
		case 70:
			fl, _ := pr.integer()
			p.Closed = fl&1 != 0
		case 38:
			p.Elevation = f(pr)
		case 210:
			nx = f(pr)
		case 220:
			ny = f(pr)
		case 230:
			p.Normal = [3]float64{nx, ny, f(pr)}
		case 10:
			x = f(pr)
		case 20:
			p.Points = append(p.Points, [2]float64{x, f(pr)})
			p.Bulges = append(p.Bulges, 0)
		case 42:
			if n := len(p.Bulges); n > 0 {
				p.Bulges[n-1] = f(pr)
			}
		}
	}
	if ferr != nil {
		return nil, ferr
	}
	return p, nil
}

// decodeSpline reads SPLINE, whose codes repeat, so it walks the ordered body: 40 are the
// knots, 41 the (optional, interleaved) control-point weights, 10/20/30 the control points,
// 11/21/31 the fit points. 70 bit 1 = closed, bit 4 = rational; 71 is the degree. The
// tolerance codes 42/43/44 are ignored (defaults rebuild the curve).
//
//nolint:funlen,gocyclo // sequential ordered-pair walk across both spline scenarios.
func decodeSpline(b []pair) (drawing.Entity, error) {
	s := &drawing.Spline{}
	var ferr error
	f := readFloat(&ferr)
	var cp, fp [3]float64
	for _, pr := range b {
		switch pr.code {
		case 5:
			s.Handle = pr.handle()
		case 70:
			fl, _ := pr.integer()
			s.Closed = fl&1 != 0
			s.Rational = fl&4 != 0
		case 71:
			s.Degree, _ = pr.integer()
		case 40:
			s.Knots = append(s.Knots, f(pr))
		case 41:
			s.Weights = append(s.Weights, f(pr))
		case 10:
			cp[0] = f(pr)
		case 20:
			cp[1] = f(pr)
		case 30:
			cp[2] = f(pr)
			s.ControlPoints = append(s.ControlPoints, cp)
			cp = [3]float64{}
		case 11:
			fp[0] = f(pr)
		case 21:
			fp[1] = f(pr)
		case 31:
			fp[2] = f(pr)
			s.FitPoints = append(s.FitPoints, fp)
			fp = [3]float64{}
		}
	}
	if ferr != nil {
		return nil, ferr
	}
	return s, nil
}

// readFloat returns a parse helper that records the first error into *ferr, so an
// ordered-pair walk can read many reals concisely and report a malformed one once.
func readFloat(ferr *error) func(pair) float64 {
	return func(p pair) float64 {
		v, err := p.float()
		if err != nil && *ferr == nil {
			*ferr = err
		}
		return v
	}
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

// optFloatDefault reads a real for code, or the given default when the code is absent (used
// for INSERT scale factors, which default to 1).
func optFloatDefault(m map[int]pair, code int, def float64) (float64, error) {
	if p, ok := m[code]; ok {
		return p.float()
	}
	return def, nil
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

// appendDecodedEntity adds a decoded entity to the geometry, recording the formatting it carried
// in the file (#2015). A nil entity is a type with no decoder and is skipped.
func appendDecodedEntity(dr *drawing.Drawing, geometry []drawing.Entity, e drawing.Entity, m map[int]pair) []drawing.Entity {
	if e == nil {
		return geometry
	}
	recordEntityStyle(dr, e, m)
	return append(geometry, e)
}
