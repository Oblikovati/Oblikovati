// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import "testing"

// TestMoveBodyOperations drives the ordered move-operation builders (free-drag, along-ray,
// rotate-about-line) — the parametric move path moveBody exposes.
func TestMoveBodyOperations(t *testing.T) {
	ops := []map[string]any{
		{"type": "freeDrag", "x": "1 mm", "y": "2 mm", "z": "3 mm"},
		{"type": "alongRay", "dir": []float64{1, 0, 0}, "dist": "5 mm"},
		{"type": "rotateAboutLine", "point": []float64{0, 0, 0}, "dir": []float64{0, 0, 1}, "angle": "30 deg"},
	}
	for _, op := range ops {
		t.Run(op["type"].(string), func(t *testing.T) {
			s, _, _ := extrudedSolid(t)
			assertNoPanic(t, "moveBody", s, map[string]any{"bodyIndex": 0, "operations": []map[string]any{op}})
		})
	}
	// An unknown operation type is a precise error.
	s, _, _ := extrudedSolid(t)
	if _, err := applyMap(t, s, "moveBody", map[string]any{"bodyIndex": 0, "operations": []map[string]any{{"type": "teleport"}}}); err == nil {
		t.Error("moveBody with an unknown operation type should error")
	}
}

// TestDirectEditOperations drives each direct-edit mode (move/size/rotate/scale) on a face.
func TestDirectEditOperations(t *testing.T) {
	cases := []struct {
		mode string
		args func(face string) map[string]any
	}{
		{"move", func(f string) map[string]any {
			return map[string]any{"operation": "move", "faceRefs": []string{f}, "translation": []float64{1, 0, 0}}
		}},
		{"size", func(f string) map[string]any {
			return map[string]any{"operation": "size", "faceRefs": []string{f}, "direction": []float64{1, 0, 0}, "distance": "1 mm"}
		}},
		{"rotate", func(f string) map[string]any {
			return map[string]any{"operation": "rotate", "faceRefs": []string{f}, "axisPoint": []float64{0, 0, 0}, "axisDir": []float64{0, 0, 1}, "angle": "30 deg"}
		}},
		{"scale", func(f string) map[string]any {
			return map[string]any{"operation": "scale", "faceRefs": []string{f}, "scale": 1.5, "base": []float64{0, 0, 0}}
		}},
	}
	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			s, _, face := extrudedSolid(t)
			assertNoPanic(t, "directEdit", s, c.args(face))
		})
	}
	// Unknown operation → error.
	s, _, face := extrudedSolid(t)
	if _, err := applyMap(t, s, "directEdit", map[string]any{"operation": "warp", "faceRefs": []string{face}}); err == nil {
		t.Error("directEdit with an unknown operation should error")
	}
}

// TestHoleTypes drives the simple/counterbore/countersink hole geometries (the two- and
// three-length resolvers).
func TestHoleTypes(t *testing.T) {
	cases := []struct {
		name string
		args func(face string) map[string]any
	}{
		{"simple", func(f string) map[string]any {
			return map[string]any{"faceRef": f, "diameter": "3 mm", "depth": "5 mm"}
		}},
		{"counterbore", func(f string) map[string]any {
			return map[string]any{"faceRef": f, "type": "counterbore", "diameter": "3 mm", "depth": "8 mm", "counterDiameter": "6 mm", "counterDepth": "2 mm"}
		}},
		{"countersink", func(f string) map[string]any {
			return map[string]any{"faceRef": f, "type": "countersink", "diameter": "3 mm", "depth": "8 mm", "sinkDiameter": "6 mm", "includedAngle": "82 deg"}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _, face := extrudedSolid(t)
			assertNoPanic(t, "hole", s, c.args(face))
		})
	}
}

