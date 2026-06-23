// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/model/feature"
)

// The Rebuild Surface tool (M36-F02) refits the running surface body's freeform faces to clean
// Class-A NURBS at a chosen degree and control-point count per direction. It is parameter-only
// (the running surface is the input — nothing is picked), driven by the generic property dialog
// over its four integer parameters, and on commit it reports the achieved deviation as a status
// notice so the user can judge whether the rebuild stayed within tolerance.

// SurfaceRebuildTool rebuilds the running surface body's freeform faces to clean NURBS.
type SurfaceRebuildTool struct {
	dialogTool
	uDegree, vDegree int
	uCount, vCount   int
	added            *feature.PartFeature
}

// NewSurfaceRebuildTool returns a rebuild tool defaulting to a degree-3, 4×4 single span — the
// cleanest Class-A target.
func NewSurfaceRebuildTool() *SurfaceRebuildTool {
	return &SurfaceRebuildTool{uDegree: 3, vDegree: 3, uCount: 4, vCount: 4}
}

// Name implements [Tool].
func (t *SurfaceRebuildTool) Name() string { return "Rebuild Surface" }

// Prompt guides the input.
func (t *SurfaceRebuildTool) Prompt(*Session) string {
	return "Set the target degree and control-point count, then OK to refit the surface."
}

// Params exposes the per-direction degree and control-point count for the generic dialog.
func (t *SurfaceRebuildTool) Params() ToolParams {
	return ToolParams{Ints: []IntParam{
		{Label: "U Degree", Get: func() int { return t.uDegree }, Set: func(v int) { t.uDegree = v }},
		{Label: "V Degree", Get: func() int { return t.vDegree }, Set: func(v int) { t.vDegree = v }},
		{Label: "U Control Points", Get: func() int { return t.uCount }, Set: func(v int) { t.uCount = v }},
		{Label: "V Control Points", Get: func() int { return t.vCount }, Set: func(v int) { t.vCount = v }},
	}}
}

// CanCommit reports whether the targets are valid (each count carries its degree).
func (t *SurfaceRebuildTool) CanCommit() bool {
	return t.uDegree >= 1 && t.vDegree >= 1 && t.uCount >= t.uDegree+1 && t.vCount >= t.vDegree+1
}

// Commit rebuilds the running surface and recomputes, then reports the achieved deviation.
func (t *SurfaceRebuildTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewRebuildFeatures(part.Features()).Add(t.uDegree, t.vDegree, t.uCount, t.vCount)
	part.Recompute()
	s.recordEdit(part, "Rebuild Surface")
	if !t.added.Health().OK() {
		return errors.New("rebuild surface: " + t.added.Health().Reason)
	}
	// The deviation goes to the Command Window (it persists, unlike the transient status notice
	// which OK clears after every tool commit), so the user sees how close the rebuild stayed.
	s.feedNotice(rebuildDeviationNotice(t.added))
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SurfaceRebuildTool) AddedFeature() *feature.PartFeature { return t.added }

// rebuildDeviationNotice formats the post-commit status with the achieved max deviation.
func rebuildDeviationNotice(pf *feature.PartFeature) string {
	if rf, ok := pf.Definition().(*feature.RebuildFeature); ok {
		return fmt.Sprintf("Rebuilt surface — max deviation %.4g", rf.Deviation())
	}
	return "Rebuilt surface"
}
