// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Driving a face fillet by its WIDTH rather than its radius (Oblikovati#1887) — Inventor's chordal
// alternative on FaceFilletDefinition.
//
// A machinist's constant-width blend is specified across the gap it fills, not by the ball that
// rolls through it, because the width is what is measured on the part. The two are related by the
// dihedral angle the blend sits in, so the width is not a second geometry: it RESOLVES to a radius
// and the existing rolling-ball fillet does the rest.
//
// For two faces meeting at interior dihedral θ, a ball of radius r touches each face at distance
// r/tan(θ/2) from the edge, and the two tangent points subtend θ at the edge. The chord between
// them — the blend's width — is therefore
//
//	w = 2·(r/tan(θ/2))·sin(θ/2) = 2·r·cos(θ/2),   so   r = w / (2·cos(θ/2)).
//
// On a right-angled corner that gives w = r·√2, the familiar chord across a quarter round.

// faceFilletRadiusForWidth resolves the rolling-ball radius that produces the requested blend width
// across the edges the two face sets share. The dihedral is measured on the shared edge's own two
// faces, so a chain of edges at different angles is refused rather than blended at a radius that is
// only right for one of them — a constant WIDTH across varying dihedrals is not a constant-radius
// fillet, and quietly picking one angle would leave the rest the wrong size.
func faceFilletRadiusForWidth(body *topo.Body, faceKeysA, faceKeysB [][]byte, width float64, feat string) (float64, error) {
	if width <= 0 {
		return 0, fmt.Errorf("%s: width %g must be > 0", feat, width)
	}
	theta, err := sharedDihedralAngle(body, faceKeysA, faceKeysB, feat)
	if err != nil {
		return 0, err
	}
	half := stdmath.Cos(theta / 2)
	if half <= faceFilletWidthTol {
		return 0, fmt.Errorf("%s: the faces meet at %.1f°, where a width names no radius "+
			"(the blend's chord collapses); give a radius instead", feat, theta*180/stdmath.Pi)
	}
	return width / (2 * half), nil
}

// faceFilletWidthTol is how near cos(θ/2) may come to zero before the width→radius relation stops
// being usable. It is scale-free: cos(θ/2) is a ratio, so no model size enters.
const faceFilletWidthTol = 1e-6

// sharedDihedralAngle is the interior dihedral of the edges the two face sets share — the angle the
// blend sits in. Every shared edge must agree, since one radius cannot hold one width across
// several angles.
func sharedDihedralAngle(body *topo.Body, faceKeysA, faceKeysB [][]byte, feat string) (float64, error) {
	edges := sharedEdgeKeys(body, faceKeysA, faceKeysB)
	if len(edges) == 0 {
		return 0, fmt.Errorf("%s: a width-driven blend needs the two face sets to share an edge, "+
			"since the width is measured across the angle they meet at", feat)
	}
	first := 0.0
	for i, key := range edges {
		theta, err := edgeDihedralAngle(body, key, feat)
		if err != nil {
			return 0, err
		}
		if i == 0 {
			first = theta
			continue
		}
		if stdmath.Abs(theta-first) > faceFilletDihedralTol {
			return 0, fmt.Errorf("%s: the shared edges meet at different angles (%.1f° and %.1f°), so no one "+
				"radius holds the width across them; give a radius instead",
				feat, first*180/stdmath.Pi, theta*180/stdmath.Pi)
		}
	}
	return first, nil
}

// faceFilletDihedralTol is how far two shared edges' dihedrals may differ and still count as one
// angle — a hair over a degree of floating-point and meshing noise, well inside any real change of
// angle a designer would draw.
const faceFilletDihedralTol = 1e-3

// edgeDihedralAngle is the interior angle between the two planar faces of one edge. Only planes
// carry a single dihedral; a curved face's angle varies along the edge, so a width across it is not
// one number and is refused rather than sampled at an arbitrary point.
func edgeDihedralAngle(body *topo.Body, key []byte, feat string) (float64, error) {
	edge, ok := body.FindEdgeByKey(key)
	if !ok {
		return 0, fmt.Errorf("%s: shared edge reference lost", feat)
	}
	faces := edge.Faces()
	if len(faces) != 2 {
		return 0, fmt.Errorf("%s: shared edge borders %d faces, not 2", feat, len(faces))
	}
	nA, err := outwardNormalOf(faces[0], feat)
	if err != nil {
		return 0, err
	}
	nB, err := outwardNormalOf(faces[1], feat)
	if err != nil {
		return 0, err
	}
	// Outward normals meeting at angle φ bound an interior dihedral of π − φ: on a box's 90° edge
	// the normals are perpendicular, and π − π/2 is the 90° the blend actually sits in.
	return stdmath.Pi - stdmath.Acos(clampUnit(float64(nA.Dot(nB)))), nil
}

// outwardNormalOf is a planar face's outward unit normal, honouring its sense.
func outwardNormalOf(f *topo.Face, feat string) (math.Vector3, error) {
	if _, planar := f.Geometry().(geom.Plane); !planar {
		return math.Vector3{}, fmt.Errorf("%s: a width-driven blend needs planar faces — a curved face's "+
			"dihedral varies along the edge, so the width is not one number", feat)
	}
	n := f.Geometry().NormalAt(0, 0)
	if f.Reversed() {
		return n.Scale(-1), nil
	}
	return n, nil
}

// clampUnit keeps a unit-vector dot product inside [-1, 1], where Acos is defined; floating-point
// noise can push it a hair outside and turn the angle into NaN.
func clampUnit(x float64) float64 {
	return stdmath.Max(-1, stdmath.Min(1, x))
}
