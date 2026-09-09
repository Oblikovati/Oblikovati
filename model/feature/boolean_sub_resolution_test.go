// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
)

// TestASubResolutionCutSickensItsFeature carries the kernel's size classification all the way to the
// browser. It drives the EXACT configuration the kernel sweep measured — the RING corpus body cut by
// an axial drill of radius 1e-10, which the sweep
// (kernel/ops/boolean/boolean_drill_sweep_test.go) shows the pipeline answered SILENTLY: the ring
// came back untouched, err=nil, nothing recorded.
//
// It combines two BUILT bodies rather than extruding a sketched circle, deliberately. The earlier
// version of this row pinned a 0.4 mm bore in a 400-unit plate, which is ~1.5x below the floor it was
// written against and was never shown to be unbuildable at all (finding 4 of the stage-6 review);
// worse, driving it through a sketch confounds the kernel's resolution floor with the sketch
// solver's own smallest workable circle. The corpus configuration has neither problem.
func TestASubResolutionCutSickensItsFeature(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(5, 0, -4), math.V3(0, 0, 1), 1e-10, 8)
	if err != nil {
		t.Fatalf("drill: %v", err)
	}
	fs := NewPartFeatures(nil)
	base := NewBaseFeatures(fs).AddBase(ring, drill)
	cut := NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)

	fs.Recompute()
	if !base.Health().OK() {
		t.Fatalf("the base bodies are valid and must stay healthy: %+v", base.Health())
	}
	if cut.Health().Status != health.Sick {
		t.Fatalf("a sub-resolution cut must sicken its feature; got %+v", cut.Health())
	}
	if !hasDiagCode(cut.Diagnostics(), ops.CodeBooleanSubResolutionTool) {
		t.Errorf("the kernel's named refusal must reach feature health as %q; got %v",
			ops.CodeBooleanSubResolutionTool, cut.Diagnostics())
	}
	for _, want := range []string{"below this model's resolution", "working unit"} {
		if !strings.Contains(cut.Health().Reason, want) {
			t.Errorf("the sick reason must name %q so the user can act on it; got %q", want, cut.Health().Reason)
		}
	}
	// Failure is LOCAL: the two bodies the refused cut was to combine survive the recompute.
	if got := len(fs.Result()); got != 2 {
		t.Errorf("the refused cut must leave the operands standing; the result holds %d bodies", got)
	}
}

// TestAnOrdinaryBoreIsNotRefusedOnSize is the counterpart the floor needs: the RD- corpus row's own
// 0.8 bore goes through the same feature path and must NOT meet the size classification. A floor that
// refuses ordinary geometry would be worse than no floor at all.
func TestAnOrdinaryBoreIsNotRefusedOnSize(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(5, 0, -4), math.V3(0, 0, 1), 0.8, 8)
	if err != nil {
		t.Fatalf("drill: %v", err)
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(ring, drill)
	cut := NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)
	fs.Recompute()

	if !cut.Health().OK() {
		t.Fatalf("the RD- corpus bore must build: %+v", cut.Health())
	}
	if hasDiagCode(cut.Diagnostics(), ops.CodeBooleanSubResolutionTool) {
		t.Errorf("an ordinary bore must not meet the size classification; got %v", cut.Diagnostics())
	}
}

// TestAPlanarRetryCannotOverturnASizeRefusal is the feature-layer half of "decide each incidence
// once" (#3524). combine has TWO kernel entries — the exact curved attempt and, when that declines,
// a planar retry on FACETED copies of both operands — and each classified the pair's size for
// itself. Measured on a 1e-10-radius cylinder cutting a 10-cube, the two disagreed: the curved entry
// refused by name and the planar retry then built on the faceted stand-in, returning the cube
// untouched with no diagnostic. The user saw a healthy feature that had removed nothing.
//
// Faceting cannot make an operand thicker, so the size verdict is final: the retry never runs, and
// the pair is classified exactly once per feature boolean.
func TestAPlanarRetryCannotOverturnASizeRefusal(t *testing.T) {
	t.Parallel()
	cube, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "cube")
	if err != nil {
		t.Fatalf("cube: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(5, 5, -1), math.V3(0, 0, 1), 1e-10, 12)
	if err != nil {
		t.Fatalf("drill: %v", err)
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(cube, drill)
	cut := NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)
	fs.Recompute()

	if cut.Health().Status != health.Sick {
		t.Fatalf("a sub-resolution cut must sicken its feature, not build silently; got %+v", cut.Health())
	}
	if !strings.Contains(cut.Health().Reason, "below this model's resolution") {
		t.Errorf("the refusal must survive to health; got %q", cut.Health().Reason)
	}
	if got := countDiagCode(cut.Diagnostics(), ops.CodeBooleanSubResolutionTool); got != 1 {
		t.Errorf("the pair must be classified exactly ONCE per feature boolean; got %d size defects in %v",
			got, cut.Diagnostics())
	}
}

// countDiagCode counts how many diagnostics carry a code — one classification, one record.
func countDiagCode(ds []diag.Diagnostic, code diag.Code) int {
	n := 0
	for _, d := range ds {
		if d.Code == code {
			n++
		}
	}
	return n
}

// TestThePlanarizedToolIsMeasuredToo is the corpus row for the operand this engine builds ITSELF.
// combine facets a curved tool before the planar boolean, and planarized() turns a 1e-10-radius
// cylinder into a 26-face prism whose side normals have partly collapsed toward the cap normal. A
// width that needed a pair of opposed planar faces read that prism's LENGTH — 12 — so the planar
// retry built on it and handed the target back untouched with nothing recorded (#3524 review C1).
//
// Measured: the 26 faces carry TWO distinct plane normals, (0,-0,1) x25 and (0,0,1) x1, which are
// parallel — so the body names ONE direction and is 12 long in it. This is therefore the row that
// exercises the principal-frame backstop (boolean.principalWidth), the only path that can measure a
// body whose own boundary names fewer than three independent directions.
//
// The row drives the planarized body directly rather than through combine, because combine now
// refuses at the curved attempt and never reaches the retry: this is the measurement itself.
func TestThePlanarizedToolIsMeasuredToo(t *testing.T) {
	t.Parallel()
	cube, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "cube")
	if err != nil {
		t.Fatalf("cube: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(5, 5, -1), math.V3(0, 0, 1), 1e-10, 12)
	if err != nil {
		t.Fatalf("drill: %v", err)
	}
	flat := planarized(drill, "sub-resolution-tool")
	if got := len(flat.Faces()); got < 3 {
		t.Fatalf("planarized() must have faceted the cylinder; got %d faces", got)
	}
	rec := &diag.Recorder{}
	body, err := ops.BooleanWithDiagnostics(ops.Cut, cube, flat, rec)
	if !errors.Is(err, ops.ErrSubResolutionOperand) {
		faces := 0
		if body != nil {
			faces = len(body.Faces())
		}
		t.Fatalf("the faceted 1e-10 tool must be refused by name; got err=%v, %d faces, %d diagnostics",
			err, faces, len(rec.Records()))
	}
	if !hasDiagCode(rec.Records(), ops.CodeBooleanSubResolutionTool) {
		t.Errorf("the refusal must reach the diagnostic channel; got %v", rec.Records())
	}
}
