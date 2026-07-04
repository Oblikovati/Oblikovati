// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// notchedTarget builds a cylinder already notched by a first cut (an oblique plane clipping the top rim), the
// target for a partial-rim second cut through the full boolean dispatch (#1732).
func notchedTarget(t *testing.T) *topo.Body {
	t.Helper()
	bare, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("bare cylinder: %v", err)
	}
	pl, _ := geom.NewPlane(math.P3(1.5, 0, 8), math.V3(1, 0, 1))
	target, err := brep.HalfSpaceCut(bare, pl)
	if err != nil {
		t.Fatalf("first cut (notch): %v", err)
	}
	return target
}

// TestPartialRimSecondCutTakesAnalyticPath drives the full boolean dispatch (BooleanWithDiagnostics), not the
// brep entry directly: a rod drilled through the still-full lower band of an already-notched cylinder, disjoint
// from the notch, must be routed to the exact analytic partial-rim path — NOT the faceted CSG fallback — and
// yield a valid genus-1 solid. This is the user-facing proof that curvedPartialRimCut is wired into the cascade.
func TestPartialRimSecondCutTakesAnalyticPath(t *testing.T) {
	rod, err := brep.SolidCylinder(math.P3(-6, 0, 3), math.V3(1, 0, 0), 1, 12)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	rec := &diag.Recorder{}
	res, err := BooleanWithDiagnostics(Cut, notchedTarget(t), rod, rec)
	if err != nil {
		t.Fatalf("partial-rim cut: %v", err)
	}
	if rec.Has(CodeBooleanCSGFallback) {
		t.Errorf("disjoint partial-rim cut fell back to CSG; want the exact analytic path. recs=%v", rec.Records())
	}
	if r := Validate(res); !r.Valid {
		t.Fatalf("partial-rim result is not a valid solid: %v", r.Issues)
	}
	if chi := res.EulerCharacteristic(); chi != 0 {
		t.Errorf("partial-rim result chi=%d; want 0 (genus-1 through-tunnel)", chi)
	}
}

// TestPartialRimCornerJunctionTakesAnalyticPath: a second cut whose imprint CROSSES the removed notch — the
// corner-junction (#1738, ADR-0048) — is now built as an exact analytic solid, no longer an observable CSG
// decline. The rod at z=7 straddles the notch floor, so its front loop crosses the section conic at two
// triple points where cylinder, rod and notch plane all meet. The result must route to the analytic path
// (NO CSG fallback) and be a valid genus-1 solid. The OCC moment + membership certification lives in
// partial_rim_corner_certification_test.go.
func TestPartialRimCornerJunctionTakesAnalyticPath(t *testing.T) {
	rod, err := brep.SolidCylinder(math.P3(-6, 0, 7), math.V3(1, 0, 0), 1, 12) // z=7: front loop crosses the notch
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	rec := &diag.Recorder{}
	res, err := BooleanWithDiagnostics(Cut, notchedTarget(t), rod, rec)
	if err != nil {
		t.Fatalf("corner-junction cut: %v", err)
	}
	if rec.Has(CodeBooleanCSGFallback) {
		t.Errorf("corner-junction cut fell back to CSG; want the exact analytic path. recs=%v", rec.Records())
	}
	if r := Validate(res); !r.Valid {
		t.Fatalf("corner-junction result is not a valid solid: %v", r.Issues)
	}
	if chi := res.EulerCharacteristic(); chi != 0 {
		t.Errorf("corner-junction result chi=%d; want 0 (genus-1 through-tunnel)", chi)
	}
}

// TestPartialRimGrazingCutDeclinesObservably: a rod that only GRAZES the notch floor (z=5.5, its top tangent
// to the section conic) is a non-transversal contact the tangency gate declines (ADR-0048 §tangency, out of
// scope). It must fall to the observable CSG fallback — never a manifold-but-wrong forced analytic solid.
func TestPartialRimGrazingCutDeclinesObservably(t *testing.T) {
	rod, err := brep.SolidCylinder(math.P3(-6, 0, 5.5), math.V3(1, 0, 0), 1, 12) // z=5.5: top grazes the notch floor
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	rec := &diag.Recorder{}
	if _, err := BooleanWithDiagnostics(Cut, notchedTarget(t), rod, rec); err != nil {
		t.Fatalf("grazing cut errored instead of falling back: %v", err)
	}
	if !rec.Has(CodeBooleanCSGFallback) {
		t.Errorf("grazing partial-rim cut did not record the CSG fallback; got %v", rec.Records())
	}
}
