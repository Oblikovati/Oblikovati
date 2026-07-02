// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestConstraintKindSelfDescription spot-checks the KindedConstraint capability
// (#1625, audit I2): the derived axis/plane kinds, and the RelatedEntities
// order the API enumerates (which consumers previously re-derived by switch).
func TestConstraintKindSelfDescription(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(2, 0))
	l2 := s.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(2, 1))
	c1 := s.Circles().AddByCenterRadius(math.P2(5, 0), 1)
	g := s.GeometricConstraints()

	sym := g.AddSymmetry(l1.A, l1.B, l2)
	if sym.ConstraintKind() != SymmetryKind {
		t.Errorf("symmetry kind = %q, want %q", sym.ConstraintKind(), SymmetryKind)
	}
	assertRelated(t, "symmetry", sym, l1.A.EntityID(), l1.B.EntityID(), l2.EntityID())

	tan := g.AddTangent(l1, c1)
	assertRelated(t, "tangent", tan, l1.EntityID(), c1.EntityID())
	if tan.ConstraintKind() != TangentKind {
		t.Errorf("tangent kind = %q, want %q", tan.ConstraintKind(), TangentKind)
	}

	grd := g.AddGroundPoints(l1.A, l2.B)
	assertRelated(t, "ground", grd, l1.A.EntityID(), l2.B.EntityID())
}

// TestConstraintKind3DDerivedKinds pins the one-type-many-kinds constraints:
// parallel-to-axis/plane derive their persisted kind from the constrained
// direction (the pre-#1625 axisRowKind/planeRowKind spellings).
func TestConstraintKind3DDerivedKinds(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(math.P3(0, 0, 0), math.P3(1, 1, 1))

	axisKinds := map[ConstraintKind]*ParallelToAxis3D{
		ParallelToXAxisKind: NewParallelToXAxis3D(l),
		ParallelToYAxisKind: NewParallelToYAxis3D(l),
		ParallelToZAxisKind: NewParallelToZAxis3D(l),
	}
	for want, c := range axisKinds {
		if c.ConstraintKind() != want {
			t.Errorf("axis kind = %q, want %q", c.ConstraintKind(), want)
		}
	}
	planeKinds := map[ConstraintKind]*ParallelToPlane3D{
		ParallelToXYKind: NewParallelToXYPlane3D(l),
		ParallelToXZKind: NewParallelToXZPlane3D(l),
		ParallelToYZKind: NewParallelToYZPlane3D(l),
	}
	for want, c := range planeKinds {
		if c.ConstraintKind() != want {
			t.Errorf("plane kind = %q, want %q", c.ConstraintKind(), want)
		}
	}
}

// TestEqual3DKeepsOperandEntities pins the #1625 Equal3D fix: the constraint
// names its radius-bearing operands, so it can enumerate refs and serialize
// (an Equal3D save failed at runtime before).
func TestEqual3DKeepsOperandEntities(t *testing.T) {
	s := NewSketches3D().Add()
	zUp, _ := math.NewUnitVector3(0, 0, 1)
	a := s.AddCircle3D(math.P3(0, 0, 0), zUp, 2)
	b := s.AddCircle3D(math.P3(5, 0, 0), zUp, 2)
	eq := mustEqual3D(t, a, b)
	if eq.ConstraintKind() != EqualKind {
		t.Errorf("equal kind = %q, want %q", eq.ConstraintKind(), EqualKind)
	}
	assertRelated(t, "equal", eq, a.EntityID(), b.EntityID())
}

// assertRelated asserts a constraint's RelatedEntities ids in order.
func assertRelated(t *testing.T, name string, c KindedConstraint, want ...ID) {
	t.Helper()
	got := c.RelatedEntities()
	if len(got) != len(want) {
		t.Fatalf("%s related = %d entities, want %d", name, len(got), len(want))
	}
	for i, e := range got {
		if e.EntityID() != want[i] {
			t.Errorf("%s related[%d] = id %d, want %d", name, i, e.EntityID(), want[i])
		}
	}
}