// TestSweepVariants exercises the optional sweep fields (profile scaling and the
// path-and-section twist stations) — their parsing runs before any geometry build.
func TestSweepVariants(t *testing.T) {
	base := func() map[string]any {
		return map[string]any{"sketchIndex": 0, "profileIndex": 0, "pathSketchIndex": 1, "pathIndex": 0}
	}
	t.Run("scaling", func(t *testing.T) {
		s := profiledPart(t)
		a := base()
		a["scaling"] = "xy"
		assertNoPanic(t, "sweep", s, a)
	})
	t.Run("twist stations", func(t *testing.T) {
		s := profiledPart(t)
		a := base()
		a["definitionType"] = "pathAndSectionTwists"
		a["twistStations"] = []map[string]any{{"t": 0.0, "angle": "0 deg"}, {"t": 1.0, "angle": "90 deg"}}
		assertNoPanic(t, "sweep", s, a)
	})
	t.Run("ascending t enforced", func(t *testing.T) {
		s := profiledPart(t)
		a := base()
		a["definitionType"] = "pathAndSectionTwists"
		a["twistStations"] = []map[string]any{{"t": 1.0, "angle": "0 deg"}, {"t": 0.0, "angle": "90 deg"}}
		if _, err := applyMap(t, s, "sweep", a); err == nil {
			t.Error("non-ascending twist stations should error")
		}
	})
}

// TestMoveFaceRotate drives the rotational move-face path (vs the translate path already
// covered).
func TestMoveFaceRotate(t *testing.T) {
	s, _, face := extrudedSolid(t)
	assertNoPanic(t, "moveFace", s, map[string]any{
		"faceRefs": []string{face}, "axisPoint": []float64{0, 0, 0}, "axisDir": []float64{0, 0, 1}, "angle": "15 deg",
	})
}

// TestPartingAndSplit drives the parting (core/cavity) and split-solid resolvers.
func TestPartingAndSplit(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	assertNoPanic(t, "coreCavity", s, map[string]any{"axis": "z", "position": "5 mm", "shrinkage": "0.02"})
	assertNoPanic(t, "splitSolid", s, map[string]any{"workPlaneIndex": 0, "keep": "both"})
	assertNoPanic(t, "grill", s, map[string]any{"sketchIndex": 0, "boundaries": []int{0}})
}

// TestLoftGuides drives the loft end-conditions and area-graph guide paths.
func TestLoftGuides(t *testing.T) {
	sections := []map[string]any{{"sketchIndex": 0, "profileIndex": 0}, {"sketchIndex": 1, "profileIndex": 0}}
	t.Run("end conditions", func(t *testing.T) {
		assertNoPanic(t, "loft", profiledPart(t), map[string]any{
			"sections": sections,
			"first":    map[string]any{"condition": "tangent", "angle": "30 deg", "impact": 1.0},
			"last":     map[string]any{"condition": "direction", "angle": "45 deg", "reversed": true},
		})
	})
	t.Run("area graph", func(t *testing.T) {
		assertNoPanic(t, "loft", profiledPart(t), map[string]any{
			"sections":  sections,
			"areaGraph": []map[string]any{{"t": 0.25, "scale": 1.2}, {"t": 0.75, "scale": 0.8}},
		})
	})
}

// TestEmbossOnSolid drives the emboss feature against a solid face.
func TestEmbossOnSolid(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	assertNoPanic(t, "emboss", s, map[string]any{"sketchIndex": 1, "profileIndex": 0, "depth": "1 mm", "engrave": true})
}

// TestModelTolerance drives a GD&T feature-control frame.
func TestModelTolerance(t *testing.T) {
	s, _, face := extrudedSolid(t)
	assertNoPanic(t, "modelTolerance", s, map[string]any{
		"frames": []map[string]any{{"geometry": face, "characteristic": "flatness", "value": "0.1 mm"}},
	})
	// An unknown characteristic is a precise error.
	if _, err := applyMap(t, s, "modelTolerance", map[string]any{
		"frames": []map[string]any{{"geometry": face, "characteristic": "wobbliness", "value": "0.1 mm"}},
	}); err == nil {
		t.Error("modelTolerance with an unknown characteristic should error")
	}
}
