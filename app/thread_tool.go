// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/api/types"
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
	classIdx    int // index into ClassOptions (0 = unspecified)
	tapered     bool
	modelDiaIdx int // index into ModelDiameterOptions (0 = major, the default)
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

// Tapered / SetTapered toggle a tapered (pipe) thread. A cut tapered thread is rejected
// (it needs a conical face), so CanCommit blocks the combination.
func (t *ThreadTool) Tapered() bool     { return t.tapered }
func (t *ThreadTool) SetTapered(v bool) { t.tapered = v }

// ClassOptions lists the tolerance-class choices for the current standard: unspecified
// first, then the standard's external and internal classes (#325).
func (t *ThreadTool) ClassOptions() []string {
	opts := []string{"Unspecified"}
	for _, internal := range []bool{false, true} {
		classes, err := feature.ThreadClasses(string(t.standard()), internal)
		if err == nil {
			opts = append(opts, classes...)
		}
	}
	return opts
}

// ClassIndex / SetClassIndex select the tolerance class within ClassOptions.
func (t *ThreadTool) ClassIndex() int { return t.classIdx }
func (t *ThreadTool) SetClassIndex(i int) {
	t.classIdx = clampRange(i, len(t.ClassOptions()))
}

// class resolves the selected tolerance class ("" when unspecified).
func (t *ThreadTool) class() string {
	if t.classIdx == 0 {
		return ""
	}
	return t.ClassOptions()[clampRange(t.classIdx, len(t.ClassOptions()))]
}

// threadModelDiameters are the model-diameter choices in ModelDiameterOptions order
// (major is the zero default).
var threadModelDiameters = []types.ModelDiameterFromThread{
	types.ThreadMajorDiameter, types.ThreadMinorDiameter,
	types.ThreadPitchDiameter, types.ThreadTapDrillDiameter,
}

// ModelDiameterOptions lists the model-diameter display choices.
func (t *ThreadTool) ModelDiameterOptions() []string {
	return []string{"Major", "Minor", "Pitch", "Tap drill"}
}

// ModelDiameterIndex / SetModelDiameterIndex select which thread diameter the modeled
// face represents.
func (t *ThreadTool) ModelDiameterIndex() int { return t.modelDiaIdx }
func (t *ThreadTool) SetModelDiameterIndex(i int) {
	t.modelDiaIdx = clampRange(i, len(threadModelDiameters))
}

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

// CanCommit reports a face is picked, the pick resolves to a designation, and the
// options are consistent (a cut tapered thread needs a conical face — rejected).
func (t *ThreadTool) CanCommit() bool {
	if t.face == nil || (t.cut && t.tapered) {
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
	t.added = feature.NewDressUpFeatures(part.Features()).AddThreadDef(&feature.ThreadDefinition{
		FaceKey: t.face.Face.ReferenceKey(), Designation: designation, Cut: t.cut,
		Class: t.class(), Tapered: t.tapered, ModelDiameter: threadModelDiameters[clampRange(t.modelDiaIdx, len(threadModelDiameters))],
	})
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

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ThreadTool) AddedFeature() *feature.PartFeature { return t.added }

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

// ClearFace empties the picked cylindrical face — the property panel's selector clear
// (⊗) — returning the tool to its pick-a-face step.
func (t *ThreadTool) ClearFace() { t.face = nil }
