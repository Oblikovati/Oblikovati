// SPDX-License-Identifier: GPL-2.0-only

// Package lasfmt is a clean-room reader for the ASPRS LAS LiDAR format (uncompressed .las), scoped
// to what a point-cloud import needs: the XYZ positions. Every LAS point data record format (0–10)
// begins with the same three little-endian int32 — X, Y, Z — and the public header carries the
// scale and offset that turn those stored integers into real coordinates
// (real = stored*scale + offset). This package reads the header and walks the fixed-length records;
// it does not decode intensity, returns, classification, colour, or waveforms, and it does not
// handle LAZ (the arithmetic-coded variant) — those are out of scope (#645).
package lasfmt

import (
	"encoding/binary"

	omath "oblikovati.org/math"
)

// Document is a parsed LAS file ready to yield its points.
type Document struct {
	header lasHeader
	data   []byte
}

// Header is the public LAS record metadata a point-cloud reader needs to decode channels.
type Header struct {
	PointDataOffset uint32
	PointFormat     uint8
	RecordLength    uint16
	PointCount      uint64
	Scale           [3]float64
	Offset          [3]float64
}

// Parse reads and validates the LAS public header. It does not decode the points — call Vertices.
func Parse(data []byte) (*Document, error) {
	header, err := parseHeader(data)
	if err != nil {
		return nil, err
	}
	return &Document{header: header, data: data}, nil
}

// Header returns the parsed public header fields.
func (d *Document) Header() Header {
	return Header{
		PointDataOffset: d.header.pointDataOffset,
		PointFormat:     d.header.pointFormat,
		RecordLength:    d.header.recordLength,
		PointCount:      d.header.pointCount,
		Scale:           d.header.scale,
		Offset:          d.header.offset,
	}
}

// Raw returns the original file bytes so callers can decode record extensions beyond XYZ.
func (d *Document) Raw() []byte { return d.data }

// CoordinateUnitMetres returns the size in metres of the file's horizontal coordinate unit as
// declared by its CRS VLRs (WKT record 2112, or the GeoTIFF GeoKeys), and whether one was found —
// 1.0 for metre, 0.3048 for the international foot, 0.30480060960121924 for the US survey foot, etc.
// A point-cloud import uses it to place a scan at true scale instead of assuming metres; ok is false
// when the file declares no linear CRS (a geographic degrees CRS, or no projection VLR at all), and
// the caller falls back to its own unit policy (Oblikovati/Oblikovati#1789).
func (d *Document) CoordinateUnitMetres() (float64, bool) {
	return coordinateUnitMetres(d.data, d.header)
}

// Vertices decodes every point record's XYZ into real coordinates. It is the position-only
// projection of Scan for callers (the mesh importer) that do not need colour or intensity.
func (d *Document) Vertices() ([]omath.Point3, error) {
	scan, err := d.Scan()
	if err != nil {
		return nil, err
	}
	return scan.Points, nil
}

// point reads the leading X/Y/Z int32 of one record and applies the header scale and offset.
func (d *Document) point(rec []byte) omath.Point3 {
	x := float64(int32(binary.LittleEndian.Uint32(rec[0:])))*d.header.scale[0] + d.header.offset[0]
	y := float64(int32(binary.LittleEndian.Uint32(rec[4:])))*d.header.scale[1] + d.header.offset[1]
	z := float64(int32(binary.LittleEndian.Uint32(rec[8:])))*d.header.scale[2] + d.header.offset[2]
	return omath.P3(omath.Scalar(x), omath.Scalar(y), omath.Scalar(z))
}
