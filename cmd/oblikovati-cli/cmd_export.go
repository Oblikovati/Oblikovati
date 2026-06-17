// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/exchange"
	"oblikovati.org/persistence"
)

// cmdExport opens a part document and writes it to a foreign file inferred from the
// destination extension: a mesh file (STL/OBJ/3MF) at a resolution (low|medium|high; default
// medium), or a .dxf of the part's first sketch at a version (r2000|r2018; default r2000):
//
//	oblikovati-cli export fixtures/bolt.opd bolt.stl high
//	oblikovati-cli export fixtures/plate.opd plate.dxf r2018
func cmdExport(args []string, out io.Writer) error {
	if len(args) < 2 || len(args) > 3 {
		return fmt.Errorf("export: expected <src.opd> <out-file> [low|medium|high | r2000|r2018], got %d arg(s)", len(args))
	}
	src, dst := args[0], args[1]
	ws := doc.NewWorkspace(persistence.NewPackageStore())
	d, err := ws.Open(src, true)
	if err != nil {
		return fmt.Errorf("export: open %q: %w", src, err)
	}
	part, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return fmt.Errorf("export: %q is not a part document", src)
	}
	if lowerExt(dst) == ".dxf" {
		return exportSketchDXF(part, src, dst, args, out)
	}
	return exportBodies(part, src, dst, args, out)
}

// exportBodies writes the part's bodies to a mesh/B-rep file at the chosen resolution.
func exportBodies(part *compdef.PartComponentDefinition, src, dst string, args []string, out io.Writer) error {
	format, err := formatFromExt(dst)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}
	res := types.ResolutionMedium
	if len(args) == 3 {
		res = types.MeshResolution(args[2])
	}
	result, err := exchange.Export(part, dst, format, res)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "exported %s to %s (%d triangles, %s resolution)\n", src, dst, result.TriangleCount, res.Normalized())
	return nil
}

// exportSketchDXF writes the part's first sketch to a .dxf file at the chosen version.
func exportSketchDXF(part *compdef.PartComponentDefinition, src, dst string, args []string, out io.Writer) error {
	if part.Sketches().Count() == 0 {
		return fmt.Errorf("export: %q has no sketch to export to DXF", src)
	}
	version := types.DXFR2000
	if len(args) == 3 {
		version = types.DXFVersion(args[2])
	}
	n, err := exchange.ExportDXFFile(part.Sketches().Item(0), dst, version)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "exported %s sketch to %s (%d curves, %s)\n", src, dst, n, version.Normalized())
	return nil
}

// lowerExt returns path's lower-cased extension (including the dot).
func lowerExt(path string) string { return strings.ToLower(filepath.Ext(path)) }
