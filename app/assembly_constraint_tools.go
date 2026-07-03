// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/model/assembly"
)

// The assembly relationship tools (M12-F01, #770): each Relationships-panel command starts an
// AssemblyConstraintTool for one constraint kind. The user picks the component faces the
// relationship relates; on commit the tool resolves each pick to the occurrence it belongs to
// and the face's primitive in that component's DEFINITION space (the picked face is in assembly
// space, so it is mapped back through the occurrence's inverse placement — the solver re-applies
// the placement), creates the constraint of its kind, solves the assembly, and records the undo
// step. The constraint math lives in model/assembly; this is the interaction shell.

// constraintBuild creates a constraint of a specific kind from the resolved geometry inputs.
type constraintBuild func(set *assembly.ConstraintSet, refs []assembly.Ref) assembly.Constraint

// AssemblyConstraintTool collects the component faces a relationship relates and creates it.
type AssemblyConstraintTool struct {
	label string
	need  int
	build constraintBuild
	faces []FaceHandle
}

// NewAssemblyConstraintTool builds a constraint tool that needs `need` face picks and creates
// its constraint with build.
func NewAssemblyConstraintTool(label string, need int, build constraintBuild) *AssemblyConstraintTool {
	return &AssemblyConstraintTool{label: label, need: need, build: build}
}

// Name is the tool's display name.
func (t *AssemblyConstraintTool) Name() string { return t.label }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *AssemblyConstraintTool) Start(*Session) {}

// AcceptedKinds declares the constraint picks component faces.
func (t *AssemblyConstraintTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectFace} }

// Picks reports the picked component faces for the unified highlight.
func (t *AssemblyConstraintTool) Picks() []Selectable { return selectables(t.faces) }

// Pick appends a picked face until the constraint has all the inputs it needs.
func (t *AssemblyConstraintTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok && len(t.faces) < t.need {
		t.faces = append(t.faces, f)
	}
}

// Prompt tells the user how many more faces to pick.
func (t *AssemblyConstraintTool) Prompt(*Session) string {
	return fmt.Sprintf("%s: pick %d component faces, then OK (%d picked).", t.label, t.need, len(t.faces))
}

// CanCommit reports whether every required face has been picked.
func (t *AssemblyConstraintTool) CanCommit() bool { return len(t.faces) == t.need }

// Cancel abandons the picks and clears the face filter.
func (t *AssemblyConstraintTool) Cancel(s *Session) {
	t.faces = nil
}

// Commit resolves the picks, creates the constraint, solves the assembly, and records the edit.
func (t *AssemblyConstraintTool) Commit(s *Session) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	refs := make([]assembly.Ref, 0, len(t.faces))
	for _, f := range t.faces {
		r, err := constraintRefFromFace(s, f)
		if err != nil {
			return err
		}
		refs = append(refs, r)
	}
	c := t.build(asm.Constraints(), refs)
	asm.SolveConstraints()
	s.recordEdit(asm, c.Name())
	return nil
}

// constraintRefFromFace resolves a picked component face to a constraint geometry input: the
// occurrence it belongs to and the face's primitive in that component's definition space (the
// picked face is in assembly space, so it is mapped back through the occurrence's inverse
// placement).
func constraintRefFromFace(s *Session, fh FaceHandle) (assembly.Ref, error) {
	if fh.Face == nil || fh.Body == nil {
		return assembly.Ref{}, errors.New("pick a component face")
	}
	occ, ok := s.OccurrenceOfBody(fh.Body)
	if !ok {
		return assembly.Ref{}, errors.New("pick a face on a placed component")
	}
	worldPrim, err := assembly.PrimitiveFromFace(fh.Face)
	if err != nil {
		return assembly.Ref{}, err
	}
	inv, ok := occ.Transform().Inverse()
	if !ok {
		return assembly.Ref{}, errors.New("the component's placement is not invertible")
	}
	return assembly.Ref{
		Occurrence: occ,
		Primitive:  worldPrim.TransformedBy(inv),
		Entity:     string(componentEdgeSuffix(fh.Face.ReferenceKey())),
	}, nil
}
