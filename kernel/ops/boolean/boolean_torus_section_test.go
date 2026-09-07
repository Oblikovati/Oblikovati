// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The torus against an axis-invariant quadric (ADR-0061 stage 5, second slice).
//
// A torus is quartic and has no implicit quadric of its own, so the ruled bucket cannot reach it. The
// substitution runs the other way instead: the torus's own chart is affine in its azimuth direction, so
// a quadric whose quadratic form is invariant about the torus axis reduces to ONE harmonic there and the
// two azimuths are an arccos (geom.TorusQuadricArc). That family is a sphere anywhere, and a cylinder or
// cone whose axis is parallel to the torus's — a ball meeting a ring, and an axial hole or boss through
// one. A skew rod is not in it and is still refused by name.

// ringAndBall is the fixture: a ring of major radius 5 and minor 1.5 about z, and a ball of radius 2
// centred on the tube's own centre circle, so it straddles the ring's surface.
func ringAndBall(t *testing.T) (ring, ball *topo.Body) {
	t.Helper()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	ball, err = brep.SolidSphere(math.P3(5, 0, 0), 2, "ball")
	if err != nil {
		t.Fatalf("ball: %v", err)
	}
	return ring, ball
}

// TestRingAndBallBooleansAgreeWithRequicha drives all three operations over the ring and the ball, and
// checks each result against the OTHER two rather than against a number: Requicha's identity
// V(A∪B) = V(A) + V(B) − V(A∩B) ties them, and V(A−B) = V(A) − V(A∩B) ties the third. Three bodies built
// by three separate trims of the same section have no reason to agree unless the section is right.
func TestRingAndBallBooleansAgreeWithRequicha(t *testing.T) {
	t.Parallel()
	ring, ball := ringAndBall(t)
	vol := func(op ops.PartFeatureOperation) float64 {
		res, err := ops.Boolean(op, ring, ball)
		if err != nil {
			t.Fatalf("%v: %v", op, err)
		}
		if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
			t.Fatalf("%v: not a valid closed manifold solid: %+v", op, r)
		}
		for _, f := range res.Faces() {
			switch f.Geometry().(type) {
			case geom.Torus, geom.Sphere:
			default:
				t.Errorf("%v: face surface %T is neither the ring's nor the ball's", op, f.Geometry())
			}
		}
		return query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	}
	union, cut, lens := vol(ops.Join), vol(ops.Cut), vol(ops.Intersect)
	ringVol := query.BodyGeometryProperties(ring, ops.DefaultQuality()).Volume
	ballVol := query.BodyGeometryProperties(ball, ops.DefaultQuality()).Volume
	if got, want := union+lens, ringVol+ballVol; stdmath.Abs(got-want) > 1e-6*want {
		t.Errorf("V(∪) + V(∩) = %.6f, want V(ring) + V(ball) = %.6f", got, want)
	}
	if got, want := cut+lens, ringVol; stdmath.Abs(got-want) > 1e-6*want {
		t.Errorf("V(−) + V(∩) = %.6f, want V(ring) = %.6f", got, want)
	}
	// The ring's own volume is 2π²Rr², which pins the identity to the geometry rather than to itself.
	if want := 2 * stdmath.Pi * stdmath.Pi * 5 * 1.5 * 1.5; stdmath.Abs(ringVol-want) > 1e-9*want {
		t.Errorf("the ring integrates to %.6f, want 2π²Rr² = %.6f", ringVol, want)
	}
}

// TestCoaxialShaftThroughARingIsExact: a cylinder COAXIAL with the ring has no azimuth dependence at
// all — its constraint on the torus is a function of the tube angle alone — so the section is whole
// circles rather than curves, and the cut is the ring with its bore opened out. It is the degenerate
// end of the same closed form, and the one a type-driven dispatch would have had to special-case.
func TestCoaxialShaftThroughARingIsExact(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	shaft, err := brep.SolidCylinder(math.P3(0, 0, -4), math.V3(0, 0, 1), 4, 8)
	if err != nil {
		t.Fatalf("shaft: %v", err)
	}
	res, err := ops.Boolean(ops.Cut, ring, shaft)
	if err != nil {
		t.Fatalf("ring − coaxial shaft: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
		t.Fatalf("not a valid closed manifold solid: %+v", r)
	}
	tori, cyls := 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Torus:
			tori++
		case geom.Cylinder:
			cyls++
		}
	}
	if tori != 1 || cyls != 1 || len(res.Faces()) != 2 {
		t.Errorf("got %d torus + %d cylinder of %d faces, want the ring's surface and the bore wall", tori, cyls, len(res.Faces()))
	}
	// The shaft takes the tube's material inside radius 4, which is the ring less two spherical-zone-like
	// caps; asserting it against the ring's own volume keeps the row honest without a second oracle.
	ringVol := query.BodyGeometryProperties(ring, ops.DefaultQuality()).Volume
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	if got >= ringVol || got < 0.8*ringVol {
		t.Errorf("the bored ring measures %.5f against the whole ring's %.5f; the shaft removes a modest bite", got, ringVol)
	}
}

// TestSkewRodThroughARingIsRefusedByName: a rod ACROSS the ring is not in the closed form's family. Its
// quadratic form is not invariant about the ring's axis, so the azimuth dependence is a second harmonic
// whose roots are a quartic rather than an arccos, and the section declines. The refusal must stay loud.
func TestSkewRodThroughARingIsRefusedByName(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	rod, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1, 9)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	if _, err := ops.Boolean(ops.Cut, ring, rod); err == nil {
		t.Fatal("a skew rod through a ring must be refused by name, not built")
	}
}

// TestAxialDrillThroughARingIsRefusedNotWrong is a NAMED GAP, and the row exists so it stays named.
//
// The section is exact — the drill's two seams come back as closed loops on the ring, each wrapping the
// drill's azimuth once — but the ruled chart's trim keeps only HALF the bore wall: it splits each seam
// at the two azimuths where the seam reaches its extreme height (ρ = R, where the tube is topmost) and
// emits one contractible patch bounded by two rulings instead of the two-rim band. The result is two
// shells and an open boundary, which the boolean's own acceptance gate refuses — so nothing wrong
// ships. The fix belongs to the wall trim, not to the section, and this row will flip from a refusal to
// a result when it lands.
func TestAxialDrillThroughARingIsRefusedNotWrong(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(5, 0, -4), math.V3(0, 0, 1), 0.8, 8)
	if err != nil {
		t.Fatalf("drill: %v", err)
	}
	body, err := ops.Boolean(ops.Cut, ring, drill)
	if err == nil {
		t.Fatalf("the axial drill built a body of %d faces; the wall trim does not close it yet, so it must be refused", len(body.Faces()))
	}
	if body != nil {
		t.Error("a refused boolean must return no body")
	}
}
