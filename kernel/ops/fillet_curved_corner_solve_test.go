// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// b3CornerCY is B3's corner-ball y-coordinate: C=(10, −√1500, 90) is the common intersection of the
// three arm spines (10²+1500 = 1600 = 40² = R−r spine radius²). Kept exact (−√1500 = −38.729833…),
// not the derivation's 6-figure round −38.7298, so the closed-form station residual is machine-zero.
var b3CornerCY = -stdmath.Sqrt(1500)

// b3CornerArms builds the B3 trihedral corner as the solver consumes it: the corner sphere and the
// three curved arms (torus W∧K, cyl W∧N, planar-cyl K∧N), each wired to its two HOST faces (wall
// cylinder R=50, cap plane z=100, radial plane x=0). Hosts are bare faces — only face.Geometry() is
// read — so no loops are needed. This is the certified geometry of m5-weld-setback-retrim-derivation.
func b3CornerArms(t *testing.T) (geom.Sphere, []edgeFillet) {
	t.Helper()
	c := math.P3(10, b3CornerCY, 90)
	sphere, err := geom.NewSphere(c, 10)
	if err != nil {
		t.Fatalf("build corner sphere: %v", err)
	}
	bld := topo.NewBuilder(true, topo.Lineage{})
	fWall := bld.AddFace(mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 50), topo.Lineage{})
	fCap := bld.AddFace(mustPlane(t, math.P3(0, 0, 100), math.V3(0, 0, 1)), topo.Lineage{})
	fRadial := bld.AddFace(mustPlane(t, math.P3(0, 0, 0), math.V3(1, 0, 0)), topo.Lineage{})
	torus, err := geom.NewTorusWithRef(math.P3(0, 0, 90), math.V3(0, 0, 1), math.V3(1, 0, 0), 40, 10)
	if err != nil {
		t.Fatalf("build torus arm: %v", err)
	}
	cylArm := mustCylinder(t, math.P3(10, b3CornerCY, 0), math.V3(0, 0, 1), 10) // W∧N vertical arm
	planarArm := mustCylinder(t, math.P3(10, 0, 90), math.V3(0, 1, 0), 10)      // K∧N radial arm
	arms := []edgeFillet{
		{a: fWall, b: fCap, armSurface: torus},
		{a: fWall, b: fRadial, armSurface: cylArm},
		{a: fCap, b: fRadial, armSurface: planarArm},
	}
	return sphere, arms
}

func mustCylinder(t *testing.T, o math.Point3, axis math.Vector3, r float64) geom.Cylinder {
	t.Helper()
	cyl, err := geom.NewCylinder(o, axis, r)
	if err != nil {
		t.Fatalf("build cylinder: %v", err)
	}
	return cyl
}

func mustPlane(t *testing.T, o math.Point3, normal math.Vector3) geom.Plane {
	t.Helper()
	p, err := geom.NewPlane(o, normal)
	if err != nil {
		t.Fatalf("build plane: %v", err)
	}
	return p
}

// TestSolveCurvedCorner_B3Stations drives the closed-form station solver: solveCurvedCorner must
// return ok and the three certified setback stations — torus major-angle −75.522°, cyl axial z=90,
// planar-cyl axial y=−√1500 — each the spine parameter where spine(station)=C.
func TestSolveCurvedCorner_B3Stations(t *testing.T) {
	t.Parallel()
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150) // B3 body diagonal ≈ √(50²+50²+100²) = 122.5; 150 covers it
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner (want ok)")
	}
	tol := res.Weld() * sphere.Radius
	wantPhi := -75.522 * stdmath.Pi / 180 // derivation's independent oracle for the torus major angle
	for _, a := range w.arms {
		assertStation(t, a, wantPhi, tol)
	}
}

// assertStation checks one arm's station against its certified value, discriminating the two
// cylinders by axis (ẑ → z=90 vertical arm, ŷ → y=−√1500 radial arm).
func assertStation(t *testing.T, a armSetback, wantPhi, tol float64) {
	t.Helper()
	switch s := a.arm.(type) {
	case geom.Torus:
		if stdmath.Abs(a.station-wantPhi) > 1e-3 { // pure angular assert; ±1e-3 rad per the brief
			t.Fatalf("torus station = %.6f rad, want %.6f (−75.522°) ±1e-3", a.station, wantPhi)
		}
	case geom.Cylinder:
		if stdmath.Abs(float64(s.AxisDir.Z())-1) < 0.5 {
			if stdmath.Abs(a.station-90) > tol {
				t.Fatalf("vertical cyl station = %.9f, want 90 ±%.1e", a.station, tol)
			}
			return
		}
		if stdmath.Abs(a.station-b3CornerCY) > tol {
			t.Fatalf("planar cyl station = %.9f, want %.9f (−√1500) ±%.1e", a.station, b3CornerCY, tol)
		}
	default:
		t.Fatalf("unexpected arm surface %T", a.arm)
	}
}

