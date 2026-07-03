// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Grip Snap (M12 #794): the interactive snap-driven constraint command. The user picks a face on the
// component to move, then a target face on another component; the tool INFERS the assembly constraint
// that snaps the first onto the second (planar faces → mate/flush, cylindrical faces → insert/tangent),
// creates it, and re-solves so the part jumps into place. The Constraint option overrides the
// inference; the HUD reports what was created. It reuses the constraint pick→resolve→solve machinery
// of [AssemblyConstraintTool] — the only new behaviour is the inference + the override.

// gripPreferOrder maps the Move-Options "Constraint" combo index to the override (0 ⇒ auto-infer).
var gripPreferOrder = []types.AssemblyConstraintType{
	0, types.ConstraintMate, types.ConstraintFlush, types.ConstraintInsert, types.ConstraintTangent,
}

// GripSnapPreferOptions are the constraint-override labels for the Move-Options panel, in index order.
func GripSnapPreferOptions() []string { return []string{"Auto", "Mate", "Flush", "Insert", "Tangent"} }

// GripSnapTool snaps the first picked face onto the second by an inferred (or chosen) constraint.
type GripSnapTool struct {
	faces    []FaceHandle
	prefer   types.AssemblyConstraintType // 0 = auto-infer
	inferred string                       // the last created constraint kind, for the HUD
}

// NewGripSnapTool returns a grip-snap tool that auto-infers the constraint.
func NewGripSnapTool() *GripSnapTool { return &GripSnapTool{} }

// Name is the tool's display name.
func (t *GripSnapTool) Name() string { return "Grip Snap" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *GripSnapTool) Start(*Session) {}

// AcceptedKinds declares grip-snap picks component faces (the moving face, then the target face).
func (t *GripSnapTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectFace} }

// Picks reports the picked faces for the unified highlight.
func (t *GripSnapTool) Picks() []Selectable { return selectables(t.faces) }

// Pick appends a face until both the moving and the target geometry are chosen.
func (t *GripSnapTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok && len(t.faces) < 2 {
		t.faces = append(t.faces, f)
	}
}

// Prompt guides the two picks.
func (t *GripSnapTool) Prompt(*Session) string {
	switch len(t.faces) {
	case 0:
		return "Grip Snap: pick a face on the component to move."
	case 1:
		return "Grip Snap: pick the target face to snap onto, then OK."
	default:
		return "Grip Snap: OK to snap, or change the Constraint."
	}
}

// CanCommit reports whether both faces have been picked.
func (t *GripSnapTool) CanCommit() bool { return len(t.faces) == 2 }

// Cancel abandons the picks and clears the filter.
func (t *GripSnapTool) Cancel(s *Session) {
	t.faces = nil
}

// PreferIndex returns the selected Constraint override as a [GripSnapPreferOptions] index.
func (t *GripSnapTool) PreferIndex() int {
	for i, k := range gripPreferOrder {
		if k == t.prefer {
			return i
		}
	}
	return 0
}

// SetPreferIndex selects the Constraint override from a [GripSnapPreferOptions] index.
func (t *GripSnapTool) SetPreferIndex(i int) {
	if i >= 0 && i < len(gripPreferOrder) {
		t.prefer = gripPreferOrder[i]
	}
}

// Inferred returns the kind created on the last commit (for the HUD), or "" before the first snap.
func (t *GripSnapTool) Inferred() string { return t.inferred }

// Commit resolves the two faces, infers (or applies the chosen) constraint, snaps the part into place
// by re-solving, and records the edit.
func (t *GripSnapTool) Commit(s *Session) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	move, err := constraintRefFromFace(s, t.faces[0])
	if err != nil {
		return err
	}
	target, err := constraintRefFromFace(s, t.faces[1])
	if err != nil {
		return err
	}
	c, kind, err := asm.Constraints().InferGripConstraint(move, target, t.prefer)
	if err != nil {
		return err
	}
	asm.SolveConstraints()
	t.inferred = kind.String()
	s.SetNotice(fmt.Sprintf("Grip Snap: created a %s constraint.", kind))
	s.recordEdit(asm, c.Name())
	return nil
}

// compile-time guard that the tool satisfies the interactive Tool + Prompted contracts.
var _ interface {
	Tool
	Prompted
} = (*GripSnapTool)(nil)
