// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math/big"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/math"
)

// Corpus for the cross-operand coincident-plane reconciliation (#2247). The failing input is a
// pair of coincident-OPPOSITE planar faces from two independently-tessellated operands whose shared
// face carries a sub-ULP jitter off their common plane, which the mesh boolean's exact coplanar
// predicate then reads as two distinct planes. These lock in that the ops-layer snap makes the two
// exactly coincident again (so the exact predicate is correct), stays exact, and does not touch a
// pair that is not really coincident.

// jitterY is a representative sub-ULP tessellation offset off a plane — the ~1e-17 seen when a
// planar face carried from a prior reconstruction is Round()ed off its exact plane.
const jitterY = 1e-16

// quadSoup builds the two triangles of the unit square [0,1]×[0,1] lying in the plane y=yBase,
// each vertex offset in y by the matching entry of jit (nil for none), all carrying tag.
func quadSoup(yBase float64, jit []float64, tag int) meshbool.TaggedSoup {
	c := [4][3]float64{{0, yBase, 0}, {1, yBase, 0}, {1, yBase, 1}, {0, yBase, 1}}
	if jit != nil {
		for i := range c {
			c[i][1] += jit[i]
		}
	}
	p := func(i int) meshbool.Point { return meshbool.FromCoords(c[i][0], c[i][1], c[i][2]) }
	return meshbool.TaggedSoup{
		Tris: [][3]meshbool.Point{{p(0), p(1), p(2)}, {p(0), p(2), p(3)}},
		Tags: []int{tag, tag},
	}
}

// planeRef makes a faceSurfaceRef carrying the plane through origin with the given normal (face is
// nil — the snap reads only the surface).
func planeRef(t *testing.T, origin math.Point3, normal math.Vector3) faceSurfaceRef {
	t.Helper()
	pl, err := geom.NewPlane(origin, normal)
	if err != nil {
		t.Fatalf("NewPlane(%v,%v): %v", origin, normal, err)
	}
	return faceSurfaceRef{surface: pl}
}

func snapRes() geom.Resolution {
	return geom.ResolutionForBox(math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1)))
}

// TestSnapProjectionLandsExactlyOnPlane: the rational projection puts a jittered point BIT-exactly
// on the plane — n·(q-o) is exactly zero, not merely small — for an oblique plane too.
func TestSnapProjectionLandsExactlyOnPlane(t *testing.T) {
	t.Parallel()
	pl := ratPlaneOf(planeSurface(t, math.P3(0.3, -0.2, 0.7), math.V3(0.4, 0.5, -0.6)))
	q := pl.project(meshbool.FromCoords(0.11, 0.22+jitterY, 0.33))
	// residual = n·(q-o), which must be the exact rational zero.
	res := ratDot(pl.n, [3]*big.Rat{
		new(big.Rat).Sub(q.X, pl.o[0]),
		new(big.Rat).Sub(q.Y, pl.o[1]),
		new(big.Rat).Sub(q.Z, pl.o[2]),
	})
	if res.Sign() != 0 {
		t.Fatalf("projected point is not exactly on the plane: residual n·(q-o)=%v, want exactly 0", res)
	}
}

// TestSnapMakesCoincidentOppositeFacesExactlyCoplanar: before the snap the exact predicate reads the
// jittered face as OFF the other operand's plane; after, it reads it exactly ON — the whole point.
func TestSnapMakesCoincidentOppositeFacesExactlyCoplanar(t *testing.T) {
	t.Parallel()
	a := quadSoup(0, []float64{+jitterY, -jitterY, +2 * jitterY, -jitterY}, 0) // operand A, jittered off y=0
	b := quadSoup(0, nil, 1)                                                   // operand B, exact on y=0, OPPOSITE normal
	refs := []faceSurfaceRef{
		planeRef(t, math.P3(0, 0, 0), math.V3(0, 1, 0)),
		planeRef(t, math.P3(0, 0, 0), math.V3(0, -1, 0)),
	}
	bTri := b.Tris[0]
	if orientAll0(bTri, a.Tris[0]) {
		t.Fatal("precondition: jittered face must NOT read as coplanar before the snap")
	}
	snapCoincidentPlanes(&a, &b, refs, 1, snapRes())
	if !orientAll0(bTri, a.Tris[0]) || !orientAll0(bTri, a.Tris[1]) {
		t.Fatal("after snap the jittered face must read as exactly coplanar with the other operand")
	}
}

