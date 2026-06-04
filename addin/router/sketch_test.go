// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"math"
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

// TestSketchAddConstraintReducesDOFAndSolves applies a horizontal constraint to a line's
// endpoints, then concentric+equalRadius to two circles, checking DOF and enumeration.
func TestSketchAddConstraintReducesDOFAndSolves(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var line wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,1]]}`, &line)
	pA, pB := line.PointIDs[0], line.PointIDs[1]

	var con wire.AddConstraintResult
	args := fmt.Sprintf(`{"sketchIndex":0,"kind":"horizontal","entities":[%d,%d]}`, pA, pB)
	call(t, r, s, "sketch.addConstraint", args, &con)
	if con.Kind != "horizontal" {
		t.Fatalf("constraint kind = %q, want horizontal", con.Kind)
	}

	var cons wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &cons)
	if len(cons.Constraints) != 1 || cons.Constraints[0].Kind != "horizontal" {
		t.Fatalf("constraints = %+v, want one horizontal", cons.Constraints)
	}

	var solved wire.SolveSketchResult
	call(t, r, s, "sketch.solve", `{"sketchIndex":0}`, &solved)
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if y0, y1 := lineEndpointYs(ents.Entities); math.Abs(y0-y1) > 1e-6 {
		t.Fatalf("after horizontal+solve endpoints Y = %v vs %v, want equal", y0, y1)
	}

	// Delete it again.
	var ok wire.OKResult
	call(t, r, s, "sketch.deleteConstraint", `{"sketchIndex":0,"constraintIndex":0}`, &ok)
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &cons)
	if len(cons.Constraints) != 0 {
		t.Fatalf("after delete: %d constraints, want 0", len(cons.Constraints))
	}
}

// lineEndpointYs returns the Y coordinates of the first enumerated line's two endpoints.
func lineEndpointYs(ents []wire.SketchEntityInfo) (float64, float64) {
	for _, e := range ents {
		if e.Kind == "line" && len(e.Points) == 2 {
			return e.Points[0][1], e.Points[1][1]
		}
	}
	return 0, 1 // unequal sentinel → fails the test loudly
}

func TestSketchAddConstraintConcentric(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var c1, c2 wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"1 cm"}`, &c1)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[2,2]],"radius":"2 cm"}`, &c2)
	var con wire.AddConstraintResult
	args := fmt.Sprintf(`{"sketchIndex":0,"kind":"concentric","entities":[%d,%d]}`, c1.EntityID, c2.EntityID)
	call(t, r, s, "sketch.addConstraint", args, &con)
	if con.Kind != "concentric" {
		t.Fatalf("kind = %q, want concentric", con.Kind)
	}
}

func TestSketchAddConstraintRejectsBadRefCount(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if _, err := r.Handle(s, "sketch.addConstraint", []byte(`{"sketchIndex":0,"kind":"parallel","entities":[1]}`)); err == nil {
		t.Fatal("expected error for parallel with one ref")
	}
}

// TestSketchConstraintKinds exercises the remaining constraint families and the
// reference-resolution error paths.
func TestSketchConstraintKinds(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var l1, l2, c1 wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &l1)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,2],[4,2]]}`, &l2)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[2,2]],"radius":"1 cm"}`, &c1)

	var con wire.AddConstraintResult
	for _, tc := range []struct{ kind, refs string }{
		{"parallel", fmt.Sprintf("[%d,%d]", l1.EntityID, l2.EntityID)},
		{"perpendicular", fmt.Sprintf("[%d,%d]", l1.EntityID, l2.EntityID)},
		{"equalLength", fmt.Sprintf("[%d,%d]", l1.EntityID, l2.EntityID)},
		{"tangent", fmt.Sprintf("[%d,%d]", l1.EntityID, c1.EntityID)},
		{"pointOnLine", fmt.Sprintf("[%d,%d]", l1.PointIDs[0], l2.EntityID)},
		{"fix", fmt.Sprintf("[%d]", l1.PointIDs[0])},
	} {
		args := fmt.Sprintf(`{"sketchIndex":0,"kind":%q,"entities":%s}`, tc.kind, tc.refs)
		call(t, r, s, "sketch.addConstraint", args, &con)
		if con.Kind != tc.kind {
			t.Fatalf("kind = %q, want %q", con.Kind, tc.kind)
		}
	}
}

func TestSketchAddConstraintErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &wire.AddSketchEntityResult{})
	for _, bad := range []string{
		`{"sketchIndex":0,"kind":"bogus","entities":[1,2]}`,              // unsupported kind
		`{"sketchIndex":0,"kind":"coincident","entities":[99999,99998]}`, // missing point id
		`{"sketchIndex":0,"kind":"concentric","entities":[99999,99998]}`, // missing circular id
		`{"sketchIndex":0,"kind":"fix","entities":[1,2]}`,                // wrong ref count
	} {
		if _, err := r.Handle(s, "sketch.addConstraint", []byte(bad)); err == nil {
			t.Errorf("expected error for %s", bad)
		}
	}
}

func TestSketchDeleteConstraintBadIndex(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if _, err := r.Handle(s, "sketch.deleteConstraint", []byte(`{"sketchIndex":0,"constraintIndex":3}`)); err == nil {
		t.Fatal("expected error deleting out-of-range constraint")
	}
}

// TestSketchRadiusDimensionDrivesCircle adds a radius dimension to a circle, solves it to
// the target, then re-drives it to a new value.
func TestSketchRadiusDimensionDrivesCircle(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var circ wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"10 mm"}`, &circ)

	var dim wire.AddDimensionResult
	args := fmt.Sprintf(`{"sketchIndex":0,"kind":"radius","entities":[%d],"expression":"20 mm"}`, circ.EntityID)
	call(t, r, s, "sketch.addDimension", args, &dim)
	if dim.Kind != "radius" || dim.Parameter == "" {
		t.Fatalf("dimension = %+v, want radius with a parameter", dim)
	}

	call(t, r, s, "sketch.solve", `{"sketchIndex":0}`, &wire.SolveSketchResult{})
	if got := circleRadiusOf(t, r, s); math.Abs(got-2) > 1e-6 { // 20 mm = 2 cm
		t.Fatalf("after radius dim + solve, radius = %v, want 2", got)
	}

	// Re-drive to 30 mm.
	call(t, r, s, "sketch.driveDimension", `{"sketchIndex":0,"dimensionIndex":0,"expression":"30 mm"}`, &wire.OKResult{})
	call(t, r, s, "sketch.solve", `{"sketchIndex":0}`, &wire.SolveSketchResult{})
	if got := circleRadiusOf(t, r, s); math.Abs(got-3) > 1e-6 {
		t.Fatalf("after drive to 30 mm, radius = %v, want 3", got)
	}

	var dims wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &dims)
	if len(dims.Dimensions) != 1 || dims.Dimensions[0].Kind != "radius" {
		t.Fatalf("dimensions = %+v, want one radius", dims.Dimensions)
	}
}