// TestCurvedClosure_B3 certifies the Gauss–Bonnet closure guard: the solved corner's spherical
// triangle has interior angles {π/2, arccos(−0.25), π/2} and area r²E = 182.348, and
// curvedClosureValid accepts it. The NEGATIVE case proves the guard bites — a rail direction
// perturbed 5° must be rejected (a closure test that passes on a broken triangle is a bad test).
func TestCurvedClosure_B3(t *testing.T) {
	t.Parallel()
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner")
	}
	dirs, ok := tangentDirs(w)
	if !ok {
		t.Fatalf("tangentDirs failed on a valid corner")
	}
	assertClosureGeometry(t, dirs, sphere.Radius, res)
	if !curvedClosureValid(w, res) {
		t.Fatalf("curvedClosureValid rejected the certified B3 corner")
	}
	assertPerturbedRailRejected(t, w, dirs, res)
}

// assertClosureGeometry checks the certified interior angles and area against the derivation.
func assertClosureGeometry(t *testing.T, dirs [3]math.UnitVector3, r float64, res Resolution) {
	t.Helper()
	angles := [3]float64{
		interiorAngle(dirs[0], dirs[1], dirs[2]), // at n_W → π/2
		interiorAngle(dirs[1], dirs[2], dirs[0]), // at n_K → arccos(−0.25)
		interiorAngle(dirs[2], dirs[0], dirs[1]), // at n_N → π/2
	}
	want := [3]float64{stdmath.Pi / 2, stdmath.Acos(-0.25), stdmath.Pi / 2}
	for i, got := range angles {
		if stdmath.Abs(got-want[i]) > 1e-4 {
			t.Fatalf("interior angle[%d] = %.6f rad, want %.6f ±1e-4", i, got, want[i])
		}
	}
	excess := sphericalExcess(dirs)
	if solid := stdmath.Abs(solidAngle(dirs)); stdmath.Abs(excess-solid) > 1e-9 {
		t.Fatalf("Gauss–Bonnet excess %.9f ≠ solid angle %.9f (independent-formula cross-check)", excess, solid)
	}
	area := r * r * excess
	if stdmath.Abs(area-r*r*stdmath.Acos(-0.25)) > res.Weld()*r*r {
		t.Fatalf("triangle area = %.6f, want r²·arccos(−0.25) ±%.1e", area, res.Weld()*r*r)
	}
	if stdmath.Abs(area-182.348) > 1e-2 { // coarse oracle check (182.348 is the 6-figure round of 182.3477)
		t.Fatalf("triangle area = %.4f, want ≈182.348 (oracle)", area)
	}
}

// assertPerturbedRailRejected is the negative case: rotate one rail direction 5° and require the
// closure guard to reject. n_K rotated 5° toward n_W moves its rail endpoint ≈0.87 off T_K, far
// beyond the res.Weld·r endpoint tolerance — the chain (invariant 1) must break.
func assertPerturbedRailRejected(t *testing.T, w cornerWeld, dirs [3]math.UnitVector3, res Resolution) {
	t.Helper()
	rad := 5 * stdmath.Pi / 180
	v := dirs[1].AsVector().Scale(stdmath.Cos(rad)).Add(dirs[0].AsVector().Scale(stdmath.Sin(rad)))
	pert, err := math.UnitVector3FromVector(v)
	if err != nil {
		t.Fatalf("build perturbed rail dir: %v", err)
	}
	w.arms[0].railDir1 = pert // arms[0] is the torus arm; railDir1 is its cap-side rail (n_K)
	if curvedClosureValid(w, res) {
		t.Fatalf("curvedClosureValid accepted a 5°-perturbed rail (the guard did not bite)")
	}
}

// TestSolveCurvedCorner_RejectsOffSpine drives armStation's off-spine guard (cylinderStation, via
// the W∧N vertical-cyl arm) in isolation: perturb ONLY that arm's own spine geometry (its axis
// line, 5 units off in x — origin x=15 instead of x=10), leaving the sphere and every host face
// untouched so the host-tangency guards (hostTangentPoint) stay satisfied and cannot mask this
// one. dist(C, perturbed axis)=5 is ~6 orders past the res.Weld()*50≈7.5e-6 gate at this model
// size, so solveCurvedCorner must decline the corner right there, before any closure math runs.
func TestSolveCurvedCorner_RejectsOffSpine(t *testing.T) {
	t.Parallel()
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	arms[1].armSurface = mustCylinder(t, math.P3(15, b3CornerCY, 0), math.V3(0, 0, 1), 10) // was x=10
	if _, ok := solveCurvedCorner(sphere, arms, res); ok {
		t.Fatalf("solveCurvedCorner accepted an arm spine 5 units off the sphere centre")
	}
}

