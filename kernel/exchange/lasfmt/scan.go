// SPDX-License-Identifier: GPL-2.0-only

package lasfmt

import (
	"encoding/binary"
	"fmt"

	omath "oblikovati.org/math"
)

// ScanData is the decoded point channels a point-cloud import needs: the XYZ positions plus the
// intensity and RGB colour when the point data record format carries them (#645, #1788). Intensity
// and RGB are nil when the format omits them and otherwise align 1:1 with Points. Every LAS record
// format (0–10) holds intensity, so any record whose stride reaches it yields an intensity column;
// RGB comes only from the colour formats (2, 3, 5, 7, 8, 10). RGB is normalised to 0..1 from the
// ASPRS 16-bit colour range (/65535), so a caller renders colour without guessing the bit depth per
// point (#1787) — the same contract as e57fmt.Scan. Intensity is the raw decoded value.
//
// Example:
//
//	d, _ := lasfmt.Parse(data)
//	scan, err := d.Scan() // scan.Points, scan.RGB (or nil), scan.Intensity (or nil)
type ScanData struct {
	Points    []omath.Point3
	RGB       [][3]float32
	Intensity []float64
}

// HasRGB reports whether the record format carried a red/green/blue triple.
func (s ScanData) HasRGB() bool { return s.RGB != nil }

// HasIntensity reports whether the records held the standard intensity field.
func (s ScanData) HasIntensity() bool { return s.Intensity != nil }

// Scan decodes every point record's XYZ and, when the record format declares them, its intensity and
// RGB colour in one pass. It errors on the same structural faults as Vertices (which delegates here):
// a record too short to hold an XYZ triple, or records that run past the end of the file.
func (d *Document) Scan() (ScanData, error) {
	h := d.header
	if h.recordLength < 12 {
		return ScanData{}, fmt.Errorf("lasfmt: point record length %d is too small to hold an XYZ triple", h.recordLength)
	}
	start := uint64(h.pointDataOffset)
	end := start + h.pointCount*uint64(h.recordLength)
	if end > uint64(len(d.data)) {
		return ScanData{}, fmt.Errorf("lasfmt: %d records of %d bytes from offset %d exceed the %d-byte file", h.pointCount, h.recordLength, start, len(d.data))
	}
	data := newLASScanData(h)
	recLen := uint64(h.recordLength)
	for i := uint64(0); i < h.pointCount; i++ {
		rec := d.data[start+i*recLen:]
		data.Points[i] = d.point(rec)
		data.setChannels(int(i), rec, h)
	}
	return data, nil
}

// newLASScanData preallocates the position column and, when the record stride reaches them, the
// intensity (offset 12) and RGB columns so the decode fills by index.
func newLASScanData(h lasHeader) ScanData {
	data := ScanData{Points: make([]omath.Point3, h.pointCount)}
	if int(h.recordLength) >= 14 {
		data.Intensity = make([]float64, h.pointCount)
	}
	if off, ok := rgbRecordOffset(h.pointFormat); ok && off+6 <= int(h.recordLength) {
		data.RGB = make([][3]float32, h.pointCount)
	}
	return data
}

// lasColorMax is the ASPRS 16-bit colour channel maximum; RGB is normalised to 0..1 by it (#1787).
const lasColorMax = 65535

// setChannels writes row i's intensity and colour columns from the point record. It reads only the
// columns newLASScanData allocated, so the stride is known to hold whatever offset it reads. Colour
// is normalised to 0..1 from the 16-bit range; intensity stays raw (the model ramps it per cloud).
func (data ScanData) setChannels(i int, rec []byte, h lasHeader) {
	if data.Intensity != nil {
		data.Intensity[i] = float64(binary.LittleEndian.Uint16(rec[12:14]))
	}
	if data.RGB != nil {
		off, _ := rgbRecordOffset(h.pointFormat)
		data.RGB[i] = [3]float32{
			float32(binary.LittleEndian.Uint16(rec[off:off+2])) / lasColorMax,
			float32(binary.LittleEndian.Uint16(rec[off+2:off+4])) / lasColorMax,
			float32(binary.LittleEndian.Uint16(rec[off+4:off+6])) / lasColorMax,
		}
	}
}

// rgbRecordOffset returns the byte offset of the red channel within a point record for the LAS point
// data record formats that carry RGB (2, 3, 5, 7, 8, 10), and whether the format has colour at all
// (ASPRS LAS R15, §2.3); green and blue follow at +2 and +4.
func rgbRecordOffset(format uint8) (int, bool) {
	switch format {
	case 2:
		return 20, true
	case 3, 5:
		return 28, true
	case 7, 8, 10:
		return 30, true
	default:
		return 0, false
	}
}
