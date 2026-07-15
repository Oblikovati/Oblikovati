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

// Arc is a circular arc from Start to End about Center (metres).
type Arc struct {
	Center, Start, End Point
	Radius             float64
}

// Sketch is one decoded sketch: its distinct points, and the entities recovered from them. Entities
// are associated geometrically from the cached point tags (the MFC point-index references need a
// full object-graph walk); Points is always populated, Lines/Circles when the entity kind is
// unambiguous (a pure-line loop or circle sketch).
type Sketch struct {
	Points      []Point
	Lines       []Line
	Circles     []Circle
	Arcs        []Arc
	Constraints []Constraint
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
// The stream is split into one region per sketch feature before entity association, so a
// multi-sketch part decodes each sketch on its own rather than merging every point into one.
func (d *Document) Sketches() []Sketch {
	stream, ok := d.sketchStream()
	if !ok {
		return nil
	}
	var out []Sketch
	for _, region := range sketchRegions(stream) {
		if sk, ok := sketchFromRegion(region); ok {
			out = append(out, sk)
		}
	}
	return out
}

// sketchFromRegion decodes one sketch's byte region into points and, when the entity kind is
// determinable, lines/arcs/circles. It returns ok=false for a region with no sketch coordinates.
func sketchFromRegion(region []byte) (Sketch, bool) {
	ordered := pointsIn(region)
	if len(ordered) == 0 {
		return Sketch{}, false
	}
	sk := Sketch{Points: distinctPoints(ordered), Constraints: constraintsIn(region)}
	hasLine := bytes.Contains(region, []byte("sgLineHandle"))
	hasArc := bytes.Contains(region, []byte("sgArcHandle"))
	switch {
	case hasLine && hasArc:
		sk.Lines, sk.Arcs = chainEntities(ordered)
	case hasArc && !hasLine:
		if a, ok := singleArc(sk.Points); ok {
			sk.Arcs = []Arc{a}
		} else {
			sk.Circles = circlesFromPoints(ordered) // centre+rim pairs -> full circles
		}
	case hasLine && !hasArc:
		sk.Lines = loopFromVertices(sk.Points)
	}
	return sk, true
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

// nameMarker precedes a feature's name in the MFC CArchive: the name-object tag 04 80 then the
// wide-string marker ff fe ff. Each sketch's geometry follows its name, so these mark the region
// boundaries used to split a multi-sketch stream.
var nameMarker = []byte{0x04, 0x80, 0xff, 0xfe, 0xff}

// sketchRegions splits the stream into one byte slice per sketch feature. Feature names (planes,
// folders, the origin, and sketches) are delimited by nameMarker; a region that holds at least two
// sketch coordinates is a sketch (planes/folders hold none, a point feature holds one). If no such
// split is found — older CFBF Config-0 lays the graph out differently — the whole sgSketch region is
// returned as a single sketch, preserving the single-sketch path.
//
// This name-boundary split is exact when each sketch introduces its own entity classes; a sketch
// that only re-uses a kind first seen in an earlier sketch loses that class string (it is written as
// an object instance), so its kind is not yet detected — the full fix is an MFC object-graph walk.
func sketchRegions(stream []byte) [][]byte {
	var starts []int
	for i := 0; i+len(nameMarker) <= len(stream); i++ {
		if bytes.Equal(stream[i:i+len(nameMarker)], nameMarker) {
			starts = append(starts, i)
		}
	}
	starts = append(starts, len(stream))
	var regions [][]byte
	for i := 0; i+1 < len(starts); i++ {
		r := stream[starts[i]:starts[i+1]]
		if len(pointsIn(r)) >= 2 {
			regions = append(regions, r)
		}
	}
	if len(regions) == 0 {
		if s := bytes.Index(stream, []byte("sgSketch")); s >= 0 {
			return [][]byte{stream[s:]}
		}
	}
	return regions
}

// pointsIn extracts sketch coordinates from a byte region in order (consecutive duplicates dropped).
// It scans for the 1e 00 point tag and reads the following X,Y float64, keeping only finite,
// in-range values (the range filter rejects the rare f64 whose bytes happen to open with 1e 00).
// Stream order carries entity structure — a circle's centre precedes its rim point — that the
// de-duplicated set loses.
func pointsIn(region []byte) []Point {
	var pts []Point
	for i := 0; ; {
		m := bytes.Index(region[i:], pointMarker)
		if m < 0 {
			break
		}
		at := i + m
		if p, ok := readPoint(region, at+len(pointMarker)); ok {
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

// chainEntities walks the ordered cached points of a line+arc profile as a connected chain and
// classifies each segment geometrically — no per-entity object tags needed. Each entity inherits its
// start from the previous entity's end; a line then caches one point (its end), an arc caches two
// (its centre, then its end). A candidate centre C is recognised because it is equidistant from the
// running start and the following point (that shared distance is the arc radius) — a straight
// vertex is not. Proven on a rounded rectangle (4 lines + 4 fillet arcs) and a standalone arc.
func chainEntities(ordered []Point) ([]Line, []Arc) {
	if len(ordered) < 2 {
		return nil, nil
	}
	var lines []Line
	var arcs []Arc
	start := ordered[0]
	for i := 1; i < len(ordered); {
		q := ordered[i]
		if i+1 < len(ordered) && isArcCentre(start, q, ordered[i+1]) {
			end := ordered[i+1]
			arcs = append(arcs, Arc{Center: q, Start: start, End: end, Radius: dist(q, start)})
			start = end
			i += 2
			continue
		}
		lines = append(lines, Line{A: start, B: q})
		start = q
		i++
	}
	return lines, arcs
}

// isArcCentre reports whether c is the centre of an arc whose endpoints are the running start and
// the next point: both must sit at the same non-zero radius from c.
func isArcCentre(start, c, next Point) bool {
	r := dist(c, start)
	return r > 1e-9 && math.Abs(dist(c, next)-r) <= 1e-7
}

// singleArc recognises a standalone arc: exactly three distinct points where one is equidistant
// from the other two (the centre). The remaining two are the endpoints in first-seen order
// (start then end). Returns false for anything else (a full circle is two points, handled apart).
func singleArc(pts []Point) (Arc, bool) {
	if len(pts) != 3 {
		return Arc{}, false
	}
	for i, c := range pts {
		a, b := pts[(i+1)%3], pts[(i+2)%3]
		r := dist(c, a)
		if r > 1e-9 && math.Abs(dist(c, b)-r) <= 1e-7 {
			return Arc{Center: c, Start: a, End: b, Radius: r}, true
		}
	}
	return Arc{}, false
}

func dist(a, b Point) float64 { return math.Hypot(a.X-b.X, a.Y-b.Y) }

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
