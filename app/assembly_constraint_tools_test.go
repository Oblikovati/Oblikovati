// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/assembly"
)

// TestAssemblyConstraintToolPickProgression: the tool accepts exactly its required number of
// face picks and only then enables commit.
func TestAssemblyConstraintToolPickProgression(t *testing.T) {
	t.Parallel()
	tool := NewAssemblyConstraintTool("Mate", 2, func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
		return set.AddMate(r[0], r[1], 0, types.MateSolutionOpposed)
	})
	if tool.Name() != "Mate" {
		t.Errorf("Name() = %q, want Mate", tool.Name())
	}
	if tool.CanCommit() {
		t.Error("no picks yet: CanCommit should be false")
	}
	tool.Pick(nil, FaceHandle{})
	if tool.CanCommit() {
		t.Error("one pick: CanCommit should be false")
	}
	tool.Pick(nil, FaceHandle{})
	if !tool.CanCommit() {
		t.Error("two picks: CanCommit should be true")
	}
	tool.Pick(nil, FaceHandle{}) // beyond need: ignored
	if len(tool.faces) != 2 {
		t.Errorf("collected %d faces, want 2 (extra picks ignored)", len(tool.faces))
	}
}

// TestAssemblyConstraintToolRejectsUnresolvedPick: committing with a face that does not resolve
// to a placed component is a clean error, not a crash (the tool stays open).
func TestAssemblyConstraintToolRejectsUnresolvedPick(t *testing.T) {
	t.Parallel()
	s := assemblySession(t)
	tool := NewAssemblyConstraintTool("Mate", 2, func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
		return set.AddMate(r[0], r[1], 0, types.MateSolutionOpposed)
	})
	tool.Pick(s, FaceHandle{})
	tool.Pick(s, FaceHandle{})
	if err := tool.Commit(s); err == nil {
		t.Error("commit with unresolved faces should return an error")
	}
}
