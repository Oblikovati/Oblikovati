//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The Center Point had a Visible flag and a browser toggle to come, but nothing ever drew it —
// showing it changed nothing on screen, which is what made it impossible to pick in the 3D view
// and project (#2016). These hold the overlay to the same visibility rule the axes follow.

// partWorkGeometry returns a fresh part's work geometry.
func partWorkGeometry(t *testing.T) (*app.Session, *feature.WorkGeometry) {
	t.Helper()
	s := app.NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "Part1", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	wg, ok := s.ActiveWorkGeometry()
	if !ok {
		t.Fatal("ActiveWorkGeometry returned false for an active part")
	}
	return s, wg
}

func TestDatumPointOverlayDrawsOnlyShownPoints(t *testing.T) {
	_, wg := partWorkGeometry(t)
	noHide := func(uint64) bool { return false }

	if items := pointsDatumOverlay(wg.WorkPoints(), nil, noHide, 1); len(items) != 0 {
		t.Errorf("overlay drew %d items with every datum point hidden, want 0", len(items))
	}

	center, ok := wg.WorkPointByRef(feature.OriginCenter)
	if !ok {
		t.Fatal("no origin centre point")
	}
	center.SetVisible(true)

	items := pointsDatumOverlay(wg.WorkPoints(), nil, noHide, 1)
	if len(items) != 1 {
		t.Fatalf("overlay drew %d items for one shown point, want 1", len(items))
	}
	// Three axis-aligned segments through the point: six vertices, three lines.
	if got := len(items[0].Positions); got != 6 {
		t.Errorf("cross has %d vertices, want 6 (three segments)", got)
	}
	if got := len(items[0].Indices); got != 6 {
		t.Errorf("cross has %d indices, want 6", got)
	}
}

// The cross is centred on the point and scales with the caller's world size, so it stays
// screen-constant as the camera zooms.
func TestDatumPointCrossIsCentredAndScaled(t *testing.T) {
	_, wg := partWorkGeometry(t)
	center, _ := wg.WorkPointByRef(feature.OriginCenter)
	center.SetVisible(true)
	noHide := func(uint64) bool { return false }

	const h = 2.5
	items := pointsDatumOverlay(wg.WorkPoints(), nil, noHide, h)
	pos := items[0].Positions
	c := center.Point()
	if pos[0].X != c.X-h || pos[1].X != c.X+h {
		t.Errorf("X arm spans %v..%v, want %v..%v", pos[0].X, pos[1].X, c.X-h, c.X+h)
	}
	if pos[2].Y != c.Y-h || pos[3].Y != c.Y+h {
		t.Errorf("Y arm spans %v..%v, want %v..%v", pos[2].Y, pos[3].Y, c.Y-h, c.Y+h)
	}
	if pos[4].Z != c.Z-h || pos[5].Z != c.Z+h {
		t.Errorf("Z arm spans %v..%v, want %v..%v", pos[4].Z, pos[5].Z, c.Z-h, c.Z+h)
	}
}

// A selected datum point is highlighted, the way a selected axis and plane are.
func TestDatumPointOverlayHighlightsSelection(t *testing.T) {
	_, wg := partWorkGeometry(t)
	center, _ := wg.WorkPointByRef(feature.OriginCenter)
	center.SetVisible(true)
	noHide := func(uint64) bool { return false }

	plain := pointsDatumOverlay(wg.WorkPoints(), nil, noHide, 1)[0].Color
	picked := pointsDatumOverlay(wg.WorkPoints(), center, noHide, 1)[0].Color
	if plain == picked {
		t.Error("a selected datum point draws in the same colour as an unselected one")
	}
}

// An edit scope hides datums created after the node being edited; a point obeys it like the rest.
func TestDatumPointOverlayObeysEditScope(t *testing.T) {
	_, wg := partWorkGeometry(t)
	center, _ := wg.WorkPointByRef(feature.OriginCenter)
	center.SetVisible(true)

	hideAll := func(uint64) bool { return true }
	if items := pointsDatumOverlay(wg.WorkPoints(), nil, hideAll, 1); len(items) != 0 {
		t.Errorf("overlay drew %d items for a scope-hidden point, want 0", len(items))
	}
}
