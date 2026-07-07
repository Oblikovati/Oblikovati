// SPDX-License-Identifier: GPL-2.0-only

package e57fmt

import omath "oblikovati.org/math"

// ScanData is the first scan's decoded channels: the cartesian positions plus any colour and
// intensity the prototype carried (#645). RGB and Intensity are nil when the scan lacks those
// channels and otherwise align 1:1 with Points. RGB is normalised to 0..1 from each colour field's
// declared bounds (colours are commonly Integer 0..255, but the bounds are read from the prototype,
// not assumed). Intensity is the raw decoded value: the point-cloud model normalises intensity
// per-cloud from the whole column, mirroring how the LAS and PLY readers hand off intensity.
type ScanData struct {
	Points    []omath.Point3
	RGB       [][3]float32
	Intensity []float64
}

// HasRGB reports whether the scan decoded a colourRed/Green/Blue triple.
func (s ScanData) HasRGB() bool { return s.RGB != nil }

// HasIntensity reports whether the scan decoded an intensity channel.
func (s ScanData) HasIntensity() bool { return s.Intensity != nil }

// Scan decodes the first scan's cartesian XYZ and, when the prototype declares them, its
// colourRed/Green/Blue and intensity channels, in a single pass over the CompressedVector. It
// errors if the prototype lacks the three cartesian channels (e.g. a spherical-only scan).
func (d *Document) Scan() (ScanData, error) {
	xi, yi, zi, err := d.cartesianIndices()
	if err != nil {
		return ScanData{}, err
	}
	ri, gi, bi, hasRGB := d.colorIndices()
	ii, hasIntensity := d.intensityIndex()

	want := []int{xi, yi, zi}
	if hasRGB {
		want = append(want, ri, gi, bi)
	}
	if hasIntensity {
		want = append(want, ii)
	}
	cols, err := d.decodeFields(want)
	if err != nil {
		return ScanData{}, err
	}
	data := ScanData{Points: zipPoints(cols[xi], cols[yi], cols[zi])}
	n := len(data.Points)
	if hasRGB {
		data.RGB = zipRGB(cols[ri], cols[gi], cols[bi], d.points.fields[ri], d.points.fields[gi], d.points.fields[bi], n)
	}
	if hasIntensity {
		data.Intensity = trimColumn(cols[ii], n)
	}
	return data, nil
}

// colorIndices returns the prototype positions of colourRed/Green/Blue, and whether all three are
// present — a partial colour triple (a scanner quirk) is treated as no colour.
func (d *Document) colorIndices() (r, g, b int, ok bool) {
	r, g, b = -1, -1, -1
	for i, f := range d.points.fields {
		switch f.name {
		case "colorRed":
			r = i
		case "colorGreen":
			g = i
		case "colorBlue":
			b = i
		}
	}
	return r, g, b, r >= 0 && g >= 0 && b >= 0
}

// intensityIndex returns the prototype position of the intensity channel, if present.
func (d *Document) intensityIndex() (int, bool) {
	for i, f := range d.points.fields {
		if f.name == "intensity" {
			return i, true
		}
	}
	return -1, false
}

// zipRGB combines the three colour columns into normalised 0..1 triples, stopping at the shortest so
// a short column never indexes past another and never past the point count.
func zipRGB(rs, gs, bs []float64, rf, gf, bf protoField, pointCount int) [][3]float32 {
	n := pointCount
	for _, c := range [][]float64{rs, gs, bs} {
		if len(c) < n {
			n = len(c)
		}
	}
	out := make([][3]float32, n)
	for i := 0; i < n; i++ {
		out[i] = [3]float32{normColor(rs[i], rf), normColor(gs[i], gf), normColor(bs[i], bf)}
	}
	return out
}

// normColor maps a decoded colour value to 0..1 using its field's declared value range. The range is
// the decoded span of the bounds (min/max mapped through scale/offset), so an Integer 0..255 field
// divides by 255 and a ScaledInteger colour uses its real bounds. A Float colour (or any field
// without usable bounds) is passed through unchanged, since E57 float colour is already 0..1.
func normColor(v float64, f protoField) float32 {
	lo := float64(f.min)*f.scale + f.offset
	hi := float64(f.max)*f.scale + f.offset
	if hi > lo {
		return float32((v - lo) / (hi - lo))
	}
	return float32(v)
}

// trimColumn caps a decoded column at the point count so a trailing extra value never mismatches the
// positions it pairs with.
func trimColumn(col []float64, n int) []float64 {
	if len(col) > n {
		return col[:n]
	}
	return col
}
