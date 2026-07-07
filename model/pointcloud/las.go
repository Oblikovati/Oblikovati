// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"encoding/binary"
	"fmt"

	"oblikovati.org/kernel/exchange/lasfmt"
	"oblikovati.org/math"
)

// LAS point reader (M17-F06, #645): the ASPRS LAS format is the standard interchange for LiDAR
// point data (airborne and terrestrial surveys). A point cloud needs only the XYZ positions, so
// this reader delegates the header/record parsing to the shared kernel/exchange/lasfmt package and
// takes its vertices plus the common intensity/RGB channels. The compressed LAZ variant is not
// handled here.
type lasReader struct{}

// NewLASReader returns the reader for ASPRS .las scan files.
func NewLASReader() PointReader { return lasReader{} }

func (lasReader) Extensions() []string { return []string{".las"} }

// FileUnitMM: LAS real coordinates (stored integers × header scale + offset, applied by lasfmt)
// are metres by ASPRS convention, so one file unit is 1000 mm (#1636). This is the static default;
// fileUnitMM overrides it per file when the header's scale shows it is really millimetres (#1789).
func (lasReader) FileUnitMM() float64 { return 1000 }

// lasMillimetreScan reports whether a LAS quantises coordinates to a whole metre or coarser on every
// axis. header.Scale is the coordinate step in the assumed metre unit; a step ≥ 1 m cannot resolve
// any real scan or survey (LiDAR is sub-centimetre), so the file's coordinates are really
// millimetres despite the metre-unit ASPRS convention — the LAS analogue of E57's integer-resolution
// scans (#1789). An unset/zero scale (a malformed header) is not flagged, leaving the metre default.
func lasMillimetreScan(scale [3]float64) bool {
	return scale[0] >= 1 && scale[1] >= 1 && scale[2] >= 1
}

// fileUnitMM overrides the metre default (see FileUnitMM) for a LAS whose coordinate quantisation is
// too coarse to be metres, reading it as millimetres so a millimetre-authored scan does not import
// 1000× oversized and kilometres from the origin, where it renders invisibly (#1789). Conformant
// surveys (sub-metre scale) keep the metre unit. ok is false only when the header cannot be parsed,
// leaving the static FileUnitMM in force (the decode then fails in ReadSamples with the real error).
func (lasReader) fileUnitMM(data []byte) (mm float64, ok bool) {
	doc, err := lasfmt.Parse(data)
	if err != nil {
		return 0, false
	}
	return scanUnitMM(lasMillimetreScan(doc.Header().Scale)), true
}

// ReadSamples decodes the LAS point records into cloud-local samples.
func (lasReader) ReadSamples(data []byte) ([]PointSample, []string, error) {
	doc, err := lasfmt.Parse(data)
	if err != nil {
		return nil, nil, err
	}
	samples, err := decodeLASSamples(doc)
	return samples, nil, err
}

// Read returns point-only coordinates for callers that do not need channels.
func (r lasReader) Read(data []byte) ([]math.Point3, []string, error) {
	samples, warns, err := r.ReadSamples(data)
	if err != nil {
		return nil, nil, err
	}
	out := make([]math.Point3, len(samples))
	for i, s := range samples {
		out[i] = s.Point
	}
	return out, warns, nil
}

func decodeLASSamples(doc *lasfmt.Document) ([]PointSample, error) {
	h := doc.Header()
	if h.RecordLength < 12 {
		return nil, fmt.Errorf("lasfmt: point record length %d is too small to hold an XYZ triple", h.RecordLength)
	}
	start := uint64(h.PointDataOffset)
	end := start + h.PointCount*uint64(h.RecordLength)
	if end > uint64(len(doc.Raw())) {
		return nil, fmt.Errorf("lasfmt: %d records of %d bytes from offset %d exceed the %d-byte file", h.PointCount, h.RecordLength, start, len(doc.Raw()))
	}
	out := make([]PointSample, 0, h.PointCount)
	recLen := uint64(h.RecordLength)
	for i := uint64(0); i < h.PointCount; i++ {
		base := start + i*recLen
		rec := doc.Raw()[base : base+recLen]
		out = append(out, decodeLASSample(rec, h))
	}
	return out, nil
}

func decodeLASSample(rec []byte, h lasfmt.Header) PointSample {
	s := PointSample{
		Point: math.P3(
			math.Scalar(float64(int32(binary.LittleEndian.Uint32(rec[0:])))*h.Scale[0]+h.Offset[0]),
			math.Scalar(float64(int32(binary.LittleEndian.Uint32(rec[4:])))*h.Scale[1]+h.Offset[1]),
			math.Scalar(float64(int32(binary.LittleEndian.Uint32(rec[8:])))*h.Scale[2]+h.Offset[2]),
		),
	}
	if len(rec) >= 14 {
		s.HasIntensity = true
		s.Intensity = float64(binary.LittleEndian.Uint16(rec[12:14]))
	}
	if rgbOffset, ok := lasRGBOffset(h.PointFormat); ok && rgbOffset+6 <= len(rec) {
		s.HasRGB = true
		s.RGB = [3]float32{
			float32(binary.LittleEndian.Uint16(rec[rgbOffset : rgbOffset+2])),
			float32(binary.LittleEndian.Uint16(rec[rgbOffset+2 : rgbOffset+4])),
			float32(binary.LittleEndian.Uint16(rec[rgbOffset+4 : rgbOffset+6])),
		}
	}
	return s
}

func lasRGBOffset(format uint8) (int, bool) {
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
