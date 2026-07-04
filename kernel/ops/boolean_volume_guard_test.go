// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// guardBlock builds an axis-aligned block body with an exact, easily reasoned volume.
func guardBlock(t *testing.T, min, max math.Point3, name string) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(min, max, name)
	if err != nil {
		t.Fatalf("SolidBlock(%v,%v): %v", min, max, err)
	}
	return b
}

// TestBooleanVolumeGuardTwoSided pins the Requicha brackets on the boolean volume guard (#1601):
// V(A∖B) ∈ [V(A)−V(B), V(A)] and V(A∪B) ∈ [max(V(A),V(B)), V(A)+V(B)]. The guard used to be
// one-sided — a cut that removed too much material, or a join that fabricated material, passed
// silently and shipped a materially wrong body.
func TestBooleanVolumeGuardTwoSided(t *testing.T) {
	target := guardBlock(t, math.P3(0, 0, 0), math.P3(2, 1, 1), "target") // V(A) = 2
	tool := guardBlock(t, math.P3(1.5, 0, 0), math.P3(2.5, 1, 1), "tool") // V(B) = 1, overlap 0.5

	cases := []struct {
		name    string
		op      PartFeatureOperation
		result  *topo.Body
		invalid bool
	}{
		{"cut within bracket", Cut, guardBlock(t, math.P3(0, 0, 0), math.P3(1.5, 1, 1), "ok"), false},   // 1.5 ∈ [1, 2]
		{"cut removed too much", Cut, guardBlock(t, math.P3(0, 0, 0), math.P3(0.5, 1, 1), "low"), true}, // 0.5 < V(A)−V(B) = 1
		{"cut removed nothing extra", Cut, guardBlock(t, math.P3(0, 0, 0), math.P3(2.5, 1, 1), "high"), true},
		{"join within bracket", Join, guardBlock(t, math.P3(0, 0, 0), math.P3(2.5, 1, 1), "ok"), false},         // 2.5 ∈ [2, 3]
		{"join fabricated material", Join, guardBlock(t, math.P3(0, 0, 0), math.P3(4, 1, 1), "high"), true},     // 4 > V(A)+V(B) = 3
		{"join lost material", Join, guardBlock(t, math.P3(0, 0, 0), math.P3(1.5, 1, 1), "low"), true},          // 1.5 < max = 2
		{"intersect within bound", Intersect, guardBlock(t, math.P3(1.5, 0, 0), math.P3(2, 1, 1), "ok"), false}, // 0.5 ≤ min = 1
		{"intersect too big", Intersect, guardBlock(t, math.P3(0, 0, 0), math.P3(1.5, 1, 1), "high"), true},     // 1.5 > min = 1
	}
	for _, c := range cases {
		if got := invalidBooleanVolume(c.op, target, tool, c.result); got != c.invalid {
			t.Errorf("%s: invalidBooleanVolume = %v, want %v", c.name, got, c.invalid)
		}
	}
}

// TestTangentUnionShipsExactCoordinates is #1600's headline guarantee (A4): an edge-tangent union
// (two boxes sharing only the vertical edge x=2,y=2) is resolved EXACTLY — the coincident dihedrals
// split by radial order — and shipped with ZERO displacement. Every output vertex is bit-identical
// to an operand vertex (distance 0 to float ulps, NOT within the retired 1e-5 nudge), the emitted
// diagnostic is the informational tangent-contact note (never the nudged-geometry Defect), and the
// result is an Euler-valid solid. Regression against the old geometry-moving retry.
func TestTangentUnionShipsExactCoordinates(t *testing.T) {
	a := guardBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2), "a")
	b := guardBlock(t, math.P3(2, 2, 0), math.P3(4, 4, 2), "b") // shares only the vertical edge x=2,y=2
	rec := &diag.Recorder{}
	res, err := BooleanWithDiagnostics(Join, a, b, rec)
	if err != nil {
		t.Fatalf("tangent union: %v", err)
	}
	if res == nil || !res.IsSolid() {
		t.Fatal("tangent union did not produce a solid")
	}
	if rec.Has(brep.CodeBooleanTangentUnresolved) {
		t.Errorf("tangent union declined to CSG instead of shipping the exact manifold; recs=%v", rec.Records())
	}
	if !rec.Has(brep.CodeBooleanTangentContact) {
		t.Errorf("tangent union did not record the exact tangent-contact diagnostic; got %v", rec.Records())
	}
	assertVerticesExactlyOnOperands(t, res, a, b)
	if r := Validate(res); !r.Valid {
		t.Fatalf("exact tangent union is not a valid solid: %v", r.Issues)
	}
}

// assertVerticesExactlyOnOperands fails if any vertex of the result is not bit-identical to a
// vertex of one of the operands — the "distance 0 within float ulps, not within 1e-5" contract of
// #1600. A pure edge-tangent union creates no new intersection vertices, so every result vertex
// must coincide EXACTLY with an operand vertex; a 1e-5 nudge would move all of operand B's.
func assertVerticesExactlyOnOperands(t *testing.T, res, a, b *topo.Body) {
	t.Helper()
	exact := map[math.Point3]bool{}
	for _, src := range []*topo.Body{a, b} {
		for _, v := range src.Vertices() {
			exact[v.Point()] = true
		}
	}
	for _, v := range res.Vertices() {
		if !exact[v.Point()] {
			t.Fatalf("result vertex %v is not bit-identical to any operand vertex (displaced geometry)", v.Point())
		}
	}
}
