// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// The Network Surface tool (M36-F10) builds a single NURBS through a grid of intersecting curves:
// pick the U-direction sketch curves, flip "Pick V curves", pick the V-direction curves, then OK. The
// picked profiles are baked to model-space polylines and fitted; the Gordon network surface
// interpolates the grid. Needs ≥2 curves each way that approximately intersect.

// NetworkTool collects U- and V-direction profile picks for a network surface.
type NetworkTool struct {
	dialogTool
	uProfiles []ProfileHandle
	vProfiles []ProfileHandle
	pickingV  bool
	added     *feature.PartFeature
}

// NewNetworkTool returns a network tool, picking U curves first.
func NewNetworkTool() *NetworkTool { return &NetworkTool{} }

// Name implements [Tool].
func (t *NetworkTool) Name() string { return "Network Surface" }

// Start is a no-op; the engine installs the profile filter from AcceptedKinds.
func (t *NetworkTool) Start(*Session) {}

// AcceptedKinds declares the tool picks sketch profiles (the grid curves).
func (t *NetworkTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectProfile} }

// Picks reports all picked profiles for the unified highlight.
func (t *NetworkTool) Picks() []Selectable {
	return profileSelectables(append(append([]ProfileHandle{}, t.uProfiles...), t.vProfiles...))
}

// Pick adds the clicked profile to the current (U or V) direction set, ignoring duplicates.
func (t *NetworkTool) Pick(_ *Session, sel Selectable) {
	h, ok := sel.(ProfileHandle)
	if !ok || t.has(h) {
		return
	}
	if t.pickingV {
		t.vProfiles = append(t.vProfiles, h)
	} else {
		t.uProfiles = append(t.uProfiles, h)
	}
}

func (t *NetworkTool) has(h ProfileHandle) bool {
	for _, p := range append(append([]ProfileHandle{}, t.uProfiles...), t.vProfiles...) {
		if p == h {
			return true
		}
	}
	return false
}

// Prompt guides the two-stage pick.
func (t *NetworkTool) Prompt(*Session) string {
	if !t.pickingV {
		return "Pick the U-direction curves, then enable \"Pick V curves\"."
	}
	return "Pick the V-direction curves, then OK."
}

// Params exposes the U/V pick-direction toggle.
func (t *NetworkTool) Params() ToolParams {
	return ToolParams{Bools: []BoolParam{{
		Label: "Pick V curves", Get: func() bool { return t.pickingV }, Set: func(v bool) { t.pickingV = v },
	}}}
}

// CanCommit reports whether there are at least two curves each way.
func (t *NetworkTool) CanCommit() bool { return len(t.uProfiles) >= 2 && len(t.vProfiles) >= 2 }

// Commit bakes the picked profiles to polylines and builds the network surface.
func (t *NetworkTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewNetworkFeatures(part.Features()).Add(bakeProfiles(t.uProfiles), bakeProfiles(t.vProfiles))
	part.Recompute()
	s.recordEdit(part, "Network Surface")
	if !t.added.Health().OK() {
		return errors.New("network surface: " + t.added.Health().Reason)
	}
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *NetworkTool) AddedFeature() *feature.PartFeature { return t.added }

// bakeProfiles converts each profile's outer-loop polygon to a model-space polyline.
func bakeProfiles(handles []ProfileHandle) [][]math.Point3 {
	out := make([][]math.Point3, 0, len(handles))
	for _, h := range handles {
		poly := h.Sketch.Profiles().Item(h.ProfileIndex).OuterLoop().Polygon()
		plane := h.Sketch.Plane()
		line := make([]math.Point3, len(poly))
		for i, p := range poly {
			line[i] = plane.ToModel(p)
		}
		out = append(out, line)
	}
	return out
}
