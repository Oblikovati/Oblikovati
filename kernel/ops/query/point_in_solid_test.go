// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	"math"
	"math/rand"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/tessellate"
	m "oblikovati.org/math"
)

// Regression suite for Oblikovati/Oblikovati#1317: point-in-solid was a single fixed-ray parity test
// that miscounted on grazing edges/vertices (ubiquitous on a closed mesh) and re-tessellated per call.
// It is now the generalized winding number (PointInMesh/windingNumber).

// TestSignedSolidAngleClosedMeshIsQuantized verifies the core identity the winding number rests on:
// the signed solid angle summed over a closed outward mesh is 4π at an interior point and 0 at an
// exterior point (winding 1 vs 0).
func TestSignedSolidAngleClosedMeshIsQuantized(t *testing.T) {
	t.Parallel()
	cube, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(2, 2, 2), "cube")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	mesh, _ := tessellate.TessellateBody(cube, DefaultQuality())
	if w := windingNumber(mesh, m.P3(1, 1, 1)); math.Abs(w-1) > 1e-6 {
		t.Errorf("interior winding = %g, want 1", w)
	}
	if w := windingNumber(mesh, m.P3(5, 5, 5)); math.Abs(w) > 1e-6 {
		t.Errorf("exterior winding = %g, want 0", w)
	}
}

// TestPointInsideRayThroughCornerIsRobust is the direct degeneracy regression: a cube centred so the
// OLD skewed ray (0.5773,0.5774,0.5775) from the interior point passes essentially through the cube's
// far CORNER (a vertex shared by three faces), the exact case that made the single ray miscount.
// TestPointInsideCubeFacePlanes checks classification just inside and just outside the plane of each
// cube face — points the surface-grazing ray handled poorly.
// TestSphereContainmentMatchesAnalytic stress-tests the winding number against the analytic sphere
// membership |p-c| < r over many random points, requiring 100% agreement outside a thin surface shell.
func TestSphereContainmentMatchesAnalytic(t *testing.T) {
	t.Parallel()
	const r = 3.0
	center := m.P3(0, 0, 0)
	sphere, err := brep.SolidSphere(center, r, "sphere")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	mesh, _ := tessellate.TessellateBody(sphere, Quality{ChordTolerance: 0.02, AngleTolerance: 5 * math.Pi / 180})
	shell := 0.1 // exclude points within this band of the surface (chord/sagitta ambiguity)
	rng := rand.New(rand.NewSource(1317))
	tested, mism := 0, 0
	for range 4000 {
		p := m.P3(m.Scalar(rng.Float64()*8-4), m.Scalar(rng.Float64()*8-4), m.Scalar(rng.Float64()*8-4))
		d := float64(p.DistanceTo(center))
		if math.Abs(d-r) < shell {
			continue
		}
		tested++
		if PointInMesh(mesh, p) != (d < r) {
			mism++
		}
	}
	if tested < 1000 {
		t.Fatalf("too few points tested: %d", tested)
	}
	if mism != 0 {
		t.Errorf("%d/%d sphere classifications disagree with analytic membership", mism, tested)
	}
}

// insideTorus is the analytic membership for a torus about +Z at the origin: the point is inside the
// tube when its distance from the central circle of radius major is below the tube radius minor.
func insideTorus(p m.Point3, major, minor float64) bool {
	rho := math.Hypot(float64(p.X), float64(p.Y))
	return (rho-major)*(rho-major)+float64(p.Z)*float64(p.Z) < minor*minor
}

// TestTorusContainmentMatchesAnalytic validates the winding number on a genuinely NON-CONVEX solid
// (a torus — its hole means many exterior points sit "between" surface patches, defeating any
// vertex-only or single-ray test). Compared against the analytic torus membership over random points.
func TestTorusContainmentMatchesAnalytic(t *testing.T) {
	t.Parallel()
	const major, minor = 5.0, 1.5
	torus, err := brep.SolidTorus(m.P3(0, 0, 0), m.V3(0, 0, 1), major, minor, "torus")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	mesh, _ := tessellate.TessellateBody(torus, Quality{ChordTolerance: 0.02, AngleTolerance: 4 * math.Pi / 180})
	// Sanity: the axis centre and the hole are OUTSIDE the solid; the tube core is INSIDE.
	if PointInMesh(mesh, m.P3(0, 0, 0)) {
		t.Error("torus axis centre classified inside (it is in the hole)")
	}
	if !PointInMesh(mesh, m.P3(major, 0, 0)) {
		t.Error("torus tube core classified outside")
	}
	rng := rand.New(rand.NewSource(424242))
	tested, mism := 0, 0
	for range 6000 {
		p := m.P3(m.Scalar(rng.Float64()*16-8), m.Scalar(rng.Float64()*16-8), m.Scalar(rng.Float64()*4-2))
		// Exclude a shell band where faceting vs analytic legitimately disagree.
		rho := math.Hypot(float64(p.X), float64(p.Y))
		band := (rho-major)*(rho-major) + float64(p.Z)*float64(p.Z)
		if math.Abs(math.Sqrt(band)-minor) < 0.12 {
			continue
		}
		tested++
		if PointInMesh(mesh, p) != insideTorus(p, major, minor) {
			mism++
		}
	}
	if tested < 1500 {
		t.Fatalf("too few points tested: %d", tested)
	}
	if mism != 0 {
		t.Errorf("%d/%d torus classifications disagree with analytic membership", mism, tested)
	}
}

// TestSignedSolidAngleDegenerate locks the degenerate guards: a query point coincident with a
// triangle corner, and a zero-area triangle, both contribute 0 (no NaN/Inf from a 0-length vector).
func TestSignedSolidAngleDegenerate(t *testing.T) {
	t.Parallel()
	a, b, c := m.P3(0, 0, 0), m.P3(1, 0, 0), m.P3(0, 1, 0)
	if w := signedSolidAngle(a, a, b, c); w != 0 {
		t.Errorf("solid angle at coincident corner = %g, want 0", w)
	}
	if w := signedSolidAngle(m.P3(0, 0, 1), a, a, a); math.Abs(w) > 1e-12 {
		t.Errorf("solid angle of degenerate triangle = %g, want 0", w)
	}
}

// TestAllVerticesInsideCubeInCube confirms allVerticesInside (the boolean classifier helper) is
// correct for nested and non-nested convex bodies — and, with the tessellation hoisted, returns the
// same verdicts as before for the cases the boolean relies on.
// allVerticesInside reports whether every vertex of inner lies strictly within outer, analytically
// (M48/C3 #3422). It prepares outer ONCE as a brep.InsideQuery and reuses that for every vertex query
// — the analytic analog of tessellating outer once and reusing the mesh (#1317) — so classification
// no longer reads a tessellation. The prepared query dispatches by representation exactly as
// PointInsideBody does (all-planar → generalized winding, curved → nearest-crossing ray), so the
// vertex verdicts match the retired mesh oracle on every consistently-oriented body.
