// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/addin/opregistry"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/compdef"
)

// emptyPartSession returns a router and a session with one active, empty part.
func emptyPartSession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	s := app.NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "scratch.obk", true); err != nil {
		t.Fatalf("add part: %v", err)
	}
	return New(opregistry.Default()), s
}

func TestSketchCreateAndRectangleThenExtrude(t *testing.T) {
	r, s := emptyPartSession(t)

	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &sk)
	if sk.SketchIndex != 0 || sk.Plane != "XY" {
		t.Fatalf("sketch.create = %+v, want index 0 plane XY", sk)
	}

	var rect wire.SketchRectangleResult
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &rect)
	if rect.Profiles != 1 {
		t.Fatalf("sketch.rectangle profiles = %d, want 1 closed profile", rect.Profiles)
	}

	var ext struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &ext)
	if ext.Bodies != 1 {
		t.Fatalf("extrude bodies = %d, want 1", ext.Bodies)
	}

	var tree wire.ModelTreeResult
	call(t, r, s, "model.tree", "{}", &tree)
	if tree.Sketches != 1 || tree.Bodies != 1 || len(tree.Features) != 1 {
		t.Fatalf("model.tree = %+v, want 1 sketch / 1 body / 1 feature", tree)
	}
}

// TestSketchSpineEnumerateEditSolveDelete exercises the F01 API spine end-to-end through
// the router: create a rectangle sketch, then list/get/enumerate/edit/solve/delete it.
func TestSketchSpineEnumerateEditSolveDelete(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})

	var list wire.ListSketchesResult
	call(t, r, s, "sketch.list", "{}", &list)
	if len(list.Sketches) != 1 {
		t.Fatalf("sketch.list = %d sketches, want 1", len(list.Sketches))
	}
	got := list.Sketches[0]
	if got.Plane != "XY" || got.Name == "" || got.EntityCount != 8 {
		t.Fatalf("sketch info = %+v, want plane XY, a name, 8 entities (4 lines + 4 points)", got)
	}
	if got.DOF != 8 {
		t.Fatalf("DOF = %d, want 8 (unconstrained rectangle: 4 points × 2)", got.DOF)
	}

	var gotOne wire.SketchInfo
	call(t, r, s, "sketch.get", `{"sketchIndex":0}`, &gotOne)
	if gotOne != got {
		t.Fatalf("sketch.get = %+v, want %+v from list", gotOne, got)
	}

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if lines := countKind(ents.Entities, "line"); lines != 4 {
		t.Fatalf("entities: %d lines, want 4 (%+v)", lines, ents.Entities)
	}
	if pts := countKind(ents.Entities, "point"); pts != 4 {
		t.Fatalf("entities: %d points, want 4", pts)
	}

	var cons wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &cons)
	if len(cons.Constraints) != 0 {
		t.Fatalf("constraints = %d, want 0 on a bare rectangle", len(cons.Constraints))
	}

	var ed wire.EditSketchResult
	call(t, r, s, "sketch.edit", `{"sketchIndex":0}`, &ed)
	if !ed.Editing {
		t.Fatal("sketch.edit: Editing = false, want true")
	}
	call(t, r, s, "sketch.exitEdit", `{"sketchIndex":0}`, &ed)
	if ed.Editing {
		t.Fatal("sketch.exitEdit: Editing = true, want false")
	}

	var solved wire.SolveSketchResult
	call(t, r, s, "sketch.solve", `{"sketchIndex":0}`, &solved)
	if solved.DOF != 8 || solved.Status != "under" {
		t.Fatalf("sketch.solve = %+v, want DOF 8 / status under", solved)
	}

	var ok wire.OKResult
	call(t, r, s, "sketch.delete", `{"sketchIndex":0}`, &ok)
	if !ok.OK {
		t.Fatal("sketch.delete: OK = false")
	}
	call(t, r, s, "sketch.list", "{}", &list)
	if len(list.Sketches) != 0 {
		t.Fatalf("after delete: %d sketches, want 0", len(list.Sketches))
	}
}

