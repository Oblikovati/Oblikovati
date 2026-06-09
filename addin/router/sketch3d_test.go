// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// TestSketch3DSpine exercises the M22-F01 API spine end-to-end through the router:
// create a 3D sketch, list/get it, toggle edit, solve, set properties, and delete.
func TestSketch3DSpine(t *testing.T) {
	r, s := emptyPartSession(t)

	var created wire.CreateSketch3DResult
	call(t, r, s, "sketch3d.create", `{}`, &created)
	if created.SketchIndex != 0 {
		t.Fatalf("sketch3d.create index = %d, want 0", created.SketchIndex)
	}

	var list wire.ListSketches3DResult
	call(t, r, s, "sketch3d.list", "{}", &list)
	if len(list.Sketches) != 1 {
		t.Fatalf("sketch3d.list = %d sketches, want 1", len(list.Sketches))
	}
	got := list.Sketches[0]
	if got.Name != "3D Sketch1" || !got.Visible || got.EntityCount != 0 || got.DOF != 0 {
		t.Fatalf("sketch3d info = %+v, want name '3D Sketch1', visible, 0 entities, 0 DOF", got)
	}

	var one wire.Sketch3DInfo
	call(t, r, s, "sketch3d.get", `{"sketchIndex":0}`, &one)
	if one != got {
		t.Fatalf("sketch3d.get = %+v, want %+v from list", one, got)
	}

	var edit wire.EditSketch3DResult
	call(t, r, s, "sketch3d.edit", `{"sketchIndex":0}`, &edit)
	if !edit.Editing {
		t.Fatal("sketch3d.edit should enter edit mode")
	}
	call(t, r, s, "sketch3d.exitEdit", `{"sketchIndex":0}`, &edit)
	if edit.Editing {
		t.Fatal("sketch3d.exitEdit should leave edit mode")
	}

	var solve wire.SolveSketch3DResult
	call(t, r, s, "sketch3d.solve", `{"sketchIndex":0}`, &solve)
	if solve.Status != "well" || !solve.Converged || !solve.Healthy {
		t.Fatalf("solve of an empty sketch = %+v, want well/converged/healthy", solve)
	}

	var prop wire.Sketch3DInfo
	call(t, r, s, "sketch3d.setProperty", `{"sketchIndex":0,"property":"name","value":"Path"}`, &prop)
	if prop.Name != "Path" {
		t.Fatalf("setProperty name = %q, want Path", prop.Name)
	}
	call(t, r, s, "sketch3d.setProperty", `{"sketchIndex":0,"property":"dimensionsVisible","value":"false"}`, &prop)
	if prop.DimensionsVisible {
		t.Fatal("setProperty dimensionsVisible=false did not stick")
	}

	var ok wire.OKResult
	call(t, r, s, "sketch3d.delete", `{"sketchIndex":0}`, &ok)
	if !ok.OK {
		t.Fatal("sketch3d.delete should succeed")
	}
	call(t, r, s, "sketch3d.list", "{}", &list)
	if len(list.Sketches) != 0 {
		t.Fatalf("after delete, list = %d sketches, want 0", len(list.Sketches))
	}
}

// TestSketch3DEnumerationAndStatus checks entity enumeration and the non-mutating
// constraint-status analysis over a 3D sketch of free points built in-model.
func TestSketch3DEnumerationAndStatus(t *testing.T) {
	r, s := emptyPartSession(t)
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	sk := part.Sketches3D().Add()
	sk.AddPoint3D(math.P3(0, 0, 0))
	sk.AddPoint3D(math.P3(3, 0, 0))

	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) != 2 || ents.Entities[0].Kind != "point" {
		t.Fatalf("entities = %+v, want 2 points", ents.Entities)
	}
	if pt := ents.Entities[1].Points[0]; pt[0] != 3 || pt[1] != 0 || pt[2] != 0 {
		t.Errorf("second point = %v, want [3,0,0]", pt)
	}

	// No driving constraints yet ⇒ two free 3D points ⇒ 6 DOF, under-constrained.
	var status wire.ConstraintStatusResult
	call(t, r, s, "sketch3d.constraintStatus", `{"sketchIndex":0}`, &status)
	if status.DOF != 6 || status.Status != "under" {
		t.Fatalf("status = %+v, want DOF 6 / under", status)
	}

	var dims wire.ListDimensions3DResult
	call(t, r, s, "sketch3d.dimensions", `{"sketchIndex":0}`, &dims)
	if len(dims.Dimensions) != 0 {
		t.Fatalf("dimensions = %+v, want none", dims.Dimensions)
	}
}