// torusStationFixture builds a torus (major=40, minor=5, spine plane z=5) and the corner-scale
// used to derive res.Weld()·scale, matching N7's degenerate corner (ADR-C4-4): a center on the
// in-plane spine circle (radius=MajorRadius) but with a nonzero axial offset from the spine
// plane must be an honest decline, not a silently-accepted station.
func torusStationFixture(t *testing.T) (geom.Torus, Resolution, float64) {
	t.Helper()
	tr, err := geom.NewTorusWithRef(math.P3(0, 0, 5), math.V3(0, 0, 1), math.V3(1, 0, 0), 40, 5)
	if err != nil {
		t.Fatalf("build torus arm: %v", err)
	}
	res := geom.ResolutionForSize(150)
	return tr, res, tr.MajorRadius // scale ~ station-solve's cornerRScale magnitude order (R = major+minor)
}

// TestTorusStation_RejectsOffPlane drives the axial-offset guard (ADR-C4-4): N7's degenerate
// corner passes a candidate centre 2·minorRadius off the torus spine plane (z=15 vs the z=5
// plane) that still satisfies the in-plane radius check (|C−centre|_inPlane == MajorRadius).
// Before the guard, torusStation wrongly accepted this and deferred the failure downstream to
// the arm rail bundle; the guard must decline right here, honestly, at the solve.
func TestTorusStation_RejectsOffPlane(t *testing.T) {
	t.Parallel()
	tr, res, scale := torusStationFixture(t)
	c := math.P3(40, 0, 15) // in-plane radius 40 = MajorRadius; axial offset (z−5) = 10 = 2·minorRadius
	if _, ok := torusStation(tr, c, scale, res); ok {
		t.Fatalf("torusStation accepted a centre 2r off the torus spine plane (axial offset=10, want decline)")
	}
}

// TestTorusStation_AcceptsOnPlane is the unchanged-behaviour companion: a centre ON the spine
// plane (axial offset 0) at the same in-plane radius must still be accepted, at angle 0 (it sits
// along the torus's Ref direction).
func TestTorusStation_AcceptsOnPlane(t *testing.T) {
	t.Parallel()
	tr, res, scale := torusStationFixture(t)
	c := math.P3(40, 0, 5) // on the z=5 spine plane, in-plane radius 40 = MajorRadius, along Ref
	phi, ok := torusStation(tr, c, scale, res)
	if !ok {
		t.Fatalf("torusStation rejected a centre on the spine plane (axial offset=0, want accept)")
	}
	if stdmath.Abs(phi) > 1e-9 {
		t.Fatalf("torusStation angle = %.9f rad, want 0 (centre lies along Ref)", phi)
	}
}

// TestSolveCurvedCorner_RejectsTooFewArms drives solveCurvedCorner's arm-count guard: a trihedral
// corner needs three arms to close a spherical triangle, so two arms (a dihedral edge, not a
// corner) must be declined outright, before any station or closure math runs.
func TestSolveCurvedCorner_RejectsTooFewArms(t *testing.T) {
	t.Parallel()
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	if _, ok := solveCurvedCorner(sphere, arms[:2], res); ok {
		t.Fatalf("solveCurvedCorner accepted only 2 arms (want ok=false: <3 cannot close a trihedral corner)")
	}
}

// TestSolveCurvedCorner_RejectsNonTangentHost drives hostTangentPoint's "host not tangent at
// radius r" guard (cylinderTangentPoint): swap the B3 wall host (R=50, tangent to the sphere at
// r=10 since dist(C,axis)=40 and |40−50|=10) for one 5 units wider (R=55, |40−55|=15≠10) so the
// sphere no longer touches it at the corner radius. Both the torus (W∧K) and vertical-cyl (W∧N)
// arms share this wall host, so either's railDir call must decline.
func TestSolveCurvedCorner_RejectsNonTangentHost(t *testing.T) {
	t.Parallel()
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	bld := topo.NewBuilder(true, topo.Lineage{})
	badWall := bld.AddFace(mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 55), topo.Lineage{})
	arms[0].a = badWall // torus arm's wall host, now off the r=10 tangency by 5 units
	arms[1].a = badWall // vertical-cyl arm's wall host, same fault
	if _, ok := solveCurvedCorner(sphere, arms, res); ok {
		t.Fatalf("solveCurvedCorner accepted a host wall not tangent to the sphere at radius r")
	}
}
