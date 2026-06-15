// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Bend Part geometry (M20-F17, #651). A bend folds the material on one side of a sketch
// bend line up around a cylindrical bend region of the given radius through the bend angle.
// This first cut is a planar-faceted approximation for a prismatic body: it splits the body
// at the bend line, takes the cut cross-section, and rebuilds the part as ONE swept solid
// along a straight→arc→straight path — the fixed flange, the faceted bend arc, and the
// rotated moving flange — so the result is a single watertight solid (a sharp fold would be
// non-manifold, hence the arc). A non-prismatic body bends as if it were the prism of its
// cut section (documented approximation); a general free-form bend is a follow-up.

// bendFacetStep caps the bend arc's facet size (~10°) so the cylindrical region is smooth
// without exploding the face count.
const bendFacetStep = stdmath.Pi / 18

// bendSolid bends body about the bend line through (linePoint, lineDir) lying on the part,
// folding the +across side up toward upNormal by angle over bend radius. radius and angle
// are in database units / radians. It returns a single solid, erroring when the bend line
// does not divide the body into two pieces.
func bendSolid(body *topo.Body, linePoint math.Point3, lineDir, upNormal math.Vector3, radius, angle float64, feat string) (*topo.Body, error) {
	frame, err := newBendFrame(linePoint, lineDir, upNormal, radius)
	if err != nil {
		return nil, err
	}
	section, err := bendCutSection(body, frame.cutPlane)
	if err != nil {
		return nil, err
	}
	lo, hi := acrossRange(body, frame.across, linePoint)
	fixedLen, movingLen := frame.at-lo, hi-frame.at
	if fixedLen <= bendTol || movingLen <= bendTol {
		return nil, fmt.Errorf("bend line lies at the body edge (fixed %.4g, moving %.4g)", fixedLen, movingLen)
	}
	sections := bendSections(section, frame, fixedLen, movingLen, angle)
	return sweptSolid(sections, false, feat)
}

// bendTol is the minimum flange length / section size below which a bend is degenerate.
const bendTol = 1e-7

// bendFrame is the resolved bend geometry: the orthonormal axes, the splitting plane, the
// arc axis (parallel to the bend line, offset by the radius toward upNormal), and the bend
// line's coordinate along the across direction.
type bendFrame struct {
	dir, up, across math.UnitVector3
	cutPlane        geom.Plane
	axisOrigin      math.Point3
	at              float64 // linePoint projected onto across
}

// newBendFrame builds the bend frame, erroring on a degenerate (parallel) line/normal pair.
func newBendFrame(linePoint math.Point3, lineDir, upNormal math.Vector3, radius float64) (bendFrame, error) {
	d, err := math.UnitVector3FromVector(lineDir)
	if err != nil {
		return bendFrame{}, fmt.Errorf("bend line direction is degenerate: %v", lineDir)
	}
	n, err := math.UnitVector3FromVector(upNormal)
	if err != nil {
		return bendFrame{}, fmt.Errorf("bend up-normal is degenerate: %v", upNormal)
	}
	across, err := math.UnitVector3FromVector(d.AsVector().Cross(n.AsVector()))
	if err != nil {
		return bendFrame{}, fmt.Errorf("bend line direction %v is parallel to the up-normal %v", lineDir, upNormal)
	}
	plane, err := geom.NewPlane(linePoint, across.AsVector())
	if err != nil {
		return bendFrame{}, err
	}
	return bendFrame{
		dir: d, up: n, across: across, cutPlane: plane,
		axisOrigin: linePoint.TranslateBy(n.AsVector().Scale(radius)),
		at:         across.AsVector().Dot(linePoint.AsVector()),
	}, nil
}

