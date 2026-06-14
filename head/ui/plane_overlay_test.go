//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// showAllDatums makes every origin plane and axis visible — the Origin-folder toggle a user flips to
// see the coordinate frame (they default hidden in a part and an assembly alike).
func showAllDatums(wg *feature.WorkGeometry) {
	planes := wg.WorkPlanes()
	for i := 0; i < planes.Count(); i++ {
		planes.Item(i).SetVisible(true)
	}
	axes := wg.WorkAxes()
	for i := 0; i < axes.Count(); i++ {
		axes.Item(i).SetVisible(true)
	}
}

// TestAssemblyOriginDatumsDrawWhenShown: the datum overlays draw an ASSEMBLY's origin planes and
// axes once shown — the viewport sources them through ActiveWorkGeometry, so an assembly gets the
// origin frame a part has (before, the overlays were part-gated and an assembly drew none).
func TestAssemblyOriginDatumsDrawWhenShown(t *testing.T) {
	s := app.NewSession()
	asmDoc, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(asmDoc); err != nil {
		t.Fatalf("activate assembly: %v", err)
	}
	wg, ok := s.ActiveWorkGeometry()
	if !ok {
		t.Fatal("ActiveWorkGeometry returned false for an active assembly")
	}
	showAllDatums(wg)
	noHide := func(uint64) bool { return false }

	if got := len(planesOverlay(wg.WorkPlanes(), nil, nil, noHide)); got == 0 {
		t.Error("planesOverlay drew nothing for an assembly's visible origin planes")
	}
	if got := len(axesOverlay(wg.WorkAxes(), nil, noHide)); got == 0 {
		t.Error("axesOverlay drew nothing for an assembly's visible origin axes")
	}
}
