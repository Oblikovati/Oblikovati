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
	"fmt"

	omath "oblikovati.org/math"
)

// Document is a parsed LAS file ready to yield its points.
type Document struct {
	header lasHeader
	data   []byte
}

// Parse reads and validates the LAS public header. It does not decode the points — call Vertices.
func Parse(data []byte) (*Document, error) {
	header, err := parseHeader(data)
	if err != nil {
		return nil, err
	}
	return &Document{header: header, data: data}, nil
}

// Vertices decodes every point record's XYZ into real coordinates. It errors if the record stride
// cannot hold an XYZ triple or the records run past the end of the file.
func (d *Document) Vertices() ([]omath.Point3, error) {
	h := d.header
	if h.recordLength < 12 {
		return nil, fmt.Errorf("lasfmt: point record length %d is too small to hold an XYZ triple", h.recordLength)
	}
	start := uint64(h.pointDataOffset)
	end := start + h.pointCount*uint64(h.recordLength)
	if end > uint64(len(d.data)) {
		return nil, fmt.Errorf("lasfmt: %d records of %d bytes from offset %d exceed the %d-byte file", h.pointCount, h.recordLength, start, len(d.data))
	}
	out := make([]omath.Point3, 0, h.pointCount)
	for i := uint64(0); i < h.pointCount; i++ {
		rec := d.data[start+i*uint64(h.recordLength):]
		out = append(out, d.point(rec))
	}
	return out, nil
}

// point reads the leading X/Y/Z int32 of one record and applies the header scale and offset.
func (d *Document) point(rec []byte) omath.Point3 {
	x := float64(int32(binary.LittleEndian.Uint32(rec[0:])))*d.header.scale[0] + d.header.offset[0]
	y := float64(int32(binary.LittleEndian.Uint32(rec[4:])))*d.header.scale[1] + d.header.offset[1]
	z := float64(int32(binary.LittleEndian.Uint32(rec[8:])))*d.header.scale[2] + d.header.offset[2]
	return omath.P3(omath.Scalar(x), omath.Scalar(y), omath.Scalar(z))
}
