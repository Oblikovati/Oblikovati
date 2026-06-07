// SPDX-License-Identifier: GPL-2.0-only

// Package examples bundles ready-to-run Lua programs that drive the model through the
// public wire API (ADR-0028). They double as a starter script library and as the corpus
// for the scripting integration tests: each program is self-contained (it creates its own
// document), so it runs unchanged from the GUI console, the CLI, or scripts.run.
//
// The programs are clean-room ports of common parametric-modelling scenarios (create
// parameters, sketch + extrude, revolve, read physical properties), expressed against this
// project's own API.
package examples

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed *.lua
var fsys embed.FS

// Names returns the bundled example filenames (e.g. "extrude_block.lua"), sorted.
func Names() ([]string, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// Source returns the Lua source of a named example (e.g. examples.Source("extrude_block.lua")).
func Source(name string) (string, error) {
	b, err := fsys.ReadFile(name)
	return string(b), err
}
