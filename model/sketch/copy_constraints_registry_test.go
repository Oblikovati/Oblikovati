// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"strings"
	"testing"

	"oblikovati.org/math"
)

// TestConstraintCarrierRegistryMatchesVocabulary: every persisted 2D constraint
// kind has a copy decision — a carry function or a documented skip — so a new
// kind cannot ship as a silent copy drop (#1637; mirrors the codec closure test).
func TestConstraintCarrierRegistryMatchesVocabulary(t *testing.T) {
	assertConstraintKindSetsEqual(t, "2D carry", registeredCarrierKinds2D(), persistedConstraintVocabulary2D)
}

// TestConstraintCarrierDecisionsAreExclusive: a carrier is either a carry
// function or a documented skip, never both and never neither (#1637).
func TestConstraintCarrierDecisionsAreExclusive(t *testing.T) {
	for kind, cc := range constraintCarriers2D {
		if (cc.carry == nil) == (cc.skipReason == "") {
			t.Errorf("carrier %q must have exactly one of carry or skipReason (carry=%v, skipReason=%q)",
				kind, cc.carry != nil, cc.skipReason)
		}
	}
}

// TestCopySmoothSketchPreservesDOF: a whole-sketch copy has the same degrees of
// freedom as the source. The fixture leans on Smooth, Ground and Offset — the
// DOF-removing kinds the pre-#1637 switch dropped, which made the copy solve to
// a different shape on the next edit (G2 regression for the Smooth spline pair).
func TestCopySmoothSketchPreservesDOF(t *testing.T) {
	src := NewSketches().Add(XYPlane())
	g := src.GeometricConstraints()
	sp1 := src.Splines().AddByPoints([]math.Point2{math.P2(0, 0), math.P2(1, 1), math.P2(2, 0)}, false)
	sp2 := src.Splines().AddByPoints([]math.Point2{math.P2(2, 0), math.P2(3, -1), math.P2(4, 0)}, false)
	j1, j2, ok := NearestSmoothJoin(sp1, sp2)
	if !ok {
		t.Fatal("no smooth join between the two splines")
	}
	g.AddSmooth(sp1, sp2, j1, j2)
	l1 := src.Lines().AddByTwoPoints(math.P2(0, 5), math.P2(4, 5))
	l2 := src.Lines().AddByTwoPoints(math.P2(0, 6), math.P2(4, 6))
	g.AddOffset(l1, l2, 1)
	g.AddGround(l1)

	dst := NewSketches().Add(XYPlane())
	if _, warns := dst.CopyEntitiesWithConstraints(src, src.Entities(), math.V2(10, 0)); len(warns) != 0 {
		t.Fatalf("copy warnings = %q, want none", warns)
	}
	if got, want := dst.DegreesOfFreedom(), src.DegreesOfFreedom(); got != want {
		t.Errorf("copied sketch DOF = %d, want %d (source)", got, want)
	}
	if got, want := dst.GeometricConstraints().Count(), src.GeometricConstraints().Count(); got != want {
		t.Errorf("carried constraints = %d, want %d", got, want)
	}
}

// TestCopyWarnsOnTextBoxAnchorConstraint: the text-box anchor is the one
// documented skip — copying a text box emits a warning instead of a silent
// drop, and copying geometry that leaves the text box behind stays silent
// (external-operand rule) (#1637).
func TestCopyWarnsOnTextBoxAnchorConstraint(t *testing.T) {
	src := NewSketches().Add(XYPlane())
	src.TextBoxes().Add(math.P2(1, 1), "NOTE", 0.5, 0, TextLeft)
	l := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))

	whole := NewSketches().Add(XYPlane())
	_, warns := whole.CopyEntitiesWithConstraints(src, src.Entities(), math.V2(10, 0))
	if len(warns) != 1 || !strings.Contains(warns[0], "textBox") {
		t.Errorf("whole-sketch copy warnings = %q, want one textBox anchor skip", warns)
	}
	if got := whole.GeometricConstraints().Count(); got != 0 {
		t.Errorf("carried constraints = %d, want 0 (the anchor cannot travel)", got)
	}

	lineOnly := NewSketches().Add(XYPlane())
	if _, warns := lineOnly.CopyEntitiesWithConstraints(src, []Entity{l}, math.V2(10, 0)); len(warns) != 0 {
		t.Errorf("line-only copy warnings = %q, want none (text box was left behind)", warns)
	}
}

// TestCopyLivePatternLinkDegradesToFrozen: a live pattern link (its offset
// closure reads the source pattern's parameters) is carried as a frozen link,
// matching the persistence codec's degradation on save (#1637).
func TestCopyLivePatternLinkDegradesToFrozen(t *testing.T) {
	src := NewSketches().Add(XYPlane())
	seed := src.Points().Add(math.P2(0, 0))
	member := src.Points().Add(math.P2(3, 0))
	src.GeometricConstraints().AddPatternLinkLive(seed, member, func() (float64, float64) { return 3, 0 })

	dst := NewSketches().Add(XYPlane())
	if _, warns := dst.CopyEntitiesWithConstraints(src, src.Entities(), math.V2(10, 0)); len(warns) != 0 {
		t.Fatalf("copy warnings = %q, want none", warns)
	}
	if got := dst.GeometricConstraints().Count(); got != 1 {
		t.Fatalf("carried constraints = %d, want 1 (frozen pattern link)", got)
	}
	link, ok := dst.GeometricConstraints().Item(0).(*PatternConstraint)
	if !ok {
		t.Fatalf("carried constraint is %T, want *PatternConstraint", dst.GeometricConstraints().Item(0))
	}
	if link.live != nil {
		t.Error("copied pattern link kept the source's live offset closure; want frozen")
	}
	if res := dst.Solve(); !res.Converged {
		t.Errorf("copied sketch did not solve: %+v", res)
	}
}
