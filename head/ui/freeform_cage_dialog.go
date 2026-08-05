//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The free-form cage editor's panel: the subdivision level, the crease sharpness, and the Crease
// action that sharpens the edges around the handle just dragged. The tool has no OK — each drag
// is its own undo step — so the panel shows Close rather than a commit bar (#2048).

// cageEditSession is what the cage panel's rows need: apply the level, crease at the handle, and
// close the tool. A consumer-side interface rather than the whole *app.Session (audit I5).
type cageEditSession interface {
	ApplyActiveCageLevel() bool
	CreaseActiveCageHandle() bool
}

var _ cageEditSession = (*app.Session)(nil)

// cageEditUI holds the panel's fields across frames.
var cageEditUI = struct {
	level     int32
	sharpness float32
	seeded    *app.FreeformCageEditTool
}{sharpness: 1}

// drawFreeformCageDialog shows the cage editor while the tool is active.
func drawFreeformCageDialog(s *app.Session) {
	c := s.ActiveCageEdit()
	if c == nil {
		cageEditUI.seeded = nil
		return
	}
	if cageEditUI.seeded != c {
		cageEditUI.level, cageEditUI.sharpness = int32(c.Level()), float32(c.Sharpness())
		cageEditUI.seeded = c
	}
	dialogSizeOnce(340, 230)
	if native.Begin("Edit Freeform Cage") {
		drawFeatureBreadcrumb("Edit Freeform Cage", "")
		native.TextWrapped("Drag a cage handle to shape the body.")
		if propertySection("Behavior") {
			drawCageLevelRow(s, c)
			drawCageCreaseRows(s, c)
		}
		native.Separator()
		if native.Button("Close") {
			s.CancelTool()
		}
	}
	native.End()
}

// drawCageLevelRow renders the subdivision-level field, applying a change to the body at once —
// the level is what the cage is FOR, so it takes effect as it is typed rather than on an OK.
func drawCageLevelRow(s cageEditSession, c *app.FreeformCageEditTool) {
	propertyRow("Subdivision Level")
	native.SetNextItemWidth(propertyFieldWidth)
	if native.InputInt("##cage-level", &cageEditUI.level) {
		c.SetLevel(int(cageEditUI.level))
		s.ApplyActiveCageLevel()
	}
}

// drawCageCreaseRows renders the crease sharpness and the action that applies it around the last
// dragged handle, disabled until one has been dragged.
func drawCageCreaseRows(s cageEditSession, c *app.FreeformCageEditTool) {
	propertyRow("Crease Sharpness")
	native.SetNextItemWidth(propertyFieldWidth)
	if native.SliderFloat("##cage-sharpness", &cageEditUI.sharpness, 0, 1) {
		c.SetSharpness(float64(cageEditUI.sharpness))
	}
	propertyRow("")
	native.BeginDisabled(c.LastVertex() < 0)
	if native.Button("Crease edges at handle") {
		s.CreaseActiveCageHandle()
	}
	native.EndDisabled()
}