// circleRadiusOf enumerates entities and returns the first circle's radius.
func circleRadiusOf(t *testing.T, r *Router, s *app.Session) float64 {
	t.Helper()
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	for _, e := range ents.Entities {
		if e.Kind == "circle" {
			return e.Radius
		}
	}
	t.Fatal("no circle entity found")
	return 0
}

func TestSketchAddDimensionErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	for _, bad := range []string{
		`{"sketchIndex":0,"kind":"radius","entities":[99999],"expression":"1 cm"}`, // missing circle
		`{"sketchIndex":0,"kind":"bogus","entities":[1],"expression":"1 cm"}`,      // unsupported kind
	} {
		if _, err := r.Handle(s, "sketch.addDimension", []byte(bad)); err == nil {
			t.Errorf("expected error for %s", bad)
		}
	}
	if _, err := r.Handle(s, "sketch.driveDimension", []byte(`{"sketchIndex":0,"dimensionIndex":5}`)); err == nil {
		t.Fatal("expected error driving an out-of-range dimension")
	}
}

// TestSketchConstraintStatusReportsDOF takes a circle from under- to fully-constrained and
// checks the non-mutating status query reflects each state.
func TestSketchConstraintStatusReportsDOF(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var circ wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"10 mm"}`, &circ)
	center := circ.PointIDs[0]

	var st wire.ConstraintStatusResult
	call(t, r, s, "sketch.constraintStatus", `{"sketchIndex":0}`, &st)
	if st.Status != "under" || st.DOF != 3 {
		t.Fatalf("bare circle status = %+v, want under / DOF 3 (center x,y + radius)", st)
	}

	call(t, r, s, "sketch.addConstraint", fmt.Sprintf(`{"sketchIndex":0,"kind":"fix","entities":[%d]}`, center), &wire.AddConstraintResult{})
	call(t, r, s, "sketch.addDimension", fmt.Sprintf(`{"sketchIndex":0,"kind":"radius","entities":[%d],"expression":"10 mm"}`, circ.EntityID), &wire.AddDimensionResult{})

	call(t, r, s, "sketch.constraintStatus", `{"sketchIndex":0}`, &st)
	if st.Status != "well" || st.DOF != 0 {
		t.Fatalf("constrained circle status = %+v, want well / DOF 0", st)
	}
}

// TestSketchProfilesEnumerated builds a rectangle with a rectangular hole and checks the
// profiles API reports the annulus region with the right area.
func TestSketchProfilesEnumerated(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[10,10]]}`, &wire.AddSketchEntityResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[2,2],[8,8]]}`, &wire.AddSketchEntityResult{})

	var prof wire.ListProfilesResult
	call(t, r, s, "sketch.profiles", `{"sketchIndex":0}`, &prof)
	var annulus *wire.ProfileInfo
	for i := range prof.Profiles {
		if prof.Profiles[i].Holes == 1 {
			annulus = &prof.Profiles[i]
		}
	}
	if annulus == nil {
		t.Fatalf("no profile with a hole among %+v", prof.Profiles)
	}
	if math.Abs(annulus.Area-64) > 1e-9 || !annulus.Closed { // 100 − 36
		t.Fatalf("annulus = %+v, want area 64 / closed", *annulus)
	}
}

// TestSketchTransformCopyAndMirror exercises the F09 edit operations through the API.
func TestSketchTransformCopyAndMirror(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var circ wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"10 mm"}`, &circ)

	// Copy the circle by (5,0) → two circles.
	var res wire.TransformSketchResult
	args := fmt.Sprintf(`{"sketchIndex":0,"op":"copy","entities":[%d],"vector":[5,0]}`, circ.EntityID)
	call(t, r, s, "sketch.transform", args, &res)
	if len(res.Created) != 1 {
		t.Fatalf("copy created = %d, want 1", len(res.Created))
	}
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if countKind(ents.Entities, "circle") != 2 {
		t.Fatalf("circles after copy = %d, want 2", countKind(ents.Entities, "circle"))
	}

	// Move the original in place by (0,3).
	var moved wire.TransformSketchResult
	args = fmt.Sprintf(`{"sketchIndex":0,"op":"move","entities":[%d],"vector":[0,3]}`, circ.EntityID)
	call(t, r, s, "sketch.transform", args, &moved)
	if len(moved.Created) != 0 {
		t.Fatalf("move created = %d, want 0 (in place)", len(moved.Created))
	}
}

func TestSketchTransformErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"1 cm"}`, &wire.AddSketchEntityResult{})
	for _, bad := range []string{
		`{"sketchIndex":0,"op":"bogus","entities":[1]}`,                   // unknown op
		`{"sketchIndex":0,"op":"move","entities":[]}`,                     // empty selection
		`{"sketchIndex":0,"op":"move","entities":[99999],"vector":[1,1]}`, // missing entity
		`{"sketchIndex":0,"op":"move","entities":[1],"vector":[1]}`,       // bad vector
	} {
		if _, err := r.Handle(s, "sketch.transform", []byte(bad)); err == nil {
			t.Errorf("expected error for %s", bad)
		}
	}
}

// TestSketchPatternsViaAPI builds rectangular and circular patterns of a circle.
func TestSketchPatternsViaAPI(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var seed wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"5 mm"}`, &seed)

	var rect wire.AddSketchPatternResult
	args := fmt.Sprintf(`{"sketchIndex":0,"kind":"rectangular","entities":[%d],"count1":3,"count2":2,"spacing1":"20 mm","spacing2":"20 mm"}`, seed.EntityID)
	call(t, r, s, "sketch.addPattern", args, &rect)
	if len(rect.Created) != 5 { // 3×2 − seed
		t.Fatalf("rectangular created = %d, want 5", len(rect.Created))
	}

	// Fresh sketch for the circular pattern.
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"circle","points":[[2,0]],"radius":"5 mm"}`, &seed)
	var circ wire.AddSketchPatternResult
	args = fmt.Sprintf(`{"sketchIndex":1,"kind":"circular","entities":[%d],"count":6,"angle":"360 deg","center":[0,0]}`, seed.EntityID)
	call(t, r, s, "sketch.addPattern", args, &circ)
	if len(circ.Created) != 5 { // 6 − seed
		t.Fatalf("circular created = %d, want 5", len(circ.Created))
	}

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":1}`, &ents)
	if countKind(ents.Entities, "circle") != 6 {
		t.Fatalf("circles after circular pattern = %d, want 6", countKind(ents.Entities, "circle"))
	}
}

func TestSketchAddPatternErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"5 mm"}`, &wire.AddSketchEntityResult{})
	for _, bad := range []string{
		`{"sketchIndex":0,"kind":"bogus","entities":[1]}`,                                                                 // unknown kind
		`{"sketchIndex":0,"kind":"rectangular","entities":[1],"count1":0,"count2":1,"spacing1":"1 cm","spacing2":"1 cm"}`, // bad count
		`{"sketchIndex":0,"kind":"circular","entities":[1],"count":1,"angle":"360 deg","center":[0,0]}`,                   // count < 2
	} {
		if _, err := r.Handle(s, "sketch.addPattern", []byte(bad)); err == nil {
			t.Errorf("expected error for %s", bad)
		}
	}
}

// TestSketchOffsetAndImage exercises the F05 offset + image-placement operations.
func TestSketchOffsetAndImage(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var c wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"10 mm"}`, &c)

	var off wire.OffsetSketchResult
	args := fmt.Sprintf(`{"sketchIndex":0,"entity":%d,"distance":"5 mm"}`, c.EntityID)
	call(t, r, s, "sketch.offset", args, &off)
	if off.Kind != "circle" {
		t.Fatalf("offset kind = %q, want circle", off.Kind)
	}
	if !hasCircleRadius2(t, r, s, 1.5) { // 10mm + 5mm = 15mm = 1.5 cm
		t.Fatalf("no radius-1.5 offset circle")
	}

	var img wire.AddSketchImageResult
	call(t, r, s, "sketch.addImage", `{"sketchIndex":0,"ref":"pkg://trace.png","anchor":[1,1],"width":"80 mm","height":"60 mm","opacity":0.5}`, &img)
	if img.EntityID == 0 {
		t.Fatal("addImage returned no entity id")
	}
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if countKind(ents.Entities, "image") != 1 {
		t.Fatalf("images = %d, want 1", countKind(ents.Entities, "image"))
	}
}

func TestSketchOffsetErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"10 mm"}`, &wire.AddSketchEntityResult{})
	if _, err := r.Handle(s, "sketch.offset", []byte(`{"sketchIndex":0,"entity":99999,"distance":"1 mm"}`)); err == nil {
		t.Fatal("expected error offsetting a missing entity")
	}
}

// hasCircleRadius2 enumerates and checks for a circle of the given radius.
func hasCircleRadius2(t *testing.T, r *Router, s *app.Session, radius float64) bool {
	t.Helper()
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	return hasCircleRadius(ents.Entities, radius)
}

// TestSketchAnnotationsAndArcSlot exercises fill-region/text/arc-slot through the API.
func TestSketchAnnotationsAndArcSlot(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[10,10]]}`, &wire.AddSketchEntityResult{})

	var fill wire.AddEntityIDResult
	call(t, r, s, "sketch.addFillRegion", `{"sketchIndex":0,"seed":[5,5],"style":"hatch"}`, &fill)
	if fill.EntityID == 0 {
		t.Fatal("addFillRegion returned no id")
	}
	var text wire.AddEntityIDResult
	call(t, r, s, "sketch.addText", `{"sketchIndex":0,"anchor":[1,1],"text":"PART A","height":"5 mm","justify":"center"}`, &text)

	var arc wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"slot","variant":"arc","points":[[20,0],[25,0],[20,5]],"width":"2 cm","ccw":true}`, &arc)
	if len(arc.EntityIDs) != 4 {
		t.Fatalf("arc slot ids = %d, want 4", len(arc.EntityIDs))
	}

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if countKind(ents.Entities, "fillRegion") != 1 || countKind(ents.Entities, "text") != 1 {
		t.Fatalf("want 1 fillRegion + 1 text, got %+v", ents.Entities)
	}
}

// TestSketchNewConstraints212 exercises ground/offset/patternLink through the API.
func TestSketchNewConstraints212(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var l1, l2 wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &l1)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,2],[4,2]]}`, &l2)

	var con wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", fmt.Sprintf(`{"sketchIndex":0,"kind":"ground","entities":[%d]}`, l1.EntityID), &con)
	if con.Kind != "ground" {
		t.Fatalf("kind = %q, want ground", con.Kind)
	}
	call(t, r, s, "sketch.addConstraint", fmt.Sprintf(`{"sketchIndex":0,"kind":"offset","entities":[%d,%d]}`, l1.EntityID, l2.EntityID), &con)
	if con.Kind != "offset" {
		t.Fatalf("kind = %q, want offset", con.Kind)
	}

	var cons wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &cons)
	kinds := map[string]bool{}
	for _, c := range cons.Constraints {
		kinds[c.Kind] = true
	}
	if !kinds["ground"] || !kinds["offset"] {
		t.Fatalf("enumerated kinds = %v, want ground + offset", kinds)
	}
}

func TestSketchGroundNeedsOneRef(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if _, err := r.Handle(s, "sketch.addConstraint", []byte(`{"sketchIndex":0,"kind":"ground","entities":[1,2]}`)); err == nil {
		t.Fatal("expected error for ground with two refs")
	}
}

// TestSketchAdvancedDimensions exercises offset/three-point-angle dims through the API.
func TestSketchAdvancedDimensions(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var line, pt wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &line)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"point","points":[[2,3]]}`, &pt)

	var dim wire.AddDimensionResult
	args := fmt.Sprintf(`{"sketchIndex":0,"kind":"offsetDim","entities":[%d,%d],"expression":"30 mm"}`, pt.PointIDs[0], line.EntityID)
	call(t, r, s, "sketch.addDimension", args, &dim)
	if dim.Kind != "offsetDim" || math.Abs(dim.Value-3) > 1e-9 { // measured perp distance = 3
		t.Fatalf("offset dim = %+v, want offsetDim / value 3", dim)
	}

	// Three-point angle on three points.
	var pa, pb wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"point","points":[[5,4]]}`, &pa)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"point","points":[[4,5]]}`, &pb)
	args = fmt.Sprintf(`{"sketchIndex":0,"kind":"threePointAngle","entities":[%d,%d,%d],"expression":"90 deg"}`, pt.PointIDs[0], pa.PointIDs[0], pb.PointIDs[0])
	call(t, r, s, "sketch.addDimension", args, &dim)
	if dim.Kind != "threePointAngle" {
		t.Fatalf("kind = %q, want threePointAngle", dim.Kind)
	}

	// Ellipse-radius on an ellipse.
	var el wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"ellipse","points":[[20,20]],"axis":[1,0],"majorRadius":"2 cm","minorRadius":"1 cm"}`, &el)
	args = fmt.Sprintf(`{"sketchIndex":0,"kind":"ellipseRadius","entities":[%d],"expression":"3 cm"}`, el.EntityID)
	call(t, r, s, "sketch.addDimension", args, &dim)
	if dim.Kind != "ellipseRadius" {
		t.Fatalf("kind = %q, want ellipseRadius", dim.Kind)
	}

	var dims wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &dims)
	if len(dims.Dimensions) != 3 {
		t.Fatalf("dimensions = %d, want 3 (offset/angle/ellipse)", len(dims.Dimensions))
	}

	for _, bad := range []string{
		`{"sketchIndex":0,"kind":"offsetDim","entities":[1],"expression":"1 cm"}`,          // wrong ref count
		`{"sketchIndex":0,"kind":"threePointAngle","entities":[1,2],"expression":"1 deg"}`, // wrong ref count
		`{"sketchIndex":0,"kind":"ellipseRadius","entities":[99999],"expression":"1 cm"}`,  // missing ellipse
	} {
		if _, err := r.Handle(s, "sketch.addDimension", []byte(bad)); err == nil {
			t.Errorf("expected error for %s", bad)
		}
	}
}

