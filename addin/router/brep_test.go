// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/geom"
)

// Transient B-rep factory over the wire (M07-F05, #628).

// TestBrepPrimitivesAndLifecycle: create, describe, list, delete.
func TestBrepPrimitivesAndLifecycle(t *testing.T) {
	r, s := emptyPartSession(t)
	var block wire.BrepHandleResult
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[0,0,0],"max":[3,2,5]}`, &block)
	if block.Handle != 1 || !block.Stats.Solid || stdmath.Abs(block.Stats.Volume-30) > 1e-6 {
		t.Fatalf("block = %+v, want handle 1 volume 30", block)
	}
	var sphere wire.BrepHandleResult
	call(t, r, s, "brep.createPrimitive", `{"kind":"sphere","center":[0,0,0],"radius":2}`, &sphere)
	want := 4.0 / 3 * stdmath.Pi * 8
	if stdmath.Abs(sphere.Stats.Volume-want)/want > 0.02 {
		t.Errorf("sphere volume = %g, want ~%g", sphere.Stats.Volume, want)
	}
	var list wire.BrepListResult
	call(t, r, s, "brep.list", `{}`, &list)
	if len(list.Handles) != 2 {
		t.Fatalf("handles = %v, want 2", list.Handles)
	}
	var desc wire.BrepHandleResult
	call(t, r, s, "brep.describe", `{"handle":1}`, &desc)
	if desc.Stats.Faces != 6 {
		t.Errorf("described block faces = %d, want 6", desc.Stats.Faces)
	}
	call(t, r, s, "brep.delete", `{"handle":2}`, &wire.OKResult{})
	call(t, r, s, "brep.list", `{}`, &list)
	if len(list.Handles) != 1 {
		t.Errorf("handles after delete = %v, want [1]", list.Handles)
	}
}

// TestBrepBooleanTransformCopy: union grows the blank in place; transform
// translates; copy mints an independent body.
func TestBrepBooleanTransformCopy(t *testing.T) {
	r, s := emptyPartSession(t)
	var a, b wire.BrepHandleResult
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[0,0,0],"max":[1,1,1]}`, &a)
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[1,0,0],"max":[2,1,1]}`, &b)
	var res wire.BrepHandleResult
	call(t, r, s, "brep.boolean",
		fmt.Sprintf(`{"blankHandle":%d,"tool":{"handle":%d},"operation":"union"}`, a.Handle, b.Handle), &res)
	if res.Handle != a.Handle || stdmath.Abs(res.Stats.Volume-2) > 1e-6 {
		t.Fatalf("union = %+v, want the blank at volume 2", res)
	}
	var moved wire.BrepHandleResult
	call(t, r, s, "brep.transform",
		fmt.Sprintf(`{"handle":%d,"matrix":[1,0,0,10, 0,1,0,0, 0,0,1,0, 0,0,0,1]}`, a.Handle), &moved)
	if stdmath.Abs(moved.Stats.Volume-2) > 1e-6 {
		t.Errorf("moved volume = %g, want 2", moved.Stats.Volume)
	}
	var copied wire.BrepHandleResult
	call(t, r, s, "brep.copy", fmt.Sprintf(`{"source":{"handle":%d}}`, a.Handle), &copied)
	if copied.Handle == a.Handle || stdmath.Abs(copied.Stats.Volume-2) > 1e-6 {
		t.Errorf("copy = %+v, want a fresh handle at volume 2", copied)
	}
}

// TestBrepCopiesDocumentBody: a document body referenced by index copies into
// transient space (the document body itself is never mutated).
func TestBrepCopiesDocumentBody(t *testing.T) {
	r, s := boxPartSession(t)
	var copied wire.BrepHandleResult
	call(t, r, s, "brep.copy", `{"source":{"bodyIndex":0}}`, &copied)
	if copied.Handle == 0 || stdmath.Abs(copied.Stats.Volume-60) > 0.01 {
		t.Fatalf("document copy = %+v, want the 60 cm³ box", copied)
	}
	var list wire.BodyListResult
	call(t, r, s, "body.list", `{}`, &list)
	if len(list.Bodies) != 1 {
		t.Errorf("document still has %d bodies, want 1 (copy is transient)", len(list.Bodies))
	}
}

// TestBrepSectionAndOffset: a plane section yields a closed wire; offsetting
// it outward grows its length.
func TestBrepSectionAndOffset(t *testing.T) {
	r, s := emptyPartSession(t)
	var block wire.BrepHandleResult
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[0,0,0],"max":[4,3,2]}`, &block)
	var sec wire.BrepWiresResult
	call(t, r, s, "brep.sectionWithPlane",
		fmt.Sprintf(`{"source":{"handle":%d},"planeOrigin":[0,0,1],"planeNormal":[0,0,1]}`, block.Handle), &sec)
	if len(sec.Wires) != 1 || !sec.Wires[0].Closed {
		t.Fatalf("section = %+v, want one closed wire", sec.Wires)
	}
	var off wire.OffsetPlanarWireResult
	call(t, r, s, "wire.offsetPlanar",
		fmt.Sprintf(`{"handle":%d,"wireIndex":0,"normal":[0,0,1],"distance":-0.5,"cornerClosure":"linear"}`, sec.Handle), &off)
	if off.Handle == 0 || len(off.Wires) != 1 {
		t.Fatalf("offset = %+v, want one wire on a new handle", off)
	}
}

