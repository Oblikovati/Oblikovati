// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The per-face boolean certificate applies the Requicha membership rule to a point on a result face:
// a face of A∪B lies on ∂A outside B or on ∂B outside A; a face of A∖B lies on ∂A outside B or on ∂B
// INSIDE A; a face of A∩B lies on ∂A inside B or on ∂B inside A. These pin that rule directly, on two
// overlapping boxes where every answer is decidable by inspection — the corpus exercises the rule
// through whole booleans, but only for the configurations it happens to contain (M48/C3).
//
// target [0,10]^3 and tool [5,15]^3 overlap in the corner cube [5,10]^3.

func certifyTestOperands() (target, tool *boundaryIndex) {
	lo, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "target")
	hi, _ := brep.SolidBlock(math.P3(5, 5, 5), math.P3(15, 15, 15), "tool")
	return newBoundaryIndex(lo), newBoundaryIndex(hi)
}

func TestMembershipRuleOnEachOperandBoundary(t *testing.T) {
	t.Parallel()
	target, tool := certifyTestOperands()
	const tol = 1e-7 // tol:numeric — the probe's on-boundary reach, well under the 5-unit overlap

	cases := []struct {
		name string
		p    math.Point3
		op   PartFeatureOperation
		want bool
	}{
		// On the TARGET's boundary alone, at a corner the tool does not reach.
		{"join keeps the target wall outside the tool", math.P3(0, 5, 5), Join, true},
		{"cut keeps the target wall outside the tool", math.P3(0, 5, 5), Cut, true},
		{"intersect drops the target wall outside the tool", math.P3(0, 5, 5), Intersect, false},

		// On the TARGET's top face, at a spot the tool contains.
		{"join drops the target wall inside the tool", math.P3(7, 7, 10), Join, false},
		{"cut drops the target wall inside the tool", math.P3(7, 7, 10), Cut, false},
		{"intersect keeps the target wall inside the tool", math.P3(7, 7, 10), Intersect, true},

		// On the TOOL's boundary alone, outside the target.
		{"join keeps the tool wall outside the target", math.P3(15, 10, 10), Join, true},
		{"cut drops the tool wall outside the target", math.P3(15, 10, 10), Cut, false},
		{"intersect drops the tool wall outside the target", math.P3(15, 10, 10), Intersect, false},

		// On the TOOL's near wall, inside the target: this is the carved wall a cut must keep.
		{"join drops the tool wall inside the target", math.P3(5, 7, 7), Join, false},
		{"cut keeps the carved tool wall inside the target", math.P3(5, 7, 7), Cut, true},
		{"intersect keeps the tool wall inside the target", math.P3(5, 7, 7), Intersect, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pointKeptBy(c.op, target, tool, c.p, tol); got != c.want {
				t.Errorf("pointKeptBy(%v, %v) = %v, want %v", c.op, c.p, got, c.want)
			}
		})
	}
}

// TestAFabricatedFaceIsRefusedByEveryOperation is the certificate's reason to exist: a result face
// lying on NEITHER operand was invented by the operation, and no membership rule can justify it.
func TestAFabricatedFaceIsRefusedByEveryOperation(t *testing.T) {
	t.Parallel()
	target, tool := certifyTestOperands()
	away := math.P3(40, 40, 40)
	for _, op := range []PartFeatureOperation{Join, Cut, Intersect} {
		if pointKeptBy(op, target, tool, away, 1e-7) {
			t.Errorf("%v accepted a point on neither operand's boundary; such a face was fabricated", op)
		}
	}
}

// TestACoincidentContactSurvivesEveryOperation: a point on BOTH boundaries is a coincident-face
// contact, where the shared wall survives once. The rule accepts it before the per-op split, and this
// pins that, because deciding it per-op would drop the shared wall from one side or keep it twice.
func TestACoincidentContactSurvivesEveryOperation(t *testing.T) {
	t.Parallel()
	target, tool := certifyTestOperands()
	shared := math.P3(5, 7, 10) // on the target's top face AND on the tool's x=5 wall
	for _, op := range []PartFeatureOperation{Join, Cut, Intersect} {
		if !pointKeptBy(op, target, tool, shared, 1e-7) {
			t.Errorf("%v refused a point on both boundaries; a coincident contact survives once, not never", op)
		}
	}
}

// TestAnOperationWithNoMembershipRuleCertifiesEverything: the rule is stated for join, cut and
// intersect. Anything else has nothing to check, and must not be refused by a rule that does not
// apply to it.
func TestAnOperationWithNoMembershipRuleCertifiesEverything(t *testing.T) {
	t.Parallel()
	target, tool := certifyTestOperands()
	if !pointKeptBy(NewBody, target, tool, math.P3(40, 40, 40), 1e-7) {
		t.Error("an operation outside the membership rule must certify, not refuse")
	}
}

// TestBoundaryIndexKeepsFacesThatHaveNoRangeBox pins the trap that made the certificate reject every
// coaxial sphere boolean in the corpus: a BOUNDARY-LESS face (a whole sphere has no vertices) has an
// empty range box, so a tree query silently never returns it. Those faces are held aside and always
// tested.
func TestBoundaryIndexKeepsFacesThatHaveNoRangeBox(t *testing.T) {
	t.Parallel()
	ball := wholeSphereBody(t, 4)
	bi := newBoundaryIndex(ball)
	if len(bi.faces)+len(bi.unboxed) != len(ball.Faces()) {
		t.Fatalf("index holds %d boxed + %d unboxed faces, but the body has %d: a face was dropped",
			len(bi.faces), len(bi.unboxed), len(ball.Faces()))
	}
	if !bi.on(math.P3(0, 0, 4), 1e-6) {
		t.Error("the sphere's north pole reads as off its own boundary; a face with no range box was lost to the tree query")
	}
}

