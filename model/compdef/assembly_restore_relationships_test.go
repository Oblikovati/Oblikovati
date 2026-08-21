// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
)

// TestRestoreSnapshotRewiresRelationshipSets verifies that an assembly snapshot restore
// replaces the relationship sets and that reference resolution reconnects them to the
// restored live occurrence collection. A relationship created after rebind must be usable
// by the solver; a relationship created before the snapshot must not survive the restore.
func TestRestoreSnapshotRewiresRelationshipSets(t *testing.T) {
	_, _, asm, widget, asmDef := placedAssembly(t)
	base := placeFromFile(t, asm, widget, asmDef, "widget:1", math.Identity4())
	moving := placeFromFile(t, asm, widget, asmDef, "widget:2", math.Translation4(math.V3(0, 0, 10)))

	baseline, err := asmDef.MarshalSnapshot()
	if err != nil {
		t.Fatalf("marshal baseline: %v", err)
	}

	zUp, _ := math.NewUnitVector3(0, 0, 1)
	zDown, _ := math.NewUnitVector3(0, 0, -1)
	asmDef.Constraints().AddMate(
		assembly.Ref{Occurrence: base, Primitive: assembly.PlanePrimitive(math.P3(0, 0, 0), zUp)},
		assembly.Ref{Occurrence: moving, Primitive: assembly.PlanePrimitive(math.P3(0, 0, 0), zDown)},
		0, types.MateSolutionOpposed)
	if got := asmDef.Constraints().Count(); got != 1 {
		t.Fatalf("after pre-restore AddMate: constraint count = %d, want 1", got)
	}

	if err := asmDef.RestoreSnapshot(baseline); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := asmDef.Constraints().Count(); got != 0 {
		t.Fatalf("constraint count before reference rebind = %d, want 0", got)
	}
	if err := asmDef.ResolveReferences(asm); err != nil {
		t.Fatalf("resolve references: %v", err)
	}
	if got := asmDef.Occurrences().Count(); got != 2 {
		t.Fatalf("restored occurrence count = %d, want 2", got)
	}

	baseRestored := asmDef.Occurrences().Item(0)
	movingRestored := asmDef.Occurrences().Item(1)
	asmDef.Constraints().AddMate(
		assembly.Ref{Occurrence: baseRestored, Primitive: assembly.PlanePrimitive(math.P3(0, 0, 0), zUp)},
		assembly.Ref{Occurrence: movingRestored, Primitive: assembly.PlanePrimitive(math.P3(0, 0, 0), zDown)},
		0, types.MateSolutionOpposed)
	if got := asmDef.Constraints().Count(); got != 1 {
		t.Fatalf("after post-restore AddMate: constraint count = %d, want 1", got)
	}
	if report := asmDef.SolveConstraints(); !report.Converged {
		t.Fatalf("post-restore solve did not converge: %+v", report)
	}
}
