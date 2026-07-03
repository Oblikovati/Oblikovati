// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestSkcovOffsetProfileRegion offsets the seeded 4×3 rectangle's whole closed profile outward
// (sketch.offset with profileIndex → offsetProfileRegion): a positive distance grows the region
// with rounded corners, so it yields a new closed loop of many line segments.
func TestSkcovOffsetProfileRegion(t *testing.T) {
	r, s := seededSession(t)
	pi := 0
	args := wire.OffsetSketchArgs{SketchIndex: 0, ProfileIndex: &pi, Distance: "1 cm", ArcSegments: 4}
	var res wire.OffsetSketchResult
	call(t, r, s, "sketch.offset", mustJSON(t, args), &res)
	if res.Kind != "line" || res.EntityID != res.Created[0] {
		t.Fatalf("offset result = %+v, want kind line with EntityID = first created", res)
	}
	if len(res.Created) < 8 { // 4 corners rounded into arc spans ⇒ well over the 4 straight edges
		t.Errorf("created %d entities, want the rounded offset loop (≥8 segments)", len(res.Created))
	}
}

// TestSkcovOffsetProfileRegionRejectsBadIndex: an out-of-range profile index is rejected with
// the offending value (offsetProfileRegion bounds check).
func TestSkcovOffsetProfileRegionRejectsBadIndex(t *testing.T) {
	r, s := seededSession(t)
	pi := 9
	args := wire.OffsetSketchArgs{SketchIndex: 0, ProfileIndex: &pi, Distance: "1 cm"}
	if _, err := r.Handle(s, "sketch.offset", []byte(mustJSON(t, args))); err == nil {
		t.Error("an out-of-range profile index must be rejected")
	}
}

// TestSkcovTransformMirror reflects the seeded rectangle's top edge across its bottom edge
// (sketch.transform op=mirror → mirrorOp): one mirrored copy is created.
func TestSkcovTransformMirror(t *testing.T) {
	r, s := seededSession(t)
	ids := skcovLineIDs(t, r, s)
	args := wire.TransformSketchArgs{SketchIndex: 0, Op: "mirror", Entities: []uint64{ids[2]}, MirrorLine: ids[0]}
	var res wire.TransformSketchResult
	call(t, r, s, "sketch.transform", mustJSON(t, args), &res)
	if len(res.Created) != 1 {
		t.Errorf("mirror created %d entities, want the single mirrored copy", len(res.Created))
	}
}

// TestSkcovTransformRotate rotates one seeded line 90° about the origin (sketch.transform
// op=rotate → rotateOp): the in-place edit maps every endpoint (x,y) → (−y,x).
func TestSkcovTransformRotate(t *testing.T) {
	r, s := seededSession(t)
	id := skcovLineIDs(t, r, s)[0]
	before := skcovEntityPoints(t, r, s, id)
	args := wire.TransformSketchArgs{SketchIndex: 0, Op: "rotate", Entities: []uint64{id}, Center: []float64{0, 0}, Angle: "90 deg"}
	call(t, r, s, "sketch.transform", mustJSON(t, args), &wire.TransformSketchResult{})
	after := skcovEntityPoints(t, r, s, id)
	for k, p := range before {
		if stdmath.Abs(after[k][0]+p[1]) > 1e-6 || stdmath.Abs(after[k][1]-p[0]) > 1e-6 {
			t.Errorf("point %d rotated to %v, want (%v,%v)", k, after[k], -p[1], p[0])
		}
	}
}

// TestSkcovTransformExtendReachesCrossing extends a line past its picked (B) end to the nearest
// crossing (sketch.transform op=extend → curveEditOp + pickNearerEnd): the pick nearest B extends
// the horizontal line at y=−2 out to the vertical support at x=3.
func TestSkcovTransformExtendReachesCrossing(t *testing.T) {
	r, s := seededSession(t)
	var ln wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,-2],[1,-2]]}`, &ln)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[3,-3],[3,-1]]}`, &wire.AddSketchEntityResult{})
	args := wire.TransformSketchArgs{SketchIndex: 0, Op: "extend", Entities: []uint64{ln.EntityID}, Vector: []float64{0.9, -2}}
	call(t, r, s, "sketch.transform", mustJSON(t, args), &wire.TransformSketchResult{})
	pts := skcovEntityPoints(t, r, s, ln.EntityID)
	if stdmath.Abs(pts[1][0]-3) > 1e-6 || stdmath.Abs(pts[1][1]+2) > 1e-6 {
		t.Errorf("extended B end = %v, want (3,-2)", pts[1])
	}
}

// TestSkcovSketch3DRegionProperties builds a planar closed square in a 3D sketch and reads its
// section properties (sketch3d.regionProperties → sketch3DRegionProperties): a 4×3 rectangle.
func TestSkcovSketch3DRegionProperties(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	for _, seg := range skcovSquare3DSegments() {
		call(t, r, s, "sketch3d.addEntity", seg, &wire.AddSketch3DEntityResult{})
	}
	var got wire.RegionPropertiesResult
	call(t, r, s, "sketch3d.regionProperties", `{"sketchIndex":0,"profileIndex":0}`, &got)
	if stdmath.Abs(got.Area-12) > 1e-6 || stdmath.Abs(got.Perimeter-14) > 1e-6 {
		t.Errorf("area/perimeter = %v/%v, want 12/14", got.Area, got.Perimeter)
	}
}

// TestSkcovSketch3DRegionPropertiesRejectsBadIndex: an out-of-range profile index errors.
func TestSkcovSketch3DRegionPropertiesRejectsBadIndex(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	if _, err := r.Handle(s, "sketch3d.regionProperties", []byte(`{"sketchIndex":0,"profileIndex":0}`)); err == nil {
		t.Error("an empty 3D sketch has no profile 0, want a rejection")
	}
}

// skcovLineIDs returns the seeded sketch's line entity ids in enumeration (add) order.
func skcovLineIDs(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	out := make([]uint64, 0, 4)
	for _, e := range ents.Entities {
		if e.Kind == "line" {
			out = append(out, e.ID)
		}
	}
	if len(out) < 4 {
		t.Fatalf("seeded sketch has %d lines, want the 4 rectangle edges", len(out))
	}
	return out
}

// skcovEntityPoints returns the endpoints of sketch-0 entity id.
func skcovEntityPoints(t *testing.T, r *Router, s *app.Session, id uint64) [][]float64 {
	t.Helper()
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	for _, e := range ents.Entities {
		if e.ID == id {
			return e.Points
		}
	}
	t.Fatalf("entity %d not found in sketch 0", id)
	return nil
}

// skcovSquare3DSegments is the four addEntity requests for a planar 4×3 rectangle in a 3D sketch.
func skcovSquare3DSegments() []string {
	return []string{
		`{"sketchIndex":0,"kind":"line","points":[[0,0,0],[4,0,0]]}`,
		`{"sketchIndex":0,"kind":"line","points":[[4,0,0],[4,3,0]]}`,
		`{"sketchIndex":0,"kind":"line","points":[[4,3,0],[0,3,0]]}`,
		`{"sketchIndex":0,"kind":"line","points":[[0,3,0],[0,0,0]]}`,
	}
}
