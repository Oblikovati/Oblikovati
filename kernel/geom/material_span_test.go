// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// A plane and the same plane facing the other way must report ONE direction, or a caller cannot
// group the two faces of a slab — and the grouping is what turns a hundred coplanar facets into one
// width.
func TestOppositeFacesOfASlabShareOneDirection(t *testing.T) {
	t.Parallel()
	bottom, err := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, -1))
	if err != nil {
		t.Fatalf("bottom: %v", err)
	}
	top, err := NewPlane(math.P3(0, 0, 4), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("top: %v", err)
	}
	dLo, ok := PlanarNormal(bottom)
	if !ok {
		t.Fatal("a plane must report a normal")
	}
	dHi, ok := PlanarNormal(top)
	if !ok {
		t.Fatal("a plane must report a normal")
	}
	if dLo != dHi {
		t.Errorf("the two faces of a slab report %v and %v; they must canonicalise to one direction", dLo, dHi)
	}
}

// Canonicalisation must be EXACT for two exactly opposite normals, in every octant: a direction and
// its negation normalise by the same length, and the flip is a sign change, so the two agree bit for
// bit rather than to within a tolerance.
func TestCanonicalDirectionIsExactForOppositeNormals(t *testing.T) {
	t.Parallel()
	for _, v := range []math.Vector3{
		math.V3(0, 0, 1), math.V3(1, 1, 1), math.V3(-3, 7, -0.5), math.V3(0, -2, 5),
	} {
		a, err := canonicalDirection(v)
		if err != nil {
			t.Fatalf("canonicalDirection(%v): %v", v, err)
		}
		b, err := canonicalDirection(v.Scale(-1))
		if err != nil {
			t.Fatalf("canonicalDirection(-%v): %v", v, err)
		}
		if a != b {
			t.Errorf("%v canonicalises to %v and its negation to %v; they must be identical", v, a, b)
		}
	}
}

// Each accessor answers only for what it describes: a cylinder is not a plane, and a plane has no
// axis of revolution. A caller reads "no direction here", never a wrong one.
func TestTheDirectionAccessorsDeclineWhatTheyDoNotDescribe(t *testing.T) {
	t.Parallel()
	cyl, err := NewCylinder(math.P3(1, 2, 3), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	if _, ok := PlanarNormal(cyl); ok {
		t.Error("a cylinder has no plane normal")
	}
	plane, err := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	if _, _, ok := RevolvedAxisLine(plane); ok {
		t.Error("a plane is not a surface of revolution")
	}
	sphere, err := NewSphere(math.P3(0, 0, 0), 1)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	if _, _, ok := RevolvedAxisLine(sphere); ok {
		t.Error("a sphere's every direction is an axis, so it distinguishes none")
	}
}

// Two coaxial faces pointing opposite ways along one axis must report ONE axis, so a body with a
// bore and a boss about the same line is measured across that line once.
func TestOppositeAxesCanonicaliseToOne(t *testing.T) {
	t.Parallel()
	up, err := NewCylinder(math.P3(1, 2, 3), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("up: %v", err)
	}
	down, err := NewCylinder(math.P3(1, 2, 9), math.V3(0, 0, -1), 5)
	if err != nil {
		t.Fatalf("down: %v", err)
	}
	_, a, ok := RevolvedAxisLine(up)
	if !ok {
		t.Fatal("a cylinder has an axis")
	}
	_, b, ok := RevolvedAxisLine(down)
	if !ok {
		t.Fatal("a cylinder has an axis")
	}
	if a != b {
		t.Errorf("two faces on one axis report %v and %v; they must canonicalise to one", a, b)
	}
	tor, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	if o, d, ok := RevolvedAxisLine(tor); !ok || o != math.P3(0, 0, 0) || d != a {
		t.Errorf("RevolvedAxisLine(torus) = %v %v %v; want the torus centre and its axis", o, d, ok)
	}
}

// The principal frame is what a body whose boundary names too few directions falls back on, so it
// must find the thin direction of a flat cloud and turn with it.
func TestPrincipalDirectionsFindTheThinDirection(t *testing.T) {
	t.Parallel()
	var cloud []math.Point3
	for i := range 5 {
		for j := range 5 {
			cloud = append(cloud, math.P3(math.Scalar(i), math.Scalar(j), math.Scalar(0.001*float64((i+j)%2))))
		}
	}
	frame, ok := PrincipalDirections(cloud)
	if !ok {
		t.Fatal("a 25-point cloud has a principal frame")
	}
	if thin := stdmath.Abs(float64(frame[2].Z())); thin < 0.999 {
		t.Errorf("the least-spread direction of a flat cloud is its normal; got %v", frame[2])
	}
	if _, ok := PrincipalDirections(cloud[:1]); ok {
		t.Error("one point has no spread and no frame")
	}
}
