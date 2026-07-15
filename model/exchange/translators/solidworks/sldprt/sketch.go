// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"bytes"
	"encoding/binary"
	"math"
	"sort"
)

// Point is a 2D sketch coordinate in metres (SolidWorks' database unit).
type Point struct{ X, Y float64 }

// Line is a sketch line segment between two points.
type Line struct{ A, B Point }

// Circle is a full circle: centre and radius (metres).
type Circle struct {
	Center Point
	Radius float64
}

// Sketch is one decoded sketch: its distinct points, and the entities recovered from them. Entities
// are associated geometrically from the cached point tags (the MFC point-index references need a
// full object-graph walk); Points is always populated, Lines/Circles when the entity kind is
// unambiguous (a pure-line loop or circle sketch).
type Sketch struct {
	Points  []Point
	Lines   []Line
	Circles []Circle
}

// pointMarker precedes every sketch coordinate in the MFC CArchive: the two bytes 1e 00 (a 2D-point
// field tag) are immediately followed by the X and Y as little-endian float64. Verified across both
// container formats and geometry kinds — a rectangle's four corners, a circle's centre + rim point —
// on generated known parts and real files.
var pointMarker = []byte{0x1e, 0x00}

const maxSketchCoord = 100.0 // metres; guards the value filter against f64 noise matching 1e 00

// Sketches decodes the part's sketch geometry. The sketch object graph lives in
// "Contents/Config-0-ResolvedFeatures" (format B) or, when that is absent, "Contents/Config-0"
// (format A) — Sketches reads whichever is present so the two container formats share one path.
func (d *Document) Sketches() []Sketch {
	stream, ok := d.sketchStream()
	if !ok {
		return nil
	}
	ordered := orderedSketchPoints(stream)
	if len(ordered) == 0 {
		return nil
	}
	sk := Sketch{Points: distinctPoints(ordered)}
	region := stream[bytes.Index(stream, []byte("sgSketch")):]
	hasLine := bytes.Contains(region, []byte("sgLineHandle"))
	hasArc := bytes.Contains(region, []byte("sgArcHandle"))
	switch {
	case hasArc && !hasLine:
		sk.Circles = circlesFromPoints(ordered)
	case hasLine && !hasArc:
		sk.Lines = loopFromVertices(distinctPoints(ordered))
	}
	return []Sketch{sk}
}

// sketchStream returns the stream that holds the sketch object graph.
func (d *Document) sketchStream() ([]byte, bool) {
	for _, name := range []string{"Contents/Config-0-ResolvedFeatures", "Contents/Config-0"} {
		if b, err := d.Stream(name); err == nil && bytes.Contains(b, []byte("sgSketch")) {
			return b, true
		}
	}
	return nil, false
}

// orderedSketchPoints extracts sketch coordinates in stream order from the sgSketch region onward
// (consecutive duplicates dropped). It scans for the 1e 00 point tag and reads the following X,Y
// float64, keeping only finite, in-range values (the range filter rejects the rare f64 whose bytes
// happen to open with 1e 00). Stream order carries entity structure — a circle's centre precedes
// its rim point — that the de-duplicated set loses.
func orderedSketchPoints(stream []byte) []Point {
	start := bytes.Index(stream, []byte("sgSketch"))
	if start < 0 {
		return nil
	}
	var pts []Point
	for i := start; ; {
		m := bytes.Index(stream[i:], pointMarker)
		if m < 0 {
			break
		}
		at := i + m
		if p, ok := readPoint(stream, at+len(pointMarker)); ok {
			if len(pts) == 0 || pts[len(pts)-1] != p {
				pts = append(pts, p)
			}
		}
		i = at + 1
	}
	return pts
}

// distinctPoints returns the unique points, preserving first-seen order.
func distinctPoints(pts []Point) []Point {
	var out []Point
	seen := map[Point]bool{}
	for _, p := range pts {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// circlesFromPoints pairs the ordered points into circles: each record stores its centre then its
// rim point (verified on generated parts — origin, off-origin, and multi-circle sketches). A point
// that repeats an already-used centre is a re-serialised reference and is skipped, so circles stay
// separated in a multi-circle sketch.
func circlesFromPoints(ordered []Point) []Circle {
	used := map[Point]bool{}
	var out []Circle
	for i := 0; i+1 < len(ordered); {
		c := ordered[i]
		if used[c] {
			i++
			continue
		}
		rim := ordered[i+1]
		out = append(out, Circle{Center: c, Radius: math.Hypot(rim.X-c.X, rim.Y-c.Y)})
		used[c] = true
		i += 2
	}
	return out
}

// loopFromVertices reconstructs a closed line loop from its corner points by ordering them around
// their centroid (the SolidWorks point-index references would give exact endpoints, but resolving
// them needs a full MFC object-graph walk — this geometry-first reconstruction, proven on the
// Inventor translator, recovers a convex loop exactly). Fewer than 3 vertices yields no lines.
func loopFromVertices(v []Point) []Line {
	if len(v) < 3 {
		return nil
	}
	cx, cy := 0.0, 0.0
	for _, p := range v {
		cx, cy = cx+p.X, cy+p.Y
	}
	cx, cy = cx/float64(len(v)), cy/float64(len(v))
	ord := append([]Point(nil), v...)
	sort.Slice(ord, func(i, j int) bool {
		return math.Atan2(ord[i].Y-cy, ord[i].X-cx) < math.Atan2(ord[j].Y-cy, ord[j].X-cx)
	})
	lines := make([]Line, len(ord))
	for i := range ord {
		lines[i] = Line{A: ord[i], B: ord[(i+1)%len(ord)]}
	}
	return lines
}

// readPoint reads an X,Y float64 pair at off, reporting ok only when both are finite and within the
// plausible sketch range (so a false 1e 00 match inside unrelated bytes is rejected).
func readPoint(stream []byte, off int) (Point, bool) {
	if off+16 > len(stream) {
		return Point{}, false
	}
	x := math.Float64frombits(binary.LittleEndian.Uint64(stream[off:]))
	y := math.Float64frombits(binary.LittleEndian.Uint64(stream[off+8:]))
	if !inSketchRange(x) || !inSketchRange(y) {
		return Point{}, false
	}
	return Point{X: x, Y: y}, true
}

// inSketchRange accepts a coordinate that is exactly zero or a normal magnitude within reach of a
// real part (>= 1 nm, <= 100 m). This rejects the subnormal f64 noise (~1e-308) that a stray 1e 00
// byte match can otherwise decode into a phantom point.
func inSketchRange(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	a := math.Abs(v)
	return a == 0 || (a >= 1e-9 && a <= maxSketchCoord)
}
