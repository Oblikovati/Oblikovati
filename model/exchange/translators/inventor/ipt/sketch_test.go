// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"testing"
)

// TestDecodeSketchRealPointMarker locks the generalization for real interactively-modelled
// parts: their sketch point nodes carry a schema marker at +12 that is NOT the 0x0C20 seen
// in API-authored corpus parts (it is 0x0A96, 0x0B5C, …). Keying on 0x0C20 found zero points
// on real parts, so a whole profile (e.g. a shaft's 7-line revolve loop) collapsed to nothing.
// This builds two point nodes with the real 0x0A96 marker plus a line that references them and
// asserts the line rebuilds from the resolved endpoints.
func TestDecodeSketchRealPointMarker(t *testing.T) {
	const realMarker = 0x00000A96
	seg := make([]byte, 0, 160)
	pointNode := func(id uint32, x, y float64) []byte {
		b := make([]byte, 40)
		binary.LittleEndian.PutUint32(b[0:], dcNodeTag)
		binary.LittleEndian.PutUint32(b[4:], id)
		binary.LittleEndian.PutUint32(b[8:], nullRef)
		binary.LittleEndian.PutUint32(b[12:], realMarker)
		binary.LittleEndian.PutUint32(b[16:], 0)
		binary.LittleEndian.PutUint32(b[20:], 0x80000019) // geometry tag (high bit)
		binary.LittleEndian.PutUint64(b[24:], math.Float64bits(x))
		binary.LittleEndian.PutUint64(b[32:], math.Float64bits(y))
		return b
	}
	seg = append(seg, pointNode(10, 1, 2)...)
	seg = append(seg, pointNode(20, 3, 4)...)
	line := make([]byte, 48)
	binary.LittleEndian.PutUint32(line[0:], 0x80000019)  // node-type pre-tag at curveHead-4
	binary.LittleEndian.PutUint32(line[4:], curveMarker) // curve header: 0x30000002 | w1 | w2 | 0x10
	binary.LittleEndian.PutUint32(line[8:], 2)
	binary.LittleEndian.PutUint32(line[12:], 2)
	binary.LittleEndian.PutUint32(line[16:], curveTailMark)
	binary.LittleEndian.PutUint32(line[20:], 0x8000000A) // refA at curveHead+16 -> id 10
	binary.LittleEndian.PutUint32(line[24:], 0x80000014) // refB at curveHead+20 -> id 20
	seg = append(seg, line...)

	s := DecodeSketch(seg)
	if len(s.Lines) != 1 || len(s.Points) != 0 {
		t.Fatalf("got %d lines, %d points; want 1 line, 0 points (%+v)", len(s.Lines), len(s.Points), s)
	}
	got := s.Lines[0]
	if absf(got.A.X-1) > 1e-9 || absf(got.A.Y-2) > 1e-9 || absf(got.B.X-3) > 1e-9 || absf(got.B.Y-4) > 1e-9 {
		t.Errorf("line = %+v, want (1,2)-(3,4)", got)
	}
}

// TestDecodeSketchAcceptsFlaggedOriginPoint locks the M1 point-set completion: a geometry point
// whose +16 flag word is nonzero (0x40 for the sketch origin, 0x8000 for some vertices) must
// still be collected. The earlier gate required +16 == 0 and silently dropped the origin, forcing
// a fragile off-by-one synthesis; now the coarse gate keys on the +12 marker's high bit (a small
// enum for a geometry carrier). It also asserts a phantom node — a real node-type tag is 0x800000XX,
// so a word like 0xFFFF9673 with only the high bit set at +20 must NOT read as a spurious point.
func TestDecodeSketchAcceptsFlaggedOriginPoint(t *testing.T) {
	const marker = 0x00000A96
	point := func(id, flag, tag uint32, x, y float64) []byte {
		b := make([]byte, 40)
		binary.LittleEndian.PutUint32(b[0:], dcNodeTag)
		binary.LittleEndian.PutUint32(b[4:], id)
		binary.LittleEndian.PutUint32(b[8:], nullRef)
		binary.LittleEndian.PutUint32(b[12:], marker)
		binary.LittleEndian.PutUint32(b[16:], flag) // +16 flag: 0x40 on the origin
		binary.LittleEndian.PutUint32(b[20:], tag)
		binary.LittleEndian.PutUint64(b[24:], math.Float64bits(x))
		binary.LittleEndian.PutUint64(b[32:], math.Float64bits(y))
		return b
	}
	var seg []byte
	seg = append(seg, point(10, 0x40, 0x80000017, 0, 0)...)     // origin, flagged +16
	seg = append(seg, point(11, 0x8000, 0x80000017, 2.5, 0)...) // flagged vertex
	seg = append(seg, point(12, 0, 0xFFFF9673, 9.9, 9.9)...)    // phantom: bad node tag at +20
	s := DecodeSketch(seg)
	if len(s.Points) != 2 {
		t.Fatalf("got %d points, want 2 (flagged origin + vertex, phantom rejected): %+v", len(s.Points), s.Points)
	}
	if s.Points[0] != (Point2D{0, 0}) || s.Points[1] != (Point2D{2.5, 0}) {
		t.Errorf("points = %+v, want (0,0) and (2.5,0)", s.Points)
	}
}

