// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Coaxial ball ∪/−/∩ rod — the ball-stud family (Oblikovati#2036). Before this, no entry in
// curvedExactPaths claimed a sphere (every ruled operand is a cylinder or a cone), so the union of a
// ball and its shank fell through to triangle-soup CSG and shipped an INSCRIBED polyhedron: ~500
// planar faces, no analytic surface left, and a volume 1.3% low that did NOT improve with tessellation
// quality because the deficit was in the B-rep, not the mesh. These tests assert the opposite of each
// of those symptoms — three analytic faces, and a volume AND area that match the closed form to the
// tessellation's own noise floor.

// ballStud is the reference model of #2036, in cm: a Ø10 ball at the origin and a coaxial Ø6 shank
// running along +Y from the centre out to 15 mm.
const (
	ballStudR   = 0.5 // ball radius
	ballStudRod = 0.3 // shank radius
	ballStudLen = 1.5 // shank length, from the ball centre
)

// ballStudSeam is the axial offset of the seam circle: OCCT's √(R_s²−R_c²), here the 3-4-5 leg 0.4.
var ballStudSeam = stdmath.Sqrt(ballStudR*ballStudR - ballStudRod*ballStudRod)

func ballStudOperands(t *testing.T) (ball, rod *topo.Body) {
	t.Helper()
	ball, err := brep.SolidSphere(math.P3(0, 0, 0), ballStudR, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	rod, err = brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), ballStudRod, ballStudLen)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	return ball, rod
}

// TestBallStudBooleansStayAnalytic is the #2036 regression. Each of the four results is measured
// against its closed form in BOTH volume and area — volume alone would pass on a body whose spherical
// face meshed as a flat lid, and area alone on one whose faces were assembled inside out.
func TestBallStudBooleansStayAnalytic(t *testing.T) {
	ball, rod := ballStudOperands(t)
	R, rc, L, d := ballStudR, ballStudRod, ballStudLen, ballStudSeam
	vBall := 4.0 / 3.0 * stdmath.Pi * R * R * R
	vRod := stdmath.Pi * rc * rc * L
	// the plug is the rod up to the seam plane plus the ball's cap above it (height R−d)
	vPlug := stdmath.Pi*rc*rc*d + stdmath.Pi*(R-d)*(R-d)*(R-(R-d)/3)
	nearCap, farCap := 2*stdmath.Pi*R*(R-d), 2*stdmath.Pi*R*(R+d) // spherical zone area = 2πR·height
	disc := stdmath.Pi * rc * rc
	band := func(length float64) float64 { return 2 * stdmath.Pi * rc * length }

	for _, c := range []struct {
		name              string
		op                PartFeatureOperation
		target, tool      *topo.Body
		wantVol, wantArea float64
	}{
		{"ball ∪ rod", Join, ball, rod, vBall + vRod - vPlug, farCap + band(L-d) + disc},
		{"rod ∪ ball", Join, rod, ball, vBall + vRod - vPlug, farCap + band(L-d) + disc},
		{"ball − rod", Cut, ball, rod, vBall - vPlug, farCap + band(d) + disc},
		{"rod − ball", Cut, rod, ball, vRod - vPlug, nearCap + band(L-d) + disc},
		{"ball ∩ rod", Intersect, ball, rod, vPlug, nearCap + band(d) + disc},
	} {
		got, err := Boolean(c.op, c.target, c.tool)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		assertBallStudSolid(t, c.name, got)
		props := BodyGeometryProperties(got, PropertyQuality())
		assertWithin(t, c.name+" volume", props.Volume, c.wantVol)
		assertWithin(t, c.name+" area", props.Area, c.wantArea)
	}
}

// ballStudBand is how far a measured value may sit off its closed form. It is a TESSELLATION budget,
// not a modelling one: at PropertyQuality a sphere's facets under-measure by ~0.02%, and an exact
// B-rep is the only way to land inside this. The CSG fallback this replaced missed by 1.3% and did not
// improve with quality.
const ballStudBand = 1e-3

func assertWithin(t *testing.T, what string, got, want float64) {
	t.Helper()
	if rel := stdmath.Abs(got-want) / want; rel > ballStudBand {
		t.Errorf("%s = %.6f, want %.6f (off by %.3f%%, budget %.2f%%)",
			what, got, want, 100*rel, 100*ballStudBand)
	}
}

// assertBallStudSolid pins the shape of every result in this family: a valid closed manifold solid of
// exactly three ANALYTIC faces — one sphere, one cylinder, one plane. A CSG fallback shows up here as
// hundreds of planar faces long before any volume check runs.
func assertBallStudSolid(t *testing.T, name string, b *topo.Body) {
	t.Helper()
	if r := Validate(b); !r.Valid || !r.Closed || !r.Manifold || !b.IsSolid() {
		t.Fatalf("%s: valid=%v closed=%v manifold=%v orientation=%v solid=%v: %v",
			name, r.Valid, r.Closed, r.Manifold, r.OrientationOK, b.IsSolid(), r.Issues)
	}
	kinds := map[string]int{}
	for _, f := range b.Faces() {
		switch f.Geometry().(type) {
		case geom.Sphere:
			kinds["sphere"]++
		case geom.Cylinder:
			kinds["cylinder"]++
		case geom.Plane:
			kinds["plane"]++
		default:
			t.Errorf("%s: face surface %T is not analytic", name, f.Geometry())
		}
	}
	if kinds["sphere"] != 1 || kinds["cylinder"] != 1 || kinds["plane"] != 1 || len(b.Faces()) != 3 {
		t.Errorf("%s: %d faces %v, want one sphere + one cylinder + one plane", name, len(b.Faces()), kinds)
	}
}

// TestBallStudVolumeConvergesWithQuality is the symptom test from #2036. The CSG fallback's deficit
// was FLAT across tessellation quality — that is what proved the error lived in the B-rep — so an exact
// result must instead CONVERGE as the facets get finer. A refinement that does not move the volume is
// the fallback coming back.
func TestBallStudVolumeConvergesWithQuality(t *testing.T) {
	ball, rod := ballStudOperands(t)
	stud, err := Boolean(Join, ball, rod)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	coarse := BodyGeometryProperties(stud, DefaultQuality()).Volume
	fine := BodyGeometryProperties(stud, PropertyQuality()).Volume
	if fine <= coarse {
		t.Errorf("refining the mesh did not raise the volume (%.6f → %.6f); an inscribed polyhedron's "+
			"deficit is in the B-rep and does not converge", coarse, fine)
	}
}

// TestOffAxisRodDoesNotTakeTheCoaxialPath: the closed form is only valid when the rod's axis passes
// through the ball's centre — off-axis, sphere ∩ cylinder is a quartic space curve, which OCCT itself
// reports as IntAna_NoGeometricSolution. The handler must decline rather than build a circle that is
// not there, leaving the (faceted, but honest) fallback to produce a valid solid.
func TestOffAxisRodDoesNotTakeTheCoaxialPath(t *testing.T) {
	ball, _ := brep.SolidSphere(math.P3(0, 0, 0), ballStudR, "ball")
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
