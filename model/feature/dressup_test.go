// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/health"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// TestChamferBevelsEdgeForReal drills a real 45° chamfer on a box edge and checks the
// removed volume (a right-triangle prism of legs d along the full edge).
func TestChamferBevelsEdgeForReal(t *testing.T) {
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var edge []byte
	for _, e := range box.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if a.X == b.X && a.Y == b.Y { // a vertical edge
			edge = e.ReferenceKey()
			break
		}
	}
	if edge == nil {
		t.Fatal("no vertical edge found on the box")
	}
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(box)
	ch := NewDressUpFeatures(fs).AddChamfer([][]byte{edge}, func() float64 { return 0.5 })
	fs.Recompute()

	if !ch.Health().OK() {
		t.Fatalf("chamfer sick: %+v", ch.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("chamfered body not a valid solid: %+v", r)
	}
	want := 8 - 0.5*0.5*0.5*2 // box 8 − wedge (½·d²·length, d=0.5, length=2) = 7.75
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, want) > 1e-6 {
		t.Errorf("chamfer volume = %g, want %g", got, want)
	}
}

// TestChamferAllFourVerticalEdges chains four wedge cuts on a 2×2×2 box (an octagonal
// prism). Each cut feeds the previous result back into the boolean — the chaining case the
// triangle-soup CSG got wrong; the B-rep boolean must keep it exact. Volume =
// (4 − 4·½·0.5²)·2 = 7.
func TestChamferAllFourVerticalEdges(t *testing.T) {
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var verticals [][]byte
	for _, e := range box.Edges() {
		if a, b := e.StartVertex().Point(), e.EndVertex().Point(); a.X == b.X && a.Y == b.Y {
			verticals = append(verticals, e.ReferenceKey())
		}
	}
	if len(verticals) != 4 {
		t.Fatalf("found %d vertical edges, want 4", len(verticals))
	}
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(box)
	ch := NewDressUpFeatures(fs).AddChamfer(verticals, func() float64 { return 0.5 })
	fs.Recompute()
	if !ch.Health().OK() {
		t.Fatalf("multi-edge chamfer sick: %+v", ch.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("octagonal prism not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, 7) > 1e-6 {
		t.Errorf("four-edge chamfer volume = %g, want 7", got)
	}
}

// prismBody builds a unit-square prism via the extrude generator (for real edges).
func prismBody() *topo.Body {
	return buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}, sketch.XYPlane(), span{near: 0, far: 1}, 0, "ext")
}

// TestFilletRoundsEdgeForReal rounds a vertical edge of a 2×2×2 box (r=0.5) through the
// feature engine: a healthy feature whose result is a valid solid with a cylinder face.
func TestFilletRoundsEdgeForReal(t *testing.T) {
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var edge []byte
	for _, e := range box.Edges() {
		if a, b := e.StartVertex().Point(), e.EndVertex().Point(); a.X == b.X && a.Y == b.Y {
			edge = e.ReferenceKey()
			break
		}
	}
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(box)
	fillet := NewDressUpFeatures(fs).AddFillet([][]byte{edge}, func() float64 { return 0.5 })
	fs.Recompute()
	if !fillet.Health().OK() {
		t.Fatalf("fillet sick: %+v", fillet.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("filleted body not a valid solid: %+v", r)
	}
	want := 8 - (0.5*0.5-stdmath.Pi*0.25*0.25)*2
	if got := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-4}).Volume; relErr(got, want) > 1e-3 {
		t.Errorf("fillet volume = %g, want ≈ %g", got, want)
	}
}

func TestFilletGoesSickOnLostEdge(t *testing.T) {
	body := prismBody()
	bogus := []byte("not-an-edge-key")

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(body)
	fillet := NewDressUpFeatures(fs).AddFillet([][]byte{bogus}, func() float64 { return 0.2 })
	fs.Recompute()
	if fillet.Health().Status != health.Sick {
		t.Errorf("fillet with a lost edge = %v, want sick", fillet.Health().Status)
	}
}

