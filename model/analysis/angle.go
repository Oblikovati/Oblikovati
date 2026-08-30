// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Angle measurement (M18-F01 PBI-164, #428): the angle between two entities, mirroring the
// reference GetAngle. Each entity contributes a direction — an edge its chord (tangent for a
// straight edge), a planar face its outward normal — and the result is the angle between those two
// directions, in degrees over [0,180]. A three-vertex form measures the angle at the apex.

// AngleDegrees returns the angle in degrees between two entities (an edge's direction or a planar
// face's normal), over [0,180].
func AngleDegrees(a, b MeasureEntity, q ops.Quality) (float64, error) {
	da, err := entityDirection(a, q)
	if err != nil {
		return 0, err
	}
	db, err := entityDirection(b, q)
	if err != nil {
		return 0, err
	}
	return angleBetweenDeg(da, db), nil
}

// ThreePointAngleDegrees returns the angle at apex between apex→p and apex→r, in degrees over
// [0,180] — the angle of the three vertices with apex in the middle.
func ThreePointAngleDegrees(p, apex, r *topo.Vertex) float64 {
	return angleBetweenDeg(apex.Point().VectorTo(p.Point()), apex.Point().VectorTo(r.Point()))
}

// entityDirection derives the angle direction of an entity: an edge's chord or a planar face's
// outward normal. A vertex has no direction and is rejected.
func entityDirection(e MeasureEntity, q ops.Quality) (math.Vector3, error) {
	switch {
	case e.Edge != nil:
		return edgeDirection(e.Edge, q)
	case e.Face != nil:
		return faceNormal(e.Face)
	}
	return math.Vector3{}, fmt.Errorf("angle: entity must be an edge or a face (got a vertex or nothing)")
}

// edgeDirection is the chord from an edge's first to last tessellated point (the exact direction of
// a straight edge). A closed edge (zero chord) has no single direction and is rejected.
func edgeDirection(e *topo.Edge, q ops.Quality) (math.Vector3, error) {
	pts := ops.TessellateEdge(e, q)
	if len(pts) < 2 {
		return math.Vector3{}, fmt.Errorf("angle: edge has %d tessellation points, need ≥2", len(pts))
	}
	d := pts[0].VectorTo(pts[len(pts)-1])
	if d.LengthSquared() == 0 {
		return math.Vector3{}, fmt.Errorf("angle: closed edge has no single direction")
	}
	return d, nil
}

// faceNormal is the outward normal of a face from its ANALYTIC surface (#3456), not a mesh
// triangle: the surface normal at a representative point on the face, negated when the face's
// material side is opposite the surface normal (Face.Reversed). It is exact and constant for a
// planar face; for a curved face it is the analytic normal at the representative point.
func faceNormal(f *topo.Face) (math.Vector3, error) {
	s := f.Geometry()
	p, ok := faceRepresentativePoint(f)
	if !ok {
		return math.Vector3{}, fmt.Errorf("angle: face has no geometry to take a normal from")
	}
	u, v := s.ParamAt(p)
	n := s.NormalAt(u, v)
	if n.LengthSquared() == 0 {
		return math.Vector3{}, fmt.Errorf("angle: face surface normal is degenerate at %v", p)
	}
	if f.Reversed() {
		n = n.Scale(-1)
	}
	return n, nil
}

// faceRepresentativePoint returns a point on the face at which to evaluate the analytic normal:
// a boundary vertex for a trimmed face, the parameter-domain midpoint for a boundary-less one.
func faceRepresentativePoint(f *topo.Face) (math.Point3, bool) {
	loops := f.Loops()
	if len(loops) == 0 {
		s := f.Geometry()
		uLo, uHi := s.UDomain()
		vLo, vHi := s.VDomain()
		if stdmath.IsInf(uLo, 0) || stdmath.IsInf(uHi, 0) || stdmath.IsInf(vLo, 0) || stdmath.IsInf(vHi, 0) {
			return math.Point3{}, false
		}
		return s.PointAt((uLo+uHi)/2, (vLo+vHi)/2), true
	}
	for _, use := range loops[0].EdgeUses() {
		return use.Edge().StartVertex().Point(), true
	}
	return math.Point3{}, false
}

// angleBetweenDeg is the angle in degrees between two vectors, over [0,180].
func angleBetweenDeg(u, v math.Vector3) float64 {
	lu, lv := float64(u.Length()), float64(v.Length())
	if lu == 0 || lv == 0 {
		return 0
	}
	cos := float64(u.Dot(v)) / (lu * lv)
	cos = stdmath.Max(-1, stdmath.Min(1, cos))
	return stdmath.Acos(cos) * 180 / stdmath.Pi
}