// TestSketch3DAddEntities exercises the discriminated 3D entity constructor over the
// router: add a point, line, circle and arc, then enumerate them.
func TestSketch3DAddEntities(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	var pt wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"point","points":[[1,2,3]]}`, &pt)
	if pt.Kind != "point" || len(pt.PointIDs) != 1 {
		t.Fatalf("add point = %+v", pt)
	}

	var line wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[3,0,4]]}`, &line)
	if line.Kind != "line" || len(line.PointIDs) != 2 {
		t.Fatalf("add line = %+v", line)
	}

	var circle wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0,0]],"radius":"10 mm"}`, &circle)
	if circle.Kind != "circle" {
		t.Fatalf("add circle = %+v", circle)
	}

	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"arc","points":[[0,0,0],[1,0,0],[0,1,0]],"ccw":true}`, &wire.AddSketch3DEntityResult{})

	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) != 4 {
		t.Fatalf("entities = %d, want 4 (point/line/circle/arc)", len(ents.Entities))
	}
	if k := kindAt(ents.Entities, 2); k != "circle" {
		t.Errorf("entity 2 kind = %q, want circle", k)
	}
	// The circle's radius enumerates in cm (10 mm = 1 cm).
	if r := ents.Entities[2].Radius; r < 0.999 || r > 1.001 {
		t.Errorf("circle radius = %v cm, want ~1", r)
	}
}

// kindAt returns the kind of the entity at index i (or "" if out of range).
func kindAt(ents []wire.Sketch3DEntityInfo, i int) string {
	if i < 0 || i >= len(ents) {
		return ""
	}
	return ents[i].Kind
}

// TestSketch3DAddHelix exercises the helix constructor's four definition modes over the
// router and checks the resulting curve geometry.
func TestSketch3DAddHelix(t *testing.T) {
	r, s := emptyPartSession(t)
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	cases := []struct {
		name      string
		json      string
		wantTurns float64
		wantPitch float64
	}{
		{"pitchRevolution", `{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"pitchRevolution","pitch":"10 mm","revolutions":3}`, 3, 1.0},
		{"pitchHeight", `{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"pitchHeight","pitch":"10 mm","height":"30 mm"}`, 3, 1.0},
		{"revolutionHeight", `{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"revolutionHeight","revolutions":3,"height":"30 mm"}`, 3, 1.0},
		{"spiral", `{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"spiral","pitch":"2 mm","revolutions":5}`, 5, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var res wire.AddSketch3DEntityResult
			call(t, r, s, "sketch3d.addEntity", tc.json, &res)
			h := lastHelix(t, part)
			if h.Turns != tc.wantTurns {
				t.Errorf("%s turns = %v, want %v", tc.name, h.Turns, tc.wantTurns)
			}
			if stdmath.Abs(h.AxialPerTurn-tc.wantPitch) > 1e-9 {
				t.Errorf("%s pitch = %v cm, want %v", tc.name, h.AxialPerTurn, tc.wantPitch)
			}
		})
	}
}

// lastHelix returns the most-recently-added helix in the active part's first 3D sketch.
func lastHelix(t *testing.T, part *compdef.PartComponentDefinition) *sketch.HelicalCurve3D {
	t.Helper()
	ents := part.Sketches3D().Item(0).Entities()
	for i := len(ents) - 1; i >= 0; i-- {
		if h, ok := ents[i].(*sketch.HelicalCurve3D); ok {
			return h
		}
	}
	t.Fatal("no helix found")
	return nil
}

// TestSketch3DAddHelixErrors covers the helix mode/validation error paths.
func TestSketch3DAddHelixErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	bad := []string{
		`{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"bogus","revolutions":3}`,
		`{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"pitchHeight","pitch":"0 mm","height":"30 mm"}`,
		`{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"pitchRevolution","pitch":"10 mm","revolutions":0}`,
	}
	for _, b := range bad {
		if _, err := r.Handle(s, "sketch3d.addEntity", []byte(b)); err == nil {
			t.Errorf("expected error for %s", b)
		}
	}
}

// TestSketch3DAddConics exercises the ellipse and elliptical-arc constructors over the
// router and checks their enumerated radius.
func TestSketch3DAddConics(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"ellipse","points":[[0,0,0]],"majorRadius":"50 mm","minorRadius":"30 mm"}`, &wire.AddSketch3DEntityResult{})
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"ellipticalArc","points":[[0,0,0]],"majorRadius":"40 mm","minorRadius":"20 mm","startAngle":"0 deg","sweepAngle":"90 deg"}`, &wire.AddSketch3DEntityResult{})

	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) != 2 || ents.Entities[0].Kind != "ellipse" || ents.Entities[1].Kind != "ellipticalArc" {
		t.Fatalf("entities = %+v, want ellipse + ellipticalArc", ents.Entities)
	}
	if r := ents.Entities[0].Radius; r < 4.999 || r > 5.001 { // 50 mm major = 5 cm
		t.Errorf("ellipse major radius = %v cm, want ~5", r)
	}
}

// TestSketch3DAddConicErrors covers the conic validation error paths.
func TestSketch3DAddConicErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	bad := []string{
		`{"sketchIndex":0,"kind":"ellipse","points":[[0,0,0]],"majorRadius":"0 mm","minorRadius":"3 mm"}`,
		`{"sketchIndex":0,"kind":"ellipticalArc","points":[[0,0,0]],"majorRadius":"4 mm","minorRadius":"2 mm","sweepAngle":"0 deg"}`,
		`{"sketchIndex":0,"kind":"ellipse","points":[[0,0,0]],"majorRadius":"oops","minorRadius":"3 mm"}`,
	}
	for _, b := range bad {
		if _, err := r.Handle(s, "sketch3d.addEntity", []byte(b)); err == nil {
			t.Errorf("expected error for %s", b)
		}
	}
}

