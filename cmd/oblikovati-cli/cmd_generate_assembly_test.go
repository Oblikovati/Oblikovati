// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"oblikovati.org/model/benchgen"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// cliTinyProfile is a small valid profile for fast CLI tests (the built-in auto30k is
// too heavy to write to disk in a unit test).
func cliTinyProfile() benchgen.Profile {
	return benchgen.Profile{
		Name: "clitiny", Systems: 2, Modules: 1, SubModules: 1, Bays: 2,
		Tiers: []benchgen.TierSpec{
			{Tier: benchgen.Fastener, UniqueMeshes: 2, Placements: 8, Sides: 6, RadiusCm: 0.4, HeightCm: 1.2},
			{Tier: benchgen.Bracket, UniqueMeshes: 4, Placements: 8, Sides: 12, RadiusCm: 2.0, HeightCm: 0.4},
			{Tier: benchgen.Machined, UniqueMeshes: 4, Placements: 4, Sides: 16, RadiusCm: 3.0, HeightCm: 4.0},
			{Tier: benchgen.System, UniqueMeshes: 2, Placements: 2, Sides: 16, RadiusCm: 6.0, HeightCm: 8.0},
		},
	}
}

func TestGenerateAssemblyWritesAndReopens(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "tinycar")
	build := doc.NewWorkspace(persistence.NewPackageStore())
	root, _, err := benchgen.Generate(build, prefix, cliTinyProfile())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := saveWorkspace(build); err != nil {
		t.Fatalf("saveWorkspace: %v", err)
	}

	// The saved set must reopen through a fresh workspace+store — exactly what the head
	// does — resolving the assembly's references to its part/sub-assembly files and
	// recomputing geometry, proving the fixture is loadable (not just generated).
	rootPath := root.FullFileName()
	ws := doc.NewWorkspace(persistence.NewPackageStore())
	reopened, err := ws.Open(rootPath, false)
	if err != nil {
		t.Fatalf("reopen %q: %v", rootPath, err)
	}
	asm, ok := reopened.Content().(*compdef.AssemblyComponentDefinition)
	if !ok {
		t.Fatalf("reopened content is %T, want *AssemblyComponentDefinition", reopened.Content())
	}
	if got := len(asm.PlacedBodies()); got == 0 {
		t.Error("reopened assembly flattened to zero placed bodies")
	}
}

func TestGenerateAssemblyArgErrors(t *testing.T) {
	var buf bytes.Buffer
	if err := cmdGenerateAssembly([]string{"--profile", "auto30k"}, &buf); err == nil {
		t.Error("expected error when --out is missing")
	}
	if err := cmdGenerateAssembly([]string{"--profile", "nope", "--out", t.TempDir()}, &buf); err == nil {
		t.Error("expected error for unknown profile")
	}
}
