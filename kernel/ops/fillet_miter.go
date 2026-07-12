// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// cornerMiter is a two-fillet corner: where two filleted edges that SHARE a face meet at a
// vertex (the third edge at that vertex stays sharp), the two rolling-ball cylinders cannot be
// joined by a sphere — instead they mutually trim along a seam curve. shared is the common
// face; sBot is the seam's lower end on the now-shortened sharp edge (the cylinders' common
// tangent point, the new top vertex of that edge); seam samples the intersection curve from the
// shared-face tangent point (sTop = seam[0]) down to sBot, as chords shared by both cylinders so
// the welded result is watertight.
type cornerMiter struct {
	vertex *topo.Vertex
	shared *topo.Face
	sBot   math.Point3
	seam   []math.Point3
}

// miterArm is one filleted edge's rolling-ball data at a miter corner: the cylinder centre at
// the corner (on the axis), the edge direction, and the outer (non-shared) face normal.
type miterArm struct {
	cen  math.Point3
	axis math.Vector3
	nF   math.Vector3
}

// solveMiter builds the seam where two filleted edges that share a face meet at v. Each edge's
// cylinder is tangent to the shared face and to its own outer face; for equal radii the two
// cylinders intersect along an elliptical seam lying in the plane through v that mirror-swaps
// the two outer faces. The seam runs from the shared-face tangent point to the sharp edge.
func solveMiter(v *topo.Vertex, ps []filletPick, r float64) (*cornerMiter, error) {
	shared := sharedFace(ps[0].edge, ps[1].edge)
	if shared == nil {
		return nil, fmt.Errorf("fillet: two filleted edges meeting at a vertex must share a face to miter (none shared)")
	}
	nS, ok := planeNormal(shared)
	if !ok {
		return nil, fmt.Errorf("fillet: miter corner's shared face must be planar")
	}
	a1, err := miterArmOf(ps[0].edge, shared, nS, v, r)
	if err != nil {
		return nil, err
	}
	a2, err := miterArmOf(ps[1].edge, shared, nS, v, r)
	if err != nil {
		return nil, err
	}
	seam, err := sampleMiterSeam(v.Point(), r, nS, a1, a2)
	if err != nil {
		return nil, err
	}
	return &cornerMiter{vertex: v, shared: shared, sBot: seam[len(seam)-1], seam: seam}, nil
}

// miterArmOf solves one edge's rolling-ball frame at the corner: centre v+offDir·r (offDir is
// the per-unit-radius offset into the solid from the shared+outer face pair) and the outer
// face normal. Both faces must be planar.
func miterArmOf(e *topo.Edge, shared *topo.Face, nS math.Vector3, v *topo.Vertex, r float64) (miterArm, error) {
	outer := otherFace(e, shared)
	if outer == nil {
		return miterArm{}, fmt.Errorf("fillet: miter edge has no outer face opposite the shared face")
	}
	nF, ok := planeNormal(outer)
	if !ok {
		return miterArm{}, fmt.Errorf("fillet: miter corner's outer face must be planar")
	}
	axis, err := math.UnitVector3FromVector(e.StartVertex().Point().VectorTo(e.EndVertex().Point()))
	if err != nil {
		return miterArm{}, fmt.Errorf("fillet: degenerate miter edge")
	}
	offDir := nS.Add(nF).Scale(-1 / (1 + nS.Dot(nF)))
	return miterArm{cen: v.Point().TranslateBy(offDir.Scale(r)), axis: axis.AsVector(), nF: nF}, nil
}

// sampleMiterSeam samples the seam between two equal-radius fillet cylinders that share a face.
// The seam is cyl(arm1) ∩ Π, where Π is the plane through v with normal nF1−nF2 (the mirror that
// swaps the two outer faces). Sampling walks the rolling-ball contact direction d from the shared
// face (nS, giving sTop on the shared face) to arm1's outer face (nF1, giving sBot on the sharp
// edge); for each d the axial position λ is solved so the cylinder point lands on Π. For equal
// radii the result lies exactly on both cylinders, so both can weld to the same point list.
func sampleMiterSeam(v math.Point3, r float64, nS math.Vector3, a1, a2 miterArm) ([]math.Point3, error) {
	m := a1.nF.Sub(a2.nF) // Π normal (its scale cancels in the λ ratio)
	denom := a1.axis.Dot(m)
	if stdmath.Abs(denom) < 1e-9 {
		return nil, fmt.Errorf("fillet: degenerate miter (edge axis lies in the seam plane)")
	}
	w := stdmath.Acos(math.Clamp(nS.Dot(a1.nF), -1, 1)) // rolling-ball wedge spanned on arm 1
	k := int(stdmath.Ceil(w / (2 * stdmath.Pi / filletChordsPerTurn)))
	if k < 4 {
		k = 4
	}
	vMinusCen := a1.cen.VectorTo(v) // v − cen1
	out := make([]math.Point3, k+1)
	for j := 0; j <= k; j++ {
		d := slerpVec(nS, a1.nF, float64(j)/float64(k)) // contact direction at this station
		lambda := (vMinusCen.Dot(m) - r*d.Dot(m)) / denom
		out[j] = a1.cen.TranslateBy(a1.axis.Scale(lambda)).TranslateBy(d.Scale(r))
	}
	return out, nil
}

// slerpVec interpolates the unit direction from a to b along the shorter great-circle arc at
// parameter s∈[0,1] (the exact rolling-ball contact direction between two face normals).
func slerpVec(a, b math.Vector3, s float64) math.Vector3 {
	w := stdmath.Acos(math.Clamp(a.Dot(b), -1, 1))
	if w < 1e-9 {
		return a
	}
	sinW := stdmath.Sin(w)
	return a.Scale(stdmath.Sin((1-s)*w) / sinW).Add(b.Scale(stdmath.Sin(s*w) / sinW))
}

// sharedFace returns the face bounding both edges, or nil if they share none.
func sharedFace(e1, e2 *topo.Edge) *topo.Face {
	for _, f1 := range e1.Faces() {
		for _, f2 := range e2.Faces() {
			if f1 == f2 {
				return f1
			}
		}
	}
	return nil
}

// planeNormal returns f's material-OUTWARD unit normal and ok=true when f is planar. It must
// respect face.Reversed() (STEP-imported faces carry an inward plane normal): the miter arm's
// offDir and seam plane are built from these normals, so a raw plane normal flips one arm on an
// imported solid — the corner-path analogue of the edgePlanarFaces fix (commit dbd28339).
func planeNormal(f *topo.Face) (math.Vector3, bool) {
	pl, ok := f.Geometry().(geom.Plane)
	if !ok {
		return math.Vector3{}, false
	}
	return outwardPlaneNormal(f, pl), true
}