// TestSketch3DAddEntityErrors covers the malformed-input paths of the constructor.
func TestSketch3DAddEntityErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	if _, err := r.Handle(s, "sketch3d.addEntity", []byte(`{"sketchIndex":0,"kind":"line","points":[[0,0,0]]}`)); err == nil {
		t.Error("a line with one point should error")
	}
	if _, err := r.Handle(s, "sketch3d.addEntity", []byte(`{"sketchIndex":0,"kind":"bogus"}`)); err == nil {
		t.Error("an unknown kind should error")
	}
	if _, err := r.Handle(s, "sketch3d.addEntity", []byte(`{"sketchIndex":0,"kind":"circle","points":[[0,0,0]],"radius":"oops"}`)); err == nil {
		t.Error("a bad radius should error")
	}
}

// TestSketch3DAddConstraints exercises the geometric-constraint constructor over the
// router: build two lines, constrain them perpendicular + one parallel-to-Z, and delete.
func TestSketch3DAddConstraints(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	var l1, l2 wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[1,0,0]]}`, &l1)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[1,1,0]]}`, &l2)

	var perp wire.AddSketch3DConstraintResult
	call(t, r, s, "sketch3d.addConstraint",
		fmt.Sprintf(`{"sketchIndex":0,"kind":"perpendicular","entities":[%d,%d]}`, l1.EntityID, l2.EntityID), &perp)
	if perp.Kind != "perpendicular" || perp.Index != 0 {
		t.Fatalf("addConstraint = %+v", perp)
	}

	var axis wire.AddSketch3DConstraintResult
	call(t, r, s, "sketch3d.addConstraint",
		fmt.Sprintf(`{"sketchIndex":0,"kind":"parallelToZAxis","entities":[%d]}`, l1.EntityID), &axis)
	if axis.Index != 1 {
		t.Fatalf("second constraint index = %d, want 1", axis.Index)
	}

	var cons wire.ListConstraints3DResult
	call(t, r, s, "sketch3d.constraints", `{"sketchIndex":0}`, &cons)
	if len(cons.Constraints) != 2 || cons.Constraints[0].Kind != "perpendicular" || cons.Constraints[1].Kind != "parallelToZAxis" {
		t.Fatalf("constraints = %+v", cons.Constraints)
	}

	var ok wire.OKResult
	call(t, r, s, "sketch3d.deleteConstraint", `{"sketchIndex":0,"constraintIndex":0}`, &ok)
	call(t, r, s, "sketch3d.constraints", `{"sketchIndex":0}`, &cons)
	if len(cons.Constraints) != 1 || cons.Constraints[0].Kind != "parallelToZAxis" {
		t.Fatalf("after delete, constraints = %+v", cons.Constraints)
	}
}

