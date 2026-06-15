// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestRepresentationsPanelExposesCommands: the Assemble tab's Representations panel exposes the
// capture + model-state commands as compact icon buttons (M12-F04).
func TestRepresentationsPanelExposesCommands(t *testing.T) {
	tab, ok := BuildRibbon(assemblySession(t)).Tab("Assemble")
	if !ok {
		t.Fatal("an active assembly should show the Assemble tab")
	}
	panel, ok := tab.Panel("Representations")
	if !ok {
		t.Fatal("Assemble tab has no Representations panel")
	}
	for _, name := range []string{"Capture View", "Capture Position", "Capture LOD", "Model State"} {
		if !hasButton(panel, name) {
			t.Errorf("Representations panel is missing the %q command", name)
		}
		if got, ok := styleOf(panel, name); !ok || got != CompactIconButton {
			t.Errorf("%q button style = %v, want CompactIconButton", name, got)
		}
	}
}

// TestRepresentationCaptureActivateAndBrowser: capturing an LOD representation lists it under
// the browser's Representations folder, and activating it re-applies its suppression (M12-F04).
func TestRepresentationCaptureActivateAndBrowser(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	placedWidget(t, s, asm, "widget:2")
	b := asm.Occurrences().Item(1)

	b.SetSuppressed(true)
	if err := s.CaptureLOD(); err != nil { // captures b suppressed
		t.Fatalf("CaptureLOD: %v", err)
	}
	b.SetSuppressed(false)

	folder := findBrowserNode(BuildBrowser(s), "representations", "Representations")
	if folder == nil || len(folder.Children) != 1 {
		t.Fatalf("Representations folder = %v, want one LOD row", folder)
	}
	h, ok := folder.Children[0].Select.(RepresentationHandle)
	if !ok || h.Family != types.RepresentationLevelOfDetail {
		t.Fatalf("row selects %T (%+v), want a LOD RepresentationHandle", folder.Children[0].Select, h)
	}

	if err := s.ActivateRepresentation(h); err != nil {
		t.Fatalf("ActivateRepresentation: %v", err)
	}
	if !b.Suppressed() {
		t.Error("activating the LOD representation did not re-suppress b")
	}
}

// TestModelStateCaptureAndActivate: a model state from the active representations appears under
// the Model States folder and re-applies its families when activated (M12-F04).
func TestModelStateCaptureAndActivate(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	a := asm.Occurrences().Item(0)

	a.SetSuppressed(true)
	if err := s.CaptureLOD(); err != nil {
		t.Fatalf("CaptureLOD: %v", err)
	}
	a.SetSuppressed(false)
	// Make the LOD rep active so the new model state selects it.
	if _, err := asm.Representations().ActivateLOD(asm.Representations().AllLODs()[0].ID()); err != nil {
		t.Fatalf("ActivateLOD: %v", err)
	}
	if err := s.NewModelState(); err != nil {
		t.Fatalf("NewModelState: %v", err)
	}

	folder := findBrowserNode(BuildBrowser(s), "modelStates", "Model States")
	if folder == nil || len(folder.Children) != 1 {
		t.Fatalf("Model States folder = %v, want one row", folder)
	}
	h, ok := folder.Children[0].Select.(ModelStateHandle)
	if !ok {
		t.Fatalf("row selects %T, want a ModelStateHandle", folder.Children[0].Select)
	}

	a.SetSuppressed(false)
	if err := s.ActivateModelState(h); err != nil {
		t.Fatalf("ActivateModelState: %v", err)
	}
	if !a.Suppressed() {
		t.Error("activating the model state did not re-apply the LOD representation")
	}
}