// wholeSphereBody is a closed ball as ONE boundary-less face: no vertices, so no range box, which is
// the configuration TestBoundaryIndexKeepsFacesThatHaveNoRangeBox exists to cover.
func wholeSphereBody(t *testing.T, radius float64) *topo.Body {
	t.Helper()
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), radius)
	if err != nil {
		t.Fatalf("NewSphere(R=%g): %v", radius, err)
	}
	lin := topo.NewLineage(topo.Tok("certifyball", "body", 0))
	bld := topo.NewBuilder(true, lin)
	bld.AddFace(sphere, topo.NewLineage(topo.Tok("certifyball", "face", 0)))
	return bld.Build()
}

// TestAResultShowingMoreBoundaryThanItsOperandsIsRefused is the per-face AREA gate the ground rules
// name ("result gates are per-face — area, surface type, loop count — against the oracle"), and its
// proof that it can fire. The oracle is the operands: every face of A op B lies on ∂A or ∂B and a
// boolean only trims them, so the result cannot show more boundary than the two have between them.
//
// The probe hands the certificate a real result with a tool too small to account for it: the exact
// 0.8 bore through the RING, certified against a drill of the same radius but half a unit long
// instead of 8. Membership still passes — the bore wall's interior point does lie on that stub, and
// the operation keeps it there — so this row exercises the AREA gate and nothing else. The result
// then shows 305.850 of boundary where ring + stub hold 302.622 between them, 1.07e-2 over, against
// a slack of 1e-6 and a corpus whose largest accepted ratio is 1 + 1e-15.
func TestAResultShowingMoreBoundaryThanItsOperandsIsRefused(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 0.8)
	body, err := Boolean(Cut, ring, drill)
	if err != nil {
		t.Fatalf("the RD- row must build: %v", err)
	}
	stub, err := brep.SolidCylinder(math.P3(5, 0, -0.25), math.V3(0, 0, 1), 0.8, 0.5)
	if err != nil {
		t.Fatalf("stub: %v", err)
	}
	if ev := certifyBooleanFaces(Cut, ring, drill, body); !ev.kept {
		t.Fatalf("the genuine operands must certify; the probe below is meaningless otherwise")
	} else if _, over := ev.overclaimsItsOperands(ring, drill); over {
		t.Fatalf("the genuine pair overclaims: %g of boundary", ev.claimed)
	}
	ev := certifyBooleanFaces(Cut, ring, stub, body)
	if !ev.kept {
		t.Fatalf("membership must still pass against the stub, or this row is not measuring the area gate")
	}
	available, over := ev.overclaimsItsOperands(ring, stub)
	if !over {
		t.Errorf("a result showing %g of boundary certified against operands holding %g between them",
			ev.claimed, available)
	}
}

// TestTheCertificateLeavesExactlyTheKnownFacesUnprobed is the RATCHET on probe coverage, at the level
// the certificate itself works: how many faces of a built result it could not classify.
//
// It exists because the count had been prose in two comments and nothing held it. #3516 round 0 took
// the whole package from 26 unprobed faces to 4; round 1 gave 22 of them back in a comment-level
// change and shipped three statements that had become false. A number stated in a docstring is not a
// ratchet.
//
// The pairs are the ones that OWN the residue, identified by instrumenting the package: the bored
// ring (this issue's own family, 0 unprobed) and the two-oval torus−box cut, whose 2-loop torus face
// is one of the four the package still cannot probe. Their totals are pinned, so a coverage loss on
// either fails here instead of reading oddly in a comment. A RISE is a regression; a FALL means the
// probe improved and the pin should come down with the change that earned it.
func TestTheCertificateLeavesExactlyTheKnownFacesUnprobed(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 0.8)
	for _, tc := range []struct {
		name         string
		op           PartFeatureOperation
		target, tool *topo.Body
		wantUnprobed int
	}{
		{"a bored ring", Cut, ring, drill, 0},
		{"a two-oval torus − box", Cut, ratchetTorus(t), ratchetBlock(t), 1},
		{"a two-oval torus ∩ box", Intersect, ratchetTorus(t), ratchetBlock(t), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := Boolean(tc.op, tc.target, tc.tool)
			if err != nil {
				t.Fatalf("the pair must build for its coverage to be measured: %v", err)
			}
			ev := certifyBooleanFaces(tc.op, tc.target, tc.tool, body)
			if ev.unprobed != tc.wantUnprobed {
				t.Errorf("%d of %d faces unprobed, want %d — a rise is lost coverage, a fall is an "+
					"improvement whose pin should come down with it", ev.unprobed, len(body.Faces()), tc.wantUnprobed)
			}
		})
	}
}

// ratchetTorus and ratchetBlock are the two-oval torus−box pair (boolean_csg_demotion_test.go's
// torusTwoOvalBox), rebuilt here so the ratchet above owns its own operands.
func ratchetTorus(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "torus")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	return b
}

func ratchetBlock(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(math.P3(-20, 2, -20), math.P3(20, 20, 20), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}