// TestDecodeSketchCurveHeaderVariant locks the within-cluster de-fragmentation fix: a line's
// 16-byte curve header is 0x30000002 | w1 | w2 | 0x10, where w1,w2 are small per-endpoint counts —
// usually 2, but 3 (rarely 4) where extra geometry meets an endpoint. Keying on the old fixed
// 0x02,0x02 header dropped those lines (a shaft's axis edge and an interior segment), leaving an
// OPEN profile that resolved by count yet revolved wrong. This builds two points and a line whose
// header carries w1=3,w2=3 and asserts the line still decodes from its resolved endpoints.
func TestDecodeSketchCurveHeaderVariant(t *testing.T) {
	const marker = 0x00000A96
	point := func(id uint32, x, y float64) []byte {
		b := make([]byte, 40)
		binary.LittleEndian.PutUint32(b[0:], dcNodeTag)
		binary.LittleEndian.PutUint32(b[4:], id)
		binary.LittleEndian.PutUint32(b[8:], nullRef)
		binary.LittleEndian.PutUint32(b[12:], marker)
		binary.LittleEndian.PutUint32(b[20:], 0x80000017)
		binary.LittleEndian.PutUint64(b[24:], math.Float64bits(x))
		binary.LittleEndian.PutUint64(b[32:], math.Float64bits(y))
		return b
	}
	seg := append(point(10, 1, 0), point(20, 3, 0)...)
	line := make([]byte, 48)
	binary.LittleEndian.PutUint32(line[0:], 0x80000017)  // node-type pre-tag at curveHead-4
	binary.LittleEndian.PutUint32(line[4:], curveMarker) // header: 0x30000002 | 3 | 3 | 0x10 (variant)
	binary.LittleEndian.PutUint32(line[8:], 3)
	binary.LittleEndian.PutUint32(line[12:], 3)
	binary.LittleEndian.PutUint32(line[16:], curveTailMark)
	binary.LittleEndian.PutUint32(line[20:], 0x8000000A) // refA -> id 10
	binary.LittleEndian.PutUint32(line[24:], 0x80000014) // refB -> id 20
	seg = append(seg, line...)

	s := DecodeSketch(seg)
	if len(s.Lines) != 1 {
		t.Fatalf("got %d lines, want 1 (w1=w2=3 header variant must decode): %+v", len(s.Lines), s)
	}
	if l := s.Lines[0]; absf(l.A.X-1) > 1e-9 || absf(l.B.X-3) > 1e-9 {
		t.Errorf("line = %+v, want (1,0)-(3,0)", l)
	}
}

func decodeSketchOf(t *testing.T, file string) Sketch {
	t.Helper()
	d := openDoc(t, file)
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		t.Fatalf("%s: no PmDCSegment", file)
	}
	return DecodeSketch(seg)
}

func TestDecodeSketchPoint(t *testing.T) {
	s := decodeSketchOf(t, "sketch_point.ipt")
	if len(s.Points) != 1 || len(s.Lines) != 0 || len(s.Circles) != 0 {
		t.Fatalf("got %d points, %d lines, %d circles; want 1/0/0", len(s.Points), len(s.Lines), len(s.Circles))
	}
	if p := s.Points[0]; absf(p.X-1) > 1e-9 || absf(p.Y-1) > 1e-9 {
		t.Errorf("point = (%.3f,%.3f) cm, want (1,1)", p.X, p.Y)
	}
}

func TestDecodeSketchLine(t *testing.T) {
	s := decodeSketchOf(t, "sketch_line.ipt")
	if len(s.Lines) != 1 || len(s.Points) != 0 {
		t.Fatalf("got %d lines, %d points; want 1 line, 0 points", len(s.Lines), len(s.Points))
	}
	l := s.Lines[0]
	// endpoints are (0,0) and (2,0) in some order
	span := absf(l.A.X-l.B.X) + absf(l.A.Y-l.B.Y)
	if absf(span-2) > 1e-9 {
		t.Errorf("line = %v..%v, want a 2 cm span along x", l.A, l.B)
	}
}

