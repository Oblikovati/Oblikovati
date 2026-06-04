// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/math"
)

func TestUnitOfZeroVectorIsZero(t *testing.T) {
	if u := unit(math.V3(0, 0, 0)); u.Length() != 0 {
		t.Fatalf("unit(0) = %v, want zero", u)
	}
}

func TestProjectOntoDegenerateDirection(t *testing.T) {
	if got := projectOnto(math.V3(1, 1, 1), math.V3(0, 0, 0)); got != 0 {
		t.Fatalf("projectOnto onto zero = %v, want 0", got)
	}
}

func TestClampDomainReplacesInfiniteBounds(t *testing.T) {
	lo, hi := clampDomain(stdmath.Inf(-1), stdmath.Inf(1))
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		t.Fatalf("clampDomain left an infinite bound: %v..%v", lo, hi)
	}
}

func TestClampBounds(t *testing.T) {
	if clamp(-1, 0, 1) != 0 || clamp(2, 0, 1) != 1 || clamp(0.5, 0, 1) != 0.5 {
		t.Fatal("clamp did not bound correctly")
	}
}

func TestNewellAreaDegenerate(t *testing.T) {
	if a := newellArea([]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)}); a != 0 {
		t.Fatalf("newellArea of < 3 points = %v, want 0", a)
	}
}

func TestLineageKeyJoinsMultipleTokens(t *testing.T) {
	key := string(NewLineage(Tok("f", "body", 0), Tok("g", "face", 2)).Key())
	if key != "f:body#0/g:face#2" {
		t.Fatalf("multi-token key = %q", key)
	}
}

func TestSurfaceBodiesRemoveNonMember(t *testing.T) {
	c := NewSurfaceBodies()
	member := c.Add(NewBuilder(true, NewLineage(Tok("a", "body", 0))).Build())
	other := NewBuilder(true, NewLineage(Tok("b", "body", 0))).Build()
	if c.Remove(other) {
		t.Error("Remove reported success for a non-member body")
	}
	if !c.Remove(member) {
		t.Error("Remove failed for a real member")
	}
}

func TestContainmentOnYNormalFaceAndCylinder(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	mk := func(p math.Point3, i int) *Vertex { return bld.AddVertex(p, NewLineage(Tok("f", "vertex", i))) }
	seg := func(p, q *Vertex, i int) *Edge {
		return bld.AddEdge(geom.NewLineSegment(p.Point(), q.Point()), p, q, NewLineage(Tok("f", "edge", i)))
	}
	// A triangle on the y=0 plane (normal +Y) exercises dropAxis's Y-dominant branch.
	a, b, c := mk(math.P3(0, 0, 0), 0), mk(math.P3(2, 0, 0), 1), mk(math.P3(0, 0, 2), 2)
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 1, 0))
	yf := bld.AddFace(plane, NewLineage(Tok("f", "face", 0)),
		OuterLoop(Fwd(seg(a, b, 0)), Fwd(seg(b, c, 1)), Rev(seg(a, c, 2))))
	if !NewFaceEvaluator(yf).Contains(math.P3(0.4, 0, 0.4)) {
		t.Error("interior point on a y=0 face not contained")
	}

	// A cylindrical (non-plane) face: Contains short-circuits to false.
	cyl, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	v := bld.AddVertex(math.P3(1, 0, 0), NewLineage(Tok("f", "vertex", 9)))
	e := bld.AddEdge(geom.NewLineSegment(math.P3(1, 0, 0), math.P3(1, 0, 1)), v, v, NewLineage(Tok("f", "edge", 9)))
	cf := bld.AddFace(cyl, NewLineage(Tok("f", "face", 1)), OuterLoop(Fwd(e)))
	if NewFaceEvaluator(cf).Contains(math.P3(1, 0, 0.5)) {
		t.Error("Contains on a non-plane face should be false")
	}
}
