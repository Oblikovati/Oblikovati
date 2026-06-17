// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"os"

	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/kernel/exchange/dwg"
	"oblikovati.org/model/sketch"
)

// ExportDWG encodes a 2D sketch's geometry as an R2000 DWG file: the inverse of ImportDWG.
// The sketch→drawing conversion is shared with every drawing format (see
// sketch_to_drawing.go); only the dwg.Write call is DWG-specific. Coordinates are written
// in database units (cm).
//
//	data, err := exchange.ExportDWG(sk)
func ExportDWG(sk *sketch.Sketch) ([]byte, error) {
	return dwg.Write(&drawing.Drawing{Entities: sketchToDrawing(sk), Units: drawing.INSCentimetres})
}

// ExportDWGFile writes ExportDWG's output to path.
func ExportDWGFile(sk *sketch.Sketch, path string) error {
	data, err := ExportDWG(sk)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("export dwg: write %q: %w", path, err)
	}
	return nil
}
