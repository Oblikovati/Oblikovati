// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// SweepTool is the interactive Sweep command, following Inventor's Sweep dialog: activate it,
// click a sketch region (the profile) and a sketch path (on another plane), then shape how the
// profile rides the path — its orientation (follow the path, or stay parallel to its original
// plane), a taper (draft) and twist along the length, and an optional guide rail that steers and
// scales the profile — and OK to sweep it into a solid. The path's sketch plane maps its 2D chain
// to the 3D rail the model sweep consumes.
type SweepTool struct {
	profile     *ProfileHandle
	path        *PathHandle
	guideRail   *PathHandle
	armRail     bool // route the next path pick to the guide rail, not the path
	twist       float64
	taper       float64
	orientation types.SweepProfileOrientation
	scaling     types.SweepProfileScaling
	operation   ops.PartFeatureOperation
	added       *feature.PartFeature
}

// NewSweepTool returns a sweep tool that creates a new body, defaulting to Inventor's defaults: the
// profile follows the path (normal to it) and a guide rail scales the profile in both X and Y.
func NewSweepTool() *SweepTool {
	return &SweepTool{
		operation:   ops.NewBody,
		orientation: types.NormalToPath,
		scaling:     types.XYProfileScaling,
	}
}

// Name implements [Tool].
func (t *SweepTool) Name() string { return "Sweep" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *SweepTool) Start(*Session) {}

// AcceptedKinds declares sweep picks a closed region (profile) and a path to sweep it along.
func (t *SweepTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectProfile, SelectPath}
}

// Picks reports the picked profile, path, and guide rail for the unified highlight.
func (t *SweepTool) Picks() []Selectable {
	return appendPick(appendPick(appendPick(nil, t.profile), t.path), t.guideRail)
}

// Pick routes a profile pick to the profile slot and a path pick to the path slot — or to the
// guide-rail slot when rail picking is armed (a rail is also a PathHandle, so the slot is chosen
// by the armed selector, mirroring Inventor's dialog). Arming clears once the rail is picked.
func (t *SweepTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case ProfileHandle:
		pc := h
		t.profile = &pc
	case PathHandle:
		pc := h
		if t.armRail {
			t.guideRail = &pc
			t.armRail = false
			return
		}
		t.path = &pc
	}
}

// SetTwist/Twist set the total twist (radians) spread along the path; SetOperation
// chooses the boolean.
func (t *SweepTool) SetTwist(radians float64)                 { t.twist = radians }
func (t *SweepTool) Twist() float64                           { return t.twist }
func (t *SweepTool) SetOperation(op ops.PartFeatureOperation) { t.operation = op }
func (t *SweepTool) Operation() ops.PartFeatureOperation      { return t.operation }

// SetTaper/Taper set the draft (taper) angle (radians) the profile scales by along the path:
// positive expands the section, negative contracts it.
func (t *SweepTool) SetTaper(radians float64) { t.taper = radians }
func (t *SweepTool) Taper() float64           { return t.taper }

// SetOrientation/Orientation choose how the profile rides the path: NormalToPath (Inventor's
// "Follow Path", the default) keeps it perpendicular to the path; ParallelToOriginalProfile
// ("Parallel") keeps it parallel to its sketch plane the whole way.
func (t *SweepTool) SetOrientation(o types.SweepProfileOrientation) { t.orientation = o }
func (t *SweepTool) Orientation() types.SweepProfileOrientation     { return t.orientation }

// SetScaling/Scaling choose how a guide rail scales the profile: XY (both axes), X (one axis),
// or None (the rail only steers orientation). Has no effect without a guide rail.
func (t *SweepTool) SetScaling(s types.SweepProfileScaling) { t.scaling = s }
func (t *SweepTool) Scaling() types.SweepProfileScaling     { return t.scaling }

// ArmGuideRailPicking routes the next path pick to the guide-rail slot (the dialog arms it when
// the Guide Rail selector is clicked). GuideRailArmed reports that state for the chip's label.
func (t *SweepTool) ArmGuideRailPicking() { t.armRail = true }
func (t *SweepTool) GuideRailArmed() bool { return t.armRail }

// PickedGuideRail / ClearGuideRail report and empty the optional guide rail (the Guide Rail chip).
func (t *SweepTool) PickedGuideRail() (PathHandle, bool) {
	if t.guideRail == nil {
		return PathHandle{}, false
	}
	return *t.guideRail, true
}

func (t *SweepTool) ClearGuideRail() {
	t.guideRail = nil
	t.armRail = false
}

// SweepType reports the placed feature's kind from what has been picked — a plain path sweep, or a
// path-and-guide-rail sweep once a rail is set. The dialog reads it to show the rail's scaling row.
func (t *SweepTool) SweepType() types.SweepType {
	if t.guideRail != nil {
		return types.PathAndGuideRailSweepType
	}
	return types.PathSweepType
}

