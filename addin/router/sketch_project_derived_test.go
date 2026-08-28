// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/geom"
)

// curvedFaceRef extrudes a cylinder and returns a reference key for its curved side face.
func curvedFaceRef(t *testing.T, r *Router, s *app.Session) string {
	t.Helper()
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"20 mm"}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct{}{})
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range part.SurfaceBodies().All()[0].Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			return string(f.ReferenceKey())
		}
	}
	t.Fatal("extruded cylinder has no curved face")
	return ""
}

// TestProjectCutEdgesOverWire extrudes a block, then projects the section curves where an XZ sketch
// cuts it — one associative projected curve per section loop (#1873).
func TestProjectCutEdgesOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	// A profile centred on the origin, so the canonical XZ plane (y=0) cuts the block's interior.
	for _, seg := range []string{
		`[[-2,-1.5],[2,-1.5]]`, `[[2,-1.5],[2,1.5]]`, `[[2,1.5],[-2,1.5]]`, `[[-2,1.5],[-2,-1.5]]`,
	} {
		call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":`+seg+`}`, &struct{}{})
	}
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct{}{})

	call(t, r, s, "sketch.create", `{"plane":"XZ"}`, &wire.CreateSketchResult{}) // sketch 1 cuts the block
	var proj wire.ProjectGeometryResult
	call(t, r, s, "sketch.projectCutEdges", `{"sketchIndex":1}`, &proj)
	if len(proj.Created) == 0 || !proj.Healthy {
		t.Fatalf("projectCutEdges = %+v, want at least one created / healthy", proj)
	}
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":1}`, &ents)
	if got := countReferenceCurves(ents.Entities); got != len(proj.Created) {
		t.Errorf("enumerated %d reference curves, want %d", got, len(proj.Created))
	}
}

// TestProjectCutEdgesNoSolidErrors: with no solid to cut, the op errors rather than creating a
// dead, geometry-less reference.
func TestProjectCutEdgesNoSolidErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if err := tryCall(t, r, s, "sketch.projectCutEdges", `{"sketchIndex":0}`); err == nil {
		t.Error("projectCutEdges with no solid body should error")
	}
}

// TestProjectSilhouetteOverWire projects a cylinder side face's silhouette onto a YZ sketch,
// selecting the +Y ruling by proximity (#1873).
func TestProjectSilhouetteOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	faceRef := curvedFaceRef(t, r, s)
	call(t, r, s, "sketch.create", `{"plane":"YZ"}`, &wire.CreateSketchResult{}) // sketch 1, normal +X

	var proj wire.ProjectGeometryResult
	args, err := json.Marshal(wire.ProjectSilhouetteArgs{
		SketchIndex: 1, FaceRef: faceRef, ProximityPoint: []float64{0, 2, 2.5}, IncludeBoundary: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	call(t, r, s, "sketch.projectSilhouette", string(args), &proj)
	if len(proj.Created) != 1 || !proj.Healthy {
		t.Fatalf("projectSilhouette = %+v, want 1 created / healthy", proj)
	}
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":1}`, &ents)
	if got := countReferenceCurves(ents.Entities); got != 1 {
		t.Errorf("enumerated %d reference curves, want 1 silhouette", got)
	}
}

// TestProjectSilhouetteUnknownFaceErrors: an unresolved face reference is a clean error.
func TestProjectSilhouetteUnknownFaceErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	args, _ := json.Marshal(wire.ProjectSilhouetteArgs{
		SketchIndex: 0, FaceRef: "no-such-face", ProximityPoint: []float64{0, 0, 0},
	})
	if err := tryCall(t, r, s, "sketch.projectSilhouette", string(args)); err == nil {
		t.Error("projectSilhouette with an unknown face should error")
	}
}
