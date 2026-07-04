// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Certification for the filled-wedge radial-edge sew resolving line/point tangencies EXACTLY
// (ADR-0047, #1726). The old sew fused a non-coplanar bowtie contact into an edge-manifold-looking
// body with odd χ and fell back to a geometry-moving nudge (0.1 µm). The Weiler radial pairing
// (pair the two boundaries of each filled dihedral wedge) resolves it with ZERO displacement:
// a pure line kiss becomes two coincident shells, a bowtie on an otherwise-connected body a single
// valid shell — never the nudge. The volume oracle is analytic (a zero-measure contact carries no
// volume, so a pure union is exactly additive) — no external kernel needed.

// assertExactContact fails unless the contact resolved to a valid manifold with zero displacement:
// the recorder must carry the informational tangent-contact note and never the unresolved Defect
// (which would mean the sew declined to CSG).
func assertExactContact(t *testing.T, rec *diag.Recorder, label string) {
	t.Helper()
	if rec.Has(brep.CodeBooleanTangentUnresolved) {
		t.Fatalf("%s: tangent contact did not resolve to a valid manifold (declined to CSG); recs=%v", label, rec.Records())
	}
	if !rec.Has(brep.CodeBooleanTangentContact) {
		t.Fatalf("%s: did not record the exact tangent-contact diagnostic; got %v", label, rec.Records())
	}
}

// assertUniqueEdgeKeys fails if two edges share an ADR-0043 reference key. The radial-edge sew mints
// coincident duplicate edges at a contact; this pins that they still name distinctly (each borders a
// distinct face pair), so downstream feature references keyed on brep:edge#N stay unambiguous.
func assertUniqueEdgeKeys(t *testing.T, res *topo.Body, label string) {
	t.Helper()
	seen := map[string]bool{}
	for _, e := range res.Edges() {
		k := string(e.ReferenceKey())
		if seen[k] {
			t.Fatalf("%s: duplicate edge reference key %x — coincident contact edges collided", label, k)
		}
		seen[k] = true
	}
}

// assertVertsSatisfyOperandSurfaces fails unless every result vertex lies on at least one operand
// face's plane to a tight tolerance — the #1726 acceptance guarantee that output coordinates satisfy
// the operand surface equations exactly (the nudge displaced operand B by 0.1 µm; 1e-9 cm catches
// that decisively while tolerating only float noise). Proves the sew moved nothing.
func assertVertsSatisfyOperandSurfaces(t *testing.T, res *topo.Body, tol float64, operands ...*topo.Body) {
	t.Helper()
	var planes []geom.Plane
	for _, o := range operands {
		for _, f := range o.Faces() {
			if pl, ok := f.Geometry().(geom.Plane); ok {
				planes = append(planes, pl)
			}
		}
	}
	for _, v := range res.Vertices() {
		p := v.Point()
		best := 1e300
		for _, pl := range planes {
			if d := absF(float64(pl.Origin.VectorTo(p).Dot(pl.Normal()))); d < best {
				best = d
			}
		}
		if best > tol {
			t.Fatalf("result vertex %v lies %.2e off every operand face plane (> %.0e) — displaced geometry", p, best, tol)
		}
	}
}

// assertVertsOnOperandCorners fails unless every result vertex is bit-identical to an operand vertex
// — the exact-coordinate guarantee for a contact that creates no new intersection vertices.
func assertVertsOnOperandCorners(t *testing.T, res *topo.Body, operands ...*topo.Body) {
	t.Helper()
	corner := map[math.Point3]bool{}
	for _, o := range operands {
		for _, v := range o.Vertices() {
			corner[v.Point()] = true
		}
	}
	for _, v := range res.Vertices() {
		if !corner[v.Point()] {
			t.Fatalf("result vertex %v is not bit-identical to any operand vertex (displaced geometry)", v.Point())
		}
	}
}

