// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// squareSeat builds a half×half seat square in the XY plane (z=0) as a curvedFace plus the planeUV loops.
func squareSeat(t *testing.T, half float64) (geom.Plane, curvedFace, [][]math.Point3) {
	t.Helper()
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	h := math.Scalar(half)
	ring := []math.Point3{math.P3(-h, -h, 0), math.P3(h, -h, 0), math.P3(h, h, 0), math.P3(-h, h, 0)}
	var edges []loopEdge
	for i := range 4 {
		edges = append(edges, loopEdge{curve: geom.NewLineSegment(ring[i], ring[(i+1)%4]), t0: 0, t1: 1})
	}
	return pl, curvedFace{surface: pl, loops: []curvedLoop{{edges: edges}}}, [][]math.Point3{ring}
}

// curvedFaceUVArea sums the shoelace area of a curvedFace's loops in the plane chart (outer positive, holes
// negative). The chart is an isometry, so this is the true 3D face area. Each edge is densely sampled so a
// conic arc contributes its real area, not a chord's.
func curvedFaceUVArea(pl geom.Plane, f curvedFace) float64 {
	total := 0.0
	for _, loop := range f.loops {
		var poly []math.Point2
		for _, e := range loop.edges {
			for i := range 64 {
				tt := e.t0 + (e.t1-e.t0)*float64(i)/64
				poly = append(poly, to2D(pl, e.curve.PointAt(tt)))
			}
		}
		total += shoelaceArea(poly)
	}
	return total
}

func shoelaceArea(poly []math.Point2) float64 {
	a := 0.0
	for i, n := 0, len(poly); i < n; i++ {
		p, q := poly[i], poly[(i+1)%n]
		a += float64(p.X)*float64(q.Y) - float64(q.X)*float64(p.Y)
	}
	return stdmath.Abs(a) / 2
}

// TestPlaneUVTrimsSquareBittenByCircle is the Slice A/2 milestone: a radius-2 imprint circle centred ON the
// right edge of a 6×6 seat square bites out the half-disc that lies inside the square (area 2π). The trimmed
// seat face must be one connected region of area 36−2π, its boundary the 3 square sides + the exact conic arc
// — proving the planeUV operand arranges a boundary-clipping conic through the shared (u,v) trimmer (#1591).
func TestPlaneUVTrimsSquareBittenByCircle(t *testing.T) {
	t.Parallel()
	pl, seat, ring3 := squareSeat(t, 3)
	circ, _ := geom.NewCircle(math.P3(3, 0, 0), math.V3(0, 0, 1), 2) // centred on the x=3 edge
	seatUV := seatLoopsUV(pl, ring3)
	c := &planeUV{
		plane:  pl,
		seatUV: seatUV,
		seat3D: ring3,
		res:    geom.ResolutionForSize(6),
		inTool: func(p math.Point3) bool { // inside the vertical drill cylinder through (3,0)
			return stdmath.Hypot(float64(p.X)-3, float64(p.Y)) < 2-1e-9
		},
	}
	faces, _, err := trimByImprint(c, seat, pl, []geom.Curve3{circ}, planeMaterial(c))
	if err != nil {
		t.Fatalf("trimByImprint: %v", err)
	}
	if len(faces) != 1 {
		t.Fatalf("kept region has %d faces, want 1 connected seat", len(faces))
	}
	got := curvedFaceUVArea(pl, faces[0])
	want := 36 - 2*stdmath.Pi
	if e := stdmath.Abs(got-want) / want; e > 0.01 {
		t.Errorf("trimmed seat area %.5f, want %.5f (36−2π); rel %.4f", got, want, e)
	}
}

// TestPlaneUVInjectsExactCrossings is the watertightness linchpin (#1591, ADR-0049 D-d): a crossing that
// falls BETWEEN conic samples (circle centred at (2,0) crosses the seat edge x=3 at y=±√3, a non-sample
// parameter) must still re-emit the seat arc terminating EXACTLY on the edge — within a weld, not the
// ~1e-4 sagitta a sampled crossing leaves. Without the exact-crossing injection the arc misses the edge and
// the tool wall's split base cannot weld.
func TestPlaneUVInjectsExactCrossings(t *testing.T) {
	t.Parallel()
	pl, seat, ring3 := squareSeat(t, 3)
	circ, _ := geom.NewCircle(math.P3(2, 0, 0), math.V3(0, 0, 1), 2) // crosses x=3 at y=±√3, between samples
	seatUV := seatLoopsUV(pl, ring3)
	c := &planeUV{plane: pl, seatUV: seatUV, seat3D: ring3, res: geom.ResolutionForSize(6),
		inTool: func(p math.Point3) bool { return stdmath.Hypot(float64(p.X)-2, float64(p.Y)) < 2-1e-9 }}
	faces, _, err := trimByImprint(c, seat, pl, []geom.Curve3{circ}, planeMaterial(c))
	if err != nil {
		t.Fatal(err)
	}
	weld := geom.ResolutionForSize(6).Weld()
	found := 0
	for _, loop := range faces[0].loops {
		for _, e := range loop.edges {
			if _, ok := e.curve.(geom.Circle); !ok {
				continue
			}
			for _, p := range []math.Point3{e.start(), e.end()} {
				found++
				if dx := stdmath.Abs(float64(p.X) - 3); dx > weld {
					t.Errorf("seat conic arc endpoint x=%.10f is %.2e off the seat edge x=3 (weld=%.1e) — no exact-crossing injection", float64(p.X), dx, weld)
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("no conic arc edge found on the trimmed seat")
	}
}

// seatLoopsUV projects 3D seat loops into the plane chart (a test mirror of the assembler's helper).
func seatLoopsUV(pl geom.Plane, loops [][]math.Point3) [][]math.Point2 {
	out := make([][]math.Point2, len(loops))
	for i, ring := range loops {
		uv := make([]math.Point2, len(ring))
		for j, p := range ring {
			uv[j] = to2D(pl, p)
		}
		out[i] = uv
	}
	return out
}