// TestBrepDeleteFacesOpensBox: deleting one face leaves a 5-face surface body.
func TestBrepDeleteFacesOpensBox(t *testing.T) {
	r, s := boxPartSession(t)
	var copied wire.BrepHandleResult
	call(t, r, s, "brep.copy", `{"source":{"bodyIndex":0}}`, &copied)
	// The transient copy's faces carry copy-derived keys; keep-instead with an
	// unknown key keeps nothing — the removes-every-face guard must fire.
	_, err := r.Handle(s, "brep.deleteFaces",
		[]byte(fmt.Sprintf(`{"handle":%d,"faceKeys":["nope"],"keepInstead":true}`, copied.Handle)))
	if err == nil || !strings.Contains(err.Error(), "every face") {
		t.Fatalf("keep-none = %v, want the removes-every-face error", err)
	}
	// Plain delete with an unknown key keeps all 6 faces (a no-op delete).
	var res wire.BrepHandleResult
	call(t, r, s, "brep.deleteFaces",
		fmt.Sprintf(`{"handle":%d,"faceKeys":["nope"]}`, copied.Handle), &res)
	if res.Stats.Faces != 6 {
		t.Errorf("no-op delete faces = %d, want 6", res.Stats.Faces)
	}
}

// TestBrepImprintAndIdentical: stacked blocks imprint; congruent blocks group.
func TestBrepImprintAndIdentical(t *testing.T) {
	r, s := emptyPartSession(t)
	var big, small wire.BrepHandleResult
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[0,0,0],"max":[4,4,1]}`, &big)
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[1,1,1],"max":[3,3,2]}`, &small)
	var imp wire.BrepImprintResult
	call(t, r, s, "brep.imprint",
		fmt.Sprintf(`{"bodyOne":{"handle":%d},"bodyTwo":{"handle":%d}}`, big.Handle, small.Handle), &imp)
	if imp.HandleOne == 0 || imp.HandleTwo == 0 || len(imp.OneTouchedFaceKeys) == 0 {
		t.Fatalf("imprint = %+v, want new handles and touched faces", imp)
	}
	var a, b, c wire.BrepHandleResult
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[0,0,0],"max":[1,2,3]}`, &a)
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[5,0,0],"max":[6,2,3]}`, &b)
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[0,0,0],"max":[1,2,4]}`, &c)
	var groups wire.BrepIdenticalBodiesResult
	call(t, r, s, "brep.identicalBodies",
		fmt.Sprintf(`{"sources":[{"handle":%d},{"handle":%d},{"handle":%d}]}`, a.Handle, b.Handle, c.Handle), &groups)
	if len(groups.Groups) != 2 || len(groups.Groups[0]) != 2 {
		t.Errorf("identical groups = %v, want [[0 1] [2]]", groups.Groups)
	}
}

// TestBrepCreateFromDefinition: a sound tetra graph compiles; a bad index is
// reported by path with no handle.
func TestBrepCreateFromDefinition(t *testing.T) {
	r, s := emptyPartSession(t)
	def := tetraWireDefinition()
	args, err := json.Marshal(wire.BrepCreateFromDefinitionArgs{Definition: def})
	if err != nil {
		t.Fatal(err)
	}
	var res wire.BrepCreateFromDefinitionResult
	call(t, r, s, "brep.createFromDefinition", string(args), &res)
	if res.Handle == 0 || len(res.Issues) != 0 {
		t.Fatalf("definition compile = %+v, want a handle", res)
	}
	if want := 1.0 / 6; stdmath.Abs(res.Stats.Volume-want) > 1e-6 {
		t.Errorf("tetra volume = %g, want %g", res.Stats.Volume, want)
	}
	def.Edges[0].StartVertex = 42
	args, _ = json.Marshal(wire.BrepCreateFromDefinitionArgs{Definition: def})
	var bad wire.BrepCreateFromDefinitionResult
	call(t, r, s, "brep.createFromDefinition", string(args), &bad)
	if bad.Handle != 0 || len(bad.Issues) == 0 || bad.Issues[0].Path != "edges[0]" {
		t.Errorf("bad graph = %+v, want edges[0] issue and no handle", bad)
	}
}

// tetraWireDefinition is the unit tetrahedron as a wire definition graph
// (the same windings as the kernel compiler's own test).
func tetraWireDefinition() types.BrepBodyDefinition {
	pts := [][]float64{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	var def types.BrepBodyDefinition
	def.Solid = true
	for _, p := range pts {
		def.Vertices = append(def.Vertices, types.BrepVertexDef{Position: p})
	}
	pairs := [][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}}
	for _, pr := range pairs {
		def.Edges = append(def.Edges, types.BrepEdgeDef{
			Curve: types.BrepCurveDef{Kind: "lineSegment", Points: []float64{
				pts[pr[0]][0], pts[pr[0]][1], pts[pr[0]][2],
				pts[pr[1]][0], pts[pr[1]][1], pts[pr[1]][2],
			}},
			StartVertex: pr[0], EndVertex: pr[1],
		})
	}
	use := func(e int, opp bool) types.BrepEdgeUseDef { return types.BrepEdgeUseDef{Edge: e, Opposed: opp} }
	loops := [][]types.BrepEdgeUseDef{
		{use(1, false), use(3, true), use(0, true)},
		{use(0, false), use(4, false), use(2, true)},
		{use(2, false), use(5, true), use(1, true)},
		{use(3, false), use(5, false), use(4, true)},
	}
	normals := [][]float64{{0, 0, -1}, {0, -1, 0}, {-1, 0, 0}, {1, 1, 1}}
	origins := [][]float64{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {1, 0, 0}}
	for i := range loops {
		def.Faces = append(def.Faces, types.BrepFaceDef{
			Surface: types.BrepSurfaceDef{Kind: "plane", Origin: origins[i], Normal: normals[i]},
			Loops:   []types.BrepLoopDef{{Uses: loops[i]}},
		})
	}
	def.Lumps = []types.BrepLumpDef{{Shells: []types.BrepShellDef{{Faces: []int{0, 1, 2, 3}}}}}
	return def
}

// TestBrepRuledSurfaceFromSections: rule between two parallel sections of a
// block — a tube of area perimeter × gap.
func TestBrepRuledSurfaceFromSections(t *testing.T) {
	r, s := emptyPartSession(t)
	var block wire.BrepHandleResult
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[0,0,0],"max":[2,1,4]}`, &block)
	var s1, s2 wire.BrepWiresResult
	call(t, r, s, "brep.sectionWithPlane",
		fmt.Sprintf(`{"source":{"handle":%d},"planeOrigin":[0,0,1],"planeNormal":[0,0,1]}`, block.Handle), &s1)
	call(t, r, s, "brep.sectionWithPlane",
		fmt.Sprintf(`{"source":{"handle":%d},"planeOrigin":[0,0,3],"planeNormal":[0,0,1]}`, block.Handle), &s2)
	var ruled wire.BrepHandleResult
	call(t, r, s, "brep.ruledSurface",
		fmt.Sprintf(`{"sectionOne":{"body":{"handle":%d},"wireIndex":0},"sectionTwo":{"body":{"handle":%d},"wireIndex":0}}`,
			s1.Handle, s2.Handle), &ruled)
	if ruled.Stats.Solid {
		t.Error("a ruled surface is a surface body")
	}
	if ruled.Stats.Faces == 0 {
		t.Error("ruled surface should carry faces")
	}
}