// PickedProfile / PickedPath report what has been gathered (for the UI/tests).
func (t *SweepTool) PickedProfile() (ProfileHandle, bool) {
	if t.profile == nil {
		return ProfileHandle{}, false
	}
	return *t.profile, true
}

func (t *SweepTool) PickedPath() (PathHandle, bool) {
	if t.path == nil {
		return PathHandle{}, false
	}
	return *t.path, true
}

// CanCommit reports whether both a profile and a path have been picked.
func (t *SweepTool) CanCommit() bool { return t.profile != nil && t.path != nil }

// ClearProfile / ClearPath empty one pick each — the property panel's selector clear
// (⊗) affordances on the Profiles and Path chips.
func (t *SweepTool) ClearProfile() { t.profile = nil }
func (t *SweepTool) ClearPath()    { t.path = nil }

// SourceSketchName returns the sketch the picked profile comes from, for the property
// panel's breadcrumb; "" until a profile is picked.
func (t *SweepTool) SourceSketchName() string {
	if t.profile == nil {
		return ""
	}
	return t.profile.Sketch.Name()
}

// Commit resolves the path to a 3D rail, adds the sweep feature, and recomputes; a sick
// feature keeps the tool open by returning an error.
func (t *SweepTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if t.added, err = t.addSweep(part.Features()); err != nil {
		return err
	}
	part.Recompute()
	s.recordEdit(part, "Sweep")
	if !t.added.Health().OK() {
		return errors.New("sweep: " + t.added.Health().Reason)
	}
	return nil
}

// resolveSweepPath turns a picked sketch path into a model-space 3D rail by mapping its
// 2D chain through the path's sketch plane.
func resolveSweepPath(h *PathHandle) (*sketch.Path3D, error) {
	paths := h.Sketch.Paths()
	if h.PathIndex < 0 || h.PathIndex >= len(paths) {
		return nil, fmt.Errorf("sweep: path %d not found (sketch has %d)", h.PathIndex, len(paths))
	}
	p := paths[h.PathIndex]
	plane := h.Sketch.Plane()
	pts := p.Points()
	chain := make([]*sketch.Point3D, len(pts))
	for i, q := range pts {
		chain[i] = sketch.NewPoint3D(plane.ToModel(q))
	}
	return sketch.NewPath3D(chain, p.IsClosed()), nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SweepTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the sweep steps.
func (t *SweepTool) Prompt(*Session) string {
	switch {
	case t.armRail:
		return "Select a guide rail to steer and scale the profile"
	case t.profile == nil:
		return "Select a region to sweep"
	case t.path == nil:
		return "Select a path to sweep along"
	default:
		return "Set the options and click OK"
	}
}

// Preview outlines the picked profile region until the sweep is committed.
// addSweep resolves the path (and any guide rail) to live 3D-rail providers and builds the sweep
// definition into engine fs — the shared constructor used by both Commit (the part's engine) and
// DraftFeature (a scratch engine), so the preview matches the committed result. The path and rail
// re-derive each recompute so a parameter driving their sketch reshapes the sweep (#1414).
func (t *SweepTool) addSweep(fs *feature.PartFeatures) (*feature.PartFeature, error) {
	if _, err := resolveSweepPath(t.path); err != nil { // validate up front; surface a bad path now
		return nil, err
	}
	twist, taper := t.twist, t.taper
	def := &feature.SweepDefinition{
		Sketch:       t.profile.Sketch,
		ProfileIndex: t.profile.ProfileIndex,
		Path:         livePathProvider(*t.path),
		Twist:        func() float64 { return twist },
		Taper:        func() float64 { return taper },
		Operation:    t.operation,
		Orientation:  t.orientation,
	}
	if t.guideRail != nil {
		def.GuideRail = livePathProvider(*t.guideRail)
		def.Scaling = t.scaling
	}
	return feature.NewSweepFeatures(fs).AddDefinition(def), nil
}

// livePathProvider re-resolves a picked sketch path to a model-space 3D rail on each recompute, so
// the sweep tracks edits to the path's (or rail's) sketch instead of snapshotting it once.
func livePathProvider(h PathHandle) func() *sketch.Path3D {
	return func() *sketch.Path3D {
		p, err := resolveSweepPath(&h)
		if err != nil {
			return nil
		}
		return p
	}
}

// DraftFeature returns the unattached sweep feature the viewport previews before commit
// (satisfying DraftPreviewable), built by the same addSweep the commit uses. Empty until both
// a profile and a path are picked.
func (t *SweepTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(t.addSweep)
}

// Cancel restores the default selection filter.
func (t *SweepTool) Cancel(*Session) {}
