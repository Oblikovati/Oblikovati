// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// This file holds the DWG bit-stream geometry decoders: one per curve type, each the exact
// inverse of the corresponding writeGeometry encoder. The entity model they produce
// (Line/Circle/… via the aliases in aliases.go) is the format-neutral kernel/exchange/drawing
// model; only the decoding is DWG-specific.

// decodeEntity decodes one geometry entity's coordinates from a reader positioned
// at its type-specific data (see seekEntityGeometry). It returns (nil, nil) for a
// type whose geometry decoder is not yet implemented, so callers can skip it.
//
//nolint:funlen // one-case-per-type geometry-decode dispatch.
func decodeEntity(r *BitReader, header ObjectHeader, version Version) (Entity, error) {
	var e Entity
	switch header.Type {
	case TypeLine:
		e = decodeLine(r, header.Handle)
	case TypeCircle:
		e = decodeCircle(r, header.Handle)
	case TypeArc:
		e = decodeArc(r, header.Handle)
	case TypePoint:
		e = decodePoint(r, header.Handle)
	case TypeEllipse:
		e = decodeEllipse(r, header.Handle)
	case TypeLwpolyline:
		e = decodeLwPolyline(r, header.Handle, version, header.Size)
	case TypeSpline:
		e = decodeSpline(r, header.Handle, version, header.Size)
	default:
		return nil, nil
	}
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("dwg: %s handle %d geometry: %w", header.Type.Name(), header.Handle, err)
	}
	return e, nil
}

// decodeLine reads LINE geometry (ODA): a z-is-zero flag, then start.x/start.y as
// raw doubles with end.x/end.y as deltas from them, optional z, then thickness and
// extrusion (consumed but unused here).
func decodeLine(r *BitReader, handle uint64) *Line {
	zZero := r.ReadBit() == 1
	sx := r.ReadRD()
	ex := r.ReadDD(sx)
	sy := r.ReadRD()
	ey := r.ReadDD(sy)
	var sz, ez float64
	if !zZero {
		sz = r.ReadRD()
		ez = r.ReadDD(sz)
	}
	r.ReadBT() // thickness
	r.ReadBE() // extrusion
	return &Line{Handle: handle, Start: [3]float64{sx, sy, sz}, End: [3]float64{ex, ey, ez}}
}

// decodeCircle reads CIRCLE geometry: centre, radius, thickness, extrusion.
func decodeCircle(r *BitReader, handle uint64) *Circle {
	center := r.Read3BD()
	radius := r.ReadBD()
	r.ReadBT() // thickness
	normal := r.ReadBE()
	return &Circle{Handle: handle, Center: center, Radius: radius, Normal: normal}
}

// decodeArc reads ARC geometry: centre, radius, thickness, extrusion, then start
// and end angles.
func decodeArc(r *BitReader, handle uint64) *Arc {
	center := r.Read3BD()
	radius := r.ReadBD()
	r.ReadBT() // thickness
	normal := r.ReadBE()
	start := r.ReadBD()
	end := r.ReadBD()
	return &Arc{Handle: handle, Center: center, Radius: radius, StartAngle: start, EndAngle: end, Normal: normal}
}

// decodePoint reads POINT geometry: position, thickness, extrusion, x-angle.
func decodePoint(r *BitReader, handle uint64) *Point {
	x := r.ReadBD()
	y := r.ReadBD()
	z := r.ReadBD()
	r.ReadBT() // thickness
	r.ReadBE() // extrusion
	r.ReadBD() // x-angle
	return &Point{Handle: handle, Position: [3]float64{x, y, z}}
}

// decodeEllipse reads ELLIPSE geometry: centre, major-axis vector, extrusion,
// axis ratio, then start and end angles.
func decodeEllipse(r *BitReader, handle uint64) *Ellipse {
	center := r.Read3BD()
	major := r.Read3BD()
	normal := r.Read3BD()
	ratio := r.ReadBD()
	start := r.ReadBD()
	end := r.ReadBD()
	return &Ellipse{Handle: handle, Center: center, MajorAxis: major, AxisRatio: ratio, StartAngle: start, EndAngle: end, Normal: normal}
}

// lwpolyFlag bits gate the optional LWPOLYLINE fields (ODA).
const (
	lwClosed      = 0x200
	lwConstWidth  = 0x004
	lwElevation   = 0x008
	lwThickness   = 0x002
	lwExtrusion   = 0x001
	lwHasBulges   = 0x010
	lwHasVertexID = 0x400
	lwHasWidths   = 0x020
)

