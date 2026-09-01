// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The assembly-feature commit verbs (#1612, audit B1): the one seam the UI
// tools and the wire router add/edit the feature program through.

func TestCommitAssemblyFeatureHostsAndNames(t *testing.T) {
	t.Parallel()
	s := activeAssemblySession(t)
	tool, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "asmTool")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	af, err := s.CommitAssemblyFeature(feature.NewAssemblyCutFeature(tool, ops.Cut), "Add Assembly Feature")
	if err != nil {
		t.Fatalf("CommitAssemblyFeature: %v", err)
	}
	if af.Name() == "" {
		t.Error("the aggregate must have named the hosted feature")
	}
	asm, err := activeAssembly(s)
	if err != nil {
		t.Fatalf("activeAssembly: %v", err)
	}
	if asm.Features().Count() != 1 {
		t.Fatalf("feature count = %d, want 1", asm.Features().Count())
	}

	// An in-place change runs mutate, then recompute + one undo record.
	if err := s.CommitAssemblyFeatureChange("Suppress Assembly Feature", func(a *compdef.AssemblyComponentDefinition) error {
		a.Features().SuppressFeatures(af.ID())
		return nil
	}); err != nil {
		t.Fatalf("CommitAssemblyFeatureChange: %v", err)
	}
	if !af.Suppressed() {
		t.Error("the change must have applied")
	}

	// A failing mutate surfaces its error without recording.
	boom := errors.New("boom")
	if err := s.CommitAssemblyFeatureChange("x", func(*compdef.AssemblyComponentDefinition) error { return boom }); !errors.Is(err, boom) {
		t.Errorf("failing mutate = %v, want boom", err)
	}
}

// TestCommitAssemblyFeatureNeedsAnAssembly covers both verbs' resolution error
// on a part document.
func TestCommitAssemblyFeatureNeedsAnAssembly(t *testing.T) {
	t.Parallel()
	s := newSessionWithPart(t)
	if _, err := s.CommitAssemblyFeature(nil, "x"); err == nil {
		t.Error("CommitAssemblyFeature on a part must error")
	}
	if err := s.CommitAssemblyFeatureChange("x", func(*compdef.AssemblyComponentDefinition) error { return nil }); err == nil {
		t.Error("CommitAssemblyFeatureChange on a part must error")
	}
}
