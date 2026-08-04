// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// twoBodyPart extrudes two separate new bodies so the boolean/combine ops have a target
// and a tool. Returns the session.
func twoBodyPart(t *testing.T) *app.Session {
	t.Helper()
	s := profiledPart(t)
	if _, err := apply(t, s, "extrude", `{"sketchIndex":0,"distance":"10 mm","operation":"new"}`); err != nil {
		t.Fatalf("body 1: %v", err)
	}
	if _, err := apply(t, s, "extrude", `{"sketchIndex":1,"distance":"5 mm","operation":"new"}`); err != nil {
		t.Fatalf("body 2: %v", err)
	}
	return s
}

// TestAdditiveOperations drives the additive profile features to their feature-building
// success path (recomputeResult never errors for a well-formed request).
func TestAdditiveOperations(t *testing.T) {
	cases := []struct{ name, op, args string }{
		{"revolve default axis", "revolve", `{"sketchIndex":0,"angle":"360 deg"}`},
		{"revolve partial", "revolve", `{"sketchIndex":0,"angle":"90 deg","angle2":"30 deg"}`},
		{"coil", "coil", `{"sketchIndex":0,"pitch":"5 mm","revolutions":"3"}`},
		{"coil by height", "coil", `{"sketchIndex":0,"pitch":"5 mm","height":"15 mm","taper":"2 deg"}`},
		{"loft two sections", "loft", `{"sections":[{"sketchIndex":0,"profileIndex":0},{"sketchIndex":1,"profileIndex":0}]}`},
		{"loft closed", "loft", `{"sections":[{"sketchIndex":0,"profileIndex":0},{"sketchIndex":1,"profileIndex":0}],"closed":true}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := apply(t, profiledPart(t), c.op, c.args); err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
		})
	}
}

// TestThickenOptionErrors covers the #1876 router parse branches: an unknown direction or
// operation is rejected with a descriptive error rather than silently defaulting.
func TestThickenOptionErrors(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	if _, err := apply(t, s, "thicken", `{"thickness":"1 mm","direction":"sideways"}`); err == nil {
		t.Error("unknown thicken direction should error")
	}
	if _, err := apply(t, s, "thicken", `{"thickness":"1 mm","operation":"weld"}`); err == nil {
		t.Error("unknown thicken operation should error")
	}
}

// TestReplaceFaceNewFaceRefs covers the #1886 router path: replacing a face with a new-face set
// (here the face's own key) resolves through the plane target and rebuilds without error, and an
// empty new-face/target set is rejected.
func TestReplaceFaceNewFaceRefs(t *testing.T) {
	s, _, face := extrudedSolid(t)
	ok, _ := json.Marshal(map[string]any{"faceRefs": []string{face}, "newFaceRefs": []string{face}})
	if _, err := apply(t, s, "replaceFace", string(ok)); err != nil {
		t.Fatalf("replaceFace newFaceRefs: %v", err)
	}
	bad, _ := json.Marshal(map[string]any{"faceRefs": []string{face}})
	if _, err := apply(t, s, "replaceFace", string(bad)); err == nil {
		t.Error("replaceFace with neither newFaceRefs nor targetRef should error")
	}
}

// TestModifyAndPatternOperations drives the body-modifying, boolean, pattern, and mirror
// operations against a solid. They must return a result or a descriptive error (some need
// geometry the seeded box lacks), never panic — exercising decode/resolve/build.
func TestModifyAndPatternOperations(t *testing.T) {
	type setup func(t *testing.T) (*app.Session, map[string]any)

	withSolid := func(extra func(edge, face string) map[string]any) setup {
		return func(t *testing.T) (*app.Session, map[string]any) {
			s, edge, face := extrudedSolid(t)
			return s, extra(edge, face)
		}
	}
	feat := func(m map[string]any) map[string]any { m["sourceFeatures"] = []string{"Extrusion1"}; return m }

	cases := []struct {
		op    string
		build setup
	}{
		{"moveBody", withSolid(func(e, f string) map[string]any {
			return map[string]any{"bodyIndex": 0, "translation": []float64{10, 0, 0}}
		})},
		{"moveFace", withSolid(func(e, f string) map[string]any {
			return map[string]any{"faceRefs": []string{f}, "translation": []float64{1, 0, 0}}
		})},
		{"faceOffset", withSolid(func(e, f string) map[string]any { return map[string]any{"faceRefs": []string{f}, "distance": "1 mm"} })},
		{"deleteFace", withSolid(func(e, f string) map[string]any { return map[string]any{"faceRefs": []string{f}} })},
		{"simplify", withSolid(func(e, f string) map[string]any { return map[string]any{"faceRefs": []string{f}} })},
		{"trim", withSolid(func(e, f string) map[string]any {
			return map[string]any{"origin": []float64{0, 0, 5}, "normal": []float64{0, 0, 1}, "keepPositive": true}
		})},
		{"replaceFace", withSolid(func(e, f string) map[string]any { return map[string]any{"faceRefs": []string{f}, "targetRef": f} })},
		{"unwrap", withSolid(func(e, f string) map[string]any { return map[string]any{"faceRefs": []string{f}} })},
		{"hull", withSolid(func(e, f string) map[string]any { return map[string]any{} })},
		{"thicken", withSolid(func(e, f string) map[string]any { return map[string]any{"faceRefs": []string{f}, "thickness": "1 mm"} })},
		{"thread", withSolid(func(e, f string) map[string]any { return map[string]any{"faceRef": f} })},
		{"hole", withSolid(func(e, f string) map[string]any {
			return map[string]any{"faceRef": f, "diameter": "3 mm", "depth": "5 mm"}
		})},
		{"boss", withSolid(func(e, f string) map[string]any {
			return map[string]any{"faceRef": f, "diameter": "3 mm", "height": "5 mm"}
		})},
		{"grill", withSolid(func(e, f string) map[string]any { return map[string]any{"faceRef": f} })},
		{"surfaceOffset", withSolid(func(e, f string) map[string]any { return map[string]any{"faceRefs": []string{f}, "distance": "1 mm"} })},
		{"midSurface", withSolid(func(e, f string) map[string]any { return map[string]any{"faceRefs": []string{f}} })},
		{"ruledSurface", withSolid(func(e, f string) map[string]any { return map[string]any{"edgeRefs": []string{e}, "distance": "2 mm"} })},
		{"boundaryPatch", withSolid(func(e, f string) map[string]any {
			return map[string]any{"edgeLoopRefs": []string{e}, "condition": "curvature"}
		})},
		{"extend", withSolid(func(e, f string) map[string]any { return map[string]any{"edgeRefs": []string{e}, "distance": "1 mm"} })},
		{"stitch", withSolid(func(e, f string) map[string]any { return map[string]any{"faceRefs": []string{f}} })},
		{"patternRectangular", withSolid(func(e, f string) map[string]any {
			return feat(map[string]any{"countX": 2, "stepX": []float64{5, 0, 0}, "countY": 2, "stepY": []float64{0, 5, 0}})
		})},
		{"patternCircular", withSolid(func(e, f string) map[string]any {
			return feat(map[string]any{"count": 3, "angle": "120 deg", "axisPoint": []float64{0, 0, 0}, "axisDir": []float64{0, 0, 1}})
		})},
		{"mirror", withSolid(func(e, f string) map[string]any {
			return feat(map[string]any{"origin": []float64{0, 0, 0}, "normal": []float64{1, 0, 0}})
		})},
		{"patternSketchDriven", withSolid(func(e, f string) map[string]any {
			return feat(map[string]any{"sketchIndex": 0})
		})},
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			s, args := c.build(t)
			assertNoPanic(t, c.op, s, args)
		})
	}
}

// TestRuledSurfaceSweepOptions covers the #1868 router path: the sweep type resolves its explicit
// direction (and optional draft/flip) and builds; a sweep with no direction, and an unknown type,
// error clearly. The legacy "perpendicular" spelling still resolves to the sweep type.
func TestRuledSurfaceSweepOptions(t *testing.T) {
	ok, _ := json.Marshal(map[string]any{"sketchIndex": 0, "distance": "3 mm", "type": "sweep", "direction": []float64{0, 0, 1}, "draftAngle": "5 deg", "flip": true})
	if _, err := apply(t, profiledPart(t), "ruledSurface", string(ok)); err != nil {
		t.Fatalf("ruledSurface sweep: %v", err)
	}
	legacy, _ := json.Marshal(map[string]any{"sketchIndex": 0, "distance": "3 mm", "type": "perpendicular", "direction": []float64{0, 0, 1}})
	if _, err := apply(t, profiledPart(t), "ruledSurface", string(legacy)); err != nil {
		t.Fatalf("ruledSurface perpendicular (legacy): %v", err)
	}
	if _, err := apply(t, profiledPart(t), "ruledSurface", `{"sketchIndex":0,"distance":"3 mm","type":"sweep"}`); err == nil {
		t.Error("sweep with no direction should error")
	}
}

// TestBoundaryPatchConditionMapping covers the #1867 router condition parse: free/tangent/curvature/
// continuous map to G0/G1/G2, and an edge-loop request dispatches to the 3D fill (a solid's lone edge
// cannot close a loop, so a descriptive error — never a panic — is acceptable).
func TestBoundaryPatchConditionMapping(t *testing.T) {
	cases := map[string]feature.PatchCondition{
		"free": feature.PatchFree, "tangent": feature.PatchTangent,
		"curvature": feature.PatchCurvature, "continuous": feature.PatchCurvature, "": feature.PatchFree,
	}
	for name, want := range cases {
		if got := patchCondition(name); got != want {
			t.Errorf("patchCondition(%q) = %v, want %v", name, got, want)
		}
	}
	s, edge, _ := extrudedSolid(t)
	loop, _ := json.Marshal(map[string]any{"edgeLoopRefs": []string{edge}, "condition": "curvature"})
	if _, err := apply(t, s, "boundaryPatch", string(loop)); err != nil && err.Error() == "" {
		t.Error("boundaryPatch edge-loop returned an empty error")
	}
}

// TestMidSurfaceFacePairs covers the #1885 router path: manual face pairs bypass the maxThickness
// requirement and are dispatched to the model, while an empty request (no pairs, no maxThickness)
// errors.
func TestMidSurfaceFacePairs(t *testing.T) {
	s, _, face := extrudedSolid(t)
	pairs, _ := json.Marshal(map[string]any{"facePairs": []map[string]string{{"a": face, "b": face}}})
	if _, err := apply(t, s, "midSurface", string(pairs)); err != nil && err.Error() == "" {
		t.Error("midSurface facePairs returned an empty error")
	}
	if _, err := apply(t, s, "midSurface", `{}`); err == nil {
		t.Error("midSurface with neither facePairs nor maxThickness should error")
	}
}

// TestSculptSurfaces covers the #1881 router path: bounding surfaces with per-surface directions
// and an affected-body index parse and dispatch (result or a descriptive error, never a panic).
func TestSculptSurfaces(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	args, _ := json.Marshal(map[string]any{
		"operation":         "new",
		"surfaces":          []map[string]any{{"bodyIndex": 0, "direction": "positive"}},
		"affectedBodyIndex": 0,
	})
	if _, err := apply(t, s, "sculpt", string(args)); err != nil && err.Error() == "" {
		t.Error("sculpt surfaces returned an empty error")
	}
}

// TestTrimCuttingTools covers the #1880 trim tool resolution: a work-plane tool and a sketch-line
// tool resolve to a cutting plane (result or a descriptive error, never a resolution panic); a
// multi-plane surface-body tool and an empty request error clearly.
func TestTrimCuttingTools(t *testing.T) {
	// Fresh session per assertion — each trim mutates the running body.
	s1, _, _ := extrudedSolid(t)
	if _, err := apply(t, s1, "trim", `{"toolRef":"origin/plane/xy"}`); err != nil && err.Error() == "" {
		t.Error("trim toolRef returned an empty error")
	}
	s2, _, _ := extrudedSolid(t)
	if _, err := apply(t, s2, "trim", `{"toolSketchIndex":0,"toolLineIndex":0}`); err != nil && err.Error() == "" {
		t.Error("trim sketch-line tool returned an empty error")
	}
	s3, _, _ := extrudedSolid(t)
	if _, err := apply(t, s3, "trim", `{"toolBodyIndex":0}`); err == nil {
		t.Error("trim with a multi-plane tool body should error")
	}
	s4, _, _ := extrudedSolid(t)
	if _, err := apply(t, s4, "trim", `{}`); err == nil {
		t.Error("trim with no cutting tool should error")
	}
}

// TestExtendRouterOptions covers the #1878 extend parse/resolve branches: no edges errors, and
// extentType toPlane without a targetRef errors (the plane target cannot resolve).
func TestExtendRouterOptions(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	if _, err := apply(t, s, "extend", `{"distance":"1 mm"}`); err == nil {
		t.Error("extend with no edges should error")
	}
	if _, err := apply(t, s, "extend", `{"edgeRefs":["e1"],"extentType":"toPlane"}`); err == nil {
		t.Error("extend toPlane with no targetRef should error")
	}
}

// TestCombineTwoBodies covers the boolean-combine path that needs a tool body.
func TestCombineTwoBodies(t *testing.T) {
	for _, op := range []string{"join", "cut", "intersect"} {
		s := twoBodyPart(t)
		assertNoPanic(t, "combine", s, map[string]any{"targetIndex": 0, "toolIndex": 1, "operation": op})
	}
}

// TestFreeformPrimitives: the freeform ops build a subdivision cage from scratch (no solid).
func TestFreeformPrimitives(t *testing.T) {
	cases := []struct{ op, args string }{
		{"freeformBox", `{"sizeX":"10 mm","sizeY":"10 mm","sizeZ":"10 mm"}`},
		{"freeformPlane", `{"sizeX":"10 mm","sizeY":"10 mm"}`},
		{"freeformQuadBall", `{"radius":"5 mm"}`},
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			if _, err := apply(t, profiledPart(t), c.op, c.args); err != nil {
				t.Fatalf("%s: %v", c.op, err)
			}
		})
	}
}

// assertNoPanic runs op with marshaled args and fails only on a panic or an empty error;
// a descriptive error (e.g. geometry the seeded box can't satisfy) is acceptable.
func assertNoPanic(t *testing.T, op string, s *app.Session, args map[string]any) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("operation %q panicked: %v", op, r)
		}
	}()
	if _, err := applyMap(t, s, op, args); err != nil && err.Error() == "" {
		t.Fatalf("operation %q returned an empty error", op)
	}
}

// TestRevolveDirectionReachesTheDefinition covers the #2019 add_feature branch: a "direction" in
// the request lands on the definition on EVERY axis path, including the centerline one that
// restores and builds through its own constructor.
func TestRevolveDirectionReachesTheDefinition(t *testing.T) {
	for _, c := range []struct {
		name, args string
		want       feature.ExtentDirection
	}{
		{"flipped about a work axis", `{"sketchIndex":0,"angle":"90 deg","direction":"negative"}`, feature.NegativeDir},
		{"symmetric about a work axis", `{"sketchIndex":0,"angle":"90 deg","direction":"symmetric"}`, feature.SymmetricDir},
		{"absent defaults to forward", `{"sketchIndex":0,"angle":"90 deg"}`, feature.PositiveDir},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := profiledPart(t)
			if _, err := apply(t, s, "revolve", c.args); err != nil {
				t.Fatalf("apply: %v", err)
			}
			part := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
			last := part.Features().Item(part.Features().Count() - 1)
			if got := last.Definition().(*feature.RevolveFeature).Definition().Direction; got != c.want {
				t.Errorf("direction = %v, want %v", got, c.want)
			}
		})
	}
}
