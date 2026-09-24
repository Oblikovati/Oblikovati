// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"oblikovati.org/math"
)

// The DIRECTIONS a body's own boundary supplies, for measuring how thin its material is.
//
// A body's thinness used to be read off its axis-aligned bounding box, which describes the
// COORDINATE FRAME as much as the body: measured on the RING corpus pair, the same 1e-10 drill is
// 2e-10 thick along Z and 3.4043798713069418 thick after a 37 degree turn about (1,1,0), so the
// boolean's size classification refused it in one orientation and built it in the other (#3524).
//
// The fix is to keep the MEASURE the bounding box made — the body's own width, its extent along a
// direction — and take the DIRECTIONS from the body instead of from the world. A plane supplies its
// normal, a surface of revolution its axis (and, with it, every direction across that axis), and a
// point set its own principal frame. All three turn with the body, so the width does too.
//
// Each function is one-sided in the same way SurfacesApart is: a value means PROVEN, and ok=false
// means "this surface supplies no direction", never "the material is thick".

// PlanarNormal is a planar surface's normal with a canonical sign, so a plane and the same plane
// facing the other way report ONE direction and a caller can group faces by it. ok=false for every
// other surface kind.
//
// Negation is exact: the flip is a sign change, and normalisation divides by a length identical for
// v and −v, so two exactly opposite face normals canonicalise to the identical unit vector rather
// than to near neighbours.
//
// Example:
//
//	if n, ok := geom.PlanarNormal(f.Geometry()); ok { widths = append(widths, extentAlong(n)) }
func PlanarNormal(s Surface) (math.UnitVector3, bool) {
	p, ok := s.(Plane)
	if !ok {
		return math.UnitVector3{}, false
	}
	dir, err := canonicalDirection(p.Normal())
	if err != nil {
		return math.UnitVector3{}, false
	}
	return dir, true
}

// RevolvedAxisLine is the axis line a surface of revolution turns about: a point on it and its
// canonically-signed direction. A body carrying such a face is at least as wide as that face's own
// circles ACROSS the axis, and as deep as its trim ALONG it, so the axis supplies both a direction
// and the family of directions perpendicular to it.
//
// ok=false for a plane, for a sphere (whose every direction is an axis, so it distinguishes none)
// and for a freeform surface.
func RevolvedAxisLine(s Surface) (math.Point3, math.UnitVector3, bool) {
	switch x := s.(type) {
	case Cylinder:
		return canonicalAxis(x.Origin, x.AxisDir)
	case Cone:
		return canonicalAxis(x.Apex, x.AxisDir)
	case Torus:
		return canonicalAxis(x.Center, x.AxisDir)
	}
	return math.Point3{}, math.UnitVector3{}, false
}

// canonicalAxis gives an axis line its canonical direction, so two faces on the same axis pointing
// opposite ways are one axis.
func canonicalAxis(origin math.Point3, dir math.UnitVector3) (math.Point3, math.UnitVector3, bool) {
	canonical, err := canonicalDirection(dir.AsVector())
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, false
	}
	return origin, canonical, true
}

// PrincipalDirections are the three orthogonal directions of a point set's OWN spread, largest
// first — the frame its scatter matrix diagonalises. It turns with the set, and it depends on no
// face normal being right, which is what makes it a direction source a caller can trust where the
// boundary's own directions are a sample it cannot check.
//
// ok=false for fewer than two points. The frame is NOT unique when two spreads are equal (a
// cylinder's two cross directions, a cube's three): the eigenvectors are then any basis of the
// shared eigenspace, so a caller must expect the CHOICE to move under rotation even though the
// frame as a whole does not — a width taken along it moves by a fraction of the point set's own
// sampling step (measured: 0.15 % to 0.86 % on sub-resolution prisms, #3524).
//
// Example:
//
//	if axes, ok := geom.PrincipalDirections(cloud); ok { thinnest = extentAlong(axes[2]) }
func PrincipalDirections(pts []math.Point3) ([3]math.UnitVector3, bool) {
	var out [3]math.UnitVector3
	if len(pts) < 2 {
		return out, false
	}
	_, vectors := jacobiEigen3(scatterMatrix(pts, centroidOf(pts)))
	for i := range 3 {
		u, err := math.UnitVector3FromVector(math.V3(
			math.Scalar(vectors[0][i]), math.Scalar(vectors[1][i]), math.Scalar(vectors[2][i])))
		if err != nil {
			return out, false
		}
		out[i] = u
	}
	return out, true
}

// canonicalDirection gives a direction and its negation the SAME representative — the first non-zero
// component is made positive.
func canonicalDirection(v math.Vector3) (math.UnitVector3, error) {
	u, err := math.UnitVector3FromVector(v)
	if err != nil {
		return math.UnitVector3{}, err
	}
	if !leadsNegative(u) {
		return u, nil
	}
	return u.Negate(), nil
}

// leadsNegative reports whether a direction's first non-zero component is negative — the total order
// that picks one of the two senses, with no tolerance so the choice is byte-identical across runs.
func leadsNegative(u math.UnitVector3) bool {
	if u.X() != 0 {
		return u.X() < 0
	}
	if u.Y() != 0 {
		return u.Y() < 0
	}
	return u.Z() < 0
}

// DistanceToAxis is the perpendicular distance from p to the line through origin along dir — how far
// off an axis a point sits.
//
// It is invariant under a rigid motion of the whole configuration, which is what makes a width taken
// ACROSS an axis independent of the frame the part happens to be in (#3524).
//
// Example:
//
//	across := 2 * geom.DistanceToAxis(cyl.Origin, cyl.AxisDir, p)
func DistanceToAxis(origin math.Point3, dir math.UnitVector3, p math.Point3) float64 {
	v := origin.VectorTo(p)
	return float64(v.Sub(dir.AsVector().Scale(v.Dot(dir.AsVector()))).Length())
}

// ParallelDirections reports whether two unit directions are parallel, either sense — the test that
// collects the faces of one slab, or the faces on one axis, into a group. It shares its tolerance
// with the cone-family test (see coneAngleWeld, whose comment names both questions).
func ParallelDirections(a, b math.UnitVector3) bool {
	return parallelDirs(a.AsVector(), b.AsVector())
}
