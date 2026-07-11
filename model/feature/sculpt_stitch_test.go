// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/sketch"
)

// cubeFaceBody builds one outward-oriented quad surface body of the unit cube.
func cubeFaceBody(feat string, p0, p1, p2, p3 math.Point3) *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	surf, _ := geom.NewPlane(p0, p0.VectorTo(p1).Cross(p1.VectorTo(p2)))
	pts := []math.Point3{p0, p1, p2, p3}
	v := make([]*topo.Vertex, 4)
	for i, p := range pts {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	uses := make([]topo.Use, 4)
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(pts[i], pts[j]), v[i], v[j], topo.NewLineage(topo.Tok(feat, "edge", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// unitCubeFaces returns the six outward-oriented quad surface bodies of the unit cube.
func unitCubeFaces() []*topo.Body {
	p := math.P3
	return []*topo.Body{
		cubeFaceBody("bottom", p(0, 0, 0), p(0, 1, 0), p(1, 1, 0), p(1, 0, 0)),
		cubeFaceBody("top", p(0, 0, 1), p(1, 0, 1), p(1, 1, 1), p(0, 1, 1)),
		cubeFaceBody("front", p(0, 0, 0), p(1, 0, 0), p(1, 0, 1), p(0, 0, 1)),
		cubeFaceBody("back", p(0, 1, 0), p(0, 1, 1), p(1, 1, 1), p(1, 1, 0)),
		cubeFaceBody("left", p(0, 0, 0), p(0, 0, 1), p(0, 1, 1), p(0, 1, 0)),
		cubeFaceBody("right", p(1, 0, 0), p(1, 1, 0), p(1, 1, 1), p(1, 0, 1)),
	}
}

func TestStitchFeatureClosesSurfacesIntoSolid(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(unitCubeFaces()...)
	pf := NewStitchFeatures(fs).Add(0, false)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("stitch went unhealthy: %+v", pf.Health())
	}
	if len(fs.Result()) != 1 {
		t.Fatalf("stitch result has %d bodies, want 1", len(fs.Result()))
	}
	body := fs.Result()[0]
	if !body.IsSolid() {
		t.Error("stitching the closed cube faces should yield a solid")
	}
	if got := len(ops.BoundaryEdges(body)); got != 0 {
		t.Errorf("stitched solid has %d boundary edges, want 0", got)
	}
	if r := ops.Validate(body); !r.Valid {
		t.Errorf("stitched solid failed validation: %+v", r)
	}
}

func TestStitchFeatureMaintainAsSurface(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(unitCubeFaces()...)
	pf := NewStitchFeatures(fs).Add(0, true)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("stitch went unhealthy: %+v", pf.Health())
	}
	if fs.Result()[0].IsSolid() {
		t.Error("MaintainAsSurface should keep the quilt a surface body")
	}
}

func TestKnitIsStitchAlias(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(unitCubeFaces()...)
	pf := NewKnitFeatures(fs).Add(0, false)
	fs.Recompute()
	if pf.Kind() != "stitch" {
		t.Errorf("knit kind = %q, want stitch (alias)", pf.Kind())
	}
	if !fs.Result()[0].IsSolid() {
		t.Error("knit of closed surfaces should yield a solid")
	}
}

func TestSculptFillsBoundedVolume(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(unitCubeFaces()...)
	pf := NewSculptFeatures(fs).Add(ops.NewBody, 0)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("sculpt went unhealthy: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if !body.IsSolid() {
		t.Error("sculpt should fill the bounded volume into a solid")
	}
	// The filled volume spans the unit cube.
	if d := body.RangeBox().Diagonal(); !approxEq(d.X, 1) || !approxEq(d.Y, 1) || !approxEq(d.Z, 1) {
		t.Errorf("sculpt box diagonal = %v, want (1,1,1)", d)
	}
}

// TestSculptDirectedFillsVolume is #1881: with an explicit direction per bounding surface (keep the
// inside of each outward cube face), sculpt intersects the directed halfspaces into the unit solid.
func TestSculptDirectedFillsVolume(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(unitCubeFaces()...)
	pf := NewSculptFeatures(fs).AddSculpt(&SculptDefinition{Operation: ops.NewBody, Directions: make([]bool, 6)})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("directed sculpt went unhealthy: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if !body.IsSolid() {
		t.Fatal("directed sculpt should fill a solid")
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; !approxEq(v, 1) {
		t.Errorf("directed sculpt volume = %g, want 1 (unit cube)", v)
	}
}

// TestSculptBodySelectionKeepsOthers is #1881: sculpting only the selected bounding surfaces leaves
// an unselected body untouched alongside the new sculpted solid.
func TestSculptBodySelectionKeepsOthers(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(unitCubeFaces()...)                                                          // bodies 0..5: cube faces
	NewBaseFeatures(fs).AddBase(buildPrism(squarePoly(10), sketch.XYPlane(), span{near: 0, far: 1}, 0, "b")) // body 6: a far solid
	pf := NewSculptFeatures(fs).AddSculpt(&SculptDefinition{Operation: ops.NewBody, BodyIndices: []int{0, 1, 2, 3, 4, 5}, Directions: make([]bool, 6)})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("body-selected sculpt sick: %+v", pf.Health())
	}
	if got := len(fs.Result()); got != 2 {
		t.Fatalf("result = %d bodies, want 2 (the far solid + the sculpted cube)", got)
	}
}

// TestSculptOptionsRoundTrip pins #1881 serialization: directions, body selection, and the affected
// index survive the recipe codec.
func TestSculptOptionsRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	affected := 2
	NewSculptFeatures(fs).AddSculpt(&SculptDefinition{
		Operation: ops.Cut, Tolerance: 0.01, Directions: []bool{true, false, true}, BodyIndices: []int{0, 1, 3}, AffectedIndex: &affected,
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if d := data[0].Sculpt; len(d.Directions) != 3 || len(d.BodyIndices) != 3 || d.AffectedIndex == nil || *d.AffectedIndex != 2 {
		t.Fatalf("serialized sculpt = %+v", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*SculptFeature).Definition()
	if len(def.Directions) != 3 || !def.Directions[0] || def.Directions[1] || len(def.BodyIndices) != 3 || def.AffectedIndex == nil || *def.AffectedIndex != 2 {
		t.Errorf("restored sculpt = %+v", def)
	}
}

func TestSculptGoesSickOnOpenSurfaces(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(unitCubeFaces()[1:]...) // drop one face → open
	pf := NewSculptFeatures(fs).Add(ops.NewBody, 0)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("sculpt of open surfaces = %v, want sick (no enclosed volume)", pf.Health().Status)
	}
}