// TestSketch3DAddConstraintErrors covers the constraint operand-validation error paths.
func TestSketch3DAddConstraintErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"point","points":[[0,0,0]]}`, &wire.AddSketch3DEntityResult{})
	bad := []string{
		`{"sketchIndex":0,"kind":"parallel","entities":[999,998]}`, // unknown line ids
		`{"sketchIndex":0,"kind":"coincident","entities":[1]}`,     // wrong operand count
		`{"sketchIndex":0,"kind":"bogus","entities":[1]}`,          // unknown kind
	}
	for _, b := range bad {
		if _, err := r.Handle(s, "sketch3d.addConstraint", []byte(b)); err == nil {
			t.Errorf("expected error for %s", b)
		}
	}
	if _, err := r.Handle(s, "sketch3d.deleteConstraint", []byte(`{"sketchIndex":0,"constraintIndex":5}`)); err == nil {
		t.Error("deleting an out-of-range constraint should error")
	}
}

// TestSketch3DDimensions exercises the dimension constructors + drive over the router.
func TestSketch3DDimensions(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	var line wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[2,0,0]]}`, &line)
	var circle wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0,0]],"radius":"5 mm"}`, &circle)

	var ll wire.AddSketch3DDimensionResult
	call(t, r, s, "sketch3d.addDimension",
		fmt.Sprintf(`{"sketchIndex":0,"kind":"lineLength","entities":[%d],"expression":"10 cm"}`, line.EntityID), &ll)
	if ll.Kind != "lineLength" || ll.Parameter == "" {
		t.Fatalf("lineLength dim = %+v", ll)
	}

	var rad wire.AddSketch3DDimensionResult
	call(t, r, s, "sketch3d.addDimension",
		fmt.Sprintf(`{"sketchIndex":0,"kind":"radius","entities":[%d],"expression":"3 cm"}`, circle.EntityID), &rad)
	if rad.Index != 1 {
		t.Fatalf("radius dim index = %d, want 1", rad.Index)
	}

	var dims wire.ListDimensions3DResult
	call(t, r, s, "sketch3d.dimensions", `{"sketchIndex":0}`, &dims)
	if len(dims.Dimensions) != 2 {
		t.Fatalf("dimensions = %+v, want 2", dims.Dimensions)
	}

	var ok wire.OKResult
	call(t, r, s, "sketch3d.driveDimension", `{"sketchIndex":0,"dimensionIndex":0,"expression":"15 cm","setDriven":true,"driven":true}`, &ok)
	if !ok.OK {
		t.Fatal("driveDimension should succeed")
	}
}

// TestSketch3DDimensionErrors covers the dimension validation error paths.
func TestSketch3DDimensionErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	bad := []string{
		`{"sketchIndex":0,"kind":"radius","entities":[999],"expression":"3 cm"}`,
		`{"sketchIndex":0,"kind":"distance","entities":[1],"expression":"3 cm"}`,
		`{"sketchIndex":0,"kind":"bogus","entities":[1],"expression":"3 cm"}`,
	}
	for _, b := range bad {
		if _, err := r.Handle(s, "sketch3d.addDimension", []byte(b)); err == nil {
			t.Errorf("expected error for %s", b)
		}
	}
	if _, err := r.Handle(s, "sketch3d.driveDimension", []byte(`{"sketchIndex":0,"dimensionIndex":7}`)); err == nil {
		t.Error("driving an out-of-range dimension should error")
	}
}

// TestSketch3DProfilesAndPaths exercises path + profile enumeration over the router: a
// planar square loop is both one closed path and one profile.
func TestSketch3DProfilesAndPaths(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	for _, seg := range [][2][3]float64{
		{{0, 0, 0}, {4, 0, 0}},
		{{4, 0, 0}, {4, 3, 0}},
		{{4, 3, 0}, {0, 3, 0}},
		{{0, 3, 0}, {0, 0, 0}},
	} {
		call(t, r, s, "sketch3d.addEntity",
			fmt.Sprintf(`{"sketchIndex":0,"kind":"line","points":[[%g,%g,%g],[%g,%g,%g]]}`,
				seg[0][0], seg[0][1], seg[0][2], seg[1][0], seg[1][1], seg[1][2]), &wire.AddSketch3DEntityResult{})
	}

	var paths wire.ListPaths3DResult
	call(t, r, s, "sketch3d.paths", `{"sketchIndex":0}`, &paths)
	if len(paths.Paths) != 1 || !paths.Paths[0].Closed || paths.Paths[0].Points != 5 {
		t.Fatalf("paths = %+v, want 1 closed path of 5 pts", paths.Paths)
	}

	var profs wire.ListProfiles3DResult
	call(t, r, s, "sketch3d.profiles", `{"sketchIndex":0}`, &profs)
	if len(profs.Profiles) != 1 {
		t.Fatalf("profiles = %+v, want 1", profs.Profiles)
	}
	if a := profs.Profiles[0].Area; a < 11.999 || a > 12.001 {
		t.Errorf("profile area = %v, want 12", a)
	}
	if len(profs.Profiles[0].Normal) != 3 {
		t.Errorf("profile normal = %v, want a 3-vector", profs.Profiles[0].Normal)
	}
}

// TestSketch3DAddSplines exercises the spline family constructors over the router.
func TestSketch3DAddSplines(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"spline","points":[[0,0,0],[1,2,0],[3,1,1]]}`, &wire.AddSketch3DEntityResult{})
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"controlPointSpline","points":[[0,0,0],[1,0,2],[2,0,0]]}`, &wire.AddSketch3DEntityResult{})
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"fixedSpline","points":[[0,0,0],[1,1,1]]}`, &wire.AddSketch3DEntityResult{})
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"equationCurve","xExpr":"cos(t)","yExpr":"sin(t)","zExpr":"t","t0":0,"t1":3.14159}`, &wire.AddSketch3DEntityResult{})

	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) != 4 {
		t.Fatalf("entities = %d, want 4", len(ents.Entities))
	}
	want := map[string]bool{"spline": false, "controlPointSpline": false, "fixedSpline": false, "equationCurve": false}
	for _, e := range ents.Entities {
		want[e.Kind] = true
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("missing enumerated kind %q (%+v)", k, ents.Entities)
		}
	}
}

// TestSketch3DAddSplineErrors covers the spline validation error paths.
func TestSketch3DAddSplineErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	bad := []string{
		`{"sketchIndex":0,"kind":"spline","points":[[0,0,0]]}`,
		`{"sketchIndex":0,"kind":"equationCurve","xExpr":"@@","yExpr":"t","zExpr":"t","t0":0,"t1":1}`,
	}
	for _, b := range bad {
		if _, err := r.Handle(s, "sketch3d.addEntity", []byte(b)); err == nil {
			t.Errorf("expected error for %s", b)
		}
	}
}

// TestSketch3DTransform exercises move/copy/rotate/delete over the router.
func TestSketch3DTransform(t *testing.T) {
	r, s := emptyPartSession(t)
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	var line wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[1,0,0]]}`, &line)

	// Copy → a second line; entity count becomes 2.
	var cp wire.Transform3DResult
	call(t, r, s, "sketch3d.transform", fmt.Sprintf(`{"sketchIndex":0,"op":"copy","entities":[%d],"vector":[0,5,0]}`, line.EntityID), &cp)
	if len(cp.Created) != 1 || cp.EntityCount != 2 {
		t.Fatalf("copy = %+v, want 1 created / 2 entities", cp)
	}

	// Move the original line.
	call(t, r, s, "sketch3d.transform", fmt.Sprintf(`{"sketchIndex":0,"op":"move","entities":[%d],"vector":[10,0,0]}`, line.EntityID), &wire.Transform3DResult{})
	l := part.Sketches3D().Item(0).Entities()[0].(*sketch.Line3D)
	if l.A.Position() != math.P3(10, 0, 0) {
		t.Errorf("moved original A = %v, want (10,0,0)", l.A.Position())
	}

	// Rotate the copy 90° about +Z through the origin.
	call(t, r, s, "sketch3d.transform", fmt.Sprintf(`{"sketchIndex":0,"op":"rotate","entities":[%d],"center":[0,0,0],"angle":"90 deg"}`, cp.Created[0]), &wire.Transform3DResult{})

	// Delete the copy.
	var del wire.Transform3DResult
	call(t, r, s, "sketch3d.transform", fmt.Sprintf(`{"sketchIndex":0,"op":"delete","entities":[%d]}`, cp.Created[0]), &del)
	if del.EntityCount != 1 {
		t.Fatalf("after delete, entityCount = %d, want 1", del.EntityCount)
	}
}

