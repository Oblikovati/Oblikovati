// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestSketchAddCenterlineMarksAxisOfRevolution is the regression for the centerline flag
// (PartDesigner #54): a line added with centerline:true becomes the sketch's single centerline,
// so a revolve with no explicit axis resolves it. A plain construction line does NOT.
func TestSketchAddCenterlineMarksAxisOfRevolution(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Item(0)
	if n := len(sk.Centerlines()); n != 0 {
		t.Fatalf("seeded sketch already has %d centerlines, want 0", n)
	}
	if err := tryCall(t, r, s, "sketch.addEntity",
		`{"sketchIndex":0,"kind":"line","points":[[0,0],[0,5]],"centerline":true}`); err != nil {
		t.Fatalf("add centerline: %v", err)
	}
	if n := len(sk.Centerlines()); n != 1 {
		t.Fatalf("Centerlines() = %d, want 1 after centerline:true", n)
	}
	if err := tryCall(t, r, s, "sketch.addEntity",
		`{"sketchIndex":0,"kind":"line","points":[[1,0],[1,5]],"construction":true}`); err != nil {
		t.Fatalf("add construction line: %v", err)
	}
	if n := len(sk.Centerlines()); n != 1 {
		t.Fatalf("Centerlines() = %d after a plain construction line, want still 1", n)
	}
}

// TestRevolveAboutCenterlineOverWire is the e2e regression for the whole PartDesigner #54 axis
// path: author an offset profile + a centerline through sketch.addEntity, then features.add a
// revolve with aboutCenterline:true. It must spin about the sketch centerline (not the default Y
// work axis) and yield the 24π-cm³ washer — proving the centerline flag and the revolve flag
// resolve together, as a procedural add-in revolving a tilted roller about its own axis relies on.
func TestRevolveAboutCenterlineOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity",
		`{"sketchIndex":1,"kind":"polyline","points":[[2,0],[4,0],[4,2],[2,2]],"closed":true}`,
		&wire.AddSketchEntityResult{})
	call(t, r, s, "sketch.addEntity",
		`{"sketchIndex":1,"kind":"line","points":[[0,0],[0,2]],"centerline":true}`,
		&wire.AddSketchEntityResult{})
	var res struct {
		Name string `json:"name"`
	}
	call(t, r, s, "features.add",
		`{"kind":"revolve","args":{"sketchIndex":1,"profileIndex":0,"aboutCenterline":true,"angle":"360 deg","operation":"new"}}`,
		&res)

	var mp wire.MassPropertiesResult
	call(t, r, s, "body.physicalProperties", `{"bodyIndex":0}`, &mp)
	const wantMm3 = 24 * 3.14159265 * 1000 // 24π cm³ washer → mm³ (outer R4, inner R2, height 2 cm)
	if mp.VolumeMm3 < wantMm3*0.97 || mp.VolumeMm3 > wantMm3*1.03 {
		t.Fatalf("centerline-revolved volume = %g mm³, want ≈%g (24π cm³); the revolve did not use the centerline", mp.VolumeMm3, wantMm3)
	}
}
