// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
)

// TestAssemblyBrowserListsConstraintsInOrder: an assembly's relationships appear under a
// Constraints folder in creation order, each a selectable row carrying an
// AssemblyConstraintHandle (M12-F01) — the order-of-creation list the user reads beside the
// parts.
func TestAssemblyBrowserListsConstraintsInOrder(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	placedWidget(t, s, asm, "widget:2")
	a := asm.Occurrences().Item(0)
	b := asm.Occurrences().Item(1)
	zUp, _ := math.NewUnitVector3(0, 0, 1)

	mate := asm.Constraints().AddMate(
		assembly.Ref{Occurrence: a, Primitive: assembly.PlanePrimitive(math.P3(0, 0, 0), zUp)},
		assembly.Ref{Occurrence: b, Primitive: assembly.PlanePrimitive(math.P3(0, 0, 0), zUp)},
		0, types.MateSolutionOpposed)
	angle := asm.Constraints().AddAngle(
		assembly.Ref{Occurrence: a, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), zUp)},
		assembly.Ref{Occurrence: b, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), zUp)},
		stdmath.Pi/4, types.AngleSolutionUndirected)

	root := BuildBrowser(s)
	folder := findBrowserNode(root, "constraints", "Constraints")
	if folder == nil {
		t.Fatal("assembly browser has no Constraints folder")
	}
	if len(folder.Children) != 2 {
		t.Fatalf("Constraints folder has %d rows, want 2", len(folder.Children))
	}
	if folder.Children[0].Label != mate.Name() || folder.Children[1].Label != angle.Name() {
		t.Errorf("constraint rows = [%q,%q], want creation order [%q,%q]",
			folder.Children[0].Label, folder.Children[1].Label, mate.Name(), angle.Name())
	}
	h, ok := folder.Children[0].Select.(AssemblyConstraintHandle)
	if !ok || h.Constraint.ID() != mate.ID() {
		t.Errorf("first row selects %T (%v), want the mate's AssemblyConstraintHandle", folder.Children[0].Select, h)
	}
}

// TestConstraintBrowserLabelSuppressed: a suppressed constraint's row is annotated, like an
// occurrence's.
func TestConstraintBrowserLabelSuppressed(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	placedWidget(t, s, asm, "widget:2")
	zUp, _ := math.NewUnitVector3(0, 0, 1)
	mate := asm.Constraints().AddMate(
		assembly.Ref{Occurrence: asm.Occurrences().Item(0), Primitive: assembly.PlanePrimitive(math.P3(0, 0, 0), zUp)},
		assembly.Ref{Occurrence: asm.Occurrences().Item(1), Primitive: assembly.PlanePrimitive(math.P3(0, 0, 0), zUp)},
		0, types.MateSolutionOpposed)

	if got := constraintBrowserLabel(mate); got != "Mate:1" {
		t.Errorf("plain label = %q, want Mate:1", got)
	}
	mate.SetSuppressed(true)
	if got := constraintBrowserLabel(mate); got != "Mate:1 (suppressed)" {
		t.Errorf("suppressed label = %q, want \"Mate:1 (suppressed)\"", got)
	}
}
