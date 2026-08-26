// SPDX-License-Identifier: GPL-2.0-only

// Package e57fmt is a clean-room reader for the ASTM E2807 (E57) 3D-imaging format, scoped to what
// a point-cloud import needs: the cartesian XYZ positions of the first scan plus its colour and
// intensity channels. E57 stores its structure in an XML descriptor and its points in a bit-packed
// CompressedVector, both inside a checksummed-page container; this package de-pages the container,
// parses the descriptor, and decodes the points' cartesianX/Y/Z, colourRed/Green/Blue, and
// intensity channels (ScaledInteger, Integer, or Float) in a single pass — see Scan. It does not
// decode spherical coordinates, images, or multiple scans — those are skipped (#645).
package e57fmt

import (
	"fmt"

	omath "oblikovati.org/math"
)

// Document is a parsed E57 file ready to yield its scan points.
type Document struct {
	header fileHeader
	paged  *pagedFile
	points pointsSection
}

// Parse reads the header, de-pages and parses the XML descriptor, and locates the first scan's
// points section. It does not yet decode the points — call Vertices for that.
func Parse(data []byte) (*Document, error) {
	header, err := parseHeader(data)
	if err != nil {
		return nil, err
	}
	paged := newPagedFile(data, header.pageSize)
	xmlDoc, _, err := paged.readLogical(header.xmlPhysicalOffset, header.xmlLogicalLength)
	if err != nil {
		return nil, err
	}
	points, err := parsePointsSection(xmlDoc)
	if err != nil {
		return nil, err
	}
	return &Document{header: header, paged: paged, points: points}, nil
}

// Vertices decodes the scan's cartesian XYZ positions, discarding any colour/intensity. It errors if
// the prototype lacks the three cartesian channels (e.g. a spherical-only scan), naming the channels
// it did find. Callers that need colour or intensity should use Scan.
func (d *Document) Vertices() ([]omath.Point3, error) {
	data, err := d.Scan()
	if err != nil {
		return nil, err
	}
	return data.Points, nil
}

// CartesianIntegerResolution reports whether the scan's cartesian XYZ channels are all stored at
// integer resolution — a ScaledInteger with unit scale, or a plain Integer. Such a file quantises
// positions to whole file-units, which is meaningless at the ASTM E2807 metre default (no scanner
// captures at 1-metre resolution), so its raw integers are in practice millimetres; callers use this
// to keep such a scan from importing 1000× oversized (Oblikovati/Oblikovati#1789). Returns false
// when any cartesian channel is a Float, carries a sub-unit scale, or is absent — i.e. whenever the
// spec's metre unit should stand.
func (d *Document) CartesianIntegerResolution() bool {
	xi, yi, zi, err := d.cartesianIndices()
	if err != nil {
		return false
	}
	for _, i := range []int{xi, yi, zi} {
		if !d.points.fields[i].integerResolution() {
			return false
		}
	}
	return true
}

// cartesianIndices returns the prototype positions of cartesianX/Y/Z.
func (d *Document) cartesianIndices() (int, int, int, error) {
	xi, yi, zi := -1, -1, -1
	for i, f := range d.points.fields {
		switch f.name {
		case "cartesianX":
			xi = i
		case "cartesianY":
			yi = i
		case "cartesianZ":
			zi = i
		}
	}
	if xi < 0 || yi < 0 || zi < 0 {
		return 0, 0, 0, fmt.Errorf("e57fmt: scan has no cartesian XYZ channels (found %v)", d.fieldNames())
	}
	return xi, yi, zi, nil
}

func (d *Document) fieldNames() []string {
	names := make([]string, len(d.points.fields))
	for i, f := range d.points.fields {
		names[i] = f.name
	}
	return names
}

// zipPoints combines equal-length coordinate columns into points, stopping at the shortest so a
// short final column never indexes past another.
func zipPoints(xs, ys, zs []float64) []omath.Point3 {
	n := min(len(zs), min(len(ys), len(xs)))
	out := make([]omath.Point3, n)
	for i := 0; i < n; i++ {
		out[i] = omath.P3(omath.Scalar(xs[i]), omath.Scalar(ys[i]), omath.Scalar(zs[i]))
	}
	return out
}
