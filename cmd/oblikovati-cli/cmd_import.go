// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"io"

	"oblikovati.org/api/types"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/exchange"
	"oblikovati.org/persistence"
)

// cmdImport creates a new part, imports a mesh file (STL/OBJ/3MF) into it as a solid (or
// surface body when the mesh is open), and saves it as an .obk package:
//
//	oblikovati-cli import bolt.stl fixtures/bolt.obk
//
// The format is inferred from the source extension.
func cmdImport(args []string, out io.Writer) error {
	if len(args) != 2 {
		return fmt.Errorf("import: expected <mesh-file> <dst.obk>, got %d arg(s)", len(args))
	}
	src, dst := args[0], withPackageExt(args[1])
	format, err := formatFromExt(src)
	if err != nil {
		return fmt.Errorf("import: %w", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore())
	d, err := compdef.AddPart(ws, dst, true)
	if err != nil {
		return fmt.Errorf("import: %w", err)
	}
	part := d.Content().(*compdef.PartComponentDefinition)
	res, err := exchange.Import(part, src, format)
	if err != nil {
		return err
	}
	if err := ws.Save(d); err != nil {
		return fmt.Errorf("import: save %q: %w", dst, err)
	}
	reportImport(out, src, dst, res)
	return nil
}

// reportImport prints a one-line human summary of an import.
func reportImport(out io.Writer, src, dst string, res exchange.ImportResult) {
	kind := "surface body"
	if res.Solid {
		kind = "solid"
	}
	fmt.Fprintf(out, "imported %s as a %s into %s\n", src, kind, dst)
	for _, w := range res.Warnings {
		fmt.Fprintf(out, "  warning: %s\n", w)
	}
}

// formatFromExt infers the exchange format from a file extension.
func formatFromExt(path string) (types.ExchangeFormat, error) {
	switch ext := lowerExt(path); ext {
	case ".stl":
		return types.FormatSTL, nil
	case ".obj":
		return types.FormatOBJ, nil
	case ".3mf":
		return types.Format3MF, nil
	case ".step", ".stp":
		return types.FormatSTEP, nil
	default:
		return "", fmt.Errorf("unsupported extension %q (want .stl|.obj|.3mf|.step)", ext)
	}
}
