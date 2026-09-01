// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The exact pass-face interaction proofs (ADR-0058, #2247). A pass-through face whose box overlaps the
// other operand is cleared — or reported interacting — by a closed form over both TRIMS (the OCCT
// IntTools_FaceFace shape), never by sampling. The decisive case is a tool INSIDE a cylinder: its box
// always overlaps the wall's box while the tool never touches the wall.

// mustPlane builds a plane through p with normal n, failing the test if the normal is degenerate.
func mustPlane(t *testing.T, p math.Point3, n math.Vector3) geom.Plane {
	t.Helper()
	pl, err := geom.NewPlane(p, n)
	if err != nil {
		t.Fatalf("NewPlane(%v, %v): %v", p, n, err)
	}
	return pl
}

// rectFace builds a rectangle in the plane spanned by the unit-length axes ea/eb, centred at c and
// half-sized (ha, hb) — the polygonal tool the pass proofs are stated against. It goes through the
// planarFaceFromRings constructor so the loop edges carry the exact ring vertices planarRings reads.
func rectFace(t *testing.T, c math.Point3, ea, eb math.Vector3, ha, hb float64) curvedFace {
	t.Helper()
	corner := func(sa, sb float64) math.Point3 {
		return c.TranslateBy(ea.Scale(math.Scalar(sa * ha))).TranslateBy(eb.Scale(math.Scalar(sb * hb)))
	}
	ring := []math.Point3{corner(-1, -1), corner(1, -1), corner(1, 1), corner(-1, 1)}
	return planarFaceFromRings(mustPlane(t, c, ea.Cross(eb)), [][]math.Point3{ring}, topo.Lineage{})
}

// discPassFace is a z=0 disc of radius r: a planar pass face with a full-circle boundary, the trim the
// polygonal interval test must respect exactly.
func discPassFace(t *testing.T, r float64) curvedFace {
	t.Helper()
	circ, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	pl := mustPlane(t, math.P3(0, 0, 0), math.V3(0, 0, 1))
	return curvedFace{surface: pl, loops: []curvedLoop{{edges: []loopEdge{fullRimEdge(circ, false)}}}}
}

// cylinderWallFace builds the full side band of a radius-r cylinder over z ∈ [0, h] — one loop carrying
// both full-circle rims, the shape fullCylinderSideBand recovers.
func cylinderWallFace(t *testing.T, r, h float64) curvedFace {
	t.Helper()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	bot, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatalf("NewCircle bottom: %v", err)
	}
	top, err := geom.NewCircle(math.P3(0, 0, math.Scalar(h)), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatalf("NewCircle top: %v", err)
	}
	edges := []loopEdge{fullRimEdge(bot, false), fullRimEdge(top, true)}
	return curvedFace{surface: cyl, loops: []curvedLoop{{edges: edges}}}
}

// fullRimEdge wraps a full circle as one closed loop edge, carrying the seam point as both endpoints —
// the shape the split pipeline emits for a rim (reversed walks the circle from 1 back to 0).
func fullRimEdge(c geom.Circle, reversed bool) loopEdge {
	t0, t1 := 0.0, 1.0
	if reversed {
		t0, t1 = 1, 0
	}
	seam := c.PointAt(0)
	return loopEdge{curve: c, t0: t0, t1: t1, v0: seam, v1: seam}
}

// TestPassPairDeclinesCurvedTool: the proofs are stated against a POLYGONAL tool trim, so a curve-edged
// other face has no closed form here and stays conservative (interacting).
func TestPassPairDeclinesCurvedTool(t *testing.T) {
	t.Parallel()
	if passPairClear(discPassFace(t, 2), discPassFace(t, 3)) {
		t.Error("passPairClear cleared a curve-edged tool face; want the conservative decline")
	}
}

// TestPlanePassClearLineMissesTool: the plane∩plane line runs outside the tool's trim, so the pair is
// clear whatever the pass face contains — the tool interval set is empty.
func TestPlanePassClearLineMissesTool(t *testing.T) {
	t.Parallel()
	// Tool: a square in the x=10 plane lifted to z ∈ [5,6]; the shared line is z=0, x=10.
	tool := rectFace(t, math.P3(10, 0, 5.5), math.V3(0, 1, 0), math.V3(0, 0, 1), 1, 0.5)
	if !planePassClear(discPassFace(t, 2), tool) {
		t.Error("planePassClear reported contact for a line that misses the tool trim")
	}
}

// TestPlanePassClearIntervalsDisjoint: the shared line crosses BOTH trims, but on disjoint spans — the
// chord of the disc is y ∈ ±√(r²−x²) ≈ ±1.936, the tool sits at y ∈ [5,6].
func TestPlanePassClearIntervalsDisjoint(t *testing.T) {
	t.Parallel()
	tool := rectFace(t, math.P3(0.5, 5.5, 0), math.V3(0, 1, 0), math.V3(0, 0, 1), 0.5, 1)
	if !planePassClear(discPassFace(t, 2), tool) {
		t.Error("planePassClear reported contact for disjoint intervals on the shared line")
	}
}

