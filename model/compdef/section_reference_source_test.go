// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// extrudeBlock builds a 4×3×5 block from a rectangle profile sketch and a distance extrude, so the
// resulting body is feature-backed (it survives Recompute and a recipe round-trip, unlike an
// injected body). Returns the part with one solid body after recompute.
func extrudeBlock(t *testing.T) *compdef.PartComponentDefinition {
	t.Helper()
	def := compdef.NewPartComponentDefinition()
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(4, 0))
	c2 := sk.Points().Add(math.P2(4, 3))
	c3 := sk.Points().Add(math.P2(0, 3))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("extrude produced %d bodies, want 1", def.SurfaceBodies().Count())
	}
	return def
}

// midHeightSketch adds a sketch on the XY plane at z=2.5, cutting the extruded block mid-height.
func midHeightSketch(t *testing.T, def *compdef.PartComponentDefinition) *sketch.Sketch {
	t.Helper()
	xy := sketch.XYPlane()
	plane, err := sketch.NewPlane(math.P3(0, 0, 2.5), xy.XAxis(), xy.YAxis())
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	return def.Sketches().Add(plane)
}

// curvedFaceKey returns the reference key of the body's first non-planar face (a cylinder's side).
func curvedFaceKey(t *testing.T, b *topo.Body) string {
	t.Helper()
	for _, f := range b.Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			return string(f.ReferenceKey())
		}
	}
	t.Fatal("body has no curved face")
	return ""
}

// projectionPoints returns a curve projection's reference-entity defining points in sketch space
// (ADR-0055 phase 3: the projection drives a concrete reference Line/Circle/Arc/Spline).
func projectionPoints(c *sketch.Projection) []math.Point2 {
	pts := sketch.DefiningPoints(c.Entity())
	out := make([]math.Point2, len(pts))
	for i, p := range pts {
		out[i] = p.Position()
	}
	return out
}

// projectedBounds returns the min/max 2D corner over every projection's reference geometry.
func projectedBounds(curves []*sketch.Projection) (lo, hi math.Point2) {
	lo = math.P2(stdmath.Inf(1), stdmath.Inf(1))
	hi = math.P2(stdmath.Inf(-1), stdmath.Inf(-1))
	for _, c := range curves {
		for _, p := range projectionPoints(c) {
			lo = math.P2(math.Scalar(stdmath.Min(float64(lo.X), float64(p.X))), math.Scalar(stdmath.Min(float64(lo.Y), float64(p.Y))))
			hi = math.P2(math.Scalar(stdmath.Max(float64(hi.X), float64(p.X))), math.Scalar(stdmath.Max(float64(hi.Y), float64(p.Y))))
		}
	}
	return lo, hi
}

