// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
)

// cmdNew creates a document of the named kind and saves it as an .obk package at
// path, the core fixture-generation command:
//
//	oblikovati-cli new part fixtures/bracket.obk
func cmdNew(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(out)
	seed := fs.Bool("seed", false, "add a sketch + user parameter to a part (in-memory only; not persisted until M07)")
	flags, operands := partitionFlags(args)
	if err := fs.Parse(flags); err != nil {
		return err
	}
	if len(operands) != 2 {
		return fmt.Errorf("new: expected <type> <path>, got %d arg(s); see `oblikovati-cli help`", len(operands))
	}
	t, err := doc.ParseDocumentType(operands[0])
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	path := withPackageExt(operands[1])
	ws := doc.NewWorkspace(persistence.NewPackageStore())
	d, err := createDocument(ws, t, path, *seed, out)
	if err != nil {
		return err
	}
	if err := ws.Save(d); err != nil {
		return fmt.Errorf("new: save %q: %w", path, err)
	}
	fmt.Fprintf(out, "created %s %q at %s\n", t, d.DisplayName(), path)
	return nil
}

// createDocument adds a document of kind t to ws. A part is realized with a component
// definition (so --seed has somewhere to write); other kinds use the default content
// and ignore --seed.
func createDocument(ws *doc.Workspace, t doc.DocumentType, path string, seed bool, out io.Writer) (*doc.Document, error) {
	if t != doc.Part {
		if seed {
			fmt.Fprintln(out, "note: --seed applies only to parts; ignored")
		}
		d, err := ws.Add(t, path, true)
		if err != nil {
			return nil, fmt.Errorf("new: %w", err)
		}
		return d, nil
	}
	d, err := compdef.AddPart(ws, path, true)
	if err != nil {
		return nil, fmt.Errorf("new: %w", err)
	}
	if seed {
		seedPart(d, out)
	}
	return d, nil
}

// seedPart populates a part with a user parameter and a sketch so a fixture is not a
// bare shell. NOTE: model streams are not yet persisted (only the manifest is — see
// persistence/store.go), so this content round-trips through save→reopen: the recipe
// (parameter + sketch + extrude) is recomputed into the same solid on open (ADR-0020).
func seedPart(d *doc.Document, out io.Writer) {
	def, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return
	}
	_, _ = def.Parameters().AddUserParameter("width", "4 cm")
	sk := def.Sketches().Add(sketch.XYPlane())
	seedRectangle(sk, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	fmt.Fprintln(out, "note: --seed builds a 4×3×5 extruded block (parameter + sketch + extrude); it round-trips through save/open")
}

// seedRectangle adds a closed w×h rectangle from four corner-sharing points.
func seedRectangle(sk *sketch.Sketch, w, h float64) {
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(w, 0))
	c2 := sk.Points().Add(math.P2(w, h))
	c3 := sk.Points().Add(math.P2(0, h))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}

// withPackageExt returns path with the [doc.PackageExtension] suffix, appending it
// when absent so generated fixtures share one predictable extension. A path that
// already ends in .obk is returned unchanged.
func withPackageExt(path string) string {
	if strings.EqualFold(filepath.Ext(path), doc.PackageExtension) {
		return path
	}
	return path + doc.PackageExtension
}
