// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/model/feature"
)

// DecalTool is the interactive Decal command (Create panel): click a face, give an image
// resource, and OK to place a decal on it. A decal is cosmetic — it does not change the solid —
// so it cannot fail; the image is the resource id/path the renderer projects onto the face.
type DecalTool struct {
	face  *FaceHandle
	image string
	added *feature.DecalFeature
}

// NewDecalTool returns a decal tool.
func NewDecalTool() *DecalTool { return &DecalTool{} }

// Name implements [Tool].
func (t *DecalTool) Name() string { return "Decal" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *DecalTool) Start(*Session) {}

// AcceptedKinds declares decal picks a face (the surface to project onto).
func (t *DecalTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectFace} }

// Picks reports the picked face for the unified highlight.
func (t *DecalTool) Picks() []Selectable {
	if t.face == nil {
		return nil
	}
	return []Selectable{*t.face}
}

// Pick captures the target face.
func (t *DecalTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok {
		fc := f
		t.face = &fc
	}
}

// SetImage/Image drive the decal's image resource.
func (t *DecalTool) SetImage(s string) { t.image = s }
func (t *DecalTool) Image() string     { return t.image }

// Params exposes the image resource for the generic property dialog.
func (t *DecalTool) Params() ToolParams {
	return ToolParams{Texts: []TextParam{{Label: "Image", Get: t.Image, Set: t.SetImage}}}
}

// PickedFace returns the target face (if any) for the unified tool highlight.
func (t *DecalTool) PickedFace() (FaceHandle, bool) {
	if t.face == nil {
		return FaceHandle{}, false
	}
	return *t.face, true
}

// CanCommit reports whether a face is picked and an image is set.
func (t *DecalTool) CanCommit() bool { return t.face != nil && t.image != "" }

// Commit places the decal on the picked face and recomputes.
func (t *DecalTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewCosmeticFeatures(part.Features()).AddDecal(t.face.Face.ReferenceKey(), t.image)
	part.Recompute()
	s.recordEdit(part, "Decal")
	return nil
}

// Cancel is a no-op; the engine restores the ambient filter.
func (t *DecalTool) Cancel(*Session) {}

// AddedFeature returns the decal created on commit (for inspection/tests).
func (t *DecalTool) AddedFeature() *feature.DecalFeature { return t.added }