// TestBowtiePureLineKissShipsTwoShellsExact is the #1726 headline on the cleanest case: two unit
// cubes meeting ONLY along one vertical edge (disjoint interiors) union into TWO coincident manifold
// shells (χ = 4), with exactly additive volume V(A)+V(B) = 2 and every coordinate bit-identical to
// an operand corner. The old sew welded the bowtie into a χ-odd pinch and nudged.
func TestBowtiePureLineKissShipsTwoShellsExact(t *testing.T) {
	a := box(0, 0, 0, 1, 1, 1)
	b := box(1, 1, 0, 1, 1, 1) // touches `a` only along the edge x=1,y=1,z∈[0,1]
	rec := &diag.Recorder{}
	u, err := brep.BooleanDiag(brep.Union, a, b, rec)
	if err != nil {
		t.Fatalf("union: %v", err)
	}
	r := ops.Validate(u)
	if !r.Valid {
		t.Fatalf("pure line-kiss union invalid: χ=%d issues=%v", r.EulerCharacteristic, r.Issues)
	}
	if r.EulerCharacteristic != 4 {
		t.Errorf("χ = %d, want 4 (two sphere-topology shells kissing along the line)", r.EulerCharacteristic)
	}
	if got := vol(u); absF(got-2.0) > 1e-12 {
		t.Errorf("volume = %.15g, want 2.0 exactly (zero-measure contact is additive)", got)
	}
	assertExactContact(t, rec, "pure line-kiss")
	assertVertsOnOperandCorners(t, u, a, b)
	assertUniqueEdgeKeys(t, u, "pure line-kiss")
}

// TestBowtieBridgedCornerStaysManifoldExact certifies a bowtie that is NOT a disconnection: two
// kitty-corner blocks both fused to a base plate touch along a vertical line, but the base bridges
// them, so the exact result is ONE valid shell (χ = 2) — shipped without a nudge.
func TestBowtieBridgedCornerStaysManifoldExact(t *testing.T) {
	base := box(-1, -1, 0, 2, 2, 0.2)
	a := box(-1, -1, 0, 1, 1, 1)
	b := box(0, 0, 0, 1, 1, 1)
	rec := &diag.Recorder{}
	body, _ := brep.BooleanDiag(brep.Union, base, a, rec)
	body, err := brep.BooleanDiag(brep.Union, body, b, rec)
	if err != nil {
		t.Fatalf("union: %v", err)
	}
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("bridged-corner union invalid: χ=%d issues=%v", r.EulerCharacteristic, r.Issues)
	}
	assertExactContact(t, rec, "bridged-corner")
}

// TestBowtieBossLugShipsExact is the exact #1726 residual: the faceted-cylinder boss whose
// longitudinal edge grazes the lug's front-face plane. It now ships EXACT (no nudge), validates as a
// closed manifold, and its volume matches the inclusion–exclusion identity
// V(A∪B) = V(A) + V(B) − V(A∩B) computed by the kernel's own booleans — proving the resolution
// neither creates nor loses material.
func TestBowtieBossLugShipsExact(t *testing.T) {
	can := cylinderZAt(0, -0.8, 0, 1.9, 1.4, "can")
	boss := cylinderZAt(0, 0, -0.15, 0, 0.45, "boss")
	body, err := brep.Boolean(brep.Union, can, boss)
	if err != nil {
		t.Fatalf("can∪boss: %v", err)
	}
	lug := box(-2.1, -1.15, -0.085, 4.2, -0.45-(-1.15), 0.17) // front face on the boss edge y=-0.45

	rec := &diag.Recorder{}
	got, err := brep.BooleanDiag(brep.Union, body, lug, rec)
	if err != nil {
		t.Fatalf("boss-lug union: %v", err)
	}
	if r := ops.Validate(got); !r.Valid {
		t.Fatalf("boss-lug union invalid: χ=%d issues=%v", r.EulerCharacteristic, r.Issues)
	}
	assertExactContact(t, rec, "boss-lug")

	inter, err := brep.Boolean(brep.Intersection, body, lug)
	if err != nil {
		t.Fatalf("boss-lug intersection: %v", err)
	}
	want := vol(body) + vol(lug) - vol(inter)
	if diff := absF(vol(got) - want); diff > 1e-9 {
		t.Errorf("boss-lug volume = %.12g, inclusion-exclusion = %.12g, diff %.2e (material created/lost)",
			vol(got), want, diff)
	}
	assertUniqueEdgeKeys(t, got, "boss-lug")
	assertVertsSatisfyOperandSurfaces(t, got, 1e-9, body, lug)
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
