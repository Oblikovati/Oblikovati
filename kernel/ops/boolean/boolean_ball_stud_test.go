// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Coaxial ball ∪/−/∩ rod — the ball-stud family (Oblikovati#2036) and its through-rod sibling (#2061).
// Before this, no entry in curvedExactPaths claimed a sphere (every ruled operand is a cylinder or a
// cone), so the union of a ball and its shank fell through to triangle-soup CSG and shipped an INSCRIBED
// polyhedron: ~500 planar faces, no analytic surface left, and a volume 1.3% low that did NOT improve
// with tessellation quality because the deficit was in the B-rep, not the mesh. These tests assert the
// opposite of each of those symptoms — a handful of analytic faces, and a volume AND area that match the
// closed form to the tessellation's own noise floor.

// The reference model of #2036, in cm: a Ø10 ball at the origin and a coaxial Ø6 shank.
const (
	ballStudR    = 0.5 // ball radius
	ballStudRod  = 0.3 // shank radius
	ballStudLen  = 1.5 // shank length, from the ball centre
	ballStudBack = 1.0 // how far the THROUGH rod runs the other way, also clear of the ball
)

// ballStudSeam is the axial offset of the seam circle: OCCT's √(R_s²−R_c²), here the 3-4-5 leg 0.4.
var ballStudSeam = stdmath.Sqrt(ballStudR*ballStudR - ballStudRod*ballStudRod)

// ballStudOperands is the ball and a shank running from the ball centre out to +Y — one cap buried.
func ballStudOperands(t *testing.T) (ball, rod *topo.Body) {
	t.Helper()
	return ballOf(t), rodOf(t, 0, ballStudLen)
}

// throughRodOperands is the ball and a rod that clears it at BOTH ends.
func throughRodOperands(t *testing.T) (ball, rod *topo.Body) {
	t.Helper()
	return ballOf(t), rodOf(t, -ballStudBack, ballStudBack+ballStudLen)
}

