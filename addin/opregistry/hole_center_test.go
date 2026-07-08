// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	stdmath "math"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// activePartBody returns the active part's first running solid body.
func activePartBody(t *testing.T, s *app.Session) *topo.Body {
	t.Helper()
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() == 0 {
		t.Fatal("part has no body")
	}
	return def.SurfaceBodies().Item(0)
}

// faceVertexMean averages a face's vertices — the point the hole op drills at by default (it
// mirrors the model's centroidOf(faceVertexPoints)).
func faceVertexMean(f *topo.Face) math.Point3 {
	vs := f.Vertices()
	var sx, sy, sz float64
	for _, v := range vs {
		p := v.Point()
		sx, sy, sz = sx+float64(p.X), sy+float64(p.Y), sz+float64(p.Z)
	}
	n := float64(len(vs))
	return math.P3(sx/n, sy/n, sz/n)
}

// boxTopFace returns the extruded box's uppermost (max-Z) face reference key and its vertex
// centroid.
func boxTopFace(t *testing.T, s *app.Session) (string, math.Point3) {
	t.Helper()
	b := activePartBody(t, s)
	var top *topo.Face
	for _, f := range b.Faces() {
		if top == nil || faceVertexMean(f).Z > faceVertexMean(top).Z {
			top = f
		}
	}
	if top == nil {
		t.Fatal("box has no faces")
	}
	return string(top.ReferenceKey()), faceVertexMean(top)
}

// bodyVolume returns the active part body's volume.
func bodyVolume(t *testing.T, s *app.Session) float64 {
	t.Helper()
	return ops.BodyGeometryProperties(activePartBody(t, s), ops.DefaultQuality()).Volume
}

// bodyCentroidX returns the active part body's center-of-mass X coordinate.
func bodyCentroidX(t *testing.T, s *app.Session) float64 {
	t.Helper()
	return float64(ops.BodyGeometryProperties(activePartBody(t, s), ops.DefaultQuality()).Centroid.X)
}

// drilledBox seeds a fresh 4×3×1 cm box and drills a blind hole on its top face; center is the
// optional explicit drill point (nil ⇒ the face centroid). It returns the session after the drill.
func drilledBox(t *testing.T, center []float64) *app.Session {
	t.Helper()
	s, _, _ := extrudedSolid(t)
	faceKey, _ := boxTopFace(t, s)
	args := map[string]any{"faceRef": faceKey, "diameter": "3 mm", "depth": "5 mm"}
	if center != nil {
		args["center"] = center
	}
	if _, err := applyMap(t, s, "hole", args); err != nil {
		t.Fatalf("drill hole (center=%v): %v", center, err)
	}
	return s
}

// TestHoleCenterAtCentroidMatchesDefault: an explicit center equal to the face centroid drills
// the same hole as the centroid default — same volume, same center of mass (no regression when
// Center is supplied redundantly).
func TestHoleCenterAtCentroidMatchesDefault(t *testing.T) {
	def := drilledBox(t, nil)
	seed := profiledPart(t) // recompute the centroid on an identical fresh box
	if _, err := apply(t, seed, "extrude", `{"sketchIndex":0,"distance":"10 mm"}`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, c := boxTopFace(t, seed)
	centered := drilledBox(t, []float64{float64(c.X), float64(c.Y), float64(c.Z)})

	if vd, vc := bodyVolume(t, def), bodyVolume(t, centered); stdmath.Abs(vd-vc) > 1e-6 {
		t.Errorf("centered-hole volume = %v, default-hole volume = %v, want equal", vc, vd)
	}
	if xd, xc := bodyCentroidX(t, def), bodyCentroidX(t, centered); stdmath.Abs(xd-xc) > 1e-6 {
		t.Errorf("centered-hole centroid.X = %v, default = %v, want equal", xc, xd)
	}
}

// TestHoleOffsetCenterShiftsBore: drilling at an offset center removes the same cylinder volume
// but at a different location, so the remaining body's center of mass shifts along the offset.
func TestHoleOffsetCenterShiftsBore(t *testing.T) {
	// Centroid of the 4×3 face is (2,1.5); offset +0.8 cm in X keeps the Ø3 mm bore well inside.
	centered := drilledBox(t, []float64{2, 1.5, 1})
	offset := drilledBox(t, []float64{2.8, 1.5, 1})

	if vc, vo := bodyVolume(t, centered), bodyVolume(t, offset); stdmath.Abs(vc-vo) > 1e-6 {
		t.Errorf("offset-hole volume = %v, centered = %v, want equal (same bore removed)", vo, vc)
	}
	xc, xo := bodyCentroidX(t, centered), bodyCentroidX(t, offset)
	if stdmath.Abs(xc-xo) < 1e-4 {
		t.Errorf("offset-hole centroid.X = %v, centered = %v, want a detectable shift", xo, xc)
	}
}

// TestHoleCenterExprResolves: the centerExpr form resolves per-coordinate expressions to the
// same drill point as the literal center.
func TestHoleCenterExprResolves(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	faceKey, _ := boxTopFace(t, s)
	args := map[string]any{"faceRef": faceKey, "diameter": "3 mm", "depth": "5 mm",
		"centerExpr": []string{"2.8 cm", "1.5 cm", "1 cm"}}
	if _, err := applyMap(t, s, "hole", args); err != nil {
		t.Fatalf("drill hole with centerExpr: %v", err)
	}
	literal := drilledBox(t, []float64{2.8, 1.5, 1})
	if xe, xl := bodyCentroidX(t, s), bodyCentroidX(t, literal); stdmath.Abs(xe-xl) > 1e-6 {
		t.Errorf("centerExpr centroid.X = %v, literal center = %v, want equal", xe, xl)
	}
}

// TestHoleCenterWrongCount: a literal center that is not a 3-vector is a clean error.
func TestHoleCenterWrongCount(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	faceKey, _ := boxTopFace(t, s)
	args := map[string]any{"faceRef": faceKey, "diameter": "3 mm", "depth": "5 mm", "center": []float64{2, 1}}
	if _, err := applyMap(t, s, "hole", args); err == nil {
		t.Error("center with 2 coordinates should error")
	}
}

// TestHoleCenterExprBadExpression: an unparseable centerExpr coordinate is a clean error.
func TestHoleCenterExprBadExpression(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	faceKey, _ := boxTopFace(t, s)
	args := map[string]any{"faceRef": faceKey, "diameter": "3 mm", "depth": "5 mm",
		"centerExpr": []string{"2 cm", "nonsense", "1 cm"}}
	if _, err := applyMap(t, s, "hole", args); err == nil {
		t.Error("an unparseable centerExpr coordinate should error")
	}
}

// TestHoleCenterExprWrongCount: centerExpr with the wrong number of coordinates is a clean error.
func TestHoleCenterExprWrongCount(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	faceKey, _ := boxTopFace(t, s)
	args := map[string]any{"faceRef": faceKey, "diameter": "3 mm", "depth": "5 mm",
		"centerExpr": []string{"2 cm", "1 cm"}}
	if _, err := applyMap(t, s, "hole", args); err == nil {
		t.Error("centerExpr with 2 coordinates should error")
	}
}