// TestSketch3DTransformErrors covers the transform validation error paths.
func TestSketch3DTransformErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"point","points":[[0,0,0]]}`, &wire.AddSketch3DEntityResult{})
	bad := []string{
		`{"sketchIndex":0,"op":"move","entities":[999],"vector":[1,0,0]}`,
		`{"sketchIndex":0,"op":"move","entities":[1],"vector":[1,0]}`,
		`{"sketchIndex":0,"op":"bogus","entities":[1]}`,
	}
	for _, b := range bad {
		if _, err := r.Handle(s, "sketch3d.transform", []byte(b)); err == nil {
			t.Errorf("expected error for %s", b)
		}
	}
}

// TestSketch3DInclude builds a box, then includes one of its edges into a 3D sketch as
// associative reference geometry (matching the 2D project-geometry flow).
func TestSketch3DInclude(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})

	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatal(err)
	}
	bodies := part.SurfaceBodies().All()
	if len(bodies) == 0 || len(bodies[0].Edges()) == 0 {
		t.Fatal("extrude produced no body edges to include")
	}
	ref := string(bodies[0].Edges()[0].ReferenceKey())

	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var inc wire.IncludeSketch3DResult
	argBytes, err := json.Marshal(wire.IncludeSketch3DArgs{SketchIndex: 0, Refs: []string{ref}})
	if err != nil {
		t.Fatal(err)
	}
	call(t, r, s, "sketch3d.include", string(argBytes), &inc)
	if len(inc.Created) != 1 || !inc.Healthy {
		t.Fatalf("include = %+v, want 1 created / healthy", inc)
	}

	// The included edge is reference geometry on the 3D sketch.
	sk := part.Sketches3D().Item(0)
	if sk.EntityCount() != 1 {
		t.Fatalf("3D sketch has %d entities, want 1 included curve", sk.EntityCount())
	}
	if _, ok := sk.Entities()[0].(*sketch.IncludedCurve3D); !ok {
		t.Errorf("included entity is %T, want *IncludedCurve3D", sk.Entities()[0])
	}
}

// TestSketch3DIncludeIsAssociative proves an included body vertex tracks the model: after
// the extrude height changes and the part recomputes, the included point re-resolves to
// the moved (top) vertex by reference key — it does not hold a stale topology pointer.
func TestSketch3DIncludeIsAssociative(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	part, _ := modelaccess.ActivePart(s)

	// Pick a top vertex (z ≈ 5 cm) to include — its Z tracks the extrude height.
	var topRef string
	for _, v := range part.SurfaceBodies().All()[0].Vertices() {
		if float64(v.Point().Z) > 4.9 {
			topRef = string(v.ReferenceKey())
			break
		}
	}
	if topRef == "" {
		t.Fatal("no top vertex found")
	}

	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	incArgs, _ := json.Marshal(wire.IncludeSketch3DArgs{SketchIndex: 0, Refs: []string{topRef}})
	call(t, r, s, "sketch3d.include", string(incArgs), &wire.IncludeSketch3DResult{})

	var before wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &before)
	if z := before.Entities[0].Points[0][2]; stdmath.Abs(z-5) > 1e-6 {
		t.Fatalf("included top vertex Z = %v, want 5", z)
	}

	// Change the extrude height 50 → 80 mm; the top vertex moves to z = 8 cm.
	pf := part.Features().Item(0)
	pf.Definition().(*feature.ExtrudeFeature).SetDistance(8)
	part.Features().MarkDirty(pf)
	part.Recompute()

	var after wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &after)
	if z := after.Entities[0].Points[0][2]; stdmath.Abs(z-8) > 1e-6 {
		t.Fatalf("included vertex did not track the height change: Z = %v, want 8", z)
	}
}

// TestSketch3DIncludeUnknownRefIsUnhealthy checks a lost reference is reported, not fatal.
func TestSketch3DIncludeUnknownRefIsUnhealthy(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var inc wire.IncludeSketch3DResult
	call(t, r, s, "sketch3d.include", `{"sketchIndex":0,"refs":["nope"]}`, &inc)
	if inc.Healthy || len(inc.Created) != 0 {
		t.Fatalf("include of a bogus ref = %+v, want unhealthy / nothing created", inc)
	}
}

// TestSketch3DIncludeSketch2D builds a 2D sketch on the XZ plane, then includes one of its
// points and one of its lines into a 3D sketch through sketch3d.includeSketch — the
// geometry is lifted through the 2D sketch's host plane and tracks edits to the source.
func TestSketch3DIncludeSketch2D(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XZ"}`, &wire.CreateSketchResult{})
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatal(err)
	}
	// Author 2D geometry directly on the source sketch: a point at (3,4) and a line.
	src := part.Sketches().Item(0)
	p2d := src.Points().Add(math.P2(3, 4))
	a := src.Points().Add(math.P2(0, 0))
	b := src.Points().Add(math.P2(2, 6))
	line := src.Lines().Add(a, b)

	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var inc wire.IncludeSketch3DResult
	args, _ := json.Marshal(wire.IncludeSketch2DArgs{
		SketchIndex:       0,
		SourceSketchIndex: 0,
		EntityIDs:         []uint64{uint64(p2d.EntityID()), uint64(line.EntityID())},
	})
	call(t, r, s, "sketch3d.includeSketch", string(args), &inc)
	if len(inc.Created) != 2 || !inc.Healthy {
		t.Fatalf("includeSketch = %+v, want 2 created / healthy", inc)
	}

	sk := part.Sketches3D().Item(0)
	if sk.EntityCount() != 2 {
		t.Fatalf("3D sketch has %d entities, want 2 (1 point + 1 curve)", sk.EntityCount())
	}
	pt, ok := sk.Entities()[0].(*sketch.IncludedPoint3D)
	if !ok {
		t.Fatalf("first included entity is %T, want *IncludedPoint3D", sk.Entities()[0])
	}
	// XZ plane: sketch (3,4) lifts to model (3,0,4).
	if got := pt.Position(); stdmath.Abs(got.X-3) > 1e-6 || stdmath.Abs(got.Z-4) > 1e-6 {
		t.Errorf("included 2D point = %v, want (3,0,4)", got)
	}
	if _, ok := sk.Entities()[1].(*sketch.IncludedCurve3D); !ok {
		t.Errorf("second included entity is %T, want *IncludedCurve3D", sk.Entities()[1])
	}
}

