// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// CommitAssemblyFeature hosts f on the active assembly, recomputes the feature
// program, and records one undo step under label — the single mutation seam
// both the head UI tools and the wire router finish an assembly-feature add
// through (#1612, audit B1). The attach policy itself (unique naming, proxy-cut
// source exclusion) lives on the aggregate's AddFeature.
//
//	af, err := s.CommitAssemblyFeature(feature.NewAssemblyProxyCutFeature(src, ops.Cut), "Add Assembly Feature")
func (s *Session) CommitAssemblyFeature(f feature.Feature, label string) (*compdef.AssemblyFeature, error) {
	asm, err := activeAssembly(s)
	if err != nil {
		return nil, err
	}
	af := asm.AddFeature(f)
	asm.RecomputeFeatures()
	s.recordEdit(asm, label) // the feature program persists + undoes (#785)
	return af, nil
}

// CommitAssemblyFeatureChange finalizes an in-place change to the active
// assembly's feature program (participation, suppression, scalar edits, the
// rollback marker): mutate runs against the assembly, then one recompute and
// one undo step under label — the shared sequencing both drivers finish
// through (#1612, audit B1).
//
//	err := s.CommitAssemblyFeatureChange("Edit Participants", func(asm *compdef.AssemblyComponentDefinition) error {
//		af.SetParticipants(occs); return nil
//	})
func (s *Session) CommitAssemblyFeatureChange(label string, mutate func(asm *compdef.AssemblyComponentDefinition) error) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	if err := mutate(asm); err != nil {
		return err
	}
	asm.RecomputeFeatures()
	s.recordEdit(asm, label)
	return nil
}
