// SPDX-License-Identifier: GPL-2.0-only

package plyfmt

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	omath "oblikovati.org/math"
)

// ScanData is the vertex element's decoded channels for a point-cloud import: the XYZ positions plus
// any intensity and colour vertex properties the header declares (#645, #1788). RGB and Intensity are
// nil when the vertex has no such property and otherwise align 1:1 with Points. RGB holds the raw
// property values (a uchar 0..255 colour stays 0..255, a uint16 colour 0..65535); per-cloud
// normalisation to 0..1 is the model layer's job, tracked separately (#1787).
//
// Example:
//
//	d, _ := plyfmt.Parse(data)
//	scan, err := d.Scan() // scan.Points, scan.RGB (or nil), scan.Intensity (or nil)
type ScanData struct {
	Points    []omath.Point3
	RGB       [][3]float32
	Intensity []float64
}

// HasRGB reports whether the vertex declared a red/green/blue triple.
func (s ScanData) HasRGB() bool { return s.RGB != nil }

// HasIntensity reports whether the vertex declared an intensity property.
func (s ScanData) HasIntensity() bool { return s.Intensity != nil }

// vertexChannelIndices are the property positions of a vertex's decoded channels within its record;
// -1 marks an absent channel.
type vertexChannelIndices struct {
	x, y, z   int
	intensity int
	rgb       [3]int
}

// hasRGB reports whether all three colour components were located.
func (ch vertexChannelIndices) hasRGB() bool {
	return ch.rgb[0] >= 0 && ch.rgb[1] >= 0 && ch.rgb[2] >= 0
}

// Scan decodes the vertex element's positions and any intensity / colour channels in a single pass.
// It errors if there is no vertex element, a list property precedes the positions (which the
// fixed-size stride cannot skip), or the x/y/z properties are absent — the same structural faults as
// Vertices, which delegates here.
func (d *Document) Scan() (ScanData, error) {
	vtx, ok := d.element("vertex")
	if !ok || vtx.Count == 0 {
		return ScanData{}, fmt.Errorf("plyfmt: PLY has no vertex element")
	}
	ch, err := vertexChannels(vtx)
	if err != nil {
		return ScanData{}, err
	}
	if d.Format == "ascii" {
		return d.asciiScan(vtx, ch)
	}
	return d.binaryScan(vtx, ch)
}

// vertexChannels locates the x/y/z (required) and intensity/red/green/blue (optional) property
// indices in a vertex element, erroring on a list property before the positions or missing x/y/z.
func vertexChannels(vtx Element) (vertexChannelIndices, error) {
	ch := vertexChannelIndices{x: -1, y: -1, z: -1, intensity: -1, rgb: [3]int{-1, -1, -1}}
	// A name→index-slot table keeps the property scan a flat map lookup rather than a wide switch.
	slot := map[string]*int{
		"x": &ch.x, "y": &ch.y, "z": &ch.z, "intensity": &ch.intensity,
		"red": &ch.rgb[0], "green": &ch.rgb[1], "blue": &ch.rgb[2],
	}
	for i, p := range vtx.Props {
		if p.Scalar == "" {
			return ch, fmt.Errorf("plyfmt: vertex has a list property %q before its positions", p.Name)
		}
		if dst, ok := slot[p.Name]; ok {
			*dst = i
		}
	}
	if ch.x < 0 || ch.y < 0 || ch.z < 0 {
		return ch, fmt.Errorf("plyfmt: vertex has no x/y/z properties")
	}
	return ch, nil
}

// newScanData preallocates the position column and, for each channel the vertex declares, its colour
// or intensity column so the decoders fill by index.
func newScanData(count int, ch vertexChannelIndices) ScanData {
	data := ScanData{Points: make([]omath.Point3, count)}
	if ch.intensity >= 0 {
		data.Intensity = make([]float64, count)
	}
	if ch.hasRGB() {
		data.RGB = make([][3]float32, count)
	}
	return data
}

