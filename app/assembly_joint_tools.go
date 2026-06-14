// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/model/assembly"
)

// The assembly joint tools (M12-F02): each Joints-panel command starts an AssemblyJointTool
// for one joint kind. The user picks the component faces that define the two joint origins;
// on commit the tool resolves each pick to its occurrence and definition-space frame (reusing
// the constraint pick resolver — a planar face carries a U-axis secondary so rigid/slider can
// lock the roll), creates the joint, runs the assembly's combined constraint+joint solve, and
// records the undo step.

// jointBuild creates a joint of a specific kind from the resolved joint-origin refs.
type jointBuild func(js *assembly.JointSet, refs []assembly.Ref) assembly.Joint

// AssemblyJointTool collects the component faces that define a joint and creates it.
type AssemblyJointTool struct {
	label string
	build jointBuild
	faces []FaceHandle
}

// NewAssemblyJointTool builds a joint tool that needs two face picks and creates its joint.
func NewAssemblyJointTool(label string, build jointBuild) *AssemblyJointTool {
	return &AssemblyJointTool{label: label, build: build}
}

// Name is the tool's display name.
func (t *AssemblyJointTool) Name() string { return t.label }

// Start restricts picking to component faces.
func (t *AssemblyJointTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectFace))
}

// Pick appends a picked face until the joint has its two origins.
func (t *AssemblyJointTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok && len(t.faces) < 2 {
		t.faces = append(t.faces, f)
	}
}

// Prompt tells the user how many more faces to pick.
func (t *AssemblyJointTool) Prompt(*Session) string {
	return fmt.Sprintf("%s joint: pick the two component faces that define the joint origins (%d/2).", t.label, len(t.faces))
}

// CanCommit reports whether both joint origins have been picked.
func (t *AssemblyJointTool) CanCommit() bool { return len(t.faces) == 2 }

// Cancel abandons the picks and clears the face filter.
func (t *AssemblyJointTool) Cancel(s *Session) {
	t.faces = nil
	s.Selection().SetFilter(NewSelectionFilter())
}

// Commit resolves the picks, creates the joint, solves the assembly, and records the edit.
func (t *AssemblyJointTool) Commit(s *Session) error {
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
	j := t.build(asm.Joints(), refs)
	asm.SolveConstraints()
	s.recordEdit(asm, j.Name())
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}
