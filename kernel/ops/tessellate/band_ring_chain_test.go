// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// choredTorusBandFace builds a CORRECT torus quarter-band (an outer rim fillet, v∈[0,π/2]) whose
// FULL-circle footprint rim (v=0) is subdivided into n straight chords — a full-360° chain of
// geom.LineSegment sub-edges, mimicking the Task-1 setback rim rebuild. It returns the intact torus
// *topo.Face plus the analytic quarter-band area 2π·minor·(major·π/2 + minor) (the outer quarter,
// cos v ≥ 0). Its top rim stays a single closed circle, so bandRingsAndSeam's single-edge path finds
// one ring and the chorded-rim fallback (chainBoundaryRings) must supply the second.
func choredTorusBandFace(t *testing.T, majorR, minorR float64, n int) (*topo.Face, float64) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "chordtorus", 0))
	tor, err := geom.NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), majorR, minorR)
	if err != nil {
		t.Fatal(err)
	}
	f := buildChordedTorusBand(t, tor, n, lin)
	area := 2 * stdmath.Pi * minorR * (majorR*stdmath.Pi/2 + minorR)
	return f, area
}

// buildChordedTorusBand assembles the torus band face: n footprint chord edges (v=0), one closed
// top-rim circle edge (v=π/2), and the meridian seam arc (v:0→π/2 at u=0) used twice by the loop.
func buildChordedTorusBand(t *testing.T, tor geom.Torus, n int, lin topo.Lineage) *topo.Face {
	t.Helper()
	bld := topo.NewBuilder(false, lin)
	foot := torusRingPoints(tor, 0, n)
	topP0 := tor.PointAt(0, stdmath.Pi/2)
	vFoot := make([]*topo.Vertex, n)
	for i, p := range foot {
		vFoot[i] = bld.AddVertex(p, lin)
	}
	vTop := bld.AddVertex(topP0, lin)
	seamArc, err := geom.Arc3dByThreePoints(foot[0], tor.PointAt(0, stdmath.Pi/4), topP0)
	if err != nil {
		t.Fatal(err)
	}
	seam := bld.AddEdge(seamArc, vFoot[0], vTop, lin)
	top := bld.AddEdge(topRimCircle(tor), vTop, vTop, lin)
	uses := []topo.Use{topo.Fwd(seam), topo.Rev(top), topo.Rev(seam)}
	for i := range n {
		j := (i + 1) % n
		e := bld.AddEdge(geom.NewLineSegment(foot[i], foot[j]), vFoot[i], vFoot[j], lin)
		uses = append(uses, topo.Fwd(e))
	}
	bld.AddFace(tor, lin, topo.OuterLoop(uses...))
	return bld.Build().Faces()[0]
}

// topRimCircle is the torus band's top rim (v=π/2): a full circle of radius Major centred one
// minor radius up the axis, sharing the torus frame so its angle-0 point is tor.PointAt(0,π/2).
func topRimCircle(tor geom.Torus) geom.Circle {
	center := tor.Center.TranslateBy(tor.AxisDir.AsVector().Scale(math.Scalar(tor.MinorRadius)))
	return geom.Circle{Center: center, Normal: tor.AxisDir, RefDir: tor.Ref, Radius: tor.MajorRadius}
}

// torusRingPoints samples n points around the torus at tube parameter v, u_k = 2πk/n.
func torusRingPoints(tor geom.Torus, v float64, n int) []math.Point3 {
	pts := make([]math.Point3, n)
	for i := range n {
		pts[i] = tor.PointAt(2*stdmath.Pi*float64(i)/float64(n), v)
	}
	return pts
}

// TestChainBoundaryRings_ChordedTorusMeshesAsBand proves the chained-ring fallback engages: a chorded
// full-360° footprint rim must mesh to the quarter-band area (not the full donut 4π²·R·r), watertight.
func TestChainBoundaryRings_ChordedTorusMeshesAsBand(t *testing.T) {
	t.Parallel()
	f, wantArea := choredTorusBandFace(t, 20, 5, 30)
	if _, isTorus := f.Geometry().(geom.Torus); !isTorus {
		t.Fatalf("synthesized face is not a torus — it will not route through bandRingsAndSeam")
	}
	m := TessellateFace(f, DefaultQuality())
	got := meshArea(m)
	if stdmath.Abs(got-wantArea)/wantArea > 0.01 {
		t.Fatalf("chorded torus band meshed to %.3f, want ≈%.3f (±1%%) — full-donut %.1f means the chained-ring fallback did not engage",
			got, wantArea, 4*stdmath.Pi*stdmath.Pi*20*5)
	}
	if !meshIsWatertight(m) {
		t.Fatalf("chorded torus band mesh is not watertight (boundary is not exactly the two rim loops)")
	}
}

