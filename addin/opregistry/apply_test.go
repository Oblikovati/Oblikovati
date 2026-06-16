// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"strings"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// profiledPart builds a session whose active part has a "width" parameter and two
// sketches, each holding a closed rectangle profile (sketch 0 on XY, sketch 1 on a
// plane 10 cm up) — enough geometry for the additive operations to consume.
func profiledPart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	d, err := s.Workspace().Add(doc.Part, "t.obk", true)
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	def := compdef.NewPartComponentDefinition()
	d.SetContent(def)
	if _, err := def.Parameters().AddUserParameter("width", "4 cm"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}
	addRect(def.Sketches().Add(sketch.XYPlane()), 4, 3)
	addRect(def.Sketches().Add(sketch.XYPlane()), 4, 3)
	def.Recompute()
	return s
}

// addRect draws a closed w×h rectangle at the sketch origin (one profile), chaining the
// four corners into a loop.
func addRect(sk *sketch.Sketch, w, h float64) {
	at := func(x, y float64) *sketch.Point { return sk.Points().Add(math.P2(x, y)) }
	corners := []*sketch.Point{at(0, 0), at(w, 0), at(w, h), at(0, h)}
	for i, c := range corners {
		sk.Lines().Add(c, corners[(i+1)%len(corners)])
	}
}

// apply runs the named default-registry operation with JSON args.
func apply(t *testing.T, s *app.Session, name, args string) (json.RawMessage, error) {
	t.Helper()
	d, ok := Default().ByName(name)
	if !ok {
		t.Fatalf("no descriptor named %q", name)
	}
	return d.Apply(s, json.RawMessage(args))
}

// TestExtrudeApply drives every extent/direction/taper branch of the reference
// operation, plus its validation errors.
func TestExtrudeApply(t *testing.T) {
	ok := []struct{ name, args string }{
		{"distance new", `{"sketchIndex":0,"profileIndex":0,"distance":"10 mm","operation":"new"}`},
		{"symmetric", `{"sketchIndex":0,"distance":"8 mm","direction":"symmetric"}`},
		{"negative + taper", `{"sketchIndex":0,"distance":"6 mm","direction":"negative","taper":"3 deg"}`},
		{"two-direction", `{"sketchIndex":0,"distance":"6 mm","secondDistance":"4 mm"}`},
		{"through-all", `{"sketchIndex":0,"extent":"through-all"}`},
		{"expression distance", `{"sketchIndex":0,"distance":"width/2"}`},
	}
	for _, c := range ok {
		t.Run(c.name, func(t *testing.T) {
			if _, err := apply(t, profiledPart(t), "extrude", c.args); err != nil {
				t.Fatalf("extrude %s: unexpected error: %v", c.name, err)
			}
		})
	}

	bad := []struct{ name, args, want string }{
		{"bad json", `{`, "invalid args"},
		{"sketch out of range", `{"sketchIndex":9,"distance":"1 mm"}`, "out of range"},
		{"unknown operation", `{"sketchIndex":0,"distance":"1 mm","operation":"melt"}`, "unknown operation"},
		{"unknown extent", `{"sketchIndex":0,"extent":"to-infinity"}`, "unknown extent"},
		{"zero distance", `{"sketchIndex":0,"distance":"0 mm"}`, "zero"},
		{"bad taper", `{"sketchIndex":0,"distance":"1 mm","taper":"banana"}`, "taper"},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			_, err := apply(t, profiledPart(t), "extrude", c.args)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("extrude %s: err = %v, want it to contain %q", c.name, err, c.want)
			}
		})
	}
}

// extrudedSolid seeds a part with one extruded box body and returns the session plus the
// reference-key strings of its first edge and first face — real keys for the dress-up ops.
func extrudedSolid(t *testing.T) (s *app.Session, edgeKey, faceKey string) {
	t.Helper()
	s = profiledPart(t)
	if _, err := apply(t, s, "extrude", `{"sketchIndex":0,"distance":"10 mm"}`); err != nil {
		t.Fatalf("seed extrude: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() == 0 {
		t.Fatal("seed extrude produced no body")
	}
	b := def.SurfaceBodies().Item(0)
	if len(b.Edges()) == 0 || len(b.Faces()) == 0 {
		t.Fatalf("extruded body has %d edges, %d faces", len(b.Edges()), len(b.Faces()))
	}
	return s, string(b.Edges()[0].ReferenceKey()), string(b.Faces()[0].ReferenceKey())
}

// applyMap marshals args (so a binary reference key is JSON-escaped the way the router
// does) and runs the operation.
func applyMap(t *testing.T, s *app.Session, op string, args map[string]any) (json.RawMessage, error) {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal %s args: %v", op, err)
	}
	return apply(t, s, op, string(raw))
}

