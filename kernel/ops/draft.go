// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// DraftFaces tapers the selected faces by angle about a pull direction — the mould-draft
// operation. Each selected face is rotated about its hinge (the line where it meets the
// neutral plane perpendicular to pull through the face's lowest vertex along pull), so the
// face tilts by angle while its base stays put, and the neighbours retrim (via
// [rebuildWithPlanes]). Faces perpendicular to pull (no hinge) are left unchanged. Sign:
// a NEGATIVE angle leans the face inward going along pull (the mould-release draft — removes
// material toward the far end); a POSITIVE angle leans it outward (adds material). See the
// convention pinned by TestDraftTapersSideFace, which drafts inward with a negative angle.
func DraftFaces(solid *topo.Body, faceKeys [][]byte, pull math.Vector3, angle float64) (*topo.Body, error) {
	sel, err := resolveFaceSet(solid, faceKeys)
	if err != nil {
		return nil, err
	}
	p, perr := math.UnitVector3FromVector(pull)
	if perr != nil {
		return nil, perr
	}
	// #1802: the plane-only rebuild panics on a curved face left by a prior fillet/chamfer.
	// Fail loudly (feature goes Sick) until the Phase-7 curved-draft modifier lands (#1809).
	if err := requirePlanarBody(solid, "draft"); err != nil {
		return nil, err
	}
	return rebuildWithPlanes(solid, "draft", true, func(f *topo.Face) geom.Plane {
		if !sel[f.ID()] {
			return f.Geometry().(geom.Plane)
		}
		return draftedPlane(f, p, angle)
	}), nil
}

// draftedPlane returns a face's plane rotated by angle about its hinge (pull × normal),
// pivoting on the face's lowest vertex along pull. Returns the unchanged plane when the
// face is perpendicular to pull (the hinge is undefined).
func draftedPlane(f *topo.Face, pull math.UnitVector3, angle float64) geom.Plane {
	pl := f.Geometry().(geom.Plane)
	hinge, err := math.UnitVector3FromVector(pull.AsVector().Cross(pl.Normal()))
	if err != nil {
		return pl // face normal parallel to pull → no draft
	}
	pivot := lowestVertexAlong(f, pull)
	rot := math.Rotation4(angle, hinge, pivot)
	newN := rot.TransformVector(pl.Normal())
	moved, perr := geom.NewPlane(pivot, newN)
	if perr != nil {
		return pl
	}
	return moved
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
