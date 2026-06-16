// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// tryCall invokes a method and returns its error (for paths where a clean error on
// finicky geometry is acceptable but a panic/crash is not).
func tryCall(t *testing.T, r *Router, s *app.Session, method, args string) error {
	t.Helper()
	_, err := r.Handle(s, method, []byte(args))
	return err
}

// TestSketchAddEntityMoreKinds drives the sketch.addEntity kinds the existing
// TestSketchAddEntityKinds does not (the composite shapes, derived curves, and the
// construction flag), so every per-kind builder is exercised.
func TestSketchAddEntityMoreKinds(t *testing.T) {
	kinds := []struct{ name, args string }{
		{"circle center", `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[0,0]],"radius":"2 cm"}`},
		{"circle threePoint", `{"sketchIndex":0,"kind":"circle","variant":"threePoint","points":[[0,0],[2,0],[1,1]]}`},
		{"rectangle center", `{"sketchIndex":0,"kind":"rectangle","variant":"center","points":[[0,0],[3,2]]}`},
		{"rectangle threePoint", `{"sketchIndex":0,"kind":"rectangle","variant":"threePoint","points":[[0,0],[4,0],[0,2]]}`},
		{"polygon", `{"sketchIndex":0,"kind":"polygon","points":[[0,0],[2,0]],"sides":6}`},
		{"slot", `{"sketchIndex":0,"kind":"slot","points":[[0,0],[5,0]],"width":"2 cm"}`},
		{"polyline", `{"sketchIndex":0,"kind":"polyline","points":[[0,0],[2,0],[2,2]]}`},
		{"construction line", `{"sketchIndex":0,"kind":"line","points":[[0,0],[1,1]],"construction":true}`},
		{"equationCurve", `{"sketchIndex":0,"kind":"equationCurve","xExpr":"t","yExpr":"t*t","t0":0,"t1":1}`},
	}
	for _, k := range kinds {
		t.Run(k.name, func(t *testing.T) {
			r, s := seededSession(t)
			if err := tryCall(t, r, s, "sketch.addEntity", k.args); err != nil {
				t.Fatalf("sketch.addEntity %s: %v", k.name, err)
			}
		})
	}
}

// TestSketchFilletChamfer drives the two entity-reference composite kinds (corner fillet
// and chamfer between two connected lines).
func TestSketchFilletChamfer(t *testing.T) {
	for _, c := range []struct{ kind, extra string }{
		{"fillet", `"radius":"1 cm"`},
		{"chamfer", `"radius":"1 cm","distance2":"2 cm"`},
	} {
		t.Run(c.kind, func(t *testing.T) {
			r, s := seededSession(t)
			var a, b wire.AddSketchEntityResult
			call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[6,0]]}`, &a)
			call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[6,0],[6,6]]}`, &b)
			args := fmt.Sprintf(`{"sketchIndex":0,"kind":%q,"entityRefs":[%d,%d],%s}`, c.kind, a.EntityID, b.EntityID, c.extra)
			if err := tryCall(t, r, s, "sketch.addEntity", args); err != nil {
				t.Fatalf("%s: %v", c.kind, err)
			}
		})
	}
}

// TestSketchAddEntityErrors covers the validation/error branches.
func TestSketchAddEntityErrors(t *testing.T) {
	bad := []string{
		`{"sketchIndex":9,"kind":"line","points":[[0,0],[1,1]]}`,                // sketch out of range
		`{"sketchIndex":0,"kind":"unicorn","points":[[0,0]]}`,                   // unknown kind
		`{"sketchIndex":0,"kind":"line","points":[[0,0]]}`,                      // too few points
		`{"sketchIndex":0,"kind":"slot","points":[[0,0],[5,0]],"width":"wide"}`, // bad unit
	}
	for i, args := range bad {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			r, s := seededSession(t)
			if err := tryCall(t, r, s, "sketch.addEntity", args); err == nil {
				t.Errorf("expected an error for %s", args)
			}
		})
	}
}