func TestDressUpWithNoBodyIsSick(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	fillet := NewDressUpFeatures(fs).AddFillet([][]byte{[]byte("x")}, func() float64 { return 1 })
	fs.Recompute() // no preceding body
	if fillet.Health().Status != health.Sick {
		t.Errorf("dress-up with no body = %v, want sick", fillet.Health().Status)
	}
}

// TestThreadResolvesThenDefers checks the thread feature (still deferred geometry) resolves
// its face and reports Warning, and that a lost-face shell goes Sick. (Shell and draft are
// real now — see their own tests.)
func TestThreadResolvesThenDefers(t *testing.T) {
	body := prismBody()
	face := body.Faces()[0].ReferenceKey()

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(body)
	du := NewDressUpFeatures(fs)
	th := du.AddThread(face, "M6x1")
	fs.Recompute()
	if th.Health().Status != health.Warning {
		t.Errorf("thread health = %v, want warning (resolved + deferred)", th.Health().Status)
	}
	if th.Kind() != "thread" {
		t.Errorf("kind = %q, want thread", th.Kind())
	}
	// A shell referencing a vanished face goes sick.
	bad := du.AddShell([][]byte{[]byte("ghost")}, func() float64 { return 0.2 })
	fs.MarkDirty(bad)
	fs.Recompute()
	if bad.Health().Status != health.Sick {
		t.Error("shell with a lost face should be sick")
	}
}

// TestDraftTapersFaceForReal drafts one side face of a 2×2×2 box inward by atan(0.25)
// about +Z: the side tilts, removing a 4·tan = 1 wedge ⇒ vol 7, a valid solid.
func TestDraftTapersFaceForReal(t *testing.T) {
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var side []byte
	for _, f := range box.Faces() {
		if f.Geometry().NormalAt(0, 0).X > 0.99 {
			side = f.ReferenceKey()
		}
	}
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(box)
	dr := NewDressUpFeatures(fs).AddDraft([][]byte{side}, func() float64 { return -stdmath.Atan(0.25) })
	fs.Recompute()
	if !dr.Health().OK() {
		t.Fatalf("draft sick: %+v", dr.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("drafted body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, 7) > 1e-6 {
		t.Errorf("draft volume = %g, want 7", got)
	}
}

// TestShellHollowsRunningBodyForReal shells the running body (a unit prism) with its top
// face removed and checks the feature is healthy and the result a valid hollow solid.
func TestShellHollowsRunningBodyForReal(t *testing.T) {
	body := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}}, sketch.XYPlane(), span{near: 0, far: 4}, 0, "box")
	var top []byte
	for _, f := range body.Faces() {
		if f.Geometry().NormalAt(0, 0).Z > 0.99 {
			top = f.ReferenceKey()
		}
	}
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(body)
	sh := NewDressUpFeatures(fs).AddShell([][]byte{top}, func() float64 { return 0.5 })
	fs.Recompute()
	if !sh.Health().OK() {
		t.Fatalf("shell sick: %+v", sh.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("shelled body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, 64-3*3*3.5) > 1e-6 {
		t.Errorf("shell volume = %g, want %g", got, 64-3*3*3.5)
	}
}

func TestDressUpDefinitionsAccessible(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	du := NewDressUpFeatures(fs)
	f := du.AddFillet([][]byte{[]byte("e")}, func() float64 { return 0.2 })
	if f.Definition().(*FilletFeature).Definition().Radius() != 0.2 {
		t.Error("fillet radius not accessible via definition")
	}
	if (&ChamferFeature{def: &ChamferDefinition{}}).Definition() == nil ||
		(&ShellFeature{def: &ShellDefinition{}}).Definition() == nil ||
		(&FaceDraftFeature{def: &FaceDraftDefinition{}}).Definition() == nil ||
		(&ThreadFeature{def: &ThreadDefinition{Designation: "M6"}}).Definition().Designation != "M6" {
		t.Error("a dress-up definition is not accessible")
	}
}