// TestCutEdgeSourcesSectionBoxIntoUnitSquare: the z=0 section of a [-1,1]³ box is the unit square,
// projected onto the XY sketch as associative reference geometry (#1873 AC1).
func TestCutEdgeSourcesSectionBoxIntoUnitSquare(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	box, err := brep.SolidBlock(math.P3(-1, -1, -1), math.P3(1, 1, 1), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	def.SurfaceBodies().Add(box)
	sk := def.Sketches().Add(sketch.XYPlane())

	sources := def.CutEdgeSources(sk.Plane())
	if len(sources) == 0 {
		t.Fatal("cut-edge sources = 0, want the box's mid section loop")
	}
	curves := sk.ProjectCutEdges(sources)
	if len(curves) == 0 {
		t.Fatal("projected no cut-edge curves")
	}
	for _, c := range curves {
		if !c.Linked() {
			t.Error("a projected cut edge must be associative (linked)")
		}
	}
	lo, hi := projectedBounds(curves)
	if !lo.IsEqualTo(math.P2(-1, -1), 1e-6) || !hi.IsEqualTo(math.P2(1, 1), 1e-6) {
		t.Errorf("section bounds = %v..%v, want (-1,-1)..(1,1)", lo, hi)
	}
}

// TestCutEdgeSourcesEmptyWhenPlaneMissesBody: a sketch plane clear of the solid yields no cut edges.
func TestCutEdgeSourcesEmptyWhenPlaneMissesBody(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	box, _ := brep.SolidBlock(math.P3(-1, -1, -1), math.P3(1, 1, 1), "box")
	def.SurfaceBodies().Add(box)
	xy := sketch.XYPlane()
	far, _ := sketch.NewPlane(math.P3(0, 0, 100), xy.XAxis(), xy.YAxis()) // parallel to XY, well above the box

	if got := def.CutEdgeSources(far); len(got) != 0 {
		t.Errorf("cut-edge sources for a plane clear of the body = %d, want 0", len(got))
	}
}

// TestSilhouetteSourceProjectsCylinderRuling: the side of a Z-axis cylinder, viewed along +X (a YZ
// sketch normal), has silhouette rulings; the one nearest the proximity point projects as an
// associative reference curve (#1873 AC2).
func TestSilhouetteSourceProjectsCylinderRuling(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	def.SurfaceBodies().Add(cyl)
	sk := def.Sketches().Add(sketch.YZPlane()) // normal +X

	src, ok := def.SilhouetteSource(curvedFaceKey(t, cyl), sk.Plane(), math.P3(0, 2, 2.5), true)
	if !ok {
		t.Fatal("silhouette source reports no geometry, want the +Y ruling")
	}
	c := sk.ProjectCurve(src)
	if !c.Linked() || len(projectionPoints(c)) == 0 {
		t.Errorf("projected silhouette linked=%v points=%d, want linked with a polyline", c.Linked(), len(projectionPoints(c)))
	}
}

// TestSilhouetteSourceLostFaceReportsNoGeometry: an unknown face key yields no source geometry.
func TestSilhouetteSourceLostFaceReportsNoGeometry(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 5)
	def.SurfaceBodies().Add(cyl)
	sk := def.Sketches().Add(sketch.YZPlane())

	if _, ok := def.SilhouetteSource("no-such-face", sk.Plane(), math.P3(0, 2, 2.5), true); ok {
		t.Error("silhouette of a non-resolving face should report no geometry")
	}
}

// TestProjectedCutEdgesRebindOnReload: projected cut edges of a feature-backed solid round-trip and
// re-link (associative) after a save/reload — the extrude rebuilds on recompute and the "cutEdge"
// resolver case re-sections it, so the projections stay live (#1873 AC3).
func TestProjectedCutEdgesRebindOnReload(t *testing.T) {
	def := extrudeBlock(t)
	sk := midHeightSketch(t, def)
	before := sk.ProjectCutEdges(def.CutEdgeSources(sk.Plane()))
	if len(before) == 0 {
		t.Fatal("no cut edges projected to round-trip")
	}

	recipe, err := def.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	got := compdef.NewPartComponentDefinition()
	if err := got.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}

	restored := got.Sketches().Item(1).Projections() // sketch 0 is the profile; 1 is the cut
	for _, c := range restored {
		if !c.Linked() {
			t.Error("restored cut edge should re-link (rebind + recompute re-section the solid)")
		}
	}
	if len(restored) != len(before) {
		t.Errorf("restored %d projected cut edges, want %d", len(restored), len(before))
	}
}

// TestSilhouetteSourceIDRoundTripsThroughResolver: the silhouette descriptor survives encode →
// resolver rebuild, even when the face key contains the '|' field delimiter (#1873).
func TestSilhouetteSourceIDRoundTripsThroughResolver(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	plane := sketch.YZPlane()
	orig := compdef.NewSilhouetteRefSource(def, "face|with|pipes", plane, math.P3(1.5, -2, 3.25), true)

	rebuilt, ok := def.CurveProjectionSource(orig.SourceKind(), orig.SourceID(), plane)
	if !ok {
		t.Fatal("resolver did not rebuild the silhouette source")
	}
	if rebuilt.SourceID() != orig.SourceID() {
		t.Errorf("rebuilt SourceID = %q, want %q", rebuilt.SourceID(), orig.SourceID())
	}
}
