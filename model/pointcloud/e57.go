// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"oblikovati.org/kernel/exchange/e57fmt"
	"oblikovati.org/math"
)

// E57 point reader (M17-F06, #645): the ASTM E2807 (E57) format is the structured, vendor-neutral
// export of most laser/structured-light scanners. This reader delegates the
// container/descriptor/CompressedVector parsing to the shared kernel/exchange/e57fmt package and
// takes its cartesian XYZ plus any colour and intensity channels the first scan carries, so E57
// clouds render in RGB and intensity modes rather than the flat fallback colour.
type e57Reader struct{}

// NewE57Reader returns the reader for ASTM E57 .e57 scan files.
func NewE57Reader() PointReader { return e57Reader{} }

func (e57Reader) Extensions() []string { return []string{".e57"} }

// FileUnitMM: E57 cartesian coordinates are metres by the ASTM E2807 spec, so one file unit is
// 1000 mm (#1636). This is the static default; fileUnitMM overrides it per file when the scan's
// encoding shows it is really in millimetres (#1789).
func (e57Reader) FileUnitMM() float64 { return 1000 }

// fileUnitMM overrides the metre default for a common class of non-conformant scans: an E57 whose
// cartesian channels are integer-resolution (see e57fmt.CartesianIntegerResolution) cannot truly be
// metres — no scanner captures at 1-metre resolution — so its raw integers are millimetres, the
// unit its writer used. Reading them as metres imports the cloud 1000× oversized and kilometres
// from the origin, where it renders invisibly (#1789); the millimetre reading matches the identical
// geometry re-exported as PLY. Conformant files (float coordinates, or a sub-metre ScaledInteger
// scale) keep the metre unit. ok is false only when the header cannot be parsed, leaving the static
// FileUnitMM in force (the decode then fails in ReadSamples with the real error).
func (e57Reader) fileUnitMM(data []byte) (mm float64, ok bool) {
	doc, err := e57fmt.Parse(data)
	if err != nil {
		return 0, false
	}
	return scanUnitMM(doc.CartesianIntegerResolution()), true
}

// ReadSamples decodes the E57's scan points into cloud-local samples, carrying colour (normalised
// 0..1 by e57fmt) and raw intensity through when the scan declares those channels.
func (e57Reader) ReadSamples(data []byte) ([]PointSample, []string, error) {
	doc, err := e57fmt.Parse(data)
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
func (r e57Reader) Read(data []byte) ([]math.Point3, []string, error) {
	samples, warns, err := r.ReadSamples(data)
	if err != nil {
		return nil, nil, err
	}
	return pointsOf(samples), warns, nil
}