// TestChainBoundaryRings_AcceptsChordedRim asserts the fallback directly: chainBoundaryRings recovers
// exactly two congruent rings plus a ≥2-point seam from the chorded band face.
func TestChainBoundaryRings_AcceptsChordedRim(t *testing.T) {
	t.Parallel()
	f, _ := choredTorusBandFace(t, 20, 5, 30)
	rings, seamN, _, ok := chainBoundaryRings(f, DefaultQuality())
	if !ok || len(rings) != 2 || seamN < 2 {
		t.Fatalf("chainBoundaryRings(chorded) = rings %d, seamN %d, ok %v; want 2 rings, seamN≥2, ok", len(rings), seamN, ok)
	}
}

// TestChainBoundaryRings_RejectsNonCongruentRings proves the congruence gate honest-rejects the
// pre-Task-1 defect: a full-circle rim vs a 118° out-and-back footprint slit (|U|/V≈0 on the slit),
// while accepting two full circles. Exercised directly on ringsCongruent because a genuine
// out-and-back slit face cannot be chained into a clean ring (it collides in the head-to-tail trace),
// so the face-level path rejects before reaching the gate — the gate math is proven here instead.
func TestChainBoundaryRings_RejectsNonCongruentRings(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 20, 5)
	if err != nil {
		t.Fatal(err)
	}
	full := torusRingPoints(tor, 0, 30)
	slit := doubledSlitRing(tor, 151.6, 270.0)
	if ringsCongruent(tor, full, slit) {
		t.Fatalf("congruence gate accepted a full-circle vs 118°-slit ring pair — it must reject (|U|/V≈0 on the slit)")
	}
	if !ringsCongruent(tor, full, torusRingPoints(tor, 0, 24)) {
		t.Fatalf("congruence gate rejected two full-circle rings — a clean band must be accepted")
	}
}

// doubledSlitRing builds the malformed footprint: points across u∈[loDeg,hiDeg] then back (out and
// back), so the signed advance U≈0 while the variation V≈2·range — the spike's 118° doubled slit.
func doubledSlitRing(tor geom.Torus, loDeg, hiDeg float64) []math.Point3 {
	const k = 12
	var pts []math.Point3
	for i := 0; i <= k; i++ {
		deg := loDeg + (hiDeg-loDeg)*float64(i)/float64(k)
		pts = append(pts, tor.PointAt(deg*stdmath.Pi/180, 0))
	}
	for i := k - 1; i >= 0; i-- {
		deg := loDeg + (hiDeg-loDeg)*float64(i)/float64(k)
		pts = append(pts, tor.PointAt(deg*stdmath.Pi/180, 0))
	}
	return pts
}

// meshIsWatertight reports whether a single lofted band mesh's boundary is exactly closed rim loops:
// every boundary vertex (incident to a free edge, one not shared by two triangles) has exactly two
// free edges. An interior crack or T-junction leaves a boundary vertex with a free-edge degree ≠ 2.
func meshIsWatertight(m *Mesh) bool {
	weld := weldVertexIndex(m.Positions)
	tri := map[[2]int]int{}
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		v := [3]int{weld[m.Indices[3*t]], weld[m.Indices[3*t+1]], weld[m.Indices[3*t+2]]}
		for k := range 3 {
			tri[undirectedEdge(v[k], v[(k+1)%3])]++
		}
	}
	freeDeg := map[int]int{}
	for e, d := range tri {
		if d != 2 {
			freeDeg[e[0]]++
			freeDeg[e[1]]++
		}
	}
	for _, d := range freeDeg {
		if d != 2 {
			return false
		}
	}
	return len(freeDeg) > 0
}

// weldVertexIndex maps each position to a canonical index, merging coincident vertices (per-face
// tessellations copy shared-edge vertices). A µm-scale quantum matches freeEdgeCount's weld.
func weldVertexIndex(pos []math.Point3) []int {
	canon := map[[3]int64]int{}
	weld := make([]int, len(pos))
	for i, p := range pos {
		k := [3]int64{quantMicron(float64(p.X)), quantMicron(float64(p.Y)), quantMicron(float64(p.Z))}
		if c, ok := canon[k]; ok {
			weld[i] = c
		} else {
			canon[k], weld[i] = i, i
		}
	}
	return weld
}

func quantMicron(x float64) int64 {
	if x < 0 {
		return int64(x*1e6 - 0.5)
	}
	return int64(x*1e6 + 0.5)
}

func undirectedEdge(a, b int) [2]int {
	if a > b {
		return [2]int{b, a}
	}
	return [2]int{a, b}
}

// meshArea sums the triangle areas of a mesh.
func meshArea(m *Mesh) float64 {
	var sum float64
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a, b, c := m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]]
		sum += a.VectorTo(b).Cross(a.VectorTo(c)).Length() / 2
	}
	return sum
}