// countKind counts enumerated entities of the given wire kind.
func countKind(ents []wire.SketchEntityInfo, kind string) int {
	n := 0
	for _, e := range ents {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

// TestSketchSetPropertyRoundTripsThroughGet sets each sketch property through the API
// and reads it back via sketch.get (F01 PBI-201).
func TestSketchSetPropertyRoundTripsThroughGet(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var info wire.SketchInfo
	call(t, r, s, "sketch.setProperty", `{"sketchIndex":0,"property":"name","value":"Profile"}`, &info)
	if info.Name != "Profile" {
		t.Fatalf("setProperty name → %q, want Profile", info.Name)
	}
	call(t, r, s, "sketch.setProperty", `{"sketchIndex":0,"property":"visible","value":"false"}`, &info)
	call(t, r, s, "sketch.setProperty", `{"sketchIndex":0,"property":"color","value":"#ff8800"}`, &info)
	call(t, r, s, "sketch.setProperty", `{"sketchIndex":0,"property":"lineType","value":"center"}`, &info)
	call(t, r, s, "sketch.setProperty", `{"sketchIndex":0,"property":"lineWeight","value":"0.5 mm"}`, &info)
	call(t, r, s, "sketch.setProperty", `{"sketchIndex":0,"property":"deferUpdates","value":"true"}`, &info)

	var got wire.SketchInfo
	call(t, r, s, "sketch.get", `{"sketchIndex":0}`, &got)
	if got.Name != "Profile" || got.Visible || got.Color != "#ff8800" || got.LineType != "center" || !got.DeferUpdates {
		t.Fatalf("after sets, get = %+v", got)
	}
	if got.LineWeight <= 0 { // 0.5 mm in model cm
		t.Fatalf("lineWeight = %v, want > 0 (0.5 mm in cm)", got.LineWeight)
	}
}

func TestSketchSetPropertyUnknownProperty(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if _, err := r.Handle(s, "sketch.setProperty", []byte(`{"sketchIndex":0,"property":"bogus","value":"x"}`)); err == nil {
		t.Fatal("expected error for unknown property")
	}
}

// TestSketchAddEntityKinds creates each F02 entity kind/variant through the API and
// verifies they enumerate with the right kinds and geometry.
func TestSketchAddEntityKinds(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var res wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &res)
	if len(res.PointIDs) != 2 || res.Kind != "line" {
		t.Fatalf("addEntity line = %+v, want kind line / 2 point ids", res)
	}
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"point","points":[[1,1]]}`, &res)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","variant":"centerRadius","points":[[0,0]],"radius":"10 mm"}`, &res)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","variant":"threePoint","points":[[1,0],[0,1],[-1,0]]}`, &res)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"arc","variant":"centerStartEnd","points":[[0,0],[2,0],[0,2]],"ccw":true}`, &res)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"arc","variant":"threePoint","points":[[2,0],[0,2],[-2,0]]}`, &res)

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if got := countKind(ents.Entities, "circle"); got != 2 {
		t.Fatalf("circles = %d, want 2", got)
	}
	if got := countKind(ents.Entities, "arc"); got != 2 {
		t.Fatalf("arcs = %d, want 2", got)
	}
	// The center-radius circle (10 mm → 1 cm) must report radius 1.
	if !hasCircleRadius(ents.Entities, 1.0) {
		t.Fatalf("no circle with radius 1 cm (10 mm) in %+v", ents.Entities)
	}
}

// TestSketchAddConicsAndSplines creates ellipse/ellipticalArc/spline through the API.
func TestSketchAddConicsAndSplines(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var res wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"ellipse","points":[[0,0]],"axis":[1,0],"majorRadius":"20 mm","minorRadius":"10 mm"}`, &res)
	if res.Kind != "ellipse" {
		t.Fatalf("ellipse kind = %q", res.Kind)
	}
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"ellipticalArc","points":[[0,0]],"axis":[1,0],"majorRadius":"20 mm","minorRadius":"10 mm","startAngle":"0 deg","endAngle":"90 deg"}`, &res)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"spline","points":[[0,0],[1,1],[2,0]]}`, &res)
	if len(res.PointIDs) != 3 {
		t.Fatalf("spline point ids = %d, want 3", len(res.PointIDs))
	}

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	for _, want := range []string{"ellipse", "ellipticalArc", "spline"} {
		if countKind(ents.Entities, want) != 1 {
			t.Errorf("want exactly one %s entity, got %+v", want, ents.Entities)
		}
	}
}

// TestSketchAddCompositeEntities builds rectangle/polygon/slot via the API and checks
// each yields its closed profile.
func TestSketchAddCompositeEntities(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var res wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[4,3]]}`, &res)
	if len(res.EntityIDs) != 4 {
		t.Fatalf("rectangle entity ids = %d, want 4 lines", len(res.EntityIDs))
	}
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"polygon","points":[[10,0],[12,0]],"sides":6}`, &res)
	if len(res.EntityIDs) != 6 {
		t.Fatalf("hexagon entity ids = %d, want 6", len(res.EntityIDs))
	}
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"slot","points":[[0,10],[6,10]],"width":"2 cm"}`, &res)
	if len(res.EntityIDs) != 4 {
		t.Fatalf("slot entity ids = %d, want 4 (2 lines + 2 arcs)", len(res.EntityIDs))
	}

	// Rectangle + hexagon + slot are three disjoint closed regions.
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatal(err)
	}
	if got := part.Sketches().Item(0).Profiles().Count(); got != 3 {
		t.Fatalf("profiles = %d, want 3 closed regions", got)
	}
}

