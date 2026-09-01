// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// DraftFaces tapers the selected faces by angle about a pull direction — the mould-draft
// operation. Each selected face is rotated about its hinge (the line where it meets the
// neutral plane perpendicular to pull through the face's lowest vertex along pull), so the
// face tilts by angle while its base stays put, and the neighbours retrim (via
// [retopo.RebuildWithPlanes]). Faces perpendicular to pull (no hinge) are left unchanged. Sign:
// a NEGATIVE angle leans the face inward going along pull (the mould-release draft — removes
// material toward the far end); a POSITIVE angle leans it outward (adds material). See the
// convention pinned by TestDraftTapersSideFace, which drafts inward with a negative angle.
func DraftFaces(solid *topo.Body, faceKeys [][]byte, pull math.Vector3, angle float64) (*topo.Body, error) {
	return DraftFacesNeutral(solid, faceKeys, pull, nil, angle)
}

// DraftFacesNeutral is DraftFaces with an explicit NEUTRAL PLANE — the fixed reference each drafted
// face pivots about (mould-parting plane). When neutral is non-nil each hinge is the line where the
// face meets it (OCCT BRepOffsetAPI_DraftAngle's neutral-plane construction); when nil the hinge
// falls back to the implicit plane ⊥ pull through the face's lowest vertex (#1801). Faces parallel to
// the neutral plane (no hinge) are left unchanged.
func DraftFacesNeutral(solid *topo.Body, faceKeys [][]byte, pull math.Vector3, neutral *geom.Plane, angle float64) (*topo.Body, error) {
	p, perr := math.UnitVector3FromVector(pull)
	if perr != nil {
		return nil, perr
	}
	// #1802 (ADR-0050 P7): the modifier-visitor tilts the drafted planar faces and re-intersects the
	// neighbours — so a body carrying curved faces from a prior fillet drafts into a valid tapered
	// solid (its fillet edges re-trimmed to arcs) instead of panicking in the plane-only rebuild.
	mod, err := newDraftMod(solid, faceKeys, p, neutral, angle)
	if err != nil {
		return nil, err
	}
	return transform.ModifyBody(solid, mod, "draft"), nil
}

// draftedPlane returns a face's plane rotated by angle about its draft hinge, pivoting on a point of
// that hinge. Returns the unchanged plane when the hinge is undefined (face parallel to pull, or to
// the neutral plane).
func draftedPlane(f *topo.Face, pull math.UnitVector3, neutral *geom.Plane, angle float64) geom.Plane {
	pl := f.Geometry().(geom.Plane)
	hinge, pivot, ok := draftHinge(f, pl, pull, neutral)
	if !ok {
		return pl
	}
	rot := math.Rotation4(angle, hinge, pivot)
	newN := rot.TransformVector(pl.Normal())
	moved, perr := geom.NewPlane(pivot, newN)
	if perr != nil {
		return pl
	}
	return moved
}

// draftHinge returns the rotation axis and a pivot point for drafting face pl: the line where pl meets
// the neutral plane when one is given (the user-chosen parting reference), else the implicit hinge
// pull × normal through the face's lowest vertex along pull. ok=false when no hinge exists.
func draftHinge(f *topo.Face, pl geom.Plane, pull math.UnitVector3, neutral *geom.Plane) (math.UnitVector3, math.Point3, bool) {
	if neutral != nil {
		return neutralHinge(pl, *neutral)
	}
	hinge, err := math.UnitVector3FromVector(pull.AsVector().Cross(pl.Normal()))
	if err != nil {
		return math.UnitVector3{}, math.Point3{}, false // face normal parallel to pull → no draft
	}
	return hinge, lowestVertexAlong(f, pull), true
}

// neutralHinge is the intersection line of the face plane and the neutral plane — the fixed edge the
// drafted face pivots about. Direction = n_face × n_neutral; the pivot is the closest point of that
// line to the world origin (any point on the line is a valid rotation pivot). ok=false when the two
// planes are parallel.
func neutralHinge(face, neutral geom.Plane) (math.UnitVector3, math.Point3, bool) {
	n1, n2 := face.Normal(), neutral.Normal()
	dir, err := math.UnitVector3FromVector(n1.Cross(n2))
	if err != nil {
		return math.UnitVector3{}, math.Point3{}, false // parallel planes
	}
	c := float64(n1.Dot(n2))
	d1 := float64(n1.Dot(face.Origin.AsVector()))
	d2 := float64(n2.Dot(neutral.Origin.AsVector()))
	denom := 1 - c*c
	a, b := (d1-d2*c)/denom, (d2-d1*c)/denom
	pivot := math.P3(0, 0, 0).TranslateBy(n1.Scale(math.Scalar(a))).TranslateBy(n2.Scale(math.Scalar(b)))
	return dir, pivot, true
}

// lowestVertexAlong returns the face vertex with the smallest projection onto pull — a
// point on the face plane that also lies on the neutral plane, hence on the hinge line.
func lowestVertexAlong(f *topo.Face, pull math.UnitVector3) math.Point3 {
	best := stdmath.Inf(1)
	var p math.Point3
	for _, e := range f.Edges() {
		for _, v := range e.Vertices() {
			if d := v.Point().AsVector().Dot(pull.AsVector()); d < best {
				best, p = d, v.Point()
			}
		}
	}
	return p
}