// TestDressUpOnSolid drives the dress-up operations with REAL edge/face reference keys,
// covering the feature-building success path of each (recomputeResult reports geometry
// health in the result, so a valid request never errors here).
func TestDressUpOnSolid(t *testing.T) {
	edgeCases := []struct {
		name, op string
		args     map[string]any
	}{
		{"fillet flat", "fillet", map[string]any{"radius": "1 mm"}},
		{"fillet set const", "fillet", map[string]any{"_set": "radius"}},
		{"fillet set variable", "fillet", map[string]any{"_set": "var"}},
		{"chamfer distance", "chamfer", map[string]any{"distance": "1 mm"}},
		{"chamfer two distances", "chamfer", map[string]any{"distance": "1 mm", "distance2": "2 mm", "chamferType": "twoDistances"}},
		{"chamfer distance angle", "chamfer", map[string]any{"distance": "1 mm", "angle": "30 deg", "chamferType": "distanceAndAngle"}},
		{"lip", "lip", map[string]any{"width": "1 mm", "height": "1 mm"}},
	}
	for _, c := range edgeCases {
		t.Run(c.name, func(t *testing.T) {
			s, edge, _ := extrudedSolid(t)
			args := c.args
			switch args["_set"] {
			case "radius":
				args = map[string]any{"edgeSets": []map[string]any{{"edgeRefs": []string{edge}, "radius": "1 mm"}}}
			case "var":
				args = map[string]any{"edgeSets": []map[string]any{{"edgeRefs": []string{edge}, "startRadius": "1 mm", "endRadius": "2 mm"}}}
			default:
				args["edgeRefs"] = []string{edge}
			}
			if _, err := applyMap(t, s, c.op, args); err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
		})
	}
	faceCases := []struct {
		name, op string
		args     map[string]any
	}{
		{"shell", "shell", map[string]any{"thickness": "1 mm"}},
		{"draft", "draft", map[string]any{"angle": "3 deg"}},
	}
	for _, c := range faceCases {
		t.Run(c.name, func(t *testing.T) {
			s, _, face := extrudedSolid(t)
			c.args["faceRefs"] = []string{face}
			if _, err := applyMap(t, s, c.op, c.args); err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
		})
	}
}

// TestDressUpRejectsEmptyRefs: each dress-up op rejects a missing reference list.
func TestDressUpRejectsEmptyRefs(t *testing.T) {
	for _, op := range []string{"fillet", "chamfer", "shell", "draft", "lip"} {
		s, _, _ := extrudedSolid(t)
		if _, err := apply(t, s, op, `{"radius":"1 mm","distance":"1 mm","thickness":"1 mm","angle":"3 deg","width":"1 mm","height":"1 mm"}`); err == nil {
			t.Errorf("%s with no refs should error", op)
		}
	}
}

// TestExtrudeNeedsActivePart: the operation surfaces the no-part error.
func TestExtrudeNeedsActivePart(t *testing.T) {
	if _, err := apply(t, app.NewSession(), "extrude", `{"sketchIndex":0,"distance":"1 mm"}`); err == nil {
		t.Fatal("extrude without an active part should error")
	}
}

// TestParseHelpers covers the pure string→enum mappers directly.
func TestParseHelpers(t *testing.T) {
	for _, c := range []struct {
		in   string
		want bool // expect ok (no error)
	}{{"", true}, {"distance", true}, {"through-all", true}, {"to-next", true}, {"nope", false}} {
		_, err := parseExtentType(c.in)
		if (err == nil) != c.want {
			t.Errorf("parseExtentType(%q) err=%v, want ok=%v", c.in, err, c.want)
		}
	}
	if parseExtentDirection("negative") == parseExtentDirection("symmetric") {
		t.Error("negative and symmetric directions must differ")
	}
	if parseExtentDirection("garbage") != parseExtentDirection("positive") {
		t.Error("unknown direction must default to positive")
	}
	for _, op := range []string{"new", "join", "cut", "intersect"} {
		if _, err := parseOperation(op); err != nil {
			t.Errorf("parseOperation(%q): %v", op, err)
		}
	}
	if _, err := parseOperation("blend"); err == nil {
		t.Error("parseOperation(blend) should error")
	}
}