// bendSections lays out the swept-path cross sections: the fixed flange (a straight prism
// from the fixed end to the bend line), the faceted bend arc, and the moving flange (a
// straight prism off the arc's end). The arc and the moving flange rotate about the bend
// axis so the moving side folds up toward the up-normal.
func bendSections(section []math.Point3, f bendFrame, fixedLen, movingLen, angle float64) [][]math.Point3 {
	out := [][]math.Point3{translatedSection(section, f.across.AsVector().Scale(-fixedLen)), section}
	steps := int(stdmath.Max(2, stdmath.Round(angle/bendFacetStep)))
	step := angle / float64(steps)
	var last []math.Point3
	for k := 1; k <= steps; k++ {
		rot := math.Rotation4(-step*float64(k), f.dir, f.axisOrigin) // negative folds toward +up
		last = transformedSection(section, rot)
		out = append(out, last)
	}
	movingDir := math.Rotation4(-angle, f.dir, f.axisOrigin).TransformVector(f.across.AsVector())
	out = append(out, translatedSection(last, movingDir.Scale(movingLen)))
	return out
}

// translatedSection / transformedSection copy a section under a translation / full transform.
func translatedSection(section []math.Point3, by math.Vector3) []math.Point3 {
	out := make([]math.Point3, len(section))
	for i, p := range section {
		out[i] = p.TranslateBy(by)
	}
	return out
}

func transformedSection(section []math.Point3, m math.Matrix4) []math.Point3 {
	out := make([]math.Point3, len(section))
	for i, p := range section {
		out[i] = m.TransformPoint(p)
	}
	return out
}

// bendCutSection splits the body at the bend plane and returns the cut cross-section of the
// moving (positive-across) piece as an ordered polygon.
func bendCutSection(body *topo.Body, plane geom.Plane) ([]math.Point3, error) {
	pieces, err := ops.SplitSolidByPlane(body, plane)
	if err != nil {
		return nil, err
	}
	if len(pieces) != 2 {
		return nil, fmt.Errorf("bend line does not divide the body into two pieces (got %d)", len(pieces))
	}
	cap := coplanarCapFace(pieces[1], plane) // positive side is the moving flange
	if cap == nil {
		return nil, fmt.Errorf("bend: could not find the cut cross-section face")
	}
	poly := orderedFacePolygon(cap)
	if len(poly) < 3 {
		return nil, fmt.Errorf("bend: cut cross-section has %d vertices, want ≥3", len(poly))
	}
	return poly, nil
}

// coplanarCapFace returns the body's planar face lying in plane (its normal parallel to the
// plane's and a vertex on it), or nil.
func coplanarCapFace(body *topo.Body, plane geom.Plane) *topo.Face {
	n := plane.Normal()
	for _, f := range body.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			continue
		}
		if stdmath.Abs(pl.Normal().Cross(n).Length()) > 1e-6 {
			continue
		}
		if stdmath.Abs(plane.Origin.VectorTo(pl.Origin).Dot(n)) < 1e-6 {
			return f
		}
	}
	return nil
}

// orderedFacePolygon walks a face's outer loop into an ordered list of model points.
func orderedFacePolygon(f *topo.Face) []math.Point3 {
	for _, l := range f.Loops() {
		if !l.IsOuter() {
			continue
		}
		uses := l.EdgeUses()
		pts := make([]math.Point3, 0, len(uses))
		for _, u := range uses {
			v := u.Edge().StartVertex()
			if u.Reversed() {
				v = u.Edge().EndVertex()
			}
			pts = append(pts, v.Point())
		}
		return pts
	}
	return nil
}

// acrossRange returns the body's extent along the across direction, measured in the same
// coordinate as the bend line's projection (so [lo,hi] brackets the bend position).
func acrossRange(body *topo.Body, across math.UnitVector3, _ math.Point3) (lo, hi float64) {
	a := across.AsVector()
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range body.Vertices() {
		t := a.Dot(v.Point().AsVector())
		lo, hi = stdmath.Min(lo, t), stdmath.Max(hi, t)
	}
	return lo, hi
}