func ballOf(t *testing.T) *topo.Body {
	t.Helper()
	ball, err := brep.SolidSphere(math.P3(0, 0, 0), ballStudR, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	return ball
}

func rodOf(t *testing.T, y0, length float64) *topo.Body {
	t.Helper()
	rod, err := brep.SolidCylinder(math.P3(0, math.Scalar(y0), 0), math.V3(0, 1, 0), ballStudRod, length)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	return rod
}

// coaxialCase is one boolean of the family with its closed-form volume, area and face census.
type coaxialCase struct {
	name              string
	op                PartFeatureOperation
	target, tool      *topo.Body
	wantVol, wantArea float64
	faces             map[string]int
}

// TestBallStudBooleansStayAnalytic is the #2036 regression: a rod that ENDS inside the ball. Each result
// is measured against its closed form in BOTH volume and area — volume alone would pass on a body whose
// spherical face meshed as a flat lid, and area alone on one assembled inside out.
func TestBallStudBooleansStayAnalytic(t *testing.T) {
	t.Parallel()
	ball, rod := ballStudOperands(t)
	R, rc, L, d := ballStudR, ballStudRod, ballStudLen, ballStudSeam
	vBall, vRod := ballVolume(R), stdmath.Pi*rc*rc*L
	vPlug := stdmath.Pi*rc*rc*d + sphereCapVolume(R, R-d)
	nearCap, farCap := sphereZoneArea(R, R-d), sphereZoneArea(R, R+d)
	disc, wall := stdmath.Pi*rc*rc, func(length float64) float64 { return 2 * stdmath.Pi * rc * length }
	triple := map[string]int{"sphere": 1, "cylinder": 1, "plane": 1}

	runCoaxialCases(t, []coaxialCase{
		{"ball ∪ rod", Join, ball, rod, vBall + vRod - vPlug, farCap + wall(L-d) + disc, triple},
		{"rod ∪ ball", Join, rod, ball, vBall + vRod - vPlug, farCap + wall(L-d) + disc, triple},
		{"ball − rod", Cut, ball, rod, vBall - vPlug, farCap + wall(d) + disc, triple},
		{"rod − ball", Cut, rod, ball, vRod - vPlug, nearCap + wall(L-d) + disc, triple},
		{"ball ∩ rod", Intersect, ball, rod, vPlug, nearCap + wall(d) + disc, triple},
	})
}

// TestThroughRodBooleansStayAnalytic is the #2061 regression: a rod passing right THROUGH the ball. Its
// ball face is a spherical ZONE straddling the equator of its own band axis — the shape that had no
// analytic mesh, and the reason this extent was declined when #2036 shipped. The bead (ball − rod) is
// the interesting one: a genus-1 solid of just two faces, whose area is the belt plus the bore.
func TestThroughRodBooleansStayAnalytic(t *testing.T) {
	t.Parallel()
	ball, rod := throughRodOperands(t)
	R, rc, d := ballStudR, ballStudRod, ballStudSeam
	hi, lo := ballStudLen-d, ballStudBack-d // the rod's free length beyond each seam
	vBall := ballVolume(R)
	vRod := stdmath.Pi * rc * rc * (ballStudBack + ballStudLen)
	vCore := stdmath.Pi*rc*rc*(2*d) + 2*sphereCapVolume(R, R-d)
	belt, cap := sphereZoneArea(R, 2*d), sphereZoneArea(R, R-d)
	disc, wall := stdmath.Pi*rc*rc, func(length float64) float64 { return 2 * stdmath.Pi * rc * length }

	runCoaxialCases(t, []coaxialCase{
		{"ball ∪ axle", Join, ball, rod, vBall + vRod - vCore,
			belt + wall(hi) + wall(lo) + 2*disc, map[string]int{"sphere": 1, "cylinder": 2, "plane": 2}},
		{"axle ∪ ball", Join, rod, ball, vBall + vRod - vCore,
			belt + wall(hi) + wall(lo) + 2*disc, map[string]int{"sphere": 1, "cylinder": 2, "plane": 2}},
		{"ball − axle (bead)", Cut, ball, rod, vBall - vCore,
			belt + wall(2*d), map[string]int{"sphere": 1, "cylinder": 1}},
		{"axle − ball (two stubs)", Cut, rod, ball, vRod - vCore,
			2*cap + wall(hi) + wall(lo) + 2*disc, map[string]int{"sphere": 2, "cylinder": 2, "plane": 2}},
		{"ball ∩ axle (core)", Intersect, ball, rod, vCore,
			2*cap + wall(2*d), map[string]int{"sphere": 2, "cylinder": 1}},
	})
}

// TestShoulderRodBooleansStayAnalytic is the third extent: the rod stops PART WAY through the ball's
// shoulder, so its end cap is neither inside nor outside — the ball's own surface crosses it, leaving an
// ANNULUS. That is the one contact in this family that is not just the seam circle: a plane∩sphere
// circle enters alongside it, and the ball's surviving surface comes in two pieces (a big cap below the
// seam and a small cap beyond the rod's end) instead of one.
func TestShoulderRodBooleansStayAnalytic(t *testing.T) {
	t.Parallel()
	const stop = 0.45 // between the seam plane at 0.4 and the pole at 0.5
	ball, rod := ballOf(t), rodOf(t, 0, stop)
	R, rc, d := ballStudR, ballStudRod, ballStudSeam
	rho := stdmath.Sqrt(R*R - stop*stop) // the ball's own radius where the rod's cap lands
	vRod := stdmath.Pi * rc * rc * stop
	vShared := shoulderSharedVolume(R, rc, stop)
	bigCap, tipCap := sphereZoneArea(R, R+d), sphereZoneArea(R, R-stop)
	shoulderBelt := sphereZoneArea(R, stop-d)
	wall := func(length float64) float64 { return 2 * stdmath.Pi * rc * length }
	disc, annulus := stdmath.Pi*rc*rc, stdmath.Pi*(rc*rc-rho*rho)
	innerDisc := stdmath.Pi * rho * rho

	runCoaxialCases(t, []coaxialCase{
		{"ball ∪ shoulder rod", Join, ball, rod, ballVolume(R) + vRod - vShared,
			bigCap + tipCap + wall(stop-d) + annulus,
			map[string]int{"sphere": 2, "cylinder": 1, "plane": 1}},
		{"ball − shoulder rod", Cut, ball, rod, ballVolume(R) - vShared,
			bigCap + tipCap + wall(d) + disc + innerDisc,
			map[string]int{"sphere": 2, "cylinder": 1, "plane": 2}},
		{"shoulder rod − ball", Cut, rod, ball, vRod - vShared,
			shoulderBelt + wall(stop-d) + annulus,
			map[string]int{"sphere": 1, "cylinder": 1, "plane": 1}},
		{"ball ∩ shoulder rod", Intersect, ball, rod, vShared,
			shoulderBelt + wall(d) + disc + innerDisc,
			map[string]int{"sphere": 1, "cylinder": 1, "plane": 2}},
	})
}

// shoulderSharedVolume is the material a rod running from the ball centre to station `stop` shares with
// the ball, where stop lands in the shoulder band (d < stop < R). Inside the ball's own circle at that
// station the rod's cap is what limits the solid; outside it the ball's surface is:
//
//	V = π·ρ²·stop + (2π/3)·((R²−ρ²)^{3/2} − (R²−r_c²)^{3/2}),   ρ = √(R²−stop²)
func shoulderSharedVolume(ballR, rodR, stop float64) float64 {
	rho2 := ballR*ballR - stop*stop
	return stdmath.Pi*rho2*stop +
		(2*stdmath.Pi/3)*(stdmath.Pow(ballR*ballR-rho2, 1.5)-stdmath.Pow(ballR*ballR-rodR*rodR, 1.5))
}

// TestShoulderRodVolumeMatchesNumericIntegration pins the closed form above against a direct numeric
// integration of the same region, so a slip in it cannot quietly move what the boolean is measured
// against.
func TestShoulderRodVolumeMatchesNumericIntegration(t *testing.T) {
	t.Parallel()
	const R, rc, stop, n = ballStudR, ballStudRod, 0.45, 400000
	numeric := 0.0
	for i := range n {
		r := rc * (float64(i) + 0.5) / n
		numeric += 2 * stdmath.Pi * r * stdmath.Min(stop, stdmath.Sqrt(R*R-r*r)) * (rc / n)
	}
	if got := shoulderSharedVolume(R, rc, stop); stdmath.Abs(got-numeric)/numeric > 1e-6 {
		t.Errorf("shoulderSharedVolume = %.8f, numeric integration = %.8f", got, numeric)
	}
}

// TestBuriedRodLeavesAnInteriorVoid: a rod ending inside the ball at BOTH ends removes nothing of the
// ball's surface, so cutting it away leaves a sealed cavity — one body of two shells whose volume is the
// ball less the rod. The union is the untouched ball.
func TestBuriedRodLeavesAnInteriorVoid(t *testing.T) {
	t.Parallel()
	ball, rod := ballOf(t), rodOf(t, -0.2, 0.4)
	vBall := ballVolume(ballStudR)
	vRod := stdmath.Pi * ballStudRod * ballStudRod * 0.4
	bored, err := Boolean(Cut, ball, rod)
	if err != nil {
		t.Fatalf("ball − buried rod: %v", err)
	}
	if n := len(bored.Shells()); n != 2 {
		t.Errorf("ball − buried rod is %d shell(s), want 2 (the ball plus the void)", n)
	}
	assertWithin(t, "ball − buried rod volume",
		query.BodyGeometryProperties(bored, PropertyQuality()).Volume, vBall-vRod)
	union, err := Boolean(Join, ball, rod)
	if err != nil {
		t.Fatalf("ball ∪ buried rod: %v", err)
	}
	assertWithin(t, "ball ∪ buried rod volume",
		query.BodyGeometryProperties(union, PropertyQuality()).Volume, vBall)
}

// TestBeadIsGenusOne pins the bead's topology, which no volume or area check can see: a ball with a
// bore right through it is a solid of genus 1, so its Euler characteristic is 2−2·1 = 0. A result that
// closed the bore off, or kept an extra shell, would still measure a plausible volume.
func TestBeadIsGenusOne(t *testing.T) {
	t.Parallel()
	ball, rod := throughRodOperands(t)
	bead, err := Boolean(Cut, ball, rod)
	if err != nil {
		t.Fatalf("ball − axle: %v", err)
	}
	if r := Validate(bead); r.EulerCharacteristic != 0 || len(bead.Shells()) != 1 {
		t.Errorf("bead has χ=%d over %d shell(s), want χ=0 over 1 (a genus-1 solid)",
			r.EulerCharacteristic, len(bead.Shells()))
	}
}

// TestTwoStubsAreTwoShells: cutting the ball out of a rod that passes through it severs the rod, so the
// result is two disjoint lumps — one body, two shells (χ = 2 per shell).
func TestTwoStubsAreTwoShells(t *testing.T) {
	t.Parallel()
	ball, rod := throughRodOperands(t)
	stubs, err := Boolean(Cut, rod, ball)
	if err != nil {
		t.Fatalf("axle − ball: %v", err)
	}
	if n := len(stubs.Shells()); n != 2 {
		t.Errorf("axle − ball is %d shell(s), want 2 (the ball severs the rod)", n)
	}
}

// runCoaxialCases drives each case through the public Boolean and checks the shape, the volume and the
// area. A CSG fallback shows up in the face census long before any measurement runs.
func runCoaxialCases(t *testing.T, cases []coaxialCase) {
	t.Helper()
	for _, c := range cases {
		got, err := Boolean(c.op, c.target, c.tool)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		assertAnalyticSolid(t, c.name, got, c.faces)
		props := query.BodyGeometryProperties(got, PropertyQuality())
		assertWithin(t, c.name+" volume", props.Volume, c.wantVol)
		assertWithin(t, c.name+" area", props.Area, c.wantArea)
	}
}

// ballVolume is 4/3·πR³; sphereCapVolume is a cap of height h on a sphere of radius R; sphereZoneArea is
// the lateral area of a zone of axial height h, which is 2πRh whatever the zone's latitude (Archimedes).
func ballVolume(r float64) float64 { return 4.0 / 3.0 * stdmath.Pi * r * r * r }
func sphereCapVolume(r, h float64) float64 {
	return stdmath.Pi * h * h * (r - h/3)
}
func sphereZoneArea(r, h float64) float64 { return 2 * stdmath.Pi * r * h }

// ballStudBand is how far a measured value may sit off its closed form. It is a TESSELLATION budget, not
// a modelling one: at PropertyQuality a sphere's facets under-measure by ~0.02%, and an exact B-rep is
// the only way to land inside this. The CSG fallback this replaced missed by 1.3% and did not improve
// with quality.
const ballStudBand = 1e-3

func assertWithin(t *testing.T, what string, got, want float64) {
	t.Helper()
	if rel := stdmath.Abs(got-want) / want; rel > ballStudBand {
		t.Errorf("%s = %.6f, want %.6f (off by %.3f%%, budget %.2f%%)",
			what, got, want, 100*rel, 100*ballStudBand)
	}
}

// assertAnalyticSolid pins the shape of a result: a valid closed manifold solid whose faces are exactly
// the expected census of ANALYTIC surfaces. A CSG fallback shows up here as hundreds of planar faces.
func assertAnalyticSolid(t *testing.T, name string, b *topo.Body, want map[string]int) {
	t.Helper()
	if r := Validate(b); !r.Valid || !r.Closed || !r.Manifold || !b.IsSolid() {
		t.Fatalf("%s: valid=%v closed=%v manifold=%v orientation=%v solid=%v: %v",
			name, r.Valid, r.Closed, r.Manifold, r.OrientationOK, b.IsSolid(), r.Issues)
	}
	got := map[string]int{}
	for _, f := range b.Faces() {
		switch f.Geometry().(type) {
		case geom.Sphere:
			got["sphere"]++
		case geom.Cylinder:
			got["cylinder"]++
		case geom.Plane:
			got["plane"]++
		default:
			t.Errorf("%s: face surface %T is not analytic", name, f.Geometry())
		}
	}
	if len(got) != len(want) {
		t.Errorf("%s: faces %v, want %v", name, got, want)
		return
	}
	for kind, n := range want {
		if got[kind] != n {
			t.Errorf("%s: faces %v, want %v", name, got, want)
			return
		}
	}
}

// TestBallStudVolumeIsQualityIndependent is the symptom test from #2036, restated for the analytic
// oracle. It used to prove the result was exact by refining the tessellation and requiring the volume
// to CONVERGE upward, because the CSG fallback's deficit was flat across quality. The measurement is
// no longer tessellated (M48/C3 #3453), so the same fact now shows directly and more strongly: an
// exact result integrates over its analytic B-rep, which means it keeps its curved faces and reports
// the SAME volume at every quality. A faceted fallback has no analytic surfaces to integrate.
func TestBallStudVolumeIsQualityIndependent(t *testing.T) {
	t.Parallel()
	ball, rod := ballStudOperands(t)
	stud, err := Boolean(Join, ball, rod)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if _, ok := query.AnalyticGeometryProperties(stud); !ok {
		t.Fatal("the ball stud declined analytic integration; the faceted fallback is back")
	}
	coarse := query.BodyGeometryProperties(stud, DefaultQuality()).Volume
	fine := query.BodyGeometryProperties(stud, PropertyQuality()).Volume
	if coarse != fine {
		t.Errorf("volume moved with tessellation quality (%.9f → %.9f); an analytic integral cannot", coarse, fine)
	}
	sum := 4.0/3*stdmath.Pi*ballStudR*ballStudR*ballStudR + stdmath.Pi*ballStudRod*ballStudRod*ballStudLen
	if coarse >= sum {
		t.Errorf("union volume %.6f is not below the sum of the operands %.6f", coarse, sum)
	}
}

// TestOffAxisRodDoesNotTakeTheCoaxialPath: the closed form is only valid when the rod's axis passes
// through the ball's centre — off-axis, sphere ∩ cylinder is a quartic space curve, which OCCT itself
// reports as IntAna_NoGeometricSolution. The handler must decline rather than build a circle that is
// not there, leaving the (faceted, but honest) fallback to produce a valid solid.
func TestOffAxisRodDoesNotTakeTheCoaxialPath(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~2s): `make test-corpus`")
	}
	t.Parallel()
	ball := ballOf(t)
	rod, err := brep.SolidCylinder(math.P3(0.2, 0, 0), math.V3(0, 1, 0), ballStudRod, ballStudLen)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	if _, ok := CurvedBoolean(Join, ball, rod); ok {
		t.Fatal("an off-axis rod was claimed by an exact analytic path; its seam is not a circle")
	}
	stud, err := Boolean(Join, ball, rod)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if r := Validate(stud); !r.Valid {
		t.Errorf("the fallback produced an invalid solid: %v", r.Issues)
	}
}
