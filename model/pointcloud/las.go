// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"oblikovati.org/kernel/exchange/lasfmt"
	"oblikovati.org/math"
)

// LAS point reader (M17-F06, #645): the ASPRS LAS format is the standard interchange for LiDAR
// point data (airborne and terrestrial surveys). All the header/record/VLR parsing lives in the
// shared kernel/exchange/lasfmt package (#1788); this reader maps its decoded XYZ plus the common
// intensity/RGB channels onto cloud-local samples and owns the LAS unit policy. The compressed LAZ
// variant is not handled.
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

// fileUnitMM overrides the static metre default (see FileUnitMM) with the file's true coordinate
// unit (#1789). It first honours an explicit CRS declaration — a WKT or GeoTIFF linear unit in the
// VLRs — so a survey in US survey feet (~3.28× off if read as metres) or a WKT-declared millimetre
// scan places at true scale. With no linear CRS it falls back to the quantisation heuristic: a scan
// quantised to a whole metre or coarser is really millimetres, else the ASPRS metre. ok is false
// only when the header cannot be parsed, leaving the static FileUnitMM in force (the decode then
// fails in ReadSamples with the real error).
func (lasReader) fileUnitMM(data []byte) (mm float64, ok bool) {
	doc, err := lasfmt.Parse(data)
	if err != nil {
		return 0, false
	}
	crsMetres, declared := doc.CoordinateUnitMetres()
	return lasUnitMM(crsMetres, declared, doc.Header().Scale), true
}

// lasUnitMM chooses the file's millimetres-per-unit: an explicit CRS linear unit wins (its
// metres-per-unit scaled to mm), else the coarse-quantisation heuristic decides mm vs metre (#1789).
// Kept pure so the CRS-over-heuristic precedence is tested without synthesising LAS bytes.
func lasUnitMM(crsMetres float64, crsDeclared bool, scale [3]float64) float64 {
	if crsDeclared {
		return crsMetres * 1000 // metres-per-unit → millimetres-per-unit
	}
	return scanUnitMM(lasMillimetreScan(scale))
}

// ReadSamples decodes the LAS point records into cloud-local samples, carrying the intensity and RGB
// channels the record format declares (raw values; the model normalises colour per-cloud, #1787).
func (lasReader) ReadSamples(data []byte) ([]PointSample, []string, error) {
	doc, err := lasfmt.Parse(data)
	if err != nil {
		return nil, nil, err
	}
	scan, err := doc.Scan()
	if err != nil {
		return nil, nil, err
	}
	return samplesFromChannels(scan.Points, scan.RGB, scan.Intensity), nil, nil
}

// Read returns point-only coordinates for callers that do not need channels.
func (r lasReader) Read(data []byte) ([]math.Point3, []string, error) {
	samples, warns, err := r.ReadSamples(data)
	if err != nil {
		return nil, nil, err
	}
	return pointsOf(samples), warns, nil
}
