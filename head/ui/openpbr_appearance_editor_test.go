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

// TestInWindowOpenPBRTabRendersAllGroups mirrors TestInWindowMaterialsWindowDrawsAllTabs
// for the OpenPBR tab (M45-F05 PBI-351): opens the real window with the Materials
// window on the OpenPBR tab and runs frames, so a mismatched ImGui Begin/End across any
// of the nine stacked parameter groups (Base/Specular/Transmission/Subsurface/Coat/
// Fuzz/ThinFilm/Emission/Geometry — all drawn unconditionally every frame, not gated
// behind a nested tab selection) would trip Dear ImGui's assertions. It just needs to
// run without aborting — this IS the "every group's tab renders without error" the PBI
// asks for, since every group draws on every frame regardless of which top-level
// Materials-window tab is active.
func TestInWindowOpenPBRTabRendersAllGroups(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()

	if len(s.Materials().OpenPBRAppearances()) == 0 {
		t.Fatal("OpenPBR appearance library empty — nothing for the editor to render")
	}
	selectedOpenPBR = s.Materials().OpenPBRAppearances()[0].ID()

	showMaterials = true
	defer func() { showMaterials = false }()

	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}

// TestOpenPBREditsPersistThroughProjectStore is PBI-351's second acceptance criterion:
// editing a non-built-in OpenPBRAppearance through the exact session call the editor
// uses (UpdateOpenPBRAppearance) must persist and round-trip through the project YAML
// store — verified with a REAL material.Store attached (not just in-memory library
// state), reloaded into a fresh session to prove the edit actually reached disk.
func TestOpenPBREditsPersistThroughProjectStore(t *testing.T) {
	fs := newFakeMaterialFS()
	store := material.NewStore("/proj/DesignData", fs)

	s := framedSession()
	if err := s.LoadProjectMaterials(store); err != nil {
		t.Fatalf("LoadProjectMaterials: %v", err)
	}
	base := s.Materials().OpenPBRAppearances()[0]
	dup, err := s.DuplicateOpenPBRAppearance(base.ID(), "Editor Test Copy")
	if err != nil {
		t.Fatalf("DuplicateOpenPBRAppearance: %v", err)
	}

	spec := dup.Spec()
	spec.Base.Metalness = 0.73
	spec.Coat.Weight = 1
	s.UpdateOpenPBRAppearance(dup.ID(), spec) // editOpenPBRAppearance's exact persistence call

	// A second session, sharing only the store (not the in-memory library), must see
	// the edit — proving it reached the YAML file, not just this session's Library.
	s2 := framedSession()
	if err := s2.LoadProjectMaterials(store); err != nil {
		t.Fatalf("LoadProjectMaterials (second session): %v", err)
	}
	got, ok := s2.Materials().OpenPBRAppearance(dup.ID())
	if !ok {
		t.Fatal("edited appearance did not round-trip through the project store")
	}
	if got.Base().Metalness != 0.73 || got.Coat().Weight != 1 {
		t.Errorf("persisted edit lost data: base metalness=%v coat weight=%v, want 0.73/1", got.Base().Metalness, got.Coat().Weight)
	}
}
