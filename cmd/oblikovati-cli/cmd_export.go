// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"oblikovati/api/types"
	"oblikovati/model/compdef"
	"oblikovati/model/doc"
	"oblikovati/model/exchange"
	"oblikovati/persistence"
)

// cmdExport opens an .obk part and writes its bodies to a mesh file (STL/OBJ/3MF) at the
// given resolution (low|medium|high; default medium):
//
//	oblikovati-cli export fixtures/bolt.obk bolt.stl high
//
// The format is inferred from the destination extension.
func cmdExport(args []string, out io.Writer) error {
	if len(args) < 2 || len(args) > 3 {
		return fmt.Errorf("export: expected <src.obk> <mesh-file> [low|medium|high], got %d arg(s)", len(args))
	}
	src, dst := args[0], args[1]
	format, err := formatFromExt(dst)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}
	res := types.ResolutionMedium
	if len(args) == 3 {
		res = types.MeshResolution(args[2])
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore())
	d, err := ws.Open(src, true)
	if err != nil {
		return fmt.Errorf("export: open %q: %w", src, err)
	}
	part, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return fmt.Errorf("export: %q is not a part document", src)
	}
	result, err := exchange.MeshExchange{}.ExportFrom(part, dst, format, res)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "exported %s to %s (%d triangles, %s resolution)\n", src, dst, result.TriangleCount, res.Normalized())
	return nil
}

// lowerExt returns path's lower-cased extension (including the dot).
func lowerExt(path string) string { return strings.ToLower(filepath.Ext(path)) }
