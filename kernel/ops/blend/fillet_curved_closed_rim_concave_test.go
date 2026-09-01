// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The concave closed-rim arm surfaces and their external-tangency contact circles are exact analytic
// geometry (concave-sphere-cone-arm-derivation.md §1/§2), so these regressions pin them against the
// derivation's hand-computed S2/S5 numbers to the last digit. A wrong offset sign (building the convex
// R−r/inward-cone spindle) would miss the centre/major by tens of units and fail loudly here.

const concaveArmEps = 1e-6

// s5Sphere / s5CapPlane build the S5 fixture geometry: a host sphere R=13 centred at the origin meeting
// its cap plane z=0 (material −z, outward +ẑ).
func s5Sphere() geom.Sphere { return geom.Sphere{Center: math.P3(0, 0, 0), Radius: 13} }
func s5CapPlane() geom.Plane {
	pl, _ := geom.NewPlaneFromAxes(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(0, 1, 0))
	return pl
}

// s2Cone / s2CapPlane build the S2 fixture geometry: a host cone apex (0,0,40), axis −ẑ, tanα=0.25 (the
// measured half-angle), meeting its cap plane z=0 (material −z, outward +ẑ).
func s2Cone() geom.Cone {
	co, _ := geom.NewConeWithRef(math.P3(0, 0, 40), math.V3(0, 0, -1), math.V3(0, -1, 0), stdmath.Atan(0.25))
	return co
}
func s2CapPlane() geom.Plane { return s5CapPlane() }

func TestConcaveSphereArmSurfaceS5(t *testing.T) {
	t.Parallel()
	res := tol.ForSize(30)
	up := math.V3(0, 0, 1).AsUnit()
	tor, reject := concaveSphereArmSurface(s5Sphere(), s5CapPlane(), up, 3, res)
	if reject != sphereArmBuilt {
		t.Fatalf("concaveSphereArmSurface(S5) rejected: %v", reject)
	}
	assertTorus(t, "S5 sphere arm", tor, math.P3(0, 0, 3), math.V3(0, 0, 1), stdmath.Sqrt(247), 3)
}

func TestConcaveConeArmSurfaceS2(t *testing.T) {
	t.Parallel()
	res := tol.ForSize(30)
	up := math.V3(0, 0, 1).AsUnit()
	tor, reject := concaveConeArmSurface(s2Cone(), s2CapPlane(), up, 8, res)
	if reject != coneArmBuilt {
		t.Fatalf("concaveConeArmSurface(S2) rejected: %v", reject)
	}
	assertTorus(t, "S2 cone arm", tor, math.P3(0, 0, 8), math.V3(0, 0, -1), 16.246211, 8)
}

func TestConcaveSphereContactCircleS5(t *testing.T) {
	t.Parallel()
	res := tol.ForSize(30)
	tor, _ := concaveSphereArmSurface(s5Sphere(), s5CapPlane(), math.V3(0, 0, 1).AsUnit(), 3, res)
	c, r, ok := concaveSphereContactCircle(s5Sphere(), tor, res)
	if !ok {
		t.Fatal("concaveSphereContactCircle(S5) rejected the external-tangency torus")
	}
	assertPoint(t, "S5 sphere-contact centre", c, math.P3(0, 0, 2.4375))
	assertScalar(t, "S5 sphere-contact radius", r, 12.769440)
	// the cap-plane contact circle radius is the arm major (the plate hole grows to it)
	_, capR, okc := concaveTorusContactCircle(s5CapPlane(), tor, res)
	if !okc {
		t.Fatal("cap contact circle unresolved")
	}
	assertScalar(t, "S5 plate-contact radius", capR, stdmath.Sqrt(247))
}

func TestConcaveConeContactCircleS2(t *testing.T) {
	t.Parallel()
	res := tol.ForSize(30)
	tor, _ := concaveConeArmSurface(s2Cone(), s2CapPlane(), math.V3(0, 0, 1).AsUnit(), 8, res)
	c, r, ok := concaveConeContactCircle(s2Cone(), tor, res)
	if !ok {
		t.Fatal("concaveConeContactCircle(S2) rejected the external-tangency torus")
	}
	assertPoint(t, "S2 cone-contact centre", c, math.P3(0, 0, 6.059715))
	assertScalar(t, "S2 cone-contact radius", r, 8.485071)
}