// TestDecodeSketchNonConvexLoop decodes the L-profile (a non-convex 6-corner polygon). A
// convex-ordering heuristic could not reproduce the notch; endpoint-reference resolution
// must. The reconstructed segment loop must enclose 6 cm^2 (4*1 + 1*2), proving the
// vertices are ordered by connectivity, not by angle about the centroid.
func TestDecodeSketchNonConvexLoop(t *testing.T) {
	d := openDoc(t, "18_lprofile.ipt")
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		t.Fatal("no PmDCSegment")
	}
	sketches := DecodeSketches(seg)
	var lines []Line
	for _, s := range sketches {
		lines = append(lines, s.Lines...)
	}
	if len(lines) != 6 {
		t.Fatalf("got %d line segments, want 6", len(lines))
	}
	if a := loopArea(lines); math.Abs(a-6) > 1e-6 {
		t.Errorf("enclosed area = %.4f cm^2, want 6 (notch not reconstructed)", a)
	}
}

// loopArea walks the (unordered) segment set into a single ring by matching shared
// endpoints, then returns the absolute shoelace area — independent of segment order or
// winding, so it validates connectivity rather than emission order.
func loopArea(lines []Line) float64 {
	ring := []Point2D{lines[0].A, lines[0].B}
	used := map[int]bool{0: true}
	for len(used) < len(lines) {
		tail := ring[len(ring)-1]
		progressed := false
		for i, l := range lines {
			if used[i] {
				continue
			}
			switch {
			case samePoint(l.A, tail):
				ring = append(ring, l.B)
			case samePoint(l.B, tail):
				ring = append(ring, l.A)
			default:
				continue
			}
			used[i] = true
			progressed = true
			break
		}
		if !progressed {
			return math.NaN() // segments do not form a single closed ring
		}
	}
	var a float64
	for i := range ring {
		j := (i + 1) % len(ring)
		a += ring[i].X*ring[j].Y - ring[j].X*ring[i].Y
	}
	return math.Abs(a) / 2
}

func samePoint(a, b Point2D) bool { return absf(a.X-b.X) < 1e-9 && absf(a.Y-b.Y) < 1e-9 }

func TestDecodeSketchCircle(t *testing.T) {
	s := decodeSketchOf(t, "sketch_circle.ipt")
	if len(s.Circles) != 1 || len(s.Points) != 0 {
		t.Fatalf("got %d circles, %d points; want 1 circle, 0 points", len(s.Circles), len(s.Points))
	}
	c := s.Circles[0]
	if absf(c.Center.X) > 1e-9 || absf(c.Center.Y) > 1e-9 || absf(c.Radius-1) > 1e-9 {
		t.Errorf("circle = center(%.3f,%.3f) r=%.3f, want center(0,0) r=1", c.Center.X, c.Center.Y, c.Radius)
	}
}

// TestResolveByRefsSynthesizesOrigin locks the connectivity fix for revolve profiles that touch
// their centreline at the sketch origin (0,0): the origin is an implicit point with no coordinate
// node, so the decoded points are one short of the references. resolveByRefs must synthesise it
// (lowest id → smallest reference) and rebuild the loop exactly, rather than fail into the convex
// fallback that scrambles a non-convex shaft profile.
func TestResolveByRefsSynthesizesOrigin(t *testing.T) {
	// A right triangle with its vertical edge on the axis through the origin. Decoded points:
	// (2,0) id20 and (2,2) id22; the origin (id 10) is referenced by two edges but not decoded.
	pts := []idPoint{{id: 20, p: Point2D{2, 0}}, {id: 22, p: Point2D{2, 2}}}
	lines := []lineRefs{
		{a: 10, b: 20, paired: true}, // origin -> (2,0)
		{a: 20, b: 22, paired: true}, // (2,0)  -> (2,2)
		{a: 22, b: 10, paired: true}, // (2,2)  -> origin
	}
	s, ok := resolveByRefs(pts, nil, lines, nil)
	if !ok || len(s.Lines) != 3 {
		t.Fatalf("resolveByRefs ok=%v, lines=%d; want ok, 3 lines", ok, len(s.Lines))
	}
	got := s.Lines[0] // ref (10,20) -> origin -> (2,0)
	if absf(got.A.X) > 1e-9 || absf(got.A.Y) > 1e-9 || absf(got.B.X-2) > 1e-9 || absf(got.B.Y) > 1e-9 {
		t.Errorf("first edge = %+v, want (0,0)-(2,0) with the synthesized origin", got)
	}
}