// decodeLwPolyline reads LWPOLYLINE geometry. After the flag-gated header fields,
// the vertices are a 2DD vector (first point full, the rest deltas from the
// previous), then a bulge vector, vertex-id vector (R2010+) and width pairs.
//
//nolint:funlen // sequential flag-gated LWPOLYLINE field reads; length is the format.
func decodeLwPolyline(r *BitReader, handle uint64, version Version, objSize int) *LwPolyline {
	flag := r.ReadBS()
	p := &LwPolyline{Handle: handle, Closed: flag&lwClosed != 0, Normal: [3]float64{0, 0, 1}}
	if flag&lwConstWidth != 0 {
		r.ReadBD() // const width
	}
	if flag&lwElevation != 0 {
		p.Elevation = r.ReadBD()
	}
	if flag&lwThickness != 0 {
		r.ReadBD() // thickness
	}
	if flag&lwExtrusion != 0 {
		p.Normal = r.Read3BD()
	}
	numPoints := r.CheckCount(r.ReadBL(), objSize)
	var numBulges, numVertexIDs, numWidths int
	if flag&lwHasBulges != 0 {
		numBulges = r.CheckCount(r.ReadBL(), objSize)
	}
	if version >= R2010 && flag&lwHasVertexID != 0 {
		numVertexIDs = r.CheckCount(r.ReadBL(), objSize)
	}
	if flag&lwHasWidths != 0 {
		numWidths = r.CheckCount(r.ReadBL(), objSize)
	}
	p.Points = readLwVertices(r, numPoints)
	p.Bulges = make([]float64, numBulges)
	for i := range p.Bulges {
		p.Bulges[i] = r.ReadBD()
	}
	for i := 0; i < numVertexIDs; i++ {
		r.ReadBL() // vertex ids (unused)
	}
	for i := 0; i < numWidths; i++ {
		r.ReadBD() // start width
		r.ReadBD() // end width
	}
	return p
}

// decodeSpline reads SPLINE geometry. Scenario 1 (control points): rational/
// closed/periodic flags, tolerances, knot and control-point counts, a weighted
// flag, then the knot vector and control points (each a 3BD, plus a weight when
// weighted). Scenario 2 (fit points): tolerance, begin/end tangents, then a fit
// point vector. On R2013+ a splineflags/knotparam pair precedes the data and can
// override the scenario.
//
//nolint:funlen // sequential SPLINE field reads across both scenarios; length is the format.
func decodeSpline(r *BitReader, handle uint64, version Version, objSize int) *Spline {
	scenario := r.ReadBL()
	if version >= R2013 {
		splineflags := r.ReadBL()
		knotparam := r.ReadBL()
		if splineflags&1 != 0 {
			scenario = 2
		}
		if knotparam == 15 {
			scenario = 1
		}
	}
	s := &Spline{Handle: handle, Degree: r.ReadBL()}
	if scenario&1 != 0 { // control-point form
		s.Rational = r.ReadBit() == 1
		s.Closed = r.ReadBit() == 1
		r.ReadBit() // periodic
		r.ReadBD()  // knot tolerance
		r.ReadBD()  // control tolerance
		numKnots := r.CheckCount(r.ReadBL(), objSize)
		numCtrl := r.CheckCount(r.ReadBL(), objSize)
		weighted := r.ReadBit() == 1
		s.Knots = make([]float64, numKnots)
		for i := range s.Knots {
			s.Knots[i] = r.ReadBD()
		}
		s.ControlPoints = make([][3]float64, numCtrl)
		if weighted {
			s.Weights = make([]float64, numCtrl)
		}
		for i := 0; i < numCtrl; i++ {
			s.ControlPoints[i] = r.Read3BD()
			if weighted {
				s.Weights[i] = r.ReadBD()
			}
		}
		return s
	}
	// fit-point form
	r.ReadBD()  // fit tolerance
	r.Read3BD() // begin tangent
	r.Read3BD() // end tangent
	numFit := r.CheckCount(r.ReadBL(), objSize)
	s.FitPoints = make([][3]float64, numFit)
	for i := range s.FitPoints {
		s.FitPoints[i] = r.Read3BD()
	}
	return s
}

// readLwVertices reads the LWPOLYLINE 2DD point vector: the first vertex is a full
// 2RD, each subsequent vertex a delta (DD) from the previous on each axis.
func readLwVertices(r *BitReader, n int) [][2]float64 {
	if n <= 0 {
		return nil
	}
	pts := make([][2]float64, n)
	pts[0] = r.Read2RD()
	for i := 1; i < n; i++ {
		x := r.ReadDD(pts[i-1][0])
		y := r.ReadDD(pts[i-1][1])
		pts[i] = [2]float64{x, y}
	}
	return pts
}