// TestSketch3DIncludeSketch2DUnknownIsUnhealthy checks a missing source entity id is
// reported, not fatal.
func TestSketch3DIncludeSketch2DUnknownIsUnhealthy(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var inc wire.IncludeSketch3DResult
	call(t, r, s, "sketch3d.includeSketch", `{"sketchIndex":0,"sourceSketchIndex":0,"entityIDs":[999999]}`, &inc)
	if inc.Healthy || len(inc.Created) != 0 {
		t.Fatalf("includeSketch of a bogus id = %+v, want unhealthy / nothing created", inc)
	}
}

// TestSketch3DSurfaceCurves builds a box and adds an intersection curve between two of
// its faces and a silhouette of one face, by reference key.
func TestSketch3DSurfaceCurves(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatal(err)
	}
	faces := part.SurfaceBodies().All()[0].Faces()
	if len(faces) < 2 {
		t.Fatalf("box has %d faces, want ≥2", len(faces))
	}
	refA := string(faces[0].ReferenceKey())
	refB := string(faces[1].ReferenceKey())

	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	var isect wire.AddSketch3DSurfaceCurveResult
	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "intersection", FaceRefs: []string{refA, refB},
		GridUMin: -10, GridUMax: 10, GridVMin: -10, GridVMax: 10,
	})
	call(t, r, s, "sketch3d.addSurfaceCurve", string(args), &isect)
	if !isect.Healthy || isect.EntityID == 0 {
		t.Fatalf("intersection = %+v, want healthy with an id", isect)
	}

	var silh wire.AddSketch3DSurfaceCurveResult
	args, _ = json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "silhouette", FaceRefs: []string{refA}, ViewDir: []float64{0, 0, 1},
		GridUMin: -10, GridUMax: 10, GridVMin: -10, GridVMax: 10,
	})
	call(t, r, s, "sketch3d.addSurfaceCurve", string(args), &silh)
	if !silh.Healthy {
		t.Fatalf("silhouette = %+v, want healthy", silh)
	}

	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) != 2 || ents.Entities[0].Kind != "intersection" || ents.Entities[1].Kind != "silhouette" {
		t.Fatalf("entities = %+v, want intersection + silhouette", ents.Entities)
	}
}

