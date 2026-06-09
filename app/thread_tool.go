// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/model/feature"
)

// ThreadTool is the interactive Thread command (3D Model ▸ Modify): activate it, click a
// cylindrical face, choose the standard / size / pitch and cosmetic-vs-cut in the property
// window, and OK to thread the face. The standard/size/pitch come from the thread catalog
// (feature.ThreadStandards / ThreadSizes); the pick resolves to a designation the thread
// feature applies.
type ThreadTool struct {
	face        *FaceHandle
	standardIdx int
	sizeIdx     int
	pitchIdx    int
	cut         bool
	added       *feature.PartFeature
}

// NewThreadTool returns a thread tool defaulting to the first standard/size/pitch, cosmetic.
func NewThreadTool() *ThreadTool { return &ThreadTool{} }

// Name implements [Tool].
func (t *ThreadTool) Name() string { return "Thread" }

// Start sets the selection filter to faces.
func (t *ThreadTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectFace)) }

// Pick accepts a clicked cylindrical face (ignoring non-cylindrical picks).
func (t *ThreadTool) Pick(_ *Session, sel Selectable) {
	h, ok := sel.(FaceHandle)
	if !ok {
		return
	}
	if _, isCyl := h.Face.Geometry().(geom.Cylinder); !isCyl {
		return
	}
	hh := h
	t.face = &hh
}

// HasFace reports whether a cylindrical face is picked.
func (t *ThreadTool) HasFace() bool { return t.face != nil }

// Standard accessors (by index into feature.ThreadStandards).
func (t *ThreadTool) StandardIndex() int { return t.standardIdx }

func (t *ThreadTool) SetStandardIndex(i int) {
	if i != t.standardIdx {
		t.standardIdx, t.sizeIdx, t.pitchIdx = i, 0, 0
	}
}

// SizeIndex / SetSizeIndex select the size within the current standard.
func (t *ThreadTool) SizeIndex() int { return t.sizeIdx }

func (t *ThreadTool) SetSizeIndex(i int) {
	if i != t.sizeIdx {
		t.sizeIdx, t.pitchIdx = i, 0
	}
}

// PitchIndex / SetPitchIndex select the pitch within the current size.
func (t *ThreadTool) PitchIndex() int     { return t.pitchIdx }
func (t *ThreadTool) SetPitchIndex(i int) { t.pitchIdx = i }

// Cut / SetCut toggle a modeled cut thread (vs cosmetic).
func (t *ThreadTool) Cut() bool       { return t.cut }
func (t *ThreadTool) SetCut(cut bool) { t.cut = cut }

// standard / size resolve the current selection, clamped to valid ranges.
func (t *ThreadTool) standard() feature.ThreadStandard {
	stds := feature.ThreadStandards()
	t.standardIdx = clampRange(t.standardIdx, len(stds))
	return stds[t.standardIdx]
}

func (t *ThreadTool) size() feature.ThreadSize {
	sizes := feature.ThreadSizes(t.standard())
	t.sizeIdx = clampRange(t.sizeIdx, len(sizes))
	return sizes[t.sizeIdx]
}

func (t *ThreadTool) pitch() float64 {
	ps := t.size().Pitches
	t.pitchIdx = clampRange(t.pitchIdx, len(ps))
	return ps[t.pitchIdx]
}

// Designation returns the parseable designation for the current pick.
func (t *ThreadTool) Designation() (string, error) {
	return feature.ThreadDesignation(t.standard(), t.size().Name, t.pitch())
}

// CanCommit reports a face is picked and the pick resolves to a designation.
func (t *ThreadTool) CanCommit() bool {
	if t.face == nil {
		return false
	}
	_, err := t.Designation()
	return err == nil
}

// Commit threads the picked face on the active part and recomputes; a sick feature keeps the
// tool open by returning an error.
func (t *ThreadTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if t.face == nil {
		return errors.New("thread: click a cylindrical face first")
	}
	designation, err := t.Designation()
	if err != nil {
		return err
	}
	t.added = feature.NewDressUpFeatures(part.Features()).AddThread(t.face.Face.ReferenceKey(), designation, t.cut)
	part.Recompute()
	s.recordEdit(part, "Thread")
	if !t.added.Health().OK() {
		return errors.New("thread: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// Cancel abandons the tool.
func (t *ThreadTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// clampRange keeps an index within [0, n).
func clampRange(i, n int) int {
	if n == 0 {
		return 0
	}
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}
