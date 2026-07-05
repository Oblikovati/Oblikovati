// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"oblikovati.org/kernel/exchange/e57fmt"
	"oblikovati.org/math"
)

// E57 point reader (M17-F06, #645): the ASTM E2807 (E57) format is the structured, vendor-neutral
// export of most laser/structured-light scanners. A point cloud needs only the cartesian XYZ of
// the first scan, so this reader delegates the container/descriptor/CompressedVector parsing to the
// shared kernel/exchange/e57fmt package and takes its vertices. Intensity/colour remain XYZ-only
// in this pass.
type e57Reader struct{}

// NewE57Reader returns the reader for ASTM E57 .e57 scan files.
func NewE57Reader() PointReader { return e57Reader{} }

func (e57Reader) Extensions() []string { return []string{".e57"} }

// FileUnitMM: E57 cartesian coordinates are metres by the ASTM E2807 spec, so one file unit is
// 1000 mm (#1636).
func (e57Reader) FileUnitMM() float64 { return 1000 }

// ReadSamples decodes the E57's scan points into cloud-local samples (intensity/colour are
// currently ignored).
func (e57Reader) ReadSamples(data []byte) ([]PointSample, []string, error) {
	doc, err := e57fmt.Parse(data)
	if err != nil {
		return nil, nil, err
	}
	pts, err := doc.Vertices()
	if err != nil {
		return nil, nil, err
	}
	samples := make([]PointSample, len(pts))
	for i, p := range pts {
		samples[i] = PointSample{Point: p}
	}
	return samples, nil, nil
}

// Read returns point-only coordinates for callers that do not need channels.
func (r e57Reader) Read(data []byte) ([]math.Point3, []string, error) {
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