// TestSketch3DSurfaceCurveErrors covers the surface-curve validation + lost-ref paths.
func TestSketch3DSurfaceCurveErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	// A lost face reference reports unhealthy (not an error).
	var res wire.AddSketch3DSurfaceCurveResult
	call(t, r, s, "sketch3d.addSurfaceCurve", `{"sketchIndex":0,"kind":"silhouette","faceRefs":["nope"],"viewDir":[0,0,1]}`, &res)
	if res.Healthy {
		t.Error("a lost face ref should report unhealthy")
	}
	// Wrong operand counts / unknown kind are errors.
	bad := []string{
		`{"sketchIndex":0,"kind":"intersection","faceRefs":["a"]}`,
		`{"sketchIndex":0,"kind":"bogus","faceRefs":["a","b"]}`,
	}
	for _, b := range bad {
		if _, err := r.Handle(s, "sketch3d.addSurfaceCurve", []byte(b)); err == nil {
			t.Errorf("expected error for %s", b)
		}
	}
}

// TestSketch3DOnFaceCurveOverAPI adds an on-face parameter-space curve via /api, resolving
// the face ref to its surface (M22-F11).
func TestSketch3DOnFaceCurveOverAPI(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatal(err)
	}
	ref := string(part.SurfaceBodies().All()[0].Faces()[0].ReferenceKey())

	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var res wire.AddSketch3DSurfaceCurveResult
	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "onFace", FaceRefs: []string{ref}, UV: []float64{0, 0, 1, 0, 1, 1},
	})
	call(t, r, s, "sketch3d.addSurfaceCurve", string(args), &res)
	if !res.Healthy || res.EntityID == 0 {
		t.Fatalf("onFace = %+v, want healthy with an id", res)
	}

	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) != 1 || ents.Entities[0].Kind != "onFace" {
		t.Fatalf("entities = %+v, want one onFace", ents.Entities)
	}
}

// TestSketch3DOnFaceCurveErrors: a lost ref is unhealthy; a malformed UV is an error.
func TestSketch3DOnFaceCurveErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	var lost wire.AddSketch3DSurfaceCurveResult
	call(t, r, s, "sketch3d.addSurfaceCurve", `{"sketchIndex":0,"kind":"onFace","faceRefs":["nope"],"uv":[0,0,1,1]}`, &lost)
	if lost.Healthy {
		t.Error("onFace with a lost face ref should report unhealthy")
	}
	// An odd-length UV is a request error.
	if _, err := r.Handle(s, "sketch3d.addSurfaceCurve", []byte(`{"sketchIndex":0,"kind":"onFace","faceRefs":["a"],"uv":[0,0,1]}`)); err == nil {
		t.Error("expected an error for an odd-length onFace uv")
	}
}

