// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
)

// addFlange folds a 90° flange onto a top +X edge of the running sheet and recomputes,
// returning the feature so a test can suppress it. The default rule (1 mm gauge, 1 mm
// radius) drives the bend, so the developed values are deterministic.
func addFlange(t *testing.T, d *PartComponentDefinition) *feature.PartFeature {
	t.Helper()
	edge := topXEdge(t, d.Features().Result()[0])
	pf := feature.NewSheetMetalFlangeFeatures(d.Features()).Add(&feature.SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(),
		Height:  func() float64 { return 1 },
	})
	d.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("flange unhealthy: %s", pf.Health().Reason)
	}
	return pf
}

// topXEdge returns an X-aligned edge on the sheet's top face — a valid edge to flange from.
func topXEdge(t *testing.T, body *topo.Body) *topo.Edge {
	t.Helper()
	maxZ := math.Inf(-1)
	for _, e := range body.Edges() {
		if z := e.StartVertex().Point().Z; z > maxZ {
			maxZ = z
		}
	}
	for _, e := range body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if math.Abs(a.X-b.X) > 1e-6 && math.Abs(a.Y-b.Y) < 1e-6 && math.Abs(a.Z-maxZ) < 1e-6 {
			return e
		}
	}
	t.Fatal("no X-aligned top edge on the sheet")
	return nil
}

// sheetWithFlange builds a sheet-metal part (square base + one flange) for the bend tests.
func sheetWithFlange(t *testing.T) (*PartComponentDefinition, *feature.PartFeature) {
	t.Helper()
	d := NewPartComponentDefinition()
	if _, err := d.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	addSquareFace(d, 4)
	return d, addFlange(t, d)
}

// TestBendsReportsFlangeLineage a flanged sheet reports its single 90° bend with the
// developed allowance the rule's unfold method computes.
func TestBendsReportsFlangeLineage(t *testing.T) {
	t.Parallel()
	d, flange := sheetWithFlange(t)

	bends := d.Bends()
	if len(bends) != 1 {
		t.Fatalf("Bends len = %d, want 1", len(bends))
	}
	b := bends[0]
	if b.Feature != flange.Name() {
		t.Errorf("bend feature = %q, want %q", b.Feature, flange.Name())
	}
	if math.Abs(b.Angle-math.Pi/2) > 1e-12 {
		t.Errorf("bend angle = %g, want π/2", b.Angle)
	}
	rule := d.SheetMetal()
	if b.Radius != rule.BendRadius() || b.Thickness != rule.Thickness() {
		t.Errorf("bend (r=%g,t=%g), want rule (r=%g,t=%g)", b.Radius, b.Thickness, rule.BendRadius(), rule.Thickness())
	}
	if want := rule.Unfold().BendAllowance(b.Angle, b.Radius, b.Thickness); b.Allowance != want {
		t.Errorf("allowance = %g, want %g", b.Allowance, want)
	}
	if got := d.TotalBendAllowance(); math.Abs(got-b.Allowance) > 1e-12 {
		t.Errorf("TotalBendAllowance = %g, want %g", got, b.Allowance)
	}
}

// TestBendsNilWhenNotSheetMetal an ordinary part has no bend lineage.
func TestBendsNilWhenNotSheetMetal(t *testing.T) {
	t.Parallel()
	d := NewPartComponentDefinition()
	if bends := d.Bends(); bends != nil {
		t.Errorf("plain part Bends = %v, want nil", bends)
	}
}

// TestBendsExcludesSuppressed a suppressed flange contributes no bend (it added no geometry).
func TestBendsExcludesSuppressed(t *testing.T) {
	t.Parallel()
	d, flange := sheetWithFlange(t)
	flange.SetSuppressed(true)
	d.Recompute()
	if bends := d.Bends(); len(bends) != 0 {
		t.Errorf("suppressed flange still reports %d bends, want 0", len(bends))
	}
}
