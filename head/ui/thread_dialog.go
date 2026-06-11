//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/feature"
)

// The Thread flow in the head: while the Thread tool runs, a modeless property panel
// (the reference panel schema) drives the tool — the cylindrical-face chip, the
// standard/size/pitch designation in a Type section, and the cosmetic-vs-cut toggle —
// then OK to thread the picked face.
func drawThreadDialog(s *app.Session) {
	t := s.ActiveThread()
	if t == nil {
		return
	}
	native.SetNextWindowSizeOnce(360, 360)
	if native.Begin("Thread") {
		drawFeatureBreadcrumb("Thread", "")
		drawThreadInputGeometry(t)
		drawThreadType(t)
		drawThreadBehavior(t)
		native.Separator()
		drawCommitCancelButtons(s, t.CanCommit())
	}
	native.End()
}

// drawThreadInputGeometry is the Input Geometry section: the required cylindrical-face chip.
func drawThreadInputGeometry(t *app.ThreadTool) {
	if !propertySection("Input Geometry") {
		return
	}
	drawPickChipRow("Face", "thread-face", pickChipText(t.HasFace(), "1 Face", "Select Face"),
		t.HasFace(), "Click a cylindrical face in the viewport to thread", t.ClearFace)
}

// drawThreadType is the Type section: the standard / size / pitch combos and the
// resulting designation row.
func drawThreadType(t *app.ThreadTool) {
	if !propertySection("Type") {
		return
	}
	size := drawThreadStandardAndSize(t)
	drawThreadPitch(t, size)
	if d, err := t.Designation(); err == nil {
		propertyRow("Designation")
		native.Text(d)
	}
}

func drawThreadStandardAndSize(t *app.ThreadTool) feature.ThreadSize {
	standards := feature.ThreadStandards()
	std := standards[clampIdx(t.StandardIndex(), len(standards))]
	drawThreadStandardCombo(t, standards, std)
	std = standards[clampIdx(t.StandardIndex(), len(standards))]
	sizes := feature.ThreadSizes(std)
	size := sizes[clampIdx(t.SizeIndex(), len(sizes))]
	drawThreadSizeCombo(t, sizes, size)
	return sizes[clampIdx(t.SizeIndex(), len(sizes))]
}

func drawThreadStandardCombo(t *app.ThreadTool, standards []feature.ThreadStandard, std feature.ThreadStandard) {
	propertyRow("Standard")
	native.SetNextItemWidth(propertyComboWidth)
	if native.BeginCombo("##thread-standard", fmt.Sprintf("%s (%s)", std, feature.StandardSystem(std))) {
		for i, st := range standards {
			if native.Selectable(fmt.Sprintf("%s (%s)", st, feature.StandardSystem(st)), i == t.StandardIndex()) {
				t.SetStandardIndex(i)
			}
		}
		native.EndCombo()
	}
}

func drawThreadSizeCombo(t *app.ThreadTool, sizes []feature.ThreadSize, size feature.ThreadSize) {
	propertyRow("Size")
	native.SetNextItemWidth(propertyComboWidth)
	if native.BeginCombo("##thread-size", size.Name) {
		for i, sz := range sizes {
			if native.Selectable(sz.Name, i == t.SizeIndex()) {
				t.SetSizeIndex(i)
			}
		}
		native.EndCombo()
	}
}

func drawThreadPitch(t *app.ThreadTool, size feature.ThreadSize) {
	propertyRow("Pitch")
	native.SetNextItemWidth(propertyComboWidth)
	pitch := size.Pitches[clampIdx(t.PitchIndex(), len(size.Pitches))]
	if native.BeginCombo("##thread-pitch", pitchLabel(size, pitch)) {
		for i, p := range size.Pitches {
			if native.Selectable(pitchLabel(size, p), i == t.PitchIndex()) {
				t.SetPitchIndex(i)
			}
		}
		native.EndCombo()
	}
}

// drawThreadBehavior is the Behavior section: the cosmetic-vs-modeled-cut toggle.
func drawThreadBehavior(t *app.ThreadTool) {
	if !propertySection("Behavior") {
		return
	}
	propertyRow("")
	cut := t.Cut()
	if native.Checkbox("Model real cut thread (else cosmetic)", &cut) {
		t.SetCut(cut)
	}
}

// pitchLabel formats a pitch for the combo: "1.25 mm" (metric) or "20 TPI" (imperial).
func pitchLabel(size feature.ThreadSize, pitch float64) string {
	if size.System == feature.SystemImperial {
		return fmt.Sprintf("%d TPI", int(25.4/pitch+0.5))
	}
	return fmt.Sprintf("%.4g mm", pitch)
}

// clampIdx keeps an index within [0, n).
func clampIdx(i, n int) int {
	if n == 0 || i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}
