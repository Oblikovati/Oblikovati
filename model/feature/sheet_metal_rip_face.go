// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Face-based rip geometry (#1965). A rip on a picked RipFace slits it through the thickness; the
// rip TYPE decides the line drawn across the face:
//
//   - faceExtents runs the face's full extent along its long (principal) axis, centred;
//   - singlePoint runs from the picked vertex across the face to the opposite boundary (the
//     direction toward the face centre, so any point — corner or edge — gives a clean cut);
//   - pointToPoint runs between the two picked face vertices.
//
// The cutter is a prism in the face's plane swept through-all, so the same slit machinery serves
// every type; only the 2D line changes.

// ripFace resolves the RipFace, derives the rip line for the type, and subtracts the slit prism.
func (f *SheetMetalRipFeature) ripFace(in Input, gap float64) (Output, error) {
	body, err := lastBody(in, "sheet-metal rip")
	if err != nil {
		return Output{}, err
	}
	faces, heals, err := resolveFaces(body, [][]byte{f.def.FaceKey}, nil)
	if err != nil {
		return Output{}, err
	}
	frame, err := newFaceRipFrame(faces[0])
	if err != nil {
		return Output{}, err
	}
	a, b, err := f.faceRipLine(frame, body)
	if err != nil {
		return Output{}, err
	}
	// The rip face's normal may point either into or out of the material (a bottom face points
	// away from it), so span the thickness BOTH ways — a one-sided through-all off the wrong face
	// sweeps into empty space and cuts nothing.
	sp, err := throughAllSpan(Extent{Type: ThroughAllExtent, Direction: SymmetricDir}, in.Bodies, frame.plane)
	if err != nil {
		return Output{}, err
	}
	tool := buildPrism(ripPolygon(a, b, gap, f.def.GapSide), frame.plane, sp, 0, f.featName)
	bodies, err := combine(in, tool, ops.Cut)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies, Heals: heals}, nil
}

// faceRipFrame is a planar rip face's 2D working frame: its sketch plane (for the cutter prism),
// the projector into that plane's coordinates, and the face's projected vertices and centroid.
type faceRipFrame struct {
	plane    sketch.Plane
	origin   math.Point3
	u, v     math.UnitVector3
	verts    []math.Point2
	centroid math.Point2
}

// newFaceRipFrame builds the working frame of a planar rip face, erroring on a non-planar face or
// one with too few vertices to span a rip line.
func newFaceRipFrame(face *topo.Face) (faceRipFrame, error) {
	pl, ok := facePlane(face)
	if !ok {
		return faceRipFrame{}, fmt.Errorf("sheet-metal rip: RipFace is not planar (a curved rip face is unsupported)")
	}
	plane, err := sketch.NewPlane(pl.Origin, pl.UAxis, pl.VAxis)
	if err != nil {
		return faceRipFrame{}, fmt.Errorf("sheet-metal rip: RipFace has no usable frame: %w", err)
	}
	fr := faceRipFrame{plane: plane, origin: pl.Origin, u: pl.UAxis, v: pl.VAxis}
	for _, p := range faceVertexPoints(face) {
		fr.verts = append(fr.verts, fr.project(p))
	}
	if len(fr.verts) < 3 {
		return faceRipFrame{}, fmt.Errorf("sheet-metal rip: RipFace has %d vertices, need at least 3", len(fr.verts))
	}
	fr.centroid = centroid2(fr.verts)
	return fr, nil
}

// project maps a model-space point to the face's 2D plane coordinates.
func (fr faceRipFrame) project(p math.Point3) math.Point2 {
	w := fr.origin.VectorTo(p)
	return math.P2(w.Dot(fr.u.AsVector()), w.Dot(fr.v.AsVector()))
}

// vertex2 resolves a vertex reference key on the body to its 2D face coordinates, erroring (named
// by what) on a missing key or one that does not resolve.
func (fr faceRipFrame) vertex2(body *topo.Body, key []byte, what string) (math.Point2, error) {
	if len(key) == 0 {
		return math.Point2{}, fmt.Errorf("sheet-metal rip: %s not given", what)
	}
	v, ok := body.FindVertexByKey(key)
	if !ok {
		return math.Point2{}, fmt.Errorf("sheet-metal rip: %s (key %x) did not resolve on the rip face", what, key)
	}
	return fr.project(v.Point()), nil
}