// TestSketchFilletViaAPI builds two corner lines then fillets them through the API,
// asserting a tangent arc appears.
func TestSketchFilletViaAPI(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var l1, l2 wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &l1)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[0,4]]}`, &l2)

	var fillet wire.AddSketchEntityResult
	args := fmt.Sprintf(`{"sketchIndex":0,"kind":"fillet","entityRefs":[%d,%d],"radius":"10 mm"}`, l1.EntityID, l2.EntityID)
	call(t, r, s, "sketch.addEntity", args, &fillet)

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if countKind(ents.Entities, "arc") != 1 {
		t.Fatalf("want exactly one arc after fillet, got %+v", ents.Entities)
	}
	if !hasArcRadius(ents.Entities, 1) { // a 10 mm = 1 cm radius fillet arc
		t.Fatalf("no radius-1 arc after fillet in %+v", ents.Entities)
	}
}

// hasArcRadius reports whether some enumerated arc has the given radius.
func hasArcRadius(ents []wire.SketchEntityInfo, radius float64) bool {
	for _, e := range ents {
		if e.Kind == "arc" && e.Radius == radius {
			return true
		}
	}
	return false
}

func TestSketchFilletNeedsTwoLineRefs(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if _, err := r.Handle(s, "sketch.addEntity", []byte(`{"sketchIndex":0,"kind":"fillet","entityRefs":[1],"radius":"1 cm"}`)); err == nil {
		t.Fatal("expected error for fillet with one ref")
	}
}

func TestSketchPolygonNeedsThreeSides(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if _, err := r.Handle(s, "sketch.addEntity", []byte(`{"sketchIndex":0,"kind":"polygon","points":[[0,0],[1,0]],"sides":2}`)); err == nil {
		t.Fatal("expected error for a 2-sided polygon")
	}
}

func TestSketchAddEllipseNeedsAxis(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if _, err := r.Handle(s, "sketch.addEntity", []byte(`{"sketchIndex":0,"kind":"ellipse","points":[[0,0]],"majorRadius":"2 cm","minorRadius":"1 cm"}`)); err == nil {
		t.Fatal("expected error for ellipse without a 2-component axis")
	}
}

func TestSketchAddEntityRejectsCollinearCircle(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if _, err := r.Handle(s, "sketch.addEntity", []byte(`{"sketchIndex":0,"kind":"circle","variant":"threePoint","points":[[0,0],[1,0],[2,0]]}`)); err == nil {
		t.Fatal("expected error for circle through collinear points")
	}
}

// hasCircleRadius reports whether some enumerated circle has the given radius.
func hasCircleRadius(ents []wire.SketchEntityInfo, radius float64) bool {
	for _, e := range ents {
		if e.Kind == "circle" && e.Radius == radius {
			return true
		}
	}
	return false
}

func TestSketchCreateUnknownPlane(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "sketch.create", []byte(`{"plane":"AB"}`)); err == nil {
		t.Fatal("expected error for unknown plane")
	}
}

func TestSketchRectangleBadIndex(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "sketch.rectangle", []byte(`{"sketchIndex":5,"width":"1 cm","height":"1 cm"}`)); err == nil {
		t.Fatal("expected error for out-of-range sketch index")
	}
}
