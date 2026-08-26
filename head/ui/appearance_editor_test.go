//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"errors"
	"testing"

	"oblikovati.org/model/material"
)

// fakeMaterialFS is an in-memory material.FileSystem — CLAUDE.md's "named fake classes,
// not inline stubs" for mocking file I/O; model/material's own equivalent fake is
// package-private, so this package needs its own small one to attach a real Store.
type fakeMaterialFS struct{ files map[string][]byte }

func newFakeMaterialFS() *fakeMaterialFS { return &fakeMaterialFS{files: map[string][]byte{}} }

var errFakeMaterialFSNotFound = errors.New("fakeMaterialFS: no such file")

func (f *fakeMaterialFS) ReadFile(path string) ([]byte, error) {
	data, ok := f.files[path]
	if !ok {
		return nil, errFakeMaterialFSNotFound
	}
	return data, nil
}

func (f *fakeMaterialFS) WriteFile(path string, data []byte) error {
	f.files[path] = append([]byte(nil), data...)
	return nil
}

// TestInWindowAppearanceTabRendersAllGroups mirrors TestInWindowMaterialsWindowDrawsAllTabs
// for the Appearance tab (M45-F05 PBI-351): opens the real window with the Materials
// window on the Appearance tab and runs frames, so a mismatched ImGui Begin/End across any
// of the nine stacked parameter groups (Base/Specular/Transmission/Subsurface/Coat/
// Fuzz/ThinFilm/Emission/Geometry — all drawn unconditionally every frame, not gated
// behind a nested tab selection) would trip Dear ImGui's assertions. It just needs to
// run without aborting — this IS the "every group's tab renders without error" the PBI
// asks for, since every group draws on every frame regardless of which top-level
// Materials-window tab is active.
func TestInWindowAppearanceTabRendersAllGroups(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()

	if len(s.Materials().Appearances()) == 0 {
		t.Fatal("appearance library empty — nothing for the editor to render")
	}
	selectedAppearance = s.Materials().Appearances()[0].ID()

	showMaterials = true
	defer func() { showMaterials = false }()

	for range 3 {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}

// TestAppearanceEditsPersistThroughProjectStore is PBI-351's second acceptance criterion:
// editing a non-built-in Appearance through the exact session call the editor uses
// (UpdateAppearance) must persist and round-trip through the project YAML store —
// verified with a REAL material.Store attached (not just in-memory library state),
// reloaded into a fresh session to prove the edit actually reached disk.
func TestAppearanceEditsPersistThroughProjectStore(t *testing.T) {
	fs := newFakeMaterialFS()
	store := material.NewStore("/proj/DesignData", fs)

	s := framedSession()
	if err := s.LoadProjectMaterials(store); err != nil {
		t.Fatalf("LoadProjectMaterials: %v", err)
	}
	base := s.Materials().Appearances()[0]
	dup, err := s.DuplicateAppearance(base.ID(), "Editor Test Copy")
	if err != nil {
		t.Fatalf("DuplicateAppearance: %v", err)
	}

	spec := dup.Spec()
	spec.Base.Metalness = 0.73
	spec.Coat.Weight = 1
	s.UpdateAppearance(dup.ID(), spec) // editAppearancePBR's exact persistence call

	// A second session, sharing only the store (not the in-memory library), must see
	// the edit — proving it reached the YAML file, not just this session's Library.
	s2 := framedSession()
	if err := s2.LoadProjectMaterials(store); err != nil {
		t.Fatalf("LoadProjectMaterials (second session): %v", err)
	}
	got, ok := s2.Materials().Appearance(dup.ID())
	if !ok {
		t.Fatal("edited appearance did not round-trip through the project store")
	}
	if got.Base().Metalness != 0.73 || got.Coat().Weight != 1 {
		t.Errorf("persisted edit lost data: base metalness=%v coat weight=%v, want 0.73/1", got.Base().Metalness, got.Coat().Weight)
	}
}