// TestPlanePassInteractsOverlappingIntervals: the same tool slid onto the disc's chord overlaps the pass
// face's interval, which is a real contact — the pass face may not be carried through untouched. This is
// also the regression for the narrow-window parity defect: the tool's span (y ∈ [-1,1]) lies WHOLLY
// inside the disc's chord (±1.936), so a conic solver bounded by the tool's own range saw no crossing at
// all and reported the pass face clear. curvedFaceLineIntervals now always solves over the face's extent.
func TestPlanePassInteractsOverlappingIntervals(t *testing.T) {
	t.Parallel()
	tool := rectFace(t, math.P3(0.5, 0, 0), math.V3(0, 1, 0), math.V3(0, 0, 1), 1, 1)
	if planePassClear(discPassFace(t, 2), tool) {
		t.Error("planePassClear cleared a tool crossing the disc's chord")
	}
}

// TestPlanePassClearParallelPlanes: parallel planes share no line; only a coplanar pair is a contact.
func TestPlanePassClearParallelPlanes(t *testing.T) {
	t.Parallel()
	above := rectFace(t, math.P3(0, 0, 5), math.V3(1, 0, 0), math.V3(0, 1, 0), 4, 4)
	if !planePassClear(discPassFace(t, 2), above) {
		t.Error("planePassClear reported contact between parallel planes 5 apart")
	}
}

// TestCylinderPassClearToolInsideWall is the motivating case (#2247): a tool wholly inside the bore
// overlaps the wall's box on every axis, yet the section circle lies outside the tool's trim.
func TestCylinderPassClearToolInsideWall(t *testing.T) {
	t.Parallel()
	inside := rectFace(t, math.P3(0, 0, 2), math.V3(1, 0, 0), math.V3(0, 1, 0), 0.5, 0.5)
	if !cylinderPassClear(cylinderWallFace(t, 2, 4), inside) {
		t.Error("cylinderPassClear reported contact for a tool strictly inside the bore")
	}
}

// TestCylinderPassInteractsToolCrossingWall: the same section circle now crosses the tool's edges inside
// the band — a real wall cut, which must not pass through.
func TestCylinderPassInteractsToolCrossingWall(t *testing.T) {
	t.Parallel()
	across := rectFace(t, math.P3(0, 0, 2), math.V3(1, 0, 0), math.V3(0, 1, 0), 3, 0.5)
	if cylinderPassClear(cylinderWallFace(t, 2, 4), across) {
		t.Error("cylinderPassClear cleared a tool whose section circle crosses the wall")
	}
}

// TestCylinderPassInteractsTangentToolEdge: a tool edge exactly tangent to the section circle gives no
// sound parity, so the grazing contact is reported as interacting.
func TestCylinderPassInteractsTangentToolEdge(t *testing.T) {
	t.Parallel()
	tangent := rectFace(t, math.P3(3, 0, 2), math.V3(1, 0, 0), math.V3(0, 1, 0), 1, 1)
	if cylinderPassClear(cylinderWallFace(t, 2, 4), tangent) {
		t.Error("cylinderPassClear cleared a tool edge tangent to the section circle")
	}
}

// TestCylinderPassClearRulingsAboveBand: a tool plane parallel to the axis sections the cylinder in a
// LINE PAIR; the tool's intervals on those rulings sit above the wall's band, so the pair is clear.
func TestCylinderPassClearRulingsAboveBand(t *testing.T) {
	t.Parallel()
	above := rectFace(t, math.P3(0, 0, 11), math.V3(1, 0, 0), math.V3(0, 0, 1), 3, 1)
	if !cylinderPassClear(cylinderWallFace(t, 2, 4), above) {
		t.Error("cylinderPassClear reported contact for rulings clipped above the wall band")
	}
}

// TestCylinderPassInteractsRulingsInBand: the same tool lowered onto the band crosses the rulings inside
// it — the wall is really cut.
func TestCylinderPassInteractsRulingsInBand(t *testing.T) {
	t.Parallel()
	through := rectFace(t, math.P3(0, 0, 2), math.V3(1, 0, 0), math.V3(0, 0, 1), 3, 1)
	if cylinderPassClear(cylinderWallFace(t, 2, 4), through) {
		t.Error("cylinderPassClear cleared a tool crossing the wall's rulings inside the band")
	}
}

// TestCylinderPassClearEllipseAboveBand: an oblique tool sections the cylinder in an ELLIPSE. The tool
// encloses it, so the decision is the conic's axial span (centre ± half-amplitude, r for a 45° plane)
// against the band — here 9 to 13 against [0,4].
func TestCylinderPassClearEllipseAboveBand(t *testing.T) {
	t.Parallel()
	oblique := rectFace(t, math.P3(0, 0, 11), math.V3(1, 0, 0), math.V3(0, 1, 1).AsUnit().AsVector(), 6, 6)
	if !cylinderPassClear(cylinderWallFace(t, 2, 4), oblique) {
		t.Error("cylinderPassClear reported contact for an ellipse section above the wall band")
	}
}
