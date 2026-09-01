// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
)

// TestCombineRecordsAnalyticFaceting is the #1601 fallback-visibility regression: cutting one
// analytic cylinder with another where no exact curved path applies facets BOTH operands for the
// planar boolean — permanently. That degradation must ride on the feature as a diagnostic through
// the DEFAULT engine recompute (it used to vanish: ops.Boolean was called with rec=nil, and the
// pre-facet happened before ops.Boolean could even see curved operands).
func TestCombineRecordsAnalyticFaceting(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	extrudes := NewExtrudeFeatures(fs)
	base := extrudes.AddByDistanceExtent(circleSketchAt(0, 0, 2), 0, ops.NewBody, func() float64 { return 3 })
	cut := extrudes.AddByDistanceExtent(circleSketchAt(1.5, 0, 1), 0, ops.Cut, func() float64 { return 3 })

	fs.Recompute()
	if !base.Health().OK() || !cut.Health().OK() {
		t.Fatalf("features went unhealthy: base=%+v cut=%+v", base.Health(), cut.Health())
	}
	if len(base.Diagnostics()) != 0 {
		t.Errorf("the plain extrude carries %d diagnostics, want 0: %v", len(base.Diagnostics()), base.Diagnostics())
	}
	if !hasDiagCode(cut.Diagnostics(), ops.CodeBooleanAnalyticFaceted) {
		t.Errorf("cylinder-on-cylinder cut faceted its analytic operands with no %q diagnostic; got %v",
			ops.CodeBooleanAnalyticFaceted, cut.Diagnostics())
	}
}

// TestExactCurvedPathStaysQuiet: a cut the exact curved boolean handles (a planar box drilled by a
// cylinder, #1472) must NOT carry the faceting defect — the diagnostic flags real degradations
// only, or it becomes noise nobody reads (#1601).
func TestExactCurvedPathStaysQuiet(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	extrudes := NewExtrudeFeatures(fs)
	base := extrudes.AddByDistanceExtent(squareSketch(4), 0, ops.NewBody, func() float64 { return 2 })
	drill := extrudes.AddByDistanceExtent(circleSketchAt(2, 2, 0.5), 0, ops.Cut, func() float64 { return 2 })

	fs.Recompute()
	if !base.Health().OK() || !drill.Health().OK() {
		t.Fatalf("features went unhealthy: base=%+v drill=%+v", base.Health(), drill.Health())
	}
	if hasDiagCode(drill.Diagnostics(), ops.CodeBooleanAnalyticFaceted) || hasDiagCode(drill.Diagnostics(), ops.CodeBooleanCSGFallback) {
		t.Errorf("the exact box-drill cut carries degradation diagnostics it should not: %v", drill.Diagnostics())
	}
}

// hasDiagCode reports whether any recorded diagnostic carries the code.
func hasDiagCode(diags []diag.Diagnostic, code diag.Code) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}