// TestConvexContactCircleRejectsConcaveTorus pins the do-no-harm gate: the CONVEX torusContactCircle's
// internal-tangency asserts (ρ=R−r sphere; +r cone) REJECT the concave (external) arm torus, so the
// convex closed-rim band (J1) can never mis-consume a concave arm — the concave path owns its own
// external-tangency branch (concaveTorusContactCircle).
func TestConvexContactCircleRejectsConcaveTorus(t *testing.T) {
	t.Parallel()
	res := tol.ForSize(30)
	sphTor, _ := concaveSphereArmSurface(s5Sphere(), s5CapPlane(), math.V3(0, 0, 1).AsUnit(), 3, res)
	if _, _, ok := torusContactCircle(s5Sphere(), sphTor, res); ok {
		t.Fatal("convex torusContactCircle accepted a concave (external-tangency) sphere torus — J1 byte-identity at risk")
	}
	coneTor, _ := concaveConeArmSurface(s2Cone(), s2CapPlane(), math.V3(0, 0, 1).AsUnit(), 8, res)
	if _, _, ok := torusContactCircle(s2Cone(), coneTor, res); ok {
		t.Fatal("convex torusContactCircle accepted a concave (external-tangency) cone torus")
	}
}

// squarePlateFace is a named fixture: a z=0 plane face whose single outer loop is the ±half square (the
// S2/S5 plate top). Its in-plane half-extent (min distance from the centre to the boundary) is exactly
// half — the spill threshold the concave band gates the plane-contact radius against.
func squarePlateFace(t *testing.T, half float64) *topo.Face {
	t.Helper()
	pl, err := geom.NewPlaneFromAxes(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("squarePlateFace plane: %v", err)
	}
	bld := topo.NewBuilder(true, topo.Lineage{})
	corners := []math.Point3{math.P3(-half, -half, 0), math.P3(half, -half, 0), math.P3(half, half, 0), math.P3(-half, half, 0)}
	verts := make([]*topo.Vertex, len(corners))
	for i, p := range corners {
		verts[i] = bld.AddVertex(p, topo.Lineage{})
	}
	uses := make([]topo.Use, len(corners))
	for i := range corners {
		a, b := verts[i], verts[(i+1)%len(corners)]
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, topo.Lineage{}))
	}
	return bld.AddFace(pl, topo.Lineage{}, topo.OuterLoop(uses...))
}

// TestConcaveBandSpillGate pins the derivation §5 honest-reject boundary: the plane-contact rail radius
// equals the arm major, so on the ±15 plate the concave band FITS a non-spilling major (S5 r=2: 14.866 <
// 15) but REJECTS the spilling corpus radii (S5 r=3: 15.716 > 15; S2 r=8: 16.246 > 15), and the reported
// half-extent is exactly the plate half-width 15 — the offending value the spill reject carries.
func TestConcaveBandSpillGate(t *testing.T) {
	t.Parallel()
	plate := squarePlateFace(t, 15)
	center := math.P3(0, 0, 0)
	for _, tc := range []struct {
		name   string
		radius float64
		fits   bool
	}{
		{"S5 r=2 (major 14.866)", 14.866070, true},
		{"S5 r=3 (major 15.716)", stdmath.Sqrt(247), false},
		{"S2 r=8 (major 16.246)", 16.246211, false},
	} {
		fits, extent := contactCircleFitsFace(plate, center, tc.radius)
		if fits != tc.fits {
			t.Fatalf("%s: contactCircleFitsFace fits=%v, want %v (extent %.4f)", tc.name, fits, tc.fits, extent)
		}
		if stdmath.Abs(extent-15) > concaveArmEps {
			t.Fatalf("%s: reported plate half-extent %.6f, want 15 (the ±15 plate half-width)", tc.name, extent)
		}
	}
}

func assertTorus(t *testing.T, name string, tor geom.Torus, center math.Point3, axis math.Vector3, major, minor float64) {
	t.Helper()
	assertPoint(t, name+" centre", tor.Center, center)
	if stdmath.Abs(stdmath.Abs(float64(tor.AxisDir.AsVector().Dot(axis.AsUnit().AsVector())))-1) > concaveArmEps {
		t.Fatalf("%s axis %v, want ±%v", name, tor.AxisDir, axis)
	}
	assertScalar(t, name+" major", tor.MajorRadius, major)
	assertScalar(t, name+" minor", tor.MinorRadius, minor)
}

func assertPoint(t *testing.T, name string, got, want math.Point3) {
	t.Helper()
	if d := float64(got.DistanceTo(want)); d > concaveArmEps {
		t.Fatalf("%s = %v, want %v (Δ=%g)", name, got, want, d)
	}
}

func assertScalar(t *testing.T, name string, got, want float64) {
	t.Helper()
	if d := stdmath.Abs(got - want); d > concaveArmEps {
		t.Fatalf("%s = %.6f, want %.6f (Δ=%g)", name, got, want, d)
	}
}