// TestSketch3DProjectAndOffsetOverAPI adds project-to-surface and offset curves whose
// source is an in-sketch line resolved by entity id (M22-F11).
func TestSketch3DProjectAndOffsetOverAPI(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatal(err)
	}
	ref := string(part.SurfaceBodies().All()[0].Faces()[0].ReferenceKey())

	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var line wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[3,0,4]]}`, &line)
	if line.EntityID == 0 {
		t.Fatal("source line has no entity id")
	}

	// Project the source line onto a part face (associative surface).
	var proj wire.AddSketch3DSurfaceCurveResult
	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "projectToSurface", FaceRefs: []string{ref}, SourceEntityID: line.EntityID,
	})
	call(t, r, s, "sketch3d.addSurfaceCurve", string(args), &proj)
	if !proj.Healthy || proj.EntityID == 0 {
		t.Fatalf("projectToSurface = %+v, want healthy with an id", proj)
	}

	// Offset the source line in the XY plane.
	var off wire.AddSketch3DSurfaceCurveResult
	args, _ = json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "offset", SourceEntityID: line.EntityID, OffsetDistance: 2, Normal: []float64{0, 0, 1},
	})
	call(t, r, s, "sketch3d.addSurfaceCurve", string(args), &off)
	if !off.Healthy || off.EntityID == 0 {
		t.Fatalf("offset = %+v, want healthy with an id", off)
	}

	// An unknown source entity id is an error.
	if _, err := r.Handle(s, "sketch3d.addSurfaceCurve", []byte(`{"sketchIndex":0,"kind":"offset","sourceEntityId":99999,"normal":[0,0,1]}`)); err == nil {
		t.Error("expected an error for an unknown source entity id")
	}
}

// TestSketch3DSurfaceCurveRebindsOnRecompute is the F11 keystone: a surface curve bound to
// a part face by reference key FOLLOWS that face when the part recomputes (the associative
// SurfaceSource re-resolves the moved surface). An onFace curve on the top face tracks the
// extrude height as a driving parameter changes.
func TestSketch3DSurfaceCurveRebindsOnRecompute(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatal(err)
	}
	ref := topFaceKey(t, part)

	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var res wire.AddSketch3DSurfaceCurveResult
	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "onFace", FaceRefs: []string{ref}, UV: []float64{0, 0, 1, 0, 1, 1},
	})
	call(t, r, s, "sketch3d.addSurfaceCurve", string(args), &res)
	if !res.Healthy {
		t.Fatal("onFace curve not healthy")
	}

	z1 := onFaceMaxZ(t, part, res.EntityID) // top face at 50mm = 5cm
	if stdmath.Abs(z1-5) > 0.5 {
		t.Fatalf("initial curve z=%v cm, want ~5", z1)
	}

	// Grow the extrude and recompute (destroys + regenerates the body); the top face moves
	// to z = 8cm and the same reference key rebinds to it.
	setExtrudeDistance(t, part, 8)

	z2 := onFaceMaxZ(t, part, res.EntityID)
	if stdmath.Abs(z2-8) > 0.5 || z2 <= z1 {
		t.Fatalf("after recompute curve z=%v cm, want ~8 (followed the moved face); z1=%v", z2, z1)
	}
}

// setExtrudeDistance changes the part's extrude feature to distance d (db units) and
// recomputes, so the body is regenerated.
func setExtrudeDistance(t *testing.T, part *compdef.PartComponentDefinition, d float64) {
	t.Helper()
	feats := part.Features()
	for i := 0; i < feats.Count(); i++ {
		pf := feats.Item(i)
		if ext, ok := pf.Definition().(*feature.ExtrudeFeature); ok {
			ext.SetDistance(d)
			feats.MarkDirty(pf)
			part.Recompute()
			return
		}
	}
	t.Fatal("no extrude feature found")
}

// topFaceKey returns the reference key of the part's top (+Z normal) planar face.
func topFaceKey(t *testing.T, part *compdef.PartComponentDefinition) string {
	t.Helper()
	for _, b := range part.SurfaceBodies().All() {
		for _, f := range b.Faces() {
			if pl, ok := f.Geometry().(geom.Plane); ok && float64(pl.Normal().Z) > 0.9 {
				return string(f.ReferenceKey())
			}
		}
	}
	t.Fatal("no top (+Z) face found")
	return ""
}

// onFaceMaxZ evaluates the onFace surface curve with the given entity id and returns the
// maximum z of its points (the face's height).
func onFaceMaxZ(t *testing.T, part *compdef.PartComponentDefinition, id uint64) float64 {
	t.Helper()
	sk := part.Sketches3D().Item(0)
	e, ok := sk.EntityByID(sketch.ID(id))
	if !ok {
		t.Fatalf("no entity %d in the 3D sketch", id)
	}
	c, ok := e.(*sketch.OnFaceCurve3D)
	if !ok {
		t.Fatalf("entity %d is %T, want *OnFaceCurve3D", id, e)
	}
	maxZ := -1e18
	for _, p := range c.Evaluate() {
		if float64(p.Z) > maxZ {
			maxZ = float64(p.Z)
		}
	}
	return maxZ
}

// TestModelReferenceKeysSurfacesAndRoundTrips: model.referenceKeys exposes a box's
// topology with keys, and a surfaced face key is consumable by a key consumer (the F08
// round-trip that makes the whole reference-key workflow usable over the wire).
func TestModelReferenceKeysSurfacesAndRoundTrips(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})

	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	if len(keys.Bodies) != 1 {
		t.Fatalf("bodies = %d, want 1", len(keys.Bodies))
	}
	b := keys.Bodies[0]
	if len(b.Faces) != 6 || len(b.Edges) != 12 || len(b.Vertices) != 8 {
		t.Fatalf("box topology = %d faces / %d edges / %d vertices, want 6/12/8", len(b.Faces), len(b.Edges), len(b.Vertices))
	}
	for _, f := range b.Faces {
		if f.Key == "" || len(f.Point) != 3 {
			t.Fatalf("face ref = %+v, want a key and a 3-coord point", f)
		}
	}

	// The surfaced key is consumable: feed it straight into a surface-curve constructor.
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var sc wire.AddSketch3DSurfaceCurveResult
	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "onFace", FaceRefs: []string{b.Faces[0].Key}, UV: []float64{0, 0, 1, 0, 1, 1},
	})
	call(t, r, s, "sketch3d.addSurfaceCurve", string(args), &sc)
	if !sc.Healthy {
		t.Fatal("a surfaced face key should be consumable by addSurfaceCurve")
	}
}

// TestConstraint3DKindMapping covers the constraint/entity wire mappings directly (the
// wire path to add 3D constraints lands in M22-F05).
func TestConstraint3DKindMapping(t *testing.T) {
	a := sketch.NewPoint3D(math.P3(0, 0, 0))
	b := sketch.NewPoint3D(math.P3(1, 0, 0))
	c := sketch.NewPoint3D(math.P3(2, 0, 0))

	if kind, ids := constraint3DKind(sketch.NewCoincident3D(a, b)); kind != types.Geo3DCoincident || len(ids) != 2 {
		t.Errorf("coincident mapping = %q/%v", kind, ids)
	}
	if kind, ids := constraint3DKind(sketch.NewCollinear3D(a, b, c)); kind != types.Geo3DCollinear || len(ids) != 3 {
		t.Errorf("collinear mapping = %q/%v", kind, ids)
	}
	if kind, _ := constraint3DKind(sketch.NewConcentric3D(a, b)); kind != types.Geo3DConcentric {
		t.Errorf("concentric mapping = %q", kind)
	}

	if info := entity3DInfo(0, a); info.Kind != string(types.Sketch3DEntityPoint) || len(info.Points) != 1 {
		t.Errorf("entity3DInfo(point) = %+v", info)
	}
}