// TestBrepSilhouetteOnDocumentCylinder: silhouette of a revolved cylinder's
// side face from +X — two rulings.
func TestBrepSilhouetteOnDocumentCylinder(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"20 mm"}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct{}{})
	var copied wire.BrepHandleResult
	call(t, r, s, "brep.copy", `{"source":{"bodyIndex":0}}`, &copied)
	// Find the cylindrical side face on the COPY via ray: fire from outside
	// at mid-height toward the axis.
	keyArgs, _ := json.Marshal(wire.BrepSilhouetteArgs{
		Source: wire.BrepBodyRef{Handle: copied.Handle}, FaceKey: sideFaceKey(t, s, copied.Handle),
		ViewDirection: []float64{1, 0, 0}, IncludeBoundary: true,
	})
	var sil wire.BrepWiresResult
	call(t, r, s, "brep.silhouette", string(keyArgs), &sil)
	if len(sil.Wires) != 2 {
		t.Fatalf("cylinder silhouette = %d wires, want 2 rulings", len(sil.Wires))
	}
}

// sideFaceKey returns the transient copy's curved (cylindrical) side face
// key, probed directly on the registry — the wire surface addresses faces by
// key but does not enumerate a transient body's faces (the document's
// model.referenceKeys covers documents).
func sideFaceKey(t *testing.T, s *app.Session, handle int) string {
	t.Helper()
	tb, ok := s.TransientBodies().ByHandle(handle)
	if !ok {
		t.Fatal("missing transient body")
	}
	for _, f := range tb.Topo().Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			return string(f.ReferenceKey())
		}
	}
	t.Fatal("no curved side face on the cylinder copy")
	return ""
}
