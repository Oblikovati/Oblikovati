// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestGatedCurvedCentralisesTheValidityGate pins the contract of gatedCurved — the single adapter the
// curved-pair paths are built from (#1502). It must apply BOTH the op guard and the validBooleanSolid
// gate in one place, so a pair added to the table cannot ship an unvalidated boolean. The non-solid case
// is the load-bearing one: a builder that reports ok=true but returns geometry that is not a closed
// manifold solid must still be rejected.
func TestGatedCurvedCentralisesTheValidityGate(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	// A body carrying the cylinder's faces but flagged NOT solid (an open shell): IsSolid() is false, so
	// validBooleanSolid rejects it regardless of its faces — the gate the adapter must apply.
	openShell := topo.MergeBodies(topo.NewLineage(topo.Tok("test", "open", 0)), false, cyl)
	returns := func(b *topo.Body, ok bool) ruledBuild {
		return func(_, _ *topo.Body, _ *diag.Recorder) (*topo.Body, bool) { return b, ok }
	}

	// Op guard: a path declines when the requested op is not the one it handles.
	if _, ok := gatedCurved(Intersect, returns(cyl, true))(Cut, cyl, cyl, nil); ok {
		t.Error("gatedCurved must decline when the op does not match its want")
	}
	// A declining builder propagates as a decline.
	if _, ok := gatedCurved(Intersect, returns(nil, false))(Intersect, cyl, cyl, nil); ok {
		t.Error("gatedCurved must decline when the builder declines (ok=false)")
	}
	// The centralised gate: a non-solid result is rejected even though the builder said ok=true.
	if _, ok := gatedCurved(Intersect, returns(openShell, true))(Intersect, cyl, cyl, nil); ok {
		t.Error("gatedCurved must gate out a non-solid result (the validBooleanSolid invariant)")
	}
	// A valid solid with a matching op is adopted unchanged.
	if res, ok := gatedCurved(Intersect, returns(cyl, true))(Intersect, cyl, cyl, nil); !ok || res != cyl {
		t.Errorf("gatedCurved must adopt a valid solid result (ok=%v, same=%v)", ok, res == cyl)
	}
}

// TestWithoutRecorderDropsTheRecorder confirms the adapter that lets a bespoke constructor (no SSI
// imprint) satisfy ruledBuild ignores the recorder and forwards the operands unchanged.
func TestWithoutRecorderDropsTheRecorder(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	var gotTarget, gotTool *topo.Body
	build := withoutRecorder(func(target, tool *topo.Body) (*topo.Body, bool) {
		gotTarget, gotTool = target, tool
		return target, true
	})
	res, ok := build(cyl, cyl, nil) // nil recorder must be fine (it is dropped)
	if !ok || res != cyl || gotTarget != cyl || gotTool != cyl {
		t.Errorf("withoutRecorder must forward operands and drop the recorder (ok=%v)", ok)
	}
}
