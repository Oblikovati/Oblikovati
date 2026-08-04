// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"os"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/kernel/exchange/dxf"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// ExportDXF encodes a 2D sketch's geometry as an ASCII DXF file of the given version: the
// inverse of ImportDXF. The sketch→drawing conversion is shared with every drawing format
// (sketch_to_drawing.go); only the dxf.Encode call is DXF-specific. It returns the bytes and
// the number of curves written. Coordinates are written — and $INSUNITS declared — in the
// document's preferred length unit (scaled from the database centimetre).
//
//	data, n, err := exchange.ExportDXF(sk, types.DXFR2018, part.Units())
func ExportDXF(sk *sketch.Sketch, version types.DXFVersion, units param.UnitsOfMeasure) ([]byte, int, error) {
	ins, scale := documentDrawingUnit(units)
	return encodeDXFStyled(drawing.ScaleEntities(sketchToDrawing(sk), scale), sketchDrawingStyles(sk), version, ins)
}

// ExportDXFFile writes ExportDXF's output to path, returning the number of curves written.
func ExportDXFFile(sk *sketch.Sketch, path string, version types.DXFVersion, units param.UnitsOfMeasure) (int, error) {
	ins, scale := documentDrawingUnit(units)
	data, n, err := encodeDXFStyled(drawing.ScaleEntities(sketchToDrawing(sk), scale), sketchDrawingStyles(sk), version, ins)
	if err != nil {
		return 0, err
	}
	return n, os.WriteFile(path, data, 0o644)
}

// encodeDXF encodes neutral drawing entities to DXF bytes of the given version, declaring the
// given $INSUNITS code — the shared core of every DXF exporter (the entities are already in
// that unit).
func encodeDXF(entities []drawing.Entity, version types.DXFVersion, insUnits int) ([]byte, int, error) {
	return encodeDXFStyled(entities, nil, version, insUnits)
}

// encodeDXFStyled is encodeDXF carrying per-entity formatting, keyed by each entity's Handle —
// which the sketch converters set to the source entity id, since the encoder allocates its own
// file handles (#2015).
func encodeDXFStyled(entities []drawing.Entity, styles map[uint64]drawing.Style, version types.DXFVersion, insUnits int) ([]byte, int, error) {
	dr := &drawing.Drawing{Entities: entities, Units: insUnits, Styles: styles}
	data, err := dxf.Encode(dr, dxfVersion(version))
	return data, len(dr.Entities), err
}

// writeDXFFile encodes the entities and writes them to path, returning the entity count.
func writeDXFFile(entities []drawing.Entity, path string, version types.DXFVersion, insUnits int) (int, error) {
	data, n, err := encodeDXF(entities, version, insUnits)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return 0, fmt.Errorf("export dxf: write %q: %w", path, err)
	}
	return n, nil
}

// dxfVersion maps the public DXFVersion enum to the codec's version, defaulting to R2000.
func dxfVersion(v types.DXFVersion) dxf.Version {
	if v.Normalized() == types.DXFR2018 {
		return dxf.R2018
	}
	return dxf.R2000
}
