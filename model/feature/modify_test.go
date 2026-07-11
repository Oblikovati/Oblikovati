// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/sketch"
)

func TestCombineJoinsTwoBodiesForReal(t *testing.T) {
	fs := NewPartFeatures(nil)
	// Two disjoint prisms in the running state, then Combine them.
	NewBaseFeatures(fs).AddBase(buildPrism(squarePoly(0), sketch.XYPlane(), span{near: 0, far: 1}, 0, "a"))
	NewBaseFeatures(fs).AddBase(buildPrism(squarePoly(10), sketch.XYPlane(), span{near: 0, far: 1}, 0, "b"))
	combine := NewModifyFeatures(fs).AddCombine(0, 1, ops.Join)
	fs.Recompute()

	if !combine.Health().OK() {
		t.Fatalf("combine sick: %+v", combine.Health())
	}
	// The two prisms are joined into one body (12 faces); 1 body remains.
	if len(fs.Result()) != 1 || len(fs.Result()[0].Faces()) != 12 {
		t.Errorf("combine result = %d bodies; want 1 with 12 faces", len(fs.Result()))
	}
	if ops.Validate(fs.Result()[0]).Manifold == false {
		t.Error("combined body should be manifold")
	}
}

func TestCombineCutOverlappingForReal(t *testing.T) {
	fs := NewPartFeatures(nil)
	// Block A: 2×2×2 at the origin (vol 8). Tool B: 2×2×2 shifted to x∈[1,3]
	// (overlap 1×2×2 = 4). A − B should leave 4.
	NewBaseFeatures(fs).AddBase(buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "a"))
	NewBaseFeatures(fs).AddBase(buildPrism([]math.Point2{{X: 1, Y: 0}, {X: 3, Y: 0}, {X: 3, Y: 2}, {X: 1, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "b"))
	cut := NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)
	fs.Recompute()

	if !cut.Health().OK() {
		t.Fatalf("overlapping cut sick: %+v", cut.Health())
	}
	if len(fs.Result()) != 1 {
		t.Fatalf("cut result = %d bodies, want 1", len(fs.Result()))
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("cut body not a valid solid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, 4) > 1e-6 {
		t.Errorf("A−B volume = %g, want 4 (8 − 4 overlap)", v)
	}
}

func TestCombineRejectsBadIndices(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	bad := NewModifyFeatures(fs).AddCombine(0, 5, ops.Join) // tool index out of range
	fs.Recompute()
	if bad.Health().Status != health.Sick {
		t.Errorf("combine with bad indices = %v, want sick", bad.Health().Status)
	}
}

func TestDirectEditsResolveThenDefer(t *testing.T) {
	body := prismBody()
	face := body.Faces()[0].ReferenceKey()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(body)
	mod := NewModifyFeatures(fs)
	// Only split still defers; the other direct edits are real (see their own tests).
	feats := map[string]*PartFeature{
		"split": mod.AddSplit([][]byte{face}),
	}
	fs.Recompute()
	for name, pf := range feats {
		if pf.Health().Status != health.Warning {
			t.Errorf("%s = %v, want warning (resolved + deferred)", name, pf.Health().Status)
		}
		if pf.Kind() != name {
			t.Errorf("kind = %q, want %q", pf.Kind(), name)
		}
	}
}

// TestMoveAndOffsetFaceRealGeometry moves the top face of a box up and offsets it in,
// checking each is healthy and changes the volume the expected way.
func TestMoveAndOffsetFaceRealGeometry(t *testing.T) {
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var top []byte
	for _, f := range box.Faces() {
		if f.Geometry().NormalAt(0, 0).Z > 0.99 {
			top = f.ReferenceKey()
		}
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	mv := NewModifyFeatures(fs).AddMoveFace([][]byte{top}, math.V3(0, 0, 1)) // grow 2×2×2 → 2×2×3
	fs.Recompute()
	if !mv.Health().OK() {
		t.Fatalf("move-face sick: %+v", mv.Health())
	}
	if got := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; relErr(got, 12) > 1e-6 {
		t.Errorf("move-face volume = %g, want 12", got)
	}
}

// TestDeleteFaceHealsInModel chamfers a box edge then deletes the chamfer face through the
// feature engine: the body heals back to the sharp box (vol 8), a valid solid.
func TestDeleteFaceHealsInModel(t *testing.T) {
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var edge []byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			edge = e.ReferenceKey()
			break
		}
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	NewDressUpFeatures(fs).AddChamfer([][]byte{edge}, func() float64 { return 0.5 })
	fs.Recompute()
	var chamfer []byte
	for _, f := range fs.Result()[0].Faces() {
		if n := f.Geometry().NormalAt(0, 0); stdmath.Abs(n.X) > 0.1 && stdmath.Abs(n.Y) > 0.1 {
			chamfer = f.ReferenceKey()
		}
	}
	del := NewModifyFeatures(fs).AddDeleteFace([][]byte{chamfer}, true)
	fs.Recompute()
	if !del.Health().OK() {
		t.Fatalf("delete-face sick: %+v", del.Health())
	}
	if got := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; relErr(got, 8) > 1e-6 {
		t.Errorf("healed volume = %g, want 8", got)
	}
}

// TestDeleteFaceOpenLeavesSurface is the heal=false arm (#1884): deleting the top face without
// healing leaves an open, non-solid surface body of five faces.
func TestDeleteFaceOpenLeavesSurface(t *testing.T) {
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var top []byte
	for _, f := range box.Faces() {
		if f.Geometry().NormalAt(0, 0).Z > 0.99 {
			top = f.ReferenceKey()
		}
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	del := NewModifyFeatures(fs).AddDeleteFace([][]byte{top}, false)
	fs.Recompute()
	if !del.Health().OK() {
		t.Fatalf("delete-face (open) sick: %+v", del.Health())
	}
	body := fs.Result()[0]
	if body.IsSolid() {
		t.Error("heal=false should leave an open (non-solid) surface body")
	}
	if got := len(body.Faces()); got != 5 {
		t.Errorf("open body has %d faces, want 5 (6 minus the deleted top)", got)
	}
}

// TestReplaceFaceIdentityIsValid exercises the replace-face feature wiring and key
// resolution on a box: replacing the top face with its own plane is a valid no-op (vol 8).
// Geometric correctness for non-identity targets is covered by the kernel ReplaceFaces tests
// (a same-body parallel/stepped target needs a stepped solid to set up).
func TestReplaceFaceIdentityIsValid(t *testing.T) {
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var top []byte
	for _, f := range box.Faces() {
		if f.Geometry().NormalAt(0, 0).Z > 0.99 {
			top = f.ReferenceKey()
		}
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	rf := NewModifyFeatures(fs).AddReplaceFace([][]byte{top}, top) // replace top with its own plane
	fs.Recompute()
	if !rf.Health().OK() {
		t.Fatalf("replace-face sick: %+v", rf.Health())
	}
	if got := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; relErr(got, 8) > 1e-6 {
		t.Errorf("identity replace-face volume = %g, want 8", got)
	}
}

// TestThickenSurfaceToSlab thickens a planar surface patch (added as the base body) into a
// slab solid through the feature engine: 2×3 patch × 0.5 = vol 3, a valid solid.
func TestThickenSurfaceToSlab(t *testing.T) {
	patch := patchSurfaceBody(2, 3)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(patch)
	th := NewModifyFeatures(fs).AddThicken(0.5)
	fs.Recompute()
	if !th.Health().OK() {
		t.Fatalf("thicken sick: %+v", th.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("thickened patch not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, 3) > 1e-6 {
		t.Errorf("slab volume = %g, want 3", got)
	}
}

// patchSurfaceBody builds a one-face planar surface (non-solid) body [0,w]×[0,h] at z=0.
func patchSurfaceBody(w, h float64) *topo.Body {
	lin := topo.NewLineage(topo.Tok("test", "patch", 0))
	bld := topo.NewBuilder(false, lin)
	p := []math.Point3{{X: 0, Y: 0}, {X: w, Y: 0}, {X: w, Y: h}, {X: 0, Y: h}}
	v := make([]*topo.Vertex, 4)
	for i, q := range p {
		v[i] = bld.AddVertex(q, lin)
	}
	uses := make([]topo.Use, 4)
	for i := range p {
		e := bld.AddEdge(geom.NewLineSegment(p[i], p[(i+1)%4]), v[i], v[(i+1)%4], lin)
		uses[i] = topo.Use{Edge: e}
	}
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, lin, topo.OuterLoop(uses...))
	return bld.Build()
}

// squarePoly returns a unit square offset by dx; planeXY is the XY sketch plane.
func squarePoly(dx float64) []math.Point2 {
	return []math.Point2{{X: dx, Y: 0}, {X: dx + 1, Y: 0}, {X: dx + 1, Y: 1}, {X: dx, Y: 1}}
}

func TestCombineDefinitionAccessible(t *testing.T) {
	fs := NewPartFeatures(nil)
	c := NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)
	if c.Definition().(*CombineFeature).Definition().Operation != ops.Cut {
		t.Error("combine definition not accessible")
	}
}

// The #331 face-edit extensions: move-face rotate mode and the approximation
// request, both surviving the recipe codec.
func TestMoveFaceRotateAndApproximationRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewModifyFeatures(fs).AddMoveFaceRotate([][]byte{[]byte("f1")},
		math.P3(0, 0, 2), math.V3(0, 1, 0), constFloat(0.15))
	NewModifyFeatures(fs).AddFaceOffsetApprox([][]byte{[]byte("f2")},
		constFloat(0.3), types.NeverTooThinApproximation)

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if d := data[0].FaceEdit; len(d.AxisDir) != 3 || d.Angle != 0.15 || d.Translation != nil {
		t.Fatalf("serialized rotate = %+v, want axis+angle and no translation", d)
	}
	if data[1].FaceEdit.Approximation != "neverTooThin" {
		t.Fatalf("serialized approximation = %q", data[1].FaceEdit.Approximation)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if _, dir, angle, rotating := fresh.Item(0).Definition().(*MoveFaceFeature).Rotation(); !rotating || dir.Y != 1 || angle != 0.15 {
		t.Errorf("restored rotate = dir %v angle %v rotating %v", dir, angle, rotating)
	}
	if got := fresh.Item(1).Definition().(*FaceOffsetFeature).Approximation(); got != types.NeverTooThinApproximation {
		t.Errorf("restored approximation = %v, want neverTooThin", got)
	}
}

// TestThickenApproximationRoundTrip: thicken carries its approximation too.
func TestThickenApproximationRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	pf := NewModifyFeatures(fs).AddThicken(0.2)
	pf.Definition().(*ThickenFeature).SetApproximation(types.MeanApproximation)
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if data[0].Thicken.Approximation != "mean" {
		t.Fatalf("serialized thicken = %+v", data[0].Thicken)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if got := fresh.Item(0).Definition().(*ThickenFeature).Approximation(); got != types.MeanApproximation {
		t.Errorf("restored thicken approximation = %v, want mean", got)
	}
}