// TestEveryOperationHandlesArgsCleanly drives each default descriptor's Apply against a
// seeded part: with representative args it must return a result OR a descriptive error,
// never panic. This exercises the argument-parsing/resolution path of every operation.
func TestEveryOperationHandlesArgsCleanly(t *testing.T) {
	// Representative args per operation. Many additive ops succeed on the seeded profile;
	// reference-bearing ops (dress-up, holes) return a clean "needs a reference" error,
	// which still exercises their decode/resolve code.
	args := map[string]string{
		"extrude":             `{"sketchIndex":0,"distance":"10 mm"}`,
		"revolve":             `{"sketchIndex":0,"angle":"360 deg"}`,
		"rib":                 `{"sketchIndex":0,"thickness":"2 mm","depth":"5 mm"}`,
		"emboss":              `{"sketchIndex":0,"depth":"1 mm"}`,
		"coil":                `{"sketchIndex":0,"pitch":"5 mm","revolutions":3}`,
		"loft":                `{"sections":[{"sketchIndex":0,"profileIndex":0},{"sketchIndex":1,"profileIndex":0}]}`,
		"sweep":               `{"sketchIndex":0,"path":{"sketchIndex":1}}`,
		"fillet":              `{"edges":["e1"],"radius":"1 mm"}`,
		"chamfer":             `{"edges":["e1"],"distance":"1 mm"}`,
		"shell":               `{"faces":["f1"],"thickness":"1 mm"}`,
		"draft":               `{"faces":["f1"],"angle":"3 deg","neutralFace":"f2"}`,
		"lip":                 `{"edges":["e1"],"width":"1 mm","height":"1 mm"}`,
		"hole":                `{"faceRef":"f1","diameter":"3 mm","depth":"5 mm"}`,
		"boss":                `{"faceRef":"f1","diameter":"3 mm","height":"5 mm"}`,
		"grill":               `{"faceRef":"f1"}`,
		"thread":              `{"faceRef":"f1"}`,
		"combine":             `{"operation":"join","toolBody":1}`,
		"thicken":             `{"faces":["f1"],"thickness":"1 mm"}`,
		"trim":                `{"sketchIndex":0}`,
		"directEdit":          `{"faces":["f1"]}`,
		"moveFace":            `{"faces":["f1"]}`,
		"faceOffset":          `{"faces":["f1"],"distance":"1 mm"}`,
		"deleteFace":          `{"faces":["f1"]}`,
		"split":               `{"faces":["f1"],"toolSketch":0}`,
		"replaceFace":         `{"faces":["f1"]}`,
		"simplify":            `{}`,
		"unwrap":              `{"faces":["f1"]}`,
		"modelTolerance":      `{"linear":"0.01 mm"}`,
		"moveBody":            `{"body":0,"type":"freeDrag","x":"1 mm"}`,
		"bendPart":            `{}`,
		"sheetMetalFace":      `{"sketchIndex":0}`,
		"sheetMetalFlange":    `{"edge":"x","height":"10 mm"}`,
		"sheetMetalHem":       `{"edge":"x","length":"6 mm"}`,
		"splitSolid":          `{"toolSketch":0}`,
		"coreCavity":          `{}`,
		"hull":                `{}`,
		"patternRectangular":  `{"feature":0,"count":2,"spacing":"5 mm"}`,
		"patternCircular":     `{"feature":0,"count":3,"angle":"120 deg"}`,
		"mirror":              `{"feature":0,"plane":"XY"}`,
		"patternSketchDriven": `{"feature":0,"sketchIndex":0}`,
		"boundaryPatch":       `{"edges":["e1"]}`,
		"ruledSurface":        `{"edges":["e1"],"distance":"1 mm"}`,
		"surfaceOffset":       `{"faces":["f1"],"distance":"1 mm"}`,
		"extend":              `{"edges":["e1"],"distance":"1 mm"}`,
		"midSurface":          `{"faces":["f1"]}`,
		"stitch":              `{"faces":["f1"]}`,
		"sculpt":              `{"faces":["f1"]}`,
		"freeformBox":         `{"length":"10 mm","width":"10 mm","height":"10 mm"}`,
		"freeformPlane":       `{"length":"10 mm","width":"10 mm"}`,
		"freeformQuadBall":    `{"radius":"5 mm"}`,
		"mesh":                `{}`,
	}
	for _, d := range Default().All() {
		a, ok := args[d.Name]
		if !ok {
			t.Errorf("no representative args for operation %q — add one", d.Name)
			continue
		}
		t.Run(d.Name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("operation %q panicked on %s: %v", d.Name, a, r)
				}
			}()
			if _, err := d.Apply(profiledPart(t), json.RawMessage(a)); err != nil {
				// A descriptive error is fine (e.g. an unresolved reference); a bare or
				// empty message is not.
				if strings.TrimSpace(err.Error()) == "" {
					t.Fatalf("operation %q returned an empty error", d.Name)
				}
			}
		})
	}
}