// TestSnapLeavesNonCoincidentPlanesUntouched: two parallel planes a real distance apart are not a
// coincident pair, so nothing is grouped and no vertex moves.
func TestSnapLeavesNonCoincidentPlanesUntouched(t *testing.T) {
	t.Parallel()
	a := quadSoup(0, []float64{+jitterY, 0, 0, 0}, 0)
	b := quadSoup(0.5, nil, 1) // half a unit away — a different plane
	refs := []faceSurfaceRef{
		planeRef(t, math.P3(0, 0, 0), math.V3(0, 1, 0)),
		planeRef(t, math.P3(0, 0.5, 0), math.V3(0, -1, 0)),
	}
	before := a.Tris[0][0]
	snapCoincidentPlanes(&a, &b, refs, 1, snapRes())
	if !a.Tris[0][0].Equal(before) {
		t.Fatal("a non-coincident plane pair must not be snapped")
	}
}

// TestSnapDeclinesMultiInterfaceCorner: a vertex shared by two DIFFERENT coincident-plane groups is
// a corner of two interfaces; snapping it onto one plane would move it off the other, so both groups
// are left untouched (a named decline — the case stays faceted, never distorted).
func TestSnapDeclinesMultiInterfaceCorner(t *testing.T) {
	t.Parallel()
	// Operand A: one face on y=0, one face on z=0, sharing the edge y=z=0 (vertices at x=0 and x=1).
	yFace := quadSoup(0, []float64{+jitterY, +jitterY, 0, 0}, 0)
	zFaceA := zQuadSoup(0, jitterY, 1)
	a := meshbool.TaggedSoup{
		Tris: append(append([][3]meshbool.Point{}, yFace.Tris...), zFaceA.Tris...),
		Tags: append(append([]int{}, yFace.Tags...), zFaceA.Tags...),
	}
	// Operand B: coincident-opposite partners on BOTH y=0 and z=0.
	yFaceB := quadSoup(0, nil, 2)
	zFaceB := zQuadSoup(0, 0, 3)
	b := meshbool.TaggedSoup{
		Tris: append(append([][3]meshbool.Point{}, yFaceB.Tris...), zFaceB.Tris...),
		Tags: append(append([]int{}, yFaceB.Tags...), zFaceB.Tags...),
	}
	refs := []faceSurfaceRef{
		planeRef(t, math.P3(0, 0, 0), math.V3(0, 1, 0)),  // tag 0: A y=0
		planeRef(t, math.P3(0, 0, 0), math.V3(0, 0, 1)),  // tag 1: A z=0
		planeRef(t, math.P3(0, 0, 0), math.V3(0, -1, 0)), // tag 2: B y=0
		planeRef(t, math.P3(0, 0, 0), math.V3(0, 0, -1)), // tag 3: B z=0
	}
	before := a.Tris[0][0] // a vertex on the shared y=z=0 edge (both interfaces)
	snapCoincidentPlanes(&a, &b, refs, 2, snapRes())
	if !a.Tris[0][0].Equal(before) {
		t.Fatal("a vertex on two coincident interfaces must be left unsnapped (honest decline), not moved")
	}
}

// zQuadSoup builds the unit square [0,1]×[0,1] in the plane z=zBase (x,y span), with an optional
// uniform z jitter, all carrying tag.
func zQuadSoup(zBase, jit float64, tag int) meshbool.TaggedSoup {
	c := [4][3]float64{{0, 0, zBase + jit}, {1, 0, zBase + jit}, {1, 1, zBase + jit}, {0, 1, zBase + jit}}
	p := func(i int) meshbool.Point { return meshbool.FromCoords(c[i][0], c[i][1], c[i][2]) }
	return meshbool.TaggedSoup{
		Tris: [][3]meshbool.Point{{p(0), p(1), p(2)}, {p(0), p(2), p(3)}},
		Tags: []int{tag, tag},
	}
}

// planeSurface builds a geom.Plane, failing the test on a degenerate normal.
func planeSurface(t *testing.T, origin math.Point3, normal math.Vector3) geom.Plane {
	t.Helper()
	pl, err := geom.NewPlane(origin, normal)
	if err != nil {
		t.Fatalf("NewPlane(%v,%v): %v", origin, normal, err)
	}
	return pl
}

// orientAll0 reports whether every vertex of tri lies exactly on the plane through base's 3 vertices.
func orientAll0(base, tri [3]meshbool.Point) bool {
	for _, v := range tri {
		if meshbool.Orient3D(base[0], base[1], base[2], v) != 0 {
			return false
		}
	}
	return true
}
