// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestSingleLineHorizontalSolves: a single-line horizontal constraint makes the whole line
// horizontal (endpoints share Y), removing one DOF, and relates the line itself (#1871).
func TestSingleLineHorizontalSolves(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 1))
	c := g.AddLineHorizontal(l)
	if satisfied(c) {
		t.Error("horizontal satisfied for a slanted line")
	}
	g.AddFix(l.A)
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if dy := float64(l.B.Position().Y - l.A.Position().Y); dy > 1e-9 || dy < -1e-9 {
		t.Errorf("after horizontal solve Δy = %v, want 0", dy)
	}
	if ents := c.RelatedEntities(); len(ents) != 1 || ents[0] != l {
		t.Errorf("related entities = %v, want the line itself", ents)
	}
}

// TestSingleLineVerticalSolves: the vertical twin — endpoints share X (#1871).
func TestSingleLineVerticalSolves(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 4))
	g.AddLineVertical(l)
	g.AddFix(l.A)
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if dx := float64(l.B.Position().X - l.A.Position().X); dx > 1e-9 || dx < -1e-9 {
		t.Errorf("after vertical solve Δx = %v, want 0", dx)
	}
}

// TestSingleLineHorizontalDistinctFromAlign: the single-line and two-point forms are distinct
// constraint types, so enumeration and the exporter can tell them apart (#1871).
func TestSingleLineHorizontalDistinctFromAlign(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	single := g.AddLineHorizontal(l)
	align := g.AddHorizontal(l.A, l.B)
	if single.ConstraintKind() == align.ConstraintKind() {
		t.Errorf("single-line and align share kind %q — they must be distinct", single.ConstraintKind())
	}
	if single.ConstraintKind() != SingleLineHorizontalKind || align.ConstraintKind() != HorizontalKind {
		t.Errorf("kinds = %q / %q, want %q / %q",
			single.ConstraintKind(), align.ConstraintKind(), SingleLineHorizontalKind, HorizontalKind)
	}
}
