// SPDX-License-Identifier: GPL-2.0-only

// Package nopscad provides the golden-reference machinery for the
// NopSCADlib-driven kernel test program: it loads the STL meshes rendered from
// NopSCADlib modules by OpenSCAD and reduces them to density-independent
// geometry metrics (volume, surface area, bounding box) that kernel-built
// bodies are asserted against.
//
// The STL goldens live in ./goldens/<module>.stl and are produced by
// render_goldens.py. Tests call Golden("washer") to get the expected metrics.
package nopscad

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	omath "oblikovati.org/math"
)

// Metrics is the density-independent geometry summary of a closed mesh, the
// same quantities ops.BodyGeometryProperties yields for a kernel body, plus the
// axis-aligned bounding box. Lengths are in millimetres (OpenSCAD's unit), which
// the kernel also uses internally (database units = mm via param length scale).
type Metrics struct {
	Volume   float64
	Area     float64
	Min, Max omath.Point3
	TriCount int
}

// Size returns the bounding-box extents (Max-Min) along each axis.
func (m Metrics) Size() omath.Vector3 { return m.Min.VectorTo(m.Max) }

// triangle is one STL facet's three corners (winding gives the outward normal).
type triangle struct{ a, b, c omath.Point3 }

// Golden loads the committed STL golden for a module and returns its metrics.
// It locates goldens/ relative to this source file so it is independent of the
// test's working directory. A missing golden is an error so callers can skip
// (and the program tracks the gap) rather than silently pass.
func Golden(module string) (Metrics, error) {
	path := filepath.Join(goldensDir(), module+".stl")
	tris, err := loadSTL(path)
	if err != nil {
		return Metrics{}, err
	}
	return meshMetrics(tris), nil
}

// goldensDir resolves the absolute path of the goldens directory next to this file.
func goldensDir() string {
	_, self, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(self), "goldens")
}

// meshMetrics reduces facets to volume/area/bbox. Volume uses the
// divergence-theorem signed-tetrahedron sum (mirrors
// kernel/ops.meshGeometryProperties); STL's right-hand winding guarantees
// outward facet normals, so no per-vertex re-orientation is needed.
func meshMetrics(tris []triangle) Metrics {
	if len(tris) == 0 {
		return Metrics{}
	}
	var vol, area float64
	min := omath.P3(omath.Scalar(math.Inf(1)), omath.Scalar(math.Inf(1)), omath.Scalar(math.Inf(1)))
	max := omath.P3(omath.Scalar(math.Inf(-1)), omath.Scalar(math.Inf(-1)), omath.Scalar(math.Inf(-1)))
	o := omath.P3(0, 0, 0)
	for _, t := range tris {
		vol += float64(o.VectorTo(t.a).Dot(o.VectorTo(t.b).Cross(o.VectorTo(t.c)))) / 6
		area += float64(t.a.VectorTo(t.b).Cross(t.a.VectorTo(t.c)).Length()) / 2
		for _, p := range []omath.Point3{t.a, t.b, t.c} {
			min = omath.P3(minS(min.X, p.X), minS(min.Y, p.Y), minS(min.Z, p.Z))
			max = omath.P3(maxS(max.X, p.X), maxS(max.Y, p.Y), maxS(max.Z, p.Z))
		}
	}
	if vol < 0 {
		vol = -vol // orientation-agnostic: report enclosed magnitude
	}
	return Metrics{Volume: vol, Area: area, Min: min, Max: max, TriCount: len(tris)}
}

func minS(a, b omath.Scalar) omath.Scalar {
	if a < b {
		return a
	}
	return b
}

func maxS(a, b omath.Scalar) omath.Scalar {
	if a > b {
		return a
	}
	return b
}

// loadSTL reads a binary or ASCII STL file into facets. The 80-byte header of a
// binary STL may spell "solid", so we detect format by file size: a binary STL
// is exactly 84 + 50*nTriangles bytes.
func loadSTL(path string) ([]triangle, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("nopscad golden %s: %w", path, err)
	}
	if len(raw) >= 84 {
		n := binary.LittleEndian.Uint32(raw[80:84])
		if int(84+50*n) == len(raw) {
			return parseBinarySTL(raw, n), nil
		}
	}
	return parseASCIISTL(string(raw))
}

func parseBinarySTL(raw []byte, n uint32) []triangle {
	tris := make([]triangle, 0, n)
	off := 84
	rd := func(o int) omath.Scalar {
		bits := binary.LittleEndian.Uint32(raw[o : o+4])
		return omath.Scalar(math.Float32frombits(bits))
	}
	for i := uint32(0); i < n; i++ {
		base := off + 12 // skip the 3-float facet normal
		p := func(k int) omath.Point3 {
			b := base + k*12
			return omath.P3(rd(b), rd(b+4), rd(b+8))
		}
		tris = append(tris, triangle{p(0), p(1), p(2)})
		off += 50
	}
	return tris
}

func parseASCIISTL(s string) ([]triangle, error) {
	var tris []triangle
	var pts []omath.Point3
	fields := func(line string) (float64, float64, float64, bool) {
		var x, y, z float64
		if _, err := fmt.Sscanf(line, "vertex %g %g %g", &x, &y, &z); err == nil {
			return x, y, z, true
		}
		return 0, 0, 0, false
	}
	for _, line := range strings.Split(s, "\n") {
		if x, y, z, ok := fields(strings.TrimSpace(line)); ok {
			pts = append(pts, omath.P3(omath.Scalar(x), omath.Scalar(y), omath.Scalar(z)))
			if len(pts) == 3 {
				tris = append(tris, triangle{pts[0], pts[1], pts[2]})
				pts = pts[:0]
			}
		}
	}
	if len(tris) == 0 {
		return nil, fmt.Errorf("nopscad: ASCII STL had no facets")
	}
	return tris, nil
}
