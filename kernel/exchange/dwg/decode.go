// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"fmt"
	"math"
)

// Drawing is the decoded, sketch-relevant content of a DWG file: its format
// generation and the model-space curve entities. It is the hand-off to the
// Sketch/Sketch3D converter.
type Drawing struct {
	Version  Version
	Entities []Entity
}

// Decode parses a DWG file end to end and returns its model-space curve entities
// (LINE/CIRCLE/ARC/POINT/ELLIPSE/LWPOLYLINE/SPLINE). Objects whose type has no
// geometry decoder yet, or that fail to decode individually, are skipped so a
// single bad record never sinks the whole import; the returned Warnings record
// what was dropped.
//
// Example:
//
//	dr, warns, err := dwg.Decode(bytes)
//	for _, e := range dr.Entities { /* convert to sketch geometry */ }
func Decode(data []byte) (*Drawing, []string, error) {
	h, err := ParseFileHeader(data)
	if err != nil {
		return nil, nil, err
	}
	omb, err := h.ObjectMapBytes(data)
	if err != nil {
		return nil, nil, fmt.Errorf("dwg: object map: %w", err)
	}
	od, err := h.ObjectData(data)
	if err != nil {
		return nil, nil, fmt.Errorf("dwg: object data: %w", err)
	}
	refs, err := parseObjectMap(omb)
	if err != nil {
		return nil, nil, err
	}
	dr := &Drawing{Version: h.Version}
	var warns []string
	for _, ref := range refs {
		hdr, err := decodeObjectHeader(od, ref, h.Version)
		if err != nil || !hdr.Type.IsSketchGeometry() {
			continue
		}
		r, err := seekEntityGeometry(od, ref, h.Version)
		if err != nil {
			warns = append(warns, err.Error())
			continue
		}
		e, err := decodeEntity(r, hdr, h.Version)
		if err != nil {
			warns = append(warns, err.Error())
			continue
		}
		if e != nil {
			dr.Entities = append(dr.Entities, e)
		}
	}
	return dr, warns, nil
}

// Planar reports whether every entity lies in one Z=constant plane (within tol)
// and returns that elevation. It routes an import to a 2D Sketch (with the
// returned elevation as the plane offset) versus a Sketch3D. An empty drawing is
// treated as planar at z=0.
func (d *Drawing) Planar(tol float64) (elevation float64, planar bool) {
	first := true
	var z float64
	check := func(v float64) bool {
		if first {
			z, first = v, false
			return true
		}
		return math.Abs(v-z) <= tol
	}
	for _, e := range d.Entities {
		for _, v := range entityZ(e) {
			if !check(v) {
				return 0, false
			}
		}
	}
	return z, true
}

// entityZ returns the Z coordinates an entity contributes to the planarity test.
func entityZ(e Entity) []float64 {
	switch g := e.(type) {
	case *Line:
		return []float64{g.Start[2], g.End[2]}
	case *Circle:
		return []float64{g.Center[2]}
	case *Arc:
		return []float64{g.Center[2]}
	case *Point:
		return []float64{g.Position[2]}
	case *Ellipse:
		return []float64{g.Center[2]}
	case *LwPolyline:
		return []float64{g.Elevation}
	case *Spline:
		zs := make([]float64, 0, len(g.ControlPoints)+len(g.FitPoints))
		for _, c := range g.ControlPoints {
			zs = append(zs, c[2])
		}
		for _, f := range g.FitPoints {
			zs = append(zs, f[2])
		}
		return zs
	default:
		return nil
	}
}
