// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
)

// Interactive free-form CAGE editing — the T-spline-style gesture the sub-D primitives exist to
// serve. With the tool active the running free-form body's cage is drawn as draggable vertex
// handles; a left-drag on one slides it in the camera-facing plane and re-subdivides live, and
// the panel sets the subdivision level and creases the edges around the picked vertices.
//
// FreeformBody.SetLevel / MoveVertices / CreaseEdges and the freeform.* wire methods shipped with
// #699, whose problem statement said "no wire methods, no client group, AND NO UI" — but the UI
// was left out of its scope section and never built, so a user could place a Box, Plane or Quad
// Ball and then not touch its cage (#2048).
//
// Mirrors ControlPointEditTool, which does the same for a NURBS control net: drag-driven, no OK,
// one undo step per drag.

// cageHandlePixels is the on-screen size of a cage-vertex handle; cagePickPixels its hit radius.
const (
	cageHandlePixels = 7
	cagePickPixels   = 12
)

// cageHandleColor / cageWireColor style the cage overlay (cyan handles, dim wire) — distinct
// from the control net's orange so the two editing modes never look alike.
var (
	cageHandleColor = [4]float32{0.2, 0.8, 1, 1}
	cageWireColor   = [4]float32{0.2, 0.8, 1, 0.5}
)

// FreeformCageEditTool arms interactive cage editing of the running free-form body. It is
// drag-driven (no clicks/OK of its own); its presence lets the viewport route a left-drag to
// moving a cage vertex, and it draws the cage via [FreeformCageEditTool.Preview].
type FreeformCageEditTool struct {
	dialogTool
	level      int     // subdivision level applied to the body
	sharpness  float64 // crease sharpness the Crease action applies (0 smooth … 1 fully sharp)
	lastVertex int     // the cage vertex last grabbed, -1 until a handle is dragged
}

// NewFreeformCageEditTool returns the cage-edit tool defaulting to a fully sharp crease.
func NewFreeformCageEditTool() *FreeformCageEditTool {
	return &FreeformCageEditTool{sharpness: 1, lastVertex: -1}
}

// Name implements [Tool].
func (t *FreeformCageEditTool) Name() string { return "Edit Freeform Cage" }

// Start seeds the level from the body so the panel opens showing what the body actually has.
func (t *FreeformCageEditTool) Start(s *Session) {
	if body, ok := activeFreeformBody(s); ok {
		t.level = body.Level()
	}
}

// LastVertex is the cage vertex last grabbed, or -1 before any drag — what the Crease action
// applies to, so creasing is expressed in terms of the handle the user just moved.
func (t *FreeformCageEditTool) LastVertex() int { return t.lastVertex }

// Prompt guides the input.
func (t *FreeformCageEditTool) Prompt(*Session) string {
	return "Drag a cage handle to shape the body; set the subdivision level, or crease the edges around a dragged vertex."
}

// CanCommit is false — the tool commits per drag, not via OK.
func (t *FreeformCageEditTool) CanCommit() bool { return false }

// Commit is a no-op (drag-driven).
func (t *FreeformCageEditTool) Commit(*Session) error { return nil }

// Level / SetLevel expose the subdivision level; setting it applies to the body immediately, the
// way the level slider is expected to behave.
func (t *FreeformCageEditTool) Level() int { return t.level }
func (t *FreeformCageEditTool) SetLevel(n int) {
	if n < 0 {
		n = 0
	}
	t.level = n
}

// Sharpness / SetSharpness hold the crease sharpness, clamped to the 0..1 the model accepts.
func (t *FreeformCageEditTool) Sharpness() float64 { return t.sharpness }
func (t *FreeformCageEditTool) SetSharpness(v float64) {
	t.sharpness = math.Clamp(v, 0, 1)
}

// Preview draws the editable cage (handles + wire) over the running free-form body.
func (t *FreeformCageEditTool) Preview(s *Session) []renderer.DrawItem {
	body, ok := activeFreeformBody(s)
	if !ok {
		return nil
	}
	return cageOverlayItems(body, s.Camera().WorldPerPixel()*cageHandlePixels)
}

// cageCarrier is any feature that owns a free-form cage — the sub-D primitives and the imported
// Alias free-form, which embeds the same feature rather than being it.
type cageCarrier interface {
	FreeformBody() *feature.FreeformBody
}

// activeFreeformCage returns the running part's last free-form feature and its cage — the one
// the cage tool edits — or ok=false when the part has none.
//
// The FEATURE matters as much as the body: a cage mutation is invisible to a bare Recompute,
// which serves the feature's cached output, so every edit path has to mark it dirty first. That
// is what CommitFeatureEdit does for the wire handlers, and skipping it made a level change land
// on the model and change nothing on screen.
func activeFreeformCage(s *Session) (*feature.PartFeature, *feature.FreeformBody, bool) {
	part, err := activePart(s)
	if err != nil {
		return nil, nil, false
	}
	feats := part.Features()
	for i := feats.Count() - 1; i >= 0; i-- {
		pf := feats.Item(i)
		if ff, ok := pf.Definition().(cageCarrier); ok {
			return pf, ff.FreeformBody(), true
		}
	}
	return nil, nil, false
}

// activeFreeformBody is activeFreeformCage for the callers that only draw the cage.
func activeFreeformBody(s *Session) (*feature.FreeformBody, bool) {
	_, body, ok := activeFreeformCage(s)
	return body, ok
}