// faceRipLine returns the 2D endpoints of the rip line for the recipe's type.
func (f *SheetMetalRipFeature) faceRipLine(fr faceRipFrame, body *topo.Body) (a, b math.Point2, err error) {
	switch f.def.Type {
	case FaceExtentsRip:
		a, b = faceExtentsLine(fr)
		return a, b, nil
	case SinglePointRip:
		return f.singlePointLine(fr, body)
	default: // PointToPointRip defined by two face vertices
		return f.twoVertexLine(fr, body)
	}
}

// faceExtentsLine is the full-extent rip line: the face's principal (long) axis through its
// centroid, spanning the projection of every vertex onto that axis.
func faceExtentsLine(fr faceRipFrame) (a, b math.Point2) {
	axis := principalAxis2(fr.verts, fr.centroid)
	lo, hi := projSpan(fr.verts, fr.centroid, axis)
	return fr.centroid.TranslateBy(axis.Scale(lo)), fr.centroid.TranslateBy(axis.Scale(hi))
}

// singlePointLine runs from the picked vertex toward the face centre and on to the opposite
// boundary — "from a point to the opposite face edge", well-defined for any point on the face.
func (f *SheetMetalRipFeature) singlePointLine(fr faceRipFrame, body *topo.Body) (a, b math.Point2, err error) {
	p, err := fr.vertex2(body, f.def.PointKey, "single-point rip point")
	if err != nil {
		return a, b, err
	}
	toCentre := p.VectorTo(fr.centroid)
	if toCentre.Length() == 0 {
		return a, b, fmt.Errorf("sheet-metal rip: single-point rip point sits at the face centre, giving no rip direction")
	}
	dir := toCentre.Scale(1 / toCentre.Length())
	reach := projMax(fr.verts, p, dir) + ripOvershoot
	return p, p.TranslateBy(dir.Scale(reach)), nil
}

// twoVertexLine runs between the two picked face vertices — the face form of a point-to-point rip.
func (f *SheetMetalRipFeature) twoVertexLine(fr faceRipFrame, body *topo.Body) (a, b math.Point2, err error) {
	if a, err = fr.vertex2(body, f.def.PointKey, "point-to-point rip point one"); err != nil {
		return a, b, err
	}
	b, err = fr.vertex2(body, f.def.PointTwoKey, "point-to-point rip point two")
	return a, b, err
}

// centroid2 is the arithmetic mean of the 2D points.
func centroid2(pts []math.Point2) math.Point2 {
	var sx, sy float64
	for _, p := range pts {
		sx += float64(p.X)
		sy += float64(p.Y)
	}
	n := float64(len(pts))
	return math.P2(math.Scalar(sx/n), math.Scalar(sy/n))
}

// principalAxis2 is the major-variance direction of the points about c (2D PCA). It is the long
// axis of a rectangular face; a square (isotropic) face falls back to the +X axis deterministically.
func principalAxis2(pts []math.Point2, c math.Point2) math.Vector2 {
	var sxx, sxy, syy float64
	for _, p := range pts {
		dx, dy := float64(p.X-c.X), float64(p.Y-c.Y)
		sxx += dx * dx
		sxy += dx * dy
		syy += dy * dy
	}
	theta := 0.5 * stdmath.Atan2(2*sxy, sxx-syy)
	return math.V2(math.Scalar(stdmath.Cos(theta)), math.Scalar(stdmath.Sin(theta)))
}

// projSpan is the min and max signed projection of the points, relative to origin, onto axis.
func projSpan(pts []math.Point2, origin math.Point2, axis math.Vector2) (lo, hi float64) {
	for i, p := range pts {
		d := float64(origin.VectorTo(p).Dot(axis))
		if i == 0 || d < lo {
			lo = d
		}
		if i == 0 || d > hi {
			hi = d
		}
	}
	return lo, hi
}

// projMax is the largest positive projection of the points, relative to origin, onto dir — how
// far the face reaches in that direction, so a rip started at origin runs to the far boundary.
func projMax(pts []math.Point2, origin math.Point2, dir math.Vector2) float64 {
	m := 0.0
	for _, p := range pts {
		if d := float64(origin.VectorTo(p).Dot(dir)); d > m {
			m = d
		}
	}
	return m
}