// binaryScan decodes the fixed-layout binary vertex records into positions and any channels.
func (d *Document) binaryScan(vtx Element, ch vertexChannelIndices) (ScanData, error) {
	offsets, sizes, stride := layout(vtx)
	order := d.byteOrder()
	data := newScanData(vtx.Count, ch)
	for i := 0; i < vtx.Count; i++ {
		base := i * stride
		if base+stride > len(d.body) {
			return ScanData{}, fmt.Errorf("plyfmt: truncated at vertex %d of %d", i, vtx.Count)
		}
		rec := d.body[base : base+stride]
		val := func(idx int) float64 {
			return scalarValue(rec[offsets[idx]:], vtx.Props[idx].Scalar, sizes[idx], order)
		}
		data.Points[i] = omath.P3(omath.Scalar(val(ch.x)), omath.Scalar(val(ch.y)), omath.Scalar(val(ch.z)))
		data.setBinaryChannels(i, val, ch)
	}
	return data, nil
}

// setBinaryChannels writes row i's intensity and colour columns from the per-record value accessor.
func (data ScanData) setBinaryChannels(i int, val func(int) float64, ch vertexChannelIndices) {
	if ch.intensity >= 0 {
		data.Intensity[i] = val(ch.intensity)
	}
	if ch.hasRGB() {
		data.RGB[i] = [3]float32{float32(val(ch.rgb[0])), float32(val(ch.rgb[1])), float32(val(ch.rgb[2]))}
	}
}

// asciiScan decodes exactly vtx.Count ascii vertex lines (ignoring later element lines such as faces)
// into positions and channels, erroring on a short or non-numeric line or a premature end.
func (d *Document) asciiScan(vtx Element, ch vertexChannelIndices) (ScanData, error) {
	data := newScanData(vtx.Count, ch)
	sc := bufio.NewScanner(bytes.NewReader(d.body))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	n := 0
	for sc.Scan() && n < vtx.Count {
		f := strings.Fields(sc.Text())
		if len(f) == 0 {
			continue
		}
		if !data.setAsciiRow(n, f, ch) {
			return ScanData{}, fmt.Errorf("plyfmt: ascii vertex %d is malformed: %q", n, sc.Text())
		}
		n++
	}
	if n < vtx.Count {
		return ScanData{}, fmt.Errorf("plyfmt: truncated at vertex %d of %d", n, vtx.Count)
	}
	return data, nil
}

// setAsciiRow parses row i's position (required) plus any intensity / colour columns, returning false
// if the line is too short for the positions or a position field is non-numeric.
func (data ScanData) setAsciiRow(i int, f []string, ch vertexChannelIndices) bool {
	pt, ok := asciiPoint(f, ch)
	if !ok {
		return false
	}
	data.Points[i] = pt
	if ch.intensity >= 0 {
		if v, ok := asciiField(f, ch.intensity); ok {
			data.Intensity[i] = v
		}
	}
	if ch.hasRGB() {
		data.RGB[i] = [3]float32{asciiColor(f, ch.rgb[0]), asciiColor(f, ch.rgb[1]), asciiColor(f, ch.rgb[2])}
	}
	return true
}

// asciiPoint parses the x/y/z columns of an ascii vertex line, reporting false if the line is too
// short to hold them or any is not numeric.
func asciiPoint(f []string, ch vertexChannelIndices) (omath.Point3, bool) {
	if len(f) <= maxIndex(ch.x, ch.y, ch.z) {
		return omath.Point3{}, false
	}
	x, ex := strconv.ParseFloat(f[ch.x], 64)
	y, ey := strconv.ParseFloat(f[ch.y], 64)
	z, ez := strconv.ParseFloat(f[ch.z], 64)
	if ex != nil || ey != nil || ez != nil {
		return omath.Point3{}, false
	}
	return omath.P3(omath.Scalar(x), omath.Scalar(y), omath.Scalar(z)), true
}

// asciiField parses the numeric value at column idx, reporting false when the column is missing or
// not a number.
func asciiField(f []string, idx int) (float64, bool) {
	if idx < 0 || idx >= len(f) {
		return 0, false
	}
	v, err := strconv.ParseFloat(f[idx], 64)
	return v, err == nil
}

// asciiColor reads one colour component column, yielding 0 for a missing or non-numeric field.
func asciiColor(f []string, idx int) float32 {
	v, _ := asciiField(f, idx)
	return float32(v)
}

// maxIndex returns the largest of the three position column indices — the minimum field count an
// ascii vertex line must exceed to carry x, y and z.
func maxIndex(x, y, z int) int {
	m := x
	if y > m {
		m = y
	}
	if z > m {
		m = z
	}
	return m
}
