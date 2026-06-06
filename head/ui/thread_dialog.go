//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati/app"
	"oblikovati/head/internal/native"
	"oblikovati/model/feature"
)

// drawThreadDialog shows the Thread tool's property window while the Thread tool is active: pick
// the standard (ISO/ANSI/JIS), then the size and pitch it offers, choose cosmetic vs a modeled
// cut, and OK to thread the picked cylindrical face.
func drawThreadDialog(s *app.Session) {
	t := s.ActiveThread()
	if t == nil {
		return
	}
	native.SetNextWindowSize(340, 300)
	if native.Begin("Thread") {
		drawThreadFaceStatus(t)
		size := drawThreadStandardAndSize(t)
		drawThreadPitch(t, size)
		drawThreadCutAndDesignation(t)
		drawCommitCancelButtons(s, t.CanCommit())
	}
	native.End()
}

func drawThreadFaceStatus(t *app.ThreadTool) {
	if t.HasFace() {
		native.Text("Cylindrical face: selected")
	} else {
		native.Text("Click a cylindrical face to thread.")
	}
	native.Separator()
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
	if native.BeginCombo("Standard", fmt.Sprintf("%s (%s)", std, feature.StandardSystem(std))) {
		for i, st := range standards {
			if native.Selectable(fmt.Sprintf("%s (%s)", st, feature.StandardSystem(st)), i == t.StandardIndex()) {
				t.SetStandardIndex(i)
			}
		}
		native.EndCombo()
	}
}

func drawThreadSizeCombo(t *app.ThreadTool, sizes []feature.ThreadSize, size feature.ThreadSize) {
	if native.BeginCombo("Size", size.Name) {
		for i, sz := range sizes {
			if native.Selectable(sz.Name, i == t.SizeIndex()) {
				t.SetSizeIndex(i)
			}
		}
		native.EndCombo()
	}
}

func drawThreadPitch(t *app.ThreadTool, size feature.ThreadSize) {
	pitch := size.Pitches[clampIdx(t.PitchIndex(), len(size.Pitches))]
	if native.BeginCombo("Pitch", pitchLabel(size, pitch)) {
		for i, p := range size.Pitches {
			if native.Selectable(pitchLabel(size, p), i == t.PitchIndex()) {
				t.SetPitchIndex(i)
			}
		}
		native.EndCombo()
	}
}

func drawThreadCutAndDesignation(t *app.ThreadTool) {
	cut := t.Cut()
	if native.Checkbox("Model real cut thread (else cosmetic)", &cut) {
		t.SetCut(cut)
	}
	native.Separator()
	if d, err := t.Designation(); err == nil {
		native.Text("Designation: " + d)
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