// TestSketchDerivedCurves exercises equation/fixed/offset-spline curves through the API.
func TestSketchDerivedCurves(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var res wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"equationCurve","xExpr":"cos(t)","yExpr":"sin(t)","t0":0,"t1":6.283185}`, &res)
	if res.Kind != "equationCurve" {
		t.Fatalf("kind = %q, want equationCurve", res.Kind)
	}
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"fixedSpline","points":[[0,0],[1,1],[2,0]]}`, &res)

	var parent wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"spline","points":[[5,5],[7,5]]}`, &parent)
	call(t, r, s, "sketch.addEntity", fmt.Sprintf(`{"sketchIndex":0,"kind":"offsetSpline","entityRefs":[%d],"radius":"5 mm"}`, parent.EntityID), &res)

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	for _, k := range []string{"equationCurve", "fixedSpline", "offsetSpline"} {
		if countKind(ents.Entities, k) != 1 {
			t.Errorf("want one %s, got %+v", k, ents.Entities)
		}
	}
}

func TestSketchEquationCurveRejectsBadExpr(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if _, err := r.Handle(s, "sketch.addEntity", []byte(`{"sketchIndex":0,"kind":"equationCurve","xExpr":"cos(u)","yExpr":"sin(t)","t0":0,"t1":1}`)); err == nil {
		t.Fatal("expected error for unknown variable in equation curve")
	}
}

// TestSketchTrimSplitExtend exercises the F09 curve-edit ops through the API.
func TestSketchTrimSplitExtend(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var l, v1, v2 wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[6,0]]}`, &l)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[2,-1],[2,1]]}`, &v1)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[4,-1],[4,1]]}`, &v2)

	// Trim the middle of the horizontal line → two stubs survive.
	var res wire.TransformSketchResult
	call(t, r, s, "sketch.transform", fmt.Sprintf(`{"sketchIndex":0,"op":"trim","entities":[%d],"vector":[3,0]}`, l.EntityID), &res)
	if len(res.Created) != 2 {
		t.Fatalf("trim returned %d lines, want 2 stubs", len(res.Created))
	}

	// Split a fresh line at its midpoint → 2.
	var l2 wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,5],[4,5]]}`, &l2)
	call(t, r, s, "sketch.transform", fmt.Sprintf(`{"sketchIndex":0,"op":"split","entities":[%d],"vector":[2,5]}`, l2.EntityID), &res)
	if len(res.Created) != 2 {
		t.Fatalf("split returned %d lines, want 2", len(res.Created))
	}
}

func TestSketchTrimErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"1 cm"}`, &wire.AddSketchEntityResult{})
	// Circle isn't a line → trim rejects it.
	if _, err := r.Handle(s, "sketch.transform", []byte(`{"sketchIndex":0,"op":"trim","entities":[2],"vector":[0,0]}`)); err == nil {
		t.Fatal("expected error trimming a non-line")
	}
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
