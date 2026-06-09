// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
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
	fs := NewPartFeatures(nil, nil)
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
	fs := NewPartFeatures(nil, nil)
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
	fs := NewPartFeatures(nil, nil)
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
	fs := NewPartFeatures(nil, nil)
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

func TestSculptGoesSickOnOpenSurfaces(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(unitCubeFaces()[1:]...) // drop one face → open
	pf := NewSculptFeatures(fs).Add(ops.NewBody, 0)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("sculpt of open surfaces = %v, want sick (no enclosed volume)", pf.Health().Status)
	}
}
